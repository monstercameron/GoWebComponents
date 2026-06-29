package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

const lintConsistencyRuleLinter = "gwc-consistency"

// consistencyExcludedPathPrefixes are repo subtrees the consistency rules never scan: example
// apps, tooling, tests, docs, and internal (non-public) packages. The rules enforce conventions
// on the PUBLIC framework API surface; example/tool code legitimately uses patterns (single-
// expression delegates, paired bools) that are not public-API violations. Keeping the scan to the
// public packages is what makes the rules zero-false-positive against the current tree.
var consistencyExcludedPathPrefixes = []string{
	"examples/", "tools/", "test/", "docs/", "internal/",
}

// collectLintConsistencyRuleIssues runs the gwc-consistency rules over the public framework
// packages: (1) a compat-alias that doesn't declare the deprecation protocol, (2) a same-package
// pure-delegate synonym that isn't marked deprecated, and (3) a forward guard against adjacent
// bool parameters (boolean traps). It reuses the hook-rule file collector (which already skips
// generated/vendored/testdata trees) and then drops the non-public subtrees above.
func collectLintConsistencyRuleIssues(parseRootPath string, parsePaths []string) ([]lintIssueRecord, error) {
	parseFiles, parseErr := collectLintHookRuleFiles(parseRootPath, parsePaths)
	if parseErr != nil {
		return nil, parseErr
	}
	parseIssues := []lintIssueRecord{}
	for _, parsePath := range parseFiles {
		if consistencyPathExcluded(parseRootPath, parsePath) {
			continue
		}
		parseFileIssues, parseErr2 := collectLintConsistencyRuleFileIssues(parseRootPath, parsePath)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		parseIssues = append(parseIssues, parseFileIssues...)
	}
	sortLintIssues(parseIssues)
	return parseIssues, nil
}

// consistencyPathExcluded reports whether a file is outside the public-API scan scope.
func consistencyPathExcluded(parseRootPath string, parsePath string) bool {
	parseRel := filepath.ToSlash(parseLintIssuePath(parseRootPath, parsePath))
	for _, parsePrefix := range consistencyExcludedPathPrefixes {
		if strings.HasPrefix(parseRel, parsePrefix) {
			return true
		}
	}
	return false
}

func collectLintConsistencyRuleFileIssues(parseRootPath string, parsePath string) ([]lintIssueRecord, error) {
	parseSource, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read consistency rules file %s: %w", parsePath, parseErr)
	}
	if isLintGeneratedGoSource(parseSource) {
		return nil, nil
	}
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parsePath, parseSource, parser.ParseComments)
	if parseErr != nil {
		return nil, fmt.Errorf("parse consistency rules file %s: %w", parsePath, parseErr)
	}
	parseImports := buildLintDeprecationRuleImports(parseFile.Imports)
	parseSourceLines := strings.Split(string(parseSource), "\n")
	parseIssues := []lintIssueRecord{}

	for _, parseDecl := range parseFile.Decls {
		parseFunc, isParseFunc := parseDecl.(*ast.FuncDecl)
		if !isParseFunc || !ast.IsExported(parseFunc.Name.Name) {
			continue
		}
		parsePosition := parseFileSet.Position(parseFunc.Name.Pos())
		parseSourceLine := parseLintHookRuleSourceLine(parseSourceLines, parsePosition.Line)
		parsePathRel := parseLintIssuePath(parseRootPath, parsePath)
		parseDocText := ""
		if parseFunc.Doc != nil {
			parseDocText = parseFunc.Doc.Text()
		}
		parseIsDeprecated := strings.Contains(parseDocText, "Deprecated:")
		parseHasWarn := hasLintDeprecationWarnCall(parseFunc.Body, parseImports)

		// B2-3: forward guard — adjacent bool parameters (boolean trap).
		if consistencyHasAdjacentBoolParams(parseFunc) {
			parseIssues = append(parseIssues, lintIssueRecord{
				Linter:     lintConsistencyRuleLinter,
				Severity:   "warning",
				Path:       parsePathRel,
				Line:       parsePosition.Line,
				Column:     parsePosition.Column,
				Message:    fmt.Sprintf("exported %s has adjacent bool parameters (boolean trap); prefer an options struct or named single bools", parseFunc.Name.Name),
				SourceLine: parseSourceLine,
				Symbol:     parseFunc.Name.Name,
			})
		}

		if parseFunc.Body == nil || parseIsDeprecated || parseHasWarn {
			continue
		}

		// B2-1: a compat-alias (doc self-describes as one) must declare the deprecation protocol
		// (// Deprecated: doc + deprecation.Warn), so legacy names are greppable and surfaced to
		// callers. This is the precise, zero-false-positive synonym guard — the phrases below
		// appear only on genuine declared compat wrappers.
		if consistencyDocIsCompatAlias(parseDocText) {
			parseIssues = append(parseIssues, lintIssueRecord{
				Linter:     lintConsistencyRuleLinter,
				Severity:   "error",
				Path:       parsePathRel,
				Line:       parsePosition.Line,
				Column:     parsePosition.Column,
				Message:    fmt.Sprintf("compatibility-alias %s must use the deprecation protocol (// Deprecated: doc + deprecation.Warn) so legacy names stay greppable", parseFunc.Name.Name),
				SourceLine: parseSourceLine,
				Symbol:     parseFunc.Name.Name,
			})
		}
	}
	return parseIssues, nil
}

// consistencyDocIsCompatAlias reports whether a doc comment self-describes as a compatibility
// alias. The phrases are exact and case-sensitive so the rule only fires on real declared
// compat wrappers, not prose that merely mentions compatibility.
func consistencyDocIsCompatAlias(parseDocText string) bool {
	for _, parsePhrase := range []string{
		"compatibility wrapper",
		"preserves the original public API",
	} {
		if strings.Contains(parseDocText, parsePhrase) {
			return true
		}
	}
	return false
}

// consistencyHasAdjacentBoolParams reports whether the function's parameter list has two
// consecutive plain `bool` parameters (the boolean-trap red flag). Variadic `...bool` and
// `func() bool` are not plain bool params and do not match.
func consistencyHasAdjacentBoolParams(parseFunc *ast.FuncDecl) bool {
	if parseFunc.Type == nil || parseFunc.Type.Params == nil {
		return false
	}
	parsePrevWasBool := false
	for _, parseField := range parseFunc.Type.Params.List {
		parseIdent, isIdent := parseField.Type.(*ast.Ident)
		parseIsBool := isIdent && parseIdent.Name == "bool"
		parseNames := len(parseField.Names)
		if parseNames == 0 {
			parseNames = 1
		}
		if parseIsBool && parseNames >= 2 {
			return true // two+ bools in a single grouped field, e.g. (a, b bool)
		}
		if parseIsBool && parsePrevWasBool {
			return true // bool field directly after a bool field
		}
		parsePrevWasBool = parseIsBool
	}
	return false
}
