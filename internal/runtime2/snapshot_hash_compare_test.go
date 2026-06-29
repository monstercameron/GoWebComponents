package runtime2_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// buildRuntime2LegacySnapshotFingerprintHash computes snapshot identity using the legacy marshal-based path.
func buildRuntime2LegacySnapshotFingerprintHash(parseEnvelope runtime2.SnapshotEnvelope) ([sha256.Size]byte, error) {
	if parseErr := runtime2.ValidateSnapshotEnvelope(parseEnvelope); parseErr != nil {
		return [sha256.Size]byte{}, parseErr
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		return [sha256.Size]byte{}, fmt.Errorf("runtime2: encode snapshot fingerprint payload: %w", parseErr)
	}
	return sha256.Sum256(parsePayload), nil
}

// TestGetSnapshotFingerprintHashMatchesLegacyMarshalEncoding verifies the optimized snapshot hash path preserves legacy digest outputs.
func TestGetSnapshotFingerprintHashMatchesLegacyMarshalEncoding(parseT *testing.T) {
	getSnapshotEnvelopes := []runtime2.SnapshotEnvelope{
		{
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     3,
			Props: map[string]any{
				"title": "Orders",
				"tick":  3,
			},
		},
		{
			RegionInstanceID: runtime2.RegionInstanceID("region-2"),
			Epoch:            7,
			InputVersion:     11,
			SourceVersion:    9,
			Props: []any{
				map[string]any{"kind": "summary", "count": 5},
				map[string]any{"kind": "detail", "label": "north"},
			},
			Sources: map[string]any{
				"filters": map[string]any{
					"status": "open",
					"limit":  50,
				},
			},
		},
	}
	for parseIndex, getSnapshotEnvelope := range getSnapshotEnvelopes {
		getLegacyHash, parseLegacyErr := buildRuntime2LegacySnapshotFingerprintHash(getSnapshotEnvelope)
		if parseLegacyErr != nil {
			parseT.Fatalf("buildRuntime2LegacySnapshotFingerprintHash(case %d) returned error: %v", parseIndex, parseLegacyErr)
		}
		getCurrentHash, parseCurrentErr := runtime2.GetSnapshotFingerprintHash(getSnapshotEnvelope)
		if parseCurrentErr != nil {
			parseT.Fatalf("GetSnapshotFingerprintHash(case %d) returned error: %v", parseIndex, parseCurrentErr)
		}
		if getLegacyHash != getCurrentHash {
			parseT.Fatalf("expected matching snapshot hash for case %d legacy=%x current=%x", parseIndex, getLegacyHash, getCurrentHash)
		}
	}
}
