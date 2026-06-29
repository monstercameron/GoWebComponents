package runtime2_test

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionUpdateSnapshotCapturesPropsAndSources verifies host-side snapshot capture includes props and declared sources for one update.
func TestHandleHostRegionUpdateSnapshotCapturesPropsAndSources(parseT *testing.T) {
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
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": 5,
				},
				map[string]uint64{
					"count": 9,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	getSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		7,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot returned error: %v", parseErr)
	}
	if getSnapshotEnvelope.Epoch != 3 {
		parseT.Fatalf("expected snapshot epoch 3, got %d", getSnapshotEnvelope.Epoch)
	}
	if getSnapshotEnvelope.InputVersion != 7 {
		parseT.Fatalf("expected snapshot input version 7, got %d", getSnapshotEnvelope.InputVersion)
	}
	if getSnapshotEnvelope.SourceVersion != 9 {
		parseT.Fatalf("expected snapshot source version 9, got %d", getSnapshotEnvelope.SourceVersion)
	}
	if getSnapshotEnvelope.Sources["count"] != 5 {
		parseT.Fatalf("expected source count=5, got %+v", getSnapshotEnvelope.Sources)
	}
	getProps, hasProps := getSnapshotEnvelope.Props.(map[string]any)
	if !hasProps || getProps["title"] != "Orders" {
		parseT.Fatalf("expected props map with title=Orders, got %+v", getSnapshotEnvelope.Props)
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.LastSnapshotVersion != 7 {
		parseT.Fatalf("expected coordinator last snapshot version 7, got %d", parseEntry.LastSnapshotVersion)
	}
	if len(parseEntry.SourceIDs) != 1 || parseEntry.SourceIDs[0] != "count" {
		parseT.Fatalf("expected coordinator source IDs [count], got %+v", parseEntry.SourceIDs)
	}
}

// TestHandleHostRegionUpdateSnapshotRejectsNotMountedRegion verifies snapshot capture fails when no mounted coordinator entry exists.
func TestHandleHostRegionUpdateSnapshotRejectsNotMountedRegion(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseErr == nil {
		parseT.Fatal("expected snapshot capture without mount to fail")
	}
}

// TestHandleHostRegionUpdateSnapshotReusesSourceMapForStableVersionTuple verifies snapshot capture reuses cached source map state when epoch and source-version tuple stay unchanged.
func TestHandleHostRegionUpdateSnapshotReusesSourceMapForStableVersionTuple(parseT *testing.T) {
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
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": 5,
				},
				map[string]uint64{
					"count": 9,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	getFirstSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		7,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(first) returned error: %v", parseErr)
	}
	getSecondSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		8,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(second) returned error: %v", parseErr)
	}
	getFirstSourcesPointer := reflect.ValueOf(getFirstSnapshotEnvelope.Sources).Pointer()
	getSecondSourcesPointer := reflect.ValueOf(getSecondSnapshotEnvelope.Sources).Pointer()
	if getFirstSourcesPointer == 0 || getSecondSourcesPointer == 0 {
		parseT.Fatalf("expected non-nil source maps, got first=%d second=%d", getFirstSourcesPointer, getSecondSourcesPointer)
	}
	if getFirstSourcesPointer != getSecondSourcesPointer {
		parseT.Fatalf(
			"expected source map reuse for stable version tuple, got first=%d second=%d",
			getFirstSourcesPointer,
			getSecondSourcesPointer,
		)
	}
}

// TestHandleHostRegionUpdateSnapshotPropsCacheInvalidatesOnInPlaceMutation verifies cached props short-circuiting does not mask in-place invalid prop mutations.
func TestHandleHostRegionUpdateSnapshotPropsCacheInvalidatesOnInPlaceMutation(parseT *testing.T) {
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
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": 5,
				},
				map[string]uint64{
					"count": 9,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	buildProps := map[string]any{
		"title": "Orders",
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            buildProps,
			SourceIDs:        []string{"count"},
		},
		7,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(initial) returned error: %v", parseErr)
	}
	buildProps["onClick"] = func() {}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            buildProps,
			SourceIDs:        []string{"count"},
		},
		8,
	)
	if parseErr == nil {
		parseT.Fatal("expected in-place prop mutation to revalidate and fail")
	}
}

