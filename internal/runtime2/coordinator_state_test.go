package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestMountRegionCreatesCoordinatorEntry verifies mount creates one live entry.
func TestMountRegionCreatesCoordinatorEntry(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID:    runtime2.RegionInstanceID("region-1"),
		RendererID:          runtime2.RendererID("dashboard.hot-panel"),
		Epoch:               1,
		AssignedWorkerShard: "worker-a",
	})
	if parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	parseEntry, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.CurrentState != runtime2.CoordinatorStateMounted {
		parseT.Fatalf("expected mounted state, got %q", parseEntry.CurrentState)
	}
}

// TestDisposeRegionRemovesCoordinatorEntry verifies dispose removes the live entry.
func TestDisposeRegionRemovesCoordinatorEntry(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.DisposeRegion(runtime2.RegionInstanceID("region-1")); parseErr != nil {
		parseT.Fatalf("DisposeRegion returned error: %v", parseErr)
	}
	if _, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1")); parseHasEntry {
		parseT.Fatal("expected disposed coordinator entry to be removed")
	}
}

// TestMountRegionRejectsDuplicateActiveEntry verifies duplicate mount attempts fail.
func TestMountRegionRejectsDuplicateActiveEntry(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	parseEntry := runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}
	if parseErr := parseCoordinator.MountRegion(parseEntry); parseErr != nil {
		parseT.Fatalf("first MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.MountRegion(parseEntry); parseErr == nil {
		parseT.Fatal("expected duplicate MountRegion to fail")
	}
}

// TestCoordinatorValidTransitionSequenceSucceeds verifies one normal state sequence succeeds.
func TestCoordinatorValidTransitionSequenceSucceeds(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.UpdateRegion(runtime2.RegionInstanceID("region-1"), 2); parseErr != nil {
		parseT.Fatalf("UpdateRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.CancelRegion(runtime2.RegionInstanceID("region-1")); parseErr != nil {
		parseT.Fatalf("CancelRegion returned error: %v", parseErr)
	}
	parseEntry, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected coordinator entry after cancel")
	}
	if parseEntry.CurrentState != runtime2.CoordinatorStateCanceled {
		parseT.Fatalf("expected canceled state, got %q", parseEntry.CurrentState)
	}
}

// TestCoordinatorInvalidTransitionSequenceFails verifies invalid transitions fail clearly.
func TestCoordinatorInvalidTransitionSequenceFails(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.UpdateRegion(runtime2.RegionInstanceID("region-1"), 1); parseErr == nil {
		parseT.Fatal("expected update before mount to fail")
	}
}

// TestRestartRegionBumpsEpoch verifies restart advances the stored epoch.
func TestRestartRegionBumpsEpoch(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.RestartRegion(runtime2.RegionInstanceID("region-1"), 2); parseErr != nil {
		parseT.Fatalf("RestartRegion returned error: %v", parseErr)
	}
	parseEntry, _ := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if parseEntry.Epoch != 2 {
		parseT.Fatalf("expected epoch 2 after restart, got %d", parseEntry.Epoch)
	}
}

// TestUpdateRegionTracksLastDispatchedVersion verifies dispatched version tracking updates monotonically.
func TestUpdateRegionTracksLastDispatchedVersion(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.UpdateRegion(runtime2.RegionInstanceID("region-1"), 1); parseErr != nil {
		parseT.Fatalf("UpdateRegion(1) returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.UpdateRegion(runtime2.RegionInstanceID("region-1"), 2); parseErr != nil {
		parseT.Fatalf("UpdateRegion(2) returned error: %v", parseErr)
	}
	parseEntry, _ := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if parseEntry.LastDispatchedVersion != 2 {
		parseT.Fatalf("expected dispatched version 2, got %d", parseEntry.LastDispatchedVersion)
	}
}

// TestUpdateRegionRejectsStaleDispatchedVersion verifies dispatched version tracking does not move backward.
func TestUpdateRegionRejectsStaleDispatchedVersion(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.UpdateRegion(runtime2.RegionInstanceID("region-1"), 2); parseErr != nil {
		parseT.Fatalf("UpdateRegion(2) returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.UpdateRegion(runtime2.RegionInstanceID("region-1"), 1); parseErr == nil {
		parseT.Fatal("expected stale dispatched version to fail")
	}
}

// TestCommitRegionTracksLastCommittedVersion verifies committed version tracking updates monotonically.
func TestCommitRegionTracksLastCommittedVersion(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.CommitRegion(runtime2.RegionInstanceID("region-1"), 1); parseErr != nil {
		parseT.Fatalf("CommitRegion(1) returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.CommitRegion(runtime2.RegionInstanceID("region-1"), 2); parseErr != nil {
		parseT.Fatalf("CommitRegion(2) returned error: %v", parseErr)
	}
	parseEntry, _ := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if parseEntry.LastCommittedVersion != 2 {
		parseT.Fatalf("expected committed version 2, got %d", parseEntry.LastCommittedVersion)
	}
}

// TestCommitRegionRejectsOlderCommittedVersion verifies committed versions do not move backward.
func TestCommitRegionRejectsOlderCommittedVersion(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.CommitRegion(runtime2.RegionInstanceID("region-1"), 2); parseErr != nil {
		parseT.Fatalf("CommitRegion(2) returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.CommitRegion(runtime2.RegionInstanceID("region-1"), 1); parseErr == nil {
		parseT.Fatal("expected older committed version to fail")
	}
}

// TestFallbackRegionMarksFallbackState verifies fallback mode becomes explicit.
func TestFallbackRegionMarksFallbackState(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.FallbackRegion(runtime2.RegionInstanceID("region-1")); parseErr != nil {
		parseT.Fatalf("FallbackRegion returned error: %v", parseErr)
	}
	parseEntry, _ := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseEntry.IsFallback || parseEntry.CurrentState != runtime2.CoordinatorStateFallback {
		parseT.Fatalf("expected fallback state, got %+v", parseEntry)
	}
}

// TestFallbackRegionBlocksLaterCommit verifies fallback suppresses later worker commits.
func TestFallbackRegionBlocksLaterCommit(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.FallbackRegion(runtime2.RegionInstanceID("region-1")); parseErr != nil {
		parseT.Fatalf("FallbackRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.CommitRegion(runtime2.RegionInstanceID("region-1"), 1); parseErr == nil {
		parseT.Fatal("expected commit during fallback to fail")
	}
}
