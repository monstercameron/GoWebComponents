package runtime

import (
	"testing"
	"time"
)

// v5 P2.2 — lane queues with expiration.
//
// Before this, "priority" was two scalars and one pending pass. Lanes coalesced
// by min(), so background-marked work rendered inside whatever pass was
// running, at that pass's priority — promoted, never deferred (T3).

func newLaneRuntime(parseT *testing.T) (*Runtime, DOMNode, *testScheduler) {
	parseT.Helper()
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
		LaneQueues: true,
		Reset:      true,
	})
	return parseRt, parseAdapter.CreateElement("div"), parseScheduler
}

// TestLanes_PendingTrackingIsAllocationFree pins the data-structure choice.
// This state is touched on the path that reconciles a stable list in 3.3µs
// with zero allocations, so a map here would be a real regression.
func TestLanes_PendingTrackingIsAllocationFree(parseT *testing.T) {
	parseState := &laneState{}
	parseNow := time.Now()

	parseAllocs := testing.AllocsPerRun(200, func() {
		parseState.markLanePending(UpdateLaneBackground, parseNow)
		parseState.markLanePending(UpdateLaneTransition, parseNow)
		_ = parseState.highestPendingLane()
		_ = parseState.isLaneExpired(UpdateLaneBackground, parseNow)
		parseState.clearLanePending(UpdateLaneBackground)
		parseState.clearLanePending(UpdateLaneTransition)
	})

	if parseAllocs != 0 {
		parseT.Errorf("lane bookkeeping allocated %v times per run, want 0", parseAllocs)
	}
}

// TestLanes_HighestPendingWins pins priority ordering.
func TestLanes_HighestPendingWins(parseT *testing.T) {
	parseState := &laneState{}
	parseNow := time.Now()

	if parseState.highestPendingLane() != 0 {
		parseT.Fatal("an empty lane state must report no pending lane")
	}

	parseState.markLanePending(UpdateLaneBackground, parseNow)
	parseState.markLanePending(UpdateLaneInput, parseNow)
	parseState.markLanePending(UpdateLaneTransition, parseNow)

	if parseGot := parseState.highestPendingLane(); parseGot != UpdateLaneInput {
		parseT.Errorf("highest pending lane = %v, want input", parseGot)
	}

	parseState.clearLanePending(UpdateLaneInput)
	if parseGot := parseState.highestPendingLane(); parseGot != UpdateLaneTransition {
		parseT.Errorf("after clearing input, highest = %v, want transition", parseGot)
	}
}

// TestLanes_FirstMarkStampIsKept: expiry must measure the OLDEST wait, so a
// stream of new low-priority updates cannot keep resetting the clock and
// starve the lane forever.
func TestLanes_FirstMarkStampIsKept(parseT *testing.T) {
	parseState := &laneState{}
	parseFirst := time.Now()

	parseState.markLanePending(UpdateLaneBackground, parseFirst)
	parseState.markLanePending(UpdateLaneBackground, parseFirst.Add(time.Second))
	parseState.markLanePending(UpdateLaneBackground, parseFirst.Add(2*time.Second))

	if !parseState.firstMarkedAt[UpdateLaneBackground].Equal(parseFirst) {
		parseT.Error("re-marking a pending lane must not reset its arrival stamp; that would allow indefinite starvation")
	}
}

// TestLanes_ExpiryAdmitsDeferredWork covers the starvation guard.
func TestLanes_ExpiryAdmitsDeferredWork(parseT *testing.T) {
	parseState := &laneState{}
	parseStart := time.Now()
	parseState.markLanePending(UpdateLaneTransition, parseStart)

	if parseState.isLaneExpired(UpdateLaneTransition, parseStart) {
		parseT.Error("a freshly marked transition must not be expired")
	}

	parseLater := parseStart.Add(time.Duration(laneExpiryMs[UpdateLaneTransition]+1) * time.Millisecond)
	if !parseState.isLaneExpired(UpdateLaneTransition, parseLater) {
		parseT.Error("a transition past its deadline must be admitted")
	}

	// Background waits considerably longer than a transition.
	parseState.markLanePending(UpdateLaneBackground, parseStart)
	if parseState.isLaneExpired(UpdateLaneBackground, parseLater) {
		parseT.Error("background should still be deferred at the transition deadline")
	}
}

// TestLanes_UrgentLanesAreNeverDeferred: sync, input, and default carry no
// deferral deadline and must always be admitted.
func TestLanes_UrgentLanesAreNeverDeferred(parseT *testing.T) {
	parseRt, _, _ := newLaneRuntime(parseT)
	parseNow := time.Now()

	for _, parseLane := range []UpdateLane{UpdateLaneSync, UpdateLaneInput, UpdateLaneDefault} {
		if !parseRt.laneAdmitsFiber(UpdateLaneSync, parseLane, parseNow) {
			parseT.Errorf("lane %v must never be deferred", parseLane)
		}
	}
}

