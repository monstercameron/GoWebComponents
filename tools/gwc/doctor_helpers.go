package main

import (
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

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

func collectGoldenPathGoFiles(parseRoot string) ([]doctorGoFileRecord, error) {
	parseRecords := []doctorGoFileRecord{}
	parseErr := filepath.WalkDir(parseRoot, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipDoctorAuditDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(parseEntry.Name()) != ".go" {
			return nil
		}
		parseContentBytes, parseErr2 := os.ReadFile(parsePath)
		if parseErr2 != nil {
			return parseErr2
		}
		parseContent := string(parseContentBytes)
		parseFile, parseErr2 := parser.ParseFile(token.NewFileSet(), parsePath, parseContent, parser.ImportsOnly|parser.ParseComments)
		if parseErr2 != nil {
			return parseErr2
		}
		parseImports := make([]string, 0, len(parseFile.Imports))
		for _, parseSpec := range parseFile.Imports {
			parseImports = append(parseImports, strings.Trim(parseSpec.Path.Value, `"`))
		}
		parseRelPath, parseErr2 := filepath.Rel(parseRoot, parsePath)
		if parseErr2 != nil {
			parseRelPath = parsePath
		}
		parseRecords = append(parseRecords, doctorGoFileRecord{
			RelPath:   filepath.ToSlash(parseRelPath),
			Content:   parseContent,
			Imports:   parseImports,
			Client:    isDoctorAuditClientFile(filepath.ToSlash(parsePath), parseContent),
			SizeBytes: int64(len(parseContentBytes)),
		})
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}
	return parseRecords, nil
}

func collectDoctorHTMLFiles(parseRoot string) ([]string, error) {
	parseFiles := []string{}
	parseErr := filepath.WalkDir(parseRoot, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipDoctorAuditDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(parseEntry.Name()) != ".html" {
			return nil
		}
		parseRelPath, parseErr2 := filepath.Rel(parseRoot, parsePath)
		if parseErr2 != nil {
			parseRelPath = parsePath
		}
		parseFiles = append(parseFiles, filepath.ToSlash(parseRelPath))
		return nil
	})
	if parseErr != nil {
		return nil, parseErr
	}
	sort.Strings(parseFiles)
	return parseFiles, nil
}

func collectDoctorRuntimeArtifacts(parseRoot string) ([]doctorStartupEvidence, []doctorRuntimeArtifact, error) {
	parseStartupReports := []doctorStartupEvidence{}
	parseWasmFiles := []doctorRuntimeArtifact{}
	parseErr := filepath.WalkDir(parseRoot, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipDoctorRuntimeDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		parseRelPath, parseErr2 := filepath.Rel(parseRoot, parsePath)
		if parseErr2 != nil {
			parseRelPath = parsePath
		}
		parseRelPath = filepath.ToSlash(parseRelPath)
		switch {
		case strings.EqualFold(parseEntry.Name(), "wasm-startup-report.json"):
			parseReport, parseErr3 := readDoctorStartupEvidence(parsePath, parseRelPath)
			if parseErr3 != nil {
				return parseErr3
			}
			parseStartupReports = append(parseStartupReports, parseReport)
		case strings.EqualFold(filepath.Ext(parseEntry.Name()), ".wasm"):
			parseInfo, parseErr4 := parseEntry.Info()
			if parseErr4 != nil {
				return parseErr4
			}
			parseWasmFiles = append(parseWasmFiles, doctorRuntimeArtifact{RelPath: parseRelPath, SizeBytes: parseInfo.Size()})
		}
		return nil
	})
	if parseErr != nil {
		return nil, nil, parseErr
	}
	sort.Slice(parseStartupReports, func(parseI int, parseJ int) bool {
		return parseStartupReports[parseI].RelPath < parseStartupReports[parseJ].RelPath
	})
	sort.Slice(parseWasmFiles, func(parseI2 int, parseJ2 int) bool {
		if parseWasmFiles[parseI2].SizeBytes != parseWasmFiles[parseJ2].SizeBytes {
			return parseWasmFiles[parseI2].SizeBytes > parseWasmFiles[parseJ2].SizeBytes
		}
		return parseWasmFiles[parseI2].RelPath < parseWasmFiles[parseJ2].RelPath
	})
	return parseStartupReports, parseWasmFiles, nil
}

