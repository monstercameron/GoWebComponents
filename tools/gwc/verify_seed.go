package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (parseL launcher) runVerify(parseArgs []string) error {
	parseFs := flag.NewFlagSet("verify", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "(deprecated) alias for -app; use -app")
	parseRoot := parseFs.String("root", "", "Project root used for test and build resolution")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	parseAgent := parseFs.Bool("agent", false, "Emit agent-native NDJSON verification events")
	parseSkipTests := parseFs.Bool("skip-tests", false, "Skip running go test even when *_test.go files are present")
	parseAudit := parseFs.Bool("audit", false, "Run the golden-path app audit as part of verify")
	parseAuditPolicy := parseFs.String("audit-policy", "strict", "Golden-path audit policy: strict or advisory")
	parseAuditBaseline := parseFs.String("audit-baseline", "", "Optional path to a JSON baseline file of accepted audit findings")
	parseAuditWriteBaseline := parseFs.String("audit-write-baseline", "", "Optional path to write the current audit findings as a JSON baseline")
	parseAuditMinSeverity := parseFs.String("audit-min-severity", "error", "Minimum golden-path audit severity that causes verify to fail: off, error, warning, or info")
	var parseAuditSuppressions stringListFlag
	parseFs.Var(&parseAuditSuppressions, "audit-suppress", "Audit check name to suppress; repeat or comma-separate")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	if *parseAgent && *parseJsonOutput {
		return errors.New("verify -agent cannot be combined with -json; agent mode already emits NDJSON")
	}
	var parseAgentStream *agentEventStream
	if *parseAgent {
		parseAgentStream = newAgentEventStream(os.Stdout, "verify")
	}

	parsePluginResults, parseErr2 := runLauncherPluginsForCapability("verify_check", "verify", parseArgs, parseL.repoRoot, launcherActiveEnterpriseSources)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseErr3 := enforcePluginChecks(parsePluginResults, "verify_check"); parseErr3 != nil {
		return parseErr3
	}

	buildConfig, parseErr2 := resolveBuildConfig(buildConfig{
		appPath:  firstNonEmpty(*parseApp, *parseMainPath),
		rootPath: *parseRoot,
		profile:  "ci",
	})
	if parseErr2 != nil {
		return parseErr2
	}
	if parseErr4 := enforceEnterpriseGoToolchainPolicy(buildConfig.rootPath, launcherActiveEnterpriseConfig.Policy); parseErr4 != nil {
		return parseErr4
	}
	parseVerifyCoveredLanes := []string{}
	if !*parseSkipTests {
		parseVerifyCoveredLanes = append(parseVerifyCoveredLanes, "unit")
	}
	if parseErr5 := enforceEnterpriseRequiredTestLanes(parseVerifyCoveredLanes, launcherActiveEnterpriseConfig.Policy); parseErr5 != nil {
		return fmt.Errorf("verify does not satisfy enterprise lane policy: %w", parseErr5)
	}

	parseSummary := verifySummary{
		AppPath:     buildConfig.appPath,
		ProjectRoot: buildConfig.rootPath,
		Resolution:  cloneResolutionTrace(buildConfig.resolution),
		Tests: verifyTestSummary{
			Command:        "go test",
			PackagePattern: "./...",
		},
	}
	if parseAgentStream != nil {
		if parseErr6 := parseAgentStream.emit(agentEvent{
			Event: "verify.plan",
			Phase: "plan",
			OK:    buildAgentBool(true),
			Data: map[string]any{
				"appPath":     parseSummary.AppPath,
				"projectRoot": parseSummary.ProjectRoot,
				"resolution":  parseSummary.Resolution,
			},
		}); parseErr6 != nil {
			return parseErr6
		}
	}

	if *parseSkipTests {
		parseSummary.Tests.Skipped = true
		if parseErr6 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
			Name:    "go-test",
			Status:  "skipped",
			OK:      true,
			Skipped: true,
			Evidence: map[string]any{
				"reason": "skip-tests flag",
			},
		}); parseErr6 != nil {
			return parseErr6
		}
	} else {
		hasTests, parseErr6 := projectHasGoTests(buildConfig.rootPath)
		if parseErr6 != nil {
			return parseErr6
		}
		if hasTests {
			parseOutput, parseErr7 := verifyRunGoTests(buildConfig.rootPath)
			if parseOutput != "" {
				parseSummary.Tests.Output = parseOutput
			}
			if parseErr7 != nil {
				parseSummary.OK = false
				parseDiagnostic := buildAgentDiagnosticFromError("GWC_AGENT_VERIFY_TEST_FAILED", parseErr7.Error(), "error")
				if parseErr8 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
					Name:       "go-test",
					Status:     "failed",
					OK:         false,
					Evidence:   parseSummary.Tests,
					Diagnostic: &parseDiagnostic,
				}); parseErr8 != nil {
					return parseErr8
				}
				if parseErr8 := emitAgentVerifySummary(parseAgentStream, parseSummary); parseErr8 != nil {
					return parseErr8
				}
				return parseErr7
			}
			parseSummary.Tests.Ran = true
			if parseErr8 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
				Name:     "go-test",
				Status:   "passed",
				OK:       true,
				Evidence: parseSummary.Tests,
			}); parseErr8 != nil {
				return parseErr8
			}
		} else {
			parseSummary.Tests.Skipped = true
			if parseErr7 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
				Name:    "go-test",
				Status:  "skipped",
				OK:      true,
				Skipped: true,
				Evidence: map[string]any{
					"reason": "no Go test files found",
				},
			}); parseErr7 != nil {
				return parseErr7
			}
		}
	}

	buildSummary, parseErr2 := verifyExecuteBuild(buildConfig)
	if parseErr2 != nil {
		parseSummary.OK = false
		parseDiagnostic := buildAgentDiagnosticFromError("GWC_AGENT_VERIFY_BUILD_FAILED", parseErr2.Error(), "error")
		if parseErr6 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
			Name:       "wasm-build",
			Status:     "failed",
			OK:         false,
			Diagnostic: &parseDiagnostic,
		}); parseErr6 != nil {
			return parseErr6
		}
		if parseErr6 := emitAgentVerifySummary(parseAgentStream, parseSummary); parseErr6 != nil {
			return parseErr6
		}
		return parseErr2
	}
	parseSummary.Build = buildSummary
	if parseErr6 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
		Name:     "wasm-build",
		Status:   "passed",
		OK:       true,
		Evidence: buildSummary,
	}); parseErr6 != nil {
		return parseErr6
	}
	parseSummary.OK = true
	var parseVerifyErr error
	if *parseAudit {
		parseMinSeverity, parseOk := normalizeDoctorAuditMinimumSeverity(*parseAuditMinSeverity)
		if !parseOk {
			return fmt.Errorf("unknown audit minimum severity %q", *parseAuditMinSeverity)
		}
		parseSummary.Audit = buildDoctorAuditReport(buildConfig.rootPath, doctorConfig{
			audit:              true,
			auditPolicy:        *parseAuditPolicy,
			auditBaselinePath:  *parseAuditBaseline,
			auditWriteBaseline: *parseAuditWriteBaseline,
			auditSuppressions:  parseAuditSuppressions.Values(),
			json:               *parseJsonOutput,
		})
		parseSummary.AuditMinSeverity = parseMinSeverity
		if strings.TrimSpace(*parseAuditWriteBaseline) != "" && parseSummary.Audit != nil {
			if parseErr8 := writeDoctorAuditBaseline(*parseAuditWriteBaseline, *parseSummary.Audit); parseErr8 != nil {
				return parseErr8
			}
		}
		if parseSummary.Audit != nil && doctorAuditHasFindingAtOrAbove(parseSummary.Audit.Checks, parseMinSeverity) {
			parseSummary.OK = false
			parseVerifyErr = fmt.Errorf("verify audit found %s-severity findings that need attention", parseMinSeverity)
		}
		parseAuditStatus := "passed"
		parseAuditOK := true
		var parseAuditDiagnostic *agentDiagnostic
		if parseVerifyErr != nil {
			parseAuditStatus = "failed"
			parseAuditOK = false
			parseDiagnostic := buildAgentDiagnosticFromError("GWC_AGENT_VERIFY_AUDIT_FAILED", parseVerifyErr.Error(), "error")
			parseAuditDiagnostic = &parseDiagnostic
		}
		if parseErr8 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
			Name:       "golden-path-audit",
			Status:     parseAuditStatus,
			OK:         parseAuditOK,
			Evidence:   parseSummary.Audit,
			Diagnostic: parseAuditDiagnostic,
			Metadata: map[string]string{
				"minimumSeverity": parseMinSeverity,
			},
		}); parseErr8 != nil {
			return parseErr8
		}
	} else if parseAgentStream != nil {
		if parseErr8 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
			Name:    "golden-path-audit",
			Status:  "skipped",
			OK:      true,
			Skipped: true,
			Evidence: map[string]any{
				"reason": "audit flag not set",
			},
		}); parseErr8 != nil {
			return parseErr8
		}
	}
	if parseAgentStream != nil {
		parseHydration, parseCommit := buildAgentTraceRepresentations()
		if parseErr8 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
			Name:     "hydration-diff",
			Status:   parseHydration.Status,
			OK:       true,
			Skipped:  parseHydration.Status == "skipped",
			Evidence: parseHydration,
		}); parseErr8 != nil {
			return parseErr8
		}
		if parseErr8 := emitAgentVerifyCheck(parseAgentStream, agentVerifyCheckRecord{
			Name:     "commit-trace",
			Status:   parseCommit.Status,
			OK:       true,
			Skipped:  parseCommit.Status == "skipped",
			Evidence: parseCommit,
		}); parseErr8 != nil {
			return parseErr8
		}
		if parseErr8 := emitAgentVerifySummary(parseAgentStream, parseSummary); parseErr8 != nil {
			return parseErr8
		}
	}

	if *parseAgent {
		// Agent mode has already emitted the complete NDJSON event stream.
	} else if *parseJsonOutput {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseErr9 := parseEncoder.Encode(parseSummary); parseErr9 != nil {
			return parseErr9
		}
	} else {
		printVerifySummary(parseSummary)
	}
	if !parseSummary.OK {
		if parseVerifyErr != nil {
			return parseVerifyErr
		}
		return errors.New("verify checks reported failures")
	}
	return nil
}

