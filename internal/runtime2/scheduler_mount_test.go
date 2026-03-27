package runtime2

import "testing"

// TestHandleSchedulerMountDispatchesFirstRegionJob verifies mount allocates one shard and enqueues the initial region job.
func TestHandleSchedulerMountDispatchesFirstRegionJob(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	getSchedulerJob, getSchedulerJobErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerJobErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerJobErr)
	}
	if getSchedulerJob.GetSchedulerJobKind != SchedulerJobKindMount {
		getTesting.Fatalf("expected mount job kind, got %q", getSchedulerJob.GetSchedulerJobKind)
	}
	if getSchedulerJob.GetSchedulerShardID == "" {
		getTesting.Fatal("expected non-empty shard ID for mount job")
	}
	if getScheduler.GetSchedulerQueueDepth() != 1 {
		getTesting.Fatalf("expected one queued job after first mount, got %d", getScheduler.GetSchedulerQueueDepth())
	}
}

// TestHandleSchedulerMountFailsWithoutAvailableShards verifies mount fails clearly when no shard is available.
func TestHandleSchedulerMountFailsWithoutAvailableShards(getTesting *testing.T) {
	getScheduler := BuildScheduler(nil)
	_, getSchedulerJobErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerJobErr == nil {
		getTesting.Fatal("expected mount to fail when no shards are available")
	}
}

// TestHandleSchedulerMountAfterDisposeReusesPolicyConsistently verifies remount uses the same deterministic assignment policy.
func TestHandleSchedulerMountAfterDisposeReusesPolicyConsistently(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	getSchedulerJobFirst, getSchedulerJobFirstErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerJobFirstErr != nil {
		getTesting.Fatalf("first HandleSchedulerMount(region-a) returned error: %v", getSchedulerJobFirstErr)
	}
	if !getScheduler.HandleSchedulerDispose("region-a") {
		getTesting.Fatal("expected HandleSchedulerDispose(region-a) to clear assignment")
	}
	getSchedulerJobSecond, getSchedulerJobSecondErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerJobSecondErr != nil {
		getTesting.Fatalf("second HandleSchedulerMount(region-a) returned error: %v", getSchedulerJobSecondErr)
	}
	if getSchedulerJobSecond.GetSchedulerShardID != getSchedulerJobFirst.GetSchedulerShardID {
		getTesting.Fatalf("expected remount to reuse deterministic policy, first=%q second=%q", getSchedulerJobFirst.GetSchedulerShardID, getSchedulerJobSecond.GetSchedulerShardID)
	}
}
