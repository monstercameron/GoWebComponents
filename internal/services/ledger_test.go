package services

import (
	"fmt"
	"testing"
)

// v5 P3.3 — the extracted idempotency ledger.
//
// The behavior under test is the CLASSIFICATION, not the bookkeeping: duplicate,
// stale, stale-epoch, and conflict are four different things and only one of
// them is an error. A ledger that collapsed them would still "work" on the happy
// path and lose data on a reordered delivery.

func TestLedgerAppliesFirstMessage(parseT *testing.T) {
	parseLedger := NewLedger()
	parseDecision, parseErr := parseLedger.Apply("stream", 1, 1, "id-1")
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if !parseDecision.ShouldApply() {
		parseT.Errorf("decision = %s, want apply", parseDecision)
	}
}

func TestLedgerSkipsExactDuplicate(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Apply("stream", 1, 5, "id-5"); parseErr != nil {
		parseT.Fatalf("first Apply: %v", parseErr)
	}
	parseDecision, parseErr := parseLedger.Apply("stream", 1, 5, "id-5")
	if parseErr != nil {
		parseT.Fatalf("duplicate must not error: %v", parseErr)
	}
	if parseDecision != DecisionSkipDuplicate {
		parseT.Errorf("decision = %s, want skip-duplicate", parseDecision)
	}
}

// TestLedgerSkipsReorderedDelivery is the guard that matters most across an
// async bridge: a never-seen version below the high-water mark carries older
// state, so applying it would regress the receiver.
func TestLedgerSkipsReorderedDelivery(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Apply("stream", 1, 10, "id-10"); parseErr != nil {
		parseT.Fatalf("Apply(10): %v", parseErr)
	}
	parseDecision, parseErr := parseLedger.Apply("stream", 1, 4, "id-4")
	if parseErr != nil {
		parseT.Fatalf("a reordered delivery is normal, not an error: %v", parseErr)
	}
	if parseDecision != DecisionSkipStale {
		parseT.Errorf("decision = %s, want skip-stale", parseDecision)
	}
}

// TestLedgerReportsConflict: two different payloads claiming one version is a
// producer bug, and the only case that deserves an error.
func TestLedgerReportsConflict(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Apply("stream", 1, 3, "id-a"); parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	parseDecision, parseErr := parseLedger.Apply("stream", 1, 3, "id-b")
	if parseErr == nil {
		parseT.Fatal("a version/identity conflict must be reported")
	}
	if parseDecision.ShouldApply() {
		parseT.Error("a conflicting message must never be applied")
	}
}

func TestLedgerIsolatesStreams(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Apply("a", 1, 9, "id-9"); parseErr != nil {
		parseT.Fatalf("Apply(a): %v", parseErr)
	}
	parseDecision, parseErr := parseLedger.Apply("b", 1, 1, "id-1")
	if parseErr != nil {
		parseT.Fatalf("Apply(b): %v", parseErr)
	}
	if !parseDecision.ShouldApply() {
		parseT.Error("one stream's high-water mark must not gate another's")
	}
}

// ------------------------------------------------------------------- epochs

func TestLedgerNewEpochResetsVersions(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Apply("stream", 1, 50, "id-50"); parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	parseDecision, parseErr := parseLedger.Apply("stream", 2, 1, "id-1")
	if parseErr != nil {
		parseT.Fatalf("Apply(new epoch): %v", parseErr)
	}
	if !parseDecision.ShouldApply() {
		parseT.Error("a new epoch restarts versioning, so version 1 must apply")
	}
}

// TestLedgerSkipsStaleEpoch pins the deliberate difference from runtime2's
// tracker, which treats ANY epoch mismatch as a reset. That is safe there only
// because its parse layer rejects an unexpected epoch first; a generic ledger
// has nothing above it, so a late delivery from a past epoch would otherwise
// wipe the current epoch's high-water mark.
func TestLedgerSkipsStaleEpoch(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Apply("stream", 2, 7, "id-7"); parseErr != nil {
		parseT.Fatalf("Apply(epoch 2): %v", parseErr)
	}

	parseDecision, parseErr := parseLedger.Apply("stream", 1, 99, "id-old")
	if parseErr != nil {
		parseT.Fatalf("a late epoch-1 delivery is normal, not an error: %v", parseErr)
	}
	if parseDecision != DecisionSkipStaleEpoch {
		parseT.Errorf("decision = %s, want skip-stale-epoch", parseDecision)
	}

	// The current epoch's high-water mark must have survived it.
	parseAfter, parseAfterErr := parseLedger.Apply("stream", 2, 3, "id-3")
	if parseAfterErr != nil {
		parseT.Fatalf("Apply(epoch 2, stale version): %v", parseAfterErr)
	}
	if parseAfter != DecisionSkipStale {
		parseT.Errorf("decision = %s, want skip-stale — the stale epoch wiped the high-water mark", parseAfter)
	}
}

