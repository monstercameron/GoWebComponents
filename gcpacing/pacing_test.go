package gcpacing_test

import (
	"runtime"
	"runtime/debug"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/gcpacing"
)

// v5 P4.4 — the GC pacing pass, acceptance M7.
//
// M7 wants the render thread's MAXIMUM GC pause under 3 ms. A pause is a
// dropped frame, so the maximum is the metric; an average of 1 ms with a 20 ms
// outlier is worse than a steady 2 ms.
//
// What these tests can and cannot establish is worth being precise about. The
// pacing knobs are identical on every platform and the DIRECTION of their effect
// is the same, so a native measurement can show that the responsive profile
// shortens the longest pause. It cannot produce M7's number: Go's js/wasm
// runtime is single-threaded and marks without the parallel assist native Go
// has. The absolute figure comes from the P0.2 browser harness.

// allocateGarbage produces collectable garbage with a live working set, which is
// what a render thread actually does — a steady churn of elements and props
// against a retained tree, not a pure allocation loop.
func allocateGarbage(parseRounds int, parseLiveSet int) []byte {
	parseLive := make([][]byte, parseLiveSet)
	for parseRound := range parseRounds {
		parseChunk := make([]byte, 4096)
		for parseIndex := range 64 {
			parseChunk[parseIndex*64] = byte(parseRound)
		}
		parseLive[parseRound%parseLiveSet] = parseChunk
	}
	return parseLive[0]
}

// measurePauses runs a workload under a profile and reports its pauses.
func measurePauses(parseT *testing.T, parseProfile gcpacing.Profile) gcpacing.PauseStats {
	parseT.Helper()

	_, parsePrevious, parseErr := gcpacing.Apply(parseProfile, 0)
	if parseErr != nil {
		parseT.Fatalf("Apply(%s): %v", parseProfile, parseErr)
	}
	defer parsePrevious.Restore()

	var parseBefore, parseAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&parseBefore)

	parseKeep := allocateGarbage(200_000, 512)

	runtime.ReadMemStats(&parseAfter)
	runtime.KeepAlive(parseKeep)

	parseStats, parseStatsErr := gcpacing.ReadPauses(parseBefore, parseAfter)
	if parseStatsErr != nil {
		parseT.Fatalf("ReadPauses: %v", parseStatsErr)
	}
	return parseStats
}

// ---------------------------------------------------------------- the pass

// TestResponsiveProfileCollectsMoreOften is the only part of P4.4's effect a
// native run can establish, and it is established deterministically.
//
// An earlier version of this test also asserted that the responsive profile
// produced a SHORTER maximum pause, and reported a tidy 13.0 ms -> 6.98 ms to
// prove it. That number was noise. Repeated runs on this machine gave, for the
// same two profiles:
//
//	throughput  n= 48  p99 0.71 ms  max  1.05 ms
//	responsive  n=354  p99 0.53 ms  max  1.51 ms
//	throughput  n= 43  p99 1.04 ms  max  9.53 ms
//	responsive  n=273  p99 0.51 ms  max  0.55 ms
//	throughput  n= 28  p99 0.00 ms  max  0.00 ms
//	responsive  n=188  p99 0.51 ms  max 13.95 ms
//
// Every median was 0.0000 ms — below the platform's timer resolution — and the
// maxima swing by an order of magnitude in BOTH directions. Two things make the
// original comparison invalid rather than merely noisy:
//
//  1. Maxima from samples of different sizes are not comparable. The responsive
//     profile takes 4-8x more samples, and the expected maximum of a sample
//     grows with its size, so it is biased upward before any real effect.
//  2. The pauses being compared are at or below measurement resolution here, so
//     what is being differenced is mostly scheduler noise.
//
// So this asserts the mechanism, which is deterministic, and leaves the pause
// claim to the browser where the pauses are large enough to measure.
func TestResponsiveProfileCollectsMoreOften(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping an allocation-heavy measurement in short mode")
	}

	parseThroughput := measurePauses(parseT, gcpacing.ProfileThroughput)
	parseResponsive := measurePauses(parseT, gcpacing.ProfileResponsive)

	parseT.Logf("throughput: %d collections, p99 %.3f ms, max %.3f ms",
		parseThroughput.Collections, float64(parseThroughput.P99Nanos)/1e6, parseThroughput.MaxPauseMillis())
	parseT.Logf("responsive: %d collections, p99 %.3f ms, max %.3f ms",
		parseResponsive.Collections, float64(parseResponsive.P99Nanos)/1e6, parseResponsive.MaxPauseMillis())

	if parseResponsive.Collections == 0 || parseThroughput.Collections == 0 {
		parseT.Skip("the workload did not trigger collections on this machine; the comparison would be vacuous")
	}

	// The mechanism: a smaller heap trigger means more, smaller collections.
	// This one is robust — it held on every observed run, by a wide margin.
	if parseResponsive.Collections <= parseThroughput.Collections {
		parseT.Errorf("responsive ran %d collections and throughput %d; responsive must collect MORE often, which is the whole mechanism",
			parseResponsive.Collections, parseThroughput.Collections)
	}
}

