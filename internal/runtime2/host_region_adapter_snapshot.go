package runtime2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func (parseHostRegionAdapter *HostRegionAdapter) clearHostRegionSourceSnapshotCache() {
	if parseHostRegionAdapter == nil {
		return
	}
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheEpoch = 0
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersion = 0
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceIDs = nil
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheVersionTuple = nil
	parseHostRegionAdapter.storeHostRegionSourceSnapshotScratchVersionTuple = nil
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceValues = nil
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersions = nil
	parseHostRegionAdapter.hasHostRegionSourceSnapshotCache = false
	parseHostRegionAdapter.clearHostRegionSnapshotPropsCache()
}

// SetHostRegionSourceLookup sets the host-side bridge used to look up declared source values and versions.
func (parseHostRegionAdapter *HostRegionAdapter) SetHostRegionSourceLookup(parseSourceLookup HostRegionSourceLookup) error {
	if parseHostRegionAdapter == nil {
		return fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseSourceLookup == nil {
		return fmt.Errorf("runtime2: host source lookup is required")
	}
	parseHostRegionAdapter.storeHostRegionSourceLookup = parseSourceLookup
	parseHostRegionAdapter.clearHostRegionSourceSnapshotCache()
	return nil
}

// HandleHostRegionDeclaredSourceLookup resolves canonical declared source IDs through the configured host source lookup bridge.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionDeclaredSourceLookup(parseSourceIDs []string) (HostRegionSourceSnapshot, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionSourceSnapshot{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	getSourceIDs, parseSourceIDsErr := NormalizeSourceIDs(parseSourceIDs)
	if parseSourceIDsErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceIDsErr
	}
	return parseHostRegionAdapter.handleHostRegionDeclaredSourceLookupNormalized(getSourceIDs)
}

// handleHostRegionDeclaredSourceLookupNormalized resolves one normalized declared source ID set through the host source lookup bridge.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionDeclaredSourceLookupNormalized(parseSourceIDs []string) (HostRegionSourceSnapshot, error) {
	return parseHostRegionAdapter.handleHostRegionDeclaredSourceLookupForEpochNormalized(0, parseSourceIDs)
}

// handleHostRegionDeclaredSourceLookupForEpochNormalized resolves one normalized declared source ID set and reuses cached source maps for stable epoch plus version tuples.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionDeclaredSourceLookupForEpochNormalized(
	parseSourceEpoch uint64,
	parseSourceIDs []string,
) (HostRegionSourceSnapshot, error) {
	if len(parseSourceIDs) == 0 {
		return HostRegionSourceSnapshot{}, nil
	}
	if parseHostRegionAdapter.storeHostRegionSourceLookup == nil {
		return HostRegionSourceSnapshot{}, fmt.Errorf("runtime2: host source lookup is not configured")
	}
	getSourceValues, getSourceVersions, parseSourceLookupErr := parseHostRegionAdapter.storeHostRegionSourceLookup(parseSourceIDs)
	if parseSourceLookupErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceLookupErr
	}
	buildSourceVersionTuple, parseSourceVersionTupleErr := buildHostRegionSourceSnapshotVersionTuple(
		parseHostRegionAdapter.storeHostRegionSourceSnapshotScratchVersionTuple[:0],
		parseSourceIDs,
		getSourceVersions,
	)
	parseHostRegionAdapter.storeHostRegionSourceSnapshotScratchVersionTuple = buildSourceVersionTuple[:0]
	if parseSourceVersionTupleErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceVersionTupleErr
	}
	if parseHostRegionAdapter.hasHostRegionSourceSnapshotCache &&
		parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheEpoch == parseSourceEpoch &&
		parseHasHostRegionExactSourceIDList(parseSourceIDs, parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceIDs) &&
		parseHasHostRegionExactSourceVersionTuple(
			buildSourceVersionTuple,
			parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheVersionTuple,
		) {
		return HostRegionSourceSnapshot{
			GetSourceValues:   parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceValues,
			GetSourceVersions: parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersions,
			GetSourceVersion:  parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersion,
		}, nil
	}
	buildSourceValues, getSourceVersion, parseSourceValuesErr := buildSnapshotSourceValuesAndVersion(
		parseSourceIDs,
		getSourceValues,
		getSourceVersions,
	)
	if parseSourceValuesErr != nil {
		return HostRegionSourceSnapshot{}, parseSourceValuesErr
	}
	buildSourceVersions := buildHostRegionSourceSnapshotVersionMap(parseSourceIDs, buildSourceVersionTuple)
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheEpoch = parseSourceEpoch
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersion = getSourceVersion
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceIDs = append(
		parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceIDs[:0],
		parseSourceIDs...,
	)
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheVersionTuple = append(
		parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheVersionTuple[:0],
		buildSourceVersionTuple...,
	)
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceValues = buildSourceValues
	parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersions = buildSourceVersions
	parseHostRegionAdapter.hasHostRegionSourceSnapshotCache = true
	return HostRegionSourceSnapshot{
		GetSourceValues:   buildSourceValues,
		GetSourceVersions: buildSourceVersions,
		GetSourceVersion:  getSourceVersion,
	}, nil
}

