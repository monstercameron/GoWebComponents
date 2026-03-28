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

func (parseL launcher) resolvedExamplesWasmDir() string {
	if strings.TrimSpace(parseL.examplesWasmDir) != "" {
		return parseL.examplesWasmDir
	}
	return resolveLauncherExamplesWasmDir(parseL.repoRoot, parseL.staticDir)
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

var releaseRunStartupProbeWithPlaywright = runReleaseStartupProbeWithPlaywright

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

func launcherCommandAndArgs(parseArgs []string) (string, []string) {
	_, parseRemaining, parseErr := parseLauncherGlobalCLIOptions(parseArgs)
	if parseErr != nil {
		parseRemaining = parseArgs
	}
	if len(parseRemaining) == 0 {
		return "", nil
	}
	return strings.TrimSpace(parseRemaining[0]), parseRemaining[1:]
}

func launcherCommandSupportsJSON(parseCommand string) bool {
	switch strings.TrimSpace(strings.ToLower(parseCommand)) {
	case "bench", "benchmark", "build", "deploy", "dev", "doctor", "env", "examples", "export", "files", "init", "inspect", "lint", "migrate", "prerender", "release", "review", "seed", "tailwind", "test", "upgrade", "verify", "wasm":
		return true
	default:
		return false
	}
}

func launcherJSONRequestedForCommand(parseCommand string, parseArgs []string) bool {
	if isExamplesManagedPathCommand(parseCommand, parseArgs) {
		return launcherJSONRequestedForCommand("examples", buildExamplesManagedPathCommandArgs(parseCommand, parseArgs))
	}
	if !launcherCommandSupportsJSON(parseCommand) {
		return false
	}
	for _, parseArg := range parseArgs {
		parseTrimmed := strings.TrimSpace(strings.ToLower(parseArg))
		if parseTrimmed == "-json" || parseTrimmed == "--json" || strings.HasPrefix(parseTrimmed, "-json=") || strings.HasPrefix(parseTrimmed, "--json=") {
			return true
		}
	}
	return false
}

func launcherRequestedJSONOutput(parseArgs []string) bool {
	parseCommand, parseCommandArgs := launcherCommandAndArgs(parseArgs)
	return launcherJSONRequestedForCommand(parseCommand, parseCommandArgs)
}

func buildLauncherFailureDiagnostic(parseArgs []string, parseErr error) launcherFailureDiagnostic {
	parseCommand, _ := launcherCommandAndArgs(parseArgs)
	parseMessage := strings.TrimSpace(parseErr.Error())
	parsePhase := "execution"
	parseCategory := "execution"
	parseCode := "command_failed"
	parseOverride := ""
	parseLower := strings.ToLower(parseMessage)

	switch {
	case detectRunnerOverrideField(parseMessage) != "":
		parsePhase = "configuration"
		parseCategory = "configuration"
		parseCode = "invalid_runner_override"
		parseOverride = detectRunnerOverrideField(parseMessage)
	case strings.Contains(parseLower, "policy") || strings.Contains(parseLower, "not approved") || strings.Contains(parseLower, "blocked by security boundary"):
		parsePhase = "policy"
		parseCategory = "policy"
		parseCode = "policy_violation"
	case strings.Contains(parseLower, "resolve ") ||
		strings.Contains(parseLower, "parse ") ||
		strings.Contains(parseLower, "unknown ") ||
		strings.Contains(parseLower, "required") ||
		strings.Contains(parseLower, "configured ") ||
		strings.Contains(parseLower, "does not exist") ||
		strings.Contains(parseLower, "interactive terminal") ||
		strings.Contains(parseLower, "use either -compression or -skip-compression"):
		parsePhase = "configuration"
		parseCategory = "configuration"
		parseCode = "invalid_configuration"
	case strings.Contains(parseLower, "go test failed") || strings.Contains(parseLower, "go build failed"):
		parsePhase = "execution"
		parseCategory = "code"
		parseCode = "code_failure"
	case strings.Contains(parseLower, "verify audit found"):
		parsePhase = "validation"
		parseCategory = "validation"
		parseCode = "audit_failed"
	case strings.Contains(parseLower, "verify checks reported failures") || strings.Contains(parseLower, "doctor found required checks"):
		parsePhase = "validation"
		parseCategory = "validation"
		parseCode = "checks_failed"
	case strings.Contains(parseLower, "smoke validation"):
		parsePhase = "validation"
		parseCategory = "validation"
		parseCode = "smoke_failed"
	case strings.TrimSpace(strings.ToLower(parseCommand)) == "dev" && (strings.Contains(parseLower, "listen") || strings.Contains(parseLower, "bind ") || strings.Contains(parseLower, "livereload") || strings.Contains(parseLower, "serve")):
		parsePhase = "runtime"
		parseCategory = "runtime"
		parseCode = "startup_failed"
	}

	return launcherFailureDiagnostic{
		OK:       false,
		Command:  parseCommand,
		Phase:    parsePhase,
		Category: parseCategory,
		Code:     parseCode,
		Message:  parseMessage,
		Override: parseOverride,
	}
}

func detectRunnerOverrideField(parseMessage string) string {
	parseLower := strings.ToLower(strings.TrimSpace(parseMessage))
	for _, parseField := range []string{
		"artifactRoot",
		"browserWorkspace",
		"generatedProjectRoot",
		"goWasmExec",
		"livereloadClientScript",
		"livereloadWorkspace",
		"wasmExecJS",
		"workspaceBuildRoot",
	} {
		parseFieldLower := strings.ToLower(parseField)
		if strings.Contains(parseLower, "configured "+strings.ToLower(parseField)) ||
			strings.Contains(parseLower, "resolve "+parseFieldLower+" override") {
			return parseField
		}
	}
	return ""
}

func printLauncherError(parseW io.Writer, parseArgs []string, parseErr error) {
	if parseW == nil {
		return
	}
	if launcherRequestedJSONOutput(parseArgs) {
		parseEncoder := json.NewEncoder(parseW)
		parseEncoder.SetIndent("", "  ")
		if parseEncodeErr := parseEncoder.Encode(buildLauncherFailureDiagnostic(parseArgs, parseErr)); parseEncodeErr == nil {
			return
		}
	}
	fmt.Fprintf(parseW, "gwc: %v\n", parseErr)
}

var mainPrintError = func(err error) {
	printLauncherError(os.Stderr, mainArgs()[1:], err)
}

func main() {
	parseRepoRoot, parseErr := mainResolveRepoRoot()
	if parseErr != nil {
		mainPrintError(parseErr)
		mainExit(1)
	}
	parseExamplesWasmDir, parseErr := resolveLauncherWorkspaceBuildPath(parseRepoRoot, "examples")
	if parseErr != nil {
		mainPrintError(fmt.Errorf("resolve examples build root: %w", parseErr))
		mainExit(1)
	}

	parseL := launcher{
		repoRoot:        parseRepoRoot,
		examplesDir:     filepath.Join(parseRepoRoot, "examples"),
		staticDir:       filepath.Join(parseRepoRoot, "examples", "static"),
		examplesWasmDir: parseExamplesWasmDir,
	}

	if parseErr2 := mainRunLauncher(parseL, mainArgs()[1:]); parseErr2 != nil {
		mainPrintError(parseErr2)
		mainExit(1)
	}
}

func (parseL launcher) run(parseArgs []string) error {
	parseGlobalOptions, parseRemainingArgs, parseErr := parseLauncherGlobalCLIOptions(parseArgs)
	if parseErr != nil {
		return parseErr
	}

	if len(parseRemainingArgs) == 0 {
		printUsage()
		return nil
	}

	parseCommand := parseRemainingArgs[0]
	parseCommandArgs := parseRemainingArgs[1:]

	switch parseCommand {
	case "help", "-h", "--help":
		printUsage()
		return nil
	}

	parseCwd, parseCwdErr := launcherConfigGetwd()
	if parseCwdErr != nil {
		parseCwd = ""
	}
	parseEnterprise, parseErr := resolveLauncherEnterpriseConfig(parseCwd, parseGlobalOptions)
	if parseErr != nil {
		return parseErr
	}
	launcherActiveEnterpriseConfig = parseEnterprise.Effective
	launcherActiveEnterpriseSources = parseEnterprise.Sources
	defer func() {
		launcherActiveEnterpriseConfig = defaultLauncherEnterpriseConfig()
		launcherActiveEnterpriseSources = launcherEnterpriseConfigSources{FrameworkDefaults: true}
	}()
	if parseErr2 := validateLauncherExtensionSecurity(launcherActiveEnterpriseConfig); parseErr2 != nil {
		return parseErr2
	}
	parseJsonOutputRequested := launcherJSONRequestedForCommand(parseCommand, parseCommandArgs)
	if !parseJsonOutputRequested {
		launcherPrintExtensionReport(parseCommand, parseEnterprise)
	}

	if parseErr3 := runLauncherCommandHooks(launcherActiveEnterpriseConfig.Hooks, "pre", parseCommand, parseCommandArgs, parseL.repoRoot, launcherActiveEnterpriseSources); parseErr3 != nil {
		return parseErr3
	}

	parseDispatchErr := parseL.dispatchCommand(parseCommand, parseCommandArgs)
	parsePostHookErr := runLauncherCommandHooks(launcherActiveEnterpriseConfig.Hooks, "post", parseCommand, parseCommandArgs, parseL.repoRoot, launcherActiveEnterpriseSources)
	if parseDispatchErr != nil {
		return parseDispatchErr
	}
	if parsePostHookErr != nil {
		return parsePostHookErr
	}
	return nil
}

func (parseL launcher) dispatchCommand(parseCommand string, parseArgs []string) error {
	switch parseCommand {
	case "test":
		return runTestCommand(parseL, parseArgs)
	case "examples":
		return runExamplesCommand(parseL, parseArgs)
	case "build":
		return runBuildCommand(parseL, parseArgs)
	case "bench", "benchmark":
		return runBenchmarkCommand(parseL, parseArgs)
	case "release":
		return runReleaseCommand(parseL, parseArgs)
	case "dev":
		return runDevCommand(parseL, parseArgs)
	case "serve":
		return runServeCommand(parseL, parseArgs)
	case "files":
		return runFilesCommand(parseL, parseArgs)
	case "lint", "review":
		return runLintCommand(parseL, parseArgs)
	case "init":
		return runInitCommand(parseL, parseArgs)
	case "inspect":
		return runInspectCommand(parseL, parseArgs)
	case "upgrade":
		return runUpgradeCommand(parseL, parseArgs)
	case "migrate":
		return runMigrateCommand(parseL, parseArgs)
	case "prerender":
		return runPrerenderCommand(parseL, parseArgs)
	case "export":
		return runExportCommand(parseL, parseArgs)
	case "tailwind":
		return runTailwindCommand(parseL, parseArgs)
	case "dashboard":
		return runDashboardCommand(parseL, parseArgs)
	case "doctor":
		return runDoctorCommand(parseL, parseArgs)
	case "deploy":
		return runDeployCommand(parseL, parseArgs)
	case "env":
		return runEnvCommand(parseL, parseArgs)
	case "verify":
		return runVerifyCommand(parseL, parseArgs)
	case "seed":
		return runSeedCommand(parseL, parseArgs)
	case "import":
		return runImportCommand(parseL, parseArgs)
	case "start":
		return runStartCommand(parseL, parseArgs)
	case "bootstrap":
		return runBootstrapCommand(parseL, parseArgs)
	case "wasm":
		return runWasmCommand(parseL, parseArgs)
	default:
		if isExamplesManagedPathCommand(parseCommand, parseArgs) {
			return parseL.runExamplesManaged(buildExamplesManagedPathCommandArgs(parseCommand, parseArgs))
		}
		return fmt.Errorf("unknown command %q", parseCommand)
	}
}

func (parseL launcher) runTest(parseArgs []string) error {
	parseFs := flag.NewFlagSet("test", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "Legacy alias for -app")
	parseRoot := parseFs.String("root", "", "Project root used for test lane resolution")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	var parseLaneFlags stringListFlag
	parseFs.Var(&parseLaneFlags, "lane", "Test lane to run; repeat or comma-separate: unit, wasm, hydration, browser, release, all")
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
	case "wasm":
		return parseL.runWasmTestLane(parseConfig.rootPath, false)
	case "hydration":
		return parseL.runWasmTestLane(parseConfig.rootPath, true)
	case "browser":
		return parseL.runBrowserTestLane(parseConfig.rootPath)
	case "release":
		return parseL.runReleaseTestLane(parseConfig)
	default:
		return testLaneSummary{}, fmt.Errorf("unknown test lane %q", parseLane)
	}
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
	for _, parsePart := range strings.Split(parseValue, ",") {
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
		case "wasm", "go-wasm":
			parseAppendLane("wasm")
		case "hydration", "hydrate":
			parseAppendLane("hydration")
		case "browser", "playwright":
			parseAppendLane("browser")
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
	if strings.HasSuffix(parseApproved, ".x") {
		parsePrefix := strings.TrimSuffix(parseApproved, ".x")
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

func (parseL launcher) runVerify(parseArgs []string) error {
	parseFs := flag.NewFlagSet("verify", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "Legacy alias for -app")
	parseRoot := parseFs.String("root", "", "Project root used for test and build resolution")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
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

	if *parseSkipTests {
		parseSummary.Tests.Skipped = true
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
				return parseErr7
			}
			parseSummary.Tests.Ran = true
		} else {
			parseSummary.Tests.Skipped = true
		}
	}

	buildSummary, parseErr2 := verifyExecuteBuild(buildConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	parseSummary.Build = buildSummary
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
	}

	if *parseJsonOutput {
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

func defaultSeedDatabasePath(parseCommandDir string) string {
	if !isChatWizardSeedCommand(parseCommandDir) {
		return ""
	}
	parseExampleRoot := filepath.Dir(filepath.Dir(parseCommandDir))
	return filepath.Join(parseExampleRoot, "bin", "runtime", "test_chat.db")
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

func (parseL launcher) runRelease(parseArgs []string) error {
	parseFs := flag.NewFlagSet("release", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "Legacy alias for -app")
	parseRoot := parseFs.String("root", "", "Project root used for output resolution")
	parseOutDir := parseFs.String("out-dir", "", "Release output directory")
	parseBinaryName := parseFs.String("binary-name", "", "Primary wasm artifact filename")
	parseManifestName := parseFs.String("manifest-name", "wasm-release-manifest.json", "Release manifest filename")
	parseBudgetsPath := parseFs.String("budgets", "", "Optional path to a JSON budgets file")
	parseCompareManifest := parseFs.String("compare-manifest", "", "Optional baseline release manifest to diff against")
	parseProfile := parseFs.String("profile", "release", "Release build profile")
	parseCompression := parseFs.String("compression", "", "Compression sidecars: none, gzip, brotli, or gzip+brotli")
	parsePostLinkOpt := parseFs.String("post-link-opt", "", "Optional post-link optimization: none or wasm-opt")
	parseSizeAttribution := parseFs.String("size-attribution", "", "Optional size attribution: none or packages")
	parseStartupMeasure := parseFs.String("startup-measure", "", "Optional startup measurement: none or browser")
	parseStartupTimeoutMs := parseFs.Int("startup-timeout-ms", 30000, "Startup measurement timeout in milliseconds")
	parseValidateSmoke := parseFs.Bool("validate-smoke", false, "Run post-build release smoke validation")
	parseSkipCompression := parseFs.Bool("skip-compression", false, "Skip gzip sidecar generation")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parsePluginResults, parseErr2 := runLauncherPluginsForCapability("release_validator", "release", parseArgs, parseL.repoRoot, launcherActiveEnterpriseSources)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseErr3 := enforcePluginChecks(parsePluginResults, "release_validator"); parseErr3 != nil {
		return parseErr3
	}

	isParseSkipCompressionSet := false
	isParseCompressionSet := false
	parseFs.Visit(func(parseFlag *flag.Flag) {
		if parseFlag.Name == "skip-compression" {
			isParseSkipCompressionSet = true
		}
		if parseFlag.Name == "compression" {
			isParseCompressionSet = true
		}
	})
	if isParseSkipCompressionSet && isParseCompressionSet {
		return errors.New("use either -compression or -skip-compression, not both")
	}
	parseCompressionPolicy := *parseCompression
	if isParseSkipCompressionSet {
		parseCompressionPolicy = "none"
	}

	parseConfig, parseErr2 := resolveReleaseConfig(releaseConfig{
		appPath:          firstNonEmpty(*parseApp, *parseMainPath),
		rootPath:         *parseRoot,
		outDir:           *parseOutDir,
		binaryName:       *parseBinaryName,
		manifestName:     *parseManifestName,
		budgetsPath:      *parseBudgetsPath,
		compareManifest:  *parseCompareManifest,
		profile:          *parseProfile,
		compression:      parseCompressionPolicy,
		postLinkOpt:      *parsePostLinkOpt,
		sizeAttribution:  *parseSizeAttribution,
		startupMeasure:   *parseStartupMeasure,
		startupTimeoutMs: *parseStartupTimeoutMs,
		validateSmoke:    *parseValidateSmoke,
		compressionSet:   isParseCompressionSet || isParseSkipCompressionSet,
		skipCompression:  *parseSkipCompression,
		skipCompressSet:  isParseSkipCompressionSet,
		json:             *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}
	if parseErr4 := enforceEnterpriseGoToolchainPolicy(parseConfig.rootPath, launcherActiveEnterpriseConfig.Policy); parseErr4 != nil {
		return parseErr4
	}
	if parseErr5 := enforceEnterpriseReleasePolicy(parseConfig, launcherActiveEnterpriseConfig.Policy); parseErr5 != nil {
		return parseErr5
	}

	parseSummary, parseErr2 := executeRelease(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printReleaseSummary(parseSummary)
	return nil
}

func (parseL launcher) runBuild(parseArgs []string) error {
	parseFs := flag.NewFlagSet("build", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseApp := parseFs.String("app", "", "Path to the app main.go file or app directory")
	parseMainPath := parseFs.String("main", "", "Legacy alias for -app")
	parseRoot := parseFs.String("root", "", "Project root used for output resolution")
	parseOut := parseFs.String("out", "", "WASM output path")
	parseOutput := parseFs.String("output", "", "Legacy alias for -out")
	parseProfile := parseFs.String("profile", "", "Build profile: development, ci, benchmark, or release")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveBuildConfig(buildConfig{
		appPath:    firstNonEmpty(*parseApp, *parseMainPath),
		rootPath:   *parseRoot,
		outputPath: firstNonEmpty(*parseOut, *parseOutput),
		profile:    *parseProfile,
		json:       *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseErr2 := executeBuild(parseConfig)
	if parseErr2 != nil {
		return parseErr2
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printBuildSummary(parseSummary)
	return nil
}

func resolveBuildConfig(parseConfig buildConfig) (buildConfig, error) {
	parseResolved := parseConfig
	parseResolved.resolution = cloneResolutionTrace(parseConfig.resolution)
	isParseExplicitOutputPath := strings.TrimSpace(parseConfig.outputPath) != ""
	parseCwd, parseErr := buildGetwd()
	if parseErr != nil {
		return buildConfig{}, parseErr
	}

	parseMetadata, parseMetadataDir, hasMetadata, parseErr := resolveScaffoldMetadataForConfig(parseCwd, parseResolved.rootPath, parseResolved.appPath)
	if parseErr != nil {
		return buildConfig{}, parseErr
	}
	if hasMetadata {
		if strings.TrimSpace(parseResolved.appPath) == "" && strings.TrimSpace(parseMetadata.Tooling.AppPath) != "" {
			parseResolved.appPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.AppPath))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.rootPath) == "" {
			parseResolved.rootPath = parseMetadataDir
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.outputPath) == "" && strings.TrimSpace(parseMetadata.Tooling.WASMPath) != "" {
			parseResolved.outputPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.WASMPath))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.profile) == "" && strings.TrimSpace(parseMetadata.Tooling.DefaultBuildProfile) != "" {
			parseResolved.profile = strings.TrimSpace(parseMetadata.Tooling.DefaultBuildProfile)
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "gwc-start.json")
		}
	}
	if strings.TrimSpace(parseConfig.appPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.rootPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "explicit flag")
	}
	if isParseExplicitOutputPath {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.profile) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "explicit flag")
	}

	if strings.TrimSpace(parseResolved.appPath) == "" {
		parseResolved.appPath, parseErr = detectAppPath(parseCwd)
		if parseErr != nil {
			return buildConfig{}, parseErr
		}
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "convention fallback")
	}
	parseResolved.appPath, parseErr = normalizeExistingPath(parseCwd, parseResolved.appPath)
	if parseErr != nil {
		return buildConfig{}, fmt.Errorf("resolve app path: %w", parseErr)
	}

	parseAppDir := parseResolved.appPath
	parseInfo, parseErr := os.Stat(parseResolved.appPath)
	if parseErr != nil {
		return buildConfig{}, fmt.Errorf("inspect app path: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		parseAppDir = filepath.Dir(parseResolved.appPath)
	}

	if strings.TrimSpace(parseResolved.rootPath) == "" {
		parseResolved.rootPath = parseAppDir
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "convention fallback")
	}
	parseResolved.rootPath, parseErr = normalizePath(parseCwd, parseResolved.rootPath)
	if parseErr != nil {
		return buildConfig{}, fmt.Errorf("resolve root path: %w", parseErr)
	}

	if strings.TrimSpace(parseResolved.outputPath) == "" {
		parseDefaultOutputPath, parseSource, parseErr2 := resolveLauncherDefaultBuildOutput(parseResolved.rootPath)
		if parseErr2 != nil {
			return buildConfig{}, parseErr2
		}
		parseResolved.outputPath = parseDefaultOutputPath
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", parseSource)
	}
	parseResolved.outputPath, parseErr = normalizePath(parseCwd, parseResolved.outputPath)
	if parseErr != nil {
		return buildConfig{}, fmt.Errorf("resolve output path: %w", parseErr)
	}

	parseProfile, parseErr := resolveBuildProfile(strings.TrimSpace(parseResolved.profile))
	if parseErr != nil {
		return buildConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.profile) == "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "convention fallback")
	}
	parseResolved.profile = parseProfile.Name
	return parseResolved, nil
}