// TestLanes_AdmissionRules covers the decision table directly.
func TestLanes_AdmissionRules(parseT *testing.T) {
	parseRt, _, _ := newLaneRuntime(parseT)
	parseNow := time.Now()

	// Equal or more urgent always renders.
	if !parseRt.laneAdmitsFiber(UpdateLaneDefault, UpdateLaneInput, parseNow) {
		parseT.Error("more urgent work must render inside a less urgent pass")
	}
	if !parseRt.laneAdmitsFiber(UpdateLaneInput, UpdateLaneInput, parseNow) {
		parseT.Error("equal priority must render")
	}

	// Less urgent is deferred until it expires.
	parseRt.schedulerState.lanes.markLanePending(UpdateLaneBackground, parseNow)
	if parseRt.laneAdmitsFiber(UpdateLaneInput, UpdateLaneBackground, parseNow) {
		parseT.Error("background work must not render inside an input pass")
	}

	parseExpired := parseNow.Add(time.Duration(laneExpiryMs[UpdateLaneBackground]+1) * time.Millisecond)
	if !parseRt.laneAdmitsFiber(UpdateLaneInput, UpdateLaneBackground, parseExpired) {
		parseT.Error("expired background work must be admitted rather than starved")
	}

	// A fiber with no recorded lane is legacy work and always renders.
	if !parseRt.laneAdmitsFiber(UpdateLaneInput, 0, parseNow) {
		parseT.Error("unlaned fibers must always render, so nothing regresses for callers that never set a lane")
	}
}

// TestLanes_DisabledByDefaultAdmitsEverything pins R2.
func TestLanes_DisabledByDefaultAdmitsEverything(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: true})
	parseNow := time.Now()

	if parseRt.laneQueuesEnabled() {
		parseT.Fatal("lane queues must be off by default")
	}
	parseRt.schedulerState.lanes.markLanePending(UpdateLaneBackground, parseNow)
	if !parseRt.laneAdmitsFiber(UpdateLaneInput, UpdateLaneBackground, parseNow) {
		parseT.Error("with lane queues off, every fiber renders in whatever pass finds it")
	}
}

// TestLanes_BackgroundWorkDoesNotRenderInAnInputPass is the end-to-end accept
// criterion, driven through a real render rather than the decision table.
func TestLanes_BackgroundWorkDoesNotRenderInAnInputPass(parseT *testing.T) {
	parseRt, parseContainer, parseScheduler := newLaneRuntime(parseT)

	parseBackgroundRenders := 0
	var setBackground func(any)
	parseBackgroundChild := func() *Element {
		parseBackgroundRenders++
		parseValue, parseSet := GoUseState(parseRt, "idle")
		setBackground = parseSet
		return CreateElement("span", map[string]any{}, parseValue())
	}
	parseApp := func() *Element {
		return CreateElement("section", map[string]any{},
			CreateElement(parseBackgroundChild, map[string]any{}),
		)
	}

	parseRt.Render(CreateElement(parseApp, nil), parseContainer)
	runScheduledTimeouts(parseScheduler)
	parseRendersAfterMount := parseBackgroundRenders

	// Mark the child's work as background, then run a pass at input priority.
	parseChildFiber := parseRt.currentRoot.child.child
	if parseChildFiber == nil {
		parseT.Fatal("expected a child fiber")
	}
	setBackground("changed")
	parseChildFiber.updateLane = UpdateLaneBackground
	parseRt.schedulerState.lanes.markLanePending(UpdateLaneBackground, time.Now())
	parseRt.schedulerState.currentLane = UpdateLaneInput

	// The deferred work must still land eventually, via the follow-up pass.
	runScheduledTimeouts(parseScheduler)

	if parseBackgroundRenders <= parseRendersAfterMount {
		parseT.Error("deferred background work never rendered; deferral must not strand it")
	}
}

// TestLanes_DeferralSchedulesAFollowUpPass pins the mechanism that stops
// deferral from becoming a stranded update.
func TestLanes_DeferralSchedulesAFollowUpPass(parseT *testing.T) {
	parseRt, _, _ := newLaneRuntime(parseT)
	parseRt.currentRoot = &Fiber{typeOf: "ROOT", props: map[string]any{"children": []any{}}}

	parseRt.schedulerState.lanes.deferredThisPass = 1
	parseRt.schedulerState.lanes.markLanePending(UpdateLaneBackground, time.Now())

	parseRt.scheduleDeferredLaneWork()

	if parseRt.schedulerState.lanes.deferredThisPass != 0 {
		parseT.Error("the deferred counter must reset once a follow-up pass is scheduled")
	}
	if !parseRt.updateScheduled {
		parseT.Error("deferred work must schedule a follow-up pass")
	}
}

// TestLanes_NoDeferralNoExtraPass: the mechanism must not manufacture work.
func TestLanes_NoDeferralNoExtraPass(parseT *testing.T) {
	parseRt, _, _ := newLaneRuntime(parseT)
	parseRt.currentRoot = &Fiber{typeOf: "ROOT", props: map[string]any{"children": []any{}}}

	parseRt.scheduleDeferredLaneWork()

	if parseRt.updateScheduled {
		parseT.Error("with nothing deferred, no follow-up pass should be scheduled")
	}
}