func shouldSkipDoctorRuntimeDir(parseName string) bool {
	switch strings.TrimSpace(parseName) {
	case ".git", "node_modules", "vendor", "dist", "tmp":
		return true
	default:
		return false
	}
}

func readDoctorStartupEvidence(parsePath string, parseRelPath string) (doctorStartupEvidence, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return doctorStartupEvidence{}, parseErr
	}
	var parsePayload map[string]interface{}
	if parseErr2 := json.Unmarshal(parseContent, &parsePayload); parseErr2 != nil {
		return doctorStartupEvidence{}, parseErr2
	}
	parseEvidence := doctorStartupEvidence{RelPath: parseRelPath}
	if parseStartup, parseOk := parsePayload["startup"].(map[string]interface{}); parseOk {
		if parseReadyMs, parseOk2 := parseStartup["readyMs"].(float64); parseOk2 {
			parseEvidence.ReadyMs = &parseReadyMs
		}
		if parseInteractionMs, parseOk3 := parseStartup["interactionMs"].(float64); parseOk3 {
			parseEvidence.InteractionMs = &parseInteractionMs
		}
	}
	return parseEvidence, nil
}

func shouldSkipDoctorAuditDir(parseName string) bool {
	switch strings.TrimSpace(parseName) {
	case ".git", "node_modules", "vendor", "bin", "dist", "tmp":
		return true
	default:
		return false
	}
}

func isDoctorAuditClientFile(parsePath string, parseContent string) bool {
	if strings.Contains(parseContent, "//go:build js && wasm") {
		return true
	}
	return strings.Contains(parsePath, "/client/")
}

func summarizeDoctorViolations(parseViolations []string, parseLimit int) string {
	if len(parseViolations) == 0 {
		return ""
	}
	if parseLimit <= 0 || len(parseViolations) <= parseLimit {
		return strings.Join(parseViolations, " | ")
	}
	return fmt.Sprintf("%s | +%d more", strings.Join(parseViolations[:parseLimit], " | "), len(parseViolations)-parseLimit)
}

func formatBytesBinary(parseSize int64) string {
	if parseSize < 1024 {
		return fmt.Sprintf("%d B", parseSize)
	}
	parseKib := float64(parseSize) / 1024
	if parseKib < 1024 {
		return fmt.Sprintf("%.1f KiB", parseKib)
	}
	return fmt.Sprintf("%.1f MiB", parseKib/1024)
}

func loadDoctorAuditBaseline(parsePath string) (doctorAuditBaseline, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return doctorAuditBaseline{}, fmt.Errorf("read audit baseline: %w", parseErr)
	}
	var parseBaseline doctorAuditBaseline
	if parseErr2 := json.Unmarshal(parseContent, &parseBaseline); parseErr2 != nil {
		return doctorAuditBaseline{}, fmt.Errorf("parse audit baseline: %w", parseErr2)
	}
	return parseBaseline, nil
}

