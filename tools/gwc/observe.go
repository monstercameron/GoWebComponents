package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// runObserveCommand runs the observe launcher subcommand.
var runObserveCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runObserve(parseArgs)
}

type observeConfig struct {
	sources  []string
	route    string
	build    string
	severity string
	limit    int
	agent    bool
	json     bool
}

type observeSummary struct {
	OK            bool                      `json:"ok"`
	SourceCount   int                       `json:"sourceCount"`
	Records       []map[string]any          `json:"records"`
	Filters       map[string]string         `json:"filters,omitempty"`
	Diagnostics   []agentDiagnostic         `json:"diagnostics,omitempty"`
	Redaction     observeRedactionReport    `json:"redaction"`
	HydrationDiff agentHydrationTraceRecord `json:"hydrationDiff"`
	CommitTrace   agentCommitTraceRecord    `json:"commitTrace"`
}

type observeRedactionReport struct {
	Applied     bool     `json:"applied"`
	Fields      []string `json:"fields,omitempty"`
	RecordCount int      `json:"recordCount"`
}

// runObserve parses observe flags and emits filtered runtime telemetry.
func (parseL launcher) runObserve(parseArgs []string) error {
	parseFs := flag.NewFlagSet("observe", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	var parseSources stringListFlag
	parseFs.Var(&parseSources, "source", "Path to a structured JSONL/NDJSON telemetry source; repeat or comma-separate")
	parseRoute := parseFs.String("route", "", "Filter records by route")
	parseBuild := parseFs.String("build", "", "Filter records by build id, build sha, or build version")
	parseSeverity := parseFs.String("severity", "", "Minimum severity to include: debug, info, warn, error, or fatal")
	parseLimit := parseFs.Int("limit", 50, "Maximum records to return")
	parseAgent := parseFs.Bool("agent", false, "Emit agent-native NDJSON observe events")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	if *parseAgent && *parseJsonOutput {
		return errors.New("observe -agent cannot be combined with -json; agent mode already emits NDJSON")
	}
	parseConfig := observeConfig{
		sources:  parseSources.Values(),
		route:    *parseRoute,
		build:    *parseBuild,
		severity: *parseSeverity,
		limit:    *parseLimit,
		agent:    *parseAgent,
		json:     *parseJsonOutput,
	}
	parseSummary, parseErr2 := buildObserveSummary(parseConfig)
	if parseConfig.agent {
		parseStream := newAgentEventStream(os.Stdout, "observe")
		parseEvent := agentEvent{
			Event: "observe.summary",
			Phase: "observe",
			OK:    buildAgentBool(parseSummary.OK),
			Data:  parseSummary,
		}
		if len(parseSummary.Diagnostics) > 0 {
			parseEvent.Diagnostics = parseSummary.Diagnostics
			if !parseSummary.OK {
				parseEvent.Error = &agentEventError{Code: parseSummary.Diagnostics[0].Code, Message: parseSummary.Diagnostics[0].Message}
			}
		}
		if parseErr3 := parseStream.emit(parseEvent); parseErr3 != nil {
			return parseErr3
		}
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseErr3 := parseEncoder.Encode(parseSummary); parseErr3 != nil {
			return parseErr3
		}
		return parseErr2
	}
	printObserveSummary(parseSummary)
	return parseErr2
}

// buildObserveSummary loads, filters, and redacts structured telemetry records.
func buildObserveSummary(parseConfig observeConfig) (observeSummary, error) {
	parseSummary := observeSummary{
		OK:      true,
		Records: []map[string]any{},
		Filters: buildObserveFilters(parseConfig),
		Redaction: observeRedactionReport{
			Applied: true,
		},
	}
	parseLimit := parseConfig.limit
	if parseLimit <= 0 {
		parseLimit = 50
	}
	parseMinimumSeverity, parseSeverityKnown := parseObserveSeverityRank(parseConfig.severity)
	if strings.TrimSpace(parseConfig.severity) != "" && !parseSeverityKnown {
		parseDiagnostic := agentDiagnostic{
			Code:     "GWC_AGENT_OBSERVE_SEVERITY",
			Message:  fmt.Sprintf("unknown observe severity %q", parseConfig.severity),
			Severity: "error",
			Hint:     "Use debug, info, warn, error, or fatal.",
		}
		parseSummary.OK = false
		parseSummary.Diagnostics = append(parseSummary.Diagnostics, parseDiagnostic)
		return parseSummary, errors.New(parseDiagnostic.Message)
	}
	for _, parseSource := range parseConfig.sources {
		parseRecords, parseReport, parseErr := readObserveSource(parseSource, parseConfig, parseMinimumSeverity, parseSeverityKnown, parseLimit-len(parseSummary.Records))
		parseSummary.SourceCount++
		parseSummary.Redaction.Fields = append(parseSummary.Redaction.Fields, parseReport.Fields...)
		parseSummary.Redaction.RecordCount += parseReport.RecordCount
		parseSummary.Records = append(parseSummary.Records, parseRecords...)
		if parseErr != nil {
			parseDiagnostic := buildAgentDiagnosticFromError("GWC_AGENT_OBSERVE_SOURCE", parseErr.Error(), "error")
			parseSummary.OK = false
			parseSummary.Diagnostics = append(parseSummary.Diagnostics, parseDiagnostic)
			return parseSummary, parseErr
		}
		if len(parseSummary.Records) >= parseLimit {
			break
		}
	}
	parseSummary.Redaction.Fields = dedupeObserveStrings(parseSummary.Redaction.Fields)
	if len(parseSummary.Records) == 0 {
		parseSummary.Records = []map[string]any{}
	}
	parseSummary.HydrationDiff, parseSummary.CommitTrace = buildAgentTraceRepresentations(parseSummary.Records...)
	return parseSummary, nil
}

// buildObserveFilters returns the non-empty query filters for the summary.
func buildObserveFilters(parseConfig observeConfig) map[string]string {
	parseFilters := map[string]string{}
	if strings.TrimSpace(parseConfig.route) != "" {
		parseFilters["route"] = strings.TrimSpace(parseConfig.route)
	}
	if strings.TrimSpace(parseConfig.build) != "" {
		parseFilters["build"] = strings.TrimSpace(parseConfig.build)
	}
	if strings.TrimSpace(parseConfig.severity) != "" {
		parseFilters["severity"] = strings.TrimSpace(parseConfig.severity)
	}
	if len(parseFilters) == 0 {
		return nil
	}
	return parseFilters
}

// readObserveSource reads one JSONL source and returns matching records.
func readObserveSource(parseSource string, parseConfig observeConfig, parseMinimumSeverity int, hasSeverityFilter bool, parseLimit int) ([]map[string]any, observeRedactionReport, error) {
	parseFile, parseErr := os.Open(strings.TrimSpace(parseSource))
	if parseErr != nil {
		return nil, observeRedactionReport{}, parseErr
	}
	defer parseFile.Close()
	return readObserveRecords(parseFile, parseConfig, parseMinimumSeverity, hasSeverityFilter, parseLimit)
}

// readObserveRecords reads structured JSONL records from a stream.
func readObserveRecords(parseReader io.Reader, parseConfig observeConfig, parseMinimumSeverity int, hasSeverityFilter bool, parseLimit int) ([]map[string]any, observeRedactionReport, error) {
	parseRecords := []map[string]any{}
	parseReport := observeRedactionReport{Applied: true}
	if parseLimit <= 0 {
		return parseRecords, parseReport, nil
	}
	parseScanner := bufio.NewScanner(parseReader)
	parseScanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for parseScanner.Scan() {
		parseLine := strings.TrimSpace(parseScanner.Text())
		if parseLine == "" {
			continue
		}
		var parseRecord map[string]any
		if parseErr := json.Unmarshal([]byte(parseLine), &parseRecord); parseErr != nil {
			return parseRecords, parseReport, fmt.Errorf("parse observe record: %w", parseErr)
		}
		if !matchesObserveRecord(parseRecord, parseConfig, parseMinimumSeverity, hasSeverityFilter) {
			continue
		}
		parseRedacted, parseFields := redactObserveValue(parseRecord)
		parseRedactedRecord, parseOk := parseRedacted.(map[string]any)
		if !parseOk {
			continue
		}
		parseRecords = append(parseRecords, parseRedactedRecord)
		parseReport.Fields = append(parseReport.Fields, parseFields...)
		parseReport.RecordCount++
		if len(parseRecords) >= parseLimit {
			break
		}
	}
	if parseErr := parseScanner.Err(); parseErr != nil {
		return parseRecords, parseReport, parseErr
	}
	return parseRecords, parseReport, nil
}

// matchesObserveRecord reports whether a telemetry record matches configured filters.
func matchesObserveRecord(parseRecord map[string]any, parseConfig observeConfig, parseMinimumSeverity int, hasSeverityFilter bool) bool {
	if strings.TrimSpace(parseConfig.route) != "" && !strings.EqualFold(extractObserveField(parseRecord, "route", "http.route", "path"), strings.TrimSpace(parseConfig.route)) {
		return false
	}
	if strings.TrimSpace(parseConfig.build) != "" && !matchesObserveBuild(parseRecord, strings.TrimSpace(parseConfig.build)) {
		return false
	}
	if hasSeverityFilter {
		parseRank, parseOk := parseObserveSeverityRank(extractObserveField(parseRecord, "severity_text", "severity", "level"))
		if !parseOk || parseRank < parseMinimumSeverity {
			return false
		}
	}
	return true
}

// matchesObserveBuild reports whether a record matches one build identity.
func matchesObserveBuild(parseRecord map[string]any, parseBuild string) bool {
	for _, parseKey := range []string{"build", "build_id", "buildId", "build_sha", "buildSha", "version"} {
		if strings.EqualFold(extractObserveField(parseRecord, parseKey), parseBuild) {
			return true
		}
	}
	return false
}

// extractObserveField returns a top-level or attributes nested field as a string.
func extractObserveField(parseRecord map[string]any, parseKeys ...string) string {
	parseAttributes, _ := parseRecord["attributes"].(map[string]any)
	for _, parseKey := range parseKeys {
		if parseValue, parseOk := parseRecord[parseKey]; parseOk {
			return strings.TrimSpace(fmt.Sprint(parseValue))
		}
		if parseAttributes != nil {
			parseValue, parseOk := parseAttributes[parseKey]
			if !parseOk {
				continue
			}
			return strings.TrimSpace(fmt.Sprint(parseValue))
		}
	}
	return ""
}

// parseObserveSeverityRank maps severity names to ordered ranks.
func parseObserveSeverityRank(parseSeverity string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(parseSeverity)) {
	case "":
		return 0, true
	case "trace", "debug":
		return 10, true
	case "info", "information":
		return 20, true
	case "warn", "warning":
		return 30, true
	case "error":
		return 40, true
	case "fatal", "panic":
		return 50, true
	default:
		return 0, false
	}
}

