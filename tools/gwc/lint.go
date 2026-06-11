package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const installLintTarget = "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"

// runLintCommand runs the lint launcher subcommand.
var runLintCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runLint(parseArgs)
}

// resolveLintExecutable resolves the golangci-lint executable path.
var resolveLintExecutable = exec.LookPath

// runLintProcess executes golangci-lint and captures stdout and stderr separately.
var runLintProcess = executeLintProcess

// getLintWorkingDir returns the current working directory for lint resolution.
var getLintWorkingDir = os.Getwd

// installLintExecutable installs golangci-lint through go install when it is missing.
var installLintExecutable = func(parseRootPath string) error {
	_, parseErr := launcherRunCommand("go", []string{"install", installLintTarget}, parseRootPath, buildNativeGoEnv())
	if parseErr != nil {
		return fmt.Errorf("go install %s: %w", installLintTarget, parseErr)
	}
	return nil
}

var parseLintVersionPattern = regexp.MustCompile(`v?(\d+)\.(\d+)(?:\.\d+)?`)

type lintConfig struct {
	rootPath       string
	toolPath       string
	configPath     string
	reportPath     string
	paths          []string
	timeout        string
	enableLinters  []string
	disableLinters []string
	noConfig       bool
	fastOnly       bool
	skipHookRules  bool
	json           bool
	resolution     map[string]string
}

type lintProcessResult struct {
	stdout   string
	stderr   string
	exitCode int
}

type lintIssueRecord struct {
	Linter     string `json:"linter"`
	Severity   string `json:"severity,omitempty"`
	Path       string `json:"path,omitempty"`
	Line       int    `json:"line,omitempty"`
	Column     int    `json:"column,omitempty"`
	Message    string `json:"message"`
	SourceLine string `json:"sourceLine,omitempty"`
}

type lintSummary struct {
	OK             bool              `json:"ok"`
	Tool           string            `json:"tool"`
	ToolPath       string            `json:"toolPath"`
	ToolVersion    string            `json:"toolVersion,omitempty"`
	Command        string            `json:"command"`
	ProjectRoot    string            `json:"projectRoot"`
	Paths          []string          `json:"paths"`
	ConfigPath     string            `json:"configPath,omitempty"`
	NoConfig       bool              `json:"noConfig,omitempty"`
	HookRules      bool              `json:"hookRules"`
	ReportPath     string            `json:"reportPath,omitempty"`
	DurationMs     int64             `json:"durationMs"`
	IssueCount     int               `json:"issueCount"`
	Issues         []lintIssueRecord `json:"issues"`
	LinterCounts   map[string]int    `json:"linterCounts,omitempty"`
	SeverityCounts map[string]int    `json:"severityCounts,omitempty"`
	Resolution     map[string]string `json:"resolution,omitempty"`
}

