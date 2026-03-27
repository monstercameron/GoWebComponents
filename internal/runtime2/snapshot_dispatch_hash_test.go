package runtime2

import (
	"math"
	"testing"
)

// TestBuildSnapshotDispatchHashIgnoresInputVersion verifies dispatch hashing ignores input-version churn for no-change scheduling.
func TestBuildSnapshotDispatchHashIgnoresInputVersion(parseT *testing.T) {
	getSnapshotEnvelope := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            2,
		InputVersion:     3,
		SourceVersion:    11,
		Props: map[string]any{
			"title": "Orders",
			"tick":  7,
		},
		Sources: map[string]any{
			"filters": map[string]any{
				"status": "open",
			},
		},
	}
	getHashFirst, parseFirstErr := buildSnapshotDispatchHash(getSnapshotEnvelope)
	if parseFirstErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(first) returned error: %v", parseFirstErr)
	}
	getSnapshotEnvelope.InputVersion = 99
	getHashSecond, parseSecondErr := buildSnapshotDispatchHash(getSnapshotEnvelope)
	if parseSecondErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(second) returned error: %v", parseSecondErr)
	}
	if getHashFirst != getHashSecond {
		parseT.Fatal("expected dispatch hash to ignore input-version changes")
	}
}

// TestBuildSnapshotDispatchHashCanonicalMapOrder verifies dispatch hashing is stable across map iteration order differences.
func TestBuildSnapshotDispatchHashCanonicalMapOrder(parseT *testing.T) {
	getSnapshotEnvelopeLeft := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-2"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"a": 1,
			"b": 2,
			"c": 3,
		},
		Sources: map[string]any{
			"zeta": 4,
			"beta": 5,
		},
	}
	getSnapshotEnvelopeRight := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-2"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"c": 3,
			"a": 1,
			"b": 2,
		},
		Sources: map[string]any{
			"beta": 5,
			"zeta": 4,
		},
	}
	getHashLeft, parseLeftErr := buildSnapshotDispatchHash(getSnapshotEnvelopeLeft)
	if parseLeftErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(left) returned error: %v", parseLeftErr)
	}
	getHashRight, parseRightErr := buildSnapshotDispatchHash(getSnapshotEnvelopeRight)
	if parseRightErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(right) returned error: %v", parseRightErr)
	}
	if getHashLeft != getHashRight {
		parseT.Fatal("expected dispatch hash to be stable for equivalent map contents")
	}
}

// TestBuildSnapshotDispatchHashRejectsNonFiniteNumber verifies dispatch hashing rejects NaN and infinities.
func TestBuildSnapshotDispatchHashRejectsNonFiniteNumber(parseT *testing.T) {
	_, parseErr := buildSnapshotDispatchHash(SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-3"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"value": math.NaN(),
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected non-finite dispatch hash input to fail")
	}
}

// TestBuildSnapshotDispatchHashCanonicalPairWithEmptyKey verifies two-key map fast path stability when one key is an empty string.
func TestBuildSnapshotDispatchHashCanonicalPairWithEmptyKey(parseT *testing.T) {
	getSnapshotEnvelopeLeft := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-pair-empty-key"),
		Epoch:            1,
		InputVersion:     1,
		Props: map[string]any{
			"":  1,
			"k": 2,
		},
		Sources: map[string]any{
			"":  "a",
			"s": "b",
		},
	}
	getSnapshotEnvelopeRight := SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-pair-empty-key"),
		Epoch:            1,
		InputVersion:     7,
		Props: map[string]any{
			"k": 2,
			"":  1,
		},
		Sources: map[string]any{
			"s": "b",
			"":  "a",
		},
	}
	getHashLeft, parseLeftErr := buildSnapshotDispatchHash(getSnapshotEnvelopeLeft)
	if parseLeftErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(left) returned error: %v", parseLeftErr)
	}
	getHashRight, parseRightErr := buildSnapshotDispatchHash(getSnapshotEnvelopeRight)
	if parseRightErr != nil {
		parseT.Fatalf("buildSnapshotDispatchHash(right) returned error: %v", parseRightErr)
	}
	if getHashLeft != getHashRight {
		parseT.Fatal("expected dispatch hash to remain stable for two-key maps including an empty key")
	}
}
