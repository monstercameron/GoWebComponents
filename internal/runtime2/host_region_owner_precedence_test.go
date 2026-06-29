package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionOwnerRerenderSuppressesOlderWorkerOutput verifies owner rerender precedence suppresses older worker output.
func TestHandleHostRegionOwnerRerenderSuppressesOlderWorkerOutput(parseT *testing.T) {
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
	if parseErr = buildHostRegionAdapter.HandleHostRegionOwnerRerender(3); parseErr != nil {
		parseT.Fatalf("HandleHostRegionOwnerRerender returned error: %v", parseErr)
	}
	getWorkerOutputResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(2)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseErr)
	}
	if !getWorkerOutputResult.HasIgnored {
		parseT.Fatal("expected older worker output to be ignored after owner rerender")
	}
	if getWorkerOutputResult.HasCommitted {
		parseT.Fatal("expected ignored older worker output not to commit")
	}
}

// TestHandleHostRegionWorkerOutputCommitsNewerVersion verifies worker output newer than owner-rerender committed version still commits.
func TestHandleHostRegionWorkerOutputCommitsNewerVersion(parseT *testing.T) {
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
	if parseErr = buildHostRegionAdapter.HandleHostRegionOwnerRerender(3); parseErr != nil {
		parseT.Fatalf("HandleHostRegionOwnerRerender returned error: %v", parseErr)
	}
	getWorkerOutputResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(4)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseErr)
	}
	if !getWorkerOutputResult.HasCommitted {
		parseT.Fatal("expected newer worker output to commit")
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.LastCommittedVersion != 4 {
		parseT.Fatalf("expected committed version 4, got %d", parseEntry.LastCommittedVersion)
	}
}
