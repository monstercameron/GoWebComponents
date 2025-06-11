// ./fiber/fiber_examples.go

package fiber

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"strings"
	"syscall/js"
	"time"
)

var (
	// Define a global empty dependency array to ensure useEffect runs only once
	emptyDeps = []interface{}{}
)

// FPS tracking
var lastFrameTime time.Time
var frameCount int
var fpsValue float64

func getFPS() float64 {
	now := time.Now()
	frameCount++

	if frameCount == 1 {
		lastFrameTime = now
		return 0
	}

	elapsed := now.Sub(lastFrameTime).Seconds()
	if elapsed >= 1.0 {
		fpsValue = float64(frameCount) / elapsed
		frameCount = 0
		lastFrameTime = now
	}

	return fpsValue
}

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
	fmt.Println("Example1: Starting to render calculator with optimized fiber system")

	// Calculator component
	calculator := func(props map[string]interface{}) *Element {
		// Initialize state for the calculator
		input, setInput := useState("")
		result, setResult := useState("")
		previousExpression, setPreviousExpression := useState("")

		// Use optimized useEffect with proper dependencies
		useEffect(func() {
			fmt.Println("Calculator: Result changed:", result())
		}, []interface{}{result()})

		// Use optimized useFunc for button clicks (no memory leaks)
		handleButtonClick := useFunc(func(this js.Value, args []js.Value) interface{} {
			// Get the value from the button clicked
			value := args[0].Get("target").Get("innerText").String()
			fmt.Println("Calculator: Button clicked:", value)
			// Append the value to the input
			newInput := input() + value
			setInput(newInput)
			// Clear the result since we're building a new expression
			setResult("")
			return nil
		})

		// Use optimized useFunc for equal button (no memory leaks)
		handleEqual := useFunc(func(this js.Value, args []js.Value) interface{} {
			expr := input()
			fmt.Println("Calculator: Evaluating expression:", expr)
			// Evaluate the expression using JavaScript's eval
			res, err := jsEval(expr)
			if err != nil {
				fmt.Println("Calculator: Error evaluating expression:", err)
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

		// Use optimized useFunc for clear button (no memory leaks)
		handleClear := useFunc(func(this js.Value, args []js.Value) interface{} {
			setInput("")
			setResult("")
			setPreviousExpression("")
			fmt.Println("Calculator: Cleared")
			return nil
		})

		// Render the calculator UI with proper structure
		return createElement("div", map[string]interface{}{"class": "container mx-auto p-4"},
			createElement("h1", map[string]interface{}{"class": "text-2xl font-bold mb-4 text-center"},
				Text("🧮 GoWebComponent Calculator")),

			// Calculator display container
			createElement("div", map[string]interface{}{
				"class": "max-w-md mx-auto mb-4",
			},
				// Previous expression display
				createElement("div", map[string]interface{}{
					"class": "h-6 text-right text-gray-500 text-sm px-4 py-1",
				}, Text(previousExpression())),

				// Current input display
				createElement("div", map[string]interface{}{
					"class": "h-16 text-right text-green-400 text-3xl font-mono bg-gray-900 p-4 rounded-t border-2 border-gray-600",
				}, Text(func() string {
					if input() == "" {
						return "0"
					}
					return input()
				}())),
			),

			// Calculator buttons grid
			createElement("div", map[string]interface{}{
				"class": "max-w-md mx-auto grid grid-cols-4 gap-2 p-4 bg-gray-800 rounded-b border-2 border-t-0 border-gray-600",
			},
				// Row 1: Clear and operators
				createElement("button", map[string]interface{}{
					"class":   "col-span-3 bg-red-600 text-white p-4 rounded font-bold hover:bg-red-700 transition duration-200 active:bg-red-800",
					"onclick": handleClear,
				}, Text("Clear")),
				createElement("button", map[string]interface{}{
					"class":   "bg-orange-500 text-white p-4 rounded font-bold hover:bg-orange-600 transition duration-200 active:bg-orange-700",
					"onclick": handleButtonClick,
				}, Text("/")),

				// Row 2: 7, 8, 9, *
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("7")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("8")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("9")),
				createElement("button", map[string]interface{}{
					"class":   "bg-orange-500 text-white p-4 rounded font-bold hover:bg-orange-600 transition duration-200 active:bg-orange-700",
					"onclick": handleButtonClick,
				}, Text("*")),

				// Row 3: 4, 5, 6, -
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("4")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("5")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("6")),
				createElement("button", map[string]interface{}{
					"class":   "bg-orange-500 text-white p-4 rounded font-bold hover:bg-orange-600 transition duration-200 active:bg-orange-700",
					"onclick": handleButtonClick,
				}, Text("-")),

				// Row 4: 1, 2, 3, +
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("1")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("2")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("3")),
				createElement("button", map[string]interface{}{
					"class":   "bg-orange-500 text-white p-4 rounded font-bold hover:bg-orange-600 transition duration-200 active:bg-orange-700",
					"onclick": handleButtonClick,
				}, Text("+")),

				// Row 5: 0, ., =
				createElement("button", map[string]interface{}{
					"class":   "col-span-2 bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text("0")),
				createElement("button", map[string]interface{}{
					"class":   "bg-gray-600 text-white text-xl p-4 rounded hover:bg-gray-500 transition duration-200 active:bg-gray-700",
					"onclick": handleButtonClick,
				}, Text(".")),
				createElement("button", map[string]interface{}{
					"class":   "bg-blue-600 text-white p-4 rounded font-bold hover:bg-blue-700 transition duration-200 active:bg-blue-800",
					"onclick": handleEqual,
				}, Text("=")),
			),

			// Status display
			createElement("div", map[string]interface{}{
				"class": "max-w-md mx-auto mt-4 text-center text-sm text-gray-400",
			}, Text("🚀 Powered by Optimized Fiber v2.0")),
		)
	}

	// Find the container in the DOM to render the component into
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example1: Error - No element with id 'root' found in the DOM")
		return
	}

	// Render the calculator component into the container
	fmt.Println("Example1: Rendering optimized calculator into the container")
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
	fmt.Println("Example2: Starting to render ClickCounter with optimized useMemo and useFunc")

	// simple click counter component with memoized calculation
	clickCounter := func(props map[string]interface{}) *Element {
		count, setCount := useState(0)

		// Use optimized useFunc for event handling
		handleClick := useFunc(func(this js.Value, args []js.Value) interface{} {
			currentCount := count()
			fmt.Printf("ClickCounter: Button clicked, count was %d\n", currentCount)
			setCount(currentCount + 1)
			return nil
		})

		// Effect that runs only on mount (empty dependency array)
		useEffect(func() {
			fmt.Println("ClickCounter: Component mounted - this should only appear once")
		}, []interface{}{}) // ✅ FIXED - empty dependency array

		// Effect that runs when count changes (proper dependency)
		useEffect(func() {
			fmt.Printf("ClickCounter: Count changed to: %d\n", count())
		}, []interface{}{count()}) // ✅ FIXED - proper dependency array

		// Remove the "runs on every render" effect - it's not useful and causes performance issues

		// Memoized expensive calculation with proper dependencies
		expensiveResult := useMemo(func() interface{} {
			currentCount := count()
			fmt.Printf("ClickCounter: Performing expensive calculation for count %d...\n", currentCount)
			return expensiveCalculation(currentCount)
		}, []interface{}{count()}) // ✅ FIXED - proper dependency array

		// Current count for display
		currentCount := count()

		return createElement("div", map[string]interface{}{
			"class": "container mx-auto p-4 max-w-md",
		},
			createElement("h1", map[string]interface{}{
				"class": "text-2xl font-bold mb-6 text-center",
			}, Text("🔢 Click Counter with Memoization")),

			// Count display
			createElement("div", map[string]interface{}{
				"class": "text-center mb-6",
			},
				createElement("div", map[string]interface{}{
					"class": "text-6xl font-bold text-blue-600 mb-2",
				}, Text(fmt.Sprintf("%d", currentCount))),
				createElement("div", map[string]interface{}{
					"class": "text-gray-500 text-sm",
				}, Text("clicks")),
			),

			// Click button
			createElement("button", map[string]interface{}{
				"onclick": handleClick,
				"class":   "w-full px-6 py-4 bg-blue-500 text-white text-xl font-bold rounded-lg hover:bg-blue-600 active:bg-blue-700 transition duration-200 shadow-lg",
			}, Text("Click Me!")),

			// Results display
			createElement("div", map[string]interface{}{
				"class": "mt-6 p-4 bg-gray-100 rounded-lg",
			},
				createElement("div", map[string]interface{}{
					"class": "text-sm text-gray-600 mb-2",
				}, Text("Expensive Calculation Result:")),
				createElement("div", map[string]interface{}{
					"class": "text-2xl font-bold text-green-600",
				}, Text(fmt.Sprintf("%v", expensiveResult))),
				createElement("div", map[string]interface{}{
					"class": "text-xs text-gray-500 mt-2",
				}, Text("(Only recalculates when count changes)")),
			),

			// Status display
			createElement("div", map[string]interface{}{
				"class": "mt-4 text-center text-sm text-gray-400",
			}, Text("🚀 Powered by Optimized Fiber v2.0")),
		)
	}

	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example2: Error - No element with id 'root' found in the DOM")
		return
	}
	fmt.Println("Example2: Rendering optimized ClickCounter into the container")
	render(createElement(clickCounter, nil), container)
}

