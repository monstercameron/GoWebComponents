package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParsePrerenderRoutesNormalizesValues verifies route normalization and validation.
func TestParsePrerenderRoutesNormalizesValues(t *testing.T) {
	parseRoutes, parseErr := parsePrerenderRoutes([]string{"/", "/docs/", "/docs"})
	if parseErr != nil {
		t.Fatalf("parse routes: %v", parseErr)
	}
	if len(parseRoutes) != 2 || parseRoutes[0] != "/" || parseRoutes[1] != "/docs" {
		t.Fatalf("unexpected normalized routes: %#v", parseRoutes)
	}
	_, parseErr = parsePrerenderRoutes([]string{"docs"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "must start with '/'") {
		t.Fatalf("expected invalid route error, got %v", parseErr)
	}
}

// TestRunPrerenderSkipBuildExportsRoutes verifies static export output in skip-build mode.
func TestRunPrerenderSkipBuildExportsRoutes(t *testing.T) {
	parseRoot := t.TempDir()
	parseOutDir := filepath.Join(parseRoot, "out")
	if writeErr := os.MkdirAll(parseOutDir, 0755); writeErr != nil {
		t.Fatalf("mkdir out: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/prerenderfixture\n\ngo 1.25\n"), 0644); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); writeErr != nil {
		t.Fatalf("write main.go: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<!doctype html><html><body>hello</body></html>"), 0644); writeErr != nil {
		t.Fatalf("write index.html: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseOutDir, "main.wasm"), []byte("wasm"), 0644); writeErr != nil {
		t.Fatalf("write main.wasm fixture: %v", writeErr)
	}
	if writeErr := os.MkdirAll(filepath.Join(parseRoot, "assets"), 0755); writeErr != nil {
		t.Fatalf("mkdir assets: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "assets", "site.css"), []byte("body{}"), 0644); writeErr != nil {
		t.Fatalf("write asset: %v", writeErr)
	}

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		t.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if runErr := (launcher{}).runPrerender([]string{
		"-root", parseRoot,
		"-app", "main.go",
		"-html", "index.html",
		"-out", parseOutDir,
		"-route", "/",
		"-route", "/docs",
		"-asset-dir", "assets",
		"-skip-build",
		"-json",
	}); runErr != nil {
		t.Fatalf("run prerender: %v", runErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		t.Fatalf("read prerender output: %v", parseOutputErr)
	}
	var parseSummary prerenderSummary
	if decodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); decodeErr != nil {
		t.Fatalf("decode prerender summary: %v\n%s", decodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.BuildExecuted {
		t.Fatalf("expected prerender skip-build summary, got %#v", parseSummary)
	}
	if !fileExists(filepath.Join(parseOutDir, "index.html")) || !fileExists(filepath.Join(parseOutDir, "docs", "index.html")) {
		t.Fatalf("expected prerendered route html files in %s", parseOutDir)
	}
	if !fileExists(filepath.Join(parseOutDir, "assets", "site.css")) {
		t.Fatalf("expected copied asset in export output")
	}
	if !fileExists(parseSummary.ManifestPath) {
		t.Fatalf("expected manifest path to exist: %s", parseSummary.ManifestPath)
	}
	if !fileExists(filepath.Join(parseOutDir, "wasm_exec.js")) {
		t.Fatalf("expected wasm_exec.js to be copied to output")
	}
}
