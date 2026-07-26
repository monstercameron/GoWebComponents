package runtime

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	schedulerMu sync.Mutex
	// globalInfiniteDeadline is a shared instance to avoid allocation
	globalInfiniteDeadline = &infiniteDeadline{}
)

type infiniteDeadline struct{}

// TimeRemaining is a core package helper.
func (parseD *infiniteDeadline) TimeRemaining() float64 { return 1000 } // lots of time
// DidTimeout is a core package helper.
func (parseD *infiniteDeadline) DidTimeout() bool { return false }

// frameBudgetDeadline is a real wall-clock slice budget (v5 P1.2).
//
// workLoop yields when TimeRemaining() drops below 1ms, so the budget is the
// share of a frame the render pass may consume before handing control back.
// The remaining frame time goes to commit, effects, and the browser's own
// style/layout/paint work.
type frameBudgetDeadline struct {
	startedAt time.Time
	budgetMs  float64
}

func newFrameBudgetDeadline(parseBudgetMs float64) *frameBudgetDeadline {
	return &frameBudgetDeadline{startedAt: time.Now(), budgetMs: parseBudgetMs}
}

func (parseD *frameBudgetDeadline) TimeRemaining() float64 {
	return parseD.budgetMs - float64(time.Since(parseD.startedAt).Microseconds())/1000
}

func (parseD *frameBudgetDeadline) DidTimeout() bool { return parseD.TimeRemaining() <= 0 }

// defaultFrameBudgetMs leaves roughly two thirds of a 60Hz frame for commit,
// effects, and the browser's own rendering work.
const defaultFrameBudgetMs = 5.0

// dispatchRuntimeWork schedules one continuation or runs it immediately when no scheduler is configured.
func dispatchRuntimeWork(parseScheduler Scheduler, parseContinueWork func()) {
	if parseContinueWork == nil {
		return
	}
	if parseScheduler == nil {
		parseContinueWork()
		return
	}
	parseScheduler.SetTimeout(parseContinueWork, 0)
}

// idleDispatchBackstopMs bounds how long idle-lane work may wait for an idle
// callback that may never arrive.
const idleDispatchBackstopMs = 50

// dispatchLaneWork routes a continuation by priority (v5 P1.3).
//
// Transition and background lanes go through RequestIdleCallback so they run in
// whatever time the browser has left after painting, instead of competing with
// input on the macrotask queue. Everything else keeps SetTimeout.
//
// RequestIdleCallback has been part of the Scheduler interface (and implemented
// by jsdom and mockdom) since before v5, but nothing in the runtime ever called
// it, so a custom Scheduler could implement it as a no-op without consequence.
// Making it load-bearing changes that contract, so idle dispatch always arms a
// SetTimeout backstop alongside it and takes whichever fires first. A no-op
// implementation therefore still makes progress — one backstop late — and says
// so once rather than stalling background work forever.
func (parseRt *Runtime) dispatchLaneWork(parseLane UpdateLane, parseContinueWork func()) {
	if parseContinueWork == nil {
		return
	}
	if parseRt == nil || parseRt.scheduler == nil {
		parseContinueWork()
		return
	}
	if parseLane != UpdateLaneTransition && parseLane != UpdateLaneBackground {
		parseRt.scheduler.SetTimeout(parseContinueWork, 0)
		return
	}

	isRun := false
	runOnce := func(parseVia string) {
		if isRun {
			return
		}
		isRun = true
		if parseVia == "backstop" && !parseRt.idleFallbackReported {
			parseRt.idleFallbackReported = true
			ReportDiagnostic("runtime", DiagnosticWarning,
				"Scheduler.RequestIdleCallback did not run idle-lane work within the backstop window; falling back to SetTimeout. A no-op RequestIdleCallback implementation will keep taking this path.")
		}
		parseContinueWork()
	}

	parseRt.scheduler.RequestIdleCallback(func(Deadline) { runOnce("idle") })
	parseRt.scheduler.SetTimeout(func() { runOnce("backstop") }, idleDispatchBackstopMs)
}

