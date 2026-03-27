//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

const (
	parallelRegionDiagnosticsCounterAtomID = "examples.parallel-region-diagnostics.count"
	parallelRegionDiagnosticsRendererID    = "examples.parallel-region-diagnostics.summary"
	parallelRegionDiagnosticsRegionID      = "examples.parallel-region-diagnostics.summary.primary"
)

type renderParallelRegionDiagnosticsProps struct {
	Count  int
	Status string
}

// formatParallelRegionDiagnosticsStatus derives the current summary tone from the owner count.
func formatParallelRegionDiagnosticsStatus(parseCount int) string {
	switch {
	case parseCount == 0:
		return "Idle"
	case parseCount > 0:
		return "Rising"
	default:
		return "Negative"
	}
}

// buildParallelRegionDiagnosticsSourceIDs binds the shared count atom into the public source contract.
func buildParallelRegionDiagnosticsSourceIDs(parseCount state.Atom[int]) []string {
	getSourceIDs, parseErr := ui.BuildParallelRegionSourceIDs(parseCount)
	if parseErr != nil {
		panic(parseErr)
	}
	return getSourceIDs
}

// registerParallelRegionDiagnosticsRenderer registers the diagnostics-region renderer.
func registerParallelRegionDiagnosticsRenderer() {
	parseErr := ui.RegisterParallelRegion(parallelRegionDiagnosticsRendererID, renderParallelRegionDiagnosticsSummary)
	if parseErr != nil {
		panic(parseErr)
	}
}

// renderParallelRegionDiagnosticsSummary renders the display-only diagnostics-region summary card.
func renderParallelRegionDiagnosticsSummary(parseProps renderParallelRegionDiagnosticsProps) ui.Node {
	return html.Div(
		html.Props{Class: "rounded-3xl border border-amber-300/20 bg-amber-400/10 p-6 shadow-xl shadow-amber-950/20"},
		html.P(
			html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-amber-100"},
			html.Text("Parallel Region"),
		),
		html.Div(
			html.Props{Class: "mt-4 text-6xl font-black tracking-tight text-white font-mono"},
			html.Text(fmt.Sprintf("%d", parseProps.Count)),
		),
		html.P(
			html.Props{Class: "mt-3 text-sm text-amber-50/90"},
			html.Text(parseProps.Status),
		),
		html.P(
			html.Props{Class: "mt-5 text-xs leading-6 text-amber-100/70"},
			html.Text("Use the diagnostics panel to simulate runtime2 fallback, transport downgrade, and worker restart transitions while the shell stays locally rendered."),
		),
	)
}

// buildParallelRegionDiagnosticsSpec converts the current owner state into one runtime2 region spec.
func buildParallelRegionDiagnosticsSpec(parseCount state.Atom[int]) runtime2.ParallelRegionSpec {
	return runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID(parallelRegionDiagnosticsRendererID),
		RegionInstanceID: runtime2.RegionInstanceID(parallelRegionDiagnosticsRegionID),
		Props: renderParallelRegionDiagnosticsProps{
			Count:  parseCount.Get(),
			Status: formatParallelRegionDiagnosticsStatus(parseCount.Get()),
		},
		SourceIDs: buildParallelRegionDiagnosticsSourceIDs(parseCount),
	}
}

// buildParallelRegionDiagnosticsAdapter creates and mounts one host adapter for diagnostics simulation.
func buildParallelRegionDiagnosticsAdapter(parseSpec runtime2.ParallelRegionSpec) *runtime2.HostRegionAdapter {
	getAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		parseSpec.RegionInstanceID,
		[]runtime2.SchedulerShardID{"worker-a", "worker-b"},
	)
	if parseErr != nil {
		panic(parseErr)
	}
	if _, parseMountErr := getAdapter.HandleHostRegionMount(parseSpec, 1); parseMountErr != nil {
		panic(parseMountErr)
	}
	return getAdapter
}

// buildParallelRegionDiagnosticsEnvelope builds one snapshot envelope for downgrade diagnostics.
func buildParallelRegionDiagnosticsEnvelope(parseSpec runtime2.ParallelRegionSpec, parseInputVersion uint64) runtime2.SnapshotEnvelope {
	getEnvelope, parseErr := runtime2.BuildSnapshotEnvelope(
		parseSpec.RegionInstanceID,
		1,
		parseInputVersion,
		parseSpec.Props,
		parseSpec.SourceIDs,
		map[string]any{
			parallelRegionDiagnosticsCounterAtomID: parseInputVersion,
		},
		map[string]uint64{
			parallelRegionDiagnosticsCounterAtomID: parseInputVersion,
		},
	)
	if parseErr != nil {
		panic(parseErr)
	}
	return getEnvelope
}

