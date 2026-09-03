//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

type example100PublicRouteRefreshCheck struct {
	GetRoute        string
	GetWantSelector string
	GetWantText     string
}

// parseAssertExample100PublicRouteRefresh verifies one public route survives direct load, reload, and history navigation.
func parseAssertExample100PublicRouteRefresh(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseCheck example100PublicRouteRefreshCheck) {
	parseT.Helper()
	parseResp, parseErr := parsePage.Goto(parseBaseURL+parseCheck.GetRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto %s: %v", parseCheck.GetRoute, parseErr)
	}
	if parseResp == nil {
		parseT.Fatalf("nil response for %s", parseCheck.GetRoute)
	}
	if parseResp.Status() >= 400 {
		parseT.Fatalf("direct load %s returned status %d", parseCheck.GetRoute, parseResp.Status())
	}
	parseWaitRouteState := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseCheck.GetRoute)
	if _, parseErr := parsePage.WaitForFunction(parseWaitRouteState, nil); parseErr != nil {
		parseT.Fatalf("wait for route state %s: %v", parseCheck.GetRoute, parseErr)
	}
	if strings.TrimSpace(parseCheck.GetWantSelector) != "" {
		if _, parseErr := parsePage.WaitForSelector(parseCheck.GetWantSelector); parseErr != nil {
			parseT.Fatalf("wait for selector %s on %s: %v", parseCheck.GetWantSelector, parseCheck.GetRoute, parseErr)
		}
	}
	if strings.TrimSpace(parseCheck.GetWantText) != "" {
		parseWaitText := fmt.Sprintf(`() => {
			const text = ((document.body && document.body.innerText) || "").toLowerCase();
			return text.includes(%q);
		}`, strings.ToLower(strings.TrimSpace(parseCheck.GetWantText)))
		if _, parseErr := parsePage.WaitForFunction(parseWaitText, nil); parseErr != nil {
			parseT.Fatalf("wait for copy %q on %s: %v", parseCheck.GetWantText, parseCheck.GetRoute, parseErr)
		}
	}
	if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("reload %s: %v", parseCheck.GetRoute, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(parseWaitRouteState, nil); parseErr != nil {
		parseT.Fatalf("wait for route state after reload %s: %v", parseCheck.GetRoute, parseErr)
	}
	if strings.TrimSpace(parseCheck.GetWantSelector) != "" {
		if _, parseErr := parsePage.WaitForSelector(parseCheck.GetWantSelector); parseErr != nil {
			parseT.Fatalf("wait for selector after reload %s on %s: %v", parseCheck.GetWantSelector, parseCheck.GetRoute, parseErr)
		}
	}
}

// TestExample100PublicRouteDirectRefreshRegression verifies direct refresh and history navigation on remaining public routes.
func TestExample100PublicRouteDirectRefreshRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18123")

	parseChecks := []example100PublicRouteRefreshCheck{
		{GetRoute: "/plans", GetWantSelector: "#pricing-top", GetWantText: "starter"},
		{GetRoute: "/about", GetWantSelector: "h1", GetWantText: "about"},
		{GetRoute: "/contact", GetWantSelector: "h1", GetWantText: "contact"},
		{GetRoute: "/privacy", GetWantSelector: "h1", GetWantText: "privacy"},
		{GetRoute: "/terms", GetWantSelector: "h1", GetWantText: "terms"},
		{GetRoute: "/security", GetWantSelector: "h1", GetWantText: "security"},
		{GetRoute: "/status", GetWantSelector: "h1", GetWantText: "status"},
	}

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseConsoleErrorCount := 0
		parsePageErrorCount := 0
		var parseMu sync.Mutex
		parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
			if strings.TrimSpace(parseMessage.Text()) == "" || parseMessage.Type() != "error" {
				return
			}
			parseMu.Lock()
			parseConsoleErrorCount++
			parseMu.Unlock()
		})
		parsePage.OnPageError(func(parseErr error) {
			if parseErr == nil {
				return
			}
			parseMu.Lock()
			parsePageErrorCount++
			parseMu.Unlock()
		})

		for _, parseCheck := range parseChecks {
			parseAssertExample100PublicRouteRefresh(parseT, parsePage, parseBaseURL, parseCheck)
		}

		for parseIndex := len(parseChecks) - 2; parseIndex >= 0; parseIndex-- {
			if _, parseErr := parsePage.GoBack(playwright.PageGoBackOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
				parseT.Fatalf("go back to %s: %v", parseChecks[parseIndex].GetRoute, parseErr)
			}
			parseWaitRouteState := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseChecks[parseIndex].GetRoute)
			if _, parseErr := parsePage.WaitForFunction(parseWaitRouteState, nil); parseErr != nil {
				parseT.Fatalf("wait for back route state %s: %v", parseChecks[parseIndex].GetRoute, parseErr)
			}
		}
		for parseIndex := 1; parseIndex < len(parseChecks); parseIndex++ {
			if _, parseErr := parsePage.GoForward(playwright.PageGoForwardOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
				parseT.Fatalf("go forward to %s: %v", parseChecks[parseIndex].GetRoute, parseErr)
			}
			parseWaitRouteState := fmt.Sprintf(`() => (window.location.pathname + window.location.search) === %q`, parseChecks[parseIndex].GetRoute)
			if _, parseErr := parsePage.WaitForFunction(parseWaitRouteState, nil); parseErr != nil {
				parseT.Fatalf("wait for forward route state %s: %v", parseChecks[parseIndex].GetRoute, parseErr)
			}
		}

		parseMu.Lock()
		parseFinalConsoleErrors := parseConsoleErrorCount
		parseFinalPageErrors := parsePageErrorCount
		parseMu.Unlock()
		if parseFinalConsoleErrors != 0 {
			parseT.Fatalf("unexpected console errors on public route refresh regression: %d", parseFinalConsoleErrors)
		}
		if parseFinalPageErrors != 0 {
			parseT.Fatalf("unexpected page errors on public route refresh regression: %d", parseFinalPageErrors)
		}
	})
}
