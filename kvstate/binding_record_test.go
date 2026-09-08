package kvstate

import "testing"

// TestBindingRecordRejectsDelayedVersions covers both live data and tombstones.
func TestBindingRecordRejectsDelayedVersions(parseTest *testing.T) {
	for _, parseResolver := range []ConflictResolver{Versioned{}, LastWriteWins{}} {
		for _, parseVersion := range []int64{0, 1, 7} {
			if shouldApplyBindingRecord(8, Record{Version: parseVersion}, parseResolver) {
				parseTest.Fatalf("accepted stale version %d", parseVersion)
			}
		}
		if !shouldApplyBindingRecord(8, Record{Version: 9}, parseResolver) {
			parseTest.Fatal("rejected newer version")
		}
	}
}
