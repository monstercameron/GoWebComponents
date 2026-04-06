package runtime2

import (
	"fmt"
	"strings"
	"testing"
)

// buildCanonicalKeyedMoveTreeForTest builds one minimal canonical tree with one keyed parent and a flat keyed child list.
func buildCanonicalKeyedMoveTreeForTest(parseParentNodeID uint64, parseChildNodeIDs []uint64) canonicalRenderTree {
	parseNodeByID := make(map[uint64]canonicalRenderNodeState, len(parseChildNodeIDs)+1)
	parseParentState := canonicalRenderNodeState{
		getNodeID:       parseParentNodeID,
		getKind:         RenderNodeKindHostElement,
		getDepth:        0,
		getTag:          "ul",
		hasKeyedChild:   len(parseChildNodeIDs) > 0,
		getParentNodeID: 0,
		getChildNodeIDs: append([]uint64(nil), parseChildNodeIDs...),
	}
	parseNodeByID[parseParentNodeID] = parseParentState
	for _, parseChildNodeID := range parseChildNodeIDs {
		parseNodeByID[parseChildNodeID] = canonicalRenderNodeState{
			getNodeID:       parseChildNodeID,
			getKind:         RenderNodeKindHostElement,
			getDepth:        1,
			getTag:          "li",
			getKey:          fmt.Sprintf("key-%d", parseChildNodeID),
			hasKey:          true,
			getParentNodeID: parseParentNodeID,
		}
	}
	return canonicalRenderTree{
		getRootNodeID: parseParentNodeID,
		getNodeByID:   parseNodeByID,
	}
}

// TestBuildCanonicalPatchStreamValidatesInputsAndNoOp verifies patch-stream construction rejects invalid arguments and short-circuits equal IRs.
func TestBuildCanonicalPatchStreamValidatesInputsAndNoOp(parseTesting *testing.T) {
	parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "text",
		"text": "same",
	})
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "text",
		"text": "same",
	})

	testCases := []struct {
		name          string
		regionID      string
		epoch         uint64
		inputVersion  uint64
		patchVersion  uint64
		expectMessage string
	}{
		{
			name:          "blank region",
			regionID:      " ",
			epoch:         1,
			inputVersion:  1,
			patchVersion:  1,
			expectMessage: "patch stream region ID is required",
		},
		{
			name:          "zero epoch",
			regionID:      "region-a",
			epoch:         0,
			inputVersion:  1,
			patchVersion:  1,
			expectMessage: "patch stream epoch is required",
		},
		{
			name:          "zero input version",
			regionID:      "region-a",
			epoch:         1,
			inputVersion:  0,
			patchVersion:  1,
			expectMessage: "patch stream input version is required",
		},
		{
			name:          "zero patch version",
			regionID:      "region-a",
			epoch:         1,
			inputVersion:  1,
			patchVersion:  0,
			expectMessage: "patch stream patch version is required",
		},
	}

	for _, parseTestCase := range testCases {
		parseTesting.Run(parseTestCase.name, func(parseSubTest *testing.T) {
			_, _, parseErr := BuildCanonicalPatchStream(
				parseTestCase.regionID,
				parseTestCase.epoch,
				parseTestCase.inputVersion,
				parseTestCase.patchVersion,
				parsePreviousIR,
				parseNextIR,
			)
			if parseErr == nil {
				parseSubTest.Fatal("BuildCanonicalPatchStream error = nil, want error")
			}
			if !strings.Contains(parseErr.Error(), parseTestCase.expectMessage) {
				parseSubTest.Fatalf("BuildCanonicalPatchStream error = %q, want %q", parseErr.Error(), parseTestCase.expectMessage)
			}
		})
	}

	parseNoOpPatch, parseNoOp, parseNoOpErr := BuildCanonicalPatchStream(
		"region-a",
		1,
		1,
		1,
		parsePreviousIR,
		parseNextIR,
	)
	if parseNoOpErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream(equal IRs) error = %v", parseNoOpErr)
	}
	if !parseNoOp {
		parseTesting.Fatal("BuildCanonicalPatchStream(equal IRs) expected no-op result")
	}
	if parseNoOpPatch.GetPatchIdentity != "" || len(parseNoOpPatch.GetOps) != 0 {
		parseTesting.Fatalf("BuildCanonicalPatchStream(equal IRs) patch = %+v, want empty patch", parseNoOpPatch)
	}
}

