package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestAgenticModelSearchAndImpact(parseT *testing.T) {
	parseRoot := writeAgenticReadFixture(parseT)

	parseManifest, parseErr := buildAgenticModelManifest(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("build model manifest: %v", parseErr)
	}
	if !parseManifest.OK {
		parseT.Fatalf("expected model manifest ok, got diagnostics: %#v", parseManifest.Diagnostics)
	}
	if !agenticComponentNames(parseManifest.Components)["Card"] || !agenticComponentNames(parseManifest.Components)["Dashboard"] {
		parseT.Fatalf("expected Card and Dashboard components, got %#v", parseManifest.Components)
	}
	parseDashboard := agenticComponentByName(parseManifest.Components, "Dashboard")
	if !slices.Contains(parseDashboard.Hooks, "ui.UseState") {
		parseT.Fatalf("expected Dashboard hook use, got %#v", parseDashboard.Hooks)
	}
	if !slices.Contains(parseDashboard.AtomReads, "dashboard.count") {
		parseT.Fatalf("expected Dashboard atom read, got %#v", parseDashboard.AtomReads)
	}

	parseSearch, parseErr := buildAgenticSearchReport(agenticSearchConfig{rootPath: parseRoot, query: "dashboard component", limit: 5})
	if parseErr != nil {
		parseT.Fatalf("build search report: %v", parseErr)
	}
	if parseSearch.Count == 0 || parseSearch.Results[0].Name != "Dashboard" {
		parseT.Fatalf("expected Dashboard as top search result, got %#v", parseSearch.Results)
	}

	parseImpact, parseErr := buildAgenticImpactReport(parseRoot, "Card")
	if parseErr != nil {
		parseT.Fatalf("build impact report: %v", parseErr)
	}
	if !agenticImpactNames(parseImpact.DirectDependents)["Dashboard"] {
		parseT.Fatalf("expected Dashboard direct dependent, got %#v", parseImpact.DirectDependents)
	}
	if !slices.Contains(parseImpact.Tests, "app_test.go") {
		parseT.Fatalf("expected app_test.go impact surface, got %#v", parseImpact.Tests)
	}
	if !slices.Contains(parseImpact.Docs, "README.md") {
		parseT.Fatalf("expected README.md impact surface, got %#v", parseImpact.Docs)
	}
}

func TestAgenticRenderRunsSSRForFixtureComponent(parseT *testing.T) {
	parseRoot := writeAgenticReadFixture(parseT)

	parseReport := buildAgenticRenderReport(agenticRenderConfig{
		rootPath:  parseRoot,
		component: "Card",
		propsJSON: `{"Label":"Hello SSR"}`,
	})
	if !parseReport.OK {
		parseT.Fatalf("expected render ok, got %#v", parseReport)
	}
	if !strings.Contains(parseReport.HTML, "Hello SSR") {
		parseT.Fatalf("expected rendered HTML to contain props label, got %q", parseReport.HTML)
	}
}

func TestAgenticExplainAndProbeReports(parseT *testing.T) {
	parseExplain := buildAgenticExplainReport("", "Forms & accessibility")
	if !parseExplain.OK || parseExplain.Kind != "capability" || parseExplain.Capability == nil {
		parseT.Fatalf("expected capability explanation, got %#v", parseExplain)
	}

	parseOriginalProbe := agenticRunBrowserProbe
	agenticRunBrowserProbe = func(parseConfig agenticProbeConfig) agenticProbeReport {
		return agenticProbeReport{OK: true, Target: parseConfig.target, URL: parseConfig.url, Status: 200, Title: "fixture"}
	}
	defer func() { agenticRunBrowserProbe = parseOriginalProbe }()

	parseConfig, parseErr := resolveAgenticProbeConfig(agenticProbeConfig{
		target:  "counter",
		url:     "http://127.0.0.1:8090",
		timeout: time.Second,
	})
	if parseErr != nil {
		parseT.Fatalf("resolve probe config: %v", parseErr)
	}
	parseReport := buildAgenticProbeReport(parseConfig)
	if !parseReport.OK || parseReport.URL != "http://127.0.0.1:8090/examples/public/counter/" {
		parseT.Fatalf("unexpected probe report: %#v", parseReport)
	}
}

