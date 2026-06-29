//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func OverviewPage() ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/",
		Title:      "Overview",
		Page:       ui.CreateElement(OverviewContent),
	})
}

func DataPage() ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/data",
		Title:      "Data Lab",
		Page:       ui.CreateElement(DataContent),
	})
}

func DataDetailPage(parseProps Attrs) ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/data",
		Title:      "Record Detail",
		Page:       ui.CreateElement(DataDetailContent, parseProps),
	})
}

func SearchPage(parseProps Attrs) ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/search",
		Title:      "Search",
		Page:       ui.CreateElement(SearchContent, parseProps),
	})
}

func SecurePage(parseProps Attrs) ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/secure",
		Title:      "Secure",
		Page:       ui.CreateElement(SecureContent, parseProps),
	})
}

func PlaygroundPage() ui.Node {
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "/playground",
		Title:      "Playground",
		Page:       ui.CreateElement(PlaygroundContent),
	})
}

func DataDetailRouteLoading(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseSlug, _ := parseProps["id"].(string)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Route Loader")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Loading record detail")),
				html.P(html.Props{Class: "mt-2 text-slate-300"}, html.Text("The router is loading route-scoped data before rendering the detail page.")),
			),
			html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/data") }), Class: "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90"}, html.Text("Back to Data Lab")),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.P(html.Props{Class: "text-sm uppercase tracking-[0.28em] text-slate-400"}, html.Text("Route slug: "+parseSlug)),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("Loading record detail...")),
		),
	)
}

func DataDetailRouteError(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseMessage, _ := parseProps["error"].(string)
	parseSlug, _ := parseProps["id"].(string)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Route Loader Error")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Record detail failed")),
			),
			html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/data") }), Class: "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90"}, html.Text("Back to Data Lab")),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-rose-400/20 bg-rose-400/10 p-8 text-rose-100"},
			html.P(html.Props{Class: "text-sm uppercase tracking-[0.28em] text-rose-200/80"}, html.Text("Route slug: "+parseSlug)),
			html.P(html.Props{Class: "mt-4"}, html.Text(parseMessage)),
		),
	)
}

func SearchRouteLoading(parseProps Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Route Loader")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Loading search results")),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("The router is revalidating search results for the current query.")),
		),
	)
}

func SearchRouteError(parseProps Attrs) ui.Node {
	parseMessage, _ := parseProps["error"].(string)
	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-rose-400/20 bg-rose-400/10 p-8 text-rose-100"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Route Loader Error")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Search route failed")),
			html.P(html.Props{Class: "mt-4"}, html.Text(parseMessage)),
		),
	)
}

func SecureRouteLoading(parseProps Attrs) ui.Node {
	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-8"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Protected Loader")),
			html.H2(html.Props{Class: "mt-3 text-4xl font-black"}, html.Text("Checking session access")),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("The router is checking whether the current request should be allowed into the protected route.")),
		),
	)
}

func SecureRouteError(parseProps Attrs) ui.Node {
	parseNav := router.UseNavigate()
	parseMessage, _ := parseProps["error"].(string)

	return html.Div(html.Props{Class: "space-y-8"},
		html.Div(html.Props{Class: "flex flex-wrap items-center justify-between gap-4"},
			html.Div(html.Props{},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("Protected Loader")),
				html.H2(html.Props{Class: "mt-2 text-3xl font-black"}, html.Text("Access denied")),
			),
			html.Div(html.Props{Class: "flex gap-3"},
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/secure?auth=true&role=admin") }), Class: "rounded-full border border-cyan-900/80 bg-cyan-950/70 px-5 py-3 font-semibold text-cyan-100 hover:bg-cyan-900/80"}, html.Text("Grant demo access")),
				html.Button(html.Props{OnClick: ui.UseEvent(func() { parseNav.Navigate("/") }), Class: "rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold text-slate-200 hover:bg-slate-800/90"}, html.Text("Back home")),
			),
		),
		html.Div(html.Props{Class: "rounded-[2rem] border border-rose-400/20 bg-rose-400/10 p-8 text-rose-100"},
			html.P(html.Props{Class: "mt-1"}, html.Text(parseMessage)),
			html.P(html.Props{Class: "mt-4 text-rose-100/80"}, html.Text("Use the demo access button to reload this protected route with a simulated authenticated session.")),
		),
	)
}

func NotFoundPage() ui.Node {
	parseNav := router.UseNavigate()
	return ui.CreateElement(Shell, ShellProps{
		ActivePath: "*",
		Title:      "Not Found",
		Page: html.Div(html.Props{
			Class: "rounded-[2rem] border border-slate-800/80 bg-[#0d1722]/85 p-10 text-center",
		},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-rose-300"}, html.Text("404")),
			html.H2(html.Props{Class: "mt-4 text-4xl font-black"}, html.Text("Route not found")),
			html.P(html.Props{Class: "mt-4 text-slate-300"}, html.Text("The requested OMI panel does not exist.")),
			html.Button(html.Props{
				OnClick: ui.UseEvent(func() { parseNav.Navigate("/") }),
				Class:   "mt-8 rounded-full border border-slate-700/80 bg-slate-900/80 px-5 py-3 font-semibold hover:bg-slate-800/90",
			}, html.Text("Return home")),
		),
	})
}
