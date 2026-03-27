package runtime2

import "testing"

// TestHandleWorkerRegionUpdateUnchangedInputProducesNoOp verifies unchanged render output reports an explicit no-op result.
func TestHandleWorkerRegionUpdateUnchangedInputProducesNoOp(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"value": "stable",
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer(valid) error = %v", parseRegisterErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) error = %v", parseMountErr)
	}
	parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(no-op) error = %v", parseUpdateErr)
	}
	if !parseUpdateResult.IsNoOp {
		parseTesting.Fatal("HandleWorkerRegionUpdate(no-op) expected explicit no-op result")
	}
	if parseUpdateResult.HasPatchReady {
		parseTesting.Fatal("HandleWorkerRegionUpdate(no-op) expected patch-ready=false")
	}
}

// TestHandleWorkerRegionUpdateChangedInputProducesPatch verifies changed render output emits patch-ready results.
func TestHandleWorkerRegionUpdateChangedInputProducesPatch(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		if parseMount.InputVersion == 1 {
			return map[string]any{
				"value": "before",
			}, nil
		}
		return map[string]any{
			"value": "after",
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer(valid) error = %v", parseRegisterErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) error = %v", parseMountErr)
	}
	parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(patch) error = %v", parseUpdateErr)
	}
	if !parseUpdateResult.HasPatchReady {
		parseTesting.Fatal("HandleWorkerRegionUpdate(patch) expected patch-ready result")
	}
	if parseUpdateResult.IsNoOp {
		parseTesting.Fatal("HandleWorkerRegionUpdate(patch) expected non-no-op result")
	}
}

// TestHandleWorkerRegionUpdateNoOpPreservesVersionProgress verifies no-op updates still advance tracked input version monotonically.
func TestHandleWorkerRegionUpdateNoOpPreservesVersionProgress(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"value": "stable",
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer(valid) error = %v", parseRegisterErr)
	}
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) error = %v", parseMountErr)
	}
	_, parseFirstUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseFirstUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(v2 no-op) error = %v", parseFirstUpdateErr)
	}
	_, parseSecondUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 3,
	})
	if parseSecondUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(v3 no-op) error = %v", parseSecondUpdateErr)
	}
	parseStoredState, hasStoredState := parseWorkerRegionRuntime.GetWorkerRegionState("region-42")
	if !hasStoredState {
		parseTesting.Fatal("GetWorkerRegionState(region-42) expected stored state")
	}
	if parseStoredState.InputVersion != 3 {
		parseTesting.Fatalf("GetWorkerRegionState(region-42) input version = %d, want 3", parseStoredState.InputVersion)
	}
}
