//go:build js && wasm

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
func TestNewBrowserRouter(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseTests := []struct {
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

	for _, parseTt := range parseTests {
		parseT.Run(parseTt.name, func(parseT2 *testing.T) {
			parseRouter := NewHistoryRouter(parseTt.options)

			if parseRouter == nil {
				parseT2.Fatal("NewHistoryRouter returned nil")
			}

			if parseRouter.routerType != "history" {
				parseT2.Fatalf("expected router type 'history', got '%s'", parseRouter.routerType)
			}

			if parseRouter.defaultRoute != parseTt.expected {
				parseT2.Fatalf("expected default route '%s', got '%s'", parseTt.expected, parseRouter.defaultRoute)
			}

			if parseRouter.routes == nil {
				parseT2.Fatal("routes map not initialized")
			}
		})
	}
}

// TestBrowserRouterType verifies router type is set correctly
func TestBrowserRouterType(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseHistoryRouter := NewHistoryRouter(RouterOptions{})
	parseHashRouter := NewHashRouter()

	if parseHistoryRouter.routerType != "history" {
		parseT.Fatalf("expected history router type, got '%s'", parseHistoryRouter.routerType)
	}

	if parseHashRouter.routerType != "hash" {
		parseT.Fatalf("expected hash router type, got '%s'", parseHashRouter.routerType)
	}
}

// TestBrowserRouterRegisterRoute tests route registration
func TestBrowserRouterRegisterRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})

	parseTestComponent := func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("test"))
	}

	parseRouter.GoRegisterRoute("/", parseTestComponent)
	parseRouter.GoRegisterRoute("/about", parseTestComponent)

	if len(parseRouter.routes) != 2 {
		parseT.Fatalf("expected 2 routes, got %d", len(parseRouter.routes))
	}

	if _, parseOk := parseRouter.routes["/"]; !parseOk {
		parseT.Fatal("root route not registered")
	}

	if _, parseOk2 := parseRouter.routes["/about"]; !parseOk2 {
		parseT.Fatal("/about route not registered")
	}
}

// TestBrowserRouterGetCurrentPath tests path detection based on location
func TestBrowserRouterGetCurrentPath(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	// Note: This test relies on window.location being set by the browser
	// In WASM, this will be the actual browser's current URL
	parseRouter := NewHistoryRouter(RouterOptions{})

	parsePath := parseRouter.GetCurrentRouterPath()

	// Path should at minimum return "/" as default
	if parsePath == "" {
		parseT.Fatal("GetCurrentRouterPath returned empty string")
	}

	// Path should start with /
	if len(parsePath) > 0 && parsePath[0] != '/' {
		parseT.Fatalf("path should start with '/', got '%s'", parsePath)
	}
}

// TestBrowserRouterHashVsHistoryPath tests that different router types read different paths
func TestBrowserRouterHashVsHistoryPath(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseHistoryRouter := NewHistoryRouter(RouterOptions{})
	parseHashRouter := NewHashRouter()

	// Both should implement GetCurrentRouterPath
	parseHistoryPath := parseHistoryRouter.GetCurrentRouterPath()
	parseHashPath := parseHashRouter.GetCurrentRouterPath()

	// They might be different or the same depending on current URL
	// Just verify both methods work and return valid paths
	if len(parseHistoryPath) > 0 && parseHistoryPath[0] != '/' && parseHistoryPath != "" {
		parseT.Fatalf("history router path invalid: '%s'", parseHistoryPath)
	}

	if parseHashPath == "" {
		parseT.Fatal("hash router path should not be empty")
	}
}

// TestBrowserRouterNavigate tests navigation method
func TestBrowserRouterNavigate(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})

	// Register a test component
	parseTestComponent := func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("navigation test"))
	}

	parseRouter.GoRegisterRoute("/nav-test", parseTestComponent)

	// Navigate should not panic
	defer func() {
		if parseR := recover(); parseR != nil {
			parseT.Fatalf("Navigate caused panic: %v", parseR)
		}
	}()

	parseRouter.Navigate("/nav-test")
}