func resolveBuildProfile(parseProfile string) (buildProfile, error) {
	switch strings.TrimSpace(strings.ToLower(parseProfile)) {
	case "", "development", "dev":
		return buildProfile{Name: "development", Trimpath: false}, nil
	case "ci", "verification", "verify":
		return buildProfile{Name: "ci", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false"}, nil
	case "benchmark", "bench":
		return buildProfile{Name: "benchmark", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false"}, nil
	case "release", "production", "prod":
		return buildProfile{Name: "release", Trimpath: true, Ldflags: "-s -w", BuildVCS: "false"}, nil
	default:
		return buildProfile{}, fmt.Errorf("unknown build profile %q", parseProfile)
	}
}

func resolveReleaseConfig(parseConfig releaseConfig) (releaseConfig, error) {
	parseResolved := parseConfig
	parseResolved.resolution = cloneResolutionTrace(parseConfig.resolution)
	isParseExplicitOutDir := strings.TrimSpace(parseConfig.outDir) != ""
	parseCwd, parseErr := buildGetwd()
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}

	parseMetadata, parseMetadataDir, hasMetadata, parseErr := resolveScaffoldMetadataForConfig(parseCwd, parseResolved.rootPath, parseResolved.appPath)
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	if hasMetadata {
		if strings.TrimSpace(parseResolved.appPath) == "" && strings.TrimSpace(parseMetadata.Tooling.AppPath) != "" {
			parseResolved.appPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.AppPath))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.rootPath) == "" {
			parseResolved.rootPath = parseMetadataDir
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.outDir) == "" && strings.TrimSpace(parseMetadata.Tooling.ReleaseOutDir) != "" {
			parseResolved.outDir = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.ReleaseOutDir))
			parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "gwc-start.json")
		}
		if strings.TrimSpace(parseResolved.binaryName) == "" && strings.TrimSpace(parseMetadata.Tooling.ReleaseBinaryName) != "" {
			parseResolved.binaryName = strings.TrimSpace(parseMetadata.Tooling.ReleaseBinaryName)
		}
		if strings.TrimSpace(parseResolved.budgetsPath) == "" && strings.TrimSpace(parseMetadata.Tooling.ReleaseBudgetsPath) != "" {
			parseResolved.budgetsPath = filepath.Join(parseMetadataDir, filepath.FromSlash(parseMetadata.Tooling.ReleaseBudgetsPath))
		}
		if !parseResolved.compressionSet {
			parseCompressionPolicy, parseErr2 := normalizeReleaseCompressionPolicy(parseMetadata.Tooling.ReleaseCompression)
			if parseErr2 != nil {
				return releaseConfig{}, parseErr2
			}
			parseResolved.compression = parseCompressionPolicy
		}
	}
	if strings.TrimSpace(parseConfig.appPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.rootPath) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "explicit flag")
	}
	if isParseExplicitOutDir {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", "explicit flag")
	}
	if strings.TrimSpace(parseConfig.profile) != "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "explicit flag")
	}

	if strings.TrimSpace(parseResolved.appPath) == "" {
		parseResolved.appPath, parseErr = detectAppPath(parseCwd)
		if parseErr != nil {
			return releaseConfig{}, parseErr
		}
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "app", "convention fallback")
	}
	parseResolved.appPath, parseErr = normalizeExistingPath(parseCwd, parseResolved.appPath)
	if parseErr != nil {
		return releaseConfig{}, fmt.Errorf("resolve app path: %w", parseErr)
	}

	parseAppDir := parseResolved.appPath
	parseInfo, parseErr := os.Stat(parseResolved.appPath)
	if parseErr != nil {
		return releaseConfig{}, fmt.Errorf("inspect app path: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		parseAppDir = filepath.Dir(parseResolved.appPath)
	}

	if strings.TrimSpace(parseResolved.rootPath) == "" {
		parseResolved.rootPath = parseAppDir
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "convention fallback")
	}
	parseResolved.rootPath, parseErr = normalizePath(parseCwd, parseResolved.rootPath)
	if parseErr != nil {
		return releaseConfig{}, fmt.Errorf("resolve root path: %w", parseErr)
	}

	if strings.TrimSpace(parseResolved.outDir) == "" {
		parseDefaultOutDir, parseSource, parseErr3 := resolveLauncherDefaultReleaseOutDir(parseResolved.rootPath)
		if parseErr3 != nil {
			return releaseConfig{}, parseErr3
		}
		parseResolved.outDir = parseDefaultOutDir
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "output", parseSource)
	}
	parseResolved.outDir, parseErr = normalizePath(parseCwd, parseResolved.outDir)
	if parseErr != nil {
		return releaseConfig{}, fmt.Errorf("resolve release output directory: %w", parseErr)
	}

	parseResolved.binaryName = filepath.Base(strings.TrimSpace(firstNonEmpty(parseResolved.binaryName, defaultScaffoldReleaseBinaryName())))
	if parseResolved.binaryName == "." || parseResolved.binaryName == string(filepath.Separator) || parseResolved.binaryName == "" {
		return releaseConfig{}, errors.New("release binary name is required")
	}
	parseResolved.manifestName = filepath.Base(strings.TrimSpace(firstNonEmpty(parseResolved.manifestName, "wasm-release-manifest.json")))
	if parseResolved.manifestName == "." || parseResolved.manifestName == string(filepath.Separator) || parseResolved.manifestName == "" {
		return releaseConfig{}, errors.New("release manifest name is required")
	}
	if strings.TrimSpace(parseResolved.budgetsPath) != "" {
		parseResolved.budgetsPath, parseErr = normalizePath(parseCwd, parseResolved.budgetsPath)
		if parseErr != nil {
			return releaseConfig{}, fmt.Errorf("resolve budgets path: %w", parseErr)
		}
	}
	if strings.TrimSpace(parseResolved.compareManifest) != "" {
		parseResolved.compareManifest, parseErr = normalizeExistingPath(parseCwd, parseResolved.compareManifest)
		if parseErr != nil {
			return releaseConfig{}, fmt.Errorf("resolve compare manifest path: %w", parseErr)
		}
	}
	parseCompressionPolicy2, parseErr := normalizeReleaseCompressionPolicy(firstNonEmpty(parseResolved.compression, defaultScaffoldReleaseCompression()))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.compression = parseCompressionPolicy2
	parseResolved.skipCompression = parseCompressionPolicy2 == "none"
	parsePostLinkOpt, parseErr := normalizeReleasePostLinkOptimization(firstNonEmpty(parseResolved.postLinkOpt, "none"))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.postLinkOpt = parsePostLinkOpt
	parseSizeAttribution, parseErr := normalizeReleaseSizeAttributionMode(firstNonEmpty(parseResolved.sizeAttribution, "none"))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.sizeAttribution = parseSizeAttribution
	parseStartupMeasure, parseErr := normalizeReleaseStartupMeasureMode(firstNonEmpty(parseResolved.startupMeasure, "none"))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	parseResolved.startupMeasure = parseStartupMeasure
	if parseResolved.startupTimeoutMs <= 0 {
		parseResolved.startupTimeoutMs = 30000
	}

	parseProfile, parseErr := resolveBuildProfile(strings.TrimSpace(firstNonEmpty(parseResolved.profile, "release")))
	if parseErr != nil {
		return releaseConfig{}, parseErr
	}
	if strings.TrimSpace(parseResolved.profile) == "" {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "profile", "convention fallback")
	}
	parseResolved.profile = parseProfile.Name
	return parseResolved, nil
}

