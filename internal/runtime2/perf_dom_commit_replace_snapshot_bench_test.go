package runtime2

import (
	"strconv"
	"testing"
)

// v5 P3.2 — measure the snapshot path that was never measured (R3).
//
// The plan called this "the mixed-ops path", which turned out to be the wrong
// name for it. Reading hasCommitRegionSnapshotOpKind: exactly ONE op kind,
// RegionPatchOpKindReplaceSubtree, forces the full parseCloneRegionNodeMap.
// Everything else already takes the incremental parseCaptureCommitNodeSnapshot
// path that the append benchmark covers.
//
// So the cost is not "transactions with a mix of ops". It is that a SINGLE
// ReplaceSubtree op makes the whole transaction clone the entire region node
// map, however small the rest of the transaction is and however large the
// region. That is the shape worth measuring: cost driven by region size rather
// than by the size of the change.

var storeDOMCommitReplaceSnapshotResultSink RegionPatchTransactionResult

// buildReplaceSnapshotFixture builds a region of parseRegionSize nodes and a
// transaction of parseOpCount ops. When isWithReplace is set, exactly one op is
// a ReplaceSubtree — the difference the benchmark isolates.
func buildReplaceSnapshotFixture(parseB *testing.B, parseRegionSize int, parseOpCount int, isWithReplace bool) (*DOMCommitter, RegionPatchTransaction) {
	parseB.Helper()

	buildRegionDOMIndex := BuildRegionDOMIndex()
	buildChildNodeIDs := make([]uint64, 0, parseRegionSize)
	if parseErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", 1, &RegionDOMNode{
		GetNodeID: 1,
		GetTag:    "ul",
	}); parseErr != nil {
		parseB.Fatalf("SetRegionDOMNode(parent): %v", parseErr)
	}
	for parseOffset := range parseRegionSize {
		getNodeID := uint64(parseOffset + 2)
		buildChildNodeIDs = append(buildChildNodeIDs, getNodeID)
		if parseErr := buildRegionDOMIndex.SetRegionDOMNode("bench-region", getNodeID, &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row-" + strconv.Itoa(parseOffset),
			GetAttrByKey:    map[string]string{"class": "row"},
		}); parseErr != nil {
			parseB.Fatalf("SetRegionDOMNode(child=%d): %v", getNodeID, parseErr)
		}
	}
	buildParentNode, parseErr := buildRegionDOMIndex.GetRegionDOMNode("bench-region", 1)
	if parseErr != nil {
		parseB.Fatalf("GetRegionDOMNode(parent): %v", parseErr)
	}
	buildParentNode.GetChildNodeIDs = buildChildNodeIDs

	buildOps := make([]RegionPatchOp, 0, parseOpCount)
	for parseOffset := range parseOpCount {
		buildOps = append(buildOps, RegionPatchOp{
			GetKind:      RegionPatchOpKindSetAttr,
			GetNodeID:    uint64(parseOffset%parseRegionSize + 2),
			GetAttrKey:   "class",
			GetAttrValue: "row-updated",
		})
	}
	if isWithReplace {
		// One replacement among otherwise cheap attribute writes. This single
		// op is what flips the whole transaction onto the full-clone path.
		//
		// The replacement subtree is deliberately TINY — a single text node —
		// so the arms differ only in which snapshot strategy runs, not in how
		// much DOM work each does. A large replacement would confound the
		// measurement with its own cost.
		parseReplacementIR, parseIRErr := BuildCanonicalRenderIR(map[string]any{
			"kind": "text",
			"text": "replaced",
		})
		if parseIRErr != nil {
			parseB.Fatalf("BuildCanonicalRenderIR: %v", parseIRErr)
		}
		buildOps[0] = RegionPatchOp{
			GetKind:             RegionPatchOpKindReplaceSubtree,
			GetNodeID:           2,
			GetReplaceSubtreeIR: parseReplacementIR,
		}
	}

	return &DOMCommitter{getRegionDOMIndex: buildRegionDOMIndex}, RegionPatchTransaction{
		GetRegionID: "bench-region",
		GetOps:      buildOps,
	}
}

// BenchmarkCommitRegionPatchTransactionReplaceSubtreeSnapshot shows the cost
// driven by REGION size rather than change size.
//
// Same eight-op transaction throughout; the only difference is whether one of
// those ops is a ReplaceSubtree. The with_replace arm should scale with the
// region while the without arm stays flat — and that gap is the whole finding.
func BenchmarkCommitRegionPatchTransactionReplaceSubtreeSnapshot(parseB *testing.B) {
	for _, parseRegionSize := range []int{64, 256, 1024} {
		parseLabel := "region-" + strconv.Itoa(parseRegionSize)

		parseB.Run(parseLabel+"/without_replace", func(parseInner *testing.B) {
			parseInner.ReportAllocs()
			for parseInner.Loop() {
				parseCommitter, parseTransaction := buildReplaceSnapshotFixture(parseInner, parseRegionSize, 8, false)
				getResult, parseErr := parseCommitter.CommitRegionPatchTransaction(parseTransaction)
				if parseErr != nil {
					parseInner.Fatalf("commit: %v", parseErr)
				}
				storeDOMCommitReplaceSnapshotResultSink = getResult
			}
		})

		parseB.Run(parseLabel+"/with_one_replace", func(parseInner *testing.B) {
			parseInner.ReportAllocs()
			for parseInner.Loop() {
				parseCommitter, parseTransaction := buildReplaceSnapshotFixture(parseInner, parseRegionSize, 8, true)
				getResult, parseErr := parseCommitter.CommitRegionPatchTransaction(parseTransaction)
				if parseErr != nil {
					parseInner.Fatalf("commit: %v", parseErr)
				}
				storeDOMCommitReplaceSnapshotResultSink = getResult
			}
		})
	}
}

// TestReplaceSubtreeIsTheOnlyFullSnapshotTrigger pins the finding this
// benchmark rests on. If another op kind is ever added to the full-clone set,
// the benchmark above stops describing the cost and this test says so.
func TestReplaceSubtreeIsTheOnlyFullSnapshotTrigger(parseT *testing.T) {
	parseAllKinds := []RegionPatchOpKind{
		RegionPatchOpKindInsertNode,
		RegionPatchOpKindRemoveNode,
		RegionPatchOpKindSetAttr,
		RegionPatchOpKindRemoveAttr,
		RegionPatchOpKindSetText,
		RegionPatchOpKindSetStyle,
		RegionPatchOpKindRemoveStyle,
		RegionPatchOpKindMoveKeyedNode,
		RegionPatchOpKindReplaceSubtree,
	}

	for _, parseKind := range parseAllKinds {
		getTriggers := hasCommitRegionSnapshotOpKind(parseKind)
		isReplace := parseKind == RegionPatchOpKindReplaceSubtree
		if getTriggers != isReplace {
			parseT.Errorf("op kind %v triggers full snapshot = %t, want %t — "+
				"the full-clone set changed, so P3.2's benchmark no longer describes the cost",
				parseKind, getTriggers, isReplace)
		}
	}
}
