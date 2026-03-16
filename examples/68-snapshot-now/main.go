//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func snapshotNowExample() ui.Node {
	count := ui.UseState(0)
	summary := ui.UseState("Press capture to take an immediate snapshot.")
	capture := ui.UseEvent(func() {
		snapshot := devtools.SnapshotNow()
		summary.Set(fmt.Sprintf("route=%s fibers=%d diagnostics=%d", snapshot.Route.Path, snapshot.Stats.TotalFibers, len(snapshot.Diagnostics)))
	})

	return shared.ExamplePage(
		"devtools.SnapshotNow",
		"Capture the current inspection snapshot on demand",
		"SnapshotNow is the imperative path. It is useful from buttons, debug drawers, or one-off diagnostics where you do not want a polling subscription.",
		shared.ExamplePanel("Imperative capture",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Increment state", ui.UseEvent(func() { count.Update(func(previous int) int { return previous + 1 }) })),
				shared.ExampleButton("Capture snapshot now", capture),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Local state", fmt.Sprintf("%d", count.Get())),
				shared.ExampleStat("Capture mode", "Imperative"),
			),
			html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(summary.Get())),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(snapshotNowExample), "#app")
	select {}
}
