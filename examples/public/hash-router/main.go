//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func hashNav() ui.Node {
	return html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
		html.A(html.Props{Href: "#/", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Home")),
		html.A(html.Props{Href: "#/docs", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Docs")),
		html.A(html.Props{Href: "#/pricing", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Pricing")),
	)
}

func hashPage(parseTitle, parseSummary string) *router.Element {
	parseInspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		parseTitle,
		"router.NewHashRouter / Register / Mount",
		parseSummary,
		shared.ExamplePanel("Hash routes",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Hash routers are the static-hosting default because they work without server rewrite rules.")),
			hashNav(),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", parseInspection.Path),
				shared.ExampleStat("URL style", "#/path"),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", func(router.Attrs) *router.Element {
		return hashPage("Hash router basics", "This route is mounted from the window.location.hash value.")
	})
	parseR.Register("/docs", func(router.Attrs) *router.Element {
		return hashPage("Docs route", "Changing the hash updates the route without any server involvement.")
	})
	parseR.Register("/pricing", func(router.Attrs) *router.Element {
		return hashPage("Pricing route", "Exact routes and hash links are enough for small static demos and docs sites.")
	})
	parseR.Register("*", func(router.Attrs) *router.Element {
		return hashPage("Hash 404", "The catch-all route handles unmatched hash paths.")
	})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
