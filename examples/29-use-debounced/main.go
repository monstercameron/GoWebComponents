//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func useDebouncedExample() ui.Node {
	query := ui.UseState("")
	debounced := ui.UseDebounced(query.Get(), 600*time.Millisecond)
	update := ui.UseEvent(func(event ui.InputEvent) { query.Set(event.GetValue()) })

	status := "Settled"
	if debounced.Pending() {
		status = "Waiting for inactivity"
	}

	return shared.ExamplePage(
		"ui.UseDebounced",
		"Delay updates until input settles",
		"Debounced values are useful for search, validation, and request-driven UIs where every keystroke should not trigger the next step immediately.",
		shared.ExamplePanel("Debounced input",
			html.Input(html.Props{Value: query.Get(), OnInput: update, Placeholder: "Type quickly to watch the debounced value lag behind", Class: "mt-3 w-full rounded-2xl border border-white/10 bg-slate-950/70 px-4 py-3 text-slate-100"}),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Immediate", query.Get()),
				shared.ExampleStat("Debounced", debounced.Get()),
				shared.ExampleStat("State", status),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useDebouncedExample), "#app")
	select {}
}
