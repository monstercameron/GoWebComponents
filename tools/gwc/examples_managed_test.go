package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stageManagedProfileCommandDir creates the managed chat-wizard command directory expected by profile resolution.
func stageManagedProfileCommandDir(parseT *testing.T, parseRootPath string) {
	parseT.Helper()
	parseCommandDir := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "cmd", "server")
	if parseErr := os.MkdirAll(parseCommandDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir managed command dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseCommandDir, "main.go"), []byte("package main\nfunc main(){}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write managed command placeholder: %v", parseErr2)
	}
}

// stageManagedPathServerDir creates a throwaway Go server package used for path-based managed lifecycle tests.
func stageManagedPathServerDir(parseT *testing.T, parseRootPath string) string {
	parseT.Helper()
	parseServerDir := filepath.Join(parseRootPath, "examples", "custom-managed-server")
	if parseErr := os.MkdirAll(parseServerDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir managed path server dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseServerDir, "main.go"), []byte("package main\nfunc main(){}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write managed path server main.go: %v", parseErr2)
	}
	return parseServerDir
}

// TestRunExamplesManagedStartWritesRuntimeState verifies start writes profile runtime state under the default artifact root.
func TestRunExamplesManagedStartWritesRuntimeState(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)

	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}

	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		if parseConfig.listenAddr != "127.0.0.1:8095" {
			parseT.Fatalf("expected default listen addr, got %q", parseConfig.listenAddr)
		}
		return 43125, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		if parseState.PID != 43125 {
			parseT.Fatalf("expected pid in wait state, got %#v", parseState)
		}
		return nil
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseLauncher := launcher{repoRoot: parseRootPath}
	if parseErr2 := parseLauncher.runExamples([]string{"start"}); parseErr2 != nil {
		parseT.Fatalf("run examples start: %v", parseErr2)
	}

	parseStatePath := filepath.Join(parseRootPath, "bin", "runtime", "examples-servers", "chat-wizard-local.json")
	parsePayload, parseErr3 := os.ReadFile(parseStatePath)
	if parseErr3 != nil {
		parseT.Fatalf("read runtime state: %v", parseErr3)
	}
	var parseState examplesManagedServerState
	if parseErr4 := json.Unmarshal(parsePayload, &parseState); parseErr4 != nil {
		parseT.Fatalf("decode runtime state: %v", parseErr4)
	}
	if parseState.PID != 43125 || parseState.ListenAddr != "127.0.0.1:8095" || parseState.HealthURL != "http://127.0.0.1:8095/healthz" {
		parseT.Fatalf("unexpected state payload: %#v", parseState)
	}
	parseOutput, parseErr5 := parseStdout()
	if parseErr5 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr5)
	}
	if !strings.Contains(parseOutput, "GWC examples managed") || !strings.Contains(parseOutput, "profile started and passed health probe") {
		parseT.Fatalf("expected managed start summary output, got %q", parseOutput)
	}
}

// TestRunExamplesManagedStartAcceptsPathArgument verifies managed start can be keyed directly by a server path.
func TestRunExamplesManagedStartAcceptsPathArgument(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseServerDir := stageManagedPathServerDir(parseT, parseRootPath)

	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}

	var parseLaunchPath string
	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		if len(parseConfig.commandArgs) != 0 {
			parseT.Fatalf("expected managed launch to run the built binary directly, got args %#v", parseConfig.commandArgs)
		}
		parseLaunchPath = parseConfig.commandPath
		return 5719, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		return nil
	}

	parseLauncher := launcher{repoRoot: parseRootPath}
	if parseErr := parseLauncher.runExamples([]string{"start", parseServerDir}); parseErr != nil {
		parseT.Fatalf("run examples start by path: %v", parseErr)
	}

	parseAbsolutePath, parseErr := normalizePath(parseRootPath, parseServerDir)
	if parseErr != nil {
		parseT.Fatalf("resolve absolute path: %v", parseErr)
	}
	parseExpectedBinaryPath := buildExamplesManagedBinaryPath(filepath.Join(parseRootPath, "bin", "runtime", "examples-servers", buildExamplesManagedPathStateKey(parseAbsolutePath)+".json"))
	if parseLaunchPath != parseExpectedBinaryPath {
		parseT.Fatalf("expected launch binary path %q, got %q", parseExpectedBinaryPath, parseLaunchPath)
	}
	parseStateKey := buildExamplesManagedPathStateKey(parseAbsolutePath)
	parseStatePath, _, parseErr2 := parseLauncher.resolveExamplesManagedStatePaths(parseStateKey)
	if parseErr2 != nil {
		parseT.Fatalf("resolve path-based state path: %v", parseErr2)
	}
	parsePayload, parseErr3 := os.ReadFile(parseStatePath)
	if parseErr3 != nil {
		parseT.Fatalf("read path-based state: %v", parseErr3)
	}
	var parseState examplesManagedServerState
	if parseErr4 := json.Unmarshal(parsePayload, &parseState); parseErr4 != nil {
		parseT.Fatalf("decode path-based state: %v", parseErr4)
	}
	if parseState.ServerPath != parseAbsolutePath || parseState.ProfileName != parseStateKey {
		parseT.Fatalf("unexpected path-based state payload: %#v", parseState)
	}
}

