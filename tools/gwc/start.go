package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type scaffoldToolingMetadata struct {
	AppPath             string `json:"appPath,omitempty"`
	HTMLPath            string `json:"htmlPath,omitempty"`
	WASMPath            string `json:"wasmPath,omitempty"`
	DevHost             string `json:"devHost,omitempty"`
	DevPort             string `json:"devPort,omitempty"`
	DefaultBuildProfile string `json:"defaultBuildProfile,omitempty"`
	ReleaseOutDir       string `json:"releaseOutDir,omitempty"`
	ReleaseBinaryName   string `json:"releaseBinaryName,omitempty"`
	ReleaseCompression  string `json:"releaseCompression,omitempty"`
	ReleaseBudgetsPath  string `json:"releaseBudgetsPath,omitempty"`
}

type scaffoldPresetMetadata struct {
	Key         string   `json:"key,omitempty"`
	Name        string   `json:"name,omitempty"`
	Summary     string   `json:"summary,omitempty"`
	Description string   `json:"description,omitempty"`
	Features    []string `json:"features,omitempty"`
}

type scaffoldEnterpriseSectionMetadata struct {
	Title    string   `json:"title,omitempty"`
	Summary  string   `json:"summary,omitempty"`
	Features []string `json:"features,omitempty"`
}

type scaffoldEnterpriseMetadata struct {
	EnabledSections []string                            `json:"enabledSections,omitempty"`
	Features        []string                            `json:"features,omitempty"`
	Sections        []scaffoldEnterpriseSectionMetadata `json:"sections,omitempty"`
}

type scaffoldOwnershipMetadata struct {
	ProjectOwnership    string `json:"projectOwnership,omitempty"`
	FrameworkSourceMode string `json:"frameworkSourceMode,omitempty"`
}

type scaffoldMetadata struct {
	SchemaVersion int                        `json:"schemaVersion,omitempty"`
	ProjectName   string                     `json:"projectName,omitempty"`
	ModulePath    string                     `json:"modulePath,omitempty"`
	Author        string                     `json:"author,omitempty"`
	Version       string                     `json:"version,omitempty"`
	Description   string                     `json:"description,omitempty"`
	TargetDir     string                     `json:"targetDir,omitempty"`
	Preset        scaffoldPresetMetadata     `json:"preset"`
	Enterprise    scaffoldEnterpriseMetadata `json:"enterprise"`
	Ownership     scaffoldOwnershipMetadata  `json:"ownership"`
	Tooling       scaffoldToolingMetadata    `json:"tooling"`
}

const currentScaffoldMetadataSchemaVersion = 1

var scaffoldResolveWasmExecPath = resolveWasmExecPath

var scaffoldWriteFile = os.WriteFile

var scaffoldReadFile = os.ReadFile

var scaffoldMarshalIndent = json.MarshalIndent

var scaffoldFormatMain = func(mainPath string) error {
	return exec.Command("gofmt", "-w", mainPath).Run()
}

var scaffoldTidyModule = func(l launcher, targetDir string) error {
	return l.tidyScaffoldModule(targetDir)
}

var resolveWasmExecGoRoot = func() string {
	parseOutput, parseErr := exec.Command("go", "env", "GOROOT").Output()
	if parseErr != nil {
		return ""
	}
	return strings.TrimSpace(string(parseOutput))
}

var resolveRepoRootCaller = runtime.Caller

var startTerminalValidator = validateStartTerminal

var startSelectionRunner = runStartTUI

var startPostRunner = runStartPostTUI

var startResolveScaffoldSections = resolveStartScaffoldPluginSections

var startGenerateScaffold = func(l launcher, selection startSelection) (scaffoldResult, error) {
	return l.generateStartScaffold(selection)
}

var startInitGit = func(targetDir string) error {
	cmd := exec.Command("git", "init")
	cmd.Dir = targetDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("initialize starter git repository: %w", err)
		}
		return fmt.Errorf("initialize starter git repository: %s", trimmed)
	}
	return nil
}

var startRunDev = func(l launcher, args []string) error {
	return l.runDev(args)
}

