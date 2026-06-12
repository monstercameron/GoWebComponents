package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

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
	Name      string `json:"name"`
	Toolchain string `json:"toolchain,omitempty"`
	Target    string `json:"target,omitempty"`
	Trimpath  bool   `json:"trimpath"`
	Ldflags   string `json:"ldflags,omitempty"`
	GCFlags   string `json:"gcflags,omitempty"`
	BuildVCS  string `json:"buildvcs,omitempty"`
	Opt       string `json:"opt,omitempty"`
	// Tags carries build tags for the profile.  Release-shaped profiles set
	// "production" so framework dev-only surfaces (devtools panels, hot-reload
	// scaffolding) are excluded from shipped artifacts.
	Tags string `json:"tags,omitempty"`
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
	// SizeWarnings lists heavyweight stdlib import chains detected in the wasm
	// dependency graph (net/http, regexp, ...) with actionable guidance.
	SizeWarnings []string `json:"sizeWarnings,omitempty"`
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
	Flags        map[string]any                   `json:"flags"`
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

var buildLookPath = exec.LookPath

var buildRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = cwd
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	return string(output), err
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
		strings.Contains(parseLower, "requires tinygo") ||
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

func printBuildSummary(parseSummary buildSummary) {
	fmt.Println("GWC build")
	fmt.Printf("  profile:      %s\n", parseSummary.Profile.Name)
	fmt.Printf("  toolchain:    %s\n", firstNonEmpty(parseSummary.Profile.Toolchain, "go"))
	fmt.Printf("  target:       %s\n", firstNonEmpty(parseSummary.Profile.Target, "js/wasm"))
	fmt.Printf("  app:          %s\n", parseSummary.AppPath)
	fmt.Printf("  project root: %s\n", parseSummary.ProjectRoot)
	fmt.Printf("  package dir:  %s\n", parseSummary.PackageDir)
	fmt.Printf("  output:       %s\n", parseSummary.OutputPath)
	fmt.Printf("  bytes:        %d\n", parseSummary.Bytes)
	fmt.Printf("  sha256:       %s\n", parseSummary.SHA256)
	fmt.Printf("  trimpath:     %t\n", parseSummary.Profile.Trimpath)
	fmt.Printf("  ldflags:      %s\n", firstNonEmpty(parseSummary.Profile.Ldflags, "<none>"))
	fmt.Printf("  gcflags:      %s\n", firstNonEmpty(parseSummary.Profile.GCFlags, "<none>"))
	fmt.Printf("  buildvcs:     %s\n", firstNonEmpty(parseSummary.Profile.BuildVCS, "default"))
	fmt.Printf("  opt:          %s\n", firstNonEmpty(parseSummary.Profile.Opt, "<none>"))
	fmt.Printf("  tags:         %s\n", firstNonEmpty(parseSummary.Profile.Tags, "<none>"))
	for _, parseWarning := range parseSummary.SizeWarnings {
		fmt.Printf("  ⚠ %s\n", parseWarning)
	}
	printResolutionTrace(parseSummary.Resolution, []string{"app", "root", "output", "profile"}, "  ")
}

func printReleaseSummary(parseSummary releaseSummary) {
	fmt.Println("GWC release")
	fmt.Printf("  profile:      %s\n", parseSummary.Profile.Name)
	fmt.Printf("  toolchain:    %s\n", firstNonEmpty(parseSummary.Profile.Toolchain, "go"))
	fmt.Printf("  target:       %s\n", firstNonEmpty(parseSummary.Profile.Target, "js/wasm"))
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
	fmt.Println("  lint       Run golangci-lint plus built-in GWC hook rules, then render a text or JSON review report (review alias supported)")
	fmt.Println("  init       Non-interactive project initialization that writes gwc-start.json and lifecycle defaults")
	fmt.Println("  inspect    Build higher-level route, dependency, ownership, and file-type project reports")
	fmt.Println("  upgrade    Non-interactive lifecycle upgrade for gwc-start.json schema and runtime assets")
	fmt.Println("  migrate    Non-interactive migration helper with compatibility API findings, safe rewrites, and report export")
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

func writeJSON(parseW http.ResponseWriter, parseStatus int, parsePayload any) {
	parseW.Header().Set("Content-Type", "application/json")
	parseW.WriteHeader(parseStatus)
	_ = json.NewEncoder(parseW).Encode(parsePayload)
}
