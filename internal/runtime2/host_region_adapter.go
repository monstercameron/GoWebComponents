package runtime2

import (
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"time"
)

// HostRegionAdapter owns host-side runtime2 handles for one live region instance.
type HostRegionAdapter struct {
	storeRegionInstanceID                            RegionInstanceID
	storeCoordinator                                 *Coordinator
	storeScheduler                                   *Scheduler
	storeRecoveryCoordinator                         *RecoveryCoordinator
	storeRegionDOMIndexHandle                        *RegionDOMIndex
	storeHostRegionSourceLookup                      HostRegionSourceLookup
	storeHostRegionSourceSnapshotCacheEpoch          uint64
	storeHostRegionSourceSnapshotCacheSourceVersion  uint64
	storeHostRegionSourceSnapshotCacheSourceIDs      []string
	storeHostRegionSourceSnapshotCacheVersionTuple   []uint64
	storeHostRegionSourceSnapshotScratchVersionTuple []uint64
	storeHostRegionSourceSnapshotCacheSourceValues   map[string]any
	storeHostRegionSourceSnapshotCacheSourceVersions map[string]uint64
	storeHostRegionSnapshotPropsCacheToken           uint64
	storeHostRegionSnapshotPropsShapeKeyCount        uint64
	storeHostRegionSnapshotPropsShapeKeyHash         uint64
	storeHostRegionSnapshotPropsShapeTypeHash        uint64
	storeHostRegionSnapshotPropsShapeKeys            []string
	storeHostRegionSnapshotPropsShapeTypeMarkers     []uint64
	storeHostRegionSnapshotPropsShapeScratchKeys     []string
	storeHostRegionSnapshotPropsShapeScratchTypes    []uint64
	storeHostRegionSnapshotFingerprint               string
	storeHostRegionSnapshotHash                      [sha256.Size]byte
	storeHostRegionSnapshotFastHash                  uint64
	storeHostRegionSnapshotHashScratch               []byte
	storeHostRegionDispatchHash                      [sha256.Size]byte
	storeHostRegionDispatchBytes                     []byte
	storeHostRegionDispatchScratch                   []byte
	storeHostRegionDispatchFastHash                  uint64
	storeHostRegionDispatchRendererID                RendererID
	storeHostRegionDispatchEpoch                     uint64
	storeHostRegionDispatchInputVersion              uint64
	storeHostRegionDispatchSourceVersion             uint64
	storeHostRegionDispatchSourceVersionTuple        []uint64
	storeHostRegionDispatchSourceVersionScratch      []uint64
	storeHostRegionDispatchPropsEntries              []buildSnapshotDispatchMapEntry
	storeHostRegionDispatchPropsOrderedKeys          []string
	storeHostRegionDispatchPropsScratchKeys          []string
	storeHostRegionCoordinatorCacheEpoch             uint64
	storeHostRegionCoordinatorCacheRendererID        RendererID
	storeHostRegionCoordinatorCacheSourceIDs         []string
	storeHostRegionCoordinatorCacheLastSnapshot      uint64
	storeHostRegionCoordinatorCacheLastDispatched    uint64
	storeHostRegionDeferredDispatch                  hostRegionDeferredDispatch
	storeHostRegionRepairRemountEpoch                uint64
	storeHostRegionRepairVersionFloor                uint64
	storeHostRegionLatestValidVersion                uint64
	storeHostRegionLastPatchVersion                  uint64
	storeHostRegionTransportTier                     TransportTier
	storeHostRegionSnapshotTier                      TransportTier
	storeHostRegionPatchTier                         TransportTier
	storeHostRegionFallbackReason                    string
	storeHostRegionDispatchAt                        time.Time
	storeHostRegionPatchReadyAt                      time.Time
	storeHostRegionCommitAt                          time.Time
	storeHostRegionDispatchToPatchNS                 uint64
	storeHostRegionDispatchToCommitNS                uint64
	storeHostRegionPatchToCommitNS                   uint64
	storeHostRegionPatchIdempotency                  *PatchIdempotencyTracker
	storeHostRegionDiagnosticRing                    []ControlEnvelope
	storeHostRegionSnapshotDowngrade                 DiagnosticDowngradeReason
	storeHostRegionPatchDowngrade                    DiagnosticDowngradeReason
	isHostRegionRemoved                              bool
	isHostRegionFallbackPending                      bool
	isHostRegionFallbackActive                       bool
	isHostRegionRepairPending                        bool
	isHostRegionHydrationComplete                    bool
	hasHostRegionPostHydrationAttached               bool
	hasHostRegionHydratedShellAnchor                 bool
	hasHostRegionSnapshotHash                        bool
	hasHostRegionSnapshotFastHash                    bool
	hasHostRegionDispatchHash                        bool
	hasHostRegionDispatchBytes                       bool
	hasHostRegionDispatchFastHash                    bool
	hasHostRegionDispatchSourceVersionTuple          bool
	hasHostRegionDispatchVersionVector               bool
	hasHostRegionDispatchPropsLayoutFresh            bool
	hasHostRegionCoordinatorCache                    bool
	hasHostRegionSourceSnapshotCache                 bool
	hasHostRegionSnapshotPropsCacheToken             bool
	hasHostRegionSnapshotPropsShapeFingerprint       bool
	hasHostRegionDeferredDispatch                    bool
	hasHostRegionSnapshotDowngrade                   bool
	hasHostRegionPatchDowngrade                      bool
	isHostRegionLocalShellOwned                      bool
	isHostRegionRoundTripTimingEnabled               bool
}

