package agentbridge

import (
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/events"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v5/state"
)

// TestSetAtomUndoRestoresPriorValue pins that a set-atom is reversible: after
// bridge.undo the atom holds exactly its pre-mutation value.
func TestSetAtomUndoRestoresPriorValue(t *testing.T) {
	resetAgentAudit()
	writeActivateAgentMode(t)
	seedAtom(t, "safety.theme", "light")

	if _, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id": "safety.theme", "value": "dark",
	}); parseErr != nil {
		t.Fatalf("set-atom: %s", parseErr.Message)
	}
	if parseGot := currentAtom(t, "safety.theme"); parseGot != "dark" {
		t.Fatalf("set-atom did not apply: %#v", parseGot)
	}

	parseUndoRaw, parseErr := safetyHandleUndo(nil)
	if parseErr != nil {
		t.Fatalf("undo: %s", parseErr.Message)
	}
	var parseUndo safetyUndoResult
	if parseDecErr := json.Unmarshal(parseUndoRaw, &parseUndo); parseDecErr != nil {
		t.Fatalf("decode undo: %v", parseDecErr)
	}
	if !parseUndo.Undone {
		t.Fatalf("expected undone=true")
	}
	if parseGot := currentAtom(t, "safety.theme"); parseGot != "light" {
		t.Fatalf("undo did not restore prior value: got %#v want \"light\"", parseGot)
	}
}

// TestUndoStackBounded pins that the undo stack and audit ring stay bounded
// under a flood of reversible mutations (regression: the undo stack used to
// grow without limit, retaining every prior-value closure forever).
func TestUndoStackBounded(t *testing.T) {
	resetAgentAudit()
	for parseI := 0; parseI < auditCap+50; parseI++ {
		RecordAgentMutation("test.op", "op", func() *EnvelopeError { return nil })
	}
	if parseN := len(AgentAuditEntries(0)); parseN > auditCap {
		t.Fatalf("audit entries unbounded: %d > cap %d", parseN, auditCap)
	}
	auditMu.Lock()
	parseStackLen := len(undoStack)
	auditMu.Unlock()
	if parseStackLen > auditCap {
		t.Fatalf("undo stack unbounded: %d > cap %d", parseStackLen, auditCap)
	}
	// The most recent op must still be undoable after trimming.
	if _, parseOk := popUndo(); !parseOk {
		t.Fatalf("expected the newest op to remain undoable after bounding")
	}
}

// TestSetAtomDryRunDoesNotApply pins that a dry-run set-atom validates and
// previews the change (from/to) without mutating the atom or recording it.
func TestSetAtomDryRunDoesNotApply(t *testing.T) {
	resetAgentAudit()
	writeActivateAgentMode(t)
	seedAtom(t, "safety.dry", "before")

	parseRaw, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id": "safety.dry", "value": "after", "dryRun": true,
	})
	if parseErr != nil {
		t.Fatalf("dry-run set-atom errored: %s", parseErr.Message)
	}
	var parsePreview writeSetAtomDryRunResult
	if parseDecErr := json.Unmarshal(parseRaw, &parsePreview); parseDecErr != nil {
		t.Fatalf("decode preview: %v", parseDecErr)
	}
	if !parsePreview.DryRun || parsePreview.From != "before" || parsePreview.To != "after" {
		t.Fatalf("preview wrong: %+v", parsePreview)
	}
	if parseGot := currentAtom(t, "safety.dry"); parseGot != "before" {
		t.Fatalf("dry-run mutated the atom: got %#v want \"before\"", parseGot)
	}
	if len(AgentAuditEntries(0)) != 0 {
		t.Fatalf("dry-run should record no audit entry, got %d", len(AgentAuditEntries(0)))
	}
}

