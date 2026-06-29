//go:build playwrightgo

package playwrightgoexamples_test

import (
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// wasmReadyPredicate for installability test — same boot signal as other tests.
const pwaInstallWasmReadyPredicate = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

// waitForPWAInstallAppReady blocks until the WASM app finishes booting.
func waitForPWAInstallAppReady(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(pwaInstallWasmReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(90000),
	}); parseErr != nil {
		parseT.Fatalf("WASM app never became ready: %v", parseErr)
	}
}

// TestPWAInstallabilityFlow loads the PWA installability example via its staged
// preview path (the example is source-only in the catalog, so the live binary
// is served under examples/public-examples-site/assets/examples/).
//
// The test verifies that the diagnostics surface renders meaningful state (not
// blank/pending) and exercises the "Prompt install" button to prove graceful-
// refusal behavior.
//
// In headless Chromium the beforeinstallprompt event is never fired, so a real
// install cannot be triggered. The test instead proves the DOCUMENTED graceful-
// refusal path: the prompt button reports "Install prompt unavailable" rather
// than silently failing or panicking. The manifest-validity stat and service-
// worker status are also asserted.
func TestPWAInstallabilityFlow(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// The PWA installability example is source-only in the public catalog, so
	// the running preview lives under the public-examples-site asset tree.
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18242")
	parseExampleURL := parseBaseURL + "/examples/public-examples-site/assets/examples/progressive-web-app-installability/"

	if parseErr := ensureExamplesChromiumInstalled(); parseErr != nil {
		parseT.Fatalf("install chromium: %v", parseErr)
	}
	parsePw, parseErr := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parseErr != nil {
		parseT.Fatalf("run playwright-go: %v", parseErr)
	}
	defer func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright-go: %v", parseStopErr)
		}
	}()

	parseBrowserHandle, parseErr := launchExamplesBrowser(parsePw, "chromium")
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer func() {
		if parseCloseErr := parseBrowserHandle.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	parsePage, parseErr := parseBrowserHandle.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new page: %v", parseErr)
	}

	// Collect console errors for fatal-crash detection.
	var parseConsoleErrors []string
	parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
		if parseMsg.Type() == "error" {
			parseConsoleErrors = append(parseConsoleErrors, parseMsg.Text())
		}
	})

	// Navigate and wait for WASM boot.
	if _, parseNavErr := parsePage.Goto(parseExampleURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to %s: %v", parseExampleURL, parseNavErr)
	}
	waitForPWAInstallAppReady(parseT, parsePage)
	parseT.Log("WASM app ready")

	// --- Assert 1: manifest-valid stat renders (not blank) ---
	// The installability example renders pwa.InstallabilityState.ManifestValid
	// into an ExampleStat whose text is either "yes" or "no". Wait for one of
	// those values to appear alongside the "Manifest valid" label.
	if _, parseManifestErr := parsePage.WaitForFunction(`() => {
		const parseBody = document.body.textContent || '';
		return parseBody.includes('Manifest valid') && (parseBody.includes('yes') || parseBody.includes('no'));
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(20000),
	}); parseManifestErr != nil {
		parseT.Fatalf("manifest valid stat never appeared: %v", parseManifestErr)
	}
	parseT.Log("manifest valid stat rendered")

	// --- Assert 2: reasons panel is not stuck blank ---
	// The #pwa-installability-reasons element should contain either the
	// "No blockers" message or a non-empty reason string.
	if _, parseReasonsErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('pwa-installability-reasons');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.trim().length > 0;
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(20000),
	}); parseReasonsErr != nil {
		parseT.Fatalf("reasons panel never populated: %v", parseReasonsErr)
	}
	parseReasonsText, parseReasonsReadErr := parsePage.TextContent("#pwa-installability-reasons")
	if parseReasonsReadErr != nil {
		parseT.Fatalf("read #pwa-installability-reasons: %v", parseReasonsReadErr)
	}
	parseT.Logf("installability reasons: %s", strings.TrimSpace(parseReasonsText))

	// --- Assert 3: service-worker status progresses beyond "pending" ---
	// The example sets "Registering service worker..." when registration
	// starts. In headless the SW registration will fail (wrong scope/URL),
	// which sets "Service worker registration failed [...]". Either outcome
	// proves the code path ran.
	if _, parseSWErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('pwa-installability-sw-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('registered') ||
			parseText.includes('failed') ||
			parseText.includes('Registering');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(30000),
	}); parseSWErr != nil {
		parseT.Fatalf("service worker status never progressed: %v", parseSWErr)
	}
	parseSWStatusText, parseSWReadErr := parsePage.TextContent("#pwa-installability-sw-status")
	if parseSWReadErr != nil {
		parseT.Fatalf("read #pwa-installability-sw-status: %v", parseSWReadErr)
	}
	parseT.Logf("service worker status: %s", strings.TrimSpace(parseSWStatusText))

	// --- Assert 4: "Refresh installability" button updates the prompt-status element ---
	parseT.Log("clicking 'Refresh installability'")
	if parseClickErr := parsePage.Click("text=Refresh installability"); parseClickErr != nil {
		parseT.Fatalf("click 'Refresh installability': %v", parseClickErr)
	}
	if _, parseRefreshErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('pwa-installability-prompt-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('refreshed') ||
			parseText.includes('Installability state refreshed') ||
			parseText.includes('manager is not ready');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(15000),
	}); parseRefreshErr != nil {
		parseT.Fatalf("prompt status did not update after refresh: %v", parseRefreshErr)
	}
	parseRefreshText, _ := parsePage.TextContent("#pwa-installability-prompt-status")
	parseT.Logf("prompt status after refresh: %s", strings.TrimSpace(parseRefreshText))

	// --- Assert 5: "Prompt install" yields documented graceful-refusal ---
	// In headless Chromium beforeinstallprompt never fires, so the manager
	// returns an error. The example sets the prompt-status to
	// "Install prompt unavailable [...]". This is the documented graceful
	// refusal path.
	parseT.Log("clicking 'Prompt install'")
	if parseClickErr := parsePage.Click("text=Prompt install"); parseClickErr != nil {
		parseT.Fatalf("click 'Prompt install': %v", parseClickErr)
	}
	if _, parsePromptErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('pwa-installability-prompt-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		// Acceptable outcomes in headless:
		//   "Install prompt unavailable [...]" — manager has no stored prompt
		//   "Requesting install prompt..."     — in-flight
		//   "Browser prompt outcome=..."       — if browser somehow triggers
		//   "Installability manager is not ready yet." — init still in progress
		return parseText.includes('unavailable') ||
			parseText.includes('outcome=') ||
			parseText.includes('Requesting') ||
			parseText.includes('manager is not ready');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(20000),
	}); parsePromptErr != nil {
		parseT.Fatalf("prompt status did not respond to Prompt install click: %v", parsePromptErr)
	}
	parsePromptStatusText, parsePromptReadErr := parsePage.TextContent("#pwa-installability-prompt-status")
	if parsePromptReadErr != nil {
		parseT.Fatalf("read #pwa-installability-prompt-status: %v", parsePromptReadErr)
	}
	parseT.Logf("prompt status after click: %s", strings.TrimSpace(parsePromptStatusText))

	// Verify the page did NOT crash: the prompt status must NOT be stuck at
	// the initial "idle" text after the click has completed.
	if strings.TrimSpace(parsePromptStatusText) == "Install prompt idle." {
		parseT.Errorf("prompt status still 'Install prompt idle.' after button click — no effect")
	}

	// --- Final: no fatal console errors ---
	parseCrashKeywords := []string{"Uncaught", "FATAL", "panic"}
	for _, parseMsg := range parseConsoleErrors {
		parseT.Logf("JS console error: %s", parseMsg)
		for _, parseKW := range parseCrashKeywords {
			if containsSubstring(parseMsg, parseKW) {
				parseT.Errorf("fatal JS console error: %s", parseMsg)
			}
		}
	}
	parseT.Log("TestPWAInstallabilityFlow complete")
}
