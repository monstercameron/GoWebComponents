//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"io"
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

	playwright "github.com/playwright-community/playwright-go"
)

var installChromiumOnce sync.Once
var installChromiumErr error

func ensureChromiumInstalled() error {
	installChromiumOnce.Do(func() {
		installChromiumErr = playwright.Install(&playwright.RunOptions{
			Browsers: []string{"chromium"},
			Verbose:  false,
		})
	})
	return installChromiumErr
}

func repoRootFromFile(parseTestFile string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(parseTestFile), "..", ".."))
}

// terminateSuiteProcessTree terminates a command process and, on Windows, its descendant processes.
func terminateSuiteProcessTree(parseCmd *exec.Cmd) {
	if parseCmd == nil || parseCmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(parseCmd.Process.Pid)).Run()
		return
	}
	_ = parseCmd.Process.Kill()
}

func startCommand(parseT *testing.T, parseDir string, parseName string, parseArgs ...string) (parseStop func()) {
	parseT.Helper()
	parseCmd := exec.Command(parseName, parseArgs...)
	parseCmd.Dir = parseDir
	parseCmd.Stdout = io.Discard
	parseCmd.Stderr = io.Discard
	if parseErr := parseCmd.Start(); parseErr != nil {
		parseT.Fatalf("start %s %v: %v", parseName, parseArgs, parseErr)
	}

	parseDone := make(chan error, 1)
	go func() {
		parseDone <- parseCmd.Wait()
	}()

	return func() {
		terminateSuiteProcessTree(parseCmd)
		select {
		case <-parseDone:
		case <-time.After(5 * time.Second):
		}
	}
}

func waitForHealthyURL(parseT *testing.T, parseHealthURL string, parseTimeout time.Duration) {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		parseResp, parseErr := http.Get(parseHealthURL)
		if parseErr == nil {
			_ = parseResp.Body.Close()
			if parseResp.StatusCode >= 200 && parseResp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	parseT.Fatalf("health check timed out: %s", parseHealthURL)
}

func buildTestAppWasm(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parseOutputDir := filepath.Join(parseRepoRoot, "bin", "test", "testapp")
	if parseErr := os.MkdirAll(parseOutputDir, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir output dir: %v", parseErr)
	}

	parseOutputWasm := filepath.Join(parseOutputDir, "main.wasm")
	parseCmd := exec.Command("go", "build", "-o", parseOutputWasm, ".")
	parseCmd.Dir = filepath.Join(parseRepoRoot, "test", "testapp")
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOut, parseErr2 := parseCmd.CombinedOutput(); parseErr2 != nil {
		parseT.Fatalf("build testapp wasm: %v\n%s", parseErr2, string(parseOut))
	}
	return parseOutputWasm
}

func runChromiumPage(parseT *testing.T, parseFn func(page playwright.Page)) {
	parseT.Helper()
	if parseErr := ensureChromiumInstalled(); parseErr != nil {
		parseT.Fatalf("install playwright chromium: %v", parseErr)
	}

	parsePw, parseErr2 := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parseErr2 != nil {
		parseT.Fatalf("run playwright-go: %v", parseErr2)
	}
	defer func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright-go: %v", parseStopErr)
		}
	}()

	parseBrowser, parseErr2 := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseErr2 != nil {
		parseT.Fatalf("launch chromium: %v", parseErr2)
	}
	defer func() {
		if parseCloseErr := parseBrowser.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	parsePage, parseErr2 := parseBrowser.NewPage()
	if parseErr2 != nil {
		parseT.Fatalf("new page: %v", parseErr2)
	}
	parseFn(parsePage)
}

func startTestAppServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseWasmPath := buildTestAppWasm(parseT, parseRepoRoot)
	parseFixturePath := filepath.Join(parseRepoRoot, "test", "fixtures", "user-123.json")
	parseStop := startCommand(
		parseT,
		parseRepoRoot,
		"go",
		"run", "./tools/gwc", "serve",
		"-root", "./test/testapp",
		"-host", "127.0.0.1",
		"-port", parsePort,
		"-wasm-route", "/main.wasm",
		"-wasm-file", parseWasmPath,
		"-fixture-json", "/api/user/123="+parseFixturePath,
	)
	parseT.Cleanup(parseStop)

	parseBaseURL := "http://127.0.0.1:" + parsePort
	waitForHealthyURL(parseT, parseBaseURL+"/healthz", 30*time.Second)
	return parseBaseURL
}

func TestComponents(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := repoRootFromFile(parseFile)
	parseBaseURL := startTestAppServer(parseT, parseRepoRoot, "18083")

	runChromiumPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(parseBaseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto test app: %v", parseErr)
		}
		if _, parseErr2 := parsePage.WaitForSelector("#main-heading"); parseErr2 != nil {
			parseT.Fatalf("wait for main heading: %v", parseErr2)
		}
		if _, parseErr3 := parsePage.WaitForSelector("button[aria-label=\"Increment main\"]"); parseErr3 != nil {
			parseT.Fatalf("wait for increment button: %v", parseErr3)
		}
		parseHeading, parseErr4 := parsePage.TextContent("#main-heading")
		if parseErr4 != nil {
			parseT.Fatalf("read heading: %v", parseErr4)
		}
		if !strings.Contains(parseHeading, "GoWebComponents Test") {
			parseT.Fatalf("unexpected heading: %q", parseHeading)
		}

		if parseErr5 := parsePage.Click("button[aria-label=\"Increment main\"]"); parseErr5 != nil {
			parseT.Fatalf("click increment main: %v", parseErr5)
		}
		if _, parseErr6 := parsePage.WaitForFunction(
			"() => document.querySelector('[data-testid=\"count-display\"]')?.textContent?.includes('Count: 1')",
			nil,
		); parseErr6 != nil {
			parseT.Fatalf("wait for count display update: %v", parseErr6)
		}
		parseCountDisplay, parseErr4 := parsePage.TextContent("[data-testid=\"count-display\"]")
		if parseErr4 != nil {
			parseT.Fatalf("read count display: %v", parseErr4)
		}
		if !strings.Contains(parseCountDisplay, "Count: 1") {
			parseT.Fatalf("unexpected count display: %q", parseCountDisplay)
		}
	})
}

func TestIntegration(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := repoRootFromFile(parseFile)
	parseBaseURL := startTestAppServer(parseT, parseRepoRoot, "18084")

	runChromiumPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(parseBaseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto test app: %v", parseErr)
		}
		if _, parseErr2 := parsePage.WaitForSelector("#todo-input"); parseErr2 != nil {
			parseT.Fatalf("wait for todo input: %v", parseErr2)
		}

		if parseErr3 := parsePage.Fill("#todo-input", "go-playwright todo"); parseErr3 != nil {
			parseT.Fatalf("fill todo input: %v", parseErr3)
		}
		if parseErr4 := parsePage.Click("#todo-add"); parseErr4 != nil {
			parseT.Fatalf("click todo add: %v", parseErr4)
		}

		parseTodoText, parseErr5 := parsePage.TextContent("#todo-text-1")
		if parseErr5 != nil {
			parseT.Fatalf("read first todo: %v", parseErr5)
		}
		if !strings.Contains(parseTodoText, "go-playwright todo") {
			parseT.Fatalf("unexpected todo text: %q", parseTodoText)
		}

		if parseErr6 := parsePage.Click("button:has-text(\"Toggle Theme\")"); parseErr6 != nil {
			parseT.Fatalf("click toggle theme: %v", parseErr6)
		}
		parseTheme, parseErr5 := parsePage.GetAttribute("body", "data-theme")
		if parseErr5 != nil {
			parseT.Fatalf("read body theme attr: %v", parseErr5)
		}
		if parseTheme == "" {
			parseT.Fatalf("expected non-empty body theme attr after toggle")
		}
	})
}

