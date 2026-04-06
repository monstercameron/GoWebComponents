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
	if len(parseStoredState.RenderIR.GetNodeRecords) == 0 {
		parseTesting.Fatalf("GetWorkerRegionState(%q) expected canonical node records", parseMountSpec.RegionID)
	}
	if parseStoredState.RenderIR.GetRootNodeID == 0 {
		parseTesting.Fatalf("GetWorkerRegionState(%q) canonical root node ID = 0, want non-zero", parseMountSpec.RegionID)
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

// TestRegisterWorkerRegionRendererWithRegisteredMetadataUsesRegistryMetadata verifies worker registration can reuse shared runtime2 registry metadata.
func TestRegisterWorkerRegionRendererWithRegisteredMetadataUsesRegistryMetadata(parseTesting *testing.T) {
	ResetRendererRegistry()
	parseTesting.Cleanup(ResetRendererRegistry)
	parseRendererID, parseRendererIDErr := ParseRendererID("dashboard.hot-panel")
	if parseRendererIDErr != nil {
		parseTesting.Fatalf("ParseRendererID returned error: %v", parseRendererIDErr)
	}
	parseRegisterErr := RegisterRenderer(parseRendererID, func() {}, RendererMetadata{
		PropSchemaVersion: "props.v1",
		FeatureFlags:      []string{"display-only", "derived-state"},
		EventSlotMetadata: EventSlotMetadata{
			Version: EventSlotMetadataVersionV1,
			Slots: []EventSlotRecord{
				{
					SlotID:    "slot-1",
					EventType: "click",
				},
			},
		},
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterRenderer returned error: %v", parseRegisterErr)
	}
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseWorkerRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithRegisteredMetadata(
		"dashboard.hot-panel",
		func(parseMount WorkerRegionMountSpec) (any, error) {
			return map[string]any{
				"kind": "div",
			}, nil
		},
	)
	if parseWorkerRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithRegisteredMetadata returned error: %v", parseWorkerRegisterErr)
	}
	getRendererMetadata := parseWorkerRegionRuntime.storeWorkerRegionRendererMetadataByID["dashboard.hot-panel"]
	if getRendererMetadata.PropSchemaVersion != "props.v1" {
		parseTesting.Fatalf("worker renderer prop schema version = %q, want props.v1", getRendererMetadata.PropSchemaVersion)
	}
	if !HasRendererFeatureFlag(getRendererMetadata, "display-only") || !HasRendererFeatureFlag(getRendererMetadata, "derived-state") {
		parseTesting.Fatalf("worker renderer feature flags = %+v, want display-only and derived-state", getRendererMetadata.FeatureFlags)
	}
	if getRendererMetadata.EventSlotMetadata.Version != EventSlotMetadataVersionV1 {
		parseTesting.Fatalf("worker renderer event-slot metadata = %+v, want v1", getRendererMetadata.EventSlotMetadata)
	}
	if len(getRendererMetadata.EventSlotMetadata.Slots) != 1 || getRendererMetadata.EventSlotMetadata.Slots[0].EventType != "click" {
		parseTesting.Fatalf("worker renderer event-slot slots = %+v, want click slot", getRendererMetadata.EventSlotMetadata.Slots)
	}
}

// TestSetWorkerRegionRendererMetadataUpdatesRegisteredRenderer verifies worker metadata can be updated after registration.
func TestSetWorkerRegionRendererMetadataUpdatesRegisteredRenderer(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{"kind": "div"}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseUpdatedMetadata := RendererMetadata{
		FeatureFlags: []string{"display-only"},
		EventSlotMetadata: EventSlotMetadata{
			Version: EventSlotMetadataVersionV1,
			Slots: []EventSlotRecord{
				{
					SlotID:    "primary.action",
					EventType: "click",
				},
			},
		},
	}
	if parseSetErr := parseWorkerRegionRuntime.SetWorkerRegionRendererMetadata("dashboard.hot-panel", parseUpdatedMetadata); parseSetErr != nil {
		parseTesting.Fatalf("SetWorkerRegionRendererMetadata returned error: %v", parseSetErr)
	}
	getRendererMetadata := parseWorkerRegionRuntime.storeWorkerRegionRendererMetadataByID["dashboard.hot-panel"]
	if len(getRendererMetadata.EventSlotMetadata.Slots) != 1 || getRendererMetadata.EventSlotMetadata.Slots[0].SlotID != "primary.action" {
		parseTesting.Fatalf("expected updated worker renderer event-slot metadata, got %+v", getRendererMetadata.EventSlotMetadata)
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
