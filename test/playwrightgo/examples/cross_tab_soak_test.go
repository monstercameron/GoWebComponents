//go:build playwrightgo

package playwrightgoexamples_test

import (
	"runtime"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// wasmReadyPredicate returns true once the WASM app has rendered at least one
// child under the #app root element, which is the standard GWC boot signal.
const crossTabWasmReadyPredicate = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

// waitForCrossTabAppReady blocks until the WASM app finishes booting on the
// given page, using a generous timeout to cover cold-start asset transfer.
func waitForCrossTabAppReady(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(crossTabWasmReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(90000),
	}); parseErr != nil {
		parseT.Fatalf("WASM app never became ready: %v", parseErr)
	}
}

// collectCrossTabConsoleErrors attaches a console listener to a page and
// returns a function that retrieves any accumulated JS error messages.
func collectCrossTabConsoleErrors(parsePage playwright.Page) func() []string {
	var parseErrors []string
	parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
		if parseMsg.Type() == "error" {
			parseErrors = append(parseErrors, parseMsg.Text())
		}
	})
	return func() []string { return parseErrors }
}

// TestCrossTabEventingSoak opens the cross-tab-sync public example in two
// pages that share a single browser context (and therefore share a
// BroadcastChannel origin), drives a broadcast on page A, and verifies that
// page B receives the event. It also checks that the diagnostics panel
// resolves a transport name and logs a received message, and that closing page
// A leaves page B interactive with no fatal console errors.
func TestCrossTabEventingSoak(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18231")

	parseRoutes := discoverPublicExampleRoutes(parseT, parseRepoRoot, []string{"cross-tab-sync"})
	parseExampleURL := parseBaseURL + parseRoutes[0]

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

	// Both pages share one browsing context so BroadcastChannel messages flow
	// between them without requiring same-origin window references.
	parseCtx, parseCtxErr := parseBrowserHandle.NewContext()
	if parseCtxErr != nil {
		parseT.Fatalf("new browser context: %v", parseCtxErr)
	}
	defer func() {
		if parseCloseErr := parseCtx.Close(); parseCloseErr != nil {
			parseT.Errorf("close browser context: %v", parseCloseErr)
		}
	}()

	// --- Page A (sender) ---
	parsePageA, parsePageAErr := parseCtx.NewPage()
	if parsePageAErr != nil {
		parseT.Fatalf("new page A: %v", parsePageAErr)
	}
	parseGetConsoleErrorsA := collectCrossTabConsoleErrors(parsePageA)

	if _, parseNavErrA := parsePageA.Goto(parseExampleURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErrA != nil {
		parseT.Fatalf("navigate page A to %s: %v", parseExampleURL, parseNavErrA)
	}
	waitForCrossTabAppReady(parseT, parsePageA)
	parseT.Log("page A WASM ready")

	// Wait for the initial diagnostics panel to show a resolved transport (not
	// "pending") on page A, confirming channels are open.
	if _, parseWaitErrA := parsePageA.WaitForFunction(`() => {
		const parseBody = document.body.textContent || '';
		return parseBody.includes('BroadcastChannel') || parseBody.includes('localStorage');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(15000),
	}); parseWaitErrA != nil {
		// Non-fatal: some browsers may keep "pending" briefly; proceed and let
		// later assertions catch a real delivery failure.
		parseT.Logf("page A transport did not resolve before theme broadcast: %v", parseWaitErrA)
	}

	// --- Page B (receiver) ---
	parsePageB, parsePageBErr := parseCtx.NewPage()
	if parsePageBErr != nil {
		parseT.Fatalf("new page B: %v", parsePageBErr)
	}
	parseGetConsoleErrorsB := collectCrossTabConsoleErrors(parsePageB)

	if _, parseNavErrB := parsePageB.Goto(parseExampleURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErrB != nil {
		parseT.Fatalf("navigate page B to %s: %v", parseExampleURL, parseNavErrB)
	}
	waitForCrossTabAppReady(parseT, parsePageB)
	parseT.Log("page B WASM ready")

	// --- Step 1: broadcast theme "dark" from page A ---
	// The "Theme: dark" button lives in the first panel of the cross-tab-sync
	// example (ExampleButton renders as a <button> with that exact text).
	parseT.Log("clicking 'Theme: dark' on page A")
	if parseClickErr := parsePageA.Click("text=Theme: dark"); parseClickErr != nil {
		parseT.Fatalf("click 'Theme: dark' on page A: %v", parseClickErr)
	}

	// Page A's own log should record the send immediately.
	if _, parseLogErrA := parsePageA.WaitForFunction(`() => {
		return (document.body.textContent || '').includes('Sent theme update: dark via');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(10000),
	}); parseLogErrA != nil {
		parseT.Fatalf("page A log never showed 'Sent theme update: dark via': %v", parseLogErrA)
	}
	parseT.Log("page A confirmed theme broadcast in its own log")

	// --- Step 2: page B must reflect the received theme in its log ---
	// The subscription handler appends: `Received theme "dark" from <source>.`
	if _, parseLogErrB := parsePageB.WaitForFunction(`() => {
		return (document.body.textContent || '').includes('Received theme');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(15000),
	}); parseLogErrB != nil {
		parseT.Fatalf("page B never received theme event from page A: %v", parseLogErrB)
	}
	parseT.Log("page B confirmed cross-tab theme delivery")

	// --- Step 3: verify diagnostics transport on page B is not "pending" ---
	// After the first send/receive cycle the channels are active, so at least
	// the theme transport stat must have resolved to a non-pending value.
	if _, parseTransportErr := parsePageB.WaitForFunction(`() => {
		const parseText = document.body.textContent || '';
		return parseText.includes('BroadcastChannel') || parseText.includes('localStorage');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(10000),
	}); parseTransportErr != nil {
		parseT.Fatalf("page B diagnostics never showed a resolved transport name: %v", parseTransportErr)
	}
	parseT.Log("page B diagnostics panel shows a resolved transport")

	// --- Step 4: broadcast a second signal (logout) from page A ---
	parseT.Log("clicking 'Broadcast logout' on page A")
	if parseClickErr := parsePageA.Click("text=Broadcast logout"); parseClickErr != nil {
		parseT.Fatalf("click 'Broadcast logout' on page A: %v", parseClickErr)
	}

	if _, parseLogoutErrB := parsePageB.WaitForFunction(`() => {
		return (document.body.textContent || '').includes('Received auth signal');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(15000),
	}); parseLogoutErrB != nil {
		parseT.Fatalf("page B never received auth/logout event: %v", parseLogoutErrB)
	}
	parseT.Log("page B confirmed cross-tab auth/logout delivery")

	// --- Step 5: broadcast a draft from page A ---
	// The draft textarea has id="cross-tab-draft". Fill it, then click
	// "Broadcast draft" so the draftSignal is sent with version 2.
	parseT.Log("filling draft textarea and broadcasting from page A")
	if parseFillErr := parsePageA.Fill("#cross-tab-draft", "soak-test draft payload"); parseFillErr != nil {
		parseT.Fatalf("fill draft textarea on page A: %v", parseFillErr)
	}
	if parseClickErr := parsePageA.Click("text=Broadcast draft"); parseClickErr != nil {
		parseT.Fatalf("click 'Broadcast draft' on page A: %v", parseClickErr)
	}

	// Page B should receive the draft and update its textarea value.
	if _, parseDraftErrB := parsePageB.WaitForFunction(`() => {
		const parseTA = document.getElementById('cross-tab-draft');
		return !!(parseTA && parseTA.value === 'soak-test draft payload');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(15000),
	}); parseDraftErrB != nil {
		parseT.Fatalf("page B textarea never reflected draft from page A: %v", parseDraftErrB)
	}
	parseT.Log("page B confirmed draft delivery into textarea")

	// Page B's log should also mention the received draft version.
	if _, parseDraftLogB := parsePageB.WaitForFunction(`() => {
		return (document.body.textContent || '').includes('Received draft version');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(10000),
	}); parseDraftLogB != nil {
		parseT.Fatalf("page B log never showed 'Received draft version': %v", parseDraftLogB)
	}
	parseT.Log("page B log confirmed draft version receipt")

	// --- Step 6: close page A; confirm page B remains interactive ---
	parseT.Log("closing page A")
	if parseCloseErrA := parsePageA.Close(); parseCloseErrA != nil {
		parseT.Errorf("close page A: %v", parseCloseErrA)
	}

	// After closing page A, page B should still be interactive: clicking
	// "Invalidate cache" should update page B's own log without crashing.
	parseT.Log("triggering 'Invalidate cache' on page B after page A is gone")
	if parseClickErr := parsePageB.Click("text=Invalidate cache"); parseClickErr != nil {
		parseT.Fatalf("click 'Invalidate cache' on page B after page A closed: %v", parseClickErr)
	}
	if _, parseCacheLog := parsePageB.WaitForFunction(`() => {
		return (document.body.textContent || '').includes('Broadcast cache invalidation rev');
	}`, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(10000),
	}); parseCacheLog != nil {
		parseT.Fatalf("page B cache invalidation log never appeared: %v", parseCacheLog)
	}
	parseT.Log("page B remains interactive after page A was closed")

	// --- Final: check for fatal console errors ---
	parseConsoleErrorsA := parseGetConsoleErrorsA()
	parseConsoleErrorsB := parseGetConsoleErrorsB()
	for _, parseMsg := range parseConsoleErrorsA {
		parseT.Logf("page A JS error: %s", parseMsg)
	}
	for _, parseMsg := range parseConsoleErrorsB {
		parseT.Logf("page B JS error: %s", parseMsg)
	}
	// Fail only on errors that indicate a crash (not WASM compile noise).
	parseCrashKeywords := []string{"Uncaught", "FATAL", "panic"}
	for _, parseMsg := range append(parseConsoleErrorsA, parseConsoleErrorsB...) {
		for _, parseKW := range parseCrashKeywords {
			if len(parseMsg) > 0 && containsSubstring(parseMsg, parseKW) {
				parseT.Errorf("fatal JS console error: %s", parseMsg)
			}
		}
	}

	parseT.Logf("TestCrossTabEventingSoak complete: cross-tab delivery verified for theme, auth, and draft channels")
}

// containsSubstring returns true if parseHaystack contains parseNeedle.
// A simple wrapper to keep the test body readable.
func containsSubstring(parseHaystack, parseNeedle string) bool {
	return len(parseHaystack) >= len(parseNeedle) &&
		func() bool {
			for parseI := 0; parseI <= len(parseHaystack)-len(parseNeedle); parseI++ {
				if parseHaystack[parseI:parseI+len(parseNeedle)] == parseNeedle {
					return true
				}
			}
			return false
		}()
}
