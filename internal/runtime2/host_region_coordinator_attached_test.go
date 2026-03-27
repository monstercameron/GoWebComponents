package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionMountCoordinatorStartsDetached verifies mounted regions start with coordinator attached state disabled.
func TestHandleHostRegionMountCoordinatorStartsDetached(parseT *testing.T) {
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
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.IsAttached {
		parseT.Fatal("expected coordinator attached state false immediately after mount")
	}
}

// TestHandleHostRegionPostHydrationAttachSetsCoordinatorAttached verifies post-hydration attach transitions coordinator attached state to true.
func TestHandleHostRegionPostHydrationAttachSetsCoordinatorAttached(parseT *testing.T) {
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
	if parseHydrationErr := buildHostRegionAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete returned error: %v", parseHydrationErr)
	}
	if parseAnchorErr := buildHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor returned error: %v", parseAnchorErr)
	}
	if _, parseAttachErr := buildHostRegionAdapter.HandleHostRegionPostHydrationAttach(); parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseAttachErr)
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if !parseEntry.IsAttached {
		parseT.Fatal("expected coordinator attached state true after post-hydration attach")
	}
}

// TestHandleHostRegionFallbackOwnershipBeginClearsCoordinatorAttached verifies fallback ownership transitions coordinator attached state to false.
func TestHandleHostRegionFallbackOwnershipBeginClearsCoordinatorAttached(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	if parseHydrationErr := buildHostRegionAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete returned error: %v", parseHydrationErr)
	}
	if parseAnchorErr := buildHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor returned error: %v", parseAnchorErr)
	}
	if _, parseAttachErr := buildHostRegionAdapter.HandleHostRegionPostHydrationAttach(); parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseAttachErr)
	}
	if parseBinaryErr := buildHostRegionAdapter.HandleHostRegionBinaryDecodeFailure(6); parseBinaryErr != nil {
		parseT.Fatalf("HandleHostRegionBinaryDecodeFailure returned error: %v", parseBinaryErr)
	}
	if _, parseFallbackErr := buildHostRegionAdapter.HandleHostRegionFallbackOwnershipBegin(); parseFallbackErr != nil {
		parseT.Fatalf("HandleHostRegionFallbackOwnershipBegin returned error: %v", parseFallbackErr)
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.IsAttached {
		parseT.Fatal("expected coordinator attached state false after fallback ownership begin")
	}
}

// TestHandleHostRegionDisposeAfterAttachRemovesCoordinatorEntry verifies dispose after attach clears coordinator ownership state.
func TestHandleHostRegionDisposeAfterAttachRemovesCoordinatorEntry(parseT *testing.T) {
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
	if parseHydrationErr := buildHostRegionAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete returned error: %v", parseHydrationErr)
	}
	if parseAnchorErr := buildHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor returned error: %v", parseAnchorErr)
	}
	if _, parseAttachErr := buildHostRegionAdapter.HandleHostRegionPostHydrationAttach(); parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseAttachErr)
	}
	if _, parseDisposeErr := buildHostRegionAdapter.HandleHostRegionDispose(); parseDisposeErr != nil {
		parseT.Fatalf("HandleHostRegionDispose returned error: %v", parseDisposeErr)
	}
	if _, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1")); parseHasEntry {
		parseT.Fatal("expected disposed coordinator entry to be removed")
	}
}
