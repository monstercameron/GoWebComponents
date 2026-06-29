package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionShellMismatchFallbackEntersLocalFallback verifies hydration mismatch handling enters local fallback ownership.
func TestHandleHostRegionShellMismatchFallbackEntersLocalFallback(parseT *testing.T) {
	parseHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount(valid) returned error: %v", parseMountErr)
	}
	parseFallbackResult, parseFallbackErr := parseHostRegionAdapter.HandleHostRegionShellMismatchFallback(2)
	if parseFallbackErr != nil {
		parseT.Fatalf("HandleHostRegionShellMismatchFallback(valid) returned error: %v", parseFallbackErr)
	}
	if !parseFallbackResult.HasFallbackEntered {
		parseT.Fatal("expected hydration mismatch fallback to enter local fallback ownership")
	}
	if !parseHostRegionAdapter.GetHostRegionIsFallbackActive() {
		parseT.Fatal("expected GetHostRegionIsFallbackActive() to report true after hydration mismatch fallback")
	}
}

// TestHandleHostRegionShellMismatchFallbackSuppressesPatchReady verifies hydration mismatch fallback suppresses later worker output.
func TestHandleHostRegionShellMismatchFallbackSuppressesPatchReady(parseT *testing.T) {
	parseHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount(valid) returned error: %v", parseMountErr)
	}
	if _, parseFallbackErr := parseHostRegionAdapter.HandleHostRegionShellMismatchFallback(2); parseFallbackErr != nil {
		parseT.Fatalf("HandleHostRegionShellMismatchFallback(valid) returned error: %v", parseFallbackErr)
	}
	parsePatchReadyResult, parsePatchReadyErr := parseHostRegionAdapter.HandleHostRegionPatchReady(3)
	if parsePatchReadyErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady(after mismatch fallback) returned error: %v", parsePatchReadyErr)
	}
	if !parsePatchReadyResult.HasIgnored {
		parseT.Fatal("expected patch-ready output to be ignored after hydration mismatch fallback")
	}
}

// TestHandleHostRegionShellMismatchFallbackRequiresFreshEpochForRecovery verifies hydration mismatch fallback allocates a fresh remount epoch for later reattach.
func TestHandleHostRegionShellMismatchFallbackRequiresFreshEpochForRecovery(parseT *testing.T) {
	parseHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount(valid) returned error: %v", parseMountErr)
	}
	parseFallbackResult, parseFallbackErr := parseHostRegionAdapter.HandleHostRegionShellMismatchFallback(2)
	if parseFallbackErr != nil {
		parseT.Fatalf("HandleHostRegionShellMismatchFallback(valid) returned error: %v", parseFallbackErr)
	}
	if parseFallbackResult.GetRemountEpoch <= 1 {
		parseT.Fatalf("expected fresh remount epoch > 1 after mismatch fallback, got %d", parseFallbackResult.GetRemountEpoch)
	}
	getRepairEpoch := parseHostRegionAdapter.GetHostRegionRepairRemountEpoch()
	if getRepairEpoch <= 1 {
		parseT.Fatalf("GetHostRegionRepairRemountEpoch() = %d, want > 1", getRepairEpoch)
	}
	getRepairResult, parseRepairErr := parseHostRegionAdapter.HandleHostRegionRepairRemount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
	)
	if parseRepairErr != nil {
		parseT.Fatalf("HandleHostRegionRepairRemount(valid) returned error: %v", parseRepairErr)
	}
	if getRepairResult.GetRemountEpoch < getRepairEpoch {
		parseT.Fatalf("expected repair remount epoch >= pending repair epoch floor, got remount=%d floor=%d", getRepairResult.GetRemountEpoch, getRepairEpoch)
	}
}
