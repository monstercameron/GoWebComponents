//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func isActivePath(parseCurrentPath, parseTargetPath string) bool {
	if parseCurrentPath == parseTargetPath {
		return true
	}
	if parseTargetPath == "/" {
		return parseCurrentPath == "/"
	}
	return strings.HasPrefix(parseCurrentPath, parseTargetPath+"/")
}

func routeLink(parseLabel, parsePath, parseCurrentPath string) ui.Node {
	parseClassName := "inline-flex rounded-full border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-slate-200 transition-colors hover:bg-cyan-500/15 hover:text-white"
	if isActivePath(parseCurrentPath, parsePath) {
		parseClassName = "inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-4 py-2 text-sm font-semibold text-cyan-100"
	}
	return html.A(html.Props{Href: "#" + parsePath, Class: parseClassName}, html.Text(parseLabel))
}

func homePage(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return shared.ExamplePage(
		"Nested Routes",
		"router.GetOutlet",
		"Keep parent navigation mounted while deeper leaf routes replace only the outlet subtree.",
		shared.ExamplePanel("Navigate",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				routeLink("Dashboard", "/dashboard/overview", parseCurrentPath),
				routeLink("Settings", "/dashboard/settings/profile", parseCurrentPath),
				routeLink("Docs", "/docs/getting-started", parseCurrentPath),
			),
		),
		shared.ExamplePanel("Signals",
			html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Layout levels", "3"),
				shared.ExampleStat("URL params", "Leaf only"),
				shared.ExampleStat("Docs tree", "Separate"),
			),
		),
	)
}

func dashboardLayout(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return shared.ExamplePage(
		"Nested Routes",
		"Dashboard layout",
		"The dashboard frame remains mounted while reports and settings swap underneath it.",
		shared.ExamplePanel("Dashboard Nav",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				routeLink("Home", "/", parseCurrentPath),
				routeLink("Overview", "/dashboard/overview", parseCurrentPath),
				routeLink("Report 7", "/dashboard/reports/7", parseCurrentPath),
				routeLink("Report 12", "/dashboard/reports/12", parseCurrentPath),
				routeLink("Settings", "/dashboard/settings/profile", parseCurrentPath),
				routeLink("Docs", "/docs/getting-started", parseCurrentPath),
			),
		),
		shared.ExamplePanel("Workspace", router.GetOutlet()),
	)
}

func dashboardOverviewPage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Route", "/dashboard/overview"),
			shared.ExampleStat("Parent", "Dashboard"),
			shared.ExampleStat("Leaf", "Overview"),
		),
		shared.ExampleBulletList(
			"This leaf route changes the dashboard content area without replacing the dashboard shell.",
			"Move to reports or settings to see the same parent frame stay mounted around new outlet content.",
		),
	)
}

func reportPage(parseProps router.Attrs) ui.Node {
	parseParams := router.UseParams()
	parseReportID := parseParams.Get("id")
	parseNextReport := "7"
	if parseReportID == "7" {
		parseNextReport = "12"
	}
	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "grid gap-4 md:grid-cols-3"},
			shared.ExampleStat("Report", "#"+parseReportID),
			shared.ExampleStat("Parent", "Dashboard"),
			shared.ExampleStat("Params", "useParams"),
		),
		shared.ExampleBulletList(
			"Leaf params stay scoped to the report route while the dashboard shell keeps its own state and nav.",
			"Switch reports and only the leaf content changes.",
		),
		routeLink("Open report #"+parseNextReport, "/dashboard/reports/"+parseNextReport, router.GetCurrentPath()),
	)
}

func settingsLayout(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return html.Div(html.Props{Class: "space-y-4"},
		html.Div(html.Props{Class: "flex flex-wrap gap-3"},
			routeLink("Profile", "/dashboard/settings/profile", parseCurrentPath),
			routeLink("Team", "/dashboard/settings/team", parseCurrentPath),
		),
		shared.ExampleBulletList(
			"This is a second layout route nested under the dashboard tree.",
			"It proves that deeper layout shells can own their own nav while still rendering inside the parent outlet.",
		),
		html.Div(html.Props{Class: "rounded-[20px] border border-white/10 bg-slate-950/60 p-4 shadow-inner shadow-black/20"}, router.GetOutlet()),
	)
}

