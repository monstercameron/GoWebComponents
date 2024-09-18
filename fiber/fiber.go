// ./fiber/fiber.go

package fiber

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"syscall/js"
)

// Global variables for tracking the current fiber and root.
var (
	wipRoot        *Fiber
	currentRoot    *Fiber
	nextUnitOfWork *Fiber
	deletions      []*Fiber
	wipFiber       *Fiber
	eventCallbacks []js.Func // Global slice to keep event callbacks alive
	rafCallbacks   []js.Func // Global slice to keep callbacks alive
)

// Element represents a virtual DOM node.
type Element struct {
	Type     interface{}
	Props    map[string]interface{}
	Children []interface{}
}

// createElement constructs an Element with the given type, props, and children.
func createElement(typ interface{}, props map[string]interface{}, children ...interface{}) *Element {
	if props == nil {
		props = make(map[string]interface{})
	}
	if len(children) > 0 {
		props["children"] = children
	} else {
		props["children"] = []interface{}{}
	}
	return &Element{
		Type:     typ,
		Props:    props,
		Children: children,
	}
}

// Text creates a text node.
func Text(content string) *Element {
	return createElement("TEXT_ELEMENT", map[string]interface{}{
		"nodeValue": content,
	})
}

// useState manages state in a component.
func useState[T any](initialValue T) (func() T, func(T)) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = &Hooks{}
		fmt.Println("useState: Initialized hooks for the current fiber.")
	}

	position := currentFiber.hooks.index
	fmt.Printf("useState: Hook position %d\n", position)
	currentFiber.hooks.index++

	if len(currentFiber.hooks.state) > position {
		// Existing state
		getter := func() T {
			stateValue := currentFiber.hooks.state[position].(T)
			fmt.Printf("stateValue: Retrieved existing state at position %d: %v\n", position, stateValue)
			return stateValue
		}
		setter := func(newValue T) {
			if !reflect.DeepEqual(currentFiber.hooks.state[position], newValue) {
				currentFiber.hooks.state[position] = newValue
				scheduleUpdate(currentFiber)
			}
		}
		return getter, setter
	} else {
		// Initial state
		fmt.Printf("stateValue: Initializing state at position %d with value: %v\n", position, initialValue)
		currentFiber.hooks.state = append(currentFiber.hooks.state, initialValue)
		getter := func() T {
			stateValue := currentFiber.hooks.state[position].(T)
			fmt.Printf("stateValue: Retrieved initialized state at position %d: %v\n", position, stateValue)
			return stateValue
		}
		setter := func(newValue T) {
			if !reflect.DeepEqual(currentFiber.hooks.state[position], newValue) {
				currentFiber.hooks.state[position] = newValue
				scheduleUpdate(currentFiber)
			}
		}
		return getter, setter
	}
}

type memoizedValue struct {
	value interface{}
	deps  []interface{}
}

// Extend the Hooks struct to include memoized values
type Hooks struct {
	state []interface{}
	deps  [][]interface{}
	index int
	memos []memoizedValue
}

func useEffect(effect func(), deps []interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = &Hooks{
			state: []interface{}{},
			deps:  [][]interface{}{},
		}
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	if len(currentFiber.hooks.deps) <= position {
		// First time this effect is used
		currentFiber.hooks.deps = append(currentFiber.hooks.deps, deps)
		// Schedule the effect
		currentFiber.effects = append(currentFiber.effects, effect)
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		var shouldRunEffect bool

		if deps == nil {
			// If deps is nil, run effect on every render
			shouldRunEffect = true
		} else if len(deps) == 0 {
			// If deps is an empty slice, only run once (on mount)
			shouldRunEffect = prevDeps == nil
		} else {
			// Otherwise, check if dependencies have changed
			shouldRunEffect = !areDepsEqual(prevDeps, deps)
		}

		if shouldRunEffect {
			// Update the dependencies
			currentFiber.hooks.deps[position] = deps
			// Schedule the effect
			currentFiber.effects = append(currentFiber.effects, effect)
		}
	}
}

func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	if prevDeps == nil || newDeps == nil {
		return false
	}
	if len(prevDeps) != len(newDeps) {
		return false
	}
	for i := range prevDeps {
		if !reflect.DeepEqual(prevDeps[i], newDeps[i]) {
			return false
		}
	}
	return true
}

