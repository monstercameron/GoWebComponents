package runtime2

import "testing"

// BenchmarkBuildStructuredCloneSnapshotEnvelopeJSON benchmarks structured-clone snapshot envelope encoding.
func BenchmarkBuildStructuredCloneSnapshotEnvelopeJSON(parseB *testing.B) {
	parseEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            3,
		InputVersion:     42,
		SourceVersion:    7,
		Props: map[string]any{
			"title":  "Orders",
			"status": "healthy",
			"count":  128,
		},
		Sources: map[string]any{
			"filter": "active",
			"page":   2,
		},
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope); parseErr != nil {
			parseB.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
		}
	}
}
