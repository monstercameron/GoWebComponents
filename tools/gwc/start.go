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

func (l launcher) runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	skipPrereqChecks := fs.Bool("skip-prereq-checks", false, "Skip start-time prerequisite checks (Go, runtime assets, and preset-required browser tooling)")
	skipTidy := fs.Bool("skip-tidy", false, "Skip go mod tidy after scaffold generation")
	skipRuntimeAssets := fs.Bool("skip-runtime-assets", false, "Skip copying runtime assets such as wasm_exec.js into the generated scaffold")
	projectMode := fs.String("mode", string(scaffoldProjectModeStandalone), "Scaffold mode: standalone or contributor-linked")
	initGit := fs.Bool("init-git", false, "Initialize a fresh git repository in the generated scaffold root")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	normalizedProjectMode, ok := normalizeScaffoldProjectMode(*projectMode)
	if !ok {
		return fmt.Errorf("unknown scaffold mode %q", *projectMode)
	}
	if err := startTerminalValidator(isInteractiveFile(os.Stdin), isInteractiveFile(os.Stdout)); err != nil {
		return err
	}
	sections, err := startResolveScaffoldSections(l.repoRoot)
	if err != nil {
		return err
	}
	startEnterpriseScaffoldSections = sections
	previousProjectMode := startDefaultProjectMode
	startDefaultProjectMode = normalizedProjectMode
	defer func() {
		startEnterpriseScaffoldSections = nil
		startDefaultProjectMode = previousProjectMode
	}()
	selection, err := startSelectionRunner()
	if err != nil {
		return err
	}
	if selection == nil {
		return nil
	}

	selection.ProjectMode = normalizedProjectMode
	selection.InitGit = *initGit
	selection.SkipGoModTidy = *skipTidy
	selection.SkipRuntimeAssets = *skipRuntimeAssets

	if !*skipPrereqChecks {
		if err := l.validateStartPrerequisites(*selection); err != nil {
			_, postErr := startPostRunner(*selection, nil, err)
			if postErr != nil {
				return postErr
			}
			return nil
		}
	}

	result, err := startGenerateScaffold(l, *selection)
	if err != nil {
		_, postErr := startPostRunner(*selection, nil, err)
		if postErr != nil {
			return postErr
		}
		return nil
	}
	if selection.InitGit {
		if err := startInitGit(result.TargetDir); err != nil {
			_, postErr := startPostRunner(*selection, &result, err)
			if postErr != nil {
				return postErr
			}
			return nil
		}
	}

	postResult, err := startPostRunner(*selection, &result, nil)
	if err != nil {
		return err
	}
	if postResult == nil || !postResult.RunDev {
		return nil
	}

	return startRunDev(l, devArgsFromScaffold(result))
}

func resolveStartScaffoldPluginSections(repoRoot string) ([]launcherPluginScaffoldSection, error) {
	results, err := runLauncherPluginsForCapability("scaffold_feature", "start", nil, repoRoot, launcherActiveEnterpriseSources)
	if err != nil {
		return nil, err
	}
	sections := []launcherPluginScaffoldSection{}
	for _, result := range results {
		sections = append(sections, result.Response.ScaffoldSections...)
	}
	return normalizeStartEnterpriseSections(sections), nil
}

func (l launcher) runBootstrap(args []string) error {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	examplesMode := fs.Bool("examples", false, "Run the examples catalog flow after prerequisite checks")
	host := fs.String("host", defaultHost, "Host used by bootstrap prerequisite checks")
	port := fs.String("port", "8080", "Port used by bootstrap prerequisite checks")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	report := l.buildDoctorReport(doctorConfig{host: *host, port: *port, json: false})
	printDoctorReport(report)
	if !report.OK {
		return errors.New("bootstrap blocked by doctor checks")
	}

	if *examplesMode {
		return runExamplesCommand(l, fs.Args())
	}
	return runStartCommand(l, fs.Args())
}

