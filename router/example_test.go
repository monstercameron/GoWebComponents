//go:build js && wasm
// +build js,wasm

package router_test

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
	"github.com/monstercameron/GoWebComponents/router"
)

func ExampleNewHashRouter() {
	// Create a new hash-based router
	r := router.NewHashRouter()

	// Define page components
	homePage := func(props dom.Attrs) *render.Element {
		return dom.Div(nil, dom.H1(nil, dom.Text("Home")))
	}

	aboutPage := func(props dom.Attrs) *render.Element {
		return dom.Div(nil, dom.H1(nil, dom.Text("About")))
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

func ExampleLink() {
	// To create a link that navigates without full page reload,
	// use a component with an onclick handler and hooks.GoUseFunc.

	// Define a component
	LinkComponent := func(props dom.Attrs) *render.Element {
		// Use GoUseFunc to create a memory-safe event handler
		handleClick := hooks.GoUseFunc(func(this js.Value, args []js.Value) interface{} {
			args[0].Call("preventDefault")
			router.Navigate("/about")
			return nil
		})

		return dom.A(
			dom.Attrs{
				"href":    "#",
				"onclick": handleClick,
			},
			dom.Text("Go to About"),
		)
	}

	// In a real app, you would render this component:
	// render.To(LinkComponent(nil), "#app")

	// For this example, we just suppress the unused variable warning
	_ = LinkComponent
}
