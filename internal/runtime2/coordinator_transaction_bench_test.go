package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// buildCoordinatorForBench builds one mounted coordinator used by transaction micro-benchmarks.
func buildCoordinatorForBench(parseB *testing.B) *runtime2.Coordinator {
	parseB.Helper()
	getCoordinator := runtime2.BuildCoordinator()
	if parseMountErr := getCoordinator.MountRegion(runtime2.CoordinatorEntry{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Epoch:            1,
		SourceIDs:        []string{"count", "status"},
	}); parseMountErr != nil {
		parseB.Fatalf("MountRegion returned error: %v", parseMountErr)
	}
	return getCoordinator
}

// BenchmarkCoordinatorDispatchTransactionCurrentVsLegacy benchmarks combined snapshot-or-dispatch coordinator transactions versus legacy split writes.
func BenchmarkCoordinatorDispatchTransactionCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("legacy_split_snapshot_and_dispatch", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorForBench(parseB)
		parseB.ReportAllocs()
		for getIteration := 0; getIteration < parseB.N; getIteration++ {
			getVersion := uint64(getIteration + 1)
			if parseSnapshotErr := getCoordinator.SetRegionLastSnapshotVersion(runtime2.RegionInstanceID("region-1"), getVersion); parseSnapshotErr != nil {
				parseB.Fatalf("SetRegionLastSnapshotVersion returned error: %v", parseSnapshotErr)
			}
			if parseUpdateErr := getCoordinator.UpdateRegion(runtime2.RegionInstanceID("region-1"), getVersion); parseUpdateErr != nil {
				parseB.Fatalf("UpdateRegion returned error: %v", parseUpdateErr)
			}
		}
	})
	parseB.Run("current_single_snapshot_and_dispatch", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorForBench(parseB)
		parseB.ReportAllocs()
		for getIteration := 0; getIteration < parseB.N; getIteration++ {
			getVersion := uint64(getIteration + 1)
			if _, parseStoreErr := getCoordinator.StoreRegionSnapshotAndDispatchedVersion(
				runtime2.RegionInstanceID("region-1"),
				getVersion,
				getVersion,
			); parseStoreErr != nil {
				parseB.Fatalf("StoreRegionSnapshotAndDispatchedVersion returned error: %v", parseStoreErr)
			}
		}
	})
}

// BenchmarkCoordinatorSnapshotStateCurrentVsLegacy benchmarks snapshot-state writes with source-ID updates using one transaction versus split writes.
func BenchmarkCoordinatorSnapshotStateCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("legacy_split_snapshot_and_sources", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorForBench(parseB)
		parseB.ReportAllocs()
		for getIteration := 0; getIteration < parseB.N; getIteration++ {
			getVersion := uint64(getIteration + 1)
			if parseSourceErr := getCoordinator.SetRegionSourceIDs(runtime2.RegionInstanceID("region-1"), []string{"status", "count", "status"}); parseSourceErr != nil {
				parseB.Fatalf("SetRegionSourceIDs returned error: %v", parseSourceErr)
			}
			if parseSnapshotErr := getCoordinator.SetRegionLastSnapshotVersion(runtime2.RegionInstanceID("region-1"), getVersion); parseSnapshotErr != nil {
				parseB.Fatalf("SetRegionLastSnapshotVersion returned error: %v", parseSnapshotErr)
			}
		}
	})
	parseB.Run("current_single_snapshot_and_sources", func(parseB *testing.B) {
		getCoordinator := buildCoordinatorForBench(parseB)
		parseB.ReportAllocs()
		for getIteration := 0; getIteration < parseB.N; getIteration++ {
			getVersion := uint64(getIteration + 1)
			if _, parseSnapshotErr := getCoordinator.SetRegionSnapshotState(
				runtime2.RegionInstanceID("region-1"),
				getVersion,
				[]string{"status", "count", "status"},
				true,
			); parseSnapshotErr != nil {
				parseB.Fatalf("SetRegionSnapshotState returned error: %v", parseSnapshotErr)
			}
		}
	})
}
