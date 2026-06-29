package telemetryredaction

import "testing"

// BenchmarkRedactValueNestedArrays exercises the array-indexed path of
// redactValue, the per-element step that builds an indexed field path for
// every array entry before recursing. Telemetry payloads (logs, traces)
// routinely carry arrays, so this path runs on redacted telemetry framework-wide.
func BenchmarkRedactValueNestedArrays(parseB *testing.B) {
	parsePayload := map[string]any{
		"events": []any{
			map[string]any{"id": "a1", "name": "open", "weight": 1},
			map[string]any{"id": "a2", "name": "click", "weight": 2},
			map[string]any{"id": "a3", "name": "scroll", "weight": 3},
			map[string]any{"id": "a4", "name": "close", "weight": 4},
		},
		"tags":    []any{"alpha", "beta", "gamma", "delta", "epsilon"},
		"metrics": []any{1.5, 2.5, 3.5, 4.5, 5.5, 6.5, 7.5, 8.5},
	}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = redactValue("root", parsePayload)
	}
}