func useMemo(compute func() interface{}, deps []interface{}) interface{} {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = &Hooks{
			state: []interface{}{},
			deps:  [][]interface{}{},
			memos: []memoizedValue{},
		}
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	if len(currentFiber.hooks.memos) <= position {
		// First time this memo is used
		value := compute()
		currentFiber.hooks.memos = append(currentFiber.hooks.memos, memoizedValue{
			value: value,
			deps:  deps,
		})
		return value
	}

	memo := &currentFiber.hooks.memos[position]
	if areDepsEqual(memo.deps, deps) {
		// Dependencies changed, recompute the value
		value := compute()
		memo.value = value
		memo.deps = deps
		return value
	}

	// Dependencies haven't changed, return the memoized value
	return memo.value
}

// Fiber represents a unit of work in the virtual DOM tree.
type Fiber struct {
	typeOf    interface{}
	props     map[string]interface{}
	hooks     *Hooks
	parent    *Fiber
	dom       js.Value
	alternate *Fiber
	child     *Fiber
	sibling   *Fiber
	effectTag string
	effects   []func()
}

// getCurrentFiber retrieves the current working fiber.
func getCurrentFiber() *Fiber {
	return wipFiber
}

// scheduleUpdate triggers a re-render of the component.
func scheduleUpdate(fiber *Fiber) {
	fmt.Println("scheduleUpdate: Scheduling update")
	wipRoot = &Fiber{
		typeOf:    "ROOT",
		dom:       currentRoot.dom,
		props:     currentRoot.props,
		alternate: currentRoot,
	}
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	fmt.Println("scheduleUpdate: wipRoot set and workLoop scheduled")
	requestIdleCallback(workLoop)
}

// render starts the rendering process.
func render(element *Element, container js.Value) {
	fmt.Println("render: Starting rendering process.")
	wipRoot = &Fiber{
		typeOf:    "ROOT", // Assign a type to the root fiber
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{element}},
		alternate: currentRoot,
	}
	fmt.Println("render: Root fiber created.")
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	fmt.Println("render: Scheduling work loop.")
	requestIdleCallback(workLoop)
}

// workLoop performs work until there is no more work left or the deadline expires.
func workLoop(deadline js.Value) {
	fmt.Println("workLoop: Starting work loop.")
	var shouldYield bool = false
	for nextUnitOfWork != nil && !shouldYield {
		fmt.Println("workLoop: Performing a unit of work.")
		nextUnitOfWork = performUnitOfWork(nextUnitOfWork)
		shouldYield = deadline.Call("timeRemaining").Float() < 1
		fmt.Printf("workLoop: timeRemaining=%f, shouldYield=%v\n", deadline.Call("timeRemaining").Float(), shouldYield)
	}

	if wipRoot != nil && nextUnitOfWork == nil {
		fmt.Println("workLoop: No more units of work. Committing root.")
		commitRoot()
	}

	if nextUnitOfWork != nil {
		fmt.Println("workLoop: Work remains. Scheduling next work loop.")
		requestIdleCallback(workLoop)
	} else {
		fmt.Println("workLoop: All work completed.")
	}
}

// performUnitOfWork performs a single unit of work.
func performUnitOfWork(fiber *Fiber) *Fiber {
	if fiber == nil {
		fmt.Println("performUnitOfWork: Fiber is nil.")
		return nil
	}

	fmt.Printf("performUnitOfWork: Processing fiber of type %v.\n", fiber.typeOf)

	if fiber.typeOf == nil || fiber.typeOf == "ROOT" {
		fmt.Println("performUnitOfWork: Fiber has typeOf nil or ROOT, reconciling children.")
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
				copy(wipFiber.hooks.deps, oldHooks.deps)
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
			fmt.Printf("performUnitOfWork: Handling host component of type '%s'.\n", fiber.typeOf.(string))
			if fiber.dom.IsUndefined() || fiber.dom.IsNull() {
				fmt.Println("performUnitOfWork: Creating DOM node for host component.")
				fiber.dom = createDom(fiber)
				fmt.Println("performUnitOfWork: DOM node created.")
			}

			if fiber.props == nil {
				fmt.Println("performUnitOfWork: Fiber props are nil. Skipping children reconciliation.")
				return nil
			}

			if propsChildren, ok := fiber.props["children"]; ok {
				fmt.Println("performUnitOfWork: Reconciling children of host component.")
				elements := propsChildren.([]interface{})
				reconcileChildren(fiber, elements)
			}
		default:
			fmt.Printf("performUnitOfWork: Unhandled fiber type %T.\n", fiber.typeOf)
		}
	}

	fmt.Printf("performUnitOfWork: Completed processing fiber of type %v.\n", fiber.typeOf)

	// Traverse to child fibers
	if fiber.child != nil {
		fmt.Printf("performUnitOfWork: Moving to child fiber of type %v.\n", fiber.child.typeOf)
		return fiber.child
	}

	nextFiber := fiber
	for nextFiber != nil {
		if nextFiber.sibling != nil {
			fmt.Printf("performUnitOfWork: Moving to sibling fiber of type %v.\n", nextFiber.sibling.typeOf)
			return nextFiber.sibling
		}
		fmt.Println("performUnitOfWork: Moving up to parent fiber.")
		nextFiber = nextFiber.parent
	}
	fmt.Println("performUnitOfWork: No more fibers to process.")
	return nil
}

