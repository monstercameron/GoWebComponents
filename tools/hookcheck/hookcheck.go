// Command/library hookcheck is a static "rules of hooks" analyzer for
// GoWebComponents. It flags hooks (any Use* function, including ui.UseEvent behind
// an On* handler) that are not called at a stable render position:
//
//   - inside a loop (the framework's #1 gotcha, G1) — a per-row hook must live in
//     its own component, not in a range/for over a variable-length list; and
//   - inside a conditional branch (an if/else body or a switch/select case body) —
//     hooks must run unconditionally, in the same order every render.
//
// Each finding names the offending hook and its enclosing component and proposes
// the specific fix. It is a compile-time check (go/ast only, no runtime cost and no
// x/tools dependency), so it can never destabilize the render path.
package hookcheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FindingKind names which rules-of-hooks violation a [Finding] reports.
type FindingKind string

const (
	// KindLoop is a hook called inside a for/range loop.
	KindLoop FindingKind = "loop"
	// KindConditional is a hook called inside a conditional branch (an if/else
	// body or a switch/select case body) rather than at the component's top level.
	KindConditional FindingKind = "conditional"
)

// Finding is one rules-of-hooks violation. Pos is the source location, Hook the
// offending hook's name, Kind the violation category, and Func the enclosing
// component/function symbol (empty when the hook is in an anonymous function), so a
// message can name exactly where the problem is.
type Finding struct {
	Pos  token.Position
	Hook string
	Kind FindingKind
	Func string
}

// in returns " in <Func>" when the enclosing symbol is known, else "".
func (parseF Finding) in() string {
	if parseF.Func == "" {
		return ""
	}
	return " in " + parseF.Func
}

// String renders the full diagnostic: location, the named hook and enclosing
// symbol, the cause, and the specific corrective action for this Kind.
func (parseF Finding) String() string {
	return fmt.Sprintf("%s:%d:%d: hook %s called %s%s — %s",
		parseF.Pos.Filename, parseF.Pos.Line, parseF.Pos.Column, parseF.Hook,
		parseF.kindPhrase(), parseF.in(), parseF.Remediation())
}

// kindPhrase is the human phrase for the violation category.
func (parseF Finding) kindPhrase() string {
	if parseF.Kind == KindConditional {
		return "conditionally"
	}
	return "inside a loop"
}

// Remediation returns the specific corrective action for this finding, naming the
// offending hook so the fix is unambiguous (not a category-level hint).
func (parseF Finding) Remediation() string {
	switch parseF.Kind {
	case KindConditional:
		return fmt.Sprintf("hooks must run unconditionally in the same order every render; call %s at the top level of the component (before any if/switch) and read its value inside the branch", parseF.Hook)
	default:
		return fmt.Sprintf("hooks must run at stable render positions; move %s out of the loop by extracting the repeated row into its own component (see the rules-of-hooks gotcha)", parseF.Hook)
	}
}

// hookNameRe matches the Use<Upper> naming convention every GWC/React-style hook
// follows (UseState, UseEffect, UseRef, UseEvent, UseDOMRef, UseGlobalKey, …).
var hookNameRe = regexp.MustCompile(`^Use\p{Lu}`)

// IgnoreDirective suppresses a finding when present as a line comment on the
// hook's line or the line immediately above it — for deliberate, controlled cases
// (e.g. a fixed-count benchmark loop).
const IgnoreDirective = "hookcheck:ignore"

// CheckSource analyzes one source file given as bytes (filename is used only for
// positions). Returns a parse error if the source does not compile syntactically.
func CheckSource(parseFilename string, parseSrc []byte) ([]Finding, error) {
	parseFset := token.NewFileSet()
	parseFile, parseErr := parser.ParseFile(parseFset, parseFilename, parseSrc, parser.ParseComments|parser.SkipObjectResolution)
	if parseErr != nil {
		return nil, parseErr
	}
	return checkFile(parseFset, parseFile, parseSrc), nil
}

// ignoreLines splits hookcheck:ignore directives into trailing (on the same line
// as code → suppresses that line) and leading (a standalone comment line →
// suppresses the next line). The distinction prevents a trailing ignore on one
// hook from accidentally suppressing the hook on the following line.
func ignoreLines(parseFset *token.FileSet, parseFile *ast.File, parseSrc []byte) (parseTrailing, parseLeading map[int]bool) {
	parseTrailing = map[int]bool{}
	parseLeading = map[int]bool{}
	for _, parseGroup := range parseFile.Comments {
		for _, parseComment := range parseGroup.List {
			if !strings.Contains(parseComment.Text, IgnoreDirective) {
				continue
			}
			parsePos := parseFset.Position(parseComment.Pos())
			parseLineStart := parsePos.Offset - (parsePos.Column - 1)
			parsePrefixIsBlank := parseLineStart >= 0 && parseLineStart <= len(parseSrc) &&
				strings.TrimSpace(string(parseSrc[parseLineStart:parsePos.Offset])) == ""
			if parsePrefixIsBlank {
				parseLeading[parsePos.Line] = true
			} else {
				parseTrailing[parsePos.Line] = true
			}
		}
	}
	return parseTrailing, parseLeading
}

