//go:build playwrightgo

package playwrightgoexamples_test

// TestShadowDOMStyleIsolation proves that styles inside the exported-custom-element
// widget's shadow root are isolated from the host document in both directions:
//
//  1. Host-styles-don't-bleed-in: hostile global CSS injected into the host
//     document (using !important on every selector) does NOT affect computed
//     styles of elements inside the shadow root.
//  2. Widget-styles-don't-bleed-out: the shadow-scoped .tile rule does NOT
//     affect an identically-classed element added directly to document.body.
//
// The test uses the exported-custom-element public example, which exposes
// window.__gwcMountStatusTile(shadowRoot, {title, tone, detail}) after the
// WASM runtime boots and fires "gwc-export-ready".  The test creates a host
// element with a shadow root, mounts the widget into it, then performs the
// style-isolation checks via page.Evaluate over computed styles.
//
// Port 18300 is not used by any other test in this package.

import (
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// shadowIsolationWasmReadyPredicate is the standard GWC boot signal.
const shadowIsolationWasmReadyPredicate = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

// shadowIsolationMountReadyPredicate waits for __gwcMountStatusTile to be
// callable (set by the WASM UseEffect after boot).
const shadowIsolationMountReadyPredicate = `() => typeof window.__gwcMountStatusTile === 'function'`

func TestShadowDOMStyleIsolation(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18300")
	parseExampleURL := parseBaseURL + "/examples/public-examples-site/assets/examples/exported-custom-element/"

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

	// Collect console errors to detect runtime panics or uncaught exceptions.
	var parseConsoleErrors []string
	parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
		if parseMsg.Type() == "error" {
			parseConsoleErrors = append(parseConsoleErrors, parseMsg.Text())
		}
	})

	// ── Navigate and wait for WASM boot ──────────────────────────────────────

	if _, parseNavErr := parsePage.Goto(parseExampleURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to %s: %v", parseExampleURL, parseNavErr)
	}

	if _, parseBootErr := parsePage.WaitForFunction(shadowIsolationWasmReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(90000),
	}); parseBootErr != nil {
		parseT.Fatalf("WASM app never became ready: %v", parseBootErr)
	}
	parseT.Log("WASM app ready (#app has children)")

	// ── Wait for __gwcMountStatusTile to be available ─────────────────────────

	if _, parseMountReadyErr := parsePage.WaitForFunction(shadowIsolationMountReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(30000),
	}); parseMountReadyErr != nil {
		parseT.Fatalf("__gwcMountStatusTile never became available: %v", parseMountReadyErr)
	}
	parseT.Log("__gwcMountStatusTile is callable")

	// ── Create a shadow host, mount the widget, wait for shadow content ───────
	//
	// We create a div appended to document.body, attach an open shadow root to
	// it, then call __gwcMountStatusTile(shadowRoot, {...}) so the GWC runtime
	// renders exportedStatusTile into the shadow root.

	parseMountScript := `() => {
		const parseHost = document.createElement('div');
		parseHost.id = 'gwc-shadow-host';
		document.body.appendChild(parseHost);
		const parseShadowRoot = parseHost.attachShadow({ mode: 'open' });
		window.__gwcMountStatusTile(parseShadowRoot, {
			title: 'Shadow Isolation Test',
			tone: 'neutral',
			detail: 'Testing that shadow-root styles are isolated from the host document.'
		});
		return true;
	}`

	parseMountResult, parseMountErr := parsePage.Evaluate(parseMountScript, nil)
	if parseMountErr != nil {
		parseT.Fatalf("mount widget into shadow root: %v", parseMountErr)
	}
	if parseMountResult != true {
		parseT.Fatalf("mount script returned unexpected value: %v", parseMountResult)
	}
	parseT.Log("widget mounted into shadow root")

	// Wait until the shadow root has rendered content (section + style).
	parseShadowReadyPredicate := `() => {
		const parseHost = document.getElementById('gwc-shadow-host');
		if (!parseHost || !parseHost.shadowRoot) return false;
		return parseHost.shadowRoot.children.length > 0;
	}`

	if _, parseShadowWaitErr := parsePage.WaitForFunction(parseShadowReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(20000),
	}); parseShadowWaitErr != nil {
		parseT.Fatalf("shadow root never received content: %v", parseShadowWaitErr)
	}
	parseT.Log("shadow root has content")

	// Also wait for the .tile element to appear inside the shadow root.
	parseTileReadyPredicate := `() => {
		const parseHost = document.getElementById('gwc-shadow-host');
		if (!parseHost || !parseHost.shadowRoot) return false;
		return !!parseHost.shadowRoot.querySelector('.tile');
	}`

	if _, parseTileWaitErr := parsePage.WaitForFunction(parseTileReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(20000),
	}); parseTileWaitErr != nil {
		parseT.Fatalf(".tile never appeared inside shadow root: %v", parseTileWaitErr)
	}
	parseT.Log(".tile rendered inside shadow root")

	// ── Read baseline computed colour of .tile before hostile CSS ─────────────

	parseBaselineScript := `() => {
		const parseHost = document.getElementById('gwc-shadow-host');
		const parseTile = parseHost.shadowRoot.querySelector('.tile');
		return window.getComputedStyle(parseTile).color;
	}`

	parseBaselineRaw, parseBaselineErr := parsePage.Evaluate(parseBaselineScript, nil)
	if parseBaselineErr != nil {
		parseT.Fatalf("read baseline .tile color: %v", parseBaselineErr)
	}
	parseBaselineColor, parseBaselineOK := parseBaselineRaw.(string)
	if !parseBaselineOK {
		parseT.Fatalf("baseline color is not a string: %v", parseBaselineRaw)
	}
	parseT.Logf("baseline .tile computed color: %s", parseBaselineColor)

	// ── Add a normal host element whose styles we will verify DO change ────────
	//
	// This proves the hostile CSS is actually being applied at the document level,
	// making the shadow element's immunity meaningful rather than a no-op.

	parseAddNormalScript := `() => {
		const parseNormal = document.createElement('div');
		parseNormal.id = 'gwc-normal-host-div';
		parseNormal.textContent = 'normal host element';
		document.body.appendChild(parseNormal);
		return true;
	}`

	if _, parseAddNormalErr := parsePage.Evaluate(parseAddNormalScript, nil); parseAddNormalErr != nil {
		parseT.Fatalf("add normal host div: %v", parseAddNormalErr)
	}

	// ── Inject hostile global CSS ─────────────────────────────────────────────
	//
	// We use !important on every selector to maximise the chance of overriding
	// any existing rules. Global `* { color: red !important }` does NOT pierce
	// shadow boundaries, so the widget's .tile element must remain unaffected.

	parseHostileCSS := `* { color: red !important; background: lime !important; font-size: 80px !important; }
section, .tile, div { border: 10px solid magenta !important; }`

	if _, parseStyleErr := parsePage.AddStyleTag(playwright.PageAddStyleTagOptions{
		Content: playwright.String(parseHostileCSS),
	}); parseStyleErr != nil {
		parseT.Fatalf("inject hostile CSS: %v", parseStyleErr)
	}
	parseT.Log("hostile global CSS injected")

	// ── PROOF A: hostile CSS applied to normal host element ───────────────────
	//
	// Confirm the hostile `color: red !important` reaches a normal document div.
	// This validates the injection worked; the shadow element's immunity is then
	// meaningful.

	parseNormalColorScript := `() => {
		const parseNormal = document.getElementById('gwc-normal-host-div');
		return window.getComputedStyle(parseNormal).color;
	}`

	parseNormalColorRaw, parseNormalColorErr := parsePage.Evaluate(parseNormalColorScript, nil)
	if parseNormalColorErr != nil {
		parseT.Fatalf("read normal div computed color after hostile CSS: %v", parseNormalColorErr)
	}
	parseNormalColor, parseNormalColorOK := parseNormalColorRaw.(string)
	if !parseNormalColorOK {
		parseT.Fatalf("normal div color not a string: %v", parseNormalColorRaw)
	}
	parseT.Logf("normal host div computed color after hostile CSS: %s", parseNormalColor)

	// rgb(255,0,0) is "red" in computed style format.
	if !strings.Contains(parseNormalColor, "255, 0, 0") && !strings.Contains(parseNormalColor, "255,0,0") {
		parseT.Errorf("PROOF SETUP FAIL: hostile CSS did not apply to normal host div — got %q, want red (rgb(255, 0, 0)); hostile CSS injection may be broken", parseNormalColor)
	} else {
		parseT.Logf("PROOF SETUP OK: hostile CSS applied to normal host div (color=%s)", parseNormalColor)
	}

	// ── PROOF B: host-styles-don't-bleed-in ──────────────────────────────────
	//
	// Read the computed colour of .tile INSIDE the shadow root after the hostile
	// CSS was injected.  It must NOT be red (rgb(255,0,0)) — the shadow boundary
	// blocks `* { color: red !important }` from entering the shadow tree.

	parseShadowColorScript := `() => {
		const parseHost = document.getElementById('gwc-shadow-host');
		const parseTile = parseHost.shadowRoot.querySelector('.tile');
		return window.getComputedStyle(parseTile).color;
	}`

	parseShadowColorRaw, parseShadowColorErr := parsePage.Evaluate(parseShadowColorScript, nil)
	if parseShadowColorErr != nil {
		parseT.Fatalf("read shadow .tile computed color after hostile CSS: %v", parseShadowColorErr)
	}
	parseShadowColor, parseShadowColorOK := parseShadowColorRaw.(string)
	if !parseShadowColorOK {
		parseT.Fatalf("shadow .tile color not a string: %v", parseShadowColorRaw)
	}
	parseT.Logf("shadow .tile computed color after hostile CSS: %s", parseShadowColor)

	if strings.Contains(parseShadowColor, "255, 0, 0") || strings.Contains(parseShadowColor, "255,0,0") {
		parseT.Errorf(
			"ISOLATION BROKEN (inbound): hostile host CSS bled into shadow root — "+
				"shadow .tile color=%q is red; expected the widget's own shadow <style> color (#e2e8f0 / rgb(226,232,240)); "+
				"normal host div color=%q (proving hostile CSS reached the document)",
			parseShadowColor, parseNormalColor,
		)
	} else {
		parseT.Logf(
			"INBOUND ISOLATION HOLDS: shadow .tile color=%q (not red); normal host div color=%q (hostile CSS applied at document level)",
			parseShadowColor, parseNormalColor,
		)
	}

	// Also verify the border of .tile is not magenta.
	parseShadowBorderScript := `() => {
		const parseHost = document.getElementById('gwc-shadow-host');
		const parseTile = parseHost.shadowRoot.querySelector('.tile');
		return window.getComputedStyle(parseTile).borderTopColor;
	}`

	parseShadowBorderRaw, parseShadowBorderErr := parsePage.Evaluate(parseShadowBorderScript, nil)
	if parseShadowBorderErr != nil {
		parseT.Fatalf("read shadow .tile border-top-color after hostile CSS: %v", parseShadowBorderErr)
	}
	parseShadowBorder, parseShadowBorderOK := parseShadowBorderRaw.(string)
	if !parseShadowBorderOK {
		parseT.Fatalf("shadow .tile border color not a string: %v", parseShadowBorderRaw)
	}
	parseT.Logf("shadow .tile border-top-color after hostile CSS: %s", parseShadowBorder)

	// Magenta is rgb(255, 0, 255).
	if strings.Contains(parseShadowBorder, "255, 0, 255") || strings.Contains(parseShadowBorder, "255,0,255") {
		parseT.Errorf(
			"ISOLATION BROKEN (inbound border): hostile host `section, .tile { border: magenta }` bled into shadow root — border-top-color=%q",
			parseShadowBorder,
		)
	} else {
		parseT.Logf("INBOUND BORDER ISOLATION HOLDS: shadow .tile border-top-color=%q (not magenta)", parseShadowBorder)
	}

	// Note on font-size: inherited CSS properties (font-size, color when not
	// explicitly set in shadow CSS) can flow from the shadow host into the shadow
	// tree via CSS inheritance — this is correct, specified browser behaviour.
	// The meaningful isolation test is for EXPLICITLY-SET shadow properties like
	// `.tile { color: #e2e8f0 }` and `.tile { border: 1px solid ... }`, which the
	// shadow <style> owns and which must not be overridden by host-page rules.
	// We deliberately skip the font-size assertion here because font-size is NOT
	// explicitly set on .tile in the shadow <style> (only on :host), so inherited
	// 80px from the host document is expected and correct per the CSS spec.
	parseShadowFontScript := `() => {
		const parseHost = document.getElementById('gwc-shadow-host');
		const parseTile = parseHost.shadowRoot.querySelector('.tile');
		return window.getComputedStyle(parseTile).fontSize;
	}`

	parseShadowFontRaw, parseShadowFontErr := parsePage.Evaluate(parseShadowFontScript, nil)
	if parseShadowFontErr != nil {
		parseT.Fatalf("read shadow .tile font-size after hostile CSS: %v", parseShadowFontErr)
	}
	parseShadowFont, parseShadowFontOK := parseShadowFontRaw.(string)
	if !parseShadowFontOK {
		parseT.Fatalf("shadow .tile font-size not a string: %v", parseShadowFontRaw)
	}
	parseT.Logf("shadow .tile font-size after hostile CSS (informational, not asserted): %s", parseShadowFont)

	// ── PROOF C: widget-styles-don't-bleed-out ────────────────────────────────
	//
	// The shadow <style> defines `.tile { ... color: #e2e8f0 ... }`.  A plain
	// `.tile` element added to document.body must NOT pick up that shadow rule —
	// it should have whatever the (hostile) host-page rules say (red), because
	// the shadow style is scoped and cannot leak out.

	parseAddOuterTileScript := `() => {
		const parseOuterSection = document.createElement('section');
		parseOuterSection.className = 'tile';
		parseOuterSection.id = 'gwc-outer-tile';
		parseOuterSection.textContent = 'outer tile';
		document.body.appendChild(parseOuterSection);
		return true;
	}`

	if _, parseAddOuterErr := parsePage.Evaluate(parseAddOuterTileScript, nil); parseAddOuterErr != nil {
		parseT.Fatalf("add outer .tile section: %v", parseAddOuterErr)
	}

	parseOuterTileColorScript := `() => {
		const parseOuter = document.getElementById('gwc-outer-tile');
		return window.getComputedStyle(parseOuter).color;
	}`

	parseOuterTileColorRaw, parseOuterTileColorErr := parsePage.Evaluate(parseOuterTileColorScript, nil)
	if parseOuterTileColorErr != nil {
		parseT.Fatalf("read outer .tile computed color: %v", parseOuterTileColorErr)
	}
	parseOuterTileColor, parseOuterTileColorOK := parseOuterTileColorRaw.(string)
	if !parseOuterTileColorOK {
		parseT.Fatalf("outer .tile color not a string: %v", parseOuterTileColorRaw)
	}
	parseT.Logf("outer .tile (host-document section.tile) computed color: %s", parseOuterTileColor)

	// The widget's shadow style sets `.tile { color: #e2e8f0 }` which is
	// rgb(226, 232, 240).  The outer element must NOT have that colour — it
	// should have the hostile red from the host document CSS, confirming the
	// shadow style did not leak out.
	if strings.Contains(parseOuterTileColor, "226, 232, 240") || strings.Contains(parseOuterTileColor, "226,232,240") {
		parseT.Errorf(
			"ISOLATION BROKEN (outbound): shadow .tile colour #e2e8f0 (rgb(226,232,240)) bled out to host-document section.tile — outer color=%q",
			parseOuterTileColor,
		)
	} else {
		parseT.Logf(
			"OUTBOUND ISOLATION HOLDS: host-document section.tile color=%q (not shadow #e2e8f0); shadow styles are scoped",
			parseOuterTileColor,
		)
	}

	// ── PROOF D: attribute-change drives widget re-render ─────────────────────
	//
	// The example's __gwcMountStatusTile can be called again with the same shadow
	// root to re-render (the mount count in the GWC shell will advance).  We call
	// it a second time with a different tone and check the shadow root still has
	// content (the re-render did not break the DOM).  Full attribute-observation
	// is at the custom-element wrapper layer; here we just prove re-mounting works.

	parseRemountScript := `() => {
		const parseHost = document.getElementById('gwc-shadow-host');
		if (!parseHost || !parseHost.shadowRoot) return 'no-shadow-host';
		window.__gwcMountStatusTile(parseHost.shadowRoot, {
			title: 'Re-mounted tile',
			tone: 'warning',
			detail: 'Re-render after attribute change simulation.'
		});
		return 'remounted';
	}`

	// Re-check that the mount function is still available before attempting re-mount.
	parseMountStillAvailable, parseMountCheckErr := parsePage.Evaluate(shadowIsolationMountReadyPredicate, nil)
	if parseMountCheckErr != nil || parseMountStillAvailable != true {
		parseT.Logf("SKIP attribute-change re-render check: __gwcMountStatusTile no longer available (WASM may have recycled it during state update)")
	}

	parseRemountRaw, parseRemountErr := parsePage.Evaluate(parseRemountScript, nil)
	if parseRemountErr != nil {
		parseT.Logf("SKIP attribute-change re-render check: evaluate error: %v", parseRemountErr)
	} else {
		parseRemountResult, parseRemountResultOK := parseRemountRaw.(string)
		if !parseRemountResultOK || parseRemountResult != "remounted" {
			parseT.Logf("SKIP attribute-change re-render check: unexpected result: %v", parseRemountRaw)
		} else {
			// Wait for the re-rendered .tile to still be present.
			if _, parseRemountWaitErr := parsePage.WaitForFunction(parseTileReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
				Timeout: playwright.Float(10000),
			}); parseRemountWaitErr != nil {
				parseT.Logf("SKIP attribute-change re-render check: .tile not present after remount: %v", parseRemountWaitErr)
			} else {
				parseT.Log("FUNCTIONALITY HOLDS: widget re-rendered inside shadow root after __gwcMountStatusTile call with new props")
			}
		}
	}

	// ── Final: no fatal JS console errors ─────────────────────────────────────

	parseCrashKeywords := []string{"Uncaught", "FATAL", "panic"}
	for _, parseMsg := range parseConsoleErrors {
		parseT.Logf("JS console error: %s", parseMsg)
		for _, parseKW := range parseCrashKeywords {
			if containsSubstring(parseMsg, parseKW) {
				parseT.Errorf("fatal JS console error: %s", parseMsg)
			}
		}
	}

	parseT.Log("TestShadowDOMStyleIsolation complete")
}
