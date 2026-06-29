//go:build js && wasm

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

// renderRuntime2StatusSummary renders the display-only parallel-region body.
func renderRuntime2StatusSummary(parseProps renderRuntime2StatusProps) ui.Node {
	parseRenderCount := trackRuntime2StatusRenderCount("runtime2-region")
	return Div(
		ClassStr("rounded-3xl border border-cyan-300/20 bg-cyan-400/10 p-6 shadow-xl shadow-cyan-950/20"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"),
			Text("Runtime2"),
		),
		Div(
			ClassStr("mt-4 text-6xl font-black tracking-tight text-white font-mono"),
			Textf("%d", parseProps.Count),
		),
		P(
			ClassStr("mt-3 text-sm text-cyan-50/90"),
			Text(parseProps.Tone),
		),
		P(
			ClassStr("mt-3 text-xs font-semibold uppercase tracking-[0.22em] text-cyan-100/80"),
			Textf("Render pass #%d", parseRenderCount),
		),
		P(
			ClassStr("mt-5 text-xs leading-6 text-cyan-100/70"),
			Text("This 200-series example keeps the region shell local-first while the inspector surface exercises the public read-only runtime2 status helper."),
		),
	)
}

// renderRuntime2StatusNoticeList renders the current operator-facing caveats directly in the example UI.
func renderRuntime2StatusNoticeList(parseNoticeTexts []string) ui.Node {
	return If(len(parseNoticeTexts) > 0,
		Div(
			ClassStr("mt-5 rounded-2xl border border-amber-300/20 bg-amber-400/10 p-4"),
			P(
				ClassStr("text-xs font-semibold uppercase tracking-[0.22em] text-amber-100"),
				Text("Warnings"),
			),
			Map(parseNoticeTexts, func(parseNoticeText string) ui.Node {
				return P(
					ClassStr("mt-3 text-sm leading-7 text-amber-50/90"),
					Text(parseNoticeText),
				)
			}),
		),
	)
}

// renderRuntime2StatusReadout renders the inspector panel that surfaces the public runtime2 status helper.
func renderRuntime2StatusReadout(parseRegionID string) ui.Node {
	parseInspectorRefreshState := ui.UseState(0)
	storeWarningByMessageRef := ui.UseRef(map[string]bool{})
	parseRenderCount := trackRuntime2StatusRenderCount("runtime2-inspector")
	parseInspectorRefreshRevision := parseInspectorRefreshState.Get()
	// This call intentionally targets the public read-only runtime2 status helper for the active region.
	getRuntimeStatus, hasRuntimeStatus, parseRuntimeStatusErr := ui.GetParallelRegionRuntimeStatus(parseRegionID)
	getNoticeTexts := buildRuntime2StatusNoticeTexts(hasRuntimeStatus, getRuntimeStatus)
	getRouteSummaryText := buildRuntime2StatusRouteSummaryText(hasRuntimeStatus, getRuntimeStatus, parseRuntimeStatusErr)
	handleRuntime2StatusNoticeEffect(parseRegionID, getNoticeTexts, storeWarningByMessageRef)
	handleRuntime2StatusRuntimeProofEffect(parseRegionID, getRouteSummaryText)
	ui.UseEffect(func() func() {
		fmt.Printf("[runtime2-status/runtime2] inspector refresh region=%s revision=%d\n", parseRegionID, parseInspectorRefreshRevision)
		return nil
	}, parseRegionID, parseInspectorRefreshRevision)
	parseForceRefresh := ui.UseEvent(func() {
		parseInspectorRefreshState.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})
	getRefreshButton := Button(
		OnClick(parseForceRefresh),
		ClassStr("rounded-xl border border-white/10 bg-white/[0.05] px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-200 transition-colors hover:bg-white/[0.08]"),
		Text("Refresh runtime2 status"),
	)
	getRenderPassBadge := P(
		ClassStr("mt-4 text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
		Textf("Render pass #%d", parseRenderCount),
	)
	getRuntimeProofNode := Div(
		ClassStr("rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.22em] text-cyan-100"),
			Text("Runtime2 Proof"),
		),
		Pre(
			ClassStr("mt-3 overflow-x-auto rounded-xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-300"),
			Text(getRouteSummaryText),
		),
	)
	getErrorNode := Div(
		ClassStr("rounded-[28px] border border-rose-300/25 bg-rose-400/10 p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-rose-100"),
			Text("Runtime2 Inspector"),
		),
		H2(
			ClassStr("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("Runtime2 status lookup failed"),
		),
		P(
			ClassStr("mt-4 text-sm leading-7 text-slate-300"),
			Text(fmt.Sprint(parseRuntimeStatusErr)),
		),
		Div(
			ClassStr("mt-4 flex flex-wrap gap-3"),
			getRefreshButton,
		),
		getRenderPassBadge,
		getRuntimeProofNode,
	)
	getWaitingNode := Div(
		ClassStr("rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-amber-100"),
			Text("Runtime2 Inspector"),
		),
		H2(
			ClassStr("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("Waiting on public runtime2 status"),
		),
		P(
			ClassStr("mt-4 text-sm leading-7 text-slate-300"),
			Text("The public helper did not return a snapshot for this region yet. This runtime2 inspector updates when the page rerenders or when you press Refresh runtime2 status."),
		),
		P(
			ClassStr("mt-4 text-xs leading-6 text-slate-400"),
			Text("The example stays intentionally narrow so the public runtime2 contract is easy to inspect while the shell remains local-first and read-only."),
		),
		Div(
			ClassStr("mt-4 flex flex-wrap gap-3"),
			getRefreshButton,
		),
		getRenderPassBadge,
		getRuntimeProofNode,
	)
	getInspectorNodes := []interface{}{
		ClassStr("rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-emerald-100"),
			Text("Runtime2 Inspector"),
		),
		H2(
			ClassStr("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("Read-only runtime2 status"),
		),
		P(
			ClassStr("mt-4 text-sm leading-7 text-slate-300"),
			Text("This panel mirrors the public runtime2 status snapshot that tooling can consume without reaching into runtime2 internals. It refreshes on page rerenders and on manual refresh."),
		),
		Div(
			ClassStr("mt-4 flex flex-wrap gap-3"),
			getRefreshButton,
		),
		getRenderPassBadge,
		getRuntimeProofNode,
		Pre(
			ClassStr("mt-6 overflow-x-auto rounded-2xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-300"),
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
		renderRuntime2StatusNoticeList(getNoticeTexts),
	}
	return IfElse(
		parseRuntimeStatusErr != nil,
		getErrorNode,
		IfElse(!hasRuntimeStatus, getWaitingNode, Div(getInspectorNodes...)),
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
		parseCount.Update(func(parsePrevious int) int {
			if parsePrevious == 0 {
				return parsePrevious
			}
			return 0
		})
	})

	return Div(
		ClassStr("flex flex-wrap gap-3"),
		Button(
			OnClick(parseDecrement),
			ClassStr("rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]"),
			Text("Decrement"),
		),
		Button(
			OnClick(parseIncrement),
			ClassStr("rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20"),
			Text("Increment"),
		),
		Button(
			OnClick(parseReset),
			ClassStr("rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60"),
			Text("Reset"),
		),
	)
}

