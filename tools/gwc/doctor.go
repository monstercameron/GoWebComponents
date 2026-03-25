package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
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

func (l launcher) runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	host := fs.String("host", defaultHost, "Host to probe for port availability")
	port := fs.String("port", "8080", "Port to probe for local development availability")
	audit := fs.Bool("audit", false, "Run the golden-path app audit in addition to prerequisite checks")
	auditPolicy := fs.String("audit-policy", "strict", "Golden-path audit policy: strict or advisory")
	auditBaseline := fs.String("audit-baseline", "", "Optional path to a JSON baseline file of accepted audit findings")
	auditWriteBaseline := fs.String("audit-write-baseline", "", "Optional path to write the current audit findings as a JSON baseline")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	var auditSuppressions stringListFlag
	fs.Var(&auditSuppressions, "audit-suppress", "Audit check name to suppress; repeat or comma-separate")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	report := l.buildDoctorReport(doctorConfig{
		host:               *host,
		port:               *port,
		audit:              *audit,
		auditPolicy:        *auditPolicy,
		auditBaselinePath:  *auditBaseline,
		auditWriteBaseline: *auditWriteBaseline,
		auditSuppressions:  auditSuppressions.Values(),
		json:               *jsonOutput,
	})
	if *audit && strings.TrimSpace(*auditWriteBaseline) != "" && report.Audit != nil {
		if err := writeDoctorAuditBaseline(*auditWriteBaseline, *report.Audit); err != nil {
			return err
		}
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		if err := encoder.Encode(report); err != nil {
			return err
		}
	} else {
		printDoctorReport(report)
	}
	if !report.OK {
		return errors.New("doctor found required checks that need attention")
	}
	return nil
}

func (l launcher) buildDoctorReport(config doctorConfig) doctorReport {
	cwd, err := doctorGetwd()
	if err != nil {
		cwd = ""
	}
	report := doctorReport{
		OK:         true,
		Checked:    time.Now().UTC().Format(time.RFC3339),
		CWD:        cwd,
		Resolution: buildDoctorResolutionTrace(cwd, config),
	}
	appendCheck := func(check doctorCheck) {
		report.Checks = append(report.Checks, check)
		if check.Status == "fail" {
			report.OK = false
		}
	}

	appendCheck(buildDoctorToolCheck("go", "Go toolchain", "version", "Install Go 1.25 or newer and ensure `go` is on PATH."))
	appendCheck(buildDoctorToolCheck("node", "Node.js", "--version", "Install Node.js for browser tooling and repo-local scripts."))
	appendCheck(buildDoctorToolCheck("npm", "npm", "--version", "Install npm alongside Node.js so repo test and asset workflows can run."))
	appendCheck(buildDoctorWasmExecCheck())
	appendCheck(buildDoctorPlaywrightCheck(l.repoRoot))
	appendCheck(buildDoctorMetadataCheck(cwd))
	appendCheck(buildDoctorProjectDetectionCheck(cwd))
	appendCheck(buildDoctorPortCheck(config.host, config.port))
	if config.audit {
		report.Audit = buildDoctorAuditReport(cwd, config)
		if report.Audit.Policy == "strict" && !report.Audit.OK {
			report.OK = false
		}
	}

	return report
}

func buildDoctorResolutionTrace(cwd string, config doctorConfig) map[string]string {
	trace := map[string]string{}
	if strings.TrimSpace(cwd) != "" {
		trace["root"] = "working directory"
	}
	if strings.TrimSpace(config.host) != "" && config.host != defaultHost {
		trace["host"] = "explicit flag"
	} else {
		trace["host"] = "convention fallback"
	}
	if strings.TrimSpace(config.port) != "" && config.port != "8080" {
		trace["port"] = "explicit flag"
	} else {
		trace["port"] = "convention fallback"
	}
	metadata, metadataDir, hasMetadata, err := resolveScaffoldMetadataForConfig(cwd, "", "")
	if err == nil && hasMetadata {
		if strings.TrimSpace(metadata.Tooling.AppPath) != "" {
			trace["app"] = "gwc-start.json"
		}
		if strings.TrimSpace(metadata.Tooling.HTMLPath) != "" {
			trace["html"] = "gwc-start.json"
		}
		if strings.TrimSpace(metadataDir) != "" {
			trace["root"] = "gwc-start.json"
		}
	}
	if _, ok := trace["app"]; !ok && strings.TrimSpace(cwd) != "" {
		if _, err := detectAppPath(cwd); err == nil {
			trace["app"] = "convention fallback"
		}
	}
	if _, ok := trace["html"]; !ok && strings.TrimSpace(cwd) != "" {
		if strings.TrimSpace(detectHTMLPath(cwd)) != "" {
			trace["html"] = "convention fallback"
		}
	}
	return trace
}

