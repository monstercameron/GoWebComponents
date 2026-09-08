package renderworker

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

type parseDispatcherTestRequest struct {
	GetValue string `json:"value"`
}

type parseDispatcherTestResult struct {
	GetValue string `json:"value"`
}

// TestBuildRenderWorkerDispatcherStoresDefaultRequestName validates that the dispatcher keeps a trimmed default request name.
func TestBuildRenderWorkerDispatcherStoresDefaultRequestName(parseT *testing.T) {
	parseDispatcher := BuildRenderWorkerDispatcher("  demo-request  ")
	parseResolved := parseDispatcher.buildRenderWorkerRequestName("")
	if parseResolved != "demo-request" {
		parseT.Fatalf("expected default request name %q, got %q", "demo-request", parseResolved)
	}
}

// TestBuildRenderWorkerDispatcherFallsBackToGenericRequestName validates fallback semantics when no request names are provided.
func TestBuildRenderWorkerDispatcherFallsBackToGenericRequestName(parseT *testing.T) {
	parseDispatcher := BuildRenderWorkerDispatcher(" ")
	parseResolved := parseDispatcher.buildRenderWorkerRequestName("")
	if parseResolved != getRenderWorkerFallbackRequestName {
		parseT.Fatalf("expected fallback request name %q, got %q", getRenderWorkerFallbackRequestName, parseResolved)
	}
}

// TestSetRenderWorkerHandlerRejectsInvalidInputs validates nil dispatcher, empty request name, and nil handler guards.
func TestSetRenderWorkerHandlerRejectsInvalidInputs(parseT *testing.T) {
	parseDispatcher := BuildRenderWorkerDispatcher("")
	if parseErr := (*RenderWorkerDispatcher)(nil).SetRenderWorkerHandler("demo", func(interop.WorkerScope, interop.WorkerMessage) {}); parseErr == nil {
		parseT.Fatalf("expected nil dispatcher registration to fail")
	}
	if parseErr := parseDispatcher.SetRenderWorkerHandler(" ", func(interop.WorkerScope, interop.WorkerMessage) {}); parseErr == nil {
		parseT.Fatalf("expected empty request name registration to fail")
	}
	if parseErr := parseDispatcher.SetRenderWorkerHandler("demo", nil); parseErr == nil {
		parseT.Fatalf("expected nil handler registration to fail")
	}
}

