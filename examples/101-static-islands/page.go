package main

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

var quoteCards = []quoteCard{
	{
		Phase:   "Pipeline",
		Title:   "Ship the launch page first.",
		Summary: "Keep the copy, comparison grid, and proof points static so first paint stays cheap, then activate only the conversion surfaces.",
	},
	{
		Phase:   "Handoff",
		Title:   "Hydrate the narrow decision rails.",
		Summary: "Treat calculators, pricing toggles, and booking widgets as explicit browser roots instead of waking the whole page.",
	},
	{
		Phase:   "Iteration",
		Title:   "Measure the activated regions separately.",
		Summary: "Startup, hydration, and interaction costs should stay visible per island so teams can prune the expensive parts first.",
	},
}

type quoteCard struct {
	Phase   string
	Title   string
	Summary string
}

func metricPill(parseLabel, parseValue, parseElementID string) ui.Node {
	parseProps := html.Props{Class: "rounded-full border border-emerald-400/20 bg-emerald-400/10 px-4 py-3"}
	parseChildren := []ui.Node{
		html.P(html.Props{Class: "text-[0.65rem] uppercase tracking-[0.35em] text-emerald-200/70"}, html.Text(parseLabel)),
	}
	parseValueProps := html.Props{Class: "mt-2 text-lg font-black text-white"}
	if parseElementID != "" {
		parseValueProps.ID = parseElementID
	}
	parseChildren = append(parseChildren, html.P(parseValueProps, html.Text(parseValue)))
	return html.Div(parseProps, parseChildren...)
}

func islandButton(parseLabel string, isSelected bool, parseHandler ui.Handler) ui.Node {
	parseClassName := "rounded-full border px-4 py-2 text-sm font-semibold transition "
	if isSelected {
		parseClassName += "border-emerald-300 bg-emerald-300/20 text-white"
	} else {
		parseClassName += "border-white/10 bg-white/5 text-slate-200 hover:bg-white/10"
	}
	return html.Button(html.Props{Class: parseClassName, OnClick: parseHandler}, html.Text(parseLabel))
}

func renderNewsletterIsland(parseTier string, parseDemos int) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-slate-950/90 p-6 shadow-[0_20px_70px_rgba(8,15,25,0.35)]"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-emerald-300"}, html.Text("Interactive island")),
		html.H2(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text("Pricing focus rail")),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Only this pricing surface hydrates. The surrounding launch narrative stays static HTML.")),
		html.Div(html.Props{Class: "mt-5 grid gap-3 sm:grid-cols-2"},
			metricPill("Selected tier", parseTier, "newsletter-selected-tier"),
			metricPill("Booked demos", fmt.Sprintf("%d", parseDemos), "newsletter-demo-count"),
		),
		html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
			islandButton("Starter", parseTier == "Starter", ui.Handler{}),
			islandButton("Team", parseTier == "Team", ui.Handler{}),
			islandButton("Enterprise", parseTier == "Enterprise", ui.Handler{}),
		),
		html.Button(html.Props{Class: "mt-5 w-full rounded-2xl border border-emerald-300/30 bg-emerald-300/15 px-5 py-4 font-semibold text-white"},
			html.Text("Book a walkthrough"),
		),
		html.P(html.Props{Class: "mt-3 text-xs leading-6 text-slate-400"}, html.Text("The button and tier chips become interactive after only this island resumes.")),
	)
}

