package runtime2

import (
	"strconv"
	"testing"
)

// v5 P3.2 — the narrow ReplaceSubtree rollback must be EQUIVALENT to the full
// region clone it replaces, not merely cheaper.
//
// A faster snapshot that restores slightly different state is a corruption bug
// that only shows up after a failed commit, which is exactly when nobody is
// looking. These tests compare the two paths on the same failure.

// buildReplaceRollbackRegion builds a small region with a replaceable subtree.
func buildReplaceRollbackRegion(parseT *testing.T, parseChildCount int) *RegionDOMIndex {
	parseT.Helper()
	parseIndex := BuildRegionDOMIndex()

	if parseErr := parseIndex.SetRegionDOMNode("r", 1, &RegionDOMNode{GetNodeID: 1, GetTag: "ul"}); parseErr != nil {
		parseT.Fatalf("set parent: %v", parseErr)
	}
	parseChildIDs := make([]uint64, 0, parseChildCount)
	for parseOffset := range parseChildCount {
		parseNodeID := uint64(parseOffset + 2)
		parseChildIDs = append(parseChildIDs, parseNodeID)
		if parseErr := parseIndex.SetRegionDOMNode("r", parseNodeID, &RegionDOMNode{
			GetNodeID:       parseNodeID,
			GetTag:          "li",
			GetParentNodeID: 1,
			GetNodeKey:      "row-" + strconv.Itoa(parseOffset),
			GetAttrByKey:    map[string]string{"class": "row"},
		}); parseErr != nil {
			parseT.Fatalf("set child %d: %v", parseNodeID, parseErr)
		}
	}
	parseParent, parseErr := parseIndex.GetRegionDOMNode("r", 1)
	if parseErr != nil {
		parseT.Fatalf("get parent: %v", parseErr)
	}
	parseParent.GetChildNodeIDs = parseChildIDs
	return parseIndex
}

// describeRegion renders a comparable summary of region state, so two rollback
// strategies can be checked for equivalence rather than merely for "no error".
func describeRegion(parseT *testing.T, parseIndex *RegionDOMIndex) map[uint64]string {
	parseT.Helper()
	parseMap := parseIndex.storeRegionDOMNodeByRegionID["r"]
	parseOut := make(map[uint64]string, len(parseMap))
	for parseNodeID, parseNode := range parseMap {
		if parseNode == nil {
			parseOut[parseNodeID] = "<nil>"
			continue
		}
		parseDescription := parseNode.GetTag + "|" + parseNode.GetNodeKey + "|parent=" + strconv.FormatUint(parseNode.GetParentNodeID, 10) + "|children="
		for _, parseChildID := range parseNode.GetChildNodeIDs {
			parseDescription += strconv.FormatUint(parseChildID, 10) + ","
		}
		for parseAttrKey, parseAttrValue := range parseNode.GetAttrByKey {
			parseDescription += "|" + parseAttrKey + "=" + parseAttrValue
		}
		parseOut[parseNodeID] = parseDescription
	}
	return parseOut
}

// TestNarrowReplaceSnapshotCapturesWhatReplaceMutates pins the premise: the
// capture covers the parent's child order, the removed subtree, and the added
// node ids — the three things CommitRegionReplaceSubtree touches.
func TestNarrowReplaceSnapshotCapturesWhatReplaceMutates(parseT *testing.T) {
	parseIndex := buildReplaceRollbackRegion(parseT, 4)
	parseRegionMap := parseIndex.storeRegionDOMNodeByRegionID["r"]

	parseIR, parseErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "new"})
	if parseErr != nil {
		parseT.Fatalf("build IR: %v", parseErr)
	}
	parseSnapshot, isCaptured := captureReplaceSubtreeSnapshot(parseRegionMap, RegionPatchOp{
		GetKind:             RegionPatchOpKindReplaceSubtree,
		GetNodeID:           3,
		GetReplaceSubtreeIR: parseIR,
	})
	if !isCaptured {
		parseT.Fatal("expected a narrow capture for a well-formed replace op")
	}

	if parseSnapshot.parentNodeID != 1 {
		parseT.Errorf("parent = %d, want 1", parseSnapshot.parentNodeID)
	}
	if len(parseSnapshot.parentChildNodeIDs) != 4 {
		parseT.Errorf("captured %d parent children, want 4", len(parseSnapshot.parentChildNodeIDs))
	}
	if _, hasTarget := parseSnapshot.removedNodes[3]; !hasTarget {
		parseT.Error("the replaced node was not captured")
	}
	if len(parseSnapshot.addedNodeIDs) == 0 {
		parseT.Error("no replacement node ids recorded; rollback could not remove them")
	}
}

