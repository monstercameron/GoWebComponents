// ./fiber/fiber.go

package fiber

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"syscall/js"
	"unsafe"
)

// Object pools for reducing allocations
var (
	fiberPool = sync.Pool{
		New: func() interface{} {
			return &Fiber{}
		},
	}

	hooksPool = sync.Pool{
		New: func() interface{} {
			return &Hooks{
				state: make([]interface{}, 0, 8),   // Larger pre-allocation
				deps:  make([][]interface{}, 0, 8), // Larger pre-allocation
				memos: make([]memoizedValue, 0, 4), // Larger pre-allocation
			}
		},
	}

	// NEW: Element pool for createElement optimization
	elementPool = sync.Pool{
		New: func() interface{} {
			return &Element{
				Props: make(map[string]interface{}, 8), // Pre-allocated map
			}
		},
	}

	// NEW: Props pool for map reuse
	propsPool = sync.Pool{
		New: func() interface{} {
			return make(map[string]interface{}, 8)
		},
	}
)

// Global variables for tracking the current fiber and root.
var (
	wipRoot         *Fiber
	currentRoot     *Fiber
	nextUnitOfWork  *Fiber
	deletions       []*Fiber
	wipFiber        *Fiber
	eventCallbacks  []js.Func // Global slice to keep event callbacks alive
	rafCallbacks    []js.Func // Global slice to keep callbacks alive
	updateScheduled bool      // Flag to prevent multiple update scheduling

	// Shared empty slice to avoid allocations
	emptyChildren = []interface{}{}
)

// Element represents a virtual DOM node.
type Element struct {
	Type     interface{}
	Props    map[string]interface{}
	Children []interface{}
}

// NEW: Fast equality checking without reflection
type FastComparable interface {
	FastEqual(other interface{}) bool
}

// NEW: Common primitive type fast equality
func fastEqual(a, b interface{}) bool {
	// Fast path: nil checks first
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Fast path: Check if types implement FastComparable
	if fc, ok := a.(FastComparable); ok {
		return fc.FastEqual(b)
	}

	// Fast path: Common primitive types (avoid reflection)
	switch va := a.(type) {
	case string:
		if vb, ok := b.(string); ok {
			return va == vb
		}
	case int:
		if vb, ok := b.(int); ok {
			return va == vb
		}
	case int64:
		if vb, ok := b.(int64); ok {
			return va == vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va == vb
		}
	case bool:
		if vb, ok := b.(bool); ok {
			return va == vb
		}
	case []string:
		if vb, ok := b.([]string); ok {
			if len(va) != len(vb) {
				return false
			}
			for i := range va {
				if va[i] != vb[i] {
					return false
				}
			}
			return true
		}
	case []interface{}:
		if vb, ok := b.([]interface{}); ok {
			if len(va) != len(vb) {
				return false
			}
			for i := range va {
				if !fastEqual(va[i], vb[i]) {
					return false
				}
			}
			return true
		}
	case []int:
		if vb, ok := b.([]int); ok {
			if len(va) != len(vb) {
				return false
			}
			for i := range va {
				if va[i] != vb[i] {
					return false
				}
			}
			return true
		}
	}

	// Use pointer equality check only for comparable types
	// Avoid == for uncomparable types like slices, maps, functions
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	// Check if types are the same
	if va.Type() != vb.Type() {
		return false
	}

	// For uncomparable types, fall back to reflect.DeepEqual
	if !va.Type().Comparable() {
		return reflect.DeepEqual(a, b)
	}

	// Safe to use == for comparable types
	return a == b
}

// createElement constructs an Element with optimized allocations
func createElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	// Get element from pool
	elem := elementPool.Get().(*Element)

	// Reset the element
	elem.Type = typ

	// Process children to support both component references and return values
	processedChildren := make([]interface{}, 0, len(children))

	for _, child := range children {
		if child == nil {
			continue
		}

		// Check if child is a component function reference
		if componentFunc, ok := child.(func(map[string]interface{}) *Element); ok {
			// Call the component function with nil props
			result := componentFunc(nil)
			if result != nil {
				processedChildren = append(processedChildren, result)
			}
		} else {
			// Child is already processed (Element, Text, etc.)
			processedChildren = append(processedChildren, child)
		}
	}

	elem.Children = processedChildren[:len(processedChildren):len(processedChildren)] // Ensure capacity equals length

	// Handle props efficiently
	if props != nil {
		// Clear existing props map efficiently
		for k := range elem.Props {
			delete(elem.Props, k)
		}
		// Copy props (map is already allocated)
		for k, v := range props {
			elem.Props[k] = v
		}
	} else {
		// Clear props if none provided
		for k := range elem.Props {
			delete(elem.Props, k)
		}
	}

	// Set children in props
	if len(processedChildren) > 0 {
		elem.Props["children"] = processedChildren
	} else {
		// Use a shared empty slice to avoid allocations while maintaining type safety
		elem.Props["children"] = emptyChildren
	}

	return elem
}

