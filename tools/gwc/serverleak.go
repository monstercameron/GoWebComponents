package main

import (
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// serverOnlyImports are packages that only make sense on the server; importing one from a
// browser-only (js && wasm) file is a leak — the code cannot run there, and it drags server
// concerns into the client bundle.
var serverOnlyImports = map[string]string{
	"os/exec":      "runs OS processes — unavailable in the browser",
	"database/sql": "direct database access belongs on the server",
	"net/smtp":     "sending mail belongs on the server",
	"os/user":      "OS user lookup is server-only",
	"net/http/cgi": "CGI is server-only",
	"plugin":       "Go plugins are server-only",
}

// collectServerLeakDiagnostics scans browser-only (js && wasm) Go files and reports any
// import of a server-only package — promoting the doctor heuristic to a real import-graph
// check wired into `gwc check`. It analyzes only files EXPLICITLY constrained to js && wasm,
// so server (`!js || !wasm`) and shared (no-constraint) files are never false-positived.
func collectServerLeakDiagnostics(parseRootPath string) []agenticDiagnostic {
	parseDiagnostics := []agenticDiagnostic{}
	parseFset := token.NewFileSet()
	_ = filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return nil
		}
		if parseEntry.IsDir() {
			if shouldSkipMutateDir(parseEntry.Name()) && parsePath != parseRootPath {
				return filepath.SkipDir
			}
			return nil
		}
		parseName := parseEntry.Name()
		if !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") {
			return nil
		}
		if !fileIsClientOnly(parsePath) {
			return nil
		}
		parseFile, parseErr := parser.ParseFile(parseFset, parsePath, nil, parser.ImportsOnly)
		if parseErr != nil {
			return nil
		}
		for _, parseImport := range parseFile.Imports {
			parseImportPath, parseUnquoteErr := strconv.Unquote(parseImport.Path.Value)
			if parseUnquoteErr != nil {
				continue
			}
			if parseReason, parseBad := serverOnlyImports[parseImportPath]; parseBad {
				parsePos := parseFset.Position(parseImport.Pos())
				parseDiagnostics = append(parseDiagnostics, agenticDiagnostic{
					Code:     "GWC-CHECK-SERVER-LEAK",
					Severity: "error",
					Message:  "browser (js && wasm) file imports server-only package " + parseImportPath + " (" + parseReason + "); move it behind a `//go:build !js || !wasm` file or a //gwc:server function",
					File:     relativeSlashPath(parseRootPath, parsePath),
					Line:     parsePos.Line,
				})
			}
		}
		return nil
	})
	return parseDiagnostics
}

// fileIsClientOnly reports whether a file's //go:build constraint makes it browser-only:
// included when js && wasm are set, excluded for an untagged target. A file with no
// constraint (shared) or a server constraint returns false.
func fileIsClientOnly(parsePath string) bool {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return false
	}
	for parseLine := range strings.SplitSeq(string(parseData), "\n") {
		parseLine = strings.TrimSpace(parseLine)
		if !constraint.IsGoBuild(parseLine) {
			if parseLine != "" && !strings.HasPrefix(parseLine, "//") && !strings.HasPrefix(parseLine, "package") {
				return false // reached code with no build line
			}
			continue
		}
		parseExpr, parseParseErr := constraint.Parse(parseLine)
		if parseParseErr != nil {
			return false
		}
		parseClient := parseExpr.Eval(func(parseTag string) bool { return parseTag == "js" || parseTag == "wasm" })
		parseUntagged := parseExpr.Eval(func(string) bool { return false })
		return parseClient && !parseUntagged
	}
	return false
}
