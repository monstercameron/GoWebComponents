//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/router"
)

// AppRouter configures and manages client-side routing for the application.
// Uses hash-based routing for reliable navigation without server configuration.
// Supports both main website and documentation routes with 404 fallback.
func AppRouter(_ Attrs) *Element {
	// Initialize hash router with homepage as default
	r := router.NewHashRouter(router.RouterOptions{
		DefaultRoute: portfolioHomeRoute,
	})

	// Register application routes
	r.Register(portfolioHomeRoute, DocsWebsite)      // Main personal website
	r.Register(portfolioDocsRoute, DocsPage)         // API documentation
	r.Register(portfolioCatchAllRoute, NotFoundPage) // 404 fallback for unmatched routes

	// Return active route component (handles re-rendering automatically)
	return r.Current()
}

// GetSiteRouter returns a configured router instance for use outside components
func GetSiteRouter() *router.Router {
	r := router.NewHashRouter(router.RouterOptions{
		DefaultRoute: portfolioHomeRoute,
	})

	// Register application routes
	r.Register(portfolioHomeRoute, DocsWebsite)
	r.Register(portfolioDocsRoute, DocsPage)
	r.Register(portfolioCatchAllRoute, NotFoundPage)

	return r
}
