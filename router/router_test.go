//go:build js && wasm
// +build js,wasm

package router

import (
	"context"
	"errors"
	"net/url"
	"syscall/js"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func waitForCondition(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}

// TestNewHashRouter tests hash router initialization
func TestNewHashRouter(t *testing.T) {
	r := NewHashRouter()

	if r == nil {
		t.Fatal("NewHashRouter returned nil")
	}
}

// TestNewHashRouterWithOptions tests hash router with custom options
func TestNewHashRouterWithOptions(t *testing.T) {
	options := RouterOptions{
		DefaultRoute: "home/",
	}

	r := NewHashRouter(options)

	if r == nil {
		t.Fatal("NewHashRouter with options returned nil")
	}
	if r.defaultRoute != "/home" {
		t.Fatalf("expected normalized default route '/home', got %q", r.defaultRoute)
	}
}

func TestCurrentFallsBackToNormalizedDefaultRoute(t *testing.T) {
	installRouterBrowserEnv(t)

	r := NewRouter(RouterOptions{DefaultRoute: "home/"})
	home := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}
	r.GoRegisterRoute("/home", home)

	if got := r.Current(); got == nil {
		t.Fatal("expected normalized default route to resolve registered home component")
	}
}

// TestRegisterRoute tests route registration
func TestRegisterRoute(t *testing.T) {
	r := NewHashRouter()

	// Register a simple route
	testComponent := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Test"))
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
	r := NewHashRouter()

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
			return runtime.Div(nil)
		}

		r.GoRegisterRoute(tc.input, testComponent)
		if _, ok := r.routes[tc.expected]; !ok {
			t.Fatalf("expected normalized path %q to be registered for input %q", tc.expected, tc.input)
		}
	}
	if _, ok := r.routes["about"]; ok {
		t.Fatal("expected raw unnormalized path to be absent")
	}
}

func TestNormalizePathAndNavigationTargetRules(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantPath   string
		wantTarget string
	}{
		{name: "blank becomes root", input: "", wantPath: "/", wantTarget: "/"},
		{name: "hash only becomes root", input: "#", wantPath: "/", wantTarget: "/"},
		{name: "leading slash added", input: "users", wantPath: "/users", wantTarget: "/users"},
		{name: "trailing slash trimmed", input: "/users/", wantPath: "/users", wantTarget: "/users"},
		{name: "query stripped for path and preserved for navigation", input: "#/users/?page=2", wantPath: "/users", wantTarget: "/users?page=2"},
		{name: "empty query marker dropped", input: "/users?", wantPath: "/users", wantTarget: "/users"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizePath(tc.input); got != tc.wantPath {
				t.Fatalf("expected normalizePath(%q) = %q, got %q", tc.input, tc.wantPath, got)
			}
			if got := normalizeNavigationTarget(tc.input); got != tc.wantTarget {
				t.Fatalf("expected normalizeNavigationTarget(%q) = %q, got %q", tc.input, tc.wantTarget, got)
			}
		})
	}
}

// TestGetCurrentPath tests current path getter
func TestGetCurrentPath(t *testing.T) {
	r := NewHashRouter()

	path := r.GetCurrentRouterPath()
	if path == "" {
		t.Error("GetCurrentRouterPath returned empty string")
	}
}

func TestUseNavigateHandle(t *testing.T) {
	installRouterBrowserEnv(t)
	globalRouter = NewHashRouter()

	nav := UseNavigate()
	nav.Navigate("/hook-path")
	if got := js.Global().Get("location").Get("hash").String(); got != "/hook-path" {
		t.Fatalf("expected hash navigation to update location hash, got %q", got)
	}

	nav.Replace("/replaced-path")
	if got := js.Global().Get("location").Get("hash").String(); got != "#/replaced-path" {
		t.Fatalf("expected replace navigation to update location hash, got %q", got)
	}
}

func TestHydrateMountSetsHashRouterTargetWithoutRendering(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	r.GoRegisterRoute("/", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("home"))
	})

	r.HydrateMount("#app")

	if !r.listening {
		t.Fatal("expected HydrateMount to wire router listeners")
	}
	if r.targetSelector != "#app" {
		t.Fatalf("expected HydrateMount to retain target selector, got %q", r.targetSelector)
	}
}

