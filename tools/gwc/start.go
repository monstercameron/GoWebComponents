package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
	Preset        scaffoldPresetMetadata     `json:"preset,omitempty"`
	Enterprise    scaffoldEnterpriseMetadata `json:"enterprise,omitempty"`
	Ownership     scaffoldOwnershipMetadata  `json:"ownership,omitempty"`
	Tooling       scaffoldToolingMetadata    `json:"tooling,omitempty"`
}

const currentScaffoldMetadataSchemaVersion = 1

var scaffoldRel = filepath.Rel

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

var resolveWasmExecGoRoot = runtime.GOROOT

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
	for parseKey, parseValue := range parseValues {
		parseCloned[parseKey] = parseValue
	}
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
	for _, parseLine := range strings.Split(string(parseContent), "\n") {
		parseTrimmed := strings.TrimSpace(parseLine)
		if strings.HasPrefix(parseTrimmed, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(parseTrimmed, "module ")), nil
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
	{Key: "forms", Label: "Forms", Description: "Form workflow placeholders are included so teams can wire typed form state quickly."},
	{Key: "fetch", Label: "Fetch", Description: "Async resource ownership is called out for data-loading and mutation setup."},
	{Key: "state", Label: "State", Description: "Shared state ownership is planned as a first-class concern in this scaffold."},
	{Key: "devtools", Label: "Devtools", Description: "Devtools adoption is surfaced as part of the starter capability model."},
	{Key: "hot-reload", Label: "Hot Reload", Description: "State-preserving local reload is included as the recommended inner-loop path."},
	{Key: "browser-tests", Label: "Browser Tests", Description: "Playwright-Go smoke-test placeholders are generated under test/playwrightgo."},
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
		return "nil"
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

func renderScaffoldExtraFiles(parseSelection startSelection) map[string][]byte {
	parseFiles := map[string][]byte{
		"FEATURE_MATRIX.md":        []byte(renderScaffoldFeatureMatrix(parseSelection)),
		"starter_test.go":          []byte(renderScaffoldFeatureBaselineTest(parseSelection)),
		".github/workflows/ci.yml": []byte(renderScaffoldGitHubActionsWorkflow(parseSelection)),
	}
	if scaffoldHasFeature(startSelectionFeatures(parseSelection), "browser-tests") {
		parseFiles["test/playwrightgo/README.md"] = []byte(renderScaffoldBrowserTestREADME(parseSelection))
		parseFiles["test/playwrightgo/smoke_test.go"] = []byte(renderScaffoldBrowserSmokeTest(parseSelection))
	}
	return parseFiles
}

func renderScaffoldMain(parseSelection startSelection, parseRepoModulePath string) string {
	parseNormalizedFeatures := startSelectionFeatures(parseSelection)
	parseFeatureList := strings.Join(parseNormalizedFeatures, ", ")
	if parseFeatureList == "" {
		parseFeatureList = "core-only"
	}
	parseFeatureCards := renderScaffoldFeatureCards(parseNormalizedFeatures)
	parseCapabilityState := renderScaffoldCapabilityState(parseNormalizedFeatures)
	parseCapabilityWidgets := renderScaffoldCapabilityWidgets(parseNormalizedFeatures)
	return fmt.Sprintf(`//go:build js && wasm
// +build js,wasm

package main

import (
	"fmt"

	%q
	%q
	%q
)

func App() ui.Node {
	parseCount := ui.UseState(0)
	parseCurrentCount := parseCount.Get()

	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(previous int) int { return previous + 1 })
	})

%s

	return html.Div(html.Props{Class: "shell"},
		html.Div(html.Props{Class: "panel"},
			html.Div(html.Props{Class: "brand-row"},
				html.Div(html.Props{Class: "brand-mark"}, html.Text("GWC")),
				html.Div(html.Props{},
					html.P(html.Props{Class: "eyebrow"}, html.Text("GoWebComponents Placeholder Brand")),
					html.P(html.Props{Class: "brand-copy"}, html.Text("Dark-mode starter scaffold generated by gwc start")),
				),
			),
			html.H1(html.Props{}, html.Text(%q)),
			html.P(html.Props{Class: "lede"}, html.Text(%q)),
			html.Div(html.Props{Class: "meta-grid"},
				html.Div(html.Props{Class: "meta-card"},
					html.Span(html.Props{Class: "meta-label"}, html.Text("Author")),
					html.Strong(html.Props{}, html.Text(%q)),
				),
				html.Div(html.Props{Class: "meta-card"},
					html.Span(html.Props{Class: "meta-label"}, html.Text("Version")),
					html.Strong(html.Props{}, html.Text(%q)),
				),
				html.Div(html.Props{Class: "meta-card"},
					html.Span(html.Props{Class: "meta-label"}, html.Text("Preset")),
					html.Strong(html.Props{}, html.Text(%q)),
				),
				html.Div(html.Props{Class: "meta-card"},
					html.Span(html.Props{Class: "meta-label"}, html.Text("Module")),
					html.Strong(html.Props{}, html.Text(%q)),
				),
			),
			html.P(html.Props{Class: "lede"}, html.Text(%q)),
			html.P(html.Props{Class: "meta"}, html.Text(%q)),
			html.H2(html.Props{Class: "feature-heading"}, html.Text("Starter capability matrix")),
			html.Div(html.Props{Class: "feature-grid"},
%s
			),
%s
			html.Div(html.Props{Class: "counter"}, html.Text(fmt.Sprintf("Count: %%d", parseCurrentCount))),
			html.Button(html.Props{OnClick: parseIncrement, Class: "button"}, html.Text("Increment")),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
`, parseRepoModulePath+"/html", parseRepoModulePath+"/ui", parseRepoModulePath+"/utils", parseCapabilityState, parseSelection.ProjectName, parseSelection.Description, parseSelection.Author, parseSelection.Version, parseSelection.Preset.Name, parseSelection.ModulePath, parseSelection.Preset.Description, "Features: "+parseFeatureList, parseFeatureCards, parseCapabilityWidgets)
}

func renderScaffoldHTML(parseSelection startSelection) string {
	parseTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>__GWC_PROJECT_TITLE__</title>
	<script src="./wasm_exec.js"></script>
	<style>
		:root { color-scheme: dark; }
		* { box-sizing: border-box; }
		body {
			margin: 0;
			font-family: "Segoe UI", sans-serif;
			background:
				radial-gradient(circle at top, rgba(97, 218, 251, 0.18), transparent 32%),
				linear-gradient(160deg, #050816 0%, #0d1326 48%, #111c2b 100%);
			color: #eef4ff;
		}
		.shell {
			min-height: 100vh;
			display: flex;
			align-items: center;
			justify-content: center;
			padding: 24px;
		}
		.panel {
			width: min(560px, 100%);
			background: rgba(8, 13, 25, 0.84);
			border: 1px solid rgba(97, 218, 251, 0.14);
			border-radius: 24px;
			padding: 32px;
			box-shadow: 0 28px 90px rgba(0, 0, 0, 0.42);
			backdrop-filter: blur(18px);
		}
		.brand-row {
			display: flex;
			align-items: center;
			gap: 14px;
			margin-bottom: 18px;
		}
		.brand-mark {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			width: 48px;
			height: 48px;
			border-radius: 14px;
			background: linear-gradient(135deg, #61dafb 0%, #35b4d8 100%);
			color: #03131d;
			font-weight: 800;
			letter-spacing: 0.08em;
		}
		.brand-copy {
			margin: 4px 0 0;
			font-size: 0.9rem;
			color: #8ca2c7;
		}
		.eyebrow {
			margin: 0 0 12px;
			text-transform: uppercase;
			letter-spacing: 0.18em;
			font-size: 12px;
			color: #61dafb;
		}
		h1 {
			margin: 0 0 12px;
			font-size: clamp(2rem, 6vw, 3.2rem);
			line-height: 1;
			color: #f7fbff;
		}
		.lede {
			margin: 0 0 12px;
			font-size: 1.05rem;
			color: #d5e0f5;
		}
		.meta {
			margin: 0 0 24px;
			font-size: 0.95rem;
			color: #8ca2c7;
		}
		.feature-heading {
			margin: 0 0 10px;
			font-size: 1rem;
			text-transform: uppercase;
			letter-spacing: 0.12em;
			color: #61dafb;
		}
		.feature-grid {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 10px;
			margin: 0 0 20px;
		}
		.feature-card {
			padding: 12px;
			border-radius: 14px;
			background: rgba(97, 218, 251, 0.06);
			border: 1px solid rgba(97, 218, 251, 0.14);
		}
		.feature-tag {
			display: block;
			font-size: 0.76rem;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.08em;
			color: #b9f0ff;
			margin-bottom: 6px;
		}
		.feature-copy {
			margin: 0;
			font-size: 0.84rem;
			color: #c6d7f2;
			line-height: 1.4;
		}
		.capability {
			margin: 0 0 14px;
			padding: 12px;
			border-radius: 14px;
			border: 1px solid rgba(97, 218, 251, 0.18);
			background: rgba(5, 15, 33, 0.66);
		}
		.capability-title {
			display: block;
			font-size: 0.74rem;
			font-weight: 700;
			text-transform: uppercase;
			letter-spacing: 0.12em;
			color: #9be8ff;
			margin-bottom: 6px;
		}
		.capability-copy {
			margin: 0 0 10px;
			font-size: 0.9rem;
			color: #dce8ff;
		}
		.capability-actions {
			display: flex;
			flex-wrap: wrap;
			gap: 8px;
		}
		.capability-button {
			border: 1px solid rgba(97, 218, 251, 0.28);
			border-radius: 999px;
			padding: 8px 14px;
			background: rgba(97, 218, 251, 0.1);
			color: #d8f6ff;
			font: inherit;
			font-size: 0.84rem;
			font-weight: 600;
			cursor: pointer;
		}
		.capability-input {
			width: 100%;
			margin-bottom: 10px;
			padding: 10px 12px;
			border-radius: 10px;
			border: 1px solid rgba(97, 218, 251, 0.22);
			background: rgba(3, 10, 26, 0.92);
			color: #f1f7ff;
			font: inherit;
		}
		.meta-grid {
			display: grid;
			grid-template-columns: repeat(2, minmax(0, 1fr));
			gap: 12px;
			margin: 0 0 20px;
		}
		.meta-card {
			padding: 14px 16px;
			border-radius: 16px;
			background: rgba(97, 218, 251, 0.06);
			border: 1px solid rgba(97, 218, 251, 0.12);
		}
		.meta-label {
			display: block;
			font-size: 0.72rem;
			text-transform: uppercase;
			letter-spacing: 0.14em;
			color: #8ca2c7;
			margin-bottom: 6px;
		}
		strong {
			color: #f7fbff;
		}
		.counter {
			font-size: 2rem;
			font-weight: 700;
			margin-bottom: 20px;
			color: #f7fbff;
		}
		.button {
			border: 0;
			border-radius: 999px;
			padding: 14px 22px;
			background: linear-gradient(135deg, #61dafb 0%, #35b4d8 100%);
			color: #03131d;
			font: inherit;
			font-weight: 700;
			cursor: pointer;
			box-shadow: 0 14px 30px rgba(53, 180, 216, 0.28);
		}
		.button:hover {
			filter: brightness(1.06);
		}
		@media (max-width: 640px) {
			.meta-grid {
				grid-template-columns: 1fr;
			}
			.feature-grid {
				grid-template-columns: 1fr;
			}
			.brand-row {
				align-items: flex-start;
			}
		}
		#boot-error {
			display: none;
			margin-top: 18px;
			padding: 12px 14px;
			border-radius: 12px;
			background: rgba(255, 107, 107, 0.16);
			color: #ffd4d4;
			border: 1px solid rgba(255, 107, 107, 0.24);
		}
	</style>
</head>
<body>
	<div id="app"></div>
	<div id="boot-error"></div>
	<script>
		const go = new Go();
		const errorBox = document.getElementById('boot-error');
		WebAssembly.instantiateStreaming(fetch('./__GWC_WASM_PATH__'), go.importObject)
			.then(result => go.run(result.instance))
			.catch(error => {
				errorBox.style.display = 'block';
				errorBox.textContent = 'Failed to start wasm app: ' + String(error);
				console.error(error);
			});
	</script>
</body>
</html>
`
	parseRendered := strings.ReplaceAll(parseTemplate, "__GWC_PROJECT_TITLE__", parseSelection.ProjectName)
	return strings.ReplaceAll(parseRendered, "__GWC_WASM_PATH__", scaffoldWASMOutputPath())
}

func renderScaffoldMetadata(parseSelection startSelection) string {
	parsePayload := defaultScaffoldMetadata(parseSelection)
	parseEncoded, parseErr := scaffoldMarshalIndent(parsePayload, "", "  ")
	if parseErr != nil {
		return "{}\n"
	}
	return string(parseEncoded) + "\n"
}

func renderScaffoldREADME(parseSelection startSelection) string {
	parseNormalizedFeatures := startSelectionFeatures(parseSelection)
	var parseBuilder strings.Builder
	parseBuilder.WriteString(fmt.Sprintf("# %s\n\n", parseSelection.ProjectName))
	parseBuilder.WriteString(fmt.Sprintf("%s\n\n", parseSelection.Description))
	parseBuilder.WriteString(fmt.Sprintf("- Author: %s\n", parseSelection.Author))
	parseBuilder.WriteString(fmt.Sprintf("- Version: %s\n", parseSelection.Version))
	parseBuilder.WriteString(fmt.Sprintf("- Preset: %s\n\n", parseSelection.Preset.Key))

	parseBuilder.WriteString("## Selected Capabilities\n\n")
	if len(parseNormalizedFeatures) == 0 {
		parseBuilder.WriteString("- none\n")
	} else {
		for _, parseFeature := range parseNormalizedFeatures {
			parseBuilder.WriteString(fmt.Sprintf("- `%s`\n", parseFeature))
		}
	}
	parseBuilder.WriteString("\n")
	if len(parseSelection.EnabledEnterpriseSections) > 0 {
		parseBuilder.WriteString("## Enterprise Extensions\n\n")
		parseBuilder.WriteString("These optional sections came from enterprise launcher plugins and were selected during `gwc start`.\n\n")
		for _, parseSection := range parseSelection.EnabledEnterpriseSections {
			parseBuilder.WriteString(fmt.Sprintf("- %s\n", parseSection))
		}
		parseBuilder.WriteString("\n")
	}
	parseBuilder.WriteString("The full generated capability contract lives in `FEATURE_MATRIX.md`.\n\n")

	parseBuilder.WriteString("## Starter Output Rules\n\n")
	parseBuilder.WriteString("- keep generated files small, readable, and conventionally organized\n")
	parseBuilder.WriteString("- treat this scaffold as disposable starter code; edit or replace it when the app shape changes\n")
	parseBuilder.WriteString("- keep app code on public framework packages instead of importing repo-internal build tooling\n")
	parseBuilder.WriteString("- avoid coupling to monorepo-only paths so this project can live independently\n\n")

	parseBuilder.WriteString("## Run\n\n")
	parseBuilder.WriteString("From the GoWebComponents repo root:\n\n")
	parseBuilder.WriteString("```powershell\n")
	parseBuilder.WriteString(fmt.Sprintf("go run ./tools/gwc dev -app %q -root %q -html %q -wasm %q\n", filepath.Join(parseSelection.TargetDir, "main.go"), parseSelection.TargetDir, filepath.Join(parseSelection.TargetDir, "index.html"), scaffoldWASMOutputPath()))
	parseBuilder.WriteString("```\n")

	parseBuilder.WriteString("\n## Verify\n\n")
	parseBuilder.WriteString("From the generated project directory:\n\n")
	parseBuilder.WriteString("```powershell\n")
	parseBuilder.WriteString("go test ./...\n")
	parseBuilder.WriteString("```\n")
	if scaffoldHasFeature(parseNormalizedFeatures, "browser-tests") {
		parseBuilder.WriteString("\nFor browser tests, start from `test/playwrightgo/smoke_test.go` and run:\n\n")
		parseBuilder.WriteString("```powershell\n")
		parseBuilder.WriteString("go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v\n")
		parseBuilder.WriteString("```\n")
	}
	return parseBuilder.String()
}