// NOTE(sched A/B 2026-07-04): dispatching the INITIAL work-loop kick via
// queueMicrotask instead of SetTimeout(0) was tried and REVERTED. Example 201
// same-run geomean vs React fell 0.658 -> 0.495 (every scenario worse, React
// itself unchanged-to-faster): the microtask fires before Go's parked
// goroutines finish enqueueing the action's remaining state writes, so one
// user action commits as multiple split render passes. The setTimeout hop is
// what lets all goroutine state writes coalesce into one pass — do not
// "optimize" it away without re-running that experiment.

// ScheduleUpdateWithLane schedules a full root update on an explicit priority lane.
func (parseRt *Runtime) ScheduleUpdateWithLane(parseLane UpdateLane) {
	parseRt.scheduleUpdateWithLane(parseLane, true)
}

// getContinueWorkFn is a core package helper.
func (parseRt *Runtime) getContinueWorkFn() func() {
	if parseRt.continueWorkFn == nil {
		parseRt.continueWorkFn = parseRt.continueWorkLoop
	}
	return parseRt.continueWorkFn
}

// ScheduleUpdate schedules a full tree update from the root
func (parseRt *Runtime) ScheduleUpdate() {
	parseRt.scheduleUpdateWithLane(UpdateLaneDefault, true)
}

// scheduleUpdateWithLane schedules a full tree update from the root.
func (parseRt *Runtime) scheduleUpdateWithLane(parseLane UpdateLane, shouldRecordReplay bool) {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	parseRt.profiling.scheduledRootUpdates++

	if parseRt.currentRoot == nil || parseRt.updateScheduled {
		if parseRt.updateScheduled {
			parseRt.coalesceScheduledUpdateLocked(parseLane)
		}
		schedulerMu.Unlock()
		return
	}

	parseRt.updateScheduled = true
	parseRt.schedulerState.beginScheduledLocked(parseLane)
	if shouldRecordReplay {
		parseRt.recordReplayUpdateLocked(replayUpdateKindRoot, nil, "", parseLane)
	}

	// Reuse the previous alternate root when available to reduce per-update
	// allocations.  The roots form a stable two-fiber cycle (current.alternate
	// is the previous generation, which becomes the next wip), so there is no
	// growing chain to leak and isFiberDirty only ever follows one hop.
	parseRt.wipRoot = acquireWorkInProgress(parseRt.currentRoot)
	*parseRt.wipRoot = Fiber{
		typeOf:       parseRt.currentRoot.typeOf,
		dom:          parseRt.currentRoot.dom,
		props:        parseRt.currentRoot.props,
		children:     parseRt.currentRoot.children,
		alternate:    parseRt.currentRoot,
		dirty:        true,
		ownerRuntime: parseRt,
	}

	parseRt.nextUnitOfWork = parseRt.wipRoot
	// Reuse deletions slice capacity if possible
	if parseRt.deletions == nil {
		parseRt.deletions = make([]*Fiber, 0)
	} else {
		parseRt.deletions = parseRt.deletions[:0]
	}
	if parseRt.pendingEffectFibers == nil {
		parseRt.pendingEffectFibers = make([]*Fiber, 0)
	} else {
		parseRt.pendingEffectFibers = parseRt.pendingEffectFibers[:0]
	}

	// Schedule work loop
	parseContinueWork := parseRt.getContinueWorkFn()
	parseScheduler := parseRt.scheduler
	schedulerMu.Unlock()
	dispatchRuntimeWork(parseScheduler, parseContinueWork)
}

// FlushScheduledDiscreteWork synchronously runs a freshly scheduled, not yet
// started render pass. The DOM event bridge calls it after a component event
// handler returns, so a discrete user action commits inside its own task
// (React-style sync discrete flush) instead of waiting out the setTimeout(0)
// kick — while state writes from goroutines the handler spawned still land in
// a follow-up pass exactly as they would have with the macrotask hop. The
// already-queued kick later finds no pending work and no-ops. It refuses to
// run when any work loop is active on this stack (a handler fired
// synchronously by a commit-phase DOM write) or when the pass has already
// consumed work slices.
func (parseRt *Runtime) FlushScheduledDiscreteWork() {
	if parseRt == nil {
		return
	}
	schedulerMu.Lock()
	shouldFlush := parseRt.updateScheduled && parseRt.workLoopDepth == 0 &&
		parseRt.wipRoot != nil && parseRt.nextUnitOfWork == parseRt.wipRoot
	schedulerMu.Unlock()
	if !shouldFlush {
		return
	}
	parseRt.continueWorkLoop()
}

