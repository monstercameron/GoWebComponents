package runtime2

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sort"
	"strings"
)

const (
	getPatchStreamOpWarnLimit    = 2048
	getPatchStreamOpHardLimit    = 16384
	getPatchMoveSiblingWarnLimit = 512
	getPatchMoveSiblingHardLimit = 8192
)

// PatchStreamOpRaw stores one raw patch-op payload with exactly one typed payload body.
type PatchStreamOpRaw struct {
	GetOpCode uint8 `json:"op_code"`

	GetInsertOp         *PatchInsertOpRaw         `json:"insert,omitempty"`
	GetRemoveOp         *PatchRemoveOpRaw         `json:"remove,omitempty"`
	GetSetTextOp        *PatchSetTextOpRaw        `json:"set_text,omitempty"`
	GetSetAttrOp        *PatchSetAttrOpRaw        `json:"set_attr,omitempty"`
	GetSetStyleOp       *PatchSetStyleOpRaw       `json:"set_style,omitempty"`
	GetRemoveAttrOp     *PatchRemoveAttrOpRaw     `json:"remove_attr,omitempty"`
	GetRemoveStyleOp    *PatchRemoveStyleOpRaw    `json:"remove_style,omitempty"`
	GetKeyedMoveOp      *PatchKeyedMoveOpRaw      `json:"move_keyed,omitempty"`
	GetReplaceSubtreeOp *PatchReplaceSubtreeOpRaw `json:"replace_subtree,omitempty"`
}

// PatchStreamRaw stores one raw patch stream payload for transport and commit parsing.
type PatchStreamRaw struct {
	GetHeader        PatchStreamHeaderRaw `json:"header"`
	GetStringTable   []string             `json:"string_table,omitempty"`
	GetOps           []PatchStreamOpRaw   `json:"ops,omitempty"`
	GetPatchIdentity string               `json:"patch_identity"`
}

// PatchStreamParseResult stores one parsed patch stream plus a commit transaction.
type PatchStreamParseResult struct {
	GetHeader      PatchStreamHeader
	GetTransaction RegionPatchTransaction
}

