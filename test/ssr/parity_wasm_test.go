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

func TestPreferredHydrationWrappersMatchCompatibilityAliasBehavior(t *testing.T) {
	bootstrap := ui.SSRBootstrap{Route: ui.SSRRouteBootstrap{Path: "/products"}}

	preferred := ssr.SmokeHydrate(t,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
		ssr.HydrationOptions{Bootstrap: bootstrap},
	)
	preferredPath := preferred.Bootstrap.Route.Path
	preferredText := preferred.ByID("hydrated-root").Text()
	preferred.Cleanup()

	compat := base.SmokeHydrate(t,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
		base.HydrationOptions{Bootstrap: bootstrap},
	)
	compatPath := compat.Bootstrap.Route.Path
	compatText := compat.ByID("hydrated-root").Text()
	compat.Cleanup()

	if preferredPath != compatPath || preferredText != compatText {
		t.Fatalf("expected preferred wrapper and compatibility alias hydration to match, got preferred=(%q,%q) compat=(%q,%q)", preferredPath, preferredText, compatPath, compatText)
	}
}
