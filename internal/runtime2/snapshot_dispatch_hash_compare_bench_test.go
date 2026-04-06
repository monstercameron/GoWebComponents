package runtime2

import (
	"reflect"
	"sort"
	"testing"
)

// appendSnapshotDispatchValueLegacyBenchmark preserves the previous dispatch-value fast-path set from before typed scalar slice and map exits were added.
func appendSnapshotDispatchValueLegacyBenchmark(parseDst []byte, parseValue any) ([]byte, error) {
	switch getValue := parseValue.(type) {
	case nil:
		return append(parseDst, getSnapshotDispatchHashMarkerNil), nil
	case bool:
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerBool)
		if getValue {
			return append(parseDst, 1), nil
		}
		return append(parseDst, 0), nil
	case int:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int8:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int16:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int32:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case int64:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint8:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint16:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint32:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uint64:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case uintptr:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case float32:
		return appendSnapshotDispatchFloat64(parseDst, float64(getValue))
	case float64:
		return appendSnapshotDispatchFloat64(parseDst, getValue)
	case string:
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerString)
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(getValue)))
		return append(parseDst, getValue...), nil
	case []any:
		parseDst = append(parseDst, getSnapshotDispatchHashMarkerList)
		parseDst = appendSnapshotDispatchUint64(parseDst, uint64(len(getValue)))
		for _, getItem := range getValue {
			var parseErr error
			parseDst, parseErr = appendSnapshotDispatchValueLegacyBenchmark(parseDst, getItem)
			if parseErr != nil {
				return nil, parseErr
			}
		}
		return parseDst, nil
	case map[string]any:
		return buildSnapshotDispatchAnyMapLegacyBenchmark(parseDst, getValue)
	default:
		return appendSnapshotDispatchReflect(parseDst, reflect.ValueOf(parseValue))
	}
}

// buildSnapshotDispatchAnyMapLegacyBenchmark preserves the previous any-map dispatch hashing behavior for compare benchmarks.
func buildSnapshotDispatchAnyMapLegacyBenchmark(parseDst []byte, parseValue map[string]any) ([]byte, error) {
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
		parseDst, parseErr = appendSnapshotDispatchValueLegacyBenchmark(parseDst, parseEntry.getValue)
		if parseErr != nil {
			return nil, parseErr
		}
	}
	return parseDst, nil
}

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

// BenchmarkAppendSnapshotDispatchValueTypedScalarContainersCurrentVsLegacy compares typed scalar container hashing against the previous reflect-heavy fallback.
func BenchmarkAppendSnapshotDispatchValueTypedScalarContainersCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("typed_string_map/current", func(parseB *testing.B) {
		parseValue := map[string]string{
			"title": "Orders",
			"state": "open",
			"owner": "ops",
			"team":  "a",
		}
		getScratch := make([]byte, 0, 256)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getScratch = getScratch[:0]
			if _, parseErr := appendSnapshotDispatchValue(getScratch, parseValue); parseErr != nil {
				parseB.Fatalf("appendSnapshotDispatchValue returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("typed_string_map/legacy", func(parseB *testing.B) {
		parseValue := map[string]string{
			"title": "Orders",
			"state": "open",
			"owner": "ops",
			"team":  "a",
		}
		getScratch := make([]byte, 0, 256)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getScratch = getScratch[:0]
			if _, parseErr := appendSnapshotDispatchValueLegacyBenchmark(getScratch, parseValue); parseErr != nil {
				parseB.Fatalf("appendSnapshotDispatchValueLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("typed_int_slice/current", func(parseB *testing.B) {
		parseValue := []int{1, 2, 3, 4, 5, 6, 7, 8}
		getScratch := make([]byte, 0, 256)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getScratch = getScratch[:0]
			if _, parseErr := appendSnapshotDispatchValue(getScratch, parseValue); parseErr != nil {
				parseB.Fatalf("appendSnapshotDispatchValue returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("typed_int_slice/legacy", func(parseB *testing.B) {
		parseValue := []int{1, 2, 3, 4, 5, 6, 7, 8}
		getScratch := make([]byte, 0, 256)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getScratch = getScratch[:0]
			if _, parseErr := appendSnapshotDispatchValueLegacyBenchmark(getScratch, parseValue); parseErr != nil {
				parseB.Fatalf("appendSnapshotDispatchValueLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
}
