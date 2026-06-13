package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

func (parseL launcher) runTest(parseArgs []string) error {
	parseFs := flag.NewFlagSet("test", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "(deprecated) alias for -app; use -app")
	parseRoot := parseFs.String("root", "", "Project root used for test lane resolution")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	parseWatch := parseFs.Bool("watch", false, "Re-run selected test lanes when Go files change")
	parseWatchOnce := parseFs.Bool("once", false, "With -watch, run one watched test pass and exit")
	parseWatchDebounce := parseFs.Duration("debounce", 500*time.Millisecond, "With -watch, polling debounce interval")
	var parseLaneFlags stringListFlag
	parseFs.Var(&parseLaneFlags, "lane", "Test lane to run; repeat or comma-separate: unit, race, wasm, hydration, browser, perf, i18n, agent, agent-browser, release, all")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveTestConfig(testConfig{
		appPath:  firstNonEmpty(*parseApp, *parseMainPath),
		rootPath: *parseRoot,
		lanes:    parseLaneFlags.Values(),
		json:     *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}
	if parseErr3 := enforceEnterpriseGoToolchainPolicy(parseConfig.rootPath, launcherActiveEnterpriseConfig.Policy); parseErr3 != nil {
		return parseErr3
	}
	if parseErr4 := enforceEnterpriseRequiredTestLanes(parseConfig.lanes, launcherActiveEnterpriseConfig.Policy); parseErr4 != nil {
		return parseErr4
	}
	if *parseWatch {
		parseWatchArgs := []string{"-root", parseConfig.rootPath, "-debounce", parseWatchDebounce.String()}
		if parseConfig.appPath != "" {
			parseWatchArgs = append(parseWatchArgs, "-app", parseConfig.appPath)
		}
		for _, parseLane := range parseConfig.lanes {
			parseWatchArgs = append(parseWatchArgs, "-lane", parseLane)
		}
		if parseConfig.json {
			parseWatchArgs = append(parseWatchArgs, "-json")
		}
		if *parseWatchOnce {
			parseWatchArgs = append(parseWatchArgs, "-once")
		}
		return runWatchCommand(parseL, parseWatchArgs)
	}

	parseSummary, parseErr2 := parseL.executeTest(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printTestSummary(parseSummary)
	return nil
}

func resolveTestConfig(parseConfig testConfig) (testConfig, error) {
	parseResolved := parseConfig
	parseResolved.resolution = cloneResolutionTrace(parseConfig.resolution)
	parseCwd, parseErr := testGetwd()
	if parseErr != nil {
		return testConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.rootPath) == "" {
		parseResolved.rootPath = parseCwd
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "convention fallback")
	} else {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "explicit flag")
	}
	parseResolved.rootPath, parseErr = normalizePath(parseCwd, parseResolved.rootPath)
	if parseErr != nil {
		return testConfig{}, fmt.Errorf("resolve test root path: %w", parseErr)
	}
	if strings.TrimSpace(parseResolved.appPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "explicit flag")
		parseResolved.appPath, parseErr = normalizeExistingPath(parseCwd, parseResolved.appPath)
		if parseErr != nil {
			return testConfig{}, fmt.Errorf("resolve app path: %w", parseErr)
		}
	}
	parseResolved.lanes, parseErr = normalizeTestLanes(parseResolved.lanes)
	if parseErr != nil {
		return testConfig{}, parseErr
	}
	return parseResolved, nil
}

func (parseL launcher) executeTest(parseConfig testConfig) (testSummary, error) {
	parseSummary := testSummary{
		OK:            true,
		AppPath:       parseConfig.appPath,
		ProjectRoot:   parseConfig.rootPath,
		SelectedLanes: append([]string(nil), parseConfig.lanes...),
		Lanes:         make([]testLaneSummary, 0, len(parseConfig.lanes)),
		Resolution:    cloneResolutionTrace(parseConfig.resolution),
	}
	for _, parseLane := range parseConfig.lanes {
		parseLaneSummary, parseErr := parseL.executeTestLane(parseConfig, parseLane)
		if parseErr != nil {
			return testSummary{}, parseErr
		}
		parseSummary.Lanes = append(parseSummary.Lanes, parseLaneSummary)
		if !parseLaneSummary.OK && !parseLaneSummary.Skipped {
			parseSummary.OK = false
		}
	}
	return parseSummary, nil
}

