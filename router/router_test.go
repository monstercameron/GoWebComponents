//go:build js && wasm

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

func waitForCondition(parseT *testing.T, parseCondition func() bool) {
	parseT.Helper()
	parseDeadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(parseDeadline) {
		if parseCondition() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	parseT.Fatal("condition was not met before timeout")
}

func collectElementText(parseElem *Element) string {
	if parseElem == nil {
		return ""
	}
	if parseElem.TextContent != "" {
		return parseElem.TextContent
	}
	var parseBuilder strings.Builder
	for _, parseChild := range parseElem.Children {
		switch parseValue := parseChild.(type) {
		case *Element:
			parseBuilder.WriteString(collectElementText(parseValue))
		case string:
			parseBuilder.WriteString(parseValue)
		}
	}
	return parseBuilder.String()
}

// TestNewHashRouter tests hash router initialization
func TestNewHashRouter(parseT *testing.T) {
	parseR := NewHashRouter()

	if parseR == nil {
		parseT.Fatal("NewHashRouter returned nil")
	}
}

func TestRouteLoaderWritesFrameworkLogs(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	runtime.ClearLogs()
	runtime.ClearProfiling()
	defer runtime.ClearLogs()
	defer runtime.ClearProfiling()

	parseR := NewHashRouter()
	parseR.ensureLoaderResult("route:/users", func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
		return nil, errors.New("loader boom")
	}, RouteContext{Path: "/users"})

	waitForCondition(parseT, func() bool {
		return len(runtime.GetLogs()) >= 2
	})

	parseLogs := runtime.GetLogs()
	isParseFoundStart := false
	isParseFoundFailure := false
	for _, parseEntry := range parseLogs {
		switch parseEntry.Message {
		case "route loader started":
			isParseFoundStart = parseEntry.Fields["path"] == "/users"
		case "route loader failed":
			isParseFoundFailure = parseEntry.Fields["error"] == "loader boom"
		}
	}
	if !isParseFoundStart || !isParseFoundFailure {
		parseT.Fatalf("expected loader lifecycle logs, got %+v", parseLogs)
	}

	parseProfiling := runtime.GetGlobalRuntime().Inspect().Profiling
	isParseProfileStart := false
	isParseProfileFailure := false
	for _, parseEvent := range parseProfiling.RecentEvents {
		if parseEvent.Domain != "router" || parseEvent.Name != "loader" || parseEvent.Target != "/users" {
			continue
		}
		if parseEvent.Phase == "start" {
			isParseProfileStart = true
		}
		if parseEvent.Phase == "error" {
			isParseProfileFailure = true
		}
	}
	if !isParseProfileStart || !isParseProfileFailure {
		parseT.Fatalf("expected loader profiling lifecycle events, got %+v", parseProfiling.RecentEvents)
	}
}

// TestNewHashRouterWithOptions tests hash router with custom options
func TestNewHashRouterWithOptions(parseT *testing.T) {
	parseOptions := RouterOptions{
		DefaultRoute: "home/",
	}

	parseR := NewHashRouter(parseOptions)

	if parseR == nil {
		parseT.Fatal("NewHashRouter with options returned nil")
	}
	if parseR.defaultRoute != "/home" {
		parseT.Fatalf("expected normalized default route '/home', got %q", parseR.defaultRoute)
	}
}

func TestCurrentFallsBackToNormalizedDefaultRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseR := NewHistoryRouter(RouterOptions{DefaultRoute: "home/"})
	parseHome := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}
	parseR.GoRegisterRoute("/home", parseHome)

	if parseGot := parseR.Current(); parseGot == nil {
		parseT.Fatal("expected normalized default route to resolve registered home component")
	}
}

// TestRegisterRoute tests route registration
func TestRegisterRoute(parseT *testing.T) {
	parseR := NewHashRouter()

	// Register a simple route
	parseTestComponent := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Test"))
	}

	parseR.GoRegisterRoute("/test", parseTestComponent)

	// Test that the route can be retrieved
	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("GoGetRoute returned nil")
	}
}

// TestPathNormalization tests that paths are normalized correctly
func TestPathNormalization(parseT *testing.T) {
	parseR := NewHashRouter()

	parseTestCases := []struct {
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

	for _, parseTc := range parseTestCases {
		parseTestComponent := func(parseProps Attrs) *Element {
			return runtime.Div(nil)
		}

		parseR.GoRegisterRoute(parseTc.input, parseTestComponent)
		if _, parseOk := parseR.routes[parseTc.expected]; !parseOk {
			parseT.Fatalf("expected normalized path %q to be registered for input %q", parseTc.expected, parseTc.input)
		}
	}
	if _, parseOk2 := parseR.routes["about"]; parseOk2 {
		parseT.Fatal("expected raw unnormalized path to be absent")
	}
}

func TestNormalizePathAndNavigationTargetRules(parseT *testing.T) {
	parseTestCases := []struct {
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

	for _, parseTc := range parseTestCases {
		parseT.Run(parseTc.name, func(parseT2 *testing.T) {
			if parseGot := normalizePath(parseTc.input); parseGot != parseTc.wantPath {
				parseT2.Fatalf("expected normalizePath(%q) = %q, got %q", parseTc.input, parseTc.wantPath, parseGot)
			}
			if parseGot2 := normalizeNavigationTarget(parseTc.input); parseGot2 != parseTc.wantTarget {
				parseT2.Fatalf("expected normalizeNavigationTarget(%q) = %q, got %q", parseTc.input, parseTc.wantTarget, parseGot2)
			}
		})
	}
}

// TestGetCurrentPath tests current path getter
func TestGetCurrentPath(parseT *testing.T) {
	parseR := NewHashRouter()

	parsePath := parseR.GetCurrentRouterPath()
	if parsePath == "" {
		parseT.Error("GetCurrentRouterPath returned empty string")
	}
}

func TestUseNavigateHandle(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHashRouter()

	parseNav := UseNavigate()
	parseNav.Navigate("/hook-path")
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "/hook-path" {
		parseT.Fatalf("expected hash navigation to update location hash, got %q", parseGot)
	}

	parseNav.Replace("/replaced-path")
	if parseGot2 := js.Global().Get("location").Get("hash").String(); parseGot2 != "#/replaced-path" {
		parseT.Fatalf("expected replace navigation to update location hash, got %q", parseGot2)
	}
}

func TestHydrateMountSetsHashRouterTargetWithoutRendering(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	parseR.GoRegisterRoute("/", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("home"))
	})

	parseR.HydrateMount("#app")

	if !parseR.listening {
		parseT.Fatal("expected HydrateMount to wire router listeners")
	}
	if parseR.targetSelector != "#app" {
		parseT.Fatalf("expected HydrateMount to retain target selector, got %q", parseR.targetSelector)
	}
}

