//go:build js && wasm
// +build js,wasm

package fiber

import (
	"testing"
)

// TestNewHashRouter tests hash router initialization
func TestNewHashRouter(t *testing.T) {
	router := NewHashRouter()

	if router == nil {
		t.Fatal("NewHashRouter returned nil")
	}

	if router.GetRouterType() != "hash" {
		t.Errorf("Expected router type 'hash', got '%s'", router.GetRouterType())
	}

	if router.GetCurrentPath() != "/" {
		t.Errorf("Expected initial path '/', got '%s'", router.GetCurrentPath())
	}
}

// TestNewHashRouterWithOptions tests hash router with custom options
func TestNewHashRouterWithOptions(t *testing.T) {
	options := RouterOptions{
		DefaultRoute: "/home",
		Type:         "hash",
	}

	router := NewHashRouter(options)

	if router.GetCurrentPath() != "/home" {
		t.Errorf("Expected initial path '/home', got '%s'", router.GetCurrentPath())
	}
}

// TestRegisterRoute tests route registration
func TestRegisterRoute(t *testing.T) {
	router := NewHashRouter()

	// Register a simple route
	componentCalled := false
	testComponent := func(props map[string]interface{}) interface{} {
		componentCalled = true
		return nil
	}

	router.Register("/test", testComponent)

	if len(router.routes) != 1 {
		t.Errorf("Expected 1 route registered, got %d", len(router.routes))
	}

	if router.routes[0].Path != "/test" {
		t.Errorf("Expected route path '/test', got '%s'", router.routes[0].Path)
	}
}

// TestPathNormalization tests that paths are normalized correctly
func TestPathNormalization(t *testing.T) {
	router := NewHashRouter()

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
		testComponent := func(props map[string]interface{}) interface{} {
			return Div(nil)
		}

		router.Register(tc.input, testComponent)
	}

	for i, tc := range testCases {
		if router.routes[i].Path != tc.expected {
			t.Errorf("Path normalization failed: input '%s' expected '%s', got '%s'",
				tc.input, tc.expected, router.routes[i].Path)
		}
	}
}

// TestGetCurrentPath tests current path getter
func TestGetCurrentPath(t *testing.T) {
	router := NewHashRouter()

	if router.GetCurrentPath() != "/" {
		t.Errorf("Expected initial path '/', got '%s'", router.GetCurrentPath())
	}

	router.SetCurrentPath("/about")

	if router.GetCurrentPath() != "/about" {
		t.Errorf("Expected path '/about', got '%s'", router.GetCurrentPath())
	}
}

// TestSetCurrentPath tests path setter
func TestSetCurrentPath(t *testing.T) {
	router := NewHashRouter()

	router.SetCurrentPath("/new-path")

	if router.GetCurrentPath() != "/new-path" {
		t.Errorf("Expected path '/new-path', got '%s'", router.GetCurrentPath())
	}
}

// TestRouteResolution tests route resolution/matching
func TestRouteResolution(t *testing.T) {
	router := NewHashRouter()

	homeComponent := func(props map[string]interface{}) interface{} {
		return Div(nil, "Home")
	}

	aboutComponent := func(props map[string]interface{}) interface{} {
		return Div(nil, "About")
	}

	router.Register("/", homeComponent)
	router.Register("/about", aboutComponent)

	// Test home route
	router.SetCurrentPath("/")
	result := router.Route("/")

	if result == nil {
		t.Error("Route resolution returned nil for '/'")
	}

	// Test about route
	result = router.Route("/about")
	if result == nil {
		t.Error("Route resolution returned nil for '/about'")
	}
}

// TestWildcardRoute tests wildcard (404) route
func TestWildcardRoute(t *testing.T) {
	router := NewHashRouter()

	homeComponent := func(props map[string]interface{}) interface{} {
		return Div(nil, "Home")
	}

	notFoundComponent := func(props map[string]interface{}) interface{} {
		return Div(nil, "Not Found")
	}

	router.Register("/", homeComponent)
	router.Register("*", notFoundComponent)

	// Unregistered route should match wildcard
	result := router.Route("/unknown-route")

	if result == nil {
		t.Error("Wildcard route did not match unknown path")
	}
}

// TestOnNavigateCallback tests navigation callback
func TestOnNavigateCallback(t *testing.T) {
	router := NewHashRouter()

	callbackCalled := false
	navigatedPath := ""

	router.OnNavigate(func(path string) {
		callbackCalled = true
		navigatedPath = path
	})

	router.SetCurrentPath("/test")

	if !callbackCalled {
		t.Error("Navigation callback was not called")
	}

	if navigatedPath != "/test" {
		t.Errorf("Expected callback path '/test', got '%s'", navigatedPath)
	}
}

