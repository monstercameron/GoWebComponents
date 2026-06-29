//go:build js && wasm

package main

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type runtime2StatusReactiveSource struct {
	GetSourceIDs []string
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

// ReactiveRegionSourceIDs returns one stable copy of the configured reactive source IDs.
func (parseSource runtime2StatusReactiveSource) ReactiveRegionSourceIDs() []string {
	return append([]string(nil), parseSource.GetSourceIDs...)
}

// buildRuntime2StatusCounterReactiveSource binds the shared counter atom ID into one ReactiveRegion source handle.
func buildRuntime2StatusCounterReactiveSource() ui.ReactiveSource {
	return runtime2StatusReactiveSource{
		GetSourceIDs: []string{getRuntime2StatusCounterAtomID},
	}
}

// buildRuntime2StatusCounterSourceIDs binds the shared counter atom ID into the public parallel-region source contract.
func buildRuntime2StatusCounterSourceIDs() []string {
	getSourceIDs, parseErr := ui.BuildParallelRegionSourceIDs(buildRuntime2StatusCounterReactiveSource())
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

// buildRuntime2StatusNoticeKey creates a stable dependency key for one set of inspector notices.
func buildRuntime2StatusNoticeKey(parseNoticeTexts []string) string {
	return strings.Join(parseNoticeTexts, "|")
}

// handleRuntime2StatusRuntimeProofEffect logs the runtime2 proof summary once per distinct status snapshot.
func handleRuntime2StatusRuntimeProofEffect(parseRegionID string, parseRouteSummaryText string) {
	parseSummaryText := strings.TrimSpace(parseRouteSummaryText)
	ui.UseEffect(func() func() {
		if parseSummaryText == "" {
			return nil
		}
		fmt.Printf("[runtime2-status/runtime2] runtime2 proof region=%s\n%s\n", parseRegionID, parseSummaryText)
		return nil
	}, parseRegionID, parseSummaryText)
}

// trackRuntime2StatusRenderCount tracks the current render pass for one runtime2 example surface and logs rerenders after commit.
func trackRuntime2StatusRenderCount(parseLabel string) int {
	parseCommittedRenderCountRef := ui.UseRef(0)
	parseRenderCount := parseCommittedRenderCountRef.Get() + 1
	ui.UseEffect(func() func() {
		parseCommittedRenderCount := parseCommittedRenderCountRef.Get() + 1
		parseCommittedRenderCountRef.Set(parseCommittedRenderCount)
		storeRuntime2StatusRenderLabelCount(parseLabel, parseCommittedRenderCount)
		parseRenderPhase := "initial render"
		if parseCommittedRenderCount > 1 {
			parseRenderPhase = "rerender"
		}
		fmt.Printf("[runtime2-status/runtime2] %s label=%s count=%d\n", parseRenderPhase, parseLabel, parseCommittedRenderCount)
		return nil
	})
	return parseRenderCount
}

// handleRuntime2StatusNoticeEffect logs each operator-facing caveat once so the example cannot quietly overclaim.
func handleRuntime2StatusNoticeEffect(parseRegionID string, parseNoticeTexts []string, parseWarnedByMessageRef ui.Ref[map[string]bool]) {
	getNoticeKey := buildRuntime2StatusNoticeKey(parseNoticeTexts)
	ui.UseEffect(func() func() {
		if len(parseNoticeTexts) == 0 {
			return nil
		}
		getWarnedByMessage := parseWarnedByMessageRef.Get()
		if getWarnedByMessage == nil {
			getWarnedByMessage = map[string]bool{}
		}
		for _, getNoticeText := range parseNoticeTexts {
			if getWarnedByMessage[getNoticeText] {
				continue
			}
			fmt.Printf("[runtime2-status/runtime2] notice region=%s: %s\n", parseRegionID, getNoticeText)
			getWarnedByMessage[getNoticeText] = true
		}
		parseWarnedByMessageRef.Set(getWarnedByMessage)
		return nil
	}, parseRegionID, getNoticeKey)
}
