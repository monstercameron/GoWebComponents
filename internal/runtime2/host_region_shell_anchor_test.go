package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionShellMissingAnchorDetectionDetectsMissingAnchor verifies missing hydrated shell anchors are detectable before attach.
func TestHandleHostRegionShellMissingAnchorDetectionDetectsMissingAnchor(parseT *testing.T) {
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
	parseAnchorCheckResult, parseAnchorCheckErr := parseHostRegionAdapter.HandleHostRegionShellMissingAnchorDetection()
	if parseAnchorCheckErr != nil {
		parseT.Fatalf("HandleHostRegionShellMissingAnchorDetection(missing) returned error: %v", parseAnchorCheckErr)
	}
	if !parseAnchorCheckResult.HasMissingAnchor {
		parseT.Fatal("expected shell missing-anchor detection to report missing=true before anchor registration")
	}
}

// TestHandleHostRegionShellMissingAnchorDetectionAcceptsRegisteredAnchor verifies registered hydrated shell anchors clear missing-anchor state.
func TestHandleHostRegionShellMissingAnchorDetectionAcceptsRegisteredAnchor(parseT *testing.T) {
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
	if parseAnchorErr := parseHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor(valid) returned error: %v", parseAnchorErr)
	}
	parseAnchorCheckResult, parseAnchorCheckErr := parseHostRegionAdapter.HandleHostRegionShellMissingAnchorDetection()
	if parseAnchorCheckErr != nil {
		parseT.Fatalf("HandleHostRegionShellMissingAnchorDetection(registered) returned error: %v", parseAnchorCheckErr)
	}
	if parseAnchorCheckResult.HasMissingAnchor {
		parseT.Fatal("expected shell missing-anchor detection to report missing=false after anchor registration")
	}
}
