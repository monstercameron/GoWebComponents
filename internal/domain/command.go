package domain

import (
	"errors"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/internal/services"
)

// Outcome is what happened to a command.
type Outcome uint8

const (
	// OutcomeApplied means the command's effects ran, for the first and only time.
	OutcomeApplied Outcome = iota
	// OutcomeReplayed means the command had already been applied, so its effects
	// were deliberately NOT run again. This is a success, not a failure.
	OutcomeReplayed
	// OutcomeCancelled means the command was cancelled before its effects ran.
	OutcomeCancelled
	// OutcomeFailed means the effects ran and returned an error. The command is
	// not recorded as applied, so a retry with the same id will run again.
	OutcomeFailed
)

func (parseOutcome Outcome) String() string {
	switch parseOutcome {
	case OutcomeApplied:
		return "applied"
	case OutcomeReplayed:
		return "replayed"
	case OutcomeCancelled:
		return "cancelled"
	case OutcomeFailed:
		return "failed"
	default:
		return fmt.Sprintf("unknown-outcome(%d)", uint8(parseOutcome))
	}
}

// Applied reports whether effects ran on this call. Callers should branch on
// this rather than comparing against OutcomeApplied, so a future outcome cannot
// silently be treated as "effects ran".
func (parseOutcome Outcome) Applied() bool {
	return parseOutcome == OutcomeApplied
}

// ErrCommandCancelled is returned when work stops because of a cancellation.
var ErrCommandCancelled = errors.New("domain: command was cancelled")

// commandEpoch is fixed at 1.
//
// The ledger models epochs for streams that restart; a command id is unique for
// the life of a command, so it has exactly one epoch by construction. Fixing it
// here rather than exposing it keeps callers from inventing a meaning for it.
const commandEpoch = uint64(1)

// commandVersion is fixed at 1 for the same reason: one command id carries one
// message. Bulk commands use their own progress model rather than versions.
const commandVersion = uint64(1)

// Runtime executes commands with replay, resume, and cancellation guarantees.
//
// Not safe for concurrent use. It is designed for one worker's message loop,
// which is how it will run; the mutex in MemoryCheckpointStore protects the
// store alone, not this.
type Runtime struct {
	ledger      *services.Ledger
	checkpoints CheckpointStore
	cancelled   map[CommandID]bool
}

// NewRuntime creates a command runtime over a checkpoint store.
//
// A nil store is allowed and disables bulk resume: bulk commands still run and
// still refuse to duplicate their effects, they simply restart from zero after a
// failure. That is a legitimate configuration for cheap work, so it is a
// documented mode rather than an error at construction.
func NewRuntime(parseCheckpoints CheckpointStore) *Runtime {
	return &Runtime{
		ledger:      services.NewLedger(),
		checkpoints: parseCheckpoints,
		cancelled:   make(map[CommandID]bool),
	}
}

// Execute runs an atomic command exactly once.
//
// The check/commit split is what makes this safe under failure. The command is
// recorded as applied only AFTER its effects succeed, so a command that fails
// partway can be retried under the same id — where recording it up front would
// make the retry look like a replay and silently drop the work.
//
// The converse is the cost, and it is inherent rather than an oversight: if the
// process dies between the effects succeeding and the record being written, the
// retry re-runs the effects. Closing that gap requires the effects and the
// record to commit together, which is what a transactional CheckpointStore
// gives and what §11-Q7 is about.
func (parseRuntime *Runtime) Execute(parseCommandID CommandID, parseEffect func() error) (Outcome, error) {
	if parseRuntime == nil {
		return OutcomeFailed, errors.New("domain: runtime is nil")
	}
	if parseCommandID == "" {
		return OutcomeFailed, errors.New("domain: command id is required")
	}
	if parseEffect == nil {
		return OutcomeFailed, errors.New("domain: command effect is required")
	}
	if parseRuntime.cancelled[parseCommandID] {
		return OutcomeCancelled, nil
	}

	parseDecision, parseCheckErr := parseRuntime.ledger.Check(
		string(parseCommandID), commandEpoch, commandVersion, string(parseCommandID))
	if parseCheckErr != nil {
		return OutcomeFailed, fmt.Errorf("domain: command %q: %w", parseCommandID, parseCheckErr)
	}
	if !parseDecision.ShouldApply() {
		return OutcomeReplayed, nil
	}

	if parseEffectErr := runContained(parseEffect); parseEffectErr != nil {
		return OutcomeFailed, fmt.Errorf("domain: command %q: %w", parseCommandID, parseEffectErr)
	}

	if parseCommitErr := parseRuntime.ledger.Commit(
		string(parseCommandID), commandEpoch, commandVersion, string(parseCommandID)); parseCommitErr != nil {
		return OutcomeFailed, fmt.Errorf("domain: command %q: %w", parseCommandID, parseCommitErr)
	}
	return OutcomeApplied, nil
}

// Cancel stops a command from running, resuming, or being retried.
//
// Cancellation is permanent for the id, and it clears any bulk progress. Both
// follow from what a cancel means: the work is not wanted. Leaving the
// checkpoint would let a later call with the same id silently resume abandoned
// work, which is criterion (c) of P3.4 — a cancelled command neither replays
// nor resumes.
//
// Cancelling a command that already applied does not undo it. Undo is a
// compensating command, not a cancellation, and conflating the two would promise
// a rollback this cannot deliver.
func (parseRuntime *Runtime) Cancel(parseCommandID CommandID) error {
	if parseRuntime == nil {
		return errors.New("domain: runtime is nil")
	}
	if parseCommandID == "" {
		return errors.New("domain: command id is required")
	}

	parseRuntime.cancelled[parseCommandID] = true
	if parseRuntime.checkpoints != nil {
		if parseClearErr := parseRuntime.checkpoints.Clear(parseCommandID); parseClearErr != nil {
			return fmt.Errorf("domain: cancelling command %q: %w", parseCommandID, parseClearErr)
		}
	}
	return nil
}

// IsCancelled reports whether a command has been cancelled.
func (parseRuntime *Runtime) IsCancelled(parseCommandID CommandID) bool {
	if parseRuntime == nil {
		return false
	}
	return parseRuntime.cancelled[parseCommandID]
}

// Forget drops all record of a command.
//
// Replay detection requires remembering, so records accumulate for the life of
// the runtime. A caller that generates unbounded command ids — one per user
// action over a long session — should forget ids it will never legitimately
// retry. Forgetting an id makes a later command with that id run again.
func (parseRuntime *Runtime) Forget(parseCommandID CommandID) error {
	if parseRuntime == nil {
		return errors.New("domain: runtime is nil")
	}
	parseRuntime.ledger.Forget(string(parseCommandID))
	delete(parseRuntime.cancelled, parseCommandID)
	if parseRuntime.checkpoints != nil {
		return parseRuntime.checkpoints.Clear(parseCommandID)
	}
	return nil
}

// TrackedCommands reports how many commands hold replay state, so a caller can
// see the growth described on Forget rather than discover it as a leak.
func (parseRuntime *Runtime) TrackedCommands() int {
	if parseRuntime == nil {
		return 0
	}
	return parseRuntime.ledger.TrackedStreams()
}

// runContained runs an effect, converting a panic into an error.
//
// A domain panic must not take down the worker: the plan's P3.8a requires a
// panic to leave recoverable UI state, and a dead worker leaves none. Containing
// it here also means the command is not recorded as applied, so it can be
// retried once the cause is fixed.
func runContained(parseEffect func() error) (parseErr error) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseErr = fmt.Errorf("panic in command effect: %v", parseRecovered)
		}
	}()
	return parseEffect()
}
