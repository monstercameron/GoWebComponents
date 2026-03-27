package runtime2

import (
	"sort"
	"testing"
)

// buildSnapshotDispatchAnyMapLegacy appends one canonical map payload using the previous allocating entry-list strategy.
func buildSnapshotDispatchAnyMapLegacy(parseDst []byte, parseValue map[string]any) ([]byte, error) {
	if parseValue == nil {
		return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
	}
	getMapLen := len(parseValue)
	if getMapLen == 0 {
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
		return appendSnapshotDispatchUint64(parseDst, 0), nil
	}
	if getMapLen == 1 {
		return appendSnapshotDispatchAnyMapSingle(parseDst, parseValue)
	}
	if getMapLen == 2 {
		return appendSnapshotDispatchAnyMapPair(parseDst, parseValue)
	}
	parseDst = append(parseDst, getSnapshotDispatchHashMarkerMap)
	type buildSnapshotDispatchAnyMapEntry struct {
		getKey   string
		getValue any
	}
	parseEntries := make([]buildSnapshotDispatchAnyMapEntry, 0, getMapLen)
	for parseKey, parseItem := range parseValue {
		parseEntries = append(parseEntries, buildSnapshotDispatchAnyMapEntry{
			getKey:   parseKey,
			getValue: parseItem,
		})
	}
	sort.Slice(parseEntries, func(parseLeftIndex int, parseRightIndex int) bool {
		return parseEntries[parseLeftIndex].getKey < parseEntries[parseRightIndex].getKey
	})
	parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEntries)))
	for _, parseEntry := range parseEntries {
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(parseEntry.getKey)))
		parseDst = append(parseDst, parseEntry.getKey...)
		var parseErr error
		parseDst, parseErr = appendSnapshotDispatchValue(parseDst, parseEntry.getValue)
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

// buildSnapshotDispatchMapWalkBenchValue builds one nested source map representative of repeated dispatch-hash workloads.
func buildSnapshotDispatchMapWalkBenchValue() map[string]any {
	return map[string]any{
		"alpha": map[string]any{
			"enabled": true,
			"limit":   25,
			"tags":    []any{"open", "critical", "paged"},
		},
		"beta": map[string]any{
			"kind":   "rolling",
			"window": 5,
		},
		"delta": 7,
		"epsilon": map[string]any{
			"nested": map[string]any{
				"count":  42,
				"status": "ready",
			},
		},
		"eta":    "queue-a",
		"gamma":  []any{1, 2, 3, 4},
		"kappa":  map[string]any{"left": 1, "right": 2, "mid": 3},
		"lambda": "ingest",
		"omega":  map[string]any{"retry": 3, "timeout": 1000},
		"sigma":  []any{"x", "y", "z"},
		"theta":  99.5,
		"zeta":   false,
	}
}

// BenchmarkAppendSnapshotDispatchAnyMapCurrentVsLegacy benchmarks pooled map-entry reuse versus allocating entry lists for map-walk hashing.
func BenchmarkAppendSnapshotDispatchAnyMapCurrentVsLegacy(parseB *testing.B) {
	getBenchValue := buildSnapshotDispatchMapWalkBenchValue()
	parseB.Run("legacy_allocating_entries", func(parseB *testing.B) {
		getScratch := make([]byte, 0, 2048)
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getScratch = getScratch[:0]
			if _, parseErr := buildSnapshotDispatchAnyMapLegacy(getScratch, getBenchValue); parseErr != nil {
				parseB.Fatalf("buildSnapshotDispatchAnyMapLegacy returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("current_pooled_entries", func(parseB *testing.B) {
		getScratch := make([]byte, 0, 2048)
		parseB.ReportAllocs()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getScratch = getScratch[:0]
			if _, parseErr := appendSnapshotDispatchAnyMap(getScratch, getBenchValue); parseErr != nil {
				parseB.Fatalf("appendSnapshotDispatchAnyMap returned error: %v", parseErr)
			}
		}
	})
}
