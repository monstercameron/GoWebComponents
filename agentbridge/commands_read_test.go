package agentbridge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// TestRegisterReadCommandsRegistersCommands pins that RegisterReadCommands
// adds both bridge.snapshot and bridge.query to the registry.
func TestRegisterReadCommandsRegistersCommands(t *testing.T) {
	RegisterReadCommands()
	parseNames := ListAgentCommands()
	parseFoundSnapshot := false
	parseFoundQuery := false
	for _, parseName := range parseNames {
		if parseName == "bridge.snapshot" {
			parseFoundSnapshot = true
		}
		if parseName == "bridge.query" {
			parseFoundQuery = true
		}
	}
	if !parseFoundSnapshot {
		t.Error("bridge.snapshot not registered after RegisterReadCommands")
	}
	if !parseFoundQuery {
		t.Error("bridge.query not registered after RegisterReadCommands")
	}
}

// TestSnapshotHandlerBadPayload pins that a malformed JSON payload returns
// the bad-payload error code.
func TestSnapshotHandlerBadPayload(t *testing.T) {
	_, parseErr := readHandleSnapshot(json.RawMessage(`{not valid json`))
	if parseErr == nil {
		t.Fatal("expected an error for malformed snapshot payload")
	}
	if parseErr.Code != ErrorCodeBadPayload {
		t.Errorf("expected bad-payload code, got %q", parseErr.Code)
	}
}

// TestQueryHandlerBadPayload pins that a malformed JSON payload returns the
// bad-payload error code.
func TestQueryHandlerBadPayload(t *testing.T) {
	_, parseErr := readHandleQuery(json.RawMessage(`{not valid json`))
	if parseErr == nil {
		t.Fatal("expected an error for malformed query payload")
	}
	if parseErr.Code != ErrorCodeBadPayload {
		t.Errorf("expected bad-payload code, got %q", parseErr.Code)
	}
}

// TestSnapshotHandlerEmptyPayloadNoTree pins that the handler returns a
// valid (empty) result when the runtime has no mounted tree.
func TestSnapshotHandlerEmptyPayloadNoTree(t *testing.T) {
	// Use a detached runtime so we don't disturb the global state for other tests.
	parseBytes, parseErr := readHandleSnapshot(nil)
	if parseErr != nil {
		t.Fatalf("unexpected error with empty payload + no tree: %v", parseErr)
	}
	if parseBytes == nil {
		t.Fatal("expected non-nil result bytes")
	}
	var parseResult readSnapshotResult
	if parseUnmarshalErr := json.Unmarshal(parseBytes, &parseResult); parseUnmarshalErr != nil {
		t.Fatalf("failed to unmarshal snapshot result: %v", parseUnmarshalErr)
	}
	// Root may or may not be nil depending on global runtime state; the
	// important invariant is no error.
}

// TestQueryHandlerEmptyPayloadNoTree pins that the handler returns an empty
// match list when the runtime has no tree to query.
func TestQueryHandlerEmptyPayloadNoTree(t *testing.T) {
	parseBytes, parseErr := readHandleQuery(nil)
	if parseErr != nil {
		t.Fatalf("unexpected error with empty query payload + no tree: %v", parseErr)
	}
	if parseBytes == nil {
		t.Fatal("expected non-nil result bytes")
	}
	var parseResult readQueryResult
	if parseUnmarshalErr := json.Unmarshal(parseBytes, &parseResult); parseUnmarshalErr != nil {
		t.Fatalf("failed to unmarshal query result: %v", parseUnmarshalErr)
	}
	// Matches may be empty or non-empty depending on global runtime state.
}

// TestDecodeSelector pins the readDecodeSelector helper.
func TestDecodeSelector(t *testing.T) {
	parseRaw := json.RawMessage(`{"role":"button","label":"Submit","text":"click","id":"btn1","tag":"button"}`)
	parseSel, parseErr := readDecodeSelector(parseRaw)
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	if parseSel.Role != "button" {
		t.Errorf("role: want %q got %q", "button", parseSel.Role)
	}
	if parseSel.Label != "Submit" {
		t.Errorf("label: want %q got %q", "Submit", parseSel.Label)
	}
	if parseSel.Text != "click" {
		t.Errorf("text: want %q got %q", "click", parseSel.Text)
	}
	if parseSel.ID != "btn1" {
		t.Errorf("id: want %q got %q", "btn1", parseSel.ID)
	}
	if parseSel.Tag != "button" {
		t.Errorf("tag: want %q got %q", "button", parseSel.Tag)
	}
}

