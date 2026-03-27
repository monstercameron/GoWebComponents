//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"syscall/js"
	"time"
)

// GetLocalStorage returns the browser localStorage wrapper.
func GetLocalStorage() (Storage, error) {
	return resolveStorage("localStorage")
}

// GetSessionStorage returns the browser sessionStorage wrapper.
func GetSessionStorage() (Storage, error) {
	return resolveStorage("sessionStorage")
}

// GetWindowEnv returns a lightweight reader for shared values attached to window.
func GetWindowEnv() (WindowEnv, error) {
	parseRawWindow, parseErr := globalProperty("WindowEnv", "window")
	if parseErr != nil {
		return WindowEnv{}, parseErr
	}
	return WindowEnv{lookup: func(parseName string) (Value, bool) {
		parseKey := strings.TrimSpace(parseName)
		if parseKey == "" {
			return Value{}, false
		}
		parseValue := parseRawWindow.Get(parseKey)
		if parseValue.IsUndefined() || parseValue.IsNull() {
			return Value{}, false
		}
		return Value{raw: parseValue}, true
	}}, nil
}

func resolveStorage(parseName string) (Storage, error) {
	parseRaw, parseErr := globalProperty("Storage", parseName)
	if parseErr != nil {
		return Storage{}, parseErr
	}
	return Storage{
		getItem: func(parseKey string) (string, bool, error) {
			parseValue := parseRaw.Call("getItem", parseKey)
			if parseValue.IsUndefined() || parseValue.IsNull() {
				return "", false, nil
			}
			return parseValue.String(), true, nil
		},
		getMany: func(parseKeys []string) (map[string]string, error) {
			return storageGetMany(parseRaw, parseKeys), nil
		},
		setItem: func(parseKey2 string, parseValue3 string) error {
			parseRaw.Call("setItem", parseKey2, parseValue3)
			return nil
		},
		removeItem: func(parseKey3 string) (parseErr2 error) {
			defer recoverInteropException("Storage.RemoveItem", parseName+".removeItem", &parseErr2)
			parseRemoveItem := parseRaw.Get("removeItem")
			if parseRemoveItem.Type() != js.TypeFunction {
				return &Error{Op: "Storage.RemoveItem", Target: parseName + ".removeItem", Code: CodeNotFunction, Err: errors.New("storage removeItem is not callable")}
			}
			parseRaw.Call("removeItem", parseKey3)
			return nil
		},
		clear: func() error {
			parseRaw.Call("clear")
			return nil
		},
		length: func() (int, error) {
			return parseRaw.Get("length").Int(), nil
		},
		key: func(parseIndex int) (string, bool, error) {
			parseValue2 := parseRaw.Call("key", parseIndex)
			if parseValue2.IsUndefined() || parseValue2.IsNull() {
				return "", false, nil
			}
			return parseValue2.String(), true, nil
		},
	}, nil
}

// GetWindowLocation returns a Location backed by the browser window.location object.
func GetWindowLocation() (Location, error) {
	parseRaw, parseErr := globalPath("Location", "window", "location")
	if parseErr != nil {
		return Location{}, parseErr
	}
	return Location{
		href:     func() string { return parseRaw.Get("href").String() },
		pathname: func() string { return parseRaw.Get("pathname").String() },
		search:   func() string { return parseRaw.Get("search").String() },
		hash:     func() string { return parseRaw.Get("hash").String() },
		origin:   func() string { return parseRaw.Get("origin").String() },
		assign: func(parseRawURL string) error {
			parseRaw.Call("assign", parseRawURL)
			return nil
		},
		replace: func(parseRawURL2 string) error {
			parseRaw.Call("replace", parseRawURL2)
			return nil
		},
		reload: func() error {
			parseRaw.Call("reload")
			return nil
		},
	}, nil
}

// GetWindowHistory returns a History backed by the browser window.history object.
func GetWindowHistory() (History, error) {
	parseRaw, parseErr := globalPath("History", "window", "history")
	if parseErr != nil {
		return History{}, parseErr
	}
	return History{
		length: func() (int, error) {
			return parseRaw.Get("length").Int(), nil
		},
		state: func() (any, error) {
			return jsValueToGo("History.State", "history.state", parseRaw.Get("state"))
		},
		back: func() error {
			parseRaw.Call("back")
			return nil
		},
		forward: func() error {
			parseRaw.Call("forward")
			return nil
		},
		goDelta: func(parseDelta int) error {
			parseRaw.Call("go", parseDelta)
			return nil
		},
		pushState: func(parseState any, parseTitle string, parseRawURL string) error {
			parseValue, parseErr2 := goValueToJS("History.PushState", "state", parseState)
			if parseErr2 != nil {
				return parseErr2
			}
			parseRaw.Call("pushState", parseValue, parseTitle, parseRawURL)
			return nil
		},
		replaceState: func(parseState2 any, parseTitle2 string, parseRawURL2 string) error {
			parseValue2, parseErr3 := goValueToJS("History.ReplaceState", "state", parseState2)
			if parseErr3 != nil {
				return parseErr3
			}
			parseRaw.Call("replaceState", parseValue2, parseTitle2, parseRawURL2)
			return nil
		},
	}, nil
}

// GetClipboard returns a Clipboard backed by navigator.clipboard.
func GetClipboard() (Clipboard, error) {
	parseRaw, parseErr := globalPath("Clipboard", "navigator", "clipboard")
	if parseErr != nil {
		return Clipboard{}, parseErr
	}
	return Clipboard{
		writeText: func(parseCtx context.Context, parseText string) error {
			_, parseErr2 := awaitValue(parseCtx, "Clipboard.WriteText", "navigator.clipboard.writeText", parseRaw.Call("writeText", parseText))
			return parseErr2
		},
		readText: func(parseCtx2 context.Context) (string, error) {
			parseValue, parseErr3 := awaitValue(parseCtx2, "Clipboard.ReadText", "navigator.clipboard.readText", parseRaw.Call("readText"))
			if parseErr3 != nil {
				return "", parseErr3
			}
			if parseValue.IsUndefined() || parseValue.IsNull() {
				return "", nil
			}
			return parseValue.String(), nil
		},
	}, nil
}

// ScheduleTimeout schedules fn to run once after delay using window.setTimeout.
func ScheduleTimeout(parseDelay time.Duration, parseFn func()) (Timer, error) {
	if parseFn == nil {
		return Timer{}, wrapError("ScheduleTimeout", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		parseCallback js.Func
		parseOnce     sync.Once
		parseId       js.Value
	)
	parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOnce.Do(func() {
			parseCallback.Release()
		})
		parseFn()
		return nil
	})
	parseId = js.Global().Call("setTimeout", parseCallback, durationMS(parseDelay))
	return Timer{
		cancel: func() error {
			parseOnce.Do(func() {
				js.Global().Call("clearTimeout", parseId)
				parseCallback.Release()
			})
			return nil
		},
	}, nil
}

// ScheduleInterval schedules fn to run repeatedly at interval using window.setInterval.
func ScheduleInterval(parseInterval time.Duration, parseFn func()) (Timer, error) {
	if parseFn == nil {
		return Timer{}, wrapError("ScheduleInterval", "", CodeInvalid, errors.New("callback is nil"))
	}
	var (
		parseCallback js.Func
		parseOnce     sync.Once
		parseId       js.Value
	)
	parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseFn()
		return nil
	})
	parseId = js.Global().Call("setInterval", parseCallback, durationMS(parseInterval))
	return Timer{
		cancel: func() error {
			parseOnce.Do(func() {
				js.Global().Call("clearInterval", parseId)
				parseCallback.Release()
			})
			return nil
		},
	}, nil
}

// GetWindowEvents returns an EventTarget for the browser window object.
func GetWindowEvents() (EventTarget, error) {
	parseRaw, parseErr := globalProperty("EventTarget", "window")
	if parseErr != nil {
		return EventTarget{}, parseErr
	}
	return newEventTarget("window", parseRaw), nil
}

// GetDocumentEvents returns an EventTarget for the browser document object.
func GetDocumentEvents() (EventTarget, error) {
	parseRaw, parseErr := globalProperty("EventTarget", "document")
	if parseErr != nil {
		return EventTarget{}, parseErr
	}
	return newEventTarget("document", parseRaw), nil
}

// GetDocument returns a Document backed by the browser document object.
func GetDocument() (Document, error) {
	parseRaw, parseErr := globalProperty("Document", "document")
	if parseErr != nil {
		return Document{}, parseErr
	}
	return Document{
		elementByID: func(parseId2 string) (Element, bool, error) {
			parseValue := parseRaw.Call("getElementById", parseId2)
			if parseValue.IsUndefined() || parseValue.IsNull() {
				return Element{}, false, nil
			}
			return newElement("document.getElementById", parseValue), true, nil
		},
		elementsByID: func(parseIds []string) (map[string]Element, error) {
			parseResolved := make(map[string]Element, len(parseIds))
			for parseId, parseValue2 := range documentElementsByID(parseRaw, parseIds) {
				parseResolved[parseId] = newElement("document.getElementById", parseValue2)
			}
			return parseResolved, nil
		},
		querySelector: func(parseSelector string) (Element, bool, error) {
			parseValue3 := parseRaw.Call("querySelector", parseSelector)
			if parseValue3.IsUndefined() || parseValue3.IsNull() {
				return Element{}, false, nil
			}
			return newElement("document.querySelector", parseValue3), true, nil
		},
	}, nil
}

var (
	storageGetManyHelperOnce sync.Once
	storageGetManyHelper     js.Value
	documentByIDHelperOnce   sync.Once
	documentByIDHelper       js.Value
)

func storageGetMany(parseRaw js.Value, parseKeys []string) map[string]string {
	storageGetManyHelperOnce.Do(func() {
		storageGetManyHelper = js.Global().Get("Function").New("storage", "keys", `
			const result = {};
			for (const key of keys) {
				const value = storage.getItem(key);
				if (value !== null && value !== undefined) {
					result[key] = value;
				}
			}
			return result;
		`)
	})
	parseValues := storageGetManyHelper.Invoke(parseRaw, stringArrayValue(parseKeys))
	if parseValues.IsUndefined() || parseValues.IsNull() {
		return map[string]string{}
	}
	parseResult := make(map[string]string, len(parseKeys))
	parseObjectKeys := js.Global().Get("Object").Call("keys", parseValues)
	for parseIndex := 0; parseIndex < parseObjectKeys.Get("length").Int(); parseIndex++ {
		parseKey := parseObjectKeys.Index(parseIndex).String()
		parseResult[parseKey] = parseValues.Get(parseKey).String()
	}
	return parseResult
}

func documentElementsByID(parseRaw js.Value, parseIds []string) map[string]js.Value {
	documentByIDHelperOnce.Do(func() {
		documentByIDHelper = js.Global().Get("Function").New("documentRef", "ids", `
			const result = {};
			for (const id of ids) {
				const value = documentRef.getElementById(id);
				if (value !== null && value !== undefined) {
					result[id] = value;
				}
			}
			return result;
		`)
	})
	parseValues := documentByIDHelper.Invoke(parseRaw, stringArrayValue(parseIds))
	if parseValues.IsUndefined() || parseValues.IsNull() {
		return map[string]js.Value{}
	}
	parseResult := make(map[string]js.Value, len(parseIds))
	parseObjectKeys := js.Global().Get("Object").Call("keys", parseValues)
	for parseIndex := 0; parseIndex < parseObjectKeys.Get("length").Int(); parseIndex++ {
		parseKey := parseObjectKeys.Index(parseIndex).String()
		parseResult[parseKey] = parseValues.Get(parseKey)
	}
	return parseResult
}

func stringArrayValue(parseValues []string) js.Value {
	parseArray := js.Global().Get("Array").New()
	for _, parseValue := range parseValues {
		parseArray.Call("push", parseValue)
	}
	return parseArray
}