func TestBrowserRouterNavigateHistoryFragmentPreservesPath(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{})
	js.Global().Get("location").Set("pathname", "/pricing")
	js.Global().Get("location").Set("search", "?plan=team")

	parseRouter.Navigate("#faq")

	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/pricing" {
		parseT.Fatalf("expected history fragment navigation to keep pathname, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("location").Get("search").String(); parseGot2 != "?plan=team" {
		parseT.Fatalf("expected history fragment navigation to keep search, got %q", parseGot2)
	}
	if parseGot3 := js.Global().Get("location").Get("hash").String(); parseGot3 != "#faq" {
		parseT.Fatalf("expected history fragment navigation to set hash, got %q", parseGot3)
	}
}

func TestBrowserRouterNavigateHistoryTargetPreservesFragment(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{})

	parseRouter.Navigate("/pricing?plan=team#faq")

	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/pricing" {
		parseT.Fatalf("expected history navigation to set pathname, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("location").Get("search").String(); parseGot2 != "?plan=team" {
		parseT.Fatalf("expected history navigation to preserve search, got %q", parseGot2)
	}
	if parseGot3 := js.Global().Get("location").Get("hash").String(); parseGot3 != "#faq" {
		parseT.Fatalf("expected history navigation to preserve hash, got %q", parseGot3)
	}
}

// TestBrowserRouterNavigateReplace tests NavigateReplace method
func TestBrowserRouterNavigateReplace(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})

	// NavigateReplace should not panic
	defer func() {
		if parseR := recover(); parseR != nil {
			parseT.Fatalf("NavigateReplace caused panic: %v", parseR)
		}
	}()

	parseRouter.NavigateReplace("/replace-test")
}

func TestBrowserRouterNavigateReplaceHistoryFragmentPreservesPath(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{})
	js.Global().Get("location").Set("pathname", "/pricing")

	parseRouter.NavigateReplace("#plans")

	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/pricing" {
		parseT.Fatalf("expected history replace fragment navigation to keep pathname, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("location").Get("hash").String(); parseGot2 != "#plans" {
		parseT.Fatalf("expected history replace fragment navigation to set hash, got %q", parseGot2)
	}
}

// TestBrowserRouterMountElement tests mounting to a DOM element
func TestBrowserRouterMountElement(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})

	parseTestComponent := func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("mounted content"))
	}

	parseRouter.GoRegisterRoute("/", parseTestComponent)

	// Get or create a test element
	parseContainer := js.Global().Get("document").Call("createElement", "div")
	parseContainer.Set("id", "router-container")

	// Mount should not panic
	defer func() {
		if parseR := recover(); parseR != nil {
			parseT.Fatalf("MountElement caused panic: %v", parseR)
		}
	}()

	parseRouter.MountElement(parseContainer)

	// Verify router is set as listening
	if !parseRouter.listening {
		parseT.Fatal("router should be marked as listening after Mount")
	}
}

func TestBrowserRouterHydrateMountElement(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})
	parseContainer := js.Global().Get("document").Call("createElement", "div")
	parseContainer.Set("id", "router-hydrate-container")

	defer func() {
		if parseR := recover(); parseR != nil {
			parseT.Fatalf("HydrateMountElement caused panic: %v", parseR)
		}
	}()

	parseRouter.HydrateMountElement(parseContainer)

	if !parseRouter.listening {
		parseT.Fatal("router should be marked as listening after HydrateMountElement")
	}
	if !parseRouter.targetElement.Equal(parseContainer) {
		parseT.Fatal("expected HydrateMountElement to retain the provided target element")
	}
}

func TestBrowserRouterLayoutRoutesRenderNestedOutlet(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/reports/7"})
	js.Global().Get("location").Set("pathname", "/dashboard/reports/7")

	parseRouter.GoRegisterRoute("/dashboard", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil,
			runtime.Text("layout|"),
			GetOutlet(),
		)
	}, Options{Layout: true})
	parseRouter.GoRegisterRoute("/dashboard/reports/:id", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report:"+UseParams().Get("id")))
	})

	parseElem := parseRouter.Current()
	if parseElem == nil {
		parseT.Fatal("expected nested history layout route element")
	}
	if parseGot := collectElementText(parseElem); parseGot != "layout|report:7" {
		parseT.Fatalf("expected history nested layout output layout|report:7, got %q", parseGot)
	}
}

