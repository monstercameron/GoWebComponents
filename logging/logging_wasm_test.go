//go:build js && wasm
// +build js,wasm

package logging

import (
	"strings"
	"syscall/js"
	"testing"
)

// loggingTestHarness keeps the fake browser surface reachable across wasm assertions.
type loggingTestHarness struct {
	entriesValue          js.Value
	windowValue           js.Value
	documentValue         js.Value
	windowHandlersValue   js.Value
	documentHandlersValue js.Value
	rootValue             js.Value
	observerCallbackValue js.Value
	removedEventsValue    js.Value
	cleanupFunc           func()
}

// TestWriteStructuredWasmWritesStructuredEntry verifies the wasm backend delegates structured logs into console entries.
func TestWriteStructuredWasmWritesStructuredEntry(parseT *testing.T) {
	parseHarness := buildLoggingTestHarness(parseT)
	defer parseHarness.cleanupFunc()

	writeStructured("warn", "demo", "plain warning", Fields{"count": 3})

	parsePayload := findLoggingTestPayload(parseT, parseHarness.entriesValue, "plain warning")
	if parsePayload.Get("scope").String() != "demo" {
		parseT.Fatalf("expected scoped warning payload, got %q", parsePayload.Get("scope").String())
	}
	if parsePayload.Get("count").Int() != 3 {
		parseT.Fatalf("expected structured field count=3, got %v", parsePayload.Get("count"))
	}
}

