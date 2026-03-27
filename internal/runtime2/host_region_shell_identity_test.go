package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionShellIdentityMismatchDetectionDetectsRegionMismatch verifies shell identity checks detect region-ID mismatches.
func TestHandleHostRegionShellIdentityMismatchDetectionDetectsRegionMismatch(parseT *testing.T) {
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
	parseMismatchResult, parseMismatchErr := parseHostRegionAdapter.HandleHostRegionShellIdentityMismatchDetection(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: runtime2.RegionInstanceID("region-2"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
	})
	if parseMismatchErr != nil {
		parseT.Fatalf("HandleHostRegionShellIdentityMismatchDetection(region mismatch) returned error: %v", parseMismatchErr)
	}
	if !parseMismatchResult.HasMismatch || !parseMismatchResult.HasRegionIDMismatch || parseMismatchResult.HasRendererIDMismatch {
		parseT.Fatalf("region mismatch detection = %+v, want mismatch=true region=true renderer=false", parseMismatchResult)
	}
}

// TestHandleHostRegionShellIdentityMismatchDetectionDetectsRendererMismatch verifies shell identity checks detect renderer-ID mismatches.
func TestHandleHostRegionShellIdentityMismatchDetectionDetectsRendererMismatch(parseT *testing.T) {
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
	parseMismatchResult, parseMismatchErr := parseHostRegionAdapter.HandleHostRegionShellIdentityMismatchDetection(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.cold-panel"),
	})
	if parseMismatchErr != nil {
		parseT.Fatalf("HandleHostRegionShellIdentityMismatchDetection(renderer mismatch) returned error: %v", parseMismatchErr)
	}
	if !parseMismatchResult.HasMismatch || parseMismatchResult.HasRegionIDMismatch || !parseMismatchResult.HasRendererIDMismatch {
		parseT.Fatalf("renderer mismatch detection = %+v, want mismatch=true region=false renderer=true", parseMismatchResult)
	}
}

// TestHandleHostRegionShellIdentityMismatchDetectionAcceptsMatchingIdentity verifies matching shell identity payloads do not trigger mismatch state.
func TestHandleHostRegionShellIdentityMismatchDetectionAcceptsMatchingIdentity(parseT *testing.T) {
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
	parseMismatchResult, parseMismatchErr := parseHostRegionAdapter.HandleHostRegionShellIdentityMismatchDetection(runtime2.SSRShellMarker{
		Version:          runtime2.SSRShellMarkerVersionV1,
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
	})
	if parseMismatchErr != nil {
		parseT.Fatalf("HandleHostRegionShellIdentityMismatchDetection(matching) returned error: %v", parseMismatchErr)
	}
	if parseMismatchResult.HasMismatch || parseMismatchResult.HasRegionIDMismatch || parseMismatchResult.HasRendererIDMismatch {
		parseT.Fatalf("matching identity detection = %+v, want all mismatch flags false", parseMismatchResult)
	}
}