func normalizeReleaseCompressionPolicy(parsePolicy string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parsePolicy)) {
	case "", "gzip":
		return "gzip", nil
	case "brotli", "br":
		return "brotli", nil
	case "gzip+brotli", "brotli+gzip", "both", "all":
		return "gzip+brotli", nil
	case "none", "off", "disabled":
		return "none", nil
	default:
		return "", fmt.Errorf("unknown release compression policy %q", parsePolicy)
	}
}

func normalizeReleasePostLinkOptimization(parseMode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "wasm-opt", "size", "optimize":
		return "wasm-opt", nil
	default:
		return "", fmt.Errorf("unknown release post-link optimization %q", parseMode)
	}
}

func normalizeReleaseSizeAttributionMode(parseMode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "packages", "package", "per-package":
		return "packages", nil
	default:
		return "", fmt.Errorf("unknown release size attribution mode %q", parseMode)
	}
}

func normalizeReleaseStartupMeasureMode(parseMode string) (string, error) {
	switch strings.TrimSpace(strings.ToLower(parseMode)) {
	case "", "none", "off", "disabled":
		return "none", nil
	case "browser", "playwright":
		return "browser", nil
	default:
		return "", fmt.Errorf("unknown release startup measurement mode %q", parseMode)
	}
}

func executeBuild(parseConfig buildConfig) (buildSummary, error) {
	parseProfile, parseErr := resolveBuildProfile(parseConfig.profile)
	if parseErr != nil {
		return buildSummary{}, parseErr
	}
	parsePackageDir := parseConfig.appPath
	if parseInfo, parseErr2 := os.Stat(parseConfig.appPath); parseErr2 == nil && !parseInfo.IsDir() {
		parsePackageDir = filepath.Dir(parseConfig.appPath)
	}
	if parseErr3 := os.MkdirAll(filepath.Dir(parseConfig.outputPath), 0755); parseErr3 != nil {
		return buildSummary{}, fmt.Errorf("create build output directory: %w", parseErr3)
	}

	buildArgs := []string{"build", "-o", parseConfig.outputPath}
	if parseProfile.Trimpath {
		buildArgs = append(buildArgs, "-trimpath")
	}
	if strings.TrimSpace(parseProfile.Ldflags) != "" {
		buildArgs = append(buildArgs, "-ldflags="+parseProfile.Ldflags)
	}
	if strings.TrimSpace(parseProfile.BuildVCS) != "" {
		buildArgs = append(buildArgs, "-buildvcs="+parseProfile.BuildVCS)
	}
	buildArgs = append(buildArgs, ".")

	parseCmd := exec.Command("go", buildArgs...)
	parseCmd.Dir = parsePackageDir
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseOutput, parseErr := parseCmd.CombinedOutput()
	if parseErr != nil {
		parseTrimmed := strings.TrimSpace(string(parseOutput))
		if parseTrimmed == "" {
			return buildSummary{}, fmt.Errorf("go build failed: %w", parseErr)
		}
		return buildSummary{}, fmt.Errorf("go build failed: %s", parseTrimmed)
	}

	parseArtifactBytes, parseErr := os.ReadFile(parseConfig.outputPath)
	if parseErr != nil {
		return buildSummary{}, fmt.Errorf("read built wasm artifact: %w", parseErr)
	}
	parseArtifactInfo, parseErr := os.Stat(parseConfig.outputPath)
	if parseErr != nil {
		return buildSummary{}, fmt.Errorf("inspect built wasm artifact: %w", parseErr)
	}
	parseHash := sha256.Sum256(parseArtifactBytes)
	return buildSummary{
		OK:          true,
		Profile:     parseProfile,
		AppPath:     parseConfig.appPath,
		ProjectRoot: parseConfig.rootPath,
		PackageDir:  parsePackageDir,
		OutputPath:  parseConfig.outputPath,
		Bytes:       parseArtifactInfo.Size(),
		SHA256:      fmt.Sprintf("%x", parseHash[:]),
		Resolution:  cloneResolutionTrace(parseConfig.resolution),
	}, nil
}

