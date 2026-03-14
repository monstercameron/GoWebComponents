// Package router provides client-side routing for single-page applications
// built with GoWebComponents.
//
// This package implements hash-based and history-based routing with support for:
//   - Multiple router instances
//   - Browser history integration
//   - Programmatic navigation
//   - Static node and component route registration
//
// Basic usage with hash routing:
//
//	import (
//	    "github.com/monstercameron/GoWebComponents/html"
//	    "github.com/monstercameron/GoWebComponents/router"
//	    "github.com/monstercameron/GoWebComponents/ui"
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
package router
