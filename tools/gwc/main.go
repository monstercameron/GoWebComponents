package main

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/parser"
	"go/token"
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
	gwchtml "github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/pwa"
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
	return resolveLauncherExamplesWasmDir(l.repoRoot, l.staticDir)
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
	appPath    string
	rootPath   string
	htmlPath   string
	wasmPath   string
	host       string
	port       string
	hot        bool
	resolution map[string]string
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

type doctorConfig struct {
	host               string
	port               string
	audit              bool
	auditPolicy        string
	auditBaselinePath  string
	auditWriteBaseline string
	auditSuppressions  []string
	json               bool
}

type doctorCheck struct {
	RuleID      string   `json:"ruleId,omitempty"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Severity    string   `json:"severity,omitempty"`
	Summary     string   `json:"summary"`
	Locations   []string `json:"locations,omitempty"`
	Hint        string   `json:"hint,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

type doctorReport struct {
	OK         bool               `json:"ok"`
	Checked    string             `json:"checked"`
	CWD        string             `json:"cwd"`
	Checks     []doctorCheck      `json:"checks"`
	Audit      *doctorAuditReport `json:"audit,omitempty"`
	Resolution map[string]string  `json:"resolution,omitempty"`
}

type doctorAuditReport struct {
	Mode         string        `json:"mode"`
	Policy       string        `json:"policy"`
	OK           bool          `json:"ok"`
	BaselinePath string        `json:"baselinePath,omitempty"`
	Suppressed   []string      `json:"suppressed,omitempty"`
	Checks       []doctorCheck `json:"checks"`
}

type doctorAuditBaseline struct {
	Mode        string                     `json:"mode"`
	GeneratedAt string                     `json:"generatedAt"`
	Checks      []doctorAuditBaselineCheck `json:"checks,omitempty"`
}

type doctorAuditBaselineCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Summary string `json:"summary"`
}

var doctorLookPath = exec.LookPath

var doctorAuditLocationPattern = regexp.MustCompile(`([A-Za-z0-9_./-]+\.(?:go|html|json|wasm))`)

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

var releaseMeasureStartup = measureReleaseStartup

var releaseResolveRepoRoot = resolveRepoRoot

var releaseResolveWasmExec = resolveWasmExecPath

var releaseValidateSmoke = validateReleaseSmoke

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

