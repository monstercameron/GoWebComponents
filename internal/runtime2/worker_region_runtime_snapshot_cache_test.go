package runtime2

import "testing"

// TestHandleWorkerRegionUpdateCachesLatestSnapshot verifies worker state keeps the latest accepted snapshot after updates.
func TestHandleWorkerRegionUpdateCachesLatestSnapshot(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		parseProps, _ := parseMount.Snapshot.Props.(map[string]any)
		return map[string]any{
			"kind": "text",
			"text": parseProps["title"],
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseMountSnapshot, parseMountSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		1,
		map[string]any{"title": "A"},
		nil,
		nil,
		nil,
	)
	if parseMountSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(mount) returned error: %v", parseMountSnapshotErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseMountSnapshot,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	parseUpdateSnapshot, parseUpdateSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		2,
		map[string]any{"title": "B"},
		nil,
		nil,
		nil,
	)
	if parseUpdateSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(update) returned error: %v", parseUpdateSnapshotErr)
	}
	_, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		Epoch:        1,
		InputVersion: 2,
		Snapshot:     parseUpdateSnapshot,
	})
	if parseUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate returned error: %v", parseUpdateErr)
	}
	parseWorkerRegionState, hasWorkerRegionState := parseWorkerRegionRuntime.GetWorkerRegionState("region-42")
	if !hasWorkerRegionState {
		parseTesting.Fatal("expected worker region state after update")
	}
	parseCachedProps, hasCachedProps := parseWorkerRegionState.Snapshot.Props.(map[string]any)
	if !hasCachedProps {
		parseTesting.Fatalf("cached snapshot props type = %T, want map[string]any", parseWorkerRegionState.Snapshot.Props)
	}
	if parseCachedProps["title"] != "B" {
		parseTesting.Fatalf("cached snapshot title = %v, want %v", parseCachedProps["title"], "B")
	}
}
