//go:build !js || !wasm

package runtime

import "testing"

func BenchmarkGoUseFetchStubInit(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_, _ = GoUseFetch("/api/test")
	}
}

func BenchmarkGoUseFetchStubGetter(parseB *testing.B) {
	parseGetter, _ := GoUseFetch("/api/test")
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = parseGetter()
	}
}

func BenchmarkGoUseFetchStubRefetch(parseB *testing.B) {
	_, parseRefetch := GoUseFetch("/api/test")
	parseB.ReportAllocs()
	for parseB.Loop() {
		parseRefetch()
	}
}
