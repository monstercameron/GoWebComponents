//go:build js && wasm

package logging

import (
	"fmt"
	"strings"
	"syscall/js"
)

// BrowserConsoleOptions configures browser event logging on js/wasm builds.
type BrowserConsoleOptions struct {
	Scope                  string
	LogClicks              bool
	LogChanges             bool
	LogSubmits             bool
	LogWindowErrors        bool
	LogNavigation          bool
	LogMount               bool
	LogDocumentReady       bool
	LogUnhandledRejections bool
	LogVisibility          bool
	LogResize              bool
	LogBeforeUnload        bool
}

// AttachBrowserConsole wires common browser lifecycle and interaction events into the public logging surface.
func AttachBrowserConsole(parseOptions BrowserConsoleOptions) func() {
	parseWindow := js.Global().Get("window")
	parseDocument := js.Global().Get("document")
	if !parseWindow.Truthy() || !parseDocument.Truthy() {
		return func() {}
	}

	parseResolved := normalizeBrowserConsoleOptions(parseOptions)
	parseLogger := New(resolveScope(parseResolved.Scope, parseWindow, parseDocument))
	parseLogger.Info("browser console attached", Fields{
		"path": pathValue(parseWindow),
		"hash": valueOrEmpty(parseWindow.Get("location").Get("hash").String()),
	})

	parseCleanups := make([]func(), 0, 12)
	if parseResolved.LogClicks {
		parseCleanups = append(parseCleanups, addEventListener(parseDocument, "click", true, func(parseArgs []js.Value) {
			parseTarget := eventTarget(parseArgs)
			parseCandidate := closestActionTarget(parseTarget)
			if !parseCandidate.Truthy() {
				return
			}
			parseLogger.Info("interaction", targetDetails(parseCandidate, false))
		}))
	}
	if parseResolved.LogChanges {
		parseCleanups = append(parseCleanups, addEventListener(parseDocument, "change", true, func(parseArgs2 []js.Value) {
			parseTarget2 := eventTarget(parseArgs2)
			if !isFormField(parseTarget2) {
				return
			}
			parseLogger.Info("field change", targetDetails(parseTarget2, true))
		}))
	}
	if parseResolved.LogSubmits {
		parseCleanups = append(parseCleanups, addEventListener(parseDocument, "submit", true, func(parseArgs3 []js.Value) {
			parseTarget3 := eventTarget(parseArgs3)
			if !parseTarget3.Truthy() {
				return
			}
			parseLogger.Info("form submit", targetDetails(parseTarget3, false))
		}))
	}
	if parseResolved.LogWindowErrors {
		parseCleanups = append(parseCleanups, addEventListener(parseWindow, "error", false, func(parseArgs4 []js.Value) {
			if len(parseArgs4) == 0 {
				return
			}
			parseEvent := parseArgs4[0]
			parseLogger.Error("window error", Fields{
				// Keep the structured event payload distinct from the top-level log message.
				"errorMessage": valueOrEmpty(parseEvent.Get("message").String()),
				"source":       valueOrEmpty(parseEvent.Get("filename").String()),
				"line":         safeNumber(parseEvent.Get("lineno")),
				"column":       safeNumber(parseEvent.Get("colno")),
			})
		}))
	}
	if parseResolved.LogNavigation {
		parseCleanups = append(parseCleanups, addEventListener(parseWindow, "hashchange", false, func(parseArgs5 []js.Value) {
			parseLogger.Info("hash navigation", Fields{"path": pathValue(parseWindow)})
		}))
		parseCleanups = append(parseCleanups, addEventListener(parseWindow, "popstate", false, func(parseArgs6 []js.Value) {
			parseLogger.Info("history navigation", Fields{"path": pathValue(parseWindow)})
		}))
	}
	if parseResolved.LogMount {
		if parseCleanup := installMountObserver(parseLogger, parseDocument); parseCleanup != nil {
			parseCleanups = append(parseCleanups, parseCleanup)
		}
	}
	if parseResolved.LogDocumentReady {
		parseLogger.Info("document state", Fields{"state": valueOrEmpty(parseDocument.Get("readyState").String())})
		parseCleanups = append(parseCleanups, addEventListener(parseDocument, "DOMContentLoaded", false, func(parseArgs7 []js.Value) {
			parseLogger.Info("document ready", Fields{"state": valueOrEmpty(parseDocument.Get("readyState").String())})
		}))
	}
	if parseResolved.LogUnhandledRejections {
		parseCleanups = append(parseCleanups, addEventListener(parseWindow, "unhandledrejection", false, func(parseArgs8 []js.Value) {
			if len(parseArgs8) == 0 {
				return
			}
			parseEvent2 := parseArgs8[0]
			parseLogger.Error("unhandled rejection", Fields{
				"reason": valueOrEmpty(fmt.Sprint(parseEvent2.Get("reason"))),
			})
		}))
	}
	if parseResolved.LogVisibility {
		parseCleanups = append(parseCleanups, addEventListener(parseDocument, "visibilitychange", false, func(parseArgs9 []js.Value) {
			parseLogger.Info("visibility changed", Fields{"state": valueOrEmpty(parseDocument.Get("visibilityState").String())})
		}))
	}
	if parseResolved.LogResize {
		parseCleanups = append(parseCleanups, addEventListener(parseWindow, "resize", false, func(parseArgs10 []js.Value) {
			parseLogger.Info("window resized", Fields{
				"width":  safeNumber(parseWindow.Get("innerWidth")),
				"height": safeNumber(parseWindow.Get("innerHeight")),
			})
		}))
	}
	if parseResolved.LogBeforeUnload {
		parseCleanups = append(parseCleanups, addEventListener(parseWindow, "beforeunload", false, func(parseArgs11 []js.Value) {
			parseLogger.Info("page unloading", Fields{"path": pathValue(parseWindow)})
		}))
	}

	parseLogger.Info("browser console ready", nil)
	return func() {
		for parseIndex := len(parseCleanups) - 1; parseIndex >= 0; parseIndex-- {
			parseCleanups[parseIndex]()
		}
	}
}