func executeRelease(parseConfig releaseConfig) (releaseSummary, error) {
	if parseErr := os.MkdirAll(parseConfig.outDir, 0755); parseErr != nil {
		return releaseSummary{}, fmt.Errorf("create release output directory: %w", parseErr)
	}
	parseWasmPath := filepath.Join(parseConfig.outDir, parseConfig.binaryName)
	buildSummary, parseErr2 := releaseExecuteBuild(buildConfig{
		appPath:    parseConfig.appPath,
		rootPath:   parseConfig.rootPath,
		outputPath: parseWasmPath,
		profile:    parseConfig.profile,
	})
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	parseArtifacts := map[string]releaseArtifactRecord{}
	parseOptimizer, parseErr2 := releaseApplyPostLinkOptimization(parseConfig.postLinkOpt, parseWasmPath)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	parseArtifacts["wasm"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, parseWasmPath)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	parseAttribution, parseErr2 := releaseWriteSizeAttribution(parseConfig.sizeAttribution, buildSummary.PackageDir, parseConfig.outDir)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseAttribution != nil {
		parseArtifacts["size_attribution"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseAttribution.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	isParseEmitGzip := parseConfig.compression == "gzip" || parseConfig.compression == "gzip+brotli"
	isParseEmitBrotli := parseConfig.compression == "brotli" || parseConfig.compression == "gzip+brotli"
	if isParseEmitGzip {
		parseGzipPath := parseWasmPath + ".gz"
		if parseErr3 := releaseWriteGzipSidecar(parseWasmPath, parseGzipPath); parseErr3 != nil {
			return releaseSummary{}, parseErr3
		}
		parseArtifacts["gzip"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, parseGzipPath)
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	if isParseEmitBrotli {
		parseBrotliPath := parseWasmPath + ".br"
		if parseErr4 := releaseWriteBrotliSidecar(parseWasmPath, parseBrotliPath); parseErr4 != nil {
			return releaseSummary{}, parseErr4
		}
		parseArtifacts["brotli"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, parseBrotliPath)
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	if strings.TrimSpace(parseConfig.budgetsPath) != "" {
		parseBudgets, parseErr5 := loadReleaseBudgets(parseConfig.budgetsPath)
		if parseErr5 != nil {
			return releaseSummary{}, parseErr5
		}
		if parseErr6 := assertReleaseBudgets(parseBudgets, parseArtifacts); parseErr6 != nil {
			return releaseSummary{}, parseErr6
		}
	}
	parseStartupConfig := parseConfig
	if parseStartupConfig.validateSmoke && parseStartupConfig.startupMeasure == "none" {
		parseStartupConfig.startupMeasure = "browser"
	}
	parseStartupReport, parseErr2 := releaseMeasureStartup(parseStartupConfig, parseArtifacts)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseStartupReport != nil {
		parseArtifacts["startup_report"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseStartupReport.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	parseManifestPath := filepath.Join(parseConfig.outDir, parseConfig.manifestName)
	parseDiffReport, parseErr2 := releaseWriteDiffReport(parseConfig.compareManifest, parseManifestPath, parseAttribution, parseArtifacts, parseConfig.outDir)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseDiffReport != nil {
		parseArtifacts["diff_report"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseDiffReport.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
	}
	parseManifestPayload := map[string]interface{}{
		"package": buildSummary.PackageDir,
		"profile": buildSummary.Profile.Name,
		"goos":    "js",
		"goarch":  "wasm",
		"flags": map[string]interface{}{
			"trimpath":             buildSummary.Profile.Trimpath,
			"ldflags":              buildSummary.Profile.Ldflags,
			"buildvcs":             firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"compression":          !parseConfig.skipCompression,
			"compressionPolicy":    parseConfig.compression,
			"compareManifest":      parseConfig.compareManifest,
			"postLinkOptimization": parseConfig.postLinkOpt,
			"sizeAttribution":      parseConfig.sizeAttribution,
			"startupMeasure":       parseConfig.startupMeasure,
			"validateSmoke":        parseConfig.validateSmoke,
			"gzip":                 isParseEmitGzip,
			"brotli":               isParseEmitBrotli,
		},
		"artifacts": parseArtifacts,
	}
	if parseOptimizer != nil {
		parseManifestPayload["optimizer"] = parseOptimizer
	}
	if parseAttribution != nil {
		parseManifestPayload["attribution"] = parseAttribution
	}
	if parseStartupReport != nil {
		parseManifestPayload["startup"] = parseStartupReport
	}
	if parseDiffReport != nil {
		parseManifestPayload["diff"] = parseDiffReport
	}
	parseEncodedManifest, parseErr2 := releaseMarshalIndent(parseManifestPayload, "", "  ")
	if parseErr2 != nil {
		return releaseSummary{}, fmt.Errorf("encode release manifest: %w", parseErr2)
	}
	parseEncodedManifest = append(parseEncodedManifest, '\n')
	if parseErr7 := os.WriteFile(parseManifestPath, parseEncodedManifest, 0644); parseErr7 != nil {
		return releaseSummary{}, fmt.Errorf("write release manifest: %w", parseErr7)
	}
	parseValidation, parseErr2 := releaseValidateSmoke(parseConfig, parseManifestPath, parseArtifacts, parseStartupReport)
	if parseErr2 != nil {
		return releaseSummary{}, parseErr2
	}
	if parseValidation != nil {
		parseArtifacts["validation_report"], parseErr2 = releaseArtifactRecordForPathFunc(parseConfig.outDir, filepath.Join(parseConfig.outDir, parseValidation.Path))
		if parseErr2 != nil {
			return releaseSummary{}, parseErr2
		}
		parseManifestPayload["validation"] = parseValidation
		parseManifestPayload["artifacts"] = parseArtifacts
		parseEncodedManifest, parseErr2 = releaseMarshalIndent(parseManifestPayload, "", "  ")
		if parseErr2 != nil {
			return releaseSummary{}, fmt.Errorf("encode release manifest: %w", parseErr2)
		}
		parseEncodedManifest = append(parseEncodedManifest, '\n')
		if parseErr8 := os.WriteFile(parseManifestPath, parseEncodedManifest, 0644); parseErr8 != nil {
			return releaseSummary{}, fmt.Errorf("write release manifest: %w", parseErr8)
		}
	}
	return releaseSummary{
		OK:           true,
		Profile:      buildSummary.Profile,
		AppPath:      buildSummary.AppPath,
		ProjectRoot:  buildSummary.ProjectRoot,
		PackageDir:   buildSummary.PackageDir,
		OutDir:       parseConfig.outDir,
		ManifestPath: parseManifestPath,
		Artifacts:    parseArtifacts,
		Flags: map[string]interface{}{
			"trimpath":             buildSummary.Profile.Trimpath,
			"ldflags":              buildSummary.Profile.Ldflags,
			"buildvcs":             firstNonEmpty(buildSummary.Profile.BuildVCS, "default"),
			"compression":          !parseConfig.skipCompression,
			"compressionPolicy":    parseConfig.compression,
			"compareManifest":      parseConfig.compareManifest,
			"postLinkOptimization": parseConfig.postLinkOpt,
			"sizeAttribution":      parseConfig.sizeAttribution,
			"startupMeasure":       parseConfig.startupMeasure,
			"validateSmoke":        parseConfig.validateSmoke,
			"gzip":                 isParseEmitGzip,
			"brotli":               isParseEmitBrotli,
		},
		Optimizer:   parseOptimizer,
		Attribution: parseAttribution,
		Diff:        parseDiffReport,
		Startup:     parseStartupReport,
		Validation:  parseValidation,
		Resolution:  cloneResolutionTrace(parseConfig.resolution),
	}, nil
}

func releaseApplyPostLinkOptimization(parseMode string, parseWasmPath string) (*releaseOptimizerRecord, error) {
	parseNormalizedMode, parseErr := normalizeReleasePostLinkOptimization(parseMode)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseNormalizedMode == "none" {
		return nil, nil
	}
	if parseNormalizedMode != "wasm-opt" {
		return nil, fmt.Errorf("unsupported release post-link optimization %q", parseNormalizedMode)
	}
	parseCommand, parseErr := resolveReleaseWasmOptCommand()
	if parseErr != nil {
		return nil, parseErr
	}
	if !parseCommand.Available {
		return nil, errors.New("post-link optimization requested but wasm-opt is unavailable; install wasm-opt and ensure it is on PATH")
	}

	parseOptimizedPath := parseWasmPath + ".opt"
	if parseErr2 := os.RemoveAll(parseOptimizedPath); parseErr2 != nil {
		return nil, fmt.Errorf("prepare optimized wasm artifact: %w", parseErr2)
	}
	parseArgs := append([]string{}, parseCommand.PrefixArgs...)
	parseArgs = append(parseArgs, parseWasmPath, "-Oz", "-o", parseOptimizedPath)
	if _, parseErr3 := releaseRunCommand(parseCommand.Command, parseArgs, filepath.Dir(parseWasmPath), os.Environ()); parseErr3 != nil {
		return nil, fmt.Errorf("run post-link optimizer %q: %w", parseNormalizedMode, parseErr3)
	}
	parseOptimizedBytes, parseErr := os.ReadFile(parseOptimizedPath)
	if parseErr != nil {
		return nil, fmt.Errorf("read optimized wasm artifact: %w", parseErr)
	}
	if parseErr4 := os.WriteFile(parseWasmPath, parseOptimizedBytes, 0644); parseErr4 != nil {
		return nil, fmt.Errorf("replace release wasm artifact with optimized output: %w", parseErr4)
	}
	if parseErr5 := os.Remove(parseOptimizedPath); parseErr5 != nil && !os.IsNotExist(parseErr5) {
		return nil, fmt.Errorf("cleanup optimized wasm artifact: %w", parseErr5)
	}
	return &releaseOptimizerRecord{
		Mode: parseNormalizedMode,
		Tool: parseCommand.Label,
		Args: parseArgs,
	}, nil
}

type releaseCommandInfo struct {
	Available  bool
	Command    string
	PrefixArgs []string
	Label      string
}

func resolveReleaseWasmOptCommand() (releaseCommandInfo, error) {
	if parsePath, parseErr := releaseLookPath("wasm-opt"); parseErr == nil {
		return releaseCommandInfo{
			Available: true,
			Command:   "wasm-opt",
			Label:     parsePath,
		}, nil
	}
	return releaseCommandInfo{}, nil
}

func releaseWriteSizeAttribution(parseMode string, parsePackageDir string, parseOutDir string) (*releaseAttributionRecord, error) {
	parseNormalizedMode, parseErr := normalizeReleaseSizeAttributionMode(parseMode)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseNormalizedMode == "none" {
		return nil, nil
	}
	if parseNormalizedMode != "packages" {
		return nil, fmt.Errorf("unsupported release size attribution mode %q", parseNormalizedMode)
	}

	parsePackages, parseErr := releaseCollectPackageSizeAttribution(parsePackageDir)
	if parseErr != nil {
		return nil, parseErr
	}
	parsePayload := map[string]interface{}{
		"mode":     parseNormalizedMode,
		"package":  parsePackageDir,
		"goos":     "js",
		"goarch":   "wasm",
		"packages": parsePackages,
	}
	parseEncoded, parseErr := releaseMarshalIndent(parsePayload, "", "  ")
	if parseErr != nil {
		return nil, fmt.Errorf("encode release size attribution: %w", parseErr)
	}
	parseEncoded = append(parseEncoded, '\n')
	parseFileName := "wasm-package-size-attribution.json"
	if parseErr2 := os.WriteFile(filepath.Join(parseOutDir, parseFileName), parseEncoded, 0644); parseErr2 != nil {
		return nil, fmt.Errorf("write release size attribution: %w", parseErr2)
	}
	return &releaseAttributionRecord{
		Mode:         parseNormalizedMode,
		Path:         parseFileName,
		PackageCount: len(parsePackages),
	}, nil
}

func releaseCollectPackageSizeAttribution(parsePackageDir string) ([]releasePackageSizeRecord, error) {
	parseOutput, parseErr := releaseRunCommand("go", []string{"list", "-deps", "-json", "-export", "."}, parsePackageDir, buildWasmGoEnv())
	if parseErr != nil {
		return nil, fmt.Errorf("collect release package attribution: %w", parseErr)
	}
	parseDecoder := json.NewDecoder(strings.NewReader(parseOutput))
	parsePackages := make([]releasePackageSizeRecord, 0, 64)
	for {
		var parsePkg releaseGoListPackage
		if parseErr2 := parseDecoder.Decode(&parsePkg); parseErr2 != nil {
			if errors.Is(parseErr2, io.EOF) {
				break
			}
			return nil, fmt.Errorf("decode release package attribution: %w", parseErr2)
		}
		if strings.TrimSpace(parsePkg.ImportPath) == "" {
			continue
		}
		parseRecord, parseErr3 := releaseBuildPackageSizeRecord(parsePkg)
		if parseErr3 != nil {
			return nil, parseErr3
		}
		if parseRecord.ArchiveBytes == 0 && parseRecord.SourceBytes == 0 && parseRecord.FileCount == 0 {
			continue
		}
		parsePackages = append(parsePackages, parseRecord)
	}
	sort.Slice(parsePackages, func(parseI int, parseJ int) bool {
		if parsePackages[parseI].ArchiveBytes != parsePackages[parseJ].ArchiveBytes {
			return parsePackages[parseI].ArchiveBytes > parsePackages[parseJ].ArchiveBytes
		}
		if parsePackages[parseI].SourceBytes != parsePackages[parseJ].SourceBytes {
			return parsePackages[parseI].SourceBytes > parsePackages[parseJ].SourceBytes
		}
		return parsePackages[parseI].ImportPath < parsePackages[parseJ].ImportPath
	})
	return parsePackages, nil
}

func releaseBuildPackageSizeRecord(parsePkg releaseGoListPackage) (releasePackageSizeRecord, error) {
	parseRecord := releasePackageSizeRecord{
		ImportPath: strings.TrimSpace(parsePkg.ImportPath),
		Dir:        strings.TrimSpace(parsePkg.Dir),
	}
	if strings.TrimSpace(parsePkg.Export) != "" {
		parseInfo, parseErr := os.Stat(parsePkg.Export)
		if parseErr != nil {
			return releasePackageSizeRecord{}, fmt.Errorf("inspect export archive for %s: %w", parsePkg.ImportPath, parseErr)
		}
		parseRecord.ArchiveBytes = parseInfo.Size()
	}
	parseSourceFiles := map[string]struct{}{}
	for _, parseFile := range append(
		append(append(append(append(append(append(append(append(append([]string{}, parsePkg.GoFiles...), parsePkg.CgoFiles...), parsePkg.CFiles...), parsePkg.CXXFiles...), parsePkg.MFiles...), parsePkg.HFiles...), parsePkg.FFiles...), parsePkg.SFiles...), parsePkg.SysoFiles...),
		parsePkg.EmbedFiles...,
	) {
		parseTrimmed := strings.TrimSpace(parseFile)
		if parseTrimmed == "" || strings.TrimSpace(parsePkg.Dir) == "" {
			continue
		}
		parseSourceFiles[filepath.Join(parsePkg.Dir, filepath.FromSlash(parseTrimmed))] = struct{}{}
	}
	parseRecord.FileCount = len(parseSourceFiles)
	for parseFile2 := range parseSourceFiles {
		parseInfo2, parseErr2 := os.Stat(parseFile2)
		if parseErr2 != nil {
			return releasePackageSizeRecord{}, fmt.Errorf("inspect source file for %s: %w", parsePkg.ImportPath, parseErr2)
		}
		parseRecord.SourceBytes += parseInfo2.Size()
	}
	return parseRecord, nil
}

func measureReleaseStartup(parseConfig releaseConfig, parseArtifacts map[string]releaseArtifactRecord) (*releaseStartupRecord, error) {
	parseMode, parseErr := normalizeReleaseStartupMeasureMode(parseConfig.startupMeasure)
	if parseErr != nil {
		return nil, parseErr
	}
	if parseMode == "none" {
		return nil, nil
	}
	parseWasmExecPath, parseErr := releaseResolveWasmExec()
	if parseErr != nil {
		return nil, fmt.Errorf("resolve wasm_exec.js for startup measurement: %w", parseErr)
	}
	parseReleaseWasmPath := filepath.Join(parseConfig.outDir, parseConfig.binaryName)
	parseProbeURL, parseTransportEncoding, parseShutdown, parseErr := startReleaseStartupProbeServer(parseConfig.outDir, parseConfig.binaryName, parseWasmExecPath, parseArtifacts, parseConfig.startupTimeoutMs)
	if parseErr != nil {
		return nil, parseErr
	}
	defer parseShutdown()

	parseReportPath := filepath.Join(parseConfig.outDir, "wasm-startup-report.json")
	if parseErr2 := releaseRunStartupProbeWithPlaywright(parseProbeURL, parseReportPath, parseConfig.startupTimeoutMs); parseErr2 != nil {
		return nil, fmt.Errorf("measure release startup: %w", parseErr2)
	}
	if !fileExists(parseReportPath) {
		return nil, fmt.Errorf("startup measurement did not produce a report for %s", parseReleaseWasmPath)
	}
	return &releaseStartupRecord{
		Mode:              parseMode,
		Path:              "wasm-startup-report.json",
		ProbeURL:          parseProbeURL,
		TransportEncoding: parseTransportEncoding,
	}, nil
}

func runReleaseStartupProbeWithPlaywright(parseProbeURL string, parseReportPath string, parseTimeoutMs int) error {
	parseRunOptions := &playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}
	if parseErr := releasePlaywrightInstall(parseRunOptions); parseErr != nil {
		return fmt.Errorf("install playwright-go runtime: %w", parseErr)
	}
	parsePw, parseErr2 := releasePlaywrightRun(parseRunOptions)
	if parseErr2 != nil {
		return fmt.Errorf("run playwright-go runtime: %w", parseErr2)
	}
	defer func() {
		_ = parsePw.Stop()
	}()
	parseBrowser, parseErr2 := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseErr2 != nil {
		return fmt.Errorf("launch chromium: %w", parseErr2)
	}
	defer func() {
		_ = parseBrowser.Close()
	}()
	parsePage, parseErr2 := parseBrowser.NewPage()
	if parseErr2 != nil {
		return fmt.Errorf("create probe page: %w", parseErr2)
	}
	if parseTimeoutMs > 0 {
		parsePage.SetDefaultTimeout(float64(parseTimeoutMs))
	}
	parseResponse, parseErr2 := parsePage.Goto(parseProbeURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr2 != nil {
		return fmt.Errorf("open startup probe page: %w", parseErr2)
	}
	if parseResponse == nil {
		return errors.New("startup probe navigation returned no response")
	}
	if parseResponse.Status() >= 400 {
		return fmt.Errorf("startup probe navigation failed: %d", parseResponse.Status())
	}
	if _, parseErr3 := parsePage.WaitForFunction("() => window.__gwcStartupProbe && window.__gwcStartupProbe.readyMs !== null", nil); parseErr3 != nil {
		return fmt.Errorf("wait for ready probe: %w", parseErr3)
	}
	if parseErr4 := parsePage.Locator("#__gwc_probe_button").Click(); parseErr4 != nil {
		return fmt.Errorf("trigger startup probe interaction: %w", parseErr4)
	}
	if _, parseErr5 := parsePage.WaitForFunction("() => window.__gwcStartupProbe && window.__gwcStartupProbe.interactionMs !== null", nil); parseErr5 != nil {
		return fmt.Errorf("wait for interaction probe: %w", parseErr5)
	}
	parseReport, parseErr2 := parsePage.Evaluate(`() => ({
		userAgent: navigator.userAgent,
		startup: window.__gwcStartupProbe || null,
		timestamp: new Date().toISOString()
	})`)
	if parseErr2 != nil {
		return fmt.Errorf("collect startup probe report: %w", parseErr2)
	}
	parseEncoded, parseErr2 := json.MarshalIndent(parseReport, "", "  ")
	if parseErr2 != nil {
		return fmt.Errorf("encode startup probe report: %w", parseErr2)
	}
	parseEncoded = append(parseEncoded, '\n')
	if parseErr6 := os.WriteFile(parseReportPath, parseEncoded, 0644); parseErr6 != nil {
		return fmt.Errorf("write startup probe report: %w", parseErr6)
	}
	return nil
}

func validateReleaseSmoke(parseConfig releaseConfig, parseManifestPath string, parseArtifacts map[string]releaseArtifactRecord, parseStartupReport *releaseStartupRecord) (*releaseValidationRecord, error) {
	if !parseConfig.validateSmoke {
		return nil, nil
	}
	parseManifestBytes, parseErr := os.ReadFile(parseManifestPath)
	if parseErr != nil {
		return nil, fmt.Errorf("read release manifest for smoke validation: %w", parseErr)
	}
	parseManifest, parseErr := pwa.ParseWasmReleaseManifestJSON(parseManifestBytes)
	if parseErr != nil {
		return nil, fmt.Errorf("validate release manifest for smoke validation: %w", parseErr)
	}
	parseChecks := []string{"manifest parses as a valid js/wasm release record"}
	for parseName, parseArtifact := range parseManifest.Artifacts {
		parseArtifactPath := filepath.Join(parseConfig.outDir, filepath.FromSlash(parseArtifact.Path))
		parseRecord, parseErr2 := releaseArtifactRecordForPathFunc(parseConfig.outDir, parseArtifactPath)
		if parseErr2 != nil {
			return nil, fmt.Errorf("validate release artifact %q: %w", parseName, parseErr2)
		}
		if parseRecord.Bytes != parseArtifact.Bytes || !strings.EqualFold(parseRecord.SHA256, parseArtifact.SHA256) {
			return nil, fmt.Errorf("validate release artifact %q: manifest record does not match on-disk artifact", parseName)
		}
		parseChecks = append(parseChecks, fmt.Sprintf("artifact %s exists and matches manifest bytes and sha256", parseName))
	}
	if parseStartupReport == nil {
		return nil, errors.New("release smoke validation requires a startup probe result")
	}
	parseWasmContentType, parseWasmContentEncoding, parseErr := releaseSmokeFetchWasmHeaders(parseConfig.outDir, parseConfig.binaryName, parseArtifacts)
	if parseErr != nil {
		return nil, parseErr
	}
	parseChecks = append(parseChecks, "wasm asset serves with application/wasm content type")
	parseChecks = append(parseChecks, "boot-time startup probe completed")
	parseRecord2 := &releaseValidationRecord{
		Path:                "wasm-release-validation.json",
		Checks:              parseChecks,
		StartupReportPath:   parseStartupReport.Path,
		WasmContentType:     parseWasmContentType,
		WasmContentEncoding: parseWasmContentEncoding,
	}
	parseEncoded, parseErr := releaseMarshalIndent(parseRecord2, "", "  ")
	if parseErr != nil {
		return nil, fmt.Errorf("encode release validation report: %w", parseErr)
	}
	parseEncoded = append(parseEncoded, '\n')
	if parseErr3 := os.WriteFile(filepath.Join(parseConfig.outDir, parseRecord2.Path), parseEncoded, 0644); parseErr3 != nil {
		return nil, fmt.Errorf("write release validation report: %w", parseErr3)
	}
	return parseRecord2, nil
}

func releaseSmokeFetchWasmHeaders(parseOutDir string, parseBinaryName string, parseArtifacts map[string]releaseArtifactRecord) (string, string, error) {
	parseListener, parseErr := net.Listen("tcp", joinHostPort(defaultHost, "0"))
	if parseErr != nil {
		return "", "", fmt.Errorf("release smoke validation listen: %w", parseErr)
	}
	defer parseListener.Close()
	parseMux := http.NewServeMux()
	parseTransportEncoding := releaseStartupTransportEncoding(parseArtifacts)
	parseMux.HandleFunc("/"+parseBinaryName, func(parseW http.ResponseWriter, parseR *http.Request) {
		releaseServeStartupWasm(parseW, parseR, parseOutDir, parseBinaryName, parseTransportEncoding)
	})
	parseServer := &http.Server{Handler: parseMux}
	defer parseServer.Close()
	go func() {
		_ = parseServer.Serve(parseListener)
	}()
	parseClient := &http.Client{
		Transport: &http.Transport{DisableCompression: true},
		Timeout:   15 * time.Second,
	}
	parseResp, parseErr := parseClient.Get("http://" + parseListener.Addr().String() + "/" + strings.TrimLeft(parseBinaryName, "/"))
	if parseErr != nil {
		return "", "", fmt.Errorf("release smoke validation fetch wasm asset: %w", parseErr)
	}
	defer parseResp.Body.Close()
	if parseResp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("release smoke validation expected wasm asset status 200, got %d", parseResp.StatusCode)
	}
	parseContentType := strings.TrimSpace(parseResp.Header.Get("Content-Type"))
	if !strings.Contains(strings.ToLower(parseContentType), "application/wasm") {
		return parseContentType, strings.TrimSpace(parseResp.Header.Get("Content-Encoding")), fmt.Errorf("release smoke validation expected application/wasm content type, got %q", parseContentType)
	}
	return parseContentType, strings.TrimSpace(parseResp.Header.Get("Content-Encoding")), nil
}

func startReleaseStartupProbeServer(parseOutDir string, parseBinaryName string, parseWasmExecPath string, parseArtifacts map[string]releaseArtifactRecord, parseTimeoutMs int) (string, string, func(), error) {
	parseListener, parseErr := net.Listen("tcp", joinHostPort(defaultHost, "0"))
	if parseErr != nil {
		return "", "", nil, fmt.Errorf("listen for startup measurement probe: %w", parseErr)
	}
	parseTransportEncoding := releaseStartupTransportEncoding(parseArtifacts)
	parseMux := http.NewServeMux()
	parseProbeHTML := renderReleaseStartupProbeHTML(parseBinaryName, parseTransportEncoding, parseTimeoutMs)
	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseR *http.Request) {
		switch parseR.URL.Path {
		case "/":
			http.Redirect(parseW, parseR, "/__gwc/startup-probe.html", http.StatusTemporaryRedirect)
			return
		case "/__gwc/startup-probe.html":
			parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
			parseW.Header().Set("Cache-Control", "no-store")
			_, _ = parseW.Write([]byte(parseProbeHTML))
			return
		case "/__gwc/wasm_exec.js":
			http.ServeFile(parseW, parseR, parseWasmExecPath)
			return
		case "/" + parseBinaryName:
			releaseServeStartupWasm(parseW, parseR, parseOutDir, parseBinaryName, parseTransportEncoding)
			return
		default:
			applyDevHeaders(parseW, parseR)
			http.FileServer(http.Dir(parseOutDir)).ServeHTTP(parseW, parseR)
			return
		}
	})
	parseServer := &http.Server{Handler: parseMux}
	go func() {
		_ = parseServer.Serve(parseListener)
	}()
	parseShutdown := func() {
		_ = parseServer.Close()
		_ = parseListener.Close()
	}
	return "http://" + parseListener.Addr().String() + "/__gwc/startup-probe.html", parseTransportEncoding, parseShutdown, nil
}

func releaseStartupTransportEncoding(parseArtifacts map[string]releaseArtifactRecord) string {
	if _, parseOk := parseArtifacts["gzip"]; parseOk {
		return "gzip"
	}
	if _, parseOk2 := parseArtifacts["brotli"]; parseOk2 {
		return "br"
	}
	return "identity"
}

func releaseServeStartupWasm(parseW http.ResponseWriter, parseR *http.Request, parseOutDir string, parseBinaryName string, parseTransportEncoding string) {
	applyDevHeaders(parseW, parseR)
	parseW.Header().Set("Cache-Control", "no-store")
	switch parseTransportEncoding {
	case "gzip":
		parseW.Header().Set("Content-Encoding", "gzip")
		http.ServeFile(parseW, parseR, filepath.Join(parseOutDir, parseBinaryName+".gz"))
	case "br":
		parseW.Header().Set("Content-Encoding", "br")
		http.ServeFile(parseW, parseR, filepath.Join(parseOutDir, parseBinaryName+".br"))
	default:
		http.ServeFile(parseW, parseR, filepath.Join(parseOutDir, parseBinaryName))
	}
}

func renderReleaseStartupProbeHTML(parseBinaryName string, parseTransportEncoding string, parseTimeoutMs int) string {
	parseProbeGzipPath := "null"
	if parseTransportEncoding == "gzip" {
		parseProbeGzipPath = jsStringLiteral("/" + parseBinaryName + ".gz")
	}
	return "<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n  <meta charset=\"utf-8\">\n  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n  <title>GWC Release Startup Probe</title>\n  <script src=\"/__gwc/wasm_exec.js\"></script>\n</head>\n<body>\n  <div id=\"app\"></div>\n  <button id=\"__gwc_probe_button\" style=\"position:fixed;top:12px;right:12px;z-index:2147483647\">Probe interaction</button>\n  <script>\n" +
		"(() => {\n" +
		"  const startup = window.__gwcStartupProbe = {\n" +
		"    startedAt: performance.now(),\n" +
		"    transportEncoding: " + jsStringLiteral(parseTransportEncoding) + ",\n" +
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
		"  setTimeout(() => markReady('timeout'), " + fmt.Sprintf("%d", parseTimeoutMs) + ");\n" +
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
		"  const gzipProbePath = " + parseProbeGzipPath + ";\n" +
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
		"  WebAssembly.instantiateStreaming(fetch(" + jsStringLiteral("/"+parseBinaryName) + ", { cache: 'no-store' }), go.importObject)\n" +
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

func releaseWriteDiffReport(parseCompareManifest string, parseManifestPath string, parseAttribution *releaseAttributionRecord, parseArtifacts map[string]releaseArtifactRecord, parseOutDir string) (*releaseDiffArtifactRecord, error) {
	if strings.TrimSpace(parseCompareManifest) == "" {
		return nil, nil
	}
	parseBaseline, parseErr := releaseReadManifestSnapshot(parseCompareManifest)
	if parseErr != nil {
		return nil, parseErr
	}
	parseArtifactChanges := releaseCompareArtifactRecords(parseBaseline.Artifacts, parseArtifacts)
	parseLikelyCulprits, parseErr := releaseComparePackageAttributionRecords(parseCompareManifest, parseBaseline.Attribution, parseOutDir, parseAttribution)
	if parseErr != nil {
		return nil, parseErr
	}
	parseFileName := "wasm-release-size-diff.json"
	parsePayload := map[string]interface{}{
		"baselineManifestPath": baselinePathForJSON(parseCompareManifest),
		"currentManifestPath":  parseManifestPath,
		"artifactChanges":      parseArtifactChanges,
		"likelyCulprits":       parseLikelyCulprits,
	}
	parseEncoded, parseErr := releaseMarshalIndent(parsePayload, "", "  ")
	if parseErr != nil {
		return nil, fmt.Errorf("encode release diff report: %w", parseErr)
	}
	parseEncoded = append(parseEncoded, '\n')
	if parseErr2 := os.WriteFile(filepath.Join(parseOutDir, parseFileName), parseEncoded, 0644); parseErr2 != nil {
		return nil, fmt.Errorf("write release diff report: %w", parseErr2)
	}
	return &releaseDiffArtifactRecord{
		Path:                 parseFileName,
		BaselineManifestPath: parseCompareManifest,
		ArtifactChanges:      parseArtifactChanges,
		LikelyCulprits:       parseLikelyCulprits,
	}, nil
}

func baselinePathForJSON(parsePath string) string {
	return parsePath
}

func releaseReadManifestSnapshot(parsePath string) (releaseManifestSnapshot, error) {
	parseManifestBytes, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return releaseManifestSnapshot{}, fmt.Errorf("read compare manifest: %w", parseErr)
	}
	var parseManifest releaseManifestSnapshot
	if parseErr2 := json.Unmarshal(parseManifestBytes, &parseManifest); parseErr2 != nil {
		return releaseManifestSnapshot{}, fmt.Errorf("parse compare manifest: %w", parseErr2)
	}
	if strings.TrimSpace(parseManifest.Package) == "" {
		return releaseManifestSnapshot{}, errors.New("compare manifest is missing package")
	}
	if parseManifest.GOOS != "js" || parseManifest.GOARCH != "wasm" {
		return releaseManifestSnapshot{}, errors.New("compare manifest must describe a js/wasm release")
	}
	if len(parseManifest.Artifacts) == 0 {
		return releaseManifestSnapshot{}, errors.New("compare manifest is missing artifacts")
	}
	return parseManifest, nil
}

func releaseCompareArtifactRecords(parseBaseline map[string]releaseArtifactRecord, parseCurrent map[string]releaseArtifactRecord) []releaseArtifactDiffRecord {
	parseKeys := map[string]struct{}{}
	for parseKey := range parseBaseline {
		parseKeys[parseKey] = struct{}{}
	}
	for parseKey2 := range parseCurrent {
		parseKeys[parseKey2] = struct{}{}
	}
	parseNames := make([]string, 0, len(parseKeys))
	for parseKey3 := range parseKeys {
		parseNames = append(parseNames, parseKey3)
	}
	sort.Strings(parseNames)
	parseChanges := make([]releaseArtifactDiffRecord, 0, len(parseNames))
	for _, parseName := range parseNames {
		parseBaselineArtifact, hasBaseline := parseBaseline[parseName]
		parseCurrentArtifact, hasCurrent := parseCurrent[parseName]
		parseRecord := releaseArtifactDiffRecord{Name: parseName}
		if hasBaseline {
			parseRecord.BaselinePath = parseBaselineArtifact.Path
			parseRecord.BaselineBytes = int64ptr(parseBaselineArtifact.Bytes)
		}
		if hasCurrent {
			parseRecord.CurrentPath = parseCurrentArtifact.Path
			parseRecord.CurrentBytes = int64ptr(parseCurrentArtifact.Bytes)
		}
		switch {
		case hasBaseline && hasCurrent:
			parseDelta := parseCurrentArtifact.Bytes - parseBaselineArtifact.Bytes
			parseRecord.DeltaBytes = int64ptr(parseDelta)
			parseRecord.DeltaPercent = releasePercentDeltaPointer(parseBaselineArtifact.Bytes, parseCurrentArtifact.Bytes)
			if parseDelta > 0 {
				parseRecord.Status = "grew"
			} else if parseDelta < 0 {
				parseRecord.Status = "shrank"
			} else {
				parseRecord.Status = "unchanged"
			}
		case hasCurrent:
			parseRecord.Status = "added"
		default:
			parseRecord.Status = "removed"
		}
		parseChanges = append(parseChanges, parseRecord)
	}
	return parseChanges
}

func releaseComparePackageAttributionRecords(parseBaselineManifestPath string, parseBaselineAttribution *releaseAttributionRecord, parseCurrentOutDir string, parseCurrentAttribution *releaseAttributionRecord) ([]releasePackageDiffRecord, error) {
	if parseBaselineAttribution == nil || parseCurrentAttribution == nil {
		return nil, nil
	}
	parseBaselinePackages, parseErr := releaseReadPackageAttributionSnapshot(filepath.Join(filepath.Dir(parseBaselineManifestPath), filepath.FromSlash(parseBaselineAttribution.Path)))
	if parseErr != nil {
		return nil, parseErr
	}
	parseCurrentPackages, parseErr := releaseReadPackageAttributionSnapshot(filepath.Join(parseCurrentOutDir, filepath.FromSlash(parseCurrentAttribution.Path)))
	if parseErr != nil {
		return nil, parseErr
	}
	parseBaselineMap := map[string]releasePackageSizeRecord{}
	for _, parseRecord := range parseBaselinePackages.Packages {
		parseBaselineMap[parseRecord.ImportPath] = parseRecord
	}
	parseCurrentMap := map[string]releasePackageSizeRecord{}
	for _, parseRecord2 := range parseCurrentPackages.Packages {
		parseCurrentMap[parseRecord2.ImportPath] = parseRecord2
	}
	parseKeys := map[string]struct{}{}
	for parseKey := range parseBaselineMap {
		parseKeys[parseKey] = struct{}{}
	}
	for parseKey2 := range parseCurrentMap {
		parseKeys[parseKey2] = struct{}{}
	}
	parseDeltas := make([]releasePackageDiffRecord, 0, len(parseKeys))
	for parseImportPath := range parseKeys {
		parseBaselineRecord, hasBaseline := parseBaselineMap[parseImportPath]
		parseCurrentRecord, hasCurrent := parseCurrentMap[parseImportPath]
		parseRecord3 := releasePackageDiffRecord{ImportPath: parseImportPath}
		if hasBaseline {
			parseRecord3.BaselineArchiveBytes = int64ptr(parseBaselineRecord.ArchiveBytes)
			parseRecord3.BaselineSourceBytes = int64ptr(parseBaselineRecord.SourceBytes)
		}
		if hasCurrent {
			parseRecord3.CurrentArchiveBytes = int64ptr(parseCurrentRecord.ArchiveBytes)
			parseRecord3.CurrentSourceBytes = int64ptr(parseCurrentRecord.SourceBytes)
		}
		switch {
		case hasBaseline && hasCurrent:
			parseArchiveDelta := parseCurrentRecord.ArchiveBytes - parseBaselineRecord.ArchiveBytes
			parseSourceDelta := parseCurrentRecord.SourceBytes - parseBaselineRecord.SourceBytes
			parseRecord3.ArchiveDeltaBytes = int64ptr(parseArchiveDelta)
			parseRecord3.ArchiveDeltaPercent = releasePercentDeltaPointer(parseBaselineRecord.ArchiveBytes, parseCurrentRecord.ArchiveBytes)
			parseRecord3.SourceDeltaBytes = int64ptr(parseSourceDelta)
			if parseArchiveDelta > 0 {
				parseRecord3.Status = "grew"
			} else if parseArchiveDelta < 0 {
				parseRecord3.Status = "shrank"
			} else if parseSourceDelta != 0 {
				parseRecord3.Status = "source-only-change"
			} else {
				parseRecord3.Status = "unchanged"
			}
		case hasCurrent:
			parseRecord3.ArchiveDeltaBytes = int64ptr(parseCurrentRecord.ArchiveBytes)
			parseRecord3.SourceDeltaBytes = int64ptr(parseCurrentRecord.SourceBytes)
			parseRecord3.Status = "added"
		default:
			parseArchiveDelta2 := -parseBaselineRecord.ArchiveBytes
			parseSourceDelta2 := -parseBaselineRecord.SourceBytes
			parseRecord3.ArchiveDeltaBytes = int64ptr(parseArchiveDelta2)
			parseRecord3.SourceDeltaBytes = int64ptr(parseSourceDelta2)
			parseRecord3.Status = "removed"
		}
		if parseRecord3.ArchiveDeltaBytes != nil && *parseRecord3.ArchiveDeltaBytes > 0 {
			parseDeltas = append(parseDeltas, parseRecord3)
		}
	}
	sort.Slice(parseDeltas, func(parseI int, parseJ int) bool {
		parseLeft := derefInt64(parseDeltas[parseI].ArchiveDeltaBytes)
		parseRight := derefInt64(parseDeltas[parseJ].ArchiveDeltaBytes)
		if parseLeft != parseRight {
			return parseLeft > parseRight
		}
		parseLeftSource := derefInt64(parseDeltas[parseI].SourceDeltaBytes)
		parseRightSource := derefInt64(parseDeltas[parseJ].SourceDeltaBytes)
		if parseLeftSource != parseRightSource {
			return parseLeftSource > parseRightSource
		}
		return parseDeltas[parseI].ImportPath < parseDeltas[parseJ].ImportPath
	})
	if len(parseDeltas) > 10 {
		parseDeltas = parseDeltas[:10]
	}
	if len(parseDeltas) == 0 {
		return nil, nil
	}
	return parseDeltas, nil
}

func releaseReadPackageAttributionSnapshot(parsePath string) (releasePackageAttributionSnapshot, error) {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return releasePackageAttributionSnapshot{}, fmt.Errorf("read package attribution artifact: %w", parseErr)
	}
	var parseSnapshot releasePackageAttributionSnapshot
	if parseErr2 := json.Unmarshal(parseData, &parseSnapshot); parseErr2 != nil {
		return releasePackageAttributionSnapshot{}, fmt.Errorf("parse package attribution artifact: %w", parseErr2)
	}
	return parseSnapshot, nil
}

func releasePercentDeltaPointer(parseBaseline int64, parseCurrent int64) *float64 {
	if parseBaseline == 0 {
		return nil
	}
	parseDelta := float64(parseCurrent-parseBaseline) / float64(parseBaseline) * 100
	return &parseDelta
}

func int64ptr(parseValue int64) *int64 {
	return &parseValue
}

func derefInt64(parseValue *int64) int64 {
	if parseValue == nil {
		return 0
	}
	return *parseValue
}

func releaseArtifactRecordForPath(parseBaseDir string, parseArtifactPath string) (releaseArtifactRecord, error) {
	parseArtifactBytes, parseErr := os.ReadFile(parseArtifactPath)
	if parseErr != nil {
		return releaseArtifactRecord{}, fmt.Errorf("read release artifact: %w", parseErr)
	}
	parseArtifactInfo, parseErr := os.Stat(parseArtifactPath)
	if parseErr != nil {
		return releaseArtifactRecord{}, fmt.Errorf("inspect release artifact: %w", parseErr)
	}
	parseRelPath, parseErr := filepath.Rel(parseBaseDir, parseArtifactPath)
	if parseErr != nil {
		return releaseArtifactRecord{}, fmt.Errorf("resolve release artifact path: %w", parseErr)
	}
	parseHash := sha256.Sum256(parseArtifactBytes)
	return releaseArtifactRecord{
		Path:   filepath.ToSlash(parseRelPath),
		Bytes:  parseArtifactInfo.Size(),
		SHA256: fmt.Sprintf("%x", parseHash[:]),
	}, nil
}

func writeGzipSidecar(parseSourcePath string, parseTargetPath string) error {
	parseInputBytes, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		return fmt.Errorf("read source artifact for gzip: %w", parseErr)
	}
	parseOutputFile, parseErr := os.Create(parseTargetPath)
	if parseErr != nil {
		return fmt.Errorf("create gzip sidecar: %w", parseErr)
	}
	defer parseOutputFile.Close()
	parseGzipWriter, parseErr := gzip.NewWriterLevel(parseOutputFile, gzip.BestCompression)
	if parseErr != nil {
		return fmt.Errorf("create gzip writer: %w", parseErr)
	}
	if _, parseErr2 := parseGzipWriter.Write(parseInputBytes); parseErr2 != nil {
		parseGzipWriter.Close()
		return fmt.Errorf("write gzip sidecar: %w", parseErr2)
	}
	if parseErr3 := parseGzipWriter.Close(); parseErr3 != nil {
		return fmt.Errorf("finalize gzip sidecar: %w", parseErr3)
	}
	return nil
}

func writeBrotliSidecar(parseSourcePath string, parseTargetPath string) error {
	parseInputBytes, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		return fmt.Errorf("read source artifact for brotli: %w", parseErr)
	}
	parseOutputFile, parseErr := os.Create(parseTargetPath)
	if parseErr != nil {
		return fmt.Errorf("create brotli sidecar: %w", parseErr)
	}
	defer parseOutputFile.Close()
	parseBrotliWriter := brotli.NewWriterLevel(parseOutputFile, brotli.BestCompression)
	if _, parseErr2 := parseBrotliWriter.Write(parseInputBytes); parseErr2 != nil {
		parseBrotliWriter.Close()
		return fmt.Errorf("write brotli sidecar: %w", parseErr2)
	}
	if parseErr3 := parseBrotliWriter.Close(); parseErr3 != nil {
		return fmt.Errorf("finalize brotli sidecar: %w", parseErr3)
	}
	return nil
}

func loadReleaseBudgets(parsePath string) (map[string]int64, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, fmt.Errorf("read budgets file: %w", parseErr)
	}
	parseRaw := map[string]interface{}{}
	if parseErr2 := json.Unmarshal(parseContent, &parseRaw); parseErr2 != nil {
		return nil, fmt.Errorf("parse budgets file: %w", parseErr2)
	}
	parseBudgets := map[string]int64{}
	for parseKey, parseValue := range parseRaw {
		parseNumber, parseOk := parseValue.(float64)
		if !parseOk {
			return nil, fmt.Errorf("budget %q must be numeric", parseKey)
		}
		parseBudgets[parseKey] = int64(parseNumber)
	}
	return parseBudgets, nil
}

