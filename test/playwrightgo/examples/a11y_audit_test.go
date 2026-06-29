//go:build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// a11yWasmReadyPredicate waits until the WASM app has rendered children into #app.
const a11yWasmReadyPredicate = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

const a11yDocsSiteReadyPredicate = `() => {
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

// parseAxeViolation holds the fields we care about for reporting.
type parseAxeViolation struct {
	ID      string
	Impact  string
	Help    string
	HelpURL string
	Targets []string
}

// injectAndRunAxe injects vendored axe-core into the page, runs a WCAG 2A/2AA
// audit, and returns only the violations with impact "serious" or "critical".
// It reads the vendored axe.min.js from testdata/ next to the test file.
func injectAndRunAxe(parseT *testing.T, parsePage playwright.Page, parseAxeSource []byte) []parseAxeViolation {
	parseT.Helper()

	// Inject axe-core from the vendored copy.
	if _, parseAddErr := parsePage.AddScriptTag(playwright.PageAddScriptTagOptions{
		Content: playwright.String(string(parseAxeSource)),
	}); parseAddErr != nil {
		parseT.Fatalf("AddScriptTag (axe-core): %v", parseAddErr)
	}

	// Run the audit restricted to WCAG 2A and 2AA rules.
	parseRaw, parseEvalErr := parsePage.Evaluate(`async () => {
		const parseResults = await axe.run(document, {
			runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa'] }
		});
		return parseResults.violations.map(parseV => ({
			id:      parseV.id,
			impact:  parseV.impact,
			help:    parseV.help,
			helpUrl: parseV.helpUrl,
			targets: parseV.nodes.map(parseN => parseN.target.join(', ')),
		}));
	}`, nil)
	if parseEvalErr != nil {
		parseT.Fatalf("axe.run evaluate: %v", parseEvalErr)
	}

	// parseRaw is []interface{}, each element is map[string]interface{}.
	parseRawSlice, parseOk := parseRaw.([]interface{})
	if !parseOk {
		// axe returned no violations or unexpected type — treat as empty.
		return nil
	}

	var parseViolations []parseAxeViolation
	for _, parseItem := range parseRawSlice {
		parseMap, parseMapOk := parseItem.(map[string]interface{})
		if !parseMapOk {
			continue
		}
		parseImpact, _ := parseMap["impact"].(string)
		// Filter to serious or critical only.
		if parseImpact != "serious" && parseImpact != "critical" {
			continue
		}
		parseID, _ := parseMap["id"].(string)
		parseHelp, _ := parseMap["help"].(string)
		parseHelpURL, _ := parseMap["helpUrl"].(string)

		var parseTargets []string
		if parseRawTargets, parseTargetOk := parseMap["targets"].([]interface{}); parseTargetOk {
			for _, parseTarget := range parseRawTargets {
				if parseTargetStr, parseTSOk := parseTarget.(string); parseTSOk {
					parseTargets = append(parseTargets, parseTargetStr)
				}
			}
		}

		parseViolations = append(parseViolations, parseAxeViolation{
			ID:      parseID,
			Impact:  parseImpact,
			Help:    parseHelp,
			HelpURL: parseHelpURL,
			Targets: parseTargets,
		})
	}
	return parseViolations
}

func failOnAxeViolations(parseT *testing.T, parseLabel string, parseViolations []parseAxeViolation) {
	parseT.Helper()
	if len(parseViolations) == 0 {
		parseT.Logf("CLEAN: no serious/critical violations in %s", parseLabel)
		return
	}
	var parseSummaryLines []string
	parseSummaryLines = append(parseSummaryLines, fmt.Sprintf(
		"axe WCAG2A/2AA audit found %d serious/critical violation(s) in %s:", len(parseViolations), parseLabel,
	))
	for _, parseV := range parseViolations {
		parseSummaryLines = append(parseSummaryLines, fmt.Sprintf(
			"  [%s] rule=%s  help=%q  url=%s",
			parseV.Impact, parseV.ID, parseV.Help, parseV.HelpURL,
		))
		for _, parseTgt := range parseV.Targets {
			parseSummaryLines = append(parseSummaryLines, fmt.Sprintf("    node: %s", parseTgt))
		}
	}
	parseT.Fatalf("%s", strings.Join(parseSummaryLines, "\n"))
}

func readVendoredAxeSource(parseT *testing.T) []byte {
	parseT.Helper()
	_, parseFile, _, _ := runtime.Caller(0)
	parseAxeSource, parseAxeReadErr := os.ReadFile(filepath.Join(filepath.Dir(parseFile), "testdata", "axe.min.js"))
	if parseAxeReadErr != nil {
		parseT.Fatalf("read testdata/axe.min.js: %v", parseAxeReadErr)
	}
	return parseAxeSource
}

// TestAccessibilityAuditDocsSiteShell loads the documentation/examples catalog
// shell itself, injects vendored axe-core, and fails on serious or critical
// WCAG 2A/2AA violations.
func TestAccessibilityAuditDocsSiteShell(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseAxeSource := readVendoredAxeSource(parseT)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18256")

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

	parseBrowserHandle, parseLaunchErr := launchExamplesBrowser(parsePw, "chromium")
	if parseLaunchErr != nil {
		parseT.Fatalf("launch chromium: %v", parseLaunchErr)
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
	defer func() {
		if parseCloseErr := parsePage.Close(); parseCloseErr != nil {
			parseT.Errorf("close page: %v", parseCloseErr)
		}
	}()

	parseDocsURL := parseBaseURL + "/examples/public-examples-site/"
	if _, parseNavErr := parsePage.Goto(parseDocsURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	}); parseNavErr != nil {
		parseT.Fatalf("navigate to docs site shell %s: %v", parseDocsURL, parseNavErr)
	}
	if _, parseWaitErr := parsePage.WaitForFunction(a11yDocsSiteReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(90000),
	}); parseWaitErr != nil {
		parseT.Fatalf("docs site shell never booted: %v", parseWaitErr)
	}
	failOnAxeViolations(parseT, "docs-site shell", injectAndRunAxe(parseT, parsePage, parseAxeSource))
}

// TestAccessibilityAuditPublicExamples loads a representative subset of public
// examples that render interactive WASM UIs, injects vendored axe-core, and
// runs a WCAG 2A/2AA audit. Only serious or critical violations cause a failure.
// The test reports exactly what axe finds — violations are not suppressed.
func TestAccessibilityAuditPublicExamples(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// Read vendored axe-core once.
	parseAxeSource, parseAxeReadErr := os.ReadFile(filepath.Join(filepath.Dir(parseFile), "testdata", "axe.min.js"))
	if parseAxeReadErr != nil {
		parseT.Fatalf("read testdata/axe.min.js: %v", parseAxeReadErr)
	}

	// Start the catalog server on a unique port (18255).
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18255")

	// These slugs are source-only in the catalog; their compiled WASM apps are
	// served under the public-examples-site asset tree — same pattern as pwa_offline_test.go.
	// Each has an app.wasm + index.html in examples/public-examples-site/assets/examples/<slug>/.
	//
	// Expanded from 6 to 19 examples covering:
	//   - Interactive primitives: counter, toggle, text-input
	//   - State management:       use-state, use-reducer, use-effect, use-callback, context-api, state-atoms
	//   - Forms & inputs:         html-forms, form-accessibility, form, todo-advanced
	//   - Overlay / focus:        accessible-overlay, overlay-stack
	//   - Routing:                hash-router, browser-router, route-params, routed-accessibility
	//   - Semantic markup:        semantic-html, todo-basic
	//   - Navigation:             composite-navigation
	//   - Widgets:                calculator
	parseExamples := []struct {
		slug        string
		description string
	}{
		// --- originally in the 6-example set ---
		{"counter", "basic counter with increment/decrement"},
		{"todo-basic", "simple todo list with add/remove"},
		{"form-accessibility", "form with labelled inputs"},
		{"accessible-overlay", "modal overlay with focus trap"},
		{"semantic-html", "semantic HTML landmark elements"},
		{"routed-accessibility", "routed app with accessibility hooks"},

		// --- routing ---
		{"hash-router", "hash-based SPA router"},
		{"browser-router", "history-API-based SPA router"},
		{"route-params", "route parameters via URL segments"},

		// --- state management ---
		{"use-state", "UseState hook demo with counter and message state"},
		{"use-reducer", "UseReducer hook demo"},
		{"use-effect", "UseEffect hook demo"},
		{"use-callback", "UseCallback hook demo"},
		{"context-api", "context API for component-tree state sharing"},
		{"state-atoms", "atom-based global reactive state"},

		// --- forms & inputs ---
		{"html-forms", "native HTML form elements"},
		{"toggle", "boolean toggle switch"},
		{"text-input", "text input with live binding"},
		{"form", "structured form with labeled name and age inputs"},
		{"todo-advanced", "rich todo with labelled inputs and accessible selects"},

		// --- overlay ---
		{"overlay-stack", "stacked overlay panels"},

		// --- navigation & composite widgets ---
		{"composite-navigation", "roving tabindex tabs and accessible listbox"},
		{"calculator", "expression calculator with labeled theme and angle selects"},
	}

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

	parseBrowserHandle, parseLaunchErr := launchExamplesBrowser(parsePw, "chromium")
	if parseLaunchErr != nil {
		parseT.Fatalf("launch chromium: %v", parseLaunchErr)
	}
	defer func() {
		if parseCloseErr := parseBrowserHandle.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	// Aggregate violations per page so we can report the full picture.
	type parsePageResult struct {
		slug       string
		violations []parseAxeViolation
	}
	var parseAllResults []parsePageResult

	for _, parseEx := range parseExamples {
		parseEx := parseEx
		parseT.Run(parseEx.slug, func(parseT *testing.T) {
			// URL pattern from pwa_offline_test.go: preview wasm lives under
			// /examples/public-examples-site/assets/examples/<slug>/
			parseExampleURL := parseBaseURL + "/examples/public-examples-site/assets/examples/" + parseEx.slug + "/"

			parsePage, parsePageErr := parseBrowserHandle.NewPage()
			if parsePageErr != nil {
				parseT.Fatalf("new page: %v", parsePageErr)
			}
			defer func() {
				if parseCloseErr := parsePage.Close(); parseCloseErr != nil {
					parseT.Errorf("close page: %v", parseCloseErr)
				}
			}()

			// Suppress console errors — a11y audit is about the DOM.
			var parseConsoleErrors []string
			parsePage.On("console", func(parseMsg playwright.ConsoleMessage) {
				if parseMsg.Type() == "error" {
					parseConsoleErrors = append(parseConsoleErrors, parseMsg.Text())
				}
			})

			parseT.Logf("navigating to %s", parseExampleURL)
			if _, parseNavErr := parsePage.Goto(parseExampleURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateLoad,
			}); parseNavErr != nil {
				parseT.Fatalf("navigate to %s: %v", parseExampleURL, parseNavErr)
			}

			// Wait for WASM boot (#app gets children).
			if _, parseWaitErr := parsePage.WaitForFunction(a11yWasmReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
				Timeout: playwright.Float(90000),
			}); parseWaitErr != nil {
				parseT.Fatalf("WASM app never booted for %s: %v", parseEx.slug, parseWaitErr)
			}
			parseT.Logf("WASM app booted: %s (%s)", parseEx.slug, parseEx.description)

			// Log console errors but do not fail on them — audit is DOM-only.
			for _, parseCErr := range parseConsoleErrors {
				parseT.Logf("console error (non-fatal for a11y): %s", parseCErr)
			}

			// Inject axe and run the audit.
			parseViolations := injectAndRunAxe(parseT, parsePage, parseAxeSource)

			if len(parseViolations) > 0 {
				parseAllResults = append(parseAllResults, parsePageResult{slug: parseEx.slug, violations: parseViolations})
				parseT.Logf("VIOLATIONS in %s (%d serious/critical):", parseEx.slug, len(parseViolations))
				for _, parseV := range parseViolations {
					parseT.Logf("  [%s] %s — %s", parseV.Impact, parseV.ID, parseV.Help)
					parseT.Logf("    help URL: %s", parseV.HelpURL)
					for _, parseTgt := range parseV.Targets {
						parseT.Logf("    node target: %s", parseTgt)
					}
				}
			} else {
				parseT.Logf("CLEAN: no serious/critical violations in %s", parseEx.slug)
			}
		})
	}

	// Final summary — fail with full list if any page had serious/critical violations.
	if len(parseAllResults) > 0 {
		var parseSummaryLines []string
		parseSummaryLines = append(parseSummaryLines, fmt.Sprintf(
			"axe WCAG2A/2AA audit found serious/critical violations on %d page(s):", len(parseAllResults),
		))
		for _, parseRes := range parseAllResults {
			parseSummaryLines = append(parseSummaryLines, fmt.Sprintf("  page: %s (%d violation(s))", parseRes.slug, len(parseRes.violations)))
			for _, parseV := range parseRes.violations {
				parseSummaryLines = append(parseSummaryLines, fmt.Sprintf(
					"    [%s] rule=%s  help=%q  url=%s",
					parseV.Impact, parseV.ID, parseV.Help, parseV.HelpURL,
				))
				for _, parseTgt := range parseV.Targets {
					parseSummaryLines = append(parseSummaryLines, fmt.Sprintf("      node: %s", parseTgt))
				}
			}
		}
		parseT.Errorf("%s", strings.Join(parseSummaryLines, "\n"))
	} else {
		parseT.Log("All audited examples are clean (no serious/critical WCAG2A/2AA violations).")
	}
}

// TestAccessibilityAuditCatchesViolation is a self-test proving the audit harness
// actually works. It creates deliberately inaccessible markup (image with no alt,
// empty button, unlabelled input) and asserts axe reports at least one violation.
// A green run of TestAccessibilityAuditPublicExamples therefore means "clean",
// not "audit broken".
func TestAccessibilityAuditCatchesViolation(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)

	// Read vendored axe-core.
	parseAxeSource, parseAxeReadErr := os.ReadFile(filepath.Join(filepath.Dir(parseFile), "testdata", "axe.min.js"))
	if parseAxeReadErr != nil {
		parseT.Fatalf("read testdata/axe.min.js: %v", parseAxeReadErr)
	}

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

	parseBrowserHandle, parseLaunchErr := launchExamplesBrowser(parsePw, "chromium")
	if parseLaunchErr != nil {
		parseT.Fatalf("launch chromium: %v", parseLaunchErr)
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
	defer func() {
		if parseCloseErr := parsePage.Close(); parseCloseErr != nil {
			parseT.Errorf("close page: %v", parseCloseErr)
		}
	}()

	// Set deliberately inaccessible content:
	//   - <img src='x'> with no alt attribute  → image-alt violation
	//   - <button></button> with no text        → button-name violation
	//   - <input> with no label                 → label violation
	if parseSetErr := parsePage.SetContent(
		`<html><body><img src='x'><button></button><input></body></html>`,
	); parseSetErr != nil {
		parseT.Fatalf("SetContent: %v", parseSetErr)
	}

	// Inject axe-core.
	if _, parseAddErr := parsePage.AddScriptTag(playwright.PageAddScriptTagOptions{
		Content: playwright.String(string(parseAxeSource)),
	}); parseAddErr != nil {
		parseT.Fatalf("AddScriptTag (axe-core): %v", parseAddErr)
	}

	// Run axe — collect ALL violations (not just serious/critical) to be robust.
	parseRaw, parseEvalErr := parsePage.Evaluate(`async () => {
		const parseResults = await axe.run(document, {
			runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa'] }
		});
		return parseResults.violations.map(parseV => ({
			id:     parseV.id,
			impact: parseV.impact,
			help:   parseV.help,
		}));
	}`, nil)
	if parseEvalErr != nil {
		parseT.Fatalf("axe.run evaluate (self-test): %v", parseEvalErr)
	}

	parseRawSlice, parseOk := parseRaw.([]interface{})
	if !parseOk || len(parseRawSlice) == 0 {
		parseT.Fatalf("self-test FAILED: axe found NO violations on deliberately inaccessible markup (harness is broken)")
	}

	parseT.Logf("self-test: axe found %d violation(s) on bad markup (harness is working):", len(parseRawSlice))
	for _, parseItem := range parseRawSlice {
		if parseMap, parseMapOk := parseItem.(map[string]interface{}); parseMapOk {
			parseT.Logf("  [%s] %s — %s", parseMap["impact"], parseMap["id"], parseMap["help"])
		}
	}
	parseT.Log("TestAccessibilityAuditCatchesViolation PASS: harness correctly detects violations.")
}
