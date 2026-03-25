package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
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
	"github.com/monstercameron/GoWebComponents/pwa"
	playwright "github.com/playwright-community/playwright-go"
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
	return resolveLauncherExamplesWasmDir(l.repoRoot, l.staticDir)
}

type buildConfig struct {
	appPath    string
	rootPath   string
	outputPath string
	profile    string
	json       bool
	resolution map[string]string
}

type buildProfile struct {
	Name     string `json:"name"`
	Trimpath bool   `json:"trimpath"`
	Ldflags  string `json:"ldflags,omitempty"`
	BuildVCS string `json:"buildvcs,omitempty"`
}

type buildSummary struct {
	OK          bool              `json:"ok"`
	Profile     buildProfile      `json:"profile"`
	AppPath     string            `json:"appPath"`
	ProjectRoot string            `json:"projectRoot"`
	PackageDir  string            `json:"packageDir"`
	OutputPath  string            `json:"outputPath"`
	Bytes       int64             `json:"bytes"`
	SHA256      string            `json:"sha256"`
	Resolution  map[string]string `json:"resolution,omitempty"`
}

type releaseConfig struct {
	appPath          string
	rootPath         string
	outDir           string
	binaryName       string
	manifestName     string
	budgetsPath      string
	compareManifest  string
	profile          string
	compression      string
	postLinkOpt      string
	sizeAttribution  string
	startupMeasure   string
	startupTimeoutMs int
	validateSmoke    bool
	compressionSet   bool
	skipCompression  bool
	skipCompressSet  bool
	json             bool
	resolution       map[string]string
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
	Optimizer    *releaseOptimizerRecord          `json:"optimizer,omitempty"`
	Attribution  *releaseAttributionRecord        `json:"attribution,omitempty"`
	Diff         *releaseDiffArtifactRecord       `json:"diff,omitempty"`
	Startup      *releaseStartupRecord            `json:"startup,omitempty"`
	Validation   *releaseValidationRecord         `json:"validation,omitempty"`
	Resolution   map[string]string                `json:"resolution,omitempty"`
}

type seedConfig struct {
	rootPath    string
	commandPath string
	dbPath      string
	json        bool
}

type seedCredentialRecord struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
}

type seedSummary struct {
	OK           bool                   `json:"ok"`
	ProjectRoot  string                 `json:"projectRoot"`
	CommandPath  string                 `json:"commandPath"`
	DatabasePath string                 `json:"databasePath,omitempty"`
	Credentials  []seedCredentialRecord `json:"credentials,omitempty"`
	Output       string                 `json:"output,omitempty"`
}

type releaseOptimizerRecord struct {
	Mode string   `json:"mode"`
	Tool string   `json:"tool"`
	Args []string `json:"args,omitempty"`
}

type releaseAttributionRecord struct {
	Mode         string `json:"mode"`
	Path         string `json:"path"`
	PackageCount int    `json:"packageCount"`
}

type releasePackageSizeRecord struct {
	ImportPath   string `json:"importPath"`
	Dir          string `json:"dir,omitempty"`
	ArchiveBytes int64  `json:"archiveBytes,omitempty"`
	SourceBytes  int64  `json:"sourceBytes,omitempty"`
	FileCount    int    `json:"fileCount,omitempty"`
}

type releaseDiffArtifactRecord struct {
	Path                 string                      `json:"path"`
	BaselineManifestPath string                      `json:"baselineManifestPath"`
	ArtifactChanges      []releaseArtifactDiffRecord `json:"artifactChanges,omitempty"`
	LikelyCulprits       []releasePackageDiffRecord  `json:"likelyCulprits,omitempty"`
}

type releaseArtifactDiffRecord struct {
	Name          string   `json:"name"`
	BaselinePath  string   `json:"baselinePath,omitempty"`
	CurrentPath   string   `json:"currentPath,omitempty"`
	BaselineBytes *int64   `json:"baselineBytes,omitempty"`
	CurrentBytes  *int64   `json:"currentBytes,omitempty"`
	DeltaBytes    *int64   `json:"deltaBytes,omitempty"`
	DeltaPercent  *float64 `json:"deltaPercent,omitempty"`
	Status        string   `json:"status"`
}

type releasePackageDiffRecord struct {
	ImportPath           string   `json:"importPath"`
	BaselineArchiveBytes *int64   `json:"baselineArchiveBytes,omitempty"`
	CurrentArchiveBytes  *int64   `json:"currentArchiveBytes,omitempty"`
	ArchiveDeltaBytes    *int64   `json:"archiveDeltaBytes,omitempty"`
	ArchiveDeltaPercent  *float64 `json:"archiveDeltaPercent,omitempty"`
	BaselineSourceBytes  *int64   `json:"baselineSourceBytes,omitempty"`
	CurrentSourceBytes   *int64   `json:"currentSourceBytes,omitempty"`
	SourceDeltaBytes     *int64   `json:"sourceDeltaBytes,omitempty"`
	Status               string   `json:"status"`
}

type releaseStartupRecord struct {
	Mode              string `json:"mode"`
	Path              string `json:"path"`
	ProbeURL          string `json:"probeURL"`
	TransportEncoding string `json:"transportEncoding,omitempty"`
}

type releaseValidationRecord struct {
	Path                string   `json:"path"`
	Checks              []string `json:"checks,omitempty"`
	StartupReportPath   string   `json:"startupReportPath,omitempty"`
	WasmContentType     string   `json:"wasmContentType,omitempty"`
	WasmContentEncoding string   `json:"wasmContentEncoding,omitempty"`
}

type releaseGoListPackage struct {
	ImportPath string   `json:"ImportPath"`
	Dir        string   `json:"Dir"`
	Export     string   `json:"Export"`
	GoFiles    []string `json:"GoFiles"`
	CgoFiles   []string `json:"CgoFiles"`
	CFiles     []string `json:"CFiles"`
	CXXFiles   []string `json:"CXXFiles"`
	MFiles     []string `json:"MFiles"`
	HFiles     []string `json:"HFiles"`
	FFiles     []string `json:"FFiles"`
	SFiles     []string `json:"SFiles"`
	SysoFiles  []string `json:"SysoFiles"`
	EmbedFiles []string `json:"EmbedFiles"`
}

type verifyTestSummary struct {
	Ran            bool   `json:"ran"`
	Skipped        bool   `json:"skipped"`
	Command        string `json:"command,omitempty"`
	PackagePattern string `json:"packagePattern,omitempty"`
	Output         string `json:"output,omitempty"`
}

type verifySummary struct {
	OK               bool               `json:"ok"`
	AppPath          string             `json:"appPath"`
	ProjectRoot      string             `json:"projectRoot"`
	Tests            verifyTestSummary  `json:"tests"`
	Build            buildSummary       `json:"build"`
	Audit            *doctorAuditReport `json:"audit,omitempty"`
	AuditMinSeverity string             `json:"auditMinSeverity,omitempty"`
	Resolution       map[string]string  `json:"resolution,omitempty"`
}

type testConfig struct {
	appPath    string
	rootPath   string
	lanes      []string
	json       bool
	resolution map[string]string
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
	Resolution    map[string]string `json:"resolution,omitempty"`
}

var verifyExecuteBuild = executeBuild

var releaseExecuteBuild = executeBuild

var releaseArtifactRecordForPathFunc = releaseArtifactRecordForPath

var releaseWriteGzipSidecar = writeGzipSidecar

var releaseWriteBrotliSidecar = writeBrotliSidecar

var releaseMarshalIndent = json.MarshalIndent

var releaseLookPath = exec.LookPath

var releaseRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
	return launcherRunCommand(command, args, cwd, env)
}

var releasePlaywrightInstall = func(options *playwright.RunOptions) error {
	return playwright.Install(options)
}

var releasePlaywrightRun = func(options *playwright.RunOptions) (*playwright.Playwright, error) {
	return playwright.Run(options)
}

var releaseMeasureStartup = measureReleaseStartup

var releaseResolveWasmExec = resolveWasmExecPath

var releaseValidateSmoke = validateReleaseSmoke

var doctorResolveWasmExec = resolveWasmExecPath

var buildGetwd = os.Getwd

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

var runDashboardCommand = func(l launcher, args []string) error {
	return l.runDashboard(args)
}

var runDoctorCommand = func(l launcher, args []string) error {
	return l.runDoctor(args)
}

var runVerifyCommand = func(l launcher, args []string) error {
	return l.runVerify(args)
}

var runSeedCommand = func(l launcher, args []string) error {
	return l.runSeed(args)
}

var runImportCommand = func(l launcher, args []string) error {
	return l.runImport(args)
}

var runStartCommand = func(l launcher, args []string) error {
	return l.runStart(args)
}

