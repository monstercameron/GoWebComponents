//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/hotreload"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/interop"
	"github.com/monstercameron/GoWebComponents/state"
)

// Counter is a reusable component for testing component reuse
func Counter(props Attrs) *Element {
	id := ""
	if props != nil {
		if idVal, ok := props["id"].(string); ok {
			id = idVal
		}
	}

	count, setCount := UseState(0)

	increment := GoUseFunc(func() {
		setCount(func(prev int) int {
			return prev + 1
		})
	})

	return Div(Attrs{"class": "counter-instance", "data-counter-id": id},
		P(Attrs{"class": "counter-value"},
			Text(fmt.Sprintf("Counter %s: %d", id, count())),
		),
		Button(Attrs{
			"onclick": increment,
			"class":   "counter-btn px-2 py-1 bg-purple-500 text-white",
		}, Text("+")),
	)
}

// Render counters for reactivity demo
var reactARenders int
var reactBRenders int
var reactBatchRenders int
var stressRenders int
var mixedStressRenders int
var mixedStressMirrorRenders int
var fineGrainedParentRenders int
var fineGrainedStaticRenders int

type fineGrainedModel struct {
	Hot int
}

// ReactA component - independent state
func ReactA(props Attrs) *Element {
	value, setValue := UseState(0)
	reactARenders++
	inc := GoUseFunc(func() {
		setValue(func(prev int) int { return prev + 1 })
	})
	return Div(Attrs{"id": "react-a"},
		P(Attrs{"id": "react-a-value"}, Text(fmt.Sprintf("A Value: %d", value()))),
		P(Attrs{"id": "react-a-renders"}, Text(fmt.Sprintf("A Renders: %d", reactARenders))),
		Button(Attrs{"id": "react-a-inc", "onclick": inc}, Text("Inc A")),
	)
}

// ReactB component - should not re-render when A changes
func ReactB(props Attrs) *Element {
	value, _ := UseState(0) // static for this test
	reactBRenders++
	return Div(Attrs{"id": "react-b"},
		P(Attrs{"id": "react-b-value"}, Text(fmt.Sprintf("B Value: %d", value()))),
		P(Attrs{"id": "react-b-renders"}, Text(fmt.Sprintf("B Renders: %d", reactBRenders))),
	)
}

// ReactBatch component - demonstrates batched logical update
func ReactBatch(props Attrs) *Element {
	value, setValue := UseState(0)
	reactBatchRenders++
	batch := GoUseFunc(func() {
		// Single state update applying multiple increments
		setValue(func(prev int) int { return prev + 3 })
	})
	return Div(Attrs{"id": "react-batch"},
		P(Attrs{"id": "react-batch-value"}, Text(fmt.Sprintf("Batch Value: %d", value()))),
		P(Attrs{"id": "react-batch-renders"}, Text(fmt.Sprintf("Batch Renders: %d", reactBatchRenders))),
		Button(Attrs{"id": "react-batch-btn", "onclick": batch}, Text("Batch +3")),
	)
}

// ReactivityDemo aggregates reactivity test components
func ReactivityDemo(props Attrs) *Element {
	return Div(Attrs{"id": "reactivity-demo", "class": "mt-8"},
		H2(nil, Text("Reactivity Demo")),
		&Element{Type: ReactA},
		&Element{Type: ReactB},
		&Element{Type: ReactBatch},
	)
}

func StateStressDemo(props Attrs) *Element {
	count, setCount := UseState(0)
	stressRenders++

	applyBurst := func(n int) {
		for i := 0; i < n; i++ {
			setCount(func(prev int) int { return prev + 1 })
		}
	}

	plus5 := GoUseFunc(func() { applyBurst(5) })
	plus25 := GoUseFunc(func() { applyBurst(25) })
	plus100 := GoUseFunc(func() { applyBurst(100) })
	reset := GoUseFunc(func() {
		setCount(0)
	})

	return Div(Attrs{"id": "state-stress-demo", "class": "mt-8"},
		H2(nil, Text("State Stress Demo")),
		P(Attrs{"id": "stress-count"}, Text(fmt.Sprintf("Stress Count: %d", count()))),
		P(Attrs{"id": "stress-renders"}, Text(fmt.Sprintf("Stress Renders: %d", stressRenders))),
		Div(Attrs{"class": "flex gap-2"},
			Button(Attrs{"id": "stress-plus-5", "onclick": plus5}, Text("+5")),
			Button(Attrs{"id": "stress-plus-25", "onclick": plus25}, Text("+25")),
			Button(Attrs{"id": "stress-plus-100", "onclick": plus100}, Text("+100")),
			Button(Attrs{"id": "stress-reset", "onclick": reset}, Text("Reset")),
		),
	)
}

