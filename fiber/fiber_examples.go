// ./fiber/fiber_examples.go

package fiber

import (
	"encoding/json"
	"fmt"
	"syscall/js"
	"time"
)

var (
	// Define a global empty dependency array to ensure useEffect runs only once
	emptyDeps = []interface{}{}
)

// Example1 is a function that renders a calculator component using GoWebComponents.
// It initializes the state for the calculator, handles button clicks for numbers and operators,
// evaluates expressions, and renders the calculator UI.
//
// The calculator component consists of a display area for the previous expression and the input expression,
// as well as buttons for numbers, operators, clear, and equal.
//
// The function takes no parameters and returns no values.
// It finds the container in the DOM to render the component into and renders the calculator component into the container.
// If no element with the id 'root' is found in the DOM, an error message is printed.
//
// Example1 is intended to be used as an example of how to use the GoWebComponents library to create a calculator component.
func Example1() {
	fmt.Println("Example1: Starting to render calculator")

	// Calculator component
	calculator := func(props map[string]interface{}) *Element {
		// Initialize state for the calculator
		input, setInput := useState("")
		result, setResult := useState("")
		previousExpression, setPreviousExpression := useState("")

		useEffect(func() {
			fmt.Println("Result changed:", result())
		}, []interface{}{result()})

		// Function to handle button clicks for numbers and operators
		handleButtonClick := func() js.Func {
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				// Get the value from the button clicked
				value := args[0].Get("target").Get("innerText").String()
				fmt.Println("Button clicked:", value)
				// Append the value to the input
				newInput := input() + value
				setInput(newInput)
				// Clear the result since we're building a new expression
				setResult("")
				return nil
			})
			// Store the callback to keep it alive
			eventCallbacks = append(eventCallbacks, cb)
			return cb
		}

		// Function to handle the equal button click
		handleEqual := func() js.Func {
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				expr := input()
				fmt.Println("Evaluating expression:", expr)
				// Evaluate the expression using JavaScript's eval
				res, err := jsEval(expr)
				if err != nil {
					fmt.Println("Error evaluating expression:", err)
					setResult("Error")
				} else {
					setResult(res)
					// Store the previous expression
					setPreviousExpression(expr + " = " + res)
					// Set the input to the result for the next calculation
					setInput(res)
				}
				return nil
			})
			// Store the callback to keep it alive
			eventCallbacks = append(eventCallbacks, cb)
			return cb
		}

		// Function to handle the clear button click
		handleClear := func() js.Func {
			cb := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				setInput("")
				setResult("")
				setPreviousExpression("")
				return nil
			})
			// Store the callback to keep it alive
			eventCallbacks = append(eventCallbacks, cb)
			return cb
		}

		// Render the calculator UI
		return createElement("div", map[string]interface{}{"class": "container mx-auto p-4 grid grid-cols-12"},
			createElement("h1", map[string]interface{}{"class": "text-2xl font-bold mb-4"}, Text("GoWebComponent Calculator")),
			createElement("div", map[string]interface{}{
				"class": "mb-4 col-start-5 col-end-9",
			},
				// Display the previous expression
				createElement("div", map[string]interface{}{"class": "h-5 text-right text-gray-500 text-sm"}, Text(previousExpression())),
				// Display the input expression
				createElement("div", map[string]interface{}{
					"class": "h-16 text-right text-green-500 text-3xl font-mono bg-gray-800 p-4 rounded",
				}, Text(input())),
			),
			// Calculator buttons
			createElement("div", map[string]interface{}{"class": "col-start-5 col-end-9 grid grid-cols-4 gap-4"},
				// Row 1: Clear (C), Divide (/)
				createElement("button", map[string]interface{}{
					"class":   "col-span-3 bg-red-600 text-white p-4 rounded hover:bg-red-700 transition duration-200",
					"onclick": handleClear(),
				}, Text("C")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("/")),
				// Row 2: 7,8,9,*
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("7")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("8")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("9")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("*")),
				// Row 3: 4,5,6,-
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("4")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("5")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("6")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("-")),
				// Row 4: 1,2,3,+
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("1")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("2")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("3")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-500 text-white p-4 rounded hover:bg-gray-700 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("+")),
				// Row 5: 0, ., =
				createElement("button", map[string]interface{}{
					"class":   "col-span-2 bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text("0")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-400 text-xl p-4 rounded hover:bg-gray-600 transition duration-200",
					"onclick": handleButtonClick(),
				}, Text(".")),
				createElement("button", map[string]interface{}{
					"class":   "bg-blue-600 text-white p-4 rounded hover:bg-blue-700 transition duration-200",
					"onclick": handleEqual(),
				}, Text("=")),
			),
		)
	}

	// Find the container in the DOM to render the component into
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example1: Error - No element with id 'root' found in the DOM")
		return
	}

	// Render the calculator component into the container
	fmt.Println("Example1: Rendering calculator into the container")
	render(createElement(calculator, nil), container)
}