// TestAttachBrowserConsoleWasmLogsLifecycleAndInteractions verifies the browser surface wiring logs the expected lifecycle and interaction events.
func TestAttachBrowserConsoleWasmLogsLifecycleAndInteractions(parseT *testing.T) {
	parseHarness := buildLoggingTestHarness(parseT)
	defer parseHarness.cleanupFunc()

	parseCleanup := AttachBrowserConsole(BrowserConsoleOptions{})
	defer parseCleanup()

	parseButton := buildLoggingTestElement("BUTTON")
	parseButton.Set("id", "save-button")
	parseButton.Set("textContent", " Save changes ")
	parseClosest := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		_ = parseArgs
		return parseButton
	})
	defer parseClosest.Release()

	parseClickTarget := buildLoggingTestElement("SPAN")
	parseClickTarget.Set("closest", parseClosest)
	emitLoggingTestEvent(parseHarness.documentHandlersValue, "click", buildLoggingTestEvent(parseClickTarget))

	parseChangeTarget := buildLoggingTestElement("INPUT")
	parseChangeTarget.Set("id", "secret-input")
	parseChangeTarget.Set("name", "password")
	parseChangeTarget.Set("type", "password")
	parseChangeTarget.Set("value", "super-secret")
	emitLoggingTestEvent(parseHarness.documentHandlersValue, "change", buildLoggingTestEvent(parseChangeTarget))

	parseSubmitTarget := buildLoggingTestElement("FORM")
	parseSubmitTarget.Set("id", "signup-form")
	parseSubmitTarget.Set("name", "signup")
	emitLoggingTestEvent(parseHarness.documentHandlersValue, "submit", buildLoggingTestEvent(parseSubmitTarget))

	parseErrorEvent := js.Global().Get("Object").New()
	parseErrorEvent.Set("message", "boom")
	parseErrorEvent.Set("filename", "app.js")
	parseErrorEvent.Set("lineno", 7)
	parseErrorEvent.Set("colno", 11)
	emitLoggingTestEvent(parseHarness.windowHandlersValue, "error", parseErrorEvent)

	parseLocation := parseHarness.windowValue.Get("location")
	parseLocation.Set("hash", "#updated")
	emitLoggingTestEvent(parseHarness.windowHandlersValue, "hashchange", js.Global().Get("Object").New())
	emitLoggingTestEvent(parseHarness.windowHandlersValue, "popstate", js.Global().Get("Object").New())

	parseHarness.rootValue.Set("textContent", strings.Repeat("mounted ", 20))
	parseHarness.rootValue.Set("childElementCount", 2)
	if !parseHarness.observerCallbackValue.Truthy() {
		parseT.Fatal("expected mutation observer callback to be registered")
	}
	parseHarness.observerCallbackValue.Invoke(js.Global().Get("Array").New())

	parseHarness.documentValue.Set("readyState", "complete")
	emitLoggingTestEvent(parseHarness.documentHandlersValue, "DOMContentLoaded", js.Global().Get("Object").New())

	parseRejectionEvent := js.Global().Get("Object").New()
	parseRejectionEvent.Set("reason", "network failed")
	emitLoggingTestEvent(parseHarness.windowHandlersValue, "unhandledrejection", parseRejectionEvent)

	parseHarness.documentValue.Set("visibilityState", "hidden")
	emitLoggingTestEvent(parseHarness.documentHandlersValue, "visibilitychange", js.Global().Get("Object").New())

	parseHarness.windowValue.Set("innerWidth", 1440)
	parseHarness.windowValue.Set("innerHeight", 900)
	emitLoggingTestEvent(parseHarness.windowHandlersValue, "resize", js.Global().Get("Object").New())
	emitLoggingTestEvent(parseHarness.windowHandlersValue, "beforeunload", js.Global().Get("Object").New())

	parseExpectedMessages := []string{
		"browser console attached",
		"interaction",
		"field change",
		"form submit",
		"window error",
		"hash navigation",
		"history navigation",
		"render surface ready",
		"document state",
		"document ready",
		"unhandled rejection",
		"visibility changed",
		"window resized",
		"page unloading",
		"browser console ready",
	}
	for _, parseExpected := range parseExpectedMessages {
		if !findLoggingTestPayload(parseT, parseHarness.entriesValue, parseExpected).Truthy() {
			parseT.Fatalf("expected log entry for %q, got %v", parseExpected, collectLoggingTestMessages(parseHarness.entriesValue))
		}
	}

	parseFieldChangePayload := findLoggingTestPayload(parseT, parseHarness.entriesValue, "field change")
	if parseFieldChangePayload.Get("scope").String() != "Demo App" {
		parseT.Fatalf("expected document title scope, got %q", parseFieldChangePayload.Get("scope").String())
	}
	if parseFieldChangePayload.Get("value").String() != redactedInteractionValue {
		parseT.Fatalf("expected redacted field value, got %q", parseFieldChangePayload.Get("value").String())
	}

	parseInteractionPayload := findLoggingTestPayload(parseT, parseHarness.entriesValue, "interaction")
	if parseInteractionPayload.Get("tag").String() != "button" || parseInteractionPayload.Get("id").String() != "save-button" {
		parseT.Fatalf("expected button interaction details, got tag=%q id=%q", parseInteractionPayload.Get("tag").String(), parseInteractionPayload.Get("id").String())
	}

	parseResizePayload := findLoggingTestPayload(parseT, parseHarness.entriesValue, "window resized")
	if parseResizePayload.Get("width").Int() != 1440 || parseResizePayload.Get("height").Int() != 900 {
		parseT.Fatalf("expected resize dimensions in payload, got width=%d height=%d", parseResizePayload.Get("width").Int(), parseResizePayload.Get("height").Int())
	}

	parseCleanup()
	if parseHarness.removedEventsValue.Length() == 0 {
		parseT.Fatal("expected event listener cleanup to unregister handlers")
	}
}

