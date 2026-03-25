//go:build js && wasm
// +build js,wasm

package devtools

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestSnapshotNowIncludesBufferedLogs(parseT *testing.T) {
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

	parseSnapshot := SnapshotNow()
	if len(parseSnapshot.Logs) != 1 {
		parseT.Fatalf("expected one buffered log, got %+v", parseSnapshot.Logs)
	}
	if parseSnapshot.Logs[0].Domain != "router" || parseSnapshot.Logs[0].Fields["target"] != "/dashboard" {
		parseT.Fatalf("unexpected buffered log payload: %+v", parseSnapshot.Logs[0])
	}
}

func TestSnapshotNowIncludesWrappedPanicMetadata(parseT *testing.T) {
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

	parseMessage := runtime.ReportUnhandledPanicContext("runtime", runtime.PanicPhaseStartup, "RenderTo", "#app", []string{"App"}, "startup boom")
	if parseMessage == "" {
		parseT.Fatal("expected wrapped panic message")
	}

	parseSnapshot := SnapshotNow()
	parseRuntimeDiagnostics := runtime.GetDiagnostics()
	if len(parseRuntimeDiagnostics) == 0 {
		parseT.Fatal("expected runtime panic diagnostic")
	}
	parseRuntimeDiagnostic := parseRuntimeDiagnostics[len(parseRuntimeDiagnostics)-1]
	if len(parseSnapshot.Diagnostics) == 0 {
		parseT.Fatal("expected panic diagnostic in snapshot")
	}
	parseDiagnostic := parseSnapshot.Diagnostics[len(parseSnapshot.Diagnostics)-1]
	if parseDiagnostic.Code != "GWC-RUNTIME-PANIC-STARTUP" || parseDiagnostic.Path != "#app" {
		parseT.Fatalf("unexpected diagnostic payload: %+v", parseDiagnostic)
	}
	if parseDiagnostic.TopFrame == "" || parseDiagnostic.Consequence == "" {
		parseT.Fatalf("expected wrapped panic diagnostic metadata, got %+v", parseDiagnostic)
	}
	if parseDiagnostic.Fields["path"] != "#app" || parseDiagnostic.Fields["phase"] != "startup" || parseDiagnostic.Fields["where"] != "RenderTo" {
		parseT.Fatalf("expected snapshot diagnostic fields to mirror runtime context, got %+v", parseDiagnostic)
	}
	if parseDiagnostic.Code != parseRuntimeDiagnostic.Code ||
		parseDiagnostic.Message != parseRuntimeDiagnostic.Message ||
		parseDiagnostic.Path != parseRuntimeDiagnostic.Path ||
		parseDiagnostic.Docs != parseRuntimeDiagnostic.Docs ||
		parseDiagnostic.Remediation != parseRuntimeDiagnostic.Remediation ||
		parseDiagnostic.Recoverable != parseRuntimeDiagnostic.Recoverable ||
		parseDiagnostic.TopFrame != parseRuntimeDiagnostic.TopFrame ||
		parseDiagnostic.Consequence != parseRuntimeDiagnostic.Consequence ||
		parseDiagnostic.Fields["path"] != parseRuntimeDiagnostic.Fields["path"] ||
		parseDiagnostic.Fields["phase"] != parseRuntimeDiagnostic.Fields["phase"] ||
		parseDiagnostic.Fields["where"] != parseRuntimeDiagnostic.Fields["where"] {
		parseT.Fatalf("expected snapshot diagnostic to mirror runtime diagnostic, snapshot=%+v runtime=%+v", parseDiagnostic, parseRuntimeDiagnostic)
	}

	parseRuntimeLogs := runtime.GetLogs()
	if len(parseRuntimeLogs) == 0 {
		parseT.Fatal("expected runtime panic log")
	}
	parseRuntimeLog := parseRuntimeLogs[len(parseRuntimeLogs)-1]
	if len(parseSnapshot.Logs) == 0 {
		parseT.Fatal("expected panic log in snapshot")
	}
	parseEntry := parseSnapshot.Logs[len(parseSnapshot.Logs)-1]
	if parseEntry.Code != "GWC-RUNTIME-PANIC-STARTUP" {
		parseT.Fatalf("unexpected log payload: %+v", parseEntry)
	}
	if parseEntry.TopFrame == "" || parseEntry.Consequence == "" {
		parseT.Fatalf("expected wrapped panic log metadata, got %+v", parseEntry)
	}
	if parseEntry.Fields["path"] != "#app" || parseEntry.Fields["runtime"] == "" || parseEntry.Fields["top_frame"] == "" {
		parseT.Fatalf("expected panic fields to mirror into log entry, got %+v", parseEntry)
	}
	if parseEntry.Code != parseRuntimeLog.Code ||
		parseEntry.Message != parseRuntimeLog.Message ||
		parseEntry.Docs != parseRuntimeLog.Docs ||
		parseEntry.Remediation != parseRuntimeLog.Remediation ||
		parseEntry.Recoverable != parseRuntimeLog.Recoverable ||
		parseEntry.TopFrame != parseRuntimeLog.TopFrame ||
		parseEntry.Consequence != parseRuntimeLog.Consequence ||
		parseEntry.Fields["path"] != parseRuntimeLog.Fields["path"] ||
		parseEntry.Fields["runtime"] != parseRuntimeLog.Fields["runtime"] ||
		parseEntry.Fields["top_frame"] != parseRuntimeLog.Fields["top_frame"] {
		parseT.Fatalf("expected snapshot log to mirror runtime log, snapshot=%+v runtime=%+v", parseEntry, parseRuntimeLog)
	}
}

