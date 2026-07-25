//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// ViewTransition runs apply (the DOM-changing callback, typically a state update
// that triggers a re-render) inside the browser's View Transitions API when
// available, animating between the before/after states; otherwise it calls apply
// directly (G27). Always safe to call — it degrades gracefully on browsers
// without startViewTransition.
//
//	ui.ViewTransition(func() { selected.Set(next) })
func ViewTransition(parseApply func()) {
	if parseApply == nil {
		return
	}
	parseDocument := js.Global().Get("document")
	if !parseDocument.Truthy() {
		parseApply()
		return
	}
	parseStart := parseDocument.Get("startViewTransition")
	// Honor prefers-reduced-motion: skip the animated transition and apply the change
	// directly when the user has requested reduced motion.
	if parseStart.Type() != js.TypeFunction || currentMediaMatch(prefersReducedMotionQuery) {
		parseApply()
		return
	}
	var parseCallback js.Func
	parseCallback = js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "ViewTransition apply")
		// Release after the transition callback fires (startViewTransition invokes
		// it synchronously to snapshot the DOM); releasing here avoids a leak.
		defer parseCallback.Release()
		parseApply()
		return nil
	})
	parseDocument.Call("startViewTransition", parseCallback)
}
