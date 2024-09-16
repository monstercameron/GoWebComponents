package fiber

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"syscall/js"
	"time"
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

type Hooks struct {
	state              []interface{}
	deps               [][]interface{}
	effects            []func()
	effectsInitialized []bool
	index              int
}

func useEffect(effect func(), deps []interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = &Hooks{
			state:              make([]interface{}, 0),
			deps:               make([][]interface{}, 0),
			effects:            make([]func(), 0),
			effectsInitialized: make([]bool, 0),
		}
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Extend slices if necessary
	for len(currentFiber.hooks.effectsInitialized) <= position {
		currentFiber.hooks.effectsInitialized = append(currentFiber.hooks.effectsInitialized, false)
		currentFiber.hooks.deps = append(currentFiber.hooks.deps, nil)
	}

	isFirstRun := !currentFiber.hooks.effectsInitialized[position]
	var shouldRunEffect bool

	if isFirstRun {
		shouldRunEffect = true
		fmt.Printf("useEffect: First run at position %d, will execute effect\n", position)
		currentFiber.hooks.effectsInitialized[position] = true
	} else if deps == nil {
		shouldRunEffect = true
		fmt.Printf("useEffect: Nil deps at position %d, will execute effect\n", position)
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		shouldRunEffect = !areDepsEqual(prevDeps, deps)
		fmt.Printf("useEffect: Comparing deps at position %d, should run effect: %v\n", position, shouldRunEffect)
	}

	if shouldRunEffect {
		fmt.Printf("useEffect: Scheduling effect at position %d\n", position)
		currentFiber.hooks.effects = append(currentFiber.hooks.effects, effect)
	} else {
		fmt.Printf("useEffect: Not scheduling effect at position %d, deps unchanged\n", position)
	}

	// Always update the deps
	currentFiber.hooks.deps[position] = deps
}

func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	if prevDeps == nil && newDeps == nil {
		return true
	}
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
}

// Global variables for tracking the current fiber and root.
var (
	wipRoot        *Fiber
	currentRoot    *Fiber
	nextUnitOfWork *Fiber
	deletions      []*Fiber
	wipFiber       *Fiber
)

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
			fmt.Println("performUnitOfWork: Rendering function component.")
			componentFunc := fiber.typeOf.(func(map[string]interface{}) *Element)
			wipFiber = fiber
			fmt.Printf("performUnitOfWork: Current fiber set to %v.\n", wipFiber.typeOf)

			// Preserve hooks from alternate fiber
			var oldHooks *Hooks
			if fiber.alternate != nil && fiber.alternate.hooks != nil {
				oldHooks = fiber.alternate.hooks
				fmt.Println("performUnitOfWork: Preserving hooks from alternate fiber.")
			} else {
				fmt.Println("performUnitOfWork: No hooks found in alternate fiber.")
			}

			// Initialize hooks
			if oldHooks != nil {
				wipFiber.hooks = &Hooks{
					state: make([]interface{}, len(oldHooks.state)),
					deps:  make([][]interface{}, len(oldHooks.deps)),
				}
				copy(wipFiber.hooks.state, oldHooks.state)
				copy(wipFiber.hooks.deps, oldHooks.deps)
				fmt.Println("performUnitOfWork: Copied hooks from alternate fiber.")
			} else {
				wipFiber.hooks = &Hooks{
					state: []interface{}{},
					deps:  [][]interface{}{},
				}
				fmt.Println("performUnitOfWork: Initialized new hooks for fiber.")
			}
			wipFiber.hooks.index = 0

			element := componentFunc(fiber.props)
			if element == nil {
				fmt.Println("performUnitOfWork: Function component returned nil element.")
				return nil
			}

			fmt.Println("performUnitOfWork: Reconciling children from function component.")
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

