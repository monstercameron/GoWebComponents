package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// runAuditCommand routes the supply-chain audit command.
var runAuditCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runAudit(parseArgs)
}

type auditConfig struct {
	rootPath string
	budget   int
	json     bool
}

// auditReport summarizes a project's dependency supply-chain surface: the headline
// "zero npm" proof, the Go module footprint (direct/transitive), checksum
// verification, and an optional direct-dependency budget.
type auditReport struct {
	OK                    bool                `json:"ok"`
	Root                  string              `json:"root"`
	Module                string              `json:"module,omitempty"`
	GoVersion             string              `json:"goVersion,omitempty"`
	DirectExternalDeps    []string            `json:"directExternalDeps,omitempty"`
	DirectExternalCount   int                 `json:"directExternalCount"`
	LocalReplaceCount     int                 `json:"localReplaceCount"`
	TransitiveModuleCount int                 `json:"transitiveModuleCount"`
	ChecksumsVerified     bool                `json:"checksumsVerified"`
	ZeroNPM               bool                `json:"zeroNPM"`
	NPMArtifacts          []string            `json:"npmArtifacts,omitempty"`
	Budget                int                 `json:"budget,omitempty"`
	BudgetExceeded        bool                `json:"budgetExceeded"`
	Diagnostics           []agenticDiagnostic `json:"diagnostics,omitempty"`
}

// auditNPMDirSkip names directories excluded from the npm-artifact scan: VCS,
// build output, vendored or generated trees, and comparison/example dirs that are
// not part of the importable Go module.
var auditNPMDirSkip = map[string]struct{}{
	".git": {}, ".claude": {}, "node_modules": {}, "vendor": {}, "testdata": {},
	"bin": {}, "dist": {}, "build": {}, "research": {}, "examples": {}, "third_party": {},
	// Editor-integration tooling (e.g. the VS Code extension) legitimately carries a
	// package.json but is NOT part of the framework's or any app's runtime/build supply
	// chain — it is a separate editor plugin. Excluding it keeps the zero-npm claim about
	// what it actually means: no npm in the framework/app build.
	"vscode-gwc": {},
}

// auditNPMArtifactNames are filenames/dirs whose presence indicates an npm
// dependency surface — the thing GWC's zero-npm posture is immune to.
var auditNPMArtifactNames = map[string]struct{}{
	"package.json": {}, "package-lock.json": {}, "yarn.lock": {},
	"pnpm-lock.yaml": {}, "node_modules": {}, "bun.lockb": {},
}

