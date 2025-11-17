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

// Note: GoUseFunc is not yet implemented in the fiber package
// func UseFunc(...) { ... }
