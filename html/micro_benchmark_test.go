package html

import "testing"

func BenchmarkDivNodeConstructionMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = Div(Props{Class: "bench"}, Text("hello"))
	}
}