const getHostRegionDiagnosticRingLimit = 32

type hostRegionDeferredDispatch struct {
	getInputVersion        uint64
	getSnapshotEnvelope    SnapshotEnvelope
	getSnapshotFingerprint string
}

// HostRegionMountResult reports host-side mount outputs for one region mount attempt.
type HostRegionMountResult struct {
	GetSchedulerJob     SchedulerJob
	GetCoordinatorEntry CoordinatorEntry
}

// HostRegionUpdateResult reports host-side update outputs for one region update dispatch.
type HostRegionUpdateResult struct {
	GetSchedulerJob      SchedulerJob
	GetDispatchedVersion uint64
}

// HostRegionDisposeResult reports host-side dispose outputs for one mounted region.
type HostRegionDisposeResult struct {
	HasCoordinatorDisposed bool
	HasSchedulerDisposed   bool
	GetClearedDOMNodeCount int
}

// HostRegionSourceLookup resolves declared source values and versions from shipped runtime state.
type HostRegionSourceLookup func(parseSourceIDs []string) (map[string]any, map[string]uint64, error)

// HostRegionSourceSnapshot stores source values and source versions returned by host-side source lookup.
type HostRegionSourceSnapshot struct {
	GetSourceValues   map[string]any
	GetSourceVersions map[string]uint64
	GetSourceVersion  uint64
}

// HostRegionSnapshotFingerprintResult reports one snapshot fingerprint decision for no-change detection.
type HostRegionSnapshotFingerprintResult struct {
	GetSnapshotFingerprint string
	HasNoChange            bool
}

type hostRegionSnapshotHashResult struct {
	getSnapshotHash [sha256.Size]byte
	hasNoChange     bool
}

// HostRegionDispatchPriority identifies runtime2 host update dispatch priority classification.
type HostRegionDispatchPriority string

const (
	// HostRegionDispatchPriorityUrgent marks an update that should dispatch immediately.
	HostRegionDispatchPriorityUrgent HostRegionDispatchPriority = "urgent"
	// HostRegionDispatchPriorityDeferred marks an update that may dispatch later under deferred policy.
	HostRegionDispatchPriorityDeferred HostRegionDispatchPriority = "deferred"
)

// ParseHostRegionDispatchPriority validates one host update dispatch priority value.
func ParseHostRegionDispatchPriority(parsePriority HostRegionDispatchPriority) (HostRegionDispatchPriority, error) {
	switch parsePriority {
	case HostRegionDispatchPriorityUrgent, HostRegionDispatchPriorityDeferred:
		return parsePriority, nil
	default:
		return "", fmt.Errorf("runtime2: host dispatch priority %q is unsupported", parsePriority)
	}
}

