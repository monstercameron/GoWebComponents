package services_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/v4/internal/services"
)

// v5 P3.3 — the extracted dispatcher must apply the same policies as the
// Scheduler it came from.
//
// The shared domain is POLICY, not routing: how work coalesces, when a cancel
// invalidates a result, when a commit is refused, and how health escalates from
// a missed heartbeat. Which specific worker a unit lands on is deliberately NOT
// shared — the source hashes over shard identities and the dispatcher over
// worker ids, and pinning that would freeze an implementation detail neither
// side promises.
//
// The other documented difference is ReplaceWorker's aggregate repair, covered
// by dispatcher_test.go.

const conformanceUnitID = "conformance-unit"

func buildConformancePair() (*runtime2.Scheduler, *services.Dispatcher) {
	return runtime2.BuildScheduler([]runtime2.SchedulerShardID{"s1", "s2", "s3"}),
		services.NewDispatcher([]services.WorkerID{"s1", "s2", "s3"})
}

// TestDispatcherMatchesRuntime2OnCoalescing pins the property that actually
// protects a saturated worker: a burst of updates must collapse to one queued
// job in both.
func TestDispatcherMatchesRuntime2OnCoalescing(parseT *testing.T) {
	parseScheduler, parseDispatcher := buildConformancePair()

	if _, parseErr := parseScheduler.HandleSchedulerMount(conformanceUnitID); parseErr != nil {
		parseT.Fatalf("runtime2 mount: %v", parseErr)
	}
	if _, parseErr := parseDispatcher.Start(conformanceUnitID); parseErr != nil {
		parseT.Fatalf("services start: %v", parseErr)
	}

	for parseIndex := range 25 {
		if _, parseErr := parseScheduler.HandleSchedulerUpdate(conformanceUnitID); parseErr != nil {
			parseT.Fatalf("runtime2 update %d: %v", parseIndex, parseErr)
		}
		if _, parseErr := parseDispatcher.Update(conformanceUnitID); parseErr != nil {
			parseT.Fatalf("services update %d: %v", parseIndex, parseErr)
		}
		if parseSchedulerDepth, parseDispatcherDepth := parseScheduler.GetSchedulerQueueDepth(), parseDispatcher.QueueDepth(); parseSchedulerDepth != parseDispatcherDepth {
			parseT.Fatalf("after update %d: runtime2 depth=%d but services depth=%d — coalescing diverged",
				parseIndex, parseSchedulerDepth, parseDispatcherDepth)
		}
	}
}

// TestDispatcherMatchesRuntime2OnCancelSemantics walks a job through cancel and
// compares staleness and commit admissibility at every step.
func TestDispatcherMatchesRuntime2OnCancelSemantics(parseT *testing.T) {
	parseScheduler, parseDispatcher := buildConformancePair()

	parseSchedulerJob, parseSchedulerErr := parseScheduler.HandleSchedulerMount(conformanceUnitID)
	parseDispatcherJob, parseDispatcherErr := parseDispatcher.Start(conformanceUnitID)
	if parseSchedulerErr != nil || parseDispatcherErr != nil {
		parseT.Fatalf("mount/start: %v / %v", parseSchedulerErr, parseDispatcherErr)
	}

	assertJobAgreement := func(parseLabel string) {
		parseT.Helper()
		parseSchedulerStale := parseScheduler.HasSchedulerJobStale(parseSchedulerJob)
		parseDispatcherStale := parseDispatcher.IsStale(parseDispatcherJob)
		if parseSchedulerStale != parseDispatcherStale {
			parseT.Fatalf("%s: runtime2 stale=%v but services stale=%v", parseLabel, parseSchedulerStale, parseDispatcherStale)
		}

		parseSchedulerCommit := parseScheduler.HasSchedulerCommitAllowed(conformanceUnitID, parseSchedulerJob.GetSchedulerCancelVersion)
		parseDispatcherCommit := parseDispatcher.CommitAllowed(conformanceUnitID, parseDispatcherJob.CancelGeneration)
		if parseSchedulerCommit != parseDispatcherCommit {
			parseT.Fatalf("%s: runtime2 commitAllowed=%v but services commitAllowed=%v",
				parseLabel, parseSchedulerCommit, parseDispatcherCommit)
		}
	}

	assertJobAgreement("fresh")

	if parseScheduler.HandleSchedulerCancel(conformanceUnitID) != parseDispatcher.Cancel(conformanceUnitID) {
		parseT.Fatal("cancel reported different results")
	}
	assertJobAgreement("after cancel")

	if parseSchedulerDepth, parseDispatcherDepth := parseScheduler.GetSchedulerQueueDepth(), parseDispatcher.QueueDepth(); parseSchedulerDepth != parseDispatcherDepth {
		parseT.Fatalf("after cancel: runtime2 depth=%d but services depth=%d", parseSchedulerDepth, parseDispatcherDepth)
	}

	// Work enqueued after the cancel must be committable again in both.
	parseSchedulerJob, parseSchedulerErr = parseScheduler.HandleSchedulerUpdate(conformanceUnitID)
	parseDispatcherJob, parseDispatcherErr = parseDispatcher.Update(conformanceUnitID)
	if parseSchedulerErr != nil || parseDispatcherErr != nil {
		parseT.Fatalf("update after cancel: %v / %v", parseSchedulerErr, parseDispatcherErr)
	}
	assertJobAgreement("after post-cancel update")
}

