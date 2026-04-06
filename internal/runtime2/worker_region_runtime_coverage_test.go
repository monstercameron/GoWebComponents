package runtime2

import (
	"fmt"
	"testing"
)

// TestWorkerRegionRuntimeRegistrationAndConfigurationGuards verifies registration and metadata mutators reject invalid inputs and accept valid updates.
func TestWorkerRegionRuntimeRegistrationAndConfigurationGuards(parseTesting *testing.T) {
	var parseNilRuntime *WorkerRegionRuntime
	if parseErr := parseNilRuntime.RegisterWorkerRegionRendererWithMetadata("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}, RendererMetadata{}); parseErr == nil {
		parseTesting.Fatal("RegisterWorkerRegionRendererWithMetadata(nil runtime) error = nil, want error")
	}

	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata("", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}, RendererMetadata{}); parseErr == nil {
		parseTesting.Fatal("RegisterWorkerRegionRendererWithMetadata(blank renderer ID) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata("dashboard.hot-panel", nil, RendererMetadata{}); parseErr == nil {
		parseTesting.Fatal("RegisterWorkerRegionRendererWithMetadata(nil renderer) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}, RendererMetadata{FeatureFlags: []string{"display-only", "display-only"}}); parseErr == nil {
		parseTesting.Fatal("RegisterWorkerRegionRendererWithMetadata(invalid metadata) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}, RendererMetadata{FeatureFlags: []string{"display-only"}}); parseErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithMetadata(valid) error = %v", parseErr)
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}, RendererMetadata{FeatureFlags: []string{"display-only"}}); parseErr == nil {
		parseTesting.Fatal("RegisterWorkerRegionRendererWithMetadata(duplicate renderer) error = nil, want error")
	}

	var parseNilUpdateRuntime *WorkerRegionRuntime
	if parseErr := parseNilUpdateRuntime.SetWorkerRegionRendererMetadata("dashboard.hot-panel", RendererMetadata{}); parseErr == nil {
		parseTesting.Fatal("SetWorkerRegionRendererMetadata(nil runtime) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.SetWorkerRegionRendererMetadata("", RendererMetadata{}); parseErr == nil {
		parseTesting.Fatal("SetWorkerRegionRendererMetadata(blank renderer ID) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.SetWorkerRegionRendererMetadata("dashboard.missing", RendererMetadata{}); parseErr == nil {
		parseTesting.Fatal("SetWorkerRegionRendererMetadata(unregistered renderer) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.SetWorkerRegionRendererMetadata("dashboard.hot-panel", RendererMetadata{FeatureFlags: []string{"display-only"}}); parseErr != nil {
		parseTesting.Fatalf("SetWorkerRegionRendererMetadata(valid) error = %v", parseErr)
	}

	if parseErr := parseNilUpdateRuntime.SetWorkerRegionRendererTrusted("dashboard.hot-panel", true); parseErr == nil {
		parseTesting.Fatal("SetWorkerRegionRendererTrusted(nil runtime) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("", true); parseErr == nil {
		parseTesting.Fatal("SetWorkerRegionRendererTrusted(blank renderer ID) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("dashboard.missing", true); parseErr == nil {
		parseTesting.Fatal("SetWorkerRegionRendererTrusted(unregistered renderer) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("dashboard.hot-panel", true); parseErr != nil {
		parseTesting.Fatalf("SetWorkerRegionRendererTrusted(true) error = %v", parseErr)
	}
	if parseErr := parseWorkerRegionRuntime.SetWorkerRegionRendererTrusted("dashboard.hot-panel", false); parseErr != nil {
		parseTesting.Fatalf("SetWorkerRegionRendererTrusted(false) error = %v", parseErr)
	}

	parseNilUpdateRuntime.SetWorkerRegionUpdateValidationEnabled(false)
	parseWorkerRegionRuntime.SetWorkerRegionUpdateValidationEnabled(false)
	if parseWorkerRegionRuntime.isWorkerRegionUpdateValidationEnabled {
		parseTesting.Fatal("SetWorkerRegionUpdateValidationEnabled(false) did not update runtime state")
	}
	parseWorkerRegionRuntime.SetWorkerRegionUpdateValidationEnabled(true)
	if !parseWorkerRegionRuntime.isWorkerRegionUpdateValidationEnabled {
		parseTesting.Fatal("SetWorkerRegionUpdateValidationEnabled(true) did not update runtime state")
	}

	ResetRendererRegistry()
	parseTesting.Cleanup(ResetRendererRegistry)
	parseRendererID, parseRendererIDErr := ParseRendererID("dashboard.registry")
	if parseRendererIDErr != nil {
		parseTesting.Fatalf("ParseRendererID returned error: %v", parseRendererIDErr)
	}
	if parseErr := RegisterRenderer(parseRendererID, func() {}, RendererMetadata{FeatureFlags: []string{"display-only"}}); parseErr != nil {
		parseTesting.Fatalf("RegisterRenderer returned error: %v", parseErr)
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithRegisteredMetadata("", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}); parseErr == nil {
		parseTesting.Fatal("RegisterWorkerRegionRendererWithRegisteredMetadata(blank renderer ID) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithRegisteredMetadata("dashboard.registry-missing", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}); parseErr == nil {
		parseTesting.Fatal("RegisterWorkerRegionRendererWithRegisteredMetadata(unregistered renderer) error = nil, want error")
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithRegisteredMetadata("dashboard.registry", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": "ok",
		}, nil
	}); parseErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithRegisteredMetadata(valid) error = %v", parseErr)
	}
}

// TestWorkerRegionRuntimeMountUpdateAndEventValidationPaths verifies the worker runtime rejects invalid mount, update, and event paths while preserving the valid happy path.
func TestWorkerRegionRuntimeMountUpdateAndEventValidationPaths(parseTesting *testing.T) {
	var parseNilRuntime *WorkerRegionRuntime
	if _, parseErr := parseNilRuntime.handleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:   "region-1",
		RendererID: "dashboard.hot-panel",
	}, true); parseErr == nil {
		parseTesting.Fatal("handleWorkerRegionMount(nil runtime) error = nil, want error")
	}
	if _, parseErr := parseNilRuntime.handleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID: "region-1",
	}, true); parseErr == nil {
		parseTesting.Fatal("handleWorkerRegionUpdate(nil runtime) error = nil, want error")
	}
	if _, parseErr := parseNilRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
		RegionID: "region-1",
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionEvent(nil runtime) error = nil, want error")
	}

	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "text",
			"text": fmt.Sprintf("%d", parseMount.InputVersion),
		}, nil
	}, RendererMetadata{
		FeatureFlags: []string{"display-only"},
		EventSlotMetadata: EventSlotMetadata{
			Version: EventSlotMetadataVersionV1,
			Slots: []EventSlotRecord{
				{SlotID: "primary.action", EventType: "click"},
			},
		},
	}); parseErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithMetadata(valid) error = %v", parseErr)
	}
	if parseErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata("dashboard.error", func(parseMount WorkerRegionMountSpec) (any, error) {
		return nil, fmt.Errorf("render failed")
	}, RendererMetadata{FeatureFlags: []string{"display-only"}}); parseErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithMetadata(error renderer) error = %v", parseErr)
	}

	parseValidSnapshot, parseValidSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "ok"},
		nil,
		nil,
		nil,
	)
	if parseValidSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope(valid) returned error: %v", parseValidSnapshotErr)
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RendererID:   "dashboard.hot-panel",
		InputVersion: 1,
		Snapshot:     parseValidSnapshot,
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(missing region ID) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: "region-2",
			Epoch:            1,
			InputVersion:     1,
			Props:            map[string]any{"title": "mismatch"},
		},
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(region mismatch) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: "region-1",
			Epoch:            2,
			InputVersion:     1,
			Props:            map[string]any{"title": "mismatch"},
		},
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(epoch mismatch) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: "region-1",
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "mismatch"},
		},
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(input version mismatch) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.unknown",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseValidSnapshot,
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(unknown renderer) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.error",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseValidSnapshot,
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionMount(renderer error) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.handleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: "region-1",
			Epoch:            1,
			InputVersion:     1,
			Props:            map[string]any{"title": "ok"},
			Sources:          map[string]any{" bad ": "x"},
		},
	}, true); parseErr == nil {
		parseTesting.Fatal("handleWorkerRegionMount(invalid source ID) error = nil, want error")
	}

	parseMountedState, parseMountErr := parseWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseValidSnapshot,
	})
	if parseMountErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionMount(valid) error = %v", parseMountErr)
	}
	parseWorkerRegionRuntime.storeWorkerRegionStateByID["region-1"] = WorkerRegionState{
		RegionID:     parseMountedState.RegionID,
		RendererID:   parseMountedState.RendererID,
		Epoch:        parseMountedState.Epoch,
		InputVersion: 2,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: "region-1",
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "ok"},
			Sources:          map[string]any{" bad ": "x"},
		},
		RenderIR: parseMountedState.RenderIR,
	}
	if _, parseErr := parseWorkerRegionRuntime.handleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-1",
		InputVersion: 3,
	}, true); parseErr == nil {
		parseTesting.Fatal("handleWorkerRegionUpdate(invalid source ID) error = nil, want error")
	}

	parseWorkerRegionRuntime.storeWorkerRegionStateByID["region-event-missing"] = WorkerRegionState{
		RegionID:     "region-event-missing",
		RendererID:   "dashboard.missing-metadata",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseValidSnapshot,
		RenderIR:     parseMountedState.RenderIR,
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
		RegionID: "region-event-missing",
		EventSlot: EventSlotDispatch{
			SlotID:    "primary.action",
			EventType: "click",
		},
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionEvent(missing metadata) error = nil, want error")
	}

	parseWorkerRegionRuntime.storeWorkerRegionStateByID["region-1"] = WorkerRegionState{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 2,
		Snapshot: SnapshotEnvelope{
			RegionInstanceID: "region-1",
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "ok"},
			Sources:          map[string]any{" bad ": "x"},
		},
		RenderIR: parseMountedState.RenderIR,
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
		RegionID: "region-1",
		EventSlot: EventSlotDispatch{
			SlotID:    "primary.action",
			EventType: "click",
		},
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionEvent(invalid source ID) error = nil, want error")
	}

	parseWorkerRegionRuntime.storeWorkerRegionStateByID["region-1"] = WorkerRegionState{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 2,
		Snapshot:     parseValidSnapshot,
		RenderIR:     parseMountedState.RenderIR,
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionEvent(WorkerRegionEventSpec{
		RegionID:     "region-1",
		InputVersion: 1,
		EventSlot: EventSlotDispatch{
			SlotID:    "primary.action",
			EventType: "click",
		},
	}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionEvent(stale input version) error = nil, want error")
	}
}