// TestNarrowReplaceSnapshotDeclinesUnknownTarget: an op the commit will reject
// must fall back to the full path, so error behavior stays identical.
func TestNarrowReplaceSnapshotDeclinesUnknownTarget(parseT *testing.T) {
	parseIndex := buildReplaceRollbackRegion(parseT, 2)
	parseRegionMap := parseIndex.storeRegionDOMNodeByRegionID["r"]

	parseIR, _ := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "new"})
	if _, isCaptured := captureReplaceSubtreeSnapshot(parseRegionMap, RegionPatchOp{
		GetKind:             RegionPatchOpKindReplaceSubtree,
		GetNodeID:           9999,
		GetReplaceSubtreeIR: parseIR,
	}); isCaptured {
		parseT.Error("a missing target must decline the narrow capture and use the full clone")
	}
}

// TestNarrowReplaceSnapshotDeclinesInvalidIR: an unparseable replacement means
// the commit fails before mutating, and the full path preserves that exactly.
func TestNarrowReplaceSnapshotDeclinesInvalidIR(parseT *testing.T) {
	parseIndex := buildReplaceRollbackRegion(parseT, 2)
	parseRegionMap := parseIndex.storeRegionDOMNodeByRegionID["r"]

	if _, isCaptured := captureReplaceSubtreeSnapshot(parseRegionMap, RegionPatchOp{
		GetKind:             RegionPatchOpKindReplaceSubtree,
		GetNodeID:           2,
		GetReplaceSubtreeIR: CanonicalRenderIR{},
	}); isCaptured {
		parseT.Error("an invalid replacement IR must decline the narrow capture")
	}
}

// TestNarrowRollbackMatchesFullCloneRollback is the equivalence test.
//
// The same failing transaction runs twice — once forced onto the full-clone
// path, once on the narrow path — and the resulting region state must be
// identical. This is the test that makes the optimization safe rather than
// merely fast.
func TestNarrowRollbackMatchesFullCloneRollback(parseT *testing.T) {
	parseIR, parseErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "replaced"})
	if parseErr != nil {
		parseT.Fatalf("build IR: %v", parseErr)
	}

	// A transaction whose replace succeeds and whose LAST op fails, so rollback
	// has to undo a completed replacement.
	buildOps := func() []RegionPatchOp {
		return []RegionPatchOp{
			{GetKind: RegionPatchOpKindReplaceSubtree, GetNodeID: 3, GetReplaceSubtreeIR: parseIR},
			{GetKind: RegionPatchOpKindSetAttr, GetNodeID: 2, GetAttrKey: "class", GetAttrValue: "touched"},
			// Fails: no such node. Forces the rollback path.
			{GetKind: RegionPatchOpKindSetAttr, GetNodeID: 424242, GetAttrKey: "class", GetAttrValue: "boom"},
		}
	}

	// Full-clone arm.
	parseFullIndex := buildReplaceRollbackRegion(parseT, 6)
	parseFullCommitter := &DOMCommitter{getRegionDOMIndex: parseFullIndex}
	parseFullBefore := describeRegion(parseT, parseFullIndex)
	parseFullRegionMap := parseFullIndex.storeRegionDOMNodeByRegionID["r"]
	parseFullSnapshot := parseCloneRegionNodeMap(parseFullRegionMap)
	for _, parseOp := range buildOps() {
		if parseErr := parseFullCommitter.parseCommitPatchOp("r", parseOp); parseErr != nil {
			parseFullCommitter.parseRestoreRegionSnapshot("r", parseFullSnapshot)
			break
		}
	}
	parseFullAfter := describeRegion(parseT, parseFullIndex)

	// Narrow arm, through the real transaction entry point.
	parseNarrowIndex := buildReplaceRollbackRegion(parseT, 6)
	parseNarrowCommitter := &DOMCommitter{getRegionDOMIndex: parseNarrowIndex}
	parseNarrowBefore := describeRegion(parseT, parseNarrowIndex)
	if _, parseErr := parseNarrowCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "r",
		GetOps:      buildOps(),
	}); parseErr == nil {
		parseT.Fatal("expected the transaction to fail so rollback runs")
	}
	parseNarrowAfter := describeRegion(parseT, parseNarrowIndex)

	// Both arms must return to their own starting state...
	if len(parseFullAfter) != len(parseFullBefore) {
		parseT.Errorf("full-clone rollback left %d nodes, started with %d", len(parseFullAfter), len(parseFullBefore))
	}
	if len(parseNarrowAfter) != len(parseNarrowBefore) {
		parseT.Errorf("narrow rollback left %d nodes, started with %d", len(parseNarrowAfter), len(parseNarrowBefore))
	}

	// ...and, being the same starting state, must agree node for node.
	for parseNodeID, parseWant := range parseFullAfter {
		parseGot, hasNode := parseNarrowAfter[parseNodeID]
		if !hasNode {
			parseT.Errorf("narrow rollback lost node %d that the full clone restored", parseNodeID)
			continue
		}
		if parseGot != parseWant {
			parseT.Errorf("node %d after rollback:\n narrow = %s\n   full = %s", parseNodeID, parseGot, parseWant)
		}
	}
	for parseNodeID := range parseNarrowAfter {
		if _, hasNode := parseFullAfter[parseNodeID]; !hasNode {
			parseT.Errorf("narrow rollback left node %d that the full clone removed", parseNodeID)
		}
	}
}

