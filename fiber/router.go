//go:build js && wasm
// +build js,wasm

package fiber

import (
	"strings"
	"syscall/js"
)

// Component represents a component function that takes props and returns an Element
type Component func(map[string]interface{}) interface{}

// RouteOptions represents optional configuration for a route
type RouteOptions struct {
	Title       string            // Page title for this route
	NoHistory   bool              // Don't add to browser history
	BeforeEnter func(string) bool // Return false to prevent navigation
}

// RouteEntry represents a registered route
type RouteEntry struct {
	Path      string
	Component Component
	Options   RouteOptions
}

// Router manages routes and navigation
type Router struct {
	routes            []RouteEntry
	currentPath       string
	onNavigate        func(string) // Callback when navigation occurs
	hashListenerSetup bool         // Track if hash listener is set up
}

// NewRouter creates a new router instance
func NewRouter() *Router {
	return &Router{
		routes:      make([]RouteEntry, 0),
		currentPath: "/",
	}
}

// Register adds a route to the router
func (r *Router) Register(path string, component Component, options ...RouteOptions) {
	// Clean path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Use first options if provided, otherwise defaults
	opts := RouteOptions{}
	if len(options) > 0 {
		opts = options[0]
	}

	route := RouteEntry{
		Path:      path,
		Component: component,
		Options:   opts,
	}

	r.routes = append(r.routes, route)
}

// Route resolves the current path and returns the component
func (r *Router) Route(path string) interface{} {
	// Clean path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Find matching route
	for _, route := range r.routes {
		if route.Path == path {
			// Create props with route info
			props := map[string]interface{}{
				"path": path,
			}
			return route.Component(props)
		}
	}

	// Return 404 component if no route found
	return Div(map[string]interface{}{
		"style": map[string]interface{}{
			"padding":    "2rem",
			"text-align": "center",
		},
	},
		H1(map[string]interface{}{}, "404 - Page Not Found"),
		P(map[string]interface{}{}, "The page you're looking for doesn't exist."),
	)
}

// Navigate to a path
func (r *Router) Navigate(path string) {
	// Clean path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Check if route exists and has beforeEnter guard
	var targetRoute *RouteEntry
	for _, route := range r.routes {
		if route.Path == path {
			targetRoute = &route
			break
		}
	}

	// Call beforeEnter guard if it exists
	if targetRoute != nil && targetRoute.Options.BeforeEnter != nil {
		if !targetRoute.Options.BeforeEnter(path) {
			return // Navigation prevented
		}
	}

	// Update current path
	r.currentPath = path

	// Update browser URL and title
	if targetRoute != nil && !targetRoute.Options.NoHistory {
		js.Global().Get("history").Call("pushState", nil, "", path)
		if targetRoute.Options.Title != "" {
			js.Global().Get("document").Set("title", targetRoute.Options.Title)
		}
	}

	// Call navigation callback
	if r.onNavigate != nil {
		r.onNavigate(path)
	}
}

// NavigateReplace replaces current history entry
func (r *Router) NavigateReplace(path string) {
	// Clean path
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	// Check if route exists and has beforeEnter guard
	var targetRoute *RouteEntry
	for _, route := range r.routes {
		if route.Path == path {
			targetRoute = &route
			break
		}
	}

	// Call beforeEnter guard if it exists
	if targetRoute != nil && targetRoute.Options.BeforeEnter != nil {
		if !targetRoute.Options.BeforeEnter(path) {
			return // Navigation prevented
		}
	}

	// Update current path
	r.currentPath = path

	// Update browser URL and title
	if targetRoute != nil && !targetRoute.Options.NoHistory {
		js.Global().Get("history").Call("replaceState", nil, "", path)
		if targetRoute.Options.Title != "" {
			js.Global().Get("document").Set("title", targetRoute.Options.Title)
		}
	}

	// Call navigation callback
	if r.onNavigate != nil {
		r.onNavigate(path)
	}
}

// GetCurrentPath returns the current path
func (r *Router) GetCurrentPath() string {
	return r.currentPath
}

// SetCurrentPath sets the current path without navigation
func (r *Router) SetCurrentPath(path string) {
	r.currentPath = path
}

// RouteWithElement renders the route for the given path and mounts it to the provided element reference
func (r *Router) RouteWithElement(path string, elemRef js.Value) {
	// Get the component for this path
	component := r.Route(path)

	// Clear the element
	elemRef.Set("innerHTML", "")

	// Render the component to the element
	if component != nil {
		if element, ok := component.(*Element); ok {
			Render(element, elemRef)
		}
	}
}