func (parseL launcher) executeTestLane(parseConfig testConfig, parseLane string) (testLaneSummary, error) {
	switch parseLane {
	case "unit":
		return parseL.runUnitTestLane(parseConfig.rootPath)
	case "race":
		return parseL.runRaceTestLane(parseConfig.rootPath)
	case "wasm":
		return parseL.runWasmTestLane(parseConfig.rootPath, false)
	case "hydration":
		return parseL.runWasmTestLane(parseConfig.rootPath, true)
	case "browser":
		return parseL.runBrowserTestLane(parseConfig.rootPath)
	case "perf":
		return parseL.runPerfBudgetTestLane(parseConfig.rootPath)
	case "i18n":
		return parseL.runI18nCompletenessTestLane(parseConfig.rootPath)
	case "agent":
		return parseL.runAgentBridgeTestLane(parseConfig.rootPath)
	case "agent-browser":
		return parseL.runAgentBridgeHeadlessTestLane(parseConfig.rootPath)
	case "release":
		return parseL.runReleaseTestLane(parseConfig)
	default:
		return testLaneSummary{}, fmt.Errorf("unknown test lane %q", parseLane)
	}
}

var testRaceDetectorSupported = func() bool {
	return isGoRaceDetectorSupported(runtime.GOOS, runtime.GOARCH)
}

func (parseL launcher) runUnitTestLane(parseRootPath string) (testLaneSummary, error) {
	parseOutputs := []string{}
	parseOutput, parseErr := launcherRunCommand("go", []string{"test", "./..."}, parseRootPath, buildNativeGoEnv())
	if parseOutput != "" {
		parseOutputs = append(parseOutputs, parseOutput)
	}
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	parseSummary := testLaneSummary{
		Name:           "unit",
		OK:             true,
		Command:        "go test ./...",
		PackagePattern: "./...",
		Workspace:      parseRootPath,
		Summary:        "Native Go tests passed.",
	}
	if parseRootPath == parseL.repoRoot {
		parseNestedRoot, parseErr2 := resolveLauncherLivereloadWorkspace(parseL.repoRoot, parseRootPath)
		if parseErr2 != nil {
			return testLaneSummary{}, parseErr2
		}
		if fileExists(filepath.Join(parseNestedRoot, "go.mod")) {
			parseNestedOutput, parseNestedErr := launcherRunCommand("go", []string{"test", "./..."}, parseNestedRoot, buildNativeGoEnv())
			if parseNestedOutput != "" {
				parseOutputs = append(parseOutputs, parseNestedOutput)
			}
			if parseNestedErr != nil {
				return testLaneSummary{}, parseNestedErr
			}
			parseSummary.Summary = "Native Go tests passed, including the nested livereload workspace."
		}
	}
	if len(parseOutputs) > 0 {
		parseSummary.Output = strings.Join(parseOutputs, "\n")
	}
	return parseSummary, nil
}

func (parseL launcher) runRaceTestLane(parseRootPath string) (testLaneSummary, error) {
	if !testRaceDetectorSupported() {
		return testLaneSummary{
			Name:      "race",
			OK:        true,
			Skipped:   true,
			Workspace: parseRootPath,
			Summary:   fmt.Sprintf("Go race detector is not supported on %s/%s; run this lane on linux/amd64 or another supported host.", runtime.GOOS, runtime.GOARCH),
		}, nil
	}
	parseArgs := []string{"test", "-race", "./..."}
	parseOutput, parseErr := launcherRunCommand("go", parseArgs, parseRootPath, buildNativeGoEnv())
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	return testLaneSummary{
		Name:           "race",
		OK:             true,
		Command:        "go " + strings.Join(parseArgs, " "),
		PackagePattern: "./...",
		Workspace:      parseRootPath,
		Output:         parseOutput,
		Summary:        "Native Go race detector tests passed.",
	}, nil
}

func (parseL launcher) runWasmTestLane(parseRootPath string, isHydrationOnly bool) (testLaneSummary, error) {
	parsePackages, parseErr := collectWasmTestPackages(parseRootPath, isHydrationOnly)
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	parseLaneName := "wasm"
	parseSummaryText := "Discovered js/wasm Go test packages passed."
	if isHydrationOnly {
		parseLaneName = "hydration"
		parseSummaryText = "Focused hydration js/wasm test packages passed."
	}
	if len(parsePackages) == 0 {
		return testLaneSummary{
			Name:      parseLaneName,
			OK:        true,
			Skipped:   true,
			Workspace: parseRootPath,
			Summary:   "No matching js/wasm test packages were found.",
		}, nil
	}
	parseWasmExec, parseErr := testResolveWasmExec(parseL.repoRoot)
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	parseArgs := append([]string{"test", "-exec", parseWasmExec}, parsePackages...)
	parseOutput, parseErr := launcherRunCommand("go", parseArgs, parseRootPath, buildWasmGoEnv())
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	return testLaneSummary{
		Name:           parseLaneName,
		OK:             true,
		Command:        "go " + strings.Join(parseArgs, " "),
		PackagePattern: strings.Join(parsePackages, " "),
		Packages:       parsePackages,
		Workspace:      parseRootPath,
		Output:         parseOutput,
		Summary:        parseSummaryText,
	}, nil
}

