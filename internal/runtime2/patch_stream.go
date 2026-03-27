package runtime2

import (
	"fmt"
	"hash"
	"hash/fnv"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	getPatchStreamOpWarnLimit    = 2048
	getPatchStreamOpHardLimit    = 16384
	getPatchMoveSiblingWarnLimit = 512
	getPatchMoveSiblingHardLimit = 8192
)

var (
	writePatchStreamIdentityTokenHeader      = []byte(`{"GetHeader":`)
	writePatchStreamIdentityTokenStringTable = []byte(`,"GetStringTable":`)
	writePatchStreamIdentityTokenOps         = []byte(`,"GetOps":`)
	writePatchStreamIdentityTokenArrayOpen   = []byte(`[`)
	writePatchStreamIdentityTokenArrayClose  = []byte(`]}`)
	writePatchStreamIdentityTokenNullClose   = []byte(`null}`)
	writePatchStreamIdentityTokenComma       = []byte(`,`)
)

var storePatchStreamIdentityHasherPool = sync.Pool{
	New: func() any {
		return fnv.New64a()
	},
}

var storePatchStreamKnownNodeIDScratchPool = sync.Pool{
	New: func() any {
		return make(map[uint64]struct{})
	},
}

var storePatchStreamSiblingCountScratchPool = sync.Pool{
	New: func() any {
		return make(map[uint64]uint32)
	},
}

var storePatchStreamRemovedNodeIDScratchPool = sync.Pool{
	New: func() any {
		return make(map[uint64]struct{})
	},
}

var storePatchStreamRemovedAttrKeyScratchPool = sync.Pool{
	New: func() any {
		return make(map[string]struct{})
	},
}

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

type patchStreamIdentityState struct {
	getHasher       hash.Hash64
	hasStartedArray bool
	hasOp           bool
}

type canonicalSiblingIndexCache struct {
	getTree                canonicalRenderTree
	getSiblingIndexByParen map[uint64]map[uint64]int
}

// BuildPatchStreamRaw builds one typed patch stream and computes deterministic idempotency metadata.
func BuildPatchStreamRaw(parseHeader PatchStreamHeaderRaw, parseStringTable RenderStringTable, parseOps []PatchStreamOpRaw) (PatchStreamRaw, error) {
	buildStringTable := append([]string(nil), parseStringTable.Entries...)
	buildPatchIdentity, parseIdentityErr := buildPatchStreamIdentityFromParts(parseHeader, buildStringTable, parseOps)
	if parseIdentityErr != nil {
		return PatchStreamRaw{}, parseIdentityErr
	}
	return buildPatchStreamRawWithIdentity(parseHeader, buildStringTable, parseOps, buildPatchIdentity)
}

// BuildPatchStreamIdentity computes one deterministic patch identity used by idempotency tracking.
func BuildPatchStreamIdentity(parseRaw PatchStreamRaw) (string, error) {
	if parseRuntimeHasTrimmedNonWhitespaceText(parseRaw.GetPatchIdentity) {
		return parseRaw.GetPatchIdentity, nil
	}
	return buildPatchStreamIdentityFromParts(parseRaw.GetHeader, parseRaw.GetStringTable, parseRaw.GetOps)
}

// buildPatchStreamRawWithIdentity builds one typed patch stream from prevalidated identity parts.
func buildPatchStreamRawWithIdentity(parseHeader PatchStreamHeaderRaw, parseStringTable []string, parseOps []PatchStreamOpRaw, parsePatchIdentity string) (PatchStreamRaw, error) {
	if _, parseHeaderErr := ParsePatchStreamHeader(parseHeader, parseHeader.RegionID); parseHeaderErr != nil {
		return PatchStreamRaw{}, parseHeaderErr
	}
	return PatchStreamRaw{
		GetHeader:        parseHeader,
		GetStringTable:   append([]string(nil), parseStringTable...),
		GetOps:           append([]PatchStreamOpRaw(nil), parseOps...),
		GetPatchIdentity: parsePatchIdentity,
	}, nil
}

// buildPatchStreamIdentityFromParts computes one deterministic patch identity from raw header, string-table, and op slices.
func buildPatchStreamIdentityFromParts(parseHeader PatchStreamHeaderRaw, parseStringTable []string, parseOps []PatchStreamOpRaw) (string, error) {
	buildState, parseStateErr := buildPatchStreamIdentityState(parseHeader, parseStringTable)
	if parseStateErr != nil {
		return "", parseStateErr
	}
	defer clearPatchStreamIdentityState(&buildState)
	if parseOps != nil {
		if parseArrayErr := startPatchStreamIdentityOpArray(&buildState); parseArrayErr != nil {
			return "", parseArrayErr
		}
		for _, getOp := range parseOps {
			if parseOpErr := appendPatchStreamIdentityOp(&buildState, getOp); parseOpErr != nil {
				return "", parseOpErr
			}
		}
	}
	return formatPatchStreamIdentityState(&buildState)
}