// storeParallelRegionDiagnosticsLog prepends one diagnostics line into the visible log.
func storeParallelRegionDiagnosticsLog(parseLogs ui.State[[]string], parseLine string) {
	parseLogs.Update(func(parsePrevious []string) []string {
		getNext := make([]string, 0, len(parsePrevious)+1)
		getNext = append(getNext, parseLine)
		getNext = append(getNext, parsePrevious...)
		if len(getNext) > 8 {
			getNext = getNext[:8]
		}
		return getNext
	})
}

// renderParallelRegionDiagnosticsPanel renders the diagnostics controls and status readout.
func renderParallelRegionDiagnosticsPanel() ui.Node {
	parseCount := state.UseAtom(parallelRegionDiagnosticsCounterAtomID, 0)
	parseLogs := ui.UseState([]string{"ready: diagnostics panel mounted"})
	parseAdapterRef := ui.UseRef[*runtime2.HostRegionAdapter](nil)

	getSpec := buildParallelRegionDiagnosticsSpec(parseCount)
	if parseAdapterRef.Get() == nil {
		parseAdapterRef.Set(buildParallelRegionDiagnosticsAdapter(getSpec))
	}

	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int {
			return parsePrevious + 1
		})
	})
	parseReset := ui.UseEvent(func() {
		parseCount.Set(0)
		parseLogs.Set([]string{"ready: diagnostics reset"})
		parseAdapterRef.Set(buildParallelRegionDiagnosticsAdapter(getSpec))
	})
	parseSimulateDowngrade := ui.UseEvent(func() {
		getEnvelope := buildParallelRegionDiagnosticsEnvelope(getSpec, uint64(parseCount.Get()+1))
		getResult, parseErr := runtime2.BuildSharedSnapshotTransportResult(
			getEnvelope,
			runtime2.CapabilityReport{
				HasWorkerSupport:                true,
				HasStructuredCloneSupport:       true,
				HasSharedBufferSupport:          true,
				HasSharedMemoryTransportSupport: true,
			},
			nil,
		)
		if parseErr != nil {
			storeParallelRegionDiagnosticsLog(parseLogs, "downgrade: error "+parseErr.Error())
			return
		}
		storeParallelRegionDiagnosticsLog(
			parseLogs,
			fmt.Sprintf("downgrade: tier=%s reason=%s", getResult.GetTransportTier, getResult.GetDowngradeReason),
		)
	})
	parseSimulateFallback := ui.UseEvent(func() {
		getAdapter := parseAdapterRef.Get()
		if getAdapter == nil {
			storeParallelRegionDiagnosticsLog(parseLogs, "fallback: adapter missing")
			return
		}
		parseInputVersion := uint64(parseCount.Get() + 1)
		if parseErr := getAdapter.HandleHostRegionBinaryDecodeFailure(parseInputVersion); parseErr != nil {
			storeParallelRegionDiagnosticsLog(parseLogs, "fallback: error "+parseErr.Error())
			return
		}
		getMirror, parseMirrorErr := getAdapter.HandleHostRegionFallbackOwnershipBegin()
		if parseMirrorErr != nil {
			storeParallelRegionDiagnosticsLog(parseLogs, "fallback: mirror error "+parseMirrorErr.Error())
			return
		}
		storeParallelRegionDiagnosticsLog(
			parseLogs,
			fmt.Sprintf("fallback: coordinator=%t scheduler=%t", getMirror.HasCoordinatorFallback, getMirror.HasSchedulerFallback),
		)
	})
	parseSimulateRestart := ui.UseEvent(func() {
		getAdapter := parseAdapterRef.Get()
		if getAdapter == nil {
			storeParallelRegionDiagnosticsLog(parseLogs, "restart: adapter missing")
			return
		}
		getWorkerDeath, parseWorkerDeathErr := getAdapter.HandleHostRegionWorkerDeath(true)
		if parseWorkerDeathErr != nil {
			storeParallelRegionDiagnosticsLog(parseLogs, "restart: worker death error "+parseWorkerDeathErr.Error())
			return
		}
		getRepair, parseRepairErr := getAdapter.HandleHostRegionRepairRemount(getSpec)
		if parseRepairErr != nil {
			storeParallelRegionDiagnosticsLog(parseLogs, "restart: repair error "+parseRepairErr.Error())
			return
		}
		storeParallelRegionDiagnosticsLog(
			parseLogs,
			fmt.Sprintf(
				"restart: reassigned=%t remount_epoch=%d cleared=%t version_floor=%d",
				getWorkerDeath.HasReassigned,
				getRepair.GetRemountEpoch,
				getRepair.HasFallbackCleared,
				getRepair.GetVersionFloor,
			),
		)
	})

	getAdapter := parseAdapterRef.Get()
	getFallbackActive := false
	getRepairPending := false
	if getAdapter != nil {
		getFallbackActive = getAdapter.GetHostRegionIsFallbackActive()
		getRepairPending = getAdapter.GetHostRegionIsRepairPending()
	}

	getLogNodes := make([]ui.Node, 0, len(parseLogs.Get()))
	for _, getLine := range parseLogs.Get() {
		getLogNodes = append(getLogNodes, html.Li(
			html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-xs leading-6 text-slate-300"},
			html.Text(getLine),
		))
	}

	return html.Div(
		html.Props{Class: "rounded-[28px] border border-white/10 bg-white/[0.05] p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"},
		html.P(
			html.Props{Class: "text-xs font-semibold uppercase tracking-[0.24em] text-amber-100"},
			html.Text("Diagnostics"),
		),
		html.H2(
			html.Props{Class: "mt-4 text-3xl font-black tracking-tight text-white"},
			html.Text("Fallback, Downgrade, Restart"),
		),
		html.P(
			html.Props{Class: "mt-4 text-sm leading-7 text-slate-300"},
			html.Text("The buttons below exercise runtime2 diagnostics directly. The parallel-region shell on the left stays locally rendered while the panel simulates downgrade, fallback, and worker restart transitions."),
		),
		html.Div(
			html.Props{Class: "mt-6 flex flex-wrap gap-3"},
			html.Button(
				html.Props{Class: "rounded-2xl border border-white/10 bg-white/[0.06] px-4 py-3 text-sm font-semibold text-white transition-colors hover:bg-white/[0.10]", OnClick: parseIncrement},
				html.Text("Increment Region"),
			),
			html.Button(
				html.Props{Class: "rounded-2xl border border-cyan-300/30 bg-cyan-400/15 px-4 py-3 text-sm font-semibold text-cyan-50 transition-colors hover:bg-cyan-400/20", OnClick: parseSimulateDowngrade},
				html.Text("Simulate Downgrade"),
			),
			html.Button(
				html.Props{Class: "rounded-2xl border border-amber-300/30 bg-amber-400/15 px-4 py-3 text-sm font-semibold text-amber-50 transition-colors hover:bg-amber-400/20", OnClick: parseSimulateFallback},
				html.Text("Simulate Fallback"),
			),
			html.Button(
				html.Props{Class: "rounded-2xl border border-emerald-300/30 bg-emerald-400/15 px-4 py-3 text-sm font-semibold text-emerald-50 transition-colors hover:bg-emerald-400/20", OnClick: parseSimulateRestart},
				html.Text("Simulate Restart"),
			),
			html.Button(
				html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-3 text-sm font-semibold text-slate-200 transition-colors hover:bg-slate-900/60", OnClick: parseReset},
				html.Text("Reset"),
			),
		),
		html.Div(
			html.Props{Class: "mt-6 grid gap-3 sm:grid-cols-2"},
			html.Div(
				html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-4"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text("Fallback Active")),
				html.P(html.Props{Class: "mt-2 text-lg font-semibold text-white"}, html.Text(fmt.Sprintf("%t", getFallbackActive))),
			),
			html.Div(
				html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/40 px-4 py-4"},
				html.P(html.Props{Class: "text-xs uppercase tracking-[0.22em] text-slate-400"}, html.Text("Repair Pending")),
				html.P(html.Props{Class: "mt-2 text-lg font-semibold text-white"}, html.Text(fmt.Sprintf("%t", getRepairPending))),
			),
		),
		html.Ul(
			html.Props{Class: "mt-6 space-y-3"},
			getLogNodes...,
		),
	)
}

