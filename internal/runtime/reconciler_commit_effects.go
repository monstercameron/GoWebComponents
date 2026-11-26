package runtime

// runCleanups runs all cleanup functions for a fiber and its child subtree.
// It deliberately does NOT traverse the argument fiber's own sibling chain:
// commitDeletion passes a DELETION-tagged fiber whose old-tree siblings may
// survive (their Hooks objects are shared with their live alternates), so
// cleaning the sibling chain would tear down effects and release event-handler
// wrappers on components that are still mounted.
func (parseRt *Runtime) runCleanups(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}
	parseRt.runFiberCleanups(parseFiber)
	for parseChild := parseFiber.child; parseChild != nil; parseChild = parseChild.sibling {
		parseRt.runCleanups(parseChild)
	}
}

// runFiberCleanups runs the cleanups and releases the event-handler wrappers
// for a single fiber, without traversing children or siblings.
func (parseRt *Runtime) runFiberCleanups(parseFiber *Fiber) {

	// Run this fiber's cleanups
	if parseFiber.hooks != nil {
		for parseIndex, parseCleanup := range parseFiber.hooks.cleanups {
			if parseCleanup != nil {
				parseStart := commitTimingStart()
				var isHandled bool
				func() {
					defer func() {
						if parseRecovered := recover(); parseRecovered != nil {
							if panicPhaseMayRecoverWithBoundary(PanicPhaseCleanup) {
								_, isHandled = parseRt.recoverBoundaryError(parseFiber, parseRecovered, boundaryPhaseCleanup)
							}
							if !isHandled {
								panic(markUnhandledPanic(parseFiber, boundaryPhaseCleanup, parseRecovered))
							}
						}
					}()
					parseCleanup()
				}()
				parseDurationNs := commitTimingSinceNs(parseStart)
				parseFiber.addCleanupDurationNs(parseDurationNs)
				parseRt.profiling.cleanupExecutions++
				parseRt.profiling.lastCleanupDurationNs = parseDurationNs
				parseRt.profiling.totalCleanupDurationNs += parseDurationNs
				recordSlowOperationDiagnostic("cleanup", parseFiber, parseDurationNs)
				parseFiber.hooks.cleanups[parseIndex] = nil
			}
		}
		// Release any wrapped event-handler funcs (e.g. js.Func in WASM) so the
		// JS GC bridge can free them.  The wrapper slot is zeroed to prevent a
		// double-release if runCleanups is called again (e.g. hot-reload path).
		for parseIdx := range parseFiber.hooks.funcs {
			releaseFuncHandlerWrapper(parseFiber.hooks.funcs[parseIdx].wrapper)
			parseFiber.hooks.funcs[parseIdx] = funcHandlerValue{}
		}
	}
}

// RefreshEffectsForFiber forces a fiber subtree's effects to clean up and rerun on the next render.
func (parseRt *Runtime) RefreshEffectsForFiber(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	parseRt.runCleanups(parseFiber)
	bumpEffectEpochs(parseFiber)
}

// bumpEffectEpochs is an internal reconciler helper.
func bumpEffectEpochs(parseFiber *Fiber) {
	if parseFiber == nil {
		return
	}

	if parseFiber.hooks != nil {
		parseFiber.hooks.effectEpoch++
	}
	if parseFiber.child != nil {
		bumpEffectEpochs(parseFiber.child)
	}
	// Siblings loop instead of recursing so wide lists do not grow the stack.
	for parseSibling := parseFiber.sibling; parseSibling != nil; parseSibling = parseSibling.sibling {
		if parseSibling.hooks != nil {
			parseSibling.hooks.effectEpoch++
		}
		bumpEffectEpochs(parseSibling.child)
	}
}

