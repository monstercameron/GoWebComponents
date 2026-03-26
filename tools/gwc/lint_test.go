package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseLintConfigAppliesDefaults verifies lint config normalization and fallback paths.
func TestParseLintConfigAppliesDefaults(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOriginalGetLintWorkingDir := getLintWorkingDir
	parseT.Cleanup(func() {
		getLintWorkingDir = parseOriginalGetLintWorkingDir
	})
	getLintWorkingDir = func() (string, error) {
		return parseRoot, nil
	}

	parseReportPath := filepath.Join(parseRoot, "reports", "lint.txt")
	parseConfig, parseErr := parseLintConfig(lintConfig{
		reportPath: parseReportPath,
	})
	if parseErr != nil {
		parseT.Fatalf("parse lint config: %v", parseErr)
	}
	if parseConfig.rootPath != parseRoot {
		parseT.Fatalf("expected root %q, got %q", parseRoot, parseConfig.rootPath)
	}
	if len(parseConfig.paths) != 1 || parseConfig.paths[0] != "./..." {
		parseT.Fatalf("expected default lint path, got %#v", parseConfig.paths)
	}
	if parseConfig.toolPath != "golangci-lint" {
		parseT.Fatalf("expected default tool path, got %q", parseConfig.toolPath)
	}
	if parseConfig.reportPath != parseReportPath {
		parseT.Fatalf("expected report path %q, got %q", parseReportPath, parseConfig.reportPath)
	}
	if parseConfig.resolution["root"] != "convention fallback" {
		parseT.Fatalf("expected root resolution trace, got %#v", parseConfig.resolution)
	}
}

// TestRunLintJSONSummarizesIssues verifies JSON output and failing exit behavior when issues exist.
func TestRunLintJSONSummarizesIssues(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseOriginalResolveLintExecutable := resolveLintExecutable
	parseOriginalRunLintProcess := runLintProcess
	parseT.Cleanup(func() {
		resolveLintExecutable = parseOriginalResolveLintExecutable
		runLintProcess = parseOriginalRunLintProcess
	})

	resolveLintExecutable = func(parseToolPath string) (string, error) {
		return filepath.Join(parseRoot, "bin", "golangci-lint"), nil
	}
	runLintProcess = func(parseCommand string, parseArgs []string, parseWorkingDir string, parseEnv []string) (lintProcessResult, error) {
		if strings.Contains(strings.Join(parseArgs, " "), "version") {
			return lintProcessResult{stdout: "2.4.0"}, nil
		}
		if len(parseArgs) == 2 && parseArgs[0] == "config" && parseArgs[1] == "path" {
			return lintProcessResult{stdout: filepath.Join(parseRoot, ".golangci.yml")}, nil
		}
		parseIssuePayload := `{"Issues":[{"FromLinter":"errcheck","Text":"error return value not checked","Severity":"error","Pos":{"Filename":"` + filepath.ToSlash(filepath.Join(parseRoot, "pkg", "app.go")) + `","Line":12,"Column":5},"SourceLines":["if err != nil {"]},{"FromLinter":"gocritic","Text":"unnamedResult","Pos":{"Filename":"pkg/other.go","Line":4,"Column":1}}]}`
		return lintProcessResult{stdout: parseIssuePayload, exitCode: 1}, nil
	}

	parseRunErr := (launcher{}).run([]string{"lint", "-root", parseRoot, "-json"})
	if parseRunErr == nil || !strings.Contains(parseRunErr.Error(), "lint found 2 issue(s)") {
		parseT.Fatalf("expected lint issue failure, got %v", parseRunErr)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}

	var parseSummary lintSummary
	if parseErr2 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr2 != nil {
		parseT.Fatalf("decode lint summary: %v\noutput=%s", parseErr2, parseOutput)
	}
	if parseSummary.OK {
		parseT.Fatalf("expected failing lint summary, got %#v", parseSummary)
	}
	if parseSummary.IssueCount != 2 {
		parseT.Fatalf("expected two issues, got %#v", parseSummary)
	}
	if parseSummary.ToolVersion != "2.4.0" {
		parseT.Fatalf("expected version 2.4.0, got %#v", parseSummary)
	}
	if parseSummary.LinterCounts["errcheck"] != 1 || parseSummary.LinterCounts["gocritic"] != 1 {
		parseT.Fatalf("expected linter counts, got %#v", parseSummary.LinterCounts)
	}
	if parseSummary.Issues[0].Path != "pkg/app.go" {
		parseT.Fatalf("expected root-relative lint path, got %#v", parseSummary.Issues[0])
	}
}

