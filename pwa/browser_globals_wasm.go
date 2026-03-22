//go:build js && wasm
// +build js,wasm

package pwa

import "syscall/js"

func browserWindow() js.Value {
	window := js.Global().Get("window")
	if window.IsUndefined() || window.IsNull() {
		return js.Undefined()
	}
	return window
}

func browserNavigator() js.Value {
	if window := browserWindow(); !window.IsUndefined() && !window.IsNull() {
		navigator := window.Get("navigator")
		if !navigator.IsUndefined() && !navigator.IsNull() {
			return navigator
		}
	}
	navigator := js.Global().Get("navigator")
	if navigator.IsUndefined() || navigator.IsNull() {
		return js.Undefined()
	}
	return navigator
}
