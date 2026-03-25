package runtime

import (
	"errors"
	"strings"
	"testing"
)

func withPanicLoggingOptions(parseT *testing.T, parseOptions PanicLoggingOptions) {
	parseT.Helper()
	parsePrevious := CurrentUnhandledPanicLoggingOptions()
	ConfigureUnhandledPanicLogging(parseOptions)
	parseT.Cleanup(func() {
		ConfigureUnhandledPanicLogging(parsePrevious)
	})
}

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

func panicDiagnosticSSRComponent() *Element {
	panic("ssr boom")
}

func panicSourceMapHelper() panicReportContext {
	return buildActionablePanicContext(ActionablePanicOptions{
		Source:  "runtime",
		Subject: "panicSourceMapHelper",
		Message: "mapped failure",
	})
}

func recoverPanicString(parseT *testing.T, parseFn func()) string {
	parseT.Helper()
	parseRecovered := ""
	func() {
		defer func() {
			if parseValue := recover(); parseValue != nil {
				if parseOriginal, parseOk := unwrapReportedPanic(parseValue); parseOk {
					parseValue = parseOriginal
				}
				parseMessage, parseOk2 := parseValue.(string)
				if !parseOk2 {
					parseT.Fatalf("expected string panic, got %T (%v)", parseValue, parseValue)
				}
				parseRecovered = parseMessage
			}
		}()
		parseFn()
	}()
	if parseRecovered == "" {
		parseT.Fatal("expected panic")
	}
	return parseRecovered
}

func captureStdout(parseT *testing.T, parseFn func()) string {
	parseT.Helper()
	parsePrevious := CurrentUnhandledPanicLoggingOptions()
	parseFormattedReports := make([]string, 0, 1)
	ConfigureUnhandledPanicLogging(PanicLoggingOptions{
		HideRawPanicOutput: parsePrevious.HideRawPanicOutput,
		OnReport: func(parseReport PanicReport) {
			parseFormattedReports = append(parseFormattedReports, parseReport.Formatted)
			if parsePrevious.OnReport != nil {
				parsePrevious.OnReport(parseReport)
			}
		},
	})
	defer ConfigureUnhandledPanicLogging(parsePrevious)
	parseFn()
	return strings.Join(parseFormattedReports, "\n")
}

func assertWrappedOutputQuality(parseT *testing.T, parseOutput string, parseCode string, parseGuidance string, parseExpectedPath string) {
	parseT.Helper()
	if strings.TrimSpace(parseOutput) == "" {
		parseT.Fatal("expected wrapped panic output")
	}
	if !strings.Contains(parseOutput, parseCode) ||
		!strings.Contains(parseOutput, "where: ") ||
		!strings.Contains(parseOutput, "error: ") ||
		!strings.Contains(parseOutput, "runtime: ") ||
		!strings.Contains(parseOutput, "next: ") ||
		!strings.Contains(parseOutput, "docs: ACTIONABLE_ERRORS.md") {
		parseT.Fatalf("expected wrapped panic contract fields, got %q", parseOutput)
	}
	if parseExpectedPath != "" && !strings.Contains(parseOutput, "path: "+parseExpectedPath) {
		parseT.Fatalf("expected path %q in wrapped output, got %q", parseExpectedPath, parseOutput)
	}
	if parseGuidance != "" && !strings.Contains(parseOutput, parseGuidance) {
		parseT.Fatalf("expected phase-specific guidance %q, got %q", parseGuidance, parseOutput)
	}
	if strings.Contains(parseOutput, "(0x") {
		parseT.Fatalf("expected sanitized where/stack formatting without pointer payloads, got %q", parseOutput)
	}
	if strings.Contains(parseOutput, "markUnhandledPanicContext") ||
		strings.Contains(parseOutput, "finalizeUnhandledPanicContext") ||
		strings.Contains(parseOutput, "SetTimeout.func1") {
		parseT.Fatalf("expected compact first-pass stack without low-signal framework frames, got %q", parseOutput)
	}
}