// redactObserveValue recursively redacts common PII and secret fields.
func redactObserveValue(parseValue any) (any, []string) {
	switch parseTyped := parseValue.(type) {
	case map[string]any:
		parseRedacted := make(map[string]any, len(parseTyped))
		parseFields := []string{}
		for parseKey, parseChild := range parseTyped {
			if shouldRedactObserveKey(parseKey) {
				parseRedacted[parseKey] = "[REDACTED]"
				parseFields = append(parseFields, parseKey)
				continue
			}
			parseChildRedacted, parseChildFields := redactObserveValue(parseChild)
			parseRedacted[parseKey] = parseChildRedacted
			parseFields = append(parseFields, parseChildFields...)
		}
		return parseRedacted, parseFields
	case []any:
		parseRedacted := make([]any, 0, len(parseTyped))
		parseFields := []string{}
		for _, parseChild := range parseTyped {
			parseChildRedacted, parseChildFields := redactObserveValue(parseChild)
			parseRedacted = append(parseRedacted, parseChildRedacted)
			parseFields = append(parseFields, parseChildFields...)
		}
		return parseRedacted, parseFields
	default:
		return parseValue, nil
	}
}

// shouldRedactObserveKey reports whether a field name should be redacted.
func shouldRedactObserveKey(parseKey string) bool {
	parseLower := strings.ToLower(strings.TrimSpace(parseKey))
	for _, parseNeedle := range []string{"authorization", "cookie", "email", "password", "secret", "token"} {
		if strings.Contains(parseLower, parseNeedle) {
			return true
		}
	}
	return false
}

