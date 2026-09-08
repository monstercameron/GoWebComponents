//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/router"
)

// AppRouter configures and manages client-side routing for the application.
// Uses hash-based routing for reliable navigation without server configuration.
// Supports both main website and documentation routes with 404 fallback.
func AppRouter(_ Attrs) *Element {
	// Initialize hash router with homepage as default
	parseR := router.NewHashRouter(router.RouterOptions{
		DefaultRoute: portfolioHomeRoute,
	})

	// Register application routes
	parseR.Register(portfolioHomeRoute, DocsWebsite)                   // Main personal website
	parseR.Register(portfolioDocsRoute, renderDocsPageCompact)         // API documentation
	parseR.Register(portfolioCatchAllRoute, renderNotFoundPageCompact) // 404 fallback for unmatched routes

	// Return active route component (handles re-rendering automatically)
	return parseR.Current()
}

// GetSiteRouter returns a configured router instance for use outside components
func GetSiteRouter() *router.Router {
	parseR := router.NewHashRouter(router.RouterOptions{
		DefaultRoute: portfolioHomeRoute,
	})

	// Register application routes
	parseR.Register(portfolioHomeRoute, DocsWebsite)
	parseR.Register(portfolioDocsRoute, renderDocsPageCompact)
	parseR.Register(portfolioCatchAllRoute, renderNotFoundPageCompact)

	return parseR
}
