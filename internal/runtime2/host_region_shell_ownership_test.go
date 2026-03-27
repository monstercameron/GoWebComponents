package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestIsHostRegionLocalShellOwnershipTracksInitialMount verifies host adapters keep local-first shell ownership active immediately after mount.
func TestIsHostRegionLocalShellOwnershipTracksInitialMount(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	if buildHostRegionAdapter.IsHostRegionLocalShellOwnership() {
		parseT.Fatal("expected local shell ownership to be false before mount")
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
	if !buildHostRegionAdapter.IsHostRegionLocalShellOwnership() {
		parseT.Fatal("expected local shell ownership to be true after mount")
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionDispose()
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionDispose returned error: %v", parseErr)
	}
	if buildHostRegionAdapter.IsHostRegionLocalShellOwnership() {
		parseT.Fatal("expected local shell ownership to clear on dispose")
	}
}
