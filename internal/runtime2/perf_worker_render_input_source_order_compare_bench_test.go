package runtime2

import "testing"

// buildWorkerRenderInputSourceOrderBenchSnapshot builds one stable snapshot with declared sources for source-order reuse benchmarks.
func buildWorkerRenderInputSourceOrderBenchSnapshot(parseB *testing.B) SnapshotEnvelope {
	parseB.Helper()
	getSnapshotEnvelope, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-worker-render-input",
		1,
		1,
		map[string]any{
			"title": "Orders",
			"tick":  1,
		},
		[]string{"alpha", "beta", "count", "flags", "gamma", "labels", "stats", "status"},
		map[string]any{
			"alpha": 1,
			"beta":  2,
			"count": 9,
			"flags": map[string]any{"live": true, "tier": "a"},
			"gamma": 3,
			"labels": map[string]any{
				"owner": "ops",
				"zone":  "east",
			},
			"stats":  map[string]any{"closed": 4, "open": 6},
			"status": "healthy",
		},
		map[string]uint64{
			"alpha":  9,
			"beta":   9,
			"count":  9,
			"flags":  9,
			"gamma":  9,
			"labels": 9,
			"stats":  9,
			"status": 9,
		},
	)
	if parseSnapshotErr != nil {
		parseB.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	return getSnapshotEnvelope
}

// buildWorkerRenderInputWithSourceOrderLegacyBenchmark preserves the previous worker render-input source-order path.
func buildWorkerRenderInputWithSourceOrderLegacyBenchmark(
	parseSnapshot SnapshotEnvelope,
	parseCachedSourceIDs []string,
) (WorkerRenderInput, []string, error) {
	if parseSnapshot.RegionInstanceID == "" {
		return WorkerRenderInput{}, nil, nil
	}
	if parseSnapshotErr := ValidateSnapshotEnvelope(parseSnapshot); parseSnapshotErr != nil {
		return WorkerRenderInput{}, nil, parseSnapshotErr
	}
	hasSourceOrderMatch := hasWorkerRenderSnapshotSourceOrderMatch(parseSnapshot.Sources, parseCachedSourceIDs)
	parseSourceIDs := parseCachedSourceIDs
	if !hasSourceOrderMatch {
		parseSourceIDs = make([]string, 0, len(parseSnapshot.Sources))
		for parseSourceID := range parseSnapshot.Sources {
			parseSourceIDs = append(parseSourceIDs, parseSourceID)
		}
		getNormalizedSourceIDs, parseSourceIDsErr := NormalizeSourceIDs(parseSourceIDs)
		if parseSourceIDsErr != nil {
			return WorkerRenderInput{}, nil, parseSourceIDsErr
		}
		parseSourceIDs = getNormalizedSourceIDs
	}
	if len(parseSourceIDs) == 0 {
		return WorkerRenderInput{
			GetProps:         parseSnapshot.Props,
			GetSourceVersion: parseSnapshot.SourceVersion,
		}, parseSourceIDs, nil
	}
	buildSourceEntries := make([]WorkerRenderSourceEntry, len(parseSourceIDs))
	for parseSourceIndex, parseSourceID := range parseSourceIDs {
		buildSourceEntries[parseSourceIndex] = WorkerRenderSourceEntry{
			GetSourceID:    parseSourceID,
			GetSourceValue: parseSnapshot.Sources[parseSourceID],
		}
	}
	buildCachedSourceIDs := parseSourceIDs
	if !hasSourceOrderMatch {
		buildCachedSourceIDs = append([]string(nil), parseSourceIDs...)
	}
	return WorkerRenderInput{
		GetProps:         parseSnapshot.Props,
		GetSourceVersion: parseSnapshot.SourceVersion,
		GetSourceEntries: buildSourceEntries,
	}, buildCachedSourceIDs, nil
}

// BenchmarkBuildWorkerRenderInputSourceOrderCurrentVsLegacy compares cached source-order render-input adaptation against per-call source-key normalization.
func BenchmarkBuildWorkerRenderInputSourceOrderCurrentVsLegacy(parseB *testing.B) {
	getSnapshotEnvelope := buildWorkerRenderInputSourceOrderBenchSnapshot(parseB)
	parseB.Run("legacy_rebuild_order_each_call", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseErr := buildWorkerRenderInputWithSourceOrderLegacyBenchmark(getSnapshotEnvelope, nil); parseErr != nil {
				parseB.Fatalf("buildWorkerRenderInputWithSourceOrderLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("current_rebuild_order_each_call", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := BuildWorkerRenderInput(getSnapshotEnvelope); parseErr != nil {
				parseB.Fatalf("BuildWorkerRenderInput returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("legacy_reuse_cached_order", func(parseB *testing.B) {
		var getCachedSourceIDs []string
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			_, getSourceIDs, parseErr := buildWorkerRenderInputWithSourceOrderLegacyBenchmark(getSnapshotEnvelope, getCachedSourceIDs)
			if parseErr != nil {
				parseB.Fatalf("buildWorkerRenderInputWithSourceOrderLegacyBenchmark returned error: %v", parseErr)
			}
			getCachedSourceIDs = getSourceIDs
		}
	})
	parseB.Run("current_reuse_cached_order", func(parseB *testing.B) {
		var getCachedSourceIDs []string
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			_, getSourceIDs, parseErr := buildWorkerRenderInputWithSourceOrder(getSnapshotEnvelope, getCachedSourceIDs)
			if parseErr != nil {
				parseB.Fatalf("buildWorkerRenderInputWithSourceOrder returned error: %v", parseErr)
			}
			getCachedSourceIDs = getSourceIDs
		}
	})
}