// HandleHostRegionUpdateSnapshot captures one update snapshot envelope from normalized props plus declared source lookup.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionUpdateSnapshot(parseSpec ParallelRegionSpec, parseInputVersion uint64) (SnapshotEnvelope, error) {
	getSnapshotEnvelope, _, _, _, _, _, parseSnapshotErr := parseHostRegionAdapter.handleHostRegionUpdateSnapshot(parseSpec, parseInputVersion, true)
	if parseSnapshotErr != nil {
		return SnapshotEnvelope{}, parseSnapshotErr
	}
	return getSnapshotEnvelope, nil
}

// handleHostRegionUpdateSnapshot captures one update snapshot and optionally stores snapshot-version coordinator state.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionUpdateSnapshot(
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseShouldStoreSnapshotVersion bool,
) (SnapshotEnvelope, bool, []string, bool, uint64, uint64, error) {
	if parseHostRegionAdapter == nil {
		return SnapshotEnvelope{}, false, nil, false, 0, 0, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseInputVersion == 0 {
		return SnapshotEnvelope{}, false, nil, false, 0, 0, fmt.Errorf("runtime2: host update snapshot input version is required")
	}
	getEntryEpoch,
		getEntryRendererID,
		getEntrySourceIDs,
		getEntryIsFallback,
		getEntryLastSnapshotVersion,
		getEntryLastDispatchedVersion,
		hasCoordinatorEntry := parseHostRegionAdapter.getHostRegionCoordinatorSnapshotDispatchState(!parseShouldStoreSnapshotVersion)
	if !hasCoordinatorEntry {
		return SnapshotEnvelope{}, false, nil, false, 0, 0, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	getSpec := parseSpec
	hasSnapshotPropsCacheHit := false
	var getSnapshotPropsCacheToken uint64
	hasSnapshotPropsCacheToken := false
	var getSnapshotPropsShapeKeyCount uint64
	var getSnapshotPropsShapeKeyHash uint64
	var getSnapshotPropsShapeTypeHash uint64
	var getSnapshotPropsShapeKeys []string
	var getSnapshotPropsShapeTypeMarkers []uint64
	hasSnapshotPropsShapeFingerprint := false
	hasExactSourceIDs := parseHasHostRegionExactSourceIDList(parseSpec.SourceIDs, getEntrySourceIDs)
	if parseSpec.RegionInstanceID == parseHostRegionAdapter.storeRegionInstanceID &&
		parseSpec.RendererID == getEntryRendererID &&
		hasExactSourceIDs {
		getSnapshotPropsCacheToken, hasSnapshotPropsCacheToken = buildHostRegionSnapshotImmutablePropsToken(parseSpec.Props)
		if hasSnapshotPropsCacheToken &&
			parseHostRegionAdapter.hasHostRegionSnapshotPropsCacheToken &&
			getSnapshotPropsCacheToken == parseHostRegionAdapter.storeHostRegionSnapshotPropsCacheToken {
			hasSnapshotPropsCacheHit = true
		} else {
			if parseHostRegionAdapter.hasHostRegionSnapshotPropsShapeFingerprint &&
				hasSerializablePropsFlatShapeFingerprintMatch(
					parseSpec.Props,
					parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyCount,
					parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys,
					parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers,
				) {
				hasSnapshotPropsCacheHit = true
			} else {
				if parsePropsErr := ValidateSerializableProps(parseSpec.Props); parsePropsErr != nil {
					return SnapshotEnvelope{}, false, nil, false, 0, 0, parsePropsErr
				}
			}
		}
	} else {
		getNormalizedSpec, parseSpecErr := NormalizeParallelRegionSpec(parseSpec)
		if parseSpecErr != nil {
			return SnapshotEnvelope{}, false, nil, false, 0, 0, parseSpecErr
		}
		getSpec = getNormalizedSpec
		if getSpec.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
			return SnapshotEnvelope{}, false, nil, false, 0, 0, fmt.Errorf(
				"runtime2: host region adapter mounted for %q cannot capture snapshot for region %q",
				parseHostRegionAdapter.storeRegionInstanceID,
				getSpec.RegionInstanceID,
			)
		}
		if getEntryRendererID != getSpec.RendererID {
			return SnapshotEnvelope{}, false, nil, false, 0, 0, fmt.Errorf(
				"runtime2: mounted renderer ID %q does not match snapshot renderer ID %q",
				getEntryRendererID,
				getSpec.RendererID,
			)
		}
		hasExactSourceIDs = parseHasHostRegionExactSourceIDList(getSpec.SourceIDs, getEntrySourceIDs)
	}
	getSourceSnapshot, parseSourceSnapshotErr := parseHostRegionAdapter.handleHostRegionDeclaredSourceLookupForEpochNormalized(
		getEntryEpoch,
		getSpec.SourceIDs,
	)
	if parseSourceSnapshotErr != nil {
		return SnapshotEnvelope{}, false, nil, false, 0, 0, parseSourceSnapshotErr
	}
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: getSpec.RegionInstanceID,
		Epoch:            getEntryEpoch,
		InputVersion:     parseInputVersion,
		SourceVersion:    getSourceSnapshot.GetSourceVersion,
		Props:            getSpec.Props,
		Sources:          getSourceSnapshot.GetSourceValues,
	}
	if !hasSnapshotPropsCacheHit {
		if !hasSnapshotPropsCacheToken {
			getSnapshotPropsCacheToken, hasSnapshotPropsCacheToken = buildHostRegionSnapshotImmutablePropsToken(getSpec.Props)
		}
		if hasSnapshotPropsCacheToken {
			parseHostRegionAdapter.storeHostRegionSnapshotPropsCacheToken = getSnapshotPropsCacheToken
			parseHostRegionAdapter.hasHostRegionSnapshotPropsCacheToken = true
			parseHostRegionAdapter.hasHostRegionSnapshotPropsShapeFingerprint = false
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyCount = 0
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyHash = 0
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeHash = 0
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys = parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys[:0]
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers = parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers[:0]
		} else {
			if !hasSnapshotPropsShapeFingerprint {
				getSnapshotPropsShapeKeyCount,
					getSnapshotPropsShapeKeyHash,
					getSnapshotPropsShapeTypeHash,
					getSnapshotPropsShapeKeys,
					getSnapshotPropsShapeTypeMarkers,
					hasSnapshotPropsShapeFingerprint = buildSerializablePropsFlatShapeFingerprintWithTypeScratch(
					getSpec.Props,
					parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeScratchKeys,
					parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeScratchTypes,
				)
				parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeScratchKeys = getSnapshotPropsShapeKeys[:0]
				parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeScratchTypes = getSnapshotPropsShapeTypeMarkers[:0]
			}
			if hasSnapshotPropsShapeFingerprint {
				parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyCount = getSnapshotPropsShapeKeyCount
				parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyHash = getSnapshotPropsShapeKeyHash
				parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeHash = getSnapshotPropsShapeTypeHash
				parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys = append(
					parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys[:0],
					getSnapshotPropsShapeKeys...,
				)
				parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers = append(
					parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers[:0],
					getSnapshotPropsShapeTypeMarkers...,
				)
				parseHostRegionAdapter.hasHostRegionSnapshotPropsShapeFingerprint = true
				parseHostRegionAdapter.hasHostRegionSnapshotPropsCacheToken = false
			} else {
				parseHostRegionAdapter.clearHostRegionSnapshotPropsCache()
			}
		}
		if !hasSnapshotPropsCacheToken && hasSnapshotPropsShapeFingerprint {
			parseHostRegionAdapter.storeHostRegionSnapshotPropsCacheToken = 0
		} else {
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyCount = 0
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyHash = 0
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeHash = 0
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys = parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys[:0]
			parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers = parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers[:0]
		}
	}
	parseHostRegionAdapter.storeHostRegionDispatchPropsLayout(getSpec.Props)
	if parseShouldStoreSnapshotVersion {
		if parseSnapshotStateErr := parseHostRegionAdapter.storeCoordinator.applyRegionSnapshotState(
			getSpec.RegionInstanceID,
			parseInputVersion,
			getSpec.SourceIDs,
			!hasExactSourceIDs,
		); parseSnapshotStateErr != nil {
			return SnapshotEnvelope{}, false, nil, false, 0, 0, parseSnapshotStateErr
		}
		parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotState(
			parseInputVersion,
			getSpec.SourceIDs,
			!hasExactSourceIDs,
		)
	}
	return getSnapshotEnvelope,
		hasExactSourceIDs,
		getSpec.SourceIDs,
		getEntryIsFallback,
		getEntryLastSnapshotVersion,
		getEntryLastDispatchedVersion,
		nil
}