// TestBeforeEnterGuard tests beforeEnter route guard
func TestBeforeEnterGuard(t *testing.T) {
	router := NewHashRouter()

	guardCalled := false
	protectedComponent := func(props map[string]interface{}) interface{} {
		return Div(nil, "Protected")
	}

	options := RouteOptions{
		BeforeEnter: func(path string) bool {
			guardCalled = true
			return false // Prevent navigation
		},
	}

	router.Register("/protected", protectedComponent, options)

	// Navigation should be prevented if guard returns false
	initialPath := router.GetCurrentPath()

	// Direct navigation attempt
	pathResult := router.Route("/protected")

	if pathResult == nil && !guardCalled {
		t.Error("BeforeEnter guard was not called")
	}
}

// TestRouterTypeValidation tests that router type is correctly set
func TestRouterTypeValidation(t *testing.T) {
	hashRouter := NewHashRouter()
	regularRouter := NewRegularRouter()

	if hashRouter.GetRouterType() != "hash" {
		t.Errorf("Hash router type mismatch: expected 'hash', got '%s'", hashRouter.GetRouterType())
	}

	if regularRouter.GetRouterType() != "regular" {
		t.Errorf("Regular router type mismatch: expected 'regular', got '%s'", regularRouter.GetRouterType())
	}
}

// TestMultipleRoutes tests registering multiple routes
func TestMultipleRoutes(t *testing.T) {
	router := NewHashRouter()

	routes := []string{"/", "/about", "/contact", "/services", "/blog"}

	for _, path := range routes {
		component := func(props map[string]interface{}) interface{} {
			return Div(nil, path)
		}
		router.Register(path, component)
	}

	if len(router.routes) != len(routes) {
		t.Errorf("Expected %d routes, got %d", len(routes), len(router.routes))
	}

	for i, expectedPath := range routes {
		if router.routes[i].Path != expectedPath {
			t.Errorf("Route %d: expected path '%s', got '%s'", i, expectedPath, router.routes[i].Path)
		}
	}
}

// TestGlobalRouter tests global router instance management
func TestGlobalRouter(t *testing.T) {
	initialRouter := GetRouter()

	if initialRouter == nil {
		t.Fatal("Global router is nil")
	}

	newRouter := NewHashRouter()
	SetGlobalRouter(newRouter)

	currentRouter := GetRouter()
	if currentRouter != newRouter {
		t.Error("SetGlobalRouter did not update global router")
	}

	// Restore original router
	SetGlobalRouter(initialRouter)
}

// TestGoRegisterRoute tests the Go-style route registration
func TestGoRegisterRoute(t *testing.T) {
	SetGlobalRouter(NewHashRouter())

	componentFunc := func(props Attrs) *Element {
		return Div(nil, "Test Component")
	}

	GoRegisterRoute("/go-test", componentFunc)

	result := Route("/go-test")
	if result == nil {
		t.Error("GoRegisterRoute did not properly register the route")
	}
}

// TestRouteOptions tests route options like Title
func TestRouteOptions(t *testing.T) {
	router := NewHashRouter()

	component := func(props map[string]interface{}) interface{} {
		return Div(nil, "Page")
	}

	options := RouteOptions{
		Title: "Test Page Title",
	}

	router.Register("/test-page", component, options)

	if router.routes[0].Options.Title != "Test Page Title" {
		t.Errorf("Route options not set correctly. Expected 'Test Page Title', got '%s'",
			router.routes[0].Options.Title)
	}
}

// TestNoHistoryOption tests NoHistory route option
func TestNoHistoryOption(t *testing.T) {
	router := NewHashRouter()

	component := func(props map[string]interface{}) interface{} {
		return Div(nil, "No History Page")
	}

	options := RouteOptions{
		NoHistory: true,
	}

	router.Register("/no-history", component, options)

	if !router.routes[0].Options.NoHistory {
		t.Error("NoHistory option was not set correctly")
	}
}

// TestEmptyPath tests handling of empty paths
func TestEmptyPath(t *testing.T) {
	router := NewHashRouter()

	component := func(props map[string]interface{}) interface{} {
		return Div(nil, "Home")
	}

	router.Register("", component) // Should normalize to "/"

	if router.routes[0].Path != "/" {
		t.Errorf("Empty path should normalize to '/', got '%s'", router.routes[0].Path)
	}
}

// TestPathWithTrailingSlash tests path with trailing slash normalization
func TestPathWithTrailingSlash(t *testing.T) {
	router := NewHashRouter()

	component := func(props map[string]interface{}) interface{} {
		return Div(nil, "About")
	}

	router.Register("/about/", component) // Should normalize to "/about"

	if router.routes[0].Path != "/about" {
		t.Errorf("Path with trailing slash should normalize to '/about', got '%s'",
			router.routes[0].Path)
	}
}
