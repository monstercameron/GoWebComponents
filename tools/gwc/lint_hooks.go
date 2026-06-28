package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const lintHookRuleLinter = "gwc-hooks"

type lintHookRuleContext struct {
	conditional    bool
	loop           bool
	nestedFunction bool
	goroutine      bool
}

// collectLintHookRuleIssues scans Go files for obvious GWC hook call-order violations.
func collectLintHookRuleIssues(parseRootPath string, parsePaths []string) ([]lintIssueRecord, error) {
	parseFiles, parseErr := collectLintHookRuleFiles(parseRootPath, parsePaths)
	if parseErr != nil {
		return nil, parseErr
	}
	parseIssues := []lintIssueRecord{}
	for _, parsePath := range parseFiles {
		parseFileIssues, parseErr2 := collectLintHookRuleFileIssues(parseRootPath, parsePath)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		parseIssues = append(parseIssues, parseFileIssues...)
	}
	sortLintIssues(parseIssues)
	return parseIssues, nil
}

// collectLintHookRuleFiles resolves lint path patterns into local Go files.
func collectLintHookRuleFiles(parseRootPath string, parsePaths []string) ([]string, error) {
	parseRootPath = strings.TrimSpace(parseRootPath)
	if parseRootPath == "" {
		parseCwd, parseErr := os.Getwd()
		if parseErr != nil {
			return nil, fmt.Errorf("resolve hook rules root: %w", parseErr)
		}
		parseRootPath = parseCwd
	}
	parseFilesByPath := map[string]struct{}{}
	for _, parseTarget := range parseLintPaths(parsePaths) {
		parseTargetFiles, parseErr := collectLintHookRuleTargetFiles(parseRootPath, parseTarget)
		if parseErr != nil {
			return nil, parseErr
		}
		for _, parsePath := range parseTargetFiles {
			parseFilesByPath[parsePath] = struct{}{}
		}
	}
	parseFiles := make([]string, 0, len(parseFilesByPath))
	for parsePath := range parseFilesByPath {
		parseFiles = append(parseFiles, parsePath)
	}
	sort.Strings(parseFiles)
	return parseFiles, nil
}

// collectLintHookRuleTargetFiles resolves one lint path pattern into local Go files.
func collectLintHookRuleTargetFiles(parseRootPath string, parseTarget string) ([]string, error) {
	parseTarget = strings.TrimSpace(parseTarget)
	if parseTarget == "" {
		parseTarget = "./..."
	}
	parseTargetSlash := filepath.ToSlash(parseTarget)
	parseRecursive := false
	parseBaseTarget := parseTarget
	switch {
	case parseTargetSlash == "..." || parseTargetSlash == "./...":
		parseRecursive = true
		parseBaseTarget = "."
	case strings.HasSuffix(parseTargetSlash, "/..."):
		parseRecursive = true
		parseBaseTarget = strings.TrimSuffix(parseTargetSlash, "/...")
		if parseBaseTarget == "" {
			parseBaseTarget = "."
		}
	}

	parseBasePath := parseBaseTarget
	if !filepath.IsAbs(parseBasePath) {
		parseBasePath = filepath.Join(parseRootPath, filepath.FromSlash(parseBaseTarget))
	}
	parseInfo, parseErr := os.Stat(parseBasePath)
	if parseErr != nil {
		// Non-local package patterns are still valid for golangci-lint. The
		// built-in hook pass only scans paths it can resolve under the root.
		return nil, nil
	}
	if parseInfo.IsDir() {
		if parseRecursive {
			return collectLintHookRuleRecursiveFiles(parseBasePath)
		}
		return collectLintHookRuleDirFiles(parseBasePath)
	}
	if filepath.Ext(parseBasePath) != ".go" {
		return nil, nil
	}
	return []string{filepath.Clean(parseBasePath)}, nil
}

// collectLintHookRuleDirFiles collects Go files directly inside one directory.
func collectLintHookRuleDirFiles(parseDirPath string) ([]string, error) {
	parseEntries, parseErr := os.ReadDir(parseDirPath)
	if parseErr != nil {
		return nil, fmt.Errorf("scan hook rules dir %s: %w", parseDirPath, parseErr)
	}
	parseFiles := []string{}
	for _, parseEntry := range parseEntries {
		if parseEntry.IsDir() || filepath.Ext(parseEntry.Name()) != ".go" {
			continue
		}
		parseFiles = append(parseFiles, filepath.Join(parseDirPath, parseEntry.Name()))
	}
	sort.Strings(parseFiles)
	return parseFiles, nil
}

