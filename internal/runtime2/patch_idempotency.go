package runtime2

import "fmt"

type patchIdempotencyState struct {
	getEpochByRegionID          uint64
	storeIdentityByPatchVersion map[uint64]string
	// getMaxPatchVersion is the highest patch version applied in the current epoch.
	// Patch versions are monotonic sequence numbers, so a never-seen version below
	// this maximum is a stale/reordered delivery (common across the async worker
	// bridge) whose older cumulative diff must NOT be applied over newer state.
	getMaxPatchVersion uint64
}

// PatchIdempotencyTracker tracks duplicate and conflicting patch identities per region.
type PatchIdempotencyTracker struct {
	storePatchStateByRegionID map[string]patchIdempotencyState
}

// BuildPatchIdempotencyTracker creates a patch idempotency tracker.
func BuildPatchIdempotencyTracker() *PatchIdempotencyTracker {
	return &PatchIdempotencyTracker{
		storePatchStateByRegionID: make(map[string]patchIdempotencyState),
	}
}

// HandlePatchIdempotency applies idempotency rules and reports whether a patch
// should be applied, recording the patch as seen in one call. It is the
// check-and-commit convenience used where a patch body has no separate
// validation step; callers that validate a patch body AFTER the idempotency
// decision must instead use CheckPatchIdempotency (before) and
// CommitPatchIdempotency (only after validation succeeds) so an invalid patch
// cannot poison version tracking — see #72.
func (parseTracker *PatchIdempotencyTracker) HandlePatchIdempotency(parseRegionID string, parseEpoch uint64, parsePatchVersion uint64, parsePatchIdentity string) (bool, error) {
	parseShouldApply, parseCheckErr := parseTracker.CheckPatchIdempotency(parseRegionID, parseEpoch, parsePatchVersion, parsePatchIdentity)
	if parseCheckErr != nil || !parseShouldApply {
		return parseShouldApply, parseCheckErr
	}
	if parseCommitErr := parseTracker.CommitPatchIdempotency(parseRegionID, parseEpoch, parsePatchVersion, parsePatchIdentity); parseCommitErr != nil {
		return false, parseCommitErr
	}
	return true, nil
}

// CheckPatchIdempotency reports whether a patch should be applied WITHOUT
// mutating tracker state. It returns false (skip, no error) for an exact
// duplicate or a stale below-high-water-mark version, an error for a
// version/identity conflict, and true when the patch is new and in-order. A
// caller MUST call CommitPatchIdempotency only after the patch body has fully
// validated, so a patch that fails body validation never advances version
// state and cannot wedge the region on a later corrected patch (#72).
func (parseTracker *PatchIdempotencyTracker) CheckPatchIdempotency(parseRegionID string, parseEpoch uint64, parsePatchVersion uint64, parsePatchIdentity string) (bool, error) {
	if parseTracker == nil {
		return false, fmt.Errorf("runtime2: patch idempotency tracker is nil")
	}
	if parseRegionID == "" {
		return false, fmt.Errorf("runtime2: patch idempotency region id is required")
	}
	if parsePatchIdentity == "" {
		return false, fmt.Errorf("runtime2: patch idempotency identity is required")
	}
	getState, hasState := parseTracker.storePatchStateByRegionID[parseRegionID]
	if !hasState || getState.getEpochByRegionID != parseEpoch {
		// A fresh region or a new epoch has recorded nothing yet, so the patch is
		// new and in-order.
		return true, nil
	}
	if getIdentity, hasIdentity := getState.storeIdentityByPatchVersion[parsePatchVersion]; hasIdentity {
		if getIdentity == parsePatchIdentity {
			return false, nil
		}
		return false, fmt.Errorf("runtime2: patch version %d for region %q already recorded as identity %q (received %q)", parsePatchVersion, parseRegionID, getIdentity, parsePatchIdentity)
	}
	// Monotonicity guard: a never-seen version below the epoch's high-water mark is a
	// stale/reordered patch (e.g. delayed worker-bridge delivery). Applying its older
	// cumulative diff over newer state would regress the DOM, so drop it idempotently
	// (skip, no error) rather than apply out of order.
	if parsePatchVersion < getState.getMaxPatchVersion {
		return false, nil
	}
	return true, nil
}

// CommitPatchIdempotency records an applied patch's identity and advances the
// epoch high-water mark. Call it ONLY after CheckPatchIdempotency returned true
// AND the patch body validated. Repeating it for the same (version, identity) is
// harmless.
func (parseTracker *PatchIdempotencyTracker) CommitPatchIdempotency(parseRegionID string, parseEpoch uint64, parsePatchVersion uint64, parsePatchIdentity string) error {
	if parseTracker == nil {
		return fmt.Errorf("runtime2: patch idempotency tracker is nil")
	}
	if parseRegionID == "" {
		return fmt.Errorf("runtime2: patch idempotency region id is required")
	}
	if parsePatchIdentity == "" {
		return fmt.Errorf("runtime2: patch idempotency identity is required")
	}
	getState, hasState := parseTracker.storePatchStateByRegionID[parseRegionID]
	if !hasState || getState.getEpochByRegionID != parseEpoch {
		getState = patchIdempotencyState{
			getEpochByRegionID:          parseEpoch,
			storeIdentityByPatchVersion: make(map[uint64]string),
		}
	}
	getState.storeIdentityByPatchVersion[parsePatchVersion] = parsePatchIdentity
	if parsePatchVersion > getState.getMaxPatchVersion {
		getState.getMaxPatchVersion = parsePatchVersion
	}
	parseTracker.storePatchStateByRegionID[parseRegionID] = getState
	return nil
}
