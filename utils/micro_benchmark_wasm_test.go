//go:build js && wasm
// +build js,wasm

package utils

import "testing"

func BenchmarkConfigureAndReadMemStatsSampleRateMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		ConfigureMemStatsSampleRate(int64((parseI % 512) + 1))
		if GetMemStatsSampleRate() == 0 {
			parseB.Fatal("expected non-zero mem stats sample rate")
		}
	}
}

func BenchmarkConfigureDebugNamespacesAndGetStatusMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		ConfigureDebugNamespaces(map[string]bool{
			"fetch": parseI%2 == 0,
			"ui":    true,
			"state": false,
		})
		parseStatus := GetDebugStatus()
		if _, parseOk := parseStatus["ui"]; !parseOk {
			parseB.Fatal("expected debug status to include configured namespace")
		}
	}
}