func TestSnapshotNowIncludesMultiClientInspection(parseT *testing.T) {
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

	parseSnapshot := SnapshotNow()
	if !parseSnapshot.MultiClient.Enabled || parseSnapshot.MultiClient.LocalPeerID != "storefront-1" || parseSnapshot.MultiClient.ResolvedTransport != "broadcast-channel" {
		parseT.Fatalf("unexpected multi-client snapshot header: %+v", parseSnapshot.MultiClient)
	}
	if len(parseSnapshot.MultiClient.Peers) != 1 || parseSnapshot.MultiClient.Peers[0].ID != "ops-1" || !parseSnapshot.MultiClient.Peers[0].Compatible {
		parseT.Fatalf("unexpected multi-client peer state: %+v", parseSnapshot.MultiClient.Peers)
	}
	if len(parseSnapshot.MultiClient.RecentTraffic) != 1 || parseSnapshot.MultiClient.RecentTraffic[0].CorrelationID != "req-1" {
		parseT.Fatalf("unexpected multi-client traffic state: %+v", parseSnapshot.MultiClient.RecentTraffic)
	}
	if len(parseSnapshot.MultiClient.FailedPublishes) != 1 || parseSnapshot.MultiClient.FailedPublishes[0].Code != "unauthorized" {
		parseT.Fatalf("unexpected multi-client failure state: %+v", parseSnapshot.MultiClient.FailedPublishes)
	}
}

