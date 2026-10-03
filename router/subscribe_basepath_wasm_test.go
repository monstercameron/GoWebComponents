//go:build js && wasm

package router

import (
	"syscall/js"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// isRouteLocationSubscribed reports whether any fiber is subscribed to the route
// location atom. DeleteAtomValueIfUnsubscribed only deletes (and returns true)
// when nobody is subscribed.
func isRouteLocationSubscribed() bool {
	return !runtime.GetGlobalRuntime().DeleteAtomValueIfUnsubscribed(routeLocationAtomID)
}

// TestRouteHooksSubscribeComponentsToNavigation proves UseParams, UseQuery,
// UseSearchParams and UseRouteData subscribe the calling component to the route
// location atom, so memoized components re-render on navigation like UseRoute.
func TestRouteHooksSubscribeComponentsToNavigation(parseT *testing.T) {
	parseHooks := map[string]func(){
		"UseParams":       func() { UseParams() },
		"UseQuery":        func() { UseQuery() },
		"UseSearchParams": func() { UseSearchParams() },
		"UseRouteData":    func() { UseRouteData() },
	}
	for parseName, parseHook := range parseHooks {
		parseT.Run(parseName, func(parseT2 *testing.T) {
			installRouterBrowserEnv(parseT2)
			lastPublishedLocationKey = ""
			globalRouter = NewHashRouter()
			registerLocationProbeRoutes(globalRouter)
			Navigate("/alpha")
			globalRouter.renderCurrentRoute(false)
			runtime.GetGlobalRuntime().DeleteAtomValueIfUnsubscribed(routeLocationAtomID)

			// Outside a component render the hook must not subscribe or panic.
			parseHook()
			if isRouteLocationSubscribed() {
				parseT2.Fatalf("%s outside a component must not subscribe", parseName)
			}

			// Inside a route factory the router fiber already re-renders.
			routeFactoryDepth++
			parseFactoryFiber := &runtime.Fiber{}
			runtime.SetCurrentFiber(parseFactoryFiber)
			parseHook()
			runtime.SetCurrentFiber(nil)
			routeFactoryDepth--
			runtime.GetGlobalRuntime().CleanupAtomSubscriptions(parseFactoryFiber)
			if isRouteLocationSubscribed() {
				parseT2.Fatalf("%s inside a route factory must not subscribe", parseName)
			}

			// Inside a component render it subscribes.
			parseFiber := &runtime.Fiber{}
			runtime.SetCurrentFiber(parseFiber)
			parseHook()
			runtime.SetCurrentFiber(nil)
			// Release this subtest's subscription so later subtests start clean.
			defer runtime.GetGlobalRuntime().CleanupAtomSubscriptions(parseFiber)
			if !isRouteLocationSubscribed() {
				parseT2.Fatalf("%s inside a component must subscribe to the route location atom", parseName)
			}
		})
	}
}

// TestUseParamsReadsCurrentParamsAfterNavigation proves a component-level UseParams
// call still returns the matched params after navigating between param routes.
func TestUseParamsReadsCurrentParamsAfterNavigation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	globalRouter = NewHashRouter()
	globalRouter.Register("/item/:id", func(Attrs) *Element { return runtime.Div(nil, runtime.Text("item")) })

	for _, parseID := range []string{"1", "2"} {
		Navigate("/item/" + parseID)
		globalRouter.renderCurrentRoute(false)
		parseFiber := &runtime.Fiber{}
		runtime.SetCurrentFiber(parseFiber)
		parseGot := UseParams().Get("id")
		runtime.SetCurrentFiber(nil)
		// Release the route-location subscription so later tests (and -count reruns) start clean.
		runtime.GetGlobalRuntime().CleanupAtomSubscriptions(parseFiber)
		if parseGot != parseID {
			parseT.Fatalf("UseParams id = %q, want %q", parseGot, parseID)
		}
	}
}

