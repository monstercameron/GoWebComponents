package runtime2_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestGetHostRegionDiagnosticsSnapshotReportsReadOnlyRedactedState verifies diagnostics snapshots return redacted events, downgrade accounting, and counters without exposing mutable state.
func TestGetHostRegionDiagnosticsSnapshotReportsReadOnlyRedactedState(parseT *testing.T) {
	buildHostRegionAdapter := buildHostRegionAdapterForTransportDispatchTests(parseT)
	_, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		buildSnapshotTransportCapabilityReport(),
		nil,
	)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport returned error: %v", parseDispatchErr)
	}
	parseDiagnosticEnvelope, parseDiagnosticErr := runtime2.BuildControlDiagnosticEnvelope("region-1", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindPatchReady,
		TransportTier:  runtime2.TransportTierStructuredClone,
		DiagnosticText: "token=secret-123",
		DiagnosticDowngrade: &runtime2.DiagnosticDowngradeReason{
			Path:   runtime2.DiagnosticDowngradePathSharedMemory,
			Reason: string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage),
		},
	})
	if parseDiagnosticErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseDiagnosticErr)
	}
	if _, parseDispatchDiagnosticErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseDiagnosticEnvelope); parseDispatchDiagnosticErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(diagnostic) returned error: %v", parseDispatchDiagnosticErr)
	}
	if _, parseCountErr := buildHostRegionAdapter.GetHostRegionCoordinator().IncrementRegionDroppedStalePatchCount(runtime2.RegionInstanceID("region-1")); parseCountErr != nil {
		parseT.Fatalf("IncrementRegionDroppedStalePatchCount returned error: %v", parseCountErr)
	}
	if _, parseCountErr := buildHostRegionAdapter.GetHostRegionCoordinator().IncrementRegionIgnoredStaleDiagnosticCount(runtime2.RegionInstanceID("region-1")); parseCountErr != nil {
		parseT.Fatalf("IncrementRegionIgnoredStaleDiagnosticCount returned error: %v", parseCountErr)
	}
	if _, parseCountErr := buildHostRegionAdapter.GetHostRegionCoordinator().IncrementRegionRepairRemountCount(runtime2.RegionInstanceID("region-1")); parseCountErr != nil {
		parseT.Fatalf("IncrementRegionRepairRemountCount returned error: %v", parseCountErr)
	}
	getSnapshot, hasSnapshot := buildHostRegionAdapter.GetHostRegionDiagnosticsSnapshot()
	if !hasSnapshot {
		parseT.Fatal("expected diagnostics snapshot for mounted region")
	}
	if len(getSnapshot.GetDiagnosticEvents) == 0 {
		parseT.Fatal("expected diagnostics snapshot to include recent events")
	}
	if strings.Contains(getSnapshot.GetDiagnosticEvents[0].DiagnosticText, "secret-123") {
		parseT.Fatalf("expected diagnostics snapshot event text to be redacted, got %q", getSnapshot.GetDiagnosticEvents[0].DiagnosticText)
	}
	if getSnapshot.GetDroppedStalePatchCount != 1 {
		parseT.Fatalf("expected dropped stale patch counter 1, got %d", getSnapshot.GetDroppedStalePatchCount)
	}
	if getSnapshot.GetIgnoredStaleDiagnosticCount != 1 {
		parseT.Fatalf("expected ignored stale diagnostic counter 1, got %d", getSnapshot.GetIgnoredStaleDiagnosticCount)
	}
	if getSnapshot.GetRepairTriggeredRemountCount != 1 {
		parseT.Fatalf("expected repair remount counter 1, got %d", getSnapshot.GetRepairTriggeredRemountCount)
	}
	if !getSnapshot.GetTransportDowngradeStatus.HasSnapshotDowngrade {
		parseT.Fatal("expected snapshot downgrade accounting")
	}
	if getSnapshot.GetTransportDowngradeStatus.GetSnapshotDowngrade.Reason != string(runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable) {
		parseT.Fatalf(
			"expected snapshot downgrade reason %q, got %q",
			runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable,
			getSnapshot.GetTransportDowngradeStatus.GetSnapshotDowngrade.Reason,
		)
	}
	if !getSnapshot.GetTransportDowngradeStatus.HasPatchDowngrade {
		parseT.Fatal("expected patch downgrade accounting")
	}
	if getSnapshot.GetTransportDowngradeStatus.GetPatchDowngrade.Reason != string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage) {
		parseT.Fatalf(
			"expected patch downgrade reason %q, got %q",
			runtime2.SharedPatchDowngradeReasonInvalidSharedPage,
			getSnapshot.GetTransportDowngradeStatus.GetPatchDowngrade.Reason,
		)
	}
	getSnapshot.GetDiagnosticEvents[0].DiagnosticText = "mutated"
	getNextSnapshot, hasNextSnapshot := buildHostRegionAdapter.GetHostRegionDiagnosticsSnapshot()
	if !hasNextSnapshot {
		parseT.Fatal("expected diagnostics snapshot after local mutation attempt")
	}
	if len(getNextSnapshot.GetDiagnosticEvents) == 0 {
		parseT.Fatal("expected diagnostics snapshot events after local mutation attempt")
	}
	if getNextSnapshot.GetDiagnosticEvents[0].DiagnosticText == "mutated" {
		parseT.Fatal("expected diagnostics snapshot getter to return immutable copies")
	}
}