// FlushGlobalDiscreteWork flushes the global runtime's freshly scheduled
// pass; the wasm DOM event bridge uses it because adapter-level wrappers do
// not carry a runtime reference.
func FlushGlobalDiscreteWork() {
	globalRuntimeMu.Lock()
	parseRt := globalRuntime
	globalRuntimeMu.Unlock()
	if parseRt != nil {
		parseRt.FlushScheduledDiscreteWork()
	}
}

// continueWorkLoop is a bound method to avoid closure allocation
func (parseRt *Runtime) continueWorkLoop() {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseFiber := parseRt.nextUnitOfWork
			if parseFiber == nil && parseRt.wipRoot != nil {
				parseFiber = parseRt.hydrationDiagnosticFiber(parseRt.wipRoot)
			}
			_, _ = finalizeUnhandledPanicContext("runtime", PanicPhaseDeferred, panicSubject(parseFiber), diagnosticPathForFiber(parseFiber), diagnosticComponentStack(parseFiber), parseRecovered)
			parseRt.recoverWorkLoopState()
			return
		}
	}()
	schedulerMu.Lock()
	parseRt.schedulerState.currentLane = parseRt.schedulerState.pendingLane
	schedulerMu.Unlock()
	parseRt.workLoop(parseRt.resolveWorkLoopDeadline())
}

// resolveWorkLoopDeadline picks the deadline for one work-loop slice (v5 P1.2).
//
// Historically this was always globalInfiniteDeadline, whose TimeRemaining()
// returns a constant 1000 — so the `TimeRemaining() < 1` yield check in workLoop
// could never fire and slicing was purely count-based. A slice of 1200 fibers
// takes whatever it takes, which on a large tree is well past a frame.
//
// The wall-clock budget makes that check real. It is gated on
// interruptRestartIsSafe: live time-slicing multiplies mid-pass yields, and each
// yield is a window for the T12 interrupt bug, so the improvement must not ship
// without P2.5's fix.
func (parseRt *Runtime) resolveWorkLoopDeadline() Deadline {
	if !parseRt.frameBudgetEnabled() {
		return globalInfiniteDeadline
	}
	return newFrameBudgetDeadline(parseRt.frameBudgetMs)
}

// frameBudgetEnabled reports whether time-based slicing may run.
func (parseRt *Runtime) frameBudgetEnabled() bool {
	if parseRt == nil || parseRt.frameBudgetMs <= 0 {
		return false
	}
	if !interruptRestartIsSafe {
		// Refuse rather than degrade quietly: a caller who asked for time
		// slicing should learn that the safety precondition is missing.
		ReportDiagnostic("runtime", DiagnosticWarning,
			"frame budget requested but the interrupt-safe restart path (v5 P2.5) is absent; falling back to count-based slicing")
		return false
	}
	return true
}

// workLoop processes work units during idle periods
func (parseRt *Runtime) workLoop(parseDeadline Deadline) {
	parseRt.workLoopDepth++
	defer func() { parseRt.workLoopDepth-- }()
	// Render and commit are frame-loop work, and marking them through the same
	// counter as event dispatch gives ownership ONE rule. Reading workLoopDepth
	// separately in insideFrameLoop would have left this region unowned, so a
	// goroutine spawned mid-render would still have been mistaken for the
	// renderer.
	parseRt.enterFrameLoop()
	defer parseRt.exitFrameLoop()
	if hookThreadingGuardEnabled && !parseRt.renderPassActive {
		parseRt.renderPassActive = true
		defer func() {
			parseRt.renderPassActive = false
			parseRt.renderPassOwnerGoroutineID = 0
		}()
	}
	parseRt.profiling.workLoopPasses++
	shouldYield := false
	parseUnits := 0
	parseMaxUnitsPerSlice := 300
	if parseDeadline == globalInfiniteDeadline {
		parseMaxUnitsPerSlice = 1200
	}
	if parseRt.schedulerState.currentLane != 0 {
		parseMaxUnitsPerSlice = parseRt.schedulerState.currentLane.maxUnitsPerSlice(parseMaxUnitsPerSlice)
	}

	// Inline check for common case
	for parseRt.nextUnitOfWork != nil && !shouldYield {
		parseRt.nextUnitOfWork = parseRt.performUnitOfWork(parseRt.nextUnitOfWork)
		parseUnits++
		parseRt.profiling.processedUnits++

		// Check if we should yield
		if parseDeadline.TimeRemaining() < 1 || parseUnits >= parseMaxUnitsPerSlice {
			shouldYield = true
		}
	}

	// If work is complete, commit
	if parseRt.wipRoot != nil && parseRt.nextUnitOfWork == nil {
		parseRt.commitRoot()
	} else if parseRt.nextUnitOfWork != nil {
		// More work remains. Route the continuation by lane (v5 P1.3) so
		// transition and background passes resume in idle time rather than
		// competing with input on the macrotask queue.
		parseRt.dispatchLaneWork(parseRt.schedulerState.currentLane, parseRt.getContinueWorkFn())
	}
}

