package runtime2

import "testing"

// TestHandleWorkerPatchStaleAfterFallbackIsIgnored verifies stale patch output is ignored for fallback-owned regions.
func TestHandleWorkerPatchStaleAfterFallbackIsIgnored(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryCoordinator.EnterRegionLocalFallback("region-a", "transport-failure", 3, 8)
	parsePatchDecision := parseRecoveryCoordinator.HandleWorkerPatch("region-a", 2, 7)
	if !parsePatchDecision.HasIgnored {
		parseTesting.Fatal("HandleWorkerPatch(stale after fallback) expected ignored decision")
	}
}

// TestHandleWorkerDiagnosticStaleAfterFallbackDoesNotRevive verifies stale diagnostics do not restore worker ownership.
func TestHandleWorkerDiagnosticStaleAfterFallbackDoesNotRevive(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryCoordinator.EnterRegionLocalFallback("region-a", "transport-failure", 3, 8)
	parseDiagnosticDecision := parseRecoveryCoordinator.HandleWorkerDiagnostic("region-a", 2, 7)
	if !parseDiagnosticDecision.HasIgnored {
		parseTesting.Fatal("HandleWorkerDiagnostic(stale after fallback) expected ignored decision")
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after stale diagnostic")
	}
}

// TestSetRegionLocalVersionAfterFallbackRemainsAuthoritative verifies local state stays authoritative after fallback.
func TestSetRegionLocalVersionAfterFallbackRemainsAuthoritative(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryCoordinator.EnterRegionLocalFallback("region-a", "transport-failure", 3, 8)
	parseRecoveryCoordinator.SetRegionLocalVersion("region-a", 12)
	parsePatchDecision := parseRecoveryCoordinator.HandleWorkerPatch("region-a", 3, 11)
	if !parsePatchDecision.HasIgnored {
		parseTesting.Fatal("HandleWorkerPatch(after local advance) expected ignored decision")
	}
	parseLocalVersion := parseRecoveryCoordinator.GetRegionLocalVersion("region-a")
	if parseLocalVersion != 12 {
		parseTesting.Fatalf("GetRegionLocalVersion(region-a) = %d, want 12", parseLocalVersion)
	}
}
