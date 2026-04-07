//go:build !js || !wasm
// +build !js !wasm

package devtools

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/plugin"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestSnapshotStubsReturnEmptyValuesOnNative(parseT *testing.T) {
	if parseGot := SnapshotNow(); parseGot.Tree != nil || len(parseGot.Cache) != 0 || parseGot.Route.Path != "" {
		parseT.Fatalf("SnapshotNow() = %+v, want empty snapshot", parseGot)
	}
	if parseGot2 := UseSnapshot(250 * time.Millisecond); parseGot2.Tree != nil || len(parseGot2.Cache) != 0 || parseGot2.Route.Path != "" {
		parseT.Fatalf("UseSnapshot() = %+v, want empty snapshot", parseGot2)
	}
	if parsePanel := Panel(PanelProps{Title: "Inspector", RefreshInterval: time.Second}); parsePanel != nil {
		parseT.Fatalf("Panel() = %#v, want nil", parsePanel)
	}
	if parseOverlay := ErrorOverlay(ErrorOverlayProps{Title: "Errors", RefreshInterval: time.Second}); parseOverlay != nil {
		parseT.Fatalf("ErrorOverlay() = %#v, want nil", parseOverlay)
	}
}

func TestMultiClientInspectionRoundTripsClonedState(parseT *testing.T) {
	parseT.Cleanup(ResetMultiClientInspection)
	parseOriginal := MultiClient{
		Enabled:           true,
		LocalPeerID:       "peer-a",
		ResolvedTransport: "broadcast-channel",
		AuthorityView:     map[string]string{"role": "leader"},
		Peers: []MultiClientPeer{{
			ID:         "peer-a",
			Encodings:  []string{"json"},
			Topics:     []string{"chat"},
			Compatible: true,
		}},
		RecentTraffic:   []MultiClientTraffic{{Direction: "outbound", Topic: "chat"}},
		FailedPublishes: []MultiClientFailure{{Op: "publish", Code: "timeout"}},
	}

	SetMultiClientInspection(parseOriginal)
	parseCloned := InspectMultiClient()
	if parseCloned.LocalPeerID != "peer-a" || !parseCloned.Enabled {
		parseT.Fatalf("unexpected cloned state: %+v", parseCloned)
	}

	parseOriginal.AuthorityView["role"] = "follower"
	parseOriginal.Peers[0].Topics[0] = "mutated"
	parseOriginal.RecentTraffic[0].Topic = "changed"
	parseOriginal.FailedPublishes[0].Code = "changed"

	parseAfterMutation := InspectMultiClient()
	if parseAfterMutation.AuthorityView["role"] != "leader" {
		parseT.Fatalf("authority view should be cloned, got %+v", parseAfterMutation.AuthorityView)
	}
	if parseAfterMutation.Peers[0].Topics[0] != "chat" {
		parseT.Fatalf("peer topics should be cloned, got %+v", parseAfterMutation.Peers)
	}
	if parseAfterMutation.RecentTraffic[0].Topic != "chat" {
		parseT.Fatalf("recent traffic should be cloned, got %+v", parseAfterMutation.RecentTraffic)
	}
	if parseAfterMutation.FailedPublishes[0].Code != "timeout" {
		parseT.Fatalf("failed publishes should be cloned, got %+v", parseAfterMutation.FailedPublishes)
	}

	ResetMultiClientInspection()
	if parseGot := InspectMultiClient(); parseGot.Enabled || parseGot.LocalPeerID != "" || len(parseGot.Peers) != 0 || len(parseGot.AuthorityView) != 0 {
		parseT.Fatalf("ResetMultiClientInspection() left residual state: %+v", parseGot)
	}
}

func TestCompareSnapshotsDetectsChangedSections(parseT *testing.T) {
	parsePrevious := Snapshot{
		Route: Route{Path: "/inbox"},
		Stats: Stats{TotalFibers: 10},
		Boundaries: BoundaryInspection{Entries: []Boundary{{
			Name:      "ssr.bootstrap",
			Kind:      "ssr-bootstrap",
			Direction: "server-to-client",
			Status:    "observed",
			SizeBytes: 100,
		}}},
		Hydration: HydrationDebug{
			CorrelationID: "hydrate-a",
			MismatchCount: 1,
		},
	}
	parseCurrent := Snapshot{
		Route: Route{Path: "/settings"},
		Stats: Stats{TotalFibers: 11},
		Boundaries: BoundaryInspection{Entries: []Boundary{{
			Name:      "worker.search",
			Kind:      "worker",
			Direction: "client-to-worker",
			Status:    "rejected",
			SizeBytes: 220,
		}}},
		Coordination: Coordination{
			Workers: []WorkerJob{{
				Name:    "worker.search",
				Status:  "running",
				Running: true,
			}},
		},
		Extensions: []ExtensionSection{{
			Name:    "Companion",
			Summary: map[string]string{"state": "ready"},
		}},
		Hydration: HydrationDebug{
			CorrelationID: "hydrate-b",
			MismatchCount: 3,
		},
		Logs: []Log{{Message: "updated"}},
	}

	parseComparison, parseErr := CompareSnapshots(parsePrevious, parseCurrent)
	if parseErr != nil {
		parseT.Fatalf("CompareSnapshots(): %v", parseErr)
	}
	if parseComparison.Equal {
		parseT.Fatalf("expected snapshots to differ: %+v", parseComparison)
	}
	if parseComparison.PreviousFingerprint == "" || parseComparison.CurrentFingerprint == "" || parseComparison.PreviousFingerprint == parseComparison.CurrentFingerprint {
		parseT.Fatalf("expected distinct fingerprints, got %+v", parseComparison)
	}
	parseChanged := strings.Join(parseComparison.ChangedSections, ",")
	for _, parseSection := range []string{"route", "stats", "boundaries", "coordination", "extensions", "hydration", "logs"} {
		if !strings.Contains(parseChanged, parseSection) {
			parseT.Fatalf("expected changed sections to contain %q, got %v", parseSection, parseComparison.ChangedSections)
		}
	}

	parseEqualComparison, parseErr := CompareSnapshots(parseCurrent, parseCurrent)
	if parseErr != nil {
		parseT.Fatalf("CompareSnapshots(equal): %v", parseErr)
	}
	if !parseEqualComparison.Equal || len(parseEqualComparison.ChangedSections) != 0 {
		parseT.Fatalf("expected equal comparison without changed sections, got %+v", parseEqualComparison)
	}
}

