//go:build js && wasm
// +build js,wasm

package devtools

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestSnapshotNowIncludesBufferedLogs(t *testing.T) {
	runtime.ClearLogs()
	runtime.ClearDiagnostics()
	ResetMultiClientInspection()
	ResetSerializationBoundaryInspection()
	ResetCoordinationInspection()
	defer runtime.ClearLogs()
	defer runtime.ClearDiagnostics()
	defer ResetMultiClientInspection()
	defer ResetSerializationBoundaryInspection()
	defer ResetCoordinationInspection()

	runtime.ReportLogWithFields("router", runtime.LogInfo, runtime.DiagnosticInformational, "navigation started", "nav-1", map[string]string{
		"target": "/dashboard",
	})

	snapshot := SnapshotNow()
	if len(snapshot.Logs) != 1 {
		t.Fatalf("expected one buffered log, got %+v", snapshot.Logs)
	}
	if snapshot.Logs[0].Domain != "router" || snapshot.Logs[0].Fields["target"] != "/dashboard" {
		t.Fatalf("unexpected buffered log payload: %+v", snapshot.Logs[0])
	}
}

func TestSnapshotNowIncludesWrappedPanicMetadata(t *testing.T) {
	runtime.ClearLogs()
	runtime.ClearDiagnostics()
	ResetMultiClientInspection()
	ResetSerializationBoundaryInspection()
	ResetCoordinationInspection()
	defer runtime.ClearLogs()
	defer runtime.ClearDiagnostics()
	defer ResetMultiClientInspection()
	defer ResetSerializationBoundaryInspection()
	defer ResetCoordinationInspection()

	message := runtime.ReportUnhandledPanicContext("runtime", runtime.PanicPhaseStartup, "RenderTo", "#app", []string{"App"}, "startup boom")
	if message == "" {
		t.Fatal("expected wrapped panic message")
	}

	snapshot := SnapshotNow()
	runtimeDiagnostics := runtime.GetDiagnostics()
	if len(runtimeDiagnostics) == 0 {
		t.Fatal("expected runtime panic diagnostic")
	}
	runtimeDiagnostic := runtimeDiagnostics[len(runtimeDiagnostics)-1]
	if len(snapshot.Diagnostics) == 0 {
		t.Fatal("expected panic diagnostic in snapshot")
	}
	diagnostic := snapshot.Diagnostics[len(snapshot.Diagnostics)-1]
	if diagnostic.Code != "GWC-RUNTIME-PANIC-STARTUP" || diagnostic.Path != "#app" {
		t.Fatalf("unexpected diagnostic payload: %+v", diagnostic)
	}
	if diagnostic.TopFrame == "" || diagnostic.Consequence == "" {
		t.Fatalf("expected wrapped panic diagnostic metadata, got %+v", diagnostic)
	}
	if diagnostic.Fields["path"] != "#app" || diagnostic.Fields["phase"] != "startup" || diagnostic.Fields["where"] != "RenderTo" {
		t.Fatalf("expected snapshot diagnostic fields to mirror runtime context, got %+v", diagnostic)
	}
	if diagnostic.Code != runtimeDiagnostic.Code ||
		diagnostic.Message != runtimeDiagnostic.Message ||
		diagnostic.Path != runtimeDiagnostic.Path ||
		diagnostic.Docs != runtimeDiagnostic.Docs ||
		diagnostic.Remediation != runtimeDiagnostic.Remediation ||
		diagnostic.Recoverable != runtimeDiagnostic.Recoverable ||
		diagnostic.TopFrame != runtimeDiagnostic.TopFrame ||
		diagnostic.Consequence != runtimeDiagnostic.Consequence ||
		diagnostic.Fields["path"] != runtimeDiagnostic.Fields["path"] ||
		diagnostic.Fields["phase"] != runtimeDiagnostic.Fields["phase"] ||
		diagnostic.Fields["where"] != runtimeDiagnostic.Fields["where"] {
		t.Fatalf("expected snapshot diagnostic to mirror runtime diagnostic, snapshot=%+v runtime=%+v", diagnostic, runtimeDiagnostic)
	}

	runtimeLogs := runtime.GetLogs()
	if len(runtimeLogs) == 0 {
		t.Fatal("expected runtime panic log")
	}
	runtimeLog := runtimeLogs[len(runtimeLogs)-1]
	if len(snapshot.Logs) == 0 {
		t.Fatal("expected panic log in snapshot")
	}
	entry := snapshot.Logs[len(snapshot.Logs)-1]
	if entry.Code != "GWC-RUNTIME-PANIC-STARTUP" {
		t.Fatalf("unexpected log payload: %+v", entry)
	}
	if entry.TopFrame == "" || entry.Consequence == "" {
		t.Fatalf("expected wrapped panic log metadata, got %+v", entry)
	}
	if entry.Fields["path"] != "#app" || entry.Fields["runtime"] == "" || entry.Fields["top_frame"] == "" {
		t.Fatalf("expected panic fields to mirror into log entry, got %+v", entry)
	}
	if entry.Code != runtimeLog.Code ||
		entry.Message != runtimeLog.Message ||
		entry.Docs != runtimeLog.Docs ||
		entry.Remediation != runtimeLog.Remediation ||
		entry.Recoverable != runtimeLog.Recoverable ||
		entry.TopFrame != runtimeLog.TopFrame ||
		entry.Consequence != runtimeLog.Consequence ||
		entry.Fields["path"] != runtimeLog.Fields["path"] ||
		entry.Fields["runtime"] != runtimeLog.Fields["runtime"] ||
		entry.Fields["top_frame"] != runtimeLog.Fields["top_frame"] {
		t.Fatalf("expected snapshot log to mirror runtime log, snapshot=%+v runtime=%+v", entry, runtimeLog)
	}
}

