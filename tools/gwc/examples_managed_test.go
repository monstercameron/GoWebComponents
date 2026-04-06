package main

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
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

// stageManagedProfileSeedCommandDir creates the managed chat-wizard seed command directory expected by launch preflight.
func stageManagedProfileSeedCommandDir(parseT *testing.T, parseRootPath string) {
	parseT.Helper()
	parseCommandDir := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	if parseErr := os.MkdirAll(parseCommandDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir managed seed command dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseCommandDir, "main.go"), []byte("package main\nfunc main(){}\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write managed seed command placeholder: %v", parseErr2)
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

// stageManagedPathHealthServerFile creates a temporary Go main file that serves /healthz at LISTEN_ADDR.
func stageManagedPathHealthServerFile(parseT *testing.T, parseRootPath string) string {
	parseT.Helper()
	parseServerDir := filepath.Join(parseRootPath, "examples", "custom-managed-health-server")
	if parseErr := os.MkdirAll(parseServerDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir managed health server dir: %v", parseErr)
	}
	parseMainPath := filepath.Join(parseServerDir, "main.go")
	parseContent := `package main

import (
	"net/http"
	"os"
)

func main() {
	parseAddr := os.Getenv("LISTEN_ADDR")
	if parseAddr == "" {
		parseAddr = "127.0.0.1:8095"
	}
	http.HandleFunc("/healthz", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
		_, _ = parseW.Write([]byte("ok"))
	})
	_ = http.ListenAndServe(parseAddr, nil)
}
`
	if parseErr2 := os.WriteFile(parseMainPath, []byte(parseContent), 0644); parseErr2 != nil {
		parseT.Fatalf("write managed health server main.go: %v", parseErr2)
	}
	return parseMainPath
}

// getManagedTestFreePort reserves and returns a local TCP port for managed lifecycle tests.
func getManagedTestFreePort(parseT *testing.T) string {
	parseT.Helper()
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("reserve free port: %v", parseErr)
	}
	defer parseListener.Close()
	parsePort := parseListener.Addr().(*net.TCPAddr).Port
	return strconv.Itoa(parsePort)
}

// TestIsExamplesManagedActionIncludesRestart verifies restart is recognized as a managed lifecycle action.
func TestIsExamplesManagedActionIncludesRestart(parseT *testing.T) {
	if !isExamplesManagedAction([]string{"restart"}) {
		parseT.Fatalf("expected restart to be treated as a managed action")
	}
	if !isExamplesManagedAction([]string{"ReStaRt"}) {
		parseT.Fatalf("expected restart detection to be case-insensitive")
	}
}

// TestRunExamplesManagedStartWritesRuntimeState verifies start writes profile runtime state under the default artifact root.
func TestRunExamplesManagedStartWritesRuntimeState(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)

	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseOriginalBuildWASM := examplesManagedBuildWASM
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
		examplesManagedBuildWASM = parseOriginalBuildWASM
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedBuildWASM = func(parseTargetPath string, parseOutputPath string, parseWorkingDir string) error {
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
	parseOriginalBuildWASM := examplesManagedBuildWASM
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
		examplesManagedBuildWASM = parseOriginalBuildWASM
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedBuildWASM = func(parseTargetPath string, parseOutputPath string, parseWorkingDir string) error {
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

// TestResolveExamplesManagedLaunchConfigBuildsWASMArtifactsForDefaultProfile verifies the managed chat-wizard profile builds browser wasm artifacts before launching the server.
func TestResolveExamplesManagedLaunchConfigBuildsWASMArtifactsForDefaultProfile(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)

	parseOriginalBuild := examplesManagedBuildBinary
	parseOriginalBuildWASM := examplesManagedBuildWASM
	parseT.Cleanup(func() {
		examplesManagedBuildBinary = parseOriginalBuild
		examplesManagedBuildWASM = parseOriginalBuildWASM
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}

	parseWASMCalls := make([]string, 0, 2)
	examplesManagedBuildWASM = func(parseTargetPath string, parseOutputPath string, parseWorkingDir string) error {
		parseWASMCalls = append(parseWASMCalls, parseTargetPath+"|"+parseOutputPath+"|"+parseWorkingDir)
		return nil
	}

	parseLauncher := launcher{repoRoot: parseRootPath}
	parseProfile, parseErr := resolveExamplesManagedProfile(parseLauncher, "chat-wizard")
	if parseErr != nil {
		parseT.Fatalf("resolve managed profile: %v", parseErr)
	}

	parseStatePath := filepath.Join(parseRootPath, "bin", "runtime", "examples-servers", "chat-wizard-local.json")
	parseLogPath := filepath.Join(parseRootPath, "bin", "runtime", "examples-servers", "chat-wizard-local.log")
	if _, parseErr2 := parseLauncher.resolveExamplesManagedLaunchConfig(parseProfile, parseStatePath, parseLogPath, "127.0.0.1:8095"); parseErr2 != nil {
		parseT.Fatalf("resolve managed launch config: %v", parseErr2)
	}

	if len(parseWASMCalls) != 2 {
		parseT.Fatalf("expected two wasm build calls, got %#v", parseWASMCalls)
	}

	parseWantClientCall := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "client") +
		"|" + filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "bin", "client", "app", "chat.wasm") +
		"|" + parseRootPath
	parseWantWorkerCall := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "client", "backgroundworker") +
		"|" + filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "bin", "client", "worker", "background-worker.wasm") +
		"|" + parseRootPath
	if parseWASMCalls[0] != parseWantClientCall {
		parseT.Fatalf("unexpected client wasm build call: got %q want %q", parseWASMCalls[0], parseWantClientCall)
	}
	if parseWASMCalls[1] != parseWantWorkerCall {
		parseT.Fatalf("unexpected worker wasm build call: got %q want %q", parseWASMCalls[1], parseWantWorkerCall)
	}
}