func TestSerializationBoundaryInspectionRoundTripsClonedState(parseT *testing.T) {
	parseT.Cleanup(ResetSerializationBoundaryInspection)

	parseOriginal := BoundaryInspection{Entries: []Boundary{{
		Name:       "worker.search",
		Kind:       "worker",
		Direction:  "client-to-worker",
		Transport:  "worker",
		Encoding:   "json",
		Status:     "observed",
		SizeBytes:  512,
		Notes:      []string{"kind=query"},
		Rejected:   []string{"payload too large"},
		Downgraded: []string{"blob handle"},
	}}}

	SetSerializationBoundaryInspection(parseOriginal)
	parseCloned := InspectSerializationBoundaries()
	if len(parseCloned.Entries) != 1 || parseCloned.Entries[0].Name != "worker.search" {
		parseT.Fatalf("unexpected cloned boundary state: %+v", parseCloned)
	}

	parseOriginal.Entries[0].Name = "mutated"
	parseOriginal.Entries[0].Notes[0] = "changed"
	parseOriginal.Entries[0].Rejected[0] = "changed"

	parseAfterMutation := InspectSerializationBoundaries()
	if parseAfterMutation.Entries[0].Name != "worker.search" || parseAfterMutation.Entries[0].Notes[0] != "kind=query" || parseAfterMutation.Entries[0].Rejected[0] != "payload too large" {
		parseT.Fatalf("expected boundary state to be cloned, got %+v", parseAfterMutation)
	}

	ResetSerializationBoundaryInspection()
	if parseGot := InspectSerializationBoundaries(); len(parseGot.Entries) != 0 {
		parseT.Fatalf("ResetSerializationBoundaryInspection() left residual state: %+v", parseGot)
	}
}

func TestInspectBootstrapBoundariesSummarizesPayloads(parseT *testing.T) {
	parseBootstrap := ui.SSRBootstrap{Data: map[string]interface{}{
		"legacy-message": "hello",
	}}
	if parseErr := ui.RegisterRouteBootstrapData(&parseBootstrap, "catalog", "/products", map[string]string{"sku": "atlas-1"}); parseErr != nil {
		parseT.Fatalf("RegisterRouteBootstrapData() error = %v", parseErr)
	}
	if parseErr2 := ui.RegisterSessionBootstrapHint(&parseBootstrap, "viewer", map[string]string{"role": "operator"}); parseErr2 != nil {
		parseT.Fatalf("RegisterSessionBootstrapHint() error = %v", parseErr2)
	}

	parseInspection, parseErr3 := InspectBootstrapBoundaries(parseBootstrap)
	if parseErr3 != nil {
		parseT.Fatalf("InspectBootstrapBoundaries() error = %v", parseErr3)
	}
	if len(parseInspection.Entries) != 4 {
		parseT.Fatalf("expected root plus three payload entries, got %+v", parseInspection.Entries)
	}
	parseRoot := parseInspection.Entries[0]
	if parseRoot.Name != "ssr.bootstrap" || parseRoot.Kind != "ssr-bootstrap" || parseRoot.SizeBytes <= 0 || parseRoot.InlineBytes <= 0 || parseRoot.BinaryBytes <= 0 {
		parseT.Fatalf("unexpected bootstrap root boundary: %+v", parseRoot)
	}

	isParseFoundLegacy := false
	isParseFoundRoute := false
	for _, parseEntry := range parseInspection.Entries[1:] {
		switch parseEntry.Name {
		case "legacy-message":
			isParseFoundLegacy = true
			if parseEntry.Status != "downgraded" || len(parseEntry.Downgraded) != 1 {
				parseT.Fatalf("expected legacy payload to be marked downgraded, got %+v", parseEntry)
			}
		case "/products::catalog":
			isParseFoundRoute = true
			if parseEntry.Scope != "route" || parseEntry.Encoding != "json" || parseEntry.SizeBytes <= 0 {
				parseT.Fatalf("expected route payload metadata to map, got %+v", parseEntry)
			}
		}
	}
	if !isParseFoundLegacy || !isParseFoundRoute {
		parseT.Fatalf("expected both legacy and route payload entries, got %+v", parseInspection.Entries)
	}
}