func (l launcher) validateStartPrerequisites(selection startSelection) error {
	checks := []doctorCheck{
		buildDoctorToolCheck("go", "Go toolchain", "version", "Install Go 1.25+ and ensure `go` is on PATH before running gwc start."),
	}
	if !selection.SkipRuntimeAssets {
		checks = append(checks, buildDoctorWasmExecCheck())
	}
	if scaffoldHasFeature(startSelectionFeatures(selection), "browser-tests") {
		checks = append(checks,
			buildDoctorPlaywrightCheck(l.repoRoot),
		)
	}

	failed := []string{}
	warnings := []string{}
	for _, check := range checks {
		switch check.Status {
		case "fail":
			failed = append(failed, fmt.Sprintf("%s: %s", check.Name, check.Summary))
		case "warn":
			warnings = append(warnings, fmt.Sprintf("%s: %s", check.Name, check.Summary))
		}
	}
	if len(warnings) > 0 {
		fmt.Println("GWC start prerequisites (warnings):")
		for _, warning := range warnings {
			fmt.Printf("  - %s\n", warning)
		}
	}
	if len(failed) == 0 {
		return nil
	}
	return fmt.Errorf("start prerequisites failed:\n  - %s", strings.Join(failed, "\n  - "))
}

type scaffoldResult struct {
	TargetDir string
	AppPath   string
	HTMLPath  string
}

func validateStartTerminal(stdinIsTerminal bool, stdoutIsTerminal bool) error {
	if stdinIsTerminal && stdoutIsTerminal {
		return nil
	}
	return errors.New("start requires an interactive terminal with TUI support; run `go run ./tools/gwc start` from a normal shell session")
}

func isInteractiveFile(file *os.File) bool {
	if file == nil {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func devArgsFromScaffold(result scaffoldResult) []string {
	return []string{
		"-app", result.AppPath,
		"-root", result.TargetDir,
		"-html", result.HTMLPath,
		"-wasm", scaffoldWASMOutputPath(),
	}
}

func scaffoldWASMOutputPath() string {
	return "main.wasm"
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

func cloneResolutionTrace(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func setResolutionSource(values map[string]string, key string, source string) map[string]string {
	if values == nil {
		values = map[string]string{}
	}
	if strings.TrimSpace(key) != "" && strings.TrimSpace(source) != "" {
		values[key] = source
	}
	return values
}

func printResolutionTrace(trace map[string]string, keys []string, indent string) {
	if len(trace) == 0 {
		return
	}
	for _, key := range keys {
		if source, ok := trace[key]; ok && strings.TrimSpace(source) != "" {
			fmt.Printf("%s%s source: %s\n", indent, key, source)
		}
	}
}

func (l launcher) generateStartScaffold(selection startSelection) (scaffoldResult, error) {
	targetDir := filepath.Clean(selection.TargetDir)
	if err := validateGeneratedTargetDir(targetDir); err != nil {
		return scaffoldResult{}, err
	}
	if err := ensureEmptyDir(targetDir); err != nil {
		return scaffoldResult{}, err
	}
	repoModulePath, err := l.readRepoModulePath()
	if err != nil {
		return scaffoldResult{}, err
	}
	return l.generateScaffoldProject(scaffoldPlan{
		Selection:         selection,
		GoMod:             renderScaffoldGoMod(selection, repoModulePath, l.repoRoot),
		MainGo:            renderScaffoldMain(selection, repoModulePath),
		HTML:              renderScaffoldHTML(selection),
		README:            renderScaffoldREADME(selection),
		Metadata:          defaultScaffoldMetadata(selection),
		ExtraFiles:        renderScaffoldExtraFiles(selection),
		SkipGoModTidy:     selection.SkipGoModTidy,
		SkipRuntimeAssets: selection.SkipRuntimeAssets,
	})
}

func (l launcher) seedScaffoldGoSum(targetDir string) error {
	rootGoSumPath := filepath.Join(l.repoRoot, "go.sum")
	goSumBytes, err := os.ReadFile(rootGoSumPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read repo go.sum: %w", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "go.sum"), goSumBytes, 0644); err != nil {
		return fmt.Errorf("write scaffold go.sum: %w", err)
	}
	return nil
}

func (l launcher) tidyScaffoldModule(targetDir string) error {
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = targetDir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return fmt.Errorf("prepare scaffold module: %w", err)
		}
		return fmt.Errorf("prepare scaffold module: %s", trimmed)
	}
	return nil
}

func ensureEmptyDir(targetDir string) error {
	if info, err := os.Stat(targetDir); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("target path exists and is not a directory: %s", targetDir)
		}
		entries, readErr := os.ReadDir(targetDir)
		if readErr != nil {
			return fmt.Errorf("read target directory: %w", readErr)
		}
		if len(entries) > 0 {
			return fmt.Errorf("target directory already exists and is not empty: %s", targetDir)
		}
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect target directory: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}
	return nil
}