func assertReleaseBudgets(parseBudgets map[string]int64, parseArtifacts map[string]releaseArtifactRecord) error {
	parseChecks := []struct {
		budgetKey string
		artifact  string
		label     string
	}{
		{budgetKey: "raw_bytes", artifact: "wasm", label: "raw wasm"},
		{budgetKey: "gzip_bytes", artifact: "gzip", label: "gzip sidecar"},
		{budgetKey: "brotli_bytes", artifact: "brotli", label: "brotli sidecar"},
	}
	for _, parseCheck := range parseChecks {
		parseLimit, parseOk := parseBudgets[parseCheck.budgetKey]
		if !parseOk {
			continue
		}
		parseArtifact, parseOk := parseArtifacts[parseCheck.artifact]
		if !parseOk {
			continue
		}
		if parseArtifact.Bytes > parseLimit {
			return fmt.Errorf("artifact budget exceeded for %s: %d bytes > %d bytes", parseCheck.label, parseArtifact.Bytes, parseLimit)
		}
	}
	return nil
}

func printBuildSummary(parseSummary buildSummary) {
	fmt.Println("GWC build")
	fmt.Printf("  profile:      %s\n", parseSummary.Profile.Name)
	fmt.Printf("  app:          %s\n", parseSummary.AppPath)
	fmt.Printf("  project root: %s\n", parseSummary.ProjectRoot)
	fmt.Printf("  package dir:  %s\n", parseSummary.PackageDir)
	fmt.Printf("  output:       %s\n", parseSummary.OutputPath)
	fmt.Printf("  bytes:        %d\n", parseSummary.Bytes)
	fmt.Printf("  sha256:       %s\n", parseSummary.SHA256)
	fmt.Printf("  trimpath:     %t\n", parseSummary.Profile.Trimpath)
	fmt.Printf("  ldflags:      %s\n", firstNonEmpty(parseSummary.Profile.Ldflags, "<none>"))
	fmt.Printf("  buildvcs:     %s\n", firstNonEmpty(parseSummary.Profile.BuildVCS, "default"))
	printResolutionTrace(parseSummary.Resolution, []string{"app", "root", "output", "profile"}, "  ")
}

