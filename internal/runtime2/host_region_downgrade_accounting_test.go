package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHostRegionTransportDowngradeStatusTracksSnapshotAndPatchSeparately verifies snapshot and patch downgrade accounting are stored independently.
func TestHostRegionTransportDowngradeStatusTracksSnapshotAndPatchSeparately(parseT *testing.T) {
	buildHostRegionAdapter := buildHostRegionAdapterForTransportDispatchTests(parseT)
	parseCapabilityReport := buildSnapshotTransportCapabilityReport()
	getDispatchResult, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		parseCapabilityReport,
		nil,
	)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport returned error: %v", parseDispatchErr)
	}
	if !getDispatchResult.HasSnapshotTransport {
		parseT.Fatal("expected snapshot transport result")
	}
	parsePatchReadyEnvelope, parsePatchReadyErr := runtime2.BuildControlPatchReadyEnvelope("region-1", 2, 2, runtime2.TransportTierStructuredClone)
	if parsePatchReadyErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parsePatchReadyErr)
	}
	if _, parseDispatchPatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parsePatchReadyEnvelope); parseDispatchPatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(patch-ready) returned error: %v", parseDispatchPatchErr)
	}
	parsePatchDiagnosticEnvelope, parsePatchDiagnosticErr := runtime2.BuildControlDiagnosticEnvelope("region-1", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindPatchReady,
		TransportTier:  runtime2.TransportTierStructuredClone,
		DiagnosticDowngrade: &runtime2.DiagnosticDowngradeReason{
			Path:   runtime2.DiagnosticDowngradePathSharedMemory,
			Reason: string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage),
		},
		DiagnosticText: "patch downgrade",
	})
	if parsePatchDiagnosticErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parsePatchDiagnosticErr)
	}
	if _, parseDiagnosticDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parsePatchDiagnosticEnvelope); parseDiagnosticDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(diagnostic patch downgrade) returned error: %v", parseDiagnosticDispatchErr)
	}
	getDowngradeStatus := buildHostRegionAdapter.GetHostRegionTransportDowngradeStatus()
	if !getDowngradeStatus.HasSnapshotDowngrade {
		parseT.Fatal("expected snapshot downgrade accounting")
	}
	if getDowngradeStatus.GetSnapshotDowngrade.Path != runtime2.DiagnosticDowngradePathSharedMemory {
		parseT.Fatalf("expected snapshot downgrade path %q, got %q", runtime2.DiagnosticDowngradePathSharedMemory, getDowngradeStatus.GetSnapshotDowngrade.Path)
	}
	if getDowngradeStatus.GetSnapshotDowngrade.Reason != string(runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable) {
		parseT.Fatalf(
			"expected snapshot downgrade reason %q, got %q",
			runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable,
			getDowngradeStatus.GetSnapshotDowngrade.Reason,
		)
	}
	if !getDowngradeStatus.HasPatchDowngrade {
		parseT.Fatal("expected patch downgrade accounting")
	}
	if getDowngradeStatus.GetPatchDowngrade.Path != runtime2.DiagnosticDowngradePathSharedMemory {
		parseT.Fatalf("expected patch downgrade path %q, got %q", runtime2.DiagnosticDowngradePathSharedMemory, getDowngradeStatus.GetPatchDowngrade.Path)
	}
	if getDowngradeStatus.GetPatchDowngrade.Reason != string(runtime2.SharedPatchDowngradeReasonInvalidSharedPage) {
		parseT.Fatalf(
			"expected patch downgrade reason %q, got %q",
			runtime2.SharedPatchDowngradeReasonInvalidSharedPage,
			getDowngradeStatus.GetPatchDowngrade.Reason,
		)
	}
	if getDowngradeStatus.GetSnapshotTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected snapshot transport tier %q, got %q", runtime2.TransportTierStructuredClone, getDowngradeStatus.GetSnapshotTransportTier)
	}
	if getDowngradeStatus.GetPatchTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected patch transport tier %q, got %q", runtime2.TransportTierStructuredClone, getDowngradeStatus.GetPatchTransportTier)
	}
}
