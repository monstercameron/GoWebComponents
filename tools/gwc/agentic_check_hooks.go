package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
)

const (
	// checkHookContextCode mirrors the runtime panic tag so a static finding and the
	// runtime diagnostic read as the same problem.
	checkHookContextCode = "GWC-HOOK-OUTSIDE-COMPONENT"
	checkHookContextDocs = "ACTIONABLE_ERRORS.md#gwc-runtime-hook-outside-component"
	// checkHookContextPanic is the exact runtime message this static rule prevents.
	checkHookContextPanic = "GoUseAtom called outside component context"
)

// collectCheckHookContextDiagnostics flags GWC hooks (Use*) called outside a component
// render context. This is the inter-procedural shape the lexical hook linter cannot
// see: a hook at the top level of an ordinary helper, a boot/catch-up function, a
// closure (effect/callback/goroutine), or a package-level initializer. Each of these
// panics at runtime with "GoUseAtom called outside component context"; catching them
// statically turns a browser-only crash into a check failure.
func collectCheckHookContextDiagnostics(parseRootPath string) []agenticDiagnostic {
	parseFiles, parseErr := collectLintHookRuleFiles(parseRootPath, []string{"./..."})
	if parseErr != nil {
		return []agenticDiagnostic{{
			Code:     checkHookContextCode,
			Severity: "error",
			Message:  fmt.Sprintf("scan hook context files: %v", parseErr),
		}}
	}
	parseDiagnostics := []agenticDiagnostic{}
	for _, parsePath := range parseFiles {
		parseDiagnostics = append(parseDiagnostics, collectCheckHookContextFileDiagnostics(parseRootPath, parsePath)...)
	}
	return parseDiagnostics
}

// collectCheckHookContextFileDiagnostics analyzes one Go file for hook-context misuse.
func collectCheckHookContextFileDiagnostics(parseRootPath string, parsePath string) []agenticDiagnostic {
	parseSource, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil
	}
	parseFileSet := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFileSet, parsePath, parseSource, 0)
	if parseErr != nil {
		// Syntax errors are already surfaced by the convention pass (GWC-CHECK-PARSE);
		// don't double-report here.
		return nil
	}
	parseImports := buildLintHookRuleImports(parseFile.Imports)
	if len(parseImports) == 0 {
		return nil
	}

	parseRel := relativeSlashPath(parseRootPath, parsePath)
	parseDiagnostics := []agenticDiagnostic{}
	parseStack := []ast.Node{}
	ast.Inspect(parseFile, func(parseNode ast.Node) bool {
		if parseNode == nil {
			if len(parseStack) > 0 {
				parseStack = parseStack[:len(parseStack)-1]
			}
			return false
		}
		if parseCall, isParseCall := parseNode.(*ast.CallExpr); isParseCall {
			if parseHookName, parseHookPos, isParseHook := parseLintHookRuleCallName(parseCall, parseImports); isParseHook {
				if parseDiagnostic, isParseFlagged := evaluateCheckHookContext(parseFileSet, parseRel, parseHookName, parseHookPos, parseStack); isParseFlagged {
					parseDiagnostics = append(parseDiagnostics, parseDiagnostic)
				}
			}
		}
		parseStack = append(parseStack, parseNode)
		return true
	})
	return parseDiagnostics
}