func TestSnapshotNowIncludesMultiClientInspection(t *testing.T) {
	ResetMultiClientInspection()
	ResetSerializationBoundaryInspection()
	ResetCoordinationInspection()
	defer ResetMultiClientInspection()
	defer ResetSerializationBoundaryInspection()
	defer ResetCoordinationInspection()

	SetMultiClientInspection(MultiClient{
		Enabled:           true,
		LocalPeerID:       "storefront-1",
		ResolvedTransport: "broadcast-channel",
		AuthorityView: map[string]string{
			"operator:inventory": "ops-1",
		},
		Peers: []MultiClientPeer{{
			ID:              "ops-1",
			App:             "atlas",
			Surface:         "tab",
			Role:            "operator",
			State:           "ready",
			LeaseDeadline:   time.Date(2026, 3, 20, 12, 0, 5, 0, time.UTC),
			LastSeen:        time.Date(2026, 3, 20, 12, 0, 0, 0, time.UTC),
			ProtocolVersion: "v1",
			Encodings:       []string{"json", "binary"},
			Topics:          []string{"clients", "operator:inventory"},
			Compatible:      true,
		}},
		RecentTraffic: []MultiClientTraffic{{
			Direction:     "outbound",
			Kind:          "query",
			Topic:         "clients",
			PeerID:        "ops-1",
			CorrelationID: "req-1",
			LatencyMs:     14,
			Timestamp:     time.Date(2026, 3, 20, 12, 0, 1, 0, time.UTC),
		}},
		FailedPublishes: []MultiClientFailure{{
			Op:        "PublishClientIntent",
			Topic:     "operator:inventory",
			Target:    "ops-1",
			Code:      "unauthorized",
			Message:   "client role is not authorized for privileged topic",
			Timestamp: time.Date(2026, 3, 20, 12, 0, 2, 0, time.UTC),
		}},
	})

	snapshot := SnapshotNow()
	if !snapshot.MultiClient.Enabled || snapshot.MultiClient.LocalPeerID != "storefront-1" || snapshot.MultiClient.ResolvedTransport != "broadcast-channel" {
		t.Fatalf("unexpected multi-client snapshot header: %+v", snapshot.MultiClient)
	}
	if len(snapshot.MultiClient.Peers) != 1 || snapshot.MultiClient.Peers[0].ID != "ops-1" || !snapshot.MultiClient.Peers[0].Compatible {
		t.Fatalf("unexpected multi-client peer state: %+v", snapshot.MultiClient.Peers)
	}
	if len(snapshot.MultiClient.RecentTraffic) != 1 || snapshot.MultiClient.RecentTraffic[0].CorrelationID != "req-1" {
		t.Fatalf("unexpected multi-client traffic state: %+v", snapshot.MultiClient.RecentTraffic)
	}
	if len(snapshot.MultiClient.FailedPublishes) != 1 || snapshot.MultiClient.FailedPublishes[0].Code != "unauthorized" {
		t.Fatalf("unexpected multi-client failure state: %+v", snapshot.MultiClient.FailedPublishes)
	}
}

