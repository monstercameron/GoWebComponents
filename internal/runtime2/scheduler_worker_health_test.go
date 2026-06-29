package runtime2

import (
	"strings"
	"testing"
)

// TestHandleSchedulerMountAcceptsHealthyWorker verifies ready workers continue to accept scheduler work.
func TestHandleSchedulerMountAcceptsHealthyWorker(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthReady); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, ready) returned error: %v", setSchedulerWorkerHealthErr)
	}
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error for ready worker: %v", getSchedulerMountErr)
	}
}

// TestHandleSchedulerMountRejectsDegradedWorkerExplicitly verifies degraded workers fail with an explicit error.
func TestHandleSchedulerMountRejectsDegradedWorkerExplicitly(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDegraded); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, degraded) returned error: %v", setSchedulerWorkerHealthErr)
	}
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr == nil {
		getTesting.Fatal("expected mount to reject degraded worker")
	}
	if !strings.Contains(getSchedulerMountErr.Error(), "degraded") {
		getTesting.Fatalf("expected explicit degraded-worker error, got: %v", getSchedulerMountErr)
	}
}

// TestHandleSchedulerUpdateReassignsDeadWorker verifies dead workers trigger reassignment when another shard is available.
func TestHandleSchedulerUpdateReassignsDeadWorker(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	getSchedulerMountJob, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	if setSchedulerWorkerHealthErr := getScheduler.SetSchedulerWorkerHealth(getSchedulerMountJob.GetSchedulerShardID, SchedulerWorkerHealthDead); setSchedulerWorkerHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(%q, dead) returned error: %v", getSchedulerMountJob.GetSchedulerShardID, setSchedulerWorkerHealthErr)
	}
	getSchedulerUpdateJob, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getSchedulerUpdateErr != nil {
		getTesting.Fatalf("expected dead-worker update to reassign when another worker is available, got: %v", getSchedulerUpdateErr)
	}
	if getSchedulerUpdateJob.GetSchedulerShardID == getSchedulerMountJob.GetSchedulerShardID {
		getTesting.Fatalf("expected reassignment away from dead shard %q", getSchedulerMountJob.GetSchedulerShardID)
	}
}
