package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

const (
	lintHookHTMLImportPath      = "github.com/monstercameron/GoWebComponents/v6/html"
	lintHookShorthandImportPath = "github.com/monstercameron/GoWebComponents/v6/html/shorthand"
)

// lintHookInlineHandlerHelp is the help text of the inline-handler heuristic of
// the gwc-hooks rule. It is appended to every finding so the limits travel with
// the diagnostic.
const lintHookInlineHandlerHelp = "each inline handler allocates a positional hook slot (UseEvent) that shifts when items are added, removed or reordered; " +
	"render the row with MapKeyedComponent so every item owns its handlers, or hoist the handler out of the list. " +
	"Heuristic: flags only an html or html/shorthand On* option called with a func literal directly inside a for/range body or an html.Map/MapKeyed callback " +
	"(a handler passed as a variable, wrapped by an earlier UseEvent, or built inside a nested func literal is not flagged)"

// buildLintHookHTMLImports maps the local names the html and html/shorthand
// packages are imported under in one file; "." marks a dot-import.
func buildLintHookHTMLImports(parseImports []*ast.ImportSpec) map[string]bool {
	parseMapped := map[string]bool{}
	for _, parseSpec := range parseImports {
		parseImportPath, parseErr := strconv.Unquote(parseSpec.Path.Value)
		if parseErr != nil {
			continue
		}
		if parseImportPath != lintHookHTMLImportPath && parseImportPath != lintHookShorthandImportPath {
			continue
		}
		parseLocalName := parseLintHookRuleImportPackage(parseImportPath)
		if parseSpec.Name != nil {
			parseLocalName = parseSpec.Name.Name
		}
		if parseLocalName == "_" || strings.TrimSpace(parseLocalName) == "" {
			continue
		}
		parseMapped[parseLocalName] = true
	}
	return parseMapped
}

// getLintHookHTMLCallName returns the called function's name when it resolves to
// the html or html/shorthand package in this file.
func getLintHookHTMLCallName(parseFun ast.Expr, parseHTMLImports map[string]bool) (string, token.Pos, bool) {
	switch parseTyped := parseLintHookRuleCallFun(parseFun).(type) {
	case *ast.SelectorExpr:
		parseIdent, isParseIdent := parseTyped.X.(*ast.Ident)
		if isParseIdent && parseHTMLImports[parseIdent.Name] {
			return parseIdent.Name + "." + parseTyped.Sel.Name, parseTyped.Sel.Pos(), true
		}
	case *ast.Ident:
		if parseHTMLImports["."] {
			return parseTyped.Name, parseTyped.Pos(), true
		}
	}
	return "", token.NoPos, false
}

// getLintHookBareName strips the package qualifier from a displayed call name.
func getLintHookBareName(parseName string) string {
	if _, parseAfter, isParseCut := strings.Cut(parseName, "."); isParseCut {
		return parseAfter
	}
	return parseName
}

// isLintHookInlineHandlerCall reports whether a call is an html On* option given
// a func literal, the shape that allocates a hook slot through toHandler.
func isLintHookInlineHandlerCall(parseCall *ast.CallExpr, parseHTMLImports map[string]bool) (string, token.Pos, bool) {
	if len(parseHTMLImports) == 0 || len(parseCall.Args) == 0 {
		return "", token.NoPos, false
	}
	if _, isParseFuncLit := parseCall.Args[0].(*ast.FuncLit); !isParseFuncLit {
		return "", token.NoPos, false
	}
	parseName, parsePos, isParseHTML := getLintHookHTMLCallName(parseCall.Fun, parseHTMLImports)
	if !isParseHTML {
		return "", token.NoPos, false
	}
	parseBare := getLintHookBareName(parseName)
	if len(parseBare) <= len("On") || !strings.HasPrefix(parseBare, "On") || parseBare[2] < 'A' || parseBare[2] > 'Z' {
		return "", token.NoPos, false
	}
	return parseName, parsePos, true
}

// getLintHookListContext names the list context enclosing a handler call, or ""
// when the call is not rendered once per list item. The walk stops at the first
// func-literal boundary unless that literal is itself an html Map/MapKeyed
// callback, so a handler inside a nested closure is never flagged.
func getLintHookListContext(parseStack []ast.Node, parseHTMLImports map[string]bool) string {
	for parseIndex := len(parseStack) - 1; parseIndex >= 0; parseIndex-- {
		switch parseNode := parseStack[parseIndex].(type) {
		case *ast.ForStmt, *ast.RangeStmt:
			return "a loop"
		case *ast.FuncLit:
			if parseIndex == 0 {
				return ""
			}
			parseParent, isParseCall := parseStack[parseIndex-1].(*ast.CallExpr)
			if !isParseCall {
				return ""
			}
			parseName, _, isParseHTML := getLintHookHTMLCallName(parseParent.Fun, parseHTMLImports)
			if !isParseHTML {
				return ""
			}
			for _, parseArg := range parseParent.Args {
				if parseArg == ast.Expr(parseNode) {
					switch getLintHookBareName(parseName) {
					case "Map", "MapKeyed":
						return "a " + getLintHookBareName(parseName) + " callback"
					}
				}
			}
			return ""
		case *ast.FuncDecl:
			return ""
		}
	}
	return ""
}

// buildLintHookInlineHandlerIssue renders the issue for one inline handler found
// in a list context.
func buildLintHookInlineHandlerIssue(parseHandlerName string, parseContext string) (string, string) {
	return fmt.Sprintf("inline handler %s(func...) inside %s: %s", parseHandlerName, parseContext, lintHookInlineHandlerHelp), parseHandlerName
}