// evaluateCheckHookContext classifies one hook call by its innermost enclosing
// function and returns a diagnostic when that context cannot be a component render.
func evaluateCheckHookContext(parseFileSet *token.FileSet, parseRel string, parseHookName string, parseHookPos token.Pos, parseStack []ast.Node) (agenticDiagnostic, bool) {
	parsePosition := parseFileSet.Position(parseHookPos)
	parseDiagnostic := agenticDiagnostic{
		Code:     checkHookContextCode,
		Severity: "error",
		File:     parseRel,
		Line:     parsePosition.Line,
		Column:   parsePosition.Column,
		Attributes: map[string]string{
			"hook": parseHookName,
			"docs": checkHookContextDocs,
		},
	}

	switch parseEnclosing := innermostEnclosingFunc(parseStack).(type) {
	case *ast.FuncDecl:
		if funcDeclIsRenderContext(parseEnclosing) {
			return agenticDiagnostic{}, false
		}
		parseName := "this function"
		if parseEnclosing.Name != nil {
			parseName = parseEnclosing.Name.Name
			parseDiagnostic.Attributes["function"] = parseName
		}
		parseDiagnostic.Message = fmt.Sprintf(
			"GWC hook %s is called in %s, which is neither a component (a function returning ui.Node) nor a Use* hook function; if it runs outside a render (boot, catch-up, an effect) it panics with %q.",
			parseHookName, parseName, checkHookContextPanic)
		parseDiagnostic.Suggestion = fmt.Sprintf(
			"If %s only runs while a component renders, rename it to Use… so it is treated as a hook; otherwise replace the hook with a non-hook accessor (a captured atom or a CurrentX() snapshot).",
			parseName)
		return parseDiagnostic, true
	case *ast.FuncLit:
		parseDiagnostic.Message = fmt.Sprintf(
			"GWC hook %s is called inside a closure (an effect, event callback, goroutine, or per-row loop render), which runs outside the component's synchronous render and panics with %q.",
			parseHookName, checkHookContextPanic)
		parseDiagnostic.Suggestion = "Call the hook at the component's top level, capture the atom it returns, and use the captured reference (for example atom.Set(...)) inside the closure."
		return parseDiagnostic, true
	default:
		parseDiagnostic.Message = fmt.Sprintf(
			"GWC hook %s is called at package level (a var initializer or init function), where no component is rendering; this panics with %q.",
			parseHookName, checkHookContextPanic)
		parseDiagnostic.Suggestion = "Move the hook into a component render, or read the value through a non-hook accessor (a captured atom or a CurrentX() snapshot)."
		return parseDiagnostic, true
	}
}

// innermostEnclosingFunc returns the nearest *ast.FuncDecl or *ast.FuncLit ancestor,
// or nil when the call sits at package level.
func innermostEnclosingFunc(parseStack []ast.Node) ast.Node {
	for parseIndex := len(parseStack) - 1; parseIndex >= 0; parseIndex-- {
		switch parseStack[parseIndex].(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return parseStack[parseIndex]
		}
	}
	return nil
}

// funcDeclIsRenderContext reports whether a function may legitimately call hooks:
// a custom hook (named use*/Use*), or a component that returns ui.Node.
func funcDeclIsRenderContext(parseFuncDecl *ast.FuncDecl) bool {
	if parseFuncDecl.Name != nil && isHookShapedFuncName(parseFuncDecl.Name.Name) {
		return true
	}
	return funcTypeReturnsNode(parseFuncDecl.Type)
}

// isHookShapedFuncName reports whether a function name follows the custom-hook
// convention — use<Upper> (unexported) or Use<Upper> (exported). A function that
// calls hooks is itself a hook and must be named this way so callers can see it
// must run during render; the analyzer trusts that contract.
func isHookShapedFuncName(parseName string) bool {
	if len(parseName) <= len("use") {
		return false
	}
	if parseName[:len("use")] != "use" && parseName[:len("Use")] != "Use" {
		return false
	}
	parseFirst := parseName[len("use")]
	return parseFirst >= 'A' && parseFirst <= 'Z'
}

// funcTypeReturnsNode reports whether a function signature returns a ui.Node result.
func funcTypeReturnsNode(parseFuncType *ast.FuncType) bool {
	if parseFuncType == nil || parseFuncType.Results == nil {
		return false
	}
	for _, parseResult := range parseFuncType.Results.List {
		if exprNamedNode(parseResult.Type) {
			return true
		}
	}
	return false
}

// exprNamedNode reports whether a type expression names the framework Node type
// (Node or ui.Node), the conventional component return type.
func exprNamedNode(parseExpr ast.Expr) bool {
	switch parseTyped := parseExpr.(type) {
	case *ast.Ident:
		return parseTyped.Name == "Node"
	case *ast.SelectorExpr:
		return parseTyped.Sel != nil && parseTyped.Sel.Name == "Node"
	default:
		return false
	}
}
