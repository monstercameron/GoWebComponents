package html

import "testing"

func BenchmarkDivNodeConstructionMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = Div(Props{Class: "bench"}, Text("hello"))
	}
}
