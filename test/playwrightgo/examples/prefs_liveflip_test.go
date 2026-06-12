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
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// parsePrefsBootHTML is the boot page for the preference-hooks fixture.
// It mounts into <div id="app"> so the GWC component can attach there.
const parsePrefsBootHTML = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>prefs-fixture</title></head>
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

// parsePrefsFindWasmExecJS locates wasm_exec.js from the Go toolchain root,
// checking both the modern path (lib/wasm/) and the legacy path (misc/wasm/).
func parsePrefsFindWasmExecJS(parseT *testing.T) string {
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

// parsePrefsBuildWasm compiles the prefsfixture package to wasm and returns the
// path to the resulting .wasm file in a temp directory.
func parsePrefsBuildWasm(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parseTmpDir := parseT.TempDir()
	parseOutWasm := filepath.Join(parseTmpDir, "fixture.wasm")

	parseCmd := exec.Command(
		"go", "build",
		"-o", parseOutWasm,
		"./test/playwrightgo/examples/testdata/prefsfixture",
	)
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")

	parseOut, parseBuildErr := parseCmd.CombinedOutput()
	if parseBuildErr != nil {
		parseT.Fatalf("build prefs wasm fixture: %v\n%s", parseBuildErr, parseOut)
	}
	return parseOutWasm
}

// parsePrefsServeFixture starts an httptest.Server that serves:
//
//	GET /           -> parsePrefsBootHTML (mounts into <div id="app">)
//	GET /fixture.wasm -> the compiled wasm binary
//	GET /wasm_exec.js -> the Go runtime wasm glue
func parsePrefsServeFixture(parseT *testing.T, parseWasmPath, parseWasmExecPath string) *httptest.Server {
	parseT.Helper()
	parseMux := http.NewServeMux()

	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Path != "/" {
			http.NotFound(parseW, parseR)
			return
		}
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(parseW, parsePrefsBootHTML)
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

// TestPreferenceHooksLiveFlip builds the prefsfixture wasm, boots it in a
// headless Chromium page, and verifies that:
//
//  1. At mount with dark+reduce emulation the component renders #scheme="dark"
//     and #motion="true".
//  2. After a live EmulateMedia flip to light+no-preference the component
//     re-renders (via the hook's MediaQueryList subscription) to #scheme="light"
//     and #motion="false" — WITHOUT a page reload.
//
// The live-flip assertion is the core proof: the GWC preference hooks subscribe
// to OS/emulated media changes and trigger a re-render automatically.
func TestPreferenceHooksLiveFlip(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// --- 1. Build fixture wasm ---
	parseWasmPath := parsePrefsBuildWasm(parseT, parseRepoRoot)

	// --- 2. Locate wasm_exec.js ---
	parseWasmExecPath := parsePrefsFindWasmExecJS(parseT)

	// --- 3. Start fixture HTTP server ---
	parseSrv := parsePrefsServeFixture(parseT, parseWasmPath, parseWasmExecPath)

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

	// --- 5. Emulate dark + reduce BEFORE loading the page so the initial
	//        media environment matches what the hooks read at mount time. ---
	parsePage, parsePageErr := parseBrowserHandle.NewPage()
	if parsePageErr != nil {
		parseT.Fatalf("new page: %v", parsePageErr)
	}

	if parseEmuErr := parsePage.EmulateMedia(playwright.PageEmulateMediaOptions{
		ColorScheme:   playwright.ColorSchemeDark,
		ReducedMotion: playwright.ReducedMotionReduce,
	}); parseEmuErr != nil {
		parseT.Fatalf("EmulateMedia dark+reduce: %v", parseEmuErr)
	}

	// Collect console errors so we can report JS crashes.
	parseConsoleErrors := parseCBCCollectErrors(parsePage)

	// --- 6. Navigate and wait for the component to mount ---
	if _, parseNavErr := parsePage.Goto(parseSrv.URL+"/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(90_000),
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to fixture: %v", parseNavErr)
	}

	// Wait for the #prefs container (rendered by the GWC component).
	if _, parseWaitErr := parsePage.WaitForSelector("#prefs", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(60_000),
	}); parseWaitErr != nil {
		parseT.Fatalf("fixture never rendered #prefs: %v", parseWaitErr)
	}

	// --- 7. Assert mount values: dark + true ---
	parseT.Log("asserting mount values with dark+reduce emulation")

	if parseWaitErr := parsePage.Locator("#scheme").WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(15_000),
	}); parseWaitErr != nil {
		parseT.Fatalf("wait for #scheme visible: %v", parseWaitErr)
	}

	// Use WaitForFunction to poll until the expected text appears, avoiding
	// timing races between the wasm hook's UseEffect and our assertion.
	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#scheme")?.textContent === "dark"`,
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(15_000)},
	); parseWFFErr != nil {
		parseSchemeText, _ := parsePage.Locator("#scheme").TextContent()
		parseT.Fatalf("mount: #scheme = %q, want \"dark\": %v", parseSchemeText, parseWFFErr)
	}

	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#motion")?.textContent === "true"`,
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(15_000)},
	); parseWFFErr != nil {
		parseMotionText, _ := parsePage.Locator("#motion").TextContent()
		parseT.Fatalf("mount: #motion = %q, want \"true\": %v", parseMotionText, parseWFFErr)
	}

	parseMountScheme, _ := parsePage.Locator("#scheme").TextContent()
	parseMountMotion, _ := parsePage.Locator("#motion").TextContent()
	parseT.Logf("mount values confirmed: #scheme=%q #motion=%q", parseMountScheme, parseMountMotion)

	// Report any JS errors that occurred before and during boot.
	parseCBCCrashFree(parseT, *parseConsoleErrors)

	// --- 8. LIVE FLIP: switch to light + no-preference (no reload) ---
	parseT.Log("flipping EmulateMedia to light+no-preference (live update test)")

	if parseEmuErr := parsePage.EmulateMedia(playwright.PageEmulateMediaOptions{
		ColorScheme:   playwright.ColorSchemeLight,
		ReducedMotion: playwright.ReducedMotionNoPreference,
	}); parseEmuErr != nil {
		parseT.Fatalf("EmulateMedia light+no-preference: %v", parseEmuErr)
	}

	// Poll until the component re-renders with the new values. The hook's
	// Subscribe callback fires on the MediaQueryList change event and calls
	// parseState.Set, which triggers a GWC re-render.
	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#scheme")?.textContent === "light"`,
		nil,
		playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(15_000),
			Polling: 50,
		},
	); parseWFFErr != nil {
		parseSchemeText, _ := parsePage.Locator("#scheme").TextContent()
		parseT.Errorf("live-flip: #scheme = %q, want \"light\" — hook subscription may not be firing: %v", parseSchemeText, parseWFFErr)
	}

	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#motion")?.textContent === "false"`,
		nil,
		playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(15_000),
			Polling: 50,
		},
	); parseWFFErr != nil {
		parseMotionText, _ := parsePage.Locator("#motion").TextContent()
		parseT.Errorf("live-flip: #motion = %q, want \"false\" — hook subscription may not be firing: %v", parseMotionText, parseWFFErr)
	}

	parseFlipScheme, _ := parsePage.Locator("#scheme").TextContent()
	parseFlipMotion, _ := parsePage.Locator("#motion").TextContent()
	parseT.Logf("live-flip values confirmed: #scheme=%q #motion=%q", parseFlipScheme, parseFlipMotion)

	// --- 9. Verify subscriptions stay live: flip back to dark+reduce ---
	parseT.Log("flipping back to dark+reduce to confirm subscription stays live")

	if parseEmuErr := parsePage.EmulateMedia(playwright.PageEmulateMediaOptions{
		ColorScheme:   playwright.ColorSchemeDark,
		ReducedMotion: playwright.ReducedMotionReduce,
	}); parseEmuErr != nil {
		parseT.Fatalf("EmulateMedia dark+reduce (second flip): %v", parseEmuErr)
	}

	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#scheme")?.textContent === "dark"`,
		nil,
		playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(15_000),
			Polling: 50,
		},
	); parseWFFErr != nil {
		parseSchemeText, _ := parsePage.Locator("#scheme").TextContent()
		parseT.Errorf("second-flip: #scheme = %q, want \"dark\": %v", parseSchemeText, parseWFFErr)
	}

	if _, parseWFFErr := parsePage.WaitForFunction(
		`() => document.querySelector("#motion")?.textContent === "true"`,
		nil,
		playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(15_000),
			Polling: 50,
		},
	); parseWFFErr != nil {
		parseMotionText, _ := parsePage.Locator("#motion").TextContent()
		parseT.Errorf("second-flip: #motion = %q, want \"true\": %v", parseMotionText, parseWFFErr)
	}

	parseSecondScheme, _ := parsePage.Locator("#scheme").TextContent()
	parseSecondMotion, _ := parsePage.Locator("#motion").TextContent()
	parseT.Logf("second-flip values confirmed: #scheme=%q #motion=%q", parseSecondScheme, parseSecondMotion)

	// Final crash-free check after all the live flips.
	parseCBCCrashFree(parseT, *parseConsoleErrors)
}
