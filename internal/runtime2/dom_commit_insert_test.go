package runtime2

import (
	"strings"
	"testing"
)

// TestCommitRegionInsertNodeUnderParentSucceeds verifies inserts append under the target parent when no anchor is provided.
func TestCommitRegionInsertNodeUnderParentSucceeds(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "div",
		GetChildNodeIDs: []uint64{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseCommitErr := parseDOMCommitter.CommitRegionInsertNode("region-42", 1, &RegionDOMNode{
		GetNodeID: 2,
		GetTag:    "span",
	}, 0)
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionInsertNode(append) error = %v", parseCommitErr)
	}
	parseParentNode, parseParentLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseParentLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseParentLookupErr)
	}
	if len(parseParentNode.GetChildNodeIDs) != 1 || parseParentNode.GetChildNodeIDs[0] != 2 {
		parseTesting.Fatalf("CommitRegionInsertNode(append) parent children = %v, want [2]", parseParentNode.GetChildNodeIDs)
	}
	parseChildNode, parseChildLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 2)
	if parseChildLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(child) error = %v", parseChildLookupErr)
	}
	if parseChildNode.GetParentNodeID != 1 {
		parseTesting.Fatalf("GetRegionDOMNode(child) parent = %d, want 1", parseChildNode.GetParentNodeID)
	}
}

// TestCommitRegionInsertNodeBeforeSiblingAnchorSucceeds verifies inserts honor a valid sibling anchor ordering point.
func TestCommitRegionInsertNodeBeforeSiblingAnchorSucceeds(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "div",
		GetChildNodeIDs: []uint64{3},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "span",
		GetParentNodeID: 1,
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(sibling) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseCommitErr := parseDOMCommitter.CommitRegionInsertNode("region-42", 1, &RegionDOMNode{
		GetNodeID: 2,
		GetTag:    "span",
	}, 3)
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionInsertNode(before anchor) error = %v", parseCommitErr)
	}
	parseParentNode, parseParentLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseParentLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseParentLookupErr)
	}
	parseWant := []uint64{2, 3}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) || parseParentNode.GetChildNodeIDs[0] != parseWant[0] || parseParentNode.GetChildNodeIDs[1] != parseWant[1] {
		parseTesting.Fatalf("CommitRegionInsertNode(before anchor) parent children = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
}

// TestCommitRegionInsertNodeInvalidAnchorFailsCleanly verifies unknown sibling anchors are rejected without partial insertion.
func TestCommitRegionInsertNodeInvalidAnchorFailsCleanly(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "div",
		GetChildNodeIDs: []uint64{3},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "span",
		GetParentNodeID: 1,
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(sibling) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseCommitErr := parseDOMCommitter.CommitRegionInsertNode("region-42", 1, &RegionDOMNode{
		GetNodeID: 2,
		GetTag:    "span",
	}, 99)
	if parseCommitErr == nil {
		parseTesting.Fatal("CommitRegionInsertNode(invalid anchor) error = nil, want error")
	}
	if !strings.Contains(parseCommitErr.Error(), "invalid sibling anchor") {
		parseTesting.Fatalf("CommitRegionInsertNode(invalid anchor) error = %q, want invalid-anchor guidance", parseCommitErr.Error())
	}
	_, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 2)
	if parseLookupErr == nil {
		parseTesting.Fatal("GetRegionDOMNode(inserted node after invalid anchor) error = nil, want missing node")
	}
}