func (l launcher) readRepoModulePath() (string, error) {
	goModPath := filepath.Join(l.repoRoot, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return "", fmt.Errorf("read repo go.mod: %w", err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "module ")), nil
		}
	}
	return "", fmt.Errorf("repo module path not found in %s", goModPath)
}

func resolveWasmExecPath() (string, error) {
	overrides, configPath, ok, err := loadLauncherOverridesForCurrentContext()
	if err != nil {
		return "", err
	}
	if ok {
		overridePath, err := resolveLauncherOverrideValue(configPath, overrides.Paths.WASMExecJS)
		if err != nil {
			return "", fmt.Errorf("resolve wasmExecJS override: %w", err)
		}
		if strings.TrimSpace(overridePath) != "" {
			if !fileExists(overridePath) {
				return "", fmt.Errorf("configured wasmExecJS path does not exist: %s", overridePath)
			}
			return overridePath, nil
		}
	}
	goRoot := resolveWasmExecGoRoot()
	if goRoot == "" {
		return "", errors.New("GOROOT is not available")
	}
	candidates := []string{
		filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("wasm_exec.js not found under GOROOT %s", goRoot)
}

func renderScaffoldGoMod(selection startSelection, repoModulePath string, repoRoot string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("module %s\n\ngo 1.25.0\n", selection.ModulePath))
	if selection.ProjectMode == scaffoldProjectModeContributorLinked && strings.TrimSpace(repoModulePath) != "" && strings.TrimSpace(repoRoot) != "" {
		builder.WriteString("\n")
		builder.WriteString(fmt.Sprintf("replace %s => %s\n", repoModulePath, filepath.ToSlash(filepath.Clean(repoRoot))))
	}
	return builder.String()
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

func startSelectionFeatures(selection startSelection) []string {
	combined := append([]string{}, selection.Preset.Features...)
	combined = append(combined, selection.EnterpriseFeatures...)
	return normalizeScaffoldFeatureKeys(combined)
}

func normalizeScaffoldFeatureKeys(features []string) []string {
	seen := make(map[string]struct{}, len(features))
	normalized := make([]string, 0, len(features))
	for _, raw := range features {
		feature := strings.ToLower(strings.TrimSpace(raw))
		if feature == "" {
			continue
		}
		if _, exists := seen[feature]; exists {
			continue
		}
		seen[feature] = struct{}{}
		normalized = append(normalized, feature)
	}
	return normalized
}

func scaffoldFeatureDescriptorFor(key string) scaffoldFeatureDescriptor {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, descriptor := range scaffoldFeatureCatalog {
		if descriptor.Key == key {
			return descriptor
		}
	}
	label := strings.TrimSpace(strings.ReplaceAll(key, "-", " "))
	if label == "" {
		label = "Unknown Capability"
	}
	return scaffoldFeatureDescriptor{
		Key:         key,
		Label:       strings.ToUpper(label[:1]) + label[1:],
		Description: "Custom starter capability carried in preset metadata.",
	}
}

func scaffoldFeatureSet(features []string) map[string]struct{} {
	set := make(map[string]struct{}, len(features))
	for _, feature := range normalizeScaffoldFeatureKeys(features) {
		set[feature] = struct{}{}
	}
	return set
}

func scaffoldHasFeature(features []string, key string) bool {
	_, exists := scaffoldFeatureSet(features)[strings.ToLower(strings.TrimSpace(key))]
	return exists
}

func renderScaffoldFeatureCards(features []string) string {
	normalized := normalizeScaffoldFeatureKeys(features)
	if len(normalized) == 0 {
		return `				html.Div(html.Props{Class: "feature-card"},
					html.Span(html.Props{Class: "feature-tag"}, html.Text("Core Starter")),
					html.P(html.Props{Class: "feature-copy"}, html.Text("No optional capabilities were selected for this starter scaffold.")),
				),`
	}

	lines := make([]string, 0, len(normalized))
	for _, feature := range normalized {
		descriptor := scaffoldFeatureDescriptorFor(feature)
		lines = append(lines, fmt.Sprintf(`				html.Div(html.Props{Class: "feature-card"},
					html.Span(html.Props{Class: "feature-tag"}, html.Text(%q)),
					html.P(html.Props{Class: "feature-copy"}, html.Text(%q)),
				),`, descriptor.Label, descriptor.Description))
	}
	return strings.Join(lines, "\n")
}

func renderScaffoldCapabilityState(features []string) string {
	selected := scaffoldFeatureSet(features)
	blocks := []string{}

	if _, ok := selected["router"]; ok {
		blocks = append(blocks, `	routeSegment := ui.UseState("home")
	showHomeRoute := ui.UseEvent(func() {
		routeSegment.Set("home")
	})
	showDashboardRoute := ui.UseEvent(func() {
		routeSegment.Set("dashboard")
	})
	showSettingsRoute := ui.UseEvent(func() {
		routeSegment.Set("settings")
	})
	currentRoute := routeSegment.Get()`)
	}

	if _, ok := selected["forms"]; ok {
		blocks = append(blocks, `	formSubmitted := ui.UseState(false)
	submitStarterForm := ui.UseEvent(func() {
		formSubmitted.Set(true)
	})
	resetStarterForm := ui.UseEvent(func() {
		formSubmitted.Set(false)
	})
	formStatus := "Draft not submitted yet."
	if formSubmitted.Get() {
		formStatus = "Last submit marked complete."
	}`)
	}

	if _, ok := selected["fetch"]; ok {
		blocks = append(blocks, `	dataStatus := ui.UseState("idle")
	refreshStarterData := ui.UseEvent(func() {
		dataStatus.Update(func(previous string) string {
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
	currentDataStatus := dataStatus.Get()`)
	}

	if _, ok := selected["state"]; ok {
		blocks = append(blocks, `	teamCount := ui.UseState(3)
	addTeammate := ui.UseEvent(func() {
		teamCount.Update(func(previous int) int { return previous + 1 })
	})
	currentTeamCount := teamCount.Get()`)
	}

	if len(blocks) == 0 {
		return ""
	}
	return strings.Join(blocks, "\n\n")
}

func renderScaffoldCapabilityWidgets(features []string) string {
	selected := scaffoldFeatureSet(features)
	widgets := []string{}

	if _, ok := selected["router"]; ok {
		widgets = append(widgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Routing")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(fmt.Sprintf("Active route: %s", currentRoute))),
				html.Div(html.Props{Class: "capability-actions"},
					html.Button(html.Props{Class: "capability-button", OnClick: showHomeRoute}, html.Text("Home")),
					html.Button(html.Props{Class: "capability-button", OnClick: showDashboardRoute}, html.Text("Dashboard")),
					html.Button(html.Props{Class: "capability-button", OnClick: showSettingsRoute}, html.Text("Settings")),
				),
			),`)
	}

	if _, ok := selected["fetch"]; ok {
		widgets = append(widgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Async Data")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(fmt.Sprintf("Data status: %s", currentDataStatus))),
				html.Button(html.Props{Class: "capability-button", OnClick: refreshStarterData}, html.Text("Advance data sync state")),
			),`)
	}

	if _, ok := selected["state"]; ok {
		widgets = append(widgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Shared State")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(fmt.Sprintf("Team members tracked: %d", currentTeamCount))),
				html.Button(html.Props{Class: "capability-button", OnClick: addTeammate}, html.Text("Add teammate")),
			),`)
	}

	if _, ok := selected["forms"]; ok {
		widgets = append(widgets, `			html.Form(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Forms")),
				html.P(html.Props{Class: "capability-copy"}, html.Text(formStatus)),
				html.Label(html.Props{}, html.Text("Email")),
				html.Input(html.Props{Type: "email", Placeholder: "you@example.com", Class: "capability-input"}),
				html.Div(html.Props{Class: "capability-actions"},
					html.Button(html.Props{Class: "capability-button", Type: "button", OnClick: submitStarterForm}, html.Text("Submit")),
					html.Button(html.Props{Class: "capability-button", Type: "button", OnClick: resetStarterForm}, html.Text("Reset")),
				),
			),`)
	}

	if _, ok := selected["ssr"]; ok {
		widgets = append(widgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Server Rendering")),
				html.P(html.Props{Class: "capability-copy"}, html.Text("This preset is intended for request-time HTML rendering with launcher-managed release defaults.")),
			),`)
	}

	if _, ok := selected["hydration"]; ok {
		widgets = append(widgets, `			html.Div(html.Props{Class: "capability"},
				html.Span(html.Props{Class: "capability-title"}, html.Text("Hydration")),
				html.P(html.Props{Class: "capability-copy"}, html.Text("Hydration-ready app ownership is expected so server output and browser interactivity stay aligned.")),
			),`)
	}

	if len(widgets) == 0 {
		return ""
	}
	return strings.Join(widgets, "\n")
}