func TestCoordinationInspectionRoundTripsClonedState(parseT *testing.T) {
	parseT.Cleanup(ResetCoordinationInspection)

	parseOriginal := Coordination{
		Workers: []WorkerJob{{
			Name:      "search-index",
			Status:    "running",
			RequestID: "req-1",
			Running:   true,
		}},
		SyncEvents: []SyncEvent{{
			Channel:   "prefs",
			Transport: "broadcast-channel",
			Direction: "outbound",
			Topic:     "theme",
			Status:    "sent",
			Timestamp: time.Unix(10, 0).UTC(),
		}},
		Replay: []ReplayEntry{{
			ID:            "mut-1",
			Kind:          "order.submit",
			Owner:         "sync-engine",
			State:         "retrying",
			Attempts:      1,
			MaxAttempts:   3,
			NextAttemptAt: time.Unix(20, 0).UTC(),
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
		},
		Conflict: ConflictState{
			Entity:   "order:42",
			Owner:    "tab-2",
			Status:   "pending",
			Strategy: "server-wins",
		},
		LastReplayError: "HTTP 409 conflict",
	}

	SetCoordinationInspection(parseOriginal)
	parseCloned := InspectCoordination()
	if len(parseCloned.Workers) != 1 || len(parseCloned.SyncEvents) != 1 || len(parseCloned.Replay) != 1 || len(parseCloned.QueueEntries) != 1 || len(parseCloned.SyncHealth) != 1 {
		parseT.Fatalf("unexpected coordination clone: %+v", parseCloned)
	}
	if parseCloned.Reconnect.State != "reconnecting" || parseCloned.Conflict.Entity != "order:42" || parseCloned.LastReplayError != "HTTP 409 conflict" {
		parseT.Fatalf("expected extended coordination fields to clone, got %+v", parseCloned)
	}

	parseOriginal.Workers[0].Name = "mutated"
	parseOriginal.SyncEvents[0].Topic = "changed"
	parseOriginal.Replay[0].State = "dead"
	parseOriginal.QueueEntries[0].Entity = "order:dead"
	parseOriginal.SyncHealth[0].Status = "stale"
	parseOriginal.Reconnect.State = "offline"
	parseOriginal.Conflict.Status = "resolved"
	parseOriginal.LastReplayError = "mutated"

	parseAfterMutation := InspectCoordination()
	if parseAfterMutation.Workers[0].Name != "search-index" || parseAfterMutation.SyncEvents[0].Topic != "theme" || parseAfterMutation.Replay[0].State != "retrying" || parseAfterMutation.QueueEntries[0].Entity != "order:42" || parseAfterMutation.SyncHealth[0].Status != "healthy" || parseAfterMutation.Reconnect.State != "reconnecting" || parseAfterMutation.Conflict.Status != "pending" || parseAfterMutation.LastReplayError != "HTTP 409 conflict" {
		parseT.Fatalf("expected coordination state to be cloned, got %+v", parseAfterMutation)
	}

	ResetCoordinationInspection()
	if parseGot := InspectCoordination(); len(parseGot.Workers) != 0 || len(parseGot.SyncEvents) != 0 || len(parseGot.Replay) != 0 || len(parseGot.QueueEntries) != 0 || len(parseGot.SyncHealth) != 0 || parseGot.Reconnect != (ReconnectStatus{}) || parseGot.Conflict != (ConflictState{}) || parseGot.LastReplayError != "" {
		parseT.Fatalf("ResetCoordinationInspection() left residual state: %+v", parseGot)
	}
}

func TestErrorOverlayActionsRoundTripClonedState(parseT *testing.T) {
	parseT.Cleanup(ResetErrorOverlayActions)

	var isInvoked bool
	parseOriginal := []ErrorOverlayAction{{
		Label:        "Retry loader",
		MatchCodes:   []string{"GWC-ROUTER-LOADER-FAILED"},
		MatchSources: []string{"router"},
		Run: func(ErrorOverlayActionContext) {
			isInvoked = true
		},
	}}

	SetErrorOverlayActions(parseOriginal)
	parseCloned := InspectErrorOverlayActions()
	if len(parseCloned) != 1 || parseCloned[0].Label != "Retry loader" || len(parseCloned[0].MatchCodes) != 1 {
		parseT.Fatalf("unexpected overlay action clone: %+v", parseCloned)
	}

	parseOriginal[0].Label = "Mutated"
	parseOriginal[0].MatchCodes[0] = "changed"

	parseAfterMutation := InspectErrorOverlayActions()
	if parseAfterMutation[0].Label != "Retry loader" || parseAfterMutation[0].MatchCodes[0] != "GWC-ROUTER-LOADER-FAILED" {
		parseT.Fatalf("expected overlay actions to be cloned, got %+v", parseAfterMutation)
	}

	parseAfterMutation[0].Run(ErrorOverlayActionContext{})
	if !isInvoked {
		parseT.Fatal("expected cloned overlay action handler to remain callable")
	}

	ResetErrorOverlayActions()
	if parseGot := InspectErrorOverlayActions(); len(parseGot) != 0 {
		parseT.Fatalf("ResetErrorOverlayActions() left residual state: %+v", parseGot)
	}
}

func TestTraceCaptureExportImportAndReplay(parseT *testing.T) {
	parseT.Cleanup(ClearTraceReplay)

	parseCapture := TraceCapture{
		Label:      "checkout-click",
		CapturedAt: "2026-03-25T12:00:00Z",
		Snapshot: Snapshot{
			Route: Route{Path: "/checkout"},
			Logs:  []Log{{Domain: "router", Message: "navigated"}},
		},
	}

	parsePayload, parseErr := ExportTraceCaptureJSON(parseCapture)
	if parseErr != nil {
		parseT.Fatalf("ExportTraceCaptureJSON() error = %v", parseErr)
	}
	parseDecoded, parseErr := ImportTraceCaptureJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ImportTraceCaptureJSON() error = %v", parseErr)
	}
	if parseDecoded.Label != "checkout-click" || parseDecoded.Snapshot.Route.Path != "/checkout" || len(parseDecoded.Snapshot.Logs) != 1 {
		parseT.Fatalf("unexpected decoded trace capture: %+v", parseDecoded)
	}

	SetTraceReplay(parseDecoded)
	parseReplayed, parseOk := CurrentTraceReplay()
	if !parseOk || parseReplayed.Label != "checkout-click" || parseReplayed.Snapshot.Route.Path != "/checkout" {
		parseT.Fatalf("expected trace replay to round-trip, got replay=%+v ok=%t", parseReplayed, parseOk)
	}
	if parseSnapshot := SnapshotNow(); parseSnapshot.Route.Path != "/checkout" || len(parseSnapshot.Logs) != 1 {
		parseT.Fatalf("expected SnapshotNow() to return replayed trace, got %+v", parseSnapshot)
	}

	ClearTraceReplay()
	if _, parseOk2 := CurrentTraceReplay(); parseOk2 {
		parseT.Fatal("expected replay to be cleared")
	}
}

