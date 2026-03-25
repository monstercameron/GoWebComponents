package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestRolloutViewFromBootstrapReadsTypedPayloads(t *testing.T) {
	payload := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: rolloutBetaPath},
	}
	view := defaultRolloutView()
	if err := ui.RegisterBootstrapPayload(&payload, rolloutConfigPayloadKey, view.Config); err != nil {
		t.Fatalf("unexpected config bootstrap registration error: %v", err)
	}
	if err := ui.RegisterBootstrapPayload(&payload, rolloutFlagsPayloadKey, view.Flags, ui.SSRPayloadOptions{Revision: view.FlagRevision}); err != nil {
		t.Fatalf("unexpected flag bootstrap registration error: %v", err)
	}

	decoded := rolloutViewFromBootstrap(payload)
	if decoded.Config.APIBaseURL != view.Config.APIBaseURL {
		t.Fatalf("expected API base %q, got %q", view.Config.APIBaseURL, decoded.Config.APIBaseURL)
	}
	if !decoded.Flags.BetaRouteEnabled {
		t.Fatal("expected beta route flag to remain enabled")
	}
	if decoded.FlagRevision != view.FlagRevision {
		t.Fatalf("expected flag revision %q, got %q", view.FlagRevision, decoded.FlagRevision)
	}
}

func TestRenderRolloutPageIncludesEnvironmentAndGate(t *testing.T) {
	view := defaultRolloutView()
	output, err := ui.RenderToString(renderRolloutPage(view, rolloutBetaPath))
	if err != nil {
		t.Fatalf("unexpected rollout render error: %v", err)
	}
	for _, snippet := range []string{
		"Environment-aware staged rollout",
		view.Config.APIBaseURL,
		view.FlagRevision,
		"Feature-gated route is live",
	} {
		if !strings.Contains(output, snippet) {
			t.Fatalf("expected rendered output to contain %q, got %q", snippet, output)
		}
	}
}