// Render starts rendering a component tree
func (parseRt *Runtime) Render(parseElement *Element, parseContainer DOMNode) {
	if parseRt.domAdapter == nil {
		panic(actionableRuntimeDOMAdapterPanic("Render"))
	}
	parseStart := time.Now()
	var (
		shouldSchedule    bool
		parseContinueWork func()
		parseScheduler    Scheduler
	)
	schedulerMu.Lock()
	if parseRt.currentRoot == nil {
		parseRt.beginStartupProfilingLocked("render")
	}

	shouldSchedule = !parseRt.updateScheduled
	parseRt.updateScheduled = true
	parseRt.schedulerState.beginScheduledLocked(UpdateLaneSync)

	// Optimization: Break the alternate chain on the current root
	if parseRt.currentRoot != nil {
		parseRt.currentRoot.alternate = nil
	}

	parseRt.wipRoot = acquireWorkInProgress(parseRt.currentRoot)
	parseChildren := []any{parseElement}
	*parseRt.wipRoot = Fiber{
		typeOf:       "ROOT",
		dom:          parseContainer,
		props:        map[string]any{"children": parseChildren},
		children:     parseChildren,
		alternate:    parseRt.currentRoot,
		dirty:        true,
		ownerRuntime: parseRt,
	}

	parseRt.nextUnitOfWork = parseRt.wipRoot
	// Reuse deletions slice capacity if possible
	if parseRt.deletions == nil {
		parseRt.deletions = make([]*Fiber, 0)
	} else {
		parseRt.deletions = parseRt.deletions[:0]
	}
	if parseRt.pendingEffectFibers == nil {
		parseRt.pendingEffectFibers = make([]*Fiber, 0)
	} else {
		parseRt.pendingEffectFibers = parseRt.pendingEffectFibers[:0]
	}

	parseDurationNs := time.Since(parseStart).Nanoseconds()
	parseRt.profiling.renderCalls++
	parseRt.profiling.lastRenderDurationNs = parseDurationNs
	parseRt.recordProfilingEventLocked(ProfilingEvent{
		Domain:     "runtime",
		Name:       "render",
		Phase:      "finish",
		Target:     "root",
		DurationNs: parseDurationNs,
		Fields: map[string]string{
			"mode": "render",
		},
	})

	if shouldSchedule {
		parseContinueWork = parseRt.getContinueWorkFn()
		parseScheduler = parseRt.scheduler
	}
	schedulerMu.Unlock()
	if shouldSchedule {
		dispatchRuntimeWork(parseScheduler, parseContinueWork)
	}
}

