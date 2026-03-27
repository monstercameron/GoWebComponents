package runtime2_test

import (
	"testing"
)

// TestHandleHostRegionPatchReadyRejectsBeforeFallbackOwnershipBegins verifies patch-ready is rejected while fallback ownership is pending.
func TestHandleHostRegionPatchReadyRejectsBeforeFallbackOwnershipBegins(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionStructuredCloneDecodeFailure(5); parseErr != nil {
		parseT.Fatalf("HandleHostRegionStructuredCloneDecodeFailure returned error: %v", parseErr)
	}
	getPatchReadyResult, parseErr := buildHostRegionAdapter.HandleHostRegionPatchReady(5)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady returned error: %v", parseErr)
	}
	if !getPatchReadyResult.HasIgnored {
		parseT.Fatal("expected patch-ready to be ignored while fallback ownership is pending")
	}
	if getPatchReadyResult.GetIgnoreReason != "fallback-pending" {
		parseT.Fatalf("expected fallback-pending ignore reason, got %q", getPatchReadyResult.GetIgnoreReason)
	}
}

// TestHandleHostRegionWorkerOutputBlockedDuringFallback verifies fallback ownership blocks worker commit attempts.
func TestHandleHostRegionWorkerOutputBlockedDuringFallback(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionBinaryDecodeFailure(6); parseErr != nil {
		parseT.Fatalf("HandleHostRegionBinaryDecodeFailure returned error: %v", parseErr)
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackOwnershipBegin returned error: %v", parseErr)
	}
	getWorkerOutputResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(6)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseErr)
	}
	if !getWorkerOutputResult.HasIgnored {
		parseT.Fatal("expected worker output to be blocked while fallback ownership is active")
	}
	if getWorkerOutputResult.HasCommitted {
		parseT.Fatal("expected blocked worker output not to commit")
	}
}

// TestHandleHostRegionPatchReadyRejectsBeforeRepairCompletes verifies patch-ready is rejected while repair remount is pending.
func TestHandleHostRegionPatchReadyRejectsBeforeRepairCompletes(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	getWorkerDeathResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	if !getWorkerDeathResult.IsRepairPending {
		parseT.Fatal("expected repair-pending state after worker death reassignment")
	}
	getPatchReadyResult, parseErr := buildHostRegionAdapter.HandleHostRegionPatchReady(getWorkerDeathResult.GetVersionFloor)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady returned error: %v", parseErr)
	}
	if !getPatchReadyResult.HasIgnored {
		parseT.Fatal("expected patch-ready to be ignored while repair remount is pending")
	}
	if getPatchReadyResult.GetIgnoreReason != "fallback-active" && getPatchReadyResult.GetIgnoreReason != "repair-pending" {
		parseT.Fatalf("expected fallback-active or repair-pending ignore reason, got %q", getPatchReadyResult.GetIgnoreReason)
	}
}
