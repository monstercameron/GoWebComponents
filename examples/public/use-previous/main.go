//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func usePreviousExample() ui.Node {
	parseQuery := ui.UseState("")
	parsePrevious := ui.UsePrevious(parseQuery.Get())
	parseUpdate := ui.UseEvent(func(parseEvent ui.InputEvent) {
		parseQuery.Set(parseEvent.GetValue())
	})

	parsePreviousLabel := "No previous committed value yet"
	if parsePrevious.Ok() {
		parsePreviousLabel = parsePrevious.Get()
	}

	return shared.ExamplePage(
		"ui.UsePrevious",
		"Compare the current value with the previous committed one",
		"UsePrevious is useful for render-time comparisons, change detection, and transition messaging without creating extra state of your own.",
		shared.ExamplePanel("Current versus previous",
			html.Input(html.Props{Value: parseQuery.Get(), OnInput: parseUpdate, Placeholder: "Type to compare values", Class: "mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current", parseQuery.Get()),
				shared.ExampleStat("Previous", parsePreviousLabel),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(usePreviousExample))
	exampleboot.WaitExampleRuntime()
}
