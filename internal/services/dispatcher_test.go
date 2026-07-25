package services

import (
	"errors"
	"fmt"
	"testing"
)

// v5 P3.3 — the extracted work dispatcher.
//
// The properties under test are the ones a compute pool depends on (P3.12):
// stable routing, coalescing that does not lose the newest work, cancellation
// that holds against results already in flight, bounded queueing, and a
// replacement path that leaves no unit stranded.

func buildTestDispatcher() *Dispatcher {
	return NewDispatcher([]WorkerID{"w1", "w2", "w3"})
}

// ---------------------------------------------------------------- assignment

func TestDispatcherAssignmentIsStable(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	parseJob, parseErr := parseDispatcher.Start("unit")
	if parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}

	for parseAttempt := range 5 {
		parseNext, parseNextErr := parseDispatcher.Update("unit")
		if parseNextErr != nil {
			parseT.Fatalf("Update %d: %v", parseAttempt, parseNextErr)
		}
		if parseNext.WorkerID != parseJob.WorkerID {
			parseT.Fatalf("attempt %d: worker moved from %q to %q — assignment is not sticky",
				parseAttempt, parseJob.WorkerID, parseNext.WorkerID)
		}
	}
}

// TestDispatcherAssignmentIsDeterministic: the same unit must reach the same
// worker across processes, which is why the hash is explicit rather than Go's
// randomized map hash.
func TestDispatcherAssignmentIsDeterministic(parseT *testing.T) {
	parseFirst := buildTestDispatcher()
	parseSecond := NewDispatcher([]WorkerID{"w3", "w1", "w2"}) // declared out of order

	for parseIndex := range 20 {
		parseUnitID := fmt.Sprintf("unit-%d", parseIndex)

		parseFirstJob, parseFirstErr := parseFirst.Start(parseUnitID)
		parseSecondJob, parseSecondErr := parseSecond.Start(parseUnitID)
		if parseFirstErr != nil || parseSecondErr != nil {
			parseT.Fatalf("Start(%s): %v / %v", parseUnitID, parseFirstErr, parseSecondErr)
		}
		if parseFirstJob.WorkerID != parseSecondJob.WorkerID {
			parseT.Fatalf("%s: pool declaration order changed the assignment (%q vs %q)",
				parseUnitID, parseFirstJob.WorkerID, parseSecondJob.WorkerID)
		}
	}
}

func TestDispatcherSpreadsUnitsAcrossWorkers(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	parseWorkerHits := map[WorkerID]int{}
	for parseIndex := range 60 {
		parseJob, parseErr := parseDispatcher.Start(fmt.Sprintf("unit-%d", parseIndex))
		if parseErr != nil {
			parseT.Fatalf("Start: %v", parseErr)
		}
		parseWorkerHits[parseJob.WorkerID]++
	}
	if len(parseWorkerHits) != 3 {
		parseT.Errorf("units landed on %d workers, want all 3 — routing is not spreading", len(parseWorkerHits))
	}
}

func TestDispatcherRejectsUpdateBeforeStart(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if _, parseErr := parseDispatcher.Update("unit"); parseErr == nil {
		parseT.Error("updating a unit that was never started must fail")
	}
}

func TestDispatcherRejectsBlankUnitID(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if _, parseErr := parseDispatcher.Start("   "); parseErr == nil {
		parseT.Error("a blank unit id must be rejected")
	}
}

// --------------------------------------------------------------- coalescing

// TestDispatcherCoalescesUpdates is the property that keeps a burst of updates
// from becoming a queue of superseded work.
func TestDispatcherCoalescesUpdates(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if _, parseErr := parseDispatcher.Start("unit"); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	parseDepthAfterStart := parseDispatcher.QueueDepth()

	for range 50 {
		if _, parseErr := parseDispatcher.Update("unit"); parseErr != nil {
			parseT.Fatalf("Update: %v", parseErr)
		}
	}

	if parseDepth := parseDispatcher.QueueDepth(); parseDepth != parseDepthAfterStart+1 {
		parseT.Errorf("queue depth = %d, want %d — 50 updates did not coalesce to one",
			parseDepth, parseDepthAfterStart+1)
	}
}