func assertUnhandledPanicRecorded(parseT *testing.T, parseCode string, parseSubject string, parsePhase boundaryPhase, parseSummary string) {
	parseT.Helper()

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		parseT.Fatal("expected diagnostic")
	}
	parseLastDiagnostic := parseDiagnostics[len(parseDiagnostics)-1]
	if parseLastDiagnostic.Code != parseCode {
		parseT.Fatalf("expected diagnostic code %q, got %+v", parseCode, parseLastDiagnostic)
	}
	if parseLastDiagnostic.Docs == "" || parseLastDiagnostic.Remediation == "" || parseLastDiagnostic.Recoverable {
		parseT.Fatalf("expected actionable non-recoverable diagnostic metadata, got %+v", parseLastDiagnostic)
	}
	if strings.TrimSpace(parseLastDiagnostic.TopFrame) == "" || strings.TrimSpace(parseLastDiagnostic.Consequence) == "" {
		parseT.Fatalf("expected panic-specific diagnostic metadata, got %+v", parseLastDiagnostic)
	}
	if !strings.Contains(parseLastDiagnostic.Message, string(parsePhase)) || !strings.Contains(parseLastDiagnostic.Message, parseSummary) {
		parseT.Fatalf("expected phase and summary in diagnostic, got %+v", parseLastDiagnostic)
	}
	if !strings.Contains(parseLastDiagnostic.Path, parseSubject) {
		parseT.Fatalf("expected diagnostic path to mention %q, got %+v", parseSubject, parseLastDiagnostic)
	}

	parseLogs := GetLogs()
	if len(parseLogs) == 0 {
		parseT.Fatal("expected log entry")
	}
	parseLastLog := parseLogs[len(parseLogs)-1]
	if parseLastLog.Code != parseCode {
		parseT.Fatalf("expected log code %q, got %+v", parseCode, parseLastLog)
	}
	if parseLastLog.Docs == "" || parseLastLog.Remediation == "" || parseLastLog.Recoverable {
		parseT.Fatalf("expected actionable non-recoverable log metadata, got %+v", parseLastLog)
	}
	if strings.TrimSpace(parseLastLog.TopFrame) == "" || strings.TrimSpace(parseLastLog.Consequence) == "" {
		parseT.Fatalf("expected panic-specific log metadata, got %+v", parseLastLog)
	}
	if !strings.Contains(parseLastLog.Message, string(parsePhase)) || !strings.Contains(parseLastLog.Message, parseSummary) {
		parseT.Fatalf("expected phase and summary in log message, got %+v", parseLastLog)
	}
	if !strings.Contains(parseLastLog.Fields["path"], parseSubject) {
		parseT.Fatalf("expected log path to mention %q, got %+v", parseSubject, parseLastLog)
	}
	if strings.TrimSpace(parseLastLog.Fields["top_frame"]) == "" || strings.TrimSpace(parseLastLog.Fields["runtime"]) == "" {
		parseT.Fatalf("expected panic fields in log entry, got %+v", parseLastLog)
	}
}

func TestWASMStackFrameMapperTranslatesTopFrame(parseT *testing.T) {
	ResetWASMStackFrameMapper()
	defer ResetWASMStackFrameMapper()

	SetWASMStackFrameMapper(func(parseFrame WASMStackFrame) (WASMStackFrame, bool) {
		if !strings.Contains(parseFrame.Function, "panicSourceMapHelper") {
			return WASMStackFrame{}, false
		}
		return WASMStackFrame{
			Function: "app.(*Dashboard).Render",
			File:     "C:/workspace/app/dashboard.go",
			Line:     42,
		}, true
	})

	parseContext := panicSourceMapHelper()
	if !strings.Contains(parseContext.TopFrame, "app.(*Dashboard).Render") || !strings.Contains(parseContext.TopFrame, "app/dashboard.go:42") {
		parseT.Fatalf("expected mapped top frame, got %+v", parseContext)
	}
	if len(parseContext.AppFrames) == 0 || !strings.Contains(parseContext.AppFrames[0], "app/dashboard.go:42") {
		parseT.Fatalf("expected mapped app frames, got %+v", parseContext.AppFrames)
	}
	parseReport := buildPanicReport(parseContext)
	if !strings.Contains(parseReport.TopFrame, "app/dashboard.go:42") || !strings.Contains(parseReport.Formatted, "app/dashboard.go:42") {
		parseT.Fatalf("expected mapped frame in report, got %+v", parseReport)
	}
}

