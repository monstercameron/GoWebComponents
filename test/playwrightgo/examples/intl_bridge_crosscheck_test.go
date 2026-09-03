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

	playwright "github.com/mxschmitt/playwright-go"
)

// parseIntlBootHTML is the standard Go wasm boot page. It loads wasm_exec.js
// then instantiates and runs the compiled wasm binary.
const parseIntlBootHTML = `<!DOCTYPE html>
<html>
<head><meta charset="utf-8"><title>intl-fixture</title></head>
<body>
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

// parseIntlFindWasmExecJS locates wasm_exec.js from the Go toolchain root.
// It checks both the modern path (lib/wasm/wasm_exec.js) and the legacy path
// (misc/wasm/wasm_exec.js) so the test works across Go versions.
func parseIntlFindWasmExecJS(parseT *testing.T) string {
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

// parseIntlBuildWasm compiles the intlfixture package to wasm and returns the
// path to the resulting .wasm file in a temp directory.
func parseIntlBuildWasm(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parseTmpDir := parseT.TempDir()
	parseOutWasm := filepath.Join(parseTmpDir, "fixture.wasm")

	parseCmd := exec.Command(
		"go", "build",
		"-o", parseOutWasm,
		"./test/playwrightgo/examples/testdata/intlfixture",
	)
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")

	parseOut, parseBuildErr := parseCmd.CombinedOutput()
	if parseBuildErr != nil {
		parseT.Fatalf("build wasm fixture: %v\n%s", parseBuildErr, parseOut)
	}
	return parseOutWasm
}

// parseIntlServeFixture starts an httptest.Server that serves:
//
//	GET /           -> parseIntlBootHTML
//	GET /fixture.wasm -> the compiled wasm binary
//	GET /wasm_exec.js -> the Go runtime wasm glue
//
// The server is registered with t.Cleanup and must not be used after the test
// returns.
func parseIntlServeFixture(parseT *testing.T, parseWasmPath, parseWasmExecPath string) *httptest.Server {
	parseT.Helper()
	parseMux := http.NewServeMux()

	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR.URL.Path != "/" {
			http.NotFound(parseW, parseR)
			return
		}
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(parseW, parseIntlBootHTML)
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

// TestIntlBridgeMatchesBrowserIntl builds the intl wasm fixture, boots it in a
// headless Chromium page, and for every rendered .intl-case div asserts that
// the GWC result (data-gwc) equals what the browser's Intl.NumberFormat
// produces for the same locale, value, and options.
//
// This is a black-box cross-check: it tests the full GWC → JS Intl round-trip
// without mocking anything; any real mismatch surfaces as a test failure and
// constitutes a genuine bridge bug.
func TestIntlBridgeMatchesBrowserIntl(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// --- 1. Build fixture wasm ---
	parseWasmPath := parseIntlBuildWasm(parseT, parseRepoRoot)

	// --- 2. Locate wasm_exec.js ---
	parseWasmExecPath := parseIntlFindWasmExecJS(parseT)

	// --- 3. Start fixture HTTP server ---
	parseSrv := parseIntlServeFixture(parseT, parseWasmPath, parseWasmExecPath)

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

	// Collect console errors so we can report JS crashes.
	parseConsoleErrors := parseCBCCollectErrors(parsePage)

	// --- 5. Navigate and wait for the sentinel ---
	if _, parseNavErr := parsePage.Goto(parseSrv.URL+"/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(90_000),
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to fixture: %v", parseNavErr)
	}

	if _, parseWaitErr := parsePage.WaitForSelector("#intl-done", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(60_000),
	}); parseWaitErr != nil {
		parseT.Fatalf("fixture never reached #intl-done sentinel: %v", parseWaitErr)
	}

	// Report any JS errors that occurred before and during boot.
	parseCBCCrashFree(parseT, *parseConsoleErrors)

	// --- 6. Assert data-intl-available is "true" ---
	parseIntlAvailRaw, parseAttrErr := parsePage.Evaluate(`() => {
		const parseEl = document.querySelector(".intl-case");
		return parseEl ? parseEl.getAttribute("data-intl-available") : "missing";
	}`, nil)
	if parseAttrErr != nil {
		parseT.Fatalf("read data-intl-available: %v", parseAttrErr)
	}
	parseIntlAvailStr, _ := parseIntlAvailRaw.(string)
	if parseIntlAvailStr != "true" {
		parseT.Errorf("data-intl-available = %q, want \"true\"", parseIntlAvailStr)
	}

	// --- 7. Collect all .intl-case divs and cross-check ---
	parseCaseCount, parseCountErr := parsePage.Locator(".intl-case").Count()
	if parseCountErr != nil {
		parseT.Fatalf("count .intl-case: %v", parseCountErr)
	}
	if parseCaseCount == 0 {
		parseT.Fatal("no .intl-case divs found — fixture did not render any cases")
	}
	parseT.Logf("found %d .intl-case divs", parseCaseCount)

	for parseI := 0; parseI < parseCaseCount; parseI++ {
		parseI := parseI
		parseLocator := parsePage.Locator(".intl-case").Nth(parseI)

		parseLocale, parseLocaleErr := parseLocator.GetAttribute("data-locale")
		if parseLocaleErr != nil {
			parseT.Errorf("case %d: read data-locale: %v", parseI, parseLocaleErr)
			continue
		}
		parseValueStr, parseValueErr := parseLocator.GetAttribute("data-value")
		if parseValueErr != nil {
			parseT.Errorf("case %d: read data-value: %v", parseI, parseValueErr)
			continue
		}
		parseOptsJSON, parseOptsErr := parseLocator.GetAttribute("data-opts")
		if parseOptsErr != nil {
			parseT.Errorf("case %d: read data-opts: %v", parseI, parseOptsErr)
			continue
		}
		parseGWC, parseGWCErr := parseLocator.GetAttribute("data-gwc")
		if parseGWCErr != nil {
			parseT.Errorf("case %d: read data-gwc: %v", parseI, parseGWCErr)
			continue
		}

		// Compute the expected result using the browser's own Intl.NumberFormat.
		// This is the ground truth — whatever the browser produces for the same
		// locale+opts+value is what GWC must also produce.
		parseExpectedRaw, parseEvalErr := parsePage.Evaluate(`([parseLocale, parseOptsJSON, parseValueStr]) => {
			const parseOpts = JSON.parse(parseOptsJSON);
			const parseNum = Number(parseValueStr);
			return new Intl.NumberFormat(parseLocale, parseOpts).format(parseNum);
		}`, []interface{}{parseLocale, parseOptsJSON, parseValueStr})
		if parseEvalErr != nil {
			parseT.Errorf("case %d (%s): browser Intl eval error: %v", parseI, parseLocale, parseEvalErr)
			continue
		}
		parseExpected, _ := parseExpectedRaw.(string)

		if parseGWC == parseExpected {
			parseT.Logf("case %d PASS  locale=%-5s value=%-12s gwc=%q", parseI, parseLocale, parseValueStr, parseGWC)
		} else {
			parseT.Errorf("case %d FAIL  locale=%-5s value=%-12s gwc=%q want=%q (opts=%s)",
				parseI, parseLocale, parseValueStr, parseGWC, parseExpected, parseOptsJSON)
		}
	}

	// --- 8. (Optional) Date cross-check ---
	// Format a fixed Unix timestamp using GWC interop from within the wasm
	// binary is not directly testable here without extending the fixture, but
	// we verify the Intl API itself is exposed by evaluating a date format
	// through the browser directly as a sanity check.
	parseDateRaw, parseDateEvalErr := parsePage.Evaluate(`() => {
		// 2024-01-15 00:00:00 UTC in milliseconds
		const parseMs = 1705276800000;
		return new Intl.DateTimeFormat("en-US", {dateStyle: "short", timeZone: "UTC"}).format(new Date(parseMs));
	}`, nil)
	if parseDateEvalErr != nil {
		parseT.Logf("date sanity eval error (non-fatal): %v", parseDateEvalErr)
	} else {
		parseDateStr, _ := parseDateRaw.(string)
		parseT.Logf("date sanity check: Intl.DateTimeFormat en-US short UTC 2024-01-15 => %q", parseDateStr)
		if parseDateStr == "" {
			parseT.Errorf("date sanity: Intl.DateTimeFormat returned empty string")
		}
	}
}
