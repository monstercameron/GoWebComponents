// ./fiber/fiber.go

//go:build js && wasm
// +build js,wasm

package fiber

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"syscall/js"
	"time"
)

// Global variables for tracking the current fiber and root.
var (
	wipRoot         *Fiber
	currentRoot     *Fiber
	nextUnitOfWork  *Fiber
	deletions       []*Fiber
	wipFiber        *Fiber
	updateScheduled bool // Flag to prevent multiple update scheduling

	// --- UI queue for main-thread safe updates with optimized buffer sizing ---
	uiQueue         chan func()
	uiQueueSize     int   = 1024 // Default buffer size
	uiQueueMaxSize  int   = 4096 // Maximum buffer size for dynamic growth
	uiQueueMinSize  int   = 256  // Minimum buffer size for dynamic shrinking
	uiQueueGrowth   int64        // Counter for queue growth events
	uiQueueShrinks  int64        // Counter for queue shrink events
	
	// UI queue monitoring
	uiQueueOverflows int64 // Counter for overflow events
	uiQueueMaxReached int64 // Track maximum queue size reached

	// Indicates we're executing on the main scheduler/commit/effect loop
	schedulerActive int32

	// Debug logging toggle - start with false so namespace control works
	debugEnabled = false

	// Performance tracking
	renderStartTime time.Time
	totalRenders    int64
	totalWorkUnits  int64
)

// Initialize UI queue with optimized buffer size
func init() {
	initializeUIQueue()
}

// initializeUIQueue creates the UI queue with optimal buffer size
func initializeUIQueue() {
	uiQueue = make(chan func(), uiQueueSize)
	debugf("FIBER", "🔧 initializeUIQueue: created UI queue with buffer size %d\n", uiQueueSize)
}

// resizeUIQueue dynamically adjusts the UI queue buffer size based on usage patterns
func resizeUIQueue(newSize int) {
	if newSize < uiQueueMinSize {
		newSize = uiQueueMinSize
	}
	if newSize > uiQueueMaxSize {
		newSize = uiQueueMaxSize
	}
	
	if newSize == uiQueueSize {
		return // No change needed
	}
	
	oldSize := uiQueueSize
	oldQueue := uiQueue
	
	// Create new queue with optimized size
	newQueue := make(chan func(), newSize)
	
	// Transfer existing items to new queue
	transferred := 0
	for {
		select {
		case fn := <-oldQueue:
			select {
			case newQueue <- fn:
				transferred++
			default:
				// New queue is full, put item back and stop
				select {
				case oldQueue <- fn:
				default:
					// Both queues full, execute immediately to prevent loss
					fn()
				}
				goto transferComplete
			}
		default:
			goto transferComplete
		}
	}
	
transferComplete:
	uiQueue = newQueue
	uiQueueSize = newSize
	
	if newSize > oldSize {
		atomic.AddInt64(&uiQueueGrowth, 1)
		debugf("FIBER", "📈 resizeUIQueue: grew UI queue %d→%d (transferred %d items)\n", oldSize, newSize, transferred)
	} else {
		atomic.AddInt64(&uiQueueShrinks, 1)
		debugf("FIBER", "📉 resizeUIQueue: shrunk UI queue %d→%d (transferred %d items)\n", oldSize, newSize, transferred)
	}
}

// optimizeUIQueueSize analyzes usage patterns and adjusts buffer size
func optimizeUIQueueSize() {
	// Skip optimization if auto-optimization is disabled
	if !uiQueueAutoOptimization {
		return
	}
	
	currentLen := len(uiQueue)
	currentCap := cap(uiQueue)
	utilizationPercent := float64(currentLen) / float64(currentCap) * 100
	
	debugf("FIBER", "📊 optimizeUIQueueSize: current=%d, capacity=%d, utilization=%.1f%%\n", 
		currentLen, currentCap, utilizationPercent)
	
	// Grow if utilization is consistently high
	if utilizationPercent > 80 && currentCap < uiQueueMaxSize {
		newSize := currentCap * 2
		if newSize > uiQueueMaxSize {
			newSize = uiQueueMaxSize
		}
		resizeUIQueue(newSize)
	}
	
	// Shrink if utilization is consistently low
	if utilizationPercent < 20 && currentCap > uiQueueMinSize {
		newSize := currentCap / 2
		if newSize < uiQueueMinSize {
			newSize = uiQueueMinSize
		}
		resizeUIQueue(newSize)
	}
}

