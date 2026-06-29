package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestBuildHostRegionAdapterOwnsRuntimeHandles verifies one host adapter owns the required runtime2 handles for a live region.
func TestBuildHostRegionAdapterOwnsRuntimeHandles(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	if buildHostRegionAdapter == nil {
		parseT.Fatal("expected host region adapter")
	}
	if buildHostRegionAdapter.GetHostRegionInstanceID() != runtime2.RegionInstanceID("region-1") {
		parseT.Fatalf("expected host region instance ID %q, got %q", "region-1", buildHostRegionAdapter.GetHostRegionInstanceID())
	}
	if buildHostRegionAdapter.GetHostRegionCoordinator() == nil {
		parseT.Fatal("expected host region coordinator handle")
	}
	if buildHostRegionAdapter.GetHostRegionScheduler() == nil {
		parseT.Fatal("expected host region scheduler handle")
	}
	if buildHostRegionAdapter.GetHostRegionRecoveryCoordinator() == nil {
		parseT.Fatal("expected host region recovery coordinator handle")
	}
	if buildHostRegionAdapter.GetHostRegionDOMIndex() == nil {
		parseT.Fatal("expected host region DOM index handle")
	}
}

// TestBuildHostRegionAdapterRejectsMissingRegionInstanceID verifies host adapter construction requires a valid region instance ID.
func TestBuildHostRegionAdapterRejectsMissingRegionInstanceID(parseT *testing.T) {
	_, parseErr := runtime2.BuildHostRegionAdapter("", []runtime2.SchedulerShardID{"shard-a"})
	if parseErr == nil {
		parseT.Fatal("expected host adapter build to fail for missing region instance ID")
	}
}

// TestBuildHostRegionAdapterRejectsMissingSchedulerShards verifies host adapter construction requires at least one scheduler shard.
func TestBuildHostRegionAdapterRejectsMissingSchedulerShards(parseT *testing.T) {
	_, parseErr := runtime2.BuildHostRegionAdapter(runtime2.RegionInstanceID("region-1"), nil)
	if parseErr == nil {
		parseT.Fatal("expected host adapter build to fail for missing scheduler shards")
	}
}
