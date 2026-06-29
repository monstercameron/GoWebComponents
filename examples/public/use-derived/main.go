//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const focusMinutesID = "catalog-state-derived-focus-minutes"

func derivedSourcePanel() ui.Node {
	parseMinutes := state.UseAtom(focusMinutesID, 45)
	parseShortSprint := ui.UseEvent(func() { parseMinutes.Set(30) })
	parseDeepWork := ui.UseEvent(func() { parseMinutes.Set(90) })
	parseAddQuarterHour := ui.UseEvent(func() { parseMinutes.Update(func(parsePrevious int) int { return parsePrevious + 15 }) })

	return shared.ExamplePanel("Source atom",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The source atom is writable. The derived atom in the next panel stays read-only and recomputes when this ID changes.")),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
			shared.ExampleStat("Focus block", fmt.Sprintf("%d min", parseMinutes.Get())),
			shared.ExampleStat("Source atom ID", focusMinutesID),
		),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
			shared.ExampleButton("30 min", parseShortSprint),
			shared.ExampleButton("90 min", parseDeepWork),
			shared.ExampleButton("+15 min", parseAddQuarterHour),
		),
	)
}

func derivedReaderPanel() ui.Node {
	parseMinutes := state.UseAtom(focusMinutesID, 0)
	parseBand := state.UseDerived("catalog-state-derived-focus-band", func() string {
		switch {
		case parseMinutes.Get() >= 90:
			return "Deep work"
		case parseMinutes.Get() >= 60:
			return "Sustained focus"
		default:
			return "Warm-up"
		}
	}, focusMinutesID)

	return shared.ExamplePanel("Derived atom",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("UseDerived registers a shared read-only atom by ID. Consumers read the computed value, but only the source atom can be written.")),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
			shared.ExampleStat("Focus band", parseBand.Get()),
			shared.ExampleStat("Observed minutes", fmt.Sprintf("%d", parseMinutes.Get())),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(useDerivedExample))
	exampleboot.WaitExampleRuntime()
}
