package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type doctorConfig struct {
	host               string
	port               string
	audit              bool
	auditPolicy        string
	auditBaselinePath  string
	auditWriteBaseline string
	auditSuppressions  []string
	json               bool
}

type doctorCheck struct {
	RuleID      string   `json:"ruleId,omitempty"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Severity    string   `json:"severity,omitempty"`
	Summary     string   `json:"summary"`
	Locations   []string `json:"locations,omitempty"`
	Hint        string   `json:"hint,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

type doctorReport struct {
	OK         bool               `json:"ok"`
	Checked    string             `json:"checked"`
	CWD        string             `json:"cwd"`
	Checks     []doctorCheck      `json:"checks"`
	Audit      *doctorAuditReport `json:"audit,omitempty"`
	Resolution map[string]string  `json:"resolution,omitempty"`
}

type doctorAuditReport struct {
	Mode         string        `json:"mode"`
	Policy       string        `json:"policy"`
	OK           bool          `json:"ok"`
	BaselinePath string        `json:"baselinePath,omitempty"`
	Suppressed   []string      `json:"suppressed,omitempty"`
	Checks       []doctorCheck `json:"checks"`
}

type doctorAuditBaseline struct {
	Mode        string                     `json:"mode"`
	GeneratedAt string                     `json:"generatedAt"`
	Checks      []doctorAuditBaselineCheck `json:"checks,omitempty"`
}

type doctorAuditBaselineCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

var doctorLookPath = exec.LookPath

var doctorAuditLocationPattern = regexp.MustCompile(`([A-Za-z0-9_./-]+\.(?:go|html|json|wasm))`)

var doctorCommandOutput = func(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

var doctorGetwd = os.Getwd

var doctorListen = net.Listen

func (parseL launcher) runDoctor(parseArgs []string) error {
	parseFs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseHost := parseFs.String("host", defaultHost, "Host to probe for port availability")
	parsePort := parseFs.String("port", "8080", "Port to probe for local development availability")
	parseAudit := parseFs.Bool("audit", false, "Run the golden-path app audit in addition to prerequisite checks")
	parseAuditPolicy := parseFs.String("audit-policy", "strict", "Golden-path audit policy: strict or advisory")
	parseAuditBaseline := parseFs.String("audit-baseline", "", "Optional path to a JSON baseline file of accepted audit findings")
	parseAuditWriteBaseline := parseFs.String("audit-write-baseline", "", "Optional path to write the current audit findings as a JSON baseline")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	var parseAuditSuppressions stringListFlag
	parseFs.Var(&parseAuditSuppressions, "audit-suppress", "Audit check name to suppress; repeat or comma-separate")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseReport := parseL.buildDoctorReport(doctorConfig{
		host:               *parseHost,
		port:               *parsePort,
		audit:              *parseAudit,
		auditPolicy:        *parseAuditPolicy,
		auditBaselinePath:  *parseAuditBaseline,
		auditWriteBaseline: *parseAuditWriteBaseline,
		auditSuppressions:  parseAuditSuppressions.Values(),
		json:               *parseJsonOutput,
	})
	if *parseAudit && strings.TrimSpace(*parseAuditWriteBaseline) != "" && parseReport.Audit != nil {
		if parseErr2 := writeDoctorAuditBaseline(*parseAuditWriteBaseline, *parseReport.Audit); parseErr2 != nil {
			return parseErr2
		}
	}
	if *parseJsonOutput {
		parseEncoder := json.NewEncoder(os.Stdout)
		if parseErr3 := parseEncoder.Encode(parseReport); parseErr3 != nil {
			return parseErr3
		}
	} else {
		printDoctorReport(parseReport)
	}
	if !parseReport.OK {
		return errors.New("doctor found required checks that need attention")
	}
	return nil
}

func (parseL launcher) buildDoctorReport(parseConfig doctorConfig) doctorReport {
	parseCwd, parseErr := doctorGetwd()
	if parseErr != nil {
		parseCwd = ""
	}
	parseReport := doctorReport{
		OK:         true,
		Checked:    time.Now().UTC().Format(time.RFC3339),
		CWD:        parseCwd,
		Resolution: buildDoctorResolutionTrace(parseCwd, parseConfig),
	}
	parseAppendCheck := func(parseCheck doctorCheck) {
		parseReport.Checks = append(parseReport.Checks, parseCheck)
		if parseCheck.Status == "fail" {
			parseReport.OK = false
		}
	}

	parseAppendCheck(buildDoctorToolCheck("go", "Go toolchain", "version", "Install Go 1.25 or newer and ensure `go` is on PATH."))
	parseAppendCheck(buildDoctorWasmExecCheck())
	parseAppendCheck(buildDoctorPlaywrightCheck(parseL.repoRoot))
	parseAppendCheck(buildDoctorMetadataCheck(parseCwd))
	parseAppendCheck(buildDoctorProjectDetectionCheck(parseCwd))
	parseAppendCheck(buildDoctorPortCheck(parseConfig.host, parseConfig.port))
	if parseConfig.audit {
		parseReport.Audit = buildDoctorAuditReport(parseCwd, parseConfig)
		if parseReport.Audit.Policy == "strict" && !parseReport.Audit.OK {
			parseReport.OK = false
		}
	}

	return parseReport
}

func buildDoctorResolutionTrace(parseCwd string, parseConfig doctorConfig) map[string]string {
	parseTrace := map[string]string{}
	if strings.TrimSpace(parseCwd) != "" {
		parseTrace["root"] = "working directory"
	}
	if strings.TrimSpace(parseConfig.host) != "" && parseConfig.host != defaultHost {
		parseTrace["host"] = "explicit flag"
	} else {
		parseTrace["host"] = "convention fallback"
	}
	if strings.TrimSpace(parseConfig.port) != "" && parseConfig.port != "8080" {
		parseTrace["port"] = "explicit flag"
	} else {
		parseTrace["port"] = "convention fallback"
	}
	parseMetadata, parseMetadataDir, hasMetadata, parseErr := resolveScaffoldMetadataForConfig(parseCwd, "", "")
	if parseErr == nil && hasMetadata {
		if strings.TrimSpace(parseMetadata.Tooling.AppPath) != "" {
			parseTrace["app"] = "gwc-start.json"
		}
		if strings.TrimSpace(parseMetadata.Tooling.HTMLPath) != "" {
			parseTrace["html"] = "gwc-start.json"
		}
		if strings.TrimSpace(parseMetadataDir) != "" {
			parseTrace["root"] = "gwc-start.json"
		}
	}
	if _, parseOk := parseTrace["app"]; !parseOk && strings.TrimSpace(parseCwd) != "" {
		if _, parseErr2 := detectAppPath(parseCwd); parseErr2 == nil {
			parseTrace["app"] = "convention fallback"
		}
	}
	if _, parseOk2 := parseTrace["html"]; !parseOk2 && strings.TrimSpace(parseCwd) != "" {
		if strings.TrimSpace(detectHTMLPath(parseCwd)) != "" {
			parseTrace["html"] = "convention fallback"
		}
	}
	return parseTrace
}

func buildDoctorAuditReport(parseCwd string, parseConfig doctorConfig) *doctorAuditReport {
	parseAudit := buildDoctorGoldenPathAudit(parseCwd)
	applyDoctorAuditAdoption(&parseAudit, parseCwd, parseConfig)
	if strings.TrimSpace(parseAudit.Policy) == "" {
		parseAudit.Policy = "strict"
	}
	return &parseAudit
}

func applyDoctorAuditAdoption(parseAudit *doctorAuditReport, parseCwd string, parseConfig doctorConfig) {
	if parseAudit == nil {
		return
	}
	parsePolicy, parseOk := normalizeDoctorAuditPolicy(parseConfig.auditPolicy)
	if !parseOk {
		parseAudit.Policy = "strict"
		parseAudit.OK = false
		parseAudit.Checks = append([]doctorCheck{{
			Name:    "Audit policy",
			Status:  "fail",
			Summary: fmt.Sprintf("Unknown audit policy %q.", parseConfig.auditPolicy),
			Hint:    "Use -audit-policy strict or -audit-policy advisory.",
		}}, parseAudit.Checks...)
		annotateDoctorAuditMetadata(parseAudit)
		return
	}
	parseAudit.Policy = parsePolicy
	parseSuppressed := map[string]struct{}{}
	for _, parseName := range parseConfig.auditSuppressions {
		parseTrimmed := strings.TrimSpace(parseName)
		if parseTrimmed != "" {
			parseSuppressed[parseTrimmed] = struct{}{}
		}
	}
	if strings.TrimSpace(parseConfig.auditBaselinePath) != "" {
		parseBaselinePath, parseErr := normalizePath(parseCwd, parseConfig.auditBaselinePath)
		if parseErr != nil {
			parseAudit.OK = false
			parseAudit.Checks = append([]doctorCheck{{
				Name:    "Audit baseline",
				Status:  "fail",
				Summary: fmt.Sprintf("Could not resolve audit baseline path: %v", parseErr),
				Hint:    "Pass a valid file path to -audit-baseline or remove the flag.",
			}}, parseAudit.Checks...)
			annotateDoctorAuditMetadata(parseAudit)
			return
		}
		parseBaseline, parseErr := loadDoctorAuditBaseline(parseBaselinePath)
		if parseErr != nil {
			parseAudit.OK = false
			parseAudit.Checks = append([]doctorCheck{{
				Name:    "Audit baseline",
				Status:  "fail",
				Summary: parseErr.Error(),
				Hint:    "Write a fresh baseline with -audit-write-baseline or fix the checked-in baseline file.",
			}}, parseAudit.Checks...)
			annotateDoctorAuditMetadata(parseAudit)
			return
		}
		parseAudit.BaselinePath = parseBaselinePath
		for _, parseEntry := range parseBaseline.Checks {
			parseTrimmed2 := strings.TrimSpace(parseEntry.Name)
			if parseTrimmed2 != "" {
				parseSuppressed[parseTrimmed2] = struct{}{}
			}
		}
	}
	if len(parseSuppressed) == 0 {
		parseAudit.OK = doctorAuditChecksPassing(parseAudit.Checks)
		annotateDoctorAuditMetadata(parseAudit)
		return
	}
	parseNames := make([]string, 0, len(parseSuppressed))
	for parseName2 := range parseSuppressed {
		parseNames = append(parseNames, parseName2)
	}
	sort.Strings(parseNames)
	parseAudit.Suppressed = parseNames
	for parseI := range parseAudit.Checks {
		parseCheck := &parseAudit.Checks[parseI]
		if parseCheck.Status == "pass" || parseCheck.Status == "suppressed" {
			continue
		}
		if _, parseOk2 := parseSuppressed[parseCheck.Name]; parseOk2 {
			parseCheck.Status = "suppressed"
			parseCheck.Summary = parseCheck.Summary + " (suppressed)"
		}
	}
	parseAudit.OK = doctorAuditChecksPassing(parseAudit.Checks)
	annotateDoctorAuditMetadata(parseAudit)
}

func normalizeDoctorAuditPolicy(parseValue string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(parseValue)) {
	case "", "strict":
		return "strict", true
	case "advisory", "warn":
		return "advisory", true
	default:
		return "", false
	}
}

func doctorAuditChecksPassing(parseChecks []doctorCheck) bool {
	for _, parseCheck := range parseChecks {
		if parseCheck.Status == "fail" {
			return false
		}
	}
	return true
}

func annotateDoctorAuditMetadata(parseAudit *doctorAuditReport) {
	if parseAudit == nil {
		return
	}
	for parseI := range parseAudit.Checks {
		parseCheck := &parseAudit.Checks[parseI]
		if parseCheck.RuleID == "" {
			parseCheck.RuleID = doctorAuditRuleIDForName(parseCheck.Name)
		}
		parseCheck.Severity = doctorAuditSeverityForStatus(parseCheck.Status)
		if len(parseCheck.Locations) == 0 {
			parseCheck.Locations = extractDoctorAuditLocations(parseCheck.Summary)
		}
		if strings.TrimSpace(parseCheck.Remediation) == "" && strings.TrimSpace(parseCheck.Hint) != "" {
			parseCheck.Remediation = parseCheck.Hint
		}
	}
}

func appendDoctorLocation(parseLocations []string, parseValue string) []string {
	parseTrimmed := strings.TrimSpace(filepath.ToSlash(parseValue))
	if parseTrimmed == "" {
		return parseLocations
	}
	for _, parseExisting := range parseLocations {
		if parseExisting == parseTrimmed {
			return parseLocations
		}
	}
	return append(parseLocations, parseTrimmed)
}

func doctorAuditLocation(parseRoot string, parsePath string) string {
	parseTrimmed := strings.TrimSpace(parsePath)
	if parseTrimmed == "" {
		return ""
	}
	if strings.TrimSpace(parseRoot) != "" {
		if parseRel, parseErr := filepath.Rel(parseRoot, parseTrimmed); parseErr == nil && parseRel != "." && !strings.HasPrefix(parseRel, "..") {
			return filepath.ToSlash(parseRel)
		}
	}
	return filepath.ToSlash(parseTrimmed)
}

func doctorAuditRuleIDForName(parseName string) string {
	switch parseName {
	case "Audit policy":
		return "audit.policy"
	case "Audit baseline":
		return "audit.baseline"
	case "Audit target":
		return "audit.target"
	case "App entrypoint":
		return "audit.app_entrypoint"
	case "HTML shell":
		return "audit.html_shell"
	case "Starter metadata anchor":
		return "audit.metadata_anchor"
	case "State and ownership boundaries":
		return "audit.state_boundaries"
	case "Local versus shared state ownership":
		return "audit.state_ownership"
	case "Route shape and delivery":
		return "audit.route_delivery"
	case "Mutation and resilience":
		return "audit.mutation_resilience"
	case "Startup cost and ownership evidence":
		return "audit.startup_evidence"
	case "Runtime evidence":
		return "audit.runtime_evidence"
	default:
		return ""
	}
}

func doctorAuditSeverityForStatus(parseStatus string) string {
	switch strings.TrimSpace(strings.ToLower(parseStatus)) {
	case "fail":
		return "error"
	case "warn":
		return "warning"
	case "suppressed":
		return "suppressed"
	case "pass":
		return "info"
	default:
		return ""
	}
}

func normalizeDoctorAuditMinimumSeverity(parseValue string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(parseValue)) {
	case "", "error":
		return "error", true
	case "warning", "warn":
		return "warning", true
	case "info":
		return "info", true
	case "off", "none":
		return "off", true
	default:
		return "", false
	}
}