func (parseL launcher) runBrowserTestLane(parseRootPath string) (testLaneSummary, error) {
	parseWorkspace, parseErr := resolveBrowserWorkspace(parseL.repoRoot, parseRootPath)
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	if parseWorkspace == "" {
		return testLaneSummary{
			Name:      "browser",
			OK:        true,
			Skipped:   true,
			Workspace: parseRootPath,
			Summary:   "No browser test workspace was found for the requested root.",
		}, nil
	}
	parsePackagePattern, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(parseWorkspace)
	if !hasPlaywrightGoSuite {
		return testLaneSummary{
			Name:      "browser",
			OK:        true,
			Skipped:   true,
			Workspace: parseWorkspace,
			Summary:   "No Playwright-Go test package was found in the browser workspace.",
		}, nil
	}
	parseArgs := []string{"test", "-tags", "playwrightgo", parsePackagePattern, "-run", "TestMainSuite", "-v"}
	parseOutput, parseErr := launcherRunCommand("go", parseArgs, parseWorkspace, buildBrowserTestEnv())
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	return testLaneSummary{
		Name:           "browser",
		OK:             true,
		Command:        "go " + strings.Join(parseArgs, " "),
		PackagePattern: parsePackagePattern,
		Workspace:      parseWorkspace,
		Output:         parseOutput,
		Summary:        "Browser Playwright-Go suite passed.",
	}, nil
}

func (parseL launcher) runPerfBudgetTestLane(parseRootPath string) (testLaneSummary, error) {
	parseWorkspace, parseErr := resolveBrowserWorkspace(parseL.repoRoot, parseRootPath)
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	if parseWorkspace == "" {
		return testLaneSummary{
			Name:      "perf",
			OK:        true,
			Skipped:   true,
			Workspace: parseRootPath,
			Summary:   "No browser test workspace was found for the requested root.",
		}, nil
	}
	parsePackagePattern, hasPerfSuite := resolvePerfBudgetTestPackagePattern(parseWorkspace)
	if !hasPerfSuite {
		return testLaneSummary{
			Name:      "perf",
			OK:        true,
			Skipped:   true,
			Workspace: parseWorkspace,
			Summary:   "No Playwright-Go perf budget package was found in the browser workspace.",
		}, nil
	}
	parseArgs := []string{"test", "-tags", "playwrightgo", parsePackagePattern, "-run", "TestPerfBudget", "-v"}
	parseOutput, parseErr := launcherRunCommand("go", parseArgs, parseWorkspace, buildBrowserTestEnv())
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	return testLaneSummary{
		Name:           "perf",
		OK:             true,
		Command:        "go " + strings.Join(parseArgs, " "),
		PackagePattern: parsePackagePattern,
		Workspace:      parseWorkspace,
		Output:         parseOutput,
		Summary:        "Performance budget tests passed; budget changes require explicit testdata review.",
	}, nil
}

func (parseL launcher) runI18nCompletenessTestLane(parseRootPath string) (testLaneSummary, error) {
	parsePackagePath := filepath.Join(parseRootPath, "i18n", "extract")
	parseInfo, parseStatErr := os.Stat(parsePackagePath)
	if parseStatErr != nil || !parseInfo.IsDir() {
		return testLaneSummary{
			Name:      "i18n",
			OK:        true,
			Skipped:   true,
			Workspace: parseRootPath,
			Summary:   "No i18n/extract package was found for the requested root.",
		}, nil
	}
	parseArgs := []string{"test", "./i18n/extract"}
	parseOutput, parseErr := launcherRunCommand("go", parseArgs, parseRootPath, buildNativeGoEnv())
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	return testLaneSummary{
		Name:           "i18n",
		OK:             true,
		Command:        "go " + strings.Join(parseArgs, " "),
		PackagePattern: "./i18n/extract",
		Workspace:      parseRootPath,
		Output:         parseOutput,
		Summary:        "i18n extraction and locale completeness checks passed.",
	}, nil
}

