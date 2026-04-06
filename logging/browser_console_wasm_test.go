//go:build js && wasm
// +build js,wasm

package logging

import (
	"strings"
	"syscall/js"
	"testing"
)

// storeLoggingTestGlobal replaces a global value for the duration of a wasm test.
func storeLoggingTestGlobal(parseT *testing.T, parseName string, parseValue js.Value) func() {
	parseT.Helper()
	parseGlobal := js.Global()
	parseOriginal := parseGlobal.Get(parseName)
	parseGlobal.Set(parseName, parseValue)
	return func() {
		parseGlobal.Set(parseName, parseOriginal)
	}
}

// buildLoggingTestConsole builds a console stub that records each structured log entry.
func buildLoggingTestConsole() (js.Value, js.Value, func()) {
	parseLogs := js.Global().Get("Array").New()
	parseConsole := js.Global().Get("Object").New()
	parseFuncs := make([]js.Func, 0, 6)
	for _, parseLevel := range []string{"debug", "info", "warn", "error", "trace", "log"} {
		parseLevel2 := parseLevel
		parseFunc := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseRecord := js.Global().Get("Object").New()
			parseRecord.Set("level", parseLevel2)
			if len(parseArgs) > 0 {
				parseRecord.Set("entry", parseArgs[0])
			}
			parseLogs.Call("push", parseRecord)
			return nil
		})
		parseConsole.Set(parseLevel2, parseFunc)
		parseFuncs = append(parseFuncs, parseFunc)
	}
	return parseConsole, parseLogs, func() {
		for _, parseFunc := range parseFuncs {
			parseFunc.Release()
		}
	}
}

// buildLoggingTestEventTarget builds a DOM-like target that records event handlers by name.
func buildLoggingTestEventTarget() (js.Value, func()) {
	parseTarget := js.Global().Get("Object").New()
	parseHandlers := js.Global().Get("Object").New()
	parseTarget.Set("__handlers", parseHandlers)
	parseAdd := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 2 {
			parseHandlers.Set(parseArgs[0].String(), parseArgs[1])
		}
		return nil
	})
	parseRemove := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) >= 1 {
			parseHandlers.Set(parseArgs[0].String(), js.Undefined())
		}
		return nil
	})
	parseTarget.Set("addEventListener", parseAdd)
	parseTarget.Set("removeEventListener", parseRemove)
	return parseTarget, func() {
		parseAdd.Release()
		parseRemove.Release()
	}
}

// runLoggingTestEvent invokes a previously registered mock event handler.
func runLoggingTestEvent(parseT *testing.T, parseTarget js.Value, parseEvent string, parseArgs ...interface{}) {
	parseT.Helper()
	parseHandler := parseTarget.Get("__handlers").Get(parseEvent)
	if parseHandler.Type() != js.TypeFunction {
		parseT.Fatalf("expected handler for %q", parseEvent)
	}
	parseValues := make([]interface{}, 0, len(parseArgs))
	parseValues = append(parseValues, parseArgs...)
	parseHandler.Invoke(parseValues...)
}

// getLoggingTestRecord finds the first recorded console entry with the requested message.
func getLoggingTestRecord(parseLogs js.Value, parseMessage string) (js.Value, bool) {
	for parseIndex := 0; parseIndex < parseLogs.Length(); parseIndex++ {
		parseRecord := parseLogs.Index(parseIndex)
		parseEntry := parseRecord.Get("entry")
		if parseEntry.Truthy() && parseEntry.Get("message").String() == parseMessage {
			return parseRecord, true
		}
	}
	return js.Undefined(), false
}

// TestNormalizeBrowserConsoleOptionsEnablesDefaults verifies the zero-value option set opts into the full browser surface.
func TestNormalizeBrowserConsoleOptionsEnablesDefaults(parseT *testing.T) {
	parseResolved := normalizeBrowserConsoleOptions(BrowserConsoleOptions{})
	if !parseResolved.LogClicks || !parseResolved.LogChanges || !parseResolved.LogSubmits || !parseResolved.LogWindowErrors || !parseResolved.LogNavigation || !parseResolved.LogMount || !parseResolved.LogDocumentReady || !parseResolved.LogUnhandledRejections || !parseResolved.LogVisibility || !parseResolved.LogResize || !parseResolved.LogBeforeUnload {
		parseT.Fatalf("expected zero-value options to enable all browser logging, got %#v", parseResolved)
	}

	parseCustom := BrowserConsoleOptions{Scope: "demo", LogResize: true}
	if parseGot := normalizeBrowserConsoleOptions(parseCustom); parseGot != parseCustom {
		parseT.Fatalf("expected explicit options to remain unchanged, got %#v", parseGot)
	}
}

