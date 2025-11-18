//go:build js && wasm
// +build js,wasm

package example

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/state"
)

// AtomExample demonstrates the use of UseAtom with two sibling components
// that share state through atoms. One component controls the state, and the other
// displays and can modify the same state.
func AtomExample(_ Attrs) *Element {
	// Track render count for the main container
	renderCount, setRenderCount := hooks.UseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render AtomExample RENDERS: %d\n", renderCount())

	return dom.Div(Attrs{
		"class": "container",
		"style": "padding: 20px; max-width: 800px; margin: 0 auto;",
	},
		dom.H1(nil, dom.Text("UseAtom Example - Shared State Between Components")),
		dom.Div(Attrs{
			"style": "background: #e9ecef; padding: 10px; border-radius: 5px; margin: 10px 0; text-align: center; border: 2px solid #6c757d;",
		},
			dom.Strong(nil, dom.Text(fmt.Sprintf("🔄 MAIN CONTAINER RENDERS: %d", renderCount()))),
		),
		dom.P(nil, dom.Text("This example demonstrates how two sibling components can share state using UseAtom.")),

		// Two sibling components that share state
		dom.Div(Attrs{
			"style": "display: flex; gap: 20px; margin-top: 20px;",
		},
			// Left component - Counter Controller
			dom.Div(Attrs{
				"style": "flex: 1; padding: 20px; border: 2px solid #007bff; border-radius: 8px;",
			},
				CounterController(nil),
			),

			// Right component - Counter Display
			dom.Div(Attrs{
				"style": "flex: 1; padding: 20px; border: 2px solid #28a745; border-radius: 8px;",
			},
				CounterDisplay(nil),
			),
		),

		// Another pair of components sharing different state
		dom.Div(Attrs{
			"style": "display: flex; gap: 20px; margin-top: 20px;",
		},
			// Left component - Text Input
			dom.Div(Attrs{
				"style": "flex: 1; padding: 20px; border: 2px solid #ffc107; border-radius: 8px;",
			},
				TextInputComponent(nil),
			),

			// Right component - Text Display
			dom.Div(Attrs{
				"style": "flex: 1; padding: 20px; border: 2px solid #dc3545; border-radius: 8px;",
			},
				TextDisplayComponent(nil),
			),
		),
	)
}

// CounterController component manages the counter state
func CounterController(_ Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := hooks.UseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render CounterController RENDERS: %d\n", renderCount())

	// Use atom to share state globally with a unique ID
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
		dom.H3(Attrs{"style": "color: #007bff;"}, dom.Text("Counter Controller")),
		dom.Div(Attrs{
			"style": "background: #cce5ff; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #007bff;",
		},
			dom.Strong(Attrs{"style": "color: #004085; font-size: 16px;"},
				dom.Text(fmt.Sprintf("🔄 RENDERS: %d", renderCount()))),
		),
		dom.P(nil, dom.Text(fmt.Sprintf("Current count: %d", count()))),

		dom.Div(Attrs{
			"style": "display: flex; gap: 10px; margin-top: 10px;",
		},
			dom.Button(Attrs{
				"onclick": handleIncrement,
				"style":   "padding: 10px 15px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;",
			},
				dom.Text("Increment (+1)"),
			),
			dom.Button(Attrs{
				"onclick": handleDecrement,
				"style":   "padding: 10px 15px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer;",
			},
				dom.Text("Decrement (-1)"),
			),
			dom.Button(Attrs{
				"onclick": handleReset,
				"style":   "padding: 10px 15px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;",
			},
				dom.Text("Reset"),
			),
		),
	)
}

