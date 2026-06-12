package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRunFmtCheckAndWriteJSON(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parsePath := filepath.Join(parseRoot, "app.go")
	parseOriginal := "package fmtfixture\r\nfunc MissingDoc( ){ }\r\n"
	if parseErr := os.WriteFile(parsePath, []byte(parseOriginal), 0o644); parseErr != nil {
		parseT.Fatalf("write source: %v", parseErr)
	}

	parseEnvelope, parseOutput, parseErr := captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runFmt([]string{"-root", parseRoot, "-check", "-json"})
	})
	if parseErr == nil {
		parseT.Fatal("expected fmt -check to report changes")
	}
	if parseEnvelope.OK || parseEnvelope.Command != "fmt" {
		parseT.Fatalf("unexpected fmt check envelope: %#v\n%s", parseEnvelope, parseOutput)
	}
	parseSummary := fmtSummary{}
	if parseErr = json.Unmarshal(parseEnvelope.Data, &parseSummary); parseErr != nil {
		parseT.Fatalf("decode fmt summary: %v\n%s", parseErr, parseOutput)
	}
	if len(parseSummary.Files) != 1 || !parseSummary.Files[0].WouldChange {
		parseT.Fatalf("expected one would-change file, got %#v", parseSummary.Files)
	}
	parseAfterCheck, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read source after check: %v", parseErr)
	}
	if string(parseAfterCheck) != parseOriginal {
		parseT.Fatalf("fmt -check changed source:\n%s", string(parseAfterCheck))
	}

	parseEnvelope, parseOutput, parseErr = captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runFmt([]string{"-root", parseRoot, "-json"})
	})
	if parseErr != nil {
		parseT.Fatalf("run fmt write: %v\n%s", parseErr, parseOutput)
	}
	if !parseEnvelope.OK {
		parseT.Fatalf("expected fmt write ok, got %#v\n%s", parseEnvelope, parseOutput)
	}
	parseFormatted, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read formatted source: %v", parseErr)
	}
	parseFormattedText := string(parseFormatted)
	if strings.Contains(parseFormattedText, "\r\n") ||
		!strings.Contains(parseFormattedText, "// MissingDoc documents MissingDoc.") ||
		!strings.Contains(parseFormattedText, "func MissingDoc() {}") {
		parseT.Fatalf("unexpected formatted source:\n%s", parseFormattedText)
	}

	parseEnvelope, parseOutput, parseErr = captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runFmt([]string{"-root", parseRoot, "-check", "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("expected formatted source to pass check, err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
}

func TestRunCleanDryRunAndApplyJSON(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for _, parsePath := range []string{
		filepath.Join(parseRoot, "bin", "app.exe"),
		filepath.Join(parseRoot, ".gwc", "cache", "entry"),
		filepath.Join(parseRoot, "dist", "app.wasm"),
	} {
		if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr != nil {
			parseT.Fatalf("mkdir %s: %v", parsePath, parseErr)
		}
		if parseErr := os.WriteFile(parsePath, []byte("generated"), 0o644); parseErr != nil {
			parseT.Fatalf("write %s: %v", parsePath, parseErr)
		}
	}

	parseEnvelope, parseOutput, parseErr := captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runClean([]string{"-root", parseRoot, "-dry-run", "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("run clean dry-run: err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
	parseSummary := cleanSummary{}
	if parseErr = json.Unmarshal(parseEnvelope.Data, &parseSummary); parseErr != nil {
		parseT.Fatalf("decode clean summary: %v\n%s", parseErr, parseOutput)
	}
	if len(parseSummary.Removed) != 3 {
		parseT.Fatalf("expected three dry-run removals, got %#v", parseSummary.Removed)
	}
	for _, parseRemoval := range parseSummary.Removed {
		if parseRemoval.Operation != "would-remove" {
			parseT.Fatalf("expected dry-run operation, got %#v", parseSummary.Removed)
		}
	}
	if !fileExists(filepath.Join(parseRoot, "bin", "app.exe")) {
		parseT.Fatal("dry-run removed bin output")
	}

	parseEnvelope, parseOutput, parseErr = captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runClean([]string{"-root", parseRoot, "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("run clean apply: err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
	for _, parsePath := range []string{"bin", filepath.Join(".gwc", "cache"), "dist"} {
		if fileExists(filepath.Join(parseRoot, parsePath)) {
			parseT.Fatalf("expected clean to remove %s", parsePath)
		}
	}
}

func TestRunDocsAndDeadcodeJSON(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	writeToolchainModule(parseT, parseRoot, map[string]string{
		"api.go": `package docsfixture

// UsedFunc is called by live code.
func UsedFunc() {}

// UnusedFunc is exported but not statically referenced.
func UnusedFunc() {}

func local() {
	UsedFunc()
}
`,
	})

	parseDocsOut := filepath.Join("docs", "api.md")
	parseEnvelope, parseOutput, parseErr := captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runDocs([]string{"-root", parseRoot, "-out", parseDocsOut, "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("run docs: err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
	parseDocsPath := filepath.Join(parseRoot, parseDocsOut)
	parseDocsBytes, parseErr := os.ReadFile(parseDocsPath)
	if parseErr != nil {
		parseT.Fatalf("read docs output: %v", parseErr)
	}
	if !strings.Contains(string(parseDocsBytes), "### UsedFunc") || !strings.Contains(string(parseDocsBytes), "### UnusedFunc") {
		parseT.Fatalf("unexpected docs output:\n%s", string(parseDocsBytes))
	}

	parseEnvelope, parseOutput, parseErr = captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runDeadcode([]string{"-root", parseRoot, "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("run deadcode: err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
	parseDeadcode := deadcodeSummary{}
	if parseErr = json.Unmarshal(parseEnvelope.Data, &parseDeadcode); parseErr != nil {
		parseT.Fatalf("decode deadcode summary: %v\n%s", parseErr, parseOutput)
	}
	parseNames := []string{}
	for _, parseSymbol := range parseDeadcode.Symbols {
		parseNames = append(parseNames, parseSymbol.Name)
	}
	if !slices.Contains(parseNames, "UnusedFunc") {
		parseT.Fatalf("expected UnusedFunc in deadcode report, got %#v", parseDeadcode.Symbols)
	}
}

func TestRunDepsJSONAndDryRunUpdate(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	writeToolchainModule(parseT, parseRoot, nil)

	parseEnvelope, parseOutput, parseErr := captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runDeps([]string{"-root", parseRoot, "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("run deps: err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
	parseSummary := depsSummary{}
	if parseErr = json.Unmarshal(parseEnvelope.Data, &parseSummary); parseErr != nil {
		parseT.Fatalf("decode deps summary: %v\n%s", parseErr, parseOutput)
	}
	if len(parseSummary.Modules) != 1 || !parseSummary.Modules[0].Main {
		parseT.Fatalf("expected only main module, got %#v", parseSummary.Modules)
	}

	parseEnvelope, parseOutput, parseErr = captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runDeps([]string{"-root", parseRoot, "-module", "example.com/lib", "-to", "v1.2.3", "-dry-run", "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("run deps dry-run update: err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
}

func TestBuildSizeSummaryAttributesSymbols(parseT *testing.T) {
	parseArtifact := filepath.Join(parseT.TempDir(), "app.wasm")
	if parseErr := os.WriteFile(parseArtifact, []byte("wasm"), 0o644); parseErr != nil {
		parseT.Fatalf("write artifact: %v", parseErr)
	}
	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
	})
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" || !slices.Equal(parseArgs, []string{"tool", "nm", "-size", parseArtifact}) {
			parseT.Fatalf("unexpected nm command: %s %#v", parseCommand, parseArgs)
		}
		return "0000 32 T example.com/app.Widget\n0001 8 D runtime.x\n", nil
	}

	parseSummary, parseErr := buildSizeSummary(parseArtifact, 10)
	if parseErr != nil {
		parseT.Fatalf("build size summary: %v", parseErr)
	}
	if !parseSummary.OK || parseSummary.AttributedBytes != 40 {
		parseT.Fatalf("unexpected size summary: %#v", parseSummary)
	}
	if len(parseSummary.Packages) == 0 || parseSummary.Packages[0].Package != "app" || parseSummary.Packages[0].Bytes != 32 {
		parseT.Fatalf("expected app package attribution first, got %#v", parseSummary.Packages)
	}
}

func TestRunWatchOnceJSONAndTestWatchAlias(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	writeToolchainModule(parseT, parseRoot, map[string]string{"app_test.go": "package toolchainfixture\n\nimport \"testing\"\n\nfunc TestSmoke(parseT *testing.T) {}\n"})

	parseOriginalRunCommand := launcherRunCommand
	parseOriginalWatchCommand := runWatchCommand
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
		runWatchCommand = parseOriginalWatchCommand
	})
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" || !slices.Equal(parseArgs, []string{"test", "./..."}) || parseCwd != parseRoot {
			parseT.Fatalf("unexpected watch test command: %s %#v @ %s", parseCommand, parseArgs, parseCwd)
		}
		return "ok example.com/toolchainfixture", nil
	}

	parseEnvelope, parseOutput, parseErr := captureAgenticToolchainEnvelope(parseT, func() error {
		return (launcher{}).runWatch([]string{"-root", parseRoot, "-lane", "unit", "-once", "-json"})
	})
	if parseErr != nil || !parseEnvelope.OK {
		parseT.Fatalf("run watch once: err=%v envelope=%#v\n%s", parseErr, parseEnvelope, parseOutput)
	}
	parseSummary := watchSummary{}
	if parseErr = json.Unmarshal(parseEnvelope.Data, &parseSummary); parseErr != nil {
		parseT.Fatalf("decode watch summary: %v\n%s", parseErr, parseOutput)
	}
	if !parseSummary.Once || parseSummary.LastResult == nil || !parseSummary.LastResult.OK {
		parseT.Fatalf("expected one successful watched test result, got %#v", parseSummary)
	}

	var parseWatchArgs []string
	runWatchCommand = func(parseL launcher, parseArgs []string) error {
		parseWatchArgs = append([]string(nil), parseArgs...)
		return nil
	}
	if parseErr = (launcher{}).runTest([]string{"-root", parseRoot, "-lane", "unit", "-watch", "-once", "-json"}); parseErr != nil {
		parseT.Fatalf("run test --watch: %v", parseErr)
	}
	for _, parseFlag := range []string{"-root", parseRoot, "-lane", "unit", "-json", "-once"} {
		if !slices.Contains(parseWatchArgs, parseFlag) {
			parseT.Fatalf("expected watch args to contain %q, got %#v", parseFlag, parseWatchArgs)
		}
	}
}

func captureAgenticToolchainEnvelope(parseT *testing.T, parseRun func() error) (agenticTestEnvelope, string, error) {
	parseT.Helper()
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	parseRunErr := parseRun()
	parseOutput, parseReadErr := parseStdout()
	parseRestoreStdout()
	if parseReadErr != nil {
		parseT.Fatalf("read stdout: %v", parseReadErr)
	}
	parseEnvelope := decodeAgenticTestEnvelope(parseT, parseOutput)
	if parseRunErr != nil && errors.Is(parseRunErr, errLauncherJSONEnvelopeWritten) {
		return parseEnvelope, parseOutput, nil
	}
	return parseEnvelope, parseOutput, parseRunErr
}

func writeToolchainModule(parseT *testing.T, parseRoot string, parseFiles map[string]string) {
	parseT.Helper()
	parseAllFiles := map[string]string{
		"go.mod": "module example.com/toolchainfixture\n\ngo 1.26.0\n",
		"app.go": "package toolchainfixture\n",
	}
	for parsePath, parseContent := range parseFiles {
		parseAllFiles[parsePath] = parseContent
	}
	for parsePath, parseContent := range parseAllFiles {
		parseFullPath := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFullPath), 0o755); parseErr != nil {
			parseT.Fatalf("mkdir %s: %v", parsePath, parseErr)
		}
		if parseErr := os.WriteFile(parseFullPath, []byte(parseContent), 0o644); parseErr != nil {
			parseT.Fatalf("write %s: %v", parsePath, parseErr)
		}
	}
}
