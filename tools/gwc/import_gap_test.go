package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunImportCoversHelpStdoutAndJSON verifies import command output branches.
func TestRunImportCoversHelpStdoutAndJSON(parseT *testing.T) {
	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolveRepoRoot: %v", parseErr)
	}
	parseSourcePath := filepath.Join(parseT.TempDir(), "catalog.html")
	if parseErr2 := os.WriteFile(parseSourcePath, []byte("<main><h1>Catalog</h1></main>"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile source: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("captureExamplesStdout: %v", parseErr3)
	}
	parseT.Cleanup(parseRestoreStdout)

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	if parseErr4 := parseLauncher.runImport([]string{"-h"}); parseErr4 != nil {
		parseT.Fatalf("runImport help: %v", parseErr4)
	}
	if parseErr5 := parseLauncher.runImport([]string{"-src", parseSourcePath}); parseErr5 != nil {
		parseT.Fatalf("runImport stdout: %v", parseErr5)
	}

	parseOutputPath := filepath.Join(parseT.TempDir(), "bin", "main.go")
	if parseErr6 := parseLauncher.runImport([]string{"-src", parseSourcePath, "-out", parseOutputPath, "-json"}); parseErr6 != nil {
		parseT.Fatalf("runImport json: %v", parseErr6)
	}

	parsePrinted, parseErr7 := parseStdout()
	if parseErr7 != nil {
		parseT.Fatalf("read stdout: %v", parseErr7)
	}
	if !strings.Contains(parsePrinted, "Usage of import:") {
		parseT.Fatalf("expected help text, got %q", parsePrinted)
	}
	if !strings.Contains(parsePrinted, "package main") || !strings.Contains(parsePrinted, "func App() ui.Node") {
		parseT.Fatalf("expected generated main.go on stdout, got %q", parsePrinted)
	}

	parseJSONStart := strings.LastIndex(parsePrinted, "{")
	if parseJSONStart < 0 {
		parseT.Fatalf("expected json summary, got %q", parsePrinted)
	}
	var parseSummary importSummary
	if parseErr8 := json.Unmarshal([]byte(parsePrinted[parseJSONStart:]), &parseSummary); parseErr8 != nil {
		parseT.Fatalf("json unmarshal summary: %v\n%s", parseErr8, parsePrinted)
	}
	if !parseSummary.OK || parseSummary.SourceKind != "html" || parseSummary.OutputPath != parseOutputPath {
		parseT.Fatalf("unexpected summary: %#v", parseSummary)
	}
}

