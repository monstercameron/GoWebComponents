//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

type example100BootCatalogRouteArtifact struct {
	GetPath                string
	GetStatus              int
	GetSelector            string
	GetRouteText           string
	HasRouteSelector       bool
	HasNoRawKeyPattern     bool
	HasNoTemplateMarkers   bool
	HasStableCopyAfterBoot bool
}

type example100BootCatalogRegressionArtifact struct {
	GetRoutes              []example100BootCatalogRouteArtifact
	GetConsoleErrorCount   int
	GetPageErrorCount      int
	GetConsoleSampleText   string
	GetPageErrorSampleText string
}

// parseNormalizeBootCatalogRouteText normalizes route text for stable copy comparisons.
func parseNormalizeBootCatalogRouteText(parseValue string) string {
	return strings.TrimSpace(strings.Join(strings.Fields(parseValue), " "))
}

// formatExample100BootCatalogRouteSummary formats one route-level first-paint artifact for concise logs.
func formatExample100BootCatalogRouteSummary(parseArtifact example100BootCatalogRouteArtifact) string {
	return fmt.Sprintf(
		"path=%q status=%d selector=%q has-selector=%t no-raw-keys=%t no-template-markers=%t stable-copy=%t route-text=%q",
		parseArtifact.GetPath,
		parseArtifact.GetStatus,
		parseArtifact.GetSelector,
		parseArtifact.HasRouteSelector,
		parseArtifact.HasNoRawKeyPattern,
		parseArtifact.HasNoTemplateMarkers,
		parseArtifact.HasStableCopyAfterBoot,
		parseArtifact.GetRouteText,
	)
}

