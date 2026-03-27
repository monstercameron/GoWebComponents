package runtime2

import "testing"

// TestCommitRegionRemoveNodeExistingNodeSucceeds verifies remove commits delete existing nodes and parent links.
func TestCommitRegionRemoveNodeExistingNodeSucceeds(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "div",
		GetChildNodeIDs: []uint64{2},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "span",
		GetParentNodeID: 1,
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseCommitResult, parseCommitErr := parseDOMCommitter.CommitRegionRemoveNode("region-42", 2)
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveNode(existing) error = %v", parseCommitErr)
	}
	if parseCommitResult.IsNoOp {
		parseTesting.Fatal("CommitRegionRemoveNode(existing) expected non-no-op result")
	}
	_, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 2)
	if parseLookupErr == nil {
		parseTesting.Fatal("GetRegionDOMNode(removed child) error = nil, want missing node")
	}
	parseParentNode, parseParentLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseParentLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseParentLookupErr)
	}
	if len(parseParentNode.GetChildNodeIDs) != 0 {
		parseTesting.Fatalf("CommitRegionRemoveNode(existing) parent children = %v, want []", parseParentNode.GetChildNodeIDs)
	}
}

// TestCommitRegionRemoveNodeUnknownNodeNoOps verifies unknown removes return a stable no-op outcome.
func TestCommitRegionRemoveNodeUnknownNodeNoOps(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseCommitResult, parseCommitErr := parseDOMCommitter.CommitRegionRemoveNode("region-42", 999)
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveNode(unknown) error = %v", parseCommitErr)
	}
	if !parseCommitResult.IsNoOp {
		parseTesting.Fatal("CommitRegionRemoveNode(unknown) expected no-op result")
	}
}

// TestCommitRegionRemoveNodeRepeatedRemoveDoesNotPanic verifies repeated remove calls remain safe.
func TestCommitRegionRemoveNodeRepeatedRemoveDoesNotPanic(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID: 2,
		GetTag:    "span",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseFirstRemoveErr := parseDOMCommitter.CommitRegionRemoveNode("region-42", 2)
	if parseFirstRemoveErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveNode(first) error = %v", parseFirstRemoveErr)
	}
	parseSecondRemoveResult, parseSecondRemoveErr := parseDOMCommitter.CommitRegionRemoveNode("region-42", 2)
	if parseSecondRemoveErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveNode(second) error = %v", parseSecondRemoveErr)
	}
	if !parseSecondRemoveResult.IsNoOp {
		parseTesting.Fatal("CommitRegionRemoveNode(second) expected no-op result")
	}
}