// GetMediaQuery returns a MediaQueryList for the given CSS media query string.
func GetMediaQuery(parseQuery string) (MediaQueryList, error) {
	parseWindow, parseErr := globalProperty("GetMediaQuery", "window")
	if parseErr != nil {
		return MediaQueryList{}, parseErr
	}
	parseRaw := parseWindow.Call("matchMedia", parseQuery)
	if parseRaw.IsUndefined() || parseRaw.IsNull() {
		return MediaQueryList{}, unavailable("GetMediaQuery", parseQuery)
	}
	return MediaQueryList{
		matches: func() bool { return parseRaw.Get("matches").Bool() },
		media:   func() string { return parseRaw.Get("media").String() },
		subscribe: func(handler func(MediaQueryEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("MatchMedia.Subscribe", parseQuery, CodeInvalid, errors.New("handler is nil"))
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				parseEvent := parseRaw
				if len(parseArgs) > 0 && !parseArgs[0].IsUndefined() && !parseArgs[0].IsNull() {
					parseEvent = parseArgs[0]
				}
				handler(MediaQueryEvent{
					Matches: parseEvent.Get("matches").Bool(),
					Media:   parseEvent.Get("media").String(),
				})
				return nil
			})
			if parseAdd := parseRaw.Get("addEventListener"); parseAdd.Type() == js.TypeFunction {
				parseRaw.Call("addEventListener", "change", parseListener)
				return Subscription{cancel: func() {
					parseRaw.Call("removeEventListener", "change", parseListener)
					parseListener.Release()
				}}, nil
			}
			parseRaw.Call("addListener", parseListener)
			return Subscription{cancel: func() {
				parseRaw.Call("removeListener", parseListener)
				parseListener.Release()
			}}, nil
		},
	}, nil
}

// ImportModule dynamically imports a JavaScript ES module by specifier.
func ImportModule(parseCtx context.Context, parseSpecifier string) (Module, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if strings.TrimSpace(parseSpecifier) == "" {
		return Module{}, wrapError("ImportModule", parseSpecifier, CodeInvalid, errors.New("specifier is empty"))
	}
	parseImportFn := js.Global().Get("__gwcImportModule")
	if parseImportFn.IsUndefined() || parseImportFn.IsNull() {
		parseConstructor := js.Global().Get("Function")
		if parseConstructor.IsUndefined() || parseConstructor.IsNull() {
			return Module{}, unavailable("ImportModule", parseSpecifier)
		}
		parseImportFn = parseConstructor.New("specifier", "return import(specifier);")
	}
	parseRawModule, parseErr := awaitValue(parseCtx, "ImportModule", parseSpecifier, parseImportFn.Invoke(parseSpecifier))
	if parseErr != nil {
		return Module{}, parseErr
	}
	parseState := &moduleState{specifier: parseSpecifier, value: parseRawModule}
	return Module{
		call: func(parseCtx2 context.Context, parseExport string, parseArgs ...any) (any, error) {
			parseRaw, parseErr2 := parseState.export(parseExport)
			if parseErr2 != nil {
				return nil, parseErr2
			}
			if parseRaw.Type() != js.TypeFunction {
				return nil, wrapError("Module.Call", parseExport, CodeNotFunction, errors.New("export is not a function"))
			}
			parseJsArgs := make([]interface{}, 0, len(parseArgs))
			for _, parseArg := range parseArgs {
				parseValue, parseErr3 := goValueToJS("Module.Call", parseExport, parseArg)
				if parseErr3 != nil {
					return nil, parseErr3
				}
				parseJsArgs = append(parseJsArgs, parseValue)
			}
			parseResult, parseErr2 := awaitValue(parseCtx2, "Module.Call", parseExport, parseRaw.Invoke(parseJsArgs...))
			if parseErr2 != nil {
				return nil, parseErr2
			}
			return jsValueToGo("Module.Call", parseExport, parseResult)
		},
		callDefault: func(parseCtx3 context.Context, parseArgs2 ...any) (any, error) {
			return parseState.callDefault(parseCtx3, parseArgs2...)
		},
		value: func(parseCtx4 context.Context, parseExport2 string) (any, error) {
			parseRaw2, parseErr4 := parseState.export(parseExport2)
			if parseErr4 != nil {
				return nil, parseErr4
			}
			parseResolved, parseErr4 := awaitValue(parseCtx4, "Module.Value", parseExport2, parseRaw2)
			if parseErr4 != nil {
				return nil, parseErr4
			}
			return jsValueToGo("Module.Value", parseExport2, parseResolved)
		},
		dispose: parseState.dispose,
	}, nil
}

// OpenCrossTabChannel opens a BroadcastChannel or storage-fallback channel with the given options.
func OpenCrossTabChannel(parseOptions CrossTabChannelOptions) (CrossTabChannel, error) {
	parseName := strings.TrimSpace(parseOptions.Name)
	if parseName == "" {
		return CrossTabChannel{}, wrapError("OpenCrossTabChannel", parseOptions.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	parseSource := fmt.Sprintf("%s-%d", parseName, time.Now().UnixNano())
	if parseCtor := js.Global().Get("BroadcastChannel"); parseCtor.Type() == js.TypeFunction {
		return newBroadcastCrossTabChannel(parseName, parseSource, parseCtor.New(parseName)), nil
	}
	return newStorageCrossTabChannel(parseName, parseSource, resolveCrossTabStorageKey(parseName, parseOptions.StorageKey))
}

// OpenSecondaryWindowChannel opens a postMessage channel to a window opened via window.open.
func OpenSecondaryWindowChannel(parseOptions WindowChannelOptions) (WindowChannel, error) {
	parseName := strings.TrimSpace(parseOptions.Name)
	if parseName == "" {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", parseOptions.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	parseRawWindow, parseErr := globalProperty("Window", "window")
	if parseErr != nil {
		return WindowChannel{}, parseErr
	}
	parseOpenFn := parseRawWindow.Get("open")
	if parseOpenFn.Type() != js.TypeFunction {
		return WindowChannel{}, unavailable("OpenSecondaryWindowChannel", parseName)
	}
	parseRawURL := strings.TrimSpace(parseOptions.URL)
	if parseRawURL == "" {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", parseName, CodeInvalid, errors.New("window URL is empty"))
	}
	parseRaw := parseOpenFn.Invoke(parseRawURL, parseName, strings.TrimSpace(parseOptions.Features))
	if parseRaw.IsUndefined() || parseRaw.IsNull() {
		return WindowChannel{}, wrapError("OpenSecondaryWindowChannel", parseName, CodeUnavailable, errors.New("window.open returned no handle"))
	}
	return newWindowChannel(parseName, resolveWindowTargetOrigin(strings.TrimSpace(parseOptions.TargetOrigin)), parseRaw, true), nil
}

// OpenWindowOpenerChannel opens a postMessage channel to the window.opener.
func OpenWindowOpenerChannel(parseOptions WindowChannelOptions) (WindowChannel, error) {
	parseName := strings.TrimSpace(parseOptions.Name)
	if parseName == "" {
		return WindowChannel{}, wrapError("OpenWindowOpenerChannel", parseOptions.Name, CodeInvalid, errors.New("channel name is empty"))
	}
	parseRawWindow, parseErr := globalProperty("Window", "window")
	if parseErr != nil {
		return WindowChannel{}, parseErr
	}
	parseOpener := parseRawWindow.Get("opener")
	if parseOpener.IsUndefined() || parseOpener.IsNull() {
		return WindowChannel{}, unavailable("OpenWindowOpenerChannel", parseName)
	}
	return newWindowChannel(parseName, resolveWindowTargetOrigin(strings.TrimSpace(parseOptions.TargetOrigin)), parseOpener, false), nil
}

const defaultWorkerReadyTimeout = 5 * time.Second

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
				parseMessage, parseMessageErr := workerMessageFromEvent("WorkerScope.Subscribe", "worker", parseArgs)
				handler(parseMessage, parseMessageErr)
				return nil
			})
			parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
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

func (parseS *goWASMWorkerState) start(parseCtx context.Context) error {
	parseWorkerOptions, parseBootstrapURL, parseErr := buildGoWASMWorkerOptions(parseS.options)
	if parseErr != nil {
		return parseErr
	}
	parseWorker, parseErr := OpenWorker(parseCtx, parseWorkerOptions)
	if parseErr != nil {
		revokeObjectURL(parseBootstrapURL)
		return parseErr
	}
	parseS.mu.Lock()
	parseS.worker = parseWorker
	parseS.bootstrapURL = parseBootstrapURL
	parseS.active = true
	parseS.mu.Unlock()
	return nil
}

func (parseS *goWASMWorkerState) current(parseOp string, parseTarget string) (Worker, error) {
	parseS.mu.RLock()
	defer parseS.mu.RUnlock()
	if !parseS.active {
		return Worker{}, wrapError(parseOp, parseTarget, CodeDisposed, errors.New("worker is not active"))
	}
	return parseS.worker, nil
}

func (parseS *goWASMWorkerState) post(parseMessage any) error {
	parseWorker, parseErr := parseS.current("Worker.Post", parseS.options.WASMURL)
	if parseErr != nil {
		return parseErr
	}
	return parseWorker.Post(parseMessage)
}

func (parseS *goWASMWorkerState) postPorts(parseMessage any, parsePorts ...MessagePort) error {
	parseWorker, parseErr := parseS.current("Worker.PostPorts", parseS.options.WASMURL)
	if parseErr != nil {
		return parseErr
	}
	return parseWorker.PostPorts(parseMessage, parsePorts...)
}

func (parseS *goWASMWorkerState) subscribe(parseHandler func(WorkerMessage, error)) (Subscription, error) {
	parseWorker, parseErr := parseS.current("Worker.Subscribe", parseS.options.WASMURL)
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	return parseWorker.Subscribe(parseHandler)
}

func (parseS *goWASMWorkerState) request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	parseWorker, parseErr := parseS.current("Worker.Request", parseName)
	if parseErr != nil {
		return WorkerMessage{}, parseErr
	}
	return parseWorker.Request(parseCtx, parseName, parsePayload, parseOnProgress)
}

func (parseS *goWASMWorkerState) terminate() error {
	parseS.mu.Lock()
	parseWorker := parseS.worker
	parseBootstrapURL := parseS.bootstrapURL
	parseActive := parseS.active
	parseS.worker = Worker{}
	parseS.bootstrapURL = ""
	parseS.active = false
	parseS.mu.Unlock()
	if !parseActive {
		return wrapError("Worker.Terminate", parseS.options.WASMURL, CodeDisposed, errors.New("worker is not active"))
	}
	parseErr := parseWorker.Terminate()
	revokeObjectURL(parseBootstrapURL)
	return parseErr
}

func (parseS *goWASMWorkerState) restart(parseCtx context.Context) error {
	if parseErr := parseS.terminate(); parseErr != nil && !IsCode(parseErr, CodeDisposed) {
		return parseErr
	}
	return parseS.start(parseCtx)
}

func (parseS *browserWorkerState) start(parseCtx context.Context) error {
	parseRaw, parseErr := createBrowserWorker(parseS.options)
	if parseErr != nil {
		return parseErr
	}
	if parseS.options.Ready {
		parseWaitCtx := parseCtx
		if _, parseOk := parseWaitCtx.Deadline(); !parseOk {
			parseTimeout := parseS.options.ReadyTimeout
			if parseTimeout <= 0 {
				parseTimeout = defaultWorkerReadyTimeout
			}
			var parseCancel context.CancelFunc
			parseWaitCtx, parseCancel = context.WithTimeout(parseWaitCtx, parseTimeout)
			defer parseCancel()
		}
		if parseErr2 := waitWorkerReady(parseWaitCtx, parseRaw, parseS.options.URL); parseErr2 != nil {
			parseRaw.Call("terminate")
			return parseErr2
		}
	}
	parseS.mu.Lock()
	parseS.raw = parseRaw
	parseS.active = true
	parseS.mu.Unlock()
	return nil
}

func (parseS *browserWorkerState) current(parseOp string, parseTarget string) (js.Value, error) {
	parseS.mu.RLock()
	defer parseS.mu.RUnlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return js.Undefined(), wrapError(parseOp, parseTarget, CodeDisposed, errors.New("worker is not active"))
	}
	return parseS.raw, nil
}

func (parseS *browserWorkerState) post(parseMessage any) error {
	parseRaw, parseErr := parseS.current("Worker.Post", parseS.options.URL)
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("Worker.Post", parseS.options.URL, parseRaw, parseMessage)
}

