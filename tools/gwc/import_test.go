package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func readImportFixture(parseT *testing.T, parseName string) []byte {
	parseT.Helper()
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseFixturePath := filepath.Join(parseRepoRoot, "test", "gwc-import-fixtures", parseName)
	parseContent, parseErr := os.ReadFile(parseFixturePath)
	if parseErr != nil {
		parseT.Fatalf("read import fixture %q: %v", parseFixturePath, parseErr)
	}
	return parseContent
}

func TestParseImportedHTMLDocumentComplex(parseT *testing.T) {
	parseDocument, parseErr := parseImportedDocument("catalog.html", readImportFixture(parseT, "catalog.html"))
	if parseErr != nil {
		parseT.Fatalf("parse imported html: %v", parseErr)
	}
	if parseDocument.SourceKind != "html" {
		parseT.Fatalf("expected html source kind, got %q", parseDocument.SourceKind)
	}
	if parseDocument.Title != "Complex Catalog" {
		parseT.Fatalf("expected html title, got %q", parseDocument.Title)
	}
	if parseDocument.Lang != "en-GB" {
		parseT.Fatalf("expected html lang, got %q", parseDocument.Lang)
	}
	if len(parseDocument.HeadNodes) != 4 {
		parseT.Fatalf("expected preserved head nodes, got %d", len(parseDocument.HeadNodes))
	}
	if len(parseDocument.BodyAttrs) != 6 {
		parseT.Fatalf("expected body attrs, got %d", len(parseDocument.BodyAttrs))
	}
	if len(parseDocument.Roots) != 3 {
		parseT.Fatalf("expected three body roots, got %d", len(parseDocument.Roots))
	}
	if parseDocument.Roots[0].Tag != "header" || parseDocument.Roots[1].Tag != "main" || parseDocument.Roots[2].Tag != "footer" {
		parseT.Fatalf("unexpected root tags: %#v", parseDocument.Roots)
	}

	parseMainGo, parseErr := renderImportedMain(parseDocument, "github.com/monstercameron/GoWebComponents")
	if parseErr != nil {
		parseT.Fatalf("render imported main: %v", parseErr)
	}
	for _, parseExpected := range []string{"html.Header(", "html.Nav(", "html.Article(", "html.Footer(", "html.Tag(", "\"price-badge\"", "\"inventory-pill\"", "\"promo-callout\"", "Data: map[string]string{", "\"surface\": \"masthead\"", "Hidden: true", "Disabled: true"} {
		if !strings.Contains(parseMainGo, parseExpected) {
			parseT.Fatalf("expected rendered main.go to contain %q\n%s", parseExpected, parseMainGo)
		}
	}

	parseIndexHTML, parseErr := renderImportedIndexHTML(startSelection{ProjectName: "catalog-import"}, parseDocument)
	if parseErr != nil {
		parseT.Fatalf("render imported index.html: %v", parseErr)
	}
	for _, parseExpected2 := range []string{"<html lang=\"en-GB\">", "<meta name=\"theme-color\" content=\"#101820\">", "<meta name=\"description\" content=\"Imported merchandising experience\">", "<link rel=\"preload\" href=\"/hero.png\" as=\"image\">", "<body class=\"shell app-shell\" data-view=\"catalog\" data-region=\"eu\" aria-live=\"polite\" aria-busy=\"false\" style=\"background: #101820; color: #f6f8fb; min-height: 100vh\">", "gwc-import-root", "boot-error"} {
		if !strings.Contains(parseIndexHTML, parseExpected2) {
			parseT.Fatalf("expected rendered index.html to contain %q\n%s", parseExpected2, parseIndexHTML)
		}
	}
}