func scheduleUpdateAtRoot() {
	debugf("FIBER", "🎯 scheduleUpdateAtRoot called - currentRoot: %p, updateScheduled: %v\n", currentRoot, updateScheduled)

	if currentRoot == nil || updateScheduled {
		if currentRoot == nil {
			debugf("FIBER", "🚨 scheduleUpdateAtRoot: currentRoot is nil, aborting\n")
		} else {
			debugf("FIBER", "⏭️ scheduleUpdateAtRoot: update already scheduled, skipping\n")
		}
		return
	}
	updateScheduled = true
	debugf("FIBER", "✅ scheduleUpdateAtRoot: update scheduled\n")

	// Reuse fiber instead of allocating new one
	if wipRoot == nil {
		debugf("FIBER", "🔧 scheduleUpdateAtRoot: getting fiber from pool\n")
		poolFiber := fiberPool.Get()
		if fiber, ok := poolFiber.(*Fiber); ok {
			// Decrement pool size counter when retrieving from pool
			atomic.AddInt32(&poolSizes.fiber, -1)
			wipRoot = fiber
			debugf("FIBER", "♻️ scheduleUpdateAtRoot: reused fiber from pool %p\n", wipRoot)
		} else {
			// This should never happen if pool is properly initialized, but handle gracefully
			debugf("FIBER", "🚨 scheduleUpdateAtRoot: fiberPool returned unexpected type %T, creating new Fiber\n", poolFiber)
			wipRoot = &Fiber{
				props: make(map[string]interface{}),
			}
		}
	}

	// Debug: Log fiber setup
	debugf("FIBER", "🔧 scheduleUpdateAtRoot: setting up wipRoot from currentRoot\n")
	debugf("FIBER", "  📋 currentRoot type: %v, dom: %v\n", currentRoot.typeOf, currentRoot.dom.Type())

	wipRoot.typeOf = currentRoot.typeOf
	wipRoot.dom = currentRoot.dom
	wipRoot.props = currentRoot.props
	wipRoot.alternate = currentRoot
	wipRoot.effectTag = ""
	wipRoot.parent = nil
	wipRoot.child = nil
	wipRoot.sibling = nil

	nextUnitOfWork = wipRoot
	
	// Smart slice management: shrink backing array if it grew too large
	const maxDeletionsCapacity = 64 // Reasonable upper bound for most apps
	const defaultDeletionsCapacity = 16 // Default capacity for new slice
	
	if cap(deletions) > maxDeletionsCapacity {
		// Backing array is too large, create new slice with reasonable capacity
		oldCapacity := cap(deletions)
		deletions = make([]*Fiber, 0, defaultDeletionsCapacity)
		debugf("FIBER", "🧹 scheduleUpdateAtRoot: shrunk deletions slice capacity %d→%d\n", oldCapacity, defaultDeletionsCapacity)
	} else {
		// Reuse existing slice
		deletions = deletions[:0]
	}

	debugf("FIBER", "🚀 scheduleUpdateAtRoot: requesting idle callback for workLoop\n")
	requestIdleCallback(workLoop)
}

// getCurrentFiber retrieves the current working fiber.
func getCurrentFiber() *Fiber {
	debugf("FIBER", "🔍 getCurrentFiber called, returning: %p\n", wipFiber)
	return wipFiber
}

// scheduleUpdate triggers a re-render of the component.
func scheduleUpdate(fiber *Fiber) {
	debugf("FIBER", "🎯 scheduleUpdate called for fiber %p (type: %v)\n", fiber, fiber.typeOf)
	// fmt.Println("scheduleUpdate: Scheduling update")
	wipRoot = &Fiber{
		typeOf:    "ROOT",
		dom:       currentRoot.dom,
		props:     currentRoot.props,
		alternate: currentRoot,
	}
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	debugf("FIBER", "✅ scheduleUpdate: wipRoot set %p, scheduling workLoop\n", wipRoot)
	// fmt.Println("scheduleUpdate: wipRoot set and workLoop scheduled")
	requestIdleCallback(workLoop)
}