func TestDispatcherDoesNotCoalesceAcrossUnits(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	for parseIndex := range 4 {
		parseUnitID := fmt.Sprintf("unit-%d", parseIndex)
		if _, parseErr := parseDispatcher.Start(parseUnitID); parseErr != nil {
			parseT.Fatalf("Start: %v", parseErr)
		}
		if _, parseErr := parseDispatcher.Update(parseUnitID); parseErr != nil {
			parseT.Fatalf("Update: %v", parseErr)
		}
	}
	if parseDepth := parseDispatcher.QueueDepth(); parseDepth != 8 {
		parseT.Errorf("queue depth = %d, want 8 — work from different units coalesced together", parseDepth)
	}
}

// TestDispatcherCoalescingSurvivesQueueCompaction: dropping one unit's work
// shifts every later job, invalidating the cached indices. A stale index would
// coalesce a new update into an unrelated job.
func TestDispatcherCoalescingSurvivesQueueCompaction(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	for _, parseUnitID := range []string{"a", "b", "c"} {
		if _, parseErr := parseDispatcher.Start(parseUnitID); parseErr != nil {
			parseT.Fatalf("Start(%s): %v", parseUnitID, parseErr)
		}
		if _, parseErr := parseDispatcher.Update(parseUnitID); parseErr != nil {
			parseT.Fatalf("Update(%s): %v", parseUnitID, parseErr)
		}
	}

	parseDispatcher.Dispose("a") // compacts the queue

	parseDepthBefore := parseDispatcher.QueueDepth()
	if _, parseErr := parseDispatcher.Update("c"); parseErr != nil {
		parseT.Fatalf("Update(c) after compaction: %v", parseErr)
	}
	if parseDepth := parseDispatcher.QueueDepth(); parseDepth != parseDepthBefore {
		parseT.Errorf("queue depth = %d, want %d — coalescing broke after compaction", parseDepth, parseDepthBefore)
	}

	for _, parseJob := range parseDispatcher.DrainQueue() {
		if parseJob.UnitID == "a" {
			parseT.Error("a disposed unit's work is still queued")
		}
	}
}

// -------------------------------------------------------------- cancellation

// TestDispatcherCancelInvalidatesInFlightWork is the guarantee that makes
// cancellation meaningful: a worker already running the old job cannot be
// recalled, so its result must be rejected on arrival.
func TestDispatcherCancelInvalidatesInFlightWork(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	parseJob, parseErr := parseDispatcher.Start("unit")
	if parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if !parseDispatcher.CommitAllowed("unit", parseJob.CancelGeneration) {
		parseT.Fatal("a fresh job must be committable")
	}

	if !parseDispatcher.Cancel("unit") {
		parseT.Fatal("Cancel reported no change on a started unit")
	}

	if !parseDispatcher.IsStale(parseJob) {
		parseT.Error("a job from before the cancel must be stale")
	}
	if parseDispatcher.CommitAllowed("unit", parseJob.CancelGeneration) {
		parseT.Error("a result from before the cancel must not be committable")
	}
}

func TestDispatcherCancelDropsQueuedWork(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if _, parseErr := parseDispatcher.Start("unit"); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if _, parseErr := parseDispatcher.Update("unit"); parseErr != nil {
		parseT.Fatalf("Update: %v", parseErr)
	}
	parseDispatcher.Cancel("unit")

	if parseDepth := parseDispatcher.QueueDepth(); parseDepth != 0 {
		parseT.Errorf("queue depth = %d, want 0 — queued work survived a cancel", parseDepth)
	}
}

