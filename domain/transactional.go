package domain

import (
	"errors"
	"fmt"
	"sync"
)

// Exactly-once under a crash (P3.4 criterion (a), §11-Q7).
//
// Execute records a command as applied only AFTER its effect succeeds. That
// ordering is deliberate and mostly right: recording it first would make a retry
// of a partially-failed command look like a replay and silently drop the work.
//
// It leaves one window that no ordering can close. If the process dies between
// the effect succeeding and the record being written, the next run sees no
// record and runs the effect again. Reversing the order moves the window rather
// than removing it — then a crash after the record and before the effect drops
// the work instead, which is worse.
//
// The window closes only when the effect and the record commit TOGETHER, which
// the domain layer cannot arrange on its own: it does not own the storage the
// effect writes to. What it can do is ask, and use the answer when there is one.
//
// A store that can host the effect inside its own transaction implements
// TransactionalCheckpointStore. Execute then hands the effect to the store,
// which applies it and records the command atomically. A store that cannot keeps
// today's behaviour, and Runtime.ExactlyOnce reports which guarantee is in force
// so an application can find out without reading this file.

// TransactionalCheckpointStore commits a command's effect and its
// applied-record in one transaction.
//
// The contract is the whole point and is stricter than it looks:
//
//   - If ApplyExactlyOnce returns nil, the effect ran and the command is
//     recorded as applied. Both, or neither — a partial outcome is a violation.
//   - If it returns ErrAlreadyApplied, the effect did NOT run in this call and
//     the command was already recorded. That is a replay, not a failure.
//   - If it returns any other error, the effect did not commit and the command
//     is not recorded, so a retry under the same id is safe.
//
// The effect must do its work through the transaction the store provides.
// A store cannot make an effect atomic if the effect writes somewhere else, and
// implementing this interface while allowing that would promise a guarantee the
// implementation cannot keep.
type TransactionalCheckpointStore interface {
	CheckpointStore
	ApplyExactlyOnce(parseCommandID CommandID, parseEffect func() error) error
}

// ErrAlreadyApplied reports that a command was already recorded as applied, so
// its effect did not run again.
//
// A sentinel rather than a boolean return, because an implementation that
// discovers this mid-transaction usually has an error value in hand and nothing
// useful to say in a second return.
var ErrAlreadyApplied = errors.New("domain: command already applied")

// ExactlyOnce reports whether this runtime can guarantee exactly-once execution
// across a crash.
//
// False does not mean commands are unsafe — replay protection still holds for
// every failure short of a crash in one specific window. It means the window
// between an effect succeeding and its record being written is open, and an
// application whose effects are not idempotent should either supply a
// transactional store or accept that bound. Reporting it is what lets that be a
// decision rather than a surprise.
func (parseRuntime *Runtime) ExactlyOnce() bool {
	if parseRuntime == nil || parseRuntime.checkpoints == nil {
		return false
	}
	_, isTransactional := parseRuntime.checkpoints.(TransactionalCheckpointStore)
	return isTransactional
}

