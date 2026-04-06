package runtime2

import (
	"strings"
	"testing"
)

func parseBuildCanonicalIRForTest(parseTesting *testing.T, parseRenderOutput any) CanonicalRenderIR {
	parseTesting.Helper()
	parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	return parseCanonicalIR
}

func parseBuildPatchStreamForTest(
	parseTesting *testing.T,
	parseRegionID string,
	parseEpoch uint64,
	parseInputVersion uint64,
	parsePatchVersion uint64,
	parsePreviousOutput any,
	parseNextOutput any,
) PatchStreamRaw {
	parseTesting.Helper()
	parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, parsePreviousOutput)
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, parseNextOutput)
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream(
		parseRegionID,
		parseEpoch,
		parseInputVersion,
		parsePatchVersion,
		parsePreviousIR,
		parseNextIR,
	)
	if parsePatchErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseTesting.Fatal("BuildCanonicalPatchStream returned no-op for differing payloads")
	}
	return parsePatchStream
}

// parseBuildCanonicalIRWithShiftedRootIDForTest rewrites one canonical IR root node ID to exercise root-delta fallback behavior.
func parseBuildCanonicalIRWithShiftedRootIDForTest(parseTesting *testing.T, parseCanonicalIR CanonicalRenderIR) CanonicalRenderIR {
	parseTesting.Helper()
	if len(parseCanonicalIR.GetNodeRecords) == 0 {
		parseTesting.Fatal("expected canonical IR node records")
	}
	buildNodeRecords := append([]RenderNodeRecordRaw(nil), parseCanonicalIR.GetNodeRecords...)
	buildExistingNodeIDs := make(map[uint64]struct{}, len(buildNodeRecords))
	for _, getNodeRecord := range buildNodeRecords {
		buildExistingNodeIDs[getNodeRecord.NodeID] = struct{}{}
	}
	buildRootNodeID := parseCanonicalIR.GetRootNodeID
	buildRootRecordIndex := -1
	for parseNodeIndex, getNodeRecord := range buildNodeRecords {
		if getNodeRecord.NodeID == buildRootNodeID {
			buildRootRecordIndex = parseNodeIndex
			break
		}
	}
	if buildRootRecordIndex < 0 {
		parseTesting.Fatalf("expected root node record %d in canonical IR", buildRootNodeID)
	}
	delete(buildExistingNodeIDs, buildRootNodeID)
	buildShiftedRootNodeID := buildRootNodeID + 1
	if buildShiftedRootNodeID == 0 {
		buildShiftedRootNodeID = 1
	}
	for {
		if _, hasNodeID := buildExistingNodeIDs[buildShiftedRootNodeID]; !hasNodeID {
			break
		}
		buildShiftedRootNodeID++
		if buildShiftedRootNodeID == 0 {
			buildShiftedRootNodeID = 1
		}
	}
	buildNodeRecords[buildRootRecordIndex].NodeID = buildShiftedRootNodeID
	buildShiftedIR := CanonicalRenderIR{
		GetRootNodeID:  buildShiftedRootNodeID,
		GetStringTable: parseCanonicalIR.GetStringTable,
		GetNodeRecords: buildNodeRecords,
		GetPropRecords: append([]RenderPropRecordRaw(nil), parseCanonicalIR.GetPropRecords...),
	}
	if _, parseTreeErr := ParseCanonicalRenderTree(buildShiftedIR); parseTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(shifted root) returned error: %v", parseTreeErr)
	}
	return buildShiftedIR
}

// TestParsePatchStreamTransactionRejectsExcessiveOpCount verifies patch parsing fails fast when op volume exceeds the hard guard limit.
func TestParsePatchStreamTransactionRejectsExcessiveOpCount(parseTesting *testing.T) {
	parsePatchStreamRaw := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-a",
			Epoch:           1,
			InputVersion:    1,
			PatchVersion:    1,
		},
		GetOps:           make([]PatchStreamOpRaw, getPatchStreamOpHardLimit+1),
		GetPatchIdentity: "guard-limit",
	}
	_, _, parsePatchErr := ParsePatchStreamTransaction(parsePatchStreamRaw, "region-a", 1, nil, nil, nil)
	if parsePatchErr == nil || !strings.Contains(parsePatchErr.Error(), "exceeds guard limit") {
		parseTesting.Fatalf("ParsePatchStreamTransaction(guard) error = %v, want guard-limit error", parsePatchErr)
	}
}

