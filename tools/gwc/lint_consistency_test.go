package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeConsistencyFixture writes a Go source fixture and returns its path.
func writeConsistencyFixture(parseT *testing.T, parseSource string) (string, string) {
	parseT.Helper()
	parseDir := parseT.TempDir()
	parsePath := filepath.Join(parseDir, "fixture.go")
	if parseErr := os.WriteFile(parsePath, []byte(parseSource), 0o644); parseErr != nil {
		parseT.Fatalf("write fixture: %v", parseErr)
	}
	return parseDir, parsePath
}

// TestConsistencyFlagsCompatAliasWithoutDeprecation proves B2-1: an exported function whose doc
// self-describes as a compatibility wrapper but lacks the deprecation protocol is flagged.
func TestConsistencyFlagsCompatAliasWithoutDeprecation(parseT *testing.T) {
	parseDir, parsePath := writeConsistencyFixture(parseT,
		"package x\n\n// Foo is a compatibility wrapper around Bar.\nfunc Foo() int { return Bar() }\n\nfunc Bar() int { return 1 }\n")
	parseIssues, parseErr := collectLintConsistencyRuleFileIssues(parseDir, parsePath)
	if parseErr != nil {
		parseT.Fatalf("collect: %v", parseErr)
	}
	if len(parseIssues) != 1 || parseIssues[0].Symbol != "Foo" || parseIssues[0].Severity != "error" {
		parseT.Fatalf("expected one error on Foo, got %+v", parseIssues)
	}
}

// TestConsistencyAcceptsDeprecatedCompatAlias proves a compat alias that DOES declare the
// deprecation protocol (Deprecated: doc + deprecation.Warn) is NOT flagged.
func TestConsistencyAcceptsDeprecatedCompatAlias(parseT *testing.T) {
	parseDir, parsePath := writeConsistencyFixture(parseT,
		"package x\n\nimport \"github.com/monstercameron/GoWebComponents/v5/deprecation\"\n\n"+
			"// Foo is a compatibility wrapper around Bar.\n//\n// Deprecated: use Bar.\n"+
			"func Foo() int { deprecation.Warn(\"x.Foo\", \"x.Bar\"); return Bar() }\n\nfunc Bar() int { return 1 }\n")
	parseIssues, parseErr := collectLintConsistencyRuleFileIssues(parseDir, parsePath)
	if parseErr != nil {
		parseT.Fatalf("collect: %v", parseErr)
	}
	if len(parseIssues) != 0 {
		parseT.Fatalf("a properly-deprecated compat alias must not be flagged, got %+v", parseIssues)
	}
}

// TestConsistencyFlagsAdjacentBoolParams proves B2-3: adjacent bool params (boolean trap) warn.
func TestConsistencyFlagsAdjacentBoolParams(parseT *testing.T) {
	parseDir, parsePath := writeConsistencyFixture(parseT,
		"package x\n\nfunc Toggle(on bool, sticky bool) {}\n")
	parseIssues, parseErr := collectLintConsistencyRuleFileIssues(parseDir, parsePath)
	if parseErr != nil {
		parseT.Fatalf("collect: %v", parseErr)
	}
	if len(parseIssues) != 1 || parseIssues[0].Severity != "warning" || parseIssues[0].Symbol != "Toggle" {
		parseT.Fatalf("expected one boolean-trap warning on Toggle, got %+v", parseIssues)
	}
}

// TestConsistencyDoesNotFlagFactories proves the zero-false-positive guarantee: a normal thin
// public factory that returns another exported same-package function's result is NOT flagged
// (this was the false-positive class that sank the broader pure-delegate idea).
func TestConsistencyDoesNotFlagFactories(parseT *testing.T) {
	parseDir, parsePath := writeConsistencyFixture(parseT,
		"package x\n\n// Px builds a pixel length.\nfunc Px(n int) Len { return NewLen(n) }\n\ntype Len struct{}\nfunc NewLen(n int) Len { return Len{} }\n")
	parseIssues, parseErr := collectLintConsistencyRuleFileIssues(parseDir, parsePath)
	if parseErr != nil {
		parseT.Fatalf("collect: %v", parseErr)
	}
	if len(parseIssues) != 0 {
		parseT.Fatalf("a normal factory must not be flagged, got %+v", parseIssues)
	}
}

// TestConsistencyRealTreeClean is the merge gate: the actual framework tree must have zero
// gwc-consistency issues (all compat aliases declare the deprecation protocol). Runs under
// `go test ./...`, so a future undeclared compat alias or boolean-trap fails CI before merge.
func TestConsistencyRealTreeClean(parseT *testing.T) {
	parseRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Skipf("repo root not resolved: %v", parseErr)
	}
	parseIssues, parseErr := collectLintConsistencyRuleIssues(parseRoot, []string{"./..."})
	if parseErr != nil {
		parseT.Fatalf("collect real tree: %v", parseErr)
	}
	if len(parseIssues) != 0 {
		parseT.Fatalf("framework tree has %d gwc-consistency issue(s); fix or deprecate them:\n%+v", len(parseIssues), parseIssues)
	}
}
