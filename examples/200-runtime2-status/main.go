//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	getRuntime2StatusCounterAtomID = "examples.runtime2-status.count"
	getRuntime2StatusRendererID    = "examples.runtime2-status.summary"
	getRuntime2StatusRegionID      = "examples.runtime2-status.summary.primary"
)

type renderRuntime2StatusProps struct {
	Count int
	Tone  string
}

// formatRuntime2StatusTone derives a short status tone from the owner counter value.
func formatRuntime2StatusTone(parseCount int) string {
	switch {
	case parseCount == 0:
		return "Ready"
	case parseCount > 0:
		return "Hot"
	default:
		return "Cooling"
	}
}

// buildRuntime2StatusSourceIDs binds the shared counter atom into the public source contract.
func buildRuntime2StatusSourceIDs(parseCount state.Atom[int]) []string {
	getSourceIDs, parseErr := ui.BuildParallelRegionSourceIDs(parseCount)
	if parseErr != nil {
		panic(parseErr)
	}
	return getSourceIDs
}

// registerRuntime2StatusRenderer registers the display-only summary renderer for the example.
func registerRuntime2StatusRenderer() {
	parseErr := ui.RegisterParallelRegion(getRuntime2StatusRendererID, renderRuntime2StatusSummary)
	if parseErr != nil {
		panic(parseErr)
	}
}

// renderRuntime2StatusSummary renders the display-only parallel-region body.
func renderRuntime2StatusSummary(parseProps renderRuntime2StatusProps) ui.Node {
	return Div(
		Class("rounded-3xl border border-cyan-300/20 bg-cyan-400/10 p-6 shadow-xl shadow-cyan-950/20"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"),
			Text("Runtime2"),
		),
		Div(
			Class("mt-4 text-6xl font-black tracking-tight text-white font-mono"),
			Textf("%d", parseProps.Count),
		),
		P(
			Class("mt-3 text-sm text-cyan-50/90"),
			Text(parseProps.Tone),
		),
		P(
			Class("mt-5 text-xs leading-6 text-cyan-100/70"),
			Text("This 200-series example keeps the region shell local-first while the inspector surface exercises the public read-only runtime2 status helper."),
		),
	)
}

// renderRuntime2StatusReadout renders the inspector panel that surfaces the public runtime2 status helper.
func renderRuntime2StatusReadout(parseRegionID string) ui.Node {
	// This call intentionally targets the public read-only runtime2 status helper for the active region.
	getRuntimeStatus, hasRuntimeStatus, parseRuntimeStatusErr := ui.GetParallelRegionRuntimeStatus(parseRegionID)
	if parseRuntimeStatusErr != nil {
		return Div(
			Class("rounded-[28px] border border-rose-300/25 bg-rose-400/10 p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
			P(
				Class("text-xs font-semibold uppercase tracking-[0.24em] text-rose-100"),
				Text("Inspector"),
			),
			H2(
				Class("mt-4 text-3xl font-black tracking-tight text-white"),
				Text("Status lookup failed"),
			),
			P(
				Class("mt-4 text-sm leading-7 text-slate-300"),
				Text(parseRuntimeStatusErr.Error()),
			),
		)
	}
	if !hasRuntimeStatus {
		return Div(
			Class("rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
			P(
				Class("text-xs font-semibold uppercase tracking-[0.24em] text-amber-100"),
				Text("Inspector"),
			),
			H2(
				Class("mt-4 text-3xl font-black tracking-tight text-white"),
				Text("Waiting on public status"),
			),
			P(
				Class("mt-4 text-sm leading-7 text-slate-300"),
				Text("The public helper did not return a snapshot for this region yet, so the inspector stays empty until the region is tracked."),
			),
			P(
				Class("mt-4 text-xs leading-6 text-slate-400"),
				Text("The example stays intentionally narrow so the public runtime-status contract is easy to inspect while the shell remains local-first."),
			),
		)
	}

	return Div(
		Class("rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.24em] text-emerald-100"),
			Text("Inspector"),
		),
		H2(
			Class("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("Read-only runtime status"),
		),
		P(
			Class("mt-4 text-sm leading-7 text-slate-300"),
			Text("This panel mirrors the public status snapshot that tooling can consume without reaching into runtime2 internals."),
		),
		Pre(
			Class("mt-6 overflow-x-auto rounded-2xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-300"),
			Text(fmt.Sprintf(
				"Region: %s\nMode: %s\nShard: %s\nRenderer: %s\nEpoch: %d\nHydration complete: %t\nHydrated shell anchor: %t\nPost-hydration attached: %t\nSnapshot version: %d\nDispatched version: %d\nCommitted version: %d\nTransport tier: %s\nDropped stale patches: %d\nIgnored stale diagnostics: %d\nFallback reason: %s",
				getRuntimeStatus.GetRegionInstanceID,
				getRuntimeStatus.GetRegionMode,
				getRuntimeStatus.GetAssignedWorkerShard,
				getRuntimeStatus.GetRendererID,
				getRuntimeStatus.GetEpoch,
				getRuntimeStatus.GetIsHydrationComplete,
				getRuntimeStatus.HasHydratedShellAnchor,
				getRuntimeStatus.HasPostHydrationAttached,
				getRuntimeStatus.GetLastSnapshotVersion,
				getRuntimeStatus.GetLastDispatchedVersion,
				getRuntimeStatus.GetLastCommittedVersion,
				getRuntimeStatus.GetTransportTier,
				getRuntimeStatus.GetDroppedStalePatchCount,
				getRuntimeStatus.GetIgnoredStaleDiagnosticCount,
				getRuntimeStatus.GetFallbackReason,
			)),
		),
	)
}

// renderRuntime2StatusControls renders the owner-side controls that drive the shared counter atom.
func renderRuntime2StatusControls() ui.Node {
	parseCount := state.UseAtom(getRuntime2StatusCounterAtomID, 0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})
	parseDecrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious - 1
		})
	})
	parseReset := ui.UseEvent(func() {
		parseCount.Set(0)
	})

	return Div(
		Class("flex flex-wrap gap-3"),
		Button(
			OnClick(parseDecrement),
			Class("rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]"),
			Text("Decrement"),
		),
		Button(
			OnClick(parseIncrement),
			Class("rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20"),
			Text("Increment"),
		),
		Button(
			OnClick(parseReset),
			Class("rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60"),
			Text("Reset"),
		),
	)
}