// collectLintHookRuleRecursiveFiles collects Go files below one directory.
func collectLintHookRuleRecursiveFiles(parseRootPath string) ([]string, error) {
	parseFiles := []string{}
	parseErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if parsePath != parseRootPath && shouldSkipLintHookRuleDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(parseEntry.Name()) == ".go" {
			parseFiles = append(parseFiles, parsePath)
		}
		return nil
	})
	if parseErr != nil {
		return nil, fmt.Errorf("scan hook rules files: %w", parseErr)
	}
	sort.Strings(parseFiles)
	return parseFiles, nil
}

// collectLintHookRuleFileIssues scans one parsed file for hook call-order violations.
func collectLintHookRuleFileIssues(parseRootPath string, parsePath string) ([]lintIssueRecord, error) {
	parseSource, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read hook rules file %s: %w", parsePath, parseErr)
	}
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parsePath, parseSource, 0)
	if parseErr != nil {
		return nil, fmt.Errorf("parse hook rules file %s: %w", parsePath, parseErr)
	}
	parseImports := buildLintHookRuleImports(parseFile.Imports)
	if len(parseImports) == 0 {
		return nil, nil
	}

	parseSourceLines := strings.Split(string(parseSource), "\n")
	parseIssues := []lintIssueRecord{}
	parseStack := []ast.Node{}
	ast.Inspect(parseFile, func(parseNode ast.Node) bool {
		if parseNode == nil {
			if len(parseStack) > 0 {
				parseStack = parseStack[:len(parseStack)-1]
			}
			return false
		}
		if parseCall, isParseCall := parseNode.(*ast.CallExpr); isParseCall {
			parseHookName, parseHookPos, isParseHook := parseLintHookRuleCallName(parseCall, parseImports)
			if isParseHook {
				parseContext := buildLintHookRuleContext(parseStack)
				if parseReason := formatLintHookRuleReason(parseContext); parseReason != "" {
					parsePosition := parseFileSet.Position(parseHookPos)
					parseIssues = append(parseIssues, lintIssueRecord{
						Linter:     lintHookRuleLinter,
						Severity:   "error",
						Path:       parseLintIssuePath(parseRootPath, parsePath),
						Line:       parsePosition.Line,
						Column:     parsePosition.Column,
						Message:    formatLintHookRuleMessage(parseHookName, parseReason),
						SourceLine: parseLintHookRuleSourceLine(parseSourceLines, parsePosition.Line),
						Symbol:     parseHookName,
					})
				}
			}
		}
		parseStack = append(parseStack, parseNode)
		return true
	})
	return parseIssues, nil
}

// buildLintHookRuleImports maps imported GWC hook package aliases to package names.
func buildLintHookRuleImports(parseImports []*ast.ImportSpec) map[string]string {
	parseMapped := map[string]string{}
	for _, parseSpec := range parseImports {
		parseImportPath, parseErr := strconv.Unquote(parseSpec.Path.Value)
		if parseErr != nil {
			parseImportPath = strings.Trim(parseSpec.Path.Value, `"`)
		}
		if !isLintHookRuleImportPath(parseImportPath) {
			continue
		}
		parsePackageName := parseLintHookRuleImportPackage(parseImportPath)
		parseLocalName := parsePackageName
		if parseSpec.Name != nil {
			parseLocalName = parseSpec.Name.Name
		}
		if parseLocalName == "_" || strings.TrimSpace(parseLocalName) == "" {
			continue
		}
		parseMapped[parseLocalName] = parsePackageName
	}
	return parseMapped
}

// buildLintHookRuleContext derives the bounded hook-rule context from AST ancestors.
func buildLintHookRuleContext(parseStack []ast.Node) lintHookRuleContext {
	parseContext := lintHookRuleContext{}
	parseFunctionDepth := 0
	for _, parseNode := range parseStack {
		switch parseNode.(type) {
		case *ast.IfStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
			parseContext.conditional = true
		case *ast.ForStmt, *ast.RangeStmt:
			parseContext.loop = true
		case *ast.GoStmt:
			parseContext.goroutine = true
		case *ast.FuncDecl:
			parseFunctionDepth++
		case *ast.FuncLit:
			if parseFunctionDepth > 0 {
				parseContext.nestedFunction = true
			}
			parseFunctionDepth++
		}
	}
	return parseContext
}