// createDom creates a DOM node from a fiber.
func createDom(fiber *Fiber) js.Value {
	fmt.Printf("createDom: Creating DOM for fiber type %v\n", fiber.typeOf)
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
		fmt.Println("createDom: Function component, no DOM node created")
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
			fmt.Println("createDom: Setting innerHTML")
			dom.Set("innerHTML", htmlContent)
			continue
		}
		if len(name) > 2 && name[:2] == "on" {
			// Event handlers
			eventType := strings.ToLower(name[2:]) // Convert event type to lowercase
			fmt.Printf("createDom: Adding event listener for %s\n", eventType)

			// Ensure the value is of the correct function type
			eventHandler, ok := value.(js.Func)
			if !ok {
				fmt.Printf("createDom: Event handler for %s is not a js.Func\n", eventType)
				continue
			}

			dom.Call("addEventListener", eventType, eventHandler)
			continue
		}
		if name == "class" {
			// Handle 'class' attribute using setAttribute
			fmt.Printf("createDom: Setting attribute 'class' to '%v'\n", value)
			dom.Call("setAttribute", "class", value)
			continue
		}
		// Set other properties directly
		fmt.Printf("createDom: Setting property '%s' to '%v'\n", name, value)
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
	fmt.Println("commitRoot: Starting to commit changes to DOM")
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
	// fmt.Println("commitRoot: Finished committing changes to DOM")

	// Execute effects after committing
	executeEffects()
}

func executeEffects() {
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

	// Execute effects
	for _, fiber := range effectFibers {
		for _, effect := range fiber.effects {
			if effect != nil {
				effect()
			}
		}
		// Clear the effects after executing them
		fiber.effects = []func(){}
	}
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
		fmt.Printf("commitDeletion: Removing child %v from parent %v\n", fiber.dom, domParent)
		domParent.Call("removeChild", fiber.dom)

		// Release event callbacks associated with this fiber
		if fiber.hooks != nil {
			for _, state := range fiber.hooks.state {
				if fn, ok := state.(js.Func); ok {
					fmt.Println("commitDeletion: Releasing event callback")
					fn.Release()
				}
			}
		}
	} else if fiber.child != nil {
		fmt.Println("commitDeletion: Deleting child fibers recursively")
		commitDeletion(fiber.child, domParent)
	}
}

func updateDom(dom js.Value, oldProps, newProps map[string]interface{}) {
	// fmt.Println("updateDom: Updating DOM properties")

	// 1. Remove old or changed event listeners
	for name, oldValue := range oldProps {
		if strings.HasPrefix(name, "on") {
			eventType := strings.ToLower(name[2:])
			fmt.Printf("updateDom: Removing event listener for %s\n", eventType)
			dom.Call("removeEventListener", eventType, oldValue.(js.Func))
		}

		// Remove properties that no longer exist, excluding event listeners
		if newProps[name] == nil && !strings.HasPrefix(name, "on") {
			fmt.Printf("updateDom: Removing property '%s'\n", name)
			dom.Set(name, js.Undefined())
		}
	}

	// 2. Add new or changed properties and event listeners
	for name, value := range newProps {
		if name == "children" {
			continue
		}
		if name == "dangerouslySetInnerHTML" {
			htmlContent := value.(map[string]string)["__html"]
			// fmt.Println("updateDom: Updating innerHTML")
			dom.Set("innerHTML", htmlContent)
			continue
		}
		if strings.HasPrefix(name, "on") {
			eventType := strings.ToLower(name[2:])
			// fmt.Printf("updateDom: Adding event listener for %s\n", eventType)
			dom.Call("addEventListener", eventType, value.(js.Func))
			continue
		}
		if name == "class" {
			// fmt.Printf("updateDom: Setting attribute 'class' to '%v'\n", value)
			dom.Call("setAttribute", "class", value)
			continue
		}
		fmt.Printf("updateDom: Setting property '%s' to '%v'\n", name, value)
		dom.Set(name, value)
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

type FetchOptions struct {
	Method  string
	Headers map[string]interface{}
	Body    interface{}
}

func useFetch(url string, options ...FetchOptions) (func() FetchState, func()) {
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