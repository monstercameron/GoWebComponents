package runtime2

import (
	"fmt"
	"sort"
	"strconv"
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
	if !parseRuntimeHasTrimmedNonWhitespaceText(parseWorkerKey) {
		return "", fmt.Errorf("runtime2: worker key is required")
	}
	if getSchedulerShardID, hasSchedulerShardID := parseSchedulerShardModel.storeSchedulerShardByWorkerKey[parseWorkerKey]; hasSchedulerShardID {
		return getSchedulerShardID, nil
	}
	parseSchedulerShardModel.storeSchedulerShardSequence++
	buildSchedulerShardID := formatSchedulerShardID(parseSchedulerShardModel.storeSchedulerShardSequence)
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
	if !parseRuntimeHasTrimmedNonWhitespaceText(parseRegionID) {
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
		if hasSchedulerShardID(getSchedulerShardIDs, getSchedulerShardAssigned) {
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
	if parseSchedulerShardModel == nil {
		return map[string]SchedulerShardID{}
	}
	getSchedulerRegionAssignments := make(map[string]SchedulerShardID, len(parseSchedulerShardModel.storeSchedulerShardByRegionID))
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
	hasSchedulerShardListCanonical := true
	getSchedulerShardLastID := parseSchedulerShardIDs[0]
	if !parseRuntimeHasTrimmedNonWhitespaceText(string(getSchedulerShardLastID)) {
		return nil, fmt.Errorf("runtime2: shard ID is required")
	}
	for getShardIndex := 1; getShardIndex < len(parseSchedulerShardIDs); getShardIndex++ {
		getSchedulerShardID := parseSchedulerShardIDs[getShardIndex]
		if !parseRuntimeHasTrimmedNonWhitespaceText(string(getSchedulerShardID)) {
			return nil, fmt.Errorf("runtime2: shard ID is required")
		}
		if getSchedulerShardID <= getSchedulerShardLastID {
			hasSchedulerShardListCanonical = false
		}
		getSchedulerShardLastID = getSchedulerShardID
	}
	if hasSchedulerShardListCanonical {
		return parseSchedulerShardIDs, nil
	}
	getSchedulerShardUnique := make(map[SchedulerShardID]struct{}, len(parseSchedulerShardIDs))
	getSchedulerShardList := make([]SchedulerShardID, 0, len(parseSchedulerShardIDs))
	for _, getSchedulerShardID := range parseSchedulerShardIDs {
		// IDs were already validated as non-empty above.
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

// buildSchedulerShardList canonicalizes one shard list for scheduler-owned lookups while preserving the existing nil-or-invalid builder behavior.
func buildSchedulerShardList(parseSchedulerShardIDs []SchedulerShardID) []SchedulerShardID {
	if len(parseSchedulerShardIDs) == 0 {
		return nil
	}
	getSchedulerShardIDs, parseSchedulerShardIDsErr := parseSchedulerShardList(parseSchedulerShardIDs)
	if parseSchedulerShardIDsErr != nil {
		return append([]SchedulerShardID(nil), parseSchedulerShardIDs...)
	}
	return append([]SchedulerShardID(nil), getSchedulerShardIDs...)
}

// hasSchedulerShardID reports whether the shard list currently contains the target shard identity.
func hasSchedulerShardID(parseSchedulerShardIDs []SchedulerShardID, parseSchedulerShardID SchedulerShardID) bool {
	getSchedulerShardIndex := sort.Search(len(parseSchedulerShardIDs), func(getIndex int) bool {
		return parseSchedulerShardIDs[getIndex] >= parseSchedulerShardID
	})
	return getSchedulerShardIndex < len(parseSchedulerShardIDs) && parseSchedulerShardIDs[getSchedulerShardIndex] == parseSchedulerShardID
}

// buildSchedulerRegionAssignment deterministically maps one region ID onto one available shard ID.
func buildSchedulerRegionAssignment(parseRegionID string, parseSchedulerShardIDs []SchedulerShardID) SchedulerShardID {
	if len(parseSchedulerShardIDs) == 1 {
		return parseSchedulerShardIDs[0]
	}
	buildRegionHash := getSchedulerRegionFNV64a(parseRegionID)
	buildShardIndex := buildRegionHash % uint64(len(parseSchedulerShardIDs))
	return parseSchedulerShardIDs[buildShardIndex]
}

// getSchedulerRegionFNV64a computes an FNV-1a hash for one region ID without heap allocation.
func getSchedulerRegionFNV64a(parseRegionID string) uint64 {
	const (
		getRegionHashOffset uint64 = 14695981039346656037
		getRegionHashPrime  uint64 = 1099511628211
	)
	getRegionHashValue := getRegionHashOffset
	for getRegionIndex := 0; getRegionIndex < len(parseRegionID); getRegionIndex++ {
		getRegionHashValue ^= uint64(parseRegionID[getRegionIndex])
		getRegionHashValue *= getRegionHashPrime
	}
	return getRegionHashValue
}

// formatSchedulerShardID builds one scheduler shard ID from the monotonic shard sequence.
func formatSchedulerShardID(parseSequence uint64) SchedulerShardID {
	parseBuffer := make([]byte, 0, 6+20)
	parseBuffer = append(parseBuffer, "shard-"...)
	parseBuffer = strconv.AppendUint(parseBuffer, parseSequence, 10)
	return SchedulerShardID(parseBuffer)
}