// TestBuildCanonicalPatchStreamGeneratesTextAndAttrOps verifies text, set-attr, and remove-attr operations are generated.
func TestBuildCanonicalPatchStreamGeneratesTextAndAttrOps(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"class":    "active",
				"data-old": "legacy",
			},
			"children": []any{
				map[string]any{"kind": "text", "text": "before"},
			},
		},
		map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"class": "idle",
			},
			"children": []any{
				map[string]any{"kind": "text", "text": "after"},
			},
		},
	)
	hasSetText := false
	hasSetAttr := false
	hasRemoveAttr := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, _ := ParsePatchOpCode(getOp.GetOpCode)
		switch parseOpCode {
		case PatchOpCodeSetText:
			hasSetText = true
		case PatchOpCodeSetAttr:
			hasSetAttr = true
		case PatchOpCodeRemoveAttr:
			hasRemoveAttr = true
		}
	}
	if !hasSetText {
		parseTesting.Fatal("expected set-text op in patch stream")
	}
	if !hasSetAttr {
		parseTesting.Fatal("expected set-attr op in patch stream")
	}
	if !hasRemoveAttr {
		parseTesting.Fatal("expected remove-attr op in patch stream")
	}
}

// TestBuildCanonicalPatchStreamGeneratesSetStyle verifies normalized style changes emit set-style operations.
func TestBuildCanonicalPatchStreamGeneratesSetStyle(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"style": map[string]any{
					"color": "red",
				},
			},
		},
		map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"style": map[string]any{
					"color": "blue",
				},
			},
		},
	)
	hasSetStyle := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, _ := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCode == PatchOpCodeSetStyle {
			hasSetStyle = true
		}
	}
	if !hasSetStyle {
		parseTesting.Fatal("expected set-style op in patch stream")
	}
}

// TestBuildCanonicalPatchStreamGeneratesRemoveStyle verifies style removals emit remove-style operations.
func TestBuildCanonicalPatchStreamGeneratesRemoveStyle(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"style": map[string]any{
					"color": "red",
				},
				"class": "card",
			},
		},
		map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"class": "card",
			},
		},
	)
	hasRemoveStyle := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, _ := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCode == PatchOpCodeRemoveStyle {
			hasRemoveStyle = true
		}
	}
	if !hasRemoveStyle {
		parseTesting.Fatal("expected remove-style op in patch stream")
	}
}

// TestBuildCanonicalPatchStreamGeneratesReplaceSubtree verifies structural mismatches emit replace-subtree operations.
func TestBuildCanonicalPatchStreamGeneratesReplaceSubtree(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{"kind": "text", "text": "before"},
		map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"props": map[string]any{
				"class": "card",
			},
			"children": []any{
				map[string]any{"kind": "text", "text": "after"},
			},
		},
	)
	hasReplaceSubtree := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, _ := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCode == PatchOpCodeReplaceSubtree {
			hasReplaceSubtree = true
		}
	}
	if !hasReplaceSubtree {
		parseTesting.Fatal("expected replace-subtree op in patch stream")
	}
}