// runLint runs the lint launcher subcommand.
func (parseL launcher) runLint(parseArgs []string) error {
	parseFs := flag.NewFlagSet("lint", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseRoot := parseFs.String("root", "", "Project root to lint; defaults to the current working directory")
	parseTool := parseFs.String("tool", "golangci-lint", "golangci-lint executable name or absolute path")
	parseConfigPath := parseFs.String("config", "", "Optional golangci-lint config file path")
	parseNoConfig := parseFs.Bool("no-config", false, "Disable golangci-lint config discovery")
	parseReportPath := parseFs.String("out", "", "Optional path to write the rendered report")
	parseTimeout := parseFs.String("timeout", "", "Optional golangci-lint timeout such as 2m or 30s")
	parseFastOnly := parseFs.Bool("fast-only", false, "Run only fast linters")
	parseSkipHookRules := parseFs.Bool("skip-hook-rules", false, "Disable built-in GWC hook call-order checks")
	parseJSON := parseFs.Bool("json", false, "Emit a machine-readable JSON report")
	var parsePaths stringListFlag
	var parseEnableLinters stringListFlag
	var parseDisableLinters stringListFlag
	parseFs.Var(&parsePaths, "path", "Package or file path to lint; repeat or comma-separate (default ./...)")
	parseFs.Var(&parseEnableLinters, "enable", "Enable a golangci-lint linter; repeat or comma-separate")
	parseFs.Var(&parseDisableLinters, "disable", "Disable a golangci-lint linter; repeat or comma-separate")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr := parseLintConfig(lintConfig{
		rootPath:       *parseRoot,
		toolPath:       *parseTool,
		configPath:     *parseConfigPath,
		reportPath:     *parseReportPath,
		paths:          parsePaths.Values(),
		timeout:        *parseTimeout,
		enableLinters:  parseEnableLinters.Values(),
		disableLinters: parseDisableLinters.Values(),
		noConfig:       *parseNoConfig,
		fastOnly:       *parseFastOnly,
		skipHookRules:  *parseSkipHookRules,
		json:           *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	if parseErr2 := enforceEnterpriseGoToolchainPolicy(parseConfig.rootPath, launcherActiveEnterpriseConfig.Policy); parseErr2 != nil {
		return parseErr2
	}

	parseSummary, isParseIssuesFound, parseErr := buildLintSummary(parseConfig)
	if parseErr != nil {
		return parseErr
	}
	parseSummary.ReportPath = parseConfig.reportPath

	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseErr2 := parseEncoder.Encode(parseSummary); parseErr2 != nil {
			return parseErr2
		}
		if parseConfig.reportPath != "" {
			parseReportBytes, parseErr3 := json.MarshalIndent(parseSummary, "", "  ")
			if parseErr3 != nil {
				return parseErr3
			}
			parseReportBytes = append(parseReportBytes, '\n')
			if parseErr4 := storeLintReport(parseConfig.reportPath, parseReportBytes); parseErr4 != nil {
				return parseErr4
			}
		}
	} else {
		parseReportText := formatLintReport(parseSummary)
		fmt.Print(parseReportText)
		if parseConfig.reportPath != "" {
			if parseErr2 := storeLintReport(parseConfig.reportPath, []byte(parseReportText)); parseErr2 != nil {
				return parseErr2
			}
		}
	}

	if isParseIssuesFound {
		return fmt.Errorf("lint found %d issue(s)", parseSummary.IssueCount)
	}
	return nil
}

// parseLintConfig normalizes lint command inputs into executable paths and defaults.
func parseLintConfig(parseConfig lintConfig) (lintConfig, error) {
	parseResolved := parseConfig
	parseResolved.resolution = cloneResolutionTrace(parseConfig.resolution)

	parseWorkingDir, parseErr := getLintWorkingDir()
	if parseErr != nil {
		return lintConfig{}, fmt.Errorf("resolve lint working directory: %w", parseErr)
	}

	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseRootPath = parseWorkingDir
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "convention fallback")
	} else {
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "root", "explicit flag")
	}
	parseAbsoluteRoot, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return lintConfig{}, fmt.Errorf("resolve lint root: %w", parseErr)
	}
	parseRootInfo, parseErr := os.Stat(parseAbsoluteRoot)
	if parseErr != nil {
		return lintConfig{}, fmt.Errorf("stat lint root: %w", parseErr)
	}
	if !parseRootInfo.IsDir() {
		return lintConfig{}, fmt.Errorf("lint root is not a directory: %s", parseAbsoluteRoot)
	}
	parseResolved.rootPath = parseAbsoluteRoot

	parseResolved.toolPath = strings.TrimSpace(parseConfig.toolPath)
	if parseResolved.toolPath == "" {
		parseResolved.toolPath = "golangci-lint"
	}

	if parseConfig.noConfig && strings.TrimSpace(parseConfig.configPath) != "" {
		return lintConfig{}, errors.New("use either -config or -no-config")
	}
	parseResolved.noConfig = parseConfig.noConfig

	if strings.TrimSpace(parseConfig.configPath) != "" {
		parseAbsoluteConfig, parseErr := filepath.Abs(strings.TrimSpace(parseConfig.configPath))
		if parseErr != nil {
			return lintConfig{}, fmt.Errorf("resolve lint config path: %w", parseErr)
		}
		if _, parseErr2 := os.Stat(parseAbsoluteConfig); parseErr2 != nil {
			return lintConfig{}, fmt.Errorf("stat lint config path: %w", parseErr2)
		}
		parseResolved.configPath = parseAbsoluteConfig
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "config", "explicit flag")
	}

	if strings.TrimSpace(parseConfig.reportPath) != "" {
		parseAbsoluteReport, parseErr := filepath.Abs(strings.TrimSpace(parseConfig.reportPath))
		if parseErr != nil {
			return lintConfig{}, fmt.Errorf("resolve lint report path: %w", parseErr)
		}
		parseResolved.reportPath = parseAbsoluteReport
		parseResolved.resolution = setResolutionSource(parseResolved.resolution, "report", "explicit flag")
	}

	parseResolved.paths = parseLintPaths(parseConfig.paths)
	parseResolved.timeout = strings.TrimSpace(parseConfig.timeout)
	parseResolved.enableLinters = parseLintList(parseConfig.enableLinters)
	parseResolved.disableLinters = parseLintList(parseConfig.disableLinters)
	parseResolved.fastOnly = parseConfig.fastOnly
	parseResolved.skipHookRules = parseConfig.skipHookRules
	parseResolved.json = parseConfig.json
	return parseResolved, nil
}

