//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

// parseDashboardRouterInt converts Playwright-evaluated numeric values into one Go int for stable assertions.
func parseDashboardRouterInt(parseT *testing.T, parseValue interface{}, parseLabel string) int {
	parseT.Helper()
	switch parseTyped := parseValue.(type) {
	case int:
		return parseTyped
	case int32:
		return int(parseTyped)
	case int64:
		return int(parseTyped)
	case float64:
		return int(parseTyped)
	default:
		parseT.Fatalf("%s returned unsupported numeric type %T (%v)", parseLabel, parseValue, parseValue)
		return 0
	}
}

// setExample100DashboardAuthCookie mirrors the browser auth token into the deep-link cookie used by direct dashboard route guards.
func setExample100DashboardAuthCookie(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) {
	parseT.Helper()
	parseTokenValue, parseErr := parsePage.Evaluate(fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey))
	if parseErr != nil {
		parseT.Fatalf("read auth token for dashboard cookie sync: %v", parseErr)
	}
	parseAuthToken := strings.TrimSpace(fmt.Sprintf("%v", parseTokenValue))
	if strings.EqualFold(parseAuthToken, "<nil>") {
		parseAuthToken = ""
	}
	if parseAuthToken == "" {
		parseT.Fatal("dashboard router regression auth token missing from localStorage")
	}
	if parseErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{
		{
			Name:  example100AdminJourneyAuthCookieName,
			Value: parseAuthToken,
			URL:   playwright.String(parseBaseURL),
		},
	}); parseErr != nil {
		parseT.Fatalf("set dashboard auth cookie: %v", parseErr)
	}
}

// assertExample100DashboardTileRoute verifies that one dashboard home tile renders as an anchor and navigates without resetting SPA context.
func assertExample100DashboardTileRoute(parseT *testing.T, parsePage playwright.Page, parseTileLabel string, parseRoute string) {
	parseT.Helper()
	// Tiles are now <a href=route> elements — select by route so the assertion is structurally exact.
	parseTileSelector := fmt.Sprintf(`a[href=%q]`, parseRoute)
	parseTile, parseErr := parsePage.WaitForSelector(parseTileSelector)
	if parseErr != nil {
		parseT.Fatalf("wait for dashboard tile anchor %q (route=%s): %v", parseTileLabel, parseRoute, parseErr)
	}
	parseTagNameValue, parseErr := parseTile.Evaluate(`parseNode => parseNode.tagName`)
	if parseErr != nil {
		parseT.Fatalf("read dashboard tile tag for %q: %v", parseTileLabel, parseErr)
	}
	if !strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", parseTagNameValue)), "A") {
		parseT.Fatalf("dashboard tile %q rendered as %v, want A", parseTileLabel, parseTagNameValue)
	}
	// Seed an in-memory marker before clicking to detect any context-destroying full page reload.
	parseMarkerValue, parseErr := parsePage.Evaluate(`() => {
		window.__example100DashboardRouterMarker = (window.__example100DashboardRouterMarker || 0) + 1;
		return window.__example100DashboardRouterMarker;
	}`)
	if parseErr != nil {
		parseT.Fatalf("seed dashboard router marker before %q click: %v", parseTileLabel, parseErr)
	}
	parseMarker := parseDashboardRouterInt(parseT, parseMarkerValue, "dashboard marker before click")
	if parseErr := parseTile.Click(); parseErr != nil {
		parseT.Fatalf("click dashboard tile %q: %v", parseTileLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => window.location.pathname === %q`, parseRoute),
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("wait for dashboard tile %q route %s: %v", parseTileLabel, parseRoute, parseErr)
	}
	// Marker must survive the navigation; a reset means the JS context was torn down (full page reload).
	parseMarkerAfterValue, parseErr := parsePage.Evaluate(`() => window.__example100DashboardRouterMarker || 0`)
	if parseErr != nil {
		parseT.Fatalf("read dashboard router marker after %q click: %v", parseTileLabel, parseErr)
	}
	if parseMarkerAfter := parseDashboardRouterInt(parseT, parseMarkerAfterValue, "dashboard marker after click"); parseMarkerAfter != parseMarker {
		parseT.Fatalf("dashboard tile %q click reset JS context (full page reload): marker before=%d after=%d", parseTileLabel, parseMarker, parseMarkerAfter)
	}
	// Back link in the slice top bar is now also an <a> pointing to /app/dashboard.
	parseBackSelector := `a[href="/app/dashboard"]`
	parseBackLink, parseErr := parsePage.WaitForSelector(parseBackSelector)
	if parseErr != nil {
		parseT.Fatalf("wait for dashboard slice back anchor after %q click: %v", parseTileLabel, parseErr)
	}
	if parseErr := parseBackLink.Click(); parseErr != nil {
		parseT.Fatalf("click dashboard slice back anchor after %q click: %v", parseTileLabel, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname === "/app/dashboard"`,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("wait for dashboard home after %q slice return: %v", parseTileLabel, parseErr)
	}
}

