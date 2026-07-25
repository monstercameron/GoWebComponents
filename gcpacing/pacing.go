// Package gcpacing tunes garbage collection for the render thread
// (plan item P4.4, acceptance M7).
//
// M7 wants the render thread's maximum GC pause under 3 ms; the v4 baseline
// measured 6.60 ms. A pause is a dropped frame, and a dropped frame during
// typing is exactly the symptom v5 exists to remove — so the metric is about
// the MAXIMUM, not the average. An average pause of 1 ms with a 20 ms outlier
// is a worse experience than a steady 2 ms.
//
// The lever is pacing, not a different collector. Go's collector does most of
// its marking concurrently; what stops the world is proportional to the work
// that cannot be done concurrently, which grows with heap size. Collecting more
// often against a smaller heap trades total CPU for shorter individual pauses,
// and on a render thread that trade is the right one: the frame budget cares
// about the longest stop, not the total.
//
// A caveat that has to be stated rather than buried. These knobs are the same
// on every platform, and their DIRECTION of effect is the same, but Go's
// js/wasm runtime is single-threaded and marks without the parallel assist
// native Go has. A pause measured natively is not the browser's pause. This
// package is therefore tested for the direction of its effect; M7's absolute
// number comes from the P0.2 browser harness.
package gcpacing

import (
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"
	"slices"
)

// Profile names a pacing trade-off.
type Profile string

const (
	// ProfileResponsive minimizes the longest pause at the cost of total CPU.
	//
	// The render thread's profile. It collects often against a small heap, so no
	// single collection has much to stop the world for.
	ProfileResponsive Profile = "responsive"
	// ProfileBalanced is Go's default pacing.
	ProfileBalanced Profile = "balanced"
	// ProfileThroughput minimizes total collection CPU at the cost of longer
	// individual pauses. Appropriate for a domain worker, where a pause delays
	// completion rather than a frame.
	ProfileThroughput Profile = "throughput"
)

// gcPercentByProfile is the heap growth ratio that triggers a collection.
//
// Lower means collect sooner, against a smaller heap, for a shorter pause. 40
// rather than something far lower because pacing has a floor: below roughly a
// third of the default, collections become frequent enough that their fixed
// per-cycle cost dominates and the mutator spends its time in write barriers
// instead of doing work — which shows up as slower frames even though each
// pause got shorter.
var gcPercentByProfile = map[Profile]int{
	ProfileResponsive: 40,
	ProfileBalanced:   100,
	ProfileThroughput: 300,
}

// Settings is one profile's applied configuration.
type Settings struct {
	Profile Profile
	// GCPercent is the heap growth ratio that triggers a collection.
	GCPercent int
	// MemoryLimitBytes is a soft ceiling, or 0 for none.
	//
	// Distinct from GCPercent and doing a different job: the ratio paces steady
	// state, and the limit bounds the worst case when a burst of allocation
	// would otherwise let the heap — and with it the pause — grow without a
	// collection intervening.
	MemoryLimitBytes int64
}

// Previous records what was replaced, so a caller can put it back.
//
// Pacing is process-global. A library that changed it permanently would be
// making a decision on behalf of an application that may have made its own, so
// every Apply returns the means to undo it.
type Previous struct {
	GCPercent        int
	MemoryLimitBytes int64
}

// Restore puts back the previous settings.
func (parsePrevious Previous) Restore() {
	debug.SetGCPercent(parsePrevious.GCPercent)
	debug.SetMemoryLimit(parsePrevious.MemoryLimitBytes)
}

// Apply sets pacing for a profile and returns what it replaced.
func Apply(parseProfile Profile, parseMemoryLimitBytes int64) (Settings, Previous, error) {
	parseGCPercent, hasProfile := gcPercentByProfile[parseProfile]
	if !hasProfile {
		return Settings{}, Previous{}, fmt.Errorf("gcpacing: unknown profile %q", parseProfile)
	}

	parsePrevious := Previous{
		// SetGCPercent returns the previous value, and SetMemoryLimit the
		// previous limit, so reading them is the same call that sets them.
		GCPercent:        debug.SetGCPercent(parseGCPercent),
		MemoryLimitBytes: debug.SetMemoryLimit(memoryLimitOrOff(parseMemoryLimitBytes)),
	}

	return Settings{
		Profile:          parseProfile,
		GCPercent:        parseGCPercent,
		MemoryLimitBytes: parseMemoryLimitBytes,
	}, parsePrevious, nil
}

