//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	_ "github.com/monstercameron/GoWebComponents/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/examples/shared"
	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/utils"
)

type replyStage string

const (
	stageDrafting replyStage = "Drafting"
	stageReview   replyStage = "Review"
	stageReady    replyStage = "Ready"
	stageQueued   replyStage = "Queued"
)

type supportThread struct {
	Key            string
	Label          string
	SuggestedReply string
	NeedsReview    bool
}

var threadOptions = []supportThread{
	{
		Key:            "billing",
		Label:          "Billing: duplicate charge",
		SuggestedReply: "Thanks for flagging this. I found the duplicate authorization and can confirm it will settle back to the original card within 3 business days.",
		NeedsReview:    false,
	},
	{
		Key:            "shipping",
		Label:          "Shipping: late replacement",
		SuggestedReply: "I have started a replacement shipment and upgraded it to priority delivery. I will send the tracking number in a separate follow-up as soon as the carrier scans it in.",
		NeedsReview:    false,
	},
	{
		Key:            "policy",
		Label:          "Policy: refund exception",
		SuggestedReply: "This request falls outside the automatic refund policy, so I have prepared the context and escalated it for approval before we commit to an exception.",
		NeedsReview:    true,
	},
}

type replyState struct {
	ActiveThread      supportThread
	Stage             replyStage
	ReviewRequired    bool
	ReviewerCount     int
	QueueStatus       string
	LastAction        string
	RefinementCount   int
	EscalationSummary string
}

type replyActionType string

const (
	actionLoadThread    replyActionType = "load_thread"
	actionRefineReply   replyActionType = "refine_reply"
	actionRequestReview replyActionType = "request_review"
	actionApprove       replyActionType = "approve"
	actionQueue         replyActionType = "queue"
	actionReset         replyActionType = "reset"
)

type replyAction struct {
	Type   replyActionType
	Thread supportThread
}

func initialReplyState() replyState {
	return replyState{
		ActiveThread:      threadOptions[0],
		Stage:             stageDrafting,
		ReviewRequired:    false,
		ReviewerCount:     0,
		QueueStatus:       "Working locally",
		LastAction:        "Loaded billing thread",
		RefinementCount:   0,
		EscalationSummary: "No escalation required",
	}
}

func reduceReplyState(state replyState, action replyAction) replyState {
	switch action.Type {
	case actionLoadThread:
		reviewers := 0
		escalation := "No escalation required"
		if action.Thread.NeedsReview {
			reviewers = 1
			escalation = "Legal or policy reviewer queued"
		}
		return replyState{
			ActiveThread:      action.Thread,
			Stage:             stageDrafting,
			ReviewRequired:    action.Thread.NeedsReview,
			ReviewerCount:     reviewers,
			QueueStatus:       "Working locally",
			LastAction:        "Loaded " + action.Thread.Key + " thread",
			RefinementCount:   0,
			EscalationSummary: escalation,
		}
	case actionRefineReply:
		next := state
		next.RefinementCount++
		next.LastAction = fmt.Sprintf("Refined reply draft (%d)", next.RefinementCount)
		if next.ReviewRequired {
			next.Stage = stageReview
			next.QueueStatus = "Waiting for reviewer sign-off"
			next.EscalationSummary = "Reviewer sees the latest policy context"
		} else {
			next.Stage = stageDrafting
			next.QueueStatus = "Working locally"
		}
		return next
	case actionRequestReview:
		next := state
		next.Stage = stageReview
		next.ReviewRequired = true
		if next.ReviewerCount == 0 {
			next.ReviewerCount = 1
		}
		next.QueueStatus = "Waiting for reviewer sign-off"
		next.LastAction = "Routed to review"
		next.EscalationSummary = "One reviewer must approve before queueing"
		return next
	case actionApprove:
		next := state
		next.Stage = stageReady
		next.ReviewRequired = false
		if next.ReviewerCount == 0 {
			next.ReviewerCount = 1
		}
		next.QueueStatus = "Ready for queue"
		next.LastAction = "Reviewer approved the draft"
		next.EscalationSummary = "Approval captured; reply can be queued"
		return next
	case actionQueue:
		if state.ReviewRequired {
			next := state
			next.Stage = stageReview
			next.QueueStatus = "Blocked until review completes"
			next.LastAction = "Queue attempt blocked"
			return next
		}
		next := state
		next.Stage = stageQueued
		next.QueueStatus = "Queued for send"
		next.LastAction = "Queued the reply"
		return next
	case actionReset:
		return initialReplyState()
	default:
		return state
	}
}

type supportReplyWorkflow struct {
	State             replyState
	UseBillingThread  ui.Handler
	UseShippingThread ui.Handler
	UsePolicyThread   ui.Handler
	RefineReply       ui.Handler
	RequestReview     ui.Handler
	ApproveReply      ui.Handler
	QueueReply        ui.Handler
	ResetWorkflow     ui.Handler
}

