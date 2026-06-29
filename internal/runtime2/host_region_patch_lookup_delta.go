package runtime2

// hostRegionPatchLookupDelta stores one incremental cache update for one committed patch transaction.
type hostRegionPatchLookupDelta struct {
	getRemovedNodeIDs          []uint64
	getAddedNodeIDs            []uint64
	getClearedSiblingParentIDs []uint64
	getSiblingCountByParent    map[uint64]uint32
}

// buildRegionDOMSubtreeNodeIDs collects one indexed subtree into one flat node-ID slice.
func buildRegionDOMSubtreeNodeIDs(
	parseRegionDOMNodeByNodeID map[uint64]*RegionDOMNode,
	parseRootNodeID uint64,
) []uint64 {
	if len(parseRegionDOMNodeByNodeID) == 0 || parseRootNodeID == 0 {
		return nil
	}
	buildNodeIDs := make([]uint64, 0, 8)
	buildNodeStack := []uint64{parseRootNodeID}
	for len(buildNodeStack) > 0 {
		getNodeIndex := len(buildNodeStack) - 1
		getNodeID := buildNodeStack[getNodeIndex]
		buildNodeStack = buildNodeStack[:getNodeIndex]
		buildNodeIDs = append(buildNodeIDs, getNodeID)
		getNode, hasNode := parseRegionDOMNodeByNodeID[getNodeID]
		if !hasNode || getNode == nil || len(getNode.GetChildNodeIDs) == 0 {
			continue
		}
		for getChildIndex := len(getNode.GetChildNodeIDs) - 1; getChildIndex >= 0; getChildIndex-- {
			buildNodeStack = append(buildNodeStack, getNode.GetChildNodeIDs[getChildIndex])
		}
	}
	return buildNodeIDs
}

// hasHostRegionPatchLookupRemovedAncestor reports whether one target node already sits under one subtree removed earlier in the same transaction.
func hasHostRegionPatchLookupRemovedAncestor(
	parseRegionDOMNodeByNodeID map[uint64]*RegionDOMNode,
	parseRemovedNodeIDs map[uint64]struct{},
	parseNodeID uint64,
) bool {
	getNode, hasNode := parseRegionDOMNodeByNodeID[parseNodeID]
	if !hasNode || getNode == nil {
		return false
	}
	getParentNodeID := getNode.GetParentNodeID
	for getParentNodeID != 0 {
		if _, hasRemovedNodeID := parseRemovedNodeIDs[getParentNodeID]; hasRemovedNodeID {
			return true
		}
		getParentNode, hasParentNode := parseRegionDOMNodeByNodeID[getParentNodeID]
		if !hasParentNode || getParentNode == nil {
			return false
		}
		getParentNodeID = getParentNode.GetParentNodeID
	}
	return false
}

// hasHostRegionPatchLookupRemoveOnlyOps reports whether one transaction contains only remove-node ops, which can reuse one parent-batched lookup delta path.
func hasHostRegionPatchLookupRemoveOnlyOps(parseOps []RegionPatchOp) bool {
	if len(parseOps) == 0 {
		return false
	}
	for _, parseOp := range parseOps {
		if parseOp.GetKind != RegionPatchOpKindRemoveNode {
			return false
		}
	}
	return true
}

