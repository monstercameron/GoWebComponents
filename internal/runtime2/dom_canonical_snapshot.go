package runtime2

import "fmt"

// RegionDOMCanonicalSnapshotResult reports one applied canonical DOM snapshot size.
type RegionDOMCanonicalSnapshotResult struct {
	GetNodeCount int
}

// ApplyRegionDOMCanonicalSnapshot replaces one region-local DOM index subtree with the provided canonical render IR.
func ApplyRegionDOMCanonicalSnapshot(
	parseRegionDOMIndex *RegionDOMIndex,
	parseRegionID string,
	parseCanonicalIR CanonicalRenderIR,
) (RegionDOMCanonicalSnapshotResult, error) {
	if parseRegionDOMIndex == nil {
		return RegionDOMCanonicalSnapshotResult{}, fmt.Errorf("runtime2: region DOM index is nil")
	}
	if parseRegionID == "" {
		return RegionDOMCanonicalSnapshotResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	parseTree, parseTreeErr := ParseCanonicalRenderTree(parseCanonicalIR)
	if parseTreeErr != nil {
		return RegionDOMCanonicalSnapshotResult{}, parseTreeErr
	}
	parseRegionNodeByNodeID := parseBuildRegionDOMNodeByNodeIDFromCanonicalTree(parseTree)
	parseRegionDOMIndex.ClearRegionDOMNodes(parseRegionID, nil)
	for getNodeID, getRegionDOMNode := range parseRegionNodeByNodeID {
		if parseSetErr := parseRegionDOMIndex.SetRegionDOMNode(parseRegionID, getNodeID, getRegionDOMNode); parseSetErr != nil {
			return RegionDOMCanonicalSnapshotResult{}, parseSetErr
		}
	}
	return RegionDOMCanonicalSnapshotResult{
		GetNodeCount: len(parseRegionNodeByNodeID),
	}, nil
}
