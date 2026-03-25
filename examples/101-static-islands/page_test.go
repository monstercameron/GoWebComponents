package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestStaticIslandsRenderHelpers(parseT *testing.T) {
	if len(quoteCards) != 3 {
		parseT.Fatalf("quoteCards len = %d, want 3", len(quoteCards))
	}
	if quoteCards[0].Phase != "Pipeline" || quoteCards[2].Phase != "Iteration" {
		parseT.Fatalf("quoteCards = %+v", quoteCards)
	}

	parsePillMarkup, parseErr := ui.RenderToString(metricPill("Selected tier", "Starter", "selected-tier"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(metricPill) error = %v", parseErr)
	}
	for _, parseExpected := range []string{"Selected tier", "Starter", `id="selected-tier"`} {
		if !strings.Contains(parsePillMarkup, parseExpected) {
			parseT.Fatalf("metricPill markup missing %q\n%s", parseExpected, parsePillMarkup)
		}
	}

	parseActiveButton, parseErr := ui.RenderToString(islandButton("Starter", true, ui.WrapHandler("click")))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(islandButton active) error = %v", parseErr)
	}
	if !strings.Contains(parseActiveButton, "border-emerald-300 bg-emerald-300/20") {
		parseT.Fatalf("active islandButton markup = %q", parseActiveButton)
	}
	parseInactiveButton, parseErr := ui.RenderToString(islandButton("Team", false, ui.WrapHandler("click")))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(islandButton inactive) error = %v", parseErr)
	}
	if !strings.Contains(parseInactiveButton, "hover:bg-white/10") {
		parseT.Fatalf("inactive islandButton markup = %q", parseInactiveButton)
	}
}

func TestStaticIslandsRenderSurfaceNodes(parseT *testing.T) {
	parseNewsletterMarkup, parseErr := ui.RenderToString(renderNewsletterIsland("Enterprise", 7))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderNewsletterIsland) error = %v", parseErr)
	}
	for _, parseExpected := range []string{
		"Pricing focus rail",
		"Enterprise",
		"7",
		"newsletter-selected-tier",
		"newsletter-demo-count",
		"Book a walkthrough",
	} {
		if !strings.Contains(parseNewsletterMarkup, parseExpected) {
			parseT.Fatalf("renderNewsletterIsland markup missing %q\n%s", parseExpected, parseNewsletterMarkup)
		}
	}

	parseQuoteMarkup, parseErr := ui.RenderToString(renderQuoteIsland(quoteCards[1], 1))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderQuoteIsland) error = %v", parseErr)
	}
	for _, parseExpected2 := range []string{
		"Handoff",
		"Hydrate the narrow decision rails.",
		"2 / 3",
		`id="quote-phase"`,
		`id="quote-title"`,
		`id="quote-summary"`,
		`id="quote-index"`,
		"Next note",
	} {
		if !strings.Contains(parseQuoteMarkup, parseExpected2) {
			parseT.Fatalf("renderQuoteIsland markup missing %q\n%s", parseExpected2, parseQuoteMarkup)
		}
	}
}

func TestStaticIslandsComponentNodesRender(parseT *testing.T) {
	parseNewsletterComponentMarkup, parseErr := ui.RenderToString(ui.CreateElement(newsletterIsland))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(newsletterIsland) error = %v", parseErr)
	}
	for _, parseExpected := range []string{
		"Pricing focus rail",
		"Starter",
		"3",
		"Book a walkthrough",
		"newsletter-selected-tier",
		"newsletter-demo-count",
	} {
		if !strings.Contains(parseNewsletterComponentMarkup, parseExpected) {
			parseT.Fatalf("newsletterIsland markup missing %q\n%s", parseExpected, parseNewsletterComponentMarkup)
		}
	}

	parseQuoteComponentMarkup, parseErr := ui.RenderToString(ui.CreateElement(quoteIsland))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(quoteIsland) error = %v", parseErr)
	}
	for _, parseExpected2 := range []string{
		"Pipeline",
		"Ship the launch page first.",
		"1 / 3",
		"Next note",
	} {
		if !strings.Contains(parseQuoteComponentMarkup, parseExpected2) {
			parseT.Fatalf("quoteIsland markup missing %q\n%s", parseExpected2, parseQuoteComponentMarkup)
		}
	}
}

func TestStaticIslandsNativeMetricsStub(parseT *testing.T) {
	if parseGot := nowMillis(); parseGot != 0 {
		parseT.Fatalf("nowMillis() = %v, want 0 in native stub", parseGot)
	}
	writeMetric("metric-startup-total", "5.00 ms")
}