// buildHostRegionPatchLookupRemoveOnlyDelta captures one remove-node-only lookup delta without paying generic add/remove structural bookkeeping costs.
func (parseHostRegionAdapter *HostRegionAdapter) buildHostRegionPatchLookupRemoveOnlyDelta(
	parseTransaction RegionPatchTransaction,
	parseRegionNodeByNodeID map[uint64]*RegionDOMNode,
) (hostRegionPatchLookupDelta, bool) {
	if parseHostRegionAdapter == nil || len(parseTransaction.GetOps) == 0 || len(parseRegionNodeByNodeID) == 0 {
		return hostRegionPatchLookupDelta{}, false
	}
	buildRemoveRootNodeIDs := make(map[uint64]struct{}, len(parseTransaction.GetOps))
	buildRemovedNodeIDs := make(map[uint64]struct{}, len(parseTransaction.GetOps))
	buildRemovedCountByParent := make(map[uint64]uint32, 4)
	buildClearedSiblingParentIDs := make(map[uint64]struct{}, 4)
	for _, parseOp := range parseTransaction.GetOps {
		if parseOp.GetNodeID == 0 {
			return hostRegionPatchLookupDelta{}, false
		}
		getTargetNode, hasTargetNode := parseRegionNodeByNodeID[parseOp.GetNodeID]
		if !hasTargetNode || getTargetNode == nil {
			return hostRegionPatchLookupDelta{}, false
		}
		if hasHostRegionPatchLookupRemovedAncestor(parseRegionNodeByNodeID, buildRemoveRootNodeIDs, parseOp.GetNodeID) {
			continue
		}
		buildRemoveRootNodeIDs[parseOp.GetNodeID] = struct{}{}
		if getTargetNode.GetParentNodeID != 0 {
			buildRemovedCountByParent[getTargetNode.GetParentNodeID]++
		}
	}
	if len(buildRemoveRootNodeIDs) == 0 {
		return hostRegionPatchLookupDelta{}, false
	}
	for getRemoveRootNodeID := range buildRemoveRootNodeIDs {
		getSubtreeNodeIDs := buildRegionDOMSubtreeNodeIDs(parseRegionNodeByNodeID, getRemoveRootNodeID)
		if len(getSubtreeNodeIDs) == 0 {
			return hostRegionPatchLookupDelta{}, false
		}
		for _, getNodeID := range getSubtreeNodeIDs {
			if _, hasRemovedNodeID := buildRemovedNodeIDs[getNodeID]; hasRemovedNodeID {
				continue
			}
			buildRemovedNodeIDs[getNodeID] = struct{}{}
			if !parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
				continue
			}
			getRemovedNode, hasRemovedNode := parseRegionNodeByNodeID[getNodeID]
			if !hasRemovedNode || getRemovedNode == nil || len(getRemovedNode.GetChildNodeIDs) == 0 {
				continue
			}
			buildClearedSiblingParentIDs[getNodeID] = struct{}{}
		}
	}
	getDelta := hostRegionPatchLookupDelta{
		getRemovedNodeIDs: make([]uint64, 0, len(buildRemovedNodeIDs)),
	}
	if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
		getDelta.getSiblingCountByParent = make(map[uint64]uint32, len(buildRemovedCountByParent))
		getDelta.getClearedSiblingParentIDs = make([]uint64, 0, len(buildClearedSiblingParentIDs)+len(buildRemovedCountByParent))
		for getParentNodeID, getRemovedCount := range buildRemovedCountByParent {
			getSiblingCount := parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[getParentNodeID]
			if getSiblingCount <= getRemovedCount {
				buildClearedSiblingParentIDs[getParentNodeID] = struct{}{}
				continue
			}
			getDelta.getSiblingCountByParent[getParentNodeID] = getSiblingCount - getRemovedCount
		}
	}
	for getNodeID := range buildRemovedNodeIDs {
		getDelta.getRemovedNodeIDs = append(getDelta.getRemovedNodeIDs, getNodeID)
	}
	for getParentNodeID := range buildClearedSiblingParentIDs {
		getDelta.getClearedSiblingParentIDs = append(getDelta.getClearedSiblingParentIDs, getParentNodeID)
	}
	return getDelta, true
}

