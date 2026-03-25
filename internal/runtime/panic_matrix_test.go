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

func hasFatalPanicCode(parseDiagnostics []Diagnostic, parseLogs []LogEntry) bool {
	for _, parseDiagnostic := range parseDiagnostics {
		if strings.HasPrefix(strings.TrimSpace(parseDiagnostic.Code), "GWC-RUNTIME-PANIC-") {
			return true
		}
	}
	for _, parseEntry := range parseLogs {
		if strings.HasPrefix(strings.TrimSpace(parseEntry.Code), "GWC-RUNTIME-PANIC-") {
			return true
		}
	}
	return false
}

func TestRecoveredRenderPanicDoesNotEmitFatalWrappedDiagnostics(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	parseBoom := func() *Element {
		panic("render boom")
	}

	parseRt.Render(CreateElement(parseBoundary, map[string]interface{}{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("p", nil, "render fallback")
		},
	}, CreateElement(parseBoom, nil)), parseContainer)
	flushScheduledWork(parseScheduler)

	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		parseT.Fatalf("expected recovered render panic not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestRecoveredEventPanicDoesNotEmitFatalWrappedDiagnostics(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseRt := GetGlobalRuntime()
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	shouldPanic := true
	parseEventComp := func() *Element {
		parseHandler := GoUseFunc(func() {
			if shouldPanic {
				panic(errors.New("event boom"))
			}
		})
		return CreateElement("button", map[string]interface{}{"onclick": parseHandler}, "click")
	}

	parseRootElement := CreateElement(parseBoundary, map[string]interface{}{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("button", map[string]interface{}{
				"onclick": func() {
					shouldPanic = false
					reset()
				},
			}, "reset")
		},
	}, CreateElement(parseEventComp, nil))

	parseRt.Render(parseRootElement, parseContainer)
	flushScheduledWork(parseScheduler)
	parseHandler2 := parseContainer.(*testDOMNode).children[0].(*testDOMNode).properties["onclick"].(func())
	parseHandler2()
	flushScheduledWork(parseScheduler)

	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		parseT.Fatalf("expected recovered event panic not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestSuccessfulRenderDoesNotEmitFatalWrappedDiagnostics(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	parseRt.Render(CreateElement("div", nil, "ok"), parseContainer)
	flushScheduledWork(parseScheduler)

	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		parseT.Fatalf("expected successful render not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestReportUnhandledPanicContextFormatsLoaderPhase(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseMessage := ReportUnhandledPanicContext("router", PanicPhaseLoader, "route loader", "/users", nil, "loader boom")
	if !strings.Contains(parseMessage, "GWC-RUNTIME-PANIC-LOADER") ||
		!strings.Contains(parseMessage, "path: /users") ||
		!strings.Contains(parseMessage, "error: loader boom") ||
		!strings.Contains(parseMessage, "runtime: no route boundary recovered this loader panic") ||
		!strings.Contains(parseMessage, "next: Match code GWC-RUNTIME-PANIC-LOADER in automation;") ||
		!strings.Contains(parseMessage, "inspect the loader or async data path named by where/path first") ||
		!strings.Contains(parseMessage, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-loader") {
		parseT.Fatalf("expected loader panic contract, got %q", parseMessage)
	}

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 || parseDiagnostics[len(parseDiagnostics)-1].Code != "GWC-RUNTIME-PANIC-LOADER" {
		parseT.Fatalf("expected loader panic diagnostic, got %+v", parseDiagnostics)
	}
}

func TestReportUnhandledPanicContextFormatsErrorPayload(parseT *testing.T) {
	parseMessage := ReportUnhandledPanicContext("runtime", PanicPhaseEvent, "button handler", "App > Button", []string{"App", "Button"}, errors.New("event failed"))
	if !strings.Contains(parseMessage, "error: event failed") || !strings.Contains(parseMessage, "path: App > Button") || !strings.Contains(parseMessage, "inspect the event handler named by where/path first") {
		parseT.Fatalf("expected error payload to round-trip, got %q", parseMessage)
	}
}

func TestMarkUnhandledPanicWrapsOnceAndPreservesOriginalPanic(parseT *testing.T) {
	parseMarked := markUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "App > Widget", []string{"App", "Widget"}, "render boom")
	parseOriginal, parseOk := unwrapReportedPanic(parseMarked)
	if !parseOk || parseOriginal != "render boom" {
		parseT.Fatalf("expected marked panic to preserve original payload, got %#v", parseMarked)
	}
	parseRemarked := markUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "", nil, parseMarked)
	parseOriginal, parseOk = unwrapReportedPanic(parseRemarked)
	if !parseOk || parseOriginal != "render boom" {
		parseT.Fatalf("expected remarked panic to avoid rewrapping, got %#v", parseRemarked)
	}
}

