package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/andybalholm/brotli"
	gwchtml "github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/tools/runnerconfig"
	"github.com/monstercameron/GoWebComponents/ui"
)

const (
	defaultHost = "127.0.0.1"
	defaultPort = "8090"
)

type launcher struct {
	repoRoot        string
	examplesDir     string
	staticDir       string
	examplesWasmDir string
}

func (l launcher) resolvedExamplesWasmDir() string {
	if strings.TrimSpace(l.examplesWasmDir) != "" {
		return l.examplesWasmDir
	}
	if strings.TrimSpace(l.repoRoot) != "" {
		resolved, err := runnerconfig.ResolveWorkspaceBuildPath(l.repoRoot, runnerconfig.FS{}, "examples")
		if err == nil && strings.TrimSpace(resolved) != "" {
			if info, statErr := os.Stat(resolved); statErr == nil && info.IsDir() {
				return resolved
			}
		}
	}
	if strings.TrimSpace(l.staticDir) != "" {
		return filepath.Join(l.staticDir, "bin")
	}
	return ""
}

type exampleLink struct {
	Name string
	Href string
}

type exampleCatalogEntry struct {
	Name        string   `json:"name"`
	Href        string   `json:"href"`
	HTMLFile    string   `json:"htmlFile"`
	Title       string   `json:"title,omitempty"`
	UsesWasm    bool     `json:"usesWasm"`
	WasmBinary  string   `json:"wasmBinary,omitempty"`
	MultiClient bool     `json:"multiClient"`
	Tags        []string `json:"tags,omitempty"`
}

type generatedExamplePage struct {
	RoutePath     string
	DirName       string
	HTMLFile      string
	Title         string
	WasmBinary    string
	ManifestHref  string
	Description   string
	GeneratedFrom string
}

type examplesCatalogPayload struct {
	GeneratedAt         string                `json:"generatedAt"`
	TotalExamples       int                   `json:"totalExamples"`
	WasmExamples        int                   `json:"wasmExamples"`
	MultiClientExamples int                   `json:"multiClientExamples"`
	Examples            []exampleCatalogEntry `json:"examples"`
}

type devConfig struct {
	appPath  string
	rootPath string
	htmlPath string
	wasmPath string
	host     string
	port     string
	hot      bool
}

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

type scaffoldMetadata struct {
	ProjectName string                  `json:"projectName,omitempty"`
	ModulePath  string                  `json:"modulePath,omitempty"`
	Author      string                  `json:"author,omitempty"`
	Version     string                  `json:"version,omitempty"`
	Description string                  `json:"description,omitempty"`
	TargetDir   string                  `json:"targetDir,omitempty"`
	Preset      scaffoldPresetMetadata  `json:"preset,omitempty"`
	Tooling     scaffoldToolingMetadata `json:"tooling,omitempty"`
}

type buildConfig struct {
	appPath    string
	rootPath   string
	outputPath string
	profile    string
	json       bool
}

type buildProfile struct {
	Name     string `json:"name"`
	Trimpath bool   `json:"trimpath"`
	Ldflags  string `json:"ldflags,omitempty"`
	BuildVCS string `json:"buildvcs,omitempty"`
}

type buildSummary struct {
	OK          bool         `json:"ok"`
	Profile     buildProfile `json:"profile"`
	AppPath     string       `json:"appPath"`
	ProjectRoot string       `json:"projectRoot"`
	PackageDir  string       `json:"packageDir"`
	OutputPath  string       `json:"outputPath"`
	Bytes       int64        `json:"bytes"`
	SHA256      string       `json:"sha256"`
}

type releaseConfig struct {
	appPath         string
	rootPath        string
	outDir          string
	binaryName      string
	manifestName    string
	budgetsPath     string
	profile         string
	compression     string
	compressionSet  bool
	skipCompression bool
	skipCompressSet bool
	json            bool
}

type releaseArtifactRecord struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type releaseSummary struct {
	OK           bool                             `json:"ok"`
	Profile      buildProfile                     `json:"profile"`
	AppPath      string                           `json:"appPath"`
	ProjectRoot  string                           `json:"projectRoot"`
	PackageDir   string                           `json:"packageDir"`
	OutDir       string                           `json:"outDir"`
	ManifestPath string                           `json:"manifestPath"`
	Artifacts    map[string]releaseArtifactRecord `json:"artifacts"`
	Flags        map[string]interface{}           `json:"flags"`
}

type verifyTestSummary struct {
	Ran            bool   `json:"ran"`
	Skipped        bool   `json:"skipped"`
	Command        string `json:"command,omitempty"`
	PackagePattern string `json:"packagePattern,omitempty"`
	Output         string `json:"output,omitempty"`
}

type verifySummary struct {
	OK          bool              `json:"ok"`
	AppPath     string            `json:"appPath"`
	ProjectRoot string            `json:"projectRoot"`
	Tests       verifyTestSummary `json:"tests"`
	Build       buildSummary      `json:"build"`
}

type testConfig struct {
	appPath  string
	rootPath string
	lanes    []string
	json     bool
}