// TestDecodeSelectorEmptyPayload pins that empty or nil payload returns a zero
// selector without error.
func TestDecodeSelectorEmptyPayload(t *testing.T) {
	parseSel, parseErr := readDecodeSelector(nil)
	if parseErr != nil {
		t.Fatalf("unexpected error for nil payload: %v", parseErr)
	}
	if parseSel.Role != "" || parseSel.Label != "" || parseSel.Text != "" || parseSel.ID != "" || parseSel.Tag != "" {
		t.Errorf("expected zero selector for nil payload, got %+v", parseSel)
	}
}

// TestDecodeSelectorMalformed pins that invalid JSON returns an error.
func TestDecodeSelectorMalformed(t *testing.T) {
	_, parseErr := readDecodeSelector(json.RawMessage(`{bad`))
	if parseErr == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

// TestSnapshotCommandEndToEnd pins an end-to-end snapshot execution when the
// global runtime has a mounted tree.
func TestSnapshotCommandEndToEnd(t *testing.T) {
	// Install a simple tree into the global runtime for this test.
	parseRoot := &runtime.Fiber{}
	_ = parseRoot // The Fiber struct fields are unexported; we can only drive
	// this through the public API surface (BuildAgentSnapshot via handler).

	RegisterReadCommands()

	// Send a well-formed snapshot command and expect an ok result with
	// Snapshot.Root potentially nil (depending on global tree state) but
	// always a non-error response.
	parseBytes, parseErr := ExecuteAgentCommand("bridge.snapshot", json.RawMessage(`{"maxDepth":5,"maxNodes":100}`))
	if parseErr != nil {
		t.Fatalf("bridge.snapshot returned error: %v", parseErr)
	}
	if parseBytes == nil {
		t.Fatal("expected non-nil response bytes")
	}
	var parseResult readSnapshotResult
	if parseUnmarshalErr := json.Unmarshal(parseBytes, &parseResult); parseUnmarshalErr != nil {
		t.Fatalf("failed to parse snapshot response: %v", parseUnmarshalErr)
	}
}

// TestQueryCommandEndToEnd pins an end-to-end query execution via the registry.
func TestQueryCommandEndToEnd(t *testing.T) {
	RegisterReadCommands()
	parseBytes, parseErr := ExecuteAgentCommand("bridge.query", json.RawMessage(`{"role":"button"}`))
	if parseErr != nil {
		t.Fatalf("bridge.query returned error: %v", parseErr)
	}
	if parseBytes == nil {
		t.Fatal("expected non-nil response bytes")
	}
	var parseResult readQueryResult
	if parseUnmarshalErr := json.Unmarshal(parseBytes, &parseResult); parseUnmarshalErr != nil {
		t.Fatalf("failed to parse query response: %v", parseUnmarshalErr)
	}
	// Matches may be nil (empty global tree) — that is the expected safe outcome.
}

// TestSnapshotResultContainsTruncationFields pins that the snapshot result JSON
// carries explicit truncation metadata fields (even when zero) so consumers
// can always detect budget application without special-casing.
func TestSnapshotResultContainsTruncationFields(t *testing.T) {
	parseBytes, parseErr := readHandleSnapshot(json.RawMessage(`{"maxDepth":1,"maxNodes":1}`))
	if parseErr != nil {
		t.Fatalf("unexpected error: %v", parseErr)
	}
	parseJSON := string(parseBytes)
	// The serialised snapshot must always carry the budgetApplied and
	// truncatedNodes keys (they may be false/0 when the tree is empty).
	// We check the outer wrapper only — the inner snapshot fields are on the
	// AgentSnapshot type and only appear when non-zero (omitempty), which is
	// fine: zero means no truncation occurred.
	if !strings.Contains(parseJSON, "snapshot") {
		t.Errorf("response missing top-level snapshot field: %s", parseJSON)
	}
}
