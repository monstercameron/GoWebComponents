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

// TestKernelPluginDevtoolsExampleBrowser verifies the kernel-backed devtools example through one focused browser flow.
func TestKernelPluginDevtoolsExampleBrowser(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := kernelPluginDevtoolsRepoRootFromFile(parseFile)
	buildKernelPluginDevtoolsExampleWasm(parseT, getRepoRoot)
	getBaseURL := startKernelPluginDevtoolsServer(parseT, getRepoRoot, getKernelPluginDevtoolsFreePort(parseT))

	withKernelPluginDevtoolsPage(parseT, func(parsePage playwright.Page) {
		getConsoleErrors := make([]string, 0, 8)
		getPageErrors := make([]string, 0, 8)
		if parseErr := parsePage.AddInitScript(playwright.Script{
			Content: playwright.String(`(() => {
				const storeKernelPluginDevtoolsErrors = [];
				const formatKernelPluginDevtoolsValue = (value) => {
					if (typeof value === "string") {
						return value;
					}
					try {
						return JSON.stringify(value);
					} catch (error) {
						return String(value);
					}
				};
				window.__kernelPluginDevtoolsConsoleErrors = storeKernelPluginDevtoolsErrors;
				const reportKernelPluginDevtoolsConsoleError = console.error.bind(console);
				console.error = (...args) => {
					storeKernelPluginDevtoolsErrors.push(args.map(formatKernelPluginDevtoolsValue).join(" | "));
					return reportKernelPluginDevtoolsConsoleError(...args);
				};
			})();`),
		}); parseErr != nil {
			parseT.Fatalf("add init script: %v", parseErr)
		}
		parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
			if parseMessage.Type() == "error" && len(getConsoleErrors) < 8 {
				getConsoleErrors = append(getConsoleErrors, strings.TrimSpace(parseMessage.Text()))
			}
		})
		parsePage.OnPageError(func(parseErr error) {
			if parseErr != nil && len(getPageErrors) < 8 {
				getPageErrors = append(getPageErrors, strings.TrimSpace(parseErr.Error()))
			}
		})
		if _, parseErr := parsePage.Goto(getBaseURL+"/examples/testing/kernel-plugin-devtools/kernel-plugin-devtools.html", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto example 111 route: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#kernel-plugin-root", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(15000),
		}); parseErr != nil {
			parseBodyText, _ := parsePage.TextContent("body")
			getStoredConsoleErrors := getKernelPluginDevtoolsStoredConsoleErrors(parsePage)
			parseT.Fatalf(
				"wait for example 111 root: %v body=%q console=%q page=%q",
				parseErr,
				parseBodyText,
				strings.Join(append(getConsoleErrors, getStoredConsoleErrors...), " || "),
				strings.Join(getPageErrors, " || "),
			)
		}
		if parseErr := parsePage.Click("#kernel-plugin-theme"); parseErr != nil {
			parseT.Fatalf("click theme toggle: %v", parseErr)
		}
		if parseErr := parsePage.Click("#kernel-plugin-increment"); parseErr != nil {
			parseT.Fatalf("click increment: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Theme: midnight", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseErr != nil {
			parseT.Fatalf("wait for runtime2 region theme update: %v", parseErr)
		}
		if parseErr := parsePage.Click("button:has-text(\"Kernel Plugin Devtools\")", playwright.PageClickOptions{
			Force:   playwright.Bool(true),
			Timeout: playwright.Float(10000),
		}); parseErr != nil {
			parseT.Fatalf("open devtools panel: %v", parseErr)
		}
		for _, parseValue := range getKernelPluginDevtoolsStoredConsoleErrors(parsePage) {
			if len(getConsoleErrors) >= 8 {
				break
			}
			getConsoleErrors = append(getConsoleErrors, parseValue)
		}
		if _, parseErr := parsePage.WaitForSelector("text=Example Kernel Plugin", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseErr != nil {
			parseBodyText, _ := parsePage.TextContent("body")
			getStoredConsoleErrors := getKernelPluginDevtoolsStoredConsoleErrors(parsePage)
			parseT.Fatalf(
				"wait for example kernel section: %v body=%q console=%q page=%q",
				parseErr,
				parseBodyText,
				strings.Join(append(getConsoleErrors, getStoredConsoleErrors...), " || "),
				strings.Join(getPageErrors, " || "),
			)
		}
		if _, parseWaitErr := parsePage.WaitForSelector("text=runtime2 regions: 1", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseWaitErr != nil {
			parseBodyText, _ := parsePage.TextContent("body")
			getStoredConsoleErrors := getKernelPluginDevtoolsStoredConsoleErrors(parsePage)
			parseT.Fatalf("wait for runtime2 region text: %v body=%q console=%q", parseWaitErr, parseBodyText, strings.Join(append(getConsoleErrors, getStoredConsoleErrors...), " || "))
		}
		if _, parseWaitErr := parsePage.WaitForSelector("text=events: 3", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseWaitErr != nil {
			parseBodyText, _ := parsePage.TextContent("body")
			getStoredConsoleErrors := getKernelPluginDevtoolsStoredConsoleErrors(parsePage)
			parseT.Fatalf("wait for refreshed kernel event count: %v body=%q console=%q", parseWaitErr, parseBodyText, strings.Join(append(getConsoleErrors, getStoredConsoleErrors...), " || "))
		}
		if _, parseWaitErr := parsePage.WaitForSelector("text=latest event: click on button", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseWaitErr != nil {
			parseBodyText, _ := parsePage.TextContent("body")
			getStoredConsoleErrors := getKernelPluginDevtoolsStoredConsoleErrors(parsePage)
			parseT.Fatalf("wait for refreshed kernel latest event: %v body=%q console=%q", parseWaitErr, parseBodyText, strings.Join(append(getConsoleErrors, getStoredConsoleErrors...), " || "))
		}
		if _, parseWaitErr := parsePage.WaitForSelector("text=style variables:", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseWaitErr != nil {
			parseBodyText, _ := parsePage.TextContent("body")
			getStoredConsoleErrors := getKernelPluginDevtoolsStoredConsoleErrors(parsePage)
			parseT.Fatalf("wait for kernel plugin diagnostics: %v body=%q console=%q", parseWaitErr, parseBodyText, strings.Join(append(getConsoleErrors, getStoredConsoleErrors...), " || "))
		}
		parseBodyText, parseBodyErr := parsePage.TextContent("body")
		if parseBodyErr != nil {
			parseT.Fatalf("read body text: %v", parseBodyErr)
		}
		getBodyText := strings.ToLower(parseBodyText)
		for _, parseNeedle := range []string{
			"example kernel plugin",
			"theme: midnight",
			"runtime2 regions: 1",
			"latest event: click on button",
			"events: 3",
			"style variables:",
		} {
			if !strings.Contains(getBodyText, strings.ToLower(parseNeedle)) {
				parseT.Fatalf("expected body text to contain %q, got %q", parseNeedle, parseBodyText)
			}
		}
		if len(getConsoleErrors) > 0 || len(getPageErrors) > 0 {
			parseT.Fatalf("unexpected browser errors: console=%q page=%q", strings.Join(getConsoleErrors, " || "), strings.Join(getPageErrors, " || "))
		}
	})
}
