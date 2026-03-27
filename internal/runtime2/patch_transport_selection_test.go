package runtime2

import "testing"

// TestSelectPatchTransportTierPrefersShared verifies patch transport selection prefers shared-buffer when supported and available.
func TestSelectPatchTransportTierPrefersShared(parseT *testing.T) {
	parseCapabilityReport := BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseTier, parseTierErr := SelectPatchTransportTier(parseCapabilityReport, true)
	if parseTierErr != nil {
		parseT.Fatalf("SelectPatchTransportTier returned error: %v", parseTierErr)
	}
	if parseTier != TransportTierSharedBuffer {
		parseT.Fatalf("expected shared-buffer tier, got %q", parseTier)
	}
}

// TestSelectPatchTransportTierPrefersBinaryWithoutSharedPage verifies patch transport selection falls back to binary when shared page availability is missing.
func TestSelectPatchTransportTierPrefersBinaryWithoutSharedPage(parseT *testing.T) {
	parseCapabilityReport := BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseTier, parseTierErr := SelectPatchTransportTier(parseCapabilityReport, false)
	if parseTierErr != nil {
		parseT.Fatalf("SelectPatchTransportTier returned error: %v", parseTierErr)
	}
	if parseTier != TransportTierBinary {
		parseT.Fatalf("expected binary tier, got %q", parseTier)
	}
}

// TestSelectPatchTransportTierFallsBackToStructuredClone verifies patch transport selection falls back to structured-clone when binary and shared transport are unavailable.
func TestSelectPatchTransportTierFallsBackToStructuredClone(parseT *testing.T) {
	parseCapabilityReport := BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport:          true,
		HasMessagePortSupport:     true,
		HasStructuredCloneSupport: true,
	})
	parseTier, parseTierErr := SelectPatchTransportTier(parseCapabilityReport, false)
	if parseTierErr != nil {
		parseT.Fatalf("SelectPatchTransportTier returned error: %v", parseTierErr)
	}
	if parseTier != TransportTierStructuredClone {
		parseT.Fatalf("expected structured-clone tier, got %q", parseTier)
	}
}

// TestSelectPatchTransportTierRejectsUnsupportedContract verifies patch transport selection fails when no transport tier is supported.
func TestSelectPatchTransportTierRejectsUnsupportedContract(parseT *testing.T) {
	parseCapabilityReport := BuildCapabilityReport(CapabilitySource{
		HasWorkerSupport: true,
	})
	if _, parseTierErr := SelectPatchTransportTier(parseCapabilityReport, false); parseTierErr == nil {
		parseT.Fatal("expected unsupported patch transport contract to fail")
	}
}
