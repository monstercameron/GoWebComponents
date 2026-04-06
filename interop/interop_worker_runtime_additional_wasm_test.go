//go:build js && wasm
// +build js,wasm

package interop

import (
	"context"
	"errors"
	"syscall/js"
	"testing"
)

// TestGoWASMWorkerStateWasmDelegatesActiveWorkerOperations verifies the active go-worker state forwards every worker surface to the wrapped worker.
func TestGoWASMWorkerStateWasmDelegatesActiveWorkerOperations(parseT *testing.T) {
	parseProgressCount := 0
	parseHandledCount := 0
	parseCancelCount := 0
	parseCurrentPostCount := 0
	parseState := &goWASMWorkerState{
		options: GoWASMWorkerOptions{WASMURL: "https://app.example.test/worker.wasm"},
		active:  true,
	}
	parseState.worker = Worker{
		post: func(parseMessage any) error {
			switch parseMessage {
			case "payload":
				parseHandledCount++
			case "current":
				parseCurrentPostCount++
			default:
				parseT.Fatalf("unexpected post payload: %#v", parseMessage)
			}
			return nil
		},
		postPorts: func(parseMessage any, parsePorts ...MessagePort) error {
			if parseMessage != "ports" || len(parsePorts) != 1 {
				parseT.Fatalf("unexpected postPorts call: message=%#v ports=%d", parseMessage, len(parsePorts))
			}
			parseHandledCount++
			return nil
		},
		subscribe: func(parseHandler func(WorkerMessage, error)) (Subscription, error) {
			parseHandledCount++
			parseHandler(WorkerMessage{Name: "subscribed"}, nil)
			return Subscription{cancel: func() {
				parseCancelCount++
			}}, nil
		},
		request: func(parseCtx context.Context, parseName string, parsePayload any, parseOnProgress func(WorkerMessage, error)) (WorkerMessage, error) {
			if parseCtx == nil || parseName != "demo" || parsePayload != "request-payload" {
				parseT.Fatalf("unexpected request call: ctx=%v name=%q payload=%#v", parseCtx, parseName, parsePayload)
			}
			parseHandledCount++
			if parseOnProgress != nil {
				parseOnProgress(WorkerMessage{Phase: "progress"}, nil)
			}
			return WorkerMessage{Name: parseName, Payload: parsePayload}, nil
		},
	}

	parseCurrentWorker, parseErr := parseState.current("Worker.Post", parseState.options.WASMURL)
	if parseErr != nil {
		parseT.Fatalf("expected active current worker, got %v", parseErr)
	}
	if parseErr2 := parseCurrentWorker.Post("current"); parseErr2 != nil {
		parseT.Fatalf("expected current worker post to succeed, got %v", parseErr2)
	}
	if parseCurrentPostCount != 1 {
		parseT.Fatalf("expected current worker post to run once, got %d", parseCurrentPostCount)
	}

	if parseErr3 := parseState.post("payload"); parseErr3 != nil {
		parseT.Fatalf("expected go worker post to succeed, got %v", parseErr3)
	}
	if parseErr4 := parseState.postPorts("ports", MessagePort{}); parseErr4 != nil {
		parseT.Fatalf("expected go worker postPorts to succeed, got %v", parseErr4)
	}

	parseSubscription, parseErr5 := parseState.subscribe(func(parseMessage WorkerMessage, parseErr6 error) {
		if parseErr6 != nil || parseMessage.Name != "subscribed" {
			parseT.Fatalf("unexpected subscribe callback: message=%#v err=%v", parseMessage, parseErr6)
		}
	})
	if parseErr5 != nil {
		parseT.Fatalf("expected go worker subscribe to succeed, got %v", parseErr5)
	}
	parseSubscription.Cancel()
	if parseCancelCount != 1 {
		parseT.Fatalf("expected subscription cancel to run once, got %d", parseCancelCount)
	}

	parseResult, parseErr7 := parseState.request(context.Background(), "demo", "request-payload", func(parseMessage WorkerMessage, parseErr8 error) {
		if parseErr8 != nil || parseMessage.Phase != "progress" {
			parseT.Fatalf("unexpected request progress callback: message=%#v err=%v", parseMessage, parseErr8)
		}
		parseProgressCount++
	})
	if parseErr7 != nil {
		parseT.Fatalf("expected go worker request to succeed, got %v", parseErr7)
	}
	if parseResult.Name != "demo" || parseResult.Payload != "request-payload" {
		parseT.Fatalf("unexpected go worker request result: %#v", parseResult)
	}
	if parseHandledCount != 4 || parseProgressCount != 1 {
		parseT.Fatalf("expected all worker delegates to run, got handled=%d progress=%d", parseHandledCount, parseProgressCount)
	}
}