// TestDispatcherMatchesRuntime2OnCommitRefusal covers the three ways a unit
// loses the right to commit.
func TestDispatcherMatchesRuntime2OnCommitRefusal(parseT *testing.T) {
	for _, parseCase := range []struct {
		label  string
		revoke func(*runtime2.Scheduler, *services.Dispatcher)
	}{
		{
			label: "fallback",
			revoke: func(parseScheduler *runtime2.Scheduler, parseDispatcher *services.Dispatcher) {
				parseScheduler.HandleSchedulerFallback(conformanceUnitID)
				parseDispatcher.Fallback(conformanceUnitID)
			},
		},
		{
			label: "dispose",
			revoke: func(parseScheduler *runtime2.Scheduler, parseDispatcher *services.Dispatcher) {
				parseScheduler.HandleSchedulerDispose(conformanceUnitID)
				parseDispatcher.Dispose(conformanceUnitID)
			},
		},
		{
			label: "cancel",
			revoke: func(parseScheduler *runtime2.Scheduler, parseDispatcher *services.Dispatcher) {
				parseScheduler.HandleSchedulerCancel(conformanceUnitID)
				parseDispatcher.Cancel(conformanceUnitID)
			},
		},
	} {
		parseScheduler, parseDispatcher := buildConformancePair()
		parseSchedulerJob, parseSchedulerErr := parseScheduler.HandleSchedulerMount(conformanceUnitID)
		parseDispatcherJob, parseDispatcherErr := parseDispatcher.Start(conformanceUnitID)
		if parseSchedulerErr != nil || parseDispatcherErr != nil {
			parseT.Fatalf("%s: mount/start: %v / %v", parseCase.label, parseSchedulerErr, parseDispatcherErr)
		}

		parseCase.revoke(parseScheduler, parseDispatcher)

		parseSchedulerCommit := parseScheduler.HasSchedulerCommitAllowed(conformanceUnitID, parseSchedulerJob.GetSchedulerCancelVersion)
		parseDispatcherCommit := parseDispatcher.CommitAllowed(conformanceUnitID, parseDispatcherJob.CancelGeneration)
		if parseSchedulerCommit != parseDispatcherCommit {
			parseT.Errorf("%s: runtime2 commitAllowed=%v but services commitAllowed=%v",
				parseCase.label, parseSchedulerCommit, parseDispatcherCommit)
		}
		if parseDispatcherCommit {
			parseT.Errorf("%s: a revoked unit must not accept commits", parseCase.label)
		}
	}
}