// renderRuntime2StatusApp renders the example page and mounts the public parallel-region shell.
func renderRuntime2StatusApp() ui.Node {
	parseCount := state.UseAtom(getRuntime2StatusCounterAtomID, 0)
	return Div(
		Class("min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.16),transparent_28%),linear-gradient(180deg,#020617_0%,#07111f_44%,#0f172a_100%)] px-4 py-10 text-white"),
		Div(
			Class("mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1.02fr_0.98fr]"),
			Div(
				Class("rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
				P(
					Class("text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"),
					Text("200-Series"),
				),
				H1(
					Class("mt-4 text-4xl font-black tracking-tight text-white"),
					Text("Runtime2 Status"),
				),
				P(
					Class("mt-4 max-w-xl text-sm leading-7 text-slate-300"),
					Text("This page demonstrates the public read-only runtime status helper. The region stays local-first today, but the inspector still exposes the observable contract: ownership mode, shard plan, versions, hydration flags, and fallback state."),
				),
				Div(
					Class("mt-8 rounded-3xl border border-white/10 bg-slate-950/40 p-5"),
					P(
						Class("text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
						Text("Owner State"),
					),
					P(
						Class("mt-3 text-5xl font-black tracking-tight text-white font-mono"),
						Textf("%d", parseCount.Get()),
					),
					Div(
						Class("mt-6"),
						ui.CreateElement(renderRuntime2StatusControls),
					),
				),
			),
			Div(
				Class("grid gap-6"),
				ui.ParallelRegion(ui.ParallelRegionSpec[renderRuntime2StatusProps]{
					RendererID:       getRuntime2StatusRendererID,
					RegionInstanceID: getRuntime2StatusRegionID,
					Props: renderRuntime2StatusProps{
						Count: parseCount.Get(),
						Tone:  formatRuntime2StatusTone(parseCount.Get()),
					},
					SourceIDs: buildRuntime2StatusSourceIDs(parseCount),
				}),
				ui.CreateElement(renderRuntime2StatusReadout, getRuntime2StatusRegionID),
			),
		),
	)
}

// main registers the example renderer and mounts the example app.
func main() {
	utils.DisableAllDebug()
	registerRuntime2StatusRenderer()
	ui.Render(ui.CreateElement(renderRuntime2StatusApp), "#app")
	select {}
}