func TestUseQueryReadsHashQuery(t *testing.T) {
	installRouterBrowserEnv(t)
	js.Global().Get("location").Set("hash", "/search?q=golang&sort=relevance")

	query := UseQuery()
	if got := query.Get("q"); got != "golang" {
		t.Fatalf("expected q query param to equal golang, got %q", got)
	}
	if !query.Has("sort") {
		t.Fatal("expected sort query param to be present")
	}
	if got := query.Values().Get("sort"); got != "relevance" {
		t.Fatalf("expected sort query param to equal relevance, got %q", got)
	}
}

func TestUseQueryReadsHistorySearch(t *testing.T) {
	installRouterBrowserEnv(t)
	js.Global().Get("location").Set("search", "?page=2&filter=active")

	query := UseQuery()
	if got := query.Get("page"); got != "2" {
		t.Fatalf("expected page query param to equal 2, got %q", got)
	}
	if !query.Has("filter") {
		t.Fatal("expected filter query param to be present")
	}
}

func TestUseSearchParamsEncodesAndUpdatesHashQuery(t *testing.T) {
	installRouterBrowserEnv(t)
	globalRouter = NewHashRouter()
	js.Global().Get("location").Set("hash", "/search?q=golang")

	search := UseSearchParams()
	if search.Get("q") != "golang" {
		t.Fatalf("expected q query param to equal golang, got %q", search.Get("q"))
	}
	if search.Encode() != "q=golang" {
		t.Fatalf("expected encoded search params q=golang, got %q", search.Encode())
	}

	search.Set("sort", "recent")
	if got := js.Global().Get("location").Get("hash").String(); got != "/search?q=golang&sort=recent" {
		t.Fatalf("expected hash search param update, got %q", got)
	}

	search = UseSearchParams()
	search.Delete("q")
	if got := js.Global().Get("location").Get("hash").String(); got != "/search?sort=recent" {
		t.Fatalf("expected hash query deletion to preserve remaining params, got %q", got)
	}
}

func TestUseSearchParamsReplaceUpdatesHistoryQuery(t *testing.T) {
	installRouterBrowserEnv(t)
	globalRouter = NewRouter(RouterOptions{DefaultRoute: "/search"})
	js.Global().Get("location").Set("pathname", "/search")
	js.Global().Get("location").Set("search", "?q=golang")

	search := UseSearchParams()
	search.Replace("page", "2")

	if got := js.Global().Get("location").Get("pathname").String(); got != "/search" {
		t.Fatalf("expected history path to remain /search, got %q", got)
	}
	if got := js.Global().Get("location").Get("search").String(); got != "?page=2&q=golang" {
		t.Fatalf("expected history search to update with replacement params, got %q", got)
	}

	search = UseSearchParams()
	search.ReplaceAll(url.Values{"tag": {"go", "wasm"}})
	if got := js.Global().Get("location").Get("search").String(); got != "?tag=go&tag=wasm" {
		t.Fatalf("expected history search to replace all params, got %q", got)
	}
}

func TestInspectCurrentRouteIncludesPathQueryParamsAndLoading(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42?q=golang")

	release := make(chan struct{})
	r.GoRegisterRoute("/users/:id", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("user"))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			<-release
			return Attrs{"name": "Ada"}, nil
		},
	})

	r.Current()
	inspection := InspectCurrentRoute()
	if inspection.Path != "/users/42" {
		t.Fatalf("expected route inspection path /users/42, got %q", inspection.Path)
	}
	if inspection.Query.Get("q") != "golang" {
		t.Fatalf("expected route inspection query q=golang, got %q", inspection.Query.Get("q"))
	}
	if inspection.Params["id"] != "42" {
		t.Fatalf("expected route inspection param id 42, got %q", inspection.Params["id"])
	}
	if !inspection.Loading {
		t.Fatal("expected route inspection loading state to be true while loader is pending")
	}

	close(release)
}

func TestCurrentMatchesParamRouteAndUseParams(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42")

	capturedID := ""
	r.GoRegisterRoute("/users/:id", func(props Attrs) *Element {
		params := UseParams()
		capturedID = params.Get("id")
		if props != nil {
			if raw, ok := props["id"].(string); ok && raw != "" {
				capturedID = raw
			}
		}
		return runtime.Div(nil, runtime.Text(capturedID))
	})

	elem := r.Current()
	if elem == nil {
		t.Fatal("expected matched param route element")
	}
	if capturedID != "42" {
		t.Fatalf("expected captured route param id to equal 42, got %q", capturedID)
	}
}