// TestWorkerRegionRuntimeCancelDisposeRestartAndHelperBranches verifies cancel, dispose, restart, and snapshot helper branches preserve the expected state transitions.
func TestWorkerRegionRuntimeCancelDisposeRestartAndHelperBranches(parseTesting *testing.T) {
	var parseNilRuntime *WorkerRegionRuntime
	if _, parseErr := parseNilRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{RegionID: "region-1"}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionCancel(nil runtime) error = nil, want error")
	}
	if _, parseErr := parseNilRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1", Epoch: 1}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionRestart(nil runtime) error = nil, want error")
	}
	if parseDisposeResult := parseNilRuntime.HandleWorkerRegionDispose("region-1"); parseDisposeResult.HasStateCleared {
		parseTesting.Fatalf("HandleWorkerRegionDispose(nil runtime) result = %+v, want empty result", parseDisposeResult)
	}
	if getState, hasState := parseNilRuntime.GetWorkerRegionState("region-1"); hasState || getState.RegionID != "" {
		parseTesting.Fatalf("GetWorkerRegionState(nil runtime) result = %+v, %t, want empty state", getState, hasState)
	}

	parseWorkerRegionRuntime := BuildWorkerRegionRuntime()
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionCancel(WorkerRegionCancelSpec{InputVersion: 1}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionCancel(blank region ID) error = nil, want error")
	}
	if parseDisposeResult := parseWorkerRegionRuntime.HandleWorkerRegionDispose(" "); parseDisposeResult.HasStateCleared {
		parseTesting.Fatalf("HandleWorkerRegionDispose(blank region ID) result = %+v, want empty result", parseDisposeResult)
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1"}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionRestart(missing epoch) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1", Epoch: 1}); parseErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionRestart(valid) error = %v", parseErr)
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1", Epoch: 1}); parseErr != nil {
		parseTesting.Fatalf("HandleWorkerRegionRestart(same epoch) error = %v", parseErr)
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1", Epoch: 0}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionRestart(zero epoch) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1", Epoch: 0}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionRestart(zero epoch repeat) error = nil, want error")
	}
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1", Epoch: 0}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionRestart(zero epoch third call) error = nil, want error")
	}

	parseWorkerRegionRuntime.storeWorkerRegionEpochFloorByID["region-1"] = 2
	if _, parseErr := parseWorkerRegionRuntime.HandleWorkerRegionRestart(WorkerRegionRestartSpec{RegionID: "region-1", Epoch: 1}); parseErr == nil {
		parseTesting.Fatal("HandleWorkerRegionRestart(stale epoch) error = nil, want error")
	}

	parseWorkerRegionRuntime.storeWorkerRegionStateByID["region-1"] = WorkerRegionState{RegionID: "region-1"}
	parseWorkerRegionRuntime.storeWorkerRegionCanceledByID["region-1"] = 5
	parseWorkerRegionRuntime.storeWorkerRegionPatchVersionByID["region-1"] = 7
	parseDisposeResult := parseWorkerRegionRuntime.HandleWorkerRegionDispose("region-1")
	if !parseDisposeResult.HasStateCleared {
		parseTesting.Fatal("HandleWorkerRegionDispose(region-1) expected state-cleared result")
	}
	if _, hasState := parseWorkerRegionRuntime.GetWorkerRegionState("region-1"); hasState {
		parseTesting.Fatal("GetWorkerRegionState(region-1) expected cleared state")
	}

	parseExplicitSnapshot := SnapshotEnvelope{
		RegionInstanceID: "region-2",
		Epoch:            3,
		InputVersion:     4,
		Props:            map[string]any{"title": "explicit"},
	}
	if parseResolved := parseResolveWorkerUpdateSnapshot(WorkerRegionUpdateSpec{Snapshot: parseExplicitSnapshot}, WorkerRegionState{}, 9); parseResolved.RegionInstanceID != parseExplicitSnapshot.RegionInstanceID || parseResolved.Epoch != parseExplicitSnapshot.Epoch || parseResolved.InputVersion != parseExplicitSnapshot.InputVersion || parseResolved.Props == nil {
		parseTesting.Fatalf("parseResolveWorkerUpdateSnapshot(explicit) = %+v, want %+v", parseResolved, parseExplicitSnapshot)
	}
	parseCachedSnapshot := SnapshotEnvelope{
		RegionInstanceID: "region-3",
		Epoch:            1,
		InputVersion:     2,
		Props:            map[string]any{"title": "cached"},
	}
	parseResolvedSnapshot := parseResolveWorkerUpdateSnapshot(
		WorkerRegionUpdateSpec{InputVersion: 8},
		WorkerRegionState{Snapshot: parseCachedSnapshot},
		9,
	)
	if parseResolvedSnapshot.RegionInstanceID != parseCachedSnapshot.RegionInstanceID || parseResolvedSnapshot.Epoch != 9 || parseResolvedSnapshot.InputVersion != 8 {
		parseTesting.Fatalf("parseResolveWorkerUpdateSnapshot(fallback) = %+v, want cached snapshot with update epoch/version", parseResolvedSnapshot)
	}
	if parseResolvedSnapshot.Props == nil {
		parseTesting.Fatal("parseResolveWorkerUpdateSnapshot(fallback) expected cached props to survive")
	}
	if parseResolvedEmpty := parseResolveWorkerUpdateSnapshot(WorkerRegionUpdateSpec{}, WorkerRegionState{}, 5); parseResolvedEmpty.RegionInstanceID != "" || parseResolvedEmpty.Epoch != 0 || parseResolvedEmpty.InputVersion != 0 {
		parseTesting.Fatalf("parseResolveWorkerUpdateSnapshot(empty) = %+v, want empty snapshot", parseResolvedEmpty)
	}
}

