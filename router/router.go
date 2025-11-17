//go:build js && wasm
// +build js,wasm

package router

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/fiber"
)

// Router manages routes and navigation for single-page applications.
type Router = fiber.Router

// Options represents configuration for individual routes.
type Options = fiber.RouteOptions

// NewHashRouter creates a hash-based router that uses URL fragments (e.g., #/about).
// Hash routing works without server configuration and is compatible with all browsers.
//
// Example:
//
//	r := router.NewHashRouter()
//	r.RegisterRoute("/", HomePage)
//	r.RegisterRoute("/about", AboutPage)
func NewHashRouter(options ...fiber.RouterOptions) *Router {
	return fiber.NewHashRouter(options...)
}

// NewRouter creates a history-based router that uses the HTML5 History API.
// This provides clean URLs without hash fragments but requires server configuration
// to handle client-side routes.
//
// Example:
//
//	r := router.NewRouter(router.RouterOptions{
//	    Type: "regular",
//	    DefaultRoute: "/",
//	})
func NewRouter(options fiber.RouterOptions) *Router {
	return fiber.NewRouter(options)
}

// RegisterRoute registers a route with the global router.
// This is a convenience function that uses the global router instance.
//
// The component can be:
//   - A function that returns *fiber.Element
//   - A pre-rendered *fiber.Element
//
// Example:
//
//	router.RegisterRoute("/", HomePage)
//	router.RegisterRoute("/about", AboutPage, router.Options{
//	    Title: "About Us",
//	})
func RegisterRoute(path string, component interface{}, options ...Options) {
	fiber.GoRegisterRoute(path, component, options...)
}

// Navigate navigates to a new route using the global router.
// This adds the new route to browser history.
//
// Example:
//
//	router.Navigate("/about")
func Navigate(path string) {
	fiber.Navigate(path)
}

// NavigateReplace navigates to a new route, replacing the current history entry.
// This is useful for redirects where you don't want the user to go back to the previous page.
//
// Example:
//
//	router.NavigateReplace("/login")
func NavigateReplace(path string) {
	fiber.NavigateReplace(path)
}

// GetCurrentPath returns the current route path from the global router.
func GetCurrentPath() string {
	return fiber.GetCurrentPath()
}

// GetRoute returns the component for the current route as an Element.
// This is useful for rendering the current route in your app.
//
// Example:
//
//	func App(props dom.Attrs) *fiber.Element {
//	    return dom.Div(nil,
//	        NavBar(nil),
//	        router.GetRoute(), // Renders current route
//	        Footer(nil),
//	    )
//	}
func GetRoute() *fiber.Element {
	return fiber.GoGetRoute()
}

// GetRouter returns the global router instance.
// Use this to access router methods on the global router.
func GetRouter() *Router {
	return fiber.GoGetRouter()
}

// RouteWithElement renders a route directly to a DOM element.
// This is a low-level function for manual rendering.
//
// Example:
//
//	elem := js.Global().Get("document").Call("getElementById", "app")
//	router.RouteWithElement("/about", elem)
func RouteWithElement(path string, elemRef js.Value) {
	fiber.RouteWithElement(path, elemRef)
}

// Component is a type alias for component functions used in routing.
type Component = func(dom.Attrs) *fiber.Element