func buildDoctorAuditReport(cwd string, config doctorConfig) *doctorAuditReport {
	audit := buildDoctorGoldenPathAudit(cwd)
	applyDoctorAuditAdoption(&audit, cwd, config)
	if strings.TrimSpace(audit.Policy) == "" {
		audit.Policy = "strict"
	}
	return &audit
}

func applyDoctorAuditAdoption(audit *doctorAuditReport, cwd string, config doctorConfig) {
	if audit == nil {
		return
	}
	policy, ok := normalizeDoctorAuditPolicy(config.auditPolicy)
	if !ok {
		audit.Policy = "strict"
		audit.OK = false
		audit.Checks = append([]doctorCheck{{
			Name:    "Audit policy",
			Status:  "fail",
			Summary: fmt.Sprintf("Unknown audit policy %q.", config.auditPolicy),
			Hint:    "Use -audit-policy strict or -audit-policy advisory.",
		}}, audit.Checks...)
		annotateDoctorAuditMetadata(audit)
		return
	}
	audit.Policy = policy
	suppressed := map[string]struct{}{}
	for _, name := range config.auditSuppressions {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			suppressed[trimmed] = struct{}{}
		}
	}
	if strings.TrimSpace(config.auditBaselinePath) != "" {
		baselinePath, err := normalizePath(cwd, config.auditBaselinePath)
		if err != nil {
			audit.OK = false
			audit.Checks = append([]doctorCheck{{
				Name:    "Audit baseline",
				Status:  "fail",
				Summary: fmt.Sprintf("Could not resolve audit baseline path: %v", err),
				Hint:    "Pass a valid file path to -audit-baseline or remove the flag.",
			}}, audit.Checks...)
			annotateDoctorAuditMetadata(audit)
			return
		}
		baseline, err := loadDoctorAuditBaseline(baselinePath)
		if err != nil {
			audit.OK = false
			audit.Checks = append([]doctorCheck{{
				Name:    "Audit baseline",
				Status:  "fail",
				Summary: err.Error(),
				Hint:    "Write a fresh baseline with -audit-write-baseline or fix the checked-in baseline file.",
			}}, audit.Checks...)
			annotateDoctorAuditMetadata(audit)
			return
		}
		audit.BaselinePath = baselinePath
		for _, entry := range baseline.Checks {
			trimmed := strings.TrimSpace(entry.Name)
			if trimmed != "" {
				suppressed[trimmed] = struct{}{}
			}
		}
	}
	if len(suppressed) == 0 {
		audit.OK = doctorAuditChecksPassing(audit.Checks)
		annotateDoctorAuditMetadata(audit)
		return
	}
	names := make([]string, 0, len(suppressed))
	for name := range suppressed {
		names = append(names, name)
	}
	sort.Strings(names)
	audit.Suppressed = names
	for i := range audit.Checks {
		check := &audit.Checks[i]
		if check.Status == "pass" || check.Status == "suppressed" {
			continue
		}
		if _, ok := suppressed[check.Name]; ok {
			check.Status = "suppressed"
			check.Summary = check.Summary + " (suppressed)"
		}
	}
	audit.OK = doctorAuditChecksPassing(audit.Checks)
	annotateDoctorAuditMetadata(audit)
}

func normalizeDoctorAuditPolicy(value string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "strict":
		return "strict", true
	case "advisory", "warn":
		return "advisory", true
	default:
		return "", false
	}
}

func doctorAuditChecksPassing(checks []doctorCheck) bool {
	for _, check := range checks {
		if check.Status == "fail" {
			return false
		}
	}
	return true
}

func annotateDoctorAuditMetadata(audit *doctorAuditReport) {
	if audit == nil {
		return
	}
	for i := range audit.Checks {
		check := &audit.Checks[i]
		if check.RuleID == "" {
			check.RuleID = doctorAuditRuleIDForName(check.Name)
		}
		check.Severity = doctorAuditSeverityForStatus(check.Status)
		if len(check.Locations) == 0 {
			check.Locations = extractDoctorAuditLocations(check.Summary)
		}
		if strings.TrimSpace(check.Remediation) == "" && strings.TrimSpace(check.Hint) != "" {
			check.Remediation = check.Hint
		}
	}
}

func appendDoctorLocation(locations []string, value string) []string {
	trimmed := strings.TrimSpace(filepath.ToSlash(value))
	if trimmed == "" {
		return locations
	}
	for _, existing := range locations {
		if existing == trimmed {
			return locations
		}
	}
	return append(locations, trimmed)
}