// BuildPatchStreamRaw builds one typed patch stream and computes deterministic idempotency metadata.
func BuildPatchStreamRaw(parseHeader PatchStreamHeaderRaw, parseStringTable RenderStringTable, parseOps []PatchStreamOpRaw) (PatchStreamRaw, error) {
	if _, parseHeaderErr := ParsePatchStreamHeader(parseHeader, parseHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	buildStream := PatchStreamRaw{
		GetHeader:      parseHeader,
		GetStringTable: append([]string(nil), parseStringTable.Entries...),
		GetOps:         append([]PatchStreamOpRaw(nil), parseOps...),
	}
	buildPatchIdentity, parseIdentityErr := BuildPatchStreamIdentity(buildStream)
	if parseIdentityErr != nil {
		return PatchStreamRaw{}, parseIdentityErr
	}
	buildStream.GetPatchIdentity = buildPatchIdentity
	return buildStream, nil
}

// BuildPatchStreamIdentity computes one deterministic patch identity used by idempotency tracking.
func BuildPatchStreamIdentity(parseRaw PatchStreamRaw) (string, error) {
	buildPayload, parsePayloadErr := json.Marshal(struct {
		GetHeader      PatchStreamHeaderRaw
		GetStringTable []string
		GetOps         []PatchStreamOpRaw
	}{
		GetHeader:      parseRaw.GetHeader,
		GetStringTable: parseRaw.GetStringTable,
		GetOps:         parseRaw.GetOps,
	})
	if parsePayloadErr != nil {
		return "", fmt.Errorf("runtime2: encode patch identity payload: %w", parsePayloadErr)
	}
	buildHasher := fnv.New64a()
	_, _ = buildHasher.Write(buildPayload)
	return fmt.Sprintf("%x", buildHasher.Sum64()), nil
}

// ParsePatchStreamTransaction decodes one patch stream, validates ordering and idempotency, and returns a commit transaction.
func ParsePatchStreamTransaction(
	parseRaw PatchStreamRaw,
	parseExpectedRegionID string,
	parseExpectedEpoch uint64,
	parseKnownNodeIDs map[uint64]struct{},
	parseSiblingCountByParent map[uint64]uint32,
	parseTracker *PatchIdempotencyTracker,
) (PatchStreamParseResult, bool, error) {
	parseHeader, parseHeaderErr := ParsePatchStreamHeader(parseRaw.GetHeader, parseExpectedRegionID)
	if parseHeaderErr != nil {
		return PatchStreamParseResult{}, false, parseHeaderErr
	}
	if parseExpectedEpoch > 0 && parseHeader.Epoch != parseExpectedEpoch {
		return PatchStreamParseResult{}, false, fmt.Errorf(
			"runtime2: patch stream epoch %d does not match expected epoch %d",
			parseHeader.Epoch,
			parseExpectedEpoch,
		)
	}
	if strings.TrimSpace(parseRaw.GetPatchIdentity) == "" {
		return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch stream identity is required")
	}
	if parseTracker != nil {
		hasApply, parseIdempotencyErr := parseTracker.HandlePatchIdempotency(
			parseHeader.RegionID,
			parseHeader.Epoch,
			parseHeader.PatchVersion,
			parseRaw.GetPatchIdentity,
		)
		if parseIdempotencyErr != nil {
			return PatchStreamParseResult{}, false, parseIdempotencyErr
		}
		if !hasApply {
			return PatchStreamParseResult{}, false, nil
		}
	}
	parseStringTable, parseStringTableErr := ParseRenderStringTable(parseRaw.GetStringTable)
	if parseStringTableErr != nil {
		return PatchStreamParseResult{}, false, parseStringTableErr
	}
	if len(parseRaw.GetOps) > getPatchStreamOpHardLimit {
		return PatchStreamParseResult{}, false, fmt.Errorf(
			"runtime2: patch stream op count %d exceeds guard limit %d",
			len(parseRaw.GetOps),
			getPatchStreamOpHardLimit,
		)
	}
	if len(parseRaw.GetOps) > getPatchStreamOpWarnLimit {
		fmt.Printf(
			"WARN: runtime2 patch stream region=%s op_count=%d exceeds soft limit=%d\n",
			parseHeader.RegionID,
			len(parseRaw.GetOps),
			getPatchStreamOpWarnLimit,
		)
	}
	buildSiblingCountByParent := make(map[uint64]uint32, len(parseSiblingCountByParent))
	for getParentNodeID, getSiblingCount := range parseSiblingCountByParent {
		buildSiblingCountByParent[getParentNodeID] = getSiblingCount
	}
	buildPatchOrderEntries, parseOrderErr := parseBuildPatchOrderEntries(parseRaw.GetOps)
	if parseOrderErr != nil {
		return PatchStreamParseResult{}, false, parseOrderErr
	}
	if parseValidateOrderErr := ValidatePatchOrder(buildPatchOrderEntries, parseKnownNodeIDs); parseValidateOrderErr != nil {
		return PatchStreamParseResult{}, false, parseValidateOrderErr
	}
	buildKnownNodeIDs := make(map[uint64]struct{}, len(parseKnownNodeIDs))
	for getNodeID := range parseKnownNodeIDs {
		buildKnownNodeIDs[getNodeID] = struct{}{}
	}
	buildRemovedNodeIDs := make(map[uint64]struct{})
	buildRemovedAttrKeys := make(map[string]struct{})
	buildTransaction := RegionPatchTransaction{
		GetRegionID: parseHeader.RegionID,
	}
	for parseOpIndex, getRawOp := range parseRaw.GetOps {
		parseOpCode, parseOpCodeErr := ParsePatchOpCode(getRawOp.GetOpCode)
		if parseOpCodeErr != nil {
			return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d has invalid op code: %w", parseOpIndex, parseOpCodeErr)
		}
		switch parseOpCode {
		case PatchOpCodeInsertNode:
			if getRawOp.GetInsertOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d insert payload is required", parseOpIndex)
			}
			parseInsertOp, parseInsertErr := ParsePatchInsertOp(*getRawOp.GetInsertOp, buildKnownNodeIDs)
			if parseInsertErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d insert is invalid: %w", parseOpIndex, parseInsertErr)
			}
			buildInsertNode, parseInsertNodeErr := parseBuildRegionDOMNodeFromPatchRecord(parseInsertOp.Node, parseStringTable)
			if parseInsertNodeErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d insert node is invalid: %w", parseOpIndex, parseInsertNodeErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: parseInsertOp.ParentNodeID,
				GetBeforeNodeID: parseInsertOp.AnchorNodeID,
				GetInsertNode:   buildInsertNode,
			})
			buildKnownNodeIDs[parseInsertOp.Node.NodeID] = struct{}{}
			buildSiblingCountByParent[parseInsertOp.ParentNodeID] = buildSiblingCountByParent[parseInsertOp.ParentNodeID] + 1
		case PatchOpCodeRemoveNode:
			if getRawOp.GetRemoveOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove payload is required", parseOpIndex)
			}
			parseRemoveOp, parseRemoveErr := ParsePatchRemoveOp(*getRawOp.GetRemoveOp, buildKnownNodeIDs, buildRemovedNodeIDs)
			if parseRemoveErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove is invalid: %w", parseOpIndex, parseRemoveErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:   RegionPatchOpKindRemoveNode,
				GetNodeID: parseRemoveOp.TargetNodeID,
			})
			delete(buildKnownNodeIDs, parseRemoveOp.TargetNodeID)
		case PatchOpCodeSetText:
			if getRawOp.GetSetTextOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d set-text payload is required", parseOpIndex)
			}
			parseSetTextOp, parseSetTextErr := ParsePatchSetTextOp(*getRawOp.GetSetTextOp, buildKnownNodeIDs, parseStringTable)
			if parseSetTextErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d set-text is invalid: %w", parseOpIndex, parseSetTextErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: parseSetTextOp.TargetNodeID,
				GetText:   parseSetTextOp.Text,
			})
		case PatchOpCodeSetAttr:
			if getRawOp.GetSetAttrOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d set-attr payload is required", parseOpIndex)
			}
			parseSetAttrOp, parseSetAttrErr := ParsePatchSetAttrOp(*getRawOp.GetSetAttrOp, buildKnownNodeIDs, parseStringTable)
			if parseSetAttrErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d set-attr is invalid: %w", parseOpIndex, parseSetAttrErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:      RegionPatchOpKindSetAttr,
				GetNodeID:    parseSetAttrOp.TargetNodeID,
				GetAttrKey:   parseSetAttrOp.Attr.Key,
				GetAttrValue: parseSetAttrOp.Attr.Value,
			})
		case PatchOpCodeSetStyle:
			if getRawOp.GetSetStyleOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d set-style payload is required", parseOpIndex)
			}
			parseSetStyleOp, parseSetStyleErr := ParsePatchSetStyleOp(*getRawOp.GetSetStyleOp, buildKnownNodeIDs, parseStringTable)
			if parseSetStyleErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d set-style is invalid: %w", parseOpIndex, parseSetStyleErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:       RegionPatchOpKindSetStyle,
				GetNodeID:     parseSetStyleOp.TargetNodeID,
				GetStyleValue: parseSetStyleOp.StyleValue,
			})
		case PatchOpCodeRemoveAttr:
			if getRawOp.GetRemoveAttrOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove-attr payload is required", parseOpIndex)
			}
			parseRemoveAttrOp, parseRemoveAttrErr := ParsePatchRemoveAttrOp(*getRawOp.GetRemoveAttrOp, buildKnownNodeIDs, parseStringTable, buildRemovedAttrKeys)
			if parseRemoveAttrErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove-attr is invalid: %w", parseOpIndex, parseRemoveAttrErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:    RegionPatchOpKindRemoveAttr,
				GetNodeID:  parseRemoveAttrOp.TargetNodeID,
				GetAttrKey: parseRemoveAttrOp.Key,
			})
		case PatchOpCodeRemoveStyle:
			if getRawOp.GetRemoveStyleOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove-style payload is required", parseOpIndex)
			}
			parseRemoveStyleOp, parseRemoveStyleErr := ParsePatchRemoveStyleOp(*getRawOp.GetRemoveStyleOp, buildKnownNodeIDs)
			if parseRemoveStyleErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove-style is invalid: %w", parseOpIndex, parseRemoveStyleErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:   RegionPatchOpKindRemoveStyle,
				GetNodeID: parseRemoveStyleOp.TargetNodeID,
			})
		case PatchOpCodeReplaceSubtree:
			if getRawOp.GetReplaceSubtreeOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d replace-subtree payload is required", parseOpIndex)
			}
			parseReplaceSubtreeOp, parseReplaceSubtreeErr := ParsePatchReplaceSubtreeOp(*getRawOp.GetReplaceSubtreeOp, buildKnownNodeIDs)
			if parseReplaceSubtreeErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d replace-subtree is invalid: %w", parseOpIndex, parseReplaceSubtreeErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:             RegionPatchOpKindReplaceSubtree,
				GetNodeID:           parseReplaceSubtreeOp.TargetNodeID,
				GetReplaceSubtreeIR: parseReplaceSubtreeOp.SubtreeIR,
			})
		case PatchOpCodeMoveKeyedChild:
			if getRawOp.GetKeyedMoveOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d keyed-move payload is required", parseOpIndex)
			}
			parseMoveOp, parseMoveErr := ParsePatchKeyedMoveOp(*getRawOp.GetKeyedMoveOp, buildKnownNodeIDs, buildSiblingCountByParent)
			if parseMoveErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d keyed-move is invalid: %w", parseOpIndex, parseMoveErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:             RegionPatchOpKindMoveKeyedNode,
				GetParentNodeID:     parseMoveOp.ParentNodeID,
				GetMoveNodeID:       parseMoveOp.SourceNodeID,
				GetDestinationIndex: parseMoveOp.DestinationIndex,
			})
		default:
			return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d has unsupported code %d", parseOpIndex, parseOpCode)
		}
	}
	return PatchStreamParseResult{
		GetHeader:      parseHeader,
		GetTransaction: buildTransaction,
	}, true, nil
}