// TestM7CannotBeSettledNatively records what this package does NOT establish.
//
// A test that says so is worth more than a green check that implies otherwise.
// M7's target is a 3 ms maximum on the render thread; the v4 baseline measured
// 6.60 ms in a browser. Native pauses here are at timer resolution, and js/wasm
// marks single-threaded without native Go's parallel assist, so neither the
// magnitude nor the shape transfers. The P0.2 harness owns the gating figure.
func TestM7CannotBeSettledNatively(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping an allocation-heavy measurement in short mode")
	}

	parseStats := measurePauses(parseT, gcpacing.ProfileResponsive)
	parseT.Logf("M7 (native, NOT a gate): %d collections, max %.3f ms, p99 %.3f ms, median %.3f ms",
		parseStats.Collections, parseStats.MaxPauseMillis(),
		float64(parseStats.P99Nanos)/1e6, float64(parseStats.MedianNanos)/1e6)
	parseT.Log("M7 remains OPEN. Native pauses sit at timer resolution and js/wasm collects differently; " +
		"this run cannot pass or fail M7 and does not claim to.")

	// MeetsM7 must abstain on a truncated sample, which a run this size produces.
	if parseStats.Truncated && parseStats.MeetsM7() {
		parseT.Error("a truncated sample reported an M7 pass; the observed maximum is only a lower bound")
	}
}

// ----------------------------------------------------------------- pacing

func TestProfilesDifferInTheDirectionTheyClaim(parseT *testing.T) {
	parseResponsive, parseRestoreResponsive, parseErr := gcpacing.Apply(gcpacing.ProfileResponsive, 0)
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	parseRestoreResponsive.Restore()

	parseThroughput, parseRestoreThroughput, parseErr := gcpacing.Apply(gcpacing.ProfileThroughput, 0)
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	parseRestoreThroughput.Restore()

	if parseResponsive.GCPercent >= parseThroughput.GCPercent {
		parseT.Errorf("responsive GCPercent %d is not below throughput's %d; the profiles are backwards",
			parseResponsive.GCPercent, parseThroughput.GCPercent)
	}
}

// TestApplyReturnsWhatItReplaced: pacing is process-global, so a library that
// changed it permanently would be deciding on behalf of an application that may
// have made its own choice.
func TestApplyReturnsWhatItReplaced(parseT *testing.T) {
	parseOriginal := debug.SetGCPercent(137)
	defer debug.SetGCPercent(parseOriginal)

	_, parsePrevious, parseErr := gcpacing.Apply(gcpacing.ProfileResponsive, 0)
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	if parsePrevious.GCPercent != 137 {
		parseT.Errorf("previous GCPercent = %d, want the 137 that was set", parsePrevious.GCPercent)
	}

	parsePrevious.Restore()
	parseAfterRestore := debug.SetGCPercent(137)
	if parseAfterRestore != 137 {
		parseT.Errorf("after Restore, GCPercent = %d, want 137", parseAfterRestore)
	}
}

