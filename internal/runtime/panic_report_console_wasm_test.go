//go:build js && wasm

package runtime

import (
	"syscall/js"
	"testing"
)

// panicConsoleTestHarness keeps the fake browser console reachable across wasm panic assertions.
type panicConsoleTestHarness struct {
	entriesValue js.Value
	cleanupFunc  func()
}

// TestEmitBrowserPanicReportWasmWritesStructuredErrorObject verifies wrapped panic reports land in the browser console as structured error objects.
func TestEmitBrowserPanicReportWasmWritesStructuredErrorObject(parseT *testing.T) {
	parseHarness := buildPanicConsoleTestHarness(parseT)
	defer parseHarness.cleanupFunc()

	parseReport := PanicReport{
		Source:          "runtime",
		Phase:           PanicPhaseRender,
		Subject:         "Widget",
		Where:           "render Widget",
		Path:            "App > Widget",
		ComponentStack:  []string{"App", "Widget"},
		Summary:         "render boom",
		Code:            "GWC-RUNTIME-PANIC-RENDER",
		Docs:            "ACTIONABLE_ERRORS.md#gwc-runtime-panic-render",
		Remediation:     "inspect the render path named by where/path first",
		Consequence:     "the current render cannot continue safely",
		AppFrames:       []string{"app.Widget"},
		FrameworkFrames: []string{"internal/runtime.renderFiber"},
		PlatformFrames:  []string{"runtime.goexit"},
		Formatted:       "formatted wrapped panic",
	}

	if !emitBrowserPanicReport(parseReport) {
		parseT.Fatal("expected browser panic report to emit into console")
	}

	parsePayload := findPanicConsoleTestPayload(parseT, parseHarness.entriesValue, parseReport.Code)
	if parsePayload.Get("console_method").String() != "error" {
		parseT.Fatalf("expected structured panic record to use console.error, got %q", parsePayload.Get("console_method").String())
	}
	if parsePayload.Get("level").String() != "error" || parsePayload.Get("severity_text").String() != "ERROR" || parsePayload.Get("severity_number").Int() != 17 {
		parseT.Fatalf("expected slog-like error level metadata, got %#v", parsePayload)
	}
	if parsePayload.Get("scope").String() != "runtime.panic" {
		parseT.Fatalf("expected panic scope, got %q", parsePayload.Get("scope").String())
	}
	if parsePayload.Get("message").String() == "" || parsePayload.Get("formatted").String() != "formatted wrapped panic" {
		parseT.Fatalf("expected panic message and formatted detail, got %#v", parsePayload)
	}
	if parsePayload.Get("phase").String() != string(PanicPhaseRender) || parsePayload.Get("subject").String() != "Widget" || parsePayload.Get("where").String() != "render Widget" {
		parseT.Fatalf("expected panic identity fields, got %#v", parsePayload)
	}
	if parsePayload.Get("attributes").Get("phase").String() != string(PanicPhaseRender) || parsePayload.Get("attributes").Get("componentStack").Length() != 2 {
		parseT.Fatalf("expected nested panic attributes, got %#v", parsePayload.Get("attributes"))
	}
	if !containsPanicConsoleTestMethod(parseHarness.entriesValue, "groupCollapsed") || !containsPanicConsoleTestMethod(parseHarness.entriesValue, "groupEnd") {
		parseT.Fatalf("expected grouped console panic output, got %v", collectPanicConsoleTestMethods(parseHarness.entriesValue))
	}
}

