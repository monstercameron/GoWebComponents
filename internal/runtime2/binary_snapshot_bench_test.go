package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// BenchmarkBuildAndParseBinarySnapshotEnvelope benchmarks binary snapshot encode and decode over one representative payload shape.
func BenchmarkBuildAndParseBinarySnapshotEnvelope(parseB *testing.B) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     19,
		SourceVersion:    11,
		Props: map[string]any{
			"title": "Orders",
			"count": 42,
			"meta": map[string]any{
				"priority": "high",
				"visible":  true,
			},
		},
		Sources: map[string]any{
			"status": "healthy",
			"items": []any{
				map[string]any{
					"id":    "a",
					"score": 9,
				},
				map[string]any{
					"id":    "b",
					"score": 7,
				},
			},
		},
	}
	parsePayload, parseErr := runtime2.BuildBinarySnapshotEnvelope(parseEnvelope)
	if parseErr != nil {
		parseB.Fatalf("BuildBinarySnapshotEnvelope warmup returned error: %v", parseErr)
	}

	parseB.Run("build", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.SetBytes(int64(len(parsePayload)))
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := runtime2.BuildBinarySnapshotEnvelope(parseEnvelope); parseErr != nil {
				parseB.Fatalf("BuildBinarySnapshotEnvelope returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("parse", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.SetBytes(int64(len(parsePayload)))
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload); parseErr != nil {
				parseB.Fatalf("ParseBinarySnapshotEnvelope returned error: %v", parseErr)
			}
		}
	})
}

// BenchmarkParseBinarySnapshotEnvelopeSourceHeavy benchmarks parse cost for source-heavy envelopes.
func BenchmarkParseBinarySnapshotEnvelopeSourceHeavy(parseB *testing.B) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-heavy"),
		Epoch:            9,
		InputVersion:     27,
		SourceVersion:    18,
		Props: map[string]any{
			"title": "Heavy Snapshot",
			"mode":  "benchmark",
		},
		Sources: map[string]any{
			"feed": []any{
				map[string]any{"id": "a", "priority": 1, "visible": true},
				map[string]any{"id": "b", "priority": 2, "visible": false},
				map[string]any{"id": "c", "priority": 3, "visible": true},
				map[string]any{"id": "d", "priority": 4, "visible": true},
			},
			"stats": map[string]any{
				"active":  1024,
				"queued":  256,
				"healthy": true,
			},
			"labels": []any{"north", "south", "east", "west"},
			"flags": map[string]any{
				"canary": true,
				"beta":   false,
				"safe":   true,
			},
		},
	}
	parsePayload, parseErr := runtime2.BuildBinarySnapshotEnvelope(parseEnvelope)
	if parseErr != nil {
		parseB.Fatalf("BuildBinarySnapshotEnvelope(seed) returned error: %v", parseErr)
	}
	parseB.ReportAllocs()
	parseB.SetBytes(int64(len(parsePayload)))
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseParseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload); parseParseErr != nil {
			parseB.Fatalf("ParseBinarySnapshotEnvelope returned error: %v", parseParseErr)
		}
	}
}
