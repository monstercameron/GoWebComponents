package runtime2

import "testing"

// TestHandlePatchIdempotencyDropsStaleLowerVersion pins the monotonicity guard: a
// never-seen patch version below the epoch's high-water mark (a stale/reordered
// delivery) is dropped (apply=false, no error) instead of applying an older
// cumulative diff over newer state.
func TestHandlePatchIdempotencyDropsStaleLowerVersion(parseT *testing.T) {
	parseTracker := BuildPatchIdempotencyTracker()

	if parseApply, parseErr := parseTracker.HandlePatchIdempotency("region-1", 1, 3, "patch-v3"); parseErr != nil || !parseApply {
		parseT.Fatalf("v3 should apply: apply=%v err=%v", parseApply, parseErr)
	}
	if parseApply, parseErr := parseTracker.HandlePatchIdempotency("region-1", 1, 5, "patch-v5"); parseErr != nil || !parseApply {
		parseT.Fatalf("v5 should apply (5 > 3): apply=%v err=%v", parseApply, parseErr)
	}
	// A never-seen version 4 arriving after 5 is stale → dropped, no error.
	if parseApply, parseErr := parseTracker.HandlePatchIdempotency("region-1", 1, 4, "patch-v4-late"); parseErr != nil || parseApply {
		parseT.Fatalf("stale v4 (< max 5) must be dropped: apply=%v err=%v", parseApply, parseErr)
	}
	// Version 2 (well below max) is also dropped.
	if parseApply, parseErr := parseTracker.HandlePatchIdempotency("region-1", 1, 2, "patch-v2-late"); parseErr != nil || parseApply {
		parseT.Fatalf("stale v2 (< max 5) must be dropped: apply=%v err=%v", parseApply, parseErr)
	}
	// Forward progress still applies.
	if parseApply, parseErr := parseTracker.HandlePatchIdempotency("region-1", 1, 6, "patch-v6"); parseErr != nil || !parseApply {
		parseT.Fatalf("v6 should apply (6 > 5): apply=%v err=%v", parseApply, parseErr)
	}
}

// TestHandlePatchIdempotencyEpochResetClearsHighWaterMark pins that a new epoch
// resets the monotonicity high-water mark, so a lower version in the new epoch is
// applied (a re-mount / re-render legitimately restarts versioning).
func TestHandlePatchIdempotencyEpochResetClearsHighWaterMark(parseT *testing.T) {
	parseTracker := BuildPatchIdempotencyTracker()

	if parseApply, parseErr := parseTracker.HandlePatchIdempotency("region-1", 1, 9, "patch-e1-v9"); parseErr != nil || !parseApply {
		parseT.Fatalf("epoch1 v9 should apply: apply=%v err=%v", parseApply, parseErr)
	}
	// New epoch: a lower version must apply (high-water mark reset).
	if parseApply, parseErr := parseTracker.HandlePatchIdempotency("region-1", 2, 1, "patch-e2-v1"); parseErr != nil || !parseApply {
		parseT.Fatalf("epoch2 v1 should apply after epoch reset: apply=%v err=%v", parseApply, parseErr)
	}
}
