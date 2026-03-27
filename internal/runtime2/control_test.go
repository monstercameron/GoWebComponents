package runtime2_test

import "testing"

import "github.com/monstercameron/GoWebComponents/internal/runtime2"

// TestParseControlEnvelopeJSONAcceptsReadyEnvelope verifies ready envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsReadyEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"ready"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONAcceptsCapabilitiesEnvelope verifies capabilities envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsCapabilitiesEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"capabilities","capabilities":{"HasWorkerSupport":true,"HasStructuredCloneSupport":true}}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONRejectsUnsupportedCapabilityValues verifies bad capability combinations fail validation.
func TestParseControlEnvelopeJSONRejectsUnsupportedCapabilityValues(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"capabilities","capabilities":{"HasSharedMemoryTransportSupport":true}}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected unsupported capability values to fail")
	}
}

// TestParseControlEnvelopeJSONAcceptsMountEnvelope verifies mount envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsMountEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"mount","region_instance_id":"region-1","renderer_id":"dashboard.hot-panel","snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":1,"props":{"title":"Orders"}}}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONRejectsMissingMountSnapshot verifies mount envelopes require snapshot payloads.
func TestParseControlEnvelopeJSONRejectsMissingMountSnapshot(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"mount","region_instance_id":"region-1","renderer_id":"dashboard.hot-panel"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected missing mount snapshot to fail")
	}
}

// TestParseControlEnvelopeJSONAcceptsUpdateEnvelope verifies update envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsUpdateEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"update","region_instance_id":"region-1","input_version":2,"snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":2,"props":{"title":"Orders"}}}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONRejectsUnknownUpdateRegion verifies unknown-region update envelopes still require region IDs.
func TestParseControlEnvelopeJSONRejectsUnknownUpdateRegion(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"update","input_version":2,"snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":2}}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected update without region_instance_id to fail")
	}
}

// TestParseControlEnvelopeJSONAcceptsCancelEnvelope verifies cancel envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsCancelEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"cancel","region_instance_id":"region-1"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONAcceptsDisposeEnvelope verifies dispose envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsDisposeEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"dispose","region_instance_id":"region-1"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONAcceptsPatchReadyEnvelope verifies patch-ready envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsPatchReadyEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"patch-ready","region_instance_id":"region-1","patch_version":3,"transport_tier":"binary"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONRejectsWrongTransportTier verifies invalid transport tiers fail validation.
func TestParseControlEnvelopeJSONRejectsWrongTransportTier(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"patch-ready","region_instance_id":"region-1","patch_version":3,"transport_tier":"tape-drive"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected invalid transport tier to fail")
	}
}

// TestParseControlEnvelopeJSONAcceptsDiagnosticEnvelope verifies diagnostic envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsDiagnosticEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"diagnostic","region_instance_id":"region-1","diagnostic_type":"mount","diagnostic_text":"mount ok"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONAcceptsDiagnosticEnvelopeWithTiming verifies structured timing diagnostics decode successfully.
func TestParseControlEnvelopeJSONAcceptsDiagnosticEnvelopeWithTiming(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"diagnostic","region_instance_id":"region-1","diagnostic_type":"update","diagnostic_timing":{"queue_ns":1200,"render_ns":3400,"diff_ns":900,"encode_ns":700,"transport_ns":1100,"commit_ns":800}}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONRejectsMalformedDiagnosticPayload verifies malformed diagnostics fail validation.
func TestParseControlEnvelopeJSONRejectsMalformedDiagnosticPayload(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"diagnostic","region_instance_id":"region-1"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected malformed diagnostic payload to fail")
	}
}

// TestParseControlEnvelopeJSONRejectsUnsupportedDiagnosticType verifies unknown lifecycle diagnostic kinds fail validation.
func TestParseControlEnvelopeJSONRejectsUnsupportedDiagnosticType(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"diagnostic","region_instance_id":"region-1","diagnostic_type":"timing","diagnostic_text":"render=2ms"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected unsupported diagnostic type to fail")
	}
}

// TestParseControlEnvelopeJSONRejectsZeroTimingDiagnosticPayload verifies all-zero structured timing payloads fail validation.
func TestParseControlEnvelopeJSONRejectsZeroTimingDiagnosticPayload(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"diagnostic","region_instance_id":"region-1","diagnostic_type":"update","diagnostic_timing":{"queue_ns":0,"render_ns":0,"diff_ns":0,"encode_ns":0,"transport_ns":0,"commit_ns":0}}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected zero timing diagnostic payload to fail")
	}
}

// TestParseControlEnvelopeJSONAcceptsRestartEnvelope verifies restart envelopes decode successfully.
func TestParseControlEnvelopeJSONAcceptsRestartEnvelope(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"restart","region_instance_id":"region-1","epoch":2}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
}

// TestParseControlEnvelopeJSONRejectsMissingRestartEpoch verifies restart envelopes require epochs.
func TestParseControlEnvelopeJSONRejectsMissingRestartEpoch(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"restart","region_instance_id":"region-1"}`)
	if _, parseErr := runtime2.ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected missing restart epoch to fail")
	}
}