func MixedStateBurstMirror(props Attrs) *Element {
	shared := state.UseAtom("stressSharedCounter", 0)
	mixedStressMirrorRenders++

	return Div(Attrs{"id": "mixed-stress-mirror"},
		P(Attrs{"id": "mixed-shared-mirror"}, Text(fmt.Sprintf("Mirror Shared: %d", shared.Get()))),
		P(Attrs{"id": "mixed-shared-mirror-renders"}, Text(fmt.Sprintf("Mirror Renders: %d", mixedStressMirrorRenders))),
	)
}

func MixedStateBurstDemo(props Attrs) *Element {
	local, setLocal := UseState(0)
	shared := state.UseAtom("stressSharedCounter", 0)
	mixedStressRenders++

	applyMixedBurst := func(n int) {
		for i := 0; i < n; i++ {
			setLocal(func(prev int) int { return prev + 1 })
			shared.Set(shared.Get() + 1)
		}
	}

	burst50 := GoUseFunc(func() { applyMixedBurst(50) })
	burst100 := GoUseFunc(func() { applyMixedBurst(100) })
	reset := GoUseFunc(func() {
		setLocal(0)
		shared.Set(0)
	})

	return Div(Attrs{"id": "mixed-state-stress-demo", "class": "mt-8"},
		H2(nil, Text("Mixed State Stress Demo")),
		P(Attrs{"id": "mixed-local-count"}, Text(fmt.Sprintf("Local Count: %d", local()))),
		P(Attrs{"id": "mixed-shared-count"}, Text(fmt.Sprintf("Shared Count: %d", shared.Get()))),
		P(Attrs{"id": "mixed-stress-renders"}, Text(fmt.Sprintf("Mixed Renders: %d", mixedStressRenders))),
		Div(Attrs{"class": "flex gap-2"},
			Button(Attrs{"id": "mixed-burst-50", "onclick": burst50}, Text("Mixed +50")),
			Button(Attrs{"id": "mixed-burst-100", "onclick": burst100}, Text("Mixed +100")),
			Button(Attrs{"id": "mixed-reset", "onclick": reset}, Text("Mixed Reset")),
		),
		&Element{Type: MixedStateBurstMirror},
	)
}

func FineGrainedStaticPanel(props Attrs) *Element {
	fineGrainedStaticRenders++
	return Div(Attrs{"id": "fg-static-panel"},
		P(Attrs{"id": "fg-static-renders"}, Text(fmt.Sprintf("Static Renders: %d", fineGrainedStaticRenders))),
		P(nil,
			Span(Attrs{"id": "fg-static-strong"}, Text("Static sibling")),
		),
	)
}

