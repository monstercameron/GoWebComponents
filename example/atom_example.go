//go:build js && wasm
// +build js,wasm

package example

import (
	"fmt"

	. "github.com/monstercameron/GoWebComponents/fiber"
)

// GoUseAtomExample demonstrates the use of GoUseAtom with two sibling components
// that share state through atoms. One component controls the state, and the other
// displays and can modify the same state.
func GoUseAtomExample(props Attrs) *Element {
	// Track render count for the main container
	renderCount, setRenderCount := GoUseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render GoUseAtomExample RENDERS: %d\n", renderCount())

	return Div(
		Attrs{"class": "container", "style": "padding: 20px; max-width: 800px; margin: 0 auto;"},
		H1(nil, "GoUseAtom Example - Shared State Between Components"),
		Div(
			Attrs{"style": "background: #e9ecef; padding: 10px; border-radius: 5px; margin: 10px 0; text-align: center; border: 2px solid #6c757d;"},
			Strong(nil, fmt.Sprintf("🔄 MAIN CONTAINER RENDERS: %d", renderCount())),
		),
		P(nil, "This example demonstrates how two sibling components can share state using GoUseAtom."),

		// Two sibling components that share state
		Div(
			Attrs{"style": "display: flex; gap: 20px; margin-top: 20px;"},

			// Left component - Counter Controller
			Div(
				Attrs{"style": "flex: 1; padding: 20px; border: 2px solid #007bff; border-radius: 8px;"},
				CounterController,
			),

			// Right component - Counter Display
			Div(
				Attrs{"style": "flex: 1; padding: 20px; border: 2px solid #28a745; border-radius: 8px;"},
				CounterDisplay,
			),
		),

		// Another pair of components sharing different state
		Div(
			Attrs{"style": "display: flex; gap: 20px; margin-top: 20px;"},

			// Left component - Text Input
			Div(
				Attrs{"style": "flex: 1; padding: 20px; border: 2px solid #ffc107; border-radius: 8px;"},
				TextInputComponent,
			),

			// Right component - Text Display
			Div(
				Attrs{"style": "flex: 1; padding: 20px; border: 2px solid #dc3545; border-radius: 8px;"},
				TextDisplayComponent,
			),
		),
	)
}

// CounterController component manages the counter state
func CounterController(props Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := GoUseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render CounterController RENDERS: %d\n", renderCount())

	// Use atom to share state globally with a unique ID
	count, setCount := GoUseAtom("shared-counter", 0)

	handleIncrement := GoUseFunc(func(event GoEvent) {
		setCount(count() + 1)
	})

	handleDecrement := GoUseFunc(func(event GoEvent) {
		setCount(count() - 1)
	})

	handleReset := GoUseFunc(func(event GoEvent) {
		setCount(0)
	})

	return Div(nil,
		H3(Attrs{"style": "color: #007bff;"}, "Counter Controller"),
		Div(
			Attrs{"style": "background: #cce5ff; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #007bff;"},
			Strong(Attrs{"style": "color: #004085; font-size: 16px;"}, fmt.Sprintf("🔄 RENDERS: %d", renderCount())),
		),
		P(nil, fmt.Sprintf("Current count: %d", count())),

		Div(
			Attrs{"style": "display: flex; gap: 10px; margin-top: 10px;"},
			Button(
				Attrs{
					"onclick": handleIncrement,
					"style":   "padding: 10px 15px; background: #007bff; color: white; border: none; border-radius: 4px; cursor: pointer;",
				},
				"Increment (+1)",
			),
			Button(
				Attrs{
					"onclick": handleDecrement,
					"style":   "padding: 10px 15px; background: #6c757d; color: white; border: none; border-radius: 4px; cursor: pointer;",
				},
				"Decrement (-1)",
			),
			Button(
				Attrs{
					"onclick": handleReset,
					"style":   "padding: 10px 15px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;",
				},
				"Reset",
			),
		),
	)
}

