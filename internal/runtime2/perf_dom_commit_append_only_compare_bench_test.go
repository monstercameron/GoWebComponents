package runtime2

import (
	"strconv"
	"testing"
)

var storeDOMCommitAppendOnlyResultSink RegionPatchTransactionResult

// buildCommitRegionPatchTransactionLegacyAppendOnlyGeneric preserves the previous append-only transaction path that still paid the generic snapshot-capture and per-op dispatch loop.
func buildCommitRegionPatchTransactionLegacyAppendOnlyGeneric(
	parseDOMCommitter *DOMCommitter,
	parseTransaction RegionPatchTransaction,
) (RegionPatchTransactionResult, error) {
	if parseDOMCommitter == nil {
		return RegionPatchTransactionResult{}, nil
	}
	parseRegionNodeMap := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID]
	var parseNodeSnapshotByID map[uint64]*RegionDOMNode
	parseDOMCommitter.beginCommitMutationBatch()
	isCommitSuccessful := false
	defer func() {
		parseDOMCommitter.finishCommitMutationBatch(parseTransaction.GetRegionID, isCommitSuccessful)
	}()
	for parseOpIndex, parseOp := range parseTransaction.GetOps {
		parseNodeSnapshotByID = parseCaptureCommitNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp)
		if parseApplyErr := parseDOMCommitter.parseCommitPatchOp(parseTransaction.GetRegionID, parseOp); parseApplyErr != nil {
			parseRestoreCommitNodeSnapshot(parseRegionNodeMap, parseNodeSnapshotByID)
			if len(parseRegionNodeMap) > 0 {
				parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID] = parseRegionNodeMap
			}
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, parseApplyErr
		}
		if parseOpIndex == len(parseTransaction.GetOps)-1 {
			isCommitSuccessful = true
		}
	}
	return RegionPatchTransactionResult{}, nil
}

// BenchmarkCommitRegionPatchTransactionAppendOnlyCurrentVsLegacy compares the append-only fast path against the previous generic insert transaction loop.
func BenchmarkCommitRegionPatchTransactionAppendOnlyCurrentVsLegacy(parseB *testing.B) {
	parseAppendCounts := []int{32, 128, 256}
	for _, parseAppendCount := range parseAppendCounts {
		parseB.Run("append-"+strconv.Itoa(parseAppendCount)+"/legacy_generic", func(parseB *testing.B) {
			parseB.ReportAllocs()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getFixture := buildDOMCommitAppendSnapshotBenchmarkFixture(parseB, 512, parseAppendCount)
				getResult, parseCommitErr := buildCommitRegionPatchTransactionLegacyAppendOnlyGeneric(getFixture.getDOMCommitter, getFixture.getTransaction)
				if parseCommitErr != nil {
					parseB.Fatalf("buildCommitRegionPatchTransactionLegacyAppendOnlyGeneric returned error: %v", parseCommitErr)
				}
				storeDOMCommitAppendOnlyResultSink = getResult
			}
		})
		parseB.Run("append-"+strconv.Itoa(parseAppendCount)+"/current_fast_path", func(parseB *testing.B) {
			parseB.ReportAllocs()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getFixture := buildDOMCommitAppendSnapshotBenchmarkFixture(parseB, 512, parseAppendCount)
				getResult, parseCommitErr := getFixture.getDOMCommitter.CommitRegionPatchTransaction(getFixture.getTransaction)
				if parseCommitErr != nil {
					parseB.Fatalf("CommitRegionPatchTransaction returned error: %v", parseCommitErr)
				}
				storeDOMCommitAppendOnlyResultSink = getResult
			}
		})
	}
}
