//go:build js && wasm
// +build js,wasm

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/monstercameron/GoWebComponents/v5/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v5/examples/shared"
	"github.com/monstercameron/GoWebComponents/v5/html"
	"github.com/monstercameron/GoWebComponents/v5/interop"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	"github.com/monstercameron/GoWebComponents/v5/utils"
)

const traceStorageKey = "catalog-use-state-rerender-trace"

type traceField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type traceTrigger struct {
	Kind      string       `json:"kind"`
	StateKey  string       `json:"stateKey"`
	Operation string       `json:"operation"`
	Note      string       `json:"note"`
	Before    []traceField `json:"before"`
	Target    []traceField `json:"target"`
}

type traceSpan struct {
	ID         string       `json:"id"`
	ParentID   string       `json:"parentId,omitempty"`
	Phase      string       `json:"phase"`
	Depth      int          `json:"depth"`
	Name       string       `json:"name"`
	Signature  string       `json:"signature"`
	Params     []traceField `json:"params,omitempty"`
	Result     string       `json:"result,omitempty"`
	StartedAt  string       `json:"startedAt"`
	DurationNs int64        `json:"durationNs"`

	getStartedAtTime time.Time
}

type traceSummary struct {
	SpanCount        int   `json:"spanCount"`
	MaxDepth         int   `json:"maxDepth"`
	TotalDurationNs  int64 `json:"totalDurationNs"`
	EventDurationNs  int64 `json:"eventDurationNs"`
	StateDurationNs  int64 `json:"stateDurationNs"`
	RenderDurationNs int64 `json:"renderDurationNs"`
}

type traceDocument struct {
	Version    int          `json:"version"`
	TraceID    string       `json:"traceId"`
	StartedAt  string       `json:"startedAt"`
	FinishedAt string       `json:"finishedAt,omitempty"`
	Trigger    traceTrigger `json:"trigger"`
	StateAfter []traceField `json:"stateAfter,omitempty"`
	Spans      []traceSpan  `json:"spans"`
	Summary    traceSummary `json:"summary"`

	getStartedAtTime time.Time
}

type traceState struct {
	getNextTraceID int
	getActive      traceDocument
	hasActive      bool
	getLast        traceDocument
	getLastVersion int
	getSpanStack   []string
	getSpanIndex   map[string]int
}

type traceViewModel struct {
	Counter       int
	CounterLabel  string
	CounterBand   string
	CounterParity string
	Label         string
	DeltaLabel    string
}

var storeTraceState traceState