// TestWorkerRegionRuntimeFastTextAndTextRecordBranches verifies fast text patch and single-text record helpers handle both positive and negative paths.
func TestWorkerRegionRuntimeFastTextAndTextRecordBranches(parseTesting *testing.T) {
	parseTextIR, parseTextIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "before",
	})
	if parseTextIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR(text) returned error: %v", parseTextIRErr)
	}
	parseOtherTextIR, parseOtherTextIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "text",
		"text": "after",
	})
	if parseOtherTextIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR(other text) returned error: %v", parseOtherTextIRErr)
	}
	if parseNodeRecord, parseTextValue, hasTextRecord := parseResolveWorkerSingleTextNodeRecord(parseTextIR); !hasTextRecord || parseTextValue != "before" || parseNodeRecord.NodeID == 0 {
		parseTesting.Fatalf("parseResolveWorkerSingleTextNodeRecord(valid) = %+v, %q, %t, want single text node", parseNodeRecord, parseTextValue, hasTextRecord)
	}
	if _, _, hasTextRecord := parseResolveWorkerSingleTextNodeRecord(CanonicalRenderIR{}); hasTextRecord {
		parseTesting.Fatal("parseResolveWorkerSingleTextNodeRecord(empty IR) expected false")
	}
	parseHostIR, parseHostIRErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "div",
	})
	if parseHostIRErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR(host) returned error: %v", parseHostIRErr)
	}
	if _, _, hasTextRecord := parseResolveWorkerSingleTextNodeRecord(parseHostIR); hasTextRecord {
		parseTesting.Fatal("parseResolveWorkerSingleTextNodeRecord(host IR) expected false")
	}
	parseChildCountIR := parseTextIR
	parseChildCountIR.GetNodeRecords = append([]RenderNodeRecordRaw(nil), parseTextIR.GetNodeRecords...)
	parseChildCountIR.GetNodeRecords[0].ChildCount = 1
	if _, _, hasTextRecord := parseResolveWorkerSingleTextNodeRecord(parseChildCountIR); hasTextRecord {
		parseTesting.Fatal("parseResolveWorkerSingleTextNodeRecord(child count mismatch) expected false")
	}
	parseMissingTextRefIR := parseTextIR
	parseMissingTextRefIR.GetNodeRecords = append([]RenderNodeRecordRaw(nil), parseTextIR.GetNodeRecords...)
	parseMissingTextRefIR.GetNodeRecords[0].TextRef = uint32(len(parseTextIR.GetStringTable.Entries))
	if _, _, hasTextRecord := parseResolveWorkerSingleTextNodeRecord(parseMissingTextRefIR); hasTextRecord {
		parseTesting.Fatal("parseResolveWorkerSingleTextNodeRecord(out-of-range text ref) expected false")
	}

	if parsePatch, hasFastPatch, parseErr := buildWorkerRegionFastSetTextPatch("region-fast", 1, 2, 3, parseTextIR, parseTextIR); parseErr != nil || hasFastPatch || parsePatch.GetPatchIdentity != "" {
		parseTesting.Fatalf("buildWorkerRegionFastSetTextPatch(same text) = %+v, %t, %v, want fast-path disabled without error", parsePatch, hasFastPatch, parseErr)
	}
	parseOtherNodeIR := parseOtherTextIR
	parseOtherNodeIR.GetNodeRecords = append([]RenderNodeRecordRaw(nil), parseOtherTextIR.GetNodeRecords...)
	parseOtherNodeIR.GetNodeRecords[0].NodeID++
	if parsePatch, hasFastPatch, parseErr := buildWorkerRegionFastSetTextPatch("region-fast", 1, 2, 3, parseTextIR, parseOtherNodeIR); parseErr != nil || hasFastPatch || parsePatch.GetPatchIdentity != "" {
		parseTesting.Fatalf("buildWorkerRegionFastSetTextPatch(node mismatch) = %+v, %t, %v, want fast-path disabled without error", parsePatch, hasFastPatch, parseErr)
	}
	if parsePatch, hasFastPatch, parseErr := buildWorkerRegionFastSetTextPatch("region-fast", 1, 2, 3, parseTextIR, parseOtherTextIR); parseErr != nil || !hasFastPatch || parsePatch.GetHeader.RegionID != "region-fast" {
		parseTesting.Fatalf("buildWorkerRegionFastSetTextPatch(valid) = %+v, %t, %v, want fast-path patch", parsePatch, hasFastPatch, parseErr)
	}
	if parsePatch, hasFastPatch, parseErr := buildWorkerRegionFastSetTextPatch("region-fast", 1, 2, 3, parseHostIR, parseOtherTextIR); parseErr != nil || hasFastPatch || parsePatch.GetPatchIdentity != "" {
		parseTesting.Fatalf("buildWorkerRegionFastSetTextPatch(non-text previous) = %+v, %t, %v, want fast-path disabled without error", parsePatch, hasFastPatch, parseErr)
	}
}
