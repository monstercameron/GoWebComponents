package runtime2

import "testing"

// BenchmarkBuildSnapshotDispatchFastHashPropKeysCurrentVsLegacy compares cached ordered prop-key hashing against the generic per-call props map path.
func BenchmarkBuildSnapshotDispatchFastHashPropKeysCurrentVsLegacy(parseB *testing.B) {
	getEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-prop-keys-bench"),
		Epoch:            9,
		InputVersion:     21,
		SourceVersion:    34,
		Props: map[string]any{
			"active":     true,
			"count":      42,
			"label":      "orders",
			"owner":      "ops",
			"priority":   "high",
			"region":     "west",
			"route":      "/dashboard/orders",
			"selection":  "all",
			"threshold":  0.85,
			"visibility": "shown",
		},
		Sources: map[string]any{
			"filters": map[string]any{
				"status": "open",
				"owner":  "ops",
			},
			"stats": map[string]any{
				"closed":  9,
				"pending": 4,
			},
		},
	}
	getPropsOrderedKeys := buildHostRegionDispatchPropsOrderedKeys(nil, getEnvelope.Props.(map[string]any))
	getSourceIDs := []string{"filters", "stats"}

	parseB.Run("legacy_sort_props_each_call", func(parseB *testing.B) {
		var getScratch []byte
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			var parseErr error
			_, getScratch, parseErr = buildSnapshotDispatchFastHashIntoWithSourceIDs(
				getEnvelope,
				getSourceIDs,
				getScratch,
			)
			if parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchFastHashIntoWithSourceIDs returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("current_cached_props_order", func(parseB *testing.B) {
		var getScratch []byte
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			var parseErr error
			_, getScratch, parseErr = buildSnapshotDispatchFastHashIntoWithSourceAndPropsKeys(
				getEnvelope,
				getSourceIDs,
				getPropsOrderedKeys,
				getScratch,
			)
			if parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchFastHashIntoWithSourceAndPropsKeys returned error: %v", parseErr)
			}
		}
	})
}
