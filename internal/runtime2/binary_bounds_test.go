package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestParseBinarySnapshotEnvelopeRejectsTruncatedHeader verifies truncated binary headers fail safely.
func TestParseBinarySnapshotEnvelopeRejectsTruncatedHeader(parseT *testing.T) {
	parsePayload := []byte("GWB1\x01\x00")
	if _, parseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload); parseErr == nil {
		parseT.Fatal("expected truncated binary header to fail")
	}
}

// TestParseBinarySnapshotEnvelopeRejectsTruncatedBody verifies truncated binary bodies fail safely.
func TestParseBinarySnapshotEnvelopeRejectsTruncatedBody(parseT *testing.T) {
	parseHeader, parseErr := runtime2.BuildBinaryEnvelopeHeader(runtime2.BinaryEnvelopeKindSnapshot, 64, 0)
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryEnvelopeHeader returned error: %v", parseErr)
	}
	parsePayload := append(parseHeader, []byte(`{"region_instance_id":"region-1","epoch":1,"input_version":1}`)...)
	if _, parseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload); parseErr == nil {
		parseT.Fatal("expected truncated binary body to fail")
	}
}

// TestParseBinarySnapshotEnvelopeRejectsOversizedDeclaredSpan verifies oversized declared payload spans fail safely.
func TestParseBinarySnapshotEnvelopeRejectsOversizedDeclaredSpan(parseT *testing.T) {
	parseHeader, parseErr := runtime2.BuildBinaryEnvelopeHeader(runtime2.BinaryEnvelopeKindSnapshot, 0xFFFFFFF0, 0)
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryEnvelopeHeader returned error: %v", parseErr)
	}
	parsePayload := append(parseHeader, []byte{}...)
	if _, parseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload); parseErr == nil {
		parseT.Fatal("expected oversized declared binary payload span to fail")
	}
}
