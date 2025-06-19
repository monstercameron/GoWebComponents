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

// RouterOptions represents configuration for router instantiation
type RouterOptions struct {
	Type         string // "hash" or "regular"
	DefaultRoute string // Default route to use
	BasePath     string // Base path for regular router
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
	onNavigate        func(string)  // Callback when navigation occurs
	hashListenerSetup bool          // Track if hash listener is set up
	options           RouterOptions // Router configuration
}

// NewRouter creates a new router instance with options
func NewRouter(options RouterOptions) *Router {
	// Set defaults
	if options.DefaultRoute == "" {
		options.DefaultRoute = "/"
	}
	if options.Type == "" {
		options.Type = "regular"
	}

	router := &Router{
		routes:      make([]RouteEntry, 0),
		currentPath: options.DefaultRoute,
		options:     options,
	}

	// Set up navigation handling based on type
	if options.Type == "hash" {
		router.setupHashListener()
		router.hashListenerSetup = true
	}

	return router
}

// NewHashRouter creates a hash-based router
func NewHashRouter(options ...RouterOptions) *Router {
	opts := RouterOptions{Type: "hash", DefaultRoute: "/"}
	if len(options) > 0 {
		opts = options[0]
		opts.Type = "hash" // Force hash type
	}
	router := NewRouter(opts)

	// Auto-make hash routers global since they're typically the main app router
	router.MakeGlobal()

	return router
}