func (parseS *browserWorkerState) postPorts(parseMessage any, parsePorts ...MessagePort) error {
	parseRaw, parseErr := parseS.current("Worker.PostPorts", parseS.options.URL)
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("Worker.PostPorts", parseS.options.URL, parseRaw, parseMessage, parsePorts...)
}

func (parseS *browserWorkerState) subscribe(parseHandler func(WorkerMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("Worker.Subscribe", parseS.options.URL, CodeInvalid, errors.New("handler is nil"))
	}
	parseRaw, parseErr := parseS.current("Worker.Subscribe", parseS.options.URL)
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseMessage, parseErr2 := workerMessageFromEvent("Worker.Subscribe", parseS.options.URL, parseArgs)
		parseHandler(parseMessage, parseErr2)
		return nil
	})
	parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseHandler(WorkerMessage{}, wrapError("Worker.Subscribe", parseS.options.URL, CodeRemote, errors.New(workerRemoteErrorSummary(parseArgs2))))
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseHandler(WorkerMessage{}, wrapError("Worker.Subscribe", parseS.options.URL, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs3))))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "error", parseErrorFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	return Subscription{cancel: func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "error", parseErrorFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseErrorFn.Release()
		parseMessageErrorFn.Release()
	}}, nil
}

func (parseS *browserWorkerState) request(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
	if parseCtx == nil {
		parseCtx = context.Background()
	}
	if strings.TrimSpace(parseName) == "" {
		return WorkerMessage{}, wrapError("Worker.Request", parseName, CodeInvalid, errors.New("request name is empty"))
	}
	parseRaw, parseErr := parseS.current("Worker.Request", parseName)
	if parseErr != nil {
		return WorkerMessage{}, parseErr
	}

	parseS.mu.Lock()
	parseS.nextRequestID++
	parseRequestID := fmt.Sprintf("worker-%d", parseS.nextRequestID)
	parseS.mu.Unlock()

	parseResultCh := make(chan WorkerMessage, 1)
	parseErrCh := make(chan error, 1)
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseMessage, parseDecodeErr := workerMessageFromEvent("Worker.Request", parseName, parseArgs)
		if parseDecodeErr != nil {
			if parseOnProgress != nil {
				parseOnProgress(WorkerMessage{}, parseDecodeErr)
			}
			parseErrCh <- parseDecodeErr
			return nil
		}
		if strings.TrimSpace(parseMessage.ID) != parseRequestID {
			return nil
		}
		switch strings.TrimSpace(parseMessage.Phase) {
		case "progress":
			if parseOnProgress != nil {
				parseOnProgress(parseMessage, nil)
			}
		case "error":
			parseErrCh <- wrapError("Worker.Request", parseName, CodeRemote, errors.New(workerRemoteEnvelopeError(parseMessage)))
		case "result", "message", "":
			parseResultCh <- parseMessage
		}
		return nil
	})
	parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseErrCh <- wrapError("Worker.Request", parseName, CodeRemote, errors.New(workerRemoteErrorSummary(parseArgs2)))
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseErrCh <- wrapError("Worker.Request", parseName, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs3)))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "error", parseErrorFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	defer func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "error", parseErrorFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseErrorFn.Release()
		parseMessageErrorFn.Release()
	}()

	if parseErr2 := parseS.post(WorkerMessage{
		ID:      parseRequestID,
		Phase:   "request",
		Name:    parseName,
		Payload: parsePayload,
	}); parseErr2 != nil {
		return WorkerMessage{}, parseErr2
	}

	select {
	case parseMessage2 := <-parseResultCh:
		return parseMessage2, nil
	case parseErr3 := <-parseErrCh:
		return WorkerMessage{}, parseErr3
	case <-parseCtx.Done():
		return WorkerMessage{}, workerContextError("Worker.Request", parseName, parseCtx.Err())
	}
}

func (parseS *browserWorkerState) terminate() error {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return wrapError("Worker.Terminate", parseS.options.URL, CodeDisposed, errors.New("worker is not active"))
	}
	parseS.raw.Call("terminate")
	parseS.raw = js.Undefined()
	parseS.active = false
	return nil
}

func (parseS *browserWorkerState) restart(parseCtx context.Context) error {
	parseS.mu.Lock()
	parseRaw := parseS.raw
	parseActive := parseS.active
	parseS.raw = js.Undefined()
	parseS.active = false
	parseS.mu.Unlock()
	if parseActive && !parseRaw.IsUndefined() && !parseRaw.IsNull() {
		parseRaw.Call("terminate")
	}
	return parseS.start(parseCtx)
}

func createBrowserWorker(parseOptions WorkerOptions) (js.Value, error) {
	parseCtor, parseErr := globalProperty("Worker", "Worker")
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	parseWorkerType := strings.TrimSpace(parseOptions.Type)
	if parseWorkerType != "" && parseWorkerType != "classic" && parseWorkerType != "module" {
		return js.Undefined(), wrapError("NewWorker", parseOptions.URL, CodeInvalid, errors.New("worker type must be classic or module"))
	}
	if parseWorkerType == "" && strings.HasSuffix(strings.ToLower(strings.TrimSpace(parseOptions.URL)), ".mjs") {
		parseWorkerType = "module"
	}
	if strings.TrimSpace(parseOptions.Name) == "" && parseWorkerType == "" {
		return parseCtor.New(parseOptions.URL), nil
	}
	parseInit := js.Global().Get("Object").New()
	if strings.TrimSpace(parseOptions.Name) != "" {
		parseInit.Set("name", parseOptions.Name)
	}
	if parseWorkerType != "" {
		parseInit.Set("type", parseWorkerType)
	}
	return parseCtor.New(parseOptions.URL, parseInit), nil
}

func newMessagePort(parseTarget string, parseRaw js.Value) MessagePort {
	parseState := &browserMessagePortState{
		raw:    parseRaw,
		active: true,
		target: parseTarget,
	}
	startMessagePort(parseRaw)
	return MessagePort{
		raw:       parseRaw,
		post:      parseState.post,
		postPorts: parseState.postPorts,
		subscribe: parseState.subscribe,
		close:     parseState.close,
	}
}

func startMessagePort(parseRaw js.Value) {
	parseStart := parseRaw.Get("start")
	if parseStart.Type() == js.TypeFunction {
		parseRaw.Call("start")
	}
}

func (parseS *browserMessagePortState) current(parseOp string) (js.Value, error) {
	parseS.mu.RLock()
	defer parseS.mu.RUnlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return js.Undefined(), wrapError(parseOp, parseS.target, CodeDisposed, errors.New("message port is not active"))
	}
	return parseS.raw, nil
}

func (parseS *browserMessagePortState) post(parsePayload any) error {
	parseRaw, parseErr := parseS.current("MessagePort.Post")
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("MessagePort.Post", parseS.target, parseRaw, parsePayload)
}

func (parseS *browserMessagePortState) postPorts(parsePayload any, parsePorts ...MessagePort) error {
	parseRaw, parseErr := parseS.current("MessagePort.PostPorts")
	if parseErr != nil {
		return parseErr
	}
	return postStructuredMessageJS("MessagePort.PostPorts", parseS.target, parseRaw, parsePayload, parsePorts...)
}

func (parseS *browserMessagePortState) subscribe(parseHandler func(MessagePortMessage, error)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("MessagePort.Subscribe", parseS.target, CodeInvalid, errors.New("handler is nil"))
	}
	parseRaw, parseErr := parseS.current("MessagePort.Subscribe")
	if parseErr != nil {
		return Subscription{}, parseErr
	}
	startMessagePort(parseRaw)
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseMessage, parseMessageErr := messagePortMessageFromEvent("MessagePort.Subscribe", parseS.target, parseArgs)
		parseHandler(parseMessage, parseMessageErr)
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseHandler(MessagePortMessage{}, wrapError("MessagePort.Subscribe", parseS.target, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs2))))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	return Subscription{cancel: func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseMessageErrorFn.Release()
	}}, nil
}

func (parseS *browserMessagePortState) close() error {
	parseS.mu.Lock()
	defer parseS.mu.Unlock()
	if !parseS.active || parseS.raw.IsUndefined() || parseS.raw.IsNull() {
		return wrapError("MessagePort.Close", parseS.target, CodeDisposed, errors.New("message port is not active"))
	}
	parseS.raw.Call("close")
	parseS.raw = js.Undefined()
	parseS.active = false
	return nil
}

func waitWorkerReady(parseCtx context.Context, parseRaw js.Value, parseTarget string) error {
	parseReadyCh := make(chan struct{}, 1)
	parseErrCh := make(chan error, 1)
	parseMessageFn := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseMessage, parseErr := workerMessageFromEvent("NewWorker", parseTarget, parseArgs)
		if parseErr != nil {
			parseErrCh <- parseErr
			return nil
		}
		if strings.TrimSpace(parseMessage.Phase) == "ready" {
			parseReadyCh <- struct{}{}
			return nil
		}
		if strings.TrimSpace(parseMessage.Phase) == "error" {
			parseErrCh <- wrapError("NewWorker", parseTarget, CodeRemote, errors.New(workerRemoteEnvelopeError(parseMessage)))
		}
		return nil
	})
	parseErrorFn := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseErrCh <- wrapError("NewWorker", parseTarget, CodeRemote, errors.New(workerRemoteErrorSummary(parseArgs2)))
		return nil
	})
	parseMessageErrorFn := js.FuncOf(func(parseThis3 js.Value, parseArgs3 []js.Value) interface{} {
		parseErrCh <- wrapError("NewWorker", parseTarget, CodeDecode, errors.New(workerRemoteErrorSummary(parseArgs3)))
		return nil
	})
	parseRaw.Call("addEventListener", "message", parseMessageFn)
	parseRaw.Call("addEventListener", "error", parseErrorFn)
	parseRaw.Call("addEventListener", "messageerror", parseMessageErrorFn)
	defer func() {
		parseRaw.Call("removeEventListener", "message", parseMessageFn)
		parseRaw.Call("removeEventListener", "error", parseErrorFn)
		parseRaw.Call("removeEventListener", "messageerror", parseMessageErrorFn)
		parseMessageFn.Release()
		parseErrorFn.Release()
		parseMessageErrorFn.Release()
	}()
	select {
	case <-parseReadyCh:
		return nil
	case parseErr2 := <-parseErrCh:
		consoleError(fmt.Sprintf("[interop/NewWorker] worker %q reported an error during startup: %v", parseTarget, parseErr2))
		return parseErr2
	case <-parseCtx.Done():
		consoleError(fmt.Sprintf("[interop/NewWorker] worker %q startup timed out — if this runs on the main goroutine inside a synchronous effect, the JS event loop is starved and the worker ready message can never arrive; wrap the call in a goroutine", parseTarget))
		return workerContextError("NewWorker", parseTarget, parseCtx.Err())
	}
}

func currentWorkerGlobal(parseOp string) (js.Value, error) {
	parseGlobal := js.Global()
	if parseDoc := parseGlobal.Get("document"); !parseDoc.IsUndefined() && !parseDoc.IsNull() {
		return js.Undefined(), unavailable(parseOp, "worker")
	}
	parsePostMessage := parseGlobal.Get("postMessage")
	if parsePostMessage.IsUndefined() || parsePostMessage.IsNull() {
		return js.Undefined(), unavailable(parseOp, "worker")
	}
	return parseGlobal, nil
}

func buildGoWASMWorkerOptions(parseOptions GoWASMWorkerOptions) (WorkerOptions, string, error) {
	parseRuntimeURL, parseErr := resolveURL("NewGoWASMWorker", parseOptions.RuntimeURL)
	if parseErr != nil {
		return WorkerOptions{}, "", parseErr
	}
	parseWasmURL, parseErr := resolveURL("NewGoWASMWorker", parseOptions.WASMURL)
	if parseErr != nil {
		return WorkerOptions{}, "", parseErr
	}
	parseBootstrapURL, parseErr := createObjectURL(goWASMWorkerBootstrapSource(parseRuntimeURL, parseWasmURL))
	if parseErr != nil {
		return WorkerOptions{}, "", parseErr
	}
	return WorkerOptions{
		URL:          parseBootstrapURL,
		Name:         parseOptions.Name,
		Type:         "classic",
		Ready:        parseOptions.Ready,
		ReadyTimeout: parseOptions.ReadyTimeout,
	}, parseBootstrapURL, nil
}

