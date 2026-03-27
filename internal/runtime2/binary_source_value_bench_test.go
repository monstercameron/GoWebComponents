package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// BenchmarkBuildBinarySourceValueAnyMapFastPath benchmarks map[string]any source-value encoding through the non-reflect fast path.
func BenchmarkBuildBinarySourceValueAnyMapFastPath(parseB *testing.B) {
	parseValue := map[string]any{
		"title": "Orders",
		"count": 42,
		"meta": map[string]any{
			"priority": "high",
			"visible":  true,
			"labels":   []any{"north", "south", "east", "west"},
		},
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := runtime2.BuildBinarySourceValue(parseValue); parseErr != nil {
			parseB.Fatalf("BuildBinarySourceValue returned error: %v", parseErr)
		}
	}
}

// BenchmarkParseBinarySourceIDTableCanonical benchmarks canonical source-ID table parse validation.
func BenchmarkParseBinarySourceIDTableCanonical(parseB *testing.B) {
	parsePayload, parseErr := runtime2.BuildBinarySourceIDTable([]string{
		"feed.items",
		"flags.beta",
		"stats.active",
		"user.id",
		"view.mode",
	})
	if parseErr != nil {
		parseB.Fatalf("BuildBinarySourceIDTable returned error: %v", parseErr)
	}
	parseB.ReportAllocs()
	parseB.SetBytes(int64(len(parsePayload)))
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := runtime2.ParseBinarySourceIDTable(parsePayload); parseErr != nil {
			parseB.Fatalf("ParseBinarySourceIDTable returned error: %v", parseErr)
		}
	}
}
