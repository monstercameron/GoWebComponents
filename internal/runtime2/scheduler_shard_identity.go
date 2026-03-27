package runtime2

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

// SchedulerShardID identifies one live scheduler shard assignment.
type SchedulerShardID string

// SchedulerShardModel tracks stable worker-to-shard identity assignments.
type SchedulerShardModel struct {
	storeSchedulerShardByWorkerKey map[string]SchedulerShardID
	storeSchedulerShardByRegionID  map[string]SchedulerShardID
	storeSchedulerShardSequence    uint64
}

// SchedulerAssignmentPolicy controls when region-to-shard assignments may change.
type SchedulerAssignmentPolicy uint8

const (
	schedulerAssignmentPolicyInvalid SchedulerAssignmentPolicy = iota

	SchedulerAssignmentPolicyKeep
	SchedulerAssignmentPolicyRebalance
	SchedulerAssignmentPolicyRepair
)

// BuildSchedulerShardModel creates a scheduler shard identity model with empty assignments.
func BuildSchedulerShardModel() *SchedulerShardModel {
	return &SchedulerShardModel{
		storeSchedulerShardByWorkerKey: make(map[string]SchedulerShardID),
		storeSchedulerShardByRegionID:  make(map[string]SchedulerShardID),
	}
}

// GetSchedulerShardID resolves or allocates a stable shard identity for one live worker key.
func (parseSchedulerShardModel *SchedulerShardModel) GetSchedulerShardID(parseWorkerKey string) (SchedulerShardID, error) {
	if parseSchedulerShardModel == nil {
		return "", fmt.Errorf("runtime2: scheduler shard model is nil")
	}
	if strings.TrimSpace(parseWorkerKey) == "" {
		return "", fmt.Errorf("runtime2: worker key is required")
	}
	if getSchedulerShardID, hasSchedulerShardID := parseSchedulerShardModel.storeSchedulerShardByWorkerKey[parseWorkerKey]; hasSchedulerShardID {
		return getSchedulerShardID, nil
	}
	parseSchedulerShardModel.storeSchedulerShardSequence++
	buildSchedulerShardID := SchedulerShardID(fmt.Sprintf("shard-%d", parseSchedulerShardModel.storeSchedulerShardSequence))
	parseSchedulerShardModel.storeSchedulerShardByWorkerKey[parseWorkerKey] = buildSchedulerShardID
	return buildSchedulerShardID, nil
}

// ClearSchedulerShardID removes one worker-to-shard assignment and returns the disposed shard identity.
func (parseSchedulerShardModel *SchedulerShardModel) ClearSchedulerShardID(parseWorkerKey string) (SchedulerShardID, bool) {
	if parseSchedulerShardModel == nil {
		return "", false
	}
	getSchedulerShardID, hasSchedulerShardID := parseSchedulerShardModel.storeSchedulerShardByWorkerKey[parseWorkerKey]
	if !hasSchedulerShardID {
		return "", false
	}
	delete(parseSchedulerShardModel.storeSchedulerShardByWorkerKey, parseWorkerKey)
	return getSchedulerShardID, true
}

// GetSchedulerRegionShardID resolves a deterministic shard assignment for one region based on policy and live shard set.
func (parseSchedulerShardModel *SchedulerShardModel) GetSchedulerRegionShardID(parseRegionID string, parseSchedulerShardIDs []SchedulerShardID, parsePolicy SchedulerAssignmentPolicy) (SchedulerShardID, error) {
	if parseSchedulerShardModel == nil {
		return "", fmt.Errorf("runtime2: scheduler shard model is nil")
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return "", fmt.Errorf("runtime2: region ID is required")
	}
	if parsePolicy == schedulerAssignmentPolicyInvalid {
		return "", fmt.Errorf("runtime2: assignment policy is required")
	}
	if parsePolicy != SchedulerAssignmentPolicyKeep && parsePolicy != SchedulerAssignmentPolicyRebalance && parsePolicy != SchedulerAssignmentPolicyRepair {
		return "", fmt.Errorf("runtime2: assignment policy %d is unsupported", parsePolicy)
	}
	getSchedulerShardIDs, getSchedulerShardIDsErr := parseSchedulerShardList(parseSchedulerShardIDs)
	if getSchedulerShardIDsErr != nil {
		return "", getSchedulerShardIDsErr
	}
	getSchedulerShardAssigned, hasSchedulerShardAssigned := parseSchedulerShardModel.storeSchedulerShardByRegionID[parseRegionID]
	if hasSchedulerShardAssigned && parsePolicy == SchedulerAssignmentPolicyKeep {
		if hasSchedulerShardAvailable := hasSchedulerShardID(getSchedulerShardIDs, getSchedulerShardAssigned); hasSchedulerShardAvailable {
			return getSchedulerShardAssigned, nil
		}
		return "", fmt.Errorf("runtime2: assigned shard %q for region %q is unavailable without rebalance or repair policy", getSchedulerShardAssigned, parseRegionID)
	}
	buildSchedulerShardAssigned := buildSchedulerRegionAssignment(parseRegionID, getSchedulerShardIDs)
	parseSchedulerShardModel.storeSchedulerShardByRegionID[parseRegionID] = buildSchedulerShardAssigned
	return buildSchedulerShardAssigned, nil
}