func TestBugCaptureBundleExportImportAndReplay(parseT *testing.T) {
	parseT.Cleanup(ClearTraceReplay)

	parseBundle := BugCaptureBundle{
		Version:    1,
		Label:      "startup-failure",
		CapturedAt: "2026-03-25T12:05:00Z",
		Trace: TraceCapture{
			Label:      "startup-failure",
			CapturedAt: "2026-03-25T12:05:00Z",
			Snapshot: Snapshot{
				Route: Route{Path: "/"},
				Diagnostics: []Diagnostic{{
					Code:     "GWC-RUNTIME-PANIC-STARTUP",
					Message:  "startup boom",
					Severity: SeverityError,
				}},
			},
		},
	}

	parsePayload, parseErr := ExportBugCaptureBundleJSON(parseBundle)
	if parseErr != nil {
		parseT.Fatalf("ExportBugCaptureBundleJSON() error = %v", parseErr)
	}
	parseDecoded, parseErr := ImportBugCaptureBundleJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ImportBugCaptureBundleJSON() error = %v", parseErr)
	}
	if parseDecoded.Label != "startup-failure" || parseDecoded.Trace.Snapshot.Route.Path != "/" || len(parseDecoded.Trace.Snapshot.Diagnostics) != 1 {
		parseT.Fatalf("unexpected decoded bug bundle: %+v", parseDecoded)
	}

	ReplayBugCaptureBundle(parseDecoded)
	parseSnapshot := SnapshotNow()
	if parseSnapshot.Route.Path != "/" || len(parseSnapshot.Diagnostics) != 1 || parseSnapshot.Diagnostics[0].Code != "GWC-RUNTIME-PANIC-STARTUP" {
		parseT.Fatalf("expected replayed bug bundle snapshot, got %+v", parseSnapshot)
	}
}

func TestCaptureTraceAndBugBundlePopulateMetadata(parseT *testing.T) {
	parseT.Cleanup(ClearTraceReplay)

	parseTrace := CaptureTrace("  refresh-route  ")
	if parseTrace.Label != "refresh-route" {
		parseT.Fatalf("expected trace label to be trimmed, got %q", parseTrace.Label)
	}
	if parseTrace.CapturedAt == "" {
		parseT.Fatalf("expected trace capture timestamp")
	}
	if _, parseErr := time.Parse(time.RFC3339Nano, parseTrace.CapturedAt); parseErr != nil {
		parseT.Fatalf("expected RFC3339 trace timestamp, got %q: %v", parseTrace.CapturedAt, parseErr)
	}

	parseBundle := CaptureBugBundle("  checkout-bug  ")
	if parseBundle.Version != currentBugCaptureBundleVersion {
		parseT.Fatalf("expected bundle version %d, got %d", currentBugCaptureBundleVersion, parseBundle.Version)
	}
	if parseBundle.Label != "checkout-bug" || parseBundle.Trace.Label != "checkout-bug" {
		parseT.Fatalf("expected trimmed bundle/trace labels, got bundle=%q trace=%q", parseBundle.Label, parseBundle.Trace.Label)
	}
	if parseBundle.CapturedAt == "" || parseBundle.Trace.CapturedAt == "" {
		parseT.Fatalf("expected bundle and trace timestamps, got bundle=%q trace=%q", parseBundle.CapturedAt, parseBundle.Trace.CapturedAt)
	}
	if _, parseErr2 := time.Parse(time.RFC3339Nano, parseBundle.CapturedAt); parseErr2 != nil {
		parseT.Fatalf("expected RFC3339 bundle timestamp, got %q: %v", parseBundle.CapturedAt, parseErr2)
	}
}

func TestExtensionSectionsRoundTripClonedState(parseT *testing.T) {
	parseT.Cleanup(ResetExtensionSections)

	parseOriginal := []ExtensionSection{{
		Name:    "Companion",
		Summary: map[string]string{"state": "ready"},
		Lines:   []string{"line one"},
	}}
	SetExtensionSections(parseOriginal)
	parseCloned := InspectExtensionSections()
	if len(parseCloned) != 1 || parseCloned[0].Name != "Companion" || parseCloned[0].Summary["state"] != "ready" {
		parseT.Fatalf("unexpected extension-section clone: %+v", parseCloned)
	}

	parseOriginal[0].Name = "Mutated"
	parseOriginal[0].Summary["state"] = "changed"
	parseOriginal[0].Lines[0] = "changed"

	parseAfterMutation := InspectExtensionSections()
	if parseAfterMutation[0].Name != "Companion" || parseAfterMutation[0].Summary["state"] != "ready" || parseAfterMutation[0].Lines[0] != "line one" {
		parseT.Fatalf("expected extension sections to be cloned, got %+v", parseAfterMutation)
	}

	ResetExtensionSections()
	if parseGot := InspectExtensionSections(); len(parseGot) != 0 {
		parseT.Fatalf("ResetExtensionSections() left residual state: %+v", parseGot)
	}
}