// parseBuildPatchOrderEntries extracts patch-order entry payloads for structural ordering validation.
func parseBuildPatchOrderEntries(parseOps []PatchStreamOpRaw) ([]PatchOrderEntry, error) {
	buildEntries := make([]PatchOrderEntry, 0, len(parseOps))
	for parseOpIndex, getRawOp := range parseOps {
		parseOpCode, parseOpCodeErr := ParsePatchOpCode(getRawOp.GetOpCode)
		if parseOpCodeErr != nil {
			return nil, fmt.Errorf("runtime2: patch op %d has invalid op code: %w", parseOpIndex, parseOpCodeErr)
		}
		buildEntry := PatchOrderEntry{
			OpCode: parseOpCode,
		}
		switch parseOpCode {
		case PatchOpCodeInsertNode:
			if getRawOp.GetInsertOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d insert payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetInsertOp.Node.NodeID
			buildEntry.ParentNodeID = getRawOp.GetInsertOp.ParentNodeID
		case PatchOpCodeRemoveNode:
			if getRawOp.GetRemoveOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d remove payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetRemoveOp.TargetNodeID
		case PatchOpCodeSetText:
			if getRawOp.GetSetTextOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d set-text payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetSetTextOp.TargetNodeID
		case PatchOpCodeSetAttr:
			if getRawOp.GetSetAttrOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d set-attr payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetSetAttrOp.TargetNodeID
		case PatchOpCodeSetStyle:
			if getRawOp.GetSetStyleOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d set-style payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetSetStyleOp.TargetNodeID
		case PatchOpCodeRemoveAttr:
			if getRawOp.GetRemoveAttrOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d remove-attr payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetRemoveAttrOp.TargetNodeID
		case PatchOpCodeRemoveStyle:
			if getRawOp.GetRemoveStyleOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d remove-style payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetRemoveStyleOp.TargetNodeID
		case PatchOpCodeMoveKeyedChild:
			if getRawOp.GetKeyedMoveOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d keyed-move payload is required", parseOpIndex)
			}
			buildEntry.ParentNodeID = getRawOp.GetKeyedMoveOp.ParentNodeID
			buildEntry.SourceNodeID = getRawOp.GetKeyedMoveOp.SourceNodeID
		case PatchOpCodeReplaceSubtree:
			if getRawOp.GetReplaceSubtreeOp == nil {
				return nil, fmt.Errorf("runtime2: patch op %d replace-subtree payload is required", parseOpIndex)
			}
			buildEntry.NodeID = getRawOp.GetReplaceSubtreeOp.TargetNodeID
		}
		buildEntries = append(buildEntries, buildEntry)
	}
	return buildEntries, nil
}

// parseBuildRegionDOMNodeFromPatchRecord builds one DOM-index node from one parsed insert record.
func parseBuildRegionDOMNodeFromPatchRecord(parseRecord RenderNodeRecord, parseStringTable RenderStringTable) (*RegionDOMNode, error) {
	buildNode := &RegionDOMNode{
		GetNodeID:    parseRecord.NodeID,
		GetAttrByKey: map[string]string{},
		GetNodeKey:   parseRecord.KeyText,
	}
	switch parseRecord.Kind {
	case RenderNodeKindText:
		getText, getTextErr := parseStringTable.GetRenderStringByRef(parseRecord.TextRef)
		if getTextErr != nil {
			return nil, fmt.Errorf("runtime2: insert node text reference %d is invalid: %w", parseRecord.TextRef, getTextErr)
		}
		buildNode.GetText = getText
	case RenderNodeKindHostElement:
		getTag, getTagErr := parseStringTable.GetRenderStringByRef(parseRecord.TextRef)
		if getTagErr != nil {
			return nil, fmt.Errorf("runtime2: insert node tag reference %d is invalid: %w", parseRecord.TextRef, getTagErr)
		}
		if parseTagErr := ValidateWorkerRenderableHostTag(getTag); parseTagErr != nil {
			return nil, parseTagErr
		}
		buildNode.GetTag = getTag
	case RenderNodeKindFragment:
		buildNode.GetTag = ""
	default:
		return nil, fmt.Errorf("runtime2: insert node kind %v is unsupported", parseRecord.Kind)
	}
	return buildNode, nil
}

