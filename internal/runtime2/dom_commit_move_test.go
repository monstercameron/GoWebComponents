package runtime2

import (
	"strings"
	"testing"
)

// TestCommitRegionMoveKeyedNodeValidMoveChangesOrder verifies keyed moves reorder parent children correctly.
func TestCommitRegionMoveKeyedNodeValidMoveChangesOrder(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child a) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child b) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseMoveErr := parseDOMCommitter.CommitRegionMoveKeyedNode("region-42", 1, 3, 2)
	if parseMoveErr != nil {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(valid) error = %v", parseMoveErr)
	}
	parseParentNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseLookupErr)
	}
	parseWant := []uint64{3, 2}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) || parseParentNode.GetChildNodeIDs[0] != parseWant[0] || parseParentNode.GetChildNodeIDs[1] != parseWant[1] {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(valid) children = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
}

// TestCommitRegionMoveKeyedNodeInvalidMoveTargetFails verifies invalid move anchors are rejected.
func TestCommitRegionMoveKeyedNodeInvalidMoveTargetFails(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child a) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child b) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseMoveErr := parseDOMCommitter.CommitRegionMoveKeyedNode("region-42", 1, 3, 99)
	if parseMoveErr == nil {
		parseTesting.Fatal("CommitRegionMoveKeyedNode(invalid target) error = nil, want error")
	}
	if !strings.Contains(parseMoveErr.Error(), "invalid move target") {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(invalid target) error = %q, want invalid-target guidance", parseMoveErr.Error())
	}
}

// TestCommitRegionMoveKeyedNodeAfterRemovalIsRejected verifies moves for removed nodes are rejected.
func TestCommitRegionMoveKeyedNodeAfterRemovalIsRejected(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child a) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child b) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseRemoveErr := parseDOMCommitter.CommitRegionRemoveNode("region-42", 3)
	if parseRemoveErr != nil {
		parseTesting.Fatalf("CommitRegionRemoveNode(before move) error = %v", parseRemoveErr)
	}
	_, parseMoveErr := parseDOMCommitter.CommitRegionMoveKeyedNode("region-42", 1, 3, 2)
	if parseMoveErr == nil {
		parseTesting.Fatal("CommitRegionMoveKeyedNode(after removal) error = nil, want error")
	}
	if !strings.Contains(parseMoveErr.Error(), "missing move node") {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(after removal) error = %q, want missing-node guidance", parseMoveErr.Error())
	}
}
