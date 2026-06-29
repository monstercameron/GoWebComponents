//go:build js && wasm

package interop

import (
	"context"
	"errors"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

type browserWorkerState struct {
	mu            sync.RWMutex
	options       WorkerOptions
	raw           js.Value
	active        bool
	nextRequestID int
}

type goWASMWorkerState struct {
	mu           sync.RWMutex
	options      GoWASMWorkerOptions
	worker       Worker
	bootstrapURL string
	active       bool
}

type browserMessagePortState struct {
	mu     sync.RWMutex
	raw    js.Value
	active bool
	target string
}

// GetSharedMemorySupport reports whether the current browser context can use
// the shared-memory worker path.
func GetSharedMemorySupport() (SharedMemorySupport, error) {
	parseSharedArrayBuffer := getGlobalValue("SharedArrayBuffer")
	parseAtomics := getGlobalValue("Atomics")
	parseIsCrossOriginIsolated := false
	parseCrossOriginIsolated := getGlobalValue("crossOriginIsolated")
	if !parseCrossOriginIsolated.IsUndefined() && !parseCrossOriginIsolated.IsNull() {
		parseIsCrossOriginIsolated = parseCrossOriginIsolated.Bool()
	}
	parseHasSharedArrayBuffer := parseSharedArrayBuffer.Type() == js.TypeFunction
	parseHasAtomics := parseAtomics.Type() == js.TypeObject || parseAtomics.Type() == js.TypeFunction
	return SharedMemorySupport{
		IsCrossOriginIsolated: parseIsCrossOriginIsolated,
		HasSharedArrayBuffer:  parseHasSharedArrayBuffer,
		HasAtomics:            parseHasAtomics,
		CanUseSharedMemory:    parseIsCrossOriginIsolated && parseHasSharedArrayBuffer && parseHasAtomics,
	}, nil
}

// OpenSharedBuffer creates a SharedArrayBuffer for worker-visible shared
// memory access.
func OpenSharedBuffer(parseByteLength int) (SharedBuffer, error) {
	if parseByteLength < 0 {
		return SharedBuffer{}, wrapError("OpenSharedBuffer", "SharedArrayBuffer", CodeInvalid, errors.New("shared buffer byte length is negative"))
	}
	parseSupport, parseErr := GetSharedMemorySupport()
	if parseErr != nil {
		return SharedBuffer{}, parseErr
	}
	if !parseSupport.IsCrossOriginIsolated {
		return SharedBuffer{}, wrapError("OpenSharedBuffer", "SharedArrayBuffer", CodeUnavailable, errors.New("shared memory requires a cross-origin-isolated context"))
	}
	parseCtor, parseErr := globalProperty("OpenSharedBuffer", "SharedArrayBuffer")
	if parseErr != nil {
		return SharedBuffer{}, parseErr
	}
	return buildSharedBuffer("SharedArrayBuffer", parseCtor.New(parseByteLength)), nil
}

// OpenWorker starts a browser Worker at the given URL and returns a Go wrapper.
func OpenWorker(parseCtx context.Context, parseOptions WorkerOptions) (Worker, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if strings.TrimSpace(parseOptions.URL) == "" {
		return Worker{}, wrapError("OpenWorker", parseOptions.URL, CodeInvalid, errors.New("worker URL is empty"))
	}
	parseState := &browserWorkerState{options: parseOptions}
	if parseErr := parseState.start(parseCtx); parseErr != nil {
		return Worker{}, parseErr
	}
	return Worker{
		post:      parseState.post,
		postPorts: parseState.postPorts,
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			return parseState.subscribe(handler)
		},
		request:   parseState.request,
		terminate: parseState.terminate,
		restart:   parseState.restart,
	}, nil
}

// OpenGoWASMWorker starts a Go WASM worker using the given runtime and WASM URLs.
func OpenGoWASMWorker(parseCtx context.Context, parseOptions GoWASMWorkerOptions) (Worker, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if strings.TrimSpace(parseOptions.RuntimeURL) == "" {
		return Worker{}, wrapError("OpenGoWASMWorker", parseOptions.RuntimeURL, CodeInvalid, errors.New("runtime URL is empty"))
	}
	if strings.TrimSpace(parseOptions.WASMURL) == "" {
		return Worker{}, wrapError("OpenGoWASMWorker", parseOptions.WASMURL, CodeInvalid, errors.New("wasm URL is empty"))
	}
	parseState := &goWASMWorkerState{options: parseOptions}
	if parseErr := parseState.start(parseCtx); parseErr != nil {
		return Worker{}, parseErr
	}
	return Worker{
		post:      parseState.post,
		postPorts: parseState.postPorts,
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			return parseState.subscribe(handler)
		},
		request:   parseState.request,
		terminate: parseState.terminate,
		restart:   parseState.restart,
	}, nil
}