// renderRuntime2StatusOwnerPanel renders the owner counter readout and controls.
func renderRuntime2StatusOwnerPanel() ui.Node {
	parseCount := state.UseAtom(getRuntime2StatusCounterAtomID, 0)
	parseOwnerCount := parseCount.Get()
	parsePanelRenderCount := trackRuntime2StatusRenderCount(getRuntime2StatusRenderLabelOwnerPanel)
	parseAppRenderCount := readRuntime2StatusRenderLabelCount(getRuntime2StatusRenderLabelApp)
	if parseAppRenderCount < 1 {
		parseAppRenderCount = 1
	}
	return Div(
		ClassStr("mt-8 rounded-3xl border border-white/10 bg-slate-950/40 p-5"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
			Text("Owner State"),
		),
		P(
			ClassStr("mt-3 text-5xl font-black tracking-tight text-white font-mono"),
			Textf("%d", parseOwnerCount),
		),
		P(
			ClassStr("mt-2 text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
			Textf("App committed render pass #%d", parseAppRenderCount),
		),
		P(
			ClassStr("mt-2 text-xs font-semibold uppercase tracking-[0.22em] text-slate-500"),
			Textf("Owner panel render pass #%d", parsePanelRenderCount),
		),
		Div(
			ClassStr("mt-6"),
			ui.CreateElement(renderRuntime2StatusControls),
		),
	)
}

// renderRuntime2StatusParallelRegionPanel renders the runtime2-backed parallel region with atom-driven props.
func renderRuntime2StatusParallelRegionPanel() ui.Node {
	parseCount := state.UseAtom(getRuntime2StatusCounterAtomID, 0)
	parseOwnerCount := parseCount.Get()
	return ui.ParallelRegion(ui.ParallelRegionSpec[renderRuntime2StatusProps]{
		RendererID:       getRuntime2StatusRendererID,
		RegionInstanceID: getRuntime2StatusRegionID,
		Props: renderRuntime2StatusProps{
			Count: parseOwnerCount,
			Tone:  formatRuntime2StatusTone(parseOwnerCount),
		},
		SourceIDs: buildRuntime2StatusCounterSourceIDs(),
	})
}

// renderRuntime2StatusWorkerFleetPanel renders the worker fleet with atom-driven count updates.
func renderRuntime2StatusWorkerFleetPanel() ui.Node {
	parseCount := state.UseAtom(getRuntime2StatusCounterAtomID, 0)
	return ui.CreateElement(renderRuntime2StatusWorkerFleet, runtime2StatusWorkerFleetProps{
		RegionID: getRuntime2StatusRegionID,
		Count:    parseCount.Get(),
	})
}

// renderRuntime2StatusRenderTracePanel renders the rerender-trace panel with owner count and app-shell pass context.
func renderRuntime2StatusRenderTracePanel() ui.Node {
	parseCount := state.UseAtom(getRuntime2StatusCounterAtomID, 0)
	return ui.CreateElement(renderRuntime2StatusRenderTrace, runtime2StatusRenderTraceProps{
		GetOwnerCount:     parseCount.Get(),
		GetAppShellPass:   readRuntime2StatusRenderLabelCount(getRuntime2StatusRenderLabelApp),
		GetOwnerPanelPass: readRuntime2StatusRenderLabelCount(getRuntime2StatusRenderLabelOwnerPanel),
	})
}

// renderRuntime2StatusApp renders the example page and mounts the public parallel-region shell.
func renderRuntime2StatusApp() ui.Node {
	parseCounterSource := buildRuntime2StatusCounterReactiveSource()
	parseShellRenderCount := trackRuntime2StatusRenderCount(getRuntime2StatusRenderLabelApp)
	return Div(
		ClassStr("min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.16),transparent_28%),linear-gradient(180deg,#020617_0%,#07111f_44%,#0f172a_100%)] px-4 py-10 text-white"),
		Div(
			ClassStr("mx-auto grid max-w-6xl gap-6 lg:grid-cols-[1.02fr_0.98fr]"),
			Div(
				ClassStr("rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
				P(
					ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"),
					Text("200-Series"),
				),
				H1(
					ClassStr("mt-4 text-4xl font-black tracking-tight text-white"),
					Text("Runtime2 Status"),
				),
				P(
					ClassStr("mt-4 max-w-xl text-sm leading-7 text-slate-300"),
					Text("This page demonstrates the public read-only runtime2 status helper. The region is mounted through the runtime2-backed ui.ParallelRegion(...) path, the inspector exposes the current observable contract honestly, the workbench below exercises nested useState surfaces, and the main runtime2 WASM now fans out status probes to an 8-worker Go WASM pool."),
				),
				P(
					ClassStr("mt-3 text-xs font-semibold uppercase tracking-[0.22em] text-slate-500"),
					Textf("App shell render pass #%d", parseShellRenderCount),
				),
				// Keep high-churn owner surfaces inside fine-grained regions so owner atom updates do not rerender the app shell.
				ui.ReactiveRegion(func() ui.Node {
					return ui.CreateElement(renderRuntime2StatusOwnerPanel)
				}, parseCounterSource),
				Div(
					ClassStr("mt-6"),
					ui.ReactiveRegion(func() ui.Node {
						return ui.CreateElement(renderRuntime2StatusWorkbench)
					}, parseCounterSource),
				),
			),
			Div(
				ClassStr("grid gap-6"),
				ui.ReactiveRegion(func() ui.Node {
					return ui.CreateElement(renderRuntime2StatusParallelRegionPanel)
				}, parseCounterSource),
				ui.CreateElement(renderRuntime2StatusReadout, getRuntime2StatusRegionID),
				ui.ReactiveRegion(func() ui.Node {
					return ui.CreateElement(renderRuntime2StatusWorkerFleetPanel)
				}, parseCounterSource),
				ui.ReactiveRegion(func() ui.Node {
					return ui.CreateElement(renderRuntime2StatusRenderTracePanel)
				}, parseCounterSource),
			),
		),
	)
}

// main registers the example renderer and mounts the example app.
func main() {
	utils.DisableAllDebug()
	resetRuntime2StatusRenderTraceStore()
	resetRuntime2StatusWorkerTraceCounter()
	registerRuntime2StatusRenderer()
	fmt.Printf("[runtime2-status/runtime2] main client boot region=%s renderer=%s workers=%d\n", getRuntime2StatusRegionID, getRuntime2StatusRendererID, getRuntime2StatusWorkerCount)
	ui.Render(ui.CreateElement(renderRuntime2StatusApp), "#app")
	select {}
}