func TestExportSnapshotJSONAndCompareSnapshots(t *testing.T) {
	before := Snapshot{
		Route:       Route{Path: "/before"},
		Cache:       []CacheEntry{{Key: "item", Ready: true, UpdatedAt: time.Unix(1, 0).UTC()}},
		MultiClient: MultiClient{ResolvedTransport: "storage-event"},
		Boundaries: BoundaryInspection{Entries: []Boundary{{
			Name:      "ssr.bootstrap",
			Kind:      "ssr-bootstrap",
			Direction: "server-to-client",
			Status:    "observed",
			SizeBytes: 128,
		}}},
		Coordination: Coordination{Workers: []WorkerJob{{Name: "worker.search", Status: "idle"}}},
		Stats:        Stats{TotalFibers: 1},
		Hydration:    HydrationDebug{CorrelationID: "hydrate-a", MismatchCount: 1},
		Logs:         []Log{{Domain: "router", Message: "before"}},
	}
	after := Snapshot{
		Route:       Route{Path: "/after"},
		Cache:       []CacheEntry{{Key: "item", Ready: true, UpdatedAt: time.Unix(1, 0).UTC()}},
		MultiClient: MultiClient{ResolvedTransport: "broadcast-channel"},
		Boundaries: BoundaryInspection{Entries: []Boundary{{
			Name:      "worker.search",
			Kind:      "worker",
			Direction: "client-to-worker",
			Status:    "rejected",
			SizeBytes: 512,
		}}},
		Coordination: Coordination{Replay: []ReplayEntry{{ID: "mut-1", Kind: "order.submit", State: "retrying", Attempts: 1, MaxAttempts: 3}}},
		Stats:        Stats{TotalFibers: 2},
		Hydration:    HydrationDebug{CorrelationID: "hydrate-b", MismatchCount: 3, Failed: true},
		Logs:         []Log{{Domain: "router", Message: "after"}},
	}

	payload, err := ExportSnapshotJSON(before)
	if err != nil {
		t.Fatalf("ExportSnapshotJSON failed: %v", err)
	}
	if len(payload) == 0 || payload[0] != '{' {
		t.Fatalf("expected JSON payload, got %q", string(payload))
	}

	comparison, err := CompareSnapshots(before, after)
	if err != nil {
		t.Fatalf("CompareSnapshots failed: %v", err)
	}
	if comparison.Equal {
		t.Fatal("expected snapshots to differ")
	}
	if comparison.PreviousFingerprint == "" || comparison.CurrentFingerprint == "" {
		t.Fatal("expected fingerprints to be populated")
	}
	if comparison.PreviousSize == 0 || comparison.CurrentSize == 0 {
		t.Fatal("expected serialized sizes to be populated")
	}
	if len(comparison.ChangedSections) == 0 {
		t.Fatal("expected changed sections to be reported")
	}
	want := map[string]bool{"route": true, "multiClient": true, "boundaries": true, "coordination": true, "stats": true, "hydration": true, "logs": true}
	for _, section := range comparison.ChangedSections {
		delete(want, section)
	}
	if len(want) != 0 {
		t.Fatalf("expected route, multiClient, boundaries, coordination, stats, hydration, and logs to change, missing %v", want)
	}
}

func TestMapHydrationIncludesDebugFields(t *testing.T) {
	mapped := mapHydration(runtime.HydrationDebugSnapshot{
		CorrelationID:        "hydrate-77",
		StartedAt:            "2026-03-25T12:00:00.000Z",
		FinishedAt:           "2026-03-25T12:00:00.014Z",
		DurationNs:           14,
		ExistingDOMNodeCount: 8,
		FallbackCount:        1,
		MismatchCount:        2,
		DiscardedNodeCount:   4,
		Strict:               true,
		Failed:               true,
		Failure:              "hydration mismatch forced replacement",
		RecentMessages: []string{
			"hydration text mismatch at App > Hero",
			"hydration fallback at App > Sidebar",
		},
	})

	if mapped.CorrelationID != "hydrate-77" || mapped.MismatchCount != 2 || mapped.DiscardedNodeCount != 4 {
		t.Fatalf("expected hydration debug fields to map, got %+v", mapped)
	}
	if !mapped.Strict || !mapped.Failed || mapped.Failure == "" {
		t.Fatalf("expected hydration strict/failure metadata to map, got %+v", mapped)
	}
	if len(mapped.RecentMessages) != 2 || mapped.RecentMessages[0] != "hydration text mismatch at App > Hero" {
		t.Fatalf("expected hydration messages to map, got %+v", mapped.RecentMessages)
	}
	if summary := hydrationSummary(mapped); summary == nil {
		t.Fatal("expected hydration summary node")
	}
}