func TestUnhandledPanicReportIncludesArtifactMetadata(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	ResetWASMArtifactMetadata()
	defer ClearDiagnostics()
	defer ClearLogs()
	defer ResetWASMArtifactMetadata()

	SetWASMArtifactMetadata(WASMArtifactMetadata{
		BuildID:      "build-42",
		ArtifactPath: "dist/app.wasm",
		SHA256:       "abc123",
		ManifestPath: "dist/wasm-release-manifest.json",
		SymbolSet:    "debug-sidecar",
		Version:      "2026.03.25",
	})

	parseReport := buildUnhandledPanicReport("runtime", PanicPhaseStartup, "bootstrap", "#app", nil, "startup boom")
	if !strings.Contains(parseReport.Formatted, "artifact_build_id=build-42") || !strings.Contains(parseReport.Formatted, "artifact_manifest=dist/wasm-release-manifest.json") {
		parseT.Fatalf("expected artifact metadata in formatted report, got %+v", parseReport)
	}
	if parseReport.Artifact.BuildID != "build-42" || parseReport.Artifact.SHA256 != "abc123" {
		parseT.Fatalf("expected artifact metadata in report payload, got %+v", parseReport.Artifact)
	}

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		parseT.Fatal("expected diagnostic")
	}
	parseLastDiagnostic := parseDiagnostics[len(parseDiagnostics)-1]
	if parseLastDiagnostic.Fields["artifact_build_id"] != "build-42" || parseLastDiagnostic.Fields["artifact_sha256"] != "abc123" || parseLastDiagnostic.Fields["artifact_manifest"] != "dist/wasm-release-manifest.json" {
		parseT.Fatalf("expected artifact fields in diagnostic, got %+v", parseLastDiagnostic)
	}

	parseLogs := GetLogs()
	if len(parseLogs) == 0 {
		parseT.Fatal("expected log entry")
	}
	parseLastLog := parseLogs[len(parseLogs)-1]
	if parseLastLog.Fields["artifact_path"] != "dist/app.wasm" || parseLastLog.Fields["artifact_symbols"] != "debug-sidecar" || parseLastLog.Fields["artifact_version"] != "2026.03.25" {
		parseT.Fatalf("expected artifact fields in log, got %+v", parseLastLog)
	}
}

func TestUnhandledRenderPanicReportsActionableDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseRt := &Runtime{}
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseFiber := &Fiber{typeOf: panicDiagnosticRenderComponent, parent: parseRoot}

	parseRecovered := recoverPanicString(parseT, func() {
		parseOutput := captureStdout(parseT, func() {
			_, _, _ = parseRt.renderFunctionComponent(parseFiber)
		})
		assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-RENDER", "inspect the component render path named by where/path first", "panicDiagnosticRenderComponent")
	})
	if parseRecovered != "render boom" {
		parseT.Fatalf("expected original render panic payload, got %q", parseRecovered)
	}

	parseReport := reportUnhandledPanic(parseFiber, boundaryPhaseRender, "render boom")
	if !strings.Contains(parseReport, "GWC-RUNTIME-PANIC-RENDER") ||
		!strings.Contains(parseReport, "uncaught render panic in panicDiagnosticRenderComponent") ||
		!strings.HasPrefix(parseReport, "render boom\n") ||
		!strings.Contains(parseReport, "where: ") ||
		!strings.Contains(parseReport, "path: panicDiagnosticRenderComponent") ||
		!strings.Contains(parseReport, "error: render boom") ||
		!strings.Contains(parseReport, "runtime: no error boundary recovered this render panic") ||
		!strings.Contains(parseReport, "next: Match code GWC-RUNTIME-PANIC-RENDER in automation") ||
		!strings.Contains(parseReport, "stack:\napp:") ||
		!strings.Contains(parseReport, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-render") {
		parseT.Fatalf("expected actionable render panic message, got %q", parseReport)
	}

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-RENDER", "panicDiagnosticRenderComponent", boundaryPhaseRender, "render boom")
}

func TestUnhandledEventPanicReportsActionableDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseRt := &Runtime{}
	parseRoot := &Fiber{typeOf: "ROOT"}
	parseOwner := &Fiber{typeOf: panicDiagnosticOwnerComponent, parent: parseRoot}
	parseHandler, parseOk := parseRt.wrapEventHandler(parseOwner, func() { panic("event boom") }).(func())
	if !parseOk {
		parseT.Fatal("expected wrapped event handler")
	}

	parseRecovered := recoverPanicString(parseT, parseHandler)
	if parseRecovered != "event boom" {
		parseT.Fatalf("expected original event panic payload, got %q", parseRecovered)
	}
	parseOutput := captureStdout(parseT, func() {
		func() {
			defer func() { _ = recover() }()
			parseHandler()
		}()
	})
	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-EVENT", "inspect the event handler named by where/path first", "panicDiagnosticOwnerComponent")

	parseReport := reportUnhandledPanic(parseOwner, boundaryPhaseEvent, "event boom")
	if !strings.Contains(parseReport, "GWC-RUNTIME-PANIC-EVENT") ||
		!strings.Contains(parseReport, "uncaught event panic in panicDiagnosticOwnerComponent") ||
		!strings.HasPrefix(parseReport, "event boom\n") ||
		!strings.Contains(parseReport, "where: ") ||
		!strings.Contains(parseReport, "path: panicDiagnosticOwnerComponent") ||
		!strings.Contains(parseReport, "error: event boom") ||
		!strings.Contains(parseReport, "runtime: no error boundary recovered this event panic") ||
		!strings.Contains(parseReport, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-event") {
		parseT.Fatalf("expected actionable event panic message, got %q", parseReport)
	}

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-EVENT", "panicDiagnosticOwnerComponent", boundaryPhaseEvent, "event boom")
}

func TestUnhandledEffectPanicReportsActionableDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	var parseRecovered string
	parseOutput := captureStdout(parseT, func() {
		parseRecovered = recoverPanicString(parseT, func() {
			parseRt.Render(CreateElement(panicDiagnosticEffectComponent, nil), parseContainer)
			flushScheduledWork(parseScheduler)
		})
	})
	if parseRecovered != "effect boom" {
		parseT.Fatalf("expected original effect panic payload, got %q", parseRecovered)
	}
	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-EFFECT", "inspect the effect body named by where/path first", "panicDiagnosticEffectComponent")

	parseReport := reportUnhandledPanic(parseRt.currentRoot.child, boundaryPhaseEffect, "effect boom")
	if !strings.Contains(parseReport, "GWC-RUNTIME-PANIC-EFFECT") ||
		!strings.Contains(parseReport, "uncaught effect panic in panicDiagnosticEffectComponent") ||
		!strings.HasPrefix(parseReport, "effect boom\n") ||
		!strings.Contains(parseReport, "where: ") ||
		!strings.Contains(parseReport, "path: panicDiagnosticEffectComponent") ||
		!strings.Contains(parseReport, "error: effect boom") ||
		!strings.Contains(parseReport, "runtime: no error boundary recovered this effect panic") ||
		!strings.Contains(parseReport, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-effect") {
		parseT.Fatalf("expected actionable effect panic message, got %q", parseReport)
	}

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-EFFECT", "panicDiagnosticEffectComponent", boundaryPhaseEffect, "effect boom")
}

func TestUnhandledBoundaryFallbackPanicPreservesRenderDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()
	withPanicLoggingOptions(parseT, PanicLoggingOptions{HideRawPanicOutput: true})

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler, HideRawPanicOutput: true})
	parseContainer := parseAdapter.CreateElement("div")
	parseBoundary := NewErrorBoundaryType()
	parseChild := func() *Element {
		panic("render boom")
	}

	parseOutput := captureStdout(parseT, func() {
		parseRt.Render(CreateElement(parseBoundary, map[string]interface{}{
			"errorFallback": func(parseErr error, reset func()) *Element {
				panic("fallback boom")
			},
		}, CreateElement(parseChild, nil)), parseContainer)
		flushScheduledWork(parseScheduler)
	})

	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-RENDER", "inspect the component render path named by where/path first", "ErrorBoundary")
	if strings.Contains(parseOutput, "GWC-RUNTIME-PANIC-DEFERRED") {
		parseT.Fatalf("expected boundary fallback panic to preserve render phase, got %q", parseOutput)
	}

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-RENDER", "ErrorBoundary", boundaryPhaseRender, "fallback boom")
}