func TestApplyHostExtensionsMapsAndRestoresHostContributions(parseT *testing.T) {
	parseT.Cleanup(ResetExtensionSections)
	parseT.Cleanup(ResetErrorOverlayActions)

	parseHost := plugin.NewHost(plugin.HostOptions{Capabilities: []plugin.Capability{plugin.CapabilityDevtools}})
	if parseErr := parseHost.AddDevtoolsSectionProvider(func() plugin.DevtoolsSection {
		return plugin.DevtoolsSection{
			Name:    "Companion",
			Summary: map[string]string{"state": "ready"},
			Lines:   []string{"line one"},
		}
	}); parseErr != nil {
		parseT.Fatalf("AddDevtoolsSectionProvider() error = %v", parseErr)
	}

	isParseInvoked := false
	if parseErr2 := parseHost.AddDevtoolsActionProvider(func() []plugin.DevtoolsAction {
		return []plugin.DevtoolsAction{{
			Label:        "Retry loader",
			MatchCodes:   []string{"GWC-ROUTER-LOADER-FAILED"},
			MatchSources: []string{"router"},
			Run: func(parseContext plugin.DevtoolsActionContext) {
				isParseInvoked = true
				if parseContext.Host != parseHost || parseContext.IssueCode != "GWC-ROUTER-LOADER-FAILED" || parseContext.IssueSource != "router" || parseContext.IssuePath != "/reports" || parseContext.IssueMessage != "route loader failed" || parseContext.IssueDocs != "docs/router" || parseContext.IssueTopFrame != "renderReports" {
					parseT.Fatalf("unexpected mapped host action context: %+v", parseContext)
				}
			},
		}}
	}); parseErr2 != nil {
		parseT.Fatalf("AddDevtoolsActionProvider() error = %v", parseErr2)
	}

	SetExtensionSections([]ExtensionSection{{Name: "Existing", Summary: map[string]string{"state": "baseline"}}})
	SetErrorOverlayActions([]ErrorOverlayAction{{Label: "Existing", Run: func(ErrorOverlayActionContext) {}}})

	parseCleanup := ApplyHostExtensions(parseHost)
	parseSections := InspectExtensionSections()
	if len(parseSections) != 1 || parseSections[0].Name != "Companion" || parseSections[0].Summary["state"] != "ready" || parseSections[0].Lines[0] != "line one" {
		parseT.Fatalf("InspectExtensionSections() = %+v, want mapped host section", parseSections)
	}

	parseActions := InspectErrorOverlayActions()
	if len(parseActions) != 1 || parseActions[0].Label != "Retry loader" || parseActions[0].MatchCodes[0] != "GWC-ROUTER-LOADER-FAILED" {
		parseT.Fatalf("InspectErrorOverlayActions() = %+v, want mapped host action", parseActions)
	}
	parseActions[0].Run(ErrorOverlayActionContext{
		Issue: ErrorOverlayIssue{
			Source:   "router",
			Code:     "GWC-ROUTER-LOADER-FAILED",
			Message:  "route loader failed",
			TopFrame: "renderReports",
			Path:     "/reports",
			Docs:     "docs/router",
		},
	})
	if !isParseInvoked {
		parseT.Fatal("expected mapped host overlay action to remain callable")
	}

	parseCleanup()
	parseRestoredSections := InspectExtensionSections()
	if len(parseRestoredSections) != 1 || parseRestoredSections[0].Name != "Existing" || parseRestoredSections[0].Summary["state"] != "baseline" {
		parseT.Fatalf("expected cleanup to restore prior extension sections, got %+v", parseRestoredSections)
	}
	parseRestoredActions := InspectErrorOverlayActions()
	if len(parseRestoredActions) != 1 || parseRestoredActions[0].Label != "Existing" {
		parseT.Fatalf("expected cleanup to restore prior overlay actions, got %+v", parseRestoredActions)
	}
}

func TestSupportDiagnosticBundleExportRedactsSensitiveValues(parseT *testing.T) {
	parseBundle := BugCaptureBundle{
		Version:    currentBugCaptureBundleVersion,
		Label:      "support-case",
		CapturedAt: "2026-03-25T13:00:00Z",
		Trace: TraceCapture{
			Label:      "support-case",
			CapturedAt: "2026-03-25T13:00:00Z",
			Snapshot: Snapshot{
				Route: Route{
					Path:  "/workspace",
					Query: map[string][]string{"token": {"abc123"}, "view": {"kanban"}},
					Params: map[string]string{
						"workspaceID": "ws-17",
					},
					Loaders: []RouteLoader{{
						Key:   "current-user",
						Path:  "/api/me?session_token=xyz",
						Error: "authorization=Bearer top-secret",
					}},
					Metadata: RouteMetadata{
						CanonicalURL: "https://example.com/workspace?api_key=super-secret",
					},
				},
				Cache: []CacheEntry{{
					Key:       "https://example.com/data?token=abc123",
					LastError: "request failed: cookie=session123",
				}},
				Tree: &Node{
					Name: "Dashboard",
					Hooks: []Hook{{
						Kind:  "state",
						Value: "token=abc123",
					}},
				},
				Hydration: HydrationDebug{
					Failure:        "https://example.com/hydrate?password=hunter2",
					RecentMessages: []string{"Authorization: Bearer top-secret"},
				},
				Diagnostics: []Diagnostic{{
					Code:    "GWC-HYDRATION-TEXT-MISMATCH",
					Message: "token=abc123",
					Fields: map[string]string{
						"authorization": "Bearer top-secret",
						"route":         "/workspace",
					},
				}},
				Logs: []Log{{
					Message: "Cookie=session123",
					Fields: map[string]string{
						"session_id": "session123",
					},
				}},
				Extensions: []ExtensionSection{{
					Name:    "Companion",
					Summary: map[string]string{"api_key": "super-secret"},
					Lines:   []string{"password=hunter2"},
				}},
			},
		},
	}

	parsePayload, parseErr := ExportSupportDiagnosticBundleJSON(parseBundle)
	if parseErr != nil {
		parseT.Fatalf("ExportSupportDiagnosticBundleJSON() error = %v", parseErr)
	}
	parseSupport, parseErr := ImportSupportDiagnosticBundleJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ImportSupportDiagnosticBundleJSON() error = %v", parseErr)
	}

	if !parseSupport.Sanitized || parseSupport.Version != currentSupportDiagnosticBundleVersion {
		parseT.Fatalf("expected sanitized support bundle metadata, got %+v", parseSupport)
	}
	if parseSupport.Trace.Snapshot.Route.Query["token"][0] != redactedSupportValue {
		parseT.Fatalf("expected sensitive route query value to be redacted, got %+v", parseSupport.Trace.Snapshot.Route.Query)
	}
	if parseSupport.Trace.Snapshot.Route.Query["view"][0] != "kanban" {
		parseT.Fatalf("expected non-sensitive route query value to remain, got %+v", parseSupport.Trace.Snapshot.Route.Query)
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Route.Loaders[0].Path, "session_token=%5Bredacted%5D") {
		parseT.Fatalf("expected loader path query to be redacted, got %+v", parseSupport.Trace.Snapshot.Route.Loaders)
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Cache[0].Key, "token=%5Bredacted%5D") {
		parseT.Fatalf("expected cache key query to be redacted, got %+v", parseSupport.Trace.Snapshot.Cache)
	}
	if parseSupport.Trace.Snapshot.Diagnostics[0].Fields["authorization"] != redactedSupportValue {
		parseT.Fatalf("expected diagnostic field to be redacted, got %+v", parseSupport.Trace.Snapshot.Diagnostics[0].Fields)
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Logs[0].Message, redactedSupportValue) {
		parseT.Fatalf("expected log message secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Logs)
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Tree.Hooks[0].Value, redactedSupportValue) {
		parseT.Fatalf("expected hook value to be redacted, got %+v", parseSupport.Trace.Snapshot.Tree.Hooks)
	}
	if parseSupport.Trace.Snapshot.Extensions[0].Summary["api_key"] != redactedSupportValue {
		parseT.Fatalf("expected extension summary to be redacted, got %+v", parseSupport.Trace.Snapshot.Extensions[0].Summary)
	}
}

