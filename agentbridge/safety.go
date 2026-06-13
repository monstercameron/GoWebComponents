package agentbridge

import (
	"encoding/json"
	"sync"
)

// auditCap bounds the in-memory audit ring so a long agent session cannot grow
// it without limit.
const auditCap = 256

// auditMu guards the audit log and the undo stack.
var auditMu sync.Mutex

// auditSeq is the monotonic id assigned to each recorded mutation.
var auditSeq uint64

// auditEntries is a bounded ring of recorded mutations, oldest first.
var auditEntries []AgentAuditEntry

// undoStack holds reversible operations in apply order; the last is undone first.
var undoStack []agentUndoOp

// AgentAuditEntry is one recorded write the bridge applied, for the bridge.audit
// trail an agent (or a human reviewing an agent) reads to see what changed.
type AgentAuditEntry struct {
	Seq        uint64 `json:"seq"`
	Command    string `json:"command"`
	Summary    string `json:"summary"`
	Reversible bool   `json:"reversible"`
}

// agentUndoOp pairs a recorded mutation with the closure that reverses it.
type agentUndoOp struct {
	seq   uint64
	label string
	undo  func() *EnvelopeError
}

// RecordAgentMutation appends an audit entry for an applied write command and,
// when parseUndo is non-nil, pushes a reversible op onto the undo stack. It
// returns the assigned audit sequence number. parseUndo must restore the exact
// prior state and is safe to call once.
func RecordAgentMutation(parseCommand string, parseSummary string, parseUndo func() *EnvelopeError) uint64 {
	auditMu.Lock()
	defer auditMu.Unlock()
	auditSeq++
	parseEntry := AgentAuditEntry{
		Seq:        auditSeq,
		Command:    parseCommand,
		Summary:    parseSummary,
		Reversible: parseUndo != nil,
	}
	auditEntries = append(auditEntries, parseEntry)
	if len(auditEntries) > auditCap {
		// Copy into a fresh slice so the dropped prefix is released for GC
		// instead of being retained by the backing array.
		parseTrimmed := make([]AgentAuditEntry, auditCap)
		copy(parseTrimmed, auditEntries[len(auditEntries)-auditCap:])
		auditEntries = parseTrimmed
	}
	if parseUndo != nil {
		undoStack = append(undoStack, agentUndoOp{seq: auditSeq, label: parseSummary, undo: parseUndo})
		// Bound the undo stack to the same cap. Without this it grows without
		// limit (each op retains a closure capturing a prior value), leaking
		// memory and letting the undo stack outlive the audit trail it mirrors.
		if len(undoStack) > auditCap {
			parseTrimmedUndo := make([]agentUndoOp, auditCap)
			copy(parseTrimmedUndo, undoStack[len(undoStack)-auditCap:])
			undoStack = parseTrimmedUndo
		}
	}
	// Keep the undo stack consistent with the audit ring: a flood of
	// non-reversible mutations can roll an old reversible op's audit entry out
	// of the ring while its undo op survives. Drop any undo op whose seq is
	// older than the oldest surviving audit entry so every undoable op always
	// has a matching audit record.
	if len(auditEntries) > 0 && len(undoStack) > 0 {
		parseOldestSeq := auditEntries[0].Seq
		parseCut := 0
		for parseCut < len(undoStack) && undoStack[parseCut].seq < parseOldestSeq {
			parseCut++
		}
		if parseCut > 0 {
			parseKept := make([]agentUndoOp, len(undoStack)-parseCut)
			copy(parseKept, undoStack[parseCut:])
			undoStack = parseKept
		}
	}
	return auditSeq
}

// AgentAuditEntries returns up to parseMax most-recent audit entries, oldest
// first. A non-positive parseMax returns the whole retained ring.
func AgentAuditEntries(parseMax int) []AgentAuditEntry {
	auditMu.Lock()
	defer auditMu.Unlock()
	parseStart := 0
	if parseMax > 0 && parseMax < len(auditEntries) {
		parseStart = len(auditEntries) - parseMax
	}
	parseOut := make([]AgentAuditEntry, len(auditEntries)-parseStart)
	copy(parseOut, auditEntries[parseStart:])
	return parseOut
}

