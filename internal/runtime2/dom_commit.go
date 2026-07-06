package runtime2

import (
	"fmt"
	"maps"
	"strings"
)

const (
	getDOMCommitTransactionOpWarnLimit = 2048
	getDOMCommitTransactionOpHardLimit = 16384
	getDOMCommitSnapshotNodeWarnLimit  = 8192
	getDOMCommitSnapshotNodeHardLimit  = 65536
)

// DOMCommitResult reports whether a commit operation changed region-local DOM state.
type DOMCommitResult struct {
	IsNoOp bool
}

// DOMCommitter applies patch operations against a region-local DOM index.
type DOMCommitter struct {
	getRegionDOMIndex        *RegionDOMIndex
	isCommitMutationBatching bool
	hasCommitMutationPending bool
}

// RegionPatchOpKind identifies one supported commit operation kind in a transaction.
type RegionPatchOpKind string

const (
	regionPatchOpKindInvalid RegionPatchOpKind = ""

	RegionPatchOpKindSetText        RegionPatchOpKind = "set_text"
	RegionPatchOpKindSetAttr        RegionPatchOpKind = "set_attr"
	RegionPatchOpKindSetStyle       RegionPatchOpKind = "set_style"
	RegionPatchOpKindRemoveAttr     RegionPatchOpKind = "remove_attr"
	RegionPatchOpKindRemoveStyle    RegionPatchOpKind = "remove_style"
	RegionPatchOpKindInsertNode     RegionPatchOpKind = "insert_node"
	RegionPatchOpKindRemoveNode     RegionPatchOpKind = "remove_node"
	RegionPatchOpKindReplaceSubtree RegionPatchOpKind = "replace_subtree"
	RegionPatchOpKindMoveKeyedNode  RegionPatchOpKind = "move_keyed_node"
)

// RegionPatchOp stores one commit operation payload inside a transaction.
type RegionPatchOp struct {
	GetKind RegionPatchOpKind

	GetNodeID uint64
	GetText   string

	GetAttrKey    string
	GetAttrValue  string
	GetStyleValue string

	GetParentNodeID     uint64
	GetInsertNode       *RegionDOMNode
	GetBeforeNodeID     uint64
	GetMoveNodeID       uint64
	GetDestinationIndex uint32
	GetReplaceSubtreeIR CanonicalRenderIR
}

// RegionPatchTransaction stores one ordered set of patch ops for a single region.
type RegionPatchTransaction struct {
	GetRegionID string
	GetOps      []RegionPatchOp
}

// RegionPatchTransactionResult reports transaction-level commit outcomes.
type RegionPatchTransactionResult struct {
	HasFallbackEntered bool
}

// BuildDOMCommitter creates a DOM committer for the provided region-local DOM index.
func BuildDOMCommitter(parseRegionDOMIndex *RegionDOMIndex) *DOMCommitter {
	if parseRegionDOMIndex == nil {
		parseRegionDOMIndex = BuildRegionDOMIndex()
	}
	return &DOMCommitter{
		getRegionDOMIndex: parseRegionDOMIndex,
	}
}

// beginCommitMutationBatch starts one deferred mutation-version batch for transaction-scoped commit work.
func (parseDOMCommitter *DOMCommitter) beginCommitMutationBatch() {
	if parseDOMCommitter == nil {
		return
	}
	parseDOMCommitter.isCommitMutationBatching = true
	parseDOMCommitter.hasCommitMutationPending = false
}

// finishCommitMutationBatch applies one deferred mutation-version bump after a successful transaction-scoped commit batch.
func (parseDOMCommitter *DOMCommitter) finishCommitMutationBatch(parseRegionID string, isCommitSuccessful bool) {
	if parseDOMCommitter == nil {
		return
	}
	hasCommitMutationPending := parseDOMCommitter.hasCommitMutationPending
	parseDOMCommitter.isCommitMutationBatching = false
	parseDOMCommitter.hasCommitMutationPending = false
	if !isCommitSuccessful || !hasCommitMutationPending {
		return
	}
	parseDOMCommitter.getRegionDOMIndex.storeRegionDOMMutationVersion(parseRegionID)
}

// storeCommitMutationVersion records one region mutation-version change, deferring the bump while a transaction batch is active.
func (parseDOMCommitter *DOMCommitter) storeCommitMutationVersion(parseRegionID string) {
	if parseDOMCommitter == nil || parseDOMCommitter.getRegionDOMIndex == nil {
		return
	}
	if parseDOMCommitter.isCommitMutationBatching {
		parseDOMCommitter.hasCommitMutationPending = true
		return
	}
	parseDOMCommitter.getRegionDOMIndex.storeRegionDOMMutationVersion(parseRegionID)
}

// storeCommitRegionDOMNode inserts or replaces one region-local DOM node entry, deferring the mutation-version bump while a transaction batch is active.
func (parseDOMCommitter *DOMCommitter) storeCommitRegionDOMNode(parseRegionID string, parseNodeID uint64, parseRegionDOMNode *RegionDOMNode) error {
	if parseDOMCommitter == nil || parseDOMCommitter.getRegionDOMIndex == nil {
		return fmt.Errorf("runtime2: dom committer is nil")
	}
	if parseDOMCommitter.isCommitMutationBatching {
		parseDOMCommitter.hasCommitMutationPending = true
	}
	return parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNode(parseRegionID, parseNodeID, parseRegionDOMNode, !parseDOMCommitter.isCommitMutationBatching)
}

// CommitRegionText applies one set-text operation to a region-local node.
func (parseDOMCommitter *DOMCommitter) CommitRegionText(parseRegionID string, parseNodeID uint64, parseText string) (DOMCommitResult, error) {
	if parseDOMCommitter == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	parseRegionDOMNode, parseLookupErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseNodeID)
	if parseLookupErr != nil {
		return DOMCommitResult{}, parseLookupErr
	}
	if parseRegionDOMNode.GetText == parseText {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	parseRegionDOMNode.GetText = parseText
	parseDOMCommitter.storeCommitMutationVersion(parseRegionID)
	return DOMCommitResult{}, nil
}

// CommitRegionAttr applies one set-attr operation to a region-local host node for supported first-slice attr kinds.
func (parseDOMCommitter *DOMCommitter) CommitRegionAttr(parseRegionID string, parseNodeID uint64, parseAttrKey string, parseAttrValue string) (DOMCommitResult, error) {
	if parseDOMCommitter == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	if strings.TrimSpace(parseAttrKey) == "" {
		return DOMCommitResult{}, fmt.Errorf("runtime2: attr key is required")
	}
	if parseAttrKey != "class" &&
		parseAttrKey != "style" &&
		!strings.HasPrefix(parseAttrKey, "aria-") &&
		!strings.HasPrefix(parseAttrKey, "data-") {
		return DOMCommitResult{}, fmt.Errorf("runtime2: unsupported attr kind %q", parseAttrKey)
	}
	parseRegionDOMNode, parseLookupErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseNodeID)
	if parseLookupErr != nil {
		return DOMCommitResult{}, parseLookupErr
	}
	if parseRegionDOMNode.GetAttrByKey == nil {
		parseRegionDOMNode.GetAttrByKey = make(map[string]string)
	}
	if parseRegionDOMNode.GetAttrByKey[parseAttrKey] == parseAttrValue {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	parseRegionDOMNode.GetAttrByKey[parseAttrKey] = parseAttrValue
	parseDOMCommitter.storeCommitMutationVersion(parseRegionID)
	return DOMCommitResult{}, nil
}

