//go:build js && wasm

package routertest

import (
	"net/url"
	"testing"
)

func BenchmarkCloneURLValuesMicroWasm(parseB *testing.B) {
	parseValues := url.Values{
		"tab":   {"billing"},
		"state": {"open", "closed"},
	}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = cloneURLValues(parseValues)
	}
}

func BenchmarkCloneStringMapMicroWasm(parseB *testing.B) {
	parseValues := map[string]string{
		"id":    "42",
		"scope": "settings",
	}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = cloneStringMap(parseValues)
	}
}
