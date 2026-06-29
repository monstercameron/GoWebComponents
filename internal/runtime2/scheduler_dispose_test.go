package runtime2

import "testing"

// TestHandleSchedulerDisposeRemovesRegionAssignment verifies dispose clears the region mapping.
func TestHandleSchedulerDisposeRemovesRegionAssignment(getTesting *testing.T) {
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
		getTesting.Fatal("expected update after dispose to fail because assignment should be removed")
	}
}

// TestHandleSchedulerDisposeClearsQueuedRegionJobs verifies dispose removes queued work for the disposed region.
func TestHandleSchedulerDisposeClearsQueuedRegionJobs(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	_, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getSchedulerUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) returned error: %v", getSchedulerUpdateErr)
	}
	if getScheduler.GetSchedulerQueueDepth() != 2 {
		getTesting.Fatalf("expected two queued jobs before dispose, got %d", getScheduler.GetSchedulerQueueDepth())
	}
	if !getScheduler.HandleSchedulerDispose("region-a") {
		getTesting.Fatal("expected HandleSchedulerDispose(region-a) to succeed")
	}
	if getScheduler.GetSchedulerQueueDepth() != 0 {
		getTesting.Fatalf("expected queued jobs for region-a to be removed on dispose, got %d", getScheduler.GetSchedulerQueueDepth())
	}
}

// TestHandleSchedulerDisposeAfterFallbackRemainsConsistent verifies fallback state does not block normal dispose behavior.
func TestHandleSchedulerDisposeAfterFallbackRemainsConsistent(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	if !getScheduler.HandleSchedulerFallback("region-a") {
		getTesting.Fatal("expected HandleSchedulerFallback(region-a) to succeed")
	}
	if !getScheduler.HandleSchedulerDispose("region-a") {
		getTesting.Fatal("expected HandleSchedulerDispose(region-a) to succeed after fallback")
	}
	if getScheduler.HandleSchedulerDispose("region-a") {
		getTesting.Fatal("expected repeated dispose to report no active region after fallback dispose")
	}
}
