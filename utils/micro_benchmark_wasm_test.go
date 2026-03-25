//go:build js && wasm
// +build js,wasm

package utils

import "testing"

func BenchmarkConfigureAndReadMemStatsSampleRateMicro(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ConfigureMemStatsSampleRate(int64((i%512)+1))
		if GetMemStatsSampleRate() == 0 {
			b.Fatal("expected non-zero mem stats sample rate")
		}
	}
}

func BenchmarkConfigureDebugNamespacesAndGetStatusMicro(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ConfigureDebugNamespaces(map[string]bool{
			"fetch": i%2 == 0,
			"ui":    true,
			"state": false,
		})
		status := GetDebugStatus()
		if _, ok := status["ui"]; !ok {
			b.Fatal("expected debug status to include configured namespace")
		}
	}
}
