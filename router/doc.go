// Package router provides client-side routing for single-page applications
// built with GoWebComponents.
//
// This package implements hash-based and history-based routing with support for:
//   - Multiple router instances
//   - Browser history integration
//   - Programmatic navigation
//   - Nested layout routes with explicit outlets
//   - Static node and component route registration
//   - Route-level chunk gates for lazy route artifacts
//
// Basic usage with hash routing:
//
//	import (
//	    "github.com/monstercameron/GoWebComponents/v4/html"
//	    "github.com/monstercameron/GoWebComponents/v4/router"
//	    "github.com/monstercameron/GoWebComponents/v4/ui"
//	)
//
//	func main() {
//	    r := router.NewHashRouter()
//
//	    r.Register("/", HomePage)
//	    r.Register("/about", AboutPage)
//	    r.Register("/contact", ContactPage)
//	    r.Register("*", NotFoundPage)
//
//	    r.Mount("#app")
//	}
//
// Route components are regular components:
//
//	func HomePage(props router.Attrs) *router.Element {
//	    goAbout := ui.UseEvent(func() {
//	        router.Navigate("/about")
//	    })
//
//	    return html.Div(html.Props{},
//	        html.H1(html.Props{}, html.Text("Welcome Home")),
//	        html.Button(html.Props{OnClick: goAbout}, html.Text("Go to About")),
//	    )
//	}
//
// The package supports both hash-based routing (using URL fragments like #/about)
// and history-based routing (using the HTML5 History API).
//
// Convenience accessors are also available for routed components:
//
//	nav := router.UseNavigate()
//	nav.Navigate("/about")
//
//	query := router.UseQuery()
//	searchTerm := query.Get("q")
//
//	params := router.UseParams()
//	userID := params.Get("id")
//	id, ok := params.Int("id")
//
//	revalidator := router.UseRevalidator()
//	revalidator.Revalidate()
//
//	layout := router.GetOutlet()
//
// Parent layout routes opt in with router.Options{Layout: true} and render the
// active child route through router.GetOutlet().
//
// Routes can also define async loaders that provide route-scoped data before
// rendering the final page component:
//
//	r.Register("/users/:id", UserProfile, router.Options{
//	    Loader: func(ctx context.Context, routeCtx router.RouteContext) (router.Attrs, error) {
//	        return router.Attrs{"userID": routeCtx.Params.Get("id")}, nil
//	    },
//	})
//
// Use RegisterLazy with RouteChunk when route code or a route-owned script must
// load before the route component and loader run.
package router
