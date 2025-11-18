//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
)

type Attrs = dom.Attrs
type Element = render.Element

// Counter component - demonstrates basic number state management
func CounterExample(_ Attrs) *Element {
	count, setCount := hooks.UseState(0)
	currentCount := count()

	increment := func(this js.Value, args []js.Value) interface{} {
		setCount(currentCount + 1)
		return nil
	}

	decrement := func(this js.Value, args []js.Value) interface{} {
		setCount(currentCount - 1)
		return nil
	}

	reset := func(this js.Value, args []js.Value) interface{} {
		setCount(0)
		return nil
	}

	return dom.Div(Attrs{
		"class": "max-w-md mx-auto mt-10 p-6 bg-white rounded-lg shadow-lg",
	},
		dom.H2(Attrs{
			"class": "text-2xl font-bold text-gray-800 mb-4",
		}, dom.Text("Counter Example")),

		dom.Div(Attrs{
			"class": "text-center mb-6",
		},
			dom.Div(Attrs{
				"class": "text-6xl font-bold text-blue-600 mb-2",
			}, dom.Text(fmt.Sprintf("%d", currentCount))),
			dom.P(Attrs{
				"class": "text-gray-600",
			}, dom.Text("Current Count")),
		),

		dom.Div(Attrs{
			"class": "flex gap-3 justify-center",
		},
			dom.Button(Attrs{
				"onclick": js.FuncOf(decrement),
				"class":   "px-6 py-3 bg-red-500 text-white font-semibold rounded-lg hover:bg-red-600 transition-colors",
			}, dom.Text("−")),

			dom.Button(Attrs{
				"onclick": js.FuncOf(reset),
				"class":   "px-6 py-3 bg-gray-500 text-white font-semibold rounded-lg hover:bg-gray-600 transition-colors",
			}, dom.Text("Reset")),

			dom.Button(Attrs{
				"onclick": js.FuncOf(increment),
				"class":   "px-6 py-3 bg-green-500 text-white font-semibold rounded-lg hover:bg-green-600 transition-colors",
			}, dom.Text("+")),
		),
	)
}

func main() {
	fmt.Println("🚀 Counter Example Started")
	
	// Find the DOM container
	container := js.Global().Get("document").Call("getElementById", "app")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("❌ No element with id 'app' found!")
		return
	}

	// Create and render the element
	element := dom.CreateElement(CounterExample, nil)
	render.ToElement(element, container)

	fmt.Println("✅ Counter Example Rendered")

	// Keep the Go program running
	select {}
}
