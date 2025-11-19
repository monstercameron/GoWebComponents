// Package router provides client-side routing for single-page applications
// built with GoWebComponents.
//
// This package implements hash-based and history-based routing with support for:
//   - Multiple router instances
//   - Route guards (beforeEnter)
//   - Browser history integration
//   - Programmatic navigation
//   - Route-specific page titles
//
// Basic usage with hash routing:
//
//	import "github.com/monstercameron/GoWebComponents/router"
//
//	func main() {
//	    r := router.NewHashRouter()
//
//	    r.RegisterRoute("/", HomePage)
//	    r.RegisterRoute("/about", AboutPage)
//	    r.RegisterRoute("/contact", ContactPage)
//	    r.RegisterRoute("*", NotFoundPage) // Wildcard for 404
//
//	    // Render the current route
//	    render.To(r.GetRoute(), "#app")
//	}
//
// Route components are regular components:
//
//	func HomePage(props dom.Attrs) *fiber.Element {
//	    return dom.Div(nil,
//	        dom.H1(nil, "Welcome Home"),
//	        dom.A(map[string]interface{}{
//	            "onclick": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	                router.Navigate("/about")
//	                return nil
//	            }),
//	        }, "Go to About"),
//	    )
//	}
//
// Route options:
//
//	r.RegisterRoute("/admin", AdminPanel, router.Options{
//	    Title: "Admin Panel",
//	    BeforeEnter: func(path string) bool {
//	        return isAuthenticated() // Return false to prevent navigation
//	    },
//	})
//
// The package supports both hash-based routing (using URL fragments like #/about)
// and history-based routing (using the HTML5 History API).
package router
