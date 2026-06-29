package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionStructuralRemountDetectsRendererIdentityChange verifies renderer identity changes trigger structural remount.
func TestHandleHostRegionStructuralRemountDetectsRendererIdentityChange(parseT *testing.T) {
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
		4,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	getRemountResult, parseErr := buildHostRegionAdapter.HandleHostRegionStructuralRemount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("renderer-b"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		true,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionStructuralRemount returned error: %v", parseErr)
	}
	if !getRemountResult.HasRendererChanged {
		parseT.Fatal("expected renderer identity change to be detected")
	}
	if !getRemountResult.HasRemounted {
		parseT.Fatal("expected renderer identity change to trigger remount")
	}
	if getRemountResult.GetRemountEpoch != 5 {
		parseT.Fatalf("expected remount epoch 5, got %d", getRemountResult.GetRemountEpoch)
	}
}

// TestHandleHostRegionStructuralRemountDetectsShellOwnershipChange verifies shell ownership changes trigger structural remount.
func TestHandleHostRegionStructuralRemountDetectsShellOwnershipChange(parseT *testing.T) {
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
		7,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	getRemountResult, parseErr := buildHostRegionAdapter.HandleHostRegionStructuralRemount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("renderer-a"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		false,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionStructuralRemount returned error: %v", parseErr)
	}
	if !getRemountResult.HasShellOwnershipChanged {
		parseT.Fatal("expected shell ownership change to be detected")
	}
	if !getRemountResult.HasRemounted {
		parseT.Fatal("expected shell ownership change to trigger remount")
	}
	if getRemountResult.GetRemountEpoch != 8 {
		parseT.Fatalf("expected remount epoch 8, got %d", getRemountResult.GetRemountEpoch)
	}
}