// NewRegularRouter creates a regular history-based router
func NewRegularRouter(options ...RouterOptions) *Router {
	opts := RouterOptions{Type: "regular", DefaultRoute: "/"}
	if len(options) > 0 {
		opts = options[0]
		opts.Type = "regular" // Force regular type
	}
	return NewRouter(opts)
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

	var wildcardRoute *RouteEntry

	// Find matching route
	for _, route := range r.routes {
		if route.Path == path {
			// Exact match found
			props := map[string]interface{}{
				"path": path,
			}
			return route.Component(props)
		} else if route.Path == "*" || route.Path == "/*" {
			// Store wildcard route as fallback
			wildcardRoute = &route
		}
	}

	// Use wildcard route if found
	if wildcardRoute != nil {
		props := map[string]interface{}{
			"path": path,
		}
		return wildcardRoute.Component(props)
	}

	// Return default 404 component if no route found
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

	// Update browser URL and title based on router type
	if targetRoute != nil && !targetRoute.Options.NoHistory {
		if r.options.Type == "hash" {
			// For hash router, update the hash
			var hash string
			if path == "/docs" {
				hash = "#/docs"
			} else {
				hash = "#/"
			}
			js.Global().Get("location").Set("hash", hash)
		} else {
			// For regular router, use pushState
			js.Global().Get("history").Call("pushState", nil, "", path)
		}

		// Update title
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

	// Update browser URL and title based on router type
	if targetRoute != nil && !targetRoute.Options.NoHistory {
		if r.options.Type == "hash" {
			// For hash router, update the hash
			var hash string
			if path == "/docs" {
				hash = "#/docs"
			} else {
				hash = "#/"
			}
			js.Global().Get("location").Set("hash", hash)
		} else {
			// For regular router, use replaceState
			js.Global().Get("history").Call("replaceState", nil, "", path)
		}

		// Update title
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

// GetRouterType returns the type of router (hash or regular)
func (r *Router) GetRouterType() string {
	return r.options.Type
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

// GoRegisterRoute registers a route with this router instance
func (r *Router) GoRegisterRoute(path string, component interface{}, options ...RouteOptions) {
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

	r.Register(path, wrappedComponent, options...)

	// Set up hash change listener if not already set up and this is a hash router
	if r.options.Type == "hash" && !r.hashListenerSetup {
		r.setupHashListener()
		r.hashListenerSetup = true
	}
}

// GoGetRoute returns the component for the current route wrapped in a stateful component
func (r *Router) GoGetRoute() *Element {
	// Return a component that manages state internally
	routerComponent := func(props Attrs) *Element {
		// State to trigger re-renders when route changes
		currentRoute, setCurrentRoute := GoUseState(r.currentPath)

		// Set up navigation callback to trigger re-renders
		GoUseEffect(func() {
			r.OnNavigate(func(path string) {
				setCurrentRoute(path)
			})
			// Set initial route
			setCurrentRoute(r.GetCurrentPath())
			return
		})

		// Use the state to ensure re-renders happen
		_ = currentRoute()

		// Get and return the actual route component
		result := r.Route(r.currentPath)
		if element, ok := result.(*Element); ok {
			return element
		}
		// Fallback to empty div if something goes wrong
		return Div(nil, "Route not found")
	}

	// Return the component instance
	return routerComponent(nil)
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

// Global router instance for backward compatibility (deprecated)
var globalRouter = NewRouter(RouterOptions{Type: "hash", DefaultRoute: "/"})

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

// SetGlobalRouter replaces the global router instance
func SetGlobalRouter(router *Router) {
	globalRouter = router
}

// MakeGlobal makes this router instance the global router
func (r *Router) MakeGlobal() {
	globalRouter = r
}

// GoGetRoute returns the component for the current route
func GoGetRoute() *Element {
	result := globalRouter.Route(globalRouter.currentPath)
	if element, ok := result.(*Element); ok {
		return element
	}
	// Fallback to empty div if something goes wrong
	return Div(nil, "Route not found")
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
	// Auto-redirect from base URL to hash URL for hash router
	currentHash := js.Global().Get("location").Get("hash").String()
	currentPath := js.Global().Get("location").Get("pathname").String()

	// If we're on the base path (/) with no hash, redirect to /#/
	if currentPath == "/" && currentHash == "" {
		js.Global().Get("location").Set("hash", "#/")
		return // The hash change will trigger the handler
	}

	// Handle hash change events (for back/forward button and direct hash changes)
	hashChangeHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		hash := js.Global().Get("location").Get("hash").String()

		var path string
		if hash == "#/docs" {
			path = "/docs"
		} else if hash == "#/" || hash == "" {
			path = "/" // Home route
		} else if strings.HasPrefix(hash, "#/") {
			// Extract path from hash (remove #)
			path = hash[1:] // Remove the # to get the path
		} else {
			path = "/" // Default fallback
		}

		// Update internal router state WITHOUT updating browser URL
		// (since the hash change already happened via back button or direct navigation)
		r.navigateInternal(path)

		return nil
	})

	// Add hash change listener
	js.Global().Call("addEventListener", "hashchange", hashChangeHandler)

	// Handle initial route from hash using same logic as hash change handler
	initialHash := js.Global().Get("location").Get("hash").String()
	var initialPath string
	if initialHash == "#/docs" {
		initialPath = "/docs"
	} else if initialHash == "#/" || initialHash == "" {
		initialPath = "/" // Home route
	} else if strings.HasPrefix(initialHash, "#/") {
		// Extract path from hash (remove #)
		initialPath = initialHash[1:] // Remove the # to get the path
	} else {
		initialPath = "/" // Default fallback
	}

	// Set initial route without updating URL (since we're reading from current hash)
	r.navigateInternal(initialPath)
}

// navigateInternal updates router state without modifying browser URL
func (r *Router) navigateInternal(path string) {
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

	// Update page title only
	if targetRoute != nil && targetRoute.Options.Title != "" {
		js.Global().Get("document").Set("title", targetRoute.Options.Title)
	}

	// Call navigation callback to trigger re-render
	if r.onNavigate != nil {
		r.onNavigate(path)
	}
}

// RouteWithElement renders the route for the given path using the global router
func RouteWithElement(path string, elemRef js.Value) {
	globalRouter.RouteWithElement(path, elemRef)
}