func TestDispatcherWorkAfterCancelIsCommittable(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if _, parseErr := parseDispatcher.Start("unit"); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	parseDispatcher.Cancel("unit")

	parseJob, parseErr := parseDispatcher.Update("unit")
	if parseErr != nil {
		parseT.Fatalf("Update after cancel: %v", parseErr)
	}
	if parseDispatcher.IsStale(parseJob) {
		parseT.Error("work enqueued after a cancel must not be stale")
	}
	if !parseDispatcher.CommitAllowed("unit", parseJob.CancelGeneration) {
		parseT.Error("work enqueued after a cancel must be committable")
	}
}

func TestDispatcherFallbackBlocksCommits(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	parseJob, parseErr := parseDispatcher.Start("unit")
	if parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if !parseDispatcher.Fallback("unit") {
		parseT.Fatal("Fallback reported no change on a started unit")
	}
	if parseDispatcher.CommitAllowed("unit", parseJob.CancelGeneration) {
		parseT.Error("a locally-owned unit must not accept worker commits")
	}
}

func TestDispatcherDisposeBlocksCommits(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	parseJob, parseErr := parseDispatcher.Start("unit")
	if parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if !parseDispatcher.Dispose("unit") {
		parseT.Fatal("Dispose reported no change on a started unit")
	}
	if parseDispatcher.CommitAllowed("unit", parseJob.CancelGeneration) {
		parseT.Error("a disposed unit must not accept worker commits")
	}
	if parseDispatcher.Dispose("unit") {
		parseT.Error("disposing twice must report no change")
	}
}

// ------------------------------------------------------------- backpressure

func TestDispatcherEnforcesQueueLimit(parseT *testing.T) {
	parseDispatcher := NewDispatcherWithQueueLimit([]WorkerID{"w1", "w2"}, 3)
	for parseIndex := range 3 {
		if _, parseErr := parseDispatcher.Start(fmt.Sprintf("unit-%d", parseIndex)); parseErr != nil {
			parseT.Fatalf("Start %d: %v", parseIndex, parseErr)
		}
	}
	_, parseErr := parseDispatcher.Start("unit-overflow")
	if !errors.Is(parseErr, ErrQueueFull) {
		parseT.Errorf("err = %v, want ErrQueueFull", parseErr)
	}
}

// TestDispatcherCoalescingIsNotBlockedByBackpressure: an update that coalesces
// into existing queued work adds nothing to the queue, so a full queue must not
// reject it. Otherwise a saturated dispatcher would start dropping the newest
// state while holding superseded work.
func TestDispatcherCoalescingIsNotBlockedByBackpressure(parseT *testing.T) {
	parseDispatcher := NewDispatcherWithQueueLimit([]WorkerID{"w1"}, 2)
	if _, parseErr := parseDispatcher.Start("unit"); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if _, parseErr := parseDispatcher.Update("unit"); parseErr != nil {
		parseT.Fatalf("Update: %v", parseErr)
	}
	if parseDispatcher.QueueDepth() != 2 {
		parseT.Fatalf("queue depth = %d, want 2 (at the limit)", parseDispatcher.QueueDepth())
	}

	if _, parseErr := parseDispatcher.Update("unit"); parseErr != nil {
		parseT.Errorf("a coalescing update at the queue limit must succeed: %v", parseErr)
	}
}

func TestDispatcherDrainClearsQueueAndDropsStaleWork(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	for _, parseUnitID := range []string{"a", "b"} {
		if _, parseErr := parseDispatcher.Start(parseUnitID); parseErr != nil {
			parseT.Fatalf("Start(%s): %v", parseUnitID, parseErr)
		}
	}

	parseDrained := parseDispatcher.DrainQueue()
	if len(parseDrained) != 2 {
		parseT.Errorf("drained %d jobs, want 2", len(parseDrained))
	}
	if parseDispatcher.QueueDepth() != 0 {
		parseT.Error("draining must empty the queue")
	}
	if parseSecondDrain := parseDispatcher.DrainQueue(); len(parseSecondDrain) != 0 {
		parseT.Errorf("second drain returned %d jobs, want 0", len(parseSecondDrain))
	}
}