// TestBuildCanonicalReplaceSubtreePatchStreamProducesExpectedPatch verifies the replace-subtree fallback emits a single replace-subtree operation with matching payload metadata.
func TestBuildCanonicalReplaceSubtreePatchStreamProducesExpectedPatch(parseTesting *testing.T) {
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "section",
		"children": []any{
			map[string]any{"kind": "text", "text": "after"},
		},
	})
	parsePatchStream, parseErr := buildCanonicalReplaceSubtreePatchStream(
		"region-a",
		1,
		2,
		3,
		parseNextIR.GetRootNodeID,
		parseNextIR,
	)
	if parseErr != nil {
		parseTesting.Fatalf("buildCanonicalReplaceSubtreePatchStream returned error: %v", parseErr)
	}
	if parsePatchStream.GetHeader.RegionID != "region-a" || parsePatchStream.GetHeader.Epoch != 1 || parsePatchStream.GetHeader.InputVersion != 2 || parsePatchStream.GetHeader.PatchVersion != 3 {
		parseTesting.Fatalf("buildCanonicalReplaceSubtreePatchStream header = %+v, want requested header values", parsePatchStream.GetHeader)
	}
	if len(parsePatchStream.GetOps) != 1 {
		parseTesting.Fatalf("buildCanonicalReplaceSubtreePatchStream ops = %d, want 1", len(parsePatchStream.GetOps))
	}
	parseOpCode, parseOpCodeErr := ParsePatchOpCode(parsePatchStream.GetOps[0].GetOpCode)
	if parseOpCodeErr != nil {
		parseTesting.Fatalf("ParsePatchOpCode returned error: %v", parseOpCodeErr)
	}
	if parseOpCode != PatchOpCodeReplaceSubtree {
		parseTesting.Fatalf("buildCanonicalReplaceSubtreePatchStream op code = %q, want replace-subtree", parseOpCode)
	}
	if parsePatchStream.GetOps[0].GetReplaceSubtreeOp == nil {
		parseTesting.Fatal("buildCanonicalReplaceSubtreePatchStream expected replace-subtree payload")
	}
	if parsePatchStream.GetOps[0].GetReplaceSubtreeOp.TargetNodeID != parseNextIR.GetRootNodeID {
		parseTesting.Fatalf("buildCanonicalReplaceSubtreePatchStream target node ID = %d, want %d", parsePatchStream.GetOps[0].GetReplaceSubtreeOp.TargetNodeID, parseNextIR.GetRootNodeID)
	}
	if parsePatchStream.GetOps[0].GetReplaceSubtreeOp.Subtree.RootNodeID != parseNextIR.GetRootNodeID {
		parseTesting.Fatalf("buildCanonicalReplaceSubtreePatchStream subtree root = %d, want %d", parsePatchStream.GetOps[0].GetReplaceSubtreeOp.Subtree.RootNodeID, parseNextIR.GetRootNodeID)
	}
}

// TestBuildCanonicalReplaceSubtreePatchStreamRejectsInvalidHeader verifies the fallback helper rejects malformed patch headers before emitting a patch stream.
func TestBuildCanonicalReplaceSubtreePatchStreamRejectsInvalidHeader(parseTesting *testing.T) {
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "text",
		"text": "after",
	})
	_, parseErr := buildCanonicalReplaceSubtreePatchStream(
		"",
		1,
		2,
		3,
		parseNextIR.GetRootNodeID,
		parseNextIR,
	)
	if parseErr == nil {
		parseTesting.Fatal("buildCanonicalReplaceSubtreePatchStream(blank region) error = nil, want error")
	}
}

