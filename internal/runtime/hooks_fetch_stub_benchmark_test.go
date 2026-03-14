//go:build !js || !wasm
// +build !js !wasm

package runtime

import "testing"

func BenchmarkGoUseFetchStubInit(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = GoUseFetch("/api/test")
	}
}

func BenchmarkGoUseFetchStubGetter(b *testing.B) {
	getter, _ := GoUseFetch("/api/test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = getter()
	}
}

func BenchmarkGoUseFetchStubRefetch(b *testing.B) {
	_, refetch := GoUseFetch("/api/test")
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		refetch()
	}
}
