package runtime2_test

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// FuzzParseBinarySnapshotEnvelope fuzzes binary snapshot decoding to ensure malformed payloads fail safely and successful decodes round-trip stably.
func FuzzParseBinarySnapshotEnvelope(parseF *testing.F) {
	parseValidPayload, parseErr := runtime2.BuildBinarySnapshotEnvelope(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		SourceVersion:    3,
		Props: map[string]any{
			"title": "Orders",
			"count": 5,
		},
		Sources: map[string]any{
			"status": "healthy",
			"items":  []any{"a", "b"},
		},
	})
	if parseErr != nil {
		parseF.Fatalf("BuildBinarySnapshotEnvelope(valid) returned error: %v", parseErr)
	}
	parseCorruptChecksumPayload := append([]byte(nil), parseValidPayload...)
	if len(parseCorruptChecksumPayload) > 0 {
		parseCorruptChecksumPayload[len(parseCorruptChecksumPayload)-1] ^= 0xFF
	}
	parseWrongKindPayload, parseErr := runtime2.BuildBinaryMountEnvelope(runtime2.BinaryMountEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		SourceIDs:        []string{"status", "items"},
		Snapshot: runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     2,
			SourceVersion:    3,
			Props:            map[string]any{"title": "Orders"},
			Sources: map[string]any{
				"status": "healthy",
				"items":  []any{"a", "b"},
			},
		},
	})
	if parseErr != nil {
		parseF.Fatalf("BuildBinaryMountEnvelope returned error: %v", parseErr)
	}

	parseF.Add(parseValidPayload)
	parseF.Add([]byte{})
	parseF.Add([]byte("GWB1"))
	parseF.Add(parseCorruptChecksumPayload)
	parseF.Add(parseWrongKindPayload)

	parseF.Fuzz(func(parseT *testing.T, parsePayload []byte) {
		parseEnvelope, parseParseErr := runtime2.ParseBinarySnapshotEnvelope(parsePayload)
		if parseParseErr != nil {
			return
		}
		if parseErr := runtime2.ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
			parseT.Fatalf("ValidateSnapshotEnvelope returned error for decoded payload: %v", parseErr)
		}
		parseRoundTripPayload, parseErr := runtime2.BuildBinarySnapshotEnvelope(parseEnvelope)
		if parseErr != nil {
			parseT.Fatalf("BuildBinarySnapshotEnvelope(round-trip) returned error: %v", parseErr)
		}
		parseRoundTripEnvelope, parseErr := runtime2.ParseBinarySnapshotEnvelope(parseRoundTripPayload)
		if parseErr != nil {
			parseT.Fatalf("ParseBinarySnapshotEnvelope(round-trip) returned error: %v", parseErr)
		}
		if !reflect.DeepEqual(parseRoundTripEnvelope, parseEnvelope) {
			parseT.Fatalf("expected round-trip envelope %#v, got %#v", parseEnvelope, parseRoundTripEnvelope)
		}
	})
}
