//go:build js && wasm
// +build js,wasm

package router

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// TestNewBrowserRouter tests browser/history router initialization
func TestNewBrowserRouter(t *testing.T) {
	installRouterBrowserEnv(t)

	tests := []struct {
		name        string
		options     RouterOptions
		expected    string
		expectError bool
	}{
		{
			name:        "creates router with default options",
			options:     RouterOptions{},
			expected:    "/",
			expectError: false,
		},
		{
			name:        "creates router with custom default route",
			options:     RouterOptions{DefaultRoute: "/home"},
			expected:    "/home",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := NewRouter(tt.options)

			if router == nil {
				t.Fatal("NewRouter returned nil")
			}

			if router.routerType != "history" {
				t.Fatalf("expected router type 'history', got '%s'", router.routerType)
			}

			if router.defaultRoute != tt.expected {
				t.Fatalf("expected default route '%s', got '%s'", tt.expected, router.defaultRoute)
			}

			if router.routes == nil {
				t.Fatal("routes map not initialized")
			}
		})
	}
}

// TestBrowserRouterType verifies router type is set correctly
func TestBrowserRouterType(t *testing.T) {
	installRouterBrowserEnv(t)

	historyRouter := NewRouter(RouterOptions{})
	hashRouter := NewHashRouter()

	if historyRouter.routerType != "history" {
		t.Fatalf("expected history router type, got '%s'", historyRouter.routerType)
	}

	if hashRouter.routerType != "hash" {
		t.Fatalf("expected hash router type, got '%s'", hashRouter.routerType)
	}
}

// TestBrowserRouterRegisterRoute tests route registration
func TestBrowserRouterRegisterRoute(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewRouter(RouterOptions{})

	testComponent := func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("test"))
	}

	router.GoRegisterRoute("/", testComponent)
	router.GoRegisterRoute("/about", testComponent)

	if len(router.routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(router.routes))
	}

	if _, ok := router.routes["/"]; !ok {
		t.Fatal("root route not registered")
	}

	if _, ok := router.routes["/about"]; !ok {
		t.Fatal("/about route not registered")
	}
}

// TestBrowserRouterGetCurrentPath tests path detection based on location
func TestBrowserRouterGetCurrentPath(t *testing.T) {
	installRouterBrowserEnv(t)

	// Note: This test relies on window.location being set by the browser
	// In WASM, this will be the actual browser's current URL
	router := NewRouter(RouterOptions{})

	path := router.GetCurrentRouterPath()

	// Path should at minimum return "/" as default
	if path == "" {
		t.Fatal("GetCurrentRouterPath returned empty string")
	}

	// Path should start with /
	if len(path) > 0 && path[0] != '/' {
		t.Fatalf("path should start with '/', got '%s'", path)
	}
}

// TestBrowserRouterHashVsHistoryPath tests that different router types read different paths
func TestBrowserRouterHashVsHistoryPath(t *testing.T) {
	installRouterBrowserEnv(t)

	historyRouter := NewRouter(RouterOptions{})
	hashRouter := NewHashRouter()

	// Both should implement GetCurrentRouterPath
	historyPath := historyRouter.GetCurrentRouterPath()
	hashPath := hashRouter.GetCurrentRouterPath()

	// They might be different or the same depending on current URL
	// Just verify both methods work and return valid paths
	if len(historyPath) > 0 && historyPath[0] != '/' && historyPath != "" {
		t.Fatalf("history router path invalid: '%s'", historyPath)
	}

	if hashPath == "" {
		t.Fatal("hash router path should not be empty")
	}
}

// TestBrowserRouterNavigate tests navigation method
func TestBrowserRouterNavigate(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewRouter(RouterOptions{})

	// Register a test component
	testComponent := func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("navigation test"))
	}

	router.GoRegisterRoute("/nav-test", testComponent)

	// Navigate should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Navigate caused panic: %v", r)
		}
	}()

	router.Navigate("/nav-test")
}

// TestBrowserRouterNavigateReplace tests NavigateReplace method
func TestBrowserRouterNavigateReplace(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewRouter(RouterOptions{})

	// NavigateReplace should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NavigateReplace caused panic: %v", r)
		}
	}()

	router.NavigateReplace("/replace-test")
}

// TestBrowserRouterMountElement tests mounting to a DOM element
func TestBrowserRouterMountElement(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewRouter(RouterOptions{})

	testComponent := func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("mounted content"))
	}

	router.GoRegisterRoute("/", testComponent)

	// Get or create a test element
	container := js.Global().Get("document").Call("createElement", "div")
	container.Set("id", "router-container")

	// Mount should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("MountElement caused panic: %v", r)
		}
	}()

	router.MountElement(container)

	// Verify router is set as listening
	if !router.listening {
		t.Fatal("router should be marked as listening after Mount")
	}
}

// TestBrowserRouterNotFound tests wildcard route handling
func TestBrowserRouterNotFound(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewRouter(RouterOptions{})

	homeComponent := func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("home"))
	}

	notFoundComponent := func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("not found"))
	}

	router.GoRegisterRoute("/", homeComponent)
	router.GoRegisterRoute("*", notFoundComponent)

	if router.notFound == nil {
		t.Fatal("notFound component not registered")
	}
}

// TestBrowserRouterGetRoute tests getting a route component
func TestBrowserRouterGetRoute(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewRouter(RouterOptions{})

	testComponent := func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("test content"))
	}

	router.GoRegisterRoute("/", testComponent)

	// GetRoute should return an element
	route := router.GoGetRoute()

	if route == nil {
		t.Fatal("GoGetRoute returned nil")
	}
}