func renderScaffoldFeatureMatrix(selection startSelection) string {
	normalized := startSelectionFeatures(selection)
	selected := scaffoldFeatureSet(normalized)
	var builder strings.Builder
	builder.WriteString("# Starter Feature Matrix\n\n")
	builder.WriteString(fmt.Sprintf("Preset: `%s` (%s)\n\n", selection.Preset.Key, selection.Preset.Name))
	if len(selection.EnabledEnterpriseSections) > 0 {
		builder.WriteString("Enterprise plugin sections selected:\n\n")
		for _, section := range selection.EnabledEnterpriseSections {
			builder.WriteString(fmt.Sprintf("- %s\n", section))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("This file is generated from the selected scaffold capabilities.\n\n")
	for _, descriptor := range scaffoldFeatureCatalog {
		marker := "[ ]"
		if _, ok := selected[descriptor.Key]; ok {
			marker = "[x]"
		}
		builder.WriteString(fmt.Sprintf("- %s `%s`: %s\n", marker, descriptor.Key, descriptor.Description))
	}
	if len(normalized) > 0 {
		builder.WriteString("\nSelected capability order:\n\n")
		for _, feature := range normalized {
			descriptor := scaffoldFeatureDescriptorFor(feature)
			builder.WriteString(fmt.Sprintf("- `%s`: %s\n", descriptor.Key, descriptor.Label))
		}
	}
	builder.WriteString("\nGenerated starter output is intentionally disposable; treat this scaffold as a starting point you can edit or replace freely.\n")
	return builder.String()
}

func renderScaffoldBrowserTestREADME(selection startSelection) string {
	return fmt.Sprintf("# Browser smoke tests for %s\n\nUse this folder for Playwright-Go smoke tests that validate starter boot and basic user interactions.\n\nRun from the generated project root:\n\n```powershell\ngo test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v\n```\n", selection.ProjectName)
}

func renderScaffoldBrowserSmokeTest(selection startSelection) string {
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
`, selection.ProjectName)
}

func renderScaffoldGoStringList(values []string) string {
	if len(values) == 0 {
		return "nil"
	}
	lines := make([]string, 0, len(values))
	for _, value := range values {
		lines = append(lines, fmt.Sprintf("\t\t%q,", value))
	}
	return "[]string{\n" + strings.Join(lines, "\n") + "\n\t}"
}

func renderScaffoldFeatureBaselineTest(selection startSelection) string {
	features := startSelectionFeatures(selection)
	mainExpectations := []string{}
	extraPaths := []string{}
	if scaffoldHasFeature(features, "router") {
		mainExpectations = append(mainExpectations, `html.Text("Routing")`, `Active route: %s`)
	}
	if scaffoldHasFeature(features, "forms") {
		mainExpectations = append(mainExpectations, `html.Text("Forms")`, `Last submit marked complete.`)
	}
	if scaffoldHasFeature(features, "fetch") {
		mainExpectations = append(mainExpectations, `html.Text("Async Data")`, `Data status: %s`)
	}
	if scaffoldHasFeature(features, "browser-tests") {
		extraPaths = append(extraPaths, "test/playwrightgo/smoke_test.go")
	}

	extraAssertions := ""
	if len(mainExpectations) > 0 || len(extraPaths) > 0 {
		extraAssertions = fmt.Sprintf(`
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
`, renderScaffoldGoStringList(mainExpectations), renderScaffoldGoStringList(extraPaths))
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
`, renderScaffoldGoStringList(features), renderScaffoldGoStringList(features), extraAssertions)
}

func renderScaffoldGitHubActionsWorkflow(selection startSelection) string {
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
        run: go build -o main.wasm .
        env:
          GOOS: js
          GOARCH: wasm
`, selection.ProjectName)
}

func renderScaffoldExtraFiles(selection startSelection) map[string][]byte {
	files := map[string][]byte{
		"FEATURE_MATRIX.md":        []byte(renderScaffoldFeatureMatrix(selection)),
		"starter_test.go":          []byte(renderScaffoldFeatureBaselineTest(selection)),
		".github/workflows/ci.yml": []byte(renderScaffoldGitHubActionsWorkflow(selection)),
	}
	if scaffoldHasFeature(startSelectionFeatures(selection), "browser-tests") {
		files["test/playwrightgo/README.md"] = []byte(renderScaffoldBrowserTestREADME(selection))
		files["test/playwrightgo/smoke_test.go"] = []byte(renderScaffoldBrowserSmokeTest(selection))
	}
	return files
}

func renderScaffoldMain(selection startSelection, repoModulePath string) string {
	normalizedFeatures := startSelectionFeatures(selection)
	featureList := strings.Join(normalizedFeatures, ", ")
	if featureList == "" {
		featureList = "core-only"
	}
	featureCards := renderScaffoldFeatureCards(normalizedFeatures)
	capabilityState := renderScaffoldCapabilityState(normalizedFeatures)
	capabilityWidgets := renderScaffoldCapabilityWidgets(normalizedFeatures)
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
	count := ui.UseState(0)
	currentCount := count.Get()

	increment := ui.UseEvent(func() {
		count.Update(func(previous int) int { return previous + 1 })
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
			html.Div(html.Props{Class: "counter"}, html.Text(fmt.Sprintf("Count: %%d", currentCount))),
			html.Button(html.Props{OnClick: increment, Class: "button"}, html.Text("Increment")),
		),
	)
}

func main() {
	utils.DisableAllDebug()
	ui.Render(ui.CreateElement(App), "#app")
	select {}
}
`, repoModulePath+"/html", repoModulePath+"/ui", repoModulePath+"/utils", capabilityState, selection.ProjectName, selection.Description, selection.Author, selection.Version, selection.Preset.Name, selection.ModulePath, selection.Preset.Description, "Features: "+featureList, featureCards, capabilityWidgets)
}

