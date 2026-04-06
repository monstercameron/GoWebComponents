package runtime2

import (
	"strings"
	"testing"
)

// TestDOMCommitterGuardsAndHelperBranches covers nil guards, attribute removal no-ops, and direct helper branches.
func TestDOMCommitterGuardsAndHelperBranches(parseTesting *testing.T) {
	var parseNilCommitter *DOMCommitter
	if _, parseErr := parseNilCommitter.CommitRegionText("region-42", 1, "after"); parseErr == nil {
		parseTesting.Fatal("expected nil committer text commit to fail")
	}
	if _, parseErr := parseNilCommitter.CommitRegionAttr("region-42", 1, "class", "card"); parseErr == nil {
		parseTesting.Fatal("expected nil committer attr commit to fail")
	}
	if _, parseErr := parseNilCommitter.CommitRegionRemoveAttr("region-42", 1, "class"); parseErr == nil {
		parseTesting.Fatal("expected nil committer remove-attr commit to fail")
	}
	if _, parseErr := parseNilCommitter.CommitRegionInsertNode("region-42", 1, &RegionDOMNode{GetNodeID: 2, GetTag: "span"}, 0); parseErr == nil {
		parseTesting.Fatal("expected nil committer insert-node commit to fail")
	}
	if _, parseErr := parseNilCommitter.CommitRegionRemoveNode("region-42", 1); parseErr == nil {
		parseTesting.Fatal("expected nil committer remove-node commit to fail")
	}
	if _, parseErr := parseNilCommitter.CommitRegionReplaceSubtree("region-42", 1, CanonicalRenderIR{}); parseErr == nil {
		parseTesting.Fatal("expected nil committer replace-subtree commit to fail")
	}
	if _, parseErr := parseNilCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{GetRegionID: "region-42"}); parseErr == nil {
		parseTesting.Fatal("expected nil committer patch transaction to fail")
	}

	parseBuildCommitter := BuildDOMCommitter(nil)
	if parseBuildCommitter == nil || parseBuildCommitter.getRegionDOMIndex == nil {
		parseTesting.Fatal("expected BuildDOMCommitter(nil) to create a usable committer")
	}

	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseResult, parseErr := parseBuildCommitter.CommitRegionRemoveAttr("region-42", 1, "class"); parseErr == nil && !parseResult.IsNoOp {
		parseTesting.Fatal("expected remove-attr on missing region to be a no-op")
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "div",
		GetAttrByKey:    map[string]string{"class": "card", "style": "color: red"},
		GetChildNodeIDs: []uint64{2, 3},
	}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) error = %v", parseErr)
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "span",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child-a) error = %v", parseErr)
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "span",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child-b) error = %v", parseErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)

	if _, parseErr := parseDOMCommitter.CommitRegionRemoveAttr("region-42", 1, "missing"); parseErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveAttr(missing) error = %v", parseErr)
	}
	if _, parseErr := parseDOMCommitter.CommitRegionRemoveAttr("region-42", 1, "class"); parseErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveAttr(existing) error = %v", parseErr)
	}
	if _, parseErr := parseDOMCommitter.CommitRegionRemoveStyle("region-42", 1); parseErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveStyle(existing) error = %v", parseErr)
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(root) error = %v", parseLookupErr)
	}
	if len(parseNode.GetAttrByKey) != 0 {
		parseTesting.Fatalf("expected root attrs to be cleared, got %+v", parseNode.GetAttrByKey)
	}

	if _, parseErr := parseDOMCommitter.CommitRegionInsertNode("region-42", 1, &RegionDOMNode{GetNodeID: 4, GetTag: "span"}, 99); parseErr == nil || !strings.Contains(parseErr.Error(), "invalid sibling anchor") {
		parseTesting.Fatalf("CommitRegionInsertNode(invalid anchor) error = %v", parseErr)
	}
	if _, parseErr := parseDOMCommitter.CommitRegionInsertNode("region-42", 1, &RegionDOMNode{GetNodeID: 4, GetTag: "span"}, 0); parseErr != nil {
		parseTesting.Fatalf("CommitRegionInsertNode(valid) error = %v", parseErr)
	}
	if _, parseErr := parseDOMCommitter.CommitRegionInsertNode("region-42", 1, &RegionDOMNode{GetNodeID: 4, GetTag: "span"}, 0); parseErr == nil || !strings.Contains(parseErr.Error(), "already exists") {
		parseTesting.Fatalf("CommitRegionInsertNode(duplicate) error = %v", parseErr)
	}
	if _, parseErr := parseDOMCommitter.CommitRegionMoveKeyedNode("region-42", 1, 3, 2); parseErr != nil {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(valid) error = %v", parseErr)
	}
	if _, parseErr := parseDOMCommitter.CommitRegionMoveKeyedNode("region-42", 1, 99, 0); parseErr == nil || !strings.Contains(parseErr.Error(), "missing move node") {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(missing) error = %v", parseErr)
	}

	parseHelperParent := []uint64{2, 3}
	parseNextChildIDs, parseErr := parseInsertChildNodeIDs(parseHelperParent, 4, 0)
	if parseErr != nil {
		parseTesting.Fatalf("parseInsertChildNodeIDs(append) error = %v", parseErr)
	}
	if len(parseNextChildIDs) != 3 || parseNextChildIDs[2] != 4 {
		parseTesting.Fatalf("parseInsertChildNodeIDs(append) children = %v", parseNextChildIDs)
	}
	if _, parseErr := parseInsertChildNodeIDs(parseHelperParent, 4, 99); parseErr == nil {
		parseTesting.Fatal("expected invalid sibling anchor to fail")
	}
	if parseFindChildNodeIndex(parseHelperParent, 3) != 1 {
		parseTesting.Fatal("expected parseFindChildNodeIndex to find existing child")
	}
	if parseFindChildNodeIndex(parseHelperParent, 99) != -1 {
		parseTesting.Fatal("expected parseFindChildNodeIndex to miss absent child")
	}
	if !parseSliceEqualUint64([]uint64{1, 2}, []uint64{1, 2}) {
		parseTesting.Fatal("expected parseSliceEqualUint64 to report equal slices")
	}
	if parseSliceEqualUint64([]uint64{1, 2}, []uint64{2, 1}) {
		parseTesting.Fatal("expected parseSliceEqualUint64 to report unequal slices")
	}

	if parseClone := parseCloneRegionDOMNode(nil); parseClone != nil {
		parseTesting.Fatal("expected nil node clone to remain nil")
	}
	parseNodeClone := parseCloneRegionDOMNode(&RegionDOMNode{
		GetNodeID:       7,
		GetTag:          "div",
		GetText:         "before",
		GetAttrByKey:    map[string]string{"class": "card"},
		GetChildNodeIDs: []uint64{8, 9},
		GetParentNodeID: 1,
		GetNodeKey:      "root",
	})
	parseNodeClone.GetAttrByKey["class"] = "changed"
	parseNodeClone.GetChildNodeIDs[0] = 99
	if parseNodeClone.GetAttrByKey["class"] != "changed" {
		parseTesting.Fatal("expected cloned node to be independently mutable")
	}

	if parseCloneMap := parseCloneRegionNodeMap(nil); parseCloneMap != nil {
		parseTesting.Fatal("expected nil node map clone to remain nil")
	}
	parseCloneMap := parseCloneRegionNodeMap(map[uint64]*RegionDOMNode{
		1: nil,
		2: {
			GetNodeID:    2,
			GetTag:       "span",
			GetAttrByKey: map[string]string{"data-state": "ready"},
		},
	})
	if len(parseCloneMap) != 1 {
		parseTesting.Fatalf("expected nil node entries to be skipped, got %d entries", len(parseCloneMap))
	}
	parseCloneMap[2].GetAttrByKey["data-state"] = "changed"
	if parseCloneMap[2].GetAttrByKey["data-state"] != "changed" {
		parseTesting.Fatal("expected cloned node map entry to be independently mutable")
	}

	parseDOMCommitter.parseRestoreRegionSnapshot("region-42", nil)
	if _, hasRegion := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID["region-42"]; hasRegion {
		parseTesting.Fatal("expected nil region snapshot restore to delete region")
	}
	parseDOMCommitter.parseRestoreRegionSnapshot("region-42", map[uint64]*RegionDOMNode{
		1: &RegionDOMNode{GetNodeID: 1, GetTag: "div"},
	})
	if _, hasRegion := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID["region-42"]; !hasRegion {
		parseTesting.Fatal("expected non-nil region snapshot restore to store region")
	}
	parseRestoreCommitNodeSnapshot(nil, nil)
	parseRestoreCommitNodeSnapshot(map[uint64]*RegionDOMNode{}, map[uint64]*RegionDOMNode{
		1: nil,
		2: &RegionDOMNode{GetNodeID: 2, GetTag: "span"},
	})

	parseCaptureNodeSnapshot := parseCaptureCommitNodeSnapshot(nil, map[uint64]*RegionDOMNode{
		2: &RegionDOMNode{GetNodeID: 2, GetText: "before"},
	}, RegionPatchOp{GetKind: RegionPatchOpKindSetText, GetNodeID: 2})
	if len(parseCaptureNodeSnapshot) != 1 {
		parseTesting.Fatalf("expected snapshot capture for set-text op, got %d entries", len(parseCaptureNodeSnapshot))
	}
	parseCaptureNoChange := parseCaptureCommitNodeSnapshot(parseCaptureNodeSnapshot, map[uint64]*RegionDOMNode{
		2: &RegionDOMNode{GetNodeID: 2, GetText: "before"},
	}, RegionPatchOp{GetKind: RegionPatchOpKindInsertNode})
	if len(parseCaptureNoChange) != len(parseCaptureNodeSnapshot) {
		parseTesting.Fatal("expected structural op capture to leave snapshot unchanged")
	}
	parseStoreSnapshot := storeCommitNodeSnapshot(nil, map[uint64]*RegionDOMNode{
		3: &RegionDOMNode{GetNodeID: 3, GetText: "before"},
	}, 3)
	if len(parseStoreSnapshot) != 1 {
		parseTesting.Fatal("expected storeCommitNodeSnapshot to capture existing node")
	}
	if parseStoreCommitSnapshot := storeCommitNodeSnapshot(parseStoreSnapshot, map[uint64]*RegionDOMNode{
		3: &RegionDOMNode{GetNodeID: 3, GetText: "before"},
	}, 3); len(parseStoreCommitSnapshot) != 1 {
		parseTesting.Fatal("expected repeated snapshot storage to remain deduplicated")
	}
}