// TestBrowserConsoleHelpersWasmCoverFallbacks verifies the direct helper branches around scope, target, preview, and listener fallbacks.
func TestBrowserConsoleHelpersWasmCoverFallbacks(parseT *testing.T) {
	parseOptions := normalizeBrowserConsoleOptions(BrowserConsoleOptions{})
	if !parseOptions.LogClicks || !parseOptions.LogBeforeUnload || !parseOptions.LogWindowErrors {
		parseT.Fatalf("expected default logging options to enable browser events, got %#v", parseOptions)
	}

	parseExplicit := normalizeBrowserConsoleOptions(BrowserConsoleOptions{LogClicks: true})
	if !parseExplicit.LogClicks || parseExplicit.LogResize {
		parseT.Fatalf("expected explicit options to remain unchanged, got %#v", parseExplicit)
	}

	parseWindow := js.Global().Get("Object").New()
	parseLocation := js.Global().Get("Object").New()
	parseLocation.Set("pathname", "/from-path")
	parseLocation.Set("hash", "#part")
	parseWindow.Set("location", parseLocation)

	parseDocument := js.Global().Get("Object").New()
	parseDocument.Set("title", "")
	if parseGot := resolveScope(" custom ", parseWindow, parseDocument); parseGot != "custom" {
		parseT.Fatalf("expected trimmed custom scope, got %q", parseGot)
	}
	if parseGot := resolveScope("", parseWindow, parseDocument); parseGot != "/from-path#part" {
		parseT.Fatalf("expected path fallback scope, got %q", parseGot)
	}

	parseTarget := buildLoggingTestElement("A")
	parseTarget.Set("id", "docs-link")
	parseTarget.Set("name", "docs")
	parseTarget.Set("href", "https://example.com/docs")
	parseTarget.Set("role", "button")
	parseTarget.Set("type", "submit")
	parseTarget.Set("textContent", "  Read   docs  ")
	parseDetails := targetDetails(parseTarget, false)
	if parseDetails["href"] != "https://example.com/docs" || parseDetails["role"] != "button" || parseDetails["type"] != "submit" {
		parseT.Fatalf("expected optional target fields, got %#v", parseDetails)
	}
	if parseDetails["text"] != "Read docs" {
		parseT.Fatalf("expected condensed text preview, got %#v", parseDetails["text"])
	}

	if !closestActionTarget(parseTarget).Equal(parseTarget) {
		parseT.Fatal("expected target without closest() match to fall back to itself")
	}
	if !isFormField(buildLoggingTestElement("SELECT")) || isFormField(buildLoggingTestElement("DIV")) {
		parseT.Fatal("expected form-field detection to match only supported tags")
	}
	if eventTarget(nil).Truthy() {
		parseT.Fatal("expected missing event args to produce a null target")
	}
	if pathValue(js.Global().Get("Object").New()) != "" {
		parseT.Fatal("expected empty path for missing location")
	}
	if safeNumber(js.Undefined()) != 0 || safeNumber(js.ValueOf(7)) != 7 {
		parseT.Fatalf("expected safe number fallback, got %d and %d", safeNumber(js.Undefined()), safeNumber(js.ValueOf(7)))
	}
	if parseGot := valueOrEmpty("  trimmed  "); parseGot != "trimmed" {
		parseT.Fatalf("expected trimmed value, got %q", parseGot)
	}

	parsePreview := preview(strings.Repeat(" padded ", 30))
	if len(parsePreview) != 120 || strings.Contains(parsePreview, "  ") {
		parseT.Fatalf("expected condensed 120-char preview, got len=%d preview=%q", len(parsePreview), parsePreview)
	}

	parseCalls := 0
	parseRegistry := js.Global().Get("Object").New()
	parseTargetNoRemove := js.Global().Get("Object").New()
	parseAddListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		parseRegistry.Set(parseArgs[0].String(), parseArgs[1])
		return nil
	})
	defer parseAddListener.Release()
	parseTargetNoRemove.Set("addEventListener", parseAddListener)

	parseCleanup := addEventListener(parseTargetNoRemove, "change", false, func(parseArgs []js.Value) {
		_ = parseArgs
		parseCalls++
	})
	parseRegistry.Get("change").Invoke(js.Global().Get("Object").New())
	parseCleanup()
	if parseCalls != 1 {
		parseT.Fatalf("expected listener callback to run once, got %d", parseCalls)
	}
}

