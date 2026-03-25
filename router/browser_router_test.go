//go:build js && wasm
// +build js,wasm

package router

import (
	"context"
	"net/url"
	"syscall/js"
	"testing"
	"time"

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
			router := NewHistoryRouter(tt.options)

			if router == nil {
				t.Fatal("NewHistoryRouter returned nil")
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

	historyRouter := NewHistoryRouter(RouterOptions{})
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

	router := NewHistoryRouter(RouterOptions{})

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
	router := NewHistoryRouter(RouterOptions{})

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

	historyRouter := NewHistoryRouter(RouterOptions{})
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

	router := NewHistoryRouter(RouterOptions{})

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

	router := NewHistoryRouter(RouterOptions{})

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

	router := NewHistoryRouter(RouterOptions{})

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

func TestBrowserRouterHydrateMountElement(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewHistoryRouter(RouterOptions{})
	container := js.Global().Get("document").Call("createElement", "div")
	container.Set("id", "router-hydrate-container")

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("HydrateMountElement caused panic: %v", r)
		}
	}()

	router.HydrateMountElement(container)

	if !router.listening {
		t.Fatal("router should be marked as listening after HydrateMountElement")
	}
	if !router.targetElement.Equal(container) {
		t.Fatal("expected HydrateMountElement to retain the provided target element")
	}
}

func TestBrowserRouterLayoutRoutesRenderNestedOutlet(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/reports/7"})
	js.Global().Get("location").Set("pathname", "/dashboard/reports/7")

	router.GoRegisterRoute("/dashboard", func(attrs Attrs) *Element {
		return runtime.Div(nil,
			runtime.Text("layout|"),
			GetOutlet(),
		)
	}, Options{Layout: true})
	router.GoRegisterRoute("/dashboard/reports/:id", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report:"+UseParams().Get("id")))
	})

	elem := router.Current()
	if elem == nil {
		t.Fatal("expected nested history layout route element")
	}
	if got := collectElementText(elem); got != "layout|report:7" {
		t.Fatalf("expected history nested layout output layout|report:7, got %q", got)
	}
}

func TestBrowserRouterLayoutBeforeLeaveBlocksNestedNavigation(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/settings/profile"})
	js.Global().Get("location").Set("pathname", "/dashboard/settings/profile")

	router.GoRegisterRoute("/dashboard", func(attrs Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	router.GoRegisterRoute("/dashboard/settings", func(attrs Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{
		Layout: true,
		BeforeLeave: func(current RouteContext, next RouteContext) GuardResult {
			if next.Path == "/docs/getting-started" {
				return BlockNavigation("Finish settings first")
			}
			return AllowNavigation()
		},
	})
	router.GoRegisterRoute("/dashboard/settings/profile", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("profile"))
	})
	router.GoRegisterRoute("/docs", func(attrs Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	router.GoRegisterRoute("/docs/getting-started", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("docs"))
	})

	router.Navigate("/docs/getting-started")
	if got := js.Global().Get("location").Get("pathname").String(); got != "/dashboard/settings/profile" {
		t.Fatalf("expected nested layout before-leave guard to keep pathname /dashboard/settings/profile, got %q", got)
	}
}

func TestBrowserRouterLayoutRouteLoaderDataScopesPerLevel(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/reports/7"})
	js.Global().Get("location").Set("pathname", "/dashboard/reports/7")

	layoutData := ""
	childData := ""
	router.GoRegisterRoute("/dashboard", func(attrs Attrs) *Element {
		if data := UseRouteData(); data != nil {
			layoutData, _ = data["section"].(string)
		}
		return runtime.Div(nil,
			runtime.Text("layout:"+layoutData+"|"),
			GetOutlet(),
		)
	}, Options{
		Layout: true,
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			return Attrs{"section": "dashboard"}, nil
		},
	})
	router.GoRegisterRoute("/dashboard/reports/:id", func(attrs Attrs) *Element {
		if data := UseRouteData(); data != nil {
			childData, _ = data["report"].(string)
		}
		return runtime.Div(nil, runtime.Text("report:"+childData))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			return Attrs{"report": routeCtx.Params.Get("id")}, nil
		},
	})

	waitForCondition(t, func() bool {
		elem := router.Current()
		if elem == nil {
			return false
		}
		return layoutData == "dashboard" && childData == "7" && collectElementText(elem) == "layout:dashboard|report:7"
	})
}

func TestBrowserRouterLeafMetadataOverridesLayoutMetadata(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/reports/7"})
	js.Global().Get("location").Set("pathname", "/dashboard/reports/7")

	router.GoRegisterRoute("/dashboard", func(attrs Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true, Title: "Dashboard", Description: "Parent dashboard description"})
	router.GoRegisterRoute("/dashboard/reports/:id", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	}, Options{Title: "Report 7", Description: "Leaf report description"})

	if elem := router.Current(); elem == nil {
		t.Fatal("expected nested history route element for metadata test")
	}
	doc := js.Global().Get("document")
	if got := doc.Get("title").String(); got != "Report 7" {
		t.Fatalf("expected leaf history route title Report 7, got %q", got)
	}
	if got := doc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); got != "Leaf report description" {
		t.Fatalf("expected leaf history route description to win, got %q", got)
	}
}