// OpenMessageChannel creates a linked pair of browser MessagePorts.
func OpenMessageChannel() (MessageChannel, error) {
	parseCtor, parseErr := globalProperty("OpenMessageChannel", "MessageChannel")
	if parseErr != nil {
		return MessageChannel{}, parseErr
	}
	parseRaw := parseCtor.New()
	parsePort1 := parseRaw.Get("port1")
	parsePort2 := parseRaw.Get("port2")
	if parsePort1.IsUndefined() || parsePort1.IsNull() || parsePort2.IsUndefined() || parsePort2.IsNull() {
		return MessageChannel{}, wrapError("OpenMessageChannel", "MessageChannel", CodeDecode, errors.New("message channel ports are unavailable"))
	}
	return MessageChannel{
		port1: newMessagePort("MessageChannel.port1", parsePort1),
		port2: newMessagePort("MessageChannel.port2", parsePort2),
	}, nil
}

// buildSharedBuffer wraps a SharedArrayBuffer JS value in the SharedBuffer
// helper surface.
func buildSharedBuffer(parseTarget string, parseRaw js.Value) SharedBuffer {
	return SharedBuffer{
		raw:           parseRaw,
		getByteLength: func() int { return parseRaw.Get("byteLength").Int() },
		readBytes: func(parseOffset int, parseDest []byte) (int, error) {
			if len(parseDest) == 0 {
				return 0, nil
			}
			parseView, parseCount, parseErr := getSharedBufferByteView("SharedBuffer.ReadBytes", parseTarget, parseRaw, parseOffset, len(parseDest))
			if parseErr != nil {
				return 0, parseErr
			}
			if parseCount == 0 {
				return 0, nil
			}
			js.CopyBytesToGo(parseDest[:parseCount], parseView)
			return parseCount, nil
		},
		writeBytes: func(parseOffset int, parseSource []byte) (int, error) {
			if len(parseSource) == 0 {
				return 0, nil
			}
			parseView, parseCount, parseErr := getSharedBufferByteView("SharedBuffer.WriteBytes", parseTarget, parseRaw, parseOffset, len(parseSource))
			if parseErr != nil {
				return 0, parseErr
			}
			if parseCount == 0 {
				return 0, nil
			}
			js.CopyBytesToJS(parseView, parseSource[:parseCount])
			return parseCount, nil
		},
		getInt32Length: func() int { return parseRaw.Get("byteLength").Int() / 4 },
		loadInt32: func(parseIndex int) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.LoadInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("load", parseView, parseIndex).Int()), nil
		},
		storeInt32: func(parseIndex int, parseValue int32) error {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.StoreInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return parseErr
			}
			parseAtomics.Call("store", parseView, parseIndex, parseValue)
			return nil
		},
		addInt32: func(parseIndex int, parseDelta int32) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.AddInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("add", parseView, parseIndex, parseDelta).Int()), nil
		},
		subInt32: func(parseIndex int, parseDelta int32) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.SubInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("sub", parseView, parseIndex, parseDelta).Int()), nil
		},
		andInt32: func(parseIndex int, parseMask int32) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.AndInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("and", parseView, parseIndex, parseMask).Int()), nil
		},
		orInt32: func(parseIndex int, parseMask int32) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.OrInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("or", parseView, parseIndex, parseMask).Int()), nil
		},
		xorInt32: func(parseIndex int, parseMask int32) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.XorInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("xor", parseView, parseIndex, parseMask).Int()), nil
		},
		exchangeInt32: func(parseIndex int, parseValue int32) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.ExchangeInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("exchange", parseView, parseIndex, parseValue).Int()), nil
		},
		compareExchangeInt32: func(parseIndex int, parseOldValue int32, parseNewValue int32) (int32, error) {
			parseView, parseAtomics, parseErr := getSharedBufferInt32Access("SharedBuffer.CompareExchangeInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			return int32(parseAtomics.Call("compareExchange", parseView, parseIndex, parseOldValue, parseNewValue).Int()), nil
		},
		waitInt32: func(parseIndex int, parseExpected int32, parseTimeout time.Duration) (string, error) {
			parseView, parseAtomics, parseErr := getSharedBufferWaitInt32Access("SharedBuffer.WaitInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return "", parseErr
			}
			if parseTimeout > 0 {
				return parseAtomics.Call("wait", parseView, parseIndex, parseExpected, durationMS(parseTimeout)).String(), nil
			}
			return parseAtomics.Call("wait", parseView, parseIndex, parseExpected).String(), nil
		},
		notifyInt32: func(parseIndex int, parseCount int) (int, error) {
			parseView, parseAtomics, parseErr := getSharedBufferNotifyInt32Access("SharedBuffer.NotifyInt32", parseTarget, parseRaw, parseIndex)
			if parseErr != nil {
				return 0, parseErr
			}
			if parseCount > 0 {
				return parseAtomics.Call("notify", parseView, parseIndex, parseCount).Int(), nil
			}
			return parseAtomics.Call("notify", parseView, parseIndex).Int(), nil
		},
	}
}

