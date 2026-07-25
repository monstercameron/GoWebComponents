// Package domain is v5's off-thread command runtime (plan item P3.4).
//
// It executes commands away from the render thread with three guarantees the
// plan names as its acceptance criteria: an atomic command that is replayed
// produces no duplicate effects, a bulk command that dies partway resumes and
// finishes exactly once, and a cancelled command does neither.
//
// All three are ordering problems, which is why this sits directly on the
// services extracted in P3.3 rather than reinventing them: the ledger decides
// replay, the dispatcher's generation model decides cancellation.
//
// The package is deliberately free of syscall/js so the guarantees can be tested
// natively. Nothing here knows it will run in a worker.
package domain

import (
	"fmt"
	"maps"
	"sync"
)

// CommandID identifies one logical command.
//
// It is the replay key, so it must be stable across retries of the SAME logical
// command and distinct between different ones. A caller that generates a fresh
// ID per attempt gets no replay protection — which is the single most likely way
// to misuse this package, hence the emphasis.
type CommandID string

// Checkpoint is a bulk command's durable progress marker.
type Checkpoint struct {
	CommandID CommandID
	// NextIndex is the first item not yet known to be complete. Resuming starts
	// here, so it is deliberately conservative: an item may be re-executed, but
	// none is ever skipped.
	NextIndex int
	// Total is the item count the checkpoint was written against. A resume
	// against a different total is refused rather than guessed at.
	Total int
	// Exact reports that NextIndex is precisely where the previous run stopped,
	// rather than the last interval boundary it happened to cross.
	//
	// It is true when a run ended gracefully — a step returned an error — because
	// the runtime then knows exactly what completed and writes it. It is false
	// for interval checkpoints, where a run killed outright may have progressed
	// further than the record shows. The distinction is the difference between
	// resuming with no re-execution and resuming with a bounded window of it, so
	// it is recorded rather than assumed either way.
	Exact bool
}

// CheckpointStore persists bulk progress across a crash.
//
// This interface is the durability seam, and the strength of the exactly-once
// guarantee depends entirely on what implements it:
//
//   - Backed by the SAME transactional store as the command's effects (P3.5's
//     SQLite), progress and effect commit together and exactly-once holds for
//     any checkpoint interval.
//   - Backed by anything else, progress and effects can diverge by up to one
//     checkpoint interval, so items between the last checkpoint and the crash
//     are re-executed on resume. The runtime reports that count rather than
//     hiding it; see BulkResult.ReplayedItems.
//
// §11-Q7 — whether WAL-mode crash atomicity holds over OPFS sync access handles
// — is exactly the question of whether the first bullet is available in the
// browser. It is unresolved, which is why the second bullet is a supported mode
// and not a degraded one.
type CheckpointStore interface {
	Load(parseCommandID CommandID) (Checkpoint, bool, error)
	Save(parseCheckpoint Checkpoint) error
	Clear(parseCommandID CommandID) error
}

// MemoryCheckpointStore is an in-process CheckpointStore.
//
// It survives a command failure but not a process restart, which makes it the
// right store for tests and for bulk work whose cost is cheaper to redo than to
// persist. It is NOT the browser durability story; that is P3.6.
type MemoryCheckpointStore struct {
	mutex               sync.Mutex
	checkpointByCommand map[CommandID]Checkpoint
}

// NewMemoryCheckpointStore creates an empty in-process checkpoint store.
func NewMemoryCheckpointStore() *MemoryCheckpointStore {
	return &MemoryCheckpointStore{checkpointByCommand: make(map[CommandID]Checkpoint)}
}

// Load reports a command's checkpoint, if one exists.
func (parseStore *MemoryCheckpointStore) Load(parseCommandID CommandID) (Checkpoint, bool, error) {
	if parseStore == nil {
		return Checkpoint{}, false, fmt.Errorf("domain: checkpoint store is nil")
	}
	parseStore.mutex.Lock()
	defer parseStore.mutex.Unlock()

	parseCheckpoint, hasCheckpoint := parseStore.checkpointByCommand[parseCommandID]
	return parseCheckpoint, hasCheckpoint, nil
}

// Save writes a command's checkpoint.
func (parseStore *MemoryCheckpointStore) Save(parseCheckpoint Checkpoint) error {
	if parseStore == nil {
		return fmt.Errorf("domain: checkpoint store is nil")
	}
	if parseCheckpoint.CommandID == "" {
		return fmt.Errorf("domain: checkpoint command id is required")
	}
	parseStore.mutex.Lock()
	defer parseStore.mutex.Unlock()

	// Progress must not move backwards. A late write from a superseded run would
	// otherwise rewind a resumed run and re-execute work it had already passed.
	if parseExisting, hasExisting := parseStore.checkpointByCommand[parseCheckpoint.CommandID]; hasExisting {
		if parseCheckpoint.NextIndex < parseExisting.NextIndex {
			return nil
		}
	}
	parseStore.checkpointByCommand[parseCheckpoint.CommandID] = parseCheckpoint
	return nil
}

// Clear removes a command's checkpoint.
func (parseStore *MemoryCheckpointStore) Clear(parseCommandID CommandID) error {
	if parseStore == nil {
		return fmt.Errorf("domain: checkpoint store is nil")
	}
	parseStore.mutex.Lock()
	defer parseStore.mutex.Unlock()

	delete(parseStore.checkpointByCommand, parseCommandID)
	return nil
}

// Snapshot copies every stored checkpoint, for diagnostics and tests.
func (parseStore *MemoryCheckpointStore) Snapshot() map[CommandID]Checkpoint {
	if parseStore == nil {
		return map[CommandID]Checkpoint{}
	}
	parseStore.mutex.Lock()
	defer parseStore.mutex.Unlock()

	parseCopy := make(map[CommandID]Checkpoint, len(parseStore.checkpointByCommand))
	maps.Copy(parseCopy, parseStore.checkpointByCommand)
	return parseCopy
}