func (parseL launcher) runStart(parseArgs []string) error {
	parseFs := flag.NewFlagSet("start", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseSkipPrereqChecks := parseFs.Bool("skip-prereq-checks", false, "Skip start-time prerequisite checks (Go, runtime assets, and preset-required browser tooling)")
	parseSkipTidy := parseFs.Bool("skip-tidy", false, "Skip go mod tidy after scaffold generation")
	parseSkipRuntimeAssets := parseFs.Bool("skip-runtime-assets", false, "Skip copying runtime assets such as wasm_exec.js into the generated scaffold")
	parseProjectMode := parseFs.String("mode", string(scaffoldProjectModeStandalone), "Scaffold mode: standalone or contributor-linked")
	parseInitGit := parseFs.Bool("init-git", false, "Initialize a fresh git repository in the generated scaffold root")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseNormalizedProjectMode, parseOk := normalizeScaffoldProjectMode(*parseProjectMode)
	if !parseOk {
		return fmt.Errorf("unknown scaffold mode %q", *parseProjectMode)
	}
	if parseErr2 := startTerminalValidator(isInteractiveFile(os.Stdin), isInteractiveFile(os.Stdout)); parseErr2 != nil {
		return parseErr2
	}
	parseSections, parseErr3 := startResolveScaffoldSections(parseL.repoRoot)
	if parseErr3 != nil {
		return parseErr3
	}
	startEnterpriseScaffoldSections = parseSections
	parsePreviousProjectMode := startDefaultProjectMode
	startDefaultProjectMode = parseNormalizedProjectMode
	defer func() {
		startEnterpriseScaffoldSections = nil
		startDefaultProjectMode = parsePreviousProjectMode
	}()
	parseSelection, parseErr3 := startSelectionRunner()
	if parseErr3 != nil {
		return parseErr3
	}
	if parseSelection == nil {
		return nil
	}

	parseSelection.ProjectMode = parseNormalizedProjectMode
	parseSelection.InitGit = *parseInitGit
	parseSelection.SkipGoModTidy = *parseSkipTidy
	parseSelection.SkipRuntimeAssets = *parseSkipRuntimeAssets

	if !*parseSkipPrereqChecks {
		if parseErr4 := parseL.validateStartPrerequisites(*parseSelection); parseErr4 != nil {
			_, parsePostErr := startPostRunner(*parseSelection, nil, parseErr4)
			if parsePostErr != nil {
				return parsePostErr
			}
			return nil
		}
	}

	parseResult, parseErr3 := startGenerateScaffold(parseL, *parseSelection)
	if parseErr3 != nil {
		_, parsePostErr2 := startPostRunner(*parseSelection, nil, parseErr3)
		if parsePostErr2 != nil {
			return parsePostErr2
		}
		return nil
	}
	if parseSelection.InitGit {
		if parseErr5 := startInitGit(parseResult.TargetDir); parseErr5 != nil {
			_, parsePostErr3 := startPostRunner(*parseSelection, &parseResult, parseErr5)
			if parsePostErr3 != nil {
				return parsePostErr3
			}
			return nil
		}
	}

	parsePostResult, parseErr3 := startPostRunner(*parseSelection, &parseResult, nil)
	if parseErr3 != nil {
		return parseErr3
	}
	if parsePostResult == nil || !parsePostResult.RunDev {
		return nil
	}

	return startRunDev(parseL, devArgsFromScaffold(parseResult))
}

func resolveStartScaffoldPluginSections(parseRepoRoot string) ([]launcherPluginScaffoldSection, error) {
	parseResults, parseErr := runLauncherPluginsForCapability("scaffold_feature", "start", nil, parseRepoRoot, launcherActiveEnterpriseSources)
	if parseErr != nil {
		return nil, parseErr
	}
	parseSections := []launcherPluginScaffoldSection{}
	for _, parseResult := range parseResults {
		parseSections = append(parseSections, parseResult.Response.ScaffoldSections...)
	}
	return normalizeStartEnterpriseSections(parseSections), nil
}

func (parseL launcher) runBootstrap(parseArgs []string) error {
	parseFs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseExamplesMode := parseFs.Bool("examples", false, "Run the examples catalog flow after prerequisite checks")
	parseHost := parseFs.String("host", defaultHost, "Host used by bootstrap prerequisite checks")
	parsePort := parseFs.String("port", "8080", "Port used by bootstrap prerequisite checks")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseReport := parseL.buildDoctorReport(doctorConfig{host: *parseHost, port: *parsePort, json: false})
	printDoctorReport(parseReport)
	if !parseReport.OK {
		return errors.New("bootstrap blocked by doctor checks")
	}

	if *parseExamplesMode {
		return runExamplesCommand(parseL, parseFs.Args())
	}
	return runStartCommand(parseL, parseFs.Args())
}

func (parseL launcher) validateStartPrerequisites(parseSelection startSelection) error {
	parseChecks := []doctorCheck{
		buildDoctorToolCheck("go", "Go toolchain", "version", "Install Go 1.25+ and ensure `go` is on PATH before running gwc start."),
	}
	if !parseSelection.SkipRuntimeAssets {
		parseChecks = append(parseChecks, buildDoctorWasmExecCheck())
	}
	if scaffoldHasFeature(startSelectionFeatures(parseSelection), "browser-tests") {
		parseChecks = append(parseChecks,
			buildDoctorPlaywrightCheck(parseL.repoRoot),
		)
	}

	parseFailed := []string{}
	parseWarnings := []string{}
	for _, parseCheck := range parseChecks {
		switch parseCheck.Status {
		case "fail":
			parseFailed = append(parseFailed, fmt.Sprintf("%s: %s", parseCheck.Name, parseCheck.Summary))
		case "warn":
			parseWarnings = append(parseWarnings, fmt.Sprintf("%s: %s", parseCheck.Name, parseCheck.Summary))
		}
	}
	if len(parseWarnings) > 0 {
		fmt.Println("GWC start prerequisites (warnings):")
		for _, parseWarning := range parseWarnings {
			fmt.Printf("  - %s\n", parseWarning)
		}
	}
	if len(parseFailed) == 0 {
		return nil
	}
	return fmt.Errorf("start prerequisites failed:\n  - %s", strings.Join(parseFailed, "\n  - "))
}

type scaffoldResult struct {
	TargetDir string
	AppPath   string
	HTMLPath  string
}

func validateStartTerminal(isStdinIsTerminal bool, isStdoutIsTerminal bool) error {
	if isStdinIsTerminal && isStdoutIsTerminal {
		return nil
	}
	return errors.New("start requires an interactive terminal with TUI support; run `go run ./tools/gwc start` from a normal shell session")
}

func isInteractiveFile(parseFile *os.File) bool {
	if parseFile == nil {
		return false
	}
	parseInfo, parseErr := parseFile.Stat()
	if parseErr != nil {
		return false
	}
	return (parseInfo.Mode() & os.ModeCharDevice) != 0
}

func devArgsFromScaffold(parseResult scaffoldResult) []string {
	return []string{
		"-app", parseResult.AppPath,
		"-root", parseResult.TargetDir,
		"-html", parseResult.HTMLPath,
		"-wasm", scaffoldWASMOutputPath(),
	}
}

func scaffoldWASMOutputPath() string {
	return filepath.ToSlash(filepath.Join("bin", "main.wasm"))
}

func defaultScaffoldBuildProfile() string {
	return "development"
}

func defaultScaffoldReleaseOutDir() string {
	return filepath.ToSlash(filepath.Join("bin", "wasm-release"))
}

func defaultScaffoldReleaseBinaryName() string {
	return "app.wasm"
}

func defaultScaffoldReleaseCompression() string {
	return "gzip+brotli"
}

func cloneResolutionTrace(parseValues map[string]string) map[string]string {
	if len(parseValues) == 0 {
		return map[string]string{}
	}
	parseCloned := make(map[string]string, len(parseValues))
	maps.Copy(parseCloned, parseValues)
	return parseCloned
}

func setResolutionSource(parseValues map[string]string, parseKey string, parseSource string) map[string]string {
	if parseValues == nil {
		parseValues = map[string]string{}
	}
	if strings.TrimSpace(parseKey) != "" && strings.TrimSpace(parseSource) != "" {
		parseValues[parseKey] = parseSource
	}
	return parseValues
}

func printResolutionTrace(parseTrace map[string]string, parseKeys []string, parseIndent string) {
	if len(parseTrace) == 0 {
		return
	}
	for _, parseKey := range parseKeys {
		if parseSource, parseOk := parseTrace[parseKey]; parseOk && strings.TrimSpace(parseSource) != "" {
			fmt.Printf("%s%s source: %s\n", parseIndent, parseKey, parseSource)
		}
	}
}

func (parseL launcher) generateStartScaffold(parseSelection startSelection) (scaffoldResult, error) {
	parseTargetDir := filepath.Clean(parseSelection.TargetDir)
	if parseErr := validateGeneratedTargetDir(parseTargetDir); parseErr != nil {
		return scaffoldResult{}, parseErr
	}
	if parseErr2 := ensureEmptyDir(parseTargetDir); parseErr2 != nil {
		return scaffoldResult{}, parseErr2
	}
	parseRepoModulePath, parseErr3 := parseL.readRepoModulePath()
	if parseErr3 != nil {
		return scaffoldResult{}, parseErr3
	}
	return parseL.generateScaffoldProject(scaffoldPlan{
		Selection:         parseSelection,
		GoMod:             renderScaffoldGoMod(parseSelection, parseRepoModulePath, parseL.repoRoot),
		MainGo:            renderScaffoldMain(parseSelection, parseRepoModulePath),
		HTML:              renderScaffoldHTML(parseSelection),
		README:            renderScaffoldREADME(parseSelection),
		Metadata:          defaultScaffoldMetadata(parseSelection),
		ExtraFiles:        renderScaffoldExtraFiles(parseSelection),
		SkipGoModTidy:     parseSelection.SkipGoModTidy,
		SkipRuntimeAssets: parseSelection.SkipRuntimeAssets,
	})
}

func (parseL launcher) seedScaffoldGoSum(parseTargetDir string) error {
	parseRootGoSumPath := filepath.Join(parseL.repoRoot, "go.sum")
	parseGoSumBytes, parseErr := os.ReadFile(parseRootGoSumPath)
	if parseErr != nil {
		if os.IsNotExist(parseErr) {
			return nil
		}
		return fmt.Errorf("read repo go.sum: %w", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseTargetDir, "go.sum"), parseGoSumBytes, 0644); parseErr2 != nil {
		return fmt.Errorf("write scaffold go.sum: %w", parseErr2)
	}
	return nil
}

func (parseL launcher) tidyScaffoldModule(parseTargetDir string) error {
	parseCmd := exec.Command("go", "mod", "tidy")
	parseCmd.Dir = parseTargetDir
	parseCmd.Env = os.Environ()
	parseOutput, parseErr := parseCmd.CombinedOutput()
	if parseErr != nil {
		parseTrimmed := strings.TrimSpace(string(parseOutput))
		if parseTrimmed == "" {
			return fmt.Errorf("prepare scaffold module: %w", parseErr)
		}
		return fmt.Errorf("prepare scaffold module: %s", parseTrimmed)
	}
	return nil
}

func ensureEmptyDir(parseTargetDir string) error {
	if parseInfo, parseErr := os.Stat(parseTargetDir); parseErr == nil {
		if !parseInfo.IsDir() {
			return fmt.Errorf("target path exists and is not a directory: %s", parseTargetDir)
		}
		parseEntries, parseReadErr := os.ReadDir(parseTargetDir)
		if parseReadErr != nil {
			return fmt.Errorf("read target directory: %w", parseReadErr)
		}
		if len(parseEntries) > 0 {
			return fmt.Errorf("target directory already exists and is not empty: %s", parseTargetDir)
		}
		return nil
	} else if !os.IsNotExist(parseErr) {
		return fmt.Errorf("inspect target directory: %w", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseTargetDir, 0755); parseErr2 != nil {
		return fmt.Errorf("create target directory: %w", parseErr2)
	}
	return nil
}

func (parseL launcher) readRepoModulePath() (string, error) {
	parseGoModPath := filepath.Join(parseL.repoRoot, "go.mod")
	parseContent, parseErr := os.ReadFile(parseGoModPath)
	if parseErr != nil {
		return "", fmt.Errorf("read repo go.mod: %w", parseErr)
	}
	for parseLine := range strings.SplitSeq(string(parseContent), "\n") {
		parseTrimmed := strings.TrimSpace(parseLine)
		if after, ok := strings.CutPrefix(parseTrimmed, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("repo module path not found in %s", parseGoModPath)
}

func resolveWasmExecPath() (string, error) {
	parseOverrides, parseConfigPath, parseOk, parseErr := loadLauncherOverridesForCurrentContext()
	if parseErr != nil {
		return "", parseErr
	}
	if parseOk {
		parseOverridePath, parseErr2 := resolveLauncherOverrideValue(parseConfigPath, parseOverrides.Paths.WASMExecJS)
		if parseErr2 != nil {
			return "", fmt.Errorf("resolve wasmExecJS override: %w", parseErr2)
		}
		if strings.TrimSpace(parseOverridePath) != "" {
			if !fileExists(parseOverridePath) {
				return "", fmt.Errorf("configured wasmExecJS path does not exist: %s", parseOverridePath)
			}
			return parseOverridePath, nil
		}
	}
	parseGoRoot := resolveWasmExecGoRoot()
	if parseGoRoot == "" {
		return "", errors.New("GOROOT is not available")
	}
	parseCandidates := []string{
		filepath.Join(parseGoRoot, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(parseGoRoot, "misc", "wasm", "wasm_exec.js"),
	}
	for _, parseCandidate := range parseCandidates {
		if _, parseErr3 := os.Stat(parseCandidate); parseErr3 == nil {
			return parseCandidate, nil
		}
	}
	return "", fmt.Errorf("wasm_exec.js not found under GOROOT %s", parseGoRoot)
}

func renderScaffoldGoMod(parseSelection startSelection, parseRepoModulePath string, parseRepoRoot string) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString(fmt.Sprintf("module %s\n\ngo 1.25.0\n", parseSelection.ModulePath))
	if parseSelection.ProjectMode == scaffoldProjectModeContributorLinked && strings.TrimSpace(parseRepoModulePath) != "" && strings.TrimSpace(parseRepoRoot) != "" {
		parseBuilder.WriteString("\n")
		parseBuilder.WriteString(fmt.Sprintf("replace %s => %s\n", parseRepoModulePath, filepath.ToSlash(filepath.Clean(parseRepoRoot))))
	}
	return parseBuilder.String()
}

type scaffoldFeatureDescriptor struct {
	Key         string
	Label       string
	Description string
}

var scaffoldFeatureCatalog = []scaffoldFeatureDescriptor{
	{Key: "router", Label: "Router", Description: "Route shell and navigation affordances are scaffolded in the starter layout."},
	{Key: "ssr", Label: "SSR", Description: "Server rendering and hydration expectations are documented in the generated matrix."},
	{Key: "forms", Label: "Forms", Description: "Starter form interactions are scaffolded so teams can extend typed form state deliberately."},
	{Key: "fetch", Label: "Fetch", Description: "Async resource ownership is called out for data-loading and mutation setup."},
	{Key: "state", Label: "State", Description: "Shared state ownership is planned as a first-class concern in this scaffold."},
	{Key: "devtools", Label: "Devtools", Description: "Devtools adoption is surfaced as part of the starter capability model."},
	{Key: "hot-reload", Label: "Hot Reload", Description: "State-preserving local reload is included as the recommended inner-loop path."},
	{Key: "browser-tests", Label: "Browser Tests", Description: "Playwright-Go smoke-test scaffolding is generated under test/playwrightgo."},
	{Key: "hydration", Label: "Hydration", Description: "Client boot and hydration ownership are expected from the first app shell."},
	{Key: "release-profile", Label: "Release Profile", Description: "Release-minded defaults are encoded in launcher metadata and docs."},
	{Key: "dev-profile", Label: "Dev Profile", Description: "Fast local iteration is pre-wired through gwc dev defaults."},
	{Key: "browser-mount", Label: "Browser Mount", Description: "Direct browser mount behavior is kept explicit in the main entrypoint."},
	{Key: "ui", Label: "UI", Description: "Component composition is scaffolded through the public ui package."},
	{Key: "html", Label: "HTML", Description: "Typed HTML builders are used as the default authoring path."},
}

func startSelectionFeatures(parseSelection startSelection) []string {
	parseCombined := append([]string{}, parseSelection.Preset.Features...)
	parseCombined = append(parseCombined, parseSelection.EnterpriseFeatures...)
	return normalizeScaffoldFeatureKeys(parseCombined)
}

func normalizeScaffoldFeatureKeys(parseFeatures []string) []string {
	parseSeen := make(map[string]struct{}, len(parseFeatures))
	parseNormalized := make([]string, 0, len(parseFeatures))
	for _, parseRaw := range parseFeatures {
		parseFeature := strings.ToLower(strings.TrimSpace(parseRaw))
		if parseFeature == "" {
			continue
		}
		if _, parseExists := parseSeen[parseFeature]; parseExists {
			continue
		}
		parseSeen[parseFeature] = struct{}{}
		parseNormalized = append(parseNormalized, parseFeature)
	}
	return parseNormalized
}

func scaffoldFeatureDescriptorFor(parseKey string) scaffoldFeatureDescriptor {
	parseKey = strings.ToLower(strings.TrimSpace(parseKey))
	for _, parseDescriptor := range scaffoldFeatureCatalog {
		if parseDescriptor.Key == parseKey {
			return parseDescriptor
		}
	}
	parseLabel := strings.TrimSpace(strings.ReplaceAll(parseKey, "-", " "))
	if parseLabel == "" {
		parseLabel = "Unknown Capability"
	}
	return scaffoldFeatureDescriptor{
		Key:         parseKey,
		Label:       strings.ToUpper(parseLabel[:1]) + parseLabel[1:],
		Description: "Custom starter capability carried in preset metadata.",
	}
}

func scaffoldFeatureSet(parseFeatures []string) map[string]struct{} {
	set := make(map[string]struct{}, len(parseFeatures))
	for _, parseFeature := range normalizeScaffoldFeatureKeys(parseFeatures) {
		set[parseFeature] = struct{}{}
	}
	return set
}

func scaffoldHasFeature(parseFeatures []string, parseKey string) bool {
	_, parseExists := scaffoldFeatureSet(parseFeatures)[strings.ToLower(strings.TrimSpace(parseKey))]
	return parseExists
}

func renderScaffoldFeatureCards(parseFeatures []string) string {
	parseNormalized := normalizeScaffoldFeatureKeys(parseFeatures)
	if len(parseNormalized) == 0 {
		return `				html.Div(html.Props{Class: "feature-card"},
					html.Span(html.Props{Class: "feature-tag"}, html.Text("Core Starter")),
					html.P(html.Props{Class: "feature-copy"}, html.Text("No optional capabilities were selected for this starter scaffold.")),
				),`
	}

	parseLines := make([]string, 0, len(parseNormalized))
	for _, parseFeature := range parseNormalized {
		parseDescriptor := scaffoldFeatureDescriptorFor(parseFeature)
		parseLines = append(parseLines, fmt.Sprintf(`				html.Div(html.Props{Class: "feature-card"},
					html.Span(html.Props{Class: "feature-tag"}, html.Text(%q)),
					html.P(html.Props{Class: "feature-copy"}, html.Text(%q)),
				),`, parseDescriptor.Label, parseDescriptor.Description))
	}
	return strings.Join(parseLines, "\n")
}

func renderScaffoldCapabilityState(parseFeatures []string) string {
	parseSelected := scaffoldFeatureSet(parseFeatures)
	parseBlocks := []string{}

	if _, parseOk := parseSelected["router"]; parseOk {
		parseBlocks = append(parseBlocks, `	parseRouteSegment := ui.UseState("home")
	parseShowHomeRoute := ui.UseEvent(func() {
		parseRouteSegment.Set("home")
	})
	parseShowDashboardRoute := ui.UseEvent(func() {
		parseRouteSegment.Set("dashboard")
	})
	parseShowSettingsRoute := ui.UseEvent(func() {
		parseRouteSegment.Set("settings")
	})
	parseCurrentRoute := parseRouteSegment.Get()`)
	}

	if _, parseOk2 := parseSelected["forms"]; parseOk2 {
		parseBlocks = append(parseBlocks, `	parseFormSubmitted := ui.UseState(false)
	parseSubmitStarterForm := ui.UseEvent(func() {
		parseFormSubmitted.Set(true)
	})
	resetStarterForm := ui.UseEvent(func() {
		parseFormSubmitted.Set(false)
	})
	parseFormStatus := "Draft not submitted yet."
	if parseFormSubmitted.Get() {
		parseFormStatus = "Last submit marked complete."
	}`)
	}

	if _, parseOk3 := parseSelected["fetch"]; parseOk3 {
		parseBlocks = append(parseBlocks, `	parseDataStatus := ui.UseState("idle")
	parseRefreshStarterData := ui.UseEvent(func() {
		parseDataStatus.Update(func(previous string) string {
			switch previous {
			case "idle":
				return "loading"
			case "loading":
				return "ready"
			default:
				return "refreshing"
			}
		})
	})
	parseCurrentDataStatus := parseDataStatus.Get()`)
	}

	if _, parseOk4 := parseSelected["state"]; parseOk4 {
		parseBlocks = append(parseBlocks, `	parseTeamCount := ui.UseState(3)
	parseAddTeammate := ui.UseEvent(func() {
		parseTeamCount.Update(func(previous int) int { return previous + 1 })
	})
	parseCurrentTeamCount := parseTeamCount.Get()`)
	}

	if len(parseBlocks) == 0 {
		return ""
	}
	return strings.Join(parseBlocks, "\n\n")
}

func renderScaffoldCapabilityWidgets(parseFeatures []string) string {
	parseSelected := scaffoldFeatureSet(parseFeatures)
	parseWidgets := []string{}

	if _, parseOk := parseSelected["router"]; parseOk {
		parseWidgets = append(parseWidgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Routing")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(fmt.Sprintf("Active route: %s", parseCurrentRoute))),
				html.Div(html.Props{Class: "capability-actions"},
					html.Button(html.Props{Class: "capability-button", OnClick: parseShowHomeRoute}, html.Text("Home")),
					html.Button(html.Props{Class: "capability-button", OnClick: parseShowDashboardRoute}, html.Text("Dashboard")),
					html.Button(html.Props{Class: "capability-button", OnClick: parseShowSettingsRoute}, html.Text("Settings")),
				),
			),`)
	}

	if _, parseOk2 := parseSelected["fetch"]; parseOk2 {
		parseWidgets = append(parseWidgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Async Data")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(fmt.Sprintf("Data status: %s", parseCurrentDataStatus))),
				html.Button(html.Props{Class: "capability-button", OnClick: parseRefreshStarterData}, html.Text("Advance data sync state")),
			),`)
	}

	if _, parseOk3 := parseSelected["state"]; parseOk3 {
		parseWidgets = append(parseWidgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Shared State")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(fmt.Sprintf("Team members tracked: %d", parseCurrentTeamCount))),
				html.Button(html.Props{Class: "capability-button", OnClick: parseAddTeammate}, html.Text("Add teammate")),
			),`)
	}

	if _, parseOk4 := parseSelected["forms"]; parseOk4 {
		parseWidgets = append(parseWidgets, `			html.Form(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Forms")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(parseFormStatus)),
				html.Label(html.Props{}, html.Text("Email")),
				html.Input(html.Props{Type: "email", Placeholder: "you@example.com", Class: "capability-input"}),
				html.Div(html.Props{Class: "capability-actions"},
					html.Button(html.Props{Class: "capability-button", Type: "button", OnClick: parseSubmitStarterForm}, html.Text("Submit")),
					html.Button(html.Props{Class: "capability-button", Type: "button", OnClick: resetStarterForm}, html.Text("Reset")),
				),
			),`)
	}

	if _, parseOk5 := parseSelected["ssr"]; parseOk5 {
		parseWidgets = append(parseWidgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Server Rendering")),
				html.P(html.Props{Class: "capability-copy"}, html.Text("This preset is intended for request-time HTML rendering with launcher-managed release defaults.")),
			),`)
	}

	if _, parseOk6 := parseSelected["hydration"]; parseOk6 {
		parseWidgets = append(parseWidgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Hydration")),
				html.P(html.Props{Class: "capability-copy"}, html.Text("Hydration-ready app ownership is expected so server output and browser interactivity stay aligned.")),
			),`)
	}

	if len(parseWidgets) == 0 {
		return ""
	}
	return strings.Join(parseWidgets, "\n")
}

func renderScaffoldFeatureMatrix(parseSelection startSelection) string {
	parseNormalized := startSelectionFeatures(parseSelection)
	parseSelected := scaffoldFeatureSet(parseNormalized)
	var parseBuilder strings.Builder
	parseBuilder.WriteString("# Starter Feature Matrix\n\n")
	parseBuilder.WriteString(fmt.Sprintf("Preset: `%s` (%s)\n\n", parseSelection.Preset.Key, parseSelection.Preset.Name))
	if len(parseSelection.EnabledEnterpriseSections) > 0 {
		parseBuilder.WriteString("Enterprise plugin sections selected:\n\n")
		for _, parseSection := range parseSelection.EnabledEnterpriseSections {
			parseBuilder.WriteString(fmt.Sprintf("- %s\n", parseSection))
		}
		parseBuilder.WriteString("\n")
	}
	parseBuilder.WriteString("This file is generated from the selected scaffold capabilities.\n\n")
	for _, parseDescriptor := range scaffoldFeatureCatalog {
		parseMarker := "[ ]"
		if _, parseOk := parseSelected[parseDescriptor.Key]; parseOk {
			parseMarker = "[x]"
		}
		parseBuilder.WriteString(fmt.Sprintf("- %s `%s`: %s\n", parseMarker, parseDescriptor.Key, parseDescriptor.Description))
	}
	if len(parseNormalized) > 0 {
		parseBuilder.WriteString("\nSelected capability order:\n\n")
		for _, parseFeature := range parseNormalized {
			parseDescriptor2 := scaffoldFeatureDescriptorFor(parseFeature)
			parseBuilder.WriteString(fmt.Sprintf("- `%s`: %s\n", parseDescriptor2.Key, parseDescriptor2.Label))
		}
	}
	parseBuilder.WriteString("\nGenerated starter output is intentionally disposable; treat this scaffold as a starting point you can edit or replace freely.\n")
	return parseBuilder.String()
}

func renderScaffoldBrowserTestREADME(parseSelection startSelection) string {
	return fmt.Sprintf("# Browser smoke tests for %s\n\nUse this folder for Playwright-Go smoke tests that validate starter boot and basic user interactions.\n\nRun from the generated project root:\n\n```powershell\ngo test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v\n```\n", parseSelection.ProjectName)
}

func renderScaffoldBrowserSmokeTest(parseSelection startSelection) string {
	return fmt.Sprintf(`//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"net/url"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestMainSuite(t *testing.T) {
	if err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}); err != nil {
		t.Fatalf("install playwright-go chromium: %%v", err)
	}

	pw, err := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if err != nil {
		t.Fatalf("run playwright-go: %%v", err)
	}
	defer func() {
		if stopErr := pw.Stop(); stopErr != nil {
			t.Errorf("stop playwright-go: %%v", stopErr)
		}
	}()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		t.Fatalf("launch chromium: %%v", err)
	}
	defer func() {
		if closeErr := browser.Close(); closeErr != nil {
			t.Errorf("close chromium: %%v", closeErr)
		}
	}()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("create browser page: %%v", err)
	}

	projectName := %q
	html := "<html><body><h1 id='starter-heading'>" + projectName + "</h1></body></html>"
	dataURL := "data:text/html," + url.PathEscape(html)
	if _, err := page.Goto(dataURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("goto starter smoke page: %%v", err)
	}

	heading, err := page.TextContent("#starter-heading")
	if err != nil {
		t.Fatalf("read starter heading: %%v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(heading), projectName) {
		t.Fatalf("unexpected starter heading: got %%q want %%q", heading, projectName)
	}
}
`, parseSelection.ProjectName)
}

func renderScaffoldGoStringList(parseValues []string) string {
	if len(parseValues) == 0 {
		return "[]string{}"
	}
	parseLines := make([]string, 0, len(parseValues))
	for _, parseValue := range parseValues {
		if strings.Contains(parseValue, "\"") || strings.Contains(parseValue, "\\") {
			if strings.Contains(parseValue, "`") {
				parseLines = append(parseLines, fmt.Sprintf("\t\t%q,", parseValue))
				continue
			}
			parseLines = append(parseLines, fmt.Sprintf("\t\t`%s`,", parseValue))
			continue
		}
		parseLines = append(parseLines, fmt.Sprintf("\t\t%q,", parseValue))
	}
	return "[]string{\n" + strings.Join(parseLines, "\n") + "\n\t}"
}

