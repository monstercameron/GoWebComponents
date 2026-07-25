package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

const lintDeprecationRuleLinter = "gwc-deprecations"

// collectLintDeprecationRuleIssues scans exported deprecated APIs for the
// required one-time deprecation diagnostic.
func collectLintDeprecationRuleIssues(parseRootPath string, parsePaths []string) ([]lintIssueRecord, error) {
	parseFiles, parseErr := collectLintHookRuleFiles(parseRootPath, parsePaths)
	if parseErr != nil {
		return nil, parseErr
	}
	parseIssues := []lintIssueRecord{}
	for _, parsePath := range parseFiles {
		parseFileIssues, parseErr2 := collectLintDeprecationRuleFileIssues(parseRootPath, parsePath)
		if parseErr2 != nil {
			return nil, parseErr2
		}
		parseIssues = append(parseIssues, parseFileIssues...)
	}
	sortLintIssues(parseIssues)
	return parseIssues, nil
}

func collectLintDeprecationRuleFileIssues(parseRootPath string, parsePath string) ([]lintIssueRecord, error) {
	parseSource, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read deprecation rules file %s: %w", parsePath, parseErr)
	}
	if isLintGeneratedGoSource(parseSource) {
		return nil, nil
	}
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parsePath, parseSource, parser.ParseComments)
	if parseErr != nil {
		return nil, fmt.Errorf("parse deprecation rules file %s: %w", parsePath, parseErr)
	}

	parseImports := buildLintDeprecationRuleImports(parseFile.Imports)
	parseSourceLines := strings.Split(string(parseSource), "\n")
	parseIssues := []lintIssueRecord{}
	for _, parseDecl := range parseFile.Decls {
		parseFunc, isParseFunc := parseDecl.(*ast.FuncDecl)
		if !isParseFunc || parseFunc.Body == nil || parseFunc.Doc == nil || !ast.IsExported(parseFunc.Name.Name) {
			continue
		}
		if !isLintDeprecationDoc(parseFunc.Doc) || hasLintDeprecationWarnCall(parseFunc.Body, parseImports) {
			continue
		}
		parsePosition := parseFileSet.Position(parseFunc.Name.Pos())
		parseAPI := formatLintDeprecatedAPIName(parseFunc)
		parseIssues = append(parseIssues, lintIssueRecord{
			Linter:     lintDeprecationRuleLinter,
			Severity:   "error",
			Path:       parseLintIssuePath(parseRootPath, parsePath),
			Line:       parsePosition.Line,
			Column:     parsePosition.Column,
			Message:    fmt.Sprintf("deprecated public API %s must call deprecation.Warn with replacement guidance", parseAPI),
			SourceLine: parseLintHookRuleSourceLine(parseSourceLines, parsePosition.Line),
		})
	}
	return parseIssues, nil
}

func buildLintDeprecationRuleImports(parseImports []*ast.ImportSpec) map[string]string {
	parseMapped := map[string]string{}
	for _, parseSpec := range parseImports {
		parseImportPath, parseErr := strconv.Unquote(parseSpec.Path.Value)
		if parseErr != nil {
			parseImportPath = strings.Trim(parseSpec.Path.Value, `"`)
		}
		if strings.TrimSpace(parseImportPath) != "github.com/monstercameron/GoWebComponents/v5/deprecation" {
			continue
		}
		parseLocalName := "deprecation"
		if parseSpec.Name != nil {
			parseLocalName = parseSpec.Name.Name
		}
		if parseLocalName == "_" || strings.TrimSpace(parseLocalName) == "" {
			continue
		}
		parseMapped[parseLocalName] = parseImportPath
	}
	return parseMapped
}

func isLintDeprecationDoc(parseDoc *ast.CommentGroup) bool {
	if parseDoc == nil {
		return false
	}
	return strings.Contains(parseDoc.Text(), "Deprecated:")
}

func hasLintDeprecationWarnCall(parseBody *ast.BlockStmt, parseImports map[string]string) bool {
	if parseBody == nil || len(parseImports) == 0 {
		return false
	}
	parseFound := false
	ast.Inspect(parseBody, func(parseNode ast.Node) bool {
		if parseFound {
			return false
		}
		parseCall, isParseCall := parseNode.(*ast.CallExpr)
		if !isParseCall {
			return true
		}
		parseFound = isLintDeprecationWarnCall(parseCall, parseImports)
		return !parseFound
	})
	return parseFound
}

func isLintDeprecationWarnCall(parseCall *ast.CallExpr, parseImports map[string]string) bool {
	switch parseFun := parseCall.Fun.(type) {
	case *ast.SelectorExpr:
		if parseFun.Sel.Name != "Warn" {
			return false
		}
		parseIdent, isParseIdent := parseFun.X.(*ast.Ident)
		if !isParseIdent {
			return false
		}
		_, isParseImport := parseImports[parseIdent.Name]
		return isParseImport
	case *ast.Ident:
		if parseFun.Name != "Warn" {
			return false
		}
		_, isParseDotImport := parseImports["."]
		return isParseDotImport
	default:
		return false
	}
}

func formatLintDeprecatedAPIName(parseFunc *ast.FuncDecl) string {
	if parseFunc == nil {
		return "<unknown>"
	}
	if parseFunc.Recv == nil || len(parseFunc.Recv.List) == 0 {
		return parseFunc.Name.Name
	}
	return formatLintDeprecatedReceiverName(parseFunc.Recv.List[0].Type) + "." + parseFunc.Name.Name
}

func formatLintDeprecatedReceiverName(parseExpr ast.Expr) string {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name
	case *ast.StarExpr:
		return formatLintDeprecatedReceiverName(parseTyped.X)
	case *ast.IndexExpr:
		return formatLintDeprecatedReceiverName(parseTyped.X)
	case *ast.IndexListExpr:
		return formatLintDeprecatedReceiverName(parseTyped.X)
	default:
		return "<receiver>"
	}
}

func isLintGeneratedGoSource(parseSource []byte) bool {
	parseText := strings.ReplaceAll(string(parseSource), "\r\n", "\n")
	parseLines := strings.Split(parseText, "\n")
	if len(parseLines) > 20 {
		parseLines = parseLines[:20]
	}
	for _, parseLine := range parseLines {
		parseLine = strings.TrimSpace(parseLine)
		if strings.Contains(parseLine, "Code generated") && strings.Contains(parseLine, "DO NOT EDIT") {
			return true
		}
	}
	return false
}
