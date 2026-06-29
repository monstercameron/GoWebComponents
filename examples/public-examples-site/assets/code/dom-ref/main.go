//go:build js && wasm

// Command dom-ref is the runnable e2e fixture for the G2 DOM-ref primitive
// (ui.UseDOMRef + shorthand.Ref). It focuses an input via the ref on mount, and
// can unmount it to prove the ref detaches — exercised by the playwright e2e.
package main

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

func App() ui.Node {
	parseShown := ui.UseState(true)
	parseField := ui.UseDOMRef()

	// G22: focus the referenced input whenever it is (re)shown — on initial mount
	// and each time it is revealed. No UseId()/getElementById, no autofocus attr.
	ui.UseAutoFocus(parseField, parseShown.Get())

	parseArgs := []any{
		FromProps(Props{ID: "app-root"}),
		Button(
			FromProps(Props{ID: "toggle", Type: "button", OnClick: ui.UseEvent(func() { parseShown.Set(!parseShown.Get()) })}),
			"toggle",
		),
	}
	if parseShown.Get() {
		parseArgs = append(parseArgs,
			Input(Ref(parseField), FromProps(Props{ID: "target", Type: "text"})),
		)
	}
	return Div(parseArgs...)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
