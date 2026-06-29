package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParsePrerenderRoutesNormalizesValues verifies route normalization and validation.
func TestParsePrerenderRoutesNormalizesValues(parseT *testing.T) {
	parseRoutes, parseErr := parsePrerenderRoutes([]string{"/", "/docs/", "/docs"})
	if parseErr != nil {
		parseT.Fatalf("parse routes: %v", parseErr)
	}
	if len(parseRoutes) != 2 || parseRoutes[0] != "/" || parseRoutes[1] != "/docs" {
		parseT.Fatalf("unexpected normalized routes: %#v", parseRoutes)
	}
	_, parseErr = parsePrerenderRoutes([]string{"docs"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "must start with '/'") {
		parseT.Fatalf("expected invalid route error, got %v", parseErr)
	}
}

// TestRunPrerenderSkipBuildExportsRoutes verifies static export output in skip-build mode.
func TestRunPrerenderSkipBuildExportsRoutes(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutDir := filepath.Join(parseRoot, "out")
	if parseWriteErr := os.MkdirAll(parseOutDir, 0755); parseWriteErr != nil {
		parseT.Fatalf("mkdir out: %v", parseWriteErr)
	}
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/prerenderfixture\n\ngo 1.25\n"), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr2)
	}
	if parseWriteErr3 := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); parseWriteErr3 != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr3)
	}
	if parseWriteErr4 := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("<!doctype html><html><body>hello</body></html>"), 0644); parseWriteErr4 != nil {
		parseT.Fatalf("write index.html: %v", parseWriteErr4)
	}
	if parseWriteErr5 := os.WriteFile(filepath.Join(parseOutDir, "main.wasm"), []byte("wasm"), 0644); parseWriteErr5 != nil {
		parseT.Fatalf("write main.wasm fixture: %v", parseWriteErr5)
	}
	if parseWriteErr6 := os.MkdirAll(filepath.Join(parseRoot, "assets"), 0755); parseWriteErr6 != nil {
		parseT.Fatalf("mkdir assets: %v", parseWriteErr6)
	}
	if parseWriteErr7 := os.WriteFile(filepath.Join(parseRoot, "assets", "site.css"), []byte("body{}"), 0644); parseWriteErr7 != nil {
		parseT.Fatalf("write asset: %v", parseWriteErr7)
	}

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		parseT.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if parseRunErr := (launcher{}).runPrerender([]string{
		"-root", parseRoot,
		"-app", "main.go",
		"-html", "index.html",
		"-out", parseOutDir,
		"-route", "/",
		"-route", "/docs",
		"-asset-dir", "assets",
		"-skip-build",
		"-json",
	}); parseRunErr != nil {
		parseT.Fatalf("run prerender: %v", parseRunErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		parseT.Fatalf("read prerender output: %v", parseOutputErr)
	}
	var parseSummary prerenderSummary
	if parseDecodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); parseDecodeErr != nil {
		parseT.Fatalf("decode prerender summary: %v\n%s", parseDecodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.BuildExecuted {
		parseT.Fatalf("expected prerender skip-build summary, got %#v", parseSummary)
	}
	if !fileExists(filepath.Join(parseOutDir, "index.html")) || !fileExists(filepath.Join(parseOutDir, "docs", "index.html")) {
		parseT.Fatalf("expected prerendered route html files in %s", parseOutDir)
	}
	if !fileExists(filepath.Join(parseOutDir, "assets", "site.css")) {
		parseT.Fatalf("expected copied asset in export output")
	}
	if !fileExists(parseSummary.ManifestPath) {
		parseT.Fatalf("expected manifest path to exist: %s", parseSummary.ManifestPath)
	}
	if !fileExists(filepath.Join(parseOutDir, "wasm_exec.js")) {
		parseT.Fatalf("expected wasm_exec.js to be copied to output")
	}
}