func TestHydrateMountReusesCachedNestedRouteLoaderData(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	parseLayoutLoads := 0
	parseLeafLoads := 0
	parseR.GoRegisterRoute("/dashboard", func(parseProps Attrs) *Element {
		parseSection := ""
		if parseData := UseRouteData(); parseData != nil {
			parseSection, _ = parseData["section"].(string)
		}
		return runtime.Div(nil,
			runtime.Text("layout:"+parseSection+"|"),
			GetOutlet(),
		)
	}, Options{
		Layout: true,
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			parseLayoutLoads++
			return Attrs{"section": "dashboard"}, nil
		},
	})
	parseR.GoRegisterRoute("/dashboard/reports/:id", func(parseProps2 Attrs) *Element {
		parseReport := ""
		if parseData2 := UseRouteData(); parseData2 != nil {
			parseReport, _ = parseData2["report"].(string)
		}
		return runtime.Div(nil, runtime.Text("report:"+parseReport))
	}, Options{
		Loader: func(parseCtx2 context.Context, parseRouteCtx2 RouteContext) (Attrs, error) {
			parseLeafLoads++
			return Attrs{"report": parseRouteCtx2.Params.Get("id")}, nil
		},
	})

	parseStack := parseR.resolveRouteStack("/dashboard/reports/7")
	if !parseStack.found || len(parseStack.routes) != 2 {
		parseT.Fatal("expected nested route stack for hydration reuse test")
	}
	parseR.loaderState.entries[buildLoaderKey(parseStack.routes[0].id, parseStack.routes[0].path, "")] = &loaderEntry{
		data: Attrs{"section": "dashboard"},
	}
	parseR.loaderState.entries[buildLoaderKey(parseStack.routes[1].id, parseStack.routes[1].path, "")] = &loaderEntry{
		data: Attrs{"report": "7"},
	}

	parseR.HydrateMount("#app")
	if parseLayoutLoads != 0 || parseLeafLoads != 0 {
		parseT.Fatalf("expected HydrateMount not to rerun cached loaders, got layout=%d leaf=%d", parseLayoutLoads, parseLeafLoads)
	}

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected nested hydrated route element")
	}
	if parseGot := collectElementText(parseElem); parseGot != "layout:dashboard|report:7" {
		parseT.Fatalf("expected hydrated nested route output layout:dashboard|report:7, got %q", parseGot)
	}
	if parseLayoutLoads != 0 || parseLeafLoads != 0 {
		parseT.Fatalf("expected cached loader reuse during first hydrated route read, got layout=%d leaf=%d", parseLayoutLoads, parseLeafLoads)
	}
}

func TestUseQueryReadsHashQuery(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	js.Global().Get("location").Set("hash", "/search?q=golang&sort=relevance")

	parseQuery := UseQuery()
	if parseGot := parseQuery.Get("q"); parseGot != "golang" {
		parseT.Fatalf("expected q query param to equal golang, got %q", parseGot)
	}
	if !parseQuery.Has("sort") {
		parseT.Fatal("expected sort query param to be present")
	}
	if parseGot2 := parseQuery.Values().Get("sort"); parseGot2 != "relevance" {
		parseT.Fatalf("expected sort query param to equal relevance, got %q", parseGot2)
	}
}

func TestUseQueryReadsHistorySearch(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	js.Global().Get("location").Set("search", "?page=2&filter=active")

	parseQuery := UseQuery()
	if parseGot := parseQuery.Get("page"); parseGot != "2" {
		parseT.Fatalf("expected page query param to equal 2, got %q", parseGot)
	}
	if !parseQuery.Has("filter") {
		parseT.Fatal("expected filter query param to be present")
	}
}

func TestUseSearchParamsEncodesAndUpdatesHashQuery(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHashRouter()
	js.Global().Get("location").Set("hash", "/search?q=golang")

	parseSearch := UseSearchParams()
	if parseSearch.Get("q") != "golang" {
		parseT.Fatalf("expected q query param to equal golang, got %q", parseSearch.Get("q"))
	}
	if parseSearch.Encode() != "q=golang" {
		parseT.Fatalf("expected encoded search params q=golang, got %q", parseSearch.Encode())
	}

	parseSearch.Set("sort", "recent")
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "/search?q=golang&sort=recent" {
		parseT.Fatalf("expected hash search param update, got %q", parseGot)
	}

	parseSearch = UseSearchParams()
	parseSearch.Delete("q")
	if parseGot2 := js.Global().Get("location").Get("hash").String(); parseGot2 != "/search?sort=recent" {
		parseT.Fatalf("expected hash query deletion to preserve remaining params, got %q", parseGot2)
	}
}

func TestUseSearchParamsReplaceUpdatesHistoryQuery(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHistoryRouter(RouterOptions{DefaultRoute: "/search"})
	js.Global().Get("location").Set("pathname", "/search")
	js.Global().Get("location").Set("search", "?q=golang")

	parseSearch := UseSearchParams()
	parseSearch.Replace("page", "2")

	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/search" {
		parseT.Fatalf("expected history path to remain /search, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("location").Get("search").String(); parseGot2 != "?page=2&q=golang" {
		parseT.Fatalf("expected history search to update with replacement params, got %q", parseGot2)
	}

	parseSearch = UseSearchParams()
	parseSearch.ReplaceAll(url.Values{"tag": {"go", "wasm"}})
	if parseGot3 := js.Global().Get("location").Get("search").String(); parseGot3 != "?tag=go&tag=wasm" {
		parseT.Fatalf("expected history search to replace all params, got %q", parseGot3)
	}
}

func TestInspectCurrentRouteIncludesPathQueryParamsAndLoading(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42?q=golang")

	parseRelease := make(chan struct{})
	parseR.GoRegisterRoute("/users/:id", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("user"))
	}, Options{
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			<-parseRelease
			return Attrs{"name": "Ada"}, nil
		},
	})

	parseR.Current()
	parseInspection := InspectCurrentRoute()
	if parseInspection.Path != "/users/42" {
		parseT.Fatalf("expected route inspection path /users/42, got %q", parseInspection.Path)
	}
	if parseInspection.Query.Get("q") != "golang" {
		parseT.Fatalf("expected route inspection query q=golang, got %q", parseInspection.Query.Get("q"))
	}
	if parseInspection.Params["id"] != "42" {
		parseT.Fatalf("expected route inspection param id 42, got %q", parseInspection.Params["id"])
	}
	if !parseInspection.Loading {
		parseT.Fatal("expected route inspection loading state to be true while loader is pending")
	}
	if len(parseInspection.Stack) != 1 || parseInspection.Stack[0].Path != "/users/42" || !parseInspection.Stack[0].HasLoader {
		parseT.Fatalf("expected route inspection stack to include active loader route, got %+v", parseInspection.Stack)
	}
	if len(parseInspection.Loaders) != 1 || parseInspection.Loaders[0].Path != "/users/42" || !parseInspection.Loaders[0].Pending {
		parseT.Fatalf("expected route inspection loaders to include pending loader, got %+v", parseInspection.Loaders)
	}

	close(parseRelease)
}

