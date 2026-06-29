package runtime2

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// buildHostRegionDispatchHashLegacyState stores legacy dispatch-hash benchmark state.
type buildHostRegionDispatchHashLegacyState struct {
	storeHostRegionDispatchHash    [sha256.Size]byte
	storeHostRegionDispatchScratch []byte
	hasHostRegionDispatchHash      bool
}

// buildHostRegionDispatchHashLegacy computes one legacy dispatch hash path without fast-prefilter short-circuiting.
func buildHostRegionDispatchHashLegacy(
	parseState *buildHostRegionDispatchHashLegacyState,
	parseSnapshotEnvelope SnapshotEnvelope,
	parseSourceIDs []string,
) (bool, error) {
	if parseState == nil {
		return false, fmt.Errorf("runtime2: dispatch hash legacy state is required")
	}
	getDispatchHash, getDispatchScratch, parseDispatchHashErr := buildSnapshotDispatchHashIntoWithSourceIDs(
		parseSnapshotEnvelope,
		parseSourceIDs,
		parseState.storeHostRegionDispatchScratch,
	)
	if parseDispatchHashErr != nil {
		return false, parseDispatchHashErr
	}
	parseState.storeHostRegionDispatchScratch = getDispatchScratch
	hasDispatchNoChange := parseState.hasHostRegionDispatchHash &&
		getDispatchHash == parseState.storeHostRegionDispatchHash
	parseState.storeHostRegionDispatchHash = getDispatchHash
	parseState.hasHostRegionDispatchHash = true
	return hasDispatchNoChange, nil
}

// buildHostRegionDispatchHashBenchEnvelope builds one benchmark envelope.
func buildHostRegionDispatchHashBenchEnvelope(parseTick int) SnapshotEnvelope {
	getSourceVersion := uint64(parseTick + 1)
	return SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-dispatch-hash-bench"),
		Epoch:            1,
		InputVersion:     uint64(parseTick + 1),
		SourceVersion:    getSourceVersion,
		Props: map[string]any{
			"title": "Orders",
			"tick":  parseTick,
			"meta": map[string]any{
				"label": "dashboard",
				"mode":  "hot",
			},
		},
		Sources: map[string]any{
			"alpha": parseTick % 3,
			"count": parseTick,
			"flags": map[string]any{
				"active": true,
				"tier":   "a",
			},
			"stats": map[string]any{
				"closed":  9,
				"pending": 4,
			},
		},
	}
}

// buildHostRegionDispatchHashBenchSourceIDs builds one canonical declared-source order for dispatch-hash benchmarks.
func buildHostRegionDispatchHashBenchSourceIDs() []string {
	return []string{"alpha", "count", "flags", "stats"}
}

// buildHostRegionDispatchHashBenchChangedEnvelopeList builds one deterministic changed-envelope sequence.
func buildHostRegionDispatchHashBenchChangedEnvelopeList() []SnapshotEnvelope {
	buildEnvelopeList := make([]SnapshotEnvelope, 0, 32)
	for parseTick := range 32 {
		buildEnvelopeList = append(buildEnvelopeList, buildHostRegionDispatchHashBenchEnvelope(parseTick))
	}
	return buildEnvelopeList
}

// buildHostRegionDispatchHashBenchAdapter builds one adapter configured for dispatch-hash benchmarking.
func buildHostRegionDispatchHashBenchAdapter() *HostRegionAdapter {
	return &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-dispatch-hash-bench"),
	}
}

// benchmarkHandleHostRegionDispatchHashCurrentVsLegacy runs the shared current-vs-legacy host dispatch-hash benchmark flow.
func benchmarkHandleHostRegionDispatchHashCurrentVsLegacy(parseB *testing.B, parseSourceIDs []string) {
	getChangedEnvelopeList := buildHostRegionDispatchHashBenchChangedEnvelopeList()
	getStableEnvelope := getChangedEnvelopeList[0]

	parseB.Run("changed_payloads_legacy_sha_only", func(parseB *testing.B) {
		buildLegacyState := &buildHostRegionDispatchHashLegacyState{}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getEnvelope := getChangedEnvelopeList[parseIndex%len(getChangedEnvelopeList)]
			if _, parseErr := buildHostRegionDispatchHashLegacy(buildLegacyState, getEnvelope, parseSourceIDs); parseErr != nil {
				parseB.Fatalf("buildHostRegionDispatchHashLegacy returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("changed_payloads_current_version_vector_gate", func(parseB *testing.B) {
		buildAdapter := buildHostRegionDispatchHashBenchAdapter()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getEnvelope := getChangedEnvelopeList[parseIndex%len(getChangedEnvelopeList)]
			if _, parseErr := buildAdapter.handleHostRegionDispatchHash(
				getEnvelope,
				parseSourceIDs,
				RendererID("dashboard.hot-panel"),
			); parseErr != nil {
				parseB.Fatalf("handleHostRegionDispatchHash returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("stable_payload_legacy_sha_only", func(parseB *testing.B) {
		buildLegacyState := &buildHostRegionDispatchHashLegacyState{}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildHostRegionDispatchHashLegacy(buildLegacyState, getStableEnvelope, parseSourceIDs); parseErr != nil {
				parseB.Fatalf("buildHostRegionDispatchHashLegacy returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("stable_payload_current_digest_guard", func(parseB *testing.B) {
		buildAdapter := buildHostRegionDispatchHashBenchAdapter()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildAdapter.handleHostRegionDispatchHash(
				getStableEnvelope,
				parseSourceIDs,
				RendererID("dashboard.hot-panel"),
			); parseErr != nil {
				parseB.Fatalf("handleHostRegionDispatchHash returned error: %v", parseErr)
			}
		}
	})
}

// BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy compares current version-vector-gated dispatch hashing against legacy SHA-only hashing.
func BenchmarkHandleHostRegionDispatchHashCurrentVsLegacy(parseB *testing.B) {
	benchmarkHandleHostRegionDispatchHashCurrentVsLegacy(parseB, buildHostRegionDispatchHashBenchSourceIDs())
}
