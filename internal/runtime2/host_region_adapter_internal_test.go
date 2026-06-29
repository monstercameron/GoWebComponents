package runtime2

import (
	"math"
	"testing"
)

// buildMountedHostRegionAdapterForHelperTest creates one mounted host region adapter for direct helper coverage.
func buildMountedHostRegionAdapterForHelperTest(parseT *testing.T) *HostRegionAdapter {
	parseT.Helper()
	parseHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
		SourceIDs:        []string{"count", "status"},
	}, 7); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	return parseHostRegionAdapter
}

// TestHostRegionDispatchPriorityAndSourceVersionHelpers verifies host adapter helper functions validate dispatch priority and build ordered source-version state.
func TestHostRegionDispatchPriorityAndSourceVersionHelpers(parseT *testing.T) {
	if getPriority, parseErr := ParseHostRegionDispatchPriority(HostRegionDispatchPriorityUrgent); parseErr != nil || getPriority != HostRegionDispatchPriorityUrgent {
		parseT.Fatalf("ParseHostRegionDispatchPriority(urgent) = (%q, %v), want (%q, nil)", getPriority, parseErr, HostRegionDispatchPriorityUrgent)
	}
	if getPriority, parseErr := ParseHostRegionDispatchPriority(HostRegionDispatchPriorityDeferred); parseErr != nil || getPriority != HostRegionDispatchPriorityDeferred {
		parseT.Fatalf("ParseHostRegionDispatchPriority(deferred) = (%q, %v), want (%q, nil)", getPriority, parseErr, HostRegionDispatchPriorityDeferred)
	}
	if _, parseErr := ParseHostRegionDispatchPriority("later"); parseErr == nil {
		parseT.Fatal("expected unsupported dispatch priority to fail")
	}

	if getMaxVersion := parseHostRegionMaxVersion(0, 3, 11, 7); getMaxVersion != 11 {
		parseT.Fatalf("parseHostRegionMaxVersion = %d, want 11", getMaxVersion)
	}
	if !parseHasHostRegionExactSourceIDList([]string{"count", "status"}, []string{"count", "status"}) {
		parseT.Fatal("expected exact source ID lists to match")
	}
	if parseHasHostRegionExactSourceIDList([]string{"count", "status"}, []string{"status", "count"}) {
		parseT.Fatal("expected reordered source ID lists to differ")
	}
	if !parseHasHostRegionExactSourceVersionTuple([]uint64{2, 5}, []uint64{2, 5}) {
		parseT.Fatal("expected exact source version tuples to match")
	}
	if parseHasHostRegionExactSourceVersionTuple([]uint64{2, 5}, []uint64{2}) {
		parseT.Fatal("expected mismatched source version tuples to differ")
	}

	getSnapshotTuple, parseTupleErr := buildHostRegionSourceSnapshotVersionTuple(nil, []string{"count", "status"}, map[string]uint64{
		"count":  2,
		"status": 5,
	})
	if parseTupleErr != nil {
		parseT.Fatalf("buildHostRegionSourceSnapshotVersionTuple returned error: %v", parseTupleErr)
	}
	if len(getSnapshotTuple) != 2 || getSnapshotTuple[0] != 2 || getSnapshotTuple[1] != 5 {
		parseT.Fatalf("snapshot version tuple = %+v, want [2 5]", getSnapshotTuple)
	}
	if getEmptyTuple, parseEmptyErr := buildHostRegionSourceSnapshotVersionTuple(make([]uint64, 0, 1), nil, map[string]uint64{"count": 2}); parseEmptyErr != nil || len(getEmptyTuple) != 0 {
		parseT.Fatalf("buildHostRegionSourceSnapshotVersionTuple(empty) = (%+v, %v), want ([], nil)", getEmptyTuple, parseEmptyErr)
	}
	if _, parseMissingErr := buildHostRegionSourceSnapshotVersionTuple(nil, []string{"count", "status"}, map[string]uint64{"count": 2}); parseMissingErr == nil {
		parseT.Fatal("expected missing snapshot source version to fail")
	}

	if getSourceVersionMap := buildHostRegionSourceSnapshotVersionMap(nil, nil); getSourceVersionMap != nil {
		parseT.Fatalf("expected empty source version map to be nil, got %+v", getSourceVersionMap)
	}
	getSourceVersionMap := buildHostRegionSourceSnapshotVersionMap([]string{"count", "status"}, []uint64{2, 5})
	if len(getSourceVersionMap) != 2 || getSourceVersionMap["count"] != 2 || getSourceVersionMap["status"] != 5 {
		parseT.Fatalf("source version map = %+v, want count=2 status=5", getSourceVersionMap)
	}

	getDispatchTuple, hasDispatchTuple := buildHostRegionDispatchSourceVersionTuple(nil, []string{"count", "status"}, map[string]uint64{
		"count":  8,
		"status": 13,
	})
	if !hasDispatchTuple || len(getDispatchTuple) != 2 || getDispatchTuple[0] != 8 || getDispatchTuple[1] != 13 {
		parseT.Fatalf("dispatch source version tuple = (%+v, %t), want ([8 13], true)", getDispatchTuple, hasDispatchTuple)
	}
	if getEmptyDispatchTuple, hasEmptyDispatchTuple := buildHostRegionDispatchSourceVersionTuple(make([]uint64, 0, 1), nil, nil); !hasEmptyDispatchTuple || len(getEmptyDispatchTuple) != 0 {
		parseT.Fatalf("buildHostRegionDispatchSourceVersionTuple(empty) = (%+v, %t), want ([], true)", getEmptyDispatchTuple, hasEmptyDispatchTuple)
	}
	if _, hasDispatchTuple := buildHostRegionDispatchSourceVersionTuple(nil, []string{"count"}, nil); hasDispatchTuple {
		parseT.Fatal("expected missing dispatch source versions to fail")
	}
	if _, hasDispatchTuple := buildHostRegionDispatchSourceVersionTuple(nil, []string{"count", "status"}, map[string]uint64{"count": 8}); hasDispatchTuple {
		parseT.Fatal("expected partial dispatch source versions to fail")
	}
}

