//go:build js && wasm
// +build js,wasm

package pwa

import "syscall/js"

func browserWindow() js.Value {
	parseWindow := js.Global().Get("window")
	if parseWindow.IsUndefined() || parseWindow.IsNull() {
		return js.Undefined()
	}
	return parseWindow
}

func browserNavigator() js.Value {
	if parseWindow := browserWindow(); !parseWindow.IsUndefined() && !parseWindow.IsNull() {
		parseNavigator := parseWindow.Get("navigator")
		if !parseNavigator.IsUndefined() && !parseNavigator.IsNull() {
			return parseNavigator
		}
	}
	parseNavigator2 := js.Global().Get("navigator")
	if parseNavigator2.IsUndefined() || parseNavigator2.IsNull() {
		return js.Undefined()
	}
	return parseNavigator2
}