// CommitRegionRemoveAttr applies one remove-attr operation to a region-local host node.
func (parseDOMCommitter *DOMCommitter) CommitRegionRemoveAttr(parseRegionID string, parseNodeID uint64, parseAttrKey string) (DOMCommitResult, error) {
	if parseDOMCommitter == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	if strings.TrimSpace(parseAttrKey) == "" {
		return DOMCommitResult{}, fmt.Errorf("runtime2: attr key is required")
	}
	parseRegionDOMNode, parseLookupErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseNodeID)
	if parseLookupErr != nil {
		return DOMCommitResult{}, parseLookupErr
	}
	if parseRegionDOMNode.GetAttrByKey == nil {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	if _, hasAttr := parseRegionDOMNode.GetAttrByKey[parseAttrKey]; !hasAttr {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	delete(parseRegionDOMNode.GetAttrByKey, parseAttrKey)
	parseDOMCommitter.storeCommitMutationVersion(parseRegionID)
	return DOMCommitResult{}, nil
}

// CommitRegionSetStyle applies one set-style operation to a region-local host node.
func (parseDOMCommitter *DOMCommitter) CommitRegionSetStyle(parseRegionID string, parseNodeID uint64, parseStyleValue string) (DOMCommitResult, error) {
	return parseDOMCommitter.CommitRegionAttr(parseRegionID, parseNodeID, "style", parseStyleValue)
}

// CommitRegionRemoveStyle applies one remove-style operation to a region-local host node.
func (parseDOMCommitter *DOMCommitter) CommitRegionRemoveStyle(parseRegionID string, parseNodeID uint64) (DOMCommitResult, error) {
	return parseDOMCommitter.CommitRegionRemoveAttr(parseRegionID, parseNodeID, "style")
}

// CommitRegionInsertNode inserts one region-local node under a parent, optionally before a sibling anchor.
func (parseDOMCommitter *DOMCommitter) CommitRegionInsertNode(parseRegionID string, parseParentNodeID uint64, parseInsertNode *RegionDOMNode, parseBeforeNodeID uint64) (DOMCommitResult, error) {
	if parseDOMCommitter == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	if parseInsertNode == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: insert node is required")
	}
	if parseInsertNode.GetNodeID == 0 {
		return DOMCommitResult{}, fmt.Errorf("runtime2: insert node ID is required")
	}
	parseParentNode, parseParentLookupErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseParentNodeID)
	if parseParentLookupErr != nil {
		return DOMCommitResult{}, parseParentLookupErr
	}
	if _, parseExistingNodeErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseInsertNode.GetNodeID); parseExistingNodeErr == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: insert node ID %d already exists in region %q", parseInsertNode.GetNodeID, parseRegionID)
	}
	parseInsertChildIndex, parseInsertChildErr := parseFindInsertChildIndex(parseParentNode.GetChildNodeIDs, parseBeforeNodeID)
	if parseInsertChildErr != nil {
		return DOMCommitResult{}, parseInsertChildErr
	}
	parseInsertNode.GetParentNodeID = parseParentNodeID
	parseSetNodeErr := parseDOMCommitter.storeCommitRegionDOMNode(parseRegionID, parseInsertNode.GetNodeID, parseInsertNode)
	if parseSetNodeErr != nil {
		return DOMCommitResult{}, parseSetNodeErr
	}
	parseParentNode.GetChildNodeIDs = parseInsertChildNodeID(parseParentNode.GetChildNodeIDs, parseInsertNode.GetNodeID, parseInsertChildIndex)
	return DOMCommitResult{}, nil
}

// parseFindInsertChildIndex resolves one optional sibling anchor into one child insertion index.
func parseFindInsertChildIndex(parseChildNodeIDs []uint64, parseBeforeNodeID uint64) (int, error) {
	if parseBeforeNodeID == 0 {
		return len(parseChildNodeIDs), nil
	}
	parseAnchorIndex := parseFindChildNodeIndex(parseChildNodeIDs, parseBeforeNodeID)
	if parseAnchorIndex < 0 {
		return -1, fmt.Errorf("runtime2: invalid sibling anchor %d", parseBeforeNodeID)
	}
	return parseAnchorIndex, nil
}

// parseInsertChildNodeIDs returns a child-node order with one inserted node, optionally before an anchor.
func parseInsertChildNodeIDs(parseChildNodeIDs []uint64, parseInsertNodeID uint64, parseBeforeNodeID uint64) ([]uint64, error) {
	parseInsertChildIndex, parseInsertChildErr := parseFindInsertChildIndex(parseChildNodeIDs, parseBeforeNodeID)
	if parseInsertChildErr != nil {
		return nil, parseInsertChildErr
	}
	parseNextChildNodeIDs := append([]uint64(nil), parseChildNodeIDs...)
	return parseInsertChildNodeID(parseNextChildNodeIDs, parseInsertNodeID, parseInsertChildIndex), nil
}

// parseInsertChildNodeID inserts one child ID at the provided index, reusing slice capacity when available.
func parseInsertChildNodeID(parseChildNodeIDs []uint64, parseInsertNodeID uint64, parseInsertChildIndex int) []uint64 {
	parseChildNodeIDs = append(parseChildNodeIDs, 0)
	copy(parseChildNodeIDs[parseInsertChildIndex+1:], parseChildNodeIDs[parseInsertChildIndex:])
	parseChildNodeIDs[parseInsertChildIndex] = parseInsertNodeID
	return parseChildNodeIDs
}

