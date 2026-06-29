//go:build js && wasm

package router

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func registerLocationProbeRoutes(parseR *Router) {
	parseR.GoRegisterRoute("/alpha", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("alpha"))
	})
	parseR.GoRegisterRoute("/beta", func(parseProps Attrs) *Element {
		return runtime.Div(nil, runtime.Text("beta"))
	})
}

// TestRenderPublishesActiveLocation proves the router publishes the active
// Location into the route atom on each render, so UseRoute subscribers observe
// the current path (G6).
func TestRenderPublishesActiveLocation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)

	Navigate("/alpha")
	globalRouter.renderCurrentRoute(false)

	parseValue, parseOk := runtime.GetGlobalRuntime().GetAtomValue(routeLocationAtomID)
	if !parseOk {
		parseT.Fatal("expected route location atom to be published")
	}
	if parseLoc, _ := parseValue.(Location); parseLoc.Path != "/alpha" {
		parseT.Fatalf("expected published path /alpha, got %q", parseLoc.Path)
	}

	Navigate("/beta")
	globalRouter.renderCurrentRoute(false)

	parseValue2, _ := runtime.GetGlobalRuntime().GetAtomValue(routeLocationAtomID)
	if parseLoc2, _ := parseValue2.(Location); parseLoc2.Path != "/beta" {
		parseT.Fatalf("expected published path /beta after navigation, got %q", parseLoc2.Path)
	}
}

// TestUseRouteReadsPublishedLocation proves UseRoute returns the live published
// Location (subscribe + read), so a component re-rendered after navigation sees
// the new path.
func TestUseRouteReadsPublishedLocation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)

	Navigate("/beta")
	globalRouter.renderCurrentRoute(false)

	parseFiber := &runtime.Fiber{}
	runtime.SetCurrentFiber(parseFiber)
	parseLoc := UseRoute()
	runtime.SetCurrentFiber(nil)

	if parseLoc.Path != "/beta" {
		parseT.Fatalf("UseRoute returned %q, want /beta", parseLoc.Path)
	}
}

// TestPublishLocationDedupesUnchangedLocation proves an unchanged location does
// not overwrite the atom (no redundant re-render notifications), while a real
// navigation does.
func TestPublishLocationDedupesUnchangedLocation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)

	Navigate("/alpha")
	globalRouter.renderCurrentRoute(false)

	// Overwrite the atom with a sentinel; a dedup'd publish must leave it intact.
	parseRt := runtime.GetGlobalRuntime()
	_ = parseRt.SetAtomValue(routeLocationAtomID, Location{Path: "SENTINEL"})

	// Same location -> publishLocation should be a no-op (dedup).
	globalRouter.renderCurrentRoute(false)
	parseValue, _ := parseRt.GetAtomValue(routeLocationAtomID)
	if parseLoc, _ := parseValue.(Location); parseLoc.Path != "SENTINEL" {
		parseT.Fatalf("unchanged location should be dedup'd, sentinel overwritten with %q", parseLoc.Path)
	}

	// A real navigation must publish again.
	Navigate("/beta")
	globalRouter.renderCurrentRoute(false)
	parseValue2, _ := parseRt.GetAtomValue(routeLocationAtomID)
	if parseLoc2, _ := parseValue2.(Location); parseLoc2.Path != "/beta" {
		parseT.Fatalf("navigation after dedup should publish, got %q", parseLoc2.Path)
	}
}

// TestOnNavigateFiresOnNavigation proves an OnNavigate callback runs with the
// new Location on each real navigation (G28).
func TestOnNavigateFiresOnNavigation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	navigationCallbacks = map[int]func(Location){}
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)

	var parseSeen []string
	parseStop := OnNavigate(func(parseLoc Location) {
		parseSeen = append(parseSeen, parseLoc.Path)
	})
	defer parseStop()

	Navigate("/alpha")
	globalRouter.renderCurrentRoute(false)
	Navigate("/beta")
	globalRouter.renderCurrentRoute(false)

	if len(parseSeen) != 2 || parseSeen[0] != "/alpha" || parseSeen[1] != "/beta" {
		parseT.Fatalf("expected [/alpha /beta], got %v", parseSeen)
	}
}