// jsEval evaluates a mathematical expression using JavaScript's eval function.
// Note: In production, using eval can be unsafe; consider using a proper parser.
func jsEval(expr string) (string, error) {
	// Use JavaScript's eval function via the Function constructor to safely evaluate the expression.
	evalFunc := js.Global().Call("Function", "expr", "try { return eval(expr).toString(); } catch (e) { return 'Error'; }")
	res := evalFunc.Invoke(expr)
	resultStr := res.String()
	if resultStr == "Error" {
		return "", fmt.Errorf("error evaluating expression")
	}
	return resultStr, nil
}

// Example2 demonstrates the usage of a simple click counter component. The click counter component keeps track of the number of times a button is clicked. It renders a div container with a heading and a button. The button displays the current count. When the button is clicked, the count is incremented and displayed. The component utilizes the useState and useEffect hooks from the GoWebComponents library. The useState hook is used to manage the count state, while the useEffect hook is used to log a message when the component is mounted. Example2 also demonstrates how to render the component into the DOM using the render function.
func Example2() {
	fmt.Println("Example2: Starting to render ClickCounter with useMemo and useFunc")

	// simple click counter component with memoized calculation
	clickCounter := func(props map[string]interface{}) *Element {
		count, setCount := useState(0)

		handleClick := useFunc(func(this js.Value, args []js.Value) interface{} {
			fmt.Printf("handleClick: Clicked, count is %d\n", count())
			setCount(count() + 1)
			return nil
		})

		// Effect that runs only on mount
		useEffect(func() {
			fmt.Println("useEffect: I should only appear once when the component is mounted")
		}, nil)

		// Effect that runs when count changes
		useEffect(func() {
			fmt.Println("useEffect: Count changed:", count())
		}, count())

		// Effect that runs on every render
		useEffect(func() {
			fmt.Println("useEffect: I run on every render")
		})

		// Memoized expensive calculation
		expensiveResult := useMemo(func() interface{} {
			fmt.Println("Performing expensive calculation...")
			return expensiveCalculation(count())
		}, count())

		return createElement("div", map[string]interface{}{"class": "container mx-auto p-4"},
			createElement("h1", map[string]interface{}{"class": "text-2xl font-bold mb-4"},
				Text("Click Counter with Memoization")),
			createElement("button", map[string]interface{}{
				"onclick": handleClick,
				"class":   "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition duration-200",
			}, Text(fmt.Sprintf("Clicked %d times", count()))),
			createElement("p", map[string]interface{}{"class": "mt-4"},
				Text(fmt.Sprintf("Expensive calculation result: %v", expensiveResult))))
	}

	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example2: Error - No element with id 'root' found in the DOM")
		return
	}
	fmt.Println("Example2: Rendering ClickCounter into the container")
	render(createElement(clickCounter, nil), container)
}

// Simulating an expensive calculation
func expensiveCalculation(count int) int {
	fmt.Println("expensiveCalculation: Started")
	time.Sleep(1000 * time.Millisecond) // Simulate expensive operation
	result := count * 2
	fmt.Println("expensiveCalculation: Finished")
	return result
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

// Example3 demonstrates rendering a BlogListComponent into a container element in the DOM.
func Example3() {
	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example5: Error - No element with id 'root' found in the DOM")
		return
	}
	render(createElement(BlogListComponent, nil), container)
}

