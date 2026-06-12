//go:build playwrightgo

package playwrightgoexamples_test

// TestCrossRootEventingRoundTrip exercises the document-level typed CustomEvent
// round-trip in the browser-interop public example.
//
// The example registers a subscriber via interop.SubscribeDecoded[demoPulse](...)
// and a dispatcher via interop.GetDocumentEvents().Dispatch("interop-demo", ...).
// Clicking "Dispatch pulse" increments a typed count field on the demoPulse
// detail and the subscriber updates two on-page stats:
//
//   - "Event count" stat    -> parseEventCount.Get()   (int, advances 1→2→3)
//   - "Event source" stat   -> parseEventSource.Get()  ("Go button")
//   - event status <p>      -> includes `Received` and `SubscribeDecoded`
//
// The test fires the button three times and asserts the count advances on each
// dispatch, proving the typed detail round-trips correctly and handlers are not
// duplicated (each click yields exactly one count increment, not two or more).

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// crossRootWasmReadyPredicate is the standard GWC boot signal: #app has children.
const crossRootWasmReadyPredicate = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

// waitForCrossRootAppReady blocks until the WASM app finishes booting.
func waitForCrossRootAppReady(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(crossRootWasmReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(90000),
	}); parseErr != nil {
		parseT.Fatalf("WASM app never became ready: %v", parseErr)
	}
}

// crossRootBodyContains returns a JS predicate that resolves when the body text
// includes parseNeedle.  Wrapping in a closure avoids Go string quoting inside
// the JS template literal.
func crossRootBodyContains(parseNeedle string) string {
	parseEscaped := strings.ReplaceAll(parseNeedle, `"`, `\"`)
	return fmt.Sprintf(`() => (document.body.textContent || '').includes("%s")`, parseEscaped)
}

// crossRootBodyHasCount returns a JS predicate that resolves once the
// "Event count" stat shows the expected integer value.  ExampleStat renders
// <small>Event count</small><p>N</p> in a div; Chromium's textContent
// concatenates them without a separator, yielding "Event countN" (no space).
func crossRootBodyHasCount(parseCount int) string {
	return fmt.Sprintf(`() => (document.body.textContent || '').includes('Event count%d')`, parseCount)
}