// renderTraceExample renders the example 203 surface and the latest rerender trace document.
func renderTraceExample() ui.Node {
	getCounter := ui.UseState(2)
	getLabel := ui.UseState("Trace the next ui.UseState rerender.")
	getCopyStatus := ui.UseState("Copy idle.")
	getPersistStatus := ui.UseState("Stored trace document status is idle.")
	getRefreshTick := ui.UseState(0)

	ui.UseEffect(func() func() {
		// Load the previous trace artifact once so a refresh still leaves a reviewable document behind.
		getDocument, hasDocument, getErr := loadTraceDocument()
		if getErr != nil {
			getPersistStatus.Set("Trace document load failed: " + getErr.Error())
			return nil
		}
		if !hasDocument {
			getPersistStatus.Set("No stored trace document found yet. Trigger one rerender to generate it.")
			return nil
		}
		storeTraceLastDocument(getDocument)
		getRefreshTick.Update(func(getPrevious int) int { return getPrevious + 1 })
		getPersistStatus.Set("Loaded the last stored trace document from LocalStorage.")
		return nil
	}, "trace-document-load")

	getIncrement := ui.UseEvent(func() {
		getPrevious := getCounter.Get()
		getTarget := buildTraceIncrementCounter(getPrevious)
		startTraceCycle(traceTrigger{
			Kind:      "ui.UseState",
			StateKey:  "counter",
			Operation: "Update",
			Note:      "Counter increment schedules one rerender through ui.UseState.Update(...).",
			Before:    buildTraceStateFields(getPrevious, getLabel.Get()),
			Target:    buildTraceStateFields(getTarget, getLabel.Get()),
		})
		getHandlerSpanID := startTraceSpan("event", "handleTraceIncrement", "handleTraceIncrement() ui.Handler", []traceField{
			{Name: "counter_before", Value: strconv.Itoa(getPrevious)},
			{Name: "counter_target", Value: strconv.Itoa(getTarget)},
		})
		getCounter.Update(func(getPreviousCounter int) int {
			getUpdateSpanID := startTraceSpan("state", "buildTraceIncrementCounter", "buildTraceIncrementCounter(parsePrevious int) int", []traceField{
				{Name: "parsePrevious", Value: strconv.Itoa(getPreviousCounter)},
			})
			getNextCounter := buildTraceIncrementCounter(getPreviousCounter)
			finishTraceSpan(getUpdateSpanID, strconv.Itoa(getNextCounter))
			return getNextCounter
		})
		finishTraceSpan(getHandlerSpanID, "counter rerender scheduled")
	})

	getRotateLabel := ui.UseEvent(func() {
		getPrevious := getLabel.Get()
		getTarget := buildTraceNextLabel(getCounter.Get(), getPrevious)
		startTraceCycle(traceTrigger{
			Kind:      "ui.UseState",
			StateKey:  "label",
			Operation: "Set",
			Note:      "Label rotation schedules one rerender through ui.UseState.Set(...).",
			Before:    buildTraceStateFields(getCounter.Get(), getPrevious),
			Target:    buildTraceStateFields(getCounter.Get(), getTarget),
		})
		getHandlerSpanID := startTraceSpan("event", "handleTraceRotateLabel", "handleTraceRotateLabel() ui.Handler", []traceField{
			{Name: "label_before", Value: getPrevious},
			{Name: "label_target", Value: getTarget},
		})
		getSetSpanID := startTraceSpan("state", "buildTraceNextLabel", "buildTraceNextLabel(parseCounter int, parseCurrent string) string", []traceField{
			{Name: "parseCounter", Value: strconv.Itoa(getCounter.Get())},
			{Name: "parseCurrent", Value: getPrevious},
		})
		getLabel.Set(getTarget)
		finishTraceSpan(getSetSpanID, getTarget)
		finishTraceSpan(getHandlerSpanID, "label rerender scheduled")
	})

	getReset := ui.UseEvent(func() {
		startTraceCycle(traceTrigger{
			Kind:      "ui.UseState",
			StateKey:  "counter,label",
			Operation: "Set",
			Note:      "Reset restores both state cells to the baseline demo values.",
			Before:    buildTraceStateFields(getCounter.Get(), getLabel.Get()),
			Target:    buildTraceStateFields(2, "Trace the next ui.UseState rerender."),
		})
		getHandlerSpanID := startTraceSpan("event", "handleTraceReset", "handleTraceReset() ui.Handler", []traceField{
			{Name: "counter_before", Value: strconv.Itoa(getCounter.Get())},
			{Name: "label_before", Value: getLabel.Get()},
		})
		getCounter.Set(2)
		getLabel.Set("Trace the next ui.UseState rerender.")
		finishTraceSpan(getHandlerSpanID, "reset rerender scheduled")
	})

	getCopy := ui.UseEvent(func() {
		getDocument := buildTraceLastDocument()
		if strings.TrimSpace(getDocument.TraceID) == "" {
			getCopyStatus.Set("No trace document is available yet.")
			return
		}
		if getErr := copyTraceDocument(getDocument); getErr != nil {
			getCopyStatus.Set("Trace document copy failed: " + getErr.Error())
			return
		}
		getCopyStatus.Set("Copied the latest trace document JSON to the clipboard.")
	})

	getReload := ui.UseEvent(func() {
		getDocument, hasDocument, getErr := loadTraceDocument()
		if getErr != nil {
			getPersistStatus.Set("Trace document reload failed: " + getErr.Error())
			return
		}
		if !hasDocument {
			getPersistStatus.Set("Trace document reload found no stored artifact.")
			return
		}
		storeTraceLastDocument(getDocument)
		getRefreshTick.Update(func(getPrevious int) int { return getPrevious + 1 })
		getPersistStatus.Set("Reloaded the trace document from LocalStorage.")
	})

	hasStartedTrace := hasTraceCycleActive()
	getRootSpanID := ""
	if hasStartedTrace {
		getRootSpanID = startTraceSpan("render", "renderTraceExample", "renderTraceExample() ui.Node", buildTraceStateFields(getCounter.Get(), getLabel.Get()))
	}

	// Keep the traced subtree narrow so the exported document is focused on one owned rerender path.
	getViewModel := buildTraceViewModel(getCounter.Get(), getLabel.Get())
	getDemoNode := renderTraceDemoPanel(getViewModel, getIncrement, getRotateLabel, getReset)

	if getRootSpanID != "" {
		finishTraceSpan(getRootSpanID, "ui.Node(example-page)")
	}
	if hasStartedTrace {
		finishTraceCycle(buildTraceStateFields(getCounter.Get(), getLabel.Get()))
	}

	getDocument := buildTraceLastDocument()
	getDocumentJSON := formatTraceDocumentJSON(getDocument)
	getTraceVersion := buildTraceLastVersion()

	ui.UseEffect(func() func() {
		if getTraceVersion == 0 || strings.TrimSpace(getDocument.TraceID) == "" {
			return nil
		}
		if getErr := storeTraceDocument(getDocument); getErr != nil {
			getPersistStatus.Set("Trace document store failed: " + getErr.Error())
			return nil
		}
		getPersistStatus.Set("Stored trace " + getDocument.TraceID + " in LocalStorage for later review.")
		return nil
	}, getTraceVersion)

	return shared.ExamplePage(
		"ui.UseState rerender trace document",
		"app-owned rerender tracing for ui.UseState",
		"Trigger one local ui.UseState update, trace the event and render call chain, then review an exported JSON document with function signatures, parameter summaries, nesting depth, and timing data.",
		getDemoNode,
		renderTraceDocumentPanel(getDocument, getDocumentJSON, getCopyStatus.Get(), getPersistStatus.Get(), getCopy, getReload),
	)
}

