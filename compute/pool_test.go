package compute_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/compute"
)

// v5 P3.12 — the compute pool.
//
// Criterion: a CPU-bound job runs off-thread with bounded queueing and
// backpressure, and its worker restarts without losing queued jobs.
//
// "Without losing queued jobs" has two halves that fail independently: jobs
// merely waiting, which were never tied to the dead worker, and jobs already
// handed to it, which vanish with it unless something puts them back. The second
// is the one an implementation loses by accident, so it gets the most attention
// here — including the payloads, since a requeued job with the right id and no
// payload is a job count that looks correct and work that can never run.

func buildPool(parseT *testing.T, parseWorkers []compute.WorkerID, parseOptions compute.Options) *compute.Pool {
	parseT.Helper()
	parsePool, parseErr := compute.NewPool(parseWorkers, parseOptions)
	if parseErr != nil {
		parseT.Fatalf("NewPool: %v", parseErr)
	}
	return parsePool
}

func buildJob(parseIndex int) compute.Job {
	return compute.Job{
		ID:      compute.JobID(fmt.Sprintf("job-%03d", parseIndex)),
		Kind:    "decode",
		Payload: []byte(fmt.Sprintf("payload-%d", parseIndex)),
	}
}

// -------------------------------------------------------------- scheduling

func TestJobsAreDispatchedToWorkers(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1", "w2"}, compute.Options{})

	for parseIndex := range 2 {
		if parseErr := parsePool.Submit(buildJob(parseIndex)); parseErr != nil {
			parseT.Fatalf("Submit: %v", parseErr)
		}
	}

	parseAssignments := parsePool.Dispatch()
	if len(parseAssignments) != 2 {
		parseT.Fatalf("assignments = %d, want 2", len(parseAssignments))
	}
	if parsePool.QueueDepth() != 0 || parsePool.InFlight() != 2 {
		parseT.Errorf("queue=%d inFlight=%d, want 0 and 2", parsePool.QueueDepth(), parsePool.InFlight())
	}
	// Payloads must reach the worker; scheduling ids alone would be useless.
	for _, parseAssignment := range parseAssignments {
		if len(parseAssignment.Job.Payload) == 0 {
			parseT.Errorf("assignment %+v carries no payload", parseAssignment)
		}
	}
}

// TestWorkIsBalancedNotHashed is why the pool does its own assignment rather
// than reusing the dispatcher's sticky routing. A compute job holds no state, so
// stickiness buys nothing and costs balance.
func TestWorkIsBalancedNotHashed(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1", "w2", "w3"},
		compute.Options{MaxInFlightPerWorker: 2})

	for parseIndex := range 6 {
		if parseErr := parsePool.Submit(buildJob(parseIndex)); parseErr != nil {
			parseT.Fatalf("Submit: %v", parseErr)
		}
	}

	parseLoad := map[compute.WorkerID]int{}
	for _, parseAssignment := range parsePool.Dispatch() {
		parseLoad[parseAssignment.WorkerID]++
	}
	if len(parseLoad) != 3 {
		parseT.Fatalf("load = %v, want all 3 workers used", parseLoad)
	}
	for parseWorkerID, parseCount := range parseLoad {
		if parseCount != 2 {
			parseT.Errorf("worker %q got %d jobs, want an even 2 — hashing would leave one idle", parseWorkerID, parseCount)
		}
	}
}

// TestCapacityIsRespectedAndExcessStaysQueued: dispatching everything and
// letting workers queue internally would move the backpressure somewhere it
// cannot be observed.
func TestCapacityIsRespectedAndExcessStaysQueued(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1", "w2"}, compute.Options{})

	for parseIndex := range 5 {
		if parseErr := parsePool.Submit(buildJob(parseIndex)); parseErr != nil {
			parseT.Fatalf("Submit: %v", parseErr)
		}
	}

	parseAssignments := parsePool.Dispatch()
	if len(parseAssignments) != 2 {
		parseT.Errorf("assignments = %d, want 2 (one per single-slot worker)", len(parseAssignments))
	}
	if parsePool.QueueDepth() != 3 {
		parseT.Errorf("queue = %d, want the remaining 3 still waiting", parsePool.QueueDepth())
	}

	// Completing frees capacity for the next dispatch.
	if !parsePool.Complete(parseAssignments[0].Job.ID) {
		parseT.Fatal("Complete reported no change for an in-flight job")
	}
	if parseNext := parsePool.Dispatch(); len(parseNext) != 1 {
		parseT.Errorf("assignments after one completion = %d, want 1", len(parseNext))
	}
}