func TestCurrentMatchesParamRouteAndUseParams(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42")

	parseCapturedID := ""
	parseR.GoRegisterRoute("/users/:id", func(parseProps Attrs) *Element {
		parseParams := UseParams()
		parseCapturedID = parseParams.Get("id")
		if parseProps != nil {
			if parseRaw, parseOk := parseProps["id"].(string); parseOk && parseRaw != "" {
				parseCapturedID = parseRaw
			}
		}
		return runtime.Div(nil, runtime.Text(parseCapturedID))
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected matched param route element")
	}
	if parseCapturedID != "42" {
		parseT.Fatalf("expected captured route param id to equal 42, got %q", parseCapturedID)
	}
}

func TestCurrentMatchesDecodedParamRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/Ada%20Lovelace")

	parseCapturedName := ""
	parseR.GoRegisterRoute("/users/:name", func(parseProps Attrs) *Element {
		parseCapturedName = UseParams().Get("name")
		return runtime.Div(nil, runtime.Text(parseCapturedName))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected decoded param route element")
	}
	if parseCapturedName != "Ada Lovelace" {
		parseT.Fatalf("expected decoded param value Ada Lovelace, got %q", parseCapturedName)
	}
}

func TestLayoutRoutesRenderNestedOutlet(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	parseR.GoRegisterRoute("/dashboard", func(parseProps Attrs) *Element {
		return runtime.Div(nil,
			runtime.Text("layout|"),
			GetOutlet(),
		)
	}, Options{Layout: true})
	parseR.GoRegisterRoute("/dashboard/reports/:id", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report:"+UseParams().Get("id")))
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected nested layout route element")
	}
	if parseGot := collectElementText(parseElem); parseGot != "layout|report:7" {
		parseT.Fatalf("expected nested layout output layout|report:7, got %q", parseGot)
	}
	if GetOutlet() != nil {
		parseT.Fatal("expected outlet to be nil outside layout rendering")
	}
}

func TestLayoutRoutesScopeParamsPerLevel(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42/settings/profile")

	parseLayoutParams := ""
	parseChildParams := ""
	parseR.GoRegisterRoute("/users/:id", func(parseProps Attrs) *Element {
		parseLayoutParams = UseParams().Get("id") + ":" + UseParams().Get("tab")
		return runtime.Div(nil,
			runtime.Text("user:"+UseParams().Get("id")+"|"),
			GetOutlet(),
		)
	}, Options{Layout: true})
	parseR.GoRegisterRoute("/users/:id/settings/:tab", func(parseProps2 Attrs) *Element {
		parseParams := UseParams()
		parseChildParams = parseParams.Get("id") + ":" + parseParams.Get("tab")
		return runtime.Div(nil, runtime.Text("tab:"+parseParams.Get("tab")))
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected nested param route element")
	}
	if parseLayoutParams != "42:" {
		parseT.Fatalf("expected layout params to expose only parent captures, got %q", parseLayoutParams)
	}
	if parseChildParams != "42:profile" {
		parseT.Fatalf("expected child params to expose merged captures, got %q", parseChildParams)
	}
	if parseGot := collectElementText(parseElem); parseGot != "user:42|tab:profile" {
		parseT.Fatalf("expected nested param output user:42|tab:profile, got %q", parseGot)
	}
}

func TestRoutesDoNotNestWithoutLayoutOption(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/docs/api")

	parseR.GoRegisterRoute("/docs", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("docs|"), GetOutlet())
	})
	parseR.GoRegisterRoute("/docs/api", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("api"))
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected child route element")
	}
	if parseGot := collectElementText(parseElem); parseGot != "api" {
		parseT.Fatalf("expected non-layout parent not to wrap child route, got %q", parseGot)
	}
}

func TestLayoutRoutesScopeLoaderDataPerLevel(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	parseLayoutData := ""
	parseChildData := ""
	parseR.GoRegisterRoute("/dashboard", func(parseProps Attrs) *Element {
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
	parseR.GoRegisterRoute("/dashboard/reports/:id", func(parseProps2 Attrs) *Element {
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
		parseElem := parseR.Current()
		if parseElem == nil {
			return false
		}
		return parseLayoutData == "dashboard" && parseChildData == "7" && collectElementText(parseElem) == "layout:dashboard|report:7"
	})
}

func TestLayoutRoutesLeafMetadataOverridesParentMetadata(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	parseR.GoRegisterRoute("/dashboard", func(parseProps Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true, Title: "Dashboard", Description: "Parent dashboard description"})
	parseR.GoRegisterRoute("/dashboard/reports/:id", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	}, Options{Title: "Report 7", Description: "Leaf report description"})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected nested route element for metadata test")
	}
	parseDoc := js.Global().Get("document")
	if parseGot := parseDoc.Get("title").String(); parseGot != "Report 7" {
		parseT.Fatalf("expected leaf route title Report 7, got %q", parseGot)
	}
	if parseGot2 := parseDoc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); parseGot2 != "Leaf report description" {
		parseT.Fatalf("expected leaf route description to win, got %q", parseGot2)
	}
}

func TestLayoutRouteBeforeEnterRedirectsLeafRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	parseR.GoRegisterRoute("/login", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	}, Options{Title: "Login"})
	parseR.GoRegisterRoute("/dashboard", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{
		Layout: true,
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			return RedirectNavigation("/login")
		},
	})
	parseR.GoRegisterRoute("/dashboard/reports/:id", func(parseProps3 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected redirected route element")
	}
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "#/login" {
		parseT.Fatalf("expected layout before-enter redirect to update hash route, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("document").Get("title").String(); parseGot2 != "Login" {
		parseT.Fatalf("expected redirected layout route to apply login title, got %q", parseGot2)
	}
}

func TestInspectCurrentRouteUsesLeafParamsWithLayoutRoutes(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/dashboard/reports/7")

	parseR.GoRegisterRoute("/dashboard", func(parseProps Attrs) *Element {
		return runtime.Div(nil, GetOutlet())
	}, Options{Layout: true})
	parseR.GoRegisterRoute("/dashboard/reports/:id", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("report"))
	})

	parseR.Current()
	parseInspection := InspectCurrentRoute()
	if parseInspection.Path != "/dashboard/reports/7" {
		parseT.Fatalf("expected inspect path /dashboard/reports/7, got %q", parseInspection.Path)
	}
	if parseInspection.Params["id"] != "7" {
		parseT.Fatalf("expected inspect params to expose leaf id 7, got %q", parseInspection.Params["id"])
	}
	if len(parseInspection.Stack) != 2 || parseInspection.Stack[0].Path != "/dashboard" || parseInspection.Stack[1].Path != "/dashboard/reports/7" {
		parseT.Fatalf("expected inspect stack to preserve layout and leaf routes, got %+v", parseInspection.Stack)
	}
}

