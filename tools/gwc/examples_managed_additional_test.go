package main

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestIsExamplesManagedChatWizardDatabaseSeedRequired covers missing, empty, and populated local auth DB states.
func TestIsExamplesManagedChatWizardDatabaseSeedRequired(parseT *testing.T) {
	parseMissingPath := filepath.Join(parseT.TempDir(), "missing.db")
	if isParseSeedRequired, parseErr := isExamplesManagedChatWizardDatabaseSeedRequired(parseMissingPath); parseErr != nil || !isParseSeedRequired {
		parseT.Fatalf("expected missing db to require seed, got required=%t err=%v", isParseSeedRequired, parseErr)
	}

	parseDBPath := filepath.Join(parseT.TempDir(), "chat_history.db")
	parseDB, parseErr := sql.Open("sqlite3", "file:"+parseDBPath+"?_pragma=busy_timeout(5000)")
	if parseErr != nil {
		parseT.Fatalf("open sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	if _, parseErr := parseDB.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL, password_hash TEXT NOT NULL, created_at TEXT NOT NULL)`); parseErr != nil {
		parseT.Fatalf("create users table: %v", parseErr)
	}

	if isParseSeedRequired, parseErr := isExamplesManagedChatWizardDatabaseSeedRequired(parseDBPath); parseErr != nil || !isParseSeedRequired {
		parseT.Fatalf("expected empty users table to require seed, got required=%t err=%v", isParseSeedRequired, parseErr)
	}

	if _, parseErr := parseDB.Exec(`INSERT INTO users (email, password_hash, created_at) VALUES ('customer@email.com', 'hash', '2026-04-06T00:00:00Z')`); parseErr != nil {
		parseT.Fatalf("insert user row: %v", parseErr)
	}
	if isParseSeedRequired, parseErr := isExamplesManagedChatWizardDatabaseSeedRequired(parseDBPath); parseErr != nil || isParseSeedRequired {
		parseT.Fatalf("expected populated db to skip seed, got required=%t err=%v", isParseSeedRequired, parseErr)
	}
}

// TestResolveExamplesManagedHelperBranches covers pure path, profile, state, env, and log helper branches.
func TestResolveExamplesManagedHelperBranches(parseT *testing.T) {
	if _, parseErr := resolveExamplesManagedServerPath("", []string{"one", "two"}); parseErr == nil || !strings.Contains(parseErr.Error(), "expected at most one server path argument") {
		parseT.Fatalf("expected too-many-args error, got %v", parseErr)
	}
	if _, parseErr := resolveExamplesManagedServerPath("flag-path", []string{"positional"}); parseErr == nil || !strings.Contains(parseErr.Error(), "use either -path") {
		parseT.Fatalf("expected flag-plus-arg conflict, got %v", parseErr)
	}
	if parsePath, parseErr := resolveExamplesManagedServerPath("  flag-path  ", nil); parseErr != nil || parsePath != "flag-path" {
		parseT.Fatalf("expected trimmed flag path, got path=%q err=%v", parsePath, parseErr)
	}
	if parsePath2, parseErr := resolveExamplesManagedServerPath("", []string{"  positional  "}); parseErr != nil || parsePath2 != "positional" {
		parseT.Fatalf("expected positional path fallback, got path=%q err=%v", parsePath2, parseErr)
	}

	if parseGot := normalizeExamplesManagedHealthPath(""); parseGot != "/healthz" {
		parseT.Fatalf("expected default health path, got %q", parseGot)
	}
	if parseGot2 := normalizeExamplesManagedHealthPath("ready"); parseGot2 != "/ready" {
		parseT.Fatalf("expected rooted health path, got %q", parseGot2)
	}
	if parseGot3 := normalizeExamplesManagedHealthPath("/probe"); parseGot3 != "/probe" {
		parseT.Fatalf("expected existing rooted health path to pass through, got %q", parseGot3)
	}

	parseRootPath := parseT.TempDir()
	parseInsidePath := filepath.Join(parseRootPath, "examples", "server", "main.go")
	parseOutsidePath := filepath.Join(filepath.Dir(parseRootPath), "outside", "server", "main.go")
	if !isExamplesManagedPathUnderRoot(parseRootPath, parseRootPath) {
		parseT.Fatal("expected root path to count as under root")
	}
	if !isExamplesManagedPathUnderRoot(parseInsidePath, parseRootPath) {
		parseT.Fatal("expected nested path to count as under root")
	}
	if isExamplesManagedPathUnderRoot(parseOutsidePath, parseRootPath) {
		parseT.Fatal("expected outside path to be rejected as under root")
	}
	if isExamplesManagedPathUnderRoot("", parseRootPath) {
		parseT.Fatal("expected empty resolved path to be rejected")
	}

	if _, parseErr := normalizeExamplesManagedProfileName(""); parseErr == nil || !strings.Contains(parseErr.Error(), "profile name is required") {
		parseT.Fatalf("expected empty profile-name error, got %v", parseErr)
	}
	if _, parseErr := normalizeExamplesManagedProfileName("bad/profile"); parseErr == nil || !strings.Contains(parseErr.Error(), "is invalid") {
		parseT.Fatalf("expected invalid profile-name error, got %v", parseErr)
	}
	if parseProfileName, parseErr := normalizeExamplesManagedProfileName(" Chat-Wizard-Local "); parseErr != nil || parseProfileName != "chat-wizard-local" {
		parseT.Fatalf("expected normalized lowercase profile name, got %q err=%v", parseProfileName, parseErr)
	}

	parseStatePath := filepath.Join(parseRootPath, "runtime-state.json")
	if parseState, parseFound, parseErr := readExamplesManagedState(parseStatePath); parseErr != nil || parseFound || parseState.PID != 0 {
		parseT.Fatalf("expected missing state to be ignored, got state=%#v found=%t err=%v", parseState, parseFound, parseErr)
	}
	if parseErr := os.WriteFile(parseStatePath, []byte("{bad json"), 0644); parseErr != nil {
		parseT.Fatalf("write invalid state payload: %v", parseErr)
	}
	if _, parseFound, parseErr := readExamplesManagedState(parseStatePath); parseErr == nil || !parseFound || !strings.Contains(parseErr.Error(), "decode managed profile state") {
		parseT.Fatalf("expected invalid-state decode error, got found=%t err=%v", parseFound, parseErr)
	}
	if parseErr := removeExamplesManagedState(filepath.Join(parseRootPath, "missing.json")); parseErr != nil {
		parseT.Fatalf("expected missing state removal to be ignored, got %v", parseErr)
	}

	parseLogPath := filepath.Join(parseRootPath, "managed.log")
	if parseTail := readExamplesManagedLogTail(parseLogPath, 0); parseTail != "" {
		parseT.Fatalf("expected zero-byte tail request to be empty, got %q", parseTail)
	}
	if parseTail2 := readExamplesManagedLogTail(parseLogPath, 16); parseTail2 != "" {
		parseT.Fatalf("expected missing log tail to be empty, got %q", parseTail2)
	}
	if parseErr := os.WriteFile(parseLogPath, []byte("0123456789"), 0644); parseErr != nil {
		parseT.Fatalf("write managed log: %v", parseErr)
	}
	if parseTail3 := readExamplesManagedLogTail(parseLogPath, 4); parseTail3 != "6789" {
		parseT.Fatalf("expected truncated log tail, got %q", parseTail3)
	}

	parseT.Setenv("LISTEN_ADDR", "127.0.0.1:1")
	parseEnv := buildExamplesManagedProcessEnv("127.0.0.1:9999")
	parseListenCount := 0
	for _, parseEntry := range parseEnv {
		if strings.HasPrefix(parseEntry, "LISTEN_ADDR=") {
			parseListenCount++
			if parseEntry != "LISTEN_ADDR=127.0.0.1:9999" {
				parseT.Fatalf("expected managed process env to replace LISTEN_ADDR, got %q", parseEntry)
			}
		}
	}
	if parseListenCount != 1 {
		parseT.Fatalf("expected one LISTEN_ADDR entry, got %d in %#v", parseListenCount, parseEnv)
	}
}

// TestApplyExamplesManagedStatusBranchHelpers covers status summaries for missing, stale, unhealthy, and healthy managed profiles.
func TestApplyExamplesManagedStatusBranchHelpers(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)
	parseLauncher := launcher{repoRoot: parseRootPath}

	parseOriginalCheck := examplesManagedCheckPIDRunning
	parseT.Cleanup(func() {
		examplesManagedCheckPIDRunning = parseOriginalCheck
	})

	parseSummary, parseErr := parseLauncher.applyExamplesManagedStatus("chat-wizard-local", "")
	if parseErr != nil {
		parseT.Fatalf("apply status with no state: %v", parseErr)
	}
	if parseSummary.IsRunning || !strings.Contains(parseSummary.Message, "profile is not running") {
		parseT.Fatalf("expected not-running summary with no state, got %#v", parseSummary)
	}

	parseStatePath, parseLogPath, parseErr := parseLauncher.resolveExamplesManagedStatePaths("chat-wizard-local")
	if parseErr != nil {
		parseT.Fatalf("resolve managed state paths: %v", parseErr)
	}
	if parseErr := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		ServerPath:  filepath.Join(parseRootPath, "examples", "100-ai-chat-wizard", "cmd", "server"),
		PID:         4444,
		URL:         "http://127.0.0.1:8095",
		HealthURL:   "http://127.0.0.1:8095/healthz",
		LogPath:     parseLogPath,
	}); parseErr != nil {
		parseT.Fatalf("write stale state: %v", parseErr)
	}
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return false
	}
	parseSummary, parseErr = parseLauncher.applyExamplesManagedStatus("chat-wizard-local", "")
	if parseErr != nil {
		parseT.Fatalf("apply status with stale state: %v", parseErr)
	}
	if parseSummary.IsRunning || !strings.Contains(parseSummary.Message, "stale state removed") {
		parseT.Fatalf("expected stale-state summary, got %#v", parseSummary)
	}
	if _, parseStatErr := os.Stat(parseStatePath); !os.IsNotExist(parseStatErr) {
		parseT.Fatalf("expected stale state file removal, stat err=%v", parseStatErr)
	}

	if parseErr := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		PID:         5555,
		URL:         "http://127.0.0.1:8095",
		HealthURL:   "http://127.0.0.1:1/healthz",
		LogPath:     parseLogPath,
	}); parseErr != nil {
		parseT.Fatalf("write probe-failure state: %v", parseErr)
	}
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return true
	}
	parseSummary, parseErr = parseLauncher.applyExamplesManagedStatus("chat-wizard-local", "")
	if parseErr != nil {
		parseT.Fatalf("apply status with probe failure: %v", parseErr)
	}
	if !parseSummary.IsRunning || parseSummary.IsHealthy || !strings.Contains(parseSummary.Message, "health probe failed") {
		parseT.Fatalf("expected running-but-probe-failed summary, got %#v", parseSummary)
	}

	parseUnhealthyServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer parseUnhealthyServer.Close()
	if parseErr := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		PID:         6666,
		URL:         parseUnhealthyServer.URL,
		HealthURL:   parseUnhealthyServer.URL,
		LogPath:     parseLogPath,
	}); parseErr != nil {
		parseT.Fatalf("write unhealthy state: %v", parseErr)
	}
	parseSummary, parseErr = parseLauncher.applyExamplesManagedStatus("chat-wizard-local", "")
	if parseErr != nil {
		parseT.Fatalf("apply status with unhealthy health endpoint: %v", parseErr)
	}
	if !parseSummary.IsRunning || parseSummary.IsHealthy || !strings.Contains(parseSummary.Message, "did not return 2xx") {
		parseT.Fatalf("expected running-but-unhealthy summary, got %#v", parseSummary)
	}

	parseHealthyServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
	}))
	defer parseHealthyServer.Close()
	if parseErr := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		PID:         7777,
		URL:         parseHealthyServer.URL,
		HealthURL:   parseHealthyServer.URL,
		LogPath:     parseLogPath,
	}); parseErr != nil {
		parseT.Fatalf("write healthy state: %v", parseErr)
	}
	parseSummary, parseErr = parseLauncher.applyExamplesManagedStatus("chat-wizard-local", "")
	if parseErr != nil {
		parseT.Fatalf("apply status with healthy endpoint: %v", parseErr)
	}
	if !parseSummary.IsRunning || !parseSummary.IsHealthy || !strings.Contains(parseSummary.Message, "running and healthy") {
		parseT.Fatalf("expected healthy running summary, got %#v", parseSummary)
	}
}