// dedupeObserveStrings returns sorted unique strings.
func dedupeObserveStrings(parseValues []string) []string {
	parseSeen := map[string]struct{}{}
	parseOutput := []string{}
	for _, parseValue := range parseValues {
		parseTrimmed := strings.TrimSpace(parseValue)
		if parseTrimmed == "" {
			continue
		}
		if _, parseOk := parseSeen[parseTrimmed]; parseOk {
			continue
		}
		parseSeen[parseTrimmed] = struct{}{}
		parseOutput = append(parseOutput, parseTrimmed)
	}
	sort.Strings(parseOutput)
	return parseOutput
}

// printObserveSummary writes a compact human-readable observe report.
func printObserveSummary(parseSummary observeSummary) {
	fmt.Println("GWC observe")
	fmt.Printf("  sources:      %d\n", parseSummary.SourceCount)
	fmt.Printf("  records:      %d\n", len(parseSummary.Records))
	if len(parseSummary.Filters) > 0 {
		parseKeys := make([]string, 0, len(parseSummary.Filters))
		for parseKey := range parseSummary.Filters {
			parseKeys = append(parseKeys, parseKey)
		}
		sort.Strings(parseKeys)
		for _, parseKey := range parseKeys {
			fmt.Printf("  filter[%s]: %s\n", parseKey, parseSummary.Filters[parseKey])
		}
	}
	if len(parseSummary.Diagnostics) > 0 {
		for _, parseDiagnostic := range parseSummary.Diagnostics {
			fmt.Printf("  diagnostic[%s]: %s\n", parseDiagnostic.Code, parseDiagnostic.Message)
		}
	}
	fmt.Printf("  hydration:   %s (%d mismatches)\n", parseSummary.HydrationDiff.Status, parseSummary.HydrationDiff.MismatchCount)
	fmt.Printf("  commit trace:%s (%d events)\n", parseSummary.CommitTrace.Status, len(parseSummary.CommitTrace.Events))
}
