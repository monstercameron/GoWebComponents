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

// getContinueWorkFn is a core package helper.
func (parseRt *Runtime) getContinueWorkFn() func() {
	if parseRt.continueWorkFn == nil {
		parseRt.continueWorkFn = parseRt.continueWorkLoop
	}
	return parseRt.continueWorkFn
}

// ScheduleUpdate schedules a full tree update from the root
func (parseRt *Runtime) ScheduleUpdate() {
	schedulerMu.Lock()
	parseRt.profiling.scheduledRootUpdates++

	if parseRt.currentRoot == nil || parseRt.updateScheduled {
		schedulerMu.Unlock()
		return
	}

	parseRt.updateScheduled = true

	// Optimization: Break the alternate chain on the current root to prevent memory leaks
	// and long traversals during isFiberDirty checks.
	if parseRt.currentRoot != nil {
		parseRt.currentRoot.alternate = nil
	}

	// Reuse the previous alternate root when available to reduce per-update allocations.
	parseRt.wipRoot = acquireWorkInProgress(parseRt.currentRoot)
	*parseRt.wipRoot = Fiber{
		typeOf:    parseRt.currentRoot.typeOf,
		dom:       parseRt.currentRoot.dom,
		props:     parseRt.currentRoot.props,
		alternate: parseRt.currentRoot,
		dirty:     true,
	}

	parseRt.nextUnitOfWork = parseRt.wipRoot
	// Reuse deletions slice capacity if possible
	if parseRt.deletions == nil {
		parseRt.deletions = make([]*Fiber, 0)
	} else {
		parseRt.deletions = parseRt.deletions[:0]
	}

	// Schedule work loop
	parseContinueWork := parseRt.getContinueWorkFn()
	parseScheduler := parseRt.scheduler
	schedulerMu.Unlock()
	parseScheduler.SetTimeout(parseContinueWork, 0)
}

// continueWorkLoop is a bound method to avoid closure allocation
func (parseRt *Runtime) continueWorkLoop() {
	defer func() {
		if parseRecovered := recover(); parseRecovered != nil {
			parseFiber := parseRt.nextUnitOfWork
			if parseFiber == nil && parseRt.wipRoot != nil {
				parseFiber = parseRt.hydrationDiagnosticFiber(parseRt.wipRoot)
			}
			if _, parseSuppressed := finalizeUnhandledPanicContext("runtime", PanicPhaseDeferred, panicSubject(parseFiber), diagnosticPathForFiber(parseFiber), diagnosticComponentStack(parseFiber), parseRecovered); parseSuppressed {
				return
			}
		}
	}()
	parseRt.workLoop(globalInfiniteDeadline)
}

// workLoop processes work units during idle periods
func (parseRt *Runtime) workLoop(parseDeadline Deadline) {
	parseRt.profiling.workLoopPasses++
	shouldYield := false
	parseUnits := 0
	parseMaxUnitsPerSlice := 300
	if parseDeadline == globalInfiniteDeadline {
		parseMaxUnitsPerSlice = 1200
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
		// More work remains, schedule next iteration
		// fmt.Printf("workLoop: more work remains, scheduling next iteration\n")
		parseRt.scheduler.SetTimeout(parseRt.getContinueWorkFn(), 0)
	}
}

// Render starts rendering a component tree
func (parseRt *Runtime) Render(parseElement *Element, parseContainer DOMNode) {
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

	// Optimization: Break the alternate chain on the current root
	if parseRt.currentRoot != nil {
		parseRt.currentRoot.alternate = nil
	}

	parseRt.wipRoot = acquireWorkInProgress(parseRt.currentRoot)
	*parseRt.wipRoot = Fiber{
		typeOf:    "ROOT",
		dom:       parseContainer,
		props:     map[string]interface{}{"children": []interface{}{parseElement}},
		alternate: parseRt.currentRoot,
		dirty:     true,
	}

	parseRt.nextUnitOfWork = parseRt.wipRoot
	// Reuse deletions slice capacity if possible
	if parseRt.deletions == nil {
		parseRt.deletions = make([]*Fiber, 0)
	} else {
		parseRt.deletions = parseRt.deletions[:0]
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
		parseScheduler.SetTimeout(parseContinueWork, 0)
	}
}

// Hydrate starts a client resume attempt from an existing container.
func (parseRt *Runtime) Hydrate(parseElement *Element, parseContainer DOMNode) {
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
	if parseRt.domAdapter != nil && parseContainer != nil && !parseContainer.IsNull() {
		for parseNode := parseRt.domAdapter.GetFirstChild(parseContainer); parseNode != nil && !parseNode.IsNull(); parseNode = parseRt.domAdapter.GetNextSibling(parseNode) {
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
	*parseRt.wipRoot = Fiber{
		typeOf:    "ROOT",
		dom:       parseContainer,
		props:     map[string]interface{}{"children": []interface{}{parseElement}},
		alternate: parseRt.currentRoot,
		dirty:     true,
		hydration: newHydrationBoundary(parseContainer, parseRt.domAdapter.GetFirstChild(parseContainer)),
	}

	parseRt.nextUnitOfWork = parseRt.wipRoot
	if parseRt.deletions == nil {
		parseRt.deletions = make([]*Fiber, 0)
	} else {
		parseRt.deletions = parseRt.deletions[:0]
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
		parseScheduler.SetTimeout(parseContinueWork, 0)
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
	parseOrigin = normalizeUpdateOrigin(parseOrigin, "hook")
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
		if isParseFirst {
			parseF.updateOrigin = parseOrigin
		} else if parseF.updateOrigin == "" {
			parseF.updateOrigin = "ancestor"
		}
		isParseFirst = false
	}

	// Schedule update from root
	if !parseRt.updateScheduled {
		parseRt.ScheduleUpdate()
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
	parseFiber.dirty = true
	parseFiber.needsUpdate = true
	parseFiber.updateOrigin = normalizeUpdateOrigin(parseOrigin, "fine-grained")
	if !parseRt.updateScheduled {
		parseRt.ScheduleUpdate()
	}
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
	parseRt.ScheduleUpdateForFiberWithOrigin(parseFiber, parseOrigin)
}

// normalizeUpdateOrigin is a core package helper.
func normalizeUpdateOrigin(parseOrigin string, parseFallback string) string {
	parseTrimmed := strings.TrimSpace(parseOrigin)
	if parseTrimmed != "" {
		return parseTrimmed
	}
	return parseFallback
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
	return parseFiber
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

// UI Queue for cross-goroutine updates
type uiQueueItem struct {
	fn func()
}

var (
	uiQueue     = make(chan uiQueueItem, 1024)
	uiQueueInit sync.Once
)

// EnqueueUI adds a function to the UI queue for main-thread execution.
func EnqueueUI(parseFn func()) {
	uiQueueInit.Do(func() {
		// Queue is already initialized
	})

	select {
	case uiQueue <- uiQueueItem{fn: parseFn}:
		// Successfully enqueued
	default:
		// Queue full, execute synchronously (fallback)
		parseFn()
	}
}

// ProcessUIQueue runs pending UI updates until the queue is empty.
func ProcessUIQueue() {
	for {
		select {
		case parseItem := <-uiQueue:
			parseItem.fn()
		default:
			return
		}
	}
}

// GetUIQueueSize returns the current UI queue size.
func GetUIQueueSize() int {
	return len(uiQueue)
}
