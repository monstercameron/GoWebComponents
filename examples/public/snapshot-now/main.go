//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v4/devtools"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func snapshotNowExample() ui.Node {
	parseCount := ui.UseState(0)
	parseSummary := ui.UseState("Press capture to take an immediate snapshot.")
	parseCapture := ui.UseEvent(func() {
		parseSnapshot := devtools.SnapshotNow()
		parseSummary.Set(fmt.Sprintf("route=%s fibers=%d diagnostics=%d", parseSnapshot.Route.Path, parseSnapshot.Stats.TotalFibers, len(parseSnapshot.Diagnostics)))
	})

	return shared.ExamplePage(
		"devtools.SnapshotNow",
		"Capture the current inspection snapshot on demand",
		"SnapshotNow is the imperative path. It is useful from buttons, debug drawers, or one-off diagnostics where you do not want a polling subscription.",
		shared.ExamplePanel("Imperative capture",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Increment state", ui.UseEvent(func() { parseCount.Update(func(parsePrevious int) int { return parsePrevious + 1 }) })),
				shared.ExampleButton("Capture snapshot now", parseCapture),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Local state", fmt.Sprintf("%d", parseCount.Get())),
				shared.ExampleStat("Capture mode", "Imperative"),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(parseSummary.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(snapshotNowExample))
	exampleboot.WaitExampleRuntime()
}