func TestExportSnapshotJSONAndCompareSnapshots(parseT *testing.T) {
	parseBefore := Snapshot{
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
	parseAfter := Snapshot{
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

	parsePayload, parseErr := ExportSnapshotJSON(parseBefore)
	if parseErr != nil {
		parseT.Fatalf("ExportSnapshotJSON failed: %v", parseErr)
	}
	if len(parsePayload) == 0 || parsePayload[0] != '{' {
		parseT.Fatalf("expected JSON payload, got %q", string(parsePayload))
	}

	parseComparison, parseErr := CompareSnapshots(parseBefore, parseAfter)
	if parseErr != nil {
		parseT.Fatalf("CompareSnapshots failed: %v", parseErr)
	}
	if parseComparison.Equal {
		parseT.Fatal("expected snapshots to differ")
	}
	if parseComparison.PreviousFingerprint == "" || parseComparison.CurrentFingerprint == "" {
		parseT.Fatal("expected fingerprints to be populated")
	}
	if parseComparison.PreviousSize == 0 || parseComparison.CurrentSize == 0 {
		parseT.Fatal("expected serialized sizes to be populated")
	}
	if len(parseComparison.ChangedSections) == 0 {
		parseT.Fatal("expected changed sections to be reported")
	}
	parseWant := map[string]bool{"route": true, "multiClient": true, "boundaries": true, "coordination": true, "stats": true, "hydration": true, "logs": true}
	for _, parseSection := range parseComparison.ChangedSections {
		delete(parseWant, parseSection)
	}
	if len(parseWant) != 0 {
		parseT.Fatalf("expected route, multiClient, boundaries, coordination, stats, hydration, and logs to change, missing %v", parseWant)
	}
}

func TestMapHydrationIncludesDebugFields(parseT *testing.T) {
	parseMapped := mapHydration(runtime.HydrationDebugSnapshot{
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

	if parseMapped.CorrelationID != "hydrate-77" || parseMapped.MismatchCount != 2 || parseMapped.DiscardedNodeCount != 4 {
		parseT.Fatalf("expected hydration debug fields to map, got %+v", parseMapped)
	}
	if !parseMapped.Strict || !parseMapped.Failed || parseMapped.Failure == "" {
		parseT.Fatalf("expected hydration strict/failure metadata to map, got %+v", parseMapped)
	}
	if len(parseMapped.RecentMessages) != 2 || parseMapped.RecentMessages[0] != "hydration text mismatch at App > Hero" {
		parseT.Fatalf("expected hydration messages to map, got %+v", parseMapped.RecentMessages)
	}
	if parseSummary := hydrationSummary(parseMapped); parseSummary == nil {
		parseT.Fatal("expected hydration summary node")
	}
}

func TestSnapshotNowIncludesSerializationBoundaries(parseT *testing.T) {
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

	parseBootstrap := ui.SSRBootstrap{Data: map[string]interface{}{
		"legacy-message": "hello",
	}}
	if parseErr := ui.RegisterSessionBootstrapHint(&parseBootstrap, "viewer", map[string]string{"role": "operator"}); parseErr != nil {
		parseT.Fatalf("RegisterSessionBootstrapHint() error = %v", parseErr)
	}
	parseInspection, parseErr2 := InspectBootstrapBoundaries(parseBootstrap)
	if parseErr2 != nil {
		parseT.Fatalf("InspectBootstrapBoundaries() error = %v", parseErr2)
	}
	SetSerializationBoundaryInspection(parseInspection)

	parseSnapshot := SnapshotNow()
	if len(parseSnapshot.Boundaries.Entries) != 3 {
		parseT.Fatalf("expected bootstrap boundary entries in snapshot, got %+v", parseSnapshot.Boundaries)
	}
	if parseSnapshot.Boundaries.Entries[0].Name != "ssr.bootstrap" || parseSnapshot.Boundaries.Entries[0].Kind != "ssr-bootstrap" {
		parseT.Fatalf("unexpected root boundary entry: %+v", parseSnapshot.Boundaries.Entries[0])
	}
	if parseSummary := boundariesSummary(parseSnapshot.Boundaries); parseSummary == nil {
		parseT.Fatal("expected boundaries summary node")
	}
}

func TestSnapshotNowIncludesCoordinationInspection(parseT *testing.T) {
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
			Owner:       "sync-engine",
			State:       "retrying",
			Attempts:    1,
			MaxAttempts: 3,
		}},
		QueueEntries: []SyncQueueEntry{{
			ID:        "queue-1",
			Entity:    "order:42",
			Operation: "submit",
			Owner:     "sync-engine",
			State:     "queued",
		}},
		SyncHealth: []SyncHealthEntry{{
			Entity:     "order:42",
			Owner:      "sync-engine",
			Status:     "healthy",
			Version:    "v12",
			PendingOps: 1,
		}},
		Reconnect: ReconnectStatus{
			State:       "reconnecting",
			Transport:   "broadcast-channel",
			Attempts:    2,
			MaxAttempts: 5,
			NextRetryAt: time.Unix(21, 0).UTC(),
			LastChange:  time.Unix(20, 0).UTC(),
		},
		Conflict: ConflictState{
			Entity:     "order:42",
			Owner:      "sync-engine",
			Status:     "pending",
			Strategy:   "server-wins",
			DetectedAt: time.Unix(19, 0).UTC(),
		},
		LastReplayError: "HTTP 409 conflict",
	})

	parseSnapshot := SnapshotNow()
	if len(parseSnapshot.Coordination.Workers) != 1 || len(parseSnapshot.Coordination.SyncEvents) != 1 || len(parseSnapshot.Coordination.Replay) != 1 || len(parseSnapshot.Coordination.QueueEntries) != 1 || len(parseSnapshot.Coordination.SyncHealth) != 1 {
		parseT.Fatalf("expected coordination state in snapshot, got %+v", parseSnapshot.Coordination)
	}
	if parseSnapshot.Coordination.Replay[0].Owner != "sync-engine" || parseSnapshot.Coordination.Reconnect.State != "reconnecting" || parseSnapshot.Coordination.Conflict.Status != "pending" || parseSnapshot.Coordination.LastReplayError != "HTTP 409 conflict" {
		parseT.Fatalf("expected extended coordination state in snapshot, got %+v", parseSnapshot.Coordination)
	}
	if parseSummary := coordinationSummary(parseSnapshot.Coordination); parseSummary == nil {
		parseT.Fatal("expected coordination summary node")
	}
}