// BuildKnownNodeIDsForRegionDOMIndex extracts known node IDs for one region from the DOM index.
func BuildKnownNodeIDsForRegionDOMIndex(parseIndex *RegionDOMIndex, parseRegionID string) map[uint64]struct{} {
	buildKnownNodeIDs, _ := BuildRegionDOMPatchLookupMaps(parseIndex, parseRegionID)
	return buildKnownNodeIDs
}

// BuildRegionDOMPatchLookupMaps extracts known node IDs and sibling counts for one region in one pass.
func BuildRegionDOMPatchLookupMaps(parseIndex *RegionDOMIndex, parseRegionID string) (map[uint64]struct{}, map[uint64]uint32) {
	buildKnownNodeIDs := map[uint64]struct{}{}
	buildSiblingCountByParent := map[uint64]uint32{}
	if parseIndex == nil || strings.TrimSpace(parseRegionID) == "" {
		return buildKnownNodeIDs, buildSiblingCountByParent
	}
	getRegionNodeByID, hasRegionNodeByID := parseIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionNodeByID {
		return buildKnownNodeIDs, buildSiblingCountByParent
	}
	buildKnownNodeIDs = make(map[uint64]struct{}, len(getRegionNodeByID))
	buildSiblingCountByParent = make(map[uint64]uint32, len(getRegionNodeByID))
	for getNodeID, getNode := range getRegionNodeByID {
		buildKnownNodeIDs[getNodeID] = struct{}{}
		if getNode == nil {
			continue
		}
		buildSiblingCountByParent[getNode.GetNodeID] = uint32(len(getNode.GetChildNodeIDs))
	}
	return buildKnownNodeIDs, buildSiblingCountByParent
}

// BuildSiblingCountByParentForRegionDOMIndex extracts sibling counts keyed by parent node ID for one region.
func BuildSiblingCountByParentForRegionDOMIndex(parseIndex *RegionDOMIndex, parseRegionID string) map[uint64]uint32 {
	_, buildSiblingCountByParent := BuildRegionDOMPatchLookupMaps(parseIndex, parseRegionID)
	return buildSiblingCountByParent
}

type patchInsertBuildOp struct {
	getParentNodeID uint64
	getNodeID       uint64
	getNodeKind     RenderNodeKind
	getNodeTag      string
	getNodeText     string
	getNodeKey      string
	getAnchorNodeID uint64
}

type patchSetTextBuildOp struct {
	getTargetNodeID uint64
	getText         string
}

type patchSetAttrBuildOp struct {
	getTargetNodeID uint64
	getKind         RenderPropKind
	getKey          string
	getValue        string
}

type patchSetStyleBuildOp struct {
	getTargetNodeID uint64
	getStyleValue   string
}

type patchRemoveAttrBuildOp struct {
	getTargetNodeID uint64
	getKey          string
}

type patchRemoveStyleBuildOp struct {
	getTargetNodeID uint64
}

