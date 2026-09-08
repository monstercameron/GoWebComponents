// Package apidump extracts a package's exported API as a stable, sorted list of signature
// lines and checks it against a committed golden file. It is the mechanism behind the
// API-baseline tests that pin a package's public surface: any change to an exported type,
// function, method, constant, or variable shows up as a failing test with a diff, so a
// breaking change can never land silently — the concrete meaning of "graduated to Stable,
// API-baseline pinned."
package apidump

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Extract parses the non-test Go files in dir (ignoring build tags, so the recorded surface
// is the union across platforms) and returns the exported API as sorted, normalized
// signature lines.
func Extract(parseDir string) ([]string, error) {
	parseEntries, parseErr := os.ReadDir(parseDir)
	if parseErr != nil {
		return nil, fmt.Errorf("read dir: %w", parseErr)
	}
	parseFset := token.NewFileSet()
	parseSeen := map[string]struct{}{}

	for _, parseEntry := range parseEntries {
		parseName := parseEntry.Name()
		if parseEntry.IsDir() || !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") {
			continue
		}
		parseFile, parseParseErr := parser.ParseFile(parseFset, filepath.Join(parseDir, parseName), nil, 0)
		if parseParseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", parseName, parseParseErr)
		}
		for _, parseDecl := range parseFile.Decls {
			for _, parseLine := range renderDecl(parseFset, parseDecl) {
				parseSeen[parseLine] = struct{}{}
			}
		}
	}

	parseLines := make([]string, 0, len(parseSeen))
	for parseLine := range parseSeen {
		parseLines = append(parseLines, parseLine)
	}
	sort.Strings(parseLines)
	return parseLines, nil
}

// ExtractSources parses an explicit source snapshot, avoiding filesystem directory handles.
func ExtractSources(parseSources map[string][]byte) ([]string, error) {
	parseFset := token.NewFileSet()
	parseSeen := map[string]struct{}{}
	for parseName, parseData := range parseSources {
		if strings.HasSuffix(parseName, "_test.go") || !strings.HasSuffix(parseName, ".go") {
			continue
		}
		parseFile, parseErr := parser.ParseFile(parseFset, parseName, parseData, 0)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", parseName, parseErr)
		}
		for _, parseDecl := range parseFile.Decls {
			for _, parseLine := range renderDecl(parseFset, parseDecl) {
				parseSeen[parseLine] = struct{}{}
			}
		}
	}
	parseLines := make([]string, 0, len(parseSeen))
	for parseLine := range parseSeen {
		parseLines = append(parseLines, parseLine)
	}
	sort.Strings(parseLines)
	return parseLines, nil
}

// CheckSources compares an embedded source snapshot with an embedded baseline.
func CheckSources(parseT testing.TB, parseSources map[string][]byte, parseGolden []byte) {
	parseT.Helper()
	parseLines, parseErr := ExtractSources(parseSources)
	if parseErr != nil {
		parseT.Fatalf("extract embedded API: %v", parseErr)
	}
	parseWant := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(parseGolden), "\r\n", "\n")), "\n")
	if strings.Join(parseLines, "\n") != strings.Join(parseWant, "\n") {
		parseT.Fatalf("API baseline mismatch\nwant:\n%s\ngot:\n%s", strings.Join(parseWant, "\n"), strings.Join(parseLines, "\n"))
	}
}

// renderDecl renders the exported API line(s) for one top-level declaration.
func renderDecl(parseFset *token.FileSet, parseDecl ast.Decl) []string {
	switch parseTyped := parseDecl.(type) {
	case *ast.FuncDecl:
		return renderFunc(parseFset, parseTyped)
	case *ast.GenDecl:
		return renderGen(parseFset, parseTyped)
	}
	return nil
}

// renderFunc renders an exported function or a method on an exported receiver.
func renderFunc(parseFset *token.FileSet, parseFunc *ast.FuncDecl) []string {
	if !parseFunc.Name.IsExported() {
		return nil
	}
	parseSignature := strings.TrimPrefix(normalize(render(parseFset, parseFunc.Type)), "func")

	if parseFunc.Recv != nil && len(parseFunc.Recv.List) == 1 {
		parseRecv := parseFunc.Recv.List[0].Type
		if !receiverIsExported(parseRecv) {
			return nil
		}
		return []string{fmt.Sprintf("func (%s) %s%s", normalize(render(parseFset, parseRecv)), parseFunc.Name.Name, parseSignature)}
	}
	return []string{fmt.Sprintf("func %s%s", parseFunc.Name.Name, parseSignature)}
}

