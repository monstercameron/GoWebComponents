//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

const (
	parallelRegionGridPulseAtomID = "examples.parallel-region-grid.pulse"
	parallelRegionGridRendererID  = "examples.parallel-region-grid.card"
	parallelRegionGridCardCount   = 24
)

type renderParallelRegionGridProps struct {
	Label      string
	Value      int
	ShardLabel string
}

// buildParallelRegionGridShardLabel resolves the deterministic scheduler shard assignment for one region ID.
func buildParallelRegionGridShardLabel(parseRegionID string) string {
	getShardModel := runtime2.BuildSchedulerShardModel()
	getShardID, parseErr := getShardModel.GetSchedulerRegionShardID(
		parseRegionID,
		[]runtime2.SchedulerShardID{"worker-a", "worker-b", "worker-c", "worker-d"},
		runtime2.SchedulerAssignmentPolicyKeep,
	)
	if parseErr != nil {
		panic(parseErr)
	}
	return string(getShardID)
}

// buildParallelRegionGridSourceIDs binds the shared pulse atom into the public parallel-region source contract.
func buildParallelRegionGridSourceIDs(parsePulse state.Atom[int]) []string {
	getSourceIDs, parseErr := ui.BuildParallelRegionSourceIDs(parsePulse)
	if parseErr != nil {
		panic(parseErr)
	}
	return getSourceIDs
}

// registerParallelRegionGridRenderer registers the stress-card renderer.
func registerParallelRegionGridRenderer() {
	parseErr := ui.RegisterParallelRegion(parallelRegionGridRendererID, renderParallelRegionGridCard)
	if parseErr != nil {
		panic(parseErr)
	}
}

// renderParallelRegionGridCard renders one stress-card surface for the public parallel-region example.
func renderParallelRegionGridCard(parseProps renderParallelRegionGridProps) ui.Node {
	return html.Div(
		html.Props{Class: "rounded-3xl border border-white/10 bg-white/[0.05] p-4 shadow-xl shadow-black/20 backdrop-blur-xl"},
		html.P(
			html.Props{Class: "text-[10px] font-semibold uppercase tracking-[0.24em] text-cyan-100"},
			html.Text(parseProps.ShardLabel),
		),
		html.H3(
			html.Props{Class: "mt-3 text-base font-semibold text-white"},
			html.Text(parseProps.Label),
		),
		html.Div(
			html.Props{Class: "mt-4 text-3xl font-black tracking-tight text-white font-mono"},
			html.Text(fmt.Sprintf("%d", parseProps.Value)),
		),
		html.P(
			html.Props{Class: "mt-3 text-xs leading-6 text-slate-300"},
			html.Text("Local-first shell today, deterministic runtime2 shard assignment now: rerenders dispatch through runtime2 while DOM ownership stays on the main thread."),
		),
	)
}

// buildParallelRegionGridCards renders the full stress grid of parallel regions.
func buildParallelRegionGridCards(parsePulse state.Atom[int]) []ui.Node {
	getSourceIDs := buildParallelRegionGridSourceIDs(parsePulse)
	getCards := make([]ui.Node, 0, parallelRegionGridCardCount)
	for parseIndex := 0; parseIndex < parallelRegionGridCardCount; parseIndex++ {
		getRegionID := fmt.Sprintf("examples.parallel-region-grid.card.%02d", parseIndex)
		getCards = append(getCards, ui.ParallelRegion(ui.ParallelRegionSpec[renderParallelRegionGridProps]{
			RendererID:       parallelRegionGridRendererID,
			RegionInstanceID: getRegionID,
			Props: renderParallelRegionGridProps{
				Label:      fmt.Sprintf("Region %02d", parseIndex+1),
				Value:      parsePulse.Get()*((parseIndex%5)+1) + parseIndex,
				ShardLabel: buildParallelRegionGridShardLabel(getRegionID),
			},
			SourceIDs: getSourceIDs,
		}))
	}
	return getCards
}

// renderParallelRegionGridControls renders the pulse controls that fan updates across the grid.
func renderParallelRegionGridControls() ui.Node {
	parsePulse := state.UseAtom(parallelRegionGridPulseAtomID, 0)
	parseAdvance := ui.UseEvent(func() {
		parsePulse.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})
	parseReset := ui.UseEvent(func() {
		parsePulse.Set(0)
	})

	return html.Div(
		html.Props{Class: "flex flex-wrap items-center gap-3"},
		html.Button(
			html.Props{
				Class:   "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20",
				OnClick: parseAdvance,
			},
			html.Text("Advance Pulse"),
		),
		html.Button(
			html.Props{
				Class:   "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60",
				OnClick: parseReset,
			},
			html.Text("Reset"),
		),
		html.P(
			html.Props{Class: "text-sm text-slate-300"},
			html.Text(fmt.Sprintf("Current pulse: %d", parsePulse.Get())),
		),
	)
}

// renderParallelRegionGridApp renders the stress example shell and region grid.
func renderParallelRegionGridApp() ui.Node {
	parsePulse := state.UseAtom(parallelRegionGridPulseAtomID, 0)
	getCards := buildParallelRegionGridCards(parsePulse)

	return html.Div(
		html.Props{Class: "min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.14),transparent_24%),radial-gradient(circle_at_top_right,rgba(14,165,233,0.10),transparent_28%),linear-gradient(180deg,#020617_0%,#07111f_42%,#0f172a_100%)] px-4 py-10 text-white"},
		html.Div(
			html.Props{Class: "mx-auto max-w-7xl"},
			html.Div(
				html.Props{Class: "rounded-[32px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"},
				html.P(
					html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"},
					html.Text("Stress Example"),
				),
				html.H1(
					html.Props{Class: "mt-4 text-4xl font-black tracking-tight text-white"},
					html.Text("Parallel Region Grid"),
				),
				html.P(
					html.Props{Class: "mt-4 max-w-3xl text-sm leading-7 text-slate-300"},
					html.Text("This page mounts many ui.ParallelRegion(...) instances at once. Each card shows deterministic runtime2 shard assignment so you can inspect dispatch planning across worker shards while the rendered shell remains local-first."),
				),
				html.Div(
					html.Props{Class: "mt-6"},
					ui.CreateElement(renderParallelRegionGridControls),
				),
			),
			html.Div(
				html.Props{Class: "mt-6 grid gap-4 sm:grid-cols-2 xl:grid-cols-4"},
				getCards...,
			),
		),
	)
}

// main registers the stress renderer and mounts the example app.
func main() {
	utils.DisableAllDebug()
	registerParallelRegionGridRenderer()
	ui.Render(ui.CreateElement(renderParallelRegionGridApp), "#app")
	select {}
}
