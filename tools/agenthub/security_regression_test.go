package agenthub

import "testing"

// TestUndoAndReplayAreLeaseGated pins that bridge.undo and bridge.replay are
// classified as mutating, so the hub's write-lease check applies to them
// (regression: bridge.undo bypassed the lease and could reverse another
// holder's mutations via the REST API).
func TestUndoAndReplayAreLeaseGated(t *testing.T) {
	for _, parseName := range []string{"bridge.undo", "bridge.replay"} {
		if !isMutatingBridgeCommand(parseName) {
			t.Fatalf("%s must be lease-gated (isMutatingBridgeCommand=false)", parseName)
		}
	}
	// Read-only commands must remain ungated.
	for _, parseName := range []string{"bridge.snapshot", "bridge.query", "bridge.describe", "bridge.audit", "bridge.wait-for"} {
		if isMutatingBridgeCommand(parseName) {
			t.Fatalf("%s must NOT be lease-gated", parseName)
		}
	}
}

// TestTokenMatchesConstantTime pins the token comparison helper accepts the
// exact token and rejects empty/wrong tokens (the comparison itself is
// constant-time via crypto/subtle).
func TestTokenMatchesConstantTime(t *testing.T) {
	parseHub, parseErr := NewAgentHub()
	if parseErr != nil {
		t.Fatalf("new hub: %v", parseErr)
	}
	if !parseHub.tokenMatches(parseHub.Token()) {
		t.Fatalf("correct token rejected")
	}
	if parseHub.tokenMatches("") {
		t.Fatalf("empty token accepted")
	}
	if parseHub.tokenMatches(parseHub.Token() + "x") {
		t.Fatalf("wrong token accepted")
	}
}
