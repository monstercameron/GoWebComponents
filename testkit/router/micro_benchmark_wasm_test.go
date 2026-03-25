//go:build js && wasm
// +build js,wasm

package routertest

import (
	"net/url"
	"testing"
)

func BenchmarkCloneURLValuesMicroWasm(b *testing.B) {
	values := url.Values{
		"tab":   {"billing"},
		"state": {"open", "closed"},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cloneURLValues(values)
	}
}

func BenchmarkCloneStringMapMicroWasm(b *testing.B) {
	values := map[string]string{
		"id":    "42",
		"scope": "settings",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = cloneStringMap(values)
	}
}