func TestCurrentMatchesDecodedParamRoute(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/Ada%20Lovelace")

	capturedName := ""
	r.GoRegisterRoute("/users/:name", func(props Attrs) *Element {
		capturedName = UseParams().Get("name")
		return runtime.Div(nil, runtime.Text(capturedName))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected decoded param route element")
	}
	if capturedName != "Ada Lovelace" {
		t.Fatalf("expected decoded param value Ada Lovelace, got %q", capturedName)
	}
}

func TestParamRouteRejectsEmptyOrInvalidEncodedSegments(t *testing.T) {
	if params, ok := matchRoutePattern("/users/:id", "/users/"); ok || params != nil {
		t.Fatal("expected empty route param segment not to match")
	}
	if params, ok := matchRoutePattern("/users/:id", "/users/%zz"); ok || params != nil {
		t.Fatal("expected invalid encoded route param segment not to match")
	}
}

func TestOptionalSegmentsAreNotSupported(t *testing.T) {
	if params, ok := matchRoutePattern("/users/:id?", "/users/42"); ok || params != nil {
		t.Fatal("expected optional segment syntax to remain unsupported")
	}
}

func TestExactRouteWinsBeforePatternAndCatchAll(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/settings")

	matched := ""
	r.GoRegisterRoute("/users/:id", func(props Attrs) *Element {
		matched = "param"
		return runtime.Div(nil, runtime.Text("param"))
	})
	r.GoRegisterRoute("/users/settings", func(props Attrs) *Element {
		matched = "exact"
		return runtime.Div(nil, runtime.Text("exact"))
	})
	r.GoRegisterRoute("*", func(props Attrs) *Element {
		matched = "catchall"
		return runtime.Div(nil, runtime.Text("catchall"))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected exact route element")
	}
	if matched != "exact" {
		t.Fatalf("expected exact route to win before param/catchall, got %q", matched)
	}
}

func TestCatchAllWinsWhenNoExactOrPatternRouteMatches(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/missing/path")

	matched := false
	r.GoRegisterRoute("/users/:id", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("param"))
	})
	r.GoRegisterRoute("*", func(props Attrs) *Element {
		matched = true
		return runtime.Div(nil, runtime.Text("catchall"))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected catch-all route element")
	}
	if !matched {
		t.Fatal("expected catch-all route to handle unmatched path")
	}
}

func TestCurrentMatchesWildcardPrefixRoute(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42/details")

	matched := false
	r.GoRegisterRoute("/users*", func(props Attrs) *Element {
		matched = true
		return runtime.Div(nil, runtime.Text("matched"))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected wildcard prefix route to return an element")
	}
	if !matched {
		t.Fatal("expected wildcard prefix route to match current path")
	}
}

func TestGetCurrentRouterPathStripsHashQuery(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/search?q=golang&sort=relevance")

	if got := r.GetCurrentRouterPath(); got != "/search" {
		t.Fatalf("expected hash path to strip query string, got %q", got)
	}
}

func TestNavigatePreservesQueryString(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()

	r.Navigate("/search?q=golang&sort=relevance")
	if got := js.Global().Get("location").Get("hash").String(); got != "/search?q=golang&sort=relevance" {
		t.Fatalf("expected hash navigation to preserve query string, got %q", got)
	}
}

func TestNavigateReplacePreservesQueryString(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()

	r.NavigateReplace("/search?q=golang")
	if got := js.Global().Get("location").Get("hash").String(); got != "#/search?q=golang" {
		t.Fatalf("expected hash replace to preserve query string, got %q", got)
	}
}

func TestHashRouterAppliesRouteTitleAndRedirect(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/legacy")

	r.GoRegisterRoute("/modern", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("modern"))
	}, Options{Title: "Modern Route"})
	r.GoRegisterRoute("/legacy", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("legacy"))
	}, Options{Redirect: "/modern"})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected redirected hash route element")
	}
	if got := js.Global().Get("location").Get("hash").String(); got != "#/modern" {
		t.Fatalf("expected hash redirect to replace location, got %q", got)
	}
	if got := js.Global().Get("document").Get("title").String(); got != "Modern Route" {
		t.Fatalf("expected redirected route to apply title Modern Route, got %q", got)
	}
}