// TestBuildCanonicalPatchKeyedMoveOpsWarnsForLargeSiblingSets verifies the keyed-move builder emits move ops for large keyed sibling orders that exceed the soft warning threshold.
func TestBuildCanonicalPatchKeyedMoveOpsWarnsForLargeSiblingSets(parseTesting *testing.T) {
	parseChildCount := getPatchMoveSiblingWarnLimit + 1
	parsePreviousChildIDs := make([]uint64, parseChildCount)
	for parseIndex := range parsePreviousChildIDs {
		parsePreviousChildIDs[parseIndex] = uint64(parseIndex + 1)
	}
	parseNextChildIDs := append([]uint64(nil), parsePreviousChildIDs[1:]...)
	parseNextChildIDs = append(parseNextChildIDs, parsePreviousChildIDs[0])

	parsePreviousTree := buildCanonicalKeyedMoveTreeForTest(100000, parsePreviousChildIDs)
	parseNextTree := buildCanonicalKeyedMoveTreeForTest(100000, parseNextChildIDs)
	if got := len(parseNextTree.getNodeByID[100000].getChildNodeIDs); got <= getPatchMoveSiblingWarnLimit {
		parseTesting.Fatalf("synthetic sibling count = %d, want > %d", got, getPatchMoveSiblingWarnLimit)
	}
	parseMoveOps, parseErr := buildCanonicalPatchKeyedMoveOps("region-a", parsePreviousTree, parseNextTree, map[uint64]struct{}{}, map[uint64]struct{}{}, false)
	if parseErr != nil {
		parseTesting.Fatalf("buildCanonicalPatchKeyedMoveOps(soft limit) error = %v", parseErr)
	}
	if len(parseMoveOps) == 0 {
		parseTesting.Fatal("buildCanonicalPatchKeyedMoveOps(soft limit) expected move ops")
	}
}

// TestBuildCanonicalPatchKeyedMoveOpsRejectsHardLimitSiblingSets verifies the keyed-move builder rejects sibling orders that exceed the hard guard limit.
func TestBuildCanonicalPatchKeyedMoveOpsRejectsHardLimitSiblingSets(parseTesting *testing.T) {
	parseChildCount := getPatchMoveSiblingHardLimit + 1
	parseChildIDs := make([]uint64, parseChildCount)
	for parseIndex := range parseChildIDs {
		parseChildIDs[parseIndex] = uint64(parseIndex + 1)
	}
	parsePreviousTree := buildCanonicalKeyedMoveTreeForTest(100000, parseChildIDs)
	parseNextTree := buildCanonicalKeyedMoveTreeForTest(100000, parseChildIDs)
	if got := len(parseNextTree.getNodeByID[100000].getChildNodeIDs); got <= getPatchMoveSiblingHardLimit {
		parseTesting.Fatalf("synthetic sibling count = %d, want > %d", got, getPatchMoveSiblingHardLimit)
	}
	_, parseErr := buildCanonicalPatchKeyedMoveOps("region-a", parsePreviousTree, parseNextTree, map[uint64]struct{}{}, map[uint64]struct{}{}, false)
	if parseErr == nil {
		parseTesting.Fatal("buildCanonicalPatchKeyedMoveOps(hard limit) error = nil, want error")
	}
}

