package runtime2

import (
	"strings"
	"testing"
)

// TestPatchStreamIdentityHelpersAndRawBuilders covers the identity writer, raw builder, and guard branches.
func TestPatchStreamIdentityHelpersAndRawBuilders(parseTesting *testing.T) {
	parseHeader := PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        "region-a",
		Epoch:           1,
		InputVersion:    2,
		PatchVersion:    3,
	}

	if _, parseErr := buildPatchStreamRawWithIdentity(
		PatchStreamHeaderRaw{
			ProtocolVersion: "gwc.parallel.invalid",
			RegionID:        "region-a",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    3,
		},
		[]string{"a"},
		nil,
		"identity",
	); parseErr == nil {
		parseTesting.Fatal("expected invalid patch-stream header to fail")
	}

	if parseErr := appendPatchStreamIdentityOp(nil, PatchStreamOpRaw{}); parseErr == nil {
		parseTesting.Fatal("expected nil patch identity state to fail")
	}
	if _, parseErr := formatPatchStreamIdentityState(nil); parseErr == nil {
		parseTesting.Fatal("expected nil patch identity state finalization to fail")
	}
	startPatchStreamIdentityOpArray(nil)
	clearPatchStreamIdentityState(nil)

	parseState, parseStateErr := buildPatchStreamIdentityState(parseHeader, []string{"a"})
	if parseStateErr != nil {
		parseTesting.Fatalf("buildPatchStreamIdentityState returned error: %v", parseStateErr)
	}
	if parseErr := appendPatchStreamIdentityOp(&parseState, PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeSetText)}); parseErr == nil {
		parseTesting.Fatal("expected append before array start to fail")
	}
	if parseErr := startPatchStreamIdentityOpArray(&parseState); parseErr != nil {
		parseTesting.Fatalf("startPatchStreamIdentityOpArray returned error: %v", parseErr)
	}
	if parseErr := startPatchStreamIdentityOpArray(&parseState); parseErr != nil {
		parseTesting.Fatalf("startPatchStreamIdentityOpArray(second) returned error: %v", parseErr)
	}
	if parseErr := appendPatchStreamIdentityOp(&parseState, PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeSetText)}); parseErr != nil {
		parseTesting.Fatalf("appendPatchStreamIdentityOp returned error: %v", parseErr)
	}
	if parseIdentity, parseErr := formatPatchStreamIdentityState(&parseState); parseErr != nil || parseIdentity == "" {
		parseTesting.Fatalf("formatPatchStreamIdentityState returned identity=%q error=%v", parseIdentity, parseErr)
	}
	clearPatchStreamIdentityState(&parseState)

	parseNullState, parseNullStateErr := buildPatchStreamIdentityState(parseHeader, []string{"a"})
	if parseNullStateErr != nil {
		parseTesting.Fatalf("buildPatchStreamIdentityState(null close) returned error: %v", parseNullStateErr)
	}
	if parseIdentity, parseErr := formatPatchStreamIdentityState(&parseNullState); parseErr != nil || parseIdentity == "" {
		parseTesting.Fatalf("formatPatchStreamIdentityState(null close) returned identity=%q error=%v", parseIdentity, parseErr)
	}
	clearPatchStreamIdentityState(&parseNullState)

	parseRaw, parseErr := BuildPatchStreamRaw(parseHeader, BuildRenderStringTable([]string{"a"}), []PatchStreamOpRaw{
		{GetOpCode: uint8(PatchOpCodeSetText)},
	})
	if parseErr != nil {
		parseTesting.Fatalf("BuildPatchStreamRaw returned error: %v", parseErr)
	}
	if parseRaw.GetHeader.RegionID != "region-a" || len(parseRaw.GetOps) != 1 {
		parseTesting.Fatalf("BuildPatchStreamRaw returned %+v, want one-op patch", parseRaw)
	}
	if _, parseErr := BuildPatchStreamRaw(
		PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    3,
		},
		BuildRenderStringTable([]string{"a"}),
		nil,
	); parseErr == nil {
		parseTesting.Fatal("expected BuildPatchStreamRaw to reject invalid header")
	}
}