func TestState(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := repoRootFromFile(parseFile)
	parseBaseURL := startTestAppServer(parseT, parseRepoRoot, "18085")

	runChromiumPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(parseBaseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto test app: %v", parseErr)
		}
		if _, parseErr2 := parsePage.WaitForSelector("#stress-plus-5"); parseErr2 != nil {
			parseT.Fatalf("wait for stress controls: %v", parseErr2)
		}
		if parseErr3 := parsePage.Click("#stress-plus-5"); parseErr3 != nil {
			parseT.Fatalf("click stress plus 5: %v", parseErr3)
		}
		if parseErr4 := parsePage.Click("#stress-plus-25"); parseErr4 != nil {
			parseT.Fatalf("click stress plus 25: %v", parseErr4)
		}

		parseStressCount, parseErr5 := parsePage.TextContent("#stress-count")
		if parseErr5 != nil {
			parseT.Fatalf("read stress count: %v", parseErr5)
		}
		if !strings.Contains(parseStressCount, "Stress Count: 5") && !strings.Contains(parseStressCount, "Stress Count: 30") {
			parseT.Fatalf("unexpected stress count: %q", parseStressCount)
		}
	})
}

func buildBenchmarkWasm(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parseOutputDir := filepath.Join(parseRepoRoot, "bin", "test", "benchmark")
	if parseErr := os.MkdirAll(parseOutputDir, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir benchmark output dir: %v", parseErr)
	}

	parseOutputWasm := filepath.Join(parseOutputDir, "benchmark.wasm")
	parseCmd := exec.Command("go", "build", "-o", parseOutputWasm, ".")
	parseCmd.Dir = filepath.Join(parseRepoRoot, "test", "benchmark")
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOut, parseErr2 := parseCmd.CombinedOutput(); parseErr2 != nil {
		parseT.Fatalf("build benchmark wasm: %v\n%s", parseErr2, string(parseOut))
	}
	return parseOutputWasm
}

func TestBenchmark(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := repoRootFromFile(parseFile)
	parseWasmPath := buildBenchmarkWasm(parseT, parseRepoRoot)
	parseStop := startCommand(
		parseT,
		parseRepoRoot,
		"go",
		"run", "./tools/gwc", "serve",
		"-root", "./test/benchmark",
		"-host", "127.0.0.1",
		"-port", "18082",
		"-wasm-route", "/bin/benchmark.wasm",
		"-wasm-file", parseWasmPath,
	)
	parseT.Cleanup(parseStop)
	parseBaseURL := "http://127.0.0.1:18082"
	waitForHealthyURL(parseT, parseBaseURL+"/healthz", 30*time.Second)

	runChromiumPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(parseBaseURL+"/benchmark.html", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto benchmark page: %v", parseErr)
		}
		if _, parseErr2 := parsePage.WaitForSelector("#btn-render"); parseErr2 != nil {
			parseT.Fatalf("wait for render button: %v", parseErr2)
		}
		if parseErr3 := parsePage.Click("#btn-render"); parseErr3 != nil {
			parseT.Fatalf("click render button: %v", parseErr3)
		}
		if _, parseErr4 := parsePage.WaitForSelector(".core-list-item"); parseErr4 != nil {
			parseT.Fatalf("wait for core list item: %v", parseErr4)
		}
		parseCoreCount, parseErr5 := parsePage.TextContent("#core-count")
		if parseErr5 != nil {
			parseT.Fatalf("read core count: %v", parseErr5)
		}
		if !strings.Contains(parseCoreCount, "Core Count: 40") {
			parseT.Fatalf("unexpected core count text: %q", parseCoreCount)
		}
	})
}

func TestMainSuite(parseT *testing.T) {
	for _, parseName := range []string{"components", "integration", "state"} {
		parseName2 := parseName
		parseT.Run(parseName2, func(parseT2 *testing.T) {
			switch parseName2 {
			case "components":
				TestComponents(parseT2)
			case "integration":
				TestIntegration(parseT2)
			case "state":
				TestState(parseT2)
			default:
				parseT2.Fatalf("unknown suite: %s", parseName2)
			}
		})
	}
}
