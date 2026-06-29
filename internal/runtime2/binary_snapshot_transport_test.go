package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestBuildBinarySnapshotEnvelopeRoundTrips verifies valid binary snapshot payloads round-trip.
func TestBuildBinarySnapshotEnvelopeRoundTrips(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		Props:            map[string]any{"title": "Orders"},
		Sources:          map[string]any{"status": "healthy"},
	}
	parsePayload, parseErr := runtime2.BuildBinarySnapshotEnvelope(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySnapshotEnvelope returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinarySnapshotEnvelope returned error: %v", parseErr)
	}
	if parseDecoded.RegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("expected region ID %q, got %q", parseEnvelope.RegionInstanceID, parseDecoded.RegionInstanceID)
	}
}

// TestParseBinarySnapshotEnvelopeRejectsTruncatedPayload verifies truncated binary payloads fail safely.
func TestParseBinarySnapshotEnvelopeRejectsTruncatedPayload(parseT *testing.T) {
	parsePayload := []byte("GWB1\x01\x00\x01\x01\x20\x00\x00\x00\x00\x00\x00\x00{}")
	if _, parseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload); parseErr == nil {
		parseT.Fatal("expected truncated binary payload to fail")
	}
}

// TestParseBinarySnapshotEnvelopeRejectsIncorrectLength verifies declared payload length mismatches fail.
func TestParseBinarySnapshotEnvelopeRejectsIncorrectLength(parseT *testing.T) {
	parseHeader, parseErr := runtime2.BuildBinaryEnvelopeHeader(runtime2.BinaryEnvelopeKindSnapshot, 100, 0)
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryEnvelopeHeader returned error: %v", parseErr)
	}
	parsePayload := append(parseHeader, []byte(`{"region_instance_id":"region-1","epoch":1,"input_version":1}`)...)
	if _, parseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload); parseErr == nil {
		parseT.Fatal("expected declared payload length mismatch to fail")
	}
}