// TestLedgerRefusesToCommitAStaleEpoch: Check already skips these, so reaching
// Commit means a caller ignored the decision. Absorbing it would roll the
// high-water mark backwards and let every applied message replay.
func TestLedgerRefusesToCommitAStaleEpoch(parseT *testing.T) {
	parseLedger := NewLedger()
	if parseErr := parseLedger.Commit("stream", 5, 1, "id-1"); parseErr != nil {
		parseT.Fatalf("Commit(epoch 5): %v", parseErr)
	}
	if parseErr := parseLedger.Commit("stream", 4, 1, "id-1"); parseErr == nil {
		parseT.Error("committing a past epoch must be refused")
	}
}

// ------------------------------------------------------- check/commit split

// TestLedgerCheckDoesNotMutate is the #72 fix: a message that fails body
// validation after Check must not record its version, or a later corrected
// message at that version is rejected as a conflict and the stream wedges
// permanently.
func TestLedgerCheckDoesNotMutate(parseT *testing.T) {
	parseLedger := NewLedger()

	parseFirst, parseFirstErr := parseLedger.Check("stream", 1, 1, "id-bad")
	if parseFirstErr != nil || !parseFirst.ShouldApply() {
		parseT.Fatalf("Check = %s, %v; want apply", parseFirst, parseFirstErr)
	}
	// Body validation fails here — no Commit.

	parseSecond, parseSecondErr := parseLedger.Check("stream", 1, 1, "id-corrected")
	if parseSecondErr != nil {
		parseT.Fatalf("the corrected message must not be a conflict: %v", parseSecondErr)
	}
	if !parseSecond.ShouldApply() {
		parseT.Errorf("decision = %s, want apply — a failed message wedged the stream", parseSecond)
	}
}

func TestLedgerCommitIsRepeatable(parseT *testing.T) {
	parseLedger := NewLedger()
	for parseAttempt := range 3 {
		if parseErr := parseLedger.Commit("stream", 1, 2, "id-2"); parseErr != nil {
			parseT.Fatalf("Commit attempt %d: %v", parseAttempt, parseErr)
		}
	}
	if parseCount := parseLedger.TrackedVersions("stream"); parseCount != 1 {
		parseT.Errorf("tracked versions = %d, want 1 — a repeated commit double-recorded", parseCount)
	}
}

// ----------------------------------------------------------------- validation

func TestLedgerRejectsMissingIdentifiers(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Check("", 1, 1, "id"); parseErr == nil {
		parseT.Error("an empty stream id must be rejected")
	}
	if _, parseErr := parseLedger.Check("stream", 1, 1, ""); parseErr == nil {
		parseT.Error("an empty identity must be rejected")
	}
	if parseErr := parseLedger.Commit("stream", 1, 1, ""); parseErr == nil {
		parseT.Error("Commit must reject an empty identity too")
	}
}

func TestLedgerNilReceiverIsSafe(parseT *testing.T) {
	var parseLedger *Ledger
	if _, parseErr := parseLedger.Check("stream", 1, 1, "id"); parseErr == nil {
		parseT.Error("a nil ledger must error rather than panic")
	}
	if parseErr := parseLedger.Commit("stream", 1, 1, "id"); parseErr == nil {
		parseT.Error("a nil ledger must error rather than panic")
	}
	parseLedger.Forget("stream")
	if parseLedger.TrackedStreams() != 0 {
		parseT.Error("a nil ledger tracks nothing")
	}
}

// ------------------------------------------------------------------ retention

