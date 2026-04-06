package runtime2

import "testing"

// TestCoordinatorReadHelpersCoversNilAndMountedPaths exercises the coordinator field readers on nil, missing, and mounted entries.
func TestCoordinatorReadHelpersCoversNilAndMountedPaths(t *testing.T) {
	var getNilCoordinator *Coordinator
	if getEpoch, getRendererID, getSourceIDs, ok := getNilCoordinator.GetEntrySnapshotFields("region-1"); ok || getEpoch != 0 || getRendererID != "" || getSourceIDs != nil {
		t.Fatalf("expected nil snapshot lookup to return zero values, got %d %q %+v %t", getEpoch, getRendererID, getSourceIDs, ok)
	}
	if isFallback, lastSnapshotVersion, lastDispatchedVersion, ok := getNilCoordinator.GetEntryDispatchValidation("region-1"); ok || isFallback || lastSnapshotVersion != 0 || lastDispatchedVersion != 0 {
		t.Fatalf("expected nil dispatch lookup to return zero values, got %t %d %d %t", isFallback, lastSnapshotVersion, lastDispatchedVersion, ok)
	}
	if getEpoch, getRendererID, getSourceIDs, isFallback, lastSnapshotVersion, lastDispatchedVersion, ok := getNilCoordinator.GetEntrySnapshotAndDispatchFields("region-1"); ok || getEpoch != 0 || getRendererID != "" || getSourceIDs != nil || isFallback || lastSnapshotVersion != 0 || lastDispatchedVersion != 0 {
		t.Fatalf("expected nil combined lookup to return zero values, got %d %q %+v %t %d %d %t", getEpoch, getRendererID, getSourceIDs, isFallback, lastSnapshotVersion, lastDispatchedVersion, ok)
	}

	getCoordinator := BuildCoordinator()
	if getEpoch, getRendererID, getSourceIDs, ok := getCoordinator.GetEntrySnapshotFields("region-1"); ok || getEpoch != 0 || getRendererID != "" || getSourceIDs != nil {
		t.Fatalf("expected missing snapshot lookup to return zero values, got %d %q %+v %t", getEpoch, getRendererID, getSourceIDs, ok)
	}
	if isFallback, lastSnapshotVersion, lastDispatchedVersion, ok := getCoordinator.GetEntryDispatchValidation("region-1"); ok || isFallback || lastSnapshotVersion != 0 || lastDispatchedVersion != 0 {
		t.Fatalf("expected missing dispatch lookup to return zero values, got %t %d %d %t", isFallback, lastSnapshotVersion, lastDispatchedVersion, ok)
	}

	if err := getCoordinator.MountRegion(CoordinatorEntry{
		RegionInstanceID: "region-1",
		RendererID:       "dashboard.hot-panel",
		SourceIDs:        []string{"status", "count", "status"},
		Epoch:            1,
	}); err != nil {
		t.Fatalf("MountRegion returned error: %v", err)
	}
	if _, err := getCoordinator.StoreRegionSnapshotDispatchState("region-1", 3, 4, []string{"status", "count", "status"}, true); err != nil {
		t.Fatalf("StoreRegionSnapshotDispatchState returned error: %v", err)
	}
	if err := getCoordinator.FallbackRegion("region-1"); err != nil {
		t.Fatalf("FallbackRegion returned error: %v", err)
	}

	getEpoch, getRendererID, getSourceIDs, ok := getCoordinator.GetEntrySnapshotFields("region-1")
	if !ok || getEpoch != 1 || getRendererID != "dashboard.hot-panel" || len(getSourceIDs) != 2 || getSourceIDs[0] != "count" || getSourceIDs[1] != "status" {
		t.Fatalf("unexpected snapshot fields: epoch=%d renderer=%q sourceIDs=%+v ok=%t", getEpoch, getRendererID, getSourceIDs, ok)
	}
	isFallback, lastSnapshotVersion, lastDispatchedVersion, ok := getCoordinator.GetEntryDispatchValidation("region-1")
	if !ok || !isFallback || lastSnapshotVersion != 3 || lastDispatchedVersion != 4 {
		t.Fatalf("unexpected dispatch validation fields: fallback=%t snapshot=%d dispatched=%d ok=%t", isFallback, lastSnapshotVersion, lastDispatchedVersion, ok)
	}
	getEpoch, getRendererID, getSourceIDs, isFallback, lastSnapshotVersion, lastDispatchedVersion, ok = getCoordinator.GetEntrySnapshotAndDispatchFields("region-1")
	if !ok || getEpoch != 1 || getRendererID != "dashboard.hot-panel" || len(getSourceIDs) != 2 || getSourceIDs[0] != "count" || getSourceIDs[1] != "status" || !isFallback || lastSnapshotVersion != 3 || lastDispatchedVersion != 4 {
		t.Fatalf("unexpected combined snapshot/dispatch fields: epoch=%d renderer=%q sourceIDs=%+v fallback=%t snapshot=%d dispatched=%d ok=%t", getEpoch, getRendererID, getSourceIDs, isFallback, lastSnapshotVersion, lastDispatchedVersion, ok)
	}
}

