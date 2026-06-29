package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestBuildSharedSnapshotTransportResultEndToEndSharedMemoryRoundTrip verifies shared-memory transport publishes and reads one snapshot envelope end to end.
func TestBuildSharedSnapshotTransportResultEndToEndSharedMemoryRoundTrip(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            12,
		InputVersion:     40,
		Props:            map[string]any{"status": "ready"},
	}
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseSharedSnapshotPage, parsePageErr := runtime2.BuildSharedSnapshotPage(1024)
	if parsePageErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parsePageErr)
	}
	parseResult, parseErr := runtime2.BuildSharedSnapshotTransportResult(parseEnvelope, parseCapabilityReport, parseSharedSnapshotPage)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotTransportResult returned error: %v", parseErr)
	}
	if parseResult.GetTransportTier != runtime2.TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer transport, got %q", parseResult.GetTransportTier)
	}
	if parseResult.GetSharedGeneration == 0 {
		parseT.Fatal("expected shared generation from shared-buffer dispatch")
	}
	if _, parseErr := parseSharedSnapshotPage.HandleSharedSnapshotReadHeaderAtGeneration(parseResult.GetSharedGeneration); parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotReadHeaderAtGeneration returned error: %v", parseErr)
	}
	parseDecodedEnvelope, parseDecodeErr := parseSharedSnapshotPage.ParseSharedSnapshotEnvelope()
	if parseDecodeErr != nil {
		parseT.Fatalf("ParseSharedSnapshotEnvelope returned error: %v", parseDecodeErr)
	}
	if parseDecodedEnvelope.RegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("expected decoded region instance ID %q, got %q", parseEnvelope.RegionInstanceID, parseDecodedEnvelope.RegionInstanceID)
	}
	if parseDecodedEnvelope.Epoch != parseEnvelope.Epoch {
		parseT.Fatalf("expected decoded epoch %d, got %d", parseEnvelope.Epoch, parseDecodedEnvelope.Epoch)
	}
	if parseDecodedEnvelope.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected decoded input version %d, got %d", parseEnvelope.InputVersion, parseDecodedEnvelope.InputVersion)
	}
}

// TestBuildSharedSnapshotTransportResultEndToEndDowngradeRoundTrip verifies downgrade dispatches still round-trip through message transport with preserved snapshot identity fields.
func TestBuildSharedSnapshotTransportResultEndToEndDowngradeRoundTrip(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-2"),
		Epoch:            13,
		InputVersion:     41,
		Props:            map[string]any{"status": "degraded"},
	}

	parseUnavailableCapabilities := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	})
	parseUnavailableResult, parseUnavailableErr := runtime2.BuildSharedSnapshotTransportResult(parseEnvelope, parseUnavailableCapabilities, nil)
	if parseUnavailableErr != nil {
		parseT.Fatalf("BuildSharedSnapshotTransportResult(unavailable) returned error: %v", parseUnavailableErr)
	}
	if parseUnavailableResult.GetTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected unavailable downgrade to structured-clone, got %q", parseUnavailableResult.GetTransportTier)
	}
	if parseUnavailableResult.GetDowngradeReason != runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable {
		parseT.Fatalf("expected unavailable downgrade reason, got %q", parseUnavailableResult.GetDowngradeReason)
	}
	parseUnavailableDecoded, parseUnavailableDecodeErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parseUnavailableResult.GetMessagePayload)
	if parseUnavailableDecodeErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON(unavailable) returned error: %v", parseUnavailableDecodeErr)
	}
	if parseUnavailableDecoded.RegionInstanceID != parseEnvelope.RegionInstanceID || parseUnavailableDecoded.Epoch != parseEnvelope.Epoch || parseUnavailableDecoded.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected unavailable downgrade identity fields to match envelope, got %+v", parseUnavailableDecoded)
	}

	parseInvalidCapabilities := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseSharedSnapshotPage, parsePageErr := runtime2.BuildSharedSnapshotPage(24)
	if parsePageErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parsePageErr)
	}
	parseInvalidResult, parseInvalidErr := runtime2.BuildSharedSnapshotTransportResult(parseEnvelope, parseInvalidCapabilities, parseSharedSnapshotPage)
	if parseInvalidErr != nil {
		parseT.Fatalf("BuildSharedSnapshotTransportResult(invalid) returned error: %v", parseInvalidErr)
	}
	if parseInvalidResult.GetTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected invalid-page downgrade to structured-clone, got %q", parseInvalidResult.GetTransportTier)
	}
	if parseInvalidResult.GetDowngradeReason != runtime2.SharedSnapshotDowngradeReasonInvalidSharedPage {
		parseT.Fatalf("expected invalid-page downgrade reason, got %q", parseInvalidResult.GetDowngradeReason)
	}
	parseInvalidDecoded, parseInvalidDecodeErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parseInvalidResult.GetMessagePayload)
	if parseInvalidDecodeErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON(invalid) returned error: %v", parseInvalidDecodeErr)
	}
	if parseInvalidDecoded.RegionInstanceID != parseEnvelope.RegionInstanceID || parseInvalidDecoded.Epoch != parseEnvelope.Epoch || parseInvalidDecoded.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected invalid-page downgrade identity fields to match envelope, got %+v", parseInvalidDecoded)
	}
}