// buildTraceIncrementCounter returns the next counter value for the increment action.
func buildTraceIncrementCounter(parsePrevious int) int {
	return parsePrevious + 1
}

// buildTraceNextLabel rotates the demo headline so a Set-based rerender has distinct payload data.
func buildTraceNextLabel(parseCounter int, parseCurrent string) string {
	getOptions := []string{
		"Trace the next ui.UseState rerender.",
		"Review the call graph before optimizing.",
		"Export the timing document for later analysis.",
	}
	for getIndex, getOption := range getOptions {
		if getOption != parseCurrent {
			if (parseCounter+getIndex)%2 == 0 {
				return getOption
			}
		}
	}
	return getOptions[(parseCounter+1)%len(getOptions)]
}

// buildTraceViewModel derives the small render model used by the demo panel.
func buildTraceViewModel(parseCounter int, parseLabel string) traceViewModel {
	getSpanID := startTraceSpan("render", "buildTraceViewModel", "buildTraceViewModel(parseCounter int, parseLabel string) traceViewModel", []traceField{
		{Name: "parseCounter", Value: strconv.Itoa(parseCounter)},
		{Name: "parseLabel", Value: parseLabel},
	})
	getViewModel := traceViewModel{
		Counter:       parseCounter,
		CounterLabel:  buildTraceCounterLabel(parseCounter),
		CounterBand:   buildTraceCounterBand(parseCounter),
		CounterParity: buildTraceCounterParity(parseCounter),
		Label:         parseLabel,
		DeltaLabel:    buildTraceDeltaLabel(parseCounter),
	}
	finishTraceSpan(getSpanID, fmt.Sprintf("traceViewModel{counter=%d, band=%s}", getViewModel.Counter, getViewModel.CounterBand))
	return getViewModel
}

// buildTraceCounterLabel formats the primary counter label shown in the traced demo panel.
func buildTraceCounterLabel(parseCounter int) string {
	getSpanID := startTraceSpan("render", "buildTraceCounterLabel", "buildTraceCounterLabel(parseCounter int) string", []traceField{
		{Name: "parseCounter", Value: strconv.Itoa(parseCounter)},
	})
	getLabel := fmt.Sprintf("Counter %02d", parseCounter)
	finishTraceSpan(getSpanID, getLabel)
	return getLabel
}