// render starts the rendering process.
func render(element *Element, container js.Value) {
	debugf("FIBER", "🎯 render called with element type: %v, container: %v\n", element.Type, container.Type())
	renderStartTime = time.Now()
	totalRenders++

	// fmt.Println("render: Starting rendering process.")
	wipRoot = &Fiber{
		typeOf:    "ROOT", // Assign a type to the root fiber
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{element}},
		alternate: currentRoot,
	}
	debugf("RENDER", "Root fiber created %p.\n", wipRoot)
	debugf("FIBER", "📊 render: total renders so far: %d\n", totalRenders)

	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	debugf("FIBER", "🚀 render: scheduling work loop\n")
	// fmt.Println("render: Scheduling work loop.")
	requestIdleCallback(workLoop)
}

// Render is the exported version of render - starts the rendering process
func Render(element *Element, container js.Value) {
	debugf("FIBER", "🎯 Render (exported) called\n")
	render(element, container)
}

// CreateElement is the exported version of createElement
func CreateElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	debugf("FIBER", "🎯 CreateElement called with type: %v, props: %+v, children: %d\n", typ, props, len(children))
	element := createElement(typ, props, children...)
	debugf("FIBER", "✅ CreateElement: created element %p\n", element)
	return element
}

// workLoop performs work until there is no more work left or the deadline expires.
func workLoop(deadline js.Value) {
	debugf("FIBER", "🎯 workLoop started with deadline\n")
	workStartTime := time.Now()
	startSchedulerSection()
	defer endSchedulerSection()

	const maxUnitsPerSlice = 300 // Prevent long monopolisation of idle period
	units := 0

	var shouldYield bool
	for nextUnitOfWork != nil {
		debugf("FIBER", "🔄 workLoop: processing unit %d, fiber %p (type: %v)\n",
			units+1, nextUnitOfWork, nextUnitOfWork.typeOf)

		nextUnitOfWork = performUnitOfWork(nextUnitOfWork)
		totalWorkUnits++

		units++
		// Yield criteria: low time remaining OR processed many units already
		timeRemaining := deadline.Call("timeRemaining").Float()
		if timeRemaining < 1 || units >= maxUnitsPerSlice {
			shouldYield = true
			debugf("FIBER", "⏱️ workLoop: yielding after %d units (time remaining: %.2fms)\n", units, timeRemaining)
		}

		if shouldYield {
			break
		}
	}

	workDuration := time.Since(workStartTime)
	debugf("FIBER", "📊 workLoop: processed %d units in %v (total work units: %d)\n",
		units, workDuration, totalWorkUnits)

	if wipRoot != nil && nextUnitOfWork == nil {
		debugf("FIBER", "🏁 workLoop: no more work, committing root\n")
		// fmt.Println("workLoop: No more units of work. Committing root.")
		commitRoot()
	}

	if nextUnitOfWork != nil {
		debugf("FIBER", "⏳ workLoop: work remains, scheduling next iteration\n")
		// fmt.Println("workLoop: Work remains. Scheduling next work loop.")
		requestIdleCallback(workLoop)
	} else {
		renderDuration := time.Since(renderStartTime)
		debugf("FIBER", "🎉 workLoop: all work completed in %v\n", renderDuration)
		updateScheduled = false
		// fmt.Println("workLoop: All work completed.")
	}

	// First, handle any queued UI tasks from background goroutines
	debugf("FIBER", "📋 workLoop: processing UI queue\n")
	processUIQueue()
}

