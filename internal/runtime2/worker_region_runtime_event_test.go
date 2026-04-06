package runtime2

import "testing"

// TestHandleWorkerRegionEventPassesSemanticEventToRenderer verifies event dispatch exposes the declared semantic event through worker render input and produces patch-ready output when the renderer reacts.
func TestHandleWorkerRegionEventPassesSemanticEventToRenderer(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata(
		"dashboard.hot-panel",
		func(parseMount WorkerRegionMountSpec) (any, error) {
			parseEventText := "idle"
			if parseMount.RenderInput.GetEventSlot != nil {
				parseEventText = parseMount.RenderInput.GetEventSlot.EventType
			}
			return map[string]any{
				"kind": "text",
				"text": parseEventText,
			}, nil
		},
		RendererMetadata{
			FeatureFlags: []string{"display-only"},
			EventSlotMetadata: EventSlotMetadata{
				Version: EventSlotMetadataVersionV1,
				Slots: []EventSlotRecord{
					{SlotID: "primary.action", EventType: "click"},
				},
			},
		},
	)
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithMetadata returned error: %v", parseRegisterErr)
	}
	parseSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
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
	parseEventResult, parseEventErr := parseWorkerRegionRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
		RegionID: "region-42",
		EventSlot: EventSlotDispatch{
			SlotID:    "primary.action",
			EventType: "click",
			Payload:   map[string]any{"source": "button"},
		},
	})
	if parseEventErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionEvent returned error: %v", parseEventErr)
	}
	if !parseEventResult.HasPatchReady {
		parseTesting.Fatalf("expected patch-ready event result, got %+v", parseEventResult)
	}
	getWorkerRegionState, hasWorkerRegionState := parseWorkerRegionRuntime.GetWorkerRegionState("region-42")
	if !hasWorkerRegionState {
		parseTesting.Fatal("expected worker region state after event dispatch")
	}
	getTextNodeRecord, getTextValue, hasTextNodeRecord := parseResolveWorkerSingleTextNodeRecord(getWorkerRegionState.RenderIR)
	if !hasTextNodeRecord {
		parseTesting.Fatalf("expected single text render IR after event dispatch, got %+v", getWorkerRegionState.RenderIR)
	}
	if getTextValue != "click" {
		parseTesting.Fatalf("expected event-driven text %q, got %q (node=%+v)", "click", getTextValue, getTextNodeRecord)
	}
}

// TestHandleWorkerRegionEventAdvancesInputVersion verifies explicit event input versions advance cached worker state and patch headers.
func TestHandleWorkerRegionEventAdvancesInputVersion(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata(
		"dashboard.hot-panel",
		func(parseMount WorkerRegionMountSpec) (any, error) {
			parseEventText := "idle"
			if parseMount.RenderInput.GetEventSlot != nil {
				parseEventText = parseMount.RenderInput.GetEventSlot.EventType
			}
			return map[string]any{
				"kind": "text",
				"text": parseEventText,
			}, nil
		},
		RendererMetadata{
			FeatureFlags: []string{"display-only"},
			EventSlotMetadata: EventSlotMetadata{
				Version: EventSlotMetadataVersionV1,
				Slots: []EventSlotRecord{
					{SlotID: "primary.action", EventType: "click"},
				},
			},
		},
	)
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithMetadata returned error: %v", parseRegisterErr)
	}
	parseSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
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
	parseEventResult, parseEventErr := parseWorkerRegionRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
		RegionID:     "region-42",
		InputVersion: 2,
		EventSlot: EventSlotDispatch{
			SlotID:    "primary.action",
			EventType: "click",
			Payload:   map[string]any{"source": "button"},
		},
	})
	if parseEventErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionEvent returned error: %v", parseEventErr)
	}
	if !parseEventResult.HasPatchReady {
		parseTesting.Fatalf("expected patch-ready event result, got %+v", parseEventResult)
	}
	if parseEventResult.PatchIR.GetHeader.InputVersion != 2 {
		parseTesting.Fatalf("expected patch header input version 2, got %d", parseEventResult.PatchIR.GetHeader.InputVersion)
	}
	getWorkerRegionState, hasWorkerRegionState := parseWorkerRegionRuntime.GetWorkerRegionState("region-42")
	if !hasWorkerRegionState {
		parseTesting.Fatal("expected worker region state after event dispatch")
	}
	if getWorkerRegionState.InputVersion != 2 {
		parseTesting.Fatalf("expected worker state input version 2, got %d", getWorkerRegionState.InputVersion)
	}
	if getWorkerRegionState.Snapshot.InputVersion != 2 {
		parseTesting.Fatalf("expected worker snapshot input version 2, got %d", getWorkerRegionState.Snapshot.InputVersion)
	}
}

// TestHandleWorkerRegionEventRejectsUndeclaredSlot verifies event dispatch fails when renderer metadata does not declare the semantic event slot.
func TestHandleWorkerRegionEventRejectsUndeclaredSlot(parseTesting *testing.T) {
	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{"kind": "text", "text": "idle"}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-42",
		1,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
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
	_, parseEventErr := parseWorkerRegionRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
		RegionID: "region-42",
		EventSlot: EventSlotDispatch{
			SlotID:    "primary.action",
			EventType: "click",
			Payload:   map[string]any{"source": "button"},
		},
	})
	if parseEventErr == nil {
		parseTesting.Fatal("expected undeclared event slot to fail")
	}
}
