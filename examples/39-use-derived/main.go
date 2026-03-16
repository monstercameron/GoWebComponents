//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const focusMinutesID = "catalog-state-derived-focus-minutes"

func derivedSourcePanel() ui.Node {
	minutes := state.UseAtom(focusMinutesID, 45)
	shortSprint := ui.UseEvent(func() { minutes.Set(30) })
	deepWork := ui.UseEvent(func() { minutes.Set(90) })
	addQuarterHour := ui.UseEvent(func() { minutes.Update(func(previous int) int { return previous + 15 }) })

	return shared.ExamplePanel("Source atom",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The source atom is writable. The derived atom in the next panel stays read-only and recomputes when this ID changes.")),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
			shared.ExampleStat("Focus block", fmt.Sprintf("%d min", minutes.Get())),
			shared.ExampleStat("Source atom ID", focusMinutesID),
		),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
			shared.ExampleButton("30 min", shortSprint),
			shared.ExampleButton("90 min", deepWork),
			shared.ExampleButton("+15 min", addQuarterHour),
		),
	)
}

func derivedReaderPanel() ui.Node {
	minutes := state.UseAtom(focusMinutesID, 0)
	band := state.UseDerived("catalog-state-derived-focus-band", func() string {
		switch {
		case minutes.Get() >= 90:
			return "Deep work"
		case minutes.Get() >= 60:
			return "Sustained focus"
		default:
			return "Warm-up"
		}
	}, focusMinutesID)

	return shared.ExamplePanel("Derived atom",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("UseDerived registers a shared read-only atom by ID. Consumers read the computed value, but only the source atom can be written.")),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
			shared.ExampleStat("Focus band", band.Get()),
			shared.ExampleStat("Observed minutes", fmt.Sprintf("%d", minutes.Get())),
		),
		shared.ExampleCode(
			`band := state.UseDerived("catalog-state-derived-focus-band", func() string { ... }, "catalog-state-derived-focus-minutes")`,
			`label := band.Get()`,
		),
	)
}

func useDerivedExample() ui.Node {
	return shared.ExamplePage(
		"state.UseDerived",
		"Register shared read-only atoms from source IDs",
		"Derived atoms turn one or more source atom IDs into a stable computed value that multiple components can subscribe to without owning the write path.",
		ui.CreateElement(derivedSourcePanel),
		ui.CreateElement(derivedReaderPanel),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(useDerivedExample), "#app")
	select {}
}