func resolveURL(parseOp string, parseInput string) (string, error) {
	parseTrimmed := strings.TrimSpace(parseInput)
	if parseTrimmed == "" {
		return "", wrapError(parseOp, parseInput, CodeInvalid, errors.New("URL is empty"))
	}
	parseUrlCtor, parseErr := globalProperty(parseOp, "URL")
	if parseErr != nil {
		return "", parseErr
	}
	parseBase := js.Global().Get("document").Get("baseURI")
	if parseBase.IsUndefined() || parseBase.IsNull() || strings.TrimSpace(parseBase.String()) == "" {
		parseLocation, parseLocationErr := globalProperty(parseOp, "location")
		if parseLocationErr != nil {
			return "", parseLocationErr
		}
		parseBase = parseLocation.Get("href")
	}
	parseResolved := parseUrlCtor.New(parseTrimmed, parseBase).Get("href").String()
	if looksLikeJSTypeDescriptor(parseResolved) {
		consoleError(fmt.Sprintf("[interop/%s] resolveURL produced a JS type descriptor %q for input %q — this usually means js.Value.String() was called on a non-string JS value", parseOp, parseResolved, parseInput))
		return "", wrapError(parseOp, parseInput, CodeInvalid, fmt.Errorf("resolved URL is a JS type descriptor %q, not a valid URL — check that the input is a string value", parseResolved))
	}
	return parseResolved, nil
}

func createObjectURL(parseSource string) (string, error) {
	parseBlobCtor, parseErr := globalProperty("NewGoWASMWorker", "Blob")
	if parseErr != nil {
		return "", parseErr
	}
	parseUrlAPI, parseErr := globalProperty("NewGoWASMWorker", "URL")
	if parseErr != nil {
		return "", parseErr
	}
	parseParts := js.Global().Get("Array").New()
	parseParts.Call("push", parseSource)
	parseOptions := js.Global().Get("Object").New()
	parseOptions.Set("type", "text/javascript")
	parseBlob := parseBlobCtor.New(parseParts, parseOptions)
	parseResult := parseUrlAPI.Call("createObjectURL", parseBlob).String()
	if looksLikeJSTypeDescriptor(parseResult) {
		consoleError(fmt.Sprintf("[interop/NewGoWASMWorker] createObjectURL returned JS type descriptor %q instead of a blob: URL", parseResult))
		return "", wrapError("NewGoWASMWorker", "createObjectURL", CodeInvalid, fmt.Errorf("createObjectURL returned %q, not a valid blob: URL", parseResult))
	}
	return parseResult, nil
}

func revokeObjectURL(parseObjectURL string) {
	parseTrimmed := strings.TrimSpace(parseObjectURL)
	if parseTrimmed == "" {
		return
	}
	parseUrlAPI := js.Global().Get("URL")
	if parseUrlAPI.IsUndefined() || parseUrlAPI.IsNull() {
		return
	}
	parseUrlAPI.Call("revokeObjectURL", parseTrimmed)
}

func goWASMWorkerBootstrapSource(parseRuntimeURL string, parseWasmURL string) string {
	parseRuntimeJSON, _ := json.Marshal(parseRuntimeURL)
	parseWasmJSON, _ := json.Marshal(parseWasmURL)
	return `(function(){
const runtimeURL=` + string(parseRuntimeJSON) + `;
const wasmURL=` + string(parseWasmJSON) + `;
const postBootstrapError = (error) => {
  const message = error && error.message ? error.message : String(error);
  try {
    self.postMessage({ phase: "error", name: "bootstrap", error: message });
  } catch (_) {}
};
const looksInvalid = (url, label) => {
  if (!url || /^<\w+>$/.test(url)) {
    const msg = "[interop/worker-bootstrap] " + label + " is invalid: " + JSON.stringify(url) + " — this usually means a Go js.Value.String() was called on a non-string JS value";
    console.error(msg);
    postBootstrapError(new Error(msg));
    return true;
  }
  return false;
};
if (looksInvalid(runtimeURL, "runtimeURL") || looksInvalid(wasmURL, "wasmURL")) { return; }
const instantiate = async (go) => {
  if (WebAssembly.instantiateStreaming) {
    try {
      return await WebAssembly.instantiateStreaming(fetch(wasmURL), go.importObject);
    } catch (_) {}
  }
  const response = await fetch(wasmURL);
  if (!response.ok) {
    throw new Error("failed to fetch worker wasm: " + response.status + " " + response.statusText);
  }
  const bytes = await response.arrayBuffer();
  return await WebAssembly.instantiate(bytes, go.importObject);
};
(async () => {
  try {
    self.importScripts(runtimeURL);
    if (typeof Go !== "function") {
      throw new Error("Go runtime was not registered by wasm_exec.js");
    }
    const go = new Go();
    const result = await instantiate(go);
    await go.run(result.instance);
  } catch (error) {
    postBootstrapError(error);
  }
})();
})();`
}

func workerMessageFromEvent(parseOp string, parseTarget string, parseArgs []js.Value) (WorkerMessage, error) {
	if len(parseArgs) == 0 {
		return WorkerMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("worker event payload is missing"))
	}
	parsePayload := parseArgs[0]
	if parsePayload.IsUndefined() || parsePayload.IsNull() {
		return WorkerMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("worker event payload is missing"))
	}
	parseData := parsePayload.Get("data")
	if parseData.IsUndefined() || parseData.IsNull() {
		return WorkerMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("worker message is missing data"))
	}
	parseValue, parseErr := jsValueToGo(parseOp, parseTarget, parseData)
	if parseErr != nil {
		return WorkerMessage{}, parseErr
	}
	parseMessage := workerMessageFromGo(parseValue)
	parseMessage.Ports = messagePortsFromValue(parseTarget, parsePayload.Get("ports"))
	return parseMessage, nil
}

func workerMessageFromGo(parseValue any) WorkerMessage {
	parseMessage := WorkerMessage{
		Phase:   "message",
		Payload: parseValue,
	}
	parseData, parseOk := parseValue.(map[string]any)
	if !parseOk {
		return parseMessage
	}
	if parseId := workerStringField(parseData, "id"); parseId != "" {
		parseMessage.ID = parseId
	}
	if parsePhase := workerStringField(parseData, "phase"); parsePhase != "" {
		parseMessage.Phase = parsePhase
	}
	if parseName := workerStringField(parseData, "name"); parseName != "" {
		parseMessage.Name = parseName
	} else if parseName2 := workerStringField(parseData, "type"); parseName2 != "" {
		parseMessage.Name = parseName2
	}
	if parsePayload, parseOk2 := parseData["payload"]; parseOk2 {
		parseMessage.Payload = parsePayload
	}
	if parseRemoteErr := workerStringField(parseData, "error"); parseRemoteErr != "" {
		parseMessage.Error = parseRemoteErr
	}
	return parseMessage
}

func messagePortMessageFromEvent(parseOp string, parseTarget string, parseArgs []js.Value) (MessagePortMessage, error) {
	if len(parseArgs) == 0 {
		return MessagePortMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("message port event payload is missing"))
	}
	parseEvent := parseArgs[0]
	if parseEvent.IsUndefined() || parseEvent.IsNull() {
		return MessagePortMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("message port event payload is missing"))
	}
	parseData := parseEvent.Get("data")
	if parseData.IsUndefined() || parseData.IsNull() {
		return MessagePortMessage{}, wrapError(parseOp, parseTarget, CodeDecode, errors.New("message port message is missing data"))
	}
	parseValue, parseErr := jsValueToGo(parseOp, parseTarget, parseData)
	if parseErr != nil {
		return MessagePortMessage{}, parseErr
	}
	return MessagePortMessage{
		Payload: parseValue,
		Ports:   messagePortsFromValue(parseTarget, parseEvent.Get("ports")),
	}, nil
}

func messagePortsFromValue(parseTarget string, parsePorts js.Value) []MessagePort {
	if parsePorts.IsUndefined() || parsePorts.IsNull() {
		return nil
	}
	parseCount := parsePorts.Length()
	if parseCount == 0 {
		return nil
	}
	parseResolved := make([]MessagePort, 0, parseCount)
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		parseRawPort := parsePorts.Index(parseIndex)
		if parseRawPort.IsUndefined() || parseRawPort.IsNull() {
			continue
		}
		parseResolved = append(parseResolved, newMessagePort(fmt.Sprintf("%s.port[%d]", parseTarget, parseIndex), parseRawPort))
	}
	return parseResolved
}

func workerStringField(parseData map[string]any, parseKey string) string {
	parseValue, parseOk := parseData[parseKey]
	if !parseOk {
		return ""
	}
	switch parseTyped := parseValue.(type) {
	case string:
		return parseTyped
	default:
		return fmt.Sprint(parseTyped)
	}
}

func workerRemoteEnvelopeError(parseMessage WorkerMessage) string {
	if strings.TrimSpace(parseMessage.Error) != "" {
		return parseMessage.Error
	}
	if parseSummary := strings.TrimSpace(fmt.Sprint(parseMessage.Payload)); parseSummary != "" && parseSummary != "<nil>" {
		return parseSummary
	}
	return "worker reported an error"
}

func workerRemoteErrorSummary(parseArgs []js.Value) string {
	if len(parseArgs) == 0 {
		return "worker reported an error"
	}
	if parseMessage := parseArgs[0].Get("message"); !parseMessage.IsUndefined() && !parseMessage.IsNull() {
		return strings.TrimSpace(parseMessage.String())
	}
	return strings.TrimSpace(jsValueSummary(parseArgs[0]))
}

func workerContextError(parseOp string, parseTarget string, parseErr error) error {
	if errors.Is(parseErr, context.DeadlineExceeded) {
		return wrapError(parseOp, parseTarget, CodeTimeout, parseErr)
	}
	return wrapError(parseOp, parseTarget, CodeCancelled, parseErr)
}

func newBroadcastCrossTabChannel(parseName string, parseSource string, parseRaw js.Value) CrossTabChannel {
	var (
		parseMu       sync.Mutex
		parseSequence int64
		isParseActive = true
	)
	parseNextEnvelope := func(parsePayload any) CrossTabEnvelope {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseSequence++
		return CrossTabEnvelope{
			Name:     parseName,
			Payload:  parsePayload,
			Source:   parseSource,
			Sequence: parseSequence,
			SentAt:   time.Now().UTC(),
		}
	}
	parseCurrent := func(parseOp string) (js.Value, error) {
		parseMu.Lock()
		defer parseMu.Unlock()
		if !isParseActive || parseRaw.IsUndefined() || parseRaw.IsNull() {
			return js.Undefined(), wrapError(parseOp, parseName, CodeDisposed, errors.New("cross-tab channel is closed"))
		}
		return parseRaw, nil
	}
	return CrossTabChannel{
		name:      func() string { return parseName },
		transport: func() string { return "broadcast-channel" },
		publish: func(parsePayload2 any) error {
			parseTarget, parseErr := parseCurrent("CrossTabChannel.Publish")
			if parseErr != nil {
				return parseErr
			}
			parseValue, parseErr := goValueToJS("CrossTabChannel.Publish", parseName, parseNextEnvelope(parsePayload2))
			if parseErr != nil {
				return parseErr
			}
			parseTarget.Call("postMessage", parseValue)
			return nil
		},
		publishClientBinary: func(parseMessage ClientMessage) error {
			parseTarget2, parseErr2 := parseCurrent("PublishClientBinaryCrossTab")
			if parseErr2 != nil {
				return parseErr2
			}
			parseEnvelope, parseErr2 := crossTabEnvelopeJS(parseName, parseSource, parseNextEnvelope(parseMessage))
			if parseErr2 != nil {
				return parseErr2
			}
			parseTarget2.Call("postMessage", parseEnvelope)
			return nil
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("CrossTabChannel.Subscribe", parseName, CodeInvalid, errors.New("handler is nil"))
			}
			parseTarget3, parseErr3 := parseCurrent("CrossTabChannel.Subscribe")
			if parseErr3 != nil {
				return Subscription{}, parseErr3
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				if len(parseArgs) == 0 {
					handler(CrossTabEnvelope{Name: parseName}, wrapError("CrossTabChannel.Subscribe", parseName, CodeDecode, errors.New("broadcast message event is missing")))
					return nil
				}
				parseData := parseArgs[0].Get("data")
				parseValue2, parseDecodeErr := jsValueToGo("CrossTabChannel.Subscribe", parseName, parseData)
				if parseDecodeErr != nil {
					handler(CrossTabEnvelope{Name: parseName}, parseDecodeErr)
					return nil
				}
				handler(crossTabEnvelopeFromGo(parseValue2, parseName), nil)
				return nil
			})
			parseTarget3.Call("addEventListener", "message", parseListener)
			return Subscription{cancel: func() {
				parseTarget3.Call("removeEventListener", "message", parseListener)
				parseListener.Release()
			}}, nil
		},
		close: func() error {
			parseTarget4, parseErr4 := parseCurrent("CrossTabChannel.Close")
			if parseErr4 != nil {
				return parseErr4
			}
			parseTarget4.Call("close")
			parseMu.Lock()
			isParseActive = false
			parseRaw = js.Undefined()
			parseMu.Unlock()
			return nil
		},
	}
}