func (parseL launcher) runAgentBridgeTestLane(parseRootPath string) (testLaneSummary, error) {
	parseOutputs := []string{}
	parseRootArgs := []string{"test", "./agentbridge", "./internal/runtime"}
	parseRootOutput, parseRootErr := launcherRunCommand("go", parseRootArgs, parseRootPath, buildNativeGoEnv())
	if parseRootOutput != "" {
		parseOutputs = append(parseOutputs, parseRootOutput)
	}
	if parseRootErr != nil {
		return testLaneSummary{}, parseRootErr
	}
	for _, parseSubmodule := range []string{filepath.Join("tools", "agenthub"), filepath.Join("tools", "livereload")} {
		parseWorkspace := filepath.Join(parseRootPath, parseSubmodule)
		if !fileExists(filepath.Join(parseWorkspace, "go.mod")) {
			continue
		}
		parseOutput, parseErr := launcherRunCommand("go", []string{"test", "./..."}, parseWorkspace, buildNativeGoEnv())
		if parseOutput != "" {
			parseOutputs = append(parseOutputs, parseOutput)
		}
		if parseErr != nil {
			return testLaneSummary{}, parseErr
		}
	}
	return testLaneSummary{
		Name:           "agent",
		OK:             true,
		Command:        "go test ./agentbridge ./internal/runtime; (cd tools/agenthub && go test ./...); (cd tools/livereload && go test ./...)",
		PackagePattern: "./agentbridge ./internal/runtime tools/agenthub/... tools/livereload/...",
		Workspace:      parseRootPath,
		Output:         strings.Join(parseOutputs, "\n"),
		Summary:        "Agent bridge, hub, and livereload integration tests passed.",
	}, nil
}

func (parseL launcher) runAgentBridgeHeadlessTestLane(parseRootPath string) (testLaneSummary, error) {
	parseWorkspace, parseErr := resolveBrowserWorkspace(parseL.repoRoot, parseRootPath)
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	if parseWorkspace == "" {
		return testLaneSummary{
			Name:      "agent-browser",
			OK:        true,
			Skipped:   true,
			Workspace: parseRootPath,
			Summary:   "No browser test workspace was found for the requested root.",
		}, nil
	}
	parsePackagePattern, hasDogfoodSuite := resolveAgentBridgeHeadlessTestPackagePattern(parseWorkspace)
	if !hasDogfoodSuite {
		return testLaneSummary{
			Name:      "agent-browser",
			OK:        true,
			Skipped:   true,
			Workspace: parseWorkspace,
			Summary:   "No ai-chat-wizard agent bridge dogfood Playwright-Go test was found.",
		}, nil
	}
	parseRunPattern := "TestExample100AgentBridgeDogfood|TestAgentBridgeDogfood|TestAgentBridgeHeadless"
	parseArgs := []string{"test", "-tags", "playwrightgo", "-run", parseRunPattern, "-count=1", "-timeout=5m", "-v", parsePackagePattern}
	parseOutput, parseErr := launcherRunCommand("go", parseArgs, parseWorkspace, buildBrowserTestEnv())
	if parseErr != nil {
		parseCommandText := "go " + strings.Join(parseArgs, " ")
		if strings.TrimSpace(parseOutput) == "" {
			return testLaneSummary{}, fmt.Errorf("go test failed for agent-browser lane (%s): %w", parseCommandText, parseErr)
		}
		return testLaneSummary{}, fmt.Errorf("go test failed for agent-browser lane (%s): %w\n%s", parseCommandText, parseErr, strings.TrimRight(parseOutput, "\r\n"))
	}
	return testLaneSummary{
		Name:           "agent-browser",
		OK:             true,
		Command:        "go " + strings.Join(parseArgs, " "),
		PackagePattern: parsePackagePattern,
		Workspace:      parseWorkspace,
		Output:         parseOutput,
		Summary:        "Headless agent bridge dogfood Playwright-Go test passed.",
	}, nil
}

func (parseL launcher) runReleaseTestLane(parseConfig testConfig) (testLaneSummary, error) {
	parseReleaseOutDir, parseErr := createLauncherTempDir(parseConfig.rootPath, "gwc-test-release-")
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	parseReleaseConfig, parseErr := resolveReleaseConfig(releaseConfig{
		appPath:  parseConfig.appPath,
		rootPath: parseConfig.rootPath,
		outDir:   parseReleaseOutDir,
		profile:  "release",
	})
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	if parseErr2 := enforceEnterpriseReleasePolicy(parseReleaseConfig, launcherActiveEnterpriseConfig.Policy); parseErr2 != nil {
		return testLaneSummary{}, parseErr2
	}
	parseReleaseSummary, parseErr := testExecuteRelease(parseReleaseConfig)
	if parseErr != nil {
		return testLaneSummary{}, parseErr
	}
	return testLaneSummary{
		Name:         "release",
		OK:           true,
		Summary:      "Release smoke build passed.",
		OutDir:       parseReleaseSummary.OutDir,
		ManifestPath: parseReleaseSummary.ManifestPath,
		Workspace:    parseReleaseSummary.ProjectRoot,
	}, nil
}

type stringListFlag struct {
	values []string
}