// TestRunLintWritesTextReport verifies the rendered text report and file output.
func TestRunLintWritesTextReport(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseReportPath := filepath.Join(parseRoot, "reports", "lint.txt")
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseOriginalResolveLintExecutable := resolveLintExecutable
	parseOriginalRunLintProcess := runLintProcess
	parseT.Cleanup(func() {
		resolveLintExecutable = parseOriginalResolveLintExecutable
		runLintProcess = parseOriginalRunLintProcess
	})

	resolveLintExecutable = func(parseToolPath string) (string, error) {
		return filepath.Join(parseRoot, "bin", "golangci-lint"), nil
	}
	runLintProcess = func(parseCommand string, parseArgs []string, parseWorkingDir string, parseEnv []string) (lintProcessResult, error) {
		if strings.Contains(strings.Join(parseArgs, " "), "version") {
			return lintProcessResult{stdout: "2.4.0"}, nil
		}
		if len(parseArgs) == 2 && parseArgs[0] == "config" && parseArgs[1] == "path" {
			return lintProcessResult{}, nil
		}
		return lintProcessResult{stdout: `{"Issues":[]}`}, nil
	}

	if parseErr2 := (launcher{}).runLint([]string{"-root", parseRoot, "-out", parseReportPath}); parseErr2 != nil {
		parseT.Fatalf("run lint: %v", parseErr2)
	}

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	if !strings.Contains(parseOutput, "GWC lint") || !strings.Contains(parseOutput, "issues:       0") || !strings.Contains(parseOutput, parseReportPath) {
		parseT.Fatalf("unexpected text lint output:\n%s", parseOutput)
	}
	parseReportBytes, parseErr := os.ReadFile(parseReportPath)
	if parseErr != nil {
		parseT.Fatalf("read report file: %v", parseErr)
	}
	if string(parseReportBytes) != parseOutput {
		parseT.Fatalf("expected report file to match stdout\nstdout=%q\nfile=%q", parseOutput, string(parseReportBytes))
	}
}

// TestRunLintHandlesMissingExecutable verifies a clear error when golangci-lint is unavailable.
func TestRunLintHandlesMissingExecutable(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOriginalResolveLintExecutable := resolveLintExecutable
	parseT.Cleanup(func() {
		resolveLintExecutable = parseOriginalResolveLintExecutable
	})
	resolveLintExecutable = func(parseToolPath string) (string, error) {
		return "", errors.New("not found")
	}

	parseErr := (launcher{}).runLint([]string{"-root", parseRoot})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "resolve golangci-lint executable") {
		parseT.Fatalf("expected missing executable error, got %v", parseErr)
	}
}

// TestBuildLintCommandArgsSupportsV1 verifies the v1 CLI compatibility argument set.
func TestBuildLintCommandArgsSupportsV1(parseT *testing.T) {
	parseConfig := lintConfig{
		configPath:     filepath.Join(parseT.TempDir(), ".golangci.yml"),
		paths:          []string{"./..."},
		timeout:        "2m",
		enableLinters:  []string{"errcheck"},
		disableLinters: []string{"gosec"},
		fastOnly:       true,
	}

	parseArgs := buildLintCommandArgs(parseConfig, 1)
	parseJoined := strings.Join(parseArgs, " ")
	for _, parseExpected := range []string{"--out-format=json:stdout,line-number:stderr", "--fast", "--timeout 2m", "--enable errcheck", "--disable gosec"} {
		if !strings.Contains(parseJoined, parseExpected) {
			parseT.Fatalf("expected v1 lint args to contain %q, got %q", parseExpected, parseJoined)
		}
	}
}
