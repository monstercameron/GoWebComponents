package services

import (
	"sort"
	"testing"
)

// v5 P3.3 — ownership arbitration.
//
// The property under test is that older state can never overwrite newer state,
// by any of the three routes it can arrive: during fallback, from a replaced
// worker, or from output computed before the local side advanced.

func TestArbiterDefaultsToRemoteOwnership(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseArbiter.Owner("unit") != OwnerRemote {
		parseT.Error("a unit that has never failed must be remotely owned")
	}
	if !parseArbiter.AdmitRemote("unit", 1, 1).Accepted() {
		parseT.Error("remote output for a healthy unit must be admitted")
	}
	if _, hasState := parseArbiter.State("unit"); hasState {
		parseT.Error("a unit that has never failed must have no fallback record")
	}
}

func TestArbiterFallbackTransfersOwnership(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureTransport, "malformed payload", 3, 12); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}
	if parseArbiter.Owner("unit") != OwnerLocal {
		parseT.Error("a failure must transfer ownership to local")
	}

	parseState, hasState := parseArbiter.State("unit")
	if !hasState {
		parseT.Fatal("a fallback must be recorded")
	}
	if parseState.Class != FailureTransport {
		parseT.Errorf("class = %q, want %q", parseState.Class, FailureTransport)
	}
	if parseState.Reason == "" {
		parseT.Error("a fallback must carry its reason for diagnostics")
	}
	if parseState.Version != 12 {
		parseT.Errorf("version = %d, want 12", parseState.Version)
	}
}

// TestArbiterRejectsRemoteOutputDuringFallback is the primary hazard: a worker
// that keeps running after the local side took over.
func TestArbiterRejectsRemoteOutputDuringFallback(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureTimeout, "no answer", 1, 5); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}

	// Even output that looks newer must be rejected — the local side is
	// authoritative, and its version numbering is its own.
	parseVerdict := parseArbiter.AdmitRemote("unit", 99, 99)
	if parseVerdict.Accepted() {
		parseT.Error("remote output must be ignored while the local side owns the unit")
	}
	if parseVerdict != VerdictRejectLocalOwned {
		parseT.Errorf("verdict = %s, want reject-local-owned", parseVerdict)
	}
}

func TestArbiterRejectsUnknownFailureClass(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureClass("typo"), "reason", 1, 1); parseErr == nil {
		parseT.Error("an unknown failure class must be rejected, not absorbed")
	}
	if parseArbiter.Owner("unit") != OwnerRemote {
		parseT.Error("a rejected fallback must not have transferred ownership")
	}
}

func TestArbiterFallbackKeepsMostRecentCause(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureTransport, "first", 1, 5); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}
	if parseErr := parseArbiter.Fallback("unit", FailurePanic, "second", 1, 3); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}

	parseState, _ := parseArbiter.State("unit")
	if parseState.Class != FailurePanic {
		parseT.Errorf("class = %q, want the most recent cause", parseState.Class)
	}
	// The version high-water mark must not regress even though the second
	// failure reported a lower one.
	if parseState.Version != 5 {
		parseT.Errorf("version = %d, want 5 — the high-water mark regressed", parseState.Version)
	}
}

// ------------------------------------------------------------------- restore

// TestArbiterRestoreRequiresAFreshEpoch: restoring at a reused epoch would
// readmit exactly the in-flight output the fallback existed to exclude.
func TestArbiterRestoreRequiresAFreshEpoch(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureWorkerDeath, "died", 4, 10); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}

	if parseErr := parseArbiter.Restore("unit", 0); parseErr == nil {
		parseT.Error("restoring at epoch 0 must be refused")
	}
	if parseErr := parseArbiter.Restore("unit", 4); parseErr == nil {
		parseT.Error("restoring at the failed epoch must be refused")
	}
	if parseErr := parseArbiter.Restore("unit", 5); parseErr != nil {
		parseT.Errorf("restoring at a fresh epoch must succeed: %v", parseErr)
	}
	if parseArbiter.Owner("unit") != OwnerRemote {
		parseT.Error("a successful restore must return ownership to remote")
	}
}

