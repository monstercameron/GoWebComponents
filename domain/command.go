package domain

import (
	"errors"
	"fmt"

	"github.com/monstercameron/GoWebComponents/v5/internal/services"
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
	// yield gives the host event loop a turn during a long bulk run. See
	// SetYield — without it, a bulk command occupies its thread from start to
	// finish and no cancel message can reach it.
	yield func()
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

// SetYield installs the function ExecuteBulk uses to give the host event loop a
// turn between chunks.
//
// This exists because of where bulk commands actually run. A domain worker is a
// single-threaded message loop: it receives a command, runs it, and only then
// takes the next message. A 50,000-item bulk run that never returns to that loop
// cannot be cancelled by a message, because the cancel message is sitting in a
// queue the worker will not read until the run it is meant to stop has finished.
// Checking a flag on every item does not help when nothing can set the flag.
//
// The yielder is the environment's, not the domain's: in a wasm worker it is a
// setTimeout(0) round trip; in a test it can be a no-op or a channel receive.
// The domain layer must not know which.
//
//	parseRuntime.SetYield(func() {
//	    parseDone := make(chan struct{})
//	    var parseCallback js.Func
//	    parseCallback = js.FuncOf(func(js.Value, []js.Value) any {
//	        parseCallback.Release()
//	        close(parseDone)
//	        return nil
//	    })
//	    js.Global().Call("setTimeout", parseCallback, 0)
//	    <-parseDone
//	})
func (parseRuntime *Runtime) SetYield(parseYield func()) {
	if parseRuntime != nil {
		parseRuntime.yield = parseYield
	}
}

// Execute runs an atomic command exactly once.
//
// The check/commit split is what makes this safe under failure. The command is
// recorded as applied only AFTER its effects succeed, so a command that fails
// partway can be retried under the same id — where recording it up front would
// make the retry look like a replay and silently drop the work.
//
// The converse is the cost: if the process dies between the effects succeeding
// and the record being written, the retry re-runs the effects. Reversing the
// order moves that window rather than removing it — a crash after the record and
// before the effect would drop the work instead, which is worse.
//
// It closes only when the effect and the record commit TOGETHER. A store that
// implements TransactionalCheckpointStore can arrange that, and Execute uses it
// when present; see transactional.go. Runtime.ExactlyOnce reports which
// guarantee an application is actually running under.
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

	// A store that can host the effect inside its own transaction closes the
	// crash window described above; one that cannot falls through to the
	// check/commit path below, which still refuses duplicates for every failure
	// short of a crash in that one window. Runtime.ExactlyOnce reports which is
	// in force.
	if parseOutcome, parseErr, isHandled := parseRuntime.executeTransactionally(parseCommandID, parseEffect); isHandled {
		return parseOutcome, parseErr
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

// ------------------------------------------------------------ hot reload

// RuntimeState is a command runtime's replay state, extracted for transfer
// across a worker reload (plan item P3.14).
//
// Only what a reload must not lose. Checkpoints are excluded on purpose: they
// already live in the CheckpointStore, which is the durable side and survives
// the reload by construction. Carrying them here would create a second copy that
// can disagree with the first.
type RuntimeState struct {
	// Applied lists commands already applied, so a retry after the reload is
	// still recognized as a replay rather than run a second time.
	//
	// This is the guarantee that would silently break: a reloaded worker with a
	// fresh ledger treats every in-flight retry as new work, so a command that
	// succeeded just before the edit runs again just after it.
	Applied []CommandID `json:"applied,omitempty"`
	// Cancelled lists commands that must stay refused.
	Cancelled []CommandID `json:"cancelled,omitempty"`
}

// ExportState captures the replay state a reload must preserve.
func (parseRuntime *Runtime) ExportState() RuntimeState {
	if parseRuntime == nil {
		return RuntimeState{}
	}

	parseState := RuntimeState{}
	for _, parseStreamID := range parseRuntime.ledger.Streams() {
		parseState.Applied = append(parseState.Applied, CommandID(parseStreamID))
	}
	for parseCommandID, isCancelled := range parseRuntime.cancelled {
		if isCancelled {
			parseState.Cancelled = append(parseState.Cancelled, parseCommandID)
		}
	}
	return parseState
}

// RestoreRuntime rebuilds a command runtime from exported state.
//
// The checkpoint store is supplied separately because it is durable and was
// never part of the export.
func RestoreRuntime(parseCheckpoints CheckpointStore, parseState RuntimeState) (*Runtime, error) {
	parseRuntime := NewRuntime(parseCheckpoints)

	for _, parseCommandID := range parseState.Applied {
		if parseCommandID == "" {
			return nil, errors.New("domain: restored state contains an empty command id")
		}
		if parseErr := parseRuntime.ledger.Commit(
			string(parseCommandID), commandEpoch, commandVersion, string(parseCommandID)); parseErr != nil {
			return nil, fmt.Errorf("domain: restoring command %q: %w", parseCommandID, parseErr)
		}
	}
	for _, parseCommandID := range parseState.Cancelled {
		if parseCommandID == "" {
			return nil, errors.New("domain: restored state contains an empty cancelled command id")
		}
		parseRuntime.cancelled[parseCommandID] = true
	}
	return parseRuntime, nil
}
