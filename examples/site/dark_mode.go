//go:build js && wasm
// +build js,wasm

package website

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

	css := `
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

	js.Global().Get("document").Get("head").Call("insertAdjacentHTML", "beforeend", css)
}

// applyDarkClass toggles the "dark" class on the document root element.
// This controls the application of dark mode styles across the entire page.
func applyDarkClass(enabled bool) {
	docEl := js.Global().Get("document").Get("documentElement")
	classList := docEl.Get("classList")
	if enabled {
		classList.Call("add", "dark")
	} else {
		classList.Call("remove", "dark")
	}
}

// getInitialDarkPref determines the initial dark mode preference.
// Priority: 1) localStorage saved preference, 2) system preference via media query.
// Returns true for dark mode preference, false for light mode.
func getInitialDarkPref() bool {
	storage := js.Global().Get("localStorage")
	if !storage.IsUndefined() && !storage.IsNull() {
		pref := storage.Call("getItem", "gwc-theme").String()
		if pref == "dark" {
			return true
		}
		if pref == "light" {
			return false
		}
	}
	// Fallback to system preference detection
	match := js.Global().Call("matchMedia", "(prefers-color-scheme: dark)")
	if !match.IsUndefined() && match.Get("matches").Bool() {
		return true
	}
	return false
}

// saveDarkPref persists the dark mode preference to localStorage.
// Enables preference retention across browser sessions.
func saveDarkPref(enabled bool) {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return // localStorage not available
	}
	if enabled {
		storage.Call("setItem", "gwc-theme", "dark")
	} else {
		storage.Call("setItem", "gwc-theme", "light")
	}
}

