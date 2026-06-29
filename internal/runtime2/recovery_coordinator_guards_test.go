package runtime2

import "testing"

// TestRecoveryCoordinatorRejectsNilAndInvalidFailureKinds verifies recovery coordinator helpers reject nil coordinators and invalid failure-kind inputs.
func TestRecoveryCoordinatorRejectsNilAndInvalidFailureKinds(parseT *testing.T) {
	var parseNilRecoveryCoordinator *RecoveryCoordinator
	if parseErr := parseNilRecoveryCoordinator.HandleTransportDecodeFailure("region-a", TransportFailureKindMalformedPatchPayload, 1, 2); parseErr == nil {
		parseT.Fatal("expected nil recovery coordinator transport failure to fail")
	}
	if parseErr := parseNilRecoveryCoordinator.HandleDOMCommitFailure("region-a", DOMCommitFailureKindMissingNodeLookup, 1, 2); parseErr == nil {
		parseT.Fatal("expected nil recovery coordinator DOM failure to fail")
	}
	if _, parseErr := parseNilRecoveryCoordinator.HandleWorkerDeath("region-a", true, 3, 2); parseErr == nil {
		parseT.Fatal("expected nil recovery coordinator worker death to fail")
	}
	if parseNilRecoveryCoordinator.GetRegionLocalVersion("region-a") != 0 {
		parseT.Fatal("expected nil recovery coordinator local version to be zero")
	}
	if parseNilRecoveryCoordinator.ClearRegionLocalFallback("region-a") {
		parseT.Fatal("expected nil recovery coordinator clear fallback to return false")
	}
	if _, hasFallbackState := parseNilRecoveryCoordinator.GetRegionFallbackState("region-a"); hasFallbackState {
		parseT.Fatal("expected nil recovery coordinator fallback state lookup to fail")
	}

	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	if parseErr := parseRecoveryCoordinator.HandleTransportDecodeFailure("region-a", "", 1, 2); parseErr == nil {
		parseT.Fatal("expected missing transport failure kind to fail")
	}
	if parseErr := parseRecoveryCoordinator.HandleTransportDecodeFailure("region-a", TransportFailureKind("unsupported"), 1, 2); parseErr == nil {
		parseT.Fatal("expected unsupported transport failure kind to fail")
	}
	if parseErr := parseRecoveryCoordinator.HandleDOMCommitFailure("region-a", "", 1, 2); parseErr == nil {
		parseT.Fatal("expected missing DOM commit failure kind to fail")
	}
	if parseErr := parseRecoveryCoordinator.HandleDOMCommitFailure("region-a", DOMCommitFailureKind("unsupported"), 1, 2); parseErr == nil {
		parseT.Fatal("expected unsupported DOM commit failure kind to fail")
	}
}

// TestRecoveryCoordinatorWorkerDeathFallbackAndClear verifies worker-death fallback without reassignment records fallback ownership and can later be cleared.
func TestRecoveryCoordinatorWorkerDeathFallbackAndClear(parseT *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryResult, parseRecoveryErr := parseRecoveryCoordinator.HandleWorkerDeath("region-a", false, 0, 12)
	if parseRecoveryErr != nil {
		parseT.Fatalf("HandleWorkerDeath(no reassign) returned error: %v", parseRecoveryErr)
	}
	if !parseRecoveryResult.HasFallbackEntered || parseRecoveryResult.HasReassigned {
		parseT.Fatalf("HandleWorkerDeath(no reassign) result = %+v, want fallback without reassignment", parseRecoveryResult)
	}

	parseFallbackState, hasFallbackState := parseRecoveryCoordinator.GetRegionFallbackState("region-a")
	if !hasFallbackState {
		parseT.Fatal("expected region fallback state after worker-death fallback")
	}
	if parseFallbackState.GetReason != string(WorkerDeathFailureKindNoReassign) {
		parseT.Fatalf("fallback reason = %q, want %q", parseFallbackState.GetReason, WorkerDeathFailureKindNoReassign)
	}

	if !parseRecoveryCoordinator.ClearRegionLocalFallback("region-a") {
		parseT.Fatal("expected fallback clear to succeed")
	}
	if parseRecoveryCoordinator.ClearRegionLocalFallback("region-a") {
		parseT.Fatal("expected fallback clear to return false after region is already cleared")
	}
}

// TestRecoveryCoordinatorEnterAndSetLocalVersionIgnoreBlankRegion verifies blank-region fallback and local-version updates are ignored and local versions remain monotonic.
func TestRecoveryCoordinatorEnterAndSetLocalVersionIgnoreBlankRegion(parseT *testing.T) {
	parseRecoveryCoordinator := BuildRecoveryCoordinator()
	parseRecoveryCoordinator.EnterRegionLocalFallback("region-a", "transport-failure", 3, 8)
	parseRecoveryCoordinator.EnterRegionLocalFallback(" \n\t ", "ignored", 9, 99)
	parseRecoveryCoordinator.SetRegionLocalVersion("region-a", 12)
	parseRecoveryCoordinator.SetRegionLocalVersion("region-a", 7)
	parseRecoveryCoordinator.SetRegionLocalVersion(" \t ", 99)

	if parseRecoveryCoordinator.GetRegionLocalVersion("region-a") != 12 {
		parseT.Fatalf("GetRegionLocalVersion(region-a) = %d, want 12", parseRecoveryCoordinator.GetRegionLocalVersion("region-a"))
	}
	if parseRecoveryCoordinator.GetRegionLocalVersion(" \t ") != 0 {
		parseT.Fatal("expected blank-region local version to remain zero")
	}
	if parseRecoveryCoordinator.IsRegionLocalOwnership(" \n\t ") {
		parseT.Fatal("expected blank-region fallback ownership to remain false")
	}
}
