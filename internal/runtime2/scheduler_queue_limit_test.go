package runtime2

import "testing"

// TestHandleSchedulerMountAcceptsWorkBelowQueueLimit verifies scheduler work is accepted while queue depth is below limit.
func TestHandleSchedulerMountAcceptsWorkBelowQueueLimit(getTesting *testing.T) {
	getScheduler := BuildSchedulerWithQueueLimit([]SchedulerShardID{"shard-1"}, 2)
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error below queue limit: %v", getSchedulerMountErr)
	}
	if getScheduler.GetSchedulerQueueDepth() != 1 {
		getTesting.Fatalf("expected queue depth 1 below limit, got %d", getScheduler.GetSchedulerQueueDepth())
	}
}

// TestHandleSchedulerMountRejectsWorkAtQueueLimit verifies scheduler work is rejected once queue depth reaches limit.
func TestHandleSchedulerMountRejectsWorkAtQueueLimit(getTesting *testing.T) {
	getScheduler := BuildSchedulerWithQueueLimit([]SchedulerShardID{"shard-1"}, 1)
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned unexpected error: %v", getSchedulerMountErr)
	}
	_, getSchedulerMountSecondErr := getScheduler.HandleSchedulerMount("region-b")
	if getSchedulerMountSecondErr == nil {
		getTesting.Fatal("expected queue-limit rejection for second mount")
	}
}

// TestHandleSchedulerCancelFreesQueueCapacity verifies cancel removes queued work and reopens queue capacity.
func TestHandleSchedulerCancelFreesQueueCapacity(getTesting *testing.T) {
	getScheduler := BuildSchedulerWithQueueLimit([]SchedulerShardID{"shard-1"}, 1)
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned unexpected error: %v", getSchedulerMountErr)
	}
	if !getScheduler.HandleSchedulerCancel("region-a") {
		getTesting.Fatal("expected HandleSchedulerCancel(region-a) to succeed")
	}
	_, getSchedulerMountSecondErr := getScheduler.HandleSchedulerMount("region-b")
	if getSchedulerMountSecondErr != nil {
		getTesting.Fatalf("expected queue capacity after cancel, got error: %v", getSchedulerMountSecondErr)
	}
}
