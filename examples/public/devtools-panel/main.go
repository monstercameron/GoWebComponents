//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"time"

	"github.com/monstercameron/GoWebComponents/v5/devtools"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

func devtoolsPanelExample() ui.Node {
	parseCount := ui.UseState(0)
	return ui.Fragment(
		shared.ExamplePage(
			"devtools.Panel",
			"Embed the kernel-backed runtime inspection overlay inside the app",
			"The panel is the all-in-one in-browser inspection surface: route state, runtime stats, diagnostics, profiling hotspots, the committed component tree, and internal kernel-backed devtools sections.",
			shared.ExamplePanel("Live app state",
				html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Use the button below, then open the devtools panel in the bottom-right corner to see the tree and hook state update.")),
				html.Div(html.Props{Class: "mt-6 flex gap-3"},
					shared.ExampleButton("Increment state", ui.UseEvent(func() { parseCount.Update(func(parsePrevious int) int { return parsePrevious + 1 }) })),
					shared.ExampleStat("Count", ui.UseMemo(func() string { return htmlText(parseCount.Get()) }, parseCount.Get())),
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

func htmlText(parseValue int) string {
	return fmt.Sprintf("%d", parseValue)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(devtoolsPanelExample))
	exampleboot.WaitExampleRuntime()
}
