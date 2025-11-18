//go:build js && wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
	"github.com/monstercameron/GoWebComponents/state"
)

type Attrs = dom.Attrs
type Element = render.Element

func AtomExample(_ Attrs) *Element {
	return dom.Div(Attrs{
		"class": "container mx-auto p-8 max-w-4xl",
	},
		dom.H1(Attrs{
			"class": "text-3xl font-bold mb-4 text-gray-800",
		}, dom.Text("UseAtom Example - Shared State")),

		dom.P(Attrs{
			"class": "mb-6 text-gray-600",
		}, dom.Text("Demonstrates how sibling components share state using atoms.")),

		// Counter pair
		dom.Div(Attrs{
			"class": "grid grid-cols-2 gap-6 mb-6",
		},
			dom.Div(Attrs{
				"class": "p-6 border-2 border-blue-500 rounded-lg bg-white",
			}, CounterController(nil)),

			dom.Div(Attrs{
				"class": "p-6 border-2 border-green-500 rounded-lg bg-white",
			}, CounterDisplay(nil)),
		),

		// Text pair
		dom.Div(Attrs{
			"class": "grid grid-cols-2 gap-6",
		},
			dom.Div(Attrs{
				"class": "p-6 border-2 border-yellow-500 rounded-lg bg-white",
			}, TextInputComponent(nil)),

			dom.Div(Attrs{
				"class": "p-6 border-2 border-red-500 rounded-lg bg-white",
			}, TextDisplayComponent(nil)),
		),
	)
}

func CounterController(_ Attrs) *Element {
	count, setCount := state.UseAtom("shared-counter", 0)

	handleIncrement := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setCount(count() + 1)
	})

	handleDecrement := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setCount(count() - 1)
	})

	handleReset := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setCount(0)
	})

	return dom.Div(nil,
		dom.H3(Attrs{
			"class": "text-xl font-semibold mb-3 text-blue-600",
		}, dom.Text("Counter Controller")),

		dom.P(Attrs{
			"class": "mb-4 text-gray-700",
		}, dom.Text(fmt.Sprintf("Current count: %d", count()))),

		dom.Div(Attrs{
			"class": "flex gap-2",
		},
			dom.Button(Attrs{
				"onclick": handleIncrement,
				"class":   "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600 transition-colors",
			}, dom.Text("Increment")),
			
			dom.Button(Attrs{
				"onclick": handleDecrement,
				"class":   "px-4 py-2 bg-gray-500 text-white rounded hover:bg-gray-600 transition-colors",
			}, dom.Text("Decrement")),
			
			dom.Button(Attrs{
				"onclick": handleReset,
				"class":   "px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors",
			}, dom.Text("Reset")),
		),
	)
}

func CounterDisplay(_ Attrs) *Element {
	count, setCount := state.UseAtom("shared-counter", 0)

	handleDouble := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setCount(count() * 2)
	})

	handleHalf := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setCount(count() / 2)
	})

	return dom.Div(nil,
		dom.H3(Attrs{
			"class": "text-xl font-semibold mb-3 text-green-600",
		}, dom.Text("Counter Display")),

		dom.P(Attrs{
			"class": "mb-2 text-gray-700",
		}, dom.Text(fmt.Sprintf("Shared count value: %d", count()))),
		
		dom.P(Attrs{
			"class": "mb-4 text-gray-700",
		}, dom.Text(fmt.Sprintf("Count squared: %d", count()*count()))),

		dom.Div(Attrs{
			"class": "flex gap-2",
		},
			dom.Button(Attrs{
				"onclick": handleDouble,
				"class":   "px-4 py-2 bg-green-500 text-white rounded hover:bg-green-600 transition-colors",
			}, dom.Text("Double (×2)")),
			
			dom.Button(Attrs{
				"onclick": handleHalf,
				"class":   "px-4 py-2 bg-teal-500 text-white rounded hover:bg-teal-600 transition-colors",
			}, dom.Text("Half (÷2)")),
		),
	)
}

func TextInputComponent(_ Attrs) *Element {
	text, setText := state.UseAtom("shared-text", "Hello, World!")

	handleInput := hooks.GoUseFunc(func(event dom.GoEvent) {
		setText(event.GetValue())
	})

	handleClear := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setText("")
	})

	return dom.Div(nil,
		dom.H3(Attrs{
			"class": "text-xl font-semibold mb-3 text-yellow-600",
		}, dom.Text("Text Input")),

		dom.P(Attrs{
			"class": "mb-2 text-gray-700",
		}, dom.Text("Enter text below:")),

		dom.Div(Attrs{
			"class": "mb-4",
		},
			dom.Input(Attrs{
				"type":        "text",
				"value":       text(),
				"oninput":     handleInput,
				"placeholder": "Type something...",
				"class":       "w-full px-4 py-2 border border-gray-300 rounded focus:outline-none focus:ring-2 focus:ring-yellow-500",
			}),
		),

		dom.Button(Attrs{
			"onclick": handleClear,
			"class":   "px-4 py-2 bg-yellow-500 text-white rounded hover:bg-yellow-600 transition-colors",
		}, dom.Text("Clear Text")),
	)
}

func TextDisplayComponent(_ Attrs) *Element {
	text, setText := state.UseAtom("shared-text", "Hello, World!")

	handleReverse := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		runes := []rune(text())
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		setText(string(runes))
	})

	textStr := text()
	return dom.Div(nil,
		dom.H3(Attrs{
			"class": "text-xl font-semibold mb-3 text-red-600",
		}, dom.Text("Text Display")),

		dom.P(Attrs{
			"class": "mb-2 text-gray-700",
		}, dom.Text("Shared text:")),

		dom.Div(Attrs{
			"class": "bg-gray-100 p-4 rounded mb-3 font-mono break-all",
		}, dom.Text(fmt.Sprintf("\"%s\"", textStr))),

		dom.P(Attrs{
			"class": "mb-4 text-gray-700",
		}, dom.Text(fmt.Sprintf("Character count: %d", len(textStr)))),

		dom.Button(Attrs{
			"onclick": handleReverse,
			"class":   "px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600 transition-colors",
		}, dom.Text("Reverse Text")),
	)
}

func main() {
	container := js.Global().Get("document").Call("getElementById", "app")
	element := dom.CreateElement(AtomExample, nil)
	render.ToElement(element, container)
	select {}
}
