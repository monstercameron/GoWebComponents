package runtime2

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

var storeBinarySourceIDTableComparePayloadSink []byte
var storeBinarySourceIDTableCompareSourceIDsSink []string

// buildBinarySourceIDTableCompareBenchmarkSourceIDs returns one stable canonical source-ID set for compare benchmarks.
func buildBinarySourceIDTableCompareBenchmarkSourceIDs() []string {
	return []string{
		"feed.items",
		"flags.beta",
		"region.count",
		"region.mode",
		"status.code",
		"user.id",
		"user.role",
		"view.layout",
	}
}

// appendBinarySourceIDTableFromNormalizedLegacyBenchmark preserves the previous source-ID table append behavior for compare benchmarks.
func appendBinarySourceIDTableFromNormalizedLegacyBenchmark(parsePayload []byte, parseNormalizedSourceIDs []string) ([]byte, error) {
	if len(parseNormalizedSourceIDs) > 0xFFFF {
		return nil, fmt.Errorf("runtime2: source ID table length %d exceeds uint16", len(parseNormalizedSourceIDs))
	}
	parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseNormalizedSourceIDs)))
	for _, parseSourceID := range parseNormalizedSourceIDs {
		if len(parseSourceID) > 0xFFFF {
			return nil, fmt.Errorf("runtime2: source ID %q is too large for binary table", parseSourceID)
		}
		parsePayload = appendBinaryUint16(parsePayload, uint16(len(parseSourceID)))
		parsePayload = append(parsePayload, parseSourceID...)
	}
	return parsePayload, nil
}

// getBinarySourceIDUnsupportedRuneLegacyBenchmark preserves the previous rune-validation behavior for compare benchmarks.
func getBinarySourceIDUnsupportedRuneLegacyBenchmark(parseSourceID string) (rune, bool) {
	for _, parseRune := range parseSourceID {
		parseAllowed := parseRune == '.' || parseRune == '-' || parseRune == '_' || parseRune == ':'
		if parseAllowed || (parseRune >= 'a' && parseRune <= 'z') || (parseRune >= 'A' && parseRune <= 'Z') || (parseRune >= '0' && parseRune <= '9') {
			continue
		}
		return parseRune, true
	}
	return 0, false
}

// parseBinarySourceIDTableIntoLegacyBenchmark preserves the previous source-ID table parse behavior for compare benchmarks.
func parseBinarySourceIDTableIntoLegacyBenchmark(parseDst []string, parsePayload []byte) ([]string, error) {
	if len(parsePayload) < 2 {
		return parseDst, fmt.Errorf("runtime2: binary source-id-table count is truncated")
	}
	parseOffset := 2
	parsePreviousSourceID := ""
	parseCount := int(binary.LittleEndian.Uint16(parsePayload[0:2]))
	for parseIndex := 0; parseIndex < parseCount; parseIndex++ {
		if parseOffset+2 > len(parsePayload) {
			return parseDst, fmt.Errorf("runtime2: decode source_id[%d]: length is truncated", parseIndex)
		}
		parseSourceIDLength := int(binary.LittleEndian.Uint16(parsePayload[parseOffset : parseOffset+2]))
		parseOffset += 2
		if parseSourceIDLength > len(parsePayload)-parseOffset {
			return parseDst, fmt.Errorf(
				"runtime2: decode source_id[%d]: length %d exceeds payload size %d",
				parseIndex,
				parseSourceIDLength,
				len(parsePayload)-parseOffset,
			)
		}
		parseSourceID := string(parsePayload[parseOffset : parseOffset+parseSourceIDLength])
		parseOffset += parseSourceIDLength
		if parseSourceID == "" {
			return parseDst, fmt.Errorf("runtime2: source ID is required")
		}
		if strings.TrimSpace(parseSourceID) != parseSourceID {
			return parseDst, fmt.Errorf("runtime2: source ID %q must not contain surrounding whitespace", parseSourceID)
		}
		parseUnsupportedRune, hasUnsupportedRune := getBinarySourceIDUnsupportedRuneLegacyBenchmark(parseSourceID)
		if hasUnsupportedRune {
			return parseDst, fmt.Errorf("runtime2: source ID %q contains unsupported character %q", parseSourceID, string(parseUnsupportedRune))
		}
		if parseIndex > 0 && parseSourceID <= parsePreviousSourceID {
			if parseSourceID == parsePreviousSourceID {
				return parseDst, fmt.Errorf("runtime2: source-id-table contains duplicates")
			}
			return parseDst, fmt.Errorf("runtime2: source-id-table is not canonical at index %d", parseIndex)
		}
		parseDst = append(parseDst, parseSourceID)
		parsePreviousSourceID = parseSourceID
	}
	if parseOffset != len(parsePayload) {
		return parseDst, fmt.Errorf("runtime2: source-id-table has %d trailing bytes", len(parsePayload)-parseOffset)
	}
	return parseDst, nil
}

// BenchmarkBinarySourceIDTableCurrentVsLegacy compares source-ID table append and parse paths against legacy behavior.
func BenchmarkBinarySourceIDTableCurrentVsLegacy(parseB *testing.B) {
	parseSourceIDs := buildBinarySourceIDTableCompareBenchmarkSourceIDs()
	parsePayload, parsePayloadErr := appendBinarySourceIDTableFromNormalized(make([]byte, 0, 256), parseSourceIDs)
	if parsePayloadErr != nil {
		parseB.Fatalf("appendBinarySourceIDTableFromNormalized(seed) returned error: %v", parsePayloadErr)
	}
	parseB.Run("append/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseBuiltPayload, parseBuildErr := appendBinarySourceIDTableFromNormalizedLegacyBenchmark(make([]byte, 0, 16), parseSourceIDs)
			if parseBuildErr != nil {
				parseB.Fatalf("appendBinarySourceIDTableFromNormalizedLegacyBenchmark returned error: %v", parseBuildErr)
			}
			storeBinarySourceIDTableComparePayloadSink = parseBuiltPayload
		}
	})
	parseB.Run("append/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseBuiltPayload, parseBuildErr := appendBinarySourceIDTableFromNormalized(make([]byte, 0, 16), parseSourceIDs)
			if parseBuildErr != nil {
				parseB.Fatalf("appendBinarySourceIDTableFromNormalized returned error: %v", parseBuildErr)
			}
			storeBinarySourceIDTableComparePayloadSink = parseBuiltPayload
		}
	})
	parseB.Run("parse/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseSourceIDsCopy, parseParseErr := parseBinarySourceIDTableIntoLegacyBenchmark(make([]string, 0, len(parseSourceIDs)), parsePayload)
			if parseParseErr != nil {
				parseB.Fatalf("parseBinarySourceIDTableIntoLegacyBenchmark returned error: %v", parseParseErr)
			}
			storeBinarySourceIDTableCompareSourceIDsSink = parseSourceIDsCopy
		}
	})
	parseB.Run("parse/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			parseSourceIDsCopy, parseParseErr := parseBinarySourceIDTableInto(make([]string, 0, len(parseSourceIDs)), parsePayload)
			if parseParseErr != nil {
				parseB.Fatalf("parseBinarySourceIDTableInto returned error: %v", parseParseErr)
			}
			storeBinarySourceIDTableCompareSourceIDsSink = parseSourceIDsCopy
		}
	})
}
