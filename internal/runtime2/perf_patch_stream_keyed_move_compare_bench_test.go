package runtime2

import "testing"

var storePatchMoveOpsSink []PatchKeyedMoveOpRaw

// buildPatchMoveBenchFixture stores canonical trees and structural delta sets for keyed-move compare benchmarks.
type buildPatchMoveBenchFixture struct {
	getRegionID        string
	getPreviousTree    canonicalRenderTree
	getNextTree        canonicalRenderTree
	getRemovedNodeIDs  map[uint64]struct{}
	getInsertedNodeIDs map[uint64]struct{}
}

// buildPatchMoveBenchmarkFixture builds one repeatable keyed-move workload where structural delta exists globally but touches only one parent.
func buildPatchMoveBenchmarkFixture() buildPatchMoveBenchFixture {
	const (
		getParentCount = 64
		getChildCount  = 16
		getRootNodeID  = uint64(1)
	)
	buildPreviousNodeByID := make(map[uint64]canonicalRenderNodeState, getParentCount*getChildCount+getParentCount+1)
	buildNextNodeByID := make(map[uint64]canonicalRenderNodeState, getParentCount*getChildCount+getParentCount+1)
	buildRootChildren := make([]uint64, 0, getParentCount)
	buildRemovedNodeIDs := make(map[uint64]struct{}, 1)
	buildInsertedNodeIDs := make(map[uint64]struct{}, 1)
	for parseParentIndex := 0; parseParentIndex < getParentCount; parseParentIndex++ {
		getParentNodeID := uint64(1_000_000 + parseParentIndex)
		buildRootChildren = append(buildRootChildren, getParentNodeID)
		buildPreviousChildren := make([]uint64, 0, getChildCount)
		buildNextChildren := make([]uint64, 0, getChildCount)
		for parseChildIndex := 0; parseChildIndex < getChildCount; parseChildIndex++ {
			getChildNodeID := getParentNodeID*1_000 + uint64(parseChildIndex+1)
			buildPreviousChildren = append(buildPreviousChildren, getChildNodeID)
			buildNextChildren = append(buildNextChildren, getChildNodeID)
			buildPreviousNodeByID[getChildNodeID] = canonicalRenderNodeState{
				getNodeID:       getChildNodeID,
				getParentNodeID: getParentNodeID,
				hasKey:          true,
			}
			buildNextNodeByID[getChildNodeID] = canonicalRenderNodeState{
				getNodeID:       getChildNodeID,
				getParentNodeID: getParentNodeID,
				hasKey:          true,
			}
		}
		switch parseParentIndex {
		case 0:
			getRemovedNodeID := buildPreviousChildren[0]
			delete(buildNextNodeByID, getRemovedNodeID)
			getInsertedNodeID := getParentNodeID*1_000 + 99_999
			buildNextNodeByID[getInsertedNodeID] = canonicalRenderNodeState{
				getNodeID:       getInsertedNodeID,
				getParentNodeID: getParentNodeID,
				hasKey:          true,
			}
			buildRemovedNodeIDs[getRemovedNodeID] = struct{}{}
			buildInsertedNodeIDs[getInsertedNodeID] = struct{}{}
			buildNextChildren = append(
				[]uint64{buildPreviousChildren[1], getInsertedNodeID},
				buildPreviousChildren[2:]...,
			)
		case 1:
			buildNextChildren[0], buildNextChildren[1] = buildNextChildren[1], buildNextChildren[0]
		}
		buildPreviousNodeByID[getParentNodeID] = canonicalRenderNodeState{
			getNodeID:       getParentNodeID,
			getChildNodeIDs: buildPreviousChildren,
			hasKeyedChild:   true,
		}
		buildNextNodeByID[getParentNodeID] = canonicalRenderNodeState{
			getNodeID:       getParentNodeID,
			getChildNodeIDs: buildNextChildren,
			hasKeyedChild:   true,
		}
	}
	buildPreviousNodeByID[getRootNodeID] = canonicalRenderNodeState{
		getNodeID:       getRootNodeID,
		getChildNodeIDs: append([]uint64(nil), buildRootChildren...),
	}
	buildNextNodeByID[getRootNodeID] = canonicalRenderNodeState{
		getNodeID:       getRootNodeID,
		getChildNodeIDs: append([]uint64(nil), buildRootChildren...),
	}
	return buildPatchMoveBenchFixture{
		getRegionID: "bench-region",
		getPreviousTree: canonicalRenderTree{
			getRootNodeID: getRootNodeID,
			getNodeByID:   buildPreviousNodeByID,
		},
		getNextTree: canonicalRenderTree{
			getRootNodeID: getRootNodeID,
			getNodeByID:   buildNextNodeByID,
		},
		getRemovedNodeIDs:  buildRemovedNodeIDs,
		getInsertedNodeIDs: buildInsertedNodeIDs,
	}
}

// buildCanonicalPatchKeyedMoveOpsLegacy preserves the previous keyed-move generation path that filtered sibling orders for every parent whenever any structural delta existed.
func buildCanonicalPatchKeyedMoveOpsLegacy(
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
		if parseHasStructuralNodeDelta {
			buildCurrentOrder = parseFilterCanonicalExistingOrder(getPreviousNode.getChildNodeIDs, parseRemovedNodeIDs, parseInsertedNodeIDs)
			buildTargetOrder = parseFilterCanonicalExistingOrder(getNextNode.getChildNodeIDs, parseRemovedNodeIDs, parseInsertedNodeIDs)
		}
		if len(buildTargetOrder) > getPatchMoveSiblingHardLimit {
			return nil, nil
		}
		if len(buildTargetOrder) > getPatchMoveSiblingWarnLimit && !hasLoggedMoveSiblingWarn {
			hasLoggedMoveSiblingWarn = true
			_ = parseRegionID
		}
		if parseHasCanonicalNodeOrderEqual(buildCurrentOrder, buildTargetOrder) {
			continue
		}
		if !getNextNode.hasKeyedChild {
			continue
		}
		if !parseHasStructuralNodeDelta {
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

// BenchmarkBuildCanonicalPatchKeyedMoveOpsCurrentVsLegacy compares keyed-move generation after per-parent structural-delta gating against the previous always-filter path.
func BenchmarkBuildCanonicalPatchKeyedMoveOpsCurrentVsLegacy(parseB *testing.B) {
	getFixture := buildPatchMoveBenchmarkFixture()
	parseB.Run("legacy_always_filter", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getMoveOps, parseMoveErr := buildCanonicalPatchKeyedMoveOpsLegacy(
				getFixture.getRegionID,
				getFixture.getPreviousTree,
				getFixture.getNextTree,
				getFixture.getRemovedNodeIDs,
				getFixture.getInsertedNodeIDs,
				true,
			)
			if parseMoveErr != nil {
				parseB.Fatalf("buildCanonicalPatchKeyedMoveOpsLegacy returned error: %v", parseMoveErr)
			}
			storePatchMoveOpsSink = getMoveOps
		}
	})
	parseB.Run("current_parent_delta_filter", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getMoveOps, parseMoveErr := buildCanonicalPatchKeyedMoveOps(
				getFixture.getRegionID,
				getFixture.getPreviousTree,
				getFixture.getNextTree,
				getFixture.getRemovedNodeIDs,
				getFixture.getInsertedNodeIDs,
				true,
			)
			if parseMoveErr != nil {
				parseB.Fatalf("buildCanonicalPatchKeyedMoveOps returned error: %v", parseMoveErr)
			}
			storePatchMoveOpsSink = getMoveOps
		}
	})
}