func newStorageCrossTabChannel(parseName string, parseSource string, parseStorageKey string) (CrossTabChannel, error) {
	parseStorage, parseErr := GetLocalStorage()
	if parseErr != nil {
		return CrossTabChannel{}, parseErr
	}
	parseWindow, parseErr := globalProperty("EventTarget", "window")
	if parseErr != nil {
		return CrossTabChannel{}, parseErr
	}
	var (
		parseMu       sync.Mutex
		parseSequence int64
		isParseActive = true
	)
	parseNextEnvelope := func(parsePayload any) CrossTabEnvelope {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseSequence++
		return CrossTabEnvelope{
			Name:     parseName,
			Payload:  parsePayload,
			Source:   parseSource,
			Sequence: parseSequence,
			SentAt:   time.Now().UTC(),
		}
	}
	parseEnsureActive := func(parseOp string) error {
		parseMu.Lock()
		defer parseMu.Unlock()
		if !isParseActive {
			return wrapError(parseOp, parseName, CodeDisposed, errors.New("cross-tab channel is closed"))
		}
		return nil
	}
	return CrossTabChannel{
		name:      func() string { return parseName },
		transport: func() string { return "storage-event" },
		publish: func(parsePayload2 any) error {
			if parseErr2 := parseEnsureActive("CrossTabChannel.Publish"); parseErr2 != nil {
				return parseErr2
			}
			parseData, parseErr3 := json.Marshal(parseNextEnvelope(parsePayload2))
			if parseErr3 != nil {
				return wrapError("CrossTabChannel.Publish", parseName, CodeEncode, parseErr3)
			}
			if parseErr4 := parseStorage.SetItem(parseStorageKey, string(parseData)); parseErr4 != nil {
				return parseErr4
			}
			return parseStorage.RemoveItem(parseStorageKey)
		},
		publishClientBinary: func(parseMessage2 ClientMessage) error {
			return wrapError("PublishClientBinaryCrossTab", parseName, CodeInvalid, errors.New("binary client payloads require broadcast-channel transport"))
		},
		subscribe: func(handler func(CrossTabEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("CrossTabChannel.Subscribe", parseName, CodeInvalid, errors.New("handler is nil"))
			}
			if parseErr5 := parseEnsureActive("CrossTabChannel.Subscribe"); parseErr5 != nil {
				return Subscription{}, parseErr5
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				if len(parseArgs) == 0 {
					return nil
				}
				parseEvent := parseArgs[0]
				if parseEvent.IsUndefined() || parseEvent.IsNull() {
					return nil
				}
				if parseEvent.Get("key").String() != parseStorageKey {
					return nil
				}
				parseNewValue := parseEvent.Get("newValue")
				if parseNewValue.IsUndefined() || parseNewValue.IsNull() || strings.TrimSpace(parseNewValue.String()) == "" {
					return nil
				}
				var parseMessage CrossTabEnvelope
				if parseErr6 := json.Unmarshal([]byte(parseNewValue.String()), &parseMessage); parseErr6 != nil {
					handler(CrossTabEnvelope{Name: parseName}, wrapError("CrossTabChannel.Subscribe", parseName, CodeDecode, parseErr6))
					return nil
				}
				if strings.TrimSpace(parseMessage.Name) == "" {
					parseMessage.Name = parseName
				}
				handler(parseMessage, nil)
				return nil
			})
			parseWindow.Call("addEventListener", "storage", parseListener)
			return Subscription{cancel: func() {
				parseWindow.Call("removeEventListener", "storage", parseListener)
				parseListener.Release()
			}}, nil
		},
		close: func() error {
			if parseErr7 := parseEnsureActive("CrossTabChannel.Close"); parseErr7 != nil {
				return parseErr7
			}
			parseMu.Lock()
			isParseActive = false
			parseMu.Unlock()
			return nil
		},
	}, nil
}

func newWindowChannel(parseName string, parseTargetOrigin string, parsePeer js.Value, isAllowClose bool) WindowChannel {
	parseSource := fmt.Sprintf("%s-%d", parseName, time.Now().UnixNano())
	parseRawWindow := js.Global().Get("window")
	return WindowChannel{
		name:         func() string { return parseName },
		targetOrigin: func() string { return parseTargetOrigin },
		publish: func(parsePayload any) error {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("WindowChannel.Publish", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseClosed := parsePeer.Get("closed"); !parseClosed.IsUndefined() && !parseClosed.IsNull() && parseClosed.Bool() {
				return wrapError("WindowChannel.Publish", parseName, CodeDisposed, errors.New("window channel peer is closed"))
			}
			parseValue, parseErr := goValueToJS("WindowChannel.Publish", parseName, WindowEnvelope{
				Name:    parseName,
				Payload: parsePayload,
				Source:  parseSource,
				SentAt:  time.Now().UTC(),
			})
			if parseErr != nil {
				return parseErr
			}
			parsePeer.Call("postMessage", parseValue, parseTargetOrigin)
			return nil
		},
		publishClientBinary: func(parseMessage ClientMessage) error {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("PublishClientBinaryWindow", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseClosed2 := parsePeer.Get("closed"); !parseClosed2.IsUndefined() && !parseClosed2.IsNull() && parseClosed2.Bool() {
				return wrapError("PublishClientBinaryWindow", parseName, CodeDisposed, errors.New("window channel peer is closed"))
			}
			parseEnvelope, parseErr2 := windowEnvelopeJS(parseName, parseSource, parseMessage)
			if parseErr2 != nil {
				return parseErr2
			}
			parsePeer.Call("postMessage", parseEnvelope, parseTargetOrigin)
			return nil
		},
		subscribe: func(handler func(WindowEnvelope, error)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("WindowChannel.Subscribe", parseName, CodeInvalid, errors.New("handler is nil"))
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				if len(parseArgs) == 0 {
					handler(WindowEnvelope{Name: parseName}, wrapError("WindowChannel.Subscribe", parseName, CodeDecode, errors.New("message event is missing")))
					return nil
				}
				parseEvent := parseArgs[0]
				if parseEvent.IsUndefined() || parseEvent.IsNull() {
					return nil
				}
				parseEventSource := parseEvent.Get("source")
				if !parseEventSource.IsUndefined() && !parseEventSource.IsNull() && !parseEventSource.Equal(parsePeer) {
					return nil
				}
				if parseTargetOrigin != "*" {
					parseOrigin := strings.TrimSpace(parseEvent.Get("origin").String())
					if parseOrigin != "" && parseOrigin != parseTargetOrigin {
						handler(WindowEnvelope{Name: parseName}, wrapError("WindowChannel.Subscribe", parseName, CodeUnauthorized, errors.New("message origin does not match target origin")))
						return nil
					}
				}
				parseValue2, parseDecodeErr := jsValueToGo("WindowChannel.Subscribe", parseName, parseEvent.Get("data"))
				if parseDecodeErr != nil {
					handler(WindowEnvelope{Name: parseName}, parseDecodeErr)
					return nil
				}
				handler(windowEnvelopeFromGo(parseValue2, parseName), nil)
				return nil
			})
			parseRawWindow.Call("addEventListener", "message", parseListener)
			return Subscription{cancel: func() {
				parseRawWindow.Call("removeEventListener", "message", parseListener)
				parseListener.Release()
			}}, nil
		},
		focus: func() error {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("WindowChannel.Focus", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseFn := parsePeer.Get("focus"); parseFn.Type() != js.TypeFunction {
				return unavailable("WindowChannel.Focus", parseName)
			}
			parsePeer.Call("focus")
			return nil
		},
		close: func() error {
			if !isAllowClose {
				return unavailable("WindowChannel.Close", parseName)
			}
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return wrapError("WindowChannel.Close", parseName, CodeDisposed, errors.New("window channel peer is unavailable"))
			}
			if parseFn2 := parsePeer.Get("close"); parseFn2.Type() != js.TypeFunction {
				return unavailable("WindowChannel.Close", parseName)
			}
			parsePeer.Call("close")
			return nil
		},
		closed: func() bool {
			if parsePeer.IsUndefined() || parsePeer.IsNull() {
				return true
			}
			parseClosed3 := parsePeer.Get("closed")
			if parseClosed3.IsUndefined() || parseClosed3.IsNull() {
				return false
			}
			return parseClosed3.Bool()
		},
	}
}

func resolveCrossTabStorageKey(parseName string, parseOverride string) string {
	parseTrimmed := strings.TrimSpace(parseOverride)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	return "__gwc_cross_tab__:" + parseName
}

func resolveWindowTargetOrigin(parseRawTargetOrigin string) string {
	parseTrimmed := strings.TrimSpace(parseRawTargetOrigin)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return "*"
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.IsUndefined() || parseLocation.IsNull() {
		return "*"
	}
	parseOrigin := strings.TrimSpace(parseLocation.Get("origin").String())
	if parseOrigin == "" {
		return "*"
	}
	return parseOrigin
}

func crossTabEnvelopeFromGo(parseValue any, parseFallbackName string) CrossTabEnvelope {
	parseMessage := CrossTabEnvelope{
		Name:    parseFallbackName,
		Payload: parseValue,
	}
	parseData, parseOk := parseValue.(map[string]any)
	if !parseOk {
		return parseMessage
	}
	if parseName := workerStringField(parseData, "name"); parseName != "" {
		parseMessage.Name = parseName
	}
	if parsePayload, parseOk2 := parseData["payload"]; parseOk2 {
		parseMessage.Payload = parsePayload
	}
	if parseSource := workerStringField(parseData, "source"); parseSource != "" {
		parseMessage.Source = parseSource
	}
	if parseSequence, parseOk3 := crossTabInt64Field(parseData["sequence"]); parseOk3 {
		parseMessage.Sequence = parseSequence
	}
	if parseSentAt, parseOk4 := crossTabTimeField(parseData["sentAt"]); parseOk4 {
		parseMessage.SentAt = parseSentAt
	}
	return parseMessage
}

func windowEnvelopeFromGo(parseValue any, parseFallbackName string) WindowEnvelope {
	parseMessage := WindowEnvelope{
		Name:    parseFallbackName,
		Payload: parseValue,
	}
	parseData, parseOk := parseValue.(map[string]any)
	if !parseOk {
		return parseMessage
	}
	if parseName := workerStringField(parseData, "name"); parseName != "" {
		parseMessage.Name = parseName
	}
	if parsePayload, parseOk2 := parseData["payload"]; parseOk2 {
		parseMessage.Payload = parsePayload
	}
	if parseSource := workerStringField(parseData, "source"); parseSource != "" {
		parseMessage.Source = parseSource
	}
	if parseSentAt, parseOk3 := crossTabTimeField(parseData["sentAt"]); parseOk3 {
		parseMessage.SentAt = parseSentAt
	}
	return parseMessage
}