// buildPatchStreamIdentityState starts one incremental patch-identity writer up through the GetOps field prefix.
func buildPatchStreamIdentityState(parseHeader PatchStreamHeaderRaw, parseStringTable []string) (patchStreamIdentityState, error) {
	buildHasher := buildPatchStreamIdentityHasher()
	if buildHasher == nil {
		return patchStreamIdentityState{}, fmt.Errorf("runtime2: patch identity hasher is nil")
	}
	buildHasher.Reset()
	buildState := patchStreamIdentityState{
		getHasher: buildHasher,
	}
	if parseTokenErr := writeJSONHashLiteral(buildState.getHasher, writePatchStreamIdentityTokenHeader); parseTokenErr != nil {
		clearPatchStreamIdentityState(&buildState)
		return patchStreamIdentityState{}, fmt.Errorf("runtime2: write patch identity header token: %w", parseTokenErr)
	}
	if parseHeaderErr := buildJSONHashDigest(buildState.getHasher, parseHeader); parseHeaderErr != nil {
		clearPatchStreamIdentityState(&buildState)
		return patchStreamIdentityState{}, fmt.Errorf("runtime2: encode patch identity header: %w", parseHeaderErr)
	}
	if parseTokenErr := writeJSONHashLiteral(buildState.getHasher, writePatchStreamIdentityTokenStringTable); parseTokenErr != nil {
		clearPatchStreamIdentityState(&buildState)
		return patchStreamIdentityState{}, fmt.Errorf("runtime2: write patch identity string table token: %w", parseTokenErr)
	}
	if parseStringTableErr := buildJSONHashDigest(buildState.getHasher, parseStringTable); parseStringTableErr != nil {
		clearPatchStreamIdentityState(&buildState)
		return patchStreamIdentityState{}, fmt.Errorf("runtime2: encode patch identity string table: %w", parseStringTableErr)
	}
	if parseTokenErr := writeJSONHashLiteral(buildState.getHasher, writePatchStreamIdentityTokenOps); parseTokenErr != nil {
		clearPatchStreamIdentityState(&buildState)
		return patchStreamIdentityState{}, fmt.Errorf("runtime2: write patch identity ops token: %w", parseTokenErr)
	}
	return buildState, nil
}

// startPatchStreamIdentityOpArray writes the GetOps array opening token for one incremental patch-identity stream.
func startPatchStreamIdentityOpArray(parseState *patchStreamIdentityState) error {
	if parseState == nil || parseState.getHasher == nil {
		return fmt.Errorf("runtime2: patch identity state is nil")
	}
	if parseState.hasStartedArray {
		return nil
	}
	if parseTokenErr := writeJSONHashLiteral(parseState.getHasher, writePatchStreamIdentityTokenArrayOpen); parseTokenErr != nil {
		return fmt.Errorf("runtime2: write patch identity op-array token: %w", parseTokenErr)
	}
	parseState.hasStartedArray = true
	return nil
}

// appendPatchStreamIdentityOp appends one patch op into the incremental patch-identity stream.
func appendPatchStreamIdentityOp(parseState *patchStreamIdentityState, parseOp PatchStreamOpRaw) error {
	if parseState == nil || parseState.getHasher == nil {
		return fmt.Errorf("runtime2: patch identity state is nil")
	}
	if !parseState.hasStartedArray {
		return fmt.Errorf("runtime2: patch identity op array is not started")
	}
	if parseState.hasOp {
		if parseTokenErr := writeJSONHashLiteral(parseState.getHasher, writePatchStreamIdentityTokenComma); parseTokenErr != nil {
			return fmt.Errorf("runtime2: write patch identity op separator: %w", parseTokenErr)
		}
	}
	if parseOpErr := buildJSONHashDigest(parseState.getHasher, parseOp); parseOpErr != nil {
		return fmt.Errorf("runtime2: encode patch identity op: %w", parseOpErr)
	}
	parseState.hasOp = true
	return nil
}

// formatPatchStreamIdentityState finalizes one incremental patch-identity stream and returns the deterministic digest.
func formatPatchStreamIdentityState(parseState *patchStreamIdentityState) (string, error) {
	if parseState == nil || parseState.getHasher == nil {
		return "", fmt.Errorf("runtime2: patch identity state is nil")
	}
	if parseState.hasStartedArray {
		if parseTokenErr := writeJSONHashLiteral(parseState.getHasher, writePatchStreamIdentityTokenArrayClose); parseTokenErr != nil {
			return "", fmt.Errorf("runtime2: write patch identity close token: %w", parseTokenErr)
		}
	} else {
		if parseTokenErr := writeJSONHashLiteral(parseState.getHasher, writePatchStreamIdentityTokenNullClose); parseTokenErr != nil {
			return "", fmt.Errorf("runtime2: write patch identity null close token: %w", parseTokenErr)
		}
	}
	return strconv.FormatUint(parseState.getHasher.Sum64(), 16), nil
}

// clearPatchStreamIdentityState resets one incremental patch-identity stream and returns its hasher to the pool.
func clearPatchStreamIdentityState(parseState *patchStreamIdentityState) {
	if parseState == nil || parseState.getHasher == nil {
		return
	}
	parseState.getHasher.Reset()
	storePatchStreamIdentityHasherPool.Put(parseState.getHasher)
	parseState.getHasher = nil
	parseState.hasStartedArray = false
	parseState.hasOp = false
}

// appendPatchStreamOpWithIdentity appends one patch op to the output slice while updating the incremental patch identity.
func appendPatchStreamOpWithIdentity(parseOps *[]PatchStreamOpRaw, parseState *patchStreamIdentityState, parseOp PatchStreamOpRaw) error {
	if parseOps == nil {
		return fmt.Errorf("runtime2: patch op destination is nil")
	}
	*parseOps = append(*parseOps, parseOp)
	if parseState == nil {
		return nil
	}
	return appendPatchStreamIdentityOp(parseState, parseOp)
}

// buildPatchStreamIdentityHasher acquires one reusable hash64 hasher for patch-identity computation.
func buildPatchStreamIdentityHasher() hash.Hash64 {
	parseHasher, hasHasher := storePatchStreamIdentityHasherPool.Get().(hash.Hash64)
	if hasHasher && parseHasher != nil {
		return parseHasher
	}
	return fnv.New64a()
}