func TestCollectOverlayIssuesHighlightsActionableFailures(parseT *testing.T) {
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

	parseSnapshot := SnapshotNow()
	parseIssues := collectOverlayIssues(parseSnapshot)
	if len(parseIssues) == 0 {
		parseT.Fatalf("expected at least one actionable overlay issue, got %+v", parseIssues)
	}
	isParseFoundLoaderIssue := false
	for _, parseIssue := range parseIssues {
		if parseIssue.Code == "GWC-ROUTER-LOADER-FAILED" {
			isParseFoundLoaderIssue = true
		}
	}
	if !isParseFoundLoaderIssue {
		parseT.Fatalf("expected loader failure issue in overlay, got %+v", parseIssues)
	}
}

func TestMatchingErrorOverlayActionsFiltersByCodeAndSource(parseT *testing.T) {
	parseIssue := ErrorOverlayIssue{
		Source: "router",
		Code:   "GWC-ROUTER-LOADER-FAILED",
	}
	parseActions := []ErrorOverlayAction{
		{Label: "Retry loader", MatchCodes: []string{"GWC-ROUTER-LOADER-FAILED"}},
		{Label: "Reveal launcher", MatchSources: []string{"runtime"}},
		{Label: "Always"},
	}

	parseMatched := matchingErrorOverlayActions(parseIssue, parseActions)
	if len(parseMatched) != 2 {
		parseT.Fatalf("expected two matching overlay actions, got %+v", parseMatched)
	}
	if parseMatched[0].Label != "Retry loader" || parseMatched[1].Label != "Always" {
		parseT.Fatalf("unexpected overlay action match order: %+v", parseMatched)
	}
}