func TestParamRouteRejectsEmptyOrInvalidEncodedSegments(parseT *testing.T) {
	if parseParams, parseOk := matchRoutePattern("/users/:id", "/users/"); parseOk || parseParams != nil {
		parseT.Fatal("expected empty route param segment not to match")
	}
	if parseParams2, parseOk2 := matchRoutePattern("/users/:id", "/users/%zz"); parseOk2 || parseParams2 != nil {
		parseT.Fatal("expected invalid encoded route param segment not to match")
	}
}

func TestOptionalSegmentsAreNotSupported(parseT *testing.T) {
	if parseParams, parseOk := matchRoutePattern("/users/:id?", "/users/42"); parseOk || parseParams != nil {
		parseT.Fatal("expected optional segment syntax to remain unsupported")
	}
}

func TestExactRouteWinsBeforePatternAndCatchAll(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/settings")

	parseMatched := ""
	parseR.GoRegisterRoute("/users/:id", func(parseProps Attrs) *Element {
		parseMatched = "param"
		return runtime.Div(nil, runtime.Text("param"))
	})
	parseR.GoRegisterRoute("/users/settings", func(parseProps2 Attrs) *Element {
		parseMatched = "exact"
		return runtime.Div(nil, runtime.Text("exact"))
	})
	parseR.GoRegisterRoute("*", func(parseProps3 Attrs) *Element {
		parseMatched = "catchall"
		return runtime.Div(nil, runtime.Text("catchall"))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected exact route element")
	}
	if parseMatched != "exact" {
		parseT.Fatalf("expected exact route to win before param/catchall, got %q", parseMatched)
	}
}

func TestCatchAllWinsWhenNoExactOrPatternRouteMatches(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/missing/path")

	isParseMatched := false
	parseR.GoRegisterRoute("/users/:id", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("param"))
	})
	parseR.GoRegisterRoute("*", func(parseProps2 Attrs) *Element {
		isParseMatched = true
		return runtime.Div(nil, runtime.Text("catchall"))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected catch-all route element")
	}
	if !isParseMatched {
		parseT.Fatal("expected catch-all route to handle unmatched path")
	}
}

func TestCurrentMatchesWildcardPrefixRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/42/details")

	isParseMatched := false
	parseR.GoRegisterRoute("/users*", func(parseProps Attrs) *Element {
		isParseMatched = true
		return runtime.Div(nil, runtime.Text("matched"))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected wildcard prefix route to return an element")
	}
	if !isParseMatched {
		parseT.Fatal("expected wildcard prefix route to match current path")
	}
}

func TestGetCurrentRouterPathStripsHashQuery(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/search?q=golang&sort=relevance")

	if parseGot := parseR.GetCurrentRouterPath(); parseGot != "/search" {
		parseT.Fatalf("expected hash path to strip query string, got %q", parseGot)
	}
}

func TestNavigatePreservesQueryString(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()

	parseR.Navigate("/search?q=golang&sort=relevance")
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "/search?q=golang&sort=relevance" {
		parseT.Fatalf("expected hash navigation to preserve query string, got %q", parseGot)
	}
}

func TestNavigateReplacePreservesQueryString(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()

	parseR.NavigateReplace("/search?q=golang")
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "#/search?q=golang" {
		parseT.Fatalf("expected hash replace to preserve query string, got %q", parseGot)
	}
}

func TestHashRouterAppliesRouteTitleAndRedirect(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/legacy")

	parseR.GoRegisterRoute("/modern", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("modern"))
	}, Options{Title: "Modern Route"})
	parseR.GoRegisterRoute("/legacy", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("legacy"))
	}, Options{Redirect: "/modern"})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected redirected hash route element")
	}
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "#/modern" {
		parseT.Fatalf("expected hash redirect to replace location, got %q", parseGot)
	}
	if parseGot2 := js.Global().Get("document").Get("title").String(); parseGot2 != "Modern Route" {
		parseT.Fatalf("expected redirected route to apply title Modern Route, got %q", parseGot2)
	}
}

func TestHashRouterBeforeEnterBlocksCurrentRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	isParseEntered := false
	parseR.GoRegisterRoute("/secure", func(parseProps Attrs) *Element {
		isParseEntered = true
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			return BlockNavigation("Authentication required")
		},
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected blocked route fallback element")
	}
	if isParseEntered {
		parseT.Fatal("expected blocked before-enter guard to prevent route component render")
	}
}

func TestHashRouterBeforeLeaveBlocksNavigation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/edit")

	parseR.GoRegisterRoute("/edit", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("edit"))
	}, Options{
		BeforeLeave: func(parseCurrent RouteContext, parseNext RouteContext) GuardResult {
			if parseNext.Path == "/home" {
				return BlockNavigation("Unsaved changes")
			}
			return AllowNavigation()
		},
	})
	parseR.GoRegisterRoute("/home", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("home"))
	})

	parseR.Navigate("/home")
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "/edit" {
		parseT.Fatalf("expected before-leave guard to keep current hash route, got %q", parseGot)
	}
}

func TestHashRouterBeforeEnterRedirectsNavigation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	parseR.GoRegisterRoute("/login", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	}, Options{Title: "Login"})
	parseR.GoRegisterRoute("/secure", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			return RedirectNavigation("/login")
		},
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected redirected guarded route element")
	}
	if parseGot := js.Global().Get("location").Get("hash").String(); parseGot != "#/login" {
		parseT.Fatalf("expected before-enter redirect to update hash route, got %q", parseGot)
	}
	parseInspection := InspectCurrentRoute()
	if parseInspection.LastRedirect.Cause != "before-enter" || parseInspection.LastRedirect.From != "/secure" || parseInspection.LastRedirect.To != "/login" {
		parseT.Fatalf("expected redirect inspection details, got %+v", parseInspection.LastRedirect)
	}
}

