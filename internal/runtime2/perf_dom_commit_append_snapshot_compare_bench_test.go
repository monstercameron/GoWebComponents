package runtime2

import (
	"strconv"
	"testing"
)

var storeDOMCommitAppendSnapshotResultSink RegionPatchTransactionResult

// buildDOMCommitAppendSnapshotBenchFixture stores one append-heavy transaction benchmark fixture for current-vs-legacy structural snapshot comparisons.
type buildDOMCommitAppendSnapshotBenchFixture struct {
	getDOMCommitter *DOMCommitter
	getTransaction  RegionPatchTransaction
}

// buildDOMCommitAppendSnapshotBenchmarkFixture builds one append-heavy successful transaction over one keyed sibling list.
func buildDOMCommitAppendSnapshotBenchmarkFixture(parseB *testing.B, parseExistingChildCount int, parseAppendCount int) buildDOMCommitAppendSnapshotBenchFixture {
	parseB.Helper()
	if parseExistingChildCount < 1 {
		parseExistingChildCount = 1
	}
	if parseAppendCount < 1 {
		parseAppendCount = 1
	}
	buildRegionDOMIndex := BuildRegionDOMIndex()
	buildChildNodeIDs := make([]uint64, 0, parseExistingChildCount+parseAppendCount)
	if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: buildChildNodeIDs,
	}); parseSetNodeErr != nil {
		parseB.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetNodeErr)
	}
	for parseChildOffset := 0; parseChildOffset < parseExistingChildCount; parseChildOffset++ {
		getNodeID := uint64(parseChildOffset + 2)
		buildChildNodeIDs = append(buildChildNodeIDs, getNodeID)
		if parseSetNodeErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row-" + strconv.Itoa(parseChildOffset),
		}); parseSetNodeErr != nil {
			parseB.Fatalf("SetRegionDOMNode(child=%d) returned error: %v", getNodeID, parseSetNodeErr)
		}
	}
	buildParentNode, parseLookupErr := buildRegionDOMIndex.GetRegionDOMNode("bench-region", 1)
	if parseLookupErr != nil {
		parseB.Fatalf("GetRegionDOMNode(parent) returned error: %v", parseLookupErr)
	}
	buildParentNode.GetChildNodeIDs = buildChildNodeIDs
	buildOps := make([]RegionPatchOp, 0, parseAppendCount)
	for parseAppendOffset := 0; parseAppendOffset < parseAppendCount; parseAppendOffset++ {
		getNodeID := uint64(parseExistingChildCount + parseAppendOffset + 2)
		buildOps = append(buildOps, RegionPatchOp{
			GetKind:         RegionPatchOpKindInsertNode,
			GetParentNodeID: 1,
			GetInsertNode: &RegionDOMNode{
				GetNodeID: getNodeID,
				GetTag:    "li",
				GetNodeKey: "row-" + strconv.Itoa(parseExistingChildCount+parseAppendOffset),
			},
		})
	}
	return buildDOMCommitAppendSnapshotBenchFixture{
		getDOMCommitter: BuildDOMCommitter(buildRegionDOMIndex),
		getTransaction: RegionPatchTransaction{
			GetRegionID: "bench-region",
			GetOps:      buildOps,
		},
	}
}

// buildCommitRegionPatchTransactionLegacyAppendSnapshotClone preserves the previous structural snapshot path that recloned the parent child-order slice on every repeated insert capture.
func buildCommitRegionPatchTransactionLegacyAppendSnapshotClone(
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
		switch parseOp.GetKind {
		case RegionPatchOpKindInsertNode:
			parseNodeSnapshotByID = buildCommitLegacyStructuralNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp.GetParentNodeID)
			if parseOp.GetInsertNode != nil {
				parseNodeSnapshotByID = storeCommitMissingNodeSnapshot(parseNodeSnapshotByID, parseOp.GetInsertNode.GetNodeID)
			}
		default:
			parseNodeSnapshotByID = parseCaptureCommitNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp)
		}
		if parseApplyErr := parseDOMCommitter.parseCommitPatchOp(parseTransaction.GetRegionID, parseOp); parseApplyErr != nil {
			parseRestoreCommitNodeSnapshot(parseRegionNodeMap, parseNodeSnapshotByID)
			if parseRegionNodeMap != nil && len(parseRegionNodeMap) > 0 {
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

// buildCommitLegacyStructuralNodeSnapshot preserves the previous structural snapshot policy that refreshed one existing parent snapshot from the latest partial child order.
func buildCommitLegacyStructuralNodeSnapshot(
	parseNodeSnapshotByID map[uint64]*RegionDOMNode,
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseNodeID uint64,
) map[uint64]*RegionDOMNode {
	if parseNodeID == 0 || parseRegionNodeMap == nil {
		return parseNodeSnapshotByID
	}
	parseRegionDOMNode, hasRegionDOMNode := parseRegionNodeMap[parseNodeID]
	if !hasRegionDOMNode || parseRegionDOMNode == nil {
		return parseNodeSnapshotByID
	}
	if parseNodeSnapshotByID == nil {
		parseNodeSnapshotByID = make(map[uint64]*RegionDOMNode, 4)
	}
	parseNodeSnapshot, hasNodeSnapshot := parseNodeSnapshotByID[parseNodeID]
	if !hasNodeSnapshot {
		parseNodeSnapshotByID[parseNodeID] = parseCloneRegionDOMNode(parseRegionDOMNode)
		return parseNodeSnapshotByID
	}
	if parseNodeSnapshot == nil {
		return parseNodeSnapshotByID
	}
	parseNodeSnapshot.GetChildNodeIDs = append([]uint64(nil), parseRegionDOMNode.GetChildNodeIDs...)
	return parseNodeSnapshotByID
}

// BenchmarkCommitRegionPatchTransactionAppendSnapshotCurrentVsLegacy compares current structural snapshot capture against the previous repeated append-time child-slice cloning path.
func BenchmarkCommitRegionPatchTransactionAppendSnapshotCurrentVsLegacy(parseB *testing.B) {
	parseAppendCounts := []int{32, 128, 256}
	for _, parseAppendCount := range parseAppendCounts {
		parseAppendCount := parseAppendCount
		parseB.Run("append-"+strconv.Itoa(parseAppendCount)+"/legacy_reclone", func(parseB *testing.B) {
			parseB.ReportAllocs()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getFixture := buildDOMCommitAppendSnapshotBenchmarkFixture(parseB, 512, parseAppendCount)
				getResult, parseCommitErr := buildCommitRegionPatchTransactionLegacyAppendSnapshotClone(getFixture.getDOMCommitter, getFixture.getTransaction)
				if parseCommitErr != nil {
					parseB.Fatalf("buildCommitRegionPatchTransactionLegacyAppendSnapshotClone returned error: %v", parseCommitErr)
				}
				storeDOMCommitAppendSnapshotResultSink = getResult
			}
		})
		parseB.Run("append-"+strconv.Itoa(parseAppendCount)+"/current_capture_once", func(parseB *testing.B) {
			parseB.ReportAllocs()
			for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
				getFixture := buildDOMCommitAppendSnapshotBenchmarkFixture(parseB, 512, parseAppendCount)
				getResult, parseCommitErr := getFixture.getDOMCommitter.CommitRegionPatchTransaction(getFixture.getTransaction)
				if parseCommitErr != nil {
					parseB.Fatalf("CommitRegionPatchTransaction returned error: %v", parseCommitErr)
				}
				storeDOMCommitAppendSnapshotResultSink = getResult
			}
		})
	}
}
