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
	parseStatus := ui.UseState("init")

	// On mount, focus the referenced input via the ref's live node, then report
	// whether the ref resolved — proving placement published the node.
	ui.UseEffect(func() func() {
		if parseShown.Get() && parseField.Mounted() {
			parseField.Focus()
			parseStatus.Set("focused")
		}
		return nil
	}, parseShown.Get())

	parseArgs := []any{
		FromProps(Props{ID: "app-root"}),
		Button(
			FromProps(Props{ID: "toggle", Type: "button", OnClick: ui.UseEvent(func() { parseShown.Set(!parseShown.Get()) })}),
			"toggle",
		),
		Span(FromProps(Props{ID: "status"}), parseStatus.Get()),
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
