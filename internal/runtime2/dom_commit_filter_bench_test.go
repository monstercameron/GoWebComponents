package runtime2

import (
	"strconv"
	"testing"
)

const benchmarkDOMCommitFilterRegionID string = "bench-region-filter"

const benchmarkDOMCommitFilterParentNodeID uint64 = 1

// buildBenchmarkDOMCommitFilterSetup seeds one parent with keyed sibling rows and returns the removal set used by filter-heavy updates.
func buildBenchmarkDOMCommitFilterSetup(parseB *testing.B, parseSiblingCount int) (*DOMCommitter, []uint64) {
	parseB.Helper()
	if parseSiblingCount < 3 {
		parseSiblingCount = 3
	}
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseChildNodeIDs := make([]uint64, 0, parseSiblingCount)
	parseRemoveNodeIDs := make([]uint64, 0, parseSiblingCount)
	for parseSiblingIndex := 0; parseSiblingIndex < parseSiblingCount; parseSiblingIndex++ {
		parseNodeID := uint64(parseSiblingIndex + 2)
		parseChildNodeIDs = append(parseChildNodeIDs, parseNodeID)
		if parseSiblingIndex%3 != 0 {
			parseRemoveNodeIDs = append(parseRemoveNodeIDs, parseNodeID)
		}
		parseInsertChildErr := parseRegionDOMIndex.SetRegionDOMNode(benchmarkDOMCommitFilterRegionID, parseNodeID, &RegionDOMNode{
			GetNodeID:       parseNodeID,
			GetTag:          "li",
			GetParentNodeID: benchmarkDOMCommitFilterParentNodeID,
			GetNodeKey:      "item",
		})
		if parseInsertChildErr != nil {
			parseB.Fatalf("SetRegionDOMNode(child) returned error: %v", parseInsertChildErr)
		}
	}
	parseInsertParentErr := parseRegionDOMIndex.SetRegionDOMNode(benchmarkDOMCommitFilterRegionID, benchmarkDOMCommitFilterParentNodeID, &RegionDOMNode{
		GetNodeID:       benchmarkDOMCommitFilterParentNodeID,
		GetTag:          "ul",
		GetChildNodeIDs: parseChildNodeIDs,
	})
	if parseInsertParentErr != nil {
		parseB.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseInsertParentErr)
	}
	return BuildDOMCommitter(parseRegionDOMIndex), parseRemoveNodeIDs
}

// BenchmarkCommitRegionRemoveNodeFilterHeavy benchmarks repeated sibling removals that mirror the Core Filter update shape.
func BenchmarkCommitRegionRemoveNodeFilterHeavy(parseB *testing.B) {
	parseSiblingCounts := []int{120, 240, 480}
	for _, parseSiblingCount := range parseSiblingCounts {
		parseSiblingCount := parseSiblingCount
		parseB.Run("siblings-"+strconv.Itoa(parseSiblingCount), func(parseB *testing.B) {
			parseB.ReportAllocs()
			for parseIteration := 0; parseIteration < parseB.N; parseIteration++ {
				parseB.StopTimer()
				parseDOMCommitter, parseRemoveNodeIDs := buildBenchmarkDOMCommitFilterSetup(parseB, parseSiblingCount)
				parseB.StartTimer()
				for _, parseRemoveNodeID := range parseRemoveNodeIDs {
					if _, parseCommitErr := parseDOMCommitter.CommitRegionRemoveNode(benchmarkDOMCommitFilterRegionID, parseRemoveNodeID); parseCommitErr != nil {
						parseB.Fatalf("CommitRegionRemoveNode returned error: %v", parseCommitErr)
					}
				}
			}
		})
	}
}