// TestArbiterRejectsStaleRemoteOutputAfterRestore is the documented difference
// from runtime2's coordinator, which accepts epoch and version arguments on its
// worker-output handlers and never reads them. That is safe there because the
// patch parse layer rejects a stale epoch first; a payload-agnostic arbiter has
// no such layer and must carry the guard itself.
func TestArbiterRejectsStaleRemoteOutputAfterRestore(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureWorkerDeath, "died", 4, 10); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}
	if parseErr := parseArbiter.Restore("unit", 5); parseErr != nil {
		parseT.Fatalf("Restore: %v", parseErr)
	}

	// The dead worker's in-flight output, arriving after its replacement started.
	if parseVerdict := parseArbiter.AdmitRemote("unit", 4, 50); parseVerdict != VerdictRejectStaleEpoch {
		parseT.Errorf("verdict = %s, want reject-stale-epoch", parseVerdict)
	}
	// Output from the new worker that is behind what the local side computed
	// while it owned the unit.
	if parseVerdict := parseArbiter.AdmitRemote("unit", 5, 9); parseVerdict != VerdictRejectStaleVersion {
		parseT.Errorf("verdict = %s, want reject-stale-version", parseVerdict)
	}
	// Current output from the new worker.
	if parseVerdict := parseArbiter.AdmitRemote("unit", 5, 11); !parseVerdict.Accepted() {
		parseT.Errorf("verdict = %s, want accept", parseVerdict)
	}
}

func TestArbiterRecordsLocalProgressMonotonically(parseT *testing.T) {
	parseArbiter := NewArbiter()
	parseArbiter.RecordLocalProgress("unit", 10)
	parseArbiter.RecordLocalProgress("unit", 4)
	if parseVersion := parseArbiter.LocalVersion("unit"); parseVersion != 10 {
		parseT.Errorf("local version = %d, want 10 — the high-water mark regressed", parseVersion)
	}
}

// --------------------------------------------------------------- worker death

func TestArbiterHandleRemoteDeathReassigns(parseT *testing.T) {
	parseArbiter := NewArbiter()
	parseOutcome, parseErr := parseArbiter.HandleRemoteDeath("unit", true, 7, 3)
	if parseErr != nil {
		parseT.Fatalf("HandleRemoteDeath: %v", parseErr)
	}
	if !parseOutcome.Reassigned || parseOutcome.FellBack {
		parseT.Errorf("outcome = %+v, want reassigned", parseOutcome)
	}
	if parseOutcome.RestoredEpoch != 7 {
		parseT.Errorf("restored epoch = %d, want 7", parseOutcome.RestoredEpoch)
	}
	if parseArbiter.Owner("unit") != OwnerRemote {
		parseT.Error("a reassigned unit must be remotely owned")
	}
}

// TestArbiterHandleRemoteDeathWithoutReassignFallsBackQuietly: this is the
// design working, not an error.
func TestArbiterHandleRemoteDeathWithoutReassignFallsBackQuietly(parseT *testing.T) {
	parseArbiter := NewArbiter()
	parseOutcome, parseErr := parseArbiter.HandleRemoteDeath("unit", false, 0, 3)
	if parseErr != nil {
		parseT.Fatalf("falling back without reassignment support is not an error: %v", parseErr)
	}
	if !parseOutcome.FellBack || parseOutcome.Reassigned {
		parseT.Errorf("outcome = %+v, want fell-back", parseOutcome)
	}
	if parseArbiter.Owner("unit") != OwnerLocal {
		parseT.Error("a unit with no reassignment path must be locally owned")
	}
}

// TestArbiterHandleRemoteDeathClaimingSupportWithNoEpochIsABug: it still falls
// back — losing availability quietly would be worse — but it says so.
func TestArbiterHandleRemoteDeathClaimingSupportWithNoEpochIsABug(parseT *testing.T) {
	parseArbiter := NewArbiter()
	parseOutcome, parseErr := parseArbiter.HandleRemoteDeath("unit", true, 0, 3)
	if parseErr == nil {
		parseT.Error("claiming reassignment support and supplying no epoch must be reported")
	}
	if !parseOutcome.FellBack {
		parseT.Error("the unit must still fall back rather than being left unowned")
	}
	if parseArbiter.Owner("unit") != OwnerLocal {
		parseT.Error("owner must be local after a failed reassignment")
	}
}

