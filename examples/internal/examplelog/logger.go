//go:build js && wasm
// +build js,wasm

package examplelog

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/interop"
)

var retainedCallbacks []js.Func

func init() {
	parseWindow := js.Global().Get("window")
	parseDocument := js.Global().Get("document")
	if !parseWindow.Truthy() || !parseDocument.Truthy() {
		return
	}

	parseLabel := deriveLabel(parseWindow, parseDocument)
	logInfo(parseLabel, "logger attached", map[string]interface{}{
		"path": pathValue(parseWindow),
		"hash": valueOrEmpty(parseWindow.Get("location").Get("hash").String()),
	})

	installClickLogger(parseLabel, parseDocument)
	installChangeLogger(parseLabel, parseDocument)
	installSubmitLogger(parseLabel, parseDocument)
	installWindowErrorLogger(parseLabel, parseWindow)
	installNavigationLogger(parseLabel, parseWindow)
	installMountObserver(parseLabel, parseDocument)
	installReadyStateLogger(parseLabel, parseDocument)
	installUnhandledRejectionLogger(parseLabel, parseWindow)
	installVisibilityLogger(parseLabel, parseDocument)
	installResizeLogger(parseLabel, parseWindow)
	installBeforeUnloadLogger(parseLabel, parseWindow)
	logInfo(parseLabel, "logger ready", nil)
}

func deriveLabel(parseWindow, parseDocument js.Value) string {
	parseTitle := strings.TrimSpace(parseDocument.Get("title").String())
	if parseTitle != "" {
		return parseTitle
	}
	parsePath := pathValue(parseWindow)
	if parsePath != "" {
		return parsePath
	}
	return "unknown example"
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

func installClickLogger(parseLabel string, parseDocument js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseTarget := eventTarget(parseArgs)
		parseCandidate := closestActionTarget(parseTarget)
		if !parseCandidate.Truthy() {
			return nil
		}
		logInfo(parseLabel, "interaction", targetDetails(parseCandidate, false))
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseDocument.Call("addEventListener", "click", parseHandler, true)
}

func installChangeLogger(parseLabel string, parseDocument js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseTarget := eventTarget(parseArgs)
		if !isFormField(parseTarget) {
			return nil
		}
		logInfo(parseLabel, "field change", targetDetails(parseTarget, true))
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseDocument.Call("addEventListener", "change", parseHandler, true)
}

func installSubmitLogger(parseLabel string, parseDocument js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		parseTarget := eventTarget(parseArgs)
		if !parseTarget.Truthy() {
			return nil
		}
		logInfo(parseLabel, "form submit", targetDetails(parseTarget, false))
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseDocument.Call("addEventListener", "submit", parseHandler, true)
}

func installWindowErrorLogger(parseLabel string, parseWindow js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseEvent := parseArgs[0]
		logError(parseLabel, "window error", map[string]interface{}{
			"message": valueOrEmpty(parseEvent.Get("message").String()),
			"source":  valueOrEmpty(parseEvent.Get("filename").String()),
			"line":    safeNumber(parseEvent.Get("lineno")),
			"column":  safeNumber(parseEvent.Get("colno")),
		})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseWindow.Call("addEventListener", "error", parseHandler)
}

func installUnhandledRejectionLogger(parseLabel string, parseWindow js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if len(parseArgs) == 0 {
			return nil
		}
		parseEvent := parseArgs[0]
		logError(parseLabel, "unhandled rejection", map[string]interface{}{
			"reason": valueOrEmpty(fmt.Sprint(parseEvent.Get("reason"))),
		})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseWindow.Call("addEventListener", "unhandledrejection", parseHandler)
}

func installNavigationLogger(parseLabel string, parseWindow js.Value) {
	parseHashHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		logInfo(parseLabel, "hash navigation", map[string]interface{}{"path": pathValue(parseWindow)})
		return nil
	})
	parsePopHandler := js.FuncOf(func(parseThis2 js.Value, parseArgs2 []js.Value) interface{} {
		logInfo(parseLabel, "history navigation", map[string]interface{}{"path": pathValue(parseWindow)})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHashHandler, parsePopHandler)
	parseWindow.Call("addEventListener", "hashchange", parseHashHandler)
	parseWindow.Call("addEventListener", "popstate", parsePopHandler)
}

