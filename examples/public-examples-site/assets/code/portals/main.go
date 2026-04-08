//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func pill(parseLabel string) ui.Node {
	return html.Span(html.Props{Class: "rounded-full border border-cyan-500/20 bg-cyan-950/60 px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-cyan-100"}, html.Text(parseLabel))
}

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
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text("This dialog is rendered through ui.Portal into #portal-root rather than inside the example mount root.")),
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

	parseChildren := []ui.Node{
		html.Div(html.Props{Class: "mx-auto max-w-5xl"},
			html.Div(html.Props{Class: "flex flex-wrap gap-3"}, pill("Public portal API"), pill("Selector targets"), pill("Overlay cleanup")),
			html.H1(html.Props{Class: "mt-6 text-4xl font-black tracking-[-0.04em]"}, html.Text("Portals for modals, tooltips, and popovers")),
			html.P(html.Props{Class: "mt-4 max-w-2xl text-base leading-7 text-slate-300"}, html.Text("The controls below live inside the example mount root, while the overlay surfaces render into #portal-root through ui.Portal.")),
			html.Div(html.Props{Class: "mt-8 grid gap-6 lg:grid-cols-[minmax(0,1.2fr)_320px]"},
				html.Section(html.Props{ID: "app-shell", Class: "rounded-[32px] border border-white/10 bg-slate-950/80 p-6 shadow-[0_20px_80px_rgba(2,6,23,0.45)] backdrop-blur"},
					html.H2(html.Props{Class: "text-lg font-bold"}, html.Text("Logical app container")),
					html.P(html.Props{Class: "mt-2 text-sm leading-6 text-slate-300"}, html.Text("These controls toggle portal content without nesting the resulting overlay DOM under this panel.")),
					html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
						html.Button(html.Props{ID: "open-modal", Class: "rounded-full border border-cyan-500/20 bg-cyan-950/80 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: ui.UseEvent(func() { parseShowModal.Set(true) })}, html.Text("Open Modal")),
						html.Button(html.Props{ID: "toggle-tooltip", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { parseShowTooltip.Set(!parseShowTooltip.Get()) })}, html.Text("Toggle Tooltip")),
						html.Button(html.Props{ID: "toggle-popover", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { parseShowPopover.Set(!parseShowPopover.Get()) })}, html.Text("Toggle Popover")),
					),
					html.P(html.Props{ID: "confirm-count", Class: "mt-6 text-sm font-semibold text-cyan-200"}, html.Text(fmt.Sprintf("Confirmed modal actions: %d", parseConfirmCount.Get()))),
				),
				html.Aside(html.Props{Class: "rounded-[28px] border border-white/10 bg-slate-950/90 p-6 text-slate-100 shadow-[0_20px_80px_rgba(2,6,23,0.45)]"},
					html.H2(html.Props{Class: "text-lg font-bold"}, html.Text("Portal target contract")),
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text("The overlay host below is the only physical DOM container for the modal, tooltip, and popover surfaces.")),
					html.Div(html.Props{ID: "portal-target-status", Class: "mt-5 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-cyan-200"}, html.Text("Target: #portal-root")),
				),
			),
		),
	}
	parseChildren = append(parseChildren, parseOverlay...)

	return html.Div(html.Props{Class: "min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(34,211,238,0.16),_transparent_26%),linear-gradient(180deg,#08111d_0%,#030712_100%)] px-6 py-10 text-slate-100"}, parseChildren...)
}

func main() {
	exampleboot.RenderExampleRoot(ui.CreateElement(App))
	exampleboot.WaitExampleRuntime()
}