// -------------------------------------------------------------------- health

// TestDispatcherHeartbeatEscalatesFromDegradedToDead: one missed window is
// usually a long task, so it degrades; two means the worker is gone.
func TestDispatcherHeartbeatEscalatesFromDegradedToDead(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()

	parseHealth, parseErr := parseDispatcher.RecordHeartbeatTimeout("w1")
	if parseErr != nil {
		parseT.Fatalf("RecordHeartbeatTimeout: %v", parseErr)
	}
	if parseHealth != HealthDegraded {
		parseT.Errorf("health = %s, want degraded after one miss", parseHealth)
	}
	if !parseDispatcher.Degraded() {
		parseT.Error("the pool must report degraded when a worker is not ready")
	}

	parseHealth, parseErr = parseDispatcher.RecordHeartbeatTimeout("w1")
	if parseErr != nil {
		parseT.Fatalf("RecordHeartbeatTimeout: %v", parseErr)
	}
	if parseHealth != HealthDead {
		parseT.Errorf("health = %s, want dead after two misses", parseHealth)
	}
}

func TestDispatcherHeartbeatRecoversAWorker(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if _, parseErr := parseDispatcher.RecordHeartbeatTimeout("w1"); parseErr != nil {
		parseT.Fatalf("RecordHeartbeatTimeout: %v", parseErr)
	}
	if parseErr := parseDispatcher.RecordHeartbeat("w1", 1); parseErr != nil {
		parseT.Fatalf("RecordHeartbeat: %v", parseErr)
	}
	if parseDispatcher.WorkerHealthOf("w1") != HealthReady {
		parseT.Error("a heartbeat must restore a degraded worker to ready")
	}
	if parseDispatcher.Degraded() {
		parseT.Error("the pool must stop reporting degraded once every worker is ready")
	}

	// The miss counter must reset too, or the next single miss would kill it.
	parseHealth, _ := parseDispatcher.RecordHeartbeatTimeout("w1")
	if parseHealth != HealthDegraded {
		parseT.Errorf("health = %s, want degraded — the miss counter did not reset", parseHealth)
	}
}

