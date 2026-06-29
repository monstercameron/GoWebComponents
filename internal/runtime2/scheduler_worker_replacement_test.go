package runtime2

import "testing"

// TestHandleSchedulerReplaceWorkerJoinsScheduler verifies a replacement shard is accepted into scheduler routing.
func TestHandleSchedulerReplaceWorkerJoinsScheduler(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDead); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, dead) returned error: %v", setSchedulerWorkerHealthErr)
	}
	if handleSchedulerReplaceErr := getScheduler.HandleSchedulerReplaceWorker("shard-1", "shard-2"); handleSchedulerReplaceErr != nil {
		getTesting.Fatalf("HandleSchedulerReplaceWorker(shard-1, shard-2) returned error: %v", handleSchedulerReplaceErr)
	}
	getSchedulerMountJob, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error after replacement: %v", getSchedulerMountErr)
	}
	if getSchedulerMountJob.GetSchedulerShardID != "shard-2" {
		getTesting.Fatalf("expected replacement shard assignment shard-2, got %q", getSchedulerMountJob.GetSchedulerShardID)
	}
}

// TestHandleSchedulerReplaceWorkerReassignsRegionsOnDeadWorker verifies dead-worker regions move to an available replacement.
func TestHandleSchedulerReplaceWorkerReassignsRegionsOnDeadWorker(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	getSchedulerMountJob, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	if getSchedulerMountJob.GetSchedulerShardID != "shard-1" {
		getTesting.Fatalf("expected initial assignment shard-1, got %q", getSchedulerMountJob.GetSchedulerShardID)
	}
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDead); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, dead) returned error: %v", setSchedulerWorkerHealthErr)
	}
	if handleSchedulerReplaceErr := getScheduler.HandleSchedulerReplaceWorker("shard-1", "shard-2"); handleSchedulerReplaceErr != nil {
		getTesting.Fatalf("HandleSchedulerReplaceWorker(shard-1, shard-2) returned error: %v", handleSchedulerReplaceErr)
	}
	getSchedulerUpdateJob, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getSchedulerUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) returned error after replacement: %v", getSchedulerUpdateErr)
	}
	if getSchedulerUpdateJob.GetSchedulerShardID != "shard-2" {
		getTesting.Fatalf("expected reassignment to replacement shard-2, got %q", getSchedulerUpdateJob.GetSchedulerShardID)
	}
}

// TestHandleSchedulerReplaceWorkerFailureEntersDegradedMode verifies replacement failures mark scheduler mode explicitly degraded.
func TestHandleSchedulerReplaceWorkerFailureEntersDegradedMode(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDead); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, dead) returned error: %v", setSchedulerWorkerHealthErr)
	}
	if handleSchedulerReplaceErr := getScheduler.HandleSchedulerReplaceWorker("shard-1", ""); handleSchedulerReplaceErr == nil {
		getTesting.Fatal("expected replacement failure for empty replacement shard ID")
	}
	if !getScheduler.GetSchedulerIsDegraded() {
		getTesting.Fatal("expected scheduler degraded mode after replacement failure")
	}
}

// TestHandleSchedulerReplaceWorkerPreservesDegradedModeForOtherShards verifies successful replacement still reports degraded mode when another shard remains non-ready.
func TestHandleSchedulerReplaceWorkerPreservesDegradedModeForOtherShards(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDead); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, dead) returned error: %v", setSchedulerWorkerHealthErr)
	}
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-2", SchedulerWorkerHealthDegraded); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-2, degraded) returned error: %v", setSchedulerWorkerHealthErr)
	}
	if handleSchedulerReplaceErr := getScheduler.HandleSchedulerReplaceWorker("shard-1", "shard-3"); handleSchedulerReplaceErr != nil {
		getTesting.Fatalf("HandleSchedulerReplaceWorker(shard-1, shard-3) returned error: %v", handleSchedulerReplaceErr)
	}
	if !getScheduler.GetSchedulerIsDegraded() {
		getTesting.Fatal("expected scheduler to remain degraded while shard-2 is degraded")
	}
}

// TestHandleSchedulerReplaceWorkerClearsDeadShardKeepaliveState verifies dead-shard keepalive counters are removed on replacement.
func TestHandleSchedulerReplaceWorkerClearsDeadShardKeepaliveState(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if handlePongErr := getScheduler.HandleSchedulerKeepalivePong("shard-1", 3); handlePongErr != nil {
		getTesting.Fatalf("HandleSchedulerKeepalivePong(shard-1,3) returned error: %v", handlePongErr)
	}
	if _, handleTimeoutErr := getScheduler.HandleSchedulerKeepaliveTimeout("shard-1"); handleTimeoutErr != nil {
		getTesting.Fatalf("HandleSchedulerKeepaliveTimeout(shard-1) returned error: %v", handleTimeoutErr)
	}
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDead); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, dead) returned error: %v", setSchedulerWorkerHealthErr)
	}
	if handleSchedulerReplaceErr := getScheduler.HandleSchedulerReplaceWorker("shard-1", "shard-2"); handleSchedulerReplaceErr != nil {
		getTesting.Fatalf("HandleSchedulerReplaceWorker(shard-1, shard-2) returned error: %v", handleSchedulerReplaceErr)
	}
	if _, hasDeadPong := getScheduler.storeSchedulerPongByShardID["shard-1"]; hasDeadPong {
		getTesting.Fatal("expected dead shard pong state to be cleared on replacement")
	}
	if _, hasDeadMissedPong := getScheduler.storeSchedulerMissedPongByShardID["shard-1"]; hasDeadMissedPong {
		getTesting.Fatal("expected dead shard missed-pong state to be cleared on replacement")
	}
	if _, hasReplacementPong := getScheduler.storeSchedulerPongByShardID["shard-2"]; !hasReplacementPong {
		getTesting.Fatal("expected replacement shard pong state to be initialized")
	}
	if _, hasReplacementMissedPong := getScheduler.storeSchedulerMissedPongByShardID["shard-2"]; !hasReplacementMissedPong {
		getTesting.Fatal("expected replacement shard missed-pong state to be initialized")
	}
}
