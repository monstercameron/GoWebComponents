//go:build js && wasm
// +build js,wasm

package ssr_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	ssr "github.com/monstercameron/GoWebComponents/test/ssr"
	base "github.com/monstercameron/GoWebComponents/testkit/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestPreferredHydrationWrappersMatchCompatibilityAliasBehavior(parseT *testing.T) {
	parseBootstrap := ui.SSRBootstrap{Route: ui.SSRRouteBootstrap{Path: "/products"}}

	parsePreferred := ssr.SmokeHydrate(parseT,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
		ssr.HydrationOptions{Bootstrap: parseBootstrap},
	)
	parsePreferredPath := parsePreferred.Bootstrap.Route.Path
	parsePreferredText := parsePreferred.ByID("hydrated-root").Text()
	parsePreferred.Cleanup()

	parseCompat := base.SmokeHydrate(parseT,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
		base.HydrationOptions{Bootstrap: parseBootstrap},
	)
	parseCompatPath := parseCompat.Bootstrap.Route.Path
	parseCompatText := parseCompat.ByID("hydrated-root").Text()
	parseCompat.Cleanup()

	if parsePreferredPath != parseCompatPath || parsePreferredText != parseCompatText {
		parseT.Fatalf("expected preferred wrapper and compatibility alias hydration to match, got preferred=(%q,%q) compat=(%q,%q)", parsePreferredPath, parsePreferredText, parseCompatPath, parseCompatText)
	}
}
