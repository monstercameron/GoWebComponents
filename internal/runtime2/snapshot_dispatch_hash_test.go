package runtime2

import (
	"crypto/sha256"
	"math"
	"testing"
)

// TestBuildSnapshotDispatchHashIgnoresInputVersion verifies dispatch hashing ignores input-version churn for no-change scheduling.
func TestBuildSnapshotDispatchHashIgnoresInputVersion(parseT *testing.T) {
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            2,
		InputVersion:     3,
		SourceVersion:    11,
		Props: map[string]any{
			"title": "Orders",
			"tick":  7,
		},
		Sources: map[string]any{
			"filters": map[string]any{
				"status": "open",
			},
		},
	}
	getHashFirst, parseFirstErr := buildSnapshotDispatchHash(getSnapshotEnvelope)
	if parseFirstErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(first) returned error: %v", parseFirstErr)
	}
	getSnapshotEnvelope.InputVersion = 99
	getHashSecond, parseSecondErr := buildSnapshotDispatchHash(getSnapshotEnvelope)
	if parseSecondErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(second) returned error: %v", parseSecondErr)
	}
	if getHashFirst != getHashSecond {
		parseT.Fatal("expected dispatch hash to ignore input-version changes")
	}
}

// TestBuildSnapshotDispatchHashCanonicalMapOrder verifies dispatch hashing is stable across map iteration order differences.
func TestBuildSnapshotDispatchHashCanonicalMapOrder(parseT *testing.T) {
	getSnapshotEnvelopeLeft := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-2"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"a": 1,
			"b": 2,
			"c": 3,
		},
		Sources: map[string]any{
			"zeta": 4,
			"beta": 5,
		},
	}
	getSnapshotEnvelopeRight := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-2"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"c": 3,
			"a": 1,
			"b": 2,
		},
		Sources: map[string]any{
			"beta": 5,
			"zeta": 4,
		},
	}
	getHashLeft, parseLeftErr := buildSnapshotDispatchHash(getSnapshotEnvelopeLeft)
	if parseLeftErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(left) returned error: %v", parseLeftErr)
	}
	getHashRight, parseRightErr := buildSnapshotDispatchHash(getSnapshotEnvelopeRight)
	if parseRightErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(right) returned error: %v", parseRightErr)
	}
	if getHashLeft != getHashRight {
		parseT.Fatal("expected dispatch hash to be stable for equivalent map contents")
	}
}

// TestBuildSnapshotDispatchHashRejectsNonFiniteNumber verifies dispatch hashing rejects NaN and infinities.
func TestBuildSnapshotDispatchHashRejectsNonFiniteNumber(parseT *testing.T) {
	_, parseErr := buildSnapshotDispatchHash(SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-3"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"value": math.NaN(),
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected non-finite dispatch hash input to fail")
	}
}

// TestBuildSnapshotDispatchHashCanonicalPairWithEmptyKey verifies two-key map fast path stability when one key is an empty string.
func TestBuildSnapshotDispatchHashCanonicalPairWithEmptyKey(parseT *testing.T) {
	getSnapshotEnvelopeLeft := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-pair-empty-key"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"":  1,
			"k": 2,
		},
		Sources: map[string]any{
			"":  "a",
			"s": "b",
		},
	}
	getSnapshotEnvelopeRight := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-pair-empty-key"),
		Epoch:            1,
		InputVersion:     7,
		Props: map[string]any{
			"k": 2,
			"":  1,
		},
		Sources: map[string]any{
			"s": "b",
			"":  "a",
		},
	}
	getHashLeft, parseLeftErr := buildSnapshotDispatchHash(getSnapshotEnvelopeLeft)
	if parseLeftErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(left) returned error: %v", parseLeftErr)
	}
	getHashRight, parseRightErr := buildSnapshotDispatchHash(getSnapshotEnvelopeRight)
	if parseRightErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(right) returned error: %v", parseRightErr)
	}
	if getHashLeft != getHashRight {
		parseT.Fatal("expected dispatch hash to remain stable for two-key maps including an empty key")
	}
}