// parseLintPaths normalizes lint target paths and preserves input order.
func parseLintPaths(parsePaths []string) []string {
	if len(parsePaths) == 0 {
		return []string{"./..."}
	}
	parseSeen := map[string]struct{}{}
	parseNormalized := make([]string, 0, len(parsePaths))
	for _, parsePath := range parsePaths {
		parseTrimmed := strings.TrimSpace(parsePath)
		if parseTrimmed == "" {
			continue
		}
		if _, isParseSeen := parseSeen[parseTrimmed]; isParseSeen {
			continue
		}
		parseSeen[parseTrimmed] = struct{}{}
		parseNormalized = append(parseNormalized, parseTrimmed)
	}
	if len(parseNormalized) == 0 {
		return []string{"./..."}
	}
	return parseNormalized
}

// parseLintList normalizes repeated linter name flags while preserving order.
func parseLintList(parseValues []string) []string {
	parseSeen := map[string]struct{}{}
	parseNormalized := make([]string, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseTrimmed := strings.TrimSpace(parseValue)
		if parseTrimmed == "" {
			continue
		}
		if _, isParseSeen := parseSeen[parseTrimmed]; isParseSeen {
			continue
		}
		parseSeen[parseTrimmed] = struct{}{}
		parseNormalized = append(parseNormalized, parseTrimmed)
	}
	return parseNormalized
}

// executeLintProcess executes an external lint command and captures stdout and stderr.
func executeLintProcess(parseCommand string, parseArgs []string, parseWorkingDir string, parseEnv []string) (lintProcessResult, error) {
	parseCmd := exec.Command(parseCommand, parseArgs...)
	parseCmd.Dir = parseWorkingDir
	parseCmd.Env = parseEnv
	var parseStdout bytes.Buffer
	var parseStderr bytes.Buffer
	parseCmd.Stdout = &parseStdout
	parseCmd.Stderr = &parseStderr
	parseErr := parseCmd.Run()
	parseResult := lintProcessResult{
		stdout: strings.TrimSpace(parseStdout.String()),
		stderr: strings.TrimSpace(parseStderr.String()),
	}
	if parseErr == nil {
		return parseResult, nil
	}
	var parseExitErr *exec.ExitError
	if errors.As(parseErr, &parseExitErr) {
		parseResult.exitCode = parseExitErr.ExitCode()
		return parseResult, nil
	}
	return parseResult, fmt.Errorf("run %s: %w", parseCommand, parseErr)
}

// buildLintSummary runs golangci-lint and translates its output into a launcher report.
func buildLintSummary(parseConfig lintConfig) (lintSummary, bool, error) {
	parseExecutablePath, parseErr := resolveLintExecutablePath(parseConfig)
	if parseErr != nil {
		return lintSummary{}, false, parseErr
	}
	parseToolVersion, parseMajorVersion := parseLintVersion(parseExecutablePath, parseConfig.rootPath)
	parseCommandArgs := buildLintCommandArgs(parseConfig, parseMajorVersion)
	parseStartTime := time.Now()
	parseResult, parseErr := runLintProcess(parseExecutablePath, parseCommandArgs, parseConfig.rootPath, buildNativeGoEnv())
	if parseErr != nil {
		return lintSummary{}, false, parseErr
	}
	parseDuration := time.Since(parseStartTime)
	if parseResult.exitCode != 0 && parseResult.exitCode != 1 {
		return lintSummary{}, false, buildLintExecutionError(parseResult, parseExecutablePath, parseCommandArgs)
	}

	parseIssues, parseErr := parseLintIssues(parseResult.stdout, parseConfig.rootPath)
	if parseErr != nil {
		return lintSummary{}, false, fmt.Errorf("parse golangci-lint JSON output: %w", parseErr)
	}
	if !parseConfig.skipHookRules {
		parseHookIssues, parseErr2 := collectLintHookRuleIssues(parseConfig.rootPath, parseConfig.paths)
		if parseErr2 != nil {
			return lintSummary{}, false, parseErr2
		}
		parseIssues = append(parseIssues, parseHookIssues...)
		sortLintIssues(parseIssues)
	}
	parseConfigPath := parseLintActiveConfigPath(parseConfig, parseExecutablePath, parseMajorVersion)
	parseLinterCounts, parseSeverityCounts := buildLintCounts(parseIssues)
	parseSummary := lintSummary{
		OK:             parseResult.exitCode == 0 && len(parseIssues) == 0,
		Tool:           "golangci-lint",
		ToolPath:       parseExecutablePath,
		ToolVersion:    parseToolVersion,
		Command:        buildLintCommandString(parseExecutablePath, parseCommandArgs),
		ProjectRoot:    parseConfig.rootPath,
		Paths:          append([]string(nil), parseConfig.paths...),
		ConfigPath:     parseConfigPath,
		NoConfig:       parseConfig.noConfig,
		HookRules:      !parseConfig.skipHookRules,
		DurationMs:     parseDuration.Milliseconds(),
		IssueCount:     len(parseIssues),
		Issues:         parseIssues,
		LinterCounts:   parseLinterCounts,
		SeverityCounts: parseSeverityCounts,
		Resolution:     cloneResolutionTrace(parseConfig.resolution),
	}
	return parseSummary, parseResult.exitCode == 1 || len(parseIssues) > 0, nil
}

