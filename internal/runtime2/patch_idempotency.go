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

// HandlePatchIdempotency applies idempotency rules and reports whether a patch should be applied.
func (parseTracker *PatchIdempotencyTracker) HandlePatchIdempotency(parseRegionID string, parseEpoch uint64, parsePatchVersion uint64, parsePatchIdentity string) (bool, error) {
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
		getState = patchIdempotencyState{
			getEpochByRegionID:          parseEpoch,
			storeIdentityByPatchVersion: make(map[uint64]string),
		}
	}
	if getIdentity, hasIdentity := getState.storeIdentityByPatchVersion[parsePatchVersion]; hasIdentity {
		if getIdentity == parsePatchIdentity {
			parseTracker.storePatchStateByRegionID[parseRegionID] = getState
			return false, nil
		}
		return false, fmt.Errorf("runtime2: patch version %d for region %q already recorded as identity %q (received %q)", parsePatchVersion, parseRegionID, getIdentity, parsePatchIdentity)
	}
	// Monotonicity guard: a never-seen version below the epoch's high-water mark is a
	// stale/reordered patch (e.g. delayed worker-bridge delivery). Applying its older
	// cumulative diff over newer state would regress the DOM, so drop it idempotently
	// (skip, no error) rather than apply out of order.
	if parsePatchVersion < getState.getMaxPatchVersion {
		parseTracker.storePatchStateByRegionID[parseRegionID] = getState
		return false, nil
	}
	getState.storeIdentityByPatchVersion[parsePatchVersion] = parsePatchIdentity
	getState.getMaxPatchVersion = parsePatchVersion
	parseTracker.storePatchStateByRegionID[parseRegionID] = getState
	return true, nil
}
