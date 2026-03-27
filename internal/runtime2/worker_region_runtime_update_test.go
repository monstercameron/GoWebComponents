package runtime2

import (
	"strings"
	"testing"
)

// TestHandleWorkerRegionUpdateProducesPatchOrNoOp verifies valid updates emit either a patch-ready or no-op result.
func TestHandleWorkerRegionUpdateProducesPatchOrNoOp(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		if parseMount.InputVersion < 2 {
			return map[string]any{
				"value": "stable",
			}, nil
		}
		return map[string]any{
			"value": "next",
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
	_, parseNoOpErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 1,
	})
	if parseNoOpErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(stale same version) error = nil, want stale error")
	}
	parseUpdateResult, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseUpdateErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(valid) error = %v", parseUpdateErr)
	}
	if !parseUpdateResult.HasPatchReady && !parseUpdateResult.IsNoOp {
		parseTesting.Fatal("HandleWorkerRegionUpdate(valid) expected patch-ready or no-op result")
	}
	if !parseUpdateResult.HasPatchReady {
		parseTesting.Fatal("HandleWorkerRegionUpdate(valid) expected patch-ready result for changed render output")
	}
}

// TestHandleWorkerRegionUpdateRejectsUnknownRegion verifies updates for unknown regions fail clearly.
func TestHandleWorkerRegionUpdateRejectsUnknownRegion(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	_, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-missing",
		InputVersion: 1,
	})
	if parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(unknown region) error = nil, want error")
	}
	if !strings.Contains(parseUpdateErr.Error(), "unknown region ID") {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(unknown region) error = %q, want unknown region guidance", parseUpdateErr.Error())
	}
}

// TestHandleWorkerRegionUpdateRejectsStaleVersion verifies stale updates are rejected consistently.
func TestHandleWorkerRegionUpdateRejectsStaleVersion(parseTesting *testing.T) {
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
		InputVersion: 3,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) error = %v", parseMountErr)
	}
	_, parseUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-42",
		InputVersion: 2,
	})
	if parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(stale) error = nil, want error")
	}
	if !strings.Contains(parseUpdateErr.Error(), "stale input version") {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(stale) error = %q, want stale-version guidance", parseUpdateErr.Error())
	}
}

// TestHandleWorkerRegionUpdateRejectsPortalLikeRenderOutput verifies update rejects portal-like renderer output.
func TestHandleWorkerRegionUpdateRejectsPortalLikeRenderOutput(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		if parseMount.InputVersion == 1 {
			return map[string]any{
				"kind": "host-element",
				"tag":  "div",
			}, nil
		}
		return map[string]any{
			"kind": "portal",
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
	if parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(portal-like output) error = nil, want error")
	}
	if !strings.Contains(parseUpdateErr.Error(), "unsupported portal") {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(portal-like output) error = %q, want portal validation guidance", parseUpdateErr.Error())
	}
}

// TestHandleWorkerRegionUpdateRejectsDirectDOMInteropRenderOutput verifies update rejects direct DOM interop output markers.
func TestHandleWorkerRegionUpdateRejectsDirectDOMInteropRenderOutput(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		if parseMount.InputVersion == 1 {
			return map[string]any{
				"kind": "host-element",
				"tag":  "div",
			}, nil
		}
		return map[string]any{
			"kind": "dom-interop",
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
	if parseUpdateErr == nil {
		parseTesting.Fatal("HandleWorkerRegionUpdate(direct DOM interop output) error = nil, want error")
	}
	if !strings.Contains(parseUpdateErr.Error(), "direct DOM interop") {
		parseTesting.Fatalf("HandleWorkerRegionUpdate(direct DOM interop output) error = %q, want direct DOM validation guidance", parseUpdateErr.Error())
	}
}
