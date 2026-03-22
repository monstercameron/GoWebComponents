package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func readImportFixture(t *testing.T, name string) []byte {
	t.Helper()
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	fixturePath := filepath.Join(repoRoot, "test", "gwc-import-fixtures", name)
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read import fixture %q: %v", fixturePath, err)
	}
	return content
}

func TestParseImportedHTMLDocumentComplex(t *testing.T) {
	document, err := parseImportedDocument("catalog.html", readImportFixture(t, "catalog.html"))
	if err != nil {
		t.Fatalf("parse imported html: %v", err)
	}
	if document.SourceKind != "html" {
		t.Fatalf("expected html source kind, got %q", document.SourceKind)
	}
	if document.Title != "Complex Catalog" {
		t.Fatalf("expected html title, got %q", document.Title)
	}
	if document.Lang != "en-GB" {
		t.Fatalf("expected html lang, got %q", document.Lang)
	}
	if len(document.HeadNodes) != 4 {
		t.Fatalf("expected preserved head nodes, got %d", len(document.HeadNodes))
	}
	if len(document.BodyAttrs) != 6 {
		t.Fatalf("expected body attrs, got %d", len(document.BodyAttrs))
	}
	if len(document.Roots) != 3 {
		t.Fatalf("expected three body roots, got %d", len(document.Roots))
	}
	if document.Roots[0].Tag != "header" || document.Roots[1].Tag != "main" || document.Roots[2].Tag != "footer" {
		t.Fatalf("unexpected root tags: %#v", document.Roots)
	}

	mainGo, err := renderImportedMain(document, "github.com/monstercameron/GoWebComponents")
	if err != nil {
		t.Fatalf("render imported main: %v", err)
	}
	for _, expected := range []string{"html.Header(", "html.Nav(", "html.Article(", "html.Footer(", "html.Tag(", "\"price-badge\"", "\"inventory-pill\"", "\"promo-callout\"", "Data: map[string]string{", "\"surface\": \"masthead\"", "Hidden: true", "Disabled: true"} {
		if !strings.Contains(mainGo, expected) {
			t.Fatalf("expected rendered main.go to contain %q\n%s", expected, mainGo)
		}
	}

	indexHTML, err := renderImportedIndexHTML(startSelection{ProjectName: "catalog-import"}, document)
	if err != nil {
		t.Fatalf("render imported index.html: %v", err)
	}
	for _, expected := range []string{"<html lang=\"en-GB\">", "<meta name=\"theme-color\" content=\"#101820\">", "<meta name=\"description\" content=\"Imported merchandising experience\">", "<link rel=\"preload\" href=\"/hero.png\" as=\"image\">", "<body class=\"shell app-shell\" data-view=\"catalog\" data-region=\"eu\" aria-live=\"polite\" aria-busy=\"false\" style=\"background: #101820; color: #f6f8fb; min-height: 100vh\">", "gwc-import-root", "boot-error"} {
		if !strings.Contains(indexHTML, expected) {
			t.Fatalf("expected rendered index.html to contain %q\n%s", expected, indexHTML)
		}
	}
}

func TestParseImportedJSXDocumentComplex(t *testing.T) {
	document, err := parseImportedDocument("catalog.jsx", readImportFixture(t, "catalog.jsx"))
	if err != nil {
		t.Fatalf("parse imported jsx: %v", err)
	}
	if document.SourceKind != "jsx" {
		t.Fatalf("expected jsx source kind, got %q", document.SourceKind)
	}
	if len(document.Roots) != 1 || document.Roots[0].Tag != "main" {
		t.Fatalf("expected one main root, got %#v", document.Roots)
	}

	mainGo, err := renderImportedMain(document, "github.com/monstercameron/GoWebComponents")
	if err != nil {
		t.Fatalf("render imported main: %v", err)
	}
	for _, expected := range []string{"html.Main(", "html.Header(", "html.Nav(", "html.Article(", "html.Footer(", "Data: map[string]string{", "\"view\": \"catalog\"", "\"region\": \"eu\"", "\"surface\": \"masthead\"", "\"track\": \"featured\"", "Aria: map[string]string{", "\"live\": \"polite\"", "\"busy\": \"false\"", "Style: map[string]string{", "\"background-color\": \"#101820\"", "\"color\": \"#f6f8fb\"", "\"padding-top\": \"24\"", "\"min-height\": \"100vh\"", "Checked: true", "html.Text(\"42\")", "html.Tag(", "\"price-badge\"", "\"inventory-pill\"", "\"marketing-card\"", "\"promo-callout\"", "\"priority\": 2", "\"tone\": \"sale\"", "\"featured\": true"} {
		if !strings.Contains(mainGo, expected) {
			t.Fatalf("expected rendered main.go to contain %q\n%s", expected, mainGo)
		}
	}
}

func TestParseImportedJSXRejectsDynamicExpressions(t *testing.T) {
	_, err := parseImportedDocument("dynamic.jsx", readImportFixture(t, "dynamic.jsx"))
	if err == nil || !strings.Contains(err.Error(), "unsupported JSX child expression") {
		t.Fatalf("expected dynamic JSX expression rejection, got %v", err)
	}
}