type testLaneSummary struct {
	Name           string   `json:"name"`
	OK             bool     `json:"ok"`
	Skipped        bool     `json:"skipped,omitempty"`
	Command        string   `json:"command,omitempty"`
	PackagePattern string   `json:"packagePattern,omitempty"`
	Packages       []string `json:"packages,omitempty"`
	Workspace      string   `json:"workspace,omitempty"`
	Output         string   `json:"output,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	OutDir         string   `json:"outDir,omitempty"`
	ManifestPath   string   `json:"manifestPath,omitempty"`
}

type testSummary struct {
	OK            bool              `json:"ok"`
	AppPath       string            `json:"appPath,omitempty"`
	ProjectRoot   string            `json:"projectRoot"`
	SelectedLanes []string          `json:"selectedLanes"`
	Lanes         []testLaneSummary `json:"lanes"`
}

type doctorConfig struct {
	host string
	port string
	json bool
}

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Hint    string `json:"hint,omitempty"`
}

type doctorReport struct {
	OK      bool          `json:"ok"`
	Checked string        `json:"checked"`
	CWD     string        `json:"cwd"`
	Checks  []doctorCheck `json:"checks"`
}

var doctorLookPath = exec.LookPath

var doctorCommandOutput = func(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

var doctorGetwd = os.Getwd

var doctorListen = net.Listen

var examplesListen = net.Listen

var examplesServe = func(server *http.Server, listener net.Listener) error {
	return server.Serve(listener)
}

var releaseExecuteBuild = executeBuild

var releaseArtifactRecordForPathFunc = releaseArtifactRecordForPath

var releaseWriteGzipSidecar = writeGzipSidecar

var releaseWriteBrotliSidecar = writeBrotliSidecar

var releaseMarshalIndent = json.MarshalIndent

var doctorResolveWasmExec = resolveWasmExecPath

var buildGetwd = os.Getwd

var devGetwd = os.Getwd

var scaffoldRel = filepath.Rel

var scaffoldResolveWasmExecPath = resolveWasmExecPath

var scaffoldWriteFile = os.WriteFile

var scaffoldReadFile = os.ReadFile

var scaffoldMarshalIndent = json.MarshalIndent

var scaffoldFormatMain = func(mainPath string) error {
	return exec.Command("gofmt", "-w", mainPath).Run()
}

var resolveWasmExecGoRoot = runtime.GOROOT

var resolveRepoRootCaller = runtime.Caller

var renderExamplesUIBootstrapScript = ui.RenderBootstrapScript

var renderExamplesBootstrapScriptFunc = renderExamplesBootstrapScript

var examplesCatalogMarshalIndent = json.MarshalIndent

var renderExamplesToString = ui.RenderToString

var startTerminalValidator = validateStartTerminal

var startSelectionRunner = runStartTUI

var startPostRunner = runStartPostTUI

var startGenerateScaffold = func(l launcher, selection startSelection) (scaffoldResult, error) {
	return l.generateStartScaffold(selection)
}

var startRunDev = func(l launcher, args []string) error {
	return l.runDev(args)
}

var runTestCommand = func(l launcher, args []string) error {
	return l.runTest(args)
}

var runExamplesCommand = func(l launcher, args []string) error {
	return l.runExamples(args)
}

var runBuildCommand = func(l launcher, args []string) error {
	return l.runBuild(args)
}

var runReleaseCommand = func(l launcher, args []string) error {
	return l.runRelease(args)
}

var runDevCommand = func(l launcher, args []string) error {
	return l.runDev(args)
}

var runDoctorCommand = func(l launcher, args []string) error {
	return l.runDoctor(args)
}

var runVerifyCommand = func(l launcher, args []string) error {
	return l.runVerify(args)
}

var runImportCommand = func(l launcher, args []string) error {
	return l.runImport(args)
}

var runStartCommand = func(l launcher, args []string) error {
	return l.runStart(args)
}

var verifyRunGoTests = func(rootPath string) (string, error) {
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = rootPath
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		if trimmed == "" {
			return "", fmt.Errorf("go test failed: %w", err)
		}
		return trimmed, fmt.Errorf("go test failed: %s", trimmed)
	}
	return trimmed, nil
}

var launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))
	if err != nil {
		if trimmed == "" {
			return "", fmt.Errorf("%s %s failed: %w", command, strings.Join(args, " "), err)
		}
		return trimmed, fmt.Errorf("%s %s failed: %s", command, strings.Join(args, " "), trimmed)
	}
	return trimmed, nil
}

var testResolveWasmExec = resolveWasmTestExec

var testExecuteRelease = executeRelease

var testGetwd = os.Getwd

var testAllLanes = []string{"unit", "wasm", "hydration", "browser", "release"}

var errStopWalk = errors.New("gwc-stop-walk")

var mainResolveRepoRoot = resolveRepoRoot

var mainRunLauncher = func(l launcher, args []string) error {
	return l.run(args)
}

var mainExit = os.Exit

var mainArgs = func() []string {
	return os.Args
}

var mainPrintError = func(err error) {
	fmt.Fprintf(os.Stderr, "gwc: %v\n", err)
}

func main() {
	repoRoot, err := mainResolveRepoRoot()
	if err != nil {
		mainPrintError(err)
		mainExit(1)
	}
	examplesWasmDir, err := runnerconfig.ResolveWorkspaceBuildPath(repoRoot, runnerconfig.FS{}, "examples")
	if err != nil {
		mainPrintError(fmt.Errorf("resolve examples build root: %w", err))
		mainExit(1)
	}

	l := launcher{
		repoRoot:        repoRoot,
		examplesDir:     filepath.Join(repoRoot, "examples"),
		staticDir:       filepath.Join(repoRoot, "examples", "static"),
		examplesWasmDir: examplesWasmDir,
	}

	if err := mainRunLauncher(l, mainArgs()[1:]); err != nil {
		mainPrintError(err)
		mainExit(1)
	}
}

func (l launcher) run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printUsage()
		return nil
	case "test":
		return runTestCommand(l, args[1:])
	case "examples":
		return runExamplesCommand(l, args[1:])
	case "build":
		return runBuildCommand(l, args[1:])
	case "release":
		return runReleaseCommand(l, args[1:])
	case "dev":
		return runDevCommand(l, args[1:])
	case "doctor":
		return runDoctorCommand(l, args[1:])
	case "verify":
		return runVerifyCommand(l, args[1:])
	case "import":
		return runImportCommand(l, args[1:])
	case "start":
		return runStartCommand(l, args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func (l launcher) runTest(args []string) error {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	app := fs.String("app", "", "Path to the app main.go file or app directory")
	mainPath := fs.String("main", "", "Legacy alias for -app")
	root := fs.String("root", "", "Project root used for test lane resolution")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	var laneFlags stringListFlag
	fs.Var(&laneFlags, "lane", "Test lane to run; repeat or comma-separate: unit, wasm, hydration, browser, release, all")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveTestConfig(testConfig{
		appPath:  firstNonEmpty(*app, *mainPath),
		rootPath: *root,
		lanes:    laneFlags.Values(),
		json:     *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := l.executeTest(config)
	if err != nil {
		return err
	}
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printTestSummary(summary)
	return nil
}

func resolveTestConfig(config testConfig) (testConfig, error) {
	resolved := config
	cwd, err := testGetwd()
	if err != nil {
		return testConfig{}, err
	}
	if strings.TrimSpace(resolved.rootPath) == "" {
		resolved.rootPath = cwd
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return testConfig{}, fmt.Errorf("resolve test root path: %w", err)
	}
	if strings.TrimSpace(resolved.appPath) != "" {
		resolved.appPath, err = normalizeExistingPath(cwd, resolved.appPath)
		if err != nil {
			return testConfig{}, fmt.Errorf("resolve app path: %w", err)
		}
	}
	resolved.lanes, err = normalizeTestLanes(resolved.lanes)
	if err != nil {
		return testConfig{}, err
	}
	return resolved, nil
}

func (l launcher) executeTest(config testConfig) (testSummary, error) {
	summary := testSummary{
		OK:            true,
		AppPath:       config.appPath,
		ProjectRoot:   config.rootPath,
		SelectedLanes: append([]string(nil), config.lanes...),
		Lanes:         make([]testLaneSummary, 0, len(config.lanes)),
	}
	for _, lane := range config.lanes {
		laneSummary, err := l.executeTestLane(config, lane)
		if err != nil {
			return testSummary{}, err
		}
		summary.Lanes = append(summary.Lanes, laneSummary)
		if !laneSummary.OK && !laneSummary.Skipped {
			summary.OK = false
		}
	}
	return summary, nil
}

func (l launcher) executeTestLane(config testConfig, lane string) (testLaneSummary, error) {
	switch lane {
	case "unit":
		return l.runUnitTestLane(config.rootPath)
	case "wasm":
		return l.runWasmTestLane(config.rootPath, false)
	case "hydration":
		return l.runWasmTestLane(config.rootPath, true)
	case "browser":
		return l.runBrowserTestLane(config.rootPath)
	case "release":
		return l.runReleaseTestLane(config)
	default:
		return testLaneSummary{}, fmt.Errorf("unknown test lane %q", lane)
	}
}

func (l launcher) runUnitTestLane(rootPath string) (testLaneSummary, error) {
	outputs := []string{}
	output, err := launcherRunCommand("go", []string{"test", "./..."}, rootPath, buildNativeGoEnv())
	if output != "" {
		outputs = append(outputs, output)
	}
	if err != nil {
		return testLaneSummary{}, err
	}
	summary := testLaneSummary{
		Name:           "unit",
		OK:             true,
		Command:        "go test ./...",
		PackagePattern: "./...",
		Workspace:      rootPath,
		Summary:        "Native Go tests passed.",
	}
	if rootPath == l.repoRoot {
		nestedRoot, err := resolveLauncherLivereloadWorkspace(l.repoRoot, rootPath)
		if err != nil {
			return testLaneSummary{}, err
		}
		if fileExists(filepath.Join(nestedRoot, "go.mod")) {
			nestedOutput, nestedErr := launcherRunCommand("go", []string{"test", "./..."}, nestedRoot, buildNativeGoEnv())
			if nestedOutput != "" {
				outputs = append(outputs, nestedOutput)
			}
			if nestedErr != nil {
				return testLaneSummary{}, nestedErr
			}
			summary.Summary = "Native Go tests passed, including the nested livereload workspace."
		}
	}
	if len(outputs) > 0 {
		summary.Output = strings.Join(outputs, "\n")
	}
	return summary, nil
}

func (l launcher) runWasmTestLane(rootPath string, hydrationOnly bool) (testLaneSummary, error) {
	packages, err := collectWasmTestPackages(rootPath, hydrationOnly)
	if err != nil {
		return testLaneSummary{}, err
	}
	laneName := "wasm"
	summaryText := "Discovered js/wasm Go test packages passed."
	if hydrationOnly {
		laneName = "hydration"
		summaryText = "Focused hydration js/wasm test packages passed."
	}
	if len(packages) == 0 {
		return testLaneSummary{
			Name:      laneName,
			OK:        true,
			Skipped:   true,
			Workspace: rootPath,
			Summary:   "No matching js/wasm test packages were found.",
		}, nil
	}
	wasmExec, err := testResolveWasmExec(l.repoRoot)
	if err != nil {
		return testLaneSummary{}, err
	}
	args := append([]string{"test", "-exec", wasmExec}, packages...)
	output, err := launcherRunCommand("go", args, rootPath, buildWasmGoEnv())
	if err != nil {
		return testLaneSummary{}, err
	}
	return testLaneSummary{
		Name:           laneName,
		OK:             true,
		Command:        "go " + strings.Join(args, " "),
		PackagePattern: strings.Join(packages, " "),
		Packages:       packages,
		Workspace:      rootPath,
		Output:         output,
		Summary:        summaryText,
	}, nil
}

func (l launcher) runBrowserTestLane(rootPath string) (testLaneSummary, error) {
	workspace, err := resolveBrowserWorkspace(l.repoRoot, rootPath)
	if err != nil {
		return testLaneSummary{}, err
	}
	if workspace == "" {
		return testLaneSummary{
			Name:      "browser",
			OK:        true,
			Skipped:   true,
			Workspace: rootPath,
			Summary:   "No Playwright workspace was found for the requested root.",
		}, nil
	}
	args := []string{"test", "--", "--reporter=list"}
	output, err := launcherRunCommand(npmCommandName(), args, workspace, buildBrowserTestEnv())
	if err != nil {
		return testLaneSummary{}, err
	}
	return testLaneSummary{
		Name:      "browser",
		OK:        true,
		Command:   npmCommandName() + " " + strings.Join(args, " "),
		Workspace: workspace,
		Output:    output,
		Summary:   "Browser Playwright workspace passed.",
	}, nil
}

func (l launcher) runReleaseTestLane(config testConfig) (testLaneSummary, error) {
	releaseOutDir, err := createLauncherTempDir(config.rootPath, "gwc-test-release-")
	if err != nil {
		return testLaneSummary{}, err
	}
	releaseConfig, err := resolveReleaseConfig(releaseConfig{
		appPath:  config.appPath,
		rootPath: config.rootPath,
		outDir:   releaseOutDir,
		profile:  "release",
	})
	if err != nil {
		return testLaneSummary{}, err
	}
	releaseSummary, err := testExecuteRelease(releaseConfig)
	if err != nil {
		return testLaneSummary{}, err
	}
	return testLaneSummary{
		Name:         "release",
		OK:           true,
		Summary:      "Release smoke build passed.",
		OutDir:       releaseSummary.OutDir,
		ManifestPath: releaseSummary.ManifestPath,
		Workspace:    releaseSummary.ProjectRoot,
	}, nil
}

type stringListFlag struct {
	values []string
}

func (f *stringListFlag) String() string {
	return strings.Join(f.values, ",")
}

func (f *stringListFlag) Set(value string) error {
	for _, part := range strings.Split(value, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			f.values = append(f.values, trimmed)
		}
	}
	return nil
}

func (f *stringListFlag) Values() []string {
	return append([]string(nil), f.values...)
}

func normalizeTestLanes(requested []string) ([]string, error) {
	if len(requested) == 0 {
		return []string{"unit", "wasm"}, nil
	}
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(requested))
	appendLane := func(lane string) {
		if _, ok := seen[lane]; ok {
			return
		}
		seen[lane] = struct{}{}
		normalized = append(normalized, lane)
	}
	for _, lane := range requested {
		switch strings.ToLower(strings.TrimSpace(lane)) {
		case "all":
			for _, candidate := range testAllLanes {
				appendLane(candidate)
			}
		case "unit", "native", "go-native":
			appendLane("unit")
		case "wasm", "go-wasm":
			appendLane("wasm")
		case "hydration", "hydrate":
			appendLane("hydration")
		case "browser", "playwright":
			appendLane("browser")
		case "release":
			appendLane("release")
		default:
			return nil, fmt.Errorf("unknown test lane %q", lane)
		}
	}
	return normalized, nil
}

func collectWasmTestPackages(rootPath string, hydrationOnly bool) ([]string, error) {
	packages := map[string]struct{}{}
	err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipTestWalkDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_wasm_test.go") {
			return nil
		}
		if hydrationOnly {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if !isHydrationTestContent(string(content)) {
				return nil
			}
		}
		relDir, err := filepath.Rel(rootPath, filepath.Dir(path))
		if err != nil {
			return err
		}
		packagePath := "."
		if relDir != "." {
			packagePath = "./" + filepath.ToSlash(relDir)
		}
		packages[packagePath] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("collect js/wasm test packages: %w", err)
	}
	ordered := make([]string, 0, len(packages))
	for pkg := range packages {
		ordered = append(ordered, pkg)
	}
	sort.Strings(ordered)
	return ordered, nil
}

func shouldSkipTestWalkDir(name string) bool {
	return name == ".git" ||
		name == "node_modules" ||
		name == "dist" ||
		name == "tmp" ||
		name == "test-results" ||
		name == "playwright-report" ||
		name == "coverage"
}

func isHydrationTestContent(content string) bool {
	return strings.Contains(content, "SmokeHydrate") ||
		strings.Contains(content, "Hydrate") ||
		strings.Contains(content, "Hydration")
}

func buildNativeGoEnv() []string {
	env := []string{}
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "GOOS=") || strings.HasPrefix(entry, "GOARCH=") {
			continue
		}
		env = append(env, entry)
	}
	return env
}

func buildWasmGoEnv() []string {
	env := buildNativeGoEnv()
	env = append(env, "GOOS=js", "GOARCH=wasm")
	return env
}

func buildBrowserTestEnv() []string {
	env := buildNativeGoEnv()
	workersSet := false
	for _, entry := range env {
		if strings.HasPrefix(entry, "PLAYWRIGHT_WORKERS=") {
			workersSet = true
			break
		}
	}
	if !workersSet {
		env = append(env, "PLAYWRIGHT_WORKERS=4")
	}
	return env
}

func resolveWasmTestExec(repoRoot string) (string, error) {
	overridePath, ok, err := resolveLauncherConfiguredPath(repoRoot, func(paths launcherOverridePaths) string {
		return paths.GoWASMExec
	}, "goWasmExec")
	if err != nil {
		return "", err
	}
	if ok {
		if !fileExists(overridePath) {
			return "", fmt.Errorf("configured goWasmExec path does not exist: %s", overridePath)
		}
		return overridePath, nil
	}
	if value := strings.TrimSpace(os.Getenv("GO_WASM_EXEC")); value != "" {
		return value, nil
	}
	if runtime.GOOS == "windows" {
		candidate := filepath.Join(repoRoot, "tools", "go_js_wasm_exec.bat")
		if fileExists(candidate) {
			return candidate, nil
		}
	}
	return "", errors.New("GO_WASM_EXEC is not set and the repo js/wasm executor helper could not be resolved")
}

func resolveBrowserWorkspace(repoRoot string, rootPath string) (string, error) {
	overridePath, ok, err := resolveLauncherConfiguredPath(rootPath, func(paths launcherOverridePaths) string {
		return paths.BrowserWorkspace
	}, "browserWorkspace")
	if err != nil {
		return "", err
	}
	if ok {
		if !fileExists(filepath.Join(overridePath, "package.json")) {
			return "", fmt.Errorf("configured browserWorkspace does not contain a package.json file: %s", overridePath)
		}
		return overridePath, nil
	}
	rootPackage := filepath.Join(rootPath, "package.json")
	if fileExists(rootPackage) && (fileExists(filepath.Join(rootPath, "playwright.config.js")) || fileExists(filepath.Join(rootPath, "playwright.config.ts"))) {
		return rootPath, nil
	}
	repoWorkspace := filepath.Join(repoRoot, "test")
	if fileExists(filepath.Join(repoWorkspace, "package.json")) {
		return repoWorkspace, nil
	}
	return "", nil
}

func detectBrowserWorkspace(repoRoot string, rootPath string) string {
	workspace, err := resolveBrowserWorkspace(repoRoot, rootPath)
	if err != nil {
		return ""
	}
	return workspace
}

func resolveLauncherLivereloadWorkspace(repoRoot string, rootPath string) (string, error) {
	overridePath, ok, err := resolveLauncherConfiguredPath(rootPath, func(paths launcherOverridePaths) string {
		return paths.LivereloadWorkspace
	}, "livereloadWorkspace")
	if err != nil {
		return "", err
	}
	if ok {
		info, statErr := os.Stat(overridePath)
		if statErr != nil || !info.IsDir() {
			return "", fmt.Errorf("configured livereloadWorkspace path does not exist: %s", overridePath)
		}
		return overridePath, nil
	}
	return filepath.Join(repoRoot, "tools", "livereload"), nil
}

func resolveLauncherLivereloadClientScript(repoRoot string, rootPath string) (string, bool, error) {
	overridePath, ok, err := resolveLauncherConfiguredPath(rootPath, func(paths launcherOverridePaths) string {
		return paths.LivereloadClientScript
	}, "livereloadClientScript")
	if err != nil {
		return "", false, err
	}
	if ok {
		if !fileExists(overridePath) {
			return "", false, fmt.Errorf("configured livereloadClientScript path does not exist: %s", overridePath)
		}
		return overridePath, true, nil
	}
	workspace, err := resolveLauncherLivereloadWorkspace(repoRoot, rootPath)
	if err != nil {
		return "", false, err
	}
	candidate := filepath.Join(workspace, "scripts", "livereload-client.js")
	if fileExists(candidate) {
		return candidate, true, nil
	}
	return "", false, nil
}

func npmCommandName() string {
	if runtime.GOOS == "windows" {
		return "npm.cmd"
	}
	return "npm"
}

func (l launcher) runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	app := fs.String("app", "", "Path to the app main.go file or app directory")
	mainPath := fs.String("main", "", "Legacy alias for -app")
	root := fs.String("root", "", "Project root used for test and build resolution")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	skipTests := fs.Bool("skip-tests", false, "Skip running go test even when *_test.go files are present")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	buildConfig, err := resolveBuildConfig(buildConfig{
		appPath:  firstNonEmpty(*app, *mainPath),
		rootPath: *root,
		profile:  "ci",
	})
	if err != nil {
		return err
	}

	summary := verifySummary{
		AppPath:     buildConfig.appPath,
		ProjectRoot: buildConfig.rootPath,
		Tests: verifyTestSummary{
			Command:        "go test",
			PackagePattern: "./...",
		},
	}

	if *skipTests {
		summary.Tests.Skipped = true
	} else {
		hasTests, err := projectHasGoTests(buildConfig.rootPath)
		if err != nil {
			return err
		}
		if hasTests {
			output, err := verifyRunGoTests(buildConfig.rootPath)
			if output != "" {
				summary.Tests.Output = output
			}
			if err != nil {
				return err
			}
			summary.Tests.Ran = true
		} else {
			summary.Tests.Skipped = true
		}
	}

	buildSummary, err := executeBuild(buildConfig)
	if err != nil {
		return err
	}
	summary.Build = buildSummary
	summary.OK = true

	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printVerifySummary(summary)
	return nil
}

func (l launcher) runRelease(args []string) error {
	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	app := fs.String("app", "", "Path to the app main.go file or app directory")
	mainPath := fs.String("main", "", "Legacy alias for -app")
	root := fs.String("root", "", "Project root used for output resolution")
	outDir := fs.String("out-dir", "", "Release output directory")
	binaryName := fs.String("binary-name", "", "Primary wasm artifact filename")
	manifestName := fs.String("manifest-name", "wasm-release-manifest.json", "Release manifest filename")
	budgetsPath := fs.String("budgets", "", "Optional path to a JSON budgets file")
	profile := fs.String("profile", "release", "Release build profile")
	compression := fs.String("compression", "", "Compression sidecars: none, gzip, brotli, or gzip+brotli")
	skipCompression := fs.Bool("skip-compression", false, "Skip gzip sidecar generation")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	skipCompressionSet := false
	compressionSet := false
	fs.Visit(func(flag *flag.Flag) {
		if flag.Name == "skip-compression" {
			skipCompressionSet = true
		}
		if flag.Name == "compression" {
			compressionSet = true
		}
	})
	if skipCompressionSet && compressionSet {
		return errors.New("use either -compression or -skip-compression, not both")
	}
	compressionPolicy := *compression
	if skipCompressionSet {
		compressionPolicy = "none"
	}

	config, err := resolveReleaseConfig(releaseConfig{
		appPath:         firstNonEmpty(*app, *mainPath),
		rootPath:        *root,
		outDir:          *outDir,
		binaryName:      *binaryName,
		manifestName:    *manifestName,
		budgetsPath:     *budgetsPath,
		profile:         *profile,
		compression:     compressionPolicy,
		compressionSet:  compressionSet || skipCompressionSet,
		skipCompression: *skipCompression,
		skipCompressSet: skipCompressionSet,
		json:            *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := executeRelease(config)
	if err != nil {
		return err
	}
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printReleaseSummary(summary)
	return nil
}

func (l launcher) runBuild(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	app := fs.String("app", "", "Path to the app main.go file or app directory")
	mainPath := fs.String("main", "", "Legacy alias for -app")
	root := fs.String("root", "", "Project root used for output resolution")
	out := fs.String("out", "", "WASM output path")
	output := fs.String("output", "", "Legacy alias for -out")
	profile := fs.String("profile", "", "Build profile: development, ci, benchmark, or release")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveBuildConfig(buildConfig{
		appPath:    firstNonEmpty(*app, *mainPath),
		rootPath:   *root,
		outputPath: firstNonEmpty(*out, *output),
		profile:    *profile,
		json:       *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := executeBuild(config)
	if err != nil {
		return err
	}
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printBuildSummary(summary)
	return nil
}

func (l launcher) runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	host := fs.String("host", defaultHost, "Host to probe for port availability")
	port := fs.String("port", "8080", "Port to probe for local development availability")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	report := l.buildDoctorReport(doctorConfig{host: *host, port: *port, json: *jsonOutput})
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
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

func (l launcher) runExamples(args []string) error {
	fs := flag.NewFlagSet("examples", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	host := fs.String("host", defaultHost, "Host to bind")
	port := fs.String("port", defaultPort, "Port to bind")
	exportStaticCatalog := fs.String("export-static-catalog", "", "Write a static examples catalog JSON file for static hosting and exit")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	if _, err := os.Stat(l.examplesDir); err != nil {
		return fmt.Errorf("examples directory not found: %w", err)
	}

	if strings.TrimSpace(*exportStaticCatalog) != "" {
		if err := l.writeStaticExamplesCatalogFile(*exportStaticCatalog); err != nil {
			return err
		}
		fmt.Printf("Wrote static examples catalog to %s\n", *exportStaticCatalog)
		return nil
	}

	addr := joinHostPort(*host, *port)
	listener, err := examplesListen("tcp", addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	server := &http.Server{
		Addr:    addr,
		Handler: l.newExamplesHandler(*host, *port),
	}

	fmt.Printf("GWC examples server listening on http://%s\n", addr)
	fmt.Printf("Examples: http://%s\n", addr)
	fmt.Printf("Counter:  http://%s/examples/01-counter/\n", addr)

	if err := examplesServe(server, listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (l launcher) newExamplesHandler(host string, port string) http.Handler {
	examplesServer := http.StripPrefix("/examples/", http.FileServer(http.Dir(l.examplesDir)))
	staticServer := http.StripPrefix("/static/", http.FileServer(http.Dir(l.staticDir)))
	wasmServer := http.StripPrefix("/static/bin/", http.FileServer(http.Dir(l.resolvedExamplesWasmDir())))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"ok":      true,
			"service": "gowebcomponents-gwc-examples",
			"time":    time.Now().UTC().Format(time.RFC3339),
			"root":    l.repoRoot,
			"host":    host,
			"port":    port,
		})
	})
	mux.HandleFunc("/examples/list", func(w http.ResponseWriter, r *http.Request) {
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		links, err := l.buildExamplesListing()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "examples_listing_failed",
				"hint":  err.Error(),
			})
			return
		}
		links = filterExampleLinks(links, query)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderExamplesListingHTML(links, query)))
	})
	mux.HandleFunc("/examples/catalog.json", func(w http.ResponseWriter, r *http.Request) {
		catalog, err := l.buildExamplesCatalog()
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "examples_catalog_failed",
				"hint":  err.Error(),
			})
			return
		}
		writeJSON(w, http.StatusOK, catalog)
	})
	mux.HandleFunc("/examples", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/examples" {
			examplesServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
	})
	mux.HandleFunc("/examples/static/index.html", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/examples/static/index.html" {
			examplesServer.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
	})
	mux.HandleFunc("/examples/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/examples/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
			return
		}
		trimmed := strings.TrimPrefix(filepath.ToSlash(r.URL.Path), "/examples/")
		trimmed = strings.TrimSpace(trimmed)
		if trimmed == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(renderExamplesAppShellHTML("/examples/", "/examples/")))
			return
		}
		if strings.HasSuffix(strings.ToLower(r.URL.Path), ".html") {
			dirName := strings.TrimSpace(strings.Split(trimmed, "/")[0])
			if dirName != "" {
				http.Redirect(w, r, "/examples/"+dirName+"/", http.StatusFound)
				return
			}
			examplePage, ok, err := l.resolveGeneratedExamplePage(r.URL.Path)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "examples_html_route_failed",
					"hint":  err.Error(),
				})
				return
			}
			if ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(renderGeneratedExampleHTML(examplePage)))
				return
			}
		}
		if !strings.Contains(trimmed, "/") {
			http.Redirect(w, r, r.URL.Path+"/", http.StatusFound)
			return
		}
		if strings.Count(strings.Trim(trimmed, "/"), "/") == 0 {
			examplePage, ok, err := l.resolveGeneratedExamplePage(r.URL.Path)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{
					"error": "examples_route_failed",
					"hint":  err.Error(),
				})
				return
			}
			if ok {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(renderGeneratedExampleHTML(examplePage)))
				return
			}
		}
		examplesServer.ServeHTTP(w, r)
	})
	mux.Handle("/static/bin/", wasmServer)
	mux.Handle("/static/", staticServer)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(renderExamplesAppShellHTML("/", "/")))
			return
		}
		examplesServer.ServeHTTP(w, r)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		applyDevHeaders(wrapped, r)
		mux.ServeHTTP(wrapped, r)
		fmt.Printf("%d %s %s %dms\n", wrapped.status, r.Method, r.URL.RequestURI(), time.Since(start).Milliseconds())
	})
}

func (l launcher) buildExamplesListing() ([]exampleLink, error) {
	catalog, err := l.buildExamplesCatalog()
	if err != nil {
		return nil, err
	}
	links := make([]exampleLink, 0, len(catalog.Examples))
	for _, entry := range catalog.Examples {
		links = append(links, exampleLink{Name: entry.Name, Href: entry.Href})
	}
	return links, nil
}

func (l launcher) buildExamplesCatalog() (examplesCatalogPayload, error) {
	return l.buildExamplesCatalogWithHref(func(dirPath string, dirName string, htmlFile string) string {
		return "/examples/" + dirName + "/"
	})
}

func (l launcher) buildStaticExamplesCatalog() (examplesCatalogPayload, error) {
	return l.buildExamplesCatalogWithHref(func(dirPath string, dirName string, htmlFile string) string {
		return "../" + dirName + "/" + htmlFile
	})
}

func (l launcher) buildExamplesCatalogWithHref(resolveHref func(dirPath string, dirName string, htmlFile string) string) (examplesCatalogPayload, error) {
	entries, err := os.ReadDir(l.examplesDir)
	if err != nil {
		return examplesCatalogPayload{}, err
	}

	pattern := regexp.MustCompile(`^\d{2}-`)
	catalogEntries := make([]exampleCatalogEntry, 0)
	for _, entry := range entries {
		if !entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		dirPath := filepath.Join(l.examplesDir, entry.Name())
		catalogEntry, ok, err := l.buildExampleCatalogEntry(dirPath, entry.Name(), resolveHref)
		if err != nil {
			return examplesCatalogPayload{}, err
		}
		if ok {
			catalogEntries = append(catalogEntries, catalogEntry)
		}
	}
	sort.Slice(catalogEntries, func(i, j int) bool { return catalogEntries[i].Name < catalogEntries[j].Name })

	wasmCount := 0
	multiClientCount := 0
	for _, entry := range catalogEntries {
		if entry.UsesWasm {
			wasmCount++
		}
		if entry.MultiClient {
			multiClientCount++
		}
	}

	return examplesCatalogPayload{
		GeneratedAt:         time.Now().UTC().Format(time.RFC3339),
		TotalExamples:       len(catalogEntries),
		WasmExamples:        wasmCount,
		MultiClientExamples: multiClientCount,
		Examples:            catalogEntries,
	}, nil
}

func (l launcher) writeStaticExamplesCatalogFile(targetPath string) error {
	catalog, err := l.buildStaticExamplesCatalog()
	if err != nil {
		return err
	}
	targetPath = strings.TrimSpace(targetPath)
	if targetPath == "" {
		return errors.New("static catalog output path is required")
	}
	if !filepath.IsAbs(targetPath) {
		resolved, err := filepath.Abs(targetPath)
		if err != nil {
			return fmt.Errorf("resolve static catalog path: %w", err)
		}
		targetPath = resolved
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create static catalog directory: %w", err)
	}
	encoded, err := examplesCatalogMarshalIndent(catalog, "", "  ")
	if err != nil {
		return fmt.Errorf("encode static catalog: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(targetPath, encoded, 0644); err != nil {
		return fmt.Errorf("write static catalog: %w", err)
	}
	return nil
}

func filterExampleLinks(links []exampleLink, query string) []exampleLink {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return links
	}
	terms := strings.Fields(query)
	filtered := make([]exampleLink, 0, len(links))
	for _, link := range links {
		haystack := strings.ToLower(link.Name + " " + link.Href)
		matchesAll := true
		for _, term := range terms {
			if !strings.Contains(haystack, term) {
				matchesAll = false
				break
			}
		}
		if matchesAll {
			filtered = append(filtered, link)
		}
	}
	return filtered
}

func (l launcher) runDev(args []string) error {
	fs := flag.NewFlagSet("dev", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	app := fs.String("app", "", "Path to the app main.go file or app directory")
	mainPath := fs.String("main", "", "Legacy alias for -app")
	root := fs.String("root", "", "Project root to watch and serve")
	html := fs.String("html", "", "HTML file to serve, relative to the project root")
	index := fs.String("index", "", "Legacy alias for -html")
	wasm := fs.String("wasm", "", "WASM output path, relative to the build directory")
	output := fs.String("output", "", "Legacy alias for -wasm")
	host := fs.String("host", "", "Host to bind")
	port := fs.String("port", "", "Port to bind")
	hot := fs.Bool("hot", true, "Always use hot reload on successful rebuilds")
	clientScript := fs.String("client-script", "", "Path to livereload-client.js")
	dryRun := fs.Bool("dry-run", false, "Resolve the dev plan and exit without starting the server")
	jsonOutput := fs.Bool("json", false, "Print the resolved dev plan as JSON")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := l.resolveDevConfig(devConfig{
		appPath:  firstNonEmpty(*app, *mainPath),
		rootPath: *root,
		htmlPath: firstNonEmpty(*html, *index),
		wasmPath: firstNonEmpty(*wasm, *output),
		host:     *host,
		port:     *port,
		hot:      *hot,
	})
	if err != nil {
		return err
	}

	forwarded := []string{"run", "."}
	forwarded = append(forwarded, "-app", config.appPath)
	if config.rootPath != "" {
		forwarded = append(forwarded, "-root", config.rootPath)
	}
	if config.htmlPath != "" {
		forwarded = append(forwarded, "-html", config.htmlPath)
	}
	if config.wasmPath != "" {
		forwarded = append(forwarded, "-wasm", config.wasmPath)
	}
	forwarded = append(forwarded, "-host", config.host, "-port", config.port, "-hot", fmt.Sprintf("%t", config.hot))
	resolvedClientScript := strings.TrimSpace(*clientScript)
	if resolvedClientScript == "" {
		if autoClientScript, ok, resolveErr := resolveLauncherLivereloadClientScript(l.repoRoot, config.rootPath); resolveErr != nil {
			return resolveErr
		} else if ok {
			resolvedClientScript = autoClientScript
		}
	}
	if resolvedClientScript != "" {
		forwarded = append(forwarded, "-client-script", resolvedClientScript)
	}

	if *jsonOutput {
		if err := printDevPlanJSON(config); err != nil {
			return err
		}
	} else {
		printDevPlan(config)
	}
	if *dryRun {
		return nil
	}
	livereloadWorkspace, err := resolveLauncherLivereloadWorkspace(l.repoRoot, config.rootPath)
	if err != nil {
		return err
	}
	cmd := exec.Command("go", forwarded...)
	cmd.Dir = livereloadWorkspace
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

func resolveBuildConfig(config buildConfig) (buildConfig, error) {
	resolved := config
	explicitOutputPath := strings.TrimSpace(config.outputPath) != ""
	cwd, err := buildGetwd()
	if err != nil {
		return buildConfig{}, err
	}

	metadata, metadataDir, hasMetadata, err := resolveScaffoldMetadataForConfig(cwd, resolved.rootPath, resolved.appPath)
	if err != nil {
		return buildConfig{}, err
	}
	if hasMetadata {
		if strings.TrimSpace(resolved.appPath) == "" && strings.TrimSpace(metadata.Tooling.AppPath) != "" {
			resolved.appPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.AppPath))
		}
		if strings.TrimSpace(resolved.rootPath) == "" {
			resolved.rootPath = metadataDir
		}
		if strings.TrimSpace(resolved.outputPath) == "" && strings.TrimSpace(metadata.Tooling.WASMPath) != "" {
			resolved.outputPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.WASMPath))
		}
		if strings.TrimSpace(resolved.profile) == "" && strings.TrimSpace(metadata.Tooling.DefaultBuildProfile) != "" {
			resolved.profile = strings.TrimSpace(metadata.Tooling.DefaultBuildProfile)
		}
	}

	if strings.TrimSpace(resolved.appPath) == "" {
		resolved.appPath, err = detectAppPath(cwd)
		if err != nil {
			return buildConfig{}, err
		}
	}
	resolved.appPath, err = normalizeExistingPath(cwd, resolved.appPath)
	if err != nil {
		return buildConfig{}, fmt.Errorf("resolve app path: %w", err)
	}

	appDir := resolved.appPath
	info, err := os.Stat(resolved.appPath)
	if err != nil {
		return buildConfig{}, fmt.Errorf("inspect app path: %w", err)
	}
	if !info.IsDir() {
		appDir = filepath.Dir(resolved.appPath)
	}

	if strings.TrimSpace(resolved.rootPath) == "" {
		resolved.rootPath = appDir
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return buildConfig{}, fmt.Errorf("resolve root path: %w", err)
	}

	if !explicitOutputPath {
		if artifactPath, ok, err := resolveLauncherArtifactPath(resolved.rootPath, scaffoldWASMOutputPath()); err != nil {
			return buildConfig{}, err
		} else if ok {
			resolved.outputPath = artifactPath
		}
	}
	if strings.TrimSpace(resolved.outputPath) == "" {
		resolved.outputPath = filepath.Join(resolved.rootPath, scaffoldWASMOutputPath())
	}
	resolved.outputPath, err = normalizePath(cwd, resolved.outputPath)
	if err != nil {
		return buildConfig{}, fmt.Errorf("resolve output path: %w", err)
	}

	profile, err := resolveBuildProfile(strings.TrimSpace(resolved.profile))
	if err != nil {
		return buildConfig{}, err
	}
	resolved.profile = profile.Name
	return resolved, nil
}

func resolveBuildProfile(profile string) (buildProfile, error) {
	switch strings.TrimSpace(strings.ToLower(profile)) {
	case "", "development", "dev":
		return buildProfile{Name: "development", Trimpath: false}, nil
	case "ci", "verification", "verify":
		return buildProfile{Name: "ci", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false"}, nil
	case "benchmark", "bench":
		return buildProfile{Name: "benchmark", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false"}, nil
	case "release", "production", "prod":
		return buildProfile{Name: "release", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false"}, nil
	default:
		return buildProfile{}, fmt.Errorf("unknown build profile %q", profile)
	}
}

func resolveReleaseConfig(config releaseConfig) (releaseConfig, error) {
	resolved := config
	explicitOutDir := strings.TrimSpace(config.outDir) != ""
	cwd, err := buildGetwd()
	if err != nil {
		return releaseConfig{}, err
	}

	metadata, metadataDir, hasMetadata, err := resolveScaffoldMetadataForConfig(cwd, resolved.rootPath, resolved.appPath)
	if err != nil {
		return releaseConfig{}, err
	}
	if hasMetadata {
		if strings.TrimSpace(resolved.appPath) == "" && strings.TrimSpace(metadata.Tooling.AppPath) != "" {
			resolved.appPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.AppPath))
		}
		if strings.TrimSpace(resolved.rootPath) == "" {
			resolved.rootPath = metadataDir
		}
		if strings.TrimSpace(resolved.outDir) == "" && strings.TrimSpace(metadata.Tooling.ReleaseOutDir) != "" {
			resolved.outDir = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.ReleaseOutDir))
		}
		if strings.TrimSpace(resolved.binaryName) == "" && strings.TrimSpace(metadata.Tooling.ReleaseBinaryName) != "" {
			resolved.binaryName = strings.TrimSpace(metadata.Tooling.ReleaseBinaryName)
		}
		if strings.TrimSpace(resolved.budgetsPath) == "" && strings.TrimSpace(metadata.Tooling.ReleaseBudgetsPath) != "" {
			resolved.budgetsPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.ReleaseBudgetsPath))
		}
		if !resolved.compressionSet {
			compressionPolicy, err := normalizeReleaseCompressionPolicy(metadata.Tooling.ReleaseCompression)
			if err != nil {
				return releaseConfig{}, err
			}
			resolved.compression = compressionPolicy
		}
	}

	if strings.TrimSpace(resolved.appPath) == "" {
		resolved.appPath, err = detectAppPath(cwd)
		if err != nil {
			return releaseConfig{}, err
		}
	}
	resolved.appPath, err = normalizeExistingPath(cwd, resolved.appPath)
	if err != nil {
		return releaseConfig{}, fmt.Errorf("resolve app path: %w", err)
	}

	appDir := resolved.appPath
	info, err := os.Stat(resolved.appPath)
	if err != nil {
		return releaseConfig{}, fmt.Errorf("inspect app path: %w", err)
	}
	if !info.IsDir() {
		appDir = filepath.Dir(resolved.appPath)
	}

	if strings.TrimSpace(resolved.rootPath) == "" {
		resolved.rootPath = appDir
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return releaseConfig{}, fmt.Errorf("resolve root path: %w", err)
	}

	if !explicitOutDir {
		if artifactPath, ok, err := resolveLauncherArtifactPath(resolved.rootPath, "wasm-release"); err != nil {
			return releaseConfig{}, err
		} else if ok {
			resolved.outDir = artifactPath
		}
	}
	if strings.TrimSpace(resolved.outDir) == "" {
		resolved.outDir = filepath.Join(resolved.rootPath, defaultScaffoldReleaseOutDir())
	}
	resolved.outDir, err = normalizePath(cwd, resolved.outDir)
	if err != nil {
		return releaseConfig{}, fmt.Errorf("resolve release output directory: %w", err)
	}

	resolved.binaryName = filepath.Base(strings.TrimSpace(firstNonEmpty(resolved.binaryName, defaultScaffoldReleaseBinaryName())))
	if resolved.binaryName == "." || resolved.binaryName == string(filepath.Separator) || resolved.binaryName == "" {
		return releaseConfig{}, errors.New("release binary name is required")
	}
	resolved.manifestName = filepath.Base(strings.TrimSpace(firstNonEmpty(resolved.manifestName, "wasm-release-manifest.json")))
	if resolved.manifestName == "." || resolved.manifestName == string(filepath.Separator) || resolved.manifestName == "" {
		return releaseConfig{}, errors.New("release manifest name is required")
	}
	if strings.TrimSpace(resolved.budgetsPath) != "" {
		resolved.budgetsPath, err = normalizePath(cwd, resolved.budgetsPath)
		if err != nil {
			return releaseConfig{}, fmt.Errorf("resolve budgets path: %w", err)
		}
	}
	compressionPolicy, err := normalizeReleaseCompressionPolicy(firstNonEmpty(resolved.compression, defaultScaffoldReleaseCompression()))
	if err != nil {
		return releaseConfig{}, err
	}
	resolved.compression = compressionPolicy
	resolved.skipCompression = compressionPolicy == "none"

	profile, err := resolveBuildProfile(strings.TrimSpace(firstNonEmpty(resolved.profile, "release")))
	if err != nil {
		return releaseConfig{}, err
	}
	resolved.profile = profile.Name
	return resolved, nil
}

func normalizeReleaseCompressionPolicy(policy string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(policy)) {
	case "", "gzip":
		return "gzip", nil
	case "brotli", "br":
		return "brotli", nil
	case "gzip+brotli", "brotli+gzip", "both", "all":
		return "gzip+brotli", nil
	case "none", "off", "disabled":
		return "none", nil
	default:
		return "", fmt.Errorf("unknown release compression policy %q", policy)
	}
}

func executeBuild(config buildConfig) (buildSummary, error) {
	profile, err := resolveBuildProfile(config.profile)
	if err != nil {
		return buildSummary{}, err
	}
	packageDir := config.appPath
	if info, err := os.Stat(config.appPath); err == nil && !info.IsDir() {
		packageDir = filepath.Dir(config.appPath)
	}
	if err := os.MkdirAll(filepath.Dir(config.outputPath), 0755); err != nil {
		return buildSummary{}, fmt.Errorf("create build output directory: %w", err)
	}

	buildArgs := []string{"build", "-o", config.outputPath}
	if profile.Trimpath {
		buildArgs = append(buildArgs, "-trimpath")
	}
	if strings.TrimSpace(profile.Ldflags) != "" {
		buildArgs = append(buildArgs, "-ldflags="+profile.Ldflags)
	}
	if strings.TrimSpace(profile.BuildVCS) != "" {
		buildArgs = append(buildArgs, "-buildvcs="+profile.BuildVCS)
	}
	buildArgs = append(buildArgs, ".")

	cmd := exec.Command("go", buildArgs...)
	cmd.Dir = packageDir
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed == "" {
			return buildSummary{}, fmt.Errorf("go build failed: %w", err)
		}
		return buildSummary{}, fmt.Errorf("go build failed: %s", trimmed)
	}

	artifactBytes, err := os.ReadFile(config.outputPath)
	if err != nil {
		return buildSummary{}, fmt.Errorf("read built wasm artifact: %w", err)
	}
	artifactInfo, err := os.Stat(config.outputPath)
	if err != nil {
		return buildSummary{}, fmt.Errorf("inspect built wasm artifact: %w", err)
	}
	hash := sha256.Sum256(artifactBytes)
	return buildSummary{
		OK:          true,
		Profile:     profile,
		AppPath:     config.appPath,
		ProjectRoot: config.rootPath,
		PackageDir:  packageDir,
		OutputPath:  config.outputPath,
		Bytes:       artifactInfo.Size(),
		SHA256:      fmt.Sprintf("%x", hash[:]),
	}, nil
}

func executeRelease(config releaseConfig) (releaseSummary, error) {
	if err := os.MkdirAll(config.outDir, 0755); err != nil {
		return releaseSummary{}, fmt.Errorf("create release output directory: %w", err)
	}
	wasmPath := filepath.Join(config.outDir, config.binaryName)
	buildSummary, err := releaseExecuteBuild(buildConfig{
		appPath:    config.appPath,
		rootPath:   config.rootPath,
		outputPath: wasmPath,
		profile:    config.profile,
	})
	if err != nil {
		return releaseSummary{}, err
	}
	artifacts := map[string]releaseArtifactRecord{}
	artifacts["wasm"], err = releaseArtifactRecordForPathFunc(config.outDir, wasmPath)
	if err != nil {
		return releaseSummary{}, err
	}
	emitGzip := config.compression == "gzip" || config.compression == "gzip+brotli"
	emitBrotli := config.compression == "brotli" || config.compression == "gzip+brotli"
	if emitGzip {
		gzipPath := wasmPath + ".gz"
		if err := releaseWriteGzipSidecar(wasmPath, gzipPath); err != nil {
			return releaseSummary{}, err
		}
		artifacts["gzip"], err = releaseArtifactRecordForPathFunc(config.outDir, gzipPath)
		if err != nil {
			return releaseSummary{}, err
		}
	}
	if emitBrotli {
		brotliPath := wasmPath + ".br"
		if err := releaseWriteBrotliSidecar(wasmPath, brotliPath); err != nil {
			return releaseSummary{}, err
		}
		artifacts["brotli"], err = releaseArtifactRecordForPathFunc(config.outDir, brotliPath)
		if err != nil {
			return releaseSummary{}, err
		}
	}
	if strings.TrimSpace(config.budgetsPath) != "" {
		budgets, err := loadReleaseBudgets(config.budgetsPath)
		if err != nil {
			return releaseSummary{}, err
		}
		if err := assertReleaseBudgets(budgets, artifacts); err != nil {
			return releaseSummary{}, err
		}
	}
	manifestPath := filepath.Join(config.outDir, config.manifestName)
	manifestPayload := map[string]interface{}{
		"package": buildSummary.PackageDir,
		"profile": buildSummary.Profile.Name,
		"goos":    "js",
		"goarch":  "wasm",
		"flags": map[string]interface{}{
			"trimpath":          buildSummary.Profile.Trimpath,
			"ldflags":           buildSummary.Profile.Ldflags,
			"buildvcs":          firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"compression":       !config.skipCompression,
			"compressionPolicy": config.compression,
			"gzip":              emitGzip,
			"brotli":            emitBrotli,
		},
		"artifacts": artifacts,
	}
	encodedManifest, err := releaseMarshalIndent(manifestPayload, "", "  ")
	if err != nil {
		return releaseSummary{}, fmt.Errorf("encode release manifest: %w", err)
	}
	encodedManifest = append(encodedManifest, '\n')
	if err := os.WriteFile(manifestPath, encodedManifest, 0644); err != nil {
		return releaseSummary{}, fmt.Errorf("write release manifest: %w", err)
	}
	return releaseSummary{
		OK:           true,
		Profile:      buildSummary.Profile,
		AppPath:      buildSummary.AppPath,
		ProjectRoot:  buildSummary.ProjectRoot,
		PackageDir:   buildSummary.PackageDir,
		OutDir:       config.outDir,
		ManifestPath: manifestPath,
		Artifacts:    artifacts,
		Flags: map[string]interface{}{
			"trimpath":          buildSummary.Profile.Trimpath,
			"ldflags":           buildSummary.Profile.Ldflags,
			"buildvcs":          firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"compression":       !config.skipCompression,
			"compressionPolicy": config.compression,
			"gzip":              emitGzip,
			"brotli":            emitBrotli,
		},
	}, nil
}

func releaseArtifactRecordForPath(baseDir string, artifactPath string) (releaseArtifactRecord, error) {
	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		return releaseArtifactRecord{}, fmt.Errorf("read release artifact: %w", err)
	}
	artifactInfo, err := os.Stat(artifactPath)
	if err != nil {
		return releaseArtifactRecord{}, fmt.Errorf("inspect release artifact: %w", err)
	}
	relPath, err := filepath.Rel(baseDir, artifactPath)
	if err != nil {
		return releaseArtifactRecord{}, fmt.Errorf("resolve release artifact path: %w", err)
	}
	hash := sha256.Sum256(artifactBytes)
	return releaseArtifactRecord{
		Path:   filepath.ToSlash(relPath),
		Bytes:  artifactInfo.Size(),
		SHA256: fmt.Sprintf("%x", hash[:]),
	}, nil
}

func writeGzipSidecar(sourcePath string, targetPath string) error {
	inputBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source artifact for gzip: %w", err)
	}
	outputFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create gzip sidecar: %w", err)
	}
	defer outputFile.Close()
	gzipWriter, err := gzip.NewWriterLevel(outputFile, gzip.BestCompression)
	if err != nil {
		return fmt.Errorf("create gzip writer: %w", err)
	}
	if _, err := gzipWriter.Write(inputBytes); err != nil {
		gzipWriter.Close()
		return fmt.Errorf("write gzip sidecar: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("finalize gzip sidecar: %w", err)
	}
	return nil
}

func writeBrotliSidecar(sourcePath string, targetPath string) error {
	inputBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source artifact for brotli: %w", err)
	}
	outputFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("create brotli sidecar: %w", err)
	}
	defer outputFile.Close()
	brotliWriter := brotli.NewWriterLevel(outputFile, brotli.BestCompression)
	if _, err := brotliWriter.Write(inputBytes); err != nil {
		brotliWriter.Close()
		return fmt.Errorf("write brotli sidecar: %w", err)
	}
	if err := brotliWriter.Close(); err != nil {
		return fmt.Errorf("finalize brotli sidecar: %w", err)
	}
	return nil
}

func loadReleaseBudgets(path string) (map[string]int64, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read budgets file: %w", err)
	}
	raw := map[string]interface{}{}
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("parse budgets file: %w", err)
	}
	budgets := map[string]int64{}
	for key, value := range raw {
		number, ok := value.(float64)
		if !ok {
			return nil, fmt.Errorf("budget %q must be numeric", key)
		}
		budgets[key] = int64(number)
	}
	return budgets, nil
}

func assertReleaseBudgets(budgets map[string]int64, artifacts map[string]releaseArtifactRecord) error {
	checks := []struct {
		budgetKey string
		artifact  string
		label     string
	}{
		{budgetKey: "raw_bytes", artifact: "wasm", label: "raw wasm"},
		{budgetKey: "gzip_bytes", artifact: "gzip", label: "gzip sidecar"},
		{budgetKey: "brotli_bytes", artifact: "brotli", label: "brotli sidecar"},
	}
	for _, check := range checks {
		limit, ok := budgets[check.budgetKey]
		if !ok {
			continue
		}
		artifact, ok := artifacts[check.artifact]
		if !ok {
			continue
		}
		if artifact.Bytes > limit {
			return fmt.Errorf("artifact budget exceeded for %s: %d bytes > %d bytes", check.label, artifact.Bytes, limit)
		}
	}
	return nil
}

func printBuildSummary(summary buildSummary) {
	fmt.Println("GWC build")
	fmt.Printf("  profile:      %s\n", summary.Profile.Name)
	fmt.Printf("  app:          %s\n", summary.AppPath)
	fmt.Printf("  project root: %s\n", summary.ProjectRoot)
	fmt.Printf("  package dir:  %s\n", summary.PackageDir)
	fmt.Printf("  output:       %s\n", summary.OutputPath)
	fmt.Printf("  bytes:        %d\n", summary.Bytes)
	fmt.Printf("  sha256:       %s\n", summary.SHA256)
	fmt.Printf("  trimpath:     %t\n", summary.Profile.Trimpath)
	fmt.Printf("  ldflags:      %s\n", firstNonEmpty(summary.Profile.Ldflags, "<none>"))
	fmt.Printf("  buildvcs:     %s\n", firstNonEmpty(summary.Profile.BuildVCS, "default"))
}

func printReleaseSummary(summary releaseSummary) {
	fmt.Println("GWC release")
	fmt.Printf("  profile:      %s\n", summary.Profile.Name)
	fmt.Printf("  app:          %s\n", summary.AppPath)
	fmt.Printf("  project root: %s\n", summary.ProjectRoot)
	fmt.Printf("  package dir:  %s\n", summary.PackageDir)
	fmt.Printf("  out dir:      %s\n", summary.OutDir)
	fmt.Printf("  manifest:     %s\n", summary.ManifestPath)
	keys := make([]string, 0, len(summary.Artifacts))
	for key := range summary.Artifacts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		artifact := summary.Artifacts[key]
		fmt.Printf("  artifact[%s]: %s (%d bytes)\n", key, artifact.Path, artifact.Bytes)
	}
}

func printVerifySummary(summary verifySummary) {
	fmt.Println("GWC verify")
	fmt.Printf("  app:          %s\n", summary.AppPath)
	fmt.Printf("  project root: %s\n", summary.ProjectRoot)
	if summary.Tests.Ran {
		fmt.Printf("  tests:        %s %s\n", summary.Tests.Command, summary.Tests.PackagePattern)
	} else {
		fmt.Println("  tests:        skipped")
	}
	fmt.Printf("  build:        %s -> %s\n", summary.Build.Profile.Name, summary.Build.OutputPath)
}

func (l launcher) runStart(args []string) error {
	fs := flag.NewFlagSet("start", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if err := startTerminalValidator(isInteractiveFile(os.Stdin), isInteractiveFile(os.Stdout)); err != nil {
		return err
	}
	selection, err := startSelectionRunner()
	if err != nil {
		return err
	}
	if selection == nil {
		return nil
	}

	result, err := startGenerateScaffold(l, *selection)
	if err != nil {
		_, postErr := startPostRunner(*selection, nil, err)
		if postErr != nil {
			return postErr
		}
		return nil
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
		Selection: selection,
		MainGo:    renderScaffoldMain(selection, repoModulePath),
		HTML:      renderScaffoldHTML(selection),
		README:    renderScaffoldREADME(selection),
		Metadata:  defaultScaffoldMetadata(selection),
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

func renderScaffoldGoMod(selection startSelection, repoModulePath string, relRepoRoot string) string {
	relRepoRoot = filepath.ToSlash(relRepoRoot)
	return fmt.Sprintf("module %s\n\ngo 1.25.0\n\nrequire %s v0.0.0\n\nreplace %s => %s\n", selection.ModulePath, repoModulePath, repoModulePath, relRepoRoot)
}

func renderScaffoldMain(selection startSelection, repoModulePath string) string {
	featureList := strings.Join(selection.Preset.Features, ", ")
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
`, repoModulePath+"/html", repoModulePath+"/ui", repoModulePath+"/utils", selection.ProjectName, selection.Description, selection.Author, selection.Version, selection.Preset.Name, selection.ModulePath, selection.Preset.Description, "Features: "+featureList)
}