// TestAuditTrailRecordsMutations pins that bridge.audit reports the applied
// mutations (the set-atom and the undo of it).
func TestAuditTrailRecordsMutations(t *testing.T) {
	resetAgentAudit()
	writeActivateAgentMode(t)
	seedAtom(t, "safety.count", 1)

	if _, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id": "safety.count", "value": 2,
	}); parseErr != nil {
		t.Fatalf("set-atom: %s", parseErr.Message)
	}
	if _, parseErr := safetyHandleUndo(nil); parseErr != nil {
		t.Fatalf("undo: %s", parseErr.Message)
	}

	parseRaw, parseErr := safetyHandleAudit(nil)
	if parseErr != nil {
		t.Fatalf("audit: %s", parseErr.Message)
	}
	var parseAudit safetyAuditResult
	if parseDecErr := json.Unmarshal(parseRaw, &parseAudit); parseDecErr != nil {
		t.Fatalf("decode audit: %v", parseDecErr)
	}
	if len(parseAudit.Entries) != 2 {
		t.Fatalf("expected 2 audit entries (set-atom, undo), got %d: %+v", len(parseAudit.Entries), parseAudit.Entries)
	}
	if parseAudit.Entries[0].Command != "bridge.set-atom" || !parseAudit.Entries[0].Reversible {
		t.Fatalf("first entry should be a reversible set-atom: %+v", parseAudit.Entries[0])
	}
	if parseAudit.Entries[1].Command != "bridge.undo" {
		t.Fatalf("second entry should be the undo: %+v", parseAudit.Entries[1])
	}
}

// TestUndoEmptyStackReportsNothing pins that undo with no reversible op returns
// undone=false (not an error).
func TestUndoEmptyStackReportsNothing(t *testing.T) {
	resetAgentAudit()
	writeActivateAgentMode(t)
	parseRaw, parseErr := safetyHandleUndo(nil)
	if parseErr != nil {
		t.Fatalf("undo: %s", parseErr.Message)
	}
	var parseUndo safetyUndoResult
	if parseDecErr := json.Unmarshal(parseRaw, &parseUndo); parseDecErr != nil {
		t.Fatalf("decode: %v", parseDecErr)
	}
	if parseUndo.Undone {
		t.Fatalf("expected undone=false on empty stack")
	}
}

// TestUndoForbiddenWhenInactive pins that undo refuses when agent mode is off.
func TestUndoForbiddenWhenInactive(t *testing.T) {
	resetAgentAudit()
	SetAgentModeActive(false)
	_, parseErr := safetyHandleUndo(nil)
	if parseErr == nil || parseErr.Code != ErrorCodeForbidden {
		t.Fatalf("expected forbidden, got %+v", parseErr)
	}
}

// TestBridgePublishReachesTypedSubscriber pins the publish fix at the bridge
// level: with a registered topic codec, a JSON bridge.publish reaches a
// concretely-typed Subscribe[T] handler.
func TestBridgePublishReachesTypedSubscriber(t *testing.T) {
	writeActivateAgentMode(t)
	events.RegisterTopic[string]("safety.saved")
	parseReceived := make(chan string, 1)
	parseUnsub := events.Subscribe[string]("safety.saved", func(parseValue string) {
		parseReceived <- parseValue
	})
	defer parseUnsub()

	if _, parseErr := writeCallHandler(writeHandlePublish, map[string]any{
		"topic": "safety.saved", "payload": "draft-7",
	}); parseErr != nil {
		t.Fatalf("publish: %s", parseErr.Message)
	}
	select {
	case parseGot := <-parseReceived:
		if parseGot != "draft-7" {
			t.Fatalf("typed subscriber got %q, want draft-7", parseGot)
		}
	default:
		t.Fatalf("typed subscriber did not receive the bridge publish (the gap is not closed)")
	}
}

// ensure imports used across the file are referenced for the linter when the
// state/runtime helpers move; seedAtom/currentAtom live in a sibling test file.
var _ = state.Snapshot{}
var _ = runtime.GetGlobalRuntime
