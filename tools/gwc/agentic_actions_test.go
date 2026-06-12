package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type agenticTestEnvelope struct {
	SchemaVersion string              `json:"schemaVersion"`
	Command       string              `json:"command"`
	OK            bool                `json:"ok"`
	Data          json.RawMessage     `json:"data"`
	Diagnostics   []agenticDiagnostic `json:"diagnostics"`
}

func decodeAgenticTestEnvelope(parseT *testing.T, parseOutput string) agenticTestEnvelope {
	parseT.Helper()
	parseEnvelope := agenticTestEnvelope{}
	if parseErr := json.Unmarshal([]byte(parseOutput), &parseEnvelope); parseErr != nil {
		parseT.Fatalf("decode agentic envelope: %v\n%s", parseErr, parseOutput)
	}
	return parseEnvelope
}

func TestRunHelpJSONIncludesAgenticCommands(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	parseT.Cleanup(parseRestoreStdout)
	if parseErr := (launcher{}).runHelp([]string{"-json"}); parseErr != nil {
		parseT.Fatalf("run help json: %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	parseEnvelope := decodeAgenticTestEnvelope(parseT, parseOutput)
	if !parseEnvelope.OK || parseEnvelope.Command != "help" {
		parseT.Fatalf("unexpected help envelope: %#v", parseEnvelope)
	}
	parseReport := gwcHelpReport{}
	if parseErr := json.Unmarshal(parseEnvelope.Data, &parseReport); parseErr != nil {
		parseT.Fatalf("decode help report: %v", parseErr)
	}
	parseSeen := map[string]bool{}
	for _, parseCommand := range parseReport.Commands {
		parseSeen[parseCommand.Name] = true
	}
	for _, parseName := range []string{"check", "mcp", "mutate", "scaffold"} {
		if !parseSeen[parseName] {
			parseT.Fatalf("expected help JSON to include %q, got %#v", parseName, parseSeen)
		}
	}
}

func TestRunMutateRenameIdentDryRunDoesNotTouchStrings(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseSourcePath := filepath.Join(parseRoot, "app.go")
	parseSource := `package app

// OldName is a component.
func OldName() string {
	parseLabel := "OldName"
	return parseLabel
}
`
	if parseErr := os.WriteFile(parseSourcePath, []byte(parseSource), 0o644); parseErr != nil {
		parseT.Fatalf("write source: %v", parseErr)
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	parseT.Cleanup(parseRestoreStdout)
	if parseErr := (launcher{}).runMutate([]string{"rename-ident", "-root", parseRoot, "-from", "OldName", "-to", "NewName", "-dry-run", "-json"}); parseErr != nil {
		parseT.Fatalf("run mutate dry-run: %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	parseEnvelope := decodeAgenticTestEnvelope(parseT, parseOutput)
	if !parseEnvelope.OK || parseEnvelope.Command != "mutate" {
		parseT.Fatalf("unexpected mutate envelope: %#v", parseEnvelope)
	}
	parseAfter, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		parseT.Fatalf("read source after dry-run: %v", parseErr)
	}
	if string(parseAfter) != parseSource {
		parseT.Fatalf("dry-run changed source:\n%s", string(parseAfter))
	}
	if !strings.Contains(parseOutput, "-func OldName() string") || !strings.Contains(parseOutput, "+func NewName() string") {
		parseT.Fatalf("expected dry-run diff to rename function only, got:\n%s", parseOutput)
	}
	if !strings.Contains(parseOutput, `\"OldName\"`) {
		parseT.Fatalf("expected string literal to remain in diff context/output, got:\n%s", parseOutput)
	}
}

func TestRunScaffoldComponentJSONWritesFile(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	parseT.Cleanup(parseRestoreStdout)
	if parseErr := (launcher{}).runAgenticScaffold([]string{"component", "-root", parseRoot, "-name", "ProfileCard", "-json", "--no-input"}); parseErr != nil {
		parseT.Fatalf("run scaffold component: %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	parseEnvelope := decodeAgenticTestEnvelope(parseT, parseOutput)
	if !parseEnvelope.OK || parseEnvelope.Command != "scaffold" {
		parseT.Fatalf("unexpected scaffold envelope: %#v", parseEnvelope)
	}
	parseGeneratedPath := filepath.Join(parseRoot, "components", "profile_card.go")
	parseContent, parseErr := os.ReadFile(parseGeneratedPath)
	if parseErr != nil {
		parseT.Fatalf("read generated component: %v", parseErr)
	}
	if !strings.Contains(string(parseContent), "func ProfileCard() ui.Node") || !strings.Contains(string(parseContent), "// ProfileCard renders") {
		parseT.Fatalf("unexpected generated component:\n%s", string(parseContent))
	}
}

func TestRunCheckJSONReportsConventionDiagnostics(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/checkfixture\n\ngo 1.26.0\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "app.go"), []byte("package checkfixture\n\nfunc MissingDoc() {}\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write app.go: %v", parseErr)
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	parseT.Cleanup(parseRestoreStdout)
	parseErr = (launcher{}).runCheck([]string{"-root", parseRoot, "-skip-tests", "-json"})
	if parseErr == nil {
		parseT.Fatal("expected check to fail on missing GoDoc")
	}
	parseOutput, parseErr2 := parseStdout()
	if parseErr2 != nil {
		parseT.Fatalf("read stdout: %v", parseErr2)
	}
	parseEnvelope := decodeAgenticTestEnvelope(parseT, parseOutput)
	if parseEnvelope.OK {
		parseT.Fatalf("expected failed check envelope, got %#v", parseEnvelope)
	}
	if len(parseEnvelope.Diagnostics) == 0 || parseEnvelope.Diagnostics[0].Code != "GWC-CONVENTION-GODOC" {
		parseT.Fatalf("expected GoDoc diagnostic, got %#v", parseEnvelope.Diagnostics)
	}
}

func TestMCPManifestAndCheckToolCall(parseT *testing.T) {
	parseManifest := buildMCPManifest()
	parseToolNames := map[string]bool{}
	for _, parseTool := range parseManifest.Tools {
		parseToolNames[parseTool.Name] = true
	}
	if !parseToolNames["gwc_check"] || !parseToolNames["gwc_mutate"] || !parseToolNames["gwc_scaffold"] {
		parseT.Fatalf("expected agentic tools in manifest, got %#v", parseToolNames)
	}
	if parseToolNames["gwc_mcp"] {
		parseT.Fatalf("mcp server should not expose itself as a tool")
	}

	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/mcpfixture\n\ngo 1.26.0\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "app.go"), []byte("package mcpfixture\n\nfunc parseHelper() {}\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write app.go: %v", parseErr)
	}
	parseArguments, parseErr := json.Marshal(map[string]any{
		"name":      "gwc_check",
		"arguments": map[string]any{"args": []string{"-root", parseRoot, "-skip-tests"}},
	})
	if parseErr != nil {
		parseT.Fatalf("marshal tool call: %v", parseErr)
	}
	parseResult, parseErr := executeMCPToolCall(launcher{}, parseArguments)
	if parseErr != nil {
		parseT.Fatalf("execute mcp tool: %v", parseErr)
	}
	if parseResult["isError"].(bool) {
		parseT.Fatalf("expected successful mcp check tool call, got %#v", parseResult)
	}
	parseContent := parseResult["content"].([]map[string]interface{})
	parseText := parseContent[0]["text"].(string)
	if !strings.Contains(parseText, `"command": "check"`) || !strings.Contains(parseText, `"ok": true`) {
		parseT.Fatalf("expected check envelope in MCP content, got:\n%s", parseText)
	}
}
