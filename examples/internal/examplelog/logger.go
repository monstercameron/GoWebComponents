//go:build js && wasm
// +build js,wasm

package examplelog

import (
	"fmt"
	"strings"
	"syscall/js"
)

var retainedCallbacks []js.Func

func init() {
	window := js.Global().Get("window")
	document := js.Global().Get("document")
	if !window.Truthy() || !document.Truthy() {
		return
	}

	label := deriveLabel(window, document)
	logInfo(label, "logger attached", map[string]interface{}{
		"path": pathValue(window),
		"hash": valueOrEmpty(window.Get("location").Get("hash").String()),
	})

	installClickLogger(label, document)
	installChangeLogger(label, document)
	installSubmitLogger(label, document)
	installWindowErrorLogger(label, window)
	installNavigationLogger(label, window)
	installMountObserver(label, document)
	installReadyStateLogger(label, document)
	installUnhandledRejectionLogger(label, window)
	installVisibilityLogger(label, document)
	installResizeLogger(label, window)
	installBeforeUnloadLogger(label, window)
	logInfo(label, "logger ready", nil)
}

func deriveLabel(window, document js.Value) string {
	title := strings.TrimSpace(document.Get("title").String())
	if title != "" {
		return title
	}
	path := pathValue(window)
	if path != "" {
		return path
	}
	return "unknown example"
}

func pathValue(window js.Value) string {
	location := window.Get("location")
	if !location.Truthy() {
		return ""
	}
	path := strings.TrimSpace(location.Get("pathname").String())
	hash := strings.TrimSpace(location.Get("hash").String())
	if hash != "" {
		return path + hash
	}
	return path
}

func installClickLogger(label string, document js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		target := eventTarget(args)
		candidate := closestActionTarget(target)
		if !candidate.Truthy() {
			return nil
		}
		logInfo(label, "interaction", targetDetails(candidate, false))
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	document.Call("addEventListener", "click", handler, true)
}

func installChangeLogger(label string, document js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		target := eventTarget(args)
		if !isFormField(target) {
			return nil
		}
		logInfo(label, "field change", targetDetails(target, true))
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	document.Call("addEventListener", "change", handler, true)
}

func installSubmitLogger(label string, document js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		target := eventTarget(args)
		if !target.Truthy() {
			return nil
		}
		logInfo(label, "form submit", targetDetails(target, false))
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	document.Call("addEventListener", "submit", handler, true)
}

func installWindowErrorLogger(label string, window js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		logError(label, "window error", map[string]interface{}{
			"message": valueOrEmpty(event.Get("message").String()),
			"source":  valueOrEmpty(event.Get("filename").String()),
			"line":    safeNumber(event.Get("lineno")),
			"column":  safeNumber(event.Get("colno")),
		})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	window.Call("addEventListener", "error", handler)
}

func installUnhandledRejectionLogger(label string, window js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		event := args[0]
		logError(label, "unhandled rejection", map[string]interface{}{
			"reason": valueOrEmpty(fmt.Sprint(event.Get("reason"))),
		})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	window.Call("addEventListener", "unhandledrejection", handler)
}

func installNavigationLogger(label string, window js.Value) {
	hashHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		logInfo(label, "hash navigation", map[string]interface{}{"path": pathValue(window)})
		return nil
	})
	popHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		logInfo(label, "history navigation", map[string]interface{}{"path": pathValue(window)})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, hashHandler, popHandler)
	window.Call("addEventListener", "hashchange", hashHandler)
	window.Call("addEventListener", "popstate", popHandler)
}

func installMountObserver(label string, document js.Value) {
	observerCtor := js.Global().Get("MutationObserver")
	if !observerCtor.Truthy() {
		return
	}
	root := document.Call("querySelector", "#app")
	if !root.Truthy() {
		root = document.Get("body")
	}
	if !root.Truthy() {
		return
	}

	mountedLogged := false
	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if mountedLogged {
			return nil
		}
		text := strings.TrimSpace(root.Get("textContent").String())
		childCount := root.Get("childElementCount").Int()
		if text == "" && childCount == 0 {
			return nil
		}
		mountedLogged = true
		logInfo(label, "render surface ready", map[string]interface{}{
			"target":      elementTag(root),
			"childCount":  childCount,
			"textPreview": preview(text),
		})
		this.Call("disconnect")
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, callback)
	observer := observerCtor.New(callback)
	config := js.Global().Get("Object").New()
	config.Set("childList", true)
	config.Set("subtree", true)
	config.Set("characterData", true)
	observer.Call("observe", root, config)
}

