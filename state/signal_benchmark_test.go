package state

import "testing"

var benchmarkComputedTextNode any

// BenchmarkComputedSignalTextTwoSources measures computed reactive-text construction for two sources.
func BenchmarkComputedSignalTextTwoSources(parseB *testing.B) {
	parseFirst := NewSignal("Ada")
	parseLast := NewSignal("Lovelace")
	parseComputed := NewComputed(func() string {
		return parseFirst.Get() + " " + parseLast.Get()
	}, parseFirst, parseLast)
	parseRender := func(parseValue string) string { return parseValue }

	parseB.ReportAllocs()
	for parseB.Loop() {
		benchmarkComputedTextNode = parseComputed.Text(parseRender)
	}
}