func (parseF *stringListFlag) String() string {
	return strings.Join(parseF.values, ",")
}

func (parseF *stringListFlag) Set(parseValue string) error {
	for parsePart := range strings.SplitSeq(parseValue, ",") {
		parseTrimmed := strings.TrimSpace(parsePart)
		if parseTrimmed != "" {
			parseF.values = append(parseF.values, parseTrimmed)
		}
	}
	return nil
}

func (parseF *stringListFlag) Values() []string {
	return append([]string(nil), parseF.values...)
}

func normalizeTestLanes(parseRequested []string) ([]string, error) {
	if len(parseRequested) == 0 {
		return []string{"unit", "wasm"}, nil
	}
	parseSeen := map[string]struct{}{}
	parseNormalized := make([]string, 0, len(parseRequested))
	parseAppendLane := func(parseLane2 string) {
		if _, parseOk := parseSeen[parseLane2]; parseOk {
			return
		}
		parseSeen[parseLane2] = struct{}{}
		parseNormalized = append(parseNormalized, parseLane2)
	}
	for _, parseLane := range parseRequested {
		switch strings.ToLower(strings.TrimSpace(parseLane)) {
		case "all":
			for _, parseCandidate := range testAllLanes {
				parseAppendLane(parseCandidate)
			}
		case "unit", "native", "go-native":
			parseAppendLane("unit")
		case "race", "race-detector", "go-race":
			parseAppendLane("race")
		case "wasm", "go-wasm":
			parseAppendLane("wasm")
		case "hydration", "hydrate":
			parseAppendLane("hydration")
		case "browser", "playwright":
			parseAppendLane("browser")
		case "perf", "performance", "perf-budget", "budget":
			parseAppendLane("perf")
		case "i18n", "locales", "locale":
			parseAppendLane("i18n")
		case "agent", "agentbridge", "agent-bridge", "bridge":
			parseAppendLane("agent")
		case "agent-browser", "agent-e2e", "bridge-e2e", "headless-bridge":
			parseAppendLane("agent-browser")
		case "release":
			parseAppendLane("release")
		default:
			return nil, fmt.Errorf("unknown test lane %q", parseLane)
		}
	}
	return parseNormalized, nil
}

func enforceEnterpriseRequiredTestLanes(parseSelectedLanes []string, parsePolicy launcherEnterprisePolicy) error {
	if len(parsePolicy.RequiredTestLanes) == 0 {
		return nil
	}
	parseSelected := map[string]struct{}{}
	for _, parseLane := range parseSelectedLanes {
		parseTrimmed := strings.TrimSpace(strings.ToLower(parseLane))
		if parseTrimmed == "" {
			continue
		}
		parseSelected[parseTrimmed] = struct{}{}
	}
	parseMissing := []string{}
	for _, parseLane2 := range parsePolicy.RequiredTestLanes {
		parseNormalized := strings.TrimSpace(strings.ToLower(parseLane2))
		if parseNormalized == "" {
			continue
		}
		if _, parseOk := parseSelected[parseNormalized]; !parseOk {
			parseMissing = append(parseMissing, parseLane2)
		}
	}
	if len(parseMissing) > 0 {
		return fmt.Errorf("missing required test lanes: %s", strings.Join(parseMissing, ", "))
	}
	return nil
}

func enforceEnterpriseGoToolchainPolicy(parseRootPath string, parsePolicy launcherEnterprisePolicy) error {
	if len(parsePolicy.ApprovedGoToolchains) == 0 {
		return nil
	}
	parseActiveToolchain, parseErr := resolveActiveGoToolchain(parseRootPath)
	if parseErr != nil {
		return parseErr
	}
	for _, parseApproved := range parsePolicy.ApprovedGoToolchains {
		if goToolchainApprovedByPolicy(parseActiveToolchain, parseApproved) {
			return nil
		}
	}
	return fmt.Errorf(
		"active Go toolchain %q is not approved; allowed values: %s",
		parseActiveToolchain,
		strings.Join(parsePolicy.ApprovedGoToolchains, ", "),
	)
}

func isGoRaceDetectorSupported(parseGOOS string, parseGOARCH string) bool {
	switch parseGOOS {
	case "darwin":
		return parseGOARCH == "amd64" || parseGOARCH == "arm64"
	case "linux":
		switch parseGOARCH {
		case "amd64", "arm64", "ppc64le", "s390x", "loong64":
			return true
		default:
			return false
		}
	case "freebsd", "netbsd", "windows":
		return parseGOARCH == "amd64"
	default:
		return false
	}
}