// TestSetRenderWorkerHandlerRejectsDuplicateNames validates duplicate request-name registration rejection.
func TestSetRenderWorkerHandlerRejectsDuplicateNames(parseT *testing.T) {
	parseDispatcher := BuildRenderWorkerDispatcher("")
	parseHandler := func(interop.WorkerScope, interop.WorkerMessage) {}
	if parseErr := parseDispatcher.SetRenderWorkerHandler("demo", parseHandler); parseErr != nil {
		parseT.Fatalf("unexpected initial handler registration error: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetRenderWorkerHandler("demo", parseHandler); parseErr == nil {
		parseT.Fatalf("expected duplicate handler registration to fail")
	}
}

// TestHandleRenderWorkerMessageInvokesRegisteredHandler validates that request messages route to registered handlers.
func TestHandleRenderWorkerMessageInvokesRegisteredHandler(parseT *testing.T) {
	parseDispatcher := BuildRenderWorkerDispatcher("")
	parseCalled := false
	if parseErr := parseDispatcher.SetRenderWorkerHandler("demo", func(parseScope interop.WorkerScope, parseMessage interop.WorkerMessage) {
		parseCalled = true
		if parseMessage.ID != "job-1" {
			parseT.Fatalf("expected message id %q, got %q", "job-1", parseMessage.ID)
		}
	}); parseErr != nil {
		parseT.Fatalf("unexpected registration error: %v", parseErr)
	}
	parseDispatcher.HandleRenderWorkerMessage(interop.WorkerScope{}, interop.WorkerMessage{
		ID:    "job-1",
		Phase: "request",
		Name:  "demo",
	})
	if !parseCalled {
		parseT.Fatalf("expected registered handler to be called")
	}
}

// TestHandleRenderWorkerMessageIgnoresNonRequestPhase validates that non-request messages are ignored.
func TestHandleRenderWorkerMessageIgnoresNonRequestPhase(parseT *testing.T) {
	parseDispatcher := BuildRenderWorkerDispatcher("")
	parseCalled := false
	if parseErr := parseDispatcher.SetRenderWorkerHandler("demo", func(interop.WorkerScope, interop.WorkerMessage) {
		parseCalled = true
	}); parseErr != nil {
		parseT.Fatalf("unexpected registration error: %v", parseErr)
	}
	parseDispatcher.HandleRenderWorkerMessage(interop.WorkerScope{}, interop.WorkerMessage{
		ID:    "job-1",
		Phase: "message",
		Name:  "demo",
	})
	if parseCalled {
		parseT.Fatalf("expected handler not to be called for non-request phase")
	}
}

// TestBuildRenderWorkerDecodedHandlerCallsTypedHandler validates typed decode and handler invocation on valid payloads.
func TestBuildRenderWorkerDecodedHandlerCallsTypedHandler(parseT *testing.T) {
	parseCalled := false
	parseHandler := BuildRenderWorkerDecodedHandler("demo", func(parseCtx context.Context, parseRequest parseDispatcherTestRequest) (parseDispatcherTestResult, error) {
		_ = parseCtx
		parseCalled = true
		return parseDispatcherTestResult{GetValue: parseRequest.GetValue}, nil
	})
	parseHandler(interop.WorkerScope{}, interop.WorkerMessage{
		ID:      "job-1",
		Phase:   "request",
		Name:    "demo",
		Payload: map[string]any{"value": "ok"},
	})
	if !parseCalled {
		parseT.Fatalf("expected typed handler to be called")
	}
}

// TestBuildRenderWorkerDecodedHandlerSkipsTypedHandlerOnDecodeFailure validates decode-failure short-circuit behavior.
func TestBuildRenderWorkerDecodedHandlerSkipsTypedHandlerOnDecodeFailure(parseT *testing.T) {
	parseCalled := false
	parseHandler := BuildRenderWorkerDecodedHandler("demo", func(parseCtx context.Context, parseRequest parseDispatcherTestRequest) (parseDispatcherTestResult, error) {
		_ = parseCtx
		_ = parseRequest
		parseCalled = true
		return parseDispatcherTestResult{}, nil
	})
	parseHandler(interop.WorkerScope{}, interop.WorkerMessage{
		ID:      "job-1",
		Phase:   "request",
		Name:    "demo",
		Payload: map[string]any{"value": map[string]any{"bad": "shape"}},
	})
	if parseCalled {
		parseT.Fatalf("expected typed handler not to be called after decode failure")
	}
}

// TestBuildRenderWorkerDecodedHandlerAppliesRequestTimeout validates that timeout options apply a context deadline.
func TestBuildRenderWorkerDecodedHandlerAppliesRequestTimeout(parseT *testing.T) {
	parseSawDeadline := false
	parseHandler := BuildRenderWorkerDecodedHandlerWithOptions(
		"demo",
		func(parseCtx context.Context, parseRequest parseDispatcherTestRequest) (parseDispatcherTestResult, error) {
			_ = parseRequest
			_, parseHasDeadline := parseCtx.Deadline()
			parseSawDeadline = parseHasDeadline
			return parseDispatcherTestResult{}, nil
		},
		DecodedHandlerOptions{
			GetRequestTimeout: 50 * time.Millisecond,
		},
	)
	parseHandler(interop.WorkerScope{}, interop.WorkerMessage{
		ID:      "job-1",
		Phase:   "request",
		Name:    "demo",
		Payload: map[string]any{"value": "ok"},
	})
	if !parseSawDeadline {
		parseT.Fatalf("expected request context to have deadline when timeout option is set")
	}
}

// TestBuildRenderWorkerDecodedHandlerRecoversPanicWhenEnabled validates panic recovery when enabled for user-defined handlers.
func TestBuildRenderWorkerDecodedHandlerRecoversPanicWhenEnabled(parseT *testing.T) {
	parseHandler := BuildRenderWorkerDecodedHandlerWithOptions(
		"demo",
		func(parseCtx context.Context, parseRequest parseDispatcherTestRequest) (parseDispatcherTestResult, error) {
			_ = parseCtx
			_ = parseRequest
			panic("boom")
		},
		DecodedHandlerOptions{
			ShouldRecoverPanic: true,
		},
	)
	parseHandler(interop.WorkerScope{}, interop.WorkerMessage{
		ID:      "job-1",
		Phase:   "request",
		Name:    "demo",
		Payload: map[string]any{"value": "ok"},
	})
}

// TestBuildRenderWorkerDecodedHandlerPanicsWhenRecoveryDisabled validates panic passthrough when recovery is disabled.
func TestBuildRenderWorkerDecodedHandlerPanicsWhenRecoveryDisabled(parseT *testing.T) {
	parseHandler := BuildRenderWorkerDecodedHandlerWithOptions(
		"demo",
		func(parseCtx context.Context, parseRequest parseDispatcherTestRequest) (parseDispatcherTestResult, error) {
			_ = parseCtx
			_ = parseRequest
			panic("boom")
		},
		DecodedHandlerOptions{
			ShouldRecoverPanic: false,
		},
	)
	defer func() {
		if parseRecovered := recover(); parseRecovered == nil {
			parseT.Fatalf("expected panic when recovery is disabled")
		}
	}()
	parseHandler(interop.WorkerScope{}, interop.WorkerMessage{
		ID:      "job-1",
		Phase:   "request",
		Name:    "demo",
		Payload: map[string]any{"value": "ok"},
	})
}

// TestSetRenderWorkerDecodedHandlerRejectsNilDispatcher validates nil-dispatcher protection for typed registration.
func TestSetRenderWorkerDecodedHandlerRejectsNilDispatcher(parseT *testing.T) {
	parseErr := SetRenderWorkerDecodedHandler(
		(*RenderWorkerDispatcher)(nil),
		"demo",
		func(parseCtx context.Context, parseRequest parseDispatcherTestRequest) (parseDispatcherTestResult, error) {
			_ = parseCtx
			_ = parseRequest
			return parseDispatcherTestResult{}, nil
		},
		DecodedHandlerOptions{},
	)
	if parseErr == nil {
		parseT.Fatalf("expected nil dispatcher registration to fail")
	}
}

// TestBuildRenderWorkerResultRequestNameSelectsBestRequestName validates result-name resolution priority.
func TestBuildRenderWorkerResultRequestNameSelectsBestRequestName(parseT *testing.T) {
	parseResolvedConfigured := buildRenderWorkerResultRequestName("configured", "message")
	if parseResolvedConfigured != "configured" {
		parseT.Fatalf("expected configured request name to win, got %q", parseResolvedConfigured)
	}
	parseResolvedMessage := buildRenderWorkerResultRequestName("", "message")
	if parseResolvedMessage != "message" {
		parseT.Fatalf("expected message request name fallback, got %q", parseResolvedMessage)
	}
	parseResolvedFallback := buildRenderWorkerResultRequestName("", "")
	if parseResolvedFallback != getRenderWorkerFallbackRequestName {
		parseT.Fatalf("expected generic fallback request name %q, got %q", getRenderWorkerFallbackRequestName, parseResolvedFallback)
	}
}
