package app

import "testing"

// BenchmarkAuthFlowTokenHash measures SHA-256 token-hash generation used by verification/reset lifecycles.
func BenchmarkAuthFlowTokenHash(parseB *testing.B) {
	parseRawToken := parseBuildOpaqueAuthFlowToken()
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseBuildAuthFlowTokenHash(parseRawToken)
	}
}
