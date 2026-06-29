package runtime2

import "testing"

// TestClearRegionDOMNodesDisposeClearsIndexedNodes verifies full region cleanup removes indexed node mappings.
func TestClearRegionDOMNodesDisposeClearsIndexedNodes(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{GetNodeID: 1, GetTag: "div"})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=1) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{GetNodeID: 2, GetTag: "span"})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=2) error = %v", parseInsertErr)
	}
	parseClearResult := parseRegionDOMIndex.ClearRegionDOMNodes("region-42", nil)
	if parseClearResult.GetClearedNodeCount != 2 {
		parseTesting.Fatalf("ClearRegionDOMNodes(full) cleared count = %d, want 2", parseClearResult.GetClearedNodeCount)
	}
	_, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr == nil {
		parseTesting.Fatal("GetRegionDOMNode(node=1 after full clear) error = nil, want error")
	}
}

// TestClearRegionDOMNodesRepeatedDisposeLeavesStableIndex verifies repeated dispose operations remain stable.
func TestClearRegionDOMNodesRepeatedDisposeLeavesStableIndex(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{GetNodeID: 1, GetTag: "div"})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=1) error = %v", parseInsertErr)
	}
	parseFirstClearResult := parseRegionDOMIndex.ClearRegionDOMNodes("region-42", nil)
	if parseFirstClearResult.GetClearedNodeCount != 1 {
		parseTesting.Fatalf("ClearRegionDOMNodes(first full clear) cleared count = %d, want 1", parseFirstClearResult.GetClearedNodeCount)
	}
	parseSecondClearResult := parseRegionDOMIndex.ClearRegionDOMNodes("region-42", nil)
	if parseSecondClearResult.GetClearedNodeCount != 0 {
		parseTesting.Fatalf("ClearRegionDOMNodes(second full clear) cleared count = %d, want 0", parseSecondClearResult.GetClearedNodeCount)
	}
}

// TestClearRegionDOMNodesPartialCleanupAvoidsStaleMappings verifies partial cleanup removes only targeted node mappings.
func TestClearRegionDOMNodesPartialCleanupAvoidsStaleMappings(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{GetNodeID: 1, GetTag: "div"})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=1) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 2, &RegionDOMNode{GetNodeID: 2, GetTag: "span"})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=2) error = %v", parseInsertErr)
	}
	parseInsertErr = parseRegionDOMIndex.SetRegionDOMNode("region-42", 3, &RegionDOMNode{GetNodeID: 3, GetTag: "p"})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(node=3) error = %v", parseInsertErr)
	}
	parseClearResult := parseRegionDOMIndex.ClearRegionDOMNodes("region-42", []uint64{1, 3})
	if parseClearResult.GetClearedNodeCount != 2 {
		parseTesting.Fatalf("ClearRegionDOMNodes(partial) cleared count = %d, want 2", parseClearResult.GetClearedNodeCount)
	}
	_, parseNode1Err := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseNode1Err == nil {
		parseTesting.Fatal("GetRegionDOMNode(node=1 after partial clear) error = nil, want error")
	}
	_, parseNode3Err := parseRegionDOMIndex.GetRegionDOMNode("region-42", 3)
	if parseNode3Err == nil {
		parseTesting.Fatal("GetRegionDOMNode(node=3 after partial clear) error = nil, want error")
	}
	parseNode2, parseNode2Err := parseRegionDOMIndex.GetRegionDOMNode("region-42", 2)
	if parseNode2Err != nil {
		parseTesting.Fatalf("GetRegionDOMNode(node=2 after partial clear) error = %v", parseNode2Err)
	}
	if parseNode2.GetNodeID != 2 {
		parseTesting.Fatalf("GetRegionDOMNode(node=2 after partial clear) node ID = %d, want 2", parseNode2.GetNodeID)
	}
}
