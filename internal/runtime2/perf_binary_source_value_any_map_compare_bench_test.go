package runtime2

import (
	"encoding/binary"
	"fmt"
	"math"
	"sort"
	"testing"
)

// buildBinarySourceValueAnyMapCompareBenchValue returns one stable map[string]any payload for fast-path encode benchmarks.
func buildBinarySourceValueAnyMapCompareBenchValue() map[string]any {
	return map[string]any{
		"title": "Orders",
		"count": 42,
		"meta": map[string]any{
			"priority": "high",
			"visible":  true,
			"labels":   []any{"north", "south", "east", "west"},
		},
	}
}

// buildBinarySourceAnyMapIntoLegacyBenchmark preserves the previous key-sort then map-lookup map[string]any encoding path.
func buildBinarySourceAnyMapIntoLegacyBenchmark(dst []byte, parseMap map[string]any) ([]byte, error) {
	if parseMap == nil {
		return append(dst, binarySourceValueKindNil), nil
	}
	if len(parseMap) > binarySourceValueMapLimit {
		return nil, fmt.Errorf("runtime2: source map length %d exceeds limit %d", len(parseMap), binarySourceValueMapLimit)
	}
	parseKeyStrings, parseKeyCache := buildBinarySourceMapKeyBuffer(len(parseMap))
	for parseKey := range parseMap {
		parseKeyStrings = append(parseKeyStrings, parseKey)
	}
	sort.Strings(parseKeyStrings)
	defer storeBinarySourceMapKeyBuffer(parseKeyStrings, parseKeyCache)
	dst = append(dst, binarySourceValueKindMap, byte(len(parseKeyStrings)))
	for _, parseKey := range parseKeyStrings {
		if len(parseKey) > math.MaxUint16 {
			return nil, fmt.Errorf("runtime2: source map key %q is too large", parseKey)
		}
		dst = binary.LittleEndian.AppendUint16(dst, uint16(len(parseKey)))
		dst = append(dst, parseKey...)
		lenOff := len(dst)
		dst = append(dst, 0, 0, 0, 0)
		itemStart := len(dst)
		var parseErr error
		dst, parseErr = buildBinarySourceValueInto(dst, parseMap[parseKey])
		if parseErr != nil {
			return nil, parseErr
		}
		binary.LittleEndian.PutUint32(dst[lenOff:], uint32(len(dst)-itemStart))
	}
	return dst, nil
}

// BenchmarkBuildBinarySourceValueAnyMapFastPathCurrentVsLegacy compares current map-entry fast path against legacy key-sort lookup behavior.
func BenchmarkBuildBinarySourceValueAnyMapFastPathCurrentVsLegacy(parseB *testing.B) {
	parseValue := buildBinarySourceValueAnyMapCompareBenchValue()
	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildBinarySourceAnyMapIntoLegacyBenchmark(make([]byte, 0, 32), parseValue); parseErr != nil {
				parseB.Fatalf("buildBinarySourceAnyMapIntoLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := BuildBinarySourceValue(parseValue); parseErr != nil {
				parseB.Fatalf("BuildBinarySourceValue returned error: %v", parseErr)
			}
		}
	})
}
