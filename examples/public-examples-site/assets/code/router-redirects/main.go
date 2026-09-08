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

func redirectsPageView() ui.Node {
	parseNav := router.UseNavigate()
	parseInspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		"router redirects",
		"Send a matched route to a different destination declaratively",
		"Redirects belong in route options when the route itself is just legacy or alias surface. The router swaps in the target path without needing a special component body.",
		shared.ExamplePanel("Redirect flow",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Go to legacy route", ui.UseEvent(func() { parseNav.Navigate("/legacy-dashboard") })),
				shared.ExampleButton("Go to modern route", ui.UseEvent(func() { parseNav.Navigate("/dashboard") })),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Current path", parseInspection.Path),
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
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/dashboard"})
	parseR.Register("/legacy-dashboard", func(router.Attrs) *router.Element {
		return redirectsPage(nil)
	}, router.Options{Redirect: "/dashboard"})
	parseR.Register("/dashboard", redirectsPage)
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
