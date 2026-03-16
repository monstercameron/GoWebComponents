//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/devtools"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func devtoolsPanelExample() ui.Node {
	count := ui.UseState(0)
	return ui.Fragment(
		shared.ExamplePage(
			"devtools.Panel",
			"Embed the runtime inspection overlay inside the app",
			"The panel is the all-in-one in-browser inspection surface: route state, runtime stats, diagnostics, profiling hotspots, and the committed component tree.",
			shared.ExamplePanel("Live app state",
				html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Use the button below, then open the devtools panel in the bottom-right corner to see the tree and hook state update.")),
				html.Div(html.Props{Class: "mt-6 flex gap-3"},
					shared.ExampleButton("Increment state", ui.UseEvent(func() { count.Update(func(previous int) int { return previous + 1 }) })),
					shared.ExampleStat("Count", ui.UseMemo(func() string { return htmlText(count.Get()) }, count.Get())),
				),
			),
		),
		ui.CreateElement(devtools.Panel, devtools.PanelProps{
			Title:           "Catalog Devtools",
			InitiallyOpen:   false,
			RefreshInterval: 500 * time.Millisecond,
			MaxDepth:        5,
		}),
	)
}

func htmlText(value int) string {
	return fmt.Sprintf("%d", value)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(devtoolsPanelExample), "#app")
	select {}
}
