//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func usePreviousExample() ui.Node {
	query := ui.UseState("")
	previous := ui.UsePrevious(query.Get())
	update := ui.UseEvent(func(event ui.InputEvent) {
		query.Set(event.GetValue())
	})

	previousLabel := "No previous committed value yet"
	if previous.Ok() {
		previousLabel = previous.Get()
	}

	return shared.ExamplePage(
		"ui.UsePrevious",
		"Compare the current value with the previous committed one",
		"UsePrevious is useful for render-time comparisons, change detection, and transition messaging without creating extra state of your own.",
		shared.ExamplePanel("Current versus previous",
			html.Input(html.Props{Value: query.Get(), OnInput: update, Placeholder: "Type to compare values", Class: "mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current", query.Get()),
				shared.ExampleStat("Previous", previousLabel),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(usePreviousExample), "#app")
	select {}
}