func doctorAuditSeverityRank(parseValue string) int {
	switch strings.TrimSpace(strings.ToLower(parseValue)) {
	case "info":
		return 1
	case "warning", "warn":
		return 2
	case "error":
		return 3
	default:
		return 0
	}
}

func doctorAuditHasFindingAtOrAbove(parseChecks []doctorCheck, parseMinimum string) bool {
	parseNormalized, parseOk := normalizeDoctorAuditMinimumSeverity(parseMinimum)
	if !parseOk || parseNormalized == "off" {
		return false
	}
	parseRequiredRank := doctorAuditSeverityRank(parseNormalized)
	for _, parseCheck := range parseChecks {
		if parseCheck.Status == "pass" || parseCheck.Status == "suppressed" {
			continue
		}
		if doctorAuditSeverityRank(parseCheck.Severity) >= parseRequiredRank {
			return true
		}
	}
	return false
}

func extractDoctorAuditLocations(parseSummary string) []string {
	parseMatches := doctorAuditLocationPattern.FindAllString(parseSummary, -1)
	if len(parseMatches) == 0 {
		return nil
	}
	parseSeen := map[string]struct{}{}
	parseLocations := make([]string, 0, len(parseMatches))
	for _, parseMatch := range parseMatches {
		if _, parseOk := parseSeen[parseMatch]; parseOk {
			continue
		}
		parseSeen[parseMatch] = struct{}{}
		parseLocations = append(parseLocations, parseMatch)
	}
	return parseLocations
}

