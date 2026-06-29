package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func TestRolloutViewFromBootstrapReadsTypedPayloads(parseT *testing.T) {
	parsePayload := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: rolloutBetaPath},
	}
	parseView := defaultRolloutView()
	if parseErr := ui.RegisterBootstrapPayload(&parsePayload, rolloutConfigPayloadKey, parseView.Config); parseErr != nil {
		parseT.Fatalf("unexpected config bootstrap registration error: %v", parseErr)
	}
	if parseErr2 := ui.RegisterBootstrapPayload(&parsePayload, rolloutFlagsPayloadKey, parseView.Flags, ui.SSRPayloadOptions{Revision: parseView.FlagRevision}); parseErr2 != nil {
		parseT.Fatalf("unexpected flag bootstrap registration error: %v", parseErr2)
	}

	parseDecoded := rolloutViewFromBootstrap(parsePayload)
	if parseDecoded.Config.APIBaseURL != parseView.Config.APIBaseURL {
		parseT.Fatalf("expected API base %q, got %q", parseView.Config.APIBaseURL, parseDecoded.Config.APIBaseURL)
	}
	if !parseDecoded.Flags.BetaRouteEnabled {
		parseT.Fatal("expected beta route flag to remain enabled")
	}
	if parseDecoded.FlagRevision != parseView.FlagRevision {
		parseT.Fatalf("expected flag revision %q, got %q", parseView.FlagRevision, parseDecoded.FlagRevision)
	}
}

func TestRenderRolloutPageIncludesEnvironmentAndGate(parseT *testing.T) {
	parseView := defaultRolloutView()
	parseOutput, parseErr := ui.RenderToString(renderRolloutPage(parseView, rolloutBetaPath))
	if parseErr != nil {
		parseT.Fatalf("unexpected rollout render error: %v", parseErr)
	}
	for _, parseSnippet := range []string{
		"Environment-aware staged rollout",
		parseView.Config.APIBaseURL,
		parseView.FlagRevision,
		"Feature-gated route is live",
	} {
		if !strings.Contains(parseOutput, parseSnippet) {
			parseT.Fatalf("expected rendered output to contain %q, got %q", parseSnippet, parseOutput)
		}
	}
}

// TestRolloutViewFromBootstrapCoversFallbackRoutes verifies route fallback and revision fallback behavior.
func TestRolloutViewFromBootstrapCoversFallbackRoutes(parseT *testing.T) {
	parseView := rolloutViewFromBootstrap(ui.SSRBootstrap{})
	if parseView.RoutePath != rolloutBetaPath {
		parseT.Fatalf("default rollout route = %q, want %q", parseView.RoutePath, rolloutBetaPath)
	}

	parsePayload := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: rolloutBetaPath},
	}
	parseFlags := rolloutFlags{
		BetaRouteEnabled: false,
		Cohort:           "control",
		EvaluationSource: "fallback",
		RolloutStage:     "off",
	}
	if parseErr := ui.RegisterBootstrapPayload(&parsePayload, rolloutFlagsPayloadKey, parseFlags, ui.SSRPayloadOptions{Revision: "   "}); parseErr != nil {
		parseT.Fatalf("unexpected flag payload registration error: %v", parseErr)
	}

	parseDecoded := rolloutViewFromBootstrap(parsePayload)
	if parseDecoded.RoutePath != rolloutControlPath {
		parseT.Fatalf("disabled beta route path = %q, want %q", parseDecoded.RoutePath, rolloutControlPath)
	}
	if parseDecoded.FlagRevision != defaultRolloutView().FlagRevision {
		parseT.Fatalf("blank revision should preserve default revision, got %q", parseDecoded.FlagRevision)
	}
}

// TestRolloutRoutePanelAndEnabledLabelCoverBranches verifies the control and unavailable route panels render their branch-specific content.
func TestRolloutRoutePanelAndEnabledLabelCoverBranches(parseT *testing.T) {
	if parseLabel := enabledLabel(true); parseLabel != "enabled" {
		parseT.Fatalf("enabledLabel(true) = %q, want enabled", parseLabel)
	}
	if parseLabel := enabledLabel(false); parseLabel != "disabled" {
		parseT.Fatalf("enabledLabel(false) = %q, want disabled", parseLabel)
	}

	parseView := defaultRolloutView()
	parseView.Flags.BetaRouteEnabled = false

	parseControlMarkup, parseErr := ui.RenderToString(rolloutRoutePanel(parseView, rolloutControlPath))
	if parseErr != nil {
		parseT.Fatalf("control rolloutRoutePanel render error = %v", parseErr)
	}
	for _, parseSnippet := range []string{
		"Always-on route reads the same snapshot",
		"disabled",
		parseView.Config.APIBaseURL,
		parseView.FlagRevision,
	} {
		if !strings.Contains(parseControlMarkup, parseSnippet) {
			parseT.Fatalf("control rolloutRoutePanel missing %q\n%s", parseSnippet, parseControlMarkup)
		}
	}

	parseUnavailableMarkup, parseErr := ui.RenderToString(rolloutRoutePanel(parseView, "/unknown"))
	if parseErr != nil {
		parseT.Fatalf("unavailable rolloutRoutePanel render error = %v", parseErr)
	}
	for _, parseSnippet := range []string{
		"Unavailable route",
		"Route is outside the active rollout",
	} {
		if !strings.Contains(parseUnavailableMarkup, parseSnippet) {
			parseT.Fatalf("unavailable rolloutRoutePanel missing %q\n%s", parseSnippet, parseUnavailableMarkup)
		}
	}
}