// TestArbiterHandleRemoteDeathRefusesToReassignOntoAStaleEpoch: reassigning at
// a reused epoch would readmit the dead worker's in-flight output.
func TestArbiterHandleRemoteDeathRefusesToReassignOntoAStaleEpoch(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureCommit, "bad commit", 9, 4); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}

	parseOutcome, parseErr := parseArbiter.HandleRemoteDeath("unit", true, 9, 4)
	if parseErr == nil {
		parseT.Error("reassigning onto a reused epoch must be refused")
	}
	if !parseOutcome.FellBack {
		parseT.Error("a refused reassignment must leave the unit locally owned")
	}
	if parseArbiter.AdmitRemote("unit", 9, 100).Accepted() {
		parseT.Error("the dead worker's output must not be admitted after a refused reassignment")
	}
}

// ------------------------------------------------------------------ inventory

// TestArbiterListsLocallyOwnedUnits covers a diagnostics gap in the source,
// which had no way to answer "which units have quietly fallen back".
func TestArbiterListsLocallyOwnedUnits(parseT *testing.T) {
	parseArbiter := NewArbiter()
	for _, parseUnitID := range []string{"a", "b", "c"} {
		if parseErr := parseArbiter.Fallback(parseUnitID, FailureTransport, "x", 1, 1); parseErr != nil {
			parseT.Fatalf("Fallback(%s): %v", parseUnitID, parseErr)
		}
	}
	if parseErr := parseArbiter.Restore("b", 2); parseErr != nil {
		parseT.Fatalf("Restore: %v", parseErr)
	}

	parseUnits := parseArbiter.LocalOwnedUnits()
	sort.Strings(parseUnits)
	if len(parseUnits) != 2 || parseUnits[0] != "a" || parseUnits[1] != "c" {
		parseT.Errorf("locally owned = %v, want [a c]", parseUnits)
	}
}

func TestArbiterForgetClearsEverything(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("unit", FailureTransport, "x", 1, 20); parseErr != nil {
		parseT.Fatalf("Fallback: %v", parseErr)
	}
	parseArbiter.Forget("unit")

	if parseArbiter.Owner("unit") != OwnerRemote {
		parseT.Error("a forgotten unit reverts to the remote default")
	}
	if parseArbiter.LocalVersion("unit") != 0 {
		parseT.Error("Forget must clear the local version high-water mark")
	}
	if len(parseArbiter.LocalOwnedUnits()) != 0 {
		parseT.Error("a forgotten unit must not be listed")
	}
}

// -------------------------------------------------------------------- guards

func TestArbiterNilReceiverIsSafe(parseT *testing.T) {
	var parseArbiter *Arbiter
	if parseErr := parseArbiter.Fallback("unit", FailureTransport, "x", 1, 1); parseErr == nil {
		parseT.Error("a nil arbiter must error rather than panic")
	}
	if parseErr := parseArbiter.Restore("unit", 1); parseErr == nil {
		parseT.Error("a nil arbiter must error rather than panic")
	}
	if _, parseErr := parseArbiter.HandleRemoteDeath("unit", true, 1, 1); parseErr == nil {
		parseT.Error("a nil arbiter must error rather than panic")
	}
	parseArbiter.RecordLocalProgress("unit", 1)
	parseArbiter.Forget("unit")
	if parseArbiter.Owner("unit") != OwnerRemote || parseArbiter.LocalVersion("unit") != 0 {
		parseT.Error("a nil arbiter reads as the default state")
	}
}

func TestArbiterRejectsEmptyUnitID(parseT *testing.T) {
	parseArbiter := NewArbiter()
	if parseErr := parseArbiter.Fallback("  ", FailureTransport, "x", 1, 1); parseErr == nil {
		parseT.Error("a blank unit id must be rejected")
	}
	if parseErr := parseArbiter.Restore("", 1); parseErr == nil {
		parseT.Error("a blank unit id must be rejected")
	}
}

func TestOwnershipLabelsAreDistinct(parseT *testing.T) {
	if OwnerLocal.String() == OwnerRemote.String() {
		parseT.Error("owner labels must be distinct")
	}
	parseSeen := map[string]bool{}
	for _, parseVerdict := range []RemoteVerdict{
		VerdictAccept, VerdictRejectLocalOwned, VerdictRejectStaleEpoch, VerdictRejectStaleVersion,
	} {
		if parseSeen[parseVerdict.String()] {
			parseT.Errorf("verdict label %q is not distinct", parseVerdict)
		}
		parseSeen[parseVerdict.String()] = true
	}
	if RemoteVerdict(200).String() == "" {
		parseT.Error("an unknown verdict must still render for diagnostics")
	}
}
