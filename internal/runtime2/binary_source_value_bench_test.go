package runtime2

import "testing"

// buildBinarySourceValueBenchmarkMapPayload builds one canonical map payload for source-value decode benchmarks.
func buildBinarySourceValueBenchmarkMapPayload(parseB *testing.B) []byte {
	parseB.Helper()
	parseValuePayload, parseErr := BuildBinarySourceValue(map[string]any{
		"alpha": "a",
		"beta":  2,
		"gamma": true,
		"delta": []any{
			map[string]any{"id": "a", "score": 9},
			map[string]any{"id": "b", "score": 7},
			map[string]any{"id": "c", "score": 5},
		},
		"theta": map[string]any{
			"city":    "Boston",
			"zip":     "02108",
			"visible": true,
		},
	})
	if parseErr != nil {
		parseB.Fatalf("BuildBinarySourceValue(map) returned error: %v", parseErr)
	}
	return parseValuePayload
}

// buildBinarySourceValueBenchmarkListPayload builds one canonical list payload for source-value decode benchmarks.
func buildBinarySourceValueBenchmarkListPayload(parseB *testing.B) []byte {
	parseB.Helper()
	parseValuePayload, parseErr := BuildBinarySourceValue([]any{
		"north",
		"south",
		12,
		64,
		true,
		map[string]any{
			"kind":  "point",
			"value": 99,
		},
		[]any{"inner-a", "inner-b", 7},
	})
	if parseErr != nil {
		parseB.Fatalf("BuildBinarySourceValue(list) returned error: %v", parseErr)
	}
	return parseValuePayload
}

// BenchmarkParseBinarySourceValueMap benchmarks map-heavy source-value decode.
func BenchmarkParseBinarySourceValueMap(parseB *testing.B) {
	parseValuePayload := buildBinarySourceValueBenchmarkMapPayload(parseB)
	parseB.ReportAllocs()
	parseB.SetBytes(int64(len(parseValuePayload)))
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := ParseBinarySourceValue(parseValuePayload); parseErr != nil {
			parseB.Fatalf("ParseBinarySourceValue(map) returned error: %v", parseErr)
		}
	}
}

// BenchmarkParseBinarySourceValueList benchmarks list-heavy source-value decode.
func BenchmarkParseBinarySourceValueList(parseB *testing.B) {
	parseValuePayload := buildBinarySourceValueBenchmarkListPayload(parseB)
	parseB.ReportAllocs()
	parseB.SetBytes(int64(len(parseValuePayload)))
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := ParseBinarySourceValue(parseValuePayload); parseErr != nil {
			parseB.Fatalf("ParseBinarySourceValue(list) returned error: %v", parseErr)
		}
	}
}