// TestResolveScopePrefersExplicitTitlePathAndFallback verifies scope resolution order.
func TestResolveScopePrefersExplicitTitlePathAndFallback(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("pathname", "/docs")
	parseLocation.Set("hash", "#intro")
	parseWindow.Set("location", parseLocation)
	parseDocument := js.Global().Get("Object").New()
	parseDocument.Set("title", " Example App ")

	if parseGot := resolveScope(" custom ", parseWindow, parseDocument); parseGot != "custom" {
		parseT.Fatalf("resolveScope(explicit) = %q, want custom", parseGot)
	}
	parseDocument.Set("title", " Docs ")
	if parseGot := resolveScope("", parseWindow, parseDocument); parseGot != "Docs" {
		parseT.Fatalf("resolveScope(title) = %q, want Docs", parseGot)
	}
	parseDocument.Set("title", " ")
	if parseGot := resolveScope("", parseWindow, parseDocument); parseGot != "/docs#intro" {
		parseT.Fatalf("resolveScope(path) = %q, want /docs#intro", parseGot)
	}
	parseLocation.Set("pathname", " ")
	parseLocation.Set("hash", " ")
	if parseGot := resolveScope("", parseWindow, parseDocument); parseGot != "browser" {
		parseT.Fatalf("resolveScope(fallback) = %q, want browser", parseGot)
	}
}

// TestBrowserConsoleHelpersFormatTargets verifies DOM helper behavior without depending on a real browser DOM.
func TestBrowserConsoleHelpersFormatTargets(parseT *testing.T) {
	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("pathname", " /settings ")
	parseLocation.Set("hash", "#security")
	parseWindow.Set("location", parseLocation)
	if parseGot := pathValue(parseWindow); parseGot != "/settings#security" {
		parseT.Fatalf("pathValue() = %q, want /settings#security", parseGot)
	}
	parseWindow.Set("location", js.Undefined())
	if parseGot := pathValue(parseWindow); parseGot != "" {
		parseT.Fatalf("pathValue() without location = %q, want empty", parseGot)
	}

	parseButton := js.Global().Get("Object").New()
	parseButton.Set("tagName", " BUTTON ")
	parseButton.Set("id", "save")
	parseButton.Set("name", "primary")
	parseButton.Set("textContent", strings.Repeat(" Click   Save ", 20))
	parseButton.Set("role", "button")
	parseButton.Set("type", "submit")
	parseButton.Set("href", "/save")
	parseFallbackClosest := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseButton
	})
	defer parseFallbackClosest.Release()

	parseSpan := js.Global().Get("Object").New()
	parseSpan.Set("tagName", "SPAN")
	parseSpan.Set("textContent", "inner")
	parseClosest := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseButton
	})
	defer parseClosest.Release()
	parseSpan.Set("closest", parseClosest)

	if parseGot := closestActionTarget(parseSpan); parseGot.Equal(parseButton) == false {
		parseT.Fatal("expected closestActionTarget() to prefer the closest actionable ancestor")
	}
	parseButton.Set("closest", parseFallbackClosest)
	if parseGot := closestActionTarget(parseButton); parseGot.Equal(parseButton) == false {
		parseT.Fatal("expected closestActionTarget() to fall back to the original target")
	}

	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("target", parseSpan)
	if parseGot := eventTarget([]js.Value{parseEvent}); parseGot.Equal(parseSpan) == false {
		parseT.Fatal("expected eventTarget() to unwrap the event target")
	}
	if eventTarget(nil).Truthy() {
		parseT.Fatal("expected eventTarget() to return null for missing args")
	}

	parsePasswordInput := js.Global().Get("Object").New()
	parsePasswordInput.Set("tagName", "INPUT")
	parsePasswordInput.Set("type", "password")
	parsePasswordInput.Set("value", "super-secret")
	parsePasswordInput.Set("name", "secret")
	parsePasswordInput.Set("textContent", " ")
	if !isFormField(parsePasswordInput) {
		parseT.Fatal("expected password input to be treated as a form field")
	}
	parsePasswordDetails := targetDetails(parsePasswordInput, true)
	if parsePasswordDetails["value"] != redactedInteractionValue {
		parseT.Fatalf("expected password values to be redacted, got %#v", parsePasswordDetails)
	}
	if parsePasswordDetails["tag"] != "input" {
		parseT.Fatalf("expected lowercase tag name, got %#v", parsePasswordDetails)
	}
	if isFormField(parseButton) {
		parseT.Fatal("expected button to be excluded from form field detection")
	}

	parseButtonDetails := targetDetails(parseButton, false)
	if parseButtonDetails["href"] != "/save" || parseButtonDetails["role"] != "button" || parseButtonDetails["type"] != "submit" {
		parseT.Fatalf("expected optional attributes to be included, got %#v", parseButtonDetails)
	}
	if !strings.HasPrefix(parseButtonDetails["text"].(string), "Click Save") {
		parseT.Fatalf("expected preview text to collapse whitespace, got %#v", parseButtonDetails)
	}

	if parseGot := elementTag(parseButton); parseGot != "button" {
		parseT.Fatalf("elementTag() = %q, want button", parseGot)
	}
	if parseGot := preview(strings.Repeat(" ab ", 80)); len(parseGot) != 120 {
		parseT.Fatalf("preview() length = %d, want 120", len(parseGot))
	}
	if parseGot := valueOrEmpty("  padded  "); parseGot != "padded" {
		parseT.Fatalf("valueOrEmpty() = %q, want padded", parseGot)
	}
	if parseGot := safeNumber(js.Undefined()); parseGot != 0 {
		parseT.Fatalf("safeNumber(undefined) = %d, want 0", parseGot)
	}
	if parseGot := safeNumber(js.ValueOf(42)); parseGot != 42 {
		parseT.Fatalf("safeNumber(42) = %d, want 42", parseGot)
	}
}