// TestPatchStreamBuildSiblingIndexAndMoveHelpers verifies the sibling-index cache, insert-anchor search, and in-place move helper branches behave as expected.
func TestPatchStreamBuildSiblingIndexAndMoveHelpers(parseTesting *testing.T) {
	parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"children": []any{
			map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "d"},
		},
	})
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"children": []any{
			map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "c"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "d"},
		},
	})
	parsePreviousTree, parsePreviousTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
	if parsePreviousTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(previous) returned error: %v", parsePreviousTreeErr)
	}
	parseNextTree, parseNextTreeErr := ParseCanonicalRenderTree(parseNextIR)
	if parseNextTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseNextTreeErr)
	}

	parseRootNode := parseNextTree.getNodeByID[parseNextIR.GetRootNodeID]
	parseSiblingCache := parseBuildCanonicalSiblingIndexCache(parseNextTree)
	if parseIndex := parseGetCanonicalSiblingIndexFromCache(nil, parseNextIR.GetRootNodeID, parseRootNode.getChildNodeIDs[0]); parseIndex != -1 {
		parseTesting.Fatalf("parseGetCanonicalSiblingIndexFromCache(nil) = %d, want -1", parseIndex)
	}
	if parseIndex := parseGetCanonicalSiblingIndexFromCache(parseSiblingCache, 999, parseRootNode.getChildNodeIDs[0]); parseIndex != -1 {
		parseTesting.Fatalf("parseGetCanonicalSiblingIndexFromCache(missing parent) = %d, want -1", parseIndex)
	}
	if parseIndex := parseGetCanonicalSiblingIndexFromCache(parseSiblingCache, parseNextIR.GetRootNodeID, parseRootNode.getChildNodeIDs[2]); parseIndex != 2 {
		parseTesting.Fatalf("parseGetCanonicalSiblingIndexFromCache(valid) = %d, want 2", parseIndex)
	}
	if parseIndex := parseGetCanonicalSiblingIndexFromCache(parseSiblingCache, parseNextIR.GetRootNodeID, 999); parseIndex != -1 {
		parseTesting.Fatalf("parseGetCanonicalSiblingIndexFromCache(missing node) = %d, want -1", parseIndex)
	}

	buildInsertedNodeIDs := map[uint64]struct{}{}
	for _, getNodeID := range parseRootNode.getChildNodeIDs {
		if _, hasPrevious := parsePreviousTree.getNodeByID[getNodeID]; !hasPrevious {
			buildInsertedNodeIDs[getNodeID] = struct{}{}
		}
	}
	parseAnchorNodeID := parseFindCanonicalInsertAnchorWithIndexCache(
		parseNextIR.GetRootNodeID,
		parseRootNode.getChildNodeIDs[1],
		parsePreviousTree,
		parseNextTree,
		buildInsertedNodeIDs,
		parseSiblingCache,
	)
	if parseAnchorNodeID != parseRootNode.getChildNodeIDs[3] {
		parseTesting.Fatalf("parseFindCanonicalInsertAnchorWithIndexCache(valid) = %d, want anchor %d", parseAnchorNodeID, parseRootNode.getChildNodeIDs[3])
	}
	if parseAnchorNodeID := parseFindCanonicalInsertAnchorWithIndexCache(999, parseRootNode.getChildNodeIDs[1], parsePreviousTree, parseNextTree, buildInsertedNodeIDs, parseSiblingCache); parseAnchorNodeID != 0 {
		parseTesting.Fatalf("parseFindCanonicalInsertAnchorWithIndexCache(missing parent) = %d, want 0", parseAnchorNodeID)
	}

	parseNodeIDs := []uint64{1, 2, 3, 4}
	parseIndexByNode := parseBuildCanonicalSiblingIndexMap(parseNodeIDs)
	parseMoveCanonicalNodeIDInPlace(parseNodeIDs, 1, 3, parseIndexByNode)
	if parseNodeIDs[0] != 1 || parseNodeIDs[1] != 3 || parseNodeIDs[2] != 4 || parseNodeIDs[3] != 2 {
		parseTesting.Fatalf("parseMoveCanonicalNodeIDInPlace(forward) = %v, want [1 3 4 2]", parseNodeIDs)
	}
	if parseIndexByNode[2] != 3 || parseIndexByNode[3] != 1 {
		parseTesting.Fatalf("parseMoveCanonicalNodeIDInPlace(forward) index map = %+v, want updated indices", parseIndexByNode)
	}

	parseNodeIDs = []uint64{1, 2, 3, 4}
	parseIndexByNode = parseBuildCanonicalSiblingIndexMap(parseNodeIDs)
	parseMoveCanonicalNodeIDInPlace(parseNodeIDs, 3, -1, parseIndexByNode)
	if parseNodeIDs[0] != 4 || parseNodeIDs[1] != 1 || parseNodeIDs[2] != 2 || parseNodeIDs[3] != 3 {
		parseTesting.Fatalf("parseMoveCanonicalNodeIDInPlace(backward clamp) = %v, want [4 1 2 3]", parseNodeIDs)
	}
	if parseIndexByNode[4] != 0 || parseIndexByNode[1] != 1 {
		parseTesting.Fatalf("parseMoveCanonicalNodeIDInPlace(backward clamp) index map = %+v, want updated indices", parseIndexByNode)
	}

	parseNodeIDs = []uint64{1, 2, 3}
	parseIndexByNode = parseBuildCanonicalSiblingIndexMap(parseNodeIDs)
	parseMoveCanonicalNodeIDInPlace(parseNodeIDs, 1, 1, parseIndexByNode)
	if parseNodeIDs[0] != 1 || parseNodeIDs[1] != 2 || parseNodeIDs[2] != 3 {
		parseTesting.Fatalf("parseMoveCanonicalNodeIDInPlace(no-op) = %v, want unchanged slice", parseNodeIDs)
	}
	parseMoveCanonicalNodeIDInPlace(parseNodeIDs, -1, 1, parseIndexByNode)
	if parseNodeIDs[0] != 1 || parseNodeIDs[1] != 2 || parseNodeIDs[2] != 3 {
		parseTesting.Fatalf("parseMoveCanonicalNodeIDInPlace(out-of-range) = %v, want unchanged slice", parseNodeIDs)
	}
}

