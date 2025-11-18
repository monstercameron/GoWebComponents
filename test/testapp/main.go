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

	increment := hooks.GoUseFunc(func() {
		setCount(func(prev int) int {
			return prev + 1
		})
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
	inc := hooks.GoUseFunc(func() {
		setValue(func(prev int) int { return prev + 1 })
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
	batch := hooks.GoUseFunc(func() {
		// Single state update applying multiple increments
		setValue(func(prev int) int { return prev + 3 })
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

// EffectChild demonstrates UseEffect cleanup on unmount
func EffectChild(props dom.Attrs) *dom.Element {
	// Use an atom to track lifecycle status so cleanup can update a node outside the child
	_, cleanupSet := state.UseAtom("cleanupStatus", "")
	hooks.UseEffect(func() func() {
		// On mount: update global cleanup status and DOM directly
		cleanupSet("mounted")
		fmt.Println("EffectChild mounted")
		js.Global().Get("document").Call("querySelector", "#cleanup-status").Set("textContent", "mounted")
		return func() {
			// On unmount: update and log
			cleanupSet("cleaned")
			fmt.Println("EffectChild cleaned up")
			js.Global().Get("document").Call("querySelector", "#cleanup-status").Set("textContent", "cleaned")
		}
	}, []interface{}{})
	return dom.Div(dom.Attrs{"id": "effect-child"},
		dom.P(dom.Attrs{"id": "effect-child-text"}, dom.Text("Effect Child")),
	)
}

// Toggle demo to mount and unmount EffectChild
func ToggleEffectDemo(props dom.Attrs) *dom.Element {
	show, setShow := hooks.UseState(false)
	toggle := hooks.GoUseFunc(func() {
		setShow(func(prev bool) bool { return !prev })
	})
	if show() {
		return dom.Div(dom.Attrs{"id": "toggle-effect-demo"},
			dom.Button(dom.Attrs{"id": "toggle-child-btn", "onclick": toggle}, dom.Text("Toggle Child")),
			&dom.Element{Type: EffectChild},
		)
	}
	return dom.Div(dom.Attrs{"id": "toggle-effect-demo"},
		dom.Button(dom.Attrs{"id": "toggle-child-btn", "onclick": toggle}, dom.Text("Toggle Child")),
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

	// UseEffect to log on mount and count changes (also set title once)
	hooks.UseEffect(func() func() {
		fmt.Printf("UseEffect ran: count is %d\n", count())
		// Set document title on mount
		js.Global().Get("document").Set("title", "GoWebComponents App")
		return nil // No cleanup needed for this simple example
	}, count())

	// UseMemo to compute expensive value (for testing)
	doubledCount := hooks.UseMemo(func() interface{} {
		result := count() * 2
		fmt.Printf("UseMemo computing: count=%d\n", count())
		return result
	}, count()).(int)

	// cleanup-status will be updated by child effect directly via Document API

	// Create increment handler using functional setState
	increment := hooks.GoUseFunc(func() {
		setCount(func(prev int) int {
			return prev + 1
		})
	})

	// Atom increment handler
	atomIncrement := hooks.GoUseFunc(func() {
		atomSet(atomGet() + 1)
	})

	// Input onchange handler
	handleInputChange := hooks.GoUseFunc(func(value string) {
		setInputValue(value)
	})

	// Form onsubmit handler with preventDefault
	handleSubmit := hooks.GoUseFunc(func(event js.Value) {
		event.Call("preventDefault")
		// Get the form input value
		formInput := event.Get("target").Call("querySelector", "#form-input")
		value := formInput.Get("value").String()
		setSubmitValue(value)
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
			"onclick":    increment,
			"class":      "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600",
			"aria-label": "Increment main",
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
				"id":      "test-input",
				"type":    "text",
				"oninput": handleInputChange,
				"class":   "border p-2",
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
		dom.Div(dom.Attrs{"class": "mt-4", "id": "reusable-components", "role": "region", "aria-label": "Reusable Counters Section"},
			dom.H2(nil, dom.Text("Reusable Components")),
			&dom.Element{Type: Counter, Props: dom.Attrs{"id": "A"}},
			&dom.Element{Type: Counter, Props: dom.Attrs{"id": "B"}},
			&dom.Element{Type: Counter, Props: dom.Attrs{"id": "C"}},
		),
		dom.Div(dom.Attrs{"role": "region", "aria-label": "Reactivity Demo Section"},
			&dom.Element{Type: ReactivityDemo},
		),
		dom.Div(dom.Attrs{"role": "region", "aria-label": "Todo App Section", "id": "todo-app"},
			&dom.Element{Type: Header},
			&dom.Element{Type: TodoList},
		),
		dom.Div(dom.Attrs{"class": "mt-4"},
			&dom.Element{Type: ToggleEffectDemo},
		),
		dom.P(dom.Attrs{"id": "cleanup-status"}, dom.Text("")),
		// Add new hook tests
		&dom.Element{Type: UseIdTestComponent},
		&dom.Element{Type: UseFetchTestComponent},
		// Add component that intentionally uses invalid props to ensure graceful handling
		&dom.Element{Type: BadProps},
	)
}

// BadProps passes intentionally invalid properties to test error handling
func BadProps(props dom.Attrs) *dom.Element {
	// class attribute as non-string, onclick as non-function to simulate invalid props
	return dom.Div(dom.Attrs{"id": "bad-props", "class": 12345, "onclick": "not-a-function"},
		dom.Text("BadProps"),
	)
}

// UseIdTestComponent demonstrates UseId hook for accessibility
func UseIdTestComponent(props dom.Attrs) *dom.Element {
	inputId := hooks.UseId()
	selectId := hooks.UseId()
	checkboxId := hooks.UseId()

	return dom.Div(dom.Attrs{"id": "use-id-test", "class": "mt-8"},
		dom.H2(nil, dom.Text("UseId Test")),
		dom.Div(dom.Attrs{"class": "mb-4"},
			dom.Label(dom.Attrs{
				"htmlFor": inputId,
				"id":      "input-label",
				"class":   "block mb-2",
			}, dom.Text("Test Input:")),
			dom.Input(dom.Attrs{
				"id":    inputId,
				"type":  "text",
				"class": "border p-2",
			}),
			dom.P(dom.Attrs{"id": "input-id-display"},
				dom.Text(fmt.Sprintf("Input ID: %s", inputId)),
			),
		),
		dom.Div(dom.Attrs{"class": "mb-4"},
			dom.Label(dom.Attrs{
				"htmlFor": selectId,
				"id":      "select-label",
				"class":   "block mb-2",
			}, dom.Text("Test Select:")),
			dom.Select(dom.Attrs{
				"id": selectId,
			},
				dom.Option(dom.Attrs{"value": "1"}, dom.Text("Option 1")),
				dom.Option(dom.Attrs{"value": "2"}, dom.Text("Option 2")),
			),
			dom.P(dom.Attrs{"id": "select-id-display"},
				dom.Text(fmt.Sprintf("Select ID: %s", selectId)),
			),
		),
		dom.Div(dom.Attrs{"class": "mb-4"},
			dom.Div(nil,
				dom.Input(dom.Attrs{
					"id":   checkboxId,
					"type": "checkbox",
				}),
				dom.Label(dom.Attrs{
					"htmlFor": checkboxId,
					"id":      "checkbox-label",
					"class":   "ml-2",
				}, dom.Text("Test Checkbox")),
			),
			dom.P(dom.Attrs{"id": "checkbox-id-display"},
				dom.Text(fmt.Sprintf("Checkbox ID: %s", checkboxId)),
			),
		),
	)
}

// UseFetchTestComponent demonstrates UseFetch hook
func UseFetchTestComponent(props dom.Attrs) *dom.Element {
	// Mock data URL - in real tests, this would be a test server endpoint
	userState, refetchUser := hooks.UseFetch("/api/user/123")

	// Use GoUseFunc hook for cleaner event handler
	fetchHandler := hooks.GoUseFunc(func() {
		refetchUser()
	})

	state := userState()

	// Render based on fetch state
	var content *dom.Element
	if state.Loading {
		content = dom.P(dom.Attrs{"id": "fetch-loading"}, dom.Text("Loading..."))
	} else if state.Error != "" {
		content = dom.P(dom.Attrs{
			"id":    "fetch-error",
			"style": "color: red;",
		}, dom.Text(fmt.Sprintf("Error: %s", state.Error)))
	} else if state.Data != nil {
		content = dom.P(dom.Attrs{"id": "fetch-data"},
			dom.Text(fmt.Sprintf("Data: %v", state.Data)),
		)
	} else {
		content = dom.P(dom.Attrs{"id": "fetch-idle"},
			dom.Text("No data fetched yet"),
		)
	}

	return dom.Div(dom.Attrs{"id": "use-fetch-test", "class": "mt-8"},
		dom.H2(nil, dom.Text("UseFetch Test")),
		dom.Button(dom.Attrs{
			"id":      "fetch-button",
			"onclick": fetchHandler,
			"class":   "px-4 py-2 bg-blue-500 text-white",
		}, dom.Text("Fetch User Data")),
		dom.Div(dom.Attrs{"class": "mt-4"},
			content,
		),
		dom.P(dom.Attrs{"id": "fetch-state-display"},
			dom.Text(fmt.Sprintf("State: Loading=%v, Error=%s", state.Loading, state.Error)),
		),
	)
}

func main() {
	fmt.Println("🚀 GoWebComponents WASM initialized")

	// Debug: Verify main is running by updating DOM directly
	js.Global().Get("document").Call("getElementById", "app").Set("innerHTML", "<div style='color: green'>Main Started</div>")

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
