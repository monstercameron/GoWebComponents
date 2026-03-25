//go:build js && wasm
// +build js,wasm

package main

import (
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func nestedShell(parseTitle string) *router.Element {
	parseInspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		parseTitle,
		"Nested layout routes with router.Outlet()",
		"A layout route stays mounted while more specific child routes render into router.Outlet(). That lets shared nav and chrome remain stable across nested navigation.",
		shared.ExamplePanel("Persistent layout shell",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				html.A(html.Props{Href: "#/dashboard/overview", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Overview")),
				html.A(html.Props{Href: "#/dashboard/reports/7", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Report 7")),
				html.A(html.Props{Href: "#/dashboard/reports/12", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Report 12")),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", parseInspection.Path),
				shared.ExampleStat("Layout route", "/dashboard"),
			),
		),
		shared.ExamplePanel("Outlet content", html.Div(html.Props{Class: "mt-3"}, router.GetOutlet())),
	)
}

func dashboardLayout(router.Attrs) *router.Element {
	return nestedShell("Nested layout routes")
}

func dashboardOverview(router.Attrs) *router.Element {
	return html.Article(html.Props{Class: "rounded-[1.5rem] border border-emerald-400/20 bg-emerald-400/10 p-6 text-emerald-50"},
		html.H3(html.Props{Class: "text-2xl font-bold"}, html.Text("Overview child route")),
		html.P(html.Props{Class: "mt-3 leading-7"}, html.Text("This child route renders into the dashboard layout outlet without replacing the outer shell.")),
	)
}

func reportPageView() ui.Node {
	parseParams := router.UseParams()
	return html.Article(html.Props{Class: "rounded-[1.5rem] border border-cyan-400/20 bg-cyan-400/10 p-6 text-cyan-50"},
		html.H3(html.Props{Class: "text-2xl font-bold"}, html.Text("Report child route")),
		html.P(html.Props{Class: "mt-3 leading-7"}, html.Text("Child params still work inside nested layouts. Current report ID: "+parseParams.Get("id"))),
	)
}

func reportPage(router.Attrs) *router.Element {
	return ui.CreateElement(reportPageView)
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/dashboard/overview"})
	parseR.Register("/dashboard", dashboardLayout, router.Options{Layout: true})
	parseR.Register("/dashboard/overview", dashboardOverview)
	parseR.Register("/dashboard/reports/:id", reportPage)
	parseR.Mount("#app")
	select {}
}