func FineGrainedDemo(props Attrs) *Element {
	rt := runtime.GetGlobalRuntime()
	if _, ok := rt.GetAtomValue("fgLeft"); !ok {
		_ = rt.SetAtomValue("fgLeft", 1)
	}
	if _, ok := rt.GetAtomValue("fgRight"); !ok {
		_ = rt.SetAtomValue("fgRight", 8)
	}
	if _, ok := rt.GetAtomValue("fgModel"); !ok {
		_ = rt.SetAtomValue("fgModel", fineGrainedModel{Hot: 1})
	}
	_ = rt.RegisterDerivedAtom("fgParity", []string{"fgModel"}, func() interface{} {
		value, _ := rt.GetAtomValue("fgModel")
		model, _ := value.(fineGrainedModel)
		if model.Hot%2 == 0 {
			return "even"
		}
		return "odd"
	})
	label, setLabel := UseState("ready")
	fineGrainedParentRenders++

	incLeft := GoUseFunc(func() {
		value, _ := rt.GetAtomValue("fgLeft")
		current, _ := value.(int)
		_ = rt.SetAtomValue("fgLeft", current+1)
	})
	incRight := GoUseFunc(func() {
		value, _ := rt.GetAtomValue("fgRight")
		current, _ := value.(int)
		_ = rt.SetAtomValue("fgRight", current+1)
	})
	rerenderParent := GoUseFunc(func() {
		setLabel(func(prev string) string {
			if prev == "ready" {
				return "updated"
			}
			return "ready"
		})
	})
	selectorSame := GoUseFunc(func() {
		value, _ := rt.GetAtomValue("fgModel")
		current, _ := value.(fineGrainedModel)
		current.Hot += 2
		_ = rt.SetAtomValue("fgModel", current)
	})
	selectorChange := GoUseFunc(func() {
		value, _ := rt.GetAtomValue("fgModel")
		current, _ := value.(fineGrainedModel)
		current.Hot += 1
		_ = rt.SetAtomValue("fgModel", current)
	})

	return Div(Attrs{"id": "fine-grained-demo", "class": "mt-8"},
		H2(nil, Text("Fine-Grained Reactivity Demo")),
		P(Attrs{"id": "fg-parent-renders"}, Text(fmt.Sprintf("Parent Renders: %d", fineGrainedParentRenders))),
		P(Attrs{"id": "fg-parent-label"}, Text(fmt.Sprintf("Parent Label: %s", label()))),
		Div(Attrs{"class": "flex gap-4"},
			Div(Attrs{"id": "fg-left-region"},
				P(nil, Text("Left region")),
				Span(Attrs{"id": "fg-left-value"},
					&Element{Type: runtime.ReactiveTextNodeType, Props: map[string]interface{}{
						"__gwc_reactive_text_atom_id": "fgLeft",
						"__gwc_reactive_text_getter": func() string {
							value, _ := rt.GetAtomValue("fgLeft")
							current, _ := value.(int)
							return fmt.Sprintf("%d", current)
						},
					}},
				),
				Button(Attrs{"id": "fg-left-inc", "onclick": incLeft, "class": "ml-2 px-3 py-1 bg-blue-500 text-white"}, Text("Inc Left")),
			),
			Div(Attrs{"id": "fg-right-region"},
				P(nil, Text("Right region")),
				Span(Attrs{"id": "fg-right-value"},
					&Element{Type: runtime.ReactiveTextNodeType, Props: map[string]interface{}{
						"__gwc_reactive_text_atom_id": "fgRight",
						"__gwc_reactive_text_getter": func() string {
							value, _ := rt.GetAtomValue("fgRight")
							current, _ := value.(int)
							return fmt.Sprintf("%d", current)
						},
					}},
				),
				Button(Attrs{"id": "fg-right-inc", "onclick": incRight, "class": "ml-2 px-3 py-1 bg-blue-500 text-white"}, Text("Inc Right")),
			),
		),
		Div(Attrs{"id": "fg-selector-region", "class": "mt-4"},
			P(nil, Text("Selector projection")),
			Span(Attrs{"id": "fg-selector-value"},
				&Element{Type: runtime.ReactiveTextNodeType, Props: map[string]interface{}{
					"__gwc_reactive_text_atom_id": "fgParity",
					"__gwc_reactive_text_getter": func() string {
						value, _ := rt.GetAtomValue("fgParity")
						current, _ := value.(string)
						return current
					},
				}},
			),
			Button(Attrs{"id": "fg-selector-same", "onclick": selectorSame, "class": "ml-2 px-3 py-1 bg-slate-600 text-white"}, Text("Keep Projection")),
			Button(Attrs{"id": "fg-selector-change", "onclick": selectorChange, "class": "ml-2 px-3 py-1 bg-slate-600 text-white"}, Text("Change Projection")),
		),
		Div(Attrs{"class": "mt-4"},
			Button(Attrs{"id": "fg-parent-rerender", "onclick": rerenderParent, "class": "px-3 py-1 bg-green-600 text-white"}, Text("Rerender Parent")),
		),
		&Element{Type: FineGrainedStaticPanel},
	)
}