// TestResolveExamplesManagedPathCommandBuildsWASMArtifactsForChatWizardServer verifies the path-based chat wizard flow reuses the managed wasm targets.
func TestResolveExamplesManagedPathCommandBuildsWASMArtifactsForChatWizardServer(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)

	parseLauncher := launcher{repoRoot: parseRootPath}
	parseProfile, parseErr := parseLauncher.resolveExamplesManagedPathCommand(filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "cmd", "server"))
	if parseErr != nil {
		parseT.Fatalf("resolve managed path command: %v", parseErr)
	}
	if len(parseProfile.buildWASMTargets) != 2 {
		parseT.Fatalf("expected two wasm build targets, got %#v", parseProfile.buildWASMTargets)
	}
	if parseProfile.buildWASMTargets[0].outputPath != filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "bin", "client", "app", "chat.wasm") {
		parseT.Fatalf("unexpected client wasm output path: %#v", parseProfile.buildWASMTargets[0])
	}
	if parseProfile.buildWASMTargets[1].outputPath != filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "bin", "client", "worker", "background-worker.wasm") {
		parseT.Fatalf("unexpected worker wasm output path: %#v", parseProfile.buildWASMTargets[1])
	}
}

// TestApplyExamplesManagedStartSeedsMissingChatWizardDatabase verifies managed chat-wizard start seeds the runtime DB when local auth state is missing.
func TestApplyExamplesManagedStartSeedsMissingChatWizardDatabase(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)
	stageManagedProfileSeedCommandDir(parseT, parseRootPath)

	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseOriginalBuildWASM := examplesManagedBuildWASM
	parseOriginalExecuteSeed := executeExamplesManagedSeed
	parseOriginalInspectSeedState := inspectExamplesManagedChatWizardDatabaseSeedState
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
		examplesManagedBuildWASM = parseOriginalBuildWASM
		executeExamplesManagedSeed = parseOriginalExecuteSeed
		inspectExamplesManagedChatWizardDatabaseSeedState = parseOriginalInspectSeedState
	})

	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedBuildWASM = func(parseTargetPath string, parseOutputPath string, parseWorkingDir string) error {
		return nil
	}

	parseCapturedSeedConfig := seedConfig{}
	inspectExamplesManagedChatWizardDatabaseSeedState = func(parseDBPath string) (bool, error) {
		return true, nil
	}
	executeExamplesManagedSeed = func(parseConfig seedConfig) (seedSummary, error) {
		parseCapturedSeedConfig = parseConfig
		return seedSummary{OK: true, DatabasePath: parseConfig.dbPath}, nil
	}
	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		return 9091, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		return nil
	}

	parseLauncher := launcher{repoRoot: parseRootPath}
	parseSummary, parseErr := parseLauncher.applyExamplesManagedStart("chat-wizard", "", "", "", "", 5*time.Second)
	if parseErr != nil {
		parseT.Fatalf("apply managed start: %v", parseErr)
	}
	if !parseSummary.OK || parseSummary.PID != 9091 {
		parseT.Fatalf("unexpected managed start summary: %#v", parseSummary)
	}
	parseWantDB := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "bin", "runtime", "chat_history.db")
	if parseCapturedSeedConfig.dbPath != parseWantDB {
		parseT.Fatalf("expected seed db path %q, got %#v", parseWantDB, parseCapturedSeedConfig)
	}
	parseWantSeedCommandPath := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "cmd", "seed-test-db")
	if parseCapturedSeedConfig.commandPath != parseWantSeedCommandPath {
		parseT.Fatalf("expected seed command path %q, got %#v", parseWantSeedCommandPath, parseCapturedSeedConfig)
	}
}

