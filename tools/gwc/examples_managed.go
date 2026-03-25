package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const examplesManagedDefaultProfile = "chat-wizard-local"

type examplesManagedProfile struct {
	profileName       string
	serverPath        string
	buildTargetPath   string
	commandPath       string
	commandArgs       []string
	commandDir        string
	defaultHost       string
	defaultPort       string
	defaultHealthPath string
}

type examplesManagedLaunchConfig struct {
	commandPath string
	commandArgs []string
	commandDir  string
	listenAddr  string
	logPath     string
}

type examplesManagedServerState struct {
	ProfileName string   `json:"profileName"`
	ServerPath  string   `json:"serverPath,omitempty"`
	PID         int      `json:"pid"`
	Host        string   `json:"host"`
	Port        string   `json:"port"`
	ListenAddr  string   `json:"listenAddr"`
	URL         string   `json:"url"`
	HealthURL   string   `json:"healthURL"`
	HealthPath  string   `json:"healthPath"`
	CommandPath string   `json:"commandPath"`
	CommandArgs []string `json:"commandArgs"`
	CommandDir  string   `json:"commandDir"`
	LogPath     string   `json:"logPath"`
	StartedAt   string   `json:"startedAt"`
}

type examplesManagedSummary struct {
	OK          bool   `json:"ok"`
	Action      string `json:"action"`
	ProfileName string `json:"profileName"`
	ServerPath  string `json:"serverPath,omitempty"`
	IsRunning   bool   `json:"isRunning"`
	IsHealthy   bool   `json:"isHealthy,omitempty"`
	PID         int    `json:"pid,omitempty"`
	URL         string `json:"url,omitempty"`
	HealthURL   string `json:"healthURL,omitempty"`
	StatePath   string `json:"statePath,omitempty"`
	LogPath     string `json:"logPath,omitempty"`
	Message     string `json:"message,omitempty"`
}

var examplesManagedResolveProfile = resolveExamplesManagedProfile

var examplesManagedLaunchProcess = launchExamplesManagedProcess

var examplesManagedWaitServerReady = waitExamplesManagedServerReady

var examplesManagedBuildBinary = buildExamplesManagedServerBinary

var examplesManagedCheckPIDRunning = checkLauncherPIDRunning

var examplesManagedTerminatePIDTree = terminateLauncherPIDTree

var examplesManagedNowUTC = func() time.Time {
	return time.Now().UTC()
}

// isExamplesManagedAction reports whether `gwc examples` was invoked with a managed lifecycle action.
func isExamplesManagedAction(parseArgs []string) bool {
	if len(parseArgs) == 0 {
		return false
	}
	switch strings.TrimSpace(strings.ToLower(parseArgs[0])) {
	case "start", "status", "stop":
		return true
	default:
		return false
	}
}