// TestBuildHostRegionSnapshotImmutablePropsTokenSupportsScalarTypes verifies immutable-props cache tokens stay stable for supported scalar inputs and reject unsupported shapes.
func TestBuildHostRegionSnapshotImmutablePropsTokenSupportsScalarTypes(parseT *testing.T) {
	const (
		getHostRegionPropsTokenSeed  uint64 = 14695981039346656037
		getHostRegionPropsTokenPrime uint64 = 1099511628211
	)
	parseTokenTests := []struct {
		name      string
		getProps  any
		wantToken uint64
		wantOK    bool
	}{
		{name: "nil", getProps: nil, wantToken: 1, wantOK: true},
		{name: "bool-false", getProps: false, wantToken: 2, wantOK: true},
		{name: "bool-true", getProps: true, wantToken: 3, wantOK: true},
		{name: "int", getProps: int(4), wantToken: uint64(int64(4))*getHostRegionPropsTokenPrime + 11, wantOK: true},
		{name: "int8", getProps: int8(5), wantToken: uint64(int64(5))*getHostRegionPropsTokenPrime + 12, wantOK: true},
		{name: "int16", getProps: int16(6), wantToken: uint64(int64(6))*getHostRegionPropsTokenPrime + 13, wantOK: true},
		{name: "int32", getProps: int32(7), wantToken: uint64(int64(7))*getHostRegionPropsTokenPrime + 14, wantOK: true},
		{name: "int64", getProps: int64(8), wantToken: uint64(8)*getHostRegionPropsTokenPrime + 15, wantOK: true},
		{name: "uint", getProps: uint(9), wantToken: uint64(9)*getHostRegionPropsTokenPrime + 16, wantOK: true},
		{name: "uint8", getProps: uint8(10), wantToken: uint64(10)*getHostRegionPropsTokenPrime + 17, wantOK: true},
		{name: "uint16", getProps: uint16(11), wantToken: uint64(11)*getHostRegionPropsTokenPrime + 18, wantOK: true},
		{name: "uint32", getProps: uint32(12), wantToken: uint64(12)*getHostRegionPropsTokenPrime + 19, wantOK: true},
		{name: "uint64", getProps: uint64(13), wantToken: uint64(13)*getHostRegionPropsTokenPrime + 20, wantOK: true},
		{name: "uintptr", getProps: uintptr(14), wantToken: uint64(14)*getHostRegionPropsTokenPrime + 21, wantOK: true},
		{name: "float32", getProps: float32(1.25), wantToken: uint64(math.Float32bits(1.25))*getHostRegionPropsTokenPrime + 22, wantOK: true},
		{name: "float64", getProps: 2.5, wantToken: math.Float64bits(2.5)*getHostRegionPropsTokenPrime + 23, wantOK: true},
		{name: "string", getProps: "props", wantToken: func() uint64 {
			getToken := getHostRegionPropsTokenSeed ^ 31
			for _, getByte := range []byte("props") {
				getToken ^= uint64(getByte)
				getToken *= getHostRegionPropsTokenPrime
			}
			return getToken
		}(), wantOK: true},
		{name: "unsupported", getProps: struct{ getValue string }{getValue: "x"}, wantToken: 0, wantOK: false},
	}
	for _, parseTokenTest := range parseTokenTests {
		getToken, hasToken := buildHostRegionSnapshotImmutablePropsToken(parseTokenTest.getProps)
		if hasToken != parseTokenTest.wantOK || getToken != parseTokenTest.wantToken {
			parseT.Fatalf("%s token = (%d, %t), want (%d, %t)", parseTokenTest.name, getToken, hasToken, parseTokenTest.wantToken, parseTokenTest.wantOK)
		}
	}
}

