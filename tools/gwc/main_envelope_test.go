package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLauncherCommandRegistryJSONCoverage verifies every registered command and alias advertises JSON.
func TestLauncherCommandRegistryJSONCoverage(parseT *testing.T) {
	parseSeen := map[string]struct{}{}
	for _, parseCommand := range listLauncherCommandRegistry() {
		if strings.TrimSpace(parseCommand.Name) == "" {
			parseT.Fatal("registry command name must not be blank")
		}
		if _, parseExists := parseSeen[parseCommand.Name]; parseExists {
			parseT.Fatalf("duplicate registry command %q", parseCommand.Name)
		}
		parseSeen[parseCommand.Name] = struct{}{}
		if !parseCommand.JSON {
			parseT.Fatalf("registered command %q must support JSON", parseCommand.Name)
		}
		if !launcherCommandSupportsJSON(parseCommand.Name) {
			parseT.Fatalf("launcherCommandSupportsJSON(%q) = false", parseCommand.Name)
		}
		if !launcherJSONRequestedForCommand(parseCommand.Name, []string{"--json"}) {
			parseT.Fatalf("launcherJSONRequestedForCommand(%q, --json) = false", parseCommand.Name)
		}
		if launcherJSONRequestedForCommand(parseCommand.Name, []string{"--json=false"}) {
			parseT.Fatalf("launcherJSONRequestedForCommand(%q, --json=false) = true", parseCommand.Name)
		}
		for _, parseAlias := range parseCommand.Aliases {
			if !launcherCommandSupportsJSON(parseAlias) {
				parseT.Fatalf("alias %q for %q must support JSON", parseAlias, parseCommand.Name)
			}
			if parseGot := normalizeLauncherCommandName(parseAlias, nil); parseGot != parseCommand.Name {
				parseT.Fatalf("normalizeLauncherCommandName(%q) = %q, want %q", parseAlias, parseGot, parseCommand.Name)
			}
		}
	}
	if len(parseSeen) == 0 {
		parseT.Fatal("expected non-empty launcher command registry")
	}
}

// TestLauncherResultEnvelopeWrapsExistingJSONSummary verifies raw command JSON is nested and mirrored.
func TestLauncherResultEnvelopeWrapsExistingJSONSummary(parseT *testing.T) {
	parsePayload := buildLauncherResultEnvelopePayload("build", []string{"-json"}, `{"ok":true,"value":42,"label":"ci"}`, nil)
	assertLauncherEnvelopePayload(parseT, parsePayload)
	if parsePayload["schemaVersion"] != buildLauncherResultEnvelopeSchemaVersion {
		parseT.Fatalf("unexpected schema version: %#v", parsePayload["schemaVersion"])
	}
	if parsePayload["command"] != "build" || parsePayload["ok"] != true {
		parseT.Fatalf("unexpected envelope command/status: %#v", parsePayload)
	}
	if parsePayload["label"] != "ci" {
		parseT.Fatalf("expected top-level data mirror, got %#v", parsePayload)
	}
	parseData, parseOk := parsePayload["data"].(map[string]any)
	if !parseOk || parseData["label"] != "ci" {
		parseT.Fatalf("expected object data payload, got %#v", parsePayload["data"])
	}
}

// TestLauncherResultEnvelopeErrorPath verifies failures keep valid envelope shape and legacy fields.
func TestLauncherResultEnvelopeErrorPath(parseT *testing.T) {
	parsePayload := buildLauncherResultEnvelopePayload("build", []string{"-json"}, `{"ok":false,"partial":true}`, errors.New("go build failed: boom"))
	assertLauncherEnvelopePayload(parseT, parsePayload)
	if parsePayload["ok"] != false {
		parseT.Fatalf("expected failing envelope, got %#v", parsePayload)
	}
	if parsePayload["code"] != "code_failure" || parsePayload["category"] != "code" {
		parseT.Fatalf("expected legacy diagnostic mirrors, got %#v", parsePayload)
	}
	parseDiagnostics, parseOk := parsePayload["diagnostics"].([]launcherEnvelopeDiagnostic)
	if !parseOk || len(parseDiagnostics) != 1 || parseDiagnostics[0].Code != "code_failure" {
		parseT.Fatalf("expected structured diagnostic, got %#v", parsePayload["diagnostics"])
	}
	parseErr, parseOk := parsePayload["error"].(launcherEnvelopeError)
	if !parseOk || parseErr.Code != "code_failure" {
		parseT.Fatalf("expected structured error, got %#v", parsePayload["error"])
	}
}

