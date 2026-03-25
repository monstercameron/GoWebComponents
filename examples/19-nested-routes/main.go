//go:build js && wasm
// +build js,wasm

package main

import (
	"strings"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"

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

func layoutSection(parseTitle, parseSubtitle string, parseNavItems ...ui.Node) ui.Node {
	return html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-6 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
		html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text(parseTitle)),
		html.H2(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text(parseSubtitle)),
		html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"}, parseNavItems...),
	)
}

func homePage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(34,211,238,0.18),_transparent_28%),linear-gradient(180deg,#08111d_0%,#030712_100%)] px-6 py-10 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl"},
			html.Div(html.Props{Class: "rounded-[2.5rem] border border-white/10 bg-slate-950/80 p-8 shadow-[0_25px_90px_rgba(2,6,23,0.48)] md:p-12"},
				html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.32em] text-cyan-300"}, html.Text("Nested Routes Demo")),
				html.H1(html.Props{Class: "mt-4 max-w-3xl text-5xl font-black leading-tight text-white md:text-6xl"}, html.Text("Dashboard shells, nested settings pages, and docs navigation using router.Outlet().")),
				html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("This example keeps layout chrome mounted at each level while the leaf route changes beneath it. Open the dashboard, drill into settings, then switch across the docs section to see different outlet stacks.")),
				html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-4"},
					routeLink("Open Dashboard", "/dashboard/overview", router.GetCurrentPath()),
					routeLink("Open Settings", "/dashboard/settings/profile", router.GetCurrentPath()),
					routeLink("Open Docs", "/docs/getting-started", router.GetCurrentPath()),
				),
			),
		),
	)
}

func dashboardLayout(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return html.Div(html.Props{Class: "min-h-screen bg-[linear-gradient(180deg,#08111d_0%,#020617_100%)] px-6 py-8 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl"},
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4 rounded-[2rem] border border-white/10 bg-slate-950/85 px-6 py-5 text-slate-100 shadow-[0_24px_80px_rgba(2,6,23,0.45)]"},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Dashboard Layout")),
					html.H1(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Operations workspace")),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					routeLink("Home", "/", parseCurrentPath),
					routeLink("Docs", "/docs/getting-started", parseCurrentPath),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-6 lg:grid-cols-[280px_minmax(0,1fr)]"},
				html.Aside(html.Props{Class: "space-y-6"},
					layoutSection("Dashboard", "Persistent parent shell",
						routeLink("Overview", "/dashboard/overview", parseCurrentPath),
						routeLink("Report 7", "/dashboard/reports/7", parseCurrentPath),
						routeLink("Report 12", "/dashboard/reports/12", parseCurrentPath),
						routeLink("Settings", "/dashboard/settings/profile", parseCurrentPath),
					),
					html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-6 text-sm leading-7 text-slate-300 shadow-[0_18px_60px_rgba(2,6,23,0.42)]"},
						html.P(html.Props{Class: "font-bold text-white"}, html.Text("What stays mounted")),
						html.P(html.Props{Class: "mt-3"}, html.Text("This dashboard shell remains stable while child routes swap inside router.Outlet(). That keeps navigation, headings, and summary chrome in one place.")),
					),
				),
				html.Main(html.Props{Class: "space-y-6"}, router.GetOutlet()),
			),
		),
	)
}

func dashboardOverviewPage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
		html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Leaf Route")),
		html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Overview")),
		html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("This is a standard child route under the dashboard layout. Switch to reports or settings and the dashboard shell stays mounted while only the outlet subtree changes.")),
	)
}

func reportPage(parseProps router.Attrs) ui.Node {
	parseParams := router.UseParams()
	parseReportID := parseParams.Get("id")
	parseNextReport := "7"
	if parseReportID == "7" {
		parseNextReport = "12"
	}
	return html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
		html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Nested Param Route")),
		html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Report #"+parseReportID)),
		html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("router.UseParams() resolves the leaf route param while the parent dashboard layout keeps its own shell state and outlet position.")),
		html.A(html.Props{Href: "#/dashboard/reports/" + parseNextReport, Class: "mt-6 inline-flex rounded-full border border-cyan-500/30 bg-cyan-500/20 px-5 py-3 text-sm font-semibold text-cyan-100 hover:bg-cyan-500/30"}, html.Text("Open report #"+parseNextReport)),
	)
}

func settingsLayout(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return html.Div(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/70 p-6 shadow-[0_18px_60px_rgba(2,6,23,0.42)]"},
		html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Nested Settings Layout")),
		html.H2(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text("Settings shell inside the dashboard tree")),
		html.P(html.Props{Class: "mt-4 max-w-2xl text-base leading-7 text-slate-300"}, html.Text("This second layout route wraps only settings pages. It demonstrates a deeper layout stack: dashboard layout -> settings layout -> settings leaf.")),
		html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
			routeLink("Profile", "/dashboard/settings/profile", parseCurrentPath),
			routeLink("Team", "/dashboard/settings/team", parseCurrentPath),
		),
		html.Div(html.Props{Class: "mt-6"}, router.GetOutlet()),
	)
}

