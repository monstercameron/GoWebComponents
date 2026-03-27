package runtime2_test

import (
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionRoundTripTimingCapturesDispatchPatchAndCommit verifies host round-trip timing capture records dispatch-to-patch-ready and commit spans.
func TestHandleHostRegionRoundTripTimingCapturesDispatchPatchAndCommit(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	buildHostRegionAdapter.SetHostRegionRoundTripTimingEnabled(true)
	if _, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
	); parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch returned error: %v", parseDispatchErr)
	}
	time.Sleep(2 * time.Millisecond)
	if _, parsePatchReadyErr := buildHostRegionAdapter.HandleHostRegionPatchReady(2); parsePatchReadyErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady returned error: %v", parsePatchReadyErr)
	}
	time.Sleep(2 * time.Millisecond)
	if _, parseWorkerOutputErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(2); parseWorkerOutputErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseWorkerOutputErr)
	}
	getTiming := buildHostRegionAdapter.GetHostRegionRoundTripTiming()
	if getTiming.GetDispatchToPatchReadyNS == 0 {
		parseT.Fatal("expected non-zero dispatch-to-patch-ready timing span")
	}
	if getTiming.GetDispatchToCommitNS == 0 {
		parseT.Fatal("expected non-zero dispatch-to-commit timing span")
	}
	if getTiming.GetPatchReadyToCommitNS == 0 {
		parseT.Fatal("expected non-zero patch-ready-to-commit timing span")
	}
	if getTiming.GetDispatchToCommitNS < getTiming.GetDispatchToPatchReadyNS {
		parseT.Fatalf(
			"expected dispatch-to-commit span >= dispatch-to-patch-ready span, got commit=%d patch=%d",
			getTiming.GetDispatchToCommitNS,
			getTiming.GetDispatchToPatchReadyNS,
		)
	}
}

// TestHandleHostRegionRoundTripTimingDisabledSkipsCapture verifies default adapters skip round-trip timing capture work.
func TestHandleHostRegionRoundTripTimingDisabledSkipsCapture(parseT *testing.T) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	); parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		2,
	); parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch returned error: %v", parseDispatchErr)
	}
	time.Sleep(2 * time.Millisecond)
	if _, parsePatchReadyErr := buildHostRegionAdapter.HandleHostRegionPatchReady(2); parsePatchReadyErr != nil {
		parseT.Fatalf("HandleHostRegionPatchReady returned error: %v", parsePatchReadyErr)
	}
	time.Sleep(2 * time.Millisecond)
	if _, parseWorkerOutputErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(2); parseWorkerOutputErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseWorkerOutputErr)
	}
	getTiming := buildHostRegionAdapter.GetHostRegionRoundTripTiming()
	if getTiming.GetDispatchToPatchReadyNS != 0 {
		parseT.Fatalf("expected dispatch-to-patch-ready timing capture disabled, got %d", getTiming.GetDispatchToPatchReadyNS)
	}
	if getTiming.GetDispatchToCommitNS != 0 {
		parseT.Fatalf("expected dispatch-to-commit timing capture disabled, got %d", getTiming.GetDispatchToCommitNS)
	}
	if getTiming.GetPatchReadyToCommitNS != 0 {
		parseT.Fatalf("expected patch-ready-to-commit timing capture disabled, got %d", getTiming.GetPatchReadyToCommitNS)
	}
}