// TestDispatcherRejectsStaleHeartbeat: an out-of-order heartbeat from before a
// failure would otherwise mark a dead worker ready again.
func TestDispatcherRejectsStaleHeartbeat(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if parseErr := parseDispatcher.RecordHeartbeat("w1", 10); parseErr != nil {
		parseT.Fatalf("RecordHeartbeat: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w1", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}

	if parseErr := parseDispatcher.RecordHeartbeat("w1", 4); parseErr == nil {
		parseT.Error("a stale heartbeat must be rejected")
	}
	if parseDispatcher.WorkerHealthOf("w1") != HealthDead {
		parseT.Error("a rejected heartbeat must not have revived the worker")
	}
}

func TestDispatcherRejectsInvalidHealthInputs(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if parseErr := parseDispatcher.RecordHeartbeat("w1", 0); parseErr == nil {
		parseT.Error("heartbeat sequence 0 must be rejected")
	}
	if parseErr := parseDispatcher.RecordHeartbeat("unknown", 1); parseErr == nil {
		parseT.Error("a heartbeat from an unknown worker must be rejected")
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w1", WorkerHealth(99)); parseErr == nil {
		parseT.Error("an invalid health state must be rejected")
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w1", healthUnknown); parseErr == nil {
		parseT.Error("the unknown health state must not be settable")
	}
}

// TestDispatcherRefusesWorkOnADegradedWorker: a degraded worker is expected to
// recover, and reassigning would discard the state it holds for the unit.
func TestDispatcherRefusesWorkOnADegradedWorker(parseT *testing.T) {
	parseDispatcher := NewDispatcher([]WorkerID{"w1"})
	if _, parseErr := parseDispatcher.Start("unit"); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w1", HealthDegraded); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}
	if _, parseErr := parseDispatcher.Update("unit"); parseErr == nil {
		parseT.Error("work must not be routed onto a degraded worker")
	}
}

// TestDispatcherRepairsOntoAnotherWorkerWhenAssignedWorkerDies: unlike degraded,
// a dead worker will not recover, so the unit moves.
func TestDispatcherRepairsOntoAnotherWorkerWhenAssignedWorkerDies(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	parseJob, parseErr := parseDispatcher.Start("unit")
	if parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetWorkerHealth(parseJob.WorkerID, HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}

	parseRepaired, parseRepairErr := parseDispatcher.Update("unit")
	if parseRepairErr != nil {
		parseT.Fatalf("Update after death: %v", parseRepairErr)
	}
	if parseRepaired.WorkerID == parseJob.WorkerID {
		parseT.Error("a unit on a dead worker must be moved")
	}
	if parseDispatcher.WorkerHealthOf(parseRepaired.WorkerID) != HealthReady {
		parseT.Error("a unit must be repaired onto a ready worker")
	}
}

func TestDispatcherFallsBackWhenNoWorkerIsReady(parseT *testing.T) {
	parseDispatcher := NewDispatcher([]WorkerID{"w1"})
	if parseErr := parseDispatcher.SetWorkerHealth("w1", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}

	_, parseErr := parseDispatcher.Start("unit")
	if !errors.Is(parseErr, ErrNoReadyWorker) {
		parseT.Errorf("err = %v, want ErrNoReadyWorker", parseErr)
	}
	if !parseDispatcher.InFallback("unit") {
		parseT.Error("a unit with nowhere to run must enter fallback rather than wait")
	}
}

// --------------------------------------------------------------- replacement

// TestReplaceWorkerRepairsEveryUnit is the fix for the source's behavior of
// returning on the first unrepairable unit, which left the remaining units
// assigned to a worker it had already deleted from the pool.
func TestReplaceWorkerRepairsEveryUnit(parseT *testing.T) {
	parseDispatcher := NewDispatcher([]WorkerID{"w1", "w2"})

	// Put several units on one worker by killing the other first, then reviving
	// it so a replacement target exists.
	if parseErr := parseDispatcher.SetWorkerHealth("w2", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}
	parseUnitIDs := []string{"u1", "u2", "u3", "u4", "u5"}
	for _, parseUnitID := range parseUnitIDs {
		if _, parseErr := parseDispatcher.Start(parseUnitID); parseErr != nil {
			parseT.Fatalf("Start(%s): %v", parseUnitID, parseErr)
		}
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w2", HealthReady); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w1", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}

	parseOutcome, parseErr := parseDispatcher.ReplaceWorker("w1", "w3")
	if parseErr != nil {
		parseT.Fatalf("ReplaceWorker: %v", parseErr)
	}
	if len(parseOutcome.Reassigned) != len(parseUnitIDs) {
		parseT.Errorf("reassigned %d units, want %d", len(parseOutcome.Reassigned), len(parseUnitIDs))
	}

	for _, parseUnitID := range parseUnitIDs {
		parseWorkerID, hasAssignment := parseDispatcher.Assignment(parseUnitID)
		if !hasAssignment {
			parseT.Errorf("%s: lost its assignment during replacement", parseUnitID)
			continue
		}
		if parseWorkerID == "w1" {
			parseT.Errorf("%s: still assigned to the replaced worker", parseUnitID)
		}
		if parseDispatcher.WorkerHealthOf(parseWorkerID) != HealthReady {
			parseT.Errorf("%s: assigned to %q which is %s", parseUnitID, parseWorkerID, parseDispatcher.WorkerHealthOf(parseWorkerID))
		}
	}
}

// buildUnplaceableReplacement sets up the one situation where a replacement
// leaves units with nowhere to go: every worker in the pool, INCLUDING the
// nominated replacement, is already known dead.
//
// This is the scenario the source mishandles. It repairs the first unit, returns
// the error, and leaves the remaining units assigned to the worker it has just
// deleted from the pool — they dispatch into nothing and never recover, with the
// symptom appearing far from the cause.
func buildUnplaceableReplacement(parseT *testing.T, parseUnitIDs []string) *Dispatcher {
	parseT.Helper()

	parseDispatcher := NewDispatcher([]WorkerID{"w1", "w2"})
	// Park everything on w1 so a single death affects every unit.
	if parseErr := parseDispatcher.SetWorkerHealth("w2", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth(w2, dead): %v", parseErr)
	}
	for _, parseUnitID := range parseUnitIDs {
		if _, parseErr := parseDispatcher.Start(parseUnitID); parseErr != nil {
			parseT.Fatalf("Start(%s): %v", parseUnitID, parseErr)
		}
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w1", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth(w1, dead): %v", parseErr)
	}

	// w2 is nominated as the replacement while already dead, so admitting it
	// leaves the pool with no ready worker at all.
	return parseDispatcher
}

