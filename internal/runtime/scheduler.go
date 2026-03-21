package runtime

import (
	"fmt"
	"sync"
	"time"
)

var (
	schedulerMu sync.Mutex
	// globalInfiniteDeadline is a shared instance to avoid allocation
	globalInfiniteDeadline = &infiniteDeadline{}
)

type infiniteDeadline struct{}

func (d *infiniteDeadline) TimeRemaining() float64 { return 1000 } // lots of time
func (d *infiniteDeadline) DidTimeout() bool       { return false }

func (rt *Runtime) getContinueWorkFn() func() {
	if rt.continueWorkFn == nil {
		rt.continueWorkFn = rt.continueWorkLoop
	}
	return rt.continueWorkFn
}

// ScheduleUpdate schedules a full tree update from the root
func (rt *Runtime) ScheduleUpdate() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	rt.profiling.scheduledRootUpdates++

	if rt.currentRoot == nil || rt.updateScheduled {
		return
	}

	rt.updateScheduled = true

	// Optimization: Break the alternate chain on the current root to prevent memory leaks
	// and long traversals during isFiberDirty checks.
	if rt.currentRoot != nil {
		rt.currentRoot.alternate = nil
	}

	// Reuse the previous alternate root when available to reduce per-update allocations.
	rt.wipRoot = acquireWorkInProgress(rt.currentRoot)
	*rt.wipRoot = Fiber{
		typeOf:    rt.currentRoot.typeOf,
		dom:       rt.currentRoot.dom,
		props:     rt.currentRoot.props,
		alternate: rt.currentRoot,
		dirty:     true,
	}

	rt.nextUnitOfWork = rt.wipRoot
	// Reuse deletions slice capacity if possible
	if rt.deletions == nil {
		rt.deletions = make([]*Fiber, 0)
	} else {
		rt.deletions = rt.deletions[:0]
	}

	// Schedule work loop
	rt.scheduler.SetTimeout(rt.getContinueWorkFn(), 0)
}

// continueWorkLoop is a bound method to avoid closure allocation
func (rt *Runtime) continueWorkLoop() {
	defer func() {
		if recovered := recover(); recovered != nil {
			fiber := rt.nextUnitOfWork
			if fiber == nil && rt.wipRoot != nil {
				fiber = rt.hydrationDiagnosticFiber(rt.wipRoot)
			}
			if _, suppressed := finalizeUnhandledPanicContext("runtime", PanicPhaseDeferred, panicSubject(fiber), diagnosticPathForFiber(fiber), diagnosticComponentStack(fiber), recovered); suppressed {
				return
			}
		}
	}()
	rt.workLoop(globalInfiniteDeadline)
}

// workLoop processes work units during idle periods
func (rt *Runtime) workLoop(deadline Deadline) {
	rt.profiling.workLoopPasses++
	shouldYield := false
	units := 0
	maxUnitsPerSlice := 300
	if deadline == globalInfiniteDeadline {
		maxUnitsPerSlice = 1200
	}

	// Inline check for common case
	for rt.nextUnitOfWork != nil && !shouldYield {
		rt.nextUnitOfWork = rt.performUnitOfWork(rt.nextUnitOfWork)
		units++
		rt.profiling.processedUnits++

		// Check if we should yield
		if deadline.TimeRemaining() < 1 || units >= maxUnitsPerSlice {
			shouldYield = true
		}
	}

	// If work is complete, commit
	if rt.wipRoot != nil && rt.nextUnitOfWork == nil {
		rt.commitRoot()
	} else if rt.nextUnitOfWork != nil {
		// More work remains, schedule next iteration
		// fmt.Printf("workLoop: more work remains, scheduling next iteration\n")
		rt.scheduler.SetTimeout(rt.getContinueWorkFn(), 0)
	}
}

// Render starts rendering a component tree
func (rt *Runtime) Render(element *Element, container DOMNode) {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	start := time.Now()
	defer func() {
		rt.profiling.renderCalls++
		rt.profiling.lastRenderDurationNs = time.Since(start).Nanoseconds()
	}()

	shouldSchedule := !rt.updateScheduled
	rt.updateScheduled = true

	// Optimization: Break the alternate chain on the current root
	if rt.currentRoot != nil {
		rt.currentRoot.alternate = nil
	}

	rt.wipRoot = acquireWorkInProgress(rt.currentRoot)
	*rt.wipRoot = Fiber{
		typeOf:    "ROOT",
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{element}},
		alternate: rt.currentRoot,
		dirty:     true,
	}

	rt.nextUnitOfWork = rt.wipRoot
	// Reuse deletions slice capacity if possible
	if rt.deletions == nil {
		rt.deletions = make([]*Fiber, 0)
	} else {
		rt.deletions = rt.deletions[:0]
	}
	if shouldSchedule {
		rt.scheduler.SetTimeout(rt.getContinueWorkFn(), 0)
	}
}