// parseLintHookRuleCallName returns the displayed hook name and position for a call.
func parseLintHookRuleCallName(parseCall *ast.CallExpr, parseImports map[string]string) (string, token.Pos, bool) {
	parseFun := parseLintHookRuleCallFun(parseCall.Fun)
	switch parseTyped := parseFun.(type) {
	case *ast.SelectorExpr:
		parseIdent, isParseIdent := parseTyped.X.(*ast.Ident)
		if !isParseIdent || !isLintHookRuleHookName(parseTyped.Sel.Name) {
			return "", token.NoPos, false
		}
		if _, isParseHookPackage := parseImports[parseIdent.Name]; !isParseHookPackage {
			return "", token.NoPos, false
		}
		return parseIdent.Name + "." + parseTyped.Sel.Name, parseTyped.Sel.Pos(), true
	case *ast.Ident:
		if _, isParseDotImport := parseImports["."]; !isParseDotImport || !isLintHookRuleHookName(parseTyped.Name) {
			return "", token.NoPos, false
		}
		return parseTyped.Name, parseTyped.Pos(), true
	default:
		return "", token.NoPos, false
	}
}

// parseLintHookRuleCallFun unwraps generic instantiations to the called function.
func parseLintHookRuleCallFun(parseExpr ast.Expr) ast.Expr {
	switch parseTyped := parseExpr.(type) {
	case *ast.IndexExpr:
		return parseLintHookRuleCallFun(parseTyped.X)
	case *ast.IndexListExpr:
		return parseLintHookRuleCallFun(parseTyped.X)
	default:
		return parseExpr
	}
}

// parseLintHookRuleImportPackage extracts the final package path segment.
func parseLintHookRuleImportPackage(parseImportPath string) string {
	parseImportPath = strings.TrimSpace(parseImportPath)
	if parseIndex := strings.LastIndex(parseImportPath, "/"); parseIndex >= 0 {
		return parseImportPath[parseIndex+1:]
	}
	return parseImportPath
}

// isLintHookRuleImportPath reports whether an import path is one of the GWC hook packages.
func isLintHookRuleImportPath(parseImportPath string) bool {
	parseImportPath = strings.TrimSpace(parseImportPath)
	if !strings.HasPrefix(parseImportPath, "github.com/monstercameron/GoWebComponents/") {
		return false
	}
	switch parseLintHookRuleImportPackage(parseImportPath) {
	case "ui", "state", "fetch", "flags", "router":
		return true
	default:
		return false
	}
}

// isLintHookRuleHookName reports whether an identifier has the exported GWC hook shape.
func isLintHookRuleHookName(parseName string) bool {
	if !strings.HasPrefix(parseName, "Use") || len(parseName) <= len("Use") {
		return false
	}
	parseFirst := parseName[len("Use")]
	return parseFirst >= 'A' && parseFirst <= 'Z'
}

// shouldSkipLintHookRuleDir reports whether a directory should be skipped by recursive scans.
func shouldSkipLintHookRuleDir(parseName string) bool {
	switch strings.TrimSpace(parseName) {
	case ".git", ".claude", "bin", "dist", "node_modules", "testdata", "third_party", "tmp", "vendor":
		return true
	default:
		return false
	}
}

// formatLintHookRuleReason renders the violated hook-rule context.
func formatLintHookRuleReason(parseContext lintHookRuleContext) string {
	parseReasons := []string{}
	if parseContext.conditional {
		parseReasons = append(parseReasons, "conditional control flow")
	}
	if parseContext.loop {
		parseReasons = append(parseReasons, "a loop")
	}
	if parseContext.goroutine {
		parseReasons = append(parseReasons, "a goroutine launch")
	}
	if parseContext.nestedFunction {
		parseReasons = append(parseReasons, "a nested function")
	}
	return strings.Join(parseReasons, ", ")
}

// formatLintHookRuleMessage renders the hook-rule issue message.
func formatLintHookRuleMessage(parseHookName string, parseReason string) string {
	return fmt.Sprintf("GWC hook %s must be called at component top level; found inside %s", parseHookName, parseReason)
}

// parseLintHookRuleSourceLine returns a trimmed one-based source line.
func parseLintHookRuleSourceLine(parseLines []string, parseLine int) string {
	if parseLine <= 0 || parseLine > len(parseLines) {
		return ""
	}
	return strings.TrimSpace(parseLines[parseLine-1])
}