// TestOnNavigateUnsubscribeStops proves the returned unsubscribe func stops
// further callbacks.
func TestOnNavigateUnsubscribeStops(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	navigationCallbacks = map[int]func(Location){}
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)

	parseCount := 0
	parseStop := OnNavigate(func(Location) { parseCount++ })

	Navigate("/alpha")
	globalRouter.renderCurrentRoute(false)
	parseStop()
	Navigate("/beta")
	globalRouter.renderCurrentRoute(false)

	if parseCount != 1 {
		parseT.Fatalf("expected exactly 1 callback before unsubscribe, got %d", parseCount)
	}
}

// TestOnNavigateDedupesUnchangedLocation proves the callback does not fire when
// the location is unchanged.
func TestOnNavigateDedupesUnchangedLocation(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	navigationCallbacks = map[int]func(Location){}
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)

	parseCount := 0
	OnNavigate(func(Location) { parseCount++ })

	Navigate("/alpha")
	globalRouter.renderCurrentRoute(false)
	globalRouter.renderCurrentRoute(false) // same location -> no fire
	globalRouter.renderCurrentRoute(false)

	if parseCount != 1 {
		parseT.Fatalf("expected 1 callback for one real navigation, got %d", parseCount)
	}
}

// TestHrefHashRouterPrefixesHash proves Href builds a hash-mode href (G7).
func TestHrefHashRouterPrefixesHash(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHashRouter()
	if parseGot := Href("/transactions"); parseGot != "#/transactions" {
		parseT.Fatalf("hash router Href = %q, want #/transactions", parseGot)
	}
}

// TestHrefHistoryRouterReturnsPath proves Href returns the bare path for history
// mode (resolved against <base href> by the browser).
func TestHrefHistoryRouterReturnsPath(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHistoryRouter()
	if parseGot := Href("/transactions"); parseGot != "/transactions" {
		parseT.Fatalf("history router Href = %q, want /transactions", parseGot)
	}
}

// TestFragmentHrefEmbedsCurrentPath proves an in-page anchor embeds the live path
// so it is safe under <base href> (G7).
func TestFragmentHrefEmbedsCurrentPath(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHistoryRouter()
	registerLocationProbeRoutes(globalRouter)
	Navigate("/beta")

	parseGot := FragmentHref("main")
	if parseGot != "/beta#main" {
		parseT.Fatalf("FragmentHref = %q, want /beta#main", parseGot)
	}
	// A leading '#' on the fragment is tolerated (not doubled).
	if parseGot2 := FragmentHref("#main"); parseGot2 != "/beta#main" {
		parseT.Fatalf("FragmentHref('#main') = %q, want /beta#main", parseGot2)
	}
}

// TestFragmentHrefHashRouterNoDoubleHash proves FragmentHref does not emit a
// malformed "#path#fragment" under a hash router (regression).
func TestFragmentHrefHashRouterNoDoubleHash(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)
	Navigate("/beta")

	parseGot := FragmentHref("main")
	if strings.Count(parseGot, "#") != 1 {
		parseT.Fatalf("FragmentHref under hash router has %d '#' (want 1): %q", strings.Count(parseGot, "#"), parseGot)
	}
	if parseGot != "/beta#main" {
		parseT.Fatalf("FragmentHref = %q, want /beta#main", parseGot)
	}
}

// TestUseLocationAliasesUseRoute proves UseLocation returns the same snapshot.
func TestUseLocationAliasesUseRoute(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	globalRouter = NewHashRouter()
	registerLocationProbeRoutes(globalRouter)
	Navigate("/alpha")
	globalRouter.renderCurrentRoute(false)

	parseFiber := &runtime.Fiber{}
	runtime.SetCurrentFiber(parseFiber)
	parseLoc := UseLocation()
	runtime.SetCurrentFiber(nil)
	if parseLoc.Path != "/alpha" {
		parseT.Fatalf("UseLocation returned %q, want /alpha", parseLoc.Path)
	}
}