// TestResolveImportConfigCoversGetwdRelativeOutputAndSourceErrors verifies import config resolution branches.
func TestResolveImportConfigCoversGetwdRelativeOutputAndSourceErrors(parseT *testing.T) {
	parseOriginalGetwd := buildGetwd
	parseT.Cleanup(func() {
		buildGetwd = parseOriginalGetwd
	})

	buildGetwd = func() (string, error) {
		return "", errors.New("cwd failed")
	}
	if _, parseErr := (launcher{}).resolveImportConfig(importConfig{sourcePath: "catalog.html"}); parseErr == nil || !strings.Contains(parseErr.Error(), "cwd failed") {
		parseT.Fatalf("expected cwd failure, got %v", parseErr)
	}

	parseRootPath := parseT.TempDir()
	parseSourcePath := filepath.Join(parseRootPath, "catalog.html")
	if parseErr2 := os.WriteFile(parseSourcePath, []byte("<main><p>Atlas</p></main>"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile source: %v", parseErr2)
	}

	buildGetwd = func() (string, error) {
		return parseRootPath, nil
	}
	if _, parseErr3 := (launcher{}).resolveImportConfig(importConfig{}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "requires -src") {
		parseT.Fatalf("expected missing source error, got %v", parseErr3)
	}

	parseConfig, parseErr4 := (launcher{}).resolveImportConfig(importConfig{
		sourcePath: "catalog.html",
		outputPath: filepath.Join("bin", "main.go"),
	})
	if parseErr4 != nil {
		parseT.Fatalf("resolveImportConfig success: %v", parseErr4)
	}
	if parseConfig.sourcePath != parseSourcePath {
		parseT.Fatalf("expected absolute source path %q, got %q", parseSourcePath, parseConfig.sourcePath)
	}
	parseExpectedOutputPath := filepath.Join(parseRootPath, "bin", "main.go")
	if parseConfig.outputPath != parseExpectedOutputPath {
		parseT.Fatalf("expected absolute output path %q, got %q", parseExpectedOutputPath, parseConfig.outputPath)
	}
	if parseConfig.document.SourceKind != "html" || len(parseConfig.document.Roots) != 1 {
		parseT.Fatalf("unexpected parsed document: %#v", parseConfig.document)
	}
}

// TestWriteImportedMainFileCoversWriteFailure verifies scaffold write failures surface clearly.
func TestWriteImportedMainFileCoversWriteFailure(parseT *testing.T) {
	parseOriginalWriteFile := scaffoldWriteFile
	parseOriginalFormat := scaffoldFormatMain
	parseT.Cleanup(func() {
		scaffoldWriteFile = parseOriginalWriteFile
		scaffoldFormatMain = parseOriginalFormat
	})

	scaffoldWriteFile = func(string, []byte, os.FileMode) error {
		return errors.New("disk full")
	}
	scaffoldFormatMain = func(string) error {
		parseT.Fatal("scaffoldFormatMain should not run after write failure")
		return nil
	}

	parseOutputPath := filepath.Join(parseT.TempDir(), "bin", "main.go")
	if parseErr := writeImportedMainFile(parseOutputPath, "package main\n"); parseErr == nil || !strings.Contains(parseErr.Error(), "write imported main.go: disk full") {
		parseT.Fatalf("expected write failure, got %v", parseErr)
	}
}

// TestImportParserHelpersCoverRemainingBranches verifies parser and document helper edge branches.
func TestImportParserHelpersCoverRemainingBranches(parseT *testing.T) {
	if _, parseErr := parseImportedDocument("catalog.svg", []byte("<svg></svg>")); parseErr == nil || !strings.Contains(parseErr.Error(), "supports only .html, .htm, .jsx, or .tsx files") {
		parseT.Fatalf("expected unsupported kind error, got %v", parseErr)
	}
	if _, parseErr2 := parseImportedDocument("catalog.jsx", []byte("const value = 1;")); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "could not find a static JSX root") {
		parseT.Fatalf("expected jsx root error, got %v", parseErr2)
	}
	if parseStart, parseErr3 := findJSXStart("const view = true; const root = <main />;"); parseErr3 != nil || parseStart < 0 {
		parseT.Fatalf("findJSXStart fallback = %d err=%v", parseStart, parseErr3)
	}

	parseParser := &jsxParser{source: "{<> <span /> <strong /> </>}"}
	parseNode, parseErr4 := parseParser.parseNode()
	if parseErr4 != nil {
		parseT.Fatalf("parseNode fragment expression: %v", parseErr4)
	}
	if parseNode.Kind != importedNodeFragment || len(parseNode.Children) != 2 {
		parseT.Fatalf("unexpected fragment node: %#v", parseNode)
	}

	parseParser = &jsxParser{source: "  Atlas   shell "}
	parseNode, parseErr4 = parseParser.parseNode()
	if parseErr4 != nil {
		parseT.Fatalf("parseNode text: %v", parseErr4)
	}
	if parseNode.Kind != importedNodeText || parseNode.Text != "Atlas shell" {
		parseT.Fatalf("unexpected text node: %#v", parseNode)
	}

	parseParser = &jsxParser{source: "<><span />"}
	if _, parseErr5 := parseParser.parseElement(); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "expected closing fragment </>") {
		parseT.Fatalf("expected fragment closing error, got %v", parseErr5)
	}

	parseParser = &jsxParser{source: "</section >"}
	parseClosingTag, parseErr6 := parseParser.parseClosingTag()
	if parseErr6 != nil || parseClosingTag != "section" {
		parseT.Fatalf("parseClosingTag = %q err=%v", parseClosingTag, parseErr6)
	}

	if _, parseErr7 := splitTopLevel("a:1}", ','); parseErr7 == nil || !strings.Contains(parseErr7.Error(), "unexpected closing delimiter") {
		parseT.Fatalf("expected unexpected closing delimiter error, got %v", parseErr7)
	}

	parseStyle, parseErr8 := parseImportedJSXStyleObject(`"--brand": "blue", content: "a:b", note: "atlas"`)
	if parseErr8 != nil {
		parseT.Fatalf("parseImportedJSXStyleObject: %v", parseErr8)
	}
	if parseStyle["--brand"] != "blue" || parseStyle["content"] != "a:b" || parseStyle["note"] != "atlas" {
		parseT.Fatalf("unexpected style map: %#v", parseStyle)
	}
}

