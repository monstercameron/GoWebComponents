//go:build !js || !wasm
// +build !js !wasm

package runtime

import "testing"

func BenchmarkGoUseFetchStubInit(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_, _ = GoUseFetch("/api/test")
	}
}

func BenchmarkGoUseFetchStubGetter(parseB *testing.B) {
	parseGetter, _ := GoUseFetch("/api/test")
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = parseGetter()
	}
}

func BenchmarkGoUseFetchStubRefetch(parseB *testing.B) {
	_, parseRefetch := GoUseFetch("/api/test")
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRefetch()
	}
}