func normalizeBrowserConsoleOptions(parseOptions BrowserConsoleOptions) BrowserConsoleOptions {
	if parseOptions.LogClicks || parseOptions.LogChanges || parseOptions.LogSubmits || parseOptions.LogWindowErrors || parseOptions.LogNavigation || parseOptions.LogMount || parseOptions.LogDocumentReady || parseOptions.LogUnhandledRejections || parseOptions.LogVisibility || parseOptions.LogResize || parseOptions.LogBeforeUnload {
		return parseOptions
	}
	parseOptions.LogClicks = true
	parseOptions.LogChanges = true
	parseOptions.LogSubmits = true
	parseOptions.LogWindowErrors = true
	parseOptions.LogNavigation = true
	parseOptions.LogMount = true
	parseOptions.LogDocumentReady = true
	parseOptions.LogUnhandledRejections = true
	parseOptions.LogVisibility = true
	parseOptions.LogResize = true
	parseOptions.LogBeforeUnload = true
	return parseOptions
}

func resolveScope(parseScope string, parseWindow, parseDocument js.Value) string {
	parseTrimmed := strings.TrimSpace(parseScope)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	parseTitle := strings.TrimSpace(parseDocument.Get("title").String())
	if parseTitle != "" {
		return parseTitle
	}
	parsePath := pathValue(parseWindow)
	if parsePath != "" {
		return parsePath
	}
	return "browser"
}

// recoverConsoleListener contains a panic in a browser-console listener so it
// cannot escape the JS bridge and terminate the wasm program, reporting it to
// the devtools console instead. The logging package has no runtime dependency,
// so it self-contains here rather than using runtime.RecoverContainedPanic.
func recoverConsoleListener(parseEvent string) {
	if parseRecovered := recover(); parseRecovered != nil {
		if parseConsole := js.Global().Get("console"); parseConsole.Truthy() {
			parseConsole.Call("error", "gwc logging: panic in "+parseEvent+" listener:", fmt.Sprint(parseRecovered))
		}
	}
}

func addEventListener(parseTarget js.Value, parseEvent string, isCapture bool, parseCallback func([]js.Value)) func() {
	if !parseTarget.Truthy() || parseTarget.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		// Contain panics: this helper wires document-wide interaction listeners
		// (click/submit/error/popstate/...), so an uncaught panic in the
		// callback or the app's Logger sink would escape the JS bridge and kill
		// the whole wasm program.
		defer recoverConsoleListener(parseEvent)
		if parseCallback != nil {
			parseCallback(parseArgs)
		}
		return nil
	})
	if isCapture {
		parseTarget.Call("addEventListener", parseEvent, parseHandler, true)
	} else {
		parseTarget.Call("addEventListener", parseEvent, parseHandler)
	}
	return func() {
		if !parseTarget.Truthy() || parseTarget.Get("removeEventListener").Type() != js.TypeFunction {
			parseHandler.Release()
			return
		}
		if isCapture {
			parseTarget.Call("removeEventListener", parseEvent, parseHandler, true)
		} else {
			parseTarget.Call("removeEventListener", parseEvent, parseHandler)
		}
		parseHandler.Release()
	}
}