// TestWriteStructuredWasmUsesConsoleEntry verifies the wasm backend forwards structured entries to the console.
func TestWriteStructuredWasmUsesConsoleEntry(parseT *testing.T) {
	parseConsole, parseLogs, parseConsoleCleanup := buildLoggingTestConsole()
	defer parseConsoleCleanup()
	parseRestoreConsole := storeLoggingTestGlobal(parseT, "console", parseConsole)
	defer parseRestoreConsole()

	writeStructured(" WARN ", "demo", "structured message", Fields{"count": 3})

	parseRecord, parseOk := getLoggingTestRecord(parseLogs, "structured message")
	if !parseOk {
		parseT.Fatalf("expected console entry, got %d records", parseLogs.Length())
	}
	if parseRecord.Get("level").String() != "warn" {
		parseT.Fatalf("expected normalized warn level, got %q", parseRecord.Get("level").String())
	}
	parseEntry := parseRecord.Get("entry")
	if parseEntry.Get("scope").String() != "demo" || parseEntry.Get("count").Int() != 3 {
		parseT.Fatalf("unexpected structured entry: scope=%q count=%d", parseEntry.Get("scope").String(), parseEntry.Get("count").Int())
	}
}

// TestAttachBrowserConsoleRegistersHandlersAndWritesEvents verifies the wasm browser logger wires handlers and logs structured events.
func TestAttachBrowserConsoleRegistersHandlersAndWritesEvents(parseT *testing.T) {
	parseConsole, parseLogs, parseConsoleCleanup := buildLoggingTestConsole()
	defer parseConsoleCleanup()
	parseRestoreConsole := storeLoggingTestGlobal(parseT, "console", parseConsole)
	defer parseRestoreConsole()

	parseWindow, parseWindowCleanup := buildLoggingTestEventTarget()
	defer parseWindowCleanup()
	parseDocument, parseDocumentCleanup := buildLoggingTestEventTarget()
	defer parseDocumentCleanup()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("pathname", "/dashboard")
	parseLocation.Set("hash", "#recent")
	parseWindow.Set("location", parseLocation)
	parseWindow.Set("innerWidth", 1280)
	parseWindow.Set("innerHeight", 720)
	parseDocument.Set("title", " Example App ")
	parseDocument.Set("readyState", "interactive")
	parseDocument.Set("visibilityState", "hidden")
	parseRoot := js.Global().Get("Object").New()
	parseRoot.Set("tagName", "DIV")
	parseRoot.Set("textContent", "Dashboard ready")
	parseRoot.Set("childElementCount", 1)
	parseDocument.Set("body", parseRoot)
	parseQuerySelector := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 && parseArgs[0].String() == "#app" {
			return parseRoot
		}
		return js.Null()
	})
	defer parseQuerySelector.Release()
	parseDocument.Set("querySelector", parseQuerySelector)
	parseRestoreWindow := storeLoggingTestGlobal(parseT, "window", parseWindow)
	defer parseRestoreWindow()
	parseRestoreDocument := storeLoggingTestGlobal(parseT, "document", parseDocument)
	defer parseRestoreDocument()

	parseCleanup := AttachBrowserConsole(BrowserConsoleOptions{
		LogClicks:              true,
		LogChanges:             true,
		LogSubmits:             true,
		LogWindowErrors:        true,
		LogNavigation:          true,
		LogDocumentReady:       true,
		LogUnhandledRejections: true,
		LogVisibility:          true,
		LogResize:              true,
		LogBeforeUnload:        true,
	})
	if parseCleanup == nil {
		parseT.Fatal("expected cleanup function")
	}

	parseAnchor := js.Global().Get("Object").New()
	parseAnchor.Set("tagName", "A")
	parseAnchor.Set("id", "open-link")
	parseAnchor.Set("name", "docs")
	parseAnchor.Set("textContent", " Open   docs ")
	parseAnchor.Set("href", "/docs")
	parseAnchor.Set("role", "button")
	parseAnchor.Set("type", "button")
	parseSpan := js.Global().Get("Object").New()
	parseClosest := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseAnchor
	})
	defer parseClosest.Release()
	parseSpan.Set("closest", parseClosest)
	parseClickEvent := js.Global().Get("Object").New()
	parseClickEvent.Set("target", parseSpan)
	runLoggingTestEvent(parseT, parseDocument, "click", parseClickEvent)

	parseInput := js.Global().Get("Object").New()
	parseInput.Set("tagName", "INPUT")
	parseInput.Set("type", "password")
	parseInput.Set("value", "top-secret")
	parseInput.Set("name", "auth")
	parseChangeEvent := js.Global().Get("Object").New()
	parseChangeEvent.Set("target", parseInput)
	runLoggingTestEvent(parseT, parseDocument, "change", parseChangeEvent)

	parseForm := js.Global().Get("Object").New()
	parseForm.Set("tagName", "FORM")
	parseSubmitEvent := js.Global().Get("Object").New()
	parseSubmitEvent.Set("target", parseForm)
	runLoggingTestEvent(parseT, parseDocument, "submit", parseSubmitEvent)

	parseErrorEvent := js.Global().Get("Object").New()
	parseErrorEvent.Set("message", "boom")
	parseErrorEvent.Set("filename", "app.js")
	parseErrorEvent.Set("lineno", 7)
	parseErrorEvent.Set("colno", 9)
	runLoggingTestEvent(parseT, parseWindow, "error", parseErrorEvent)
	runLoggingTestEvent(parseT, parseWindow, "hashchange", js.Global().Get("Object").New())
	runLoggingTestEvent(parseT, parseWindow, "popstate", js.Global().Get("Object").New())
	runLoggingTestEvent(parseT, parseDocument, "DOMContentLoaded", js.Global().Get("Object").New())

	parseRejectionEvent := js.Global().Get("Object").New()
	parseRejectionEvent.Set("reason", "bad promise")
	runLoggingTestEvent(parseT, parseWindow, "unhandledrejection", parseRejectionEvent)
	runLoggingTestEvent(parseT, parseDocument, "visibilitychange", js.Global().Get("Object").New())
	runLoggingTestEvent(parseT, parseWindow, "resize", js.Global().Get("Object").New())
	runLoggingTestEvent(parseT, parseWindow, "beforeunload", js.Global().Get("Object").New())

	for _, parseMessage := range []string{
		"browser console attached",
		"document state",
		"browser console ready",
		"interaction",
		"field change",
		"form submit",
		"window error",
		"hash navigation",
		"history navigation",
		"document ready",
		"unhandled rejection",
		"visibility changed",
		"window resized",
		"page unloading",
	} {
		if _, parseOk := getLoggingTestRecord(parseLogs, parseMessage); !parseOk {
			parseT.Fatalf("expected log message %q, got %d records", parseMessage, parseLogs.Length())
		}
	}

	parseInteractionRecord, parseOk := getLoggingTestRecord(parseLogs, "interaction")
	if !parseOk {
		parseT.Fatal("expected interaction record")
	}
	if parseInteractionRecord.Get("entry").Get("scope").String() != "Example App" {
		parseT.Fatalf("expected document title scope, got %q", parseInteractionRecord.Get("entry").Get("scope").String())
	}

	parseFieldChangeRecord, parseOk := getLoggingTestRecord(parseLogs, "field change")
	if !parseOk {
		parseT.Fatal("expected field change record")
	}
	if parseFieldChangeRecord.Get("entry").Get("value").String() != redactedInteractionValue {
		parseT.Fatalf("expected redacted password value, got %q", parseFieldChangeRecord.Get("entry").Get("value").String())
	}

	parseCleanup()
	if parseWindow.Get("__handlers").Get("hashchange").Type() != js.TypeUndefined || parseDocument.Get("__handlers").Get("click").Type() != js.TypeUndefined {
		parseT.Fatal("expected cleanup to unregister handlers")
	}
}

