package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseInspectConfigNormalizesRoot verifies inspect config normalization and validation.
func TestParseInspectConfigNormalizesRoot(t *testing.T) {
	parseRoot := t.TempDir()
	parseConfig, parseErr := parseInspectConfig(inspectConfig{rootPath: parseRoot, json: true})
	if parseErr != nil {
		t.Fatalf("parse inspect config: %v", parseErr)
	}
	if parseConfig.rootPath != parseRoot {
		t.Fatalf("expected root %q, got %q", parseRoot, parseConfig.rootPath)
	}
	if !parseConfig.json {
		t.Fatalf("expected json mode to be preserved")
	}
}

// TestParseInspectConfigRejectsFilePath verifies inspect root path validation.
func TestParseInspectConfigRejectsFilePath(t *testing.T) {
	parseRoot := t.TempDir()
	parseFilePath := filepath.Join(parseRoot, "not-a-directory.txt")
	if writeErr := os.WriteFile(parseFilePath, []byte("file"), 0644); writeErr != nil {
		t.Fatalf("write fixture file: %v", writeErr)
	}
	_, parseErr := parseInspectConfig(inspectConfig{rootPath: parseFilePath})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "inspect root is not a directory") {
		t.Fatalf("expected inspect directory error, got %v", parseErr)
	}
}

// TestBuildInspectSummaryCollectsCoreReports verifies inspect report collection on a valid project root.
func TestBuildInspectSummaryCollectsCoreReports(t *testing.T) {
	parseRoot := t.TempDir()
	parseModule := "module example.com/inspectfixture\n\ngo 1.25\n"
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	parseMain := `package main

import "fmt"

// router.Register("/pricing", nil)
// ui.Lazy(func() {})
// prerender signal

func main() {
	fmt.Println("hello")
}
`
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); writeErr != nil {
		t.Fatalf("write main.go: %v", writeErr)
	}
	if writeErr := os.MkdirAll(filepath.Join(parseRoot, "web"), 0755); writeErr != nil {
		t.Fatalf("mkdir web: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "web", "index.html"), []byte("<!doctype html>"), 0644); writeErr != nil {
		t.Fatalf("write web/index.html: %v", writeErr)
	}
	parseSummary := buildInspectSummary(inspectConfig{rootPath: parseRoot})
	if !parseSummary.OK {
		t.Fatalf("expected inspect summary to succeed, got %#v", parseSummary)
	}
	if parseSummary.Routes.Registrations == 0 {
		t.Fatalf("expected route registration detection, got %#v", parseSummary.Routes)
	}
	if !parseSummary.Routes.HasLazySplit || !parseSummary.Routes.HasPrerenderHint || !parseSummary.Routes.HasMarketingRoute {
		t.Fatalf("expected route hints to be detected, got %#v", parseSummary.Routes)
	}
	if parseSummary.Dependencies.TotalImports == 0 {
		t.Fatalf("expected dependency import discovery, got %#v", parseSummary.Dependencies)
	}
	if parseSummary.FileTypes.TotalFiles < 2 {
		t.Fatalf("expected file type report to include files, got %#v", parseSummary.FileTypes)
	}
}

// TestRunInspectOutputsJSON verifies JSON output shape for inspect command execution.
func TestRunInspectOutputsJSON(t *testing.T) {
	parseRoot := t.TempDir()
	parseModule := "module example.com/inspectfixture\n\ngo 1.25\n"
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); writeErr != nil {
		t.Fatalf("write go.mod: %v", writeErr)
	}
	parseMain := `package main

func main() {}
`
	if writeErr := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); writeErr != nil {
		t.Fatalf("write main.go: %v", writeErr)
	}

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		t.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if runErr := (launcher{}).runInspect([]string{"-root", parseRoot, "-json"}); runErr != nil {
		t.Fatalf("run inspect: %v", runErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		t.Fatalf("read inspect output: %v", parseOutputErr)
	}
	var parseSummary inspectSummary
	if decodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); decodeErr != nil {
		t.Fatalf("decode inspect summary: %v\n%s", decodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.Root != parseRoot {
		t.Fatalf("unexpected inspect json summary: %#v", parseSummary)
	}
}
