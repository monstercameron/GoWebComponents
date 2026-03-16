//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func pill(label string) ui.Node {
	return html.Span(html.Props{Class: "rounded-full bg-[#dce8dd] px-3 py-1 text-xs font-semibold uppercase tracking-[0.18em] text-[#35544a]"}, html.Text(label))
}

func App() ui.Node {
	showModal := ui.UseState(false)
	showTooltip := ui.UseState(false)
	showPopover := ui.UseState(false)
	confirmCount := ui.UseState(0)

	overlay := make([]ui.Node, 0, 3)
	if showModal.Get() {
		overlay = append(overlay, ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: "#portal-root"},
			Child: html.Div(html.Props{ID: "modal-surface", Class: "fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-6"},
				html.Div(html.Props{Role: "dialog", Class: "w-full max-w-md rounded-3xl bg-[#f8f3e9] p-6 shadow-2xl ring-1 ring-black/10"},
					html.H2(html.Props{Class: "text-2xl font-bold text-[#18352f]"}, html.Text("Portal modal")),
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-[#45635a]"}, html.Text("This dialog is rendered through ui.Portal into #portal-root rather than inside #app.")),
					html.Div(html.Props{Class: "mt-5 flex gap-3"},
						html.Button(html.Props{ID: "confirm-modal", Class: "rounded-full bg-[#18352f] px-4 py-2 text-sm font-semibold text-white", OnClick: ui.UseEvent(func() {
							confirmCount.Update(func(prev int) int { return prev + 1 })
							showModal.Set(false)
						})}, html.Text("Confirm Modal")),
						html.Button(html.Props{ID: "close-modal", Class: "rounded-full border border-[#18352f]/20 px-4 py-2 text-sm font-semibold text-[#18352f]", OnClick: ui.UseEvent(func() { showModal.Set(false) })}, html.Text("Close")),
					),
				),
			),
		}))
	}
	if showTooltip.Get() {
		overlay = append(overlay, ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: "#portal-root"},
			Child: html.Div(html.Props{ID: "tooltip-surface", Role: "tooltip", Class: "fixed left-6 top-6 z-40 rounded-full bg-[#18352f] px-4 py-2 text-sm font-medium text-white shadow-xl"}, html.Text("Tooltip rendered through the portal root")),
		}))
	}
	if showPopover.Get() {
		overlay = append(overlay, ui.Portal(ui.PortalProps{
			Target: ui.PortalTarget{Selector: "#portal-root"},
			Child: html.Div(html.Props{ID: "popover-surface", Class: "fixed bottom-6 right-6 z-40 w-72 rounded-3xl bg-[#18352f] p-5 text-[#f7f4ee] shadow-2xl"},
				html.P(html.Props{Class: "text-xs font-semibold uppercase tracking-[0.2em] text-[#c7dbc9]"}, html.Text("Popover")),
				html.P(html.Props{Class: "mt-3 text-sm leading-6"}, html.Text("Popovers, tooltips, and dialogs can share one dedicated portal mount while staying outside the logical app container.")),
				html.Button(html.Props{ID: "dismiss-popover", Class: "mt-4 rounded-full bg-[#f7f4ee] px-4 py-2 text-sm font-semibold text-[#18352f]", OnClick: ui.UseEvent(func() { showPopover.Set(false) })}, html.Text("Dismiss")),
			),
		}))
	}

	children := []ui.Node{
		html.Div(html.Props{Class: "mx-auto max-w-5xl"},
			html.Div(html.Props{Class: "flex flex-wrap gap-3"}, pill("Public portal API"), pill("Selector targets"), pill("Overlay cleanup")),
			html.H1(html.Props{Class: "mt-6 text-4xl font-black tracking-[-0.04em]"}, html.Text("Portals for modals, tooltips, and popovers")),
			html.P(html.Props{Class: "mt-4 max-w-2xl text-base leading-7 text-[#45635a]"}, html.Text("The controls below live inside #app, while the overlay surfaces render into #portal-root through ui.Portal.")),
			html.Div(html.Props{Class: "mt-8 grid gap-6 lg:grid-cols-[minmax(0,1.2fr)_320px]"},
				html.Section(html.Props{ID: "app-shell", Class: "rounded-[32px] border border-[#18352f]/10 bg-white/70 p-6 shadow-[0_20px_80px_rgba(24,53,47,0.08)] backdrop-blur"},
					html.H2(html.Props{Class: "text-lg font-bold"}, html.Text("Logical app container")),
					html.P(html.Props{Class: "mt-2 text-sm leading-6 text-[#45635a]"}, html.Text("These controls toggle portal content without nesting the resulting overlay DOM under this panel.")),
					html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
						html.Button(html.Props{ID: "open-modal", Class: "rounded-full bg-[#18352f] px-5 py-3 text-sm font-semibold text-white", OnClick: ui.UseEvent(func() { showModal.Set(true) })}, html.Text("Open Modal")),
						html.Button(html.Props{ID: "toggle-tooltip", Class: "rounded-full border border-[#18352f]/20 px-5 py-3 text-sm font-semibold", OnClick: ui.UseEvent(func() { showTooltip.Set(!showTooltip.Get()) })}, html.Text("Toggle Tooltip")),
						html.Button(html.Props{ID: "toggle-popover", Class: "rounded-full border border-[#18352f]/20 px-5 py-3 text-sm font-semibold", OnClick: ui.UseEvent(func() { showPopover.Set(!showPopover.Get()) })}, html.Text("Toggle Popover")),
					),
					html.P(html.Props{ID: "confirm-count", Class: "mt-6 text-sm font-semibold text-[#35544a]"}, html.Text(fmt.Sprintf("Confirmed modal actions: %d", confirmCount.Get()))),
				),
				html.Aside(html.Props{Class: "rounded-[28px] bg-[#18352f] p-6 text-[#f7f4ee] shadow-[0_20px_80px_rgba(24,53,47,0.2)]"},
					html.H2(html.Props{Class: "text-lg font-bold"}, html.Text("Portal target contract")),
					html.P(html.Props{Class: "mt-3 text-sm leading-6 text-[#d8e5d7]"}, html.Text("The overlay host below is the only physical DOM container for the modal, tooltip, and popover surfaces.")),
					html.Div(html.Props{ID: "portal-target-status", Class: "mt-5 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 text-xs font-semibold uppercase tracking-[0.2em] text-[#c7dbc9]"}, html.Text("Target: #portal-root")),
				),
			),
		),
	}
	children = append(children, overlay...)

	return html.Div(html.Props{Class: "min-h-screen bg-[#f4efe4] px-6 py-10 text-[#18352f]"}, children...)
}

func main() {
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}