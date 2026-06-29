package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
)

var runSnapshotDiffCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runSnapshotDiff(parseArgs)
}

type snapshotDiffConfig struct {
	beforePath string
	afterPath  string
	json       bool
}

// snapshotDiffReport is the result of `gwc snapshot-diff`. Added/Removed/Changed each list
// the top-level atom keys (snapshot map keys) that were added, removed, or whose value
// changed between the two snapshots — not JSON pointers or flattened paths.
type snapshotDiffReport struct {
	OK      bool     `json:"ok"`
	Before  string   `json:"before"`
	After   string   `json:"after"`
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
	Changed []string `json:"changed,omitempty"`
}

func (parseL launcher) runSnapshotDiff(parseArgs []string) error {
	parseArgs = reorderAgenticKnownFlags(parseArgs, []string{"before", "after"}, []string{"json"})
	parseFlags := flag.NewFlagSet("snapshot-diff", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseBefore := parseFlags.String("before", "", "Path to the before bridge snapshot JSON")
	parseAfter := parseFlags.String("after", "", "Path to the after bridge snapshot JSON")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig := snapshotDiffConfig{beforePath: strings.TrimSpace(*parseBefore), afterPath: strings.TrimSpace(*parseAfter), json: *parseJSON}
	parseReport, parseErr := buildSnapshotDiffReport(parseConfig)
	if parseConfig.json {
		parseDiagnostics := []agenticDiagnostic(nil)
		if parseErr != nil {
			parseDiagnostics = append(parseDiagnostics, buildAgenticCommandError("GWC-SNAPSHOT-DIFF", parseErr))
		}
		if parseWriteErr := writeAgenticEnvelope("snapshot-diff", parseErr == nil, parseReport, parseDiagnostics, parseErr); parseWriteErr != nil {
			return parseWriteErr
		}
	}
	if parseErr != nil {
		return parseErr
	}
	if !parseConfig.json {
		printAgenticHumanSummary("GWC snapshot-diff", true, []string{
			fmt.Sprintf("added: %d", len(parseReport.Added)),
			fmt.Sprintf("removed: %d", len(parseReport.Removed)),
			fmt.Sprintf("changed: %d", len(parseReport.Changed)),
		})
	}
	return nil
}

func buildSnapshotDiffReport(parseConfig snapshotDiffConfig) (snapshotDiffReport, error) {
	parseReport := snapshotDiffReport{Before: parseConfig.beforePath, After: parseConfig.afterPath}
	if parseConfig.beforePath == "" || parseConfig.afterPath == "" {
		return parseReport, fmt.Errorf("snapshot-diff requires -before and -after")
	}
	parseBefore, parseErr := readSnapshotDiffFile(parseConfig.beforePath)
	if parseErr != nil {
		return parseReport, parseErr
	}
	parseAfter, parseErr := readSnapshotDiffFile(parseConfig.afterPath)
	if parseErr != nil {
		return parseReport, parseErr
	}
	parseBeforeNodes := map[string]string{}
	parseAfterNodes := map[string]string{}
	flattenSnapshotDiffNodes(parseBefore, "root", parseBeforeNodes)
	flattenSnapshotDiffNodes(parseAfter, "root", parseAfterNodes)
	for parseRef, parseAfterJSON := range parseAfterNodes {
		parseBeforeJSON, parseExists := parseBeforeNodes[parseRef]
		if !parseExists {
			parseReport.Added = append(parseReport.Added, parseRef)
			continue
		}
		if parseBeforeJSON != parseAfterJSON {
			parseReport.Changed = append(parseReport.Changed, parseRef)
		}
	}
	for parseRef := range parseBeforeNodes {
		if _, parseExists := parseAfterNodes[parseRef]; !parseExists {
			parseReport.Removed = append(parseReport.Removed, parseRef)
		}
	}
	sort.Strings(parseReport.Added)
	sort.Strings(parseReport.Removed)
	sort.Strings(parseReport.Changed)
	parseReport.OK = true
	return parseReport, nil
}

func readSnapshotDiffFile(parsePath string) (any, error) {
	parseRaw, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read snapshot %s: %w", parsePath, parseErr)
	}
	var parseValue any
	if parseErr := json.Unmarshal(parseRaw, &parseValue); parseErr != nil {
		return nil, fmt.Errorf("decode snapshot %s: %w", parsePath, parseErr)
	}
	return parseValue, nil
}

func flattenSnapshotDiffNodes(parseValue any, parseFallback string, parseOut map[string]string) {
	switch parseTyped := parseValue.(type) {
	case map[string]any:
		parseRef := parseFallback
		if parseAgentRef, parseOK := parseTyped["agentRef"].(string); parseOK && strings.TrimSpace(parseAgentRef) != "" {
			parseRef = parseAgentRef
		}
		if _, parseHasName := parseTyped["name"]; parseHasName {
			parseOut[parseRef] = canonicalSnapshotDiffJSON(parseTyped)
		}
		for parseKey, parseChild := range parseTyped {
			flattenSnapshotDiffNodes(parseChild, parseRef+"/"+parseKey, parseOut)
		}
	case []any:
		for parseIndex, parseChild := range parseTyped {
			flattenSnapshotDiffNodes(parseChild, fmt.Sprintf("%s/%d", parseFallback, parseIndex), parseOut)
		}
	}
}

func canonicalSnapshotDiffJSON(parseValue any) string {
	parseBytes, parseErr := json.Marshal(parseValue)
	if parseErr != nil {
		return fmt.Sprintf("%#v", parseValue)
	}
	var parseBuffer bytes.Buffer
	if parseErr := json.Compact(&parseBuffer, parseBytes); parseErr != nil {
		return string(parseBytes)
	}
	return parseBuffer.String()
}
