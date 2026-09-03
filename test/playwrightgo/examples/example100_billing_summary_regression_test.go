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

type example100BillingSummaryRegressionArtifact struct {
	GetSettingsStatus      int
	HasBillingSection      bool
	HasBillingLabels       bool
	HasBillingFormula      bool
	HasBillingModelValues  bool
	GetPlatformFeeUSD      float64
	GetUsageCostUSD        float64
	GetServicePremiumUSD   float64
	GetTotalUSD            float64
	GetPremiumPercent      float64
	GetConsoleErrorCount   int
	GetPageErrorCount      int
	GetConsoleSampleText   string
	GetPageErrorSampleText string
}

// formatExample100BillingSummaryRegressionSummary formats one billing-summary regression artifact for concise logs.
func formatExample100BillingSummaryRegressionSummary(parseArtifact example100BillingSummaryRegressionArtifact) string {
	return fmt.Sprintf(
		"settings-status=%d section=%t labels=%t formula=%t model-values=%t platform=%.6f usage=%.6f premium=%.6f total=%.6f premium-pct=%.4f console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetSettingsStatus,
		parseArtifact.HasBillingSection,
		parseArtifact.HasBillingLabels,
		parseArtifact.HasBillingFormula,
		parseArtifact.HasBillingModelValues,
		parseArtifact.GetPlatformFeeUSD,
		parseArtifact.GetUsageCostUSD,
		parseArtifact.GetServicePremiumUSD,
		parseArtifact.GetTotalUSD,
		parseArtifact.GetPremiumPercent,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100BillingSummaryRegressionArtifact executes login and billing/settings formula checks in one browser session.
func captureExample100BillingSummaryRegressionArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100BillingSummaryRegressionArtifact {
	parseT.Helper()
	parseArtifact := example100BillingSummaryRegressionArtifact{}
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

	parseAppResp, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto /login for billing regression login: %v", parseErr)
	}
	if parseAppResp == nil {
		parseT.Fatalf("/login route returned nil response")
	}
	if parseAppResp.Status() >= 400 {
		parseT.Fatalf("/login route returned status %d", parseAppResp.Status())
	}
	if _, parseErr = parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for auth email input: %v", parseErr)
	}
	if parseErr = parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("fill auth email: %v", parseErr)
	}
	if parseErr = parsePage.Fill("#auth-password-input", "password"); parseErr != nil {
		parseT.Fatalf("fill auth password: %v", parseErr)
	}
	if parseErr = parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit auth form: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			title: document.title || "",
			body: ((document.body && document.body.innerText) || "").slice(0, 1400),
		})`)
		parseT.Fatalf("wait for chat input after login: %v debug=%#v", parseErr, parseDebugValue)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
		parseT.Fatalf("wait for boot shell removal after login: %v", parseErr)
	}

	parseSettingsResp, parseErr := parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-billing", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto billing settings route: %v", parseErr)
	}
	if parseSettingsResp == nil {
		parseT.Fatalf("billing settings route returned nil response")
	}
	parseArtifact.GetSettingsStatus = parseSettingsResp.Status()
	if _, parseErr = parsePage.WaitForSelector("#settings-billing"); parseErr != nil {
		parseT.Fatalf("wait for billing settings section: %v", parseErr)
	}
	parseArtifact.HasBillingSection = true

	parseBillingValue, parseErr := parsePage.Evaluate(`() => {
		const parseCurrency = (value) => {
			const normalized = String(value || "").replace(/[^0-9.-]/g, "");
			const parsed = Number(normalized);
			return Number.isFinite(parsed) ? parsed : 0;
		};
		const text = ((document.body && document.body.innerText) || "").toLowerCase();
		const hasLabels =
			text.includes("platform fee") &&
			text.includes("raw usage cost") &&
			text.includes("service premium") &&
			text.includes("total");
		const platform = parseCurrency(document.getElementById("billing-platform-fee-value")?.textContent);
		const usage = parseCurrency(document.getElementById("billing-usage-value")?.textContent);
		const premium = parseCurrency(document.getElementById("billing-premium-value")?.textContent);
		const total = parseCurrency(document.getElementById("billing-total-value")?.textContent);
		const premiumPct = Number(window.__relaydesk_usage_premium_percent || 0);
		const expectedPremium = usage * premiumPct / 100;
		const expectedTotal = platform + usage + premium;
		const close = (left, right) => Math.abs(left - right) <= 0.02;
		const hasModelValues = platform > 0 && premiumPct > 0 && total >= platform;
		return {
			hasLabels,
			platform,
			usage,
			premium,
			total,
			premiumPct,
			hasModelValues,
			hasFormula: close(premium, expectedPremium) && close(total, expectedTotal),
		};
	}`)
	if parseErr != nil {
		parseT.Fatalf("evaluate billing summary labels+formula: %v", parseErr)
	}
	parseBillingMap, parseOk := parseBillingValue.(map[string]any)
	if !parseOk {
		parseT.Fatalf("billing summary evaluation returned non-map value: %#v", parseBillingValue)
	}
	parseArtifact.HasBillingLabels, _ = parseBillingMap["hasLabels"].(bool)
	parseArtifact.HasBillingFormula, _ = parseBillingMap["hasFormula"].(bool)
	parseArtifact.HasBillingModelValues, _ = parseBillingMap["hasModelValues"].(bool)
	parseArtifact.GetPlatformFeeUSD, _ = parseBillingMap["platform"].(float64)
	parseArtifact.GetUsageCostUSD, _ = parseBillingMap["usage"].(float64)
	parseArtifact.GetServicePremiumUSD, _ = parseBillingMap["premium"].(float64)
	parseArtifact.GetTotalUSD, _ = parseBillingMap["total"].(float64)
	parseArtifact.GetPremiumPercent, _ = parseBillingMap["premiumPct"].(float64)

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100BillingSummaryRegression verifies customer-facing billing/settings labels and formula parity with the billing model.
func TestExample100BillingSummaryRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18108")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100BillingSummaryRegressionArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 billing summary regression: %s", formatExample100BillingSummaryRegressionSummary(parseArtifact))

		if parseArtifact.GetSettingsStatus >= 400 || !parseArtifact.HasBillingSection || !parseArtifact.HasBillingLabels || !parseArtifact.HasBillingFormula || !parseArtifact.HasBillingModelValues {
			parseT.Fatalf("billing summary regression failed: %s", formatExample100BillingSummaryRegressionSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100BillingSummaryRegressionSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100BillingSummaryRegressionSummary(parseArtifact))
		}
	})
}
