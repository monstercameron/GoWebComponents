package runtime2

import "testing"

// TestGetSchedulerRegionShardIDKeepsStableAssignmentForOneRegion verifies one region keeps one shard until policy allows reassignment.
func TestGetSchedulerRegionShardIDKeepsStableAssignmentForOneRegion(getTesting *testing.T) {
	getSchedulerShardModel := BuildSchedulerShardModel()
	getSchedulerShardIDs := []SchedulerShardID{"shard-1", "shard-2", "shard-3"}
	getRegionShardFirst, getRegionShardFirstErr := getSchedulerShardModel.GetSchedulerRegionShardID("region-a", getSchedulerShardIDs, SchedulerAssignmentPolicyKeep)
	if getRegionShardFirstErr != nil {
		getTesting.Fatalf("GetSchedulerRegionShardID(region-a) first call returned error: %v", getRegionShardFirstErr)
	}
	getRegionShardSecond, getRegionShardSecondErr := getSchedulerShardModel.GetSchedulerRegionShardID("region-a", getSchedulerShardIDs, SchedulerAssignmentPolicyKeep)
	if getRegionShardSecondErr != nil {
		getTesting.Fatalf("GetSchedulerRegionShardID(region-a) second call returned error: %v", getRegionShardSecondErr)
	}
	if getRegionShardFirst != getRegionShardSecond {
		getTesting.Fatalf("expected stable assignment for region-a, first=%q second=%q", getRegionShardFirst, getRegionShardSecond)
	}
}

// TestGetSchedulerRegionShardIDAllowsDifferentRegionsToSpreadAcrossShards verifies multiple regions can map to different shards.
func TestGetSchedulerRegionShardIDAllowsDifferentRegionsToSpreadAcrossShards(getTesting *testing.T) {
	getSchedulerShardModel := BuildSchedulerShardModel()
	getSchedulerShardIDs := []SchedulerShardID{"shard-1", "shard-2"}
	getUniqueShardSet := map[SchedulerShardID]struct{}{}
	getRegionIDs := []string{"region-a", "region-b", "region-c", "region-d", "region-e"}
	for _, getRegionID := range getRegionIDs {
		getRegionShardID, getRegionShardErr := getSchedulerShardModel.GetSchedulerRegionShardID(getRegionID, getSchedulerShardIDs, SchedulerAssignmentPolicyKeep)
		if getRegionShardErr != nil {
			getTesting.Fatalf("GetSchedulerRegionShardID(%s) returned error: %v", getRegionID, getRegionShardErr)
		}
		getUniqueShardSet[getRegionShardID] = struct{}{}
	}
	if len(getUniqueShardSet) < 2 {
		getTesting.Fatalf("expected at least two shard assignments across regions, got %d", len(getUniqueShardSet))
	}
}

// TestGetSchedulerRegionShardIDOnlyChangesWhenPolicyAllows verifies reassignment needs an explicit rebalance or repair policy.
func TestGetSchedulerRegionShardIDOnlyChangesWhenPolicyAllows(getTesting *testing.T) {
	getSchedulerShardModel := BuildSchedulerShardModel()
	getInitialShardIDs := []SchedulerShardID{"shard-1", "shard-2"}
	getRegionShardInitial, getRegionShardInitialErr := getSchedulerShardModel.GetSchedulerRegionShardID("region-a", getInitialShardIDs, SchedulerAssignmentPolicyKeep)
	if getRegionShardInitialErr != nil {
		getTesting.Fatalf("GetSchedulerRegionShardID(region-a) initial call returned error: %v", getRegionShardInitialErr)
	}
	getLimitedShardIDs := []SchedulerShardID{"shard-1"}
	if getRegionShardInitial == "shard-1" {
		getLimitedShardIDs = []SchedulerShardID{"shard-2"}
	}
	_, getRegionShardKeepErr := getSchedulerShardModel.GetSchedulerRegionShardID("region-a", getLimitedShardIDs, SchedulerAssignmentPolicyKeep)
	if getRegionShardKeepErr == nil {
		getTesting.Fatal("expected keep policy to reject assignment change when assigned shard is unavailable")
	}
	getRegionShardRepair, getRegionShardRepairErr := getSchedulerShardModel.GetSchedulerRegionShardID("region-a", getLimitedShardIDs, SchedulerAssignmentPolicyRepair)
	if getRegionShardRepairErr != nil {
		getTesting.Fatalf("GetSchedulerRegionShardID(region-a) repair call returned error: %v", getRegionShardRepairErr)
	}
	if getRegionShardRepair == getRegionShardInitial {
		getTesting.Fatalf("expected repair policy to change assignment, initial=%q repair=%q", getRegionShardInitial, getRegionShardRepair)
	}
}

// TestGetSchedulerRegionShardIDKeepHandlesUnsortedDuplicateShardList verifies keep-policy availability checks work for unsorted duplicate shard inputs.
func TestGetSchedulerRegionShardIDKeepHandlesUnsortedDuplicateShardList(getTesting *testing.T) {
	getSchedulerShardModel := BuildSchedulerShardModel()
	getSchedulerShardIDs := []SchedulerShardID{"shard-2", "shard-1", "shard-2"}
	getRegionShardInitial, getRegionShardInitialErr := getSchedulerShardModel.GetSchedulerRegionShardID("region-a", getSchedulerShardIDs, SchedulerAssignmentPolicyKeep)
	if getRegionShardInitialErr != nil {
		getTesting.Fatalf("GetSchedulerRegionShardID(region-a) initial call returned error: %v", getRegionShardInitialErr)
	}
	getRegionShardSecond, getRegionShardSecondErr := getSchedulerShardModel.GetSchedulerRegionShardID("region-a", getSchedulerShardIDs, SchedulerAssignmentPolicyKeep)
	if getRegionShardSecondErr != nil {
		getTesting.Fatalf("GetSchedulerRegionShardID(region-a) second keep call returned error: %v", getRegionShardSecondErr)
	}
	if getRegionShardSecond != getRegionShardInitial {
		getTesting.Fatalf("expected stable keep assignment with unsorted duplicate shard list, initial=%q second=%q", getRegionShardInitial, getRegionShardSecond)
	}
}