// TestPatchStreamScratchAndLookupHelpers covers scratch-map pooling, mutable copy helpers, and lookup helpers.
func TestPatchStreamScratchAndLookupHelpers(parseTesting *testing.T) {
	parseKnownScratch := buildPatchStreamKnownNodeIDScratchMap()
	parseKnownScratch[1] = struct{}{}
	clearPatchStreamKnownNodeIDScratchMap(parseKnownScratch)
	if len(parseKnownScratch) != 0 {
		parseTesting.Fatalf("expected known-node scratch map to be cleared, got %d entries", len(parseKnownScratch))
	}

	parseSiblingScratch := buildPatchStreamSiblingCountScratchMap()
	parseSiblingScratch[1] = 2
	clearPatchStreamSiblingCountScratchMap(parseSiblingScratch)
	if len(parseSiblingScratch) != 0 {
		parseTesting.Fatalf("expected sibling-count scratch map to be cleared, got %d entries", len(parseSiblingScratch))
	}

	parseRemovedNodeScratch := buildPatchStreamRemovedNodeIDScratchMap()
	parseRemovedNodeScratch[1] = struct{}{}
	clearPatchStreamRemovedNodeIDScratchMap(parseRemovedNodeScratch)
	if len(parseRemovedNodeScratch) != 0 {
		parseTesting.Fatalf("expected removed-node scratch map to be cleared, got %d entries", len(parseRemovedNodeScratch))
	}

	parseRemovedAttrScratch := buildPatchStreamRemovedAttrKeyScratchMap()
	parseRemovedAttrScratch[patchRemovedAttrKey{parseTargetNodeID: 1, parseKey: "class"}] = struct{}{}
	clearPatchStreamRemovedAttrKeyScratchMap(parseRemovedAttrScratch)
	if len(parseRemovedAttrScratch) != 0 {
		parseTesting.Fatalf("expected removed-attr scratch map to be cleared, got %d entries", len(parseRemovedAttrScratch))
	}

	parseKnownNodeIDs := map[uint64]struct{}{1: {}, 2: {}}
	var parseMutableKnownNodeIDs map[uint64]struct{}
	parseKnownCopied := false
	parseCopiedKnownNodeIDs := getPatchMutableKnownNodeIDs(parseKnownNodeIDs, &parseMutableKnownNodeIDs, &parseKnownCopied)
	if !parseKnownCopied || len(parseCopiedKnownNodeIDs) != 2 {
		parseTesting.Fatalf("expected copied known-node map, got len=%d copied=%t", len(parseCopiedKnownNodeIDs), parseKnownCopied)
	}
	parseMutableKnownNodeIDs = nil
	parseCopiedKnownNodeIDs = getPatchMutableKnownNodeIDs(parseKnownNodeIDs, &parseMutableKnownNodeIDs, &parseKnownCopied)
	if len(parseCopiedKnownNodeIDs) != 0 {
		parseTesting.Fatalf("expected nil cached known-node map to stay empty, got len=%d", len(parseCopiedKnownNodeIDs))
	}

	parseSiblingCounts := map[uint64]uint32{1: 2}
	var parseMutableSiblingCounts map[uint64]uint32
	parseSiblingCopied := false
	parseCopiedSiblingCounts := getPatchMutableSiblingCountByParent(parseSiblingCounts, &parseMutableSiblingCounts, &parseSiblingCopied)
	if !parseSiblingCopied || len(parseCopiedSiblingCounts) != 1 {
		parseTesting.Fatalf("expected copied sibling-count map, got len=%d copied=%t", len(parseCopiedSiblingCounts), parseSiblingCopied)
	}
	parseMutableSiblingCounts = nil
	parseCopiedSiblingCounts = getPatchMutableSiblingCountByParent(parseSiblingCounts, &parseMutableSiblingCounts, &parseSiblingCopied)
	if len(parseCopiedSiblingCounts) != 0 {
		parseTesting.Fatalf("expected nil cached sibling-count map to stay empty, got len=%d", len(parseCopiedSiblingCounts))
	}

	parseOps := []PatchStreamOpRaw{
		{GetOpCode: uint8(PatchOpCodeSetText)},
		{GetOpCode: uint8(PatchOpCodeMoveKeyedChild)},
	}
	if !parseHasPatchKeyedMoveOp(parseOps) {
		parseTesting.Fatal("expected keyed-move op to be detected")
	}
	if !parseHasPatchKeyedMoveOpFromIndex(parseOps, -1) {
		parseTesting.Fatal("expected keyed-move op to be detected from negative index")
	}
	if parseHasPatchKeyedMoveOpFromIndex(parseOps, len(parseOps)) {
		parseTesting.Fatal("expected keyed-move lookup beyond range to return false")
	}
	parseHasMove := false
	parseHasMoveKnown := false
	if !hasPatchTrackSiblingCountByParent(parseOps, map[uint64]uint32{1: 2}, &parseHasMove, &parseHasMoveKnown, 0) {
		parseTesting.Fatal("expected sibling-count tracking to be required when keyed moves are present")
	}
	parseHasMoveKnown = true
	if hasPatchTrackSiblingCountByParent(parseOps, map[uint64]uint32{1: 2}, nil, &parseHasMoveKnown, 0) {
		parseTesting.Fatal("expected cached keyed-move lookup with nil result slot to return false")
	}
	parseHasMoveKnown = false
	if hasPatchTrackSiblingCountByParent(parseOps, map[uint64]uint32{}, &parseHasMove, &parseHasMoveKnown, 0) {
		parseTesting.Fatal("expected empty sibling-count map to skip keyed-move tracking")
	}
	if hasPatchOnlyAppendInsertOps(nil) {
		parseTesting.Fatal("expected nil append-only patch stream to return false")
	}
	if hasPatchOnlyAppendInsertOps([]PatchStreamOpRaw{{GetOpCode: uint8(PatchOpCodeInsertNode), GetInsertOp: &PatchInsertOpRaw{ParentNodeID: 1, Node: RenderNodeRecordRaw{NodeID: 2}}}}) {
		parseTesting.Fatal("expected single-op append-only patch stream to return false")
	}
	if !hasPatchOnlyAppendInsertOps([]PatchStreamOpRaw{
		{GetOpCode: uint8(PatchOpCodeInsertNode), GetInsertOp: &PatchInsertOpRaw{ParentNodeID: 1, Node: RenderNodeRecordRaw{NodeID: 2}}},
		{GetOpCode: uint8(PatchOpCodeInsertNode), GetInsertOp: &PatchInsertOpRaw{ParentNodeID: 2, Node: RenderNodeRecordRaw{NodeID: 3}}},
	}) {
		parseTesting.Fatal("expected multi-op anchor-free insert patch stream to be append-only")
	}
	if hasPatchOnlyAppendInsertOps([]PatchStreamOpRaw{
		{GetOpCode: uint8(PatchOpCodeInsertNode), GetInsertOp: &PatchInsertOpRaw{ParentNodeID: 1, AnchorNodeID: 7, Node: RenderNodeRecordRaw{NodeID: 2}}},
		{GetOpCode: uint8(PatchOpCodeInsertNode), GetInsertOp: &PatchInsertOpRaw{ParentNodeID: 1, Node: RenderNodeRecordRaw{NodeID: 3}}},
	}) {
		parseTesting.Fatal("expected anchored insert patch stream to skip append-only fast path")
	}
	if getStartIndex := findPatchRemoveOnlyStartIndex(nil); getStartIndex != 0 {
		parseTesting.Fatalf("findPatchRemoveOnlyStartIndex(nil) = %d, want 0", getStartIndex)
	}
	if getStartIndex := findPatchRemoveOnlyStartIndex([]PatchStreamOpRaw{
		{GetOpCode: uint8(PatchOpCodeSetText)},
		{GetOpCode: uint8(PatchOpCodeRemoveNode)},
		{GetOpCode: uint8(PatchOpCodeRemoveNode)},
	}); getStartIndex != 1 {
		parseTesting.Fatalf("findPatchRemoveOnlyStartIndex(set/remove/remove) = %d, want 1", getStartIndex)
	}
	if getStartIndex := findPatchRemoveOnlyStartIndex([]PatchStreamOpRaw{
		{GetOpCode: uint8(PatchOpCodeRemoveNode)},
		{GetOpCode: uint8(PatchOpCodeSetText)},
		{GetOpCode: uint8(PatchOpCodeRemoveNode)},
	}); getStartIndex != 2 {
		parseTesting.Fatalf("findPatchRemoveOnlyStartIndex(remove/set/remove) = %d, want 2", getStartIndex)
	}
	if getStartIndex := findPatchRemoveOnlyStartIndex([]PatchStreamOpRaw{
		{GetOpCode: uint8(PatchOpCodeSetText)},
		{GetOpCode: uint8(PatchOpCodeSetAttr)},
	}); getStartIndex != 2 {
		parseTesting.Fatalf("findPatchRemoveOnlyStartIndex(no remove suffix) = %d, want 2", getStartIndex)
	}

	if !parseHasCanonicalStringTableSortedUnique([]string{}) {
		parseTesting.Fatal("expected empty string table to be canonical")
	}
	if !parseHasCanonicalStringTableSortedUnique([]string{"a"}) {
		parseTesting.Fatal("expected single-entry string table to be canonical")
	}
	if !parseHasCanonicalStringTableSortedUnique([]string{"a", "b"}) {
		parseTesting.Fatal("expected sorted unique string table to be canonical")
	}
	if parseHasCanonicalStringTableSortedUnique([]string{"a", "a"}) {
		parseTesting.Fatal("expected duplicate string table entries to be rejected")
	}
	if parseHasCanonicalStringTableSortedUnique([]string{"b", "a"}) {
		parseTesting.Fatal("expected unsorted string table entries to be rejected")
	}

	parseStringTable := BuildRenderStringTable([]string{"div", "hello"})
	parseTextNode, parseTextErr := parseBuildRegionDOMNodeFromPatchRecord(RenderNodeRecord{NodeID: 1, Kind: RenderNodeKindText, TextRef: 1}, parseStringTable)
	if parseTextErr != nil {
		parseTesting.Fatalf("parseBuildRegionDOMNodeFromPatchRecord(text) returned error: %v", parseTextErr)
	}
	if parseTextNode.GetText != "hello" {
		parseTesting.Fatalf("expected text node payload to be resolved, got %q", parseTextNode.GetText)
	}
	parseHostNode, parseHostErr := parseBuildRegionDOMNodeFromPatchRecord(RenderNodeRecord{NodeID: 2, Kind: RenderNodeKindHostElement, TextRef: 0}, parseStringTable)
	if parseHostErr != nil {
		parseTesting.Fatalf("parseBuildRegionDOMNodeFromPatchRecord(host) returned error: %v", parseHostErr)
	}
	if parseHostNode.GetTag != "div" {
		parseTesting.Fatalf("expected host tag to be resolved, got %q", parseHostNode.GetTag)
	}
	parseFragmentNode, parseFragmentErr := parseBuildRegionDOMNodeFromPatchRecord(RenderNodeRecord{NodeID: 3, Kind: RenderNodeKindFragment}, parseStringTable)
	if parseFragmentErr != nil {
		parseTesting.Fatalf("parseBuildRegionDOMNodeFromPatchRecord(fragment) returned error: %v", parseFragmentErr)
	}
	if parseFragmentNode.GetTag != "" {
		parseTesting.Fatalf("expected fragment tag to be empty, got %q", parseFragmentNode.GetTag)
	}
	if _, parseErr := parseBuildRegionDOMNodeFromPatchRecord(RenderNodeRecord{Kind: RenderNodeKind(255)}, parseStringTable); parseErr == nil {
		parseTesting.Fatal("expected unsupported render-node kind to fail")
	}

	parseRegionDOMIndex := BuildRegionDOMIndex()
	if got := BuildKnownNodeIDsForRegionDOMIndex(nil, "region-a"); len(got) != 0 {
		parseTesting.Fatalf("expected nil region DOM index known-node set to be empty, got %d entries", len(got))
	}
	if got := BuildKnownNodeIDsForRegionDOMIndex(parseRegionDOMIndex, " "); len(got) != 0 {
		parseTesting.Fatalf("expected blank region ID known-node set to be empty, got %d entries", len(got))
	}
	if gotKnown, gotSibling := BuildRegionDOMPatchLookupMaps(nil, "region-a"); len(gotKnown) != 0 || len(gotSibling) != 0 {
		parseTesting.Fatalf("expected nil region DOM index lookup maps to be empty, got known=%+v sibling=%+v", gotKnown, gotSibling)
	}
	if got := BuildSiblingCountByParentForRegionDOMIndex(nil, "region-a"); len(got) != 0 {
		parseTesting.Fatalf("expected nil sibling-count lookup to be empty, got %d entries", len(got))
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-a", 1, &RegionDOMNode{GetNodeID: 1, GetTag: "div", GetChildNodeIDs: []uint64{2, 3}}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) returned error: %v", parseErr)
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-a", 2, &RegionDOMNode{GetNodeID: 2, GetTag: "span", GetParentNodeID: 1}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child-a) returned error: %v", parseErr)
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-a", 3, &RegionDOMNode{GetNodeID: 3, GetTag: "span", GetParentNodeID: 1}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child-b) returned error: %v", parseErr)
	}
	parseKnownRegionIDs := BuildKnownNodeIDsForRegionDOMIndex(parseRegionDOMIndex, "region-a")
	if len(parseKnownRegionIDs) != 3 {
		parseTesting.Fatalf("expected three known node IDs, got %d", len(parseKnownRegionIDs))
	}
	parseKnownRegionIDsByMap, parseSiblingCountsByMap := BuildRegionDOMPatchLookupMaps(parseRegionDOMIndex, "region-a")
	if len(parseKnownRegionIDsByMap) != 3 {
		parseTesting.Fatalf("expected three known node IDs from patch lookup maps, got %d", len(parseKnownRegionIDsByMap))
	}
	if parseSiblingCountsByMap[1] != 2 {
		parseTesting.Fatalf("expected sibling count 2 for parent 1, got %d", parseSiblingCountsByMap[1])
	}
	parseSiblingCountsByParent := BuildSiblingCountByParentForRegionDOMIndex(parseRegionDOMIndex, "region-a")
	if parseSiblingCountsByParent[1] != 2 {
		parseTesting.Fatalf("expected sibling count 2 for parent 1, got %d", parseSiblingCountsByParent[1])
	}
}