// buildLoggingTestHarness installs a fake browser surface that records structured console entries and event listener registration.
func buildLoggingTestHarness(parseT *testing.T) *loggingTestHarness {
	parseT.Helper()

	parseHarness := &loggingTestHarness{}
	parseObjectCtor := js.Global().Get("Object")
	parseArrayCtor := js.Global().Get("Array")
	parseEntries := parseArrayCtor.New()
	parseRemovedEvents := parseArrayCtor.New()
	parseWindowHandlers := parseObjectCtor.New()
	parseDocumentHandlers := parseObjectCtor.New()
	parseReleases := make([]js.Func, 0, 24)

	parseConsole := parseObjectCtor.New()
	for _, parseLevel := range []string{"log", "info", "warn", "error", "debug", "trace"} {
		parseLevel2 := parseLevel
		parseMethod := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			_ = parseThis
			parseRecord := parseObjectCtor.New()
			parseRecord.Set("level", parseLevel2)
			if len(parseArgs) > 0 {
				parseRecord.Set("payload", parseArgs[0])
			}
			parseEntries.Call("push", parseRecord)
			return nil
		})
		parseReleases = append(parseReleases, parseMethod)
		parseConsole.Set(parseLevel2, parseMethod)
	}

	// Keep event registration in plain JS objects so the production code interacts with a realistic browser-like surface.
	parseBuildRegistrar := func(parseRegistry js.Value, parsePrefix string) (js.Func, js.Func) {
		parseAdd := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			_ = parseThis
			parseEvent := parseArgs[0].String()
			parseList := parseRegistry.Get(parseEvent)
			if !parseList.Truthy() {
				parseList = parseArrayCtor.New()
				parseRegistry.Set(parseEvent, parseList)
			}
			parseList.Call("push", parseArgs[1])
			return nil
		})
		parseRemove := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
			_ = parseThis
			parseEvent := parseArgs[0].String()
			parseRemovedEvents.Call("push", parsePrefix+":"+parseEvent)
			return nil
		})
		return parseAdd, parseRemove
	}

	parseWindow := parseObjectCtor.New()
	parseWindow.Set("innerWidth", 1280)
	parseWindow.Set("innerHeight", 720)
	parseLocation := parseObjectCtor.New()
	parseLocation.Set("pathname", "/demo")
	parseLocation.Set("hash", "#initial")
	parseWindow.Set("location", parseLocation)
	parseWindowAdd, parseWindowRemove := parseBuildRegistrar(parseWindowHandlers, "window")
	parseReleases = append(parseReleases, parseWindowAdd, parseWindowRemove)
	parseWindow.Set("addEventListener", parseWindowAdd)
	parseWindow.Set("removeEventListener", parseWindowRemove)

	parseRoot := buildLoggingTestElement("DIV")
	parseBody := buildLoggingTestElement("BODY")
	parseDocument := parseObjectCtor.New()
	parseDocument.Set("title", "Demo App")
	parseDocument.Set("readyState", "interactive")
	parseDocument.Set("visibilityState", "visible")
	parseDocument.Set("body", parseBody)
	parseQuerySelector := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		_ = parseThis
		if len(parseArgs) > 0 && parseArgs[0].String() == "#app" {
			return parseRoot
		}
		return js.Null()
	})
	parseDocumentAdd, parseDocumentRemove := parseBuildRegistrar(parseDocumentHandlers, "document")
	parseReleases = append(parseReleases, parseQuerySelector, parseDocumentAdd, parseDocumentRemove)
	parseDocument.Set("querySelector", parseQuerySelector)
	parseDocument.Set("addEventListener", parseDocumentAdd)
	parseDocument.Set("removeEventListener", parseDocumentRemove)

	parseMutationObserverCtor := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) > 0 {
			parseHarness.observerCallbackValue = parseArgs[0]
		}
		parseObserve := js.FuncOf(func(parseObserverThis js.Value, parseObserverArgs []js.Value) interface{} {
			_ = parseObserverThis
			_ = parseObserverArgs
			return nil
		})
		parseDisconnect := js.FuncOf(func(parseObserverThis js.Value, parseObserverArgs []js.Value) interface{} {
			_ = parseObserverThis
			_ = parseObserverArgs
			parseRemovedEvents.Call("push", "observer:disconnect")
			return nil
		})
		parseReleases = append(parseReleases, parseObserve, parseDisconnect)
		parseThis.Set("observe", parseObserve)
		parseThis.Set("disconnect", parseDisconnect)
		return nil
	})
	parseReleases = append(parseReleases, parseMutationObserverCtor)

	parseOriginalConsole := js.Global().Get("console")
	parseOriginalWindow := js.Global().Get("window")
	parseOriginalDocument := js.Global().Get("document")
	parseOriginalObserver := js.Global().Get("MutationObserver")
	js.Global().Set("console", parseConsole)
	js.Global().Set("window", parseWindow)
	js.Global().Set("document", parseDocument)
	js.Global().Set("MutationObserver", parseMutationObserverCtor)

	parseHarness.entriesValue = parseEntries
	parseHarness.windowValue = parseWindow
	parseHarness.documentValue = parseDocument
	parseHarness.windowHandlersValue = parseWindowHandlers
	parseHarness.documentHandlersValue = parseDocumentHandlers
	parseHarness.rootValue = parseRoot
	parseHarness.removedEventsValue = parseRemovedEvents
	parseHarness.cleanupFunc = func() {
		js.Global().Set("console", parseOriginalConsole)
		js.Global().Set("window", parseOriginalWindow)
		js.Global().Set("document", parseOriginalDocument)
		js.Global().Set("MutationObserver", parseOriginalObserver)
		for parseIndex := len(parseReleases) - 1; parseIndex >= 0; parseIndex-- {
			parseReleases[parseIndex].Release()
		}
	}
	return parseHarness
}

