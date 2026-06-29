package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseInspectConfigNormalizesRoot verifies inspect config normalization and validation.
func TestParseInspectConfigNormalizesRoot(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfig, parseErr := parseInspectConfig(inspectConfig{rootPath: parseRoot, json: true})
	if parseErr != nil {
		parseT.Fatalf("parse inspect config: %v", parseErr)
	}
	if parseConfig.rootPath != parseRoot {
		parseT.Fatalf("expected root %q, got %q", parseRoot, parseConfig.rootPath)
	}
	if !parseConfig.json {
		parseT.Fatalf("expected json mode to be preserved")
	}
}

// TestParseInspectConfigRejectsFilePath verifies inspect root path validation.
func TestParseInspectConfigRejectsFilePath(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseFilePath := filepath.Join(parseRoot, "not-a-directory.txt")
	if parseWriteErr := os.WriteFile(parseFilePath, []byte("file"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write fixture file: %v", parseWriteErr)
	}
	_, parseErr := parseInspectConfig(inspectConfig{rootPath: parseFilePath})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "inspect root is not a directory") {
		parseT.Fatalf("expected inspect directory error, got %v", parseErr)
	}
}

// TestBuildInspectSummaryCollectsCoreReports verifies inspect report collection on a valid project root.
func TestBuildInspectSummaryCollectsCoreReports(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseModule := "module example.com/inspectfixture\n\ngo 1.25\n"
	if parseWriteErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
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
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr2)
	}
	if parseWriteErr3 := os.MkdirAll(filepath.Join(parseRoot, "web"), 0755); parseWriteErr3 != nil {
		parseT.Fatalf("mkdir web: %v", parseWriteErr3)
	}
	if parseWriteErr4 := os.WriteFile(filepath.Join(parseRoot, "web", "index.html"), []byte("<!doctype html>"), 0644); parseWriteErr4 != nil {
		parseT.Fatalf("write web/index.html: %v", parseWriteErr4)
	}
	parseSummary := buildInspectSummary(inspectConfig{rootPath: parseRoot})
	if !parseSummary.OK {
		parseT.Fatalf("expected inspect summary to succeed, got %#v", parseSummary)
	}
	if parseSummary.Routes.Registrations == 0 {
		parseT.Fatalf("expected route registration detection, got %#v", parseSummary.Routes)
	}
	if !parseSummary.Routes.HasLazySplit || !parseSummary.Routes.HasPrerenderHint || !parseSummary.Routes.HasMarketingRoute {
		parseT.Fatalf("expected route hints to be detected, got %#v", parseSummary.Routes)
	}
	if parseSummary.Dependencies.TotalImports == 0 {
		parseT.Fatalf("expected dependency import discovery, got %#v", parseSummary.Dependencies)
	}
	if parseSummary.FileTypes.TotalFiles < 2 {
		parseT.Fatalf("expected file type report to include files, got %#v", parseSummary.FileTypes)
	}
}

// TestRunInspectOutputsJSON verifies JSON output shape for inspect command execution.
func TestRunInspectOutputsJSON(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseModule := "module example.com/inspectfixture\n\ngo 1.25\n"
	if parseWriteErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte(parseModule), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
	}
	parseMain := `package main

func main() {}
`
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseRoot, "main.go"), []byte(parseMain), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr2)
	}

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		parseT.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if parseRunErr := (launcher{}).runInspect([]string{"-root", parseRoot, "-json"}); parseRunErr != nil {
		parseT.Fatalf("run inspect: %v", parseRunErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		parseT.Fatalf("read inspect output: %v", parseOutputErr)
	}
	var parseSummary inspectSummary
	if parseDecodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); parseDecodeErr != nil {
		parseT.Fatalf("decode inspect summary: %v\n%s", parseDecodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.Root != parseRoot {
		parseT.Fatalf("unexpected inspect json summary: %#v", parseSummary)
	}
}