// TestHostRegionDispatchPropsLayoutHelpersSortReuseAndClear verifies dispatch props helpers sort stable key order, reuse known layouts, and clear stale state.
func TestHostRegionDispatchPropsLayoutHelpersSortReuseAndClear(parseT *testing.T) {
	parseProps := map[string]any{"beta": 2, "alpha": 1}
	if parseHasHostRegionExactPropsKeyList(nil, []string{"alpha"}) {
		parseT.Fatal("expected nil props map not to match cached keys")
	}
	if !parseHasHostRegionExactPropsKeyList(parseProps, []string{"alpha", "beta"}) {
		parseT.Fatal("expected exact props key list to match")
	}
	if parseHasHostRegionExactPropsKeyList(parseProps, []string{"alpha"}) {
		parseT.Fatal("expected props key count mismatch to fail exact-key test")
	}

	getOrderedKeys := buildHostRegionDispatchPropsOrderedKeys(nil, parseProps)
	if len(getOrderedKeys) != 2 || getOrderedKeys[0] != "alpha" || getOrderedKeys[1] != "beta" {
		parseT.Fatalf("ordered props keys = %+v, want [alpha beta]", getOrderedKeys)
	}
	getPropsEntries, hasPropsEntries := buildHostRegionDispatchPropsEntries(nil, parseProps, getOrderedKeys)
	if !hasPropsEntries || len(getPropsEntries) != 2 || getPropsEntries[0].getKey != "alpha" || getPropsEntries[0].getValue != 1 || getPropsEntries[1].getKey != "beta" || getPropsEntries[1].getValue != 2 {
		parseT.Fatalf("dispatch props entries = (%+v, %t), want alpha/beta entries", getPropsEntries, hasPropsEntries)
	}
	if getMissingEntries, hasPropsEntries := buildHostRegionDispatchPropsEntries(nil, parseProps, []string{"alpha", "gamma"}); hasPropsEntries || len(getMissingEntries) != 0 {
		parseT.Fatalf("expected missing props entry build to fail, got (%+v, %t)", getMissingEntries, hasPropsEntries)
	}

	var parseNilHostRegionAdapter *HostRegionAdapter
	parseNilHostRegionAdapter.clearHostRegionDispatchPropsLayout()
	parseNilHostRegionAdapter.storeHostRegionDispatchPropsLayout(parseProps)

	parseHostRegionAdapter := &HostRegionAdapter{
		storeHostRegionDispatchPropsEntries:     []buildSnapshotDispatchMapEntry{{getKey: "stale", getValue: 1}},
		storeHostRegionDispatchPropsOrderedKeys: []string{"stale"},
		storeHostRegionDispatchPropsScratchKeys: []string{"scratch"},
		hasHostRegionDispatchPropsLayoutFresh:   true,
	}
	parseHostRegionAdapter.clearHostRegionDispatchPropsLayout()
	if len(parseHostRegionAdapter.storeHostRegionDispatchPropsEntries) != 0 || len(parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys) != 0 || len(parseHostRegionAdapter.storeHostRegionDispatchPropsScratchKeys) != 0 || parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh {
		parseT.Fatalf("dispatch props layout clear did not reset state: %+v", parseHostRegionAdapter)
	}

	parseHostRegionAdapter.storeHostRegionDispatchPropsLayout(struct{}{})
	if len(parseHostRegionAdapter.storeHostRegionDispatchPropsEntries) != 0 || parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh {
		parseT.Fatalf("non-map props should leave cleared dispatch layout, got %+v", parseHostRegionAdapter.storeHostRegionDispatchPropsEntries)
	}

	parseHostRegionAdapter.storeHostRegionDispatchPropsLayout(parseProps)
	if !parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh {
		parseT.Fatal("expected fresh dispatch props layout after storing map props")
	}
	if len(parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys) != 2 || parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys[0] != "alpha" || parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys[1] != "beta" {
		parseT.Fatalf("stored ordered keys = %+v, want [alpha beta]", parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys)
	}

	parseHostRegionAdapter.hasHostRegionDispatchPropsLayoutFresh = false
	parseHostRegionAdapter.storeHostRegionDispatchPropsLayout(map[string]any{"alpha": 10, "beta": 20})
	if parseHostRegionAdapter.storeHostRegionDispatchPropsEntries[0].getValue != 10 || parseHostRegionAdapter.storeHostRegionDispatchPropsEntries[1].getValue != 20 {
		parseT.Fatalf("expected existing ordered key layout to be reused, got %+v", parseHostRegionAdapter.storeHostRegionDispatchPropsEntries)
	}

	parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys = nil
	parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys = []string{"delta", "gamma"}
	parseHostRegionAdapter.storeHostRegionDispatchPropsLayout(map[string]any{"gamma": 3, "delta": 4})
	if len(parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys) != 2 || parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys[0] != "delta" || parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys[1] != "gamma" {
		parseT.Fatalf("expected snapshot props shape key reuse, got %+v", parseHostRegionAdapter.storeHostRegionDispatchPropsOrderedKeys)
	}
}

