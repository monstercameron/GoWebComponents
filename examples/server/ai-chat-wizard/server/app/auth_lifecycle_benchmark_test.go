package app

import "testing"

// BenchmarkAuthFlowTokenHash measures SHA-256 token-hash generation used by verification/reset lifecycles.
func BenchmarkAuthFlowTokenHash(parseB *testing.B) {
	parseRawToken := parseBuildOpaqueAuthFlowToken()
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseBuildAuthFlowTokenHash(parseRawToken)
	}
}
