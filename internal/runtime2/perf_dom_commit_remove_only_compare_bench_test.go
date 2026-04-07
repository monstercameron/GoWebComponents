package runtime2

import (
	"strconv"
	"testing"
)

var storeDOMCommitRemoveOnlyResultSink RegionPatchTransactionResult

// buildDOMCommitRemoveOnlyBenchFixture stores one remove-only transaction benchmark fixture for current-vs-legacy comparisons.
type buildDOMCommitRemoveOnlyBenchFixture struct {
	getDOMCommitter *DOMCommitter
	getTransaction  RegionPatchTransaction
}

// buildDOMCommitRemoveOnlyFilterBenchmarkFixture builds one filter-shaped remove-only transaction over one keyed sibling list.
func buildDOMCommitRemoveOnlyFilterBenchmarkFixture(parseB *testing.B, parseSiblingCount int) buildDOMCommitRemoveOnlyBenchFixture {
	parseB.Helper()
	if parseSiblingCount < 3 {
		parseSiblingCount = 3
	}
	buildRegionDOMIndex := BuildRegionDOMIndex()
	buildChildNodeIDs := make([]uint64, 0, parseSiblingCount)
	buildOps := make([]RegionPatchOp, 0, parseSiblingCount)
	if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: buildChildNodeIDs,
	}); parseSetNodeErr != nil {
		parseB.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetNodeErr)
	}
	for parseSiblingIndex := 0; parseSiblingIndex < parseSiblingCount; parseSiblingIndex++ {
		getNodeID := uint64(parseSiblingIndex + 2)
		buildChildNodeIDs = append(buildChildNodeIDs, getNodeID)
		if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row-" + strconv.Itoa(parseSiblingIndex),
		}); parseSetNodeErr != nil {
			parseB.Fatalf("SetRegionDOMNode(child=%d) returned error: %v", getNodeID, parseSetNodeErr)
		}
		if parseSiblingIndex%3 != 0 {
			buildOps = append(buildOps, RegionPatchOp{
				GetKind:   RegionPatchOpKindRemoveNode,
				GetNodeID: getNodeID,
			})
		}
	}
	buildParentNode, parseLookupErr := buildRegionDOMIndex.GetRegionDOMNode("bench-region", 1)
	if parseLookupErr != nil {
		parseB.Fatalf("GetRegionDOMNode(parent) returned error: %v", parseLookupErr)
	}
	buildParentNode.GetChildNodeIDs = buildChildNodeIDs
	return buildDOMCommitRemoveOnlyBenchFixture{
		getDOMCommitter: BuildDOMCommitter(buildRegionDOMIndex),
		getTransaction: RegionPatchTransaction{
			GetRegionID: "bench-region",
			GetOps:      buildOps,
		},
	}
}

// buildCommitRegionPatchTransactionLegacyRemoveOnlySequential preserves the previous remove-only transaction path that committed one remove op at a time.
func buildCommitRegionPatchTransactionLegacyRemoveOnlySequential(
	parseDOMCommitter *DOMCommitter,
	parseTransaction RegionPatchTransaction,
) (RegionPatchTransactionResult, error) {
	if parseDOMCommitter == nil {
		return RegionPatchTransactionResult{}, nil
	}
	for _, parseOp := range parseTransaction.GetOps {
		if parseApplyErr := parseDOMCommitter.parseCommitPatchOp(parseTransaction.GetRegionID, parseOp); parseApplyErr != nil {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, parseApplyErr
		}
	}
	return RegionPatchTransactionResult{}, nil
}

// BenchmarkCommitRegionPatchTransactionRemoveOnlyCurrentVsLegacy compares the batched remove-only transaction path against the previous sequential remove-op loop.
func BenchmarkCommitRegionPatchTransactionRemoveOnlyCurrentVsLegacy(parseB *testing.B) {
	parseSiblingCounts := []int{120, 240, 480}
	for _, parseSiblingCount := range parseSiblingCounts {
		parseSiblingCount := parseSiblingCount
		parseB.Run("siblings-"+strconv.Itoa(parseSiblingCount)+"/legacy_sequential", func(parseB *testing.B) {
			parseB.ReportAllocs()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getFixture := buildDOMCommitRemoveOnlyFilterBenchmarkFixture(parseB, parseSiblingCount)
				getResult, parseCommitErr := buildCommitRegionPatchTransactionLegacyRemoveOnlySequential(getFixture.getDOMCommitter, getFixture.getTransaction)
				if parseCommitErr != nil {
					parseB.Fatalf("buildCommitRegionPatchTransactionLegacyRemoveOnlySequential returned error: %v", parseCommitErr)
				}
				storeDOMCommitRemoveOnlyResultSink = getResult
			}
		})
		parseB.Run("siblings-"+strconv.Itoa(parseSiblingCount)+"/current_batched", func(parseB *testing.B) {
			parseB.ReportAllocs()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getFixture := buildDOMCommitRemoveOnlyFilterBenchmarkFixture(parseB, parseSiblingCount)
				getResult, parseCommitErr := getFixture.getDOMCommitter.CommitRegionPatchTransaction(getFixture.getTransaction)
				if parseCommitErr != nil {
					parseB.Fatalf("CommitRegionPatchTransaction returned error: %v", parseCommitErr)
				}
				storeDOMCommitRemoveOnlyResultSink = getResult
			}
		})
	}
}