// queuePendingEffectFiber records one fiber that scheduled post-commit effects during the current render pass.
func (parseRt *Runtime) queuePendingEffectFiber(parseFiber *Fiber) {
	if parseRt == nil || parseFiber == nil || len(parseFiber.effects) == 0 {
		return
	}
	parseLimit := parseRt.limits.withDefaults().MaxPendingEffectFibers
	if parseLimit > 0 && len(parseRt.pendingEffectFibers) >= parseLimit {
		clear(parseRt.pendingEffectFibers)
		parseRt.pendingEffectFibers = parseRt.pendingEffectFibers[:0]
		hasAlreadyOverflowed := parseRt.pendingEffectOverflow
		parseRt.pendingEffectOverflow = true
		parseRt.pendingEffectOverflowCount++
		parseRt.tracksPendingEffects = false
		// R6 — diagnostics carry an allocation budget, and this one did not.
		//
		// Emptying the queue means the next 1024 fibers refill it and overflow
		// AGAIN within the same pass, so a large tree hit this path repeatedly.
		// Each visit built a fiber path string and walked the component stack:
		// per-overflow allocation, on the render thread, during the pass already
		// large enough to have overflowed. Reporting only on the transition into
		// the degraded state keeps the signal and drops the cost, and P4.2's
		// budget signal carries the count that the repeated warnings were
		// standing in for.
		if !hasAlreadyOverflowed {
			ReportDiagnosticWithContext("runtime", DiagnosticWarning, "pending effect queue exceeded its bounded capacity; falling back to full-tree effect scan", diagnosticPathForFiber(parseFiber), diagnosticComponentStack(parseFiber))
		}
		return
	}
	parseRt.tracksPendingEffects = true
	parseRt.pendingEffectFibers = append(parseRt.pendingEffectFibers, parseFiber)
}

// commitLayoutEffectsAndDeferPassive is the P1.1 commit tail.
//
// Layout effects run here, synchronously, still inside the commit task — they
// exist to observe committed DOM before paint. Passive effects are handed to
// the scheduler so the browser reaches its rendering step first.
//
// Two behaviors are deliberate:
//
//   - With no scheduler configured (native tests, SSR), passive effects run
//     inline. There is no paint to wait for, and deferring them would silently
//     change semantics for every native caller.
//   - The deferred drain re-reads its state at run time rather than capturing
//     it, because another commit may land between the schedule and the drain.
func (parseRt *Runtime) commitLayoutEffectsAndDeferPassive(parseCommittedRoot *Fiber) {
	isTrackedDrain := parseRt.tracksPendingEffects && !parseRt.pendingEffectOverflow

	if isTrackedDrain {
		parseRt.runPendingEffectsTier(effectTierLayout)
	} else {
		parseRt.runEffectsTier(parseCommittedRoot, effectTierLayout)
	}

	// No scheduler: nothing to yield to, so keep the single-task semantics
	// native callers already depend on.
	if parseRt.scheduler == nil {
		parseRt.drainPassiveEffects(parseCommittedRoot, isTrackedDrain)
		return
	}

	// A second commit before the drain fires would otherwise queue a second
	// drain for the same work; one pending drain is enough, and it always
	// reads current state.
	if parseRt.passiveDrainScheduled {
		return
	}
	parseRt.passiveDrainScheduled = true
	parseRt.scheduler.SetTimeout(func() {
		// A flush ahead of a new pass may have already run these and cleared the
		// debt. Draining again would take the untracked path — the queue is now
		// empty — and re-run every passive effect in the tree.
		if !parseRt.passiveDrainScheduled {
			return
		}
		parseRt.passiveDrainScheduled = false
		parseRt.drainPassiveEffects(parseRt.currentRoot, parseRt.tracksPendingEffects && !parseRt.pendingEffectOverflow)
	}, 0)
}