func TestUnhandledCleanupPanicReportsActionableDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")

	parseRt.Render(CreateElement(panicDiagnosticCleanupComponent, nil), parseContainer)
	flushScheduledWork(parseScheduler)

	var parseRecovered string
	parseOutput := captureStdout(parseT, func() {
		parseRecovered = recoverPanicString(parseT, func() {
			parseRt.Render(CreateElement("div", nil, "replacement"), parseContainer)
			flushScheduledWork(parseScheduler)
		})
	})
	if parseRecovered != "cleanup boom" {
		parseT.Fatalf("expected original cleanup panic payload, got %q", parseRecovered)
	}
	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-CLEANUP", "inspect the cleanup path named by where/path first", "panicDiagnosticCleanupComponent")

	parseReport := reportUnhandledPanic(parseRt.currentRoot.child, boundaryPhaseCleanup, "cleanup boom")
	if !strings.Contains(parseReport, "GWC-RUNTIME-PANIC-CLEANUP") ||
		!strings.Contains(parseReport, "uncaught cleanup panic in panicDiagnosticCleanupComponent") ||
		!strings.HasPrefix(parseReport, "cleanup boom\n") ||
		!strings.Contains(parseReport, "where: ") ||
		!strings.Contains(parseReport, "path: panicDiagnosticCleanupComponent") ||
		!strings.Contains(parseReport, "error: cleanup boom") ||
		!strings.Contains(parseReport, "runtime: no error boundary recovered this cleanup panic") ||
		!strings.Contains(parseReport, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-cleanup") {
		parseT.Fatalf("expected actionable cleanup panic message, got %q", parseReport)
	}

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-CLEANUP", "panicDiagnosticCleanupComponent", boundaryPhaseCleanup, "cleanup boom")
}

func TestStartupPanicUsesWrappedMessage(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseRt := NewRuntime(Config{DOMAdapter: newQueryTestDOMAdapter(), Scheduler: newTestScheduler()})
	var parseRecovered string
	parseOutput := captureStdout(parseT, func() {
		parseRecovered = recoverPanicString(parseT, func() {
			parseRt.RenderTo("#missing", &Element{Type: "div", Props: map[string]interface{}{}})
		})
	})

	if parseRecovered != "RenderTo failed because the target container selector was not found: #missing" {
		parseT.Fatalf("expected wrapped startup panic, got %q", parseRecovered)
	}
	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-STARTUP", "verify the startup target and bootstrap inputs named by where/path first", "#missing")
}

func TestStrictHydrationPanicUsesWrappedMessage(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	parseContainer := parseAdapter.CreateElement("div")
	parseServerNode := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseRt.SetNextHydrationStrict(true)
	parseRt.Hydrate(CreateElement("div", nil, "client"), parseContainer)
	var parseRecovered string
	parseOutput := captureStdout(parseT, func() {
		parseRecovered = recoverPanicString(parseT, func() {
			runHydrationWork(parseT, parseScheduler)
		})
	})

	if parseRecovered != "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>" {
		parseT.Fatalf("expected wrapped hydration panic, got %q", parseRecovered)
	}
	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-HYDRATION", "compare server markup with the first client render at where/path first", "")
}

func TestDeferredPanicUsesWrappedMessage(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: parseScheduler})
	parseRt.ScheduleTransition(func() {
		panic("deferred boom")
	})

	var parseRecovered string
	parseOutput := captureStdout(parseT, func() {
		parseRecovered = recoverPanicString(parseT, func() {
			flushScheduledWork(parseScheduler)
		})
	})

	if parseRecovered != "deferred boom" {
		parseT.Fatalf("expected wrapped deferred panic, got %q", parseRecovered)
	}
	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-DEFERRED", "inspect the deferred callback or transition work named by where/path first", "scheduled transition")
}