// Hydrate starts a client resume attempt from an existing container.
func (parseRt *Runtime) Hydrate(parseElement *Element, parseContainer DOMNode) {
	if parseRt.domAdapter == nil {
		panic(actionableRuntimeDOMAdapterPanic("Hydrate"))
	}
	parseStart := time.Now()
	var (
		shouldSchedule    bool
		parseContinueWork func()
		parseScheduler    Scheduler
	)
	schedulerMu.Lock()
	if parseRt.currentRoot == nil {
		parseRt.beginStartupProfilingLocked("hydrate")
	}

	parseExistingChildren := 0
	if parseRt.domAdapter != nil && !IsDOMNodeNull(parseContainer) {
		for parseNode := parseRt.domAdapter.GetFirstChild(parseContainer); !IsDOMNodeNull(parseNode); parseNode = parseRt.domAdapter.GetNextSibling(parseNode) {
			parseExistingChildren++
		}
	}

	if parseExistingChildren > 0 {
		ReportDiagnostic("runtime", DiagnosticInfo, fmt.Sprintf("Hydrate found %d existing container child nodes and will attempt DOM reuse before falling back per subtree", parseExistingChildren))
	} else {
		ReportDiagnostic("runtime", DiagnosticInfo, "Hydrate found no existing container children and is proceeding with a fresh client render")
	}

	shouldSchedule = !parseRt.updateScheduled
	parseRt.updateScheduled = true
	parseRt.schedulerState.beginScheduledLocked(UpdateLaneSync)
	parseRt.hydrating = true
	parseRt.strictHydration = parseRt.nextHydrationStrict
	parseRt.beginHydrationMetrics(parseExistingChildren, parseRt.nextHydrationStrict)
	parseRt.nextHydrationStrict = false
	parseRt.deferredHydrationSubscriptions = parseRt.deferredHydrationSubscriptions[:0]
	if parseRt.deferredHydrationUpdates == nil {
		parseRt.deferredHydrationUpdates = make(map[*Fiber]bool)
	} else {
		clear(parseRt.deferredHydrationUpdates)
	}

	if parseRt.currentRoot != nil {
		parseRt.currentRoot.alternate = nil
	}

	parseRt.wipRoot = acquireWorkInProgress(parseRt.currentRoot)
	parseChildren := []any{parseElement}
	*parseRt.wipRoot = Fiber{
		typeOf:       "ROOT",
		dom:          parseContainer,
		props:        map[string]any{"children": parseChildren},
		children:     parseChildren,
		alternate:    parseRt.currentRoot,
		dirty:        true,
		hydration:    newHydrationBoundary(parseContainer, parseRt.domAdapter.GetFirstChild(parseContainer)),
		ownerRuntime: parseRt,
	}

	parseRt.nextUnitOfWork = parseRt.wipRoot
	if parseRt.deletions == nil {
		parseRt.deletions = make([]*Fiber, 0)
	} else {
		parseRt.deletions = parseRt.deletions[:0]
	}
	if parseRt.pendingEffectFibers == nil {
		parseRt.pendingEffectFibers = make([]*Fiber, 0)
	} else {
		parseRt.pendingEffectFibers = parseRt.pendingEffectFibers[:0]
	}

	parseDurationNs := time.Since(parseStart).Nanoseconds()
	parseRt.profiling.renderCalls++
	parseRt.profiling.lastRenderDurationNs = parseDurationNs
	parseRt.recordProfilingEventLocked(ProfilingEvent{
		Domain:     "runtime",
		Name:       "render",
		Phase:      "finish",
		Target:     "root",
		DurationNs: parseDurationNs,
		Fields: map[string]string{
			"mode": "hydrate",
		},
	})

	if shouldSchedule {
		parseContinueWork = parseRt.getContinueWorkFn()
		parseScheduler = parseRt.scheduler
	}
	schedulerMu.Unlock()
	if shouldSchedule {
		dispatchRuntimeWork(parseScheduler, parseContinueWork)
	}
}

// ScheduleUpdateForFiber schedules an update for a specific fiber.
func (parseRt *Runtime) ScheduleUpdateForFiber(parseFiber *Fiber) {
	parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, "hook")
}

// ScheduleUpdateForFiberWithOrigin schedules an update for a specific fiber and
// records the triggering cause for profiling and devtools inspection.
func (parseRt *Runtime) ScheduleUpdateForFiberWithOrigin(parseFiber *Fiber, parseOrigin string) {
	if parseFiber == nil {
		return
	}
	if parseRt.hydrating {
		if parseRt.deferredHydrationUpdates == nil {
			parseRt.deferredHydrationUpdates = make(map[*Fiber]bool)
		}
		parseRt.deferredHydrationUpdates[parseFiber] = true
		return
	}
	parseRt.profiling.scheduledFiberMarks++
	parseLane := laneForUpdateOrigin(parseOrigin)
	parseRt.recordReplayUpdate(replayUpdateKindFiber, fiberPathIndexes(parseFiber), parseOrigin, parseLane)
	parseCurrentOrigin := ""
	if parseFiber.dirty || parseFiber.needsUpdate {
		parseCurrentOrigin = parseFiber.updateOrigin
	}
	parseOrigin = buildScheduledUpdateOrigin(parseCurrentOrigin, parseOrigin, "hook")
	parseFiber.updateOrigin = parseOrigin

	// Mark fiber and parents as dirty
	isParseFirst := true
	for parseF := parseFiber; parseF != nil; parseF = parseF.parent {
		// Optimization: if fiber is already dirty and marked for update,
		// we can assume the path above it is already marked.
		// The originating fiber itself cannot short-circuit ancestor marking.
		if !isParseFirst && parseF.dirty && parseF.needsUpdate {
			break
		}
		parseF.dirty = true
		parseF.needsUpdate = true
		parseF.updateLane = parseLane
		if isParseFirst {
			parseF.updateOrigin = parseOrigin
		} else if parseF.updateOrigin == "" {
			parseF.updateOrigin = "ancestor"
		}
		isParseFirst = false
	}

	// Schedule update from root
	if !parseRt.updateScheduled {
		parseRt.scheduleUpdateWithLane(parseLane, false)
	} else {
		schedulerMu.Lock()
		parseRt.coalesceScheduledUpdateLocked(parseLane)
		schedulerMu.Unlock()
	}
}