// TestDispatcherMatchesRuntime2OnHealthEscalation pins the degrade-then-die
// ladder and the recovery that resets it.
func TestDispatcherMatchesRuntime2OnHealthEscalation(parseT *testing.T) {
	parseScheduler, parseDispatcher := buildConformancePair()

	assertHealthAgrees := func(parseLabel string, parseSchedulerHealth runtime2.SchedulerWorkerHealth, parseDispatcherHealth services.WorkerHealth) {
		parseT.Helper()
		parseAgrees := (parseSchedulerHealth == runtime2.SchedulerWorkerHealthReady && parseDispatcherHealth == services.HealthReady) ||
			(parseSchedulerHealth == runtime2.SchedulerWorkerHealthDegraded && parseDispatcherHealth == services.HealthDegraded) ||
			(parseSchedulerHealth == runtime2.SchedulerWorkerHealthDead && parseDispatcherHealth == services.HealthDead) ||
			(parseSchedulerHealth == runtime2.SchedulerWorkerHealthRestarting && parseDispatcherHealth == services.HealthRestarting)
		if !parseAgrees {
			parseT.Fatalf("%s: runtime2 health=%d but services health=%s", parseLabel, parseSchedulerHealth, parseDispatcherHealth)
		}
	}

	parseSchedulerHealth, parseSchedulerErr := parseScheduler.HandleSchedulerKeepaliveTimeout("s1")
	parseDispatcherHealth, parseDispatcherErr := parseDispatcher.RecordHeartbeatTimeout("s1")
	if parseSchedulerErr != nil || parseDispatcherErr != nil {
		parseT.Fatalf("first timeout: %v / %v", parseSchedulerErr, parseDispatcherErr)
	}
	assertHealthAgrees("one missed window", parseSchedulerHealth, parseDispatcherHealth)

	if parseScheduler.GetSchedulerIsDegraded() != parseDispatcher.Degraded() {
		parseT.Fatalf("degraded flag diverged: runtime2=%v services=%v",
			parseScheduler.GetSchedulerIsDegraded(), parseDispatcher.Degraded())
	}

	parseSchedulerHealth, parseSchedulerErr = parseScheduler.HandleSchedulerKeepaliveTimeout("s1")
	parseDispatcherHealth, parseDispatcherErr = parseDispatcher.RecordHeartbeatTimeout("s1")
	if parseSchedulerErr != nil || parseDispatcherErr != nil {
		parseT.Fatalf("second timeout: %v / %v", parseSchedulerErr, parseDispatcherErr)
	}
	assertHealthAgrees("two missed windows", parseSchedulerHealth, parseDispatcherHealth)

	// A heartbeat must restore readiness AND reset the miss counter in both, or
	// the next single miss would kill the worker.
	if parseErr := parseScheduler.HandleSchedulerKeepalivePong("s1", 1); parseErr != nil {
		parseT.Fatalf("runtime2 pong: %v", parseErr)
	}
	if parseErr := parseDispatcher.RecordHeartbeat("s1", 1); parseErr != nil {
		parseT.Fatalf("services heartbeat: %v", parseErr)
	}
	if parseScheduler.GetSchedulerIsDegraded() != parseDispatcher.Degraded() {
		parseT.Fatalf("degraded flag diverged after recovery: runtime2=%v services=%v",
			parseScheduler.GetSchedulerIsDegraded(), parseDispatcher.Degraded())
	}

	parseSchedulerHealth, _ = parseScheduler.HandleSchedulerKeepaliveTimeout("s1")
	parseDispatcherHealth, _ = parseDispatcher.RecordHeartbeatTimeout("s1")
	assertHealthAgrees("first miss after recovery", parseSchedulerHealth, parseDispatcherHealth)
}

// TestDispatcherMatchesRuntime2OnStaleHeartbeats: an out-of-order heartbeat
// must be refused by both, or a dead worker is marked ready again.
func TestDispatcherMatchesRuntime2OnStaleHeartbeats(parseT *testing.T) {
	parseScheduler, parseDispatcher := buildConformancePair()

	if parseErr := parseScheduler.HandleSchedulerKeepalivePong("s1", 10); parseErr != nil {
		parseT.Fatalf("runtime2 pong: %v", parseErr)
	}
	if parseErr := parseDispatcher.RecordHeartbeat("s1", 10); parseErr != nil {
		parseT.Fatalf("services heartbeat: %v", parseErr)
	}

	parseSchedulerErr := parseScheduler.HandleSchedulerKeepalivePong("s1", 3)
	parseDispatcherErr := parseDispatcher.RecordHeartbeat("s1", 3)
	if (parseSchedulerErr == nil) != (parseDispatcherErr == nil) {
		parseT.Errorf("stale heartbeat: runtime2 err=%v but services err=%v", parseSchedulerErr, parseDispatcherErr)
	}

	// Sequence zero and unknown workers are rejected by both.
	if (parseScheduler.HandleSchedulerKeepalivePong("s1", 0) == nil) != (parseDispatcher.RecordHeartbeat("s1", 0) == nil) {
		parseT.Error("zero heartbeat sequence handled differently")
	}
	if (parseScheduler.HandleSchedulerKeepalivePong("unknown", 1) == nil) != (parseDispatcher.RecordHeartbeat("unknown", 1) == nil) {
		parseT.Error("unknown worker handled differently")
	}
}