var runBootstrapCommand = func(l launcher, args []string) error {
	return l.runBootstrap(args)
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

var seedGetwd = os.Getwd

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

type launcherFailureDiagnostic struct {
	OK       bool   `json:"ok"`
	Command  string `json:"command,omitempty"`
	Phase    string `json:"phase"`
	Category string `json:"category"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	Override string `json:"override,omitempty"`
}

func launcherCommandAndArgs(args []string) (string, []string) {
	_, remaining, err := parseLauncherGlobalCLIOptions(args)
	if err != nil {
		remaining = args
	}
	if len(remaining) == 0 {
		return "", nil
	}
	return strings.TrimSpace(remaining[0]), remaining[1:]
}

func launcherCommandSupportsJSON(command string) bool {
	switch strings.TrimSpace(strings.ToLower(command)) {
	case "bench", "benchmark", "build", "dev", "doctor", "env", "files", "release", "seed", "tailwind", "test", "verify", "wasm":
		return true
	default:
		return false
	}
}

func launcherJSONRequestedForCommand(command string, args []string) bool {
	if !launcherCommandSupportsJSON(command) {
		return false
	}
	for _, arg := range args {
		trimmed := strings.TrimSpace(strings.ToLower(arg))
		if trimmed == "-json" || trimmed == "--json" || strings.HasPrefix(trimmed, "-json=") || strings.HasPrefix(trimmed, "--json=") {
			return true
		}
	}
	return false
}

func launcherRequestedJSONOutput(args []string) bool {
	command, commandArgs := launcherCommandAndArgs(args)
	return launcherJSONRequestedForCommand(command, commandArgs)
}

func buildLauncherFailureDiagnostic(args []string, err error) launcherFailureDiagnostic {
	command, _ := launcherCommandAndArgs(args)
	message := strings.TrimSpace(err.Error())
	phase := "execution"
	category := "execution"
	code := "command_failed"
	override := ""
	lower := strings.ToLower(message)

	switch {
	case detectRunnerOverrideField(message) != "":
		phase = "configuration"
		category = "configuration"
		code = "invalid_runner_override"
		override = detectRunnerOverrideField(message)
	case strings.Contains(lower, "policy") || strings.Contains(lower, "not approved") || strings.Contains(lower, "blocked by security boundary"):
		phase = "policy"
		category = "policy"
		code = "policy_violation"
	case strings.Contains(lower, "resolve ") ||
		strings.Contains(lower, "parse ") ||
		strings.Contains(lower, "unknown ") ||
		strings.Contains(lower, "required") ||
		strings.Contains(lower, "configured ") ||
		strings.Contains(lower, "does not exist") ||
		strings.Contains(lower, "interactive terminal") ||
		strings.Contains(lower, "use either -compression or -skip-compression"):
		phase = "configuration"
		category = "configuration"
		code = "invalid_configuration"
	case strings.Contains(lower, "go test failed") || strings.Contains(lower, "go build failed"):
		phase = "execution"
		category = "code"
		code = "code_failure"
	case strings.Contains(lower, "verify audit found"):
		phase = "validation"
		category = "validation"
		code = "audit_failed"
	case strings.Contains(lower, "verify checks reported failures") || strings.Contains(lower, "doctor found required checks"):
		phase = "validation"
		category = "validation"
		code = "checks_failed"
	case strings.Contains(lower, "smoke validation"):
		phase = "validation"
		category = "validation"
		code = "smoke_failed"
	case strings.TrimSpace(strings.ToLower(command)) == "dev" && (strings.Contains(lower, "listen") || strings.Contains(lower, "bind ") || strings.Contains(lower, "livereload") || strings.Contains(lower, "serve")):
		phase = "runtime"
		category = "runtime"
		code = "startup_failed"
	}

	return launcherFailureDiagnostic{
		OK:       false,
		Command:  command,
		Phase:    phase,
		Category: category,
		Code:     code,
		Message:  message,
		Override: override,
	}
}

func detectRunnerOverrideField(message string) string {
	lower := strings.ToLower(strings.TrimSpace(message))
	for _, field := range []string{
		"artifactRoot",
		"browserWorkspace",
		"generatedProjectRoot",
		"goWasmExec",
		"livereloadClientScript",
		"livereloadWorkspace",
		"wasmExecJS",
		"workspaceBuildRoot",
	} {
		fieldLower := strings.ToLower(field)
		if strings.Contains(lower, "configured "+strings.ToLower(field)) ||
			strings.Contains(lower, "resolve "+fieldLower+" override") {
			return field
		}
	}
	return ""
}

func printLauncherError(w io.Writer, args []string, err error) {
	if w == nil {
		return
	}
	if launcherRequestedJSONOutput(args) {
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(buildLauncherFailureDiagnostic(args, err)); encodeErr == nil {
			return
		}
	}
	fmt.Fprintf(w, "gwc: %v\n", err)
}

var mainPrintError = func(err error) {
	printLauncherError(os.Stderr, mainArgs()[1:], err)
}

func main() {
	repoRoot, err := mainResolveRepoRoot()
	if err != nil {
		mainPrintError(err)
		mainExit(1)
	}
	examplesWasmDir, err := resolveLauncherWorkspaceBuildPath(repoRoot, "examples")
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
	globalOptions, remainingArgs, err := parseLauncherGlobalCLIOptions(args)
	if err != nil {
		return err
	}

	if len(remainingArgs) == 0 {
		printUsage()
		return nil
	}

	command := remainingArgs[0]
	commandArgs := remainingArgs[1:]

	switch command {
	case "help", "-h", "--help":
		printUsage()
		return nil
	}

	cwd, cwdErr := launcherConfigGetwd()
	if cwdErr != nil {
		cwd = ""
	}
	enterprise, err := resolveLauncherEnterpriseConfig(cwd, globalOptions)
	if err != nil {
		return err
	}
	launcherActiveEnterpriseConfig = enterprise.Effective
	launcherActiveEnterpriseSources = enterprise.Sources
	defer func() {
		launcherActiveEnterpriseConfig = defaultLauncherEnterpriseConfig()
		launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}
	}()
	if err := validateLauncherExtensionSecurity(launcherActiveEnterpriseConfig); err != nil {
		return err
	}
	jsonOutputRequested := launcherJSONRequestedForCommand(command, commandArgs)
	if !jsonOutputRequested {
		launcherPrintExtensionReport(command, enterprise)
	}

	if err := runLauncherCommandHooks(launcherActiveEnterpriseConfig.Hooks, "pre", command, commandArgs, l.repoRoot, launcherActiveEnterpriseSources); err != nil {
		return err
	}

	dispatchErr := l.dispatchCommand(command, commandArgs)
	postHookErr := runLauncherCommandHooks(launcherActiveEnterpriseConfig.Hooks, "post", command, commandArgs, l.repoRoot, launcherActiveEnterpriseSources)
	if dispatchErr != nil {
		return dispatchErr
	}
	if postHookErr != nil {
		return postHookErr
	}
	return nil
}

func (l launcher) dispatchCommand(command string, args []string) error {
	switch command {
	case "test":
		return runTestCommand(l, args)
	case "examples":
		return runExamplesCommand(l, args)
	case "build":
		return runBuildCommand(l, args)
	case "bench", "benchmark":
		return runBenchmarkCommand(l, args)
	case "release":
		return runReleaseCommand(l, args)
	case "dev":
		return runDevCommand(l, args)
	case "serve":
		return runServeCommand(l, args)
	case "files":
		return runFilesCommand(l, args)
	case "tailwind":
		return runTailwindCommand(l, args)
	case "dashboard":
		return runDashboardCommand(l, args)
	case "doctor":
		return runDoctorCommand(l, args)
	case "env":
		return runEnvCommand(l, args)
	case "verify":
		return runVerifyCommand(l, args)
	case "seed":
		return runSeedCommand(l, args)
	case "import":
		return runImportCommand(l, args)
	case "start":
		return runStartCommand(l, args)
	case "bootstrap":
		return runBootstrapCommand(l, args)
	case "wasm":
		return runWasmCommand(l, args)
	default:
		return fmt.Errorf("unknown command %q", command)
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
	if err := enforceEnterpriseGoToolchainPolicy(config.rootPath, launcherActiveEnterpriseConfig.Policy); err != nil {
		return err
	}
	if err := enforceEnterpriseRequiredTestLanes(config.lanes, launcherActiveEnterpriseConfig.Policy); err != nil {
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
	resolved.resolution = cloneResolutionTrace(config.resolution)
	cwd, err := testGetwd()
	if err != nil {
		return testConfig{}, err
	}
	if strings.TrimSpace(resolved.rootPath) == "" {
		resolved.rootPath = cwd
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "convention fallback")
	} else {
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "explicit flag")
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return testConfig{}, fmt.Errorf("resolve test root path: %w", err)
	}
	if strings.TrimSpace(resolved.appPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "explicit flag")
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
		Resolution:    cloneResolutionTrace(config.resolution),
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
			Summary:   "No browser test workspace was found for the requested root.",
		}, nil
	}
	packagePattern, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(workspace)
	if !hasPlaywrightGoSuite {
		return testLaneSummary{
			Name:      "browser",
			OK:        true,
			Skipped:   true,
			Workspace: workspace,
			Summary:   "No Playwright-Go test package was found in the browser workspace.",
		}, nil
	}
	args := []string{"test", "-tags", "playwrightgo", packagePattern, "-run", "TestMainSuite", "-v"}
	output, err := launcherRunCommand("go", args, workspace, buildBrowserTestEnv())
	if err != nil {
		return testLaneSummary{}, err
	}
	return testLaneSummary{
		Name:           "browser",
		OK:             true,
		Command:        "go " + strings.Join(args, " "),
		PackagePattern: packagePattern,
		Workspace:      workspace,
		Output:         output,
		Summary:        "Browser Playwright-Go suite passed.",
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
	if err := enforceEnterpriseReleasePolicy(releaseConfig, launcherActiveEnterpriseConfig.Policy); err != nil {
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

func enforceEnterpriseRequiredTestLanes(selectedLanes []string, policy launcherEnterprisePolicy) error {
	if len(policy.RequiredTestLanes) == 0 {
		return nil
	}
	selected := map[string]struct{}{}
	for _, lane := range selectedLanes {
		trimmed := strings.TrimSpace(strings.ToLower(lane))
		if trimmed == "" {
			continue
		}
		selected[trimmed] = struct{}{}
	}
	missing := []string{}
	for _, lane := range policy.RequiredTestLanes {
		normalized := strings.TrimSpace(strings.ToLower(lane))
		if normalized == "" {
			continue
		}
		if _, ok := selected[normalized]; !ok {
			missing = append(missing, lane)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required test lanes: %s", strings.Join(missing, ", "))
	}
	return nil
}

func enforceEnterpriseGoToolchainPolicy(rootPath string, policy launcherEnterprisePolicy) error {
	if len(policy.ApprovedGoToolchains) == 0 {
		return nil
	}
	activeToolchain, err := resolveActiveGoToolchain(rootPath)
	if err != nil {
		return err
	}
	for _, approved := range policy.ApprovedGoToolchains {
		if goToolchainApprovedByPolicy(activeToolchain, approved) {
			return nil
		}
	}
	return fmt.Errorf(
		"active Go toolchain %q is not approved; allowed values: %s",
		activeToolchain,
		strings.Join(policy.ApprovedGoToolchains, ", "),
	)
}

func resolveActiveGoToolchain(rootPath string) (string, error) {
	cwd := strings.TrimSpace(rootPath)
	if cwd == "" {
		cwd = "."
	}
	output, err := launcherRunCommand("go", []string{"env", "GOVERSION"}, cwd, buildNativeGoEnv())
	if err != nil {
		return "", fmt.Errorf("resolve active Go toolchain: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	version := strings.ToLower(strings.TrimSpace(lines[0]))
	if version == "" {
		return "", errors.New("resolve active Go toolchain: go env GOVERSION returned empty output")
	}
	return version, nil
}

func goToolchainApprovedByPolicy(activeToolchain string, approvedValue string) bool {
	active := normalizeGoToolchainPolicyValue(activeToolchain)
	approved := normalizeGoToolchainPolicyValue(approvedValue)
	if active == "" || approved == "" {
		return false
	}
	if active == approved {
		return true
	}
	if strings.HasSuffix(approved, ".x") {
		prefix := strings.TrimSuffix(approved, ".x")
		if prefix == "" {
			return false
		}
		if active == prefix {
			return true
		}
		return strings.HasPrefix(active, prefix+".")
	}
	return strings.HasPrefix(active, approved+".")
}

func normalizeGoToolchainPolicyValue(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "go") {
		return value
	}
	return "go" + value
}

func enforceEnterpriseReleasePolicy(config releaseConfig, policy launcherEnterprisePolicy) error {
	if policy.RequireReleaseBudgets != nil && *policy.RequireReleaseBudgets && strings.TrimSpace(config.budgetsPath) == "" {
		return errors.New("enterprise policy requires release budgets; provide -budgets or configure releaseBudgetsPath")
	}
	if requiredCompression := strings.TrimSpace(policy.RequiredReleaseCompression); requiredCompression != "" {
		normalizedRequired, err := normalizeReleaseCompressionPolicy(requiredCompression)
		if err != nil {
			return fmt.Errorf("normalize enterprise required release compression: %w", err)
		}
		if config.compression != normalizedRequired {
			return fmt.Errorf("release compression policy %q does not satisfy enterprise requirement %q", config.compression, normalizedRequired)
		}
	}
	if pattern := strings.TrimSpace(policy.ReleaseBinaryPattern); pattern != "" {
		matched, err := regexp.MatchString(pattern, config.binaryName)
		if err != nil {
			return fmt.Errorf("compile enterprise release binary pattern: %w", err)
		}
		if !matched {
			return fmt.Errorf("release binary name %q does not satisfy enterprise pattern %q", config.binaryName, pattern)
		}
	}
	if pattern := strings.TrimSpace(policy.ReleaseManifestPattern); pattern != "" {
		matched, err := regexp.MatchString(pattern, config.manifestName)
		if err != nil {
			return fmt.Errorf("compile enterprise release manifest pattern: %w", err)
		}
		if !matched {
			return fmt.Errorf("release manifest name %q does not satisfy enterprise pattern %q", config.manifestName, pattern)
		}
	}
	return nil
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

func resolveBrowserTestPackagePattern(workspace string) (string, bool) {
	if strings.TrimSpace(workspace) == "" {
		return "", false
	}
	candidates := []struct {
		path    string
		pattern string
	}{
		{path: filepath.Join(workspace, "playwrightgo"), pattern: "./playwrightgo"},
		{path: filepath.Join(workspace, "test", "playwrightgo"), pattern: "./test/playwrightgo"},
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate.path)
		if err != nil || !info.IsDir() {
			continue
		}
		return candidate.pattern, true
	}
	return "", false
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
		info, statErr := os.Stat(overridePath)
		if statErr != nil || !info.IsDir() {
			return "", fmt.Errorf("configured browserWorkspace path does not exist: %s", overridePath)
		}
		if _, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(overridePath); !hasPlaywrightGoSuite {
			return "", fmt.Errorf("configured browserWorkspace does not contain a Playwright-Go suite: %s", overridePath)
		}
		return overridePath, nil
	}
	if _, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(rootPath); hasPlaywrightGoSuite {
		return rootPath, nil
	}
	repoWorkspace := filepath.Join(repoRoot, "test")
	if _, hasPlaywrightGoSuite := resolveBrowserTestPackagePattern(repoWorkspace); hasPlaywrightGoSuite {
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
	return "", false, nil
}

func (l launcher) runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	app := fs.String("app", "", "Path to the app main.go file or app directory")
	mainPath := fs.String("main", "", "Legacy alias for -app")
	root := fs.String("root", "", "Project root used for test and build resolution")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	skipTests := fs.Bool("skip-tests", false, "Skip running go test even when *_test.go files are present")
	audit := fs.Bool("audit", false, "Run the golden-path app audit as part of verify")
	auditPolicy := fs.String("audit-policy", "strict", "Golden-path audit policy: strict or advisory")
	auditBaseline := fs.String("audit-baseline", "", "Optional path to a JSON baseline file of accepted audit findings")
	auditWriteBaseline := fs.String("audit-write-baseline", "", "Optional path to write the current audit findings as a JSON baseline")
	auditMinSeverity := fs.String("audit-min-severity", "error", "Minimum golden-path audit severity that causes verify to fail: off, error, warning, or info")
	var auditSuppressions stringListFlag
	fs.Var(&auditSuppressions, "audit-suppress", "Audit check name to suppress; repeat or comma-separate")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	pluginResults, err := runLauncherPluginsForCapability("verify_check", "verify", args, l.repoRoot, launcherActiveEnterpriseSources)
	if err != nil {
		return err
	}
	if err := enforcePluginChecks(pluginResults, "verify_check"); err != nil {
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
	if err := enforceEnterpriseGoToolchainPolicy(buildConfig.rootPath, launcherActiveEnterpriseConfig.Policy); err != nil {
		return err
	}
	verifyCoveredLanes := []string{}
	if !*skipTests {
		verifyCoveredLanes = append(verifyCoveredLanes, "unit")
	}
	if err := enforceEnterpriseRequiredTestLanes(verifyCoveredLanes, launcherActiveEnterpriseConfig.Policy); err != nil {
		return fmt.Errorf("verify does not satisfy enterprise lane policy: %w", err)
	}

	summary := verifySummary{
		AppPath:     buildConfig.appPath,
		ProjectRoot: buildConfig.rootPath,
		Resolution:  cloneResolutionTrace(buildConfig.resolution),
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

	buildSummary, err := verifyExecuteBuild(buildConfig)
	if err != nil {
		return err
	}
	summary.Build = buildSummary
	summary.OK = true
	var verifyErr error
	if *audit {
		minSeverity, ok := normalizeDoctorAuditMinimumSeverity(*auditMinSeverity)
		if !ok {
			return fmt.Errorf("unknown audit minimum severity %q", *auditMinSeverity)
		}
		summary.Audit = buildDoctorAuditReport(buildConfig.rootPath, doctorConfig{
			audit:              true,
			auditPolicy:        *auditPolicy,
			auditBaselinePath:  *auditBaseline,
			auditWriteBaseline: *auditWriteBaseline,
			auditSuppressions:  auditSuppressions.Values(),
			json:               *jsonOutput,
		})
		summary.AuditMinSeverity = minSeverity
		if strings.TrimSpace(*auditWriteBaseline) != "" && summary.Audit != nil {
			if err := writeDoctorAuditBaseline(*auditWriteBaseline, *summary.Audit); err != nil {
				return err
			}
		}
		if summary.Audit != nil && doctorAuditHasFindingAtOrAbove(summary.Audit.Checks, minSeverity) {
			summary.OK = false
			verifyErr = fmt.Errorf("verify audit found %s-severity findings that need attention", minSeverity)
		}
	}

	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(summary); err != nil {
			return err
		}
	} else {
		printVerifySummary(summary)
	}
	if !summary.OK {
		if verifyErr != nil {
			return verifyErr
		}
		return errors.New("verify checks reported failures")
	}
	return nil
}

func (l launcher) runSeed(args []string) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	root := fs.String("root", "", "Project root used for seed command discovery")
	commandPath := fs.String("command", "", "Path to the seed command package directory or main.go file")
	dbPath := fs.String("db-path", "", "Override CHAT_DB_PATH for known seeders")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveSeedConfig(seedConfig{
		rootPath:    *root,
		commandPath: *commandPath,
		dbPath:      *dbPath,
		json:        *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := executeSeed(config)
	if err != nil {
		return err
	}
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(summary)
	}
	printSeedSummary(summary)
	return nil
}

func resolveSeedConfig(config seedConfig) (seedConfig, error) {
	resolved := config
	cwd, err := seedGetwd()
	if err != nil {
		return seedConfig{}, err
	}
	if strings.TrimSpace(resolved.rootPath) == "" {
		resolved.rootPath = cwd
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return seedConfig{}, fmt.Errorf("resolve seed root path: %w", err)
	}
	if strings.TrimSpace(resolved.commandPath) != "" {
		resolved.commandPath, err = normalizeExistingPath(cwd, resolved.commandPath)
		if err != nil {
			return seedConfig{}, fmt.Errorf("resolve seed command path: %w", err)
		}
	} else {
		resolved.commandPath, err = detectSeedCommandPath(resolved.rootPath)
		if err != nil {
			return seedConfig{}, err
		}
	}
	commandDir, err := normalizeSeedCommandDir(resolved.commandPath)
	if err != nil {
		return seedConfig{}, err
	}
	resolved.commandPath = commandDir
	if strings.TrimSpace(resolved.dbPath) == "" {
		resolved.dbPath = defaultSeedDatabasePath(commandDir)
	} else {
		resolved.dbPath, err = normalizePath(cwd, resolved.dbPath)
		if err != nil {
			return seedConfig{}, fmt.Errorf("resolve seed database path: %w", err)
		}
	}
	return resolved, nil
}

func detectSeedCommandPath(rootPath string) (string, error) {
	candidates := []string{
		filepath.Join(rootPath, "cmd", "seed"),
		filepath.Join(rootPath, "cmd", "seed-test-db"),
		filepath.Join(rootPath, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db"),
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("no seed command found under %s; pass -command to select a seed package", rootPath)
}

func normalizeSeedCommandDir(commandPath string) (string, error) {
	info, err := os.Stat(commandPath)
	if err != nil {
		return "", fmt.Errorf("inspect seed command path: %w", err)
	}
	if info.IsDir() {
		return commandPath, nil
	}
	return filepath.Dir(commandPath), nil
}

func defaultSeedDatabasePath(commandDir string) string {
	if !isChatWizardSeedCommand(commandDir) {
		return ""
	}
	exampleRoot := filepath.Dir(filepath.Dir(commandDir))
	return filepath.Join(exampleRoot, "bin", "runtime", "test_chat.db")
}

func isChatWizardSeedCommand(commandDir string) bool {
	commandDir = filepath.Clean(commandDir)
	suffix := filepath.Join("examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	return strings.HasSuffix(commandDir, suffix)
}

func executeSeed(config seedConfig) (seedSummary, error) {
	env := buildNativeGoEnv()
	if strings.TrimSpace(config.dbPath) != "" {
		env = replaceEnvVar(env, "CHAT_DB_PATH", config.dbPath)
	}
	output, err := launcherRunCommand("go", []string{"run", "."}, config.commandPath, env)
	if err != nil {
		return seedSummary{}, err
	}
	summary := seedSummary{
		OK:           true,
		ProjectRoot:  config.rootPath,
		CommandPath:  config.commandPath,
		DatabasePath: config.dbPath,
		Output:       output,
	}
	if isChatWizardSeedCommand(config.commandPath) {
		summary.Credentials = []seedCredentialRecord{
			{Email: "demo@example.com", Password: "password123", Role: "demo"},
			{Email: "admin@example.com", Password: "password", Role: "admin"},
		}
	}
	return summary, nil
}

func replaceEnvVar(env []string, key string, value string) []string {
	prefix := key + "="
	replaced := false
	updated := make([]string, 0, len(env)+1)
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			if !replaced {
				updated = append(updated, prefix+value)
				replaced = true
			}
			continue
		}
		updated = append(updated, entry)
	}
	if !replaced {
		updated = append(updated, prefix+value)
	}
	return updated
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
	compareManifest := fs.String("compare-manifest", "", "Optional baseline release manifest to diff against")
	profile := fs.String("profile", "release", "Release build profile")
	compression := fs.String("compression", "", "Compression sidecars: none, gzip, brotli, or gzip+brotli")
	postLinkOpt := fs.String("post-link-opt", "", "Optional post-link optimization: none or wasm-opt")
	sizeAttribution := fs.String("size-attribution", "", "Optional size attribution: none or packages")
	startupMeasure := fs.String("startup-measure", "", "Optional startup measurement: none or browser")
	startupTimeoutMs := fs.Int("startup-timeout-ms", 30000, "Startup measurement timeout in milliseconds")
	validateSmoke := fs.Bool("validate-smoke", false, "Run post-build release smoke validation")
	skipCompression := fs.Bool("skip-compression", false, "Skip gzip sidecar generation")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	pluginResults, err := runLauncherPluginsForCapability("release_validator", "release", args, l.repoRoot, launcherActiveEnterpriseSources)
	if err != nil {
		return err
	}
	if err := enforcePluginChecks(pluginResults, "release_validator"); err != nil {
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
		appPath:          firstNonEmpty(*app, *mainPath),
		rootPath:         *root,
		outDir:           *outDir,
		binaryName:       *binaryName,
		manifestName:     *manifestName,
		budgetsPath:      *budgetsPath,
		compareManifest:  *compareManifest,
		profile:          *profile,
		compression:      compressionPolicy,
		postLinkOpt:      *postLinkOpt,
		sizeAttribution:  *sizeAttribution,
		startupMeasure:   *startupMeasure,
		startupTimeoutMs: *startupTimeoutMs,
		validateSmoke:    *validateSmoke,
		compressionSet:   compressionSet || skipCompressionSet,
		skipCompression:  *skipCompression,
		skipCompressSet:  skipCompressionSet,
		json:             *jsonOutput,
	})
	if err != nil {
		return err
	}
	if err := enforceEnterpriseGoToolchainPolicy(config.rootPath, launcherActiveEnterpriseConfig.Policy); err != nil {
		return err
	}
	if err := enforceEnterpriseReleasePolicy(config, launcherActiveEnterpriseConfig.Policy); err != nil {
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

func resolveBuildConfig(config buildConfig) (buildConfig, error) {
	resolved := config
	resolved.resolution = cloneResolutionTrace(config.resolution)
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
			resolved.resolution = setResolutionSource(resolved.resolution, "app", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.rootPath) == "" {
			resolved.rootPath = metadataDir
			resolved.resolution = setResolutionSource(resolved.resolution, "root", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.outputPath) == "" && strings.TrimSpace(metadata.Tooling.WASMPath) != "" {
			resolved.outputPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.WASMPath))
			resolved.resolution = setResolutionSource(resolved.resolution, "output", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.profile) == "" && strings.TrimSpace(metadata.Tooling.DefaultBuildProfile) != "" {
			resolved.profile = strings.TrimSpace(metadata.Tooling.DefaultBuildProfile)
			resolved.resolution = setResolutionSource(resolved.resolution, "profile", "gwc-start.json")
		}
	}
	if strings.TrimSpace(config.appPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(config.rootPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "explicit flag")
	}
	if explicitOutputPath {
		resolved.resolution = setResolutionSource(resolved.resolution, "output", "explicit flag")
	}
	if strings.TrimSpace(config.profile) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "profile", "explicit flag")
	}

	if strings.TrimSpace(resolved.appPath) == "" {
		resolved.appPath, err = detectAppPath(cwd)
		if err != nil {
			return buildConfig{}, err
		}
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "convention fallback")
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
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "convention fallback")
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return buildConfig{}, fmt.Errorf("resolve root path: %w", err)
	}

	if strings.TrimSpace(resolved.outputPath) == "" {
		defaultOutputPath, source, err := resolveLauncherDefaultBuildOutput(resolved.rootPath)
		if err != nil {
			return buildConfig{}, err
		}
		resolved.outputPath = defaultOutputPath
		resolved.resolution = setResolutionSource(resolved.resolution, "output", source)
	}
	resolved.outputPath, err = normalizePath(cwd, resolved.outputPath)
	if err != nil {
		return buildConfig{}, fmt.Errorf("resolve output path: %w", err)
	}

	profile, err := resolveBuildProfile(strings.TrimSpace(resolved.profile))
	if err != nil {
		return buildConfig{}, err
	}
	if strings.TrimSpace(resolved.profile) == "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "profile", "convention fallback")
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
	resolved.resolution = cloneResolutionTrace(config.resolution)
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
			resolved.resolution = setResolutionSource(resolved.resolution, "app", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.rootPath) == "" {
			resolved.rootPath = metadataDir
			resolved.resolution = setResolutionSource(resolved.resolution, "root", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.outDir) == "" && strings.TrimSpace(metadata.Tooling.ReleaseOutDir) != "" {
			resolved.outDir = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.ReleaseOutDir))
			resolved.resolution = setResolutionSource(resolved.resolution, "output", "gwc-start.json")
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
	if strings.TrimSpace(config.appPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(config.rootPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "explicit flag")
	}
	if explicitOutDir {
		resolved.resolution = setResolutionSource(resolved.resolution, "output", "explicit flag")
	}
	if strings.TrimSpace(config.profile) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "profile", "explicit flag")
	}

	if strings.TrimSpace(resolved.appPath) == "" {
		resolved.appPath, err = detectAppPath(cwd)
		if err != nil {
			return releaseConfig{}, err
		}
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "convention fallback")
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
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "convention fallback")
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return releaseConfig{}, fmt.Errorf("resolve root path: %w", err)
	}

	if strings.TrimSpace(resolved.outDir) == "" {
		defaultOutDir, source, err := resolveLauncherDefaultReleaseOutDir(resolved.rootPath)
		if err != nil {
			return releaseConfig{}, err
		}
		resolved.outDir = defaultOutDir
		resolved.resolution = setResolutionSource(resolved.resolution, "output", source)
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
	if strings.TrimSpace(resolved.compareManifest) != "" {
		resolved.compareManifest, err = normalizeExistingPath(cwd, resolved.compareManifest)
		if err != nil {
			return releaseConfig{}, fmt.Errorf("resolve compare manifest path: %w", err)
		}
	}
	compressionPolicy, err := normalizeReleaseCompressionPolicy(firstNonEmpty(resolved.compression, defaultScaffoldReleaseCompression()))
	if err != nil {
		return releaseConfig{}, err
	}
	resolved.compression = compressionPolicy
	resolved.skipCompression = compressionPolicy == "none"
	postLinkOpt, err := normalizeReleasePostLinkOptimization(firstNonEmpty(resolved.postLinkOpt, "none"))
	if err != nil {
		return releaseConfig{}, err
	}
	resolved.postLinkOpt = postLinkOpt
	sizeAttribution, err := normalizeReleaseSizeAttributionMode(firstNonEmpty(resolved.sizeAttribution, "none"))
	if err != nil {
		return releaseConfig{}, err
	}
	resolved.sizeAttribution = sizeAttribution
	startupMeasure, err := normalizeReleaseStartupMeasureMode(firstNonEmpty(resolved.startupMeasure, "none"))
	if err != nil {
		return releaseConfig{}, err
	}
	resolved.startupMeasure = startupMeasure
	if resolved.startupTimeoutMs <= 0 {
		resolved.startupTimeoutMs = 30000
	}

	profile, err := resolveBuildProfile(strings.TrimSpace(firstNonEmpty(resolved.profile, "release")))
	if err != nil {
		return releaseConfig{}, err
	}
	if strings.TrimSpace(resolved.profile) == "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "profile", "convention fallback")
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

func normalizeReleasePostLinkOptimization(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "wasm-opt", "size", "optimize":
		return "wasm-opt", nil
	default:
		return "", fmt.Errorf("unknown release post-link optimization %q", mode)
	}
}

func normalizeReleaseSizeAttributionMode(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "packages", "package", "per-package":
		return "packages", nil
	default:
		return "", fmt.Errorf("unknown release size attribution mode %q", mode)
	}
}

func normalizeReleaseStartupMeasureMode(mode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "browser", "playwright":
		return "browser", nil
	default:
		return "", fmt.Errorf("unknown release startup measurement mode %q", mode)
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
		Resolution:  cloneResolutionTrace(config.resolution),
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
	optimizer, err := releaseApplyPostLinkOptimization(config.postLinkOpt, wasmPath)
	if err != nil {
		return releaseSummary{}, err
	}
	artifacts["wasm"], err = releaseArtifactRecordForPathFunc(config.outDir, wasmPath)
	if err != nil {
		return releaseSummary{}, err
	}
	attribution, err := releaseWriteSizeAttribution(config.sizeAttribution, buildSummary.PackageDir, config.outDir)
	if err != nil {
		return releaseSummary{}, err
	}
	if attribution != nil {
		artifacts["size_attribution"], err = releaseArtifactRecordForPathFunc(config.outDir, filepath.Join(config.outDir, attribution.Path))
		if err != nil {
			return releaseSummary{}, err
		}
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
	startupConfig := config
	if startupConfig.validateSmoke && startupConfig.startupMeasure == "none" {
		startupConfig.startupMeasure = "browser"
	}
	startupReport, err := releaseMeasureStartup(startupConfig, artifacts)
	if err != nil {
		return releaseSummary{}, err
	}
	if startupReport != nil {
		artifacts["startup_report"], err = releaseArtifactRecordForPathFunc(config.outDir, filepath.Join(config.outDir, startupReport.Path))
		if err != nil {
			return releaseSummary{}, err
		}
	}
	manifestPath := filepath.Join(config.outDir, config.manifestName)
	diffReport, err := releaseWriteDiffReport(config.compareManifest, manifestPath, attribution, artifacts, config.outDir)
	if err != nil {
		return releaseSummary{}, err
	}
	if diffReport != nil {
		artifacts["diff_report"], err = releaseArtifactRecordForPathFunc(config.outDir, filepath.Join(config.outDir, diffReport.Path))
		if err != nil {
			return releaseSummary{}, err
		}
	}
	manifestPayload := map[string]interface{}{
		"package": buildSummary.PackageDir,
		"profile": buildSummary.Profile.Name,
		"goos":    "js",
		"goarch":  "wasm",
		"flags": map[string]interface{}{
			"trimpath":             buildSummary.Profile.Trimpath,
			"ldflags":              buildSummary.Profile.Ldflags,
			"buildvcs":             firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"compression":          !config.skipCompression,
			"compressionPolicy":    config.compression,
			"compareManifest":      config.compareManifest,
			"postLinkOptimization": config.postLinkOpt,
			"sizeAttribution":      config.sizeAttribution,
			"startupMeasure":       config.startupMeasure,
			"validateSmoke":        config.validateSmoke,
			"gzip":                 emitGzip,
			"brotli":               emitBrotli,
		},
		"artifacts": artifacts,
	}
	if optimizer != nil {
		manifestPayload["optimizer"] = optimizer
	}
	if attribution != nil {
		manifestPayload["attribution"] = attribution
	}
	if startupReport != nil {
		manifestPayload["startup"] = startupReport
	}
	if diffReport != nil {
		manifestPayload["diff"] = diffReport
	}
	encodedManifest, err := releaseMarshalIndent(manifestPayload, "", "  ")
	if err != nil {
		return releaseSummary{}, fmt.Errorf("encode release manifest: %w", err)
	}
	encodedManifest = append(encodedManifest, '\n')
	if err := os.WriteFile(manifestPath, encodedManifest, 0644); err != nil {
		return releaseSummary{}, fmt.Errorf("write release manifest: %w", err)
	}
	validation, err := releaseValidateSmoke(config, manifestPath, artifacts, startupReport)
	if err != nil {
		return releaseSummary{}, err
	}
	if validation != nil {
		artifacts["validation_report"], err = releaseArtifactRecordForPathFunc(config.outDir, filepath.Join(config.outDir, validation.Path))
		if err != nil {
			return releaseSummary{}, err
		}
		manifestPayload["validation"] = validation
		manifestPayload["artifacts"] = artifacts
		encodedManifest, err = releaseMarshalIndent(manifestPayload, "", "  ")
		if err != nil {
			return releaseSummary{}, fmt.Errorf("encode release manifest: %w", err)
		}
		encodedManifest = append(encodedManifest, '\n')
		if err := os.WriteFile(manifestPath, encodedManifest, 0644); err != nil {
			return releaseSummary{}, fmt.Errorf("write release manifest: %w", err)
		}
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
			"trimpath":             buildSummary.Profile.Trimpath,
			"ldflags":              buildSummary.Profile.Ldflags,
			"buildvcs":             firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"compression":          !config.skipCompression,
			"compressionPolicy":    config.compression,
			"compareManifest":      config.compareManifest,
			"postLinkOptimization": config.postLinkOpt,
			"sizeAttribution":      config.sizeAttribution,
			"startupMeasure":       config.startupMeasure,
			"validateSmoke":        config.validateSmoke,
			"gzip":                 emitGzip,
			"brotli":               emitBrotli,
		},
		Optimizer:   optimizer,
		Attribution: attribution,
		Diff:        diffReport,
		Startup:     startupReport,
		Validation:  validation,
		Resolution:  cloneResolutionTrace(config.resolution),
	}, nil
}

func releaseApplyPostLinkOptimization(mode string, wasmPath string) (*releaseOptimizerRecord, error) {
	normalizedMode, err := normalizeReleasePostLinkOptimization(mode)
	if err != nil {
		return nil, err
	}
	if normalizedMode == "none" {
		return nil, nil
	}
	if normalizedMode != "wasm-opt" {
		return nil, fmt.Errorf("unsupported release post-link optimization %q", normalizedMode)
	}
	command, err := resolveReleaseWasmOptCommand()
	if err != nil {
		return nil, err
	}
	if !command.Available {
		return nil, errors.New("post-link optimization requested but wasm-opt is unavailable; install wasm-opt and ensure it is on PATH")
	}

	optimizedPath := wasmPath + ".opt"
	if err := os.RemoveAll(optimizedPath); err != nil {
		return nil, fmt.Errorf("prepare optimized wasm artifact: %w", err)
	}
	args := append([]string{}, command.PrefixArgs...)
	args = append(args, wasmPath, "-Oz", "-o", optimizedPath)
	if _, err := releaseRunCommand(command.Command, args, filepath.Dir(wasmPath), os.Environ()); err != nil {
		return nil, fmt.Errorf("run post-link optimizer %q: %w", normalizedMode, err)
	}
	optimizedBytes, err := os.ReadFile(optimizedPath)
	if err != nil {
		return nil, fmt.Errorf("read optimized wasm artifact: %w", err)
	}
	if err := os.WriteFile(wasmPath, optimizedBytes, 0644); err != nil {
		return nil, fmt.Errorf("replace release wasm artifact with optimized output: %w", err)
	}
	if err := os.Remove(optimizedPath); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("cleanup optimized wasm artifact: %w", err)
	}
	return &releaseOptimizerRecord{
		Mode: normalizedMode,
		Tool: command.Label,
		Args: args,
	}, nil
}

type releaseCommandInfo struct {
	Available  bool
	Command    string
	PrefixArgs []string
	Label      string
}

func resolveReleaseWasmOptCommand() (releaseCommandInfo, error) {
	if path, err := releaseLookPath("wasm-opt"); err == nil {
		return releaseCommandInfo{
			Available: true,
			Command:   "wasm-opt",
			Label:     path,
		}, nil
	}
	return releaseCommandInfo{}, nil
}

func releaseWriteSizeAttribution(mode string, packageDir string, outDir string) (*releaseAttributionRecord, error) {
	normalizedMode, err := normalizeReleaseSizeAttributionMode(mode)
	if err != nil {
		return nil, err
	}
	if normalizedMode == "none" {
		return nil, nil
	}
	if normalizedMode != "packages" {
		return nil, fmt.Errorf("unsupported release size attribution mode %q", normalizedMode)
	}

	packages, err := releaseCollectPackageSizeAttribution(packageDir)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"mode":     normalizedMode,
		"package":  packageDir,
		"goos":     "js",
		"goarch":   "wasm",
		"packages": packages,
	}
	encoded, err := releaseMarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode release size attribution: %w", err)
	}
	encoded = append(encoded, '\n')
	fileName := "wasm-package-size-attribution.json"
	if err := os.WriteFile(filepath.Join(outDir, fileName), encoded, 0644); err != nil {
		return nil, fmt.Errorf("write release size attribution: %w", err)
	}
	return &releaseAttributionRecord{
		Mode:         normalizedMode,
		Path:         fileName,
		PackageCount: len(packages),
	}, nil
}

func releaseCollectPackageSizeAttribution(packageDir string) ([]releasePackageSizeRecord, error) {
	output, err := releaseRunCommand("go", []string{"list", "-deps", "-json", "-export", "."}, packageDir, buildWasmGoEnv())
	if err != nil {
		return nil, fmt.Errorf("collect release package attribution: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(output))
	packages := make([]releasePackageSizeRecord, 0, 64)
	for {
		var pkg releaseGoListPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode release package attribution: %w", err)
		}
		if strings.TrimSpace(pkg.ImportPath) == "" {
			continue
		}
		record, err := releaseBuildPackageSizeRecord(pkg)
		if err != nil {
			return nil, err
		}
		if record.ArchiveBytes == 0 && record.SourceBytes == 0 && record.FileCount == 0 {
			continue
		}
		packages = append(packages, record)
	}
	sort.Slice(packages, func(i int, j int) bool {
		if packages[i].ArchiveBytes != packages[j].ArchiveBytes {
			return packages[i].ArchiveBytes > packages[j].ArchiveBytes
		}
		if packages[i].SourceBytes != packages[j].SourceBytes {
			return packages[i].SourceBytes > packages[j].SourceBytes
		}
		return packages[i].ImportPath < packages[j].ImportPath
	})
	return packages, nil
}

func releaseBuildPackageSizeRecord(pkg releaseGoListPackage) (releasePackageSizeRecord, error) {
	record := releasePackageSizeRecord{
		ImportPath: strings.TrimSpace(pkg.ImportPath),
		Dir:        strings.TrimSpace(pkg.Dir),
	}
	if strings.TrimSpace(pkg.Export) != "" {
		info, err := os.Stat(pkg.Export)
		if err != nil {
			return releasePackageSizeRecord{}, fmt.Errorf("inspect export archive for %s: %w", pkg.ImportPath, err)
		}
		record.ArchiveBytes = info.Size()
	}
	sourceFiles := map[string]struct{}{}
	for _, file := range append(
		append(append(append(append(append(append(append(append(append([]string{}, pkg.GoFiles...), pkg.CgoFiles...), pkg.CFiles...), pkg.CXXFiles...), pkg.MFiles...), pkg.HFiles...), pkg.FFiles...), pkg.SFiles...), pkg.SysoFiles...),
		pkg.EmbedFiles...,
	) {
		trimmed := strings.TrimSpace(file)
		if trimmed == "" || strings.TrimSpace(pkg.Dir) == "" {
			continue
		}
		sourceFiles[filepath.Join(pkg.Dir, filepath.FromSlash(trimmed))] = struct{}{}
	}
	record.FileCount = len(sourceFiles)
	for file := range sourceFiles {
		info, err := os.Stat(file)
		if err != nil {
			return releasePackageSizeRecord{}, fmt.Errorf("inspect source file for %s: %w", pkg.ImportPath, err)
		}
		record.SourceBytes += info.Size()
	}
	return record, nil
}

func measureReleaseStartup(config releaseConfig, artifacts map[string]releaseArtifactRecord) (*releaseStartupRecord, error) {
	mode, err := normalizeReleaseStartupMeasureMode(config.startupMeasure)
	if err != nil {
		return nil, err
	}
	if mode == "none" {
		return nil, nil
	}
	wasmExecPath, err := releaseResolveWasmExec()
	if err != nil {
		return nil, fmt.Errorf("resolve wasm_exec.js for startup measurement: %w", err)
	}
	releaseWasmPath := filepath.Join(config.outDir, config.binaryName)
	probeURL, transportEncoding, shutdown, err := startReleaseStartupProbeServer(config.outDir, config.binaryName, wasmExecPath, artifacts, config.startupTimeoutMs)
	if err != nil {
		return nil, err
	}
	defer shutdown()

	reportPath := filepath.Join(config.outDir, "wasm-startup-report.json")
	if err := runReleaseStartupProbeWithPlaywright(probeURL, reportPath, config.startupTimeoutMs); err != nil {
		return nil, fmt.Errorf("measure release startup: %w", err)
	}
	if !fileExists(reportPath) {
		return nil, fmt.Errorf("startup measurement did not produce a report for %s", releaseWasmPath)
	}
	return &releaseStartupRecord{
		Mode:              mode,
		Path:              "wasm-startup-report.json",
		ProbeURL:          probeURL,
		TransportEncoding: transportEncoding,
	}, nil
}

func runReleaseStartupProbeWithPlaywright(probeURL string, reportPath string, timeoutMs int) error {
	runOptions := &playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}
	if err := releasePlaywrightInstall(runOptions); err != nil {
		return fmt.Errorf("install playwright-go runtime: %w", err)
	}
	pw, err := releasePlaywrightRun(runOptions)
	if err != nil {
		return fmt.Errorf("run playwright-go runtime: %w", err)
	}
	defer func() {
		_ = pw.Stop()
	}()
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("launch chromium: %w", err)
	}
	defer func() {
		_ = browser.Close()
	}()
	page, err := browser.NewPage()
	if err != nil {
		return fmt.Errorf("create probe page: %w", err)
	}
	if timeoutMs > 0 {
		page.SetDefaultTimeout(float64(timeoutMs))
	}
	response, err := page.Goto(probeURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return fmt.Errorf("open startup probe page: %w", err)
	}
	if response == nil {
		return errors.New("startup probe navigation returned no response")
	}
	if response.Status() >= 400 {
		return fmt.Errorf("startup probe navigation failed: %d", response.Status())
	}
	if _, err := page.WaitForFunction("() => window.__gwcStartupProbe && window.__gwcStartupProbe.readyMs !== null", nil); err != nil {
		return fmt.Errorf("wait for ready probe: %w", err)
	}
	if err := page.Click("#__gwc_probe_button"); err != nil {
		return fmt.Errorf("trigger startup probe interaction: %w", err)
	}
	if _, err := page.WaitForFunction("() => window.__gwcStartupProbe && window.__gwcStartupProbe.interactionMs !== null", nil); err != nil {
		return fmt.Errorf("wait for interaction probe: %w", err)
	}
	report, err := page.Evaluate(`() => ({
		userAgent: navigator.userAgent,
		startup: window.__gwcStartupProbe || null,
		timestamp: new Date().toISOString()
	})`)
	if err != nil {
		return fmt.Errorf("collect startup probe report: %w", err)
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("encode startup probe report: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(reportPath, encoded, 0644); err != nil {
		return fmt.Errorf("write startup probe report: %w", err)
	}
	return nil
}

func validateReleaseSmoke(config releaseConfig, manifestPath string, artifacts map[string]releaseArtifactRecord, startupReport *releaseStartupRecord) (*releaseValidationRecord, error) {
	if !config.validateSmoke {
		return nil, nil
	}
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read release manifest for smoke validation: %w", err)
	}
	manifest, err := pwa.ParseWasmReleaseManifestJSON(manifestBytes)
	if err != nil {
		return nil, fmt.Errorf("validate release manifest for smoke validation: %w", err)
	}
	checks := []string{"manifest parses as a valid js/wasm release record"}
	for name, artifact := range manifest.Artifacts {
		artifactPath := filepath.Join(config.outDir, filepath.FromSlash(artifact.Path))
		record, err := releaseArtifactRecordForPathFunc(config.outDir, artifactPath)
		if err != nil {
			return nil, fmt.Errorf("validate release artifact %q: %w", name, err)
		}
		if record.Bytes != artifact.Bytes || !strings.EqualFold(record.SHA256, artifact.SHA256) {
			return nil, fmt.Errorf("validate release artifact %q: manifest record does not match on-disk artifact", name)
		}
		checks = append(checks, fmt.Sprintf("artifact %s exists and matches manifest bytes and sha256", name))
	}
	if startupReport == nil {
		return nil, errors.New("release smoke validation requires a startup probe result")
	}
	wasmContentType, wasmContentEncoding, err := releaseSmokeFetchWasmHeaders(config.outDir, config.binaryName, artifacts)
	if err != nil {
		return nil, err
	}
	checks = append(checks, "wasm asset serves with application/wasm content type")
	checks = append(checks, "boot-time startup probe completed")
	record := &releaseValidationRecord{
		Path:                "wasm-release-validation.json",
		Checks:              checks,
		StartupReportPath:   startupReport.Path,
		WasmContentType:     wasmContentType,
		WasmContentEncoding: wasmContentEncoding,
	}
	encoded, err := releaseMarshalIndent(record, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode release validation report: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(config.outDir, record.Path), encoded, 0644); err != nil {
		return nil, fmt.Errorf("write release validation report: %w", err)
	}
	return record, nil
}

func releaseSmokeFetchWasmHeaders(outDir string, binaryName string, artifacts map[string]releaseArtifactRecord) (string, string, error) {
	listener, err := net.Listen("tcp", joinHostPort(defaultHost, "0"))
	if err != nil {
		return "", "", fmt.Errorf("release smoke validation listen: %w", err)
	}
	defer listener.Close()
	mux := http.NewServeMux()
	transportEncoding := releaseStartupTransportEncoding(artifacts)
	mux.HandleFunc("/"+binaryName, func(w http.ResponseWriter, r *http.Request) {
		releaseServeStartupWasm(w, r, outDir, binaryName, transportEncoding)
	})
	server := &http.Server{Handler: mux}
	defer server.Close()
	go func() {
		_ = server.Serve(listener)
	}()
	client := &http.Client{
		Transport: &http.Transport{DisableCompression: true},
		Timeout:   15 * time.Second,
	}
	resp, err := client.Get("http://" + listener.Addr().String() + "/" + strings.TrimLeft(binaryName, "/"))
	if err != nil {
		return "", "", fmt.Errorf("release smoke validation fetch wasm asset: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("release smoke validation expected wasm asset status 200, got %d", resp.StatusCode)
	}
	contentType := strings.TrimSpace(resp.Header.Get("Content-Type"))
	if !strings.Contains(strings.ToLower(contentType), "application/wasm") {
		return contentType, strings.TrimSpace(resp.Header.Get("Content-Encoding")), fmt.Errorf("release smoke validation expected application/wasm content type, got %q", contentType)
	}
	return contentType, strings.TrimSpace(resp.Header.Get("Content-Encoding")), nil
}

func startReleaseStartupProbeServer(outDir string, binaryName string, wasmExecPath string, artifacts map[string]releaseArtifactRecord, timeoutMs int) (string, string, func(), error) {
	listener, err := net.Listen("tcp", joinHostPort(defaultHost, "0"))
	if err != nil {
		return "", "", nil, fmt.Errorf("listen for startup measurement probe: %w", err)
	}
	transportEncoding := releaseStartupTransportEncoding(artifacts)
	mux := http.NewServeMux()
	probeHTML := renderReleaseStartupProbeHTML(binaryName, transportEncoding, timeoutMs)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			http.Redirect(w, r, "/__gwc/startup-probe.html", http.StatusTemporaryRedirect)
			return
		case "/__gwc/startup-probe.html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write([]byte(probeHTML))
			return
		case "/__gwc/wasm_exec.js":
			http.ServeFile(w, r, wasmExecPath)
			return
		case "/" + binaryName:
			releaseServeStartupWasm(w, r, outDir, binaryName, transportEncoding)
			return
		default:
			applyDevHeaders(w, r)
			http.FileServer(http.Dir(outDir)).ServeHTTP(w, r)
			return
		}
	})
	server := &http.Server{Handler: mux}
	go func() {
		_ = server.Serve(listener)
	}()
	shutdown := func() {
		_ = server.Close()
		_ = listener.Close()
	}
	return "http://" + listener.Addr().String() + "/__gwc/startup-probe.html", transportEncoding, shutdown, nil
}

func releaseStartupTransportEncoding(artifacts map[string]releaseArtifactRecord) string {
	if _, ok := artifacts["gzip"]; ok {
		return "gzip"
	}
	if _, ok := artifacts["brotli"]; ok {
		return "br"
	}
	return "identity"
}

func releaseServeStartupWasm(w http.ResponseWriter, r *http.Request, outDir string, binaryName string, transportEncoding string) {
	applyDevHeaders(w, r)
	w.Header().Set("Cache-Control", "no-store")
	switch transportEncoding {
	case "gzip":
		w.Header().Set("Content-Encoding", "gzip")
		http.ServeFile(w, r, filepath.Join(outDir, binaryName+".gz"))
	case "br":
		w.Header().Set("Content-Encoding", "br")
		http.ServeFile(w, r, filepath.Join(outDir, binaryName+".br"))
	default:
		http.ServeFile(w, r, filepath.Join(outDir, binaryName))
	}
}

func renderReleaseStartupProbeHTML(binaryName string, transportEncoding string, timeoutMs int) string {
	probeGzipPath := "null"
	if transportEncoding == "gzip" {
		probeGzipPath = jsStringLiteral("/" + binaryName + ".gz")
	}
	return "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>GWC Release Startup Probe</title>\n  <script src=\"/__gwc/wasm_exec.js\"></script>\n</head>\n<body>\n  <div id=\"app\"></div>\n  <button id=\"__gwc_probe_button\" style=\"position:fixed;top:12px;right:12px;z-index:2147483647\">Probe interaction</button>\n  <script>\n" +
		"(() => {\n" +
		"  const startup = window.__gwcStartupProbe = {\n" +
		"    startedAt: performance.now(),\n" +
		"    transportEncoding: " + jsStringLiteral(transportEncoding) + ",\n" +
		"    readyMs: null,\n" +
		"    readyReason: '',\n" +
		"    interactionMs: null,\n" +
		"    instantiate: null,\n" +
		"    decompression: null,\n" +
		"    error: null\n" +
		"  };\n" +
		"  const mount = document.getElementById('app');\n" +
		"  const button = document.getElementById('__gwc_probe_button');\n" +
		"  const markReady = (reason) => {\n" +
		"    if (startup.readyMs !== null) return;\n" +
		"    startup.readyMs = performance.now() - startup.startedAt;\n" +
		"    startup.readyReason = reason;\n" +
		"  };\n" +
		"  if ((mount.textContent || '').trim() !== '' || mount.childNodes.length > 0) {\n" +
		"    markReady('preexisting');\n" +
		"  }\n" +
		"  const observer = new MutationObserver(() => {\n" +
		"    if ((mount.textContent || '').trim() !== '' || mount.childNodes.length > 0) {\n" +
		"      observer.disconnect();\n" +
		"      requestAnimationFrame(() => markReady('mutation'));\n" +
		"    }\n" +
		"  });\n" +
		"  observer.observe(mount, { childList: true, subtree: true, characterData: true });\n" +
		"  setTimeout(() => markReady('timeout'), " + fmt.Sprintf("%d", timeoutMs) + ");\n" +
		"  button.addEventListener('click', () => {\n" +
		"    const interactionStart = performance.now();\n" +
		"    requestAnimationFrame(() => {\n" +
		"      startup.interactionMs = performance.now() - interactionStart;\n" +
		"    });\n" +
		"  });\n" +
		"  const originalInstantiateStreaming = WebAssembly.instantiateStreaming.bind(WebAssembly);\n" +
		"  WebAssembly.instantiateStreaming = async (source, importObject) => {\n" +
		"    const start = performance.now();\n" +
		"    try {\n" +
		"      const result = await originalInstantiateStreaming(source, importObject);\n" +
		"      startup.instantiate = { mode: 'instantiateStreaming', durationMs: performance.now() - start };\n" +
		"      return result;\n" +
		"    } catch (error) {\n" +
		"      startup.instantiate = { mode: 'instantiateStreaming', durationMs: performance.now() - start, error: String(error) };\n" +
		"      throw error;\n" +
		"    }\n" +
		"  };\n" +
		"  const gzipProbePath = " + probeGzipPath + ";\n" +
		"  if (gzipProbePath && typeof DecompressionStream === 'function') {\n" +
		"    fetch(gzipProbePath, { cache: 'no-store' })\n" +
		"      .then(async (response) => {\n" +
		"        const fetchStart = performance.now();\n" +
		"        const encoded = await response.arrayBuffer();\n" +
		"        const fetchDone = performance.now();\n" +
		"        const decoded = await new Response(new Blob([encoded]).stream().pipeThrough(new DecompressionStream('gzip'))).arrayBuffer();\n" +
		"        const decodedDone = performance.now();\n" +
		"        startup.decompression = {\n" +
		"          encoding: 'gzip',\n" +
		"          encodedBytes: encoded.byteLength,\n" +
		"          decodedBytes: decoded.byteLength,\n" +
		"          fetchMs: fetchDone - fetchStart,\n" +
		"          decompressMs: decodedDone - fetchDone\n" +
		"        };\n" +
		"      })\n" +
		"      .catch((error) => {\n" +
		"        startup.decompression = { encoding: 'gzip', error: String(error) };\n" +
		"      });\n" +
		"  }\n" +
		"  const go = new Go();\n" +
		"  WebAssembly.instantiateStreaming(fetch(" + jsStringLiteral("/"+binaryName) + ", { cache: 'no-store' }), go.importObject)\n" +
		"    .then((result) => go.run(result.instance))\n" +
		"    .catch((error) => {\n" +
		"      startup.error = String(error);\n" +
		"      markReady('error');\n" +
		"    });\n" +
		"})();\n" +
		"  </script>\n</body>\n</html>\n"
}

type releaseManifestSnapshot struct {
	Package     string                           `json:"package"`
	Profile     string                           `json:"profile"`
	GOOS        string                           `json:"goos"`
	GOARCH      string                           `json:"goarch"`
	Artifacts   map[string]releaseArtifactRecord `json:"artifacts"`
	Attribution *releaseAttributionRecord        `json:"attribution,omitempty"`
}

type releasePackageAttributionSnapshot struct {
	Mode     string                     `json:"mode"`
	Packages []releasePackageSizeRecord `json:"packages"`
}

func releaseWriteDiffReport(compareManifest string, manifestPath string, attribution *releaseAttributionRecord, artifacts map[string]releaseArtifactRecord, outDir string) (*releaseDiffArtifactRecord, error) {
	if strings.TrimSpace(compareManifest) == "" {
		return nil, nil
	}
	baseline, err := releaseReadManifestSnapshot(compareManifest)
	if err != nil {
		return nil, err
	}
	artifactChanges := releaseCompareArtifactRecords(baseline.Artifacts, artifacts)
	likelyCulprits, err := releaseComparePackageAttributionRecords(compareManifest, baseline.Attribution, outDir, attribution)
	if err != nil {
		return nil, err
	}
	fileName := "wasm-release-size-diff.json"
	payload := map[string]interface{}{
		"baselineManifestPath": baselinePathForJSON(compareManifest),
		"currentManifestPath":  manifestPath,
		"artifactChanges":      artifactChanges,
		"likelyCulprits":       likelyCulprits,
	}
	encoded, err := releaseMarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode release diff report: %w", err)
	}
	encoded = append(encoded, '\n')
	if err := os.WriteFile(filepath.Join(outDir, fileName), encoded, 0644); err != nil {
		return nil, fmt.Errorf("write release diff report: %w", err)
	}
	return &releaseDiffArtifactRecord{
		Path:                 fileName,
		BaselineManifestPath: compareManifest,
		ArtifactChanges:      artifactChanges,
		LikelyCulprits:       likelyCulprits,
	}, nil
}

func baselinePathForJSON(path string) string {
	return path
}

func releaseReadManifestSnapshot(path string) (releaseManifestSnapshot, error) {
	manifestBytes, err := os.ReadFile(path)
	if err != nil {
		return releaseManifestSnapshot{}, fmt.Errorf("read compare manifest: %w", err)
	}
	var manifest releaseManifestSnapshot
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return releaseManifestSnapshot{}, fmt.Errorf("parse compare manifest: %w", err)
	}
	if strings.TrimSpace(manifest.Package) == "" {
		return releaseManifestSnapshot{}, errors.New("compare manifest is missing package")
	}
	if manifest.GOOS != "js" || manifest.GOARCH != "wasm" {
		return releaseManifestSnapshot{}, errors.New("compare manifest must describe a js/wasm release")
	}
	if len(manifest.Artifacts) == 0 {
		return releaseManifestSnapshot{}, errors.New("compare manifest is missing artifacts")
	}
	return manifest, nil
}

func releaseCompareArtifactRecords(baseline map[string]releaseArtifactRecord, current map[string]releaseArtifactRecord) []releaseArtifactDiffRecord {
	keys := map[string]struct{}{}
	for key := range baseline {
		keys[key] = struct{}{}
	}
	for key := range current {
		keys[key] = struct{}{}
	}
	names := make([]string, 0, len(keys))
	for key := range keys {
		names = append(names, key)
	}
	sort.Strings(names)
	changes := make([]releaseArtifactDiffRecord, 0, len(names))
	for _, name := range names {
		baselineArtifact, hasBaseline := baseline[name]
		currentArtifact, hasCurrent := current[name]
		record := releaseArtifactDiffRecord{Name: name}
		if hasBaseline {
			record.BaselinePath = baselineArtifact.Path
			record.BaselineBytes = int64ptr(baselineArtifact.Bytes)
		}
		if hasCurrent {
			record.CurrentPath = currentArtifact.Path
			record.CurrentBytes = int64ptr(currentArtifact.Bytes)
		}
		switch {
		case hasBaseline && hasCurrent:
			delta := currentArtifact.Bytes - baselineArtifact.Bytes
			record.DeltaBytes = int64ptr(delta)
			record.DeltaPercent = releasePercentDeltaPointer(baselineArtifact.Bytes, currentArtifact.Bytes)
			if delta > 0 {
				record.Status = "grew"
			} else if delta < 0 {
				record.Status = "shrank"
			} else {
				record.Status = "unchanged"
			}
		case hasCurrent:
			record.Status = "added"
		default:
			record.Status = "removed"
		}
		changes = append(changes, record)
	}
	return changes
}

func releaseComparePackageAttributionRecords(baselineManifestPath string, baselineAttribution *releaseAttributionRecord, currentOutDir string, currentAttribution *releaseAttributionRecord) ([]releasePackageDiffRecord, error) {
	if baselineAttribution == nil || currentAttribution == nil {
		return nil, nil
	}
	baselinePackages, err := releaseReadPackageAttributionSnapshot(filepath.Join(filepath.Dir(baselineManifestPath), filepath.FromSlash(baselineAttribution.Path)))
	if err != nil {
		return nil, err
	}
	currentPackages, err := releaseReadPackageAttributionSnapshot(filepath.Join(currentOutDir, filepath.FromSlash(currentAttribution.Path)))
	if err != nil {
		return nil, err
	}
	baselineMap := map[string]releasePackageSizeRecord{}
	for _, record := range baselinePackages.Packages {
		baselineMap[record.ImportPath] = record
	}
	currentMap := map[string]releasePackageSizeRecord{}
	for _, record := range currentPackages.Packages {
		currentMap[record.ImportPath] = record
	}
	keys := map[string]struct{}{}
	for key := range baselineMap {
		keys[key] = struct{}{}
	}
	for key := range currentMap {
		keys[key] = struct{}{}
	}
	deltas := make([]releasePackageDiffRecord, 0, len(keys))
	for importPath := range keys {
		baselineRecord, hasBaseline := baselineMap[importPath]
		currentRecord, hasCurrent := currentMap[importPath]
		record := releasePackageDiffRecord{ImportPath: importPath}
		if hasBaseline {
			record.BaselineArchiveBytes = int64ptr(baselineRecord.ArchiveBytes)
			record.BaselineSourceBytes = int64ptr(baselineRecord.SourceBytes)
		}
		if hasCurrent {
			record.CurrentArchiveBytes = int64ptr(currentRecord.ArchiveBytes)
			record.CurrentSourceBytes = int64ptr(currentRecord.SourceBytes)
		}
		switch {
		case hasBaseline && hasCurrent:
			archiveDelta := currentRecord.ArchiveBytes - baselineRecord.ArchiveBytes
			sourceDelta := currentRecord.SourceBytes - baselineRecord.SourceBytes
			record.ArchiveDeltaBytes = int64ptr(archiveDelta)
			record.ArchiveDeltaPercent = releasePercentDeltaPointer(baselineRecord.ArchiveBytes, currentRecord.ArchiveBytes)
			record.SourceDeltaBytes = int64ptr(sourceDelta)
			if archiveDelta > 0 {
				record.Status = "grew"
			} else if archiveDelta < 0 {
				record.Status = "shrank"
			} else if sourceDelta != 0 {
				record.Status = "source-only-change"
			} else {
				record.Status = "unchanged"
			}
		case hasCurrent:
			record.ArchiveDeltaBytes = int64ptr(currentRecord.ArchiveBytes)
			record.SourceDeltaBytes = int64ptr(currentRecord.SourceBytes)
			record.Status = "added"
		default:
			archiveDelta := -baselineRecord.ArchiveBytes
			sourceDelta := -baselineRecord.SourceBytes
			record.ArchiveDeltaBytes = int64ptr(archiveDelta)
			record.SourceDeltaBytes = int64ptr(sourceDelta)
			record.Status = "removed"
		}
		if record.ArchiveDeltaBytes != nil && *record.ArchiveDeltaBytes > 0 {
			deltas = append(deltas, record)
		}
	}
	sort.Slice(deltas, func(i int, j int) bool {
		left := derefInt64(deltas[i].ArchiveDeltaBytes)
		right := derefInt64(deltas[j].ArchiveDeltaBytes)
		if left != right {
			return left > right
		}
		leftSource := derefInt64(deltas[i].SourceDeltaBytes)
		rightSource := derefInt64(deltas[j].SourceDeltaBytes)
		if leftSource != rightSource {
			return leftSource > rightSource
		}
		return deltas[i].ImportPath < deltas[j].ImportPath
	})
	if len(deltas) > 10 {
		deltas = deltas[:10]
	}
	if len(deltas) == 0 {
		return nil, nil
	}
	return deltas, nil
}

func releaseReadPackageAttributionSnapshot(path string) (releasePackageAttributionSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return releasePackageAttributionSnapshot{}, fmt.Errorf("read package attribution artifact: %w", err)
	}
	var snapshot releasePackageAttributionSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return releasePackageAttributionSnapshot{}, fmt.Errorf("parse package attribution artifact: %w", err)
	}
	return snapshot, nil
}

func releasePercentDeltaPointer(baseline int64, current int64) *float64 {
	if baseline == 0 {
		return nil
	}
	delta := float64(current-baseline) / float64(baseline) * 100
	return &delta
}

func int64ptr(value int64) *int64 {
	return &value
}

func derefInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
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
	printResolutionTrace(summary.Resolution, []string{"app", "root", "output", "profile"}, "  ")
}

func printReleaseSummary(summary releaseSummary) {
	fmt.Println("GWC release")
	fmt.Printf("  profile:      %s\n", summary.Profile.Name)
	fmt.Printf("  app:          %s\n", summary.AppPath)
	fmt.Printf("  project root: %s\n", summary.ProjectRoot)
	fmt.Printf("  package dir:  %s\n", summary.PackageDir)
	fmt.Printf("  out dir:      %s\n", summary.OutDir)
	fmt.Printf("  manifest:     %s\n", summary.ManifestPath)
	if summary.Optimizer != nil {
		fmt.Printf("  optimizer:    %s (%s)\n", summary.Optimizer.Mode, summary.Optimizer.Tool)
	}
	if summary.Attribution != nil {
		fmt.Printf("  attribution:  %s (%s, %d packages)\n", summary.Attribution.Path, summary.Attribution.Mode, summary.Attribution.PackageCount)
	}
	if summary.Startup != nil {
		fmt.Printf("  startup:      %s (%s via %s)\n", summary.Startup.Path, summary.Startup.Mode, summary.Startup.TransportEncoding)
	}
	if summary.Validation != nil {
		fmt.Printf("  validation:   %s (%s)\n", summary.Validation.Path, summary.Validation.WasmContentType)
	}
	if summary.Diff != nil {
		fmt.Printf("  diff:         %s (baseline %s)\n", summary.Diff.Path, summary.Diff.BaselineManifestPath)
	}
	keys := make([]string, 0, len(summary.Artifacts))
	for key := range summary.Artifacts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		artifact := summary.Artifacts[key]
		fmt.Printf("  artifact[%s]: %s (%d bytes)\n", key, artifact.Path, artifact.Bytes)
	}
	printResolutionTrace(summary.Resolution, []string{"app", "root", "output", "profile"}, "  ")
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
	if summary.Audit != nil {
		auditStatus := "PASS"
		if !summary.Audit.OK {
			auditStatus = "FAIL"
		}
		fmt.Printf("  audit[%s]: %s", summary.Audit.Mode, auditStatus)
		if summary.Audit.Policy != "" {
			fmt.Printf(" (policy: %s)", summary.Audit.Policy)
		}
		if strings.TrimSpace(summary.AuditMinSeverity) != "" {
			fmt.Printf(" (min severity: %s)", summary.AuditMinSeverity)
		}
		fmt.Println()
	}
	printResolutionTrace(summary.Resolution, []string{"app", "root", "output", "profile"}, "  ")
}

func printUsage() {
	fmt.Println("GWC launcher")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  go run ./tools/gwc <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  bench      Discover native/js-wasm benchmark packages, capture raw benchmark output, compare files with benchstat, and write docs/benchmarks JSON output")
	fmt.Println("  build      Build a js/wasm app with an explicit launcher profile")
	fmt.Println("  test       Run explicit launcher-owned test lanes such as unit, wasm, hydration, browser, and release")
	fmt.Println("  examples   Serve the examples catalog from a Go-native server")
	fmt.Println("  dev        Run the Go-native dev entrypoint and forward to livereload")
	fmt.Println("  serve      Serve a static directory, wasm artifact, wasm_exec.js, and optional JSON fixtures")
	fmt.Println("  files      List project files with repeatable extension and directory filters")
	fmt.Println("  tailwind   Build shared Tailwind CSS and generated class manifests through the launcher-owned Tailwind path")
	fmt.Println("  dashboard  Monitor live-reload clients and project AI provider configuration from a launcher-owned dashboard")
	fmt.Println("  doctor     Check local toolchains, runtime assets, project signals, and optional golden-path audit anchors")
	fmt.Println("  env        Print launcher-relevant environment variables and current values")
	fmt.Println("  seed       Provision local dev identities and fixture data through a seed package")
	fmt.Println("  import     Convert a static HTML or JSX file into an inspectable GWC project")
	fmt.Println("  release    Package a js/wasm release with manifest and compressed sidecars")
	fmt.Println("  verify     Run app-local Go tests when present and perform a CI-profile wasm build")
	fmt.Println("  wasm       Run wasm-focused build experiment helpers such as `wasm measure`")
	fmt.Println("  start      Run the scaffold TUI for preset and project setup")
	fmt.Println("  bootstrap  Run prerequisite checks, then start a scaffold or examples bootstrap flow")
}

func printSeedSummary(summary seedSummary) {
	fmt.Println("GWC seed")
	fmt.Printf("  project root: %s\n", summary.ProjectRoot)
	fmt.Printf("  command:      %s\n", summary.CommandPath)
	if summary.DatabasePath != "" {
		fmt.Printf("  database:     %s\n", summary.DatabasePath)
	}
	for _, credential := range summary.Credentials {
		fmt.Printf("  account:      %s / %s", credential.Email, credential.Password)
		if credential.Role != "" {
			fmt.Printf(" (%s)", credential.Role)
		}
		fmt.Println()
	}
	if summary.Output != "" {
		fmt.Printf("  output:       %s\n", summary.Output)
	}
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
	printResolutionTrace(summary.Resolution, []string{"app", "root"}, "  ")
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