// performUnitOfWork performs a single unit of work.
func performUnitOfWork(fiber *Fiber) *Fiber {
	if fiber == nil {
		debugf("FIBER", "🚨 performUnitOfWork: received nil fiber\n")
		// fmt.Println("performUnitOfWork: Fiber is nil.")
		return nil
	}

	debugf("FIBER", "🔄 performUnitOfWork: processing fiber %p (type: %v)\n", fiber, fiber.typeOf)

	// Memory diagnostics for this fiber
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	debugf("FIBER", "💾 performUnitOfWork: memory - heap objects: %d, allocs: %d, sys: %d KB\n",
		memStats.HeapObjects, memStats.Mallocs-memStats.Frees, memStats.Sys/1024)

	// fmt.Printf("performUnitOfWork: Processing fiber of type %v.\n", fiber.typeOf)

	if fiber.typeOf == nil || fiber.typeOf == "ROOT" {
		debugf("FIBER", "🌳 performUnitOfWork: processing ROOT fiber, reconciling children\n")
		// fmt.Println("performUnitOfWork: Fiber has typeOf nil or ROOT, reconciling children.")
		if children, ok := fiber.props["children"].([]interface{}); ok {
			debugf("FIBER", "📋 performUnitOfWork: found %d children to reconcile\n", len(children))
			reconcileChildren(fiber, children)
		} else {
			// Handle case where children is not the expected type
			debugf("FIBER", "🚨 performUnitOfWork: fiber.props[\"children\"] is not []interface{}, got %T\n", fiber.props["children"])
			// Try to reconcile with empty children to avoid crash
			emptyChildren := make([]interface{}, 0)
			reconcileChildren(fiber, emptyChildren)
		}
	} else {
		switch fiber.typeOf.(type) {
		case func(map[string]interface{}) *Element:
			debugf("FIBER", "🧩 performUnitOfWork: processing function component\n")
			// Function component with map[string]interface{} props
			componentFunc, ok := fiber.typeOf.(func(map[string]interface{}) *Element)
			if !ok {
				debugf("FIBER", "🚨 performUnitOfWork: fiber.typeOf is not func(map[string]interface{}) *Element, got %T\n", fiber.typeOf)
				return nil
			}
			wipFiber = fiber
			debugf("FIBER", "🎯 performUnitOfWork: set wipFiber to %p\n", wipFiber)

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil {
				oldHooks = fiber.alternate.hooks
				debugf("FIBER", "♻️ performUnitOfWork: found existing hooks %p from alternate\n", oldHooks)
			}

			if oldHooks != nil {
				// Reuse existing Hooks instance and reset per-render state
				debugf("FIBER", "🔄 performUnitOfWork: reusing hooks, resetting state\n")
				wipFiber.hooks = oldHooks
				wipFiber.hooks.index = 0
				wipFiber.hooks.callOrder = wipFiber.hooks.callOrder[:0]
				wipFiber.hooks.orderChecked = false
				debugf("FIBER", "✅ performUnitOfWork: hooks reset - index: %d, callOrder len: %d\n",
					wipFiber.hooks.index, len(wipFiber.hooks.callOrder))
				// prevOrder already contains the last render's sequence
			} else {
				// First render – allocate a fresh Hooks container
				debugf("FIBER", "🆕 performUnitOfWork: creating new hooks container\n")
				wipFiber.hooks = &Hooks{
					state:        []interface{}{},
					deps:         [][]interface{}{},
					memos:        []memoizedValue{},
					callOrder:    []HookCall{},
					prevOrder:    []HookCall{},
					orderChecked: false,
					index:        0,
				}
				debugf("FIBER", "✅ performUnitOfWork: new hooks created %p\n", wipFiber.hooks)
			}

			// Clear effects for this render
			wipFiber.effects = wipFiber.effects[:0]
			debugf("FIBER", "🧹 performUnitOfWork: cleared effects\n")

			// Call the component function
			debugf("FIBER", "🎬 performUnitOfWork: calling component function with props: %+v\n", fiber.props)
			componentStartTime := time.Now()
			element := componentFunc(fiber.props)
			componentDuration := time.Since(componentStartTime)
			debugf("FIBER", "⚡ performUnitOfWork: component function completed in %v, returned: %p\n",
				componentDuration, element)

			// Finalize hook order validation
			if err := finalizeHookOrder(wipFiber.hooks); err != nil {
				debugf("FIBER", "🚨 performUnitOfWork: hook order validation failed: %v\n", err)
			}

			if element != nil {
				debugf("FIBER", "📋 performUnitOfWork: reconciling component children\n")
				reconcileChildren(fiber, []interface{}{element})
			} else {
				debugf("FIBER", "🚨 performUnitOfWork: component returned nil element\n")
				reconcileChildren(fiber, []interface{}{})
			}

		case func(Attrs) *Element:
			debugf("FIBER", "🧩 performUnitOfWork: processing Attrs component\n")
			// Function component with Attrs props
			componentFunc, ok := fiber.typeOf.(func(Attrs) *Element)
			if !ok {
				debugf("FIBER", "🚨 performUnitOfWork: fiber.typeOf is not func(Attrs) *Element, got %T\n", fiber.typeOf)
				return nil
			}
			wipFiber = fiber
			debugf("FIBER", "🎯 performUnitOfWork: set wipFiber to %p\n", wipFiber)

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil {
				oldHooks = fiber.alternate.hooks
				debugf("FIBER", "♻️ performUnitOfWork: found existing hooks %p from alternate\n", oldHooks)
			}

			if oldHooks != nil {
				// Reuse existing Hooks instance and reset per-render state
				debugf("FIBER", "🔄 performUnitOfWork: reusing hooks, resetting state\n")
				wipFiber.hooks = oldHooks
				wipFiber.hooks.index = 0
				wipFiber.hooks.callOrder = wipFiber.hooks.callOrder[:0]
				wipFiber.hooks.orderChecked = false
				debugf("FIBER", "✅ performUnitOfWork: hooks reset - index: %d, callOrder len: %d\n",
					wipFiber.hooks.index, len(wipFiber.hooks.callOrder))
				// prevOrder already contains the last render's sequence
			} else {
				// First render – allocate a fresh Hooks container
				debugf("FIBER", "🆕 performUnitOfWork: creating new hooks container\n")
				wipFiber.hooks = &Hooks{
					state:        []interface{}{},
					deps:         [][]interface{}{},
					memos:        []memoizedValue{},
					prevOrder:    []HookCall{},
					callOrder:    []HookCall{},
					orderChecked: false,
					index:        0,
				}
				debugf("FIBER", "✅ performUnitOfWork: new hooks created %p\n", wipFiber.hooks)
			}

			// Clear effects for this render
			wipFiber.effects = wipFiber.effects[:0]
			debugf("FIBER", "🧹 performUnitOfWork: cleared effects\n")

			// Convert map[string]interface{} to Attrs
			var attrs Attrs
			if fiber.props != nil {
				attrs = Attrs(fiber.props)
				debugf("FIBER", "🔄 performUnitOfWork: converted props to Attrs: %+v\n", attrs)
			}

			// Call the component function
			debugf("FIBER", "🎬 performUnitOfWork: calling Attrs component function\n")
			componentStartTime := time.Now()
			element := componentFunc(attrs)
			componentDuration := time.Since(componentStartTime)
			debugf("FIBER", "⚡ performUnitOfWork: Attrs component completed in %v, returned: %p\n",
				componentDuration, element)

			// Finalize hook order validation after component execution
			if err := finalizeHookOrder(wipFiber.hooks); err != nil {
				componentName := getFunctionName(fiber.typeOf)
				debugf("FIBER", "🚨 performUnitOfWork: hook validation failed for '%s': %v\n", componentName, err)
			}

			if element != nil {
				debugf("FIBER", "📋 performUnitOfWork: reconciling Attrs component children\n")
				reconcileChildren(fiber, []interface{}{element})
			} else {
				debugf("FIBER", "🚨 performUnitOfWork: Attrs component returned nil element\n")
				reconcileChildren(fiber, []interface{}{})
			}

		case string:
			debugf("FIBER", "🏷️ performUnitOfWork: processing DOM element: %v\n", fiber.typeOf)
			// Host component (HTML element)
			// fmt.Printf("performUnitOfWork: Handling host component of type '%s'.\n", fiber.typeOf.(string))
			if fiber.dom.IsUndefined() || fiber.dom.IsNull() {
				debugf("FIBER", "🔧 performUnitOfWork: creating DOM element\n")
				// fmt.Println("performUnitOfWork: Creating DOM node for host component.")
				fiber.dom = createDom(fiber)
				debugf("FIBER", "✅ performUnitOfWork: DOM element created: %v\n", fiber.dom.Type())
				// fmt.Println("performUnitOfWork: DOM node created.")
			}

			if fiber.props == nil {
				debugf("FIBER", "🚨 performUnitOfWork: fiber props are nil, skipping children\n")
				// fmt.Println("performUnitOfWork: Fiber props are nil. Skipping children reconciliation.")
				return getNextUnitOfWork(fiber)
			}

			if propsChildren, ok := fiber.props["children"]; ok {
				debugf("FIBER", "📋 performUnitOfWork: reconciling DOM children\n")
				// fmt.Println("performUnitOfWork: Reconciling children of host component.")
				if elements, elementsOk := propsChildren.([]interface{}); elementsOk {
					debugf("FIBER", "📋 performUnitOfWork: found %d DOM children to reconcile\n", len(elements))
					reconcileChildren(fiber, elements)
				} else {
					debugf("FIBER", "🚨 performUnitOfWork: fiber.props[\"children\"] is not []interface{}, got %T\n", propsChildren)
					// Try to reconcile with empty children to avoid crash
					emptyChildren := make([]interface{}, 0)
					reconcileChildren(fiber, emptyChildren)
				}
			} else {
				debugf("FIBER", "📋 performUnitOfWork: no children found in props\n")
			}

		default:
			debugf("FIBER", "❓ performUnitOfWork: unhandled fiber type %T\n", fiber.typeOf)
			// Robust error handling: mark for deletion, cleanup, and skip
			fiber.effectTag = "DELETION"
			deletions = append(deletions, fiber)
			debugf("FIBER", "🗑️ performUnitOfWork: marked unhandled fiber for deletion and cleanup\n")
			return nil
		}
	}

	// fmt.Printf("performUnitOfWork: Completed processing fiber of type %v.\n", fiber.typeOf)

	// Return next unit of work
	nextWork := getNextUnitOfWork(fiber)
	debugf("FIBER", "➡️ performUnitOfWork: next unit of work: %p\n", nextWork)
	return nextWork
}