// buildLoggingTestElement creates a minimal DOM-like element object for browser-console helper tests.
func buildLoggingTestElement(parseTag string) js.Value {
	parseElement := js.Global().Get("Object").New()
	parseElement.Set("tagName", parseTag)
	parseElement.Set("textContent", "")
	parseElement.Set("childElementCount", 0)
	parseElement.Set("id", "")
	parseElement.Set("name", "")
	parseElement.Set("href", "")
	parseElement.Set("role", "")
	parseElement.Set("type", "")
	parseElement.Set("value", "")
	return parseElement
}

// buildLoggingTestEvent wraps a target inside the event shape consumed by browser listeners.
func buildLoggingTestEvent(parseTarget js.Value) js.Value {
	parseEvent := js.Global().Get("Object").New()
	parseEvent.Set("target", parseTarget)
	return parseEvent
}

// emitLoggingTestEvent invokes every registered handler for the named event with the supplied event payload.
func emitLoggingTestEvent(parseRegistry js.Value, parseEvent string, parseEventValue js.Value) {
	parseHandlers := parseRegistry.Get(parseEvent)
	if !parseHandlers.Truthy() {
		return
	}
	for parseIndex := 0; parseIndex < parseHandlers.Length(); parseIndex++ {
		parseHandlers.Index(parseIndex).Invoke(parseEventValue)
	}
}

// findLoggingTestPayload returns the recorded structured payload whose message matches the requested text.
func findLoggingTestPayload(parseT *testing.T, parseEntries js.Value, parseMessage string) js.Value {
	parseT.Helper()
	for parseIndex := 0; parseIndex < parseEntries.Length(); parseIndex++ {
		parseRecord := parseEntries.Index(parseIndex)
		parsePayload := parseRecord.Get("payload")
		if parsePayload.Truthy() && parsePayload.Get("message").String() == parseMessage {
			parsePayload.Set("level", parseRecord.Get("level").String())
			return parsePayload
		}
	}
	parseT.Fatalf("expected payload for %q, got %v", parseMessage, collectLoggingTestMessages(parseEntries))
	return js.Null()
}

// collectLoggingTestMessages flattens recorded console payload messages to make test failures readable.
func collectLoggingTestMessages(parseEntries js.Value) []string {
	parseMessages := make([]string, 0, parseEntries.Length())
	for parseIndex := 0; parseIndex < parseEntries.Length(); parseIndex++ {
		parsePayload := parseEntries.Index(parseIndex).Get("payload")
		if parsePayload.Truthy() {
			parseMessages = append(parseMessages, parsePayload.Get("message").String())
		}
	}
	return parseMessages
}
