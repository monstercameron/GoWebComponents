//go:build js && wasm

package main

import (
	"fmt"
	"strings"
	"sync"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	getRuntime2StatusRenderTraceHistoryLimit  = 96
	getRuntime2StatusRenderTraceDisplayLimit  = 12
	getRuntime2StatusRenderTraceVerdictWindow = 24
	getRuntime2StatusRenderLabelApp           = "runtime2-app"
	getRuntime2StatusRenderLabelInspector     = "runtime2-inspector"
	getRuntime2StatusRenderLabelOwnerPanel    = "runtime2-owner-panel"
	getRuntime2StatusRenderLabelRegion        = "runtime2-region"
	getRuntime2StatusRenderLabelTracePanel    = "runtime2-render-trace"
	getRuntime2StatusRenderLabelWorkerFleet   = "runtime2-worker-fleet"
	getRuntime2StatusRenderLabelWorkbench     = "runtime2-workbench"
)

type runtime2StatusRenderTraceProps struct {
	GetOwnerCount     int
	GetAppShellPass   int
	GetOwnerPanelPass int
}

type runtime2StatusRenderTraceSample struct {
	GetTracePanelRenderPass      int
	GetAppShellRenderPass        int
	GetOwnerPanelRenderPass      int
	GetOwnerCount                int
	GetOwnerDelta                int
	GetTracePanelDelta           int
	GetAppShellDelta             int
	GetOwnerPanelDelta           int
	GetInspectorDelta            int
	GetRegionDelta               int
	GetWorkerFleetDelta          int
	GetWorkbenchDelta            int
	GetInspectorRenderPass       int
	GetRegionRenderPass          int
	GetWorkerFleetRenderPass     int
	GetWorkbenchRenderPass       int
	IsOwnerChanged               bool
	IsBackgroundUpdate           bool
	IsSuspiciousAppShellRerender bool
}

type runtime2StatusRenderTraceReport struct {
	GetGraphText            string
	GetSummaryText          string
	GetRowsText             string
	GetVerdictLabel         string
	GetVerdictReason        string
	GetSuspiciousCount      int
	GetBackgroundCount      int
	GetOwnerChangedCount    int
	GetSuspiciousStreakSize int
}

type runtime2StatusRenderTraceState struct {
	GetLabelRenderCountByLabel map[string]int
	GetSampleHistory           []runtime2StatusRenderTraceSample
	GetVerdictLabel            string
	GetVerdictReason           string
}

var (
	getRuntime2StatusRenderTraceMutex = sync.Mutex{}
	getRuntime2StatusRenderTraceStore = runtime2StatusRenderTraceState{
		GetLabelRenderCountByLabel: map[string]int{},
	}
)

// resetRuntime2StatusRenderTraceStore resets all render-trace counters and history for a fresh app boot.
func resetRuntime2StatusRenderTraceStore() {
	getRuntime2StatusRenderTraceMutex.Lock()
	defer getRuntime2StatusRenderTraceMutex.Unlock()
	getRuntime2StatusRenderTraceStore = runtime2StatusRenderTraceState{
		GetLabelRenderCountByLabel: map[string]int{},
	}
}

// storeRuntime2StatusRenderLabelCount stores the latest committed render pass for one runtime2 example surface label.
func storeRuntime2StatusRenderLabelCount(parseLabel string, parseRenderPass int) {
	getRuntime2StatusRenderTraceMutex.Lock()
	defer getRuntime2StatusRenderTraceMutex.Unlock()
	getRuntime2StatusRenderTraceStore.GetLabelRenderCountByLabel[parseLabel] = parseRenderPass
}

// buildRuntime2StatusNextRenderLabelCount increments and returns the next render pass for one runtime2 example surface label.
func buildRuntime2StatusNextRenderLabelCount(parseLabel string) int {
	getRuntime2StatusRenderTraceMutex.Lock()
	defer getRuntime2StatusRenderTraceMutex.Unlock()
	parseNextRenderPass := getRuntime2StatusRenderTraceStore.GetLabelRenderCountByLabel[parseLabel] + 1
	getRuntime2StatusRenderTraceStore.GetLabelRenderCountByLabel[parseLabel] = parseNextRenderPass
	return parseNextRenderPass
}

// readRuntime2StatusRenderLabelCount reads the latest committed render pass for one runtime2 example surface label.
func readRuntime2StatusRenderLabelCount(parseLabel string) int {
	getRuntime2StatusRenderTraceMutex.Lock()
	defer getRuntime2StatusRenderTraceMutex.Unlock()
	return getRuntime2StatusRenderTraceStore.GetLabelRenderCountByLabel[parseLabel]
}

