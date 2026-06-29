package runtime2_test

import "testing"

import "github.com/monstercameron/GoWebComponents/v4/internal/runtime2"

// TestBuildSnapshotEnvelopeAcceptsMinimalValidEnvelope verifies the snapshot envelope contract accepts the minimal valid shape.
func TestBuildSnapshotEnvelopeAcceptsMinimalValidEnvelope(parseT *testing.T) {
	parseEnvelope, parseErr := runtime2.BuildSnapshotEnvelope(
		runtime2.RegionInstanceID("region-1"),
		1,
		1,
		map[string]any{"title": "Orders"},
		[]string{"status"},
		map[string]any{"status": "healthy", "ignored": "skip"},
		map[string]uint64{"status": 7, "ignored": 9},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope returned error: %v", parseErr)
	}
	if parseEnvelope.InputVersion != 1 {
		parseT.Fatalf("expected input version 1, got %d", parseEnvelope.InputVersion)
	}
	if parseEnvelope.SourceVersion != 7 {
		parseT.Fatalf("expected source version 7, got %d", parseEnvelope.SourceVersion)
	}
	if _, parseHasIgnored := parseEnvelope.Sources["ignored"]; parseHasIgnored {
		parseT.Fatal("expected undeclared source to be excluded")
	}
}

// TestValidateSnapshotEnvelopeRejectsMissingInputVersion verifies input versions are required.
func TestValidateSnapshotEnvelopeRejectsMissingInputVersion(parseT *testing.T) {
	parseErr := runtime2.ValidateSnapshotEnvelope(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
	})
	if parseErr == nil {
		parseT.Fatal("expected missing input version to fail")
	}
}

// TestValidateSnapshotEnvelopeRejectsMissingEpoch verifies epochs are required.
func TestValidateSnapshotEnvelopeRejectsMissingEpoch(parseT *testing.T) {
	parseErr := runtime2.ValidateSnapshotEnvelope(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		InputVersion:     1,
	})
	if parseErr == nil {
		parseT.Fatal("expected missing epoch to fail")
	}
}

// TestValidateMonotonicInputVersionAcceptsLargerAndEqualVersions verifies the version contract accepts forward progress and duplicate delivery.
func TestValidateMonotonicInputVersionAcceptsLargerAndEqualVersions(parseT *testing.T) {
	if parseErr := runtime2.ValidateMonotonicInputVersion(1, 2); parseErr != nil {
		parseT.Fatalf("expected increasing input version to pass, got %v", parseErr)
	}
	if parseErr := runtime2.ValidateMonotonicInputVersion(2, 2); parseErr != nil {
		parseT.Fatalf("expected duplicate input version to pass, got %v", parseErr)
	}
}

// TestValidateMonotonicInputVersionRejectsOlderVersion verifies backward version movement fails.
func TestValidateMonotonicInputVersionRejectsOlderVersion(parseT *testing.T) {
	if parseErr := runtime2.ValidateMonotonicInputVersion(2, 1); parseErr == nil {
		parseT.Fatal("expected older input version to fail")
	}
}

// TestBuildSourceSnapshotRejectsMissingDeclaredSource verifies all declared source IDs must be present.
func TestBuildSourceSnapshotRejectsMissingDeclaredSource(parseT *testing.T) {
	if _, parseErr := runtime2.BuildSourceSnapshot([]string{"status"}, map[string]any{}); parseErr == nil {
		parseT.Fatal("expected missing declared source value to fail")
	}
}

// TestValidateSourceSnapshotConsistencyAcceptsOneVersion verifies one coherent source version passes.
func TestValidateSourceSnapshotConsistencyAcceptsOneVersion(parseT *testing.T) {
	parseSourceVersion, parseErr := runtime2.ValidateSourceSnapshotConsistency([]string{"a", "b"}, map[string]uint64{"a": 9, "b": 9})
	if parseErr != nil {
		parseT.Fatalf("ValidateSourceSnapshotConsistency returned error: %v", parseErr)
	}
	if parseSourceVersion != 9 {
		parseT.Fatalf("expected source version 9, got %d", parseSourceVersion)
	}
}

// TestValidateSourceSnapshotConsistencyRejectsMixedVersions verifies inconsistent source versions fail.
func TestValidateSourceSnapshotConsistencyRejectsMixedVersions(parseT *testing.T) {
	if _, parseErr := runtime2.ValidateSourceSnapshotConsistency([]string{"a", "b"}, map[string]uint64{"a": 9, "b": 10}); parseErr == nil {
		parseT.Fatal("expected mixed source versions to fail")
	}
}

// TestValidateSourceSnapshotConsistencyAcceptsEmptyPayload verifies empty source sets behave consistently.
func TestValidateSourceSnapshotConsistencyAcceptsEmptyPayload(parseT *testing.T) {
	parseSourceVersion, parseErr := runtime2.ValidateSourceSnapshotConsistency(nil, nil)
	if parseErr != nil {
		parseT.Fatalf("ValidateSourceSnapshotConsistency returned error: %v", parseErr)
	}
	if parseSourceVersion != 0 {
		parseT.Fatalf("expected zero source version, got %d", parseSourceVersion)
	}
}

// TestGetSnapshotFingerprintKeepsEquivalentPayloadsStable verifies equivalent snapshots keep the same fingerprint.
func TestGetSnapshotFingerprintKeepsEquivalentPayloadsStable(parseT *testing.T) {
	parseEnvelopeA := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		SourceVersion:    7,
		Props:            map[string]any{"a": 1, "b": 2},
		Sources:          map[string]any{"status": "healthy", "count": 3},
	}
	parseEnvelopeB := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		SourceVersion:    7,
		Props:            map[string]any{"b": 2, "a": 1},
		Sources:          map[string]any{"count": 3, "status": "healthy"},
	}
	parseFingerprintA, parseErr := runtime2.GetSnapshotFingerprint(parseEnvelopeA)
	if parseErr != nil {
		parseT.Fatalf("GetSnapshotFingerprint A returned error: %v", parseErr)
	}
	parseFingerprintB, parseErr := runtime2.GetSnapshotFingerprint(parseEnvelopeB)
	if parseErr != nil {
		parseT.Fatalf("GetSnapshotFingerprint B returned error: %v", parseErr)
	}
	if parseFingerprintA != parseFingerprintB {
		parseT.Fatalf("expected stable fingerprint, first=%q second=%q", parseFingerprintA, parseFingerprintB)
	}
}

// TestGetSnapshotFingerprintChangesForDifferentPayloads verifies materially different snapshots change fingerprint.
func TestGetSnapshotFingerprintChangesForDifferentPayloads(parseT *testing.T) {
	parseEnvelopeA := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		Props:            map[string]any{"count": 1},
	}
	parseEnvelopeB := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		Props:            map[string]any{"count": 2},
	}
	parseFingerprintA, parseErr := runtime2.GetSnapshotFingerprint(parseEnvelopeA)
	if parseErr != nil {
		parseT.Fatalf("GetSnapshotFingerprint A returned error: %v", parseErr)
	}
	parseFingerprintB, parseErr := runtime2.GetSnapshotFingerprint(parseEnvelopeB)
	if parseErr != nil {
		parseT.Fatalf("GetSnapshotFingerprint B returned error: %v", parseErr)
	}
	if parseFingerprintA == parseFingerprintB {
		parseT.Fatalf("expected different snapshots to change fingerprint, got %q", parseFingerprintA)
	}
}