// TestBuildExamplesManagedWASMArtifactWritesBrotliSidecar verifies managed wasm builds emit the Brotli sidecar required by the chat bootstrap.
func TestBuildExamplesManagedWASMArtifactWritesBrotliSidecar(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseTargetPath := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "client")
	parseOutputPath := filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "bin", "client", "app", "chat.wasm")
	parseWorkingDir := parseRootPath

	parseOriginalRun := launcherRunCommand
	parseOriginalWriteBrotli := writeExamplesManagedBrotliSidecar
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRun
		writeExamplesManagedBrotliSidecar = parseOriginalWriteBrotli
	})

	parseBuildCalls := []string{}
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCWD string, parseEnv []string) (string, error) {
		parseBuildCalls = append(parseBuildCalls, parseCommand+"|"+strings.Join(parseArgs, " ")+"|"+parseCWD)
		if parseErr := os.MkdirAll(filepath.Dir(parseOutputPath), 0o755); parseErr != nil {
			parseT.Fatalf("mkdir output dir in stub: %v", parseErr)
		}
		if parseErr := os.WriteFile(parseOutputPath, []byte("wasm"), 0o644); parseErr != nil {
			parseT.Fatalf("write wasm output in stub: %v", parseErr)
		}
		return "", nil
	}

	parseBrotliCalls := []string{}
	writeExamplesManagedBrotliSidecar = func(parseSourcePath string, parseTargetPath string) error {
		parseBrotliCalls = append(parseBrotliCalls, parseSourcePath+"|"+parseTargetPath)
		if parseSourcePath != parseOutputPath {
			parseT.Fatalf("unexpected brotli source path %q", parseSourcePath)
		}
		if parseErr := os.WriteFile(parseTargetPath, []byte("brotli"), 0o644); parseErr != nil {
			parseT.Fatalf("write brotli output in stub: %v", parseErr)
		}
		return nil
	}

	if parseErr := buildExamplesManagedWASMArtifact(parseTargetPath, parseOutputPath, parseWorkingDir); parseErr != nil {
		parseT.Fatalf("build managed wasm artifact: %v", parseErr)
	}
	if len(parseBuildCalls) != 1 {
		parseT.Fatalf("expected one wasm build call, got %#v", parseBuildCalls)
	}
	if len(parseBrotliCalls) != 1 {
		parseT.Fatalf("expected one brotli sidecar call, got %#v", parseBrotliCalls)
	}
	if _, parseErr := os.Stat(parseOutputPath + ".br"); parseErr != nil {
		parseT.Fatalf("expected brotli sidecar output: %v", parseErr)
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

// TestRunExamplesManagedRestartTerminatesAndStarts verifies restart terminates stale state and starts a fresh process.
func TestRunExamplesManagedRestartTerminatesAndStarts(parseT *testing.T) {
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
	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseOriginalBuildWASM := examplesManagedBuildWASM
	parseT.Cleanup(func() {
		examplesManagedCheckPIDRunning = parseOriginalCheck
		examplesManagedTerminatePIDTree = parseOriginalTerminate
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
		examplesManagedBuildWASM = parseOriginalBuildWASM
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedBuildWASM = func(parseTargetPath string, parseOutputPath string, parseWorkingDir string) error {
		return nil
	}

	parseTerminatedPID := 0
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return parsePID == 7788
	}
	examplesManagedTerminatePIDTree = func(parsePID int) error {
		parseTerminatedPID = parsePID
		return nil
	}
	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		return 9901, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		return nil
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := parseLauncher.runExamples([]string{"restart", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run examples restart: %v", parseErr4)
	}
	if parseTerminatedPID != 7788 {
		parseT.Fatalf("expected restart to terminate pid 7788, got %d", parseTerminatedPID)
	}
	parseOutput, parseErr5 := parseStdout()
	if parseErr5 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr5)
	}
	var parseSummary examplesManagedSummary
	if parseErr6 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr6 != nil {
		parseT.Fatalf("decode restart summary: %v", parseErr6)
	}
	if parseSummary.Action != "restart" || !parseSummary.IsRunning || !parseSummary.IsHealthy || parseSummary.PID != 9901 {
		parseT.Fatalf("unexpected restart summary: %#v", parseSummary)
	}
	if !strings.Contains(parseSummary.Message, "profile process tree terminated") || !strings.Contains(parseSummary.Message, "profile started and passed health probe") {
		parseT.Fatalf("expected restart message to include stop+start context, got %q", parseSummary.Message)
	}

	parsePayload, parseErr7 := os.ReadFile(parseStatePath)
	if parseErr7 != nil {
		parseT.Fatalf("read restarted state: %v", parseErr7)
	}
	var parseState examplesManagedServerState
	if parseErr8 := json.Unmarshal(parsePayload, &parseState); parseErr8 != nil {
		parseT.Fatalf("decode restarted state: %v", parseErr8)
	}
	if parseState.PID != 9901 {
		parseT.Fatalf("expected restarted state pid 9901, got %#v", parseState)
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
	parseOriginalBuildWASM := examplesManagedBuildWASM
	parseT.Cleanup(func() {
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedTerminatePIDTree = parseOriginalTerminate
		examplesManagedBuildBinary = parseOriginalBuild
		examplesManagedBuildWASM = parseOriginalBuildWASM
	})
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedBuildWASM = func(parseTargetPath string, parseOutputPath string, parseWorkingDir string) error {
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
	if parseErr := parseLauncher.runExamples([]string{parseServerDir, "restart"}); parseErr != nil {
		parseT.Fatalf("run examples path-then-action restart: %v", parseErr)
	}
}

// TestLauncherJSONRequestedForPathFirstManagedAction verifies path-first managed actions honor `-json` detection.
func TestLauncherJSONRequestedForPathFirstManagedAction(parseT *testing.T) {
	parsePath := filepath.Join(".", "examples", "100-ai-chat-wizard", "cmd", "server")
	if !launcherJSONRequestedForCommand(parsePath, []string{"status", "-json"}) {
		parseT.Fatalf("expected launcher json detection for path-first managed action")
	}
	if !launcherJSONRequestedForCommand(parsePath, []string{"restart", "-json"}) {
		parseT.Fatalf("expected launcher json detection for path-first restart action")
	}
	if launcherJSONRequestedForCommand(parsePath, []string{"status"}) {
		parseT.Fatalf("expected launcher json detection to stay false without -json")
	}
	if launcherJSONRequestedForCommand(parsePath, []string{"restart"}) {
		parseT.Fatalf("expected launcher json detection to stay false for restart without -json")
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

// TestRunExamplesManagedPathLifecycleLeavesNoOrphanPID verifies real path lifecycle stop kills the process and closes the port.
func TestRunExamplesManagedPathLifecycleLeavesNoOrphanPID(parseT *testing.T) {
	parseRepoRoot := parseT.TempDir()
	parseExternalRoot := parseT.TempDir()
	parseServerPath := stageManagedPathHealthServerFile(parseT, parseExternalRoot)
	parsePort := getManagedTestFreePort(parseT)

	parseLauncher := launcher{repoRoot: parseRepoRoot}
	if parseErr := parseLauncher.runExamples([]string{parseServerPath, "start", "-port", parsePort, "-health-timeout", "20s"}); parseErr != nil {
		parseT.Fatalf("run examples path start: %v", parseErr)
	}

	parseAbsolutePath, parseErr := normalizePath(parseRepoRoot, parseServerPath)
	if parseErr != nil {
		parseT.Fatalf("resolve path: %v", parseErr)
	}
	parseStateKey := buildExamplesManagedPathStateKey(parseAbsolutePath)
	parseStatePath, _, parseErr2 := parseLauncher.resolveExamplesManagedStatePaths(parseStateKey)
	if parseErr2 != nil {
		parseT.Fatalf("resolve managed state path: %v", parseErr2)
	}
	parseState, parseFound, parseErr3 := readExamplesManagedState(parseStatePath)
	if parseErr3 != nil {
		parseT.Fatalf("read managed state: %v", parseErr3)
	}
	if !parseFound || parseState.PID <= 0 {
		parseT.Fatalf("expected running managed state with pid, got found=%t state=%#v", parseFound, parseState)
	}

	if parseErr4 := parseLauncher.runExamples([]string{parseServerPath, "stop"}); parseErr4 != nil {
		parseT.Fatalf("run examples path stop: %v", parseErr4)
	}

	parseDeadline := time.Now().Add(4 * time.Second)
	for {
		parsePIDRunning := checkLauncherPIDRunning(parseState.PID)
		parseConnected := false
		parseConn, parseDialErr := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", parsePort), 150*time.Millisecond)
		if parseDialErr == nil {
			parseConnected = true
			_ = parseConn.Close()
		}
		if !parsePIDRunning && !parseConnected {
			break
		}
		if time.Now().After(parseDeadline) {
			parseT.Fatalf("expected stop to leave no orphan process/listener (pidRunning=%t connected=%t pid=%d port=%s)", parsePIDRunning, parseConnected, parseState.PID, parsePort)
		}
		time.Sleep(120 * time.Millisecond)
	}

	if _, parseErr5 := os.Stat(parseStatePath); !os.IsNotExist(parseErr5) {
		parseT.Fatalf("expected managed state removal after stop, stat err=%v", parseErr5)
	}
}
