package agentbridge

import (
	"encoding/json"
	"testing"
)

// TestReplayLifecycle pins the start → status → stop → replay flow: status
// reflects recording state, stop returns a capture count, and replay reports
// the same count replayed. Exact captured counts depend on runtime scheduling,
// so the test asserts the state machine and field plumbing, not a fixed count.
func TestReplayLifecycle(t *testing.T) {
	resetReplayState()
	writeActivateAgentMode(t)

	// status before anything: not recording, nothing captured.
	parseStatus := replayCall(t, `{"action":"status"}`)
	if parseStatus.Recording || parseStatus.Captured != 0 {
		t.Fatalf("initial status wrong: %+v", parseStatus)
	}

	// start: recording becomes true.
	parseStart := replayCall(t, `{"action":"start"}`)
	if !parseStart.Recording {
		t.Fatalf("expected recording after start: %+v", parseStart)
	}

	// stop: recording false, captured count is non-negative and consistent.
	parseStop := replayCall(t, `{"action":"stop"}`)
	if parseStop.Recording {
		t.Fatalf("expected not recording after stop: %+v", parseStop)
	}

	// replay: replayed equals captured.
	parseReplay := replayCall(t, `{"action":"replay"}`)
	if parseReplay.Replayed != parseReplay.Captured {
		t.Fatalf("replayed (%d) != captured (%d)", parseReplay.Replayed, parseReplay.Captured)
	}
}

// TestReplayDoubleStartRejected pins that starting a second recording while one
// is active is refused (regression: it used to silently reset the recorder and
// drop the in-progress capture window).
func TestReplayDoubleStartRejected(t *testing.T) {
	resetReplayState()
	writeActivateAgentMode(t)
	if _, parseErr := replayHandle(json.RawMessage(`{"action":"start"}`)); parseErr != nil {
		t.Fatalf("first start: %s", parseErr.Message)
	}
	_, parseErr := replayHandle(json.RawMessage(`{"action":"start"}`))
	if parseErr == nil || parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("expected bad-payload on double start, got %+v", parseErr)
	}
	// Clean up the recorder.
	_, _ = replayHandle(json.RawMessage(`{"action":"stop"}`))
}

// TestReplayActionIsAudited pins that a replay re-applying updates to live
// state records an audit entry (closes the audit blind spot).
func TestReplayActionIsAudited(t *testing.T) {
	resetReplayState()
	resetAgentAudit()
	writeActivateAgentMode(t)
	if _, parseErr := replayHandle(json.RawMessage(`{"action":"start"}`)); parseErr != nil {
		t.Fatalf("start: %s", parseErr.Message)
	}
	if _, parseErr := replayHandle(json.RawMessage(`{"action":"stop"}`)); parseErr != nil {
		t.Fatalf("stop: %s", parseErr.Message)
	}
	if _, parseErr := replayHandle(json.RawMessage(`{"action":"replay"}`)); parseErr != nil {
		t.Fatalf("replay: %s", parseErr.Message)
	}
	parseEntries := AgentAuditEntries(0)
	if len(parseEntries) != 1 || parseEntries[0].Command != "bridge.replay" {
		t.Fatalf("replay not audited: %+v", parseEntries)
	}
}

// TestReplayUnknownActionRejected pins a structured rejection for a bad action.
func TestReplayUnknownActionRejected(t *testing.T) {
	resetReplayState()
	writeActivateAgentMode(t)
	_, parseErr := replayHandle(json.RawMessage(`{"action":"explode"}`))
	if parseErr == nil || parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("expected bad-payload for unknown action, got %+v", parseErr)
	}
}

// TestReplayStopForbiddenWhenInactive pins that stop (a recorder mutation) is
// agent-mode gated, consistent with start/replay; status stays a free read.
func TestReplayStopForbiddenWhenInactive(t *testing.T) {
	resetReplayState()
	SetAgentModeActive(false)
	if _, parseErr := replayHandle(json.RawMessage(`{"action":"stop"}`)); parseErr == nil || parseErr.Code != ErrorCodeForbidden {
		t.Fatalf("expected forbidden for stop while inactive, got %+v", parseErr)
	}
	// status remains a free read.
	if _, parseErr := replayHandle(json.RawMessage(`{"action":"status"}`)); parseErr != nil {
		t.Fatalf("status should be readable without agent mode: %s", parseErr.Message)
	}
}

// TestReplayStartForbiddenWhenInactive pins that start refuses without agent mode.
func TestReplayStartForbiddenWhenInactive(t *testing.T) {
	resetReplayState()
	SetAgentModeActive(false)
	_, parseErr := replayHandle(json.RawMessage(`{"action":"start"}`))
	if parseErr == nil || parseErr.Code != ErrorCodeForbidden {
		t.Fatalf("expected forbidden, got %+v", parseErr)
	}
}

// replayCall invokes the replay handler and decodes the result.
func replayCall(parseTB testing.TB, parsePayload string) replayResult {
	parseTB.Helper()
	parseRaw, parseErr := replayHandle(json.RawMessage(parsePayload))
	if parseErr != nil {
		parseTB.Fatalf("replay(%s): %s", parsePayload, parseErr.Message)
	}
	var parseResult replayResult
	if parseDecErr := json.Unmarshal(parseRaw, &parseResult); parseDecErr != nil {
		parseTB.Fatalf("decode replay result: %v", parseDecErr)
	}
	return parseResult
}