// BuildCanonicalPatchStream diffs previous and next canonical IR and emits one canonical typed patch stream.
func BuildCanonicalPatchStream(
	parseRegionID string,
	parseEpoch uint64,
	parseInputVersion uint64,
	parsePatchVersion uint64,
	parsePreviousIR CanonicalRenderIR,
	parseNextIR CanonicalRenderIR,
) (PatchStreamRaw, bool, error) {
	if strings.TrimSpace(parseRegionID) == "" {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream region ID is required")
	}
	if parseEpoch == 0 {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream epoch is required")
	}
	if parseInputVersion == 0 {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream input version is required")
	}
	if parsePatchVersion == 0 {
		return PatchStreamRaw{}, false, fmt.Errorf("runtime2: patch stream patch version is required")
	}
	if IsCanonicalRenderIREqual(parsePreviousIR, parseNextIR) {
		return PatchStreamRaw{}, true, nil
	}
	parsePreviousTree, parsePreviousTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
	if parsePreviousTreeErr != nil {
		return PatchStreamRaw{}, false, parsePreviousTreeErr
	}
	parseNextTree, parseNextTreeErr := ParseCanonicalRenderTree(parseNextIR)
	if parseNextTreeErr != nil {
		return PatchStreamRaw{}, false, parseNextTreeErr
	}
	hasStructuralMismatch := false
	for getNodeID, getPreviousNode := range parsePreviousTree.getNodeByID {
		getNextNode, hasNextNode := parseNextTree.getNodeByID[getNodeID]
		if !hasNextNode {
			continue
		}
		if getPreviousNode.getKind != getNextNode.getKind ||
			(getPreviousNode.getKind == RenderNodeKindHostElement && getPreviousNode.getTag != getNextNode.getTag) {
			hasStructuralMismatch = true
			break
		}
	}
	if hasStructuralMismatch {
		buildReplaceSubtreeOp := PatchReplaceSubtreeOpRaw{
			TargetNodeID: parsePreviousTree.getRootNodeID,
			Subtree:      parseBuildReplaceSubtreePayload(parseNextIR),
		}
		buildPatchStreamRaw, parsePatchStreamErr := BuildPatchStreamRaw(PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        parseRegionID,
			Epoch:           parseEpoch,
			InputVersion:    parseInputVersion,
			PatchVersion:    parsePatchVersion,
		}, RenderStringTable{}, []PatchStreamOpRaw{
			{
				GetOpCode:           uint8(PatchOpCodeReplaceSubtree),
				GetReplaceSubtreeOp: &buildReplaceSubtreeOp,
			},
		})
		if parsePatchStreamErr != nil {
			return PatchStreamRaw{}, false, parsePatchStreamErr
		}
		return buildPatchStreamRaw, false, nil
	}
	buildRemovedNodeIDs := make(map[uint64]struct{})
	buildInsertedNodeIDs := make(map[uint64]struct{})
	for getNodeID := range parsePreviousTree.getNodeByID {
		if _, hasNodeID := parseNextTree.getNodeByID[getNodeID]; !hasNodeID {
			buildRemovedNodeIDs[getNodeID] = struct{}{}
		}
	}
	for getNodeID := range parseNextTree.getNodeByID {
		if _, hasNodeID := parsePreviousTree.getNodeByID[getNodeID]; !hasNodeID {
			buildInsertedNodeIDs[getNodeID] = struct{}{}
		}
	}
	buildRemoveNodeIDs := make([]uint64, 0, len(buildRemovedNodeIDs))
	for getNodeID := range buildRemovedNodeIDs {
		if getNodeID == parsePreviousTree.getRootNodeID {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: root structural replacement is unsupported in first-slice patch diff")
		}
		buildRemoveNodeIDs = append(buildRemoveNodeIDs, getNodeID)
	}
	sort.Slice(buildRemoveNodeIDs, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftDepth := parsePreviousTree.GetCanonicalNodeDepth(buildRemoveNodeIDs[parseLeftIndex])
		parseRightDepth := parsePreviousTree.GetCanonicalNodeDepth(buildRemoveNodeIDs[parseRightIndex])
		if parseLeftDepth == parseRightDepth {
			return buildRemoveNodeIDs[parseLeftIndex] < buildRemoveNodeIDs[parseRightIndex]
		}
		return parseLeftDepth > parseRightDepth
	})
	buildInsertNodeIDs := make([]uint64, 0, len(buildInsertedNodeIDs))
	for getNodeID := range buildInsertedNodeIDs {
		buildInsertNodeIDs = append(buildInsertNodeIDs, getNodeID)
	}
	sort.Slice(buildInsertNodeIDs, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftNodeID := buildInsertNodeIDs[parseLeftIndex]
		parseRightNodeID := buildInsertNodeIDs[parseRightIndex]
		parseLeftDepth := parseNextTree.GetCanonicalNodeDepth(parseLeftNodeID)
		parseRightDepth := parseNextTree.GetCanonicalNodeDepth(parseRightNodeID)
		if parseLeftDepth != parseRightDepth {
			return parseLeftDepth < parseRightDepth
		}
		parseLeftNode := parseNextTree.getNodeByID[parseLeftNodeID]
		parseRightNode := parseNextTree.getNodeByID[parseRightNodeID]
		if parseLeftNode.getParentNodeID != parseRightNode.getParentNodeID {
			return parseLeftNode.getParentNodeID < parseRightNode.getParentNodeID
		}
		parseLeftSiblingIndex := parseFindCanonicalSiblingIndex(parseNextTree.getNodeByID[parseLeftNode.getParentNodeID].getChildNodeIDs, parseLeftNodeID)
		parseRightSiblingIndex := parseFindCanonicalSiblingIndex(parseNextTree.getNodeByID[parseRightNode.getParentNodeID].getChildNodeIDs, parseRightNodeID)
		if parseLeftSiblingIndex == parseRightSiblingIndex {
			return parseLeftNodeID < parseRightNodeID
		}
		return parseLeftSiblingIndex > parseRightSiblingIndex
	})
	buildInsertOps := make([]patchInsertBuildOp, 0, len(buildInsertNodeIDs))
	buildSetTextOps := []patchSetTextBuildOp{}
	buildSetAttrOps := []patchSetAttrBuildOp{}
	buildSetStyleOps := []patchSetStyleBuildOp{}
	buildRemoveAttrOps := []patchRemoveAttrBuildOp{}
	buildRemoveStyleOps := []patchRemoveStyleBuildOp{}
	for _, getNodeID := range buildInsertNodeIDs {
		getNextNode := parseNextTree.getNodeByID[getNodeID]
		if getNodeID == parseNextTree.getRootNodeID || getNextNode.getParentNodeID == 0 {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: root structural replacement is unsupported in first-slice patch diff")
		}
		buildAnchorNodeID := parseFindCanonicalInsertAnchor(getNextNode.getParentNodeID, getNodeID, parsePreviousTree, parseNextTree, buildInsertedNodeIDs)
		buildInsertOps = append(buildInsertOps, patchInsertBuildOp{
			getParentNodeID: getNextNode.getParentNodeID,
			getNodeID:       getNodeID,
			getNodeKind:     getNextNode.getKind,
			getNodeTag:      getNextNode.getTag,
			getNodeText:     getNextNode.getText,
			getNodeKey:      getNextNode.getKey,
			getAnchorNodeID: buildAnchorNodeID,
		})
		for _, getProp := range getNextNode.getPropByKey {
			if getProp.Kind == RenderPropKindTextAdjacent {
				continue
			}
			if getProp.Kind == RenderPropKindStyle {
				buildSetStyleOps = append(buildSetStyleOps, patchSetStyleBuildOp{
					getTargetNodeID: getNodeID,
					getStyleValue:   getProp.Value,
				})
				continue
			}
			buildSetAttrOps = append(buildSetAttrOps, patchSetAttrBuildOp{
				getTargetNodeID: getNodeID,
				getKind:         getProp.Kind,
				getKey:          getProp.Key,
				getValue:        getProp.Value,
			})
		}
	}
	for getNodeID, getPreviousNode := range parsePreviousTree.getNodeByID {
		getNextNode, hasNextNode := parseNextTree.getNodeByID[getNodeID]
		if !hasNextNode {
			continue
		}
		if _, hasRemovedNode := buildRemovedNodeIDs[getNodeID]; hasRemovedNode {
			continue
		}
		if _, hasInsertedNode := buildInsertedNodeIDs[getNodeID]; hasInsertedNode {
			continue
		}
		if getPreviousNode.getKind == RenderNodeKindText && getPreviousNode.getText != getNextNode.getText {
			buildSetTextOps = append(buildSetTextOps, patchSetTextBuildOp{
				getTargetNodeID: getNodeID,
				getText:         getNextNode.getText,
			})
		}
		for getPropKey, getNextProp := range getNextNode.getPropByKey {
			if getNextProp.Kind == RenderPropKindTextAdjacent {
				continue
			}
			getPreviousProp, hasPreviousProp := getPreviousNode.getPropByKey[getPropKey]
			if !hasPreviousProp || getPreviousProp.Value != getNextProp.Value || getPreviousProp.Kind != getNextProp.Kind {
				if getNextProp.Kind == RenderPropKindStyle {
					buildSetStyleOps = append(buildSetStyleOps, patchSetStyleBuildOp{
						getTargetNodeID: getNodeID,
						getStyleValue:   getNextProp.Value,
					})
					continue
				}
				buildSetAttrOps = append(buildSetAttrOps, patchSetAttrBuildOp{
					getTargetNodeID: getNodeID,
					getKind:         getNextProp.Kind,
					getKey:          getNextProp.Key,
					getValue:        getNextProp.Value,
				})
			}
		}
		for getPropKey, getPreviousProp := range getPreviousNode.getPropByKey {
			if getPreviousProp.Kind == RenderPropKindTextAdjacent {
				continue
			}
			if _, hasNextProp := getNextNode.getPropByKey[getPropKey]; !hasNextProp {
				if getPreviousProp.Kind == RenderPropKindStyle {
					buildRemoveStyleOps = append(buildRemoveStyleOps, patchRemoveStyleBuildOp{
						getTargetNodeID: getNodeID,
					})
					continue
				}
				buildRemoveAttrOps = append(buildRemoveAttrOps, patchRemoveAttrBuildOp{
					getTargetNodeID: getNodeID,
					getKey:          getPropKey,
				})
			}
		}
	}
	buildMoveOps := []PatchKeyedMoveOpRaw{}
	hasLoggedMoveSiblingWarn := false
	for getParentNodeID, getPreviousNode := range parsePreviousTree.getNodeByID {
		getNextNode, hasNextNode := parseNextTree.getNodeByID[getParentNodeID]
		if !hasNextNode {
			continue
		}
		buildCurrentOrder := parseFilterCanonicalExistingOrder(getPreviousNode.getChildNodeIDs, buildRemovedNodeIDs, buildInsertedNodeIDs)
		buildTargetOrder := parseFilterCanonicalExistingOrder(getNextNode.getChildNodeIDs, buildRemovedNodeIDs, buildInsertedNodeIDs)
		if len(buildTargetOrder) > getPatchMoveSiblingHardLimit {
			return PatchStreamRaw{}, false, fmt.Errorf(
				"runtime2: keyed-move sibling count %d exceeds guard limit %d for parent %d",
				len(buildTargetOrder),
				getPatchMoveSiblingHardLimit,
				getParentNodeID,
			)
		}
		if len(buildTargetOrder) > getPatchMoveSiblingWarnLimit && !hasLoggedMoveSiblingWarn {
			hasLoggedMoveSiblingWarn = true
			fmt.Printf(
				"WARN: runtime2 patch diff region=%s keyed-move sibling count=%d exceeds soft limit=%d\n",
				parseRegionID,
				len(buildTargetOrder),
				getPatchMoveSiblingWarnLimit,
			)
		}
		for parseTargetIndex, getTargetNodeID := range buildTargetOrder {
			getTargetNode := parseNextTree.getNodeByID[getTargetNodeID]
			if strings.TrimSpace(getTargetNode.getKey) == "" {
				continue
			}
			parseCurrentIndex := parseFindCanonicalSiblingIndex(buildCurrentOrder, getTargetNodeID)
			if parseCurrentIndex < 0 || parseCurrentIndex == parseTargetIndex {
				continue
			}
			buildCurrentOrder = parseMoveCanonicalNodeID(buildCurrentOrder, parseCurrentIndex, parseTargetIndex)
			buildMoveOps = append(buildMoveOps, PatchKeyedMoveOpRaw{
				ParentNodeID:     getParentNodeID,
				SourceNodeID:     getTargetNodeID,
				DestinationIndex: uint32(parseTargetIndex),
			})
		}
	}
	sort.Slice(buildSetTextOps, func(parseLeftIndex int, parseRightIndex int) bool {
		return buildSetTextOps[parseLeftIndex].getTargetNodeID < buildSetTextOps[parseRightIndex].getTargetNodeID
	})
	sort.Slice(buildSetAttrOps, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftOp := buildSetAttrOps[parseLeftIndex]
		parseRightOp := buildSetAttrOps[parseRightIndex]
		if parseLeftOp.getTargetNodeID == parseRightOp.getTargetNodeID {
			return parseLeftOp.getKey < parseRightOp.getKey
		}
		return parseLeftOp.getTargetNodeID < parseRightOp.getTargetNodeID
	})
	sort.Slice(buildSetStyleOps, func(parseLeftIndex int, parseRightIndex int) bool {
		return buildSetStyleOps[parseLeftIndex].getTargetNodeID < buildSetStyleOps[parseRightIndex].getTargetNodeID
	})
	sort.Slice(buildRemoveAttrOps, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftOp := buildRemoveAttrOps[parseLeftIndex]
		parseRightOp := buildRemoveAttrOps[parseRightIndex]
		if parseLeftOp.getTargetNodeID == parseRightOp.getTargetNodeID {
			return parseLeftOp.getKey < parseRightOp.getKey
		}
		return parseLeftOp.getTargetNodeID < parseRightOp.getTargetNodeID
	})
	sort.Slice(buildRemoveStyleOps, func(parseLeftIndex int, parseRightIndex int) bool {
		return buildRemoveStyleOps[parseLeftIndex].getTargetNodeID < buildRemoveStyleOps[parseRightIndex].getTargetNodeID
	})
	sort.Slice(buildMoveOps, func(parseLeftIndex int, parseRightIndex int) bool {
		parseLeftOp := buildMoveOps[parseLeftIndex]
		parseRightOp := buildMoveOps[parseRightIndex]
		if parseLeftOp.ParentNodeID == parseRightOp.ParentNodeID {
			if parseLeftOp.DestinationIndex == parseRightOp.DestinationIndex {
				return parseLeftOp.SourceNodeID < parseRightOp.SourceNodeID
			}
			return parseLeftOp.DestinationIndex < parseRightOp.DestinationIndex
		}
		return parseLeftOp.ParentNodeID < parseRightOp.ParentNodeID
	})
	buildPatchStrings := []string{}
	for _, getInsertOp := range buildInsertOps {
		if getInsertOp.getNodeKind == RenderNodeKindText {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeText)
		}
		if getInsertOp.getNodeKind == RenderNodeKindHostElement {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeTag)
		}
		if strings.TrimSpace(getInsertOp.getNodeKey) != "" {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeKey)
		}
	}
	for _, getSetTextOp := range buildSetTextOps {
		buildPatchStrings = append(buildPatchStrings, getSetTextOp.getText)
	}
	for _, getSetAttrOp := range buildSetAttrOps {
		buildPatchStrings = append(buildPatchStrings, getSetAttrOp.getKey, getSetAttrOp.getValue)
	}
	for _, getSetStyleOp := range buildSetStyleOps {
		buildPatchStrings = append(buildPatchStrings, getSetStyleOp.getStyleValue)
	}
	for _, getRemoveAttrOp := range buildRemoveAttrOps {
		buildPatchStrings = append(buildPatchStrings, getRemoveAttrOp.getKey)
	}
	buildPatchStringTable := BuildRenderStringTable(buildPatchStrings)
	buildOps := make([]PatchStreamOpRaw, 0, len(buildRemoveNodeIDs)+len(buildInsertOps)+len(buildSetTextOps)+len(buildSetAttrOps)+len(buildSetStyleOps)+len(buildRemoveAttrOps)+len(buildRemoveStyleOps)+len(buildMoveOps))
	for _, getNodeID := range buildRemoveNodeIDs {
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveNode),
			GetRemoveOp: &PatchRemoveOpRaw{
				TargetNodeID: getNodeID,
			},
		})
	}
	for _, getInsertOp := range buildInsertOps {
		buildNodeTextRef := uint32(0)
		switch getInsertOp.getNodeKind {
		case RenderNodeKindText:
			getTextRef, hasTextRef := buildPatchStringTable.GetRenderStringRef(getInsertOp.getNodeText)
			if !hasTextRef {
				return PatchStreamRaw{}, false, fmt.Errorf("runtime2: insert text %q missing from patch table", getInsertOp.getNodeText)
			}
			buildNodeTextRef = getTextRef
		case RenderNodeKindHostElement:
			getTagRef, hasTagRef := buildPatchStringTable.GetRenderStringRef(getInsertOp.getNodeTag)
			if !hasTagRef {
				return PatchStreamRaw{}, false, fmt.Errorf("runtime2: insert tag %q missing from patch table", getInsertOp.getNodeTag)
			}
			buildNodeTextRef = getTagRef
		}
		buildNodeRaw := RenderNodeRecordRaw{
			NodeID:  getInsertOp.getNodeID,
			Kind:    uint8(getInsertOp.getNodeKind),
			TextRef: buildNodeTextRef,
		}
		if strings.TrimSpace(getInsertOp.getNodeKey) != "" {
			buildNodeRaw.KeyText = getInsertOp.getNodeKey
			buildNodeRaw.KeyHash = parseHashCanonicalString("key:"+getInsertOp.getNodeKey, nil)
		}
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeInsertNode),
			GetInsertOp: &PatchInsertOpRaw{
				ParentNodeID: getInsertOp.getParentNodeID,
				AnchorNodeID: getInsertOp.getAnchorNodeID,
				Node:         buildNodeRaw,
			},
		})
	}
	for _, getSetTextOp := range buildSetTextOps {
		getTextRef, hasTextRef := buildPatchStringTable.GetRenderStringRef(getSetTextOp.getText)
		if !hasTextRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: set-text payload %q missing from patch table", getSetTextOp.getText)
		}
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetText),
			GetSetTextOp: &PatchSetTextOpRaw{
				TargetNodeID: getSetTextOp.getTargetNodeID,
				TextRef:      getTextRef,
			},
		})
	}
	for _, getSetAttrOp := range buildSetAttrOps {
		getKeyRef, hasKeyRef := buildPatchStringTable.GetRenderStringRef(getSetAttrOp.getKey)
		if !hasKeyRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: set-attr key %q missing from patch table", getSetAttrOp.getKey)
		}
		getValueRef, hasValueRef := buildPatchStringTable.GetRenderStringRef(getSetAttrOp.getValue)
		if !hasValueRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: set-attr value %q missing from patch table", getSetAttrOp.getValue)
		}
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetAttr),
			GetSetAttrOp: &PatchSetAttrOpRaw{
				TargetNodeID: getSetAttrOp.getTargetNodeID,
				Attr: RenderPropRecordRaw{
					Kind:     uint8(getSetAttrOp.getKind),
					KeyRef:   getKeyRef,
					ValueRef: getValueRef,
				},
			},
		})
	}
	for _, getSetStyleOp := range buildSetStyleOps {
		getStyleRef, hasStyleRef := buildPatchStringTable.GetRenderStringRef(getSetStyleOp.getStyleValue)
		if !hasStyleRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: set-style payload %q missing from patch table", getSetStyleOp.getStyleValue)
		}
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetStyle),
			GetSetStyleOp: &PatchSetStyleOpRaw{
				TargetNodeID: getSetStyleOp.getTargetNodeID,
				StyleRef:     getStyleRef,
			},
		})
	}
	for _, getRemoveAttrOp := range buildRemoveAttrOps {
		getKeyRef, hasKeyRef := buildPatchStringTable.GetRenderStringRef(getRemoveAttrOp.getKey)
		if !hasKeyRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: remove-attr key %q missing from patch table", getRemoveAttrOp.getKey)
		}
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveAttr),
			GetRemoveAttrOp: &PatchRemoveAttrOpRaw{
				TargetNodeID: getRemoveAttrOp.getTargetNodeID,
				KeyRef:       getKeyRef,
			},
		})
	}
	for _, getRemoveStyleOp := range buildRemoveStyleOps {
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveStyle),
			GetRemoveStyleOp: &PatchRemoveStyleOpRaw{
				TargetNodeID: getRemoveStyleOp.getTargetNodeID,
			},
		})
	}
	for _, getMoveOp := range buildMoveOps {
		buildMoveOp := getMoveOp
		buildOps = append(buildOps, PatchStreamOpRaw{
			GetOpCode:      uint8(PatchOpCodeMoveKeyedChild),
			GetKeyedMoveOp: &buildMoveOp,
		})
	}
	getPatchStreamRaw, parsePatchStreamErr := BuildPatchStreamRaw(PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        parseRegionID,
		Epoch:           parseEpoch,
		InputVersion:    parseInputVersion,
		PatchVersion:    parsePatchVersion,
	}, buildPatchStringTable, buildOps)
	if parsePatchStreamErr != nil {
		return PatchStreamRaw{}, false, parsePatchStreamErr
	}
	return getPatchStreamRaw, false, nil
}

