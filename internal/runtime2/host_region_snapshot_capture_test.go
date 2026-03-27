package runtime2_test

import (
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