// TestBuildCanonicalPatchStreamRootIDDeltaFallsBackToReplaceSubtree verifies root insert-remove deltas reuse replace-subtree fallback instead of erroring.
func TestBuildCanonicalPatchStreamRootIDDeltaFallsBackToReplaceSubtree(parseTesting *testing.T) {
	parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{"kind": "text", "text": "before"},
		},
	})
	parseNextIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{"kind": "text", "text": "after"},
		},
	})
	parseShiftedNextIR := parseBuildCanonicalIRWithShiftedRootIDForTest(parseTesting, parseNextIR)
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream(
		"region-a",
		1,
		2,
		2,
		parsePreviousIR,
		parseShiftedNextIR,
	)
	if parsePatchErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseTesting.Fatal("BuildCanonicalPatchStream returned no-op for shifted root payloads")
	}
	if len(parsePatchStream.GetOps) != 1 {
		parseTesting.Fatalf("expected one replace-subtree op, got %d ops", len(parsePatchStream.GetOps))
	}
	parseOpCode, parseOpCodeErr := ParsePatchOpCode(parsePatchStream.GetOps[0].GetOpCode)
	if parseOpCodeErr != nil {
		parseTesting.Fatalf("ParsePatchOpCode returned error: %v", parseOpCodeErr)
	}
	if parseOpCode != PatchOpCodeReplaceSubtree {
		parseTesting.Fatalf("expected replace-subtree op, got %q", parseOpCode)
	}
	if parsePatchStream.GetOps[0].GetReplaceSubtreeOp == nil {
		parseTesting.Fatal("expected replace-subtree payload")
	}
	if parsePatchStream.GetOps[0].GetReplaceSubtreeOp.TargetNodeID != parsePreviousIR.GetRootNodeID {
		parseTesting.Fatalf(
			"replace-subtree target node ID = %d, want previous root %d",
			parsePatchStream.GetOps[0].GetReplaceSubtreeOp.TargetNodeID,
			parsePreviousIR.GetRootNodeID,
		)
	}
	if parsePatchStream.GetOps[0].GetReplaceSubtreeOp.Subtree.RootNodeID != parseShiftedNextIR.GetRootNodeID {
		parseTesting.Fatalf(
			"replace-subtree subtree root node ID = %d, want shifted next root %d",
			parsePatchStream.GetOps[0].GetReplaceSubtreeOp.Subtree.RootNodeID,
			parseShiftedNextIR.GetRootNodeID,
		)
	}
}

// TestBuildCanonicalPatchStreamGeneratesInsertAndRemove verifies insert and remove operations are generated.
func TestBuildCanonicalPatchStreamGeneratesInsertAndRemove(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{
			"kind": "host-element",
			"tag":  "ul",
			"children": []any{
				map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
			},
		},
		map[string]any{
			"kind": "host-element",
			"tag":  "ul",
			"children": []any{
				map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "c"},
			},
		},
	)
	hasInsert := false
	hasRemove := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, _ := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCode == PatchOpCodeInsertNode {
			hasInsert = true
		}
		if parseOpCode == PatchOpCodeRemoveNode {
			hasRemove = true
		}
	}
	if !hasInsert {
		parseTesting.Fatal("expected insert op in patch stream")
	}
	if !hasRemove {
		parseTesting.Fatal("expected remove op in patch stream")
	}
}

// TestBuildCanonicalPatchStreamGeneratesKeyedMove verifies keyed sibling reorder emits keyed-move operations.
func TestBuildCanonicalPatchStreamGeneratesKeyedMove(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{
			"kind": "host-element",
			"tag":  "ul",
			"children": []any{
				map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "c"},
			},
		},
		map[string]any{
			"kind": "host-element",
			"tag":  "ul",
			"children": []any{
				map[string]any{"kind": "host-element", "tag": "li", "key": "c"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
			},
		},
	)
	hasMove := false
	for _, getOp := range parsePatchStream.GetOps {
		parseOpCode, _ := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCode == PatchOpCodeMoveKeyedChild {
			hasMove = true
			break
		}
	}
	if !hasMove {
		parseTesting.Fatal("expected keyed-move op in patch stream")
	}
}

// TestBuildCanonicalPatchStreamCanonicalOrdering verifies remove operations are emitted before insert operations.
func TestBuildCanonicalPatchStreamCanonicalOrdering(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{
			"kind": "host-element",
			"tag":  "ul",
			"children": []any{
				map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
			},
		},
		map[string]any{
			"kind": "host-element",
			"tag":  "ul",
			"children": []any{
				map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
				map[string]any{"kind": "host-element", "tag": "li", "key": "c"},
			},
		},
	)
	parseFirstInsertIndex := -1
	parseFirstRemoveIndex := -1
	for parseIndex, getOp := range parsePatchStream.GetOps {
		parseOpCode, _ := ParsePatchOpCode(getOp.GetOpCode)
		if parseOpCode == PatchOpCodeRemoveNode && parseFirstRemoveIndex < 0 {
			parseFirstRemoveIndex = parseIndex
		}
		if parseOpCode == PatchOpCodeInsertNode && parseFirstInsertIndex < 0 {
			parseFirstInsertIndex = parseIndex
		}
	}
	if parseFirstInsertIndex < 0 || parseFirstRemoveIndex < 0 {
		parseTesting.Fatalf("expected both remove and insert ops, got %+v", parsePatchStream.GetOps)
	}
	if parseFirstRemoveIndex > parseFirstInsertIndex {
		parseTesting.Fatalf("expected remove ops before insert ops, got remove=%d insert=%d", parseFirstRemoveIndex, parseFirstInsertIndex)
	}
}