// buildTraceCounterBand classifies the counter into a stable review band.
func buildTraceCounterBand(parseCounter int) string {
	getSpanID := startTraceSpan("render", "buildTraceCounterBand", "buildTraceCounterBand(parseCounter int) string", []traceField{
		{Name: "parseCounter", Value: strconv.Itoa(parseCounter)},
	})
	getBand := "steady"
	if parseCounter >= 6 {
		getBand = "stress"
	} else if parseCounter >= 3 {
		getBand = "warm"
	}
	finishTraceSpan(getSpanID, getBand)
	return getBand
}

// buildTraceCounterParity returns the readable parity label for the current counter.
func buildTraceCounterParity(parseCounter int) string {
	getSpanID := startTraceSpan("render", "buildTraceCounterParity", "buildTraceCounterParity(parseCounter int) string", []traceField{
		{Name: "parseCounter", Value: strconv.Itoa(parseCounter)},
	})
	getParity := "odd"
	if parseCounter%2 == 0 {
		getParity = "even"
	}
	finishTraceSpan(getSpanID, getParity)
	return getParity
}

// buildTraceDeltaLabel explains the next expected delta from the current counter.
func buildTraceDeltaLabel(parseCounter int) string {
	getSpanID := startTraceSpan("render", "buildTraceDeltaLabel", "buildTraceDeltaLabel(parseCounter int) string", []traceField{
		{Name: "parseCounter", Value: strconv.Itoa(parseCounter)},
	})
	getLabel := fmt.Sprintf("Next increment lands on %d.", parseCounter+1)
	finishTraceSpan(getSpanID, getLabel)
	return getLabel
}

// renderTraceDemoPanel renders the stateful demo surface whose rerenders are traced.
func renderTraceDemoPanel(parseViewModel traceViewModel, parseIncrement ui.Handler, parseRotateLabel ui.Handler, parseReset ui.Handler) ui.Node {
	getSpanID := startTraceSpan("render", "renderTraceDemoPanel", "renderTraceDemoPanel(parseViewModel traceViewModel, parseIncrement ui.Handler, parseRotateLabel ui.Handler, parseReset ui.Handler) ui.Node", []traceField{
		{Name: "counter", Value: strconv.Itoa(parseViewModel.Counter)},
		{Name: "label", Value: parseViewModel.Label},
	})
	getNode := shared.ExamplePanel(
		"Trace the rerender",
		html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("Use one button at a time. Each action records the event work, the state transition helper, and the owned render helper chain for the rerender that follows.")),
		html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
			shared.ExampleButton("Increment counter", parseIncrement),
			shared.ExampleButton("Rotate headline", parseRotateLabel),
			shared.ExampleButton("Reset demo", parseReset),
		),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
			renderTraceMetricCard("Counter", strconv.Itoa(parseViewModel.Counter), parseViewModel.CounterLabel),
			renderTraceMetricCard("Band", parseViewModel.CounterBand, parseViewModel.DeltaLabel),
			renderTraceMetricCard("Parity", parseViewModel.CounterParity, "Derived during render."),
			renderTraceMetricCard("Headline", parseViewModel.Label, "Set-based rerender target."),
		),
	)
	finishTraceSpan(getSpanID, "ui.Node(trace-demo-panel)")
	return getNode
}

// renderTraceMetricCard renders one narrow stat card inside the traced demo panel.
func renderTraceMetricCard(parseLabel string, parseValue string, parseDetail string) ui.Node {
	getSpanID := startTraceSpan("render", "renderTraceMetricCard", "renderTraceMetricCard(parseLabel string, parseValue string, parseDetail string) ui.Node", []traceField{
		{Name: "parseLabel", Value: parseLabel},
		{Name: "parseValue", Value: parseValue},
	})
	getNode := html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/60 p-4"},
		html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text(parseValue)),
		html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-300"}, html.Text(parseDetail)),
	)
	finishTraceSpan(getSpanID, "ui.Node(metric-card)")
	return getNode
}