// resolveLintExecutablePath resolves golangci-lint and installs it when the default tool is missing.
func resolveLintExecutablePath(parseConfig lintConfig) (string, error) {
	parseExecutablePath, parseErr := resolveLintExecutable(parseConfig.toolPath)
	if parseErr == nil {
		return parseExecutablePath, nil
	}
	if !parseLintInstallableTool(parseConfig.toolPath) {
		return "", fmt.Errorf("resolve golangci-lint executable: %w", parseErr)
	}
	if parseErr2 := installLintExecutable(parseConfig.rootPath); parseErr2 != nil {
		return "", fmt.Errorf("install golangci-lint executable: %w", parseErr2)
	}
	parseExecutablePath, parseErr = resolveLintExecutable(parseConfig.toolPath)
	if parseErr == nil {
		return parseExecutablePath, nil
	}
	parseInstalledPath, isParseFound, parseErr2 := buildLintInstalledExecutablePath(parseConfig.rootPath, parseConfig.toolPath)
	if parseErr2 != nil {
		return "", parseErr2
	}
	if isParseFound {
		return parseInstalledPath, nil
	}
	return "", fmt.Errorf("resolve golangci-lint executable: %w", parseErr)
}

// parseLintInstallableTool reports whether a missing tool can be satisfied with go install.
func parseLintInstallableTool(parseToolPath string) bool {
	parseTrimmed := strings.TrimSpace(parseToolPath)
	if parseTrimmed == "" {
		return true
	}
	if filepath.Base(parseTrimmed) != parseTrimmed {
		return false
	}
	parseLower := strings.ToLower(strings.TrimSuffix(parseTrimmed, filepath.Ext(parseTrimmed)))
	return parseLower == "golangci-lint"
}

// buildLintInstalledExecutablePath locates the installed golangci-lint binary in GOBIN or GOPATH/bin.
func buildLintInstalledExecutablePath(parseRootPath string, parseToolPath string) (string, bool, error) {
	parseOutput, parseErr := launcherRunCommand("go", []string{"env", "-json", "GOBIN", "GOPATH", "GOEXE"}, parseRootPath, buildNativeGoEnv())
	if parseErr != nil {
		return "", false, fmt.Errorf("resolve golangci-lint install location: %w", parseErr)
	}
	var parseEnv struct {
		GOBIN  string
		GOPATH string
		GOEXE  string
	}
	if parseErr2 := json.Unmarshal([]byte(parseOutput), &parseEnv); parseErr2 != nil {
		return "", false, fmt.Errorf("parse golangci-lint install location: %w", parseErr2)
	}
	parseBinDir := strings.TrimSpace(parseEnv.GOBIN)
	if parseBinDir == "" {
		parseGOPATHEntries := filepath.SplitList(parseEnv.GOPATH)
		if len(parseGOPATHEntries) > 0 && strings.TrimSpace(parseGOPATHEntries[0]) != "" {
			parseBinDir = filepath.Join(parseGOPATHEntries[0], "bin")
		}
	}
	if parseBinDir == "" {
		return "", false, nil
	}
	parseExecutableName := filepath.Base(strings.TrimSpace(parseToolPath))
	if parseExecutableName == "" {
		parseExecutableName = "golangci-lint"
	}
	if parseEnv.GOEXE != "" && filepath.Ext(parseExecutableName) == "" {
		parseExecutableName += parseEnv.GOEXE
	}
	parseInstalledPath := filepath.Join(parseBinDir, parseExecutableName)
	if !fileExists(parseInstalledPath) {
		return "", false, nil
	}
	return parseInstalledPath, true, nil
}

