package runtime2

import (
	"fmt"
	"strings"
	"testing"
)

var storeDOMCommitStructuralRollbackResultSink RegionPatchTransactionResult

// buildDOMCommitStructuralRollbackBenchFixture stores one structural rollback benchmark fixture for current-vs-legacy commit comparisons.
type buildDOMCommitStructuralRollbackBenchFixture struct {
	getDOMCommitter *DOMCommitter
	getTransaction  RegionPatchTransaction
}

// buildDOMCommitInsertRollbackBenchmarkFixture builds one append-heavy failed transaction that should only need parent-local rollback state.
func buildDOMCommitInsertRollbackBenchmarkFixture(parseB *testing.B) buildDOMCommitStructuralRollbackBenchFixture {
	parseB.Helper()
	const getBenchChildCount = 2048
	buildRegionDOMIndex := BuildRegionDOMIndex()
	buildChildNodeIDs := make([]uint64, 0, getBenchChildCount)
	if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: buildChildNodeIDs,
	}); parseSetNodeErr != nil {
		parseB.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetNodeErr)
	}
	for parseChildOffset := 0; parseChildOffset < getBenchChildCount; parseChildOffset++ {
		getNodeID := uint64(parseChildOffset + 2)
		buildChildNodeIDs = append(buildChildNodeIDs, getNodeID)
		if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row",
		}); parseSetNodeErr != nil {
			parseB.Fatalf("SetRegionDOMNode(child=%d) returned error: %v", getNodeID, parseSetNodeErr)
		}
	}
	buildParentNode, parseLookupErr := buildRegionDOMIndex.GetRegionDOMNode("bench-region", 1)
	if parseLookupErr != nil {
		parseB.Fatalf("GetRegionDOMNode(parent) returned error: %v", parseLookupErr)
	}
	buildParentNode.GetChildNodeIDs = buildChildNodeIDs
	return buildDOMCommitStructuralRollbackBenchFixture{
		getDOMCommitter: BuildDOMCommitter(buildRegionDOMIndex),
		getTransaction: RegionPatchTransaction{
			GetRegionID: "bench-region",
			GetOps: []RegionPatchOp{
				{
					GetKind:         RegionPatchOpKindInsertNode,
					GetParentNodeID: 1,
					GetInsertNode: &RegionDOMNode{
						GetNodeID: uint64(getBenchChildCount + 2),
						GetTag:    "li",
					},
				},
				{
					GetKind:   RegionPatchOpKindSetText,
					GetNodeID: 999_999,
					GetText:   "broken",
				},
			},
		},
	}
}

// buildDOMCommitMoveRollbackBenchmarkFixture builds one large keyed-move failed transaction that should only need parent-local rollback state.
func buildDOMCommitMoveRollbackBenchmarkFixture(parseB *testing.B) buildDOMCommitStructuralRollbackBenchFixture {
	parseB.Helper()
	const getBenchChildCount = 2048
	buildRegionDOMIndex := BuildRegionDOMIndex()
	buildChildNodeIDs := make([]uint64, 0, getBenchChildCount)
	if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: buildChildNodeIDs,
	}); parseSetNodeErr != nil {
		parseB.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetNodeErr)
	}
	for parseChildOffset := 0; parseChildOffset < getBenchChildCount; parseChildOffset++ {
		getNodeID := uint64(parseChildOffset + 2)
		buildChildNodeIDs = append(buildChildNodeIDs, getNodeID)
		if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row",
		}); parseSetNodeErr != nil {
			parseB.Fatalf("SetRegionDOMNode(child=%d) returned error: %v", getNodeID, parseSetNodeErr)
		}
	}
	buildParentNode, parseLookupErr := buildRegionDOMIndex.GetRegionDOMNode("bench-region", 1)
	if parseLookupErr != nil {
		parseB.Fatalf("GetRegionDOMNode(parent) returned error: %v", parseLookupErr)
	}
	buildParentNode.GetChildNodeIDs = buildChildNodeIDs
	return buildDOMCommitStructuralRollbackBenchFixture{
		getDOMCommitter: BuildDOMCommitter(buildRegionDOMIndex),
		getTransaction: RegionPatchTransaction{
			GetRegionID: "bench-region",
			GetOps: []RegionPatchOp{
				{
					GetKind:             RegionPatchOpKindMoveKeyedNode,
					GetParentNodeID:     1,
					GetMoveNodeID:       uint64(getBenchChildCount + 1),
					GetDestinationIndex: 0,
				},
				{
					GetKind:   RegionPatchOpKindSetText,
					GetNodeID: 999_999,
					GetText:   "broken",
				},
			},
		},
	}
}