// TestHostRegionDispatchDigestAndSnapshotCacheHelpersResetState verifies digest and snapshot cache helpers clear all cached adapter fields.
func TestHostRegionDispatchDigestAndSnapshotCacheHelpersResetState(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	parseNilHostRegionAdapter.clearHostRegionDispatchDigestState()
	parseNilHostRegionAdapter.clearHostRegionSnapshotPropsCache()

	parseHostRegionAdapter := &HostRegionAdapter{
		storeHostRegionDispatchHash:                  [32]byte{1, 2, 3},
		storeHostRegionDispatchBytes:                 []byte{4, 5},
		storeHostRegionDispatchFastHash:              9,
		hasHostRegionDispatchHash:                    true,
		hasHostRegionDispatchBytes:                   true,
		hasHostRegionDispatchFastHash:                true,
		storeHostRegionSnapshotPropsCacheToken:       11,
		storeHostRegionSnapshotPropsShapeKeyCount:    2,
		storeHostRegionSnapshotPropsShapeKeyHash:     13,
		storeHostRegionSnapshotPropsShapeTypeHash:    17,
		storeHostRegionSnapshotPropsShapeKeys:        []string{"alpha", "beta"},
		storeHostRegionSnapshotPropsShapeTypeMarkers: []uint64{1, 2},
		hasHostRegionSnapshotPropsCacheToken:         true,
		hasHostRegionSnapshotPropsShapeFingerprint:   true,
	}
	parseHostRegionAdapter.clearHostRegionDispatchDigestState()
	if parseHostRegionAdapter.hasHostRegionDispatchHash || parseHostRegionAdapter.hasHostRegionDispatchBytes || parseHostRegionAdapter.hasHostRegionDispatchFastHash || parseHostRegionAdapter.storeHostRegionDispatchFastHash != 0 || parseHostRegionAdapter.storeHostRegionDispatchBytes != nil {
		parseT.Fatalf("dispatch digest clear did not reset state: %+v", parseHostRegionAdapter)
	}
	parseHostRegionAdapter.clearHostRegionSnapshotPropsCache()
	if parseHostRegionAdapter.storeHostRegionSnapshotPropsCacheToken != 0 || parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyCount != 0 || parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeyHash != 0 || parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeHash != 0 || len(parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeKeys) != 0 || len(parseHostRegionAdapter.storeHostRegionSnapshotPropsShapeTypeMarkers) != 0 || parseHostRegionAdapter.hasHostRegionSnapshotPropsCacheToken || parseHostRegionAdapter.hasHostRegionSnapshotPropsShapeFingerprint {
		parseT.Fatalf("snapshot props cache clear did not reset state: %+v", parseHostRegionAdapter)
	}
}