// NEW: Release element back to pool
func releaseElement(elem *Element) {
	if elem != nil {
		elem.Type = nil
		elem.Children = nil
		// Keep Props map allocated for reuse
		elementPool.Put(elem)
	}
}

// Text creates a text node.
func Text(content string) *Element {
	return createElement("TEXT_ELEMENT", map[string]interface{}{
		"nodeValue": content,
	})
}

// useState manages state in a component with optimized equality checking
func useState[T any](initialValue T) (func() T, func(T)) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	if len(currentFiber.hooks.state) > position {
		// Existing state
	} else {
		// Initial state - grow slice efficiently
		if cap(currentFiber.hooks.state) <= position {
			// Double capacity when needed
			newCap := max(8, len(currentFiber.hooks.state)*2)
			newState := make([]interface{}, len(currentFiber.hooks.state), newCap)
			copy(newState, currentFiber.hooks.state)
			currentFiber.hooks.state = newState
		}
		currentFiber.hooks.state = append(currentFiber.hooks.state, initialValue)
	}

	// Capture hooks and position
	hooks := currentFiber.hooks
	idx := position

	getter := func() T {
		return hooks.state[idx].(T)
	}

	setter := func(newValue T) {
		// Use fast equality check instead of reflect.DeepEqual
		if hooks.state[idx] == nil || !fastEqual(hooks.state[idx], newValue) {
			hooks.state[idx] = newValue
			scheduleUpdateAtRoot()
		}
	}

	return getter, setter
}

// NEW: Get hooks from pool with reset
func getHooksFromPool() *Hooks {
	hooks := hooksPool.Get().(*Hooks)
	hooks.index = 0
	// Reuse slices, just reset length
	hooks.state = hooks.state[:0]
	hooks.deps = hooks.deps[:0]
	hooks.memos = hooks.memos[:0]
	return hooks
}

// NEW: Return hooks to pool
func releaseHooks(hooks *Hooks) {
	if hooks != nil {
		// Don't clear slices, just reset for reuse
		hooksPool.Put(hooks)
	}
}