func buildDoctorGoldenPathAudit(parseCwd string) doctorAuditReport {
	parseReport := doctorAuditReport{
		Mode: "golden-path",
		OK:   true,
	}
	parseAppendCheck := func(parseCheck doctorCheck) {
		parseReport.Checks = append(parseReport.Checks, parseCheck)
		if parseCheck.Status == "fail" {
			parseReport.OK = false
		}
	}
	if strings.TrimSpace(parseCwd) == "" {
		parseAppendCheck(doctorCheck{
			RuleID:  "audit.target",
			Name:    "Audit target",
			Status:  "fail",
			Summary: "The current working directory could not be resolved for golden-path auditing.",
			Hint:    "Run `gwc doctor -audit` from the target app root.",
		})
		annotateDoctorAuditMetadata(&parseReport)
		return parseReport
	}
	parseAppPath, parseAppErr := detectAppPath(parseCwd)
	if parseAppErr != nil {
		parseAppendCheck(doctorCheck{
			RuleID:  "audit.app_entrypoint",
			Name:    "App entrypoint",
			Status:  "fail",
			Summary: "No launcher-detectable app entrypoint was found for golden-path auditing.",
			Hint:    "Keep main.go or cmd/web/main.go at the documented locations, or add scaffold metadata that pins the app path.",
		})
	} else {
		parseAppendCheck(doctorCheck{
			RuleID:    "audit.app_entrypoint",
			Name:      "App entrypoint",
			Status:    "pass",
			Summary:   fmt.Sprintf("Auditing app entrypoint %s", parseAppPath),
			Locations: []string{doctorAuditLocation(parseCwd, parseAppPath)},
		})
	}
	parseHtmlPath := detectHTMLPath(parseCwd)
	if strings.TrimSpace(parseHtmlPath) == "" {
		parseAppendCheck(doctorCheck{
			RuleID:  "audit.html_shell",
			Name:    "HTML shell",
			Status:  "warn",
			Summary: "No HTML shell was auto-detected for the current app root.",
			Hint:    "Keep index.html or a documented equivalent near the app so later delivery audits can reason about the served shell.",
		})
	} else {
		parseAppendCheck(doctorCheck{
			RuleID:    "audit.html_shell",
			Name:      "HTML shell",
			Status:    "pass",
			Summary:   fmt.Sprintf("Detected HTML shell %s", parseHtmlPath),
			Locations: []string{doctorAuditLocation(parseCwd, parseHtmlPath)},
		})
	}
	parseMetadata, parseOk, parseErr := loadScaffoldMetadata(parseCwd)
	if parseErr != nil {
		parseAppendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "fail",
			Summary:   parseErr.Error(),
			Locations: []string{"gwc-start.json"},
			Hint:      "Fix or regenerate gwc-start.json so golden-path audits can resolve intended launcher ownership.",
		})
	} else if !parseOk {
		parseAppendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "warn",
			Summary:   "No gwc-start.json metadata anchor was found for this app.",
			Locations: []string{"gwc-start.json"},
			Hint:      "Generated starters should keep scaffold metadata so future audit rules can trace intended ownership and output paths.",
		})
	} else {
		parseAppendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "pass",
			Summary:   fmt.Sprintf("Using starter metadata for %s", firstNonEmpty(parseMetadata.ProjectName, "<unnamed>")),
			Locations: []string{"gwc-start.json"},
		})
	}
	parseAppendCheck(buildDoctorOwnershipBoundaryCheck(parseCwd))
	parseAppendCheck(buildDoctorStateOwnershipCheck(parseCwd))
	parseAppendCheck(buildDoctorRouteDeliveryCheck(parseCwd))
	parseAppendCheck(buildDoctorMutationResilienceCheck(parseCwd))
	parseAppendCheck(buildDoctorStartupEvidenceCheck(parseCwd))
	parseAppendCheck(buildDoctorRuntimeEvidenceCheck(parseCwd))
	annotateDoctorAuditMetadata(&parseReport)
	return parseReport
}