func TestBrowserRouterLayoutBeforeLeaveBlocksNestedNavigation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/settings/profile"})
	js.Global().Get("location").Set("pathname", "/dashboard/settings/profile")

	parseRouter.GoRegisterRoute("/dashboard", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	parseRouter.GoRegisterRoute("/dashboard/settings", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{
		Layout: true,
		BeforeLeave: func(parseCurrent RouteContext, parseNext RouteContext) GuardResult {
			if parseNext.Path == "/docs/getting-started" {
				return BlockNavigation("Finish settings first")
			}
			return AllowNavigation()
		},
	})
	parseRouter.GoRegisterRoute("/dashboard/settings/profile", func(parseAttrs3 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("profile"))
	})
	parseRouter.GoRegisterRoute("/docs", func(parseAttrs4 Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	parseRouter.GoRegisterRoute("/docs/getting-started", func(parseAttrs5 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("docs"))
	})

	parseRouter.Navigate("/docs/getting-started")
	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/dashboard/settings/profile" {
		parseT.Fatalf("expected nested layout before-leave guard to keep pathname /dashboard/settings/profile, got %q", parseGot)
	}
}

func TestBrowserRouterLayoutRouteLoaderDataScopesPerLevel(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/reports/7"})
	js.Global().Get("location").Set("pathname", "/dashboard/reports/7")

	parseLayoutData := ""
	parseChildData := ""
	parseRouter.GoRegisterRoute("/dashboard", func(parseAttrs Attrs) *Element {
		if parseData := UseRouteData(); parseData != nil {
			parseLayoutData, _ = parseData["section"].(string)
		}
		return runtime.Div(nil,
			runtime.Text("layout:"+parseLayoutData+"|"),
			GetOutlet(),
		)
	}, Options{
		Layout: true,
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			return Attrs{"section": "dashboard"}, nil
		},
	})
	parseRouter.GoRegisterRoute("/dashboard/reports/:id", func(parseAttrs2 Attrs) *Element {
		if parseData2 := UseRouteData(); parseData2 != nil {
			parseChildData, _ = parseData2["report"].(string)
		}
		return runtime.Div(nil, runtime.Text("report:"+parseChildData))
	}, Options{
		Loader: func(parseCtx2 context.Context, parseRouteCtx2 RouteContext) (Attrs, error) {
			return Attrs{"report": parseRouteCtx2.Params.Get("id")}, nil
		},
	})

	waitForCondition(parseT, func() bool {
		parseElem := parseRouter.Current()
		if parseElem == nil {
			return false
		}
		return parseLayoutData == "dashboard" && parseChildData == "7" && collectElementText(parseElem) == "layout:dashboard|report:7"
	})
}

func TestBrowserRouterLeafMetadataOverridesLayoutMetadata(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/dashboard/reports/7"})
	js.Global().Get("location").Set("pathname", "/dashboard/reports/7")

	parseRouter.GoRegisterRoute("/dashboard", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true, Title: "Dashboard", Description: "Parent dashboard description"})
	parseRouter.GoRegisterRoute("/dashboard/reports/:id", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	}, Options{Title: "Report 7", Description: "Leaf report description"})

	if parseElem := parseRouter.Current(); parseElem == nil {
		parseT.Fatal("expected nested history route element for metadata test")
	}
	parseDoc := js.Global().Get("document")
	if parseGot := parseDoc.Get("title").String(); parseGot != "Report 7" {
		parseT.Fatalf("expected leaf history route title Report 7, got %q", parseGot)
	}
	if parseGot2 := parseDoc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); parseGot2 != "Leaf report description" {
		parseT.Fatalf("expected leaf history route description to win, got %q", parseGot2)
	}
}

// TestBrowserRouterNotFound tests wildcard route handling
func TestBrowserRouterNotFound(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})

	parseHomeComponent := func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("home"))
	}

	parseNotFoundComponent := func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("not found"))
	}

	parseRouter.GoRegisterRoute("/", parseHomeComponent)
	parseRouter.GoRegisterRoute("*", parseNotFoundComponent)

	if parseRouter.notFound == nil {
		parseT.Fatal("notFound component not registered")
	}
}

