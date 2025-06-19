//go:build js && wasm
// +build js,wasm

package website

import (
	. "github.com/monstercameron/GoWebComponents/fiber"
)

// AppRouter sets up and manages the application routing
func AppRouter(props Attrs) *Element {
	// Create a hash router instance with options
	router := NewHashRouter(RouterOptions{
		DefaultRoute: "/",
	})

	// Register routes - library handles all state and navigation
	router.GoRegisterRoute("/", DocsWebsite)
	router.GoRegisterRoute("/docs", DocsPage)

	// Register wildcard route for 404 (catch-all)
	router.GoRegisterRoute("*", NotFoundPage)

	// Return the current route component - library handles re-renders internally
	return router.GoGetRoute()
}
