package domain

import (
	"errors"
	"fmt"
	"testing"
)

// v5 P3.4 — bulk command guarantees.
//
// Criterion (b): a bulk command that dies at row 30,000 of 50,000 resumes and
// finishes at exactly 50,000.
// Criterion (c), bulk half: a cancelled command does not resume.

// crashingCheckpointStore is a store that can be told to die.
//
// It models the failure that matters and that a graceful error does NOT model:
// a process killed outright, where writes issued after the moment of death are
// simply lost. Without it, every "crash" test would actually be testing the
// clean shutdown path, where the runtime records exactly where it stopped and
// nothing is ever re-executed — which would make the resume guarantee look far
// stronger than it is.
type crashingCheckpointStore struct {
	inner   *MemoryCheckpointStore
	hasDied bool
}

func newCrashingCheckpointStore() *crashingCheckpointStore {
	return &crashingCheckpointStore{inner: NewMemoryCheckpointStore()}
}

func (parseStore *crashingCheckpointStore) Load(parseCommandID CommandID) (Checkpoint, bool, error) {
	return parseStore.inner.Load(parseCommandID)
}

func (parseStore *crashingCheckpointStore) Save(parseCheckpoint Checkpoint) error {
	if parseStore.hasDied {
		return nil // the write is lost, exactly as it would be on a killed process
	}
	return parseStore.inner.Save(parseCheckpoint)
}

func (parseStore *crashingCheckpointStore) Clear(parseCommandID CommandID) error {
	if parseStore.hasDied {
		return nil
	}
	return parseStore.inner.Clear(parseCommandID)
}

// ------------------------------------------------------------- happy path

func TestBulkRunsEveryItemOnce(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseRuns := make([]int, 100)

	parseResult, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: "bulk", Total: 100, CheckpointEvery: 10},
		func(parseIndex int) error {
			parseRuns[parseIndex]++
			return nil
		})
	if parseErr != nil {
		parseT.Fatalf("ExecuteBulk: %v", parseErr)
	}
	if !parseResult.Completed || parseResult.Processed != 100 || parseResult.NextIndex != 100 {
		parseT.Fatalf("result = %+v, want 100 processed and completed", parseResult)
	}
	for parseIndex, parseCount := range parseRuns {
		if parseCount != 1 {
			parseT.Fatalf("item %d ran %d times, want 1", parseIndex, parseCount)
		}
	}
}

// TestBulkReplayDoesNoWork: a completed bulk command re-issued under the same id
// must not walk 50,000 items again.
func TestBulkReplayDoesNoWork(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseCalls := 0
	parseCommand := BulkCommand{ID: "bulk", Total: 500, CheckpointEvery: 50}
	parseStep := func(int) error {
		parseCalls++
		return nil
	}

	if _, parseErr := parseRuntime.ExecuteBulk(parseCommand, parseStep); parseErr != nil {
		parseT.Fatalf("first run: %v", parseErr)
	}
	parseResult, parseErr := parseRuntime.ExecuteBulk(parseCommand, parseStep)
	if parseErr != nil {
		parseT.Fatalf("replay: %v", parseErr)
	}

	if parseResult.Outcome != OutcomeReplayed {
		parseT.Errorf("outcome = %s, want replayed", parseResult.Outcome)
	}
	if !parseResult.Completed {
		parseT.Error("a replayed bulk command must report completed")
	}
	if parseCalls != 500 {
		parseT.Errorf("step ran %d times across two executions, want 500", parseCalls)
	}
}

// ----------------------------------------------------- criterion (b): resume

