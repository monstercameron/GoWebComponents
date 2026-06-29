package runtime2_test

import "testing"

import "github.com/monstercameron/GoWebComponents/v4/internal/runtime2"

// TestHandleHostRegionPostHydrationAttachBlocksBeforeHydrationComplete verifies worker attach is blocked before hydration completes.
func TestHandleHostRegionPostHydrationAttachBlocksBeforeHydrationComplete(parseT *testing.T) {
	parseHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount(valid) returned error: %v", parseMountErr)
	}
	parseAttachResult, parseAttachErr := parseHostRegionAdapter.HandleHostRegionPostHydrationAttach()
	if parseAttachErr == nil {
		parseT.Fatal("expected post-hydration attach before hydration completion to fail")
	}
	if !parseAttachResult.HasBlocked {
		parseT.Fatal("expected post-hydration attach before hydration completion to report blocked=true")
	}
}

// TestHandleHostRegionPostHydrationAttachSucceedsAfterHydrationComplete verifies worker attach is allowed after hydration completes.
func TestHandleHostRegionPostHydrationAttachSucceedsAfterHydrationComplete(parseT *testing.T) {
	parseHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount(valid) returned error: %v", parseMountErr)
	}
	if parseHydrationErr := parseHostRegionAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete(valid) returned error: %v", parseHydrationErr)
	}
	if parseAnchorErr := parseHostRegionAdapter.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor(valid) returned error: %v", parseAnchorErr)
	}
	if !parseHostRegionAdapter.HasHostRegionHydratedShellAnchor() {
		parseT.Fatal("expected HasHostRegionHydratedShellAnchor() to report true after anchor registration")
	}
	if _, parseDOMIndexErr := parseHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", 1); parseDOMIndexErr != nil {
		parseT.Fatalf("GetRegionDOMNode(region-1,1) returned error after anchor registration: %v", parseDOMIndexErr)
	}
	if !parseHostRegionAdapter.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected GetHostRegionIsHydrationComplete() to report true after hydration completion")
	}
	parseAttachResult, parseAttachErr := parseHostRegionAdapter.HandleHostRegionPostHydrationAttach()
	if parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach(valid) returned error: %v", parseAttachErr)
	}
	if !parseAttachResult.HasAttached {
		parseT.Fatal("expected post-hydration attach to report attached=true")
	}
	if !parseHostRegionAdapter.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected HasHostRegionPostHydrationAttached() to report true after attach")
	}
}

// TestHandleHostRegionPostHydrationAttachBlocksWithoutRegisteredAnchor verifies attach remains blocked until a hydrated shell anchor is indexed.
func TestHandleHostRegionPostHydrationAttachBlocksWithoutRegisteredAnchor(parseT *testing.T) {
	parseHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount(valid) returned error: %v", parseMountErr)
	}
	if parseHydrationErr := parseHostRegionAdapter.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete(valid) returned error: %v", parseHydrationErr)
	}
	parseAttachResult, parseAttachErr := parseHostRegionAdapter.HandleHostRegionPostHydrationAttach()
	if parseAttachErr == nil {
		parseT.Fatal("expected post-hydration attach without anchor registration to fail")
	}
	if !parseAttachResult.HasBlocked {
		parseT.Fatal("expected post-hydration attach without anchor registration to report blocked=true")
	}
}
