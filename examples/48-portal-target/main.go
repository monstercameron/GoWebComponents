//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const explicitPortalRootID = "catalog-explicit-portal-root"

func portalTargetExample() ui.Node {
	parseOpen := ui.UseState(false)
	parseShow := ui.UseEvent(func() { parseOpen.Set(true) })
	parseClose := ui.UseEvent(func() { parseOpen.Set(false) })
	parseHost := js.Global().Get("document").Call("getElementById", explicitPortalRootID)
	parseHostReady := parseHost.Truthy()

	parsePage := shared.ExamplePage(
		"ui.Portal explicit node target",
		"Render into a DOM host node you already resolved yourself",
		"Explicit node targets are useful when another integration point owns the host element and you want to hand that exact DOM node to the portal runtime instead of relying on selector lookup.",
		shared.ExamplePanel("Explicit target",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("This example resolves the host with document.getElementById and passes the node directly through ui.PortalTarget.Node.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Open explicit-node portal", parseShow),
				shared.ExampleButton("Close portal", parseClose),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Host resolved", map[bool]string{true: "Yes", false: "No"}[parseHostReady]),
				shared.ExampleStat("Portal open", map[bool]string{true: "Yes", false: "No"}[parseOpen.Get()]),
			),
		),
	)

	if !parseOpen.Get() || !parseHostReady {
		return parsePage
	}

	return ui.Fragment(
		parsePage,
		ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Node: parseHost},
			Child: html.Div(html.Props{Class: "fixed bottom-8 right-8 z-50 w-96 rounded-[2rem] border border-amber-400/20 bg-slate-950 p-6 text-slate-100 shadow-2xl"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-amber-300"}, html.Text("Explicit DOM Node")),
				html.H2(html.Props{Class: "mt-4 text-2xl font-black"}, html.Text("Targeted by node reference")),
				html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("This surface skipped selector lookup and mounted into the specific DOM node you passed to the portal target.")),
				html.Div(html.Props{Class: "mt-6 flex gap-3"}, shared.ExampleButton("Dismiss", parseClose)),
			),
		}),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(portalTargetExample), "#app")
	select {}
}