// TestBuildCanonicalPatchStreamRejectsInvalidCanonicalInput verifies invalid canonical trees fail fast before diffing.
func TestBuildCanonicalPatchStreamRejectsInvalidCanonicalInput(parseTesting *testing.T) {
	parseValidIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "text",
		"text": "ok",
	})
	parseInvalidIR := CanonicalRenderIR{}

	if _, _, parseErr := BuildCanonicalPatchStream("region-a", 1, 2, 3, parseInvalidIR, parseValidIR); parseErr == nil {
		parseTesting.Fatal("BuildCanonicalPatchStream(invalid previous IR) error = nil, want error")
	}
	if _, _, parseErr := BuildCanonicalPatchStream("region-a", 1, 2, 3, parseValidIR, parseInvalidIR); parseErr == nil {
		parseTesting.Fatal("BuildCanonicalPatchStream(invalid next IR) error = nil, want error")
	}
}

// TestBuildCanonicalPatchStreamHandlesHostTagMismatchAndTextAdjacentProps verifies host-tag mismatches and text-adjacent props both flow through the replace-subtree and skip branches.
func TestBuildCanonicalPatchStreamHandlesHostTagMismatchAndTextAdjacentProps(parseTesting *testing.T) {
	parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": "card",
			"text":  "before",
		},
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "a",
				"props": map[string]any{
					"text": "old",
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "b",
				"props": map[string]any{
					"class": "keep",
					"text":  "before",
				},
			},
		},
	})
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "section",
		"props": map[string]any{
			"class": "card-updated",
			"text":  "after",
		},
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "b",
				"props": map[string]any{
					"class": "changed",
					"text":  "after",
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "c",
				"props": map[string]any{
					"text": "new",
				},
			},
		},
	})
	parsePatchStream, hasNoOp, parseErr := BuildCanonicalPatchStream("region-a", 1, 2, 3, parsePreviousIR, parseNextIR)
	if parseErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream(tag mismatch + text props) error = %v", parseErr)
	}
	if hasNoOp {
		parseTesting.Fatal("BuildCanonicalPatchStream(tag mismatch + text props) expected patch output")
	}
	hasReplaceSubtree := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, parseOpCodeErr := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCodeErr != nil {
			parseTesting.Fatalf("ParsePatchOpCode returned error: %v", parseOpCodeErr)
		}
		if parseOpCode == PatchOpCodeReplaceSubtree {
			hasReplaceSubtree = true
		}
	}
	if !hasReplaceSubtree {
		parseTesting.Fatal("BuildCanonicalPatchStream(tag mismatch + text props) expected replace-subtree output")
	}
}

