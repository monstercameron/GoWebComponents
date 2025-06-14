// ./fiber/fiber.go

//go:build js && wasm
// +build js,wasm

package fiber

import (
	"reflect"
	"syscall/js"
)

// Global variables for tracking the current fiber and root.
var (
	wipRoot         *Fiber
	currentRoot     *Fiber
	nextUnitOfWork  *Fiber
	deletions       []*Fiber
	wipFiber        *Fiber
	updateScheduled bool // Flag to prevent multiple update scheduling

	// --- UI queue for main-thread safe updates ---
	uiQueue = make(chan func(), 1024) // Buffer to avoid blocking background goroutines

	// Indicates we're executing on the main scheduler/commit/effect loop
	schedulerActive int32

	// Debug logging toggle
	debugEnabled = true
)

func scheduleUpdateAtRoot() {
	if currentRoot == nil || updateScheduled {
		return
	}
	updateScheduled = true

	// Reuse fiber instead of allocating new one
	if wipRoot == nil {
		poolFiber := fiberPool.Get()
		if fiber, ok := poolFiber.(*Fiber); ok {
			wipRoot = fiber
		} else {
			// This should never happen if pool is properly initialized, but handle gracefully
			debugf("FIBER", "🚨 scheduleUpdateAtRoot: fiberPool returned unexpected type %T, creating new Fiber\n", poolFiber)
			wipRoot = &Fiber{
				props: make(map[string]interface{}),
			}
		}
	}

	wipRoot.typeOf = currentRoot.typeOf
	wipRoot.dom = currentRoot.dom
	wipRoot.props = currentRoot.props
	wipRoot.alternate = currentRoot
	wipRoot.effectTag = ""
	wipRoot.parent = nil
	wipRoot.child = nil
	wipRoot.sibling = nil

	nextUnitOfWork = wipRoot
	deletions = deletions[:0] // Reuse slice
	requestIdleCallback(workLoop)
}

// getCurrentFiber retrieves the current working fiber.
func getCurrentFiber() *Fiber {
	return wipFiber
}

// scheduleUpdate triggers a re-render of the component.
func scheduleUpdate(fiber *Fiber) {
	// fmt.Println("scheduleUpdate: Scheduling update")
	wipRoot = &Fiber{
		typeOf:    "ROOT",
		dom:       currentRoot.dom,
		props:     currentRoot.props,
		alternate: currentRoot,
	}
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	// fmt.Println("scheduleUpdate: wipRoot set and workLoop scheduled")
	requestIdleCallback(workLoop)
}

// render starts the rendering process.
func render(element *Element, container js.Value) {
	// fmt.Println("render: Starting rendering process.")
	wipRoot = &Fiber{
		typeOf:    "ROOT", // Assign a type to the root fiber
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{element}},
		alternate: currentRoot,
	}
	debugf("RENDER", "Root fiber created.\n")
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	// fmt.Println("render: Scheduling work loop.")
	requestIdleCallback(workLoop)
}

// Render is the exported version of render - starts the rendering process
func Render(element *Element, container js.Value) {
	render(element, container)
}

// CreateElement is the exported version of createElement
func CreateElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	return createElement(typ, props, children...)
}

// workLoop performs work until there is no more work left or the deadline expires.
func workLoop(deadline js.Value) {
	// fmt.Println("workLoop: Starting work loop.")
	startSchedulerSection()
	defer endSchedulerSection()

	const maxUnitsPerSlice = 300 // Prevent long monopolisation of idle period
	units := 0

	var shouldYield bool
	for nextUnitOfWork != nil {
		nextUnitOfWork = performUnitOfWork(nextUnitOfWork)

		units++
		// Yield criteria: low time remaining OR processed many units already
		if deadline.Call("timeRemaining").Float() < 1 || units >= maxUnitsPerSlice {
			shouldYield = true
		}

		if shouldYield {
			break
		}
	}

	if wipRoot != nil && nextUnitOfWork == nil {
		// fmt.Println("workLoop: No more units of work. Committing root.")
		commitRoot()
	}

	if nextUnitOfWork != nil {
		// fmt.Println("workLoop: Work remains. Scheduling next work loop.")
		requestIdleCallback(workLoop)
	} else {
		// fmt.Println("workLoop: All work completed.")
	}

	// First, handle any queued UI tasks from background goroutines
	processUIQueue()
}

