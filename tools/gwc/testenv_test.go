package main

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// testEnvCleanupWindow bounds how long cleanup retries a locked file on Windows.
const testEnvCleanupWindow = 10 * time.Second

// useIsolatedGoTmp points GOTMPDIR at a fresh per-test directory so a leaked, locked
// build executable lands somewhere the test controls instead of the global temp dir.
// The directory is removed with retries after the test; a lock that outlives the retry
// window is logged, not failed, because it is the host's file-scanner holding the exe.
func useIsolatedGoTmp(parseT *testing.T) string {
	parseT.Helper()
	parseDir, parseErr := os.MkdirTemp("", "gwc-gotmp-")
	if parseErr != nil {
		parseT.Fatalf("create isolated GOTMPDIR: %v", parseErr)
	}
	parseT.Setenv("GOTMPDIR", parseDir)
	parseT.Cleanup(func() {
		if parseRemoveErr := removeAllWithRetry(parseDir, testEnvCleanupWindow); parseRemoveErr != nil {
			parseT.Logf("isolated GOTMPDIR %s not fully removed: %v", parseDir, parseRemoveErr)
		}
	})
	return parseDir
}

// removeAllWithRetry removes parsePath, retrying with backoff while the OS reports a
// transient lock, until parseWindow elapses. It returns the last error if it never clears.
func removeAllWithRetry(parsePath string, parseWindow time.Duration) error {
	parseDeadline := time.Now().Add(parseWindow)
	parseDelay := 50 * time.Millisecond
	for {
		parseErr := os.RemoveAll(parsePath)
		if parseErr == nil {
			return nil
		}
		if time.Now().Add(parseDelay).After(parseDeadline) {
			return parseErr
		}
		time.Sleep(parseDelay)
		if parseDelay < time.Second {
			parseDelay *= 2
		}
	}
}

// isBenignGoCleanupOutput reports whether a failed child `go test`/`go build` run only
// failed to delete its freshly built *.exe after a passing run. It delegates to the
// shared production classifier, which never reports a FAIL, panic or other error as benign.
func isBenignGoCleanupOutput(parseOutput string) bool {
	_, parseIsBenign := classifyBenchBenignExit(normalizeChildGoOutput(parseOutput))
	return parseIsBenign
}

// isCleanupOnlyStderr reports whether stderr holds only unlinkat/Access-denied *.exe
// cleanup lines; it delegates to the shared production helper.
func isCleanupOnlyStderr(parseStderr string) bool {
	return isGoCleanupOnlyOutput(parseStderr)
}

// tolerateGoCleanupError returns nil when parseErr is only the benign Windows exe-cleanup
// failure described by parseOutput, logging it; otherwise it returns parseErr unchanged.
func tolerateGoCleanupError(parseT *testing.T, parseErr error, parseOutput string) error {
	parseT.Helper()
	if parseErr == nil {
		return nil
	}
	if isBenignGoCleanupOutput(parseOutput) {
		parseT.Logf("tolerating benign Windows exe cleanup failure after passing child run: %v", parseErr)
		return nil
	}
	return parseErr
}

// runChildGoCommand runs parseCmd and returns its combined output; a non-zero exit that is
// only the benign exe-cleanup failure after a passing run is treated as success.
func runChildGoCommand(parseT *testing.T, parseCmd *exec.Cmd) (string, error) {
	parseT.Helper()
	parseOutput, parseErr := parseCmd.CombinedOutput()
	return string(parseOutput), tolerateGoCleanupError(parseT, parseErr, string(parseOutput))
}

// useBenignTolerantGoTests wraps verifyRunGoTests for one test so the in-process verify
// flow survives the benign exe-cleanup failure but still fails on any real test failure.
func useBenignTolerantGoTests(parseT *testing.T) {
	parseT.Helper()
	parseOriginal := verifyRunGoTests
	verifyRunGoTests = func(parseRoot string) (string, error) {
		parseOutput, parseErr := parseOriginal(parseRoot)
		if parseErr != nil && isBenignGoCleanupOutput(parseErr.Error()) {
			parseT.Logf("tolerating benign Windows exe cleanup failure in verify go test: %v", parseErr)
			return parseOutput, nil
		}
		return parseOutput, parseErr
	}
	parseT.Cleanup(func() { verifyRunGoTests = parseOriginal })
}

// useBenignTolerantLauncherRuns wraps the launcher's nested `go test` runners for one test
// so an in-process launcher command survives the benign exe-cleanup failure; any other
// failure, including a FAIL or panic line next to the cleanup error, is returned unchanged.
func useBenignTolerantLauncherRuns(parseT *testing.T) {
	parseT.Helper()
	useBenignTolerantGoTests(parseT)
	parseOriginal := launcherRunCommand
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		parseOutput, parseErr := parseOriginal(parseCommand, parseArgs, parseCwd, parseEnv)
		if parseErr != nil && isBenignGoCleanupOutput(parseOutput+"\n"+parseErr.Error()) {
			parseT.Logf("tolerating benign Windows exe cleanup failure in launcher run: %v", parseErr)
			return parseOutput, nil
		}
		return parseOutput, parseErr
	}
	parseT.Cleanup(func() { launcherRunCommand = parseOriginal })
}

// runLauncherCapturingStdout runs a launcher command in-process and returns what it printed
// to stdout. A goroutine drains the pipe so large outputs cannot block the command; a
// command error fails the test with the captured output attached.
func runLauncherCapturingStdout(parseT *testing.T, parseLauncher launcher, parseArgs []string) string {
	parseT.Helper()
	parseReader, parseWriter, parsePipeErr := os.Pipe()
	if parsePipeErr != nil {
		parseT.Fatalf("create stdout pipe: %v", parsePipeErr)
	}
	parseOriginalStdout := os.Stdout
	os.Stdout = parseWriter
	parseDone := make(chan string, 1)
	go func() {
		parseBytes, _ := io.ReadAll(parseReader)
		parseDone <- string(parseBytes)
	}()
	parseRunErr := parseLauncher.run(parseArgs)
	os.Stdout = parseOriginalStdout
	_ = parseWriter.Close()
	parseOutput := <-parseDone
	_ = parseReader.Close()
	if parseRunErr != nil {
		parseT.Fatalf("run launcher command %q: %v\n%s", strings.Join(parseArgs, " "), parseRunErr, parseOutput)
	}
	return parseOutput
}