// TestBrowserRouterNotFound tests wildcard route handling
func TestBrowserRouterNotFound(t *testing.T) {
	installRouterBrowserEnv(t)

	router := NewHistoryRouter(RouterOptions{})

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

	router := NewHistoryRouter(RouterOptions{})

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

func TestBrowserRouterSearchParamsNavigatePreservesPath(t *testing.T) {
	installRouterBrowserEnv(t)
	globalRouter = NewHistoryRouter(RouterOptions{DefaultRoute: "/users"})
	js.Global().Get("location").Set("pathname", "/users")
	js.Global().Get("location").Set("search", "?page=1")

	search := UseSearchParams()
	search.Navigate(url.Values{"page": {"2"}, "filter": {"active"}})

	if got := js.Global().Get("location").Get("pathname").String(); got != "/users" {
		t.Fatalf("expected path to remain /users, got %q", got)
	}
	if got := js.Global().Get("location").Get("search").String(); got != "?filter=active&page=2" {
		t.Fatalf("expected query navigation to update search, got %q", got)
	}
}

func TestBrowserRouterAppliesRouteTitleAndRedirect(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/legacy"})
	js.Global().Get("location").Set("pathname", "/legacy")

	router.GoRegisterRoute("/dashboard", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("dashboard"))
	}, Options{Title: "Dashboard"})
	router.GoRegisterRoute("/legacy", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("legacy"))
	}, Options{Redirect: "/dashboard"})

	if route := router.GoGetRoute(); route == nil {
		t.Fatal("expected redirected history route element")
	}
	if got := js.Global().Get("location").Get("pathname").String(); got != "/dashboard" {
		t.Fatalf("expected history redirect to replace pathname, got %q", got)
	}
	if got := js.Global().Get("document").Get("title").String(); got != "Dashboard" {
		t.Fatalf("expected redirected history route to apply title Dashboard, got %q", got)
	}
}

func TestBrowserRouterBeforeLeaveBlocksNavigation(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/edit"})
	js.Global().Get("location").Set("pathname", "/edit")

	router.GoRegisterRoute("/edit", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("edit"))
	}, Options{
		BeforeLeave: func(current RouteContext, next RouteContext) GuardResult {
			if next.Path == "/dashboard" {
				return BlockNavigation("Unsaved changes")
			}
			return AllowNavigation()
		},
	})
	router.GoRegisterRoute("/dashboard", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("dashboard"))
	})

	router.Navigate("/dashboard")
	if got := js.Global().Get("location").Get("pathname").String(); got != "/edit" {
		t.Fatalf("expected history before-leave guard to keep pathname /edit, got %q", got)
	}
}

func TestBrowserRouterBeforeEnterRedirectsNavigation(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/secure"})
	js.Global().Get("location").Set("pathname", "/secure")

	router.GoRegisterRoute("/login", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	}, Options{Title: "Login"})
	router.GoRegisterRoute("/secure", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnter: func(ctx RouteContext) GuardResult {
			return RedirectNavigation("/login")
		},
	})

	if route := router.GoGetRoute(); route == nil {
		t.Fatal("expected guarded history route to redirect")
	}
	if got := js.Global().Get("location").Get("pathname").String(); got != "/login" {
		t.Fatalf("expected before-enter redirect to update history pathname, got %q", got)
	}
}

func TestBrowserRouterAsyncGuardDoubleNavigationDropsStaleAttempt(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/start"})
	js.Global().Get("location").Set("pathname", "/start")

	started := make(chan string, 2)
	releaseFirst := make(chan struct{})
	router.GoRegisterRoute("/start", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("start"))
	}, Options{
		BeforeLeaveAsync: func(ctx context.Context, current RouteContext, next RouteContext) GuardDecision {
			started <- next.Path
			if next.Path == "/slow" {
				<-releaseFirst
			}
			return GuardDecision{}
		},
	})
	router.GoRegisterRoute("/slow", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("slow"))
	})
	router.GoRegisterRoute("/fast", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("fast"))
	})

	doneSlow := make(chan struct{})
	go func() {
		defer close(doneSlow)
		router.Navigate("/slow")
	}()

	select {
	case got := <-started:
		if got != "/slow" {
			t.Fatalf("expected slow navigation to start first, got %q", got)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("slow navigation did not start")
	}

	doneFast := make(chan struct{})
	go func() {
		defer close(doneFast)
		router.Navigate("/fast")
	}()

	waitForCondition(t, func() bool {
		return len(started) >= 1 && js.Global().Get("location").Get("pathname").String() == "/fast"
	})
	close(releaseFirst)
	<-doneSlow
	<-doneFast

	if got := js.Global().Get("location").Get("pathname").String(); got != "/fast" {
		t.Fatalf("expected stale slow navigation to be ignored, got pathname %q", got)
	}
}