func crossTabInt64Field(parseValue any) (int64, bool) {
	switch parseTyped := parseValue.(type) {
	case float64:
		return int64(parseTyped), true
	case float32:
		return int64(parseTyped), true
	case int:
		return int64(parseTyped), true
	case int64:
		return parseTyped, true
	case json.Number:
		parseParsed, parseErr := parseTyped.Int64()
		return parseParsed, parseErr == nil
	default:
		return 0, false
	}
}

func crossTabTimeField(parseValue any) (time.Time, bool) {
	parseText, parseOk := parseValue.(string)
	if !parseOk || strings.TrimSpace(parseText) == "" {
		return time.Time{}, false
	}
	parseParsed, parseErr := time.Parse(time.RFC3339Nano, parseText)
	if parseErr != nil {
		return time.Time{}, false
	}
	return parseParsed, true
}

type moduleState struct {
	mu        sync.RWMutex
	specifier string
	value     js.Value
	disposed  bool
}

func (parseM *moduleState) export(parseName string) (js.Value, error) {
	parseM.mu.RLock()
	defer parseM.mu.RUnlock()
	if parseM.disposed {
		return js.Undefined(), wrapError("Module", parseM.specifier, CodeDisposed, errors.New("module handle is disposed"))
	}
	parseRaw := parseM.value.Get(parseName)
	if parseRaw.IsUndefined() || parseRaw.IsNull() {
		return js.Undefined(), wrapError("Module.Value", parseName, CodeMissingExport, errors.New("export not found"))
	}
	return parseRaw, nil
}

func (parseM *moduleState) callDefault(parseCtx context.Context, parseArgs ...any) (any, error) {
	parseRaw, parseErr := parseM.export("default")
	if parseErr != nil {
		return nil, parseErr
	}
	if parseRaw.Type() != js.TypeFunction {
		return nil, wrapError("Module.CallDefault", "default", CodeNotFunction, errors.New("default export is not a function"))
	}
	parseJsArgs := make([]interface{}, 0, len(parseArgs))
	for _, parseArg := range parseArgs {
		parseValue, parseErr2 := goValueToJS("Module.CallDefault", "default", parseArg)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		parseJsArgs = append(parseJsArgs, parseValue)
	}
	parseResult, parseErr := awaitValue(parseCtx, "Module.CallDefault", "default", parseRaw.Invoke(parseJsArgs...))
	if parseErr != nil {
		return nil, parseErr
	}
	return jsValueToGo("Module.CallDefault", "default", parseResult)
}

func (parseM *moduleState) dispose() error {
	parseM.mu.Lock()
	defer parseM.mu.Unlock()
	parseM.disposed = true
	parseM.value = js.Undefined()
	return nil
}

func newEventTarget(parseName string, parseRaw js.Value) EventTarget {
	return EventTarget{
		dispatch: func(parseEventName string, parseDetail2 any) error {
			parseCustomEventCtor := js.Global().Get("CustomEvent")
			if parseCustomEventCtor.IsUndefined() || parseCustomEventCtor.IsNull() {
				return unavailable("EventTarget.Dispatch", parseName)
			}
			parseDetailValue, parseErr := goValueToJS("EventTarget.Dispatch", parseEventName, parseDetail2)
			if parseErr != nil {
				return parseErr
			}
			parseInit := js.Global().Get("Object").New()
			parseInit.Set("detail", parseDetailValue)
			parseRaw.Call("dispatchEvent", parseCustomEventCtor.New(parseEventName, parseInit))
			return nil
		},
		listen: func(parseEventName2 string, handler func(BrowserEvent)) (Subscription, error) {
			if handler == nil {
				return Subscription{}, wrapError("EventTarget.Listen", parseEventName2, CodeInvalid, errors.New("handler is nil"))
			}
			if parseAdd := parseRaw.Get("addEventListener"); parseAdd.Type() != js.TypeFunction {
				return Subscription{}, unavailable("EventTarget.Listen", parseName)
			}
			parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
				if len(parseArgs) == 0 {
					handler(BrowserEvent{Type: parseEventName2})
					return nil
				}
				parseEvent := parseArgs[0]
				parseDetail, _ := jsValueToGo("EventTarget.Listen", parseEventName2, parseEvent.Get("detail"))
				handler(BrowserEvent{
					Type:          parseEvent.Get("type").String(),
					Detail:        parseDetail,
					Target:        elementFromJSValue(parseEvent.Get("target")),
					CurrentTarget: elementFromJSValue(parseEvent.Get("currentTarget")),
				})
				return nil
			})
			parseRaw.Call("addEventListener", parseEventName2, parseListener)
			return Subscription{cancel: func() {
				parseRaw.Call("removeEventListener", parseEventName2, parseListener)
				parseListener.Release()
			}}, nil
		},
	}
}

func newElement(parseName string, parseRaw js.Value) Element {
	return Element{
		raw:       parseRaw,
		tagName:   func() string { return parseRaw.Get("tagName").String() },
		id:        func() string { return parseRaw.Get("id").String() },
		className: func() string { return parseRaw.Get("className").String() },
		focus: func() error {
			if parseFn := parseRaw.Get("focus"); parseFn.Type() != js.TypeFunction {
				return unavailable("Element.Focus", parseName)
			}
			parseRaw.Call("focus")
			return nil
		},
		blur: func() error {
			if parseFn2 := parseRaw.Get("blur"); parseFn2.Type() != js.TypeFunction {
				return unavailable("Element.Blur", parseName)
			}
			parseRaw.Call("blur")
			return nil
		},
		click: func() error {
			if parseFn3 := parseRaw.Get("click"); parseFn3.Type() != js.TypeFunction {
				return unavailable("Element.Click", parseName)
			}
			parseRaw.Call("click")
			return nil
		},
		setScrollTop: func(parseScrollTop float64) error {
			parseRaw.Set("scrollTop", parseScrollTop)
			return nil
		},
		scrollIntoView: func(parseOptions ScrollIntoViewOptions) error {
			if parseFn4 := parseRaw.Get("scrollIntoView"); parseFn4.Type() != js.TypeFunction {
				return unavailable("Element.ScrollIntoView", parseName)
			}
			parseInit := js.Global().Get("Object").New()
			hasOptions := false
			if strings.TrimSpace(parseOptions.Behavior) != "" {
				parseInit.Set("behavior", parseOptions.Behavior)
				hasOptions = true
			}
			if strings.TrimSpace(parseOptions.Block) != "" {
				parseInit.Set("block", parseOptions.Block)
				hasOptions = true
			}
			if strings.TrimSpace(parseOptions.Inline) != "" {
				parseInit.Set("inline", parseOptions.Inline)
				hasOptions = true
			}
			if !hasOptions {
				parseRaw.Call("scrollIntoView")
				return nil
			}
			parseRaw.Call("scrollIntoView", parseInit)
			return nil
		},
		boundingClientRect: func() (Rect, error) {
			if parseFn5 := parseRaw.Get("getBoundingClientRect"); parseFn5.Type() != js.TypeFunction {
				return Rect{}, unavailable("Element.BoundingClientRect", parseName)
			}
			return rectFromJSValue(parseRaw.Call("getBoundingClientRect")), nil
		},
		events: func() (EventTarget, error) {
			return newEventTarget(parseName, parseRaw), nil
		},
		observeResize: func(handler func(ResizeEntry)) (Subscription, error) {
			return observeResize(parseName, parseRaw, handler)
		},
		observeIntersection: func(parseOptions2 IntersectionObserverOptions, handler func(IntersectionEntry)) (Subscription, error) {
			return observeIntersection(parseName, parseRaw, parseOptions2, handler)
		},
		scrollMetrics: func() (float64, float64, float64, error) {
			return parseRaw.Get("scrollTop").Float(), parseRaw.Get("scrollHeight").Float(), parseRaw.Get("clientHeight").Float(), nil
		},
	}
}

func elementFromJSValue(parseValue js.Value) Element {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return Element{}
	}
	return newElement("element", parseValue)
}

func rectFromJSValue(parseValue js.Value) Rect {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return Rect{}
	}
	return Rect{
		X:      parseValue.Get("x").Float(),
		Y:      parseValue.Get("y").Float(),
		Width:  parseValue.Get("width").Float(),
		Height: parseValue.Get("height").Float(),
		Top:    parseValue.Get("top").Float(),
		Right:  parseValue.Get("right").Float(),
		Bottom: parseValue.Get("bottom").Float(),
		Left:   parseValue.Get("left").Float(),
	}
}

func observeResize(parseName string, parseRaw js.Value, parseHandler func(ResizeEntry)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("Element.ObserveResize", parseName, CodeInvalid, errors.New("handler is nil"))
	}
	parseCtor := js.Global().Get("ResizeObserver")
	if parseCtor.IsUndefined() || parseCtor.IsNull() {
		return Subscription{}, unavailable("Element.ObserveResize", parseName)
	}
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseEntries := parseArgs[0]
		parseLength := parseEntries.Get("length").Int()
		for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
			parseEntry := parseEntries.Index(parseIndex)
			parseHandler(ResizeEntry{
				Target:      elementFromJSValue(parseEntry.Get("target")),
				ContentRect: rectFromJSValue(parseEntry.Get("contentRect")),
			})
		}
		return nil
	})
	parseObserver := parseCtor.New(parseCallback)
	parseObserver.Call("observe", parseRaw)
	return Subscription{cancel: func() {
		parseObserver.Call("disconnect")
		parseCallback.Release()
	}}, nil
}

func observeIntersection(parseName string, parseRaw js.Value, parseOptions IntersectionObserverOptions, parseHandler func(IntersectionEntry)) (Subscription, error) {
	if parseHandler == nil {
		return Subscription{}, wrapError("Element.ObserveIntersection", parseName, CodeInvalid, errors.New("handler is nil"))
	}
	parseCtor := js.Global().Get("IntersectionObserver")
	if parseCtor.IsUndefined() || parseCtor.IsNull() {
		return Subscription{}, unavailable("Element.ObserveIntersection", parseName)
	}
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseEntries := parseArgs[0]
		parseLength := parseEntries.Get("length").Int()
		for parseIndex := 0; parseIndex < parseLength; parseIndex++ {
			parseEntry := parseEntries.Index(parseIndex)
			var parseRootBounds *Rect
			if parseBounds := parseEntry.Get("rootBounds"); !parseBounds.IsUndefined() && !parseBounds.IsNull() {
				parseRect := rectFromJSValue(parseBounds)
				parseRootBounds = &parseRect
			}
			parseHandler(IntersectionEntry{
				Target:             elementFromJSValue(parseEntry.Get("target")),
				IsIntersecting:     parseEntry.Get("isIntersecting").Bool(),
				IntersectionRatio:  parseEntry.Get("intersectionRatio").Float(),
				BoundingClientRect: rectFromJSValue(parseEntry.Get("boundingClientRect")),
				IntersectionRect:   rectFromJSValue(parseEntry.Get("intersectionRect")),
				RootBounds:         parseRootBounds,
			})
		}
		return nil
	})
	if len(parseOptions.Thresholds) == 0 && parseOptions.RootMargin == "" && parseOptions.Root.raw == nil {
		parseObserver := parseCtor.New(parseCallback)
		parseObserver.Call("observe", parseRaw)
		return Subscription{cancel: func() {
			parseObserver.Call("disconnect")
			parseCallback.Release()
		}}, nil
	}
	parseInit := js.Global().Get("Object").New()
	if parseOptions.RootMargin != "" {
		parseInit.Set("rootMargin", parseOptions.RootMargin)
	}
	if len(parseOptions.Thresholds) > 0 {
		parseThresholds := js.Global().Get("Array").New()
		for _, parseThreshold := range parseOptions.Thresholds {
			parseThresholds.Call("push", parseThreshold)
		}
		parseInit.Set("threshold", parseThresholds)
	}
	if parseRoot, parseOk := parseOptions.Root.raw.(js.Value); parseOk && !parseRoot.IsUndefined() && !parseRoot.IsNull() {
		parseInit.Set("root", parseRoot)
	}
	parseObserver2 := parseCtor.New(parseCallback, parseInit)
	parseObserver2.Call("observe", parseRaw)
	return Subscription{cancel: func() {
		parseObserver2.Call("disconnect")
		parseCallback.Release()
	}}, nil
}