// renderTraceDocumentPanel renders the review surface for the latest exported trace document.
func renderTraceDocumentPanel(parseDocument traceDocument, parseDocumentJSON string, parseCopyStatus string, parsePersistStatus string, parseCopy ui.Handler, parseReload ui.Handler) ui.Node {
	getSummaryNode := renderTraceSummaryPanel(parseDocument, parseCopyStatus, parsePersistStatus, parseCopy, parseReload)
	getTriggerNode := renderTraceTriggerPanel(parseDocument)
	getSpanNode := renderTraceSpanPanel(parseDocument)
	getJSONNode := renderTraceJSONPanel(parseDocumentJSON)
	return html.Div(html.Props{Class: "grid gap-6"},
		getSummaryNode,
		getTriggerNode,
		getSpanNode,
		getJSONNode,
	)
}

// renderTraceSummaryPanel renders the top-level trace summary and export controls.
func renderTraceSummaryPanel(parseDocument traceDocument, parseCopyStatus string, parsePersistStatus string, parseCopy ui.Handler, parseReload ui.Handler) ui.Node {
	getTraceID := parseDocument.TraceID
	if strings.TrimSpace(getTraceID) == "" {
		getTraceID = "not-captured-yet"
	}
	return shared.ExamplePanel(
		"Trace summary",
		html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
			shared.ExampleButton("Copy JSON", parseCopy),
			shared.ExampleButton("Reload stored trace", parseReload),
		),
		html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-4"},
			shared.ExampleStat("Trace ID", getTraceID),
			shared.ExampleStat("Spans", strconv.Itoa(parseDocument.Summary.SpanCount)),
			shared.ExampleStat("Render time", formatTraceDuration(parseDocument.Summary.RenderDurationNs)),
			shared.ExampleStat("Max depth", strconv.Itoa(parseDocument.Summary.MaxDepth)),
		),
		html.P(html.Props{Class: "mt-6 text-slate-300"}, html.Text(parseCopyStatus)),
		html.P(html.Props{Class: "mt-2 text-sm leading-6 text-slate-400"}, html.Text(parsePersistStatus)),
	)
}

// renderTraceTriggerPanel renders the trigger metadata that started the current trace cycle.
func renderTraceTriggerPanel(parseDocument traceDocument) ui.Node {
	getRows := []ui.Node{
		renderTraceFieldRow("Kind", parseDocument.Trigger.Kind),
		renderTraceFieldRow("State key", parseDocument.Trigger.StateKey),
		renderTraceFieldRow("Operation", parseDocument.Trigger.Operation),
		renderTraceFieldRow("Note", parseDocument.Trigger.Note),
		renderTraceFieldRow("Before", formatTraceFields(parseDocument.Trigger.Before)),
		renderTraceFieldRow("Target", formatTraceFields(parseDocument.Trigger.Target)),
		renderTraceFieldRow("After", formatTraceFields(parseDocument.StateAfter)),
	}
	return shared.ExamplePanel(
		"Trigger and state snapshots",
		html.Div(html.Props{Class: "mt-3 grid gap-3"}, getRows...),
	)
}

// renderTraceSpanPanel renders the ordered call table for the captured trace spans.
func renderTraceSpanPanel(parseDocument traceDocument) ui.Node {
	if len(parseDocument.Spans) == 0 {
		return shared.ExamplePanel(
			"Call stack",
			html.P(html.Props{Class: "mt-3 text-slate-300"}, html.Text("No trace spans captured yet. Trigger one rerender to populate the table.")),
		)
	}
	getRows := make([]ui.Node, 0, len(parseDocument.Spans)+1)
	getRows = append(getRows,
		html.Tr(html.Props{Class: "text-left text-xs uppercase tracking-[0.2em] text-slate-400"},
			html.Th(html.Props{Class: "px-3 py-2 font-semibold"}, html.Text("Depth")),
			html.Th(html.Props{Class: "px-3 py-2 font-semibold"}, html.Text("Phase")),
			html.Th(html.Props{Class: "px-3 py-2 font-semibold"}, html.Text("Function")),
			html.Th(html.Props{Class: "px-3 py-2 font-semibold"}, html.Text("Params")),
			html.Th(html.Props{Class: "px-3 py-2 font-semibold"}, html.Text("Duration")),
			html.Th(html.Props{Class: "px-3 py-2 font-semibold"}, html.Text("Result")),
		),
	)
	for _, getSpan := range parseDocument.Spans {
		getRows = append(getRows, renderTraceSpanRow(getSpan))
	}
	return shared.ExamplePanel(
		"Call stack",
		html.Div(html.Props{Class: "mt-3 overflow-x-auto rounded-2xl border border-white/10 bg-black/30"},
			html.Table(html.Props{Class: "min-w-full border-collapse text-sm text-slate-200"},
				html.Tbody(html.Props{}, getRows...),
			),
		),
	)
}

