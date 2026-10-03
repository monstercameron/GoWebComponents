//go:build js && wasm

package fetch

import (
	"testing"
	"time"
)

func BenchmarkNormalizeMutationIDMicro(parseB *testing.B) {
	parseNow := time.Unix(1710000000, 0).UTC()

	parseB.ReportAllocs()
	parseI := 0
	for parseB.Loop() {
		_ = normalizeMutationID("", parseNow, parseI)
		parseI++
	}
}
