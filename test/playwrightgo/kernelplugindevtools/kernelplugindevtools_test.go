//go:build playwrightgo
// +build playwrightgo

package kernelplugindevtools_test

import (
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

var storeKernelPluginDevtoolsBrowserState struct {
	getOnce sync.Once
	getErr  error
}

// ensureKernelPluginDevtoolsBrowserInstalled installs Chromium once for the focused browser test package.
func ensureKernelPluginDevtoolsBrowserInstalled() error {
	storeKernelPluginDevtoolsBrowserState.getOnce.Do(func() {
		storeKernelPluginDevtoolsBrowserState.getErr = playwright.Install(&playwright.RunOptions{
			Browsers: []string{"chromium"},
			Verbose:  false,
		})
	})
	return storeKernelPluginDevtoolsBrowserState.getErr
}

// kernelPluginDevtoolsRepoRootFromFile resolves the repo root from the current test file location.
func kernelPluginDevtoolsRepoRootFromFile(parseTestFile string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(parseTestFile), "..", "..", ".."))
}

// terminateKernelPluginDevtoolsProcessTree terminates one command and its descendants on Windows.
func terminateKernelPluginDevtoolsProcessTree(parseCommand *exec.Cmd) {
	if parseCommand == nil || parseCommand.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(parseCommand.Process.Pid)).Run()
		return
	}
	_ = parseCommand.Process.Kill()
}

// startKernelPluginDevtoolsCommand starts one background command from the provided working directory.
func startKernelPluginDevtoolsCommand(parseT *testing.T, parseDir string, parseName string, parseArgs ...string) func() {
	parseT.Helper()
	getCommand := exec.Command(parseName, parseArgs...)
	getCommand.Dir = parseDir
	getCommand.Stdout = io.Discard
	getCommand.Stderr = io.Discard
	if parseErr := getCommand.Start(); parseErr != nil {
		parseT.Fatalf("start %s %v: %v", parseName, parseArgs, parseErr)
	}
	getDone := make(chan error, 1)
	go func() {
		getDone <- getCommand.Wait()
	}()
	return func() {
		terminateKernelPluginDevtoolsProcessTree(getCommand)
		select {
		case <-getDone:
		case <-time.After(5 * time.Second):
		}
	}
}

// waitForKernelPluginDevtoolsHealthyURL waits for one HTTP endpoint to respond with a non-5xx status.
func waitForKernelPluginDevtoolsHealthyURL(parseT *testing.T, parseURL string, parseTimeout time.Duration) {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		parseResponse, parseErr := http.Get(parseURL)
		if parseErr == nil {
			_ = parseResponse.Body.Close()
			if parseResponse.StatusCode >= 200 && parseResponse.StatusCode < 500 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	parseT.Fatalf("health check timed out: %s", parseURL)
}

// getKernelPluginDevtoolsFreePort reserves one loopback TCP port number for the focused browser test server.
func getKernelPluginDevtoolsFreePort(parseT *testing.T) string {
	parseT.Helper()
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("listen for free port: %v", parseErr)
	}
	defer parseListener.Close()
	parseAddress, parseOk := parseListener.Addr().(*net.TCPAddr)
	if !parseOk {
		parseT.Fatalf("unexpected listener address type %T", parseListener.Addr())
	}
	return strconv.Itoa(parseAddress.Port)
}

// withKernelPluginDevtoolsPage opens one Chromium page and runs the provided callback.
func withKernelPluginDevtoolsPage(parseT *testing.T, parseFn func(playwright.Page)) {
	parseT.Helper()
	if parseErr := ensureKernelPluginDevtoolsBrowserInstalled(); parseErr != nil {
		parseT.Fatalf("install Chromium: %v", parseErr)
	}
	parsePlaywright, parseErr := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parseErr != nil {
		parseT.Fatalf("run Playwright: %v", parseErr)
	}
	defer func() {
		if parseStopErr := parsePlaywright.Stop(); parseStopErr != nil {
			parseT.Errorf("stop Playwright: %v", parseStopErr)
		}
	}()
	parseBrowser, parseErr := parsePlaywright.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseErr != nil {
		parseT.Fatalf("launch Chromium: %v", parseErr)
	}
	defer func() {
		if parseCloseErr := parseBrowser.Close(); parseCloseErr != nil {
			parseT.Errorf("close Chromium: %v", parseCloseErr)
		}
	}()
	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new browser page: %v", parseErr)
	}
	parseFn(parsePage)
}

// getKernelPluginDevtoolsStoredConsoleErrors returns the page-side captured console error payloads.
func getKernelPluginDevtoolsStoredConsoleErrors(parsePage playwright.Page) []string {
	if parsePage == nil {
		return nil
	}
	getValues, parseErr := parsePage.Evaluate(`() => window.__kernelPluginDevtoolsConsoleErrors || []`)
	if parseErr != nil {
		return nil
	}
	getInterfaces, hasInterfaces := getValues.([]interface{})
	if !hasInterfaces {
		return nil
	}
	buildErrors := make([]string, 0, len(getInterfaces))
	for _, parseValue := range getInterfaces {
		getText, hasText := parseValue.(string)
		if !hasText {
			continue
		}
		buildErrors = append(buildErrors, strings.TrimSpace(getText))
	}
	return buildErrors
}

