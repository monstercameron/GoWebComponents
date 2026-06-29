package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildExportTestReportWritesDeterministicTest(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseRecordingPath := filepath.Join(parseRoot, "recording.json")
	parseOutPath := filepath.Join(parseRoot, "counter_recording_test.go")
	parseRecording := `{
  "name": "counter flow",
  "component": "CounterApp",
  "commands": [
    {"name": "bridge.emit", "payload": {"id": "increment", "event": "click"}},
    {"name": "bridge.wait-for", "payload": {"query": {"text": "Count: 1"}}}
  ],
  "finalSnapshot": {"name": "App", "agentRef": "app", "children": [{"name": "Output", "text": "Count: 1"}]}
}`
	if parseErr := os.WriteFile(parseRecordingPath, []byte(parseRecording), 0o644); parseErr != nil {
		parseT.Fatalf("write recording: %v", parseErr)
	}
	parseReport, parseErr := buildExportTestReport(exportTestConfig{
		recordingPath: parseRecordingPath,
		outPath:       parseOutPath,
		packageName:   "main",
		testName:      "TestCounterRecording",
	})
	if parseErr != nil {
		parseT.Fatalf("export test report: %v", parseErr)
	}
	if !parseReport.OK || parseReport.CommandCount != 2 {
		parseT.Fatalf("unexpected report: %#v", parseReport)
	}
	parseSource, parseErr := os.ReadFile(parseOutPath)
	if parseErr != nil {
		parseT.Fatalf("read generated test: %v", parseErr)
	}
	parseText := string(parseSource)
	for _, parseNeedle := range []string{
		"func TestCounterRecording(parseT *testing.T)",
		"parseFixture.Render(ui.CreateElement(CounterApp))",
		`parseFixture.DispatchByID("increment", "click", render.Event{})`,
		`parseFixture.ByText("Count: 1")`,
		`strings.Contains(parseFixture.Text(), "Count: 1")`,
	} {
		if !strings.Contains(parseText, parseNeedle) {
			parseT.Fatalf("generated test missing %q:\n%s", parseNeedle, parseText)
		}
	}
	if _, parseErr := parser.ParseFile(token.NewFileSet(), parseOutPath, parseSource, parser.ParseComments); parseErr != nil {
		parseT.Fatalf("generated test should parse: %v\n%s", parseErr, parseText)
	}

	parseReport2, parseErr := buildExportTestReport(exportTestConfig{
		recordingPath: parseRecordingPath,
		packageName:   "main",
		testName:      "TestCounterRecording",
	})
	if parseErr != nil {
		parseT.Fatalf("second export test report: %v", parseErr)
	}
	if parseReport.Source != parseReport2.Source {
		parseT.Fatalf("expected deterministic generated source")
	}
}