// TestBuildSnapshotDispatchHashStreamedMatchesBuffered verifies the streaming dispatch-hash path matches the buffered hash output.
func TestBuildSnapshotDispatchHashStreamedMatchesBuffered(parseT *testing.T) {
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-streamed"),
		Epoch:            3,
		InputVersion:     99,
		SourceVersion:    5,
		Props: map[string]any{
			"title": "Orders",
			"tick":  7,
		},
		Sources: map[string]any{
			"filters": map[string]any{
				"status": "open",
			},
		},
	}
	getBufferedHash, parseBufferedErr := buildSnapshotDispatchHash(getSnapshotEnvelope)
	if parseBufferedErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(buffered) returned error: %v", parseBufferedErr)
	}
	getStreamedHash, parseStreamedErr := buildSnapshotDispatchHashStreamed(getSnapshotEnvelope)
	if parseStreamedErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHashStreamed returned error: %v", parseStreamedErr)
	}
	if getBufferedHash != getStreamedHash {
		parseT.Fatal("expected streamed and buffered dispatch hashes to match")
	}
}

// TestBuildSnapshotDispatchHashIntoWithSourceIDsMatchesGeneric verifies ordered source IDs preserve the generic canonical dispatch hash bytes.
func TestBuildSnapshotDispatchHashIntoWithSourceIDsMatchesGeneric(parseT *testing.T) {
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-source-order"),
		Epoch:            5,
		InputVersion:     8,
		SourceVersion:    13,
		Props: map[string]any{
			"title": "Orders",
		},
		Sources: map[string]any{
			"stats": map[string]any{
				"pending": 4,
			},
			"filters": map[string]any{
				"status": "open",
			},
			"count": 9,
		},
	}
	getGenericHash, _, parseGenericErr := buildSnapshotDispatchHashInto(getSnapshotEnvelope, nil)
	if parseGenericErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHashInto(generic) returned error: %v", parseGenericErr)
	}
	getOrderedHash, _, parseOrderedErr := buildSnapshotDispatchHashIntoWithSourceIDs(
		getSnapshotEnvelope,
		[]string{"count", "filters", "stats"},
		nil,
	)
	if parseOrderedErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHashIntoWithSourceIDs returned error: %v", parseOrderedErr)
	}
	if getGenericHash != getOrderedHash {
		parseT.Fatal("expected ordered source IDs to preserve the canonical dispatch hash output")
	}
}

// TestHandleHostRegionDispatchHashStoresFastHashState verifies host dispatch hashing stores fast-hash and version-vector guard state.
func TestHandleHostRegionDispatchHashStoresFastHashState(parseT *testing.T) {
	getHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-scratch"),
	}
	_, parseHashErr := getHostRegionAdapter.handleHostRegionDispatchHash(SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-scratch"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"title": "Orders",
		},
	}, nil, RendererID("dashboard.hot-panel"))
	if parseHashErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash returned error: %v", parseHashErr)
	}
	if !getHostRegionAdapter.hasHostRegionDispatchFastHash {
		parseT.Fatal("expected dispatch fast-hash state to be stored")
	}
	if !getHostRegionAdapter.hasHostRegionDispatchVersionVector {
		parseT.Fatal("expected dispatch version-vector state to be stored")
	}
	if getHostRegionAdapter.hasHostRegionDispatchHash {
		parseT.Fatal("expected dispatch digest state to stay cleared on fast-hash path")
	}
	if getHostRegionAdapter.storeHostRegionDispatchRendererID != RendererID("dashboard.hot-panel") {
		parseT.Fatalf(
			"expected dispatch renderer vector to store dashboard.hot-panel, got %q",
			getHostRegionAdapter.storeHostRegionDispatchRendererID,
		)
	}
	if getHostRegionAdapter.storeHostRegionDispatchInputVersion != 1 {
		parseT.Fatalf(
			"expected dispatch input-version vector to store 1, got %d",
			getHostRegionAdapter.storeHostRegionDispatchInputVersion,
		)
	}
}