// TestBulkResumesAfterAHardCrashAndFinishesAtExactly50k is P3.4 criterion (b),
// written as the plan states it.
//
// The first run is killed at row 30,000 with its checkpoint store dead, so the
// last durable record is the interval boundary before it — which is what a real
// process kill leaves behind.
func TestBulkResumesAfterAHardCrashAndFinishesAtExactly50k(parseT *testing.T) {
	const parseTotal = 50000
	const parseCrashAt = 30000
	const parseInterval = 1000

	parseStore := newCrashingCheckpointStore()
	parseCommand := BulkCommand{ID: "import", Total: parseTotal, CheckpointEvery: parseInterval}
	parseRunCountByIndex := make([]int, parseTotal)

	// Run 1 — killed at row 30,000.
	parseFirstRuntime := NewRuntime(parseStore)
	_, parseFirstErr := parseFirstRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		if parseIndex == parseCrashAt {
			parseStore.hasDied = true // nothing more reaches disk, including the failure record
			return errors.New("process killed")
		}
		parseRunCountByIndex[parseIndex]++
		return nil
	})
	if parseFirstErr == nil {
		parseT.Fatal("the first run was supposed to die")
	}

	// Run 2 — a fresh runtime, as a restarted worker would be. Only the durable
	// checkpoint carries over; all in-memory state is gone.
	parseStore.hasDied = false
	parseSecondRuntime := NewRuntime(parseStore)
	parseResult, parseErr := parseSecondRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		parseRunCountByIndex[parseIndex]++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("resume: %v", parseErr)
	}

	if !parseResult.Completed {
		parseT.Fatalf("result = %+v, want completed", parseResult)
	}
	if parseResult.NextIndex != parseTotal {
		parseT.Errorf("NextIndex = %d, want exactly %d", parseResult.NextIndex, parseTotal)
	}
	if parseResult.ResumedFrom != parseCrashAt {
		parseT.Errorf("ResumedFrom = %d, want %d — the crash lost more or less than one interval",
			parseResult.ResumedFrom, parseCrashAt)
	}

	// The load-bearing assertion. "Finishes at exactly 50k" means every one of the
	// 50,000 rows is complete AND none was done twice — a resume that skipped rows
	// and one that redid them would both satisfy a weaker no-zeroes check.
	//
	// Exactly-once holds here despite a non-transactional store because the crash
	// landed on an interval boundary, so the durable record happened to be current.
	// That is luck, not a guarantee, which is what MaxReplayWindow reports and what
	// TestBulkResumeReExecutesTheWindowAfterAMidIntervalCrash covers.
	for parseIndex, parseCount := range parseRunCountByIndex {
		if parseCount != 1 {
			parseT.Fatalf("row %d ran %d times, want exactly 1", parseIndex, parseCount)
		}
	}
	if parseResult.MaxReplayWindow != parseInterval {
		parseT.Errorf("MaxReplayWindow = %d, want %d", parseResult.MaxReplayWindow, parseInterval)
	}
}

// TestBulkResumeReExecutesTheWindowAfterAMidIntervalCrash is the case the
// guarantee is actually bounded by, and the one a caller with non-idempotent
// steps has to care about.
//
// The crash lands mid-interval, so the durable record trails the work that was
// really done, and the resume redoes precisely that gap — no more, and never
// less, since redoing is safe where skipping is not.
func TestBulkResumeReExecutesTheWindowAfterAMidIntervalCrash(parseT *testing.T) {
	const parseTotal = 50000
	const parseCrashAt = 30500
	const parseInterval = 1000
	const parseLastCheckpoint = 30000

	parseStore := newCrashingCheckpointStore()
	parseCommand := BulkCommand{ID: "import", Total: parseTotal, CheckpointEvery: parseInterval}
	parseRunCountByIndex := make([]int, parseTotal)

	parseFirstRuntime := NewRuntime(parseStore)
	if _, parseErr := parseFirstRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		if parseIndex == parseCrashAt {
			parseStore.hasDied = true
			return errors.New("process killed")
		}
		parseRunCountByIndex[parseIndex]++
		return nil
	}); parseErr == nil {
		parseT.Fatal("the first run was supposed to die")
	}

	parseStore.hasDied = false
	parseSecondRuntime := NewRuntime(parseStore)
	parseResult, parseErr := parseSecondRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		parseRunCountByIndex[parseIndex]++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("resume: %v", parseErr)
	}
	if !parseResult.Completed || parseResult.NextIndex != parseTotal {
		parseT.Fatalf("result = %+v, want completed at %d", parseResult, parseTotal)
	}
	if parseResult.ResumedFrom != parseLastCheckpoint {
		parseT.Fatalf("ResumedFrom = %d, want %d", parseResult.ResumedFrom, parseLastCheckpoint)
	}

	for parseIndex, parseCount := range parseRunCountByIndex {
		parseWantCount := 1
		if parseIndex >= parseLastCheckpoint && parseIndex < parseCrashAt {
			// Executed by the dead run and again by the resume: the window.
			parseWantCount = 2
		}
		if parseCount != parseWantCount {
			parseT.Fatalf("row %d ran %d times, want %d", parseIndex, parseCount, parseWantCount)
		}
	}

	// The window that was actually re-executed must not exceed the bound the
	// result reports, or the report is worse than useless.
	parseActualWindow := parseCrashAt - parseLastCheckpoint
	if parseActualWindow > parseResult.MaxReplayWindow {
		parseT.Errorf("re-executed %d items but reported a bound of %d — the bound is wrong",
			parseActualWindow, parseResult.MaxReplayWindow)
	}
}

