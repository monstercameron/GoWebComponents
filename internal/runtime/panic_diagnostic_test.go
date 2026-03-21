package runtime

import (
	"strings"
	"testing"
)

func panicDiagnosticRenderComponent() *Element {
	panic("render boom")
}

func panicDiagnosticOwnerComponent() *Element {
	return CreateElement("button", nil, "owner")
}

func panicDiagnosticEffectComponent() *Element {
	GoUseEffect(func() func() {
		panic("effect boom")
	})
	return CreateElement("span", nil, "effect")
}

func panicDiagnosticCleanupComponent() *Element {
	GoUseEffect(func() func() {
		return func() {
			panic("cleanup boom")
		}
	})
	return CreateElement("span", nil, "cleanup")
}

func recoverPanicString(t *testing.T, fn func()) string {
	t.Helper()
	recovered := ""
	func() {
		defer func() {
			if value := recover(); value != nil {
				message, ok := value.(string)
				if !ok {
					t.Fatalf("expected string panic, got %T (%v)", value, value)
				}
				recovered = message
			}
		}()
		fn()
	}()
	if recovered == "" {
		t.Fatal("expected panic")
	}
	return recovered
}

func assertUnhandledPanicRecorded(t *testing.T, code string, subject string, phase boundaryPhase, summary string) {
	t.Helper()

	diagnostics := GetDiagnostics()
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostic")
	}
	lastDiagnostic := diagnostics[len(diagnostics)-1]
	if lastDiagnostic.Code != code {
		t.Fatalf("expected diagnostic code %q, got %+v", code, lastDiagnostic)
	}
	if lastDiagnostic.Docs == "" || lastDiagnostic.Remediation == "" || lastDiagnostic.Recoverable {
		t.Fatalf("expected actionable non-recoverable diagnostic metadata, got %+v", lastDiagnostic)
	}
	if !strings.Contains(lastDiagnostic.Message, string(phase)) || !strings.Contains(lastDiagnostic.Message, summary) {
		t.Fatalf("expected phase and summary in diagnostic, got %+v", lastDiagnostic)
	}
	if !strings.Contains(lastDiagnostic.Path, subject) {
		t.Fatalf("expected diagnostic path to mention %q, got %+v", subject, lastDiagnostic)
	}

	logs := GetLogs()
	if len(logs) == 0 {
		t.Fatal("expected log entry")
	}
	lastLog := logs[len(logs)-1]
	if lastLog.Code != code {
		t.Fatalf("expected log code %q, got %+v", code, lastLog)
	}
	if lastLog.Docs == "" || lastLog.Remediation == "" || lastLog.Recoverable {
		t.Fatalf("expected actionable non-recoverable log metadata, got %+v", lastLog)
	}
	if !strings.Contains(lastLog.Message, string(phase)) || !strings.Contains(lastLog.Message, summary) {
		t.Fatalf("expected phase and summary in log message, got %+v", lastLog)
	}
	if !strings.Contains(lastLog.Fields["path"], subject) {
		t.Fatalf("expected log path to mention %q, got %+v", subject, lastLog)
	}
}

func TestUnhandledRenderPanicReportsActionableDiagnostic(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	rt := &Runtime{}
	root := &Fiber{typeOf: "ROOT"}
	fiber := &Fiber{typeOf: panicDiagnosticRenderComponent, parent: root}

	recovered := recoverPanicString(t, func() {
		_, _, _ = rt.renderFunctionComponent(fiber)
	})

	if !strings.Contains(recovered, "GWC-RUNTIME-PANIC-RENDER") ||
		!strings.Contains(recovered, "uncaught render panic in panicDiagnosticRenderComponent") ||
		!strings.HasPrefix(recovered, "render boom\n") ||
		!strings.Contains(recovered, "where: panicDiagnosticRenderComponent") ||
		!strings.Contains(recovered, "error: render boom") ||
		!strings.Contains(recovered, "render boom") ||
		!strings.Contains(recovered, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-render") {
		t.Fatalf("expected actionable render panic message, got %q", recovered)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-RENDER", "panicDiagnosticRenderComponent", boundaryPhaseRender, "render boom")
}

func TestUnhandledEventPanicReportsActionableDiagnostic(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	rt := &Runtime{}
	root := &Fiber{typeOf: "ROOT"}
	owner := &Fiber{typeOf: panicDiagnosticOwnerComponent, parent: root}
	handler, ok := rt.wrapEventHandler(owner, func() { panic("event boom") }).(func())
	if !ok {
		t.Fatal("expected wrapped event handler")
	}

	recovered := recoverPanicString(t, handler)

	if !strings.Contains(recovered, "GWC-RUNTIME-PANIC-EVENT") ||
		!strings.Contains(recovered, "uncaught event panic in panicDiagnosticOwnerComponent") ||
		!strings.HasPrefix(recovered, "event boom\n") ||
		!strings.Contains(recovered, "where: panicDiagnosticOwnerComponent") ||
		!strings.Contains(recovered, "error: event boom") ||
		!strings.Contains(recovered, "event boom") ||
		!strings.Contains(recovered, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-event") {
		t.Fatalf("expected actionable event panic message, got %q", recovered)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-EVENT", "panicDiagnosticOwnerComponent", boundaryPhaseEvent, "event boom")
}

func TestUnhandledEffectPanicReportsActionableDiagnostic(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	recovered := recoverPanicString(t, func() {
		rt.Render(CreateElement(panicDiagnosticEffectComponent, nil), container)
		flushScheduledWork(scheduler)
	})

	if !strings.Contains(recovered, "GWC-RUNTIME-PANIC-EFFECT") ||
		!strings.Contains(recovered, "uncaught effect panic in panicDiagnosticEffectComponent") ||
		!strings.HasPrefix(recovered, "effect boom\n") ||
		!strings.Contains(recovered, "where: panicDiagnosticEffectComponent") ||
		!strings.Contains(recovered, "error: effect boom") ||
		!strings.Contains(recovered, "effect boom") ||
		!strings.Contains(recovered, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-effect") {
		t.Fatalf("expected actionable effect panic message, got %q", recovered)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-EFFECT", "panicDiagnosticEffectComponent", boundaryPhaseEffect, "effect boom")
}

func TestUnhandledCleanupPanicReportsActionableDiagnostic(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")

	rt.Render(CreateElement(panicDiagnosticCleanupComponent, nil), container)
	flushScheduledWork(scheduler)

	recovered := recoverPanicString(t, func() {
		rt.Render(CreateElement("div", nil, "replacement"), container)
		flushScheduledWork(scheduler)
	})

	if !strings.Contains(recovered, "GWC-RUNTIME-PANIC-CLEANUP") ||
		!strings.Contains(recovered, "uncaught cleanup panic in panicDiagnosticCleanupComponent") ||
		!strings.HasPrefix(recovered, "cleanup boom\n") ||
		!strings.Contains(recovered, "where: panicDiagnosticCleanupComponent") ||
		!strings.Contains(recovered, "error: cleanup boom") ||
		!strings.Contains(recovered, "cleanup boom") ||
		!strings.Contains(recovered, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-cleanup") {
		t.Fatalf("expected actionable cleanup panic message, got %q", recovered)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-CLEANUP", "panicDiagnosticCleanupComponent", boundaryPhaseCleanup, "cleanup boom")
}
