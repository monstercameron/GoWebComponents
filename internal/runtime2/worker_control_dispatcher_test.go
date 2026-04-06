package runtime2_test

import "testing"

import "github.com/monstercameron/GoWebComponents/internal/runtime2"

// buildWorkerControlDispatcherRuntime creates one worker runtime with a deterministic test renderer.
func buildWorkerControlDispatcherRuntime(parseTesting *testing.T) *runtime2.WorkerRegionRuntime {
	parseTesting.Helper()
	parseWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"region": parseMount.RegionID,
			"input":  parseMount.InputVersion,
		}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	return parseWorkerRegionRuntime
}

// buildWorkerControlDispatcherSnapshot builds one valid snapshot for control-dispatch tests.
func buildWorkerControlDispatcherSnapshot(parseTesting *testing.T, parseInputVersion uint64) runtime2.SnapshotEnvelope {
	parseTesting.Helper()
	parseSnapshotEnvelope, parseSnapshotErr := runtime2.BuildSnapshotEnvelope(
		"region-1",
		1,
		parseInputVersion,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseSnapshotErr != nil {
		parseTesting.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	return parseSnapshotEnvelope
}

// TestHandleWorkerControlEnvelopeDispatchesMount verifies mount envelopes route into HandleWorkerRegionMount.
func TestHandleWorkerControlEnvelopeDispatchesMount(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseEnvelope, parseEnvelopeErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseEnvelopeErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseEnvelopeErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseEnvelope)
	if parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasMountResult {
		parseTesting.Fatal("expected mount dispatch result")
	}
}

// TestHandleWorkerControlEnvelopeDispatchesUpdate verifies update envelopes route into HandleWorkerRegionUpdate.
func TestHandleWorkerControlEnvelopeDispatchesUpdate(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseUpdateEnvelope, parseUpdateErr := runtime2.BuildControlUpdateEnvelope(buildWorkerControlDispatcherSnapshot(parseTesting, 2))
	if parseUpdateErr != nil {
		parseTesting.Fatalf("BuildControlUpdateEnvelope returned error: %v", parseUpdateErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseUpdateEnvelope)
	if parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(update) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasUpdateResult {
		parseTesting.Fatal("expected update dispatch result")
	}
}

// TestHandleWorkerControlEnvelopeDispatchesEvent verifies event envelopes route into HandleWorkerRegionEvent.
func TestHandleWorkerControlEnvelopeDispatchesEvent(parseTesting *testing.T) {
	parseWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRendererWithMetadata(
		"dashboard.hot-panel",
		func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
			parseEventText := "idle"
			if parseMount.RenderInput.GetEventSlot != nil {
				parseEventText = parseMount.RenderInput.GetEventSlot.EventType
			}
			return map[string]any{
				"kind": "text",
				"text": parseEventText,
			}, nil
		},
		runtime2.RendererMetadata{
			FeatureFlags: []string{"display-only"},
			EventSlotMetadata: runtime2.EventSlotMetadata{
				Version: runtime2.EventSlotMetadataVersionV1,
				Slots: []runtime2.EventSlotRecord{
					{SlotID: "primary.action", EventType: "click"},
				},
			},
		},
	)
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRendererWithMetadata returned error: %v", parseRegisterErr)
	}
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseEventEnvelope, parseEventErr := runtime2.BuildControlEventEnvelope("region-1", runtime2.EventSlotDispatch{
		SlotID:    "primary.action",
		EventType: "click",
		Payload:   map[string]any{"source": "button"},
	})
	if parseEventErr != nil {
		parseTesting.Fatalf("BuildControlEventEnvelope returned error: %v", parseEventErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseEventEnvelope)
	if parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(event) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasEventResult {
		parseTesting.Fatal("expected event dispatch result")
	}
	if !parseDispatchResult.GetEventResult.HasPatchReady {
		parseTesting.Fatalf("expected patch-ready event result, got %+v", parseDispatchResult.GetEventResult)
	}
}

// TestHandleWorkerControlEnvelopeDispatchesCancel verifies cancel envelopes route into HandleWorkerRegionCancel.
func TestHandleWorkerControlEnvelopeDispatchesCancel(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseCancelEnvelope, parseCancelErr := runtime2.BuildControlCancelEnvelope("region-1")
	if parseCancelErr != nil {
		parseTesting.Fatalf("BuildControlCancelEnvelope returned error: %v", parseCancelErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseCancelEnvelope)
	if parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(cancel) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasCancelResult {
		parseTesting.Fatal("expected cancel dispatch result")
	}
}

// TestHandleWorkerControlEnvelopeDispatchesDispose verifies dispose envelopes route into HandleWorkerRegionDispose.
func TestHandleWorkerControlEnvelopeDispatchesDispose(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseDisposeEnvelope, parseDisposeErr := runtime2.BuildControlDisposeEnvelope("region-1")
	if parseDisposeErr != nil {
		parseTesting.Fatalf("BuildControlDisposeEnvelope returned error: %v", parseDisposeErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseDisposeEnvelope)
	if parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(dispose) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasDisposeResult {
		parseTesting.Fatal("expected dispose dispatch result")
	}
}

// TestHandleWorkerControlEnvelopeDispatchesRestart verifies restart envelopes route into HandleWorkerRegionRestart.
func TestHandleWorkerControlEnvelopeDispatchesRestart(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseRestartEnvelope, parseRestartErr := runtime2.BuildControlRestartEnvelope("region-1", 2)
	if parseRestartErr != nil {
		parseTesting.Fatalf("BuildControlRestartEnvelope returned error: %v", parseRestartErr)
	}
	parseDispatchResult, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseRestartEnvelope)
	if parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(restart) returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasRestartResult {
		parseTesting.Fatal("expected restart dispatch result")
	}
}

// TestHandleWorkerControlEnvelopeRejectsUnsupportedKind verifies unsupported worker dispatch kinds fail clearly.
func TestHandleWorkerControlEnvelopeRejectsUnsupportedKind(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	_, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, runtime2.BuildControlReadyEnvelope())
	if parseDispatchErr == nil {
		parseTesting.Fatal("expected unsupported worker dispatch kind to fail")
	}
}

// TestHandleWorkerControlEnvelopeMountPassesSnapshotToRenderer verifies mount dispatch passes the full snapshot envelope to the worker renderer.
func TestHandleWorkerControlEnvelopeMountPassesSnapshotToRenderer(parseTesting *testing.T) {
	parseWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	var parseCapturedSnapshot runtime2.SnapshotEnvelope
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
		parseCapturedSnapshot = parseMount.Snapshot
		return map[string]any{"kind": "text", "text": "ok"}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseSnapshot := buildWorkerControlDispatcherSnapshot(parseTesting, 1)
	parseSnapshot.Props = map[string]any{"title": "Orders"}
	parseSnapshot.Sources = map[string]any{"status": "healthy"}
	parseSnapshot.SourceVersion = 3
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", parseSnapshot)
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	if parseCapturedSnapshot.RegionInstanceID != runtime2.RegionInstanceID("region-1") {
		parseTesting.Fatalf("renderer snapshot region = %q, want %q", parseCapturedSnapshot.RegionInstanceID, runtime2.RegionInstanceID("region-1"))
	}
	if parseCapturedSnapshot.Epoch != 1 || parseCapturedSnapshot.InputVersion != 1 {
		parseTesting.Fatalf("renderer snapshot version fields = epoch=%d input=%d, want epoch=1 input=1", parseCapturedSnapshot.Epoch, parseCapturedSnapshot.InputVersion)
	}
}

// TestHandleWorkerControlEnvelopeUpdatePassesSnapshotToRenderer verifies update dispatch passes the full snapshot envelope to the worker renderer.
func TestHandleWorkerControlEnvelopeUpdatePassesSnapshotToRenderer(parseTesting *testing.T) {
	parseWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	var parseCapturedSnapshot runtime2.SnapshotEnvelope
	parseRegisterErr := parseWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
		parseCapturedSnapshot = parseMount.Snapshot
		return map[string]any{"kind": "text", "text": "ok"}, nil
	})
	if parseRegisterErr != nil {
		parseTesting.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseMountSnapshot := buildWorkerControlDispatcherSnapshot(parseTesting, 1)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", parseMountSnapshot)
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseUpdateSnapshot := buildWorkerControlDispatcherSnapshot(parseTesting, 2)
	parseUpdateSnapshot.Props = map[string]any{"title": "Invoices"}
	parseUpdateEnvelope, parseUpdateErr := runtime2.BuildControlUpdateEnvelope(parseUpdateSnapshot)
	if parseUpdateErr != nil {
		parseTesting.Fatalf("BuildControlUpdateEnvelope returned error: %v", parseUpdateErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseUpdateEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(update) returned error: %v", parseDispatchErr)
	}
	if parseCapturedSnapshot.InputVersion != 2 {
		parseTesting.Fatalf("renderer update snapshot input version = %d, want 2", parseCapturedSnapshot.InputVersion)
	}
	parseCapturedProps, hasCapturedProps := parseCapturedSnapshot.Props.(map[string]any)
	if !hasCapturedProps {
		parseTesting.Fatalf("renderer update snapshot props type = %T, want map[string]any", parseCapturedSnapshot.Props)
	}
	if parseCapturedProps["title"] != "Invoices" {
		parseTesting.Fatalf("renderer update snapshot props[title] = %v, want %v", parseCapturedProps["title"], "Invoices")
	}
}

// TestHandleWorkerControlEnvelopeUpdateRejectsRendererMismatch verifies update dispatch rejects renderer mismatches against mounted worker state.
func TestHandleWorkerControlEnvelopeUpdateRejectsRendererMismatch(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseUpdateEnvelope, parseUpdateErr := runtime2.BuildControlUpdateEnvelope(buildWorkerControlDispatcherSnapshot(parseTesting, 2))
	if parseUpdateErr != nil {
		parseTesting.Fatalf("BuildControlUpdateEnvelope returned error: %v", parseUpdateErr)
	}
	parseUpdateEnvelope.RendererID = runtime2.RendererID("dashboard.other-panel")
	_, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseUpdateEnvelope)
	if parseDispatchErr == nil {
		parseTesting.Fatal("expected renderer mismatch to fail worker update dispatch")
	}
}