// ClearSchedulerRegionShardID removes one region-to-shard assignment and reports whether it existed.
func (parseSchedulerShardModel *SchedulerShardModel) ClearSchedulerRegionShardID(parseRegionID string) bool {
	if parseSchedulerShardModel == nil {
		return false
	}
	if _, hasSchedulerShardAssigned := parseSchedulerShardModel.storeSchedulerShardByRegionID[parseRegionID]; !hasSchedulerShardAssigned {
		return false
	}
	delete(parseSchedulerShardModel.storeSchedulerShardByRegionID, parseRegionID)
	return true
}

// GetSchedulerRegionAssignedShardID looks up the current shard assignment for one region without allocating a new assignment.
func (parseSchedulerShardModel *SchedulerShardModel) GetSchedulerRegionAssignedShardID(parseRegionID string) (SchedulerShardID, bool) {
	if parseSchedulerShardModel == nil {
		return "", false
	}
	getSchedulerShardID, hasSchedulerShardID := parseSchedulerShardModel.storeSchedulerShardByRegionID[parseRegionID]
	if !hasSchedulerShardID {
		return "", false
	}
	return getSchedulerShardID, true
}

// GetSchedulerRegionAssignments returns a copy of current region-to-shard assignments.
func (parseSchedulerShardModel *SchedulerShardModel) GetSchedulerRegionAssignments() map[string]SchedulerShardID {
	getSchedulerRegionAssignments := make(map[string]SchedulerShardID)
	if parseSchedulerShardModel == nil {
		return getSchedulerRegionAssignments
	}
	for getRegionID, getSchedulerShardID := range parseSchedulerShardModel.storeSchedulerShardByRegionID {
		getSchedulerRegionAssignments[getRegionID] = getSchedulerShardID
	}
	return getSchedulerRegionAssignments
}

// parseSchedulerShardList validates and canonicalizes shard IDs for deterministic region assignment.
func parseSchedulerShardList(parseSchedulerShardIDs []SchedulerShardID) ([]SchedulerShardID, error) {
	if len(parseSchedulerShardIDs) == 0 {
		return nil, fmt.Errorf("runtime2: at least one shard ID is required")
	}
	getSchedulerShardUnique := make(map[SchedulerShardID]struct{}, len(parseSchedulerShardIDs))
	getSchedulerShardList := make([]SchedulerShardID, 0, len(parseSchedulerShardIDs))
	for _, getSchedulerShardID := range parseSchedulerShardIDs {
		if strings.TrimSpace(string(getSchedulerShardID)) == "" {
			return nil, fmt.Errorf("runtime2: shard ID is required")
		}
		if _, hasSchedulerShardID := getSchedulerShardUnique[getSchedulerShardID]; hasSchedulerShardID {
			continue
		}
		getSchedulerShardUnique[getSchedulerShardID] = struct{}{}
		getSchedulerShardList = append(getSchedulerShardList, getSchedulerShardID)
	}
	sort.Slice(getSchedulerShardList, func(getLeftIndex, getRightIndex int) bool {
		return getSchedulerShardList[getLeftIndex] < getSchedulerShardList[getRightIndex]
	})
	return getSchedulerShardList, nil
}

// hasSchedulerShardID reports whether the shard list currently contains the target shard identity.
func hasSchedulerShardID(parseSchedulerShardIDs []SchedulerShardID, parseSchedulerShardID SchedulerShardID) bool {
	for _, getSchedulerShardID := range parseSchedulerShardIDs {
		if getSchedulerShardID == parseSchedulerShardID {
			return true
		}
	}
	return false
}

// buildSchedulerRegionAssignment deterministically maps one region ID onto one available shard ID.
func buildSchedulerRegionAssignment(parseRegionID string, parseSchedulerShardIDs []SchedulerShardID) SchedulerShardID {
	buildRegionHash := fnv.New64a()
	_, _ = buildRegionHash.Write([]byte(parseRegionID))
	buildShardIndex := buildRegionHash.Sum64() % uint64(len(parseSchedulerShardIDs))
	return parseSchedulerShardIDs[buildShardIndex]
}
