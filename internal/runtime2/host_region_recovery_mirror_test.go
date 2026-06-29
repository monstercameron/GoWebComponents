package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

func buildMountedHostRegionAdapterForRecoveryTests(parseT *testing.T) *runtime2.HostRegionAdapter {
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
		2,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	return buildHostRegionAdapter
}

// TestHandleHostRegionStructuredCloneDecodeFailureRoutesThroughRecovery verifies structured-clone decode failures enter recovery fallback.
func TestHandleHostRegionStructuredCloneDecodeFailureRoutesThroughRecovery(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionStructuredCloneDecodeFailure(5); parseErr != nil {
		parseT.Fatalf("HandleHostRegionStructuredCloneDecodeFailure returned error: %v", parseErr)
	}
	if !buildHostRegionAdapter.GetHostRegionRecoveryCoordinator().IsRegionLocalOwnership("region-1") {
		parseT.Fatal("expected structured-clone decode failure to enter recovery fallback ownership")
	}
}

// TestHandleHostRegionBinaryDecodeFailureRoutesThroughRecovery verifies binary decode failures enter recovery fallback.
func TestHandleHostRegionBinaryDecodeFailureRoutesThroughRecovery(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionBinaryDecodeFailure(5); parseErr != nil {
		parseT.Fatalf("HandleHostRegionBinaryDecodeFailure returned error: %v", parseErr)
	}
	if !buildHostRegionAdapter.GetHostRegionRecoveryCoordinator().IsRegionLocalOwnership("region-1") {
		parseT.Fatal("expected binary decode failure to enter recovery fallback ownership")
	}
}

// TestHandleHostRegionSharedPageDecodeFailureRoutesThroughRecovery verifies shared-page decode failures enter recovery fallback.
func TestHandleHostRegionSharedPageDecodeFailureRoutesThroughRecovery(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionSharedPageDecodeFailure(5); parseErr != nil {
		parseT.Fatalf("HandleHostRegionSharedPageDecodeFailure returned error: %v", parseErr)
	}
	if !buildHostRegionAdapter.GetHostRegionRecoveryCoordinator().IsRegionLocalOwnership("region-1") {
		parseT.Fatal("expected shared-page decode failure to enter recovery fallback ownership")
	}
}

// TestHandleHostRegionDOMPatchTransactionFailureRoutesThroughRecovery verifies DOM patch-transaction failures enter recovery fallback.
func TestHandleHostRegionDOMPatchTransactionFailureRoutesThroughRecovery(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionDOMPatchTransactionFailure(runtime2.DOMCommitFailureKindMissingParentAnchor, 6); parseErr != nil {
		parseT.Fatalf("HandleHostRegionDOMPatchTransactionFailure returned error: %v", parseErr)
	}
	if !buildHostRegionAdapter.GetHostRegionRecoveryCoordinator().IsRegionLocalOwnership("region-1") {
		parseT.Fatal("expected DOM patch transaction failure to enter recovery fallback ownership")
	}
}

// TestHandleHostRegionFallbackOwnershipBeginMirrorsToCoordinatorAndScheduler verifies recovery fallback mirrors into coordinator and scheduler ownership.
func TestHandleHostRegionFallbackOwnershipBeginMirrorsToCoordinatorAndScheduler(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionStructuredCloneDecodeFailure(6); parseErr != nil {
		parseT.Fatalf("HandleHostRegionStructuredCloneDecodeFailure returned error: %v", parseErr)
	}
	getFallbackMirrorResult, parseErr := buildHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin()
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackOwnershipBegin returned error: %v", parseErr)
	}
	if !getFallbackMirrorResult.HasCoordinatorFallback {
		parseT.Fatal("expected coordinator fallback mirror to be active")
	}
	if !getFallbackMirrorResult.HasSchedulerFallback {
		parseT.Fatal("expected scheduler fallback mirror to be active")
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if !parseEntry.IsFallback {
		parseT.Fatal("expected coordinator entry fallback state after fallback mirror")
	}
	if !buildHostRegionAdapter.GetHostRegionScheduler().HasSchedulerFallbackOwnership("region-1") {
		parseT.Fatal("expected scheduler fallback ownership marker after fallback mirror")
	}
}

// TestHandleHostRegionFallbackMirrorMirrorsToCoordinatorAndScheduler verifies fallback mirror API aliases recovery fallback mirroring behavior.
func TestHandleHostRegionFallbackMirrorMirrorsToCoordinatorAndScheduler(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionBinaryDecodeFailure(6); parseErr != nil {
		parseT.Fatalf("HandleHostRegionBinaryDecodeFailure returned error: %v", parseErr)
	}
	getFallbackMirrorResult, parseErr := buildHostRegionAdapter.HandleHostRegionFallbackMirror()
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackMirror returned error: %v", parseErr)
	}
	if !getFallbackMirrorResult.HasCoordinatorFallback {
		parseT.Fatal("expected coordinator fallback mirror to be active")
	}
	if !getFallbackMirrorResult.HasSchedulerFallback {
		parseT.Fatal("expected scheduler fallback mirror to be active")
	}
}