// flushPendingPassiveEffectsBeforeSchedule runs the previous commit's passive
// effects before a new pass begins.
//
// Without it those effects are silently DROPPED. The commit split leaves them
// queued in pendingEffectFibers with a drain scheduled for after paint, and
// scheduleUpdateWithLane truncates that list at the top of every pass. An update
// scheduled in the window between commit and drain — a layout effect calling
// setState, an atom write, a worker reply, none of them exotic — leaves the
// drain reading an empty list.
//
// Nothing reports it. The component simply never gets its UseEffect, so a
// subscription is never opened, a fetch never starts, a timer is never armed,
// and the bug presents as a feature that intermittently does not work.
//
// Running them here rather than merely preserving the list is what keeps the
// ordering honest: an effect from commit N must not observe the tree from commit
// N+1. This is the same reason React flushes passive effects at the top of
// scheduleUpdateOnFiber.
func (parseRt *Runtime) flushPendingPassiveEffectsBeforeSchedule() {
	if parseRt == nil || parseRt.flushingPassiveBeforeSchedule {
		return
	}
	if !parseRt.passiveDrainScheduled || len(parseRt.pendingEffectFibers) == 0 {
		return
	}
	parseRt.flushingPassiveBeforeSchedule = true
	defer func() { parseRt.flushingPassiveBeforeSchedule = false }()

	// Cleared first so the callback already queued with the scheduler sees the
	// debt settled and returns without draining a second time.
	parseRt.passiveDrainScheduled = false
	parseRt.drainPassiveEffects(parseRt.currentRoot,
		parseRt.tracksPendingEffects && !parseRt.pendingEffectOverflow)
}

// drainPassiveEffects runs the passive tier and clears the pass bookkeeping the
// layout tier deliberately left intact.
func (parseRt *Runtime) drainPassiveEffects(parseRoot *Fiber, isTrackedDrain bool) {
	if isTrackedDrain {
		parseRt.runPendingEffectsTier(effectTierPassive)
	} else {
		parseRt.runEffectsTier(parseRoot, effectTierPassive)
	}
	parseRt.tracksPendingEffects = false
	parseRt.pendingEffectOverflow = false
}

// effectTier selects which queued effects a drain pass runs (v5 P1.1).
//
// The split exists because layout and passive effects have opposite timing
// requirements: layout effects must observe the committed DOM *before* the
// browser paints, while passive effects must not delay that paint. Running both
// in one pass — the pre-v5 behavior — means any slow UseEffect holds the frame.
type effectTier uint8

const (
	// effectTierAll runs every queued effect in one pass, layout first.
	// Pre-P1.1 behavior, retained for the flag-off path.
	effectTierAll effectTier = iota
	// effectTierLayout runs only layout effects and LEAVES THE QUEUE INTACT
	// so the passive pass can still find its own effects.
	effectTierLayout
	// effectTierPassive runs only passive effects and consumes the queue.
	effectTierPassive
)

// runPendingEffects runs only the fibers that scheduled effects during the current render pass.
func (parseRt *Runtime) runPendingEffects() {
	parseRt.runPendingEffectsTier(effectTierAll)
}

// runPendingEffectsTier drains one tier across the fibers that queued effects.
//
// The pending list is cleared only by a queue-consuming tier, so a layout pass
// can run first and leave the same list for the passive pass that follows it
// after paint.
func (parseRt *Runtime) runPendingEffectsTier(parseTier effectTier) {
	if parseRt == nil || len(parseRt.pendingEffectFibers) == 0 {
		return
	}
	for _, parseFiber := range parseRt.pendingEffectFibers {
		parseRt.runFiberEffectsTier(parseFiber, parseTier)
	}
	if parseTier != effectTierLayout {
		clear(parseRt.pendingEffectFibers)
		parseRt.pendingEffectFibers = parseRt.pendingEffectFibers[:0]
	}
}