var eventCallbacks []js.Func // Global slice to keep event callbacks alive

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
	fmt.Println("executeEffects: Executing side effects")
	var effectFibers []*Fiber
	var collectEffects func(fiber *Fiber)
	collectEffects = func(fiber *Fiber) {
		if fiber == nil {
			return
		}
		if fiber.hooks != nil && len(fiber.hooks.effects) > 0 {
			effectFibers = append(effectFibers, fiber)
		}
		collectEffects(fiber.child)
		collectEffects(fiber.sibling)
	}

	// Collect fibers with effects starting from the root
	collectEffects(currentRoot.child)

	// Execute effects
	for _, fiber := range effectFibers {
		fmt.Printf("executeEffects: Executing effects for fiber %p\n", fiber)
		for i, effect := range fiber.hooks.effects {
			fmt.Printf("executeEffects: Running effect %d\n", i)
			effect()
		}
		// Clear the effects after executing them
		fiber.hooks.effects = fiber.hooks.effects[:0]
	}

	// Reset hook index for next render
	resetHookIndex(currentRoot.child)
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

var rafCallbacks []js.Func // Global slice to keep callbacks alive

// requestIdleCallback schedules work during idle periods.
func requestIdleCallback(callback func(js.Value)) {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback(args[0])
		return nil
	})
	rafCallbacks = append(rafCallbacks, cb) // Keep the function alive
	js.Global().Call("requestIdleCallback", cb)
}

// BlogPost represents a blog post structure.
type BlogPost struct {
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Tags        []string  `json:"tags"`
	Content     string    `json:"content"`
}

// getBlogPosts fetches blog posts from the API.
func getBlogPosts(callback func([]BlogPost)) {
	fmt.Println("getBlogPosts: Fetching posts")
	fetchPromise := js.Global().Call("fetch", "http://localhost:8080/api/blog")
	fetchPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		if !response.Get("ok").Bool() {
			errorMsg := fmt.Sprintf("HTTP error! status: %s", response.Get("status").String())
			fmt.Println("getBlogPosts:", errorMsg)
			callback(nil)
			return nil
		}
		response.Call("json").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			data := args[0]
			jsonStr := js.Global().Get("JSON").Call("stringify", data).String()
			var posts []BlogPost
			err := json.Unmarshal([]byte(jsonStr), &posts)
			if err != nil {
				fmt.Println("Error parsing blog posts:", err)
				callback(nil)
			} else {
				fmt.Println("getBlogPosts: Successfully fetched posts")
				callback(posts)
			}
			return nil
		}))
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := args[0]
		fmt.Println("Fetch error:", err)
		callback(nil)
		return nil
	}))
}

// Define a global empty dependency array to ensure useEffect runs only once
var emptyDeps = []interface{}{}