// EffectChild demonstrates UseEffect cleanup on unmount
func EffectChild(props Attrs) *Element {
	// Use an atom to track lifecycle status so cleanup can update a node outside the child
	cleanupStatus := state.UseAtom("cleanupStatus", "")
	UseEffect(func() func() {
		// On mount: update global cleanup status and DOM directly
		cleanupStatus.Set("mounted")
		fmt.Println("EffectChild mounted")
		js.Global().Get("document").Call("querySelector", "#cleanup-status").Set("textContent", "mounted")
		return func() {
			// On unmount: update and log
			cleanupStatus.Set("cleaned")
			fmt.Println("EffectChild cleaned up")
			js.Global().Get("document").Call("querySelector", "#cleanup-status").Set("textContent", "cleaned")
		}
	}, []interface{}{})
	return Div(Attrs{"id": "effect-child"},
		P(Attrs{"id": "effect-child-text"}, Text("Effect Child")),
	)
}

// Toggle demo to mount and unmount EffectChild
func ToggleEffectDemo(props Attrs) *Element {
	show, setShow := UseState(false)
	toggle := GoUseFunc(func() {
		setShow(func(prev bool) bool { return !prev })
	})
	if show() {
		return Div(Attrs{"id": "toggle-effect-demo"},
			Button(Attrs{"id": "toggle-child-btn", "onclick": toggle}, Text("Toggle Child")),
			&Element{Type: EffectChild},
		)
	}
	return Div(Attrs{"id": "toggle-effect-demo"},
		Button(Attrs{"id": "toggle-child-btn", "onclick": toggle}, Text("Toggle Child")),
	)
}