// parseFindCanonicalInsertAnchor resolves the next stable sibling anchor for one inserted node.
func parseFindCanonicalInsertAnchor(
	parseParentNodeID uint64,
	parseNodeID uint64,
	parsePreviousTree canonicalRenderTree,
	parseNextTree canonicalRenderTree,
	parseInsertedNodeIDs map[uint64]struct{},
) uint64 {
	parseParentNode, hasParentNode := parseNextTree.getNodeByID[parseParentNodeID]
	if !hasParentNode {
		return 0
	}
	parseSiblingIndex := parseFindCanonicalSiblingIndex(parseParentNode.getChildNodeIDs, parseNodeID)
	if parseSiblingIndex < 0 {
		return 0
	}
	for parseIndex := parseSiblingIndex + 1; parseIndex < len(parseParentNode.getChildNodeIDs); parseIndex++ {
		getSiblingNodeID := parseParentNode.getChildNodeIDs[parseIndex]
		if _, hasInsertedSibling := parseInsertedNodeIDs[getSiblingNodeID]; hasInsertedSibling {
			continue
		}
		if _, hasPreviousSibling := parsePreviousTree.getNodeByID[getSiblingNodeID]; hasPreviousSibling {
			return getSiblingNodeID
		}
	}
	return 0
}

// parseFindCanonicalSiblingIndex returns one child index lookup for one node ID.
func parseFindCanonicalSiblingIndex(parseChildNodeIDs []uint64, parseNodeID uint64) int {
	for parseIndex, getNodeID := range parseChildNodeIDs {
		if getNodeID == parseNodeID {
			return parseIndex
		}
	}
	return -1
}

