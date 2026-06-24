//go:build js && wasm

package ui

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// defaultBindGlobalEvent attaches a managed listener to document/window and
// returns an unbind that removes it and releases the js.Func — so the callback is
// never leaked. Mirrors the established overlay/focus-trap cleanup pattern.
func defaultBindGlobalEvent(parseScope globalScope, parseEventType string, parseHandler func(Event)) func() {
	parseTarget := globalEventTargetValue(parseScope)
	if !parseTarget.Truthy() {
		return nil
	}
	// Guard against a global without addEventListener (no-DOM-event hosts) so we
	// degrade to a no-op instead of throwing — and never create a js.Func we
	// couldn't bind. Real browsers and Web Workers always have it.
	if parseTarget.Get("addEventListener").Type() != js.TypeFunction {
		return nil
	}
	parseListener := js.FuncOf(func(parseThis js.Value, parseArgs []js.Value) any {
		defer runtime.RecoverContainedPanic("ui", "UseGlobalEvent callback")
		var parseEvent js.Value
		if len(parseArgs) > 0 {
			parseEvent = parseArgs[0]
		}
		parseHandler(runtime.NewGoEvent(parseEvent))
		return nil
	})
	parseTarget.Call("addEventListener", parseEventType, parseListener)
	return func() {
		parseTarget.Call("removeEventListener", parseEventType, parseListener)
		parseListener.Release()
	}
}

func globalEventTargetValue(parseScope globalScope) js.Value {
	if parseScope == scopeWindow {
		return js.Global()
	}
	return js.Global().Get("document")
}
