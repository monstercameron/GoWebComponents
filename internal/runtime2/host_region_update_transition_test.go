package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionUpdateDispatchWithTransitionUsesDeferredPriority verifies transition updates dispatch through deferred priority.
func TestHandleHostRegionUpdateDispatchWithTransitionUsesDeferredPriority(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	getDispatchResult, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransition(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		true,
	)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransition returned error: %v", parseDispatchErr)
	}
	if getDispatchResult.GetDispatchPriority != runtime2.HostRegionDispatchPriorityDeferred {
		parseT.Fatalf("expected deferred transition dispatch priority, got %q", getDispatchResult.GetDispatchPriority)
	}
	if !getDispatchResult.HasDeferredQueued {
		parseT.Fatal("expected transition dispatch to queue deferred work")
	}
}

// TestHandleHostRegionUpdateDispatchWithTransitionUsesUrgentPriority verifies non-transition updates dispatch through urgent priority.
func TestHandleHostRegionUpdateDispatchWithTransitionUsesUrgentPriority(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	getDispatchResult, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransition(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		false,
	)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransition returned error: %v", parseDispatchErr)
	}
	if getDispatchResult.GetDispatchPriority != runtime2.HostRegionDispatchPriorityUrgent {
		parseT.Fatalf("expected urgent non-transition dispatch priority, got %q", getDispatchResult.GetDispatchPriority)
	}
	if !getDispatchResult.HasScheduled {
		parseT.Fatal("expected non-transition dispatch to schedule urgent worker update")
	}
}