// buildHostRegionPatchLookupDelta captures one incremental structural lookup delta before commit so the hot cache can survive structural patch traffic.
func (parseHostRegionAdapter *HostRegionAdapter) buildHostRegionPatchLookupDelta(
	parseTransaction RegionPatchTransaction,
) (hostRegionPatchLookupDelta, bool) {
	if parseHostRegionAdapter == nil ||
		!parseHostRegionAdapter.hasHostRegionPatchLookupCache ||
		parseHostRegionAdapter.storeRegionDOMIndexHandle == nil {
		return hostRegionPatchLookupDelta{}, false
	}
	getRegionNodeByNodeID, hasRegionNodeByNodeID := parseHostRegionAdapter.storeRegionDOMIndexHandle.storeRegionDOMNodeByRegionID[string(parseHostRegionAdapter.storeRegionInstanceID)]
	if !hasRegionNodeByNodeID {
		return hostRegionPatchLookupDelta{}, false
	}
	if hasHostRegionPatchLookupRemoveOnlyOps(parseTransaction.GetOps) {
		return parseHostRegionAdapter.buildHostRegionPatchLookupRemoveOnlyDelta(parseTransaction, getRegionNodeByNodeID)
	}
	buildRemovedNodeIDSet := map[uint64]struct{}{}
	buildAddedNodeIDSet := map[uint64]struct{}{}
	buildClearedSiblingParentIDSet := map[uint64]struct{}{}
	buildSiblingCountByParent := map[uint64]uint32{}
	hasStructuralOp := false
	getSiblingCountByParent := func(parseParentNodeID uint64) uint32 {
		if parseParentNodeID == 0 || !parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
			return 0
		}
		if getSiblingCount, hasSiblingCount := buildSiblingCountByParent[parseParentNodeID]; hasSiblingCount {
			return getSiblingCount
		}
		if _, hasClearedSiblingParentID := buildClearedSiblingParentIDSet[parseParentNodeID]; hasClearedSiblingParentID {
			return 0
		}
		return parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[parseParentNodeID]
	}
	for _, getOp := range parseTransaction.GetOps {
		switch getOp.GetKind {
		case RegionPatchOpKindSetText,
			RegionPatchOpKindSetAttr,
			RegionPatchOpKindSetStyle,
			RegionPatchOpKindRemoveAttr,
			RegionPatchOpKindRemoveStyle,
			RegionPatchOpKindMoveKeyedNode:
			continue
		case RegionPatchOpKindInsertNode:
			hasStructuralOp = true
			if getOp.GetInsertNode == nil || getOp.GetInsertNode.GetNodeID == 0 {
				return hostRegionPatchLookupDelta{}, false
			}
			delete(buildRemovedNodeIDSet, getOp.GetInsertNode.GetNodeID)
			buildAddedNodeIDSet[getOp.GetInsertNode.GetNodeID] = struct{}{}
			if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache && getOp.GetParentNodeID != 0 {
				buildSiblingCountByParent[getOp.GetParentNodeID] = getSiblingCountByParent(getOp.GetParentNodeID) + 1
				delete(buildClearedSiblingParentIDSet, getOp.GetParentNodeID)
			}
			if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache && len(getOp.GetInsertNode.GetChildNodeIDs) > 0 {
				buildSiblingCountByParent[getOp.GetInsertNode.GetNodeID] = uint32(len(getOp.GetInsertNode.GetChildNodeIDs))
				delete(buildClearedSiblingParentIDSet, getOp.GetInsertNode.GetNodeID)
			}
		case RegionPatchOpKindRemoveNode:
			hasStructuralOp = true
			if getOp.GetNodeID == 0 || hasHostRegionPatchLookupRemovedAncestor(getRegionNodeByNodeID, buildRemovedNodeIDSet, getOp.GetNodeID) {
				continue
			}
			getTargetNode, hasTargetNode := getRegionNodeByNodeID[getOp.GetNodeID]
			if !hasTargetNode || getTargetNode == nil {
				return hostRegionPatchLookupDelta{}, false
			}
			getRemovedNodeIDs := buildRegionDOMSubtreeNodeIDs(getRegionNodeByNodeID, getOp.GetNodeID)
			if len(getRemovedNodeIDs) == 0 {
				return hostRegionPatchLookupDelta{}, false
			}
			for _, getRemovedNodeID := range getRemovedNodeIDs {
				delete(buildAddedNodeIDSet, getRemovedNodeID)
				buildRemovedNodeIDSet[getRemovedNodeID] = struct{}{}
				if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
					delete(buildSiblingCountByParent, getRemovedNodeID)
					buildClearedSiblingParentIDSet[getRemovedNodeID] = struct{}{}
				}
			}
			if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache &&
				getTargetNode.GetParentNodeID != 0 {
				if _, hasRemovedParentNodeID := buildRemovedNodeIDSet[getTargetNode.GetParentNodeID]; !hasRemovedParentNodeID {
					getParentSiblingCount := getSiblingCountByParent(getTargetNode.GetParentNodeID)
					if getParentSiblingCount <= 1 {
						delete(buildSiblingCountByParent, getTargetNode.GetParentNodeID)
						buildClearedSiblingParentIDSet[getTargetNode.GetParentNodeID] = struct{}{}
					} else {
						buildSiblingCountByParent[getTargetNode.GetParentNodeID] = getParentSiblingCount - 1
						delete(buildClearedSiblingParentIDSet, getTargetNode.GetParentNodeID)
					}
				}
			}
		case RegionPatchOpKindReplaceSubtree:
			hasStructuralOp = true
			if getOp.GetNodeID == 0 || hasHostRegionPatchLookupRemovedAncestor(getRegionNodeByNodeID, buildRemovedNodeIDSet, getOp.GetNodeID) {
				continue
			}
			if _, hasTargetNode := getRegionNodeByNodeID[getOp.GetNodeID]; !hasTargetNode {
				return hostRegionPatchLookupDelta{}, false
			}
			getRemovedNodeIDs := buildRegionDOMSubtreeNodeIDs(getRegionNodeByNodeID, getOp.GetNodeID)
			if len(getRemovedNodeIDs) == 0 {
				return hostRegionPatchLookupDelta{}, false
			}
			getReplacementTree, parseReplacementTreeErr := ParseCanonicalRenderTree(getOp.GetReplaceSubtreeIR)
			if parseReplacementTreeErr != nil {
				return hostRegionPatchLookupDelta{}, false
			}
			for _, getRemovedNodeID := range getRemovedNodeIDs {
				delete(buildAddedNodeIDSet, getRemovedNodeID)
				buildRemovedNodeIDSet[getRemovedNodeID] = struct{}{}
				if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
					delete(buildSiblingCountByParent, getRemovedNodeID)
					buildClearedSiblingParentIDSet[getRemovedNodeID] = struct{}{}
				}
			}
			for getNodeID, getNode := range getReplacementTree.getNodeByID {
				delete(buildRemovedNodeIDSet, getNodeID)
				buildAddedNodeIDSet[getNodeID] = struct{}{}
				if !parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
					continue
				}
				if len(getNode.getChildNodeIDs) == 0 {
					delete(buildSiblingCountByParent, getNodeID)
					buildClearedSiblingParentIDSet[getNodeID] = struct{}{}
					continue
				}
				buildSiblingCountByParent[getNodeID] = uint32(len(getNode.getChildNodeIDs))
				delete(buildClearedSiblingParentIDSet, getNodeID)
			}
		default:
			return hostRegionPatchLookupDelta{}, false
		}
	}
	if !hasStructuralOp {
		return hostRegionPatchLookupDelta{}, false
	}
	getDelta := hostRegionPatchLookupDelta{
		getRemovedNodeIDs:          make([]uint64, 0, len(buildRemovedNodeIDSet)),
		getAddedNodeIDs:            make([]uint64, 0, len(buildAddedNodeIDSet)),
		getClearedSiblingParentIDs: make([]uint64, 0, len(buildClearedSiblingParentIDSet)),
		getSiblingCountByParent:    buildSiblingCountByParent,
	}
	for getNodeID := range buildRemovedNodeIDSet {
		getDelta.getRemovedNodeIDs = append(getDelta.getRemovedNodeIDs, getNodeID)
	}
	for getNodeID := range buildAddedNodeIDSet {
		getDelta.getAddedNodeIDs = append(getDelta.getAddedNodeIDs, getNodeID)
	}
	for getParentNodeID := range buildClearedSiblingParentIDSet {
		getDelta.getClearedSiblingParentIDs = append(getDelta.getClearedSiblingParentIDs, getParentNodeID)
	}
	return getDelta, true
}