func settingsProfilePage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-900/90 p-6"},
		html.H3(html.Props{Class: "text-2xl font-black text-white"}, html.Text("Profile settings")),
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Keep identity, notification preferences, and workspace defaults in a dedicated settings outlet without remounting the surrounding dashboard or settings chrome.")),
	)
}

func settingsTeamPage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.5rem] border border-white/10 bg-slate-900/90 p-6"},
		html.H3(html.Props{Class: "text-2xl font-black text-white"}, html.Text("Team settings")),
		html.P(html.Props{Class: "mt-3 leading-7 text-slate-300"}, html.Text("Nested settings pages can share their own local shell while still living underneath the broader dashboard route.")),
	)
}

func docsLayout(parseProps router.Attrs) ui.Node {
	parseCurrentPath := router.GetCurrentPath()
	return html.Div(html.Props{Class: "min-h-screen bg-[linear-gradient(180deg,#08111d_0%,#020617_100%)] px-6 py-8 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-6xl"},
			html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4 rounded-[2rem] border border-white/10 bg-slate-950/80 px-6 py-5 shadow-[0_24px_80px_rgba(2,6,23,0.45)]"},
				html.Div(html.Props{},
					html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Docs Layout")),
					html.H1(html.Props{Class: "mt-2 text-3xl font-black text-white"}, html.Text("Guide navigation")),
				),
				html.Div(html.Props{Class: "flex flex-wrap gap-3"},
					routeLink("Home", "/", parseCurrentPath),
					routeLink("Dashboard", "/dashboard/overview", parseCurrentPath),
				),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-6 lg:grid-cols-[280px_minmax(0,1fr)]"},
				html.Aside(html.Props{Class: "space-y-6"},
					layoutSection("Docs", "Persistent docs navigation",
						routeLink("Getting Started", "/docs/getting-started", parseCurrentPath),
						routeLink("Routing", "/docs/routing", parseCurrentPath),
					),
					html.Div(html.Props{Class: "rounded-[2rem] border border-cyan-500/20 bg-cyan-950/35 p-6 text-slate-100 shadow-[0_18px_60px_rgba(2,6,23,0.4)]"},
						html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.24em] text-cyan-300"}, html.Text("Why docs benefit from layouts")),
						html.P(html.Props{Class: "mt-3 leading-7 text-slate-200"}, html.Text("Docs trees usually want a stable sidebar, breadcrumbs, and article frame while the content page changes underneath.")),
					),
				),
				html.Main(html.Props{Class: "space-y-6"}, router.GetOutlet()),
			),
		),
	)
}

func docsGettingStartedPage(parseProps router.Attrs) ui.Node {
	return html.Article(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
		html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Docs Leaf")),
		html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Getting started")),
		html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("Start with a layout route for each major section, then register deeper leaves under that prefix. Parent layouts render their child route explicitly with router.Outlet().")),
	)
}

func docsRoutingPage(parseProps router.Attrs) ui.Node {
	return html.Article(html.Props{Class: "rounded-[2rem] border border-white/10 bg-slate-950/75 p-8 shadow-[0_20px_70px_rgba(2,6,23,0.42)]"},
		html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Docs Leaf")),
		html.H2(html.Props{Class: "mt-3 text-4xl font-black text-white"}, html.Text("Routing rules")),
		html.P(html.Props{Class: "mt-5 max-w-2xl text-lg leading-8 text-slate-300"}, html.Text("Only routes marked with Options{Layout: true} participate as parent layouts. Parent params stay scoped to their matched prefix, while the leaf route sees the full merged param set for the final path.")),
	)
}

func notFoundPage(parseProps router.Attrs) ui.Node {
	return html.Div(html.Props{Class: "min-h-screen bg-[linear-gradient(180deg,#08111d_0%,#020617_100%)] px-6 py-10 text-slate-100"},
		html.Div(html.Props{Class: "mx-auto max-w-4xl rounded-[2.5rem] border border-white/10 bg-slate-950/80 p-10 shadow-[0_24px_80px_rgba(2,6,23,0.45)]"},
			html.P(html.Props{Class: "text-xs font-black uppercase tracking-[0.28em] text-cyan-300"}, html.Text("Not Found")),
			html.H1(html.Props{Class: "mt-3 text-5xl font-black text-white"}, html.Text("That route does not exist.")),
			html.P(html.Props{Class: "mt-5 text-lg leading-8 text-slate-300"}, html.Text("Use the demo links to jump back into the dashboard or docs route trees.")),
			html.Div(html.Props{Class: "mt-8 flex flex-wrap gap-4"},
				routeLink("Dashboard", "/dashboard/overview", router.GetCurrentPath()),
				routeLink("Docs", "/docs/getting-started", router.GetCurrentPath()),
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
	parseR.Mount("#app")

	select {}
}
