//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

// parseSpringBootHTML is the boot page for the spring-hook fixture. It mounts
// into <div id="app"> so the GWC component can attach there.
const parseSpringBootHTML = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>spring-fixture</title></head>
<body>
<div id="app"></div>
<script src="/wasm_exec.js"></script>
<script>
(async function() {
  const go = new Go();
  const result = await WebAssembly.instantiateStreaming(fetch("/fixture.wasm"), go.importObject);
  go.run(result.instance);
})();
</script>
</body>
</html>
`

// parseSpringFindWasmExecJS locates wasm_exec.js from the Go toolchain root.
func parseSpringFindWasmExecJS(parseT *testing.T) string {
	parseT.Helper()
	parseGoRoot, parseErr := exec.Command("go", "env", "GOROOT").Output()
	if parseErr != nil {
		parseT.Fatalf("go env GOROOT: %v", parseErr)
	}
	parseRoot := strings.TrimSpace(string(parseGoRoot))
	parseCandidates := []string{
		filepath.Join(parseRoot, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(parseRoot, "misc", "wasm", "wasm_exec.js"),
	}
	for _, parsePath := range parseCandidates {
		if _, parseStatErr := os.Stat(parsePath); parseStatErr == nil {
			return parsePath
		}
	}
	parseT.Fatalf("wasm_exec.js not found in GOROOT %s (tried %v)", parseRoot, parseCandidates)
	return ""
}

// parseSpringBuildWasm compiles the springfixture package to wasm and returns
// the path to the resulting .wasm file in a temp directory.
func parseSpringBuildWasm(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parseTmpDir := parseT.TempDir()
	parseOutWasm := filepath.Join(parseTmpDir, "fixture.wasm")

	parseCmd := exec.Command(
		"go", "build",
		"-o", parseOutWasm,
		"./test/playwrightgo/examples/testdata/springfixture",
	)
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")

	parseOut, parseBuildErr := parseCmd.CombinedOutput()
	if parseBuildErr != nil {
		parseT.Fatalf("build spring wasm fixture: %v\n%s", parseBuildErr, parseOut)
	}
	return parseOutWasm
}

// parseSpringServeFixture starts an httptest.Server that serves:
//
//	GET /            -> parseSpringBootHTML (mounts into <div id="app">)
//	GET /fixture.wasm -> the compiled wasm binary
//	GET /wasm_exec.js -> the Go runtime wasm glue
func parseSpringServeFixture(parseT *testing.T, parseWasmPath, parseWasmExecPath string) *httptest.Server {
	parseT.Helper()
	parseMux := http.NewServeMux()

	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Path != "/" {
			http.NotFound(parseW, parseR)
			return
		}
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(parseW, parseSpringBootHTML)
	})

	parseMux.HandleFunc("/fixture.wasm", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(parseW, parseR, parseWasmPath)
	})

	parseMux.HandleFunc("/wasm_exec.js", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(parseW, parseR, parseWasmExecPath)
	})

	parseSrv := httptest.NewServer(parseMux)
	parseT.Cleanup(parseSrv.Close)
	return parseSrv
}

// parseSpringReadValue reads the current integer text from #spring-value.
func parseSpringReadValue(parseT *testing.T, parsePage playwright.Page) int {
	parseT.Helper()
	parseRaw, parseErr := parsePage.Locator("#spring-value").TextContent(playwright.LocatorTextContentOptions{
		Timeout: playwright.Float(5_000),
	})
	if parseErr != nil {
		parseT.Fatalf("read #spring-value: %v", parseErr)
	}
	parseN, parseConvErr := strconv.Atoi(strings.TrimSpace(parseRaw))
	if parseConvErr != nil {
		parseT.Fatalf("parse #spring-value %q: %v", parseRaw, parseConvErr)
	}
	return parseN
}

// TestUseSpringAnimates builds the springfixture wasm, boots it in a headless
// Chromium page, and verifies that:
//
//  1. Before clicking #spring-go the value shows 0.
//  2. After clicking #spring-go the value PROGRESSES over time (read at two
//     points separated by a real delay and assert it increased), proving that
//     the spring animates frame-by-frame via requestAnimationFrame rather than
//     jumping straight to the target.
//  3. Eventually the value REACHES ~300 and #spring-done shows "settled".
//  4. No Uncaught/panic console errors occur throughout.
func TestUseSpringAnimates(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// --- 1. Build fixture wasm ---
	parseWasmPath := parseSpringBuildWasm(parseT, parseRepoRoot)

	// --- 2. Locate wasm_exec.js ---
	parseWasmExecPath := parseSpringFindWasmExecJS(parseT)

	// --- 3. Start fixture HTTP server (unique port 18199) ---
	parseSrv := parseSpringServeFixture(parseT, parseWasmPath, parseWasmExecPath)

	// --- 4. Install and launch headless Chromium ---
	if parseInstallErr := ensureExamplesChromiumInstalled(); parseInstallErr != nil {
		parseT.Fatalf("install chromium: %v", parseInstallErr)
	}
	parsePw, parsePwErr := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parsePwErr != nil {
		parseT.Fatalf("playwright.Run: %v", parsePwErr)
	}
	parseT.Cleanup(func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Logf("stop playwright: %v", parseStopErr)
		}
	})

	parseBrowserHandle, parseLaunchErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseLaunchErr != nil {
		parseT.Fatalf("launch chromium: %v", parseLaunchErr)
	}
	parseT.Cleanup(func() {
		if parseCloseErr := parseBrowserHandle.Close(); parseCloseErr != nil {
			parseT.Logf("close chromium: %v", parseCloseErr)
		}
	})

	parsePage, parsePageErr := parseBrowserHandle.NewPage()
	if parsePageErr != nil {
		parseT.Fatalf("new page: %v", parsePageErr)
	}

	// Collect console errors so we can assert no JS crashes occurred.
	parseConsoleErrors := parseCBCCollectErrors(parsePage)

	// --- 5. Navigate and wait for the component to mount ---
	if _, parseNavErr := parsePage.Goto(parseSrv.URL+"/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(90_000),
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to fixture: %v", parseNavErr)
	}

	// Wait for #spring-root to appear (component mounted).
	if _, parseWaitErr := parsePage.WaitForSelector("#spring-root", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(60_000),
	}); parseWaitErr != nil {
		parseT.Fatalf("fixture never rendered #spring-root: %v", parseWaitErr)
	}

	// --- 6. Assert initial value is 0 ---
	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#spring-value")?.textContent?.trim() === "0"`,
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(15_000)},
	); parseWFFErr != nil {
		parseInitialText, _ := parsePage.Locator("#spring-value").TextContent()
		parseT.Fatalf("initial: #spring-value = %q, want \"0\": %v", parseInitialText, parseWFFErr)
	}
	parseT.Log("initial value confirmed: 0")

	// Report any JS errors that occurred before and during boot.
	parseCBCCrashFree(parseT, *parseConsoleErrors)

	// --- 7. Click #spring-go to start the animation ---
	if parseClickErr := parsePage.Click("#spring-go"); parseClickErr != nil {
		parseT.Fatalf("click #spring-go: %v", parseClickErr)
	}
	parseT.Log("clicked #spring-go")

	// --- 8. Assert the value PROGRESSES (frame-by-frame animation check) ---
	// Read the value a short time after the click starts; it should be greater
	// than 0 but less than 300 — evidence that rAF is stepping the spring.
	parseT.Log("waiting for animation to begin progressing...")

	// Wait until the value leaves 0 (first frame fired).
	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => { const v = parseInt(document.querySelector("#spring-value")?.textContent || "0", 10); return v > 0; }`,
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(15_000), Polling: 16},
	); parseWFFErr != nil {
		parseFirstText, _ := parsePage.Locator("#spring-value").TextContent()
		parseT.Fatalf("animation never started: #spring-value still %q after click: %v", parseFirstText, parseWFFErr)
	}

	// Read the value at point A.
	parseValueA := parseSpringReadValue(parseT, parsePage)
	parseT.Logf("value at point A (just started): %d", parseValueA)

	// Wait a short real-time pause so multiple rAF frames can fire, then read B.
	// 150 ms gives roughly 9 frames at 60 fps — enough for the spring to advance
	// meaningfully without reaching the target on a wobbly spring.
	time.Sleep(150 * time.Millisecond)

	parseValueB := parseSpringReadValue(parseT, parsePage)
	parseT.Logf("value at point B (150 ms later): %d", parseValueB)

	// If B == A the spring did not advance at all between the two reads —
	// either the loop stopped prematurely or rAF is not cycling.
	// Allow the special case where A was already ~300 on a very fast machine;
	// in that case we skip the progress assertion and only verify settlement.
	if parseValueA < 295 && parseValueB <= parseValueA {
		parseT.Errorf("spring did not progress: valueA=%d valueB=%d — rAF loop may not be cycling", parseValueA, parseValueB)
	} else {
		parseT.Logf("progress confirmed: %d → %d", parseValueA, parseValueB)
	}

	// --- 9. Wait for the spring to settle near 300 ---
	parseT.Log("waiting for spring to settle near 300...")
	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#spring-done")?.textContent?.trim() === "settled"`,
		nil,
		playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(10_000),
			Polling: 50,
		},
	); parseWFFErr != nil {
		parseFinalValue, _ := parsePage.Locator("#spring-value").TextContent()
		parseDoneText, _ := parsePage.Locator("#spring-done").TextContent()
		parseT.Errorf("spring never settled: #spring-value=%q #spring-done=%q: %v",
			parseFinalValue, parseDoneText, parseWFFErr)
	}

	parseFinalValue := parseSpringReadValue(parseT, parsePage)
	parseT.Logf("spring settled at value: %d", parseFinalValue)

	if parseFinalValue < 295 || parseFinalValue > 305 {
		parseT.Errorf("settled value %d is not close to 300", parseFinalValue)
	}

	// --- 10. Final crash-free check ---
	parseCBCCrashFree(parseT, *parseConsoleErrors)

	parseT.Log("TestUseSpringAnimates: PASS — spring animated frame-by-frame and settled at target")
}
