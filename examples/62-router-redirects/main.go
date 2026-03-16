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

func redirectsPageView() ui.Node {
	nav := router.UseNavigate()
	inspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		"router redirects",
		"Send a matched route to a different destination declaratively",
		"Redirects belong in route options when the route itself is just legacy or alias surface. The router swaps in the target path without needing a special component body.",
		shared.ExamplePanel("Redirect flow",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Go to legacy route", ui.UseEvent(func() { nav.Navigate("/legacy-dashboard") })),
				shared.ExampleButton("Go to modern route", ui.UseEvent(func() { nav.Navigate("/dashboard") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", inspection.Path),
				shared.ExampleStat("Legacy target", "/legacy-dashboard -> /dashboard"),
			),
			shared.ExampleCode(
				`r.Register("/legacy-dashboard", LegacyPage, router.Options{Redirect: "/dashboard"})`,
			),
		),
	)
}

func redirectsPage(_ router.Attrs) *router.Element {
	return ui.CreateElement(redirectsPageView)
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/dashboard"})
	r.Register("/legacy-dashboard", func(router.Attrs) *router.Element {
		return redirectsPage(nil)
	}, router.Options{Redirect: "/dashboard"})
	r.Register("/dashboard", redirectsPage)
	r.Mount("#app")
	select {}
}
