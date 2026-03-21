package runtime

import (
	"errors"
	"strings"
	"testing"
)

type panicPayload struct {
	Kind string
	ID   int
}

func hasFatalPanicCode(diagnostics []Diagnostic, logs []LogEntry) bool {
	for _, diagnostic := range diagnostics {
		if strings.HasPrefix(strings.TrimSpace(diagnostic.Code), "GWC-RUNTIME-PANIC-") {
			return true
		}
	}
	for _, entry := range logs {
		if strings.HasPrefix(strings.TrimSpace(entry.Code), "GWC-RUNTIME-PANIC-") {
			return true
		}
	}
	return false
}

func TestRecoveredRenderPanicDoesNotEmitFatalWrappedDiagnostics(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	boundary := NewErrorBoundaryType()
	boom := func() *Element {
		panic("render boom")
	}

	rt.Render(CreateElement(boundary, map[string]interface{}{
		"errorFallback": func(err error, reset func()) *Element {
			return CreateElement("p", nil, "render fallback")
		},
	}, CreateElement(boom, nil)), container)
	flushScheduledWork(scheduler)

	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		t.Fatalf("expected recovered render panic not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestRecoveredEventPanicDoesNotEmitFatalWrappedDiagnostics(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	rt := GetGlobalRuntime()
	container := adapter.CreateElement("div")
	boundary := NewErrorBoundaryType()
	shouldPanic := true
	eventComp := func() *Element {
		handler := GoUseFunc(func() {
			if shouldPanic {
				panic(errors.New("event boom"))
			}
		})
		return CreateElement("button", map[string]interface{}{"onclick": handler}, "click")
	}

	rootElement := CreateElement(boundary, map[string]interface{}{
		"errorFallback": func(err error, reset func()) *Element {
			return CreateElement("button", map[string]interface{}{
				"onclick": func() {
					shouldPanic = false
					reset()
				},
			}, "reset")
		},
	}, CreateElement(eventComp, nil))

	rt.Render(rootElement, container)
	flushScheduledWork(scheduler)
	handler := container.(*testDOMNode).children[0].(*testDOMNode).properties["onclick"].(func())
	handler()
	flushScheduledWork(scheduler)

	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		t.Fatalf("expected recovered event panic not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestSuccessfulRenderDoesNotEmitFatalWrappedDiagnostics(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	rt.Render(CreateElement("div", nil, "ok"), container)
	flushScheduledWork(scheduler)

	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		t.Fatalf("expected successful render not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestReportUnhandledPanicContextFormatsLoaderPhase(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	message := ReportUnhandledPanicContext("router", PanicPhaseLoader, "route loader", "/users", nil, "loader boom")
	if !strings.Contains(message, "GWC-RUNTIME-PANIC-LOADER") ||
		!strings.Contains(message, "path: /users") ||
		!strings.Contains(message, "error: loader boom") ||
		!strings.Contains(message, "runtime: no route boundary recovered this loader panic") ||
		!strings.Contains(message, "next: Match code GWC-RUNTIME-PANIC-LOADER in automation;") ||
		!strings.Contains(message, "inspect the loader or async data path named by where/path first") ||
		!strings.Contains(message, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-loader") {
		t.Fatalf("expected loader panic contract, got %q", message)
	}

	diagnostics := GetDiagnostics()
	if len(diagnostics) == 0 || diagnostics[len(diagnostics)-1].Code != "GWC-RUNTIME-PANIC-LOADER" {
		t.Fatalf("expected loader panic diagnostic, got %+v", diagnostics)
	}
}

func TestReportUnhandledPanicContextFormatsErrorPayload(t *testing.T) {
	message := ReportUnhandledPanicContext("runtime", PanicPhaseEvent, "button handler", "App > Button", []string{"App", "Button"}, errors.New("event failed"))
	if !strings.Contains(message, "error: event failed") || !strings.Contains(message, "path: App > Button") || !strings.Contains(message, "inspect the event handler named by where/path first") {
		t.Fatalf("expected error payload to round-trip, got %q", message)
	}
}

func TestMarkUnhandledPanicWrapsOnceAndPreservesOriginalPanic(t *testing.T) {
	marked := markUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "App > Widget", []string{"App", "Widget"}, "render boom")
	original, ok := unwrapReportedPanic(marked)
	if !ok || original != "render boom" {
		t.Fatalf("expected marked panic to preserve original payload, got %#v", marked)
	}
	remarked := markUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "", nil, marked)
	original, ok = unwrapReportedPanic(remarked)
	if !ok || original != "render boom" {
		t.Fatalf("expected remarked panic to avoid rewrapping, got %#v", remarked)
	}
}

func TestReportUnhandledPanicContextFormatsStructPayload(t *testing.T) {
	message := ReportUnhandledPanicContext("runtime", PanicPhaseDeferred, "task queue", "TaskQueue", nil, panicPayload{Kind: "deferred", ID: 7})
	if !strings.Contains(message, "error: {deferred 7}") {
		t.Fatalf("expected struct payload to render via fmt, got %q", message)
	}
}

func TestReportUnhandledPanicContextFormatsEmptyStringPayload(t *testing.T) {
	message := ReportUnhandledPanicContext("runtime", PanicPhaseStartup, "RenderTo", "#app", nil, "")
	if !strings.HasPrefix(message, "panic without message\n") || !strings.Contains(message, "error: panic without message") {
		t.Fatalf("expected empty panic payload fallback, got %q", message)
	}
}

func TestPanicPhaseMayRecoverWithBoundaryPolicy(t *testing.T) {
	if !panicPhaseMayRecoverWithBoundary(PanicPhaseRender) ||
		!panicPhaseMayRecoverWithBoundary(PanicPhaseEvent) ||
		!panicPhaseMayRecoverWithBoundary(PanicPhaseEffect) ||
		!panicPhaseMayRecoverWithBoundary(PanicPhaseCleanup) {
		t.Fatal("expected render/event/effect/cleanup to be boundary-recoverable phases")
	}
	if panicPhaseMayRecoverWithBoundary(PanicPhaseLoader) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseHydration) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseStartup) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseDeferred) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseSSR) {
		t.Fatal("expected loader/hydration/startup/deferred/ssr to remain fatal rethrow phases")
	}
}
