package runtime

import (
	"errors"
	"strings"
	"testing"
)

func withPanicLoggingOptions(t *testing.T, options PanicLoggingOptions) {
	t.Helper()
	previous := CurrentUnhandledPanicLoggingOptions()
	ConfigureUnhandledPanicLogging(options)
	t.Cleanup(func() {
		ConfigureUnhandledPanicLogging(previous)
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

func recoverPanicString(t *testing.T, fn func()) string {
	t.Helper()
	recovered := ""
	func() {
		defer func() {
			if value := recover(); value != nil {
				if original, ok := unwrapReportedPanic(value); ok {
					value = original
				}
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

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	previous := CurrentUnhandledPanicLoggingOptions()
	formattedReports := make([]string, 0, 1)
	ConfigureUnhandledPanicLogging(PanicLoggingOptions{
		HideRawPanicOutput: previous.HideRawPanicOutput,
		OnReport: func(report PanicReport) {
			formattedReports = append(formattedReports, report.Formatted)
			if previous.OnReport != nil {
				previous.OnReport(report)
			}
		},
	})
	defer ConfigureUnhandledPanicLogging(previous)
	fn()
	return strings.Join(formattedReports, "\n")
}

func assertWrappedOutputQuality(t *testing.T, output string, code string, guidance string, expectedPath string) {
	t.Helper()
	if strings.TrimSpace(output) == "" {
		t.Fatal("expected wrapped panic output")
	}
	if !strings.Contains(output, code) ||
		!strings.Contains(output, "where: ") ||
		!strings.Contains(output, "error: ") ||
		!strings.Contains(output, "runtime: ") ||
		!strings.Contains(output, "next: ") ||
		!strings.Contains(output, "docs: ACTIONABLE_ERRORS.md") {
		t.Fatalf("expected wrapped panic contract fields, got %q", output)
	}
	if expectedPath != "" && !strings.Contains(output, "path: "+expectedPath) {
		t.Fatalf("expected path %q in wrapped output, got %q", expectedPath, output)
	}
	if guidance != "" && !strings.Contains(output, guidance) {
		t.Fatalf("expected phase-specific guidance %q, got %q", guidance, output)
	}
	if strings.Contains(output, "(0x") {
		t.Fatalf("expected sanitized where/stack formatting without pointer payloads, got %q", output)
	}
	if strings.Contains(output, "markUnhandledPanicContext") ||
		strings.Contains(output, "finalizeUnhandledPanicContext") ||
		strings.Contains(output, "SetTimeout.func1") {
		t.Fatalf("expected compact first-pass stack without low-signal framework frames, got %q", output)
	}
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
	if strings.TrimSpace(lastDiagnostic.TopFrame) == "" || strings.TrimSpace(lastDiagnostic.Consequence) == "" {
		t.Fatalf("expected panic-specific diagnostic metadata, got %+v", lastDiagnostic)
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
	if strings.TrimSpace(lastLog.TopFrame) == "" || strings.TrimSpace(lastLog.Consequence) == "" {
		t.Fatalf("expected panic-specific log metadata, got %+v", lastLog)
	}
	if !strings.Contains(lastLog.Message, string(phase)) || !strings.Contains(lastLog.Message, summary) {
		t.Fatalf("expected phase and summary in log message, got %+v", lastLog)
	}
	if !strings.Contains(lastLog.Fields["path"], subject) {
		t.Fatalf("expected log path to mention %q, got %+v", subject, lastLog)
	}
	if strings.TrimSpace(lastLog.Fields["top_frame"]) == "" || strings.TrimSpace(lastLog.Fields["runtime"]) == "" {
		t.Fatalf("expected panic fields in log entry, got %+v", lastLog)
	}
}

func TestWASMStackFrameMapperTranslatesTopFrame(t *testing.T) {
	ResetWASMStackFrameMapper()
	defer ResetWASMStackFrameMapper()

	SetWASMStackFrameMapper(func(frame WASMStackFrame) (WASMStackFrame, bool) {
		if !strings.Contains(frame.Function, "panicSourceMapHelper") {
			return WASMStackFrame{}, false
		}
		return WASMStackFrame{
			Function: "app.(*Dashboard).Render",
			File:     "C:/workspace/app/dashboard.go",
			Line:     42,
		}, true
	})

	context := panicSourceMapHelper()
	if !strings.Contains(context.TopFrame, "app.(*Dashboard).Render") || !strings.Contains(context.TopFrame, "app/dashboard.go:42") {
		t.Fatalf("expected mapped top frame, got %+v", context)
	}
	if len(context.AppFrames) == 0 || !strings.Contains(context.AppFrames[0], "app/dashboard.go:42") {
		t.Fatalf("expected mapped app frames, got %+v", context.AppFrames)
	}
	report := buildPanicReport(context)
	if !strings.Contains(report.TopFrame, "app/dashboard.go:42") || !strings.Contains(report.Formatted, "app/dashboard.go:42") {
		t.Fatalf("expected mapped frame in report, got %+v", report)
	}
}

func TestUnhandledPanicReportIncludesArtifactMetadata(t *testing.T) {
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

	report := buildUnhandledPanicReport("runtime", PanicPhaseStartup, "bootstrap", "#app", nil, "startup boom")
	if !strings.Contains(report.Formatted, "artifact_build_id=build-42") || !strings.Contains(report.Formatted, "artifact_manifest=dist/wasm-release-manifest.json") {
		t.Fatalf("expected artifact metadata in formatted report, got %+v", report)
	}
	if report.Artifact.BuildID != "build-42" || report.Artifact.SHA256 != "abc123" {
		t.Fatalf("expected artifact metadata in report payload, got %+v", report.Artifact)
	}

	diagnostics := GetDiagnostics()
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostic")
	}
	lastDiagnostic := diagnostics[len(diagnostics)-1]
	if lastDiagnostic.Fields["artifact_build_id"] != "build-42" || lastDiagnostic.Fields["artifact_sha256"] != "abc123" || lastDiagnostic.Fields["artifact_manifest"] != "dist/wasm-release-manifest.json" {
		t.Fatalf("expected artifact fields in diagnostic, got %+v", lastDiagnostic)
	}

	logs := GetLogs()
	if len(logs) == 0 {
		t.Fatal("expected log entry")
	}
	lastLog := logs[len(logs)-1]
	if lastLog.Fields["artifact_path"] != "dist/app.wasm" || lastLog.Fields["artifact_symbols"] != "debug-sidecar" || lastLog.Fields["artifact_version"] != "2026.03.25" {
		t.Fatalf("expected artifact fields in log, got %+v", lastLog)
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
		output := captureStdout(t, func() {
			_, _, _ = rt.renderFunctionComponent(fiber)
		})
		assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-RENDER", "inspect the component render path named by where/path first", "panicDiagnosticRenderComponent")
	})
	if recovered != "render boom" {
		t.Fatalf("expected original render panic payload, got %q", recovered)
	}

	report := reportUnhandledPanic(fiber, boundaryPhaseRender, "render boom")
	if !strings.Contains(report, "GWC-RUNTIME-PANIC-RENDER") ||
		!strings.Contains(report, "uncaught render panic in panicDiagnosticRenderComponent") ||
		!strings.HasPrefix(report, "render boom\n") ||
		!strings.Contains(report, "where: ") ||
		!strings.Contains(report, "path: panicDiagnosticRenderComponent") ||
		!strings.Contains(report, "error: render boom") ||
		!strings.Contains(report, "runtime: no error boundary recovered this render panic") ||
		!strings.Contains(report, "next: Match code GWC-RUNTIME-PANIC-RENDER in automation") ||
		!strings.Contains(report, "stack:\napp:") ||
		!strings.Contains(report, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-render") {
		t.Fatalf("expected actionable render panic message, got %q", report)
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
	if recovered != "event boom" {
		t.Fatalf("expected original event panic payload, got %q", recovered)
	}
	output := captureStdout(t, func() {
		func() {
			defer func() { _ = recover() }()
			handler()
		}()
	})
	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-EVENT", "inspect the event handler named by where/path first", "panicDiagnosticOwnerComponent")

	report := reportUnhandledPanic(owner, boundaryPhaseEvent, "event boom")
	if !strings.Contains(report, "GWC-RUNTIME-PANIC-EVENT") ||
		!strings.Contains(report, "uncaught event panic in panicDiagnosticOwnerComponent") ||
		!strings.HasPrefix(report, "event boom\n") ||
		!strings.Contains(report, "where: ") ||
		!strings.Contains(report, "path: panicDiagnosticOwnerComponent") ||
		!strings.Contains(report, "error: event boom") ||
		!strings.Contains(report, "runtime: no error boundary recovered this event panic") ||
		!strings.Contains(report, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-event") {
		t.Fatalf("expected actionable event panic message, got %q", report)
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

	var recovered string
	output := captureStdout(t, func() {
		recovered = recoverPanicString(t, func() {
			rt.Render(CreateElement(panicDiagnosticEffectComponent, nil), container)
			flushScheduledWork(scheduler)
		})
	})
	if recovered != "effect boom" {
		t.Fatalf("expected original effect panic payload, got %q", recovered)
	}
	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-EFFECT", "inspect the effect body named by where/path first", "panicDiagnosticEffectComponent")

	report := reportUnhandledPanic(rt.currentRoot.child, boundaryPhaseEffect, "effect boom")
	if !strings.Contains(report, "GWC-RUNTIME-PANIC-EFFECT") ||
		!strings.Contains(report, "uncaught effect panic in panicDiagnosticEffectComponent") ||
		!strings.HasPrefix(report, "effect boom\n") ||
		!strings.Contains(report, "where: ") ||
		!strings.Contains(report, "path: panicDiagnosticEffectComponent") ||
		!strings.Contains(report, "error: effect boom") ||
		!strings.Contains(report, "runtime: no error boundary recovered this effect panic") ||
		!strings.Contains(report, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-effect") {
		t.Fatalf("expected actionable effect panic message, got %q", report)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-EFFECT", "panicDiagnosticEffectComponent", boundaryPhaseEffect, "effect boom")
}

func TestUnhandledBoundaryFallbackPanicPreservesRenderDiagnostic(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()
	withPanicLoggingOptions(t, PanicLoggingOptions{HideRawPanicOutput: true})

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler, HideRawPanicOutput: true})
	container := adapter.CreateElement("div")
	boundary := NewErrorBoundaryType()
	child := func() *Element {
		panic("render boom")
	}

	output := captureStdout(t, func() {
		rt.Render(CreateElement(boundary, map[string]interface{}{
			"errorFallback": func(err error, reset func()) *Element {
				panic("fallback boom")
			},
		}, CreateElement(child, nil)), container)
		flushScheduledWork(scheduler)
	})

	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-RENDER", "inspect the component render path named by where/path first", "ErrorBoundary")
	if strings.Contains(output, "GWC-RUNTIME-PANIC-DEFERRED") {
		t.Fatalf("expected boundary fallback panic to preserve render phase, got %q", output)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-RENDER", "ErrorBoundary", boundaryPhaseRender, "fallback boom")
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

	var recovered string
	output := captureStdout(t, func() {
		recovered = recoverPanicString(t, func() {
			rt.Render(CreateElement("div", nil, "replacement"), container)
			flushScheduledWork(scheduler)
		})
	})
	if recovered != "cleanup boom" {
		t.Fatalf("expected original cleanup panic payload, got %q", recovered)
	}
	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-CLEANUP", "inspect the cleanup path named by where/path first", "panicDiagnosticCleanupComponent")

	report := reportUnhandledPanic(rt.currentRoot.child, boundaryPhaseCleanup, "cleanup boom")
	if !strings.Contains(report, "GWC-RUNTIME-PANIC-CLEANUP") ||
		!strings.Contains(report, "uncaught cleanup panic in panicDiagnosticCleanupComponent") ||
		!strings.HasPrefix(report, "cleanup boom\n") ||
		!strings.Contains(report, "where: ") ||
		!strings.Contains(report, "path: panicDiagnosticCleanupComponent") ||
		!strings.Contains(report, "error: cleanup boom") ||
		!strings.Contains(report, "runtime: no error boundary recovered this cleanup panic") ||
		!strings.Contains(report, "ACTIONABLE_ERRORS.md#gwc-runtime-panic-cleanup") {
		t.Fatalf("expected actionable cleanup panic message, got %q", report)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-CLEANUP", "panicDiagnosticCleanupComponent", boundaryPhaseCleanup, "cleanup boom")
}

func TestStartupPanicUsesWrappedMessage(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	rt := NewRuntime(Config{DOMAdapter: newQueryTestDOMAdapter(), Scheduler: newTestScheduler()})
	var recovered string
	output := captureStdout(t, func() {
		recovered = recoverPanicString(t, func() {
			rt.RenderTo("#missing", &Element{Type: "div", Props: map[string]interface{}{}})
		})
	})

	if recovered != "RenderTo failed because the target container selector was not found: #missing" {
		t.Fatalf("expected wrapped startup panic, got %q", recovered)
	}
	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-STARTUP", "verify the startup target and bootstrap inputs named by where/path first", "#missing")
}

func TestStrictHydrationPanicUsesWrappedMessage(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	container := adapter.CreateElement("div")
	serverNode := adapter.CreateElement("span")
	adapter.AppendChild(container, serverNode)

	rt.SetNextHydrationStrict(true)
	rt.Hydrate(CreateElement("div", nil, "client"), container)
	var recovered string
	output := captureStdout(t, func() {
		recovered = recoverPanicString(t, func() {
			runHydrationWork(t, scheduler)
		})
	})

	if recovered != "hydration fell back to client rendering for <div>: DOM node <span> did not match expected <div>" {
		t.Fatalf("expected wrapped hydration panic, got %q", recovered)
	}
	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-HYDRATION", "compare server markup with the first client render at where/path first", "")
}

func TestDeferredPanicUsesWrappedMessage(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: scheduler})
	rt.ScheduleTransition(func() {
		panic("deferred boom")
	})

	var recovered string
	output := captureStdout(t, func() {
		recovered = recoverPanicString(t, func() {
			flushScheduledWork(scheduler)
		})
	})

	if recovered != "deferred boom" {
		t.Fatalf("expected wrapped deferred panic, got %q", recovered)
	}
	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-DEFERRED", "inspect the deferred callback or transition work named by where/path first", "scheduled transition")
}

func TestSSRRenderPanicUsesWrappedMessage(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	var recovered string
	output := captureStdout(t, func() {
		recovered = recoverPanicString(t, func() {
			_, _ = RenderToString(CreateElement(panicDiagnosticSSRComponent, nil))
		})
	})

	if recovered != "ssr boom" {
		t.Fatalf("expected wrapped ssr panic, got %q", recovered)
	}
	assertWrappedOutputQuality(t, output, "GWC-RUNTIME-PANIC-SSR", "inspect the server render path named by where/path first", "RenderToString")

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-SSR", "RenderToString", PanicPhaseSSR, "ssr boom")
}

func TestWrappedPanicStringsAreNotDoubleWrapped(t *testing.T) {
	message := ReportUnhandledPanicContext("runtime", PanicPhaseStartup, "RenderTo", "#app", nil, errors.New("boom"))
	wrapped := ReportUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler work loop", "", nil, message)
	if wrapped != message {
		t.Fatalf("expected wrapped message to pass through unchanged, got %q", wrapped)
	}
}

func TestHideRawPanicOutputSuppressesFinalRethrow(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()
	withPanicLoggingOptions(t, PanicLoggingOptions{HideRawPanicOutput: true})

	didPanic := false
	func() {
		defer func() {
			if recover() != nil {
				didPanic = true
			}
		}()
		panicFinalUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "App > Widget", []string{"App", "Widget"}, "render boom")
	}()

	if didPanic {
		t.Fatal("expected raw panic output suppression to avoid rethrowing the original panic")
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-RENDER", "Widget", PanicPhaseRender, "render boom")
}

func TestSSRRenderPanicReturnsErrorWhenRawPanicOutputHidden(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()
	withPanicLoggingOptions(t, PanicLoggingOptions{HideRawPanicOutput: true})

	markup, err := RenderToString(CreateElement(panicDiagnosticSSRComponent, nil))
	if err == nil {
		t.Fatal("expected SSR panic to surface as an error when raw panic output is hidden")
	}
	if markup != "" {
		t.Fatalf("expected empty markup on suppressed SSR panic, got %q", markup)
	}
	if !strings.Contains(err.Error(), "ssr boom") {
		t.Fatalf("expected SSR error to retain original panic summary, got %v", err)
	}

	assertUnhandledPanicRecorded(t, "GWC-RUNTIME-PANIC-SSR", "RenderToString", PanicPhaseSSR, "ssr boom")
}

func TestUnhandledPanicHookReceivesStructuredReport(t *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()

	var reports []PanicReport
	withPanicLoggingOptions(t, PanicLoggingOptions{
		HideRawPanicOutput: true,
		OnReport: func(report PanicReport) {
			reports = append(reports, report)
		},
	})

	panicFinalUnhandledPanicContext("runtime", PanicPhaseRender, "Widget", "App > Widget", []string{"App", "Widget"}, "render boom")

	if len(reports) != 1 {
		t.Fatalf("expected one hook report, got %+v", reports)
	}
	report := reports[0]
	if report.Code != "GWC-RUNTIME-PANIC-RENDER" || report.Phase != PanicPhaseRender {
		t.Fatalf("expected render panic metadata, got %+v", report)
	}
	if report.Subject != "Widget" || report.Path != "App > Widget" {
		t.Fatalf("expected subject/path to round-trip, got %+v", report)
	}
	if report.Where == "" || report.Formatted == "" || !strings.Contains(report.Formatted, "GWC-RUNTIME-PANIC-RENDER") {
		t.Fatalf("expected structured hook report to include where and formatted output, got %+v", report)
	}
	if !strings.Contains(report.Remediation, "Match code GWC-RUNTIME-PANIC-RENDER in automation") {
		t.Fatalf("expected actionable remediation in hook payload, got %+v", report)
	}

	wrapped := markUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "scheduler", nil, "deferred boom")
	_, _ = finalizeUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduler", "scheduler", nil, wrapped)
	if len(reports) != 2 {
		t.Fatalf("expected exactly one additional hook report for mark/finalize path, got %+v", reports)
	}
}
