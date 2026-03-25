package shared

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestExampleComponentsRenderMarkup(parseT *testing.T) {
	parsePage := ExamplePage("Plugin Host", "plugin.NewHost", "Reviewable extension wiring", ExampleStat("Plugins", "3"))
	parseMarkup, parseErr := ui.RenderToString(parsePage)
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExamplePage) error = %v", parseErr)
	}
	for _, parseExpected := range []string{"Plugin Host", "plugin.NewHost", "Reviewable extension wiring", "Plugins", "3"} {
		if !strings.Contains(parseMarkup, parseExpected) {
			parseT.Fatalf("ExamplePage markup missing %q\n%s", parseExpected, parseMarkup)
		}
	}

	parseCode, parseErr := ui.RenderToString(ExampleCode("line one", "line two"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExampleCode) error = %v", parseErr)
	}
	if !strings.Contains(parseCode, "line one") || !strings.Contains(parseCode, "line two") {
		parseT.Fatalf("ExampleCode markup = %q, want both lines", parseCode)
	}

	parseButton, parseErr := ui.RenderToString(ExampleButton("Inspect", ui.WrapHandler("click")))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExampleButton) error = %v", parseErr)
	}
	if !strings.Contains(parseButton, "Inspect") {
		parseT.Fatalf("ExampleButton markup = %q, want label", parseButton)
	}
}

func TestExampleCopyCoversAdditionalSubjectFamilies(parseT *testing.T) {
	parseTests := []struct {
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

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.feature, func(parseT2 *testing.T) {
			parseOverviewLead, parseOverviewBullets := overviewCopy(parseTest.title, parseTest.feature)
			parseFunctionalBullets := functionalCopy(parseTest.title, parseTest.feature)
			parseImplementationLead, parseImplementationBullets, parseCodeLines := implementationCopy(parseTest.title, parseTest.feature)

			parseAllText := strings.ToLower(strings.Join(append(append([]string{parseOverviewLead, parseImplementationLead}, parseOverviewBullets...), append(parseFunctionalBullets, parseImplementationBullets...)...), " "))
			if !strings.Contains(parseAllText, parseTest.want) {
				parseT2.Fatalf("combined example copy for %q missing %q\n%s", parseTest.feature, parseTest.want, parseAllText)
			}
			if len(parseCodeLines) == 0 {
				parseT2.Fatalf("implementationCopy(%q) returned no code lines", parseTest.feature)
			}
		})
	}
}

func TestExampleDocumentationAdditionalBranches(parseT *testing.T) {
	parseTests := []struct {
		title   string
		feature string
		want    string
	}{
		{title: "Runtime Inspector", feature: "devtools.OpenPanel", want: "instrumentation"},
		{title: "Server Render", feature: "ui.RenderToString", want: "server-side html generation"},
		{title: "Bootstrap Resume", feature: "ui.RenderBootstrapScript, ui.ReadBootstrapScript", want: "existing html already being in the document"},
		{title: "Resume Existing DOM", feature: "router.HydrateMount", want: "existing html already being in the document"},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.feature, func(parseT2 *testing.T) {
			parseLead, parseBullets, parseCode := implementationCopy(parseTest.title, parseTest.feature)
			parseAllText := strings.ToLower(strings.Join(append([]string{parseLead}, parseBullets...), " "))
			if !strings.Contains(parseAllText, parseTest.want) {
				parseT2.Fatalf("implementationCopy(%q) missing %q\n%s", parseTest.feature, parseTest.want, parseAllText)
			}
			if len(parseCode) == 0 {
				parseT2.Fatalf("implementationCopy(%q) returned no code lines", parseTest.feature)
			}
		})
	}

	parseBulletsMarkup, parseErr := ui.RenderToString(ExampleBulletList("first", "", "second"))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExampleBulletList) error = %v", parseErr)
	}
	if strings.Contains(parseBulletsMarkup, "<li></li>") || !strings.Contains(parseBulletsMarkup, "first") || !strings.Contains(parseBulletsMarkup, "second") {
		parseT.Fatalf("ExampleBulletList markup = %q, want trimmed non-empty bullets", parseBulletsMarkup)
	}

	parsePanelMarkup, parseErr := ui.RenderToString(ExamplePanel("Implementation", ExampleStat("Coverage", "80%")))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(ExamplePanel) error = %v", parseErr)
	}
	if !strings.Contains(parsePanelMarkup, "Implementation") || !strings.Contains(parsePanelMarkup, "Coverage") {
		parseT.Fatalf("ExamplePanel markup = %q, want title and content", parsePanelMarkup)
	}
}
