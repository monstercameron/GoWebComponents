package runtime2

import "testing"

// TestParsePatchStreamInvalidBodyDoesNotPoisonIdempotency pins the #72 fix: a
// patch with a valid header/identity but an INVALID body must NOT advance the
// idempotency tracker. Previously the tracker was committed BEFORE the body was
// validated, so a patch that failed op decoding still recorded its version +
// identity — and a later corrected patch at the same version was then rejected
// as a conflict, permanently wedging the region. The tracker is now committed
// only after the whole body validates.
func TestParsePatchStreamInvalidBodyDoesNotPoisonIdempotency(parseT *testing.T) {
	parseTracker := BuildPatchIdempotencyTracker()

	// Valid header + identity, but op 0 carries an invalid op code, so body
	// validation fails.
	parseInvalid := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-1",
			Epoch:           1,
			InputVersion:    1,
			PatchVersion:    5,
		},
		GetPatchIdentity: "patch-invalid",
		GetOps:           []PatchStreamOpRaw{{GetOpCode: 255}},
	}
	if _, _, parseErr := ParsePatchStreamTransaction(parseInvalid, "region-1", 1, map[uint64]struct{}{}, map[uint64]uint32{}, parseTracker); parseErr == nil {
		parseT.Fatal("expected invalid patch body to return an error")
	}

	// The failed patch must not have poisoned version tracking: a corrected patch
	// at the SAME version with a DIFFERENT identity must still be accepted, not
	// rejected as a version/identity conflict.
	parseShouldApply, parseCheckErr := parseTracker.CheckPatchIdempotency("region-1", 1, 5, "patch-corrected")
	if parseCheckErr != nil {
		parseT.Fatalf("corrected patch at version 5 wrongly rejected as conflict: %v", parseCheckErr)
	}
	if !parseShouldApply {
		parseT.Fatal("corrected patch at version 5 should apply, got skip (tracker was poisoned)")
	}
}

// TestParsePatchStreamRejectsStaleEpochBeforeNodeIndexResolution pins the #72
// node-index-reuse defense: a patch from a PRE-restart epoch, delivered after the
// region restarted to a higher epoch, must be rejected on the epoch mismatch
// BEFORE any op/node-index decoding. This is what makes worker-restart node-index
// reuse safe — a stale patch that references node indices the new epoch may have
// reused for different nodes never reaches DOM application, and never touches the
// (epoch-scoped) idempotency tracker.
func TestParsePatchStreamRejectsStaleEpochBeforeNodeIndexResolution(parseT *testing.T) {
	parseTracker := BuildPatchIdempotencyTracker()

	// Region restarted to epoch 2; this patch is still stamped with the old epoch 1.
	parseStale := PatchStreamRaw{
		GetHeader: PatchStreamHeaderRaw{
			ProtocolVersion: PatchStreamProtocolVersion,
			RegionID:        "region-1",
			Epoch:           1,
			InputVersion:    1,
			PatchVersion:    9,
		},
		GetPatchIdentity: "stale-epoch-patch",
	}
	if _, _, parseErr := ParsePatchStreamTransaction(parseStale, "region-1", 2, map[uint64]struct{}{}, map[uint64]uint32{}, parseTracker); parseErr == nil {
		parseT.Fatal("expected a stale-epoch patch to be rejected before node-index resolution")
	}
	// The epoch check runs before the idempotency tracker, so a fresh epoch-2 patch
	// at the same version is unaffected.
	if parseApply, parseErr := parseTracker.CheckPatchIdempotency("region-1", 2, 9, "fresh-epoch2"); parseErr != nil || !parseApply {
		parseT.Fatalf("epoch-2 patch should be unaffected by the rejected stale patch: apply=%v err=%v", parseApply, parseErr)
	}
}

// TestCheckThenCommitPatchIdempotencyBehavesLikeHandle verifies the split
// Check/Commit path matches the check-and-commit HandlePatchIdempotency for the
// duplicate, conflict, and forward-progress cases.
func TestCheckThenCommitPatchIdempotencyBehavesLikeHandle(parseT *testing.T) {
	parseTracker := BuildPatchIdempotencyTracker()

	// New version: check says apply, commit records it.
	if parseApply, parseErr := parseTracker.CheckPatchIdempotency("r", 1, 3, "id-3"); parseErr != nil || !parseApply {
		parseT.Fatalf("v3 check: apply=%v err=%v", parseApply, parseErr)
	}
	if parseErr := parseTracker.CommitPatchIdempotency("r", 1, 3, "id-3"); parseErr != nil {
		parseT.Fatalf("v3 commit err=%v", parseErr)
	}
	// Exact duplicate: skip.
	if parseApply, parseErr := parseTracker.CheckPatchIdempotency("r", 1, 3, "id-3"); parseErr != nil || parseApply {
		parseT.Fatalf("duplicate v3 should skip: apply=%v err=%v", parseApply, parseErr)
	}
	// Conflicting identity at a recorded version: error.
	if _, parseErr := parseTracker.CheckPatchIdempotency("r", 1, 3, "id-other"); parseErr == nil {
		parseT.Fatal("conflicting identity at v3 should error")
	}
	// Forward progress applies.
	if parseApply, parseErr := parseTracker.CheckPatchIdempotency("r", 1, 4, "id-4"); parseErr != nil || !parseApply {
		parseT.Fatalf("v4 check: apply=%v err=%v", parseApply, parseErr)
	}
}
