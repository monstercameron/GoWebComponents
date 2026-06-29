package runtime2

import (
	"strings"
	"testing"
)

// TestCommitRegionPatchTransactionFullyValidCommitsAllOps verifies valid streams commit every operation.
func TestCommitRegionPatchTransactionFullyValidCommitsAllOps(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:    1,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 1,
				GetText:   "after",
			},
			{
				GetKind:      RegionPatchOpKindSetAttr,
				GetNodeID:    1,
				GetAttrKey:   "class",
				GetAttrValue: "card",
			},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(valid) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(valid) expected no fallback")
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(host) error = %v", parseLookupErr)
	}
	if parseNode.GetText != "after" {
		parseTesting.Fatalf("GetRegionDOMNode(host) text = %q, want %q", parseNode.GetText, "after")
	}
	if parseNode.GetAttrByKey["class"] != "card" {
		parseTesting.Fatalf("GetRegionDOMNode(host) class = %q, want %q", parseNode.GetAttrByKey["class"], "card")
	}
}

// TestCommitRegionPatchTransactionBumpsMutationVersionOnce verifies successful multi-op transactions invalidate region lookup state with one monotonic version bump instead of one bump per op.
func TestCommitRegionPatchTransactionBumpsMutationVersionOnce(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:    1,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseInitialMutationVersion := parseRegionDOMIndex.GetRegionDOMMutationVersion("region-42")
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 1,
				GetText:   "after",
			},
			{
				GetKind:      RegionPatchOpKindSetAttr,
				GetNodeID:    1,
				GetAttrKey:   "class",
				GetAttrValue: "card",
			},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(valid version batch) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(valid version batch) expected no fallback")
	}
	parseFinalMutationVersion := parseRegionDOMIndex.GetRegionDOMMutationVersion("region-42")
	if parseFinalMutationVersion != parseInitialMutationVersion+1 {
		parseTesting.Fatalf("GetRegionDOMMutationVersion(region-42) = %d, want %d after one successful multi-op transaction", parseFinalMutationVersion, parseInitialMutationVersion+1)
	}
}

// TestCommitRegionPatchTransactionInvalidMidStreamTriggersFallback verifies invalid mid-stream operations trigger fallback.
func TestCommitRegionPatchTransactionInvalidMidStreamTriggersFallback(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:    1,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 1,
				GetText:   "after",
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 999,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(invalid mid-stream) error = nil, want error")
	}
	if !parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(invalid mid-stream) expected fallback result")
	}
}

// TestCommitRegionPatchTransactionRollsBackPartialState verifies failed streams do not leave partial commit effects active.
func TestCommitRegionPatchTransactionRollsBackPartialState(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:    1,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 1,
				GetText:   "after",
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 999,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(invalid mid-stream) error = nil, want error")
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(host) error = %v", parseLookupErr)
	}
	if parseNode.GetText != "before" {
		parseTesting.Fatalf("GetRegionDOMNode(host) text after rollback = %q, want %q", parseNode.GetText, "before")
	}
}

// TestCommitRegionPatchTransactionInsertRollbackRemovesInsertedNode verifies insert-only structural rollback restores the parent order without cloning the full region.
func TestCommitRegionPatchTransactionInsertRollbackRemovesInsertedNode(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode: &RegionDOMNode{
					GetNodeID: 3,
					GetTag:    "li",
				},
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 99,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(insert rollback) error = nil, want error")
	}
	if !parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(insert rollback) expected fallback result")
	}
	parseParentNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseLookupErr)
	}
	parseWant := []uint64{2}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) || parseParentNode.GetChildNodeIDs[0] != parseWant[0] {
		parseTesting.Fatalf("GetRegionDOMNode(parent) children after rollback = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
	if _, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 3); parseLookupErr == nil {
		parseTesting.Fatal("GetRegionDOMNode(inserted child after rollback) error = nil, want missing node")
	}
}

// TestCommitRegionPatchTransactionInsertRollbackRestoresOriginalParentOrderAfterMultipleInserts verifies repeated structural snapshots keep the original pre-transaction sibling order instead of the latest partial state.
func TestCommitRegionPatchTransactionInsertRollbackRestoresOriginalParentOrderAfterMultipleInserts(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode: &RegionDOMNode{
					GetNodeID: 3,
					GetTag:    "li",
				},
			},
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode: &RegionDOMNode{
					GetNodeID: 4,
					GetTag:    "li",
				},
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 99,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(multi-insert rollback) error = nil, want error")
	}
	if !parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(multi-insert rollback) expected fallback result")
	}
	parseParentNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseLookupErr)
	}
	parseWant := []uint64{2}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) || parseParentNode.GetChildNodeIDs[0] != parseWant[0] {
		parseTesting.Fatalf("GetRegionDOMNode(parent) children after multi-insert rollback = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
	for _, getNodeID := range []uint64{3, 4} {
		if _, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", getNodeID); parseLookupErr == nil {
			parseTesting.Fatalf("GetRegionDOMNode(inserted child=%d after rollback) error = nil, want missing node", getNodeID)
		}
	}
}