func TestParseImportedJSXDocumentComplex(parseT *testing.T) {
	parseDocument, parseErr := parseImportedDocument("catalog.jsx", readImportFixture(parseT, "catalog.jsx"))
	if parseErr != nil {
		parseT.Fatalf("parse imported jsx: %v", parseErr)
	}
	if parseDocument.SourceKind != "jsx" {
		parseT.Fatalf("expected jsx source kind, got %q", parseDocument.SourceKind)
	}
	if len(parseDocument.Roots) != 1 || parseDocument.Roots[0].Tag != "main" {
		parseT.Fatalf("expected one main root, got %#v", parseDocument.Roots)
	}

	parseMainGo, parseErr := renderImportedMain(parseDocument, "github.com/monstercameron/GoWebComponents")
	if parseErr != nil {
		parseT.Fatalf("render imported main: %v", parseErr)
	}
	for _, parseExpected := range []string{"html.Main(", "html.Header(", "html.Nav(", "html.Article(", "html.Footer(", "Data: map[string]string{", "\"view\": \"catalog\"", "\"region\": \"eu\"", "\"surface\": \"masthead\"", "\"track\": \"featured\"", "Aria: map[string]string{", "\"live\": \"polite\"", "\"busy\": \"false\"", "Style: map[string]string{", "\"background-color\": \"#101820\"", "\"color\": \"#f6f8fb\"", "\"padding-top\": \"24\"", "\"min-height\": \"100vh\"", "Checked: true", "html.Text(\"42\")", "html.Tag(", "\"price-badge\"", "\"inventory-pill\"", "\"marketing-card\"", "\"promo-callout\"", "\"priority\": 2", "\"tone\": \"sale\"", "\"featured\": true"} {
		if !strings.Contains(parseMainGo, parseExpected) {
			parseT.Fatalf("expected rendered main.go to contain %q\n%s", parseExpected, parseMainGo)
		}
	}
}

func TestParseImportedJSXRejectsDynamicExpressions(parseT *testing.T) {
	_, parseErr := parseImportedDocument("dynamic.jsx", readImportFixture(parseT, "dynamic.jsx"))
	if parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported JSX child expression") {
		parseT.Fatalf("expected dynamic JSX expression rejection, got %v", parseErr)
	}
}

func TestRunImportWritesSingleMainGoForHTML(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseSourcePath := filepath.Join(parseRepoRoot, "test", "gwc-import-fixtures", "catalog.html")
	parseProjectDir := createImportBuildProject(parseT, parseRepoRoot, "example.com/catalog-import")
	parseOutputPath := filepath.Join(parseProjectDir, "bin", "main.go")

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	if parseErr2 := parseLauncher.run([]string{"import", "-src", parseSourcePath, "-out", parseOutputPath}); parseErr2 != nil {
		parseT.Fatalf("run import html: %v", parseErr2)
	}
	tidyImportBuildProject(parseT, parseProjectDir)

	for _, parsePath := range []string{
		parseOutputPath,
	} {
		if _, parseErr3 := os.Stat(parsePath); parseErr3 != nil {
			parseT.Fatalf("expected generated file %q: %v", parsePath, parseErr3)
		}
	}
	for _, parseUnexpected := range []string{
		filepath.Join(parseProjectDir, "index.html"),
		filepath.Join(parseProjectDir, "gwc-start.json"),
		filepath.Join(parseProjectDir, "README.md"),
		filepath.Join(parseProjectDir, "main.go"),
	} {
		if _, parseErr4 := os.Stat(parseUnexpected); parseErr4 == nil {
			parseT.Fatalf("expected import to avoid generating %q", parseUnexpected)
		}
	}

	parseMainBytes, parseErr := os.ReadFile(parseOutputPath)
	if parseErr != nil {
		parseT.Fatalf("read generated main.go: %v", parseErr)
	}
	for _, parseExpected := range []string{"html.Fragment(", "html.Main(", "html.Nav(", "html.Footer(", "html.Tag(", "\"price-badge\"", "\"promo-callout\"", "html.Text(\"Catalog\")", "html.Text(\"Sold out\")", "html.Text(\"Terms apply.\")"} {
		if !strings.Contains(string(parseMainBytes), parseExpected) {
			parseT.Fatalf("expected generated main.go to contain %q\n%s", parseExpected, string(parseMainBytes))
		}
	}

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(parseProjectDir, "bin", "main.wasm"), "./bin")
	buildCmd.Dir = parseProjectDir
	buildCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseOutput, parseErr := buildCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("expected imported html project to build, got %v\n%s", parseErr, string(parseOutput))
	}
}