// ScheduleGranularUpdateForFiber marks only the target fiber dirty and lets
// clean ancestors clone through to the dirty descendant on the next root pass.
func (parseRt *Runtime) ScheduleGranularUpdateForFiber(parseFiber *Fiber) {
	parseRt.ScheduleGranularUpdateForFiberWithOrigin(parseFiber, "fine-grained")
}

// ScheduleGranularUpdateForFiberWithOrigin marks only the target fiber dirty
// and records the triggering cause for profiling and devtools inspection.
func (parseRt *Runtime) ScheduleGranularUpdateForFiberWithOrigin(parseFiber *Fiber, parseOrigin string) {
	if parseFiber == nil {
		return
	}
	if parseRt.hydrating {
		if parseRt.deferredHydrationUpdates == nil {
			parseRt.deferredHydrationUpdates = make(map[*Fiber]bool)
		}
		parseRt.deferredHydrationUpdates[parseFiber] = true
		return
	}
	parseRt.profiling.scheduledFiberMarks++
	parseRt.profiling.scheduledGranularMarks++
	parseLane := laneForUpdateOrigin(parseOrigin)
	parseRt.recordReplayUpdate(replayUpdateKindGranular, fiberPathIndexes(parseFiber), parseOrigin, parseLane)
	parseCurrentOrigin := ""
	if parseFiber.dirty || parseFiber.needsUpdate {
		parseCurrentOrigin = parseFiber.updateOrigin
	}
	parseFiber.dirty = true
	parseFiber.subtreeDirty = true
	parseFiber.needsUpdate = true
	parseFiber.updateLane = parseLane
	parseRt.markFiberSubtreeDirty(parseFiber.parent)
	parseFiber.updateOrigin = buildScheduledUpdateOrigin(parseCurrentOrigin, parseOrigin, "fine-grained")
	if !parseRt.updateScheduled {
		parseRt.scheduleUpdateWithLane(parseLane, false)
	} else {
		schedulerMu.Lock()
		parseRt.coalesceScheduledUpdateLocked(parseLane)
		schedulerMu.Unlock()
	}
}

// markFiberSubtreeDirty marks one live ancestor chain as carrying descendant work.
func (parseRt *Runtime) markFiberSubtreeDirty(parseFiber *Fiber) {
	for parseCursor := parseFiber; parseCursor != nil; parseCursor = parseCursor.parent {
		parseCursor.subtreeDirty = true
	}
}

// ScheduleOwnedFiberUpdate chooses the narrowest safe scheduling path for one component-owned update.
func (parseRt *Runtime) ScheduleOwnedFiberUpdate(parseFiber *Fiber) {
	parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseFiber, "local-state")
}

// ScheduleOwnedFiberUpdateWithOrigin chooses the narrowest safe scheduling path for one component-owned update and records the triggering cause.
func (parseRt *Runtime) ScheduleOwnedFiberUpdateWithOrigin(parseFiber *Fiber, parseOrigin string) {
	if parseFiber == nil {
		return
	}
	getTargetFiber := parseRt.resolveOwnedFiberTarget(parseFiber)
	if getTargetFiber == nil {
		if parseRt.currentRoot == nil || parseRt.currentRoot.child == nil {
			parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, parseOrigin)
		}
		return
	}
	parseRt.ScheduleGranularUpdateForFiberWithOrigin(getTargetFiber, parseOrigin)
}