// buildPatchStreamKnownNodeIDScratchMap acquires one reusable known-node scratch map.
func buildPatchStreamKnownNodeIDScratchMap() map[uint64]struct{} {
	parseMap, hasMap := storePatchStreamKnownNodeIDScratchPool.Get().(map[uint64]struct{})
	if hasMap && parseMap != nil {
		return parseMap
	}
	return make(map[uint64]struct{})
}

// clearPatchStreamKnownNodeIDScratchMap resets one known-node scratch map for pool reuse.
func clearPatchStreamKnownNodeIDScratchMap(parseMap map[uint64]struct{}) {
	for parseNodeID := range parseMap {
		delete(parseMap, parseNodeID)
	}
}

// buildPatchStreamSiblingCountScratchMap acquires one reusable sibling-count scratch map.
func buildPatchStreamSiblingCountScratchMap() map[uint64]uint32 {
	parseMap, hasMap := storePatchStreamSiblingCountScratchPool.Get().(map[uint64]uint32)
	if hasMap && parseMap != nil {
		return parseMap
	}
	return make(map[uint64]uint32)
}

// clearPatchStreamSiblingCountScratchMap resets one sibling-count scratch map for pool reuse.
func clearPatchStreamSiblingCountScratchMap(parseMap map[uint64]uint32) {
	for parseParentNodeID := range parseMap {
		delete(parseMap, parseParentNodeID)
	}
}

// buildPatchStreamRemovedNodeIDScratchMap acquires one reusable removed-node scratch map.
func buildPatchStreamRemovedNodeIDScratchMap() map[uint64]struct{} {
	parseMap, hasMap := storePatchStreamRemovedNodeIDScratchPool.Get().(map[uint64]struct{})
	if hasMap && parseMap != nil {
		return parseMap
	}
	return make(map[uint64]struct{})
}

// clearPatchStreamRemovedNodeIDScratchMap resets one removed-node scratch map for pool reuse.
func clearPatchStreamRemovedNodeIDScratchMap(parseMap map[uint64]struct{}) {
	for parseNodeID := range parseMap {
		delete(parseMap, parseNodeID)
	}
}

// buildPatchStreamRemovedAttrKeyScratchMap acquires one reusable removed-attr scratch map.
func buildPatchStreamRemovedAttrKeyScratchMap() map[string]struct{} {
	parseMap, hasMap := storePatchStreamRemovedAttrKeyScratchPool.Get().(map[string]struct{})
	if hasMap && parseMap != nil {
		return parseMap
	}
	return make(map[string]struct{})
}

// clearPatchStreamRemovedAttrKeyScratchMap resets one removed-attr scratch map for pool reuse.
func clearPatchStreamRemovedAttrKeyScratchMap(parseMap map[string]struct{}) {
	for parseAttrKey := range parseMap {
		delete(parseMap, parseAttrKey)
	}
}

// ParsePatchStreamTransaction decodes one patch stream, validates patch semantics and idempotency, and returns a commit transaction.
func ParsePatchStreamTransaction(
	parseRaw PatchStreamRaw,
	parseExpectedRegionID string,
	parseExpectedEpoch uint64,
	parseKnownNodeIDs map[uint64]struct{},
	parseSiblingCountByParent map[uint64]uint32,
	parseTracker *PatchIdempotencyTracker,
) (PatchStreamParseResult, bool, error) {
	return parseParsePatchStreamTransaction(
		parseRaw,
		parseExpectedRegionID,
		parseExpectedEpoch,
		parseKnownNodeIDs,
		parseSiblingCountByParent,
		parseTracker,
		false,
		false,
	)
}

// ParsePatchStreamTransactionWithKeyedMoveHint decodes one patch stream using one caller-supplied keyed-move presence hint.
func ParsePatchStreamTransactionWithKeyedMoveHint(
	parseRaw PatchStreamRaw,
	parseExpectedRegionID string,
	parseExpectedEpoch uint64,
	parseKnownNodeIDs map[uint64]struct{},
	parseSiblingCountByParent map[uint64]uint32,
	parseTracker *PatchIdempotencyTracker,
	parseHasPatchKeyedMove bool,
) (PatchStreamParseResult, bool, error) {
	return parseParsePatchStreamTransaction(
		parseRaw,
		parseExpectedRegionID,
		parseExpectedEpoch,
		parseKnownNodeIDs,
		parseSiblingCountByParent,
		parseTracker,
		parseHasPatchKeyedMove,
		true,
	)
}

