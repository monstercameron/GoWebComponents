package runtime

import (
	"testing"
	"time"
)

// Lane deferral must not consume the work it declined to do (v5 P2.2).
//
// performUnitOfWork defers a fiber by calling noteLaneDeferred — which marks the
// fiber's lane pending so commitRoot's scheduleDeferredLaneWork can start a
// follow-up pass — and then downgrading the visit to subtree-only so descendants
// still render. The subtree-only path continues into clearFiberDirty.
//
// That is the whole problem. clearFiberDirty clears dirty, needsUpdate, and then
// calls clearLanePending on exactly the lane noteLaneDeferred just marked,
// zeroing fiber.updateLane on the way out. The bookkeeping that says "a pass is
// owed" is erased by the same visit that created it, so highestPendingLane
// returns 0, scheduleDeferredLaneWork returns early, and the fiber is no longer
// dirty for anything else to find.
//
// TestLanes_BackgroundWorkDoesNotRenderInAnInputPass does not catch this: it
// calls a state setter, which schedules an ordinary update, and that update
// renders the component whether or not deferral kept its own bookkeeping. These
// tests drive performUnitOfWork directly so only the deferral path is in play.

// newDeferralFiber builds a mounted runtime and returns the child fiber to defer.
func newDeferralFiber(parseT *testing.T) (*Runtime, *Fiber) {
	parseT.Helper()
	parseRt, parseContainer, parseScheduler := newLaneRuntime(parseT)

	parseChild := func() *Element {
		return CreateElement("span", map[string]any{}, "child")
	}
	parseApp := func() *Element {
		return CreateElement("section", map[string]any{},
			CreateElement(parseChild, map[string]any{}),
		)
	}
	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)

	parseFiber := parseRt.currentRoot.child.child
	if parseFiber == nil {
		parseT.Fatal("expected a mounted child fiber")
	}
	return parseRt, parseFiber
}

// TestLanes_DeferralKeepsThePendingLane is the direct statement of the bug.
func TestLanes_DeferralKeepsThePendingLane(parseT *testing.T) {
	parseRt, parseFiber := newDeferralFiber(parseT)

	// Background work found by an input-priority pass: the fiber must be
	// declined here and owed a later pass.
	parseFiber.dirty = true
	parseFiber.needsUpdate = true
	parseFiber.updateLane = UpdateLaneBackground
	parseRt.schedulerState.lanes.markLanePending(UpdateLaneBackground, time.Now())
	parseRt.schedulerState.currentLane = UpdateLaneInput

	parseRt.performUnitOfWork(parseFiber)

	if parseRt.schedulerState.lanes.deferredThisPass == 0 {
		parseT.Fatal("the fiber was not deferred at all; this test is no longer exercising the deferral path")
	}
	if parseRt.schedulerState.lanes.highestPendingLane() == 0 {
		parseT.Error("deferral cleared the pending lane it had just marked, so scheduleDeferredLaneWork will find nothing to schedule and the update is lost")
	}
	if parseFiber.updateLane != UpdateLaneBackground {
		parseT.Errorf("the deferred fiber's lane was reset to %v; the follow-up pass can no longer tell what it owes", parseFiber.updateLane)
	}
	if !parseRt.isFiberDirty(parseFiber) {
		parseT.Error("the deferred fiber is no longer dirty, so the follow-up pass would render nothing")
	}
}

// TestLanes_DeferredWorkStillSchedulesAFollowUpPass closes the loop: after the
// pass that deferred it, the runtime must actually schedule the pass it owes.
//
// TestLanes_DeferralSchedulesAFollowUpPass already covers scheduleDeferredLaneWork
// in isolation, by setting deferredThisPass and the pending lane by hand. This
// one lets the real deferral produce that state, which is where it was lost.
func TestLanes_DeferredWorkStillSchedulesAFollowUpPass(parseT *testing.T) {
	parseRt, parseFiber := newDeferralFiber(parseT)

	parseFiber.dirty = true
	parseFiber.needsUpdate = true
	parseFiber.updateLane = UpdateLaneBackground
	parseRt.schedulerState.lanes.markLanePending(UpdateLaneBackground, time.Now())
	parseRt.schedulerState.currentLane = UpdateLaneInput

	parseRt.performUnitOfWork(parseFiber)

	parseRt.updateScheduled = false
	parseRt.scheduleDeferredLaneWork()

	if !parseRt.updateScheduled {
		parseT.Error("no follow-up pass was scheduled after a real deferral, so the declined work is stranded")
	}
}

// TestLanes_NonDeferredWorkStillClearsItsLane guards the fix from the other
// side. Keeping the lane pending is correct ONLY for a fiber that was declined;
// a fiber that actually rendered must release its lane, or every later pass
// re-defers work that is already done and the runtime never goes quiet.
func TestLanes_NonDeferredWorkStillClearsItsLane(parseT *testing.T) {
	parseRt, parseFiber := newDeferralFiber(parseT)

	// Same lane for the pass and the fiber, so it is admitted and rendered.
	parseFiber.dirty = true
	parseFiber.needsUpdate = true
	parseFiber.updateLane = UpdateLaneBackground
	parseRt.schedulerState.lanes.markLanePending(UpdateLaneBackground, time.Now())
	parseRt.schedulerState.currentLane = UpdateLaneBackground

	parseRt.performUnitOfWork(parseFiber)

	if parseRt.schedulerState.lanes.deferredThisPass != 0 {
		parseT.Fatal("the fiber was deferred when it should have been admitted")
	}
	if parseRt.schedulerState.lanes.highestPendingLane() != 0 {
		parseT.Error("a fiber that rendered must release its pending lane, or later passes keep re-deferring finished work")
	}
	if parseFiber.updateLane != 0 {
		parseT.Error("a fiber that rendered must clear its lane")
	}
	if parseRt.isFiberDirty(parseFiber) {
		parseT.Error("a fiber that rendered must not stay dirty")
	}
}