var scaffoldTidyModule = func(l launcher, targetDir string) error {
	return l.tidyScaffoldModule(targetDir)
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
	case "build", "dev", "doctor", "files", "release", "seed", "test", "verify":
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
	case "release":
		return runReleaseCommand(l, args)
	case "dev":
		return runDevCommand(l, args)
	case "serve":
		return runServeCommand(l, args)
	case "files":
		return runFilesCommand(l, args)
	case "dashboard":
		return runDashboardCommand(l, args)
	case "doctor":
		return runDoctorCommand(l, args)
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

func (l launcher) runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	host := fs.String("host", defaultHost, "Host to probe for port availability")
	port := fs.String("port", "8080", "Port to probe for local development availability")
	audit := fs.Bool("audit", false, "Run the golden-path app audit in addition to prerequisite checks")
	auditPolicy := fs.String("audit-policy", "strict", "Golden-path audit policy: strict or advisory")
	auditBaseline := fs.String("audit-baseline", "", "Optional path to a JSON baseline file of accepted audit findings")
	auditWriteBaseline := fs.String("audit-write-baseline", "", "Optional path to write the current audit findings as a JSON baseline")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	var auditSuppressions stringListFlag
	fs.Var(&auditSuppressions, "audit-suppress", "Audit check name to suppress; repeat or comma-separate")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	report := l.buildDoctorReport(doctorConfig{
		host:               *host,
		port:               *port,
		audit:              *audit,
		auditPolicy:        *auditPolicy,
		auditBaselinePath:  *auditBaseline,
		auditWriteBaseline: *auditWriteBaseline,
		auditSuppressions:  auditSuppressions.Values(),
		json:               *jsonOutput,
	})
	if *audit && strings.TrimSpace(*auditWriteBaseline) != "" && report.Audit != nil {
		if err := writeDoctorAuditBaseline(*auditWriteBaseline, *report.Audit); err != nil {
			return err
		}
	}
	if *jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
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
	tui := fs.Bool("tui", false, "Show an interactive status TUI while gwc dev runs")
	clientScript := fs.String("client-script", "", "Optional override path to a custom livereload client script")
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
	plan := describeDevPlan(config)
	cmd := exec.Command("go", forwarded...)
	cmd.Dir = livereloadWorkspace
	cmd.Env = os.Environ()
	if *tui {
		if plan.ServerMode != "livereload-wasm" || strings.TrimSpace(plan.StatusURL) == "" {
			return errors.New("dev -tui is only supported for livereload-backed js/wasm app runs")
		}
		return runDevStatusTUI(plan, plan.StatusURL, cmd)
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
		return nil, errors.New("post-link optimization requested but wasm-opt is unavailable; install wasm-opt or make npx binaryen available")
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
	if path, err := releaseLookPath("npx"); err == nil {
		return releaseCommandInfo{
			Available:  true,
			Command:    "npx",
			PrefixArgs: []string{"--yes", "--package", "binaryen", "wasm-opt"},
			Label:      path + " --yes --package binaryen wasm-opt",
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

	repoRoot, err := releaseResolveRepoRoot()
	if err != nil {
		return nil, fmt.Errorf("resolve repo root for startup measurement: %w", err)
	}
	workspace, err := resolveBrowserWorkspace(repoRoot, config.rootPath)
	if err != nil {
		return nil, fmt.Errorf("resolve browser workspace for startup measurement: %w", err)
	}
	if strings.TrimSpace(workspace) == "" {
		return nil, errors.New("startup measurement requested but no browser workspace was found")
	}
	if !fileExists(filepath.Join(workspace, "node_modules", "@playwright", "test", "package.json")) {
		return nil, fmt.Errorf("startup measurement requested but Playwright is not installed under %s", filepath.Join(workspace, "node_modules"))
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
	scriptPath := filepath.Join(repoRoot, "examples", "tools", "release-startup-probe.cjs")
	if !fileExists(scriptPath) {
		return nil, fmt.Errorf("startup measurement probe script was not found: %s", scriptPath)
	}
	args := []string{scriptPath, probeURL, reportPath, fmt.Sprintf("%d", config.startupTimeoutMs)}
	output, err := releaseRunCommand("node", args, workspace, buildBrowserTestEnv())
	if err != nil {
		trimmed := strings.TrimSpace(output)
		if trimmed == "" {
			return nil, fmt.Errorf("measure release startup: %w", err)
		}
		return nil, fmt.Errorf("measure release startup: %s", trimmed)
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
			buildDoctorToolCheck("node", "Node.js", "--version", "Install Node.js so browser-test tooling can run for this starter."),
			buildDoctorToolCheck("npm", "npm", "--version", "Install npm so browser-test tooling can run for this starter."),
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
	{Key: "browser-tests", Label: "Browser Tests", Description: "Playwright-ready smoke-test placeholders are generated under test/browser."},
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
	return fmt.Sprintf("# Browser smoke tests for %s\n\nUse this folder for Playwright specs that prove starter boot, routing, and basic user interactions.\n\nExample command from the repo root:\n\n```powershell\ngo run ./tools/gwc test -lane browser\n```\n", selection.ProjectName)
}

func renderScaffoldBrowserSmokeTest(selection startSelection) string {
	return fmt.Sprintf("import { expect, test } from '@playwright/test';\n\ntest('starter shell renders', async ({ page }) => {\n\tawait page.goto('/');\n\tawait expect(page.getByRole('heading', { name: /%s/i })).toBeVisible();\n});\n", selection.ProjectName)
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
		extraPaths = append(extraPaths, "test/browser/smoke.spec.ts")
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
		files["test/browser/README.md"] = []byte(renderScaffoldBrowserTestREADME(selection))
		files["test/browser/smoke.spec.ts"] = []byte(renderScaffoldBrowserSmokeTest(selection))
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
		builder.WriteString("\nFor browser tests, start from `test/browser/smoke.spec.ts` and run:\n\n")
		builder.WriteString("```powershell\n")
		builder.WriteString("go run ./tools/gwc test -lane browser\n")
		builder.WriteString("```\n")
	}
	return builder.String()
}

func (l launcher) resolveDevConfig(config devConfig) (devConfig, error) {
	resolved := config
	resolved.resolution = cloneResolutionTrace(config.resolution)
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
			resolved.resolution = setResolutionSource(resolved.resolution, "app", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.rootPath) == "" {
			resolved.rootPath = metadataDir
			resolved.resolution = setResolutionSource(resolved.resolution, "root", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.htmlPath) == "" && strings.TrimSpace(metadata.Tooling.HTMLPath) != "" {
			resolved.htmlPath = filepath.Join(metadataDir, filepath.FromSlash(metadata.Tooling.HTMLPath))
			resolved.resolution = setResolutionSource(resolved.resolution, "html", "gwc-start.json")
		}
		if strings.TrimSpace(resolved.wasmPath) == "" && strings.TrimSpace(metadata.Tooling.WASMPath) != "" {
			resolved.wasmPath = metadata.Tooling.WASMPath
			resolved.resolution = setResolutionSource(resolved.resolution, "wasm", "gwc-start.json")
		}
		if resolved.host == "" {
			resolved.host = strings.TrimSpace(metadata.Tooling.DevHost)
			resolved.resolution = setResolutionSource(resolved.resolution, "host", "gwc-start.json")
		}
		if resolved.port == "" {
			resolved.port = strings.TrimSpace(metadata.Tooling.DevPort)
			resolved.resolution = setResolutionSource(resolved.resolution, "port", "gwc-start.json")
		}
	}
	if strings.TrimSpace(config.appPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "explicit flag")
	}
	if strings.TrimSpace(config.rootPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "explicit flag")
	}
	if strings.TrimSpace(config.htmlPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "html", "explicit flag")
	}
	if strings.TrimSpace(config.wasmPath) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "wasm", "explicit flag")
	}
	if strings.TrimSpace(config.host) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "host", "explicit flag")
	}
	if strings.TrimSpace(config.port) != "" {
		resolved.resolution = setResolutionSource(resolved.resolution, "port", "explicit flag")
	}
	if resolved.host == "" {
		resolved.host = "127.0.0.1"
		resolved.resolution = setResolutionSource(resolved.resolution, "host", "convention fallback")
	}
	if resolved.port == "" {
		resolved.port = "8080"
		resolved.resolution = setResolutionSource(resolved.resolution, "port", "convention fallback")
	}

	if strings.TrimSpace(resolved.appPath) == "" {
		resolved.appPath, err = detectAppPath(cwd)
		if err != nil {
			return devConfig{}, err
		}
		resolved.resolution = setResolutionSource(resolved.resolution, "app", "convention fallback")
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
		resolved.resolution = setResolutionSource(resolved.resolution, "root", "convention fallback")
	}
	resolved.rootPath, err = normalizePath(cwd, resolved.rootPath)
	if err != nil {
		return devConfig{}, fmt.Errorf("resolve root path: %w", err)
	}

	if strings.TrimSpace(resolved.htmlPath) == "" {
		resolved.htmlPath = detectHTMLPath(resolved.rootPath)
		if strings.TrimSpace(resolved.htmlPath) != "" {
			resolved.resolution = setResolutionSource(resolved.resolution, "html", "convention fallback")
		}
	}
	if strings.TrimSpace(resolved.htmlPath) != "" {
		resolved.htmlPath, err = normalizePath(cwd, resolved.htmlPath)
		if err != nil {
			return devConfig{}, fmt.Errorf("resolve html path: %w", err)
		}
	}

	if strings.TrimSpace(resolved.wasmPath) != "" {
		resolved.wasmPath = strings.TrimSpace(resolved.wasmPath)
	} else {
		resolved.resolution = setResolutionSource(resolved.resolution, "wasm", "convention fallback")
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
	if err := normalizeScaffoldMetadata(&metadata); err != nil {
		return scaffoldMetadata{}, false, err
	}
	return metadata, true, nil
}