func parseHostRegionMaxVersion(parseVersions ...uint64) uint64 {
	var parseMaxVersion uint64
	for _, parseVersion := range parseVersions {
		if parseVersion > parseMaxVersion {
			parseMaxVersion = parseVersion
		}
	}
	return parseMaxVersion
}

// parseHasHostRegionExactSourceIDList reports whether two source-ID lists match exactly in length and order.
func parseHasHostRegionExactSourceIDList(parseLeft []string, parseRight []string) bool {
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

// parseHasHostRegionExactSourceVersionTuple reports whether two source-version tuples match exactly in length and order.
func parseHasHostRegionExactSourceVersionTuple(parseLeft []uint64, parseRight []uint64) bool {
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

// buildHostRegionSourceSnapshotVersionTuple builds one ordered source-version tuple that follows parseSourceIDs.
func buildHostRegionSourceSnapshotVersionTuple(
	parseTarget []uint64,
	parseSourceIDs []string,
	parseSourceVersions map[string]uint64,
) ([]uint64, error) {
	if len(parseSourceIDs) == 0 {
		return parseTarget[:0], nil
	}
	if cap(parseTarget) < len(parseSourceIDs) {
		parseTarget = make([]uint64, len(parseSourceIDs))
	} else {
		parseTarget = parseTarget[:len(parseSourceIDs)]
	}
	for parseIndex, getSourceID := range parseSourceIDs {
		getSourceVersion, hasSourceVersion := parseSourceVersions[getSourceID]
		if !hasSourceVersion {
			return parseTarget[:0], fmt.Errorf("runtime2: missing declared source version for %q", getSourceID)
		}
		parseTarget[parseIndex] = getSourceVersion
	}
	return parseTarget, nil
}

// buildHostRegionSourceSnapshotVersionMap copies one ordered source-version tuple into a declared-source map.
func buildHostRegionSourceSnapshotVersionMap(
	parseSourceIDs []string,
	parseSourceVersionTuple []uint64,
) map[string]uint64 {
	if len(parseSourceIDs) == 0 {
		return nil
	}
	buildSourceVersions := make(map[string]uint64, len(parseSourceIDs))
	for parseIndex, getSourceID := range parseSourceIDs {
		buildSourceVersions[getSourceID] = parseSourceVersionTuple[parseIndex]
	}
	return buildSourceVersions
}

// buildHostRegionSnapshotImmutablePropsToken builds one cheap cache token for immutable props payloads.
func buildHostRegionSnapshotImmutablePropsToken(parseProps any) (uint64, bool) {
	const (
		getHostRegionPropsTokenSeed  uint64 = 14695981039346656037
		getHostRegionPropsTokenPrime uint64 = 1099511628211
	)
	switch getProps := parseProps.(type) {
	case nil:
		return 1, true
	case bool:
		if getProps {
			return 3, true
		}
		return 2, true
	case int:
		return uint64(int64(getProps))*getHostRegionPropsTokenPrime + 11, true
	case int8:
		return uint64(int64(getProps))*getHostRegionPropsTokenPrime + 12, true
	case int16:
		return uint64(int64(getProps))*getHostRegionPropsTokenPrime + 13, true
	case int32:
		return uint64(int64(getProps))*getHostRegionPropsTokenPrime + 14, true
	case int64:
		return uint64(getProps)*getHostRegionPropsTokenPrime + 15, true
	case uint:
		return uint64(getProps)*getHostRegionPropsTokenPrime + 16, true
	case uint8:
		return uint64(getProps)*getHostRegionPropsTokenPrime + 17, true
	case uint16:
		return uint64(getProps)*getHostRegionPropsTokenPrime + 18, true
	case uint32:
		return uint64(getProps)*getHostRegionPropsTokenPrime + 19, true
	case uint64:
		return getProps*getHostRegionPropsTokenPrime + 20, true
	case uintptr:
		return uint64(getProps)*getHostRegionPropsTokenPrime + 21, true
	case float32:
		return uint64(math.Float32bits(getProps))*getHostRegionPropsTokenPrime + 22, true
	case float64:
		return math.Float64bits(getProps)*getHostRegionPropsTokenPrime + 23, true
	case string:
		buildToken := getHostRegionPropsTokenSeed ^ 31
		for parseIndex := 0; parseIndex < len(getProps); parseIndex++ {
			buildToken ^= uint64(getProps[parseIndex])
			buildToken *= getHostRegionPropsTokenPrime
		}
		return buildToken, true
	default:
		return 0, false
	}
}

// buildHostRegionDispatchSourceVersionTuple builds one ordered source-version tuple that follows parseSourceIDs.
func buildHostRegionDispatchSourceVersionTuple(
	parseTarget []uint64,
	parseSourceIDs []string,
	parseSourceVersions map[string]uint64,
) ([]uint64, bool) {
	if len(parseSourceIDs) == 0 {
		return parseTarget[:0], true
	}
	if len(parseSourceVersions) == 0 {
		return parseTarget[:0], false
	}
	if cap(parseTarget) < len(parseSourceIDs) {
		parseTarget = make([]uint64, len(parseSourceIDs))
	} else {
		parseTarget = parseTarget[:len(parseSourceIDs)]
	}
	for parseIndex, getSourceID := range parseSourceIDs {
		getSourceVersion, hasSourceVersion := parseSourceVersions[getSourceID]
		if !hasSourceVersion {
			return parseTarget[:0], false
		}
		parseTarget[parseIndex] = getSourceVersion
	}
	return parseTarget, true
}

// parseHasHostRegionExactPropsKeyList reports whether one props map matches one cached ordered key list exactly.
func parseHasHostRegionExactPropsKeyList(parseProps map[string]any, parseOrderedKeys []string) bool {
	if len(parseProps) == 0 || len(parseProps) != len(parseOrderedKeys) {
		return false
	}
	for _, parseKey := range parseOrderedKeys {
		if _, hasParseKey := parseProps[parseKey]; !hasParseKey {
			return false
		}
	}
	return true
}

// buildHostRegionDispatchPropsOrderedKeys builds one sorted key list for a props map into parseTarget.
func buildHostRegionDispatchPropsOrderedKeys(parseTarget []string, parseProps map[string]any) []string {
	parseTarget = parseTarget[:0]
	for parseKey := range parseProps {
		parseTarget = append(parseTarget, parseKey)
	}
	sort.Strings(parseTarget)
	return parseTarget
}

// buildHostRegionDispatchPropsEntries builds one sorted key/value slice for a props map in caller-provided key order.
func buildHostRegionDispatchPropsEntries(
	parseTarget []buildSnapshotDispatchMapEntry,
	parseProps map[string]any,
	parseOrderedKeys []string,
) ([]buildSnapshotDispatchMapEntry, bool) {
	if len(parseProps) == 0 || len(parseOrderedKeys) != len(parseProps) {
		return parseTarget[:0], false
	}
	parseTarget = parseTarget[:0]
	for _, parseKey := range parseOrderedKeys {
		parseValue, hasParseValue := parseProps[parseKey]
		if !hasParseValue {
			return parseTarget[:0], false
		}
		parseTarget = append(parseTarget, buildSnapshotDispatchMapEntry{
			getKey:   parseKey,
			getValue: parseValue,
		})
	}
	return parseTarget, true
}

// clearHostRegionDispatchPropsLayout clears cached ordered props layout state used by hot dispatch hashing.
func (parseHostRegionAdapter *HostRegionAdapter) clearHostRegionDispatchPropsLayout() {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionDispatchPropsEntries = parseHostRegionAdapter.storeHostRegionDispatchPropsEntries[:0]
	parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys = parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys[:0]
	parseHostRegionAdapter.storeHostRegionDispatchPropsScratchKeys = parseHostRegionAdapter.storeHostRegionDispatchPropsScratchKeys[:0]
	parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh = false
}

// storeHostRegionDispatchPropsLayout stores one compact ordered props layout for flat map[string]any dispatch hashing.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionDispatchPropsLayout(parseProps any) {
	if parseHostRegionAdapter == nil {
		return
	}
	parsePropsMap, hasPropsMap := parseProps.(map[string]any)
	if !hasPropsMap || len(parsePropsMap) == 0 {
		parseHostRegionAdapter.clearHostRegionDispatchPropsLayout()
		return
	}
	var getPropsOrderedKeys []string
	switch {
	case parseHasHostRegionExactPropsKeyList(parsePropsMap, parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys):
		getPropsOrderedKeys = parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys
	case parseHasHostRegionExactPropsKeyList(parsePropsMap, parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys):
		parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys = append(
			parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys[:0],
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys...,
		)
		getPropsOrderedKeys = parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys
	default:
		getPropsScratchKeys := buildHostRegionDispatchPropsOrderedKeys(
			parseHostRegionAdapter.storeHostRegionDispatchPropsScratchKeys[:0],
			parsePropsMap,
		)
		parseHostRegionAdapter.storeHostRegionDispatchPropsScratchKeys = getPropsScratchKeys[:0]
		parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys = append(
			parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys[:0],
			getPropsScratchKeys...,
		)
		getPropsOrderedKeys = parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys
	}
	getPropsEntries, hasPropsEntries := buildHostRegionDispatchPropsEntries(
		parseHostRegionAdapter.storeHostRegionDispatchPropsEntries[:0],
		parsePropsMap,
		getPropsOrderedKeys,
	)
	if !hasPropsEntries {
		parseHostRegionAdapter.clearHostRegionDispatchPropsLayout()
		return
	}
	parseHostRegionAdapter.storeHostRegionDispatchPropsEntries = getPropsEntries
	parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh = true
}

// clearHostRegionDispatchDigestState clears cached dispatch digest state that is invalid after a version-vector mismatch.
func (parseHostRegionAdapter *HostRegionAdapter) clearHostRegionDispatchDigestState() {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionDispatchHash = [sha256.Size]byte{}
	parseHostRegionAdapter.hasHostRegionDispatchHash = false
	parseHostRegionAdapter.storeHostRegionDispatchBytes = nil
	parseHostRegionAdapter.hasHostRegionDispatchBytes = false
	parseHostRegionAdapter.storeHostRegionDispatchFastHash = 0
	parseHostRegionAdapter.hasHostRegionDispatchFastHash = false
}

// clearHostRegionSnapshotPropsCache clears cached props token state used by snapshot validation short-circuit checks.
func (parseHostRegionAdapter *HostRegionAdapter) clearHostRegionSnapshotPropsCache() {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionSnapshotPropsCacheToken = 0
	parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyCount = 0
	parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyHash = 0
	parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeHash = 0
	parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys = parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys[:0]
	parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers = parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers[:0]
	parseHostRegionAdapter.hasHostRegionSnapshotPropsCacheToken = false
	parseHostRegionAdapter.hasHostRegionSnapshotPropsShapeFingerprint = false
}

// copyHostRegionCoordinatorSourceIDs copies one source-ID list into reusable target storage.
func copyHostRegionCoordinatorSourceIDs(parseTarget []string, parseSourceIDs []string) []string {
	if len(parseSourceIDs) == 0 {
		return parseTarget[:0]
	}
	return append(parseTarget[:0], parseSourceIDs...)
}

// storeHostRegionCoordinatorCacheState stores one coordinator snapshot-dispatch field set in adapter-local cache.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionCoordinatorCacheState(
	parseEpoch uint64,
	parseRendererID RendererID,
	parseSourceIDs []string,
	parseIsFallback bool,
	parseLastSnapshotVersion uint64,
	parseLastDispatchedVersion uint64,
) {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheEpoch = parseEpoch
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheRendererID = parseRendererID
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs = copyHostRegionCoordinatorSourceIDs(
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs,
		parseSourceIDs,
	)
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot = parseLastSnapshotVersion
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched = parseLastDispatchedVersion
	parseHostRegionAdapter.isHostRegionFallbackActive = parseIsFallback
	parseHostRegionAdapter.hasHostRegionCoordinatorCache = true
}

