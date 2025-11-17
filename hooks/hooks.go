//go:build js && wasm
// +build js,wasm

package hooks

import (
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Type alias for Element from runtime
type Element = runtime.Element

// UseState manages state in a component with optimized equality checking.
// This is the primary hook for adding reactive state to functional components.
//
// The hook returns two functions:
//   - A getter function that returns the current state value
//   - A setter function that updates the state and triggers a re-render
//
// The setter accepts either a direct value or a function that takes the previous
// state and returns the new state. The functional form is useful when the new
// state depends on the previous state, especially during rapid updates.
//
// Type parameter T can be any Go type. The hook uses generic type parameters
// for type safety and better developer experience.
//
// Example with direct value:
//
//	func Counter(props dom.Attrs) *fiber.Element {
//	    count, setCount := hooks.UseState(0)
//
//	    increment := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        setCount(count() + 1)  // Direct value
//	        return nil
//	    })
//
//	    return dom.Div(nil,
//	        dom.P(nil, fmt.Sprintf("Count: %d", count())),
//	        dom.Button(map[string]interface{}{"onclick": increment}, "Increment"),
//	    )
//	}
//
// Example with functional update (recommended for rapid updates):
//
//	increment := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	    setCount(func(prev int) int { return prev + 1 })  // Functional update
//	    return nil
//	})
//
// Important: Always call UseState at the top level of your component,
// never inside conditions or loops. The order of hook calls must be
// consistent across renders.
func UseState[T any](initialValue T) (func() T, func(interface{})) {
	return runtime.GoUseStateGlobal(initialValue)
}

// UseEffect runs side effects in a component with dependency tracking.
// Effects run after the component renders and the DOM has been updated.
//
// The effect function is called:
//   - On the first render (mount)
//   - Whenever any dependency value changes
//   - On every render if no dependencies are provided
//
// Dependencies should be values that the effect uses. When any dependency
// changes between renders, the effect will run again.
//
// Example with dependencies:
//
//	func DataFetcher(props dom.Attrs) *fiber.Element {
//	    url := props["url"].(string)
//	    data, setData := hooks.UseState("")
//
//	    hooks.UseEffect(func() {
//	        // Fetch data from URL
//	        result := fetchData(url)
//	        setData(result)
//	    }, url) // Re-run effect when URL changes
//
//	    return dom.Div(nil, dom.P(nil, data()))
//	}
//
// Example without dependencies (runs every render):
//
//	hooks.UseEffect(func() {
//	    fmt.Println("Component rendered")
//	})
//
// Example with empty dependencies (runs once on mount):
//
//	hooks.UseEffect(func() {
//	    fmt.Println("Component mounted")
//	}, []interface{}{})
func UseEffect(effect func() func(), deps ...interface{}) {
	runtime.GoUseEffectGlobal(effect, deps...)
}

// UseMemo memoizes expensive computations with dependency tracking.
// The memoized value is only recomputed when dependencies change.
//
// This hook is useful for:
//   - Expensive calculations that don't need to run on every render
//   - Creating stable object references to prevent unnecessary re-renders
//   - Optimizing performance of child components
//
// The compute function is called:
//   - On the first render
//   - Whenever any dependency changes
//
// Example:
//
//	func ExpensiveList(props dom.Attrs) *fiber.Element {
//	    items := props["items"].([]string)
//	    filter := props["filter"].(string)
//
//	    // Only filter items when items or filter changes
//	    filteredItems := hooks.UseMemo(func() interface{} {
//	        result := []string{}
//	        for _, item := range items {
//	            if strings.Contains(item, filter) {
//	                result = append(result, item)
//	            }
//	        }
//	        return result
//	    }, items, filter).([]string)
//
//	    // Render filtered items...
//	    return dom.Ul(nil, /* render items */)
//	}
//
// Note: The returned value is interface{}, so you'll need to type assert
// to the expected type.
func UseMemo(compute func() interface{}, deps ...interface{}) interface{} {
	return runtime.GoUseMemoGlobal(compute, deps...)
}

// UseCallback memoizes a callback function with dependency tracking.
// The memoized function is only updated when dependencies change.
//
// This hook is useful for:
//   - Preventing unnecessary re-renders of child components
//   - Stable function references for event handlers
//   - Optimizing performance when passing callbacks as props
//
// The callback function is returned as-is on first render,
// and the same function reference is returned on subsequent renders
// if dependencies haven't changed.
//
// Example with event handler:
//
//	func Button(props dom.Attrs) *fiber.Element {
//	    count, setCount := hooks.UseState(0)
//
//	    // Callback only changes when count changes
//	    handleClick := hooks.UseCallback(func(this js.Value, args []js.Value) interface{} {
//	        setCount(func(prev int) int { return prev + 1 })
//	        return nil
//	    }, count())
//
//	    return dom.Button(map[string]interface{}{"onclick": handleClick},
//	        fmt.Sprintf("Count: %d", count()))
//	}
//
// Example passing callback to child:
//
//	parent := func(props dom.Attrs) *fiber.Element {
//	    items, setItems := hooks.UseState([]string{})
//
//	    // Callback only changes when setItems changes (never)
//	    handleAdd := hooks.UseCallback(func(item string) {
//	        setItems(func(prev []string) []string {
//	            return append(prev, item)
//	        })
//	    })
//
//	    return Child(map[string]interface{}{"onAdd": handleAdd})
//	}
//
// Note: The returned value is interface{}, so you'll need to type assert
// to the expected function type (js.Func, func(...), etc.)
func UseCallback(fn interface{}, deps ...interface{}) interface{} {
	return runtime.GoUseCallbackGlobal(fn, deps...)
}

// UseRef creates a mutable reference that persists across renders.
// Unlike state, updating a ref does NOT trigger a re-render.
// This hook is useful for:
//   - Storing mutable values that don't affect rendering (like timers, intervals)
//   - Direct DOM manipulation or element access
//   - Keeping track of previous values
//   - Storing imperative API handles
//
// The ref is returned as a RefValue object with a .Current field that can
// hold any value. You modify the value by changing .Current, and it will
// persist across re-renders.
//
// Example storing interval ID:
//
//	func Timer(props dom.Attrs) *fiber.Element {
//	    count, setCount := hooks.UseState(0)
//	    intervalRef := hooks.UseRef(nil)
//
//	    handleStart := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        intervalRef.Current = js.Global().Call("setInterval", func() {
//	            setCount(func(prev int) int { return prev + 1 })
//	        }, 1000)
//	        return nil
//	    })
//
//	    handleStop := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        if intervalRef.Current != nil {
//	            js.Global().Call("clearInterval", intervalRef.Current)
//	        }
//	        return nil
//	    })
//
//	    return dom.Div(nil,
//	        dom.P(nil, fmt.Sprintf("Count: %d", count())),
//	        dom.Button(map[string]interface{}{"onclick": handleStart}, "Start"),
//	        dom.Button(map[string]interface{}{"onclick": handleStop}, "Stop"),
//	    )
//	}
//
// Example storing previous value:
//
//	func PreviousValue(props dom.Attrs) *fiber.Element {
//	    count, setCount := hooks.UseState(0)
//	    prevCountRef := hooks.UseRef(0)
//
//	    // Update ref after render
//	    hooks.UseEffect(func() func() {
//	        prevCountRef.Current = count()
//	        return nil
//	    }, count())
//
//	    return dom.Div(nil,
//	        dom.P(nil, fmt.Sprintf("Current: %d", count())),
//	        dom.P(nil, fmt.Sprintf("Previous: %d", prevCountRef.Current)),
//	    )
//	}
//
// Note: The RefValue has a .Current field of type interface{}, so you'll need
// to type assert when reading values from the ref.
func UseRef(initialValue interface{}) *runtime.RefValue {
	return runtime.GoUseRefGlobal(initialValue)
}

// UseId generates a unique, stable identifier for accessibility attributes.
// The ID is generated once per component and persists across renders,
// making it ideal for connecting labels to form inputs and other elements.
//
// This hook is useful for:
//   - Connecting label elements to form inputs
//   - Creating unique IDs for ARIA attributes
//   - Generating predictable but unique identifiers
//   - Accessibility improvements (a11y)
//
// The generated ID has format: "gwc:<unique-number>:<hook-position>"
// This format ensures uniqueness globally and stability across renders.
//
// Example connecting label to input:
//
//	func TextInput(props dom.Attrs) *fiber.Element {
//	    inputId := hooks.UseId()
//	    value, setValue := hooks.UseState("")
//
//	    return dom.Div(nil,
//	        dom.Label(map[string]interface{}{
//	            "htmlFor": inputId,
//	        }, dom.Text("Name:")),
//	        dom.Input(map[string]interface{}{
//	            "id": inputId,
//	            "type": "text",
//	            "value": value(),
//	            "onchange": func(this js.Value, args []js.Value) interface{} {
//	                setValue(args[0].Get("target").Get("value").String())
//	                return nil
//	            },
//	        }),
//	    )
//	}
//
// Example with ARIA attributes:
//
//	func ComboBox(props dom.Attrs) *fiber.Element {
//	    labelId := hooks.UseId()
//	    listId := hooks.UseId()
//	    open, setOpen := hooks.UseState(false)
//
//	    return dom.Div(nil,
//	        dom.Label(map[string]interface{}{
//	            "id": labelId,
//	        }, dom.Text("Select option:")),
//	        dom.Div(map[string]interface{}{
//	            "id": listId,
//	            "role": "listbox",
//	            "aria-labelledby": labelId,
//	        },
//	            dom.Div(nil, dom.Text("Option 1")),
//	            dom.Div(nil, dom.Text("Option 2")),
//	        ),
//	    )
//	}
//
// Note: The returned ID is a string and remains stable across re-renders.
// Each call to UseId in a component generates a different ID.
func UseId() string {
	return runtime.GoUseIdGlobal()
}

// UseFetch manages asynchronous data fetching with manual trigger control.
// This hook is designed for scenarios where you want fine-grained control
// over when data is fetched, rather than automatic fetching on mount or dependency change.
//
// The hook returns:
//   - A getter function that returns the current FetchState (Data, Error, Loading)
//   - A refetch function to manually trigger the fetch operation
//
// The FetchState contains:
//   - Data: The fetched data (nil until fetch completes)
//   - Error: Error message if fetch failed (empty string if no error)
//   - Loading: Boolean indicating if fetch is in progress
//
// The fetch operation runs asynchronously in a goroutine using channels,
// allowing the component to remain responsive during data loading.
//
// Example basic usage:
//
//	func UserProfile(props dom.Attrs) *fiber.Element {
//	    userState, refetchUser := hooks.UseFetch("https://api.example.com/user/123")
//
//	    // Handle loading state
//	    if userState().Loading {
//	        return dom.Div(nil, dom.Text("Loading user..."))
//	    }
//
//	    // Handle error state
//	    if userState().Error != "" {
//	        return dom.Div(nil, dom.Text("Error: " + userState().Error))
//	    }
//
//	    // Render data
//	    return dom.Div(nil,
//	        dom.P(nil, dom.Text("User data loaded")),
//	        dom.Button(map[string]interface{}{
//	            "onclick": func(this js.Value, args []js.Value) interface{} {
//	                refetchUser()
//	                return nil
//	            },
//	        }, "Refresh"),
//	    )
//	}
//
// Example with dependent data loading:
//
//	func PostWithComments(props dom.Attrs) *fiber.Element {
//	    postId, _ := hooks.UseState("123")
//	    postState, refetchPost := hooks.UseFetch("https://api.example.com/posts/" + postId())
//	    
//	    loadPost := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	        refetchPost()
//	        return nil
//	    })
//
//	    return dom.Div(nil,
//	        dom.Button(map[string]interface{}{"onclick": loadPost}, "Load Post"),
//	        renderPostContent(postState()),
//	    )
//	}
//
// Example with polling/refresh logic:
//
//	func RealtimeData(props dom.Attrs) *fiber.Element {
//	    dataState, refetch := hooks.UseFetch("https://api.example.com/stats")
//	    isPolling, setIsPolling := hooks.UseState(false)
//
//	    // Set up polling interval
//	    hooks.UseEffect(func() func() {
//	        if !isPolling() {
//	            return nil
//	        }
//	        
//	        ticker := time.NewTicker(5 * time.Second)
//	        go func() {
//	            for range ticker.C {
//	                refetch()
//	            }
//	        }()
//	        
//	        return func() {
//	            ticker.Stop()
//	        }
//	    }, isPolling())
//
//	    return dom.Div(nil,
//	        dom.Button(map[string]interface{}{
//	            "onclick": func(this js.Value, args []js.Value) interface{} {
//	                setIsPolling(func(v interface{}) interface{} {
//	                    return !v.(bool)
//	                })
//	                return nil
//	            },
//	        }, "Toggle Polling"),
//	        renderData(dataState()),
//	    )
//	}
//
// Important: Always call UseFetch at the top level of your component,
// never inside conditions or loops. Call refetch() to trigger the fetch operation.
func UseFetch(url string, options ...interface{}) (func() runtime.FetchState, func()) {
	return runtime.GoUseFetchGlobal(url, options...)
}

// Note: GoUseFunc is not yet implemented in the fiber package
// func UseFunc(...) { ... }

// GoUseFunc wraps Go functions as JavaScript event handlers
// It automatically detects the function signature and creates the appropriate wrapper
//
// This hook makes event handlers much cleaner by eliminating verbose js.FuncOf boilerplate.
//
// Supported function signatures:
//   - func() - Simple handler with no arguments
//   - func(string) - Input handler that extracts event.target.value
//   - func(js.Value) - Full event handler receiving the raw JS event object
//   - func() error - Handler that can return errors gracefully
//   - func(js.Value) error - Event handler that can fail
//
// The returned value can be used directly as an event handler in dom attributes.
//
// Example with simple click handler:
//
//	func Counter(props dom.Attrs) *dom.Element {
//	    count, setCount := hooks.UseState(0)
//
//	    increment := hooks.GoUseFunc(func() {
//	        setCount(func(prev int) int { return prev + 1 })
//	    })
//
//	    return dom.Div(nil,
//	        dom.P(nil, dom.Text(fmt.Sprintf("Count: %d", count()))),
//	        dom.Button(dom.Attrs{"onclick": increment}, dom.Text("Increment")),
//	    )
//	}
//
// Example with input handler that extracts value:
//
//	func TextInput(props dom.Attrs) *dom.Element {
//	    value, setValue := hooks.UseState("")
//
//	    handleChange := hooks.GoUseFunc(func(inputValue string) {
//	        setValue(inputValue)
//	    })
//
//	    return dom.Div(nil,
//	        dom.Input(dom.Attrs{
//	            "type": "text",
//	            "onchange": handleChange,
//	        }),
//	        dom.P(nil, dom.Text(fmt.Sprintf("Value: %s", value()))),
//	    )
//	}
//
// Example with full event handler:
//
//	func FormSubmit(props dom.Attrs) *dom.Element {
//	    submitted, setSubmitted := hooks.UseState("")
//
//	    handleSubmit := hooks.GoUseFunc(func(event js.Value) {
//	        event.Call("preventDefault")
//	        input := event.Get("target").Call("querySelector", "#form-input")
//	        value := input.Get("value").String()
//	        setSubmitted(value)
//	    })
//
//	    return dom.Form(dom.Attrs{"onsubmit": handleSubmit},
//	        dom.Input(dom.Attrs{"id": "form-input", "type": "text"}),
//	        dom.Button(dom.Attrs{"type": "submit"}, dom.Text("Submit")),
//	        dom.P(nil, dom.Text(fmt.Sprintf("Submitted: %s", submitted()))),
//	    )
//	}
//
// Benefits over direct js.FuncOf:
//   - ✓ Much cleaner and more readable code
//   - ✓ Automatic argument extraction (no manual unpacking needed)
//   - ✓ Type-safe - compiler catches signature mismatches
//   - ✓ Less boilerplate - no need for nil returns and type assertions
//   - ✓ Familiar pattern - follows React hooks conventions
//
// Important: Always call GoUseFunc at the top level of your component,
// never inside conditions or loops. The order of hook calls must be consistent.
func GoUseFunc(fn interface{}) interface{} {
	return runtime.GoUseFuncGlobal(fn)
}

