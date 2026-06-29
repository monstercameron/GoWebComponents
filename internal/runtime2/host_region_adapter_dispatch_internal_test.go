package runtime2

import (
	"errors"
	"strings"
	"testing"
)

// TestHostRegionDispatchEntryPointsRejectNilAdapterAndInvalidPriority verifies dispatch entrypoints fail fast for nil adapters and unsupported priority values.
func TestHostRegionDispatchEntryPointsRejectNilAdapterAndInvalidPriority(parseT *testing.T) {
	var parseNilHostRegionAdapter *HostRegionAdapter
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionUpdateDispatch(ParallelRegionSpec{}, 1); parseErr == nil {
		parseT.Fatal("expected nil host region update dispatch to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransition(ParallelRegionSpec{}, 1, true); parseErr == nil {
		parseT.Fatal("expected nil host region transition dispatch to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(ParallelRegionSpec{}, 1, HostRegionDispatchPriorityUrgent); parseErr == nil {
		parseT.Fatal("expected nil host region priority dispatch to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionOwnerInvalidate(); parseErr == nil {
		parseT.Fatal("expected nil host region owner invalidation to fail")
	}
	if parseErr := parseNilHostRegionAdapter.HandleHostRegionOwnerRerender(1); parseErr == nil {
		parseT.Fatal("expected nil host region owner rerender to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionWorkerOutput(1); parseErr == nil {
		parseT.Fatal("expected nil host region worker output to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionPatchCommit(PatchStreamRaw{}, nil); parseErr == nil {
		parseT.Fatal("expected nil host region patch commit to fail")
	}
	if _, parseErr := parseNilHostRegionAdapter.HandleHostRegionOwnerRemove(); parseErr == nil {
		parseT.Fatal("expected nil host region owner remove to fail")
	}

	parseHostRegionAdapter := &HostRegionAdapter{}
	if _, parseErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(ParallelRegionSpec{}, 1, HostRegionDispatchPriority("later")); parseErr == nil {
		parseT.Fatal("expected unsupported host region dispatch priority to fail")
	} else if !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected unsupported-priority error, got %v", parseErr)
	}
}

// TestHostRegionDispatchGuardsRejectRepairPendingAndUnmountedOperations verifies direct dispatch and owner-side operations reject repair-pending or unmounted adapters before mutating runtime state.
func TestHostRegionDispatchGuardsRejectRepairPendingAndUnmountedOperations(parseT *testing.T) {
	parseGuardedHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID:     RegionInstanceID("region-1"),
		isHostRegionRepairPending: true,
	}
	if _, parseErr := parseGuardedHostRegionAdapter.handleHostRegionUpdateDispatchWithKnownPriority(ParallelRegionSpec{}, 1, HostRegionDispatchPriorityUrgent); parseErr == nil {
		parseT.Fatal("expected repair-pending dispatch to fail")
	} else if !strings.Contains(parseErr.Error(), "repair remount is pending") {
		parseT.Fatalf("expected repair-pending dispatch error, got %v", parseErr)
	}

	parseHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	if _, parseErr := parseHostRegionAdapter.HandleHostRegionOwnerInvalidate(); parseErr == nil {
		parseT.Fatal("expected unmounted owner invalidation to fail")
	}
	if parseErr := parseHostRegionAdapter.HandleHostRegionOwnerRerender(0); parseErr == nil {
		parseT.Fatal("expected zero-input owner rerender to fail")
	}
	if parseErr := parseHostRegionAdapter.HandleHostRegionOwnerRerender(2); parseErr == nil {
		parseT.Fatal("expected unmounted owner rerender to fail")
	}
	if _, parseErr := parseHostRegionAdapter.HandleHostRegionPatchCommit(PatchStreamRaw{}, nil); parseErr == nil {
		parseT.Fatal("expected unmounted patch commit to fail")
	}

	parseHostRegionAdapter.storeHostRegionCoordinatorCacheState(7, RendererID("dashboard.hot-panel"), []string{"count"}, false, 2, 3)
	parseOwnerRemoveResult, parseOwnerRemoveErr := parseHostRegionAdapter.HandleHostRegionOwnerRemove()
	if parseOwnerRemoveErr != nil {
		parseT.Fatalf("HandleHostRegionOwnerRemove returned error: %v", parseOwnerRemoveErr)
	}
	if parseOwnerRemoveResult.HasDisposed || !parseOwnerRemoveResult.HasLateWorkerOutputSuppressed {
		parseT.Fatalf("unexpected owner remove result for unmounted adapter: %+v", parseOwnerRemoveResult)
	}
	if !parseHostRegionAdapter.isHostRegionRemoved || parseHostRegionAdapter.hasHostRegionCoordinatorCache {
		parseT.Fatalf("expected unmounted owner remove to mark region removed and clear cache, got %+v", parseHostRegionAdapter)
	}
}

// TestParseHandleHostRegionWorkerOutputCommitAndDOMFailureKind verify ignored unmounted worker output and DOM failure classification branches.
func TestParseHandleHostRegionWorkerOutputCommitAndDOMFailureKind(parseT *testing.T) {
	parseHostRegionAdapter := &HostRegionAdapter{
		storeRegionInstanceID: RegionInstanceID("region-1"),
		storeCoordinator:      BuildCoordinator(),
	}
	parseWorkerOutputResult, parseWorkerOutputErr := parseHostRegionAdapter.parseHandleHostRegionWorkerOutputCommit(3)
	if parseWorkerOutputErr != nil {
		parseT.Fatalf("parseHandleHostRegionWorkerOutputCommit returned error: %v", parseWorkerOutputErr)
	}
	if !parseWorkerOutputResult.HasIgnored || parseWorkerOutputResult.HasCommitted {
		parseT.Fatalf("unexpected worker output commit result: %+v", parseWorkerOutputResult)
	}

	if getFailureKind := parseGetHostRegionDOMFailureKind(nil); getFailureKind != DOMCommitFailureKindMissingNodeLookup {
		parseT.Fatalf("nil DOM failure kind = %q, want %q", getFailureKind, DOMCommitFailureKindMissingNodeLookup)
	}
	if getFailureKind := parseGetHostRegionDOMFailureKind(errors.New("missing parent anchor during patch")); getFailureKind != DOMCommitFailureKindMissingParentAnchor {
		parseT.Fatalf("parent-anchor DOM failure kind = %q, want %q", getFailureKind, DOMCommitFailureKindMissingParentAnchor)
	}
	if getFailureKind := parseGetHostRegionDOMFailureKind(errors.New("move destination is invalid")); getFailureKind != DOMCommitFailureKindInvalidKeyedMove {
		parseT.Fatalf("keyed-move DOM failure kind = %q, want %q", getFailureKind, DOMCommitFailureKindInvalidKeyedMove)
	}
	if getFailureKind := parseGetHostRegionDOMFailureKind(errors.New("lookup failed")); getFailureKind != DOMCommitFailureKindMissingNodeLookup {
		parseT.Fatalf("default DOM failure kind = %q, want %q", getFailureKind, DOMCommitFailureKindMissingNodeLookup)
	}
}