// CommitRegionRemoveNode removes one region-local node and any indexed descendants from the DOM index.
func (parseDOMCommitter *DOMCommitter) CommitRegionRemoveNode(parseRegionID string, parseNodeID uint64) (DOMCommitResult, error) {
	if parseDOMCommitter == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	if parseNodeID == 0 {
		return DOMCommitResult{}, fmt.Errorf("runtime2: node ID is required")
	}
	getRegionDOMNodeByNodeID, hasRegionDOMNodeByNodeID := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionDOMNodeByNodeID {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	parseRegionDOMNode, hasRegionDOMNode := getRegionDOMNodeByNodeID[parseNodeID]
	if !hasRegionDOMNode {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	if parseRegionDOMNode.GetParentNodeID != 0 {
		parseParentNode, hasParentNode := getRegionDOMNodeByNodeID[parseRegionDOMNode.GetParentNodeID]
		if hasParentNode {
			parseParentNode.GetChildNodeIDs = parseFilterChildNodeIDs(parseParentNode.GetChildNodeIDs, parseNodeID)
		}
	}
	parseDeleteNodeIDs(getRegionDOMNodeByNodeID, parseNodeID)
	if len(getRegionDOMNodeByNodeID) == 0 {
		delete(parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID, parseRegionID)
		delete(parseDOMCommitter.getRegionDOMIndex.storeRegionDOMVersionByRegionID, parseRegionID)
		return DOMCommitResult{}, nil
	}
	parseDOMCommitter.storeCommitMutationVersion(parseRegionID)
	return DOMCommitResult{}, nil
}

// parseDeleteNodeIDs deletes one node subtree from a region-local node map.
// The node is removed BEFORE recursing into its children, so the map's own
// membership acts as a visited-set: a malformed cyclic child graph (e.g. a
// replace-subtree op carrying nodes that reference each other as children —
// which the canonical-tree parser does not currently reject) terminates on the
// back-edge instead of recursing until the Go stack overflows and crashes the
// worker.
func parseDeleteNodeIDs(parseRegionDOMNodeByNodeID map[uint64]*RegionDOMNode, parseNodeID uint64) {
	parseRegionDOMNode, hasRegionDOMNode := parseRegionDOMNodeByNodeID[parseNodeID]
	if !hasRegionDOMNode {
		return
	}
	delete(parseRegionDOMNodeByNodeID, parseNodeID)
	for _, getChildNodeID := range parseRegionDOMNode.GetChildNodeIDs {
		parseDeleteNodeIDs(parseRegionDOMNodeByNodeID, getChildNodeID)
	}
}

// parseFilterChildNodeIDs returns a child list that excludes one node ID.
func parseFilterChildNodeIDs(parseChildNodeIDs []uint64, parseSkipNodeID uint64) []uint64 {
	parseSkipIndex := parseFindChildNodeIndex(parseChildNodeIDs, parseSkipNodeID)
	if parseSkipIndex < 0 {
		return parseChildNodeIDs
	}
	parseTailIndex := len(parseChildNodeIDs) - 1
	copy(parseChildNodeIDs[parseSkipIndex:], parseChildNodeIDs[parseSkipIndex+1:])
	parseChildNodeIDs[parseTailIndex] = 0
	return parseChildNodeIDs[:parseTailIndex]
}

// parseFilterChildNodeIDsBySet returns a child list that excludes every node ID stored in the provided set.
func parseFilterChildNodeIDsBySet(parseChildNodeIDs []uint64, parseSkipNodeIDSet map[uint64]struct{}) []uint64 {
	if len(parseChildNodeIDs) == 0 || len(parseSkipNodeIDSet) == 0 {
		return parseChildNodeIDs
	}
	parseWriteIndex := 0
	for _, getChildNodeID := range parseChildNodeIDs {
		if _, hasSkipNodeID := parseSkipNodeIDSet[getChildNodeID]; hasSkipNodeID {
			continue
		}
		parseChildNodeIDs[parseWriteIndex] = getChildNodeID
		parseWriteIndex++
	}
	clear(parseChildNodeIDs[parseWriteIndex:])
	return parseChildNodeIDs[:parseWriteIndex]
}

// CommitRegionMoveKeyedNode reorders one keyed child under a parent, optionally before a target sibling.
func (parseDOMCommitter *DOMCommitter) CommitRegionMoveKeyedNode(parseRegionID string, parseParentNodeID uint64, parseMoveNodeID uint64, parseBeforeNodeID uint64) (DOMCommitResult, error) {
	if parseDOMCommitter == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	parseParentNode, parseParentLookupErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseParentNodeID)
	if parseParentLookupErr != nil {
		return DOMCommitResult{}, parseParentLookupErr
	}
	parseMoveNode, parseMoveLookupErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseMoveNodeID)
	if parseMoveLookupErr != nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: missing move node %d for region %q", parseMoveNodeID, parseRegionID)
	}
	if strings.TrimSpace(parseMoveNode.GetNodeKey) == "" {
		return DOMCommitResult{}, fmt.Errorf("runtime2: move node %d is not keyed", parseMoveNodeID)
	}
	parseCurrentChildIndex, parseDestinationChildIndex, parseMoveChildErr := parseFindMoveDestinationChildIndex(
		parseParentNode.GetChildNodeIDs,
		parseParentNodeID,
		parseMoveNodeID,
		parseBeforeNodeID,
	)
	if parseMoveChildErr != nil {
		return DOMCommitResult{}, parseMoveChildErr
	}
	if parseCurrentChildIndex == parseDestinationChildIndex {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	parseParentNode.GetChildNodeIDs = parseMoveChildNodeID(parseParentNode.GetChildNodeIDs, parseCurrentChildIndex, parseDestinationChildIndex)
	parseDOMCommitter.storeCommitMutationVersion(parseRegionID)
	return DOMCommitResult{}, nil
}

