//go:build js && wasm
// +build js,wasm

package router_test

import (
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/router"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ExampleNewHashRouter() {
	// Create a new hash-based router
	parseR := router.NewHashRouter()

	// Define page components
	parseHomePage := func(parseProps router.Attrs) *router.Element {
		return html.Div(html.Props{}, html.H1(html.Props{}, html.Text("Home")))
	}

	parseAboutPage := func(parseProps2 router.Attrs) *router.Element {
		return html.Div(html.Props{}, html.H1(html.Props{}, html.Text("About")))
	}

	// Register routes
	parseR.GoRegisterRoute("/", parseHomePage)
	parseR.GoRegisterRoute("/about", parseAboutPage)

	// In a real app, you would mount the router to the DOM
	// r.Mount("#app")
}

func ExampleNavigate() {
	// Navigate to a new path
	// This works with both hash and history routers
	router.Navigate("/about")
}

func Example_navigationLink() {
	// To create a link that navigates without full page reload,
	// use a component with an onclick handler and ui.UseEvent.

	// Define a component
	parseLinkComponent := func(parseProps router.Attrs) *router.Element {
		handleClick := ui.UseEvent(func(parseEvent ui.Event) {
			parseEvent.PreventDefault()
			router.Navigate("/about")
		})

		return html.A(
			html.Props{
				Href:    "#",
				OnClick: handleClick,
			},
			html.Text("Go to About"),
		)
	}

	// In a real app, you would render this component with ui.Render.

	// For this example, we just suppress the unused variable warning
	_ = parseLinkComponent
}