// TestHandleWorkerControlEnvelopeMountRejectsSnapshotRegionMismatch verifies mount dispatch rejects snapshot region mismatch against control envelope region.
func TestHandleWorkerControlEnvelopeMountRejectsSnapshotRegionMismatch(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	parseMountEnvelope.RegionInstanceID = runtime2.RegionInstanceID("region-other")
	_, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope)
	if parseDispatchErr == nil {
		parseTesting.Fatal("expected mount snapshot region mismatch to fail worker dispatch")
	}
}

// TestHandleWorkerControlEnvelopeUpdateRejectsSnapshotRegionMismatch verifies update dispatch rejects snapshot region mismatch against control envelope region.
func TestHandleWorkerControlEnvelopeUpdateRejectsSnapshotRegionMismatch(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseUpdateEnvelope, parseUpdateErr := runtime2.BuildControlUpdateEnvelope(buildWorkerControlDispatcherSnapshot(parseTesting, 2))
	if parseUpdateErr != nil {
		parseTesting.Fatalf("BuildControlUpdateEnvelope returned error: %v", parseUpdateErr)
	}
	parseUpdateEnvelope.RegionInstanceID = runtime2.RegionInstanceID("region-other")
	_, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseUpdateEnvelope)
	if parseDispatchErr == nil {
		parseTesting.Fatal("expected update snapshot region mismatch to fail worker dispatch")
	}
}