func normalizeScaffoldMetadata(metadata *scaffoldMetadata) error {
	if metadata == nil {
		return nil
	}
	switch metadata.SchemaVersion {
	case 0:
		metadata.SchemaVersion = currentScaffoldMetadataSchemaVersion
		if strings.TrimSpace(metadata.Ownership.ProjectOwnership) == "" {
			metadata.Ownership.ProjectOwnership = "standalone"
		}
		if strings.TrimSpace(metadata.Ownership.FrameworkSourceMode) == "" {
			metadata.Ownership.FrameworkSourceMode = "module-proxy"
		}
		return nil
	case currentScaffoldMetadataSchemaVersion:
		if strings.TrimSpace(metadata.Ownership.ProjectOwnership) == "" {
			metadata.Ownership.ProjectOwnership = "standalone"
		}
		if strings.TrimSpace(metadata.Ownership.FrameworkSourceMode) == "" {
			metadata.Ownership.FrameworkSourceMode = "module-proxy"
		}
		return nil
	default:
		return fmt.Errorf("unsupported scaffold metadata schema version %d", metadata.SchemaVersion)
	}
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
	if plan.ServerMode == "livereload-wasm" {
		fmt.Printf("  status URL:    %s\n", plan.StatusURL)
	}
	printResolutionTrace(config.resolution, []string{"app", "root", "html", "wasm", "host", "port"}, "  ")
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
		"statusURL":    plan.StatusURL,
		"resolution":   config.resolution,
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
	StatusURL    string
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
	statusURL := ""
	if serverMode == "livereload-wasm" {
		statusURL = "http://" + joinHostPort(config.host, config.port) + "/__gwc/status"
	}
	return devPlanSummary{
		ProjectRoot:  projectRoot,
		AppMode:      appMode,
		ServerMode:   serverMode,
		ListeningURL: "http://" + joinHostPort(config.host, config.port),
		StatusURL:    statusURL,
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
	fmt.Println("  serve      Serve a static directory, wasm artifact, wasm_exec.js, and optional JSON fixtures")
	fmt.Println("  files      List project files with repeatable extension and directory filters")
	fmt.Println("  dashboard  Monitor live-reload clients and project AI provider configuration from a launcher-owned dashboard")
	fmt.Println("  doctor     Check local toolchains, runtime assets, project signals, and optional golden-path audit anchors")
	fmt.Println("  seed       Provision local dev identities and fixture data through a seed package")
	fmt.Println("  import     Convert a static HTML or JSX file into an inspectable GWC project")
	fmt.Println("  release    Package a js/wasm release with manifest and compressed sidecars")
	fmt.Println("  verify     Run app-local Go tests when present and perform a CI-profile wasm build")
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

func (l launcher) buildDoctorReport(config doctorConfig) doctorReport {
	cwd, err := doctorGetwd()
	if err != nil {
		cwd = ""
	}
	report := doctorReport{
		OK:         true,
		Checked:    time.Now().UTC().Format(time.RFC3339),
		CWD:        cwd,
		Resolution: buildDoctorResolutionTrace(cwd, config),
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
	if config.audit {
		report.Audit = buildDoctorAuditReport(cwd, config)
		if report.Audit.Policy == "strict" && !report.Audit.OK {
			report.OK = false
		}
	}

	return report
}

func buildDoctorResolutionTrace(cwd string, config doctorConfig) map[string]string {
	trace := map[string]string{}
	if strings.TrimSpace(cwd) != "" {
		trace["root"] = "working directory"
	}
	if strings.TrimSpace(config.host) != "" && config.host != defaultHost {
		trace["host"] = "explicit flag"
	} else {
		trace["host"] = "convention fallback"
	}
	if strings.TrimSpace(config.port) != "" && config.port != "8080" {
		trace["port"] = "explicit flag"
	} else {
		trace["port"] = "convention fallback"
	}
	metadata, metadataDir, hasMetadata, err := resolveScaffoldMetadataForConfig(cwd, "", "")
	if err == nil && hasMetadata {
		if strings.TrimSpace(metadata.Tooling.AppPath) != "" {
			trace["app"] = "gwc-start.json"
		}
		if strings.TrimSpace(metadata.Tooling.HTMLPath) != "" {
			trace["html"] = "gwc-start.json"
		}
		if strings.TrimSpace(metadataDir) != "" {
			trace["root"] = "gwc-start.json"
		}
	}
	if _, ok := trace["app"]; !ok && strings.TrimSpace(cwd) != "" {
		if _, err := detectAppPath(cwd); err == nil {
			trace["app"] = "convention fallback"
		}
	}
	if _, ok := trace["html"]; !ok && strings.TrimSpace(cwd) != "" {
		if strings.TrimSpace(detectHTMLPath(cwd)) != "" {
			trace["html"] = "convention fallback"
		}
	}
	return trace
}

func buildDoctorAuditReport(cwd string, config doctorConfig) *doctorAuditReport {
	audit := buildDoctorGoldenPathAudit(cwd)
	applyDoctorAuditAdoption(&audit, cwd, config)
	if strings.TrimSpace(audit.Policy) == "" {
		audit.Policy = "strict"
	}
	return &audit
}

func applyDoctorAuditAdoption(audit *doctorAuditReport, cwd string, config doctorConfig) {
	if audit == nil {
		return
	}
	policy, ok := normalizeDoctorAuditPolicy(config.auditPolicy)
	if !ok {
		audit.Policy = "strict"
		audit.OK = false
		audit.Checks = append([]doctorCheck{{
			Name:    "Audit policy",
			Status:  "fail",
			Summary: fmt.Sprintf("Unknown audit policy %q.", config.auditPolicy),
			Hint:    "Use -audit-policy strict or -audit-policy advisory.",
		}}, audit.Checks...)
		annotateDoctorAuditMetadata(audit)
		return
	}
	audit.Policy = policy
	suppressed := map[string]struct{}{}
	for _, name := range config.auditSuppressions {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			suppressed[trimmed] = struct{}{}
		}
	}
	if strings.TrimSpace(config.auditBaselinePath) != "" {
		baselinePath, err := normalizePath(cwd, config.auditBaselinePath)
		if err != nil {
			audit.OK = false
			audit.Checks = append([]doctorCheck{{
				Name:    "Audit baseline",
				Status:  "fail",
				Summary: fmt.Sprintf("Could not resolve audit baseline path: %v", err),
				Hint:    "Pass a valid file path to -audit-baseline or remove the flag.",
			}}, audit.Checks...)
			annotateDoctorAuditMetadata(audit)
			return
		}
		baseline, err := loadDoctorAuditBaseline(baselinePath)
		if err != nil {
			audit.OK = false
			audit.Checks = append([]doctorCheck{{
				Name:    "Audit baseline",
				Status:  "fail",
				Summary: err.Error(),
				Hint:    "Write a fresh baseline with -audit-write-baseline or fix the checked-in baseline file.",
			}}, audit.Checks...)
			annotateDoctorAuditMetadata(audit)
			return
		}
		audit.BaselinePath = baselinePath
		for _, entry := range baseline.Checks {
			trimmed := strings.TrimSpace(entry.Name)
			if trimmed != "" {
				suppressed[trimmed] = struct{}{}
			}
		}
	}
	if len(suppressed) == 0 {
		audit.OK = doctorAuditChecksPassing(audit.Checks)
		annotateDoctorAuditMetadata(audit)
		return
	}
	names := make([]string, 0, len(suppressed))
	for name := range suppressed {
		names = append(names, name)
	}
	sort.Strings(names)
	audit.Suppressed = names
	for i := range audit.Checks {
		check := &audit.Checks[i]
		if check.Status == "pass" || check.Status == "suppressed" {
			continue
		}
		if _, ok := suppressed[check.Name]; ok {
			check.Status = "suppressed"
			check.Summary = check.Summary + " (suppressed)"
		}
	}
	audit.OK = doctorAuditChecksPassing(audit.Checks)
	annotateDoctorAuditMetadata(audit)
}

func normalizeDoctorAuditPolicy(value string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "strict":
		return "strict", true
	case "advisory", "warn":
		return "advisory", true
	default:
		return "", false
	}
}

func doctorAuditChecksPassing(checks []doctorCheck) bool {
	for _, check := range checks {
		if check.Status == "fail" {
			return false
		}
	}
	return true
}

func annotateDoctorAuditMetadata(audit *doctorAuditReport) {
	if audit == nil {
		return
	}
	for i := range audit.Checks {
		check := &audit.Checks[i]
		if check.RuleID == "" {
			check.RuleID = doctorAuditRuleIDForName(check.Name)
		}
		check.Severity = doctorAuditSeverityForStatus(check.Status)
		if len(check.Locations) == 0 {
			check.Locations = extractDoctorAuditLocations(check.Summary)
		}
		if strings.TrimSpace(check.Remediation) == "" && strings.TrimSpace(check.Hint) != "" {
			check.Remediation = check.Hint
		}
	}
}

func appendDoctorLocation(locations []string, value string) []string {
	trimmed := strings.TrimSpace(filepath.ToSlash(value))
	if trimmed == "" {
		return locations
	}
	for _, existing := range locations {
		if existing == trimmed {
			return locations
		}
	}
	return append(locations, trimmed)
}

func doctorAuditLocation(root string, path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if strings.TrimSpace(root) != "" {
		if rel, err := filepath.Rel(root, trimmed); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(trimmed)
}

func doctorAuditRuleIDForName(name string) string {
	switch name {
	case "Audit policy":
		return "audit.policy"
	case "Audit baseline":
		return "audit.baseline"
	case "Audit target":
		return "audit.target"
	case "App entrypoint":
		return "audit.app_entrypoint"
	case "HTML shell":
		return "audit.html_shell"
	case "Starter metadata anchor":
		return "audit.metadata_anchor"
	case "State and ownership boundaries":
		return "audit.state_boundaries"
	case "Local versus shared state ownership":
		return "audit.state_ownership"
	case "Route shape and delivery":
		return "audit.route_delivery"
	case "Mutation and resilience":
		return "audit.mutation_resilience"
	case "Startup cost and ownership evidence":
		return "audit.startup_evidence"
	case "Runtime evidence":
		return "audit.runtime_evidence"
	default:
		return ""
	}
}

func doctorAuditSeverityForStatus(status string) string {
	switch strings.TrimSpace(strings.ToLower(status)) {
	case "fail":
		return "error"
	case "warn":
		return "warning"
	case "suppressed":
		return "suppressed"
	case "pass":
		return "info"
	default:
		return ""
	}
}

func normalizeDoctorAuditMinimumSeverity(value string) (string, bool) {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "", "error":
		return "error", true
	case "warning", "warn":
		return "warning", true
	case "info":
		return "info", true
	case "off", "none":
		return "off", true
	default:
		return "", false
	}
}

func doctorAuditSeverityRank(value string) int {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "info":
		return 1
	case "warning", "warn":
		return 2
	case "error":
		return 3
	default:
		return 0
	}
}

