package runtime2

import "testing"

// TestHandleWorkerDeathCanTriggerReassignment verifies worker death can reassign when recovery support exists.
func TestHandleWorkerDeathCanTriggerReassignment(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryResult, parseRecoveryErr := parseRecoveryCoordinator.HandleWorkerDeath("region-a", true, 5, 12)
	if parseRecoveryErr != nil {
		parseTesting.Fatalf("HandleWorkerDeath(reassign supported) error = %v", parseRecoveryErr)
	}
	if !parseRecoveryResult.HasReassigned {
		parseTesting.Fatal("HandleWorkerDeath(reassign supported) expected reassignment")
	}
	if parseRecoveryResult.HasFallbackEntered {
		parseTesting.Fatal("HandleWorkerDeath(reassign supported) expected no fallback")
	}
}

// TestHandleWorkerDeathReassignmentUsesFreshEpoch verifies reassignment reports the remount epoch explicitly.
func TestHandleWorkerDeathReassignmentUsesFreshEpoch(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryResult, parseRecoveryErr := parseRecoveryCoordinator.HandleWorkerDeath("region-a", true, 7, 12)
	if parseRecoveryErr != nil {
		parseTesting.Fatalf("HandleWorkerDeath(reassign supported) error = %v", parseRecoveryErr)
	}
	if parseRecoveryResult.GetRemountEpoch != 7 {
		parseTesting.Fatalf("HandleWorkerDeath(reassign supported) remount epoch = %d, want 7", parseRecoveryResult.GetRemountEpoch)
	}
}

// TestHandleWorkerDeathRepairFailureEntersFallback verifies reassignment failure falls back to local ownership.
func TestHandleWorkerDeathRepairFailureEntersFallback(parseTesting *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryResult, parseRecoveryErr := parseRecoveryCoordinator.HandleWorkerDeath("region-a", true, 0, 12)
	if parseRecoveryErr == nil {
		parseTesting.Fatal("HandleWorkerDeath(repair failure) error = nil, want error")
	}
	if !parseRecoveryResult.HasFallbackEntered {
		parseTesting.Fatal("HandleWorkerDeath(repair failure) expected fallback result")
	}
	if !parseRecoveryCoordinator.IsRegionLocalOwnership("region-a") {
		parseTesting.Fatal("IsRegionLocalOwnership(region-a) expected true after repair failure")
	}
}
