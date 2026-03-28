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

type example100PricingRegressionArtifact struct {
	GetPricingStatus          int
	HasPricingHeroCopy        bool
	HasPlanCardsCopy          bool
	HasPricingFAQ             bool
	HasContactCTATargets      bool
	HasNoStalePricingLanguage bool
	GetConsoleErrorCount      int
	GetPageErrorCount         int
	GetConsoleSampleText      string
	GetPageErrorSampleText    string
}

// formatExample100PricingRegressionSummary formats one pricing regression artifact for concise logs.
func formatExample100PricingRegressionSummary(parseArtifact example100PricingRegressionArtifact) string {
	return fmt.Sprintf(
		"pricing-status=%d hero-copy=%t plan-copy=%t faq=%t contact-ctas=%t no-stale-language=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetPricingStatus,
		parseArtifact.HasPricingHeroCopy,
		parseArtifact.HasPlanCardsCopy,
		parseArtifact.HasPricingFAQ,
		parseArtifact.HasContactCTATargets,
		parseArtifact.HasNoStalePricingLanguage,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100PricingRegressionArtifact executes the pricing route regression checks in one browser session.
func captureExample100PricingRegressionArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100PricingRegressionArtifact {
	parseT.Helper()
	parseArtifact := example100PricingRegressionArtifact{}
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
			// Ignore one known transient boot abort when the test switches routes
			// before the initial "/" WASM fetch finishes.
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
	if parseBootResp == nil {
		parseT.Fatalf("bootstrap route returned nil response")
	}
	if _, parseErr = parsePage.Evaluate(`() => {
		window.localStorage.setItem("chat-wizard:locale", "en")
		return true
	}`); parseErr != nil {
		parseT.Fatalf("set pricing regression locale: %v", parseErr)
	}

	parsePricingResp, parseErr := parsePage.Goto(parseBaseURL+"/pricing", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto pricing route: %v", parseErr)
	}
	if parsePricingResp == nil {
		parseT.Fatalf("pricing route returned nil response")
	}
	parseArtifact.GetPricingStatus = parsePricingResp.Status()

	if _, parseErr = parsePage.WaitForSelector("#pricing-top"); parseErr != nil {
		parseT.Fatalf("wait for pricing shell mount: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForSelector(`a[href="#plans"]`); parseErr != nil {
		parseT.Fatalf("wait for pricing hero plans cta: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForSelector(`a[href="#compare"]`); parseErr != nil {
		parseT.Fatalf("wait for pricing hero compare cta: %v", parseErr)
	}
	parseArtifact.HasPricingHeroCopy = true

	if _, parseErr = parsePage.WaitForSelector("#plans"); parseErr != nil {
		parseT.Fatalf("wait for pricing plans section: %v", parseErr)
	}
	parsePlanCopyValue, parseErr := parsePage.Evaluate(`() => {
		const text = ((document.body && document.body.innerText) || "").toLowerCase();
		return {
			hasStarter: text.includes("starter"),
			hasTeam: text.includes("team"),
			hasEnterprise: text.includes("enterprise"),
			hasPlatformFee: text.includes("platform fee"),
			hasServicePremium: text.includes("service premium"),
			sample: text.slice(0, 2000),
		};
	}`)
	if parseErr != nil {
		parseT.Fatalf("evaluate pricing plan-card copy: %v", parseErr)
	}
	parsePlanCopyMap, parseOk := parsePlanCopyValue.(map[string]any)
	if !parseOk {
		parseT.Fatalf("plan-card copy evaluation returned non-map value: %#v", parsePlanCopyValue)
	}
	parseHasStarter, _ := parsePlanCopyMap["hasStarter"].(bool)
	parseHasTeam, _ := parsePlanCopyMap["hasTeam"].(bool)
	parseHasEnterprise, _ := parsePlanCopyMap["hasEnterprise"].(bool)
	parseHasPlatformFee, _ := parsePlanCopyMap["hasPlatformFee"].(bool)
	parseHasServicePremium, _ := parsePlanCopyMap["hasServicePremium"].(bool)
	parseTextSample, _ := parsePlanCopyMap["sample"].(string)
	if !parseHasStarter || !parseHasTeam || !parseHasEnterprise || !parseHasPlatformFee || !parseHasServicePremium {
		parseT.Fatalf(
			"pricing plan-card copy missing required terms starter=%t team=%t enterprise=%t platform_fee=%t service_premium=%t sample=%q",
			parseHasStarter,
			parseHasTeam,
			parseHasEnterprise,
			parseHasPlatformFee,
			parseHasServicePremium,
			parseTextSample,
		)
	}
	parseArtifact.HasPlanCardsCopy = true

	if _, parseErr = parsePage.WaitForSelector("#faq"); parseErr != nil {
		parseT.Fatalf("wait for faq section: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForFunction(`() => document.body && document.body.innerText.includes("Questions buyers ask first.")`, nil); parseErr != nil {
		parseT.Fatalf("wait for faq copy: %v", parseErr)
	}
	parseArtifact.HasPricingFAQ = true

	if _, parseErr = parsePage.WaitForSelector(`a[href="mailto:sales@relaydesk.com"]`); parseErr != nil {
		parseT.Fatalf("wait for sales contact cta: %v", parseErr)
	}
	if _, parseErr = parsePage.WaitForSelector(`a[href="mailto:hello@relaydesk.com"]`); parseErr != nil {
		parseT.Fatalf("wait for team contact cta: %v", parseErr)
	}
	parseArtifact.HasContactCTATargets = true

	parseStaleLanguageValue, parseErr := parsePage.Evaluate(`() => {
		const text = ((document.body && document.body.innerText) || "").toLowerCase();
		return /\bfree\b/.test(text) || text.includes("unlimited") || text.includes("no token caps");
	}`)
	if parseErr != nil {
		parseT.Fatalf("evaluate stale pricing language: %v", parseErr)
	}
	if parseHasStaleLanguage, parseOk := parseStaleLanguageValue.(bool); !parseOk {
		parseT.Fatalf("stale language evaluation returned non-bool value: %#v", parseStaleLanguageValue)
	} else if parseHasStaleLanguage {
		parseT.Fatalf("expected no stale pricing language on /pricing route")
	}
	parseArtifact.HasNoStalePricingLanguage = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100PricingRegression verifies /pricing route load, plan-card copy, FAQ, contact CTAs, and stale-language guardrails.
func TestExample100PricingRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18107")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100PricingRegressionArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 pricing regression: %s", formatExample100PricingRegressionSummary(parseArtifact))

		if parseArtifact.GetPricingStatus >= 400 || !parseArtifact.HasPricingHeroCopy || !parseArtifact.HasPlanCardsCopy || !parseArtifact.HasPricingFAQ || !parseArtifact.HasContactCTATargets || !parseArtifact.HasNoStalePricingLanguage {
			parseT.Fatalf("pricing regression failed: %s", formatExample100PricingRegressionSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100PricingRegressionSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100PricingRegressionSummary(parseArtifact))
		}
	})
}
