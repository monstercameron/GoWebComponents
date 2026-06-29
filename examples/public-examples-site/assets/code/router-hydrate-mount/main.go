//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func routeStat(parseLabel, parseValue string) ui.Node {
	return shared.ExampleStat(parseLabel, parseValue)
}

func routeButton(parseLabel string, parseHandler ui.Handler) ui.Node {
	return shared.ExampleButton(parseLabel, parseHandler)
}

func routeShell(parseSection, parseSummary string) *router.Element {
	parseNav := router.UseNavigate()
	return shared.ExamplePage(
		"Router Hydrate Mount",
		"router.HydrateMount",
		"Attach router listeners after hydration so the first route stays intact and later navigation renders normally.",
		shared.ExamplePanel("Route State",
			html.Div(html.Props{Class: "grid gap-3 md:grid-cols-2"},
				routeStat("Current Section", parseSection),
				routeStat("Attach Mode", "HydrateMount"),
			),
			html.P(html.Props{Class: "text-sm leading-6 text-slate-300"}, html.Text(parseSummary)),
		),
		shared.ExamplePanel("Navigate",
			html.Div(html.Props{Class: "flex flex-wrap gap-2"},
				routeButton("Open Overview", ui.UseEvent(func() { parseNav.Navigate("/docs/overview") })),
				routeButton("Open API", ui.UseEvent(func() { parseNav.Navigate("/docs/api") })),
			),
		),
	)
}

func ensureHash(parsePath string) {
	parseWindow := js.Global().Get("window")
	if !parseWindow.Truthy() {
		return
	}
	parseLocation := parseWindow.Get("location")
	if parseLocation.Get("hash").String() == "" {
		parseLocation.Set("hash", "#"+parsePath)
	}
}

func main() {
	utils.DisableAllDebug()
	ensureHash("/docs/overview")

	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/docs/overview"})
	parseR.Register("/docs/:section", func(parseAttrs router.Attrs) *router.Element {
		parseSection := router.UseParams().Get("section")
		parseSummary := "The first route was hydrated first, then the router started listening for future hash changes without forcing a replacement render."
		if parseSection == "api" {
			parseSummary = "You navigated after hydration. HydrateMount only special-cases the initial attach; future route updates render normally."
		}
		return routeShell(parseSection, parseSummary)
	})

	parseRoot := ui.CreateElement(func() ui.Node { return parseR.Current() })
	_, _ = exampleboot.ApplyExampleHydration(parseRoot)
	exampleboot.ApplyExampleHydratedRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