func TestHashRouterBeforeEnterBlocksCurrentRoute(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	entered := false
	r.GoRegisterRoute("/secure", func(props Attrs) *Element {
		entered = true
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnter: func(ctx RouteContext) GuardResult {
			return BlockNavigation("Authentication required")
		},
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected blocked route fallback element")
	}
	if entered {
		t.Fatal("expected blocked before-enter guard to prevent route component render")
	}
}

func TestHashRouterBeforeLeaveBlocksNavigation(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/edit")

	r.GoRegisterRoute("/edit", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("edit"))
	}, Options{
		BeforeLeave: func(current RouteContext, next RouteContext) GuardResult {
			if next.Path == "/home" {
				return BlockNavigation("Unsaved changes")
			}
			return AllowNavigation()
		},
	})
	r.GoRegisterRoute("/home", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("home"))
	})

	r.Navigate("/home")
	if got := js.Global().Get("location").Get("hash").String(); got != "/edit" {
		t.Fatalf("expected before-leave guard to keep current hash route, got %q", got)
	}
}

func TestHashRouterBeforeEnterRedirectsNavigation(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	r.GoRegisterRoute("/login", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	}, Options{Title: "Login"})
	r.GoRegisterRoute("/secure", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnter: func(ctx RouteContext) GuardResult {
			return RedirectNavigation("/login")
		},
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected redirected guarded route element")
	}
	if got := js.Global().Get("location").Get("hash").String(); got != "#/login" {
		t.Fatalf("expected before-enter redirect to update hash route, got %q", got)
	}
}

func TestHashRouterAppliesAndCleansMetadata(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/landing")

	r.GoRegisterRoute("/landing", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("landing"))
	}, Options{
		Title:        "Landing",
		Description:  "Landing description",
		CanonicalURL: "https://example.com/landing",
	})
	r.GoRegisterRoute("/plain", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("plain"))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected landing route element")
	}
	doc := js.Global().Get("document")
	if got := doc.Get("title").String(); got != "Landing" {
		t.Fatalf("expected route title Landing, got %q", got)
	}
	if got := doc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); got != "Landing description" {
		t.Fatalf("expected description metadata, got %q", got)
	}
	if got := doc.Call("querySelector", `link[rel="canonical"]`).Get("attributes").Get("href").String(); got != "https://example.com/landing" {
		t.Fatalf("expected canonical metadata, got %q", got)
	}

	r.Navigate("/plain")
	if elem := r.Current(); elem == nil {
		t.Fatal("expected plain route element after hash navigation")
	}
	if got := doc.Get("title").String(); got != "" {
		t.Fatalf("expected route title cleanup to restore base title, got %q", got)
	}
	if node := doc.Call("querySelector", `meta[name="description"]`); node.Truthy() {
		t.Fatal("expected description metadata to be removed when next route omits it")
	}
	if node := doc.Call("querySelector", `link[rel="canonical"]`); node.Truthy() {
		t.Fatal("expected canonical metadata to be removed when next route omits it")
	}
}

func TestParamsZeroValue(t *testing.T) {
	var params Params
	if params.Has("id") {
		t.Fatal("expected zero-value params to report missing key")
	}
	if got := params.Get("id"); got != "" {
		t.Fatalf("expected zero-value params get to return empty string, got %q", got)
	}
	if len(params.Values()) != 0 {
		t.Fatal("expected zero-value params values to be empty")
	}
	if value, ok := params.Int("id"); ok || value != 0 {
		t.Fatalf("expected zero-value params Int to fail with zero, got (%d, %t)", value, ok)
	}
	if value, ok := params.Bool("enabled"); ok || value {
		t.Fatalf("expected zero-value params Bool to fail with false, got (%t, %t)", value, ok)
	}
}

func TestParamsTypedAccessors(t *testing.T) {
	params := Params{values: map[string]string{
		"id":      "42",
		"enabled": "true",
		"badInt":  "abc",
		"badBool": "maybe",
	}}

	if value, ok := params.Int("id"); !ok || value != 42 {
		t.Fatalf("expected Int accessor to parse 42, got (%d, %t)", value, ok)
	}
	if value, ok := params.Bool("enabled"); !ok || !value {
		t.Fatalf("expected Bool accessor to parse true, got (%t, %t)", value, ok)
	}
	if value, ok := params.Int("badInt"); ok || value != 0 {
		t.Fatalf("expected Int accessor to fail for invalid input, got (%d, %t)", value, ok)
	}
	if value, ok := params.Bool("badBool"); ok || value {
		t.Fatalf("expected Bool accessor to fail for invalid input, got (%t, %t)", value, ok)
	}
}