// resetAgentAudit clears the audit ring and undo stack. It exists for tests so
// one test's mutations do not leak into another's assertions.
func resetAgentAudit() {
	auditMu.Lock()
	defer auditMu.Unlock()
	auditSeq = 0
	auditEntries = nil
	undoStack = nil
}

// popUndo removes and returns the most recent reversible op, or ok=false when
// the stack is empty.
func popUndo() (agentUndoOp, bool) {
	auditMu.Lock()
	defer auditMu.Unlock()
	if len(undoStack) == 0 {
		return agentUndoOp{}, false
	}
	parseOp := undoStack[len(undoStack)-1]
	undoStack = undoStack[:len(undoStack)-1]
	return parseOp, true
}

// ---------------------------------------------------------------------------
// Commands: bridge.audit (read), bridge.undo (reverse last mutation)
// ---------------------------------------------------------------------------

// RegisterSafetyCommands installs bridge.audit and bridge.undo. bridge.audit is
// read-only; bridge.undo reverses the most recent reversible mutation and is
// itself gated by agent mode.
func RegisterSafetyCommands() {
	RegisterAgentCommand("bridge.audit", safetyHandleAudit)
	RegisterAgentCommand("bridge.undo", safetyHandleUndo)
}

// safetyAuditPayload is the decoded bridge.audit request.
type safetyAuditPayload struct {
	Max int `json:"max"`
}

// safetyAuditResult is the bridge.audit response.
type safetyAuditResult struct {
	Entries []AgentAuditEntry `json:"entries"`
}

// safetyHandleAudit returns the recent mutation audit trail. It is read-only
// and works regardless of agent mode (reading what happened is always allowed).
func safetyHandleAudit(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	parseReq := safetyAuditPayload{}
	if len(parsePayload) > 0 {
		if parseErr := json.Unmarshal(parsePayload, &parseReq); parseErr != nil {
			return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: "audit: malformed payload: " + parseErr.Error()}
		}
	}
	parseOut := safetyAuditResult{Entries: AgentAuditEntries(parseReq.Max)}
	parseBytes, parseErr := json.Marshal(parseOut)
	if parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseErr.Error()}
	}
	return parseBytes, nil
}

// safetyUndoResult is the bridge.undo response.
type safetyUndoResult struct {
	Undone  bool   `json:"undone"`
	Seq     uint64 `json:"seq,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// safetyHandleUndo reverses the most recent reversible mutation. It refuses with
// ErrorCodeForbidden when agent mode is off (undo mutates live state), and
// reports undone=false when there is nothing to undo (not an error).
func safetyHandleUndo(parsePayload json.RawMessage) (json.RawMessage, *EnvelopeError) {
	if !IsAgentModeActive() {
		return nil, &EnvelopeError{Code: ErrorCodeForbidden, Message: "agent mode is not active"}
	}
	parseOp, parseOk := popUndo()
	if !parseOk {
		parseBytes, _ := json.Marshal(safetyUndoResult{Undone: false})
		return parseBytes, nil
	}
	if parseErr := parseOp.undo(); parseErr != nil {
		return nil, parseErr
	}
	// Record the undo itself so the trail shows the reversal (not reversible).
	RecordAgentMutation("bridge.undo", "undo of #"+itoa(parseOp.seq)+" ("+parseOp.label+")", nil)
	parseBytes, parseErr := json.Marshal(safetyUndoResult{Undone: true, Seq: parseOp.seq, Summary: parseOp.label})
	if parseErr != nil {
		return nil, &EnvelopeError{Code: ErrorCodeBadPayload, Message: parseErr.Error()}
	}
	return parseBytes, nil
}

// itoa renders a uint64 without importing strconv at the call site.
func itoa(parseValue uint64) string {
	if parseValue == 0 {
		return "0"
	}
	parseBuf := [20]byte{}
	parseIdx := len(parseBuf)
	for parseValue > 0 {
		parseIdx--
		parseBuf[parseIdx] = byte('0' + parseValue%10)
		parseValue /= 10
	}
	return string(parseBuf[parseIdx:])
}
