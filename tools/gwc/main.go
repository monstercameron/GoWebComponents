package main

import (
	"bytes"
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

	buildLauncherResultEnvelopeSchemaVersion = "gwc.agentic.v1"
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

var buildExecuteBuild = executeBuild

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

var testAllLanes = []string{"unit", "race", "wasm", "hydration", "browser", "perf", "i18n", "agent", "agent-browser", "release"}

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

// launcherCommandMetadata describes one command in the gwc registry.
type launcherCommandMetadata struct {
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases,omitempty"`
	Summary     string   `json:"summary"`
	JSON        bool     `json:"json"`
	Mutating    bool     `json:"mutating"`
	LongRunning bool     `json:"longRunning,omitempty"`
}

// launcherEnvelopeDiagnostic is the diagnostic shape used by gwc JSON envelopes.
type launcherEnvelopeDiagnostic struct {
	Phase    string `json:"phase,omitempty"`
	Category string `json:"category,omitempty"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	FixHint  string `json:"fixHint,omitempty"`
	Override string `json:"override,omitempty"`
}

// launcherEnvelopeError is the error shape used by gwc JSON envelopes.
type launcherEnvelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

var errLauncherJSONEnvelopeWritten = errors.New("gwc json envelope written")

type launcherJSONEnvelopeWrittenError struct {
	err error
}

// Error returns the wrapped command failure message.
func (parseErr launcherJSONEnvelopeWrittenError) Error() string {
	if parseErr.err == nil {
		return errLauncherJSONEnvelopeWritten.Error()
	}
	return parseErr.err.Error()
}

// Unwrap returns the underlying command failure.
func (parseErr launcherJSONEnvelopeWrittenError) Unwrap() error {
	return parseErr.err
}

// Is reports whether the error marks an already-written JSON envelope.
func (parseErr launcherJSONEnvelopeWrittenError) Is(parseTarget error) bool {
	return parseTarget == errLauncherJSONEnvelopeWritten
}

// listLauncherCommandRegistry returns the command metadata that backs JSON support checks.
func listLauncherCommandRegistry() []launcherCommandMetadata {
	return []launcherCommandMetadata{
		{Name: "bench", Aliases: []string{"benchmark"}, Summary: "Discover native/js-wasm benchmark packages, capture raw benchmark output, compare files with benchstat, and write docs/benchmarks JSON output.", JSON: true},
		{Name: "bootstrap", Summary: "Run prerequisite checks, then start a scaffold or examples bootstrap flow.", JSON: true, Mutating: true, LongRunning: true},
		{Name: "build", Summary: "Build a js/wasm app with an explicit launcher profile.", JSON: true, Mutating: true},
		{Name: "check", Summary: "Run agent-shaped diagnostics across tests and source conventions.", JSON: true},
		{Name: "clean", Summary: "Remove launcher-owned build artifacts, caches, and generated outputs with dry-run support.", JSON: true, Mutating: true},
		{Name: "crash-report", Summary: "Fetch the latest live agent bridge crash report for a session.", JSON: true},
		{Name: "dashboard", Summary: "Monitor live-reload clients and project AI provider configuration from a launcher-owned dashboard.", JSON: true},
		{Name: "deadcode", Summary: "Report exported symbols with no static in-repo dependents.", JSON: true},
		{Name: "deploy", Summary: "Package validated release artifacts through explicit deployment adapters.", JSON: true, Mutating: true},
		{Name: "dev", Summary: "Run the native gwc dev orchestration path with integrated livereload runtime.", JSON: true, Mutating: true, LongRunning: true},
		{Name: "deps", Aliases: []string{"update"}, Summary: "Inspect Go module dependencies and apply guarded dependency updates.", JSON: true, Mutating: true},
		{Name: "delete-atom", Summary: "Delete a live agent bridge atom when safe or explicitly forced.", JSON: true, Mutating: true},
		{Name: "doctor", Summary: "Check local toolchains, runtime assets, project signals, and optional golden-path audit anchors.", JSON: true},
		{Name: "docs", Summary: "Generate exported API documentation from the static GWC model.", JSON: true, Mutating: true},
		{Name: "emit", Summary: "Emit a live agent bridge event handler payload.", JSON: true, Mutating: true},
		{Name: "env", Summary: "Print launcher-relevant environment variables and current values.", JSON: true},
		{Name: "examples", Summary: "Serve the examples catalog or run managed example-server lifecycle actions.", JSON: true, Mutating: true, LongRunning: true},
		{Name: "export", Summary: "Alias for prerender.", JSON: true, Mutating: true},
		{Name: "export-test", Summary: "Generate a testkit Go test from a live-session recording.", JSON: true, Mutating: true},
		{Name: "files", Summary: "List project files with repeatable extension and directory filters.", JSON: true},
		{Name: "fmt", Summary: "Format Go source, normalize line endings, and report/fix GWC doc-comment conventions.", JSON: true, Mutating: true},
		{Name: "help", Aliases: []string{"-h", "--help"}, Summary: "Print human help or structured command metadata.", JSON: true},
		{Name: "import", Summary: "Convert a static HTML or JSX file into an inspectable GWC project.", JSON: true, Mutating: true},
		{Name: "init", Summary: "Non-interactive project initialization that writes gwc-start.json and lifecycle defaults.", JSON: true, Mutating: true},
		{Name: "inspect", Summary: "Build higher-level route, dependency, ownership, and file-type project reports.", JSON: true},
		{Name: "lint", Aliases: []string{"review"}, Summary: "Run golangci-lint plus built-in GWC hook rules, then render a text or JSON review report.", JSON: true},
		{Name: "lease", Summary: "Acquire, release, or explicitly steal a live agent bridge write lease.", JSON: true, Mutating: true},
		{Name: "logs", Summary: "Fetch retained live agent bridge logs and diagnostics.", JSON: true},
		{Name: "mcp", Summary: "Serve JSON-capable gwc commands as MCP tools over stdio, or print the tool manifest as JSON.", JSON: true},
		{Name: "migrate", Summary: "Non-interactive migration helper with compatibility API findings, safe rewrites, and report export.", JSON: true, Mutating: true},
		{Name: "model", Summary: "Emit a static component manifest for agent planning.", JSON: true},
		{Name: "mount", Summary: "Mount a bridge-registered component into a live agent session.", JSON: true, Mutating: true},
		{Name: "mutate", Summary: "Apply safe AST-backed source mutations with dry-run and JSON diff output.", JSON: true, Mutating: true},
		{Name: "navigate", Summary: "Navigate a live agent bridge router session.", JSON: true, Mutating: true},
		{Name: "explain", Summary: "Resolve diagnostic error codes and framework capabilities.", JSON: true},
		{Name: "observe", Summary: "Query redacted runtime/crash telemetry streams.", JSON: true},
		{Name: "probe", Summary: "Run a browser-oracle probe for a URL or example target.", JSON: true},
		{Name: "prerender", Summary: "Build one static export output with route HTML, wasm artifacts, and a manifest.", JSON: true, Mutating: true},
		{Name: "publish", Summary: "Publish a live agent bridge topic event.", JSON: true, Mutating: true},
		{Name: "query", Summary: "Query live agent bridge nodes by semantic selectors.", JSON: true},
		{Name: "release", Summary: "Package a js/wasm release with manifest and compressed sidecars.", JSON: true, Mutating: true},
		{Name: "rebuild", Summary: "Rebuild a live agent session and restore state through the hub JSON API.", JSON: true, Mutating: true},
		{Name: "recording", Summary: "Fetch or clear a live agent bridge command recording.", JSON: true, Mutating: true},
		{Name: "render", Summary: "Render a component through the headless SSR oracle.", JSON: true},
		{Name: "scaffold", Summary: "Generate components, hooks, examples, or starter apps without prompts.", JSON: true, Mutating: true},
		{Name: "search", Summary: "Search exported APIs by intent.", JSON: true},
		{Name: "seed", Summary: "Provision local dev identities and fixture data through a seed package.", JSON: true, Mutating: true},
		{Name: "serve", Summary: "Serve a static directory, wasm artifact, wasm_exec.js, and optional JSON fixtures.", JSON: true, LongRunning: true},
		{Name: "sessions", Summary: "List live gwc agent bridge sessions.", JSON: true},
		{Name: "set-atom", Summary: "Set a live agent bridge atom value.", JSON: true, Mutating: true},
		{Name: "set-state", Summary: "Set a live agent bridge hook state slot.", JSON: true, Mutating: true},
		{Name: "size", Summary: "Attribute wasm/native artifact size by package and symbol using go tool nm.", JSON: true},
		{Name: "snapshot", Summary: "Read a live agent bridge runtime snapshot.", JSON: true},
		{Name: "snapshot-diff", Summary: "Diff two agent bridge snapshots by stable ref.", JSON: true},
		{Name: "start", Summary: "Run the scaffold TUI for preset and project setup.", JSON: true, Mutating: true},
		{Name: "tailwind", Summary: "Build shared Tailwind CSS and generated class manifests through the launcher-owned Tailwind path.", JSON: true, Mutating: true},
		{Name: "test", Summary: "Run explicit launcher-owned test lanes such as unit, race, wasm, hydration, browser, agent, agent-browser, and release.", JSON: true},
		{Name: "upgrade", Summary: "Non-interactive lifecycle upgrade for gwc-start.json schema and runtime assets.", JSON: true, Mutating: true},
		{Name: "unmount", Summary: "Unmount a bridge-mounted component from a live agent session.", JSON: true, Mutating: true},
		{Name: "verify", Summary: "Run app-local Go tests when present and perform a CI-profile wasm build.", JSON: true, Mutating: true},
		{Name: "watch", Summary: "Watch Go files and rerun selected launcher-owned test lanes.", JSON: true, LongRunning: true},
		{Name: "wasm", Summary: "Run wasm-focused build experiment helpers such as wasm measure.", JSON: true, Mutating: true},
	}
}

// getLauncherCommandMetadata looks up command metadata by canonical name or alias.
func getLauncherCommandMetadata(parseCommand string) (launcherCommandMetadata, bool) {
	parseCommand = strings.ToLower(strings.TrimSpace(parseCommand))
	if parseCommand == "" {
		return launcherCommandMetadata{}, false
	}
	for _, parseMetadata := range listLauncherCommandRegistry() {
		if parseMetadata.Name == parseCommand {
			return parseMetadata, true
		}
		for _, parseAlias := range parseMetadata.Aliases {
			if strings.EqualFold(strings.TrimSpace(parseAlias), parseCommand) {
				return parseMetadata, true
			}
		}
	}
	return launcherCommandMetadata{}, false
}

// normalizeLauncherCommandName returns the stable registry name for a command token.
func normalizeLauncherCommandName(parseCommand string, parseArgs []string) string {
	if isExamplesManagedPathCommand(parseCommand, parseArgs) {
		return "examples"
	}
	if parseMetadata, parseOk := getLauncherCommandMetadata(parseCommand); parseOk {
		return parseMetadata.Name
	}
	return strings.TrimSpace(parseCommand)
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
	parseMetadata, parseOk := getLauncherCommandMetadata(parseCommand)
	return parseOk && parseMetadata.JSON
}

func launcherJSONRequestedForCommand(parseCommand string, parseArgs []string) bool {
	if isExamplesManagedPathCommand(parseCommand, parseArgs) {
		return launcherJSONRequestedForCommand("examples", buildExamplesManagedPathCommandArgs(parseCommand, parseArgs))
	}
	if !launcherCommandSupportsJSON(parseCommand) {
		return false
	}
	for _, parseArg := range parseArgs {
		if hasLauncherJSONFlag(parseArg) {
			return true
		}
	}
	return false
}

// hasLauncherJSONFlag reports whether one argument explicitly requests JSON output.
func hasLauncherJSONFlag(parseArg string) bool {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseArg))
	switch parseTrimmed {
	case "-json", "--json":
		return true
	}
	for _, parsePrefix := range []string{"-json=", "--json="} {
		if strings.HasPrefix(parseTrimmed, parsePrefix) {
			parseValue := strings.TrimSpace(strings.TrimPrefix(parseTrimmed, parsePrefix))
			return parseValue != "false" && parseValue != "0"
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

// buildLauncherResultEnvelopePayload builds a stable JSON result envelope around command output.
func buildLauncherResultEnvelopePayload(parseCommand string, parseArgs []string, parseOutput string, parseErr error) map[string]any {
	parseData, parseDataObject := decodeLauncherResultData(parseOutput)
	if parseErr == nil && isLauncherResultEnvelopeObject(parseDataObject) {
		return parseDataObject
	}

	isParseOK := parseErr == nil
	if parseDataOK, parseHasOK := parseDataObject["ok"].(bool); parseHasOK && !parseDataOK {
		isParseOK = false
	}
	parsePayload := map[string]any{
		"schemaVersion": buildLauncherResultEnvelopeSchemaVersion,
		"command":       normalizeLauncherCommandName(parseCommand, parseArgs),
		"ok":            isParseOK,
		"data":          parseData,
		"diagnostics":   []launcherEnvelopeDiagnostic{},
		"error":         nil,
	}
	mergeLauncherEnvelopeData(parsePayload, parseDataObject)
	if parseErr != nil {
		parseDiagnostic := buildLauncherFailureDiagnostic(append([]string{parseCommand}, parseArgs...), parseErr)
		parseDiagnostic.Command = normalizeLauncherCommandName(parseCommand, parseArgs)
		parseEnvelopeDiagnostic := launcherEnvelopeDiagnostic{
			Phase:    parseDiagnostic.Phase,
			Category: parseDiagnostic.Category,
			Code:     parseDiagnostic.Code,
			Message:  parseDiagnostic.Message,
			FixHint:  launcherEnvelopeFixHint(parseDiagnostic.Code),
			Override: parseDiagnostic.Override,
		}
		parsePayload["diagnostics"] = []launcherEnvelopeDiagnostic{parseEnvelopeDiagnostic}
		parsePayload["error"] = launcherEnvelopeError{Code: parseDiagnostic.Code, Message: parseDiagnostic.Message}

		// Preserve the previous flat diagnostic shape for existing decoders while
		// agent callers move to diagnostics[] and error.
		parsePayload["phase"] = parseDiagnostic.Phase
		parsePayload["category"] = parseDiagnostic.Category
		parsePayload["code"] = parseDiagnostic.Code
		parsePayload["message"] = parseDiagnostic.Message
		if parseDiagnostic.Override != "" {
			parsePayload["override"] = parseDiagnostic.Override
		}
	}
	return parsePayload
}

// buildLauncherErrorEnvelopePayload builds a stable JSON envelope for errors outside dispatch.
func buildLauncherErrorEnvelopePayload(parseArgs []string, parseErr error) map[string]any {
	parseCommand, parseCommandArgs := launcherCommandAndArgs(parseArgs)
	return buildLauncherResultEnvelopePayload(parseCommand, parseCommandArgs, "", parseErr)
}

// decodeLauncherResultData decodes existing command JSON output into envelope data.
func decodeLauncherResultData(parseOutput string) (any, map[string]any) {
	parseTrimmed := bytes.TrimSpace([]byte(parseOutput))
	if len(parseTrimmed) == 0 {
		parseData := map[string]any{}
		return parseData, parseData
	}

	var parseValue any
	parseDecoder := json.NewDecoder(bytes.NewReader(parseTrimmed))
	parseDecoder.UseNumber()
	if parseErr := parseDecoder.Decode(&parseValue); parseErr == nil {
		parseRemainder := bytes.TrimSpace(parseTrimmed[int(parseDecoder.InputOffset()):])
		if len(parseRemainder) == 0 {
			if parseObject, parseOk := parseValue.(map[string]any); parseOk {
				return parseObject, parseObject
			}
			return parseValue, map[string]any{}
		}
	}
	parseData := map[string]any{"output": strings.TrimRight(parseOutput, "\r\n")}
	return parseData, parseData
}

// isLauncherResultEnvelopeObject reports whether data already has the stable envelope shape.
func isLauncherResultEnvelopeObject(parseObject map[string]any) bool {
	if len(parseObject) == 0 {
		return false
	}
	_, hasSchema := parseObject["schemaVersion"]
	_, hasCommand := parseObject["command"]
	_, hasOK := parseObject["ok"]
	_, hasData := parseObject["data"]
	_, hasDiagnostics := parseObject["diagnostics"]
	_, hasError := parseObject["error"]
	return hasSchema && hasCommand && hasOK && hasData && hasDiagnostics && hasError
}

// mergeLauncherEnvelopeData mirrors object data at top level for older callers.
func mergeLauncherEnvelopeData(parsePayload map[string]any, parseData map[string]any) {
	for parseKey, parseValue := range parseData {
		switch parseKey {
		case "schemaVersion", "command", "data", "diagnostics", "error", "ok":
			continue
		default:
			parsePayload[parseKey] = parseValue
		}
	}
}

// launcherEnvelopeFixHint returns a machine-usable hint for known diagnostic codes.
func launcherEnvelopeFixHint(parseCode string) string {
	switch parseCode {
	case "invalid_configuration":
		return "Check the command flags and any gwc-runner.json path overrides, then retry."
	case "invalid_runner_override":
		return "Fix or remove the configured runner override path."
	case "policy_violation":
		return "Review the active policy pack or choose inputs allowed by policy."
	case "code_failure":
		return "Inspect the command output and fix the reported Go build or test failure."
	case "audit_failed":
		return "Resolve or suppress the audit finding according to the selected audit policy."
	case "checks_failed":
		return "Inspect failed checks and rerun the command after remediation."
	case "smoke_failed":
		return "Inspect the release smoke report and browser startup evidence."
	case "startup_failed":
		return "Check port availability and runtime server configuration."
	default:
		return ""
	}
}

// writeLauncherResultEnvelope writes one indented JSON envelope.
func writeLauncherResultEnvelope(parseW io.Writer, parsePayload map[string]any) error {
	parseEncoder := json.NewEncoder(parseW)
	parseEncoder.SetIndent("", "  ")
	return parseEncoder.Encode(parsePayload)
}

// captureLauncherStdout captures command stdout without risking pipe-buffer deadlocks.
func captureLauncherStdout(parseRun func() error) (string, error) {
	parseOriginalStdout := os.Stdout
	parseTempFile, parseErr := os.CreateTemp("", "gwc-json-stdout-*.tmp")
	if parseErr != nil {
		return "", parseErr
	}
	parseTempPath := parseTempFile.Name()
	defer os.Remove(parseTempPath)

	os.Stdout = parseTempFile
	defer func() {
		os.Stdout = parseOriginalStdout
	}()
	parseRunErr := parseRun()
	parseCloseErr := parseTempFile.Close()

	parseBytes, parseReadErr := os.ReadFile(parseTempPath)
	if parseRunErr != nil {
		return string(parseBytes), parseRunErr
	}
	if parseCloseErr != nil {
		return string(parseBytes), parseCloseErr
	}
	if parseReadErr != nil {
		return string(parseBytes), parseReadErr
	}
	return string(parseBytes), nil
}

func printLauncherError(parseW io.Writer, parseArgs []string, parseErr error) {
	if parseW == nil {
		return
	}
	if launcherRequestedJSONOutput(parseArgs) {
		if parseEncodeErr := writeLauncherResultEnvelope(parseW, buildLauncherErrorEnvelopePayload(parseArgs, parseErr)); parseEncodeErr == nil {
			return
		}
	}
	fmt.Fprintf(parseW, "gwc: %v\n", parseErr)
}

var mainPrintError = func(err error) {
	if launcherRequestedJSONOutput(mainArgs()[1:]) {
		printLauncherError(os.Stdout, mainArgs()[1:], err)
		return
	}
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
		if errors.Is(parseErr2, errLauncherJSONEnvelopeWritten) {
			mainExit(1)
			return
		}
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
		if launcherJSONRequestedForCommand(parseCommand, parseCommandArgs) {
			parseCommand = "help"
			break
		}
		return runHelpCommand(parseL, parseCommandArgs)
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

	if parseJsonOutputRequested {
		parseCapturedOutput, parseDispatchErr := captureLauncherStdout(func() error {
			return parseL.dispatchCommand(parseCommand, parseCommandArgs)
		})
		parsePostHookErr := runLauncherCommandHooks(launcherActiveEnterpriseConfig.Hooks, "post", parseCommand, parseCommandArgs, parseL.repoRoot, launcherActiveEnterpriseSources)
		parseEnvelopeErr := parseDispatchErr
		if parseEnvelopeErr == nil {
			parseEnvelopeErr = parsePostHookErr
		}
		if parseWriteErr := writeLauncherResultEnvelope(os.Stdout, buildLauncherResultEnvelopePayload(parseCommand, parseCommandArgs, parseCapturedOutput, parseEnvelopeErr)); parseWriteErr != nil {
			return parseWriteErr
		}
		if parseEnvelopeErr != nil {
			return launcherJSONEnvelopeWrittenError{err: parseEnvelopeErr}
		}
		return nil
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
	case "check":
		return runCheckCommand(parseL, parseArgs)
	case "clean":
		return runCleanCommand(parseL, parseArgs)
	case "bench", "benchmark":
		return runBenchmarkCommand(parseL, parseArgs)
	case "deadcode":
		return runDeadcodeCommand(parseL, parseArgs)
	case "release":
		return runReleaseCommand(parseL, parseArgs)
	case "deps", "update":
		return runDepsCommand(parseL, parseArgs)
	case "dev":
		return runDevCommand(parseL, parseArgs)
	case "docs":
		return runDocsCommand(parseL, parseArgs)
	case "serve":
		return runServeCommand(parseL, parseArgs)
	case "files":
		return runFilesCommand(parseL, parseArgs)
	case "fmt":
		return runFmtCommand(parseL, parseArgs)
	case "lint", "review":
		return runLintCommand(parseL, parseArgs)
	case "help":
		return runHelpCommand(parseL, parseArgs)
	case "init":
		return runInitCommand(parseL, parseArgs)
	case "inspect":
		if agenticArgsContainFlag(parseArgs, "impact") {
			return runInspectImpactCommand(parseL, parseArgs)
		}
		return runInspectCommand(parseL, parseArgs)
	case "model":
		return runModelCommand(parseL, parseArgs)
	case "render":
		return runRenderCommand(parseL, parseArgs)
	case "probe":
		return runProbeCommand(parseL, parseArgs)
	case "explain":
		return runExplainCommand(parseL, parseArgs)
	case "search":
		return runSearchCommand(parseL, parseArgs)
	case "upgrade":
		return runUpgradeCommand(parseL, parseArgs)
	case "mcp":
		return runMCPCommand(parseL, parseArgs)
	case "rebuild":
		return runRebuildCommand(parseL, parseArgs)
	case "export-test":
		return runExportTestCommand(parseL, parseArgs)
	case "migrate":
		return runMigrateCommand(parseL, parseArgs)
	case "mutate":
		return runMutateCommand(parseL, parseArgs)
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
	case "observe":
		return runObserveCommand(parseL, parseArgs)
	case "verify":
		return runVerifyCommand(parseL, parseArgs)
	case "seed":
		return runSeedCommand(parseL, parseArgs)
	case "size":
		return runSizeCommand(parseL, parseArgs)
	case "screenshot":
		return runScreenshotCommand(parseL, parseArgs)
	case "import":
		return runImportCommand(parseL, parseArgs)
	case "scaffold":
		return runAgenticScaffoldCommand(parseL, parseArgs)
	case "start":
		return runStartCommand(parseL, parseArgs)
	case "bootstrap":
		return runBootstrapCommand(parseL, parseArgs)
	case "wasm":
		return runWasmCommand(parseL, parseArgs)
	case "watch":
		return runWatchCommand(parseL, parseArgs)
	case "sessions", "snapshot", "query", "describe", "wait-for", "audit", "undo", "replay", "render-tree", "set-atom", "set-state", "mount", "unmount", "delete-atom", "emit", "publish", "navigate", "logs", "crash-report", "recording", "lease":
		return runLiveBridgeCommand(parseL, parseCommand, parseArgs)
	case "snapshot-diff":
		return runSnapshotDiffCommand(parseL, parseArgs)
	default:
		if isExamplesManagedPathCommand(parseCommand, parseArgs) {
			return parseL.runExamplesManaged(buildExamplesManagedPathCommandArgs(parseCommand, parseArgs))
		}
		return fmt.Errorf("unknown command %q", parseCommand)
	}
}

func agenticArgsContainFlag(parseArgs []string, parseFlag string) bool {
	parseFlag = strings.TrimLeft(strings.ToLower(strings.TrimSpace(parseFlag)), "-")
	for _, parseArg := range parseArgs {
		parseTrimmed := strings.TrimLeft(strings.ToLower(strings.TrimSpace(parseArg)), "-")
		if parseTrimmed == parseFlag || strings.HasPrefix(parseTrimmed, parseFlag+"=") {
			return true
		}
	}
	return false
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
	fmt.Println("  check      Run agent-shaped diagnostics across tests and source conventions")
	fmt.Println("  clean      Remove launcher-owned build artifacts, caches, and generated outputs")
	fmt.Println("  test       Run explicit launcher-owned test lanes such as unit, race, wasm, hydration, browser, agent, agent-browser, and release")
	fmt.Println("  watch      Watch Go files and rerun selected launcher-owned test lanes")
	fmt.Println("  examples   Serve the examples catalog or run managed example-server lifecycle actions (start|status|stop|restart)")
	fmt.Println("  dev        Run the native gwc dev orchestration path with integrated livereload runtime")
	fmt.Println("  serve      Serve a static directory, wasm artifact, wasm_exec.js, and optional JSON fixtures")
	fmt.Println("  files      List project files with repeatable extension and directory filters")
	fmt.Println("  fmt        Format Go source, normalize line endings, and report/fix GWC doc-comment conventions")
	fmt.Println("  lint       Run golangci-lint plus built-in GWC hook rules, then render a text or JSON review report (review alias supported)")
	fmt.Println("  init       Non-interactive project initialization that writes gwc-start.json and lifecycle defaults")
	fmt.Println("  inspect    Build higher-level route, dependency, ownership, and file-type project reports")
	fmt.Println("  model      Emit a static component manifest for agent planning")
	fmt.Println("  mcp        Serve JSON-capable gwc commands as MCP tools over stdio, or print the tool manifest as JSON")
	fmt.Println("  mutate     Apply safe AST-backed source mutations with dry-run and JSON diff output")
	fmt.Println("  docs       Generate exported API documentation from the static GWC model")
	fmt.Println("  deadcode   Report exported symbols with no static in-repo dependents")
	fmt.Println("  deps       Inspect Go module dependencies and apply guarded dependency updates (update alias supported)")
	fmt.Println("  size       Attribute wasm/native artifact size by package and symbol using go tool nm")
	fmt.Println("  render     Render a component through the headless SSR oracle")
	fmt.Println("  probe      Run a browser-oracle probe for a URL or example target")
	fmt.Println("  explain    Resolve diagnostic error codes and framework capabilities")
	fmt.Println("  search     Search exported APIs by intent")
	fmt.Println("  observe    Query redacted runtime/crash telemetry streams")
	fmt.Println("  upgrade    Non-interactive lifecycle upgrade for gwc-start.json schema and runtime assets")
	fmt.Println("  migrate    Non-interactive migration helper with compatibility API findings, safe rewrites, and report export")
	fmt.Println("  prerender  Build one static export output with route HTML, wasm artifacts, and a manifest")
	fmt.Println("  export     Alias for `prerender`")
	fmt.Println("  export-test Generate a testkit Go test from a live-session recording")
	fmt.Println("  tailwind   Build shared Tailwind CSS and generated class manifests through the launcher-owned Tailwind path")
	fmt.Println("  dashboard  Monitor live-reload clients and project AI provider configuration from a launcher-owned dashboard")
	fmt.Println("  doctor     Check local toolchains, runtime assets, project signals, and optional golden-path audit anchors")
	fmt.Println("  deploy     Package validated release artifacts through explicit deployment adapters")
	fmt.Println("  env        Print launcher-relevant environment variables and current values")
	fmt.Println("  seed       Provision local dev identities and fixture data through a seed package")
	fmt.Println("  import     Convert a static HTML or JSX file into an inspectable GWC project")
	fmt.Println("  release    Package a js/wasm release with manifest and compressed sidecars")
	fmt.Println("  rebuild    Rebuild a live agent session and restore state through the hub JSON API")
	fmt.Println("  scaffold   Generate components, hooks, examples, or starter apps without prompts")
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