// TestCommitRegionPatchTransactionMoveRollbackRestoresSiblingOrder verifies move-only structural rollback restores keyed sibling order without cloning the full region.
func TestCommitRegionPatchTransactionMoveRollbackRestoresSiblingOrder(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3, 4},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	for _, getNodeID := range []uint64{2, 3, 4} {
		if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row",
		}); parseInsertErr != nil {
			parseTesting.Fatalf("SetRegionDOMNode(child=%d) error = %v", getNodeID, parseInsertErr)
		}
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:             RegionPatchOpKindMoveKeyedNode,
				GetParentNodeID:     1,
				GetMoveNodeID:       4,
				GetDestinationIndex: 0,
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 99,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(move rollback) error = nil, want error")
	}
	if !parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(move rollback) expected fallback result")
	}
	parseParentNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseLookupErr)
	}
	parseWant := []uint64{2, 3, 4}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) || parseParentNode.GetChildNodeIDs[0] != parseWant[0] || parseParentNode.GetChildNodeIDs[1] != parseWant[1] || parseParentNode.GetChildNodeIDs[2] != parseWant[2] {
		parseTesting.Fatalf("GetRegionDOMNode(parent) children after rollback = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
}

// TestCommitRegionPatchTransactionRemoveRollbackRestoresSubtree verifies remove-only structural rollback restores removed subtrees without cloning the full region.
func TestCommitRegionPatchTransactionRemoveRollbackRestoresSubtree(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 4},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetChildNodeIDs: []uint64{3},
		GetNodeKey:      "row-a",
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent child) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "span",
		GetParentNodeID: 2,
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(descendant) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 4, &RegionDOMNode{
		GetNodeID:       4,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(sibling) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindRemoveNode,
				GetNodeID: 2,
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 99,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(remove rollback) error = nil, want error")
	}
	if !parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(remove rollback) expected fallback result")
	}
	parseRootNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(root) error = %v", parseLookupErr)
	}
	parseWant := []uint64{2, 4}
	if len(parseRootNode.GetChildNodeIDs) != len(parseWant) || parseRootNode.GetChildNodeIDs[0] != parseWant[0] || parseRootNode.GetChildNodeIDs[1] != parseWant[1] {
		parseTesting.Fatalf("GetRegionDOMNode(root) children after remove rollback = %v, want %v", parseRootNode.GetChildNodeIDs, parseWant)
	}
	for _, getRestoredNodeID := range []uint64{2, 3} {
		if _, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", getRestoredNodeID); parseLookupErr != nil {
			parseTesting.Fatalf("GetRegionDOMNode(restored node=%d) error = %v", getRestoredNodeID, parseLookupErr)
		}
	}
}

// TestCommitRegionPatchTransactionRemoveOnlyCommitsWithoutFallback verifies remove-only transactions use the no-snapshot fast path and still commit cleanly.
func TestCommitRegionPatchTransactionRemoveOnlyCommitsWithoutFallback(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3, 4},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) error = %v", parseInsertErr)
	}
	for _, getNodeID := range []uint64{2, 3, 4} {
		if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row",
		}); parseInsertErr != nil {
			parseTesting.Fatalf("SetRegionDOMNode(child=%d) error = %v", getNodeID, parseInsertErr)
		}
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 2},
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 4},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(remove only) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(remove only) expected no fallback")
	}
	parseRootNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(root) error = %v", parseLookupErr)
	}
	parseWant := []uint64{3}
	if len(parseRootNode.GetChildNodeIDs) != len(parseWant) || parseRootNode.GetChildNodeIDs[0] != parseWant[0] {
		parseTesting.Fatalf("GetRegionDOMNode(root) children after remove-only commit = %v, want %v", parseRootNode.GetChildNodeIDs, parseWant)
	}
}

