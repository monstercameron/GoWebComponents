package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHostRegionIntegratedDOMPatchPrevalidationCoverage verifies invalid DOM patch transactions can route fallback through host recovery handling.
func TestHostRegionIntegratedDOMPatchPrevalidationCoverage(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	buildDOMCommitter := runtime2.BuildDOMCommitter(buildHostRegionAdapter.GetHostRegionDOMIndex())
	if parseErr := buildHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode(
		"region-1",
		1,
		&runtime2.RegionDOMNode{
			GetNodeID:       1,
			GetTag:          "div",
			GetChildNodeIDs: []uint64{},
		},
	); parseErr != nil {
		parseT.Fatalf("SetRegionDOMNode returned error: %v", parseErr)
	}
	getTransactionResult, _ := buildDOMCommitter.CommitRegionPatchTransaction(runtime2.RegionPatchTransaction{
		GetRegionID: "region-1",
		GetOps: []runtime2.RegionPatchOp{
			{
				GetKind:         runtime2.RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetBeforeNodeID: 9,
				GetInsertNode: &runtime2.RegionDOMNode{
					GetNodeID: 2,
					GetTag:    "span",
				},
			},
		},
	})
	if !getTransactionResult.HasFallbackEntered {
		parseT.Fatal("expected invalid DOM patch transaction to trigger fallback transaction result")
	}
	if parseErr := buildHostRegionAdapter.HandleHostRegionDOMPatchTransactionFailure(runtime2.DOMCommitFailureKindMissingParentAnchor, 5); parseErr != nil {
		parseT.Fatalf("HandleHostRegionDOMPatchTransactionFailure returned error: %v", parseErr)
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackOwnershipBegin returned error: %v", parseErr)
	}
	if !buildHostRegionAdapter.GetHostRegionIsFallbackActive() {
		parseT.Fatal("expected host fallback active after DOM patch prevalidation failure routing")
	}
}

// TestHostRegionWorkerRestartRaceCoverage verifies restart-like race output is rejected while repair remount is pending.
func TestHostRegionWorkerRestartRaceCoverage(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	getWorkerDeathResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	getPatchReadyResult, parseErr := buildHostRegionAdapter.HandleHostRegionPatchReady(getWorkerDeathResult.GetVersionFloor)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady returned error: %v", parseErr)
	}
	if !getPatchReadyResult.HasIgnored {
		parseT.Fatal("expected restart-race patch-ready output to be ignored before repair remount")
	}
}

// TestHostRegionWorkerDeathDuringPatchCoverage verifies worker death during patch-ready handling blocks commit.
func TestHostRegionWorkerDeathDuringPatchCoverage(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	getWorkerDeathResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	getWorkerOutputResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(getWorkerDeathResult.GetVersionFloor)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseErr)
	}
	if !getWorkerOutputResult.HasIgnored {
		parseT.Fatal("expected worker output to be ignored while death-repair flow is pending")
	}
}

// TestHostRegionRapidMountDisposeMountChurnCoverage verifies repeated mount-dispose-mount churn remains stable.
func TestHostRegionRapidMountDisposeMountChurnCoverage(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	for parseIteration := 1; parseIteration <= 3; parseIteration++ {
		if _, parseErr = buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		}, uint64(parseIteration)); parseErr != nil {
			parseT.Fatalf("HandleHostRegionMount iteration %d returned error: %v", parseIteration, parseErr)
		}
		if _, parseErr = buildHostRegionAdapter.HandleHostRegionDispose(); parseErr != nil {
			parseT.Fatalf("HandleHostRegionDispose iteration %d returned error: %v", parseIteration, parseErr)
		}
	}
}

// TestHostRegionRapidUpdateCancelUpdateChurnCoverage verifies repeated update-cancel-update churn remains stable.
func TestHostRegionRapidUpdateCancelUpdateChurnCoverage(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{"count": 1}, map[string]uint64{"count": 1}, nil
		},
	); parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	for parseIteration := uint64(3); parseIteration <= 5; parseIteration++ {
		if _, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"tick": parseIteration},
			SourceIDs:        []string{"count"},
		}, parseIteration); parseErr != nil {
			parseT.Fatalf("HandleHostRegionUpdateDispatch iteration %d returned error: %v", parseIteration, parseErr)
		}
		if _, parseErr := buildHostRegionAdapter.HandleHostRegionOwnerInvalidate(); parseErr != nil {
			parseT.Fatalf("HandleHostRegionOwnerInvalidate iteration %d returned error: %v", parseIteration, parseErr)
		}
	}
}

// TestHostRegionDisposeDuringFallbackCoverage verifies dispose remains stable while fallback ownership is active.
func TestHostRegionDisposeDuringFallbackCoverage(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionSharedPageDecodeFailure(7); parseErr != nil {
		parseT.Fatalf("HandleHostRegionSharedPageDecodeFailure returned error: %v", parseErr)
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackOwnershipBegin returned error: %v", parseErr)
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionDispose(); parseErr != nil {
		parseT.Fatalf("HandleHostRegionDispose returned error: %v", parseErr)
	}
}

// TestHostRegionFallbackThenRemountCoverage verifies fallback ownership can clear after a healthy remount handshake.
func TestHostRegionFallbackThenRemountCoverage(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseErr := buildHostRegionAdapter.HandleHostRegionBinaryDecodeFailure(8); parseErr != nil {
		parseT.Fatalf("HandleHostRegionBinaryDecodeFailure returned error: %v", parseErr)
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackOwnershipBegin returned error: %v", parseErr)
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerDeath(true); parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerDeath returned error: %v", parseErr)
	}
	if _, parseErr := buildHostRegionAdapter.HandleHostRegionRepairRemount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}); parseErr != nil {
		parseT.Fatalf("HandleHostRegionRepairRemount returned error: %v", parseErr)
	}
	if buildHostRegionAdapter.GetHostRegionIsFallbackActive() {
		parseT.Fatal("expected fallback ownership to clear after fallback-then-remount handshake")
	}
}
