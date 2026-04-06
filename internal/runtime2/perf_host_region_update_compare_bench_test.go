package runtime2

import (
	"fmt"
	"testing"
)

// buildRuntime2LegacyHandleHostRegionUpdate preserves the previous full-entry read plus write-and-copy host update path for benchmark comparison.
func buildRuntime2LegacyHandleHostRegionUpdate(
	parseHostRegionAdapter *HostRegionAdapter,
	parseInputVersion uint64,
) (HostRegionUpdateResult, error) {
	if parseHostRegionAdapter == nil {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseInputVersion == 0 {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host update input version is required")
	}
	if parseHostRegionAdapter.isHostRegionRepairPending {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q repair remount is pending", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseInputVersion < parseHostRegionAdapter.storeHostRegionRepairVersionFloor {
		return HostRegionUpdateResult{}, fmt.Errorf(
			"runtime2: host update input version %d is below repair floor %d",
			parseInputVersion,
			parseHostRegionAdapter.storeHostRegionRepairVersionFloor,
		)
	}
	getCoordinatorEntry, hasCoordinatorEntry := parseHostRegionAdapter.storeCoordinator.GetEntry(parseHostRegionAdapter.storeRegionInstanceID)
	if !hasCoordinatorEntry {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q is not mounted", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if getCoordinatorEntry.IsFallback {
		return HostRegionUpdateResult{}, fmt.Errorf("runtime2: host region %q is in fallback mode", parseHostRegionAdapter.storeRegionInstanceID)
	}
	if parseVersionErr := ValidateMonotonicInputVersion(getCoordinatorEntry.LastDispatchedVersion, parseInputVersion); parseVersionErr != nil {
		return HostRegionUpdateResult{}, parseVersionErr
	}
	getSchedulerJob, parseSchedulerErr := parseHostRegionAdapter.storeScheduler.HandleSchedulerUpdate(string(parseHostRegionAdapter.storeRegionInstanceID))
	if parseSchedulerErr != nil {
		return HostRegionUpdateResult{}, parseSchedulerErr
	}
	getCoordinatorEntry, parseCoordinatorUpdateErr := parseHostRegionAdapter.storeCoordinator.UpdateRegionAndGetEntry(
		parseHostRegionAdapter.storeRegionInstanceID,
		parseInputVersion,
	)
	if parseCoordinatorUpdateErr != nil {
		return HostRegionUpdateResult{}, parseCoordinatorUpdateErr
	}
	parseHostRegionAdapter.storeHostRegionCoordinatorCacheState(
		getCoordinatorEntry.Epoch,
		getCoordinatorEntry.RendererID,
		getCoordinatorEntry.SourceIDs,
		getCoordinatorEntry.IsFallback,
		getCoordinatorEntry.LastSnapshotVersion,
		getCoordinatorEntry.LastDispatchedVersion,
	)
	parseHostRegionAdapter.storeHostRegionLatestValidVersion = parseHostRegionMaxVersion(
		parseHostRegionAdapter.storeHostRegionLatestValidVersion,
		parseInputVersion,
	)
	return HostRegionUpdateResult{
		GetSchedulerJob:      getSchedulerJob,
		GetDispatchedVersion: getCoordinatorEntry.LastDispatchedVersion,
	}, nil
}

// BenchmarkHandleHostRegionUpdateCurrentVsLegacy compares the current host update path against the previous full-entry read plus write-and-copy path.
func BenchmarkHandleHostRegionUpdateCurrentVsLegacy(parseB *testing.B) {
	buildHostRegionAdapter := func() *HostRegionAdapter {
		getHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(
			RegionInstanceID("region-update-bench"),
			[]SchedulerShardID{"shard-a"},
		)
		if parseBuildErr != nil {
			parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
		}
		_, parseMountErr := getHostRegionAdapter.HandleHostRegionMount(
			ParallelRegionSpec{
				RendererID:       RendererID("dashboard.hot-panel"),
				RegionInstanceID: RegionInstanceID("region-update-bench"),
			},
			1,
		)
		if parseMountErr != nil {
			parseB.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
		}
		return getHostRegionAdapter
	}

	parseB.Run("current_dispatch_fields_plus_trusted_write", func(parseB *testing.B) {
		getHostRegionAdapter := buildHostRegionAdapter()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := getHostRegionAdapter.HandleHostRegionUpdate(uint64(parseIndex + 2)); parseErr != nil {
				parseB.Fatalf("HandleHostRegionUpdate returned error: %v", parseErr)
			}
		}
	})

	parseB.Run("legacy_full_entry_plus_update_and_get", func(parseB *testing.B) {
		getHostRegionAdapter := buildHostRegionAdapter()
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := buildRuntime2LegacyHandleHostRegionUpdate(getHostRegionAdapter, uint64(parseIndex+2)); parseErr != nil {
				parseB.Fatalf("buildRuntime2LegacyHandleHostRegionUpdate returned error: %v", parseErr)
			}
		}
	})
}