// TestLoaderCompletionRepublishesLocation proves a loader finishing without a path
// change publishes a new revision, so UseRouteData subscribers are notified.
func TestLoaderCompletionRepublishesLocation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)
	Navigate("/alpha")
	globalRouter.renderCurrentRoute(false)
	parseFirst := lastPublishedLocationKey

	routeDataRevision++
	globalRouter.renderCurrentRoute(false)
	if lastPublishedLocationKey == parseFirst {
		parseT.Fatal("a data revision bump must change the published location key")
	}
}

func TestWithBasePathHistoryRouter(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	js.Global().Get("location").Set("pathname", "/app/x")
	parseRouter := NewHistoryRouter(WithBasePath("/app/"))
	globalRouter = parseRouter
	parseRouter.Register("/x", func(Attrs) *Element { return runtime.Div(nil, runtime.Text("x-page")) })
	parseRouter.Register("/users/:id", func(Attrs) *Element { return runtime.Div(nil, runtime.Text("user")) })

	if parseGot := parseRouter.GetCurrentPath(); parseGot != "/x" {
		parseT.Fatalf("current path = %q, want /x (base stripped)", parseGot)
	}
	if parseElem := parseRouter.Current(); parseElem == nil || collectElementText(parseElem) != "x-page" {
		parseT.Fatal("expected /app/x to match pattern /x")
	}
	if parseGot := Href("/x"); parseGot != "/app/x" {
		parseT.Fatalf("Href = %q, want /app/x", parseGot)
	}

	parseRouter.Navigate("/users/9")
	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/app/users/9" {
		parseT.Fatalf("pathname after Navigate = %q, want /app/users/9", parseGot)
	}
	if parseGot := parseRouter.GetCurrentPath(); parseGot != "/users/9" {
		parseT.Fatalf("current path after Navigate = %q, want /users/9", parseGot)
	}
	if parseGot := currentLocationSnapshot().Path; parseGot != "/users/9" {
		parseT.Fatalf("location snapshot path = %q, want /users/9", parseGot)
	}

	parseRouter.NavigateReplace("/x?q=1")
	if parseGot := js.Global().Get("location").Get("pathname").String(); parseGot != "/app/x" {
		parseT.Fatalf("pathname after NavigateReplace = %q, want /app/x", parseGot)
	}
	if parseGot := FragmentHref("main"); parseGot != "/app/x?q=1#main" {
		parseT.Fatalf("FragmentHref = %q, want /app/x?q=1#main", parseGot)
	}
}

func TestWithBasePathIgnoredByHashRouter(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHashRouter(WithBasePath("/app"))
	if parseGot := Href("/x"); parseGot != "#/x" {
		parseT.Fatalf("hash Href = %q, want #/x", parseGot)
	}
}

func TestPathOutsideBasePathMatchesNothing(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	js.Global().Get("location").Set("pathname", "/other/x")
	parseRouter := NewHistoryRouter(WithBasePath("/app"))
	globalRouter = parseRouter
	parseRouter.Register("/x", func(Attrs) *Element { return runtime.Div(nil, runtime.Text("x-page")) })
	if parseElem := parseRouter.Current(); parseElem == nil || collectElementText(parseElem) != routeNotFoundText {
		parseT.Fatal("a pathname outside the base path must not match /x")
	}
}

func TestRegisterAppliesEveryRouteOption(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	parseRouter := NewHashRouter(WithDefaultRoute("/home"), RouterOptions{})
	if parseRouter.defaultRoute != "/home" {
		parseT.Fatalf("default route = %q, want /home", parseRouter.defaultRoute)
	}
	parseRouter.Register("/a", func(Attrs) *Element { return nil },
		Options{Title: "Struct"}, WithDescription("desc"), WithLayout(), WithTitle("Func"))
	parseOpt := parseRouter.routeOptions["/a"]
	if parseOpt.Title != "Func" || parseOpt.Description != "desc" || !parseOpt.Layout {
		parseT.Fatalf("merged options = %+v", parseOpt)
	}
}