func globalProperty(parseOp string, parseName string) (js.Value, error) {
	parseValue := js.Global().Get(parseName)
	if (parseValue.IsUndefined() || parseValue.IsNull()) && parseName != "window" {
		if parseWindow := js.Global().Get("window"); !parseWindow.IsUndefined() && !parseWindow.IsNull() {
			parseValue = parseWindow.Get(parseName)
		}
	}
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return js.Undefined(), unavailable(parseOp, parseName)
	}
	return parseValue, nil
}

func globalPath(parseOp string, parseHead string, parseTail string) (js.Value, error) {
	parseRoot, parseErr := globalProperty(parseOp, parseHead)
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	parseValue := parseRoot.Get(parseTail)
	if (parseValue.IsUndefined() || parseValue.IsNull()) && parseHead != "window" {
		if parseWindow := js.Global().Get("window"); !parseWindow.IsUndefined() && !parseWindow.IsNull() {
			parseNextRoot := parseWindow.Get(parseHead)
			if !parseNextRoot.IsUndefined() && !parseNextRoot.IsNull() {
				parseValue = parseNextRoot.Get(parseTail)
			}
		}
	}
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return js.Undefined(), unavailable(parseOp, parseHead+"."+parseTail)
	}
	return parseValue, nil
}

func durationMS(parseValue time.Duration) int {
	if parseValue <= 0 {
		return 0
	}
	return int(parseValue / time.Millisecond)
}

func awaitValue(parseCtx context.Context, parseOp string, parseTarget string, parseValue js.Value) (js.Value, error) {
	if !isPromise(parseValue) {
		return parseValue, nil
	}
	parseResolvedCh := make(chan js.Value, 1)
	parseRejectedCh := make(chan js.Value, 1)
	var (
		parseResolve js.Func
		parseReject  js.Func
		parseOnce    sync.Once
	)
	parseCleanup := func() {
		parseResolve.Release()
		parseReject.Release()
	}
	parseResolve = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseOnce.Do(func() {
			if len(parseArgs) > 0 {
				parseResolvedCh <- parseArgs[0]
			} else {
				parseResolvedCh <- js.Undefined()
			}
			parseCleanup()
		})
		return nil
	})
	parseReject = js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		parseOnce.Do(func() {
			if len(parseArgs2) > 0 {
				parseRejectedCh <- parseArgs2[0]
			} else {
				parseRejectedCh <- js.ValueOf("promise rejected")
			}
			parseCleanup()
		})
		return nil
	})
	parseValue.Call("then", parseResolve)
	parseValue.Call("catch", parseReject)

	select {
	case parseResolved := <-parseResolvedCh:
		return parseResolved, nil
	case parseRejected := <-parseRejectedCh:
		return js.Undefined(), wrapError(parseOp, parseTarget, CodePromiseRejected, errors.New(jsValueSummary(parseRejected)))
	case <-parseCtx.Done():
		return js.Undefined(), wrapError(parseOp, parseTarget, CodePromiseRejected, parseCtx.Err())
	}
}

func isPromise(parseValue js.Value) bool {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return false
	}
	if parseValue.Type() != js.TypeObject && parseValue.Type() != js.TypeFunction {
		return false
	}
	parseThen := parseValue.Get("then")
	return parseThen.Type() == js.TypeFunction
}

func goValueToJS(parseOp string, parseTarget string, parseValue any) (interface{}, error) {
	switch parseTyped := parseValue.(type) {
	case nil:
		return js.Null(), nil
	case []byte:
		parseArray := js.Global().Get("Uint8Array").New(len(parseTyped))
		js.CopyBytesToJS(parseArray, parseTyped)
		return parseArray, nil
	case SharedBuffer:
		if parseRaw, parseOk := getSharedBufferRaw(parseTyped); parseOk && !parseRaw.IsUndefined() && !parseRaw.IsNull() {
			return parseRaw, nil
		}
		return js.Undefined(), unavailable(parseOp, parseTarget)
	case Value:
		if parseRaw, parseOk := parseTyped.rawValue(); parseOk {
			return parseRaw, nil
		}
		return js.Undefined(), unavailable(parseOp, parseTarget)
	case js.Value:
		return parseTyped, nil
	case bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return js.ValueOf(parseTyped), nil
	default:
		parseData, parseErr := json.Marshal(parseValue)
		if parseErr != nil {
			return nil, wrapError(parseOp, parseTarget, CodeEncode, parseErr)
		}
		return js.Global().Get("JSON").Call("parse", string(parseData)), nil
	}
}

func goValuesToJS(parseOp string, parseTarget string, parseValues ...any) ([]interface{}, error) {
	if len(parseValues) == 0 {
		return nil, nil
	}
	parseConverted := make([]interface{}, len(parseValues))
	for parseIndex, parseValue := range parseValues {
		parseJsValue, parseErr := goValueToJS(parseOp, parseTarget, parseValue)
		if parseErr != nil {
			return nil, parseErr
		}
		parseConverted[parseIndex] = parseJsValue
	}
	return parseConverted, nil
}

func jsValueToGo(parseOp string, parseTarget string, parseValue js.Value) (any, error) {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return nil, nil
	}
	if parseSharedBuffer, parseOk := getSharedBufferFromJS(parseTarget, parseValue); parseOk {
		return parseSharedBuffer, nil
	}
	if parseBytes, parseOk := jsValueToBytes(parseValue); parseOk {
		return parseBytes, nil
	}
	switch parseValue.Type() {
	case js.TypeBoolean:
		return parseValue.Bool(), nil
	case js.TypeString:
		return parseValue.String(), nil
	case js.TypeNumber:
		return parseValue.Float(), nil
	case js.TypeObject:
		if parseValue.InstanceOf(js.Global().Get("Array")) {
			parseItems := make([]any, parseValue.Length())
			for parseIndex := 0; parseIndex < parseValue.Length(); parseIndex++ {
				parseItem, parseErr := jsValueToGo(parseOp, parseTarget, parseValue.Index(parseIndex))
				if parseErr != nil {
					return nil, parseErr
				}
				parseItems[parseIndex] = parseItem
			}
			return parseItems, nil
		}
		parseKeys := js.Global().Get("Object").Call("keys", parseValue)
		parseDecoded := make(map[string]any, parseKeys.Length())
		for parseIndex2 := 0; parseIndex2 < parseKeys.Length(); parseIndex2++ {
			parseKey := parseKeys.Index(parseIndex2).String()
			parseItem2, parseErr2 := jsValueToGo(parseOp, parseTarget, parseValue.Get(parseKey))
			if parseErr2 != nil {
				return nil, parseErr2
			}
			parseDecoded[parseKey] = parseItem2
		}
		return parseDecoded, nil
	case js.TypeFunction:
		return nil, wrapError(parseOp, parseTarget, CodeDecode, errors.New("function values are not serializable"))
	default:
		return parseValue.String(), nil
	}
}

func jsValueToBytes(parseValue js.Value) ([]byte, bool) {
	if parseValue.IsUndefined() || parseValue.IsNull() {
		return nil, false
	}
	parseArrayBuffer := js.Global().Get("ArrayBuffer")
	if parseArrayBuffer.Type() == js.TypeFunction && parseValue.InstanceOf(parseArrayBuffer) {
		parseView := js.Global().Get("Uint8Array").New(parseValue)
		parseBytes := make([]byte, parseView.Length())
		js.CopyBytesToGo(parseBytes, parseView)
		return parseBytes, true
	}
	if parseArrayBuffer.Type() != js.TypeFunction {
		return nil, false
	}
	isView := parseArrayBuffer.Get("isView")
	if isView.Type() != js.TypeFunction || !isView.Invoke(parseValue).Bool() {
		return nil, false
	}
	parseView2 := js.Global().Get("Uint8Array").New(parseValue.Get("buffer"), parseValue.Get("byteOffset"), parseValue.Get("byteLength"))
	parseBytes2 := make([]byte, parseView2.Length())
	js.CopyBytesToGo(parseBytes2, parseView2)
	return parseBytes2, true
}

// getSharedBufferFromJS wraps a SharedArrayBuffer JS value as SharedBuffer.
func getSharedBufferFromJS(parseTarget string, parseValue js.Value) (SharedBuffer, bool) {
	parseCtor := getGlobalValue("SharedArrayBuffer")
	if parseCtor.Type() != js.TypeFunction || !parseValue.InstanceOf(parseCtor) {
		return SharedBuffer{}, false
	}
	return buildSharedBuffer(parseTarget, parseValue), true
}

func crossTabEnvelopeJS(parseName string, parseSource string, parseEnvelope CrossTabEnvelope) (js.Value, error) {
	parseValue := js.Global().Get("Object").New()
	parseValue.Set("name", parseName)
	parseValue.Set("source", parseSource)
	parseValue.Set("sequence", parseEnvelope.Sequence)
	parseValue.Set("sentAt", parseEnvelope.SentAt.Format(time.RFC3339Nano))
	parsePayload, parseErr := goValueToJSStructured("CrossTabChannel.Publish", parseName, parseEnvelope.Payload)
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	parseValue.Set("payload", parsePayload)
	return parseValue, nil
}

func windowEnvelopeJS(parseName string, parseSource string, parsePayload any) (js.Value, error) {
	parseValue := js.Global().Get("Object").New()
	parseValue.Set("name", parseName)
	parseValue.Set("source", parseSource)
	parseValue.Set("sentAt", time.Now().UTC().Format(time.RFC3339Nano))
	parseConverted, parseErr := goValueToJSStructured("WindowChannel.Publish", parseName, parsePayload)
	if parseErr != nil {
		return js.Undefined(), parseErr
	}
	parseValue.Set("payload", parseConverted)
	return parseValue, nil
}

func postStructuredMessageJS(parseOp string, parseTarget string, parseRaw js.Value, parsePayload any, parsePorts ...MessagePort) (parseErr error) {
	defer recoverInteropException(parseOp, parseTarget, &parseErr)
	parseConverted, parseErr := goValueToJSStructured(parseOp, parseTarget, parsePayload)
	if parseErr != nil {
		return parseErr
	}
	if len(parsePorts) == 0 {
		parseRaw.Call("postMessage", parseConverted)
		return nil
	}
	parseTransfers, parseErr := messagePortTransferListJS(parseOp, parseTarget, parsePorts)
	if parseErr != nil {
		return parseErr
	}
	parseRaw.Call("postMessage", parseConverted, parseTransfers)
	return nil
}

func messagePortTransferListJS(parseOp string, parseTarget string, parsePorts []MessagePort) (js.Value, error) {
	parseTransfers := js.Global().Get("Array").New(len(parsePorts))
	for parseIndex, parsePort := range parsePorts {
		parseRawPort, parseOk := parsePort.raw.(js.Value)
		if !parseOk || parseRawPort.IsUndefined() || parseRawPort.IsNull() {
			return js.Undefined(), wrapError(parseOp, parseTarget, CodeInvalid, fmt.Errorf("message port %d is unavailable", parseIndex))
		}
		parseTransfers.SetIndex(parseIndex, parseRawPort)
	}
	return parseTransfers, nil
}

