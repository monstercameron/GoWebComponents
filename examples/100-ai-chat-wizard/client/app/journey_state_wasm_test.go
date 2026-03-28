//go:build js && wasm

package app

import "testing"

// TestParseBuildMarketingJourneyStage verifies public-route stage derivation for the journey strip.
func TestParseBuildMarketingJourneyStage(parseT *testing.T) {
	parseTests := []struct {
		name        string
		currentPath string
		wantStageID string
	}{
		{name: "home route", currentPath: marketingHomeRoute, wantStageID: "landing"},
		{name: "root route", currentPath: authLandingRoute, wantStageID: "landing"},
		{name: "pricing route", currentPath: marketingPricingRoute, wantStageID: "pricing"},
		{name: "plans route", currentPath: marketingPlansRoute, wantStageID: "pricing"},
		{name: "signup route", currentPath: marketingSignupRoute, wantStageID: "signup"},
	}
	for _, parseTest := range parseTests {
		if parseGot := parseBuildMarketingJourneyStage(parseTest.currentPath); parseGot != parseTest.wantStageID {
			parseT.Fatalf("%s: parseBuildMarketingJourneyStage(%q) = %q, want %q", parseTest.name, parseTest.currentPath, parseGot, parseTest.wantStageID)
		}
	}
}

// TestParseBuildAuthJourneyStage verifies login vs signup stage mapping for the auth shell.
func TestParseBuildAuthJourneyStage(parseT *testing.T) {
	if parseGot := parseBuildAuthJourneyStage(false); parseGot != "login" {
		parseT.Fatalf("parseBuildAuthJourneyStage(false) = %q, want %q", parseGot, "login")
	}
	if parseGot := parseBuildAuthJourneyStage(true); parseGot != "signup" {
		parseT.Fatalf("parseBuildAuthJourneyStage(true) = %q, want %q", parseGot, "signup")
	}
}

// TestParseBuildChatJourneyState verifies first-chat journey stage derivation across empty, streaming, and post-reply states.
func TestParseBuildChatJourneyState(parseT *testing.T) {
	parseT.Parallel()

	parseTests := []struct {
		name            string
		messages        []message
		isStreaming     bool
		wantStageID     string
		wantStepLabel   string
		wantHeadingText string
		wantPlaceholder string
	}{
		{
			name:            "empty thread uses first prompt state",
			messages:        nil,
			isStreaming:     false,
			wantStageID:     "compose",
			wantStepLabel:   "Step 5 of 7 - Send your first prompt",
			wantHeadingText: "Start your first chat",
			wantPlaceholder: "Ask your first question...",
		},
		{
			name: "first reply streaming",
			messages: []message{
				{Role: roleUser, Content: "hello"},
			},
			isStreaming:     true,
			wantStageID:     "streaming",
			wantStepLabel:   "Step 6 of 7 - Generating first reply",
			wantHeadingText: "First reply in progress",
			wantPlaceholder: "RelayDesk is generating a reply...",
		},
		{
			name: "first reply pending when stream inactive",
			messages: []message{
				{Role: roleUser, Content: "hello"},
			},
			isStreaming:     false,
			wantStageID:     "streaming",
			wantStepLabel:   "Step 6 of 7 - First reply pending",
			wantHeadingText: "Waiting for the first reply",
			wantPlaceholder: "Retry or refine your first question...",
		},
		{
			name: "first reply complete",
			messages: []message{
				{Role: roleUser, Content: "hello"},
				{Role: roleAssistant, Content: "world"},
			},
			isStreaming:     false,
			wantStageID:     "complete",
			wantStepLabel:   "Step 7 of 7 - First reply complete",
			wantHeadingText: "First reply delivered",
			wantPlaceholder: "Ask a follow-up...",
		},
		{
			name: "multi reply thread uses active state",
			messages: []message{
				{Role: roleUser, Content: "hello"},
				{Role: roleAssistant, Content: "world"},
				{Role: roleUser, Content: "next"},
				{Role: roleAssistant, Content: "again"},
			},
			isStreaming:     false,
			wantStageID:     "complete",
			wantStepLabel:   "Step 7 of 7 - Active thread",
			wantHeadingText: "Continue the conversation",
			wantPlaceholder: "Continue the thread...",
		},
	}

	for _, parseTest := range parseTests {
		parseTest2 := parseTest
		parseT.Run(parseTest2.name, func(parseT2 *testing.T) {
			parseT2.Parallel()
			parseJourney := parseBuildChatJourneyState(parseTest2.messages, parseTest2.isStreaming)
			if parseJourney.parseStageID != parseTest2.wantStageID {
				parseT2.Fatalf("parseBuildChatJourneyState() stage = %q, want %q", parseJourney.parseStageID, parseTest2.wantStageID)
			}
			if parseJourney.parseStepLabel != parseTest2.wantStepLabel {
				parseT2.Fatalf("parseBuildChatJourneyState() step label = %q, want %q", parseJourney.parseStepLabel, parseTest2.wantStepLabel)
			}
			if parseJourney.parseHeadingText != parseTest2.wantHeadingText {
				parseT2.Fatalf("parseBuildChatJourneyState() heading = %q, want %q", parseJourney.parseHeadingText, parseTest2.wantHeadingText)
			}
			if parseJourney.parsePlaceholder != parseTest2.wantPlaceholder {
				parseT2.Fatalf("parseBuildChatJourneyState() placeholder = %q, want %q", parseJourney.parsePlaceholder, parseTest2.wantPlaceholder)
			}
		})
	}
}