// TestHandleHostRegionUpdateSnapshotPropsCacheInvalidatesOnTypeFlip verifies cached flat-shape props fingerprints invalidate when one existing key changes value type.
func TestHandleHostRegionUpdateSnapshotPropsCacheInvalidatesOnTypeFlip(parseT *testing.T) {
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
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": 5,
				},
				map[string]uint64{
					"count": 9,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	buildProps := map[string]any{
		"title": "Orders",
		"count": 1,
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            buildProps,
			SourceIDs:        []string{"count"},
		},
		7,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(initial) returned error: %v", parseErr)
	}
	buildProps["count"] = func() {}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            buildProps,
			SourceIDs:        []string{"count"},
		},
		8,
	)
	if parseErr == nil {
		parseT.Fatal("expected prop type flip to revalidate and fail")
	}
}

// TestHandleHostRegionUpdateSnapshotInvalidatesSourceMapReuseOnVersionChange verifies snapshot capture rebuilds source map state when the source-version tuple advances.
func TestHandleHostRegionUpdateSnapshotInvalidatesSourceMapReuseOnVersionChange(parseT *testing.T) {
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
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	getSourceValue := 5
	getSourceVersion := uint64(9)
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": getSourceValue,
				},
				map[string]uint64{
					"count": getSourceVersion,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	getFirstSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		7,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(first) returned error: %v", parseErr)
	}
	getSourceValue = 6
	getSourceVersion = 10
	getSecondSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		8,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(second) returned error: %v", parseErr)
	}
	getFirstSourcesPointer := reflect.ValueOf(getFirstSnapshotEnvelope.Sources).Pointer()
	getSecondSourcesPointer := reflect.ValueOf(getSecondSnapshotEnvelope.Sources).Pointer()
	if getFirstSourcesPointer == getSecondSourcesPointer {
		parseT.Fatalf(
			"expected source map rebuild for advanced version tuple, got first=%d second=%d",
			getFirstSourcesPointer,
			getSecondSourcesPointer,
		)
	}
	if getSecondSnapshotEnvelope.SourceVersion != 10 {
		parseT.Fatalf("expected updated source version 10, got %d", getSecondSnapshotEnvelope.SourceVersion)
	}
	if getSecondSnapshotEnvelope.Sources["count"] != 6 {
		parseT.Fatalf("expected updated source value count=6, got %+v", getSecondSnapshotEnvelope.Sources)
	}
}

// TestHandleHostRegionUpdateSnapshotInvalidatesSourceMapReuseOnRemountEpoch verifies snapshot capture rebuilds source map state after structural remount advances epoch.
func TestHandleHostRegionUpdateSnapshotInvalidatesSourceMapReuseOnRemountEpoch(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("renderer-a"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		3,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": 5,
				},
				map[string]uint64{
					"count": 9,
				},
				nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	getFirstSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("renderer-a"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		7,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(first) returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionStructuralRemount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("renderer-b"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		true,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionStructuralRemount returned error: %v", parseErr)
	}
	getSecondSnapshotEnvelope, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("renderer-b"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		8,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(second) returned error: %v", parseErr)
	}
	getFirstSourcesPointer := reflect.ValueOf(getFirstSnapshotEnvelope.Sources).Pointer()
	getSecondSourcesPointer := reflect.ValueOf(getSecondSnapshotEnvelope.Sources).Pointer()
	if getFirstSourcesPointer == getSecondSourcesPointer {
		parseT.Fatalf(
			"expected source map rebuild for remount epoch change, got first=%d second=%d",
			getFirstSourcesPointer,
			getSecondSourcesPointer,
		)
	}
	if getSecondSnapshotEnvelope.Epoch != 4 {
		parseT.Fatalf("expected remount epoch 4, got %d", getSecondSnapshotEnvelope.Epoch)
	}
}