// TestImportedBuilderNameCoversKnownMappings verifies builder lookup for supported tags.
func TestImportedBuilderNameCoversKnownMappings(parseT *testing.T) {
	parseCases := map[string]string{
		"a": "A", "article": "Article", "aside": "Aside", "blockquote": "Blockquote", "br": "Br",
		"button": "Button", "code": "Code", "dialog": "Dialog", "div": "Div", "em": "Em",
		"fieldset": "Fieldset", "footer": "Footer", "form": "Form", "h1": "H1", "h2": "H2",
		"h3": "H3", "h4": "H4", "h5": "H5", "h6": "H6", "header": "Header", "hr": "Hr",
		"img": "Img", "input": "Input", "label": "Label", "legend": "Legend", "li": "Li",
		"main": "Main", "nav": "Nav", "option": "Option", "p": "P", "pre": "Pre",
		"section": "Section", "select": "Select", "small": "Small", "span": "Span",
		"strong": "Strong", "textarea": "Textarea", "time": "Time", "ul": "Ul",
	}
	for parseTag, parseWantBuilder := range parseCases {
		parseBuilder, isParseTyped := importedBuilderName(parseTag)
		if parseBuilder != parseWantBuilder || !isParseTyped {
			parseT.Fatalf("importedBuilderName(%q) = (%q, %t), want (%q, true)", parseTag, parseBuilder, isParseTyped, parseWantBuilder)
		}
	}
}

// TestRenderImportedPropsLiteralCoversRecognizedFields verifies the large props switch renders all major field groups.
func TestRenderImportedPropsLiteralCoversRecognizedFields(parseT *testing.T) {
	parseAttrs := []importedAttr{
		{Name: "id", Value: importedValue{Kind: importedValueString, String: "hero"}},
		{Name: "className", Value: importedValue{Kind: importedValueString, String: "shell"}},
		{Name: "key", Value: importedValue{Kind: importedValueString, String: "row-1"}},
		{Name: "slot", Value: importedValue{Kind: importedValueString, String: "summary"}},
		{Name: "title", Value: importedValue{Kind: importedValueString, String: "Atlas"}},
		{Name: "type", Value: importedValue{Kind: importedValueString, String: "button"}},
		{Name: "name", Value: importedValue{Kind: importedValueString, String: "mode"}},
		{Name: "value", Value: importedValue{Kind: importedValueString, String: "sale"}},
		{Name: "placeholder", Value: importedValue{Kind: importedValueString, String: "Search"}},
		{Name: "accept", Value: importedValue{Kind: importedValueString, String: "image/*"}},
		{Name: "href", Value: importedValue{Kind: importedValueString, String: "/catalog"}},
		{Name: "src", Value: importedValue{Kind: importedValueString, String: "/hero.png"}},
		{Name: "alt", Value: importedValue{Kind: importedValueString, String: "Hero"}},
		{Name: "htmlFor", Value: importedValue{Kind: importedValueString, String: "email"}},
		{Name: "role", Value: importedValue{Kind: importedValueString, String: "status"}},
		{Name: "target", Value: importedValue{Kind: importedValueString, String: "_blank"}},
		{Name: "rel", Value: importedValue{Kind: importedValueString, String: "noopener"}},
		{Name: "as", Value: importedValue{Kind: importedValueString, String: "image"}},
		{Name: "action", Value: importedValue{Kind: importedValueString, String: "/submit"}},
		{Name: "method", Value: importedValue{Kind: importedValueString, String: "post"}},
		{Name: "encType", Value: importedValue{Kind: importedValueString, String: "multipart/form-data"}},
		{Name: "autoComplete", Value: importedValue{Kind: importedValueString, String: "email"}},
		{Name: "min", Value: importedValue{Kind: importedValueString, String: "1"}},
		{Name: "max", Value: importedValue{Kind: importedValueString, String: "9"}},
		{Name: "step", Value: importedValue{Kind: importedValueString, String: "2"}},
		{Name: "rows", Value: importedValue{Kind: importedValueNumber, Number: "4"}},
		{Name: "cols", Value: importedValue{Kind: importedValueString, String: "8"}},
		{Name: "checked", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "disabled", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "selected", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "required", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "readOnly", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "hidden", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "multiple", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "autoFocus", Value: importedValue{Kind: importedValueBool, Bool: true}},
		{Name: "style", Value: importedValue{Kind: importedValueStyle, Style: map[string]string{"color": "#fff", "z-index": "4"}}},
		{Name: "data-state", Value: importedValue{Kind: importedValueString, String: "ready"}},
		{Name: "aria-live", Value: importedValue{Kind: importedValueString, String: "polite"}},
		{Name: "priority", Value: importedValue{Kind: importedValueNumber, Number: "7"}},
		{Name: "featured", Value: importedValue{Kind: importedValueBool, Bool: false}},
		{Name: "note", Value: importedValue{Kind: importedValueString, String: "atlas"}},
		{Name: "", Value: importedValue{Kind: importedValueString, String: "skip"}},
		{Name: "ignored", Value: importedValue{Kind: importedValueNull}},
	}

	parseRendered, parseErr := renderImportedPropsLiteral(parseAttrs, "\t")
	if parseErr != nil {
		parseT.Fatalf("renderImportedPropsLiteral: %v", parseErr)
	}
	for _, parseExpected := range []string{
		`ID: "hero"`, `Class: "shell"`, `Key: "row-1"`, `Slot: "summary"`, `Title: "Atlas"`,
		`Type: "button"`, `Name: "mode"`, `Value: "sale"`, `Placeholder: "Search"`,
		`Accept: "image/*"`, `Href: "/catalog"`, `Src: "/hero.png"`, `Alt: "Hero"`,
		`For: "email"`, `Role: "status"`, `Target: "_blank"`, `Rel: "noopener"`,
		`As: "image"`, `Action: "/submit"`, `Method: "post"`, `EncType: "multipart/form-data"`,
		`AutoComplete: "email"`, `Min: "1"`, `Max: "9"`, `Step: "2"`, `Rows: 4`, `Cols: 8`,
		`Checked: true`, `Disabled: true`, `Selected: true`, `Required: true`, `ReadOnly: true`,
		`Hidden: true`, `Multiple: true`, `AutoFocus: true`, `Style: map[string]string{`,
		`"color": "#fff"`, `"z-index": "4"`, `Data: map[string]string{`, `"state": "ready"`,
		`Aria: map[string]string{`, `"live": "polite"`, `Raw: map[string]interface{}{`,
		`"priority": 7`, `"featured": false`, `"note": "atlas"`,
	} {
		if !strings.Contains(parseRendered, parseExpected) {
			parseT.Fatalf("expected rendered props to contain %q\n%s", parseExpected, parseRendered)
		}
	}
}

