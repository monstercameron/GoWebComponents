package runtime2

import (
	"strings"
	"testing"
)

// TestHandleWorkerRegionRestartInvalidatesPriorEpochState verifies restart blocks older-epoch updates.
func TestHandleWorkerRegionRestartInvalidatesPriorEpochState(parseTesting *testing.T) {
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
		Epoch:        1,
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(epoch=1) error = %v", parseMountErr)
	}
	_, parseRestartErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{
		RegionID: "region-42",
		Epoch:    2,
	})
	if parseRestartErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionRestart(epoch=2) error = %v", parseRestartErr)
	}
	_, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		Epoch:        1,
		InputVersion: 2,
	})
	if parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(old epoch) error = nil, want stale-epoch error")
	}
	if !strings.Contains(parseUpdateErr.Error(), "stale epoch") {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(old epoch) error = %q, want stale-epoch guidance", parseUpdateErr.Error())
	}
}

// TestHandleWorkerRegionRestartAllowsFreshMount verifies restart allows remount with the new epoch.
func TestHandleWorkerRegionRestartAllowsFreshMount(parseTesting *testing.T) {
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
		Epoch:        1,
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(epoch=1) error = %v", parseMountErr)
	}
	_, parseRestartErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{
		RegionID: "region-42",
		Epoch:    2,
	})
	if parseRestartErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionRestart(epoch=2) error = %v", parseRestartErr)
	}
	_, parseRemountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		Epoch:        2,
		InputVersion: 1,
	})
	if parseRemountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(remount epoch=2) error = %v", parseRemountErr)
	}
}

// TestHandleWorkerRegionRestartSuppressesPriorEpochPatch verifies old-epoch updates cannot produce patch-ready output post-restart.
func TestHandleWorkerRegionRestartSuppressesPriorEpochPatch(parseTesting *testing.T) {
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
		Epoch:        1,
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(epoch=1) error = %v", parseMountErr)
	}
	_, parseRestartErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{
		RegionID: "region-42",
		Epoch:    2,
	})
	if parseRestartErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionRestart(epoch=2) error = %v", parseRestartErr)
	}
	parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		Epoch:        1,
		InputVersion: 2,
	})
	if parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(old epoch) error = nil, want stale-epoch error")
	}
	if parseUpdateResult.HasPatchReady {
		parseTesting.Fatal("HandleWorkerRegionUpdate(old epoch) expected no patch-ready result")
	}
}