// isExamplesManagedPathCommand reports whether a top-level path command should dispatch to managed examples actions.
func isExamplesManagedPathCommand(parseCommand string, parseArgs []string) bool {
	parseCommand = strings.TrimSpace(parseCommand)
	if parseCommand == "" {
		return false
	}
	if !isExamplesManagedAction(parseArgs) {
		return false
	}
	if strings.Contains(parseCommand, `\`) || strings.Contains(parseCommand, "/") || strings.HasPrefix(parseCommand, ".") {
		return true
	}
	if len(parseCommand) >= 2 && parseCommand[1] == ':' {
		return true
	}
	return false
}

// buildExamplesManagedPathCommandArgs rewrites path-first argv into managed examples argv.
func buildExamplesManagedPathCommandArgs(parseCommand string, parseArgs []string) []string {
	if len(parseArgs) == 0 {
		return nil
	}
	parseManagedArgs := []string{parseArgs[0], "-path", parseCommand}
	if len(parseArgs) > 1 {
		parseManagedArgs = append(parseManagedArgs, parseArgs[1:]...)
	}
	return parseManagedArgs
}

// runExamplesManaged dispatches managed example-server lifecycle actions.
func (parseL launcher) runExamplesManaged(parseArgs []string) error {
	if len(parseArgs) == 0 {
		return errors.New("examples managed action is required: start, status, or stop")
	}
	parseAction := strings.TrimSpace(strings.ToLower(parseArgs[0]))
	parseActionArgs := parseArgs[1:]
	switch parseAction {
	case "start":
		return parseL.runExamplesManagedStart(parseActionArgs)
	case "status":
		return parseL.runExamplesManagedStatus(parseActionArgs)
	case "stop":
		return parseL.runExamplesManagedStop(parseActionArgs)
	default:
		return fmt.Errorf("unknown examples managed action %q", parseArgs[0])
	}
}

// runExamplesManagedStart starts one managed example-server profile and persists runtime state.
func (parseL launcher) runExamplesManagedStart(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("examples start", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseProfileName := parseFlags.String("profile", examplesManagedDefaultProfile, "Managed profile name to start")
	parseServerPath := parseFlags.String("path", "", "Path to a Go server package directory or main.go file")
	parseHost := parseFlags.String("host", "", "Optional host override for LISTEN_ADDR and health probing")
	parsePort := parseFlags.String("port", "", "Optional port override for LISTEN_ADDR and health probing")
	parseHealthPath := parseFlags.String("health-path", "", "Optional health endpoint path override")
	parseHealthTimeout := parseFlags.Duration("health-timeout", 45*time.Second, "Maximum time to wait for a healthy server")
	parseJSON := parseFlags.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseResolvedPath, parseErr := resolveExamplesManagedServerPath(*parseServerPath, parseFlags.Args())
	if parseErr != nil {
		return parseErr
	}
	parseSummary, parseErr := parseL.applyExamplesManagedStart(*parseProfileName, parseResolvedPath, *parseHost, *parsePort, *parseHealthPath, *parseHealthTimeout)
	if parseErr != nil {
		return parseErr
	}
	return renderExamplesManagedSummary(parseSummary, *parseJSON)
}

// runExamplesManagedStatus reports process and health status for one managed example-server profile.
func (parseL launcher) runExamplesManagedStatus(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("examples status", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseProfileName := parseFlags.String("profile", examplesManagedDefaultProfile, "Managed profile name to inspect")
	parseServerPath := parseFlags.String("path", "", "Path to the managed Go server package directory or main.go file")
	parseJSON := parseFlags.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseResolvedPath, parseErr := resolveExamplesManagedServerPath(*parseServerPath, parseFlags.Args())
	if parseErr != nil {
		return parseErr
	}
	parseSummary, parseErr := parseL.applyExamplesManagedStatus(*parseProfileName, parseResolvedPath)
	if parseErr != nil {
		return parseErr
	}
	return renderExamplesManagedSummary(parseSummary, *parseJSON)
}

// runExamplesManagedStop stops one managed example-server profile and clears persisted state.
func (parseL launcher) runExamplesManagedStop(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("examples stop", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseProfileName := parseFlags.String("profile", examplesManagedDefaultProfile, "Managed profile name to stop")
	parseServerPath := parseFlags.String("path", "", "Path to the managed Go server package directory or main.go file")
	parseJSON := parseFlags.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseResolvedPath, parseErr := resolveExamplesManagedServerPath(*parseServerPath, parseFlags.Args())
	if parseErr != nil {
		return parseErr
	}
	parseSummary, parseErr := parseL.applyExamplesManagedStop(*parseProfileName, parseResolvedPath)
	if parseErr != nil {
		return parseErr
	}
	return renderExamplesManagedSummary(parseSummary, *parseJSON)
}

// resolveExamplesManagedServerPath resolves a `-path` flag or one positional server path argument.
func resolveExamplesManagedServerPath(parseFlagPath string, parseArgs []string) (string, error) {
	parseFlagPath = strings.TrimSpace(parseFlagPath)
	if len(parseArgs) > 1 {
		return "", fmt.Errorf("expected at most one server path argument, got %d", len(parseArgs))
	}
	if parseFlagPath != "" && len(parseArgs) == 1 {
		return "", errors.New("use either -path or one positional server path argument, not both")
	}
	if parseFlagPath == "" && len(parseArgs) == 1 {
		parseFlagPath = strings.TrimSpace(parseArgs[0])
	}
	return parseFlagPath, nil
}

// applyExamplesManagedStart resolves config, launches a profile process, persists state, and waits for health.
func (parseL launcher) applyExamplesManagedStart(parseProfileName string, parseServerPath string, parseHost string, parsePort string, parseHealthPath string, parseHealthTimeout time.Duration) (examplesManagedSummary, error) {
	parseProfile, parseErr := parseL.resolveExamplesManagedCommand(parseProfileName, parseServerPath)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	return parseL.applyExamplesManagedStartForProfile(parseProfile, parseHost, parsePort, parseHealthPath, parseHealthTimeout)
}

// applyExamplesManagedStartForProfile launches and persists state for a fully resolved managed profile.
func (parseL launcher) applyExamplesManagedStartForProfile(parseProfile examplesManagedProfile, parseHost string, parsePort string, parseHealthPath string, parseHealthTimeout time.Duration) (examplesManagedSummary, error) {
	parseHost = firstNonEmpty(strings.TrimSpace(parseHost), parseProfile.defaultHost)
	parsePort = firstNonEmpty(strings.TrimSpace(parsePort), parseProfile.defaultPort)
	parseHealthPath = normalizeExamplesManagedHealthPath(firstNonEmpty(strings.TrimSpace(parseHealthPath), parseProfile.defaultHealthPath))

	parseStatePath, parseLogPath, parseErr := parseL.resolveExamplesManagedStatePaths(parseProfile.profileName)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	parseState, parseFoundState, parseErr := readExamplesManagedState(parseStatePath)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	if parseFoundState && examplesManagedCheckPIDRunning(parseState.PID) {
		parseHealthy, _ := probeExamplesManagedHealth(parseState.HealthURL, 1200*time.Millisecond)
		return examplesManagedSummary{
			OK:          true,
			Action:      "start",
			ProfileName: parseState.ProfileName,
			ServerPath:  parseState.ServerPath,
			IsRunning:   true,
			IsHealthy:   parseHealthy,
			PID:         parseState.PID,
			URL:         parseState.URL,
			HealthURL:   parseState.HealthURL,
			StatePath:   parseStatePath,
			LogPath:     parseState.LogPath,
			Message:     "profile is already running",
		}, nil
	}
	if parseFoundState {
		_ = removeExamplesManagedState(parseStatePath)
	}
	parseListenAddr := joinHostPort(parseHost, parsePort)
	parseURL := "http://" + parseListenAddr
	parseHealthURL := parseURL + parseHealthPath
	parseLaunchConfig, parseErr := parseL.resolveExamplesManagedLaunchConfig(parseProfile, parseStatePath, parseLogPath, parseListenAddr)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	parsePID, parseErr := examplesManagedLaunchProcess(parseLaunchConfig)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}

	parseState = examplesManagedServerState{
		ProfileName: parseProfile.profileName,
		ServerPath:  parseProfile.serverPath,
		PID:         parsePID,
		Host:        parseHost,
		Port:        parsePort,
		ListenAddr:  parseListenAddr,
		URL:         parseURL,
		HealthURL:   parseHealthURL,
		HealthPath:  parseHealthPath,
		CommandPath: parseLaunchConfig.commandPath,
		CommandArgs: append([]string(nil), parseLaunchConfig.commandArgs...),
		CommandDir:  parseLaunchConfig.commandDir,
		LogPath:     parseLogPath,
		StartedAt:   examplesManagedNowUTC().Format(time.RFC3339),
	}
	if parseErr2 := writeExamplesManagedState(parseStatePath, parseState); parseErr2 != nil {
		_ = examplesManagedTerminatePIDTree(parsePID)
		return examplesManagedSummary{}, parseErr2
	}
	if parseErr3 := examplesManagedWaitServerReady(parseState, parseHealthTimeout); parseErr3 != nil {
		_ = examplesManagedTerminatePIDTree(parsePID)
		_ = removeExamplesManagedState(parseStatePath)
		parseLogTail := strings.TrimSpace(readExamplesManagedLogTail(parseLogPath, 4096))
		if parseLogTail != "" {
			return examplesManagedSummary{}, fmt.Errorf("examples start %q: %w\n\n%s", parseProfile.profileName, parseErr3, parseLogTail)
		}
		return examplesManagedSummary{}, fmt.Errorf("examples start %q: %w", parseProfile.profileName, parseErr3)
	}
	return examplesManagedSummary{
		OK:          true,
		Action:      "start",
		ProfileName: parseProfile.profileName,
		ServerPath:  parseProfile.serverPath,
		IsRunning:   true,
		IsHealthy:   true,
		PID:         parsePID,
		URL:         parseURL,
		HealthURL:   parseHealthURL,
		StatePath:   parseStatePath,
		LogPath:     parseLogPath,
		Message:     "profile started and passed health probe",
	}, nil
}

// applyExamplesManagedStatus reads managed runtime state and reports process plus health signals.
func (parseL launcher) applyExamplesManagedStatus(parseProfileName string, parseServerPath string) (examplesManagedSummary, error) {
	parseProfile, parseErr := parseL.resolveExamplesManagedCommand(parseProfileName, parseServerPath)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	parseStatePath, parseLogPath, parseErr := parseL.resolveExamplesManagedStatePaths(parseProfile.profileName)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	parseState, parseFoundState, parseErr := readExamplesManagedState(parseStatePath)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	if !parseFoundState {
		return examplesManagedSummary{
			OK:          true,
			Action:      "status",
			ProfileName: parseProfile.profileName,
			ServerPath:  parseProfile.serverPath,
			IsRunning:   false,
			StatePath:   parseStatePath,
			LogPath:     parseLogPath,
			Message:     "profile is not running",
		}, nil
	}
	parseSummary := examplesManagedSummary{
		OK:          true,
		Action:      "status",
		ProfileName: parseState.ProfileName,
		ServerPath:  firstNonEmpty(parseState.ServerPath, parseProfile.serverPath),
		IsRunning:   false,
		PID:         parseState.PID,
		URL:         parseState.URL,
		HealthURL:   parseState.HealthURL,
		StatePath:   parseStatePath,
		LogPath:     firstNonEmpty(parseState.LogPath, parseLogPath),
	}
	if !examplesManagedCheckPIDRunning(parseState.PID) {
		_ = removeExamplesManagedState(parseStatePath)
		parseSummary.Message = "profile is not running (stale state removed)"
		return parseSummary, nil
	}
	parseHealthy, parseProbeErr := probeExamplesManagedHealth(parseState.HealthURL, 1500*time.Millisecond)
	parseSummary.IsRunning = true
	parseSummary.IsHealthy = parseHealthy
	if parseProbeErr != nil {
		parseSummary.Message = fmt.Sprintf("profile process is running but health probe failed: %v", parseProbeErr)
		return parseSummary, nil
	}
	if parseHealthy {
		parseSummary.Message = "profile process is running and healthy"
		return parseSummary, nil
	}
	parseSummary.Message = "profile process is running but health endpoint did not return 2xx"
	return parseSummary, nil
}

// applyExamplesManagedStop terminates the managed profile process tree and removes persisted state.
func (parseL launcher) applyExamplesManagedStop(parseProfileName string, parseServerPath string) (examplesManagedSummary, error) {
	parseProfile, parseErr := parseL.resolveExamplesManagedCommand(parseProfileName, parseServerPath)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	parseStatePath, parseLogPath, parseErr := parseL.resolveExamplesManagedStatePaths(parseProfile.profileName)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	parseState, parseFoundState, parseErr := readExamplesManagedState(parseStatePath)
	if parseErr != nil {
		return examplesManagedSummary{}, parseErr
	}
	if !parseFoundState {
		return examplesManagedSummary{
			OK:          true,
			Action:      "stop",
			ProfileName: parseProfile.profileName,
			ServerPath:  parseProfile.serverPath,
			IsRunning:   false,
			StatePath:   parseStatePath,
			LogPath:     parseLogPath,
			Message:     "profile is not running",
		}, nil
	}
	parseSummary := examplesManagedSummary{
		OK:          true,
		Action:      "stop",
		ProfileName: parseState.ProfileName,
		ServerPath:  firstNonEmpty(parseState.ServerPath, parseProfile.serverPath),
		IsRunning:   false,
		PID:         parseState.PID,
		URL:         parseState.URL,
		HealthURL:   parseState.HealthURL,
		StatePath:   parseStatePath,
		LogPath:     firstNonEmpty(parseState.LogPath, parseLogPath),
	}
	if examplesManagedCheckPIDRunning(parseState.PID) {
		if parseErr2 := examplesManagedTerminatePIDTree(parseState.PID); parseErr2 != nil {
			return examplesManagedSummary{}, parseErr2
		}
		parseSummary.Message = "profile process tree terminated"
	} else {
		parseSummary.Message = "profile process was not running (stale state removed)"
	}
	if parseErr3 := removeExamplesManagedState(parseStatePath); parseErr3 != nil {
		return examplesManagedSummary{}, parseErr3
	}
	return parseSummary, nil
}

// resolveExamplesManagedCommand resolves either a path-target command or a named profile command.
func (parseL launcher) resolveExamplesManagedCommand(parseProfileName string, parseServerPath string) (examplesManagedProfile, error) {
	if strings.TrimSpace(parseServerPath) != "" {
		return parseL.resolveExamplesManagedPathCommand(parseServerPath)
	}
	return examplesManagedResolveProfile(parseL, parseProfileName)
}

// resolveExamplesManagedPathCommand resolves one managed path into a deterministic runtime state key and command plan.
func (parseL launcher) resolveExamplesManagedPathCommand(parseServerPath string) (examplesManagedProfile, error) {
	parseServerPath = strings.TrimSpace(parseServerPath)
	if parseServerPath == "" {
		return examplesManagedProfile{}, errors.New("managed examples server path is required")
	}
	parseBasePath := firstNonEmpty(strings.TrimSpace(parseL.repoRoot), ".")
	parseResolvedPath, parseErr := normalizePath(parseBasePath, parseServerPath)
	if parseErr != nil {
		return examplesManagedProfile{}, fmt.Errorf("resolve managed server path: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseResolvedPath)
	if parseErr != nil {
		return examplesManagedProfile{}, fmt.Errorf("stat managed server path: %w", parseErr)
	}
	if !parseInfo.IsDir() && strings.ToLower(filepath.Ext(parseResolvedPath)) != ".go" {
		return examplesManagedProfile{}, fmt.Errorf("managed server path %q must be a Go package directory or .go entrypoint", parseResolvedPath)
	}
	parseCommandDir := strings.TrimSpace(parseL.repoRoot)
	if parseCommandDir == "" || !isExamplesManagedPathUnderRoot(parseResolvedPath, parseCommandDir) {
		if parseInfo.IsDir() {
			parseCommandDir = parseResolvedPath
		} else {
			parseCommandDir = filepath.Dir(parseResolvedPath)
		}
	}
	parseStateKey := buildExamplesManagedPathStateKey(parseResolvedPath)
	return examplesManagedProfile{
		profileName:       parseStateKey,
		serverPath:        parseResolvedPath,
		buildTargetPath:   parseResolvedPath,
		commandPath:       "go",
		commandArgs:       []string{"run", parseResolvedPath},
		commandDir:        parseCommandDir,
		defaultHost:       defaultHost,
		defaultPort:       "8095",
		defaultHealthPath: "/healthz",
	}, nil
}

// isExamplesManagedPathUnderRoot reports whether a resolved path is within a candidate root directory.
func isExamplesManagedPathUnderRoot(parseResolvedPath string, parseRootPath string) bool {
	parseResolvedPath = strings.TrimSpace(parseResolvedPath)
	parseRootPath = strings.TrimSpace(parseRootPath)
	if parseResolvedPath == "" || parseRootPath == "" {
		return false
	}
	parseRelativePath, parseErr := filepath.Rel(parseRootPath, parseResolvedPath)
	if parseErr != nil {
		return false
	}
	parseRelativePath = strings.TrimSpace(parseRelativePath)
	if parseRelativePath == "." {
		return true
	}
	if parseRelativePath == ".." {
		return false
	}
	parseUpPrefix := ".." + string(filepath.Separator)
	return !strings.HasPrefix(parseRelativePath, parseUpPrefix)
}

// resolveExamplesManagedProfile resolves one supported managed example-server profile.
func resolveExamplesManagedProfile(parseL launcher, parseProfileName string) (examplesManagedProfile, error) {
	parseProfileName = strings.ToLower(strings.TrimSpace(parseProfileName))
	switch parseProfileName {
	case "", "chat-wizard", "chat-wizard-local", "relaydesk-local":
		parseCommandDir := filepath.Join(parseL.repoRoot, "examples", "100-ai-chat-wizard", "cmd", "server")
		parseInfo, parseErr := os.Stat(parseCommandDir)
		if parseErr != nil || !parseInfo.IsDir() {
			return examplesManagedProfile{}, fmt.Errorf("managed profile %q command directory not found: %s", examplesManagedDefaultProfile, parseCommandDir)
		}
		return examplesManagedProfile{
			profileName:       examplesManagedDefaultProfile,
			serverPath:        parseCommandDir,
			buildTargetPath:   parseCommandDir,
			commandPath:       "go",
			commandArgs:       []string{"run", "./examples/100-ai-chat-wizard/cmd/server"},
			commandDir:        parseL.repoRoot,
			defaultHost:       defaultHost,
			defaultPort:       "8095",
			defaultHealthPath: "/healthz",
		}, nil
	default:
		return examplesManagedProfile{}, fmt.Errorf("unknown managed examples profile %q", parseProfileName)
	}
}

// buildExamplesManagedPathStateKey builds a deterministic runtime-state key from a managed server path.
func buildExamplesManagedPathStateKey(parseServerPath string) string {
	parseNormalizedPath := filepath.ToSlash(filepath.Clean(strings.TrimSpace(parseServerPath)))
	parseDigest := sha1.Sum([]byte(parseNormalizedPath))
	parseEncoded := hex.EncodeToString(parseDigest[:])
	return "path-" + parseEncoded[:16]
}

// resolveExamplesManagedLaunchConfig resolves command launch configuration and builds an executable when requested.
func (parseL launcher) resolveExamplesManagedLaunchConfig(parseProfile examplesManagedProfile, parseStatePath string, parseLogPath string, parseListenAddr string) (examplesManagedLaunchConfig, error) {
	_ = parseL
	parseCommandPath := parseProfile.commandPath
	parseCommandArgs := append([]string(nil), parseProfile.commandArgs...)
	if strings.TrimSpace(parseProfile.buildTargetPath) != "" {
		parseBinaryPath := buildExamplesManagedBinaryPath(parseStatePath)
		if parseErr := examplesManagedBuildBinary(parseProfile.buildTargetPath, parseBinaryPath, parseProfile.commandDir); parseErr != nil {
			return examplesManagedLaunchConfig{}, parseErr
		}
		parseCommandPath = parseBinaryPath
		parseCommandArgs = nil
	}
	return examplesManagedLaunchConfig{
		commandPath: parseCommandPath,
		commandArgs: parseCommandArgs,
		commandDir:  parseProfile.commandDir,
		listenAddr:  parseListenAddr,
		logPath:     parseLogPath,
	}, nil
}

// buildExamplesManagedBinaryPath resolves one managed-server binary artifact path from a state-file path.
func buildExamplesManagedBinaryPath(parseStatePath string) string {
	parseBinaryPath := strings.TrimSuffix(parseStatePath, filepath.Ext(parseStatePath))
	if runtime.GOOS == "windows" {
		return parseBinaryPath + ".exe"
	}
	return parseBinaryPath
}

// resolveExamplesManagedRuntimeDir resolves the runtime directory used for managed example-server artifacts.
func (parseL launcher) resolveExamplesManagedRuntimeDir() (string, error) {
	parseRootPath := strings.TrimSpace(parseL.repoRoot)
	if parseRootPath == "" {
		return "", errors.New("resolve managed runtime directory: launcher repo root is empty")
	}
	if parseArtifactRoot, parseHasArtifactRoot, parseErr := resolveLauncherArtifactRoot(parseRootPath); parseErr != nil {
		return "", fmt.Errorf("resolve managed runtime artifact root: %w", parseErr)
	} else if parseHasArtifactRoot {
		return filepath.Join(parseArtifactRoot, "runtime", "examples-servers"), nil
	}
	return filepath.Join(parseRootPath, "bin", "runtime", "examples-servers"), nil
}

// resolveExamplesManagedStatePaths resolves state and log artifact paths for one managed profile.
func (parseL launcher) resolveExamplesManagedStatePaths(parseProfileName string) (string, string, error) {
	parseProfileKey, parseErr := normalizeExamplesManagedProfileName(parseProfileName)
	if parseErr != nil {
		return "", "", parseErr
	}
	parseRuntimeDir, parseErr := parseL.resolveExamplesManagedRuntimeDir()
	if parseErr != nil {
		return "", "", parseErr
	}
	return filepath.Join(parseRuntimeDir, parseProfileKey+".json"), filepath.Join(parseRuntimeDir, parseProfileKey+".log"), nil
}

// normalizeExamplesManagedProfileName validates and normalizes profile names for runtime artifact file names.
func normalizeExamplesManagedProfileName(parseProfileName string) (string, error) {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseProfileName))
	if parseNormalized == "" {
		return "", errors.New("managed examples profile name is required")
	}
	parseValidProfileName := regexp.MustCompile(`^[a-z0-9-]+$`)
	if !parseValidProfileName.MatchString(parseNormalized) {
		return "", fmt.Errorf("managed examples profile name %q is invalid; use lowercase letters, numbers, and hyphens", parseProfileName)
	}
	return parseNormalized, nil
}

// readExamplesManagedState reads one managed profile runtime state file when present.
func readExamplesManagedState(parseStatePath string) (examplesManagedServerState, bool, error) {
	parsePayload, parseErr := os.ReadFile(parseStatePath)
	if parseErr != nil {
		if os.IsNotExist(parseErr) {
			return examplesManagedServerState{}, false, nil
		}
		return examplesManagedServerState{}, false, fmt.Errorf("read managed profile state: %w", parseErr)
	}
	var parseState examplesManagedServerState
	if parseErr2 := json.Unmarshal(parsePayload, &parseState); parseErr2 != nil {
		return examplesManagedServerState{}, true, fmt.Errorf("decode managed profile state %s: %w", parseStatePath, parseErr2)
	}
	return parseState, true, nil
}

// writeExamplesManagedState writes one managed profile runtime state file atomically.
func writeExamplesManagedState(parseStatePath string, parseState examplesManagedServerState) error {
	parseStateDir := filepath.Dir(parseStatePath)
	if parseErr := os.MkdirAll(parseStateDir, 0755); parseErr != nil {
		return fmt.Errorf("create managed profile runtime directory: %w", parseErr)
	}
	parsePayload, parseErr := json.MarshalIndent(parseState, "", "  ")
	if parseErr != nil {
		return fmt.Errorf("encode managed profile state: %w", parseErr)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr2 := os.WriteFile(parseStatePath, parsePayload, 0644); parseErr2 != nil {
		return fmt.Errorf("write managed profile state: %w", parseErr2)
	}
	return nil
}

// removeExamplesManagedState removes one managed profile runtime state file when present.
func removeExamplesManagedState(parseStatePath string) error {
	parseErr := os.Remove(parseStatePath)
	if parseErr == nil || os.IsNotExist(parseErr) {
		return nil
	}
	return fmt.Errorf("remove managed profile state: %w", parseErr)
}

// renderExamplesManagedSummary renders managed examples lifecycle output as JSON or plain text.
func renderExamplesManagedSummary(parseSummary examplesManagedSummary, isJSON bool) error {
	if isJSON {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		return parseEncoder.Encode(parseSummary)
	}
	printExamplesManagedSummary(parseSummary)
	return nil
}

// printExamplesManagedSummary prints a plain-text managed examples lifecycle summary.
func printExamplesManagedSummary(parseSummary examplesManagedSummary) {
	fmt.Println("GWC examples managed")
	fmt.Printf("  action:   %s\n", parseSummary.Action)
	fmt.Printf("  profile:  %s\n", parseSummary.ProfileName)
	if strings.TrimSpace(parseSummary.ServerPath) != "" {
		fmt.Printf("  path:     %s\n", parseSummary.ServerPath)
	}
	fmt.Printf("  running:  %t\n", parseSummary.IsRunning)
	if parseSummary.PID > 0 {
		fmt.Printf("  pid:      %d\n", parseSummary.PID)
	}
	if strings.TrimSpace(parseSummary.URL) != "" {
		fmt.Printf("  url:      %s\n", parseSummary.URL)
	}
	if strings.TrimSpace(parseSummary.HealthURL) != "" {
		fmt.Printf("  health:   %s\n", parseSummary.HealthURL)
		fmt.Printf("  healthy:  %t\n", parseSummary.IsHealthy)
	}
	if strings.TrimSpace(parseSummary.StatePath) != "" {
		fmt.Printf("  state:    %s\n", parseSummary.StatePath)
	}
	if strings.TrimSpace(parseSummary.LogPath) != "" {
		fmt.Printf("  logs:     %s\n", parseSummary.LogPath)
	}
	if strings.TrimSpace(parseSummary.Message) != "" {
		fmt.Printf("  message:  %s\n", parseSummary.Message)
	}
}

// normalizeExamplesManagedHealthPath normalizes health-path input into a rooted URL path.
func normalizeExamplesManagedHealthPath(parseHealthPath string) string {
	parseHealthPath = strings.TrimSpace(parseHealthPath)
	if parseHealthPath == "" {
		return "/healthz"
	}
	if !strings.HasPrefix(parseHealthPath, "/") {
		return "/" + parseHealthPath
	}
	return parseHealthPath
}

// launchExamplesManagedProcess starts a detached managed profile command and returns its PID.
func launchExamplesManagedProcess(parseConfig examplesManagedLaunchConfig) (int, error) {
	parseLogDir := filepath.Dir(parseConfig.logPath)
	if parseErr := os.MkdirAll(parseLogDir, 0755); parseErr != nil {
		return 0, fmt.Errorf("create managed profile log directory: %w", parseErr)
	}
	parseLogFile, parseErr := os.OpenFile(parseConfig.logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if parseErr != nil {
		return 0, fmt.Errorf("open managed profile log file: %w", parseErr)
	}
	parseCmd := exec.Command(parseConfig.commandPath, parseConfig.commandArgs...)
	parseCmd.Dir = parseConfig.commandDir
	parseCmd.Env = buildExamplesManagedProcessEnv(parseConfig.listenAddr)
	parseCmd.Stdout = parseLogFile
	parseCmd.Stderr = parseLogFile
	if parseErr2 := parseCmd.Start(); parseErr2 != nil {
		_ = parseLogFile.Close()
		return 0, fmt.Errorf("start managed profile command: %w", parseErr2)
	}
	_ = parseLogFile.Close()
	if parseCmd.Process == nil {
		return 0, errors.New("start managed profile command: missing process handle")
	}
	return parseCmd.Process.Pid, nil
}

// buildExamplesManagedServerBinary builds a managed server target into a launcher-owned runtime binary path.
func buildExamplesManagedServerBinary(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
	parseTargetPath = strings.TrimSpace(parseTargetPath)
	parseBinaryPath = strings.TrimSpace(parseBinaryPath)
	parseWorkingDir = strings.TrimSpace(parseWorkingDir)
	if parseTargetPath == "" || parseBinaryPath == "" {
		return errors.New("managed server build target and output path are required")
	}
	if parseWorkingDir == "" {
		parseWorkingDir = "."
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseBinaryPath), 0755); parseErr != nil {
		return fmt.Errorf("create managed server binary directory: %w", parseErr)
	}
	if _, parseErr := launcherRunCommand("go", []string{"build", "-o", parseBinaryPath, parseTargetPath}, parseWorkingDir, buildNativeGoEnv()); parseErr != nil {
		return fmt.Errorf("build managed server binary: %w", parseErr)
	}
	return nil
}

// buildExamplesManagedProcessEnv builds the process environment with a launcher-owned LISTEN_ADDR override.
func buildExamplesManagedProcessEnv(parseListenAddr string) []string {
	parseEnv := buildNativeGoEnv()
	parseListenEnv := "LISTEN_ADDR=" + strings.TrimSpace(parseListenAddr)
	parseHasListenEnv := false
	for parseIndex := range parseEnv {
		if strings.HasPrefix(parseEnv[parseIndex], "LISTEN_ADDR=") {
			parseEnv[parseIndex] = parseListenEnv
			parseHasListenEnv = true
			break
		}
	}
	if !parseHasListenEnv {
		parseEnv = append(parseEnv, parseListenEnv)
	}
	return parseEnv
}

// waitExamplesManagedServerReady waits for the managed profile process and health endpoint to become ready.
func waitExamplesManagedServerReady(parseState examplesManagedServerState, parseTimeout time.Duration) error {
	if parseTimeout <= 0 {
		return nil
	}
	parseDeadline := time.Now().Add(parseTimeout)
	parseLastProbeErr := error(nil)
	for {
		if !examplesManagedCheckPIDRunning(parseState.PID) {
			return errors.New("managed profile process exited before reporting healthy")
		}
		parseHealthy, parseProbeErr := probeExamplesManagedHealth(parseState.HealthURL, 1200*time.Millisecond)
		parseLastProbeErr = parseProbeErr
		if parseHealthy {
			return nil
		}
		if time.Now().After(parseDeadline) {
			if parseLastProbeErr != nil {
				return fmt.Errorf("timed out waiting for managed profile health: %w", parseLastProbeErr)
			}
			return errors.New("timed out waiting for managed profile health")
		}
		time.Sleep(250 * time.Millisecond)
	}
}

// probeExamplesManagedHealth issues one HTTP probe against the managed profile health URL.
func probeExamplesManagedHealth(parseHealthURL string, parseTimeout time.Duration) (bool, error) {
	parseClient := &http.Client{Timeout: parseTimeout}
	parseResp, parseErr := parseClient.Get(parseHealthURL)
	if parseErr != nil {
		return false, parseErr
	}
	defer parseResp.Body.Close()
	return parseResp.StatusCode >= 200 && parseResp.StatusCode < 300, nil
}

// readExamplesManagedLogTail reads the trailing bytes from a managed profile log file.
func readExamplesManagedLogTail(parseLogPath string, parseMaxBytes int) string {
	if parseMaxBytes <= 0 {
		return ""
	}
	parsePayload, parseErr := os.ReadFile(parseLogPath)
	if parseErr != nil {
		return ""
	}
	if len(parsePayload) > parseMaxBytes {
		parsePayload = parsePayload[len(parsePayload)-parseMaxBytes:]
	}
	return string(parsePayload)
}