func installReadyStateLogger(label string, document js.Value) {
	state := document.Get("readyState").String()
	logInfo(label, "document state", map[string]interface{}{"state": valueOrEmpty(state)})
	if document.Get("addEventListener").Type() != js.TypeFunction {
		return
	}
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		logInfo(label, "document ready", map[string]interface{}{"state": valueOrEmpty(document.Get("readyState").String())})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	document.Call("addEventListener", "DOMContentLoaded", handler)
}

func installVisibilityLogger(label string, document js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		logInfo(label, "visibility changed", map[string]interface{}{"state": valueOrEmpty(document.Get("visibilityState").String())})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	document.Call("addEventListener", "visibilitychange", handler)
}

func installResizeLogger(label string, window js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		logInfo(label, "window resized", map[string]interface{}{
			"width":  safeNumber(window.Get("innerWidth")),
			"height": safeNumber(window.Get("innerHeight")),
		})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	window.Call("addEventListener", "resize", handler)
}

func installBeforeUnloadLogger(label string, window js.Value) {
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		logInfo(label, "page unloading", map[string]interface{}{"path": pathValue(window)})
		return nil
	})
	retainedCallbacks = append(retainedCallbacks, handler)
	window.Call("addEventListener", "beforeunload", handler)
}

func eventTarget(args []js.Value) js.Value {
	if len(args) == 0 {
		return js.Null()
	}
	event := args[0]
	if !event.Truthy() {
		return js.Null()
	}
	return event.Get("target")
}

func closestActionTarget(target js.Value) js.Value {
	if !target.Truthy() {
		return js.Null()
	}
	closest := target.Get("closest")
	if closest.Type() == js.TypeFunction {
		candidate := target.Call("closest", "button, a, [role='button']")
		if candidate.Truthy() {
			return candidate
		}
	}
	return target
}

func isFormField(target js.Value) bool {
	if !target.Truthy() {
		return false
	}
	tag := elementTag(target)
	return tag == "input" || tag == "select" || tag == "textarea"
}

func targetDetails(target js.Value, includeValue bool) map[string]interface{} {
	if !target.Truthy() {
		return map[string]interface{}{}
	}
	details := map[string]interface{}{
		"tag":  valueOrEmpty(elementTag(target)),
		"id":   valueOrEmpty(target.Get("id").String()),
		"name": valueOrEmpty(target.Get("name").String()),
		"text": preview(strings.TrimSpace(target.Get("textContent").String())),
	}
	if href := valueOrEmpty(target.Get("href").String()); href != "" {
		details["href"] = href
	}
	if role := valueOrEmpty(target.Get("role").String()); role != "" {
		details["role"] = role
	}
	if kind := valueOrEmpty(target.Get("type").String()); kind != "" {
		details["type"] = kind
	}
	if includeValue {
		value := target.Get("value").String()
		if strings.EqualFold(target.Get("type").String(), "password") {
			value = "[redacted]"
		}
		details["value"] = preview(strings.TrimSpace(value))
	}
	return details
}

func elementTag(target js.Value) string {
	if !target.Truthy() {
		return ""
	}
	tag := strings.TrimSpace(target.Get("tagName").String())
	return strings.ToLower(tag)
}

func preview(value string) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if len(trimmed) > 120 {
		return trimmed[:120]
	}
	return trimmed
}

func valueOrEmpty(value string) string {
	return strings.TrimSpace(value)
}

func safeNumber(value js.Value) int {
	if !value.Truthy() {
		return 0
	}
	return value.Int()
}

func logInfo(label, message string, details map[string]interface{}) {
	logWithMethod("log", label, message, details)
}

func logError(label, message string, details map[string]interface{}) {
	logWithMethod("error", label, message, details)
}

func logWithMethod(method, label, message string, details map[string]interface{}) {
	console := js.Global().Get("console")
	if !console.Truthy() {
		return
	}
	prefix := fmt.Sprintf("[GoWebComponents example][Go] %s", label)
	if len(details) == 0 {
		console.Call(method, prefix, message)
		return
	}
	console.Call(method, prefix, message, toObject(details))
}

func toObject(details map[string]interface{}) js.Value {
	obj := js.Global().Get("Object").New()
	for key, value := range details {
		switch typed := value.(type) {
		case string:
			if typed != "" {
				obj.Set(key, typed)
			}
		case int:
			obj.Set(key, typed)
		case bool:
			obj.Set(key, typed)
		default:
			if value != nil {
				obj.Set(key, fmt.Sprint(value))
			}
		}
	}
	return obj
}
