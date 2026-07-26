package runtime

import "time"

const (
	transitionPendingAtomID = "__runtime_transition_pending"
	transitionDelayMs       = 16
)

// Transition scope is per-goroutine, not per-runtime.
//
// transitionDepth was a single counter, so ShouldDeferStateUpdates answered "is a
// transition running anywhere" — and every caller got that answer. A worker reply,
// a gRPC callback, or a timer that happened to write state while an unrelated
// StartTransition was open on another goroutine had its update demoted to the
// transition lane.
//
// That was harmless only by accident. The transition lane did not actually defer
// anything until the fiber clones started carrying updateLane; now they do, and a
// deferred transition waits up to its 500 ms expiry. So a coincidence of timing
// could hold an urgent write for half a second.
//
// This is the same question insideFrameLoop asks, and it is answered the same way:
// by goroutine identity (frame_loop_owner.go). The cost is placed where it does
// not matter. transitionDepth stays as a cheap total, checked first, so the common
// case — no transition anywhere — returns false with one mutex and no stack read.
// The identity lookup happens only in StartTransition itself, which is
// user-initiated and rare, and in ShouldDeferStateUpdates while a transition is
// actually in flight. A per-goroutine map rather than a single owner id, so nested
// transitions and two goroutines transitioning at once are both correct.

// StartTransition marks state and atom updates inside fn as non-urgent.
//
// The scope is this goroutine and this call. A goroutine spawned by fn is NOT
// inside the transition — the same rule React's startTransition follows, and the
// same distinction the async inbox draws — because the spawned work outlives the
// call and its urgency is not fn's to declare.
func (parseRt *Runtime) StartTransition(parseFn func()) {
	if parseFn == nil {
		return
	}
	if parseRt == nil {
		parseFn()
		return
	}

	parseOwner := frameLoopGoroutineID()
	parseRt.transitionMu.Lock()
	parseRt.transitionDepth++
	if parseRt.transitionOwners == nil {
		parseRt.transitionOwners = make(map[uint64]int, 1)
	}
	parseRt.transitionOwners[parseOwner]++
	parseRt.transitionMu.Unlock()
	defer func() {
		parseRt.transitionMu.Lock()
		if parseRt.transitionDepth > 0 {
			parseRt.transitionDepth--
		}
		if parseRt.transitionOwners[parseOwner] > 1 {
			parseRt.transitionOwners[parseOwner]--
		} else {
			delete(parseRt.transitionOwners, parseOwner)
		}
		parseRt.transitionMu.Unlock()
	}()

	parseFn()
}

// ShouldDeferStateUpdates reports whether the CALLER is inside a transition.
func (parseRt *Runtime) ShouldDeferStateUpdates() bool {
	if parseRt == nil {
		return false
	}
	parseRt.transitionMu.Lock()
	parseAnyOpen := parseRt.transitionDepth > 0
	parseRt.transitionMu.Unlock()
	if !parseAnyOpen {
		// The hot path: no transition anywhere, so no stack read. This is what
		// every ordinary state write takes.
		return false
	}

	parseOwner := frameLoopGoroutineID()
	parseRt.transitionMu.Lock()
	defer parseRt.transitionMu.Unlock()
	return parseRt.transitionOwners[parseOwner] > 0
}

// ScheduleTransition is a core package helper.
func (parseRt *Runtime) ScheduleTransition(parseFn func()) {
	if parseRt == nil || parseFn == nil {
		if parseFn != nil {
			// No runtime in hand, so the package-level recorder is the only option.
			ReportProfilingEvent("runtime", "transition", "immediate", "state-update", 0, nil)
			parseFn()
		}
		return
	}
	if parseRt.scheduler == nil {
		// Recorded on THIS runtime. The package-level ReportProfilingEvent resolves
		// through GetGlobalRuntime, so a second runtime's transitions were landing
		// in the global runtime's event ring — invisible on the runtime that
		// actually ran them, and noise on the one that did not. Same class as the
		// ui.PostAsync resolution fixed in 5751b608: reach for the global only when
		// there is genuinely no instance to use.
		parseRt.RecordProfilingEvent(ProfilingEvent{
			Domain: "runtime",
			Name:   "transition",
			Phase:  "immediate",
			Target: "state-update",
		})
		parseFn()
		return
	}
	parseScheduledAt := time.Now()
	parseRt.RecordProfilingEvent(ProfilingEvent{
		Domain: "runtime",
		Name:   "transition",
		Phase:  "scheduled",
		Target: "state-update",
		Fields: map[string]string{
			"delay_ms": "16",
		},
	})

	parseRt.transitionMu.Lock()
	parseRt.pendingTransitions++
	parseRt.transitionMu.Unlock()
	parseRt.setTransitionPending(true)

	parseRt.scheduler.SetTimeout(func() {
		defer func() {
			if parseRecovered := recover(); parseRecovered != nil {
				panicFinalUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduled transition", "", nil, parseRecovered)
			}
		}()
		parseRt.RecordProfilingEvent(ProfilingEvent{
			Domain:     "runtime",
			Name:       "transition",
			Phase:      "run",
			Target:     "state-update",
			DurationNs: time.Since(parseScheduledAt).Nanoseconds(),
		})
		defer parseRt.finishTransition()
		parseFn()
	}, transitionDelayMs)
}

// finishTransition is a core package helper.
func (parseRt *Runtime) finishTransition() {
	if parseRt == nil {
		return
	}

	parseRt.transitionMu.Lock()
	if parseRt.pendingTransitions > 0 {
		parseRt.pendingTransitions--
	}
	isParsePending := parseRt.pendingTransitions > 0
	parseRt.transitionMu.Unlock()

	parseRt.setTransitionPending(isParsePending)
}

// setTransitionPending is a core package helper.
func (parseRt *Runtime) setTransitionPending(isPending bool) {
	if parseRt == nil || parseRt.atomRegistry == nil {
		return
	}
	parseRt.atomRegistry.InitAtom(transitionPendingAtomID, false)
	parseRt.atomRegistry.setAtomAndNotify(transitionPendingAtomID, isPending, func(parseFiber *Fiber) {
		parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, "atom")
	})
}

// resolveStateUpdateValue is a core package helper.
func resolveStateUpdateValue[T any](parseCurrentValue T, parseNewValueOrUpdater any, isNilableState bool) (T, bool) {
	var parseNewValue T
	if parseFn, parseOk := parseNewValueOrUpdater.(func(T) T); parseOk {
		parseNewValue = parseFn(parseCurrentValue)
	} else if parseDirectValue, parseOk2 := parseNewValueOrUpdater.(T); parseOk2 {
		parseNewValue = parseDirectValue
	} else if parseNewValueOrUpdater == nil && isNilableState {
		var parseZero T
		parseNewValue = parseZero
	} else {
		var parseZero2 T
		return parseZero2, false
	}

	return parseNewValue, true
}
