package render

import "testing"

func BenchmarkNewResourceControllerMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = NewResourceController[int]()
	}
}
