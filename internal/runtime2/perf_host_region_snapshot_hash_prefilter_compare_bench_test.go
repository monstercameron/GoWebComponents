package runtime2

import (
	"crypto/sha256"
	"testing"
)

// hostRegionSnapshotHashLegacyState stores legacy snapshot-hash benchmark state.
type hostRegionSnapshotHashLegacyState struct {
	storeHostRegionSnapshotHash [sha256.Size]byte
	hasHostRegionSnapshotHash   bool
}

// buildHostRegionSnapshotHashLegacy runs the pre-prefilter snapshot hash path.
// It always computes SHA-256 via getSnapshotFingerprintHashWithoutValidation, with no fast content guard.
func buildHostRegionSnapshotHashLegacy(
	parseState *hostRegionSnapshotHashLegacyState,
	parseSnapshotEnvelope SnapshotEnvelope,
) (bool, error) {
	buildFingerprintEnvelope := parseSnapshotEnvelope
	buildFingerprintEnvelope.InputVersion = 1
	getSnapshotHash, parseSnapshotHashErr := getSnapshotFingerprintHashWithoutValidation(buildFingerprintEnvelope)
	if parseSnapshotHashErr != nil {
		return false, parseSnapshotHashErr
	}
	hasNoChange := parseState.hasHostRegionSnapshotHash && getSnapshotHash == parseState.storeHostRegionSnapshotHash
	parseState.storeHostRegionSnapshotHash = getSnapshotHash
	parseState.hasHostRegionSnapshotHash = true
	return hasNoChange, nil
}

// buildHostRegionSnapshotHashPrefilterBenchAdapter builds one adapter pre-warmed for snapshot-hash benchmarks.
func buildHostRegionSnapshotHashPrefilterBenchAdapter(parseB *testing.B, parseRegionID RegionInstanceID, parseEnvelope SnapshotEnvelope) *HostRegionAdapter {
	parseB.Helper()
	buildAdapter := &HostRegionAdapter{
		storeRegionInstanceID: parseRegionID,
	}
	// Warm up so the fast-hash and SHA-256 stored state is populated for the no-change path.
	if _, parseWarmErr := buildAdapter.handleHostRegionSnapshotHash(parseEnvelope); parseWarmErr != nil {
		parseB.Fatalf("handleHostRegionSnapshotHash(warm-up) returned error: %v", parseWarmErr)
	}
	return buildAdapter
}

// buildHostRegionSnapshotHashPrefilterBenchEnvelope builds one small snapshot envelope for fingerprint benchmarks.
func buildHostRegionSnapshotHashPrefilterBenchEnvelope(parseRegionID RegionInstanceID) SnapshotEnvelope {
	return SnapshotEnvelope{
		RegionInstanceID: parseRegionID,
		Epoch:            1,
		InputVersion:     1,
		SourceVersion:    1,
		Props: map[string]any{
			"title": "Orders",
			"mode":  "hot",
		},
		Sources: map[string]any{
			"count": 42,
			"flags": map[string]any{"active": true},
		},
	}
}

// BenchmarkHandleHostRegionSnapshotHashPrefilterCurrentVsLegacy compares the fast-prefiltered snapshot-hash path
// against the legacy path that always runs SHA-256, across no-change and changed sub-cases.
func BenchmarkHandleHostRegionSnapshotHashPrefilterCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("no_change_legacy_sha_always", func(parseB *testing.B) {
		buildRegionID := RegionInstanceID("region-snaphash-nochange-legacy")
		buildEnvelope := buildHostRegionSnapshotHashPrefilterBenchEnvelope(buildRegionID)
		buildState := &hostRegionSnapshotHashLegacyState{}
		// Warm up legacy state.
		if _, parseWarmErr := buildHostRegionSnapshotHashLegacy(buildState, buildEnvelope); parseWarmErr != nil {
			parseB.Fatalf("buildHostRegionSnapshotHashLegacy(warm-up) returned error: %v", parseWarmErr)
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildHostRegionSnapshotHashLegacy(buildState, buildEnvelope); parseErr != nil {
				parseB.Fatalf("buildHostRegionSnapshotHashLegacy returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("no_change_current_fast_prefilter", func(parseB *testing.B) {
		buildRegionID := RegionInstanceID("region-snaphash-nochange-current")
		buildEnvelope := buildHostRegionSnapshotHashPrefilterBenchEnvelope(buildRegionID)
		buildAdapter := buildHostRegionSnapshotHashPrefilterBenchAdapter(parseB, buildRegionID, buildEnvelope)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildAdapter.handleHostRegionSnapshotHash(buildEnvelope); parseErr != nil {
				parseB.Fatalf("handleHostRegionSnapshotHash returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("changed_legacy_sha_always", func(parseB *testing.B) {
		buildRegionID := RegionInstanceID("region-snaphash-changed-legacy")
		buildState := &hostRegionSnapshotHashLegacyState{}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildEnvelope := buildHostRegionSnapshotHashPrefilterBenchEnvelope(buildRegionID)
			buildEnvelope.SourceVersion = uint64(parseIndex + 1)
			buildEnvelope.Sources = map[string]any{
				"count": parseIndex,
				"flags": map[string]any{"active": parseIndex%2 == 0},
			}
			if _, parseErr := buildHostRegionSnapshotHashLegacy(buildState, buildEnvelope); parseErr != nil {
				parseB.Fatalf("buildHostRegionSnapshotHashLegacy returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("changed_current_fast_prefilter", func(parseB *testing.B) {
		buildRegionID := RegionInstanceID("region-snaphash-changed-current")
		buildEnvelope := buildHostRegionSnapshotHashPrefilterBenchEnvelope(buildRegionID)
		buildAdapter := buildHostRegionSnapshotHashPrefilterBenchAdapter(parseB, buildRegionID, buildEnvelope)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			buildEnvelope.SourceVersion = uint64(parseIndex + 1)
			buildEnvelope.Sources = map[string]any{
				"count": parseIndex,
				"flags": map[string]any{"active": parseIndex%2 == 0},
			}
			if _, parseErr := buildAdapter.handleHostRegionSnapshotHash(buildEnvelope); parseErr != nil {
				parseB.Fatalf("handleHostRegionSnapshotHash returned error: %v", parseErr)
			}
		}
	})
}
