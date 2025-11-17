//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/dom"
	"github.com/monstercameron/GoWebComponents/hooks"
	"github.com/monstercameron/GoWebComponents/render"
	"github.com/monstercameron/GoWebComponents/state"
)

// Counter is a reusable component for testing component reuse
func Counter(props dom.Attrs) *dom.Element {
	id := ""
	if props != nil {
		if idVal, ok := props["id"].(string); ok {
			id = idVal
		}
	}

	count, setCount := hooks.UseState(0)

	increment := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		setCount(func(prev int) int {
			return prev + 1
		})
		return nil
	})

	return dom.Div(dom.Attrs{"class": "counter-instance", "data-counter-id": id},
		dom.P(dom.Attrs{"class": "counter-value"},
			dom.Text(fmt.Sprintf("Counter %s: %d", id, count())),
		),
		dom.Button(dom.Attrs{
			"onclick": increment,
			"class":   "counter-btn px-2 py-1 bg-purple-500 text-white",
		}, dom.Text("+")),
	)
}

// Render counters for reactivity demo
var reactARenders int
var reactBRenders int
var reactBatchRenders int

// ReactA component - independent state
func ReactA(props dom.Attrs) *dom.Element {
	value, setValue := hooks.UseState(0)
	reactARenders++
	inc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		setValue(func(prev int) int { return prev + 1 })
		return nil
	})
	return dom.Div(dom.Attrs{"id": "react-a"},
		dom.P(dom.Attrs{"id": "react-a-value"}, dom.Text(fmt.Sprintf("A Value: %d", value()))),
		dom.P(dom.Attrs{"id": "react-a-renders"}, dom.Text(fmt.Sprintf("A Renders: %d", reactARenders))),
		dom.Button(dom.Attrs{"id": "react-a-inc", "onclick": inc}, dom.Text("Inc A")),
	)
}

// ReactB component - should not re-render when A changes
func ReactB(props dom.Attrs) *dom.Element {
	value, _ := hooks.UseState(0) // static for this test
	reactBRenders++
	return dom.Div(dom.Attrs{"id": "react-b"},
		dom.P(dom.Attrs{"id": "react-b-value"}, dom.Text(fmt.Sprintf("B Value: %d", value()))),
		dom.P(dom.Attrs{"id": "react-b-renders"}, dom.Text(fmt.Sprintf("B Renders: %d", reactBRenders))),
	)
}

// ReactBatch component - demonstrates batched logical update
func ReactBatch(props dom.Attrs) *dom.Element {
	value, setValue := hooks.UseState(0)
	reactBatchRenders++
	batch := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Single state update applying multiple increments
		setValue(func(prev int) int { return prev + 3 })
		return nil
	})
	return dom.Div(dom.Attrs{"id": "react-batch"},
		dom.P(dom.Attrs{"id": "react-batch-value"}, dom.Text(fmt.Sprintf("Batch Value: %d", value()))),
		dom.P(dom.Attrs{"id": "react-batch-renders"}, dom.Text(fmt.Sprintf("Batch Renders: %d", reactBatchRenders))),
		dom.Button(dom.Attrs{"id": "react-batch-btn", "onclick": batch}, dom.Text("Batch +3")),
	)
}

// ReactivityDemo aggregates reactivity test components
func ReactivityDemo(props dom.Attrs) *dom.Element {
	return dom.Div(dom.Attrs{"id": "reactivity-demo", "class": "mt-8"},
		dom.H2(nil, dom.Text("Reactivity Demo")),
		&dom.Element{Type: ReactA},
		&dom.Element{Type: ReactB},
		&dom.Element{Type: ReactBatch},
	)
}