func settingsProfilePage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		shared.ExampleStat("Settings page", "Profile"),
		shared.ExampleBulletList(
			"Use a nested settings layout when several related pages need local navigation and shared framing.",
			"The parent dashboard shell still stays mounted around this deeper route.",
		),
	)
}

func settingsTeamPage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		shared.ExampleStat("Settings page", "Team"),
		shared.ExampleBulletList(
			"Sibling settings leaves share the same nested shell.",
			"Only the settings outlet changes when you switch between profile and team.",
		),
	)
}

func docsLayout(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return shared.ExamplePage(
		"Nested Routes",
		"Docs layout",
		"A second route tree can keep its own sidebar and content frame without sharing the dashboard shell.",
		shared.ExamplePanel("Docs Nav",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				routeLink("Home", "/", parseCurrentPath),
				routeLink("Dashboard", "/dashboard/overview", parseCurrentPath),
				routeLink("Getting Started", "/docs/getting-started", parseCurrentPath),
				routeLink("Routing", "/docs/routing", parseCurrentPath),
			),
		),
		shared.ExamplePanel("Article", router.GetOutlet()),
	)
}

func docsGettingStartedPage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		shared.ExampleStat("Doc", "Getting Started"),
		shared.ExampleBulletList(
			"Start with one layout route per major section, then register leaf pages beneath that prefix.",
			"Parent layouts render router.GetOutlet to place the current child route inside the shared frame.",
		),
	)
}

func docsRoutingPage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-4"},
		shared.ExampleStat("Doc", "Routing"),
		shared.ExampleBulletList(
			"Only routes marked with Layout true act as parent shells.",
			"Leaf routes see the final merged params while each layout keeps control of its own mounted UI.",
		),
	)
}

func notFoundPage(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return shared.ExamplePage(
		"Nested Routes",
		"Not found",
		"Missing paths should still direct users back into one of the valid route trees.",
		shared.ExamplePanel("Current route", shared.ExampleStat("Path", parseCurrentPath)),
		shared.ExamplePanel("Navigate",
			html.Div(html.Props{Class: "flex flex-wrap gap-3"},
				routeLink("Dashboard", "/dashboard/overview", parseCurrentPath),
				routeLink("Docs", "/docs/getting-started", parseCurrentPath),
			),
		),
	)
}

func main() {
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", homePage)
	parseR.Register("/dashboard", dashboardLayout, router.Options{Layout: true, Title: "Nested Routes Demo Dashboard"})
	parseR.Register("/dashboard/overview", dashboardOverviewPage, router.Options{Title: "Nested Routes Demo Overview"})
	parseR.Register("/dashboard/reports/:id", reportPage, router.Options{Title: "Nested Routes Demo Report"})
	parseR.Register("/dashboard/settings", settingsLayout, router.Options{Layout: true, Title: "Nested Routes Demo Settings"})
	parseR.Register("/dashboard/settings/profile", settingsProfilePage, router.Options{Title: "Nested Routes Demo Profile"})
	parseR.Register("/dashboard/settings/team", settingsTeamPage, router.Options{Title: "Nested Routes Demo Team"})
	parseR.Register("/docs", docsLayout, router.Options{Layout: true, Title: "Nested Routes Demo Docs"})
	parseR.Register("/docs/getting-started", docsGettingStartedPage, router.Options{Title: "Nested Routes Demo Getting Started"})
	parseR.Register("/docs/routing", docsRoutingPage, router.Options{Title: "Nested Routes Demo Routing"})
	parseR.Register("*", notFoundPage, router.Options{Title: "Nested Routes Demo Not Found"})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
