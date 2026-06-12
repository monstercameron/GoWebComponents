package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const agenticEnvelopeSchemaVersion = "gwc.agentic.v1"

type agenticEnvelope struct {
	SchemaVersion string              `json:"schemaVersion"`
	Command       string              `json:"command"`
	OK            bool                `json:"ok"`
	Data          any                 `json:"data,omitempty"`
	Diagnostics   []agenticDiagnostic `json:"diagnostics"`
	Error         *agenticError       `json:"error"`
}

type agenticDiagnostic struct {
	Code       string              `json:"code"`
	Severity   string              `json:"severity,omitempty"`
	Message    string              `json:"message"`
	File       string              `json:"file,omitempty"`
	Line       int                 `json:"line,omitempty"`
	Column     int                 `json:"column,omitempty"`
	Suggestion string              `json:"suggestion,omitempty"`
	Edits      []agenticTextEdit   `json:"edits,omitempty"`
	Attributes map[string]string   `json:"attributes,omitempty"`
	Children   []agenticDiagnostic `json:"children,omitempty"`
}

type agenticTextEdit struct {
	File        string `json:"file"`
	Description string `json:"description,omitempty"`
	OldText     string `json:"oldText,omitempty"`
	NewText     string `json:"newText,omitempty"`
}

type agenticError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// writeAgenticEnvelope writes one stable JSON envelope for agent-facing commands.
func writeAgenticEnvelope(parseCommand string, isParseOK bool, parseData any, parseDiagnostics []agenticDiagnostic, parseErr error) error {
	parseEnvelope := agenticEnvelope{
		SchemaVersion: agenticEnvelopeSchemaVersion,
		Command:       parseCommand,
		OK:            isParseOK,
		Data:          parseData,
		Diagnostics:   parseDiagnostics,
	}
	if parseErr != nil {
		parseEnvelope.Error = &agenticError{
			Code:    "GWC-AGENTIC-COMMAND-FAILED",
			Message: parseErr.Error(),
		}
	}
	if parseEnvelope.Diagnostics == nil {
		parseEnvelope.Diagnostics = []agenticDiagnostic{}
	}
	parseEncoder := json.NewEncoder(os.Stdout)
	parseEncoder.SetIndent("", "  ")
	return parseEncoder.Encode(parseEnvelope)
}

// buildAgenticCommandError converts an error into a reusable diagnostic.
func buildAgenticCommandError(parseCode string, parseErr error) agenticDiagnostic {
	parseMessage := ""
	if parseErr != nil {
		parseMessage = parseErr.Error()
	}
	return agenticDiagnostic{
		Code:     parseCode,
		Severity: "error",
		Message:  parseMessage,
	}
}

// printAgenticHumanSummary prints a compact human-readable command summary.
func printAgenticHumanSummary(parseTitle string, isParseOK bool, parseLines []string) {
	fmt.Println(parseTitle)
	if isParseOK {
		fmt.Println("  status: ok")
	} else {
		fmt.Println("  status: failed")
	}
	for _, parseLine := range parseLines {
		if parseLine != "" {
			fmt.Printf("  %s\n", parseLine)
		}
	}
}

// reorderAgenticKnownFlags moves known flags before positional args so agentic
// commands accept both `cmd --json target` and `cmd target --json`.
func reorderAgenticKnownFlags(parseArgs []string, parseValueFlags []string, parseBoolFlags []string) []string {
	parseValue := map[string]bool{}
	parseBool := map[string]bool{}
	for _, parseFlag := range parseValueFlags {
		parseValue[strings.TrimLeft(strings.ToLower(parseFlag), "-")] = true
	}
	for _, parseFlag := range parseBoolFlags {
		parseBool[strings.TrimLeft(strings.ToLower(parseFlag), "-")] = true
	}
	parseFlags := []string{}
	parsePositionals := []string{}
	for parseIndex := 0; parseIndex < len(parseArgs); parseIndex++ {
		parseArg := parseArgs[parseIndex]
		parseTrimmed := strings.TrimSpace(parseArg)
		if !strings.HasPrefix(parseTrimmed, "-") || parseTrimmed == "-" {
			parsePositionals = append(parsePositionals, parseArg)
			continue
		}
		parseName := strings.TrimLeft(parseTrimmed, "-")
		if parseEq := strings.Index(parseName, "="); parseEq >= 0 {
			parseName = parseName[:parseEq]
		}
		parseName = strings.ToLower(parseName)
		if parseBool[parseName] {
			parseFlags = append(parseFlags, parseArg)
			continue
		}
		if parseValue[parseName] {
			parseFlags = append(parseFlags, parseArg)
			if !strings.Contains(parseArg, "=") && parseIndex+1 < len(parseArgs) {
				parseIndex++
				parseFlags = append(parseFlags, parseArgs[parseIndex])
			}
			continue
		}
		parsePositionals = append(parsePositionals, parseArg)
	}
	return append(parseFlags, parsePositionals...)
}
