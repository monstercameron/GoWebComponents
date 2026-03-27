package runtime2

import "testing"

// TestHandleWorkerRegionCancelBeforeCompletionSuppressesPatchReady verifies preemptive cancel suppresses later patch-ready output.
func TestHandleWorkerRegionCancelBeforeCompletionSuppressesPatchReady(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"value": parseMount.InputVersion,
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
	_, parseCancelErr := parseWorkerRegionRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseCancelErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionCancel(before completion) error = %v", parseCancelErr)
	}
	parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(canceled version) error = %v", parseUpdateErr)
	}
	if parseUpdateResult.HasPatchReady {
		parseTesting.Fatal("HandleWorkerRegionUpdate(canceled version) expected patch-ready to be suppressed")
	}
	if !parseUpdateResult.IsCanceled {
		parseTesting.Fatal("HandleWorkerRegionUpdate(canceled version) expected canceled result")
	}
}

// TestHandleWorkerRegionCancelAfterCompletionRemainsConsistent verifies cancel after a completed update does not corrupt later updates.
func TestHandleWorkerRegionCancelAfterCompletionRemainsConsistent(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"value": parseMount.InputVersion,
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
	_, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(v2) error = %v", parseUpdateErr)
	}
	parseCancelResult, parseCancelErr := parseWorkerRegionRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseCancelErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionCancel(after completion) error = %v", parseCancelErr)
	}
	if !parseCancelResult.HasCancelRecorded {
		parseTesting.Fatal("HandleWorkerRegionCancel(after completion) expected recorded cancel state")
	}
	parseUpdateResult, parseUpdateV3Err := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 3,
	})
	if parseUpdateV3Err != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(v3) error = %v", parseUpdateV3Err)
	}
	if parseUpdateResult.IsCanceled {
		parseTesting.Fatal("HandleWorkerRegionUpdate(v3) expected uncanceled result")
	}
}

// TestHandleWorkerRegionCancelRepeatedCancelRemainsSafe verifies duplicate cancel requests do not panic or fail.
func TestHandleWorkerRegionCancelRepeatedCancelRemainsSafe(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"value": parseMount.InputVersion,
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
	_, parseFirstCancelErr := parseWorkerRegionRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseFirstCancelErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionCancel(first) error = %v", parseFirstCancelErr)
	}
	_, parseSecondCancelErr := parseWorkerRegionRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseSecondCancelErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionCancel(second) error = %v", parseSecondCancelErr)
	}
}