func installMountObserver(parseLogger Logger, parseDocument js.Value) func() {
	parseObserverCtor := js.Global().Get("MutationObserver")
	if !parseObserverCtor.Truthy() {
		return nil
	}
	parseRoot := parseDocument.Call("querySelector", "#app")
	if !parseRoot.Truthy() {
		parseRoot = parseDocument.Get("body")
	}
	if !parseRoot.Truthy() {
		return nil
	}

	var parseObserver js.Value
	isParseMountedLogged := false
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		defer recoverConsoleListener("mutation-observer")
		if isParseMountedLogged {
			return nil
		}
		parseText := strings.TrimSpace(parseRoot.Get("textContent").String())
		parseChildCount := parseRoot.Get("childElementCount").Int()
		if parseText == "" && parseChildCount == 0 {
			return nil
		}
		isParseMountedLogged = true
		parseLogger.Info("render surface ready", Fields{
			"target":      elementTag(parseRoot),
			"childCount":  parseChildCount,
			"textPreview": preview(parseText),
		})
		if parseObserver.Truthy() {
			parseObserver.Call("disconnect")
		}
		return nil
	})
	parseObserver = parseObserverCtor.New(parseCallback)
	parseConfig := js.Global().Get("Object").New()
	parseConfig.Set("childList", true)
	parseConfig.Set("subtree", true)
	parseConfig.Set("characterData", true)
	parseObserver.Call("observe", parseRoot, parseConfig)
	return func() {
		if parseObserver.Truthy() {
			parseObserver.Call("disconnect")
		}
		parseCallback.Release()
	}
}

func pathValue(parseWindow js.Value) string {
	parseLocation := parseWindow.Get("location")
	if !parseLocation.Truthy() {
		return ""
	}
	parsePath := strings.TrimSpace(parseLocation.Get("pathname").String())
	parseHash := strings.TrimSpace(parseLocation.Get("hash").String())
	if parseHash != "" {
		return parsePath + parseHash
	}
	return parsePath
}

func eventTarget(parseArgs []js.Value) js.Value {
	if len(parseArgs) == 0 || !parseArgs[0].Truthy() {
		return js.Null()
	}
	return parseArgs[0].Get("target")
}

func closestActionTarget(parseTarget js.Value) js.Value {
	if !parseTarget.Truthy() {
		return js.Null()
	}
	parseClosest := parseTarget.Get("closest")
	if parseClosest.Type() == js.TypeFunction {
		parseCandidate := parseTarget.Call("closest", "button, a, [role='button']")
		if parseCandidate.Truthy() {
			return parseCandidate
		}
	}
	return parseTarget
}

func isFormField(parseTarget js.Value) bool {
	if !parseTarget.Truthy() {
		return false
	}
	parseTag := elementTag(parseTarget)
	return parseTag == "input" || parseTag == "select" || parseTag == "textarea"
}

func targetDetails(parseTarget js.Value, isIncludeValue bool) Fields {
	if !parseTarget.Truthy() {
		return Fields{}
	}
	parseDetails := Fields{
		"tag":  valueOrEmpty(elementTag(parseTarget)),
		"id":   valueOrEmpty(parseTarget.Get("id").String()),
		"name": valueOrEmpty(parseTarget.Get("name").String()),
		"text": preview(strings.TrimSpace(parseTarget.Get("textContent").String())),
	}
	if parseHref := valueOrEmpty(parseTarget.Get("href").String()); parseHref != "" {
		parseDetails["href"] = parseHref
	}
	if parseRole := valueOrEmpty(parseTarget.Get("role").String()); parseRole != "" {
		parseDetails["role"] = parseRole
	}
	if parseKind := valueOrEmpty(parseTarget.Get("type").String()); parseKind != "" {
		parseDetails["type"] = parseKind
	}
	if isIncludeValue {
		parseValue := redactInteractionValue(parseTarget.Get("type").String(), parseTarget.Get("value").String())
		parseDetails["value"] = preview(strings.TrimSpace(parseValue))
	}
	return parseDetails
}

func elementTag(parseTarget js.Value) string {
	if !parseTarget.Truthy() {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(parseTarget.Get("tagName").String()))
}

func preview(parseValue string) string {
	parseTrimmed := strings.Join(strings.Fields(strings.TrimSpace(parseValue)), " ")
	if len(parseTrimmed) > 120 {
		return parseTrimmed[:120]
	}
	return parseTrimmed
}

func valueOrEmpty(parseValue string) string {
	return strings.TrimSpace(parseValue)
}

func safeNumber(parseValue js.Value) int {
	if !parseValue.Truthy() {
		return 0
	}
	return parseValue.Int()
}
