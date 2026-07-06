package runtime2

import (
	"fmt"
	"hash"
	"hash/fnv"
	"maps"
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
		return make(map[patchRemovedAttrKey]struct{})
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

// VerifyPatchStreamIdentity recomputes the deterministic identity from a decoded
// stream's parts (header + string table + ops) and checks it against the identity
// carried in the stream. It is a defense-in-depth integrity check for patches that
// have crossed a trust/serialization boundary (e.g. the parallel-region worker
// postMessage bridge): a body corrupted or tampered while the carried identity is
// left intact is caught here rather than silently applied. A stream that carries no
// identity is accepted (nothing to check against; callers compute it on demand).
func VerifyPatchStreamIdentity(parseRaw PatchStreamRaw) error {
	parseCarried := strings.TrimSpace(parseRaw.GetPatchIdentity)
	if parseCarried == "" {
		return nil
	}
	parseRecomputed, parseErr := buildPatchStreamIdentityFromParts(parseRaw.GetHeader, parseRaw.GetStringTable, parseRaw.GetOps)
	if parseErr != nil {
		return parseErr
	}
	if parseRecomputed != parseCarried {
		return fmt.Errorf("runtime2: patch identity mismatch (carried %q, recomputed %q): corrupted or tampered patch stream", parseCarried, parseRecomputed)
	}
	return nil
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
func buildPatchStreamRemovedAttrKeyScratchMap() map[patchRemovedAttrKey]struct{} {
	parseMap, hasMap := storePatchStreamRemovedAttrKeyScratchPool.Get().(map[patchRemovedAttrKey]struct{})
	if hasMap && parseMap != nil {
		return parseMap
	}
	return make(map[patchRemovedAttrKey]struct{})
}

// clearPatchStreamRemovedAttrKeyScratchMap resets one removed-attr scratch map for pool reuse.
func clearPatchStreamRemovedAttrKeyScratchMap(parseMap map[patchRemovedAttrKey]struct{}) {
	for parseAttrKey := range parseMap {
		delete(parseMap, parseAttrKey)
	}
}

// getPatchMutableKnownNodeIDs returns one mutable known-node set, cloning into pooled scratch storage on first mutation.
func getPatchMutableKnownNodeIDs(
	parseKnownNodeIDs map[uint64]struct{},
	parseBuildKnownNodeIDs *map[uint64]struct{},
	parseHasCopiedKnownNodeIDs *bool,
) map[uint64]struct{} {
	if parseHasCopiedKnownNodeIDs != nil && *parseHasCopiedKnownNodeIDs {
		if parseBuildKnownNodeIDs == nil || *parseBuildKnownNodeIDs == nil {
			buildKnownNodeIDs := buildPatchStreamKnownNodeIDScratchMap()
			if parseBuildKnownNodeIDs != nil {
				*parseBuildKnownNodeIDs = buildKnownNodeIDs
			}
			return buildKnownNodeIDs
		}
		return *parseBuildKnownNodeIDs
	}
	buildKnownNodeIDs := buildPatchStreamKnownNodeIDScratchMap()
	for getNodeID := range parseKnownNodeIDs {
		buildKnownNodeIDs[getNodeID] = struct{}{}
	}
	if parseBuildKnownNodeIDs != nil {
		*parseBuildKnownNodeIDs = buildKnownNodeIDs
	}
	if parseHasCopiedKnownNodeIDs != nil {
		*parseHasCopiedKnownNodeIDs = true
	}
	return buildKnownNodeIDs
}

// getPatchMutableSiblingCountByParent returns one mutable parent-sibling-count map, cloning into pooled scratch storage on first mutation.
func getPatchMutableSiblingCountByParent(
	parseSiblingCountByParent map[uint64]uint32,
	parseBuildSiblingCountByParent *map[uint64]uint32,
	parseHasCopiedSiblingCountByParent *bool,
) map[uint64]uint32 {
	if parseHasCopiedSiblingCountByParent != nil && *parseHasCopiedSiblingCountByParent {
		if parseBuildSiblingCountByParent == nil || *parseBuildSiblingCountByParent == nil {
			buildSiblingCountByParent := buildPatchStreamSiblingCountScratchMap()
			if parseBuildSiblingCountByParent != nil {
				*parseBuildSiblingCountByParent = buildSiblingCountByParent
			}
			return buildSiblingCountByParent
		}
		return *parseBuildSiblingCountByParent
	}
	buildSiblingCountByParent := buildPatchStreamSiblingCountScratchMap()
	maps.Copy(buildSiblingCountByParent, parseSiblingCountByParent)
	if parseBuildSiblingCountByParent != nil {
		*parseBuildSiblingCountByParent = buildSiblingCountByParent
	}
	if parseHasCopiedSiblingCountByParent != nil {
		*parseHasCopiedSiblingCountByParent = true
	}
	return buildSiblingCountByParent
}

// hasPatchTrackSiblingCountByParent reports whether sibling-count tracking is required for keyed-move validation.
func hasPatchTrackSiblingCountByParent(
	parseRawOps []PatchStreamOpRaw,
	parseSiblingCountByParent map[uint64]uint32,
	parseHasPatchKeyedMoveOp *bool,
	parseHasPatchKeyedMoveOpKnown *bool,
	parseOpIndex int,
) bool {
	if parseHasPatchKeyedMoveOpKnown != nil && *parseHasPatchKeyedMoveOpKnown {
		if parseHasPatchKeyedMoveOp == nil {
			return false
		}
		return *parseHasPatchKeyedMoveOp
	}
	if len(parseSiblingCountByParent) == 0 {
		if parseHasPatchKeyedMoveOpKnown != nil {
			*parseHasPatchKeyedMoveOpKnown = true
		}
		if parseHasPatchKeyedMoveOp != nil {
			*parseHasPatchKeyedMoveOp = false
		}
		return false
	}
	buildHasPatchKeyedMoveOp := parseHasPatchKeyedMoveOpFromIndex(parseRawOps, parseOpIndex)
	if parseHasPatchKeyedMoveOp != nil {
		*parseHasPatchKeyedMoveOp = buildHasPatchKeyedMoveOp
	}
	if parseHasPatchKeyedMoveOpKnown != nil {
		*parseHasPatchKeyedMoveOpKnown = true
	}
	return buildHasPatchKeyedMoveOp
}

// findPatchRemoveOnlyStartIndex reports the first op index whose suffix contains only remove-node ops, or len(parseOps) when no remove-only suffix exists.
func findPatchRemoveOnlyStartIndex(parseOps []PatchStreamOpRaw) int {
	getRemoveOnlyStartIndex := len(parseOps)
	for parseOpIndex := len(parseOps) - 1; parseOpIndex >= 0; parseOpIndex-- {
		if parseOps[parseOpIndex].GetOpCode != uint8(PatchOpCodeRemoveNode) {
			break
		}
		getRemoveOnlyStartIndex = parseOpIndex
	}
	return getRemoveOnlyStartIndex
}

// hasPatchOnlyAppendInsertOps reports whether one raw patch stream contains only anchor-free insert ops, which can validate against the base known-node set plus one small added-node overlay.
func hasPatchOnlyAppendInsertOps(parseOps []PatchStreamOpRaw) bool {
	if len(parseOps) < 2 {
		return false
	}
	for _, getRawOp := range parseOps {
		if getRawOp.GetOpCode != uint8(PatchOpCodeInsertNode) ||
			getRawOp.GetInsertOp == nil ||
			getRawOp.GetInsertOp.AnchorNodeID != 0 {
			return false
		}
	}
	return true
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
) (parseResult PatchStreamParseResult, parseShouldApply bool, parseErr error) {
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
	if !parseRuntimeHasTrimmedNonWhitespaceText(parseRaw.GetPatchIdentity) {
		return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch stream identity is required")
	}
	if parseTracker != nil {
		// #72: CHECK idempotency (non-mutating) before decoding/validating the
		// patch body, but COMMIT the version state only after the whole body
		// validates. Committing up-front let a patch that failed body validation
		// record its version + identity, so a later corrected patch at the same
		// version was rejected as a conflict and the region wedged. The deferred
		// commit fires exactly when this call returns a successful, applied patch.
		hasApply, parseIdempotencyErr := parseTracker.CheckPatchIdempotency(
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
		defer func() {
			if parseErr == nil && parseShouldApply {
				_ = parseTracker.CommitPatchIdempotency(
					parseHeader.RegionID,
					parseHeader.Epoch,
					parseHeader.PatchVersion,
					parseRaw.GetPatchIdentity,
				)
			}
		}()
	}
	var parseStringTable RenderStringTable
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
	if hasPatchOnlyAppendInsertOps(parseRaw.GetOps) {
		return parseBuildPatchStreamAppendOnlyTransaction(parseHeader, parseStringTable, parseRaw.GetOps, parseKnownNodeIDs)
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
	buildSiblingCountByParent := parseSiblingCountByParent
	hasCopiedSiblingCountByParent := false
	defer func() {
		if !hasCopiedSiblingCountByParent || buildSiblingCountByParent == nil {
			return
		}
		clearPatchStreamSiblingCountScratchMap(buildSiblingCountByParent)
		storePatchStreamSiblingCountScratchPool.Put(buildSiblingCountByParent)
	}()
	hasPatchKeyedMoveOpKnown := parseHasPatchKeyedMoveKnown
	hasPatchKeyedMoveOp := parseHasPatchKeyedMove
	var buildRemovedNodeIDs map[uint64]struct{}
	var buildRemovedAttrKeys map[patchRemovedAttrKey]struct{}
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
	getRemoveOnlyStartIndex := findPatchRemoveOnlyStartIndex(parseRaw.GetOps)
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
			parseKnownNodeIDsForMutation := getPatchMutableKnownNodeIDs(
				parseKnownNodeIDs,
				&buildKnownNodeIDs,
				&hasCopiedKnownNodeIDs,
			)
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
			if hasPatchTrackSiblingCountByParent(
				parseRaw.GetOps,
				parseSiblingCountByParent,
				&hasPatchKeyedMoveOp,
				&hasPatchKeyedMoveOpKnown,
				parseOpIndex+1,
			) {
				buildSiblingCountByParentForMutation := getPatchMutableSiblingCountByParent(
					parseSiblingCountByParent,
					&buildSiblingCountByParent,
					&hasCopiedSiblingCountByParent,
				)
				buildSiblingCountByParentForMutation[parseInsertOp.ParentNodeID] = buildSiblingCountByParentForMutation[parseInsertOp.ParentNodeID] + 1
			}
		case PatchOpCodeRemoveNode:
			if getRawOp.GetRemoveOp == nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove payload is required", parseOpIndex)
			}
			if buildRemovedNodeIDs == nil {
				buildRemovedNodeIDs = buildPatchStreamRemovedNodeIDScratchMap()
			}
			var parseKnownNodeIDsForMutation map[uint64]struct{}
			if !hasCopiedKnownNodeIDs && parseOpIndex >= getRemoveOnlyStartIndex {
				parseKnownNodeIDsForMutation = parseKnownNodeIDs
			} else {
				parseKnownNodeIDsForMutation = getPatchMutableKnownNodeIDs(
					parseKnownNodeIDs,
					&buildKnownNodeIDs,
					&hasCopiedKnownNodeIDs,
				)
			}
			parseRemoveOp, parseRemoveErr := ParsePatchRemoveOp(*getRawOp.GetRemoveOp, parseKnownNodeIDsForMutation, buildRemovedNodeIDs)
			if parseRemoveErr != nil {
				return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d remove is invalid: %w", parseOpIndex, parseRemoveErr)
			}
			buildTransaction.GetOps = append(buildTransaction.GetOps, RegionPatchOp{
				GetKind:   RegionPatchOpKindRemoveNode,
				GetNodeID: parseRemoveOp.TargetNodeID,
			})
			if hasCopiedKnownNodeIDs {
				delete(parseKnownNodeIDsForMutation, parseRemoveOp.TargetNodeID)
			}
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
			parseSiblingCountByParentForValidation := buildSiblingCountByParent
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

// parseBuildPatchStreamAppendOnlyTransaction decodes one append-only insert patch stream using one added-node overlay instead of cloning the full known-node set.
func parseBuildPatchStreamAppendOnlyTransaction(
	parseHeader PatchStreamHeader,
	parseStringTable RenderStringTable,
	parseRawOps []PatchStreamOpRaw,
	parseKnownNodeIDs map[uint64]struct{},
) (PatchStreamParseResult, bool, error) {
	buildAddedNodeIDs := make(map[uint64]struct{}, len(parseRawOps))
	buildTransaction := RegionPatchTransaction{
		GetRegionID: parseHeader.RegionID,
		GetOps:      make([]RegionPatchOp, 0, len(parseRawOps)),
	}
	for parseOpIndex, getRawOp := range parseRawOps {
		if getRawOp.GetInsertOp == nil {
			return PatchStreamParseResult{}, false, fmt.Errorf("runtime2: patch op %d insert payload is required", parseOpIndex)
		}
		parseInsertOp, parseInsertErr := parseParsePatchAppendOnlyInsertOp(*getRawOp.GetInsertOp, parseKnownNodeIDs, buildAddedNodeIDs)
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
			GetBeforeNodeID: 0,
			GetInsertNode:   buildInsertNode,
		})
		buildAddedNodeIDs[parseInsertOp.Node.NodeID] = struct{}{}
	}
	return PatchStreamParseResult{
		GetHeader:      parseHeader,
		GetTransaction: buildTransaction,
	}, true, nil
}

// parseParsePatchAppendOnlyInsertOp validates one append-only insert op against the existing region nodes plus any node IDs introduced earlier in the same patch.
func parseParsePatchAppendOnlyInsertOp(
	parseRaw PatchInsertOpRaw,
	parseKnownNodeIDs map[uint64]struct{},
	parseAddedNodeIDs map[uint64]struct{},
) (PatchInsertOp, error) {
	if parseRaw.ParentNodeID == 0 {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op parent reference is required")
	}
	if !hasPatchKnownNodeID(parseKnownNodeIDs, parseAddedNodeIDs, parseRaw.ParentNodeID) {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op parent node id %d is unknown", parseRaw.ParentNodeID)
	}
	if parseRaw.AnchorNodeID != 0 {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op anchor node id %d is unsupported for append-only validation", parseRaw.AnchorNodeID)
	}
	if hasPatchKnownNodeID(parseKnownNodeIDs, parseAddedNodeIDs, parseRaw.Node.NodeID) {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op node id %d already exists", parseRaw.Node.NodeID)
	}
	parseNodeRecord, parseErr := ParseRenderNodeRecord(parseRaw.Node)
	if parseErr != nil {
		return PatchInsertOp{}, fmt.Errorf("runtime2: insert op node payload is invalid: %w", parseErr)
	}
	return PatchInsertOp{
		ParentNodeID: parseRaw.ParentNodeID,
		AnchorNodeID: 0,
		Node:         parseNodeRecord,
	}, nil
}

// hasPatchKnownNodeID reports whether one node ID exists either in the committed known-node set or in the patch-local added-node overlay.
func hasPatchKnownNodeID(
	parseKnownNodeIDs map[uint64]struct{},
	parseAddedNodeIDs map[uint64]struct{},
	parseNodeID uint64,
) bool {
	if parseNodeID == 0 {
		return false
	}
	if _, hasAddedNodeID := parseAddedNodeIDs[parseNodeID]; hasAddedNodeID {
		return true
	}
	_, hasKnownNodeID := parseKnownNodeIDs[parseNodeID]
	return hasKnownNodeID
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
