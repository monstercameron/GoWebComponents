//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const overlayStackRoot = "#overlay-stack-root"

func overlayStackExample() ui.Node {
	parentOpen := ui.UseState(false)
	nestedOpen := ui.UseState(false)
	popoverOpen := ui.UseState(false)
	confirmed := ui.UseState(0)
	parentTitleID := ui.UseId() + "-parent-title"
	parentDescriptionID := ui.UseId() + "-parent-description"
	nestedTitleID := ui.UseId() + "-nested-title"
	nestedDescriptionID := ui.UseId() + "-nested-description"

	openParent := ui.UseEvent(func() {
		parentOpen.Set(true)
	})
	dismissParent := func() {
		popoverOpen.Set(false)
		nestedOpen.Set(false)
		parentOpen.Set(false)
	}
	openNested := ui.UseEvent(func() {
		nestedOpen.Set(true)
	})
	dismissNested := func() {
		nestedOpen.Set(false)
	}
	togglePopover := ui.UseEvent(func() {
		popoverOpen.Set(!popoverOpen.Get())
	})
	confirmRelease := ui.UseEvent(func() {
		confirmed.Update(func(previous int) int { return previous + 1 })
		dismissParent()
	})

	parentOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:                 parentOpen.Get(),
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
		LabelledBy:           parentTitleID,
		DescribedBy:          parentDescriptionID,
		InitialFocusSelector: "#open-nested-overlay-dialog",
		BackdropClass:        "fixed inset-0 flex items-center justify-center bg-slate-950/82 p-6",
		SurfaceClass:         "w-full max-w-2xl rounded-[2rem] border border-cyan-300/20 bg-slate-950 p-8 text-slate-100 shadow-[0_40px_140px_rgba(8,145,178,0.24)]",
		OnDismiss:            dismissParent,
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Primary dialog")),
			html.H2(html.Props{ID: parentTitleID, Class: "mt-3 text-3xl font-black tracking-tight text-white"}, html.Text("Release orchestration board")),
			html.P(html.Props{ID: parentDescriptionID, Class: "mt-4 text-base leading-7 text-slate-300"}, html.Text("Open the nested dialog or the side popover to verify that escape and focus restore route to the topmost eligible layer before the parent dialog reacts.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5 md:grid-cols-3"},
				shared.ExampleStat("Depth owner", "Parent dialog"),
				shared.ExampleStat("Nested open", fmt.Sprintf("%t", nestedOpen.Get())),
				shared.ExampleStat("Popover open", fmt.Sprintf("%t", popoverOpen.Get())),
			),
			html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
				html.Button(html.Props{ID: "open-nested-overlay-dialog", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: openNested}, html.Text("Open nested dialog")),
				html.Button(html.Props{ID: "open-overlay-popover", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200", OnClick: togglePopover}, html.Text("Toggle review popover")),
				html.Button(html.Props{ID: "confirm-overlay-stack", Class: "rounded-full border border-emerald-400/30 bg-emerald-400/10 px-5 py-3 text-sm font-semibold text-emerald-100", OnClick: confirmRelease}, html.Text("Confirm release")),
			),
			html.P(html.Props{Class: "mt-6 text-sm leading-7 text-slate-300"}, html.Text("Expected routing: first Escape closes the popover if open, otherwise the nested dialog, and only then the parent dialog. Body scroll stays locked until the last modal layer closes.")),
		),
	})

	popoverOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:                parentOpen.Get() && popoverOpen.Get(),
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
			popoverOpen.Set(false)
		},
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-amber-300"}, html.Text("Popover layer")),
			html.H3(html.Props{Class: "mt-3 text-xl font-bold text-white"}, html.Text("Review notes")),
			html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text("This non-modal layer sits above the dialog, owns outside-click dismissal while open, and leaves the parent dialog mounted and stable when it closes.")),
			html.Div(html.Props{Class: "mt-5 flex gap-3"},
				html.Button(html.Props{ID: "dismiss-overlay-popover", Class: "rounded-full border border-amber-300/25 bg-amber-300/10 px-4 py-2 text-sm font-semibold text-amber-100", OnClick: ui.UseEvent(func() { popoverOpen.Set(false) })}, html.Text("Dismiss popover")),
			),
		),
	})

	nestedOverlay := ui.CreateElement(ui.Overlay, ui.OverlayProps{
		Open:                 parentOpen.Get() && nestedOpen.Get(),
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
		LabelledBy:           nestedTitleID,
		DescribedBy:          nestedDescriptionID,
		InitialFocusSelector: "#confirm-nested-overlay-dialog",
		BackdropClass:        "fixed inset-0 flex items-center justify-center bg-slate-950/60 p-6 backdrop-blur-[2px]",
		SurfaceClass:         "w-full max-w-lg rounded-[1.75rem] border border-fuchsia-300/25 bg-slate-950 p-7 text-slate-100 shadow-[0_35px_120px_rgba(192,38,211,0.22)]",
		OnDismiss:            dismissNested,
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.24em] text-fuchsia-300"}, html.Text("Nested dialog")),
			html.H3(html.Props{ID: nestedTitleID, Class: "mt-3 text-2xl font-black text-white"}, html.Text("Sign off the final checklist")),
			html.P(html.Props{ID: nestedDescriptionID, Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text("This child dialog should close first on Escape and restore focus to the parent dialog trigger instead of jumping back to the page behind both overlays.")),
			html.Div(html.Props{Class: "mt-6 flex gap-3"},
				html.Button(html.Props{ID: "confirm-nested-overlay-dialog", Class: "rounded-full border border-fuchsia-300/25 bg-fuchsia-300/10 px-5 py-3 text-sm font-semibold text-fuchsia-100", OnClick: ui.UseEvent(func() { dismissNested() })}, html.Text("Close nested dialog")),
				html.Button(html.Props{ID: "cancel-nested-overlay-dialog", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { dismissNested() })}, html.Text("Cancel")),
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
						html.Button(html.Props{ID: "open-overlay-stack-dialog", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: openParent}, html.Text("Open release board")),
						html.A(html.Props{Href: "#overlay-stack-notes", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200"}, html.Text("Read stack notes")),
					),
					html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
						shared.ExampleStat("Parent open", fmt.Sprintf("%t", parentOpen.Get())),
						shared.ExampleStat("Confirmed", fmt.Sprintf("%d", confirmed.Get())),
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
		parentOverlay,
		popoverOverlay,
		nestedOverlay,
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(overlayStackExample), "#app")
	select {}
}