func TestRunImportWritesSingleMainGoForJSX(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseSourcePath := filepath.Join(parseRepoRoot, "test", "gwc-import-fixtures", "catalog.jsx")
	parseProjectDir := createImportBuildProject(parseT, parseRepoRoot, "example.com/catalog-jsx-import")
	parseOutputPath := filepath.Join(parseProjectDir, "bin", "main.go")

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	if parseErr2 := parseLauncher.run([]string{"import", "-src", parseSourcePath, "-out", parseOutputPath}); parseErr2 != nil {
		parseT.Fatalf("run import jsx: %v", parseErr2)
	}
	tidyImportBuildProject(parseT, parseProjectDir)

	parseMainBytes, parseErr := os.ReadFile(parseOutputPath)
	if parseErr != nil {
		parseT.Fatalf("read generated main.go: %v", parseErr)
	}
	for _, parseExpected := range []string{"\"background-color\": \"#101820\"", "\"color\":            \"#f6f8fb\"", "\"padding-top\":      \"24\"", "\"min-height\":       \"100vh\"", "\"priority\": 2", "\"featured\": true", "html.Tag(", "\"price-badge\"", "\"inventory-pill\"", "\"marketing-card\"", "\"promo-callout\"", "html.Footer("} {
		if !strings.Contains(string(parseMainBytes), parseExpected) {
			parseT.Fatalf("expected jsx import main.go to contain %q\n%s", parseExpected, string(parseMainBytes))
		}
	}
	for _, parseUnexpected := range []string{
		filepath.Join(parseProjectDir, "index.html"),
		filepath.Join(parseProjectDir, "gwc-start.json"),
		filepath.Join(parseProjectDir, "README.md"),
		filepath.Join(parseProjectDir, "main.go"),
	} {
		if _, parseErr3 := os.Stat(parseUnexpected); parseErr3 == nil {
			parseT.Fatalf("expected import to avoid generating %q", parseUnexpected)
		}
	}

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(parseProjectDir, "bin", "main.wasm"), "./bin")
	buildCmd.Dir = parseProjectDir
	buildCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseOutput, parseErr := buildCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("expected imported jsx project to build, got %v\n%s", parseErr, string(parseOutput))
	}
}

func createImportBuildProject(parseT *testing.T, parseRepoRoot string, parseModulePath string) string {
	parseT.Helper()
	parseProjectDir := parseT.TempDir()
	parseGoMod := "module " + parseModulePath + "\n\ngo 1.25.0\n\nrequire github.com/monstercameron/GoWebComponents v0.0.0\n\nreplace github.com/monstercameron/GoWebComponents => " + filepath.ToSlash(parseRepoRoot) + "\n"
	if parseErr := os.WriteFile(filepath.Join(parseProjectDir, "go.mod"), []byte(parseGoMod), 0644); parseErr != nil {
		parseT.Fatalf("write import build go.mod: %v", parseErr)
	}
	if parseGoSumBytes, parseErr2 := os.ReadFile(filepath.Join(parseRepoRoot, "go.sum")); parseErr2 == nil {
		if parseWriteErr := os.WriteFile(filepath.Join(parseProjectDir, "go.sum"), parseGoSumBytes, 0644); parseWriteErr != nil {
			parseT.Fatalf("write import build go.sum: %v", parseWriteErr)
		}
	}
	return parseProjectDir
}

func tidyImportBuildProject(parseT *testing.T, parseProjectDir string) {
	parseT.Helper()
	parseTidyCmd := exec.Command("go", "mod", "tidy")
	parseTidyCmd.Dir = parseProjectDir
	parseTidyOutput, parseErr := parseTidyCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("tidy import build project: %v\n%s", parseErr, string(parseTidyOutput))
	}
}

func TestCreateLauncherTempDirUsesBinTmp(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseDir, parseErr := createLauncherTempDir(parseRoot, "gwc-test-")
	if parseErr != nil {
		parseT.Fatalf("create launcher temp dir: %v", parseErr)
	}
	if !strings.HasPrefix(parseDir, filepath.Join(parseRoot, "bin", "tmp")+string(os.PathSeparator)) {
		parseT.Fatalf("expected temp dir under bin/tmp, got %q", parseDir)
	}
	if _, parseErr2 := os.Stat(filepath.Join(parseRoot, "bin", "tmp")); parseErr2 != nil {
		parseT.Fatalf("expected bin/tmp directory to exist: %v", parseErr2)
	}
}

func TestCreateLauncherTempDirUsesArtifactRootOverride(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write gwc-runner.json: %v", parseErr)
	}
	parseDir, parseErr2 := createLauncherTempDir(parseRoot, "gwc-test-")
	if parseErr2 != nil {
		parseT.Fatalf("create launcher temp dir: %v", parseErr2)
	}
	parseWantPrefix := filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "tmp") + string(os.PathSeparator)
	if !strings.HasPrefix(parseDir, parseWantPrefix) {
		parseT.Fatalf("expected temp dir under artifact-root tmp %q, got %q", parseWantPrefix, parseDir)
	}
	if _, parseErr3 := os.Stat(filepath.Join(parseRoot, "enterprise-artifacts", filepath.Base(parseRoot), "tmp")); parseErr3 != nil {
		parseT.Fatalf("expected artifact-root tmp directory to exist: %v", parseErr3)
	}
}

