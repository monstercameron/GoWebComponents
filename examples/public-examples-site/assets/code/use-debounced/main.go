//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func useDebouncedExample() ui.Node {
	parseQuery := ui.UseState("")
	parseDebounced := ui.UseDebounced(parseQuery.Get(), 600*time.Millisecond)
	parseUpdate := ui.UseEvent(func(parseEvent ui.InputEvent) { parseQuery.Set(parseEvent.GetValue()) })

	parseStatus := "Settled"
	if parseDebounced.Pending() {
		parseStatus = "Waiting for inactivity"
	}

	return shared.ExamplePage(
		"ui.UseDebounced",
		"Delay updates until input settles",
		"Debounced values are useful for search, validation, and request-driven UIs where every keystroke should not trigger the next step immediately.",
		shared.ExamplePanel("Debounced input",
			html.Input(html.Props{Value: parseQuery.Get(), OnInput: parseUpdate, Placeholder: "Type quickly to watch the debounced value lag behind", Class: "mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Immediate", parseQuery.Get()),
				shared.ExampleStat("Debounced", parseDebounced.Get()),
				shared.ExampleStat("State", parseStatus),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(useDebouncedExample))
	exampleboot.WaitExampleRuntime()
}
