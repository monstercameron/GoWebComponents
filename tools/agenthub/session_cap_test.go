package agenthub

import (
	"strconv"
	"testing"
)

// TestSessionHistoryIsBounded pins that the append-only session history is capped:
// past maxRetainedSessions the oldest sessions are dropped, keeping the newest.
// Crashed/reloaded predecessors were previously retained forever for crash-report
// linkage, so a long dev session with repeated hot-reloads grew the slice (and its
// per-session ring buffers) without bound.
func TestSessionHistoryIsBounded(parseT *testing.T) {
	parseHub, parseErr := NewAgentHub()
	if parseErr != nil {
		parseT.Fatalf("NewAgentHub: %v", parseErr)
	}
	parsePrevCap := maxRetainedSessions
	maxRetainedSessions = 4
	defer func() { maxRetainedSessions = parsePrevCap }()

	const parseTotal = 20
	parseHub.mu.Lock()
	for parseI := 0; parseI < parseTotal; parseI++ {
		parseHub.appendSessionLocked(&Session{ID: "sess-" + strconv.Itoa(parseI)})
	}
	parseCount := len(parseHub.sessions)
	parseFirst := parseHub.sessions[0].ID
	parseLast := parseHub.sessions[len(parseHub.sessions)-1].ID
	parseHub.mu.Unlock()

	if parseCount != 4 {
		parseT.Fatalf("session history not capped: got %d, want 4", parseCount)
	}
	// The newest sessions are retained (16..19); the oldest are pruned.
	if parseFirst != "sess-16" {
		parseT.Fatalf("expected oldest retained session sess-16, got %q", parseFirst)
	}
	if parseLast != "sess-19" {
		parseT.Fatalf("expected newest session sess-19, got %q", parseLast)
	}
	// A pruned predecessor id no longer resolves (chain walk stops gracefully).
	if parseHub.findSession("sess-0") != nil {
		parseT.Fatal("pruned session sess-0 should not be findable")
	}
	if parseHub.findSession("sess-19") == nil {
		parseT.Fatal("newest session sess-19 must still be findable")
	}
}
