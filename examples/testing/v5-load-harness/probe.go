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

	// Cycles that completed INSIDE this window are previous+1 through current.
	//
	// The loop ran previous through current-1, which is wrong at both ends: it
	// included the collection that finished BEFORE the window opened and
	// excluded the last one inside it. The included one is the expensive case —
	// a collection following a gap costs ~6ms where a warm one costs ~0.2ms — so
	// a startup pause was being reported as a pause during a measured window,
	// repeatedly, in whichever window happened to follow it.
	//
	// Go documents the pause for the cycle that produced NumGC == k as living at
	// PauseNs[(k+255)%256], so the index expression is unchanged; only the range
	// was wrong.
	getMaxPauseNs := uint64(0)
	for getCycle := getStartCycle + 1; getCycle <= getCurrentNumGC; getCycle++ {
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
// registerPauseNsAudit exposes window.__gwcV5PauseAudit, which answers whether
// MemStats.PauseNs is a trustworthy measure of main-thread blockage in js/wasm.
//
// # WHY THIS EXISTS
//
// M7 budgets 3ms for the max GC pause and reports 9-12ms. Eight mitigations have
// moved that number by nothing: it is invariant to heap size (11MB vs 1.2MB),
// collection frequency (6 vs 69 per run), pointer density, arena pre-growth,
// live js.Func count, and collector warmup. A quiescent probe on the same machine
// at the same heap sizes pauses 0.3-0.7ms.
//
// Invariance across every input that should matter is the signature of a broken
// instrument, so before optimising against the number, check the number. Go's
// wasm runtime is single-threaded and cooperatively scheduled on the JS event
// loop; if a wall-clock PauseNs window can span a return to that loop, it bills
// unrelated browser work as collection cost.
//
// # THE TEST
//
// Bracket a forced collection with performance.now() and compare the wall-clock
// delta against the PauseNs that Go attributes to that same cycle. Both numbers
// describe the same event, so:
//
//   - wall ~= pauseNs        -> PauseNs is honest; M7's number is real work.
//   - wall << pauseNs        -> PauseNs is inflated; M7 is measuring the wrong
//     thing and must be re-instrumented before anyone
//     optimises against it.
//
// Run under load, because that is the condition under which M7 fails.
func registerPauseNsAudit() {
	js.Global().Set("__gwcV5PauseAudit", js.FuncOf(func(js.Value, []js.Value) any {
		parsePerformance := js.Global().Get("performance")
		var parseBefore, parseAfter goruntime.MemStats

		goruntime.ReadMemStats(&parseBefore)
		parseWallStart := parsePerformance.Call("now").Float()
		goruntime.GC()
		parseWallEnd := parsePerformance.Call("now").Float()
		goruntime.ReadMemStats(&parseAfter)

		// The cycle Go just completed is NumGC; its pause lives at the same index
		// the harness's window logic uses.
		parseCycles := parseAfter.NumGC - parseBefore.NumGC
		parseReportedNs := uint64(0)
		for parseCycle := parseBefore.NumGC + 1; parseCycle <= parseAfter.NumGC; parseCycle++ {
			if parsePause := parseAfter.PauseNs[(parseCycle+255)%256]; parsePause > parseReportedNs {
				parseReportedNs = parsePause
			}
		}

		parseResult := js.Global().Get("Object").New()
		parseResult.Set("wallMs", parseWallEnd-parseWallStart)
		parseResult.Set("reportedPauseMs", float64(parseReportedNs)/1e6)
		parseResult.Set("cycles", parseCycles)
		parseResult.Set("heapAllocMB", float64(parseAfter.HeapAlloc)/(1024*1024))
		return parseResult
	}))
}

// registerPauseTrace dumps EVERY recorded GC pause with its cycle number.
//
// M7 reports a max, which cannot distinguish "collections cost ~10ms" from "one
// specific cycle cost 10ms and the rest were free". Those need opposite fixes,
// and forced collections report 0.1-1.3ms in every condition tested (idle, under
// load, after load) while the run's natural collections report 5.8-13.7ms — so
// the max is coming from somewhere the forced-collection audit cannot reach.
// This exposes the whole ring so the shape is visible instead of inferred.
func registerPauseTrace() {
	js.Global().Set("__gwcV5PauseTrace", js.FuncOf(func(js.Value, []js.Value) any {
		var parseMemStats goruntime.MemStats
		goruntime.ReadMemStats(&parseMemStats)

		parseOut := js.Global().Get("Array").New()
		parseFirst := uint32(1)
		if parseMemStats.NumGC > uint32(len(parseMemStats.PauseNs)) {
			parseFirst = parseMemStats.NumGC - uint32(len(parseMemStats.PauseNs)) + 1
		}
		for parseCycle := parseFirst; parseCycle <= parseMemStats.NumGC; parseCycle++ {
			parseEntry := js.Global().Get("Object").New()
			parseEntry.Set("cycle", int(parseCycle))
			parseEntry.Set("pauseMs", float64(parseMemStats.PauseNs[(parseCycle+255)%256])/1e6)
			parseOut.Call("push", parseEntry)
		}
		return parseOut
	}))
}

func registerV5LoadHarnessProbe() {
	registerPauseNsAudit()
	registerPauseTrace()
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