// TestBulkResumeReExecutesNothingAfterAGracefulFailure is the other half of the
// honesty claim on MaxReplayWindow. A run that ends by returning an error knows
// exactly where it stopped and records it, so its resume duplicates nothing.
func TestBulkResumeReExecutesNothingAfterAGracefulFailure(parseT *testing.T) {
	const parseTotal = 5000
	const parseFailAt = 3333

	parseStore := NewMemoryCheckpointStore()
	parseCommand := BulkCommand{ID: "import", Total: parseTotal, CheckpointEvery: 1000}
	parseRunCountByIndex := make([]int, parseTotal)

	parseFirstRuntime := NewRuntime(parseStore)
	if _, parseErr := parseFirstRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		if parseIndex == parseFailAt {
			return errors.New("bad row")
		}
		parseRunCountByIndex[parseIndex]++
		return nil
	}); parseErr == nil {
		parseT.Fatal("the first run was supposed to fail")
	}

	parseSecondRuntime := NewRuntime(parseStore)
	parseResult, parseErr := parseSecondRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		parseRunCountByIndex[parseIndex]++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("resume: %v", parseErr)
	}

	if parseResult.ResumedFrom != parseFailAt {
		parseT.Errorf("ResumedFrom = %d, want %d — a graceful failure must record its exact position",
			parseResult.ResumedFrom, parseFailAt)
	}
	if parseResult.MaxReplayWindow != 0 {
		parseT.Errorf("MaxReplayWindow = %d, want 0 after a graceful failure", parseResult.MaxReplayWindow)
	}
	for parseIndex, parseCount := range parseRunCountByIndex {
		if parseCount != 1 {
			parseT.Fatalf("row %d ran %d times, want exactly 1", parseIndex, parseCount)
		}
	}
}

// TestBulkCheckpointEveryItemGivesExactlyOnce: the interval that closes the
// replay window entirely, at the cost of a write per item.
func TestBulkCheckpointEveryItemGivesExactlyOnce(parseT *testing.T) {
	const parseTotal = 2000
	const parseCrashAt = 1200

	parseStore := newCrashingCheckpointStore()
	parseCommand := BulkCommand{ID: "import", Total: parseTotal, CheckpointEvery: 1}
	parseRunCountByIndex := make([]int, parseTotal)

	parseFirstRuntime := NewRuntime(parseStore)
	if _, parseErr := parseFirstRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		if parseIndex == parseCrashAt {
			parseStore.hasDied = true
			return errors.New("process killed")
		}
		parseRunCountByIndex[parseIndex]++
		return nil
	}); parseErr == nil {
		parseT.Fatal("the first run was supposed to die")
	}

	parseStore.hasDied = false
	parseSecondRuntime := NewRuntime(parseStore)
	parseResult, parseErr := parseSecondRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		parseRunCountByIndex[parseIndex]++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("resume: %v", parseErr)
	}
	if !parseResult.Completed {
		parseT.Fatalf("result = %+v, want completed", parseResult)
	}

	for parseIndex, parseCount := range parseRunCountByIndex {
		if parseCount != 1 {
			parseT.Fatalf("row %d ran %d times, want exactly 1 with per-item checkpointing", parseIndex, parseCount)
		}
	}
}

// TestBulkResumeRefusesAChangedItemCount: a checkpoint written against a
// different total describes a different command wearing the same id. Resuming
// into it would skip or repeat an arbitrary range.
func TestBulkResumeRefusesAChangedItemCount(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	parseRuntime := NewRuntime(parseStore)

	if _, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: "import", Total: 100, CheckpointEvery: 10},
		func(parseIndex int) error {
			if parseIndex == 50 {
				return errors.New("stop")
			}
			return nil
		}); parseErr == nil {
		parseT.Fatal("expected a failure")
	}

	parseFreshRuntime := NewRuntime(parseStore)
	if _, parseErr := parseFreshRuntime.ExecuteBulk(
		BulkCommand{ID: "import", Total: 200, CheckpointEvery: 10},
		func(int) error { return nil }); parseErr == nil {
		parseT.Error("resuming against a different item count must be refused")
	}
}