// TestParsePatchStreamTransactionAppendOnlySupportsNestedInsertedParents verifies append-only patch streams validate child inserts against parents introduced earlier in the same patch without cloning the base known-node set.
func TestParsePatchStreamTransactionAppendOnlySupportsNestedInsertedParents(parseTesting *testing.T) {
	parseStringTable := BuildRenderStringTable([]string{"div", "span"})
	parsePatchStream := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-a",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    2,
		},
		GetStringTable: parseStringTable.Entries,
		GetOps: []PatchStreamOpRaw{
			{
				GetOpCode: uint8(PatchOpCodeInsertNode),
				GetInsertOp: &PatchInsertOpRaw{
					ParentNodeID: 1,
					Node: RenderNodeRecordRaw{
						NodeID:  2,
						Kind:    uint8(RenderNodeKindHostElement),
						TextRef: 0,
					},
				},
			},
			{
				GetOpCode: uint8(PatchOpCodeInsertNode),
				GetInsertOp: &PatchInsertOpRaw{
					ParentNodeID: 2,
					Node: RenderNodeRecordRaw{
						NodeID:  3,
						Kind:    uint8(RenderNodeKindHostElement),
						TextRef: 1,
					},
				},
			},
		},
		GetPatchIdentity: "append-only-nested",
	}
	parseParseResult, hasPatchApply, parseParseErr := ParsePatchStreamTransaction(
		parsePatchStream,
		"region-a",
		1,
		map[uint64]struct{}{1: {}},
		nil,
		nil,
	)
	if parseParseErr != nil {
		parseTesting.Fatalf("ParsePatchStreamTransaction(append-only nested) error = %v", parseParseErr)
	}
	if !hasPatchApply {
		parseTesting.Fatal("ParsePatchStreamTransaction(append-only nested) expected apply=true")
	}
	if len(parseParseResult.GetTransaction.GetOps) != 2 {
		parseTesting.Fatalf("ParsePatchStreamTransaction(append-only nested) op count = %d, want 2", len(parseParseResult.GetTransaction.GetOps))
	}
	if parseParseResult.GetTransaction.GetOps[0].GetParentNodeID != 1 || parseParseResult.GetTransaction.GetOps[1].GetParentNodeID != 2 {
		parseTesting.Fatalf("ParsePatchStreamTransaction(append-only nested) parent chain = [%d %d], want [1 2]", parseParseResult.GetTransaction.GetOps[0].GetParentNodeID, parseParseResult.GetTransaction.GetOps[1].GetParentNodeID)
	}
}

