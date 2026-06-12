package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestLintHelperParsingAndFormatting verifies lint JSON parsing, formatting, and report persistence helpers.
func TestLintHelperParsingAndFormatting(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseIssues, parseErr := parseLintIssues(`{"issues":[{"Linter":"govet","Message":"bad","Pos":{"Filename":"`+filepath.ToSlash(filepath.Join(parseRoot, "pkg", "file.go"))+`","Line":"7","Column":2},"SourceLines":[" line "]}]}`, parseRoot)
	if parseErr != nil {
		parseT.Fatalf("parseLintIssues(lowercase): %v", parseErr)
	}
	if len(parseIssues) != 1 || parseIssues[0].Path != "pkg/file.go" || parseIssues[0].Line != 7 || parseIssues[0].Column != 2 || parseIssues[0].Severity != "unspecified" {
		parseT.Fatalf("unexpected parsed issues %#v", parseIssues)
	}
	if parseIssues[0].SourceLine != "line" {
		parseT.Fatalf("expected trimmed source line, got %#v", parseIssues[0])
	}

	if parseEmpty, parseErr2 := parseLintIssues(`{"Issues":null}`, parseRoot); parseErr2 != nil || parseEmpty != nil {
		parseT.Fatalf("expected nil issue list, got %#v err=%v", parseEmpty, parseErr2)
	}
	if _, parseErr3 := parseLintIssues(`{"Issues":{}}`, parseRoot); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "missing issues array") {
		parseT.Fatalf("expected issues array error, got %v", parseErr3)
	}

	parseIssue := parseLintIssueRecord(map[string]any{
		"FromLinter": json.Number("0"),
		"Text":       "",
		"Message":    "fallback message",
		"Severity":   "",
		"Pos": map[string]any{
			"Filename": filepath.ToSlash(filepath.Join(parseRoot, "pkg", "branch.go")),
			"Line":     json.Number("8"),
			"Column":   float64(4),
		},
	}, parseRoot)
	if parseIssue.Linter != "0" || parseIssue.Message != "fallback message" || parseIssue.Path != "pkg/branch.go" {
		parseT.Fatalf("unexpected parsed issue record %#v", parseIssue)
	}

	if parseLintString(bytes.NewBufferString("reader")) != "reader" {
		parseT.Fatalf("expected fmt.Stringer conversion")
	}
	if parseLintInt(json.Number("19")) != 19 || parseLintInt("bad") != 0 {
		parseT.Fatalf("unexpected parseLintInt branches")
	}
	if parseStrings := parseLintStrings([]any{" a ", "", 42}); len(parseStrings) != 2 || parseStrings[0] != "a" || parseStrings[1] != "42" {
		parseT.Fatalf("unexpected parseLintStrings %#v", parseStrings)
	}

	parseSummary := lintSummary{
		Tool:           "golangci-lint",
		ToolVersion:    "2.4.0",
		ToolPath:       "C:/bin/golangci-lint",
		ProjectRoot:    parseRoot,
		Paths:          []string{"./...", "./cmd"},
		ConfigPath:     filepath.Join(parseRoot, ".golangci.yml"),
		DurationMs:     42,
		IssueCount:     1,
		Issues:         parseIssues,
		LinterCounts:   map[string]int{"govet": 1},
		SeverityCounts: map[string]int{"warning": 1},
		ReportPath:     filepath.Join(parseRoot, "reports", "lint.txt"),
		Resolution:     map[string]string{"root": "flag", "config": "auto", "report": "flag"},
	}
	parseRendered := formatLintReport(parseSummary)
	for _, parseNeedle := range []string{"GWC lint", "govet", "pkg/file.go:7:2", "report:", "root source: flag", "source:   line"} {
		if !strings.Contains(parseRendered, parseNeedle) {
			parseT.Fatalf("expected lint report to contain %q\n%s", parseNeedle, parseRendered)
		}
	}

	if parseGot := formatLintLocation(lintIssueRecord{}); parseGot != "<unknown>" {
		parseT.Fatalf("expected unknown location, got %q", parseGot)
	}
	if parseGot2 := formatLintCounts(nil); parseGot2 != "<none>" {
		parseT.Fatalf("expected empty counts label, got %q", parseGot2)
	}
	if parseGot3 := formatLintCounts(map[string]int{"b": 2, "a": 1}); parseGot3 != "a=1, b=2" {
		parseT.Fatalf("unexpected sorted count label %q", parseGot3)
	}

	parseReportPath := filepath.Join(parseRoot, "reports", "lint.txt")
	if parseErr4 := storeLintReport(parseReportPath, []byte("lint report")); parseErr4 != nil {
		parseT.Fatalf("storeLintReport(write): %v", parseErr4)
	}
	if parseErr5 := storeLintReport("", []byte("ignored")); parseErr5 != nil {
		parseT.Fatalf("storeLintReport(empty): %v", parseErr5)
	}
}

// TestLintExecutionHelpersVerifyCommandFailures exercises process execution and error formatting branches.
func TestLintExecutionHelpersVerifyCommandFailures(parseT *testing.T) {
	parseCommand := "sh"
	parseArgs := []string{"-c", "printf 'stdout\\n'; printf 'stderr\\n' >&2; exit 3"}
	if runtime.GOOS == "windows" {
		parseCommand = "cmd"
		parseArgs = []string{"/c", "echo stdout & echo stderr 1>&2 & exit /b 3"}
	}

	parseResult, parseErr := executeLintProcess(parseCommand, parseArgs, parseT.TempDir(), nil)
	if parseErr != nil {
		parseT.Fatalf("executeLintProcess(exit): %v", parseErr)
	}
	if parseResult.exitCode != 3 || !strings.Contains(parseResult.stdout, "stdout") || !strings.Contains(parseResult.stderr, "stderr") {
		parseT.Fatalf("unexpected lint process result %#v", parseResult)
	}

	if _, parseErr2 := executeLintProcess("gwc-command-that-does-not-exist", nil, parseT.TempDir(), nil); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "run gwc-command-that-does-not-exist") {
		parseT.Fatalf("expected missing command error, got %v", parseErr2)
	}

	parseExecErr := buildLintExecutionError(lintProcessResult{exitCode: 9, stderr: "stderr", stdout: "stdout"}, "golangci-lint", []string{"run", "--path", "pkg with space"})
	if parseExecErr == nil || !strings.Contains(parseExecErr.Error(), "\"pkg with space\"") || !strings.Contains(parseExecErr.Error(), "stderr\nstdout") {
		parseT.Fatalf("unexpected execution error %v", parseExecErr)
	}
	parseNoDiagErr := buildLintExecutionError(lintProcessResult{exitCode: 2}, "golangci-lint", []string{"run"})
	if parseNoDiagErr == nil || !strings.Contains(parseNoDiagErr.Error(), "no diagnostic output") {
		parseT.Fatalf("expected no diagnostic output branch, got %v", parseNoDiagErr)
	}
}