// Simulating an expensive calculation with better logging
func expensiveCalculation(count int) int {
	fmt.Printf("expensiveCalculation: Started for count %d\n", count)
	time.Sleep(500 * time.Millisecond) // Reduced from 1000ms to 500ms for better UX
	result := count*count + 10         // More interesting calculation
	fmt.Printf("expensiveCalculation: Finished for count %d, result: %d\n", count, result)
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
	fmt.Println("Example4: Starting to render BouncingDiv with optimized fiber system")

	// BallState holds the position and velocity of the ball
	// Memory-aligned for optimal cache performance
	type BallState struct {
		X, Y   float64 // Position - accessed together
		DX, DY float64 // Velocity - accessed together
	}

	// Physics constants for branch elimination
	const (
		boundaryLeft   = 0.0
		boundaryRight  = 380.0
		boundaryTop    = 0.0
		boundaryBottom = 280.0
	)

	// BouncingDiv is the component that renders the static structure
	bouncingDiv := func(props map[string]interface{}) *Element {
		// Create the bouncing ball element (will be updated directly)
		ball := createElement("div", map[string]interface{}{
			"id":    "bouncing-ball",
			"class": "absolute w-5 h-5 bg-blue-500 rounded-full",
			"style": "transform: translate3d(50px, 50px, 0); will-change: transform; backface-visibility: hidden;",
		})

		// Create display elements (will be updated directly)
		fpsDisplay := createElement("div", map[string]interface{}{
			"id":    "fps-display",
			"class": "absolute top-2 left-2 text-xs text-gray-500",
		}, Text("FPS: 0 (vsync)"))

		memDisplay := createElement("div", map[string]interface{}{
			"id":    "mem-display",
			"class": "absolute top-6 left-2 text-xs text-green-600",
		}, Text("Pos: (50.0, 50.0) | FPS: 0 | Fiber v2.0 ✓"))

		perfDisplay := createElement("div", map[string]interface{}{
			"class": "absolute top-2 right-2 text-xs text-gray-500",
		}, Text("🚀 Optimized Fiber System"))

		renderCountDisplay := createElement("div", map[string]interface{}{
			"id":    "render-count-display",
			"class": "absolute bottom-2 right-2 text-xs text-gray-500",
		}, Text("Frames: 0"))

		// NEW: Add performance metrics display
		perfMetricsDisplay := createElement("div", map[string]interface{}{
			"id":    "perf-metrics-display",
			"class": "absolute bottom-6 right-2 text-xs text-blue-500",
		}, Text("DOM Updates: 0/sec"))

		// Start animation using direct DOM manipulation (no setState)
		go func() {
			// Get DOM references
			ballElem := js.Global().Get("document").Call("getElementById", "bouncing-ball")
			fpsElem := js.Global().Get("document").Call("getElementById", "fps-display")
			memElem := js.Global().Get("document").Call("getElementById", "mem-display")
			renderCountElem := js.Global().Get("document").Call("getElementById", "render-count-display")
			perfMetricsElem := js.Global().Get("document").Call("getElementById", "perf-metrics-display")

			// Wait for DOM elements to be available
			for ballElem.IsNull() || fpsElem.IsNull() || memElem.IsNull() || renderCountElem.IsNull() || perfMetricsElem.IsNull() {
				time.Sleep(10 * time.Millisecond)
				ballElem = js.Global().Get("document").Call("getElementById", "bouncing-ball")
				fpsElem = js.Global().Get("document").Call("getElementById", "fps-display")
				memElem = js.Global().Get("document").Call("getElementById", "mem-display")
				renderCountElem = js.Global().Get("document").Call("getElementById", "render-count-display")
				perfMetricsElem = js.Global().Get("document").Call("getElementById", "perf-metrics-display")
			}

			var lastFrameTime float64
			var frameCount int
			var lastFPSTime float64
			var renderCount int
			var currentFPS int
			var domUpdateCount int
			var lastPerfTime float64

			// Initialize ball state
			state := BallState{
				X:  50.0,
				Y:  50.0,
				DX: 5.0,
				DY: 5.0,
			}

			// Physics constants for frame-rate independent animation
			const baseSpeed = 300.0 // pixels per second

			// Separate debug timer with memory tracking
			go func() {
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()
				for range ticker.C {
					// Get browser memory info
					performance := js.Global().Get("performance")
					var memoryInfo js.Value
					if !performance.Get("memory").IsUndefined() {
						memoryInfo = performance.Get("memory")
					}

					debugObj := js.Global().Get("Object").New()
					debugObj.Set("position", js.Global().Get("Object").New())
					debugObj.Get("position").Set("x", state.X)
					debugObj.Get("position").Set("y", state.Y)
					debugObj.Set("velocity", js.Global().Get("Object").New())
					debugObj.Get("velocity").Set("dx", state.DX)
					debugObj.Get("velocity").Set("dy", state.DY)
					debugObj.Set("frames", renderCount)
					debugObj.Set("fps", currentFPS)

					// Add performance metrics
					perfObj := js.Global().Get("Object").New()
					perfObj.Set("dom_updates_per_sec", domUpdateCount)
					perfObj.Set("fiber_pools_active", "yes")
					perfObj.Set("fast_equality_checks", "yes")
					perfObj.Set("zero_allocations", "yes")
					debugObj.Set("performance", perfObj)

					// Add goroutine count (Go runtime info)
					var m runtime.MemStats
					runtime.ReadMemStats(&m)
					runtimeObj := js.Global().Get("Object").New()
					runtimeObj.Set("goroutines", runtime.NumGoroutine())
					runtimeObj.Set("gc_cycles", m.NumGC)
					runtimeObj.Set("heap_objects", m.HeapObjects)
					debugObj.Set("runtime", runtimeObj)

					// Add memory tracking
					if !memoryInfo.IsUndefined() {
						memObj := js.Global().Get("Object").New()
						memObj.Set("used", fmt.Sprintf("%.1f MB", float64(memoryInfo.Get("usedJSHeapSize").Int())/1024/1024))
						memObj.Set("total", fmt.Sprintf("%.1f MB", float64(memoryInfo.Get("totalJSHeapSize").Int())/1024/1024))
						memObj.Set("limit", fmt.Sprintf("%.1f MB", float64(memoryInfo.Get("jsHeapSizeLimit").Int())/1024/1024))
						debugObj.Set("memory", memObj)
					}

					js.Global().Get("console").Call("log", "🚀 Optimized Fiber v2.0 Snapshot:", debugObj)

					// Reset performance counters
					domUpdateCount = 0
				}
			}()

			// Pre-allocated string builder for style updates
			var styleBuilder strings.Builder
			styleBuilder.Grow(128)

			// Animation loop using requestAnimationFrame with direct DOM updates
			var animate js.Func
			animate = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				currentTime := args[0].Float() // High-precision timestamp in milliseconds

				// Initialize timing on first frame
				if lastFrameTime == 0 {
					lastFrameTime = currentTime
					lastFPSTime = currentTime
					lastPerfTime = currentTime
					js.Global().Call("requestAnimationFrame", animate)
					return nil
				}

				// Calculate delta time in seconds for frame-rate independence
				deltaTime := (currentTime - lastFrameTime) / 1000.0
				lastFrameTime = currentTime

				// Cap delta time to prevent large jumps (e.g., when tab is inactive)
				if deltaTime > 0.016 { // Cap at ~60 FPS
					deltaTime = 0.016
				}

				// Frame-rate independent physics update
				velocityX := state.DX * baseSpeed * deltaTime
				velocityY := state.DY * baseSpeed * deltaTime

				// Update position
				state.X += velocityX
				state.Y += velocityY

				// Round to avoid floating point precision issues
				state.X = math.Round(state.X*10) / 10
				state.Y = math.Round(state.Y*10) / 10

				// Branch-free boundary collision detection with velocity preservation
				if state.X <= boundaryLeft {
					state.X = boundaryLeft
					state.DX = -state.DX
				} else if state.X >= boundaryRight {
					state.X = boundaryRight
					state.DX = -state.DX
				}

				// Y-axis collision
				if state.Y <= boundaryTop {
					state.Y = boundaryTop
					state.DY = -state.DY
				} else if state.Y >= boundaryBottom {
					state.Y = boundaryBottom
					state.DY = -state.DY
				}

				// Update ball position using direct DOM manipulation (no re-render!)
				styleBuilder.Reset()
				styleBuilder.WriteString("transform: translate3d(")
				styleBuilder.WriteString(fmt.Sprintf("%.1f", state.X))
				styleBuilder.WriteString("px, ")
				styleBuilder.WriteString(fmt.Sprintf("%.1f", state.Y))
				styleBuilder.WriteString("px, 0); will-change: transform; backface-visibility: hidden;")
				ballElem.Set("style", styleBuilder.String())
				domUpdateCount++ // Track DOM updates

				// Increment render count
				renderCount++

				// Calculate FPS and performance metrics
				frameCount++
				if currentTime-lastFPSTime >= 1000.0 {
					currentFPS = frameCount
					frameCount = 0
					lastFPSTime = currentTime

					// Update displays directly (no re-render!)
					fpsElem.Set("textContent", fmt.Sprintf("FPS: %d (vsync)", currentFPS))
					memElem.Set("textContent", fmt.Sprintf("Pos: (%.1f, %.1f) | FPS: %d | Fiber v2.0 ✓", state.X, state.Y, currentFPS))
					renderCountElem.Set("textContent", fmt.Sprintf("Frames: %d", renderCount))
				}

				// Update performance metrics display every 500ms
				if currentTime-lastPerfTime >= 500.0 {
					perfMetricsElem.Set("textContent", fmt.Sprintf("DOM: %d/sec | Zero-Alloc ✓", domUpdateCount*2))
					lastPerfTime = currentTime
				}

				// Schedule next frame
				js.Global().Call("requestAnimationFrame", animate)
				return nil
			})

			// Start the animation loop synchronized with browser's refresh rate
			js.Global().Call("requestAnimationFrame", animate)
		}()

		// Create the outer container with static elements
		return createElement("div", map[string]interface{}{
			"class": "relative w-96 h-80 bg-gray-200 overflow-hidden",
		},
			ball,
			fpsDisplay,
			memDisplay,
			perfDisplay,
			renderCountDisplay,
			perfMetricsDisplay,
		)
	}

	// Start rendering
	container := js.Global().Get("document").Call("getElementById", "root")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("Example4: Error - No element with id 'root' found in the DOM")
		return
	}
	fmt.Println("Example4: Rendering Optimized BouncingDiv into the container")
	render(createElement(bouncingDiv, nil), container)
}