// TestRenderImportedMainAndTempDirCoversErrorBranches verifies remaining import renderer and temp-dir branches.
func TestRenderImportedMainAndTempDirCoversErrorBranches(parseT *testing.T) {
	parseMain, parseErr := renderImportedMain(importedDocument{Roots: []importedNode{{Kind: importedNodeText, Text: "Atlas"}}}, "example.com/repo")
	if parseErr != nil {
		parseT.Fatalf("renderImportedMain text root: %v", parseErr)
	}
	if !strings.Contains(parseMain, `"example.com/repo/html"`) || !strings.Contains(parseMain, `ui.Render(ui.CreateElement(App), "#gwc-import-root")`) {
		parseT.Fatalf("unexpected main.go contents:\n%s", parseMain)
	}

	if _, parseErr2 := renderImportedMain(importedDocument{Roots: []importedNode{{Kind: importedNodeKind("mystery")}}}, "example.com/repo"); parseErr2 == nil || !strings.Contains(parseErr2.Error(), `unsupported imported node kind "mystery"`) {
		parseT.Fatalf("expected renderImportedMain error, got %v", parseErr2)
	}

	parseOriginalGetwd := buildGetwd
	parseOriginalMkdirTemp := launcherMkdirTemp
	parseT.Cleanup(func() {
		buildGetwd = parseOriginalGetwd
		launcherMkdirTemp = parseOriginalMkdirTemp
	})

	parseRootPath := parseT.TempDir()
	buildGetwd = func() (string, error) {
		return parseRootPath, nil
	}
	parsePath, parseErr3 := createLauncherTempDir("", "gwc-import-")
	if parseErr3 != nil {
		parseT.Fatalf("createLauncherTempDir cwd fallback: %v", parseErr3)
	}
	if !strings.HasPrefix(parsePath, filepath.Join(parseRootPath, "bin", "tmp")+string(os.PathSeparator)) {
		parseT.Fatalf("expected cwd fallback temp dir under bin/tmp, got %q", parsePath)
	}

	launcherMkdirTemp = func(string, string) (string, error) {
		return "", errors.New("mktemp failed")
	}
	if _, parseErr4 := createLauncherTempDir(parseRootPath, "gwc-import-"); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "create launcher temp directory: mktemp failed") {
		parseT.Fatalf("expected mkdir temp failure, got %v", parseErr4)
	}
}
