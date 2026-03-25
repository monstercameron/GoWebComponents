package shared

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestExampleComponentsRenderMarkup(t *testing.T) {
	page := ExamplePage("Plugin Host", "plugin.NewHost", "Reviewable extension wiring", ExampleStat("Plugins", "3"))
	markup, err := ui.RenderToString(page)
	if err != nil {
		t.Fatalf("RenderToString(ExamplePage) error = %v", err)
	}
	for _, expected := range []string{"Plugin Host", "plugin.NewHost", "Reviewable extension wiring", "Plugins", "3"} {
		if !strings.Contains(markup, expected) {
			t.Fatalf("ExamplePage markup missing %q\n%s", expected, markup)
		}
	}

	code, err := ui.RenderToString(ExampleCode("line one", "line two"))
	if err != nil {
		t.Fatalf("RenderToString(ExampleCode) error = %v", err)
	}
	if !strings.Contains(code, "line one") || !strings.Contains(code, "line two") {
		t.Fatalf("ExampleCode markup = %q, want both lines", code)
	}

	button, err := ui.RenderToString(ExampleButton("Inspect", ui.RawHandler("click")))
	if err != nil {
		t.Fatalf("RenderToString(ExampleButton) error = %v", err)
	}
	if !strings.Contains(button, "Inspect") {
		t.Fatalf("ExampleButton markup = %q, want label", button)
	}
}

func TestExampleCopyCoversAdditionalSubjectFamilies(t *testing.T) {
	tests := []struct {
		title   string
		feature string
		want    string
	}{
		{title: "Cache Storage", feature: "pwa.OpenCacheStorageManager", want: "offline"},
		{title: "Plugin Host", feature: "plugin.NewHost", want: "extension"},
		{title: "Hash Routing", feature: "router.NewHashRouter", want: "route"},
		{title: "Shared Atom", feature: "state.UseAtom", want: "shared"},
		{title: "Resource Loader", feature: "fetch.UseResource", want: "loading"},
		{title: "Semantic Markup", feature: "html.Div", want: "markup"},
		{title: "Generic Example", feature: "feature", want: "understand the purpose"},
	}

	for _, test := range tests {
		t.Run(test.feature, func(t *testing.T) {
			overviewLead, overviewBullets := overviewCopy(test.title, test.feature)
			functionalBullets := functionalCopy(test.title, test.feature)
			implementationLead, implementationBullets, codeLines := implementationCopy(test.title, test.feature)

			allText := strings.ToLower(strings.Join(append(append([]string{overviewLead, implementationLead}, overviewBullets...), append(functionalBullets, implementationBullets...)...), " "))
			if !strings.Contains(allText, test.want) {
				t.Fatalf("combined example copy for %q missing %q\n%s", test.feature, test.want, allText)
			}
			if len(codeLines) == 0 {
				t.Fatalf("implementationCopy(%q) returned no code lines", test.feature)
			}
		})
	}
}

func TestExampleDocumentationAdditionalBranches(t *testing.T) {
	tests := []struct {
		title   string
		feature string
		want    string
	}{
		{title: "Runtime Inspector", feature: "devtools.OpenPanel", want: "instrumentation"},
		{title: "Server Render", feature: "ui.RenderToString", want: "server-side html generation"},
		{title: "Bootstrap Resume", feature: "ui.RenderBootstrapScript, ui.ReadBootstrapScript", want: "existing html already being in the document"},
		{title: "Resume Existing DOM", feature: "router.HydrateMount", want: "existing html already being in the document"},
	}

	for _, test := range tests {
		t.Run(test.feature, func(t *testing.T) {
			lead, bullets, code := implementationCopy(test.title, test.feature)
			allText := strings.ToLower(strings.Join(append([]string{lead}, bullets...), " "))
			if !strings.Contains(allText, test.want) {
				t.Fatalf("implementationCopy(%q) missing %q\n%s", test.feature, test.want, allText)
			}
			if len(code) == 0 {
				t.Fatalf("implementationCopy(%q) returned no code lines", test.feature)
			}
		})
	}

	bulletsMarkup, err := ui.RenderToString(ExampleBulletList("first", "", "second"))
	if err != nil {
		t.Fatalf("RenderToString(ExampleBulletList) error = %v", err)
	}
	if strings.Contains(bulletsMarkup, "<li></li>") || !strings.Contains(bulletsMarkup, "first") || !strings.Contains(bulletsMarkup, "second") {
		t.Fatalf("ExampleBulletList markup = %q, want trimmed non-empty bullets", bulletsMarkup)
	}

	panelMarkup, err := ui.RenderToString(ExamplePanel("Implementation", ExampleStat("Coverage", "80%")))
	if err != nil {
		t.Fatalf("RenderToString(ExamplePanel) error = %v", err)
	}
	if !strings.Contains(panelMarkup, "Implementation") || !strings.Contains(panelMarkup, "Coverage") {
		t.Fatalf("ExamplePanel markup = %q, want title and content", panelMarkup)
	}
}
