package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildBinaryMountEnvelopeRoundTrips verifies valid binary mount envelopes round-trip.
func TestBuildBinaryMountEnvelopeRoundTrips(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinaryMountEnvelope(runtime2.BinaryMountEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		SourceIDs:        []string{"status"},
		Snapshot: runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     1,
			Props:            map[string]any{"title": "Orders"},
			Sources:          map[string]any{"status": "ok"},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryMountEnvelope returned error: %v", parseErr)
	}
	parseEnvelope, parseErr := runtime2.ParseBinaryMountEnvelope(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinaryMountEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.RendererID != runtime2.RendererID("dashboard.hot-panel") {
		parseT.Fatalf("expected renderer ID dashboard.hot-panel, got %q", parseEnvelope.RendererID)
	}
}

// TestBuildBinaryMountEnvelopeRejectsSnapshotRegionMismatch verifies binary mount encode rejects mismatched snapshot region IDs.
func TestBuildBinaryMountEnvelopeRejectsSnapshotRegionMismatch(parseT *testing.T) {
	_, parseErr := runtime2.BuildBinaryMountEnvelope(runtime2.BinaryMountEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Snapshot: runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-2"),
			Epoch:            1,
			InputVersion:     1,
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected binary mount region mismatch to fail")
	}
}

// TestBuildBinaryUpdateEnvelopeRoundTrips verifies valid binary update envelopes round-trip.
func TestBuildBinaryUpdateEnvelopeRoundTrips(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinaryUpdateEnvelope(runtime2.BinaryUpdateEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		InputVersion:     2,
		Snapshot: runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "Orders"},
		},
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinaryUpdateEnvelope returned error: %v", parseErr)
	}
	parseEnvelope, parseErr := runtime2.ParseBinaryUpdateEnvelope(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseBinaryUpdateEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.InputVersion != 2 {
		parseT.Fatalf("expected input version 2, got %d", parseEnvelope.InputVersion)
	}
}

// TestBuildBinaryUpdateEnvelopeRejectsSnapshotVersionMismatch verifies binary update encode rejects mismatched snapshot input versions.
func TestBuildBinaryUpdateEnvelopeRejectsSnapshotVersionMismatch(parseT *testing.T) {
	_, parseErr := runtime2.BuildBinaryUpdateEnvelope(runtime2.BinaryUpdateEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		InputVersion:     2,
		Snapshot: runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     3,
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected binary update input-version mismatch to fail")
	}
}

// TestParseBinaryMountEnvelopeRejectsWrongHeaderKind verifies mount decode rejects snapshot headers.
func TestParseBinaryMountEnvelopeRejectsWrongHeaderKind(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinarySnapshotEnvelope(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySnapshotEnvelope returned error: %v", parseErr)
	}
	if _, parseErr := runtime2.ParseBinaryMountEnvelope(parsePayload); parseErr == nil {
		parseT.Fatal("expected wrong binary header kind to fail mount decode")
	}
}

// TestParseBinaryUpdateEnvelopeRejectsWrongHeaderKind verifies update decode rejects snapshot headers.
func TestParseBinaryUpdateEnvelopeRejectsWrongHeaderKind(parseT *testing.T) {
	parsePayload, parseErr := runtime2.BuildBinarySnapshotEnvelope(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildBinarySnapshotEnvelope returned error: %v", parseErr)
	}
	if _, parseErr := runtime2.ParseBinaryUpdateEnvelope(parsePayload); parseErr == nil {
		parseT.Fatal("expected wrong binary header kind to fail update decode")
	}
}