// CommitRegionReplaceSubtree replaces one target node subtree with one canonical replacement subtree and rebuilds local index entries.
func (parseDOMCommitter *DOMCommitter) CommitRegionReplaceSubtree(parseRegionID string, parseTargetNodeID uint64, parseSubtreeIR CanonicalRenderIR) (DOMCommitResult, error) {
	if parseDOMCommitter == nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	if strings.TrimSpace(parseRegionID) == "" {
		return DOMCommitResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	if parseTargetNodeID == 0 {
		return DOMCommitResult{}, fmt.Errorf("runtime2: replace-subtree target node ID is required")
	}
	parseSubtreeTree, parseSubtreeTreeErr := ParseCanonicalRenderTree(parseSubtreeIR)
	if parseSubtreeTreeErr != nil {
		return DOMCommitResult{}, fmt.Errorf("runtime2: replace-subtree payload is invalid: %w", parseSubtreeTreeErr)
	}
	parseRegionNodeByNodeID, hasRegionNodeByNodeID := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionNodeByNodeID {
		return DOMCommitResult{}, fmt.Errorf("runtime2: missing node ID %d for region %q", parseTargetNodeID, parseRegionID)
	}
	parseTargetNode, hasTargetNode := parseRegionNodeByNodeID[parseTargetNodeID]
	if !hasTargetNode {
		return DOMCommitResult{}, fmt.Errorf("runtime2: missing node ID %d for region %q", parseTargetNodeID, parseRegionID)
	}
	parseParentNodeID := parseTargetNode.GetParentNodeID
	parseReplaceRootNodeID := parseSubtreeTree.getRootNodeID
	parseReplacementNodeByNodeID := parseBuildRegionDOMNodeByNodeIDFromCanonicalTree(parseSubtreeTree)
	parseReplacementRootNode, hasReplacementRootNode := parseReplacementNodeByNodeID[parseReplaceRootNodeID]
	if !hasReplacementRootNode {
		return DOMCommitResult{}, fmt.Errorf("runtime2: replace-subtree root node id %d is missing", parseReplaceRootNodeID)
	}
	if parseParentNodeID != 0 {
		parseParentNode, hasParentNode := parseRegionNodeByNodeID[parseParentNodeID]
		if !hasParentNode {
			return DOMCommitResult{}, fmt.Errorf("runtime2: missing parent node %d for target %d", parseParentNodeID, parseTargetNodeID)
		}
		parseTargetChildIndex := parseFindChildNodeIndex(parseParentNode.GetChildNodeIDs, parseTargetNodeID)
		if parseTargetChildIndex < 0 {
			return DOMCommitResult{}, fmt.Errorf("runtime2: target node %d not found under parent %d", parseTargetNodeID, parseParentNodeID)
		}
		parseParentNode.GetChildNodeIDs[parseTargetChildIndex] = parseReplaceRootNodeID
		parseReplacementRootNode.GetParentNodeID = parseParentNodeID
	} else {
		parseReplacementRootNode.GetParentNodeID = 0
	}
	parseDeleteNodeIDs(parseRegionNodeByNodeID, parseTargetNodeID)
	maps.Copy(parseRegionNodeByNodeID, parseReplacementNodeByNodeID)
	parseDOMCommitter.storeCommitMutationVersion(parseRegionID)
	return DOMCommitResult{}, nil
}

// parseBuildRegionDOMNodeByNodeIDFromCanonicalTree builds region-local DOM node entries from one canonical render tree.
func parseBuildRegionDOMNodeByNodeIDFromCanonicalTree(parseTree canonicalRenderTree) map[uint64]*RegionDOMNode {
	parseRegionNodeByNodeID := make(map[uint64]*RegionDOMNode, len(parseTree.getNodeByID))
	for getNodeID, getNode := range parseTree.getNodeByID {
		parseAttrByKey := make(map[string]string, len(getNode.getPropByKey))
		for getPropKey, getPropRecord := range getNode.getPropByKey {
			parseAttrByKey[getPropKey] = getPropRecord.Value
		}
		parseRegionNodeByNodeID[getNodeID] = &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          getNode.getTag,
			GetText:         getNode.getText,
			GetAttrByKey:    parseAttrByKey,
			GetChildNodeIDs: append([]uint64(nil), getNode.getChildNodeIDs...),
			GetParentNodeID: getNode.getParentNodeID,
			GetNodeKey:      getNode.getKey,
		}
	}
	return parseRegionNodeByNodeID
}

// parseResolveMoveBeforeNodeID resolves a destination-index move into a concrete sibling-anchor ID.
func parseResolveMoveBeforeNodeID(
	parseDOMCommitter *DOMCommitter,
	parseRegionID string,
	parseParentNodeID uint64,
	parseMoveNodeID uint64,
	parseDestinationIndex uint32,
) (uint64, error) {
	parseParentNode, parseParentLookupErr := parseDOMCommitter.getRegionDOMIndex.GetRegionDOMNode(parseRegionID, parseParentNodeID)
	if parseParentLookupErr != nil {
		return 0, parseParentLookupErr
	}
	parseCurrentIndex := parseFindChildNodeIndex(parseParentNode.GetChildNodeIDs, parseMoveNodeID)
	if parseCurrentIndex < 0 {
		return 0, fmt.Errorf("runtime2: missing move node %d under parent %d", parseMoveNodeID, parseParentNodeID)
	}
	parseRemainingChildCount := len(parseParentNode.GetChildNodeIDs) - 1
	if parseDestinationIndex > uint32(parseRemainingChildCount) {
		return 0, fmt.Errorf("runtime2: destination index %d is out of range for parent %d", parseDestinationIndex, parseParentNodeID)
	}
	if parseDestinationIndex == uint32(parseRemainingChildCount) {
		return 0, nil
	}
	parseTargetChildIndex := int(parseDestinationIndex)
	if parseTargetChildIndex >= parseCurrentIndex {
		parseTargetChildIndex++
	}
	return parseParentNode.GetChildNodeIDs[parseTargetChildIndex], nil
}

// parseFindMoveDestinationChildIndex resolves one keyed move into current and destination child indexes without rebuilding the sibling slice.
func parseFindMoveDestinationChildIndex(
	parseChildNodeIDs []uint64,
	parseParentNodeID uint64,
	parseMoveNodeID uint64,
	parseBeforeNodeID uint64,
) (int, int, error) {
	parseCurrentChildIndex := parseFindChildNodeIndex(parseChildNodeIDs, parseMoveNodeID)
	if parseCurrentChildIndex < 0 {
		return -1, -1, fmt.Errorf("runtime2: missing move node %d under parent %d", parseMoveNodeID, parseParentNodeID)
	}
	parseDestinationChildIndex := len(parseChildNodeIDs) - 1
	if parseBeforeNodeID == 0 {
		return parseCurrentChildIndex, parseDestinationChildIndex, nil
	}
	parseBeforeChildIndex := parseFindChildNodeIndex(parseChildNodeIDs, parseBeforeNodeID)
	if parseBeforeChildIndex < 0 || parseBeforeNodeID == parseMoveNodeID {
		return -1, -1, fmt.Errorf("runtime2: invalid move target %d under parent %d", parseBeforeNodeID, parseParentNodeID)
	}
	parseDestinationChildIndex = parseBeforeChildIndex
	if parseBeforeChildIndex > parseCurrentChildIndex {
		parseDestinationChildIndex--
	}
	return parseCurrentChildIndex, parseDestinationChildIndex, nil
}

// parseMoveChildNodeID reorders one existing child ID in place to the provided destination index.
func parseMoveChildNodeID(parseChildNodeIDs []uint64, parseCurrentChildIndex int, parseDestinationChildIndex int) []uint64 {
	if parseCurrentChildIndex == parseDestinationChildIndex {
		return parseChildNodeIDs
	}
	parseMoveNodeID := parseChildNodeIDs[parseCurrentChildIndex]
	if parseCurrentChildIndex < parseDestinationChildIndex {
		copy(parseChildNodeIDs[parseCurrentChildIndex:], parseChildNodeIDs[parseCurrentChildIndex+1:parseDestinationChildIndex+1])
	} else {
		copy(parseChildNodeIDs[parseDestinationChildIndex+1:], parseChildNodeIDs[parseDestinationChildIndex:parseCurrentChildIndex])
	}
	parseChildNodeIDs[parseDestinationChildIndex] = parseMoveNodeID
	return parseChildNodeIDs
}

// parseFindChildNodeIndex returns the position of one child ID or -1 when absent.
func parseFindChildNodeIndex(parseChildNodeIDs []uint64, parseNodeID uint64) int {
	for parseChildIndex, getChildNodeID := range parseChildNodeIDs {
		if getChildNodeID == parseNodeID {
			return parseChildIndex
		}
	}
	return -1
}

// parseSliceEqualUint64 reports whether two uint64 slices contain identical values in order.
func parseSliceEqualUint64(parseLeft []uint64, parseRight []uint64) bool {
	if len(parseLeft) != len(parseRight) {
		return false
	}
	for parseIndex := range parseLeft {
		if parseLeft[parseIndex] != parseRight[parseIndex] {
			return false
		}
	}
	return true
}