func TestBrowserRouterAsyncGuardBackAndForwardUsesFreshAttempt(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/beta"})
	js.Global().Get("location").Set("pathname", "/beta")
	js.Global().Get("history").Call("replaceState", nil, "", "/beta")

	releaseBeta := make(chan struct{})
	betaStarted := make(chan struct{}, 1)

	router.GoRegisterRoute("/beta", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("beta"))
	}, Options{
		BeforeEnterAsync: func(ctx context.Context, next RouteContext) GuardDecision {
			if next.Path == "/beta" {
				betaStarted <- struct{}{}
				<-releaseBeta
			}
			return GuardDecision{}
		},
	})
	router.GoRegisterRoute("/alpha", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("alpha"))
	})

	router.Navigate("/alpha")
	if got := js.Global().Get("location").Get("pathname").String(); got != "/alpha" {
		t.Fatalf("expected alpha navigation to complete, got %q", got)
	}

	backDone := make(chan struct{})
	go func() {
		defer close(backDone)
		js.Global().Get("history").Call("back")
	}()

	select {
	case <-betaStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("back navigation did not start beta guard")
	}

	forwardDone := make(chan struct{})
	go func() {
		defer close(forwardDone)
		js.Global().Get("history").Call("forward")
	}()

	waitForCondition(t, func() bool {
		return js.Global().Get("location").Get("pathname").String() == "/alpha"
	})
	close(releaseBeta)
	<-backDone
	<-forwardDone

	if got := js.Global().Get("location").Get("pathname").String(); got != "/alpha" {
		t.Fatalf("expected stale back navigation to be ignored, got pathname %q", got)
	}
}

func TestBrowserRouterAsyncGuardDelaysLoaderUntilAllowed(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/start"})
	js.Global().Get("location").Set("pathname", "/start")

	releaseGuard := make(chan struct{})
	guardStarted := make(chan struct{}, 1)
	loaderStarted := 0
	router.GoRegisterRoute("/start", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("start"))
	})
	router.GoRegisterRoute("/guarded", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("guarded"))
	}, Options{
		BeforeEnterAsync: func(ctx context.Context, next RouteContext) GuardDecision {
			guardStarted <- struct{}{}
			<-releaseGuard
			return GuardDecision{}
		},
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			loaderStarted++
			return Attrs{"ready": true}, nil
		},
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		router.Navigate("/guarded")
	}()

	select {
	case <-guardStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("guarded navigation did not start")
	}
	if loaderStarted != 0 {
		t.Fatalf("expected loader to wait for guard release, got %d starts", loaderStarted)
	}
	close(releaseGuard)
	<-done

	waitForCondition(t, func() bool {
		return js.Global().Get("location").Get("pathname").String() == "/guarded" && loaderStarted == 1
	})
}

func TestBrowserRouterAsyncGuardUnmountCleanup(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/guarded"})
	js.Global().Get("location").Set("pathname", "/guarded")

	releaseGuard := make(chan struct{})
	guardStarted := make(chan struct{}, 1)
	otherRendered := make(chan struct{}, 1)
	router.GoRegisterRoute("/guarded", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("guarded"))
	}, Options{
		BeforeEnterAsync: func(ctx context.Context, next RouteContext) GuardDecision {
			guardStarted <- struct{}{}
			<-releaseGuard
			return GuardDecision{}
		},
	})
	router.GoRegisterRoute("/other", func(attrs Attrs) *Element {
		otherRendered <- struct{}{}
		return runtime.Div(nil, runtime.Text("other"))
	})

	container := js.Global().Get("document").Call("createElement", "div")
	renderDone := make(chan struct{})
	go func() {
		defer close(renderDone)
		router.MountElement(container)
	}()

	select {
	case <-guardStarted:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("guarded mount did not start")
	}

	router.Navigate("/other")
	select {
	case <-otherRendered:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("replacement route did not render while guarded mount was pending")
	}
	close(releaseGuard)
	<-renderDone

	waitForCondition(t, func() bool {
		return js.Global().Get("location").Get("pathname").String() == "/other"
	})
}

func TestBrowserRouterReplacesMetadataAcrossRoutes(t *testing.T) {
	installRouterBrowserEnv(t)
	router := NewHistoryRouter(RouterOptions{DefaultRoute: "/first"})
	js.Global().Get("location").Set("pathname", "/first")

	router.GoRegisterRoute("/first", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("first"))
	}, Options{
		Title:        "First",
		Description:  "First description",
		CanonicalURL: "https://example.com/first",
	})
	router.GoRegisterRoute("/second", func(attrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("second"))
	}, Options{
		Title:        "Second",
		Description:  "Second description",
		CanonicalURL: "https://example.com/second",
	})

	if route := router.GoGetRoute(); route == nil {
		t.Fatal("expected first route element")
	}
	router.Navigate("/second")
	doc := js.Global().Get("document")
	if got := doc.Get("title").String(); got != "Second" {
		t.Fatalf("expected route title Second after navigation, got %q", got)
	}
	if got := doc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); got != "Second description" {
		t.Fatalf("expected replaced description metadata, got %q", got)
	}
	if got := doc.Call("querySelector", `link[rel="canonical"]`).Get("attributes").Get("href").String(); got != "https://example.com/second" {
		t.Fatalf("expected replaced canonical metadata, got %q", got)
	}
}