// TestRunExamplesManagedStartUsesArtifactRootOverride verifies state artifacts honor gwc-runner artifactRoot.
func TestRunExamplesManagedStartUsesArtifactRootOverride(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)
	if parseErr := os.WriteFile(filepath.Join(parseRootPath, "gwc-runner.json"), []byte(`{"paths":{"artifactRoot":"enterprise-artifacts"}}`), 0644); parseErr != nil {
		parseT.Fatalf("write runner config: %v", parseErr)
	}

	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		return 9911, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		return nil
	}

	parseLauncher := launcher{repoRoot: parseRootPath}
	if parseErr2 := parseLauncher.runExamples([]string{"start", "-profile", "chat-wizard"}); parseErr2 != nil {
		parseT.Fatalf("run examples start with artifactRoot: %v", parseErr2)
	}
	parseStatePath, _, parseErr3 := parseLauncher.resolveExamplesManagedStatePaths("chat-wizard-local")
	if parseErr3 != nil {
		parseT.Fatalf("resolve managed state path: %v", parseErr3)
	}
	if _, parseErr3 := os.Stat(parseStatePath); parseErr3 != nil {
		parseT.Fatalf("expected state at artifactRoot path: %v", parseErr3)
	}
}

// TestRunExamplesManagedStopTerminatesAndClearsState verifies stop terminates running profile processes and removes state.
func TestRunExamplesManagedStopTerminatesAndClearsState(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)

	parseLauncher := launcher{repoRoot: parseRootPath}
	parseStatePath, parseLogPath, parseErr := parseLauncher.resolveExamplesManagedStatePaths("chat-wizard-local")
	if parseErr != nil {
		parseT.Fatalf("resolve state paths: %v", parseErr)
	}
	if parseErr2 := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		PID:         7788,
		URL:         "http://127.0.0.1:8095",
		HealthURL:   "http://127.0.0.1:8095/healthz",
		LogPath:     parseLogPath,
	}); parseErr2 != nil {
		parseT.Fatalf("write managed state: %v", parseErr2)
	}

	parseOriginalCheck := examplesManagedCheckPIDRunning
	parseOriginalTerminate := examplesManagedTerminatePIDTree
	parseT.Cleanup(func() {
		examplesManagedCheckPIDRunning = parseOriginalCheck
		examplesManagedTerminatePIDTree = parseOriginalTerminate
	})
	parseTerminatedPID := 0
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return parsePID == 7788
	}
	examplesManagedTerminatePIDTree = func(parsePID int) error {
		parseTerminatedPID = parsePID
		return nil
	}

	if parseErr3 := parseLauncher.runExamples([]string{"stop"}); parseErr3 != nil {
		parseT.Fatalf("run examples stop: %v", parseErr3)
	}
	if parseTerminatedPID != 7788 {
		parseT.Fatalf("expected stop to terminate pid 7788, got %d", parseTerminatedPID)
	}
	if _, parseErr4 := os.Stat(parseStatePath); !os.IsNotExist(parseErr4) {
		parseT.Fatalf("expected stop to remove state file, stat err=%v", parseErr4)
	}
}

// TestRunExamplesManagedStatusRemovesStaleState verifies status cleans stale state when pid no longer exists.
func TestRunExamplesManagedStatusRemovesStaleState(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)

	parseLauncher := launcher{repoRoot: parseRootPath}
	parseStatePath, parseLogPath, parseErr := parseLauncher.resolveExamplesManagedStatePaths("chat-wizard-local")
	if parseErr != nil {
		parseT.Fatalf("resolve state paths: %v", parseErr)
	}
	if parseErr2 := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		PID:         5566,
		URL:         "http://127.0.0.1:8095",
		HealthURL:   "http://127.0.0.1:8095/healthz",
		LogPath:     parseLogPath,
	}); parseErr2 != nil {
		parseT.Fatalf("write state: %v", parseErr2)
	}

	parseOriginalCheck := examplesManagedCheckPIDRunning
	parseT.Cleanup(func() {
		examplesManagedCheckPIDRunning = parseOriginalCheck
	})
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return false
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := parseLauncher.runExamples([]string{"status"}); parseErr4 != nil {
		parseT.Fatalf("run examples status: %v", parseErr4)
	}
	if _, parseErr5 := os.Stat(parseStatePath); !os.IsNotExist(parseErr5) {
		parseT.Fatalf("expected stale state removal, stat err=%v", parseErr5)
	}
	parseOutput, parseErr6 := parseStdout()
	if parseErr6 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr6)
	}
	if !strings.Contains(parseOutput, "stale state removed") {
		parseT.Fatalf("expected stale-state summary output, got %q", parseOutput)
	}
}