// renderTraceSpanRow renders one ordered span row inside the captured call stack table.
func renderTraceSpanRow(parseSpan traceSpan) ui.Node {
	getIndent := strings.Repeat("..", parseSpan.Depth)
	getName := parseSpan.Name
	if getIndent != "" {
		getName = getIndent + " " + getName
	}
	return html.Tr(html.Props{Class: "border-t border-white/5 align-top"},
		html.Td(html.Props{Class: "px-3 py-3 font-mono text-cyan-200"}, html.Text(strconv.Itoa(parseSpan.Depth))),
		html.Td(html.Props{Class: "px-3 py-3"}, html.Text(parseSpan.Phase)),
		html.Td(html.Props{Class: "px-3 py-3"},
			html.Div(html.Props{Class: "font-semibold text-white"}, html.Text(getName)),
			html.Code(html.Props{Class: "mt-1 block text-xs text-slate-400"}, html.Text(parseSpan.Signature)),
		),
		html.Td(html.Props{Class: "px-3 py-3 text-slate-300"}, html.Text(formatTraceFields(parseSpan.Params))),
		html.Td(html.Props{Class: "px-3 py-3 font-mono text-emerald-300"}, html.Text(formatTraceDuration(parseSpan.DurationNs))),
		html.Td(html.Props{Class: "px-3 py-3 text-slate-300"}, html.Text(parseSpan.Result)),
	)
}

// renderTraceJSONPanel renders the raw JSON document for later review or copy workflows.
func renderTraceJSONPanel(parseDocumentJSON string) ui.Node {
	return shared.ExamplePanel(
		"Raw JSON document",
		html.Pre(html.Props{Class: "mt-3 overflow-x-auto rounded-2xl border border-white/10 bg-black/40 p-4 text-sm text-slate-300"},
			html.Code(html.Props{}, html.Text(parseDocumentJSON)),
		),
	)
}

// renderTraceFieldRow renders one two-column metadata row.
func renderTraceFieldRow(parseLabel string, parseValue string) ui.Node {
	getValue := strings.TrimSpace(parseValue)
	if getValue == "" {
		getValue = "-"
	}
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 p-4"},
		html.Small(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-sm leading-6 text-slate-200"}, html.Text(getValue)),
	)
}

// formatTraceDocumentJSON serializes one trace document into stable indented JSON for review.
func formatTraceDocumentJSON(parseDocument traceDocument) string {
	if strings.TrimSpace(parseDocument.TraceID) == "" {
		return "{\n  \"message\": \"No trace document captured yet.\"\n}"
	}
	getData, getErr := json.MarshalIndent(parseDocument, "", "  ")
	if getErr != nil {
		return "{\"error\": " + strconv.Quote(getErr.Error()) + "}"
	}
	return string(getData)
}

// formatTraceFields flattens one field set into a compact review string.
func formatTraceFields(parseFields []traceField) string {
	if len(parseFields) == 0 {
		return "-"
	}
	getParts := make([]string, 0, len(parseFields))
	for _, getField := range parseFields {
		if strings.TrimSpace(getField.Name) == "" {
			continue
		}
		getParts = append(getParts, getField.Name+"="+getField.Value)
	}
	if len(getParts) == 0 {
		return "-"
	}
	return strings.Join(getParts, ", ")
}

