//go:build js && wasm
// +build js,wasm

package router

import (
	"context"
	"errors"
	"net/url"
	"strings"
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

func collectElementText(elem *Element) string {
	if elem == nil {
		return ""
	}
	if elem.TextContent != "" {
		return elem.TextContent
	}
	var builder strings.Builder
	for _, child := range elem.Children {
		switch value := child.(type) {
		case *Element:
			builder.WriteString(collectElementText(value))
		case string:
			builder.WriteString(value)
		}
	}
	return builder.String()
}

// TestNewHashRouter tests hash router initialization
func TestNewHashRouter(t *testing.T) {
	r := NewHashRouter()

	if r == nil {
		t.Fatal("NewHashRouter returned nil")
	}
}

func TestRouteLoaderWritesFrameworkLogs(t *testing.T) {
	installRouterBrowserEnv(t)
	runtime.ClearLogs()
	runtime.ClearProfiling()
	defer runtime.ClearLogs()
	defer runtime.ClearProfiling()

	r := NewHashRouter()
	r.ensureLoaderResult("route:/users", func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
		return nil, errors.New("loader boom")
	}, RouteContext{Path: "/users"})

	waitForCondition(t, func() bool {
		return len(runtime.GetLogs()) >= 2
	})

	logs := runtime.GetLogs()
	foundStart := false
	foundFailure := false
	for _, entry := range logs {
		switch entry.Message {
		case "route loader started":
			foundStart = entry.Fields["path"] == "/users"
		case "route loader failed":
			foundFailure = entry.Fields["error"] == "loader boom"
		}
	}
	if !foundStart || !foundFailure {
		t.Fatalf("expected loader lifecycle logs, got %+v", logs)
	}

	profiling := runtime.GetGlobalRuntime().Inspect().Profiling
	profileStart := false
	profileFailure := false
	for _, event := range profiling.RecentEvents {
		if event.Domain != "router" || event.Name != "loader" || event.Target != "/users" {
			continue
		}
		if event.Phase == "start" {
			profileStart = true
		}
		if event.Phase == "error" {
			profileFailure = true
		}
	}
	if !profileStart || !profileFailure {
		t.Fatalf("expected loader profiling lifecycle events, got %+v", profiling.RecentEvents)
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

	r := NewHistoryRouter(RouterOptions{DefaultRoute: "home/"})
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

func TestHydrateMountReusesCachedNestedRouteLoaderData(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	layoutLoads := 0
	leafLoads := 0
	r.GoRegisterRoute("/dashboard", func(props Attrs) *Element {
		section := ""
		if data := UseRouteData(); data != nil {
			section, _ = data["section"].(string)
		}
		return runtime.Div(nil,
			runtime.Text("layout:"+section+"|"),
			GetOutlet(),
		)
	}, Options{
		Layout: true,
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			layoutLoads++
			return Attrs{"section": "dashboard"}, nil
		},
	})
	r.GoRegisterRoute("/dashboard/reports/:id", func(props Attrs) *Element {
		report := ""
		if data := UseRouteData(); data != nil {
			report, _ = data["report"].(string)
		}
		return runtime.Div(nil, runtime.Text("report:"+report))
	}, Options{
		Loader: func(ctx context.Context, routeCtx RouteContext) (Attrs, error) {
			leafLoads++
			return Attrs{"report": routeCtx.Params.Get("id")}, nil
		},
	})

	stack := r.resolveRouteStack("/dashboard/reports/7")
	if !stack.found || len(stack.routes) != 2 {
		t.Fatal("expected nested route stack for hydration reuse test")
	}
	r.loaderState.entries[buildLoaderKey(stack.routes[0].id, stack.routes[0].path, "")] = &loaderEntry{
		data: Attrs{"section": "dashboard"},
	}
	r.loaderState.entries[buildLoaderKey(stack.routes[1].id, stack.routes[1].path, "")] = &loaderEntry{
		data: Attrs{"report": "7"},
	}

	r.HydrateMount("#app")
	if layoutLoads != 0 || leafLoads != 0 {
		t.Fatalf("expected HydrateMount not to rerun cached loaders, got layout=%d leaf=%d", layoutLoads, leafLoads)
	}

	elem := r.Current()
	if elem == nil {
		t.Fatal("expected nested hydrated route element")
	}
	if got := collectElementText(elem); got != "layout:dashboard|report:7" {
		t.Fatalf("expected hydrated nested route output layout:dashboard|report:7, got %q", got)
	}
	if layoutLoads != 0 || leafLoads != 0 {
		t.Fatalf("expected cached loader reuse during first hydrated route read, got layout=%d leaf=%d", layoutLoads, leafLoads)
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
	globalRouter = NewHistoryRouter(RouterOptions{DefaultRoute: "/search"})
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

func TestLayoutRoutesRenderNestedOutlet(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	r.GoRegisterRoute("/dashboard", func(props Attrs) *Element {
		return runtime.Div(nil,
			runtime.Text("layout|"),
			GetOutlet(),
		)
	}, Options{Layout: true})
	r.GoRegisterRoute("/dashboard/reports/:id", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report:"+UseParams().Get("id")))
	})

	elem := r.Current()
	if elem == nil {
		t.Fatal("expected nested layout route element")
	}
	if got := collectElementText(elem); got != "layout|report:7" {
		t.Fatalf("expected nested layout output layout|report:7, got %q", got)
	}
	if GetOutlet() != nil {
		t.Fatal("expected outlet to be nil outside layout rendering")
	}
}

func TestLayoutRoutesScopeParamsPerLevel(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42/settings/profile")

	layoutParams := ""
	childParams := ""
	r.GoRegisterRoute("/users/:id", func(props Attrs) *Element {
		layoutParams = UseParams().Get("id") + ":" + UseParams().Get("tab")
		return runtime.Div(nil,
			runtime.Text("user:"+UseParams().Get("id")+"|"),
			GetOutlet(),
		)
	}, Options{Layout: true})
	r.GoRegisterRoute("/users/:id/settings/:tab", func(props Attrs) *Element {
		params := UseParams()
		childParams = params.Get("id") + ":" + params.Get("tab")
		return runtime.Div(nil, runtime.Text("tab:"+params.Get("tab")))
	})

	elem := r.Current()
	if elem == nil {
		t.Fatal("expected nested param route element")
	}
	if layoutParams != "42:" {
		t.Fatalf("expected layout params to expose only parent captures, got %q", layoutParams)
	}
	if childParams != "42:profile" {
		t.Fatalf("expected child params to expose merged captures, got %q", childParams)
	}
	if got := collectElementText(elem); got != "user:42|tab:profile" {
		t.Fatalf("expected nested param output user:42|tab:profile, got %q", got)
	}
}

func TestRoutesDoNotNestWithoutLayoutOption(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/docs/api")

	r.GoRegisterRoute("/docs", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("docs|"), GetOutlet())
	})
	r.GoRegisterRoute("/docs/api", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("api"))
	})

	elem := r.Current()
	if elem == nil {
		t.Fatal("expected child route element")
	}
	if got := collectElementText(elem); got != "api" {
		t.Fatalf("expected non-layout parent not to wrap child route, got %q", got)
	}
}