// HelloWorld component demonstrates basic usage
func HelloWorld(props Attrs) *Element {
	count, setCount := UseState(0)
	// Setup shared atom for demonstration/testing
	sharedCounter := state.UseAtom("sharedCounter", 0)
	// Input state for onchange test
	inputValue, setInputValue := UseState("")
	// Submit state for form test
	submitValue, setSubmitValue := UseState("")

	// UseEffect to log on mount and count changes (also set title once)
	UseEffect(func() func() {
		fmt.Printf("UseEffect ran: count is %d\n", count())
		// Set document title on mount
		js.Global().Get("document").Set("title", "GoWebComponents App")
		return nil // No cleanup needed for this simple example
	}, count())

	// UseMemo to compute expensive value (for testing)
	doubledCount := UseMemo(func() int {
		result := count() * 2
		fmt.Printf("UseMemo computing: count=%d\n", count())
		return result
	}, count())

	memoReloadCount := UseMemo(func() int {
		global, err := interop.GlobalThis()
		if err != nil {
			return 1
		}
		runs := 0
		if value := global.Get("__memoReloadRuns"); value.Present() {
			runs = value.Int()
		}
		runs++
		global.Set("__memoReloadRuns", runs)
		return runs
	}, "memo-reload-boundary")
	memoReloadRuns := 0
	if global, err := interop.GlobalThis(); err == nil {
		if value := global.Get("__memoReloadRuns"); value.Present() {
			memoReloadRuns = value.Int()
		}
	}

	// cleanup-status will be updated by child effect directly via Document API

	// Create increment handler using functional setState
	increment := GoUseFunc(func() {
		setCount(func(prev int) int {
			return prev + 1
		})
	})

	// Atom increment handler
	atomIncrement := GoUseFunc(func() {
		sharedCounter.Set(sharedCounter.Get() + 1)
	})

	// Input onchange handler
	handleInputChange := GoUseFunc(func(value string) {
		setInputValue(value)
	})

	// Form onsubmit handler with preventDefault
	handleSubmit := GoUseFunc(func(event js.Value) {
		event.Call("preventDefault")
		// Get the form input value
		formInput := event.Get("target").Call("querySelector", "#form-input")
		value := formInput.Get("value").String()
		setSubmitValue(value)
	})

	return Div(Attrs{"class": "container mx-auto p-8"},
		H1(Attrs{"class": "text-4xl font-bold mb-4", "id": "main-heading"},
			Text("GoWebComponents Test"),
		),
		P(Attrs{"class": "mb-4", "data-testid": "count-display"},
			Text(fmt.Sprintf("Count: %d", count())),
		),
		P(Attrs{"class": "mb-4", "id": "doubled", "style": "font-weight: bold;"},
			Text(fmt.Sprintf("Doubled: %d", doubledCount)),
		),
		P(Attrs{"class": "mb-4", "id": "memo-value"},
			Text(fmt.Sprintf("Memo Value: %d", memoReloadCount)),
		),
		P(Attrs{"class": "mb-4", "id": "memo-runs"},
			Text(fmt.Sprintf("Memo Runs: %d", memoReloadRuns)),
		),
		Button(Attrs{
			"onclick":    increment,
			"class":      "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600",
			"aria-label": "Increment main",
		},
			Text("Increment"),
		),
		Div(nil,
			P(Attrs{"id": "atom-value-a"}, Text(fmt.Sprintf("AtomA: %d", sharedCounter.Get()))),
			P(Attrs{"id": "atom-value-b"}, Text(fmt.Sprintf("AtomB: %d", sharedCounter.Get()))),
			Button(Attrs{"id": "atom-increment", "onclick": atomIncrement}, Text("Atom Increment")),
		),
		Div(Attrs{"class": "mt-4"},
			H2(nil, Text("Input Test")),
			Input(Attrs{
				"id":      "test-input",
				"type":    "text",
				"oninput": handleInputChange,
				"class":   "border p-2",
			}),
			P(Attrs{"id": "input-value"},
				Text(fmt.Sprintf("Input: %s", inputValue())),
			),
		),
		Div(Attrs{"class": "mt-4"},
			H2(nil, Text("Form Test")),
			Form(Attrs{
				"id":       "test-form",
				"onsubmit": handleSubmit,
			},
				Input(Attrs{
					"id":    "form-input",
					"type":  "text",
					"class": "border p-2",
				}),
				Button(Attrs{
					"type":  "submit",
					"class": "ml-2 px-4 py-2 bg-green-500 text-white",
				}, Text("Submit")),
			),
			P(Attrs{"id": "submit-value"},
				Text(fmt.Sprintf("Submitted: %s", submitValue())),
			),
		),
		Div(Attrs{"class": "mt-4", "id": "reusable-components", "role": "region", "aria-label": "Reusable Counters Section"},
			H2(nil, Text("Reusable Components")),
			&Element{Type: Counter, Props: Attrs{"id": "A"}},
			&Element{Type: Counter, Props: Attrs{"id": "B"}},
			&Element{Type: Counter, Props: Attrs{"id": "C"}},
		),
		Div(Attrs{"role": "region", "aria-label": "Reactivity Demo Section"},
			&Element{Type: ReactivityDemo},
		),
		Div(Attrs{"role": "region", "aria-label": "Fine Grained Demo Section"},
			&Element{Type: FineGrainedDemo},
		),
		Div(Attrs{"role": "region", "aria-label": "State Stress Section"},
			&Element{Type: StateStressDemo},
			&Element{Type: MixedStateBurstDemo},
		),
		Div(Attrs{"role": "region", "aria-label": "Todo App Section", "id": "todo-app"},
			&Element{Type: Header},
			&Element{Type: TodoList},
		),
		Div(Attrs{"class": "mt-4"},
			&Element{Type: ToggleEffectDemo},
		),
		P(Attrs{"id": "cleanup-status"}, Text("")),
		// Add new hook tests
		&Element{Type: UseIdTestComponent},
		&Element{Type: UseFetchTestComponent},
		// Add component that intentionally uses invalid props to ensure graceful handling
		&Element{Type: BadProps},
	)
}

