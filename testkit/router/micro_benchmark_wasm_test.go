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
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = cloneURLValues(parseValues)
	}
}

func BenchmarkCloneStringMapMicroWasm(parseB *testing.B) {
	parseValues := map[string]string{
		"id":    "42",
		"scope": "settings",
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = cloneStringMap(parseValues)
	}
}
