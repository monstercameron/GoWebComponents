package runtime2

import (
	"fmt"
	"strings"
	"testing"
)

var storeDOMCommitRollbackResultSink RegionPatchTransactionResult

// buildDOMCommitRollbackBenchFixture stores one rollback benchmark fixture for current-vs-legacy commit comparisons.
type buildDOMCommitRollbackBenchFixture struct {
	getDOMCommitter *DOMCommitter
	getTransaction  RegionPatchTransaction
}

// buildDOMCommitRollbackBenchmarkFixture builds one repeatable failed-transaction workload that always enters rollback.
func buildDOMCommitRollbackBenchmarkFixture(parseB *testing.B) buildDOMCommitRollbackBenchFixture {
	parseB.Helper()
	const getBenchNodeCount = 1024
	buildRegionDOMIndex := BuildRegionDOMIndex()
	for parseNodeOffset := 0; parseNodeOffset < getBenchNodeCount; parseNodeOffset++ {
		getNodeID := uint64(parseNodeOffset + 1)
		if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", getNodeID, &RegionDOMNode{
			GetNodeID:    getNodeID,
			GetTag:       "div",
			GetText:      "before",
			GetAttrByKey: map[string]string{},
		}); parseSetNodeErr != nil {
			parseB.Fatalf("SetRegionDOMNode returned error: %v", parseSetNodeErr)
		}
	}
	return buildDOMCommitRollbackBenchFixture{
		getDOMCommitter: BuildDOMCommitter(buildRegionDOMIndex),
		getTransaction: RegionPatchTransaction{
			GetRegionID: "bench-region",
			GetOps: []RegionPatchOp{
				{
					GetKind:   RegionPatchOpKindSetText,
					GetNodeID: 1,
					GetText:   "after",
				},
				{
					GetKind:   RegionPatchOpKindSetText,
					GetNodeID: uint64(getBenchNodeCount + 99_999),
					GetText:   "broken",
				},
			},
		},
	}
}

// buildCommitRegionPatchTransactionLegacyRollbackClone preserves the previous rollback path that cloned snapshots again while restoring.
func buildCommitRegionPatchTransactionLegacyRollbackClone(
	parseDOMCommitter *DOMCommitter,
	parseTransaction RegionPatchTransaction,
) (RegionPatchTransactionResult, error) {
	if parseDOMCommitter == nil {
		return RegionPatchTransactionResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	if strings.TrimSpace(parseTransaction.GetRegionID) == "" {
		return RegionPatchTransactionResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	if len(parseTransaction.GetOps) == 0 {
		return RegionPatchTransactionResult{}, nil
	}
	parseRegionNodeMap := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID]
	parseRegionSnapshot := parseCloneRegionNodeMap(parseRegionNodeMap)
	for parseOpIndex, parseOp := range parseTransaction.GetOps {
		if parseApplyErr := parseDOMCommitter.parseCommitPatchOp(parseTransaction.GetRegionID, parseOp); parseApplyErr != nil {
			parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID] = parseCloneRegionNodeMap(parseRegionSnapshot)
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, parseApplyErr)
		}
	}
	return RegionPatchTransactionResult{}, nil
}

// BenchmarkCommitRegionPatchTransactionRollbackCurrentVsLegacy compares current rollback restore against the previous double-clone snapshot restore.
func BenchmarkCommitRegionPatchTransactionRollbackCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("legacy_clone_on_restore", func(parseB *testing.B) {
		getFixture := buildDOMCommitRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := buildCommitRegionPatchTransactionLegacyRollbackClone(getFixture.getDOMCommitter, getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyRollbackClone error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyRollbackClone fallback = false, want true")
			}
			storeDOMCommitRollbackResultSink = getResult
		}
	})
	parseB.Run("current_restore_snapshot", func(parseB *testing.B) {
		getFixture := buildDOMCommitRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := getFixture.getDOMCommitter.CommitRegionPatchTransaction(getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("CommitRegionPatchTransaction error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("CommitRegionPatchTransaction fallback = false, want true")
			}
			storeDOMCommitRollbackResultSink = getResult
		}
	})
}
