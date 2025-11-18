//go:build js && wasm
// +build js,wasm

package website

import (
	"github.com/monstercameron/GoWebComponents/router"
)

// AppRouter configures and manages client-side routing for the application.
// Uses hash-based routing for reliable navigation without server configuration.
// Supports both main website and documentation routes with 404 fallback.
func AppRouter(_ Attrs) *Element {
	// Initialize hash router with homepage as default
	r := router.NewHashRouter(router.RouterOptions{
		DefaultRoute: "/",
	})

	// Register application routes
	r.GoRegisterRoute("/", DocsWebsite)   // Main personal website
	r.GoRegisterRoute("/docs", DocsPage)  // API documentation
	r.GoRegisterRoute("*", NotFoundPage)  // 404 fallback for unmatched routes

	// Return active route component (handles re-rendering automatically)
	return r.GoGetRoute()
}

// GetSiteRouter returns a configured router instance for use outside components
func GetSiteRouter() *router.Router {
	r := router.NewHashRouter(router.RouterOptions{
		DefaultRoute: "/",
	})

	// Register application routes
	r.GoRegisterRoute("/", DocsWebsite)
	r.GoRegisterRoute("/docs", DocsPage)
	r.GoRegisterRoute("*", NotFoundPage)

	return r
}