func TestMapInspectionFineGrainedMetadata(parseT *testing.T) {
	parseNode := mapNode(&runtime.FiberSnapshot{
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
	if parseNode == nil {
		parseT.Fatal("expected mapped node")
	}
	if !parseNode.FineGrained {
		parseT.Fatal("expected fine-grained flag to map")
	}
	if parseNode.ReactiveSource != "count" {
		parseT.Fatalf("expected reactive source count, got %q", parseNode.ReactiveSource)
	}
	if parseNode.UpdateOrigin != "fine-grained" {
		parseT.Fatalf("expected update origin fine-grained, got %q", parseNode.UpdateOrigin)
	}
	if len(parseNode.Hooks) != 1 || parseNode.Hooks[0].Slot != 1 || parseNode.Hooks[0].Dependencies != `"theme", true` || parseNode.Hooks[0].Status != "cleanup=registered epoch=4" {
		parseT.Fatalf("expected hook inspection metadata to map, got %+v", parseNode.Hooks)
	}

	parseStats := mapStats(runtime.InspectionStats{FineGrainedFibers: 1})
	if parseStats.FineGrainedFibers != 1 {
		parseT.Fatalf("expected fine-grained fiber count to map, got %d", parseStats.FineGrainedFibers)
	}

	parseProfiling := mapProfiling(runtime.ProfilingSnapshot{ScheduledGranularMarks: 4, FineGrainedCommits: 3, FineGrainedDescendantHostCommits: 6, FineGrainedDescendantTextCommits: 2})
	if parseProfiling.ScheduledGranularMarks != 4 {
		parseT.Fatalf("expected granular marks to map, got %d", parseProfiling.ScheduledGranularMarks)
	}
	if parseProfiling.FineGrainedCommits != 3 {
		parseT.Fatalf("expected fine-grained commits to map, got %d", parseProfiling.FineGrainedCommits)
	}
	if parseProfiling.FineGrainedDescendantHostCommits != 6 {
		parseT.Fatalf("expected descendant host commits to map, got %d", parseProfiling.FineGrainedDescendantHostCommits)
	}
	if parseProfiling.FineGrainedDescendantTextCommits != 2 {
		parseT.Fatalf("expected descendant text commits to map, got %d", parseProfiling.FineGrainedDescendantTextCommits)
	}

	parseExtended := mapProfiling(runtime.ProfilingSnapshot{
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
			WASMTransferBytes:          1024,
			WASMDecodedBytes:           2048,
			BootstrapDecodedBytes:      512,
			CacheWarmupDurationNs:      3,
			ServiceWorkerOverheadNs:    2,
			InitialRouteDataBytes:      144,
			HydrationDurationNs:        14,
			StartupCommitDurationNs:    6,
			FirstInteractionDurationNs: 42,
			FirstInteractionCaptured:   true,
			FirstInteractionEvent:      "event",
			RouteBudgets: []runtime.RouteStartupBudgetSnapshot{{
				RouteFamily:                       "/reports/*",
				LastRoutePath:                     "/reports/7",
				SampleCount:                       3,
				AverageBootstrapReadDurationNs:    8,
				AverageWASMTransferBytes:          1000,
				AverageWASMDecodedBytes:           2100,
				AverageBootstrapDecodedBytes:      500,
				AverageCacheWarmupDurationNs:      4,
				AverageServiceWorkerOverheadNs:    3,
				AverageInitialRouteDataBytes:      140,
				AverageHydrationDurationNs:        13,
				AverageStartupCommitDurationNs:    5,
				AverageFirstInteractionDurationNs: 37,
			}},
		},
	})
	if parseExtended.PhaseTotals.CommitDurationNs != 5 || parseExtended.PhaseTotals.DiffDurationNs != 7 {
		parseT.Fatalf("expected phase totals to map, got %+v", parseExtended.PhaseTotals)
	}
	if len(parseExtended.RecentEvents) != 1 || parseExtended.RecentEvents[0].Domain != "router" || parseExtended.RecentEvents[0].Fields["mode"] != "push" {
		parseT.Fatalf("expected profiling events to map, got %+v", parseExtended.RecentEvents)
	}
	if len(parseExtended.ComponentRenders) != 1 || parseExtended.ComponentRenders[0].Name != "Dashboard" || parseExtended.ComponentRenders[0].TriggerCounts["hook"] != 3 {
		parseT.Fatalf("expected component render traces to map, got %+v", parseExtended.ComponentRenders)
	}
	if len(parseExtended.HotBranches) != 1 || parseExtended.HotBranches[0].RenderDurationNs != 10 || parseExtended.HotBranches[0].DiffDurationNs != 4 {
		parseT.Fatalf("expected hot branch render/diff attribution to map, got %+v", parseExtended.HotBranches)
	}
	if len(parseExtended.FlamegraphFrames) != 1 || parseExtended.FlamegraphFrames[0].Depth != 1 || parseExtended.FlamegraphFrames[0].DurationNs != 40 {
		parseT.Fatalf("expected flamegraph frames to map, got %+v", parseExtended.FlamegraphFrames)
	}
	if parseExtended.Startup.Mode != "hydrate" || !parseExtended.Startup.FirstInteractionCaptured || parseExtended.Startup.FirstInteractionDurationNs != 42 {
		parseT.Fatalf("expected startup profiling to map, got %+v", parseExtended.Startup)
	}
	if parseExtended.Startup.WASMTransferBytes != 1024 || parseExtended.Startup.BootstrapDecodedBytes != 512 || parseExtended.Startup.InitialRouteDataBytes != 144 {
		parseT.Fatalf("expected startup cost attribution to map, got %+v", parseExtended.Startup)
	}
	if len(parseExtended.Startup.RouteBudgets) != 1 || parseExtended.Startup.RouteBudgets[0].RouteFamily != "/reports/*" || parseExtended.Startup.RouteBudgets[0].AverageFirstInteractionDurationNs != 37 {
		parseT.Fatalf("expected route startup budgets to map, got %+v", parseExtended.Startup.RouteBudgets)
	}
}

func TestRouteSummaryMappingIncludesStackLoadersAndRedirect(parseT *testing.T) {
	parseSnapshot := Snapshot{
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

	if parseSummary := routeSummary(parseSnapshot.Route); parseSummary == nil {
		parseT.Fatal("expected route summary node")
	}
	if parseSnapshot.Route.Stack[1].HasLoader != true || parseSnapshot.Route.LastRedirect.To != "/login" || parseSnapshot.Route.Metadata.Title != "Report" {
		parseT.Fatalf("expected route debug mapping data to round-trip, got %+v", parseSnapshot.Route)
	}
}
