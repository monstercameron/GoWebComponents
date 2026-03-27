package runtime2

import (
	"strings"
	"testing"
)

// TestHandleWorkerRegionMountStoresInitialRegionState verifies a valid mount stores initial region state.
func TestHandleWorkerRegionMountStoresInitialRegionState(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"renderer_id": parseMount.RendererID,
			"region_id":   parseMount.RegionID,
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer(valid) error = %v", parseRegisterErr)
	}
	parseMountSpec := WorkerRegionMountSpec{
		RegionID:     "region-42",
		RendererID:   "dashboard.hot-panel",
		InputVersion: 7,
	}
	parseMountedState, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(parseMountSpec)
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) error = %v", parseMountErr)
	}
	if parseMountedState.RendererID != parseMountSpec.RendererID {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) renderer ID = %q, want %q", parseMountedState.RendererID, parseMountSpec.RendererID)
	}
	if parseMountedState.InputVersion != parseMountSpec.InputVersion {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) input version = %d, want %d", parseMountedState.InputVersion, parseMountSpec.InputVersion)
	}
	parseStoredState, hasStoredState := parseWorkerRegionRuntime.GetWorkerRegionState(parseMountSpec.RegionID)
	if !hasStoredState {
		parseTesting.Fatalf("GetWorkerRegionState(%q) did not report stored state", parseMountSpec.RegionID)
	}
	parseStoredRenderIR, hasStoredRenderIR := parseStoredState.RenderIR.(map[string]any)
	if !hasStoredRenderIR {
		parseTesting.Fatalf("GetWorkerRegionState(%q) render IR type = %T, want map[string]any", parseMountSpec.RegionID, parseStoredState.RenderIR)
	}
	if parseStoredRenderIR["region_id"] != parseMountSpec.RegionID {
		parseTesting.Fatalf("GetWorkerRegionState(%q) render IR region_id = %v, want %q", parseMountSpec.RegionID, parseStoredRenderIR["region_id"], parseMountSpec.RegionID)
	}
}

// TestHandleWorkerRegionMountRejectsMissingRendererID verifies mount fails when renderer identity is absent.
func TestHandleWorkerRegionMountRejectsMissingRendererID(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID: "region-42",
	})
	if parseMountErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(missing renderer ID) error = nil, want error")
	}
	if !strings.Contains(parseMountErr.Error(), "renderer ID is required") {
		parseTesting.Fatalf("HandleWorkerRegionMount(missing renderer ID) error = %q, want renderer ID guidance", parseMountErr.Error())
	}
}

// TestHandleWorkerRegionMountRejectsUnknownRendererID verifies mount fails when renderer lookup misses.
func TestHandleWorkerRegionMountRejectsUnknownRendererID(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	_, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:   "region-42",
		RendererID: "dashboard.unknown",
	})
	if parseMountErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(unknown renderer ID) error = nil, want error")
	}
	if !strings.Contains(parseMountErr.Error(), "unknown renderer ID") {
		parseTesting.Fatalf("HandleWorkerRegionMount(unknown renderer ID) error = %q, want unknown renderer guidance", parseMountErr.Error())
	}
}

// TestHandleWorkerRegionMountRejectsPortalLikeRenderOutput verifies mount rejects portal-like renderer output.
func TestHandleWorkerRegionMountRejectsPortalLikeRenderOutput(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
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
	if parseMountErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(portal-like output) error = nil, want error")
	}
	if !strings.Contains(parseMountErr.Error(), "unsupported portal") {
		parseTesting.Fatalf("HandleWorkerRegionMount(portal-like output) error = %q, want portal validation guidance", parseMountErr.Error())
	}
}

// TestHandleWorkerRegionMountRejectsDirectDOMInteropRenderOutput verifies mount rejects direct DOM interop output markers.
func TestHandleWorkerRegionMountRejectsDirectDOMInteropRenderOutput(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
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
	if parseMountErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(direct DOM interop output) error = nil, want error")
	}
	if !strings.Contains(parseMountErr.Error(), "direct DOM interop") {
		parseTesting.Fatalf("HandleWorkerRegionMount(direct DOM interop output) error = %q, want direct DOM validation guidance", parseMountErr.Error())
	}
}