func printReleaseSummary(parseSummary releaseSummary) {
	fmt.Println("GWC release")
	fmt.Printf("  profile:      %s\n", parseSummary.Profile.Name)
	fmt.Printf("  app:          %s\n", parseSummary.AppPath)
	fmt.Printf("  project root: %s\n", parseSummary.ProjectRoot)
	fmt.Printf("  package dir:  %s\n", parseSummary.PackageDir)
	fmt.Printf("  out dir:      %s\n", parseSummary.OutDir)
	fmt.Printf("  manifest:     %s\n", parseSummary.ManifestPath)
	if parseSummary.Optimizer != nil {
		fmt.Printf("  optimizer:    %s (%s)\n", parseSummary.Optimizer.Mode, parseSummary.Optimizer.Tool)
	}
	if parseSummary.Attribution != nil {
		fmt.Printf("  attribution:  %s (%s, %d packages)\n", parseSummary.Attribution.Path, parseSummary.Attribution.Mode, parseSummary.Attribution.PackageCount)
	}
	if parseSummary.Startup != nil {
		fmt.Printf("  startup:      %s (%s via %s)\n", parseSummary.Startup.Path, parseSummary.Startup.Mode, parseSummary.Startup.TransportEncoding)
	}
	if parseSummary.Validation != nil {
		fmt.Printf("  validation:   %s (%s)\n", parseSummary.Validation.Path, parseSummary.Validation.WasmContentType)
	}
	if parseSummary.Diff != nil {
		fmt.Printf("  diff:         %s (baseline %s)\n", parseSummary.Diff.Path, parseSummary.Diff.BaselineManifestPath)
	}
	parseKeys := make([]string, 0, len(parseSummary.Artifacts))
	for parseKey := range parseSummary.Artifacts {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	for _, parseKey2 := range parseKeys {
		parseArtifact := parseSummary.Artifacts[parseKey2]
		fmt.Printf("  artifact[%s]: %s (%d bytes)\n", parseKey2, parseArtifact.Path, parseArtifact.Bytes)
	}
	printResolutionTrace(parseSummary.Resolution, []string{"app", "root", "output", "profile"}, "  ")
}

func printVerifySummary(parseSummary verifySummary) {
	fmt.Println("GWC verify")
	fmt.Printf("  app:          %s\n", parseSummary.AppPath)
	fmt.Printf("  project root: %s\n", parseSummary.ProjectRoot)
	if parseSummary.Tests.Ran {
		fmt.Printf("  tests:        %s %s\n", parseSummary.Tests.Command, parseSummary.Tests.PackagePattern)
	} else {
		fmt.Println("  tests:        skipped")
	}
	fmt.Printf("  build:        %s -> %s\n", parseSummary.Build.Profile.Name, parseSummary.Build.OutputPath)
	if parseSummary.Audit != nil {
		parseAuditStatus := "PASS"
		if !parseSummary.Audit.OK {
			parseAuditStatus = "FAIL"
		}
		fmt.Printf("  audit[%s]: %s", parseSummary.Audit.Mode, parseAuditStatus)
		if parseSummary.Audit.Policy != "" {
			fmt.Printf(" (policy: %s)", parseSummary.Audit.Policy)
		}
		if strings.TrimSpace(parseSummary.AuditMinSeverity) != "" {
			fmt.Printf(" (min severity: %s)", parseSummary.AuditMinSeverity)
		}
		fmt.Println()
	}
	printResolutionTrace(parseSummary.Resolution, []string{"app", "root", "output", "profile"}, "  ")
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
	fmt.Println("  examples   Serve the examples catalog or run managed example-server lifecycle actions (start|status|stop|restart)")
	fmt.Println("  dev        Run the native gwc dev orchestration path with integrated livereload runtime")
	fmt.Println("  serve      Serve a static directory, wasm artifact, wasm_exec.js, and optional JSON fixtures")
	fmt.Println("  files      List project files with repeatable extension and directory filters")
	fmt.Println("  lint       Run golangci-lint, capture structured findings, and render a text or JSON review report (review alias supported)")
	fmt.Println("  init       Non-interactive project initialization that writes gwc-start.json and lifecycle defaults")
	fmt.Println("  inspect    Build higher-level route, dependency, ownership, and file-type project reports")
	fmt.Println("  upgrade    Non-interactive lifecycle upgrade for gwc-start.json schema and runtime assets")
	fmt.Println("  migrate    Non-interactive migration helper with compatibility API findings and report export")
	fmt.Println("  prerender  Build one static export output with route HTML, wasm artifacts, and a manifest")
	fmt.Println("  export     Alias for `prerender`")
	fmt.Println("  tailwind   Build shared Tailwind CSS and generated class manifests through the launcher-owned Tailwind path")
	fmt.Println("  dashboard  Monitor live-reload clients and project AI provider configuration from a launcher-owned dashboard")
	fmt.Println("  doctor     Check local toolchains, runtime assets, project signals, and optional golden-path audit anchors")
	fmt.Println("  deploy     Package validated release artifacts through explicit deployment adapters")
	fmt.Println("  env        Print launcher-relevant environment variables and current values")
	fmt.Println("  seed       Provision local dev identities and fixture data through a seed package")
	fmt.Println("  import     Convert a static HTML or JSX file into an inspectable GWC project")
	fmt.Println("  release    Package a js/wasm release with manifest and compressed sidecars")
	fmt.Println("  verify     Run app-local Go tests when present and perform a CI-profile wasm build")
	fmt.Println("  wasm       Run wasm-focused build experiment helpers such as `wasm measure`")
	fmt.Println("  start      Run the scaffold TUI for preset and project setup")
	fmt.Println("  bootstrap  Run prerequisite checks, then start a scaffold or examples bootstrap flow")
}

func printSeedSummary(parseSummary seedSummary) {
	fmt.Println("GWC seed")
	fmt.Printf("  project root: %s\n", parseSummary.ProjectRoot)
	fmt.Printf("  command:      %s\n", parseSummary.CommandPath)
	if parseSummary.DatabasePath != "" {
		fmt.Printf("  database:     %s\n", parseSummary.DatabasePath)
	}
	for _, parseCredential := range parseSummary.Credentials {
		fmt.Printf("  account:      %s / %s", parseCredential.Email, parseCredential.Password)
		if parseCredential.Role != "" {
			fmt.Printf(" (%s)", parseCredential.Role)
		}
		fmt.Println()
	}
	if parseSummary.Output != "" {
		fmt.Printf("  output:       %s\n", parseSummary.Output)
	}
}

func printTestSummary(parseSummary testSummary) {
	fmt.Println("GWC test")
	if parseSummary.AppPath != "" {
		fmt.Printf("  app:          %s\n", parseSummary.AppPath)
	}
	fmt.Printf("  project root: %s\n", parseSummary.ProjectRoot)
	fmt.Printf("  lanes:        %s\n", strings.Join(parseSummary.SelectedLanes, ", "))
	for _, parseLane := range parseSummary.Lanes {
		parseStatus := "ok"
		if parseLane.Skipped {
			parseStatus = "skipped"
		}
		fmt.Printf("  [%s] %s", parseStatus, parseLane.Name)
		if parseLane.Summary != "" {
			fmt.Printf(": %s", parseLane.Summary)
		}
		fmt.Println()
	}
	printResolutionTrace(parseSummary.Resolution, []string{"app", "root"}, "  ")
}

func projectHasGoTests(parseRootPath string) (bool, error) {
	isParseFound := false
	parseErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipTestWalkDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(parseEntry.Name(), "_test.go") {
			isParseFound = true
			return errStopWalk
		}
		return nil
	})
	if parseErr != nil && !errors.Is(parseErr, errStopWalk) {
		return false, fmt.Errorf("scan project tests: %w", parseErr)
	}
	return isParseFound, nil
}