// Hydrate starts a client resume attempt from an existing container.
func (rt *Runtime) Hydrate(element *Element, container DOMNode) {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()
	start := time.Now()
	defer func() {
		rt.profiling.renderCalls++
		rt.profiling.lastRenderDurationNs = time.Since(start).Nanoseconds()
	}()

	existingChildren := 0
	if rt.domAdapter != nil && container != nil && !container.IsNull() {
		for node := rt.domAdapter.GetFirstChild(container); node != nil && !node.IsNull(); node = rt.domAdapter.GetNextSibling(node) {
			existingChildren++
		}
	}

	if existingChildren > 0 {
		ReportDiagnostic("runtime", DiagnosticInfo, fmt.Sprintf("Hydrate found %d existing container child nodes and will attempt DOM reuse before falling back per subtree", existingChildren))
	} else {
		ReportDiagnostic("runtime", DiagnosticInfo, "Hydrate found no existing container children and is proceeding with a fresh client render")
	}

	shouldSchedule := !rt.updateScheduled
	rt.updateScheduled = true
	rt.hydrating = true
	rt.strictHydration = rt.nextHydrationStrict
	rt.beginHydrationMetrics(existingChildren, rt.nextHydrationStrict)
	rt.nextHydrationStrict = false
	rt.deferredHydrationSubscriptions = rt.deferredHydrationSubscriptions[:0]
	if rt.deferredHydrationUpdates == nil {
		rt.deferredHydrationUpdates = make(map[*Fiber]bool)
	} else {
		clear(rt.deferredHydrationUpdates)
	}

	if rt.currentRoot != nil {
		rt.currentRoot.alternate = nil
	}

	rt.wipRoot = acquireWorkInProgress(rt.currentRoot)
	*rt.wipRoot = Fiber{
		typeOf:    "ROOT",
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{element}},
		alternate: rt.currentRoot,
		dirty:     true,
		hydration: newHydrationBoundary(container, rt.domAdapter.GetFirstChild(container)),
	}

	rt.nextUnitOfWork = rt.wipRoot
	if rt.deletions == nil {
		rt.deletions = make([]*Fiber, 0)
	} else {
		rt.deletions = rt.deletions[:0]
	}
	if shouldSchedule {
		rt.scheduler.SetTimeout(rt.getContinueWorkFn(), 0)
	}
}

// ScheduleUpdateForFiber schedules an update for a specific fiber
func (rt *Runtime) ScheduleUpdateForFiber(fiber *Fiber) {
	if fiber == nil {
		return
	}
	if rt.hydrating {
		if rt.deferredHydrationUpdates == nil {
			rt.deferredHydrationUpdates = make(map[*Fiber]bool)
		}
		rt.deferredHydrationUpdates[fiber] = true
		return
	}
	rt.profiling.scheduledFiberMarks++
	fiber.updateOrigin = "hook"

	// Mark fiber and parents as dirty
	first := true
	for f := fiber; f != nil; f = f.parent {
		// Optimization: if fiber is already dirty and marked for update,
		// we can assume the path above it is already marked.
		// The originating fiber itself cannot short-circuit ancestor marking.
		if !first && f.dirty && f.needsUpdate {
			break
		}
		f.dirty = true
		f.needsUpdate = true
		if first {
			f.updateOrigin = "hook"
		} else if f.updateOrigin == "" {
			f.updateOrigin = "ancestor"
		}
		first = false
	}

	// Schedule update from root
	if !rt.updateScheduled {
		rt.ScheduleUpdate()
	}
}

// ScheduleGranularUpdateForFiber marks only the target fiber dirty and lets
// clean ancestors clone through to the dirty descendant on the next root pass.
func (rt *Runtime) ScheduleGranularUpdateForFiber(fiber *Fiber) {
	if fiber == nil {
		return
	}
	if rt.hydrating {
		if rt.deferredHydrationUpdates == nil {
			rt.deferredHydrationUpdates = make(map[*Fiber]bool)
		}
		rt.deferredHydrationUpdates[fiber] = true
		return
	}
	rt.profiling.scheduledFiberMarks++
	rt.profiling.scheduledGranularMarks++
	fiber.dirty = true
	fiber.needsUpdate = true
	fiber.updateOrigin = "fine-grained"
	if !rt.updateScheduled {
		rt.ScheduleUpdate()
	}
}

// ScheduleSubscribedFiberUpdate chooses the narrowest safe scheduling path for a subscription target.
func (rt *Runtime) ScheduleSubscribedFiberUpdate(fiber *Fiber) {
	if fiber == nil {
		return
	}
	fiber = rt.resolveSubscribedFiberTarget(fiber)
	if fiber == nil {
		return
	}
	if fiber.fineGrained {
		rt.ScheduleGranularUpdateForFiber(fiber)
		return
	}
	rt.ScheduleUpdateForFiber(fiber)
}

func (rt *Runtime) resolveSubscribedFiberTarget(fiber *Fiber) *Fiber {
	if rt == nil || fiber == nil {
		return fiber
	}
	if rt.isFiberInCurrentTree(fiber) {
		return fiber
	}
	if fiber.alternate != nil && rt.isFiberInCurrentTree(fiber.alternate) {
		return fiber.alternate
	}
	return fiber
}

func (rt *Runtime) isFiberInCurrentTree(fiber *Fiber) bool {
	if rt == nil || fiber == nil || rt.currentRoot == nil {
		return false
	}
	root := fiber
	for root.parent != nil {
		root = root.parent
	}
	return root == rt.currentRoot
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
func EnqueueUI(fn func()) {
	uiQueueInit.Do(func() {
		// Queue is already initialized
	})

	select {
	case uiQueue <- uiQueueItem{fn: fn}:
		// Successfully enqueued
	default:
		// Queue full, execute synchronously (fallback)
		fn()
	}
}

// ProcessUIQueue runs pending UI updates until the queue is empty.
func ProcessUIQueue() {
	for {
		select {
		case item := <-uiQueue:
			item.fn()
		default:
			return
		}
	}
}

// GetUIQueueSize returns the current UI queue size.
func GetUIQueueSize() int {
	return len(uiQueue)
}