// memoryLimitOrOff converts a zero limit to the runtime's "no limit" value.
//
// math.MaxInt64 is how the runtime spells "off"; passing 0 would set an
// unreachable limit and send the collector into a permanent panic-collect loop.
func memoryLimitOrOff(parseLimitBytes int64) int64 {
	if parseLimitBytes <= 0 {
		const parseNoLimit = int64(1<<63 - 1)
		return parseNoLimit
	}
	return parseLimitBytes
}

// PauseStats summarizes observed stop-the-world pauses.
type PauseStats struct {
	// Collections is how many pauses were observed.
	Collections int
	// MaxNanos is the metric M7 gates on. A single long pause is a dropped
	// frame regardless of how good the average was.
	MaxNanos int64
	// P99Nanos and MedianNanos describe the distribution.
	P99Nanos    int64
	MedianNanos int64
	// Truncated reports that more collections occurred than the runtime's
	// 256-entry pause buffer retains, so the maximum may be understated.
	//
	// Surfaced because an understated maximum is the one direction that matters:
	// it makes a failing M7 look passing.
	Truncated bool
}

// pauseBufferLength is the size of MemStats.PauseNs.
const pauseBufferLength = 256

// ReadPauses reports the pauses that occurred between two MemStats readings.
//
// Taking both readings from the caller rather than sampling here keeps the
// measurement window explicit: a pause that happened before the workload
// started is not the workload's.
func ReadPauses(parseBefore runtime.MemStats, parseAfter runtime.MemStats) (PauseStats, error) {
	if parseAfter.NumGC < parseBefore.NumGC {
		return PauseStats{}, errors.New("gcpacing: the after reading predates the before reading")
	}

	parseCollections := int(parseAfter.NumGC - parseBefore.NumGC)
	if parseCollections == 0 {
		return PauseStats{}, nil
	}

	parseStats := PauseStats{Collections: parseCollections}
	parseWindow := parseCollections
	if parseWindow > pauseBufferLength {
		// Older pauses have been overwritten; only the most recent 256 survive.
		parseWindow = pauseBufferLength
		parseStats.Truncated = true
	}

	parseSamples := make([]int64, 0, parseWindow)
	for parseOffset := range parseWindow {
		parseIndex := (parseAfter.NumGC - uint32(parseOffset) + 255) % 256
		parseSamples = append(parseSamples, int64(parseAfter.PauseNs[parseIndex]))
	}
	slices.Sort(parseSamples)

	parseStats.MaxNanos = parseSamples[len(parseSamples)-1]
	parseStats.MedianNanos = parseSamples[len(parseSamples)/2]
	parseStats.P99Nanos = parseSamples[percentileIndex(len(parseSamples), 0.99)]
	return parseStats, nil
}

// percentileIndex returns the index of a percentile in a sorted sample.
func percentileIndex(parseLength int, parseQuantile float64) int {
	if parseLength <= 1 {
		return 0
	}
	parseIndex := int(parseQuantile * float64(parseLength-1))
	if parseIndex >= parseLength {
		parseIndex = parseLength - 1
	}
	return parseIndex
}

// MaxPauseMillis reports the maximum pause in milliseconds, which is M7's unit.
func (parseStats PauseStats) MaxPauseMillis() float64 {
	return float64(parseStats.MaxNanos) / 1e6
}

// M7BudgetMillis is the maximum render-thread GC pause M7 allows.
const M7BudgetMillis = 3.0

// MeetsM7 reports whether observed pauses are inside M7's budget.
//
// Two ways to answer "no" that are really "cannot say", and both return false
// because a metric that reports a pass it cannot support is worse than one that
// abstains:
//
//   - No collections at all. A workload too small to collect says nothing about
//     M7, and returning true is the easiest way to fake this metric.
//   - A TRUNCATED sample. More than 256 collections means older pauses were
//     overwritten, so the observed maximum is a lower bound on the real one.
//     Passing on a lower bound is exactly the direction that turns a failing M7
//     into a green check.
func (parseStats PauseStats) MeetsM7() bool {
	if parseStats.Collections == 0 || parseStats.Truncated {
		return false
	}
	return parseStats.MaxPauseMillis() <= M7BudgetMillis
}
