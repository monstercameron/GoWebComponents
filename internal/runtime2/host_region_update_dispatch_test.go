package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionUpdateDispatchShortCircuitsNoChange verifies host update dispatch skips scheduler work when snapshot fingerprint is unchanged.
func TestHandleHostRegionUpdateDispatchShortCircuitsNoChange(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{"count": 5}, map[string]uint64{"count": 3}, nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	getFirstDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		2,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch(first) returned error: %v", parseErr)
	}
	if !getFirstDispatchResult.HasScheduled {
		parseT.Fatal("expected first dispatch to schedule worker update")
	}
	if getFirstDispatchResult.HasNoChange {
		parseT.Fatal("expected first dispatch to be treated as changed")
	}
	getQueueDepthAfterFirstDispatch := buildHostRegionAdapter.GetHostRegionScheduler().GetSchedulerQueueDepth()

	getSecondDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch(second) returned error: %v", parseErr)
	}
	if getSecondDispatchResult.HasScheduled {
		parseT.Fatal("expected unchanged snapshot to short-circuit before scheduler dispatch")
	}
	if !getSecondDispatchResult.HasNoChange {
		parseT.Fatal("expected unchanged snapshot to be reported as no-change")
	}
	if buildHostRegionAdapter.GetHostRegionScheduler().GetSchedulerQueueDepth() != getQueueDepthAfterFirstDispatch {
		parseT.Fatalf(
			"expected scheduler queue depth to stay %d after no-change short-circuit, got %d",
			getQueueDepthAfterFirstDispatch,
			buildHostRegionAdapter.GetHostRegionScheduler().GetSchedulerQueueDepth(),
		)
	}
}

// TestHandleHostRegionUpdateDispatchNoChangeDoesNotAdvanceDispatchedVersion verifies no-change short-circuits do not advance coordinator dispatched-version state.
func TestHandleHostRegionUpdateDispatchNoChangeDoesNotAdvanceDispatchedVersion(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{"count": 5}, map[string]uint64{"count": 3}, nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		2,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch(2) returned error: %v", parseErr)
	}
	parseEntryAfterFirstDispatch, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntryAfterFirstDispatch.LastDispatchedVersion != 2 {
		parseT.Fatalf("expected first dispatched version 2, got %d", parseEntryAfterFirstDispatch.LastDispatchedVersion)
	}

	getNoChangeDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch(3) returned error: %v", parseErr)
	}
	if !getNoChangeDispatchResult.HasNoChange {
		parseT.Fatal("expected second dispatch to be no-change")
	}
	parseEntryAfterNoChangeDispatch, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry after no-change dispatch")
	}
	if parseEntryAfterNoChangeDispatch.LastDispatchedVersion != 2 {
		parseT.Fatalf("expected dispatched version to remain 2 after no-change dispatch, got %d", parseEntryAfterNoChangeDispatch.LastDispatchedVersion)
	}
}

// TestHandleHostRegionUpdateDispatchChangedPropsSchedulesUpdate verifies changed props do not short-circuit as no-change.
func TestHandleHostRegionUpdateDispatchChangedPropsSchedulesUpdate(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	getFirstDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"tick": 1},
		},
		2,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch(first changed) returned error: %v", parseErr)
	}
	if !getFirstDispatchResult.HasScheduled || getFirstDispatchResult.HasNoChange {
		parseT.Fatalf("expected first changed dispatch to schedule without no-change, got %+v", getFirstDispatchResult)
	}
	getSecondDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"tick": 2},
		},
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch(second changed) returned error: %v", parseErr)
	}
	if !getSecondDispatchResult.HasScheduled {
		parseT.Fatalf("expected changed dispatch to schedule worker update, got %+v", getSecondDispatchResult)
	}
	if getSecondDispatchResult.HasNoChange {
		parseT.Fatalf("expected changed dispatch not to short-circuit as no-change, got %+v", getSecondDispatchResult)
	}
}
