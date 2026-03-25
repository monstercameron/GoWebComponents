package html

import "testing"

func BenchmarkDivNodeConstructionMicro(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Div(Props{Class: "bench"}, Text("hello"))
	}
}