func writeDoctorAuditBaseline(parsePath string, parseAudit doctorAuditReport) error {
	parseResolvedPath := parsePath
	if parseCwd, parseErr := doctorGetwd(); parseErr == nil {
		if parseNormalized, parseNormalizeErr := normalizePath(parseCwd, parsePath); parseNormalizeErr == nil && strings.TrimSpace(parseNormalized) != "" {
			parseResolvedPath = parseNormalized
		}
	}
	if parseDir := filepath.Dir(parseResolvedPath); strings.TrimSpace(parseDir) != "" && parseDir != "." {
		if parseErr2 := os.MkdirAll(parseDir, 0755); parseErr2 != nil {
			return fmt.Errorf("create audit baseline directory: %w", parseErr2)
		}
	}
	parseBaseline := doctorAuditBaseline{
		Mode:        parseAudit.Mode,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Checks:      make([]doctorAuditBaselineCheck, 0, len(parseAudit.Checks)),
	}
	for _, parseCheck := range parseAudit.Checks {
		if parseCheck.Status == "pass" {
			continue
		}
		parseBaseline.Checks = append(parseBaseline.Checks, doctorAuditBaselineCheck{
			Name:    parseCheck.Name,
			Status:  parseCheck.Status,
			Summary: parseCheck.Summary,
		})
	}
	parseContent, parseErr3 := json.MarshalIndent(parseBaseline, "", "  ")
	if parseErr3 != nil {
		return fmt.Errorf("marshal audit baseline: %w", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseResolvedPath, append(parseContent, '\n'), 0644); parseErr4 != nil {
		return fmt.Errorf("write audit baseline: %w", parseErr4)
	}
	return nil
}

func buildDoctorToolCheck(parseCommand string, parseName string, parseVersionArg string, parseHint string) doctorCheck {
	parsePath, parseErr := doctorLookPath(parseCommand)
	if parseErr != nil {
		return doctorCheck{Name: parseName, Status: "fail", Summary: fmt.Sprintf("%s was not found on PATH.", parseCommand), Hint: parseHint}
	}
	parseOutput, parseErr := doctorCommandOutput(parseCommand, parseVersionArg)
	if parseErr != nil {
		parseSummary := strings.TrimSpace(parseOutput)
		if parseSummary == "" {
			parseSummary = parseErr.Error()
		}
		return doctorCheck{Name: parseName, Status: "fail", Summary: fmt.Sprintf("%s is on PATH at %s but did not report a version: %s", parseCommand, parsePath, parseSummary), Hint: parseHint}
	}
	return doctorCheck{Name: parseName, Status: "pass", Summary: fmt.Sprintf("%s (%s)", parseOutput, parsePath)}
}

func buildDoctorWasmExecCheck() doctorCheck {
	parseWasmExecPath, parseErr := doctorResolveWasmExec()
	if parseErr != nil {
		return doctorCheck{Name: "wasm_exec.js", Status: "fail", Summary: "Matching wasm_exec.js could not be resolved from the active Go toolchain.", Hint: "Use the same Go toolchain for both the wasm binary and wasm_exec.js."}
	}
	return doctorCheck{Name: "wasm_exec.js", Status: "pass", Summary: fmt.Sprintf("Resolved matching runtime asset at %s", parseWasmExecPath)}
}

func buildDoctorPlaywrightCheck(parseRepoRoot string) doctorCheck {
	parseWorkspace, parseErr := resolveBrowserWorkspace(parseRepoRoot, parseRepoRoot)
	if parseErr != nil {
		return doctorCheck{Name: "Browser tests", Status: "fail", Summary: parseErr.Error(), Hint: "Fix the browserWorkspace override or remove it so launcher defaults can be used."}
	}
	if strings.TrimSpace(parseWorkspace) == "" {
		return doctorCheck{
			Name:    "Browser tests",
			Status:  "warn",
			Summary: "No Playwright-Go browser suite was detected.",
			Hint:    "Create test/playwrightgo (or playwrightgo) and run `go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v`.",
		}
	}
	parsePackagePattern, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(parseWorkspace)
	if hasPlaywrightGoSuite {
		return doctorCheck{
			Name:    "Browser tests",
			Status:  "pass",
			Summary: fmt.Sprintf("Playwright-Go browser suite is available in %s (%s).", parseWorkspace, parsePackagePattern),
		}
	}
	return doctorCheck{
		Name:    "Browser tests",
		Status:  "warn",
		Summary: fmt.Sprintf("Browser workspace %s does not include a Playwright-Go suite.", parseWorkspace),
		Hint:    fmt.Sprintf("Run `go test -tags playwrightgo %s -run TestMainSuite -v` from %s.", firstNonEmpty(parsePackagePattern, "./playwrightgo"), parseWorkspace),
	}
}

func buildDoctorMetadataCheck(parseCwd string) doctorCheck {
	if strings.TrimSpace(parseCwd) == "" {
		return doctorCheck{Name: "Scaffold metadata", Status: "warn", Summary: "The current working directory could not be resolved.", Hint: "Run doctor from the target app directory to inspect scaffold metadata."}
	}
	parseMetadata, parseOk, parseErr := loadScaffoldMetadata(parseCwd)
	if parseErr != nil {
		return doctorCheck{Name: "Scaffold metadata", Status: "fail", Summary: parseErr.Error(), Hint: "Fix or regenerate the scaffold metadata file."}
	}
	if !parseOk {
		return doctorCheck{Name: "Scaffold metadata", Status: "warn", Summary: "No gwc-start.json metadata file was found in the current directory.", Hint: "Generated starters should carry scaffold metadata; hand-built apps can ignore this warning for now."}
	}
	parseProjectName := firstNonEmpty(parseMetadata.ProjectName, "<unnamed>")
	parseModulePath := firstNonEmpty(parseMetadata.ModulePath, "<missing modulePath>")
	return doctorCheck{Name: "Scaffold metadata", Status: "pass", Summary: fmt.Sprintf("Detected starter metadata for %s (%s)", parseProjectName, parseModulePath)}
}

func buildDoctorProjectDetectionCheck(parseCwd string) doctorCheck {
	if strings.TrimSpace(parseCwd) == "" {
		return doctorCheck{Name: "Project detection", Status: "warn", Summary: "The current working directory could not be resolved.", Hint: "Run doctor from an app root to preview gwc dev detection."}
	}
	parseAppPath, parseAppErr := detectAppPath(parseCwd)
	parseHtmlPath := detectHTMLPath(parseCwd)
	if parseAppErr != nil {
		return doctorCheck{Name: "Project detection", Status: "warn", Summary: "gwc dev would not auto-detect an app entrypoint in the current directory.", Hint: "Pass -app explicitly or keep main.go or cmd/web/main.go at the documented locations."}
	}
	parseParts := []string{fmt.Sprintf("App entrypoint: %s", parseAppPath)}
	if strings.TrimSpace(parseHtmlPath) != "" {
		parseParts = append(parseParts, fmt.Sprintf("HTML shell: %s", parseHtmlPath))
	}
	return doctorCheck{Name: "Project detection", Status: "pass", Summary: strings.Join(parseParts, " | ")}
}

func buildDoctorPortCheck(parseHost string, parsePort string) doctorCheck {
	parseAddress := joinHostPort(parseHost, parsePort)
	parseListener, parseErr := doctorListen("tcp", parseAddress)
	if parseErr != nil {
		return doctorCheck{Name: "Port availability", Status: "fail", Summary: fmt.Sprintf("Could not bind %s: %v", parseAddress, parseErr), Hint: "Stop the conflicting process or choose a different port before running gwc dev or gwc examples."}
	}
	_ = parseListener.Close()
	return doctorCheck{Name: "Port availability", Status: "pass", Summary: fmt.Sprintf("Port %s is available for local launcher commands.", parseAddress)}
}

func printDoctorReport(parseReport doctorReport) {
	parseStatus := "PASS"
	if !parseReport.OK {
		parseStatus = "FAIL"
	}
	fmt.Printf("GWC doctor: %s\n", parseStatus)
	if strings.TrimSpace(parseReport.CWD) != "" {
		fmt.Printf("  cwd: %s\n", parseReport.CWD)
	}
	for _, parseCheck := range parseReport.Checks {
		parseLabel := strings.ToUpper(parseCheck.Status)
		fmt.Printf("  [%s] %s: %s\n", parseLabel, parseCheck.Name, parseCheck.Summary)
		if strings.TrimSpace(parseCheck.Hint) != "" && parseCheck.Status != "pass" {
			fmt.Printf("         hint: %s\n", parseCheck.Hint)
		}
	}
	if parseReport.Audit != nil {
		parseAuditStatus := "PASS"
		if !parseReport.Audit.OK {
			parseAuditStatus = "FAIL"
		}
		fmt.Printf("  audit[%s]: %s", parseReport.Audit.Mode, parseAuditStatus)
		if parseReport.Audit.Policy != "" {
			fmt.Printf(" (policy: %s)", parseReport.Audit.Policy)
		}
		fmt.Println()
		if strings.TrimSpace(parseReport.Audit.BaselinePath) != "" {
			fmt.Printf("    baseline: %s\n", parseReport.Audit.BaselinePath)
		}
		if len(parseReport.Audit.Suppressed) > 0 {
			fmt.Printf("    suppressed: %s\n", strings.Join(parseReport.Audit.Suppressed, ", "))
		}
		for _, parseCheck2 := range parseReport.Audit.Checks {
			parseLabel2 := strings.ToUpper(parseCheck2.Status)
			fmt.Printf("    [%s] %s: %s\n", parseLabel2, parseCheck2.Name, parseCheck2.Summary)
			if strings.TrimSpace(parseCheck2.Hint) != "" && parseCheck2.Status != "pass" {
				fmt.Printf("           hint: %s\n", parseCheck2.Hint)
			}
		}
	}
	printResolutionTrace(parseReport.Resolution, []string{"app", "root", "html", "host", "port"}, "  ")
}