// parseMoveCanonicalNodeID reorders one node ID from one index into another.
func parseMoveCanonicalNodeID(parseNodeIDs []uint64, parseFromIndex int, parseToIndex int) []uint64 {
	if parseFromIndex < 0 || parseFromIndex >= len(parseNodeIDs) {
		return parseNodeIDs
	}
	if parseToIndex < 0 {
		parseToIndex = 0
	}
	if parseToIndex > len(parseNodeIDs) {
		parseToIndex = len(parseNodeIDs)
	}
	buildNodeIDs := append([]uint64(nil), parseNodeIDs...)
	getNodeID := buildNodeIDs[parseFromIndex]
	buildNodeIDs = append(buildNodeIDs[:parseFromIndex], buildNodeIDs[parseFromIndex+1:]...)
	if parseToIndex > len(buildNodeIDs) {
		parseToIndex = len(buildNodeIDs)
	}
	buildNodeIDs = append(buildNodeIDs, 0)
	copy(buildNodeIDs[parseToIndex+1:], buildNodeIDs[parseToIndex:])
	buildNodeIDs[parseToIndex] = getNodeID
	return buildNodeIDs
}

// parseFilterCanonicalExistingOrder filters one child order to IDs that remain in-place candidates.
func parseFilterCanonicalExistingOrder(parseNodeIDs []uint64, parseRemovedNodeIDs map[uint64]struct{}, parseInsertedNodeIDs map[uint64]struct{}) []uint64 {
	buildNodeIDs := make([]uint64, 0, len(parseNodeIDs))
	for _, getNodeID := range parseNodeIDs {
		if _, hasRemovedNode := parseRemovedNodeIDs[getNodeID]; hasRemovedNode {
			continue
		}
		if _, hasInsertedNode := parseInsertedNodeIDs[getNodeID]; hasInsertedNode {
			continue
		}
		buildNodeIDs = append(buildNodeIDs, getNodeID)
	}
	return buildNodeIDs
}

// parseBuildReplaceSubtreePayload builds one replace-subtree payload from canonical render IR.
func parseBuildReplaceSubtreePayload(parseIR CanonicalRenderIR) PatchReplaceSubtreePayloadRaw {
	return PatchReplaceSubtreePayloadRaw{
		RootNodeID:  parseIR.GetRootNodeID,
		StringTable: append([]string(nil), parseIR.GetStringTable.Entries...),
		NodeRecords: append([]RenderNodeRecordRaw(nil), parseIR.GetNodeRecords...),
		PropRecords: append([]RenderPropRecordRaw(nil), parseIR.GetPropRecords...),
	}
}
