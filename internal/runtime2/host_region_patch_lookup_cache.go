package runtime2

// clearHostRegionPatchLookupCache clears cached patch-parse lookup state for the adapter-owned region.
func (parseHostRegionAdapter *HostRegionAdapter) clearHostRegionPatchLookupCache() {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs = nil
	parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent = nil
	parseHostRegionAdapter.storeHostRegionPatchLookupVersion = 0
	parseHostRegionAdapter.hasHostRegionPatchLookupCache = false
	parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache = false
}

// storeHostRegionPatchLookupState stores known-node and sibling-count lookup state for the adapter-owned region.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionPatchLookupState(
	parseKnownNodeIDs map[uint64]struct{},
	parseSiblingCountByParent map[uint64]uint32,
	parseHasSiblingCount bool,
) {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs = parseKnownNodeIDs
	if parseHasSiblingCount {
		parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent = parseSiblingCountByParent
	} else {
		parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent = nil
	}
	parseHostRegionAdapter.storeHostRegionPatchLookupVersion = parseHostRegionAdapter.storeRegionDOMIndexHandle.GetRegionDOMMutationVersion(string(parseHostRegionAdapter.storeRegionInstanceID))
	parseHostRegionAdapter.hasHostRegionPatchLookupCache = parseKnownNodeIDs != nil
	parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache = parseHasSiblingCount && parseSiblingCountByParent != nil
}

// getHostRegionPatchLookupState returns cached patch-parse lookup state or rebuilds it when the region DOM version changed.
func (parseHostRegionAdapter *HostRegionAdapter) getHostRegionPatchLookupState(parseHasPatchKeyedMove bool) (map[uint64]struct{}, map[uint64]uint32) {
	if parseHostRegionAdapter == nil || parseHostRegionAdapter.storeRegionDOMIndexHandle == nil {
		return map[uint64]struct{}{}, map[uint64]uint32{}
	}
	parseRegionID := string(parseHostRegionAdapter.storeRegionInstanceID)
	parseLookupVersion := parseHostRegionAdapter.storeRegionDOMIndexHandle.GetRegionDOMMutationVersion(parseRegionID)
	if parseHostRegionAdapter.hasHostRegionPatchLookupCache &&
		parseHostRegionAdapter.storeHostRegionPatchLookupVersion == parseLookupVersion {
		if !parseHasPatchKeyedMove || parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache {
			return parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs, parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent
		}
	}
	if parseHasPatchKeyedMove {
		parseKnownNodeIDs, parseSiblingCountByParent := BuildRegionDOMPatchLookupMaps(parseHostRegionAdapter.storeRegionDOMIndexHandle, parseRegionID)
		parseHostRegionAdapter.storeHostRegionPatchLookupState(parseKnownNodeIDs, parseSiblingCountByParent, true)
		return parseKnownNodeIDs, parseSiblingCountByParent
	}
	parseKnownNodeIDs := BuildKnownNodeIDsForRegionDOMIndex(parseHostRegionAdapter.storeRegionDOMIndexHandle, parseRegionID)
	parseHostRegionAdapter.storeHostRegionPatchLookupState(parseKnownNodeIDs, nil, false)
	return parseKnownNodeIDs, nil
}

// canHostRegionPatchLookupApplyTransaction reports whether one committed transaction can update the hot lookup cache incrementally without precomputing one structural delta.
func (parseHostRegionAdapter *HostRegionAdapter) canHostRegionPatchLookupApplyTransaction(parseTransaction RegionPatchTransaction) bool {
	if parseHostRegionAdapter == nil || !parseHostRegionAdapter.hasHostRegionPatchLookupCache {
		return false
	}
	for _, parseOp := range parseTransaction.GetOps {
		switch parseOp.GetKind {
		case RegionPatchOpKindSetText,
			RegionPatchOpKindSetAttr,
			RegionPatchOpKindSetStyle,
			RegionPatchOpKindRemoveAttr,
			RegionPatchOpKindRemoveStyle,
			RegionPatchOpKindMoveKeyedNode:
			continue
		case RegionPatchOpKindInsertNode:
			if parseOp.GetInsertNode == nil || parseOp.GetInsertNode.GetNodeID == 0 {
				return false
			}
		case RegionPatchOpKindRemoveNode,
			RegionPatchOpKindReplaceSubtree:
			return false
		default:
			return false
		}
	}
	return true
}

// applyHostRegionPatchLookupTransaction updates cached lookup state after a successful patch commit when the committed ops preserve incremental cache correctness.
func (parseHostRegionAdapter *HostRegionAdapter) applyHostRegionPatchLookupTransaction(parseTransaction RegionPatchTransaction) bool {
	if !parseHostRegionAdapter.canHostRegionPatchLookupApplyTransaction(parseTransaction) {
		return false
	}
	for _, parseOp := range parseTransaction.GetOps {
		switch parseOp.GetKind {
		case RegionPatchOpKindSetText,
			RegionPatchOpKindSetAttr,
			RegionPatchOpKindSetStyle,
			RegionPatchOpKindRemoveAttr,
			RegionPatchOpKindRemoveStyle,
			RegionPatchOpKindMoveKeyedNode:
			continue
		case RegionPatchOpKindInsertNode:
			parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[parseOp.GetInsertNode.GetNodeID] = struct{}{}
			if parseHostRegionAdapter.hasHostRegionPatchSiblingCountCache && parseOp.GetParentNodeID != 0 {
				parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[parseOp.GetParentNodeID] = parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[parseOp.GetParentNodeID] + 1
			}
		}
	}
	parseHostRegionAdapter.storeHostRegionPatchLookupVersion = parseHostRegionAdapter.storeRegionDOMIndexHandle.GetRegionDOMMutationVersion(string(parseHostRegionAdapter.storeRegionInstanceID))
	return true
}
