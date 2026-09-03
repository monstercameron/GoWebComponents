//go:build playwrightgo

package playwrightgoexamples_test

import (
	"runtime"
	"strings"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

// wasmReadyPredicate for offline cache test — same boot signal as other tests.
const pwaOfflineWasmReadyPredicate = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

// waitForPWAOfflineAppReady blocks until the WASM app finishes booting.
func waitForPWAOfflineAppReady(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(pwaOfflineWasmReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(90000),
	}); parseErr != nil {
		parseT.Fatalf("WASM app never became ready: %v", parseErr)
	}
}

// TestPWAOfflineMutationReplay loads the PWA offline-cache example via its
// staged preview path (source-only in the catalog, live binary served under
// examples/public-examples-site/assets/examples/) and exercises the documented
// offline-queue flow:
//
//  1. Wait for Cache Storage manager + mutation queue to initialize.
//  2. Warm the offline shell cache.
//  3. Go offline via Playwright BrowserContext.SetOffline(true).
//  4. Queue an offline write and a conflicting write.
//  5. Come back online via SetOffline(false).
//  6. Replay queued writes — the no-op handler returns nil so all succeed.
//  7. Inspect diagnostics and assert the structured snapshot contains cache
//     entries, queue counts, and background-sync capability.
//
// The offline/online transition is exercised with real Playwright network
// gating. Cache Storage and IndexedDB are local so writes succeed regardless
// of network state; replay is run online so the no-op handler succeeds.
func TestPWAOfflineMutationReplay(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// The PWA offline-cache example is source-only in the public catalog, so
	// the running preview lives under the public-examples-site asset tree.
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18243")
	parseExampleURL := parseBaseURL + "/examples/public-examples-site/assets/examples/progressive-web-app-offline-cache/"

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

	// Use a named BrowserContext so we can call SetOffline on it.
	parseCtx, parseCtxErr := parseBrowserHandle.NewContext()
	if parseCtxErr != nil {
		parseT.Fatalf("new browser context: %v", parseCtxErr)
	}
	defer func() {
		if parseCloseErr := parseCtx.Close(); parseCloseErr != nil {
			parseT.Errorf("close browser context: %v", parseCloseErr)
		}
	}()

	parsePage, parsePageErr := parseCtx.NewPage()
	if parsePageErr != nil {
		parseT.Fatalf("new page: %v", parsePageErr)
	}

	// Collect console errors for fatal-crash detection.
	var parseConsoleErrors []string
	parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
		if parseMsg.Type() == "error" {
			parseConsoleErrors = append(parseConsoleErrors, parseMsg.Text())
		}
	})

	// Navigate and wait for WASM boot (must be online for initial page load
	// so the WASM binary can be fetched).
	if _, parseNavErr := parsePage.Goto(parseExampleURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to %s: %v", parseExampleURL, parseNavErr)
	}
	waitForPWAOfflineAppReady(parseT, parsePage)
	parseT.Log("WASM app ready")

	// --- Wait for Cache Storage manager and mutation queue to initialize ---
	// Both are initialized in UseEffect goroutines; we wait for their
	// ready status messages before interacting with the buttons.
	if _, parseCacheReadyErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('offline-cache-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('ready') || parseText.includes('unavailable') || parseText.includes('failed');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(30000),
	}); parseCacheReadyErr != nil {
		parseT.Fatalf("cache storage manager never signalled ready: %v", parseCacheReadyErr)
	}
	parseCacheStatusText, _ := parsePage.TextContent("#offline-cache-status")
	parseT.Logf("cache status: %s", strings.TrimSpace(parseCacheStatusText))

	if _, parseQueueReadyErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('offline-queue-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('ready') || parseText.includes('unavailable') || parseText.includes('failed');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(30000),
	}); parseQueueReadyErr != nil {
		parseT.Fatalf("mutation queue never signalled ready: %v", parseQueueReadyErr)
	}
	parseQueueStatusText, _ := parsePage.TextContent("#offline-queue-status")
	parseT.Logf("queue status: %s", strings.TrimSpace(parseQueueStatusText))

	// --- Step 1: Warm offline cache (online) ---
	// After warmup the cache-status reads:
	//   "Cached N release entries into pwa-offline-cache-demo-v1."
	// On network failure it reads: "Cache warmup failed: ..."
	parseT.Log("clicking 'Warm offline cache'")
	if parseClickErr := parsePage.Click("text=Warm offline cache"); parseClickErr != nil {
		parseT.Fatalf("click 'Warm offline cache': %v", parseClickErr)
	}
	if _, parseWarmErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('offline-cache-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('Cached') || parseText.includes('failed') || parseText.includes('Warming');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(60000),
	}); parseWarmErr != nil {
		parseT.Fatalf("cache warmup never updated status: %v", parseWarmErr)
	}
	parseWarmText, _ := parsePage.TextContent("#offline-cache-status")
	parseT.Logf("warm result: %s", strings.TrimSpace(parseWarmText))

	// --- Step 2: Simulate going offline, then queue an offline write ---
	parseT.Log("going offline")
	if parseOfflineErr := parseCtx.SetOffline(true); parseOfflineErr != nil {
		parseT.Fatalf("set offline: %v", parseOfflineErr)
	}

	parseT.Log("clicking 'Queue offline write' while offline")
	if parseClickErr := parsePage.Click("text=Queue offline write"); parseClickErr != nil {
		parseT.Fatalf("click 'Queue offline write': %v", parseClickErr)
	}
	// Queue writes use IndexedDB which is local and always succeeds.
	// Success: "Queued offline write <id> with state=queued."
	if _, parseQueueErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('offline-queue-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('Queued offline write') || parseText.includes('failed');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(20000),
	}); parseQueueErr != nil {
		parseT.Fatalf("queue write never updated status: %v", parseQueueErr)
	}
	parseQueueWriteText, _ := parsePage.TextContent("#offline-queue-status")
	parseT.Logf("queue write result: %s", strings.TrimSpace(parseQueueWriteText))

	// Fail the test if the queue write itself failed.
	if strings.Contains(parseQueueWriteText, "failed") && !strings.Contains(parseQueueWriteText, "Queued") {
		parseT.Errorf("offline queue write failed (unexpected): %s", strings.TrimSpace(parseQueueWriteText))
	}

	// --- Step 3: Queue a conflicting write while still offline ---
	parseT.Log("clicking 'Queue conflicting write' while offline")
	if parseClickErr := parsePage.Click("text=Queue conflicting write"); parseClickErr != nil {
		parseT.Fatalf("click 'Queue conflicting write': %v", parseClickErr)
	}
	// Success: "Queued conflict demo write <id> at revision local-1."
	if _, parseConflictErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('offline-conflict-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('Queued conflict demo write') || parseText.includes('failed');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(20000),
	}); parseConflictErr != nil {
		parseT.Fatalf("conflict queue write never updated status: %v", parseConflictErr)
	}
	parseConflictText, _ := parsePage.TextContent("#offline-conflict-status")
	parseT.Logf("conflict write result: %s", strings.TrimSpace(parseConflictText))

	// --- Step 4: Come back online and replay queued writes ---
	parseT.Log("coming back online")
	if parseOnlineErr := parseCtx.SetOffline(false); parseOnlineErr != nil {
		parseT.Fatalf("set online: %v", parseOnlineErr)
	}

	parseT.Log("clicking 'Replay queued writes'")
	if parseClickErr := parsePage.Click("text=Replay queued writes"); parseClickErr != nil {
		parseT.Fatalf("click 'Replay queued writes': %v", parseClickErr)
	}
	// The no-op handler always returns nil, so all queued writes succeed.
	// Success: "Replay succeeded=N retried=N dead=N remaining=N."
	// The demo.conflict entry gets replayed too here (no conflict is
	// detected by the plain replay handler).
	if _, parseReplayErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('offline-replay-status');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('Replay succeeded') || parseText.includes('failed');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(30000),
	}); parseReplayErr != nil {
		parseT.Fatalf("replay status never updated: %v", parseReplayErr)
	}
	parseReplayText, _ := parsePage.TextContent("#offline-replay-status")
	parseT.Logf("replay result: %s", strings.TrimSpace(parseReplayText))

	// Assert: replay must report "succeeded" (the no-op handler never fails).
	if !strings.Contains(parseReplayText, "succeeded") {
		parseT.Errorf("replay did not report succeeded: %s", strings.TrimSpace(parseReplayText))
	}

	// --- Step 5: Inspect diagnostics and assert the structured panel ---
	parseT.Log("clicking 'Inspect diagnostics'")
	if parseClickErr := parsePage.Click("text=Inspect diagnostics"); parseClickErr != nil {
		parseT.Fatalf("click 'Inspect diagnostics': %v", parseClickErr)
	}
	// The diagnostics preview <pre> shows lines like:
	//   "manifest valid: true"
	//   "cache entries: N"
	//   "queued writes: N"
	//   "storage pressure: normal"
	//   "background sync available: true/false"
	if _, parseDiagnosticsErr := parsePage.WaitForFunction(`() => {
		const parseEl = document.getElementById('offline-diagnostics-preview');
		if (!parseEl) return false;
		const parseText = parseEl.textContent || '';
		return parseText.includes('manifest valid:') && parseText.includes('cache entries:');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(30000),
	}); parseDiagnosticsErr != nil {
		parseT.Fatalf("diagnostics preview never populated with structured data: %v", parseDiagnosticsErr)
	}
	parseDiagnosticsText, parseDiagnosticsReadErr := parsePage.TextContent("#offline-diagnostics-preview")
	if parseDiagnosticsReadErr != nil {
		parseT.Fatalf("read #offline-diagnostics-preview: %v", parseDiagnosticsReadErr)
	}
	parseT.Logf("diagnostics preview:\n%s", strings.TrimSpace(parseDiagnosticsText))

	// Assert all required structured fields are present.
	parseDiagnosticsRequired := []string{
		"manifest valid:",
		"cache entries:",
		"queued writes:",
		"storage pressure:",
		"background sync available:",
	}
	for _, parseField := range parseDiagnosticsRequired {
		if !strings.Contains(parseDiagnosticsText, parseField) {
			parseT.Errorf("diagnostics preview missing field %q", parseField)
		}
	}

	// --- Assert: diagnostics summary panel rendered ---
	// offlineSummary() returns "cache entries=N | queued=N | retrying=N | dead=N | storage=... N% used"
	parseDiagnosticsSummary, parseSummaryReadErr := parsePage.TextContent("#offline-diagnostics-summary")
	if parseSummaryReadErr != nil {
		parseT.Fatalf("read #offline-diagnostics-summary: %v", parseSummaryReadErr)
	}
	parseT.Logf("diagnostics summary: %s", strings.TrimSpace(parseDiagnosticsSummary))

	if !strings.Contains(parseDiagnosticsSummary, "cache entries=") {
		parseT.Errorf("diagnostics summary missing 'cache entries=': %s", strings.TrimSpace(parseDiagnosticsSummary))
	}
	if !strings.Contains(parseDiagnosticsSummary, "queued=") {
		parseT.Errorf("diagnostics summary missing 'queued=': %s", strings.TrimSpace(parseDiagnosticsSummary))
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
	parseT.Log("TestPWAOfflineMutationReplay complete")
}