func TestLayoutRoutesScopeLoaderDataPerLevel(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	layoutData := ""
	childData := ""
	r.GoRegisterRoute("/dashboard", func(props Attrs) *Element {
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
	r.GoRegisterRoute("/dashboard/reports/:id", func(props Attrs) *Element {
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
		elem := r.Current()
		if elem == nil {
			return false
		}
		return layoutData == "dashboard" && childData == "7" && collectElementText(elem) == "layout:dashboard|report:7"
	})
}

func TestLayoutRoutesLeafMetadataOverridesParentMetadata(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	r.GoRegisterRoute("/dashboard", func(props Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true, Title: "Dashboard", Description: "Parent dashboard description"})
	r.GoRegisterRoute("/dashboard/reports/:id", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	}, Options{Title: "Report 7", Description: "Leaf report description"})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected nested route element for metadata test")
	}
	doc := js.Global().Get("document")
	if got := doc.Get("title").String(); got != "Report 7" {
		t.Fatalf("expected leaf route title Report 7, got %q", got)
	}
	if got := doc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); got != "Leaf report description" {
		t.Fatalf("expected leaf route description to win, got %q", got)
	}
}

func TestLayoutRouteBeforeEnterRedirectsLeafRoute(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	r.GoRegisterRoute("/login", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	}, Options{Title: "Login"})
	r.GoRegisterRoute("/dashboard", func(props Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{
		Layout: true,
		BeforeEnter: func(ctx RouteContext) GuardResult {
			return RedirectNavigation("/login")
		},
	})
	r.GoRegisterRoute("/dashboard/reports/:id", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected redirected route element")
	}
	if got := js.Global().Get("location").Get("hash").String(); got != "#/login" {
		t.Fatalf("expected layout before-enter redirect to update hash route, got %q", got)
	}
	if got := js.Global().Get("document").Get("title").String(); got != "Login" {
		t.Fatalf("expected redirected layout route to apply login title, got %q", got)
	}
}

