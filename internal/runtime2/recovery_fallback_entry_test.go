package runtime2

import "testing"

// TestEnterRegionLocalFallbackSwapsToLocalOwnership verifies fallback marks one region as locally owned.
func TestEnterRegionLocalFallbackSwapsToLocalOwnership(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryCoordinator.EnterRegionLocalFallback("region-a", "transport-failure", 3, 8)
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after fallback")
	}
}

// TestHandleWorkerPatchAfterFallbackIsIgnored verifies worker patches for a fallback-owned region are ignored.
func TestHandleWorkerPatchAfterFallbackIsIgnored(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryCoordinator.EnterRegionLocalFallback("region-a", "transport-failure", 3, 8)
	parsePatchDecision := parseRecoveryCoordinator.HandleWorkerPatch("region-a", 3, 9)
	if !parsePatchDecision.HasIgnored {
		parseTesting.Fatal("HandleWorkerPatch(region-a fallback) expected ignored patch decision")
	}
}

// TestHandleWorkerPatchForOtherRegionsKeepsWorking verifies fallback for one region does not block other regions.
func TestHandleWorkerPatchForOtherRegionsKeepsWorking(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryCoordinator.EnterRegionLocalFallback("region-a", "transport-failure", 3, 8)
	parsePatchDecision := parseRecoveryCoordinator.HandleWorkerPatch("region-b", 3, 9)
	if parsePatchDecision.HasIgnored {
		parseTesting.Fatal("HandleWorkerPatch(region-b) expected accepted patch decision")
	}
}
