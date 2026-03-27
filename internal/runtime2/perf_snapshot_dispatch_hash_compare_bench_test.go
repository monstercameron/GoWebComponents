package runtime2

import "testing"

// buildPerfSnapshotDispatchHashEnvelope builds one representative nested snapshot envelope for dispatch-hash benchmarks.
func buildPerfSnapshotDispatchHashEnvelope() SnapshotEnvelope {
	return SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-bench"),
		Epoch:            3,
		InputVersion:     11,
		SourceVersion:    7,
		Props: map[string]any{
			"title":   "Orders",
			"tick":    42,
			"active":  true,
			"filters": map[string]any{"status": "open", "owner": "ops"},
			"rows": []any{
				map[string]any{"id": "a", "count": 1},
				map[string]any{"id": "b", "count": 2},
				map[string]any{"id": "c", "count": 3},
			},
		},
		Sources: map[string]any{
			"filters": map[string]any{
				"status": "open",
				"owner":  "ops",
			},
			"stats": map[string]any{
				"pending": 4,
				"closed":  9,
			},
		},
	}
}

// BenchmarkBuildSnapshotDispatchHashCurrentVsLegacy compares the streamed dispatch hash against the previous buffered payload path.
func BenchmarkBuildSnapshotDispatchHashCurrentVsLegacy(parseB *testing.B) {
	parseEnvelope := buildPerfSnapshotDispatchHashEnvelope()
	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildSnapshotDispatchHash(parseEnvelope); parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchHash returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseErr := buildSnapshotDispatchHashInto(parseEnvelope, nil); parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchHashInto returned error: %v", parseErr)
			}
		}
	})
}