func TestSSRRenderPanicUsesWrappedMessage(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	var parseRecovered string
	parseOutput := captureStdout(parseT, func() {
		parseRecovered = recoverPanicString(parseT, func() {
			_, _ = RenderToString(CreateElement(panicDiagnosticSSRComponent, nil))
		})
	})

	if parseRecovered != "ssr boom" {
		parseT.Fatalf("expected wrapped ssr panic, got %q", parseRecovered)
	}
	assertWrappedOutputQuality(parseT, parseOutput, "GWC-RUNTIME-PANIC-SSR", "inspect the server render path named by where/path first", "RenderToString")

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-SSR", "RenderToString", PanicPhaseSSR, "ssr boom")
}

func TestWrappedPanicStringsAreNotDoubleWrapped(parseT *testing.T) {
	parseMessage := ReportUnhandledPanicContext("runtime", PanicPhaseStartup, "RenderTo", "#app", nil, errors.New("boom"))
	parseWrapped := ReportUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler work loop", "", nil, parseMessage)
	if parseWrapped != parseMessage {
		parseT.Fatalf("expected wrapped message to pass through unchanged, got %q", parseWrapped)
	}
}

func TestHideRawPanicOutputSuppressesFinalRethrow(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()
	withPanicLoggingOptions(parseT, PanicLoggingOptions{HideRawPanicOutput: true})

	isParseDidPanic := false
	func() {
		defer func() {
			if recover() != nil {
				isParseDidPanic = true
			}
		}()
		panicFinalUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "App > Widget", []string{"App", "Widget"}, "render boom")
	}()

	if isParseDidPanic {
		parseT.Fatal("expected raw panic output suppression to avoid rethrowing the original panic")
	}

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-RENDER", "Widget", PanicPhaseRender, "render boom")
}

func TestSSRRenderPanicReturnsErrorWhenRawPanicOutputHidden(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()
	withPanicLoggingOptions(parseT, PanicLoggingOptions{HideRawPanicOutput: true})

	parseMarkup, parseErr := RenderToString(CreateElement(panicDiagnosticSSRComponent, nil))
	if parseErr == nil {
		parseT.Fatal("expected SSR panic to surface as an error when raw panic output is hidden")
	}
	if parseMarkup != "" {
		parseT.Fatalf("expected empty markup on suppressed SSR panic, got %q", parseMarkup)
	}
	if !strings.Contains(parseErr.Error(), "ssr boom") {
		parseT.Fatalf("expected SSR error to retain original panic summary, got %v", parseErr)
	}

	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-SSR", "RenderToString", PanicPhaseSSR, "ssr boom")
}

func TestUnhandledPanicHookReceivesStructuredReport(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	var parseReports []PanicReport
	withPanicLoggingOptions(parseT, PanicLoggingOptions{
		HideRawPanicOutput: true,
		OnReport: func(parseReport2 PanicReport) {
			parseReports = append(parseReports, parseReport2)
		},
	})

	panicFinalUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "App > Widget", []string{"App", "Widget"}, "render boom")

	if len(parseReports) != 1 {
		parseT.Fatalf("expected one hook report, got %+v", parseReports)
	}
	parseReport := parseReports[0]
	if parseReport.Code != "GWC-RUNTIME-PANIC-RENDER" || parseReport.Phase != PanicPhaseRender {
		parseT.Fatalf("expected render panic metadata, got %+v", parseReport)
	}
	if parseReport.Subject != "Widget" || parseReport.Path != "App > Widget" {
		parseT.Fatalf("expected subject/path to round-trip, got %+v", parseReport)
	}
	if parseReport.Where == "" || parseReport.Formatted == "" || !strings.Contains(parseReport.Formatted, "GWC-RUNTIME-PANIC-RENDER") {
		parseT.Fatalf("expected structured hook report to include where and formatted output, got %+v", parseReport)
	}
	if !strings.Contains(parseReport.Remediation, "Match code GWC-RUNTIME-PANIC-RENDER in automation") {
		parseT.Fatalf("expected actionable remediation in hook payload, got %+v", parseReport)
	}

	parseWrapped := markUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "scheduler", nil, "deferred boom")
	_, _ = finalizeUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "scheduler", nil, parseWrapped)
	if len(parseReports) != 2 {
		parseT.Fatalf("expected exactly one additional hook report for mark/finalize path, got %+v", parseReports)
	}
}