// TestZeroMemoryLimitMeansOffNotZero is a landmine worth a test.
//
// The runtime spells "no limit" as math.MaxInt64. Passing a literal 0 would set
// an unreachable limit and send the collector into a permanent collect loop.
func TestZeroMemoryLimitMeansOffNotZero(parseT *testing.T) {
	_, parsePrevious, parseErr := gcpacing.Apply(gcpacing.ProfileBalanced, 0)
	if parseErr != nil {
		parseT.Fatalf("Apply: %v", parseErr)
	}
	defer parsePrevious.Restore()

	// Reading the limit back: if 0 had been passed through, this would be 0.
	parseLimit := debug.SetMemoryLimit(-1)
	if parseLimit <= 0 {
		parseT.Errorf("memory limit = %d; a zero limit was passed through instead of being read as off", parseLimit)
	}
}

func TestUnknownProfileIsRejected(parseT *testing.T) {
	if _, _, parseErr := gcpacing.Apply(gcpacing.Profile("aggressive"), 0); parseErr == nil {
		parseT.Error("an unknown profile must be rejected rather than silently applying a default")
	}
}

// ------------------------------------------------------------ pause stats

// TestNoCollectionsDoesNotMeetM7 closes the easiest way to fake this metric.
func TestNoCollectionsDoesNotMeetM7(parseT *testing.T) {
	parseStats := gcpacing.PauseStats{Collections: 0}
	if parseStats.MeetsM7() {
		parseT.Error("a run that never collected says nothing about M7 and must not report a pass")
	}
}

// TestTruncatedSamplesDoNotMeetM7: past 256 collections the observed maximum is
// a LOWER BOUND on the real one, and passing on a lower bound is exactly the
// direction that turns a failing M7 into a green check.
func TestTruncatedSamplesDoNotMeetM7(parseT *testing.T) {
	parseStats := gcpacing.PauseStats{
		Collections: 1000,
		MaxNanos:    100_000, // 0.1 ms — comfortably inside the budget, and unreliable
		Truncated:   true,
	}
	if parseStats.MeetsM7() {
		parseT.Error("a truncated sample cannot support a pass, however good its observed maximum looks")
	}
}

func TestMeetsM7UsesTheMaximumNotTheMedian(parseT *testing.T) {
	// A distribution that is excellent on average and fails on its worst case.
	parseStats := gcpacing.PauseStats{
		Collections: 100,
		MedianNanos: 500_000,   // 0.5 ms
		P99Nanos:    900_000,   // 0.9 ms
		MaxNanos:    9_000_000, // 9 ms — a dropped frame
	}
	if parseStats.MeetsM7() {
		parseT.Error("a 9 ms outlier is a dropped frame; M7 gates on the maximum for exactly this case")
	}
}

func TestReadPausesRejectsAnInvertedWindow(parseT *testing.T) {
	parseBefore := runtime.MemStats{NumGC: 10}
	parseAfter := runtime.MemStats{NumGC: 5}
	if _, parseErr := gcpacing.ReadPauses(parseBefore, parseAfter); parseErr == nil {
		parseT.Error("an after-reading that predates the before-reading must be rejected")
	}
}

func TestReadPausesWithNoCollectionsIsEmpty(parseT *testing.T) {
	parseStats, parseErr := gcpacing.ReadPauses(runtime.MemStats{NumGC: 7}, runtime.MemStats{NumGC: 7})
	if parseErr != nil {
		parseT.Fatalf("ReadPauses: %v", parseErr)
	}
	if parseStats.Collections != 0 || parseStats.MaxNanos != 0 {
		parseT.Errorf("stats = %+v, want empty", parseStats)
	}
}

// TestTruncationIsReported: an understated maximum is the one direction that
// matters, because it makes a failing M7 look like a passing one.
func TestTruncationIsReported(parseT *testing.T) {
	var parseAfter runtime.MemStats
	parseAfter.NumGC = 1000
	for parseIndex := range parseAfter.PauseNs {
		parseAfter.PauseNs[parseIndex] = 1_000_000
	}

	parseStats, parseErr := gcpacing.ReadPauses(runtime.MemStats{NumGC: 0}, parseAfter)
	if parseErr != nil {
		parseT.Fatalf("ReadPauses: %v", parseErr)
	}
	if !parseStats.Truncated {
		parseT.Error("1000 collections exceed the runtime's 256-entry buffer and must be reported as truncated")
	}
	if parseStats.Collections != 1000 {
		parseT.Errorf("collections = %d, want the true count of 1000 even though only 256 pauses survive",
			parseStats.Collections)
	}
}
