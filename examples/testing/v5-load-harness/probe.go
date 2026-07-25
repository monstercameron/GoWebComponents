//go:build js && wasm

package main

import (
	"fmt"
	goruntime "runtime"
	"syscall/js"

	gwcruntime "github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// V5 load-harness probe.
//
// The 201 phase probe reports MemStats.PauseTotalNs — a monotonic sum across
// the whole process. That cannot answer M7 ("GC max pause on the render
// thread < 3ms"), because a run with one 9ms pause and a run with thirty 0.3ms
// pauses report the same total while only one of them drops a frame.
//
// This probe reports the MAXIMUM pause observed since the previous call, read
// out of MemStats.PauseNs. That buffer is a 256-entry circular log indexed by
// (NumGC+255)%256 for the most recent collection, so the pauses belonging to
// the window between two calls are exactly the entries for GC cycles
// [previousNumGC, currentNumGC).
//
// Register it from the harness app's main alongside the normal render setup.

// v5ProbeState carries the previous sample so each read can report a windowed
// maximum instead of a cumulative total.
type v5ProbeState struct {
	previousNumGC uint32
	isSeeded      bool
}

// buildWindowMaxPauseNs returns the largest GC pause recorded for cycles in
// [parseState.previousNumGC, parseMemStats.NumGC) and advances the state.
//
// Two correctness details that are easy to get wrong:
//   - PauseNs is circular with 256 slots. If more than 256 collections
//     happened between calls, older entries were overwritten and the true
//     maximum is unknowable; the scan is clamped and the caller is told the
//     sample was truncated so a report can flag it rather than lie.
//   - The first call has no previous sample, so it seeds and reports zero
//     rather than attributing the whole process history to one window.
func (parseState *v5ProbeState) buildWindowMaxPauseNs(parseMemStats *goruntime.MemStats) (uint64, bool) {
	getCurrentNumGC := parseMemStats.NumGC

	if !parseState.isSeeded {
		parseState.isSeeded = true
		parseState.previousNumGC = getCurrentNumGC
		return 0, false
	}

	getStartCycle := parseState.previousNumGC
	parseState.previousNumGC = getCurrentNumGC
	if getCurrentNumGC <= getStartCycle {
		return 0, false
	}

	isTruncated := false
	if getCurrentNumGC-getStartCycle > uint32(len(parseMemStats.PauseNs)) {
		getStartCycle = getCurrentNumGC - uint32(len(parseMemStats.PauseNs))
		isTruncated = true
	}

	getMaxPauseNs := uint64(0)
	for getCycle := getStartCycle; getCycle < getCurrentNumGC; getCycle++ {
		getPauseNs := parseMemStats.PauseNs[(getCycle+255)%256]
		if getPauseNs > getMaxPauseNs {
			getMaxPauseNs = getPauseNs
		}
	}
	return getMaxPauseNs, isTruncated
}

// registerV5LoadHarnessProbe exposes window.__gwcV5Probe for metrics.js.
//
// Diagnostic-only: the harness reads it at window boundaries, never inside a
// measured window, because ReadMemStats stops the world briefly and would
// perturb the very frames being measured.
func registerV5LoadHarnessProbe() {
	parseState := &v5ProbeState{}

	js.Global().Set("__gwcV5Probe", js.FuncOf(func(js.Value, []js.Value) any {
		parseSnapshot := gwcruntime.GetGlobalRuntime().Inspect()
		parseTotals := parseSnapshot.Profiling.PhaseTotals

		var parseMemStats goruntime.MemStats
		goruntime.ReadMemStats(&parseMemStats)
		getWindowMaxPauseNs, isTruncated := parseState.buildWindowMaxPauseNs(&parseMemStats)

		return fmt.Sprintf(
			`{"renderNs":%d,"diffNs":%d,"commitNs":%d,"effectNs":%d,"cleanupNs":%d,`+
				`"processedUnits":%d,"commitCount":%d,"workLoopPasses":%d,`+
				`"numGC":%d,"pauseTotalNs":%d,"windowMaxPauseNs":%d,"pauseSampleTruncated":%t,`+
				`"heapAllocBytes":%d,"nextGCBytes":%d}`,
			parseTotals.RenderDurationNs,
			parseTotals.DiffDurationNs,
			parseTotals.CommitDurationNs,
			parseTotals.EffectDurationNs,
			parseTotals.CleanupDurationNs,
			parseSnapshot.Profiling.ProcessedUnits,
			parseSnapshot.Profiling.CommitCount,
			parseSnapshot.Profiling.WorkLoopPasses,
			parseMemStats.NumGC,
			parseMemStats.PauseTotalNs,
			getWindowMaxPauseNs,
			isTruncated,
			parseMemStats.HeapAlloc,
			parseMemStats.NextGC,
		)
	}))
}
