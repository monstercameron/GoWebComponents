package runtime2

import (
	"errors"
	"testing"
)

// TestSnapshotValidationAndBuildersRejectInvalidInputs verifies snapshot helpers reject invalid source identifiers, non-serializable props, missing source versions, and invalid envelopes.
func TestSnapshotValidationAndBuildersRejectInvalidInputs(parseT *testing.T) {
	if _, parseErr := BuildSourceSnapshot([]string{" status"}, map[string]any{"status": "healthy"}); parseErr == nil {
		parseT.Fatal("expected invalid declared source ID to fail snapshot build")
	}
	if _, parseErr := ValidateSourceSnapshotConsistency([]string{" status"}, map[string]uint64{"status": 7}); parseErr == nil {
		parseT.Fatal("expected invalid declared source ID to fail source-version validation")
	}
	if parseErr := ValidateSnapshotEnvelope(SnapshotEnvelope{
		RegionInstanceID: RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
		Props:            func() {},
	}); parseErr == nil {
		parseT.Fatal("expected non-serializable snapshot props to fail envelope validation")
	}
	if _, parseErr := GetSnapshotFingerprint(SnapshotEnvelope{}); parseErr == nil {
		parseT.Fatal("expected invalid snapshot fingerprint request to fail")
	}
	if _, parseErr := GetSnapshotFingerprintHash(SnapshotEnvelope{}); parseErr == nil {
		parseT.Fatal("expected invalid snapshot fingerprint hash request to fail")
	}
	if _, _, parseErr := buildSnapshotSourceValuesAndVersion(
		[]string{"status"},
		map[string]any{"status": "healthy"},
		map[string]uint64{},
	); parseErr == nil {
		parseT.Fatal("expected missing declared source version to fail snapshot source build")
	}
	if _, parseErr := buildSnapshotEnvelopeFromNormalizedSourceSnapshotWithoutValidation(
		RegionInstanceID("region-1"),
		1,
		1,
		nil,
		[]string{"status"},
		map[string]any{"status": "healthy"},
		map[string]uint64{},
	); parseErr == nil {
		parseT.Fatal("expected missing normalized source version to fail snapshot envelope build")
	}
}

// TestSnapshotFingerprintHelpersPropagateHasherWriteErrors verifies snapshot fingerprint helper writers surface downstream hasher write errors.
func TestSnapshotFingerprintHelpersPropagateHasherWriteErrors(parseT *testing.T) {
	parseLiteralErr := errors.New("literal write boom")
	if parseErr := writeSnapshotFingerprintLiteral(buildJSONHashTestHasher(parseLiteralErr), []byte("{")); !errors.Is(parseErr, parseLiteralErr) {
		parseT.Fatalf("expected literal write error propagation, got %v", parseErr)
	}

	parseUintErr := errors.New("uint write boom")
	if parseErr := writeSnapshotFingerprintUint(buildJSONHashTestHasher(parseUintErr), 7); !errors.Is(parseErr, parseUintErr) {
		parseT.Fatalf("expected uint write error propagation, got %v", parseErr)
	}

	parseQuotedErr := errors.New("quoted write boom")
	if parseErr := writeSnapshotFingerprintQuotedString(buildJSONHashTestHasher(parseQuotedErr), "region-1"); !errors.Is(parseErr, parseQuotedErr) {
		parseT.Fatalf("expected quoted-string write error propagation, got %v", parseErr)
	}
}
