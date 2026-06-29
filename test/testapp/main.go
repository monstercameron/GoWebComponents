//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"
	"strings"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v4/hotreload"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/interop"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func shouldCrash(parseMode string) bool {
	parseSearch := js.Global().Get("location").Get("search")
	if !parseSearch.Truthy() {
		return false
	}
	parseQuery := strings.ToLower(strings.TrimSpace(parseSearch.String()))
	parseWant := strings.ToLower(strings.TrimSpace(parseMode))
	return strings.Contains(parseQuery, "crash="+parseWant)
}

// Counter is a reusable component for testing component reuse
func Counter(parseProps Attrs) *Element {
	parseId := ""
	if parseProps != nil {
		if parseIdVal, parseOk := parseProps["id"].(string); parseOk {
			parseId = parseIdVal
		}
	}

	parseCount, setCount := UseState(0)

	parseIncrement := GoUseFunc(func() {
		setCount(func(parsePrev int) int {
			return parsePrev + 1
		})
	})

	return Div(Attrs{"class": "counter-instance", "data-counter-id": parseId},
		P(Attrs{"class": "counter-value"},
			Text(fmt.Sprintf("Counter %s: %d", parseId, parseCount())),
		),
		Button(Attrs{
			"onclick": parseIncrement,
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
func ReactA(parseProps Attrs) *Element {
	parseValue, setValue := UseState(0)
	reactARenders++
	parseInc := GoUseFunc(func() {
		setValue(func(parsePrev int) int { return parsePrev + 1 })
	})
	return Div(Attrs{"id": "react-a"},
		P(Attrs{"id": "react-a-value"}, Text(fmt.Sprintf("A Value: %d", parseValue()))),
		P(Attrs{"id": "react-a-renders"}, Text(fmt.Sprintf("A Renders: %d", reactARenders))),
		Button(Attrs{"id": "react-a-inc", "onclick": parseInc}, Text("Inc A")),
	)
}

// ReactB component - should not re-render when A changes
func ReactB(parseProps Attrs) *Element {
	parseValue, _ := UseState(0) // static for this test
	reactBRenders++
	return Div(Attrs{"id": "react-b"},
		P(Attrs{"id": "react-b-value"}, Text(fmt.Sprintf("B Value: %d", parseValue()))),
		P(Attrs{"id": "react-b-renders"}, Text(fmt.Sprintf("B Renders: %d", reactBRenders))),
	)
}

// ReactBatch component - demonstrates batched logical update
func ReactBatch(parseProps Attrs) *Element {
	parseValue, setValue := UseState(0)
	reactBatchRenders++
	parseBatch := GoUseFunc(func() {
		// Single state update applying multiple increments
		setValue(func(parsePrev int) int { return parsePrev + 3 })
	})
	return Div(Attrs{"id": "react-batch"},
		P(Attrs{"id": "react-batch-value"}, Text(fmt.Sprintf("Batch Value: %d", parseValue()))),
		P(Attrs{"id": "react-batch-renders"}, Text(fmt.Sprintf("Batch Renders: %d", reactBatchRenders))),
		Button(Attrs{"id": "react-batch-btn", "onclick": parseBatch}, Text("Batch +3")),
	)
}

// ReactivityDemo aggregates reactivity test components
func ReactivityDemo(parseProps Attrs) *Element {
	return Div(Attrs{"id": "reactivity-demo", "class": "mt-8"},
		H2(nil, Text("Reactivity Demo")),
		&Element{Type: ReactA},
		&Element{Type: ReactB},
		&Element{Type: ReactBatch},
	)
}

func StateStressDemo(parseProps Attrs) *Element {
	parseCount, setCount := UseState(0)
	stressRenders++

	applyBurst := func(parseN int) {
		for parseI := 0; parseI < parseN; parseI++ {
			setCount(func(parsePrev int) int { return parsePrev + 1 })
		}
	}

	parsePlus5 := GoUseFunc(func() { applyBurst(5) })
	parsePlus25 := GoUseFunc(func() { applyBurst(25) })
	parsePlus100 := GoUseFunc(func() { applyBurst(100) })
	reset := GoUseFunc(func() {
		setCount(0)
	})

	return Div(Attrs{"id": "state-stress-demo", "class": "mt-8"},
		H2(nil, Text("State Stress Demo")),
		P(Attrs{"id": "stress-count"}, Text(fmt.Sprintf("Stress Count: %d", parseCount()))),
		P(Attrs{"id": "stress-renders"}, Text(fmt.Sprintf("Stress Renders: %d", stressRenders))),
		Div(Attrs{"class": "flex gap-2"},
			Button(Attrs{"id": "stress-plus-5", "onclick": parsePlus5}, Text("+5")),
			Button(Attrs{"id": "stress-plus-25", "onclick": parsePlus25}, Text("+25")),
			Button(Attrs{"id": "stress-plus-100", "onclick": parsePlus100}, Text("+100")),
			Button(Attrs{"id": "stress-reset", "onclick": reset}, Text("Reset")),
		),
	)
}

func MixedStateBurstMirror(parseProps Attrs) *Element {
	parseShared := state.UseAtom("stressSharedCounter", 0)
	mixedStressMirrorRenders++

	return Div(Attrs{"id": "mixed-stress-mirror"},
		P(Attrs{"id": "mixed-shared-mirror"}, Text(fmt.Sprintf("Mirror Shared: %d", parseShared.Get()))),
		P(Attrs{"id": "mixed-shared-mirror-renders"}, Text(fmt.Sprintf("Mirror Renders: %d", mixedStressMirrorRenders))),
	)
}

func MixedStateBurstDemo(parseProps Attrs) *Element {
	parseLocal, setLocal := UseState(0)
	parseShared := state.UseAtom("stressSharedCounter", 0)
	mixedStressRenders++

	applyMixedBurst := func(parseN int) {
		for parseI := 0; parseI < parseN; parseI++ {
			setLocal(func(parsePrev int) int { return parsePrev + 1 })
			parseShared.Set(parseShared.Get() + 1)
		}
	}

	parseBurst50 := GoUseFunc(func() { applyMixedBurst(50) })
	parseBurst100 := GoUseFunc(func() { applyMixedBurst(100) })
	reset := GoUseFunc(func() {
		setLocal(0)
		parseShared.Set(0)
	})

	return Div(Attrs{"id": "mixed-state-stress-demo", "class": "mt-8"},
		H2(nil, Text("Mixed State Stress Demo")),
		P(Attrs{"id": "mixed-local-count"}, Text(fmt.Sprintf("Local Count: %d", parseLocal()))),
		P(Attrs{"id": "mixed-shared-count"}, Text(fmt.Sprintf("Shared Count: %d", parseShared.Get()))),
		P(Attrs{"id": "mixed-stress-renders"}, Text(fmt.Sprintf("Mixed Renders: %d", mixedStressRenders))),
		Div(Attrs{"class": "flex gap-2"},
			Button(Attrs{"id": "mixed-burst-50", "onclick": parseBurst50}, Text("Mixed +50")),
			Button(Attrs{"id": "mixed-burst-100", "onclick": parseBurst100}, Text("Mixed +100")),
			Button(Attrs{"id": "mixed-reset", "onclick": reset}, Text("Mixed Reset")),
		),
		&Element{Type: MixedStateBurstMirror},
	)
}

func FineGrainedStaticPanel(parseProps Attrs) *Element {
	fineGrainedStaticRenders++
	return Div(Attrs{"id": "fg-static-panel"},
		P(Attrs{"id": "fg-static-renders"}, Text(fmt.Sprintf("Static Renders: %d", fineGrainedStaticRenders))),
		P(nil,
			Span(Attrs{"id": "fg-static-strong"}, Text("Static sibling")),
		),
	)
}

func FineGrainedDemo(parseProps Attrs) *Element {
	parseRt := runtime.GetGlobalRuntime()
	if _, parseOk := parseRt.GetAtomValue("fgLeft"); !parseOk {
		_ = parseRt.SetAtomValue("fgLeft", 1)
	}
	if _, parseOk2 := parseRt.GetAtomValue("fgRight"); !parseOk2 {
		_ = parseRt.SetAtomValue("fgRight", 8)
	}
	if _, parseOk3 := parseRt.GetAtomValue("fgModel"); !parseOk3 {
		_ = parseRt.SetAtomValue("fgModel", fineGrainedModel{Hot: 1})
	}
	_ = parseRt.RegisterDerivedAtom("fgParity", []string{"fgModel"}, func() interface{} {
		parseValue, _ := parseRt.GetAtomValue("fgModel")
		parseModel, _ := parseValue.(fineGrainedModel)
		if parseModel.Hot%2 == 0 {
			return "even"
		}
		return "odd"
	})
	parseLabel, setLabel := UseState("ready")
	fineGrainedParentRenders++

	parseIncLeft := GoUseFunc(func() {
		parseValue2, _ := parseRt.GetAtomValue("fgLeft")
		parseCurrent, _ := parseValue2.(int)
		_ = parseRt.SetAtomValue("fgLeft", parseCurrent+1)
	})
	parseIncRight := GoUseFunc(func() {
		parseValue3, _ := parseRt.GetAtomValue("fgRight")
		parseCurrent2, _ := parseValue3.(int)
		_ = parseRt.SetAtomValue("fgRight", parseCurrent2+1)
	})
	parseRerenderParent := GoUseFunc(func() {
		setLabel(func(parsePrev string) string {
			if parsePrev == "ready" {
				return "updated"
			}
			return "ready"
		})
	})
	parseSelectorSame := GoUseFunc(func() {
		parseValue4, _ := parseRt.GetAtomValue("fgModel")
		parseCurrent3, _ := parseValue4.(fineGrainedModel)
		parseCurrent3.Hot += 2
		_ = parseRt.SetAtomValue("fgModel", parseCurrent3)
	})
	parseSelectorChange := GoUseFunc(func() {
		parseValue5, _ := parseRt.GetAtomValue("fgModel")
		parseCurrent4, _ := parseValue5.(fineGrainedModel)
		parseCurrent4.Hot += 1
		_ = parseRt.SetAtomValue("fgModel", parseCurrent4)
	})

	return Div(Attrs{"id": "fine-grained-demo", "class": "mt-8"},
		H2(nil, Text("Fine-Grained Reactivity Demo")),
		P(Attrs{"id": "fg-parent-renders"}, Text(fmt.Sprintf("Parent Renders: %d", fineGrainedParentRenders))),
		P(Attrs{"id": "fg-parent-label"}, Text(fmt.Sprintf("Parent Label: %s", parseLabel()))),
		Div(Attrs{"class": "flex gap-4"},
			Div(Attrs{"id": "fg-left-region"},
				P(nil, Text("Left region")),
				Span(Attrs{"id": "fg-left-value"},
					&Element{Type: runtime.ReactiveTextNodeType, Props: map[string]interface{}{
						"__gwc_reactive_text_atom_id": "fgLeft",
						"__gwc_reactive_text_getter": func() string {
							parseValue6, _ := parseRt.GetAtomValue("fgLeft")
							parseCurrent5, _ := parseValue6.(int)
							return fmt.Sprintf("%d", parseCurrent5)
						},
					}},
				),
				Button(Attrs{"id": "fg-left-inc", "onclick": parseIncLeft, "class": "ml-2 px-3 py-1 bg-blue-500 text-white"}, Text("Inc Left")),
			),
			Div(Attrs{"id": "fg-right-region"},
				P(nil, Text("Right region")),
				Span(Attrs{"id": "fg-right-value"},
					&Element{Type: runtime.ReactiveTextNodeType, Props: map[string]interface{}{
						"__gwc_reactive_text_atom_id": "fgRight",
						"__gwc_reactive_text_getter": func() string {
							parseValue7, _ := parseRt.GetAtomValue("fgRight")
							parseCurrent6, _ := parseValue7.(int)
							return fmt.Sprintf("%d", parseCurrent6)
						},
					}},
				),
				Button(Attrs{"id": "fg-right-inc", "onclick": parseIncRight, "class": "ml-2 px-3 py-1 bg-blue-500 text-white"}, Text("Inc Right")),
			),
		),
		Div(Attrs{"id": "fg-selector-region", "class": "mt-4"},
			P(nil, Text("Selector projection")),
			Span(Attrs{"id": "fg-selector-value"},
				&Element{Type: runtime.ReactiveTextNodeType, Props: map[string]interface{}{
					"__gwc_reactive_text_atom_id": "fgParity",
					"__gwc_reactive_text_getter": func() string {
						parseValue8, _ := parseRt.GetAtomValue("fgParity")
						parseCurrent7, _ := parseValue8.(string)
						return parseCurrent7
					},
				}},
			),
			Button(Attrs{"id": "fg-selector-same", "onclick": parseSelectorSame, "class": "ml-2 px-3 py-1 bg-slate-600 text-white"}, Text("Keep Projection")),
			Button(Attrs{"id": "fg-selector-change", "onclick": parseSelectorChange, "class": "ml-2 px-3 py-1 bg-slate-600 text-white"}, Text("Change Projection")),
		),
		Div(Attrs{"class": "mt-4"},
			Button(Attrs{"id": "fg-parent-rerender", "onclick": parseRerenderParent, "class": "px-3 py-1 bg-green-600 text-white"}, Text("Rerender Parent")),
		),
		&Element{Type: FineGrainedStaticPanel},
	)
}

// EffectChild demonstrates UseEffect cleanup on unmount
func EffectChild(parseProps Attrs) *Element {
	// Use an atom to track lifecycle status so cleanup can update a node outside the child
	parseCleanupStatus := state.UseAtom("cleanupStatus", "")
	UseEffect(func() func() {
		// On mount: update global cleanup status and DOM directly
		parseCleanupStatus.Set("mounted")
		fmt.Println("EffectChild mounted")
		js.Global().Get("document").Call("querySelector", "#cleanup-status").Set("textContent", "mounted")
		return func() {
			// On unmount: update and log
			parseCleanupStatus.Set("cleaned")
			fmt.Println("EffectChild cleaned up")
			js.Global().Get("document").Call("querySelector", "#cleanup-status").Set("textContent", "cleaned")
		}
	}, []interface{}{})
	return Div(Attrs{"id": "effect-child"},
		P(Attrs{"id": "effect-child-text"}, Text("Effect Child")),
	)
}

// ToggleEffectDemo is a demo that mounts and unmounts EffectChild.
func ToggleEffectDemo(parseProps Attrs) *Element {
	parseShow, setShow := UseState(false)
	parseToggle := GoUseFunc(func() {
		setShow(func(isPrev bool) bool { return !isPrev })
	})
	if parseShow() {
		return Div(Attrs{"id": "toggle-effect-demo"},
			Button(Attrs{"id": "toggle-child-btn", "onclick": parseToggle}, Text("Toggle Child")),
			&Element{Type: EffectChild},
		)
	}
	return Div(Attrs{"id": "toggle-effect-demo"},
		Button(Attrs{"id": "toggle-child-btn", "onclick": parseToggle}, Text("Toggle Child")),
	)
}

func BoundaryCrashChild(parseProps Attrs) *Element {
	if shouldCrash("boundary-render") {
		panic("intentional boundary render crash for Playwright logging")
	}

	return P(Attrs{"id": "boundary-child-ok"},
		Text("Boundary child rendered without crashing"),
	)
}

func BoundaryCrashDemo(parseProps Attrs) ui.Node {
	return ui.CreateElement(ui.ErrorBoundary, ui.ErrorBoundaryProps{
		ErrorFallback: func(parseErr error, reset func()) ui.Node {
			return Div(Attrs{"id": "boundary-fallback", "data-error": parseErr.Error()},
				P(Attrs{"id": "boundary-fallback-title"}, Text("Boundary fallback rendered")),
				P(Attrs{"id": "boundary-fallback-error"}, Text(parseErr.Error())),
			)
		},
		Child: &Element{Type: BoundaryCrashChild},
	})
}

// HelloWorld component demonstrates basic usage
func HelloWorld(parseProps Attrs) *Element {
	if shouldCrash("render") {
		panic("intentional render crash for Playwright logging")
	}

	parseCount, setCount := UseState(0)
	// Setup shared atom for demonstration/testing
	parseSharedCounter := state.UseAtom("sharedCounter", 0)
	// Input state for onchange test
	parseInputValue, setInputValue := UseState("")
	// Submit state for form test
	parseSubmitValue, setSubmitValue := UseState("")

	// UseEffect to log on mount and count changes (also set title once)
	UseEffect(func() func() {
		fmt.Printf("UseEffect ran: count is %d\n", parseCount())
		// Set document title on mount
		js.Global().Get("document").Set("title", "GoWebComponents App")
		return nil // No cleanup needed for this simple example
	}, parseCount())

	// UseMemo to compute expensive value (for testing)
	parseDoubledCount := UseMemo(func() int {
		parseResult := parseCount() * 2
		fmt.Printf("UseMemo computing: count=%d\n", parseCount())
		return parseResult
	}, parseCount())

	parseMemoReloadCount := UseMemo(func() int {
		parseGlobal, parseErr := interop.GetGlobalThis()
		if parseErr != nil {
			return 1
		}
		parseRuns := 0
		if parseValue := parseGlobal.Get("__memoReloadRuns"); parseValue.Present() {
			parseRuns = parseValue.Int()
		}
		parseRuns++
		parseGlobal.Set("__memoReloadRuns", parseRuns)
		return parseRuns
	}, "memo-reload-boundary")
	parseMemoReloadRuns := 0
	if parseGlobal2, parseErr2 := interop.GetGlobalThis(); parseErr2 == nil {
		if parseValue2 := parseGlobal2.Get("__memoReloadRuns"); parseValue2.Present() {
			parseMemoReloadRuns = parseValue2.Int()
		}
	}

	// cleanup-status will be updated by child effect directly via Document API

	// Create increment handler using functional setState
	parseIncrement := GoUseFunc(func() {
		setCount(func(parsePrev int) int {
			return parsePrev + 1
		})
	})

	// Atom increment handler
	parseAtomIncrement := GoUseFunc(func() {
		parseSharedCounter.Set(parseSharedCounter.Get() + 1)
	})

	// Input onchange handler
	handleInputChange := GoUseFunc(func(parseValue4 string) {
		setInputValue(parseValue4)
	})

	// Form onsubmit handler with preventDefault
	handleSubmit := GoUseFunc(func(parseEvent js.Value) {
		parseEvent.Call("preventDefault")
		// Get the form input value
		parseFormInput := parseEvent.Get("target").Call("querySelector", "#form-input")
		parseValue3 := parseFormInput.Get("value").String()
		setSubmitValue(parseValue3)
	})

	return Div(Attrs{"class": "container mx-auto p-8"},
		H1(Attrs{"class": "text-4xl font-bold mb-4", "id": "main-heading"},
			Text("GoWebComponents Test"),
		),
		P(Attrs{"class": "mb-4", "data-testid": "count-display"},
			Text(fmt.Sprintf("Count: %d", parseCount())),
		),
		P(Attrs{"class": "mb-4", "id": "doubled", "style": "font-weight: bold;"},
			Text(fmt.Sprintf("Doubled: %d", parseDoubledCount)),
		),
		P(Attrs{"class": "mb-4", "id": "memo-value"},
			Text(fmt.Sprintf("Memo Value: %d", parseMemoReloadCount)),
		),
		P(Attrs{"class": "mb-4", "id": "memo-runs"},
			Text(fmt.Sprintf("Memo Runs: %d", parseMemoReloadRuns)),
		),
		Button(Attrs{
			"onclick":    parseIncrement,
			"class":      "px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600",
			"aria-label": "Increment main",
		},
			Text("Increment"),
		),
		Div(nil,
			P(Attrs{"id": "atom-value-a"}, Text(fmt.Sprintf("AtomA: %d", parseSharedCounter.Get()))),
			P(Attrs{"id": "atom-value-b"}, Text(fmt.Sprintf("AtomB: %d", parseSharedCounter.Get()))),
			Button(Attrs{"id": "atom-increment", "onclick": parseAtomIncrement}, Text("Atom Increment")),
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
				Text(fmt.Sprintf("Input: %s", parseInputValue())),
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
				Text(fmt.Sprintf("Submitted: %s", parseSubmitValue())),
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
		Div(Attrs{"class": "mt-4", "id": "boundary-demo"},
			H2(nil, Text("Boundary Demo")),
			BoundaryCrashDemo(nil),
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
func BadProps(parseProps Attrs) *Element {
	// class attribute as non-string, onclick as non-function to simulate invalid props
	return Div(Attrs{"id": "bad-props", "class": 12345, "onclick": "not-a-function"},
		Text("BadProps"),
	)
}

// UseIdTestComponent demonstrates UseId hook for accessibility
func UseIdTestComponent(parseProps Attrs) *Element {
	parseInputId := UseId()
	parseSelectId := UseId()
	parseCheckboxId := UseId()

	return Div(Attrs{"id": "use-id-test", "class": "mt-8"},
		H2(nil, Text("UseId Test")),
		Div(Attrs{"class": "mb-4"},
			Label(Attrs{
				"htmlFor": parseInputId,
				"id":      "input-label",
				"class":   "block mb-2",
			}, Text("Test Input:")),
			Input(Attrs{
				"id":    parseInputId,
				"type":  "text",
				"class": "border p-2",
			}),
			P(Attrs{"id": "input-id-display"},
				Text(fmt.Sprintf("Input ID: %s", parseInputId)),
			),
		),
		Div(Attrs{"class": "mb-4"},
			Label(Attrs{
				"htmlFor": parseSelectId,
				"id":      "select-label",
				"class":   "block mb-2",
			}, Text("Test Select:")),
			Select(Attrs{
				"id": parseSelectId,
			},
				Option(Attrs{"value": "1"}, Text("Option 1")),
				Option(Attrs{"value": "2"}, Text("Option 2")),
			),
			P(Attrs{"id": "select-id-display"},
				Text(fmt.Sprintf("Select ID: %s", parseSelectId)),
			),
		),
		Div(Attrs{"class": "mb-4"},
			Div(nil,
				Input(Attrs{
					"id":   parseCheckboxId,
					"type": "checkbox",
				}),
				Label(Attrs{
					"htmlFor": parseCheckboxId,
					"id":      "checkbox-label",
					"class":   "ml-2",
				}, Text("Test Checkbox")),
			),
			P(Attrs{"id": "checkbox-id-display"},
				Text(fmt.Sprintf("Checkbox ID: %s", parseCheckboxId)),
			),
		),
	)
}

// UseFetchTestComponent demonstrates UseFetch hook
func UseFetchTestComponent(parseProps Attrs) *Element {
	// Mock data URL - in real tests, this would be a test server endpoint
	parseUserState, parseRefetchUser := UseFetch("/api/user/123")

	// Use GoUseFunc hook for cleaner event handler
	parseFetchHandler := GoUseFunc(func() {
		parseRefetchUser()
	})

	parseState := parseUserState()

	// Render based on fetch state
	var parseContent *Element
	if parseState.Loading {
		parseContent = P(Attrs{"id": "fetch-loading"}, Text("Loading..."))
	} else if parseState.Error != "" {
		parseContent = P(Attrs{
			"id":    "fetch-error",
			"style": "color: red;",
		}, Text(fmt.Sprintf("Error: %s", parseState.Error)))
	} else if parseState.Data != nil {
		parseContent = P(Attrs{"id": "fetch-data"},
			Text(fmt.Sprintf("Data: %v", parseState.Data)),
		)
	} else {
		parseContent = P(Attrs{"id": "fetch-idle"},
			Text("No data fetched yet"),
		)
	}

	return Div(Attrs{"id": "use-fetch-test", "class": "mt-8"},
		H2(nil, Text("UseFetch Test")),
		Button(Attrs{
			"id":      "fetch-button",
			"onclick": parseFetchHandler,
			"class":   "px-4 py-2 bg-blue-500 text-white",
		}, Text("Fetch User Data")),
		Div(Attrs{"class": "mt-4"},
			parseContent,
		),
		P(Attrs{"id": "fetch-state-display"},
			Text(fmt.Sprintf("State: Loading=%v, Error=%s", parseState.Loading, parseState.Error)),
		),
	)
}

func main() {
	fmt.Println("🚀 GoWebComponents WASM initialized")
	hotreload.Enable()

	// Create root element that will call HelloWorld during render
	parseApp := &Element{
		Type:  HelloWorld,
		Props: make(map[string]interface{}),
	}

	fmt.Println("About to call To...")
	To(parseApp, "#app")
	fmt.Println("To completed")

	// Keep the Go program running
	select {}
}