// HandleHostRegionSnapshotFingerprint computes and stores one stable snapshot fingerprint for host-side no-change detection.
func (parseHostRegionAdapter *HostRegionAdapter) HandleHostRegionSnapshotFingerprint(parseSnapshotEnvelope SnapshotEnvelope) (HostRegionSnapshotFingerprintResult, error) {
	if parseSnapshotErr := ValidateSnapshotEnvelope(parseSnapshotEnvelope); parseSnapshotErr != nil {
		return HostRegionSnapshotFingerprintResult{}, parseSnapshotErr
	}
	getSnapshotHashResult, parseSnapshotHashErr := parseHostRegionAdapter.handleHostRegionSnapshotHash(parseSnapshotEnvelope)
	if parseSnapshotHashErr != nil {
		return HostRegionSnapshotFingerprintResult{}, parseSnapshotHashErr
	}
	if getSnapshotHashResult.hasNoChange && parseHostRegionAdapter.storeHostRegionSnapshotFingerprint != "" {
		return HostRegionSnapshotFingerprintResult{
			GetSnapshotFingerprint: parseHostRegionAdapter.storeHostRegionSnapshotFingerprint,
			HasNoChange:            true,
		}, nil
	}
	getSnapshotFingerprint := hex.EncodeToString(getSnapshotHashResult.getSnapshotHash[:])
	parseHostRegionAdapter.storeHostRegionSnapshotFingerprint = getSnapshotFingerprint
	return HostRegionSnapshotFingerprintResult{
		GetSnapshotFingerprint: getSnapshotFingerprint,
		HasNoChange:            getSnapshotHashResult.hasNoChange,
	}, nil
}

