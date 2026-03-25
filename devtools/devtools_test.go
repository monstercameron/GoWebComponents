//go:build !js || !wasm
// +build !js !wasm

package devtools

import (
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestSnapshotStubsReturnEmptyValuesOnNative(t *testing.T) {
	if got := SnapshotNow(); got.Tree != nil || len(got.Cache) != 0 || got.Route.Path != "" {
		t.Fatalf("SnapshotNow() = %+v, want empty snapshot", got)
	}
	if got := UseSnapshot(250 * time.Millisecond); got.Tree != nil || len(got.Cache) != 0 || got.Route.Path != "" {
		t.Fatalf("UseSnapshot() = %+v, want empty snapshot", got)
	}
	if panel := Panel(PanelProps{Title: "Inspector", RefreshInterval: time.Second}); panel != nil {
		t.Fatalf("Panel() = %#v, want nil", panel)
	}
	if overlay := ErrorOverlay(ErrorOverlayProps{Title: "Errors", RefreshInterval: time.Second}); overlay != nil {
		t.Fatalf("ErrorOverlay() = %#v, want nil", overlay)
	}
}

func TestMultiClientInspectionRoundTripsClonedState(t *testing.T) {
	t.Cleanup(ResetMultiClientInspection)
	original := MultiClient{
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

	SetMultiClientInspection(original)
	cloned := InspectMultiClient()
	if cloned.LocalPeerID != "peer-a" || !cloned.Enabled {
		t.Fatalf("unexpected cloned state: %+v", cloned)
	}

	original.AuthorityView["role"] = "follower"
	original.Peers[0].Topics[0] = "mutated"
	original.RecentTraffic[0].Topic = "changed"
	original.FailedPublishes[0].Code = "changed"

	afterMutation := InspectMultiClient()
	if afterMutation.AuthorityView["role"] != "leader" {
		t.Fatalf("authority view should be cloned, got %+v", afterMutation.AuthorityView)
	}
	if afterMutation.Peers[0].Topics[0] != "chat" {
		t.Fatalf("peer topics should be cloned, got %+v", afterMutation.Peers)
	}
	if afterMutation.RecentTraffic[0].Topic != "chat" {
		t.Fatalf("recent traffic should be cloned, got %+v", afterMutation.RecentTraffic)
	}
	if afterMutation.FailedPublishes[0].Code != "timeout" {
		t.Fatalf("failed publishes should be cloned, got %+v", afterMutation.FailedPublishes)
	}

	ResetMultiClientInspection()
	if got := InspectMultiClient(); got.Enabled || got.LocalPeerID != "" || len(got.Peers) != 0 || len(got.AuthorityView) != 0 {
		t.Fatalf("ResetMultiClientInspection() left residual state: %+v", got)
	}
}

func TestCompareSnapshotsDetectsChangedSections(t *testing.T) {
	previous := Snapshot{
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
	current := Snapshot{
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

	comparison, err := CompareSnapshots(previous, current)
	if err != nil {
		t.Fatalf("CompareSnapshots(): %v", err)
	}
	if comparison.Equal {
		t.Fatalf("expected snapshots to differ: %+v", comparison)
	}
	if comparison.PreviousFingerprint == "" || comparison.CurrentFingerprint == "" || comparison.PreviousFingerprint == comparison.CurrentFingerprint {
		t.Fatalf("expected distinct fingerprints, got %+v", comparison)
	}
	changed := strings.Join(comparison.ChangedSections, ",")
	for _, section := range []string{"route", "stats", "boundaries", "coordination", "extensions", "hydration", "logs"} {
		if !strings.Contains(changed, section) {
			t.Fatalf("expected changed sections to contain %q, got %v", section, comparison.ChangedSections)
		}
	}

	equalComparison, err := CompareSnapshots(current, current)
	if err != nil {
		t.Fatalf("CompareSnapshots(equal): %v", err)
	}
	if !equalComparison.Equal || len(equalComparison.ChangedSections) != 0 {
		t.Fatalf("expected equal comparison without changed sections, got %+v", equalComparison)
	}
}

func TestSerializationBoundaryInspectionRoundTripsClonedState(t *testing.T) {
	t.Cleanup(ResetSerializationBoundaryInspection)

	original := BoundaryInspection{Entries: []Boundary{{
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

	SetSerializationBoundaryInspection(original)
	cloned := InspectSerializationBoundaries()
	if len(cloned.Entries) != 1 || cloned.Entries[0].Name != "worker.search" {
		t.Fatalf("unexpected cloned boundary state: %+v", cloned)
	}

	original.Entries[0].Name = "mutated"
	original.Entries[0].Notes[0] = "changed"
	original.Entries[0].Rejected[0] = "changed"

	afterMutation := InspectSerializationBoundaries()
	if afterMutation.Entries[0].Name != "worker.search" || afterMutation.Entries[0].Notes[0] != "kind=query" || afterMutation.Entries[0].Rejected[0] != "payload too large" {
		t.Fatalf("expected boundary state to be cloned, got %+v", afterMutation)
	}

	ResetSerializationBoundaryInspection()
	if got := InspectSerializationBoundaries(); len(got.Entries) != 0 {
		t.Fatalf("ResetSerializationBoundaryInspection() left residual state: %+v", got)
	}
}

func TestInspectBootstrapBoundariesSummarizesPayloads(t *testing.T) {
	bootstrap := ui.SSRBootstrap{Data: map[string]interface{}{
		"legacy-message": "hello",
	}}
	if err := ui.RegisterRouteBootstrapData(&bootstrap, "catalog", "/products", map[string]string{"sku": "atlas-1"}); err != nil {
		t.Fatalf("RegisterRouteBootstrapData() error = %v", err)
	}
	if err := ui.RegisterSessionBootstrapHint(&bootstrap, "viewer", map[string]string{"role": "operator"}); err != nil {
		t.Fatalf("RegisterSessionBootstrapHint() error = %v", err)
	}

	inspection, err := InspectBootstrapBoundaries(bootstrap)
	if err != nil {
		t.Fatalf("InspectBootstrapBoundaries() error = %v", err)
	}
	if len(inspection.Entries) != 4 {
		t.Fatalf("expected root plus three payload entries, got %+v", inspection.Entries)
	}
	root := inspection.Entries[0]
	if root.Name != "ssr.bootstrap" || root.Kind != "ssr-bootstrap" || root.SizeBytes <= 0 || root.InlineBytes <= 0 || root.BinaryBytes <= 0 {
		t.Fatalf("unexpected bootstrap root boundary: %+v", root)
	}

	foundLegacy := false
	foundRoute := false
	for _, entry := range inspection.Entries[1:] {
		switch entry.Name {
		case "legacy-message":
			foundLegacy = true
			if entry.Status != "downgraded" || len(entry.Downgraded) != 1 {
				t.Fatalf("expected legacy payload to be marked downgraded, got %+v", entry)
			}
		case "/products::catalog":
			foundRoute = true
			if entry.Scope != "route" || entry.Encoding != "json" || entry.SizeBytes <= 0 {
				t.Fatalf("expected route payload metadata to map, got %+v", entry)
			}
		}
	}
	if !foundLegacy || !foundRoute {
		t.Fatalf("expected both legacy and route payload entries, got %+v", inspection.Entries)
	}
}

func TestCoordinationInspectionRoundTripsClonedState(t *testing.T) {
	t.Cleanup(ResetCoordinationInspection)

	original := Coordination{
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

	SetCoordinationInspection(original)
	cloned := InspectCoordination()
	if len(cloned.Workers) != 1 || len(cloned.SyncEvents) != 1 || len(cloned.Replay) != 1 || len(cloned.QueueEntries) != 1 || len(cloned.SyncHealth) != 1 {
		t.Fatalf("unexpected coordination clone: %+v", cloned)
	}
	if cloned.Reconnect.State != "reconnecting" || cloned.Conflict.Entity != "order:42" || cloned.LastReplayError != "HTTP 409 conflict" {
		t.Fatalf("expected extended coordination fields to clone, got %+v", cloned)
	}

	original.Workers[0].Name = "mutated"
	original.SyncEvents[0].Topic = "changed"
	original.Replay[0].State = "dead"
	original.QueueEntries[0].Entity = "order:dead"
	original.SyncHealth[0].Status = "stale"
	original.Reconnect.State = "offline"
	original.Conflict.Status = "resolved"
	original.LastReplayError = "mutated"

	afterMutation := InspectCoordination()
	if afterMutation.Workers[0].Name != "search-index" || afterMutation.SyncEvents[0].Topic != "theme" || afterMutation.Replay[0].State != "retrying" || afterMutation.QueueEntries[0].Entity != "order:42" || afterMutation.SyncHealth[0].Status != "healthy" || afterMutation.Reconnect.State != "reconnecting" || afterMutation.Conflict.Status != "pending" || afterMutation.LastReplayError != "HTTP 409 conflict" {
		t.Fatalf("expected coordination state to be cloned, got %+v", afterMutation)
	}

	ResetCoordinationInspection()
	if got := InspectCoordination(); len(got.Workers) != 0 || len(got.SyncEvents) != 0 || len(got.Replay) != 0 || len(got.QueueEntries) != 0 || len(got.SyncHealth) != 0 || got.Reconnect != (ReconnectStatus{}) || got.Conflict != (ConflictState{}) || got.LastReplayError != "" {
		t.Fatalf("ResetCoordinationInspection() left residual state: %+v", got)
	}
}

func TestErrorOverlayActionsRoundTripClonedState(t *testing.T) {
	t.Cleanup(ResetErrorOverlayActions)

	var invoked bool
	original := []ErrorOverlayAction{{
		Label:        "Retry loader",
		MatchCodes:   []string{"GWC-ROUTER-LOADER-FAILED"},
		MatchSources: []string{"router"},
		Run: func(ErrorOverlayActionContext) {
			invoked = true
		},
	}}

	SetErrorOverlayActions(original)
	cloned := InspectErrorOverlayActions()
	if len(cloned) != 1 || cloned[0].Label != "Retry loader" || len(cloned[0].MatchCodes) != 1 {
		t.Fatalf("unexpected overlay action clone: %+v", cloned)
	}

	original[0].Label = "Mutated"
	original[0].MatchCodes[0] = "changed"

	afterMutation := InspectErrorOverlayActions()
	if afterMutation[0].Label != "Retry loader" || afterMutation[0].MatchCodes[0] != "GWC-ROUTER-LOADER-FAILED" {
		t.Fatalf("expected overlay actions to be cloned, got %+v", afterMutation)
	}

	afterMutation[0].Run(ErrorOverlayActionContext{})
	if !invoked {
		t.Fatal("expected cloned overlay action handler to remain callable")
	}

	ResetErrorOverlayActions()
	if got := InspectErrorOverlayActions(); len(got) != 0 {
		t.Fatalf("ResetErrorOverlayActions() left residual state: %+v", got)
	}
}

func TestTraceCaptureExportImportAndReplay(t *testing.T) {
	t.Cleanup(ClearTraceReplay)

	capture := TraceCapture{
		Label:      "checkout-click",
		CapturedAt: "2026-03-25T12:00:00Z",
		Snapshot: Snapshot{
			Route: Route{Path: "/checkout"},
			Logs:  []Log{{Domain: "router", Message: "navigated"}},
		},
	}

	payload, err := ExportTraceCaptureJSON(capture)
	if err != nil {
		t.Fatalf("ExportTraceCaptureJSON() error = %v", err)
	}
	decoded, err := ImportTraceCaptureJSON(payload)
	if err != nil {
		t.Fatalf("ImportTraceCaptureJSON() error = %v", err)
	}
	if decoded.Label != "checkout-click" || decoded.Snapshot.Route.Path != "/checkout" || len(decoded.Snapshot.Logs) != 1 {
		t.Fatalf("unexpected decoded trace capture: %+v", decoded)
	}

	SetTraceReplay(decoded)
	replayed, ok := CurrentTraceReplay()
	if !ok || replayed.Label != "checkout-click" || replayed.Snapshot.Route.Path != "/checkout" {
		t.Fatalf("expected trace replay to round-trip, got replay=%+v ok=%t", replayed, ok)
	}
	if snapshot := SnapshotNow(); snapshot.Route.Path != "/checkout" || len(snapshot.Logs) != 1 {
		t.Fatalf("expected SnapshotNow() to return replayed trace, got %+v", snapshot)
	}

	ClearTraceReplay()
	if _, ok := CurrentTraceReplay(); ok {
		t.Fatal("expected replay to be cleared")
	}
}

func TestBugCaptureBundleExportImportAndReplay(t *testing.T) {
	t.Cleanup(ClearTraceReplay)

	bundle := BugCaptureBundle{
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

	payload, err := ExportBugCaptureBundleJSON(bundle)
	if err != nil {
		t.Fatalf("ExportBugCaptureBundleJSON() error = %v", err)
	}
	decoded, err := ImportBugCaptureBundleJSON(payload)
	if err != nil {
		t.Fatalf("ImportBugCaptureBundleJSON() error = %v", err)
	}
	if decoded.Label != "startup-failure" || decoded.Trace.Snapshot.Route.Path != "/" || len(decoded.Trace.Snapshot.Diagnostics) != 1 {
		t.Fatalf("unexpected decoded bug bundle: %+v", decoded)
	}

	ReplayBugCaptureBundle(decoded)
	snapshot := SnapshotNow()
	if snapshot.Route.Path != "/" || len(snapshot.Diagnostics) != 1 || snapshot.Diagnostics[0].Code != "GWC-RUNTIME-PANIC-STARTUP" {
		t.Fatalf("expected replayed bug bundle snapshot, got %+v", snapshot)
	}
}

func TestCaptureTraceAndBugBundlePopulateMetadata(t *testing.T) {
	t.Cleanup(ClearTraceReplay)

	trace := CaptureTrace("  refresh-route  ")
	if trace.Label != "refresh-route" {
		t.Fatalf("expected trace label to be trimmed, got %q", trace.Label)
	}
	if trace.CapturedAt == "" {
		t.Fatalf("expected trace capture timestamp")
	}
	if _, err := time.Parse(time.RFC3339Nano, trace.CapturedAt); err != nil {
		t.Fatalf("expected RFC3339 trace timestamp, got %q: %v", trace.CapturedAt, err)
	}

	bundle := CaptureBugBundle("  checkout-bug  ")
	if bundle.Version != currentBugCaptureBundleVersion {
		t.Fatalf("expected bundle version %d, got %d", currentBugCaptureBundleVersion, bundle.Version)
	}
	if bundle.Label != "checkout-bug" || bundle.Trace.Label != "checkout-bug" {
		t.Fatalf("expected trimmed bundle/trace labels, got bundle=%q trace=%q", bundle.Label, bundle.Trace.Label)
	}
	if bundle.CapturedAt == "" || bundle.Trace.CapturedAt == "" {
		t.Fatalf("expected bundle and trace timestamps, got bundle=%q trace=%q", bundle.CapturedAt, bundle.Trace.CapturedAt)
	}
	if _, err := time.Parse(time.RFC3339Nano, bundle.CapturedAt); err != nil {
		t.Fatalf("expected RFC3339 bundle timestamp, got %q: %v", bundle.CapturedAt, err)
	}
}

func TestExtensionSectionsRoundTripClonedState(t *testing.T) {
	t.Cleanup(ResetExtensionSections)

	original := []ExtensionSection{{
		Name:    "Companion",
		Summary: map[string]string{"state": "ready"},
		Lines:   []string{"line one"},
	}}
	SetExtensionSections(original)
	cloned := InspectExtensionSections()
	if len(cloned) != 1 || cloned[0].Name != "Companion" || cloned[0].Summary["state"] != "ready" {
		t.Fatalf("unexpected extension-section clone: %+v", cloned)
	}

	original[0].Name = "Mutated"
	original[0].Summary["state"] = "changed"
	original[0].Lines[0] = "changed"

	afterMutation := InspectExtensionSections()
	if afterMutation[0].Name != "Companion" || afterMutation[0].Summary["state"] != "ready" || afterMutation[0].Lines[0] != "line one" {
		t.Fatalf("expected extension sections to be cloned, got %+v", afterMutation)
	}

	ResetExtensionSections()
	if got := InspectExtensionSections(); len(got) != 0 {
		t.Fatalf("ResetExtensionSections() left residual state: %+v", got)
	}
}

func TestSupportDiagnosticBundleExportRedactsSensitiveValues(t *testing.T) {
	bundle := BugCaptureBundle{
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

	payload, err := ExportSupportDiagnosticBundleJSON(bundle)
	if err != nil {
		t.Fatalf("ExportSupportDiagnosticBundleJSON() error = %v", err)
	}
	support, err := ImportSupportDiagnosticBundleJSON(payload)
	if err != nil {
		t.Fatalf("ImportSupportDiagnosticBundleJSON() error = %v", err)
	}

	if !support.Sanitized || support.Version != currentSupportDiagnosticBundleVersion {
		t.Fatalf("expected sanitized support bundle metadata, got %+v", support)
	}
	if support.Trace.Snapshot.Route.Query["token"][0] != redactedSupportValue {
		t.Fatalf("expected sensitive route query value to be redacted, got %+v", support.Trace.Snapshot.Route.Query)
	}
	if support.Trace.Snapshot.Route.Query["view"][0] != "kanban" {
		t.Fatalf("expected non-sensitive route query value to remain, got %+v", support.Trace.Snapshot.Route.Query)
	}
	if !strings.Contains(support.Trace.Snapshot.Route.Loaders[0].Path, "session_token=%5Bredacted%5D") {
		t.Fatalf("expected loader path query to be redacted, got %+v", support.Trace.Snapshot.Route.Loaders)
	}
	if !strings.Contains(support.Trace.Snapshot.Cache[0].Key, "token=%5Bredacted%5D") {
		t.Fatalf("expected cache key query to be redacted, got %+v", support.Trace.Snapshot.Cache)
	}
	if support.Trace.Snapshot.Diagnostics[0].Fields["authorization"] != redactedSupportValue {
		t.Fatalf("expected diagnostic field to be redacted, got %+v", support.Trace.Snapshot.Diagnostics[0].Fields)
	}
	if !strings.Contains(support.Trace.Snapshot.Logs[0].Message, redactedSupportValue) {
		t.Fatalf("expected log message secret to be redacted, got %+v", support.Trace.Snapshot.Logs)
	}
	if !strings.Contains(support.Trace.Snapshot.Tree.Hooks[0].Value, redactedSupportValue) {
		t.Fatalf("expected hook value to be redacted, got %+v", support.Trace.Snapshot.Tree.Hooks)
	}
	if support.Trace.Snapshot.Extensions[0].Summary["api_key"] != redactedSupportValue {
		t.Fatalf("expected extension summary to be redacted, got %+v", support.Trace.Snapshot.Extensions[0].Summary)
	}
}

func TestSupportDiagnosticBundleImportAndCaptureFallbacks(t *testing.T) {
	captured := CaptureSupportDiagnosticBundle("  support-capture  ")
	if !captured.Sanitized || captured.Version != currentSupportDiagnosticBundleVersion {
		t.Fatalf("expected sanitized captured support bundle metadata, got %+v", captured)
	}
	if captured.Label != "support-capture" {
		t.Fatalf("expected capture label to be trimmed, got %q", captured.Label)
	}

	empty, err := ImportSupportDiagnosticBundleJSON(nil)
	if err != nil {
		t.Fatalf("ImportSupportDiagnosticBundleJSON(nil) error = %v", err)
	}
	if empty.Version != 0 || empty.Sanitized || empty.Label != "" || empty.CapturedAt != "" || empty.Trace.Label != "" || empty.Trace.CapturedAt != "" {
		t.Fatalf("expected empty import payload to return zero-value bundle, got %+v", empty)
	}

	imported, err := ImportSupportDiagnosticBundleJSON([]byte(`{
		"trace": {
			"label": " trace-fallback ",
			"capturedAt": " 2026-03-25T13:30:00Z "
		}
	}`))
	if err != nil {
		t.Fatalf("ImportSupportDiagnosticBundleJSON() error = %v", err)
	}
	if imported.Version != currentSupportDiagnosticBundleVersion || !imported.Sanitized {
		t.Fatalf("expected import fallback version/sanitized metadata, got %+v", imported)
	}
	if imported.Label != "trace-fallback" || imported.CapturedAt != "2026-03-25T13:30:00Z" {
		t.Fatalf("expected import to fall back to trimmed trace metadata, got %+v", imported)
	}
}

func TestSupportSanitizeSnapshotDeepFields(t *testing.T) {
	support := SanitizeBugCaptureBundleForSupport(BugCaptureBundle{
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

	if support.Trace.Snapshot.MultiClient.AuthorityView["authorization"] != redactedSupportValue {
		t.Fatalf("expected multi-client authority secret to be redacted, got %+v", support.Trace.Snapshot.MultiClient.AuthorityView)
	}
	if support.Trace.Snapshot.MultiClient.AuthorityView["role"] != "operator" {
		t.Fatalf("expected non-sensitive authority field to remain, got %+v", support.Trace.Snapshot.MultiClient.AuthorityView)
	}
	if !strings.Contains(support.Trace.Snapshot.MultiClient.Peers[0].ID, redactedSupportValue) {
		t.Fatalf("expected peer id secret to be redacted, got %+v", support.Trace.Snapshot.MultiClient.Peers[0])
	}
	if !strings.Contains(support.Trace.Snapshot.Boundaries.Entries[0].Name, redactedSupportValue) {
		t.Fatalf("expected boundary name secret to be redacted, got %+v", support.Trace.Snapshot.Boundaries.Entries[0])
	}
	if !strings.Contains(support.Trace.Snapshot.Coordination.Workers[0].URL, "token=%5Bredacted%5D") {
		t.Fatalf("expected worker URL query secret to be redacted, got %+v", support.Trace.Snapshot.Coordination.Workers[0])
	}
	if !strings.Contains(support.Trace.Snapshot.Coordination.Replay[0].URL, "token=%5Bredacted%5D") {
		t.Fatalf("expected replay URL query secret to be redacted, got %+v", support.Trace.Snapshot.Coordination.Replay[0])
	}
	if !strings.Contains(support.Trace.Snapshot.Coordination.QueueEntries[0].URL, "token=%5Bredacted%5D") {
		t.Fatalf("expected queue URL query secret to be redacted, got %+v", support.Trace.Snapshot.Coordination.QueueEntries[0])
	}
	if !strings.Contains(support.Trace.Snapshot.Coordination.SyncHealth[0].Version, redactedSupportValue) {
		t.Fatalf("expected sync health secret to be redacted, got %+v", support.Trace.Snapshot.Coordination.SyncHealth[0])
	}
	if !strings.Contains(support.Trace.Snapshot.Coordination.Reconnect.Transport, "token=%5Bredacted%5D") {
		t.Fatalf("expected reconnect transport query secret to be redacted, got %+v", support.Trace.Snapshot.Coordination.Reconnect)
	}
	if !strings.Contains(support.Trace.Snapshot.Coordination.LastReplayError, redactedSupportValue) {
		t.Fatalf("expected last replay error secret to be redacted, got %+v", support.Trace.Snapshot.Coordination)
	}
}

func TestBoundaryInspectionAndSupportHelperUtilities(t *testing.T) {
	notes := appendBoundaryNotes([]string{"warn-a"}, []string{"err-a"})
	if len(notes) != 2 || notes[0] != "warning: warn-a" || notes[1] != "error: err-a" {
		t.Fatalf("unexpected appended boundary notes: %+v", notes)
	}

	if status := boundaryStatus(ui.SSRBootstrapSizeReport{Errors: []string{"too large"}}); status != "rejected" {
		t.Fatalf("expected rejected boundary status, got %q", status)
	}
	if status := boundaryStatus(ui.SSRBootstrapSizeReport{Warnings: []string{"near threshold"}}); status != "warning" {
		t.Fatalf("expected warning boundary status, got %q", status)
	}
	if status := boundaryStatus(ui.SSRBootstrapSizeReport{}); status != "observed" {
		t.Fatalf("expected observed boundary status, got %q", status)
	}

	if got := approximateBoundarySize(nil); got != 0 {
		t.Fatalf("expected nil approximate boundary size to be zero, got %d", got)
	}
	if got := approximateBoundarySize([]byte("abc")); got != 3 {
		t.Fatalf("expected []byte approximate boundary size 3, got %d", got)
	}
	if got := approximateBoundarySize("abcd"); got != 4 {
		t.Fatalf("expected string approximate boundary size 4, got %d", got)
	}
	if got := approximateBoundarySize(map[string]any{"bad": func() {}}); got != 0 {
		t.Fatalf("expected marshal-error approximate boundary size to be zero, got %d", got)
	}

	if !isSupportSensitiveKey("session_id") || isSupportSensitiveKey("route") {
		t.Fatalf("expected sensitive-key detector to classify session_id=true and route=false")
	}
	if redacted, ok := redactSupportURL("https://example.com?token=abc123&view=kanban"); !ok || !strings.Contains(redacted, "token=%5Bredacted%5D") {
		t.Fatalf("expected support URL redaction for sensitive query key, got %q ok=%t", redacted, ok)
	}
	if redacted, ok := redactSupportURL("https://example.com?view=kanban"); !ok || redacted != "https://example.com?view=kanban" {
		t.Fatalf("expected support URL pass-through for non-sensitive query, got %q ok=%t", redacted, ok)
	}
	if _, ok := redactSupportURL("plain text without query"); ok {
		t.Fatalf("expected non-url string to bypass URL redaction path")
	}
}

func TestTraceAndBugImportReplayFallbackBranches(t *testing.T) {
	t.Cleanup(ClearTraceReplay)

	if _, err := ImportTraceCaptureJSON([]byte(`{`)); err == nil {
		t.Fatal("expected ImportTraceCaptureJSON to fail on malformed JSON")
	}
	emptyTrace, err := ImportTraceCaptureJSON(nil)
	if err != nil {
		t.Fatalf("ImportTraceCaptureJSON(nil) error = %v", err)
	}
	if emptyTrace.Label != "" || emptyTrace.CapturedAt != "" {
		t.Fatalf("expected nil trace payload to decode to zero metadata, got %+v", emptyTrace)
	}

	trace, err := ImportTraceCaptureJSON([]byte(`{"label":"  route-trace  ","capturedAt":" 2026-03-25T15:00:00Z "}`))
	if err != nil {
		t.Fatalf("ImportTraceCaptureJSON(trimmed) error = %v", err)
	}
	if trace.Label != "route-trace" || trace.CapturedAt != "2026-03-25T15:00:00Z" {
		t.Fatalf("expected trimmed trace metadata, got %+v", trace)
	}

	if _, err := ImportBugCaptureBundleJSON([]byte(`{`)); err == nil {
		t.Fatal("expected ImportBugCaptureBundleJSON to fail on malformed JSON")
	}
	emptyBundle, err := ImportBugCaptureBundleJSON(nil)
	if err != nil {
		t.Fatalf("ImportBugCaptureBundleJSON(nil) error = %v", err)
	}
	if emptyBundle.Version != 0 || emptyBundle.Label != "" || emptyBundle.CapturedAt != "" {
		t.Fatalf("expected nil bug bundle payload to decode to zero metadata, got %+v", emptyBundle)
	}

	importedBundle, err := ImportBugCaptureBundleJSON([]byte(`{
		"trace": {
			"label": " replay-fallback ",
			"capturedAt": " 2026-03-25T15:05:00Z "
		}
	}`))
	if err != nil {
		t.Fatalf("ImportBugCaptureBundleJSON(fallback) error = %v", err)
	}
	if importedBundle.Version != currentBugCaptureBundleVersion || importedBundle.Label != "replay-fallback" || importedBundle.CapturedAt != "2026-03-25T15:05:00Z" {
		t.Fatalf("expected version/metadata fallback from trace fields, got %+v", importedBundle)
	}

	ReplayBugCaptureBundle(BugCaptureBundle{
		Label:      "bundle-replay",
		CapturedAt: "2026-03-25T15:10:00Z",
		Trace:      TraceCapture{},
	})
	replayed, ok := CurrentTraceReplay()
	if !ok {
		t.Fatal("expected replay to be active after ReplayBugCaptureBundle")
	}
	if replayed.Label != "bundle-replay" || replayed.CapturedAt != "2026-03-25T15:10:00Z" {
		t.Fatalf("expected replay metadata fallback from bundle fields, got %+v", replayed)
	}
}

func TestSupportProfilingAndSensitiveKeyBranches(t *testing.T) {
	for _, tc := range []struct {
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
		if got := isSupportSensitiveKey(tc.key); got != tc.want {
			t.Fatalf("isSupportSensitiveKey(%q) = %t, want %t", tc.key, got, tc.want)
		}
	}

	profiling := sanitizeProfilingForSupport(Profiling{
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
	if !strings.Contains(profiling.ComponentRenders[0].LastTrigger, redactedSupportValue) {
		t.Fatalf("expected profiling last trigger to be redacted, got %+v", profiling.ComponentRenders)
	}
	if !strings.Contains(profiling.RecentEvents[0].Target, "token=%5Bredacted%5D") {
		t.Fatalf("expected profiling target URL query to be redacted, got %+v", profiling.RecentEvents)
	}
	if !strings.Contains(profiling.RecentEvents[0].CorrelationID, redactedSupportValue) {
		t.Fatalf("expected profiling correlation id to be redacted, got %+v", profiling.RecentEvents)
	}
	if profiling.RecentEvents[0].Fields["api_key"] != redactedSupportValue || profiling.RecentEvents[0].Fields["view"] != "kanban" {
		t.Fatalf("expected sensitive field redacted and non-sensitive field preserved, got %+v", profiling.RecentEvents[0].Fields)
	}
	if !strings.Contains(profiling.Startup.FirstInteractionEvent, redactedSupportValue) {
		t.Fatalf("expected startup event to be redacted, got %+v", profiling.Startup)
	}

	support := SanitizeBugCaptureBundleForSupport(BugCaptureBundle{
		Trace: TraceCapture{
			Label:      " sanitized-trace ",
			CapturedAt: " 2026-03-25T15:20:00Z ",
		},
	})
	if support.Label != "sanitized-trace" || support.CapturedAt != "2026-03-25T15:20:00Z" {
		t.Fatalf("expected support bundle to fall back to trace metadata, got %+v", support)
	}
}
