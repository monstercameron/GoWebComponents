package runtime2

import (
	"strings"
	"testing"
)

// buildMountedHostRegionAdapterForDiagnosticsTest creates one mounted host region adapter used for direct diagnostics helper coverage.
func buildMountedHostRegionAdapterForDiagnosticsTest(parseT *testing.T) *HostRegionAdapter {
	parseT.Helper()
	parseHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	return parseHostRegionAdapter
}

// TestHandleHostRegionDiagnosticEnvelopeRejectsNilAdapterAndStoresRedactedDiagnostic verifies the direct host diagnostic entrypoint rejects nil adapters and stores redacted diagnostics for mounted regions.
func TestHandleHostRegionDiagnosticEnvelopeRejectsNilAdapterAndStoresRedactedDiagnostic(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionDiagnosticEnvelope(ControlEnvelope{}); parseErr == nil {
		parseT.Fatal("expected nil host region adapter diagnostic handling to fail")
	}

	parseHostRegionAdapter := buildMountedHostRegionAdapterForDiagnosticsTest(parseT)
	parseEnvelope, parseEnvelopeErr := BuildControlDiagnosticEnvelope("region-1", ControlDiagnosticEnvelopeSpec{
		DiagnosticType: DiagnosticEventKindUpdate,
		DiagnosticText: "token=secret-123",
	})
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseDiagnosticResult, parseDiagnosticErr := parseHostRegionAdapter.HandleHostRegionDiagnosticEnvelope(parseEnvelope)
	if parseDiagnosticErr != nil {
		parseT.Fatalf("HandleHostRegionDiagnosticEnvelope returned error: %v", parseDiagnosticErr)
	}
	if !parseDiagnosticResult.HasStored {
		parseT.Fatalf("expected stored diagnostic result, got %+v", parseDiagnosticResult)
	}
	if strings.Contains(parseDiagnosticResult.GetDiagnostic.DiagnosticText, "secret-123") {
		parseT.Fatalf("expected direct diagnostic result text to be redacted, got %q", parseDiagnosticResult.GetDiagnostic.DiagnosticText)
	}
	if len(parseHostRegionAdapter.storeHostRegionDiagnosticRing) != 1 {
		parseT.Fatalf("expected one stored diagnostic entry, got %d", len(parseHostRegionAdapter.storeHostRegionDiagnosticRing))
	}
}

// TestHandleHostRegionDiagnosticEnvelopeValidatedRejectsWrongKindAndRegion verifies validated diagnostic handling rejects non-diagnostic envelopes and wrong-region diagnostics.
func TestHandleHostRegionDiagnosticEnvelopeValidatedRejectsWrongKindAndRegion(parseT *testing.T) {
	parseHostRegionAdapter := buildMountedHostRegionAdapterForDiagnosticsTest(parseT)
	if _, parseErr := parseHostRegionAdapter.handleHostRegionDiagnosticEnvelopeValidated(BuildControlReadyEnvelope()); parseErr == nil {
		parseT.Fatal("expected non-diagnostic control envelope to fail validated diagnostic handling")
	}

	parseEnvelope, parseEnvelopeErr := BuildControlDiagnosticEnvelope("region-1", ControlDiagnosticEnvelopeSpec{
		DiagnosticType: DiagnosticEventKindUpdate,
		DiagnosticText: "region mismatch diagnostic",
	})
	if parseEnvelopeErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseEnvelope.RegionInstanceID = RegionInstanceID("region-other")
	if _, parseErr := parseHostRegionAdapter.handleHostRegionDiagnosticEnvelopeValidated(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected wrong-region diagnostic to fail validated diagnostic handling")
	}
}

// TestHostRegionDiagnosticHelpersHandleNilFloorAndPatchTier verifies diagnostic helper accessors reject nil adapters and store parsed patch transport tiers.
func TestHostRegionDiagnosticHelpersHandleNilFloorAndPatchTier(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	if parseNilHostRegionAdapter.getHostRegionDiagnosticVersionFloor(CoordinatorEntry{}) != 0 {
		parseT.Fatal("expected nil host region adapter diagnostic version floor to be zero")
	}
	if parseErr := parseNilHostRegionAdapter.SetHostRegionPatchTransportTier(TransportTierBinary); parseErr == nil {
		parseT.Fatal("expected nil host region adapter patch tier setter to fail")
	}

	parseHostRegionAdapter := buildMountedHostRegionAdapterForDiagnosticsTest(parseT)
	if parseErr := parseHostRegionAdapter.SetHostRegionPatchTransportTier(""); parseErr == nil {
		parseT.Fatal("expected empty patch transport tier to fail")
	}
	if parseErr := parseHostRegionAdapter.SetHostRegionPatchTransportTier(TransportTierStructuredClone); parseErr != nil {
		parseT.Fatalf("SetHostRegionPatchTransportTier(valid) returned error: %v", parseErr)
	}
	if parseHostRegionAdapter.storeHostRegionPatchTier != TransportTierStructuredClone {
		parseT.Fatalf("stored patch transport tier = %q, want %q", parseHostRegionAdapter.storeHostRegionPatchTier, TransportTierStructuredClone)
	}
	if parseHostRegionAdapter.storeHostRegionTransportTier != TransportTierStructuredClone {
		parseT.Fatalf("stored latest transport tier = %q, want %q", parseHostRegionAdapter.storeHostRegionTransportTier, TransportTierStructuredClone)
	}
}
