//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"net/url"

	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

func queryPageView() ui.Node {
	parseQuery := router.UseQuery()
	parseSearch := router.UseSearchParams()

	setGo := ui.UseEvent(func() { parseSearch.Set("q", "go") })
	setWasm := ui.UseEvent(func() { parseSearch.Set("q", "wasm") })
	setSort := ui.UseEvent(func() { parseSearch.Replace("sort", "stars") })
	clearSort := ui.UseEvent(func() { parseSearch.Delete("sort") })
	resetAll := ui.UseEvent(func() {
		parseValues := url.Values{}
		parseValues.Set("q", "router")
		parseValues.Set("page", "2")
		parseSearch.Navigate(parseValues)
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
				shared.ExampleStat("q", parseQuery.Get("q")),
				shared.ExampleStat("sort", parseQuery.Get("sort")),
				shared.ExampleStat("page", parseQuery.Get("page")),
				shared.ExampleStat("encoded", parseQuery.Encode()),
			),
			shared.ExampleCode(
				`query := router.UseQuery()`,
				`search := router.UseSearchParams()`,
				`search.Set("q", "go")`,
			),
		),
	)
}

func queryPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(queryPageView)
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/search"})
	parseR.Register("/search", queryPage)
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
