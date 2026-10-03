package hotreload

import "testing"

func BenchmarkEnableDisableCycleMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		Enable()
		Disable()
		_ = IsEnabled()
	}
}