// TestBrowserRouterGetRoute tests getting a route component
func TestBrowserRouterGetRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{})

	parseTestComponent := func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("test content"))
	}

	parseRouter.GoRegisterRoute("/", parseTestComponent)

	// GetRoute should return an element
	parseRoute := parseRouter.GoGetRoute()

	if parseRoute == nil {
		parseT.Fatal("GoGetRoute returned nil")
	}
}

func TestBrowserRouterSearchParamsNavigatePreservesPath(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHistoryRouter(RouterOptions{DefaultRoute: "/users"})
	js.Global().Get("location").Set("pathname", "/users")
	js.Global().Get("location").Set("search", "?page=1")

	parseSearch := UseSearchParams()
	parseSearch.Navigate(url.Values{"page": {"2"}, "filter": {"active"}})

	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/users" {
		parseT.Fatalf("expected path to remain /users, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("location").Get("search").String(); parseGot2 != "?filter=active&page=2" {
		parseT.Fatalf("expected query navigation to update search, got %q", parseGot2)
	}
}

func TestBrowserRouterAppliesRouteTitleAndRedirect(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/legacy"})
	js.Global().Get("location").Set("pathname", "/legacy")

	parseRouter.GoRegisterRoute("/dashboard", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("dashboard"))
	}, Options{Title: "Dashboard"})
	parseRouter.GoRegisterRoute("/legacy", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("legacy"))
	}, Options{Redirect: "/dashboard"})

	if parseRoute := parseRouter.GoGetRoute(); parseRoute == nil {
		parseT.Fatal("expected redirected history route element")
	}
	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/dashboard" {
		parseT.Fatalf("expected history redirect to replace pathname, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("document").Get("title").String(); parseGot2 != "Dashboard" {
		parseT.Fatalf("expected redirected history route to apply title Dashboard, got %q", parseGot2)
	}
}

func TestBrowserRouterBeforeLeaveBlocksNavigation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/edit"})
	js.Global().Get("location").Set("pathname", "/edit")

	parseRouter.GoRegisterRoute("/edit", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("edit"))
	}, Options{
		BeforeLeave: func(parseCurrent RouteContext, parseNext RouteContext) GuardResult {
			if parseNext.Path == "/dashboard" {
				return BlockNavigation("Unsaved changes")
			}
			return AllowNavigation()
		},
	})
	parseRouter.GoRegisterRoute("/dashboard", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("dashboard"))
	})

	parseRouter.Navigate("/dashboard")
	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/edit" {
		parseT.Fatalf("expected history before-leave guard to keep pathname /edit, got %q", parseGot)
	}
}

func TestBrowserRouterBeforeEnterRedirectsNavigation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/secure"})
	js.Global().Get("location").Set("pathname", "/secure")

	parseRouter.GoRegisterRoute("/login", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	}, Options{Title: "Login"})
	parseRouter.GoRegisterRoute("/secure", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			return RedirectNavigation("/login")
		},
	})

	if parseRoute := parseRouter.GoGetRoute(); parseRoute == nil {
		parseT.Fatal("expected guarded history route to redirect")
	}
	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/login" {
		parseT.Fatalf("expected before-enter redirect to update history pathname, got %q", parseGot)
	}
}

