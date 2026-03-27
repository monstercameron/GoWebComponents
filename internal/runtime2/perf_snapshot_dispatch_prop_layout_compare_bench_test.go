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
