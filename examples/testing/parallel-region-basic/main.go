//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
)

const (
	parallelRegionBasicCounterAtomID = "examples.parallel-region-basic.count"
	parallelRegionBasicRendererID    = "examples.parallel-region-basic.summary"
	parallelRegionBasicRegionID      = "examples.parallel-region-basic.summary.primary"
)

type renderParallelRegionBasicProps struct {
	Count  int
	Status string
}

// formatParallelRegionBasicStatus derives a compact display status from the current counter value.
func formatParallelRegionBasicStatus(parseCount int) string {
	switch {
	case parseCount == 0:
		return "Centered"
	case parseCount > 0:
		return "Positive drift"
	default:
		return "Negative drift"
	}
}

// buildParallelRegionBasicSourceIDs binds the shared counter atom into the public parallel-region source contract.
func buildParallelRegionBasicSourceIDs(parseCount state.Atom[int]) []string {
	getSourceIDs, parseErr := ui.BuildParallelRegionSourceIDs(parseCount)
	if parseErr != nil {
		panic(parseErr)
	}
	return getSourceIDs
}

// registerParallelRegionBasicRenderer registers the display-only summary renderer for the example.
func registerParallelRegionBasicRenderer() {
	parseErr := ui.RegisterParallelRegion(parallelRegionBasicRendererID, renderParallelRegionBasicSummary)
	if parseErr != nil {
		panic(parseErr)
	}
}

// renderParallelRegionBasicSummary renders the display-only parallel-region body.
func renderParallelRegionBasicSummary(parseProps renderParallelRegionBasicProps) ui.Node {
	return html.Div(
		html.Props{Class: "rounded-3xl border border-cyan-300/20 bg-cyan-400/10 p-6 shadow-xl shadow-cyan-950/20"},
		html.P(
			html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"},
			html.Text("Parallel Region"),
		),
		html.Div(
			html.Props{Class: "mt-4 text-6xl font-black tracking-tight text-white font-mono"},
			html.Text(fmt.Sprintf("%d", parseProps.Count)),
		),
		html.P(
			html.Props{Class: "mt-3 text-sm text-cyan-50/90"},
			html.Text(parseProps.Status),
		),
		html.P(
			html.Props{Class: "mt-5 text-xs leading-6 text-cyan-100/70"},
			html.Text("This region renders through ui.ParallelRegion(...): the shell and visible subtree stay local-first, while prop and declared-source changes still publish runtime2 dispatch state for scheduling, diagnostics, and transport selection."),
		),
	)
}

// renderParallelRegionBasicControls renders the owner-side controls that drive the shared counter atom.
func renderParallelRegionBasicControls() ui.Node {
	parseCount := state.UseAtom(parallelRegionBasicCounterAtomID, 0)
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

	return html.Div(
		html.Props{Class: "flex flex-wrap gap-3"},
		html.Button(
			html.Props{
				Class:   "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]",
				OnClick: parseDecrement,
			},
			html.Text("Decrement"),
		),
		html.Button(
			html.Props{
				Class:   "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20",
				OnClick: parseIncrement,
			},
			html.Text("Increment"),
		),
		html.Button(
			html.Props{
				Class:   "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60",
				OnClick: parseReset,
			},
			html.Text("Reset"),
		),
	)
}

// renderParallelRegionBasicApp renders the example page and mounts the public parallel-region shell.
func renderParallelRegionBasicApp() ui.Node {
	parseCount := state.UseAtom(parallelRegionBasicCounterAtomID, 0)
	return html.Div(
		html.Props{Class: "min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(34,211,238,0.16),transparent_28%),linear-gradient(180deg,#020617_0%,#07111f_44%,#0f172a_100%)] px-4 py-10 text-white"},
		html.Div(
			html.Props{Class: "mx-auto grid max-w-5xl gap-6 lg:grid-cols-[1.05fr_0.95fr]"},
			html.Div(
				html.Props{Class: "rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"},
				html.P(
					html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-cyan-100"},
					html.Text("Agent 1 Example"),
				),
				html.H1(
					html.Props{Class: "mt-4 text-4xl font-black tracking-tight text-white"},
					html.Text("Basic Parallel Region"),
				),
				html.P(
					html.Props{Class: "mt-4 max-w-xl text-sm leading-7 text-slate-300"},
					html.Text("The owner component keeps normal shared state and controls. The hot display surface is registered separately and mounted through ui.ParallelRegion(...), so the shell identity and the renderer ID stay explicit."),
				),
				html.Div(
					html.Props{Class: "mt-8 rounded-3xl border border-white/10 bg-slate-950/40 p-5"},
					html.P(
						html.Props{Class: "text-xs font-semibold uppercase tracking-[0.22em] text-slate-400"},
						html.Text("Owner State"),
					),
					html.P(
						html.Props{Class: "mt-3 text-5xl font-black tracking-tight text-white font-mono"},
						html.Text(fmt.Sprintf("%d", parseCount.Get())),
					),
					html.Div(
						html.Props{Class: "mt-6"},
						ui.CreateElement(renderParallelRegionBasicControls),
					),
				),
			),
			ui.ParallelRegion(ui.ParallelRegionSpec[renderParallelRegionBasicProps]{
				RendererID:       parallelRegionBasicRendererID,
				RegionInstanceID: parallelRegionBasicRegionID,
				Props: renderParallelRegionBasicProps{
					Count:  parseCount.Get(),
					Status: formatParallelRegionBasicStatus(parseCount.Get()),
				},
				SourceIDs: buildParallelRegionBasicSourceIDs(parseCount),
			}),
		),
	)
}

// main registers the example renderer and mounts the example app.
func main() {
	utils.DisableAllDebug()
	registerParallelRegionBasicRenderer()
	ui.Render(ui.CreateElement(renderParallelRegionBasicApp), "#app")
	select {}
}