// runFiberEffects runs one fiber's queued effects without traversing descendants.
//
// Layout effects (UseLayoutEffect, G36) run before passive effects (UseEffect)
// within the fiber, so measure/focus/scroll work observes the committed DOM
// before any passive effect mutates further. When the fiber has no layout
// effects (the common case) the effects run in their queued order in a single
// pass.
// runFiberEffectsTier runs one fiber's queued effects for a single tier.
//
// Queue consumption is tier-dependent and load-bearing. effectTierLayout leaves
// the queue in place because its passive counterpart still has to find those
// effects after paint; every other tier truncates, because effects must run
// exactly once — a bailout-reused fiber keeps its Fiber object across commits,
// and the full-tree runEffects fallback would otherwise re-execute stale queued
// effects, re-firing mount effects on unrelated updates and overwriting their
// cleanups without calling them.
func (parseRt *Runtime) runFiberEffectsTier(parseFiber *Fiber, parseTier effectTier) {
	if parseFiber == nil {
		return
	}

	parseEffects := parseFiber.effects
	if len(parseEffects) == 0 {
		return
	}
	// Only reset the accumulator on a pass that starts the fiber's work, so a
	// split layout+passive commit reports their sum rather than the last tier.
	if parseTier != effectTierPassive {
		parseFiber.setEffectDurationNs(0)
	}

	if parseTier != effectTierLayout {
		defer func() {
			// clear() before the reslice: an Effect holds Fn, a closure over the
			// render that queued it. Truncating alone left THIS render's closures
			// — and everything they captured, including the props of the render
			// that just finished — reachable in the backing array until a later
			// render appended over the slot. A component that mounts, runs its
			// effects and then sits idle never reaches that point, and one whose
			// effect count shrinks never reaches it for the tail.
			clear(parseFiber.effects)
			parseFiber.effects = parseFiber.effects[:0]
		}()
	}

	switch parseTier {
	case effectTierLayout:
		for parseI := range parseEffects {
			if parseEffects[parseI].Layout {
				parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
			}
		}
		return
	case effectTierPassive:
		for parseI := range parseEffects {
			if !parseEffects[parseI].Layout {
				parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
			}
		}
		return
	}

	parseHasLayout := false
	for parseI := range parseEffects {
		if parseEffects[parseI].Layout {
			parseHasLayout = true
			break
		}
	}

	if !parseHasLayout {
		for parseI := range parseEffects {
			parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
		}
		return
	}

	// Two passes: layout effects first, then passive effects.
	for parseI := range parseEffects {
		if parseEffects[parseI].Layout {
			parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
		}
	}
	for parseI := range parseEffects {
		if !parseEffects[parseI].Layout {
			parseRt.runOneEffect(parseFiber, &parseEffects[parseI])
		}
	}
}

// runOneEffect runs a single effect with effect-boundary panic containment and
// profiling, storing any returned cleanup at the effect's cleanup index.
func (parseRt *Runtime) runOneEffect(parseFiber *Fiber, parseEffect *Effect) {
	// The outgoing cleanup runs HERE, not where the effect was queued.
	//
	// GoUseEffect used to run it inline in the component body, during render.
	// A render can be interrupted, restarted, or discarded, and a cleanup that
	// ran during one that never committed has torn down state with nothing
	// queued to replace it. Running it at commit also gives the ordering
	// callers expect: cleanup immediately precedes the setup that supersedes it,
	// with the committed DOM already in place.
	parseRt.runOutgoingCleanup(parseFiber, parseEffect.CleanupIndex)

	parseStart := commitTimingStart()
	parseCleanup := func() func() {
		var parseCleanupFn func()
		var isHandled bool
		func() {
			defer func() {
				if parseRecovered := recover(); parseRecovered != nil {
					if panicPhaseMayRecoverWithBoundary(PanicPhaseEffect) {
						_, isHandled = parseRt.recoverBoundaryError(parseFiber, parseRecovered, boundaryPhaseEffect)
					}
					if !isHandled {
						panic(markUnhandledPanic(parseFiber, boundaryPhaseEffect, parseRecovered))
					}
				}
			}()
			parseCleanupFn = parseEffect.Fn()
		}()
		return parseCleanupFn
	}()
	parseDurationNs := commitTimingSinceNs(parseStart)
	parseFiber.addEffectDurationNs(parseDurationNs)
	parseRt.profiling.effectExecutions++
	parseRt.profiling.lastEffectDurationNs = parseDurationNs
	parseRt.profiling.totalEffectDurationNs += parseDurationNs
	recordSlowOperationDiagnostic("effect", parseFiber, parseDurationNs)
	if parseCleanup != nil {
		parseFiber.hooks.cleanups[parseEffect.CleanupIndex] = parseCleanup
	}
	parseRt.checkStrictEffectCleanupSymmetry(parseFiber, parseEffect.CleanupIndex, parseCleanup != nil)
}