// renderParallelRegionDiagnosticsApp renders the diagnostics example surface.
func renderParallelRegionDiagnosticsApp() ui.Node {
	parseCount := state.UseAtom(parallelRegionDiagnosticsCounterAtomID, 0)

	return html.Div(
		html.Props{Class: "min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(251,191,36,0.16),transparent_28%),radial-gradient(circle_at_top_right,rgba(34,211,238,0.12),transparent_24%),linear-gradient(180deg,#020617_0%,#07111f_44%,#0f172a_100%)] px-4 py-10 text-white"},
		html.Div(
			html.Props{Class: "mx-auto grid max-w-6xl gap-6 lg:grid-cols-[0.92fr_1.08fr]"},
			ui.ParallelRegion(ui.ParallelRegionSpec[renderParallelRegionDiagnosticsProps]{
				RendererID:       parallelRegionDiagnosticsRendererID,
				RegionInstanceID: parallelRegionDiagnosticsRegionID,
				Props: renderParallelRegionDiagnosticsProps{
					Count:  parseCount.Get(),
					Status: formatParallelRegionDiagnosticsStatus(parseCount.Get()),
				},
				SourceIDs: buildParallelRegionDiagnosticsSourceIDs(parseCount),
			}),
			ui.CreateElement(renderParallelRegionDiagnosticsPanel),
		),
	)
}

// main registers the diagnostics renderer and mounts the example app.
func main() {
	utils.DisableAllDebug()
	registerParallelRegionDiagnosticsRenderer()
	ui.Render(ui.CreateElement(renderParallelRegionDiagnosticsApp), "#app")
	select {}
}
