//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const overlayStackRoot = "#overlay-stack-root"

func overlayStackExample() ui.Node {
	parseParentOpen := ui.UseState(false)
	parseNestedOpen := ui.UseState(false)
	parsePopoverOpen := ui.UseState(false)
	parseConfirmed := ui.UseState(0)
	parseParentTitleID := ui.UseId() + "-parent-title"
	parseParentDescriptionID := ui.UseId() + "-parent-description"
	parseNestedTitleID := ui.UseId() + "-nested-title"
	parseNestedDescriptionID := ui.UseId() + "-nested-description"

	parseOpenParent := ui.UseEvent(func() {
		parseParentOpen.Set(true)
	})
	parseDismissParent := func() {
		parsePopoverOpen.Set(false)
		parseNestedOpen.Set(false)
		parseParentOpen.Set(false)
	}
	parseOpenNested := ui.UseEvent(func() {
		parseNestedOpen.Set(true)
	})
	parseDismissNested := func() {
		parseNestedOpen.Set(false)
	}
	parseTogglePopover := ui.UseEvent(func() {
		parsePopoverOpen.Set(!parsePopoverOpen.Get())
	})
	parseConfirmRelease := ui.UseEvent(func() {
		parseConfirmed.Update(func(parsePrevious int) int { return parsePrevious + 1 })
		parseDismissParent()
	})

	parseParentOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:                 parseParentOpen.Get(),
		Target:               ui.PortalTarget{Selector: overlayStackRoot},
		AppRootSelector:      "#overlay-stack-shell",
		SurfaceID:            "overlay-stack-parent-dialog",
		Kind:                 ui.OverlayKindDialog,
		Modal:                true,
		Backdrop:             true,
		TrapFocus:            true,
		RestoreFocus:         true,
		CloseOnEscape:        true,
		CloseOnOutsideClick:  true,
		LockScroll:           true,
		BackgroundInert:      true,
		LabelledBy:           parseParentTitleID,
		DescribedBy:          parseParentDescriptionID,
		InitialFocusSelector: "#open-nested-overlay-dialog",
		BackdropClass:        "fixed inset-0 flex items-center justify-center bg-slate-950/82 p-6",
		SurfaceClass:         "w-full max-w-2xl rounded-[2rem] border border-cyan-300/20 bg-slate-950 p-8 text-slate-100 shadow-[0_40px_140px_rgba(8,145,178,0.24)]",
		OnDismiss:            parseDismissParent,
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Primary dialog")),
			html.H2(html.Props{ID: parseParentTitleID, Class: "mt-3 text-3xl font-black tracking-tight text-white"}, html.Text("Release orchestration board")),
			html.P(html.Props{ID: parseParentDescriptionID, Class: "mt-4 text-base leading-7 text-slate-300"}, html.Text("Open the nested dialog or the side popover to verify that escape and focus restore route to the topmost eligible layer before the parent dialog reacts.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5 md:grid-cols-3"},
				shared.ExampleStat("Depth owner", "Parent dialog"),
				shared.ExampleStat("Nested open", fmt.Sprintf("%t", parseNestedOpen.Get())),
				shared.ExampleStat("Popover open", fmt.Sprintf("%t", parsePopoverOpen.Get())),
			),
			html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
				html.Button(html.Props{ID: "open-nested-overlay-dialog", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: parseOpenNested}, html.Text("Open nested dialog")),
				html.Button(html.Props{ID: "open-overlay-popover", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200", OnClick: parseTogglePopover}, html.Text("Toggle review popover")),
				html.Button(html.Props{ID: "confirm-overlay-stack", Class: "rounded-full border border-emerald-400/30 bg-emerald-400/10 px-5 py-3 text-sm font-semibold text-emerald-100", OnClick: parseConfirmRelease}, html.Text("Confirm release")),
			),
			html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-300"}, html.Text("Expected routing: first Escape closes the popover if open, otherwise the nested dialog, and only then the parent dialog. Body scroll stays locked until the last modal layer closes.")),
		),
	})

	parsePopoverOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:                parseParentOpen.Get() && parsePopoverOpen.Get(),
		Target:              ui.PortalTarget{Selector: overlayStackRoot},
		SurfaceID:           "overlay-stack-popover",
		Kind:                ui.OverlayKindPopover,
		Role:                "dialog",
		CloseOnEscape:       true,
		CloseOnOutsideClick: true,
		AnchorSelector:      "#open-overlay-popover",
		Positioning:         "anchored with manual placement and viewport clamping guidance",
		SurfaceClass:        "fixed left-[calc(50%+12rem)] top-[calc(50%-2rem)] w-80 rounded-[1.5rem] border border-amber-300/25 bg-slate-950/98 p-5 text-slate-100 shadow-[0_25px_90px_rgba(245,158,11,0.18)]",
		OnDismiss: func() {
			parsePopoverOpen.Set(false)
		},
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-amber-300"}, html.Text("Popover layer")),
			html.H3(html.Props{Class: "mt-3 text-xl font-bold text-white"}, html.Text("Review notes")),
			html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text("This non-modal layer sits above the dialog, owns outside-click dismissal while open, and leaves the parent dialog mounted and stable when it closes.")),
			html.Div(html.Props{Class: "mt-5 flex gap-3"},
				html.Button(html.Props{ID: "dismiss-overlay-popover", Class: "rounded-full border border-amber-300/25 bg-amber-300/10 px-4 py-2 text-sm font-semibold text-amber-100", OnClick: ui.UseEvent(func() { parsePopoverOpen.Set(false) })}, html.Text("Dismiss popover")),
			),
		),
	})

	parseNestedOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:                 parseParentOpen.Get() && parseNestedOpen.Get(),
		Target:               ui.PortalTarget{Selector: overlayStackRoot},
		AppRootSelector:      "#overlay-stack-shell",
		SurfaceID:            "overlay-stack-nested-dialog",
		Kind:                 ui.OverlayKindDialog,
		Modal:                true,
		Backdrop:             true,
		TrapFocus:            true,
		RestoreFocus:         true,
		CloseOnEscape:        true,
		CloseOnOutsideClick:  true,
		LockScroll:           true,
		BackgroundInert:      true,
		LabelledBy:           parseNestedTitleID,
		DescribedBy:          parseNestedDescriptionID,
		InitialFocusSelector: "#confirm-nested-overlay-dialog",
		BackdropClass:        "fixed inset-0 flex items-center justify-center bg-slate-950/60 p-6 backdrop-blur-[2px]",
		SurfaceClass:         "w-full max-w-lg rounded-[1.75rem] border border-fuchsia-300/25 bg-slate-950 p-7 text-slate-100 shadow-[0_35px_120px_rgba(192,38,211,0.22)]",
		OnDismiss:            parseDismissNested,
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-fuchsia-300"}, html.Text("Nested dialog")),
			html.H3(html.Props{ID: parseNestedTitleID, Class: "mt-3 text-2xl font-black text-white"}, html.Text("Sign off the final checklist")),
			html.P(html.Props{ID: parseNestedDescriptionID, Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("This child dialog should close first on Escape and restore focus to the parent dialog trigger instead of jumping back to the page behind both overlays.")),
			html.Div(html.Props{Class: "mt-6 flex gap-3"},
				html.Button(html.Props{ID: "confirm-nested-overlay-dialog", Class: "rounded-full border border-fuchsia-300/25 bg-fuchsia-300/10 px-5 py-3 text-sm font-semibold text-fuchsia-100", OnClick: ui.UseEvent(func() { parseDismissNested() })}, html.Text("Close nested dialog")),
				html.Button(html.Props{ID: "cancel-nested-overlay-dialog", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { parseDismissNested() })}, html.Text("Cancel")),
			),
		),
	})

	return html.Div(html.Props{},
		html.Div(html.Props{ID: "overlay-stack-shell"},
			shared.ExamplePage(
				"ui.Overlay stack coordination",
				"Shared portal layering with nested dismissal and focus restore",
				"Use ui.Overlay when several portal-backed surfaces must share stack order, z-index ownership, focus routing, escape handling, and modal side effects instead of each overlay wiring those behaviors independently.",
				shared.ExamplePanel("Stacked modal flow",
					html.P(html.Props{Class: "mt-3 max-w-3xl text-slate-300"}, html.Text("Open the parent dialog, then the nested dialog and popover in different orders. The shared overlay manager keeps dismissal and focus targeted at the topmost eligible layer without tearing down the parent flow.")),
					html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
						html.Button(html.Props{ID: "open-overlay-stack-dialog", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: parseOpenParent}, html.Text("Open release board")),
						html.A(html.Props{Href: "#overlay-stack-notes", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200"}, html.Text("Read stack notes")),
					),
					html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
						shared.ExampleStat("Parent open", fmt.Sprintf("%t", parseParentOpen.Get())),
						shared.ExampleStat("Confirmed", fmt.Sprintf("%d", parseConfirmed.Get())),
						shared.ExampleStat("Portal target", overlayStackRoot),
					),
				),
				shared.ExamplePanel("Background shell",
					html.Div(html.Props{Class: "mt-3 rounded-[1.5rem] border border-white/10 bg-slate-950/40 p-6"},
						html.H2(html.Props{Class: "text-2xl font-bold text-white"}, html.Text("Release timeline")),
						html.P(html.Props{ID: "overlay-stack-notes", Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("The shell should remain inert for assistive technology while either modal dialog is open. The nested modal increments the same lock and inert ownership instead of replacing the parent registration.")),
					),
				),
			),
		),
		// Portal target rendered by the example itself so it works under
		// any hosting shell (the generated catalog shell only provides #app).
		html.Div(html.Props{ID: "overlay-stack-root"}),
		parseParentOverlay,
		parsePopoverOverlay,
		parseNestedOverlay,
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(overlayStackExample))
	exampleboot.WaitExampleRuntime()
}