// CheckDir walks root, parses every non-vendored .go file, and returns all
// findings sorted by position. testdata, vendor, and dot-directories are skipped.
func CheckDir(parseRoot string) ([]Finding, error) {
	var parseFindings []Finding
	parseErr := filepath.WalkDir(parseRoot, func(parsePath string, parseEntry os.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			parseName := parseEntry.Name()
			if parseName != "." && (strings.HasPrefix(parseName, ".") || parseName == "vendor" || parseName == "testdata" || parseName == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(parsePath, ".go") {
			return nil
		}
		parseSrc, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		parseFileFindings, parseCheckErr := CheckSource(parsePath, parseSrc)
		if parseCheckErr != nil {
			// A syntactically valid file always parses (go/parser ignores build
			// constraints — they're comments), so this branch means a genuine
			// syntax error. Surface it to stderr instead of silently dropping the
			// file, which would be a blind spot (a hook violation in a file that
			// also has a typo would go unreported).
			fmt.Fprintf(os.Stderr, "hookcheck: skipping unparseable %s: %v\n", parsePath, parseCheckErr)
			return nil
		}
		parseFindings = append(parseFindings, parseFileFindings...)
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}
	sort.Slice(parseFindings, func(parseI, parseJ int) bool {
		if parseFindings[parseI].Pos.Filename != parseFindings[parseJ].Pos.Filename {
			return parseFindings[parseI].Pos.Filename < parseFindings[parseJ].Pos.Filename
		}
		return parseFindings[parseI].Pos.Offset < parseFindings[parseJ].Pos.Offset
	})
	return parseFindings, nil
}

func checkFile(parseFset *token.FileSet, parseFile *ast.File, parseSrc []byte) []Finding {
	parseTrailing, parseLeading := ignoreLines(parseFset, parseFile, parseSrc)
	var parseFindings []Finding
	var parsePath []ast.Node
	ast.Inspect(parseFile, func(parseNode ast.Node) bool {
		if parseNode == nil {
			parsePath = parsePath[:len(parsePath)-1]
			return false
		}
		if parseCall, parseOk := parseNode.(*ast.CallExpr); parseOk {
			if parseName, parseIsHook := hookName(parseCall.Fun); parseIsHook {
				// A hook in a loop is the more severe G1 violation and takes
				// precedence; otherwise a hook inside a conditional branch is a
				// rules-of-hooks (call-order) violation.
				parseKind := FindingKind("")
				if loopDepthFromPath(parsePath) > 0 {
					parseKind = KindLoop
				} else if conditionalFromPath(parsePath) {
					parseKind = KindConditional
				}
				if parseKind != "" {
					parsePos := parseFset.Position(parseCall.Pos())
					// Suppress on a trailing directive on the hook's line, or a
					// standalone directive on the line immediately above.
					if !parseTrailing[parsePos.Line] && !parseLeading[parsePos.Line-1] {
						parseFindings = append(parseFindings, Finding{
							Pos:  parsePos,
							Hook: parseName,
							Kind: parseKind,
							Func: enclosingFuncName(parsePath),
						})
					}
				}
			}
		}
		parsePath = append(parsePath, parseNode)
		return true
	})
	return parseFindings
}

// loopDepthFromPath counts the for/range loops enclosing the current node, but
// stops at the nearest function-literal boundary: a hook inside a handler closure
// is in a new scope, so a loop OUTSIDE that closure does not make it "in a loop".
func loopDepthFromPath(parsePath []ast.Node) int {
	parseDepth := 0
	for parseI := len(parsePath) - 1; parseI >= 0; parseI-- {
		switch parsePath[parseI].(type) {
		case *ast.FuncLit:
			return parseDepth
		case *ast.ForStmt, *ast.RangeStmt:
			parseDepth++
		}
	}
	return parseDepth
}

// conditionalFromPath reports whether the current node sits inside a conditional
// branch body — an if/else block or a switch/select case body — within the same
// function scope. A hook in the *condition* of an if (e.g. `if UseFoo() {}`) or the
// tag of a switch runs every render and is NOT flagged; only branch bodies are.
// The scan stops at the nearest function-literal boundary, like loopDepthFromPath.
func conditionalFromPath(parsePath []ast.Node) bool {
	for parseI := len(parsePath) - 1; parseI >= 0; parseI-- {
		switch parseNode := parsePath[parseI].(type) {
		case *ast.FuncLit:
			return false
		case *ast.CaseClause, *ast.CommClause:
			// A hook inside a switch case or a select comm clause body only runs on
			// the matching branch — conditional.
			return true
		case *ast.IfStmt:
			// Only the Body or Else branch is conditional; the Init/Cond run every
			// render. parsePath[parseI+1] is the child the hook descends through.
			if parseI+1 < len(parsePath) {
				parseChild := parsePath[parseI+1]
				if parseChild == ast.Node(parseNode.Body) || (parseNode.Else != nil && parseChild == parseNode.Else) {
					return true
				}
			}
		}
	}
	return false
}

// enclosingFuncName returns the name of the nearest enclosing top-level function
// (the component), or "" when the hook is inside an anonymous function with no
// named declaration on the path.
func enclosingFuncName(parsePath []ast.Node) string {
	for parseI := len(parsePath) - 1; parseI >= 0; parseI-- {
		if parseDecl, parseOk := parsePath[parseI].(*ast.FuncDecl); parseOk && parseDecl.Name != nil {
			return parseDecl.Name.Name
		}
	}
	return ""
}

// hookName returns the called function's name and whether it matches the hook
// convention, for both bare (UseState) and qualified (ui.UseState) calls.
func hookName(parseFun ast.Expr) (string, bool) {
	switch parseTyped := parseFun.(type) {
	case *ast.Ident:
		if hookNameRe.MatchString(parseTyped.Name) {
			return parseTyped.Name, true
		}
	case *ast.SelectorExpr:
		if hookNameRe.MatchString(parseTyped.Sel.Name) {
			return parseTyped.Sel.Name, true
		}
	}
	return "", false
}
