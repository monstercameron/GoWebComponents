package domain

import (
	"errors"
	"testing"
)

// Exactly-once across a crash (P3.4 criterion (a)).
//
// The window is specific: a process that dies AFTER a command's effect succeeds
// and BEFORE its applied-record is written. On restart the next run sees no
// record and runs the effect again. Everything else about Execute already
// refuses duplicates, which is why this needed a test that simulates the crash
// rather than a failure — a returned error exercises the path that already
// worked.
//
// "Crash" here is modelled as: the effect committed, and the in-process ledger
// was lost. That is exactly what a restart looks like to the domain runtime —
// the durable store survives, the memory does not — so a fresh Runtime over the
// SAME store is the honest reproduction.

// countingEffect records how many times the underlying work actually ran.
type countingEffect struct {
	runs int
}

func (parseEffect *countingEffect) apply() error {
	parseEffect.runs++
	return nil
}

// TestExactlyOnce_EffectDoesNotRerunAfterACrash is the criterion.
func TestExactlyOnce_EffectDoesNotRerunAfterACrash(parseT *testing.T) {
	parseStore := NewMemoryTransactionalCheckpointStore()
	parseEffect := &countingEffect{}
	const parseID CommandID = "charge-card-42"

	parseFirst := NewRuntime(parseStore)
	if !parseFirst.ExactlyOnce() {
		parseT.Fatal("a transactional store must report an exactly-once guarantee")
	}
	parseOutcome, parseErr := parseFirst.Execute(parseID, parseEffect.apply)
	if parseErr != nil {
		parseT.Fatalf("first execute: %v", parseErr)
	}
	if parseOutcome != OutcomeApplied {
		parseT.Fatalf("first outcome = %v, want %v", parseOutcome, OutcomeApplied)
	}
	if parseEffect.runs != 1 {
		parseT.Fatalf("effect ran %d times on first execute, want 1", parseEffect.runs)
	}

	// The crash. A new Runtime over the same store has no in-process ledger,
	// which is precisely the state a restart leaves behind.
	parseAfterRestart := NewRuntime(parseStore)
	parseOutcome, parseErr = parseAfterRestart.Execute(parseID, parseEffect.apply)
	if parseErr != nil {
		parseT.Fatalf("execute after restart: %v", parseErr)
	}

	if parseEffect.runs != 1 {
		parseT.Errorf("effect ran %d times across a restart, want 1 — a card would have been charged twice",
			parseEffect.runs)
	}
	if parseOutcome != OutcomeReplayed {
		parseT.Errorf("outcome after restart = %v, want %v", parseOutcome, OutcomeReplayed)
	}
}

// TestExactlyOnce_WithoutATransactionalStoreTheEffectRerunsAndSaysSo pins the
// weaker guarantee, so the difference between the two stores is visible rather
// than implied.
//
// This is not a bug being tested — it is the documented bound of the
// non-transactional path, and a test is the only place a bound like that stays
// honest as the code changes.
func TestExactlyOnce_WithoutATransactionalStoreTheEffectRerunsAndSaysSo(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	parseEffect := &countingEffect{}
	const parseID CommandID = "charge-card-43"

	parseFirst := NewRuntime(parseStore)
	if parseFirst.ExactlyOnce() {
		parseT.Fatal("a plain store must NOT claim an exactly-once guarantee")
	}
	if _, parseErr := parseFirst.Execute(parseID, parseEffect.apply); parseErr != nil {
		parseT.Fatalf("first execute: %v", parseErr)
	}

	parseAfterRestart := NewRuntime(parseStore)
	if _, parseErr := parseAfterRestart.Execute(parseID, parseEffect.apply); parseErr != nil {
		parseT.Fatalf("execute after restart: %v", parseErr)
	}

	if parseEffect.runs != 2 {
		parseT.Errorf("effect ran %d times, want 2 — this test exists to pin the WEAKER guarantee, and if it no longer reruns then ExactlyOnce is reporting the wrong thing",
			parseEffect.runs)
	}
}

// TestExactlyOnce_FailedEffectIsNotRecorded keeps the fix from over-reaching.
//
// A transaction that rolls back must leave the command unrecorded, or a retry
// under the same id would be refused as a replay and the work would be silently
// dropped — trading duplicate execution for lost execution.
func TestExactlyOnce_FailedEffectIsNotRecorded(parseT *testing.T) {
	parseStore := NewMemoryTransactionalCheckpointStore()
	const parseID CommandID = "charge-card-44"

	parseAttempts := 0
	parseFlaky := func() error {
		parseAttempts++
		if parseAttempts == 1 {
			return errors.New("network refused")
		}
		return nil
	}

	parseRuntime := NewRuntime(parseStore)
	if parseOutcome, parseErr := parseRuntime.Execute(parseID, parseFlaky); parseErr == nil {
		parseT.Fatalf("a failing effect must surface as an error, got outcome %v", parseOutcome)
	}
	if parseStore.AppliedCount() != 0 {
		parseT.Fatalf("a rolled-back command was recorded as applied (%d records)", parseStore.AppliedCount())
	}

	// The retry must run, because nothing committed.
	parseOutcome, parseErr := parseRuntime.Execute(parseID, parseFlaky)
	if parseErr != nil {
		parseT.Fatalf("retry: %v", parseErr)
	}
	if parseOutcome != OutcomeApplied {
		parseT.Errorf("retry outcome = %v, want %v; a rolled-back command must be retryable",
			parseOutcome, OutcomeApplied)
	}
	if parseAttempts != 2 {
		parseT.Errorf("effect attempted %d times, want 2", parseAttempts)
	}
}

// TestExactlyOnce_CancelStillWins keeps the transactional path inside the rest
// of P3.4: a cancelled command neither replays nor resumes, and routing through
// a store must not smuggle it past that.
func TestExactlyOnce_CancelStillWins(parseT *testing.T) {
	parseStore := NewMemoryTransactionalCheckpointStore()
	parseEffect := &countingEffect{}
	const parseID CommandID = "charge-card-45"

	parseRuntime := NewRuntime(parseStore)
	if parseErr := parseRuntime.Cancel(parseID); parseErr != nil {
		parseT.Fatalf("cancel: %v", parseErr)
	}

	parseOutcome, parseErr := parseRuntime.Execute(parseID, parseEffect.apply)
	if parseErr != nil {
		parseT.Fatalf("execute after cancel: %v", parseErr)
	}
	if parseOutcome != OutcomeCancelled {
		parseT.Errorf("outcome = %v, want %v", parseOutcome, OutcomeCancelled)
	}
	if parseEffect.runs != 0 {
		parseT.Errorf("a cancelled command ran its effect %d times", parseEffect.runs)
	}
}