func goValueToJSStructured(parseOp string, parseTarget string, parseValue any) (interface{}, error) {
	switch parseTyped := parseValue.(type) {
	case WorkerMessage:
		parseMessage := js.Global().Get("Object").New()
		if strings.TrimSpace(parseTyped.ID) != "" {
			parseMessage.Set("id", parseTyped.ID)
		}
		if strings.TrimSpace(parseTyped.Phase) != "" {
			parseMessage.Set("phase", parseTyped.Phase)
		}
		if strings.TrimSpace(parseTyped.Name) != "" {
			parseMessage.Set("name", parseTyped.Name)
		}
		if strings.TrimSpace(parseTyped.Error) != "" {
			parseMessage.Set("error", parseTyped.Error)
		}
		if parseTyped.Payload != nil {
			parsePayload, parseErr := buildStructuredJSPayload(parseOp, parseTarget, parseTyped.Payload, map[uintptr]struct{}{})
			if parseErr != nil {
				return nil, parseErr
			}
			parseMessage.Set("payload", parsePayload)
		}
		return parseMessage, nil
	case ClientMessage:
		parseMessage := js.Global().Get("Object").New()
		if strings.TrimSpace(parseTyped.ID) != "" {
			parseMessage.Set("id", parseTyped.ID)
		}
		parseMessage.Set("kind", string(parseTyped.Kind))
		parseMessage.Set("topic", parseTyped.Topic)
		parseMessage.Set("source", clientIdentityJS(parseTyped.Source))
		if parseTyped.Capabilities != nil {
			parseMessage.Set("capabilities", clientCapabilitiesJS(*parseTyped.Capabilities))
		}
		if strings.TrimSpace(parseTyped.Target) != "" {
			parseMessage.Set("target", parseTyped.Target)
		}
		if parseTyped.Encoding != "" {
			parseMessage.Set("encoding", string(parseTyped.Encoding))
		}
		if strings.TrimSpace(parseTyped.ContentType) != "" {
			parseMessage.Set("contentType", parseTyped.ContentType)
		}
		if strings.TrimSpace(parseTyped.Revision) != "" {
			parseMessage.Set("revision", parseTyped.Revision)
		}
		if strings.TrimSpace(parseTyped.Error) != "" {
			parseMessage.Set("error", parseTyped.Error)
		}
		if !parseTyped.SentAt.IsZero() {
			parseMessage.Set("sentAt", parseTyped.SentAt.Format(time.RFC3339Nano))
		}
		if parseTyped.Payload != nil {
			parsePayload, parseErr := buildStructuredJSPayload(parseOp, parseTarget, parseTyped.Payload, map[uintptr]struct{}{})
			if parseErr != nil {
				return nil, parseErr
			}
			parseMessage.Set("payload", parsePayload)
		}
		return parseMessage, nil
	default:
		return buildStructuredJSPayload(parseOp, parseTarget, parseValue, map[uintptr]struct{}{})
	}
}

// buildStructuredJSPayload recursively converts JSON-shaped Go values into JS
// objects while preserving `SharedBuffer` leaves for structured-clone paths.
func buildStructuredJSPayload(parseOp string, parseTarget string, parseValue any, parseSeen map[uintptr]struct{}) (interface{}, error) {
	if parseValue == nil {
		return js.Null(), nil
	}
	return buildStructuredJSReflect(parseOp, parseTarget, reflect.ValueOf(parseValue), parseSeen)
}

// buildStructuredJSReflect walks one reflected Go value into its JS structured
// clone equivalent while preserving supported interop wrapper leaves.
func buildStructuredJSReflect(parseOp string, parseTarget string, parseValue reflect.Value, parseSeen map[uintptr]struct{}) (interface{}, error) {
	if !parseValue.IsValid() {
		return js.Null(), nil
	}
	if parseValue.CanInterface() {
		switch parseValue.Interface().(type) {
		case nil, []byte, SharedBuffer, Value, js.Value, bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		case json.Marshaler, encoding.TextMarshaler:
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		}
	}

	switch parseValue.Kind() {
	case reflect.Interface, reflect.Pointer:
		if parseValue.IsNil() {
			return js.Null(), nil
		}
		parseVisit, parseErr := buildStructuredJSVisit(parseOp, parseTarget, parseValue, parseSeen)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseVisit != 0 {
			defer delete(parseSeen, parseVisit)
		}
		return buildStructuredJSReflect(parseOp, parseTarget, parseValue.Elem(), parseSeen)
	case reflect.Map:
		if parseValue.IsNil() {
			return js.Null(), nil
		}
		if parseValue.Type().Key().Kind() != reflect.String {
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		}
		parseVisit, parseErr := buildStructuredJSVisit(parseOp, parseTarget, parseValue, parseSeen)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseVisit != 0 {
			defer delete(parseSeen, parseVisit)
		}
		parseObject := js.Global().Get("Object").New()
		for _, parseKey := range parseValue.MapKeys() {
			parseConverted, parseErr := buildStructuredJSReflect(parseOp, parseTarget, parseValue.MapIndex(parseKey), parseSeen)
			if parseErr != nil {
				return nil, parseErr
			}
			parseObject.Set(parseKey.String(), parseConverted)
		}
		return parseObject, nil
	case reflect.Slice, reflect.Array:
		parseVisit, parseErr := buildStructuredJSVisit(parseOp, parseTarget, parseValue, parseSeen)
		if parseErr != nil {
			return nil, parseErr
		}
		if parseVisit != 0 {
			defer delete(parseSeen, parseVisit)
		}
		parseArray := js.Global().Get("Array").New(parseValue.Len())
		for parseIndex := 0; parseIndex < parseValue.Len(); parseIndex++ {
			parseConverted, parseErr := buildStructuredJSReflect(parseOp, parseTarget, parseValue.Index(parseIndex), parseSeen)
			if parseErr != nil {
				return nil, parseErr
			}
			parseArray.SetIndex(parseIndex, parseConverted)
		}
		return parseArray, nil
	case reflect.Struct:
		parseObject := js.Global().Get("Object").New()
		parseType := parseValue.Type()
		for parseIndex := 0; parseIndex < parseValue.NumField(); parseIndex++ {
			parseField := parseType.Field(parseIndex)
			parseFieldName, isParseOmitEmpty, isParseSkip := buildStructuredJSFieldName(parseField)
			if isParseSkip {
				continue
			}
			parseFieldValue := parseValue.Field(parseIndex)
			if isParseOmitEmpty && isStructuredEmptyValue(parseFieldValue) {
				continue
			}
			parseConverted, parseErr := buildStructuredJSReflect(parseOp, parseFieldName, parseFieldValue, parseSeen)
			if parseErr != nil {
				return nil, parseErr
			}
			parseObject.Set(parseFieldName, parseConverted)
		}
		return parseObject, nil
	default:
		if parseValue.CanInterface() {
			return goValueToJS(parseOp, parseTarget, parseValue.Interface())
		}
		return nil, wrapError(parseOp, parseTarget, CodeEncode, errors.New("value is not serializable"))
	}
}

// buildStructuredJSVisit tracks pointer-like values so recursive structured
// conversion fails cleanly on cycles instead of recursing forever.
func buildStructuredJSVisit(parseOp string, parseTarget string, parseValue reflect.Value, parseSeen map[uintptr]struct{}) (uintptr, error) {
	if parseSeen == nil {
		return 0, nil
	}
	switch parseValue.Kind() {
	case reflect.Map:
		if parseValue.IsNil() {
			return 0, nil
		}
		parsePointer := parseValue.Pointer()
		if _, parseOk := parseSeen[parsePointer]; parseOk {
			return 0, wrapError(parseOp, parseTarget, CodeEncode, errors.New("cyclic structured payload is unsupported"))
		}
		parseSeen[parsePointer] = struct{}{}
		return parsePointer, nil
	case reflect.Pointer:
		if parseValue.IsNil() {
			return 0, nil
		}
		parsePointer := parseValue.Pointer()
		if _, parseOk := parseSeen[parsePointer]; parseOk {
			return 0, wrapError(parseOp, parseTarget, CodeEncode, errors.New("cyclic structured payload is unsupported"))
		}
		parseSeen[parsePointer] = struct{}{}
		return parsePointer, nil
	case reflect.Slice:
		if parseValue.IsNil() {
			return 0, nil
		}
		parsePointer := parseValue.Pointer()
		if parsePointer == 0 {
			return 0, nil
		}
		if _, parseOk := parseSeen[parsePointer]; parseOk {
			return 0, wrapError(parseOp, parseTarget, CodeEncode, errors.New("cyclic structured payload is unsupported"))
		}
		parseSeen[parsePointer] = struct{}{}
		return parsePointer, nil
	default:
		return 0, nil
	}
}

// buildStructuredJSFieldName resolves the JSON field name and omitempty
// behavior for one exported struct field.
func buildStructuredJSFieldName(parseField reflect.StructField) (string, bool, bool) {
	if parseField.PkgPath != "" {
		return "", false, true
	}
	parseTag := strings.TrimSpace(parseField.Tag.Get("json"))
	if parseTag == "-" {
		return "", false, true
	}
	parseName := parseField.Name
	isParseOmitEmpty := false
	if parseTag != "" {
		parseParts := strings.Split(parseTag, ",")
		if strings.TrimSpace(parseParts[0]) != "" {
			parseName = strings.TrimSpace(parseParts[0])
		}
		for _, parsePart := range parseParts[1:] {
			if strings.TrimSpace(parsePart) == "omitempty" {
				isParseOmitEmpty = true
			}
		}
	}
	return parseName, isParseOmitEmpty, false
}

// isStructuredEmptyValue reports whether one reflected field should be skipped
// for an `omitempty` structured payload field.
func isStructuredEmptyValue(parseValue reflect.Value) bool {
	if !parseValue.IsValid() {
		return true
	}
	switch parseValue.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return parseValue.Len() == 0
	case reflect.Bool:
		return !parseValue.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return parseValue.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return parseValue.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return parseValue.Float() == 0
	case reflect.Interface, reflect.Pointer:
		return parseValue.IsNil()
	default:
		return parseValue.IsZero()
	}
}

func clientIdentityJS(parseIdentity ClientIdentity) js.Value {
	parseValue := js.Global().Get("Object").New()
	parseValue.Set("id", parseIdentity.ID)
	parseValue.Set("app", parseIdentity.App)
	parseValue.Set("surface", parseIdentity.Surface)
	if strings.TrimSpace(parseIdentity.Role) != "" {
		parseValue.Set("role", parseIdentity.Role)
	}
	if strings.TrimSpace(parseIdentity.Version) != "" {
		parseValue.Set("version", parseIdentity.Version)
	}
	return parseValue
}

func clientCapabilitiesJS(parseCapabilities ClientCapabilities) js.Value {
	parseValue := js.Global().Get("Object").New()
	if strings.TrimSpace(parseCapabilities.ProtocolVersion) != "" {
		parseValue.Set("protocolVersion", parseCapabilities.ProtocolVersion)
	}
	if len(parseCapabilities.Transports) > 0 {
		parseValue.Set("transports", stringArrayValue(parseCapabilities.Transports))
	}
	if len(parseCapabilities.Encodings) > 0 {
		parseValue.Set("encodings", stringArrayValue(parseCapabilities.Encodings))
	}
	if len(parseCapabilities.Topics) > 0 {
		parseValue.Set("topics", stringArrayValue(parseCapabilities.Topics))
	}
	if parseCapabilities.MaxJSONBytes > 0 {
		parseValue.Set("maxJsonBytes", parseCapabilities.MaxJSONBytes)
	}
	if parseCapabilities.MaxBinaryBytes > 0 {
		parseValue.Set("maxBinaryBytes", parseCapabilities.MaxBinaryBytes)
	}
	return parseValue
}

func jsValueSummary(parseValue js.Value) string {
	if parseValue.IsUndefined() {
		return "undefined"
	}
	if parseValue.IsNull() {
		return "null"
	}
	switch parseValue.Type() {
	case js.TypeString:
		return parseValue.String()
	case js.TypeBoolean:
		if parseValue.Bool() {
			return "true"
		}
		return "false"
	case js.TypeNumber:
		return fmt.Sprint(parseValue.Float())
	default:
		parseStringified := js.Global().Get("JSON").Call("stringify", parseValue)
		if parseStringified.IsUndefined() || parseStringified.IsNull() {
			return parseValue.String()
		}
		return parseStringified.String()
	}
}

// looksLikeJSTypeDescriptor returns true if a string looks like a Go syscall/js
// type descriptor (e.g. "<object>", "<undefined>", "<null>", "<function>") rather
// than a genuine value. Go 1.26+ returns these when js.Value.String() is called
// on a non-string JS value.
func looksLikeJSTypeDescriptor(parseS string) bool {
	parseTrimmed := strings.TrimSpace(parseS)
	switch parseTrimmed {
	case "<object>", "<undefined>", "<null>", "<function>", "<symbol>", "<number>", "<boolean>":
		return true
	}
	return false
}

// consoleError logs an error message to the browser console (console.error).
func consoleError(parseMsg string) {
	parseC := js.Global().Get("console")
	if parseC.IsUndefined() || parseC.IsNull() {
		return
	}
	parseC.Call("error", parseMsg)
}
