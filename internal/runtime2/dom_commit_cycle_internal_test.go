package runtime2

import "testing"

// TestParseDeleteNodeIDsTerminatesOnCyclicGraph pins that the region-node
// delete walk cannot be driven into an unbounded recursion (a fatal worker
// stack overflow) by a malformed cyclic child graph — e.g. a replace-subtree
// op carrying two nodes that reference each other as children, which the
// canonical-tree parser does not currently reject. Deleting the node before
// recursing makes the map's own membership the visited-set.
func TestParseDeleteNodeIDsTerminatesOnCyclicGraph(parseT *testing.T) {
	parseNodeMap := map[uint64]*RegionDOMNode{
		100: {GetNodeID: 100, GetChildNodeIDs: []uint64{200}},
		200: {GetNodeID: 200, GetChildNodeIDs: []uint64{100}},
	}
	// Must return (not stack-overflow) and remove the whole reachable set.
	parseDeleteNodeIDs(parseNodeMap, 100)
	if len(parseNodeMap) != 0 {
		parseT.Fatalf("expected cyclic subtree fully deleted, %d nodes remain", len(parseNodeMap))
	}
}

// TestHasCommitRemoveRootAncestorTerminatesOnCyclicParents pins the bounded
// parent walk against a malformed parent-pointer cycle.
func TestHasCommitRemoveRootAncestorTerminatesOnCyclicParents(parseT *testing.T) {
	parseNodeMap := map[uint64]*RegionDOMNode{
		1: {GetNodeID: 1, GetParentNodeID: 2},
		2: {GetNodeID: 2, GetParentNodeID: 1},
	}
	// No node is a remove-root, so this must terminate and return false rather
	// than looping forever on the 1<->2 parent cycle.
	if hasCommitRemoveRootAncestor(map[uint64]struct{}{}, parseNodeMap, 1) {
		parseT.Fatal("expected false for a cyclic parent chain with no remove-root")
	}
}