// TestLedgerRetentionBoundsMemory: the source kept one map entry per version for
// the life of an epoch, which a long-lived domain stream grows without bound.
func TestLedgerRetentionBoundsMemory(parseT *testing.T) {
	parseLedger := NewLedgerWithRetention(8)
	for parseVersion := uint64(1); parseVersion <= 100; parseVersion++ {
		if _, parseErr := parseLedger.Apply("stream", 1, parseVersion, fmt.Sprintf("id-%d", parseVersion)); parseErr != nil {
			parseT.Fatalf("Apply(%d): %v", parseVersion, parseErr)
		}
	}
	if parseCount := parseLedger.TrackedVersions("stream"); parseCount != 8 {
		parseT.Errorf("tracked versions = %d, want 8 — retention is not bounding memory", parseCount)
	}
}

// TestLedgerRetentionNeverChangesApplyOutcome is the argument that makes
// bounding safe, run as a test rather than asserted in a comment.
//
// Eviction only ever removes versions BELOW the high-water mark, and the
// monotonicity guard already skips every never-seen version below that mark. So
// an evicted duplicate is classified skip-stale instead of skip-duplicate — a
// different reason for the identical outcome of not applying it. A ledger with
// unbounded retention and one with a tiny window must agree on every
// apply/skip decision.
func TestLedgerRetentionNeverChangesApplyOutcome(parseT *testing.T) {
	parseBounded := NewLedgerWithRetention(4)
	parseUnbounded := NewLedgerWithRetention(1 << 20)

	// A sequence mixing in-order arrivals, duplicates, and deep reorders.
	parseSequence := []uint64{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 9, 4, 10, 2, 11, 1, 12}

	for parseStep, parseVersion := range parseSequence {
		parseIdentity := fmt.Sprintf("id-%d", parseVersion)

		parseBoundedDecision, parseBoundedErr := parseBounded.Apply("stream", 1, parseVersion, parseIdentity)
		parseUnboundedDecision, parseUnboundedErr := parseUnbounded.Apply("stream", 1, parseVersion, parseIdentity)

		if (parseBoundedErr == nil) != (parseUnboundedErr == nil) {
			parseT.Fatalf("step %d (version %d): bounded err=%v, unbounded err=%v",
				parseStep, parseVersion, parseBoundedErr, parseUnboundedErr)
		}
		if parseBoundedDecision.ShouldApply() != parseUnboundedDecision.ShouldApply() {
			parseT.Fatalf("step %d (version %d): bounded said %s, unbounded said %s — retention changed an apply outcome",
				parseStep, parseVersion, parseBoundedDecision, parseUnboundedDecision)
		}
	}
}

// ------------------------------------------------------------------- lifecycle

// TestLedgerForgetReleasesState covers what the source could not do at all:
// streams that genuinely end should release state rather than accumulate.
func TestLedgerForgetReleasesState(parseT *testing.T) {
	parseLedger := NewLedger()
	if _, parseErr := parseLedger.Apply("stream", 1, 1, "id-1"); parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if parseLedger.TrackedStreams() != 1 {
		parseT.Fatalf("tracked streams = %d, want 1", parseLedger.TrackedStreams())
	}

	parseLedger.Forget("stream")
	if parseLedger.TrackedStreams() != 0 {
		parseT.Errorf("tracked streams = %d, want 0 after Forget", parseLedger.TrackedStreams())
	}

	// Documented consequence: forgetting loses the high-water mark, so an
	// in-flight stale delivery would now be applied. Pinned so the trade-off is
	// visible rather than discovered.
	parseDecision, parseErr := parseLedger.Apply("stream", 1, 1, "id-1")
	if parseErr != nil {
		parseT.Fatalf("Apply after Forget: %v", parseErr)
	}
	if !parseDecision.ShouldApply() {
		parseT.Errorf("decision = %s, want apply — Forget clears history by design", parseDecision)
	}
}

func TestDecisionStringIsDistinct(parseT *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseDecision := range []Decision{
		DecisionApply, DecisionSkipDuplicate, DecisionSkipStale, DecisionSkipStaleEpoch,
	} {
		parseLabel := parseDecision.String()
		if parseSeen[parseLabel] {
			parseT.Errorf("decision label %q is not distinct", parseLabel)
		}
		parseSeen[parseLabel] = true
	}
	if Decision(200).String() == "" {
		parseT.Error("an unknown decision must still render for diagnostics")
	}
}
