package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
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

// TestSetRegionAttachedStoresAttachedState verifies coordinator entries track attached-state transitions.
func TestSetRegionAttachedStoresAttachedState(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.SetRegionAttached(runtime2.RegionInstanceID("region-1"), true); parseErr != nil {
		parseT.Fatalf("SetRegionAttached(true) returned error: %v", parseErr)
	}
	parseEntry, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if !parseEntry.IsAttached {
		parseT.Fatal("expected coordinator attached state true after SetRegionAttached(true)")
	}
	if parseErr := parseCoordinator.SetRegionAttached(runtime2.RegionInstanceID("region-1"), false); parseErr != nil {
		parseT.Fatalf("SetRegionAttached(false) returned error: %v", parseErr)
	}
	parseEntry, _ = parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if parseEntry.IsAttached {
		parseT.Fatal("expected coordinator attached state false after SetRegionAttached(false)")
	}
}

// TestSetRegionSourceIDsStoresCanonicalSourceIDs verifies coordinator entries track canonical declared source IDs.
func TestSetRegionSourceIDsStoresCanonicalSourceIDs(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.SetRegionSourceIDs(runtime2.RegionInstanceID("region-1"), []string{"status", "count", "status"}); parseErr != nil {
		parseT.Fatalf("SetRegionSourceIDs returned error: %v", parseErr)
	}
	parseEntry, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if len(parseEntry.SourceIDs) != 2 || parseEntry.SourceIDs[0] != "count" || parseEntry.SourceIDs[1] != "status" {
		parseT.Fatalf("expected canonical source IDs [count status], got %+v", parseEntry.SourceIDs)
	}
}

// TestSetRegionLastSnapshotVersionTracksMonotonicVersion verifies coordinator snapshot version tracking does not move backward.
func TestSetRegionLastSnapshotVersionTracksMonotonicVersion(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.SetRegionLastSnapshotVersion(runtime2.RegionInstanceID("region-1"), 3); parseErr != nil {
		parseT.Fatalf("SetRegionLastSnapshotVersion(3) returned error: %v", parseErr)
	}
	if parseErr := parseCoordinator.SetRegionLastSnapshotVersion(runtime2.RegionInstanceID("region-1"), 2); parseErr == nil {
		parseT.Fatal("expected older snapshot version to fail")
	}
	parseEntry, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.LastSnapshotVersion != 3 {
		parseT.Fatalf("expected last snapshot version 3, got %d", parseEntry.LastSnapshotVersion)
	}
}

// TestStoreRegionSnapshotAndDispatchedVersionTracksBothVersions verifies snapshot and dispatched versions update in one coordinator transaction.
func TestStoreRegionSnapshotAndDispatchedVersionTracksBothVersions(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	getEntry, parseErr := parseCoordinator.StoreRegionSnapshotAndDispatchedVersion(runtime2.RegionInstanceID("region-1"), 3, 3)
	if parseErr != nil {
		parseT.Fatalf("StoreRegionSnapshotAndDispatchedVersion returned error: %v", parseErr)
	}
	if getEntry.LastSnapshotVersion != 3 {
		parseT.Fatalf("expected last snapshot version 3, got %d", getEntry.LastSnapshotVersion)
	}
	if getEntry.LastDispatchedVersion != 3 {
		parseT.Fatalf("expected last dispatched version 3, got %d", getEntry.LastDispatchedVersion)
	}
	if getEntry.CurrentState != runtime2.CoordinatorStateActive {
		parseT.Fatalf("expected active coordinator state, got %q", getEntry.CurrentState)
	}
	if _, parseErr = parseCoordinator.StoreRegionSnapshotAndDispatchedVersion(runtime2.RegionInstanceID("region-1"), 2, 2); parseErr == nil {
		parseT.Fatal("expected stale snapshot or dispatched version to fail")
	}
}

// TestStoreRegionSnapshotDispatchStateStoresCanonicalSourceIDs verifies one coordinator transaction can update snapshot, dispatch, and source IDs together.
func TestStoreRegionSnapshotDispatchStateStoresCanonicalSourceIDs(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	getEntry, parseErr := parseCoordinator.StoreRegionSnapshotDispatchState(
		runtime2.RegionInstanceID("region-1"),
		7,
		7,
		[]string{"status", "count", "status"},
		true,
	)
	if parseErr != nil {
		parseT.Fatalf("StoreRegionSnapshotDispatchState returned error: %v", parseErr)
	}
	if getEntry.LastSnapshotVersion != 7 {
		parseT.Fatalf("expected last snapshot version 7, got %d", getEntry.LastSnapshotVersion)
	}
	if getEntry.LastDispatchedVersion != 7 {
		parseT.Fatalf("expected last dispatched version 7, got %d", getEntry.LastDispatchedVersion)
	}
	if len(getEntry.SourceIDs) != 2 || getEntry.SourceIDs[0] != "count" || getEntry.SourceIDs[1] != "status" {
		parseT.Fatalf("expected canonical source IDs [count status], got %+v", getEntry.SourceIDs)
	}
}