// TestParsePatchStreamTransactionAppendOnlyRejectsDuplicateInsertedNodeIDs verifies append-only patch validation rejects duplicate node IDs introduced in the same patch.
func TestParsePatchStreamTransactionAppendOnlyRejectsDuplicateInsertedNodeIDs(parseTesting *testing.T) {
	parseStringTable := BuildRenderStringTable([]string{"div"})
	parsePatchStream := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-a",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    2,
		},
		GetStringTable: parseStringTable.Entries,
		GetOps: []PatchStreamOpRaw{
			{
				GetOpCode: uint8(PatchOpCodeInsertNode),
				GetInsertOp: &PatchInsertOpRaw{
					ParentNodeID: 1,
					Node: RenderNodeRecordRaw{
						NodeID:  2,
						Kind:    uint8(RenderNodeKindHostElement),
						TextRef: 0,
					},
				},
			},
			{
				GetOpCode: uint8(PatchOpCodeInsertNode),
				GetInsertOp: &PatchInsertOpRaw{
					ParentNodeID: 1,
					Node: RenderNodeRecordRaw{
						NodeID:  2,
						Kind:    uint8(RenderNodeKindHostElement),
						TextRef: 0,
					},
				},
			},
		},
		GetPatchIdentity: "append-only-duplicate",
	}
	_, _, parseParseErr := ParsePatchStreamTransaction(
		parsePatchStream,
		"region-a",
		1,
		map[uint64]struct{}{1: {}},
		nil,
		nil,
	)
	if parseParseErr == nil || !strings.Contains(parseParseErr.Error(), "already exists") {
		parseTesting.Fatalf("ParsePatchStreamTransaction(append-only duplicate) error = %v, want duplicate-node error", parseParseErr)
	}
}

