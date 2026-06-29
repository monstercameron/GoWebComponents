//go:build js && wasm
// +build js,wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type workflowState struct {
	Step   string
	Review bool
}

type workflowAction string

const (
	actionDraft  workflowAction = "draft"
	actionReview workflowAction = "review"
	actionShip   workflowAction = "ship"
	actionReset  workflowAction = "reset"
)

func reducer(parseState workflowState, parseAction workflowAction) workflowState {
	switch parseAction {
	case actionDraft:
		return workflowState{Step: "Draft", Review: false}
	case actionReview:
		return workflowState{Step: "Review", Review: true}
	case actionShip:
		return workflowState{Step: "Shipped", Review: true}
	case actionReset:
		return workflowState{Step: "Draft", Review: false}
	default:
		return parseState
	}
}

func useReducerExample() ui.Node {
	parseWorkflow := ui.UseReducer(reducer, workflowState{Step: "Draft", Review: false})
	parseState := parseWorkflow.Get()

	return shared.ExamplePage(
		"ui.UseReducer",
		"Model local state transitions with explicit actions",
		"This example uses reducer actions for a small editorial workflow so each state transition is named instead of scattered across manual updates.",
		shared.ExamplePanel("Workflow reducer",
			html.Div(html.Props{Class: "mt-3 grid gap-4 md:grid-cols-2"},
				shared.ExampleStat("Step", parseState.Step),
				shared.ExampleStat("Review required", map[bool]string{true: "Yes", false: "No"}[parseState.Review]),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Draft", ui.UseEvent(func() { parseWorkflow.Dispatch(actionDraft) })),
				shared.ExampleButton("Review", ui.UseEvent(func() { parseWorkflow.Dispatch(actionReview) })),
				shared.ExampleButton("Ship", ui.UseEvent(func() { parseWorkflow.Dispatch(actionShip) })),
				shared.ExampleButton("Reset", ui.UseEvent(func() { parseWorkflow.Dispatch(actionReset) })),
			),
			shared.ExampleCode(
				"workflow := ui.UseReducer(reducer, workflowState{Step: \"Draft\"})",
				"workflow.Dispatch(actionReview)",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	exampleboot.RenderExampleRoot(ui.CreateElement(useReducerExample))
	exampleboot.WaitExampleRuntime()
}
