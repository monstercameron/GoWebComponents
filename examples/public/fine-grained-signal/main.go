//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// ticksSignal lives outside the render function (a package var) because a signal is created once and
// shared, not minted per render. It updates only the DOM text nodes bound to it via signal.Text —
// the owning component never rerenders.
var ticksSignal = state.NewSignal(0)

// signalExample demonstrates state.Signal fine-grained reactivity: incrementing the signal updates
// the bound text node directly, with no component rerender. The render-count stat proves the
// component body does not re-run on each signal change.
func signalExample() ui.Node {
	parseRenders := ui.UseRef(0)
	parseRenders.Set(parseRenders.Get() + 1)

	parseIncrement := ui.UseEvent(func() {
		ticksSignal.Update(func(parsePrev int) int { return parsePrev + 1 })
	})
	parseReset := ui.UseEvent(func() {
		ticksSignal.Set(0)
	})

	return shared.ExamplePage(
		"state.Signal",
		"Fine-grained reactivity without component rerenders",
		"The signal updates only the text node bound through signal.Text — the component function does not re-run, so the render counter stays put while the live value changes. NewComputed derives from declared signal sources the same way.",
		shared.ExamplePanel("Live signal",
			html.P(html.Props{Class: "mt-3 text-2xl text-slate-100"},
				html.Text("ticks = "),
				ticksSignal.Text(func(parseValue int) string { return fmt.Sprintf("%d", parseValue) }),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-4"},
				shared.ExampleButton("Tick (updates signal only)", parseIncrement),
				shared.ExampleButton("Reset", parseReset),
				shared.ExampleStat("Component renders", fmt.Sprintf("%d", parseRenders.Get())),
			),
		),
	)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(signalExample))
	exampleboot.WaitExampleRuntime()
}