// BadProps passes intentionally invalid properties to test error handling
func BadProps(props Attrs) *Element {
	// class attribute as non-string, onclick as non-function to simulate invalid props
	return Div(Attrs{"id": "bad-props", "class": 12345, "onclick": "not-a-function"},
		Text("BadProps"),
	)
}

// UseIdTestComponent demonstrates UseId hook for accessibility
func UseIdTestComponent(props Attrs) *Element {
	inputId := UseId()
	selectId := UseId()
	checkboxId := UseId()

	return Div(Attrs{"id": "use-id-test", "class": "mt-8"},
		H2(nil, Text("UseId Test")),
		Div(Attrs{"class": "mb-4"},
			Label(Attrs{
				"htmlFor": inputId,
				"id":      "input-label",
				"class":   "block mb-2",
			}, Text("Test Input:")),
			Input(Attrs{
				"id":    inputId,
				"type":  "text",
				"class": "border p-2",
			}),
			P(Attrs{"id": "input-id-display"},
				Text(fmt.Sprintf("Input ID: %s", inputId)),
			),
		),
		Div(Attrs{"class": "mb-4"},
			Label(Attrs{
				"htmlFor": selectId,
				"id":      "select-label",
				"class":   "block mb-2",
			}, Text("Test Select:")),
			Select(Attrs{
				"id": selectId,
			},
				Option(Attrs{"value": "1"}, Text("Option 1")),
				Option(Attrs{"value": "2"}, Text("Option 2")),
			),
			P(Attrs{"id": "select-id-display"},
				Text(fmt.Sprintf("Select ID: %s", selectId)),
			),
		),
		Div(Attrs{"class": "mb-4"},
			Div(nil,
				Input(Attrs{
					"id":   checkboxId,
					"type": "checkbox",
				}),
				Label(Attrs{
					"htmlFor": checkboxId,
					"id":      "checkbox-label",
					"class":   "ml-2",
				}, Text("Test Checkbox")),
			),
			P(Attrs{"id": "checkbox-id-display"},
				Text(fmt.Sprintf("Checkbox ID: %s", checkboxId)),
			),
		),
	)
}

// UseFetchTestComponent demonstrates UseFetch hook
func UseFetchTestComponent(props Attrs) *Element {
	// Mock data URL - in real tests, this would be a test server endpoint
	userState, refetchUser := UseFetch("/api/user/123")

	// Use GoUseFunc hook for cleaner event handler
	fetchHandler := GoUseFunc(func() {
		refetchUser()
	})

	state := userState()

	// Render based on fetch state
	var content *Element
	if state.Loading {
		content = P(Attrs{"id": "fetch-loading"}, Text("Loading..."))
	} else if state.Error != "" {
		content = P(Attrs{
			"id":    "fetch-error",
			"style": "color: red;",
		}, Text(fmt.Sprintf("Error: %s", state.Error)))
	} else if state.Data != nil {
		content = P(Attrs{"id": "fetch-data"},
			Text(fmt.Sprintf("Data: %v", state.Data)),
		)
	} else {
		content = P(Attrs{"id": "fetch-idle"},
			Text("No data fetched yet"),
		)
	}

	return Div(Attrs{"id": "use-fetch-test", "class": "mt-8"},
		H2(nil, Text("UseFetch Test")),
		Button(Attrs{
			"id":      "fetch-button",
			"onclick": fetchHandler,
			"class":   "px-4 py-2 bg-blue-500 text-white",
		}, Text("Fetch User Data")),
		Div(Attrs{"class": "mt-4"},
			content,
		),
		P(Attrs{"id": "fetch-state-display"},
			Text(fmt.Sprintf("State: Loading=%v, Error=%s", state.Loading, state.Error)),
		),
	)
}

func main() {
	fmt.Println("🚀 GoWebComponents WASM initialized")
	hotreload.Enable()

	// Create root element that will call HelloWorld during render
	app := &Element{
		Type:  HelloWorld,
		Props: make(map[string]interface{}),
	}

	fmt.Println("About to call To...")
	To(app, "#app")
	fmt.Println("To completed")

	// Keep the Go program running
	select {}
}