// storeRuntime2StatusRenderTraceSample stores one trace sample and returns the latest computed report.
func storeRuntime2StatusRenderTraceSample(parseOwnerCount int, parseAppShellRenderPass int, parseOwnerPanelRenderPass int, parseTracePanelRenderPass int) runtime2StatusRenderTraceReport {
	getRuntime2StatusRenderTraceMutex.Lock()
	defer getRuntime2StatusRenderTraceMutex.Unlock()

	parseSampleHistory := getRuntime2StatusRenderTraceStore.GetSampleHistory
	if len(parseSampleHistory) > 0 {
		parseLastSample := parseSampleHistory[len(parseSampleHistory)-1]
		if parseLastSample.GetTracePanelRenderPass == parseTracePanelRenderPass {
			return buildRuntime2StatusRenderTraceReportLocked()
		}
	}

	parseLabelRenderCountByLabel := getRuntime2StatusRenderTraceStore.GetLabelRenderCountByLabel
	parseSample := runtime2StatusRenderTraceSample{
		GetTracePanelRenderPass:  parseTracePanelRenderPass,
		GetAppShellRenderPass:    parseAppShellRenderPass,
		GetOwnerPanelRenderPass:  parseOwnerPanelRenderPass,
		GetOwnerCount:            parseOwnerCount,
		GetInspectorRenderPass:   parseLabelRenderCountByLabel[getRuntime2StatusRenderLabelInspector],
		GetRegionRenderPass:      parseLabelRenderCountByLabel[getRuntime2StatusRenderLabelRegion],
		GetWorkerFleetRenderPass: parseLabelRenderCountByLabel[getRuntime2StatusRenderLabelWorkerFleet],
		GetWorkbenchRenderPass:   parseLabelRenderCountByLabel[getRuntime2StatusRenderLabelWorkbench],
	}
	if len(parseSampleHistory) > 0 {
		parsePreviousSample := parseSampleHistory[len(parseSampleHistory)-1]
		parseSample.GetOwnerDelta = parseSample.GetOwnerCount - parsePreviousSample.GetOwnerCount
		parseSample.GetTracePanelDelta = parseSample.GetTracePanelRenderPass - parsePreviousSample.GetTracePanelRenderPass
		parseSample.GetAppShellDelta = parseSample.GetAppShellRenderPass - parsePreviousSample.GetAppShellRenderPass
		parseSample.GetOwnerPanelDelta = parseSample.GetOwnerPanelRenderPass - parsePreviousSample.GetOwnerPanelRenderPass
		parseSample.GetInspectorDelta = parseSample.GetInspectorRenderPass - parsePreviousSample.GetInspectorRenderPass
		parseSample.GetRegionDelta = parseSample.GetRegionRenderPass - parsePreviousSample.GetRegionRenderPass
		parseSample.GetWorkerFleetDelta = parseSample.GetWorkerFleetRenderPass - parsePreviousSample.GetWorkerFleetRenderPass
		parseSample.GetWorkbenchDelta = parseSample.GetWorkbenchRenderPass - parsePreviousSample.GetWorkbenchRenderPass
		parseSample.IsOwnerChanged = parseSample.GetOwnerDelta != 0
		parseSample.IsBackgroundUpdate = !parseSample.IsOwnerChanged &&
			parseSample.GetAppShellDelta == 0 &&
			(parseSample.GetInspectorDelta > 0 ||
				parseSample.GetRegionDelta > 0 ||
				parseSample.GetWorkerFleetDelta > 0 ||
				parseSample.GetWorkbenchDelta > 0 ||
				parseSample.GetOwnerPanelDelta > 0)
		parseSample.IsSuspiciousAppShellRerender = !parseSample.IsOwnerChanged && parseSample.GetAppShellDelta > 0 && !parseSample.IsBackgroundUpdate
	}

	getRuntime2StatusRenderTraceStore.GetSampleHistory = append(getRuntime2StatusRenderTraceStore.GetSampleHistory, parseSample)
	if len(getRuntime2StatusRenderTraceStore.GetSampleHistory) > getRuntime2StatusRenderTraceHistoryLimit {
		getRuntime2StatusRenderTraceStore.GetSampleHistory = append(
			[]runtime2StatusRenderTraceSample(nil),
			getRuntime2StatusRenderTraceStore.GetSampleHistory[len(getRuntime2StatusRenderTraceStore.GetSampleHistory)-getRuntime2StatusRenderTraceHistoryLimit:]...,
		)
	}

	parseReport := buildRuntime2StatusRenderTraceReportLocked()
	if parseReport.GetVerdictLabel != getRuntime2StatusRenderTraceStore.GetVerdictLabel ||
		parseReport.GetVerdictReason != getRuntime2StatusRenderTraceStore.GetVerdictReason {
		fmt.Printf(
			"[runtime2-status/runtime2] render verdict=%s suspicious=%d owner-changes=%d streak=%d reason=%s\n",
			parseReport.GetVerdictLabel,
			parseReport.GetSuspiciousCount,
			parseReport.GetOwnerChangedCount,
			parseReport.GetSuspiciousStreakSize,
			parseReport.GetVerdictReason,
		)
		getRuntime2StatusRenderTraceStore.GetVerdictLabel = parseReport.GetVerdictLabel
		getRuntime2StatusRenderTraceStore.GetVerdictReason = parseReport.GetVerdictReason
	}
	return parseReport
}

