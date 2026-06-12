package runtime

import (
	"errors"
	"strings"
	"testing"
	"time"
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

	parseRt.Render(CreateElement(parseBoundary, map[string]any{
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
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, Reset: true})
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
		return CreateElement("button", map[string]any{"onclick": parseHandler}, "click")
	}

	parseRootElement := CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("button", map[string]any{
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

func TestEffectPanicMatrixCommitsThenBoundaryRecovers(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	SetCurrentFiber(nil)
	parseT.Cleanup(func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	})
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	hasPanicked := false
	parseEffectComp := func() *Element {
		GoUseEffect(func() func() {
			if !hasPanicked {
				hasPanicked = true
				panic("effect boom")
			}
			return nil
		})
		return CreateElement("span", nil, "effect ready")
	}

	parseRt.Render(CreateElement(parseBoundary, map[string]any{
		"errorFallback": func(parseErr error, reset func()) *Element {
			return CreateElement("p", nil, "effect fallback")
		},
	}, CreateElement(parseEffectComp, nil)), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 10)

	if parseGot := nodeTextContent(parseContainer); parseGot != "effect fallback" {
		parseT.Fatalf("expected effect boundary fallback after recovery update, got %q", parseGot)
	}
	if parseRt.currentRoot == nil || parseRt.wipRoot != nil || parseRt.updateScheduled {
		parseT.Fatalf("expected settled committed tree after effect recovery, current=%p wip=%p scheduled=%t", parseRt.currentRoot, parseRt.wipRoot, parseRt.updateScheduled)
	}
	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		parseT.Fatalf("expected boundary-recovered effect panic not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestCleanupPanicMatrixRecoversAfterDeletion(parseT *testing.T) {
	resetGlobalRuntimeForTest()
	SetCurrentFiber(nil)
	parseT.Cleanup(func() {
		SetCurrentFiber(nil)
		resetGlobalRuntimeForTest()
	})
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	shouldRenderChild := true
	parseChild := func() *Element {
		GoUseEffect(func() func() {
			return func() { panic("cleanup boom") }
		})
		return CreateElement("span", nil, "cleanup child")
	}
	parseRoot := func() *Element {
		parseContent := CreateElement("span", nil, "after cleanup")
		if shouldRenderChild {
			parseContent = CreateElement(parseChild, nil)
		}
		return CreateElement(parseBoundary, map[string]any{
			"errorFallback": func(parseErr error, reset func()) *Element {
				return CreateElement("p", nil, "cleanup fallback")
			},
		}, parseContent)
	}

	parseRt.Render(parseRoot(), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 10)
	if parseGot := nodeTextContent(parseContainer); parseGot != "cleanup child" {
		parseT.Fatalf("expected initial child text, got %q", parseGot)
	}

	shouldRenderChild = false
	parseRt.Render(parseRoot(), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 10)

	if parseGot := nodeTextContent(parseContainer); parseGot != "cleanup fallback" {
		parseT.Fatalf("expected cleanup boundary fallback after recovery update, got %q", parseGot)
	}
	if parseRt.currentRoot == nil || parseRt.wipRoot != nil || parseRt.updateScheduled {
		parseT.Fatalf("expected settled committed tree after cleanup recovery, current=%p wip=%p scheduled=%t", parseRt.currentRoot, parseRt.wipRoot, parseRt.updateScheduled)
	}
	if hasFatalPanicCode(GetDiagnostics(), GetLogs()) {
		parseT.Fatalf("expected boundary-recovered cleanup panic not to emit fatal panic metadata, diagnostics=%+v logs=%+v", GetDiagnostics(), GetLogs())
	}
}

func TestAsyncPanicMatrixReportsAndPreservesCommittedUI(parseT *testing.T) {
	SetCurrentFiber(nil)
	parseT.Cleanup(func() {
		SetCurrentFiber(nil)
	})
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseGetReports := capturePanicReports(parseT)
	parseContainer := parseAdapter.CreateElement("div")

	parseRt.Render(CreateElement("div", nil, "async alive"), parseContainer)
	drainScheduledTimeouts(parseT, parseScheduler, 10)

	parseDone := make(chan struct{})
	SafeGo("test", "async matrix task", func() {
		defer close(parseDone)
		panic("async boom")
	})
	select {
	case <-parseDone:
	case <-time.After(5 * time.Second):
		parseT.Fatal("async matrix task did not finish")
	}

	parseDeadline := time.Now().Add(5 * time.Second)
	for len(parseGetReports()) == 0 && time.Now().Before(parseDeadline) {
		time.Sleep(time.Millisecond)
	}
	parseReports := parseGetReports()
	if len(parseReports) != 1 || parseReports[0].Phase != PanicPhaseAsync {
		parseT.Fatalf("expected one async panic report, got %+v", parseReports)
	}
	if parseGot := nodeTextContent(parseContainer); parseGot != "async alive" {
		parseT.Fatalf("expected async containment to preserve committed UI, got %q", parseGot)
	}
	if parseRt.currentRoot == nil || parseRt.wipRoot != nil || parseRt.updateScheduled {
		parseT.Fatalf("expected async containment not to leave render work pending, current=%p wip=%p scheduled=%t", parseRt.currentRoot, parseRt.wipRoot, parseRt.updateScheduled)
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