// handleHostRegionSnapshotHash computes and stores one snapshot SHA-256 digest for host-side no-change detection.
// A fast non-cryptographic prefilter hash is applied first so the SHA-256 fingerprint path is skipped when content is unchanged.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionSnapshotHash(parseSnapshotEnvelope SnapshotEnvelope) (hostRegionSnapshotHashResult, error) {
	if parseHostRegionAdapter == nil {
		return hostRegionSnapshotHashResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseSnapshotEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return hostRegionSnapshotHashResult{}, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot fingerprint snapshot for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseSnapshotEnvelope.RegionInstanceID,
		)
	}
	// Compute the canonical dispatch payload to derive one fast prefilter hash.
	// The fast dispatch hash path intentionally ignores InputVersion, so no envelope copy is needed here.
	getSnapshotFastHash, getSnapshotHashScratch, parseSnapshotFastHashErr := buildSnapshotDispatchFastHashInto(
		parseSnapshotEnvelope,
		parseHostRegionAdapter.storeHostRegionSnapshotHashScratch[:0],
	)
	if parseSnapshotFastHashErr != nil {
		return hostRegionSnapshotHashResult{}, parseSnapshotFastHashErr
	}
	parseHostRegionAdapter.storeHostRegionSnapshotHashScratch = getSnapshotHashScratch
	if parseHostRegionAdapter.hasHostRegionSnapshotFastHash &&
		getSnapshotFastHash == parseHostRegionAdapter.storeHostRegionSnapshotFastHash &&
		parseHostRegionAdapter.hasHostRegionSnapshotHash {
		// Fast prefilter hash matched: content is unchanged, skip SHA-256 recomputation.
		return hostRegionSnapshotHashResult{
			getSnapshotHash: parseHostRegionAdapter.storeHostRegionSnapshotHash,
			hasNoChange:     true,
		}, nil
	}
	// Fast prefilter hash mismatch or no stored state: compute SHA-256 and update stored digests.
	buildFingerprintEnvelope := parseSnapshotEnvelope
	buildFingerprintEnvelope.InputVersion = 1
	getSnapshotHash, parseSnapshotHashErr := getSnapshotFingerprintHashWithoutValidation(buildFingerprintEnvelope)
	if parseSnapshotHashErr != nil {
		return hostRegionSnapshotHashResult{}, parseSnapshotHashErr
	}
	hasNoChange := parseHostRegionAdapter.hasHostRegionSnapshotHash && getSnapshotHash == parseHostRegionAdapter.storeHostRegionSnapshotHash
	parseHostRegionAdapter.storeHostRegionSnapshotHash = getSnapshotHash
	parseHostRegionAdapter.hasHostRegionSnapshotHash = true
	parseHostRegionAdapter.storeHostRegionSnapshotFastHash = getSnapshotFastHash
	parseHostRegionAdapter.hasHostRegionSnapshotFastHash = true
	return hostRegionSnapshotHashResult{
		getSnapshotHash: getSnapshotHash,
		hasNoChange:     hasNoChange,
	}, nil
}