// buildDOMCommitRemoveRollbackBenchmarkFixture builds one filter-style failed transaction that removes one large keyed subtree before a forced rollback.
func buildDOMCommitRemoveRollbackBenchmarkFixture(parseB *testing.B) buildDOMCommitStructuralRollbackBenchFixture {
	parseB.Helper()
	const getBenchChildCount = 2048
	buildRegionDOMIndex := BuildRegionDOMIndex()
	buildChildNodeIDs := make([]uint64, 0, 2)
	if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: buildChildNodeIDs,
	}); parseSetNodeErr != nil {
		parseB.Fatalf("SetRegionDOMNode(root) returned error: %v", parseSetNodeErr)
	}
	buildSubtreeChildNodeIDs := make([]uint64, 0, getBenchChildCount)
	if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetChildNodeIDs: buildSubtreeChildNodeIDs,
		GetNodeKey:      "row-a",
	}); parseSetNodeErr != nil {
		parseB.Fatalf("SetRegionDOMNode(subtree root) returned error: %v", parseSetNodeErr)
	}
	if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	}); parseSetNodeErr != nil {
		parseB.Fatalf("SetRegionDOMNode(sibling) returned error: %v", parseSetNodeErr)
	}
	for parseChildOffset := 0; parseChildOffset < getBenchChildCount; parseChildOffset++ {
		getNodeID := uint64(parseChildOffset + 4)
		buildSubtreeChildNodeIDs = append(buildSubtreeChildNodeIDs, getNodeID)
		if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "span",
			GetParentNodeID: 2,
		}); parseSetNodeErr != nil {
			parseB.Fatalf("SetRegionDOMNode(descendant=%d) returned error: %v", getNodeID, parseSetNodeErr)
		}
	}
	buildRootNode, parseRootLookupErr := buildRegionDOMIndex.GetRegionDOMNode("bench-region", 1)
	if parseRootLookupErr != nil {
		parseB.Fatalf("GetRegionDOMNode(root) returned error: %v", parseRootLookupErr)
	}
	buildRootNode.GetChildNodeIDs = []uint64{2, 3}
	buildSubtreeRootNode, parseSubtreeLookupErr := buildRegionDOMIndex.GetRegionDOMNode("bench-region", 2)
	if parseSubtreeLookupErr != nil {
		parseB.Fatalf("GetRegionDOMNode(subtree root) returned error: %v", parseSubtreeLookupErr)
	}
	buildSubtreeRootNode.GetChildNodeIDs = buildSubtreeChildNodeIDs
	return buildDOMCommitStructuralRollbackBenchFixture{
		getDOMCommitter: BuildDOMCommitter(buildRegionDOMIndex),
		getTransaction: RegionPatchTransaction{
			GetRegionID: "bench-region",
			GetOps: []RegionPatchOp{
				{
					GetKind:   RegionPatchOpKindRemoveNode,
					GetNodeID: 2,
				},
				{
					GetKind:   RegionPatchOpKindSetText,
					GetNodeID: 999_999,
					GetText:   "broken",
				},
			},
		},
	}
}

// buildCommitRegionPatchTransactionLegacyStructuralSnapshot preserves the previous rollback path that cloned the full region for any structural op.
func buildCommitRegionPatchTransactionLegacyStructuralSnapshot(
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
			parseDOMCommitter.parseRestoreRegionSnapshot(parseTransaction.GetRegionID, parseRegionSnapshot)
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, parseApplyErr)
		}
	}
	return RegionPatchTransactionResult{}, nil
}

// BenchmarkCommitRegionPatchTransactionStructuralRollbackCurrentVsLegacy compares partial structural rollback snapshots against the previous full-region snapshot path.
func BenchmarkCommitRegionPatchTransactionStructuralRollbackCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("insert_rollback/legacy_full_snapshot", func(parseB *testing.B) {
		getFixture := buildDOMCommitInsertRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := buildCommitRegionPatchTransactionLegacyStructuralSnapshot(getFixture.getDOMCommitter, getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyStructuralSnapshot(insert) error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyStructuralSnapshot(insert) fallback = false, want true")
			}
			storeDOMCommitStructuralRollbackResultSink = getResult
		}
	})
	parseB.Run("insert_rollback/current_partial_snapshot", func(parseB *testing.B) {
		getFixture := buildDOMCommitInsertRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := getFixture.getDOMCommitter.CommitRegionPatchTransaction(getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("CommitRegionPatchTransaction(insert) error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("CommitRegionPatchTransaction(insert) fallback = false, want true")
			}
			storeDOMCommitStructuralRollbackResultSink = getResult
		}
	})
	parseB.Run("move_rollback/legacy_full_snapshot", func(parseB *testing.B) {
		getFixture := buildDOMCommitMoveRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := buildCommitRegionPatchTransactionLegacyStructuralSnapshot(getFixture.getDOMCommitter, getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyStructuralSnapshot(move) error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyStructuralSnapshot(move) fallback = false, want true")
			}
			storeDOMCommitStructuralRollbackResultSink = getResult
		}
	})
	parseB.Run("move_rollback/current_partial_snapshot", func(parseB *testing.B) {
		getFixture := buildDOMCommitMoveRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := getFixture.getDOMCommitter.CommitRegionPatchTransaction(getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("CommitRegionPatchTransaction(move) error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("CommitRegionPatchTransaction(move) fallback = false, want true")
			}
			storeDOMCommitStructuralRollbackResultSink = getResult
		}
	})
	parseB.Run("remove_rollback/legacy_full_snapshot", func(parseB *testing.B) {
		getFixture := buildDOMCommitRemoveRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := buildCommitRegionPatchTransactionLegacyStructuralSnapshot(getFixture.getDOMCommitter, getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyStructuralSnapshot(remove) error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("buildCommitRegionPatchTransactionLegacyStructuralSnapshot(remove) fallback = false, want true")
			}
			storeDOMCommitStructuralRollbackResultSink = getResult
		}
	})
	parseB.Run("remove_rollback/current_partial_snapshot", func(parseB *testing.B) {
		getFixture := buildDOMCommitRemoveRollbackBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getResult, parseCommitErr := getFixture.getDOMCommitter.CommitRegionPatchTransaction(getFixture.getTransaction)
			if parseCommitErr == nil {
				parseB.Fatal("CommitRegionPatchTransaction(remove) error = nil, want rollback error")
			}
			if !getResult.HasFallbackEntered {
				parseB.Fatal("CommitRegionPatchTransaction(remove) fallback = false, want true")
			}
			storeDOMCommitStructuralRollbackResultSink = getResult
		}
	})
}