func resolveActiveGoToolchain(parseRootPath string) (string, error) {
	parseCwd := strings.TrimSpace(parseRootPath)
	if parseCwd == "" {
		parseCwd = "."
	}
	parseOutput, parseErr := launcherRunCommand("go", []string{"env", "GOVERSION"}, parseCwd, buildNativeGoEnv())
	if parseErr != nil {
		return "", fmt.Errorf("resolve active Go toolchain: %w", parseErr)
	}
	parseLines := strings.Split(strings.TrimSpace(parseOutput), "\n")
	parseVersion := strings.ToLower(strings.TrimSpace(parseLines[0]))
	if parseVersion == "" {
		return "", errors.New("resolve active Go toolchain: go env GOVERSION returned empty output")
	}
	return parseVersion, nil
}

func goToolchainApprovedByPolicy(parseActiveToolchain string, parseApprovedValue string) bool {
	parseActive := normalizeGoToolchainPolicyValue(parseActiveToolchain)
	parseApproved := normalizeGoToolchainPolicyValue(parseApprovedValue)
	if parseActive == "" || parseApproved == "" {
		return false
	}
	if parseActive == parseApproved {
		return true
	}
	if before, ok := strings.CutSuffix(parseApproved, ".x"); ok {
		parsePrefix := before
		if parsePrefix == "" {
			return false
		}
		if parseActive == parsePrefix {
			return true
		}
		return strings.HasPrefix(parseActive, parsePrefix+".")
	}
	return strings.HasPrefix(parseActive, parseApproved+".")
}

func normalizeGoToolchainPolicyValue(parseValue string) string {
	parseValue = strings.TrimSpace(strings.ToLower(parseValue))
	if parseValue == "" {
		return ""
	}
	if strings.HasPrefix(parseValue, "go") {
		return parseValue
	}
	return "go" + parseValue
}

func enforceEnterpriseReleasePolicy(parseConfig releaseConfig, parsePolicy launcherEnterprisePolicy) error {
	if parsePolicy.RequireReleaseBudgets != nil && *parsePolicy.RequireReleaseBudgets && strings.TrimSpace(parseConfig.budgetsPath) == "" {
		return errors.New("enterprise policy requires release budgets; provide -budgets or configure releaseBudgetsPath")
	}
	if parseRequiredCompression := strings.TrimSpace(parsePolicy.RequiredReleaseCompression); parseRequiredCompression != "" {
		parseNormalizedRequired, parseErr := normalizeReleaseCompressionPolicy(parseRequiredCompression)
		if parseErr != nil {
			return fmt.Errorf("normalize enterprise required release compression: %w", parseErr)
		}
		if parseConfig.compression != parseNormalizedRequired {
			return fmt.Errorf("release compression policy %q does not satisfy enterprise requirement %q", parseConfig.compression, parseNormalizedRequired)
		}
	}
	if parsePattern := strings.TrimSpace(parsePolicy.ReleaseBinaryPattern); parsePattern != "" {
		parseMatched, parseErr2 := regexp.MatchString(parsePattern, parseConfig.binaryName)
		if parseErr2 != nil {
			return fmt.Errorf("compile enterprise release binary pattern: %w", parseErr2)
		}
		if !parseMatched {
			return fmt.Errorf("release binary name %q does not satisfy enterprise pattern %q", parseConfig.binaryName, parsePattern)
		}
	}
	if parsePattern2 := strings.TrimSpace(parsePolicy.ReleaseManifestPattern); parsePattern2 != "" {
		parseMatched2, parseErr3 := regexp.MatchString(parsePattern2, parseConfig.manifestName)
		if parseErr3 != nil {
			return fmt.Errorf("compile enterprise release manifest pattern: %w", parseErr3)
		}
		if !parseMatched2 {
			return fmt.Errorf("release manifest name %q does not satisfy enterprise pattern %q", parseConfig.manifestName, parsePattern2)
		}
	}
	return nil
}

func collectWasmTestPackages(parseRootPath string, isHydrationOnly bool) ([]string, error) {
	parsePackages := map[string]struct{}{}
	parseErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipTestWalkDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			// Skip nested Go modules so root-lane package patterns stay valid.
			if parsePath != parseRootPath && fileExists(filepath.Join(parsePath, "go.mod")) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(parseEntry.Name(), "_wasm_test.go") {
			return nil
		}
		if isHydrationOnly {
			parseContent, parseErr2 := os.ReadFile(parsePath)
			if parseErr2 != nil {
				return parseErr2
			}
			if !isHydrationTestContent(string(parseContent)) {
				return nil
			}
		}
		parseRelDir, parseErr3 := filepath.Rel(parseRootPath, filepath.Dir(parsePath))
		if parseErr3 != nil {
			return parseErr3
		}
		parsePackagePath := "."
		if parseRelDir != "." {
			parsePackagePath = "./" + filepath.ToSlash(parseRelDir)
		}
		parsePackages[parsePackagePath] = struct{}{}
		return nil
	})
	if parseErr != nil {
		return nil, fmt.Errorf("collect js/wasm test packages: %w", parseErr)
	}
	parseOrdered := make([]string, 0, len(parsePackages))
	for parsePkg := range parsePackages {
		parseOrdered = append(parseOrdered, parsePkg)
	}
	sort.Strings(parseOrdered)
	return parseOrdered, nil
}

