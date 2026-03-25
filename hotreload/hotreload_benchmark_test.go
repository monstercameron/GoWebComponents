package hotreload

import "testing"

func BenchmarkEnabled(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = Enabled()
	}
}

func BenchmarkEnableDisableCycle(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		Enable()
		Disable()
	}
}