// CounterDisplay component displays and can modify the same counter state
func CounterDisplay(_ Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := hooks.UseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render CounterDisplay RENDERS: %d\n", renderCount())

	// Use the same atom ID to access the shared state
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
		dom.H3(Attrs{"style": "color: #28a745;"}, dom.Text("Counter Display & Modifier")),
		dom.Div(Attrs{
			"style": "background: #d4edda; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #28a745;",
		},
			dom.Strong(Attrs{"style": "color: #155724; font-size: 16px;"},
				dom.Text(fmt.Sprintf("🔄 RENDERS: %d", renderCount()))),
		),
		dom.P(nil, dom.Text(fmt.Sprintf("Shared count value: %d", count()))),
		dom.P(nil, dom.Text(fmt.Sprintf("Count squared: %d", count()*count()))),

		dom.Div(Attrs{
			"style": "display: flex; gap: 10px; margin-top: 10px;",
		},
			dom.Button(Attrs{
				"onclick": handleDouble,
				"style":   "padding: 10px 15px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;",
			},
				dom.Text("Double (×2)"),
			),
			dom.Button(Attrs{
				"onclick": handleHalf,
				"style":   "padding: 10px 15px; background: #17a2b8; color: white; border: none; border-radius: 4px; cursor: pointer;",
			},
				dom.Text("Half (÷2)"),
			),
		),
	)
}

// TextInputComponent allows user to input text
func TextInputComponent(_ Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := hooks.UseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render TextInputComponent RENDERS: %d\n", renderCount())

	// Use atom to share text state
	text, setText := state.UseAtom("shared-text", "Hello, World!")

	handleInput := hooks.GoUseFunc(func(event dom.GoEvent) {
		setText(event.GetValue())
	})

	handleClear := hooks.GoUseFunc(func(event dom.GoEvent) {
		event.PreventDefault()
		setText("")
	})

	return dom.Div(nil,
		dom.H3(Attrs{"style": "color: #ffc107;"}, dom.Text("Text Input Controller")),
		dom.Div(Attrs{
			"style": "background: #fff3cd; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #ffc107;",
		},
			dom.Strong(Attrs{"style": "color: #856404; font-size: 16px;"},
				dom.Text(fmt.Sprintf("🔄 RENDERS: %d", renderCount()))),
		),
		dom.P(nil, dom.Text("Enter text below:")),

		dom.Div(Attrs{
			"style": "margin: 10px 0;",
		},
			dom.Input(Attrs{
				"type":        "text",
				"value":       text(),
				"oninput":     handleInput,
				"placeholder": "Type something...",
				"style":       "width: 100%; padding: 10px; border: 1px solid #ccc; border-radius: 4px;",
			}),
		),

		dom.Button(Attrs{
			"onclick": handleClear,
			"style":   "padding: 10px 15px; background: #ffc107; color: black; border: none; border-radius: 4px; cursor: pointer;",
		},
			dom.Text("Clear Text"),
		),
	)
}

// TextDisplayComponent displays and manipulates the shared text
func TextDisplayComponent(_ Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := hooks.UseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render TextDisplayComponent RENDERS: %d\n", renderCount())

	// Use the same atom ID to access the shared text state
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
		dom.H3(Attrs{"style": "color: #dc3545;"}, dom.Text("Text Display & Transformer")),
		dom.Div(Attrs{
			"style": "background: #f8d7da; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #dc3545;",
		},
			dom.Strong(Attrs{"style": "color: #721c24; font-size: 16px;"},
				dom.Text(fmt.Sprintf("🔄 RENDERS: %d", renderCount()))),
		),
		dom.P(nil, dom.Text("Shared text:")),

		dom.Div(Attrs{
			"style": "background: #f8f9fa; padding: 15px; border-radius: 4px; margin: 10px 0; font-family: monospace; word-break: break-all;",
		},
			dom.Text(fmt.Sprintf("\"%s\"", textStr)),
		),

		dom.P(nil, dom.Text(fmt.Sprintf("Character count: %d", len(textStr)))),

		dom.Div(Attrs{
			"style": "display: flex; gap: 10px; margin-top: 10px;",
		},
			dom.Button(Attrs{
				"onclick": handleReverse,
				"style":   "padding: 10px 15px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;",
			},
				dom.Text("Reverse Text"),
			),
		),
	)
}
