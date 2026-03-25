//go:build js && wasm
// +build js,wasm

package fetch

import (
	"testing"
	"time"
)

func BenchmarkNormalizeMutationIDMicro(parseB *testing.B) {
	parseNow := time.Unix(1710000000, 0).UTC()

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = normalizeMutationID("", parseNow, parseI)
	}
}