// TestCoordinatorMutationHelpersCoversPrivateMutationPaths exercises the private mutation helpers and store writeback branches.
func TestCoordinatorMutationHelpersCoversPrivateMutationPaths(t *testing.T) {
	var getNilCoordinator *Coordinator
	if _, err := getNilCoordinator.getMutableEntry("region-1"); err == nil {
		t.Fatal("expected nil coordinator mutable lookup to fail")
	}
	if err := getNilCoordinator.applyRegionSnapshotState("region-1", 1, nil, false); err == nil {
		t.Fatal("expected nil coordinator snapshot update to fail")
	}
	if err := getNilCoordinator.applyRegionDispatchedVersionTrusted("region-1", 1); err == nil {
		t.Fatal("expected nil coordinator dispatch update to fail")
	}
	if err := getNilCoordinator.applyRegionSnapshotDispatchState("region-1", 1, 1, nil, false); err == nil {
		t.Fatal("expected nil coordinator snapshot-dispatch update to fail")
	}
	if err := getNilCoordinator.storeMutableEntry(CoordinatorEntry{RegionInstanceID: "region-1"}); err == nil {
		t.Fatal("expected nil coordinator store writeback to fail")
	}

	getCoordinator := BuildCoordinator()
	if _, err := getCoordinator.getMutableEntry("region-1"); err == nil {
		t.Fatal("expected missing mutable entry lookup to fail")
	}
	if err := getCoordinator.applyRegionSnapshotState("region-1", 0, nil, false); err == nil {
		t.Fatal("expected zero snapshot version to fail")
	}
	if err := getCoordinator.applyRegionDispatchedVersionTrusted("region-1", 1); err == nil {
		t.Fatal("expected missing trusted dispatch update to fail")
	}
	if err := getCoordinator.applyRegionSnapshotDispatchState("region-1", 1, 0, nil, false); err == nil {
		t.Fatal("expected zero dispatched version to fail")
	}
	if err := getCoordinator.applyRegionSnapshotDispatchState("region-1", 1, 1, nil, false); err == nil {
		t.Fatal("expected missing mounted entry to fail")
	}

	if err := getCoordinator.MountRegion(CoordinatorEntry{
		RegionInstanceID: "region-1",
		RendererID:       "dashboard.hot-panel",
		Epoch:            1,
	}); err != nil {
		t.Fatalf("MountRegion returned error: %v", err)
	}

	getEntry, err := getCoordinator.getMutableEntry("region-1")
	if err != nil {
		t.Fatalf("getMutableEntry returned error: %v", err)
	}
	if getEntry.RegionInstanceID != "region-1" {
		t.Fatalf("expected mutable entry for region-1, got %q", getEntry.RegionInstanceID)
	}

	if err := getCoordinator.applyRegionSnapshotState("region-1", 2, []string{"status", "count", "status"}, true); err != nil {
		t.Fatalf("applyRegionSnapshotState returned error: %v", err)
	}
	if err := getCoordinator.applyRegionDispatchedVersionTrusted("region-1", 3); err != nil {
		t.Fatalf("applyRegionDispatchedVersionTrusted returned error: %v", err)
	}
	if err := getCoordinator.applyRegionSnapshotDispatchState("region-1", 4, 5, []string{"count", "status", "count"}, true); err != nil {
		t.Fatalf("applyRegionSnapshotDispatchState returned error: %v", err)
	}

	getUpdatedEntry, err := getCoordinator.getMutableEntry("region-1")
	if err != nil {
		t.Fatalf("getMutableEntry after updates returned error: %v", err)
	}
	if getUpdatedEntry.LastSnapshotVersion != 4 || getUpdatedEntry.LastDispatchedVersion != 5 {
		t.Fatalf("unexpected updated versions: snapshot=%d dispatched=%d", getUpdatedEntry.LastSnapshotVersion, getUpdatedEntry.LastDispatchedVersion)
	}
	if len(getUpdatedEntry.SourceIDs) != 2 || getUpdatedEntry.SourceIDs[0] != "count" || getUpdatedEntry.SourceIDs[1] != "status" {
		t.Fatalf("expected canonical source IDs [count status], got %+v", getUpdatedEntry.SourceIDs)
	}

	getUpdatedEntry.AssignedWorkerShard = "worker-a"
	if err := getCoordinator.storeMutableEntry(getUpdatedEntry); err != nil {
		t.Fatalf("storeMutableEntry update returned error: %v", err)
	}
	if getStoredEntry, ok := getCoordinator.GetEntry("region-1"); !ok || getStoredEntry.AssignedWorkerShard != "worker-a" {
		t.Fatalf("expected stored entry to preserve updated shard, got %+v ok=%t", getStoredEntry, ok)
	}

	if err := getCoordinator.storeMutableEntry(CoordinatorEntry{
		RegionInstanceID: "region-2",
		RendererID:       "dashboard.hot-panel",
		Epoch:            1,
	}); err != nil {
		t.Fatalf("storeMutableEntry insert returned error: %v", err)
	}
	if getInsertedEntry, ok := getCoordinator.GetEntry("region-2"); !ok || getInsertedEntry.RegionInstanceID != "region-2" {
		t.Fatalf("expected inserted entry for region-2, got %+v ok=%t", getInsertedEntry, ok)
	}

	if err := getCoordinator.FallbackRegion("region-1"); err != nil {
		t.Fatalf("FallbackRegion returned error: %v", err)
	}
	if err := getCoordinator.applyRegionDispatchedVersionTrusted("region-1", 6); err == nil {
		t.Fatal("expected dispatched update to fail while region is in fallback mode")
	}
	if err := getCoordinator.applyRegionSnapshotDispatchState("region-1", 5, 6, nil, false); err == nil {
		t.Fatal("expected snapshot-dispatch update to fail while region is in fallback mode")
	}
}
