package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCollectLintHookRuleIssuesFlagsBoundedViolations verifies conditional, loop, and nested function hook diagnostics.
func TestCollectLintHookRuleIssuesFlagsBoundedViolations(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parsePath := filepath.Join(parseRoot, "app", "component.go")
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0755); parseErr != nil {
		parseT.Fatalf("create app dir: %v", parseErr)
	}
	parseSource := `package app

import (
	gwcui "github.com/monstercameron/GoWebComponents/ui"
	"github.com/monstercameron/GoWebComponents/state"
	"github.com/monstercameron/GoWebComponents/fetch"
	"github.com/monstercameron/GoWebComponents/flags"
	"github.com/monstercameron/GoWebComponents/router"
)

func Component() gwcui.Node {
	parseTop := gwcui.UseState(0)
	_ = parseTop
	if true {
		_ = gwcui.UseState[int](1)
	}
	for parseI := 0; parseI < 1; parseI++ {
		_ = state.UseAtom("counter", 0)
	}
	parseLater := func() {
		_ = fetch.UseFetch("/api")
	}
	_ = parseLater
	go func() {
		_ = gwcui.UseRef(0)
	}()
	switch "new-nav" {
	case "new-nav":
		_ = flags.UseFlag("new-nav", false)
	}
	_ = router.UseNavigate()
	return nil
}
`
	if parseErr := os.WriteFile(parsePath, []byte(parseSource), 0644); parseErr != nil {
		parseT.Fatalf("write app source: %v", parseErr)
	}

	parseIssues, parseErr := collectLintHookRuleIssues(parseRoot, []string{"./..."})
	if parseErr != nil {
		parseT.Fatalf("collect hook rule issues: %v", parseErr)
	}
	if len(parseIssues) != 5 {
		parseT.Fatalf("expected five hook rule issues, got %#v", parseIssues)
	}
	parseMessages := make([]string, 0, len(parseIssues))
	for _, parseIssue := range parseIssues {
		parseMessages = append(parseMessages, parseIssue.Message)
	}
	parseJoined := strings.Join(parseMessages, "\n")
	for _, parseNeedle := range []string{
		"gwcui.UseState",
		"conditional control flow",
		"state.UseAtom",
		"a loop",
		"fetch.UseFetch",
		"a nested function",
		"gwcui.UseRef",
		"a goroutine launch",
		"flags.UseFlag",
	} {
		if !strings.Contains(parseJoined, parseNeedle) {
			parseT.Fatalf("expected hook issues to contain %q, got:\n%s", parseNeedle, parseJoined)
		}
	}
	if strings.Contains(parseJoined, "router.UseNavigate") {
		parseT.Fatalf("expected top-level router hook to pass, got:\n%s", parseJoined)
	}
	for _, parseIssue := range parseIssues {
		if parseIssue.Linter != lintHookRuleLinter || parseIssue.Severity != "error" || parseIssue.Path != "app/component.go" {
			parseT.Fatalf("unexpected issue metadata %#v", parseIssue)
		}
		if parseIssue.SourceLine == "" {
			parseT.Fatalf("expected source line on issue %#v", parseIssue)
		}
	}
}

