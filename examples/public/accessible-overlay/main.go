//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

const accessibleOverlayRoot = "#accessible-overlay-root"

func accessibleOverlayExample() ui.Node {
	parseOpen := ui.UseState(false)
	parseConfirmed := ui.UseState(0)
	parseTitleID := ui.UseId() + "-title"
	parseDescriptionID := ui.UseId() + "-description"

	parseOpenDialog := ui.UseEvent(func() {
		parseOpen.Set(true)
	})
	parseDismissDialog := func() {
		parseOpen.Set(false)
	}
	parseConfirmDialog := ui.UseEvent(func() {
		parseConfirmed.Update(func(parsePrevious int) int { return parsePrevious + 1 })
		parseOpen.Set(false)
	})

	parseOverlay := ui.CreateElement(ui.AccessibleOverlay, ui.AccessibleOverlayProps{
		Open:                 parseOpen.Get(),
		Target:               ui.PortalTarget{Selector: accessibleOverlayRoot},
		AppRootSelector:      "#overlay-page-shell",
		LabelledBy:           parseTitleID,
		DescribedBy:          parseDescriptionID,
		InitialFocusSelector: "#confirm-accessible-dialog",
		Modal:                true,
		TrapFocus:            true,
		RestoreFocus:         true,
		CloseOnEscape:        true,
		CloseOnOutsideClick:  true,
		LockScroll:           true,
		BackdropClass:        "fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 p-6",
		SurfaceClass:         "w-full max-w-xl rounded-[2rem] border border-cyan-400/20 bg-slate-950 p-8 text-slate-100 shadow-[0_30px_120px_rgba(8,145,178,0.25)]",
		OnDismiss:            parseDismissDialog,
		Child: html.Div(html.Props{},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-cyan-300"}, html.Text("Accessible overlay")),
			html.H2(html.Props{ID: parseTitleID, Class: "mt-3 text-3xl font-black tracking-tight text-white"}, html.Text("Review the release checklist")),
			html.P(html.Props{ID: parseDescriptionID, Class: "mt-4 text-base leading-7 text-slate-300"}, html.Text("The dialog traps keyboard focus, closes on escape, restores focus to the trigger, and marks the app shell inert while it is open.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 rounded-[1.5rem] border border-white/10 bg-white/5 p-5"},
				html.P(html.Props{Class: "text-sm text-slate-300"}, html.Text("Before shipping, confirm that the accessibility review, browser smoke, and documentation notes are complete.")),
				html.Ul(html.Props{Class: "grid gap-2 text-sm text-slate-200"},
					html.Li(html.Props{}, html.Text("Accessibility review sign-off")),
					html.Li(html.Props{}, html.Text("Browser smoke on modal keyboard flow")),
					html.Li(html.Props{}, html.Text("Release notes updated")),
				),
			),
			html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-3"},
				html.Button(html.Props{ID: "confirm-accessible-dialog", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: parseConfirmDialog}, html.Text("Confirm release")),
				html.Button(html.Props{ID: "close-accessible-dialog", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200", OnClick: ui.UseEvent(func() { parseDismissDialog() })}, html.Text("Close")),
			),
		),
	})

	return html.Div(html.Props{},
		html.Div(html.Props{ID: "overlay-page-shell"},
			shared.ExamplePage(
				"ui.AccessibleOverlay",
				"Portal-backed dialog semantics with focus trapping and restoration",
				"Use ui.AccessibleOverlay when a dialog or overlay should render through a portal while also trapping focus, restoring the trigger focus on close, dismissing on escape, and temporarily hiding the background app shell from assistive technology.",
				shared.ExamplePanel("Modal trigger",
					html.P(html.Props{Class: "mt-3 max-w-3xl text-slate-300"}, html.Text("Open the dialog, then use Tab, Shift+Tab, Escape, or the backdrop click path. The primitive keeps focus inside the modal surface and restores the trigger when the overlay closes.")),
					html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
						html.Button(html.Props{ID: "open-accessible-overlay", Class: "rounded-full border border-cyan-400/30 bg-cyan-400/10 px-5 py-3 text-sm font-semibold text-cyan-100", OnClick: parseOpenDialog}, html.Text("Open accessible dialog")),
						html.A(html.Props{Href: "#overlay-release-notes", Class: "rounded-full border border-white/10 px-5 py-3 text-sm font-semibold text-slate-200"}, html.Text("Read release notes")),
					),
					html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
						shared.ExampleStat("Dialog open", fmt.Sprintf("%t", parseOpen.Get())),
						shared.ExampleStat("Confirmed", fmt.Sprintf("%d", parseConfirmed.Get())),
						shared.ExampleStat("Portal target", accessibleOverlayRoot),
					),
				),
				shared.ExamplePanel("Background shell",
					html.Div(html.Props{Class: "mt-3 rounded-[1.5rem] border border-white/10 bg-slate-950/40 p-6"},
						html.H2(html.Props{Class: "text-2xl font-bold text-white"}, html.Text("Release summary")),
						html.P(html.Props{ID: "overlay-release-notes", Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("The background content remains visible for sighted users, but the overlay primitive marks it aria-hidden while the modal is open so assistive technology stays inside the dialog flow.")),
					),
				),
			),
		),
		// The portal target must exist in the DOM before the overlay commits;
		// rendering it here keeps the example self-contained instead of
		// depending on the hosting HTML shell to provide the root element.
		html.Div(html.Props{ID: "accessible-overlay-root"}),
		parseOverlay,
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(accessibleOverlayExample))
	exampleboot.WaitExampleRuntime()
}