// clearHostRegionCoordinatorCacheState clears adapter-local cached coordinator fields after region disposal.
func (parseHostRegionAdapter *HostRegionAdapter) clearHostRegionCoordinatorCacheState() {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheEpoch = 0
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheRendererID = ""
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs = parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs[:0]
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot = 0
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched = 0
	parseHostRegionAdapter.hasHostRegionCoordinatorCache = false
}

// storeHostRegionCoordinatorCacheSnapshotState stores one snapshot-version update in adapter-local coordinator cache.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionCoordinatorCacheSnapshotState(
	parseSnapshotVersion uint64,
	parseSourceIDs []string,
	parseShouldStoreSourceIDs bool,
) {
	if parseHostRegionAdapter == nil || !parseHostRegionAdapter.hasHostRegionCoordinatorCache {
		return
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot = parseSnapshotVersion
	if parseShouldStoreSourceIDs {
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs = copyHostRegionCoordinatorSourceIDs(
			parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs,
			parseSourceIDs,
		)
	}
}

// storeHostRegionCoordinatorCacheDispatchedVersion stores one dispatched-version update in adapter-local coordinator cache.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionCoordinatorCacheDispatchedVersion(parseDispatchedVersion uint64) {
	if parseHostRegionAdapter == nil || !parseHostRegionAdapter.hasHostRegionCoordinatorCache {
		return
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched = parseDispatchedVersion
}

// storeHostRegionCoordinatorCacheSnapshotDispatchState stores one snapshot+dispatch version update in adapter-local coordinator cache.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionCoordinatorCacheSnapshotDispatchState(
	parseSnapshotVersion uint64,
	parseDispatchedVersion uint64,
	parseSourceIDs []string,
	parseShouldStoreSourceIDs bool,
) {
	if parseHostRegionAdapter == nil || !parseHostRegionAdapter.hasHostRegionCoordinatorCache {
		return
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot = parseSnapshotVersion
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched = parseDispatchedVersion
	if parseShouldStoreSourceIDs {
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs = copyHostRegionCoordinatorSourceIDs(
			parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs,
			parseSourceIDs,
		)
	}
}

// getHostRegionCoordinatorSnapshotDispatchState returns adapter-local cached coordinator fields when valid or refreshes from coordinator.
func (parseHostRegionAdapter *HostRegionAdapter) getHostRegionCoordinatorSnapshotDispatchState(
	parseShouldUseCache bool,
) (
	getEpoch uint64,
	getRendererID RendererID,
	getSourceIDs []string,
	isFallback bool,
	lastSnapshotVersion uint64,
	lastDispatchedVersion uint64,
	ok bool,
) {
	if parseHostRegionAdapter == nil {
		return 0, "", nil, false, 0, 0, false
	}
	if parseShouldUseCache && parseHostRegionAdapter.hasHostRegionCoordinatorCache {
		return parseHostRegionAdapter.storeHostRegionCoordinatorCacheEpoch,
			parseHostRegionAdapter.storeHostRegionCoordinatorCacheRendererID,
			parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs,
			parseHostRegionAdapter.isHostRegionFallbackActive,
			parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot,
			parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched,
			true
	}
	getEntryEpoch,
		getEntryRendererID,
		getEntrySourceIDs,
		getEntryIsFallback,
		getEntryLastSnapshotVersion,
		getEntryLastDispatchedVersion,
		hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntrySnapshotAndDispatchFields(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return 0, "", nil, false, 0, 0, false
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheState(
		getEntryEpoch,
		getEntryRendererID,
		getEntrySourceIDs,
		getEntryIsFallback,
		getEntryLastSnapshotVersion,
		getEntryLastDispatchedVersion,
	)
	return parseHostRegionAdapter.storeHostRegionCoordinatorCacheEpoch,
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheRendererID,
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs,
		parseHostRegionAdapter.isHostRegionFallbackActive,
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot,
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched,
		true
}

// storeHostRegionDispatchVersionVector stores one dispatch version-vector snapshot for the next no-change gate.
func (parseHostRegionAdapter *HostRegionAdapter) storeHostRegionDispatchVersionVector(
	parseRendererID RendererID,
	parseSnapshotEnvelope SnapshotEnvelope,
	parseSourceVersionTuple []uint64,
	hasSourceVersionTuple bool,
) {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionDispatchRendererID = parseRendererID
	parseHostRegionAdapter.storeHostRegionDispatchEpoch = parseSnapshotEnvelope.Epoch
	parseHostRegionAdapter.storeHostRegionDispatchInputVersion = parseSnapshotEnvelope.InputVersion
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersion = parseSnapshotEnvelope.SourceVersion
	parseHostRegionAdapter.hasHostRegionDispatchVersionVector = true
	if hasSourceVersionTuple {
		parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple = append(
			parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple[:0],
			parseSourceVersionTuple...,
		)
		parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple = true
		return
	}
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple = parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple[:0]
	parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple = false
}

// HostRegionUpdateDispatchResult reports one host-side update dispatch decision including short-circuit behavior.
type HostRegionUpdateDispatchResult struct {
	HasScheduled           bool
	HasNoChange            bool
	HasDeferredQueued      bool
	HasDeferredSuperseded  bool
	HasDeferredCanceled    bool
	GetDispatchPriority    HostRegionDispatchPriority
	GetSchedulerJob        SchedulerJob
	GetSnapshotEnvelope    SnapshotEnvelope
	GetSnapshotFingerprint string
}

// HostRegionOwnerInvalidateResult reports cleanup actions applied after owner-side invalidation.
type HostRegionOwnerInvalidateResult struct {
	HasSchedulerCanceled bool
	HasDeferredCleared   bool
}

// HostRegionWorkerOutputResult reports whether one worker output attempt committed or was ignored.
type HostRegionWorkerOutputResult struct {
	HasCommitted bool
	HasIgnored   bool
}

// HostRegionOwnerRemoveResult reports owner-removal handling for one mounted region.
type HostRegionOwnerRemoveResult struct {
	HasDisposed                   bool
	HasLateWorkerOutputSuppressed bool
}

// HostRegionStructuralRemountResult reports renderer or shell-ownership changes and remount decisions.
type HostRegionStructuralRemountResult struct {
	HasRendererChanged       bool
	HasShellOwnershipChanged bool
	HasRemounted             bool
	GetRemountEpoch          uint64
}

// HostRegionFallbackMirrorResult reports fallback mirroring into coordinator and scheduler state.
type HostRegionFallbackMirrorResult struct {
	HasCoordinatorFallback bool
	HasSchedulerFallback   bool
}

// HostRegionPatchReadyResult reports whether one patch-ready result may continue to commit processing.
type HostRegionPatchReadyResult struct {
	HasAccepted     bool
	HasIgnored      bool
	GetIgnoreReason string
}

// HostRegionDiagnosticResult reports whether one worker diagnostic was stored or ignored.
type HostRegionDiagnosticResult struct {
	HasStored        bool
	HasIgnored       bool
	GetIgnoreReason  string
	GetDiagnostic    ControlEnvelope
	GetDiagnosticLen int
}

// HostRegionWorkerDeathResult reports host-side worker death handling outcomes.
type HostRegionWorkerDeathResult struct {
	HasReassigned      bool
	HasFallbackEntered bool
	IsRepairPending    bool
	GetRemountEpoch    uint64
	GetVersionFloor    uint64
}

// HostRegionRepairRemountResult reports repair-driven remount handshake outcomes.
type HostRegionRepairRemountResult struct {
	HasRemounted       bool
	HasFallbackCleared bool
	GetRemountEpoch    uint64
	GetVersionFloor    uint64
}

// HostRegionHydrationAttachResult reports post-hydration worker attach handling outcomes.
type HostRegionHydrationAttachResult struct {
	HasAttached bool
	HasBlocked  bool
}

// HostRegionShellIdentityMismatchResult reports shell-marker identity mismatches for region or renderer IDs.
type HostRegionShellIdentityMismatchResult struct {
	HasMismatch           bool
	HasRegionIDMismatch   bool
	HasRendererIDMismatch bool
}

// HostRegionShellAnchorCheckResult reports whether one hydrated shell anchor is missing for the mounted region.
type HostRegionShellAnchorCheckResult struct {
	HasMissingAnchor bool
}

// HostRegionShellMismatchFallbackResult reports fallback ownership and remount-floor outcomes after shell mismatch.
type HostRegionShellMismatchFallbackResult struct {
	HasFallbackEntered  bool
	HasOutputSuppressed bool
	GetRemountEpoch     uint64
	GetVersionFloor     uint64
}

// HostRegionRuntimeMode identifies one runtime-status ownership mode for a host region.
type HostRegionRuntimeMode string

const (
	// HostRegionRuntimeModeLocalShell reports local-shell ownership before worker attach.
	HostRegionRuntimeModeLocalShell HostRegionRuntimeMode = "local-shell"
	// HostRegionRuntimeModeWorkerAttached reports active worker-backed attach.
	HostRegionRuntimeModeWorkerAttached HostRegionRuntimeMode = "worker-attached"
	// HostRegionRuntimeModeFallback reports locally-owned fallback mode.
	HostRegionRuntimeModeFallback HostRegionRuntimeMode = "fallback"
)

// HostRegionRuntimeStatus reports one host region runtime status snapshot for observability surfaces.
type HostRegionRuntimeStatus struct {
	GetRegionInstanceID            RegionInstanceID
	GetRegionMode                  HostRegionRuntimeMode
	GetAssignedWorkerShard         string
	GetRendererID                  RendererID
	GetEpoch                       uint64
	GetIsHydrationComplete         bool
	HasHydratedShellAnchor         bool
	HasPostHydrationAttached       bool
	GetLastSnapshotVersion         uint64
	GetLastDispatchedVersion       uint64
	GetLastCommittedVersion        uint64
	GetTransportTier               TransportTier
	HasSnapshotDowngrade           bool
	GetSnapshotDowngradePath       DiagnosticDowngradePath
	GetSnapshotDowngradeReason     string
	HasPatchDowngrade              bool
	GetPatchDowngradePath          DiagnosticDowngradePath
	GetPatchDowngradeReason        string
	GetDroppedStalePatchCount      uint64
	GetIgnoredStaleDiagnosticCount uint64
	GetFallbackReason              string
}

// HostRegionRoundTripTiming reports dispatch, patch-ready, and commit timing spans for one region.
type HostRegionRoundTripTiming struct {
	GetDispatchToPatchReadyNS uint64
	GetDispatchToCommitNS     uint64
	GetPatchReadyToCommitNS   uint64
}

// HostRegionTransportDowngradeStatus reports separate snapshot and patch transport downgrade accounting for one region.
type HostRegionTransportDowngradeStatus struct {
	GetSnapshotTransportTier TransportTier
	HasSnapshotDowngrade     bool
	GetSnapshotDowngrade     DiagnosticDowngradeReason
	GetPatchTransportTier    TransportTier
	HasPatchDowngrade        bool
	GetPatchDowngrade        DiagnosticDowngradeReason
}

// HostRegionDiagnosticsSnapshot reports one read-only host diagnostics snapshot for redacted events, downgrade accounting, and counters.
type HostRegionDiagnosticsSnapshot struct {
	GetRegionInstanceID            RegionInstanceID
	GetDiagnosticEvents            []ControlEnvelope
	GetTransportDowngradeStatus    HostRegionTransportDowngradeStatus
	GetDroppedStalePatchCount      uint64
	GetIgnoredStaleDiagnosticCount uint64
	GetRepairTriggeredRemountCount uint64
}

// BuildHostRegionAdapter creates one host-side region adapter with coordinator, scheduler, recovery, and DOM-index handles.