func newsletterIsland() ui.Node {
	parseTier := ui.UseState("Starter")
	parseDemos := ui.UseState(3)
	setTier := func(parseNext string) ui.Handler {
		return ui.UseEvent(func() {
			parseStarted := nowMillis()
			parseTier.Set(parseNext)
			writeMetric("metric-newsletter-interaction", fmt.Sprintf("%.2f ms", nowMillis()-parseStarted))
		})
	}
	parseBookWalkthrough := ui.UseEvent(func() {
		parseStarted2 := nowMillis()
		parseDemos.Update(func(parsePrevious int) int { return parsePrevious + 1 })
		writeMetric("metric-newsletter-interaction", fmt.Sprintf("%.2f ms", nowMillis()-parseStarted2))
	})

	return html.Div(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-slate-950/90 p-6 shadow-[0_20px_70px_rgba(8,15,25,0.35)]"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-emerald-300"}, html.Text("Interactive island")),
		html.H2(html.Props{Class: "mt-3 text-3xl font-black text-white"}, html.Text("Pricing focus rail")),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300"}, html.Text("Only this pricing surface hydrates. The surrounding launch narrative stays static HTML.")),
		html.Div(html.Props{Class: "mt-5 grid gap-3 sm:grid-cols-2"},
			metricPill("Selected tier", parseTier.Get(), "newsletter-selected-tier"),
			metricPill("Booked demos", fmt.Sprintf("%d", parseDemos.Get()), "newsletter-demo-count"),
		),
		html.Div(html.Props{Class: "mt-5 flex flex-wrap gap-3"},
			islandButton("Starter", parseTier.Get() == "Starter", setTier("Starter")),
			islandButton("Team", parseTier.Get() == "Team", setTier("Team")),
			islandButton("Enterprise", parseTier.Get() == "Enterprise", setTier("Enterprise")),
		),
		html.Button(html.Props{Class: "mt-5 w-full rounded-2xl border border-emerald-300/30 bg-emerald-300/15 px-5 py-4 font-semibold text-white", OnClick: parseBookWalkthrough},
			html.Text("Book a walkthrough"),
		),
		html.P(html.Props{Class: "mt-3 text-xs leading-6 text-slate-400"}, html.Text("The button and tier chips become interactive after only this island resumes.")),
	)
}

func renderQuoteIsland(parseCard quoteCard, parseIndex int) ui.Node {
	return html.Div(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-[#111a28] p-6"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Interactive island")),
		html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.35em] text-slate-400", ID: "quote-phase"}, html.Text(parseCard.Phase)),
		html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white", ID: "quote-title"}, html.Text(parseCard.Title)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300", ID: "quote-summary"}, html.Text(parseCard.Summary)),
		html.Div(html.Props{Class: "mt-5 flex items-center justify-between"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-500", ID: "quote-index"}, html.Text(fmt.Sprintf("%d / %d", parseIndex+1, len(quoteCards)))),
			html.Button(html.Props{Class: "rounded-full border border-cyan-300/30 bg-cyan-300/10 px-4 py-2 text-sm font-semibold text-white"},
				html.Text("Next note"),
			),
		),
	)
}

func quoteIsland() ui.Node {
	parseIndex := ui.UseState(0)
	parseNext := ui.UseEvent(func() {
		parseStarted := nowMillis()
		parseIndex.Update(func(parsePrevious int) int { return (parsePrevious + 1) % len(quoteCards) })
		writeMetric("metric-quote-interaction", fmt.Sprintf("%.2f ms", nowMillis()-parseStarted))
	})
	parseCard := quoteCards[parseIndex.Get()]

	return html.Div(html.Props{Class: "rounded-[1.75rem] border border-white/10 bg-[#111a28] p-6"},
		html.P(html.Props{Class: "text-xs uppercase tracking-[0.35em] text-cyan-300"}, html.Text("Interactive island")),
		html.P(html.Props{Class: "mt-3 text-xs uppercase tracking-[0.35em] text-slate-400", ID: "quote-phase"}, html.Text(parseCard.Phase)),
		html.H2(html.Props{Class: "mt-3 text-2xl font-black text-white", ID: "quote-title"}, html.Text(parseCard.Title)),
		html.P(html.Props{Class: "mt-3 text-sm leading-7 text-slate-300", ID: "quote-summary"}, html.Text(parseCard.Summary)),
		html.Div(html.Props{Class: "mt-5 flex items-center justify-between"},
			html.P(html.Props{Class: "text-xs uppercase tracking-[0.25em] text-slate-500", ID: "quote-index"}, html.Text(fmt.Sprintf("%d / %d", parseIndex.Get()+1, len(quoteCards)))),
			html.Button(html.Props{Class: "rounded-full border border-cyan-300/30 bg-cyan-300/10 px-4 py-2 text-sm font-semibold text-white", OnClick: parseNext},
				html.Text("Next note"),
			),
		),
	)
}
