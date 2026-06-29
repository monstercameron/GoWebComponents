package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunImportRequiresOutputForJSON verifies the JSON-only output guard in the import command.
func TestRunImportRequiresOutputForJSON(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolveRepoRoot: %v", parseErr)
	}
	parseSourcePath := filepath.Join(parseT.TempDir(), "catalog.html")
	if parseErr2 := os.WriteFile(parseSourcePath, []byte("<main><h1>Catalog</h1></main>"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile source: %v", parseErr2)
	}

	parseErr = (launcher{repoRoot: parseRepoRoot}).runImport([]string{"-src", parseSourcePath, "-json"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "requires -out when -json is used") {
		parseT.Fatalf("expected json output path error, got %v", parseErr)
	}
}

// TestResolveImportConfigRejectsDirectory verifies source-path validation for import configs.
func TestResolveImportConfigRejectsDirectory(parseT *testing.T) {
	if _, parseErr := (launcher{}).resolveImportConfig(importConfig{sourcePath: parseT.TempDir()}); parseErr == nil || !strings.Contains(parseErr.Error(), "must be a file") {
		parseT.Fatalf("expected directory source error, got %v", parseErr)
	}
}

// TestWriteImportedMainFileSurfacesFormatFailures verifies import output write validation branches.
func TestWriteImportedMainFileSurfacesFormatFailures(parseT *testing.T) {
	if parseErr := writeImportedMainFile("", "package main"); parseErr == nil || !strings.Contains(parseErr.Error(), "output path is required") {
		parseT.Fatalf("expected missing output path error, got %v", parseErr)
	}

	parseOriginalFormat := scaffoldFormatMain
	parseT.Cleanup(func() {
		scaffoldFormatMain = parseOriginalFormat
	})
	scaffoldFormatMain = func(string) error { return errors.New("format failure") }

	parseOutputPath := filepath.Join(parseT.TempDir(), "bin", "main.go")
	if parseErr := writeImportedMainFile(parseOutputPath, "package main\n"); parseErr == nil || !strings.Contains(parseErr.Error(), "format imported main.go") {
		parseT.Fatalf("expected format failure, got %v", parseErr)
	}
}

// TestParseImportedJSXDocumentBuildsHTMLShell verifies JSX shell extraction from an html root element.
func TestParseImportedJSXDocumentBuildsHTMLShell(parseT *testing.T) {
	parseDocument, parseErr := parseImportedJSXDocument(`
		<html lang="en">
			<head>
				<title>Atlas shell</title>
				<meta name="description" content="Atlas" />
			</head>
			<body data-view="dashboard">
				<main><h1>Atlas</h1></main>
			</body>
		</html>
	`)
	if parseErr != nil {
		parseT.Fatalf("parseImportedJSXDocument html shell: %v", parseErr)
	}
	if parseDocument.Lang != "en" || parseDocument.Title != "Atlas shell" {
		parseT.Fatalf("unexpected shell metadata: %#v", parseDocument)
	}
	if len(parseDocument.HeadNodes) != 1 || len(parseDocument.BodyAttrs) != 1 || len(parseDocument.Roots) != 1 {
		parseT.Fatalf("unexpected shell document structure: %#v", parseDocument)
	}
	if parseDocument.Roots[0].Tag != "main" {
		parseT.Fatalf("expected main root, got %#v", parseDocument.Roots)
	}
}

// TestParseImportedJSXParserHandlesFragmentsAndErrors verifies JSX fragment parsing and validation errors.
func TestParseImportedJSXParserHandlesFragmentsAndErrors(parseT *testing.T) {
	parseDocument, parseErr := parseImportedJSXDocument(`
		function App() {
			return <>
				{"hello"}
				{/* note */}
				<section style={{backgroundColor: "red", fontSize: 16}} disabled={true}>Atlas</section>
				{null}
				{false}
			</>
		}
	`)
	if parseErr != nil {
		parseT.Fatalf("parseImportedJSXDocument fragment: %v", parseErr)
	}
	if len(parseDocument.Roots) != 2 {
		parseT.Fatalf("expected fragment roots to flatten, got %#v", parseDocument.Roots)
	}
	if parseDocument.Roots[0].Kind != importedNodeText || parseDocument.Roots[0].Text != "hello" {
		parseT.Fatalf("unexpected first fragment node: %#v", parseDocument.Roots[0])
	}
	if parseDocument.Roots[1].Tag != "section" || parseDocument.Roots[1].Attrs[0].Name != "style" {
		parseT.Fatalf("unexpected section node: %#v", parseDocument.Roots[1])
	}

	if _, parseErr2 := parseImportedJSXDocument(`<div {...props}></div>`); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "spread attributes") {
		parseT.Fatalf("expected spread attribute rejection, got %v", parseErr2)
	}
	if _, parseErr3 := parseImportedJSXDocument(`<div>{items.map(renderItem)}</div>`); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "unsupported JSX child expression") {
		parseT.Fatalf("expected dynamic child rejection, got %v", parseErr3)
	}
	if _, parseErr4 := parseImportedJSXDocument(`<div title=value></div>`); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "expected quoted or braced attribute value") {
		parseT.Fatalf("expected unquoted attr rejection, got %v", parseErr4)
	}
}

// TestJSXParsingHelpersHandleCommentsAndEscapes verifies lower-level JSX parsing helpers.
func TestJSXParsingHelpersHandleCommentsAndEscapes(parseT *testing.T) {
	parseSource := `
		const parseMarkup = "<div>ignored</div>";
		function App() { return <main>{"Atlas"}</main> }
	`
	parseStart, parseErr := findJSXStart(parseSource)
	if parseErr != nil {
		parseT.Fatalf("findJSXStart: %v", parseErr)
	}
	if !strings.HasPrefix(strings.TrimSpace(parseSource[parseStart:]), "<main>") {
		parseT.Fatalf("unexpected JSX start slice: %q", parseSource[parseStart:])
	}
	if parseGot := findKeywordOutsideJSX("returning return value", "return"); parseGot <= 0 {
		parseT.Fatalf("expected standalone return keyword, got %d", parseGot)
	}
	parseCommentSource := `"quoted <div>" // line <div>
/* block <div> */ <main>`
	if parseGot2 := findCharOutsideJSX(parseCommentSource, '<'); parseGot2 < 0 || !strings.HasPrefix(parseCommentSource[parseGot2:], "<main>") {
		parseT.Fatalf("expected char outside strings/comments, got %d", parseGot2)
	}

	parseParser := &jsxParser{source: `"Atlas \"shell\""`}
	parseQuoted, parseErr := parseParser.readQuotedString()
	if parseErr != nil || parseQuoted != `Atlas "shell"` {
		parseT.Fatalf("readQuotedString = %q err=%v", parseQuoted, parseErr)
	}
	parseParser = &jsxParser{source: `{foo: {bar: "baz"}}`}
	parseBalanced, parseErr := parseParser.readBalanced('{', '}')
	if parseErr != nil || strings.TrimSpace(parseBalanced) != `foo: {bar: "baz"}` {
		parseT.Fatalf("readBalanced = %q err=%v", parseBalanced, parseErr)
	}
	parseParser = &jsxParser{source: `"unterminated`}
	if _, parseErr2 := parseParser.readQuotedString(); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "unterminated string literal") {
		parseT.Fatalf("expected unterminated string error, got %v", parseErr2)
	}
	parseParser = &jsxParser{source: `{unterminated`}
	if _, parseErr3 := parseParser.readBalanced('{', '}'); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "unterminated") {
		parseT.Fatalf("expected unterminated balanced error, got %v", parseErr3)
	}
}
