package runtime2_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

var storeRuntime2SnapshotHashBenchmarkSink [sha256.Size]byte

// buildRuntime2SnapshotHashCompareRenderOutput builds one keyed render-like payload for snapshot-hash benchmark coverage.
func buildRuntime2SnapshotHashCompareRenderOutput(parseItemCount int, parseRotation int, parseClass string) map[string]any {
	if parseItemCount <= 0 {
		parseItemCount = 1
	}
	buildChildren := make([]any, 0, parseItemCount)
	for parseIndex := 0; parseIndex < parseItemCount; parseIndex++ {
		getItemIndex := (parseIndex + parseRotation) % parseItemCount
		buildChildren = append(buildChildren, map[string]any{
			"kind": "host-element",
			"tag":  "li",
			"key":  fmt.Sprintf("item-%d", getItemIndex),
			"children": []any{
				map[string]any{
					"kind": "text",
					"text": fmt.Sprintf("value-%d", getItemIndex),
				},
			},
		})
	}
	return map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": parseClass,
		},
		"children": []any{
			map[string]any{
				"kind":     "host-element",
				"tag":      "ul",
				"children": buildChildren,
			},
		},
	}
}

// buildRuntime2LegacySnapshotFingerprintHashForBenchmark computes snapshot identity using the legacy marshal-based path.
func buildRuntime2LegacySnapshotFingerprintHashForBenchmark(parseEnvelope runtime2.SnapshotEnvelope) ([sha256.Size]byte, error) {
	if parseErr := runtime2.ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return [sha256.Size]byte{}, parseErr
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return [sha256.Size]byte{}, fmt.Errorf("runtime2: encode snapshot fingerprint payload: %w", parseErr)
	}
	return sha256.Sum256(parsePayload), nil
}

// BenchmarkGetSnapshotFingerprintHashCurrentVsLegacy compares the optimized snapshot hash path against the legacy marshal-based implementation.
func BenchmarkGetSnapshotFingerprintHashCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("small/current", func(parseB *testing.B) {
		getSnapshotEnvelope := runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-hash-small"),
			Epoch:            1,
			InputVersion:     1,
			Props: map[string]any{
				"title": "Orders",
			},
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getHash, parseHashErr := runtime2.GetSnapshotFingerprintHash(getSnapshotEnvelope)
			if parseHashErr != nil {
				parseB.Fatalf("GetSnapshotFingerprintHash returned error: %v", parseHashErr)
			}
			storeRuntime2SnapshotHashBenchmarkSink = getHash
		}
	})
	parseB.Run("small/legacy", func(parseB *testing.B) {
		getSnapshotEnvelope := runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-hash-small"),
			Epoch:            1,
			InputVersion:     1,
			Props: map[string]any{
				"title": "Orders",
			},
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getHash, parseHashErr := buildRuntime2LegacySnapshotFingerprintHashForBenchmark(getSnapshotEnvelope)
			if parseHashErr != nil {
				parseB.Fatalf("buildRuntime2LegacySnapshotFingerprintHashForBenchmark returned error: %v", parseHashErr)
			}
			storeRuntime2SnapshotHashBenchmarkSink = getHash
		}
	})
	parseB.Run("large/current", func(parseB *testing.B) {
		getSnapshotEnvelope := runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-hash-large"),
			Epoch:            3,
			InputVersion:     7,
			SourceVersion:    5,
			Props:            buildRuntime2SnapshotHashCompareRenderOutput(64, 1, "active"),
			Sources: map[string]any{
				"filters": map[string]any{
					"status": "open",
					"limit":  50,
				},
				"selection": []any{"north", "south", "west"},
			},
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getHash, parseHashErr := runtime2.GetSnapshotFingerprintHash(getSnapshotEnvelope)
			if parseHashErr != nil {
				parseB.Fatalf("GetSnapshotFingerprintHash returned error: %v", parseHashErr)
			}
			storeRuntime2SnapshotHashBenchmarkSink = getHash
		}
	})
	parseB.Run("large/legacy", func(parseB *testing.B) {
		getSnapshotEnvelope := runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-hash-large"),
			Epoch:            3,
			InputVersion:     7,
			SourceVersion:    5,
			Props:            buildRuntime2SnapshotHashCompareRenderOutput(64, 1, "active"),
			Sources: map[string]any{
				"filters": map[string]any{
					"status": "open",
					"limit":  50,
				},
				"selection": []any{"north", "south", "west"},
			},
		}
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getHash, parseHashErr := buildRuntime2LegacySnapshotFingerprintHashForBenchmark(getSnapshotEnvelope)
			if parseHashErr != nil {
				parseB.Fatalf("buildRuntime2LegacySnapshotFingerprintHashForBenchmark returned error: %v", parseHashErr)
			}
			storeRuntime2SnapshotHashBenchmarkSink = getHash
		}
	})
}
