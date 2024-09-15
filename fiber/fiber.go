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
	}
	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	if len(currentFiber.hooks.state) > position {
		// Existing state
		getter := func() T {
			return currentFiber.hooks.state[position].(T)
		}
		setter := func(newValue T) {
			currentFiber.hooks.state[position] = newValue
			scheduleUpdate(currentFiber)
		}
		return getter, setter
	} else {
		// Initial state
		currentFiber.hooks.state = append(currentFiber.hooks.state, initialValue)
		getter := func() T {
			return currentFiber.hooks.state[position].(T)
		}
		setter := func(newValue T) {
			currentFiber.hooks.state[position] = newValue
			scheduleUpdate(currentFiber)
		}
		return getter, setter
	}
}

// useEffect handles side effects in a component.
func useEffect(effect func(), deps []interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = &Hooks{}
	}
	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	var hasChanged bool
	if len(currentFiber.hooks.deps) > position {
		hasChanged = !areDepsEqual(currentFiber.hooks.deps[position], deps)
	} else {
		hasChanged = true
	}

	if hasChanged {
		// Store the effect to be executed after commit
		if currentFiber.hooks.effects == nil {
			currentFiber.hooks.effects = make([]func(), 0)
		}
		currentFiber.hooks.effects = append(currentFiber.hooks.effects, effect)
		if len(currentFiber.hooks.deps) > position {
			currentFiber.hooks.deps[position] = deps
		} else {
			currentFiber.hooks.deps = append(currentFiber.hooks.deps, deps)
		}
	}
}

func areDepsEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// Hooks stores the state and effect dependencies of a component.
type Hooks struct {
	state   []interface{}
	deps    [][]interface{}
	index   int
	effects []func()
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
		dom:       currentRoot.dom,
		props:     currentRoot.props,
		alternate: currentRoot,
	}
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	requestIdleCallback(workLoop)
}

// render starts the rendering process.
func render(element *Element, container js.Value) {
	fmt.Println("render: Starting rendering process")
	wipRoot = &Fiber{
		dom:       container,
		props:     map[string]interface{}{"children": []interface{}{element}},
		alternate: currentRoot,
	}
	nextUnitOfWork = wipRoot
	deletions = []*Fiber{}
	requestIdleCallback(workLoop)
}

// workLoop performs work until there is no more work left.
func workLoop(deadline js.Value) {
	var shouldYield bool = false
	for nextUnitOfWork != nil && !shouldYield {
		nextUnitOfWork = performUnitOfWork(nextUnitOfWork)
		// Corrected line
		shouldYield = deadline.Call("timeRemaining").Float() < 1
	}

	if wipRoot != nil && nextUnitOfWork == nil {
		fmt.Println("workLoop: Committing root")
		commitRoot()
	}

	// Continue the work loop if there's more work to do
	if nextUnitOfWork != nil {
		requestIdleCallback(workLoop)
	} else {
		fmt.Println("workLoop: No more work to do")
	}
}