func TestSupportDiagnosticBundleImportAndCaptureFallbacks(parseT *testing.T) {
	parseCaptured := CaptureSupportDiagnosticBundle("  support-capture  ")
	if !parseCaptured.Sanitized || parseCaptured.Version != currentSupportDiagnosticBundleVersion {
		parseT.Fatalf("expected sanitized captured support bundle metadata, got %+v", parseCaptured)
	}
	if parseCaptured.Label != "support-capture" {
		parseT.Fatalf("expected capture label to be trimmed, got %q", parseCaptured.Label)
	}

	parseEmpty, parseErr := ImportSupportDiagnosticBundleJSON(nil)
	if parseErr != nil {
		parseT.Fatalf("ImportSupportDiagnosticBundleJSON(nil) error = %v", parseErr)
	}
	if parseEmpty.Version != 0 || parseEmpty.Sanitized || parseEmpty.Label != "" || parseEmpty.CapturedAt != "" || parseEmpty.Trace.Label != "" || parseEmpty.Trace.CapturedAt != "" {
		parseT.Fatalf("expected empty import payload to return zero-value bundle, got %+v", parseEmpty)
	}

	parseImported, parseErr := ImportSupportDiagnosticBundleJSON([]byte(`{
		"trace": {
			"label": " trace-fallback ",
			"capturedAt": " 2026-03-25T13:30:00Z "
		}
	}`))
	if parseErr != nil {
		parseT.Fatalf("ImportSupportDiagnosticBundleJSON() error = %v", parseErr)
	}
	if parseImported.Version != currentSupportDiagnosticBundleVersion || !parseImported.Sanitized {
		parseT.Fatalf("expected import fallback version/sanitized metadata, got %+v", parseImported)
	}
	if parseImported.Label != "trace-fallback" || parseImported.CapturedAt != "2026-03-25T13:30:00Z" {
		parseT.Fatalf("expected import to fall back to trimmed trace metadata, got %+v", parseImported)
	}
}