// TestPanicFinalUnhandledPanicContextWasmSuppressesRethrow verifies wrapped runtime panics log to console without rethrowing when raw panic output is hidden.
func TestPanicFinalUnhandledPanicContextWasmSuppressesRethrow(parseT *testing.T) {
	ClearDiagnostics()
	ClearLogs()
	defer ClearDiagnostics()
	defer ClearLogs()
	withPanicLoggingOptions(parseT, PanicLoggingOptions{HideRawPanicOutput: true})

	parseHarness := buildPanicConsoleTestHarness(parseT)
	defer parseHarness.cleanupFunc()

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
		parseT.Fatal("expected hidden raw panic output to avoid rethrowing on wasm")
	}

	parsePayload := findPanicConsoleTestPayload(parseT, parseHarness.entriesValue, "GWC-RUNTIME-PANIC-RENDER")
	if parsePayload.Get("console_method").String() != "error" || parsePayload.Get("error").String() != "render boom" {
		parseT.Fatalf("expected console.error panic payload with original summary, got %#v", parsePayload)
	}
	assertUnhandledPanicRecorded(parseT, "GWC-RUNTIME-PANIC-RENDER", "Widget", PanicPhaseRender, "render boom")
}

// buildPanicConsoleTestHarness installs a fake browser console that records panic-report console calls.
func buildPanicConsoleTestHarness(parseT *testing.T) *panicConsoleTestHarness {
	parseT.Helper()

	parseHarness := &panicConsoleTestHarness{}
	parseObjectCtor := js.Global().Get("Object")
	parseArrayCtor := js.Global().Get("Array")
	parseEntries := parseArrayCtor.New()
	parseReleases := make([]js.Func, 0, 6)

	parseConsole := parseObjectCtor.New()
	for _, parseMethodName := range []string{"groupCollapsed", "error", "log", "groupEnd"} {
		parseMethodName2 := parseMethodName
		parseMethod := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			_ = parseThis
			parseRecord := parseObjectCtor.New()
			parseRecord.Set("method", parseMethodName2)
			if len(parseArgs) > 0 {
				parseRecord.Set("payload", parseArgs[0])
			}
			parseEntries.Call("push", parseRecord)
			return nil
		})
		parseReleases = append(parseReleases, parseMethod)
		parseConsole.Set(parseMethodName2, parseMethod)
	}

	parseOriginalConsole := js.Global().Get("console")
	js.Global().Set("console", parseConsole)

	parseHarness.entriesValue = parseEntries
	parseHarness.cleanupFunc = func() {
		js.Global().Set("console", parseOriginalConsole)
		for parseIndex := len(parseReleases) - 1; parseIndex >= 0; parseIndex-- {
			parseReleases[parseIndex].Release()
		}
	}
	return parseHarness
}

// findPanicConsoleTestPayload returns the recorded structured panic payload for one diagnostic code.
func findPanicConsoleTestPayload(parseT *testing.T, parseEntries js.Value, parseCode string) js.Value {
	parseT.Helper()
	for parseIndex := 0; parseIndex < parseEntries.Length(); parseIndex++ {
		parseRecord := parseEntries.Index(parseIndex)
		parsePayload := parseRecord.Get("payload")
		if parsePayload.Type() != js.TypeObject {
			continue
		}
		if parsePayload.Get("code").String() == parseCode {
			parsePayload.Set("console_method", parseRecord.Get("method").String())
			return parsePayload
		}
	}
	parseT.Fatalf("expected structured panic payload for %q, got %v", parseCode, collectPanicConsoleTestMethods(parseEntries))
	return js.Null()
}

// containsPanicConsoleTestMethod reports whether the fake console recorded the named method.
func containsPanicConsoleTestMethod(parseEntries js.Value, parseMethod string) bool {
	for parseIndex := 0; parseIndex < parseEntries.Length(); parseIndex++ {
		if parseEntries.Index(parseIndex).Get("method").String() == parseMethod {
			return true
		}
	}
	return false
}

// collectPanicConsoleTestMethods flattens recorded console methods to keep wasm failure output readable.
func collectPanicConsoleTestMethods(parseEntries js.Value) []string {
	parseMethods := make([]string, 0, parseEntries.Length())
	for parseIndex := 0; parseIndex < parseEntries.Length(); parseIndex++ {
		parseMethods = append(parseMethods, parseEntries.Index(parseIndex).Get("method").String())
	}
	return parseMethods
}
