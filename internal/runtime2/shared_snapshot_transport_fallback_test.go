package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestBuildSharedSnapshotTransportResultFallsBackWhenSharedMemoryUnavailable verifies unavailable shared memory downgrades to message transport.
func TestBuildSharedSnapshotTransportResultFallsBackWhenSharedMemoryUnavailable(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     21,
		Props:            map[string]any{"title": "Orders"},
	}
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	})
	parseResult, parseErr := runtime2.BuildSharedSnapshotTransportResult(parseEnvelope, parseCapabilityReport, nil)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotTransportResult returned error: %v", parseErr)
	}
	if parseResult.GetTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone transport fallback, got %q", parseResult.GetTransportTier)
	}
	if parseResult.GetDowngradeReason != runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable {
		parseT.Fatalf("expected shared-memory-unavailable reason, got %q", parseResult.GetDowngradeReason)
	}
	if len(parseResult.GetMessagePayload) == 0 {
		parseT.Fatal("expected fallback message payload")
	}
	parseDecodedEnvelope, parseDecodeErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parseResult.GetMessagePayload)
	if parseDecodeErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseDecodeErr)
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

// TestBuildSharedSnapshotTransportResultFallsBackWhenSharedPageInvalid verifies invalid shared-page publication downgrades to message transport.
func TestBuildSharedSnapshotTransportResultFallsBackWhenSharedPageInvalid(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     22,
		Props:            map[string]any{"title": "Orders"},
	}
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseSharedSnapshotPage, parseErr := runtime2.BuildSharedSnapshotPage(24)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseResult, parseErr := runtime2.BuildSharedSnapshotTransportResult(parseEnvelope, parseCapabilityReport, parseSharedSnapshotPage)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotTransportResult returned error: %v", parseErr)
	}
	if parseResult.GetTransportTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected invalid shared page fallback to structured-clone, got %q", parseResult.GetTransportTier)
	}
	if parseResult.GetDowngradeReason != runtime2.SharedSnapshotDowngradeReasonInvalidSharedPage {
		parseT.Fatalf("expected invalid-shared-page reason, got %q", parseResult.GetDowngradeReason)
	}
	if len(parseResult.GetMessagePayload) == 0 {
		parseT.Fatal("expected fallback message payload after invalid shared page")
	}
}

// TestBuildSharedSnapshotTransportResultFallbackPreservesIdentityFields verifies downgrade paths preserve region instance ID, epoch, and input version exactly.
func TestBuildSharedSnapshotTransportResultFallbackPreservesIdentityFields(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-7"),
		Epoch:            11,
		InputVersion:     31,
		Props:            map[string]any{"status": "ok"},
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
	if parseUnavailableResult.GetRegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("expected unavailable fallback region instance ID %q, got %q", parseEnvelope.RegionInstanceID, parseUnavailableResult.GetRegionInstanceID)
	}
	if parseUnavailableResult.GetEpoch != parseEnvelope.Epoch {
		parseT.Fatalf("expected unavailable fallback epoch %d, got %d", parseEnvelope.Epoch, parseUnavailableResult.GetEpoch)
	}
	if parseUnavailableResult.GetInputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected unavailable fallback input version %d, got %d", parseEnvelope.InputVersion, parseUnavailableResult.GetInputVersion)
	}
	parseUnavailableDecoded, parseUnavailableDecodeErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parseUnavailableResult.GetMessagePayload)
	if parseUnavailableDecodeErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON(unavailable) returned error: %v", parseUnavailableDecodeErr)
	}
	if parseUnavailableDecoded.RegionInstanceID != parseEnvelope.RegionInstanceID || parseUnavailableDecoded.Epoch != parseEnvelope.Epoch || parseUnavailableDecoded.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected unavailable fallback decoded identity fields to match envelope, got %+v", parseUnavailableDecoded)
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
	if parseInvalidResult.GetRegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("expected invalid-page fallback region instance ID %q, got %q", parseEnvelope.RegionInstanceID, parseInvalidResult.GetRegionInstanceID)
	}
	if parseInvalidResult.GetEpoch != parseEnvelope.Epoch {
		parseT.Fatalf("expected invalid-page fallback epoch %d, got %d", parseEnvelope.Epoch, parseInvalidResult.GetEpoch)
	}
	if parseInvalidResult.GetInputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected invalid-page fallback input version %d, got %d", parseEnvelope.InputVersion, parseInvalidResult.GetInputVersion)
	}
	parseInvalidDecoded, parseInvalidDecodeErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parseInvalidResult.GetMessagePayload)
	if parseInvalidDecodeErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON(invalid) returned error: %v", parseInvalidDecodeErr)
	}
	if parseInvalidDecoded.RegionInstanceID != parseEnvelope.RegionInstanceID || parseInvalidDecoded.Epoch != parseEnvelope.Epoch || parseInvalidDecoded.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected invalid-page fallback decoded identity fields to match envelope, got %+v", parseInvalidDecoded)
	}
}
