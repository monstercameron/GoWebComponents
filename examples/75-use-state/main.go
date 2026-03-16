//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useStateExample() ui.Node {
	counter := ui.UseState(2)
	message := ui.UseState("Ship the feature-isolated catalog")

	increment := ui.UseEvent(func() {
		counter.Update(func(previous int) int { return previous + 1 })
	})
	decrement := ui.UseEvent(func() {
		counter.Update(func(previous int) int { return previous - 1 })
	})
	reset := ui.UseEvent(func() {
		counter.Set(2)
		message.Set("Ship the feature-isolated catalog")
	})
	setAlpha := ui.UseEvent(func() { message.Set("Audit every example page") })
	setBeta := ui.UseEvent(func() { message.Set("Document the implementation details") })

	return shared.ExamplePage(
		"ui.UseState",
		"Create local reactive state with typed get, set, and update helpers",
		"UseState is the core local state primitive. This example shows direct Set calls for replacing values and Update calls for deriving the next value from the previous state safely.",
		shared.ExamplePanel("State transitions",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The counter demonstrates numeric update flows, while the message state demonstrates direct replacement of a typed string value.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("-1", decrement),
				shared.ExampleButton("+1", increment),
				shared.ExampleButton("Reset", reset),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Counter", fmt.Sprintf("%d", counter.Get())),
				shared.ExampleStat("Message length", fmt.Sprintf("%d chars", len(message.Get()))),
			),
		),
		shared.ExamplePanel("String state",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Set audit message", setAlpha),
				shared.ExampleButton("Set docs message", setBeta),
			),
			html.Div(html.Props{Class: "mt-6 rounded-2xl border border-white/10 bg-slate-950/45 p-5"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text("Current message")),
				html.P(html.Props{Class: "mt-3 text-xl font-semibold text-white"}, html.Text(message.Get())),
			),
			shared.ExampleCode(
				"counter := ui.UseState(2)",
				"counter.Set(10)",
				"counter.Update(func(previous int) int { return previous + 1 })",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useStateExample), "#app")
	select {}
}