// getNextUnitOfWork determines the next fiber to process
func getNextUnitOfWork(fiber *Fiber) *Fiber {
	debugf("FIBER", "🔍 getNextUnitOfWork: finding next work for fiber %p\n", fiber)

	// If this fiber has a child, return it
	if fiber.child != nil {
		debugf("FIBER", "👶 getNextUnitOfWork: returning child %p\n", fiber.child)
		return fiber.child
	}

	// Walk up the fiber tree to find the next sibling
	nextFiber := fiber
	for nextFiber != nil {
		if nextFiber.sibling != nil {
			debugf("FIBER", "👫 getNextUnitOfWork: returning sibling %p\n", nextFiber.sibling)
			return nextFiber.sibling
		}
		debugf("FIBER", "⬆️ getNextUnitOfWork: moving up to parent %p\n", nextFiber.parent)
		nextFiber = nextFiber.parent
	}

	debugf("FIBER", "🏁 getNextUnitOfWork: no more work found\n")
	return nil
}

// createDom moved to dom.go

// reconcileChildren reconciles the children of a fiber.
func reconcileChildren(wipFiber *Fiber, elements []interface{}) {
	debugf("FIBER", "🔄 reconcileChildren: reconciling %d children for fiber %p (type: %v)\n",
		len(elements), wipFiber, wipFiber.typeOf)
	reconcileStartTime := time.Now()

	// fmt.Printf("reconcileChildren: Reconciling %d children for fiber type %v\n", len(elements), wipFiber.typeOf)
	index := 0
	var oldFiber *Fiber
	if wipFiber.alternate != nil {
		oldFiber = wipFiber.alternate.child
		debugf("FIBER", "♻️ reconcileChildren: found existing child %p from alternate\n", oldFiber)
	} else {
		debugf("FIBER", "🆕 reconcileChildren: no alternate, creating fresh children\n")
	}
	var prevSibling *Fiber

	for index < len(elements) || oldFiber != nil {
		debugf("FIBER", "🔄 reconcileChildren: processing index %d (elements: %d, oldFiber: %p)\n",
			index, len(elements), oldFiber)

		var element interface{}
		if index < len(elements) {
			element = elements[index]
			debugf("FIBER", "📋 reconcileChildren: found element at index %d: %+v\n", index, element)
		}

		var newFiber *Fiber

		sameType := false
		if oldFiber != nil && element != nil {
			debugf("FIBER", "🔍 reconcileChildren: comparing types for reuse - old: %v, new: %v\n",
				oldFiber.typeOf, func() interface{} {
					if elem, ok := element.(*Element); ok {
						return elem.Type
					}
					return "unknown"
				}())
			if elem, ok := element.(*Element); ok {
				switch elemType := elem.Type.(type) {
				case func(map[string]interface{}) *Element:
					// Function component with map[string]interface{} props: Compare function pointers using reflect
					funcPtrNew := reflect.ValueOf(elemType).Pointer()
					funcPtrOld, ok := oldFiber.typeOf.(func(map[string]interface{}) *Element)
					if ok {
						funcPtrOldValue := reflect.ValueOf(funcPtrOld).Pointer()
						if funcPtrNew == funcPtrOldValue {
							sameType = true
						}
					}
				case func(Attrs) *Element:
					// Function component with Attrs props: Compare function pointers using reflect
					funcPtrNew := reflect.ValueOf(elemType).Pointer()
					funcPtrOld, ok := oldFiber.typeOf.(func(Attrs) *Element)
					if ok {
						funcPtrOldValue := reflect.ValueOf(funcPtrOld).Pointer()
						if funcPtrNew == funcPtrOldValue {
							sameType = true
						}
					}
				case string:
					// Host component: Use reflect.DeepEqual for string comparison
					if reflect.DeepEqual(elem.Type, oldFiber.typeOf) {
						sameType = true
					}
				default:
					// Other types: Use reflect.DeepEqual
					if reflect.DeepEqual(elem.Type, oldFiber.typeOf) {
						sameType = true
					}
				}
			} else {
				debugf("FIBER", "🚨 reconcileChildren: element is not *Element, got %T\n", element)
			}
		}

		// Keyed diff: if a 'key' prop is present, require key equality for reuse
		if element != nil {
			var elemKey interface{}
			if elemEl, ok := element.(*Element); ok {
				elemKey = elemEl.Props["key"]
			}
			var oldKey interface{}
			if oldFiber != nil && oldFiber.props != nil {
				oldKey = oldFiber.props["key"]
			}
			if elemKey != nil || oldKey != nil {
				if !fastEqual(elemKey, oldKey) {
					sameType = false
				}
			}
		}

		if sameType {
			// Reuse the existing fiber
			debugf("FIBER", "♻️ reconcileChildren: reusing existing fiber of type %v\n", oldFiber.typeOf)
			// fmt.Printf("reconcileChildren: Reusing existing fiber of type %v\n", oldFiber.typeOf)
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:    oldFiber.typeOf,
					props:     elem.Props,
					dom:       oldFiber.dom,
					parent:    wipFiber,
					alternate: oldFiber,
					effectTag: "UPDATE",
				}
				debugf("FIBER", "✅ reconcileChildren: created UPDATE fiber %p\n", newFiber)
			} else {
				debugf("FIBER", "🚨 reconcileChildren: element is not *Element for reuse, got %T\n", element)
			}
		} else if element != nil {
			// Create a new fiber
			debugf("FIBER", "🆕 reconcileChildren: creating new fiber\n")
			// fmt.Printf("reconcileChildren: Creating new fiber of type %v\n", element.(*Element).Type)
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:    elem.Type,
					props:     elem.Props,
					dom:       js.Value{},
					parent:    wipFiber,
					effectTag: "PLACEMENT",
				}
				debugf("FIBER", "✅ reconcileChildren: created PLACEMENT fiber %p (type: %v)\n",
					newFiber, elem.Type)
			} else {
				debugf("FIBER", "🚨 reconcileChildren: element is not *Element for creation, got %T\n", element)
			}
		}

		if oldFiber != nil && !sameType {
			// Mark the old fiber for deletion
			debugf("FIBER", "🗑️ reconcileChildren: marking fiber for deletion (type: %v)\n", oldFiber.typeOf)
			// fmt.Printf("reconcileChildren: Deleting fiber of type %v\n", oldFiber.typeOf)
			oldFiber.effectTag = "DELETION"
			deletions = append(deletions, oldFiber)
			debugf("FIBER", "📋 reconcileChildren: total deletions queued: %d\n", len(deletions))
		}

		if oldFiber != nil {
			oldFiber = oldFiber.sibling
		}

		if index == 0 {
			wipFiber.child = newFiber
			// fmt.Println("reconcileChildren: Setting first child fiber")
		} else if element != nil && prevSibling != nil {
			prevSibling.sibling = newFiber
			// fmt.Printf("reconcileChildren: Linking sibling fiber of type %v\n", newFiber.typeOf)
		}

		prevSibling = newFiber
		index++
	}

	// fmt.Printf("reconcileChildren: Completed reconciliation for fiber type %v\n", wipFiber.typeOf)
	reconcileDuration := time.Since(reconcileStartTime)
	debugf("FIBER", "✅ reconcileChildren: completed reconciliation in %v for fiber %p\n", reconcileDuration, wipFiber)
}