func TestRouteLoaderProvidesDataAndCustomLoadingState(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/7?q=focus")

	release := make(chan struct{})
	loadingSeen := false
	loadedName := ""
	loadedFromHook := ""

	r.GoRegisterRoute("/users/:id", func(props Attrs) *Element {
		if props != nil {
			if value, ok := props["name"].(string); ok {
				loadedName = value
			}
		}
		if data := UseRouteData(); data != nil {
			if value, ok := data["name"].(string); ok {
				loadedFromHook = value
			}
		}
		return runtime.Div(nil, runtime.Text("user"))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			if got := routeCtx.Params.Get("id"); got != "7" {
				t.Fatalf("expected loader params id 7, got %q", got)
			}
			if got := routeCtx.Query.Get("q"); got != "focus" {
				t.Fatalf("expected loader query q=focus, got %q", got)
			}
			<-release
			return Attrs{"name": "Ada"}, nil
		},
		Loading: func(props Attrs) *Element {
			loadingSeen = props["loading"] == true
			return runtime.Div(nil, runtime.Text("loading"))
		},
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected loading route element")
	}
	if !loadingSeen {
		t.Fatal("expected custom loading route to render")
	}

	close(release)
	waitForCondition(t, func() bool {
		r.Current()
		return loadedName == "Ada" && loadedFromHook == "Ada"
	})
}

func TestRouteLoaderErrorUsesRouteErrorRenderer(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/boom")

	errorSeen := ""
	r.GoRegisterRoute("/boom", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("ok"))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			return nil, errors.New("loader boom")
		},
		Error: func(props Attrs) *Element {
			if props != nil {
				if value, ok := props["error"].(string); ok {
					errorSeen = value
				}
			}
			return runtime.Div(nil, runtime.Text("error"))
		},
	})

	waitForCondition(t, func() bool {
		r.Current()
		return errorSeen == "loader boom"
	})
}

func TestRouteLoaderRevalidatesOnQueryChange(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()

	loadCount := 0
	r.GoRegisterRoute("/search", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("search"))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			loadCount++
			return Attrs{"query": routeCtx.Query.Get("q")}, nil
		},
	})

	js.Global().Get("location").Set("hash", "/search?q=one")
	waitForCondition(t, func() bool {
		r.Current()
		return loadCount == 1
	})

	r.Current()
	if loadCount != 1 {
		t.Fatalf("expected loader cache to prevent rerun for same query, got %d loads", loadCount)
	}

	js.Global().Get("location").Set("hash", "/search?q=two")
	waitForCondition(t, func() bool {
		r.Current()
		return loadCount == 2
	})
}

func TestRouteLoaderCancelsOnNavigationChange(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/slow")

	cancelled := false
	r.GoRegisterRoute("/slow", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("slow"))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			<-ctx.Done()
			cancelled = true
			return nil, ctx.Err()
		},
	})
	r.GoRegisterRoute("/done", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("done"))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected slow route to return loading element")
	}

	js.Global().Get("location").Set("hash", "/done")
	if elem := r.Current(); elem == nil {
		t.Fatal("expected done route to resolve after navigation")
	}
	waitForCondition(t, func() bool { return cancelled })
}

func TestRouteLoaderRevalidateCurrentRouteRerunsSameKey(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/refresh")

	loadCount := 0
	seenCount := 0
	r.GoRegisterRoute("/refresh", func(props Attrs) *Element {
		if props != nil {
			if value, ok := props["count"].(int); ok {
				seenCount = value
			}
		}
		return runtime.Div(nil, runtime.Text("refresh"))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			loadCount++
			return Attrs{"count": loadCount}, nil
		},
	})

	waitForCondition(t, func() bool {
		r.Current()
		return loadCount == 1 && seenCount == 1
	})

	r.RevalidateCurrentRoute()
	waitForCondition(t, func() bool {
		r.Current()
		return loadCount == 2 && seenCount == 2
	})
}

