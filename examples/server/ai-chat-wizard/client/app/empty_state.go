//go:build js && wasm

package app

import (
	. "github.com/monstercameron/GoWebComponents/v4/html/shorthand"
	"github.com/monstercameron/GoWebComponents/v4/ui"
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

// parseEmptyState renders the first-run chat empty state with explicit first-action guidance and starter prompt actions.
func parseEmptyState(parseJourney chatJourneyState, parseApplyStarterPrompt ui.Handler) ui.Node {
	parseStarterPrompts := parseBuildStarterPrompts()
	return Div(
		ID(idEmptyState),
		ClassStr("thread-screen flex flex-col items-center gap-6 text-white/40 select-none"),
		Img(
			Src(brandChatIconURL),
			Attr("alt", appBrandName),
			ClassStr("h-16 w-16 rounded-full object-cover"),
		),
		// three first-action cards
		Div(
			ClassStr("w-full max-w-3xl"),
			Div(ClassStr("mb-3 text-left text-[11px] uppercase tracking-[0.16em] text-[#9c9fbd]"), Text("Where to start")),
			Div(
				ClassStr("grid gap-3 md:grid-cols-3"),
				// action 1: ask first question — triggers the first starter prompt
				Button(
					ClassStr("rounded-2xl border border-[#8fffd8]/18 bg-[#12121c] px-4 py-4 text-left transition-colors hover:bg-[#181830]"),
					Data(dataStarterPrompt, parseStarterPrompts[0].parseText),
					OnClick(parseApplyStarterPrompt),
					Div(ClassStr("mb-1 text-xs font-semibold text-[#ecebf7]"), Text("Ask your first question")),
					P(ClassStr("text-[11px] leading-4 text-[#9b9bb1]"), Text("Start a thread and get an answer. Your workspace is ready, no further setup needed.")),
				),
				// action 2: billing visibility — links to settings billing panel
				A(
					ClassStr("block rounded-2xl border border-white/[0.06] bg-[#12121c] px-4 py-4 text-left transition-colors hover:bg-[#181830]"),
					Href(settingsRoutePath+"?"+settingsPanelQueryKey+"="+settingsSectionBilling),
					Div(ClassStr("mb-1 text-xs font-semibold text-[#ecebf7]"), Text("Review billing visibility")),
					P(ClassStr("text-[11px] leading-4 text-[#9b9bb1]"), Text("See platform fee, usage cost, and current-period totals. Every AI turn is reflected in real time.")),
				),
				// action 3: workspace defaults — links to settings
				A(
					ClassStr("block rounded-2xl border border-white/[0.06] bg-[#12121c] px-4 py-4 text-left transition-colors hover:bg-[#181830]"),
					Href(settingsRoutePath),
					Div(ClassStr("mb-1 text-xs font-semibold text-[#ecebf7]"), Text("Set workspace defaults")),
					P(ClassStr("text-[11px] leading-4 text-[#9b9bb1]"), Text("Choose your default model, set tone preferences, and configure system prompt defaults for every new thread.")),
				),
			),
			// secondary guidance
			P(
				ClassStr("mt-4 text-[11px] leading-5 text-[#9b9bb1]"),
				Text("Your first thread's title is derived from your opening message. Model choice and system-prompt scope are configured in Settings. All thread history is stored per workspace."),
			),
		),
		// starter prompts — secondary path for users who prefer guided options
		Div(
			ClassStr("w-full max-w-3xl"),
			Div(ClassStr("mb-2 text-left text-[11px] uppercase tracking-[0.16em] text-[#9c9fbd]"), Text("Or try a starter prompt")),
			Div(
				ClassStr("grid gap-2 md:grid-cols-3"),
				Map(parseStarterPrompts, func(parsePrompt parseStarterPrompt) ui.Node {
					return Button(
						ClassStr("rounded-2xl border border-[#8fffd8]/18 bg-[#12121c] px-3 py-3 text-left text-sm text-[#dff5ff] transition-colors hover:bg-[#181830]"),
						Data(dataStarterPrompt, parsePrompt.parseText),
						OnClick(parseApplyStarterPrompt),
						Span(ClassStr("font-semibold text-[#ecebf7]"), Text(parsePrompt.parseLabel)),
					)
				}),
			),
		),
		Div(ClassStr("w-full max-w-3xl"), renderJourneyProgressBand(parseJourney.parseStageID)),
	)
}
