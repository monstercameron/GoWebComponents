package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// formatLintHookGolden renders hook-rule issues as stable golden lines.
func formatLintHookGolden(parseIssues []lintIssueRecord) string {
	var parseBuilder strings.Builder
	for _, parseIssue := range parseIssues {
		fmt.Fprintf(&parseBuilder, "%d:%d %s %s %s | %s\n", parseIssue.Line, parseIssue.Column, parseIssue.Severity, parseIssue.Symbol, parseIssue.SourceLine, parseIssue.Message)
	}
	return parseBuilder.String()
}

// TestLintHookInlineHandlerGolden pins the inline-handler heuristic against a
// fixture: handlers in range bodies and Map/MapKeyed callbacks are flagged;
// top-level, variable, nested-closure and MapKeyedComponent handlers are not.
// Set GWC_UPDATE_GOLDEN=1 to rewrite the golden file.
func TestLintHookInlineHandlerGolden(parseT *testing.T) {
	parseDir := filepath.Join("testdata", "lint_hooks_inline_handler")
	parseFixture := filepath.Join(parseDir, "component.go")
	parseIssues, parseErr := collectLintHookRuleFileIssues(parseDir, parseFixture)
	if parseErr != nil {
		parseT.Fatalf("collect fixture issues: %v", parseErr)
	}
	parseGot := strings.ReplaceAll(formatLintHookGolden(parseIssues), "\r\n", "\n")
	parseGoldenPath := filepath.Join(parseDir, "component.golden")
	if os.Getenv("GWC_UPDATE_GOLDEN") == "1" {
		if parseWriteErr := os.WriteFile(parseGoldenPath, []byte(parseGot), 0644); parseWriteErr != nil {
			parseT.Fatalf("write golden: %v", parseWriteErr)
		}
	}
	parseWantBytes, parseReadErr := os.ReadFile(parseGoldenPath)
	if parseReadErr != nil {
		parseT.Fatalf("read golden: %v", parseReadErr)
	}
	parseWant := strings.ReplaceAll(string(parseWantBytes), "\r\n", "\n")
	if parseGot != parseWant {
		parseT.Fatalf("inline handler issues differ from golden.\ngot:\n%s\nwant:\n%s", parseGot, parseWant)
	}
	if len(parseIssues) != 3 {
		parseT.Fatalf("expected exactly the three flagged handlers, got %d", len(parseIssues))
	}
	for _, parseIssue := range parseIssues {
		if parseIssue.Linter != lintHookRuleLinter || !strings.Contains(parseIssue.Message, "MapKeyedComponent") || !strings.Contains(parseIssue.Message, "Heuristic") {
			parseT.Fatalf("issue must name the fix and the heuristic, got %#v", parseIssue)
		}
	}
}

// TestLintHookInlineHandlerRequiresHTMLImport keeps the rule off files that do
// not import the html packages, so unrelated On* functions never match.
func TestLintHookInlineHandlerRequiresHTMLImport(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parsePath := filepath.Join(parseRoot, "other.go")
	parseSource := "package app\n\nimport other \"example.com/other\"\n\nfunc F(parseItems []int) {\n\tfor range parseItems {\n\t\tother.OnClick(func() {})\n\t}\n}\n"
	if parseErr := os.WriteFile(parsePath, []byte(parseSource), 0644); parseErr != nil {
		parseT.Fatalf("write source: %v", parseErr)
	}
	parseIssues, parseErr := collectLintHookRuleIssues(parseRoot, []string{"./..."})
	if parseErr != nil || len(parseIssues) != 0 {
		parseT.Fatalf("expected no issues, got %#v (err %v)", parseIssues, parseErr)
	}
}