func TestMarkUnhandledPanicPreservesUncomparablePayload(parseT *testing.T) {
	parsePayload := map[string]string{"kind": "js.Error", "message": "audio constructor failed"}
	parseMarked := markUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "App > Audio", []string{"App", "Audio"}, parsePayload)
	parseOriginal, parseOk := unwrapReportedPanic(parseMarked)
	if !parseOk {
		parseT.Fatalf("expected marked panic to unwrap uncomparable payload, got %#v", parseMarked)
	}
	parseDecoded, parseOk := parseOriginal.(map[string]string)
	if !parseOk || parseDecoded["kind"] != "js.Error" || parseDecoded["message"] != "audio constructor failed" {
		parseT.Fatalf("expected original map payload to round-trip, got %#v", parseOriginal)
	}

	parseRemarked := markUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "App > Audio", []string{"App", "Audio"}, parseMarked)
	parseOriginal, parseOk = unwrapReportedPanic(parseRemarked)
	if !parseOk {
		parseT.Fatalf("expected remarked panic to unwrap uncomparable payload, got %#v", parseRemarked)
	}
	parseDecoded, parseOk = parseOriginal.(map[string]string)
	if !parseOk || parseDecoded["kind"] != "js.Error" || parseDecoded["message"] != "audio constructor failed" {
		parseT.Fatalf("expected remarked panic to preserve original map payload, got %#v", parseOriginal)
	}
}

func TestReportUnhandledPanicContextFormatsStructPayload(parseT *testing.T) {
	parseMessage := ReportUnhandledPanicContext("runtime", PanicPhaseDeferred, "task queue", "TaskQueue", nil, panicPayload{Kind: "deferred", ID: 7})
	if !strings.Contains(parseMessage, "error: {deferred 7}") {
		parseT.Fatalf("expected struct payload to render via fmt, got %q", parseMessage)
	}
}

func TestReportUnhandledPanicContextFormatsEmptyStringPayload(parseT *testing.T) {
	parseMessage := ReportUnhandledPanicContext("runtime", PanicPhaseStartup, "RenderTo", "#app", nil, "")
	if !strings.HasPrefix(parseMessage, "panic without message\n") || !strings.Contains(parseMessage, "error: panic without message") {
		parseT.Fatalf("expected empty panic payload fallback, got %q", parseMessage)
	}
}

func TestPanicPhaseMayRecoverWithBoundaryPolicy(parseT *testing.T) {
	if !panicPhaseMayRecoverWithBoundary(PanicPhaseRender) ||
		!panicPhaseMayRecoverWithBoundary(PanicPhaseEvent) ||
		!panicPhaseMayRecoverWithBoundary(PanicPhaseEffect) ||
		!panicPhaseMayRecoverWithBoundary(PanicPhaseCleanup) {
		parseT.Fatal("expected render/event/effect/cleanup to be boundary-recoverable phases")
	}
	if panicPhaseMayRecoverWithBoundary(PanicPhaseLoader) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseHydration) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseStartup) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseDeferred) ||
		panicPhaseMayRecoverWithBoundary(PanicPhaseSSR) {
		parseT.Fatal("expected loader/hydration/startup/deferred/ssr to remain fatal rethrow phases")
	}
}