func TestCrossRootEventingRoundTrip(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// Port 18288 is not used by any other test in this package.
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18288")

	parseRoutes := discoverPublicExampleRoutes(parseT, parseRepoRoot, []string{"browser-interop"})
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

	parsePage, parsePageErr := parseBrowserHandle.NewPage()
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

	// Navigate to the browser-interop example and wait for WASM boot.
	if _, parseNavErr := parsePage.Goto(parseExampleURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to %s: %v", parseExampleURL, parseNavErr)
	}
	waitForCrossRootAppReady(parseT, parsePage)
	parseT.Log("WASM app ready")

	// Wait for the event subscriber to initialise (UseEffect fires once on mount;
	// initial status is "Waiting for interop-demo events.").
	if _, parseInitErr := parsePage.WaitForFunction(
		crossRootBodyContains("Waiting for interop-demo events."),
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(20000)},
	); parseInitErr != nil {
		parseT.Fatalf("event subscriber status never appeared: %v", parseInitErr)
	}
	parseT.Log("event subscriber initialised")

	// ---- Dispatch 1 -------------------------------------------------------
	// Click "Dispatch pulse" and assert:
	//   1. Event count stat shows 1  (typed detail.Count round-tripped).
	//   2. Event source stat shows "Go button" (typed detail.Source).
	//   3. Status paragraph mentions "Received" and "SubscribeDecoded".

	parseT.Log("clicking 'Dispatch pulse' (dispatch 1)")
	if parseClickErr := parsePage.Click("text=Dispatch pulse"); parseClickErr != nil {
		parseT.Fatalf("click 'Dispatch pulse': %v", parseClickErr)
	}

	// Wait for event status to reflect receipt of the first pulse.
	if _, parseWaitErr := parsePage.WaitForFunction(
		crossRootBodyContains("Received"),
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(15000)},
	); parseWaitErr != nil {
		parseT.Fatalf("event status never showed 'Received' after dispatch 1: %v", parseWaitErr)
	}

	// Verify Event count = 1 in the stat.
	// ExampleStat renders <small>Event count</small><p>1</p>; textContent yields "Event count1".
	if _, parseCountErr := parsePage.WaitForFunction(
		crossRootBodyHasCount(1),
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseCountErr != nil {
		parseT.Fatalf("Event count stat did not reach 1 after dispatch 1: %v", parseCountErr)
	}
	parseT.Log("dispatch 1: Event count stat = 1 (typed detail.Count round-tripped)")

	// Assert event source is "Go button" (typed detail.Source).
	if _, parseSourceErr := parsePage.WaitForFunction(
		crossRootBodyContains("Go button"),
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(5000)},
	); parseSourceErr != nil {
		parseT.Fatalf("Event source stat did not show 'Go button' after dispatch 1: %v", parseSourceErr)
	}
	parseT.Log("dispatch 1: Event source stat = 'Go button' (typed detail.Source round-tripped)")

	// Assert status mentions SubscribeDecoded (proves the subscriber path, not JS).
	if _, parseSubErr := parsePage.WaitForFunction(
		crossRootBodyContains("SubscribeDecoded"),
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(5000)},
	); parseSubErr != nil {
		parseT.Fatalf("event status did not mention 'SubscribeDecoded' after dispatch 1: %v", parseSubErr)
	}
	parseT.Log("dispatch 1: status confirms interop.SubscribeDecoded path")

	// ---- Dispatch 2 -------------------------------------------------------
	// Click again and assert count advances to 2 (no double-subscription).

	parseT.Log("clicking 'Dispatch pulse' (dispatch 2)")
	if parseClickErr := parsePage.Click("text=Dispatch pulse"); parseClickErr != nil {
		parseT.Fatalf("click 'Dispatch pulse' (dispatch 2): %v", parseClickErr)
	}

	if _, parseCount2Err := parsePage.WaitForFunction(
		crossRootBodyHasCount(2),
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseCount2Err != nil {
		parseT.Fatalf("Event count stat did not reach 2 after dispatch 2: %v", parseCount2Err)
	}
	parseT.Log("dispatch 2: Event count stat = 2 (no duplicate subscription, exactly +1)")

	// ---- Dispatch 3 -------------------------------------------------------
	// Click a third time; count must reach 3, proving repeated dispatch works.

	parseT.Log("clicking 'Dispatch pulse' (dispatch 3)")
	if parseClickErr := parsePage.Click("text=Dispatch pulse"); parseClickErr != nil {
		parseT.Fatalf("click 'Dispatch pulse' (dispatch 3): %v", parseClickErr)
	}

	if _, parseCount3Err := parsePage.WaitForFunction(
		crossRootBodyHasCount(3),
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseCount3Err != nil {
		parseT.Fatalf("Event count stat did not reach 3 after dispatch 3: %v", parseCount3Err)
	}
	parseT.Log("dispatch 3: Event count stat = 3 (typed detail advances correctly each time)")

	// ---- Final: no fatal console errors -----------------------------------
	parseCrashKeywords := []string{"Uncaught", "FATAL", "panic"}
	for _, parseMsg := range parseConsoleErrors {
		parseT.Logf("JS console error: %s", parseMsg)
		for _, parseKW := range parseCrashKeywords {
			if containsSubstring(parseMsg, parseKW) {
				parseT.Errorf("fatal JS console error: %s", parseMsg)
			}
		}
	}

	parseBodyText, parseBodyErr := parsePage.TextContent("body")
	if parseBodyErr != nil {
		parseT.Fatalf("read body text: %v", parseBodyErr)
	}
	if !strings.Contains(parseBodyText, "Received") {
		parseT.Errorf("final body text does not contain 'Received': event subscriber may have been lost")
	}

	parseT.Log("TestCrossRootEventingRoundTrip complete: dispatch->subscribe round trip verified 3×")
}