// applyHostRegionPatchLookupDelta applies one precomputed structural cache delta after a successful transaction commit.
func (parseHostRegionAdapter *HostRegionAdapter) applyHostRegionPatchLookupDelta(parseDelta hostRegionPatchLookupDelta) bool {
	if parseHostRegionAdapter == nil || !parseHostRegionAdapter.hasHostRegionPatchLookupCache {
		return false
	}
	for _, getNodeID := range parseDelta.getRemovedNodeIDs {
		delete(parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs, getNodeID)
	}
	for _, getNodeID := range parseDelta.getAddedNodeIDs {
		parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[getNodeID] = struct{}{}
	}
	if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
		for _, getParentNodeID := range parseDelta.getClearedSiblingParentIDs {
			delete(parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent, getParentNodeID)
		}
		for getParentNodeID, getSiblingCount := range parseDelta.getSiblingCountByParent {
			if getSiblingCount == 0 {
				delete(parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent, getParentNodeID)
				continue
			}
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[getParentNodeID] = getSiblingCount
		}
	}
	parseHostRegionAdapter.storeHostRegionPatchLookupVersion = parseHostRegionAdapter.storeRegionDOMIndexHandle.GetRegionDOMMutationVersion(string(parseHostRegionAdapter.storeRegionInstanceID))
	return true
}