// TestExample100DashboardRouterRegression verifies that dashboard navigation links and slice tiles stay inside the SPA router.
// It asserts: tiles render as anchor elements, SPA JS context is preserved across navigation (no full page reload),
// the "Back to workspace" top-bar link returns to /app without reloading, and the auth token survives the full journey.
func TestExample100DashboardRouterRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100AdminJourneyServer(parseT, parseRepoRoot, "18108")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseLoginExample100AuthEntryUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)
		setExample100DashboardAuthCookie(parseT, parsePage, parseBaseURL)
		if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto /app/dashboard for router regression: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(
			`() => window.location.pathname === "/app/dashboard"`,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
		); parseErr != nil {
			parseT.Fatalf("wait for /app/dashboard before tile assertions: %v", parseErr)
		}

		// Exercise each ready slice tile and its back-link, verifying no JS-context teardown per round trip.
		assertExample100DashboardTileRoute(parseT, parsePage, "Business", "/app/dashboard/business")
		assertExample100DashboardTileRoute(parseT, parsePage, "Customers", "/app/dashboard/customers")
		assertExample100DashboardTileRoute(parseT, parsePage, "Providers", "/app/dashboard/providers")
		assertExample100DashboardTileRoute(parseT, parsePage, "Ops", "/app/dashboard/ops")

		// Verify the "Back to workspace" anchor on the dashboard home is an <a> pointing to /app.
		parseBackWorkspaceSelector := `a[href="/app"]`
		parseBackWorkspace, parseErr := parsePage.WaitForSelector(parseBackWorkspaceSelector)
		if parseErr != nil {
			parseT.Fatalf("wait for 'Back to workspace' anchor: %v", parseErr)
		}
		parseTagValue, parseErr := parseBackWorkspace.Evaluate(`parseNode => parseNode.tagName`)
		if parseErr != nil {
			parseT.Fatalf("read 'Back to workspace' tag: %v", parseErr)
		}
		if !strings.EqualFold(strings.TrimSpace(fmt.Sprintf("%v", parseTagValue)), "A") {
			parseT.Fatalf("'Back to workspace' rendered as %v, want A", parseTagValue)
		}

		// Seed a marker to detect any full page reload caused by the workspace back-link click.
		parseMarkerValue, parseErr := parsePage.Evaluate(`() => {
			window.__example100DashboardRouterMarker = (window.__example100DashboardRouterMarker || 0) + 100;
			return window.__example100DashboardRouterMarker;
		}`)
		if parseErr != nil {
			parseT.Fatalf("seed workspace back-link marker: %v", parseErr)
		}
		parseMarker := parseDashboardRouterInt(parseT, parseMarkerValue, "workspace back-link marker before click")

		if parseErr := parseBackWorkspace.Click(); parseErr != nil {
			parseT.Fatalf("click 'Back to workspace' anchor: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(
			`() => window.location.pathname === "/app"`,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
		); parseErr != nil {
			parseT.Fatalf("wait for /app after 'Back to workspace' click: %v", parseErr)
		}
		parseMarkerAfterValue, parseErr := parsePage.Evaluate(`() => window.__example100DashboardRouterMarker || 0`)
		if parseErr != nil {
			parseT.Fatalf("read workspace back-link marker after click: %v", parseErr)
		}
		if parseMarkerAfter := parseDashboardRouterInt(parseT, parseMarkerAfterValue, "workspace back-link marker after click"); parseMarkerAfter != parseMarker {
			parseT.Fatalf("'Back to workspace' click reset JS context (full page reload): marker before=%d after=%d", parseMarker, parseMarkerAfter)
		}

		// Auth token must still be present in localStorage — confirms authenticated shell state was not torn down.
		parseTokenValue, parseErr := parsePage.Evaluate(fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey))
		if parseErr != nil {
			parseT.Fatalf("read auth token after dashboard navigation: %v", parseErr)
		}
		parseAuthToken := strings.TrimSpace(fmt.Sprintf("%v", parseTokenValue))
		if strings.EqualFold(parseAuthToken, "<nil>") {
			parseAuthToken = ""
		}
		if parseAuthToken == "" {
			parseT.Fatal("auth token missing from localStorage after dashboard navigation: authenticated shell state was torn down")
		}
	})
}
