package domain

import (
	"errors"
	"testing"
)

// v5 P3.4 — atomic command guarantees.
//
// Criterion (a): a replayed command produces no duplicate effects.
// Criterion (c), atomic half: a cancelled command does not run.

func TestExecuteRunsEffectsOnce(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseEffects := 0

	parseOutcome, parseErr := parseRuntime.Execute("cmd-1", func() error {
		parseEffects++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("Execute: %v", parseErr)
	}
	if !parseOutcome.Applied() {
		parseT.Errorf("outcome = %s, want applied", parseOutcome)
	}
	if parseEffects != 1 {
		parseT.Errorf("effects ran %d times, want 1", parseEffects)
	}
}

// TestExecuteReplayProducesNoDuplicateEffects is P3.4 criterion (a).
func TestExecuteReplayProducesNoDuplicateEffects(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseEffects := 0
	parseEffect := func() error {
		parseEffects++
		return nil
	}

	for parseAttempt := range 10 {
		parseOutcome, parseErr := parseRuntime.Execute("cmd-1", parseEffect)
		if parseErr != nil {
			parseT.Fatalf("attempt %d: %v", parseAttempt, parseErr)
		}
		if parseAttempt == 0 && !parseOutcome.Applied() {
			parseT.Fatalf("first attempt: outcome = %s, want applied", parseOutcome)
		}
		if parseAttempt > 0 && parseOutcome != OutcomeReplayed {
			parseT.Fatalf("attempt %d: outcome = %s, want replayed", parseAttempt, parseOutcome)
		}
	}

	if parseEffects != 1 {
		parseT.Errorf("effects ran %d times across 10 executions, want 1", parseEffects)
	}
}

// TestFailedCommandCanBeRetried is the check/commit split earning its keep. If
// the command were recorded before its effects succeeded, this retry would look
// like a replay and the work would be silently dropped.
func TestFailedCommandCanBeRetried(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseAttempts := 0

	parseEffect := func() error {
		parseAttempts++
		if parseAttempts < 3 {
			return errors.New("transient failure")
		}
		return nil
	}

	for parseAttempt := range 3 {
		parseOutcome, parseErr := parseRuntime.Execute("cmd-retry", parseEffect)
		if parseAttempt < 2 {
			if parseErr == nil {
				parseT.Fatalf("attempt %d: expected an error", parseAttempt)
			}
			if parseOutcome != OutcomeFailed {
				parseT.Fatalf("attempt %d: outcome = %s, want failed", parseAttempt, parseOutcome)
			}
			continue
		}
		if parseErr != nil {
			parseT.Fatalf("final attempt: %v", parseErr)
		}
		if !parseOutcome.Applied() {
			parseT.Fatalf("final attempt: outcome = %s, want applied", parseOutcome)
		}
	}

	if parseAttempts != 3 {
		parseT.Errorf("effect ran %d times, want 3", parseAttempts)
	}

	// And once it has succeeded, it is a replay like any other.
	parseOutcome, _ := parseRuntime.Execute("cmd-retry", parseEffect)
	if parseOutcome != OutcomeReplayed {
		parseT.Errorf("outcome = %s, want replayed after success", parseOutcome)
	}
	if parseAttempts != 3 {
		parseT.Errorf("effect ran %d times after the replay, want 3", parseAttempts)
	}
}

// TestPanicInEffectIsContained: a domain panic must not take down the worker,
// and must leave the command retryable.
func TestPanicInEffectIsContained(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseAttempts := 0

	parseOutcome, parseErr := parseRuntime.Execute("cmd-panic", func() error {
		parseAttempts++
		panic("domain blew up")
	})
	if parseErr == nil {
		parseT.Fatal("a panic must surface as an error")
	}
	if parseOutcome != OutcomeFailed {
		parseT.Errorf("outcome = %s, want failed", parseOutcome)
	}

	// Retryable: the panic did not record the command as applied.
	if _, parseRetryErr := parseRuntime.Execute("cmd-panic", func() error {
		parseAttempts++
		return nil
	}); parseRetryErr != nil {
		parseT.Fatalf("retry after panic: %v", parseRetryErr)
	}
	if parseAttempts != 2 {
		parseT.Errorf("effect ran %d times, want 2 — the panic wedged the command", parseAttempts)
	}
}

// TestCancelledCommandDoesNotRun is P3.4 criterion (c) for atomic commands.
func TestCancelledCommandDoesNotRun(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseEffects := 0

	if parseErr := parseRuntime.Cancel("cmd-cancelled"); parseErr != nil {
		parseT.Fatalf("Cancel: %v", parseErr)
	}

	parseOutcome, parseErr := parseRuntime.Execute("cmd-cancelled", func() error {
		parseEffects++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("Execute: %v", parseErr)
	}
	if parseOutcome != OutcomeCancelled {
		parseT.Errorf("outcome = %s, want cancelled", parseOutcome)
	}
	if parseEffects != 0 {
		parseT.Errorf("effects ran %d times on a cancelled command, want 0", parseEffects)
	}
}

// TestCancelDoesNotUndoAnAppliedCommand: undo is a compensating command, not a
// cancellation, and conflating them would promise a rollback this cannot give.
func TestCancelDoesNotUndoAnAppliedCommand(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseEffects := 0

	if _, parseErr := parseRuntime.Execute("cmd", func() error {
		parseEffects++
		return nil
	}); parseErr != nil {
		parseT.Fatalf("Execute: %v", parseErr)
	}
	if parseErr := parseRuntime.Cancel("cmd"); parseErr != nil {
		parseT.Fatalf("Cancel: %v", parseErr)
	}

	if parseEffects != 1 {
		parseT.Errorf("effects = %d — cancel must not roll back applied work", parseEffects)
	}
}

func TestCommandsAreIndependent(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseEffects := map[CommandID]int{}

	for _, parseCommandID := range []CommandID{"a", "b", "c"} {
		if _, parseErr := parseRuntime.Execute(parseCommandID, func() error {
			parseEffects[parseCommandID]++
			return nil
		}); parseErr != nil {
			parseT.Fatalf("Execute(%s): %v", parseCommandID, parseErr)
		}
	}

	if len(parseEffects) != 3 {
		parseT.Errorf("ran %d distinct commands, want 3 — ids collided", len(parseEffects))
	}
	if parseRuntime.TrackedCommands() != 3 {
		parseT.Errorf("tracked %d commands, want 3", parseRuntime.TrackedCommands())
	}
}

// TestForgetReleasesReplayState covers the growth documented on Forget: replay
// detection requires remembering, so a caller with unbounded ids must be able to
// release them.
func TestForgetReleasesReplayState(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseEffects := 0
	parseEffect := func() error {
		parseEffects++
		return nil
	}

	if _, parseErr := parseRuntime.Execute("cmd", parseEffect); parseErr != nil {
		parseT.Fatalf("Execute: %v", parseErr)
	}
	if parseErr := parseRuntime.Forget("cmd"); parseErr != nil {
		parseT.Fatalf("Forget: %v", parseErr)
	}
	if parseRuntime.TrackedCommands() != 0 {
		parseT.Errorf("tracked %d commands after Forget, want 0", parseRuntime.TrackedCommands())
	}

	// Documented consequence: a forgotten id runs again.
	parseOutcome, _ := parseRuntime.Execute("cmd", parseEffect)
	if !parseOutcome.Applied() || parseEffects != 2 {
		parseT.Errorf("outcome = %s, effects = %d — Forget clears replay state by design", parseOutcome, parseEffects)
	}
}

func TestExecuteRejectsBadInput(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	if _, parseErr := parseRuntime.Execute("", func() error { return nil }); parseErr == nil {
		parseT.Error("an empty command id must be rejected")
	}
	if _, parseErr := parseRuntime.Execute("cmd", nil); parseErr == nil {
		parseT.Error("a nil effect must be rejected")
	}
	if parseErr := parseRuntime.Cancel(""); parseErr == nil {
		parseT.Error("cancelling an empty id must be rejected")
	}

	var parseNil *Runtime
	if _, parseErr := parseNil.Execute("cmd", func() error { return nil }); parseErr == nil {
		parseT.Error("a nil runtime must error rather than panic")
	}
}

func TestOutcomeLabelsAreDistinct(parseT *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseOutcome := range []Outcome{OutcomeApplied, OutcomeReplayed, OutcomeCancelled, OutcomeFailed} {
		if parseSeen[parseOutcome.String()] {
			parseT.Errorf("outcome label %q is not distinct", parseOutcome)
		}
		parseSeen[parseOutcome.String()] = true
	}
	if Outcome(200).String() == "" {
		parseT.Error("an unknown outcome must still render for diagnostics")
	}
}