// TestApplyExamplesManagedStopAndRestartBranchHelpers covers stop error branches and restart state transitions.
func TestApplyExamplesManagedStopAndRestartBranchHelpers(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	stageManagedProfileCommandDir(parseT, parseRootPath)
	parseLauncher := launcher{repoRoot: parseRootPath}

	parseOriginalCheck := examplesManagedCheckPIDRunning
	parseOriginalTerminate := examplesManagedTerminatePIDTree
	parseOriginalLaunch := examplesManagedLaunchProcess
	parseOriginalWait := examplesManagedWaitServerReady
	parseOriginalBuild := examplesManagedBuildBinary
	parseT.Cleanup(func() {
		examplesManagedCheckPIDRunning = parseOriginalCheck
		examplesManagedTerminatePIDTree = parseOriginalTerminate
		examplesManagedLaunchProcess = parseOriginalLaunch
		examplesManagedWaitServerReady = parseOriginalWait
		examplesManagedBuildBinary = parseOriginalBuild
	})

	parseSummary, parseErr := parseLauncher.applyExamplesManagedStop("chat-wizard-local", "")
	if parseErr != nil {
		parseT.Fatalf("apply stop with no state: %v", parseErr)
	}
	if parseSummary.IsRunning || !strings.Contains(parseSummary.Message, "profile is not running") {
		parseT.Fatalf("expected not-running stop summary, got %#v", parseSummary)
	}

	parseStatePath, parseLogPath, parseErr := parseLauncher.resolveExamplesManagedStatePaths("chat-wizard-local")
	if parseErr != nil {
		parseT.Fatalf("resolve stop state paths: %v", parseErr)
	}
	if parseErr := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		PID:         8888,
		URL:         "http://127.0.0.1:8095",
		HealthURL:   "http://127.0.0.1:8095/healthz",
		LogPath:     parseLogPath,
	}); parseErr != nil {
		parseT.Fatalf("write stop-error state: %v", parseErr)
	}
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return true
	}
	examplesManagedTerminatePIDTree = func(parsePID int) error {
		return errors.New("terminate boom")
	}
	if _, parseErr := parseLauncher.applyExamplesManagedStop("chat-wizard-local", ""); parseErr == nil || !strings.Contains(parseErr.Error(), "terminate boom") {
		parseT.Fatalf("expected terminate error from stop, got %v", parseErr)
	}

	if parseErr := writeExamplesManagedState(parseStatePath, examplesManagedServerState{
		ProfileName: "chat-wizard-local",
		PID:         9999,
		URL:         "http://127.0.0.1:8095",
		HealthURL:   "http://127.0.0.1:8095/healthz",
		LogPath:     parseLogPath,
	}); parseErr != nil {
		parseT.Fatalf("write restart seed state: %v", parseErr)
	}
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return parsePID == 9999
	}
	parseTerminatedPID := 0
	examplesManagedTerminatePIDTree = func(parsePID int) error {
		parseTerminatedPID = parsePID
		return nil
	}
	examplesManagedBuildBinary = func(parseTargetPath string, parseBinaryPath string, parseWorkingDir string) error {
		return nil
	}
	examplesManagedLaunchProcess = func(parseConfig examplesManagedLaunchConfig) (int, error) {
		return 10001, nil
	}
	examplesManagedWaitServerReady = func(parseState examplesManagedServerState, parseTimeout time.Duration) error {
		return nil
	}
	parseSummary, parseErr = parseLauncher.applyExamplesManagedRestart("chat-wizard-local", "", "", "", "", 5*time.Second)
	if parseErr != nil {
		parseT.Fatalf("apply restart: %v", parseErr)
	}
	if parseTerminatedPID != 9999 {
		parseT.Fatalf("expected restart to terminate prior pid 9999, got %d", parseTerminatedPID)
	}
	if parseSummary.Action != "restart" || parseSummary.PID != 10001 || !parseSummary.IsRunning || !parseSummary.IsHealthy {
		parseT.Fatalf("expected running restart summary, got %#v", parseSummary)
	}
	if !strings.Contains(parseSummary.Message, "profile process tree terminated") || !strings.Contains(parseSummary.Message, "profile started and passed health probe") {
		parseT.Fatalf("expected restart summary to compose stop/start messages, got %q", parseSummary.Message)
	}
}

