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

func TestExampleCopyCoversRemainingSpecializedBranches(parseT *testing.T) {
	parseTests := []struct {
		name               string
		title              string
		feature            string
		wantOverview       string
		wantFunctional     string
		wantImplementation string
		wantCode           string
	}{
		{
			name:               "use-effect",
			title:              "Effect Demo",
			feature:            "ui.UseEffect",
			wantOverview:       "synchronize with work outside the pure render path",
			wantFunctional:     "smallest hook or primitive",
			wantImplementation: "hook state, and event handlers",
			wantCode:           "ui.Render",
		},
		{
			name:               "use-reducer",
			title:              "Reducer Demo",
			feature:            "ui.UseReducer",
			wantOverview:       "named transitions",
			wantFunctional:     "drive the same state machine",
			wantImplementation: "single transition table",
			wantCode:           "workflow.Dispatch",
		},
		{
			name:               "render",
			title:              "Client Mount",
			feature:            "ui.Render",
			wantOverview:       "mount a component tree",
			wantFunctional:     "smallest hook or primitive",
			wantImplementation: "client entrypoint",
			wantCode:           "ui.Render(ui.CreateElement",
		},
		{
			name:               "hydrate",
			title:              "Hydration",
			feature:            "ui.Hydrate",
			wantOverview:       "matching html already exists",
			wantFunctional:     "smallest hook or primitive",
			wantImplementation: "existing html already being in the document",
			wantCode:           "ui.Hydrate",
		},
		{
			name:               "state-atom",
			title:              "Shared Atom",
			feature:            "state.UseAtom",
			wantOverview:       "shared values",
			wantFunctional:     "shared values",
			wantImplementation: "stable atom ids",
			wantCode:           "state.UseAtom",
		},
		{
			name:               "fetch-resource",
			title:              "Resource Loader",
			feature:            "fetch.UseResource",
			wantOverview:       "load, cache, retry",
			wantFunctional:     "loading, ready, retry",
			wantImplementation: "async lifecycle",
			wantCode:           "fetch.UseResource",
		},
		{
			name:               "html-helper",
			title:              "Semantic Markup",
			feature:            "html.Div",
			wantOverview:       "semantic dom construction helpers",
			wantFunctional:     "dom shape",
			wantImplementation: "typed html helpers",
			wantCode:           "ui.Render",
		},
		{
			name:               "devtools",
			title:              "Runtime Inspector",
			feature:            "devtools.OpenPanel",
			wantOverview:       "runtime inspection",
			wantFunctional:     "observability",
			wantImplementation: "instrumentation cost",
			wantCode:           "ui.Render",
		},
		{
			name:               "generic-default",
			title:              "Generic Example",
			feature:            "Feature",
			wantOverview:       "understand the purpose",
			wantFunctional:     "functional result",
			wantImplementation: "focused go js/wasm package",
			wantCode:           "ui.Render(ui.CreateElement",
		},
	}

	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			parseOverviewLead, parseOverviewBullets := overviewCopy(parseTest.title, parseTest.feature)
			parseFunctionalBullets := functionalCopy(parseTest.title, parseTest.feature)
			parseImplementationLead, parseImplementationBullets, parseCodeLines := implementationCopy(parseTest.title, parseTest.feature)

			if !strings.Contains(strings.ToLower(parseOverviewLead+" "+strings.Join(parseOverviewBullets, " ")), strings.ToLower(parseTest.wantOverview)) {
				parseT2.Fatalf("overviewCopy(%q) missing %q\n%s\n%v", parseTest.feature, parseTest.wantOverview, parseOverviewLead, parseOverviewBullets)
			}
			if !strings.Contains(strings.ToLower(strings.Join(parseFunctionalBullets, " ")), strings.ToLower(parseTest.wantFunctional)) {
				parseT2.Fatalf("functionalCopy(%q) missing %q\n%v", parseTest.feature, parseTest.wantFunctional, parseFunctionalBullets)
			}
			if !strings.Contains(strings.ToLower(parseImplementationLead+" "+strings.Join(parseImplementationBullets, " ")), strings.ToLower(parseTest.wantImplementation)) {
				parseT2.Fatalf("implementationCopy(%q) missing %q\n%s\n%v", parseTest.feature, parseTest.wantImplementation, parseImplementationLead, parseImplementationBullets)
			}
			if !strings.Contains(strings.Join(parseCodeLines, "\n"), parseTest.wantCode) {
				parseT2.Fatalf("implementationCopy(%q) code missing %q\n%v", parseTest.feature, parseTest.wantCode, parseCodeLines)
			}
		})
	}
}
