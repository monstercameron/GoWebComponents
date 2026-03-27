package runtime2

import "testing"

// TestHandleSchedulerUpdateCoalescesQueuedRegionUpdates verifies repeated updates for one region reuse one queued update slot.
func TestHandleSchedulerUpdateCoalescesQueuedRegionUpdates(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if _, getMountErr := getScheduler.HandleSchedulerMount("region-a"); getMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getMountErr)
	}
	if _, getUpdateErr := getScheduler.HandleSchedulerUpdate("region-a"); getUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) first call returned error: %v", getUpdateErr)
	}
	getQueueDepthAfterFirstUpdate := getScheduler.GetSchedulerQueueDepth()
	if getQueueDepthAfterFirstUpdate != 2 {
		getTesting.Fatalf("expected queue depth 2 after mount + first update, got %d", getQueueDepthAfterFirstUpdate)
	}
	if _, getUpdateErr := getScheduler.HandleSchedulerUpdate("region-a"); getUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) second call returned error: %v", getUpdateErr)
	}
	if _, getUpdateErr := getScheduler.HandleSchedulerUpdate("region-a"); getUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) third call returned error: %v", getUpdateErr)
	}
	if getScheduler.GetSchedulerQueueDepth() != getQueueDepthAfterFirstUpdate {
		getTesting.Fatalf(
			"expected queue depth to remain %d after coalesced updates, got %d",
			getQueueDepthAfterFirstUpdate,
			getScheduler.GetSchedulerQueueDepth(),
		)
	}
}

// TestHandleSchedulerUpdateEquivalentQueuedUpdateSkipsHealthChecks verifies equivalent queued updates short-circuit before shard-health checks.
func TestHandleSchedulerUpdateEquivalentQueuedUpdateSkipsHealthChecks(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if _, getMountErr := getScheduler.HandleSchedulerMount("region-a"); getMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getMountErr)
	}
	if _, getUpdateErr := getScheduler.HandleSchedulerUpdate("region-a"); getUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) first call returned error: %v", getUpdateErr)
	}
	if setHealthErr := getScheduler.SetSchedulerWorkerHealth("shard-1", SchedulerWorkerHealthDegraded); setHealthErr != nil {
		getTesting.Fatalf("SetSchedulerWorkerHealth(shard-1, degraded) returned error: %v", setHealthErr)
	}
	getNoOpUpdate, getNoOpUpdateErr := getScheduler.HandleSchedulerUpdate("region-a")
	if getNoOpUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) equivalent queued call returned error: %v", getNoOpUpdateErr)
	}
	if getNoOpUpdate.GetSchedulerJobKind != SchedulerJobKindUpdate {
		getTesting.Fatalf("expected update job kind for equivalent queued no-op, got %q", getNoOpUpdate.GetSchedulerJobKind)
	}
	if getScheduler.GetSchedulerQueueDepth() != 2 {
		getTesting.Fatalf("expected queue depth to remain 2 after equivalent queued no-op, got %d", getScheduler.GetSchedulerQueueDepth())
	}
}

// TestHandleSchedulerUpdateCoalesceIndexRebuildsAfterQueueCompaction verifies update coalesce lookups remain correct after cancel-driven queue compaction.
func TestHandleSchedulerUpdateCoalesceIndexRebuildsAfterQueueCompaction(getTesting *testing.T) {
	getScheduler := BuildScheduler([]SchedulerShardID{"shard-1"})
	if _, getMountErr := getScheduler.HandleSchedulerMount("region-a"); getMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-a) returned error: %v", getMountErr)
	}
	if _, getMountErr := getScheduler.HandleSchedulerMount("region-b"); getMountErr != nil {
		getTesting.Fatalf("HandleSchedulerMount(region-b) returned error: %v", getMountErr)
	}
	if _, getUpdateErr := getScheduler.HandleSchedulerUpdate("region-a"); getUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-a) returned error: %v", getUpdateErr)
	}
	if _, getUpdateErr := getScheduler.HandleSchedulerUpdate("region-b"); getUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-b) returned error: %v", getUpdateErr)
	}
	if getScheduler.GetSchedulerQueueDepth() != 4 {
		getTesting.Fatalf("expected queue depth 4 before compaction, got %d", getScheduler.GetSchedulerQueueDepth())
	}
	if !getScheduler.HandleSchedulerCancel("region-a") {
		getTesting.Fatal("expected HandleSchedulerCancel(region-a) to succeed")
	}
	if getScheduler.GetSchedulerQueueDepth() != 2 {
		getTesting.Fatalf("expected queue depth 2 after region-a compaction, got %d", getScheduler.GetSchedulerQueueDepth())
	}
	getNoOpUpdate, getNoOpUpdateErr := getScheduler.HandleSchedulerUpdate("region-b")
	if getNoOpUpdateErr != nil {
		getTesting.Fatalf("HandleSchedulerUpdate(region-b) after compaction returned error: %v", getNoOpUpdateErr)
	}
	if getNoOpUpdate.GetSchedulerJobKind != SchedulerJobKindUpdate {
		getTesting.Fatalf("expected update job kind after compaction, got %q", getNoOpUpdate.GetSchedulerJobKind)
	}
	if getScheduler.GetSchedulerQueueDepth() != 2 {
		getTesting.Fatalf("expected queue depth to remain 2 after post-compaction coalesce, got %d", getScheduler.GetSchedulerQueueDepth())
	}
}
