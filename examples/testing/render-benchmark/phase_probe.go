//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	goruntime "runtime"
	"syscall/js"

	gwcruntime "github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// registerBenchmarkPhaseProbe exposes the reconciler's cumulative phase totals
// as window.__gwcPhaseTotals so browser benchmark runs can split DOM-ready time
// into render (component functions), diff (reconcile), commit (DOM writes), and
// effect phases. Read it before and after a scenario and subtract; values are
// cumulative nanoseconds since app start. Diagnostic-only: never called inside
// a measured window by the harness.
func registerBenchmarkPhaseProbe() {
	js.Global().Set("__gwcPhaseTotals", js.FuncOf(func(js.Value, []js.Value) any {
		parseSnapshot := gwcruntime.GetGlobalRuntime().Inspect()
		parseTotals := parseSnapshot.Profiling.PhaseTotals
		var parseMemStats goruntime.MemStats
		goruntime.ReadMemStats(&parseMemStats)
		return fmt.Sprintf(
			`{"renderNs":%d,"diffNs":%d,"commitNs":%d,"effectNs":%d,"cleanupNs":%d,"processedUnits":%d,"commitCount":%d,"workLoopPasses":%d,"numGC":%d,"gcPauseNs":%d,"heapAllocBytes":%d}`,
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
			parseMemStats.HeapAlloc,
		)
	}))
}
