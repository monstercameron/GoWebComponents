package runtime

import "time"

// Lane queues with expiration (v5 P2.2).
//
// Before this, "priority" was two scalars — pendingLane and currentLane — and a
// single pending pass. Lanes coalesced by min(), so an update marked background
// rendered inside whatever pass happened to be running, at that pass's
// priority. Nothing was ever actually deferred; low-priority work was
// *promoted* (T3).
//
// The model here is deliberately narrower than React's full lane machinery,
// because the accept criteria are narrow: background work must not run inside
// an input-lane pass, and a deferred lane must still complete within a bounded
// time. That needs three things — a record of which lanes have pending work,
// a per-fiber lane so a pass can tell what belongs to it, and a deadline per
// lane so deferral cannot become starvation.
//
// Deferral reuses the reconciler's existing "clone through" tier rather than
// skipping the walk. A deferred fiber keeps its dirty flag and does NOT
// re-render, but its subtree is still traversed, so higher-priority work below
// it renders normally. Skipping the walk instead would strand descendants
// behind a low-priority ancestor.
//
// Allocation-free by construction: lane state is a fixed-size array indexed by
// lane, never a map, because this sits on the path that reconciles a stable
// list in 3.3µs with zero allocations.

// laneCount bounds the fixed-size lane arrays. Lanes are 1-based
// (UpdateLaneSync == 1), so index 0 is unused.
const laneCount = int(UpdateLaneBackground) + 1

// laneExpiryMs is how long a lane may sit deferred before it is promoted into
// the next pass regardless of priority.
//
// Deferral without expiry is starvation: under sustained input a transition
// would never complete. The values are deliberately far apart — a transition is
// user-visible and must land quickly, background work can wait.
var laneExpiryMs = [laneCount]int64{
	UpdateLaneSync:       0, // never deferred
	UpdateLaneInput:      0, // never deferred
	UpdateLaneDefault:    0, // never deferred
	UpdateLaneTransition: 500,
	UpdateLaneBackground: 2000,
}

// laneState tracks pending work per lane without allocating.
type laneState struct {
	// pending marks lanes with work waiting.
	pending [laneCount]bool
	// firstMarkedAt records when each lane's oldest pending work arrived, which
	// is what the expiry check measures against.
	firstMarkedAt [laneCount]time.Time
	// deferredThisPass counts fibers a pass declined to render because they
	// belonged to a lower lane. Non-zero means a follow-up pass is required.
	deferredThisPass int
}

// markLanePending records pending work for a lane, stamping its arrival the
// first time so expiry measures the oldest wait rather than the newest.
func (parseState *laneState) markLanePending(parseLane UpdateLane, parseNow time.Time) {
	parseIndex := int(normalizeUpdateLane(parseLane))
	if parseIndex <= 0 || parseIndex >= laneCount {
		return
	}
	if !parseState.pending[parseIndex] {
		parseState.pending[parseIndex] = true
		parseState.firstMarkedAt[parseIndex] = parseNow
	}
}

// clearLanePending marks a lane's work as consumed.
func (parseState *laneState) clearLanePending(parseLane UpdateLane) {
	parseIndex := int(normalizeUpdateLane(parseLane))
	if parseIndex <= 0 || parseIndex >= laneCount {
		return
	}
	parseState.pending[parseIndex] = false
	parseState.firstMarkedAt[parseIndex] = time.Time{}
}

// isLaneExpired reports whether a lane has waited past its deadline and must be
// admitted to the next pass regardless of its priority.
func (parseState *laneState) isLaneExpired(parseLane UpdateLane, parseNow time.Time) bool {
	parseIndex := int(normalizeUpdateLane(parseLane))
	if parseIndex <= 0 || parseIndex >= laneCount {
		return false
	}
	if !parseState.pending[parseIndex] {
		return false
	}
	parseExpiryMs := laneExpiryMs[parseIndex]
	if parseExpiryMs <= 0 {
		return true // lanes with no deferral deadline are always admitted
	}
	return parseNow.Sub(parseState.firstMarkedAt[parseIndex]).Milliseconds() >= parseExpiryMs
}

// highestPendingLane returns the most urgent lane with pending work, or 0.
func (parseState *laneState) highestPendingLane() UpdateLane {
	for parseIndex := 1; parseIndex < laneCount; parseIndex++ {
		if parseState.pending[parseIndex] {
			return UpdateLane(parseIndex)
		}
	}
	return 0
}

// laneAdmitsFiber decides whether a pass running at parsePassLane should render
// a fiber marked at parseFiberLane.
//
// Admission rules, in order:
//   - a fiber with no recorded lane is legacy work and always renders, so
//     nothing regresses for callers that never set a lane
//   - equal or more urgent work always renders
//   - less urgent work renders only once its lane has expired
func (parseRt *Runtime) laneAdmitsFiber(parsePassLane UpdateLane, parseFiberLane UpdateLane, parseNow time.Time) bool {
	if !parseRt.laneQueuesEnabled() {
		return true
	}
	if parseFiberLane == 0 || parsePassLane == 0 {
		return true
	}
	if parseFiberLane <= parsePassLane {
		return true
	}
	// A lane with no deadline is not deferrable at all. Sync, input, and
	// default must render wherever they are found; only transition and
	// background can wait. Checking the configured deadline rather than
	// pending state matters, because a non-deferrable lane may legitimately
	// have no pending entry recorded.
	parseIndex := int(normalizeUpdateLane(parseFiberLane))
	if parseIndex > 0 && parseIndex < laneCount && laneExpiryMs[parseIndex] <= 0 {
		return true
	}
	return parseRt.schedulerState.lanes.isLaneExpired(parseFiberLane, parseNow)
}

// laneQueuesEnabled reports whether per-lane deferral is active (R2: off by
// default until its acceptance test passes).
func (parseRt *Runtime) laneQueuesEnabled() bool {
	return parseRt != nil && parseRt.laneQueues
}

// noteLaneDeferred records that a pass declined to render a fiber, so
// commitRoot knows a follow-up pass is owed.
func (parseRt *Runtime) noteLaneDeferred(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseRt.schedulerState.lanes.deferredThisPass++
	parseRt.schedulerState.lanes.markLanePending(parseFiber.updateLane, time.Now())
}

// scheduleDeferredLaneWork starts a follow-up pass for whatever a completed
// pass declined to render.
//
// Called from commitRoot, after the tree is consistent. Without it a deferred
// fiber stays dirty with nothing scheduled to consume it — deferral would
// become the stranded-update bug it was meant to avoid.
func (parseRt *Runtime) scheduleDeferredLaneWork() {
	if !parseRt.laneQueuesEnabled() {
		return
	}
	if parseRt.schedulerState.lanes.deferredThisPass == 0 {
		return
	}
	parseRt.schedulerState.lanes.deferredThisPass = 0

	parseNextLane := parseRt.schedulerState.lanes.highestPendingLane()
	if parseNextLane == 0 {
		return
	}
	parseRt.scheduleUpdateWithLane(parseNextLane, false)
}
