//go:build js && wasm

package router

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// routeLocationAtomID is the well-known atom the router publishes the active
// Location into on every navigation. UseRoute subscribes to it. (A colon is fine
// here — this is an atom id / event key, never a CSS element id.)
const routeLocationAtomID = "gwc:router:location"

// routeDataRevision counts loader completions; it is folded into the published
// Location so UseRouteData subscribers see data that arrives after navigation.
var routeDataRevision int

// routeFactoryDepth is above zero while a route factory runs. Factories execute
// inside the router's own fiber, which already re-renders on navigation, so
// they must not add subscription hooks (that would make the router fiber's hook
// order depend on the matched route).
var routeFactoryDepth int

// lastPublishedLocationKey dedupes publishes so an unchanged location does not
// schedule redundant re-renders.
var lastPublishedLocationKey string

// currentLocationSnapshot builds a Location from the router's current globals.
func currentLocationSnapshot() Location {
	return Location{
		Path:   GetCurrentPath(),
		Query:  getCurrentQueryValues().Encode(),
		Params: copyParams(currentParams),

		revision: routeDataRevision,
	}
}

// subscribeRouteLocation subscribes the calling component to navigation so the
// non-Location hooks (UseParams, UseQuery, UseSearchParams, UseRouteData) re-render
// memoized components like UseRoute does. It is a no-op outside a component render
// (event handlers, guards, tests) and inside route factories.
func subscribeRouteLocation() {
	if routeFactoryDepth > 0 || runtime.GetCurrentFiber() == nil {
		return
	}
	UseRoute()
}

// UseRoute subscribes the calling component to navigation and returns the active
// Location. When the route changes, every component that called UseRoute
// re-renders — including memoized chrome — so the active-nav highlight and
// breadcrumb update without prop-drilling the path (G6).
//
//	func Sidebar(_ SidebarProps) ui.Node {
//	    loc := router.UseRoute()
//	    // highlight the link whose href == loc.Path
//	}
func UseRoute() Location {
	parseGet, _ := runtime.GoUseAtomGlobal(routeLocationAtomID, currentLocationSnapshot())
	return parseGet()
}

// UseLocation is an alias for UseRoute, matching the naming many router users
// expect.
func UseLocation() Location {
	return UseRoute()
}

// publishLocation writes the current Location into the route atom, notifying
// subscribed components, and fires registered OnNavigate callbacks. It is called
// from renderCurrentRoute on every navigation, and dedupes on path+query so an
// unchanged location is a no-op (callbacks fire only on a real change).
func publishLocation() {
	parseRt := runtime.GetGlobalRuntime()
	if parseRt == nil {
		return
	}
	parseLocation := currentLocationSnapshot()
	parseKey := parseLocation.Path + "?" + parseLocation.Query + "#" + strconv.Itoa(parseLocation.revision)
	if parseKey == lastPublishedLocationKey {
		return
	}
	lastPublishedLocationKey = parseKey
	_ = parseRt.SetAtomValue(routeLocationAtomID, parseLocation)
	fireNavigationCallbacks(parseLocation)
}

// navigationCallbacks holds OnNavigate subscribers keyed by a monotonic id so
// unsubscribe is O(1) and order-stable.
var (
	navigationCallbacks   = map[int]func(Location){}
	navigationCallbackSeq int
)

// OnNavigate registers fn to run after each navigation with the new Location —
// the seam for SPA scroll-reset, analytics page views, and title side effects
// (G28). It returns an unsubscribe func.
//
// Firing semantics: fn runs once on the initial render (the first location is a
// change from "none", so analytics see the landing page view), then once per
// subsequent navigation. It does NOT fire when the location is unchanged (a
// re-render at the same path+query is deduped). Register before mounting if you
// need the initial fire.
//
//	stop := router.OnNavigate(func(loc router.Location) { scrollTop() })
//	defer stop()
func OnNavigate(parseFn func(Location)) func() {
	if parseFn == nil {
		return func() {}
	}
	navigationCallbackSeq++
	parseID := navigationCallbackSeq
	navigationCallbacks[parseID] = parseFn
	return func() { delete(navigationCallbacks, parseID) }
}

func fireNavigationCallbacks(parseLocation Location) {
	for _, parseFn := range navigationCallbacks {
		if parseFn != nil {
			parseFn(parseLocation)
		}
	}
}

// Href returns the correct href attribute value for navigating to path under the
// active router mode (G7). For a hash router it returns "#<path>"; for a history
// router it returns the normalized path (which the browser resolves against any
// <base href>). Use it instead of hand-building href strings so links work
// regardless of router mode or base href.
//
//	A(Href(router.Href("/transactions")), "Transactions")
func Href(parsePath string) string {
	parseNormalized := normalizePath(parsePath)
	if parseRouter := globalRouter; parseRouter != nil {
		if parseRouter.routerType == routerTypeHash {
			return "#" + parseNormalized
		}
		return parseRouter.addBasePath(parseNormalized)
	}
	return parseNormalized
}

// FragmentHref returns an in-page anchor href (e.g. a "skip to content" link)
// that is safe under a <base href> (G7). With a <base href> a bare "#main"
// resolves against the base (navigating to root); embedding the live path keeps
// the anchor in-page. It returns "<current-path>#<fragment>".
//
// This targets history routers, where the <base href> footgun exists. Hash
// routers keep the whole route in the URL fragment, so a second in-page fragment
// cannot be expressed as an href (use programmatic scrolling instead); to avoid a
// malformed "#path#fragment", FragmentHref does NOT add the hash-router prefix.
//
// The current query string is preserved (e.g. "/list?page=2#main"), so clicking
// the anchor stays on the exact current location rather than dropping query state.
//
//	A(html.Href(router.FragmentHref("main")), "Skip to content")
func FragmentHref(parseFragment string) string {
	parseFragment = strings.TrimPrefix(parseFragment, "#")
	parsePath := normalizePath(GetCurrentPath())
	if parseRouter := globalRouter; parseRouter != nil {
		parsePath = parseRouter.addBasePath(parsePath)
	}
	if parseQuery := getCurrentQueryValues().Encode(); parseQuery != "" {
		return parsePath + "?" + parseQuery + "#" + parseFragment
	}
	return parsePath + "#" + parseFragment
}
