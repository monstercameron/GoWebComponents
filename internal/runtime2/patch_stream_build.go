package runtime2

import (
	"fmt"
	"sort"
	"strings"
)

type patchInsertBuildOp struct {
	getParentNodeID uint64
	getNodeID       uint64
	getNodeKind     RenderNodeKind
	getNodeTag      string
	getNodeText     string
	getNodeKey      string
	hasNodeKey      bool
	getAnchorNodeID uint64
}

type patchSetTextBuildOp struct {
	getTargetNodeID uint64
	getText         string
}

type patchSetAttrBuildOp struct {
	getTargetNodeID uint64
	getKind         RenderPropKind
	getKey          string
	getValue        string
}

type patchSetStyleBuildOp struct {
	getTargetNodeID uint64
	getStyleValue   string
}

type patchRemoveAttrBuildOp struct {
	getTargetNodeID uint64
	getKey          string
}

type patchRemoveStyleBuildOp struct {
	getTargetNodeID uint64
}

// buildPatchStringCapacityFromPatchOps computes one stable capacity hint for patch string accumulation.
func buildPatchStringCapacityFromPatchOps(
	parseInsertOps []patchInsertBuildOp,
	parseSetTextOps []patchSetTextBuildOp,
	parseSetAttrOps []patchSetAttrBuildOp,
	parseSetStyleOps []patchSetStyleBuildOp,
	parseRemoveAttrOps []patchRemoveAttrBuildOp,
) int {
	parseCapacity := len(parseSetTextOps) + len(parseSetStyleOps) + len(parseRemoveAttrOps) + (len(parseSetAttrOps) * 2)
	for _, getInsertOp := range parseInsertOps {
		if getInsertOp.getNodeKind == RenderNodeKindText || getInsertOp.getNodeKind == RenderNodeKindHostElement {
			parseCapacity++
		}
		if getInsertOp.hasNodeKey {
			parseCapacity++
		}
	}
	return parseCapacity
}

// hasCanonicalNodeOrderStructuralDelta reports whether one sibling order references inserted or removed node IDs.
func hasCanonicalNodeOrderStructuralDelta(
	parseNodeIDs []uint64,
	parseRemovedNodeIDs map[uint64]struct{},
	parseInsertedNodeIDs map[uint64]struct{},
) bool {
	for _, getNodeID := range parseNodeIDs {
		if _, hasRemovedNode := parseRemovedNodeIDs[getNodeID]; hasRemovedNode {
			return true
		}
		if _, hasInsertedNode := parseInsertedNodeIDs[getNodeID]; hasInsertedNode {
			return true
		}
	}
	return false
}