func scheduleUpdateAtRoot() {
	if currentRoot == nil || updateScheduled {
		return
	}
	updateScheduled = true

	// Reuse fiber instead of allocating new one
	if wipRoot == nil {
		wipRoot = fiberPool.Get().(*Fiber)
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

type memoizedValue struct {
	value interface{}
	deps  []interface{}
}

// Hooks struct optimized for memory alignment and access patterns
type Hooks struct {
	// Hot path data - accessed most frequently
	index int // 8 bytes (padded)

	// Slice headers grouped together (each is 24 bytes)
	state []interface{}   // 24 bytes
	deps  [][]interface{} // 24 bytes
	memos []memoizedValue // 24 bytes
	// Total: 80 bytes, well-aligned
}

func useEffect(effect func(), deps ...interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Grow deps slice efficiently
	for len(currentFiber.hooks.deps) <= position {
		currentFiber.hooks.deps = append(currentFiber.hooks.deps, nil)
	}

	if currentFiber.hooks.deps[position] == nil {
		// First time this effect is used
		currentFiber.hooks.deps[position] = deps
		// Grow effects slice efficiently
		if cap(currentFiber.effects) <= len(currentFiber.effects) {
			newCap := max(4, cap(currentFiber.effects)*2)
			newEffects := make([]func(), len(currentFiber.effects), newCap)
			copy(newEffects, currentFiber.effects)
			currentFiber.effects = newEffects
		}
		currentFiber.effects = append(currentFiber.effects, effect)
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		shouldRun := len(deps) == 0 || !areDepsEqual(prevDeps, deps)
		if shouldRun {
			// Dependencies have changed or no dependencies provided
			currentFiber.hooks.deps[position] = deps
			currentFiber.effects = append(currentFiber.effects, effect)
		}
	}
}

// Optimized dependency comparison
func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	if len(prevDeps) != len(newDeps) {
		return false
	}

	// Fast path for empty deps
	if len(prevDeps) == 0 {
		return true
	}

	// Use fast equality check
	for i := range prevDeps {
		if !fastEqual(prevDeps[i], newDeps[i]) {
			return false
		}
	}
	return true
}

func useMemo(compute func() interface{}, deps ...interface{}) interface{} {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Grow memos slice efficiently
	for len(currentFiber.hooks.memos) <= position {
		currentFiber.hooks.memos = append(currentFiber.hooks.memos, memoizedValue{})
	}

	memo := &currentFiber.hooks.memos[position]

	if memo.value == nil {
		// First time this memo is used
		value := compute() // Remove goroutine overhead for simple computations
		memo.value = value
		memo.deps = deps
		return value
	}

	shouldCompute := len(deps) == 0 || !areDepsEqual(memo.deps, deps)
	if shouldCompute {
		value := compute() // Direct call, no goroutine
		memo.value = value
		memo.deps = deps
		return value
	}

	// Dependencies haven't changed, return the memoized value
	return memo.value
}

// NEW: Utility function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Fiber represents a unit of work in the virtual DOM tree.
// Fiber struct optimized for memory alignment and cache efficiency
type Fiber struct {
	// Group pointers together for better cache locality (40 bytes)
	parent    *Fiber // 8 bytes
	alternate *Fiber // 8 bytes
	child     *Fiber // 8 bytes
	sibling   *Fiber // 8 bytes
	hooks     *Hooks // 8 bytes

	// Group interface and map together (48 bytes)
	typeOf interface{}            // 16 bytes
	props  map[string]interface{} // 8 bytes
	dom    js.Value               // 24 bytes

	// Smaller types grouped at end (40 bytes)
	effectTag string   // 16 bytes
	effects   []func() // 24 bytes
}

// NEW: Reset fiber for pool reuse
func resetFiber(f *Fiber) {
	f.parent = nil
	f.alternate = nil
	f.child = nil
	f.sibling = nil
	if f.hooks != nil {
		releaseHooks(f.hooks)
		f.hooks = nil
	}
	f.typeOf = nil
	f.props = nil
	f.dom = js.Value{}
	f.effectTag = ""
	f.effects = f.effects[:0] // Reuse slice
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
	fmt.Println("render: Root fiber created.")
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	// fmt.Println("render: Scheduling work loop.")
	requestIdleCallback(workLoop)
}

// workLoop performs work until there is no more work left or the deadline expires.
func workLoop(deadline js.Value) {
	// fmt.Println("workLoop: Starting work loop.")
	var shouldYield bool = false
	for nextUnitOfWork != nil && !shouldYield {
		// fmt.Println("workLoop: Performing a unit of work.")
		nextUnitOfWork = performUnitOfWork(nextUnitOfWork)
		shouldYield = deadline.Call("timeRemaining").Float() < 1
		// fmt.Printf("workLoop: timeRemaining=%f, shouldYield=%v\n", deadline.Call("timeRemaining").Float(), shouldYield)
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
		reconcileChildren(fiber, fiber.props["children"].([]interface{}))
	} else {
		switch fiber.typeOf.(type) {
		case func(map[string]interface{}) *Element:
			// Function component
			componentFunc := fiber.typeOf.(func(map[string]interface{}) *Element)
			wipFiber = fiber

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil && fiber.alternate.hooks != nil {
				oldHooks = fiber.alternate.hooks
			}

			// Initialize hooks
			if oldHooks != nil {
				wipFiber.hooks = &Hooks{
					state: make([]interface{}, len(oldHooks.state)),
					deps:  make([][]interface{}, len(oldHooks.deps)),
				}
				copy(wipFiber.hooks.state, oldHooks.state)

				// Deep copy the deps slices
				for i := range oldHooks.deps {
					if oldHooks.deps[i] != nil {
						wipFiber.hooks.deps[i] = make([]interface{}, len(oldHooks.deps[i]))
						copy(wipFiber.hooks.deps[i], oldHooks.deps[i])
					}
				}
			} else {
				wipFiber.hooks = &Hooks{
					state: []interface{}{},
					deps:  [][]interface{}{},
				}
			}
			wipFiber.hooks.index = 0

			// Initialize effects
			wipFiber.effects = []func(){}

			element := componentFunc(fiber.props)
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
				elements := propsChildren.([]interface{})
				reconcileChildren(fiber, elements)
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

// createDom creates a DOM node from a fiber.
func createDom(fiber *Fiber) js.Value {
	// fmt.Printf("createDom: Creating DOM for fiber type %v\n", fiber.typeOf)
	var dom js.Value
	switch t := fiber.typeOf.(type) {
	case string:
		if t == "TEXT_ELEMENT" {
			dom = js.Global().Get("document").Call("createTextNode", fiber.props["nodeValue"])
		} else {
			dom = js.Global().Get("document").Call("createElement", t)
		}
	default:
		// Function components do not create DOM nodes here
		// fmt.Println("createDom: Function component, no DOM node created")
		return js.Value{}
	}

	// Add event listeners and properties
	for name, value := range fiber.props {
		if name == "children" {
			continue
		}
		if name == "dangerouslySetInnerHTML" {
			// Set innerHTML directly
			htmlContent := value.(map[string]string)["__html"]
			// fmt.Println("createDom: Setting innerHTML")
			dom.Set("innerHTML", htmlContent)
			continue
		}
		if len(name) > 2 && name[:2] == "on" {
			// Event handlers
			eventType := strings.ToLower(name[2:]) // Convert event type to lowercase
			// fmt.Printf("createDom: Adding event listener for %s\n", eventType)

			// Ensure the value is of the correct function type
			eventHandler, ok := value.(js.Func)
			if !ok {
				// fmt.Printf("createDom: Event handler for %s is not a js.Func\n", eventType)
				continue
			}

			dom.Call("addEventListener", eventType, eventHandler)
			continue
		}
		if name == "class" {
			// Handle 'class' attribute using setAttribute
			// fmt.Printf("createDom: Setting attribute 'class' to '%v'\n", value)
			dom.Call("setAttribute", "class", value)
			continue
		}
		// Set other properties directly
		// fmt.Printf("createDom: Setting property '%s' to '%v'\n", name, value)
		dom.Set(name, value)
	}
	return dom
}

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
			switch elemType := element.(*Element).Type.(type) {
			case func(map[string]interface{}) *Element:
				// Function component: Compare function pointers using reflect
				funcPtrNew := reflect.ValueOf(elemType).Pointer()
				funcPtrOld, ok := oldFiber.typeOf.(func(map[string]interface{}) *Element)
				if ok {
					funcPtrOldValue := reflect.ValueOf(funcPtrOld).Pointer()
					if funcPtrNew == funcPtrOldValue {
						sameType = true
					}
				}
			case string:
				// Host component: Use reflect.DeepEqual for string comparison
				if reflect.DeepEqual(element.(*Element).Type, oldFiber.typeOf) {
					sameType = true
				}
			default:
				// Other types: Use reflect.DeepEqual
				if reflect.DeepEqual(element.(*Element).Type, oldFiber.typeOf) {
					sameType = true
				}
			}
		}

		if sameType {
			// Reuse the existing fiber
			// fmt.Printf("reconcileChildren: Reusing existing fiber of type %v\n", oldFiber.typeOf)
			newFiber = &Fiber{
				typeOf:    oldFiber.typeOf,
				props:     element.(*Element).Props,
				dom:       oldFiber.dom,
				parent:    wipFiber,
				alternate: oldFiber,
				effectTag: "UPDATE",
			}
		} else if element != nil {
			// Create a new fiber
			// fmt.Printf("reconcileChildren: Creating new fiber of type %v\n", element.(*Element).Type)
			newFiber = &Fiber{
				typeOf:    element.(*Element).Type,
				props:     element.(*Element).Props,
				dom:       js.Value{},
				parent:    wipFiber,
				effectTag: "PLACEMENT",
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

// commitRoot commits the changes to the DOM.
func commitRoot() {
	// fmt.Println("commitRoot: Starting to commit changes to DOM")
	for _, deletion := range deletions {
		// fmt.Printf("commitRoot: Processing deletion for fiber type %v\n", deletion.typeOf)
		commitWork(deletion)
	}
	if wipRoot.child != nil {
		// fmt.Printf("commitRoot: Committing child fiber of type %v\n", wipRoot.child.typeOf)
		commitWork(wipRoot.child)
	}
	currentRoot = wipRoot
	wipRoot = nil
	deletions = nil
	updateScheduled = false // Reset flag after commit
	// fmt.Println("commitRoot: Finished committing changes to DOM")

	// Execute effects after committing
	executeEffects()
}

func executeEffects() {
	if currentRoot == nil {
		return
	}

	var effectFibers []*Fiber
	var collectEffects func(fiber *Fiber)
	collectEffects = func(fiber *Fiber) {
		if fiber == nil {
			return
		}
		if len(fiber.effects) > 0 {
			effectFibers = append(effectFibers, fiber)
		}
		collectEffects(fiber.child)
		collectEffects(fiber.sibling)
	}

	// Collect fibers with effects starting from the root
	collectEffects(currentRoot.child)

	var wg sync.WaitGroup

	// Execute effects in parallel with bounded concurrency
	for _, fiber := range effectFibers {
		for _, effect := range fiber.effects {
			if effect != nil {
				wg.Add(1)
				go func(effect func()) {
					defer wg.Done()
					effect()
				}(effect)
			}
		}
		// Clear the effects after executing them to prevent accumulation
		fiber.effects = nil // Set to nil instead of empty slice to release memory
	}

	wg.Wait()

	// Clear the slice to prevent memory leaks
	effectFibers = nil
}

func resetHookIndex(fiber *Fiber) {
	if fiber == nil {
		return
	}
	if fiber.hooks != nil {
		fiber.hooks.index = 0
	}
	resetHookIndex(fiber.child)
	resetHookIndex(fiber.sibling)
}

// commitWork recursively commits work to the DOM.
func commitWork(fiber *Fiber) {
	if fiber == nil {
		return
	}
	var domParentFiber = fiber.parent
	for domParentFiber != nil && (domParentFiber.dom.IsUndefined() || domParentFiber.dom.IsNull()) {
		domParentFiber = domParentFiber.parent
	}
	if domParentFiber == nil {
		// fmt.Println("commitWork: No valid parent DOM fiber found")
		return
	}
	domParent := domParentFiber.dom

	switch fiber.effectTag {
	case "PLACEMENT":
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			// fmt.Printf("commitWork: Appending child %v to parent %v\n", fiber.dom, domParent)
			domParent.Call("appendChild", fiber.dom)
		} else {
			// fmt.Println("commitWork: Fiber has no DOM node, committing its children")
			commitWork(fiber.child)
			return
		}
	case "UPDATE":
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			// fmt.Printf("commitWork: Updating DOM node for fiber type %v\n", fiber.typeOf)
			updateDom(fiber.dom, fiber.alternate.props, fiber.props)
		}
	case "DELETION":
		// fmt.Println("commitWork: Deleting DOM node")
		commitDeletion(fiber, domParent)
		return
	}

	// Commit children and siblings
	commitWork(fiber.child)
	commitWork(fiber.sibling)
}

func commitDeletion(fiber *Fiber, domParent js.Value) {
	if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
		// fmt.Printf("commitDeletion: Removing child %v from parent %v\n", fiber.dom, domParent)
		domParent.Call("removeChild", fiber.dom)

		// Release event callbacks associated with this fiber
		if fiber.hooks != nil {
			for _, state := range fiber.hooks.state {
				if fn, ok := state.(js.Func); ok {
					// fmt.Println("commitDeletion: Releasing event callback")
					fn.Release()
				}
			}
		}
	} else if fiber.child != nil {
		// fmt.Println("commitDeletion: Deleting child fibers recursively")
		commitDeletion(fiber.child, domParent)
	}
}

func updateDom(dom js.Value, oldProps, newProps map[string]interface{}) {
	// Fast path: check if maps are equal first
	if len(oldProps) == 0 && len(newProps) == 0 {
		return
	}

	// Ultra-fast path: pointer equality check
	if unsafe.Pointer(&oldProps) == unsafe.Pointer(&newProps) {
		return
	}

	// 1. Remove old or changed event listeners (optimized)
	for name, oldValue := range oldProps {
		// Branch optimization: check first character before string operations
		if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
			// Only remove if not in new props or value changed
			if newValue, exists := newProps[name]; !exists || !fastEqual(oldValue, newValue) {
				eventType := strings.ToLower(name[2:])
				dom.Call("removeEventListener", eventType, oldValue.(js.Func))
			}
		} else if newProps[name] == nil && name != "children" {
			// Remove properties that no longer exist, excluding event listeners and children
			dom.Set(name, js.Undefined())
		}
	}

	// 2. Add new or changed properties and event listeners (optimized)
	for name, value := range newProps {
		// Skip common exclusions first (most frequent check)
		if name == "children" {
			continue
		}

		// Skip if value hasn't changed
		if oldValue, exists := oldProps[name]; exists && fastEqual(oldValue, value) {
			continue
		}

		// Branch optimization: inline checks for most common patterns
		switch name {
		case "class":
			dom.Call("setAttribute", "class", value)
		case "style":
			dom.Set("style", value)
		case "id":
			dom.Set("id", value)
		case "value":
			dom.Set("value", value)
		case "dangerouslySetInnerHTML":
			htmlContent := value.(map[string]string)["__html"]
			dom.Set("innerHTML", htmlContent)
		default:
			// Check for event handlers (less common)
			if len(name) > 2 && name[0] == 'o' && name[1] == 'n' {
				eventType := strings.ToLower(name[2:])
				dom.Call("addEventListener", eventType, value.(js.Func))
			} else {
				dom.Set(name, value)
			}
		}
	}
}

// requestIdleCallback schedules work during idle periods.
func requestIdleCallback(callback func(js.Value)) {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback(args[0])
		return nil
	})
	rafCallbacks = append(rafCallbacks, cb) // Keep the function alive
	js.Global().Call("requestIdleCallback", cb)
}