// performUnitOfWork performs a single unit of work.
// performUnitOfWork performs a single unit of work.
func performUnitOfWork(fiber *Fiber) *Fiber {
    if fiber == nil {
        fmt.Println("performUnitOfWork: Fiber is nil")
        return nil
    }
    fmt.Printf("performUnitOfWork: Fiber type %T\n", fiber.typeOf)

    // If typeOf is nil, skip processing this fiber but continue with its children
    if fiber.typeOf == nil {
        fmt.Println("performUnitOfWork: typeOf is nil for fiber, skipping processing")
    } else {
        switch fiber.typeOf.(type) {
        case func(map[string]interface{}) *Element:
            // Function component
            fmt.Println("performUnitOfWork: Rendering function component")
            componentFunc := fiber.typeOf.(func(map[string]interface{}) *Element)
            wipFiber = fiber

            // Ensure hooks are correctly initialized
            var oldHooks *Hooks
            if fiber.alternate != nil && fiber.alternate.hooks != nil {
                oldHooks = fiber.alternate.hooks
            }

            if oldHooks != nil {
                fmt.Println("performUnitOfWork: Preserving hooks from alternate fiber")
            } else {
                fmt.Println("performUnitOfWork: No hooks found in alternate fiber")
            }

            wipFiber.hooks = &Hooks{
                state: make([]interface{}, len(oldHooks.state)),
                deps:  make([][]interface{}, len(oldHooks.deps)),
            }

            if oldHooks != nil {
                copy(wipFiber.hooks.state, oldHooks.state)
                copy(wipFiber.hooks.deps, oldHooks.deps)
            }

            wipFiber.hooks.index = 0

            element := componentFunc(fiber.props)
            if element == nil {
                fmt.Println("performUnitOfWork: Function component returned nil element")
                return nil
            }

            reconcileChildren(fiber, []interface{}{element})
        case string:
            // Host component (HTML element)
            fmt.Printf("performUnitOfWork: Handling host component of type %s\n", fiber.typeOf.(string))

            if fiber.dom.IsUndefined() || fiber.dom.IsNull() {
                fmt.Println("performUnitOfWork: Creating DOM for host component")
                fiber.dom = createDom(fiber)
            }

            if fiber.props == nil {
                fmt.Println("performUnitOfWork: props is nil for fiber")
                return nil
            }

            if propsChildren, ok := fiber.props["children"]; ok {
                elements := propsChildren.([]interface{})
                reconcileChildren(fiber, elements)
            }
        default:
            fmt.Printf("performUnitOfWork: Unhandled fiber type %T\n", fiber.typeOf)
        }
    }

    // Traverse to child fibers regardless of typeOf
    if fiber.child != nil {
        return fiber.child
    }

    nextFiber := fiber
    for nextFiber != nil {
        if nextFiber.sibling != nil {
            return nextFiber.sibling
        }
        nextFiber = nextFiber.parent
    }
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
			dom.Set("innerHTML", htmlContent)
		} else if len(name) > 2 && name[:2] == "on" {
			eventType := strings.ToLower(name[2:]) // Convert event type to lowercase
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				value.(func(js.Value))(args[0])
				return nil
			})
			eventCallbacks = append(eventCallbacks, cb) // Keep callback alive
			dom.Call("addEventListener", eventType, cb)
		} else {
			dom.Set(name, value)
		}
	}
	return dom
}

// reconcileChildren reconciles the children of a fiber.
func reconcileChildren(wipFiber *Fiber, elements []interface{}) {
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
			if reflect.DeepEqual(element.(*Element).Type, oldFiber.typeOf) {
				sameType = true
			}
		}

		if sameType {
			// Update the node
			newFiber = &Fiber{
				typeOf:    oldFiber.typeOf,
				props:     element.(*Element).Props,
				dom:       oldFiber.dom,
				parent:    wipFiber,
				alternate: oldFiber,
				effectTag: "UPDATE",
			}
		} else if element != nil {
			// Add this node
			newFiber = &Fiber{
				typeOf:    element.(*Element).Type,
				props:     element.(*Element).Props,
				dom:       js.Value{},
				parent:    wipFiber,
				effectTag: "PLACEMENT",
			}
		}

		if oldFiber != nil && !sameType {
			// Delete the old node
			oldFiber.effectTag = "DELETION"
			deletions = append(deletions, oldFiber)
		}

		if oldFiber != nil {
			oldFiber = oldFiber.sibling
		}

		if index == 0 {
			wipFiber.child = newFiber
		} else if element != nil {
			prevSibling.sibling = newFiber
		}

		prevSibling = newFiber
		index++
	}
}

// commitRoot commits the changes to the DOM.
func commitRoot() {
	fmt.Println("commitRoot: Starting to commit changes to DOM")
	for _, deletion := range deletions {
		commitWork(deletion)
	}
	commitWork(wipRoot.child)
	currentRoot = wipRoot
	wipRoot = nil
	fmt.Println("commitRoot: Finished committing changes")

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
		for _, effect := range fiber.hooks.effects {
			effect()
		}
		// Clear the effects after executing them
		fiber.hooks.effects = nil
	}
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
		return
	}
	domParent := domParentFiber.dom
	if fiber.effectTag == "PLACEMENT" {
		if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
			fmt.Printf("commitWork: Appending child %v to parent %v\n", fiber.dom, domParent)
			domParent.Call("appendChild", fiber.dom)
		} else {
			// If the fiber doesn't have a DOM node, commit its children
			commitWork(fiber.child)
			return
		}
	} else if fiber.effectTag == "UPDATE" && !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
		// Handle updates to DOM node properties here
		fmt.Println("commitWork: Updating DOM node properties")
	} else if fiber.effectTag == "DELETION" {
		commitDeletion(fiber, domParent)
		return
	}
	commitWork(fiber.child)
	commitWork(fiber.sibling)
}

