package runtime2

import (
	"encoding/binary"
	"fmt"
	"math"
	"testing"
)

// parseBinarySourceValueLegacyBenchmark preserves the previous recursive binary source-value decode path.
func parseBinarySourceValueLegacyBenchmark(parsePayload []byte) (any, error) {
	if len(parsePayload) == 0 {
		return nil, fmt.Errorf("runtime2: binary source value payload is empty")
	}
	parseKind := parsePayload[0]
	switch parseKind {
	case binarySourceValueKindBoolTrue:
		return true, nil
	case binarySourceValueKindBoolFalse:
		return false, nil
	case binarySourceValueKindNumber:
		if len(parsePayload) != 9 {
			return nil, fmt.Errorf("runtime2: source-value number payload length %d is invalid", len(parsePayload))
		}
		return math.Float64frombits(binary.LittleEndian.Uint64(parsePayload[1:9])), nil
	case binarySourceValueKindString:
		if len(parsePayload) < 5 {
			return nil, fmt.Errorf("runtime2: source-value string payload is truncated")
		}
		parseLength := int(binary.LittleEndian.Uint32(parsePayload[1:5]))
		if parseLength > len(parsePayload)-5 {
			return nil, fmt.Errorf("runtime2: source-value string length %d exceeds payload size %d", parseLength, len(parsePayload)-5)
		}
		if 5+parseLength != len(parsePayload) {
			return nil, fmt.Errorf("runtime2: source-value string has %d trailing bytes", len(parsePayload)-(5+parseLength))
		}
		return string(parsePayload[5 : 5+parseLength]), nil
	case binarySourceValueKindList:
		return parseBinarySourceListValueLegacyBenchmark(parsePayload)
	case binarySourceValueKindNil:
		return nil, nil
	case binarySourceValueKindMap:
		return parseBinarySourceMapValueLegacyBenchmark(parsePayload)
	default:
		return nil, fmt.Errorf("runtime2: binary source value kind %d is unsupported", parseKind)
	}
}

// parseBinarySourceListValueLegacyBenchmark preserves the previous append-based list decode loop.
func parseBinarySourceListValueLegacyBenchmark(parsePayload []byte) ([]any, error) {
	if len(parsePayload) < 2 {
		return nil, fmt.Errorf("runtime2: source-value list payload is truncated")
	}
	parseCount := int(parsePayload[1])
	parseList := make([]any, 0, parseCount)
	parseOffset := 2
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		if parseOffset+4 > len(parsePayload) {
			return nil, fmt.Errorf("runtime2: decode list item[%d] length: payload is truncated", parseIndex)
		}
		parseItemLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
		parseOffset += 4
		if parseItemLength > len(parsePayload)-parseOffset {
			return nil, fmt.Errorf(
				"runtime2: decode list item[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseItemLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseItemValue, parseErr := parseBinarySourceValueLegacyBenchmark(parsePayload[parseOffset : parseOffset+parseItemLength])
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode list item[%d]: %w", parseIndex, parseErr)
		}
		parseList = append(parseList, parseItemValue)
		parseOffset += parseItemLength
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-value list has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseList, nil
}

// parseBinarySourceMapValueLegacyBenchmark preserves the previous map decode loop with direct len(payload)-offset checks.
func parseBinarySourceMapValueLegacyBenchmark(parsePayload []byte) (map[string]any, error) {
	if len(parsePayload) < 2 {
		return nil, fmt.Errorf("runtime2: source-value map payload is truncated")
	}
	parseCount := int(parsePayload[1])
	parseMap := make(map[string]any, parseCount)
	parsePreviousKey := ""
	parseOffset := 2
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		if parseOffset+2 > len(parsePayload) {
			return nil, fmt.Errorf("runtime2: decode map key[%d]: key length is truncated", parseIndex)
		}
		parseKeyLength := int(binary.LittleEndian.Uint16(parsePayload[parseOffset : parseOffset+2]))
		parseOffset += 2
		if parseKeyLength > len(parsePayload)-parseOffset {
			return nil, fmt.Errorf(
				"runtime2: decode map key[%d]: key length %d exceeds payload size %d",
				parseIndex,
				parseKeyLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseKey := string(parsePayload[parseOffset : parseOffset+parseKeyLength])
		parseOffset += parseKeyLength
		if parseIndex > 0 && parseKey <= parsePreviousKey {
			return nil, fmt.Errorf("runtime2: source-value map key %q is not in canonical order", parseKey)
		}
		if parseOffset+4 > len(parsePayload) {
			return nil, fmt.Errorf("runtime2: decode map value[%d] length: payload is truncated", parseIndex)
		}
		parseValueLength := int(binary.LittleEndian.Uint32(parsePayload[parseOffset : parseOffset+4]))
		parseOffset += 4
		if parseValueLength > len(parsePayload)-parseOffset {
			return nil, fmt.Errorf(
				"runtime2: decode map value[%d] payload: length %d exceeds payload size %d",
				parseIndex,
				parseValueLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseValue, parseErr := parseBinarySourceValueLegacyBenchmark(parsePayload[parseOffset : parseOffset+parseValueLength])
		if parseErr != nil {
			return nil, fmt.Errorf("runtime2: decode map value[%d]: %w", parseIndex, parseErr)
		}
		parseMap[parseKey] = parseValue
		parsePreviousKey = parseKey
		parseOffset += parseValueLength
	}
	if parseOffset != len(parsePayload) {
		return nil, fmt.Errorf("runtime2: source-value map has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseMap, nil
}

// BenchmarkParseBinarySourceValueNestedCurrentVsLegacy compares current recursive list/map decode loops against legacy behavior.
func BenchmarkParseBinarySourceValueNestedCurrentVsLegacy(parseB *testing.B) {
	parsePayload, parsePayloadErr := BuildBinarySourceValue(map[string]any{
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
	})
	if parsePayloadErr != nil {
		parseB.Fatalf("BuildBinarySourceValue returned error: %v", parsePayloadErr)
	}
	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseBinarySourceValueLegacyBenchmark(parsePayload); parseErr != nil {
				parseB.Fatalf("parseBinarySourceValueLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := ParseBinarySourceValue(parsePayload); parseErr != nil {
				parseB.Fatalf("ParseBinarySourceValue returned error: %v", parseErr)
			}
		}
	})
}