// TestLauncherEnvelopeSchemaContract verifies the checked-in schema and emitted JSON agree on required fields.
func TestLauncherEnvelopeSchemaContract(parseT *testing.T) {
	parseSchemaBytes, parseErr := os.ReadFile(filepath.Join("gwc-result-envelope.schema.json"))
	if parseErr != nil {
		parseT.Fatalf("read envelope schema: %v", parseErr)
	}
	var parseSchema struct {
		Required []string `json:"required"`
	}
	if parseErr2 := json.Unmarshal(parseSchemaBytes, &parseSchema); parseErr2 != nil {
		parseT.Fatalf("parse envelope schema: %v", parseErr2)
	}

	parseOutput := captureLauncherEnvelopeOutput(parseT, func() error {
		return writeLauncherResultEnvelope(os.Stdout, buildLauncherResultEnvelopePayload("env", []string{"-json"}, `{"ok":true}`, nil))
	})
	var parsePayload map[string]any
	if parseErr3 := json.Unmarshal([]byte(parseOutput), &parsePayload); parseErr3 != nil {
		parseT.Fatalf("parse envelope output: %v\n%s", parseErr3, parseOutput)
	}
	for _, parseField := range parseSchema.Required {
		if _, parseExists := parsePayload[parseField]; !parseExists {
			parseT.Fatalf("schema requires %q but payload omitted it: %#v", parseField, parsePayload)
		}
	}
}

// TestLauncherRunJSONEnvelopeWrapsCommandOutput verifies launcher.run envelopes stubbed command JSON.
func TestLauncherRunJSONEnvelopeWrapsCommandOutput(parseT *testing.T) {
	parseOriginalRunFilesCommand := runFilesCommand
	parseT.Cleanup(func() {
		runFilesCommand = parseOriginalRunFilesCommand
	})
	runFilesCommand = func(parseL launcher, parseArgs []string) error {
		fmt.Fprint(os.Stdout, `{"ok":true,"count":2}`)
		return nil
	}

	parseOutput := captureLauncherEnvelopeOutput(parseT, func() error {
		return (launcher{}).run([]string{"files", "-json"})
	})
	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseOutput), &parsePayload); parseErr != nil {
		parseT.Fatalf("parse launcher envelope: %v\n%s", parseErr, parseOutput)
	}
	assertLauncherEnvelopePayload(parseT, parsePayload)
	if parsePayload["command"] != "files" || parsePayload["ok"] != true {
		parseT.Fatalf("unexpected envelope: %#v", parsePayload)
	}
	if parsePayload["count"] != float64(2) {
		parseT.Fatalf("expected top-level compatibility count, got %#v", parsePayload)
	}
}

// TestLauncherRunJSONEnvelopeFailureReturnsOriginalError verifies JSON failures emit one envelope and keep the original error text.
func TestLauncherRunJSONEnvelopeFailureReturnsOriginalError(parseT *testing.T) {
	parseOriginalRunBuildCommand := runBuildCommand
	parseT.Cleanup(func() {
		runBuildCommand = parseOriginalRunBuildCommand
	})
	runBuildCommand = func(parseL launcher, parseArgs []string) error {
		fmt.Fprint(os.Stdout, `{"ok":false}`)
		return errors.New("go build failed: boom")
	}

	var parseRunErr error
	parseOutput := captureLauncherEnvelopeOutput(parseT, func() error {
		parseRunErr = (launcher{}).run([]string{"build", "-json"})
		return nil
	})
	if parseRunErr == nil || !strings.Contains(parseRunErr.Error(), "go build failed: boom") {
		parseT.Fatalf("expected original command error text, got %v", parseRunErr)
	}
	if !errors.Is(parseRunErr, errLauncherJSONEnvelopeWritten) {
		parseT.Fatalf("expected envelope-written sentinel, got %v", parseRunErr)
	}

	var parsePayload map[string]any
	if parseErr := json.Unmarshal([]byte(parseOutput), &parsePayload); parseErr != nil {
		parseT.Fatalf("parse failure envelope: %v\n%s", parseErr, parseOutput)
	}
	if parsePayload["ok"] != false || parsePayload["code"] != "code_failure" {
		parseT.Fatalf("unexpected failure envelope: %#v", parsePayload)
	}
}

// assertLauncherEnvelopePayload checks the stable envelope fields common to all results.
func assertLauncherEnvelopePayload(parseT *testing.T, parsePayload map[string]any) {
	parseT.Helper()
	for _, parseField := range []string{"schemaVersion", "command", "ok", "data", "diagnostics", "error"} {
		if _, parseExists := parsePayload[parseField]; !parseExists {
			parseT.Fatalf("missing envelope field %q in %#v", parseField, parsePayload)
		}
	}
}

// captureLauncherEnvelopeOutput captures stdout for focused envelope tests.
func captureLauncherEnvelopeOutput(parseT *testing.T, parseRun func() error) string {
	parseT.Helper()
	parseOriginalStdout := os.Stdout
	parseTempFile, parseErr := os.CreateTemp("", "gwc-envelope-test-*.tmp")
	if parseErr != nil {
		parseT.Fatalf("create stdout capture: %v", parseErr)
	}
	parseTempPath := parseTempFile.Name()
	defer os.Remove(parseTempPath)

	os.Stdout = parseTempFile
	parseRunErr := parseRun()
	parseCloseErr := parseTempFile.Close()
	os.Stdout = parseOriginalStdout
	if parseRunErr != nil {
		parseT.Fatalf("captured function failed: %v", parseRunErr)
	}
	if parseCloseErr != nil {
		parseT.Fatalf("close stdout capture: %v", parseCloseErr)
	}
	parseBytes, parseReadErr := os.ReadFile(parseTempPath)
	if parseReadErr != nil {
		parseT.Fatalf("read stdout capture: %v", parseReadErr)
	}
	return strings.TrimSpace(string(parseBytes))
}
