package shared

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestExampleSubjectSelectsBestCandidate(parseT *testing.T) {
	parseTests := []struct {
		name    string
		title   string
		feature string
		want    string
	}{
		{name: "prefers dotted title", title: "ui.UseState", feature: "Counter", want: "ui.UseState"},
		{name: "uses dotted feature when title is plain", title: "Counter", feature: "fetch.UseResource", want: "fetch.UseResource"},
		{name: "router-like feature wins", title: "Screen", feature: "Route Hydrate", want: "Route Hydrate"},
		{name: "falls back to trimmed title", title: "  Plain demo  ", feature: "", want: "Plain demo"},
	}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := exampleSubject(parseTest.title, parseTest.feature); parseGot != parseTest.want {
				parseT2.Fatalf("exampleSubject(%q, %q) = %q, want %q", parseTest.title, parseTest.feature, parseGot, parseTest.want)
			}
		})
	}
}

func TestExampleCopyBranches(parseT *testing.T) {
	parseLead, parseBullets := overviewCopy("Use state", "ui.UseState")
	if !strings.Contains(parseLead, "single component owns interactive local state") {
		parseT.Fatalf("unexpected overview lead: %q", parseLead)
	}
	if len(parseBullets) != 2 {
		parseT.Fatalf("expected 2 overview bullets for ui.UseState, got %d", len(parseBullets))
	}

	parseFunctional := functionalCopy("Offline", "pwa.RegisterServiceWorker")
	if len(parseFunctional) < 4 {
		parseT.Fatalf("expected pwa functional copy to include specialized bullets, got %v", parseFunctional)
	}
	if !strings.Contains(strings.Join(parseFunctional, "\n"), "offline support matters") {
		parseT.Fatalf("expected pwa-specific functional guidance, got %v", parseFunctional)
	}

	parseImplementationLead, parseImplementationBullets, parseCodeLines := implementationCopy("Plugin host", "plugin.NewHost")
	if !strings.Contains(parseImplementationLead, "explicit host") {
		parseT.Fatalf("unexpected implementation lead: %q", parseImplementationLead)
	}
	if len(parseImplementationBullets) < 4 {
		parseT.Fatalf("expected implementation bullets to include plugin-specific guidance, got %v", parseImplementationBullets)
	}
	if strings.Join(parseCodeLines, "\n") == "" || !strings.Contains(strings.Join(parseCodeLines, "\n"), "plugin.NewHost") {
		parseT.Fatalf("expected plugin implementation code example, got %v", parseCodeLines)
	}
}

func TestExampleComponentsRenderExpectedMarkup(parseT *testing.T) {
	parsePage := ExamplePage("Catalog demo", "ui.UseState", "Short summary")
	parseMarkup, parseErr := ui.RenderToString(parsePage)
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExamplePage): %v", parseErr)
	}
	for _, parseExpected := range []string{
		"Catalog demo",
		"Short summary",
		"Overview",
		"Functional",
		"Implementation",
	} {
		if !strings.Contains(parseMarkup, parseExpected) {
			parseT.Fatalf("expected ExamplePage markup to contain %q, got %s", parseExpected, parseMarkup)
		}
	}

	parseListMarkup, parseErr := ui.RenderToString(ExampleBulletList(" keep ", "", "trim me"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExampleBulletList): %v", parseErr)
	}
	if !strings.Contains(parseListMarkup, "keep") || !strings.Contains(parseListMarkup, "trim me") {
		parseT.Fatalf("unexpected bullet list markup: %s", parseListMarkup)
	}

	parseCodeMarkup, parseErr := ui.RenderToString(ExampleCode("line one", "line two"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExampleCode): %v", parseErr)
	}
	if !strings.Contains(parseCodeMarkup, "<pre") || !strings.Contains(parseCodeMarkup, "line one") || !strings.Contains(parseCodeMarkup, "line two") {
		parseT.Fatalf("unexpected code markup: %s", parseCodeMarkup)
	}

	parsePanelMarkup, parseErr := ui.RenderToString(ExamplePanel("Panel", ui.Text("body")))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExamplePanel): %v", parseErr)
	}
	if !strings.Contains(parsePanelMarkup, "Panel") || !strings.Contains(parsePanelMarkup, "body") {
		parseT.Fatalf("unexpected panel markup: %s", parsePanelMarkup)
	}

	parseButtonMarkup, parseErr := ui.RenderToString(ExampleButton("Click", ui.Handler{}))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExampleButton): %v", parseErr)
	}
	if !strings.Contains(parseButtonMarkup, "<button") || !strings.Contains(parseButtonMarkup, "Click") {
		parseT.Fatalf("unexpected button markup: %s", parseButtonMarkup)
	}

	parseStatMarkup, parseErr := ui.RenderToString(ExampleStat("Latency", "42ms"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExampleStat): %v", parseErr)
	}
	if !strings.Contains(parseStatMarkup, "Latency") || !strings.Contains(parseStatMarkup, "42ms") {
		parseT.Fatalf("unexpected stat markup: %s", parseStatMarkup)
	}
}