// TestHandleHostRegionDispatchHashIgnoresInputVersion verifies host dispatch hashing does not require envelope copies to ignore input-version churn.
func TestHandleHostRegionDispatchHashIgnoresInputVersion(parseT *testing.T) {
	getHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-ignore-input"),
	}
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-ignore-input"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"title": "Orders",
		},
		Sources: map[string]any{
			"filters": map[string]any{
				"status": "open",
			},
		},
	}
	getFirstNoChange, parseFirstErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"filters"},
		RendererID("dashboard.hot-panel"),
	)
	if parseFirstErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(first) returned error: %v", parseFirstErr)
	}
	if getFirstNoChange {
		parseT.Fatal("expected first dispatch hash to be treated as changed")
	}
	getSnapshotEnvelope.InputVersion = 99
	getSecondNoChange, parseSecondErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"filters"},
		RendererID("dashboard.hot-panel"),
	)
	if parseSecondErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(second) returned error: %v", parseSecondErr)
	}
	if !getSecondNoChange {
		parseT.Fatal("expected input-version churn to be ignored by host dispatch hash")
	}
}

// TestHandleHostRegionDispatchHashVersionVectorExactMatchUsesFastHashGuard verifies exact version-vector matches still pass through fast-hash no-change guards.
func TestHandleHostRegionDispatchHashVersionVectorExactMatchUsesFastHashGuard(parseT *testing.T) {
	getHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-version-match"),
		storeHostRegionSourceSnapshotCacheSourceVersions: map[string]uint64{
			"alpha": 9,
			"beta":  9,
		},
	}
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-version-match"),
		Epoch:            2,
		InputVersion:     7,
		SourceVersion:    9,
		Props: map[string]any{
			"title": "Orders",
		},
		Sources: map[string]any{
			"alpha": 1,
			"beta":  2,
		},
	}
	getFirstNoChange, parseFirstErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"alpha", "beta"},
		RendererID("dashboard.hot-panel"),
	)
	if parseFirstErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(first) returned error: %v", parseFirstErr)
	}
	if getFirstNoChange {
		parseT.Fatal("expected first dispatch hash to be treated as changed")
	}
	getHostRegionAdapter.storeHostRegionDispatchHash = [sha256.Size]byte{}
	getHostRegionAdapter.hasHostRegionDispatchHash = false
	getSecondNoChange, parseSecondErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"alpha", "beta"},
		RendererID("dashboard.hot-panel"),
	)
	if parseSecondErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(second) returned error: %v", parseSecondErr)
	}
	if !getSecondNoChange {
		parseT.Fatal("expected exact version-vector match to short-circuit as no-change")
	}
	if getHostRegionAdapter.hasHostRegionDispatchHash {
		parseT.Fatal("expected digest state to remain untouched by exact version-vector short-circuit")
	}
}

// TestHandleHostRegionDispatchHashVersionVectorExactMatchPayloadChangeSchedules verifies payload changes with unchanged version vectors do not short-circuit as no-change.
func TestHandleHostRegionDispatchHashVersionVectorExactMatchPayloadChangeSchedules(parseT *testing.T) {
	getHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-version-equal-change"),
		storeHostRegionSourceSnapshotCacheSourceVersions: map[string]uint64{
			"alpha": 13,
		},
	}
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-version-equal-change"),
		Epoch:            2,
		InputVersion:     9,
		SourceVersion:    13,
		Props: map[string]any{
			"title": "Orders",
		},
		Sources: map[string]any{
			"alpha": 1,
		},
	}
	getFirstNoChange, parseFirstErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"alpha"},
		RendererID("dashboard.hot-panel"),
	)
	if parseFirstErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(first) returned error: %v", parseFirstErr)
	}
	if getFirstNoChange {
		parseT.Fatal("expected first dispatch hash to be treated as changed")
	}
	getSnapshotEnvelope.Props = map[string]any{
		"title": "Orders (updated)",
	}
	getSecondNoChange, parseSecondErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"alpha"},
		RendererID("dashboard.hot-panel"),
	)
	if parseSecondErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(second) returned error: %v", parseSecondErr)
	}
	if getSecondNoChange {
		parseT.Fatal("expected payload change with unchanged version-vector fields to be treated as changed")
	}
}

