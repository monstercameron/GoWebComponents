package runtime

import "time"

const (
	transitionPendingAtomID = "__runtime_transition_pending"
	transitionDelayMs       = 16
)

// StartTransition marks state and atom updates inside fn as non-urgent.
func (parseRt *Runtime) StartTransition(parseFn func()) {
	if parseFn == nil {
		return
	}
	if parseRt == nil {
		parseFn()
		return
	}

	parseRt.transitionMu.Lock()
	parseRt.transitionDepth++
	parseRt.transitionMu.Unlock()
	defer func() {
		parseRt.transitionMu.Lock()
		if parseRt.transitionDepth > 0 {
			parseRt.transitionDepth--
		}
		parseRt.transitionMu.Unlock()
	}()

	parseFn()
}

// ShouldDeferStateUpdates is a core package helper.
func (parseRt *Runtime) ShouldDeferStateUpdates() bool {
	if parseRt == nil {
		return false
	}
	parseRt.transitionMu.Lock()
	defer parseRt.transitionMu.Unlock()
	return parseRt.transitionDepth > 0
}

// ScheduleTransition is a core package helper.
func (parseRt *Runtime) ScheduleTransition(parseFn func()) {
	if parseRt == nil || parseFn == nil {
		if parseFn != nil {
			ReportProfilingEvent("runtime", "transition", "immediate", "state-update", 0, nil)
			parseFn()
		}
		return
	}
	if parseRt.scheduler == nil {
		ReportProfilingEvent("runtime", "transition", "immediate", "state-update", 0, nil)
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
func resolveStateUpdateValue[T any](parseCurrentValue T, parseNewValueOrUpdater interface{}, isNilableState bool) (T, bool) {
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
