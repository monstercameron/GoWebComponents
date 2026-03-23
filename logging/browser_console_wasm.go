//go:build js && wasm
// +build js,wasm

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
func AttachBrowserConsole(options BrowserConsoleOptions) func() {
	window := js.Global().Get("window")
	document := js.Global().Get("document")
	if !window.Truthy() || !document.Truthy() {
		return func() {}
	}

	resolved := normalizeBrowserConsoleOptions(options)
	logger := New(resolveScope(resolved.Scope, window, document))
	logger.Info("browser console attached", Fields{
		"path": pathValue(window),
		"hash": valueOrEmpty(window.Get("location").Get("hash").String()),
	})

	cleanups := make([]func(), 0, 12)
	if resolved.LogClicks {
		cleanups = append(cleanups, addEventListener(document, "click", true, func(args []js.Value) {
			target := eventTarget(args)
			candidate := closestActionTarget(target)
			if !candidate.Truthy() {
				return
			}
			logger.Info("interaction", targetDetails(candidate, false))
		}))
	}
	if resolved.LogChanges {
		cleanups = append(cleanups, addEventListener(document, "change", true, func(args []js.Value) {
			target := eventTarget(args)
			if !isFormField(target) {
				return
			}
			logger.Info("field change", targetDetails(target, true))
		}))
	}
	if resolved.LogSubmits {
		cleanups = append(cleanups, addEventListener(document, "submit", true, func(args []js.Value) {
			target := eventTarget(args)
			if !target.Truthy() {
				return
			}
			logger.Info("form submit", targetDetails(target, false))
		}))
	}
	if resolved.LogWindowErrors {
		cleanups = append(cleanups, addEventListener(window, "error", false, func(args []js.Value) {
			if len(args) == 0 {
				return
			}
			event := args[0]
			logger.Error("window error", Fields{
				"message": valueOrEmpty(event.Get("message").String()),
				"source":  valueOrEmpty(event.Get("filename").String()),
				"line":    safeNumber(event.Get("lineno")),
				"column":  safeNumber(event.Get("colno")),
			})
		}))
	}
	if resolved.LogNavigation {
		cleanups = append(cleanups, addEventListener(window, "hashchange", false, func(args []js.Value) {
			logger.Info("hash navigation", Fields{"path": pathValue(window)})
		}))
		cleanups = append(cleanups, addEventListener(window, "popstate", false, func(args []js.Value) {
			logger.Info("history navigation", Fields{"path": pathValue(window)})
		}))
	}
	if resolved.LogMount {
		if cleanup := installMountObserver(logger, document); cleanup != nil {
			cleanups = append(cleanups, cleanup)
		}
	}
	if resolved.LogDocumentReady {
		logger.Info("document state", Fields{"state": valueOrEmpty(document.Get("readyState").String())})
		cleanups = append(cleanups, addEventListener(document, "DOMContentLoaded", false, func(args []js.Value) {
			logger.Info("document ready", Fields{"state": valueOrEmpty(document.Get("readyState").String())})
		}))
	}
	if resolved.LogUnhandledRejections {
		cleanups = append(cleanups, addEventListener(window, "unhandledrejection", false, func(args []js.Value) {
			if len(args) == 0 {
				return
			}
			event := args[0]
			logger.Error("unhandled rejection", Fields{
				"reason": valueOrEmpty(fmt.Sprint(event.Get("reason"))),
			})
		}))
	}
	if resolved.LogVisibility {
		cleanups = append(cleanups, addEventListener(document, "visibilitychange", false, func(args []js.Value) {
			logger.Info("visibility changed", Fields{"state": valueOrEmpty(document.Get("visibilityState").String())})
		}))
	}
	if resolved.LogResize {
		cleanups = append(cleanups, addEventListener(window, "resize", false, func(args []js.Value) {
			logger.Info("window resized", Fields{
				"width":  safeNumber(window.Get("innerWidth")),
				"height": safeNumber(window.Get("innerHeight")),
			})
		}))
	}
	if resolved.LogBeforeUnload {
		cleanups = append(cleanups, addEventListener(window, "beforeunload", false, func(args []js.Value) {
			logger.Info("page unloading", Fields{"path": pathValue(window)})
		}))
	}

	logger.Info("browser console ready", nil)
	return func() {
		for index := len(cleanups) - 1; index >= 0; index-- {
			cleanups[index]()
		}
	}
}

func normalizeBrowserConsoleOptions(options BrowserConsoleOptions) BrowserConsoleOptions {
	if options.LogClicks || options.LogChanges || options.LogSubmits || options.LogWindowErrors || options.LogNavigation || options.LogMount || options.LogDocumentReady || options.LogUnhandledRejections || options.LogVisibility || options.LogResize || options.LogBeforeUnload {
		return options
	}
	options.LogClicks = true
	options.LogChanges = true
	options.LogSubmits = true
	options.LogWindowErrors = true
	options.LogNavigation = true
	options.LogMount = true
	options.LogDocumentReady = true
	options.LogUnhandledRejections = true
	options.LogVisibility = true
	options.LogResize = true
	options.LogBeforeUnload = true
	return options
}

func resolveScope(scope string, window, document js.Value) string {
	trimmed := strings.TrimSpace(scope)
	if trimmed != "" {
		return trimmed
	}
	title := strings.TrimSpace(document.Get("title").String())
	if title != "" {
		return title
	}
	path := pathValue(window)
	if path != "" {
		return path
	}
	return "browser"
}

func addEventListener(target js.Value, event string, capture bool, callback func([]js.Value)) func() {
	if !target.Truthy() || target.Get("addEventListener").Type() != js.TypeFunction {
		return func() {}
	}
	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if callback != nil {
			callback(args)
		}
		return nil
	})
	if capture {
		target.Call("addEventListener", event, handler, true)
	} else {
		target.Call("addEventListener", event, handler)
	}
	return func() {
		if !target.Truthy() || target.Get("removeEventListener").Type() != js.TypeFunction {
			handler.Release()
			return
		}
		if capture {
			target.Call("removeEventListener", event, handler, true)
		} else {
			target.Call("removeEventListener", event, handler)
		}
		handler.Release()
	}
}

func installMountObserver(logger Logger, document js.Value) func() {
	observerCtor := js.Global().Get("MutationObserver")
	if !observerCtor.Truthy() {
		return nil
	}
	root := document.Call("querySelector", "#app")
	if !root.Truthy() {
		root = document.Get("body")
	}
	if !root.Truthy() {
		return nil
	}

	var observer js.Value
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
		logger.Info("render surface ready", Fields{
			"target":      elementTag(root),
			"childCount":  childCount,
			"textPreview": preview(text),
		})
		if observer.Truthy() {
			observer.Call("disconnect")
		}
		return nil
	})
	observer = observerCtor.New(callback)
	config := js.Global().Get("Object").New()
	config.Set("childList", true)
	config.Set("subtree", true)
	config.Set("characterData", true)
	observer.Call("observe", root, config)
	return func() {
		if observer.Truthy() {
			observer.Call("disconnect")
		}
		callback.Release()
	}
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

func eventTarget(args []js.Value) js.Value {
	if len(args) == 0 || !args[0].Truthy() {
		return js.Null()
	}
	return args[0].Get("target")
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

func targetDetails(target js.Value, includeValue bool) Fields {
	if !target.Truthy() {
		return Fields{}
	}
	details := Fields{
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
	return strings.ToLower(strings.TrimSpace(target.Get("tagName").String()))
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
