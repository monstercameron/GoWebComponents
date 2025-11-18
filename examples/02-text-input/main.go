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

// Text input component - demonstrates string state management
func TextInputExample(_ Attrs) *Element {
	text, setText := hooks.UseState("")
	currentText := text()

	handleInput := func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			newText := args[0].Get("target").Get("value").String()
			setText(newText)
		}
		return nil
	}

	clear := func(this js.Value, args []js.Value) interface{} {
		setText("")
		return nil
	}

	return dom.Div(Attrs{
		"class": "max-w-md mx-auto mt-10 p-6 bg-white rounded-lg shadow-lg",
	},
		dom.H2(Attrs{
			"class": "text-2xl font-bold text-gray-800 mb-4",
		}, dom.Text("Text Input Example")),

		dom.Div(Attrs{
			"class": "mb-4",
		},
			dom.Label(Attrs{
				"class": "block text-gray-700 text-sm font-bold mb-2",
			}, dom.Text("Type something:")),

			dom.Input(Attrs{
				"type":        "text",
				"value":       currentText,
				"oninput":     js.FuncOf(handleInput),
				"class":       "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 text-black",
				"placeholder": "Enter text here...",
			}),
		),

		dom.Div(Attrs{
			"class": "mb-4",
		},
			dom.Button(Attrs{
				"onclick": js.FuncOf(clear),
				"class":   "px-4 py-2 bg-red-500 text-white font-semibold rounded-lg hover:bg-red-600 transition-colors",
			}, dom.Text("Clear")),
		),

		dom.Div(Attrs{
			"class": "p-4 bg-gray-50 rounded-lg",
		},
			dom.P(Attrs{
				"class": "text-gray-700 mb-2",
			}, dom.Text(fmt.Sprintf("You typed: %s", currentText))),

			dom.P(Attrs{
				"class": "text-gray-600 text-sm",
			}, dom.Text(fmt.Sprintf("Character count: %d", len(currentText)))),
		),
	)
}

func main() {
	fmt.Println("🚀 Text Input Example Started")

	// Find the DOM container
	container := js.Global().Get("document").Call("getElementById", "app")
	if container.IsUndefined() || container.IsNull() {
		fmt.Println("❌ No element with id 'app' found!")
		return
	}

	// Create and render the element
	element := dom.CreateElement(TextInputExample, nil)
	render.ToElement(element, container)

	fmt.Println("✅ Text Input Example Rendered")

	// Keep the Go program running
	select {}
}
