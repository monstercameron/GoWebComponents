package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionSnapshotFingerprintReusesStableFingerprint verifies host adapters reuse stable snapshot fingerprints for unchanged update payloads.
func TestHandleHostRegionSnapshotFingerprintReusesStableFingerprint(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	getSnapshotEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            3,
		InputVersion:     7,
		Props:            map[string]any{"title": "Orders"},
		Sources:          map[string]any{"count": 5},
	}
	getFirstFingerprintResult, parseErr := buildHostRegionAdapter.HandleHostRegionSnapshotFingerprint(getSnapshotEnvelope)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionSnapshotFingerprint(first) returned error: %v", parseErr)
	}
	if getFirstFingerprintResult.HasNoChange {
		parseT.Fatal("expected first snapshot fingerprint to be treated as changed")
	}
	getSecondFingerprintResult, parseErr := buildHostRegionAdapter.HandleHostRegionSnapshotFingerprint(getSnapshotEnvelope)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionSnapshotFingerprint(second) returned error: %v", parseErr)
	}
	if !getSecondFingerprintResult.HasNoChange {
		parseT.Fatal("expected identical snapshot fingerprint to be reused as no-change")
	}
	if getFirstFingerprintResult.GetSnapshotFingerprint != getSecondFingerprintResult.GetSnapshotFingerprint {
		parseT.Fatalf(
			"expected stable fingerprint reuse for identical snapshot, first=%q second=%q",
			getFirstFingerprintResult.GetSnapshotFingerprint,
			getSecondFingerprintResult.GetSnapshotFingerprint,
		)
	}
}

// TestHandleHostRegionSnapshotFingerprintRejectsWrongRegion verifies snapshot fingerprint handling is scoped to one adapter-owned region.
func TestHandleHostRegionSnapshotFingerprintRejectsWrongRegion(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionSnapshotFingerprint(runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-2"),
		Epoch:            1,
		InputVersion:     1,
	})
	if parseErr == nil {
		parseT.Fatal("expected snapshot fingerprint for wrong region to fail")
	}
}