// TestDOMCommitReplaceSubtreeAndTransactionBranches covers replace-subtree commits and transaction error branches.
func TestDOMCommitReplaceSubtreeAndTransactionBranches(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseReplacementIR, parseReplacementIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "after",
	})
	if parseReplacementIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR returned error: %v", parseReplacementIRErr)
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", parseReplacementIR.GetRootNodeID, &RegionDOMNode{
		GetNodeID:    parseReplacementIR.GetRootNodeID,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{"class": "card"},
	}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) error = %v", parseErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	if _, parseErr := parseDOMCommitter.CommitRegionReplaceSubtree(" ", 1, parseReplacementIR); parseErr == nil {
		parseTesting.Fatal("expected blank region ID replace-subtree to fail")
	}
	if _, parseErr := parseDOMCommitter.CommitRegionReplaceSubtree("region-42", 0, parseReplacementIR); parseErr == nil {
		parseTesting.Fatal("expected zero target node ID replace-subtree to fail")
	}
	if _, parseErr := parseDOMCommitter.CommitRegionReplaceSubtree("region-42", 999, parseReplacementIR); parseErr == nil {
		parseTesting.Fatal("expected missing target node replace-subtree to fail")
	}
	if _, parseErr := parseDOMCommitter.CommitRegionReplaceSubtree("region-42", parseReplacementIR.GetRootNodeID, CanonicalRenderIR{}); parseErr == nil {
		parseTesting.Fatal("expected invalid replacement subtree to fail")
	}
	parseReplaceResult, parseReplaceErr := parseDOMCommitter.CommitRegionReplaceSubtree("region-42", parseReplacementIR.GetRootNodeID, parseReplacementIR)
	if parseReplaceErr != nil {
		parseTesting.Fatalf("CommitRegionReplaceSubtree(valid) error = %v", parseReplaceErr)
	}
	if parseReplaceResult.IsNoOp {
		parseTesting.Fatal("expected replace-subtree commit to change state")
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", parseReplacementIR.GetRootNodeID)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(replaced root) error = %v", parseLookupErr)
	}
	if parseNode.GetText != "after" {
		parseTesting.Fatalf("expected replaced text to be %q, got %q", "after", parseNode.GetText)
	}

	parseTransactionRegionDOMIndex := BuildRegionDOMIndex()
	if parseErr := parseTransactionRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "div",
		GetText:         "before",
		GetAttrByKey:    map[string]string{},
		GetChildNodeIDs: []uint64{2, 3},
	}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(transaction root) error = %v", parseErr)
	}
	if parseErr := parseTransactionRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "span",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(transaction child-a) error = %v", parseErr)
	}
	if parseErr := parseTransactionRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "span",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	}); parseErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(transaction child-b) error = %v", parseErr)
	}
	parseTransactionCommitter := BuildDOMCommitter(parseTransactionRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseTransactionCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindSetText, GetNodeID: 1, GetText: "txn-after"},
			{GetKind: RegionPatchOpKindSetAttr, GetNodeID: 1, GetAttrKey: "data-state", GetAttrValue: "ready"},
			{GetKind: RegionPatchOpKindSetStyle, GetNodeID: 1, GetStyleValue: "color: blue"},
			{GetKind: RegionPatchOpKindRemoveAttr, GetNodeID: 1, GetAttrKey: "data-state"},
			{GetKind: RegionPatchOpKindRemoveStyle, GetNodeID: 1},
			{GetKind: RegionPatchOpKindMoveKeyedNode, GetParentNodeID: 1, GetMoveNodeID: 3, GetDestinationIndex: 0},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(valid) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("expected valid patch transaction to avoid fallback")
	}

	parseNilTransactionResult, parseNilTransactionErr := parseTransactionCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{GetRegionID: "region-42"})
	if parseNilTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(empty) error = %v", parseNilTransactionErr)
	}
	if parseNilTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("expected empty patch transaction to be a no-op")
	}
	if _, parseErr := parseTransactionCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{GetRegionID: " "}); parseErr == nil {
		parseTesting.Fatal("expected blank transaction region to fail")
	}

	if parseErr := parseTransactionCommitter.parseCommitPatchOp("region-42", RegionPatchOp{GetKind: regionPatchOpKindInvalid}); parseErr == nil {
		parseTesting.Fatal("expected invalid patch op kind to fail")
	}
	if parseErr := parseTransactionCommitter.parseCommitPatchOp("region-42", RegionPatchOp{GetKind: RegionPatchOpKindReplaceSubtree, GetNodeID: 1, GetReplaceSubtreeIR: CanonicalRenderIR{}}); parseErr == nil {
		parseTesting.Fatal("expected invalid replace-subtree op to fail through dispatcher")
	}
}
