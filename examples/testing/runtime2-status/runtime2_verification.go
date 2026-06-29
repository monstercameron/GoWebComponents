package main

import (
	"fmt"
	"strings"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// buildRuntime2StatusNoticeTexts builds the operator-facing caveats that example 200 should surface.
func buildRuntime2StatusNoticeTexts(parseHasRuntimeStatus bool, parseRuntimeStatus ui.ParallelRegionStatus) []string {
	if !parseHasRuntimeStatus {
		return nil
	}
	getNoticeTexts := make([]string, 0, 2)
	if parseRuntimeStatus.GetAssignedWorkerShard == "ui-parallel-region" {
		getNoticeTexts = append(getNoticeTexts,
			"Runtime2 is active through the public host adapter even when the inspector still reports the framework-owned bridge shard label. Local-shell is the runtime2 path before worker attach; the fleet card below is the separate Go WASM probe surface.",
		)
	}
	if !parseRuntimeStatus.GetIsHydrationComplete &&
		!parseRuntimeStatus.HasHydratedShellAnchor &&
		!parseRuntimeStatus.HasPostHydrationAttached {
		getNoticeTexts = append(getNoticeTexts,
			"This page mounts with ui.Render(...), not ui.Hydrate(...), so the runtime2 hydration fields stay false until the browser attaches the worker-backed route.",
		)
	}
	return getNoticeTexts
}

// buildRuntime2StatusRouteSummaryText formats the runtime2 verdict text shown in the inspector and console logs.
func buildRuntime2StatusRouteSummaryText(parseHasRuntimeStatus bool, parseRuntimeStatus ui.ParallelRegionStatus, parseRuntimeStatusErr error) string {
	getSourceLabel := "ui.GetParallelRegionRuntimeStatus(...)"
	getRegionLabel := parseRuntimeStatus.GetRegionInstanceID
	if getRegionLabel == "" {
		getRegionLabel = "untracked"
	}
	if parseRuntimeStatusErr != nil {
		return fmt.Sprintf(
			"Runtime2 proof\nSource: %s\nRegion: %s\nVerified: error\nRoute: status lookup failed\nError: %v",
			getSourceLabel,
			getRegionLabel,
			parseRuntimeStatusErr,
		)
	}
	if !parseHasRuntimeStatus {
		return fmt.Sprintf(
			"Runtime2 proof\nSource: %s\nRegion: %s\nVerified: pending\nRoute: waiting on public runtime2 status",
			getSourceLabel,
			getRegionLabel,
		)
	}

	getRouteLabel := "runtime2 active (local-shell, awaiting worker attach)"
	getWorkerBackedLabel := "no"
	switch parseRuntimeStatus.GetRegionMode {
	case string(runtime2.HostRegionRuntimeModeWorkerAttached):
		getRouteLabel = "runtime2 active (worker-backed)"
		getWorkerBackedLabel = "yes"
	case string(runtime2.HostRegionRuntimeModeFallback):
		getRouteLabel = "runtime2 fallback"
	}
	getTransportLabel := parseRuntimeStatus.GetTransportTier
	if getTransportLabel == "" {
		getTransportLabel = string(runtime2.TransportTierStructuredClone)
	}
	getShardLabel := parseRuntimeStatus.GetAssignedWorkerShard
	if getShardLabel == "" {
		getShardLabel = "unassigned"
	}
	getRendererLabel := parseRuntimeStatus.GetRendererID
	if getRendererLabel == "" {
		getRendererLabel = "unregistered"
	}
	getFallbackReason := strings.TrimSpace(parseRuntimeStatus.GetFallbackReason)
	if getFallbackReason == "" {
		getFallbackReason = "none"
	}
	return fmt.Sprintf(
		"Runtime2 proof\nSource: %s\nRegion: %s\nVerified: yes\nRoute: %s\nMode: %s\nWorker-backed: %s\nAssigned shard: %s\nRenderer: %s\nTransport tier: %s\nFallback reason: %s",
		getSourceLabel,
		getRegionLabel,
		getRouteLabel,
		parseRuntimeStatus.GetRegionMode,
		getWorkerBackedLabel,
		getShardLabel,
		getRendererLabel,
		getTransportLabel,
		getFallbackReason,
	)
}
