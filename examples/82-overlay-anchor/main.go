//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	overlayAnchorSelectorRoot = "#overlay-anchor-root"
	overlayAnchorExplicitID   = "overlay-anchor-explicit-root"
)

func overlayAnchorExample() ui.Node {
	parseMenuOpen := ui.UseState(false)
	parseTooltipOpen := ui.UseState(false)
	parseTargetMode := ui.UseState("selector")
	parseHost := js.Global().Get("document").Call("getElementById", overlayAnchorExplicitID)
	parseHostReady := parseHost.Truthy()

	parseOpenMenu := ui.UseEvent(func() {
		parseMenuOpen.Set(true)
	})
	parseDismissMenu := func() {
		parseTooltipOpen.Set(false)
		parseMenuOpen.Set(false)
	}
	parseToggleTooltip := ui.UseEvent(func() {
		parseTooltipOpen.Set(!parseTooltipOpen.Get())
	})
	parseUseExplicitTarget := ui.UseEvent(func() {
		parseTargetMode.Set("explicit")
	})
	parseUseSelectorTarget := ui.UseEvent(func() {
		parseTargetMode.Set("selector")
	})

	parseTooltipTarget := ui.PortalTarget{Selector: overlayAnchorSelectorRoot}
	if parseTargetMode.Get() == "explicit" && parseHostReady {
		parseTooltipTarget = ui.PortalTarget{Node: parseHost}
	}

	parseMenuOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:                parseMenuOpen.Get(),
		Target:              ui.PortalTarget{Selector: overlayAnchorSelectorRoot},
		SurfaceID:           "overlay-anchor-menu",
		Kind:                ui.OverlayKindMenu,
		Role:                "menu",
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
		AnchorSelector:      "#open-overlay-anchor-menu",
		Positioning:         "anchored menu with viewport clamping guidance",
		SurfaceClass:        "fixed left-[calc(50%-13rem)] top-[calc(50%-1rem)] w-96 rounded-[1.75rem] border border-cyan-300/20 bg-slate-950/98 p-6 text-slate-100 shadow-[0_28px_110px_rgba(8,145,178,0.22)]",
		OnDismiss:           parseDismissMenu,
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Anchored menu")),
			html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white"}, html.Text("Portal target lab")),
			html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("Switch the tooltip target between the shared selector root and an explicitly resolved DOM node. The tooltip stays above the menu because stack order is global even when portal hosts differ.")),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				html.Button(html.Props{ID: "use-selector-overlay-target", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-4 py-2 text-sm font-semibold text-cyan-100", OnClick: parseUseSelectorTarget}, html.Text("Tooltip to selector root")),
				html.Button(html.Props{ID: "use-explicit-overlay-target", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200", OnClick: parseUseExplicitTarget}, html.Text("Tooltip to explicit node")),
				html.Button(html.Props{ID: "toggle-overlay-tooltip", Class: "rounded-full border border-amber-300/25 bg-amber-300/10 px-4 py-2 text-sm font-semibold text-amber-100", OnClick: parseToggleTooltip}, html.Text("Toggle tooltip")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Tooltip target", parseTargetMode.Get()),
				shared.ExampleStat("Tooltip open", fmt.Sprintf("%t", parseTooltipOpen.Get())),
				shared.ExampleStat("Explicit host", map[bool]string{true: "Ready", false: "Missing"}[parseHostReady]),
			),
			html.Div(html.Props{Class: "mt-6 flex gap-3"},
				html.Button(html.Props{ID: "dismiss-overlay-anchor-menu", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { parseDismissMenu() })}, html.Text("Dismiss menu")),
			),
		),
	})

	parseTooltipOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:           parseMenuOpen.Get() && parseTooltipOpen.Get(),
		Target:         parseTooltipTarget,
		SurfaceID:      "overlay-anchor-tooltip",
		Kind:           ui.OverlayKindTooltip,
		Role:           "tooltip",
		AnchorSelector: "#toggle-overlay-tooltip",
		Positioning:    "anchored tooltip with scroll and resize observers owned by the app",
		SurfaceClass:   "fixed left-[calc(50%+10rem)] top-[calc(50%-5rem)] max-w-xs rounded-full border border-amber-300/25 bg-slate-950/98 px-4 py-3 text-sm font-medium text-amber-100 shadow-[0_24px_80px_rgba(245,158,11,0.16)]",
		Child:          html.Text("Tooltip layer above the menu, even when portaled into the explicit host."),
	})

	return html.Div(html.Props{},
		shared.ExamplePage(
			"Anchored overlays and portal retargeting",
			"ui.Overlay across selector and explicit portal roots",
			"Use anchored overlays with one owner for placement math, scroll or resize observation, and collision handling, while letting ui.Overlay own stack order and dismissal routing across shared or explicit portal targets.",
			shared.ExamplePanel("Anchored menu and tooltip",
				html.P(html.Props{Class: "mt-3 max-w-3xl text-slate-300"}, html.Text("Open the menu, switch the tooltip host between selector and explicit-node targets, then toggle the tooltip. The menu and tooltip should remain layered consistently even though they can render into different physical DOM containers.")),
				html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
					html.Button(html.Props{ID: "open-overlay-anchor-menu", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: parseOpenMenu}, html.Text("Open anchored menu")),
					html.A(html.Props{Href: "#overlay-anchor-guidance", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200"}, html.Text("Read positioning notes")),
				),
			),
			shared.ExamplePanel("Positioning ownership",
				html.Div(html.Props{Class: "mt-3 rounded-[1.5rem] border border-white/10 bg-slate-950/40 p-6"},
					html.H2(html.Props{Class: "text-2xl font-bold text-white"}, html.Text("Anchoring guidance")),
					html.P(html.Props{ID: "overlay-anchor-guidance", Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("The framework stack manager decides ordering and dismissal. Your app still owns anchor measurement, viewport collision handling, and when to recompute placement after scroll or resize.")),
				),
			),
		),
		parseMenuOverlay,
		parseTooltipOverlay,
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(overlayAnchorExample), "#app")
	select {}
}