// TestRunExamplesManagedStartCleansUpOnHealthFailure verifies start terminates launched pid and clears state on readiness failure.
func TestRunExamplesManagedStartCleansUpOnHealthFailure(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)

	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalTerminate := examplesManagedTerminatePIDTree
	parseOriginalBuild := examplesManagedBuildBinary
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedTerminatePIDTree = parseOriginalTerminate
		examplesManagedBuildBinary = parseOriginalBuild
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}

	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		return 6611, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		return errors.New("health timeout")
	}
	parseTerminatedPID := 0
	examplesManagedTerminatePIDTree = func(parsePID int) error {
		parseTerminatedPID = parsePID
		return nil
	}

	parseLauncher := launcher{repoRoot: parseRootPath}
	parseErr := parseLauncher.runExamples([]string{"start"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "health timeout") {
		parseT.Fatalf("expected health timeout failure, got %v", parseErr)
	}
	if parseTerminatedPID != 6611 {
		parseT.Fatalf("expected failed start to terminate pid 6611, got %d", parseTerminatedPID)
	}
	parseStatePath := filepath.Join(parseRootPath, "bin", "runtime", "examples-servers", "chat-wizard-local.json")
	if _, parseErr2 := os.Stat(parseStatePath); !os.IsNotExist(parseErr2) {
		parseT.Fatalf("expected failed start to remove state file, stat err=%v", parseErr2)
	}
}

// TestDispatchCommandSupportsPathFirstManagedAction verifies path-first managed actions dispatch through `gwc <path> <action>`.
func TestDispatchCommandSupportsPathFirstManagedAction(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseServerDir := stageManagedPathServerDir(parseT, parseRootPath)
	parseLauncher := launcher{repoRoot: parseRootPath}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr2 := parseLauncher.dispatchCommand(parseServerDir, []string{"status", "-json"}); parseErr2 != nil {
		parseT.Fatalf("dispatch path-first managed status: %v", parseErr2)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr3)
	}
	if !strings.Contains(parseOutput, `"action": "status"`) {
		parseT.Fatalf("expected path-first status output, got %q", parseOutput)
	}
}

// TestRunExamplesSupportsPathThenAction verifies `gwc examples <path> <action>` dispatches to managed lifecycle flows.
func TestRunExamplesSupportsPathThenAction(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseServerDir := stageManagedPathServerDir(parseT, parseRootPath)

	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		return 9191, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		return nil
	}

	parseLauncher := launcher{repoRoot: parseRootPath}
	if parseErr := parseLauncher.runExamples([]string{parseServerDir, "start"}); parseErr != nil {
		parseT.Fatalf("run examples path-then-action start: %v", parseErr)
	}
}

// TestLauncherJSONRequestedForPathFirstManagedAction verifies path-first managed actions honor `-json` detection.
func TestLauncherJSONRequestedForPathFirstManagedAction(parseT *testing.T) {
	parsePath := filepath.Join(".", "examples", "100-ai-chat-wizard", "cmd", "server")
	if !launcherJSONRequestedForCommand(parsePath, []string{"status", "-json"}) {
		parseT.Fatalf("expected launcher json detection for path-first managed action")
	}
	if launcherJSONRequestedForCommand(parsePath, []string{"status"}) {
		parseT.Fatalf("expected launcher json detection to stay false without -json")
	}
}

// TestResolveExamplesManagedPathCommandUsesServerDirWhenOutsideRepoRoot verifies external server paths run from their own directory.
func TestResolveExamplesManagedPathCommandUsesServerDirWhenOutsideRepoRoot(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseExternalRoot := parseT.TempDir()
	parseServerDir := stageManagedPathServerDir(parseT, parseExternalRoot)

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	parseProfile, parseErr := parseLauncher.resolveExamplesManagedPathCommand(parseServerDir)
	if parseErr != nil {
		parseT.Fatalf("resolve managed path command: %v", parseErr)
	}
	if parseProfile.commandDir != parseServerDir {
		parseT.Fatalf("expected external path commandDir=%q, got %q", parseServerDir, parseProfile.commandDir)
	}
}