// BlogListComponent represents the main component handling blog list and single blog view.
func BlogListComponent(props map[string]interface{}) *Element {
	fmt.Println("BlogListComponent: Rendering")
	blogs, setBlogs := useState([]BlogPost{})
	currentPage, setCurrentPage := useState(1)
	currentBlog, setCurrentBlog := useState[*BlogPost](nil)

	// Event handlers
	viewBlog := func(slug string) js.Func {
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				event := args[0]
				event.Call("preventDefault") // Prevent default behavior, though using <button> minimizes this need
			}
			fmt.Printf("viewBlog: Viewing blog with slug %s\n", slug)
			for _, blog := range blogs() {
				if blog.Slug == slug {
					blogCopy := blog // Create a copy to avoid pointer reuse
					setCurrentBlog(&blogCopy)
					return nil
				}
			}
			setCurrentBlog(nil)
			return nil
		})
		eventCallbacks = append(eventCallbacks, cb) // Keep callback alive
		return cb
	}

	backToList := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fmt.Println("backToList: Going back to blog list")
		setCurrentBlog(nil)
		return nil
	})
	eventCallbacks = append(eventCallbacks, backToList) // Keep callback alive

	goToPage := func(page int) js.Func {
		cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			fmt.Printf("goToPage: Going to page %d\n", page)
			setCurrentPage(page)
			return nil
		})
		eventCallbacks = append(eventCallbacks, cb) // Keep callback alive
		return cb
	}

	// Fetch blogs on mount
	useEffect(func() {
		fmt.Println("useEffect: Fetching blogs")
		getBlogPosts(func(bp []BlogPost) {
			if bp != nil {
				fmt.Println("useEffect: Setting blogs state")
				setBlogs(bp)
			} else {
				fmt.Println("useEffect: No posts fetched")
			}
		})
	}, []interface{}{}) // Use the static emptyDeps

	// Render functions
	blogListItem := func(post BlogPost) *Element {
		fmt.Printf("blogListItem: Creating list item for %s\n", post.Title)
		return createElement("div", map[string]interface{}{
			"class": "mb-6 p-6 bg-white rounded-lg shadow hover:shadow-lg transition-shadow duration-200",
		},
			createElement("h2", map[string]interface{}{"class": "text-2xl font-bold mb-2"},
				createElement("button", map[string]interface{}{
					"onclick": viewBlog(post.Slug),
					"class":   "text-blue-500 hover:underline focus:outline-none",
				}, Text(post.Title)),
			),
			createElement("p", map[string]interface{}{"class": "text-gray-600 mb-2"}, Text(post.Date.Format("January 2, 2006"))),
			createElement("p", map[string]interface{}{"class": "text-gray-700"}, Text(post.Description)),
		)
	}

	// Pagination component
	paginationComponent := func(totalPages int) *Element {
		fmt.Printf("paginationComponent: Creating pagination for %d pages\n", totalPages)
		paginationItems := []interface{}{}

		if currentPage() > 1 {
			prevPage := currentPage() - 1
			paginationItems = append(paginationItems, createElement("button", map[string]interface{}{
				"class":   "mx-1 px-3 py-1 border bg-white text-blue-500 rounded-full hover:bg-blue-500 hover:text-white transition duration-200",
				"onclick": goToPage(prevPage),
			}, Text("Previous")))
		}

		for i := 1; i <= totalPages; i++ {
			page := i
			pageClass := "mx-1 px-3 py-1 border rounded-full"
			if currentPage() == i {
				pageClass += " bg-blue-500 text-white"
			} else {
				pageClass += " bg-white text-blue-500 hover:bg-blue-500 hover:text-white transition duration-200"
			}
			paginationItems = append(paginationItems, createElement("button", map[string]interface{}{
				"class":   pageClass,
				"onclick": goToPage(page),
			}, Text(fmt.Sprintf("%d", i))))
		}

		if currentPage() < totalPages {
			nextPage := currentPage() + 1
			paginationItems = append(paginationItems, createElement("button", map[string]interface{}{
				"class":   "mx-1 px-3 py-1 border bg-white text-blue-500 rounded-full hover:bg-blue-500 hover:text-white transition duration-200",
				"onclick": goToPage(nextPage),
			}, Text("Next")))
		}

		return createElement("div", map[string]interface{}{"class": "flex justify-center mt-4"}, paginationItems...)
	}

	// Breadcrumbs component
	breadcrumbsComponent := func() *Element {
		fmt.Println("breadcrumbsComponent: Creating breadcrumbs")
		breadcrumbs := []interface{}{
			createElement("a", map[string]interface{}{
				"href":  "/",
				"class": "text-blue-500 hover:underline transition duration-200",
			}, Text("Home")),
		}
		if currentBlog() != nil {
			breadcrumbs = append(breadcrumbs, Text(" / "))
			breadcrumbs = append(breadcrumbs, createElement("a", map[string]interface{}{
				"href":    "#",
				"onclick": backToList,
				"class":   "text-blue-500 hover:underline transition duration-200",
			}, Text("Blog")))
			breadcrumbs = append(breadcrumbs, Text(" / "))
			breadcrumbs = append(breadcrumbs, createElement("span", nil, Text(currentBlog().Title)))
		} else {
			breadcrumbs = append(breadcrumbs, Text(" / "))
			breadcrumbs = append(breadcrumbs, createElement("span", nil, Text("Blog")))
		}
		return createElement("nav", map[string]interface{}{"class": "text-sm mb-4 text-gray-700"}, breadcrumbs...)
	}

	var content *Element

	if currentBlog() != nil {
		// Render single blog post
		fmt.Println("BlogListComponent: Rendering single blog post view")
		post := currentBlog()

		// Find index of current blog in blogs
		currentIndex := -1
		for i, b := range blogs() {
			if b.Slug == post.Slug {
				currentIndex = i
				break
			}
		}

		// Determine previous and next posts
		var prevPost, nextPost *BlogPost
		if currentIndex > 0 {
			prevPost = &blogs()[currentIndex-1]
		}
		if currentIndex >= 0 && currentIndex < len(blogs())-1 {
			nextPost = &blogs()[currentIndex+1]
		}

		// Build navigation buttons
		navButtons := []interface{}{}

		if prevPost != nil {
			navButtons = append(navButtons, createElement("button", map[string]interface{}{
				"class":   "mx-1 px-3 py-1 bg-blue-500 text-white rounded-full hover:bg-blue-600 transition duration-200",
				"onclick": viewBlog(prevPost.Slug), // Use viewBlog with prevPost.Slug
			}, Text("Previous")))
		}

		if nextPost != nil {
			navButtons = append(navButtons, createElement("button", map[string]interface{}{
				"class":   "mx-1 px-3 py-1 bg-blue-500 text-white rounded-full hover:bg-blue-600 transition duration-200",
				"onclick": viewBlog(nextPost.Slug), // Use viewBlog with nextPost.Slug
			}, Text("Next")))
		}

		// Back button
		backButton := createElement("button", map[string]interface{}{
			"class":   "mb-4 px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition duration-200",
			"onclick": backToList,
		}, Text("Back to List"))

		content = createElement("div", nil,
			breadcrumbsComponent(),
			backButton,
			createElement("article", map[string]interface{}{"class": "prose lg:prose-xl"},
				createElement("h1", map[string]interface{}{"class": "text-3xl font-bold mb-4"}, Text(post.Title)),
				createElement("p", map[string]interface{}{"class": "text-gray-600 mb-4"}, Text(post.Date.Format("January 2, 2006"))),
				createElement("div", map[string]interface{}{"dangerouslySetInnerHTML": map[string]string{"__html": post.Content}}),
			),
			createElement("div", map[string]interface{}{"class": "flex justify-between mt-8"}, navButtons...),
		)
	} else {
		// Render blog list with pagination
		fmt.Println("BlogListComponent: Rendering blog list view")
		totalBlogs := len(blogs())
		blogsPerPage := 3
		totalPages := (totalBlogs + blogsPerPage - 1) / blogsPerPage
		if totalPages == 0 {
			totalPages = 1
		}
		if currentPage() > totalPages {
			setCurrentPage(totalPages)
		} else if currentPage() < 1 {
			setCurrentPage(1)
		}
		startIndex := (currentPage() - 1) * blogsPerPage
		endIndex := startIndex + blogsPerPage
		if endIndex > totalBlogs {
			endIndex = totalBlogs
		}
		blogsForPage := blogs()[startIndex:endIndex]

		blogListItems := []interface{}{}
		for _, post := range blogsForPage {
			blogListItems = append(blogListItems, blogListItem(post))
		}

		content = createElement("div", nil,
			breadcrumbsComponent(),
			createElement("div", nil, blogListItems...),
			paginationComponent(totalPages),
		)
	}

	return createElement("div", map[string]interface{}{"class": "container mx-auto p-4 pb-8"}, content)
}



