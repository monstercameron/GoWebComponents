package runtime

import (
	"sync"
)

var (
	schedulerMu sync.Mutex
	// globalInfiniteDeadline is a shared instance to avoid allocation
	globalInfiniteDeadline = &infiniteDeadline{}
)

type infiniteDeadline struct{}

func (d *infiniteDeadline) TimeRemaining() float64 { return 1000 } // lots of time
func (d *infiniteDeadline) DidTimeout() bool       { return false }

// ScheduleUpdate schedules a full tree update from the root
func (rt *Runtime) ScheduleUpdate() {
	schedulerMu.Lock()
	defer schedulerMu.Unlock()

	if rt.currentRoot == nil || rt.updateScheduled {
		return
	}

	rt.updateScheduled = true

	// Optimization: Break the alternate chain on the current root to prevent memory leaks
	// and long traversals during isFiberDirty checks.
	if rt.currentRoot != nil {
		rt.currentRoot.alternate = nil
	}

	// Create new work-in-progress root
	rt.wipRoot = &Fiber{
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
	rt.scheduler.SetTimeout(rt.continueWorkLoop, 0)
}

// continueWorkLoop is a bound method to avoid closure allocation
func (rt *Runtime) continueWorkLoop() {
	rt.workLoop(globalInfiniteDeadline)
}

// workLoop processes work units during idle periods
func (rt *Runtime) workLoop(deadline Deadline) {
	shouldYield := false
	units := 0
	const maxUnitsPerSlice = 300
	
	// Inline check for common case
	for rt.nextUnitOfWork != nil && !shouldYield {
		rt.nextUnitOfWork = rt.performUnitOfWork(rt.nextUnitOfWork)
		units++

		// Check if we should yield
		if deadline.TimeRemaining() < 1 || units >= maxUnitsPerSlice {
			shouldYield = true
		}
	}

	// If work is complete, commit
	if rt.wipRoot != nil && rt.nextUnitOfWork == nil {
		rt.commitRoot()
		rt.updateScheduled = false
	} else if rt.nextUnitOfWork != nil {
		// More work remains, schedule next iteration
		// fmt.Printf("workLoop: more work remains, scheduling next iteration\n")
		rt.scheduler.SetTimeout(rt.continueWorkLoop, 0)
	}
}

// Render starts rendering a component tree
func (rt *Runtime) Render(element *Element, container DOMNode) {
	// Optimization: Break the alternate chain on the current root
	if rt.currentRoot != nil {
		rt.currentRoot.alternate = nil
	}

	rt.wipRoot = &Fiber{
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
	rt.scheduler.SetTimeout(rt.continueWorkLoop, 0)
}

// ScheduleUpdateForFiber schedules an update for a specific fiber
func (rt *Runtime) ScheduleUpdateForFiber(fiber *Fiber) {
	if fiber == nil {
		return
	}

	// Mark fiber and parents as dirty
	f := fiber
	for f != nil {
		// Optimization: if fiber is already dirty and marked for update,
		// we can assume the path to root is already marked.
		if f.dirty && f.needsUpdate {
			break
		}
		f.dirty = true
		f.needsUpdate = true
		f = f.parent
	}

	// Schedule update from root
	if !rt.updateScheduled {
		rt.ScheduleUpdate()
	}
}

// UI Queue for cross-goroutine updates
type uiQueueItem struct {
	fn func()
}

var (
	uiQueue     = make(chan uiQueueItem, 1024)
	uiQueueInit sync.Once
)

// EnqueueUI adds a function to the UI queue for main thread execution
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

// ProcessUIQueue processes pending UI updates
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

// GetUIQueueSize returns the current UI queue size
func GetUIQueueSize() int {
	return len(uiQueue)
}