// renderGen renders exported types, consts, and vars from a general declaration.
func renderGen(parseFset *token.FileSet, parseGen *ast.GenDecl) []string {
	var parseLines []string
	for _, parseSpec := range parseGen.Specs {
		switch parseTyped := parseSpec.(type) {
		case *ast.TypeSpec:
			if !parseTyped.Name.IsExported() {
				continue
			}
			parseKind := "type"
			if parseTyped.Assign.IsValid() {
				parseKind = "type ="
			}
			parseTypeParams := ""
			if parseTyped.TypeParams != nil {
				parseTypeParams = normalize(render(parseFset, parseTyped.TypeParams))
			}
			parseLines = append(parseLines, fmt.Sprintf("%s %s%s %s", parseKind, parseTyped.Name.Name, parseTypeParams, normalize(render(parseFset, parseTyped.Type))))
		case *ast.ValueSpec:
			parseKeyword := "var"
			if parseGen.Tok == token.CONST {
				parseKeyword = "const"
			}
			// Include the explicitly-declared type so a breaking type change to an
			// exported var/const (e.g. `var Default *Config` -> `*OtherConfig`) is
			// caught instead of collapsing to the same name-only line. Untyped
			// declarations (Type == nil, e.g. `var ErrX = errors.New(...)` or an
			// untyped const) stay name-only — the concrete type isn't in the AST
			// without go/types, so this is a conservative, no-inference improvement.
			parseTypeSuffix := ""
			if parseTyped.Type != nil {
				parseTypeSuffix = " " + normalize(render(parseFset, parseTyped.Type))
			}
			for _, parseName := range parseTyped.Names {
				if parseName.IsExported() {
					parseLines = append(parseLines, fmt.Sprintf("%s %s%s", parseKeyword, parseName.Name, parseTypeSuffix))
				}
			}
		}
	}
	return parseLines
}

// receiverIsExported reports whether a method receiver type (T or *T or T[…]) is exported.
func receiverIsExported(parseExpr ast.Expr) bool {
	switch parseTyped := parseExpr.(type) {
	case *ast.StarExpr:
		return receiverIsExported(parseTyped.X)
	case *ast.Ident:
		return parseTyped.IsExported()
	case *ast.IndexExpr:
		return receiverIsExported(parseTyped.X)
	case *ast.IndexListExpr:
		return receiverIsExported(parseTyped.X)
	}
	return false
}

// render prints an AST node back to source.
func render(parseFset *token.FileSet, parseNode ast.Node) string {
	var parseBuf bytes.Buffer
	if parseErr := format.Node(&parseBuf, parseFset, parseNode); parseErr != nil {
		return ""
	}
	return parseBuf.String()
}

// normalize collapses all whitespace runs to single spaces so multi-line struct/interface
// renderings compare as one stable line.
func normalize(parseSource string) string {
	return strings.Join(strings.Fields(parseSource), " ")
}

// Check compares dir's current exported API against the golden file, failing the test on
// any drift. Set UPDATE_API_BASELINE=1 to (re)write the golden after an intentional change.
func Check(parseT testing.TB, parseDir, parseGolden string) {
	parseT.Helper()
	parseLines, parseErr := Extract(parseDir)
	if parseErr != nil {
		parseT.Fatalf("extract API: %v", parseErr)
	}
	parseGot := strings.Join(parseLines, "\n") + "\n"

	if os.Getenv("UPDATE_API_BASELINE") == "1" {
		if parseErr := os.WriteFile(parseGolden, []byte(parseGot), 0644); parseErr != nil {
			parseT.Fatalf("write golden: %v", parseErr)
		}
		return
	}

	parseWant, parseErr := os.ReadFile(parseGolden)
	if parseErr != nil {
		parseT.Fatalf("read golden %s (generate it with UPDATE_API_BASELINE=1): %v", parseGolden, parseErr)
	}
	// Normalize line endings before comparing: the generated API uses "\n", but a golden file
	// checked out on Windows (autocrlf) carries "\r\n", which would otherwise fail the comparison
	// on every line despite identical content. This keeps the gate stable across platforms/checkouts.
	parseWantNormalized := strings.ReplaceAll(string(parseWant), "\r\n", "\n")
	if parseWantNormalized != parseGot {
		parseT.Fatalf("public API of %s changed.\nIf intentional, regenerate with UPDATE_API_BASELINE=1 and review the diff.\n\n--- current API ---\n%s", parseDir, parseGot)
	}
}
