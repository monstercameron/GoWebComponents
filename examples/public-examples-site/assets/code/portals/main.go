//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func App() ui.Node {
	parseShowModal := ui.UseState(false)
	parseShowTooltip := ui.UseState(false)
	parseShowPopover := ui.UseState(false)
	parseConfirmCount := ui.UseState(0)

	parseOverlay := make([]ui.Node, 0, 3)
	if parseShowModal.Get() {
		parseOverlay = append(parseOverlay, ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: "#portal-root"},
			Child: html.Div(html.Props{ID: "modal-surface", Class: "fixed inset-0 z-50 flex items-center justify-center bg-black/65 p-6"},
				html.Div(html.Props{Role: "dialog", Class: "w-full max-w-md rounded-3xl border border-white/10 bg-slate-950/95 p-6 text-slate-100 shadow-2xl"},
					html.H2(html.Props{Class: "text-2xl font-bold text-white"}, html.Text("Portal modal")),
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text("This dialog renders through ui.Portal into #portal-root rather than inside the example mount root.")),
					html.Div(html.Props{Class: "mt-5 flex gap-3"},
						html.Button(html.Props{ID: "confirm-modal", Class: "rounded-full border border-cyan-500/20 bg-cyan-950/80 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: ui.UseEvent(func() {
							parseConfirmCount.Update(func(parsePrev int) int { return parsePrev + 1 })
							parseShowModal.Set(false)
						})}, html.Text("Confirm Modal")),
						html.Button(html.Props{ID: "close-modal", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { parseShowModal.Set(false) })}, html.Text("Close")),
					),
				),
			),
		}))
	}
	if parseShowTooltip.Get() {
		parseOverlay = append(parseOverlay, ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: "#portal-root"},
			Child:  html.Div(html.Props{ID: "tooltip-surface", Role: "tooltip", Class: "fixed left-6 top-6 z-40 rounded-full border border-cyan-500/20 bg-slate-950/95 px-4 py-2 text-sm font-medium text-cyan-100 shadow-xl"}, html.Text("Tooltip rendered through the portal root")),
		}))
	}
	if parseShowPopover.Get() {
		parseOverlay = append(parseOverlay, ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: "#portal-root"},
			Child: html.Div(html.Props{ID: "popover-surface", Class: "fixed bottom-6 right-6 z-40 w-72 rounded-3xl border border-white/10 bg-slate-950/95 p-5 text-slate-100 shadow-2xl"},
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.2em] text-cyan-300"}, html.Text("Popover")),
				html.P(html.Props{Class: "mt-3 text-sm leading-6"}, html.Text("Popovers, tooltips, and dialogs can share one dedicated portal mount while staying outside the logical app container.")),
				html.Button(html.Props{ID: "dismiss-popover", Class: "mt-4 rounded-full border border-cyan-500/20 bg-cyan-950/80 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: ui.UseEvent(func() { parseShowPopover.Set(false) })}, html.Text("Dismiss")),
			),
		}))
	}

	parseRootChildren := []ui.Node{
		shared.ExamplePage(
			"Portals",
			"ui.Portal",
			"Keep controls in the example tree while rendering modal, tooltip, and popover surfaces into a separate DOM target.",
			shared.ExamplePanel("Controls",
				html.Div(html.Props{Class: "flex flex-wrap gap-2"},
					shared.ExampleButton("Open Modal", ui.UseEvent(func() { parseShowModal.Set(true) })),
					shared.ExampleButton("Toggle Tooltip", ui.UseEvent(func() { parseShowTooltip.Set(!parseShowTooltip.Get()) })),
					shared.ExampleButton("Toggle Popover", ui.UseEvent(func() { parseShowPopover.Set(!parseShowPopover.Get()) })),
				),
				html.Div(html.Props{Class: "grid gap-3 sm:grid-cols-2 lg:grid-cols-3"},
					shared.ExampleStat("Modal Confirms", fmt.Sprintf("%d", parseConfirmCount.Get())),
					shared.ExampleStat("Tooltip", fmt.Sprintf("%t", parseShowTooltip.Get())),
					shared.ExampleStat("Popover", fmt.Sprintf("%t", parseShowPopover.Get())),
				),
			),
			shared.ExamplePanel("Portal Contract",
				html.Div(html.Props{ID: "app-shell", Class: "rounded-[20px] border border-white/10 bg-slate-950/60 p-5 text-slate-300"},
					html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.18em] text-cyan-300"}, html.Text("Logical app container")),
					html.P(html.Props{Class: "mt-3 text-sm leading-6"}, html.Text("These controls stay inside the example mount root. The actual overlay DOM nodes render under the external portal target instead.")),
					html.Div(html.Props{ID: "portal-target-status", Class: "mt-4 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-cyan-200"}, html.Text("Target: #portal-root")),
				),
			),
		),
	}
	parseRootChildren = append(parseRootChildren, parseOverlay...)

	// Portal target rendered by the example itself so it works under any
	// hosting shell (the generated catalog shell only provides #app).  It must
	// precede the portal elements in the tree: portals resolve their target at
	// commit time and an unresolved target is not retried.
	parseRootChildren = append([]ui.Node{html.Div(html.Props{ID: "portal-root"})}, parseRootChildren...)
	return html.Div(html.Props{}, parseRootChildren...)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(App))
	exampleboot.WaitExampleRuntime()
}