// TestQueuedOrderIsPreserved: skipping ahead when capacity runs out would
// reorder jobs for no gain, since capacity is what is missing, not eligibility.
func TestQueuedOrderIsPreserved(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1"}, compute.Options{})

	for parseIndex := range 4 {
		if parseErr := parsePool.Submit(buildJob(parseIndex)); parseErr != nil {
			parseT.Fatalf("Submit: %v", parseErr)
		}
	}

	var parseDispatchOrder []compute.JobID
	for range 4 {
		for _, parseAssignment := range parsePool.Dispatch() {
			parseDispatchOrder = append(parseDispatchOrder, parseAssignment.Job.ID)
			parsePool.Complete(parseAssignment.Job.ID)
		}
	}

	for parseIndex, parseJobID := range parseDispatchOrder {
		if parseJobID != buildJob(parseIndex).ID {
			parseT.Fatalf("dispatch order = %v, want submission order", parseDispatchOrder)
		}
	}
}

// ------------------------------------------------------------ backpressure

func TestQueueLimitRefusesRatherThanGrowing(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1"}, compute.Options{QueueLimit: 3})

	for parseIndex := range 3 {
		if parseErr := parsePool.Submit(buildJob(parseIndex)); parseErr != nil {
			parseT.Fatalf("Submit %d: %v", parseIndex, parseErr)
		}
	}
	parseErr := parsePool.Submit(buildJob(99))
	if !errors.Is(parseErr, compute.ErrQueueFull) {
		parseT.Errorf("err = %v, want ErrQueueFull — an unbounded queue turns an overrunning producer into a memory leak", parseErr)
	}
}

func TestDuplicateJobIDIsRefused(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1"}, compute.Options{})
	if parseErr := parsePool.Submit(buildJob(1)); parseErr != nil {
		parseT.Fatalf("Submit: %v", parseErr)
	}
	if parseErr := parsePool.Submit(buildJob(1)); parseErr == nil {
		parseT.Error("a duplicate job id must be refused — running it twice and completing once leaves a caller waiting forever")
	}
}

func TestNoReadyWorkerLeavesJobsQueued(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1"}, compute.Options{})
	if parseErr := parsePool.Submit(buildJob(1)); parseErr != nil {
		parseT.Fatalf("Submit: %v", parseErr)
	}
	if parseErr := parsePool.SetWorkerHealth("w1", compute.HealthDead); parseErr != nil {
		parseT.Fatalf("SetWorkerHealth: %v", parseErr)
	}

	if parseAssignments := parsePool.Dispatch(); len(parseAssignments) != 0 {
		parseT.Errorf("assignments = %d, want 0 with no ready worker", len(parseAssignments))
	}
	// Queued, not dropped: the pool may recover.
	if parsePool.QueueDepth() != 1 {
		parseT.Errorf("queue = %d, want the job still waiting", parsePool.QueueDepth())
	}
}

// -------------------------------------------------- P3.12: worker restart