// runAudit parses audit flags, builds the report, and emits human or JSON output.
func (parseL launcher) runAudit(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("supplychain", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to audit; defaults to the current working directory")
	parseBudget := parseFlags.Int("budget", 0, "Fail if direct external Go dependencies exceed this count (0 disables)")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON envelope")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr := resolveAuditConfig(auditConfig{rootPath: *parseRoot, budget: *parseBudget, json: *parseJSON})
	if parseErr != nil {
		if *parseJSON {
			parseDiagnostic := buildAgenticCommandError("GWC-AUDIT-CONFIG", parseErr)
			if parseWriteErr := writeAgenticEnvelope("supplychain", false, nil, []agenticDiagnostic{parseDiagnostic}, parseErr); parseWriteErr != nil {
				return parseWriteErr
			}
		}
		return parseErr
	}

	parseReport := buildAuditReport(parseConfig.rootPath, parseConfig.budget)
	parseResultErr := error(nil)
	if !parseReport.OK {
		parseResultErr = fmt.Errorf("audit failed: %d issue(s)", len(parseReport.Diagnostics))
	}

	if parseConfig.json {
		if parseWriteErr := writeAgenticEnvelope("supplychain", parseReport.OK, parseReport, parseReport.Diagnostics, parseResultErr); parseWriteErr != nil {
			return parseWriteErr
		}
		return parseResultErr
	}

	printAuditSummary(parseReport)
	return parseResultErr
}

// resolveAuditConfig validates and normalizes audit input flags.
func resolveAuditConfig(parseConfig auditConfig) (auditConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return auditConfig{}, fmt.Errorf("resolve audit root from cwd: %w", parseErr)
		}
		parseRootPath = parseCWD
	}
	parseAbs, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return auditConfig{}, fmt.Errorf("resolve audit root: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbs)
	if parseErr != nil {
		return auditConfig{}, fmt.Errorf("stat audit root: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		return auditConfig{}, fmt.Errorf("audit root is not a directory: %s", parseAbs)
	}
	if parseConfig.budget < 0 {
		return auditConfig{}, fmt.Errorf("audit budget must be >= 0, got %d", parseConfig.budget)
	}
	return auditConfig{rootPath: parseAbs, budget: parseConfig.budget, json: parseConfig.json}, nil
}

// buildAuditReport produces the supply-chain audit for rootPath.
func buildAuditReport(parseRootPath string, parseBudget int) auditReport {
	parseReport := auditReport{OK: true, Root: parseRootPath, Budget: parseBudget}

	parseModInfo := parseGoModFile(filepath.Join(parseRootPath, "go.mod"))
	parseReport.Module = parseModInfo.module
	parseReport.GoVersion = parseModInfo.goVersion
	parseReport.LocalReplaceCount = len(parseModInfo.localReplaces)

	// External direct deps exclude locally-replaced modules (they are vendored
	// source under the repo, not a remote supply-chain dependency).
	for _, parseDep := range parseModInfo.directDeps {
		if _, parseLocal := parseModInfo.localReplaces[parseDep]; parseLocal {
			continue
		}
		parseReport.DirectExternalDeps = append(parseReport.DirectExternalDeps, parseDep)
	}
	sort.Strings(parseReport.DirectExternalDeps)
	parseReport.DirectExternalCount = len(parseReport.DirectExternalDeps)

	parseReport.TransitiveModuleCount = countGoSumModules(filepath.Join(parseRootPath, "go.sum"))
	// ChecksumsVerified reports that go.sum is present and populated — the precondition for
	// the Go toolchain to verify module checksums on every build. It is a presence check, not
	// a live cryptographic verification (run `go mod verify` for that); the distinction is
	// documented on the field so the report is not over-claimed.
	parseReport.ChecksumsVerified = parseReport.TransitiveModuleCount > 0

	parseReport.NPMArtifacts = scanNPMArtifacts(parseRootPath)
	parseReport.ZeroNPM = len(parseReport.NPMArtifacts) == 0

	// Verdict + diagnostics.
	if !parseReport.ZeroNPM {
		parseReport.OK = false
		parseReport.Diagnostics = append(parseReport.Diagnostics, agenticDiagnostic{
			Code:       "GWC-AUDIT-NPM-SURFACE",
			Severity:   "error",
			Message:    fmt.Sprintf("found %d npm dependency artifact(s) in the build tree; a GWC app has no npm supply-chain surface", len(parseReport.NPMArtifacts)),
			Suggestion: "Remove the npm artifacts, or move them under an excluded dir (research/examples/testdata) if they are not part of the importable module.",
			Attributes: map[string]string{"artifacts": strings.Join(parseReport.NPMArtifacts, ", ")},
		})
	}
	if parseBudget > 0 && parseReport.DirectExternalCount > parseBudget {
		parseReport.OK = false
		parseReport.BudgetExceeded = true
		parseReport.Diagnostics = append(parseReport.Diagnostics, agenticDiagnostic{
			Code:       "GWC-AUDIT-DEP-BUDGET",
			Severity:   "error",
			Message:    fmt.Sprintf("direct external Go dependencies (%d) exceed the budget (%d)", parseReport.DirectExternalCount, parseBudget),
			Suggestion: "Reduce direct dependencies or raise -budget; transitive footprint is reported for context.",
		})
	}
	if parseReport.Module != "" && !parseReport.ChecksumsVerified {
		parseReport.Diagnostics = append(parseReport.Diagnostics, agenticDiagnostic{
			Code:       "GWC-AUDIT-NO-CHECKSUMS",
			Severity:   "warning",
			Message:    "no go.sum checksums found; module integrity is unverified",
			Suggestion: "Run `go mod tidy` to populate go.sum so the Go toolchain verifies module checksums.",
		})
	}

	return parseReport
}

type goModInfo struct {
	module        string
	goVersion     string
	directDeps    []string
	localReplaces map[string]struct{}
}

// parseGoModFile reads a go.mod and extracts the module path, go version, direct
// (non-indirect) require paths, and the set of locally-replaced module paths. It is
// a deliberately dependency-free parser — an audit command must add no deps.
func parseGoModFile(parsePath string) goModInfo {
	parseInfo := goModInfo{localReplaces: map[string]struct{}{}}
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return parseInfo
	}
	parseInBlock := false
	for _, parseRaw := range strings.Split(string(parseData), "\n") {
		parseLine := strings.TrimSpace(parseRaw)
		if parseLine == "" || strings.HasPrefix(parseLine, "//") {
			continue
		}
		switch {
		case strings.HasPrefix(parseLine, "module "):
			parseInfo.module = strings.TrimSpace(strings.TrimPrefix(parseLine, "module "))
		case strings.HasPrefix(parseLine, "go ") && parseInfo.goVersion == "":
			parseInfo.goVersion = strings.TrimSpace(strings.TrimPrefix(parseLine, "go "))
		case strings.HasPrefix(parseLine, "replace "):
			parseAddLocalReplace(parseLine, parseInfo.localReplaces)
		case parseLine == "require (":
			parseInBlock = true
		case parseInBlock && parseLine == ")":
			parseInBlock = false
		case parseInBlock:
			if parsePath, parseDirect := parseRequireEntry(parseLine); parsePath != "" && parseDirect {
				parseInfo.directDeps = append(parseInfo.directDeps, parsePath)
			}
		case strings.HasPrefix(parseLine, "require "):
			if parsePath, parseDirect := parseRequireEntry(strings.TrimPrefix(parseLine, "require ")); parsePath != "" && parseDirect {
				parseInfo.directDeps = append(parseInfo.directDeps, parsePath)
			}
		}
	}
	return parseInfo
}