func doctorAuditLocation(root string, path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if strings.TrimSpace(root) != "" {
		if rel, err := filepath.Rel(root, trimmed); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(trimmed)
}

func doctorAuditRuleIDForName(name string) string {
	switch name {
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

func doctorAuditSeverityForStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
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

func normalizeDoctorAuditMinimumSeverity(value string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
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

func doctorAuditSeverityRank(value string) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
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

func doctorAuditHasFindingAtOrAbove(checks []doctorCheck, minimum string) bool {
	normalized, ok := normalizeDoctorAuditMinimumSeverity(minimum)
	if !ok || normalized == "off" {
		return false
	}
	requiredRank := doctorAuditSeverityRank(normalized)
	for _, check := range checks {
		if check.Status == "pass" || check.Status == "suppressed" {
			continue
		}
		if doctorAuditSeverityRank(check.Severity) >= requiredRank {
			return true
		}
	}
	return false
}

func extractDoctorAuditLocations(summary string) []string {
	matches := doctorAuditLocationPattern.FindAllString(summary, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	locations := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		locations = append(locations, match)
	}
	return locations
}

func buildDoctorGoldenPathAudit(cwd string) doctorAuditReport {
	report := doctorAuditReport{
		Mode: "golden-path",
		OK:   true,
	}
	appendCheck := func(check doctorCheck) {
		report.Checks = append(report.Checks, check)
		if check.Status == "fail" {
			report.OK = false
		}
	}
	if strings.TrimSpace(cwd) == "" {
		appendCheck(doctorCheck{
			RuleID:  "audit.target",
			Name:    "Audit target",
			Status:  "fail",
			Summary: "The current working directory could not be resolved for golden-path auditing.",
			Hint:    "Run `gwc doctor -audit` from the target app root.",
		})
		annotateDoctorAuditMetadata(&report)
		return report
	}
	appPath, appErr := detectAppPath(cwd)
	if appErr != nil {
		appendCheck(doctorCheck{
			RuleID:  "audit.app_entrypoint",
			Name:    "App entrypoint",
			Status:  "fail",
			Summary: "No launcher-detectable app entrypoint was found for golden-path auditing.",
			Hint:    "Keep main.go or cmd/web/main.go at the documented locations, or add scaffold metadata that pins the app path.",
		})
	} else {
		appendCheck(doctorCheck{
			RuleID:    "audit.app_entrypoint",
			Name:      "App entrypoint",
			Status:    "pass",
			Summary:   fmt.Sprintf("Auditing app entrypoint %s", appPath),
			Locations: []string{doctorAuditLocation(cwd, appPath)},
		})
	}
	htmlPath := detectHTMLPath(cwd)
	if strings.TrimSpace(htmlPath) == "" {
		appendCheck(doctorCheck{
			RuleID:  "audit.html_shell",
			Name:    "HTML shell",
			Status:  "warn",
			Summary: "No HTML shell was auto-detected for the current app root.",
			Hint:    "Keep index.html or a documented equivalent near the app so later delivery audits can reason about the served shell.",
		})
	} else {
		appendCheck(doctorCheck{
			RuleID:    "audit.html_shell",
			Name:      "HTML shell",
			Status:    "pass",
			Summary:   fmt.Sprintf("Detected HTML shell %s", htmlPath),
			Locations: []string{doctorAuditLocation(cwd, htmlPath)},
		})
	}
	metadata, ok, err := loadScaffoldMetadata(cwd)
	if err != nil {
		appendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "fail",
			Summary:   err.Error(),
			Locations: []string{"gwc-start.json"},
			Hint:      "Fix or regenerate gwc-start.json so golden-path audits can resolve intended launcher ownership.",
		})
	} else if !ok {
		appendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "warn",
			Summary:   "No gwc-start.json metadata anchor was found for this app.",
			Locations: []string{"gwc-start.json"},
			Hint:      "Generated starters should keep scaffold metadata so future audit rules can trace intended ownership and output paths.",
		})
	} else {
		appendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "pass",
			Summary:   fmt.Sprintf("Using starter metadata for %s", firstNonEmpty(metadata.ProjectName, "<unnamed>")),
			Locations: []string{"gwc-start.json"},
		})
	}
	appendCheck(buildDoctorOwnershipBoundaryCheck(cwd))
	appendCheck(buildDoctorStateOwnershipCheck(cwd))
	appendCheck(buildDoctorRouteDeliveryCheck(cwd))
	appendCheck(buildDoctorMutationResilienceCheck(cwd))
	appendCheck(buildDoctorStartupEvidenceCheck(cwd))
	appendCheck(buildDoctorRuntimeEvidenceCheck(cwd))
	annotateDoctorAuditMetadata(&report)
	return report
}

func buildDoctorOwnershipBoundaryCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.state_boundaries",
			Name:    "State and ownership boundaries",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for ownership-boundary auditing: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	violations := []string{}
	locations := []string{}
	for _, file := range files {
		for _, importPath := range file.Imports {
			switch {
			case file.Client && strings.Contains(importPath, "/server/"):
				violations = append(violations, fmt.Sprintf("%s imports server package %s", file.RelPath, importPath))
				locations = appendDoctorLocation(locations, file.RelPath)
			case file.Client && (importPath == "database/sql" || importPath == "os/exec"):
				violations = append(violations, fmt.Sprintf("%s imports server-only package %s", file.RelPath, importPath))
				locations = appendDoctorLocation(locations, file.RelPath)
			case !file.Client && importPath == "syscall/js":
				violations = append(violations, fmt.Sprintf("%s imports browser-only package %s outside a js/wasm boundary", file.RelPath, importPath))
				locations = appendDoctorLocation(locations, file.RelPath)
			}
		}
	}
	if len(violations) == 0 {
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
		Summary:   summarizeDoctorViolations(violations, 3),
		Locations: locations,
		Hint:      "Keep client files on browser-safe imports, keep server packages out of js/wasm paths, and keep syscall/js usage behind explicit js/wasm build tags.",
	}
}

func buildDoctorStateOwnershipCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.state_ownership",
			Name:    "Local versus shared state ownership",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for mixed state ownership heuristics: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	violations := []string{}
	locations := []string{}
	for _, file := range files {
		if !file.Client {
			continue
		}
		if strings.Contains(file.Content, "fetch.UseCachedResource") &&
			(strings.Contains(file.Content, "localStorage") || strings.Contains(file.Content, "sessionStorage") || strings.Contains(file.Content, "indexedDB")) {
			violations = append(violations, fmt.Sprintf("%s mixes fetch.UseCachedResource with direct browser storage access", file.RelPath))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if len(violations) == 0 {
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
		Summary:   summarizeDoctorViolations(violations, 3),
		Locations: locations,
		Hint:      "Prefer one obvious owner per state slice: either fetch/cache-backed shared data or browser-local persistence, not both in the same controller without an explicit boundary.",
	}
}

func buildDoctorRouteDeliveryCheck(cwd string) doctorCheck {
	goFiles, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for route-shape auditing: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	htmlFiles, err := collectDoctorHTMLFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan HTML shells for route-shape auditing: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	warnings := []string{}
	locations := []string{}
	if len(htmlFiles) > 1 {
		warnings = append(warnings, fmt.Sprintf("multiple HTML entry shells detected (%s)", strings.Join(htmlFiles, ", ")))
		for _, path := range htmlFiles {
			locations = appendDoctorLocation(locations, path)
		}
	}
	routeCount := 0
	hasLazySplit := false
	hasPrerenderSignal := false
	marketingRoutes := false
	for _, file := range goFiles {
		routeHits := strings.Count(file.Content, "MustDefineRoute(")
		routeHits += strings.Count(file.Content, "router.Register(")
		if routeHits > 0 {
			routeCount += routeHits
			locations = appendDoctorLocation(locations, file.RelPath)
		}
		if strings.Contains(file.Content, "ui.Lazy(") || strings.Contains(file.Content, "ui.CreateElement(ui.Lazy") {
			hasLazySplit = true
		}
		lowered := strings.ToLower(file.Content)
		if strings.Contains(lowered, "prerender") || strings.Contains(lowered, "static shell") {
			hasPrerenderSignal = true
		}
		if strings.Contains(file.Content, `"/pricing"`) || strings.Contains(file.Content, `"/capabilities"`) || strings.Contains(file.Content, `"/about"`) || strings.Contains(file.Content, `"/docs"`) {
			marketingRoutes = true
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if routeCount >= 3 && !hasLazySplit {
		warnings = append(warnings, fmt.Sprintf("route tree defines %d route registration points with no ui.Lazy split signal", routeCount))
	}
	if marketingRoutes && !hasPrerenderSignal {
		warnings = append(warnings, "marketing-style routes were detected with no static/prerender delivery hint")
	}
	if len(warnings) == 0 {
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
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Prefer one app shell, prerender or keep static-friendly marketing routes cheap, and use ui.Lazy or equivalent delivery splits when route trees start carrying distinct feature surfaces.",
	}
}

func buildDoctorMutationResilienceCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.mutation_resilience",
			Name:    "Mutation and resilience",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for mutation resilience heuristics: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	mutationIndicators := []string{"http.methodpost", "http.methodput", "http.methodpatch", "http.methoddelete", ".exec(", "deleteconversation", "upsert", "setselected", "setcustom", "signup(", "login(", "save", "submit"}
	resilienceIndicators := []string{"retry", "backoff", "idempot", "rollback", "conflict", "offline", "replay", "timeout", "deadline"}
	warnings := []string{}
	locations := []string{}
	for _, file := range files {
		lowered := strings.ToLower(file.Content)
		hasMutation := false
		for _, indicator := range mutationIndicators {
			if strings.Contains(lowered, indicator) {
				hasMutation = true
				break
			}
		}
		if !hasMutation {
			continue
		}
		hasResilienceSignal := false
		for _, indicator := range resilienceIndicators {
			if strings.Contains(lowered, indicator) {
				hasResilienceSignal = true
				break
			}
		}
		if !hasResilienceSignal {
			warnings = append(warnings, fmt.Sprintf("%s exposes mutation-shaped code with no retry/idempotency/conflict/offline signal", file.RelPath))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if len(warnings) == 0 {
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
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Document or encode retry posture, idempotency boundaries, rollback/conflict handling, or offline replay semantics around important writes instead of shipping only the happy path.",
	}
}

func buildDoctorStartupEvidenceCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.startup_evidence",
			Name:    "Startup cost and ownership evidence",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for startup-cost evidence: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	warnings := []string{}
	locations := []string{}
	for _, file := range files {
		if !file.Client {
			continue
		}
		if file.SizeBytes > 64*1024 {
			warnings = append(warnings, fmt.Sprintf("%s is %s of client-owned source in the initial app path", file.RelPath, formatBytesBinary(file.SizeBytes)))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
		if len(file.Imports) > 10 {
			warnings = append(warnings, fmt.Sprintf("%s imports %d packages from one client-owned file", file.RelPath, len(file.Imports)))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if len(warnings) == 0 {
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
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Keep large or dependency-heavy files out of the initial client path when they can stay server-owned or move behind a lazy boundary.",
	}
}

func buildDoctorRuntimeEvidenceCheck(cwd string) doctorCheck {
	startupReports, wasmFiles, err := collectDoctorRuntimeArtifacts(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.runtime_evidence",
			Name:    "Runtime evidence",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not collect runtime evidence artifacts: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	if len(startupReports) == 0 && len(wasmFiles) == 0 {
		return doctorCheck{
			Name:    "Runtime evidence",
			Status:  "pass",
			Summary: "No local runtime evidence artifacts were found yet.",
		}
	}
	warnings := []string{}
	summaries := []string{}
	locations := []string{}
	if len(wasmFiles) > 0 {
		wasm := wasmFiles[0]
		locations = appendDoctorLocation(locations, wasm.RelPath)
		summaries = append(summaries, fmt.Sprintf("wasm %s (%s)", wasm.RelPath, formatBytesBinary(wasm.SizeBytes)))
		if wasm.SizeBytes > 5*1024*1024 {
			warnings = append(warnings, fmt.Sprintf("%s weighs %s on disk", wasm.RelPath, formatBytesBinary(wasm.SizeBytes)))
		}
	}
	if len(startupReports) > 0 {
		report := startupReports[0]
		locations = appendDoctorLocation(locations, report.RelPath)
		summaries = append(summaries, fmt.Sprintf("startup %s", report.RelPath))
		if report.ReadyMs != nil {
			summaries = append(summaries, fmt.Sprintf("ready=%.0fms", *report.ReadyMs))
			if *report.ReadyMs > 2500 {
				warnings = append(warnings, fmt.Sprintf("%s reports readyMs=%.0f", report.RelPath, *report.ReadyMs))
			}
		}
		if report.InteractionMs != nil {
			summaries = append(summaries, fmt.Sprintf("interaction=%.0fms", *report.InteractionMs))
			if *report.InteractionMs > 500 {
				warnings = append(warnings, fmt.Sprintf("%s reports interactionMs=%.0f", report.RelPath, *report.InteractionMs))
			}
		}
	}
	if len(warnings) == 0 {
		return doctorCheck{
			RuleID:    "audit.runtime_evidence",
			Name:      "Runtime evidence",
			Status:    "pass",
			Summary:   strings.Join(summaries, " | "),
			Locations: locations,
		}
	}
	return doctorCheck{
		RuleID:    "audit.runtime_evidence",
		Name:      "Runtime evidence",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Collect and track wasm payload size plus startup probe timing so the audit can distinguish cheap shells from expensive startup paths with observed evidence.",
	}
}

type doctorGoFileRecord struct {
	RelPath   string
	Content   string
	Imports   []string
	Client    bool
	SizeBytes int64
}

type doctorRuntimeArtifact struct {
	RelPath   string
	SizeBytes int64
}

type doctorStartupEvidence struct {
	RelPath       string
	ReadyMs       *float64
	InteractionMs *float64
}

func collectGoldenPathGoFiles(root string) ([]doctorGoFileRecord, error) {
	records := []doctorGoFileRecord{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDoctorAuditDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" {
			return nil
		}
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(contentBytes)
		file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return err
		}
		imports := make([]string, 0, len(file.Imports))
		for _, spec := range file.Imports {
			imports = append(imports, strings.Trim(spec.Path.Value, `"`))
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		records = append(records, doctorGoFileRecord{
			RelPath:   filepath.ToSlash(relPath),
			Content:   content,
			Imports:   imports,
			Client:    isDoctorAuditClientFile(filepath.ToSlash(path), content),
			SizeBytes: int64(len(contentBytes)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func collectDoctorHTMLFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDoctorAuditDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".html" {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		files = append(files, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func collectDoctorRuntimeArtifacts(root string) ([]doctorStartupEvidence, []doctorRuntimeArtifact, error) {
	startupReports := []doctorStartupEvidence{}
	wasmFiles := []doctorRuntimeArtifact{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDoctorRuntimeDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)
		switch {
		case strings.EqualFold(entry.Name(), "wasm-startup-report.json"):
			report, err := readDoctorStartupEvidence(path, relPath)
			if err != nil {
				return err
			}
			startupReports = append(startupReports, report)
		case strings.EqualFold(filepath.Ext(entry.Name()), ".wasm"):
			info, err := entry.Info()
			if err != nil {
				return err
			}
			wasmFiles = append(wasmFiles, doctorRuntimeArtifact{RelPath: relPath, SizeBytes: info.Size()})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(startupReports, func(i int, j int) bool { return startupReports[i].RelPath < startupReports[j].RelPath })
	sort.Slice(wasmFiles, func(i int, j int) bool {
		if wasmFiles[i].SizeBytes != wasmFiles[j].SizeBytes {
			return wasmFiles[i].SizeBytes > wasmFiles[j].SizeBytes
		}
		return wasmFiles[i].RelPath < wasmFiles[j].RelPath
	})
	return startupReports, wasmFiles, nil
}

func shouldSkipDoctorRuntimeDir(name string) bool {
	switch strings.TrimSpace(name) {
	case ".git", "node_modules", "vendor", "dist", "tmp":
		return true
	default:
		return false
	}
}

func readDoctorStartupEvidence(path string, relPath string) (doctorStartupEvidence, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return doctorStartupEvidence{}, err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		return doctorStartupEvidence{}, err
	}
	evidence := doctorStartupEvidence{RelPath: relPath}
	if startup, ok := payload["startup"].(map[string]interface{}); ok {
		if readyMs, ok := startup["readyMs"].(float64); ok {
			evidence.ReadyMs = &readyMs
		}
		if interactionMs, ok := startup["interactionMs"].(float64); ok {
			evidence.InteractionMs = &interactionMs
		}
	}
	return evidence, nil
}

func shouldSkipDoctorAuditDir(name string) bool {
	switch strings.TrimSpace(name) {
	case ".git", "node_modules", "vendor", "bin", "dist", "tmp":
		return true
	default:
		return false
	}
}

func isDoctorAuditClientFile(path string, content string) bool {
	if strings.Contains(content, "//go:build js && wasm") {
		return true
	}
	return strings.Contains(path, "/client/")
}

func summarizeDoctorViolations(violations []string, limit int) string {
	if len(violations) == 0 {
		return ""
	}
	if limit <= 0 || len(violations) <= limit {
		return strings.Join(violations, " | ")
	}
	return fmt.Sprintf("%s | +%d more", strings.Join(violations[:limit], " | "), len(violations)-limit)
}

func formatBytesBinary(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	kib := float64(size) / 1024
	if kib < 1024 {
		return fmt.Sprintf("%.1f KiB", kib)
	}
	return fmt.Sprintf("%.1f MiB", kib/1024)
}

func loadDoctorAuditBaseline(path string) (doctorAuditBaseline, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return doctorAuditBaseline{}, fmt.Errorf("read audit baseline: %w", err)
	}
	var baseline doctorAuditBaseline
	if err := json.Unmarshal(content, &baseline); err != nil {
		return doctorAuditBaseline{}, fmt.Errorf("parse audit baseline: %w", err)
	}
	return baseline, nil
}

func writeDoctorAuditBaseline(path string, audit doctorAuditReport) error {
	resolvedPath := path
	if cwd, err := doctorGetwd(); err == nil {
		if normalized, normalizeErr := normalizePath(cwd, path); normalizeErr == nil && strings.TrimSpace(normalized) != "" {
			resolvedPath = normalized
		}
	}
	if dir := filepath.Dir(resolvedPath); strings.TrimSpace(dir) != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create audit baseline directory: %w", err)
		}
	}
	baseline := doctorAuditBaseline{
		Mode:        audit.Mode,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Checks:      make([]doctorAuditBaselineCheck, 0, len(audit.Checks)),
	}
	for _, check := range audit.Checks {
		if check.Status == "pass" {
			continue
		}
		baseline.Checks = append(baseline.Checks, doctorAuditBaselineCheck{
			Name:    check.Name,
			Status:  check.Status,
			Summary: check.Summary,
		})
	}
	content, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal audit baseline: %w", err)
	}
	if err := os.WriteFile(resolvedPath, append(content, '\n'), 0644); err != nil {
		return fmt.Errorf("write audit baseline: %w", err)
	}
	return nil
}

func buildDoctorToolCheck(command string, name string, versionArg string, hint string) doctorCheck {
	path, err := doctorLookPath(command)
	if err != nil {
		return doctorCheck{Name: name, Status: "fail", Summary: fmt.Sprintf("%s was not found on PATH.", command), Hint: hint}
	}
	output, err := doctorCommandOutput(command, versionArg)
	if err != nil {
		summary := strings.TrimSpace(output)
		if summary == "" {
			summary = err.Error()
		}
		return doctorCheck{Name: name, Status: "fail", Summary: fmt.Sprintf("%s is on PATH at %s but did not report a version: %s", command, path, summary), Hint: hint}
	}
	return doctorCheck{Name: name, Status: "pass", Summary: fmt.Sprintf("%s (%s)", output, path)}
}

func buildDoctorWasmExecCheck() doctorCheck {
	wasmExecPath, err := doctorResolveWasmExec()
	if err != nil {
		return doctorCheck{Name: "wasm_exec.js", Status: "fail", Summary: "Matching wasm_exec.js could not be resolved from the active Go toolchain.", Hint: "Use the same Go toolchain for both the wasm binary and wasm_exec.js."}
	}
	return doctorCheck{Name: "wasm_exec.js", Status: "pass", Summary: fmt.Sprintf("Resolved matching runtime asset at %s", wasmExecPath)}
}

func buildDoctorPlaywrightCheck(repoRoot string) doctorCheck {
	workspace, err := resolveBrowserWorkspace(repoRoot, repoRoot)
	if err != nil {
		return doctorCheck{Name: "Browser tests", Status: "fail", Summary: err.Error(), Hint: "Fix the browserWorkspace override or remove it so launcher defaults can be used."}
	}
	if strings.TrimSpace(workspace) == "" {
		return doctorCheck{Name: "Browser tests", Status: "warn", Summary: "The repo test/package.json file was not found.", Hint: "Run doctor from the repo or restore the test workspace if browser coverage matters."}
	}
	playwrightPackagePath := filepath.Join(workspace, "node_modules", "@playwright", "test", "package.json")
	if fileExists(playwrightPackagePath) {
		return doctorCheck{Name: "Browser tests", Status: "pass", Summary: fmt.Sprintf("Playwright is installed at %s", playwrightPackagePath)}
	}
	packagePattern, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(workspace)
	if hasPlaywrightGoSuite {
		return doctorCheck{Name: "Browser tests", Status: "pass", Summary: fmt.Sprintf("Playwright-Go browser suite is available in %s (%s).", workspace, packagePattern)}
	}
	return doctorCheck{
		Name:    "Browser tests",
		Status:  "warn",
		Summary: fmt.Sprintf("Playwright dependencies are not installed under %s.", filepath.Join(workspace, "node_modules")),
		Hint:    fmt.Sprintf("Run `go test -tags playwrightgo %s -run TestMainSuite -v` from %s or install JS Playwright dependencies if you still rely on legacy suites.", firstNonEmpty(packagePattern, "./playwrightgo"), workspace),
	}
}

func buildDoctorMetadataCheck(cwd string) doctorCheck {
	if strings.TrimSpace(cwd) == "" {
		return doctorCheck{Name: "Scaffold metadata", Status: "warn", Summary: "The current working directory could not be resolved.", Hint: "Run doctor from the target app directory to inspect scaffold metadata."}
	}
	metadata, ok, err := loadScaffoldMetadata(cwd)
	if err != nil {
		return doctorCheck{Name: "Scaffold metadata", Status: "fail", Summary: err.Error(), Hint: "Fix or regenerate the scaffold metadata file."}
	}
	if !ok {
		return doctorCheck{Name: "Scaffold metadata", Status: "warn", Summary: "No gwc-start.json metadata file was found in the current directory.", Hint: "Generated starters should carry scaffold metadata; hand-built apps can ignore this warning for now."}
	}
	projectName := firstNonEmpty(metadata.ProjectName, "<unnamed>")
	modulePath := firstNonEmpty(metadata.ModulePath, "<missing modulePath>")
	return doctorCheck{Name: "Scaffold metadata", Status: "pass", Summary: fmt.Sprintf("Detected starter metadata for %s (%s)", projectName, modulePath)}
}

func buildDoctorProjectDetectionCheck(cwd string) doctorCheck {
	if strings.TrimSpace(cwd) == "" {
		return doctorCheck{Name: "Project detection", Status: "warn", Summary: "The current working directory could not be resolved.", Hint: "Run doctor from an app root to preview gwc dev detection."}
	}
	appPath, appErr := detectAppPath(cwd)
	htmlPath := detectHTMLPath(cwd)
	if appErr != nil {
		return doctorCheck{Name: "Project detection", Status: "warn", Summary: "gwc dev would not auto-detect an app entrypoint in the current directory.", Hint: "Pass -app explicitly or keep main.go or cmd/web/main.go at the documented locations."}
	}
	parts := []string{fmt.Sprintf("App entrypoint: %s", appPath)}
	if strings.TrimSpace(htmlPath) != "" {
		parts = append(parts, fmt.Sprintf("HTML shell: %s", htmlPath))
	}
	return doctorCheck{Name: "Project detection", Status: "pass", Summary: strings.Join(parts, " | ")}
}

func buildDoctorPortCheck(host string, port string) doctorCheck {
	address := joinHostPort(host, port)
	listener, err := doctorListen("tcp", address)
	if err != nil {
		return doctorCheck{Name: "Port availability", Status: "fail", Summary: fmt.Sprintf("Could not bind %s: %v", address, err), Hint: "Stop the conflicting process or choose a different port before running gwc dev or gwc examples."}
	}
	_ = listener.Close()
	return doctorCheck{Name: "Port availability", Status: "pass", Summary: fmt.Sprintf("Port %s is available for local launcher commands.", address)}
}

func printDoctorReport(report doctorReport) {
	status := "PASS"
	if !report.OK {
		status = "FAIL"
	}
	fmt.Printf("GWC doctor: %s\n", status)
	if strings.TrimSpace(report.CWD) != "" {
		fmt.Printf("  cwd: %s\n", report.CWD)
	}
	for _, check := range report.Checks {
		label := strings.ToUpper(check.Status)
		fmt.Printf("  [%s] %s: %s\n", label, check.Name, check.Summary)
		if strings.TrimSpace(check.Hint) != "" && check.Status != "pass" {
			fmt.Printf("         hint: %s\n", check.Hint)
		}
	}
	if report.Audit != nil {
		auditStatus := "PASS"
		if !report.Audit.OK {
			auditStatus = "FAIL"
		}
		fmt.Printf("  audit[%s]: %s", report.Audit.Mode, auditStatus)
		if report.Audit.Policy != "" {
			fmt.Printf(" (policy: %s)", report.Audit.Policy)
		}
		fmt.Println()
		if strings.TrimSpace(report.Audit.BaselinePath) != "" {
			fmt.Printf("    baseline: %s\n", report.Audit.BaselinePath)
		}
		if len(report.Audit.Suppressed) > 0 {
			fmt.Printf("    suppressed: %s\n", strings.Join(report.Audit.Suppressed, ", "))
		}
		for _, check := range report.Audit.Checks {
			label := strings.ToUpper(check.Status)
			fmt.Printf("    [%s] %s: %s\n", label, check.Name, check.Summary)
			if strings.TrimSpace(check.Hint) != "" && check.Status != "pass" {
				fmt.Printf("           hint: %s\n", check.Hint)
			}
		}
	}
	printResolutionTrace(report.Resolution, []string{"app", "root", "html", "host", "port"}, "  ")
}