// formatTraceDuration formats nanoseconds into one readable duration string.
func formatTraceDuration(parseDurationNs int64) string {
	if parseDurationNs <= 0 {
		return "0s"
	}
	return time.Duration(parseDurationNs).String()
}

// buildTraceStateFields snapshots the example state cells into a compact review form.
func buildTraceStateFields(parseCounter int, parseLabel string) []traceField {
	return []traceField{
		{Name: "counter", Value: strconv.Itoa(parseCounter)},
		{Name: "label", Value: parseLabel},
	}
}

// startTraceCycle resets the active trace document and begins one new capture window.
func startTraceCycle(parseTrigger traceTrigger) {
	storeTraceState.getNextTraceID++
	getTraceID := fmt.Sprintf("trace-%03d", storeTraceState.getNextTraceID)
	getNow := time.Now().UTC()
	storeTraceState.getActive = traceDocument{
		Version:          1,
		TraceID:          getTraceID,
		StartedAt:        getNow.Format(time.RFC3339Nano),
		Trigger:          parseTrigger,
		Spans:            make([]traceSpan, 0, 16),
		getStartedAtTime: getNow,
	}
	storeTraceState.hasActive = true
	storeTraceState.getSpanStack = nil
	storeTraceState.getSpanIndex = make(map[string]int, 16)
}

// hasTraceCycleActive reports whether one trace cycle is currently collecting spans.
func hasTraceCycleActive() bool {
	return storeTraceState.hasActive
}

// startTraceSpan records one nested timed span inside the active trace document.
func startTraceSpan(parsePhase string, parseName string, parseSignature string, parseParams []traceField) string {
	if !storeTraceState.hasActive {
		return ""
	}
	getParentID := ""
	if len(storeTraceState.getSpanStack) > 0 {
		getParentID = storeTraceState.getSpanStack[len(storeTraceState.getSpanStack)-1]
	}
	getSpanID := fmt.Sprintf("%s-span-%02d", storeTraceState.getActive.TraceID, len(storeTraceState.getActive.Spans)+1)
	getNow := time.Now().UTC()
	getSpan := traceSpan{
		ID:               getSpanID,
		ParentID:         getParentID,
		Phase:            parsePhase,
		Depth:            len(storeTraceState.getSpanStack),
		Name:             parseName,
		Signature:        parseSignature,
		Params:           parseParams,
		StartedAt:        getNow.Format(time.RFC3339Nano),
		getStartedAtTime: getNow,
	}
	storeTraceState.getActive.Spans = append(storeTraceState.getActive.Spans, getSpan)
	storeTraceState.getSpanIndex[getSpanID] = len(storeTraceState.getActive.Spans) - 1
	storeTraceState.getSpanStack = append(storeTraceState.getSpanStack, getSpanID)
	return getSpanID
}

// finishTraceSpan closes one active timed span and stores its result summary.
func finishTraceSpan(parseSpanID string, parseResult string) {
	if !storeTraceState.hasActive || strings.TrimSpace(parseSpanID) == "" {
		return
	}
	getIndex, hasIndex := storeTraceState.getSpanIndex[parseSpanID]
	if !hasIndex || getIndex < 0 || getIndex >= len(storeTraceState.getActive.Spans) {
		return
	}
	getSpan := &storeTraceState.getActive.Spans[getIndex]
	if getSpan.getStartedAtTime.IsZero() {
		return
	}
	getSpan.DurationNs = time.Since(getSpan.getStartedAtTime).Nanoseconds()
	getSpan.Result = parseResult
	getSpan.getStartedAtTime = time.Time{}
	if len(storeTraceState.getSpanStack) > 0 && storeTraceState.getSpanStack[len(storeTraceState.getSpanStack)-1] == parseSpanID {
		storeTraceState.getSpanStack = storeTraceState.getSpanStack[:len(storeTraceState.getSpanStack)-1]
		return
	}
	for getIndex := len(storeTraceState.getSpanStack) - 1; getIndex >= 0; getIndex-- {
		if storeTraceState.getSpanStack[getIndex] == parseSpanID {
			storeTraceState.getSpanStack = append(storeTraceState.getSpanStack[:getIndex], storeTraceState.getSpanStack[getIndex+1:]...)
			return
		}
	}
}