func TestInspectCurrentRouteUsesLeafParamsWithLayoutRoutes(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	r.GoRegisterRoute("/dashboard", func(props Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	r.GoRegisterRoute("/dashboard/reports/:id", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	})

	r.Current()
	inspection := InspectCurrentRoute()
	if inspection.Path != "/dashboard/reports/7" {
		t.Fatalf("expected inspect path /dashboard/reports/7, got %q", inspection.Path)
	}
	if inspection.Params["id"] != "7" {
		t.Fatalf("expected inspect params to expose leaf id 7, got %q", inspection.Params["id"])
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

func TestHashRouterBeforeEnterUsesUnauthorizedFallback(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	entered := false
	r.GoRegisterRoute("/secure", func(props Attrs) *Element {
		entered = true
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnterAsync: func(ctx context.Context, routeCtx RouteContext) GuardDecision {
			return GuardDecision{Blocked: true, Denied: true, Reason: "Billing access required"}
		},
		Unauthorized: func(props Attrs) *Element {
			if !props["unauthorized"].(bool) || !props["denied"].(bool) {
				t.Fatalf("expected unauthorized guard props, got %#v", props)
			}
			return runtime.Div(nil, runtime.Text("unauthorized:"+props["reason"].(string)))
		},
	})

	elem := r.Current()
	if elem == nil {
		t.Fatal("expected unauthorized fallback element")
	}
	if entered {
		t.Fatal("expected unauthorized guard fallback to prevent route component render")
	}
	if got := collectElementText(elem); got != "unauthorized:Billing access required" {
		t.Fatalf("expected unauthorized fallback text, got %q", got)
	}
}

func TestHashRouterBeforeEnterUsesAuthorizingFallback(t *testing.T) {
	installRouterBrowserEnv(t)
	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	r.GoRegisterRoute("/secure", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnterAsync: func(ctx context.Context, routeCtx RouteContext) GuardDecision {
			return GuardDecision{Blocked: true, Retryable: true, Reason: "Session still loading"}
		},
		Authorizing: func(props Attrs) *Element {
			if !props["authorizing"].(bool) || !props["retryable"].(bool) {
				t.Fatalf("expected authorizing guard props, got %#v", props)
			}
			return runtime.Div(nil, runtime.Text("authorizing:"+props["reason"].(string)))
		},
	})

	elem := r.Current()
	if elem == nil {
		t.Fatal("expected authorizing fallback element")
	}
	if got := collectElementText(elem); got != "authorizing:Session still loading" {
		t.Fatalf("expected authorizing fallback text, got %q", got)
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

func TestHashRouterCleansServerManagedMetadataOnUntitledRoute(t *testing.T) {
	installRouterBrowserEnv(t)
	doc := js.Global().Get("document")
	head := doc.Get("head")

	title := doc.Call("createElement", "title")
	title.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	title.Set("textContent", "Server title")
	head.Call("appendChild", title)
	doc.Set("title", "Server title")

	description := doc.Call("createElement", "meta")
	description.Call("setAttribute", "name", "description")
	description.Call("setAttribute", "content", "Server description")
	description.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	head.Call("appendChild", description)

	canonical := doc.Call("createElement", "link")
	canonical.Call("setAttribute", "rel", "canonical")
	canonical.Call("setAttribute", "href", "https://example.com/server")
	canonical.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	head.Call("appendChild", canonical)

	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/plain")
	r.GoRegisterRoute("/plain", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("plain"))
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected plain route element")
	}
	if got := doc.Get("title").String(); got != "" {
		t.Fatalf("expected server-managed title cleanup to clear stale title, got %q", got)
	}
	if node := doc.Call("querySelector", `meta[name="description"][data-gwc-router-managed="true"]`); node.Truthy() {
		t.Fatal("expected managed description metadata to be removed")
	}
	if node := doc.Call("querySelector", `link[rel="canonical"][data-gwc-router-managed="true"]`); node.Truthy() {
		t.Fatal("expected managed canonical metadata to be removed")
	}
}

func TestHashRouterDedupesManagedHydratedMetadata(t *testing.T) {
	installRouterBrowserEnv(t)
	doc := js.Global().Get("document")
	head := doc.Get("head")

	appendManagedMeta := func(tag string, attrs map[string]string, text string) {
		node := doc.Call("createElement", tag)
		for key, value := range attrs {
			node.Call("setAttribute", key, value)
		}
		node.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
		if text != "" {
			node.Set("textContent", text)
		}
		head.Call("appendChild", node)
	}

	appendManagedMeta("title", map[string]string{}, "stale one")
	appendManagedMeta("title", map[string]string{}, "stale two")
	appendManagedMeta("meta", map[string]string{"name": "description", "content": "stale one"}, "")
	appendManagedMeta("meta", map[string]string{"name": "description", "content": "stale two"}, "")
	appendManagedMeta("link", map[string]string{"rel": "canonical", "href": "https://example.com/stale-one"}, "")
	appendManagedMeta("link", map[string]string{"rel": "canonical", "href": "https://example.com/stale-two"}, "")
	doc.Set("title", "stale two")

	r := NewHashRouter()
	js.Global().Get("location").Set("hash", "/landing")
	r.GoRegisterRoute("/landing", func(props Attrs) *Element {
		return runtime.Div(nil, runtime.Text("landing"))
	}, Options{
		Title:        "Landing",
		Description:  "Landing description",
		CanonicalURL: "https://example.com/landing",
	})

	if elem := r.Current(); elem == nil {
		t.Fatal("expected landing route element")
	}
	if got := doc.Call("querySelectorAll", `title[data-gwc-router-managed="true"]`).Get("length").Int(); got != 1 {
		t.Fatalf("expected exactly one managed title after dedupe, got %d", got)
	}
	if got := doc.Call("querySelectorAll", `meta[name="description"][data-gwc-router-managed="true"]`).Get("length").Int(); got != 1 {
		t.Fatalf("expected exactly one managed description after dedupe, got %d", got)
	}
	if got := doc.Call("querySelectorAll", `link[rel="canonical"][data-gwc-router-managed="true"]`).Get("length").Int(); got != 1 {
		t.Fatalf("expected exactly one managed canonical after dedupe, got %d", got)
	}
	if got := doc.Get("title").String(); got != "Landing" {
		t.Fatalf("expected deduped managed title to update to Landing, got %q", got)
	}
	if got := doc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); got != "Landing description" {
		t.Fatalf("expected deduped managed description to update, got %q", got)
	}
	if got := doc.Call("querySelector", `link[rel="canonical"]`).Get("attributes").Get("href").String(); got != "https://example.com/landing" {
		t.Fatalf("expected deduped managed canonical to update, got %q", got)
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

	r.Revalidate()
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
	if diagnostics[0].Code != "GWC-ROUTER-DUPLICATE-ROUTE" {
		t.Fatalf("expected duplicate route code, got %+v", diagnostics[0])
	}
	if diagnostics[0].Docs == "" || diagnostics[0].Remediation == "" || !diagnostics[0].Recoverable {
		t.Fatalf("expected duplicate route guidance, got %+v", diagnostics[0])
	}
}

func TestMakeRouteFactoryNilPanicIncludesActionableGuidance(t *testing.T) {
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("expected panic")
		}
		message := recovered.(string)
		if !strings.Contains(message, "GWC-ROUTER-COMPONENT-NIL") || !strings.Contains(message, "ACTIONABLE_ERRORS.md#gwc-router-component-nil") || !strings.Contains(message, "where:") || !strings.Contains(message, "runtime:") {
			t.Fatalf("expected actionable router panic, got %q", message)
		}
	}()

	_ = makeRouteFactory(nil)
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
	regularRouter := NewHistoryRouter(RouterOptions{})

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

func TestPreserveReturnToNormalizesInternalTarget(t *testing.T) {
	values := url.Values{"tab": {"security"}, "page": {"2"}}
	if got := PreserveReturnTo("settings", values); got != "/settings?page=2&tab=security" {
		t.Fatalf("expected normalized internal return target, got %q", got)
	}
}

func TestReadReturnToRejectsExternalTargets(t *testing.T) {
	values := url.Values{ReturnToParam: {"https://evil.example/phish"}}
	if got := ReadReturnTo(values, "/signin"); got != "/signin" {
		t.Fatalf("expected fallback for external return target, got %q", got)
	}
}