// TestGoWASMWorkerStateWasmRejectsInactiveOperations verifies the disposed go-worker state consistently reports CodeDisposed.
func TestGoWASMWorkerStateWasmRejectsInactiveOperations(parseT *testing.T) {
	parseState := &goWASMWorkerState{
		options: GoWASMWorkerOptions{WASMURL: "https://app.example.test/worker.wasm"},
		active:  false,
	}

	if _, parseErr := parseState.current("Worker.Post", parseState.options.WASMURL); !IsCode(parseErr, CodeDisposed) {
		parseT.Fatalf("expected disposed current error, got %v", parseErr)
	}
	if parseErr2 := parseState.post("payload"); !IsCode(parseErr2, CodeDisposed) {
		parseT.Fatalf("expected disposed post error, got %v", parseErr2)
	}
	if parseErr3 := parseState.postPorts("payload"); !IsCode(parseErr3, CodeDisposed) {
		parseT.Fatalf("expected disposed postPorts error, got %v", parseErr3)
	}
	if _, parseErr4 := parseState.subscribe(func(WorkerMessage, error) {}); !IsCode(parseErr4, CodeDisposed) {
		parseT.Fatalf("expected disposed subscribe error, got %v", parseErr4)
	}
	if _, parseErr5 := parseState.request(context.Background(), "demo", nil, nil); !IsCode(parseErr5, CodeDisposed) {
		parseT.Fatalf("expected disposed request error, got %v", parseErr5)
	}
}

// TestWorkerRuntimeWasmHelpersCoverErrorFallbacks verifies the worker-runtime helper fallbacks stay stable under missing browser globals and empty envelopes.
func TestWorkerRuntimeWasmHelpersCoverErrorFallbacks(parseT *testing.T) {
	parseURL := js.Global().Get("Object").New()
	parseRevokedURL := ""
	parseRevokeObjectURL := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			parseRevokedURL = parseArgs[0].String()
		}
		return nil
	})
	defer parseRevokeObjectURL.Release()
	parseURL.Set("revokeObjectURL", parseRevokeObjectURL)
	parseRestoreURL := setGlobalValue("URL", parseURL)
	defer parseRestoreURL()

	revokeObjectURL("  blob:gwc-runtime  ")
	if parseRevokedURL != "blob:gwc-runtime" {
		parseT.Fatalf("expected trimmed revokeObjectURL input, got %q", parseRevokedURL)
	}
	revokeObjectURL(" ")
	if parseRevokedURL != "blob:gwc-runtime" {
		parseT.Fatalf("expected blank revokeObjectURL input to be ignored, got %q", parseRevokedURL)
	}

	parseRestoreMissingURL := setGlobalValue("URL", js.Undefined())
	revokeObjectURL("blob:skip")
	parseRestoreMissingURL()

	if parseGot := workerRemoteEnvelopeError(WorkerMessage{Error: "worker failed"}); parseGot != "worker failed" {
		parseT.Fatalf("workerRemoteEnvelopeError(error) = %q, want worker failed", parseGot)
	}
	if parseGot2 := workerRemoteEnvelopeError(WorkerMessage{Payload: map[string]any{"status": "bad"}}); parseGot2 != "map[status:bad]" {
		parseT.Fatalf("workerRemoteEnvelopeError(payload) = %q, want payload summary", parseGot2)
	}
	if parseGot3 := workerRemoteEnvelopeError(WorkerMessage{}); parseGot3 != "worker reported an error" {
		parseT.Fatalf("workerRemoteEnvelopeError(empty) = %q, want generic fallback", parseGot3)
	}

	if !IsCode(workerContextError("Worker.Request", "demo", context.DeadlineExceeded), CodeTimeout) {
		parseT.Fatal("expected deadline exceeded to map to CodeTimeout")
	}
	if !IsCode(workerContextError("Worker.Request", "demo", context.Canceled), CodeCancelled) {
		parseT.Fatal("expected cancellation to map to CodeCancelled")
	}
	if !errors.Is(workerContextError("Worker.Request", "demo", context.Canceled), context.Canceled) {
		parseT.Fatal("expected workerContextError to preserve the original cancellation")
	}
}