// buildKernelPluginDevtoolsExampleWasm builds the example 111 wasm artifact into the shared examples static bin path.
func buildKernelPluginDevtoolsExampleWasm(parseT *testing.T, parseRepoRoot string) {
	parseT.Helper()
	stageKernelPluginDevtoolsWasmExecScript(parseT, parseRepoRoot)
	getCommand := exec.Command(
		"go",
		"run", "./tools/gwc", "build",
		"-app", "./examples/testing/kernel-plugin-devtools/main.go",
		"-root", "./examples/testing/kernel-plugin-devtools",
		"-out", "./examples/static/bin/kernel-plugin-devtools.wasm",
	)
	getCommand.Dir = parseRepoRoot
	if getOutput, parseErr := getCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build example 111 wasm: %v\n%s", parseErr, string(getOutput))
	}
}

// stageKernelPluginDevtoolsWasmExecScript copies Go's wasm_exec.js into the examples static script path for browser boot.
func stageKernelPluginDevtoolsWasmExecScript(parseT *testing.T, parseRepoRoot string) {
	parseT.Helper()
	getGOROOT := runtime.GOROOT()
	if strings.TrimSpace(getGOROOT) == "" {
		parseT.Fatal("runtime.GOROOT() returned an empty path")
	}
	getSourceCandidates := []string{
		filepath.Join(getGOROOT, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(getGOROOT, "misc", "wasm", "wasm_exec.js"),
	}
	getSourcePath := ""
	for _, parseSourceCandidate := range getSourceCandidates {
		if _, parseStatErr := os.Stat(parseSourceCandidate); parseStatErr == nil {
			getSourcePath = parseSourceCandidate
			break
		}
	}
	if getSourcePath == "" {
		parseT.Fatalf("could not find wasm_exec.js under GOROOT=%q", getGOROOT)
	}
	getSourceBytes, parseReadErr := os.ReadFile(getSourcePath)
	if parseReadErr != nil {
		parseT.Fatalf("read wasm_exec.js: %v", parseReadErr)
	}
	getTargetDir := filepath.Join(parseRepoRoot, "examples", "static", "script")
	if parseMkdirErr := os.MkdirAll(getTargetDir, 0o755); parseMkdirErr != nil {
		parseT.Fatalf("create examples/static/script: %v", parseMkdirErr)
	}
	getTargetPath := filepath.Join(getTargetDir, "wasm_exec.js")
	if parseWriteErr := os.WriteFile(getTargetPath, getSourceBytes, 0o644); parseWriteErr != nil {
		parseT.Fatalf("write staged wasm_exec.js: %v", parseWriteErr)
	}
}

// startKernelPluginDevtoolsServer starts a repo-root static server for the example route and shared assets.
func startKernelPluginDevtoolsServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseStop := startKernelPluginDevtoolsCommand(
		parseT,
		parseRepoRoot,
		"go",
		"run", "./tools/gwc", "serve",
		"-root", ".",
		"-host", "127.0.0.1",
		"-port", parsePort,
	)
	parseT.Cleanup(parseStop)
	parseBaseURL := "http://127.0.0.1:" + parsePort
	waitForKernelPluginDevtoolsHealthyURL(parseT, parseBaseURL+"/healthz", 30*time.Second)
	return parseBaseURL
}

// TestKernelPluginDevtoolsRetiredRouteIsUnavailable protects the intentional runtime2 retirement boundary.
func TestKernelPluginDevtoolsRetiredRouteIsUnavailable(parseT *testing.T) {
	// eeda966e removed this runtime2-backed plugin example in v5 P5.2.
	_, parseFile, _, _ := runtime.Caller(0)
	parseRoot := kernelPluginDevtoolsRepoRootFromFile(parseFile)
	if _, parseErr := os.Stat(filepath.Join(parseRoot, "examples", "testing", "kernel-plugin-devtools", "main.go")); !os.IsNotExist(parseErr) {
		parseT.Fatalf("retired kernel plugin example source unexpectedly present or unreadable: %v", parseErr)
	}
	parseBaseURL := startKernelPluginDevtoolsServer(parseT, parseRoot, getKernelPluginDevtoolsFreePort(parseT))
	withKernelPluginDevtoolsPage(parseT, func(parsePage playwright.Page) {
		parseResponse, parseErr := parsePage.Goto(parseBaseURL+"/examples/testing/kernel-plugin-devtools/kernel-plugin-devtools.html", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
		if parseErr != nil || parseResponse == nil || parseResponse.Status() != 404 {
			parseT.Fatalf("retired plugin route must explicitly return 404, got response=%v error=%v", parseResponse, parseErr)
		}
	})
}
