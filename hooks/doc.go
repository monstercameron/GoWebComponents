// Package hooks provides React-style hooks for managing component state and side effects
// in GoWebComponents.
//
// This package implements the core hook functions that enable functional components
// to manage local state, perform side effects, memoize expensive computations,
// and optimize rendering performance.
//
// Basic usage:
//
//	import "github.com/monstercameron/GoWebComponents/hooks"
//
//	func Counter(props dom.Attrs) *fiber.Element {
//	    count, setCount := hooks.UseState(0)
//
//	    return dom.Div(nil,
//	        dom.H1(nil, fmt.Sprintf("Count: %d", count())),
//	        dom.Button(map[string]interface{}{
//	            "onclick": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	                setCount(count() + 1)
//	                return nil
//	            }),
//	        }, "Increment"),
//	    )
//	}
//
// Available hooks:
//
//   - UseState[T]: Manages local component state with automatic re-rendering
//   - UseEffect: Runs side effects with dependency tracking
//   - UseMemo: Memoizes expensive computations
//   - UseFunc: Creates stable function references (coming soon)
//
// Hook Rules:
//  1. Only call hooks at the top level of your component function
//  2. Don't call hooks inside loops, conditions, or nested functions
//  3. Always call hooks in the same order on every render
//
// Violating these rules will cause runtime errors or unpredictable behavior.
package hooks