// HelloWorld component demonstrates basic usage
func HelloWorld(props dom.Attrs) *dom.Element {
	count, setCount := hooks.UseState(0)
	// Setup shared atom for demonstration/testing
	atomGet, atomSet := state.UseAtom("sharedCounter", 0)
	// Input state for onchange test
	inputValue, setInputValue := hooks.UseState("")
	// Submit state for form test
	submitValue, setSubmitValue := hooks.UseState("")

	// UseEffect to log on mount and count changes
	hooks.UseEffect(func() func() {
		fmt.Printf("UseEffect ran: count is %d\n", count())
		return nil // No cleanup needed for this simple example
	}, count())

	// UseMemo to compute expensive value (for testing)
	doubledCount := hooks.UseMemo(func() interface{} {
		fmt.Printf("UseMemo computing: count=%d\n", count())
		return count() * 2
	}, count()).(int)

	// Create increment handler using functional setState
	increment := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		setCount(func(prev int) int {
			return prev + 1
		})
		return nil
	})

	// Atom increment handler
	atomIncrement := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		atomSet(atomGet() + 1)
		return nil
	})

	// Input onchange handler
	handleInputChange := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			event := args[0]
			value := event.Get("target").Get("value").String()
			setInputValue(value)
		}
		return nil
	})

	// Form onsubmit handler with preventDefault
	handleSubmit := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			event := args[0]
			event.Call("preventDefault")
			// Get the form input value
			formInput := event.Get("target").Call("querySelector", "#form-input")
			value := formInput.Get("value").String()
			setSubmitValue(value)
		}
		return nil
	})

	return dom.Div(dom.Attrs{"class": "container mx-auto p-8"},
		dom.H1(dom.Attrs{"class": "text-4xl font-bold mb-4", "id": "main-heading"},
			dom.Text("GoWebComponents Test"),
		),
		dom.P(dom.Attrs{"class": "mb-4", "data-testid": "count-display"},
			dom.Text(fmt.Sprintf("Count: %d", count())),
		),
		dom.P(dom.Attrs{"class": "mb-4", "id": "doubled", "style": "font-weight: bold;"},
			dom.Text(fmt.Sprintf("Doubled: %d", doubledCount)),
		),
		dom.Button(dom.Attrs{
			"onclick": increment,
			"class":   "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600",
		},
			dom.Text("Increment"),
		),
		dom.Div(nil,
			dom.P(dom.Attrs{"id": "atom-value-a"}, dom.Text(fmt.Sprintf("AtomA: %d", atomGet()))),
			dom.P(dom.Attrs{"id": "atom-value-b"}, dom.Text(fmt.Sprintf("AtomB: %d", atomGet()))),
			dom.Button(dom.Attrs{"id": "atom-increment", "onclick": atomIncrement}, dom.Text("Atom Increment")),
		),
		dom.Div(dom.Attrs{"class": "mt-4"},
			dom.H2(nil, dom.Text("Input Test")),
			dom.Input(dom.Attrs{
				"id":       "test-input",
				"type":     "text",
				"onchange": handleInputChange,
				"class":    "border p-2",
			}),
			dom.P(dom.Attrs{"id": "input-value"},
				dom.Text(fmt.Sprintf("Input: %s", inputValue())),
			),
		),
		dom.Div(dom.Attrs{"class": "mt-4"},
			dom.H2(nil, dom.Text("Form Test")),
			dom.Form(dom.Attrs{
				"id":       "test-form",
				"onsubmit": handleSubmit,
			},
				dom.Input(dom.Attrs{
					"id":    "form-input",
					"type":  "text",
					"class": "border p-2",
				}),
				dom.Button(dom.Attrs{
					"type":  "submit",
					"class": "ml-2 px-4 py-2 bg-green-500 text-white",
				}, dom.Text("Submit")),
			),
			dom.P(dom.Attrs{"id": "submit-value"},
				dom.Text(fmt.Sprintf("Submitted: %s", submitValue())),
			),
		),
		dom.Div(dom.Attrs{"class": "mt-4", "id": "reusable-components"},
			dom.H2(nil, dom.Text("Reusable Components")),
			&dom.Element{Type: Counter, Props: dom.Attrs{"id": "A"}},
			&dom.Element{Type: Counter, Props: dom.Attrs{"id": "B"}},
			&dom.Element{Type: Counter, Props: dom.Attrs{"id": "C"}},
		),
		dom.Div(nil,
			&dom.Element{Type: ReactivityDemo},
		),
	)
}

func main() {
	fmt.Println("🚀 GoWebComponents WASM initialized")

	// Create root element that will call HelloWorld during render
	app := &dom.Element{
		Type:  HelloWorld,
		Props: make(map[string]interface{}),
	}

	fmt.Println("About to call render.To...")
	render.To(app, "#app")
	fmt.Println("render.To completed")

	// Keep the Go program running
	select {}
}
