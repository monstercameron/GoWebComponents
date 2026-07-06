package runtime

import (
	"math"
	"testing"
)

// TestFastEqualTreatsNaNAsEqual pins Object.is-style dependency comparison: a NaN
// dependency must compare EQUAL to itself, otherwise areDepsEqual reports "changed"
// on every render and the associated effect/memo re-runs forever.
func TestFastEqualTreatsNaNAsEqual(parseT *testing.T) {
	parseNaN64 := math.NaN()
	if !fastEqual(parseNaN64, parseNaN64) {
		parseT.Fatal("float64 NaN should compare equal to itself")
	}
	parseNaN32 := float32(math.NaN())
	if !fastEqual(parseNaN32, parseNaN32) {
		parseT.Fatal("float32 NaN should compare equal to itself")
	}
	// Sanity: distinct finite floats still compare unequal.
	if fastEqual(1.5, 2.5) {
		parseT.Fatal("distinct floats must not compare equal")
	}
	// A NaN dep must not spuriously re-run an effect.
	if !areDepsEqual([]any{parseNaN64}, []any{parseNaN64}) {
		parseT.Fatal("a NaN dependency should be considered unchanged across renders")
	}
}
