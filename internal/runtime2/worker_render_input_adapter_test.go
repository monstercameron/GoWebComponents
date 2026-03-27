package runtime2

import "testing"

// TestHandleWorkerRegionMountBuildsDeterministicRenderInput verifies mount render-input adapter provides stable source ordering.
func TestHandleWorkerRegionMountBuildsDeterministicRenderInput(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	var parseCapturedInput WorkerRenderInput
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		parseCapturedInput = parseMount.RenderInput
		return map[string]any{"kind": "text", "text": "ok"}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		1,
		map[string]any{"title": "Orders"},
		[]string{"zeta", "alpha"},
		map[string]any{"zeta": "late", "alpha": "first"},
		map[string]uint64{"alpha": 7, "zeta": 7},
	)
	if parseSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseSnapshot,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	if len(parseCapturedInput.GetSourceEntries) != 2 {
		parseTesting.Fatalf("render input source entry count = %d, want 2", len(parseCapturedInput.GetSourceEntries))
	}
	if parseCapturedInput.GetSourceEntries[0].GetSourceID != "alpha" || parseCapturedInput.GetSourceEntries[1].GetSourceID != "zeta" {
		parseTesting.Fatalf("render input source ordering = %+v, want alpha then zeta", parseCapturedInput.GetSourceEntries)
	}
}