// TestWaitExamplesManagedServerReadyBranches covers fast-success, fast-failure, and timeout branches for readiness polling.
func TestWaitExamplesManagedServerReadyBranches(parseT *testing.T) {
	parseOriginalCheck := examplesManagedCheckPIDRunning
	parseT.Cleanup(func() {
		examplesManagedCheckPIDRunning = parseOriginalCheck
	})

	if parseErr := waitExamplesManagedServerReady(examplesManagedServerState{}, 0); parseErr != nil {
		parseT.Fatalf("expected zero timeout to skip readiness polling, got %v", parseErr)
	}

	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return false
	}
	if parseErr := waitExamplesManagedServerReady(examplesManagedServerState{PID: 1, HealthURL: "http://127.0.0.1:1"}, time.Second); parseErr == nil || !strings.Contains(parseErr.Error(), "exited before reporting healthy") {
		parseT.Fatalf("expected exited-process readiness error, got %v", parseErr)
	}

	parseHealthyServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusOK)
	}))
	defer parseHealthyServer.Close()
	examplesManagedCheckPIDRunning = func(parsePID int) bool {
		return true
	}
	if parseErr := waitExamplesManagedServerReady(examplesManagedServerState{PID: 2, HealthURL: parseHealthyServer.URL}, time.Second); parseErr != nil {
		parseT.Fatalf("expected healthy readiness probe to succeed, got %v", parseErr)
	}

	if parseErr := waitExamplesManagedServerReady(examplesManagedServerState{PID: 3, HealthURL: "http://127.0.0.1:1"}, time.Nanosecond); parseErr == nil || !strings.Contains(parseErr.Error(), "timed out waiting for managed profile health") {
		parseT.Fatalf("expected readiness timeout with probe error, got %v", parseErr)
	}

	parseUnhealthyServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer parseUnhealthyServer.Close()
	if parseErr := waitExamplesManagedServerReady(examplesManagedServerState{PID: 4, HealthURL: parseUnhealthyServer.URL}, time.Nanosecond); parseErr == nil || parseErr.Error() != "timed out waiting for managed profile health" {
		parseT.Fatalf("expected readiness timeout without probe error, got %v", parseErr)
	}
}