// buildCanonicalPatchKeyedMoveOps computes keyed-move operations for canonical sibling reorders.
func buildCanonicalPatchKeyedMoveOps(
	parseRegionID string,
	parsePreviousTree canonicalRenderTree,
	parseNextTree canonicalRenderTree,
	parseRemovedNodeIDs map[uint64]struct{},
	parseInsertedNodeIDs map[uint64]struct{},
	parseHasStructuralNodeDelta bool,
) ([]PatchKeyedMoveOpRaw, error) {
	buildMoveOps := []PatchKeyedMoveOpRaw{}
	hasLoggedMoveSiblingWarn := false
	for getParentNodeID, getPreviousNode := range parsePreviousTree.getNodeByID {
		getNextNode, hasNextNode := parseNextTree.getNodeByID[getParentNodeID]
		if !hasNextNode {
			continue
		}
		buildCurrentOrder := getPreviousNode.getChildNodeIDs
		buildTargetOrder := getNextNode.getChildNodeIDs
		hasCurrentOrderScratch := false
		if parseHasStructuralNodeDelta {
			hasCurrentOrderStructuralDelta := hasCanonicalNodeOrderStructuralDelta(
				buildCurrentOrder,
				parseRemovedNodeIDs,
				parseInsertedNodeIDs,
			)
			hasTargetOrderStructuralDelta := hasCanonicalNodeOrderStructuralDelta(
				buildTargetOrder,
				parseRemovedNodeIDs,
				parseInsertedNodeIDs,
			)
			if hasCurrentOrderStructuralDelta || hasTargetOrderStructuralDelta {
				buildCurrentOrder = parseFilterCanonicalExistingOrder(buildCurrentOrder, parseRemovedNodeIDs, parseInsertedNodeIDs)
				buildTargetOrder = parseFilterCanonicalExistingOrder(buildTargetOrder, parseRemovedNodeIDs, parseInsertedNodeIDs)
				hasCurrentOrderScratch = true
			}
		}
		if len(buildTargetOrder) > getPatchMoveSiblingHardLimit {
			return nil, fmt.Errorf(
				"runtime2: keyed-move sibling count %d exceeds guard limit %d for parent %d",
				len(buildTargetOrder),
				getPatchMoveSiblingHardLimit,
				getParentNodeID,
			)
		}
		if len(buildTargetOrder) > getPatchMoveSiblingWarnLimit && !hasLoggedMoveSiblingWarn {
			hasLoggedMoveSiblingWarn = true
			fmt.Printf(
				"WARN: runtime2 patch diff region=%s keyed-move sibling count=%d exceeds soft limit=%d\n",
				parseRegionID,
				len(buildTargetOrder),
				getPatchMoveSiblingWarnLimit,
			)
		}
		if parseHasCanonicalNodeOrderEqual(buildCurrentOrder, buildTargetOrder) {
			continue
		}
		if !getNextNode.hasKeyedChild {
			continue
		}
		if !hasCurrentOrderScratch {
			// Keep canonical trees immutable while avoiding unnecessary filtered-order allocations.
			buildCurrentOrder = append([]uint64(nil), buildCurrentOrder...)
		}
		buildCurrentIndexByNode := parseBuildCanonicalSiblingIndexMap(buildCurrentOrder)
		for parseTargetIndex, getTargetNodeID := range buildTargetOrder {
			getTargetNode := parseNextTree.getNodeByID[getTargetNodeID]
			if !getTargetNode.hasKey {
				continue
			}
			parseCurrentIndex, hasCurrentIndex := buildCurrentIndexByNode[getTargetNodeID]
			if !hasCurrentIndex || parseCurrentIndex == parseTargetIndex {
				continue
			}
			parseMoveCanonicalNodeIDInPlace(buildCurrentOrder, parseCurrentIndex, parseTargetIndex, buildCurrentIndexByNode)
			buildMoveOps = append(buildMoveOps, PatchKeyedMoveOpRaw{
				ParentNodeID:     getParentNodeID,
				SourceNodeID:     getTargetNodeID,
				DestinationIndex: uint32(parseTargetIndex),
			})
		}
	}
	return buildMoveOps, nil
}

// buildCanonicalReplaceSubtreePatchStream emits one replace-subtree patch stream for full-root fallback cases.
func buildCanonicalReplaceSubtreePatchStream(
	parseRegionID string,
	parseEpoch uint64,
	parseInputVersion uint64,
	parsePatchVersion uint64,
	parseTargetNodeID uint64,
	parseNextIR CanonicalRenderIR,
) (PatchStreamRaw, error) {
	buildReplaceSubtreeOp := PatchReplaceSubtreeOpRaw{
		TargetNodeID: parseTargetNodeID,
		Subtree:      parseBuildReplaceSubtreePayload(parseNextIR),
	}
	buildHeader := PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        parseRegionID,
		Epoch:           parseEpoch,
		InputVersion:    parseInputVersion,
		PatchVersion:    parsePatchVersion,
	}
	buildIdentityState, parseIdentityStateErr := buildPatchStreamIdentityState(buildHeader, nil)
	if parseIdentityStateErr != nil {
		return PatchStreamRaw{}, parseIdentityStateErr
	}
	defer clearPatchStreamIdentityState(&buildIdentityState)
	if parseStartErr := startPatchStreamIdentityOpArray(&buildIdentityState); parseStartErr != nil {
		return PatchStreamRaw{}, parseStartErr
	}
	buildOps := make([]PatchStreamOpRaw, 0, 1)
	if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
		GetOpCode:           uint8(PatchOpCodeReplaceSubtree),
		GetReplaceSubtreeOp: &buildReplaceSubtreeOp,
	}); parseAppendErr != nil {
		return PatchStreamRaw{}, parseAppendErr
	}
	buildPatchIdentity, _ := formatPatchStreamIdentityState(&buildIdentityState)
	return buildPatchStreamRawWithIdentity(buildHeader, nil, buildOps, buildPatchIdentity)
}