// TestReplaceWorkerReportsEveryUnitItCouldNotPlace pins that no unit is left
// stranded when the replacement cannot take work.
func TestReplaceWorkerReportsEveryUnitItCouldNotPlace(parseT *testing.T) {
	parseUnitIDs := []string{"u1", "u2", "u3", "u4"}
	parseDispatcher := buildUnplaceableReplacement(parseT, parseUnitIDs)

	parseOutcome, parseErr := parseDispatcher.ReplaceWorker("w1", "w2")
	if parseErr == nil {
		parseT.Fatal("a replacement that placed no unit must report an error")
	}
	if len(parseOutcome.FellBack) != len(parseUnitIDs) {
		parseT.Errorf("fell back %d units, want %d — units were stranded silently",
			len(parseOutcome.FellBack), len(parseUnitIDs))
	}

	// The real damage the source does: an assignment pointing at a worker that
	// no longer exists in the pool.
	for _, parseUnitID := range parseUnitIDs {
		parseWorkerID, hasAssignment := parseDispatcher.Assignment(parseUnitID)
		if hasAssignment {
			parseT.Errorf("%s: still assigned to %q after an unplaceable replacement", parseUnitID, parseWorkerID)
		}
		if !parseDispatcher.InFallback(parseUnitID) {
			parseT.Errorf("%s: must be in fallback when it could not be placed", parseUnitID)
		}
	}
}

// TestReplaceWorkerDoesNotResurrectADeadReplacement: admitting a nominated
// replacement as ready when it is already known dead would route work straight
// back into a worker that cannot run it.
func TestReplaceWorkerDoesNotResurrectADeadReplacement(parseT *testing.T) {
	parseDispatcher := buildUnplaceableReplacement(parseT, []string{"u1"})

	if _, parseErr := parseDispatcher.ReplaceWorker("w1", "w2"); parseErr == nil {
		parseT.Fatal("expected an error when the replacement cannot take work")
	}
	if parseDispatcher.WorkerHealthOf("w2") != HealthDead {
		parseT.Errorf("w2 health = %s, want dead — a known-dead replacement was resurrected",
			parseDispatcher.WorkerHealthOf("w2"))
	}
	if len(parseDispatcher.ReadyWorkers()) != 0 {
		parseT.Errorf("ready workers = %v, want none", parseDispatcher.ReadyWorkers())
	}
}

