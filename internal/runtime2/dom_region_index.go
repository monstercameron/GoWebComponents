package runtime2

import (
	"fmt"
	"strings"
)

// RegionDOMNode stores one region-local host node representation for commit and indexing tests.
type RegionDOMNode struct {
	GetNodeID       uint64
	GetTag          string
	GetText         string
	GetAttrByKey    map[string]string
	GetChildNodeIDs []uint64
	GetParentNodeID uint64
	GetNodeKey      string
}

// RegionDOMIndex maps region and node identity pairs onto region-local DOM node handles.
type RegionDOMIndex struct {
	storeRegionDOMNodeByRegionID    map[string]map[uint64]*RegionDOMNode
	storeRegionDOMVersionByRegionID map[string]uint64
}

// RegionDOMClearResult reports how many region-local DOM node mappings were removed.
type RegionDOMClearResult struct {
	GetClearedNodeCount int
}

// BuildRegionDOMIndex creates an empty region-local DOM index.
func BuildRegionDOMIndex() *RegionDOMIndex {
	return &RegionDOMIndex{
		storeRegionDOMNodeByRegionID:    make(map[string]map[uint64]*RegionDOMNode),
		storeRegionDOMVersionByRegionID: make(map[string]uint64),
	}
}

// storeRegionDOMMutationVersion bumps one region-local DOM mutation version after a successful persistent state change.
func (parseRegionDOMIndex *RegionDOMIndex) storeRegionDOMMutationVersion(parseRegionID string) {
	if parseRegionDOMIndex == nil || strings.TrimSpace(parseRegionID) == "" {
		return
	}
	parseRegionDOMIndex.storeRegionDOMVersionByRegionID[parseRegionID] = parseRegionDOMIndex.storeRegionDOMVersionByRegionID[parseRegionID] + 1
}

// SetRegionDOMNode inserts or replaces one region-local DOM node entry.
func (parseRegionDOMIndex *RegionDOMIndex) SetRegionDOMNode(parseRegionID string, parseNodeID uint64, parseRegionDOMNode *RegionDOMNode) error {
	return parseRegionDOMIndex.storeRegionDOMNode(parseRegionID, parseNodeID, parseRegionDOMNode, true)
}

// storeRegionDOMNode inserts or replaces one region-local DOM node entry and optionally bumps the region mutation version.
func (parseRegionDOMIndex *RegionDOMIndex) storeRegionDOMNode(parseRegionID string, parseNodeID uint64, parseRegionDOMNode *RegionDOMNode, isStoreMutationVersion bool) error {
	if parseRegionDOMIndex == nil {
		return fmt.Errorf("runtime2: region DOM index is nil")
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return fmt.Errorf("runtime2: region ID is required")
	}
	if parseNodeID == 0 {
		return fmt.Errorf("runtime2: node ID is required")
	}
	if parseRegionDOMNode == nil {
		return fmt.Errorf("runtime2: region DOM node is required")
	}
	getRegionDOMNodeByNodeID, hasRegionDOMNodeByNodeID := parseRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionDOMNodeByNodeID {
		getRegionDOMNodeByNodeID = make(map[uint64]*RegionDOMNode)
		parseRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID] = getRegionDOMNodeByNodeID
	}
	getRegionDOMNodeByNodeID[parseNodeID] = parseRegionDOMNode
	if isStoreMutationVersion {
		parseRegionDOMIndex.storeRegionDOMMutationVersion(parseRegionID)
	}
	return nil
}

// GetRegionDOMNode resolves one indexed region-local DOM node by region and node ID.
func (parseRegionDOMIndex *RegionDOMIndex) GetRegionDOMNode(parseRegionID string, parseNodeID uint64) (*RegionDOMNode, error) {
	if parseRegionDOMIndex == nil {
		return nil, fmt.Errorf("runtime2: region DOM index is nil")
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return nil, fmt.Errorf("runtime2: region ID is required")
	}
	if parseNodeID == 0 {
		return nil, fmt.Errorf("runtime2: node ID is required")
	}
	getRegionDOMNodeByNodeID, hasRegionDOMNodeByNodeID := parseRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionDOMNodeByNodeID {
		return nil, fmt.Errorf("runtime2: missing node ID %d for region %q", parseNodeID, parseRegionID)
	}
	getRegionDOMNode, hasRegionDOMNode := getRegionDOMNodeByNodeID[parseNodeID]
	if !hasRegionDOMNode {
		return nil, fmt.Errorf("runtime2: missing node ID %d for region %q", parseNodeID, parseRegionID)
	}
	return getRegionDOMNode, nil
}

// GetRegionDOMMutationVersion reports one monotonic mutation version for the requested region.
func (parseRegionDOMIndex *RegionDOMIndex) GetRegionDOMMutationVersion(parseRegionID string) uint64 {
	if parseRegionDOMIndex == nil || strings.TrimSpace(parseRegionID) == "" {
		return 0
	}
	return parseRegionDOMIndex.storeRegionDOMVersionByRegionID[parseRegionID]
}

// ClearRegionDOMNodes removes region-local DOM node mappings for one region, either fully or by targeted node IDs.
func (parseRegionDOMIndex *RegionDOMIndex) ClearRegionDOMNodes(parseRegionID string, parseNodeIDs []uint64) RegionDOMClearResult {
	if parseRegionDOMIndex == nil {
		return RegionDOMClearResult{}
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return RegionDOMClearResult{}
	}
	getRegionDOMNodeByNodeID, hasRegionDOMNodeByNodeID := parseRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionDOMNodeByNodeID {
		return RegionDOMClearResult{}
	}
	if len(parseNodeIDs) == 0 {
		getClearedNodeCount := len(getRegionDOMNodeByNodeID)
		delete(parseRegionDOMIndex.storeRegionDOMNodeByRegionID, parseRegionID)
		delete(parseRegionDOMIndex.storeRegionDOMVersionByRegionID, parseRegionID)
		return RegionDOMClearResult{
			GetClearedNodeCount: getClearedNodeCount,
		}
	}
	getClearedNodeCount := 0
	getSeenNodeID := make(map[uint64]bool, len(parseNodeIDs))
	for _, getNodeID := range parseNodeIDs {
		if getNodeID == 0 {
			continue
		}
		if getSeenNodeID[getNodeID] {
			continue
		}
		getSeenNodeID[getNodeID] = true
		if _, hasRegionDOMNode := getRegionDOMNodeByNodeID[getNodeID]; !hasRegionDOMNode {
			continue
		}
		delete(getRegionDOMNodeByNodeID, getNodeID)
		getClearedNodeCount++
	}
	if len(getRegionDOMNodeByNodeID) == 0 {
		delete(parseRegionDOMIndex.storeRegionDOMNodeByRegionID, parseRegionID)
		delete(parseRegionDOMIndex.storeRegionDOMVersionByRegionID, parseRegionID)
		return RegionDOMClearResult{
			GetClearedNodeCount: getClearedNodeCount,
		}
	}
	if getClearedNodeCount > 0 {
		parseRegionDOMIndex.storeRegionDOMMutationVersion(parseRegionID)
	}
	return RegionDOMClearResult{
		GetClearedNodeCount: getClearedNodeCount,
	}
}
