package runtime2_test

import "testing"

import "github.com/monstercameron/GoWebComponents/internal/runtime2"

// TestBuildControlReadyEnvelopeBuildsValidatedEnvelope verifies ready control builders produce one valid envelope.
func TestBuildControlReadyEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseEnvelope := runtime2.BuildControlReadyEnvelope()
	if parseEnvelope.Kind != runtime2.ControlKindReady {
		parseT.Fatalf("expected ready kind, got %q", parseEnvelope.Kind)
	}
	if parseEnvelope.ProtocolVersion != runtime2.ProtocolVersionParallelV1 {
		parseT.Fatalf("expected protocol version %q, got %q", runtime2.ProtocolVersionParallelV1, parseEnvelope.ProtocolVersion)
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(ready builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlCapabilitiesEnvelopeBuildsValidatedEnvelope verifies capabilities control builders produce one valid envelope.
func TestBuildControlCapabilitiesEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseCapabilityReport := runtime2.BuildCapabilityReport(runtime2.CapabilitySource{
		HasWorkerSupport:                true,
		HasMessagePortSupport:           true,
		HasStructuredCloneSupport:       true,
		HasBinaryTransportSupport:       true,
		HasSharedBufferSupport:          true,
		HasSharedMemoryTransportSupport: true,
	})
	parseEnvelope, parseErr := runtime2.BuildControlCapabilitiesEnvelope(parseCapabilityReport)
	if parseErr != nil {
		parseT.Fatalf("BuildControlCapabilitiesEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindCapabilities {
		parseT.Fatalf("expected capabilities kind, got %q", parseEnvelope.Kind)
	}
	if parseEnvelope.Capabilities == nil {
		parseT.Fatal("expected capabilities payload to be present")
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(capabilities builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlMountEnvelopeBuildsValidatedEnvelope verifies mount control builders produce one valid envelope.
func TestBuildControlMountEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseSnapshot, parseErr := runtime2.BuildSnapshotEnvelope(
		"region-1",
		2,
		1,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseErr)
	}
	parseEnvelope, parseErr := runtime2.BuildControlMountEnvelope("dashboard.hot-panel", parseSnapshot)
	if parseErr != nil {
		parseT.Fatalf("BuildControlMountEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindMount {
		parseT.Fatalf("expected mount kind, got %q", parseEnvelope.Kind)
	}
	if parseEnvelope.Snapshot == nil {
		parseT.Fatal("expected mount snapshot payload to be present")
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(mount builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlUpdateEnvelopeBuildsValidatedEnvelope verifies update control builders produce one valid envelope.
func TestBuildControlUpdateEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseSnapshot, parseErr := runtime2.BuildSnapshotEnvelope(
		"region-1",
		2,
		4,
		map[string]any{"title": "Orders"},
		nil,
		nil,
		nil,
	)
	if parseErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseErr)
	}
	parseEnvelope, parseErr := runtime2.BuildControlUpdateEnvelope(parseSnapshot)
	if parseErr != nil {
		parseT.Fatalf("BuildControlUpdateEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindUpdate {
		parseT.Fatalf("expected update kind, got %q", parseEnvelope.Kind)
	}
	if parseEnvelope.Snapshot == nil {
		parseT.Fatal("expected update snapshot payload to be present")
	}
	if parseEnvelope.InputVersion != parseSnapshot.InputVersion {
		parseT.Fatalf("expected input version %d, got %d", parseSnapshot.InputVersion, parseEnvelope.InputVersion)
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(update builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlCancelEnvelopeBuildsValidatedEnvelope verifies cancel control builders produce one valid envelope.
func TestBuildControlCancelEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseEnvelope, parseErr := runtime2.BuildControlCancelEnvelope("region-1")
	if parseErr != nil {
		parseT.Fatalf("BuildControlCancelEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindCancel {
		parseT.Fatalf("expected cancel kind, got %q", parseEnvelope.Kind)
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(cancel builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlDisposeEnvelopeBuildsValidatedEnvelope verifies dispose control builders produce one valid envelope.
func TestBuildControlDisposeEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseEnvelope, parseErr := runtime2.BuildControlDisposeEnvelope("region-1")
	if parseErr != nil {
		parseT.Fatalf("BuildControlDisposeEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindDispose {
		parseT.Fatalf("expected dispose kind, got %q", parseEnvelope.Kind)
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(dispose builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlRestartEnvelopeBuildsValidatedEnvelope verifies restart control builders produce one valid envelope.
func TestBuildControlRestartEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseEnvelope, parseErr := runtime2.BuildControlRestartEnvelope("region-1", 3)
	if parseErr != nil {
		parseT.Fatalf("BuildControlRestartEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindRestart {
		parseT.Fatalf("expected restart kind, got %q", parseEnvelope.Kind)
	}
	if parseEnvelope.Epoch != 3 {
		parseT.Fatalf("expected restart epoch 3, got %d", parseEnvelope.Epoch)
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(restart builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlPatchReadyEnvelopeBuildsValidatedEnvelope verifies patch-ready control builders produce one valid envelope.
func TestBuildControlPatchReadyEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseEnvelope, parseErr := runtime2.BuildControlPatchReadyEnvelope("region-1", 7, runtime2.TransportTierBinary)
	if parseErr != nil {
		parseT.Fatalf("BuildControlPatchReadyEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindPatchReady {
		parseT.Fatalf("expected patch-ready kind, got %q", parseEnvelope.Kind)
	}
	if parseEnvelope.PatchVersion != 7 {
		parseT.Fatalf("expected patch version 7, got %d", parseEnvelope.PatchVersion)
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(patch-ready builder output) returned error: %v", parseErr)
	}
}

// TestBuildControlDiagnosticEnvelopeBuildsValidatedEnvelope verifies diagnostic control builders produce one valid envelope.
func TestBuildControlDiagnosticEnvelopeBuildsValidatedEnvelope(parseT *testing.T) {
	parseEnvelope, parseErr := runtime2.BuildControlDiagnosticEnvelope("region-1", runtime2.ControlDiagnosticEnvelopeSpec{
		DiagnosticType: runtime2.DiagnosticEventKindUpdate,
		DiagnosticText: "update ok",
	})
	if parseErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.Kind != runtime2.ControlKindDiagnostic {
		parseT.Fatalf("expected diagnostic kind, got %q", parseEnvelope.Kind)
	}
	if parseEnvelope.DiagnosticType != string(runtime2.DiagnosticEventKindUpdate) {
		parseT.Fatalf("expected diagnostic type %q, got %q", runtime2.DiagnosticEventKindUpdate, parseEnvelope.DiagnosticType)
	}
	if parseErr := runtime2.ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(diagnostic builder output) returned error: %v", parseErr)
	}
}