func useFunc(callback func(js.Value, []js.Value) interface{}) js.Func {
	cb := js.FuncOf(callback)
	eventCallbacks = append(eventCallbacks, cb) // Keep callback alive
	return cb
}

type FetchState struct {
	Data    interface{}
	Error   string
	Loading bool
}

// Implement FastComparable for FetchState to ensure proper state updates
func (fs FetchState) FastEqual(other interface{}) bool {
	otherFS, ok := other.(FetchState)
	if !ok {
		return false
	}

	// Compare Loading and Error first (simple comparisons)
	if fs.Loading != otherFS.Loading || fs.Error != otherFS.Error {
		return false
	}

	// For Data field, use simple pointer/nil comparison to avoid deep comparison issues
	// This ensures that any data change triggers an update
	if (fs.Data == nil) != (otherFS.Data == nil) {
		return false
	}

	// If both are nil, they're equal
	if fs.Data == nil && otherFS.Data == nil {
		return true
	}

	// If both are non-nil, consider them different to force updates
	// This is safe because fetch operations should produce new data objects
	return false
}

type FetchOptions struct {
	Method  string
	Headers map[string]interface{}
	Body    interface{}
}

type FetchResult struct {
	Data interface{}
	Err  error
}

func useFetch2(url string, options ...FetchOptions) (func() FetchState, func()) {
	getState, setState := useState(FetchState{Loading: true})

	var opts FetchOptions
	if len(options) > 0 {
		opts = options[0]
	}

	fetchData := func() {
		fmt.Println("useFetch: Fetching data from", url)

		// Set loading state
		setState(FetchState{Loading: true})

		// Create fetch options
		fetchOptions := js.Global().Get("Object").New()
		if opts.Method != "" {
			fetchOptions.Set("method", opts.Method)
		}
		if len(opts.Headers) > 0 {
			headers := js.Global().Get("Object").New()
			for key, value := range opts.Headers {
				headers.Set(key, value)
			}
			fetchOptions.Set("headers", headers)
		}
		if opts.Body != nil {
			switch v := opts.Body.(type) {
			case string:
				fetchOptions.Set("body", v)
			default:
				bodyJSON, err := json.Marshal(v)
				if err != nil {
					setState(FetchState{Error: "Error encoding request body: " + err.Error(), Loading: false})
					return
				}
				fetchOptions.Set("body", string(bodyJSON))
			}
		}

		fetchPromise := js.Global().Call("fetch", url, fetchOptions)
		fetchPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			response := args[0]
			if !response.Get("ok").Bool() {
				errorMsg := fmt.Sprintf("HTTP error! status: %s", response.Get("status").String())
				fmt.Println("useFetch:", errorMsg)
				setState(FetchState{Error: errorMsg, Loading: false})
				return nil
			}
			response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				data := args[0]
				jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
				var parsedData interface{}
				err := json.Unmarshal([]byte(jsonStr), &parsedData)
				if err != nil {
					fmt.Println("Error parsing data:", err)
					setState(FetchState{Error: err.Error(), Loading: false})
				} else {
					fmt.Println("useFetch: Successfully fetched data")
					setState(FetchState{Data: parsedData, Loading: false})
				}
				return nil
			}))
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			err := args[0]
			errorMsg := fmt.Sprintf("Fetch error: %s", err.Get("message").String())
			fmt.Println(errorMsg)
			setState(FetchState{Error: errorMsg, Loading: false})
			return nil
		}))
	}

	useEffect(func() {
		fetchData()
	}, []interface{}{url})

	return getState, fetchData
}