// performUnitOfWork performs a single unit of work.
func performUnitOfWork(fiber *Fiber) *Fiber {
	if fiber == nil {
		// fmt.Println("performUnitOfWork: Fiber is nil.")
		return nil
	}

	// fmt.Printf("performUnitOfWork: Processing fiber of type %v.\n", fiber.typeOf)

	if fiber.typeOf == nil || fiber.typeOf == "ROOT" {
		// fmt.Println("performUnitOfWork: Fiber has typeOf nil or ROOT, reconciling children.")
		if children, ok := fiber.props["children"].([]interface{}); ok {
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
			// Function component with map[string]interface{} props
			componentFunc, ok := fiber.typeOf.(func(map[string]interface{}) *Element)
			if !ok {
				debugf("FIBER", "🚨 performUnitOfWork: fiber.typeOf is not func(map[string]interface{}) *Element, got %T\n", fiber.typeOf)
				return nil
			}
			wipFiber = fiber

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil {
				oldHooks = fiber.alternate.hooks
			}

			if oldHooks != nil {
				// Reuse existing Hooks instance and reset per-render state
				wipFiber.hooks = oldHooks
				wipFiber.hooks.index = 0
				wipFiber.hooks.callOrder = wipFiber.hooks.callOrder[:0]
				wipFiber.hooks.orderChecked = false
				// prevOrder already contains the last render's sequence
			} else {
				// First render – allocate a fresh Hooks container
				wipFiber.hooks = &Hooks{
					state:        []interface{}{},
					deps:         [][]interface{}{},
					memos:        []memoizedValue{},
					prevOrder:    []HookCall{},
					callOrder:    []HookCall{},
					orderChecked: false,
				}
			}

			// Initialize effects
			wipFiber.effects = []func(){}

			element := componentFunc(fiber.props)

			// Finalize hook order validation after component execution
			if wipFiber.hooks != nil && !wipFiber.hooks.orderChecked {
				if err := finalizeHookOrder(wipFiber.hooks); err != nil {
					componentName := getFunctionName(fiber.typeOf)
					debugf("HOOKS", "🚨 Component '%s': %v\n", componentName, err)
				}
			}

			if element == nil {
				return nil
			}

			reconcileChildren(fiber, []interface{}{element})
		case func(Attrs) *Element:
			// Function component with Attrs props
			componentFunc, ok := fiber.typeOf.(func(Attrs) *Element)
			if !ok {
				debugf("FIBER", "🚨 performUnitOfWork: fiber.typeOf is not func(Attrs) *Element, got %T\n", fiber.typeOf)
				return nil
			}
			wipFiber = fiber

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil {
				oldHooks = fiber.alternate.hooks
			}

			if oldHooks != nil {
				// Reuse existing Hooks instance and reset per-render state
				wipFiber.hooks = oldHooks
				wipFiber.hooks.index = 0
				wipFiber.hooks.callOrder = wipFiber.hooks.callOrder[:0]
				wipFiber.hooks.orderChecked = false
				// prevOrder already contains the last render's sequence
			} else {
				// First render – allocate a fresh Hooks container
				wipFiber.hooks = &Hooks{
					state:        []interface{}{},
					deps:         [][]interface{}{},
					memos:        []memoizedValue{},
					prevOrder:    []HookCall{},
					callOrder:    []HookCall{},
					orderChecked: false,
				}
			}

			// Initialize effects
			wipFiber.effects = []func(){}

			// Convert map[string]interface{} to Attrs
			var attrs Attrs
			if fiber.props != nil {
				attrs = Attrs(fiber.props)
			}

			element := componentFunc(attrs)

			// Finalize hook order validation after component execution
			if wipFiber.hooks != nil && !wipFiber.hooks.orderChecked {
				if err := finalizeHookOrder(wipFiber.hooks); err != nil {
					componentName := getFunctionName(fiber.typeOf)
					debugf("HOOKS", "🚨 Component '%s': %v\n", componentName, err)
				}
			}

			if element == nil {
				return nil
			}

			reconcileChildren(fiber, []interface{}{element})
		case string:
			// Host component (HTML element)
			// fmt.Printf("performUnitOfWork: Handling host component of type '%s'.\n", fiber.typeOf.(string))
			if fiber.dom.IsUndefined() || fiber.dom.IsNull() {
				// fmt.Println("performUnitOfWork: Creating DOM node for host component.")
				fiber.dom = createDom(fiber)
				// fmt.Println("performUnitOfWork: DOM node created.")
			}

			if fiber.props == nil {
				// fmt.Println("performUnitOfWork: Fiber props are nil. Skipping children reconciliation.")
				return nil
			}

			if propsChildren, ok := fiber.props["children"]; ok {
				// fmt.Println("performUnitOfWork: Reconciling children of host component.")
				if elements, elementsOk := propsChildren.([]interface{}); elementsOk {
					reconcileChildren(fiber, elements)
				} else {
					debugf("FIBER", "🚨 performUnitOfWork: fiber.props[\"children\"] is not []interface{}, got %T\n", propsChildren)
					// Try to reconcile with empty children to avoid crash
					emptyChildren := make([]interface{}, 0)
					reconcileChildren(fiber, emptyChildren)
				}
			}
		default:
			// fmt.Printf("performUnitOfWork: Unhandled fiber type %T.\n", fiber.typeOf)
		}
	}

	// fmt.Printf("performUnitOfWork: Completed processing fiber of type %v.\n", fiber.typeOf)

	// Traverse to child fibers
	if fiber.child != nil {
		// fmt.Printf("performUnitOfWork: Moving to child fiber of type %v.\n", fiber.child.typeOf)
		return fiber.child
	}

	nextFiber := fiber
	for nextFiber != nil {
		if nextFiber.sibling != nil {
			// fmt.Printf("performUnitOfWork: Moving to sibling fiber of type %v.\n", nextFiber.sibling.typeOf)
			return nextFiber.sibling
		}
		// fmt.Println("performUnitOfWork: Moving up to parent fiber.")
		nextFiber = nextFiber.parent
	}
	// fmt.Println("performUnitOfWork: No more fibers to process.")
	return nil
}

