//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

type parseStarterPrompt struct {
	parseLabel string
	parseText  string
}

// parseBuildStarterPrompts returns the curated first-run starter prompt set.
func parseBuildStarterPrompts() []parseStarterPrompt {
	return []parseStarterPrompt{
		{
			parseLabel: "Plan my first week",
			parseText:  "Plan my first week using RelayDesk. Give me a day-by-day action list with one measurable outcome per day.",
		},
		{
			parseLabel: "Summarize a workflow",
			parseText:  "I run support, product, and operations. Propose a compact workflow that keeps these aligned without daily meetings.",
		},
		{
			parseLabel: "Draft a launch note",
			parseText:  "Draft a concise launch update for customers: what changed, who benefits, and what to do next.",
		},
	}
}

// parseEmptyState renders the first-run chat empty state with explicit journey cues and starter prompt actions.
func parseEmptyState(parseJourney chatJourneyState, parseApplyStarterPrompt ui.Handler) ui.Node {
	parseStarterPrompts := parseBuildStarterPrompts()
	return Div(
		ID(idEmptyState),
		Class("thread-screen flex flex-col items-center gap-4 text-white/40 select-none"),
		Img(
			Src(brandChatIconURL),
			Attr("alt", appBrandName),
			Class("h-16 w-16 rounded-full object-cover"),
		),
		P(Class("text-xs font-semibold uppercase tracking-[0.18em] text-[#7ee9ff] text-center"), Text(parseJourney.parseStepLabel)),
		P(Class("text-2xl font-semibold text-[#f0f0f8] text-center tracking-[-0.02em]"), Text(parseJourney.parseHeadingText)),
		P(Class("text-base text-center max-w-2xl text-[#8a8a9a]"), Text(parseJourney.parseBodyText)),
		Div(
			Class("w-full max-w-3xl"),
			Div(Class("mb-2 text-left text-[11px] uppercase tracking-[0.16em] text-[#9eb9d8]"), Text("Starter prompts")),
			Div(
				Class("grid gap-2 md:grid-cols-3"),
				Map(parseStarterPrompts, func(parsePrompt parseStarterPrompt) ui.Node {
					return Button(
						Class("rounded-2xl border border-[#8fffd8]/18 bg-[#101a27] px-3 py-3 text-left text-sm text-[#dff5ff] transition-colors hover:bg-[#132235]"),
						Data(dataStarterPrompt, parsePrompt.parseText),
						OnClick(parseApplyStarterPrompt),
						Span(Class("font-semibold text-[#eaf8ff]"), Text(parsePrompt.parseLabel)),
					)
				}),
			),
		),
		Div(Class("w-full max-w-3xl"), renderJourneyProgressBand(parseJourney.parseStageID)),
	)
}
