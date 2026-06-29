package runtime2

import (
	"encoding/binary"
	"fmt"
	"testing"
)

// buildBinarySourceValuesSectionCompareBenchmarkInput returns one stable source-id and source-value set for section benchmarks.
func buildBinarySourceValuesSectionCompareBenchmarkInput(parseB *testing.B) ([]string, map[string]any) {
	parseB.Helper()
	parseSourceIDs := []string{"feed", "flags", "labels", "stats"}
	parseSourceValues := map[string]any{
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
	}
	return parseSourceIDs, parseSourceValues
}

// appendBinarySourceValuesSectionLegacyBenchmark preserves the previous checked source-values encode loop.
func appendBinarySourceValuesSectionLegacyBenchmark(
	parsePayload []byte,
	parseSourceIDs []string,
	parseSourceValues map[string]any,
) ([]byte, error) {
	for _, parseSourceID := range parseSourceIDs {
		parseSourceValue, parseHasSourceValue := parseSourceValues[parseSourceID]
		if !parseHasSourceValue {
			return nil, fmt.Errorf("runtime2: missing source value for %q", parseSourceID)
		}
		lenOff := len(parsePayload)
		parsePayload = append(parsePayload, 0, 0, 0, 0)
		itemStart := len(parsePayload)
		var parseErr error
		parsePayload, parseErr = buildBinarySourceValueInto(parsePayload, parseSourceValue)
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: encode source %q: %w", parseSourceID, parseErr)
		}
		binary.LittleEndian.PutUint32(parsePayload[lenOff:], uint32(len(parsePayload)-itemStart))
	}
	return parsePayload, nil
}

// parseBinarySourceValuesSectionLegacyBenchmark preserves the previous offset-based decode loop.
func parseBinarySourceValuesSectionLegacyBenchmark(parseSourceIDs []string, parsePayload []byte) (map[string]any, error) {
	if len(parseSourceIDs) == 0 {
		if len(parsePayload) != 0 {
			return nil, fmt.Errorf("runtime2: source-values payload has %d bytes with no source IDs", len(parsePayload))
		}
		return nil, nil
	}
	parseSources := make(map[string]any, len(parseSourceIDs))
	parseOffset := 0
	for parseIndex, parseSourceID := range parseSourceIDs {
		parseRemaining := len(parsePayload) - parseOffset
		if parseRemaining < 4 {
			return nil, fmt.Errorf("runtime2: decode source value[%d] length: payload is truncated", parseIndex)
		}
		parseValueLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset:]))
		parseOffset += 4
		parseRemaining -= 4
		if parseValueLength > parseRemaining {
			return nil, fmt.Errorf(
				"runtime2: decode source value[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseValueLength,
				parseRemaining,
			)
		}
		parseValue, parseErr := ParseBinarySourceValue(parsePayload[parseOffset : parseOffset+parseValueLength])
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode source %q: %w", parseSourceID, parseErr)
		}
		parseSources[parseSourceID] = parseValue
		parseOffset += parseValueLength
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-values payload has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseSources, nil
}

// BenchmarkBinarySourceValuesSectionCurrentVsLegacy compares source-values section encode/decode paths against legacy behavior.
func BenchmarkBinarySourceValuesSectionCurrentVsLegacy(parseB *testing.B) {
	parseSourceIDs, parseSourceValues := buildBinarySourceValuesSectionCompareBenchmarkInput(parseB)
	parsePayload, parsePayloadErr := appendBinarySourceValuesSection(make([]byte, 0, 512), parseSourceIDs, parseSourceValues)
	if parsePayloadErr != nil {
		parseB.Fatalf("appendBinarySourceValuesSection(seed) returned error: %v", parsePayloadErr)
	}
	parseB.Run("append/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := appendBinarySourceValuesSectionLegacyBenchmark(make([]byte, 0, 512), parseSourceIDs, parseSourceValues); parseErr != nil {
				parseB.Fatalf("appendBinarySourceValuesSectionLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("append/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := appendBinarySourceValuesSectionTrusted(make([]byte, 0, 512), parseSourceIDs, parseSourceValues); parseErr != nil {
				parseB.Fatalf("appendBinarySourceValuesSectionTrusted returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("parse/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseBinarySourceValuesSectionLegacyBenchmark(parseSourceIDs, parsePayload); parseErr != nil {
				parseB.Fatalf("parseBinarySourceValuesSectionLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("parse/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseBinarySourceValuesSection(parseSourceIDs, parsePayload); parseErr != nil {
				parseB.Fatalf("parseBinarySourceValuesSection returned error: %v", parseErr)
			}
		}
	})
}