// buildRuntime2StatusRenderTraceReportLocked builds the latest graph/report snapshot from the in-memory trace state.
func buildRuntime2StatusRenderTraceReportLocked() runtime2StatusRenderTraceReport {
	parseSampleHistory := getRuntime2StatusRenderTraceStore.GetSampleHistory
	if len(parseSampleHistory) == 0 {
		return runtime2StatusRenderTraceReport{
			GetVerdictLabel:  "PENDING",
			GetVerdictReason: "waiting for first app render sample",
			GetGraphText:     "Graph: waiting for samples",
			GetSummaryText:   "No render trace samples have been recorded yet.",
			GetRowsText:      "Trace: no samples",
		}
	}

	parseWindowStart := 0
	if len(parseSampleHistory) > getRuntime2StatusRenderTraceVerdictWindow {
		parseWindowStart = len(parseSampleHistory) - getRuntime2StatusRenderTraceVerdictWindow
	}
	parseWindow := parseSampleHistory[parseWindowStart:]
	parseSuspiciousCount := 0
	parseBackgroundCount := 0
	parseOwnerChangedCount := 0
	parseSuspiciousStreakSize := 0
	parseStreakSize := 0
	for _, parseSample := range parseWindow {
		if parseSample.IsOwnerChanged {
			parseOwnerChangedCount++
		}
		if parseSample.IsBackgroundUpdate {
			parseBackgroundCount++
		}
		if parseSample.IsSuspiciousAppShellRerender {
			parseSuspiciousCount++
			parseStreakSize++
			if parseStreakSize > parseSuspiciousStreakSize {
				parseSuspiciousStreakSize = parseStreakSize
			}
			continue
		}
		parseStreakSize = 0
	}

	parseVerdictLabel := "GOOD"
	parseVerdictReason := "app-shell rerenders are tied to owner changes or expected background updates"
	if parseSuspiciousStreakSize >= 8 || (parseSuspiciousCount >= 10 && parseOwnerChangedCount == 0) {
		parseVerdictLabel = "BAD"
		parseVerdictReason = "app-shell rerenders keep increasing without owner state changes (likely rerender loop)"
	}

	return runtime2StatusRenderTraceReport{
		GetGraphText:            buildRuntime2StatusRenderTraceGraphText(parseWindow),
		GetSummaryText:          fmt.Sprintf("Window samples: %d | owner changes: %d | background updates: %d | suspicious rerenders: %d | longest suspicious streak: %d", len(parseWindow), parseOwnerChangedCount, parseBackgroundCount, parseSuspiciousCount, parseSuspiciousStreakSize),
		GetRowsText:             buildRuntime2StatusRenderTraceRowsText(parseSampleHistory),
		GetVerdictLabel:         parseVerdictLabel,
		GetVerdictReason:        parseVerdictReason,
		GetSuspiciousCount:      parseSuspiciousCount,
		GetBackgroundCount:      parseBackgroundCount,
		GetOwnerChangedCount:    parseOwnerChangedCount,
		GetSuspiciousStreakSize: parseSuspiciousStreakSize,
	}
}

// buildRuntime2StatusRenderTraceGraphText builds a compact ASCII graph for the recent render trace window.
func buildRuntime2StatusRenderTraceGraphText(parseSampleWindow []runtime2StatusRenderTraceSample) string {
	parseGraph := strings.Builder{}
	parseGraph.WriteString("Graph legend: '+' owner changed, '~' expected background update, 'x' suspicious rerender, '.' stable\n")
	parseGraph.WriteString("Graph: ")
	for _, parseSample := range parseSampleWindow {
		switch {
		case parseSample.IsOwnerChanged:
			parseGraph.WriteByte('+')
		case parseSample.IsBackgroundUpdate:
			parseGraph.WriteByte('~')
		case parseSample.IsSuspiciousAppShellRerender:
			parseGraph.WriteByte('x')
		default:
			parseGraph.WriteByte('.')
		}
	}
	return parseGraph.String()
}

