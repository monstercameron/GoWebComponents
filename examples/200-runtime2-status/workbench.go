//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	getRuntime2StatusWorkbenchPaneOverview = "overview"
	getRuntime2StatusWorkbenchPaneSignals  = "signals"
	getRuntime2StatusWorkbenchPaneWorkers  = "workers"
)

// renderRuntime2StatusWorkbench renders a nested local-state workbench for the runtime2 example.
func renderRuntime2StatusWorkbench() ui.Node {
	parseCount := state.UseAtom(getRuntime2StatusCounterAtomID, 0)
	parseRenderCount := trackRuntime2StatusRenderCount("runtime2-workbench")
	parseActivePane := ui.UseState(getRuntime2StatusWorkbenchPaneOverview)
	parseStepSize := ui.UseState(1)
	parseSelectedLane := ui.UseState(1)
	parseIsRawOpen := ui.UseState(false)

	parseCountValue := parseCount.Get()
	parseTone := formatRuntime2StatusTone(parseCountValue)
	parseActivePaneValue := parseActivePane.Get()
	parseStepValue := parseStepSize.Get()
	parseSelectedLaneValue := parseSelectedLane.Get()
	parseIsRawOpenValue := parseIsRawOpen.Get()

	parseSetOverview := ui.UseEvent(func() {
		parseActivePane.Set(getRuntime2StatusWorkbenchPaneOverview)
	})
	parseSetSignals := ui.UseEvent(func() {
		parseActivePane.Set(getRuntime2StatusWorkbenchPaneSignals)
	})
	parseSetWorkers := ui.UseEvent(func() {
		parseActivePane.Set(getRuntime2StatusWorkbenchPaneWorkers)
	})
	parseSetStepOne := ui.UseEvent(func() {
		parseStepSize.Set(1)
	})
	parseSetStepTwo := ui.UseEvent(func() {
		parseStepSize.Set(2)
	})
	parseSetStepFive := ui.UseEvent(func() {
		parseStepSize.Set(5)
	})
	parseSetStepEight := ui.UseEvent(func() {
		parseStepSize.Set(8)
	})
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious + parseStepSize.Get()
		})
	})
	parseDecrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious - parseStepSize.Get()
		})
	})
	parseReset := ui.UseEvent(func() {
		parseCount.Set(0)
	})
	parseToggleRaw := ui.UseEvent(func() {
		parseIsRawOpen.Set(!parseIsRawOpen.Get())
	})

	parseLaneButtons := make([]interface{}, 0, getRuntime2StatusWorkerCount)
	for parseLane := 1; parseLane <= getRuntime2StatusWorkerCount; parseLane++ {
		parseCurrentLane := parseLane
		parseLaneButtons = append(parseLaneButtons,
			Button(
				OnClick(ui.UseEvent(func() {
					parseSelectedLane.Set(parseCurrentLane)
				})),
				Class(ClassNames(
					"rounded-xl border px-3 py-2 text-xs font-semibold transition-colors",
					When(parseCurrentLane == parseSelectedLaneValue, "border-cyan-300/40 bg-cyan-400/20 text-cyan-50"),
					When(parseCurrentLane != parseSelectedLaneValue, "border-white/10 bg-white/[0.05] text-slate-300 hover:bg-white/[0.08]"),
				)),
				Textf("%d", parseCurrentLane),
			),
		)
	}
	parseLaneRowNodes := append([]interface{}{
		Class("mt-3 grid grid-cols-4 gap-2 sm:grid-cols-8"),
	}, parseLaneButtons...)

	parsePaneButtonClass := func(parseIsActive bool) string {
		return ClassNames(
			"rounded-2xl border px-4 py-2 text-sm font-semibold transition-colors",
			When(parseIsActive, "border-cyan-300/40 bg-cyan-400/20 text-cyan-50"),
			When(!parseIsActive, "border-white/10 bg-white/[0.05] text-slate-200 hover:bg-white/[0.08]"),
		)
	}
	parseStepButtonClass := func(parseIsActive bool) string {
		return ClassNames(
			"rounded-xl border px-3 py-2 text-xs font-semibold transition-colors",
			When(parseIsActive, "border-emerald-300/40 bg-emerald-400/20 text-emerald-50"),
			When(!parseIsActive, "border-white/10 bg-white/[0.05] text-slate-200 hover:bg-white/[0.08]"),
		)
	}

	return Div(
		Class("rounded-[28px] border border-fuchsia-300/20 bg-fuchsia-400/10 p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.24em] text-fuchsia-100"),
			Text("Runtime2 Workbench"),
		),
		H2(
			Class("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("Nested useState surfaces"),
		),
		P(
			Class("mt-4 text-sm leading-7 text-slate-300"),
			Text("This widget keeps a shared runtime2 counter, a nested pane selector, a step rail, and an eight-lane focus rail in local state so surface churn is easy to inspect."),
		),
		P(
			Class("mt-3 text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
			Textf("Render pass #%d", parseRenderCount),
		),
		Div(
			Class("mt-5 flex flex-wrap gap-2"),
			Button(
				OnClick(parseSetOverview),
				Class(parsePaneButtonClass(parseActivePaneValue == getRuntime2StatusWorkbenchPaneOverview)),
				Text("Overview"),
			),
			Button(
				OnClick(parseSetSignals),
				Class(parsePaneButtonClass(parseActivePaneValue == getRuntime2StatusWorkbenchPaneSignals)),
				Text("Signals"),
			),
			Button(
				OnClick(parseSetWorkers),
				Class(parsePaneButtonClass(parseActivePaneValue == getRuntime2StatusWorkbenchPaneWorkers)),
				Text("Workers"),
			),
		),
		Div(
			Class("mt-4 rounded-2xl border border-white/10 bg-slate-950/45 p-4"),
			P(
				Class("text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
				Text("Step Rail"),
			),
			Div(
				Class("mt-3 flex flex-wrap gap-2"),
				Button(
					OnClick(parseSetStepOne),
					Class(parseStepButtonClass(parseStepValue == 1)),
					Text("x1"),
				),
				Button(
					OnClick(parseSetStepTwo),
					Class(parseStepButtonClass(parseStepValue == 2)),
					Text("x2"),
				),
				Button(
					OnClick(parseSetStepFive),
					Class(parseStepButtonClass(parseStepValue == 5)),
					Text("x5"),
				),
				Button(
					OnClick(parseSetStepEight),
					Class(parseStepButtonClass(parseStepValue == 8)),
					Text("x8"),
				),
				Span(
					Class("ml-1 inline-flex items-center rounded-xl border border-white/10 bg-white/[0.04] px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-300"),
					Textf("current x%d", parseStepValue),
				),
			),
		),
		Div(
			Class("mt-4 flex flex-wrap gap-2"),
			Button(
				OnClick(parseDecrement),
				Class("rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.08]"),
				Textf("-%d", parseStepValue),
			),
			Button(
				OnClick(parseIncrement),
				Class("rounded-2xl border border-fuchsia-300/30 bg-fuchsia-400/15 px-4 py-3 text-sm font-semibold text-fuchsia-50 transition-colors hover:bg-fuchsia-400/20"),
				Textf("+%d", parseStepValue),
			),
			Button(
				OnClick(parseReset),
				Class("rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60"),
				Text("Reset"),
			),
			Button(
				OnClick(parseToggleRaw),
				Class("rounded-2xl border border-white/10 bg-white/[0.05] px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-white/[0.08]"),
				IfElse(parseIsRawOpenValue, Text("Hide raw"), Text("Show raw")),
			),
		),
		Div(
			Class("mt-4 rounded-2xl border border-white/10 bg-slate-950/45 p-4"),
			P(
				Class("text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"),
				Text("Focus Lane"),
			),
			Div(parseLaneRowNodes...),
		),
		Div(
			Class("mt-5"),
			IfElse(parseActivePaneValue == getRuntime2StatusWorkbenchPaneOverview,
				renderRuntime2StatusWorkbenchOverviewSurface(parseCountValue, parseTone, parseStepValue),
				IfElse(parseActivePaneValue == getRuntime2StatusWorkbenchPaneSignals,
					renderRuntime2StatusWorkbenchSignalsSurface(parseCountValue, parseTone, parseStepValue, parseSelectedLaneValue, parseIsRawOpenValue),
					renderRuntime2StatusWorkbenchWorkersSurface(parseSelectedLaneValue),
				),
			),
		),
	)
}

// renderRuntime2StatusWorkbenchOverviewSurface renders the overview surface for the runtime2 workbench.
func renderRuntime2StatusWorkbenchOverviewSurface(parseCount int, parseTone string, parseStep int) ui.Node {
	return Div(
		Class("rounded-[24px] border border-white/10 bg-white/[0.05] p-4"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.22em] text-fuchsia-100"),
			Text("Overview Surface"),
		),
		Div(
			Class("mt-4 grid gap-3 sm:grid-cols-3"),
			Div(
				Class("rounded-2xl border border-cyan-300/20 bg-cyan-400/10 p-4"),
				P(Class("text-[11px] font-semibold uppercase tracking-[0.18em] text-cyan-100"), Text("Shared Count")),
				Div(Class("mt-2 text-4xl font-black tracking-tight text-white font-mono"), Textf("%d", parseCount)),
			),
			Div(
				Class("rounded-2xl border border-emerald-300/20 bg-emerald-400/10 p-4"),
				P(Class("text-[11px] font-semibold uppercase tracking-[0.18em] text-emerald-100"), Text("Tone")),
				Div(Class("mt-2 text-2xl font-black tracking-tight text-white"), Text(parseTone)),
			),
			Div(
				Class("rounded-2xl border border-white/10 bg-slate-950/50 p-4"),
				P(Class("text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400"), Text("Step")),
				Div(Class("mt-2 text-2xl font-black tracking-tight text-white"), Textf("x%d", parseStep)),
			),
		),
		P(
			Class("mt-4 text-sm leading-7 text-slate-300"),
			Text("This is the local-state demo surface: the workbench can mutate the shared counter without touching the runtime2 inspector or the worker pool."),
		),
	)
}