// requestIdleCallback schedules work during idle periods.
func requestIdleCallback(callback func(js.Value)) {
	// Check memory pressure before creating new callback
	checkMemoryPressure()

	var cb js.Func
	cb = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		defer cb.Release() // Clean up immediately after use
		callback(args[0])
		return nil
	})

	debugf("MEMORY", "🔗 requestIdleCallback created callback (will auto-release)\n")

	// Feature-detect requestIdleCallback support
	ric := js.Global().Get("requestIdleCallback")
	if !ric.IsUndefined() && ric.Truthy() {
		ric.Invoke(cb)
	} else {
		// Fallback: schedule soon via setTimeout 1ms
		js.Global().Call("setTimeout", cb, 1)
	}
}

// SetUIQueueBufferSize configures the UI queue buffer size
// This allows applications to tune buffer size based on their specific needs
func SetUIQueueBufferSize(size int) {
	if size < uiQueueMinSize {
		size = uiQueueMinSize
	}
	if size > uiQueueMaxSize {
		size = uiQueueMaxSize
	}
	
	debugf("FIBER", "🔧 SetUIQueueBufferSize: resizing UI queue to %d\n", size)
	resizeUIQueue(size)
}

// SetUIQueueLimits configures the min and max buffer sizes for the UI queue
func SetUIQueueLimits(minSize, maxSize int) {
	if minSize < 64 {
		minSize = 64 // Absolute minimum
	}
	if maxSize < minSize {
		maxSize = minSize * 2
	}
	if maxSize > 16384 {
		maxSize = 16384 // Absolute maximum to prevent excessive memory usage
	}
	
	oldMin, oldMax := uiQueueMinSize, uiQueueMaxSize
	uiQueueMinSize = minSize
	uiQueueMaxSize = maxSize
	
	debugf("FIBER", "🔧 SetUIQueueLimits: updated limits min=%d→%d, max=%d→%d\n", 
		oldMin, minSize, oldMax, maxSize)
	
	// Adjust current size if it's outside new limits
	if uiQueueSize < minSize {
		resizeUIQueue(minSize)
	} else if uiQueueSize > maxSize {
		resizeUIQueue(maxSize)
	}
}