// TestReplaceWorkerRetargetsQueuedWork: queued work naming the dead worker is
// not stale, only misaddressed, so it follows the unit rather than being lost.
func TestReplaceWorkerRetargetsQueuedWork(parseT *testing.T) {
	parseDispatcher := NewDispatcher([]WorkerID{"w1", "w2"})
	if parseErr := parseDispatcher.SetWorkerHealth("w2", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}
	if _, parseErr := parseDispatcher.Start("unit"); parseErr != nil {
		parseT.Fatalf("Start: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w2", HealthReady); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetWorkerHealth("w1", HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}

	if _, parseErr := parseDispatcher.ReplaceWorker("w1", "w3"); parseErr != nil {
		parseT.Fatalf("ReplaceWorker: %v", parseErr)
	}

	parseDrained := parseDispatcher.DrainQueue()
	if len(parseDrained) != 1 {
		parseT.Fatalf("drained %d jobs, want 1 — queued work was lost in the replacement", len(parseDrained))
	}
	if parseDrained[0].WorkerID == "w1" {
		parseT.Error("queued work still names the replaced worker")
	}
	if parseDispatcher.WorkerHealthOf(parseDrained[0].WorkerID) != HealthReady {
		parseT.Errorf("queued work retargeted onto %q which is not ready", parseDrained[0].WorkerID)
	}
}

func TestReplaceWorkerRejectsBadInputs(parseT *testing.T) {
	parseDispatcher := buildTestDispatcher()
	if _, parseErr := parseDispatcher.ReplaceWorker("w1", "  "); parseErr == nil {
		parseT.Error("a blank replacement id must be rejected")
	}
	if _, parseErr := parseDispatcher.ReplaceWorker("unknown", "w9"); parseErr == nil {
		parseT.Error("replacing an unknown worker must be rejected")
	}
	// Replacing a live worker would strand whatever it is running.
	if _, parseErr := parseDispatcher.ReplaceWorker("w1", "w9"); parseErr == nil {
		parseT.Error("replacing a worker that is not marked dead must be rejected")
	}
}

// ------------------------------------------------------------------- guards

func TestDispatcherNilReceiverIsSafe(parseT *testing.T) {
	var parseDispatcher *Dispatcher
	if _, parseErr := parseDispatcher.Start("unit"); parseErr == nil {
		parseT.Error("a nil dispatcher must error rather than panic")
	}
	if _, parseErr := parseDispatcher.Update("unit"); parseErr == nil {
		parseT.Error("a nil dispatcher must error rather than panic")
	}
	if _, parseErr := parseDispatcher.ReplaceWorker("a", "b"); parseErr == nil {
		parseT.Error("a nil dispatcher must error rather than panic")
	}
	if parseDispatcher.Cancel("unit") || parseDispatcher.Dispose("unit") || parseDispatcher.Fallback("unit") {
		parseT.Error("a nil dispatcher reports no changes")
	}
	if !parseDispatcher.IsStale(Job{}) {
		parseT.Error("a nil dispatcher must treat all work as stale rather than committable")
	}
	if parseDispatcher.CommitAllowed("unit", 0) {
		parseT.Error("a nil dispatcher must not allow commits")
	}
}

func TestDispatcherDeduplicatesAndIgnoresBlankWorkers(parseT *testing.T) {
	parseDispatcher := NewDispatcher([]WorkerID{"w1", "w1", "  ", "w2"})
	if parseReady := parseDispatcher.ReadyWorkers(); len(parseReady) != 2 {
		parseT.Errorf("ready workers = %v, want 2 distinct", parseReady)
	}
}

func TestWorkerHealthLabelsAreDistinct(parseT *testing.T) {
	parseSeen := map[string]bool{}
	for _, parseHealth := range []WorkerHealth{HealthReady, HealthDegraded, HealthRestarting, HealthDead} {
		if parseSeen[parseHealth.String()] {
			parseT.Errorf("health label %q is not distinct", parseHealth)
		}
		parseSeen[parseHealth.String()] = true
	}
	if healthUnknown.String() != "unknown" {
		parseT.Error("the zero health value must render as unknown")
	}
}