func TestRunModelOutputsJSON(parseT *testing.T) {
	parseRoot := writeAgenticReadFixture(parseT)
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr = (launcher{}).runModel([]string{"-root", parseRoot, "-json"}); parseErr != nil {
		parseT.Fatalf("run model: %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	var parseManifest agenticModelManifest
	if parseErr = json.Unmarshal([]byte(parseOutput), &parseManifest); parseErr != nil {
		parseT.Fatalf("decode model JSON: %v\n%s", parseErr, parseOutput)
	}
	if len(parseManifest.Components) != 2 {
		parseT.Fatalf("expected two components, got %#v", parseManifest.Components)
	}
}

func writeAgenticReadFixture(parseT *testing.T) string {
	parseT.Helper()
	parseRoot := parseT.TempDir()
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseFiles := map[string]string{
		"go.mod": "module example.com/agenticfixture\n\ngo 1.26.0\n\nrequire github.com/monstercameron/GoWebComponents/v4 v4.0.0\n\nreplace github.com/monstercameron/GoWebComponents/v4 => " + filepath.ToSlash(parseRepoRoot) + "\n",
		"app.go": `package agenticfixture

import (
	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/state"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

// CardProps configures Card.
type CardProps struct {
	Label string
}

// Card renders one labeled card.
func Card(parseProps CardProps) ui.Node {
	return html.Div(html.Props{}, html.Text(parseProps.Label))
}

// Dashboard renders the Card component for a route.
func Dashboard() ui.Node {
	_ = ui.UseState(0)
	_ = state.UseAtom[int]("dashboard.count", 0)
	return html.Div(html.Props{}, ui.CreateElement(Card, CardProps{Label: "Dashboard"}))
}
`,
		"routes.go": `package agenticfixture

// registerRoutes documents the /dashboard route.
func registerRoutes() {
	// router.Register("/dashboard", Dashboard)
}
`,
		"app_test.go": `package agenticfixture

import "testing"

func TestCardSurface(parseT *testing.T) {
	_ = Card
}
`,
		"README.md": "The Card surface is documented here.\n",
	}
	for parsePath, parseContent := range parseFiles {
		parseFullPath := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr = os.MkdirAll(filepath.Dir(parseFullPath), 0755); parseErr != nil {
			parseT.Fatalf("mkdir %s: %v", parsePath, parseErr)
		}
		if parseErr = os.WriteFile(parseFullPath, []byte(parseContent), 0644); parseErr != nil {
			parseT.Fatalf("write %s: %v", parsePath, parseErr)
		}
	}
	parseGoSum, parseErr := os.ReadFile(filepath.Join(parseRepoRoot, "go.sum"))
	if parseErr != nil {
		parseT.Fatalf("read repo go.sum: %v", parseErr)
	}
	if parseErr = os.WriteFile(filepath.Join(parseRoot, "go.sum"), parseGoSum, 0644); parseErr != nil {
		parseT.Fatalf("write fixture go.sum: %v", parseErr)
	}
	parseTidy := exec.Command("go", "mod", "tidy")
	parseTidy.Dir = parseRoot
	if parseOutput, parseTidyErr := parseTidy.CombinedOutput(); parseTidyErr != nil {
		parseT.Fatalf("tidy fixture module: %v\n%s", parseTidyErr, strings.TrimSpace(string(parseOutput)))
	}
	return parseRoot
}

func agenticComponentNames(parseComponents []agenticComponent) map[string]bool {
	parseNames := map[string]bool{}
	for _, parseComponent := range parseComponents {
		parseNames[parseComponent.Name] = true
	}
	return parseNames
}

func agenticComponentByName(parseComponents []agenticComponent, parseName string) agenticComponent {
	for _, parseComponent := range parseComponents {
		if parseComponent.Name == parseName {
			return parseComponent
		}
	}
	return agenticComponent{}
}

func agenticImpactNames(parseTargets []agenticImpactTarget) map[string]bool {
	parseNames := map[string]bool{}
	for _, parseTarget := range parseTargets {
		parseNames[parseTarget.Name] = true
	}
	return parseNames
}
