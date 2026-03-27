package runtime2

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

// buildBinarySourceIDTableFromNormalizedLegacyBenchmark preserves the previous coarse-capacity source-id table build path.
func buildBinarySourceIDTableFromNormalizedLegacyBenchmark(parseNormalizedSourceIDs []string) ([]byte, error) {
	parsePayload := make([]byte, 0, 2+len(parseNormalizedSourceIDs)*4)
	return appendBinarySourceIDTableFromNormalized(parsePayload, parseNormalizedSourceIDs)
}

// parseBinarySourceIDTableIntoLegacyBenchmark preserves the previous source-id table parse validation loop.
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

// getBinarySourceIDUnsupportedRuneLegacyBenchmark preserves the previous rune-loop source-id validator.
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

// BenchmarkParseBinarySourceIDTableCanonicalCurrentVsLegacy compares current source-id table build/parse paths against legacy behavior.
func BenchmarkParseBinarySourceIDTableCanonicalCurrentVsLegacy(parseB *testing.B) {
	parseSourceIDs := []string{
		"feed.items",
		"flags.beta",
		"stats.active",
		"user.id",
		"view.mode",
	}
	parsePayload, parsePayloadErr := BuildBinarySourceIDTable(parseSourceIDs)
	if parsePayloadErr != nil {
		parseB.Fatalf("BuildBinarySourceIDTable(seed) returned error: %v", parsePayloadErr)
	}
	parseB.Run("build/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildBinarySourceIDTableFromNormalizedLegacyBenchmark(parseSourceIDs); parseErr != nil {
				parseB.Fatalf("buildBinarySourceIDTableFromNormalizedLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("build/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildBinarySourceIDTableFromNormalized(parseSourceIDs); parseErr != nil {
				parseB.Fatalf("buildBinarySourceIDTableFromNormalized returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("parse/legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseBinarySourceIDTableIntoLegacyBenchmark(make([]string, 0, len(parseSourceIDs)), parsePayload); parseErr != nil {
				parseB.Fatalf("parseBinarySourceIDTableIntoLegacyBenchmark returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("parse/current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := parseBinarySourceIDTableInto(make([]string, 0, len(parseSourceIDs)), parsePayload); parseErr != nil {
				parseB.Fatalf("parseBinarySourceIDTableInto returned error: %v", parseErr)
			}
		}
	})
}