// ------------------------------------------------ criterion (c): cancellation

// TestCancelledBulkCommandNeitherReplaysNorResumes is P3.4 criterion (c).
func TestCancelledBulkCommandNeitherReplaysNorResumes(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	parseRuntime := NewRuntime(parseStore)
	parseCommand := BulkCommand{ID: "import", Total: 1000, CheckpointEvery: 100}
	parseCalls := 0

	// Get partway, then stop.
	if _, parseErr := parseRuntime.ExecuteBulk(parseCommand, func(parseIndex int) error {
		parseCalls++
		if parseIndex == 500 {
			return errors.New("stop")
		}
		return nil
	}); parseErr == nil {
		parseT.Fatal("expected a failure")
	}
	parseCallsBeforeCancel := parseCalls

	if parseErr := parseRuntime.Cancel(parseCommand.ID); parseErr != nil {
		parseT.Fatalf("Cancel: %v", parseErr)
	}

	// Does not resume.
	parseResult, parseErr := parseRuntime.ExecuteBulk(parseCommand, func(int) error {
		parseCalls++
		return nil
	})
	if parseErr != nil {
		parseT.Fatalf("ExecuteBulk after cancel: %v", parseErr)
	}
	if parseResult.Outcome != OutcomeCancelled {
		parseT.Errorf("outcome = %s, want cancelled", parseResult.Outcome)
	}
	if parseCalls != parseCallsBeforeCancel {
		parseT.Errorf("step ran %d more times after a cancel, want 0", parseCalls-parseCallsBeforeCancel)
	}

	// And its progress is gone, so a fresh runtime cannot resume it either.
	if parseCheckpoints := parseStore.Snapshot(); len(parseCheckpoints) != 0 {
		parseT.Errorf("checkpoints remain after a cancel: %v", parseCheckpoints)
	}
}

// TestCancelStopsAnInFlightBulkCommand: a cancel arriving during a long run must
// stop it, not be noticed after it finishes.
func TestCancelStopsAnInFlightBulkCommand(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	parseCalls := 0

	parseResult, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: "import", Total: 10000, CheckpointEvery: 100},
		func(parseIndex int) error {
			parseCalls++
			if parseIndex == 250 {
				// A cancel delivered mid-run, as a message would be.
				return parseRuntime.Cancel("import")
			}
			return nil
		})
	if parseErr != nil {
		parseT.Fatalf("ExecuteBulk: %v", parseErr)
	}
	if parseResult.Outcome != OutcomeCancelled {
		parseT.Errorf("outcome = %s, want cancelled", parseResult.Outcome)
	}
	if parseCalls > 300 {
		parseT.Errorf("step ran %d times, want the run to stop shortly after the cancel", parseCalls)
	}
	if parseResult.Completed {
		parseT.Error("a cancelled run must not report completed")
	}
}

// ------------------------------------------------------------------- guards

func TestBulkWithoutACheckpointStoreStillRuns(parseT *testing.T) {
	parseRuntime := NewRuntime(nil)
	parseCalls := 0

	parseResult, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: "import", Total: 50},
		func(int) error {
			parseCalls++
			return nil
		})
	if parseErr != nil {
		parseT.Fatalf("ExecuteBulk: %v", parseErr)
	}
	if !parseResult.Completed || parseCalls != 50 {
		parseT.Errorf("result = %+v, calls = %d — a nil store must disable resume, not execution", parseResult, parseCalls)
	}

	// Replay protection still holds without a store.
	parseReplay, _ := parseRuntime.ExecuteBulk(BulkCommand{ID: "import", Total: 50}, func(int) error {
		parseCalls++
		return nil
	})
	if parseReplay.Outcome != OutcomeReplayed || parseCalls != 50 {
		parseT.Errorf("outcome = %s, calls = %d — replay protection must not depend on the store", parseReplay.Outcome, parseCalls)
	}
}