// parseLintVersion queries golangci-lint for its version and extracts the major version.
func parseLintVersion(parseExecutablePath string, parseRootPath string) (string, int) {
	for _, parseArgs := range [][]string{
		{"version", "--short"},
		{"version"},
		{"--version"},
	} {
		parseResult, parseErr := runLintProcess(parseExecutablePath, parseArgs, parseRootPath, buildNativeGoEnv())
		if parseErr != nil || parseResult.exitCode != 0 {
			continue
		}
		parseVersionText, parseMajorVersion := parseLintVersionText(parseResult.stdout)
		if parseVersionText != "" {
			return parseVersionText, parseMajorVersion
		}
	}
	return "", 2
}

// parseLintVersionText extracts a semantic version and its major number from version output.
func parseLintVersionText(parseOutput string) (string, int) {
	parseMatch := parseLintVersionPattern.FindStringSubmatch(strings.TrimSpace(parseOutput))
	if len(parseMatch) < 3 {
		return strings.TrimSpace(parseOutput), 2
	}
	parseMajorVersion, parseErr := strconv.Atoi(parseMatch[1])
	if parseErr != nil {
		return parseMatch[0], 2
	}
	return strings.TrimPrefix(parseMatch[0], "v"), parseMajorVersion
}

// buildLintCommandArgs builds golangci-lint arguments for either the v1 or v2 CLI surface.
func buildLintCommandArgs(parseConfig lintConfig, parseMajorVersion int) []string {
	if parseMajorVersion <= 1 {
		return buildLintCommandArgsV1(parseConfig)
	}
	return buildLintCommandArgsV2(parseConfig)
}

// buildLintCommandArgsV2 builds golangci-lint v2 run arguments with JSON on stdout.
func buildLintCommandArgsV2(parseConfig lintConfig) []string {
	parseArgs := []string{"run", "--issues-exit-code=1", "--show-stats=false", "--output.json.path=stdout", "--output.text.path=stderr"}
	if parseConfig.noConfig {
		parseArgs = append(parseArgs, "--no-config")
	}
	if parseConfig.configPath != "" {
		parseArgs = append(parseArgs, "--config", parseConfig.configPath)
	}
	if parseConfig.fastOnly {
		parseArgs = append(parseArgs, "--fast-only")
	}
	if parseConfig.timeout != "" {
		parseArgs = append(parseArgs, "--timeout", parseConfig.timeout)
	}
	for _, parseLinter := range parseConfig.enableLinters {
		parseArgs = append(parseArgs, "--enable", parseLinter)
	}
	for _, parseLinter := range parseConfig.disableLinters {
		parseArgs = append(parseArgs, "--disable", parseLinter)
	}
	parseArgs = append(parseArgs, parseConfig.paths...)
	return parseArgs
}

// buildLintCommandArgsV1 builds golangci-lint v1 run arguments with JSON on stdout.
func buildLintCommandArgsV1(parseConfig lintConfig) []string {
	parseArgs := []string{"run", "--issues-exit-code=1", "--show-stats=false", "--out-format=json:stdout,line-number:stderr", "--print-issued-lines=false", "--print-linter-name=false"}
	if parseConfig.noConfig {
		parseArgs = append(parseArgs, "--no-config")
	}
	if parseConfig.configPath != "" {
		parseArgs = append(parseArgs, "--config", parseConfig.configPath)
	}
	if parseConfig.fastOnly {
		parseArgs = append(parseArgs, "--fast")
	}
	if parseConfig.timeout != "" {
		parseArgs = append(parseArgs, "--timeout", parseConfig.timeout)
	}
	for _, parseLinter := range parseConfig.enableLinters {
		parseArgs = append(parseArgs, "--enable", parseLinter)
	}
	for _, parseLinter := range parseConfig.disableLinters {
		parseArgs = append(parseArgs, "--disable", parseLinter)
	}
	parseArgs = append(parseArgs, parseConfig.paths...)
	return parseArgs
}

