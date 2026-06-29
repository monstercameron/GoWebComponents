package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// buildHostRegionAdapterForTransportDispatchTests creates one mounted host adapter for snapshot transport hook tests.
func buildHostRegionAdapterForTransportDispatchTests(parseT *testing.T) *runtime2.HostRegionAdapter {
	parseT.Helper()
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
	return buildHostRegionAdapter
}

// buildSnapshotTransportCapabilityReport creates one capability report with shared-memory transport support enabled.
func buildSnapshotTransportCapabilityReport() runtime2.CapabilityReport {
	return runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
}

// TestHandleHostRegionUpdateDispatchWithTransportSelectsSharedTier verifies scheduled host updates select shared snapshot transport when capability and page support are available.
func TestHandleHostRegionUpdateDispatchWithTransportSelectsSharedTier(parseT *testing.T) {
	buildHostRegionAdapter := buildHostRegionAdapterForTransportDispatchTests(parseT)
	buildSharedSnapshotPage, parseSharedPageErr := runtime2.BuildSharedSnapshotPage(4096)
	if parseSharedPageErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseSharedPageErr)
	}
	getDispatchResult, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		buildSnapshotTransportCapabilityReport(),
		buildSharedSnapshotPage,
	)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport returned error: %v", parseDispatchErr)
	}
	if !getDispatchResult.GetDispatchResult.HasScheduled {
		parseT.Fatal("expected scheduled host update dispatch")
	}
	if !getDispatchResult.HasSnapshotTransport {
		parseT.Fatal("expected snapshot transport selection for scheduled host update")
	}
	if getDispatchResult.GetSnapshotTransportResult.GetTransportTier != runtime2.TransportTierSharedBuffer {
		parseT.Fatalf(
			"expected shared-buffer snapshot transport tier, got %q",
			getDispatchResult.GetSnapshotTransportResult.GetTransportTier,
		)
	}
}

// TestHandleHostRegionUpdateDispatchWithTransportSkipsNoChange verifies no-change host updates skip snapshot transport selection.
func TestHandleHostRegionUpdateDispatchWithTransportSkipsNoChange(parseT *testing.T) {
	buildHostRegionAdapter := buildHostRegionAdapterForTransportDispatchTests(parseT)
	buildSharedSnapshotPage, parseSharedPageErr := runtime2.BuildSharedSnapshotPage(4096)
	if parseSharedPageErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseSharedPageErr)
	}
	parseCapabilityReport := buildSnapshotTransportCapabilityReport()
	_, parseFirstDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
		parseCapabilityReport,
		buildSharedSnapshotPage,
	)
	if parseFirstDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport(first) returned error: %v", parseFirstDispatchErr)
	}
	getNoChangeDispatchResult, parseNoChangeDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		3,
		parseCapabilityReport,
		buildSharedSnapshotPage,
	)
	if parseNoChangeDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport(second) returned error: %v", parseNoChangeDispatchErr)
	}
	if !getNoChangeDispatchResult.GetDispatchResult.HasNoChange {
		parseT.Fatal("expected no-change host update dispatch")
	}
	if getNoChangeDispatchResult.HasSnapshotTransport {
		parseT.Fatal("expected no transport selection when host update dispatch short-circuits as no-change")
	}
}
