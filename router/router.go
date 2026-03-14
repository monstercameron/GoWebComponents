//go:build js && wasm
// +build js,wasm

package router

import (
	"reflect"
	"strings"
	"sync"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
)

// Component is a type alias for component functions used in routing.
type Component = func(dom.Attrs) *render.Element

// Options represents configuration for individual routes.
// Placeholder for future per-route settings (e.g., titles, guards).
type Options struct {
	Title string
}

// RouterOptions configures router defaults.
type RouterOptions struct {
	DefaultRoute string
}

// Router manages routes and navigation for single-page applications.
type Router struct {
	routes         map[string]routeFactory
	defaultRoute   string
	notFound       routeFactory
	targetSelector string
	targetElement  js.Value
	listening      bool
	routerType     string // "hash" or "history"
}

type routeFactory func() *render.Element

// NewHashRouter creates a hash-based router that reads from window.location.hash.
func NewHashRouter(options ...RouterOptions) *Router {
	cfg := RouterOptions{DefaultRoute: "/"}
	if len(options) > 0 {
		cfg = options[0]
	}

	return &Router{
		routes:       make(map[string]routeFactory),
		defaultRoute: cfg.DefaultRoute,
		routerType:   "hash",
	}
}

// NewRouter creates a history-based router using the HTML5 History API.
// This router uses window.location.pathname instead of hash fragments.
// Requires server to redirect all routes to the app's entry point.
func NewRouter(options RouterOptions) *Router {
	// Use provided options or defaults
	if options.DefaultRoute == "" {
		options.DefaultRoute = "/"
	}

	router := &Router{
		routes:       make(map[string]routeFactory),
		defaultRoute: options.DefaultRoute,
		routerType:   "history",
	}

	// Setup browser sync for history-based navigation
	router.setupHistoryListener()

	return router
}

// setupHistoryListener sets up History API listeners for popstate events
func (r *Router) setupHistoryListener() {
	// Handler for browser back/forward buttons
	popstateHandler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r.renderCurrentRoute()
		return nil
	})

	js.Global().Get("window").Call("addEventListener", "popstate", popstateHandler)

	// Clean up on unload
	registerCleanup(popstateHandler)
}

// GoRegisterRoute registers a route on the router instance.
func (r *Router) GoRegisterRoute(path string, component interface{}, options ...Options) {
	r.Register(path, component, options...)
}

// Register registers a route using either a component function or a static node.
func (r *Router) Register(path string, component interface{}, options ...Options) {
	if r.routes == nil {
		r.routes = make(map[string]routeFactory)
	}

	normalize := path
	if normalize == "" {
		normalize = "/"
	}

	r.routes[normalize] = makeRouteFactory(component)

	if normalize == "*" {
		r.notFound = r.routes[normalize]
	}
}

// GoGetRoute returns the element for the current route.
func (r *Router) GoGetRoute() *render.Element {
	return r.Current()
}

// Current returns the current route element.
func (r *Router) Current() *render.Element {
	path := r.GetCurrentRouterPath()
	if path == "" {
		path = r.defaultRoute
	}

	if comp, ok := r.routes[path]; ok {
		return comp()
	}

	if r.notFound != nil {
		return r.notFound()
	}

	if r.defaultRoute != "" {
		if comp, ok := r.routes[r.defaultRoute]; ok {
			return comp()
		}
	}

	return dom.Div(nil, dom.Text("Route not found"))
}

// Mount renders the router into a DOM node selected by CSS selector and wires hashchange listeners.
func (r *Router) Mount(selector string) {
	r.targetSelector = selector
	r.targetElement = js.Null()
	r.renderCurrentRoute()
	r.ensureListener()
}

// MountElement renders the router into an existing DOM element reference.
func (r *Router) MountElement(elem js.Value) {
	r.targetElement = elem
	r.targetSelector = ""
	r.renderCurrentRoute()
	r.ensureListener()
}

func (r *Router) renderCurrentRoute() {
	routeElement := r.GoGetRoute()
	switch {
	case r.targetSelector != "":
		render.To(routeElement, r.targetSelector)
	case r.targetElement.Truthy():
		render.ToElement(routeElement, r.targetElement)
	}
}

