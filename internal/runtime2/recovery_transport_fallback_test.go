package runtime2

import "testing"

// TestHandleTransportDecodeFailureMalformedControlPayloadEntersFallback verifies malformed control-plane payloads trigger fallback.
func TestHandleTransportDecodeFailureMalformedControlPayloadEntersFallback(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseFailureErr := parseRecoveryCoordinator.HandleTransportDecodeFailure("region-a", TransportFailureKindMalformedControlPayload, 3, 8)
	if parseFailureErr != nil {
		parseTesting.Fatalf("HandleTransportDecodeFailure(control payload) error = %v", parseFailureErr)
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after malformed control payload")
	}
}

// TestHandleTransportDecodeFailureMalformedPatchPayloadEntersFallback verifies malformed patch payloads trigger fallback.
func TestHandleTransportDecodeFailureMalformedPatchPayloadEntersFallback(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseFailureErr := parseRecoveryCoordinator.HandleTransportDecodeFailure("region-a", TransportFailureKindMalformedPatchPayload, 3, 8)
	if parseFailureErr != nil {
		parseTesting.Fatalf("HandleTransportDecodeFailure(patch payload) error = %v", parseFailureErr)
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after malformed patch payload")
	}
}

// TestHandleTransportDecodeFailureMalformedSharedPageEntersFallback verifies malformed shared-memory pages trigger fallback.
func TestHandleTransportDecodeFailureMalformedSharedPageEntersFallback(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseFailureErr := parseRecoveryCoordinator.HandleTransportDecodeFailure("region-a", TransportFailureKindMalformedSharedMemoryPage, 3, 8)
	if parseFailureErr != nil {
		parseTesting.Fatalf("HandleTransportDecodeFailure(shared page) error = %v", parseFailureErr)
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after malformed shared-memory page")
	}
}