func TestSnapshotNowIncludesSerializationBoundaries(t *testing.T) {
	runtime.ClearLogs()
	runtime.ClearDiagnostics()
	ResetMultiClientInspection()
	ResetSerializationBoundaryInspection()
	ResetCoordinationInspection()
	defer runtime.ClearLogs()
	defer runtime.ClearDiagnostics()
	defer ResetMultiClientInspection()
	defer ResetSerializationBoundaryInspection()
	defer ResetCoordinationInspection()

	bootstrap := ui.SSRBootstrap{Data: map[string]interface{}{
		"legacy-message": "hello",
	}}
	if err := ui.RegisterSessionBootstrapHint(&bootstrap, "viewer", map[string]string{"role": "operator"}); err != nil {
		t.Fatalf("RegisterSessionBootstrapHint() error = %v", err)
	}
	inspection, err := InspectBootstrapBoundaries(bootstrap)
	if err != nil {
		t.Fatalf("InspectBootstrapBoundaries() error = %v", err)
	}
	SetSerializationBoundaryInspection(inspection)

	snapshot := SnapshotNow()
	if len(snapshot.Boundaries.Entries) != 3 {
		t.Fatalf("expected bootstrap boundary entries in snapshot, got %+v", snapshot.Boundaries)
	}
	if snapshot.Boundaries.Entries[0].Name != "ssr.bootstrap" || snapshot.Boundaries.Entries[0].Kind != "ssr-bootstrap" {
		t.Fatalf("unexpected root boundary entry: %+v", snapshot.Boundaries.Entries[0])
	}
	if summary := boundariesSummary(snapshot.Boundaries); summary == nil {
		t.Fatal("expected boundaries summary node")
	}
}

func TestSnapshotNowIncludesCoordinationInspection(t *testing.T) {
	runtime.ClearLogs()
	runtime.ClearDiagnostics()
	ResetMultiClientInspection()
	ResetSerializationBoundaryInspection()
	ResetCoordinationInspection()
	defer runtime.ClearLogs()
	defer runtime.ClearDiagnostics()
	defer ResetMultiClientInspection()
	defer ResetSerializationBoundaryInspection()
	defer ResetCoordinationInspection()

	SetCoordinationInspection(Coordination{
		Workers: []WorkerJob{{
			Name:      "worker.search",
			Status:    "running",
			RequestID: "req-22",
			Running:   true,
		}},
		SyncEvents: []SyncEvent{{
			Channel:   "prefs",
			Transport: "broadcast-channel",
			Direction: "outbound",
			Topic:     "theme",
			Target:    "tab-2",
			Status:    "sent",
			Timestamp: time.Unix(20, 0).UTC(),
		}},
		Replay: []ReplayEntry{{
			ID:          "mut-1",
			Kind:        "order.submit",
			State:       "retrying",
			Attempts:    1,
			MaxAttempts: 3,
		}},
	})

	snapshot := SnapshotNow()
	if len(snapshot.Coordination.Workers) != 1 || len(snapshot.Coordination.SyncEvents) != 1 || len(snapshot.Coordination.Replay) != 1 {
		t.Fatalf("expected coordination state in snapshot, got %+v", snapshot.Coordination)
	}
	if summary := coordinationSummary(snapshot.Coordination); summary == nil {
		t.Fatal("expected coordination summary node")
	}
}

func TestCollectOverlayIssuesHighlightsActionableFailures(t *testing.T) {
	runtime.ClearLogs()
	runtime.ClearDiagnostics()
	ResetMultiClientInspection()
	ResetSerializationBoundaryInspection()
	ResetCoordinationInspection()
	defer runtime.ClearLogs()
	defer runtime.ClearDiagnostics()
	defer ResetMultiClientInspection()
	defer ResetSerializationBoundaryInspection()
	defer ResetCoordinationInspection()

	runtime.ReportDiagnostic("runtime", runtime.DiagnosticWarning, "hydration fallback at App > Shell")
	runtime.ReportDiagnosticWithContext("router", runtime.DiagnosticError, "route loader failed", "/reports/7", []string{"App", "Reports"})

	snapshot := SnapshotNow()
	issues := collectOverlayIssues(snapshot)
	if len(issues) != 2 {
		t.Fatalf("expected loader failure and hydration issue in overlay, got %+v", issues)
	}
	if issues[0].Code == "" && issues[1].Code == "" {
		t.Fatalf("expected overlay issues to preserve stable codes, got %+v", issues)
	}
}

