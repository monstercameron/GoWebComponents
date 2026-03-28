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

type example100RouteSmokeArtifact struct {
	GetPricingStatus       int
	GetAuthStatus          int
	GetSettingsStatus      int
	HasPricingFAQ          bool
	HasAuthInputs          bool
	HasSettingsProfile     bool
	GetConsoleErrorCount   int
	GetPageErrorCount      int
	GetConsoleSampleText   string
	GetPageErrorSampleText string
}

// formatExample100RouteSmokeSummary formats one route-smoke artifact for concise logs.
func formatExample100RouteSmokeSummary(parseArtifact example100RouteSmokeArtifact) string {
	return fmt.Sprintf(
		"pricing-status=%d auth-status=%d settings-status=%d pricing-faq=%t auth-inputs=%t settings-profile=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetPricingStatus,
		parseArtifact.GetAuthStatus,
		parseArtifact.GetSettingsStatus,
		parseArtifact.HasPricingFAQ,
		parseArtifact.HasAuthInputs,
		parseArtifact.HasSettingsProfile,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100RouteSmokeArtifact runs pricing/auth/dashboard route checks on one authenticated browser session.
func captureExample100RouteSmokeArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100RouteSmokeArtifact {
	parseT.Helper()
	parseArtifact := example100RouteSmokeArtifact{}
	parseConsoleSamples := make([]string, 0, 120)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseMu.Lock()
		defer parseMu.Unlock()
		if len(parseConsoleSamples) < 120 {
			parseConsoleSamples = append(parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
		}
		if parseMessage.Type() == "error" {
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

	parsePricingResp, parseErr := parsePage.Goto(parseBaseURL+"/pricing#faq", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto pricing route: %v", parseErr)
	}
	if parsePricingResp == nil {
		parseT.Fatalf("pricing route returned nil response")
	}
	parseArtifact.GetPricingStatus = parsePricingResp.Status()
	if _, parseErr := parsePage.WaitForFunction(`() => document.body && document.body.innerText.toLowerCase().includes("faq")`, nil); parseErr != nil {
		parseT.Fatalf("wait for pricing FAQ copy: %v", parseErr)
	}
	parseArtifact.HasPricingFAQ = true

	parseAuthResp, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto auth route: %v", parseErr)
	}
	if parseAuthResp == nil {
		parseT.Fatalf("auth route returned nil response")
	}
	parseArtifact.GetAuthStatus = parseAuthResp.Status()
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for auth email input: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-password-input"); parseErr != nil {
		parseT.Fatalf("wait for auth password input: %v", parseErr)
	}
	parseArtifact.HasAuthInputs = true

	if parseErr := parsePage.Fill("#auth-email-input", "demo@example.com"); parseErr != nil {
		parseT.Fatalf("fill auth email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", "password123"); parseErr != nil {
		parseT.Fatalf("fill auth password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit auth form: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after login: %v", parseErr)
	}

	parseSettingsResp, parseErr := parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-profile", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto settings dashboard route: %v", parseErr)
	}
	if parseSettingsResp == nil {
		parseT.Fatalf("settings route returned nil response")
	}
	parseArtifact.GetSettingsStatus = parseSettingsResp.Status()
	if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
		parseT.Fatalf("wait for settings profile section: %v", parseErr)
	}
	parseArtifact.HasSettingsProfile = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100RouteSmokePricingAuthDashboard verifies pricing, auth, and settings-dashboard routes in one browser smoke.
func TestExample100RouteSmokePricingAuthDashboard(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18105")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100RouteSmokeArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 route smoke: %s", formatExample100RouteSmokeSummary(parseArtifact))

		if parseArtifact.GetPricingStatus >= 400 || !parseArtifact.HasPricingFAQ {
			parseT.Fatalf("pricing smoke failed: %s", formatExample100RouteSmokeSummary(parseArtifact))
		}
		if parseArtifact.GetAuthStatus >= 400 || !parseArtifact.HasAuthInputs {
			parseT.Fatalf("auth smoke failed: %s", formatExample100RouteSmokeSummary(parseArtifact))
		}
		if parseArtifact.GetSettingsStatus >= 400 || !parseArtifact.HasSettingsProfile {
			parseT.Fatalf("settings dashboard smoke failed: %s", formatExample100RouteSmokeSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100RouteSmokeSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100RouteSmokeSummary(parseArtifact))
		}
	})
}