func (parseL launcher) runSeed(parseArgs []string) error {
	parseFs := flag.NewFlagSet("seed", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseRoot := parseFs.String("root", "", "Project root used for seed command discovery")
	parseCommandPath := parseFs.String("command", "", "Path to the seed command package directory or main.go file")
	parseDbPath := parseFs.String("db-path", "", "Override CHAT_DB_PATH for known seeders")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveSeedConfig(seedConfig{
		rootPath:    *parseRoot,
		commandPath: *parseCommandPath,
		dbPath:      *parseDbPath,
		json:        *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseErr2 := executeSeed(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printSeedSummary(parseSummary)
	return nil
}

func resolveSeedConfig(parseConfig seedConfig) (seedConfig, error) {
	parseResolved := parseConfig
	parseCwd, parseErr := seedGetwd()
	if parseErr != nil {
		return seedConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.rootPath) == "" {
		parseResolved.rootPath = parseCwd
	}
	parseResolved.rootPath, parseErr = normalizePath(parseCwd, parseResolved.rootPath)
	if parseErr != nil {
		return seedConfig{}, fmt.Errorf("resolve seed root path: %w", parseErr)
	}
	if strings.TrimSpace(parseResolved.commandPath) != "" {
		parseResolved.commandPath, parseErr = normalizeExistingPath(parseCwd, parseResolved.commandPath)
		if parseErr != nil {
			return seedConfig{}, fmt.Errorf("resolve seed command path: %w", parseErr)
		}
	} else {
		parseResolved.commandPath, parseErr = detectSeedCommandPath(parseResolved.rootPath)
		if parseErr != nil {
			return seedConfig{}, parseErr
		}
	}
	parseCommandDir, parseErr := normalizeSeedCommandDir(parseResolved.commandPath)
	if parseErr != nil {
		return seedConfig{}, parseErr
	}
	parseResolved.commandPath = parseCommandDir
	if strings.TrimSpace(parseResolved.dbPath) == "" {
		parseResolved.dbPath = defaultSeedDatabasePath(parseCommandDir)
	} else {
		parseResolved.dbPath, parseErr = normalizePath(parseCwd, parseResolved.dbPath)
		if parseErr != nil {
			return seedConfig{}, fmt.Errorf("resolve seed database path: %w", parseErr)
		}
	}
	return parseResolved, nil
}

func detectSeedCommandPath(parseRootPath string) (string, error) {
	parseCandidates := []string{
		filepath.Join(parseRootPath, "cmd", "seed"),
		filepath.Join(parseRootPath, "cmd", "seed-test-db"),
		filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db"),
	}
	for _, parseCandidate := range parseCandidates {
		parseInfo, parseErr := os.Stat(parseCandidate)
		if parseErr == nil && parseInfo.IsDir() {
			return parseCandidate, nil
		}
	}
	return "", fmt.Errorf("no seed command found under %s; pass -command to select a seed package", parseRootPath)
}

func normalizeSeedCommandDir(parseCommandPath string) (string, error) {
	parseInfo, parseErr := os.Stat(parseCommandPath)
	if parseErr != nil {
		return "", fmt.Errorf("inspect seed command path: %w", parseErr)
	}
	if parseInfo.IsDir() {
		return parseCommandPath, nil
	}
	return filepath.Dir(parseCommandPath), nil
}

// buildChatWizardRuntimeDatabasePath returns the default runtime database path for one Example 100 root.
func buildChatWizardRuntimeDatabasePath(parseExampleRoot string) string {
	parseExampleRoot = filepath.Clean(strings.TrimSpace(parseExampleRoot))
	if parseExampleRoot == "" {
		return ""
	}
	return filepath.Join(parseExampleRoot, "bin", "runtime", "chat_history.db")
}

func defaultSeedDatabasePath(parseCommandDir string) string {
	if !isChatWizardSeedCommand(parseCommandDir) {
		return ""
	}
	parseExampleRoot := filepath.Dir(filepath.Dir(parseCommandDir))
	return buildChatWizardRuntimeDatabasePath(parseExampleRoot)
}

func isChatWizardSeedCommand(parseCommandDir string) bool {
	parseCommandDir = filepath.Clean(parseCommandDir)
	parseSuffix := filepath.Join("examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	return strings.HasSuffix(parseCommandDir, parseSuffix)
}

func executeSeed(parseConfig seedConfig) (seedSummary, error) {
	parseEnv := buildNativeGoEnv()
	if strings.TrimSpace(parseConfig.dbPath) != "" {
		parseEnv = replaceEnvVar(parseEnv, "CHAT_DB_PATH", parseConfig.dbPath)
	}
	parseOutput, parseErr := launcherRunCommand("go", []string{"run", "."}, parseConfig.commandPath, parseEnv)
	if parseErr != nil {
		return seedSummary{}, parseErr
	}
	parseSummary := seedSummary{
		OK:           true,
		ProjectRoot:  parseConfig.rootPath,
		CommandPath:  parseConfig.commandPath,
		DatabasePath: parseConfig.dbPath,
		Output:       parseOutput,
	}
	if isChatWizardSeedCommand(parseConfig.commandPath) {
		parseSummary.Credentials = []seedCredentialRecord{
			{Email: "customer@email.com", Password: "password", Role: "customer"},
			{Email: "admin@email.com", Password: "password", Role: "admin"},
		}
	}
	return parseSummary, nil
}

func replaceEnvVar(parseEnv []string, parseKey string, parseValue string) []string {
	parsePrefix := parseKey + "="
	isParseReplaced := false
	parseUpdated := make([]string, 0, len(parseEnv)+1)
	for _, parseEntry := range parseEnv {
		if strings.HasPrefix(parseEntry, parsePrefix) {
			if !isParseReplaced {
				parseUpdated = append(parseUpdated, parsePrefix+parseValue)
				isParseReplaced = true
			}
			continue
		}
		parseUpdated = append(parseUpdated, parseEntry)
	}
	if !isParseReplaced {
		parseUpdated = append(parseUpdated, parsePrefix+parseValue)
	}
	return parseUpdated
}
