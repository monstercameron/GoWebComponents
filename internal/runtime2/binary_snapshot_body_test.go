package runtime2

import (
	"encoding/binary"
	"strings"
	"testing"
)

// buildBinarySnapshotBodyVersionValidationTestPayload builds one valid binary snapshot body used by required-version parse tests.
func buildBinarySnapshotBodyVersionValidationTestPayload(parseT *testing.T) []byte {
	parseT.Helper()
	parsePayload, parseErr := BuildBinarySnapshotBody(SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            7,
		InputVersion:     19,
		Props:            map[string]any{"title": "Orders"},
		Sources:          map[string]any{"status": "healthy"},
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySnapshotBody returned error: %v", parseErr)
	}
	return parsePayload
}

// TestParseBinarySnapshotBodyRejectsEpochZero verifies body parse rejects payloads with epoch zero.
func TestParseBinarySnapshotBodyRejectsEpochZero(parseT *testing.T) {
	parsePayload := buildBinarySnapshotBodyVersionValidationTestPayload(parseT)
	parseEpochOffset := 2 + len("region-1")
	binary.LittleEndian.PutUint64(parsePayload[parseEpochOffset:parseEpochOffset+8], 0)
	if _, parseErr := ParseBinarySnapshotBody(parsePayload); parseErr == nil {
		parseT.Fatal("expected ParseBinarySnapshotBody to fail when epoch is zero")
	} else if !strings.Contains(parseErr.Error(), "snapshot epoch is required") {
		parseT.Fatalf("expected epoch-required error, got %v", parseErr)
	}
}

// TestParseBinarySnapshotBodyRejectsInputVersionZero verifies body parse rejects payloads with input version zero.
func TestParseBinarySnapshotBodyRejectsInputVersionZero(parseT *testing.T) {
	parsePayload := buildBinarySnapshotBodyVersionValidationTestPayload(parseT)
	parseInputVersionOffset := 2 + len("region-1") + 8
	binary.LittleEndian.PutUint64(parsePayload[parseInputVersionOffset:parseInputVersionOffset+8], 0)
	if _, parseErr := ParseBinarySnapshotBody(parsePayload); parseErr == nil {
		parseT.Fatal("expected ParseBinarySnapshotBody to fail when input version is zero")
	} else if !strings.Contains(parseErr.Error(), "snapshot input version is required") {
		parseT.Fatalf("expected input-version-required error, got %v", parseErr)
	}
}