// CommitRegionPatchTransaction applies a patch stream as an atomic transaction with rollback on failure.
func (parseDOMCommitter *DOMCommitter) CommitRegionPatchTransaction(parseTransaction RegionPatchTransaction) (RegionPatchTransactionResult, error) {
	if parseDOMCommitter == nil {
		return RegionPatchTransactionResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	if strings.TrimSpace(parseTransaction.GetRegionID) == "" {
		return RegionPatchTransactionResult{}, fmt.Errorf("runtime2: region ID is required")
	}
	if len(parseTransaction.GetOps) > getDOMCommitTransactionOpHardLimit {
		return RegionPatchTransactionResult{}, fmt.Errorf(
			"runtime2: patch transaction op count %d exceeds guard limit %d",
			len(parseTransaction.GetOps),
			getDOMCommitTransactionOpHardLimit,
		)
	}
	if len(parseTransaction.GetOps) > getDOMCommitTransactionOpWarnLimit {
		fmt.Printf(
			"WARN: runtime2 dom commit region=%s op_count=%d exceeds soft limit=%d\n",
			parseTransaction.GetRegionID,
			len(parseTransaction.GetOps),
			getDOMCommitTransactionOpWarnLimit,
		)
	}
	if len(parseTransaction.GetOps) == 0 {
		return RegionPatchTransactionResult{}, nil
	}
	if hasCommitTransactionRemoveOnlyOps(parseTransaction.GetOps) {
		return parseDOMCommitter.commitRemoveOnlyPatchTransaction(parseTransaction)
	}
	if hasCommitTransactionAppendOnlyOps(parseTransaction.GetOps) {
		return parseDOMCommitter.commitAppendOnlyPatchTransaction(parseTransaction)
	}
	parseRegionNodeMap := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID]
	if len(parseRegionNodeMap) > getDOMCommitSnapshotNodeHardLimit {
		return RegionPatchTransactionResult{}, fmt.Errorf(
			"runtime2: region %q node count %d exceeds guard limit %d",
			parseTransaction.GetRegionID,
			len(parseRegionNodeMap),
			getDOMCommitSnapshotNodeHardLimit,
		)
	}
	if len(parseRegionNodeMap) > getDOMCommitSnapshotNodeWarnLimit {
		fmt.Printf(
			"WARN: runtime2 dom commit region=%s node_count=%d exceeds soft limit=%d\n",
			parseTransaction.GetRegionID,
			len(parseRegionNodeMap),
			getDOMCommitSnapshotNodeWarnLimit,
		)
	}
	hasCommitRegionSnapshotOps := hasCommitTransactionRegionSnapshotOps(parseTransaction.GetOps)
	var parseRegionSnapshot map[uint64]*RegionDOMNode
	var parseNodeSnapshotByID map[uint64]*RegionDOMNode
	parseDOMCommitter.beginCommitMutationBatch()
	isCommitSuccessful := false
	defer func() {
		parseDOMCommitter.finishCommitMutationBatch(parseTransaction.GetRegionID, isCommitSuccessful)
	}()
	if hasCommitRegionSnapshotOps {
		parseRegionSnapshot = parseCloneRegionNodeMap(parseRegionNodeMap)
	}
	for parseOpIndex, parseOp := range parseTransaction.GetOps {
		if !hasCommitRegionSnapshotOps {
			parseNodeSnapshotByID = parseCaptureCommitNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp)
		}
		if parseApplyErr := parseDOMCommitter.parseCommitPatchOp(parseTransaction.GetRegionID, parseOp); parseApplyErr != nil {
			if hasCommitRegionSnapshotOps {
				parseDOMCommitter.parseRestoreRegionSnapshot(parseTransaction.GetRegionID, parseRegionSnapshot)
			} else {
				parseRestoreCommitNodeSnapshot(parseRegionNodeMap, parseNodeSnapshotByID)
				if len(parseRegionNodeMap) > 0 {
					parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID] = parseRegionNodeMap
				}
			}
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, parseApplyErr)
		}
	}
	isCommitSuccessful = true
	return RegionPatchTransactionResult{}, nil
}

// commitRemoveOnlyPatchTransaction applies one remove-node-only patch transaction with shared parent filtering and subtree-root deduplication.
func (parseDOMCommitter *DOMCommitter) commitRemoveOnlyPatchTransaction(parseTransaction RegionPatchTransaction) (RegionPatchTransactionResult, error) {
	if parseDOMCommitter == nil {
		return RegionPatchTransactionResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	getRegionNodeMap, hasRegionNodeMap := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID]
	if !hasRegionNodeMap || len(getRegionNodeMap) == 0 {
		return RegionPatchTransactionResult{}, nil
	}
	getRemoveRootNodeIDSet := make(map[uint64]struct{}, len(parseTransaction.GetOps))
	getTouchedParentNodeIDSet := make(map[uint64]struct{}, 4)
	hasCommittedMutation := false
	for parseOpIndex, parseOp := range parseTransaction.GetOps {
		if parseOp.GetNodeID == 0 {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, fmt.Errorf("runtime2: node ID is required"))
		}
		getRegionDOMNode, hasRegionDOMNode := getRegionNodeMap[parseOp.GetNodeID]
		if !hasRegionDOMNode || getRegionDOMNode == nil {
			continue
		}
		if hasCommitRemoveRootAncestor(getRemoveRootNodeIDSet, getRegionNodeMap, getRegionDOMNode.GetParentNodeID) {
			continue
		}
		getRemoveRootNodeIDSet[parseOp.GetNodeID] = struct{}{}
		hasCommittedMutation = true
		if getRegionDOMNode.GetParentNodeID == 0 {
			continue
		}
		getTouchedParentNodeIDSet[getRegionDOMNode.GetParentNodeID] = struct{}{}
	}
	if !hasCommittedMutation {
		return RegionPatchTransactionResult{}, nil
	}
	for getParentNodeID := range getTouchedParentNodeIDSet {
		getParentNode, hasParentNode := getRegionNodeMap[getParentNodeID]
		if !hasParentNode || getParentNode == nil {
			continue
		}
		getParentNode.GetChildNodeIDs = parseFilterChildNodeIDsBySet(getParentNode.GetChildNodeIDs, getRemoveRootNodeIDSet)
	}
	for getRemoveRootNodeID := range getRemoveRootNodeIDSet {
		parseDeleteNodeIDs(getRegionNodeMap, getRemoveRootNodeID)
	}
	if len(getRegionNodeMap) == 0 {
		delete(parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID, parseTransaction.GetRegionID)
		delete(parseDOMCommitter.getRegionDOMIndex.storeRegionDOMVersionByRegionID, parseTransaction.GetRegionID)
		return RegionPatchTransactionResult{}, nil
	}
	parseDOMCommitter.getRegionDOMIndex.storeRegionDOMMutationVersion(parseTransaction.GetRegionID)
	return RegionPatchTransactionResult{}, nil
}