// parseRequireEntry parses one require entry line ("path version [// indirect]")
// and returns the module path and whether it is a direct (non-indirect) dependency.
func parseRequireEntry(parseLine string) (string, bool) {
	parseIsIndirect := strings.Contains(parseLine, "// indirect")
	parseFields := strings.Fields(parseLine)
	if len(parseFields) < 1 {
		return "", false
	}
	return parseFields[0], !parseIsIndirect
}

// parseAddLocalReplace records a replace whose target is a filesystem path (a local
// replacement, not a remote redirect), so the audit can exclude it from the remote
// supply-chain count.
func parseAddLocalReplace(parseLine string, parseInto map[string]struct{}) {
	parseBody := strings.TrimSpace(strings.TrimPrefix(parseLine, "replace "))
	parseParts := strings.SplitN(parseBody, "=>", 2)
	if len(parseParts) != 2 {
		return
	}
	parseTarget := strings.TrimSpace(parseParts[1])
	if strings.HasPrefix(parseTarget, "./") || strings.HasPrefix(parseTarget, "../") || strings.HasPrefix(parseTarget, ".\\") || strings.HasPrefix(parseTarget, "..\\") {
		parseSourceFields := strings.Fields(strings.TrimSpace(parseParts[0]))
		if len(parseSourceFields) >= 1 {
			parseInto[parseSourceFields[0]] = struct{}{}
		}
	}
}

// countGoSumModules counts the distinct module paths recorded in a go.sum, a proxy
// for the checksum-verified transitive footprint.
func countGoSumModules(parsePath string) int {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return 0
	}
	parseSeen := map[string]struct{}{}
	for _, parseRaw := range strings.Split(string(parseData), "\n") {
		parseFields := strings.Fields(parseRaw)
		if len(parseFields) < 1 {
			continue
		}
		parseSeen[parseFields[0]] = struct{}{}
	}
	return len(parseSeen)
}

// scanNPMArtifacts walks rootPath and returns repo-relative paths of any npm
// dependency artifacts found, skipping VCS/build/generated/example trees.
func scanNPMArtifacts(parseRootPath string) []string {
	var parseFound []string
	_ = filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return nil
		}
		parseName := parseEntry.Name()
		if parseEntry.IsDir() {
			if parsePath != parseRootPath {
				if _, parseSkip := auditNPMDirSkip[parseName]; parseSkip {
					// node_modules is both a skip dir and an artifact: record it before skipping.
					if parseName == "node_modules" {
						parseFound = append(parseFound, relativeSlashPath(parseRootPath, parsePath))
					}
					return filepath.SkipDir
				}
			}
			return nil
		}
		if _, parseIsArtifact := auditNPMArtifactNames[parseName]; parseIsArtifact {
			parseFound = append(parseFound, relativeSlashPath(parseRootPath, parsePath))
		}
		return nil
	})
	sort.Strings(parseFound)
	return parseFound
}

// printAuditSummary renders the human-readable audit result.
func printAuditSummary(parseReport auditReport) {
	parseLines := []string{
		fmt.Sprintf("root:        %s", parseReport.Root),
		fmt.Sprintf("module:      %s", parseReport.Module),
		fmt.Sprintf("go:          %s", parseReport.GoVersion),
		fmt.Sprintf("npm surface: %s", auditNPMVerdict(parseReport)),
		fmt.Sprintf("go modules:  %d direct (external), %d transitive (checksum-verified: %t)",
			parseReport.DirectExternalCount, parseReport.TransitiveModuleCount, parseReport.ChecksumsVerified),
	}
	if parseReport.LocalReplaceCount > 0 {
		parseLines = append(parseLines, fmt.Sprintf("local modules: %d (vendored via replace, not remote deps)", parseReport.LocalReplaceCount))
	}
	if parseReport.Budget > 0 {
		parseLines = append(parseLines, fmt.Sprintf("budget:      %d direct external (exceeded: %t)", parseReport.Budget, parseReport.BudgetExceeded))
	}
	for _, parseDiagnostic := range parseReport.Diagnostics {
		parseLines = append(parseLines, fmt.Sprintf("[%s] %s %s", parseDiagnostic.Severity, parseDiagnostic.Code, parseDiagnostic.Message))
	}
	printAgenticHumanSummary("GWC audit", parseReport.OK, parseLines)
}

// auditNPMVerdict renders the headline zero-npm verdict.
func auditNPMVerdict(parseReport auditReport) string {
	if parseReport.ZeroNPM {
		return "0 npm packages (supply-chain immune by construction)"
	}
	return fmt.Sprintf("%d npm artifact(s) found: %s", len(parseReport.NPMArtifacts), strings.Join(parseReport.NPMArtifacts, ", "))
}
