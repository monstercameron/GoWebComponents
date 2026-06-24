//go:build js && wasm

package ui

import "syscall/js"

// defaultReadOnlineStatus reads navigator.onLine, defaulting to online when the
// value is unavailable.
func defaultReadOnlineStatus() bool {
	parseNavigator := js.Global().Get("navigator")
	if !parseNavigator.Truthy() {
		return true
	}
	parseOnLine := parseNavigator.Get("onLine")
	if parseOnLine.Type() != js.TypeBoolean {
		return true
	}
	return parseOnLine.Bool()
}
