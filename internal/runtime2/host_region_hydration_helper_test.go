package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildHostRegionHydrationAttachHelperRejectsNilAdapter verifies hydration helper builders reject nil adapter handles.
func TestBuildHostRegionHydrationAttachHelperRejectsNilAdapter(parseT *testing.T) {
	if _, parseErr := runtime2.BuildHostRegionHydrationAttachHelper(nil); parseErr == nil {
		parseT.Fatal("expected BuildHostRegionHydrationAttachHelper(nil) to fail")
	}
}

// TestHostRegionHydrationAttachHelperRequiresHydrationAndAnchor verifies helper post-hydration attach requires both hydration completion and shell-anchor registration.
func TestHostRegionHydrationAttachHelperRequiresHydrationAndAnchor(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	buildHydrationHelper, parseHelperErr := runtime2.BuildHostRegionHydrationAttachHelper(buildHostRegionAdapter)
	if parseHelperErr != nil {
		parseT.Fatalf("BuildHostRegionHydrationAttachHelper returned error: %v", parseHelperErr)
	}
	getAttachResult, parseAttachErr := buildHydrationHelper.HandleHostRegionPostHydrationAttach()
	if parseAttachErr == nil {
		parseT.Fatal("expected post-hydration attach before hydration completion and anchor registration to fail")
	}
	if !getAttachResult.HasBlocked {
		parseT.Fatal("expected blocked attach result before hydration completion and anchor registration")
	}
	if parseAnchorErr := buildHydrationHelper.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseAnchorErr != nil {
		parseT.Fatalf("HandleHostRegionRegisterHydratedShellAnchor returned error: %v", parseAnchorErr)
	}
	getAttachResult, parseAttachErr = buildHydrationHelper.HandleHostRegionPostHydrationAttach()
	if parseAttachErr == nil {
		parseT.Fatal("expected post-hydration attach before hydration completion to fail")
	}
	if !getAttachResult.HasBlocked {
		parseT.Fatal("expected blocked attach result before hydration completion")
	}
	if parseHydrationErr := buildHydrationHelper.HandleHostRegionHydrationComplete(); parseHydrationErr != nil {
		parseT.Fatalf("HandleHostRegionHydrationComplete returned error: %v", parseHydrationErr)
	}
	getAttachResult, parseAttachErr = buildHydrationHelper.HandleHostRegionPostHydrationAttach()
	if parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseAttachErr)
	}
	if !getAttachResult.HasAttached {
		parseT.Fatal("expected post-hydration attach to succeed after hydration completion and anchor registration")
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if !parseEntry.IsAttached {
		parseT.Fatal("expected helper attach flow to mark coordinator entry attached")
	}
}