// renderRuntime2StatusWorkbenchSignalsSurface renders the signals surface for the runtime2 workbench.
func renderRuntime2StatusWorkbenchSignalsSurface(parseCount int, parseTone string, parseStep int, parseSelectedLane int, parseIsRawOpen bool) ui.Node {
	return Div(
		Class("rounded-[24px] border border-white/10 bg-white/[0.05] p-4"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.22em] text-sky-100"),
			Text("Signals Surface"),
		),
		Div(
			Class("mt-4 grid gap-3 lg:grid-cols-3"),
			Div(
				Class("rounded-2xl border border-white/10 bg-slate-950/55 p-4"),
				P(Class("text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400"), Text("Pane")),
				Div(Class("mt-2 text-lg font-black tracking-tight text-white"), Text("Local state selector")),
				P(Class("mt-2 text-sm leading-6 text-slate-300"), Text("The active pane is tracked entirely in useState, which makes re-renders easy to force and observe.")),
			),
			Div(
				Class("rounded-2xl border border-white/10 bg-slate-950/55 p-4"),
				P(Class("text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400"), Text("Payload")),
				Div(Class("mt-2 text-lg font-black tracking-tight text-white"), Textf("count %d", parseCount)),
				P(Class("mt-2 text-sm leading-6 text-slate-300"), Textf("step x%d drives the counter delta and keeps the widget intentionally noisy.", parseStep)),
			),
			Div(
				Class("rounded-2xl border border-white/10 bg-slate-950/55 p-4"),
				P(Class("text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400"), Text("Lane")),
				Div(Class("mt-2 text-lg font-black tracking-tight text-white"), Textf("focus %d / %d", parseSelectedLane, getRuntime2StatusWorkerCount)),
				P(Class("mt-2 text-sm leading-6 text-slate-300"), Text("The selected lane is a pure UI focus target, not a worker-affinity change.")),
			),
		),
		IfElse(parseIsRawOpen,
			Pre(
				Class("mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-300"),
				Text(fmt.Sprintf(
					"pane: signals\ncount: %d\ntone: %s\nstep: %d\nfocused lane: %d\nworker pool: %d\nregion: %s\nrenderer: %s",
					parseCount,
					parseTone,
					parseStep,
					parseSelectedLane,
					getRuntime2StatusWorkerCount,
					getRuntime2StatusRegionID,
					getRuntime2StatusRendererID,
				)),
			),
			Div(
				Class("mt-4 rounded-2xl border border-white/10 bg-slate-950/50 p-4"),
				P(
					Class("text-sm leading-7 text-slate-300"),
					Text("Toggle the raw view to expose the local pane, lane, and step state as a compact debug blob."),
				),
			),
		),
	)
}

