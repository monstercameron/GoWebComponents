package runtime

const (
	transitionPendingAtomID = "__runtime_transition_pending"
	transitionDelayMs       = 16
)

// StartTransition marks state and atom updates inside fn as non-urgent.
func (rt *Runtime) StartTransition(fn func()) {
	if fn == nil {
		return
	}
	if rt == nil {
		fn()
		return
	}

	rt.transitionMu.Lock()
	rt.transitionDepth++
	rt.transitionMu.Unlock()
	defer func() {
		rt.transitionMu.Lock()
		if rt.transitionDepth > 0 {
			rt.transitionDepth--
		}
		rt.transitionMu.Unlock()
	}()

	fn()
}

func (rt *Runtime) ShouldDeferStateUpdates() bool {
	if rt == nil {
		return false
	}
	rt.transitionMu.Lock()
	defer rt.transitionMu.Unlock()
	return rt.transitionDepth > 0
}

func (rt *Runtime) ScheduleTransition(fn func()) {
	if rt == nil || fn == nil {
		if fn != nil {
			fn()
		}
		return
	}
	if rt.scheduler == nil {
		fn()
		return
	}

	rt.transitionMu.Lock()
	rt.pendingTransitions++
	rt.transitionMu.Unlock()
	rt.setTransitionPending(true)

	rt.scheduler.SetTimeout(func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				panicFinalUnhandledPanicContext("runtime", PanicPhaseDeferred, "scheduled transition", "", nil, recovered)
			}
		}()
		defer rt.finishTransition()
		fn()
	}, transitionDelayMs)
}

func (rt *Runtime) finishTransition() {
	if rt == nil {
		return
	}

	rt.transitionMu.Lock()
	if rt.pendingTransitions > 0 {
		rt.pendingTransitions--
	}
	pending := rt.pendingTransitions > 0
	rt.transitionMu.Unlock()

	rt.setTransitionPending(pending)
}

func (rt *Runtime) setTransitionPending(pending bool) {
	if rt == nil || rt.atomRegistry == nil {
		return
	}
	rt.atomRegistry.InitAtom(transitionPendingAtomID, false)
	rt.atomRegistry.setAtomAndNotify(transitionPendingAtomID, pending, rt.ScheduleUpdateForFiber)
}

func resolveStateUpdateValue[T any](currentValue T, newValueOrUpdater interface{}, nilableState bool) (T, bool) {
	var newValue T
	if fn, ok := newValueOrUpdater.(func(T) T); ok {
		newValue = fn(currentValue)
	} else if directValue, ok := newValueOrUpdater.(T); ok {
		newValue = directValue
	} else if newValueOrUpdater == nil && nilableState {
		var zero T
		newValue = zero
	} else {
		var zero T
		return zero, false
	}

	return newValue, true
}