// parseLintActiveConfigPath resolves the active golangci-lint config path when available.
func parseLintActiveConfigPath(parseConfig lintConfig, parseExecutablePath string, parseMajorVersion int) string {
	if parseConfig.noConfig {
		return ""
	}
	if parseConfig.configPath != "" {
		return parseConfig.configPath
	}
	if parseMajorVersion <= 1 {
		return ""
	}
	parseArgs := []string{"config", "path"}
	parseResult, parseErr := runLintProcess(parseExecutablePath, parseArgs, parseConfig.rootPath, buildNativeGoEnv())
	if parseErr != nil || parseResult.exitCode != 0 {
		return ""
	}
	parseConfigPath := strings.TrimSpace(parseResult.stdout)
	if parseConfigPath == "" {
		return ""
	}
	parseAbsolutePath, parseErr := filepath.Abs(parseConfigPath)
	if parseErr != nil {
		return parseConfigPath
	}
	return parseAbsolutePath
}

// buildLintExecutionError formats a non-issue golangci-lint process failure.
func buildLintExecutionError(parseResult lintProcessResult, parseExecutablePath string, parseArgs []string) error {
	parseParts := []string{}
	if parseResult.stderr != "" {
		parseParts = append(parseParts, parseResult.stderr)
	}
	if parseResult.stdout != "" {
		parseParts = append(parseParts, parseResult.stdout)
	}
	parseMessage := strings.TrimSpace(strings.Join(parseParts, "\n"))
	if parseMessage == "" {
		parseMessage = "no diagnostic output"
	}
	return fmt.Errorf("%s failed with exit code %d: %s", buildLintCommandString(parseExecutablePath, parseArgs), parseResult.exitCode, parseMessage)
}

// buildLintCommandString renders a readable command line for report metadata.
func buildLintCommandString(parseExecutablePath string, parseArgs []string) string {
	parseParts := []string{parseExecutablePath}
	for _, parseArg := range parseArgs {
		if strings.ContainsAny(parseArg, " \t") {
			parseParts = append(parseParts, fmt.Sprintf("%q", parseArg))
			continue
		}
		parseParts = append(parseParts, parseArg)
	}
	return strings.Join(parseParts, " ")
}

// parseLintIssues parses golangci-lint JSON output into normalized issue records.
func parseLintIssues(parseOutput string, parseRootPath string) ([]lintIssueRecord, error) {
	if strings.TrimSpace(parseOutput) == "" {
		return nil, nil
	}
	var parseEnvelope map[string]interface{}
	if parseErr := json.Unmarshal([]byte(parseOutput), &parseEnvelope); parseErr != nil {
		return nil, parseErr
	}
	parseRawIssues, isParseFound := parseEnvelope["Issues"]
	if !isParseFound {
		parseRawIssues = parseEnvelope["issues"]
	}
	if parseRawIssues == nil {
		return nil, nil
	}
	parseIssueList, isParseList := parseRawIssues.([]interface{})
	if !isParseList {
		return nil, errors.New("missing issues array")
	}
	parseIssues := make([]lintIssueRecord, 0, len(parseIssueList))
	for _, parseRawIssue := range parseIssueList {
		parseIssueMap, isParseMap := parseRawIssue.(map[string]interface{})
		if !isParseMap {
			continue
		}
		parseIssues = append(parseIssues, parseLintIssueRecord(parseIssueMap, parseRootPath))
	}
	sortLintIssues(parseIssues)
	return parseIssues, nil
}

// sortLintIssues applies the stable lint issue ordering used by reports.
func sortLintIssues(parseIssues []lintIssueRecord) {
	sort.Slice(parseIssues, func(parseI int, parseJ int) bool {
		parseLeft := parseIssues[parseI]
		parseRight := parseIssues[parseJ]
		if parseLeft.Path != parseRight.Path {
			return parseLeft.Path < parseRight.Path
		}
		if parseLeft.Line != parseRight.Line {
			return parseLeft.Line < parseRight.Line
		}
		if parseLeft.Column != parseRight.Column {
			return parseLeft.Column < parseRight.Column
		}
		if parseLeft.Linter != parseRight.Linter {
			return parseLeft.Linter < parseRight.Linter
		}
		return parseLeft.Message < parseRight.Message
	})
}

// parseLintIssueRecord translates one raw golangci-lint issue into launcher metadata.
func parseLintIssueRecord(parseIssue map[string]interface{}, parseRootPath string) lintIssueRecord {
	parsePosition := parseLintMap(parseIssue["Pos"])
	parseFilename := parseLintIssuePath(parseRootPath, parseLintString(parsePosition["Filename"]))
	parseSeverity := parseLintSeverity(parseLintString(parseIssue["Severity"]))
	parseSourceLine := ""
	parseSourceLines := parseLintStrings(parseIssue["SourceLines"])
	if len(parseSourceLines) > 0 {
		parseSourceLine = strings.TrimSpace(parseSourceLines[0])
	}
	return lintIssueRecord{
		Linter:     firstNonEmpty(parseLintString(parseIssue["FromLinter"]), parseLintString(parseIssue["Linter"]), "unknown"),
		Severity:   parseSeverity,
		Path:       parseFilename,
		Line:       parseLintInt(parsePosition["Line"]),
		Column:     parseLintInt(parsePosition["Column"]),
		Message:    firstNonEmpty(parseLintString(parseIssue["Text"]), parseLintString(parseIssue["Message"]), "lint issue"),
		SourceLine: parseSourceLine,
	}
}