func buildDoctorOwnershipBoundaryCheck(parseCwd string) doctorCheck {
	parseFiles, parseErr := collectGoldenPathGoFiles(parseCwd)
	if parseErr != nil {
		return doctorCheck{
			RuleID:  "audit.state_boundaries",
			Name:    "State and ownership boundaries",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for ownership-boundary auditing: %v", parseErr),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	parseViolations := []string{}
	parseLocations := []string{}
	for _, parseFile := range parseFiles {
		for _, parseImportPath := range parseFile.Imports {
			switch {
			case parseFile.Client && strings.Contains(parseImportPath, "/server/"):
				parseViolations = append(parseViolations, fmt.Sprintf("%s imports server package %s", parseFile.RelPath, parseImportPath))
				parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
			case parseFile.Client && (parseImportPath == "database/sql" || parseImportPath == "os/exec"):
				parseViolations = append(parseViolations, fmt.Sprintf("%s imports server-only package %s", parseFile.RelPath, parseImportPath))
				parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
			case !parseFile.Client && parseImportPath == "syscall/js":
				parseViolations = append(parseViolations, fmt.Sprintf("%s imports browser-only package %s outside a js/wasm boundary", parseFile.RelPath, parseImportPath))
				parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
			}
		}
	}
	if len(parseViolations) == 0 {
		return doctorCheck{
			RuleID:  "audit.state_boundaries",
			Name:    "State and ownership boundaries",
			Status:  "pass",
			Summary: "No static client/server ownership boundary leaks were detected in Go imports.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.state_boundaries",
		Name:      "State and ownership boundaries",
		Status:    "fail",
		Summary:   summarizeDoctorViolations(parseViolations, 3),
		Locations: parseLocations,
		Hint:      "Keep client files on browser-safe imports, keep server packages out of js/wasm paths, and keep syscall/js usage behind explicit js/wasm build tags.",
	}
}

func buildDoctorStateOwnershipCheck(parseCwd string) doctorCheck {
	parseFiles, parseErr := collectGoldenPathGoFiles(parseCwd)
	if parseErr != nil {
		return doctorCheck{
			RuleID:  "audit.state_ownership",
			Name:    "Local versus shared state ownership",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for mixed state ownership heuristics: %v", parseErr),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	parseViolations := []string{}
	parseLocations := []string{}
	for _, parseFile := range parseFiles {
		if !parseFile.Client {
			continue
		}
		if strings.Contains(parseFile.Content, "fetch.UseCachedResource") &&
			(strings.Contains(parseFile.Content, "localStorage") || strings.Contains(parseFile.Content, "sessionStorage") || strings.Contains(parseFile.Content, "indexedDB")) {
			parseViolations = append(parseViolations, fmt.Sprintf("%s mixes fetch.UseCachedResource with direct browser storage access", parseFile.RelPath))
			parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
		}
	}
	if len(parseViolations) == 0 {
		return doctorCheck{
			RuleID:  "audit.state_ownership",
			Name:    "Local versus shared state ownership",
			Status:  "pass",
			Summary: "No mixed local-versus-shared state ownership heuristics were detected in client files.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.state_ownership",
		Name:      "Local versus shared state ownership",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(parseViolations, 3),
		Locations: parseLocations,
		Hint:      "Prefer one obvious owner per state slice: either fetch/cache-backed shared data or browser-local persistence, not both in the same controller without an explicit boundary.",
	}
}

func buildDoctorRouteDeliveryCheck(parseCwd string) doctorCheck {
	parseGoFiles, parseErr := collectGoldenPathGoFiles(parseCwd)
	if parseErr != nil {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for route-shape auditing: %v", parseErr),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	parseHtmlFiles, parseErr := collectDoctorHTMLFiles(parseCwd)
	if parseErr != nil {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan HTML shells for route-shape auditing: %v", parseErr),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	parseWarnings := []string{}
	parseLocations := []string{}
	if len(parseHtmlFiles) > 1 {
		parseWarnings = append(parseWarnings, fmt.Sprintf("multiple HTML entry shells detected (%s)", strings.Join(parseHtmlFiles, ", ")))
		for _, parsePath := range parseHtmlFiles {
			parseLocations = appendDoctorLocation(parseLocations, parsePath)
		}
	}
	parseRouteCount := 0
	hasLazySplit := false
	hasPrerenderSignal := false
	isParseMarketingRoutes := false
	for _, parseFile := range parseGoFiles {
		parseRouteHits := strings.Count(parseFile.Content, "MustDefineRoute(")
		parseRouteHits += strings.Count(parseFile.Content, "router.Register(")
		if parseRouteHits > 0 {
			parseRouteCount += parseRouteHits
			parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
		}
		if strings.Contains(parseFile.Content, "ui.Lazy(") || strings.Contains(parseFile.Content, "ui.CreateElement(ui.Lazy") {
			hasLazySplit = true
		}
		parseLowered := strings.ToLower(parseFile.Content)
		if strings.Contains(parseLowered, "prerender") || strings.Contains(parseLowered, "static shell") {
			hasPrerenderSignal = true
		}
		if strings.Contains(parseFile.Content, `"/pricing"`) || strings.Contains(parseFile.Content, `"/capabilities"`) || strings.Contains(parseFile.Content, `"/about"`) || strings.Contains(parseFile.Content, `"/docs"`) {
			isParseMarketingRoutes = true
			parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
		}
	}
	if parseRouteCount >= 3 && !hasLazySplit {
		parseWarnings = append(parseWarnings, fmt.Sprintf("route tree defines %d route registration points with no ui.Lazy split signal", parseRouteCount))
	}
	if isParseMarketingRoutes && !hasPrerenderSignal {
		parseWarnings = append(parseWarnings, "marketing-style routes were detected with no static/prerender delivery hint")
	}
	if len(parseWarnings) == 0 {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "pass",
			Summary: "No first-pass route-shape or delivery warnings were detected.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.route_delivery",
		Name:      "Route shape and delivery",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(parseWarnings, 3),
		Locations: parseLocations,
		Hint:      "Prefer one app shell, prerender or keep static-friendly marketing routes cheap, and use ui.Lazy or equivalent delivery splits when route trees start carrying distinct feature surfaces.",
	}
}

func buildDoctorMutationResilienceCheck(parseCwd string) doctorCheck {
	parseFiles, parseErr := collectGoldenPathGoFiles(parseCwd)
	if parseErr != nil {
		return doctorCheck{
			RuleID:  "audit.mutation_resilience",
			Name:    "Mutation and resilience",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for mutation resilience heuristics: %v", parseErr),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	parseMutationIndicators := []string{"http.methodpost", "http.methodput", "http.methodpatch", "http.methoddelete", ".exec(", "deleteconversation", "upsert", "setselected", "setcustom", "signup(", "login(", "save", "submit"}
	parseResilienceIndicators := []string{"retry", "backoff", "idempot", "rollback", "conflict", "offline", "replay", "timeout", "deadline"}
	parseWarnings := []string{}
	parseLocations := []string{}
	for _, parseFile := range parseFiles {
		parseLowered := strings.ToLower(parseFile.Content)
		hasMutation := false
		for _, parseIndicator := range parseMutationIndicators {
			if strings.Contains(parseLowered, parseIndicator) {
				hasMutation = true
				break
			}
		}
		if !hasMutation {
			continue
		}
		hasResilienceSignal := false
		for _, parseIndicator2 := range parseResilienceIndicators {
			if strings.Contains(parseLowered, parseIndicator2) {
				hasResilienceSignal = true
				break
			}
		}
		if !hasResilienceSignal {
			parseWarnings = append(parseWarnings, fmt.Sprintf("%s exposes mutation-shaped code with no retry/idempotency/conflict/offline signal", parseFile.RelPath))
			parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
		}
	}
	if len(parseWarnings) == 0 {
		return doctorCheck{
			RuleID:  "audit.mutation_resilience",
			Name:    "Mutation and resilience",
			Status:  "pass",
			Summary: "No first-pass mutation resilience warnings were detected.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.mutation_resilience",
		Name:      "Mutation and resilience",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(parseWarnings, 3),
		Locations: parseLocations,
		Hint:      "Document or encode retry posture, idempotency boundaries, rollback/conflict handling, or offline replay semantics around important writes instead of shipping only the happy path.",
	}
}

func buildDoctorStartupEvidenceCheck(parseCwd string) doctorCheck {
	parseFiles, parseErr := collectGoldenPathGoFiles(parseCwd)
	if parseErr != nil {
		return doctorCheck{
			RuleID:  "audit.startup_evidence",
			Name:    "Startup cost and ownership evidence",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for startup-cost evidence: %v", parseErr),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	parseWarnings := []string{}
	parseLocations := []string{}
	for _, parseFile := range parseFiles {
		if !parseFile.Client {
			continue
		}
		if parseFile.SizeBytes > 64*1024 {
			parseWarnings = append(parseWarnings, fmt.Sprintf("%s is %s of client-owned source in the initial app path", parseFile.RelPath, formatBytesBinary(parseFile.SizeBytes)))
			parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
		}
		if len(parseFile.Imports) > 10 {
			parseWarnings = append(parseWarnings, fmt.Sprintf("%s imports %d packages from one client-owned file", parseFile.RelPath, len(parseFile.Imports)))
			parseLocations = appendDoctorLocation(parseLocations, parseFile.RelPath)
		}
	}
	if len(parseWarnings) == 0 {
		return doctorCheck{
			RuleID:  "audit.startup_evidence",
			Name:    "Startup cost and ownership evidence",
			Status:  "pass",
			Summary: "No first-pass startup-cost evidence warnings were detected in client-owned source files.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.startup_evidence",
		Name:      "Startup cost and ownership evidence",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(parseWarnings, 3),
		Locations: parseLocations,
		Hint:      "Keep large or dependency-heavy files out of the initial client path when they can stay server-owned or move behind a lazy boundary.",
	}
}

func buildDoctorRuntimeEvidenceCheck(parseCwd string) doctorCheck {
	parseStartupReports, parseWasmFiles, parseErr := collectDoctorRuntimeArtifacts(parseCwd)
	if parseErr != nil {
		return doctorCheck{
			RuleID:  "audit.runtime_evidence",
			Name:    "Runtime evidence",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not collect runtime evidence artifacts: %v", parseErr),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	if len(parseStartupReports) == 0 && len(parseWasmFiles) == 0 {
		return doctorCheck{
			Name:    "Runtime evidence",
			Status:  "pass",
			Summary: "No local runtime evidence artifacts were found yet.",
		}
	}
	parseWarnings := []string{}
	parseSummaries := []string{}
	parseLocations := []string{}
	if len(parseWasmFiles) > 0 {
		parseWasm := parseWasmFiles[0]
		parseLocations = appendDoctorLocation(parseLocations, parseWasm.RelPath)
		parseSummaries = append(parseSummaries, fmt.Sprintf("wasm %s (%s)", parseWasm.RelPath, formatBytesBinary(parseWasm.SizeBytes)))
		if parseWasm.SizeBytes > 5*1024*1024 {
			parseWarnings = append(parseWarnings, fmt.Sprintf("%s weighs %s on disk", parseWasm.RelPath, formatBytesBinary(parseWasm.SizeBytes)))
		}
	}
	if len(parseStartupReports) > 0 {
		parseReport := parseStartupReports[0]
		parseLocations = appendDoctorLocation(parseLocations, parseReport.RelPath)
		parseSummaries = append(parseSummaries, fmt.Sprintf("startup %s", parseReport.RelPath))
		if parseReport.ReadyMs != nil {
			parseSummaries = append(parseSummaries, fmt.Sprintf("ready=%.0fms", *parseReport.ReadyMs))
			if *parseReport.ReadyMs > 2500 {
				parseWarnings = append(parseWarnings, fmt.Sprintf("%s reports readyMs=%.0f", parseReport.RelPath, *parseReport.ReadyMs))
			}
		}
		if parseReport.InteractionMs != nil {
			parseSummaries = append(parseSummaries, fmt.Sprintf("interaction=%.0fms", *parseReport.InteractionMs))
			if *parseReport.InteractionMs > 500 {
				parseWarnings = append(parseWarnings, fmt.Sprintf("%s reports interactionMs=%.0f", parseReport.RelPath, *parseReport.InteractionMs))
			}
		}
	}
	if len(parseWarnings) == 0 {
		return doctorCheck{
			RuleID:    "audit.runtime_evidence",
			Name:      "Runtime evidence",
			Status:    "pass",
			Summary:   strings.Join(parseSummaries, " | "),
			Locations: parseLocations,
		}
	}
	return doctorCheck{
		RuleID:    "audit.runtime_evidence",
		Name:      "Runtime evidence",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(parseWarnings, 3),
		Locations: parseLocations,
		Hint:      "Collect and track wasm payload size plus startup probe timing so the audit can distinguish cheap shells from expensive startup paths with observed evidence.",
	}
}