func commitDeletion(fiber *Fiber, domParent js.Value) {
	if !fiber.dom.IsUndefined() && !fiber.dom.IsNull() {
		fmt.Printf("commitDeletion: Removing child %v from parent %v\n", fiber.dom, domParent)
		domParent.Call("removeChild", fiber.dom)
		// Release event callbacks associated with this fiber
		// Implement a way to track and release event callbacks
	} else if fiber.child != nil {
		commitDeletion(fiber.child, domParent)
	}
}

var rafCallbacks []js.Func // Global slice to keep callbacks alive

func requestIdleCallback(callback func(js.Value)) {
	cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		callback(args[0])
		return nil
	})
	rafCallbacks = append(rafCallbacks, cb) // Keep the function alive
	js.Global().Call("requestIdleCallback", cb)
}

type BlogPost struct {
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Date        time.Time `json:"date"`
	Tags        []string  `json:"tags"`
	Content     string    `json:"content"`
}

func getBlogPosts(callback func([]BlogPost)) {
	fmt.Println("getBlogPosts: Fetching posts")
	fetchPromise := js.Global().Call("fetch", "http://localhost:8080/api/blog")
	fetchPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		response := args[0]
		if !response.Get("ok").Bool() {
			errorMsg := fmt.Sprintf("HTTP error! status: %s", response.Get("status").String())
			fmt.Println(errorMsg)
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

// Example5 reimplemented using the new architecture.
func Example5() {
	fmt.Println("Example5: Starting")
	// Define the BlogListComponent
	BlogListComponent := func(props map[string]interface{}) *Element {
		fmt.Println("BlogListComponent: Rendering")
		blogs, setBlogs := useState([]BlogPost{})
		currentPage, setCurrentPage := useState(1)
		currentBlog, setCurrentBlog := useState[*BlogPost](nil)

		// Event handlers
		viewBlog := func(slug string) func(js.Value) {
			return func(event js.Value) {
				fmt.Printf("viewBlog: Viewing blog with slug %s\n", slug)
				for _, blog := range blogs() {
					if blog.Slug == slug {
						setCurrentBlog(&blog)
						return
					}
				}
				setCurrentBlog(nil)
			}
		}

		backToList := func(event js.Value) {
			fmt.Println("backToList: Going back to blog list")
			setCurrentBlog(nil)
		}

		goToPage := func(page int) func(js.Value) {
			return func(event js.Value) {
				fmt.Printf("goToPage: Going to page %d\n", page)
				setCurrentPage(page)
			}
		}

		// Fetch blogs on mount
		useEffect(func() {
			fmt.Println("useEffect: Fetching blogs")
			getBlogPosts(func(bp []BlogPost) {
				if bp != nil {
					setBlogs(bp)
				} else {
					fmt.Println("No posts fetched")
				}
			})
		}, []interface{}{})

		// Render functions
		blogListItem := func(post BlogPost) *Element {
			return createElement("div", map[string]interface{}{
				"class": "mb-6 p-6 bg-white rounded-lg shadow hover:shadow-lg transition-shadow duration-200",
			},
				createElement("h2", map[string]interface{}{"class": "text-2xl font-bold mb-2"},
					createElement("a", map[string]interface{}{
						"href":    "#",
						"onclick": viewBlog(post.Slug),
						"class":   "text-blue-500 hover:underline",
					}, Text(post.Title)),
				),
				createElement("p", map[string]interface{}{"class": "text-gray-600 mb-2"}, Text(post.Date.Format("January 2, 2006"))),
				createElement("p", map[string]interface{}{"class": "text-gray-700"}, Text(post.Description)),
			)
		}

		// Pagination component
		paginationComponent := func(totalPages int) *Element {
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
					"onclick": viewBlog(prevPost.Slug),
				}, Text("Previous")))
			}

			if nextPost != nil {
				navButtons = append(navButtons, createElement("button", map[string]interface{}{
					"class":   "mx-1 px-3 py-1 bg-blue-500 text-white rounded-full hover:bg-blue-600 transition duration-200",
					"onclick": viewBlog(nextPost.Slug),
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

	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	render(createElement(BlogListComponent, nil), container)
}
