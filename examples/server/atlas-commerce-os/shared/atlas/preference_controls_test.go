package atlas

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestSettingsRouteRendersPreferenceFields verifies settings route preference controls render with core labels and submit action.
func TestSettingsRouteRendersPreferenceFields(parseT *testing.T) {
	parseT.Parallel()

	parsePayload := samplePayloadForRoute(RouteSettings, settingsPage{
		Summary: sampleSummary("Shell settings"),
	}, nil)
	// component + props, not App(payload): App's hooks need a render fiber.
	parseMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(App, parsePayload))
	if parseErr != nil {
		parseT.Fatalf("settings route render failed: %v", parseErr)
	}
	for _, parseExpected := range []string{
		"Operator preferences",
		"Theme",
		"Locale",
		"Density",
		"Default warehouse",
		"Save preferences",
	} {
		if !strings.Contains(parseMarkup, parseExpected) {
			parseT.Fatalf("expected %q in settings preference markup, got %q", parseExpected, parseMarkup)
		}
	}
}

// TestSettingsRouteRendersSavedViewControls verifies saved-view listbox controls render with keyboard guidance and active view details.
func TestSettingsRouteRendersSavedViewControls(parseT *testing.T) {
	parseT.Parallel()

	parsePayload := samplePayloadForRoute(RouteSettings, settingsPage{
		Summary: sampleSummary("Shell settings"),
	}, nil)
	// component + props, not App(payload): App's hooks need a render fiber.
	parseMarkup, parseErr := renderAtlasNodeForTest(ui.CreateElement(App, parsePayload))
	if parseErr != nil {
		parseT.Fatalf("settings route render failed: %v", parseErr)
	}
	if !strings.Contains(parseMarkup, `role="listbox"`) {
		parseT.Fatalf("expected saved-view listbox control, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "ArrowUp, ArrowDown, Home, End, or typeahead") {
		parseT.Fatalf("expected saved-view keyboard guidance, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Low stock triage") {
		parseT.Fatalf("expected seeded saved view label in settings, got %q", parseMarkup)
	}
	if !strings.Contains(parseMarkup, "Sort key") || !strings.Contains(parseMarkup, "Direction") {
		parseT.Fatalf("expected saved-view detail cards in settings, got %q", parseMarkup)
	}
}
