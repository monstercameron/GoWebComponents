package runtime2

import (
	"strings"
	"testing"
)

var storeTrimCheckBenchmarkSink bool

// parseRuntimeHasTrimmedNonWhitespaceTextLegacyBenchmark preserves the previous TrimSpace-based check for benchmark comparison.
func parseRuntimeHasTrimmedNonWhitespaceTextLegacyBenchmark(parseText string) bool {
	return strings.TrimSpace(parseText) != ""
}

// buildTrimCheckBenchmarkValues builds representative text inputs for trim-check benchmarking.
func buildTrimCheckBenchmarkValues() []string {
	return []string{
		"class",
		"style",
		"aria-label",
		"data-user-id",
		"text",
		"  padded",
		"\tindented",
		"",
		"   ",
	}
}

// BenchmarkParseRuntimeHasTrimmedNonWhitespaceTextCurrentVsLegacy compares the ASCII-fast trim-check helper against the legacy TrimSpace check.
func BenchmarkParseRuntimeHasTrimmedNonWhitespaceTextCurrentVsLegacy(parseB *testing.B) {
	parseValues := buildTrimCheckBenchmarkValues()
	parseB.Run("current", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeTrimCheckBenchmarkSink = parseRuntimeHasTrimmedNonWhitespaceText(parseValues[parseIndex%len(parseValues)])
		}
	})
	parseB.Run("legacy", func(parseB *testing.B) {
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			storeTrimCheckBenchmarkSink = parseRuntimeHasTrimmedNonWhitespaceTextLegacyBenchmark(parseValues[parseIndex%len(parseValues)])
		}
	})
}