// handleHostRegionDispatchHash computes and stores one dispatch-local snapshot hash for no-change scheduling short-circuits.
func (parseHostRegionAdapter *HostRegionAdapter) handleHostRegionDispatchHash(
	parseSnapshotEnvelope SnapshotEnvelope,
	parseSourceIDs []string,
	parseRendererID RendererID,
) (bool, error) {
	if parseHostRegionAdapter == nil {
		return false, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseSnapshotEnvelope.RegionInstanceID != parseHostRegionAdapter.storeRegionInstanceID {
		return false, fmt.Errorf(
			"runtime2: host region adapter mounted for %q cannot fingerprint snapshot for region %q",
			parseHostRegionAdapter.storeRegionInstanceID,
			parseSnapshotEnvelope.RegionInstanceID,
		)
	}
	getDispatchSourceVersionTuple, hasDispatchSourceVersionTuple := buildHostRegionDispatchSourceVersionTuple(
		parseHostRegionAdapter.storeHostRegionDispatchSourceVersionScratch[:0],
		parseSourceIDs,
		parseHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersions,
	)
	parseHostRegionAdapter.storeHostRegionDispatchSourceVersionScratch = getDispatchSourceVersionTuple[:0]
	if parseHostRegionAdapter.hasHostRegionDispatchVersionVector {
		hasDispatchVectorBaseMatch := parseRendererID == parseHostRegionAdapter.storeHostRegionDispatchRendererID &&
			parseSnapshotEnvelope.Epoch == parseHostRegionAdapter.storeHostRegionDispatchEpoch
		if !hasDispatchVectorBaseMatch {
			parseHostRegionAdapter.storeHostRegionDispatchVersionVector(
				parseRendererID,
				parseSnapshotEnvelope,
				getDispatchSourceVersionTuple,
				hasDispatchSourceVersionTuple,
			)
			parseHostRegionAdapter.clearHostRegionDispatchDigestState()
			return false, nil
		}
		hasDispatchSourceVersionMatch := false
		if hasDispatchSourceVersionTuple && parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple {
			hasDispatchSourceVersionMatch = parseHasHostRegionExactSourceVersionTuple(
				getDispatchSourceVersionTuple,
				parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple,
			)
		} else if !hasDispatchSourceVersionTuple && !parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple {
			hasDispatchSourceVersionMatch = parseSnapshotEnvelope.SourceVersion == parseHostRegionAdapter.storeHostRegionDispatchSourceVersion
		}
		if !hasDispatchSourceVersionMatch {
			parseHostRegionAdapter.storeHostRegionDispatchVersionVector(
				parseRendererID,
				parseSnapshotEnvelope,
				getDispatchSourceVersionTuple,
				hasDispatchSourceVersionTuple,
			)
			parseHostRegionAdapter.clearHostRegionDispatchDigestState()
			return false, nil
		}
	}
	if !parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh {
		parseHostRegionAdapter.storeHostRegionDispatchPropsLayout(parseSnapshotEnvelope.Props)
	}
	getDispatchFastHash, getDispatchScratch, parseDispatchFastHashErr := buildSnapshotDispatchFastHashIntoWithSourceAndPropsLayout(
		parseSnapshotEnvelope,
		parseSourceIDs,
		parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys,
		parseHostRegionAdapter.storeHostRegionDispatchPropsEntries,
		parseHostRegionAdapter.storeHostRegionDispatchScratch[:0],
	)
	if parseDispatchFastHashErr != nil {
		return false, parseDispatchFastHashErr
	}
	parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh = false
	parseHostRegionAdapter.storeHostRegionDispatchScratch = getDispatchScratch
	hasDispatchNoChange := parseHostRegionAdapter.hasHostRegionDispatchFastHash &&
		getDispatchFastHash == parseHostRegionAdapter.storeHostRegionDispatchFastHash
	parseHostRegionAdapter.storeHostRegionDispatchVersionVector(
		parseRendererID,
		parseSnapshotEnvelope,
		getDispatchSourceVersionTuple,
		hasDispatchSourceVersionTuple,
	)
	parseHostRegionAdapter.storeHostRegionDispatchFastHash = getDispatchFastHash
	parseHostRegionAdapter.hasHostRegionDispatchFastHash = true
	parseHostRegionAdapter.storeHostRegionDispatchHash = [sha256.Size]byte{}
	parseHostRegionAdapter.hasHostRegionDispatchHash = false
	parseHostRegionAdapter.storeHostRegionDispatchBytes = nil
	parseHostRegionAdapter.hasHostRegionDispatchBytes = false
	return hasDispatchNoChange, nil
}

// HandleHostRegionUpdateDispatch captures one update snapshot, applies no-change short-circuit rules, and schedules worker update dispatch only when needed.
// It routes directly to the inner dispatch path with the known urgent priority constant to avoid ParseHostRegionDispatchPriority overhead on every call.