// TestHandleHostRegionDispatchHashVersionVectorMismatchResetsDigestState verifies source-version mismatches short-circuit before digest reuse.
func TestHandleHostRegionDispatchHashVersionVectorMismatchResetsDigestState(parseT *testing.T) {
	getHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-version-vector"),
	}
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-version-vector"),
		Epoch:            1,
		InputVersion:     1,
		SourceVersion:    1,
		Props: map[string]any{
			"title": "Orders",
		},
		Sources: map[string]any{
			"count": 1,
		},
	}
	getFirstNoChange, parseFirstErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"count"},
		RendererID("dashboard.hot-panel"),
	)
	if parseFirstErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(first) returned error: %v", parseFirstErr)
	}
	if getFirstNoChange {
		parseT.Fatal("expected first dispatch hash to be treated as changed")
	}
	getSnapshotEnvelope.SourceVersion = 2
	getSecondNoChange, parseSecondErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"count"},
		RendererID("dashboard.hot-panel"),
	)
	if parseSecondErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(second) returned error: %v", parseSecondErr)
	}
	if getSecondNoChange {
		parseT.Fatal("expected source-version mismatch to short-circuit as changed")
	}
	if !getHostRegionAdapter.hasHostRegionDispatchVersionVector {
		parseT.Fatal("expected dispatch version vector state to be tracked")
	}
	if getHostRegionAdapter.hasHostRegionDispatchHash {
		parseT.Fatal("expected digest state to reset after version-vector mismatch")
	}
}

// TestHandleHostRegionDispatchHashVersionTupleMismatchResetsDigestState verifies ordered source-version tuple mismatches short-circuit before digest reuse.
func TestHandleHostRegionDispatchHashVersionTupleMismatchResetsDigestState(parseT *testing.T) {
	getHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-version-tuple"),
		storeHostRegionSourceSnapshotCacheSourceVersions: map[string]uint64{
			"alpha": 11,
			"beta":  11,
		},
	}
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-version-tuple"),
		Epoch:            1,
		InputVersion:     1,
		SourceVersion:    11,
		Props: map[string]any{
			"title": "Orders",
		},
		Sources: map[string]any{
			"alpha": 1,
			"beta":  2,
		},
	}
	getFirstNoChange, parseFirstErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"alpha", "beta"},
		RendererID("dashboard.hot-panel"),
	)
	if parseFirstErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(first) returned error: %v", parseFirstErr)
	}
	if getFirstNoChange {
		parseT.Fatal("expected first dispatch hash to be treated as changed")
	}
	getHostRegionAdapter.storeHostRegionSourceSnapshotCacheSourceVersions = map[string]uint64{
		"alpha": 11,
		"beta":  12,
	}
	getSecondNoChange, parseSecondErr := getHostRegionAdapter.handleHostRegionDispatchHash(
		getSnapshotEnvelope,
		[]string{"alpha", "beta"},
		RendererID("dashboard.hot-panel"),
	)
	if parseSecondErr != nil {
		parseT.Fatalf("handleHostRegionDispatchHash(second) returned error: %v", parseSecondErr)
	}
	if getSecondNoChange {
		parseT.Fatal("expected source-version tuple mismatch to short-circuit as changed")
	}
	if getHostRegionAdapter.hasHostRegionDispatchHash {
		parseT.Fatal("expected digest state to reset after source-version tuple mismatch")
	}
}
