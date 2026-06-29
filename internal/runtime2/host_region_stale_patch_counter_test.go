package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionPatchReadyStaleBeforeRepairFloorIncrementsDroppedCounter verifies stale-before-repair-floor patch-ready drops increment coordinator stale-drop counters.
func TestHandleHostRegionPatchReadyStaleBeforeRepairFloorIncrementsDroppedCounter(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if _, parseUpdateErr := buildHostRegionAdapter.HandleHostRegionUpdate(7); parseUpdateErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate(7) returned error: %v", parseUpdateErr)
	}
	getWorkerDeathResult, parseWorkerDeathErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseWorkerDeathErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseWorkerDeathErr)
	}
	if _, parseRepairErr := buildHostRegionAdapter.HandleHostRegionRepairRemount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}); parseRepairErr != nil {
		parseT.Fatalf("HandleHostRegionRepairRemount returned error: %v", parseRepairErr)
	}
	parseStalePatchReadyResult, parseStalePatchReadyErr := buildHostRegionAdapter.HandleHostRegionPatchReady(getWorkerDeathResult.GetVersionFloor - 1)
	if parseStalePatchReadyErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady(stale) returned error: %v", parseStalePatchReadyErr)
	}
	if !parseStalePatchReadyResult.HasIgnored || parseStalePatchReadyResult.GetIgnoreReason != "stale-before-repair-floor" {
		parseT.Fatalf("expected stale-before-repair-floor ignore result, got %+v", parseStalePatchReadyResult)
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.DroppedStalePatchCount != 1 {
		parseT.Fatalf("expected dropped stale patch counter 1, got %d", parseEntry.DroppedStalePatchCount)
	}
}

// TestHandleHostRegionWorkerOutputOlderVersionIncrementsDroppedCounter verifies older-or-duplicate worker outputs increment coordinator stale-drop counters.
func TestHandleHostRegionWorkerOutputOlderVersionIncrementsDroppedCounter(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if _, parseWorkerOutputErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(1); parseWorkerOutputErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput(1 first) returned error: %v", parseWorkerOutputErr)
	}
	parseWorkerOutputResult, parseWorkerOutputErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(1)
	if parseWorkerOutputErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput(1 duplicate) returned error: %v", parseWorkerOutputErr)
	}
	if !parseWorkerOutputResult.HasIgnored {
		parseT.Fatal("expected duplicate worker output to be ignored as stale")
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.DroppedStalePatchCount != 1 {
		parseT.Fatalf("expected dropped stale patch counter 1, got %d", parseEntry.DroppedStalePatchCount)
	}
}
