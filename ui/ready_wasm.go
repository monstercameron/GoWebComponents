//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// On wasm, dispatch a "gwc:ready" event on document after the first commit so a
// host page can drop its splash with no Go glue. Registered at init; it fires
// when the app's first render commits (or immediately if that already happened).
func init() {
	runtime.OnFirstCommit(dispatchReadyEvent)
}

func dispatchReadyEvent() {
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		return
	}
	parseCustomEvent := js.Global().Get("CustomEvent")
	if !parseCustomEvent.Truthy() {
		return
	}
	parseEvent := parseCustomEvent.New("gwc:ready")
	parseDocument.Call("dispatchEvent", parseEvent)
}
