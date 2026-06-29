package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// TestBuildRuntime2StatusRouteSummaryTextReportsLocalShellRuntime2Active verifies local-shell still means the runtime2 host path is active.
func TestBuildRuntime2StatusRouteSummaryTextReportsLocalShellRuntime2Active(parseT *testing.T) {
	getSummaryText := buildRuntime2StatusRouteSummaryText(true, ui.ParallelRegionStatus{
		GetRegionInstanceID:    "examples.runtime2-status.summary.primary",
		GetRegionMode:          string(runtime2.HostRegionRuntimeModeLocalShell),
		GetAssignedWorkerShard: "ui-parallel-region",
		GetRendererID:          "examples.runtime2-status.summary",
		GetTransportTier:       string(runtime2.TransportTierStructuredClone),
	}, nil)
	if !strings.Contains(getSummaryText, "Verified: yes") {
		parseT.Fatalf("summary text missing verified status: %s", getSummaryText)
	}
	if !strings.Contains(getSummaryText, "Route: runtime2 active (local-shell, awaiting worker attach)") {
		parseT.Fatalf("summary text missing local-shell runtime2 route: %s", getSummaryText)
	}
	if !strings.Contains(getSummaryText, "Worker-backed: no") {
		parseT.Fatalf("summary text missing worker-backed=no: %s", getSummaryText)
	}
}

// TestBuildRuntime2StatusRouteSummaryTextReportsWorkerBackedRuntime2Active verifies worker-attached mode is reported as worker-backed runtime2.
func TestBuildRuntime2StatusRouteSummaryTextReportsWorkerBackedRuntime2Active(parseT *testing.T) {
	getSummaryText := buildRuntime2StatusRouteSummaryText(true, ui.ParallelRegionStatus{
		GetRegionInstanceID:    "examples.runtime2-status.summary.primary",
		GetRegionMode:          string(runtime2.HostRegionRuntimeModeWorkerAttached),
		GetAssignedWorkerShard: "shard-a",
		GetRendererID:          "examples.runtime2-status.summary",
		GetTransportTier:       string(runtime2.TransportTierSharedBuffer),
	}, nil)
	if !strings.Contains(getSummaryText, "Route: runtime2 active (worker-backed)") {
		parseT.Fatalf("summary text missing worker-backed runtime2 route: %s", getSummaryText)
	}
	if !strings.Contains(getSummaryText, "Worker-backed: yes") {
		parseT.Fatalf("summary text missing worker-backed=yes: %s", getSummaryText)
	}
}

// TestBuildRuntime2StatusRouteSummaryTextReportsPendingLookup verifies the summary explains when the public runtime2 status is not ready yet.
func TestBuildRuntime2StatusRouteSummaryTextReportsPendingLookup(parseT *testing.T) {
	getSummaryText := buildRuntime2StatusRouteSummaryText(false, ui.ParallelRegionStatus{
		GetRegionInstanceID: "examples.runtime2-status.summary.primary",
	}, nil)
	if !strings.Contains(getSummaryText, "Verified: pending") {
		parseT.Fatalf("summary text missing pending verification state: %s", getSummaryText)
	}
	if !strings.Contains(getSummaryText, "waiting on public runtime2 status") {
		parseT.Fatalf("summary text missing waiting copy: %s", getSummaryText)
	}
}