func (r *Router) ensureListener() {
	if r.listening {
		return
	}
	r.listening = true

	handler := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		r.renderCurrentRoute()
		return nil
	})
	js.Global().Get("window").Call("addEventListener", "hashchange", handler)
	registerCleanup(handler)
}

// RegisterRoute registers a route with the global router.
func RegisterRoute(path string, component interface{}, options ...Options) {
	GetRouter().Register(path, component, options...)
}

// GetCurrentRouterPath returns the current path from a router instance based on its type.
func (r *Router) GetCurrentRouterPath() string {
	loc := js.Global().Get("location")
	if !loc.Truthy() {
		return "/"
	}

	if r.routerType == "history" {
		// For history router, use pathname
		pathname := loc.Get("pathname").String()
		if pathname == "" || pathname == "/" {
			return "/"
		}
		return pathname
	}

	// For hash router, use hash fragment
	hash := strings.TrimPrefix(loc.Get("hash").String(), "#")
	if hash == "" || hash == "/" || hash == "#" {
		return "/"
	}
	return hash
}

// Navigate navigates to a path using the appropriate method for this router type.
func (r *Router) Navigate(path string) {
	if r.routerType == "history" {
		// For history router, use pushState
		js.Global().Get("history").Call("pushState", nil, "", path)
		r.renderCurrentRoute()
	} else {
		// Hash routers re-render through the hashchange listener.
		js.Global().Get("location").Set("hash", path)
	}
}

// NavigateReplace replaces the current history entry using the appropriate method for this router type.
func (r *Router) NavigateReplace(path string) {
	if r.routerType == "history" {
		// For history router, use replaceState
		js.Global().Get("history").Call("replaceState", nil, "", path)
		r.renderCurrentRoute()
	} else {
		// Hash routers re-render through the hashchange listener.
		loc := js.Global().Get("location")
		loc.Call("replace", "#"+strings.TrimPrefix(path, "#"))
	}
}

// Navigate updates the URL using the appropriate method for the current router.
func Navigate(path string) {
	GetRouter().Navigate(path)
}

// NavigateReplace replaces the current history entry using the appropriate method for the current router.
func NavigateReplace(path string) {
	GetRouter().NavigateReplace(path)
}

// GetCurrentPath returns the current route path from the global router.
func GetCurrentPath() string {
	return GetRouter().GetCurrentRouterPath()
}

// GetRoute returns the component for the current route as an Element.
func GetRoute() *render.Element {
	return GetRouter().Current()
}

// GetRouter returns the global router instance.
func GetRouter() *Router {
	return globalRouter
}

// RouteWithElement renders a route directly to a DOM element and sets up hash listening.
func RouteWithElement(path string, elemRef js.Value) {
	Navigate(path)
	globalRouter.MountElement(elemRef)
}

var globalRouter = NewHashRouter()
var cleanupOnce sync.Once

func getCurrentPath() string {
	loc := js.Global().Get("location")
	if !loc.Truthy() {
		return "/"
	}
	hash := strings.TrimPrefix(loc.Get("hash").String(), "#")
	if hash == "" || hash == "/" || hash == "#" {
		return "/"
	}
	return hash
}

func registerCleanup(handler js.Func) {
	cleanupOnce.Do(func() {
		// Register unload listener to release the hashchange handler to avoid leaks in hot reload.
		unload := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			handler.Release()
			return nil
		})
		js.Global().Get("window").Call("addEventListener", "beforeunload", unload)
	})
}

func makeRouteFactory(component interface{}) routeFactory {
	if component == nil {
		panic("router: component cannot be nil")
	}

	if element, ok := component.(*render.Element); ok {
		return func() *render.Element {
			return element
		}
	}

	value := reflect.ValueOf(component)
	if !value.IsValid() || value.Kind() != reflect.Func {
		panic("router: unsupported component type")
	}

	typ := value.Type()
	if typ.NumOut() != 1 {
		panic("router: route component must return one element")
	}

	return func() *render.Element {
		args := []reflect.Value{}
		if typ.NumIn() == 1 {
			arg := reflect.Zero(typ.In(0))
			args = append(args, arg)
		}

		results := value.Call(args)
		if len(results) == 0 || !results[0].IsValid() || results[0].IsNil() {
			return nil
		}

		element, _ := results[0].Interface().(*render.Element)
		return element
	}
}
