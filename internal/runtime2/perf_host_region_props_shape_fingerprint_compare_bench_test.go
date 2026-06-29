package runtime2

import "testing"

// buildHostRegionPropsShapeFingerprintCompareSpec builds one flat scalar-heavy props spec used by shape-fingerprint compare benchmarks.
func buildHostRegionPropsShapeFingerprintCompareSpec() ParallelRegionSpec {
	return ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-shape-bench"),
		Props: map[string]any{
			"title":       "Orders",
			"subtitle":    "North America",
			"count":       12,
			"retryCount":  1,
			"isLoading":   false,
			"isVisible":   true,
			"route":       "/orders",
			"tenant":      "default",
			"threshold":   0.85,
			"status":      "healthy",
			"statusCode":  200,
			"batchNumber": 9,
		},
		SourceIDs: []string{"count", "status"},
	}
}

// buildHostRegionPropsShapeFingerprintCompareAdapter builds one mounted host adapter used by props shape-fingerprint compare benchmarks.
func buildHostRegionPropsShapeFingerprintCompareAdapter(parseB *testing.B) *HostRegionAdapter {
	parseB.Helper()
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(
		RegionInstanceID("region-shape-bench"),
		[]SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-shape-bench"),
	}, 3)
	if parseMountErr != nil {
		parseB.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseLookupErr := parseHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count":  5,
					"status": "healthy",
				},
				map[string]uint64{
					"count":  9,
					"status": 9,
				},
				nil
		},
	)
	if parseLookupErr != nil {
		parseB.Fatalf("SetHostRegionSourceLookup returned error: %v", parseLookupErr)
	}
	return parseHostRegionAdapter
}

// BenchmarkHandleHostRegionUpdateSnapshotPropsShapeFingerprintCurrentVsLegacy compares current flat-props shape fingerprint caching with the legacy always-validate path.
func BenchmarkHandleHostRegionUpdateSnapshotPropsShapeFingerprintCurrentVsLegacy(parseB *testing.B) {
	parseBuildSpec := func() ParallelRegionSpec {
		return buildHostRegionPropsShapeFingerprintCompareSpec()
	}
	parseB.Run("current-stable-flat-props-wide", func(parseB *testing.B) {
		parseHostRegionAdapter := buildHostRegionPropsShapeFingerprintCompareAdapter(parseB)
		parseSpec := parseBuildSpec()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, _, _, _, _, parseErr := parseHostRegionAdapter.handleHostRegionUpdateSnapshot(
				parseSpec,
				uint64(parseIndex+1),
				true,
			); parseErr != nil {
				parseB.Fatalf("handleHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("legacy-stable-flat-props-wide", func(parseB *testing.B) {
		parseHostRegionAdapter := buildHostRegionPropsShapeFingerprintCompareAdapter(parseB)
		parseSpec := parseBuildSpec()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, _, parseErr := buildRuntime2LegacyHostRegionUpdateSnapshot(
				parseHostRegionAdapter,
				parseSpec,
				uint64(parseIndex+1),
				true,
			); parseErr != nil {
				parseB.Fatalf("buildRuntime2LegacyHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("current-changed-flat-props-wide", func(parseB *testing.B) {
		parseHostRegionAdapter := buildHostRegionPropsShapeFingerprintCompareAdapter(parseB)
		parseSpec := parseBuildSpec()
		parseProps := parseSpec.Props.(map[string]any)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseProps["count"] = parseIndex
			parseProps["retryCount"] = parseIndex % 4
			parseProps["batchNumber"] = parseIndex % 13
			if _, _, _, _, _, _, parseErr := parseHostRegionAdapter.handleHostRegionUpdateSnapshot(
				parseSpec,
				uint64(parseIndex+1),
				true,
			); parseErr != nil {
				parseB.Fatalf("handleHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("legacy-changed-flat-props-wide", func(parseB *testing.B) {
		parseHostRegionAdapter := buildHostRegionPropsShapeFingerprintCompareAdapter(parseB)
		parseSpec := parseBuildSpec()
		parseProps := parseSpec.Props.(map[string]any)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseProps["count"] = parseIndex
			parseProps["retryCount"] = parseIndex % 4
			parseProps["batchNumber"] = parseIndex % 13
			if _, _, _, parseErr := buildRuntime2LegacyHostRegionUpdateSnapshot(
				parseHostRegionAdapter,
				parseSpec,
				uint64(parseIndex+1),
				true,
			); parseErr != nil {
				parseB.Fatalf("buildRuntime2LegacyHostRegionUpdateSnapshot returned error: %v", parseErr)
			}
		}
	})
}