// commitAppendOnlyPatchTransaction applies one insert-only append transaction after validating every parent and node ID up front so the whole batch can commit without rollback snapshots.
func (parseDOMCommitter *DOMCommitter) commitAppendOnlyPatchTransaction(parseTransaction RegionPatchTransaction) (RegionPatchTransactionResult, error) {
	if parseDOMCommitter == nil {
		return RegionPatchTransactionResult{}, fmt.Errorf("runtime2: dom committer is nil")
	}
	getRegionNodeMap := parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID]
	if getRegionNodeMap == nil {
		getRegionNodeMap = make(map[uint64]*RegionDOMNode, len(parseTransaction.GetOps))
		parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseTransaction.GetRegionID] = getRegionNodeMap
	}
	buildInsertNodeByNodeID := make(map[uint64]*RegionDOMNode, len(parseTransaction.GetOps))
	buildAppendNodeIDsByParent := make(map[uint64][]uint64, 4)
	for parseOpIndex, parseOp := range parseTransaction.GetOps {
		if parseOp.GetInsertNode == nil {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, fmt.Errorf("runtime2: insert node is required"))
		}
		if parseOp.GetInsertNode.GetNodeID == 0 {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, fmt.Errorf("runtime2: insert node ID is required"))
		}
		if parseOp.GetBeforeNodeID != 0 {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, fmt.Errorf("runtime2: append-only transaction requires no sibling anchor"))
		}
		if _, hasInsertNode := buildInsertNodeByNodeID[parseOp.GetInsertNode.GetNodeID]; hasInsertNode {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, fmt.Errorf("runtime2: insert node ID %d already exists in region %q", parseOp.GetInsertNode.GetNodeID, parseTransaction.GetRegionID))
		}
		if getRegionNodeMap != nil {
			if _, hasRegionNode := getRegionNodeMap[parseOp.GetInsertNode.GetNodeID]; hasRegionNode {
				return RegionPatchTransactionResult{
					HasFallbackEntered: true,
				}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, fmt.Errorf("runtime2: insert node ID %d already exists in region %q", parseOp.GetInsertNode.GetNodeID, parseTransaction.GetRegionID))
			}
		}
		if _, parseParentErr := getCommitAppendParentNode(getRegionNodeMap, buildInsertNodeByNodeID, parseTransaction.GetRegionID, parseOp.GetParentNodeID); parseParentErr != nil {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, fmt.Errorf("runtime2: patch op %d failed: %w", parseOpIndex, parseParentErr)
		}
		buildInsertNodeByNodeID[parseOp.GetInsertNode.GetNodeID] = parseOp.GetInsertNode
		buildAppendNodeIDsByParent[parseOp.GetParentNodeID] = append(buildAppendNodeIDsByParent[parseOp.GetParentNodeID], parseOp.GetInsertNode.GetNodeID)
	}
	parseDOMCommitter.beginCommitMutationBatch()
	isCommitSuccessful := false
	defer func() {
		parseDOMCommitter.finishCommitMutationBatch(parseTransaction.GetRegionID, isCommitSuccessful)
	}()
	for _, parseOp := range parseTransaction.GetOps {
		parseOp.GetInsertNode.GetParentNodeID = parseOp.GetParentNodeID
		getRegionNodeMap[parseOp.GetInsertNode.GetNodeID] = parseOp.GetInsertNode
	}
	for getParentNodeID, getAppendNodeIDs := range buildAppendNodeIDsByParent {
		getParentNode, parseParentErr := getCommitAppendParentNode(
			getRegionNodeMap,
			buildInsertNodeByNodeID,
			parseTransaction.GetRegionID,
			getParentNodeID,
		)
		if parseParentErr != nil {
			return RegionPatchTransactionResult{
				HasFallbackEntered: true,
			}, parseParentErr
		}
		getParentNode.GetChildNodeIDs = append(getParentNode.GetChildNodeIDs, getAppendNodeIDs...)
	}
	parseDOMCommitter.storeCommitMutationVersion(parseTransaction.GetRegionID)
	isCommitSuccessful = true
	return RegionPatchTransactionResult{}, nil
}

// hasCommitTransactionRegionSnapshotOps reports whether one transaction contains structural ops that still require a full region snapshot for rollback.
func hasCommitTransactionRegionSnapshotOps(parseOps []RegionPatchOp) bool {
	for _, parseOp := range parseOps {
		if hasCommitRegionSnapshotOpKind(parseOp.GetKind) {
			return true
		}
	}
	return false
}

// hasCommitTransactionRemoveOnlyOps reports whether one transaction contains only remove-node ops, which commit without rollback snapshots because the remove path cannot fail after parser validation.
func hasCommitTransactionRemoveOnlyOps(parseOps []RegionPatchOp) bool {
	if len(parseOps) == 0 {
		return false
	}
	for _, parseOp := range parseOps {
		if parseOp.GetKind != RegionPatchOpKindRemoveNode {
			return false
		}
	}
	return true
}

// hasCommitTransactionAppendOnlyOps reports whether one transaction contains only anchor-free insert ops that can validate up front and then commit without rollback snapshots.
func hasCommitTransactionAppendOnlyOps(parseOps []RegionPatchOp) bool {
	if len(parseOps) == 0 {
		return false
	}
	for _, parseOp := range parseOps {
		if parseOp.GetKind != RegionPatchOpKindInsertNode || parseOp.GetBeforeNodeID != 0 {
			return false
		}
	}
	return true
}

// hasCommitRemoveRootAncestor reports whether one remove-root set already contains an ancestor of the candidate node.
func hasCommitRemoveRootAncestor(
	parseRemoveRootNodeIDSet map[uint64]struct{},
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseParentNodeID uint64,
) bool {
	// Bound the walk by the node count: a malformed parent-pointer cycle would
	// otherwise loop forever. The chain can visit each node at most once.
	for parseSteps := 0; parseParentNodeID != 0 && parseSteps <= len(parseRegionNodeMap); parseSteps++ {
		if _, hasRemoveRoot := parseRemoveRootNodeIDSet[parseParentNodeID]; hasRemoveRoot {
			return true
		}
		getParentNode, hasParentNode := parseRegionNodeMap[parseParentNodeID]
		if !hasParentNode || getParentNode == nil {
			return false
		}
		parseParentNodeID = getParentNode.GetParentNodeID
	}
	return false
}

// getCommitAppendParentNode resolves one append-only insert parent from either the committed region map or an earlier insert in the same transaction.
func getCommitAppendParentNode(
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseInsertNodeByNodeID map[uint64]*RegionDOMNode,
	parseRegionID string,
	parseParentNodeID uint64,
) (*RegionDOMNode, error) {
	if parseParentNodeID == 0 {
		return nil, fmt.Errorf("runtime2: node ID is required")
	}
	if getInsertParentNode, hasInsertParentNode := parseInsertNodeByNodeID[parseParentNodeID]; hasInsertParentNode && getInsertParentNode != nil {
		return getInsertParentNode, nil
	}
	if parseRegionNodeMap == nil {
		return nil, fmt.Errorf("runtime2: missing node ID %d for region %q", parseParentNodeID, parseRegionID)
	}
	getParentNode, hasParentNode := parseRegionNodeMap[parseParentNodeID]
	if !hasParentNode || getParentNode == nil {
		return nil, fmt.Errorf("runtime2: missing node ID %d for region %q", parseParentNodeID, parseRegionID)
	}
	return getParentNode, nil
}