func TestRunImportWritesSingleMainGoForHTML(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	sourcePath := filepath.Join(repoRoot, "test", "gwc-import-fixtures", "catalog.html")
	projectDir := createImportBuildProject(t, repoRoot, "example.com/catalog-import")
	outputPath := filepath.Join(projectDir, "bin", "main.go")

	launcher := launcher{repoRoot: repoRoot}
	if err := launcher.run([]string{"import", "-src", sourcePath, "-out", outputPath}); err != nil {
		t.Fatalf("run import html: %v", err)
	}
	tidyImportBuildProject(t, projectDir)

	for _, path := range []string{
		outputPath,
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected generated file %q: %v", path, err)
		}
	}
	for _, unexpected := range []string{
		filepath.Join(projectDir, "index.html"),
		filepath.Join(projectDir, "gwc-start.json"),
		filepath.Join(projectDir, "README.md"),
		filepath.Join(projectDir, "main.go"),
	} {
		if _, err := os.Stat(unexpected); err == nil {
			t.Fatalf("expected import to avoid generating %q", unexpected)
		}
	}

	mainBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read generated main.go: %v", err)
	}
	for _, expected := range []string{"html.Fragment(", "html.Main(", "html.Nav(", "html.Footer(", "html.Tag(", "\"price-badge\"", "\"promo-callout\"", "html.Text(\"Catalog\")", "html.Text(\"Sold out\")", "html.Text(\"Terms apply.\")"} {
		if !strings.Contains(string(mainBytes), expected) {
			t.Fatalf("expected generated main.go to contain %q\n%s", expected, string(mainBytes))
		}
	}

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(projectDir, "bin", "main.wasm"), "./bin")
	buildCmd.Dir = projectDir
	buildCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected imported html project to build, got %v\n%s", err, string(output))
	}
}

func TestRunImportWritesSingleMainGoForJSX(t *testing.T) {
	repoRoot, err := resolveRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	sourcePath := filepath.Join(repoRoot, "test", "gwc-import-fixtures", "catalog.jsx")
	projectDir := createImportBuildProject(t, repoRoot, "example.com/catalog-jsx-import")
	outputPath := filepath.Join(projectDir, "bin", "main.go")

	launcher := launcher{repoRoot: repoRoot}
	if err := launcher.run([]string{"import", "-src", sourcePath, "-out", outputPath}); err != nil {
		t.Fatalf("run import jsx: %v", err)
	}
	tidyImportBuildProject(t, projectDir)

	mainBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read generated main.go: %v", err)
	}
	for _, expected := range []string{"\"background-color\": \"#101820\"", "\"color\":            \"#f6f8fb\"", "\"padding-top\":      \"24\"", "\"min-height\":       \"100vh\"", "\"priority\": 2", "\"featured\": true", "html.Tag(", "\"price-badge\"", "\"inventory-pill\"", "\"marketing-card\"", "\"promo-callout\"", "html.Footer("} {
		if !strings.Contains(string(mainBytes), expected) {
			t.Fatalf("expected jsx import main.go to contain %q\n%s", expected, string(mainBytes))
		}
	}
	for _, unexpected := range []string{
		filepath.Join(projectDir, "index.html"),
		filepath.Join(projectDir, "gwc-start.json"),
		filepath.Join(projectDir, "README.md"),
		filepath.Join(projectDir, "main.go"),
	} {
		if _, err := os.Stat(unexpected); err == nil {
			t.Fatalf("expected import to avoid generating %q", unexpected)
		}
	}

	buildCmd := exec.Command("go", "build", "-o", filepath.Join(projectDir, "bin", "main.wasm"), "./bin")
	buildCmd.Dir = projectDir
	buildCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("expected imported jsx project to build, got %v\n%s", err, string(output))
	}
}

func createImportBuildProject(t *testing.T, repoRoot string, modulePath string) string {
	t.Helper()
	projectDir := t.TempDir()
	goMod := "module " + modulePath + "\n\ngo 1.25.0\n\nrequire github.com/monstercameron/GoWebComponents v0.0.0\n\nreplace github.com/monstercameron/GoWebComponents => " + filepath.ToSlash(repoRoot) + "\n"
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatalf("write import build go.mod: %v", err)
	}
	if goSumBytes, err := os.ReadFile(filepath.Join(repoRoot, "go.sum")); err == nil {
		if writeErr := os.WriteFile(filepath.Join(projectDir, "go.sum"), goSumBytes, 0644); writeErr != nil {
			t.Fatalf("write import build go.sum: %v", writeErr)
		}
	}
	return projectDir
}

func tidyImportBuildProject(t *testing.T, projectDir string) {
	t.Helper()
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = projectDir
	tidyOutput, err := tidyCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("tidy import build project: %v\n%s", err, string(tidyOutput))
	}
}

func TestCreateLauncherTempDirUsesBinTmp(t *testing.T) {
	root := t.TempDir()
	dir, err := createLauncherTempDir(root, "gwc-test-")
	if err != nil {
		t.Fatalf("create launcher temp dir: %v", err)
	}
	if !strings.HasPrefix(dir, filepath.Join(root, "bin", "tmp")+string(os.PathSeparator)) {
		t.Fatalf("expected temp dir under bin/tmp, got %q", dir)
	}
	if _, err := os.Stat(filepath.Join(root, "bin", "tmp")); err != nil {
		t.Fatalf("expected bin/tmp directory to exist: %v", err)
	}
}

func TestCreateLauncherTempDirUsesArtifactRootOverride(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); err != nil {
		t.Fatalf("write gwc-runner.json: %v", err)
	}
	dir, err := createLauncherTempDir(root, "gwc-test-")
	if err != nil {
		t.Fatalf("create launcher temp dir: %v", err)
	}
	wantPrefix := filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "tmp") + string(os.PathSeparator)
	if !strings.HasPrefix(dir, wantPrefix) {
		t.Fatalf("expected temp dir under artifact-root tmp %q, got %q", wantPrefix, dir)
	}
	if _, err := os.Stat(filepath.Join(root, "enterprise-artifacts", filepath.Base(root), "tmp")); err != nil {
		t.Fatalf("expected artifact-root tmp directory to exist: %v", err)
	}
}
