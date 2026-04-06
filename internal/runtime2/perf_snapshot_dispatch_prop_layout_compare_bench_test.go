package runtime2

import (
	"sort"
	"testing"
)

// buildSnapshotDispatchPropLayoutBenchmarkMap builds one stable props map for dispatch hash layout comparison.
func buildSnapshotDispatchPropLayoutBenchmarkMap() map[string]any {
	return map[string]any{
		"count":      42,
		"enabled":    true,
		"title":      "runtime2 hot panel",
		"threshold":  0.85,
		"status":     "active",
		"isLoading":  false,
		"retryCount": 3,
		"route":      "/dashboard/hot-panel",
		"region":     "region-1",
		"epoch":      7,
	}
}

// buildSnapshotDispatchPropEntries builds one sorted key/value slice that approximates a []PropKV dispatch-hash layout.
func buildSnapshotDispatchPropEntries(parseProps map[string]any) []buildSnapshotDispatchMapEntry {
	parseEntries := make([]buildSnapshotDispatchMapEntry, 0, len(parseProps))
	for parseKey, parseValue := range parseProps {
		parseEntries = append(parseEntries, buildSnapshotDispatchMapEntry{
			getKey:   parseKey,
			getValue: parseValue,
		})
	}
	sort.Slice(parseEntries, func(parseI int, parseJ int) bool {
		return parseEntries[parseI].getKey < parseEntries[parseJ].getKey
	})
	return parseEntries
}

// appendSnapshotDispatchAnyMapEntries appends one canonical map representation from pre-sorted key/value entries without per-call map iteration.
func appendSnapshotDispatchAnyMapEntries(parseDst []byte, parseEntries []buildSnapshotDispatchMapEntry) ([]byte, error) {
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEntries)))
	for _, getEntry := range parseEntries {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(getEntry.getKey)))
		parseDst = append(parseDst, getEntry.getKey...)
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchValue(parseDst, getEntry.getValue)
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

// BenchmarkSnapshotDispatchPropLayoutCurrentVsEntries compares current map hashing against pre-sorted key/value slice encoding.
func BenchmarkSnapshotDispatchPropLayoutCurrentVsEntries(parseB *testing.B) {
	parseProps := buildSnapshotDispatchPropLayoutBenchmarkMap()
	parseEntries := buildSnapshotDispatchPropEntries(parseProps)

	parseB.Run("current_map_iterate_sort_each_call", func(parseB *testing.B) {
		parseScratch := make([]byte, 0, 512)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePayload, parseErr := appendSnapshotDispatchAnyMap(parseScratch[:0], parseProps)
			if parseErr != nil {
				parseB.Fatalf("appendSnapshotDispatchAnyMap returned error: %v", parseErr)
			}
			parseScratch = parsePayload[:0]
		}
	})

	parseB.Run("candidate_sorted_entries_slice_scan", func(parseB *testing.B) {
		parseScratch := make([]byte, 0, 512)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parsePayload, parseErr := appendSnapshotDispatchAnyMapEntries(parseScratch[:0], parseEntries)
			if parseErr != nil {
				parseB.Fatalf("appendSnapshotDispatchAnyMapEntries returned error: %v", parseErr)
			}
			parseScratch = parsePayload[:0]
		}
	})
}

// BenchmarkBuildSnapshotDispatchFastHashPropLayoutCurrentVsKeys compares ordered-key hashing against one pre-sorted compact props-entry layout on the streamed fast-hash path.
func BenchmarkBuildSnapshotDispatchFastHashPropLayoutCurrentVsKeys(parseB *testing.B) {
	getEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-prop-layout-bench"),
		Epoch:            11,
		InputVersion:     23,
		SourceVersion:    37,
		Props:            buildSnapshotDispatchPropLayoutBenchmarkMap(),
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
	getSourceIDs := []string{"filters", "stats"}
	getPropsOrderedKeys := buildHostRegionDispatchPropsOrderedKeys(nil, getEnvelope.Props.(map[string]any))
	getPropsEntries := buildSnapshotDispatchPropEntries(getEnvelope.Props.(map[string]any))

	parseB.Run("current_cached_props_keys", func(parseB *testing.B) {
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

	parseB.Run("current_compact_prop_entries", func(parseB *testing.B) {
		var getScratch []byte
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			var parseErr error
			_, getScratch, parseErr = buildSnapshotDispatchFastHashIntoWithSourceAndPropsEntries(
				getEnvelope,
				getSourceIDs,
				getPropsEntries,
				getScratch,
			)
			if parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchFastHashIntoWithSourceAndPropsEntries returned error: %v", parseErr)
			}
		}
	})
}
