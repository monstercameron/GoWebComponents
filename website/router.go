//go:build js && wasm
// +build js,wasm

package website

import (
	. "github.com/monstercameron/GoWebComponents/fiber"
)

// AppRouter configures and manages client-side routing for the application.
// Uses hash-based routing for reliable navigation without server configuration.
// Supports both main website and documentation routes with 404 fallback.
func AppRouter(props Attrs) *Element {
	// Initialize hash router with homepage as default
	router := NewHashRouter(RouterOptions{
		DefaultRoute: "/",
	})

	// Register application routes
	router.GoRegisterRoute("/", DocsWebsite)  // Main personal website
	router.GoRegisterRoute("/docs", DocsPage) // API documentation
	router.GoRegisterRoute("*", NotFoundPage) // 404 fallback for unmatched routes

	// Return active route component (handles re-rendering automatically)
	return router.GoGetRoute()
}