func Example5() {
	fmt.Println("Example5: Starting to render Star Wars Character Viewer")

	starWarsComponent := func(props map[string]interface{}) *Element {
		fmt.Println("Rendering starWarsComponent")

		// State for character ID
		getCharId, setCharId := useState(1)

		// Fetch character data
		getCharState := useFetch(fmt.Sprintf("https://swapi.dev/api/people/%d", getCharId()))

		// Event handler for "Next Character" button
		handleNextChar := useFunc(func(this js.Value, args []js.Value) interface{} {
			setCharId(getCharId() + 1)
			return nil
		})

		// Memoized character name
		getCharName := useMemo(func() interface{} {
			charState := getCharState()
			if charState.Data == nil {
				return "Unknown"
			}
			charData, ok := charState.Data.(map[string]interface{})
			if !ok {
				return "Unknown"
			}
			return charData["name"]
		}, []interface{}{getCharState()})

		// Render function
		charState := getCharState() // Get the current state here

		return createElement("div", map[string]interface{}{"class": "container mx-auto p-4"},
			createElement("h1", map[string]interface{}{"class": "text-2xl font-bold mb-4"},
				Text("Star Wars Character Viewer")),
			createElement("div", map[string]interface{}{"class": "mb-4"},
				func() *Element {
					if charState.Loading {
						return createElement("p", nil, Text("Loading..."))
					}
					if charState.Error != "" {
						return createElement("p", map[string]interface{}{"class": "text-red-500"},
							Text(fmt.Sprintf("Error: %s", charState.Error)))
					}
					charData, ok := charState.Data.(map[string]interface{})
					if !ok {
						return createElement("p", nil, Text("Error: Unexpected data format"))
					}
					return createElement("div", nil,
						createElement("h2", map[string]interface{}{"class": "text-xl font-semibold"},
							Text(fmt.Sprintf("Name: %s", charData["name"]))),
						createElement("p", nil, Text(fmt.Sprintf("Height: %s", charData["height"]))),
						createElement("p", nil, Text(fmt.Sprintf("Mass: %s", charData["mass"]))),
						createElement("p", nil, Text(fmt.Sprintf("Hair Color: %s", charData["hair_color"]))),
						createElement("p", nil, Text(fmt.Sprintf("Eye Color: %s", charData["eye_color"]))))
				}()),
			createElement("button",
				map[string]interface{}{
					"onclick": handleNextChar,
					"class":   "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition duration-200",
				},
				Text("Next Character")),
			createElement("p", map[string]interface{}{"class": "mt-4"},
				Text(fmt.Sprintf("Memoized Character Name: %s", getCharName))))
	}

	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example5: Error - No element with id 'root' found in the DOM")
		return
	}
	fmt.Println("Example5: Rendering Star Wars Character Viewer into the container")
	render(createElement(starWarsComponent, nil), container)
}

// Example4 is a benchmark that renders a bouncing div and tracks render count and FPS
func Example4() {
	fmt.Println("Example4: Starting to render BouncingDiv")

	// BallState holds the position and velocity of the ball
	type BallState struct {
		X  float64
		Y  float64
		DX float64
		DY float64
	}

	// BouncingDiv is the component that renders the bouncing ball and FPS/render count
	bouncingDiv := func(props map[string]interface{}) *Element {
		// Initialize states
		getBallState, setBallState := useState(BallState{
			X:  50.0,
			Y:  50.0,
			DX: 5.0,
			DY: 5.0,
		})
		getFPS, setFPS := useState(0)
		getRenderCount, setRenderCount := useState(0)

		// Start the goroutine to update ball position and FPS
		useEffect(func() {
			lastTime := time.Now()
			frameCount := 0

			go func() {
				ticker := time.NewTicker(100 * time.Millisecond) // Approximately 60 FPS
				defer ticker.Stop()

				for range ticker.C {
					// Get current ball state
					state := getBallState()

					// Update position
					state.X += state.DX
					state.Y += state.DY

					// Check boundaries and reverse direction if necessary
					if state.X <= 0 || state.X >= 380 {
						state.DX = -state.DX
						// Clamp position to boundaries
						if state.X <= 0 {
							state.X = 0
						} else {
							state.X = 380
						}
					}
					if state.Y <= 0 || state.Y >= 280 {
						state.DY = -state.DY
						// Clamp position to boundaries
						if state.Y <= 0 {
							state.Y = 0
						} else {
							state.Y = 280
						}
					}

					// Update the ball state
					setBallState(state)

					// Increment render count
					setRenderCount(getRenderCount() + 1)

					// Increment frame count
					frameCount++

					fmt.Printf("FPS: %d\n",  getFPS())

					// Calculate FPS every second
					now := time.Now()
					if now.Sub(lastTime) >= time.Second {
						setFPS(frameCount)
						frameCount = 0
						lastTime = now
					}
				}
			}()
		}) // No dependencies; runs once on mount

		// Retrieve current states
		ballState := getBallState()
		fps := getFPS()
		renderCount := getRenderCount()

		// Create the bouncing ball element
		ball := createElement("div", map[string]interface{}{
			"class": "absolute w-5 h-5 bg-blue-500 rounded-full",
			"style": fmt.Sprintf("transform: translate(%.2fpx, %.2fpx);", ballState.X, ballState.Y),
		})

		// Create the FPS display element
		fpsDisplay := createElement("div", map[string]interface{}{
			"class": "absolute top-2 left-2 text-xs text-gray-500",
		},
			Text(fmt.Sprintf("FPS: %d", fps)),
		)

		// Create the Render Count display element
		renderCountDisplay := createElement("div", map[string]interface{}{
			"class": "absolute bottom-2 right-2 text-xs text-gray-500",
		},
			Text(fmt.Sprintf("Render count: %d", renderCount)),
		)

		// Create the outer container with the ball and displays as children
		return createElement("div", map[string]interface{}{
			"class": "relative w-96 h-80 bg-gray-200 overflow-hidden",
		},
			ball,
			fpsDisplay,
			renderCountDisplay,
		)
	}

	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example4: Error - No element with id 'root' found in the DOM")
		return
	}
	fmt.Println("Example4: Rendering BouncingDiv into the container")
	render(createElement(bouncingDiv, nil), container)
}
