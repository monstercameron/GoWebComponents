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
			parseStepLabel:   "Step 7 of 7 - Conversation ready",
			parseHeadingText: "Conversation ready",
			parseBodyText:    "Continue when you're ready.",
			parsePlaceholder: "Ask a follow-up...",
		}
	}
	return chatJourneyState{
		parseActiveStep:  7,
		parseStageID:     "complete",
		parseStepLabel:   "Step 7 of 7 - Conversation ready",
		parseHeadingText: "Conversation ready",
		parseBodyText:    "Continue when you're ready.",
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
		{parseID: "complete", parseLabel: "Ready", parseBody: "Conversation ready"},
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

// renderJourneyProgressBand renders a compact one-line progress indicator: a thin
// fill bar scaled to the current step fraction and a single "Step X of 7 · Label"
// text. This replaces the previous seven-pill dot-track which dominated the page.
func renderJourneyProgressBand(parseStageID string) ui.Node {
	parseSteps := parseBuildJourneySteps()
	parseActiveIndex := parseBuildJourneyStageIndex(parseStageID)
	parseTotal := len(parseSteps)
	// Fill fraction: completed steps are fully filled; active step is half-filled so
	// the bar visually reads as "in progress" rather than done.
	parseFillPct := 0
	if parseTotal > 1 {
		parseFillPct = (parseActiveIndex*100*2 + 100) / (parseTotal * 2)
	}
	parseLabel := "Step " + strconv.Itoa(parseActiveIndex+1) + " of " + strconv.Itoa(parseTotal) + " · " + parseBuildJourneySummary(parseStageID)
	return Div(
		ClassStr("journey-progress-wrap"),
		Div(
			ClassStr("flex items-center gap-3"),
			// thin fill bar
			Div(
				ClassStr("relative h-[3px] min-w-0 flex-1 overflow-hidden rounded-full bg-white/8"),
				Div(
					ClassStr("absolute inset-y-0 left-0 rounded-full bg-gradient-to-r from-[#8e7bff]/60 to-[#8e7bff]/30 transition-all duration-500"),
					Style(map[string]string{"width": strconv.Itoa(parseFillPct) + "%"}),
				),
			),
			// current stage label — shrink-0 so it never wraps the bar
			Span(
				ClassStr("shrink-0 text-[0.68rem] leading-none tracking-[0.04em] text-white/55"),
				Text(parseLabel),
			),
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
