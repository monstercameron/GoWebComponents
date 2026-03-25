package shared

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestExampleSubjectSelectsBestCandidate(t *testing.T) {
	tests := []struct {
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := exampleSubject(test.title, test.feature); got != test.want {
				t.Fatalf("exampleSubject(%q, %q) = %q, want %q", test.title, test.feature, got, test.want)
			}
		})
	}
}

func TestExampleCopyBranches(t *testing.T) {
	lead, bullets := overviewCopy("Use state", "ui.UseState")
	if !strings.Contains(lead, "single component owns interactive local state") {
		t.Fatalf("unexpected overview lead: %q", lead)
	}
	if len(bullets) != 2 {
		t.Fatalf("expected 2 overview bullets for ui.UseState, got %d", len(bullets))
	}

	functional := functionalCopy("Offline", "pwa.RegisterServiceWorker")
	if len(functional) < 4 {
		t.Fatalf("expected pwa functional copy to include specialized bullets, got %v", functional)
	}
	if !strings.Contains(strings.Join(functional, "\n"), "offline support matters") {
		t.Fatalf("expected pwa-specific functional guidance, got %v", functional)
	}

	implementationLead, implementationBullets, codeLines := implementationCopy("Plugin host", "plugin.NewHost")
	if !strings.Contains(implementationLead, "explicit host") {
		t.Fatalf("unexpected implementation lead: %q", implementationLead)
	}
	if len(implementationBullets) < 4 {
		t.Fatalf("expected implementation bullets to include plugin-specific guidance, got %v", implementationBullets)
	}
	if strings.Join(codeLines, "\n") == "" || !strings.Contains(strings.Join(codeLines, "\n"), "plugin.NewHost") {
		t.Fatalf("expected plugin implementation code example, got %v", codeLines)
	}
}

func TestExampleComponentsRenderExpectedMarkup(t *testing.T) {
	page := ExamplePage("Catalog demo", "ui.UseState", "Short summary")
	markup, err := ui.RenderToString(page)
	if err != nil {
		t.Fatalf("RenderToString(ExamplePage): %v", err)
	}
	for _, expected := range []string{
		"Catalog demo",
		"Short summary",
		"Overview",
		"Functional",
		"Implementation",
	} {
		if !strings.Contains(markup, expected) {
			t.Fatalf("expected ExamplePage markup to contain %q, got %s", expected, markup)
		}
	}

	listMarkup, err := ui.RenderToString(ExampleBulletList(" keep ", "", "trim me"))
	if err != nil {
		t.Fatalf("RenderToString(ExampleBulletList): %v", err)
	}
	if !strings.Contains(listMarkup, "keep") || !strings.Contains(listMarkup, "trim me") {
		t.Fatalf("unexpected bullet list markup: %s", listMarkup)
	}

	codeMarkup, err := ui.RenderToString(ExampleCode("line one", "line two"))
	if err != nil {
		t.Fatalf("RenderToString(ExampleCode): %v", err)
	}
	if !strings.Contains(codeMarkup, "<pre") || !strings.Contains(codeMarkup, "line one") || !strings.Contains(codeMarkup, "line two") {
		t.Fatalf("unexpected code markup: %s", codeMarkup)
	}

	panelMarkup, err := ui.RenderToString(ExamplePanel("Panel", ui.Text("body")))
	if err != nil {
		t.Fatalf("RenderToString(ExamplePanel): %v", err)
	}
	if !strings.Contains(panelMarkup, "Panel") || !strings.Contains(panelMarkup, "body") {
		t.Fatalf("unexpected panel markup: %s", panelMarkup)
	}

	buttonMarkup, err := ui.RenderToString(ExampleButton("Click", ui.Handler{}))
	if err != nil {
		t.Fatalf("RenderToString(ExampleButton): %v", err)
	}
	if !strings.Contains(buttonMarkup, "<button") || !strings.Contains(buttonMarkup, "Click") {
		t.Fatalf("unexpected button markup: %s", buttonMarkup)
	}

	statMarkup, err := ui.RenderToString(ExampleStat("Latency", "42ms"))
	if err != nil {
		t.Fatalf("RenderToString(ExampleStat): %v", err)
	}
	if !strings.Contains(statMarkup, "Latency") || !strings.Contains(statMarkup, "42ms") {
		t.Fatalf("unexpected stat markup: %s", statMarkup)
	}
}
