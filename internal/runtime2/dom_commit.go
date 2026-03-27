package runtime2

import (
	"fmt"
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
	getRegionDOMIndex *RegionDOMIndex
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
	parseChildNodeIDs, parseChildNodeIDsErr := parseInsertChildNodeIDs(parseParentNode.GetChildNodeIDs, parseInsertNode.GetNodeID, parseBeforeNodeID)
	if parseChildNodeIDsErr != nil {
		return DOMCommitResult{}, parseChildNodeIDsErr
	}
	parseInsertNode.GetParentNodeID = parseParentNodeID
	parseSetNodeErr := parseDOMCommitter.getRegionDOMIndex.SetRegionDOMNode(parseRegionID, parseInsertNode.GetNodeID, parseInsertNode)
	if parseSetNodeErr != nil {
		return DOMCommitResult{}, parseSetNodeErr
	}
	parseParentNode.GetChildNodeIDs = parseChildNodeIDs
	return DOMCommitResult{}, nil
}

// parseInsertChildNodeIDs returns a child-node order with one inserted node, optionally before an anchor.
func parseInsertChildNodeIDs(parseChildNodeIDs []uint64, parseInsertNodeID uint64, parseBeforeNodeID uint64) ([]uint64, error) {
	if parseBeforeNodeID == 0 {
		parseNextChildNodeIDs := append([]uint64(nil), parseChildNodeIDs...)
		parseNextChildNodeIDs = append(parseNextChildNodeIDs, parseInsertNodeID)
		return parseNextChildNodeIDs, nil
	}
	parseAnchorIndex := -1
	for parseChildIndex, getChildNodeID := range parseChildNodeIDs {
		if getChildNodeID == parseBeforeNodeID {
			parseAnchorIndex = parseChildIndex
			break
		}
	}
	if parseAnchorIndex < 0 {
		return nil, fmt.Errorf("runtime2: invalid sibling anchor %d", parseBeforeNodeID)
	}
	parseNextChildNodeIDs := make([]uint64, 0, len(parseChildNodeIDs)+1)
	parseNextChildNodeIDs = append(parseNextChildNodeIDs, parseChildNodeIDs[:parseAnchorIndex]...)
	parseNextChildNodeIDs = append(parseNextChildNodeIDs, parseInsertNodeID)
	parseNextChildNodeIDs = append(parseNextChildNodeIDs, parseChildNodeIDs[parseAnchorIndex:]...)
	return parseNextChildNodeIDs, nil
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
	}
	return DOMCommitResult{}, nil
}

// parseDeleteNodeIDs deletes one node subtree from a region-local node map.
func parseDeleteNodeIDs(parseRegionDOMNodeByNodeID map[uint64]*RegionDOMNode, parseNodeID uint64) {
	parseRegionDOMNode, hasRegionDOMNode := parseRegionDOMNodeByNodeID[parseNodeID]
	if !hasRegionDOMNode {
		return
	}
	for _, getChildNodeID := range parseRegionDOMNode.GetChildNodeIDs {
		parseDeleteNodeIDs(parseRegionDOMNodeByNodeID, getChildNodeID)
	}
	delete(parseRegionDOMNodeByNodeID, parseNodeID)
}

// parseFilterChildNodeIDs returns a child list that excludes one node ID.
func parseFilterChildNodeIDs(parseChildNodeIDs []uint64, parseSkipNodeID uint64) []uint64 {
	parseNextChildNodeIDs := make([]uint64, 0, len(parseChildNodeIDs))
	for _, getChildNodeID := range parseChildNodeIDs {
		if getChildNodeID == parseSkipNodeID {
			continue
		}
		parseNextChildNodeIDs = append(parseNextChildNodeIDs, getChildNodeID)
	}
	return parseNextChildNodeIDs
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
	parseCurrentChildIndex := parseFindChildNodeIndex(parseParentNode.GetChildNodeIDs, parseMoveNodeID)
	if parseCurrentChildIndex < 0 {
		return DOMCommitResult{}, fmt.Errorf("runtime2: missing move node %d under parent %d", parseMoveNodeID, parseParentNodeID)
	}
	parseNextChildNodeIDs := append([]uint64(nil), parseParentNode.GetChildNodeIDs...)
	parseNextChildNodeIDs = append(parseNextChildNodeIDs[:parseCurrentChildIndex], parseNextChildNodeIDs[parseCurrentChildIndex+1:]...)
	parseInsertIndex := len(parseNextChildNodeIDs)
	if parseBeforeNodeID != 0 {
		parseInsertIndex = parseFindChildNodeIndex(parseNextChildNodeIDs, parseBeforeNodeID)
		if parseInsertIndex < 0 {
			return DOMCommitResult{}, fmt.Errorf("runtime2: invalid move target %d under parent %d", parseBeforeNodeID, parseParentNodeID)
		}
	}
	parseNextChildNodeIDs = append(parseNextChildNodeIDs, 0)
	copy(parseNextChildNodeIDs[parseInsertIndex+1:], parseNextChildNodeIDs[parseInsertIndex:])
	parseNextChildNodeIDs[parseInsertIndex] = parseMoveNodeID
	if parseSliceEqualUint64(parseParentNode.GetChildNodeIDs, parseNextChildNodeIDs) {
		return DOMCommitResult{
			IsNoOp: true,
		}, nil
	}
	parseParentNode.GetChildNodeIDs = parseNextChildNodeIDs
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
	for getNodeID, getNode := range parseReplacementNodeByNodeID {
		parseRegionNodeByNodeID[getNodeID] = getNode
	}
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
	parseCurrentChildNodeIDs := append([]uint64(nil), parseParentNode.GetChildNodeIDs...)
	parseCurrentIndex := parseFindChildNodeIndex(parseCurrentChildNodeIDs, parseMoveNodeID)
	if parseCurrentIndex < 0 {
		return 0, fmt.Errorf("runtime2: missing move node %d under parent %d", parseMoveNodeID, parseParentNodeID)
	}
	parseCurrentChildNodeIDs = append(parseCurrentChildNodeIDs[:parseCurrentIndex], parseCurrentChildNodeIDs[parseCurrentIndex+1:]...)
	if parseDestinationIndex > uint32(len(parseCurrentChildNodeIDs)) {
		return 0, fmt.Errorf("runtime2: destination index %d is out of range for parent %d", parseDestinationIndex, parseParentNodeID)
	}
	if parseDestinationIndex == uint32(len(parseCurrentChildNodeIDs)) {
		return 0, nil
	}
	return parseCurrentChildNodeIDs[parseDestinationIndex], nil
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
	parseDOMCommitter.getRegionDOMIndex.storeRegionDOMNodeByRegionID[parseRegionID] = parseCloneRegionNodeMap(parseRegionSnapshot)
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
		parseCloneAttrByKey := make(map[string]string, len(getRegionDOMNode.GetAttrByKey))
		for getAttrKey, getAttrValue := range getRegionDOMNode.GetAttrByKey {
			parseCloneAttrByKey[getAttrKey] = getAttrValue
		}
		parseCloneNodeMap[getNodeID] = &RegionDOMNode{
			GetNodeID:       getRegionDOMNode.GetNodeID,
			GetTag:          getRegionDOMNode.GetTag,
			GetText:         getRegionDOMNode.GetText,
			GetAttrByKey:    parseCloneAttrByKey,
			GetChildNodeIDs: append([]uint64(nil), getRegionDOMNode.GetChildNodeIDs...),
			GetParentNodeID: getRegionDOMNode.GetParentNodeID,
			GetNodeKey:      getRegionDOMNode.GetNodeKey,
		}
	}
	return parseCloneNodeMap
}