// TestParsePatchStreamTransactionCoversCanonicalizationAndValidPaths covers parser canonicalization plus valid op kinds.
func TestParsePatchStreamTransactionCoversCanonicalizationAndValidPaths(parseTesting *testing.T) {
	parseCanonicalizeRaw := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-a",
			Epoch:           1,
			InputVersion:    2,
			PatchVersion:    3,
		},
		GetStringTable:   []string{"b", "a"},
		GetPatchIdentity: "canonicalize",
	}
	if _, _, parseErr := ParsePatchStreamTransaction(parseCanonicalizeRaw, "region-a", 1, nil, nil, nil); parseErr == nil || !strings.Contains(parseErr.Error(), "string table is not canonical") {
		parseTesting.Fatalf("expected canonicalization guard error, got %v", parseErr)
	}

	parseTextAttrResult, parseTextAttrHasApply, parseTextAttrErr := ParsePatchStreamTransaction(
		parseBuildPatchStreamForTest(
			parseTesting,
			"region-a",
			1,
			2,
			3,
			map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"props": map[string]any{
					"class":    "active",
					"data-old": "legacy",
					"style":    map[string]any{"color": "red"},
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
					"style": map[string]any{"color": "blue"},
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "after"},
				},
			},
		),
		"region-a",
		1,
		func() map[uint64]struct{} {
			parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"props": map[string]any{
					"class":    "active",
					"data-old": "legacy",
					"style":    map[string]any{"color": "red"},
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "before"},
				},
			})
			parsePreviousTree, parseTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
			if parseTreeErr != nil {
				parseTesting.Fatalf("ParseCanonicalRenderTree(previous text/attr) returned error: %v", parseTreeErr)
			}
			parseKnownNodeIDs := make(map[uint64]struct{}, len(parsePreviousTree.getNodeByID))
			for parseNodeID := range parsePreviousTree.getNodeByID {
				parseKnownNodeIDs[parseNodeID] = struct{}{}
			}
			return parseKnownNodeIDs
		}(),
		func() map[uint64]uint32 {
			parsePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
				"kind": "host-element",
				"tag":  "div",
				"props": map[string]any{
					"class":    "active",
					"data-old": "legacy",
					"style":    map[string]any{"color": "red"},
				},
				"children": []any{
					map[string]any{"kind": "text", "text": "before"},
				},
			})
			parsePreviousTree, parseTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
			if parseTreeErr != nil {
				parseTesting.Fatalf("ParseCanonicalRenderTree(previous text/attr sibling) returned error: %v", parseTreeErr)
			}
			parseSiblingCounts := make(map[uint64]uint32, len(parsePreviousTree.getNodeByID))
			for parseNodeID, parseNodeState := range parsePreviousTree.getNodeByID {
				if len(parseNodeState.getChildNodeIDs) == 0 {
					continue
				}
				parseSiblingCounts[parseNodeID] = uint32(len(parseNodeState.getChildNodeIDs))
			}
			return parseSiblingCounts
		}(),
		nil,
	)
	if parseTextAttrErr != nil {
		parseTesting.Fatalf("ParsePatchStreamTransaction(text/attr/style) returned error: %v", parseTextAttrErr)
	}
	if !parseTextAttrHasApply || len(parseTextAttrResult.GetTransaction.GetOps) == 0 {
		parseTesting.Fatal("expected valid text/attr/style patch to apply and produce ops")
	}

	parseInsertRemoveRaw := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		3,
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
	parseInsertPreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"children": []any{
			map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
		},
	})
	parseInsertPreviousTree, parseInsertTreeErr := ParseCanonicalRenderTree(parseInsertPreviousIR)
	if parseInsertTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(previous insert/remove) returned error: %v", parseInsertTreeErr)
	}
	parseInsertKnownNodeIDs := make(map[uint64]struct{}, len(parseInsertPreviousTree.getNodeByID))
	parseInsertSiblingCounts := make(map[uint64]uint32, len(parseInsertPreviousTree.getNodeByID))
	for parseNodeID, parseNodeState := range parseInsertPreviousTree.getNodeByID {
		parseInsertKnownNodeIDs[parseNodeID] = struct{}{}
		if len(parseNodeState.getChildNodeIDs) == 0 {
			continue
		}
		parseInsertSiblingCounts[parseNodeID] = uint32(len(parseNodeState.getChildNodeIDs))
	}
	parseInsertResult, parseInsertHasApply, parseInsertErr := ParsePatchStreamTransaction(parseInsertRemoveRaw, "region-a", 1, parseInsertKnownNodeIDs, parseInsertSiblingCounts, nil)
	if parseInsertErr != nil {
		parseTesting.Fatalf("ParsePatchStreamTransaction(insert/remove) returned error: %v", parseInsertErr)
	}
	if !parseInsertHasApply || len(parseInsertResult.GetTransaction.GetOps) == 0 {
		parseTesting.Fatal("expected insert/remove patch to apply and produce ops")
	}

	parseMoveRaw := parseBuildPatchStreamForTest(
		parseTesting,
		"region-a",
		1,
		2,
		3,
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
	parseMovePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"children": []any{
			map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "c"},
		},
	})
	parseMovePreviousTree, parseMoveTreeErr := ParseCanonicalRenderTree(parseMovePreviousIR)
	if parseMoveTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(previous move) returned error: %v", parseMoveTreeErr)
	}
	parseMoveKnownNodeIDs := make(map[uint64]struct{}, len(parseMovePreviousTree.getNodeByID))
	parseMoveSiblingCounts := make(map[uint64]uint32, len(parseMovePreviousTree.getNodeByID))
	for parseNodeID, parseNodeState := range parseMovePreviousTree.getNodeByID {
		parseMoveKnownNodeIDs[parseNodeID] = struct{}{}
		if len(parseNodeState.getChildNodeIDs) == 0 {
			continue
		}
		parseMoveSiblingCounts[parseNodeID] = uint32(len(parseNodeState.getChildNodeIDs))
	}
	parseMoveResult, parseMoveHasApply, parseMoveErr := ParsePatchStreamTransaction(parseMoveRaw, "region-a", 1, parseMoveKnownNodeIDs, parseMoveSiblingCounts, nil)
	if parseMoveErr != nil {
		parseTesting.Fatalf("ParsePatchStreamTransaction(move) returned error: %v", parseMoveErr)
	}
	if !parseMoveHasApply || len(parseMoveResult.GetTransaction.GetOps) == 0 {
		parseTesting.Fatal("expected move patch to apply and produce ops")
	}

	parseReplacePreviousIR := parseBuildCanonicalIRForTest(parseTesting, map[string]any{"kind": "text", "text": "before"})
	parseReplaceNextIR := parseBuildCanonicalIRWithShiftedRootIDForTest(parseTesting, parseBuildCanonicalIRForTest(parseTesting, map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{"kind": "text", "text": "after"},
		},
	}))
	parseReplaceRaw, parseReplaceHasNoOp, parseReplaceErr := BuildCanonicalPatchStream("region-a", 1, 2, 3, parseReplacePreviousIR, parseReplaceNextIR)
	if parseReplaceErr != nil {
		parseTesting.Fatalf("BuildCanonicalPatchStream(replace) returned error: %v", parseReplaceErr)
	}
	if parseReplaceHasNoOp {
		parseTesting.Fatal("expected replace patch to produce ops")
	}
	parseReplaceTree, parseReplaceTreeErr := ParseCanonicalRenderTree(parseReplacePreviousIR)
	if parseReplaceTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(previous replace) returned error: %v", parseReplaceTreeErr)
	}
	parseReplaceKnownNodeIDs := make(map[uint64]struct{}, len(parseReplaceTree.getNodeByID))
	for parseNodeID := range parseReplaceTree.getNodeByID {
		parseReplaceKnownNodeIDs[parseNodeID] = struct{}{}
	}
	parseReplaceResult, parseReplaceHasApply, parseReplaceParseErr := ParsePatchStreamTransaction(parseReplaceRaw, "region-a", 1, parseReplaceKnownNodeIDs, nil, nil)
	if parseReplaceParseErr != nil {
		parseTesting.Fatalf("ParsePatchStreamTransaction(replace-subtree) returned error: %v", parseReplaceParseErr)
	}
	if !parseReplaceHasApply || len(parseReplaceResult.GetTransaction.GetOps) == 0 {
		parseTesting.Fatal("expected replace-subtree patch to apply and produce ops")
	}
}

