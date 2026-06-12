package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunDevAgentDryRunEmitsNDJSON verifies dev agent mode prints events, not human plan text.
func TestRunDevAgentDryRunEmitsNDJSON(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcdevagent\n\ngo 1.25.0\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr := (launcher{}).run([]string{"dev", "-app", parseMainPath, "-root", parseTempApp, "-dry-run", "-agent"}); parseErr != nil {
		parseT.Fatalf("run dev agent dry-run: %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	if strings.Contains(parseOutput, "GWC dev plan") {
		parseT.Fatalf("agent output should not contain human dev plan text:\n%s", parseOutput)
	}
	parseEvents := parseAgentEvents(parseT, parseOutput)
	if len(parseEvents) < 2 {
		parseT.Fatalf("expected at least plan and summary events, got %#v", parseEvents)
	}
	if parseEvents[0].Event != "dev.plan" || parseEvents[0].Command != "dev" {
		parseT.Fatalf("expected first event to be dev plan, got %#v", parseEvents[0])
	}
	if parseEvents[len(parseEvents)-1].Event != "dev.summary" {
		parseT.Fatalf("expected final event to be dev summary, got %#v", parseEvents[len(parseEvents)-1])
	}
}

// TestRunVerifyAgentEmitsChecksAndTraceRepresentations verifies verify agent evidence.
func TestRunVerifyAgentEmitsChecksAndTraceRepresentations(parseT *testing.T) {
	parseTempApp := parseT.TempDir()
	parseMainPath := filepath.Join(parseTempApp, "main.go")
	if parseErr := os.WriteFile(filepath.Join(parseTempApp, "go.mod"), []byte("module example.com/gwcverifyagent\n\ngo 1.25.0\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write go.mod: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseMainPath, []byte("package main\nfunc main() {}\n"), 0o644); parseErr != nil {
		parseT.Fatalf("write main.go: %v", parseErr)
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr := (launcher{}).run([]string{"verify", "-app", parseMainPath, "-root", parseTempApp, "-skip-tests", "-agent"}); parseErr != nil {
		parseT.Fatalf("run verify agent: %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	parseEvents := parseAgentEvents(parseT, parseOutput)
	parseChecks := map[string]agentVerifyCheckRecord{}
	for _, parseEvent := range parseEvents {
		if parseEvent.Event != "verify.check" {
			continue
		}
		parseEncoded, parseErr := json.Marshal(parseEvent.Data)
		if parseErr != nil {
			parseT.Fatalf("marshal check data: %v", parseErr)
		}
		var parseCheck agentVerifyCheckRecord
		if parseErr := json.Unmarshal(parseEncoded, &parseCheck); parseErr != nil {
			parseT.Fatalf("unmarshal check data: %v", parseErr)
		}
		parseChecks[parseCheck.Name] = parseCheck
	}
	for _, parseName := range []string{"go-test", "wasm-build", "golden-path-audit", "hydration-diff", "commit-trace"} {
		if _, parseOk := parseChecks[parseName]; !parseOk {
			parseT.Fatalf("missing verify check %q in events %#v", parseName, parseChecks)
		}
	}
	if !parseChecks["go-test"].Skipped || parseChecks["wasm-build"].Status != "passed" {
		parseT.Fatalf("unexpected verify checks: %#v", parseChecks)
	}
	if !parseChecks["hydration-diff"].Skipped || !parseChecks["commit-trace"].Skipped {
		parseT.Fatalf("expected trace representations to be explicit skipped checks: %#v", parseChecks)
	}
}

// TestRunObserveAgentFiltersAndRedacts verifies observe agent query and redaction behavior.
func TestRunObserveAgentFiltersAndRedacts(parseT *testing.T) {
	parseLogPath := filepath.Join(parseT.TempDir(), "telemetry.ndjson")
	parsePayload := strings.Join([]string{
		`{"timestamp":"2026-06-12T12:00:00Z","level":"info","attributes":{"route":"/home","build":"abc"}}`,
		`{"timestamp":"2026-06-12T12:01:00Z","level":"error","attributes":{"route":"/checkout","build":"abc","email":"user@example.com","password":"secret"},"message":"checkout failed"}`,
		`{"timestamp":"2026-06-12T12:02:00Z","level":"warn","attributes":{"route":"/checkout","build":"def"}}`,
	}, "\n") + "\n"
	if parseErr := os.WriteFile(parseLogPath, []byte(parsePayload), 0o644); parseErr != nil {
		parseT.Fatalf("write telemetry: %v", parseErr)
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr := (launcher{}).run([]string{"observe", "-source", parseLogPath, "-route", "/checkout", "-build", "abc", "-severity", "error", "-agent"}); parseErr != nil {
		parseT.Fatalf("run observe agent: %v", parseErr)
	}
	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	parseEvents := parseAgentEvents(parseT, parseOutput)
	if len(parseEvents) != 1 || parseEvents[0].Event != "observe.summary" {
		parseT.Fatalf("expected one observe summary event, got %#v", parseEvents)
	}
	parseEncoded, parseErr := json.Marshal(parseEvents[0].Data)
	if parseErr != nil {
		parseT.Fatalf("marshal observe summary: %v", parseErr)
	}
	var parseSummary observeSummary
	if parseErr := json.Unmarshal(parseEncoded, &parseSummary); parseErr != nil {
		parseT.Fatalf("unmarshal observe summary: %v", parseErr)
	}
	if len(parseSummary.Records) != 1 {
		parseT.Fatalf("expected one matching record, got %#v", parseSummary.Records)
	}
	parseRecordJSON, parseErr := json.Marshal(parseSummary.Records[0])
	if parseErr != nil {
		parseT.Fatalf("marshal record: %v", parseErr)
	}
	if strings.Contains(string(parseRecordJSON), "user@example.com") || strings.Contains(string(parseRecordJSON), "secret") {
		parseT.Fatalf("expected PII fields to be redacted, got %s", string(parseRecordJSON))
	}
	if !strings.Contains(string(parseRecordJSON), "[REDACTED]") {
		parseT.Fatalf("expected redacted marker, got %s", string(parseRecordJSON))
	}
}

// parseAgentEvents decodes one NDJSON event stream for tests.
func parseAgentEvents(parseT *testing.T, parseOutput string) []agentEvent {
	parseT.Helper()
	parseLines := strings.Split(strings.TrimSpace(parseOutput), "\n")
	parseEvents := make([]agentEvent, 0, len(parseLines))
	for _, parseLine := range parseLines {
		parseTrimmed := strings.TrimSpace(parseLine)
		if parseTrimmed == "" {
			continue
		}
		var parseEvent agentEvent
		if parseErr := json.Unmarshal([]byte(parseTrimmed), &parseEvent); parseErr != nil {
			parseT.Fatalf("invalid agent event line %q: %v\nfull output:\n%s", parseTrimmed, parseErr, parseOutput)
		}
		if parseEvent.SchemaVersion != agentEventSchemaVersion {
			parseT.Fatalf("unexpected schema version in %#v", parseEvent)
		}
		parseEvents = append(parseEvents, parseEvent)
	}
	return parseEvents
}