// parseLintIssuePath rewrites absolute lint file paths relative to the lint root when possible.
func parseLintIssuePath(parseRootPath string, parseFilename string) string {
	parseTrimmed := strings.TrimSpace(parseFilename)
	if parseTrimmed == "" {
		return ""
	}
	if filepath.IsAbs(parseTrimmed) && strings.TrimSpace(parseRootPath) != "" {
		parseRelativePath, parseErr := filepath.Rel(parseRootPath, parseTrimmed)
		if parseErr == nil && !strings.HasPrefix(parseRelativePath, "..") {
			return filepath.ToSlash(parseRelativePath)
		}
	}
	return filepath.ToSlash(parseTrimmed)
}

// parseLintSeverity normalizes empty severities into a stable report label.
func parseLintSeverity(parseSeverity string) string {
	parseTrimmed := strings.TrimSpace(strings.ToLower(parseSeverity))
	if parseTrimmed == "" {
		return "unspecified"
	}
	return parseTrimmed
}

// parseLintMap safely converts a JSON value into an object map.
func parseLintMap(parseValue interface{}) map[string]interface{} {
	parseMap, _ := parseValue.(map[string]interface{})
	return parseMap
}

// parseLintString safely converts a JSON scalar into a string.
func parseLintString(parseValue interface{}) string {
	if parseValue == nil {
		return ""
	}
	switch parseTyped := parseValue.(type) {
	case string:
		return strings.TrimSpace(parseTyped)
	case fmt.Stringer:
		return strings.TrimSpace(parseTyped.String())
	default:
		return strings.TrimSpace(fmt.Sprint(parseValue))
	}
}

// parseLintInt safely converts a JSON number into an integer.
func parseLintInt(parseValue interface{}) int {
	switch parseTyped := parseValue.(type) {
	case float64:
		return int(parseTyped)
	case int:
		return parseTyped
	case int32:
		return int(parseTyped)
	case int64:
		return int(parseTyped)
	case json.Number:
		parseInt, parseErr := parseTyped.Int64()
		if parseErr == nil {
			return int(parseInt)
		}
	}
	parseParsed, parseErr := strconv.Atoi(strings.TrimSpace(fmt.Sprint(parseValue)))
	if parseErr != nil {
		return 0
	}
	return parseParsed
}

// parseLintStrings safely converts a JSON array into a string slice.
func parseLintStrings(parseValue interface{}) []string {
	parseList, isParseList := parseValue.([]interface{})
	if !isParseList {
		return nil
	}
	parseStrings := make([]string, 0, len(parseList))
	for _, parseEntry := range parseList {
		parseTrimmed := strings.TrimSpace(parseLintString(parseEntry))
		if parseTrimmed == "" {
			continue
		}
		parseStrings = append(parseStrings, parseTrimmed)
	}
	return parseStrings
}

// buildLintCounts groups lint issues by linter and severity.
func buildLintCounts(parseIssues []lintIssueRecord) (map[string]int, map[string]int) {
	parseLinterCounts := map[string]int{}
	parseSeverityCounts := map[string]int{}
	for _, parseIssue := range parseIssues {
		parseLinterCounts[parseIssue.Linter]++
		parseSeverityCounts[parseIssue.Severity]++
	}
	return parseLinterCounts, parseSeverityCounts
}