// getGlobalValue resolves a global or window-attached property without turning
// absence into an error.
func getGlobalValue(parseName string) js.Value {
	parseValue := js.Global().Get(parseName)
	if (parseValue.IsUndefined() || parseValue.IsNull()) && parseName != "window" {
		if parseWindow := js.Global().Get("window"); !parseWindow.IsUndefined() && !parseWindow.IsNull() {
			parseValue = parseWindow.Get(parseName)
		}
	}
	return parseValue
}

// getSharedBufferRaw resolves the JS SharedArrayBuffer value from a SharedBuffer
// wrapper.
func getSharedBufferRaw(parseBuffer SharedBuffer) (js.Value, bool) {
	parseRaw, parseOk := parseBuffer.raw.(js.Value)
	return parseRaw, parseOk
}

// getSharedBufferByteView resolves a Uint8Array view over the requested byte
// range.
func getSharedBufferByteView(parseOp string, parseTarget string, parseRaw js.Value, parseOffset int, parseLength int) (js.Value, int, error) {
	if parseOffset < 0 {
		return js.Undefined(), 0, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("byte offset is negative"))
	}
	if parseLength < 0 {
		return js.Undefined(), 0, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("byte length is negative"))
	}
	parseByteLength := parseRaw.Get("byteLength").Int()
	if parseOffset > parseByteLength {
		return js.Undefined(), 0, wrapError(parseOp, parseTarget, CodeInvalid, errors.New("byte offset exceeds shared buffer length"))
	}
	parseCount := parseLength
	if parseOffset+parseCount > parseByteLength {
		parseCount = parseByteLength - parseOffset
	}
	parseCtor, parseErr := globalProperty(parseOp, "Uint8Array")
	if parseErr != nil {
		return js.Undefined(), 0, parseErr
	}
	return parseCtor.New(parseRaw, parseOffset, parseCount), parseCount, nil
}

// getSharedBufferInt32View resolves an Int32Array view over the addressable
// int32 portion of the shared buffer.
func getSharedBufferInt32View(parseOp string, parseTarget string, parseRaw js.Value) (js.Value, error) {
	_ = parseTarget
	parseByteLength := parseRaw.Get("byteLength").Int()
	parseLength := parseByteLength / 4
	parseCtor, parseErr := globalProperty(parseOp, "Int32Array")
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	return parseCtor.New(parseRaw, 0, parseLength), nil
}

// getSharedAtomics resolves the Atomics object for shared-memory operations.
func getSharedAtomics(parseOp string, parseTarget string) (js.Value, error) {
	parseAtomics := getGlobalValue("Atomics")
	if parseAtomics.IsUndefined() || parseAtomics.IsNull() {
		return js.Undefined(), unavailable(parseOp, parseTarget)
	}
	return parseAtomics, nil
}

