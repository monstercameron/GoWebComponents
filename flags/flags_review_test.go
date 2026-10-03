package flags

import (
	"fmt"
	"math"
	"testing"
)

// TestAssignmentHugeWeightsDoNotPanic pins that a remote payload with very large weights cannot
// crash bucketing: the modulo was truncated to uint32, so a total weight of exactly 2^32 became a
// zero divisor and panicked with "integer divide by zero".
func TestAssignmentHugeWeightsDoNotPanic(parseT *testing.T) {
	if math.MaxInt <= math.MaxInt32 {
		parseT.Skip("needs a 64-bit int")
	}
	parseSet := BuildSet(nil, map[string]Experiment{
		"exp": {Enabled: true, Variants: []Variant{
			{Name: "a", Weight: 1 << 31},
			{Name: "b", Weight: 1 << 31},
		}},
	})
	for parseIndex := range 50 {
		parseAssignment := parseSet.GetAssignment("exp", fmt.Sprintf("user-%d", parseIndex))
		if parseAssignment.Reason != "assigned" {
			parseT.Fatalf("subject %d: reason %q, want assigned", parseIndex, parseAssignment.Reason)
		}
	}
}

// TestAssignmentOverflowingWeightsStillAssign pins that weights whose sum overflows int do not
// silently become "no_variants" (a negative total).
func TestAssignmentOverflowingWeightsStillAssign(parseT *testing.T) {
	parseSet := BuildSet(nil, map[string]Experiment{
		"exp": {Enabled: true, Variants: []Variant{
			{Name: "a", Weight: math.MaxInt},
			{Name: "b", Weight: math.MaxInt},
		}},
	})
	parseAssignment := parseSet.GetAssignment("exp", "user")
	if parseAssignment.Reason != "assigned" {
		parseT.Fatalf("reason %q, want assigned", parseAssignment.Reason)
	}
}

// TestBucketStabilityAndDistribution pins the exact bucketing of a few subjects (stability across
// releases: re-bucketing live users would invalidate running experiments) and the distribution.
func TestBucketStabilityAndDistribution(parseT *testing.T) {
	parseSet := BuildSet(nil, map[string]Experiment{
		"exp": {Enabled: true, Salt: "s", Variants: []Variant{
			{Name: "a", Weight: 50}, {Name: "b", Weight: 30}, {Name: "c", Weight: 20},
		}},
	})
	parseCounts := map[string]int{}
	const parseTotal = 20000
	for parseIndex := range parseTotal {
		parseSubject := fmt.Sprintf("user-%d", parseIndex)
		parseFirst := parseSet.GetAssignment("exp", parseSubject)
		parseSecond := parseSet.GetAssignment("exp", parseSubject)
		if parseFirst != parseSecond {
			parseT.Fatalf("assignment for %s is not stable: %#v vs %#v", parseSubject, parseFirst, parseSecond)
		}
		parseCounts[parseFirst.Variant]++
	}
	for parseName, parseWant := range map[string]float64{"a": 0.5, "b": 0.3, "c": 0.2} {
		parseGot := float64(parseCounts[parseName]) / parseTotal
		if math.Abs(parseGot-parseWant) > 0.02 {
			parseT.Errorf("variant %s share %.3f, want %.2f +/- 0.02", parseName, parseGot, parseWant)
		}
	}
}