// TestWorkerRestartLosesNoJobs is P3.12's criterion.
func TestWorkerRestartLosesNoJobs(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1", "w2"},
		compute.Options{MaxInFlightPerWorker: 2})

	const parseJobCount = 8
	for parseIndex := range parseJobCount {
		if parseErr := parsePool.Submit(buildJob(parseIndex)); parseErr != nil {
			parseT.Fatalf("Submit: %v", parseErr)
		}
	}

	// Every job that is ever handed to a worker is counted here, including the
	// pre-restart batch — otherwise jobs that ran on the SURVIVING worker look
	// like they never ran, and the test fails for a reason that is about the
	// test.
	parseRan := map[compute.JobID]int{}

	parseAssignments := parsePool.Dispatch()
	parseDoomed := map[compute.JobID]bool{}
	parseSurvivors := map[compute.JobID]bool{}
	for _, parseAssignment := range parseAssignments {
		parseRan[parseAssignment.Job.ID]++
		if parseAssignment.WorkerID == "w1" {
			parseDoomed[parseAssignment.Job.ID] = true
		} else {
			parseSurvivors[parseAssignment.Job.ID] = true
		}
	}
	if len(parseDoomed) == 0 {
		parseT.Fatal("no jobs landed on w1; the test would prove nothing")
	}
	parseQueuedBefore := parsePool.QueueDepth()

	parseRequeued, parseErr := parsePool.RestartWorker("w1", "w3")
	if parseErr != nil {
		parseT.Fatalf("RestartWorker: %v", parseErr)
	}
	if parseRequeued != len(parseDoomed) {
		parseT.Errorf("requeued %d jobs, want the %d that were in flight on w1", parseRequeued, len(parseDoomed))
	}
	// Both halves: the in-flight jobs came back AND the merely-waiting ones stayed.
	if parsePool.QueueDepth() != parseQueuedBefore+len(parseDoomed) {
		parseT.Errorf("queue = %d, want %d — jobs were lost in the restart",
			parsePool.QueueDepth(), parseQueuedBefore+len(parseDoomed))
	}

	// The surviving worker's jobs finish normally; a restart elsewhere must not
	// have disturbed them.
	for parseJobID := range parseSurvivors {
		if !parsePool.Complete(parseJobID) {
			parseT.Errorf("job %q on the surviving worker was disturbed by the restart", parseJobID)
		}
	}

	// Every submitted job must eventually run, and the requeued ones exactly once.
	for range 20 {
		parseBatch := parsePool.Dispatch()
		if len(parseBatch) == 0 && parsePool.QueueDepth() == 0 {
			break
		}
		for _, parseAssignment := range parseBatch {
			parseRan[parseAssignment.Job.ID]++
			if len(parseAssignment.Job.Payload) == 0 {
				parseT.Fatalf("job %q came back from the restart with no payload", parseAssignment.Job.ID)
			}
			parsePool.Complete(parseAssignment.Job.ID)
		}
	}

	for parseIndex := range parseJobCount {
		parseJobID := buildJob(parseIndex).ID
		if parseRan[parseJobID] == 0 && !parseDoomed[parseJobID] {
			parseT.Errorf("job %q never ran", parseJobID)
		}
	}
	// A job interrupted by a worker death is dispatched TWICE: once to the
	// worker that died with it, once after the restart. That is at-least-once,
	// and it is the honest semantics of a pool that cannot know how far a dead
	// worker got — the alternative is dropping the job, which is worse. Compute
	// jobs must therefore be idempotent or have their first result discarded.
	for parseJobID := range parseDoomed {
		if parseRan[parseJobID] != 2 {
			parseT.Errorf("job %q was dispatched %d times, want 2 (the lost attempt and the retry)",
				parseJobID, parseRan[parseJobID])
		}
	}
	// A job that was never touched by the failure must run exactly once.
	for parseJobID := range parseSurvivors {
		if parseRan[parseJobID] != 1 {
			parseT.Errorf("job %q on the surviving worker was dispatched %d times, want 1",
				parseJobID, parseRan[parseJobID])
		}
	}
}

