//go:build js && wasm
// +build js,wasm

package ssr

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestSmokeHydrateRendersIntoFixture(t *testing.T) {
	bootstrap := ui.SSRBootstrap{Route: ui.SSRRouteBootstrap{Path: "/products"}}
	harness := SmokeHydrate(t,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
		HydrationOptions{Bootstrap: bootstrap},
	)

	if got := harness.Bootstrap.Route.Path; got != "/products" {
		t.Fatalf("expected bootstrap route path to round-trip, got %q", got)
	}
	if node := harness.ByID("hydrated-root"); node == nil || node.Text() != "Hydrated" {
		t.Fatalf("expected hydrated fixture content, got %#v", node)
	}
}