func doctorAuditHasFindingAtOrAbove(checks []doctorCheck, minimum string) bool {
	normalized, ok := normalizeDoctorAuditMinimumSeverity(minimum)
	if !ok || normalized == "off" {
		return false
	}
	requiredRank := doctorAuditSeverityRank(normalized)
	for _, check := range checks {
		if check.Status == "pass" || check.Status == "suppressed" {
			continue
		}
		if doctorAuditSeverityRank(check.Severity) >= requiredRank {
			return true
		}
	}
	return false
}

func extractDoctorAuditLocations(summary string) []string {
	matches := doctorAuditLocationPattern.FindAllString(summary, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	locations := make([]string, 0, len(matches))
	for _, match := range matches {
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		locations = append(locations, match)
	}
	return locations
}

func buildDoctorGoldenPathAudit(cwd string) doctorAuditReport {
	report := doctorAuditReport{
		Mode: "golden-path",
		OK:   true,
	}
	appendCheck := func(check doctorCheck) {
		report.Checks = append(report.Checks, check)
		if check.Status == "fail" {
			report.OK = false
		}
	}
	if strings.TrimSpace(cwd) == "" {
		appendCheck(doctorCheck{
			RuleID:  "audit.target",
			Name:    "Audit target",
			Status:  "fail",
			Summary: "The current working directory could not be resolved for golden-path auditing.",
			Hint:    "Run `gwc doctor -audit` from the target app root.",
		})
		annotateDoctorAuditMetadata(&report)
		return report
	}
	appPath, appErr := detectAppPath(cwd)
	if appErr != nil {
		appendCheck(doctorCheck{
			RuleID:  "audit.app_entrypoint",
			Name:    "App entrypoint",
			Status:  "fail",
			Summary: "No launcher-detectable app entrypoint was found for golden-path auditing.",
			Hint:    "Keep main.go or cmd/web/main.go at the documented locations, or add scaffold metadata that pins the app path.",
		})
	} else {
		appendCheck(doctorCheck{
			RuleID:    "audit.app_entrypoint",
			Name:      "App entrypoint",
			Status:    "pass",
			Summary:   fmt.Sprintf("Auditing app entrypoint %s", appPath),
			Locations: []string{doctorAuditLocation(cwd, appPath)},
		})
	}
	htmlPath := detectHTMLPath(cwd)
	if strings.TrimSpace(htmlPath) == "" {
		appendCheck(doctorCheck{
			RuleID:  "audit.html_shell",
			Name:    "HTML shell",
			Status:  "warn",
			Summary: "No HTML shell was auto-detected for the current app root.",
			Hint:    "Keep index.html or a documented equivalent near the app so later delivery audits can reason about the served shell.",
		})
	} else {
		appendCheck(doctorCheck{
			RuleID:    "audit.html_shell",
			Name:      "HTML shell",
			Status:    "pass",
			Summary:   fmt.Sprintf("Detected HTML shell %s", htmlPath),
			Locations: []string{doctorAuditLocation(cwd, htmlPath)},
		})
	}
	metadata, ok, err := loadScaffoldMetadata(cwd)
	if err != nil {
		appendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "fail",
			Summary:   err.Error(),
			Locations: []string{"gwc-start.json"},
			Hint:      "Fix or regenerate gwc-start.json so golden-path audits can resolve intended launcher ownership.",
		})
	} else if !ok {
		appendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "warn",
			Summary:   "No gwc-start.json metadata anchor was found for this app.",
			Locations: []string{"gwc-start.json"},
			Hint:      "Generated starters should keep scaffold metadata so future audit rules can trace intended ownership and output paths.",
		})
	} else {
		appendCheck(doctorCheck{
			RuleID:    "audit.metadata_anchor",
			Name:      "Starter metadata anchor",
			Status:    "pass",
			Summary:   fmt.Sprintf("Using starter metadata for %s", firstNonEmpty(metadata.ProjectName, "<unnamed>")),
			Locations: []string{"gwc-start.json"},
		})
	}
	appendCheck(buildDoctorOwnershipBoundaryCheck(cwd))
	appendCheck(buildDoctorStateOwnershipCheck(cwd))
	appendCheck(buildDoctorRouteDeliveryCheck(cwd))
	appendCheck(buildDoctorMutationResilienceCheck(cwd))
	appendCheck(buildDoctorStartupEvidenceCheck(cwd))
	appendCheck(buildDoctorRuntimeEvidenceCheck(cwd))
	annotateDoctorAuditMetadata(&report)
	return report
}

func buildDoctorOwnershipBoundaryCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.state_boundaries",
			Name:    "State and ownership boundaries",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for ownership-boundary auditing: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	violations := []string{}
	locations := []string{}
	for _, file := range files {
		for _, importPath := range file.Imports {
			switch {
			case file.Client && strings.Contains(importPath, "/server/"):
				violations = append(violations, fmt.Sprintf("%s imports server package %s", file.RelPath, importPath))
				locations = appendDoctorLocation(locations, file.RelPath)
			case file.Client && (importPath == "database/sql" || importPath == "os/exec"):
				violations = append(violations, fmt.Sprintf("%s imports server-only package %s", file.RelPath, importPath))
				locations = appendDoctorLocation(locations, file.RelPath)
			case !file.Client && importPath == "syscall/js":
				violations = append(violations, fmt.Sprintf("%s imports browser-only package %s outside a js/wasm boundary", file.RelPath, importPath))
				locations = appendDoctorLocation(locations, file.RelPath)
			}
		}
	}
	if len(violations) == 0 {
		return doctorCheck{
			RuleID:  "audit.state_boundaries",
			Name:    "State and ownership boundaries",
			Status:  "pass",
			Summary: "No static client/server ownership boundary leaks were detected in Go imports.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.state_boundaries",
		Name:      "State and ownership boundaries",
		Status:    "fail",
		Summary:   summarizeDoctorViolations(violations, 3),
		Locations: locations,
		Hint:      "Keep client files on browser-safe imports, keep server packages out of js/wasm paths, and keep syscall/js usage behind explicit js/wasm build tags.",
	}
}

func buildDoctorStateOwnershipCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.state_ownership",
			Name:    "Local versus shared state ownership",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for mixed state ownership heuristics: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	violations := []string{}
	locations := []string{}
	for _, file := range files {
		if !file.Client {
			continue
		}
		if strings.Contains(file.Content, "fetch.UseCachedResource") &&
			(strings.Contains(file.Content, "localStorage") || strings.Contains(file.Content, "sessionStorage") || strings.Contains(file.Content, "indexedDB")) {
			violations = append(violations, fmt.Sprintf("%s mixes fetch.UseCachedResource with direct browser storage access", file.RelPath))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if len(violations) == 0 {
		return doctorCheck{
			RuleID:  "audit.state_ownership",
			Name:    "Local versus shared state ownership",
			Status:  "pass",
			Summary: "No mixed local-versus-shared state ownership heuristics were detected in client files.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.state_ownership",
		Name:      "Local versus shared state ownership",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(violations, 3),
		Locations: locations,
		Hint:      "Prefer one obvious owner per state slice: either fetch/cache-backed shared data or browser-local persistence, not both in the same controller without an explicit boundary.",
	}
}

func buildDoctorRouteDeliveryCheck(cwd string) doctorCheck {
	goFiles, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for route-shape auditing: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	htmlFiles, err := collectDoctorHTMLFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan HTML shells for route-shape auditing: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	warnings := []string{}
	locations := []string{}
	if len(htmlFiles) > 1 {
		warnings = append(warnings, fmt.Sprintf("multiple HTML entry shells detected (%s)", strings.Join(htmlFiles, ", ")))
		for _, path := range htmlFiles {
			locations = appendDoctorLocation(locations, path)
		}
	}
	routeCount := 0
	hasLazySplit := false
	hasPrerenderSignal := false
	marketingRoutes := false
	for _, file := range goFiles {
		routeHits := strings.Count(file.Content, "MustDefineRoute(")
		routeHits += strings.Count(file.Content, "router.Register(")
		if routeHits > 0 {
			routeCount += routeHits
			locations = appendDoctorLocation(locations, file.RelPath)
		}
		if strings.Contains(file.Content, "ui.Lazy(") || strings.Contains(file.Content, "ui.CreateElement(ui.Lazy") {
			hasLazySplit = true
		}
		lowered := strings.ToLower(file.Content)
		if strings.Contains(lowered, "prerender") || strings.Contains(lowered, "static shell") {
			hasPrerenderSignal = true
		}
		if strings.Contains(file.Content, `"/pricing"`) || strings.Contains(file.Content, `"/capabilities"`) || strings.Contains(file.Content, `"/about"`) || strings.Contains(file.Content, `"/docs"`) {
			marketingRoutes = true
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if routeCount >= 3 && !hasLazySplit {
		warnings = append(warnings, fmt.Sprintf("route tree defines %d route registration points with no ui.Lazy split signal", routeCount))
	}
	if marketingRoutes && !hasPrerenderSignal {
		warnings = append(warnings, "marketing-style routes were detected with no static/prerender delivery hint")
	}
	if len(warnings) == 0 {
		return doctorCheck{
			RuleID:  "audit.route_delivery",
			Name:    "Route shape and delivery",
			Status:  "pass",
			Summary: "No first-pass route-shape or delivery warnings were detected.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.route_delivery",
		Name:      "Route shape and delivery",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Prefer one app shell, prerender or keep static-friendly marketing routes cheap, and use ui.Lazy or equivalent delivery splits when route trees start carrying distinct feature surfaces.",
	}
}

func buildDoctorMutationResilienceCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.mutation_resilience",
			Name:    "Mutation and resilience",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for mutation resilience heuristics: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	mutationIndicators := []string{"http.methodpost", "http.methodput", "http.methodpatch", "http.methoddelete", ".exec(", "deleteconversation", "upsert", "setselected", "setcustom", "signup(", "login(", "save", "submit"}
	resilienceIndicators := []string{"retry", "backoff", "idempot", "rollback", "conflict", "offline", "replay", "timeout", "deadline"}
	warnings := []string{}
	locations := []string{}
	for _, file := range files {
		lowered := strings.ToLower(file.Content)
		hasMutation := false
		for _, indicator := range mutationIndicators {
			if strings.Contains(lowered, indicator) {
				hasMutation = true
				break
			}
		}
		if !hasMutation {
			continue
		}
		hasResilienceSignal := false
		for _, indicator := range resilienceIndicators {
			if strings.Contains(lowered, indicator) {
				hasResilienceSignal = true
				break
			}
		}
		if !hasResilienceSignal {
			warnings = append(warnings, fmt.Sprintf("%s exposes mutation-shaped code with no retry/idempotency/conflict/offline signal", file.RelPath))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if len(warnings) == 0 {
		return doctorCheck{
			RuleID:  "audit.mutation_resilience",
			Name:    "Mutation and resilience",
			Status:  "pass",
			Summary: "No first-pass mutation resilience warnings were detected.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.mutation_resilience",
		Name:      "Mutation and resilience",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Document or encode retry posture, idempotency boundaries, rollback/conflict handling, or offline replay semantics around important writes instead of shipping only the happy path.",
	}
}

func buildDoctorStartupEvidenceCheck(cwd string) doctorCheck {
	files, err := collectGoldenPathGoFiles(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.startup_evidence",
			Name:    "Startup cost and ownership evidence",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not scan Go files for startup-cost evidence: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	warnings := []string{}
	locations := []string{}
	for _, file := range files {
		if !file.Client {
			continue
		}
		if file.SizeBytes > 64*1024 {
			warnings = append(warnings, fmt.Sprintf("%s is %s of client-owned source in the initial app path", file.RelPath, formatBytesBinary(file.SizeBytes)))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
		if len(file.Imports) > 10 {
			warnings = append(warnings, fmt.Sprintf("%s imports %d packages from one client-owned file", file.RelPath, len(file.Imports)))
			locations = appendDoctorLocation(locations, file.RelPath)
		}
	}
	if len(warnings) == 0 {
		return doctorCheck{
			RuleID:  "audit.startup_evidence",
			Name:    "Startup cost and ownership evidence",
			Status:  "pass",
			Summary: "No first-pass startup-cost evidence warnings were detected in client-owned source files.",
		}
	}
	return doctorCheck{
		RuleID:    "audit.startup_evidence",
		Name:      "Startup cost and ownership evidence",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Keep large or dependency-heavy files out of the initial client path when they can stay server-owned or move behind a lazy boundary.",
	}
}

func buildDoctorRuntimeEvidenceCheck(cwd string) doctorCheck {
	startupReports, wasmFiles, err := collectDoctorRuntimeArtifacts(cwd)
	if err != nil {
		return doctorCheck{
			RuleID:  "audit.runtime_evidence",
			Name:    "Runtime evidence",
			Status:  "fail",
			Summary: fmt.Sprintf("Could not collect runtime evidence artifacts: %v", err),
			Hint:    "Fix unreadable files or directory permissions before rerunning `gwc doctor -audit`.",
		}
	}
	if len(startupReports) == 0 && len(wasmFiles) == 0 {
		return doctorCheck{
			Name:    "Runtime evidence",
			Status:  "pass",
			Summary: "No local runtime evidence artifacts were found yet.",
		}
	}
	warnings := []string{}
	summaries := []string{}
	locations := []string{}
	if len(wasmFiles) > 0 {
		wasm := wasmFiles[0]
		locations = appendDoctorLocation(locations, wasm.RelPath)
		summaries = append(summaries, fmt.Sprintf("wasm %s (%s)", wasm.RelPath, formatBytesBinary(wasm.SizeBytes)))
		if wasm.SizeBytes > 5*1024*1024 {
			warnings = append(warnings, fmt.Sprintf("%s weighs %s on disk", wasm.RelPath, formatBytesBinary(wasm.SizeBytes)))
		}
	}
	if len(startupReports) > 0 {
		report := startupReports[0]
		locations = appendDoctorLocation(locations, report.RelPath)
		summaries = append(summaries, fmt.Sprintf("startup %s", report.RelPath))
		if report.ReadyMs != nil {
			summaries = append(summaries, fmt.Sprintf("ready=%.0fms", *report.ReadyMs))
			if *report.ReadyMs > 2500 {
				warnings = append(warnings, fmt.Sprintf("%s reports readyMs=%.0f", report.RelPath, *report.ReadyMs))
			}
		}
		if report.InteractionMs != nil {
			summaries = append(summaries, fmt.Sprintf("interaction=%.0fms", *report.InteractionMs))
			if *report.InteractionMs > 500 {
				warnings = append(warnings, fmt.Sprintf("%s reports interactionMs=%.0f", report.RelPath, *report.InteractionMs))
			}
		}
	}
	if len(warnings) == 0 {
		return doctorCheck{
			RuleID:    "audit.runtime_evidence",
			Name:      "Runtime evidence",
			Status:    "pass",
			Summary:   strings.Join(summaries, " | "),
			Locations: locations,
		}
	}
	return doctorCheck{
		RuleID:    "audit.runtime_evidence",
		Name:      "Runtime evidence",
		Status:    "warn",
		Summary:   summarizeDoctorViolations(warnings, 3),
		Locations: locations,
		Hint:      "Collect and track wasm payload size plus startup probe timing so the audit can distinguish cheap shells from expensive startup paths with observed evidence.",
	}
}

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

func collectGoldenPathGoFiles(root string) ([]doctorGoFileRecord, error) {
	records := []doctorGoFileRecord{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDoctorAuditDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".go" {
			return nil
		}
		contentBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(contentBytes)
		file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ImportsOnly|parser.ParseComments)
		if err != nil {
			return err
		}
		imports := make([]string, 0, len(file.Imports))
		for _, spec := range file.Imports {
			imports = append(imports, strings.Trim(spec.Path.Value, `"`))
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		records = append(records, doctorGoFileRecord{
			RelPath:   filepath.ToSlash(relPath),
			Content:   content,
			Imports:   imports,
			Client:    isDoctorAuditClientFile(filepath.ToSlash(path), content),
			SizeBytes: int64(len(contentBytes)),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return records, nil
}

func collectDoctorHTMLFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDoctorAuditDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(entry.Name()) != ".html" {
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		files = append(files, filepath.ToSlash(relPath))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func collectDoctorRuntimeArtifacts(root string) ([]doctorStartupEvidence, []doctorRuntimeArtifact, error) {
	startupReports := []doctorStartupEvidence{}
	wasmFiles := []doctorRuntimeArtifact{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipDoctorRuntimeDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)
		switch {
		case strings.EqualFold(entry.Name(), "wasm-startup-report.json"):
			report, err := readDoctorStartupEvidence(path, relPath)
			if err != nil {
				return err
			}
			startupReports = append(startupReports, report)
		case strings.EqualFold(filepath.Ext(entry.Name()), ".wasm"):
			info, err := entry.Info()
			if err != nil {
				return err
			}
			wasmFiles = append(wasmFiles, doctorRuntimeArtifact{RelPath: relPath, SizeBytes: info.Size()})
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(startupReports, func(i int, j int) bool { return startupReports[i].RelPath < startupReports[j].RelPath })
	sort.Slice(wasmFiles, func(i int, j int) bool {
		if wasmFiles[i].SizeBytes != wasmFiles[j].SizeBytes {
			return wasmFiles[i].SizeBytes > wasmFiles[j].SizeBytes
		}
		return wasmFiles[i].RelPath < wasmFiles[j].RelPath
	})
	return startupReports, wasmFiles, nil
}

func shouldSkipDoctorRuntimeDir(name string) bool {
	switch strings.TrimSpace(name) {
	case ".git", "node_modules", "vendor", "dist", "tmp":
		return true
	default:
		return false
	}
}

func readDoctorStartupEvidence(path string, relPath string) (doctorStartupEvidence, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return doctorStartupEvidence{}, err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(content, &payload); err != nil {
		return doctorStartupEvidence{}, err
	}
	evidence := doctorStartupEvidence{RelPath: relPath}
	if startup, ok := payload["startup"].(map[string]interface{}); ok {
		if readyMs, ok := startup["readyMs"].(float64); ok {
			evidence.ReadyMs = &readyMs
		}
		if interactionMs, ok := startup["interactionMs"].(float64); ok {
			evidence.InteractionMs = &interactionMs
		}
	}
	return evidence, nil
}

func shouldSkipDoctorAuditDir(name string) bool {
	switch strings.TrimSpace(name) {
	case ".git", "node_modules", "vendor", "bin", "dist", "tmp":
		return true
	default:
		return false
	}
}

func isDoctorAuditClientFile(path string, content string) bool {
	if strings.Contains(content, "//go:build js && wasm") {
		return true
	}
	return strings.Contains(path, "/client/")
}

func summarizeDoctorViolations(violations []string, limit int) string {
	if len(violations) == 0 {
		return ""
	}
	if limit <= 0 || len(violations) <= limit {
		return strings.Join(violations, " | ")
	}
	return fmt.Sprintf("%s | +%d more", strings.Join(violations[:limit], " | "), len(violations)-limit)
}

func formatBytesBinary(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	kib := float64(size) / 1024
	if kib < 1024 {
		return fmt.Sprintf("%.1f KiB", kib)
	}
	return fmt.Sprintf("%.1f MiB", kib/1024)
}

func loadDoctorAuditBaseline(path string) (doctorAuditBaseline, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return doctorAuditBaseline{}, fmt.Errorf("read audit baseline: %w", err)
	}
	var baseline doctorAuditBaseline
	if err := json.Unmarshal(content, &baseline); err != nil {
		return doctorAuditBaseline{}, fmt.Errorf("parse audit baseline: %w", err)
	}
	return baseline, nil
}

func writeDoctorAuditBaseline(path string, audit doctorAuditReport) error {
	resolvedPath := path
	if cwd, err := doctorGetwd(); err == nil {
		if normalized, normalizeErr := normalizePath(cwd, path); normalizeErr == nil && strings.TrimSpace(normalized) != "" {
			resolvedPath = normalized
		}
	}
	if dir := filepath.Dir(resolvedPath); strings.TrimSpace(dir) != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create audit baseline directory: %w", err)
		}
	}
	baseline := doctorAuditBaseline{
		Mode:        audit.Mode,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Checks:      make([]doctorAuditBaselineCheck, 0, len(audit.Checks)),
	}
	for _, check := range audit.Checks {
		if check.Status == "pass" {
			continue
		}
		baseline.Checks = append(baseline.Checks, doctorAuditBaselineCheck{
			Name:    check.Name,
			Status:  check.Status,
			Summary: check.Summary,
		})
	}
	content, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal audit baseline: %w", err)
	}
	if err := os.WriteFile(resolvedPath, append(content, '\n'), 0644); err != nil {
		return fmt.Errorf("write audit baseline: %w", err)
	}
	return nil
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
	if report.Audit != nil {
		auditStatus := "PASS"
		if !report.Audit.OK {
			auditStatus = "FAIL"
		}
		fmt.Printf("  audit[%s]: %s", report.Audit.Mode, auditStatus)
		if report.Audit.Policy != "" {
			fmt.Printf(" (policy: %s)", report.Audit.Policy)
		}
		fmt.Println()
		if strings.TrimSpace(report.Audit.BaselinePath) != "" {
			fmt.Printf("    baseline: %s\n", report.Audit.BaselinePath)
		}
		if len(report.Audit.Suppressed) > 0 {
			fmt.Printf("    suppressed: %s\n", strings.Join(report.Audit.Suppressed, ", "))
		}
		for _, check := range report.Audit.Checks {
			label := strings.ToUpper(check.Status)
			fmt.Printf("    [%s] %s: %s\n", label, check.Name, check.Summary)
			if strings.TrimSpace(check.Hint) != "" && check.Status != "pass" {
				fmt.Printf("           hint: %s\n", check.Hint)
			}
		}
	}
	printResolutionTrace(report.Resolution, []string{"app", "root", "html", "host", "port"}, "  ")
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
		gwchtml.Tag("title", gwchtml.Props{}, gwchtml.Text(document.Title)),
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