func shouldSkipTestWalkDir(parseName string) bool {
	return parseName == ".git" ||
		parseName == "node_modules" ||
		parseName == "dist" ||
		parseName == "tmp" ||
		parseName == "test-results" ||
		parseName == "playwright-report" ||
		parseName == "coverage"
}

func isHydrationTestContent(parseContent string) bool {
	return strings.Contains(parseContent, "SmokeHydrate") ||
		strings.Contains(parseContent, "Hydrate") ||
		strings.Contains(parseContent, "Hydration")
}

func buildNativeGoEnv() []string {
	parseEnv := []string{}
	for _, parseEntry := range os.Environ() {
		if strings.HasPrefix(parseEntry, "GOOS=") || strings.HasPrefix(parseEntry, "GOARCH=") {
			continue
		}
		parseEnv = append(parseEnv, parseEntry)
	}
	return parseEnv
}

func buildWasmGoEnv() []string {
	parseEnv := buildNativeGoEnv()
	parseEnv = append(parseEnv, "GOOS=js", "GOARCH=wasm")
	return parseEnv
}

func buildBrowserTestEnv() []string {
	parseEnv := buildNativeGoEnv()
	isParseWorkersSet := false
	for _, parseEntry := range parseEnv {
		if strings.HasPrefix(parseEntry, "PLAYWRIGHT_WORKERS=") {
			isParseWorkersSet = true
			break
		}
	}
	if !isParseWorkersSet {
		parseEnv = append(parseEnv, "PLAYWRIGHT_WORKERS=4")
	}
	return parseEnv
}

func resolveBrowserTestPackagePattern(parseWorkspace string) (string, bool) {
	if strings.TrimSpace(parseWorkspace) == "" {
		return "", false
	}
	parseCandidates := []struct {
		path    string
		pattern string
	}{
		{path: filepath.Join(parseWorkspace, "playwrightgo"), pattern: "./playwrightgo"},
		{path: filepath.Join(parseWorkspace, "test", "playwrightgo"), pattern: "./test/playwrightgo"},
	}
	for _, parseCandidate := range parseCandidates {
		parseInfo, parseErr := os.Stat(parseCandidate.path)
		if parseErr != nil || !parseInfo.IsDir() {
			continue
		}
		return parseCandidate.pattern, true
	}
	return "", false
}

func resolvePerfBudgetTestPackagePattern(parseWorkspace string) (string, bool) {
	if strings.TrimSpace(parseWorkspace) == "" {
		return "", false
	}
	parseCandidates := []struct {
		path    string
		pattern string
	}{
		{path: filepath.Join(parseWorkspace, "playwrightgo", "examples"), pattern: "./playwrightgo/examples"},
		{path: filepath.Join(parseWorkspace, "test", "playwrightgo", "examples"), pattern: "./test/playwrightgo/examples"},
	}
	for _, parseCandidate := range parseCandidates {
		parseInfo, parseErr := os.Stat(parseCandidate.path)
		if parseErr != nil || !parseInfo.IsDir() {
			continue
		}
		return parseCandidate.pattern, true
	}
	return "", false
}

func resolveAgentBridgeHeadlessTestPackagePattern(parseWorkspace string) (string, bool) {
	if strings.TrimSpace(parseWorkspace) == "" {
		return "", false
	}
	parseCandidates := []struct {
		path    string
		pattern string
	}{
		{path: filepath.Join(parseWorkspace, "test", "playwrightgo", "examples"), pattern: "./test/playwrightgo/examples"},
		{path: filepath.Join(parseWorkspace, "playwrightgo", "examples"), pattern: "./playwrightgo/examples"},
	}
	for _, parseCandidate := range parseCandidates {
		parseInfo, parseErr := os.Stat(parseCandidate.path)
		if parseErr != nil || !parseInfo.IsDir() {
			continue
		}
		if containsAgentBridgeDogfoodTest(parseCandidate.path) {
			return parseCandidate.pattern, true
		}
	}
	return "", false
}

