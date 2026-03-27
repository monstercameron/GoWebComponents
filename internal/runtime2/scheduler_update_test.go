package runtime2

import "testing"

// TestHandleSchedulerUpdateReusesPriorShard verifies update uses the existing region assignment.
func TestHandleSchedulerUpdateReusesPriorShard(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	getSchedulerMountJob, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	getSchedulerUpdateJob, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getSchedulerUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) returned error: %v", getSchedulerUpdateErr)
	}
	if getSchedulerUpdateJob.GetSchedulerJobKind != SchedulerJobKindUpdate {
		getTesting.Fatalf("expected update job kind, got %q", getSchedulerUpdateJob.GetSchedulerJobKind)
	}
	if getSchedulerUpdateJob.GetSchedulerShardID != getSchedulerMountJob.GetSchedulerShardID {
		getTesting.Fatalf("expected update to reuse prior shard, mount=%q update=%q", getSchedulerMountJob.GetSchedulerShardID, getSchedulerUpdateJob.GetSchedulerShardID)
	}
}

// TestHandleSchedulerUpdateFailsForUnknownRegion verifies update does not allocate a shard for unknown regions.
func TestHandleSchedulerUpdateFailsForUnknownRegion(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	_, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-missing")
	if getSchedulerUpdateErr == nil {
		getTesting.Fatal("expected update for unknown region to fail")
	}
}

// TestHandleSchedulerUpdateRejectsDisposedRegion verifies update is rejected once region assignment is disposed.
func TestHandleSchedulerUpdateRejectsDisposedRegion(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	if !getScheduler.HandleSchedulerDispose("region-a") {
		getTesting.Fatal("expected HandleSchedulerDispose(region-a) to clear assignment")
	}
	_, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getSchedulerUpdateErr == nil {
		getTesting.Fatal("expected update after dispose to be rejected")
	}
}
