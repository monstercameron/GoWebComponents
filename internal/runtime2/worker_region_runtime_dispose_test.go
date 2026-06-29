package runtime2

import (
	"strings"
	"testing"
)

// TestHandleWorkerRegionDisposeClearsCachedState verifies dispose removes cached region state.
func TestHandleWorkerRegionDisposeClearsCachedState(parseTesting *testing.T) {
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
	parseDisposeResult := parseWorkerRegionRuntime.HandleWorkerRegionDispose("region-42")
	if !parseDisposeResult.HasStateCleared {
		parseTesting.Fatal("HandleWorkerRegionDispose(region-42) expected state-cleared result")
	}
	_, hasStoredState := parseWorkerRegionRuntime.GetWorkerRegionState("region-42")
	if hasStoredState {
		parseTesting.Fatal("GetWorkerRegionState(region-42) expected no state after dispose")
	}
}

// TestHandleWorkerRegionDisposeUpdateAfterDisposeFails verifies updates fail clearly after dispose.
func TestHandleWorkerRegionDisposeUpdateAfterDisposeFails(parseTesting *testing.T) {
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
	parseWorkerRegionRuntime.HandleWorkerRegionDispose("region-42")
	_, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(after dispose) error = nil, want error")
	}
	if !strings.Contains(parseUpdateErr.Error(), "unknown region ID") {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(after dispose) error = %q, want unknown-region guidance", parseUpdateErr.Error())
	}
}

// TestHandleWorkerRegionDisposeRepeatedDisposeRemainsSafe verifies repeated dispose requests are safe.
func TestHandleWorkerRegionDisposeRepeatedDisposeRemainsSafe(parseTesting *testing.T) {
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
	parseFirstDisposeResult := parseWorkerRegionRuntime.HandleWorkerRegionDispose("region-42")
	if !parseFirstDisposeResult.HasStateCleared {
		parseTesting.Fatal("HandleWorkerRegionDispose(first) expected cleared state")
	}
	parseSecondDisposeResult := parseWorkerRegionRuntime.HandleWorkerRegionDispose("region-42")
	if parseSecondDisposeResult.HasStateCleared {
		parseTesting.Fatal("HandleWorkerRegionDispose(second) expected stable no-op dispose result")
	}
}