// createDom moved to dom.go

// reconcileChildren reconciles the children of a fiber.
func reconcileChildren(wipFiber *Fiber, elements []interface{}) {
	// fmt.Printf("reconcileChildren: Reconciling %d children for fiber type %v\n", len(elements), wipFiber.typeOf)
	index := 0
	var oldFiber *Fiber
	if wipFiber.alternate != nil {
		oldFiber = wipFiber.alternate.child
	}
	var prevSibling *Fiber

	for index < len(elements) || oldFiber != nil {
		var element interface{}
		if index < len(elements) {
			element = elements[index]
		}

		var newFiber *Fiber

		sameType := false
		if oldFiber != nil && element != nil {
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
			} else {
				debugf("FIBER", "🚨 reconcileChildren: element is not *Element for reuse, got %T\n", element)
			}
		} else if element != nil {
			// Create a new fiber
			// fmt.Printf("reconcileChildren: Creating new fiber of type %v\n", element.(*Element).Type)
			if elem, ok := element.(*Element); ok {
				newFiber = &Fiber{
					typeOf:    elem.Type,
					props:     elem.Props,
					dom:       js.Value{},
					parent:    wipFiber,
					effectTag: "PLACEMENT",
				}
			} else {
				debugf("FIBER", "🚨 reconcileChildren: element is not *Element for creation, got %T\n", element)
			}
		}

		if oldFiber != nil && !sameType {
			// Mark the old fiber for deletion
			// fmt.Printf("reconcileChildren: Deleting fiber of type %v\n", oldFiber.typeOf)
			oldFiber.effectTag = "DELETION"
			deletions = append(deletions, oldFiber)
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
}

// requestIdleCallback schedules work during idle periods.
func requestIdleCallback(callback func(js.Value)) {
	// Check memory pressure before creating new callback
	checkMemoryPressure()

	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback(args[0])
		return nil
	})
	rafCallbacks = append(rafCallbacks, cb) // Keep the function alive

	debugf("MEMORY", "🔗 requestIdleCallback created - total callbacks: %d\n", len(eventCallbacks)+len(rafCallbacks))

	// Feature-detect requestIdleCallback support
	ric := js.Global().Get("requestIdleCallback")
	if !ric.IsUndefined() && ric.Truthy() {
		ric.Invoke(cb)
	} else {
		// Fallback: schedule soon via setTimeout 1ms
		js.Global().Call("setTimeout", cb, 1)
	}
}