// renderRuntime2StatusWorkbenchWorkersSurface renders the worker focus surface for the runtime2 workbench.
func renderRuntime2StatusWorkbenchWorkersSurface(parseSelectedLane int) ui.Node {
	parseLaneCards := make([]interface{}, 0, getRuntime2StatusWorkerCount)
	for parseLane := 1; parseLane <= getRuntime2StatusWorkerCount; parseLane++ {
		parseCurrentLane := parseLane
		parseLaneCards = append(parseLaneCards,
			Div(
				Class(ClassNames(
					"rounded-2xl border p-4 transition-colors",
					When(parseCurrentLane == parseSelectedLane, "border-cyan-300/35 bg-cyan-400/15"),
					When(parseCurrentLane != parseSelectedLane, "border-white/10 bg-slate-950/55"),
				)),
				P(
					Class(ClassNames(
						"text-[11px] font-semibold uppercase tracking-[0.18em]",
						When(parseCurrentLane == parseSelectedLane, "text-cyan-100"),
						When(parseCurrentLane != parseSelectedLane, "text-slate-400"),
					)),
					Textf("Lane %d", parseCurrentLane),
				),
				P(
					Class("mt-2 text-sm font-semibold text-white"),
					IfElse(parseCurrentLane == parseSelectedLane, Text("Focused"), Text("Idle")),
				),
				P(
					Class("mt-2 text-xs leading-6 text-slate-300"),
					Text("This rail mirrors the worker pool size without changing the actual runtime2 shard assignment."),
				),
			),
		)
	}
	parseLaneGridNodes := append([]interface{}{
		Class("mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4"),
	}, parseLaneCards...)

	return Div(
		Class("rounded-[24px] border border-white/10 bg-white/[0.05] p-4"),
		P(
			Class("text-xs font-semibold uppercase tracking-[0.22em] text-emerald-100"),
			Text("Workers Surface"),
		),
		P(
			Class("mt-3 text-sm leading-7 text-slate-300"),
			Text("The eight-lane focus rail is intentionally busy so the local-state churn is easy to inspect while the runtime2 worker pool stays separate."),
		),
		Div(parseLaneGridNodes...),
		Div(
			Class("mt-4 rounded-2xl border border-white/10 bg-slate-950/55 p-4"),
			P(Class("text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400"), Text("Focus Summary")),
			P(
				Class("mt-2 text-sm leading-7 text-slate-300"),
				Textf("Focused lane %d of %d with a runtime2 worker pool size of %d.", parseSelectedLane, getRuntime2StatusWorkerCount, getRuntime2StatusWorkerCount),
			),
		),
	)
}
