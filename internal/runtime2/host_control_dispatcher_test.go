package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostControlEnvelopeDispatchesPatchReady verifies patch-ready envelopes route into host patch-ready gating.
func TestHandleHostControlEnvelopeDispatchesPatchReady(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	parseEnvelope, parseEnvelopeErr := runtime2.BuildControlPatchReadyEnvelope("region-1", 1, 1, runtime2.TransportTierStructuredClone)
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(patch-ready) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasPatchReadyResult {
		parseT.Fatal("expected patch-ready dispatch result")
	}
	if !parseDispatchResult.GetPatchReadyResult.HasAccepted {
		parseT.Fatal("expected accepted patch-ready result")
	}
}

// TestHandleHostControlEnvelopeDispatchesDiagnostic verifies diagnostic envelopes route into host diagnostic handling.
func TestHandleHostControlEnvelopeDispatchesDiagnostic(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	parseEnvelope, parseEnvelopeErr := runtime2.BuildControlDiagnosticEnvelope("region-1", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindUpdate,
		DiagnosticText: "update complete",
	})
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(diagnostic) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasDiagnosticResult {
		parseT.Fatal("expected diagnostic dispatch result")
	}
	if parseDispatchResult.GetDiagnosticType != runtime2.DiagnosticEventKindUpdate {
		parseT.Fatalf("expected diagnostic type %q, got %q", runtime2.DiagnosticEventKindUpdate, parseDispatchResult.GetDiagnosticType)
	}
}

// TestHandleHostControlEnvelopeDispatchesRestart verifies restart envelopes route into host restart handling.
func TestHandleHostControlEnvelopeDispatchesRestart(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	parseEnvelope, parseEnvelopeErr := runtime2.BuildControlRestartEnvelope("region-1", 3)
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlRestartEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(restart) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasRestartResult {
		parseT.Fatal("expected restart dispatch result")
	}
	parseCoordinatorEntry, parseHasCoordinatorEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry("region-1")
	if !parseHasCoordinatorEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseCoordinatorEntry.Epoch != 3 {
		parseT.Fatalf("expected restarted epoch 3, got %d", parseCoordinatorEntry.Epoch)
	}
}

// TestHandleHostControlEnvelopeDispatchesPong verifies pong envelopes route into host liveness dispatch handling.
func TestHandleHostControlEnvelopeDispatchesPong(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	parseEnvelope, parseEnvelopeErr := runtime2.BuildControlPongEnvelope("shard-a", 5)
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlPongEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, parseEnvelope)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostControlEnvelope(pong) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasPongResult {
		parseT.Fatal("expected pong dispatch result")
	}
	if parseDispatchResult.GetPongShardID != "shard-a" {
		parseT.Fatalf("expected pong shard %q, got %q", "shard-a", parseDispatchResult.GetPongShardID)
	}
	if parseDispatchResult.GetPongSequence != 5 {
		parseT.Fatalf("expected pong sequence 5, got %d", parseDispatchResult.GetPongSequence)
	}
}

// TestHandleHostControlEnvelopeRejectsUnsupportedKind verifies unsupported host dispatch kinds fail clearly.
func TestHandleHostControlEnvelopeRejectsUnsupportedKind(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	_, parseDispatchErr := runtime2.HandleHostControlEnvelope(buildHostRegionAdapter, runtime2.BuildControlReadyEnvelope())
	if parseDispatchErr == nil {
		parseT.Fatal("expected unsupported host dispatch kind to fail")
	}
}
