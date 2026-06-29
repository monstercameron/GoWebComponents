package router_test

import (
	"github.com/monstercameron/GoWebComponents/router"
)

// ExampleDefineRoute shows declaring a typed route contract, which validates the
// pattern up front and lets the app build type-safe hrefs for the route.
func ExampleDefineRoute() {
	parseUserRoute := router.MustDefineRoute("/users/:id")
	// parseUserRoute.Path(...) builds a concrete URL for the :id parameter.
	_ = parseUserRoute
}