func resolveRepoRoot() (string, error) {
	_, parseCurrentFile, _, parseOk := resolveRepoRootCaller(0)
	if !parseOk {
		return "", errors.New("unable to resolve launcher source path")
	}
	parseRepoRoot := filepath.Clean(filepath.Join(filepath.Dir(parseCurrentFile), "..", ".."))
	if _, parseErr := os.Stat(filepath.Join(parseRepoRoot, "go.mod")); parseErr != nil {
		return "", fmt.Errorf("unable to resolve repo root from %s", parseRepoRoot)
	}
	return parseRepoRoot, nil
}

func joinHostPort(parseHost string, parsePort string) string {
	parseHost = strings.TrimSpace(parseHost)
	parsePort = strings.TrimSpace(parsePort)
	if parseHost == "" {
		parseHost = defaultHost
	}
	if parsePort == "" {
		parsePort = defaultPort
	}
	return parseHost + ":" + parsePort
}

func applyDevHeaders(parseW http.ResponseWriter, parseR *http.Request) {
	if strings.HasSuffix(strings.ToLower(parseR.URL.Path), ".wasm") {
		parseW.Header().Set("Content-Type", "application/wasm")
	}
	parseW.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
}

func writeJSON(parseW http.ResponseWriter, parseStatus int, parsePayload interface{}) {
	parseW.Header().Set("Content-Type", "application/json")
	parseW.WriteHeader(parseStatus)
	_ = json.NewEncoder(parseW).Encode(parsePayload)
}
