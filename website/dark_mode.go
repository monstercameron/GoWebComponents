//go:build js && wasm
// +build js,wasm

package website

import (
	"syscall/js"
)

// initDarkModeCSS injects a simple CSS filter-based dark theme once per session.
// It uses the classic invert+hue-rotate trick which requires almost no changes to existing markup.
// Images/videos are double-inverted so they keep their natural colors.
func initDarkModeCSS() {
	if !js.Global().Get("document").Call("querySelector", "#gwc-darkmode-css").IsNull() {
		// Already injected
		return
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

// applyDarkClass sets or removes the "dark" class on <html>.
func applyDarkClass(enabled bool) {
	docEl := js.Global().Get("document").Get("documentElement")
	classList := docEl.Get("classList")
	if enabled {
		classList.Call("add", "dark")
	} else {
		classList.Call("remove", "dark")
	}
}

// getInitialDarkPref reads the saved preference (or system preference) to seed the state.
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
	// Fallback to system preference
	match := js.Global().Call("matchMedia", "(prefers-color-scheme: dark)")
	if !match.IsUndefined() && match.Get("matches").Bool() {
		return true
	}
	return false
}

// saveDarkPref persists the choice to localStorage
func saveDarkPref(enabled bool) {
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	if enabled {
		storage.Call("setItem", "gwc-theme", "dark")
	} else {
		storage.Call("setItem", "gwc-theme", "light")
	}
}