func TestBrowserRouterAsyncGuardDoubleNavigationDropsStaleAttempt(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/start"})
	js.Global().Get("location").Set("pathname", "/start")

	parseStarted := make(chan string, 2)
	parseReleaseFirst := make(chan struct{})
	parseRouter.GoRegisterRoute("/start", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("start"))
	}, Options{
		BeforeLeaveAsync: func(parseCtx context.Context, parseCurrent RouteContext, parseNext RouteContext) GuardDecision {
			parseStarted <- parseNext.Path
			if parseNext.Path == "/slow" {
				<-parseReleaseFirst
			}
			return GuardDecision{}
		},
	})
	parseRouter.GoRegisterRoute("/slow", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("slow"))
	})
	parseRouter.GoRegisterRoute("/fast", func(parseAttrs3 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("fast"))
	})

	parseDoneSlow := make(chan struct{})
	go func() {
		defer close(parseDoneSlow)
		parseRouter.Navigate("/slow")
	}()

	select {
	case parseGot := <-parseStarted:
		if parseGot != "/slow" {
			parseT.Fatalf("expected slow navigation to start first, got %q", parseGot)
		}
	case <-time.After(250 * time.Millisecond):
		parseT.Fatal("slow navigation did not start")
	}

	parseDoneFast := make(chan struct{})
	go func() {
		defer close(parseDoneFast)
		parseRouter.Navigate("/fast")
	}()

	waitForCondition(parseT, func() bool {
		return len(parseStarted) >= 1 && js.Global().Get("location").Get("pathname").String() == "/fast"
	})
	close(parseReleaseFirst)
	<-parseDoneSlow
	<-parseDoneFast

	if parseGot2 := js.Global().Get("location").Get("pathname").String(); parseGot2 != "/fast" {
		parseT.Fatalf("expected stale slow navigation to be ignored, got pathname %q", parseGot2)
	}
}

func TestBrowserRouterAsyncGuardBackAndForwardUsesFreshAttempt(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/beta"})
	js.Global().Get("location").Set("pathname", "/beta")
	js.Global().Get("history").Call("replaceState", nil, "", "/beta")

	parseReleaseBeta := make(chan struct{})
	parseBetaStarted := make(chan struct{}, 1)

	parseRouter.GoRegisterRoute("/beta", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("beta"))
	}, Options{
		BeforeEnterAsync: func(parseCtx context.Context, parseNext RouteContext) GuardDecision {
			if parseNext.Path == "/beta" {
				parseBetaStarted <- struct{}{}
				<-parseReleaseBeta
			}
			return GuardDecision{}
		},
	})
	parseRouter.GoRegisterRoute("/alpha", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("alpha"))
	})

	parseRouter.Navigate("/alpha")
	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/alpha" {
		parseT.Fatalf("expected alpha navigation to complete, got %q", parseGot)
	}

	parseBackDone := make(chan struct{})
	go func() {
		defer close(parseBackDone)
		js.Global().Get("history").Call("back")
	}()

	select {
	case <-parseBetaStarted:
	case <-time.After(250 * time.Millisecond):
		parseT.Fatal("back navigation did not start beta guard")
	}

	parseForwardDone := make(chan struct{})
	go func() {
		defer close(parseForwardDone)
		js.Global().Get("history").Call("forward")
	}()

	waitForCondition(parseT, func() bool {
		return js.Global().Get("location").Get("pathname").String() == "/alpha"
	})
	close(parseReleaseBeta)
	<-parseBackDone
	<-parseForwardDone

	if parseGot2 := js.Global().Get("location").Get("pathname").String(); parseGot2 != "/alpha" {
		parseT.Fatalf("expected stale back navigation to be ignored, got pathname %q", parseGot2)
	}
}

func TestBrowserRouterHistoryHashchangeRerendersCurrentRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/pricing"})
	js.Global().Get("location").Set("pathname", "/pricing")

	parseRenderCount := 0
	parseRouter.GoRegisterRoute("/pricing", func(parseAttrs Attrs) *Element {
		parseRenderCount++
		return runtime.Div(nil, runtime.Text(js.Global().Get("location").Get("hash").String()))
	})

	parseContainer := js.Global().Get("document").Call("createElement", "div")
	parseRouter.MountElement(parseContainer)
	if parseRenderCount == 0 {
		parseT.Fatal("expected initial route render")
	}

	js.Global().Get("location").Set("hash", "#faq")
	js.Global().Get("window").Call("dispatchEvent", map[string]interface{}{"type": "hashchange"})

	waitForCondition(parseT, func() bool {
		return parseRenderCount >= 2
	})
}