// TestCollectLintHookRuleIssuesHandlesDotImportsAndTargets verifies dot imports and local lint path scoping.
func TestCollectLintHookRuleIssuesHandlesDotImportsAndTargets(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseScopedPath := filepath.Join(parseRoot, "scoped", "component.go")
	parseOtherPath := filepath.Join(parseRoot, "other", "component.go")
	for _, parsePath := range []string{parseScopedPath, parseOtherPath} {
		if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0755); parseErr != nil {
			parseT.Fatalf("create source dir: %v", parseErr)
		}
	}
	parseScopedSource := `package scoped

import . "github.com/monstercameron/GoWebComponents/ui"

func Component() {
	if true {
		_ = UseRef(0)
	}
}
`
	parseOtherSource := `package other

import "github.com/monstercameron/GoWebComponents/ui"

func Component() {
	if true {
		_ = ui.UseId()
	}
}
`
	if parseErr := os.WriteFile(parseScopedPath, []byte(parseScopedSource), 0644); parseErr != nil {
		parseT.Fatalf("write scoped source: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseOtherPath, []byte(parseOtherSource), 0644); parseErr != nil {
		parseT.Fatalf("write other source: %v", parseErr)
	}

	parseIssues, parseErr := collectLintHookRuleIssues(parseRoot, []string{"./scoped"})
	if parseErr != nil {
		parseT.Fatalf("collect scoped hook rule issues: %v", parseErr)
	}
	if len(parseIssues) != 1 {
		parseT.Fatalf("expected one scoped issue, got %#v", parseIssues)
	}
	if parseIssues[0].Path != "scoped/component.go" || !strings.Contains(parseIssues[0].Message, "UseRef") {
		parseT.Fatalf("unexpected scoped issue %#v", parseIssues[0])
	}

	parseSkipped, parseErr := collectLintHookRuleIssues(parseRoot, []string{"example.com/external/package", "./missing/..."})
	if parseErr != nil {
		parseT.Fatalf("collect non-local hook rule issues: %v", parseErr)
	}
	if len(parseSkipped) != 0 {
		parseT.Fatalf("expected non-local and missing targets to be skipped, got %#v", parseSkipped)
	}
}

// TestBuildLintSummaryIncludesHookRuleIssues verifies built-in hook findings join the lint report.
func TestBuildLintSummaryIncludesHookRuleIssues(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseSourcePath := filepath.Join(parseRoot, "component.go")
	parseSource := `package app

import "github.com/monstercameron/GoWebComponents/ui"

func Component() {
	if true {
		_ = ui.UseId()
	}
}
`
	if parseErr := os.WriteFile(parseSourcePath, []byte(parseSource), 0644); parseErr != nil {
		parseT.Fatalf("write hook source: %v", parseErr)
	}

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
		if len(parseArgs) > 0 && parseArgs[0] == "version" {
			return lintProcessResult{stdout: "2.4.0"}, nil
		}
		if len(parseArgs) == 2 && parseArgs[0] == "config" && parseArgs[1] == "path" {
			return lintProcessResult{}, nil
		}
		return lintProcessResult{stdout: `{"Issues":[]}`}, nil
	}

	parseSummary, isParseIssuesFound, parseErr := buildLintSummary(lintConfig{
		rootPath: parseRoot,
		toolPath: "golangci-lint",
		paths:    []string{"./..."},
	})
	if parseErr != nil {
		parseT.Fatalf("build lint summary: %v", parseErr)
	}
	if !isParseIssuesFound || parseSummary.OK || parseSummary.IssueCount != 1 {
		parseT.Fatalf("expected one hook-rule lint failure, found=%v summary=%#v", isParseIssuesFound, parseSummary)
	}
	if !parseSummary.HookRules || parseSummary.LinterCounts[lintHookRuleLinter] != 1 {
		parseT.Fatalf("expected hook rule counts, got %#v", parseSummary)
	}

	parseSkippedSummary, isParseSkippedIssuesFound, parseErr := buildLintSummary(lintConfig{
		rootPath:       parseRoot,
		toolPath:       "golangci-lint",
		paths:          []string{"./..."},
		skipHookRules:  true,
		disableLinters: []string{"gwc-hooks"},
	})
	if parseErr != nil {
		parseT.Fatalf("build skipped lint summary: %v", parseErr)
	}
	if isParseSkippedIssuesFound || !parseSkippedSummary.OK || parseSkippedSummary.IssueCount != 0 || parseSkippedSummary.HookRules {
		parseT.Fatalf("expected skipped hook rules to pass, found=%v summary=%#v", isParseSkippedIssuesFound, parseSkippedSummary)
	}
}
