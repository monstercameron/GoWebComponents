//go:build playwrightgo

package playwrightgoexamples_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// M10 — time to first worker command.
//
// The packaging test records the size half and states the timing half as open,
// because it is a browser measurement: how long after navigation the app can
// actually get an answer out of services.wasm. Size alone cannot say, since a
// small worker that instantiates slowly and a large one that instantiates
// quickly land in different places.
//
// Measured to the command COMPLETING, not to the worker existing. A worker that
// has booted but cannot yet answer has not moved the app closer to being usable,
// and measuring to "ready" would report the flattering half of the number.
//
// The path exercised is the whole one: page loads app.wasm, app.wasm starts the
// worker, worker.js fetches and instantiates services.wasm, services installs a
// message handler and announces itself, the app posts a command, the worker runs
// it through the domain runtime against SQLite, and the reply is applied on the
// render thread through the async inbox.

// buildTwoArtifactPair compiles both halves into the example directory.
func buildTwoArtifactPair(parseT *testing.T, parseRepoRoot string, parseDir string) {
	parseT.Helper()
	for _, parseTarget := range []struct {
		Package string
		Output  string
	}{
		{"./examples/v5-two-artifact/app", "app.wasm"},
		{"./examples/v5-two-artifact/services", "services.wasm"},
	} {
		parseCmd := exec.Command("go", "build", "-o", filepath.Join(parseDir, parseTarget.Output), parseTarget.Package)
		parseCmd.Dir = parseRepoRoot
		parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		if parseOut, parseErr := parseCmd.CombinedOutput(); parseErr != nil {
			parseT.Fatalf("build %s: %v\n%s", parseTarget.Package, parseErr, parseOut)
		}
	}
}

// TestV5TwoArtifactTimeToFirstCommand closes M10's timing half.
func TestV5TwoArtifactTimeToFirstCommand(parseT *testing.T) {
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", "..", ".."))
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseDir := filepath.Join(parseRepoRoot, "examples", "v5-two-artifact")
	buildTwoArtifactPair(parseT, parseRepoRoot, parseDir)

	parseShim := filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js")
	if _, parseStatErr := os.Stat(parseShim); parseStatErr != nil {
		parseShim = filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js")
	}
	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/wasm_exec.js", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "text/javascript")
		http.ServeFile(parseW, parseR, parseShim)
	})
	parseMux.Handle("/", http.FileServer(http.Dir(parseDir)))
	parseServer := httptest.NewServer(parseMux)
	defer parseServer.Close()

	parsePW, parseErr := playwright.Run()
	if parseErr != nil {
		parseT.Skipf("playwright unavailable: %v", parseErr)
	}
	defer parsePW.Stop()
	parseBrowser, parseErr := parsePW.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer parseBrowser.Close()
	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new page: %v", parseErr)
	}

	// Console and page errors are captured BEFORE navigating. A wasm app that
	// fails to boot produces a blank page and a timeout, and without these the
	// only evidence is "waited 60 seconds" — which names no cause.
	parseConsole := []string{}
	parsePage.On("console", func(parseMessage playwright.ConsoleMessage) {
		parseConsole = append(parseConsole, parseMessage.Type()+": "+parseMessage.Text())
	})
	parsePage.On("pageerror", func(parseErr error) {
		parseConsole = append(parseConsole, "pageerror: "+parseErr.Error())
	})
	parsePage.On("requestfailed", func(parseRequest playwright.Request) {
		parseConsole = append(parseConsole, "requestfailed: "+parseRequest.URL())
	})

	if _, parseErr := parsePage.Goto(parseServer.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateCommit, Timeout: playwright.Float(60000),
	}); parseErr != nil {
		parseT.Fatalf("goto: %v", parseErr)
	}

	// Either outcome ends the wait. Waiting only for success turns a failed
	// command into a 60-second timeout with nothing to read.
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.__v5FirstCommandMs !== undefined || window.__v5FirstCommandError !== undefined`,
		nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(60000)}); parseErr != nil {
		for _, parseLine := range parseConsole {
			parseT.Logf("  console | %s", parseLine)
		}
		parseT.Fatalf("wait for first command: %v", parseErr)
	}

	if parseFailure, parseErr := parsePage.Evaluate(`() => window.__v5FirstCommandError ?? ""`); parseErr == nil {
		if parseText, parseOk := parseFailure.(string); parseOk && parseText != "" {
			parseT.Fatalf("the first command failed, so M10 has no timing to report: %s", parseText)
		}
	}

	parseRaw, parseErr := parsePage.Evaluate(`() => window.__v5FirstCommandMs`)
	if parseErr != nil {
		parseT.Fatalf("read timing: %v", parseErr)
	}
	parseMs, parseOk := parseRaw.(float64)
	if !parseOk {
		parseT.Fatalf("timing was %T, want a number", parseRaw)
	}

	parseAppSize, parseServicesSize := 0.0, 0.0
	if parseInfo, parseErr := os.Stat(filepath.Join(parseDir, "app.wasm")); parseErr == nil {
		parseAppSize = float64(parseInfo.Size()) / (1 << 20)
	}
	if parseInfo, parseErr := os.Stat(filepath.Join(parseDir, "services.wasm")); parseErr == nil {
		parseServicesSize = float64(parseInfo.Size()) / (1 << 20)
	}

	parseT.Logf("M10 time to first worker command: %.0f ms (target 400 ms)", parseMs)
	parseT.Logf("  app.wasm %.2f MB uncompressed, services.wasm %.2f MB uncompressed",
		parseAppSize, parseServicesSize)
	parseT.Logf("  covers: page load, app instantiate, worker spawn, services instantiate, " +
		"SQLite open, command round trip, and the reply applied through the inbox")

	// Reported, not gated, and deliberately so on a first measurement: this is a
	// developer build over a local server on one machine, and turning a single
	// uncalibrated reading into a merge gate is how a metric starts failing for
	// reasons unrelated to the code.
	if parseMs > 400 {
		parseT.Logf("  OVER the 400 ms target by %.0f ms — M10 timing is MISSED", parseMs-400)
	} else {
		parseT.Logf("  within the 400 ms target")
	}
}