func renderScaffoldFeatureBaselineTest(parseSelection startSelection) string {
	parseFeatures := startSelectionFeatures(parseSelection)
	parseMainExpectations := []string{}
	parseExtraPaths := []string{}
	if scaffoldHasFeature(parseFeatures, "router") {
		parseMainExpectations = append(parseMainExpectations, `html.Text("Routing")`, `Active route: %s`)
	}
	if scaffoldHasFeature(parseFeatures, "forms") {
		parseMainExpectations = append(parseMainExpectations, `html.Text("Forms")`, `Last submit marked complete.`)
	}
	if scaffoldHasFeature(parseFeatures, "fetch") {
		parseMainExpectations = append(parseMainExpectations, `html.Text("Async Data")`, `Data status: %s`)
	}
	if scaffoldHasFeature(parseFeatures, "browser-tests") {
		parseExtraPaths = append(parseExtraPaths, "test/playwrightgo/smoke_test.go")
	}

	parseExtraAssertions := ""
	if len(parseMainExpectations) > 0 || len(parseExtraPaths) > 0 {
		parseExtraAssertions = fmt.Sprintf(`
func TestStarterFeatureSpecificPlaceholders(t *testing.T) {
	mainSource := readStarterFile(t, "main.go")
	for _, expected := range %s {
		if !strings.Contains(mainSource, expected) {
			t.Fatalf("expected generated main.go to contain %%q", expected)
		}
	}
	for _, relativePath := range %s {
		if _, err := os.Stat(filepath.FromSlash(relativePath)); err != nil {
			t.Fatalf("expected generated scaffold path %%q: %%v", relativePath, err)
		}
	}
}
`, renderScaffoldGoStringList(parseMainExpectations), renderScaffoldGoStringList(parseExtraPaths))
	}

	return fmt.Sprintf(`package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type generatedStarterMetadata struct {
	Preset struct {
		Features []string `+"`json:\"features,omitempty\"`"+`
	} `+"`json:\"preset\"`"+`
	Enterprise struct {
		Features []string `+"`json:\"features,omitempty\"`"+`
	} `+"`json:\"enterprise\"`"+`
}

func readStarterFile(t *testing.T, relativePath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.FromSlash(relativePath))
	if err != nil {
		t.Fatalf("read %%s: %%v", relativePath, err)
	}
	return string(data)
}

func starterHasFeature(features []string, target string) bool {
	for _, feature := range features {
		if strings.EqualFold(strings.TrimSpace(feature), strings.TrimSpace(target)) {
			return true
		}
	}
	return false
}

func TestStarterMetadataIncludesSelectedFeatures(t *testing.T) {
	var metadata generatedStarterMetadata
	if err := json.Unmarshal([]byte(readStarterFile(t, "gwc-start.json")), &metadata); err != nil {
		t.Fatalf("unmarshal metadata: %%v", err)
	}
	features := append([]string{}, metadata.Preset.Features...)
	features = append(features, metadata.Enterprise.Features...)
	for _, expected := range %s {
		if !starterHasFeature(features, expected) {
			t.Fatalf("expected scaffold metadata to record selected feature %%q, got %%v", expected, features)
		}
	}
}

func TestStarterFeatureMatrixMarksSelectedFeatures(t *testing.T) {
	matrix := readStarterFile(t, "FEATURE_MATRIX.md")
	for _, feature := range %s {
		if !strings.Contains(matrix, "- [x] "+string(rune(96))+feature+string(rune(96))) {
			t.Fatalf("expected feature matrix to mark %%q as selected", feature)
		}
	}
}
%s
`, renderScaffoldGoStringList(parseFeatures), renderScaffoldGoStringList(parseFeatures), parseExtraAssertions)
}

func renderScaffoldGitHubActionsWorkflow(parseSelection startSelection) string {
	return fmt.Sprintf(`name: %s CI

on:
  push:
    branches:
      - main
      - master
  pull_request:
  workflow_dispatch:

jobs:
  test-and-build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v6

      - name: Setup Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod

      - name: Run Baseline Go Tests
        run: go test ./...

      - name: Build WASM Entry
        run: |
          mkdir -p bin
          go build -o %s .
        env:
          GOOS: js
          GOARCH: wasm
`, parseSelection.ProjectName, scaffoldWASMOutputPath())
}