// ScheduleSubscribedFiberUpdate chooses the narrowest safe scheduling path for a subscription target.
func (parseRt *Runtime) ScheduleSubscribedFiberUpdate(parseFiber *Fiber) {
	parseRt.ScheduleSubscribedFiberUpdateWithOrigin(parseFiber, "subscription")
}

// ScheduleSubscribedFiberUpdateWithOrigin chooses the narrowest safe
// scheduling path for a subscription target and records the triggering cause.
func (parseRt *Runtime) ScheduleSubscribedFiberUpdateWithOrigin(parseFiber *Fiber, parseOrigin string) {
	if parseFiber == nil {
		return
	}
	parseFiber = parseRt.resolveSubscribedFiberTarget(parseFiber)
	if parseFiber == nil {
		return
	}
	if parseFiber.fineGrained {
		parseRt.ScheduleGranularUpdateForFiberWithOrigin(parseFiber, parseOrigin)
		return
	}
	if parseRt.buildSubscribedHasFineGrainedAncestor(parseFiber) {
		parseRt.ScheduleGranularUpdateForFiberWithOrigin(parseFiber, parseOrigin)
		return
	}
	parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, parseOrigin)
}

// buildSubscribedHasFineGrainedAncestor reports whether a subscription target lives under a fine-grained boundary.
func (parseRt *Runtime) buildSubscribedHasFineGrainedAncestor(parseFiber *Fiber) bool {
	if parseRt == nil || parseFiber == nil {
		return false
	}
	// Single upward pass: every ancestor on this chain shares one root, so the
	// in-current-tree check costs one O(depth) walk for the whole chain instead
	// of one per fine-grained ancestor (previously O(depth²) per notification).
	parseHasFineGrained := false
	parseRoot := parseFiber
	for parseCursor := parseFiber.parent; parseCursor != nil; parseCursor = parseCursor.parent {
		if parseCursor.fineGrained {
			parseHasFineGrained = true
		}
		parseRoot = parseCursor
	}
	if parseHasFineGrained && parseRoot == parseRt.currentRoot {
		return true
	}
	// Slow fallback for fibers whose live chain is not rooted in the current
	// tree: a fine-grained ancestor may still be reachable via its alternate.
	for parseCursor := parseFiber.parent; parseCursor != nil; parseCursor = parseCursor.parent {
		if !parseCursor.fineGrained {
			continue
		}
		if parseCursor.alternate != nil && parseCursor.alternate.fineGrained && parseRt.isFiberInCurrentTree(parseCursor.alternate) {
			return true
		}
	}
	return false
}

// normalizeUpdateOrigin is a core package helper.
func normalizeUpdateOrigin(parseOrigin string, parseFallback string) string {
	parseTrimmed := strings.TrimSpace(parseOrigin)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	return parseFallback
}

// buildScheduledUpdateOrigin merges one pending update origin with one new origin while preserving transition-lane classification.
func buildScheduledUpdateOrigin(parseCurrentOrigin string, parseNextOrigin string, parseFallback string) string {
	getCurrentOrigin := strings.TrimSpace(parseCurrentOrigin)
	getNextOrigin := normalizeUpdateOrigin(parseNextOrigin, parseFallback)
	if getCurrentOrigin == "" {
		return getNextOrigin
	}
	if strings.HasPrefix(getCurrentOrigin, "transition") {
		return getCurrentOrigin
	}
	if strings.HasPrefix(getNextOrigin, "transition") {
		return getNextOrigin
	}
	return getNextOrigin
}

// resolveSubscribedFiberTarget is a core package helper.
func (parseRt *Runtime) resolveSubscribedFiberTarget(parseFiber *Fiber) *Fiber {
	if parseRt == nil || parseFiber == nil {
		return parseFiber
	}
	if parseRt.isFiberInCurrentTree(parseFiber) {
		return parseFiber
	}
	if parseFiber.alternate != nil && parseRt.isFiberInCurrentTree(parseFiber.alternate) {
		return parseFiber.alternate
	}
	if parseMappedFiber := parseRt.buildSubscribedFiberFromAncestorAlternate(parseFiber); parseMappedFiber != nil {
		return parseMappedFiber
	}
	if parseFiber.parent == nil && parseFiber.alternate == nil && parseRt.currentRoot != nil && parseRt.currentRoot.child == nil {
		return parseFiber
	}
	return nil
}

