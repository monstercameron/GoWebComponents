//go:build js && wasm

package ui

import "syscall/js"

// applyThemeAttribute sets (or clears) the data-theme attribute on the document
// root element, guarding against a missing document (Web Worker / no-DOM host).
func applyThemeAttribute(parseName string) {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseRoot := parseDocument.Get("documentElement")
	if !parseRoot.Truthy() {
		return
	}
	if parseName == "" {
		parseRoot.Call("removeAttribute", "data-theme")
		return
	}
	parseRoot.Call("setAttribute", "data-theme", parseName)
}