// TestParsePatchStreamTransactionRejectsMissingPayloadBranches covers the remaining parser guard branches for missing payloads and invalid op codes.
func TestParsePatchStreamTransactionRejectsMissingPayloadBranches(parseTesting *testing.T) {
	testCases := []struct {
		name        string
		parseOp     PatchStreamOpRaw
		wantMessage string
	}{
		{
			name:        "invalid op code",
			parseOp:     PatchStreamOpRaw{GetOpCode: 255},
			wantMessage: "invalid op code",
		},
		{
			name:        "missing insert payload",
			parseOp:     PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeInsertNode)},
			wantMessage: "insert payload is required",
		},
		{
			name:        "missing remove payload",
			parseOp:     PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeRemoveNode)},
			wantMessage: "remove payload is required",
		},
		{
			name:        "missing set-text payload",
			parseOp:     PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeSetText)},
			wantMessage: "set-text payload is required",
		},
		{
			name:        "missing set-attr payload",
			parseOp:     PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeSetAttr)},
			wantMessage: "set-attr payload is required",
		},
		{
			name:        "missing remove-attr payload",
			parseOp:     PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeRemoveAttr)},
			wantMessage: "remove-attr payload is required",
		},
		{
			name:        "missing keyed-move payload",
			parseOp:     PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeMoveKeyedChild)},
			wantMessage: "keyed-move payload is required",
		},
		{
			name:        "missing replace-subtree payload",
			parseOp:     PatchStreamOpRaw{GetOpCode: uint8(PatchOpCodeReplaceSubtree)},
			wantMessage: "replace-subtree payload is required",
		},
	}
	for _, parseTestCase := range testCases {
		parseTesting.Run(parseTestCase.name, func(parseSubTest *testing.T) {
			parsePatchStream := parseBuildInvalidPatchStream(parseTestCase.parseOp)
			_, _, parseErr := ParsePatchStreamTransaction(
				parsePatchStream,
				"region-a",
				1,
				map[uint64]struct{}{1: {}},
				map[uint64]uint32{1: 1},
				nil,
			)
			if parseErr == nil {
				parseSubTest.Fatal("expected patch parse to fail")
			}
			if !strings.Contains(parseErr.Error(), parseTestCase.wantMessage) {
				parseSubTest.Fatalf("expected %q in error, got %v", parseTestCase.wantMessage, parseErr)
			}
		})
	}
}