// finishTraceCycle closes the active trace document and stores it as the latest review artifact.
func finishTraceCycle(parseStateAfter []traceField) {
	if !storeTraceState.hasActive {
		return
	}
	getNow := time.Now().UTC()
	storeTraceState.getActive.FinishedAt = getNow.Format(time.RFC3339Nano)
	storeTraceState.getActive.StateAfter = parseStateAfter
	storeTraceState.getActive.Summary = buildTraceSummary(storeTraceState.getActive)
	storeTraceState.getLast = storeTraceState.getActive
	storeTraceState.getLastVersion++
	storeTraceState.getActive = traceDocument{}
	storeTraceState.hasActive = false
	storeTraceState.getSpanStack = nil
	storeTraceState.getSpanIndex = nil
}

// buildTraceSummary calculates the aggregate trace metrics used by the review panel.
func buildTraceSummary(parseDocument traceDocument) traceSummary {
	getSummary := traceSummary{
		SpanCount:       len(parseDocument.Spans),
		TotalDurationNs: 0,
	}
	if !parseDocument.getStartedAtTime.IsZero() {
		getFinishedAt, getErr := time.Parse(time.RFC3339Nano, parseDocument.FinishedAt)
		if getErr == nil {
			getSummary.TotalDurationNs = getFinishedAt.Sub(parseDocument.getStartedAtTime).Nanoseconds()
		}
	}
	for _, getSpan := range parseDocument.Spans {
		if getSpan.Depth > getSummary.MaxDepth {
			getSummary.MaxDepth = getSpan.Depth
		}
		switch getSpan.Phase {
		case "event":
			getSummary.EventDurationNs += getSpan.DurationNs
		case "state":
			getSummary.StateDurationNs += getSpan.DurationNs
		case "render":
			getSummary.RenderDurationNs += getSpan.DurationNs
		}
	}
	return getSummary
}

// buildTraceLastDocument returns the latest completed trace document.
func buildTraceLastDocument() traceDocument {
	return storeTraceState.getLast
}

// buildTraceLastVersion returns the latest completed trace version counter.
func buildTraceLastVersion() int {
	return storeTraceState.getLastVersion
}

// storeTraceLastDocument installs one pre-existing document as the latest review artifact.
func storeTraceLastDocument(parseDocument traceDocument) {
	storeTraceState.getLast = parseDocument
	storeTraceState.getLastVersion++
}

// storeTraceDocument persists one trace document into browser LocalStorage for later review.
func storeTraceDocument(parseDocument traceDocument) error {
	getStorage, getErr := interop.GetLocalStorage()
	if getErr != nil {
		return getErr
	}
	getData, getErr := json.MarshalIndent(parseDocument, "", "  ")
	if getErr != nil {
		return getErr
	}
	return getStorage.SetItem(traceStorageKey, string(getData))
}

// loadTraceDocument restores the last persisted trace document from browser LocalStorage.
func loadTraceDocument() (traceDocument, bool, error) {
	getStorage, getErr := interop.GetLocalStorage()
	if getErr != nil {
		return traceDocument{}, false, getErr
	}
	getValue, hasValue, getErr := getStorage.GetItem(traceStorageKey)
	if getErr != nil {
		return traceDocument{}, false, getErr
	}
	if !hasValue || strings.TrimSpace(getValue) == "" {
		return traceDocument{}, false, nil
	}
	var getDocument traceDocument
	if getErr := json.Unmarshal([]byte(getValue), &getDocument); getErr != nil {
		return traceDocument{}, false, getErr
	}
	return getDocument, true, nil
}

// copyTraceDocument writes the latest JSON document into the browser clipboard.
func copyTraceDocument(parseDocument traceDocument) error {
	getClipboard, getErr := interop.GetClipboard()
	if getErr != nil {
		return getErr
	}
	return getClipboard.WriteText(context.Background(), formatTraceDocumentJSON(parseDocument))
}

// main mounts the example 203 client application.
func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(renderTraceExample), "#app")
	select {}
}
