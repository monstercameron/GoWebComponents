package agentbridge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// TestMountIsAuditedAndReversible pins DEFECT-1/DEFECT-2 fixes: a successful
// bridge.mount is recorded in the audit trail as a reversible mutation, and
// bridge.undo unmounts it (registry returns to baseline).
func TestMountIsAuditedAndReversible(t *testing.T) {
	resetAgentAudit()
	writeActivateAgentMode(t)
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "audit-root")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	writeMountMu.Lock()
	parseBaseline := len(writeMountedRoots)
	writeMountMu.Unlock()

	RegisterMountComponent("agentbridge.audit.Panel", func(map[string]any) *runtime.Element {
		return runtime.CreateElement("div", map[string]any{"id": "audited-panel"})
	})

	if _, parseErr := writeHandleMount(json.RawMessage(`{"id":"audit-1","component":"agentbridge.audit.Panel","selector":"#audit-root"}`)); parseErr != nil {
		t.Fatalf("mount: %s", parseErr.Message)
	}

	parseEntries := AgentAuditEntries(0)
	if len(parseEntries) == 0 || parseEntries[len(parseEntries)-1].Command != "bridge.mount" {
		t.Fatalf("mount was not audited; entries=%+v", parseEntries)
	}
	if !parseEntries[len(parseEntries)-1].Reversible {
		t.Fatalf("mount audit entry should be reversible")
	}

	parseUndoRaw, parseUndoErr := safetyHandleUndo(nil)
	if parseUndoErr != nil {
		t.Fatalf("undo mount: %s", parseUndoErr.Message)
	}
	if !strings.Contains(string(parseUndoRaw), `"undone":true`) {
		t.Fatalf("undo result = %s", string(parseUndoRaw))
	}
	writeMountMu.Lock()
	parseFinal := len(writeMountedRoots)
	writeMountMu.Unlock()
	if parseFinal != parseBaseline {
		t.Fatalf("undo did not unmount: registry %d want %d", parseFinal, parseBaseline)
	}
}

// TestUnmountIsAuditedAndReversible pins the third-pass fix: a direct
// bridge.unmount is recorded in the audit trail and is reversible — undo
// remounts the same component with its original props.
func TestUnmountIsAuditedAndReversible(t *testing.T) {
	resetAgentAudit()
	writeActivateAgentMode(t)
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "unmount-root")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	RegisterMountComponent("agentbridge.unmount.Panel", func(map[string]any) *runtime.Element {
		return runtime.CreateElement("div", map[string]any{"id": "unmount-panel"})
	})
	if _, parseErr := writeHandleMount(json.RawMessage(`{"id":"u1","component":"agentbridge.unmount.Panel","selector":"#unmount-root"}`)); parseErr != nil {
		t.Fatalf("mount: %s", parseErr.Message)
	}
	// Isolate the unmount in the audit/undo state.
	resetAgentAudit()

	if _, parseErr := writeHandleUnmount(json.RawMessage(`{"id":"u1"}`)); parseErr != nil {
		t.Fatalf("unmount: %s", parseErr.Message)
	}
	parseEntries := AgentAuditEntries(0)
	if len(parseEntries) != 1 || parseEntries[0].Command != "bridge.unmount" || !parseEntries[0].Reversible {
		t.Fatalf("unmount not audited as reversible: %+v", parseEntries)
	}
	writeMountMu.Lock()
	_, parseStillMounted := writeMountedRoots["u1"]
	writeMountMu.Unlock()
	if parseStillMounted {
		t.Fatal("unmount did not remove the registry entry")
	}

	if _, parseErr := safetyHandleUndo(nil); parseErr != nil {
		t.Fatalf("undo unmount: %s", parseErr.Message)
	}
	writeMountMu.Lock()
	_, parseRemounted := writeMountedRoots["u1"]
	writeMountMu.Unlock()
	if !parseRemounted {
		t.Fatal("undo of unmount did not remount the component")
	}
}

// TestMountRejectsMissingSelector pins DEFECT-4: a bridge.mount onto a selector
// that does not resolve is rejected (rather than silently succeeding because
// the runtime's not-found panic is suppressed) and leaves no stuck reservation.
func TestMountRejectsMissingSelector(t *testing.T) {
	resetAgentAudit()
	writeActivateAgentMode(t)
	parseAdapter := mockdom.NewMockDOMAdapter()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.SetAttribute(parseContainer, "id", "real-root")
	runtime.InitGlobalRuntime(runtime.Config{DOMAdapter: parseAdapter, Reset: true})

	RegisterMountComponent("agentbridge.sel.Panel", func(map[string]any) *runtime.Element {
		return runtime.CreateElement("div", map[string]any{"id": "sel-panel"})
	})

	_, parseErr := writeHandleMount(json.RawMessage(`{"id":"sel-1","component":"agentbridge.sel.Panel","selector":"#ghost-does-not-exist"}`))
	if parseErr == nil || parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("expected bad-payload for a missing selector, got %+v", parseErr)
	}
	writeMountMu.Lock()
	_, parseStuck := writeMountedRoots["sel-1"]
	writeMountMu.Unlock()
	if parseStuck {
		t.Fatal("a rejected mount left a stuck reservation")
	}
}

// TestAuditUndoConsistencyUnderMixedFlood pins DEFECT-5: after a flood of
// non-reversible mutations rolls the audit ring, no surviving undo op has a
// seq older than the oldest surviving audit entry (every undoable op keeps a
// matching audit record).
func TestAuditUndoConsistencyUnderMixedFlood(t *testing.T) {
	resetAgentAudit()
	// First a batch of reversible ops (these push undo ops with low seqs).
	for parseI := 0; parseI < auditCap; parseI++ {
		RecordAgentMutation("test.rev", "rev", func() *EnvelopeError { return nil })
	}
	// Then a flood of non-reversible ops that roll the audit ring forward.
	for parseI := 0; parseI < auditCap+20; parseI++ {
		RecordAgentMutation("test.nonrev", "nonrev", nil)
	}

	auditMu.Lock()
	defer auditMu.Unlock()
	if len(auditEntries) == 0 {
		t.Fatal("expected audit entries")
	}
	parseOldestAuditSeq := auditEntries[0].Seq
	for _, parseOp := range undoStack {
		if parseOp.seq < parseOldestAuditSeq {
			t.Fatalf("undo op seq %d is older than oldest audit seq %d (orphaned undo)", parseOp.seq, parseOldestAuditSeq)
		}
	}
}