// TestParsePatchStreamTransactionRejectsRegionMismatch verifies region mismatch headers are rejected.
func TestParsePatchStreamTransactionRejectsRegionMismatch(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{"kind": "text", "text": "a"},
		map[string]any{"kind": "text", "text": "b"},
	)
	parseKnownNodeIDs := map[uint64]struct{}{}
	_, _, parseErr := ParsePatchStreamTransaction(parsePatchStream, "region-b", 1, parseKnownNodeIDs, map[uint64]uint32{}, BuildPatchIdempotencyTracker())
	if parseErr == nil {
		parseTesting.Fatal("expected region mismatch to fail patch parse")
	}
	if !strings.Contains(parseErr.Error(), "does not match expected") {
		parseTesting.Fatalf("expected region mismatch details, got %v", parseErr)
	}
}

// TestParsePatchStreamTransactionRejectsEpochMismatch verifies epoch mismatch headers are rejected.
func TestParsePatchStreamTransactionRejectsEpochMismatch(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{"kind": "text", "text": "a"},
		map[string]any{"kind": "text", "text": "b"},
	)
	parseKnownNodeIDs := map[uint64]struct{}{}
	_, _, parseErr := ParsePatchStreamTransaction(parsePatchStream, "region-a", 9, parseKnownNodeIDs, map[uint64]uint32{}, BuildPatchIdempotencyTracker())
	if parseErr == nil {
		parseTesting.Fatal("expected epoch mismatch to fail patch parse")
	}
	if !strings.Contains(parseErr.Error(), "does not match expected epoch") {
		parseTesting.Fatalf("expected epoch mismatch details, got %v", parseErr)
	}
}

// TestParsePatchStreamTransactionIdempotencyDuplicateIgnored verifies duplicate patch identities are ignored by parser tracking.
func TestParsePatchStreamTransactionIdempotencyDuplicateIgnored(parseTesting *testing.T) {
	parsePatchStream := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		2,
		map[string]any{"kind": "text", "text": "before"},
		map[string]any{"kind": "text", "text": "after"},
	)
	parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{"kind": "text", "text": "before"})
	parsePreviousTree, parseTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
	if parseTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree returned error: %v", parseTreeErr)
	}
	parseKnownNodeIDs := make(map[uint64]struct{}, len(parsePreviousTree.getNodeByID))
	for getNodeID := range parsePreviousTree.getNodeByID {
		parseKnownNodeIDs[getNodeID] = struct{}{}
	}
	parseTracker := BuildPatchIdempotencyTracker()
	_, hasApplyFirst, parseFirstErr := ParsePatchStreamTransaction(parsePatchStream, "region-a", 1, parseKnownNodeIDs, map[uint64]uint32{}, parseTracker)
	if parseFirstErr != nil {
		parseTesting.Fatalf("ParsePatchStreamTransaction(first) returned error: %v", parseFirstErr)
	}
	if !hasApplyFirst {
		parseTesting.Fatal("expected first patch parse to apply")
	}
	_, hasApplySecond, parseSecondErr := ParsePatchStreamTransaction(parsePatchStream, "region-a", 1, parseKnownNodeIDs, map[uint64]uint32{}, parseTracker)
	if parseSecondErr != nil {
		parseTesting.Fatalf("ParsePatchStreamTransaction(second) returned error: %v", parseSecondErr)
	}
	if hasApplySecond {
		parseTesting.Fatal("expected duplicate patch identity to be ignored")
	}
}