func useFetch(url string) func() FetchState {
	getState, setState := useState(FetchState{Loading: true, Data: nil, Error: ""})

	useEffect(func() {
		// Set loading state
		setState(FetchState{Loading: true, Data: nil, Error: ""})

		fetchPromise := js.Global().Call("fetch", url)
		fetchPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			response := args[0]
			if !response.Get("ok").Bool() {
				errorMsg := fmt.Sprintf("HTTP error! status: %s", response.Get("status").String())
				setState(FetchState{Error: errorMsg, Loading: false})
				return nil
			}
			response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				data := args[0]
				jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
				var parsedData interface{}
				err := json.Unmarshal([]byte(jsonStr), &parsedData)
				if err != nil {
					setState(FetchState{Error: err.Error(), Loading: false})
				} else {
					setState(FetchState{Data: parsedData, Loading: false})
				}
				return nil
			}))
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			err := args[0]
			errorMsg := fmt.Sprintf("Fetch error: %s", err.Get("message").String())
			setState(FetchState{Error: errorMsg, Loading: false})
			return nil
		}))
	}, []interface{}{url})

	return getState
}

// GoFetch performs an asynchronous fetch operation and returns a channel for the result
func GoFetch(url string, options FetchOptions) <-chan FetchResult {
	resultChan := make(chan FetchResult, 1) // Buffered channel to avoid goroutine leak

	go func() {
		defer close(resultChan)

		fetchOptions := js.Global().Get("Object").New()
		setFetchOptions(fetchOptions, options)

		promiseResultChan := make(chan FetchResult, 1)
		performFetch(url, fetchOptions, promiseResultChan)

		result := <-promiseResultChan
		resultChan <- result
	}()

	return resultChan
}

