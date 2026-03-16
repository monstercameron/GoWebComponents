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

func rawHandlerExample() ui.Node {
	wrappedCount := ui.UseState(0)
	rawCount := ui.UseState(0)
	lastMode := ui.UseState("No clicks yet")

	wrapped := ui.UseEvent(func() {
		wrappedCount.Update(func(previous int) int { return previous + 1 })
		lastMode.Set("ui.UseEvent")
	})
	raw := ui.RawHandler(func() {
		rawCount.Update(func(previous int) int { return previous + 1 })
		lastMode.Set("ui.RawHandler")
	})

	return shared.ExamplePage(
		"ui.RawHandler",
		"Forward a raw function value instead of using the event hook wrapper",
		"RawHandler exists for edge cases and interop. The preferred default is still ui.UseEvent, because it tracks the handler through the hook system and matches the rest of the public event model.",
		shared.ExamplePanel("Wrapped versus raw",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Both buttons work. The difference is API intent: UseEvent is the normal path, RawHandler is the escape hatch when you already have a function value you need to pass through directly.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Increment via ui.UseEvent", wrapped),
				html.Button(html.Props{OnClick: raw, Class: "rounded-full border border-amber-500/40 bg-amber-500/10 px-5 py-3 font-semibold text-amber-100 hover:bg-amber-500/20"}, html.Text("Increment via ui.RawHandler")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Wrapped clicks", fmt.Sprintf("%d", wrappedCount.Get())),
				shared.ExampleStat("Raw clicks", fmt.Sprintf("%d", rawCount.Get())),
				shared.ExampleStat("Last path", lastMode.Get()),
			),
			shared.ExampleCode(
				`preferred := ui.UseEvent(func() { ... })`,
				`escapeHatch := ui.RawHandler(func() { ... })`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(rawHandlerExample), "#app")
	select {}
}