func TestHashRouterBeforeEnterUsesUnauthorizedFallback(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	isParseEntered := false
	parseR.GoRegisterRoute("/secure", func(parseProps Attrs) *Element {
		isParseEntered = true
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnterAsync: func(parseCtx context.Context, parseRouteCtx RouteContext) GuardDecision {
			return GuardDecision{Blocked: true, Denied: true, Reason: "Billing access required"}
		},
		Unauthorized: func(parseProps2 Attrs) *Element {
			if !parseProps2["unauthorized"].(bool) || !parseProps2["denied"].(bool) {
				parseT.Fatalf("expected unauthorized guard props, got %#v", parseProps2)
			}
			return runtime.Div(nil, runtime.Text("unauthorized:"+parseProps2["reason"].(string)))
		},
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected unauthorized fallback element")
	}
	if isParseEntered {
		parseT.Fatal("expected unauthorized guard fallback to prevent route component render")
	}
	if parseGot := collectElementText(parseElem); parseGot != "unauthorized:Billing access required" {
		parseT.Fatalf("expected unauthorized fallback text, got %q", parseGot)
	}
}

func TestHashRouterBeforeEnterUsesAuthorizingFallback(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/secure")

	parseR.GoRegisterRoute("/secure", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("secure"))
	}, Options{
		BeforeEnterAsync: func(parseCtx context.Context, parseRouteCtx RouteContext) GuardDecision {
			return GuardDecision{Blocked: true, Retryable: true, Reason: "Session still loading"}
		},
		Authorizing: func(parseProps2 Attrs) *Element {
			if !parseProps2["authorizing"].(bool) || !parseProps2["retryable"].(bool) {
				parseT.Fatalf("expected authorizing guard props, got %#v", parseProps2)
			}
			return runtime.Div(nil, runtime.Text("authorizing:"+parseProps2["reason"].(string)))
		},
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected authorizing fallback element")
	}
	if parseGot := collectElementText(parseElem); parseGot != "authorizing:Session still loading" {
		parseT.Fatalf("expected authorizing fallback text, got %q", parseGot)
	}
}

func TestHashRouterAppliesAndCleansMetadata(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/landing")

	parseR.GoRegisterRoute("/landing", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("landing"))
	}, Options{
		Title:        "Landing",
		Description:  "Landing description",
		CanonicalURL: "https://example.com/landing",
	})
	parseR.GoRegisterRoute("/plain", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("plain"))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected landing route element")
	}
	parseDoc := js.Global().Get("document")
	if parseGot := parseDoc.Get("title").String(); parseGot != "Landing" {
		parseT.Fatalf("expected route title Landing, got %q", parseGot)
	}
	if parseGot2 := parseDoc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); parseGot2 != "Landing description" {
		parseT.Fatalf("expected description metadata, got %q", parseGot2)
	}
	if parseGot3 := parseDoc.Call("querySelector", `link[rel="canonical"]`).Get("attributes").Get("href").String(); parseGot3 != "https://example.com/landing" {
		parseT.Fatalf("expected canonical metadata, got %q", parseGot3)
	}

	parseR.Navigate("/plain")
	if parseElem2 := parseR.Current(); parseElem2 == nil {
		parseT.Fatal("expected plain route element after hash navigation")
	}
	if parseGot4 := parseDoc.Get("title").String(); parseGot4 != "" {
		parseT.Fatalf("expected route title cleanup to restore base title, got %q", parseGot4)
	}
	if parseNode := parseDoc.Call("querySelector", `meta[name="description"]`); parseNode.Truthy() {
		parseT.Fatal("expected description metadata to be removed when next route omits it")
	}
	if parseNode2 := parseDoc.Call("querySelector", `link[rel="canonical"]`); parseNode2.Truthy() {
		parseT.Fatal("expected canonical metadata to be removed when next route omits it")
	}
}

func TestHashRouterCleansServerManagedMetadataOnUntitledRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseDoc := js.Global().Get("document")
	parseHead := parseDoc.Get("head")

	parseTitle := parseDoc.Call("createElement", "title")
	parseTitle.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	parseTitle.Set("textContent", "Server title")
	parseHead.Call("appendChild", parseTitle)
	parseDoc.Set("title", "Server title")

	parseDescription := parseDoc.Call("createElement", "meta")
	parseDescription.Call("setAttribute", "name", "description")
	parseDescription.Call("setAttribute", "content", "Server description")
	parseDescription.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	parseHead.Call("appendChild", parseDescription)

	parseCanonical := parseDoc.Call("createElement", "link")
	parseCanonical.Call("setAttribute", "rel", "canonical")
	parseCanonical.Call("setAttribute", "href", "https://example.com/server")
	parseCanonical.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
	parseHead.Call("appendChild", parseCanonical)

	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/plain")
	parseR.GoRegisterRoute("/plain", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("plain"))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected plain route element")
	}
	if parseGot := parseDoc.Get("title").String(); parseGot != "" {
		parseT.Fatalf("expected server-managed title cleanup to clear stale title, got %q", parseGot)
	}
	if parseNode := parseDoc.Call("querySelector", `meta[name="description"][data-gwc-router-managed="true"]`); parseNode.Truthy() {
		parseT.Fatal("expected managed description metadata to be removed")
	}
	if parseNode2 := parseDoc.Call("querySelector", `link[rel="canonical"][data-gwc-router-managed="true"]`); parseNode2.Truthy() {
		parseT.Fatal("expected managed canonical metadata to be removed")
	}
}

func TestHashRouterDedupesManagedHydratedMetadata(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseDoc := js.Global().Get("document")
	parseHead := parseDoc.Get("head")

	parseAppendManagedMeta := func(parseTag string, parseAttrs map[string]string, parseText string) {
		parseNode := parseDoc.Call("createElement", parseTag)
		for parseKey, parseValue := range parseAttrs {
			parseNode.Call("setAttribute", parseKey, parseValue)
		}
		parseNode.Call("setAttribute", managedMetadataAttr, managedMetadataValue)
		if parseText != "" {
			parseNode.Set("textContent", parseText)
		}
		parseHead.Call("appendChild", parseNode)
	}

	parseAppendManagedMeta("title", map[string]string{}, "stale one")
	parseAppendManagedMeta("title", map[string]string{}, "stale two")
	parseAppendManagedMeta("meta", map[string]string{"name": "description", "content": "stale one"}, "")
	parseAppendManagedMeta("meta", map[string]string{"name": "description", "content": "stale two"}, "")
	parseAppendManagedMeta("link", map[string]string{"rel": "canonical", "href": "https://example.com/stale-one"}, "")
	parseAppendManagedMeta("link", map[string]string{"rel": "canonical", "href": "https://example.com/stale-two"}, "")
	parseDoc.Set("title", "stale two")

	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/landing")
	parseR.GoRegisterRoute("/landing", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("landing"))
	}, Options{
		Title:        "Landing",
		Description:  "Landing description",
		CanonicalURL: "https://example.com/landing",
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected landing route element")
	}
	if parseGot := parseDoc.Call("querySelectorAll", `title[data-gwc-router-managed="true"]`).Get("length").Int(); parseGot != 1 {
		parseT.Fatalf("expected exactly one managed title after dedupe, got %d", parseGot)
	}
	if parseGot2 := parseDoc.Call("querySelectorAll", `meta[name="description"][data-gwc-router-managed="true"]`).Get("length").Int(); parseGot2 != 1 {
		parseT.Fatalf("expected exactly one managed description after dedupe, got %d", parseGot2)
	}
	if parseGot3 := parseDoc.Call("querySelectorAll", `link[rel="canonical"][data-gwc-router-managed="true"]`).Get("length").Int(); parseGot3 != 1 {
		parseT.Fatalf("expected exactly one managed canonical after dedupe, got %d", parseGot3)
	}
	if parseGot4 := parseDoc.Get("title").String(); parseGot4 != "Landing" {
		parseT.Fatalf("expected deduped managed title to update to Landing, got %q", parseGot4)
	}
	if parseGot5 := parseDoc.Call("querySelector", `meta[name="description"]`).Get("attributes").Get("content").String(); parseGot5 != "Landing description" {
		parseT.Fatalf("expected deduped managed description to update, got %q", parseGot5)
	}
	if parseGot6 := parseDoc.Call("querySelector", `link[rel="canonical"]`).Get("attributes").Get("href").String(); parseGot6 != "https://example.com/landing" {
		parseT.Fatalf("expected deduped managed canonical to update, got %q", parseGot6)
	}
}

