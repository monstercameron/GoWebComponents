//go:build js && wasm

package runtime

import (
	"syscall/js"
	"testing"
)

// panicConsoleTestHarness keeps the fake browser console reachable across wasm panic assertions.
type panicConsoleTestHarness struct {
	entriesValue js.Value
	eventsValue  js.Value
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
	if parseHarness.eventsValue.Length() != 1 {
		parseT.Fatalf("expected one gwc:runtime-panic event, got %d", parseHarness.eventsValue.Length())
	}
	parseEventPayload := parseHarness.eventsValue.Index(0)
	if parseEventPayload.Get("code").String() != parseReport.Code || parseEventPayload.Get("scope").String() != "runtime.panic" {
		parseT.Fatalf("expected structured runtime panic event payload, got %#v", parseEventPayload)
	}
	if parseEventPayload.Get("attributes").Get("appFrames").Length() != 1 {
		parseT.Fatalf("expected event payload to include app stack bucket, got %#v", parseEventPayload.Get("attributes"))
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
	parseEvents := parseArrayCtor.New()
	parseReleases := make([]js.Func, 0, 10)

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

	parseOriginalCustomEvent := js.Global().Get("CustomEvent")
	parseOriginalAddEventListener := js.Global().Get("addEventListener")
	parseOriginalRemoveEventListener := js.Global().Get("removeEventListener")
	parseOriginalDispatchEvent := js.Global().Get("dispatchEvent")
	parseEventListeners := make([]js.Value, 0, 2)
	parseCustomEvent := js.Global().Get("Function").New("type", "init", "this.type = type; this.detail = init && init.detail;")
	parseAddEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		if len(parseArgs) >= 2 && parseArgs[0].String() == "gwc:runtime-panic" {
			parseEventListeners = append(parseEventListeners, parseArgs[1])
		}
		return nil
	})
	parseRemoveEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		if len(parseArgs) < 2 || parseArgs[0].String() != "gwc:runtime-panic" {
			return nil
		}
		parseListener := parseArgs[1]
		for parseIndex, parseCurrent := range parseEventListeners {
			if parseCurrent.Equal(parseListener) {
				parseEventListeners = append(parseEventListeners[:parseIndex], parseEventListeners[parseIndex+1:]...)
				break
			}
		}
		return nil
	})
	parseDispatchEvent := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		if len(parseArgs) == 0 || parseArgs[0].Get("type").String() != "gwc:runtime-panic" {
			return true
		}
		for _, parseListener := range append([]js.Value(nil), parseEventListeners...) {
			parseListener.Invoke(parseArgs[0])
		}
		return true
	})
	js.Global().Set("CustomEvent", parseCustomEvent)
	js.Global().Set("addEventListener", parseAddEventListener)
	js.Global().Set("removeEventListener", parseRemoveEventListener)
	js.Global().Set("dispatchEvent", parseDispatchEvent)
	parseReleases = append(parseReleases, parseAddEventListener, parseRemoveEventListener, parseDispatchEvent)

	parseEventListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		if len(parseArgs) > 0 {
			parseEvents.Call("push", parseArgs[0].Get("detail"))
		}
		return nil
	})
	js.Global().Call("addEventListener", "gwc:runtime-panic", parseEventListener)
	parseReleases = append(parseReleases, parseEventListener)

	parseHarness.entriesValue = parseEntries
	parseHarness.eventsValue = parseEvents
	parseHarness.cleanupFunc = func() {
		js.Global().Call("removeEventListener", "gwc:runtime-panic", parseEventListener)
		js.Global().Set("console", parseOriginalConsole)
		js.Global().Set("CustomEvent", parseOriginalCustomEvent)
		js.Global().Set("addEventListener", parseOriginalAddEventListener)
		js.Global().Set("removeEventListener", parseOriginalRemoveEventListener)
		js.Global().Set("dispatchEvent", parseOriginalDispatchEvent)
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
