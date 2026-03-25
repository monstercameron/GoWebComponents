//go:build js && wasm
// +build js,wasm

package fetch

import (
	"testing"
	"time"
)

func BenchmarkNormalizeMutationIDMicro(b *testing.B) {
	now := time.Unix(1710000000, 0).UTC()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = normalizeMutationID("", now, i)
	}
}
