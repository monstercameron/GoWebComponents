//go:build js && wasm

package app

import (
	"strconv"
	"strings"

	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type chatJourneyState struct {
	parseActiveStep  int
	parseStageID     string
	parseStepLabel   string
	parseHeadingText string
	parseBodyText    string
	parsePlaceholder string
}

type parseJourneyStep struct {
	parseID    string
	parseLabel string
	parseBody  string
}

// parseBuildChatJourneyState derives one cohesive first-chat journey state from message and stream activity.
func parseBuildChatJourneyState(parseMessages []message, isParseStreaming bool) chatJourneyState {
	isParseHasUserMessage := false
	parseAssistantMessageCount := 0
	for _, parseMessage := range parseMessages {
		if parseMessage.Role == roleUser && strings.TrimSpace(parseMessage.Content) != "" {
			isParseHasUserMessage = true
		}
		if parseMessage.Role == roleAssistant && strings.TrimSpace(parseMessage.Content) != "" && !parseMessage.Pending {
			parseAssistantMessageCount++
		}
	}

	if !isParseHasUserMessage {
		return chatJourneyState{
			parseActiveStep:  5,
			parseStageID:     "compose",
			parseStepLabel:   "Step 5 of 7 - Send your first prompt",
			parseHeadingText: "Start your first chat",
			parseBodyText:    "Send one concrete question. RelayDesk creates the thread automatically and keeps the route in sync.",
			parsePlaceholder: "Ask your first question...",
		}
	}
	if isParseStreaming && parseAssistantMessageCount == 0 {
		return chatJourneyState{
			parseActiveStep:  6,
			parseStageID:     "streaming",
			parseStepLabel:   "Step 6 of 7 - Generating first reply",
			parseHeadingText: "First reply in progress",
			parseBodyText:    "RelayDesk is streaming your first response. Keep this thread open until the reply is complete.",
			parsePlaceholder: "RelayDesk is generating a reply...",
		}
	}
	if parseAssistantMessageCount == 0 {
		return chatJourneyState{
			parseActiveStep:  6,
			parseStageID:     "streaming",
			parseStepLabel:   "Step 6 of 7 - First reply pending",
			parseHeadingText: "Waiting for the first reply",
			parseBodyText:    "The first answer has not completed yet. Retry the send from this thread if needed.",
			parsePlaceholder: "Retry or refine your first question...",
		}
	}
	if parseAssistantMessageCount == 1 {
		return chatJourneyState{
			parseActiveStep:  7,
			parseStageID:     "complete",
			parseStepLabel:   "Step 7 of 7 - First reply complete",
			parseHeadingText: "First reply delivered",
			parseBodyText:    "You are now in a live thread. Continue with follow-ups or start a new chat from the sidebar.",
			parsePlaceholder: "Ask a follow-up...",
		}
	}
	return chatJourneyState{
		parseActiveStep:  7,
		parseStageID:     "complete",
		parseStepLabel:   "Step 7 of 7 - Active thread",
		parseHeadingText: "Continue the conversation",
		parseBodyText:    "Keep iterating in this thread, or branch into a new one when you want a separate context.",
		parsePlaceholder: "Continue the thread...",
	}
}

// parseBuildJourneySteps returns the full seven-stage milestone list used across marketing, auth, and first-chat surfaces.
func parseBuildJourneySteps() []parseJourneyStep {
	return []parseJourneyStep{
		{parseID: "landing", parseLabel: "Landing", parseBody: "Discover the product story"},
		{parseID: "pricing", parseLabel: "Pricing", parseBody: "Review plans and value"},
		{parseID: "signup", parseLabel: "Signup", parseBody: "Create your workspace account"},
		{parseID: "login", parseLabel: "Login", parseBody: "Sign in to continue"},
		{parseID: "compose", parseLabel: "First prompt", parseBody: "Write your first question"},
		{parseID: "streaming", parseLabel: "Streaming", parseBody: "Watch the first reply arrive"},
		{parseID: "complete", parseLabel: "Post-reply", parseBody: "Continue in the same thread"},
	}
}

// parseBuildMarketingJourneyStage maps one public route to its journey stage.
func parseBuildMarketingJourneyStage(parseCurrentPath string) string {
	parseTrimmedPath := strings.TrimSpace(parseCurrentPath)
	switch parseTrimmedPath {
	case marketingPricingRoute, marketingPlansRoute:
		return "pricing"
	case marketingSignupRoute:
		return "signup"
	default:
		return "landing"
	}
}

// parseBuildAuthJourneyStage maps the auth shell mode into its journey stage.
func parseBuildAuthJourneyStage(isParseSignup bool) string {
	if isParseSignup {
		return "signup"
	}
	return "login"
}

// parseBuildJourneyStageByActiveStep resolves one 1-based step index to the matching stage ID.
func parseBuildJourneyStageByActiveStep(parseActiveStep int) string {
	parseSteps := parseBuildJourneySteps()
	if parseActiveStep <= 0 || parseActiveStep > len(parseSteps) {
		return parseSteps[0].parseID
	}
	return parseSteps[parseActiveStep-1].parseID
}

// parseBuildJourneyStageIndex resolves one stage ID to its zero-based index.
func parseBuildJourneyStageIndex(parseStageID string) int {
	parseSteps := parseBuildJourneySteps()
	for parseIndex, parseStep := range parseSteps {
		if parseStep.parseID == parseStageID {
			return parseIndex
		}
	}
	return 0
}

// parseBuildJourneySummary returns one short sentence for the current stage.
func parseBuildJourneySummary(parseStageID string) string {
	parseSteps := parseBuildJourneySteps()
	for _, parseStep := range parseSteps {
		if parseStep.parseID == parseStageID {
			return parseStep.parseBody
		}
	}
	return parseSteps[0].parseBody
}

// renderJourneyProgressBand renders one shared step strip with completed and active stage emphasis.
func renderJourneyProgressBand(parseStageID string) ui.Node {
	parseSteps := parseBuildJourneySteps()
	parseActiveIndex := parseBuildJourneyStageIndex(parseStageID)
	return Div(
		Class("journey-progress-wrap"),
		Div(
			Class("journey-progress-title-row"),
			Span(Class("journey-progress-kicker"), Text("First-chat journey")),
			Span(Class("journey-progress-summary"), Text(parseBuildJourneySummary(parseStageID))),
		),
		Div(
			Class("journey-progress-track"),
			Map(parseSteps, func(parseStep parseJourneyStep) ui.Node {
				parseStepIndex := parseBuildJourneyStageIndex(parseStep.parseID)
				parseStepClass := "journey-progress-step"
				if parseStepIndex < parseActiveIndex {
					parseStepClass += " journey-progress-step-done"
				} else if parseStepIndex == parseActiveIndex {
					parseStepClass += " journey-progress-step-active"
				}
				return Div(
					Class(parseStepClass),
					Span(Class("journey-progress-dot"), Text("\u2022")),
					Span(Class("journey-progress-label"), Text(parseStep.parseLabel)),
				)
			}),
		),
	)
}

// renderJourneyProgressStrip renders one compatibility wrapper that accepts a 1-based active step.
func renderJourneyProgressStrip(parseActiveStep int) ui.Node {
	return renderJourneyProgressBand(parseBuildJourneyStageByActiveStep(parseActiveStep))
}

// parseBuildJourneyStepLabel returns one 1-based human-friendly step label for diagnostics and tests.
func parseBuildJourneyStepLabel(parseStageID string) string {
	parseSteps := parseBuildJourneySteps()
	for parseIndex, parseStep := range parseSteps {
		if parseStep.parseID == parseStageID {
			return "Step " + strconv.Itoa(parseIndex+1) + " of " + strconv.Itoa(len(parseSteps))
		}
	}
	return "Step 1 of " + strconv.Itoa(len(parseSteps))
}
