package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
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