// formatLintReport renders the human-readable lint report.
func formatLintReport(parseSummary lintSummary) string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("GWC lint\n")
	parseBuilder.WriteString(fmt.Sprintf("  tool:         %s\n", parseSummary.Tool))
	if parseSummary.ToolVersion != "" {
		parseBuilder.WriteString(fmt.Sprintf("  version:      %s\n", parseSummary.ToolVersion))
	}
	parseBuilder.WriteString(fmt.Sprintf("  executable:   %s\n", parseSummary.ToolPath))
	parseBuilder.WriteString(fmt.Sprintf("  project root: %s\n", parseSummary.ProjectRoot))
	parseBuilder.WriteString(fmt.Sprintf("  paths:        %s\n", strings.Join(parseSummary.Paths, ", ")))
	switch {
	case parseSummary.NoConfig:
		parseBuilder.WriteString("  config:       disabled (--no-config)\n")
	case parseSummary.ConfigPath != "":
		parseBuilder.WriteString(fmt.Sprintf("  config:       %s\n", parseSummary.ConfigPath))
	default:
		parseBuilder.WriteString("  config:       <auto>\n")
	}
	if parseSummary.HookRules {
		parseBuilder.WriteString("  hook rules:   enabled\n")
	} else {
		parseBuilder.WriteString("  hook rules:   disabled\n")
	}
	parseBuilder.WriteString(fmt.Sprintf("  duration:     %dms\n", parseSummary.DurationMs))
	parseBuilder.WriteString(fmt.Sprintf("  issues:       %d\n", parseSummary.IssueCount))
	parseBuilder.WriteString(fmt.Sprintf("  linters:      %s\n", formatLintCounts(parseSummary.LinterCounts)))
	parseBuilder.WriteString(fmt.Sprintf("  severities:   %s\n", formatLintCounts(parseSummary.SeverityCounts)))
	if parseSummary.ReportPath != "" {
		parseBuilder.WriteString(fmt.Sprintf("  report:       %s\n", parseSummary.ReportPath))
	}
	formatLintResolutionTrace(&parseBuilder, parseSummary.Resolution)
	parseBuilder.WriteString("\nIssues:\n")
	if len(parseSummary.Issues) == 0 {
		parseBuilder.WriteString("  none\n")
		return parseBuilder.String()
	}
	for parseIndex, parseIssue := range parseSummary.Issues {
		parseLocation := formatLintLocation(parseIssue)
		parseBuilder.WriteString(fmt.Sprintf("  %d. [%s] %s\n", parseIndex+1, parseIssue.Linter, parseLocation))
		parseBuilder.WriteString(fmt.Sprintf("     severity: %s\n", parseIssue.Severity))
		parseBuilder.WriteString(fmt.Sprintf("     message:  %s\n", parseIssue.Message))
		if parseIssue.SourceLine != "" {
			parseBuilder.WriteString(fmt.Sprintf("     source:   %s\n", parseIssue.SourceLine))
		}
	}
	return parseBuilder.String()
}

// formatLintResolutionTrace appends lint resolution metadata to the rendered report.
func formatLintResolutionTrace(parseBuilder *strings.Builder, parseTrace map[string]string) {
	if parseBuilder == nil || len(parseTrace) == 0 {
		return
	}
	for _, parseKey := range []string{"root", "config", "report"} {
		parseSource, isParseFound := parseTrace[parseKey]
		if !isParseFound || strings.TrimSpace(parseSource) == "" {
			continue
		}
		parseBuilder.WriteString(fmt.Sprintf("  %s source: %s\n", parseKey, parseSource))
	}
}

// formatLintLocation renders a stable file location label for one lint issue.
func formatLintLocation(parseIssue lintIssueRecord) string {
	parseLocation := firstNonEmpty(parseIssue.Path, "<unknown>")
	if parseIssue.Line > 0 {
		parseLocation = fmt.Sprintf("%s:%d", parseLocation, parseIssue.Line)
	}
	if parseIssue.Column > 0 {
		parseLocation = fmt.Sprintf("%s:%d", parseLocation, parseIssue.Column)
	}
	return parseLocation
}

// formatLintCounts renders grouped lint counts in deterministic order.
func formatLintCounts(parseCounts map[string]int) string {
	if len(parseCounts) == 0 {
		return "<none>"
	}
	parseKeys := make([]string, 0, len(parseCounts))
	for parseKey := range parseCounts {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	parseParts := make([]string, 0, len(parseKeys))
	for _, parseKey := range parseKeys {
		parseParts = append(parseParts, fmt.Sprintf("%s=%d", parseKey, parseCounts[parseKey]))
	}
	return strings.Join(parseParts, ", ")
}

// storeLintReport writes a rendered lint report to disk.
func storeLintReport(parsePath string, parseContent []byte) error {
	if strings.TrimSpace(parsePath) == "" {
		return nil
	}
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0755); parseErr != nil {
		return fmt.Errorf("create lint report directory: %w", parseErr)
	}
	if parseErr := os.WriteFile(parsePath, parseContent, 0644); parseErr != nil {
		return fmt.Errorf("write lint report: %w", parseErr)
	}
	return nil
}
