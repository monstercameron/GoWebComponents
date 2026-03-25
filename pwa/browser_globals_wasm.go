//go:build js && wasm
// +build js,wasm

package pwa

import "syscall/js"

func browserWindow() js.Value {
	parseBrowserWindow := js.Global().Get("window")
	if parseBrowserWindow.IsUndefined() || parseBrowserWindow.IsNull() {
		return js.Undefined()
	}
	return parseBrowserWindow
}

func browserNavigator() js.Value {
	if parseBrowserWindow := browserWindow(); !parseBrowserWindow.IsUndefined() && !parseBrowserWindow.IsNull() {
		parseBrowserNavigator := parseBrowserWindow.Get("navigator")
		if !parseBrowserNavigator.IsUndefined() && !parseBrowserNavigator.IsNull() {
			return parseBrowserNavigator
		}
	}
	parseGlobalNavigator := js.Global().Get("navigator")
	if parseGlobalNavigator.IsUndefined() || parseGlobalNavigator.IsNull() {
		return js.Undefined()
	}
	return parseGlobalNavigator
}
