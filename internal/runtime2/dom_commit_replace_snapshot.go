package runtime2

// Targeted rollback snapshot for ReplaceSubtree (v5 P3.2).
//
// A ReplaceSubtree op used to force parseCloneRegionNodeMap — a deep copy of
// the ENTIRE region node map — because the generic rollback path had no way to
// describe what a replacement touched. Measured cost was O(region) rather than
// O(change): roughly 475 bytes per region node, so replacing one text node in a
// 1024-node region cost ~486 KB and ~523 µs of pure rollback overhead.
//
// It only ever mutates three things (see CommitRegionReplaceSubtree):
//
//  1. the parent's GetChildNodeIDs slice, at one index
//  2. the target subtree's nodes, deleted from the region map
//  3. the replacement nodes, added to the region map
//
// All three are bounded by the change. Capturing exactly those makes rollback
// proportional to what was replaced instead of to how large the region happens
// to be.

// replaceSubtreeSnapshot records the minimum needed to undo one ReplaceSubtree.
type replaceSubtreeSnapshot struct {
	// parentNodeID is 0 when the target was the region root, in which case
	// there is no parent child-order to restore.
	parentNodeID uint64
	// parentChildNodeIDs is a copy of the parent's child order taken BEFORE the
	// replacement, since the commit overwrites one entry in place.
	parentChildNodeIDs []uint64
	// removedNodes holds the target subtree exactly as it was.
	removedNodes map[uint64]*RegionDOMNode
	// addedNodeIDs are the replacement nodes to delete on rollback. Recorded
	// rather than recomputed: the replacement IR is parsed during the commit,
	// and re-deriving the id set at rollback time could disagree with what was
	// actually inserted.
	addedNodeIDs []uint64
}

// captureReplaceSubtreeSnapshot records the pre-mutation state a ReplaceSubtree
// will disturb. Returns false when the op cannot be snapshotted narrowly, in
// which case the caller must fall back to the full region clone.
func captureReplaceSubtreeSnapshot(
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseOp RegionPatchOp,
) (replaceSubtreeSnapshot, bool) {
	if parseRegionNodeMap == nil || parseOp.GetNodeID == 0 {
		return replaceSubtreeSnapshot{}, false
	}
	parseTargetNode, hasTargetNode := parseRegionNodeMap[parseOp.GetNodeID]
	if !hasTargetNode || parseTargetNode == nil {
		// The commit will fail on this anyway; let the full path handle it so
		// error behavior is unchanged.
		return replaceSubtreeSnapshot{}, false
	}

	parseSnapshot := replaceSubtreeSnapshot{
		parentNodeID: parseTargetNode.GetParentNodeID,
		removedNodes: make(map[uint64]*RegionDOMNode, 8),
	}

	if parseSnapshot.parentNodeID != 0 {
		parseParentNode, hasParentNode := parseRegionNodeMap[parseSnapshot.parentNodeID]
		if !hasParentNode || parseParentNode == nil {
			return replaceSubtreeSnapshot{}, false
		}
		parseSnapshot.parentChildNodeIDs = append([]uint64(nil), parseParentNode.GetChildNodeIDs...)
	}

	collectReplaceSubtreeNodes(parseRegionNodeMap, parseOp.GetNodeID, parseSnapshot.removedNodes)

	// The replacement's node ids come from its IR. Deriving them here, before
	// the commit runs, keeps capture and rollback describing the same set.
	if parseTree, parseErr := ParseCanonicalRenderTree(parseOp.GetReplaceSubtreeIR); parseErr == nil {
		for parseNodeID := range parseBuildRegionDOMNodeByNodeIDFromCanonicalTree(parseTree) {
			parseSnapshot.addedNodeIDs = append(parseSnapshot.addedNodeIDs, parseNodeID)
		}
	} else {
		// An unparseable IR means the commit will fail before mutating; the
		// full path keeps that behavior identical.
		return replaceSubtreeSnapshot{}, false
	}

	return parseSnapshot, true
}

// collectReplaceSubtreeNodes deep-copies one subtree, mirroring the traversal
// parseDeleteNodeIDs uses so capture and deletion cover the same nodes.
func collectReplaceSubtreeNodes(
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseNodeID uint64,
	parseInto map[uint64]*RegionDOMNode,
) {
	parseNode, hasNode := parseRegionNodeMap[parseNodeID]
	if !hasNode || parseNode == nil {
		return
	}
	parseInto[parseNodeID] = parseCloneRegionDOMNode(parseNode)
	for _, parseChildNodeID := range parseNode.GetChildNodeIDs {
		collectReplaceSubtreeNodes(parseRegionNodeMap, parseChildNodeID, parseInto)
	}
}

// restoreReplaceSubtreeSnapshot undoes one ReplaceSubtree.
//
// Order matters: added nodes are removed first, then the original subtree is
// restored, then the parent's child order. Restoring the parent last means its
// child ids always point at nodes that exist again.
func restoreReplaceSubtreeSnapshot(
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseSnapshot replaceSubtreeSnapshot,
) {
	if parseRegionNodeMap == nil {
		return
	}
	for _, parseNodeID := range parseSnapshot.addedNodeIDs {
		delete(parseRegionNodeMap, parseNodeID)
	}
	for parseNodeID, parseNode := range parseSnapshot.removedNodes {
		parseRegionNodeMap[parseNodeID] = parseNode
	}
	if parseSnapshot.parentNodeID != 0 && parseSnapshot.parentChildNodeIDs != nil {
		if parseParentNode, hasParentNode := parseRegionNodeMap[parseSnapshot.parentNodeID]; hasParentNode && parseParentNode != nil {
			parseParentNode.GetChildNodeIDs = parseSnapshot.parentChildNodeIDs
		}
	}
}
