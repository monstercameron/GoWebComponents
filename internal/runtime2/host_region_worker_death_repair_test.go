package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionWorkerDeathAllocatesFreshRemountEpoch verifies worker death reassignment allocates a fresh remount epoch.
func TestHandleHostRegionWorkerDeathAllocatesFreshRemountEpoch(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionUpdate(4); parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate returned error: %v", parseErr)
	}
	getWorkerDeathResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	if !getWorkerDeathResult.HasReassigned {
		parseT.Fatal("expected worker death reassignment result")
	}
	if getWorkerDeathResult.GetRemountEpoch <= 2 {
		parseT.Fatalf("expected fresh remount epoch above 2, got %d", getWorkerDeathResult.GetRemountEpoch)
	}
}

// TestHandleHostRegionWorkerDeathPropagatesRemountEpochToCoordinator verifies remount epoch is propagated into coordinator state.
func TestHandleHostRegionWorkerDeathPropagatesRemountEpochToCoordinator(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	getWorkerDeathResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.Epoch != getWorkerDeathResult.GetRemountEpoch {
		parseT.Fatalf("expected coordinator epoch %d, got %d", getWorkerDeathResult.GetRemountEpoch, parseEntry.Epoch)
	}
}

// TestHandleHostRegionRepairRemountReissuesCleanMountBeforeUpdates verifies repair remount handshake is required before updates can continue.
func TestHandleHostRegionRepairRemountReissuesCleanMountBeforeUpdates(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	getWorkerDeathResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	if _, parseErr = buildHostRegionAdapter.HandleHostRegionUpdate(getWorkerDeathResult.GetVersionFloor + 1); parseErr == nil {
		parseT.Fatal("expected update dispatch to be blocked before repair remount handshake")
	}
	getRepairRemountResult, parseErr := buildHostRegionAdapter.HandleHostRegionRepairRemount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	})
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionRepairRemount returned error: %v", parseErr)
	}
	if !getRepairRemountResult.HasRemounted {
		parseT.Fatal("expected repair remount handshake to remount region")
	}
	if _, parseErr = buildHostRegionAdapter.HandleHostRegionUpdate(getWorkerDeathResult.GetVersionFloor + 1); parseErr != nil {
		parseT.Fatalf("expected update dispatch after repair remount handshake to succeed, got %v", parseErr)
	}
}

// TestHandleHostRegionRepairRemountKeepsLatestValidInputVersion verifies repair remount keeps the latest valid version floor.
func TestHandleHostRegionRepairRemountKeepsLatestValidInputVersion(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionUpdate(7); parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate(7) returned error: %v", parseErr)
	}
	getWorkerDeathResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	if _, parseErr = buildHostRegionAdapter.HandleHostRegionRepairRemount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}); parseErr != nil {
		parseT.Fatalf("HandleHostRegionRepairRemount returned error: %v", parseErr)
	}
	getPatchReadyOlderResult, parseErr := buildHostRegionAdapter.HandleHostRegionPatchReady(getWorkerDeathResult.GetVersionFloor - 1)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady(older) returned error: %v", parseErr)
	}
	if !getPatchReadyOlderResult.HasIgnored {
		parseT.Fatal("expected older-than-floor patch-ready result to be ignored after repair remount")
	}
	if getPatchReadyOlderResult.GetIgnoreReason != "stale-before-repair-floor" {
		parseT.Fatalf("expected stale-before-repair-floor reason, got %q", getPatchReadyOlderResult.GetIgnoreReason)
	}
}

// TestHandleHostRegionFallbackClearsOnlyAfterHealthyRepairRemount verifies fallback ownership remains active until healthy remount handshake.
func TestHandleHostRegionFallbackClearsOnlyAfterHealthyRepairRemount(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true); parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	if !buildHostRegionAdapter.GetHostRegionIsFallbackActive() {
		parseT.Fatal("expected fallback ownership to remain active during repair-pending period")
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionRepairRemount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}); parseErr != nil {
		parseT.Fatalf("HandleHostRegionRepairRemount returned error: %v", parseErr)
	}
	if buildHostRegionAdapter.GetHostRegionIsFallbackActive() {
		parseT.Fatal("expected fallback ownership to clear after healthy repair remount handshake")
	}
	if buildHostRegionAdapter.GetHostRegionRecoveryCoordinator().IsRegionLocalOwnership("region-1") {
		parseT.Fatal("expected recovery fallback ownership to clear after healthy repair remount handshake")
	}
	if buildHostRegionAdapter.GetHostRegionScheduler().HasSchedulerFallbackOwnership("region-1") {
		parseT.Fatal("expected scheduler fallback ownership to clear after healthy repair remount handshake")
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.IsFallback {
		parseT.Fatal("expected coordinator fallback state to clear after healthy repair remount handshake")
	}
}

// TestHandleHostRegionRepairRemountIncrementsCoordinatorRepairCounter verifies successful repair remounts increment coordinator repair counters.
func TestHandleHostRegionRepairRemountIncrementsCoordinatorRepairCounter(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if _, parseWorkerDeathErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true); parseWorkerDeathErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseWorkerDeathErr)
	}
	if _, parseRepairRemountErr := buildHostRegionAdapter.HandleHostRegionRepairRemount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}); parseRepairRemountErr != nil {
		parseT.Fatalf("HandleHostRegionRepairRemount returned error: %v", parseRepairRemountErr)
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.RepairRemountCount != 1 {
		parseT.Fatalf("expected repair-remount counter 1, got %d", parseEntry.RepairRemountCount)
	}
}