func containsAgentBridgeDogfoodTest(parsePackagePath string) bool {
	parseEntries, parseErr := os.ReadDir(parsePackagePath)
	if parseErr != nil {
		return false
	}
	parsePattern := regexp.MustCompile(`func\s+(TestExample100AgentBridgeDogfood|TestAgentBridgeDogfood|TestAgentBridgeHeadless)\s*\(`)
	for _, parseEntry := range parseEntries {
		if parseEntry.IsDir() || !strings.HasSuffix(parseEntry.Name(), "_test.go") {
			continue
		}
		parsePayload, parseReadErr := os.ReadFile(filepath.Join(parsePackagePath, parseEntry.Name()))
		if parseReadErr != nil {
			continue
		}
		if parsePattern.Match(parsePayload) {
			return true
		}
	}
	return false
}

func resolveWasmTestExec(parseRepoRoot string) (string, error) {
	parseOverridePath, parseOk, parseErr := resolveLauncherConfiguredPath(parseRepoRoot, func(parsePaths launcherOverridePaths) string {
		return parsePaths.GoWASMExec
	}, "goWasmExec")
	if parseErr != nil {
		return "", parseErr
	}
	if parseOk {
		if !fileExists(parseOverridePath) {
			return "", fmt.Errorf("configured goWasmExec path does not exist: %s", parseOverridePath)
		}
		return parseOverridePath, nil
	}
	if parseValue := strings.TrimSpace(os.Getenv("GO_WASM_EXEC")); parseValue != "" {
		return parseValue, nil
	}
	if runtime.GOOS == "windows" {
		parseCandidate := filepath.Join(parseRepoRoot, "tools", "go_js_wasm_exec.bat")
		if fileExists(parseCandidate) {
			return parseCandidate, nil
		}
	}
	return "", errors.New("GO_WASM_EXEC is not set and the repo js/wasm executor helper could not be resolved")
}

func resolveBrowserWorkspace(parseRepoRoot string, parseRootPath string) (string, error) {
	parseOverridePath, parseOk, parseErr := resolveLauncherConfiguredPath(parseRootPath, func(parsePaths launcherOverridePaths) string {
		return parsePaths.BrowserWorkspace
	}, "browserWorkspace")
	if parseErr != nil {
		return "", parseErr
	}
	if parseOk {
		parseInfo, parseStatErr := os.Stat(parseOverridePath)
		if parseStatErr != nil || !parseInfo.IsDir() {
			return "", fmt.Errorf("configured browserWorkspace path does not exist: %s", parseOverridePath)
		}
		if _, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(parseOverridePath); !hasPlaywrightGoSuite {
			return "", fmt.Errorf("configured browserWorkspace does not contain a Playwright-Go suite: %s", parseOverridePath)
		}
		return parseOverridePath, nil
	}
	if _, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(parseRootPath); hasPlaywrightGoSuite {
		return parseRootPath, nil
	}
	parseRepoWorkspace := filepath.Join(parseRepoRoot, "test")
	if _, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(parseRepoWorkspace); hasPlaywrightGoSuite {
		return parseRepoWorkspace, nil
	}
	return "", nil
}

func detectBrowserWorkspace(parseRepoRoot string, parseRootPath string) string {
	parseWorkspace, parseErr := resolveBrowserWorkspace(parseRepoRoot, parseRootPath)
	if parseErr != nil {
		return ""
	}
	return parseWorkspace
}

func resolveLauncherLivereloadWorkspace(parseRepoRoot string, parseRootPath string) (string, error) {
	parseOverridePath, parseOk, parseErr := resolveLauncherConfiguredPath(parseRootPath, func(parsePaths launcherOverridePaths) string {
		return parsePaths.LivereloadWorkspace
	}, "livereloadWorkspace")
	if parseErr != nil {
		return "", parseErr
	}
	if parseOk {
		parseInfo, parseStatErr := os.Stat(parseOverridePath)
		if parseStatErr != nil || !parseInfo.IsDir() {
			return "", fmt.Errorf("configured livereloadWorkspace path does not exist: %s", parseOverridePath)
		}
		return parseOverridePath, nil
	}
	return filepath.Join(parseRepoRoot, "tools", "livereload"), nil
}

func resolveLauncherLivereloadClientScript(parseRootPath string, _ ...string) (string, bool, error) {
	parseOverridePath, parseOk, parseErr := resolveLauncherConfiguredPath(parseRootPath, func(parsePaths launcherOverridePaths) string {
		return parsePaths.LivereloadClientScript
	}, "livereloadClientScript")
	if parseErr != nil {
		return "", false, parseErr
	}
	if parseOk {
		if !fileExists(parseOverridePath) {
			return "", false, fmt.Errorf("configured livereloadClientScript path does not exist: %s", parseOverridePath)
		}
		return parseOverridePath, true, nil
	}
	return "", false, nil
}
