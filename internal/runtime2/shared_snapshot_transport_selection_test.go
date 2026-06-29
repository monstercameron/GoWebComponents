package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestGetSharedSnapshotTransportTierPrefersSharedWhenContractAllows verifies selection prefers shared-buffer transport only when capability and page availability requirements are met.
func TestGetSharedSnapshotTransportTierPrefersSharedWhenContractAllows(parseT *testing.T) {
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseTier, parseErr := runtime2.GetSharedSnapshotTransportTier(parseCapabilityReport, true)
	if parseErr != nil {
		parseT.Fatalf("GetSharedSnapshotTransportTier returned error: %v", parseErr)
	}
	if parseTier != runtime2.TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer tier, got %q", parseTier)
	}
}

// TestGetSharedSnapshotTransportTierRequiresPageAvailability verifies missing shared-page availability downgrades to structured-clone transport.
func TestGetSharedSnapshotTransportTierRequiresPageAvailability(parseT *testing.T) {
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseTier, parseErr := runtime2.GetSharedSnapshotTransportTier(parseCapabilityReport, false)
	if parseErr != nil {
		parseT.Fatalf("GetSharedSnapshotTransportTier returned error: %v", parseErr)
	}
	if parseTier != runtime2.TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone tier when shared page unavailable, got %q", parseTier)
	}
}

// TestGetSharedSnapshotTransportTierRejectsInvalidCapabilityContract verifies invalid shared-memory capability contracts fail selection.
func TestGetSharedSnapshotTransportTierRejectsInvalidCapabilityContract(parseT *testing.T) {
	parseInvalidCapabilityReport := runtime2.CapabilityReport{
		HasWorkerSupport:                true,
		HasSharedMemoryTransportSupport: true,
	}
	if _, parseErr := runtime2.GetSharedSnapshotTransportTier(parseInvalidCapabilityReport, true); parseErr == nil {
		parseT.Fatal("expected invalid capability contract to fail")
	}
}

// TestBuildSharedSnapshotTransportResultUsesSharedBufferWhenContractAllows verifies dispatch uses shared-buffer transport when the capability contract allows it.
func TestBuildSharedSnapshotTransportResultUsesSharedBufferWhenContractAllows(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            3,
		InputVersion:     8,
		Props:            map[string]any{"status": "healthy"},
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
		parseT.Fatal("expected shared generation after shared-buffer publish")
	}
	if parseResult.GetDowngradeReason != "" {
		parseT.Fatalf("expected empty downgrade reason for shared-buffer success, got %q", parseResult.GetDowngradeReason)
	}
	if len(parseResult.GetMessagePayload) != 0 {
		parseT.Fatal("expected no message payload when shared-buffer transport is active")
	}
}

// TestParseSharedSnapshotDowngradeReasonAcceptsKnownReasons verifies downgrade-reason plumbing exposes stable known values.
func TestParseSharedSnapshotDowngradeReasonAcceptsKnownReasons(parseT *testing.T) {
	parseKnownReasons := []string{
		"",
		string(runtime2.SharedSnapshotDowngradeReasonSharedMemoryUnavailable),
		string(runtime2.SharedSnapshotDowngradeReasonInvalidSharedPage),
	}
	for _, parseReason := range parseKnownReasons {
		parseParsedReason, parseErr := runtime2.ParseSharedSnapshotDowngradeReason(parseReason)
		if parseErr != nil {
			parseT.Fatalf("ParseSharedSnapshotDowngradeReason(%q) returned error: %v", parseReason, parseErr)
		}
		if string(parseParsedReason) != parseReason {
			parseT.Fatalf("expected parsed downgrade reason %q, got %q", parseReason, parseParsedReason)
		}
	}
}

// TestParseSharedSnapshotDowngradeReasonRejectsUnknownReason verifies unsupported downgrade reasons fail validation.
func TestParseSharedSnapshotDowngradeReasonRejectsUnknownReason(parseT *testing.T) {
	if _, parseErr := runtime2.ParseSharedSnapshotDowngradeReason("network-flake"); parseErr == nil {
		parseT.Fatal("expected unsupported downgrade reason to fail")
	}
}