func TestBrowserRouterAsyncGuardDelaysLoaderUntilAllowed(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/start"})
	js.Global().Get("location").Set("pathname", "/start")

	parseReleaseGuard := make(chan struct{})
	parseGuardStarted := make(chan struct{}, 1)
	parseLoaderStarted := 0
	parseRouter.GoRegisterRoute("/start", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("start"))
	})
	parseRouter.GoRegisterRoute("/guarded", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("guarded"))
	}, Options{
		BeforeEnterAsync: func(parseCtx context.Context, parseNext RouteContext) GuardDecision {
			parseGuardStarted <- struct{}{}
			<-parseReleaseGuard
			return GuardDecision{}
		},
		Loader: func(parseCtx2 context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			parseLoaderStarted++
			return Attrs{"ready": true}, nil
		},
	})

	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseRouter.Navigate("/guarded")
	}()

	select {
	case <-parseGuardStarted:
	case <-time.After(250 * time.Millisecond):
		parseT.Fatal("guarded navigation did not start")
	}
	if parseLoaderStarted != 0 {
		parseT.Fatalf("expected loader to wait for guard release, got %d starts", parseLoaderStarted)
	}
	close(parseReleaseGuard)
	<-parseDone

	waitForCondition(parseT, func() bool {
		return js.Global().Get("location").Get("pathname").String() == "/guarded" && parseLoaderStarted == 1
	})
}

func TestBrowserRouterAsyncGuardUnmountCleanup(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/guarded"})
	js.Global().Get("location").Set("pathname", "/guarded")

	parseReleaseGuard := make(chan struct{})
	parseGuardStarted := make(chan struct{}, 1)
	parseOtherRendered := make(chan struct{}, 1)
	parseRouter.GoRegisterRoute("/guarded", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("guarded"))
	}, Options{
		BeforeEnterAsync: func(parseCtx context.Context, parseNext RouteContext) GuardDecision {
			parseGuardStarted <- struct{}{}
			<-parseReleaseGuard
			return GuardDecision{}
		},
	})
	parseRouter.GoRegisterRoute("/other", func(parseAttrs2 Attrs) *Element {
		parseOtherRendered <- struct{}{}
		return runtime.Div(nil, runtime.Text("other"))
	})

	parseContainer := js.Global().Get("document").Call("createElement", "div")
	renderDone := make(chan struct{})
	go func() {
		defer close(renderDone)
		parseRouter.MountElement(parseContainer)
	}()

	select {
	case <-parseGuardStarted:
	case <-time.After(250 * time.Millisecond):
		parseT.Fatal("guarded mount did not start")
	}

	parseRouter.Navigate("/other")
	select {
	case <-parseOtherRendered:
	case <-time.After(250 * time.Millisecond):
		parseT.Fatal("replacement route did not render while guarded mount was pending")
	}
	close(parseReleaseGuard)
	<-renderDone

	waitForCondition(parseT, func() bool {
		return js.Global().Get("location").Get("pathname").String() == "/other"
	})
}

func TestBrowserRouterReplacesMetadataAcrossRoutes(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/first"})
	js.Global().Get("location").Set("pathname", "/first")

	parseRouter.GoRegisterRoute("/first", func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("first"))
	}, Options{
		Title:        "First",
		Description:  "First description",
		CanonicalURL: "https://example.com/first",
	})
	parseRouter.GoRegisterRoute("/second", func(parseAttrs2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("second"))
	}, Options{
		Title:        "Second",
		Description:  "Second description",
		CanonicalURL: "https://example.com/second",
	})

	if parseRoute := parseRouter.GoGetRoute(); parseRoute == nil {
		parseT.Fatal("expected first route element")
	}
	parseRouter.Navigate("/second")
	parseDoc := js.Global().Get("document")
	if parseGot := parseDoc.Get("title").String(); parseGot != "Second" {
		parseT.Fatalf("expected route title Second after navigation, got %q", parseGot)
	}
	if parseGot2 := parseDoc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); parseGot2 != "Second description" {
		parseT.Fatalf("expected replaced description metadata, got %q", parseGot2)
	}
	if parseGot3 := parseDoc.Call("querySelector", `link[rel="canonical"]`).Get("attributes").Get("href").String(); parseGot3 != "https://example.com/second" {
		parseT.Fatalf("expected replaced canonical metadata, got %q", parseGot3)
	}
}
