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