// hasCommitRegionSnapshotOpKind reports whether one op kind requires a full region snapshot because node-local rollback is insufficient.
func hasCommitRegionSnapshotOpKind(parseOpKind RegionPatchOpKind) bool {
	switch parseOpKind {
	case RegionPatchOpKindReplaceSubtree:
		return true
	default:
		return false
	}
}

// parseCaptureCommitNodeSnapshot stores one pre-mutation node snapshot for node-local rollback when a full region snapshot is unnecessary.
func parseCaptureCommitNodeSnapshot(
	parseNodeSnapshotByID map[uint64]*RegionDOMNode,
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseOp RegionPatchOp,
) map[uint64]*RegionDOMNode {
	switch parseOp.GetKind {
	case RegionPatchOpKindSetText,
		RegionPatchOpKindSetAttr,
		RegionPatchOpKindSetStyle,
		RegionPatchOpKindRemoveAttr,
		RegionPatchOpKindRemoveStyle:
		return storeCommitNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp.GetNodeID)
	case RegionPatchOpKindInsertNode:
		parseNodeSnapshotByID = storeCommitStructuralNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp.GetParentNodeID)
		if parseOp.GetInsertNode != nil {
			parseNodeSnapshotByID = storeCommitMissingNodeSnapshot(parseNodeSnapshotByID, parseOp.GetInsertNode.GetNodeID)
		}
		return parseNodeSnapshotByID
	case RegionPatchOpKindRemoveNode:
		parseNodeSnapshotByID = storeCommitRemovedSubtreeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp.GetNodeID)
		parseRegionDOMNode, hasRegionDOMNode := parseRegionNodeMap[parseOp.GetNodeID]
		if hasRegionDOMNode && parseRegionDOMNode != nil {
			parseNodeSnapshotByID = storeCommitStructuralNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseRegionDOMNode.GetParentNodeID)
		}
		return parseNodeSnapshotByID
	case RegionPatchOpKindMoveKeyedNode:
		return storeCommitStructuralNodeSnapshot(parseNodeSnapshotByID, parseRegionNodeMap, parseOp.GetParentNodeID)
	default:
		return parseNodeSnapshotByID
	}
}

// storeCommitNodeSnapshot stores one node snapshot only once so rollback can restore pre-mutation node state.
func storeCommitNodeSnapshot(
	parseNodeSnapshotByID map[uint64]*RegionDOMNode,
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseNodeID uint64,
) map[uint64]*RegionDOMNode {
	if parseNodeID == 0 || parseRegionNodeMap == nil {
		return parseNodeSnapshotByID
	}
	if _, hasNodeSnapshot := parseNodeSnapshotByID[parseNodeID]; hasNodeSnapshot {
		return parseNodeSnapshotByID
	}
	parseRegionDOMNode, hasRegionDOMNode := parseRegionNodeMap[parseNodeID]
	if !hasRegionDOMNode || parseRegionDOMNode == nil {
		return parseNodeSnapshotByID
	}
	if parseNodeSnapshotByID == nil {
		parseNodeSnapshotByID = make(map[uint64]*RegionDOMNode, 4)
	}
	parseNodeSnapshotByID[parseNodeID] = parseCloneRegionDOMNodeForNonStructuralRollback(parseRegionDOMNode)
	return parseNodeSnapshotByID
}

// storeCommitStructuralNodeSnapshot stores one node snapshot with an owned child-order slice so structural rollback can restore sibling order.
func storeCommitStructuralNodeSnapshot(
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
	if _, hasNodeSnapshot := parseNodeSnapshotByID[parseNodeID]; !hasNodeSnapshot {
		parseNodeSnapshotByID[parseNodeID] = parseCloneRegionDOMNode(parseRegionDOMNode)
	}
	return parseNodeSnapshotByID
}

// storeCommitMissingNodeSnapshot stores one nil snapshot marker so rollback can delete nodes created during the transaction.
func storeCommitMissingNodeSnapshot(parseNodeSnapshotByID map[uint64]*RegionDOMNode, parseNodeID uint64) map[uint64]*RegionDOMNode {
	if parseNodeID == 0 {
		return parseNodeSnapshotByID
	}
	if parseNodeSnapshotByID == nil {
		parseNodeSnapshotByID = make(map[uint64]*RegionDOMNode, 4)
	}
	if _, hasNodeSnapshot := parseNodeSnapshotByID[parseNodeID]; hasNodeSnapshot {
		return parseNodeSnapshotByID
	}
	parseNodeSnapshotByID[parseNodeID] = nil
	return parseNodeSnapshotByID
}

// storeCommitRemovedSubtreeSnapshot stores one deep-cloned subtree snapshot so remove-node rollback can restore the removed nodes without cloning the full region.
func storeCommitRemovedSubtreeSnapshot(
	parseNodeSnapshotByID map[uint64]*RegionDOMNode,
	parseRegionNodeMap map[uint64]*RegionDOMNode,
	parseNodeID uint64,
) map[uint64]*RegionDOMNode {
	if parseNodeID == 0 || parseRegionNodeMap == nil {
		return parseNodeSnapshotByID
	}
	getRemovedNodeIDs := buildRegionDOMSubtreeNodeIDs(parseRegionNodeMap, parseNodeID)
	if len(getRemovedNodeIDs) == 0 {
		return parseNodeSnapshotByID
	}
	if parseNodeSnapshotByID == nil {
		parseNodeSnapshotByID = make(map[uint64]*RegionDOMNode, len(getRemovedNodeIDs))
	}
	for _, getRemovedNodeID := range getRemovedNodeIDs {
		if _, hasNodeSnapshot := parseNodeSnapshotByID[getRemovedNodeID]; hasNodeSnapshot {
			continue
		}
		getRegionDOMNode, hasRegionDOMNode := parseRegionNodeMap[getRemovedNodeID]
		if !hasRegionDOMNode || getRegionDOMNode == nil {
			continue
		}
		parseNodeSnapshotByID[getRemovedNodeID] = parseCloneRegionDOMNode(getRegionDOMNode)
	}
	return parseNodeSnapshotByID
}