// getSharedBufferInt32Access validates an int32 index and returns the shared
// Int32Array view plus the Atomics object.
func getSharedBufferInt32Access(parseOp string, parseTarget string, parseRaw js.Value, parseIndex int) (js.Value, js.Value, error) {
	if parseIndex < 0 {
		return js.Undefined(), js.Undefined(), wrapError(parseOp, parseTarget, CodeInvalid, errors.New("int32 index is negative"))
	}
	parseView, parseErr := getSharedBufferInt32View(parseOp, parseTarget, parseRaw)
	if parseErr != nil {
		return js.Undefined(), js.Undefined(), parseErr
	}
	if parseIndex >= parseView.Length() {
		return js.Undefined(), js.Undefined(), wrapError(parseOp, parseTarget, CodeInvalid, errors.New("int32 index exceeds shared buffer length"))
	}
	parseAtomics, parseErr := getSharedAtomics(parseOp, parseTarget)
	if parseErr != nil {
		return js.Undefined(), js.Undefined(), parseErr
	}
	return parseView, parseAtomics, nil
}

// getSharedBufferWaitInt32Access validates worker-owned wait access for one
// int32 slot and ensures `Atomics.wait` is callable.
func getSharedBufferWaitInt32Access(parseOp string, parseTarget string, parseRaw js.Value, parseIndex int) (js.Value, js.Value, error) {
	if _, parseErr := currentWorkerGlobal(parseOp); parseErr != nil {
		return js.Undefined(), js.Undefined(), parseErr
	}
	parseView, parseAtomics, parseErr := getSharedBufferInt32Access(parseOp, parseTarget, parseRaw, parseIndex)
	if parseErr != nil {
		return js.Undefined(), js.Undefined(), parseErr
	}
	parseWait := parseAtomics.Get("wait")
	if parseWait.Type() != js.TypeFunction {
		return js.Undefined(), js.Undefined(), unavailable(parseOp, parseTarget)
	}
	return parseView, parseAtomics, nil
}

// getSharedBufferNotifyInt32Access validates notify access for one int32 slot
// and ensures `Atomics.notify` is callable.
func getSharedBufferNotifyInt32Access(parseOp string, parseTarget string, parseRaw js.Value, parseIndex int) (js.Value, js.Value, error) {
	parseView, parseAtomics, parseErr := getSharedBufferInt32Access(parseOp, parseTarget, parseRaw, parseIndex)
	if parseErr != nil {
		return js.Undefined(), js.Undefined(), parseErr
	}
	parseNotify := parseAtomics.Get("notify")
	if parseNotify.Type() != js.TypeFunction {
		return js.Undefined(), js.Undefined(), unavailable(parseOp, parseTarget)
	}
	return parseView, parseAtomics, nil
}

// GetWorkerScope returns a WorkerScope for posting and receiving messages within a worker.
func GetWorkerScope() (WorkerScope, error) {
	parseRaw, parseErr := currentWorkerGlobal("GetWorkerScope")
	if parseErr != nil {
		return WorkerScope{}, parseErr
	}
	return WorkerScope{
		post: func(parseMessage2 WorkerMessage) error {
			return postStructuredMessageJS("WorkerScope.Post", "worker", parseRaw, parseMessage2)
		},
		postPorts: func(parseMessage2 WorkerMessage, parsePorts ...MessagePort) error {
			return postStructuredMessageJS("WorkerScope.PostPorts", "worker", parseRaw, parseMessage2, parsePorts...)
		},
		subscribe: func(handler func(WorkerMessage, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("WorkerScope.Subscribe", "worker", CodeInvalid, errors.New("handler is nil"))
			}
			parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				defer RecoverContainedPanic("GetWorkerScope callback")
				parseMessage, parseMessageErr := workerMessageFromEvent("WorkerScope.Subscribe", "worker", parseArgs)
				handler(parseMessage, parseMessageErr)
				return nil
			})
			parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
				defer RecoverContainedPanic("GetWorkerScope callback")
				handler(WorkerMessage{}, wrapError("WorkerScope.Subscribe", "worker", CodeRemote, errors.New(workerRemoteErrorSummary(parseArgs2))))
				return nil
			})
			parseRaw.Call("addEventListener", "message", parseMessageFn)
			parseRaw.Call("addEventListener", "messageerror", parseErrorFn)
			return Subscription{cancel: func() {
				parseRaw.Call("removeEventListener", "message", parseMessageFn)
				parseRaw.Call("removeEventListener", "messageerror", parseErrorFn)
				parseMessageFn.Release()
				parseErrorFn.Release()
			}}, nil
		},
	}, nil
}
