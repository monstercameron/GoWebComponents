package ui

import (
	"strings"
	"testing"
)

func TestHydrationIslandNormalizesTriggerOptions(parseT *testing.T) {
	parseGot, parseErr := NormalizeHydrationIslandOptions(HydrationIslandOptions{
		ID:       "hero",
		Strategy: HydrateOnInteraction,
		Events:   []string{"click", " keydown ", "click", ""},
	})
	if parseErr != nil {
		parseT.Fatalf("NormalizeHydrationIslandOptions returned error: %v", parseErr)
	}
	if parseGot.Selector != `[data-gwc-hydration-island="hero"]` {
		parseT.Fatalf("selector = %q", parseGot.Selector)
	}
	if len(parseGot.Events) != 2 || parseGot.Events[0] != "click" || parseGot.Events[1] != "keydown" {
		parseT.Fatalf("events = %#v", parseGot.Events)
	}

	parseDefault, parseErr2 := NormalizeHydrationIslandOptions(HydrationIslandOptions{Selector: "#island"})
	if parseErr2 != nil {
		parseT.Fatalf("NormalizeHydrationIslandOptions returned error: %v", parseErr2)
	}
	if parseDefault.Strategy != HydrateOnVisible {
		parseT.Fatalf("default strategy = %q", parseDefault.Strategy)
	}
}

func TestHydrationIslandBudgetReport(parseT *testing.T) {
	parseReport := InspectHydrationIslandBudget(HydrationIslandPlan{
		Islands: []HydrationIslandOptions{
			{ID: "above-fold", Strategy: HydrateImmediately},
			{ID: "chart", Strategy: HydrateOnVisible},
			{ID: "drawer", Strategy: HydrateOnInteraction},
			{ID: "drawer", Strategy: HydrateOnIdle},
		},
		Budget: HydrationIslandBudget{MaxInitial: 1, MaxConcurrent: 2},
	})
	if parseReport.OK() {
		parseT.Fatalf("expected duplicate/concurrency violations, got %#v", parseReport)
	}
	if parseReport.Initial != 1 || parseReport.Deferred != 3 {
		parseT.Fatalf("counts = initial %d deferred %d", parseReport.Initial, parseReport.Deferred)
	}
	parseJoined := strings.Join(parseReport.Violations, "\n")
	if !strings.Contains(parseJoined, `island "drawer" is duplicated`) {
		parseT.Fatalf("missing duplicate violation in %#v", parseReport.Violations)
	}
	if !strings.Contains(parseJoined, "deferred hydration islands 3 exceed concurrency budget 2") {
		parseT.Fatalf("missing budget violation in %#v", parseReport.Violations)
	}
}

func TestHydrationIslandRendersStableSSRMarker(parseT *testing.T) {
	parseNode := HydrationIsland(HydrationIslandOptions{
		ID:         "pricing",
		Strategy:   HydrateOnInteraction,
		Events:     []string{"pointerenter", "click"},
		RootMargin: "200px",
	}, Text("static pricing"))
	parseMarkup, parseErr := RenderToString(parseNode)
	if parseErr != nil {
		parseT.Fatalf("RenderToString returned error: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `data-gwc-hydration-island="pricing"`) ||
		!strings.Contains(parseMarkup, `data-gwc-hydration="interaction"`) ||
		!strings.Contains(parseMarkup, `data-gwc-hydration-events="pointerenter click"`) ||
		!strings.Contains(parseMarkup, `static pricing`) {
		parseT.Fatalf("unexpected island markup: %s", parseMarkup)
	}
}
