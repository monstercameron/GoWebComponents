//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/v4/examples/internal/exampleboot"
	_ "github.com/monstercameron/GoWebComponents/v4/examples/internal/examplelog"
	"github.com/monstercameron/GoWebComponents/v4/examples/shared"
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
	"github.com/monstercameron/GoWebComponents/v4/utils"
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

func reduceReplyState(parseState replyState, parseAction replyAction) replyState {
	switch parseAction.Type {
	case actionLoadThread:
		parseReviewers := 0
		parseEscalation := "No escalation required"
		if parseAction.Thread.NeedsReview {
			parseReviewers = 1
			parseEscalation = "Legal or policy reviewer queued"
		}
		return replyState{
			ActiveThread:      parseAction.Thread,
			Stage:             stageDrafting,
			ReviewRequired:    parseAction.Thread.NeedsReview,
			ReviewerCount:     parseReviewers,
			QueueStatus:       "Working locally",
			LastAction:        "Loaded " + parseAction.Thread.Key + " thread",
			RefinementCount:   0,
			EscalationSummary: parseEscalation,
		}
	case actionRefineReply:
		parseNext := parseState
		parseNext.RefinementCount++
		parseNext.LastAction = fmt.Sprintf("Refined reply draft (%d)", parseNext.RefinementCount)
		if parseNext.ReviewRequired {
			parseNext.Stage = stageReview
			parseNext.QueueStatus = "Waiting for reviewer sign-off"
			parseNext.EscalationSummary = "Reviewer sees the latest policy context"
		} else {
			parseNext.Stage = stageDrafting
			parseNext.QueueStatus = "Working locally"
		}
		return parseNext
	case actionRequestReview:
		parseNext2 := parseState
		parseNext2.Stage = stageReview
		parseNext2.ReviewRequired = true
		if parseNext2.ReviewerCount == 0 {
			parseNext2.ReviewerCount = 1
		}
		parseNext2.QueueStatus = "Waiting for reviewer sign-off"
		parseNext2.LastAction = "Routed to review"
		parseNext2.EscalationSummary = "One reviewer must approve before queueing"
		return parseNext2
	case actionApprove:
		parseNext3 := parseState
		parseNext3.Stage = stageReady
		parseNext3.ReviewRequired = false
		if parseNext3.ReviewerCount == 0 {
			parseNext3.ReviewerCount = 1
		}
		parseNext3.QueueStatus = "Ready for queue"
		parseNext3.LastAction = "Reviewer approved the draft"
		parseNext3.EscalationSummary = "Approval captured; reply can be queued"
		return parseNext3
	case actionQueue:
		if parseState.ReviewRequired {
			parseNext4 := parseState
			parseNext4.Stage = stageReview
			parseNext4.QueueStatus = "Blocked until review completes"
			parseNext4.LastAction = "Queue attempt blocked"
			return parseNext4
		}
		parseNext5 := parseState
		parseNext5.Stage = stageQueued
		parseNext5.QueueStatus = "Queued for send"
		parseNext5.LastAction = "Queued the reply"
		return parseNext5
	case actionReset:
		return initialReplyState()
	default:
		return parseState
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
	parseWorkflow := ui.UseReducer(reduceReplyState, initialReplyState())
	parseLoadThread := func(parseThread supportThread) ui.Handler {
		return ui.UseEvent(func() {
			parseWorkflow.Dispatch(replyAction{Type: actionLoadThread, Thread: parseThread})
		})
	}

	return supportReplyWorkflow{
		State:             parseWorkflow.Get(),
		UseBillingThread:  parseLoadThread(threadOptions[0]),
		UseShippingThread: parseLoadThread(threadOptions[1]),
		UsePolicyThread:   parseLoadThread(threadOptions[2]),
		RefineReply: ui.UseEvent(func() {
			parseWorkflow.Dispatch(replyAction{Type: actionRefineReply})
		}),
		RequestReview: ui.UseEvent(func() {
			parseWorkflow.Dispatch(replyAction{Type: actionRequestReview})
		}),
		ApproveReply: ui.UseEvent(func() {
			parseWorkflow.Dispatch(replyAction{Type: actionApprove})
		}),
		QueueReply: ui.UseEvent(func() {
			parseWorkflow.Dispatch(replyAction{Type: actionQueue})
		}),
		ResetWorkflow: ui.UseEvent(func() {
			parseWorkflow.Dispatch(replyAction{Type: actionReset})
		}),
	}
}

func threadButton(parseLabel string, isActive bool, parseHandler ui.Handler) ui.Node {
	parseClassName := "rounded-full border px-4 py-2 text-sm font-semibold transition "
	if isActive {
		parseClassName += "border-cyan-300 bg-cyan-300/15 text-cyan-100"
	} else {
		parseClassName += "border-white/10 bg-slate-950/45 text-slate-300 hover:border-cyan-900 hover:text-cyan-100"
	}
	return html.Button(html.Props{OnClick: parseHandler, Class: parseClassName}, html.Text(parseLabel))
}

func noteCard(parseLabel, parseValue string) ui.Node {
	return html.Div(html.Props{Class: "rounded-2xl border border-white/10 bg-slate-950/45 p-4"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-400"}, html.Text(parseLabel)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-200"}, html.Text(parseValue)),
	)
}

func reducerScalingExample() ui.Node {
	parseWorkflow := useSupportReplyWorkflow()
	parseState := parseWorkflow.State

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
				threadButton("Billing", parseState.ActiveThread.Key == threadOptions[0].Key, parseWorkflow.UseBillingThread),
				threadButton("Shipping", parseState.ActiveThread.Key == threadOptions[1].Key, parseWorkflow.UseShippingThread),
				threadButton("Policy", parseState.ActiveThread.Key == threadOptions[2].Key, parseWorkflow.UsePolicyThread),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4"},
				shared.ExampleStat("Active thread", parseState.ActiveThread.Label),
				shared.ExampleStat("Stage", string(parseState.Stage)),
				shared.ExampleStat("Review required", map[bool]string{true: "Yes", false: "No"}[parseState.ReviewRequired]),
				shared.ExampleStat("Reviewers", fmt.Sprintf("%d", parseState.ReviewerCount)),
			),
			html.Div(html.Props{Class: "mt-4 grid gap-4 md:grid-cols-2"},
				noteCard("Suggested reply", parseState.ActiveThread.SuggestedReply),
				noteCard("Workflow summary", parseState.EscalationSummary),
			),
			html.Div(html.Props{Class: "mt-6 grid gap-4 md:grid-cols-3"},
				shared.ExampleStat("Queue status", parseState.QueueStatus),
				shared.ExampleStat("Refinements", fmt.Sprintf("%d", parseState.RefinementCount)),
				shared.ExampleStat("Last action", parseState.LastAction),
			),
			html.Div(html.Props{Class: "mt-6 flex flex-wrap gap-3"},
				shared.ExampleButton("Refine reply", parseWorkflow.RefineReply),
				shared.ExampleButton("Request review", parseWorkflow.RequestReview),
				shared.ExampleButton("Approve", parseWorkflow.ApproveReply),
				shared.ExampleButton("Queue reply", parseWorkflow.QueueReply),
				shared.ExampleButton("Reset", parseWorkflow.ResetWorkflow),
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
	exampleboot.RenderExampleRoot(ui.CreateElement(reducerScalingExample))
	exampleboot.WaitExampleRuntime()
}