// TestIncrementRegionIgnoredStaleDiagnosticCountTracksStaleDiagnosticDrops verifies stale-diagnostic ignore counters increment monotonically.
func TestIncrementRegionIgnoredStaleDiagnosticCountTracksStaleDiagnosticDrops(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if _, parseErr := parseCoordinator.IncrementRegionIgnoredStaleDiagnosticCount(runtime2.RegionInstanceID("region-1")); parseErr != nil {
		parseT.Fatalf("IncrementRegionIgnoredStaleDiagnosticCount(1) returned error: %v", parseErr)
	}
	getCount, parseErr := parseCoordinator.IncrementRegionIgnoredStaleDiagnosticCount(runtime2.RegionInstanceID("region-1"))
	if parseErr != nil {
		parseT.Fatalf("IncrementRegionIgnoredStaleDiagnosticCount(2) returned error: %v", parseErr)
	}
	if getCount != 2 {
		parseT.Fatalf("expected stale diagnostic counter 2, got %d", getCount)
	}
	parseEntry, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.IgnoredStaleDiagnosticCount != 2 {
		parseT.Fatalf("expected stored stale diagnostic counter 2, got %d", parseEntry.IgnoredStaleDiagnosticCount)
	}
}

// TestIncrementRegionRepairRemountCountTracksSuccessfulRepairs verifies repair-remount counters increment for each successful remount.
func TestIncrementRegionRepairRemountCountTracksSuccessfulRepairs(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	if _, parseErr := parseCoordinator.IncrementRegionRepairRemountCount(runtime2.RegionInstanceID("region-1")); parseErr != nil {
		parseT.Fatalf("IncrementRegionRepairRemountCount(1) returned error: %v", parseErr)
	}
	getCount, parseErr := parseCoordinator.IncrementRegionRepairRemountCount(runtime2.RegionInstanceID("region-1"))
	if parseErr != nil {
		parseT.Fatalf("IncrementRegionRepairRemountCount(2) returned error: %v", parseErr)
	}
	if getCount != 2 {
		parseT.Fatalf("expected repair-remount counter 2, got %d", getCount)
	}
	parseEntry, parseHasEntry := parseCoordinator.GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.RepairRemountCount != 2 {
		parseT.Fatalf("expected stored repair-remount counter 2, got %d", parseEntry.RepairRemountCount)
	}
}

// TestSetRegionSnapshotStateUpdatesSnapshotAndSources verifies one coordinator transaction updates snapshot version and canonical source IDs together.
func TestSetRegionSnapshotStateUpdatesSnapshotAndSources(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	parseEntry, parseErr := parseCoordinator.SetRegionSnapshotState(
		runtime2.RegionInstanceID("region-1"),
		4,
		[]string{"status", "count", "status"},
		true,
	)
	if parseErr != nil {
		parseT.Fatalf("SetRegionSnapshotState returned error: %v", parseErr)
	}
	if parseEntry.LastSnapshotVersion != 4 {
		parseT.Fatalf("expected snapshot version 4, got %d", parseEntry.LastSnapshotVersion)
	}
	if len(parseEntry.SourceIDs) != 2 || parseEntry.SourceIDs[0] != "count" || parseEntry.SourceIDs[1] != "status" {
		parseT.Fatalf("expected canonical source IDs [count status], got %+v", parseEntry.SourceIDs)
	}
}

// TestUpdateRegionAndGetEntryTracksLastDispatchedVersion verifies one coordinator update transaction returns the updated dispatched version.
func TestUpdateRegionAndGetEntryTracksLastDispatchedVersion(parseT *testing.T) {
	parseCoordinator := runtime2.BuildCoordinator()
	if parseErr := parseCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
	}); parseErr != nil {
		parseT.Fatalf("MountRegion returned error: %v", parseErr)
	}
	parseEntry, parseErr := parseCoordinator.UpdateRegionAndGetEntry(runtime2.RegionInstanceID("region-1"), 2)
	if parseErr != nil {
		parseT.Fatalf("UpdateRegionAndGetEntry returned error: %v", parseErr)
	}
	if parseEntry.LastDispatchedVersion != 2 {
		parseT.Fatalf("expected dispatched version 2, got %d", parseEntry.LastDispatchedVersion)
	}
}
