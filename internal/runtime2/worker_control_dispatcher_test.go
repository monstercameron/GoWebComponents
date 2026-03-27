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