// formatExample100BootCatalogRegressionSummary formats one full first-paint artifact for concise logs.
func formatExample100BootCatalogRegressionSummary(parseArtifact example100BootCatalogRegressionArtifact) string {
	parseRouteSummaries := make([]string, 0, len(parseArtifact.GetRoutes))
	for _, parseRouteArtifact := range parseArtifact.GetRoutes {
		parseRouteSummaries = append(parseRouteSummaries, formatExample100BootCatalogRouteSummary(parseRouteArtifact))
	}
	return fmt.Sprintf(
		"routes=[%s] console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		strings.Join(parseRouteSummaries, " || "),
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100BootCatalogRegressionArtifact executes multi-route first-paint copy checks for boot-injected catalogs.
func captureExample100BootCatalogRegressionArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100BootCatalogRegressionArtifact {
	parseT.Helper()
	parseArtifact := example100BootCatalogRegressionArtifact{}
	parseConsoleSamples := make([]string, 0, 160)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseMu.Lock()
		defer parseMu.Unlock()
		if len(parseConsoleSamples) < 160 {
			parseConsoleSamples = append(parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
		}
		if parseMessage.Type() == "error" {
			if strings.Contains(parseText, "WASM failed to load: TypeError: WebAssembly compilation aborted: Network error: Response body loading was aborted") {
				return
			}
			parseArtifact.GetConsoleErrorCount++
		}
	})
	parsePage.OnPageError(func(parseErr error) {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseArtifact.GetPageErrorCount++
		if parseErr != nil && len(parsePageErrorSamples) < 12 {
			parsePageErrorSamples = append(parsePageErrorSamples, strings.TrimSpace(parseErr.Error()))
		}
	})

	parseBootResp, parseErr := parsePage.Goto(parseBaseURL+"/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto bootstrap route for locale setup: %v", parseErr)
	}
	if parseBootResp == nil || parseBootResp.Status() >= 400 {
		parseT.Fatalf("bootstrap route status invalid: %#v", parseBootResp)
	}
	if _, parseErr = parsePage.Evaluate(`() => {
		window.localStorage.setItem("chat-wizard:locale", "en")
		return true
	}`); parseErr != nil {
		parseT.Fatalf("set boot-catalog regression locale: %v", parseErr)
	}

	parseRouteSpecs := []example100BootCatalogRouteArtifact{
		{GetPath: "/home", GetSelector: "#demo"},
		{GetPath: "/pricing", GetSelector: "#pricing-top"},
		{GetPath: "/signup", GetSelector: "main h1"},
		{GetPath: "/", GetSelector: "#demo"},
	}
	parseRouteArtifacts := make([]example100BootCatalogRouteArtifact, 0, len(parseRouteSpecs))
	for _, parseRouteSpec := range parseRouteSpecs {
		parseRouteArtifact := parseRouteSpec
		parseRouteResp, parseErr := parsePage.Goto(parseBaseURL+parseRouteArtifact.GetPath, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		})
		if parseErr != nil {
			parseT.Fatalf("goto %s route: %v", parseRouteArtifact.GetPath, parseErr)
		}
		if parseRouteResp == nil {
			parseT.Fatalf("%s route returned nil response", parseRouteArtifact.GetPath)
		}
		parseRouteArtifact.GetStatus = parseRouteResp.Status()
		if _, parseErr = parsePage.WaitForSelector(parseRouteArtifact.GetSelector); parseErr != nil {
			parseT.Fatalf("wait for selector %q on route %s: %v", parseRouteArtifact.GetSelector, parseRouteArtifact.GetPath, parseErr)
		}
		parseRouteArtifact.HasRouteSelector = true
		if _, parseErr = parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
			parseT.Fatalf("wait for boot shell removal on route %s: %v", parseRouteArtifact.GetPath, parseErr)
		}
		parseRouteEvalValue, parseErr := parsePage.Evaluate(`(selector) => {
			const body = (document.body && document.body.innerText) || "";
			const routeNode = document.querySelector(String(selector || ""));
			const routeText = String(routeNode ? routeNode.textContent || "" : "").replace(/\s+/g, " ").trim();
			const hasRawKeyPattern = /\b(?:marketing|auth|chat)\.[a-z0-9][a-z0-9_.-]*\b/i.test(body);
			const hasTemplateMarkers = body.includes("{{") || body.includes("}}");
			return {
				routeText,
				hasRawKeyPattern,
				hasTemplateMarkers,
			};
		}`, parseRouteArtifact.GetSelector)
		if parseErr != nil {
			parseT.Fatalf("evaluate route %s first-paint copy checks: %v", parseRouteArtifact.GetPath, parseErr)
		}
		parseRouteEvalMap, parseOk := parseRouteEvalValue.(map[string]any)
		if !parseOk {
			parseT.Fatalf("route %s first-paint evaluation returned non-map value: %#v", parseRouteArtifact.GetPath, parseRouteEvalValue)
		}
		parseRouteArtifact.GetRouteText, _ = parseRouteEvalMap["routeText"].(string)
		parseHasRawKeyPattern, _ := parseRouteEvalMap["hasRawKeyPattern"].(bool)
		parseHasTemplateMarkers, _ := parseRouteEvalMap["hasTemplateMarkers"].(bool)
		parseRouteArtifact.HasNoRawKeyPattern = !parseHasRawKeyPattern
		parseRouteArtifact.HasNoTemplateMarkers = !parseHasTemplateMarkers

		parseFirstCopy := parseNormalizeBootCatalogRouteText(parseRouteArtifact.GetRouteText)
		parsePage.WaitForTimeout(400)
		parseSecondCopyValue, parseErr := parsePage.Evaluate(`(selector) => {
			const routeNode = document.querySelector(String(selector || ""));
			return String(routeNode ? routeNode.textContent || "" : "").replace(/\s+/g, " ").trim();
		}`, parseRouteArtifact.GetSelector)
		if parseErr != nil {
			parseT.Fatalf("evaluate route %s post-hydration copy: %v", parseRouteArtifact.GetPath, parseErr)
		}
		parseSecondCopy := parseNormalizeBootCatalogRouteText(fmt.Sprintf("%v", parseSecondCopyValue))
		parseRouteArtifact.HasStableCopyAfterBoot = parseFirstCopy != "" && parseFirstCopy == parseSecondCopy
		parseRouteArtifacts = append(parseRouteArtifacts, parseRouteArtifact)
	}
	parseArtifact.GetRoutes = parseRouteArtifacts

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100BootCatalogFirstPaintRegression verifies boot-injected marketing/auth catalogs on first paint across key public routes.
func TestExample100BootCatalogFirstPaintRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18109")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100BootCatalogRegressionArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 boot catalog first-paint regression: %s", formatExample100BootCatalogRegressionSummary(parseArtifact))

		for _, parseRouteArtifact := range parseArtifact.GetRoutes {
			if parseRouteArtifact.GetStatus >= 400 || !parseRouteArtifact.HasRouteSelector || !parseRouteArtifact.HasNoRawKeyPattern || !parseRouteArtifact.HasNoTemplateMarkers || !parseRouteArtifact.HasStableCopyAfterBoot {
				parseT.Fatalf("boot catalog first-paint route failed: %s", formatExample100BootCatalogRouteSummary(parseRouteArtifact))
			}
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100BootCatalogRegressionSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100BootCatalogRegressionSummary(parseArtifact))
		}
	})
}
