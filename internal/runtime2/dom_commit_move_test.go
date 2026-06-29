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

// TestCommitRegionMoveKeyedNodeMoveBeforeNextSiblingIsNoOp verifies moves to the current destination index avoid rewriting sibling order.
func TestCommitRegionMoveKeyedNodeMoveBeforeNextSiblingIsNoOp(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3, 4},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	for _, getNodeID := range []uint64{2, 3, 4} {
		parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row",
		})
		if parseInsertErr != nil {
			parseTesting.Fatalf("SetRegionDOMNode(child=%d) error = %v", getNodeID, parseInsertErr)
		}
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseMoveResult, parseMoveErr := parseDOMCommitter.CommitRegionMoveKeyedNode("region-42", 1, 3, 4)
	if parseMoveErr != nil {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(no-op) error = %v", parseMoveErr)
	}
	if !parseMoveResult.IsNoOp {
		parseTesting.Fatal("CommitRegionMoveKeyedNode(no-op) result should report IsNoOp")
	}
	parseParentNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseLookupErr)
	}
	parseWant := []uint64{2, 3, 4}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) || parseParentNode.GetChildNodeIDs[0] != parseWant[0] || parseParentNode.GetChildNodeIDs[1] != parseWant[1] || parseParentNode.GetChildNodeIDs[2] != parseWant[2] {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(no-op) children = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
}

// TestCommitRegionMoveKeyedNodeMoveToEndChangesOrder verifies keyed moves can append a child to the tail without rebuilding sibling slices.
func TestCommitRegionMoveKeyedNodeMoveToEndChangesOrder(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3, 4},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	for _, getNodeID := range []uint64{2, 3, 4} {
		parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row",
		})
		if parseInsertErr != nil {
			parseTesting.Fatalf("SetRegionDOMNode(child=%d) error = %v", getNodeID, parseInsertErr)
		}
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseMoveErr := parseDOMCommitter.CommitRegionMoveKeyedNode("region-42", 1, 2, 0)
	if parseMoveErr != nil {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(move to end) error = %v", parseMoveErr)
	}
	parseParentNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(parent) error = %v", parseLookupErr)
	}
	parseWant := []uint64{3, 4, 2}
	if len(parseParentNode.GetChildNodeIDs) != len(parseWant) || parseParentNode.GetChildNodeIDs[0] != parseWant[0] || parseParentNode.GetChildNodeIDs[1] != parseWant[1] || parseParentNode.GetChildNodeIDs[2] != parseWant[2] {
		parseTesting.Fatalf("CommitRegionMoveKeyedNode(move to end) children = %v, want %v", parseParentNode.GetChildNodeIDs, parseWant)
	}
}

// TestParseResolveMoveBeforeNodeIDUsesCurrentSiblingOrder verifies destination-index moves map onto the correct sibling anchor without rebuilding the sibling slice.
func TestParseResolveMoveBeforeNodeIDUsesCurrentSiblingOrder(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 3, 4},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) error = %v", parseInsertErr)
	}
	for _, getNodeID := range []uint64{2, 3, 4} {
		parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row",
		})
		if parseInsertErr != nil {
			parseTesting.Fatalf("SetRegionDOMNode(child=%d) error = %v", getNodeID, parseInsertErr)
		}
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseBeforeNodeID, parseBeforeNodeErr := parseResolveMoveBeforeNodeID(parseDOMCommitter, "region-42", 1, 3, 0)
	if parseBeforeNodeErr != nil {
		parseTesting.Fatalf("parseResolveMoveBeforeNodeID(destination=0) error = %v", parseBeforeNodeErr)
	}
	if parseBeforeNodeID != 2 {
		parseTesting.Fatalf("parseResolveMoveBeforeNodeID(destination=0) = %d, want 2", parseBeforeNodeID)
	}
	parseBeforeNodeID, parseBeforeNodeErr = parseResolveMoveBeforeNodeID(parseDOMCommitter, "region-42", 1, 3, 1)
	if parseBeforeNodeErr != nil {
		parseTesting.Fatalf("parseResolveMoveBeforeNodeID(destination=1) error = %v", parseBeforeNodeErr)
	}
	if parseBeforeNodeID != 4 {
		parseTesting.Fatalf("parseResolveMoveBeforeNodeID(destination=1) = %d, want 4", parseBeforeNodeID)
	}
	parseBeforeNodeID, parseBeforeNodeErr = parseResolveMoveBeforeNodeID(parseDOMCommitter, "region-42", 1, 3, 2)
	if parseBeforeNodeErr != nil {
		parseTesting.Fatalf("parseResolveMoveBeforeNodeID(destination=2) error = %v", parseBeforeNodeErr)
	}
	if parseBeforeNodeID != 0 {
		parseTesting.Fatalf("parseResolveMoveBeforeNodeID(destination=2) = %d, want 0", parseBeforeNodeID)
	}
}
