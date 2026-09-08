//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v6/examples/internal/examplelog"

	"github.com/monstercameron/GoWebComponents/v6/examples/shared"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"github.com/monstercameron/GoWebComponents/v6/utils"
)

type routeContentProps struct {
	Heading string
	Body    string
}

func routedAccessibilityPageView(parseProps routeContentProps) ui.Node {
	parseInspection := router.InspectCurrentRoute()
	parseAnnouncer := ui.UseAnnouncer()
	parseFocus := ui.UseFocusManager()

	ui.UseEffect(func() func() {
		parseAnnouncer.Polite("Loaded " + parseProps.Heading)
		parseFocus.FocusSelector("#route-page-heading")
		return nil
	}, parseInspection.Path, parseProps.Heading)

	return shared.ExamplePage(
		"Routed accessibility guidance",
		"Route-change announcements and heading focus after navigation",
		"A routed app still needs explicit spoken announcements and focus movement when the visible page changes. Each route page announces itself through ui.UseAnnouncer and then moves keyboard focus to the new page heading after navigation.",
		parseAnnouncer.Region(),
		shared.ExamplePanel("Route navigation",
			html.Nav(html.Props{Class: "mt-3 flex flex-wrap gap-3", Aria: map[string]string{"label": "Accessibility routes"}},
				html.A(html.Props{Href: "#/accessibility/overview", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200"}, html.Text("Overview")),
				html.A(html.Props{Href: "#/accessibility/settings", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200"}, html.Text("Settings")),
				html.A(html.Props{Href: "#/accessibility/reports", Class: "rounded-full border border-white/10 px-4 py-2 text-sm font-semibold text-slate-200"}, html.Text("Reports")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", parseInspection.Path),
				shared.ExampleStat("Loading", map[bool]string{true: "true", false: "false"}[parseInspection.Loading]),
			),
		),
		shared.ExamplePanel("Active route content",
			html.Article(html.Props{Class: "mt-3 rounded-[1.75rem] border border-white/10 bg-slate-950/45 p-6"},
				html.H2(html.Props{ID: "route-page-heading", Class: "text-3xl font-black tracking-tight text-white", Raw: map[string]interface{}{"tabIndex": -1}}, html.Text(parseProps.Heading)),
				html.P(html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"}, html.Text(parseProps.Body)),
			),
		),
	)
}

func routedAccessibilityPage(parseHeading, parseBody string) func(router.Attrs) *router.Element {
	return func(router.Attrs) *router.Element {
		return ui.CreateElement(routedAccessibilityPageView, routeContentProps{Heading: parseHeading, Body: parseBody})
	}
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/accessibility/overview"})
	parseR.Register("/accessibility/overview", routedAccessibilityPage("Overview accessibility", "This route announces itself through a polite live region and moves focus to the page heading after navigation."), router.Options{Title: "Accessibility overview"})
	parseR.Register("/accessibility/settings", routedAccessibilityPage("Settings accessibility", "Settings pages are a common place to miss focus restoration and page-change announcements because the UI often stays inside one app shell."), router.Options{Title: "Accessibility settings"})
	parseR.Register("/accessibility/reports", routedAccessibilityPage("Reports accessibility", "Route-driven dashboards should still expose a clear heading and a spoken status message when the screen meaningfully changes."), router.Options{Title: "Accessibility reports"})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