// BuildCanonicalPatchStream diffs previous and next canonical IR and emits one canonical typed patch stream.
func BuildCanonicalPatchStream(
	parseRegionID string,
	parseEpoch uint64,
	parseInputVersion uint64,
	parsePatchVersion uint64,
	parsePreviousIR CanonicalRenderIR,
	parseNextIR CanonicalRenderIR,
) (PatchStreamRaw, bool, error) {
	if strings.TrimSpace(parseRegionID) == "" {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream region ID is required")
	}
	if parseEpoch == 0 {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream epoch is required")
	}
	if parseInputVersion == 0 {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream input version is required")
	}
	if parsePatchVersion == 0 {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream patch version is required")
	}
	if IsCanonicalRenderIREqual(parsePreviousIR, parseNextIR) {
		return PatchStreamRaw{}, true, nil
	}
	parsePreviousTree, parsePreviousTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
	if parsePreviousTreeErr != nil {
		return PatchStreamRaw{}, false, parsePreviousTreeErr
	}
	parseNextTree, parseNextTreeErr := ParseCanonicalRenderTree(parseNextIR)
	if parseNextTreeErr != nil {
		return PatchStreamRaw{}, false, parseNextTreeErr
	}
	hasStructuralMismatch := false
	for getNodeID, getPreviousNode := range parsePreviousTree.getNodeByID {
		getNextNode, hasNextNode := parseNextTree.getNodeByID[getNodeID]
		if !hasNextNode {
			continue
		}
		if getPreviousNode.getKind != getNextNode.getKind ||
			(getPreviousNode.getKind == RenderNodeKindHostElement && getPreviousNode.getTag != getNextNode.getTag) {
			hasStructuralMismatch = true
			break
		}
	}
	if hasStructuralMismatch {
		buildPatchStreamRaw, _ := buildCanonicalReplaceSubtreePatchStream(
			parseRegionID,
			parseEpoch,
			parseInputVersion,
			parsePatchVersion,
			parsePreviousTree.getRootNodeID,
			parseNextIR,
		)
		return buildPatchStreamRaw, false, nil
	}
	buildRemovedNodeIDs := make(map[uint64]struct{})
	buildInsertedNodeIDs := make(map[uint64]struct{})
	for getNodeID := range parsePreviousTree.getNodeByID {
		if _, hasNodeID := parseNextTree.getNodeByID[getNodeID]; !hasNodeID {
			buildRemovedNodeIDs[getNodeID] = struct{}{}
		}
	}
	for getNodeID := range parseNextTree.getNodeByID {
		if _, hasNodeID := parsePreviousTree.getNodeByID[getNodeID]; !hasNodeID {
			buildInsertedNodeIDs[getNodeID] = struct{}{}
		}
	}
	if _, hasRemovedRoot := buildRemovedNodeIDs[parsePreviousTree.getRootNodeID]; hasRemovedRoot {
		buildPatchStreamRaw, _ := buildCanonicalReplaceSubtreePatchStream(
			parseRegionID,
			parseEpoch,
			parseInputVersion,
			parsePatchVersion,
			parsePreviousTree.getRootNodeID,
			parseNextIR,
		)
		return buildPatchStreamRaw, false, nil
	}
	for getNodeID := range buildInsertedNodeIDs {
		getNextNode := parseNextTree.getNodeByID[getNodeID]
		if getNodeID == parseNextTree.getRootNodeID || getNextNode.getParentNodeID == 0 {
			buildPatchStreamRaw, _ := buildCanonicalReplaceSubtreePatchStream(
				parseRegionID,
				parseEpoch,
				parseInputVersion,
				parsePatchVersion,
				parsePreviousTree.getRootNodeID,
				parseNextIR,
			)
			return buildPatchStreamRaw, false, nil
		}
	}
	hasStructuralNodeDelta := len(buildRemovedNodeIDs) > 0 || len(buildInsertedNodeIDs) > 0
	buildRemoveNodeIDs := make([]uint64, 0, len(buildRemovedNodeIDs))
	for getNodeID := range buildRemovedNodeIDs {
		if getNodeID == parsePreviousTree.getRootNodeID {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: root structural replacement is unsupported in first-slice patch diff")
		}
		buildRemoveNodeIDs = append(buildRemoveNodeIDs, getNodeID)
	}
	sort.Slice(buildRemoveNodeIDs, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftDepth := parsePreviousTree.GetCanonicalNodeDepth(buildRemoveNodeIDs[parseLeftIndex])
		parseRightDepth := parsePreviousTree.GetCanonicalNodeDepth(buildRemoveNodeIDs[parseRightIndex])
		if parseLeftDepth == parseRightDepth {
			return buildRemoveNodeIDs[parseLeftIndex] < buildRemoveNodeIDs[parseRightIndex]
		}
		return parseLeftDepth > parseRightDepth
	})
	buildNextSiblingIndexCache := parseBuildCanonicalSiblingIndexCache(parseNextTree)
	buildInsertNodeIDs := make([]uint64, 0, len(buildInsertedNodeIDs))
	for getNodeID := range buildInsertedNodeIDs {
		buildInsertNodeIDs = append(buildInsertNodeIDs, getNodeID)
	}
	sort.Slice(buildInsertNodeIDs, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftNodeID := buildInsertNodeIDs[parseLeftIndex]
		parseRightNodeID := buildInsertNodeIDs[parseRightIndex]
		parseLeftDepth := parseNextTree.GetCanonicalNodeDepth(parseLeftNodeID)
		parseRightDepth := parseNextTree.GetCanonicalNodeDepth(parseRightNodeID)
		if parseLeftDepth != parseRightDepth {
			return parseLeftDepth < parseRightDepth
		}
		parseLeftNode := parseNextTree.getNodeByID[parseLeftNodeID]
		parseRightNode := parseNextTree.getNodeByID[parseRightNodeID]
		if parseLeftNode.getParentNodeID != parseRightNode.getParentNodeID {
			return parseLeftNode.getParentNodeID < parseRightNode.getParentNodeID
		}
		parseLeftSiblingIndex := parseGetCanonicalSiblingIndexFromCache(buildNextSiblingIndexCache, parseLeftNode.getParentNodeID, parseLeftNodeID)
		parseRightSiblingIndex := parseGetCanonicalSiblingIndexFromCache(buildNextSiblingIndexCache, parseRightNode.getParentNodeID, parseRightNodeID)
		if parseLeftSiblingIndex == parseRightSiblingIndex {
			return parseLeftNodeID < parseRightNodeID
		}
		return parseLeftSiblingIndex > parseRightSiblingIndex
	})
	buildInsertOps := make([]patchInsertBuildOp, 0, len(buildInsertNodeIDs))
	buildSetTextOps := []patchSetTextBuildOp{}
	buildSetAttrOps := []patchSetAttrBuildOp{}
	buildSetStyleOps := []patchSetStyleBuildOp{}
	buildRemoveAttrOps := []patchRemoveAttrBuildOp{}
	buildRemoveStyleOps := []patchRemoveStyleBuildOp{}
	for _, getNodeID := range buildInsertNodeIDs {
		getNextNode := parseNextTree.getNodeByID[getNodeID]
		if getNodeID == parseNextTree.getRootNodeID || getNextNode.getParentNodeID == 0 {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: root structural replacement is unsupported in first-slice patch diff")
		}
		buildAnchorNodeID := parseFindCanonicalInsertAnchorWithIndexCache(
			getNextNode.getParentNodeID,
			getNodeID,
			parsePreviousTree,
			parseNextTree,
			buildInsertedNodeIDs,
			buildNextSiblingIndexCache,
		)
		buildInsertOps = append(buildInsertOps, patchInsertBuildOp{
			getParentNodeID: getNextNode.getParentNodeID,
			getNodeID:       getNodeID,
			getNodeKind:     getNextNode.getKind,
			getNodeTag:      getNextNode.getTag,
			getNodeText:     getNextNode.getText,
			getNodeKey:      getNextNode.getKey,
			hasNodeKey:      getNextNode.hasKey,
			getAnchorNodeID: buildAnchorNodeID,
		})
		for _, getProp := range getNextNode.getPropByKey {
			if getProp.Kind == RenderPropKindTextAdjacent {
				continue
			}
			if getProp.Kind == RenderPropKindStyle {
				buildSetStyleOps = append(buildSetStyleOps, patchSetStyleBuildOp{
					getTargetNodeID: getNodeID,
					getStyleValue:   getProp.Value,
				})
				continue
			}
			buildSetAttrOps = append(buildSetAttrOps, patchSetAttrBuildOp{
				getTargetNodeID: getNodeID,
				getKind:         getProp.Kind,
				getKey:          getProp.Key,
				getValue:        getProp.Value,
			})
		}
	}
	for getNodeID, getPreviousNode := range parsePreviousTree.getNodeByID {
		getNextNode, hasNextNode := parseNextTree.getNodeByID[getNodeID]
		if !hasNextNode {
			continue
		}
		if _, hasRemovedNode := buildRemovedNodeIDs[getNodeID]; hasRemovedNode {
			continue
		}
		if _, hasInsertedNode := buildInsertedNodeIDs[getNodeID]; hasInsertedNode {
			continue
		}
		if getPreviousNode.getKind == RenderNodeKindText && getPreviousNode.getText != getNextNode.getText {
			buildSetTextOps = append(buildSetTextOps, patchSetTextBuildOp{
				getTargetNodeID: getNodeID,
				getText:         getNextNode.getText,
			})
		}
		for getPropKey, getNextProp := range getNextNode.getPropByKey {
			if getNextProp.Kind == RenderPropKindTextAdjacent {
				continue
			}
			getPreviousProp, hasPreviousProp := getPreviousNode.getPropByKey[getPropKey]
			if !hasPreviousProp || getPreviousProp.Value != getNextProp.Value || getPreviousProp.Kind != getNextProp.Kind {
				if getNextProp.Kind == RenderPropKindStyle {
					buildSetStyleOps = append(buildSetStyleOps, patchSetStyleBuildOp{
						getTargetNodeID: getNodeID,
						getStyleValue:   getNextProp.Value,
					})
					continue
				}
				buildSetAttrOps = append(buildSetAttrOps, patchSetAttrBuildOp{
					getTargetNodeID: getNodeID,
					getKind:         getNextProp.Kind,
					getKey:          getNextProp.Key,
					getValue:        getNextProp.Value,
				})
			}
		}
		for getPropKey, getPreviousProp := range getPreviousNode.getPropByKey {
			if getPreviousProp.Kind == RenderPropKindTextAdjacent {
				continue
			}
			if _, hasNextProp := getNextNode.getPropByKey[getPropKey]; !hasNextProp {
				if getPreviousProp.Kind == RenderPropKindStyle {
					buildRemoveStyleOps = append(buildRemoveStyleOps, patchRemoveStyleBuildOp{
						getTargetNodeID: getNodeID,
					})
					continue
				}
				buildRemoveAttrOps = append(buildRemoveAttrOps, patchRemoveAttrBuildOp{
					getTargetNodeID: getNodeID,
					getKey:          getPropKey,
				})
			}
		}
	}
	buildMoveOps, parseMoveOpsErr := buildCanonicalPatchKeyedMoveOps(
		parseRegionID,
		parsePreviousTree,
		parseNextTree,
		buildRemovedNodeIDs,
		buildInsertedNodeIDs,
		hasStructuralNodeDelta,
	)
	if parseMoveOpsErr != nil {
		return PatchStreamRaw{}, false, parseMoveOpsErr
	}
	sort.Slice(buildSetTextOps, func(parseLeftIndex int, parseRightIndex int) bool {
		return buildSetTextOps[parseLeftIndex].getTargetNodeID < buildSetTextOps[parseRightIndex].getTargetNodeID
	})
	sort.Slice(buildSetAttrOps, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftOp := buildSetAttrOps[parseLeftIndex]
		parseRightOp := buildSetAttrOps[parseRightIndex]
		if parseLeftOp.getTargetNodeID == parseRightOp.getTargetNodeID {
			return parseLeftOp.getKey < parseRightOp.getKey
		}
		return parseLeftOp.getTargetNodeID < parseRightOp.getTargetNodeID
	})
	sort.Slice(buildSetStyleOps, func(parseLeftIndex int, parseRightIndex int) bool {
		return buildSetStyleOps[parseLeftIndex].getTargetNodeID < buildSetStyleOps[parseRightIndex].getTargetNodeID
	})
	sort.Slice(buildRemoveAttrOps, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftOp := buildRemoveAttrOps[parseLeftIndex]
		parseRightOp := buildRemoveAttrOps[parseRightIndex]
		if parseLeftOp.getTargetNodeID == parseRightOp.getTargetNodeID {
			return parseLeftOp.getKey < parseRightOp.getKey
		}
		return parseLeftOp.getTargetNodeID < parseRightOp.getTargetNodeID
	})
	sort.Slice(buildRemoveStyleOps, func(parseLeftIndex int, parseRightIndex int) bool {
		return buildRemoveStyleOps[parseLeftIndex].getTargetNodeID < buildRemoveStyleOps[parseRightIndex].getTargetNodeID
	})
	sort.Slice(buildMoveOps, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftOp := buildMoveOps[parseLeftIndex]
		parseRightOp := buildMoveOps[parseRightIndex]
		if parseLeftOp.ParentNodeID == parseRightOp.ParentNodeID {
			if parseLeftOp.DestinationIndex == parseRightOp.DestinationIndex {
				return parseLeftOp.SourceNodeID < parseRightOp.SourceNodeID
			}
			return parseLeftOp.DestinationIndex < parseRightOp.DestinationIndex
		}
		return parseLeftOp.ParentNodeID < parseRightOp.ParentNodeID
	})
	buildPatchStrings := make([]string, 0, buildPatchStringCapacityFromPatchOps(
		buildInsertOps,
		buildSetTextOps,
		buildSetAttrOps,
		buildSetStyleOps,
		buildRemoveAttrOps,
	))
	for _, getInsertOp := range buildInsertOps {
		if getInsertOp.getNodeKind == RenderNodeKindText {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeText)
		}
		if getInsertOp.getNodeKind == RenderNodeKindHostElement {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeTag)
		}
		if getInsertOp.hasNodeKey {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeKey)
		}
	}
	for _, getSetTextOp := range buildSetTextOps {
		buildPatchStrings = append(buildPatchStrings, getSetTextOp.getText)
	}
	for _, getSetAttrOp := range buildSetAttrOps {
		buildPatchStrings = append(buildPatchStrings, getSetAttrOp.getKey, getSetAttrOp.getValue)
	}
	for _, getSetStyleOp := range buildSetStyleOps {
		buildPatchStrings = append(buildPatchStrings, getSetStyleOp.getStyleValue)
	}
	for _, getRemoveAttrOp := range buildRemoveAttrOps {
		buildPatchStrings = append(buildPatchStrings, getRemoveAttrOp.getKey)
	}
	// The table is built from the exact strings collected above, so each lookup below must succeed.
	buildPatchStringTable := BuildRenderStringTable(buildPatchStrings)
	buildPatchStringEntries := append([]string(nil), buildPatchStringTable.Entries...)
	buildOps := make([]PatchStreamOpRaw, 0, len(buildRemoveNodeIDs)+len(buildInsertOps)+len(buildSetTextOps)+len(buildSetAttrOps)+len(buildSetStyleOps)+len(buildRemoveAttrOps)+len(buildRemoveStyleOps)+len(buildMoveOps))
	buildHeader := PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        parseRegionID,
		Epoch:           parseEpoch,
		InputVersion:    parseInputVersion,
		PatchVersion:    parsePatchVersion,
	}
	buildIdentityState, parseIdentityStateErr := buildPatchStreamIdentityState(buildHeader, buildPatchStringEntries)
	if parseIdentityStateErr != nil {
		return PatchStreamRaw{}, false, parseIdentityStateErr
	}
	defer clearPatchStreamIdentityState(&buildIdentityState)
	if parseArrayErr := startPatchStreamIdentityOpArray(&buildIdentityState); parseArrayErr != nil {
		return PatchStreamRaw{}, false, parseArrayErr
	}
	for _, getNodeID := range buildRemoveNodeIDs {
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveNode),
			GetRemoveOp: &PatchRemoveOpRaw{
				TargetNodeID: getNodeID,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getInsertOp := range buildInsertOps {
		buildNodeTextRef := uint32(0)
		switch getInsertOp.getNodeKind {
		case RenderNodeKindText:
			getTextRef, _ := buildPatchStringTable.GetRenderStringRef(getInsertOp.getNodeText)
			buildNodeTextRef = getTextRef
		case RenderNodeKindHostElement:
			getTagRef, _ := buildPatchStringTable.GetRenderStringRef(getInsertOp.getNodeTag)
			buildNodeTextRef = getTagRef
		}
		buildNodeRaw := RenderNodeRecordRaw{
			NodeID:  getInsertOp.getNodeID,
			Kind:    uint8(getInsertOp.getNodeKind),
			TextRef: buildNodeTextRef,
		}
		if getInsertOp.hasNodeKey {
			buildNodeRaw.KeyText = getInsertOp.getNodeKey
			buildNodeRaw.KeyHash = parseHashCanonicalString("key:"+getInsertOp.getNodeKey, nil)
		}
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeInsertNode),
			GetInsertOp: &PatchInsertOpRaw{
				ParentNodeID: getInsertOp.getParentNodeID,
				AnchorNodeID: getInsertOp.getAnchorNodeID,
				Node:         buildNodeRaw,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getSetTextOp := range buildSetTextOps {
		getTextRef, _ := buildPatchStringTable.GetRenderStringRef(getSetTextOp.getText)
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetText),
			GetSetTextOp: &PatchSetTextOpRaw{
				TargetNodeID: getSetTextOp.getTargetNodeID,
				TextRef:      getTextRef,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getSetAttrOp := range buildSetAttrOps {
		getKeyRef, _ := buildPatchStringTable.GetRenderStringRef(getSetAttrOp.getKey)
		getValueRef, _ := buildPatchStringTable.GetRenderStringRef(getSetAttrOp.getValue)
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetAttr),
			GetSetAttrOp: &PatchSetAttrOpRaw{
				TargetNodeID: getSetAttrOp.getTargetNodeID,
				Attr: RenderPropRecordRaw{
					Kind:     uint8(getSetAttrOp.getKind),
					KeyRef:   getKeyRef,
					ValueRef: getValueRef,
				},
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getSetStyleOp := range buildSetStyleOps {
		getStyleRef, _ := buildPatchStringTable.GetRenderStringRef(getSetStyleOp.getStyleValue)
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetStyle),
			GetSetStyleOp: &PatchSetStyleOpRaw{
				TargetNodeID: getSetStyleOp.getTargetNodeID,
				StyleRef:     getStyleRef,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getRemoveAttrOp := range buildRemoveAttrOps {
		getKeyRef, _ := buildPatchStringTable.GetRenderStringRef(getRemoveAttrOp.getKey)
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveAttr),
			GetRemoveAttrOp: &PatchRemoveAttrOpRaw{
				TargetNodeID: getRemoveAttrOp.getTargetNodeID,
				KeyRef:       getKeyRef,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getRemoveStyleOp := range buildRemoveStyleOps {
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveStyle),
			GetRemoveStyleOp: &PatchRemoveStyleOpRaw{
				TargetNodeID: getRemoveStyleOp.getTargetNodeID,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getMoveOp := range buildMoveOps {
		buildMoveOp := getMoveOp
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode:      uint8(PatchOpCodeMoveKeyedChild),
			GetKeyedMoveOp: &buildMoveOp,
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	buildPatchIdentity, parseIdentityErr := formatPatchStreamIdentityState(&buildIdentityState)
	if parseIdentityErr != nil {
		return PatchStreamRaw{}, false, parseIdentityErr
	}
	getPatchStreamRaw, _ := buildPatchStreamRawWithIdentity(buildHeader, buildPatchStringEntries, buildOps, buildPatchIdentity)
	return getPatchStreamRaw, false, nil
}

// parseFindCanonicalInsertAnchorWithIndexCache resolves one inserted-node anchor using one reusable sibling-index cache.
func parseFindCanonicalInsertAnchorWithIndexCache(
	parseParentNodeID uint64,
	parseNodeID uint64,
	parsePreviousTree canonicalRenderTree,
	parseNextTree canonicalRenderTree,
	parseInsertedNodeIDs map[uint64]struct{},
	parseSiblingIndexCache *canonicalSiblingIndexCache,
) uint64 {
	parseParentNode, hasParentNode := parseNextTree.getNodeByID[parseParentNodeID]
	if !hasParentNode {
		return 0
	}
	parseSiblingIndex := parseGetCanonicalSiblingIndexFromCache(parseSiblingIndexCache, parseParentNodeID, parseNodeID)
	if parseSiblingIndex < 0 {
		return 0
	}
	for parseIndex := parseSiblingIndex + 1; parseIndex < len(parseParentNode.getChildNodeIDs); parseIndex++ {
		getSiblingNodeID := parseParentNode.getChildNodeIDs[parseIndex]
		if _, hasInsertedSibling := parseInsertedNodeIDs[getSiblingNodeID]; hasInsertedSibling {
			continue
		}
		if _, hasPreviousSibling := parsePreviousTree.getNodeByID[getSiblingNodeID]; hasPreviousSibling {
			return getSiblingNodeID
		}
	}
	return 0
}

// parseBuildCanonicalSiblingIndexMap builds one nodeID->index lookup map for one sibling order list.
func parseBuildCanonicalSiblingIndexMap(parseChildNodeIDs []uint64) map[uint64]int {
	buildSiblingIndexByNode := make(map[uint64]int, len(parseChildNodeIDs))
	for parseIndex, getNodeID := range parseChildNodeIDs {
		buildSiblingIndexByNode[getNodeID] = parseIndex
	}
	return buildSiblingIndexByNode
}

// parseBuildCanonicalSiblingIndexCache builds one lazy sibling-index cache for one canonical tree.
func parseBuildCanonicalSiblingIndexCache(parseTree canonicalRenderTree) *canonicalSiblingIndexCache {
	return &canonicalSiblingIndexCache{
		getTree:                parseTree,
		getSiblingIndexByParen: make(map[uint64]map[uint64]int),
	}
}

// parseGetCanonicalSiblingIndexFromCache resolves one sibling index using one cached parent lookup map.
func parseGetCanonicalSiblingIndexFromCache(parseCache *canonicalSiblingIndexCache, parseParentNodeID uint64, parseNodeID uint64) int {
	if parseCache == nil {
		return -1
	}
	buildSiblingIndexByNode, hasSiblingIndexByNode := parseCache.getSiblingIndexByParen[parseParentNodeID]
	if !hasSiblingIndexByNode {
		parseParentNode, hasParentNode := parseCache.getTree.getNodeByID[parseParentNodeID]
		if !hasParentNode {
			return -1
		}
		buildSiblingIndexByNode = parseBuildCanonicalSiblingIndexMap(parseParentNode.getChildNodeIDs)
		parseCache.getSiblingIndexByParen[parseParentNodeID] = buildSiblingIndexByNode
	}
	parseSiblingIndex, hasSiblingIndex := buildSiblingIndexByNode[parseNodeID]
	if !hasSiblingIndex {
		return -1
	}
	return parseSiblingIndex
}

// parseMoveCanonicalNodeIDInPlace reorders one node ID in place and keeps the sibling-index map in sync.
func parseMoveCanonicalNodeIDInPlace(parseNodeIDs []uint64, parseFromIndex int, parseToIndex int, parseIndexByNode map[uint64]int) {
	if parseFromIndex < 0 || parseFromIndex >= len(parseNodeIDs) {
		return
	}
	if parseToIndex < 0 {
		parseToIndex = 0
	}
	if parseToIndex >= len(parseNodeIDs) {
		parseToIndex = len(parseNodeIDs) - 1
	}
	if parseFromIndex == parseToIndex {
		return
	}
	getNodeID := parseNodeIDs[parseFromIndex]
	if parseFromIndex < parseToIndex {
		copy(parseNodeIDs[parseFromIndex:parseToIndex], parseNodeIDs[parseFromIndex+1:parseToIndex+1])
		for parseIndex := parseFromIndex; parseIndex < parseToIndex; parseIndex++ {
			parseIndexByNode[parseNodeIDs[parseIndex]] = parseIndex
		}
		parseNodeIDs[parseToIndex] = getNodeID
		parseIndexByNode[getNodeID] = parseToIndex
		return
	}
	copy(parseNodeIDs[parseToIndex+1:parseFromIndex+1], parseNodeIDs[parseToIndex:parseFromIndex])
	for parseIndex := parseToIndex + 1; parseIndex <= parseFromIndex; parseIndex++ {
		parseIndexByNode[parseNodeIDs[parseIndex]] = parseIndex
	}
	parseNodeIDs[parseToIndex] = getNodeID
	parseIndexByNode[getNodeID] = parseToIndex
}

// parseFilterCanonicalExistingOrder filters one child order to IDs that remain in-place candidates.
func parseFilterCanonicalExistingOrder(parseNodeIDs []uint64, parseRemovedNodeIDs map[uint64]struct{}, parseInsertedNodeIDs map[uint64]struct{}) []uint64 {
	buildNodeIDs := make([]uint64, 0, len(parseNodeIDs))
	for _, getNodeID := range parseNodeIDs {
		if _, hasRemovedNode := parseRemovedNodeIDs[getNodeID]; hasRemovedNode {
			continue
		}
		if _, hasInsertedNode := parseInsertedNodeIDs[getNodeID]; hasInsertedNode {
			continue
		}
		buildNodeIDs = append(buildNodeIDs, getNodeID)
	}
	return buildNodeIDs
}

// parseHasCanonicalNodeOrderEqual reports whether two canonical node-order slices match exactly.
func parseHasCanonicalNodeOrderEqual(parseLeftNodeIDs []uint64, parseRightNodeIDs []uint64) bool {
	if len(parseLeftNodeIDs) != len(parseRightNodeIDs) {
		return false
	}
	for parseIndex := range parseLeftNodeIDs {
		if parseLeftNodeIDs[parseIndex] != parseRightNodeIDs[parseIndex] {
			return false
		}
	}
	return true
}

// parseBuildReplaceSubtreePayload builds one replace-subtree payload from canonical render IR.
func parseBuildReplaceSubtreePayload(parseIR CanonicalRenderIR) PatchReplaceSubtreePayloadRaw {
	return PatchReplaceSubtreePayloadRaw{
		RootNodeID:  parseIR.GetRootNodeID,
		StringTable: append([]string(nil), parseIR.GetStringTable.Entries...),
		NodeRecords: append([]RenderNodeRecordRaw(nil), parseIR.GetNodeRecords...),
		PropRecords: append([]RenderPropRecordRaw(nil), parseIR.GetPropRecords...),
	}
}