// TestNarrowRollbackRestoresParentChildOrder targets the field most likely to
// be missed: the parent's child slice is mutated IN PLACE at one index, so a
// snapshot holding the same backing array would restore nothing.
func TestNarrowRollbackRestoresParentChildOrder(parseT *testing.T) {
	parseIndex := buildReplaceRollbackRegion(parseT, 5)
	parseCommitter := &DOMCommitter{getRegionDOMIndex: parseIndex}

	parseParentBefore, _ := parseIndex.GetRegionDOMNode("r", 1)
	parseOrderBefore := append([]uint64(nil), parseParentBefore.GetChildNodeIDs...)

	parseIR, _ := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "x"})
	if _, parseErr := parseCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "r",
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindReplaceSubtree, GetNodeID: 4, GetReplaceSubtreeIR: parseIR},
			{GetKind: RegionPatchOpKindSetAttr, GetNodeID: 999999, GetAttrKey: "k", GetAttrValue: "v"},
		},
	}); parseErr == nil {
		parseT.Fatal("expected failure so rollback runs")
	}

	parseParentAfter, _ := parseIndex.GetRegionDOMNode("r", 1)
	if len(parseParentAfter.GetChildNodeIDs) != len(parseOrderBefore) {
		parseT.Fatalf("child count = %d, want %d", len(parseParentAfter.GetChildNodeIDs), len(parseOrderBefore))
	}
	for parseOffset, parseWant := range parseOrderBefore {
		if parseParentAfter.GetChildNodeIDs[parseOffset] != parseWant {
			parseT.Errorf("child[%d] = %d, want %d — parent child order was not restored",
				parseOffset, parseParentAfter.GetChildNodeIDs[parseOffset], parseWant)
		}
	}
}

// TestNarrowPathSucceedsWithoutRollback: the happy path must be unchanged.
func TestNarrowPathSucceedsWithoutRollback(parseT *testing.T) {
	parseIndex := buildReplaceRollbackRegion(parseT, 4)
	parseCommitter := &DOMCommitter{getRegionDOMIndex: parseIndex}

	parseIR, _ := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "replaced"})
	if _, parseErr := parseCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "r",
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindReplaceSubtree, GetNodeID: 3, GetReplaceSubtreeIR: parseIR},
		},
	}); parseErr != nil {
		parseT.Fatalf("successful replace returned error: %v", parseErr)
	}

	if _, hasOldNode := parseIndex.storeRegionDOMNodeByRegionID["r"][3]; hasOldNode {
		parseT.Error("the replaced node should be gone after a successful commit")
	}
}
