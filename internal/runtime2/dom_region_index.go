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
	storeRegionDOMNodeByRegionID map[string]map[uint64]*RegionDOMNode
}

// RegionDOMClearResult reports how many region-local DOM node mappings were removed.
type RegionDOMClearResult struct {
	GetClearedNodeCount int
}

// BuildRegionDOMIndex creates an empty region-local DOM index.
func BuildRegionDOMIndex() *RegionDOMIndex {
	return &RegionDOMIndex{
		storeRegionDOMNodeByRegionID: make(map[string]map[uint64]*RegionDOMNode),
	}
}

// SetRegionDOMNode inserts or replaces one region-local DOM node entry.
func (parseRegionDOMIndex *RegionDOMIndex) SetRegionDOMNode(parseRegionID string, parseNodeID uint64, parseRegionDOMNode *RegionDOMNode) error {
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
	}
	return RegionDOMClearResult{
		GetClearedNodeCount: getClearedNodeCount,
	}
}
