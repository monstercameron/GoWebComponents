//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const selectorPortalRoot = "#catalog-selector-portal-root"

func portalSelectorExample() ui.Node {
	open := ui.UseState(false)
	show := ui.UseEvent(func() { open.Set(true) })
	close := ui.UseEvent(func() { open.Set(false) })

	page := shared.ExamplePage(
		"ui.Portal selector target",
		"Render an overlay into a DOM node found by selector",
		"Selector-based portals are the common case for modals, toasts, and overlays that belong outside the current app container but still share app state.",
		shared.ExamplePanel("App tree",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("The button below lives under #app. The overlay renders into a sibling portal root resolved from a selector string.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Open selector portal", show),
				shared.ExampleButton("Close portal", close),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Target selector", selectorPortalRoot),
				shared.ExampleStat("Portal open", map[bool]string{true: "Yes", false: "No"}[open.Get()]),
			),
		),
	)

	if !open.Get() {
		return page
	}

	return ui.Fragment(
		page,
		ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: selectorPortalRoot},
			Child: html.Div(html.Props{Class: "fixed inset-0 z-50 flex items-center justify-center bg-black/55 p-6"},
				html.Div(html.Props{Role: "dialog", Class: "w-full max-w-lg rounded-[2rem] border border-cyan-400/20 bg-slate-950 p-8 text-slate-100 shadow-2xl"},
					html.P(html.Props{Class: "text-xs uppercase tracking-[0.3em] text-cyan-300"}, html.Text("Selector Portal")),
					html.H2(html.Props{Class: "mt-4 text-3xl font-black"}, html.Text("Mounted outside #app")),
					html.P(html.Props{Class: "mt-4 leading-7 text-slate-300"}, html.Text("The runtime found the host using the selector string and mounted this subtree there.")),
					html.Div(html.Props{Class: "mt-6 flex gap-3"}, shared.ExampleButton("Close overlay", close)),
				),
			),
		}),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(portalSelectorExample), "#app")
	select {}
}