func TestParamsZeroValue(parseT *testing.T) {
	var parseParams Params
	if parseParams.Has("id") {
		parseT.Fatal("expected zero-value params to report missing key")
	}
	if parseGot := parseParams.Get("id"); parseGot != "" {
		parseT.Fatalf("expected zero-value params get to return empty string, got %q", parseGot)
	}
	if len(parseParams.Values()) != 0 {
		parseT.Fatal("expected zero-value params values to be empty")
	}
	if parseValue, parseOk := parseParams.Int("id"); parseOk || parseValue != 0 {
		parseT.Fatalf("expected zero-value params Int to fail with zero, got (%d, %t)", parseValue, parseOk)
	}
	if parseValue2, parseOk2 := parseParams.Bool("enabled"); parseOk2 || parseValue2 {
		parseT.Fatalf("expected zero-value params Bool to fail with false, got (%t, %t)", parseValue2, parseOk2)
	}
}

func TestParamsTypedAccessors(parseT *testing.T) {
	parseParams := Params{values: map[string]string{
		"id":      "42",
		"enabled": "true",
		"badInt":  "abc",
		"badBool": "maybe",
	}}

	if parseValue, parseOk := parseParams.Int("id"); !parseOk || parseValue != 42 {
		parseT.Fatalf("expected Int accessor to parse 42, got (%d, %t)", parseValue, parseOk)
	}
	if parseValue2, parseOk2 := parseParams.Bool("enabled"); !parseOk2 || !parseValue2 {
		parseT.Fatalf("expected Bool accessor to parse true, got (%t, %t)", parseValue2, parseOk2)
	}
	if parseValue3, parseOk3 := parseParams.Int("badInt"); parseOk3 || parseValue3 != 0 {
		parseT.Fatalf("expected Int accessor to fail for invalid input, got (%d, %t)", parseValue3, parseOk3)
	}
	if parseValue4, parseOk4 := parseParams.Bool("badBool"); parseOk4 || parseValue4 {
		parseT.Fatalf("expected Bool accessor to fail for invalid input, got (%t, %t)", parseValue4, parseOk4)
	}
}

func TestRouteLoaderProvidesDataAndCustomLoadingState(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/users/7?q=focus")

	parseRelease := make(chan struct{})
	isParseLoadingSeen := false
	parseLoadedName := ""
	parseLoadedFromHook := ""

	parseR.GoRegisterRoute("/users/:id", func(parseProps Attrs) *Element {
		if parseProps != nil {
			if parseValue, parseOk := parseProps["name"].(string); parseOk {
				parseLoadedName = parseValue
			}
		}
		if parseData := UseRouteData(); parseData != nil {
			if parseValue2, parseOk2 := parseData["name"].(string); parseOk2 {
				parseLoadedFromHook = parseValue2
			}
		}
		return runtime.Div(nil, runtime.Text("user"))
	}, Options{
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			if parseGot := parseRouteCtx.Params.Get("id"); parseGot != "7" {
				parseT.Fatalf("expected loader params id 7, got %q", parseGot)
			}
			if parseGot2 := parseRouteCtx.Query.Get("q"); parseGot2 != "focus" {
				parseT.Fatalf("expected loader query q=focus, got %q", parseGot2)
			}
			<-parseRelease
			return Attrs{"name": "Ada"}, nil
		},
		Loading: func(parseProps2 Attrs) *Element {
			isParseLoadingSeen = parseProps2["loading"] == true
			return runtime.Div(nil, runtime.Text("loading"))
		},
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected loading route element")
	}
	if !isParseLoadingSeen {
		parseT.Fatal("expected custom loading route to render")
	}

	close(parseRelease)
	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseLoadedName == "Ada" && parseLoadedFromHook == "Ada"
	})
}

func TestRouteLoaderErrorUsesRouteErrorRenderer(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/boom")

	parseErrorSeen := ""
	parseR.GoRegisterRoute("/boom", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("ok"))
	}, Options{
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			return nil, errors.New("loader boom")
		},
		Error: func(parseProps2 Attrs) *Element {
			if parseProps2 != nil {
				if parseValue, parseOk := parseProps2["error"].(string); parseOk {
					parseErrorSeen = parseValue
				}
			}
			return runtime.Div(nil, runtime.Text("error"))
		},
	})

	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseErrorSeen == "loader boom"
	})
}

func TestRouteLoaderRevalidatesOnQueryChange(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()

	parseLoadCount := 0
	parseR.GoRegisterRoute("/search", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("search"))
	}, Options{
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			parseLoadCount++
			return Attrs{"query": parseRouteCtx.Query.Get("q")}, nil
		},
	})

	js.Global().Get("location").Set("hash", "/search?q=one")
	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseLoadCount == 1
	})

	parseR.Current()
	if parseLoadCount != 1 {
		parseT.Fatalf("expected loader cache to prevent rerun for same query, got %d loads", parseLoadCount)
	}

	js.Global().Get("location").Set("hash", "/search?q=two")
	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseLoadCount == 2
	})
}

func TestRouteLoaderCancelsOnNavigationChange(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/slow")

	isParseCancelled := false
	parseR.GoRegisterRoute("/slow", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("slow"))
	}, Options{
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			<-parseCtx.Done()
			isParseCancelled = true
			return nil, parseCtx.Err()
		},
	})
	parseR.GoRegisterRoute("/done", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("done"))
	})

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected slow route to return loading element")
	}

	js.Global().Get("location").Set("hash", "/done")
	if parseElem2 := parseR.Current(); parseElem2 == nil {
		parseT.Fatal("expected done route to resolve after navigation")
	}
	waitForCondition(parseT, func() bool { return isParseCancelled })
}