// executeTransactionally runs one command through a transactional store.
//
// Returns handled=false when the store cannot do this, so Execute falls through
// to the check/commit path rather than silently degrading to a weaker guarantee
// under the same name.
func (parseRuntime *Runtime) executeTransactionally(parseCommandID CommandID, parseEffect func() error) (Outcome, error, bool) {
	parseStore, isTransactional := parseRuntime.checkpoints.(TransactionalCheckpointStore)
	if !isTransactional {
		return OutcomeFailed, nil, false
	}

	// The in-memory ledger is still consulted FIRST. It is the faster check and
	// it is authoritative for this process; the store's own record is what
	// survives a restart. Skipping it would make every replay pay a storage
	// round trip to learn something already known.
	parseDecision, parseCheckErr := parseRuntime.ledger.Check(
		string(parseCommandID), commandEpoch, commandVersion, string(parseCommandID))
	if parseCheckErr != nil {
		return OutcomeFailed, fmt.Errorf("domain: command %q: %w", parseCommandID, parseCheckErr), true
	}
	if !parseDecision.ShouldApply() {
		return OutcomeReplayed, nil, true
	}

	parseApplyErr := parseStore.ApplyExactlyOnce(parseCommandID, func() error {
		return runContained(parseEffect)
	})
	switch {
	case errors.Is(parseApplyErr, ErrAlreadyApplied):
		// Recorded by an earlier process. The effect did not run again, which is
		// the guarantee this path exists to provide.
		parseRuntime.commitLedgerBestEffort(parseCommandID)
		return OutcomeReplayed, nil, true
	case parseApplyErr != nil:
		return OutcomeFailed, fmt.Errorf("domain: command %q: %w", parseCommandID, parseApplyErr), true
	}

	// The durable record is already written; the in-memory ledger is a cache of
	// it. A failure here cannot un-apply the command, so it must not be reported
	// as a failed command — that would invite a retry of work that has committed.
	parseRuntime.commitLedgerBestEffort(parseCommandID)
	return OutcomeApplied, nil, true
}

// commitLedgerBestEffort records a command in the in-process ledger, ignoring an
// error the caller cannot act on.
//
// Deliberately silent. This runs after the durable record exists, so a ledger
// failure costs a redundant replay check later and nothing else; surfacing it as
// a command failure would tell the caller to retry work that has committed.
func (parseRuntime *Runtime) commitLedgerBestEffort(parseCommandID CommandID) {
	_ = parseRuntime.ledger.Commit(
		string(parseCommandID), commandEpoch, commandVersion, string(parseCommandID))
}

// MemoryTransactionalCheckpointStore is an in-process store that applies an
// effect and its record together.
//
// It exists so the transactional path is testable and so an application can see
// what the interface asks for. It is NOT durable — a process restart loses
// everything — so it demonstrates atomicity, not persistence. A real
// implementation is the SQLite store writing the applied-record inside the same
// transaction as the effect's own writes.
type MemoryTransactionalCheckpointStore struct {
	*MemoryCheckpointStore
	mutex     sync.Mutex
	appliedID map[CommandID]bool
}

// NewMemoryTransactionalCheckpointStore creates an in-process transactional store.
func NewMemoryTransactionalCheckpointStore() *MemoryTransactionalCheckpointStore {
	return &MemoryTransactionalCheckpointStore{
		MemoryCheckpointStore: NewMemoryCheckpointStore(),
		appliedID:             make(map[CommandID]bool),
	}
}

// ApplyExactlyOnce runs the effect and records the command under one lock.
//
// The lock is this store's stand-in for a transaction: nothing else can observe
// the state between the effect running and the record being written, which is
// the property a database transaction provides and the reason the window this
// closes exists at all.
func (parseStore *MemoryTransactionalCheckpointStore) ApplyExactlyOnce(parseCommandID CommandID, parseEffect func() error) error {
	if parseCommandID == "" {
		return errors.New("domain: command id is required")
	}
	if parseEffect == nil {
		return errors.New("domain: command effect is required")
	}

	parseStore.mutex.Lock()
	defer parseStore.mutex.Unlock()

	if parseStore.appliedID[parseCommandID] {
		return ErrAlreadyApplied
	}
	if parseEffectErr := parseEffect(); parseEffectErr != nil {
		// Not recorded: the transaction rolls back, so a retry under the same id
		// runs the effect again, which is correct because it did not commit.
		return parseEffectErr
	}
	parseStore.appliedID[parseCommandID] = true
	return nil
}

// AppliedCount reports how many distinct commands are recorded, so a test can
// assert the record survived independently of the in-memory ledger.
func (parseStore *MemoryTransactionalCheckpointStore) AppliedCount() int {
	parseStore.mutex.Lock()
	defer parseStore.mutex.Unlock()
	return len(parseStore.appliedID)
}