// OnNavigate sets the navigation callback
func (r *Router) OnNavigate(callback func(string)) {
	r.onNavigate = callback
}

// SetupBrowserSync sets up browser history synchronization
func (r *Router) SetupBrowserSync() {
	// Handle browser back/forward buttons
	js.Global().Call("addEventListener", "popstate", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		path := js.Global().Get("location").Get("pathname").String()
		r.SetCurrentPath(path)
		if r.onNavigate != nil {
			r.onNavigate(path)
		}
		return nil
	}))

	// Set initial path from browser
	currentPath := js.Global().Get("location").Get("pathname").String()
	if currentPath == "" {
		currentPath = "/"
	}
	r.SetCurrentPath(currentPath)
}

// Global router instance for backward compatibility
var globalRouter = NewRouter()

// Register adds a route to the global router
func Register(path string, component Component, options ...RouteOptions) {
	globalRouter.Register(path, component, options...)
}

// Route resolves a path using the global router
func Route(path string) interface{} {
	return globalRouter.Route(path)
}

// Navigate to a path using the global router
func Navigate(path string) {
	globalRouter.Navigate(path)
}

// NavigateReplace replaces current history entry using the global router
func NavigateReplace(path string) {
	globalRouter.NavigateReplace(path)
}

// GetCurrentPath returns the current path from the global router
func GetCurrentPath() string {
	return globalRouter.GetCurrentPath()
}

// SetCurrentPath sets the current path in the global router
func SetCurrentPath(path string) {
	globalRouter.SetCurrentPath(path)
}

// OnNavigate sets the navigation callback for the global router
func OnNavigate(callback func(string)) {
	globalRouter.OnNavigate(callback)
}

// SetupBrowserSync sets up browser history synchronization for the global router
func SetupBrowserSync() {
	globalRouter.SetupBrowserSync()
}

// GetRouter returns the global router instance
func GetRouter() *Router {
	return globalRouter
}

// GoGetRouter returns the global router instance (Go-style naming)
func GoGetRouter() *Router {
	return globalRouter
}

// GoGetRoute returns the component for the current route
func GoGetRoute() interface{} {
	return globalRouter.Route(globalRouter.currentPath)
}

// GoRegisterRoute registers a route with the global router (Go-style naming)
// Accepts component functions directly like DocsPage or DocsPage(nil)
func GoRegisterRoute(path string, component interface{}, options ...RouteOptions) {
	// Handle different component types
	var wrappedComponent Component

	switch comp := component.(type) {
	case func(Attrs) *Element:
		// Component function with Attrs - wrap it
		wrappedComponent = func(props map[string]interface{}) interface{} {
			return comp(props)
		}
	case *Element:
		// Pre-called component (like DocsPage(nil)) - wrap it
		wrappedComponent = func(props map[string]interface{}) interface{} {
			return comp
		}
	case Component:
		// Already correct type
		wrappedComponent = comp
	default:
		// Fallback - assume it's a function that returns *Element
		wrappedComponent = func(props map[string]interface{}) interface{} {
			return component
		}
	}

	globalRouter.Register(path, wrappedComponent, options...)

	// Set up hash change listener if not already set up
	if !globalRouter.hashListenerSetup {
		globalRouter.setupHashListener()
		globalRouter.hashListenerSetup = true
	}
}

// setupHashListener sets up hash change handling at the library level
func (r *Router) setupHashListener() {
	// Handle hash change events
	hashChangeHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		hash := js.Global().Get("location").Get("hash").String()

		var path string
		if hash == "#/docs" {
			path = "/docs"
		} else {
			path = "/" // Default to home for #/ or empty hash
		}

		// Navigate using the router
		r.Navigate(path)

		return nil
	})

	// Add hash change listener
	js.Global().Call("addEventListener", "hashchange", hashChangeHandler)

	// Handle initial route from hash
	initialHash := js.Global().Get("location").Get("hash").String()
	var initialPath string
	if initialHash == "#/docs" {
		initialPath = "/docs"
	} else {
		initialPath = "/"
	}

	// Set initial route
	r.Navigate(initialPath)
}

// RouteWithElement renders the route for the given path using the global router
func RouteWithElement(path string, elemRef js.Value) {
	globalRouter.RouteWithElement(path, elemRef)
}
