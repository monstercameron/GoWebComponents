//go:build js && wasm
// +build js,wasm

package ssr

import (
	"strings"
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

func TestRoundTripHydrateSeedsServerMarkupAndReusesRootNode(t *testing.T) {
	harness := RoundTripHydrate(t,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
	)

	if !strings.Contains(harness.Markup, `id="hydrated-root"`) {
		t.Fatalf("expected server markup to be captured, got %q", harness.Markup)
	}
	if got := harness.Seeded.NodeIDs["hydrated-root"]; got == 0 {
		t.Fatalf("expected seeded node id for hydrated root, got %+v", harness.Seeded.NodeIDs)
	} else if node := harness.ByID("hydrated-root"); node == nil || node.NodeID() != got {
		t.Fatalf("expected hydrated root node to be reused, seeded=%d node=%#v", got, node)
	}
}
