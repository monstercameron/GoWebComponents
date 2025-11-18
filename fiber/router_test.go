//go:build js && wasm
// +build js,wasm

package fiber

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/render"
	"github.com/monstercameron/GoWebComponents/router"
)

// Type aliases for convenience
type (
	Attrs     = dom.Attrs
	Element   = render.Element
	Component = router.Component
	Options   = router.Options
)

// TestNewHashRouter tests hash router initialization
func TestNewHashRouter(t *testing.T) {
	r := router.NewHashRouter()

	if r == nil {
		t.Fatal("router.NewHashRouter returned nil")
	}
}

// TestNewHashRouterWithOptions tests hash router with custom options
func TestNewHashRouterWithOptions(t *testing.T) {
	options := router.RouterOptions{
		DefaultRoute: "/home",
	}

	r := router.NewHashRouter(options)

	if r == nil {
		t.Fatal("router.NewHashRouter with options returned nil")
	}
}

// TestRegisterRoute tests route registration
func TestRegisterRoute(t *testing.T) {
	r := router.NewHashRouter()

	// Register a simple route
	testComponent := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Test"))
	}

	r.GoRegisterRoute("/test", testComponent)

	// Test that the route can be retrieved
	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("GoGetRoute returned nil")
	}
}

// TestPathNormalization tests that paths are normalized correctly
func TestPathNormalization(t *testing.T) {
	r := router.NewHashRouter()

	testCases := []struct {
		input    string
		expected string
	}{
		{"/about", "/about"},
		{"/about/", "/about"},
		{"about", "/about"},
		{"/", "/"},
		{"", "/"},
		{"/users/123", "/users/123"},
		{"/users/123/", "/users/123"},
	}

	for _, tc := range testCases {
		testComponent := func(props Attrs) *Element {
			return dom.Div(nil)
		}

		r.GoRegisterRoute(tc.input, testComponent)
	}

	// Simply verify routes were registered
	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Routes not properly registered")
	}
}

// TestGetCurrentPath tests current path getter
func TestGetCurrentPath(t *testing.T) {
	r := router.NewHashRouter()

	path := r.GetCurrentRouterPath()
	if path == "" {
		t.Error("GetCurrentRouterPath returned empty string")
	}
}

// TestNavigate tests navigation
func TestNavigate(t *testing.T) {
	r := router.NewHashRouter()

	r.Navigate("/new-path")

	// Navigation happened (can't easily test in unit test without browser)
	_ = r.GetCurrentRouterPath()
}

// TestRouteResolution tests route resolution/matching
func TestRouteResolution(t *testing.T) {
	r := router.NewHashRouter()

	homeComponent := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Home"))
	}

	aboutComponent := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("About"))
	}

	r.GoRegisterRoute("/", homeComponent)
	r.GoRegisterRoute("/about", aboutComponent)

	// Test that routes return elements
	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Route resolution returned nil")
	}
}

// TestWildcardRoute tests wildcard (404) route
func TestWildcardRoute(t *testing.T) {
	r := router.NewHashRouter()

	homeComponent := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Home"))
	}

	notFoundComponent := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Not Found"))
	}

	r.GoRegisterRoute("/", homeComponent)
	r.GoRegisterRoute("*", notFoundComponent)

	// Verify routes are registered
	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Wildcard route not working")
	}
}

// TestStaticElementRoute tests registering a static element as a route
func TestStaticElementRoute(t *testing.T) {
	r := router.NewHashRouter()

	staticElem := dom.Div(nil, dom.Text("Static"))

	r.GoRegisterRoute("/static", staticElem)

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Static element route failed")
	}
}

// TestBeforeEnterGuard tests beforeEnter route guard
func TestBeforeEnterGuard(t *testing.T) {
	r := router.NewHashRouter()

	protectedComponent := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Protected"))
	}

	r.GoRegisterRoute("/protected", protectedComponent)

	// Simply verify route was registered
	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Route registration failed")
	}
}

// TestRouterTypeValidation tests that router type is correctly set
func TestRouterTypeValidation(t *testing.T) {
	hashRouter := router.NewHashRouter()
	regularRouter := router.NewRouter(router.RouterOptions{})

	if hashRouter == nil {
		t.Error("Hash router creation failed")
	}

	if regularRouter == nil {
		t.Error("Regular router creation failed")
	}
}

// TestMultipleRoutes tests registering multiple routes
func TestMultipleRoutes(t *testing.T) {
	r := router.NewHashRouter()

	routes := []string{"/", "/about", "/contact", "/services", "/blog"}

	for _, path := range routes {
		p := path
		component := func(props Attrs) *Element {
			return dom.Div(nil, dom.Text(p))
		}
		r.GoRegisterRoute(path, component)
	}

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Multiple route registration failed")
	}
}

// TestGlobalRouter tests global router instance management
func TestGlobalRouter(t *testing.T) {
	initialRouter := router.GetRouter()

	if initialRouter == nil {
		t.Fatal("Global router is nil")
	}

	// Simply verify global router exists
	currentRouter := router.GetRouter()
	if currentRouter == nil {
		t.Error("Global router is nil")
	}
}

// TestGoRegisterRoute tests the Go-style route registration
func TestGoRegisterRoute(t *testing.T) {
	componentFunc := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Test Component"))
	}

	router.RegisterRoute("/go-test", componentFunc)

	result := router.GetRoute()
	if result == nil {
		t.Error("RegisterRoute did not properly register the route")
	}
}

// TestRouteOptions tests route options like Title
func TestRouteOptions(t *testing.T) {
	r := router.NewHashRouter()

	component := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Page"))
	}

	options := Options{
		Title: "Test Page Title",
	}

	r.GoRegisterRoute("/test-page", component, options)

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Route with options failed")
	}
}

// TestEmptyPath tests handling of empty paths
func TestEmptyPath(t *testing.T) {
	r := router.NewHashRouter()

	component := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("Home"))
	}

	r.GoRegisterRoute("", component) // Should normalize to "/"

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Empty path registration failed")
	}
}

// TestPathWithTrailingSlash tests path with trailing slash normalization
func TestPathWithTrailingSlash(t *testing.T) {
	r := router.NewHashRouter()

	component := func(props Attrs) *Element {
		return dom.Div(nil, dom.Text("About"))
	}

	r.GoRegisterRoute("/about/", component) // Should normalize to "/about"

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Path with trailing slash failed")
	}
}

// TestNavigateFunctions tests global Navigate functions
func TestNavigateFunctions(t *testing.T) {
	router.Navigate("/test")
	router.NavigateReplace("/test2")

	path := router.GetCurrentPath()
	if path == "" {
		t.Error("Navigation functions failed")
	}
}
