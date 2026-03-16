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

func navigateHomeView() ui.Node {
	nav := router.UseNavigate()
	return shared.ExamplePage(
		"router.UseNavigate",
		"Push or replace routes from event handlers",
		"UseNavigate is the imperative routing handle. It keeps navigation logic inside normal UI events instead of relying only on links.",
		shared.ExamplePanel("Navigation actions",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				shared.ExampleButton("Navigate to review", ui.UseEvent(func() { nav.Navigate("/review") })),
				shared.ExampleButton("Replace with final", ui.UseEvent(func() { nav.Replace("/final") })),
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

func navigateLeafView(props navigateLeafProps) ui.Node {
	nav := router.UseNavigate()
	inspection := router.InspectCurrentRoute()
	return shared.ExamplePage(
		props.Title,
		"Imperative routing handle",
		props.Summary,
		shared.ExamplePanel("Current route",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Path", inspection.Path),
				shared.ExampleStat("API", "router.UseNavigate"),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Back home", ui.UseEvent(func() { nav.Navigate("/") })),
				shared.ExampleButton("Replace home", ui.UseEvent(func() { nav.Replace("/") })),
			),
		),
	)
}

func navigateLeaf(title, summary string) *router.Element {
	return ui.CreateElement(navigateLeafView, navigateLeafProps{Title: title, Summary: summary})
}

func main() {
	utils.DisableAllDebug()
	r := router.NewHashRouter(router.RouterOptions{DefaultRoute: "/"})
	r.Register("/", navigateHome)
	r.Register("/review", func(router.Attrs) *router.Element {
		return navigateLeaf("Review route", "Navigate pushed this route as a new entry.")
	})
	r.Register("/final", func(router.Attrs) *router.Element {
		return navigateLeaf("Final route", "Replace swapped the current entry instead of pushing a new one.")
	})
	r.Mount("#app")
	select {}
}