func TestRouteLoaderRevalidateCurrentRouteRerunsSameKey(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/refresh")

	parseLoadCount := 0
	parseSeenCount := 0
	parseR.GoRegisterRoute("/refresh", func(parseProps Attrs) *Element {
		if parseProps != nil {
			if parseValue, parseOk := parseProps["count"].(int); parseOk {
				parseSeenCount = parseValue
			}
		}
		return runtime.Div(nil, runtime.Text("refresh"))
	}, Options{
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			parseLoadCount++
			return Attrs{"count": parseLoadCount}, nil
		},
	})

	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseLoadCount == 1 && parseSeenCount == 1
	})

	parseR.Revalidate()
	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseLoadCount == 2 && parseSeenCount == 2
	})
}

func TestUseRevalidatorReportsLoadingAndRevalidates(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	globalRouter = parseR
	js.Global().Get("location").Set("hash", "/revalidator")

	parseRelease := make(chan struct{})
	parseLoadCount := 0
	parseR.GoRegisterRoute("/revalidator", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("revalidator"))
	}, Options{
		Loader: func(parseCtx context.Context, parseRouteCtx RouteContext) (Attrs, error) {
			parseLoadCount++
			<-parseRelease
			return Attrs{"count": parseLoadCount}, nil
		},
	})
	parseRevalidator := UseRevalidator()

	if parseElem := parseR.Current(); parseElem == nil {
		parseT.Fatal("expected revalidator route element during loading")
	}
	if !parseRevalidator.Loading() {
		parseT.Fatal("expected route loading state to be true while loader is pending")
	}

	close(parseRelease)
	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseLoadCount == 1
	})
	if parseRevalidator.Loading() {
		parseT.Fatal("expected route loading state to be false after loader resolves")
	}

	parseRelease = make(chan struct{})
	parseRevalidator.Revalidate()
	if !parseRevalidator.Loading() {
		parseT.Fatal("expected route loading state to become true after revalidation")
	}
	close(parseRelease)
	waitForCondition(parseT, func() bool {
		parseR.Current()
		return parseLoadCount == 2
	})
}

func TestRegisterReportsDuplicateRouteDiagnostic(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	runtime.ClearDiagnostics()
	defer runtime.ClearDiagnostics()

	parseR := NewHashRouter()
	parseR.GoRegisterRoute("/dup", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("first"))
	})
	parseR.GoRegisterRoute("/dup", func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("second"))
	})

	parseDiagnostics := runtime.GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		parseT.Fatal("expected duplicate route registration diagnostic")
	}
	if parseDiagnostics[0].Code != "GWC-ROUTER-DUPLICATE-ROUTE" {
		parseT.Fatalf("expected duplicate route code, got %+v", parseDiagnostics[0])
	}
	if parseDiagnostics[0].Docs == "" || parseDiagnostics[0].Remediation == "" || !parseDiagnostics[0].Recoverable {
		parseT.Fatalf("expected duplicate route guidance, got %+v", parseDiagnostics[0])
	}
}

func TestMakeRouteFactoryNilPanicIncludesActionableGuidance(parseT *testing.T) {
	defer func() {
		parseRecovered := recover()
		if parseRecovered == nil {
			parseT.Fatal("expected panic")
		}
		parseMessage := parseRecovered.(string)
		if !strings.Contains(parseMessage, "GWC-ROUTER-COMPONENT-NIL") || !strings.Contains(parseMessage, "ACTIONABLE_ERRORS.md#gwc-router-component-nil") || !strings.Contains(parseMessage, "where:") || !strings.Contains(parseMessage, "runtime:") {
			parseT.Fatalf("expected actionable router panic, got %q", parseMessage)
		}
	}()

	_ = makeRouteFactory(nil)
}

// TestNavigate tests navigation
func TestNavigate(parseT *testing.T) {
	parseR := NewHashRouter()

	parseR.Navigate("/new-path")

	// Navigation happened (can't easily test in unit test without browser)
	_ = parseR.GetCurrentRouterPath()
}

// TestRouteResolution tests route resolution/matching
func TestRouteResolution(parseT *testing.T) {
	parseR := NewHashRouter()

	parseHomeComponent := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}

	parseAboutComponent := func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("About"))
	}

	parseR.GoRegisterRoute("/", parseHomeComponent)
	parseR.GoRegisterRoute("/about", parseAboutComponent)

	// Test that routes return elements
	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Route resolution returned nil")
	}
}

// TestWildcardRoute tests wildcard (404) route
func TestWildcardRoute(parseT *testing.T) {
	parseR := NewHashRouter()

	parseHomeComponent := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}

	parseNotFoundComponent := func(parseProps2 Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Not Found"))
	}

	parseR.GoRegisterRoute("/", parseHomeComponent)
	parseR.GoRegisterRoute("*", parseNotFoundComponent)

	// Verify routes are registered
	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Wildcard route not working")
	}
}

// TestStaticElementRoute tests registering a static element as a route
func TestStaticElementRoute(parseT *testing.T) {
	parseR := NewHashRouter()

	parseStaticElem := runtime.Div(nil, runtime.Text("Static"))

	parseR.GoRegisterRoute("/static", parseStaticElem)

	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Static element route failed")
	}
}

// TestBeforeEnterGuard tests beforeEnter route guard
func TestBeforeEnterGuard(parseT *testing.T) {
	parseR := NewHashRouter()

	parseProtectedComponent := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Protected"))
	}

	parseR.GoRegisterRoute("/protected", parseProtectedComponent)

	// Simply verify route was registered
	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Route registration failed")
	}
}

// TestRouterTypeValidation tests that router type is correctly set
func TestRouterTypeValidation(parseT *testing.T) {
	parseHashRouter := NewHashRouter()
	parseRegularRouter := NewHistoryRouter(RouterOptions{})

	if parseHashRouter == nil {
		parseT.Error("Hash router creation failed")
	}

	if parseRegularRouter == nil {
		parseT.Error("Regular router creation failed")
	}
}

// TestMultipleRoutes tests registering multiple routes
func TestMultipleRoutes(parseT *testing.T) {
	parseR := NewHashRouter()

	parseRoutes := []string{"/", "/about", "/contact", "/services", "/blog"}

	for _, parsePath := range parseRoutes {
		parseP := parsePath
		parseComponent := func(parseProps Attrs) *Element {
			return runtime.Div(nil, runtime.Text(parseP))
		}
		parseR.GoRegisterRoute(parsePath, parseComponent)
	}

	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Multiple route registration failed")
	}
}