// TestBuildCanonicalPatchStreamEmitsMixedOpsForStructuralAndPropDelta verifies one mixed diff emits the expected patch op families for inserts, removes, moves, text updates, and prop/style changes.
func TestBuildCanonicalPatchStreamEmitsMixedOpsForStructuralAndPropDelta(parseTesting *testing.T) {
	parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"props": map[string]any{
			"class":     "old-root",
			"style":     "color:red",
			"data-drop": "gone",
		},
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "a",
				"props": map[string]any{
					"class": "stay",
					"style": "font-weight:bold",
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "alpha"},
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "b",
				"props": map[string]any{
					"class": "before",
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "d",
			},
		},
	})
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"props": map[string]any{
			"class": "new-root",
			"id":    "root",
			"style": "color:blue",
		},
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "b",
				"props": map[string]any{
					"class": "after",
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "c",
				"props": map[string]any{
					"data-kind": "fresh",
					"style":     "color:green",
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "gamma"},
				},
			},
			map[string]any{
				"kind": "host-element",
				"tag":  "li",
				"key":  "a",
				"props": map[string]any{
					"class": "stay",
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "alpha-updated"},
				},
			},
		},
	})

	parsePatchStream, hasNoOp, parseErr := BuildCanonicalPatchStream("region-a", 1, 2, 3, parsePreviousIR, parseNextIR)
	if parseErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream(mixed delta) error = %v", parseErr)
	}
	if hasNoOp {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected patch output")
	}

	parseOpCounts := map[PatchOpCode]int{}
	hasInsertedTextNode := false
	hasInsertedKeyedNode := false
	hasAnchoredInsert := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, parseOpCodeErr := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCodeErr != nil {
			parseTesting.Fatalf("ParsePatchOpCode returned error: %v", parseOpCodeErr)
		}
		parseOpCounts[parseOpCode]++
		if parseOpCode != PatchOpCodeInsertNode || getOp.GetInsertOp == nil {
			continue
		}
		if getOp.GetInsertOp.AnchorNodeID != 0 {
			hasAnchoredInsert = true
		}
		if RenderNodeKind(getOp.GetInsertOp.Node.Kind) == RenderNodeKindText {
			hasInsertedTextNode = true
			continue
		}
		if getOp.GetInsertOp.Node.KeyText == "c" {
			hasInsertedKeyedNode = true
		}
	}

	if parseOpCounts[PatchOpCodeRemoveNode] == 0 {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected remove-node op")
	}
	if !hasInsertedTextNode {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected inserted text-node op")
	}
	if !hasInsertedKeyedNode {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected inserted keyed host-node op")
	}
	if !hasAnchoredInsert {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected non-root insert anchor")
	}
	if parseOpCounts[PatchOpCodeSetText] == 0 {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected set-text op")
	}
	if parseOpCounts[PatchOpCodeSetAttr] == 0 {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected set-attr op")
	}
	if parseOpCounts[PatchOpCodeSetStyle] == 0 {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected set-style op")
	}
	if parseOpCounts[PatchOpCodeRemoveAttr] == 0 {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected remove-attr op")
	}
	if parseOpCounts[PatchOpCodeRemoveStyle] == 0 {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected remove-style op")
	}
	if parseOpCounts[PatchOpCodeMoveKeyedChild] == 0 {
		parseTesting.Fatal("BuildCanonicalPatchStream(mixed delta) expected keyed-move op")
	}
}