func installMountObserver(parseLabel string, parseDocument js.Value) {
	parseObserverCtor := js.Global().Get("MutationObserver")
	if !parseObserverCtor.Truthy() {
		return
	}
	parseSelector := "#app"
	parseEnv, parseErr := interop.GetWindowEnv()
	if parseErr == nil {
		parseSelector = parseEnv.String("__gwcExampleMountSelector", parseSelector)
	}

	// Follow the same host-provided selector the example runtime uses so embedded
	// mounts and standalone mounts log the correct render surface.
	parseRoot := parseDocument.Call("querySelector", parseSelector)
	if !parseRoot.Truthy() {
		parseRoot = parseDocument.Get("body")
	}
	if !parseRoot.Truthy() {
		return
	}

	isParseMountedLogged := false
	parseCallback := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		if isParseMountedLogged {
			return nil
		}
		parseText := strings.TrimSpace(parseRoot.Get("textContent").String())
		parseChildCount := parseRoot.Get("childElementCount").Int()
		if parseText == "" && parseChildCount == 0 {
			return nil
		}
		isParseMountedLogged = true
		logInfo(parseLabel, "render surface ready", map[string]interface{}{
			"target":      elementTag(parseRoot),
			"childCount":  parseChildCount,
			"textPreview": preview(parseText),
		})
		parseThis.Call("disconnect")
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseCallback)
	parseObserver := parseObserverCtor.New(parseCallback)
	parseConfig := js.Global().Get("Object").New()
	parseConfig.Set("childList", true)
	parseConfig.Set("subtree", true)
	parseConfig.Set("characterData", true)
	parseObserver.Call("observe", parseRoot, parseConfig)
}

func installReadyStateLogger(parseLabel string, parseDocument js.Value) {
	parseState := parseDocument.Get("readyState").String()
	logInfo(parseLabel, "document state", map[string]interface{}{"state": valueOrEmpty(parseState)})
	if parseDocument.Get("addEventListener").Type() != js.TypeFunction {
		return
	}
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		logInfo(parseLabel, "document ready", map[string]interface{}{"state": valueOrEmpty(parseDocument.Get("readyState").String())})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseDocument.Call("addEventListener", "DOMContentLoaded", parseHandler)
}

func installVisibilityLogger(parseLabel string, parseDocument js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		logInfo(parseLabel, "visibility changed", map[string]interface{}{"state": valueOrEmpty(parseDocument.Get("visibilityState").String())})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseDocument.Call("addEventListener", "visibilitychange", parseHandler)
}

func installResizeLogger(parseLabel string, parseWindow js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		logInfo(parseLabel, "window resized", map[string]interface{}{
			"width":  safeNumber(parseWindow.Get("innerWidth")),
			"height": safeNumber(parseWindow.Get("innerHeight")),
		})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseWindow.Call("addEventListener", "resize", parseHandler)
}

func installBeforeUnloadLogger(parseLabel string, parseWindow js.Value) {
	parseHandler := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) interface{} {
		logInfo(parseLabel, "page unloading", map[string]interface{}{"path": pathValue(parseWindow)})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, parseHandler)
	parseWindow.Call("addEventListener", "beforeunload", parseHandler)
}

func eventTarget(parseArgs []js.Value) js.Value {
	if len(parseArgs) == 0 {
		return js.Null()
	}
	parseEvent := parseArgs[0]
	if !parseEvent.Truthy() {
		return js.Null()
	}
	return parseEvent.Get("target")
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

func targetDetails(parseTarget js.Value, isIncludeValue bool) map[string]interface{} {
	if !parseTarget.Truthy() {
		return map[string]interface{}{}
	}
	parseDetails := map[string]interface{}{
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
		parseValue := parseTarget.Get("value").String()
		if strings.EqualFold(parseTarget.Get("type").String(), "password") {
			parseValue = "[redacted]"
		}
		parseDetails["value"] = preview(strings.TrimSpace(parseValue))
	}
	return parseDetails
}

func elementTag(parseTarget js.Value) string {
	if !parseTarget.Truthy() {
		return ""
	}
	parseTag := strings.TrimSpace(parseTarget.Get("tagName").String())
	return strings.ToLower(parseTag)
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

func logInfo(parseLabel, parseMessage string, parseDetails map[string]interface{}) {
	logWithMethod("log", parseLabel, parseMessage, parseDetails)
}

func logError(parseLabel, parseMessage string, parseDetails map[string]interface{}) {
	logWithMethod("error", parseLabel, parseMessage, parseDetails)
}

func logWithMethod(parseMethod, parseLabel, parseMessage string, parseDetails map[string]interface{}) {
	parseConsole := js.Global().Get("console")
	if !parseConsole.Truthy() {
		return
	}
	parsePrefix := fmt.Sprintf("[GoWebComponents example][Go] %s", parseLabel)
	if len(parseDetails) == 0 {
		parseConsole.Call(parseMethod, parsePrefix, parseMessage)
		return
	}
	parseConsole.Call(parseMethod, parsePrefix, parseMessage, toObject(parseDetails))
}

func toObject(parseDetails map[string]interface{}) js.Value {
	parseObj := js.Global().Get("Object").New()
	for parseKey, parseValue := range parseDetails {
		switch parseTyped := parseValue.(type) {
		case string:
			if parseTyped != "" {
				parseObj.Set(parseKey, parseTyped)
			}
		case int:
			parseObj.Set(parseKey, parseTyped)
		case bool:
			parseObj.Set(parseKey, parseTyped)
		default:
			if parseValue != nil {
				parseObj.Set(parseKey, fmt.Sprint(parseValue))
			}
		}
	}
	return parseObj
}