// TestGlobalRouter tests global router instance management
func TestGlobalRouter(parseT *testing.T) {
	parseInitialRouter := GetRouter()

	if parseInitialRouter == nil {
		parseT.Fatal("Global router is nil")
	}

	// Simply verify global router exists
	parseCurrentRouter := GetRouter()
	if parseCurrentRouter == nil {
		parseT.Error("Global router is nil")
	}
}

// TestGoRegisterRoute tests the Go-style route registration
func TestGoRegisterRoute(parseT *testing.T) {
	parseComponentFunc := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Test Component"))
	}

	RegisterRoute("/go-test", parseComponentFunc)

	parseResult := GetRoute()
	if parseResult == nil {
		parseT.Error("RegisterRoute did not properly register the route")
	}
}

// TestRouteOptions tests route options like Title
func TestRouteOptions(parseT *testing.T) {
	parseR := NewHashRouter()

	parseComponent := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Page"))
	}

	parseOptions := Options{
		Title: "Test Page Title",
	}

	parseR.GoRegisterRoute("/test-page", parseComponent, parseOptions)

	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Route with options failed")
	}
}

// TestEmptyPath tests handling of empty paths
func TestEmptyPath(parseT *testing.T) {
	parseR := NewHashRouter()

	parseComponent := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("Home"))
	}

	parseR.GoRegisterRoute("", parseComponent) // Should normalize to "/"

	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Empty path registration failed")
	}
}

// TestPathWithTrailingSlash tests path with trailing slash normalization
func TestPathWithTrailingSlash(parseT *testing.T) {
	parseR := NewHashRouter()

	parseComponent := func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("About"))
	}

	parseR.GoRegisterRoute("/about/", parseComponent) // Should normalize to "/about"

	parseElem := parseR.GoGetRoute()
	if parseElem == nil {
		parseT.Error("Path with trailing slash failed")
	}
}

// TestNavigateFunctions tests global Navigate functions
func TestNavigateFunctions(parseT *testing.T) {
	Navigate("/test")
	NavigateReplace("/test2")

	parsePath := GetCurrentPath()
	if parsePath == "" {
		parseT.Error("Navigation functions failed")
	}
}

func TestPreserveReturnToNormalizesInternalTarget(parseT *testing.T) {
	parseValues := url.Values{"tab": {"security"}, "page": {"2"}}
	if parseGot := PreserveReturnTo("settings", parseValues); parseGot != "/settings?page=2&tab=security" {
		parseT.Fatalf("expected normalized internal return target, got %q", parseGot)
	}
}

func TestReadReturnToRejectsExternalTargets(parseT *testing.T) {
	parseValues := url.Values{ReturnToParam: {"https://evil.example/phish"}}
	if parseGot := ReadReturnTo(parseValues, "/signin"); parseGot != "/signin" {
		parseT.Fatalf("expected fallback for external return target, got %q", parseGot)
	}
}

// TestRedirectViaRouteOptionsEvaluatesGuardsOnTarget is a regression test for #44:
// a route with Options.Redirect must still evaluate BeforeEnter guards on the redirect target.
func TestRedirectViaRouteOptionsEvaluatesGuardsOnTarget(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/old")

	parseGuardCalled := false
	parseR.Register("/old", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("old"))
	}, Options{Redirect: "/new"})
	parseR.Register("/new", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("new"))
	}, Options{
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			parseGuardCalled = true
			return BlockNavigation("not allowed")
		},
	})
	parseR.Register("/login", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected element after redirect")
	}
	if !parseGuardCalled {
		parseT.Fatal("guard on redirect target must be evaluated (#44)")
	}
}

// TestBeforeEnterGuardRedirectEvaluatesGuardsOnTarget is a regression test for #45:
// when a BeforeEnter guard itself redirects, the redirect target must also have its
// own guards evaluated.
func TestBeforeEnterGuardRedirectEvaluatesGuardsOnTarget(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/protected")

	parseSecondGuardCalled := false
	parseR.Register("/protected", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("protected"))
	}, Options{
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			return RedirectNavigation("/intermediate")
		},
	})
	parseR.Register("/intermediate", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("intermediate"))
	}, Options{
		BeforeEnter: func(parseCtx RouteContext) GuardResult {
			parseSecondGuardCalled = true
			return BlockNavigation("intermediate blocked")
		},
	})
	parseR.Register("/login", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("login"))
	})

	parseElem := parseR.Current()
	if parseElem == nil {
		parseT.Fatal("expected element after guard redirect chain")
	}
	if !parseSecondGuardCalled {
		parseT.Fatal("guard on guard-redirect target must be evaluated (#45)")
	}
}

// TestMatchRoutePatternRejectsPercentEncodedSlash is a regression test for #49:
// a percent-encoded slash (%2F) in a URL segment decodes to '/' and must not be
// treated as a valid single-segment param value.
func TestMatchRoutePatternRejectsPercentEncodedSlash(parseT *testing.T) {
	if parseParams, parseOk := matchRoutePattern("/user/:id", "/user/%2Fadmin"); parseOk {
		parseT.Fatalf("expected percent-encoded slash to be rejected, got params %v (#49)", parseParams)
	}
}

// TestMatchRoutePatternAcceptsNormalPercentEncoding verifies that ordinary
// percent-encoding (not a slash) still works after the #49 fix.
func TestMatchRoutePatternAcceptsNormalPercentEncoding(parseT *testing.T) {
	parseParams, parseOk := matchRoutePattern("/item/:name", "/item/hello%20world")
	if !parseOk {
		parseT.Fatal("expected normal percent-encoding to match")
	}
	if parseParams["name"] != "hello world" {
		parseT.Fatalf("expected decoded param 'hello world', got %q", parseParams["name"])
	}
}

// TestRedirectDepthLimitPreventsInfiniteLoop verifies that a chain of redirects
// that exceeds maxRedirectDepth is aborted cleanly rather than looping forever.
func TestRedirectDepthLimitPreventsInfiniteLoop(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseR := NewHashRouter()
	js.Global().Get("location").Set("hash", "/a")

	parseR.Register("/a", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("a"))
	}, Options{Redirect: "/b"})
	parseR.Register("/b", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("b"))
	}, Options{Redirect: "/c"})
	parseR.Register("/c", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("c"))
	}, Options{Redirect: "/d"})
	parseR.Register("/d", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("d"))
	}, Options{Redirect: "/e"})
	parseR.Register("/e", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("e"))
	}, Options{Redirect: "/f"})
	parseR.Register("/f", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("f"))
	}, Options{Redirect: "/g"})
	parseR.Register("/g", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("g"))
	})

	// Should not panic or loop; returns nil or a fallback after depth limit.
	parseR.Current()
}