// parseParsePatchStreamTransaction decodes one patch stream and optionally uses one caller-supplied keyed-move presence hint.
func parseParsePatchStreamTransaction(
	parseRaw PatchStreamRaw,
	parseExpectedRegionID string,
	parseExpectedEpoch uint64,
	parseKnownNodeIDs map[uint64]struct{},
	parseSiblingCountByParent map[uint64]uint32,
	parseTracker *PatchIdempotencyTracker,
	parseHasPatchKeyedMove bool,
	parseHasPatchKeyedMoveKnown bool,
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
	parseStringTable := RenderStringTable{}
	if parseHasCanonicalStringTableSortedUnique(parseRaw.GetStringTable) {
		parseStringTable = RenderStringTable{
			Entries: parseRaw.GetStringTable,
		}
	} else {
		parseCanonicalStringTable, parseStringTableErr := ParseRenderStringTable(parseRaw.GetStringTable)
		if parseStringTableErr != nil {
			return PatchStreamParseResult{}, false, parseStringTableErr
		}
		parseStringTable = parseCanonicalStringTable
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
	buildKnownNodeIDs := parseKnownNodeIDs
	hasCopiedKnownNodeIDs := false
	defer func() {
		if !hasCopiedKnownNodeIDs || buildKnownNodeIDs == nil {
			return
		}
		clearPatchStreamKnownNodeIDScratchMap(buildKnownNodeIDs)
		storePatchStreamKnownNodeIDScratchPool.Put(buildKnownNodeIDs)
	}()
	parseGetMutableKnownNodeIDs := func() map[uint64]struct{} {
		if hasCopiedKnownNodeIDs {
			if buildKnownNodeIDs == nil {
				buildKnownNodeIDs = buildPatchStreamKnownNodeIDScratchMap()
			}
			return buildKnownNodeIDs
		}
		buildKnownNodeIDs = buildPatchStreamKnownNodeIDScratchMap()
		for getNodeID := range parseKnownNodeIDs {
			buildKnownNodeIDs[getNodeID] = struct{}{}
		}
		hasCopiedKnownNodeIDs = true
		return buildKnownNodeIDs
	}
	buildSiblingCountByParent := parseSiblingCountByParent
	hasCopiedSiblingCountByParent := false
	defer func() {
		if !hasCopiedSiblingCountByParent || buildSiblingCountByParent == nil {
			return
		}
		clearPatchStreamSiblingCountScratchMap(buildSiblingCountByParent)
		storePatchStreamSiblingCountScratchPool.Put(buildSiblingCountByParent)
	}()
	parseGetMutableSiblingCountByParent := func() map[uint64]uint32 {
		if hasCopiedSiblingCountByParent {
			if buildSiblingCountByParent == nil {
				buildSiblingCountByParent = buildPatchStreamSiblingCountScratchMap()
			}
			return buildSiblingCountByParent
		}
		buildSiblingCountByParent = buildPatchStreamSiblingCountScratchMap()
		for getParentNodeID, getSiblingCount := range parseSiblingCountByParent {
			buildSiblingCountByParent[getParentNodeID] = getSiblingCount
		}
		hasCopiedSiblingCountByParent = true
		return buildSiblingCountByParent
	}
	hasPatchKeyedMoveOpKnown := parseHasPatchKeyedMoveKnown
	hasPatchKeyedMoveOp := parseHasPatchKeyedMove
	parseShouldTrackSiblingCountByParent := func(parseOpIndex int) bool {
		if hasPatchKeyedMoveOpKnown {
			return hasPatchKeyedMoveOp
		}
		if len(parseSiblingCountByParent) == 0 {
			hasPatchKeyedMoveOpKnown = true
			hasPatchKeyedMoveOp = false
			return false
		}
		hasPatchKeyedMoveOp = parseHasPatchKeyedMoveOpFromIndex(parseRaw.GetOps, parseOpIndex)
		hasPatchKeyedMoveOpKnown = true
		return hasPatchKeyedMoveOp
	}
	var buildRemovedNodeIDs map[uint64]struct{}
	var buildRemovedAttrKeys map[string]struct{}
	defer func() {
		if buildRemovedNodeIDs != nil {
			clearPatchStreamRemovedNodeIDScratchMap(buildRemovedNodeIDs)
			storePatchStreamRemovedNodeIDScratchPool.Put(buildRemovedNodeIDs)
		}
		if buildRemovedAttrKeys != nil {
			clearPatchStreamRemovedAttrKeyScratchMap(buildRemovedAttrKeys)
			storePatchStreamRemovedAttrKeyScratchPool.Put(buildRemovedAttrKeys)
		}
	}()
	buildTransaction := RegionPatchTransaction{
		GetRegionID: parseHeader.RegionID,
		GetOps:      make([]RegionPatchOp, 0, len(parseRaw.GetOps)),
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
			parseKnownNodeIDsForMutation := parseGetMutableKnownNodeIDs()
			parseInsertOp, parseInsertErr := ParsePatchInsertOp(*getRawOp.GetInsertOp, parseKnownNodeIDsForMutation)
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
			parseKnownNodeIDsForMutation[parseInsertOp.Node.NodeID] = struct{}{}
			if parseShouldTrackSiblingCountByParent(parseOpIndex + 1) {
				buildSiblingCountByParentForMutation := parseGetMutableSiblingCountByParent()
				buildSiblingCountByParentForMutation[parseInsertOp.ParentNodeID] = buildSiblingCountByParentForMutation[parseInsertOp.ParentNodeID] + 1
			}
		case PatchOpCodeRemoveNode:
			if getRawOp.GetRemoveOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove payload is required", parseOpIndex)
			}
			if buildRemovedNodeIDs == nil {
				buildRemovedNodeIDs = buildPatchStreamRemovedNodeIDScratchMap()
			}
			parseKnownNodeIDsForMutation := parseGetMutableKnownNodeIDs()
			parseRemoveOp, parseRemoveErr := ParsePatchRemoveOp(*getRawOp.GetRemoveOp, parseKnownNodeIDsForMutation, buildRemovedNodeIDs)
			if parseRemoveErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove is invalid: %w", parseOpIndex, parseRemoveErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:   RegionPatchOpKindRemoveNode,
				GetNodeID: parseRemoveOp.TargetNodeID,
			})
			delete(parseKnownNodeIDsForMutation, parseRemoveOp.TargetNodeID)
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
			if buildRemovedAttrKeys == nil {
				buildRemovedAttrKeys = buildPatchStreamRemovedAttrKeyScratchMap()
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
			hasPatchKeyedMoveOpKnown = true
			hasPatchKeyedMoveOp = true
			parseSiblingCountByParentForValidation := parseGetMutableSiblingCountByParent()
			parseMoveOp, parseMoveErr := ParsePatchKeyedMoveOp(*getRawOp.GetKeyedMoveOp, buildKnownNodeIDs, parseSiblingCountByParentForValidation)
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

// parseHasPatchKeyedMoveOp reports whether one raw patch stream includes keyed-move operations.
func parseHasPatchKeyedMoveOp(parseOps []PatchStreamOpRaw) bool {
	return parseHasPatchKeyedMoveOpFromIndex(parseOps, 0)
}

// parseHasPatchKeyedMoveOpFromIndex reports whether one raw patch stream includes keyed-move operations at or after one op index.
func parseHasPatchKeyedMoveOpFromIndex(parseOps []PatchStreamOpRaw, parseFromIndex int) bool {
	if parseFromIndex < 0 {
		parseFromIndex = 0
	}
	if parseFromIndex >= len(parseOps) {
		return false
	}
	for _, getRawOp := range parseOps[parseFromIndex:] {
		if getRawOp.GetOpCode == uint8(PatchOpCodeMoveKeyedChild) {
			return true
		}
	}
	return false
}

// parseHasCanonicalStringTableSortedUnique reports whether one string table is canonical sorted and duplicate-free.
func parseHasCanonicalStringTableSortedUnique(parseEntries []string) bool {
	if len(parseEntries) <= 1 {
		return true
	}
	for parseIndex := 1; parseIndex < len(parseEntries); parseIndex++ {
		if parseEntries[parseIndex-1] >= parseEntries[parseIndex] {
			return false
		}
	}
	return true
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
		GetNodeID:  parseRecord.NodeID,
		GetNodeKey: parseRecord.KeyText,
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
	buildKnownNodeIDs := map[uint64]struct{}{}
	if parseIndex == nil || strings.TrimSpace(parseRegionID) == "" {
		return buildKnownNodeIDs
	}
	getRegionNodeByID, hasRegionNodeByID := parseIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionNodeByID {
		return buildKnownNodeIDs
	}
	buildKnownNodeIDs = make(map[uint64]struct{}, len(getRegionNodeByID))
	for getNodeID := range getRegionNodeByID {
		buildKnownNodeIDs[getNodeID] = struct{}{}
	}
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
		getChildNodeCount := len(getNode.GetChildNodeIDs)
		if getChildNodeCount == 0 {
			continue
		}
		buildSiblingCountByParent[getNode.GetNodeID] = uint32(getChildNodeCount)
	}
	return buildKnownNodeIDs, buildSiblingCountByParent
}

