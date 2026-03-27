//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/ui"
)

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
	parseRenderCountRef := ui.UseRef(0)
	parseRenderCount := parseRenderCountRef.Get() + 1
	storeRuntime2StatusRenderLabelCount(parseLabel, parseRenderCount)
	ui.UseEffect(func() func() {
		parseRenderCountRef.Set(parseRenderCount)
		parseRenderPhase := "initial render"
		if parseRenderCount > 1 {
			parseRenderPhase = "rerender"
		}
		fmt.Printf("[runtime2-status/runtime2] %s label=%s count=%d\n", parseRenderPhase, parseLabel, parseRenderCount)
		return nil
	}, parseLabel, parseRenderCount)
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