// Fetch performs an asynchronous fetch operation and invokes a callback with the result
func Fetch(url string, options FetchOptions, callback func(FetchResult)) {
	go func() {
		fetchOptions := js.Global().Get("Object").New()
		setFetchOptions(fetchOptions, options)

		promiseResultChan := make(chan FetchResult, 1)
		performFetch(url, fetchOptions, promiseResultChan)

		result := <-promiseResultChan
		callback(result)
	}()
}

func setFetchOptions(fetchOptions js.Value, options FetchOptions) {
	if options.Method != "" {
		fetchOptions.Set("method", options.Method)
	}

	if len(options.Headers) > 0 {
		headers := js.Global().Get("Object").New()
		for key, value := range options.Headers {
			headers.Set(key, value)
		}
		fetchOptions.Set("headers", headers)
	}

	if options.Body != nil {
		switch v := options.Body.(type) {
		case string:
			fetchOptions.Set("body", v)
		default:
			bodyJSON, err := json.Marshal(v)
			if err != nil {
				fetchOptions.Set("body", fmt.Sprintf("error encoding body: %v", err))
			} else {
				fetchOptions.Set("body", string(bodyJSON))
			}
		}
	}
}

func performFetch(url string, fetchOptions js.Value, resultChan chan<- FetchResult) {
	promise := js.Global().Call("fetch", url, fetchOptions)
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		if !response.Get("ok").Bool() {
			resultChan <- FetchResult{Err: fmt.Errorf("HTTP error! status: %s", response.Get("status").String())}
			return nil
		}

		response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			data := args[0]
			jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
			var parsedData interface{}
			err := json.Unmarshal([]byte(jsonStr), &parsedData)
			if err != nil {
				resultChan <- FetchResult{Err: fmt.Errorf("error parsing response: %w", err)}
			} else {
				resultChan <- FetchResult{Data: parsedData}
			}
			return nil
		}))
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := args[0]
		resultChan <- FetchResult{Err: fmt.Errorf("fetch error: %s", err.Get("message").String())}
		return nil
	}))
}