func renderScaffoldHTML(selection startSelection) string {
	template := `<!DOCTYPE html>
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
		WebAssembly.instantiateStreaming(fetch('./main.wasm'), go.importObject)
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
	return strings.ReplaceAll(template, "__GWC_PROJECT_TITLE__", selection.ProjectName)
}

func renderScaffoldMetadata(selection startSelection) string {
	payload := defaultScaffoldMetadata(selection)
	encoded, err := scaffoldMarshalIndent(payload, "", "  ")
	if err != nil {
		return "{}\n"
	}
	return string(encoded) + "\n"
}

func renderScaffoldREADME(selection startSelection) string {
	normalizedFeatures := startSelectionFeatures(selection)
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# %s\n\n", selection.ProjectName))
	builder.WriteString(fmt.Sprintf("%s\n\n", selection.Description))
	builder.WriteString(fmt.Sprintf("- Author: %s\n", selection.Author))
	builder.WriteString(fmt.Sprintf("- Version: %s\n", selection.Version))
	builder.WriteString(fmt.Sprintf("- Preset: %s\n\n", selection.Preset.Key))

	builder.WriteString("## Selected Capabilities\n\n")
	if len(normalizedFeatures) == 0 {
		builder.WriteString("- none\n")
	} else {
		for _, feature := range normalizedFeatures {
			builder.WriteString(fmt.Sprintf("- `%s`\n", feature))
		}
	}
	builder.WriteString("\n")
	if len(selection.EnabledEnterpriseSections) > 0 {
		builder.WriteString("## Enterprise Extensions\n\n")
		builder.WriteString("These optional sections came from enterprise launcher plugins and were selected during `gwc start`.\n\n")
		for _, section := range selection.EnabledEnterpriseSections {
			builder.WriteString(fmt.Sprintf("- %s\n", section))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("The full generated capability contract lives in `FEATURE_MATRIX.md`.\n\n")

	builder.WriteString("## Starter Output Rules\n\n")
	builder.WriteString("- keep generated files small, readable, and conventionally organized\n")
	builder.WriteString("- treat this scaffold as disposable starter code; edit or replace it when the app shape changes\n")
	builder.WriteString("- keep app code on public framework packages instead of importing repo-internal build tooling\n")
	builder.WriteString("- avoid coupling to monorepo-only paths so this project can live independently\n\n")

	builder.WriteString("## Run\n\n")
	builder.WriteString("From the GoWebComponents repo root:\n\n")
	builder.WriteString("```powershell\n")
	builder.WriteString(fmt.Sprintf("go run ./tools/gwc dev -app %q -root %q -html %q -wasm %q\n", filepath.Join(selection.TargetDir, "main.go"), selection.TargetDir, filepath.Join(selection.TargetDir, "index.html"), scaffoldWASMOutputPath()))
	builder.WriteString("```\n")

	builder.WriteString("\n## Verify\n\n")
	builder.WriteString("From the generated project directory:\n\n")
	builder.WriteString("```powershell\n")
	builder.WriteString("go test ./...\n")
	builder.WriteString("```\n")
	if scaffoldHasFeature(normalizedFeatures, "browser-tests") {
		builder.WriteString("\nFor browser tests, start from `test/playwrightgo/smoke_test.go` and run:\n\n")
		builder.WriteString("```powershell\n")
		builder.WriteString("go test -tags playwrightgo ./test/playwrightgo -run TestMainSuite -v\n")
		builder.WriteString("```\n")
	}
	return builder.String()
}