// GetUIQueueStats returns comprehensive UI queue statistics
func GetUIQueueStats() map[string]int64 {
	return map[string]int64{
		"currentSize":     int64(len(uiQueue)),
		"bufferSize":      int64(cap(uiQueue)),
		"minSize":         int64(uiQueueMinSize),
		"maxSize":         int64(uiQueueMaxSize),
		"overflowCount":   atomic.LoadInt64(&uiQueueOverflows),
		"maxReached":      atomic.LoadInt64(&uiQueueMaxReached),
		"growthEvents":    atomic.LoadInt64(&uiQueueGrowth),
		"shrinkEvents":    atomic.LoadInt64(&uiQueueShrinks),
	}
}

// ResetUIQueueStats resets UI queue statistics counters
func ResetUIQueueStats() {
	atomic.StoreInt64(&uiQueueOverflows, 0)
	atomic.StoreInt64(&uiQueueMaxReached, 0)
	atomic.StoreInt64(&uiQueueGrowth, 0)
	atomic.StoreInt64(&uiQueueShrinks, 0)
	debugf("FIBER", "🔄 ResetUIQueueStats: statistics counters reset\n")
}

// EnableUIQueueAutoOptimization enables automatic buffer size optimization
// This is enabled by default but can be disabled for manual control
var uiQueueAutoOptimization bool = true

func SetUIQueueAutoOptimization(enabled bool) {
	uiQueueAutoOptimization = enabled
	debugf("FIBER", "🔧 SetUIQueueAutoOptimization: %v\n", enabled)
}
