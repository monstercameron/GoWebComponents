//go:build js && wasm

package ssr

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestSmokeHydrateRendersIntoFixture(parseT *testing.T) {
	parseBootstrap := ui.SSRBootstrap{Route: ui.SSRRouteBootstrap{Path: "/products"}}
	parseHarness := SmokeHydrate(parseT,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
		HydrationOptions{Bootstrap: parseBootstrap},
	)

	if parseGot := parseHarness.Bootstrap.Route.Path; parseGot != "/products" {
		parseT.Fatalf("expected bootstrap route path to round-trip, got %q", parseGot)
	}
	if parseNode := parseHarness.ByID("hydrated-root"); parseNode == nil || parseNode.Text() != "Hydrated" {
		parseT.Fatalf("expected hydrated fixture content, got %#v", parseNode)
	}
}

func TestRoundTripHydrateSeedsServerMarkupAndReusesRootNode(parseT *testing.T) {
	parseHarness := RoundTripHydrate(parseT,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
	)

	if !strings.Contains(parseHarness.Markup, `id="hydrated-root"`) {
		parseT.Fatalf("expected server markup to be captured, got %q", parseHarness.Markup)
	}
	if parseGot := parseHarness.Seeded.NodeIDs["hydrated-root"]; parseGot == 0 {
		parseT.Fatalf("expected seeded node id for hydrated root, got %+v", parseHarness.Seeded.NodeIDs)
	} else if parseNode := parseHarness.ByID("hydrated-root"); parseNode == nil || parseNode.NodeID() != parseGot {
		parseT.Fatalf("expected hydrated root node to be reused, seeded=%d node=%#v", parseGot, parseNode)
	}
}

func TestRoundTripHydrateMismatchUsesMutatedMarkup(parseT *testing.T) {
	parseHarness := RoundTripHydrateMismatch(parseT,
		html.Div(html.Props{ID: "hydrated-root"}, html.Text("Hydrated")),
		func(parseMarkup string) string {
			return strings.Replace(parseMarkup, "Hydrated", "ServerDrift", 1)
		},
	)

	if !strings.Contains(parseHarness.Markup, "ServerDrift") {
		parseT.Fatalf("expected mismatch helper to preserve mutated server markup, got %q", parseHarness.Markup)
	}
	if parseNode := parseHarness.ByID("hydrated-root"); parseNode == nil || parseNode.Text() != "Hydrated" {
		parseT.Fatalf("expected client render to recover hydrated text, got %#v", parseNode)
	}
}