func TestBulkPanicIsContainedAndProgressSurvives(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	parseRuntime := NewRuntime(parseStore)

	if _, parseErr := parseRuntime.ExecuteBulk(
		BulkCommand{ID: "import", Total: 1000, CheckpointEvery: 100},
		func(parseIndex int) error {
			if parseIndex == 640 {
				panic("bad row")
			}
			return nil
		}); parseErr == nil {
		parseT.Fatal("a panic must surface as an error")
	}

	parseCheckpoint, hasCheckpoint, _ := parseStore.Load("import")
	if !hasCheckpoint {
		parseT.Fatal("progress must survive a panic")
	}
	if parseCheckpoint.NextIndex != 640 {
		parseT.Errorf("NextIndex = %d, want 640 — progress was lost or overstated", parseCheckpoint.NextIndex)
	}
}

func TestBulkRejectsBadInput(parseT *testing.T) {
	parseRuntime := NewRuntime(NewMemoryCheckpointStore())
	if _, parseErr := parseRuntime.ExecuteBulk(BulkCommand{Total: 10}, func(int) error { return nil }); parseErr == nil {
		parseT.Error("an empty command id must be rejected")
	}
	if _, parseErr := parseRuntime.ExecuteBulk(BulkCommand{ID: "x", Total: -1}, func(int) error { return nil }); parseErr == nil {
		parseT.Error("a negative total must be rejected")
	}
	if _, parseErr := parseRuntime.ExecuteBulk(BulkCommand{ID: "x", Total: 10}, nil); parseErr == nil {
		parseT.Error("a nil step must be rejected")
	}

	var parseNil *Runtime
	if _, parseErr := parseNil.ExecuteBulk(BulkCommand{ID: "x", Total: 1}, func(int) error { return nil }); parseErr == nil {
		parseT.Error("a nil runtime must error rather than panic")
	}
}

// TestCheckpointStoreRefusesToRewind: a late write from a superseded run would
// otherwise rewind a resumed run into work it had already passed.
func TestCheckpointStoreRefusesToRewind(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	if parseErr := parseStore.Save(Checkpoint{CommandID: "x", NextIndex: 500, Total: 1000}); parseErr != nil {
		parseT.Fatalf("Save: %v", parseErr)
	}
	if parseErr := parseStore.Save(Checkpoint{CommandID: "x", NextIndex: 100, Total: 1000}); parseErr != nil {
		parseT.Fatalf("Save: %v", parseErr)
	}

	parseCheckpoint, _, _ := parseStore.Load("x")
	if parseCheckpoint.NextIndex != 500 {
		parseT.Errorf("NextIndex = %d, want 500 — progress moved backwards", parseCheckpoint.NextIndex)
	}
}

func TestCheckpointStoreRejectsMissingID(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	if parseErr := parseStore.Save(Checkpoint{NextIndex: 1}); parseErr == nil {
		parseT.Error("a checkpoint without a command id must be rejected")
	}

	var parseNil *MemoryCheckpointStore
	if _, _, parseErr := parseNil.Load("x"); parseErr == nil {
		parseT.Error("a nil store must error rather than panic")
	}
	if parseErr := parseNil.Save(Checkpoint{CommandID: "x"}); parseErr == nil {
		parseT.Error("a nil store must error rather than panic")
	}
}

// TestBulkOverManyCommandsKeepsProgressSeparate guards the obvious way an index
// bug would show: one command's progress applied to another.
func TestBulkOverManyCommandsKeepsProgressSeparate(parseT *testing.T) {
	parseStore := NewMemoryCheckpointStore()
	parseRuntime := NewRuntime(parseStore)

	for parseCommandIndex := range 5 {
		parseCommandID := CommandID(fmt.Sprintf("bulk-%d", parseCommandIndex))
		parseRuns := make([]int, 100)

		parseResult, parseErr := parseRuntime.ExecuteBulk(
			BulkCommand{ID: parseCommandID, Total: 100, CheckpointEvery: 25},
			func(parseIndex int) error {
				parseRuns[parseIndex]++
				return nil
			})
		if parseErr != nil {
			parseT.Fatalf("%s: %v", parseCommandID, parseErr)
		}
		if parseResult.ResumedFrom != 0 {
			parseT.Errorf("%s resumed from %d, want 0 — it inherited another command's progress",
				parseCommandID, parseResult.ResumedFrom)
		}
		for parseIndex, parseCount := range parseRuns {
			if parseCount != 1 {
				parseT.Fatalf("%s item %d ran %d times, want 1", parseCommandID, parseIndex, parseCount)
			}
		}
	}
}