// buildRuntime2StatusRenderTraceRowsText formats the latest trace rows so regressions are easy to inspect.
func buildRuntime2StatusRenderTraceRowsText(parseSampleHistory []runtime2StatusRenderTraceSample) string {
	parseRowBuilder := strings.Builder{}
	parseRowBuilder.WriteString("Trace (latest first):\n")
	parseStart := len(parseSampleHistory) - getRuntime2StatusRenderTraceDisplayLimit
	if parseStart < 0 {
		parseStart = 0
	}
	for parseIndex := len(parseSampleHistory) - 1; parseIndex >= parseStart; parseIndex-- {
		parseSample := parseSampleHistory[parseIndex]
		parseRowBuilder.WriteString(fmt.Sprintf(
			"trace=%d app=%d ownerPanel=%d owner=%d dOwner=%+d dApp=%+d dOwnerPanel=%+d dInspector=%+d dFleet=%+d dRegion=%+d dWorkbench=%+d background=%t suspicious=%t\n",
			parseSample.GetTracePanelRenderPass,
			parseSample.GetAppShellRenderPass,
			parseSample.GetOwnerPanelRenderPass,
			parseSample.GetOwnerCount,
			parseSample.GetOwnerDelta,
			parseSample.GetAppShellDelta,
			parseSample.GetOwnerPanelDelta,
			parseSample.GetInspectorDelta,
			parseSample.GetWorkerFleetDelta,
			parseSample.GetRegionDelta,
			parseSample.GetWorkbenchDelta,
			parseSample.IsBackgroundUpdate,
			parseSample.IsSuspiciousAppShellRerender,
		))
	}
	return strings.TrimSpace(parseRowBuilder.String())
}

// buildRuntime2StatusRenderTraceVerdictClass returns the color class for the current verdict badge.
func buildRuntime2StatusRenderTraceVerdictClass(parseVerdictLabel string) string {
	if parseVerdictLabel == "BAD" {
		return "text-rose-100"
	}
	if parseVerdictLabel == "PENDING" {
		return "text-amber-100"
	}
	return "text-emerald-100"
}

// renderRuntime2StatusRenderTrace renders the render-trace graph and verdict panel for example 200.
func renderRuntime2StatusRenderTrace(parseProps runtime2StatusRenderTraceProps) ui.Node {
	parseTracePanelRenderPass := trackRuntime2StatusRenderCount(getRuntime2StatusRenderLabelTracePanel)
	parseReport := storeRuntime2StatusRenderTraceSample(parseProps.GetOwnerCount, parseProps.GetAppShellPass, parseProps.GetOwnerPanelPass, parseTracePanelRenderPass)
	return Div(
		ClassStr("rounded-[28px] border border-violet-300/20 bg-violet-400/10 p-6 shadow-2xl shadow-black/30 backdrop-blur-xl"),
		P(
			ClassStr("text-xs font-semibold uppercase tracking-[0.24em] text-violet-100"),
			Text("Render Trace"),
		),
		H2(
			ClassStr("mt-4 text-3xl font-black tracking-tight text-white"),
			Text("Rerender health"),
		),
		Div(
			ClassStr("mt-4 rounded-2xl border border-white/10 bg-slate-950/55 p-4"),
			P(
				ClassStr(ClassNames("text-sm font-semibold uppercase tracking-[0.2em]", buildRuntime2StatusRenderTraceVerdictClass(parseReport.GetVerdictLabel))),
				Textf("Verdict: %s", parseReport.GetVerdictLabel),
			),
			P(
				ClassStr("mt-3 text-sm leading-7 text-slate-300"),
				Text(parseReport.GetVerdictReason),
			),
			P(
				ClassStr("mt-3 text-xs leading-6 text-slate-400"),
				Text(parseReport.GetSummaryText),
			),
		),
		Pre(
			ClassStr("mt-5 overflow-x-auto rounded-2xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-200"),
			Text(parseReport.GetGraphText),
		),
		Pre(
			ClassStr("mt-4 overflow-x-auto rounded-2xl border border-white/10 bg-slate-950/60 p-4 text-xs leading-6 text-slate-200"),
			Text(parseReport.GetRowsText),
		),
	)
}