func TestMatchingErrorOverlayActionsFiltersByCodeAndSource(t *testing.T) {
	issue := ErrorOverlayIssue{
		Source: "router",
		Code:   "GWC-ROUTER-LOADER-FAILED",
	}
	actions := []ErrorOverlayAction{
		{Label: "Retry loader", MatchCodes: []string{"GWC-ROUTER-LOADER-FAILED"}},
		{Label: "Reveal launcher", MatchSources: []string{"runtime"}},
		{Label: "Always"},
	}

	matched := matchingErrorOverlayActions(issue, actions)
	if len(matched) != 2 {
		t.Fatalf("expected two matching overlay actions, got %+v", matched)
	}
	if matched[0].Label != "Retry loader" || matched[1].Label != "Always" {
		t.Fatalf("unexpected overlay action match order: %+v", matched)
	}
}

func TestMapInspectionFineGrainedMetadata(t *testing.T) {
	node := mapNode(&runtime.FiberSnapshot{
		Name:           "ReactiveText",
		Kind:           "text",
		FineGrained:    true,
		ReactiveSource: "count",
		UpdateOrigin:   "fine-grained",
		Hooks: []runtime.HookSnapshot{{
			Slot:         1,
			Kind:         "effect",
			Value:        "deps=2",
			Dependencies: `"theme", true`,
			Status:       "cleanup=registered epoch=4",
		}},
	})
	if node == nil {
		t.Fatal("expected mapped node")
	}
	if !node.FineGrained {
		t.Fatal("expected fine-grained flag to map")
	}
	if node.ReactiveSource != "count" {
		t.Fatalf("expected reactive source count, got %q", node.ReactiveSource)
	}
	if node.UpdateOrigin != "fine-grained" {
		t.Fatalf("expected update origin fine-grained, got %q", node.UpdateOrigin)
	}
	if len(node.Hooks) != 1 || node.Hooks[0].Slot != 1 || node.Hooks[0].Dependencies != `"theme", true` || node.Hooks[0].Status != "cleanup=registered epoch=4" {
		t.Fatalf("expected hook inspection metadata to map, got %+v", node.Hooks)
	}

	stats := mapStats(runtime.InspectionStats{FineGrainedFibers: 1})
	if stats.FineGrainedFibers != 1 {
		t.Fatalf("expected fine-grained fiber count to map, got %d", stats.FineGrainedFibers)
	}

	profiling := mapProfiling(runtime.ProfilingSnapshot{ScheduledGranularMarks: 4, FineGrainedCommits: 3, FineGrainedDescendantHostCommits: 6, FineGrainedDescendantTextCommits: 2})
	if profiling.ScheduledGranularMarks != 4 {
		t.Fatalf("expected granular marks to map, got %d", profiling.ScheduledGranularMarks)
	}
	if profiling.FineGrainedCommits != 3 {
		t.Fatalf("expected fine-grained commits to map, got %d", profiling.FineGrainedCommits)
	}
	if profiling.FineGrainedDescendantHostCommits != 6 {
		t.Fatalf("expected descendant host commits to map, got %d", profiling.FineGrainedDescendantHostCommits)
	}
	if profiling.FineGrainedDescendantTextCommits != 2 {
		t.Fatalf("expected descendant text commits to map, got %d", profiling.FineGrainedDescendantTextCommits)
	}

	extended := mapProfiling(runtime.ProfilingSnapshot{
		PhaseTotals: runtime.ProfilingPhaseTotalsSnapshot{
			RenderDurationNs:  12,
			DiffDurationNs:    7,
			CommitDurationNs:  5,
			EffectDurationNs:  3,
			CleanupDurationNs: 2,
		},
		RecentEvents: []runtime.ProfilingEvent{{
			Domain:     "router",
			Name:       "navigation",
			Phase:      "finish",
			Target:     "/dashboard",
			DurationNs: 14,
			Fields: map[string]string{
				"mode": "push",
			},
		}},
		ComponentRenders: []runtime.ComponentRenderTraceSnapshot{{
			Name:                    "Dashboard",
			Path:                    "App > Dashboard",
			RenderCount:             4,
			RerenderCount:           3,
			LastTrigger:             "hook",
			LastRenderDurationNs:    9,
			TotalRenderDurationNs:   24,
			AverageRenderDurationNs: 6,
			TriggerCounts: map[string]int{
				"hook":  3,
				"mount": 1,
			},
		}},
		HotBranches: []runtime.HotBranchSnapshot{{
			Name:              "Dashboard",
			Path:              "App > Dashboard",
			RenderDurationNs:  10,
			DiffDurationNs:    4,
			CommitDurationNs:  6,
			EffectDurationNs:  2,
			CleanupDurationNs: 1,
			SelfDurationNs:    23,
			SubtreeDurationNs: 40,
		}},
		FlamegraphFrames: []runtime.FlamegraphFrameSnapshot{{
			Name:              "Dashboard",
			Path:              "App > Dashboard",
			Depth:             1,
			StartNs:           2,
			DurationNs:        40,
			SelfDurationNs:    23,
			RenderDurationNs:  10,
			DiffDurationNs:    4,
			CommitDurationNs:  6,
			EffectDurationNs:  2,
			CleanupDurationNs: 1,
		}},
		Startup: runtime.StartupProfilingSnapshot{
			Mode:                       "hydrate",
			StartedAt:                  "2026-03-24T15:04:05.000Z",
			BootstrapReadDurationNs:    9,
			HydrationDurationNs:        14,
			StartupCommitDurationNs:    6,
			FirstInteractionDurationNs: 42,
			FirstInteractionCaptured:   true,
			FirstInteractionEvent:      "event",
		},
	})
	if extended.PhaseTotals.CommitDurationNs != 5 || extended.PhaseTotals.DiffDurationNs != 7 {
		t.Fatalf("expected phase totals to map, got %+v", extended.PhaseTotals)
	}
	if len(extended.RecentEvents) != 1 || extended.RecentEvents[0].Domain != "router" || extended.RecentEvents[0].Fields["mode"] != "push" {
		t.Fatalf("expected profiling events to map, got %+v", extended.RecentEvents)
	}
	if len(extended.ComponentRenders) != 1 || extended.ComponentRenders[0].Name != "Dashboard" || extended.ComponentRenders[0].TriggerCounts["hook"] != 3 {
		t.Fatalf("expected component render traces to map, got %+v", extended.ComponentRenders)
	}
	if len(extended.HotBranches) != 1 || extended.HotBranches[0].RenderDurationNs != 10 || extended.HotBranches[0].DiffDurationNs != 4 {
		t.Fatalf("expected hot branch render/diff attribution to map, got %+v", extended.HotBranches)
	}
	if len(extended.FlamegraphFrames) != 1 || extended.FlamegraphFrames[0].Depth != 1 || extended.FlamegraphFrames[0].DurationNs != 40 {
		t.Fatalf("expected flamegraph frames to map, got %+v", extended.FlamegraphFrames)
	}
	if extended.Startup.Mode != "hydrate" || !extended.Startup.FirstInteractionCaptured || extended.Startup.FirstInteractionDurationNs != 42 {
		t.Fatalf("expected startup profiling to map, got %+v", extended.Startup)
	}
}

func TestRouteSummaryMappingIncludesStackLoadersAndRedirect(t *testing.T) {
	snapshot := Snapshot{
		Route: Route{
			Path:    "/dashboard/reports/7",
			Loading: true,
			Stack: []RouteStack{
				{Path: "/dashboard", HasBeforeLeave: true},
				{Path: "/dashboard/reports/7", HasLoader: true, HasBeforeEnter: true},
			},
			Loaders: []RouteLoader{{
				Key:     "report@/dashboard/reports/7",
				Path:    "/dashboard/reports/7",
				Pending: true,
				HasData: false,
			}},
			LastRedirect: RouteRedirect{Cause: "before-enter", From: "/secure", To: "/login"},
			Metadata: RouteMetadata{
				Title:        "Report",
				Description:  "Report detail",
				CanonicalURL: "/dashboard/reports/7",
			},
		},
	}

	if summary := routeSummary(snapshot.Route); summary == nil {
		t.Fatal("expected route summary node")
	}
	if snapshot.Route.Stack[1].HasLoader != true || snapshot.Route.LastRedirect.To != "/login" || snapshot.Route.Metadata.Title != "Report" {
		t.Fatalf("expected route debug mapping data to round-trip, got %+v", snapshot.Route)
	}
}