func useSupportReplyWorkflow() supportReplyWorkflow {
	workflow := ui.UseReducer(reduceReplyState, initialReplyState())
	loadThread := func(thread supportThread) ui.Handler {
		return ui.UseEvent(func() {
			workflow.Dispatch(replyAction{Type: actionLoadThread, Thread: thread})
		})
	}

	return supportReplyWorkflow{
		State:             workflow.Get(),
		UseBillingThread:  loadThread(threadOptions[0]),
		UseShippingThread: loadThread(threadOptions[1]),
		UsePolicyThread:   loadThread(threadOptions[2]),
		RefineReply: ui.UseEvent(func() {
			workflow.Dispatch(replyAction{Type: actionRefineReply})
		}),
		RequestReview: ui.UseEvent(func() {
			workflow.Dispatch(replyAction{Type: actionRequestReview})
		}),
		ApproveReply: ui.UseEvent(func() {
			workflow.Dispatch(replyAction{Type: actionApprove})
		}),
		QueueReply: ui.UseEvent(func() {
			workflow.Dispatch(replyAction{Type: actionQueue})
		}),
		ResetWorkflow: ui.UseEvent(func() {
			workflow.Dispatch(replyAction{Type: actionReset})
		}),
	}
}

func threadButton(label string, active bool, handler ui.Handler) ui.Node {
	className := "rounded-full border px-4 py-2 text-sm font-semibold transition "
	if active {
		className += "border-cyan-300 bg-cyan-300/15 text-cyan-100"
	} else {
		className += "border-white/10 bg-slate-950/45 text-slate-300 hover:border-cyan-900 hover:text-cyan-100"
	}
	return html.Button(html.Props{OnClick: handler, Class: className}, html.Text(label))
}

func noteCard(label, value string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(label)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text(value)),
	)
}

func reducerScalingExample() ui.Node {
	workflow := useSupportReplyWorkflow()
	state := workflow.State

	return shared.ExamplePage(
		"Scaling local state with UseReducer",
		"ui.UseReducer",
		"This page wraps ui.UseReducer in an app-specific hook so the renderer reads like a workflow surface instead of a list of unrelated field updates.",
		shared.ExamplePanel("Why reducers help here",
			html.P(html.Props{Class: "mt-3 text-slate-300 leading-7"}, html.Text("Selecting a thread, routing it to review, approving it, and queueing it all move multiple fields together. The reducer owns those transitions so the component just calls semantic operations.")),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				noteCard("State that changes together", "Thread selection, review status, escalation notes, queue status, and the last visible action all move in lockstep."),
				noteCard("App-specific hook surface", "The page calls UsePolicyThread, RequestReview, ApproveReply, and QueueReply instead of dispatching raw string actions inline everywhere."),
				noteCard("When this is worth it", "Use this shape when one feature has several valid transitions and you want one place to name, test, and review them."),
			),
		),
		shared.ExamplePanel("Reply workflow",
			html.Div(html.Props{Class: "mt-3 flex flex-wrap gap-3"},
				threadButton("Billing", state.ActiveThread.Key == threadOptions[0].Key, workflow.UseBillingThread),
				threadButton("Shipping", state.ActiveThread.Key == threadOptions[1].Key, workflow.UseShippingThread),
				threadButton("Policy", state.ActiveThread.Key == threadOptions[2].Key, workflow.UsePolicyThread),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4"},
				shared.ExampleStat("Active thread", state.ActiveThread.Label),
				shared.ExampleStat("Stage", string(state.Stage)),
				shared.ExampleStat("Review required", map[bool]string{true: "Yes", false: "No"}[state.ReviewRequired]),
				shared.ExampleStat("Reviewers", fmt.Sprintf("%d", state.ReviewerCount)),
			),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-2"},
				noteCard("Suggested reply", state.ActiveThread.SuggestedReply),
				noteCard("Workflow summary", state.EscalationSummary),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Queue status", state.QueueStatus),
				shared.ExampleStat("Refinements", fmt.Sprintf("%d", state.RefinementCount)),
				shared.ExampleStat("Last action", state.LastAction),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Refine reply", workflow.RefineReply),
				shared.ExampleButton("Request review", workflow.RequestReview),
				shared.ExampleButton("Approve", workflow.ApproveReply),
				shared.ExampleButton("Queue reply", workflow.QueueReply),
				shared.ExampleButton("Reset", workflow.ResetWorkflow),
			),
			shared.ExampleCode(
				"func useSupportReplyWorkflow() supportReplyWorkflow {",
				"    workflow := ui.UseReducer(reduceReplyState, initialReplyState())",
				"    return supportReplyWorkflow{",
				"        State: workflow.Get(),",
				"        RequestReview: ui.UseEvent(func() { workflow.Dispatch(replyAction{Type: actionRequestReview}) }),",
				"    }",
				"}",
			),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(reducerScalingExample), "#app")
	select {}
}