// CounterDisplay component displays and can modify the same counter state
func CounterDisplay(props Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := GoUseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render CounterDisplay RENDERS: %d\n", renderCount())

	// Use the same atom ID to access the shared state
	count, setCount := GoUseAtom("shared-counter", 0)

	handleDouble := GoUseFunc(func(event GoEvent) {
		setCount(count() * 2)
	})

	handleHalf := GoUseFunc(func(event GoEvent) {
		setCount(count() / 2)
	})

	return Div(nil,
		H3(Attrs{"style": "color: #28a745;"}, "Counter Display & Modifier"),
		Div(
			Attrs{"style": "background: #d4edda; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #28a745;"},
			Strong(Attrs{"style": "color: #155724; font-size: 16px;"}, fmt.Sprintf("🔄 RENDERS: %d", renderCount())),
		),
		P(nil, fmt.Sprintf("Shared count value: %d", count())),
		P(nil, fmt.Sprintf("Count squared: %d", count()*count())),

		Div(
			Attrs{"style": "display: flex; gap: 10px; margin-top: 10px;"},
			Button(
				Attrs{
					"onclick": handleDouble,
					"style":   "padding: 10px 15px; background: #28a745; color: white; border: none; border-radius: 4px; cursor: pointer;",
				},
				"Double (×2)",
			),
			Button(
				Attrs{
					"onclick": handleHalf,
					"style":   "padding: 10px 15px; background: #17a2b8; color: white; border: none; border-radius: 4px; cursor: pointer;",
				},
				"Half (÷2)",
			),
		),
	)
}

// TextInputComponent allows user to input text
func TextInputComponent(props Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := GoUseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render TextInputComponent RENDERS: %d\n", renderCount())

	// Use atom to share text state
	text, setText := GoUseAtom("shared-text", "Hello, World!")

	handleInput := GoUseFunc(func(event GoEvent) {
		setText(event.GetValue())
	})

	handleClear := GoUseFunc(func(event GoEvent) {
		setText("")
	})

	return Div(nil,
		H3(Attrs{"style": "color: #ffc107;"}, "Text Input Controller"),
		Div(
			Attrs{"style": "background: #fff3cd; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #ffc107;"},
			Strong(Attrs{"style": "color: #856404; font-size: 16px;"}, fmt.Sprintf("🔄 RENDERS: %d", renderCount())),
		),
		P(nil, "Enter text below:"),

		Div(
			Attrs{"style": "margin: 10px 0;"},
			Input(Attrs{
				"type":        "text",
				"value":       text(),
				"oninput":     handleInput,
				"placeholder": "Type something...",
				"style":       "width: 100%; padding: 10px; border: 1px solid #ccc; border-radius: 4px;",
			}),
		),

		Button(
			Attrs{
				"onclick": handleClear,
				"style":   "padding: 10px 15px; background: #ffc107; color: black; border: none; border-radius: 4px; cursor: pointer;",
			},
			"Clear Text",
		),
	)
}

// TextDisplayComponent displays and manipulates the shared text
func TextDisplayComponent(props Attrs) *Element {
	// Track render count to observe fine-grained reactivity
	renderCount, setRenderCount := GoUseState(0)
	setRenderCount(renderCount() + 1)

	// Console log for debugging
	fmt.Printf("render TextDisplayComponent RENDERS: %d\n", renderCount())

	// Use the same atom ID to access the shared text state
	text, setText := GoUseAtom("shared-text", "Hello, World!")

	handleReverse := GoUseFunc(func(event GoEvent) {
		runes := []rune(text())
		for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
			runes[i], runes[j] = runes[j], runes[i]
		}
		setText(string(runes))
	})

	return Div(nil,
		H3(Attrs{"style": "color: #dc3545;"}, "Text Display & Transformer"),
		Div(
			Attrs{"style": "background: #f8d7da; padding: 8px; border-radius: 4px; margin: 8px 0; text-align: center; border-left: 4px solid #dc3545;"},
			Strong(Attrs{"style": "color: #721c24; font-size: 16px;"}, fmt.Sprintf("🔄 RENDERS: %d", renderCount())),
		),
		P(nil, "Shared text:"),

		Div(
			Attrs{
				"style": "background: #f8f9fa; padding: 15px; border-radius: 4px; margin: 10px 0; font-family: monospace; word-break: break-all;",
			},
			fmt.Sprintf("\"%s\"", text()),
		),

		P(nil, fmt.Sprintf("Character count: %d", len(text()))),

		Div(
			Attrs{"style": "display: flex; gap: 10px; margin-top: 10px;"},
			Button(
				Attrs{
					"onclick": handleReverse,
					"style":   "padding: 10px 15px; background: #dc3545; color: white; border: none; border-radius: 4px; cursor: pointer;",
				},
				"Reverse Text",
			),
		),
	)
}