// TestDispatcherMatchesRuntime2OnBackpressure pins that both refuse new work at
// the queue limit rather than growing without bound.
func TestDispatcherMatchesRuntime2OnBackpressure(parseT *testing.T) {
	const parseLimit = 4
	parseScheduler := runtime2.BuildSchedulerWithQueueLimit([]runtime2.SchedulerShardID{"s1", "s2"}, parseLimit)
	parseDispatcher := services.NewDispatcherWithQueueLimit([]services.WorkerID{"s1", "s2"}, parseLimit)

	for parseIndex := range parseLimit + 2 {
		parseUnitID := fmt.Sprintf("unit-%d", parseIndex)
		parseSchedulerErr := func() error {
			_, parseErr := parseScheduler.HandleSchedulerMount(parseUnitID)
			return parseErr
		}()
		parseDispatcherErr := func() error {
			_, parseErr := parseDispatcher.Start(parseUnitID)
			return parseErr
		}()

		if (parseSchedulerErr == nil) != (parseDispatcherErr == nil) {
			parseT.Fatalf("unit %d: runtime2 err=%v but services err=%v", parseIndex, parseSchedulerErr, parseDispatcherErr)
		}
		if parseIndex >= parseLimit && parseDispatcherErr == nil {
			parseT.Fatalf("unit %d: work was accepted past the queue limit %d", parseIndex, parseLimit)
		}
	}

	if parseScheduler.GetSchedulerQueueDepth() != parseDispatcher.QueueDepth() {
		parseT.Errorf("queue depth diverged: runtime2=%d services=%d",
			parseScheduler.GetSchedulerQueueDepth(), parseDispatcher.QueueDepth())
	}
}

// TestDispatcherMatchesRuntime2OnDegradedWorkerRefusal: a degraded worker keeps
// its units and refuses work in both, rather than triggering a reassignment
// that would discard the state it holds.
func TestDispatcherMatchesRuntime2OnDegradedWorkerRefusal(parseT *testing.T) {
	parseScheduler := runtime2.BuildScheduler([]runtime2.SchedulerShardID{"s1"})
	parseDispatcher := services.NewDispatcher([]services.WorkerID{"s1"})

	if _, parseErr := parseScheduler.HandleSchedulerMount(conformanceUnitID); parseErr != nil {
		parseT.Fatalf("runtime2 mount: %v", parseErr)
	}
	if _, parseErr := parseDispatcher.Start(conformanceUnitID); parseErr != nil {
		parseT.Fatalf("services start: %v", parseErr)
	}

	if parseErr := parseScheduler.SetSchedulerWorkerHealth("s1", runtime2.SchedulerWorkerHealthDegraded); parseErr != nil {
		parseT.Fatalf("runtime2 set health: %v", parseErr)
	}
	if parseErr := parseDispatcher.SetWorkerHealth("s1", services.HealthDegraded); parseErr != nil {
		parseT.Fatalf("services set health: %v", parseErr)
	}

	_, parseSchedulerErr := parseScheduler.HandleSchedulerUpdate(conformanceUnitID)
	_, parseDispatcherErr := parseDispatcher.Update(conformanceUnitID)
	if (parseSchedulerErr == nil) != (parseDispatcherErr == nil) {
		parseT.Errorf("degraded refusal: runtime2 err=%v but services err=%v", parseSchedulerErr, parseDispatcherErr)
	}
	if parseDispatcherErr == nil {
		parseT.Error("work must not be routed onto a degraded worker")
	}
}