func renderScaffoldHTML(selection startSelection) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>%s</title>
	<script src="./wasm_exec.js"></script>
	<style>
		:root { color-scheme: dark; }
		* { box-sizing: border-box; }
		body {
			margin: 0;
			font-family: "Segoe UI", sans-serif;
			background:
				radial-gradient(circle at top, rgba(97, 218, 251, 0.18), transparent 32%%),
				linear-gradient(160deg, #050816 0%%, #0d1326 48%%, #111c2b 100%%);
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
			width: min(560px, 100%%);
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
			background: linear-gradient(135deg, #61dafb 0%%, #35b4d8 100%%);
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
			background: linear-gradient(135deg, #61dafb 0%%, #35b4d8 100%%);
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
`, selection.ProjectName)
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
	return fmt.Sprintf("# %s\n\n%s\n\n- Author: %s\n- Version: %s\n- Preset: %s\n\n## Run\n\nFrom the GoWebComponents repo root:\n\n```powershell\ngo run ./tools/gwc dev -app %q -root %q -html %q -wasm %q\n```\n", selection.ProjectName, selection.Description, selection.Author, selection.Version, selection.Preset.Key, filepath.Join(selection.TargetDir, "main.go"), selection.TargetDir, filepath.Join(selection.TargetDir, "index.html"), scaffoldWASMOutputPath())
}

func (l launcher) resolveDevConfig(config devConfig) (devConfig, error) {
	resolved := config
	resolved.host = strings.TrimSpace(resolved.host)
	resolved.port = strings.TrimSpace(resolved.port)

	cwd, err := devGetwd()
	if err != nil {
		return devConfig{}, err
	}

	metadata, metadataDir, hasMetadata, err := resolveScaffoldMetadataForConfig(cwd, resolved.rootPath, resolved.appPath)
	if err != nil {
		return devConfig{}, err
	}
	if hasMetadata {
		if strings.TrimSpace(resolved.appPath) == "" && strings.TrimSpace(metadata.Tooling.AppPath) != "" {
			resolved.appPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.AppPath))
		}
		if strings.TrimSpace(resolved.rootPath) == "" {
			resolved.rootPath = metadataDir
		}
		if strings.TrimSpace(resolved.htmlPath) == "" && strings.TrimSpace(metadata.Tooling.HTMLPath) != "" {
			resolved.htmlPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.HTMLPath))
		}
		if strings.TrimSpace(resolved.wasmPath) == "" && strings.TrimSpace(metadata.Tooling.WASMPath) != "" {
			resolved.wasmPath = metadata.Tooling.WASMPath
		}
		if resolved.host == "" {
			resolved.host = strings.TrimSpace(metadata.Tooling.DevHost)
		}
		if resolved.port == "" {
			resolved.port = strings.TrimSpace(metadata.Tooling.DevPort)
		}
	}
	if resolved.host == "" {
		resolved.host = "127.0.0.1"
	}
	if resolved.port == "" {
		resolved.port = "8080"
	}

	if strings.TrimSpace(resolved.appPath) == "" {
		resolved.appPath, err = detectAppPath(cwd)
		if err != nil {
			return devConfig{}, err
		}
	}
	resolved.appPath, err = normalizeExistingPath(cwd, resolved.appPath)
	if err != nil {
		return devConfig{}, fmt.Errorf("resolve app path: %w", err)
	}

	appDir := resolved.appPath
	info, err := os.Stat(resolved.appPath)
	if err != nil {
		return devConfig{}, fmt.Errorf("inspect app path: %w", err)
	}
	if !info.IsDir() {
		appDir = filepath.Dir(resolved.appPath)
	}

	if strings.TrimSpace(resolved.rootPath) == "" {
		resolved.rootPath = appDir
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return devConfig{}, fmt.Errorf("resolve root path: %w", err)
	}

	if strings.TrimSpace(resolved.htmlPath) == "" {
		resolved.htmlPath = detectHTMLPath(resolved.rootPath)
	}
	if strings.TrimSpace(resolved.htmlPath) != "" {
		resolved.htmlPath, err = normalizePath(cwd, resolved.htmlPath)
		if err != nil {
			return devConfig{}, fmt.Errorf("resolve html path: %w", err)
		}
	}

	if strings.TrimSpace(resolved.wasmPath) != "" {
		resolved.wasmPath = strings.TrimSpace(resolved.wasmPath)
	}

	return resolved, nil
}

func detectAppPath(cwd string) (string, error) {
	directMain := filepath.Join(cwd, "main.go")
	if fileExists(directMain) {
		return directMain, nil
	}
	cmdWebMain := filepath.Join(cwd, "cmd", "web", "main.go")
	if fileExists(cmdWebMain) {
		return cmdWebMain, nil
	}
	return "", errors.New("dev could not detect an app entrypoint; pass -app or run from a directory with main.go or cmd/web/main.go")
}

func detectHTMLPath(root string) string {
	indexPath := filepath.Join(root, "index.html")
	if fileExists(indexPath) {
		return indexPath
	}

	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".html") {
			return filepath.Join(root, entry.Name())
		}
	}
	return ""
}

func resolveScaffoldMetadataForConfig(cwd string, configuredRoot string, configuredApp string) (scaffoldMetadata, string, bool, error) {
	candidates := []string{}
	for _, candidate := range []string{configuredRoot, configuredApp, cwd} {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		resolved := candidate
		if !filepath.IsAbs(resolved) {
			absolute, err := normalizePath(cwd, resolved)
			if err != nil {
				return scaffoldMetadata{}, "", false, err
			}
			resolved = absolute
		}
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() {
			resolved = filepath.Dir(resolved)
		}
		alreadyIncluded := false
		for _, existing := range candidates {
			if existing == resolved {
				alreadyIncluded = true
				break
			}
		}
		if !alreadyIncluded {
			candidates = append(candidates, resolved)
		}
	}
	for _, candidate := range candidates {
		metadata, ok, err := loadScaffoldMetadata(candidate)
		if err != nil {
			return scaffoldMetadata{}, "", false, err
		}
		if ok {
			return metadata, candidate, true, nil
		}
	}
	return scaffoldMetadata{}, "", false, nil
}

func loadScaffoldMetadata(dir string) (scaffoldMetadata, bool, error) {
	metadataPath := filepath.Join(strings.TrimSpace(dir), "gwc-start.json")
	if !fileExists(metadataPath) {
		return scaffoldMetadata{}, false, nil
	}
	content, err := os.ReadFile(metadataPath)
	if err != nil {
		return scaffoldMetadata{}, false, fmt.Errorf("read scaffold metadata: %w", err)
	}
	var metadata scaffoldMetadata
	if err := json.Unmarshal(content, &metadata); err != nil {
		return scaffoldMetadata{}, false, fmt.Errorf("parse scaffold metadata: %w", err)
	}
	return metadata, true, nil
}

func normalizeExistingPath(base string, target string) (string, error) {
	resolved, err := normalizePath(base, target)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(resolved); err != nil {
		return "", err
	}
	return resolved, nil
}

func normalizePath(base string, target string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", nil
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target), nil
	}
	return filepath.Abs(filepath.Join(base, target))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func printDevPlan(config devConfig) {
	plan := describeDevPlan(config)
	fmt.Println("GWC dev plan")
	fmt.Printf("  project root:  %s\n", plan.ProjectRoot)
	fmt.Printf("  app mode:      %s\n", plan.AppMode)
	fmt.Printf("  server mode:   %s\n", plan.ServerMode)
	fmt.Printf("  app:           %s\n", config.appPath)
	fmt.Printf("  root:          %s\n", config.rootPath)
	if config.htmlPath != "" {
		fmt.Printf("  html:          %s\n", config.htmlPath)
	} else {
		fmt.Println("  html:          <auto-detect skipped>")
	}
	if config.wasmPath != "" {
		fmt.Printf("  wasm:          %s\n", config.wasmPath)
	} else {
		fmt.Println("  wasm:          <livereload default>")
	}
	fmt.Printf("  host:          %s\n", config.host)
	fmt.Printf("  port:          %s\n", config.port)
	fmt.Printf("  hot:           %t\n", config.hot)
	fmt.Printf("  listening URL: %s\n", plan.ListeningURL)
}

func printDevPlanJSON(config devConfig) error {
	plan := describeDevPlan(config)
	payload := map[string]interface{}{
		"app":          config.appPath,
		"root":         config.rootPath,
		"projectRoot":  plan.ProjectRoot,
		"appMode":      plan.AppMode,
		"serverMode":   plan.ServerMode,
		"html":         config.htmlPath,
		"wasm":         config.wasmPath,
		"host":         config.host,
		"port":         config.port,
		"hot":          config.hot,
		"listeningURL": plan.ListeningURL,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(payload)
}

type devPlanSummary struct {
	ProjectRoot  string
	AppMode      string
	ServerMode   string
	ListeningURL string
}

func describeDevPlan(config devConfig) devPlanSummary {
	projectRoot := strings.TrimSpace(config.rootPath)
	if projectRoot == "" {
		projectRoot = filepath.Dir(strings.TrimSpace(config.appPath))
	}
	appMode := "client-only-wasm"
	serverMode := "livereload-wasm"
	if strings.Contains(strings.ToLower(filepath.ToSlash(strings.TrimSpace(config.appPath))), "/cmd/web/main.go") {
		appMode = "server-app"
		serverMode = "server-entrypoint"
	}
	return devPlanSummary{
		ProjectRoot:  projectRoot,
		AppMode:      appMode,
		ServerMode:   serverMode,
		ListeningURL: "http://" + joinHostPort(config.host, config.port),
	}
}

func printUsage() {
	fmt.Println("GWC launcher")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run ./tools/gwc <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build      Build a js/wasm app with an explicit launcher profile")
	fmt.Println("  test       Run explicit launcher-owned test lanes such as unit, wasm, hydration, browser, and release")
	fmt.Println("  examples   Serve the examples catalog from a Go-native server")
	fmt.Println("  dev        Run the Go-native dev entrypoint and forward to livereload")
	fmt.Println("  doctor     Check local toolchains, runtime assets, project signals, and port availability")
	fmt.Println("  import     Convert a static HTML or JSX file into an inspectable GWC project")
	fmt.Println("  release    Package a js/wasm release with manifest and compressed sidecars")
	fmt.Println("  verify     Run app-local Go tests when present and perform a CI-profile wasm build")
	fmt.Println("  start      Run the scaffold TUI for preset and project setup")
}

func printTestSummary(summary testSummary) {
	fmt.Println("GWC test")
	if summary.AppPath != "" {
		fmt.Printf("  app:          %s\n", summary.AppPath)
	}
	fmt.Printf("  project root: %s\n", summary.ProjectRoot)
	fmt.Printf("  lanes:        %s\n", strings.Join(summary.SelectedLanes, ", "))
	for _, lane := range summary.Lanes {
		status := "ok"
		if lane.Skipped {
			status = "skipped"
		}
		fmt.Printf("  [%s] %s", status, lane.Name)
		if lane.Summary != "" {
			fmt.Printf(": %s", lane.Summary)
		}
		fmt.Println()
	}
}

func projectHasGoTests(rootPath string) (bool, error) {
	found := false
	err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipTestWalkDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(entry.Name(), "_test.go") {
			found = true
			return errStopWalk
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStopWalk) {
		return false, fmt.Errorf("scan project tests: %w", err)
	}
	return found, nil
}

func (l launcher) buildDoctorReport(config doctorConfig) doctorReport {
	cwd, err := doctorGetwd()
	if err != nil {
		cwd = ""
	}
	report := doctorReport{
		OK:      true,
		Checked: time.Now().UTC().Format(time.RFC3339),
		CWD:     cwd,
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

	return report
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
	if !fileExists(playwrightPackagePath) {
		return doctorCheck{Name: "Browser tests", Status: "warn", Summary: fmt.Sprintf("Playwright dependencies are not installed under %s.", filepath.Join(workspace, "node_modules")), Hint: fmt.Sprintf("Run `npm install` in %s before browser suites or launcher verify flows.", workspace)}
	}
	return doctorCheck{Name: "Browser tests", Status: "pass", Summary: fmt.Sprintf("Playwright is installed at %s", playwrightPackagePath)}
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
}

func resolveRepoRoot() (string, error) {
	_, currentFile, _, ok := resolveRepoRootCaller(0)
	if !ok {
		return "", errors.New("unable to resolve launcher source path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	if _, err := os.Stat(filepath.Join(repoRoot, "go.mod")); err != nil {
		return "", fmt.Errorf("unable to resolve repo root from %s", repoRoot)
	}
	return repoRoot, nil
}

func joinHostPort(host string, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" {
		host = defaultHost
	}
	if port == "" {
		port = defaultPort
	}
	return host + ":" + port
}

func applyDevHeaders(w http.ResponseWriter, r *http.Request) {
	if strings.HasSuffix(strings.ToLower(r.URL.Path), ".wasm") {
		w.Header().Set("Content-Type", "application/wasm")
	}
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func firstHTMLFileName(dirPath string) (string, bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", false, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			return entry.Name(), true, nil
		}
	}
	return "", false, nil
}

func (l launcher) buildExampleCatalogEntry(dirPath string, dirName string, resolveHref func(dirPath string, dirName string, htmlFile string) string) (exampleCatalogEntry, bool, error) {
	htmlFile, ok, err := firstHTMLFileName(dirPath)
	if err != nil || !ok {
		return exampleCatalogEntry{}, ok, err
	}
	htmlPath := filepath.Join(dirPath, htmlFile)
	wasmBinary, usesWasm, err := detectAvailableExampleWasmBinary(l.resolvedExamplesWasmDir(), htmlPath)
	if err != nil {
		return exampleCatalogEntry{}, false, err
	}
	title, _ := detectHTMLTitle(htmlPath)
	tags := exampleCatalogTags(dirName, wasmBinary, usesWasm)
	href := "/examples/" + dirName + "/"
	if resolveHref != nil {
		href = resolveHref(dirPath, dirName, htmlFile)
	}
	return exampleCatalogEntry{
		Name:        dirName,
		Href:        href,
		HTMLFile:    htmlFile,
		Title:       title,
		UsesWasm:    usesWasm,
		WasmBinary:  wasmBinary,
		MultiClient: hasAnyTag(tags, "multi-client", "cross-tab", "multi-window"),
		Tags:        tags,
	}, true, nil
}

func (l launcher) resolveExampleCatalogEntry(dirName string) (exampleCatalogEntry, bool, error) {
	dirName = strings.TrimSpace(dirName)
	if dirName == "" {
		return exampleCatalogEntry{}, false, nil
	}
	dirPath := filepath.Join(l.examplesDir, dirName)
	info, err := os.Stat(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return exampleCatalogEntry{}, false, nil
		}
		return exampleCatalogEntry{}, false, err
	}
	if !info.IsDir() {
		return exampleCatalogEntry{}, false, nil
	}
	return l.buildExampleCatalogEntry(dirPath, dirName, func(dirPath string, dirName string, htmlFile string) string {
		return "/examples/" + dirName + "/"
	})
}

func (l launcher) resolveGeneratedExamplePage(routePath string) (generatedExamplePage, bool, error) {
	trimmed := strings.Trim(strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(routePath)), "/examples/"), "/")
	if trimmed == "" {
		return generatedExamplePage{}, false, nil
	}
	parts := strings.Split(trimmed, "/")
	dirName := strings.TrimSpace(parts[0])
	if dirName == "" {
		return generatedExamplePage{}, false, nil
	}
	entry, ok, err := l.resolveExampleCatalogEntry(dirName)
	if err != nil || !ok {
		return generatedExamplePage{}, ok, err
	}
	if !entry.UsesWasm {
		return generatedExamplePage{}, false, nil
	}
	manifestHref := ""
	if fileExists(filepath.Join(l.examplesDir, dirName, "manifest.webmanifest")) {
		manifestHref = "./manifest.webmanifest"
	}

	return generatedExamplePage{
		RoutePath:     routePath,
		DirName:       dirName,
		HTMLFile:      entry.HTMLFile,
		Title:         firstNonEmpty(entry.Title, defaultExampleTitle(dirName, entry.HTMLFile)),
		WasmBinary:    entry.WasmBinary,
		ManifestHref:  manifestHref,
		Description:   fmt.Sprintf("Generated wasm host page for %s. The Go examples server sends the compiled wasm bundle and a #app mount container to the browser.", dirName),
		GeneratedFrom: entry.HTMLFile,
	}, true, nil
}

func detectExampleWasmBinary(htmlPath string) (string, bool, error) {
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", false, err
	}
	matches := regexp.MustCompile(`static/bin/([A-Za-z0-9._-]+\.wasm)`).FindSubmatch(content)
	if len(matches) < 2 {
		return "", false, nil
	}
	return string(matches[1]), true, nil
}

func detectAvailableExampleWasmBinary(wasmDir string, htmlPath string) (string, bool, error) {
	wasmBinary, usesWasm, err := detectExampleWasmBinary(htmlPath)
	if err != nil || !usesWasm {
		return wasmBinary, usesWasm, err
	}
	wasmBinary = strings.TrimSpace(wasmBinary)
	if wasmBinary == "" {
		return "", false, nil
	}
	if !fileExists(filepath.Join(wasmDir, wasmBinary)) {
		return "", false, nil
	}
	return wasmBinary, true, nil
}

func detectHTMLTitle(htmlPath string) (string, error) {
	content, err := os.ReadFile(htmlPath)
	if err != nil {
		return "", err
	}
	matches := regexp.MustCompile(`(?is)<title>(.*?)</title>`).FindSubmatch(content)
	if len(matches) < 2 {
		return "", nil
	}
	title := strings.TrimSpace(string(matches[1]))
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Join(strings.Fields(title), " ")
	return title, nil
}

func defaultExampleTitle(dirName string, htmlFile string) string {
	base := strings.TrimSuffix(htmlFile, filepath.Ext(htmlFile))
	if base == "" {
		base = dirName
	}
	base = strings.ReplaceAll(base, "-", " ")
	base = strings.ReplaceAll(base, "_", " ")
	parts := strings.Fields(base)
	for index, part := range parts {
		parts[index] = strings.ToUpper(part[:1]) + part[1:]
	}
	if len(parts) == 0 {
		return dirName
	}
	return strings.Join(parts, " ") + " - GoWebComponents"
}

func exampleCatalogTags(dirName string, wasmBinary string, usesWasm bool) []string {
	trimmedName := dirName
	if parts := strings.SplitN(dirName, "-", 2); len(parts) == 2 {
		trimmedName = parts[1]
	}
	nameLower := strings.ToLower(trimmedName)

	seen := map[string]struct{}{}
	tags := make([]string, 0, 8)
	addTag := func(tag string) {
		tag = strings.TrimSpace(strings.ToLower(tag))
		if tag == "" {
			return
		}
		if _, ok := seen[tag]; ok {
			return
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	}

	if usesWasm {
		addTag("wasm")
	}
	if strings.TrimSpace(wasmBinary) != "" {
		addTag(strings.TrimSuffix(strings.ToLower(wasmBinary), ".wasm"))
	}

	for _, token := range strings.FieldsFunc(strings.ToLower(trimmedName), func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	}) {
		addTag(token)
	}

	addFrameworkTags := func() {
		switch {
		case strings.HasPrefix(dirName, "21-") || strings.HasPrefix(dirName, "22-") || strings.HasPrefix(dirName, "23-") || strings.HasPrefix(dirName, "24-") || strings.HasPrefix(dirName, "25-") || strings.HasPrefix(dirName, "26-") || strings.HasPrefix(dirName, "27-") || strings.HasPrefix(dirName, "28-") || strings.HasPrefix(dirName, "29-") || strings.HasPrefix(dirName, "30-") || strings.HasPrefix(dirName, "31-") || strings.HasPrefix(dirName, "32-") || strings.HasPrefix(dirName, "33-") || strings.HasPrefix(dirName, "34-") || strings.HasPrefix(dirName, "35-") || strings.HasPrefix(dirName, "36-") || strings.HasPrefix(dirName, "46-") || strings.HasPrefix(dirName, "47-") || strings.HasPrefix(dirName, "48-") || strings.HasPrefix(dirName, "49-") || strings.HasPrefix(dirName, "50-") || strings.HasPrefix(dirName, "51-") || strings.HasPrefix(dirName, "75-") || strings.HasPrefix(dirName, "76-") || strings.HasPrefix(dirName, "77-") || strings.HasPrefix(dirName, "78-") || strings.HasPrefix(dirName, "79-") || strings.HasPrefix(dirName, "80-") || strings.HasPrefix(dirName, "81-") || strings.HasPrefix(dirName, "82-"):
			addTag("ui")
		case strings.HasPrefix(dirName, "01-") || strings.HasPrefix(dirName, "02-") || strings.HasPrefix(dirName, "03-") || strings.HasPrefix(dirName, "04-") || strings.HasPrefix(dirName, "05-") || strings.HasPrefix(dirName, "06-") || strings.HasPrefix(dirName, "07-") || strings.HasPrefix(dirName, "08-") || strings.HasPrefix(dirName, "09-") || strings.HasPrefix(dirName, "10-") || strings.HasPrefix(dirName, "11-") || strings.HasPrefix(dirName, "12-") || strings.HasPrefix(dirName, "13-") || strings.HasPrefix(dirName, "14-") || strings.HasPrefix(dirName, "15-") || strings.HasPrefix(dirName, "16-") || strings.HasPrefix(dirName, "17-") || strings.HasPrefix(dirName, "18-") || strings.HasPrefix(dirName, "19-") || strings.HasPrefix(dirName, "20-"):
			addTag("ui")
		}

		switch {
		case strings.HasPrefix(dirName, "37-") || strings.HasPrefix(dirName, "38-") || strings.HasPrefix(dirName, "39-") || strings.HasPrefix(dirName, "40-") || strings.HasPrefix(dirName, "41-"):
			addTag("state")
		}

		switch {
		case strings.HasPrefix(dirName, "42-") || strings.HasPrefix(dirName, "43-") || strings.HasPrefix(dirName, "44-") || strings.HasPrefix(dirName, "45-") || strings.HasPrefix(dirName, "93-"):
			addTag("fetch")
		}

		switch {
		case strings.HasPrefix(dirName, "52-") || strings.HasPrefix(dirName, "53-") || strings.HasPrefix(dirName, "54-") || strings.HasPrefix(dirName, "88-") || strings.HasPrefix(dirName, "89-"):
			addTag("html")
		}

		switch {
		case strings.HasPrefix(dirName, "55-") || strings.HasPrefix(dirName, "56-") || strings.HasPrefix(dirName, "57-") || strings.HasPrefix(dirName, "58-") || strings.HasPrefix(dirName, "59-") || strings.HasPrefix(dirName, "60-") || strings.HasPrefix(dirName, "61-") || strings.HasPrefix(dirName, "62-") || strings.HasPrefix(dirName, "63-") || strings.HasPrefix(dirName, "64-") || strings.HasPrefix(dirName, "65-") || strings.HasPrefix(dirName, "92-") || strings.HasPrefix(dirName, "96-"):
			addTag("router")
		}

		switch {
		case strings.HasPrefix(dirName, "66-") || strings.HasPrefix(dirName, "67-") || strings.HasPrefix(dirName, "68-") || strings.HasPrefix(dirName, "69-") || strings.HasPrefix(dirName, "98-"):
			addTag("devtools")
		}

		switch {
		case strings.HasPrefix(dirName, "70-") || strings.HasPrefix(dirName, "71-") || strings.HasPrefix(dirName, "72-") || strings.HasPrefix(dirName, "73-") || strings.HasPrefix(dirName, "74-") || strings.HasPrefix(dirName, "84-") || strings.HasPrefix(dirName, "87-"):
			addTag("ssr")
		}

		switch {
		case strings.HasPrefix(dirName, "71-") || strings.HasPrefix(dirName, "72-") || strings.HasPrefix(dirName, "74-") || strings.HasPrefix(dirName, "84-"):
			addTag("hydration")
		}

		switch {
		case strings.HasPrefix(dirName, "83-") || strings.HasPrefix(dirName, "84-") || strings.HasPrefix(dirName, "85-"):
			addTag("i18n")
		}

		switch {
		case strings.HasPrefix(dirName, "90-") || strings.HasPrefix(dirName, "91-") || strings.HasPrefix(dirName, "94-") || strings.HasPrefix(dirName, "95-") || strings.Contains(nameLower, "multi-client") || strings.Contains(nameLower, "cross-tab") || strings.Contains(nameLower, "multi-window"):
			addTag("interop")
		}

		if strings.Contains(nameLower, "pwa") {
			addTag("pwa")
		}
		if strings.Contains(nameLower, "offline") {
			addTag("offline")
		}
		if strings.Contains(nameLower, "form") {
			addTag("forms")
		}
		if strings.Contains(nameLower, "worker") {
			addTag("workers")
		}
		if strings.Contains(nameLower, "overlay") || strings.Contains(nameLower, "portal") {
			addTag("overlays")
		}
		if strings.Contains(nameLower, "accessible") || strings.Contains(nameLower, "accessibility") {
			addTag("accessibility")
		}
		if strings.Contains(nameLower, "web-components") || strings.Contains(nameLower, "custom-element") || strings.Contains(nameLower, "custom-elements") {
			addTag("custom-elements")
		}
		if strings.Contains(nameLower, "transition") || strings.Contains(nameLower, "deferred") || strings.Contains(nameLower, "debounced") || strings.Contains(nameLower, "throttled") || strings.Contains(nameLower, "goroutines") {
			addTag("scheduling")
		}
		if strings.Contains(nameLower, "code-splitting") {
			addTag("code-splitting")
		}
	}

	addFrameworkTags()

	joined := nameLower
	if strings.Contains(joined, "multi-client") {
		addTag("multi-client")
	}
	if strings.Contains(joined, "cross-tab") {
		addTag("cross-tab")
	}
	if strings.Contains(joined, "multi-window") {
		addTag("multi-window")
	}
	if strings.Contains(joined, "pwa") {
		addTag("pwa")
	}
	if strings.Contains(joined, "ssr") {
		addTag("ssr")
	}
	if strings.Contains(joined, "router") {
		addTag("router")
	}
	if strings.Contains(joined, "state") {
		addTag("state")
	}
	if strings.Contains(joined, "fetch") {
		addTag("fetch")
	}
	if strings.Contains(joined, "devtools") {
		addTag("devtools")
	}

	return tags
}

func hasAnyTag(tags []string, expected ...string) bool {
	for _, tag := range tags {
		for _, candidate := range expected {
			if tag == candidate {
				return true
			}
		}
	}
	return false
}

func renderExamplesAppShellHTML(routePath string, catalogHref string) string {
	routePath = strings.TrimSpace(routePath)
	if routePath == "" {
		routePath = "/examples/"
	}
	catalogHref = strings.TrimSpace(catalogHref)
	if catalogHref == "" {
		catalogHref = "/examples/"
	}
	return renderExamplesShellHTML(examplesShellDocument{
		Title:             "GoWebComponents Examples",
		Description:       "GoWebComponents examples catalog powered by a Go server and a wasm-first, multi-client catalog app.",
		BodyClass:         "example-shell bg-[#08111d] text-white min-h-screen",
		RoutePath:         routePath,
		CatalogHref:       catalogHref,
		WasmURL:           "/static/bin/gwc-examples-site.wasm",
		FailureTitle:      "GoWebComponents Examples",
		FailureMessage:    "The wasm catalog failed to start.",
		FailureHref:       "/examples/list",
		FailureLinkLabel:  "Open raw examples listing",
		NoScriptMessage:   "The primary catalog now boots from a Go/wasm app. Enable JavaScript to browse the interactive multi-client catalog, or use the raw listing below.",
		NoScriptHref:      "/examples/list",
		NoScriptLinkLabel: "Open raw examples listing",
	})
}

func renderGeneratedExampleHTML(page generatedExamplePage) string {
	return renderExamplesShellHTML(examplesShellDocument{
		Title:            page.Title,
		Description:      page.Description,
		BodyClass:        "example-shell bg-[#08111d] text-white min-h-screen",
		BodyData:         map[string]string{"gwc-example": page.DirName, "gwc-entry": page.HTMLFile},
		RoutePath:        page.RoutePath,
		ExampleSlug:      page.DirName,
		ManifestHref:     page.ManifestHref,
		WasmURL:          "/static/bin/" + page.WasmBinary,
		FailureTitle:     page.Title,
		FailureMessage:   "Failed to start the example wasm bundle.",
		FailureHref:      "/examples/static/index.html",
		FailureLinkLabel: "Back to examples catalog",
	})
}

type examplesShellDocument struct {
	Title             string
	Description       string
	BodyClass         string
	BodyData          map[string]string
	RoutePath         string
	CatalogHref       string
	ExampleSlug       string
	ManifestHref      string
	WasmURL           string
	FailureTitle      string
	FailureMessage    string
	FailureHref       string
	FailureLinkLabel  string
	NoScriptMessage   string
	NoScriptHref      string
	NoScriptLinkLabel string
}

func renderExamplesShellHTML(document examplesShellDocument) string {
	bootstrapScript := renderExamplesBootstrapDataScript(document)
	loaderScript := renderExamplesLoaderScriptTag(document.WasmURL, document.FailureTitle, document.FailureMessage, document.FailureHref, document.FailureLinkLabel)
	headChildren := []ui.Node{
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"charset": "utf-8"}}),
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"name": "viewport", "content": "width=device-width, initial-scale=1"}}),
		gwchtml.Meta(gwchtml.Props{Raw: map[string]interface{}{"name": "description", "content": document.Description}}),
		gwchtml.Title(gwchtml.Props{}, gwchtml.Text(document.Title)),
		gwchtml.Link(gwchtml.Props{Rel: "stylesheet", Href: "/static/css/tailwind.css"}),
		gwchtml.Link(gwchtml.Props{Rel: "stylesheet", Href: "/static/css/example-shell.css"}),
		gwchtml.Script(gwchtml.Props{Src: "/static/script/wasm_exec.js"}),
		gwchtml.Script(gwchtml.Props{Src: "/static/script/example-logger.js"}),
	}
	if strings.TrimSpace(document.ManifestHref) != "" {
		headChildren = append(headChildren, gwchtml.Link(gwchtml.Props{Rel: "manifest", Href: document.ManifestHref}))
	}

	bodyChildren := []ui.Node{gwchtml.Div(gwchtml.Props{ID: "app"})}
	if strings.TrimSpace(document.NoScriptMessage) != "" {
		bodyChildren = append(bodyChildren,
			gwchtml.NoScript(gwchtml.Props{},
				gwchtml.Main(gwchtml.Props{Style: map[string]string{"max-width": "72rem", "margin": "0 auto", "padding": "2rem", "font-family": "'Segoe UI Variable', 'Segoe UI', sans-serif"}},
					gwchtml.H1(gwchtml.Props{}, gwchtml.Text(document.FailureTitle)),
					gwchtml.P(gwchtml.Props{}, gwchtml.Text(document.NoScriptMessage)),
					gwchtml.P(gwchtml.Props{}, gwchtml.A(gwchtml.Props{Href: document.NoScriptHref}, gwchtml.Text(document.NoScriptLinkLabel))),
				),
			),
		)
	}

	bodyProps := gwchtml.Props{Class: document.BodyClass, Data: document.BodyData}
	markup, err := renderExamplesToString(gwchtml.Html(gwchtml.Props{Raw: map[string]interface{}{"lang": "en"}},
		gwchtml.Head(gwchtml.Props{}, headChildren...),
		gwchtml.Body(bodyProps, bodyChildren...),
	))
	if err != nil {
		return renderExamplesShellHTMLFallback(document, bootstrapScript)
	}
	if bootstrapScript != "" {
		markup = strings.Replace(markup, `<div id="app"></div>`, `<div id="app"></div>`+bootstrapScript, 1)
	}
	if loaderScript != "" {
		markup = strings.Replace(markup, `</body>`, loaderScript+`</body>`, 1)
	}
	return "<!doctype html>\n" + markup
}

func renderExamplesBootstrapDataScript(document examplesShellDocument) string {
	bootstrap := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: document.RoutePath},
		Data: map[string]interface{}{
			"examples": map[string]interface{}{
				"mode":        "server",
				"catalogURL":  "/examples/catalog.json",
				"assetBase":   "/static/",
				"wasmBase":    "/static/bin/",
				"catalogHref": document.CatalogHref,
				"slug":        document.ExampleSlug,
			},
		},
	}
	script, err := renderExamplesUIBootstrapScript(bootstrap, "")
	if err != nil {
		return ""
	}
	return script
}

func renderExamplesBootstrapScript(wasmURL string, failureTitle string, failureMessage string, failureHref string, failureLinkLabel string) string {
	return "const GWC_EXAMPLES_CACHE = 'gwc-examples-runtime-v1';\n" +
		"async function loadCachedWasm(url, importObject) {\n" +
		"  if (!('caches' in globalThis)) {\n" +
		"    const response = await fetch(url, { cache: 'no-store' });\n" +
		"    if (!response.ok) {\n" +
		"      throw new Error('Failed to fetch wasm: ' + response.status + ' ' + response.statusText);\n" +
		"    }\n" +
		"    const bytes = await response.arrayBuffer();\n" +
		"    console.info('[gwc examples] wasm source: network (no Cache Storage)', url);\n" +
		"    return WebAssembly.instantiate(bytes, importObject);\n" +
		"  }\n" +
		"  const cache = await caches.open(GWC_EXAMPLES_CACHE);\n" +
		"  let response = await cache.match(url);\n" +
		"  let source = 'Cache Storage';\n" +
		"  if (!response) {\n" +
		"    response = await fetch(url, { cache: 'no-store' });\n" +
		"    if (!response.ok) {\n" +
		"      throw new Error('Failed to fetch wasm: ' + response.status + ' ' + response.statusText);\n" +
		"    }\n" +
		"    await cache.put(url, response.clone());\n" +
		"    source = 'network';\n" +
		"  }\n" +
		"  const bytes = await response.arrayBuffer();\n" +
		"  console.info('[gwc examples] wasm source: ' + source, url);\n" +
		"  return WebAssembly.instantiate(bytes, importObject);\n" +
		"}\n" +
		"const go = new Go();\n" +
		"loadCachedWasm(" + jsStringLiteral(wasmURL) + ", go.importObject)\n" +
		"  .then(result => go.run(result.instance))\n" +
		"  .catch(error => {\n" +
		"    const root = document.getElementById('app');\n" +
		"    if (root) {\n" +
		"      root.innerHTML = '<main style=\"max-width:72rem;margin:0 auto;padding:2rem;font-family:\\'Segoe UI Variable\\',\\'Segoe UI\\',sans-serif;\"><h1>" + escapeHTML(jsSingleQuoted(failureTitle)) + "</h1><p>" + escapeHTML(jsSingleQuoted(failureMessage)) + "</p><p><a href=\"" + escapeHTML(failureHref) + "\">" + escapeHTML(jsSingleQuoted(failureLinkLabel)) + "</a></p></main>';\n" +
		"    }\n" +
		"    console.error(error);\n" +
		"  });"
}

func renderExamplesLoaderScriptTag(wasmURL string, failureTitle string, failureMessage string, failureHref string, failureLinkLabel string) string {
	scriptBody := renderExamplesBootstrapScriptFunc(wasmURL, failureTitle, failureMessage, failureHref, failureLinkLabel)
	if strings.TrimSpace(scriptBody) == "" {
		return ""
	}
	return `<script>` + scriptBody + `</script>`
}

func renderExamplesShellHTMLFallback(document examplesShellDocument, bootstrapScript string) string {
	manifestLink := ""
	if strings.TrimSpace(document.ManifestHref) != "" {
		manifestLink = "\n  <link rel=\"manifest\" href=\"" + escapeHTML(document.ManifestHref) + "\">"
	}
	noscript := ""
	if strings.TrimSpace(document.NoScriptMessage) != "" {
		noscript = "\n  <noscript><main style=\"max-width:72rem;margin:0 auto;padding:2rem;font-family:'Segoe UI Variable','Segoe UI',sans-serif;\"><h1>" + escapeHTML(document.FailureTitle) + "</h1><p>" + escapeHTML(document.NoScriptMessage) + "</p><p><a href=\"" + escapeHTML(document.NoScriptHref) + "\">" + escapeHTML(document.NoScriptLinkLabel) + "</a></p></main></noscript>"
	}
	bodyAttrs := " class=\"" + escapeHTML(document.BodyClass) + "\""
	for key, value := range document.BodyData {
		bodyAttrs += " data-" + escapeHTML(key) + "=\"" + escapeHTML(value) + "\""
	}
	return "<!doctype html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <meta name=\"description\" content=\"" + escapeHTML(document.Description) + "\">" + manifestLink + "\n  <title>" + escapeHTML(document.Title) + "</title>\n  <link rel=\"stylesheet\" href=\"/static/css/tailwind.css\">\n  <link rel=\"stylesheet\" href=\"/static/css/example-shell.css\">\n  <script src=\"/static/script/wasm_exec.js\"></script>\n  <script src=\"/static/script/example-logger.js\"></script>\n</head>\n<body" + bodyAttrs + ">\n  <div id=\"app\"></div>" + noscript + bootstrapScript + "\n  <script>\n" + renderExamplesBootstrapScript(document.WasmURL, document.FailureTitle, document.FailureMessage, document.FailureHref, document.FailureLinkLabel) + "\n  </script>\n</body>\n</html>"
}

func jsStringLiteral(value string) string {
	return "'" + jsSingleQuoted(value) + "'"
}

func jsSingleQuoted(text string) string {
	text = strings.ReplaceAll(text, `\`, `\\`)
	text = strings.ReplaceAll(text, `'`, `\'`)
	return text
}

func renderExamplesListingHTML(links []exampleLink, query string) string {
	var items strings.Builder
	for _, link := range links {
		items.WriteString(`<li><a href="` + link.Href + `">` + link.Name + `</a></li>`)
	}
	if items.Len() == 0 {
		items.WriteString(`<li>No examples matched this search yet.</li>`)
	}
	metaText := "Generated from example folders under /examples."
	if strings.TrimSpace(query) != "" {
		metaText = fmt.Sprintf("Filtered examples for %q.", query)
	}
	return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>GoWebComponents Examples</title>
  <style>
    body { font-family: Segoe UI, Arial, sans-serif; margin: 2rem; line-height: 1.5; }
    h1 { margin-top: 0; }
    ul { padding-left: 1.2rem; }
    li { margin: 0.35rem 0; }
    a { color: #0b63ce; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .meta { margin-bottom: 1rem; color: #555; }
		.search { display: flex; gap: 0.5rem; margin: 1rem 0 1.25rem; flex-wrap: wrap; }
		.search input { min-width: 18rem; max-width: 28rem; padding: 0.55rem 0.7rem; font: inherit; }
		.search button { padding: 0.55rem 0.85rem; font: inherit; cursor: pointer; }
  </style>
</head>
<body>
  <h1>GoWebComponents Examples</h1>
	<p class="meta">` + metaText + `</p>
  <p><a href="/examples/static/index.html">Open styled showcase page</a></p>
	<form class="search" method="get" action="/examples/list">
		<input type="search" name="q" value="` + escapeHTML(query) + `" placeholder="Search examples by keyword">
		<button type="submit">Filter</button>
	</form>
  <ul>` + items.String() + `</ul>
</body>
</html>`
}

func escapeHTML(text string) string {
	replacer := strings.NewReplacer(
		"&", "&amp;",
		`"`, "&quot;",
		"<", "&lt;",
		">", "&gt;",
	)
	return replacer.Replace(text)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

var _ http.ResponseWriter = (*statusWriter)(nil)
