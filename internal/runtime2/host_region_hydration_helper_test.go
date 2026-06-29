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

// TestHostRegionHydrationAttachHelperZeroValueRejectsMutationAndReportsEmptyState verifies zero-value helpers fail closed without exposing adapter internals.
func TestHostRegionHydrationAttachHelperZeroValueRejectsMutationAndReportsEmptyState(parseT *testing.T) {
	var buildHydrationHelper runtime2.HostRegionHydrationAttachHelper
	if buildHydrationHelper.GetHostRegionInstanceID() != "" {
		parseT.Fatalf("expected zero-value helper instance ID to be empty, got %q", buildHydrationHelper.GetHostRegionInstanceID())
	}
	if buildHydrationHelper.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected zero-value helper hydration state to be false")
	}
	if buildHydrationHelper.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected zero-value helper attach state to be false")
	}
	if parseErr := buildHydrationHelper.HandleHostRegionHydrationComplete(); parseErr == nil {
		parseT.Fatal("expected zero-value helper hydration completion to fail")
	}
	if parseErr := buildHydrationHelper.HandleHostRegionRegisterHydratedShellAnchor(1, "div"); parseErr == nil {
		parseT.Fatal("expected zero-value helper anchor registration to fail")
	}
	if _, parseErr := buildHydrationHelper.HandleHostRegionPostHydrationAttach(); parseErr == nil {
		parseT.Fatal("expected zero-value helper post-hydration attach to fail")
	}
}

// TestHostRegionHydrationAttachHelperRequiresHydrationAndAnchor verifies helper post-hydration attach requires both hydration completion and shell-anchor registration.
func TestHostRegionHydrationAttachHelperRequiresHydrationAndAnchor(parseT *testing.T) {
	buildHostRegionAdapter := buildMountedHostRegionAdapterForRecoveryTests(parseT)
	buildHydrationHelper, parseHelperErr := runtime2.BuildHostRegionHydrationAttachHelper(buildHostRegionAdapter)
	if parseHelperErr != nil {
		parseT.Fatalf("BuildHostRegionHydrationAttachHelper returned error: %v", parseHelperErr)
	}
	if buildHydrationHelper.GetHostRegionInstanceID() != runtime2.RegionInstanceID("region-1") {
		parseT.Fatalf("expected helper instance ID %q, got %q", "region-1", buildHydrationHelper.GetHostRegionInstanceID())
	}
	if buildHydrationHelper.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected hydration to start incomplete")
	}
	if buildHydrationHelper.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected attach state to start false")
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
	if !buildHydrationHelper.GetHostRegionIsHydrationComplete() {
		parseT.Fatal("expected helper hydration state to report complete after hydration")
	}
	getAttachResult, parseAttachErr = buildHydrationHelper.HandleHostRegionPostHydrationAttach()
	if parseAttachErr != nil {
		parseT.Fatalf("HandleHostRegionPostHydrationAttach returned error: %v", parseAttachErr)
	}
	if !getAttachResult.HasAttached {
		parseT.Fatal("expected post-hydration attach to succeed after hydration completion and anchor registration")
	}
	if !buildHydrationHelper.HasHostRegionPostHydrationAttached() {
		parseT.Fatal("expected helper attach state to report true after post-hydration attach")
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if !parseEntry.IsAttached {
		parseT.Fatal("expected helper attach flow to mark coordinator entry attached")
	}
}
