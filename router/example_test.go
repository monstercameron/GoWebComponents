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
	r := router.NewHashRouter()

	// Define page components
	homePage := func(props router.Attrs) *router.Element {
		return html.Div(html.Props{}, html.H1(html.Props{}, html.Text("Home")))
	}

	aboutPage := func(props router.Attrs) *router.Element {
		return html.Div(html.Props{}, html.H1(html.Props{}, html.Text("About")))
	}

	// Register routes
	r.GoRegisterRoute("/", homePage)
	r.GoRegisterRoute("/about", aboutPage)

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
	LinkComponent := func(props router.Attrs) *router.Element {
		handleClick := ui.UseEvent(func(event ui.Event) {
			event.PreventDefault()
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
	_ = LinkComponent
}