// TestHostRegionCoordinatorCacheHelpersStoreClearAndRefresh verifies coordinator cache helpers update local cache state and refresh from mounted coordinator ownership.
func TestHostRegionCoordinatorCacheHelpersStoreClearAndRefresh(parseT *testing.T) {
	if getCopiedSourceIDs := copyHostRegionCoordinatorSourceIDs(make([]string, 0, 2), nil); len(getCopiedSourceIDs) != 0 {
		parseT.Fatalf("expected empty copied source ID list, got %+v", getCopiedSourceIDs)
	}
	getCopiedSourceIDs := copyHostRegionCoordinatorSourceIDs([]string{"stale"}, []string{"count", "status"})
	if len(getCopiedSourceIDs) != 2 || getCopiedSourceIDs[0] != "count" || getCopiedSourceIDs[1] != "status" {
		parseT.Fatalf("copied source IDs = %+v, want [count status]", getCopiedSourceIDs)
	}

	var parseNilHostRegionAdapter *HostRegionAdapter
	parseNilHostRegionAdapter.storeHostRegionCoordinatorCacheState(1, RendererID("dashboard.hot-panel"), []string{"count"}, true, 2, 3)
	parseNilHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotState(4, []string{"status"}, true)
	parseNilHostRegionAdapter.storeHostRegionCoordinatorCacheDispatchedVersion(5)
	parseNilHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotDispatchState(6, 7, []string{"status"}, true)
	if _, _, _, _, _, _, hasCoordinatorEntry := parseNilHostRegionAdapter.getHostRegionCoordinatorSnapshotDispatchState(false); hasCoordinatorEntry {
		parseT.Fatal("expected nil host region adapter cache refresh to fail")
	}

	parseHostRegionAdapter := &HostRegionAdapter{}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotState(4, []string{"status"}, true)
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheDispatchedVersion(5)
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotDispatchState(6, 7, []string{"status"}, true)
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheState(7, RendererID("dashboard.hot-panel"), []string{"count"}, true, 2, 3)
	if !parseHostRegionAdapter.hasHostRegionCoordinatorCache || parseHostRegionAdapter.storeHostRegionCoordinatorCacheEpoch != 7 || parseHostRegionAdapter.storeHostRegionCoordinatorCacheRendererID != RendererID("dashboard.hot-panel") || parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot != 2 || parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched != 3 || !parseHostRegionAdapter.isHostRegionFallbackActive {
		parseT.Fatalf("coordinator cache state = %+v, want populated cache", parseHostRegionAdapter)
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotState(8, []string{"status"}, false)
	if parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot != 8 || len(parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs) != 1 || parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs[0] != "count" {
		parseT.Fatalf("coordinator snapshot-only cache update = %+v, want snapshot update without source replacement", parseHostRegionAdapter)
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheDispatchedVersion(9)
	if parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched != 9 {
		parseT.Fatalf("dispatched version = %d, want 9", parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched)
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheSnapshotDispatchState(10, 11, []string{"status"}, true)
	if parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot != 10 || parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched != 11 || len(parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs) != 1 || parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs[0] != "status" {
		parseT.Fatalf("snapshot+dispatch cache update = %+v, want updated snapshot, dispatch, and source IDs", parseHostRegionAdapter)
	}
	getEpoch, getRendererID, getSourceIDs, isFallback, getLastSnapshotVersion, getLastDispatchedVersion, hasCoordinatorEntry := parseHostRegionAdapter.getHostRegionCoordinatorSnapshotDispatchState(true)
	if !hasCoordinatorEntry || getEpoch != 7 || getRendererID != RendererID("dashboard.hot-panel") || len(getSourceIDs) != 1 || getSourceIDs[0] != "status" || !isFallback || getLastSnapshotVersion != 10 || getLastDispatchedVersion != 11 {
		parseT.Fatalf("cached snapshot-dispatch state = (%d, %q, %+v, %t, %d, %d, %t), want populated cached fields", getEpoch, getRendererID, getSourceIDs, isFallback, getLastSnapshotVersion, getLastDispatchedVersion, hasCoordinatorEntry)
	}
	parseHostRegionAdapter.clearHostRegionCoordinatorCacheState()
	if parseHostRegionAdapter.hasHostRegionCoordinatorCache || parseHostRegionAdapter.storeHostRegionCoordinatorCacheEpoch != 0 || parseHostRegionAdapter.storeHostRegionCoordinatorCacheRendererID != "" || parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastSnapshot != 0 || parseHostRegionAdapter.storeHostRegionCoordinatorCacheLastDispatched != 0 || len(parseHostRegionAdapter.storeHostRegionCoordinatorCacheSourceIDs) != 0 {
		parseT.Fatalf("clearHostRegionCoordinatorCacheState did not reset cache: %+v", parseHostRegionAdapter)
	}

	parseMountedHostRegionAdapter := buildMountedHostRegionAdapterForHelperTest(parseT)
	parseMountedHostRegionAdapter.clearHostRegionCoordinatorCacheState()
	getEpoch, getRendererID, getSourceIDs, isFallback, getLastSnapshotVersion, getLastDispatchedVersion, hasCoordinatorEntry = parseMountedHostRegionAdapter.getHostRegionCoordinatorSnapshotDispatchState(false)
	if !hasCoordinatorEntry || getEpoch != 7 || getRendererID != RendererID("dashboard.hot-panel") || len(getSourceIDs) != 2 || getSourceIDs[0] != "count" || getSourceIDs[1] != "status" || isFallback || getLastSnapshotVersion != 0 || getLastDispatchedVersion != 0 {
		parseT.Fatalf("refreshed coordinator snapshot-dispatch state = (%d, %q, %+v, %t, %d, %d, %t), want mounted coordinator fields", getEpoch, getRendererID, getSourceIDs, isFallback, getLastSnapshotVersion, getLastDispatchedVersion, hasCoordinatorEntry)
	}
}

// TestStoreHostRegionDispatchVersionVectorCopiesAndClearsTuple verifies dispatch version-vector caching copies source tuples when present and clears them when absent.
func TestStoreHostRegionDispatchVersionVectorCopiesAndClearsTuple(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	parseNilHostRegionAdapter.storeHostRegionDispatchVersionVector(RendererID("dashboard.hot-panel"), SnapshotEnvelope{Epoch: 1, InputVersion: 2, SourceVersion: 3}, []uint64{4}, true)

	parseHostRegionAdapter := &HostRegionAdapter{}
	parseSourceVersionTuple := []uint64{5, 8}
	parseHostRegionAdapter.storeHostRegionDispatchVersionVector(
		RendererID("dashboard.hot-panel"),
		SnapshotEnvelope{Epoch: 7, InputVersion: 11, SourceVersion: 13},
		parseSourceVersionTuple,
		true,
	)
	parseSourceVersionTuple[0] = 99
	if parseHostRegionAdapter.storeHostRegionDispatchRendererID != RendererID("dashboard.hot-panel") || parseHostRegionAdapter.storeHostRegionDispatchEpoch != 7 || parseHostRegionAdapter.storeHostRegionDispatchInputVersion != 11 || parseHostRegionAdapter.storeHostRegionDispatchSourceVersion != 13 || !parseHostRegionAdapter.hasHostRegionDispatchVersionVector || !parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple || len(parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple) != 2 || parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple[0] != 5 || parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple[1] != 8 {
		parseT.Fatalf("dispatch version vector state = %+v, want copied tuple and version metadata", parseHostRegionAdapter)
	}

	parseHostRegionAdapter.storeHostRegionDispatchVersionVector(
		RendererID("dashboard.fallback-panel"),
		SnapshotEnvelope{Epoch: 17, InputVersion: 19, SourceVersion: 23},
		nil,
		false,
	)
	if parseHostRegionAdapter.storeHostRegionDispatchRendererID != RendererID("dashboard.fallback-panel") || parseHostRegionAdapter.storeHostRegionDispatchEpoch != 17 || parseHostRegionAdapter.storeHostRegionDispatchInputVersion != 19 || parseHostRegionAdapter.storeHostRegionDispatchSourceVersion != 23 || !parseHostRegionAdapter.hasHostRegionDispatchVersionVector || parseHostRegionAdapter.hasHostRegionDispatchSourceVersionTuple || len(parseHostRegionAdapter.storeHostRegionDispatchSourceVersionTuple) != 0 {
		parseT.Fatalf("dispatch version vector clear path = %+v, want cleared tuple with refreshed version metadata", parseHostRegionAdapter)
	}
}