// runOutgoingCleanup runs the cleanup left by the previous instance of one
// effect, if any, and clears the slot.
//
// Separate from runOneEffect's own accounting so the cleanup is attributed to
// cleanupDurationNs rather than to the effect that replaced it — otherwise a
// slow teardown is reported as a slow setup, and the diagnostic points at the
// wrong function.
func (parseRt *Runtime) runOutgoingCleanup(parseFiber *Fiber, parseCleanupIndex int) {
	if parseFiber == nil || parseFiber.hooks == nil {
		return
	}
	if parseCleanupIndex < 0 || parseCleanupIndex >= len(parseFiber.hooks.cleanups) {
		return
	}
	parseCleanup := parseFiber.hooks.cleanups[parseCleanupIndex]
	if parseCleanup == nil {
		return
	}
	parseFiber.hooks.cleanups[parseCleanupIndex] = nil

	parseStart := commitTimingStart()
	parseRt.callOutgoingCleanupContained(parseFiber, parseCleanup)
	parseDurationNs := commitTimingSinceNs(parseStart)
	parseFiber.addCleanupDurationNs(parseDurationNs)
	recordSlowOperationDiagnostic("cleanup", parseFiber, parseDurationNs)
}

// callOutgoingCleanupContained calls one superseded effect cleanup and routes a
// panic to the nearest error boundary, as runOneEffect does for the effect body.
//
// Without containment a throwing cleanup unwound through the whole effect
// flush: the remaining effects of this fiber and every later fiber never ran,
// although their dependencies were already stored at render, so they would not
// run again either.
func (parseRt *Runtime) callOutgoingCleanupContained(parseFiber *Fiber, parseCleanup func()) {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			var isHandled bool
			if panicPhaseMayRecoverWithBoundary(PanicPhaseCleanup) {
				_, isHandled = parseRt.recoverBoundaryError(parseFiber, parseRecovered, boundaryPhaseCleanup)
			}
			if !isHandled {
				panic(markUnhandledPanic(parseFiber, boundaryPhaseCleanup, parseRecovered))
			}
		}
	}()
	parseCleanup()
}

// runEffects runs all effects for a fiber tree.
//
// Ordering is DELIBERATE and load-bearing (do not "fix" it to match React):
// effect SETUP runs top-down parent->child (this fiber, then child, then
// sibling), where React runs setup child->parent. The corresponding cleanup
// order (parent->child on deletion) is the one React guarantees and is pinned
// by ui/effect_ordering_native_test.go — see its architectural note. Switching
// setup to bottom-up is an owner-level architecture change, not an audit fix.
func (parseRt *Runtime) runEffects(parseFiber *Fiber) {
	parseRt.runEffectsTier(parseFiber, effectTierAll)
}

// runEffectsTier is the tree-walking fallback used when the pending-effect
// queue overflowed. It preserves runEffects' top-down setup order per tier.
func (parseRt *Runtime) runEffectsTier(parseFiber *Fiber, parseTier effectTier) {
	if parseFiber == nil {
		return
	}

	parseRt.runFiberEffectsTier(parseFiber, parseTier)

	// Recursively run effects for children and siblings
	if parseFiber.child != nil {
		parseRt.runEffectsTier(parseFiber.child, parseTier)
	}
	// Siblings loop instead of recursing so wide lists do not grow the stack.
	for parseSibling := parseFiber.sibling; parseSibling != nil; parseSibling = parseSibling.sibling {
		parseRt.runFiberEffectsTier(parseSibling, parseTier)
		parseRt.runEffectsTier(parseSibling.child, parseTier)
	}
}