// BuildSiblingCountByParentForRegionDOMIndex extracts sibling counts keyed by parent node ID for one region.
func BuildSiblingCountByParentForRegionDOMIndex(parseIndex *RegionDOMIndex, parseRegionID string) map[uint64]uint32 {
	buildSiblingCountByParent := map[uint64]uint32{}
	if parseIndex == nil || strings.TrimSpace(parseRegionID) == "" {
		return buildSiblingCountByParent
	}
	getRegionNodeByID, hasRegionNodeByID := parseIndex.storeRegionDOMNodeByRegionID[parseRegionID]
	if !hasRegionNodeByID {
		return buildSiblingCountByParent
	}
	buildSiblingCountByParent = make(map[uint64]uint32, len(getRegionNodeByID))
	for _, getNode := range getRegionNodeByID {
		if getNode == nil {
			continue
		}
		getChildNodeCount := len(getNode.GetChildNodeIDs)
		if getChildNodeCount == 0 {
			continue
		}
		buildSiblingCountByParent[getNode.GetNodeID] = uint32(getChildNodeCount)
	}
	return buildSiblingCountByParent
}

type patchInsertBuildOp struct {
	getParentNodeID uint64
	getNodeID       uint64
	getNodeKind     RenderNodeKind
	getNodeTag      string
	getNodeText     string
	getNodeKey      string
	hasNodeKey      bool
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

// buildPatchStringCapacityFromPatchOps computes one stable capacity hint for patch string accumulation.
func buildPatchStringCapacityFromPatchOps(
	parseInsertOps []patchInsertBuildOp,
	parseSetTextOps []patchSetTextBuildOp,
	parseSetAttrOps []patchSetAttrBuildOp,
	parseSetStyleOps []patchSetStyleBuildOp,
	parseRemoveAttrOps []patchRemoveAttrBuildOp,
) int {
	parseCapacity := len(parseSetTextOps) + len(parseSetStyleOps) + len(parseRemoveAttrOps) + (len(parseSetAttrOps) * 2)
	for _, getInsertOp := range parseInsertOps {
		if getInsertOp.getNodeKind == RenderNodeKindText || getInsertOp.getNodeKind == RenderNodeKindHostElement {
			parseCapacity++
		}
		if getInsertOp.hasNodeKey {
			parseCapacity++
		}
	}
	return parseCapacity
}

// hasCanonicalNodeOrderStructuralDelta reports whether one sibling order references inserted or removed node IDs.
func hasCanonicalNodeOrderStructuralDelta(
	parseNodeIDs []uint64,
	parseRemovedNodeIDs map[uint64]struct{},
	parseInsertedNodeIDs map[uint64]struct{},
) bool {
	for _, getNodeID := range parseNodeIDs {
		if _, hasRemovedNode := parseRemovedNodeIDs[getNodeID]; hasRemovedNode {
			return true
		}
		if _, hasInsertedNode := parseInsertedNodeIDs[getNodeID]; hasInsertedNode {
			return true
		}
	}
	return false
}

// buildCanonicalPatchKeyedMoveOps computes keyed-move operations for canonical sibling reorders.
func buildCanonicalPatchKeyedMoveOps(
	parseRegionID string,
	parsePreviousTree canonicalRenderTree,
	parseNextTree canonicalRenderTree,
	parseRemovedNodeIDs map[uint64]struct{},
	parseInsertedNodeIDs map[uint64]struct{},
	parseHasStructuralNodeDelta bool,
) ([]PatchKeyedMoveOpRaw, error) {
	buildMoveOps := []PatchKeyedMoveOpRaw{}
	hasLoggedMoveSiblingWarn := false
	for getParentNodeID, getPreviousNode := range parsePreviousTree.getNodeByID {
		getNextNode, hasNextNode := parseNextTree.getNodeByID[getParentNodeID]
		if !hasNextNode {
			continue
		}
		buildCurrentOrder := getPreviousNode.getChildNodeIDs
		buildTargetOrder := getNextNode.getChildNodeIDs
		hasCurrentOrderScratch := false
		if parseHasStructuralNodeDelta {
			hasCurrentOrderStructuralDelta := hasCanonicalNodeOrderStructuralDelta(
				buildCurrentOrder,
				parseRemovedNodeIDs,
				parseInsertedNodeIDs,
			)
			hasTargetOrderStructuralDelta := hasCanonicalNodeOrderStructuralDelta(
				buildTargetOrder,
				parseRemovedNodeIDs,
				parseInsertedNodeIDs,
			)
			if hasCurrentOrderStructuralDelta || hasTargetOrderStructuralDelta {
				buildCurrentOrder = parseFilterCanonicalExistingOrder(buildCurrentOrder, parseRemovedNodeIDs, parseInsertedNodeIDs)
				buildTargetOrder = parseFilterCanonicalExistingOrder(buildTargetOrder, parseRemovedNodeIDs, parseInsertedNodeIDs)
				hasCurrentOrderScratch = true
			}
		}
		if len(buildTargetOrder) > getPatchMoveSiblingHardLimit {
			return nil, fmt.Errorf(
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
		if parseHasCanonicalNodeOrderEqual(buildCurrentOrder, buildTargetOrder) {
			continue
		}
		if !getNextNode.hasKeyedChild {
			continue
		}
		if !hasCurrentOrderScratch {
			// Keep canonical trees immutable while avoiding unnecessary filtered-order allocations.
			buildCurrentOrder = append([]uint64(nil), buildCurrentOrder...)
		}
		buildCurrentIndexByNode := parseBuildCanonicalSiblingIndexMap(buildCurrentOrder)
		for parseTargetIndex, getTargetNodeID := range buildTargetOrder {
			getTargetNode := parseNextTree.getNodeByID[getTargetNodeID]
			if !getTargetNode.hasKey {
				continue
			}
			parseCurrentIndex, hasCurrentIndex := buildCurrentIndexByNode[getTargetNodeID]
			if !hasCurrentIndex || parseCurrentIndex == parseTargetIndex {
				continue
			}
			parseMoveCanonicalNodeIDInPlace(buildCurrentOrder, parseCurrentIndex, parseTargetIndex, buildCurrentIndexByNode)
			buildMoveOps = append(buildMoveOps, PatchKeyedMoveOpRaw{
				ParentNodeID:     getParentNodeID,
				SourceNodeID:     getTargetNodeID,
				DestinationIndex: uint32(parseTargetIndex),
			})
		}
	}
	return buildMoveOps, nil
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
		buildHeader := PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        parseRegionID,
			Epoch:           parseEpoch,
			InputVersion:    parseInputVersion,
			PatchVersion:    parsePatchVersion,
		}
		buildIdentityState, parseIdentityStateErr := buildPatchStreamIdentityState(buildHeader, nil)
		if parseIdentityStateErr != nil {
			return PatchStreamRaw{}, false, parseIdentityStateErr
		}
		defer clearPatchStreamIdentityState(&buildIdentityState)
		if parseArrayErr := startPatchStreamIdentityOpArray(&buildIdentityState); parseArrayErr != nil {
			return PatchStreamRaw{}, false, parseArrayErr
		}
		buildOps := make([]PatchStreamOpRaw, 0, 1)
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode:           uint8(PatchOpCodeReplaceSubtree),
			GetReplaceSubtreeOp: &buildReplaceSubtreeOp,
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
		buildPatchIdentity, parseIdentityErr := formatPatchStreamIdentityState(&buildIdentityState)
		if parseIdentityErr != nil {
			return PatchStreamRaw{}, false, parseIdentityErr
		}
		buildPatchStreamRaw, parsePatchStreamErr := buildPatchStreamRawWithIdentity(buildHeader, nil, buildOps, buildPatchIdentity)
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
	hasStructuralNodeDelta := len(buildRemovedNodeIDs) > 0 || len(buildInsertedNodeIDs) > 0
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
	buildNextSiblingIndexCache := parseBuildCanonicalSiblingIndexCache(parseNextTree)
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
		parseLeftSiblingIndex := parseGetCanonicalSiblingIndexFromCache(buildNextSiblingIndexCache, parseLeftNode.getParentNodeID, parseLeftNodeID)
		parseRightSiblingIndex := parseGetCanonicalSiblingIndexFromCache(buildNextSiblingIndexCache, parseRightNode.getParentNodeID, parseRightNodeID)
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
		buildAnchorNodeID := parseFindCanonicalInsertAnchorWithIndexCache(
			getNextNode.getParentNodeID,
			getNodeID,
			parsePreviousTree,
			parseNextTree,
			buildInsertedNodeIDs,
			buildNextSiblingIndexCache,
		)
		buildInsertOps = append(buildInsertOps, patchInsertBuildOp{
			getParentNodeID: getNextNode.getParentNodeID,
			getNodeID:       getNodeID,
			getNodeKind:     getNextNode.getKind,
			getNodeTag:      getNextNode.getTag,
			getNodeText:     getNextNode.getText,
			getNodeKey:      getNextNode.getKey,
			hasNodeKey:      getNextNode.hasKey,
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
	buildMoveOps, parseMoveOpsErr := buildCanonicalPatchKeyedMoveOps(
		parseRegionID,
		parsePreviousTree,
		parseNextTree,
		buildRemovedNodeIDs,
		buildInsertedNodeIDs,
		hasStructuralNodeDelta,
	)
	if parseMoveOpsErr != nil {
		return PatchStreamRaw{}, false, parseMoveOpsErr
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
	buildPatchStrings := make([]string, 0, buildPatchStringCapacityFromPatchOps(
		buildInsertOps,
		buildSetTextOps,
		buildSetAttrOps,
		buildSetStyleOps,
		buildRemoveAttrOps,
	))
	for _, getInsertOp := range buildInsertOps {
		if getInsertOp.getNodeKind == RenderNodeKindText {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeText)
		}
		if getInsertOp.getNodeKind == RenderNodeKindHostElement {
			buildPatchStrings = append(buildPatchStrings, getInsertOp.getNodeTag)
		}
		if getInsertOp.hasNodeKey {
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
	buildPatchStringEntries := append([]string(nil), buildPatchStringTable.Entries...)
	buildOps := make([]PatchStreamOpRaw, 0, len(buildRemoveNodeIDs)+len(buildInsertOps)+len(buildSetTextOps)+len(buildSetAttrOps)+len(buildSetStyleOps)+len(buildRemoveAttrOps)+len(buildRemoveStyleOps)+len(buildMoveOps))
	buildHeader := PatchStreamHeaderRaw{
		ProtocolVersion: PatchStreamProtocolVersion,
		RegionID:        parseRegionID,
		Epoch:           parseEpoch,
		InputVersion:    parseInputVersion,
		PatchVersion:    parsePatchVersion,
	}
	buildIdentityState, parseIdentityStateErr := buildPatchStreamIdentityState(buildHeader, buildPatchStringEntries)
	if parseIdentityStateErr != nil {
		return PatchStreamRaw{}, false, parseIdentityStateErr
	}
	defer clearPatchStreamIdentityState(&buildIdentityState)
	if parseArrayErr := startPatchStreamIdentityOpArray(&buildIdentityState); parseArrayErr != nil {
		return PatchStreamRaw{}, false, parseArrayErr
	}
	for _, getNodeID := range buildRemoveNodeIDs {
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveNode),
			GetRemoveOp: &PatchRemoveOpRaw{
				TargetNodeID: getNodeID,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
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
		if getInsertOp.hasNodeKey {
			buildNodeRaw.KeyText = getInsertOp.getNodeKey
			buildNodeRaw.KeyHash = parseHashCanonicalString("key:"+getInsertOp.getNodeKey, nil)
		}
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeInsertNode),
			GetInsertOp: &PatchInsertOpRaw{
				ParentNodeID: getInsertOp.getParentNodeID,
				AnchorNodeID: getInsertOp.getAnchorNodeID,
				Node:         buildNodeRaw,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getSetTextOp := range buildSetTextOps {
		getTextRef, hasTextRef := buildPatchStringTable.GetRenderStringRef(getSetTextOp.getText)
		if !hasTextRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: set-text payload %q missing from patch table", getSetTextOp.getText)
		}
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetText),
			GetSetTextOp: &PatchSetTextOpRaw{
				TargetNodeID: getSetTextOp.getTargetNodeID,
				TextRef:      getTextRef,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
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
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetAttr),
			GetSetAttrOp: &PatchSetAttrOpRaw{
				TargetNodeID: getSetAttrOp.getTargetNodeID,
				Attr: RenderPropRecordRaw{
					Kind:     uint8(getSetAttrOp.getKind),
					KeyRef:   getKeyRef,
					ValueRef: getValueRef,
				},
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getSetStyleOp := range buildSetStyleOps {
		getStyleRef, hasStyleRef := buildPatchStringTable.GetRenderStringRef(getSetStyleOp.getStyleValue)
		if !hasStyleRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: set-style payload %q missing from patch table", getSetStyleOp.getStyleValue)
		}
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeSetStyle),
			GetSetStyleOp: &PatchSetStyleOpRaw{
				TargetNodeID: getSetStyleOp.getTargetNodeID,
				StyleRef:     getStyleRef,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getRemoveAttrOp := range buildRemoveAttrOps {
		getKeyRef, hasKeyRef := buildPatchStringTable.GetRenderStringRef(getRemoveAttrOp.getKey)
		if !hasKeyRef {
			return PatchStreamRaw{}, false, fmt.Errorf("runtime2: remove-attr key %q missing from patch table", getRemoveAttrOp.getKey)
		}
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveAttr),
			GetRemoveAttrOp: &PatchRemoveAttrOpRaw{
				TargetNodeID: getRemoveAttrOp.getTargetNodeID,
				KeyRef:       getKeyRef,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getRemoveStyleOp := range buildRemoveStyleOps {
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode: uint8(PatchOpCodeRemoveStyle),
			GetRemoveStyleOp: &PatchRemoveStyleOpRaw{
				TargetNodeID: getRemoveStyleOp.getTargetNodeID,
			},
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	for _, getMoveOp := range buildMoveOps {
		buildMoveOp := getMoveOp
		if parseAppendErr := appendPatchStreamOpWithIdentity(&buildOps, &buildIdentityState, PatchStreamOpRaw{
			GetOpCode:      uint8(PatchOpCodeMoveKeyedChild),
			GetKeyedMoveOp: &buildMoveOp,
		}); parseAppendErr != nil {
			return PatchStreamRaw{}, false, parseAppendErr
		}
	}
	buildPatchIdentity, parseIdentityErr := formatPatchStreamIdentityState(&buildIdentityState)
	if parseIdentityErr != nil {
		return PatchStreamRaw{}, false, parseIdentityErr
	}
	getPatchStreamRaw, parsePatchStreamErr := buildPatchStreamRawWithIdentity(buildHeader, buildPatchStringEntries, buildOps, buildPatchIdentity)
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
	return parseFindCanonicalInsertAnchorWithIndexCache(
		parseParentNodeID,
		parseNodeID,
		parsePreviousTree,
		parseNextTree,
		parseInsertedNodeIDs,
		parseBuildCanonicalSiblingIndexCache(parseNextTree),
	)
}

// parseFindCanonicalInsertAnchorWithIndexCache resolves one inserted-node anchor using one reusable sibling-index cache.
func parseFindCanonicalInsertAnchorWithIndexCache(
	parseParentNodeID uint64,
	parseNodeID uint64,
	parsePreviousTree canonicalRenderTree,
	parseNextTree canonicalRenderTree,
	parseInsertedNodeIDs map[uint64]struct{},
	parseSiblingIndexCache *canonicalSiblingIndexCache,
) uint64 {
	parseParentNode, hasParentNode := parseNextTree.getNodeByID[parseParentNodeID]
	if !hasParentNode {
		return 0
	}
	parseSiblingIndex := parseGetCanonicalSiblingIndexFromCache(parseSiblingIndexCache, parseParentNodeID, parseNodeID)
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

// parseBuildCanonicalSiblingIndexMap builds one nodeID->index lookup map for one sibling order list.
func parseBuildCanonicalSiblingIndexMap(parseChildNodeIDs []uint64) map[uint64]int {
	buildSiblingIndexByNode := make(map[uint64]int, len(parseChildNodeIDs))
	for parseIndex, getNodeID := range parseChildNodeIDs {
		buildSiblingIndexByNode[getNodeID] = parseIndex
	}
	return buildSiblingIndexByNode
}

// parseBuildCanonicalSiblingIndexCache builds one lazy sibling-index cache for one canonical tree.
func parseBuildCanonicalSiblingIndexCache(parseTree canonicalRenderTree) *canonicalSiblingIndexCache {
	return &canonicalSiblingIndexCache{
		getTree:                parseTree,
		getSiblingIndexByParen: make(map[uint64]map[uint64]int),
	}
}

// parseGetCanonicalSiblingIndexFromCache resolves one sibling index using one cached parent lookup map.
func parseGetCanonicalSiblingIndexFromCache(parseCache *canonicalSiblingIndexCache, parseParentNodeID uint64, parseNodeID uint64) int {
	if parseCache == nil {
		return -1
	}
	buildSiblingIndexByNode, hasSiblingIndexByNode := parseCache.getSiblingIndexByParen[parseParentNodeID]
	if !hasSiblingIndexByNode {
		parseParentNode, hasParentNode := parseCache.getTree.getNodeByID[parseParentNodeID]
		if !hasParentNode {
			return -1
		}
		buildSiblingIndexByNode = parseBuildCanonicalSiblingIndexMap(parseParentNode.getChildNodeIDs)
		parseCache.getSiblingIndexByParen[parseParentNodeID] = buildSiblingIndexByNode
	}
	parseSiblingIndex, hasSiblingIndex := buildSiblingIndexByNode[parseNodeID]
	if !hasSiblingIndex {
		return -1
	}
	return parseSiblingIndex
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

// parseMoveCanonicalNodeIDInPlace reorders one node ID in place and keeps the sibling-index map in sync.
func parseMoveCanonicalNodeIDInPlace(parseNodeIDs []uint64, parseFromIndex int, parseToIndex int, parseIndexByNode map[uint64]int) {
	if parseFromIndex < 0 || parseFromIndex >= len(parseNodeIDs) {
		return
	}
	if parseToIndex < 0 {
		parseToIndex = 0
	}
	if parseToIndex >= len(parseNodeIDs) {
		parseToIndex = len(parseNodeIDs) - 1
	}
	if parseFromIndex == parseToIndex {
		return
	}
	getNodeID := parseNodeIDs[parseFromIndex]
	if parseFromIndex < parseToIndex {
		copy(parseNodeIDs[parseFromIndex:parseToIndex], parseNodeIDs[parseFromIndex+1:parseToIndex+1])
		for parseIndex := parseFromIndex; parseIndex < parseToIndex; parseIndex++ {
			parseIndexByNode[parseNodeIDs[parseIndex]] = parseIndex
		}
		parseNodeIDs[parseToIndex] = getNodeID
		parseIndexByNode[getNodeID] = parseToIndex
		return
	}
	copy(parseNodeIDs[parseToIndex+1:parseFromIndex+1], parseNodeIDs[parseToIndex:parseFromIndex])
	for parseIndex := parseToIndex + 1; parseIndex <= parseFromIndex; parseIndex++ {
		parseIndexByNode[parseNodeIDs[parseIndex]] = parseIndex
	}
	parseNodeIDs[parseToIndex] = getNodeID
	parseIndexByNode[getNodeID] = parseToIndex
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

// parseHasCanonicalNodeOrderEqual reports whether two canonical node-order slices match exactly.
func parseHasCanonicalNodeOrderEqual(parseLeftNodeIDs []uint64, parseRightNodeIDs []uint64) bool {
	if len(parseLeftNodeIDs) != len(parseRightNodeIDs) {
		return false
	}
	for parseIndex := range parseLeftNodeIDs {
		if parseLeftNodeIDs[parseIndex] != parseRightNodeIDs[parseIndex] {
			return false
		}
	}
	return true
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