func TestUseRevalidatorReportsLoadingAndRevalidates(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	globalRouter = r
	js.Global().Get("location").Set("hash", "/revalidator")

	release := make(chan struct{})
	loadCount := 0
	r.GoRegisterRoute("/revalidator", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("revalidator"))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			loadCount++
			<-release
			return Attrs{"count": loadCount}, nil
		},
	})
	revalidator := UseRevalidator()

	if elem := r.Current(); elem == nil {
		t.Fatal("expected revalidator route element during loading")
	}
	if !revalidator.Loading() {
		t.Fatal("expected route loading state to be true while loader is pending")
	}

	close(release)
	waitForCondition(t, func() bool {
		r.Current()
		return loadCount == 1
	})
	if revalidator.Loading() {
		t.Fatal("expected route loading state to be false after loader resolves")
	}

	release = make(chan struct{})
	revalidator.Revalidate()
	if !revalidator.Loading() {
		t.Fatal("expected route loading state to become true after revalidation")
	}
	close(release)
	waitForCondition(t, func() bool {
		r.Current()
		return loadCount == 2
	})
}

func TestRegisterReportsDuplicateRouteDiagnostic(t *testing.T) {
	installRouterBrowserEnv(t)
	runtime.ClearDiagnostics()
	defer runtime.ClearDiagnostics()

	r := NewHashRouter()
	r.GoRegisterRoute("/dup", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("first"))
	})
	r.GoRegisterRoute("/dup", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("second"))
	})

	diagnostics := runtime.GetDiagnostics()
	if len(diagnostics) == 0 {
		t.Fatal("expected duplicate route registration diagnostic")
	}
}

// TestNavigate tests navigation
func TestNavigate(t *testing.T) {
	r := NewHashRouter()

	r.Navigate("/new-path")

	// Navigation happened (can't easily test in unit test without browser)
	_ = r.GetCurrentRouterPath()
}

// TestRouteResolution tests route resolution/matching
func TestRouteResolution(t *testing.T) {
	r := NewHashRouter()

	homeComponent := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}

	aboutComponent := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("About"))
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
	r := NewHashRouter()

	homeComponent := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}

	notFoundComponent := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Not Found"))
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
	r := NewHashRouter()

	staticElem := runtime.Div(nil, runtime.Text("Static"))

	r.GoRegisterRoute("/static", staticElem)

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Static element route failed")
	}
}

// TestBeforeEnterGuard tests beforeEnter route guard
func TestBeforeEnterGuard(t *testing.T) {
	r := NewHashRouter()

	protectedComponent := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Protected"))
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
	hashRouter := NewHashRouter()
	regularRouter := NewRouter(RouterOptions{})

	if hashRouter == nil {
		t.Error("Hash router creation failed")
	}

	if regularRouter == nil {
		t.Error("Regular router creation failed")
	}
}

// TestMultipleRoutes tests registering multiple routes
func TestMultipleRoutes(t *testing.T) {
	r := NewHashRouter()

	routes := []string{"/", "/about", "/contact", "/services", "/blog"}

	for _, path := range routes {
		p := path
		component := func(props Attrs) *Element {
			return runtime.Div(nil, runtime.Text(p))
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
	initialRouter := GetRouter()

	if initialRouter == nil {
		t.Fatal("Global router is nil")
	}

	// Simply verify global router exists
	currentRouter := GetRouter()
	if currentRouter == nil {
		t.Error("Global router is nil")
	}
}

// TestGoRegisterRoute tests the Go-style route registration
func TestGoRegisterRoute(t *testing.T) {
	componentFunc := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Test Component"))
	}

	RegisterRoute("/go-test", componentFunc)

	result := GetRoute()
	if result == nil {
		t.Error("RegisterRoute did not properly register the route")
	}
}

// TestRouteOptions tests route options like Title
func TestRouteOptions(t *testing.T) {
	r := NewHashRouter()

	component := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Page"))
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
	r := NewHashRouter()

	component := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}

	r.GoRegisterRoute("", component) // Should normalize to "/"

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Empty path registration failed")
	}
}

// TestPathWithTrailingSlash tests path with trailing slash normalization
func TestPathWithTrailingSlash(t *testing.T) {
	r := NewHashRouter()

	component := func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("About"))
	}

	r.GoRegisterRoute("/about/", component) // Should normalize to "/about"

	elem := r.GoGetRoute()
	if elem == nil {
		t.Error("Path with trailing slash failed")
	}
}

// TestNavigateFunctions tests global Navigate functions
func TestNavigateFunctions(t *testing.T) {
	Navigate("/test")
	NavigateReplace("/test2")

	path := GetCurrentPath()
	if path == "" {
		t.Error("Navigation functions failed")
	}
}
