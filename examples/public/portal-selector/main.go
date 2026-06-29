//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

const selectorPortalRoot = "#catalog-selector-portal-root"

func portalSelectorExample() ui.Node {
	parseOpen := ui.UseState(false)
	parseShow := ui.UseEvent(func() { parseOpen.Set(true) })
	parseClose := ui.UseEvent(func() { parseOpen.Set(false) })

	parsePage := shared.ExamplePage(
		"ui.Portal selector target",
		"Render an overlay into a DOM node found by selector",
		"Selector-based portals are the common case for modals, toasts, and overlays that belong outside the current app container but still share app state.",
		shared.ExamplePanel("App tree",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The button below lives inside the example mount root. The overlay renders into a sibling portal root resolved from a selector string.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Open selector portal", parseShow),
				shared.ExampleButton("Close portal", parseClose),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Target selector", selectorPortalRoot),
				shared.ExampleStat("Portal open", map[bool]string{true: "Yes", false: "No"}[parseOpen.Get()]),
			),
		),
	)

	if !parseOpen.Get() {
		return parsePage
	}

	return ui.Fragment(
		parsePage,
		ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: selectorPortalRoot},
			Child: html.Div(html.Props{Class: "fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-6"},
				html.Div(html.Props{Role: "dialog", Class: "w-full max-w-lg rounded-[2rem] border border-cyan-400/20 bg-slate-950 p-8 text-slate-100 shadow-2xl"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Selector Portal")),
					html.H2(html.Props{Class: "mt-4 text-3xl font-black"}, html.Text("Mounted outside the example root")),
					html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("The runtime found the host using the selector string and mounted this subtree there.")),
					html.Div(html.Props{Class: "mt-6 flex gap-3"}, shared.ExampleButton("Close overlay", parseClose)),
				),
			),
		}),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(portalSelectorExample))
	exampleboot.WaitExampleRuntime()
}
