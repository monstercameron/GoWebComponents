//go:build js && wasm
// +build js,wasm

package main

import (
	"net/url"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func queryPage(_ router.Attrs) *router.Element {
	query := router.UseQuery()
	search := router.UseSearchParams()

	setGo := ui.UseEvent(func() { search.Set("q", "go") })
	setWasm := ui.UseEvent(func() { search.Set("q", "wasm") })
	setSort := ui.UseEvent(func() { search.Replace("sort", "stars") })
	clearSort := ui.UseEvent(func() { search.Delete("sort") })
	resetAll := ui.UseEvent(func() {
		values := url.Values{}
		values.Set("q", "router")
		values.Set("page", "2")
		search.Navigate(values)
	})

	return shared.ExamplePage(
		"router.UseQuery / router.UseSearchParams",
		"Read and update query strings without changing the route path",
		"UseQuery is the read-only view. UseSearchParams layers navigation helpers on top so filters, sorting, and pagination stay tied to the current route.",
		shared.ExamplePanel("Query controls",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("q=go", setGo),
				shared.ExampleButton("q=wasm", setWasm),
				shared.ExampleButton("Replace sort=stars", setSort),
				shared.ExampleButton("Delete sort", clearSort),
				shared.ExampleButton("Reset query set", resetAll),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
				shared.ExampleStat("q", query.Get("q")),
				shared.ExampleStat("sort", query.Get("sort")),
				shared.ExampleStat("page", query.Get("page")),
				shared.ExampleStat("encoded", query.Encode()),
			),
			shared.ExampleCode(
				`query := router.UseQuery()`,
				`search := router.UseSearchParams()`,
				`search.Set("q", "go")`,
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/search"})
	r.Register("/search", queryPage)
	r.Mount("#app")
	select {}
}