func TestImportHelperSourceKindAndDefaultProjectName(parseT *testing.T) {
	if parseKind, parseErr := detectImportSourceKind("landing.html"); parseErr != nil || parseKind != "html" {
		parseT.Fatalf("detect html source kind = %q err=%v", parseKind, parseErr)
	}
	if parseKind2, parseErr2 := detectImportSourceKind("landing.tsx"); parseErr2 != nil || parseKind2 != "jsx" {
		parseT.Fatalf("detect jsx source kind = %q err=%v", parseKind2, parseErr2)
	}
	if _, parseErr3 := detectImportSourceKind("landing.md"); parseErr3 == nil {
		parseT.Fatalf("expected unsupported extension error")
	}

	if parseGot := defaultImportedProjectName("  My Fancy_App!.html "); parseGot != "my-fancy-app" {
		parseT.Fatalf("default imported project name = %q, want my-fancy-app", parseGot)
	}
	if parseGot2 := defaultImportedProjectName("$$$"); parseGot2 != "imported-app" {
		parseT.Fatalf("default imported project fallback = %q, want imported-app", parseGot2)
	}
}

func TestImportHelperValueConversions(parseT *testing.T) {
	if parseGot := importedValueAsString(importedValue{Kind: importedValueBool, Bool: true}); parseGot != "true" {
		parseT.Fatalf("bool->string conversion = %q, want true", parseGot)
	}
	if parseGot2 := importedValueAsString(importedValue{Kind: importedValueNumber, Number: "42"}); parseGot2 != "42" {
		parseT.Fatalf("number->string conversion = %q, want 42", parseGot2)
	}
	parseStyleString := importedValueAsString(importedValue{Kind: importedValueStyle, Style: map[string]string{
		"color":       "#fff",
		"font-size":   "14px",
		"line-height": "1.4",
	}})
	for _, parseExpected := range []string{"color: #fff", "font-size: 14px", "line-height: 1.4"} {
		if !strings.Contains(parseStyleString, parseExpected) {
			parseT.Fatalf("style string %q missing %q", parseStyleString, parseExpected)
		}
	}

	if parseGot3 := importedValueAsNumber(importedValue{Kind: importedValueNumber, Number: "123"}); parseGot3 != "123" {
		parseT.Fatalf("number->number conversion = %q, want 123", parseGot3)
	}
	if parseGot4 := importedValueAsNumber(importedValue{Kind: importedValueString, String: " 77 "}); parseGot4 != "77" {
		parseT.Fatalf("string numeric conversion = %q, want 77", parseGot4)
	}
	if parseGot5 := importedValueAsNumber(importedValue{Kind: importedValueString, String: "7.7"}); parseGot5 != "" {
		parseT.Fatalf("non-integer numeric string conversion = %q, want empty", parseGot5)
	}

	if !importedValueAsBool(importedValue{Kind: importedValueBool, Bool: true}) {
		parseT.Fatalf("bool true conversion should be true")
	}
	if !importedValueAsBool(importedValue{Kind: importedValueString, String: ""}) {
		parseT.Fatalf("empty attribute string should convert to true")
	}
	if importedValueAsBool(importedValue{Kind: importedValueString, String: "false"}) {
		parseT.Fatalf("false string should convert to false")
	}
	if importedValueAsBool(importedValue{Kind: importedValueNull}) {
		parseT.Fatalf("null value should convert to false")
	}

	parseStyleMap := importedValueAsStyleMap(importedValue{Kind: importedValueString, String: "color: #111; font-weight: 600; ;"})
	if parseStyleMap["color"] != "#111" || parseStyleMap["font-weight"] != "600" {
		parseT.Fatalf("style map conversion mismatch: %#v", parseStyleMap)
	}
	if parseGot6 := importedValueAsStyleMap(importedValue{Kind: importedValueStyle, Style: map[string]string{"display": "grid"}}); parseGot6["display"] != "grid" {
		parseT.Fatalf("style map passthrough mismatch: %#v", parseGot6)
	}

	parseParsed := parseImportedStyleString("color: red; font-size: 16px; malformed;")
	if parseParsed["color"] != "red" || parseParsed["font-size"] != "16px" {
		parseT.Fatalf("parsed style map mismatch: %#v", parseParsed)
	}
}
