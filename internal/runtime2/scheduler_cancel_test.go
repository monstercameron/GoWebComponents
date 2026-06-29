package runtime2

import "testing"

// TestHandleSchedulerCancelMarksQueuedJobsStale verifies cancel invalidates queued work for the region.
func TestHandleSchedulerCancelMarksQueuedJobsStale(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	getSchedulerMountJob, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	getSchedulerUpdateJob, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getSchedulerUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) returned error: %v", getSchedulerUpdateErr)
	}
	if !getScheduler.HandleSchedulerCancel("region-a") {
		getTesting.Fatal("expected HandleSchedulerCancel(region-a) to succeed")
	}
	if !getScheduler.HasSchedulerJobStale(getSchedulerMountJob) {
		getTesting.Fatal("expected mounted queued job to be stale after cancel")
	}
	if !getScheduler.HasSchedulerJobStale(getSchedulerUpdateJob) {
		getTesting.Fatal("expected updated queued job to be stale after cancel")
	}
}

// TestHandleSchedulerCancelSuppressesFutureStaleCommit verifies stale commit attempts are rejected once cancel advances generation.
func TestHandleSchedulerCancelSuppressesFutureStaleCommit(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1", "shard-2"})
	getSchedulerMountJob, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	if !getScheduler.HandleSchedulerCancel("region-a") {
		getTesting.Fatal("expected HandleSchedulerCancel(region-a) to succeed")
	}
	if getScheduler.HasSchedulerCommitAllowed("region-a", getSchedulerMountJob.GetSchedulerCancelVersion) {
		getTesting.Fatal("expected stale commit version to be rejected after cancel")
	}
	getSchedulerUpdateJob, getSchedulerUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getSchedulerUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) returned error after cancel: %v", getSchedulerUpdateErr)
	}
	if !getScheduler.HasSchedulerCommitAllowed("region-a", getSchedulerUpdateJob.GetSchedulerCancelVersion) {
		getTesting.Fatal("expected latest generation commit version to remain allowed")
	}
}

// TestHandleSchedulerCancelRemainsSafeOnRepeat verifies repeated cancel calls do not corrupt scheduler state.
func TestHandleSchedulerCancelRemainsSafeOnRepeat(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	_, getSchedulerMountErr := getScheduler.HandleSchedulerMount("region-a")
	if getSchedulerMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getSchedulerMountErr)
	}
	if !getScheduler.HandleSchedulerCancel("region-a") {
		getTesting.Fatal("expected first cancel to succeed")
	}
	if !getScheduler.HandleSchedulerCancel("region-a") {
		getTesting.Fatal("expected repeated cancel to remain safe")
	}
}