// parseCommitPatchOp applies one commit op payload to a region.
func (parseDOMCommitter *DOMCommitter) parseCommitPatchOp(parseRegionID string, parseOp RegionPatchOp) error {
	switch parseOp.GetKind {
	case RegionPatchOpKindSetText:
		_, parseErr := parseDOMCommitter.CommitRegionText(parseRegionID, parseOp.GetNodeID, parseOp.GetText)
		return parseErr
	case RegionPatchOpKindSetAttr:
		_, parseErr := parseDOMCommitter.CommitRegionAttr(parseRegionID, parseOp.GetNodeID, parseOp.GetAttrKey, parseOp.GetAttrValue)
		return parseErr
	case RegionPatchOpKindSetStyle:
		_, parseErr := parseDOMCommitter.CommitRegionSetStyle(parseRegionID, parseOp.GetNodeID, parseOp.GetStyleValue)
		return parseErr
	case RegionPatchOpKindRemoveAttr:
		_, parseErr := parseDOMCommitter.CommitRegionRemoveAttr(parseRegionID, parseOp.GetNodeID, parseOp.GetAttrKey)
		return parseErr
	case RegionPatchOpKindRemoveStyle:
		_, parseErr := parseDOMCommitter.CommitRegionRemoveStyle(parseRegionID, parseOp.GetNodeID)
		return parseErr
	case RegionPatchOpKindInsertNode:
		_, parseErr := parseDOMCommitter.CommitRegionInsertNode(parseRegionID, parseOp.GetParentNodeID, parseOp.GetInsertNode, parseOp.GetBeforeNodeID)
		return parseErr
	case RegionPatchOpKindRemoveNode:
		_, parseErr := parseDOMCommitter.CommitRegionRemoveNode(parseRegionID, parseOp.GetNodeID)
		return parseErr
	case RegionPatchOpKindReplaceSubtree:
		_, parseErr := parseDOMCommitter.CommitRegionReplaceSubtree(parseRegionID, parseOp.GetNodeID, parseOp.GetReplaceSubtreeIR)
		return parseErr
	case RegionPatchOpKindMoveKeyedNode:
		parseMoveNodeID := parseOp.GetMoveNodeID
		if parseMoveNodeID == 0 {
			parseMoveNodeID = parseOp.GetNodeID
		}
		parseBeforeNodeID := parseOp.GetBeforeNodeID
		if parseBeforeNodeID == 0 {
			buildBeforeNodeID, parseBeforeNodeErr := parseResolveMoveBeforeNodeID(parseDOMCommitter, parseRegionID, parseOp.GetParentNodeID, parseMoveNodeID, parseOp.GetDestinationIndex)
			if parseBeforeNodeErr != nil {
				return parseBeforeNodeErr
			}
			parseBeforeNodeID = buildBeforeNodeID
		}
		_, parseErr := parseDOMCommitter.CommitRegionMoveKeyedNode(parseRegionID, parseOp.GetParentNodeID, parseMoveNodeID, parseBeforeNodeID)
		return parseErr
	case regionPatchOpKindInvalid:
		return fmt.Errorf("runtime2: patch op kind is required")
	default:
		return fmt.Errorf("runtime2: patch op kind %q is unsupported", parseOp.GetKind)
	}
}

// parseRestoreRegionSnapshot restores one region's node map snapshot after a failed transaction.
func (parseDOMCommitter *DOMCommitter) parseRestoreRegionSnapshot(parseRegionID string, parseRegionSnapshot map[uint64]*RegionDOMNode) {
	if parseRegionSnapshot == nil {
		delete(parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID, parseRegionID)
		return
	}
	parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID] = parseRegionSnapshot
}

// parseRestoreCommitNodeSnapshot restores one node-level rollback snapshot set into the active region node map.
func parseRestoreCommitNodeSnapshot(parseRegionNodeMap map[uint64]*RegionDOMNode, parseNodeSnapshotByID map[uint64]*RegionDOMNode) {
	if parseRegionNodeMap == nil || len(parseNodeSnapshotByID) == 0 {
		return
	}
	for getNodeID, getNodeSnapshot := range parseNodeSnapshotByID {
		if getNodeSnapshot == nil {
			delete(parseRegionNodeMap, getNodeID)
			continue
		}
		parseRegionNodeMap[getNodeID] = getNodeSnapshot
	}
}

// parseCloneRegionDOMNode deep-copies one region-local DOM node.
func parseCloneRegionDOMNode(parseRegionDOMNode *RegionDOMNode) *RegionDOMNode {
	if parseRegionDOMNode == nil {
		return nil
	}
	var parseCloneAttrByKey map[string]string
	if len(parseRegionDOMNode.GetAttrByKey) > 0 {
		parseCloneAttrByKey = make(map[string]string, len(parseRegionDOMNode.GetAttrByKey))
		maps.Copy(parseCloneAttrByKey, parseRegionDOMNode.GetAttrByKey)
	}
	return &RegionDOMNode{
		GetNodeID:       parseRegionDOMNode.GetNodeID,
		GetTag:          parseRegionDOMNode.GetTag,
		GetText:         parseRegionDOMNode.GetText,
		GetAttrByKey:    parseCloneAttrByKey,
		GetChildNodeIDs: append([]uint64(nil), parseRegionDOMNode.GetChildNodeIDs...),
		GetParentNodeID: parseRegionDOMNode.GetParentNodeID,
		GetNodeKey:      parseRegionDOMNode.GetNodeKey,
	}
}

// parseCloneRegionDOMNodeForNonStructuralRollback deep-copies one node for text/attr rollback without cloning stable child-order slices.
func parseCloneRegionDOMNodeForNonStructuralRollback(parseRegionDOMNode *RegionDOMNode) *RegionDOMNode {
	if parseRegionDOMNode == nil {
		return nil
	}
	var parseCloneAttrByKey map[string]string
	if len(parseRegionDOMNode.GetAttrByKey) > 0 {
		parseCloneAttrByKey = make(map[string]string, len(parseRegionDOMNode.GetAttrByKey))
		maps.Copy(parseCloneAttrByKey, parseRegionDOMNode.GetAttrByKey)
	}
	return &RegionDOMNode{
		GetNodeID:       parseRegionDOMNode.GetNodeID,
		GetTag:          parseRegionDOMNode.GetTag,
		GetText:         parseRegionDOMNode.GetText,
		GetAttrByKey:    parseCloneAttrByKey,
		GetChildNodeIDs: parseRegionDOMNode.GetChildNodeIDs,
		GetParentNodeID: parseRegionDOMNode.GetParentNodeID,
		GetNodeKey:      parseRegionDOMNode.GetNodeKey,
	}
}

// parseCloneRegionNodeMap deep-copies one region-local node map for rollback safety.
func parseCloneRegionNodeMap(parseRegionNodeMap map[uint64]*RegionDOMNode) map[uint64]*RegionDOMNode {
	if parseRegionNodeMap == nil {
		return nil
	}
	parseCloneNodeMap := make(map[uint64]*RegionDOMNode, len(parseRegionNodeMap))
	for getNodeID, getRegionDOMNode := range parseRegionNodeMap {
		if getRegionDOMNode == nil {
			continue
		}
		parseCloneNodeMap[getNodeID] = parseCloneRegionDOMNode(getRegionDOMNode)
	}
	return parseCloneNodeMap
}