// resolveOwnedFiberTarget resolves one component-owned update target onto the live tree when possible.
func (parseRt *Runtime) resolveOwnedFiberTarget(parseFiber *Fiber) *Fiber {
	if parseRt == nil || parseFiber == nil {
		return parseFiber
	}
	if parseRt.isFiberInCurrentTree(parseFiber) {
		return parseFiber
	}
	if parseFiber.alternate != nil && parseRt.isFiberInCurrentTree(parseFiber.alternate) {
		return parseFiber.alternate
	}
	if parseMappedFiber := parseRt.buildSubscribedFiberFromAncestorAlternate(parseFiber); parseMappedFiber != nil {
		return parseMappedFiber
	}
	return nil
}

// buildSubscribedFiberFromAncestorAlternate attempts to map one stale subscribed fiber onto the live tree.
func (parseRt *Runtime) buildSubscribedFiberFromAncestorAlternate(parseFiber *Fiber) *Fiber {
	if parseRt == nil || parseFiber == nil {
		return nil
	}
	parsePath := make([]int, 0, 8)
	parseCursor := parseFiber
	var parseLiveAncestor *Fiber
	for parseCursor != nil {
		parseParent := parseCursor.parent
		if parseParent == nil {
			break
		}
		parseChildIndex := buildSubscribedSiblingIndex(parseParent, parseCursor)
		if parseChildIndex < 0 {
			return nil
		}
		parsePath = append(parsePath, parseChildIndex)
		if parseParent.alternate != nil && parseRt.isFiberInCurrentTree(parseParent.alternate) {
			parseLiveAncestor = parseParent.alternate
			break
		}
		parseCursor = parseParent
	}
	if parseLiveAncestor == nil {
		return nil
	}
	parseLiveCursor := parseLiveAncestor
	for parsePathIndex := len(parsePath) - 1; parsePathIndex >= 0; parsePathIndex-- {
		parseLiveChild := parseLiveCursor.child
		parseChildIndex := parsePath[parsePathIndex]
		for parseStep := 0; parseStep < parseChildIndex && parseLiveChild != nil; parseStep++ {
			parseLiveChild = parseLiveChild.sibling
		}
		if parseLiveChild == nil {
			return nil
		}
		parseLiveCursor = parseLiveChild
	}
	if parseRt.isFiberInCurrentTree(parseLiveCursor) {
		return parseLiveCursor
	}
	return nil
}

// buildSubscribedSiblingIndex finds one child index inside a parent sibling chain.
func buildSubscribedSiblingIndex(parseParent *Fiber, parseChild *Fiber) int {
	if parseParent == nil || parseChild == nil {
		return -1
	}
	parseSiblingIndex := 0
	for parseSibling := parseParent.child; parseSibling != nil; parseSibling = parseSibling.sibling {
		if parseSibling == parseChild {
			return parseSiblingIndex
		}
		parseSiblingIndex++
	}
	return -1
}

// isFiberInCurrentTree is a core package helper.
func (parseRt *Runtime) isFiberInCurrentTree(parseFiber *Fiber) bool {
	if parseRt == nil || parseFiber == nil || parseRt.currentRoot == nil {
		return false
	}
	parseRoot := parseFiber
	for parseRoot.parent != nil {
		parseRoot = parseRoot.parent
	}
	return parseRoot == parseRt.currentRoot
}

// NOTE(v5 P2.4): EnqueueUI / ProcessUIQueue / GetUIQueueSize were REMOVED.
//
// They were the async inbox idea left unfinished. The package-level channel was
// never drained by anything outside tests — ProcessUIQueue had no production
// caller in the entire repository — so posted work simply accumulated until the
// 1024-slot buffer filled. Worse, the queue-full path ran the callback
// synchronously on the calling goroutine, which is exactly the cross-goroutine
// hand-off the queue existed to prevent: under load it degraded into the
// failure mode it was written to avoid.
//
// Runtime.PostAsync / DrainAsyncInbox (inbox.go, v5 P2.1) replace them, with
// the two properties this lacked: a real drain point on the frame loop, and
// per-runtime rather than package-global state.
