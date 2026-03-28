package renderworker

import "testing"

// TestBuildRenderWorkerGenerationTrackerBuildsEmptyTracker validates tracker construction starts from zero generations.
func TestBuildRenderWorkerGenerationTrackerBuildsEmptyTracker(parseT *testing.T) {
	parseTracker := BuildRenderWorkerGenerationTracker()
	if parseTracker == nil {
		parseT.Fatal("expected non-nil generation tracker")
	}
	if parseTracker.GetRenderWorkerLatestGeneration("demo") != 0 {
		parseT.Fatalf("expected empty tracker latest generation 0")
	}
}

// TestShouldRenderWorkerDropStaleRequestTracksLatest validates latest generation tracking and stale-drop behavior.
func TestShouldRenderWorkerDropStaleRequestTracksLatest(parseT *testing.T) {
	parseTracker := BuildRenderWorkerGenerationTracker()
	if parseTracker.ShouldRenderWorkerDropStaleRequest("demo", 10) {
		parseT.Fatalf("expected first generation to be accepted")
	}
	if parseTracker.GetRenderWorkerLatestGeneration("demo") != 10 {
		parseT.Fatalf("expected latest generation to be 10")
	}
	if !parseTracker.ShouldRenderWorkerDropStaleRequest("demo", 9) {
		parseT.Fatalf("expected lower generation to be marked stale")
	}
	if parseTracker.ShouldRenderWorkerDropStaleRequest("demo", 10) {
		parseT.Fatalf("expected equal generation to be accepted")
	}
	if parseTracker.ShouldRenderWorkerDropStaleRequest("demo", 11) {
		parseT.Fatalf("expected higher generation to be accepted")
	}
	if parseTracker.GetRenderWorkerLatestGeneration("demo") != 11 {
		parseT.Fatalf("expected latest generation to be 11")
	}
}

// TestShouldRenderWorkerDropStaleRequestIgnoresInvalidInputs validates nil tracker, blank request names, and zero generations do not stale-drop.
func TestShouldRenderWorkerDropStaleRequestIgnoresInvalidInputs(parseT *testing.T) {
	if (*RenderWorkerGenerationTracker)(nil).ShouldRenderWorkerDropStaleRequest("demo", 1) {
		parseT.Fatalf("expected nil tracker to not stale-drop")
	}
	parseTracker := BuildRenderWorkerGenerationTracker()
	if parseTracker.ShouldRenderWorkerDropStaleRequest(" ", 1) {
		parseT.Fatalf("expected blank request name to not stale-drop")
	}
	if parseTracker.ShouldRenderWorkerDropStaleRequest("demo", 0) {
		parseT.Fatalf("expected zero generation to not stale-drop")
	}
}