// TestHandleWorkerControlEnvelopeUpdateRejectsSnapshotEpochMismatch verifies update dispatch rejects snapshot epoch mismatch against mounted worker epoch.
func TestHandleWorkerControlEnvelopeUpdateRejectsSnapshotEpochMismatch(parseTesting *testing.T) {
	parseWorkerRegionRuntime := buildWorkerControlDispatcherRuntime(parseTesting)
	parseMountEnvelope, parseMountErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", buildWorkerControlDispatcherSnapshot(parseTesting, 1))
	if parseMountErr != nil {
		parseTesting.Fatalf("BuildControlMountEnvelope returned error: %v", parseMountErr)
	}
	if _, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseMountEnvelope); parseDispatchErr != nil {
		parseTesting.Fatalf("HandleWorkerControlEnvelope(mount) returned error: %v", parseDispatchErr)
	}
	parseUpdateSnapshot := buildWorkerControlDispatcherSnapshot(parseTesting, 2)
	parseUpdateSnapshot.Epoch = 9
	parseUpdateEnvelope, parseUpdateErr := runtime2.BuildControlUpdateEnvelope(parseUpdateSnapshot)
	if parseUpdateErr != nil {
		parseTesting.Fatalf("BuildControlUpdateEnvelope returned error: %v", parseUpdateErr)
	}
	_, parseDispatchErr := runtime2.HandleWorkerControlEnvelope(parseWorkerRegionRuntime, parseUpdateEnvelope)
	if parseDispatchErr == nil {
		parseTesting.Fatal("expected update snapshot epoch mismatch to fail worker dispatch")
	}
}
