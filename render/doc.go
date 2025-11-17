// Package render provides functions to render GoWebComponents to the DOM.
//
// This package is the entry point for mounting your application to the browser DOM.
// It handles the initial render and sets up the reactive rendering system that
// automatically updates the DOM when component state changes.
//
// Basic usage:
//
//	import (
//	    "github.com/monstercameron/GoWebComponents/render"
//	    "github.com/monstercameron/GoWebComponents/dom"
//	)
//
//	func main() {
//	    app := dom.Div(nil,
//	        dom.H1(nil, "Hello, GoWebComponents!"),
//	        dom.P(nil, "Welcome to reactive WASM development"),
//	    )
//
//	    render.To(app, "#app")
//	}
//
// The render package provides two main functions:
//
//   - To: Renders to a CSS selector (most common)
//   - ToElement: Renders to a specific DOM element (advanced)
//
// After the initial render, the framework automatically manages subsequent renders
// when component state changes through hooks like hooks.UseState or state.UseAtom.
//
// Example with a component:
//
//	func App(props dom.Attrs) *fiber.Element {
//	    count, setCount := hooks.UseState(0)
//
//	    return dom.Div(nil,
//	        dom.H1(nil, fmt.Sprintf("Count: %d", count())),
//	        dom.Button(map[string]interface{}{
//	            "onclick": js.FuncOf(func(this js.Value, args []js.Value) interface{} {
//	                setCount(count() + 1) // Automatically triggers re-render
//	                return nil
//	            }),
//	        }, "Increment"),
//	    )
//	}
//
//	func main() {
//	    render.To(App(nil), "#app")
//	}
//
// The render package integrates with the fiber reconciliation engine to provide
// efficient, incremental DOM updates. Only the parts of the DOM that actually
// changed are updated, making it performant even for complex UIs.
package render
