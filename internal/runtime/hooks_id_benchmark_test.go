package runtime

import (
	"fmt"
	"testing"
)

var parseHookIDBenchmarkSink string

// BenchmarkFormatHookIDCurrentVsLegacy compares the current hook-ID formatter against the previous fmt.Sprintf path.
func BenchmarkFormatHookIDCurrentVsLegacy(parseB *testing.B) {
	parseIDs := []struct {
		parseID       int
		parsePosition int
	}{
		{parseID: 1, parsePosition: 0},
		{parseID: 42, parsePosition: 3},
		{parseID: 1234567, parsePosition: 17},
	}
	parseB.Run("legacy", func(parseLegacyB *testing.B) {
		parseLegacyB.ReportAllocs()
		for parseIndex := 0; parseLegacyB.Loop(); parseIndex++ {
			parsePair := parseIDs[parseIndex%len(parseIDs)]
			parseHookIDBenchmarkSink = parseFormatHookIDLegacy(parsePair.parseID, parsePair.parsePosition)
		}
	})
	parseB.Run("current", func(parseCurrentB *testing.B) {
		parseCurrentB.ReportAllocs()
		for parseIndex := 0; parseCurrentB.Loop(); parseIndex++ {
			parsePair := parseIDs[parseIndex%len(parseIDs)]
			parseHookIDBenchmarkSink = formatHookID(parsePair.parseID, parsePair.parsePosition)
		}
	})
}

// parseFormatHookIDLegacy preserves the previous fmt.Sprintf hook-ID formatting path for benchmark comparison.
func parseFormatHookIDLegacy(parseID int, parsePosition int) string {
	return fmt.Sprintf("gwc:%d:%d", parseID, parsePosition)
}