// TestCommitRegionPatchTransactionRemoveOnlyCollapsesNestedTargets verifies remove-only transactions stay stable when both a subtree root and one descendant appear in the same patch.
func TestCommitRegionPatchTransactionRemoveOnlyCollapsesNestedTargets(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 4},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetChildNodeIDs: []uint64{3},
		GetNodeKey:      "row-a",
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent child) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "span",
		GetParentNodeID: 2,
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(descendant) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 4, &RegionDOMNode{
		GetNodeID:       4,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(sibling) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 3},
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 2},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(remove nested) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(remove nested) expected no fallback")
	}
	parseRootNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(root) error = %v", parseLookupErr)
	}
	parseWant := []uint64{4}
	if len(parseRootNode.GetChildNodeIDs) != len(parseWant) || parseRootNode.GetChildNodeIDs[0] != parseWant[0] {
		parseTesting.Fatalf("GetRegionDOMNode(root) children after nested remove-only commit = %v, want %v", parseRootNode.GetChildNodeIDs, parseWant)
	}
	if _, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 2); parseLookupErr == nil {
		parseTesting.Fatal("GetRegionDOMNode(removed parent child) error = nil, want missing node")
	}
	if _, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 3); parseLookupErr == nil {
		parseTesting.Fatal("GetRegionDOMNode(removed descendant) error = nil, want missing node")
	}
}

// TestCommitRegionPatchTransactionAppendOnlyCommitsWithoutFallback verifies append-only insert transactions use the validated fast path and keep one mutation-version bump for the whole batch.
func TestCommitRegionPatchTransactionAppendOnlyCommitsWithoutFallback(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(existing child) error = %v", parseInsertErr)
	}
	parseInitialMutationVersion := parseRegionDOMIndex.GetRegionDOMMutationVersion("region-42")
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode: &RegionDOMNode{
					GetNodeID:  3,
					GetTag:     "li",
					GetNodeKey: "row-b",
				},
			},
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode: &RegionDOMNode{
					GetNodeID:  4,
					GetTag:     "li",
					GetNodeKey: "row-c",
				},
			},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(append only) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(append only) expected no fallback")
	}
	parseParentNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseLookupErr)
	}
	parseWant := []uint64{2, 3, 4}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) ||
		parseParentNode.GetChildNodeIDs[0] != parseWant[0] ||
		parseParentNode.GetChildNodeIDs[1] != parseWant[1] ||
		parseParentNode.GetChildNodeIDs[2] != parseWant[2] {
		parseTesting.Fatalf("GetRegionDOMNode(parent) children after append-only commit = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
	if parseRegionDOMIndex.GetRegionDOMMutationVersion("region-42") != parseInitialMutationVersion+1 {
		parseTesting.Fatalf("GetRegionDOMMutationVersion(region-42) = %d, want %d after append-only commit", parseRegionDOMIndex.GetRegionDOMMutationVersion("region-42"), parseInitialMutationVersion+1)
	}
}

// TestCommitRegionPatchTransactionAppendOnlySupportsNestedInsertedParents verifies append-only insert transactions can attach descendants to parents inserted earlier in the same batch.
func TestCommitRegionPatchTransactionAppendOnlySupportsNestedInsertedParents(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "section",
		GetChildNodeIDs: []uint64{},
	}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode: &RegionDOMNode{
					GetNodeID: 2,
					GetTag:    "div",
				},
			},
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 2,
				GetInsertNode: &RegionDOMNode{
					GetNodeID: 3,
					GetTag:    "span",
				},
			},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(nested append only) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(nested append only) expected no fallback")
	}
	parseRootNode, parseRootLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseRootLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(root) error = %v", parseRootLookupErr)
	}
	if len(parseRootNode.GetChildNodeIDs) != 1 || parseRootNode.GetChildNodeIDs[0] != 2 {
		parseTesting.Fatalf("GetRegionDOMNode(root) children after nested append-only commit = %v, want [2]", parseRootNode.GetChildNodeIDs)
	}
	parseInsertedParentNode, parseParentLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 2)
	if parseParentLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(inserted parent) error = %v", parseParentLookupErr)
	}
	if len(parseInsertedParentNode.GetChildNodeIDs) != 1 || parseInsertedParentNode.GetChildNodeIDs[0] != 3 {
		parseTesting.Fatalf("GetRegionDOMNode(inserted parent) children after nested append-only commit = %v, want [3]", parseInsertedParentNode.GetChildNodeIDs)
	}
}

// TestCommitRegionPatchTransactionRejectsExcessiveOpCount verifies hard guard limits fail fast before deep snapshot cloning work.
func TestCommitRegionPatchTransactionRejectsExcessiveOpCount(parseTesting *testing.T) {
	parseDOMCommitter := BuildDOMCommitter(BuildRegionDOMIndex())
	parseOps := make([]RegionPatchOp, getDOMCommitTransactionOpHardLimit+1)
	_, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps:      parseOps,
	})
	if parseTransactionErr == nil || !strings.Contains(parseTransactionErr.Error(), "exceeds guard limit") {
		parseTesting.Fatalf("CommitRegionPatchTransaction(guard) error = %v, want guard-limit error", parseTransactionErr)
	}
}
