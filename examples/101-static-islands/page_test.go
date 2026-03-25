package main

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestStaticIslandsRenderHelpers(t *testing.T) {
	if len(quoteCards) != 3 {
		t.Fatalf("quoteCards len = %d, want 3", len(quoteCards))
	}
	if quoteCards[0].Phase != "Pipeline" || quoteCards[2].Phase != "Iteration" {
		t.Fatalf("quoteCards = %+v", quoteCards)
	}

	pillMarkup, err := ui.RenderToString(metricPill("Selected tier", "Starter", "selected-tier"))
	if err != nil {
		t.Fatalf("RenderToString(metricPill) error = %v", err)
	}
	for _, expected := range []string{"Selected tier", "Starter", `id="selected-tier"`} {
		if !strings.Contains(pillMarkup, expected) {
			t.Fatalf("metricPill markup missing %q\n%s", expected, pillMarkup)
		}
	}

	activeButton, err := ui.RenderToString(islandButton("Starter", true, ui.RawHandler("click")))
	if err != nil {
		t.Fatalf("RenderToString(islandButton active) error = %v", err)
	}
	if !strings.Contains(activeButton, "border-emerald-300 bg-emerald-300/20") {
		t.Fatalf("active islandButton markup = %q", activeButton)
	}
	inactiveButton, err := ui.RenderToString(islandButton("Team", false, ui.RawHandler("click")))
	if err != nil {
		t.Fatalf("RenderToString(islandButton inactive) error = %v", err)
	}
	if !strings.Contains(inactiveButton, "hover:bg-white/10") {
		t.Fatalf("inactive islandButton markup = %q", inactiveButton)
	}
}

func TestStaticIslandsRenderSurfaceNodes(t *testing.T) {
	newsletterMarkup, err := ui.RenderToString(renderNewsletterIsland("Enterprise", 7))
	if err != nil {
		t.Fatalf("RenderToString(renderNewsletterIsland) error = %v", err)
	}
	for _, expected := range []string{
		"Pricing focus rail",
		"Enterprise",
		"7",
		"newsletter-selected-tier",
		"newsletter-demo-count",
		"Book a walkthrough",
	} {
		if !strings.Contains(newsletterMarkup, expected) {
			t.Fatalf("renderNewsletterIsland markup missing %q\n%s", expected, newsletterMarkup)
		}
	}

	quoteMarkup, err := ui.RenderToString(renderQuoteIsland(quoteCards[1], 1))
	if err != nil {
		t.Fatalf("RenderToString(renderQuoteIsland) error = %v", err)
	}
	for _, expected := range []string{
		"Handoff",
		"Hydrate the narrow decision rails.",
		"2 / 3",
		`id="quote-phase"`,
		`id="quote-title"`,
		`id="quote-summary"`,
		`id="quote-index"`,
		"Next note",
	} {
		if !strings.Contains(quoteMarkup, expected) {
			t.Fatalf("renderQuoteIsland markup missing %q\n%s", expected, quoteMarkup)
		}
	}
}

func TestStaticIslandsComponentNodesRender(t *testing.T) {
	newsletterComponentMarkup, err := ui.RenderToString(ui.CreateElement(newsletterIsland))
	if err != nil {
		t.Fatalf("RenderToString(newsletterIsland) error = %v", err)
	}
	for _, expected := range []string{
		"Pricing focus rail",
		"Starter",
		"3",
		"Book a walkthrough",
		"newsletter-selected-tier",
		"newsletter-demo-count",
	} {
		if !strings.Contains(newsletterComponentMarkup, expected) {
			t.Fatalf("newsletterIsland markup missing %q\n%s", expected, newsletterComponentMarkup)
		}
	}

	quoteComponentMarkup, err := ui.RenderToString(ui.CreateElement(quoteIsland))
	if err != nil {
		t.Fatalf("RenderToString(quoteIsland) error = %v", err)
	}
	for _, expected := range []string{
		"Pipeline",
		"Ship the launch page first.",
		"1 / 3",
		"Next note",
	} {
		if !strings.Contains(quoteComponentMarkup, expected) {
			t.Fatalf("quoteIsland markup missing %q\n%s", expected, quoteComponentMarkup)
		}
	}
}

func TestStaticIslandsNativeMetricsStub(t *testing.T) {
	if got := nowMillis(); got != 0 {
		t.Fatalf("nowMillis() = %v, want 0 in native stub", got)
	}
	writeMetric("metric-startup-total", "5.00 ms")
}
