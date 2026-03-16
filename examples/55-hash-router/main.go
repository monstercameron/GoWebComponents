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

func hashNav() ui.Node {
	return html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
		html.A(html.Props{Href: "#/", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Home")),
		html.A(html.Props{Href: "#/docs", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Docs")),
		html.A(html.Props{Href: "#/pricing", Class: "rounded-full border border-white/10 px-4 py-2 text-sm text-slate-200"}, html.Text("Pricing")),
	)
}

func hashPage(title, summary string) *router.Element {
	inspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		title,
		"router.NewHashRouter / Register / Mount",
		summary,
		shared.ExamplePanel("Hash routes",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Hash routers are the static-hosting default because they work without server rewrite rules.")),
			hashNav(),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", inspection.Path),
				shared.ExampleStat("URL style", "#/path"),
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", func(router.Attrs) *router.Element {
		return hashPage("Hash router basics", "This route is mounted from the window.location.hash value.")
	})
	r.Register("/docs", func(router.Attrs) *router.Element {
		return hashPage("Docs route", "Changing the hash updates the route without any server involvement.")
	})
	r.Register("/pricing", func(router.Attrs) *router.Element {
		return hashPage("Pricing route", "Exact routes and hash links are enough for small static demos and docs sites.")
	})
	r.Register("*", func(router.Attrs) *router.Element {
		return hashPage("Hash 404", "The catch-all route handles unmatched hash paths.")
	})
	r.Mount("#app")
	select {}
}
