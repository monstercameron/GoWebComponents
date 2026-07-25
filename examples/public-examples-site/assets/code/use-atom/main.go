//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/state"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

const sharedCounterID = "catalog-state-use-atom-count"

func atomWriter() ui.Node {
	parseCount := state.UseAtom(sharedCounterID, 3)
	parseDecrement := ui.UseEvent(func() { parseCount.Update(func(parsePrevious int) int { return parsePrevious - 1 }) })
	parseIncrement := ui.UseEvent(func() { parseCount.Update(func(parsePrevious2 int) int { return parsePrevious2 + 1 }) })
	parseBoost := ui.UseEvent(func() { parseCount.Update(func(parsePrevious3 int) int { return parsePrevious3 + 5 }) })
	reset := ui.UseEvent(func() { parseCount.Set(3) })

	return shared.ExamplePanel("Write from one component",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This component initializes the atom and updates it with Set and Update helpers.")),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
			shared.ExampleStat("Current value", fmt.Sprintf("%d", parseCount.Get())),
			shared.ExampleStat("Last action target", "Shared global atom"),
		),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
			shared.ExampleButton("-1", parseDecrement),
			shared.ExampleButton("+1", parseIncrement),
			shared.ExampleButton("+5", parseBoost),
			shared.ExampleButton("Reset", reset),
		),
	)
}

func atomReader() ui.Node {
	parseCount := state.UseAtom(sharedCounterID, 0)
	parseParity := "Odd"
	if parseCount.Get()%2 == 0 {
		parseParity = "Even"
	}

	return shared.ExamplePanel("Read from another component",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This panel never receives props. It subscribes to the same atom by ID and re-renders automatically.")),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
			shared.ExampleStat("Mirrored value", fmt.Sprintf("%d", parseCount.Get())),
			shared.ExampleStat("Parity", parseParity),
		),
		shared.ExampleCode(
			`count := state.UseAtom("catalog-state-use-atom-count", 3)`,
			`count.Update(func(previous int) int { return previous + 1 })`,
		),
	)
}

func useAtomExample() ui.Node {
	return shared.ExamplePage(
		"state.UseAtom",
		"Share global state by stable atom ID",
		"Atoms live outside the local component tree, so any component can subscribe to the same value and update it without prop threading.",
		ui.CreateElement(atomWriter),
		ui.CreateElement(atomReader),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(useAtomExample))
	exampleboot.WaitExampleRuntime()
}
