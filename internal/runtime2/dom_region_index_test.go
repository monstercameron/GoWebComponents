package runtime2

import (
	"strings"
	"testing"
)

// TestSetRegionDOMNodeInsertSucceeds verifies region-local DOM index insertion succeeds.
func TestSetRegionDOMNodeInsertSucceeds(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 17, &RegionDOMNode{
		GetNodeID: 17,
		GetTag:    "span",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(valid) error = %v", parseInsertErr)
	}
}

// TestGetRegionDOMNodeLookupSucceeds verifies lookup returns the indexed DOM node.
func TestGetRegionDOMNodeLookupSucceeds(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 17, &RegionDOMNode{
		GetNodeID: 17,
		GetTag:    "span",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(valid) error = %v", parseInsertErr)
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 17)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(valid) error = %v", parseLookupErr)
	}
	if parseNode.GetNodeID != 17 {
		parseTesting.Fatalf("GetRegionDOMNode(valid) node ID = %d, want 17", parseNode.GetNodeID)
	}
}

// TestGetRegionDOMNodeMissingLookupFails verifies lookup fails clearly for missing nodes.
func TestGetRegionDOMNodeMissingLookupFails(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	_, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 99)
	if parseLookupErr == nil {
		parseTesting.Fatal("GetRegionDOMNode(missing) error = nil, want error")
	}
	if !strings.Contains(parseLookupErr.Error(), "missing node ID") {
		parseTesting.Fatalf("GetRegionDOMNode(missing) error = %q, want missing-node guidance", parseLookupErr.Error())
	}
}

// TestRegionDOMIndexRejectsInvalidInputsAndIgnoresDuplicateClearTargets verifies index guards reject nil or empty inputs while partial clears ignore duplicate, zero, and missing node IDs.
func TestRegionDOMIndexRejectsInvalidInputsAndIgnoresDuplicateClearTargets(parseTesting *testing.T) {
	var parseNilRegionDOMIndex *RegionDOMIndex
	if parseErr := parseNilRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{GetNodeID: 1}); parseErr == nil {
		parseTesting.Fatal("expected nil region DOM index set to fail")
	}
	if _, parseErr := parseNilRegionDOMIndex.GetRegionDOMNode("region-42", 1); parseErr == nil {
		parseTesting.Fatal("expected nil region DOM index lookup to fail")
	}
	if parseClearResult := parseNilRegionDOMIndex.ClearRegionDOMNodes("region-42", nil); parseClearResult.GetClearedNodeCount != 0 {
		parseTesting.Fatalf("expected nil region DOM index clear to be ignored, got %+v", parseClearResult)
	}

	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("", 1, &RegionDOMNode{GetNodeID: 1}); parseErr == nil {
		parseTesting.Fatal("expected empty region ID set to fail")
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 0, &RegionDOMNode{GetNodeID: 0}); parseErr == nil {
		parseTesting.Fatal("expected zero node ID set to fail")
	}
	if parseErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, nil); parseErr == nil {
		parseTesting.Fatal("expected nil region DOM node set to fail")
	}
	if _, parseErr := parseRegionDOMIndex.GetRegionDOMNode("", 1); parseErr == nil {
		parseTesting.Fatal("expected empty region ID lookup to fail")
	}
	if _, parseErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 0); parseErr == nil {
		parseTesting.Fatal("expected zero node ID lookup to fail")
	}
	if parseClearResult := parseRegionDOMIndex.ClearRegionDOMNodes("", nil); parseClearResult.GetClearedNodeCount != 0 {
		parseTesting.Fatalf("expected empty-region clear to be ignored, got %+v", parseClearResult)
	}

	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{GetNodeID: 1, GetTag: "div"}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=1) error = %v", parseInsertErr)
	}
	if parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{GetNodeID: 2, GetTag: "span"}); parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=2) error = %v", parseInsertErr)
	}
	parseClearResult := parseRegionDOMIndex.ClearRegionDOMNodes("region-42", []uint64{0, 1, 1, 9, 2})
	if parseClearResult.GetClearedNodeCount != 2 {
		parseTesting.Fatalf("ClearRegionDOMNodes(duplicate and missing targets) cleared count = %d, want 2", parseClearResult.GetClearedNodeCount)
	}
	if _, parseErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1); parseErr == nil {
		parseTesting.Fatal("expected cleared node 1 lookup to fail")
	}
	if _, parseErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 2); parseErr == nil {
		parseTesting.Fatal("expected cleared node 2 lookup to fail")
	}
}