func TestSupportSanitizeSnapshotDeepFields(parseT *testing.T) {
	parseSupport := SanitizeBugCaptureBundleForSupport(BugCaptureBundle{
		Label:      "deep-sanitize",
		CapturedAt: "2026-03-25T13:40:00Z",
		Trace: TraceCapture{
			Label:      "deep-sanitize",
			CapturedAt: "2026-03-25T13:40:00Z",
			Snapshot: Snapshot{
				MultiClient: MultiClient{
					AuthorityView: map[string]string{
						"authorization": "Bearer super-secret",
						"role":          "operator",
					},
					Peers: []MultiClientPeer{{
						ID:              "peer-token=abc123",
						App:             "atlas",
						Surface:         "window",
						Role:            "leader",
						State:           "connected",
						ProtocolVersion: "v1",
						Encodings:       []string{"json", "jwt=secret"},
						Topics:          []string{"chat", "token=abc123"},
					}},
					RecentTraffic: []MultiClientTraffic{{
						Kind:          "publish",
						Topic:         "session=secret",
						PeerID:        "peer-token=abc123",
						CorrelationID: "corr-token=abc123",
					}},
					FailedPublishes: []MultiClientFailure{{
						Topic:   "api_key=abc123",
						Target:  "ws://local?token=abc123",
						Code:    "authorization",
						Message: "cookie=session123",
					}},
				},
				Boundaries: BoundaryInspection{
					Entries: []Boundary{{
						Name:          "token=abc123",
						Kind:          "auth",
						Target:        "https://example.com?token=abc123",
						CorrelationID: "corr-token=abc123",
						Notes:         []string{"password=hunter2"},
						Redacted:      []string{"authorization=Bearer secret"},
						Downgraded:    []string{"session=secret"},
						Rejected:      []string{"api_key=abc123"},
					}},
				},
				Coordination: Coordination{
					Workers: []WorkerJob{{
						Name:        "worker?token=abc123",
						URL:         "https://example.com/worker?token=abc123",
						Kind:        "search",
						RequestID:   "req-token=abc123",
						Progress:    "password=hunter2",
						Result:      "session=secret",
						Error:       "authorization=Bearer secret",
						Correlation: "corr-token=abc123",
					}},
					SyncEvents: []SyncEvent{{
						Channel:     "session-channel",
						Topic:       "token=abc123",
						Target:      "cookie=session123",
						Error:       "api_key=abc123",
						Correlation: "corr-token=abc123",
					}},
					Replay: []ReplayEntry{{
						ID:        "replay-token=abc123",
						Kind:      "mutation",
						Method:    "POST",
						URL:       "https://example.com/mutate?token=abc123",
						Owner:     "tab?token=abc123",
						State:     "session=secret",
						LastError: "authorization=Bearer secret",
					}},
					QueueEntries: []SyncQueueEntry{{
						ID:        "queue-token=abc123",
						Entity:    "doc?token=abc123",
						Operation: "upsert",
						Owner:     "tab?token=abc123",
						State:     "retrying",
						URL:       "https://example.com/queue?token=abc123",
						LastError: "cookie=session123",
					}},
					SyncHealth: []SyncHealthEntry{{
						Entity:    "doc?token=abc123",
						Owner:     "tab?token=abc123",
						Status:    "degraded",
						Version:   "api_key=abc123",
						LastError: "authorization=Bearer secret",
					}},
					Reconnect: ReconnectStatus{
						State:     "token=abc123",
						Transport: "ws://local?token=abc123",
					},
					Conflict: ConflictState{
						Entity:    "doc?token=abc123",
						Owner:     "tab?token=abc123",
						Status:    "pending",
						Strategy:  "cookie=session123",
						LastError: "authorization=Bearer secret",
					},
					LastReplayError: "api_key=abc123",
				},
			},
		},
	})

	if parseSupport.Trace.Snapshot.MultiClient.AuthorityView["authorization"] != redactedSupportValue {
		parseT.Fatalf("expected multi-client authority secret to be redacted, got %+v", parseSupport.Trace.Snapshot.MultiClient.AuthorityView)
	}
	if parseSupport.Trace.Snapshot.MultiClient.AuthorityView["role"] != "operator" {
		parseT.Fatalf("expected non-sensitive authority field to remain, got %+v", parseSupport.Trace.Snapshot.MultiClient.AuthorityView)
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.MultiClient.Peers[0].ID, redactedSupportValue) {
		parseT.Fatalf("expected peer id secret to be redacted, got %+v", parseSupport.Trace.Snapshot.MultiClient.Peers[0])
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Boundaries.Entries[0].Name, redactedSupportValue) {
		parseT.Fatalf("expected boundary name secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Boundaries.Entries[0])
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Coordination.Workers[0].URL, "token=%5Bredacted%5D") {
		parseT.Fatalf("expected worker URL query secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Coordination.Workers[0])
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Coordination.Replay[0].URL, "token=%5Bredacted%5D") {
		parseT.Fatalf("expected replay URL query secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Coordination.Replay[0])
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Coordination.QueueEntries[0].URL, "token=%5Bredacted%5D") {
		parseT.Fatalf("expected queue URL query secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Coordination.QueueEntries[0])
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Coordination.SyncHealth[0].Version, redactedSupportValue) {
		parseT.Fatalf("expected sync health secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Coordination.SyncHealth[0])
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Coordination.Reconnect.Transport, "token=%5Bredacted%5D") {
		parseT.Fatalf("expected reconnect transport query secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Coordination.Reconnect)
	}
	if !strings.Contains(parseSupport.Trace.Snapshot.Coordination.LastReplayError, redactedSupportValue) {
		parseT.Fatalf("expected last replay error secret to be redacted, got %+v", parseSupport.Trace.Snapshot.Coordination)
	}
}

func TestBoundaryInspectionAndSupportHelperUtilities(parseT *testing.T) {
	parseNotes := appendBoundaryNotes([]string{"warn-a"}, []string{"err-a"})
	if len(parseNotes) != 2 || parseNotes[0] != "warning: warn-a" || parseNotes[1] != "error: err-a" {
		parseT.Fatalf("unexpected appended boundary notes: %+v", parseNotes)
	}

	if parseStatus := boundaryStatus(ui.SSRBootstrapSizeReport{Errors: []string{"too large"}}); parseStatus != "rejected" {
		parseT.Fatalf("expected rejected boundary status, got %q", parseStatus)
	}
	if parseStatus2 := boundaryStatus(ui.SSRBootstrapSizeReport{Warnings: []string{"near threshold"}}); parseStatus2 != "warning" {
		parseT.Fatalf("expected warning boundary status, got %q", parseStatus2)
	}
	if parseStatus3 := boundaryStatus(ui.SSRBootstrapSizeReport{}); parseStatus3 != "observed" {
		parseT.Fatalf("expected observed boundary status, got %q", parseStatus3)
	}

	if parseGot := approximateBoundarySize(nil); parseGot != 0 {
		parseT.Fatalf("expected nil approximate boundary size to be zero, got %d", parseGot)
	}
	if parseGot2 := approximateBoundarySize([]byte("abc")); parseGot2 != 3 {
		parseT.Fatalf("expected []byte approximate boundary size 3, got %d", parseGot2)
	}
	if parseGot3 := approximateBoundarySize("abcd"); parseGot3 != 4 {
		parseT.Fatalf("expected string approximate boundary size 4, got %d", parseGot3)
	}
	if parseGot4 := approximateBoundarySize(map[string]any{"bad": func() {}}); parseGot4 != 0 {
		parseT.Fatalf("expected marshal-error approximate boundary size to be zero, got %d", parseGot4)
	}

	if !isSupportSensitiveKey("session_id") || isSupportSensitiveKey("route") {
		parseT.Fatalf("expected sensitive-key detector to classify session_id=true and route=false")
	}
	if parseRedacted, parseOk := redactSupportURL("https://example.com?token=abc123&view=kanban"); !parseOk || !strings.Contains(parseRedacted, "token=%5Bredacted%5D") {
		parseT.Fatalf("expected support URL redaction for sensitive query key, got %q ok=%t", parseRedacted, parseOk)
	}
	if parseRedacted2, parseOk2 := redactSupportURL("https://example.com?view=kanban"); !parseOk2 || parseRedacted2 != "https://example.com?view=kanban" {
		parseT.Fatalf("expected support URL pass-through for non-sensitive query, got %q ok=%t", parseRedacted2, parseOk2)
	}
	if _, parseOk3 := redactSupportURL("plain text without query"); parseOk3 {
		parseT.Fatalf("expected non-url string to bypass URL redaction path")
	}
}

func TestTraceAndBugImportReplayFallbackBranches(parseT *testing.T) {
	parseT.Cleanup(ClearTraceReplay)

	if _, parseErr := ImportTraceCaptureJSON([]byte(`{`)); parseErr == nil {
		parseT.Fatal("expected ImportTraceCaptureJSON to fail on malformed JSON")
	}
	parseEmptyTrace, parseErr2 := ImportTraceCaptureJSON(nil)
	if parseErr2 != nil {
		parseT.Fatalf("ImportTraceCaptureJSON(nil) error = %v", parseErr2)
	}
	if parseEmptyTrace.Label != "" || parseEmptyTrace.CapturedAt != "" {
		parseT.Fatalf("expected nil trace payload to decode to zero metadata, got %+v", parseEmptyTrace)
	}

	parseTrace, parseErr2 := ImportTraceCaptureJSON([]byte(`{"label":"  route-trace  ","capturedAt":" 2026-03-25T15:00:00Z "}`))
	if parseErr2 != nil {
		parseT.Fatalf("ImportTraceCaptureJSON(trimmed) error = %v", parseErr2)
	}
	if parseTrace.Label != "route-trace" || parseTrace.CapturedAt != "2026-03-25T15:00:00Z" {
		parseT.Fatalf("expected trimmed trace metadata, got %+v", parseTrace)
	}

	if _, parseErr3 := ImportBugCaptureBundleJSON([]byte(`{`)); parseErr3 == nil {
		parseT.Fatal("expected ImportBugCaptureBundleJSON to fail on malformed JSON")
	}
	parseEmptyBundle, parseErr2 := ImportBugCaptureBundleJSON(nil)
	if parseErr2 != nil {
		parseT.Fatalf("ImportBugCaptureBundleJSON(nil) error = %v", parseErr2)
	}
	if parseEmptyBundle.Version != 0 || parseEmptyBundle.Label != "" || parseEmptyBundle.CapturedAt != "" {
		parseT.Fatalf("expected nil bug bundle payload to decode to zero metadata, got %+v", parseEmptyBundle)
	}

	parseImportedBundle, parseErr2 := ImportBugCaptureBundleJSON([]byte(`{
		"trace": {
			"label": " replay-fallback ",
			"capturedAt": " 2026-03-25T15:05:00Z "
		}
	}`))
	if parseErr2 != nil {
		parseT.Fatalf("ImportBugCaptureBundleJSON(fallback) error = %v", parseErr2)
	}
	if parseImportedBundle.Version != currentBugCaptureBundleVersion || parseImportedBundle.Label != "replay-fallback" || parseImportedBundle.CapturedAt != "2026-03-25T15:05:00Z" {
		parseT.Fatalf("expected version/metadata fallback from trace fields, got %+v", parseImportedBundle)
	}

	ReplayBugCaptureBundle(BugCaptureBundle{
		Label:      "bundle-replay",
		CapturedAt: "2026-03-25T15:10:00Z",
		Trace:      TraceCapture{},
	})
	parseReplayed, parseOk := CurrentTraceReplay()
	if !parseOk {
		parseT.Fatal("expected replay to be active after ReplayBugCaptureBundle")
	}
	if parseReplayed.Label != "bundle-replay" || parseReplayed.CapturedAt != "2026-03-25T15:10:00Z" {
		parseT.Fatalf("expected replay metadata fallback from bundle fields, got %+v", parseReplayed)
	}
}

func TestSupportProfilingAndSensitiveKeyBranches(parseT *testing.T) {
	for _, parseTc := range []struct {
		key  string
		want bool
	}{
		{key: "", want: false},
		{key: "password", want: true},
		{key: "passwd_hash", want: true},
		{key: "secretValue", want: true},
		{key: "auth_header", want: true},
		{key: "cookie", want: true},
		{key: "session_id", want: true},
		{key: "api-key", want: true},
		{key: "jwt_token", want: true},
		{key: "credentialID", want: true},
		{key: "route", want: false},
	} {
		if parseGot := isSupportSensitiveKey(parseTc.key); parseGot != parseTc.want {
			parseT.Fatalf("isSupportSensitiveKey(%q) = %t, want %t", parseTc.key, parseGot, parseTc.want)
		}
	}

	parseProfiling := sanitizeProfilingForSupport(Profiling{
		ComponentRenders: []ComponentRenderTrace{{
			LastTrigger: "token=abc123",
		}},
		RecentEvents: []ProfilingEvent{{
			Target:        "https://example.com/workspace?token=abc123",
			CorrelationID: "jwt=abc123",
			Fields: map[string]string{
				"api_key": "abc123",
				"view":    "kanban",
			},
		}},
		Startup: StartupProfiling{
			FirstInteractionEvent: "password=hunter2",
		},
	})
	if !strings.Contains(parseProfiling.ComponentRenders[0].LastTrigger, redactedSupportValue) {
		parseT.Fatalf("expected profiling last trigger to be redacted, got %+v", parseProfiling.ComponentRenders)
	}
	if !strings.Contains(parseProfiling.RecentEvents[0].Target, "token=%5Bredacted%5D") {
		parseT.Fatalf("expected profiling target URL query to be redacted, got %+v", parseProfiling.RecentEvents)
	}
	if !strings.Contains(parseProfiling.RecentEvents[0].CorrelationID, redactedSupportValue) {
		parseT.Fatalf("expected profiling correlation id to be redacted, got %+v", parseProfiling.RecentEvents)
	}
	if parseProfiling.RecentEvents[0].Fields["api_key"] != redactedSupportValue || parseProfiling.RecentEvents[0].Fields["view"] != "kanban" {
		parseT.Fatalf("expected sensitive field redacted and non-sensitive field preserved, got %+v", parseProfiling.RecentEvents[0].Fields)
	}
	if !strings.Contains(parseProfiling.Startup.FirstInteractionEvent, redactedSupportValue) {
		parseT.Fatalf("expected startup event to be redacted, got %+v", parseProfiling.Startup)
	}

	parseSupport := SanitizeBugCaptureBundleForSupport(BugCaptureBundle{
		Trace: TraceCapture{
			Label:      " sanitized-trace ",
			CapturedAt: " 2026-03-25T15:20:00Z ",
		},
	})
	if parseSupport.Label != "sanitized-trace" || parseSupport.CapturedAt != "2026-03-25T15:20:00Z" {
		parseT.Fatalf("expected support bundle to fall back to trace metadata, got %+v", parseSupport)
	}
}
