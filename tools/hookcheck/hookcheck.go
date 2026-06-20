// Command/library hookcheck is a static "rules of hooks" analyzer for
// GoWebComponents. It flags the framework's #1 gotcha (G1): a hook (any Use*
// function, including ui.UseEvent behind an On* handler) called inside a loop.
// Hooks must run at stable render positions, so a per-row hook must live in its
// own component, not in a range/for over a variable-length list.
//
// It is a compile-time check (go/ast only, no runtime cost and no x/tools
// dependency), so it can never destabilize the render path.
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

// Finding is one hook-in-loop violation.
type Finding struct {
	Pos  token.Position
	Hook string
}

func (parseF Finding) String() string {
	return fmt.Sprintf(
		"%s:%d:%d: hook %s called inside a loop — hooks must run at stable render positions; extract the row into its own component (see the rules-of-hooks gotcha)",
		parseF.Pos.Filename, parseF.Pos.Line, parseF.Pos.Column, parseF.Hook,
	)
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
			// Skip files that don't parse (e.g. build-tagged stubs); they are not
			// the analyzer's concern.
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
			if parseName, parseIsHook := hookName(parseCall.Fun); parseIsHook && loopDepthFromPath(parsePath) > 0 {
				parsePos := parseFset.Position(parseCall.Pos())
				// Suppress on a trailing directive on the hook's line, or a
				// standalone directive on the line immediately above.
				if !parseTrailing[parsePos.Line] && !parseLeading[parsePos.Line-1] {
					parseFindings = append(parseFindings, Finding{Pos: parsePos, Hook: parseName})
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
