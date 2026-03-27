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