// TestInstallMountObserverLogsOnceAndDisconnects verifies the mount observer only reports the first ready signal.
func TestInstallMountObserverLogsOnceAndDisconnects(parseT *testing.T) {
	parseConsole, parseLogs, parseConsoleCleanup := buildLoggingTestConsole()
	defer parseConsoleCleanup()
	parseRestoreConsole := storeLoggingTestGlobal(parseT, "console", parseConsole)
	defer parseRestoreConsole()

	parseRoot := js.Global().Get("Object").New()
	parseRoot.Set("tagName", "SECTION")
	parseRoot.Set("textContent", " Rendered   surface ")
	parseRoot.Set("childElementCount", 2)
	parseDocument := js.Global().Get("Object").New()
	parseDocument.Set("body", parseRoot)
	parseQuerySelector := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		return parseRoot
	})
	defer parseQuerySelector.Release()
	parseDocument.Set("querySelector", parseQuerySelector)

	parseObserverState := js.Global().Get("Object").New()
	parseObserverState.Set("disconnectCount", 0)
	var parseObserve js.Func
	var parseDisconnect js.Func
	parseObserverCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseObserver := js.Global().Get("Object").New()
		parseObserverState.Set("callback", parseArgs[0])
		parseObserve = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseObserverState.Set("target", parseArgs[0])
			return nil
		})
		parseDisconnect = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			parseObserverState.Set("disconnectCount", parseObserverState.Get("disconnectCount").Int()+1)
			return nil
		})
		parseObserver.Set("observe", parseObserve)
		parseObserver.Set("disconnect", parseDisconnect)
		return parseObserver
	})
	defer parseObserverCtor.Release()
	defer func() {
		if parseObserve.Type() != js.TypeUndefined {
			parseObserve.Release()
		}
		if parseDisconnect.Type() != js.TypeUndefined {
			parseDisconnect.Release()
		}
	}()
	parseRestoreObserver := storeLoggingTestGlobal(parseT, "MutationObserver", parseObserverCtor.Value)
	defer parseRestoreObserver()

	parseCleanup := installMountObserver(New("mount"), parseDocument)
	if parseCleanup == nil {
		parseT.Fatal("expected mount observer cleanup")
	}
	if parseObserverState.Get("target").Equal(parseRoot) == false {
		parseT.Fatal("expected observer to watch the render root")
	}

	parseCallback := parseObserverState.Get("callback")
	parseCallback.Invoke(js.Global().Get("Array").New())
	parseCallback.Invoke(js.Global().Get("Array").New())

	parseRecord, parseOk := getLoggingTestRecord(parseLogs, "render surface ready")
	if !parseOk {
		parseT.Fatalf("expected render surface ready log, got %d records", parseLogs.Length())
	}
	if parseRecord.Get("entry").Get("childCount").Int() != 2 || parseRecord.Get("entry").Get("target").String() != "section" {
		parseT.Fatalf("unexpected mount log payload: childCount=%d target=%q", parseRecord.Get("entry").Get("childCount").Int(), parseRecord.Get("entry").Get("target").String())
	}
	if parseObserverState.Get("disconnectCount").Int() != 1 {
		parseT.Fatalf("expected observer disconnect after first mount, got %d", parseObserverState.Get("disconnectCount").Int())
	}

	parseCleanup()
	if parseObserverState.Get("disconnectCount").Int() != 2 {
		parseT.Fatalf("expected cleanup to disconnect observer, got %d", parseObserverState.Get("disconnectCount").Int())
	}
}
