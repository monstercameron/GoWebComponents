//go:build js && wasm
// +build js,wasm

package main

import (
	"syscall/js"
)

// initDarkModeCSS injects CSS for filter-based dark theme implementation.
// Uses the classic invert+hue-rotate technique that requires minimal markup changes.
// Images and videos are double-inverted to preserve their natural appearance.
// Safe to call multiple times - includes duplicate injection guard.
func initDarkModeCSS() {
	if !js.Global().Get("document").Call("querySelector", "#gwc-darkmode-css").IsNull() {
		return // CSS already injected
	}

	parseCss := `
<style id="gwc-darkmode-css">
html.dark {
  filter: invert(1) hue-rotate(180deg);
  background-color: #0a0a0a;
  color-scheme: dark;
}
html.dark img, html.dark video, html.dark iframe, html.dark canvas {
  filter: invert(1) hue-rotate(180deg);
}
</style>`

	js.Global().Get("document").Get("head").Call("insertAdjacentHTML", "beforeend", parseCss)
}

// applyDarkClass toggles the "dark" class on the document root element.
// This controls the application of dark mode styles across the entire page.
func applyDarkClass(isEnabled bool) {
	parseDocEl := js.Global().Get("document").Get("documentElement")
	parseClassList := parseDocEl.Get("classList")
	if isEnabled {
		parseClassList.Call("add", "dark")
	} else {
		parseClassList.Call("remove", "dark")
	}
}

// getInitialDarkPref determines the initial dark mode preference.
// Priority: 1) localStorage saved preference, 2) system preference via media query.
// Returns true for dark mode preference, false for light mode.
func getInitialDarkPref() bool {
	parseStorage := js.Global().Get("localStorage")
	if !parseStorage.IsUndefined() && !parseStorage.IsNull() {
		parsePref := parseStorage.Call("getItem", "gwc-theme").String()
		if parsePref == "dark" {
			return true
		}
		if parsePref == "light" {
			return false
		}
	}
	// Fallback to system preference detection
	parseMatch := js.Global().Call("matchMedia", "(prefers-color-scheme: dark)")
	if !parseMatch.IsUndefined() && parseMatch.Get("matches").Bool() {
		return true
	}
	return false
}

// saveDarkPref persists the dark mode preference to localStorage.
// Enables preference retention across browser sessions.
func saveDarkPref(isEnabled bool) {
	parseStorage := js.Global().Get("localStorage")
	if parseStorage.IsUndefined() || parseStorage.IsNull() {
		return // localStorage not available
	}
	if isEnabled {
		parseStorage.Call("setItem", "gwc-theme", "dark")
	} else {
		parseStorage.Call("setItem", "gwc-theme", "light")
	}
}
