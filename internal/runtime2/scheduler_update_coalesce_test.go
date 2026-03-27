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