// TestRequeuedJobsGoToTheFront: they have already waited once, and putting them
// behind newer work starves exactly the jobs a restart already delayed.
func TestRequeuedJobsGoToTheFront(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1", "w2"}, compute.Options{})

	if parseErr := parsePool.Submit(buildJob(0)); parseErr != nil {
		parseT.Fatalf("Submit: %v", parseErr)
	}
	if parseErr := parsePool.Submit(buildJob(1)); parseErr != nil {
		parseT.Fatalf("Submit: %v", parseErr)
	}
	parseFirstBatch := parsePool.Dispatch()

	// Submit newer work while the first batch is running.
	for parseIndex := 10; parseIndex < 13; parseIndex++ {
		if parseErr := parsePool.Submit(buildJob(parseIndex)); parseErr != nil {
			parseT.Fatalf("Submit: %v", parseErr)
		}
	}

	var parseVictimWorker compute.WorkerID = parseFirstBatch[0].WorkerID
	parseVictimJob := parseFirstBatch[0].Job.ID
	parseReplacement := compute.WorkerID("w9")
	if _, parseErr := parsePool.RestartWorker(parseVictimWorker, parseReplacement); parseErr != nil {
		parseT.Fatalf("RestartWorker: %v", parseErr)
	}

	parseNext := parsePool.Dispatch()
	if len(parseNext) == 0 {
		parseT.Fatal("nothing dispatched after the restart")
	}
	if parseNext[0].Job.ID != parseVictimJob {
		parseT.Errorf("first dispatched job = %q, want the requeued %q — a restarted job was put behind newer work",
			parseNext[0].Job.ID, parseVictimJob)
	}
}

func TestRestartRefusesAnUnknownWorker(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1"}, compute.Options{})
	if _, parseErr := parsePool.RestartWorker("nope", "w2"); parseErr == nil {
		parseT.Error("restarting an unknown worker must be refused")
	}
}

// ------------------------------------------------------------------ health

func TestHeartbeatTimeoutDegradesThenRemovesFromRotation(parseT *testing.T) {
	parsePool := buildPool(parseT, []compute.WorkerID{"w1", "w2"}, compute.Options{})

	parseHealth, parseErr := parsePool.RecordHeartbeatTimeout("w1")
	if parseErr != nil {
		parseT.Fatalf("RecordHeartbeatTimeout: %v", parseErr)
	}
	if parseHealth != compute.HealthDegraded {
		parseT.Errorf("health = %s, want degraded after one miss", parseHealth)
	}
	if !parsePool.Degraded() {
		parseT.Error("the pool must report degraded")
	}

	// A degraded worker takes no new work.
	if parseErr := parsePool.Submit(buildJob(1)); parseErr != nil {
		parseT.Fatalf("Submit: %v", parseErr)
	}
	for _, parseAssignment := range parsePool.Dispatch() {
		if parseAssignment.WorkerID == "w1" {
			parseT.Error("work was routed to a degraded worker")
		}
	}

	if parseErr := parsePool.RecordHeartbeat("w1", 1); parseErr != nil {
		parseT.Fatalf("RecordHeartbeat: %v", parseErr)
	}
	if parsePool.Degraded() {
		parseT.Error("a heartbeat must restore the worker")
	}
}

// ------------------------------------------------------------------ guards

func TestPoolRejectsBadInput(parseT *testing.T) {
	if _, parseErr := compute.NewPool(nil, compute.Options{}); parseErr == nil {
		parseT.Error("a pool with no workers must be rejected")
	}

	parsePool := buildPool(parseT, []compute.WorkerID{"w1"}, compute.Options{})
	if parseErr := parsePool.Submit(compute.Job{}); parseErr == nil {
		parseT.Error("a job without an id must be rejected")
	}
	if parsePool.Complete("never-submitted") {
		parseT.Error("completing an unknown job must report no change")
	}
}

func TestNilPoolIsSafe(parseT *testing.T) {
	var parsePool *compute.Pool
	if parseErr := parsePool.Submit(buildJob(1)); parseErr == nil {
		parseT.Error("a nil pool must error rather than panic")
	}
	if _, parseErr := parsePool.RestartWorker("a", "b"); parseErr == nil {
		parseT.Error("a nil pool must error rather than panic")
	}
	if parsePool.Dispatch() != nil || parsePool.QueueDepth() != 0 || parsePool.InFlight() != 0 ||
		parsePool.ReadyWorkers() != nil || parsePool.Degraded() {
		parseT.Error("a nil pool reads as empty")
	}
	if parsePool.Complete("x") {
		parseT.Error("a nil pool completes nothing")
	}
}
