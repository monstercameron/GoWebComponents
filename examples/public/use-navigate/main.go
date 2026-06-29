//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/router"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

func navigateHomeView() ui.Node {
	parseNav := router.UseNavigate()
	return shared.ExamplePage(
		"router.UseNavigate",
		"Push or replace routes from event handlers",
		"UseNavigate is the imperative routing handle. It keeps navigation logic inside normal UI events instead of relying only on links.",
		shared.ExamplePanel("Navigation actions",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Navigate to review", ui.UseEvent(func() { parseNav.Navigate("/review") })),
				shared.ExampleButton("Replace with final", ui.UseEvent(func() { parseNav.Replace("/final") })),
			),
			shared.ExampleCode(
				`nav := router.UseNavigate()`,
				`nav.Navigate("/review")`,
				`nav.Replace("/final")`,
			),
		),
	)
}

func navigateHome(_ router.Attrs) *router.Element {
	return ui.CreateElement(navigateHomeView)
}

type navigateLeafProps struct {
	Title   string
	Summary string
}

func navigateLeafView(parseProps navigateLeafProps) ui.Node {
	parseNav := router.UseNavigate()
	parseInspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		parseProps.Title,
		"Imperative routing handle",
		parseProps.Summary,
		shared.ExamplePanel("Current route",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Path", parseInspection.Path),
				shared.ExampleStat("API", "router.UseNavigate"),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Back home", ui.UseEvent(func() { parseNav.Navigate("/") })),
				shared.ExampleButton("Replace home", ui.UseEvent(func() { parseNav.Replace("/") })),
			),
		),
	)
}

func navigateLeaf(parseTitle, parseSummary string) *router.Element {
	return ui.CreateElement(navigateLeafView, navigateLeafProps{Title: parseTitle, Summary: parseSummary})
}

func main() {
	utils.DisableAllDebug()
	parseR := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	parseR.Register("/", navigateHome)
	parseR.Register("/review", func(router.Attrs) *router.Element {
		return navigateLeaf("Review route", "Navigate pushed this route as a new entry.")
	})
	parseR.Register("/final", func(router.Attrs) *router.Element {
		return navigateLeaf("Final route", "Replace swapped the current entry instead of pushing a new one.")
	})
	exampleboot.RenderExampleRouter(parseR)
	exampleboot.WaitExampleRuntime()
}
