//go:build js && wasm

package router

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// TestDefaultRouteFallbackKeepsLayout pins #82 finding (4): an unmatched path that
// falls back to the default route must be wrapped in the same layout stack it gets
// when navigated to directly, not rendered bare. Previously resolveRouteStack
// returned only the default leaf, skipping every parent Layout route.
func TestDefaultRouteFallbackKeepsLayout(parseT *testing.T) {
	installRouterBrowserEnv(parseT)

	parseRouter := NewHistoryRouter(RouterOptions{DefaultRoute: "/home"})
	parseT.Cleanup(func() { parseRouter.teardownHistoryListener() })

	parseComponent := func(parseAttrs Attrs) *Element {
		return runtime.Div(nil, runtime.Text("x"))
	}
	// A root layout route + the default route target.
	parseRouter.Register("/", parseComponent, Options{Layout: true})
	parseRouter.Register("/home", parseComponent)

	// Sanity: navigating to the default route directly includes the layout.
	parseDirect := parseRouter.resolveRouteStack("/home")
	if !parseDirect.found || !stackHasLayout(parseDirect) {
		parseT.Fatalf("direct default-route resolution should include the layout, got %+v", parseDirect)
	}

	// The fix: an UNMATCHED path falling back to the default route keeps that layout.
	parseFallback := parseRouter.resolveRouteStack("/does-not-exist")
	if !parseFallback.found {
		parseT.Fatal("unmatched path should resolve to the default route")
	}
	if !stackHasLayout(parseFallback) {
		parseT.Fatalf("default-route fallback must keep its layout stack, got %d route(s) with no layout", len(parseFallback.routes))
	}
	// The leaf is still the default route.
	parseLeaf := parseFallback.routes[len(parseFallback.routes)-1]
	if parseLeaf.path != "/home" {
		parseT.Fatalf("fallback leaf must be the default route, got %q", parseLeaf.path)
	}
}

func stackHasLayout(parseStack resolvedRouteStack) bool {
	for _, parseRoute := range parseStack.routes {
		if parseRoute.option.Layout {
			return true
		}
	}
	return false
}
