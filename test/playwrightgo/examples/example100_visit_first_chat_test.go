//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

type example100VisitFirstChatArtifact struct {
	GetLandingStatus          int
	GetPricingStatus          int
	GetSignupStatus           int
	HasLandingBootMount       bool
	HasPricingFAQ             bool
	HasSignupFields           bool
	HasAuthenticatedShellBoot bool
	HasFirstPromptVisible     bool
	HasStreamCompletion       bool
	HasCanonicalThreadRoute   bool
	GetThreadPath             string
	GetConsoleErrorCount      int
	GetPageErrorCount         int
	GetConsoleSampleText      string
	GetPageErrorSampleText    string
}

// formatExample100VisitFirstChatSummary formats one flow artifact for concise test logs.
func formatExample100VisitFirstChatSummary(parseArtifact example100VisitFirstChatArtifact) string {
	return fmt.Sprintf(
		"landing=%d pricing=%d signup=%d landing-boot=%t pricing-faq=%t signup-fields=%t shell-boot=%t first-prompt=%t stream-done=%t canonical-route=%t thread-path=%q console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetLandingStatus,
		parseArtifact.GetPricingStatus,
		parseArtifact.GetSignupStatus,
		parseArtifact.HasLandingBootMount,
		parseArtifact.HasPricingFAQ,
		parseArtifact.HasSignupFields,
		parseArtifact.HasAuthenticatedShellBoot,
		parseArtifact.HasFirstPromptVisible,
		parseArtifact.HasStreamCompletion,
		parseArtifact.HasCanonicalThreadRoute,
		parseArtifact.GetThreadPath,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100VisitFirstChatArtifact runs landing-to-first-chat in one browser flow.
func captureExample100VisitFirstChatArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100VisitFirstChatArtifact {
	parseT.Helper()
	parseArtifact := example100VisitFirstChatArtifact{}
	parseConsoleSamples := make([]string, 0, 140)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseMu.Lock()
		defer parseMu.Unlock()
		if len(parseConsoleSamples) < 140 {
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

	parseLandingResp, parseErr := parsePage.Goto(parseBaseURL+"/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto landing route: %v", parseErr)
	}
	if parseLandingResp == nil {
		parseT.Fatalf("landing route returned nil response")
	}
	parseArtifact.GetLandingStatus = parseLandingResp.Status()
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			const app = document.getElementById("app");
			const boot = document.getElementById("boot-shell");
			const mounted = !!app && ((app.children && app.children.length > 0) || String(app.textContent || "").trim().length > 0);
			return mounted && !boot;
		}`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for landing wasm mount + boot-shell removal: %v", parseErr)
	}
	parseArtifact.HasLandingBootMount = true

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
		parseT.Fatalf("wait for pricing faq section: %v", parseErr)
	}
	parseArtifact.HasPricingFAQ = true

	parseSignupResp, parseErr := parsePage.Goto(parseBaseURL+"/signup", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto signup route: %v", parseErr)
	}
	if parseSignupResp == nil {
		parseT.Fatalf("signup route returned nil response")
	}
	parseArtifact.GetSignupStatus = parseSignupResp.Status()
	if _, parseErr := parsePage.WaitForSelector("#auth-name-input"); parseErr != nil {
		parseT.Fatalf("wait for signup name input: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for signup email input: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-password-input"); parseErr != nil {
		parseT.Fatalf("wait for signup password input: %v", parseErr)
	}
	parseArtifact.HasSignupFields = true

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app before login handoff: %v", parseErr)
	}
	parseSignupModeValue, parseErr := parsePage.Evaluate(`() => !!document.getElementById("auth-name-input")`)
	if parseErr != nil {
		parseT.Fatalf("detect signup-mode auth shell: %v", parseErr)
	}
	if parseSignupModeValue == true {
		if parseErr := parsePage.Click(`a:has-text("Log in")`); parseErr != nil {
			parseT.Fatalf("switch signup shell to login mode: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("auth-name-input")`, nil); parseErr != nil {
			parseT.Fatalf("wait for login-mode auth shell after handoff: %v", parseErr)
		}
	}

	parsePrompt := fmt.Sprintf("visit-first-chat-%d", time.Now().UnixNano()%1_000_000)
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for auth email input before login: %v", parseErr)
	}
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
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
		parseT.Fatalf("wait for authenticated shell boot completion: %v", parseErr)
	}
	parseArtifact.HasAuthenticatedShellBoot = true

	if parseErr := parsePage.Click(`button:has-text("New chat")`); parseErr != nil {
		parseT.Fatalf("click new chat: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#chat-input", parsePrompt); parseErr != nil {
		parseT.Fatalf("fill first prompt: %v", parseErr)
	}
	if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
		parseT.Fatalf("click send for first prompt: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parsePrompt),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for first prompt visibility: %v", parseErr)
	}
	parseArtifact.HasFirstPromptVisible = true
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("streaming-assistant-bubble")`, nil); parseErr != nil {
		parseT.Fatalf("wait for first stream completion: %v", parseErr)
	}
	parseArtifact.HasStreamCompletion = true

	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(12000)},
	); parseErr != nil {
		if _, parseFallbackErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseFallbackErr != nil {
			parseT.Fatalf("fallback goto /app for conversation reopen: %v", parseFallbackErr)
		}
		parseConversationSelector := fmt.Sprintf(`#conversation-list button:has-text(%q)`, parsePrompt)
		if _, parseFallbackErr := parsePage.WaitForSelector(parseConversationSelector); parseFallbackErr != nil {
			parseT.Fatalf("wait for canonical /app/thread/:publicID route: %v (fallback row wait failed: %v)", parseErr, parseFallbackErr)
		}
		if parseFallbackErr := parsePage.Click(parseConversationSelector); parseFallbackErr != nil {
			parseT.Fatalf("fallback click to normalize thread route failed: %v", parseFallbackErr)
		}
		if _, parseFallbackErr := parsePage.WaitForFunction(
			`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
			nil,
		); parseFallbackErr != nil {
			parseT.Fatalf("wait for canonical /app/thread/:publicID route: %v (fallback normalization failed: %v)", parseErr, parseFallbackErr)
		}
	}
	parseArtifact.HasCanonicalThreadRoute = true
	parsePathValue, parseErr := parsePage.Evaluate(`() => window.location.pathname`)
	if parseErr == nil {
		parseArtifact.GetThreadPath = strings.TrimSpace(fmt.Sprintf("%v", parsePathValue))
	}

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100VisitToFirstChatRegression verifies one end-to-end visitor-to-first-chat browser flow.
func TestExample100VisitToFirstChatRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18106")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100VisitFirstChatArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 visit-to-first-chat: %s", formatExample100VisitFirstChatSummary(parseArtifact))

		if parseArtifact.GetLandingStatus >= 400 || !parseArtifact.HasLandingBootMount {
			parseT.Fatalf("landing step failed: %s", formatExample100VisitFirstChatSummary(parseArtifact))
		}
		if parseArtifact.GetPricingStatus >= 400 || !parseArtifact.HasPricingFAQ {
			parseT.Fatalf("pricing step failed: %s", formatExample100VisitFirstChatSummary(parseArtifact))
		}
		if parseArtifact.GetSignupStatus >= 400 || !parseArtifact.HasSignupFields {
			parseT.Fatalf("signup handoff step failed: %s", formatExample100VisitFirstChatSummary(parseArtifact))
		}
		if !parseArtifact.HasAuthenticatedShellBoot {
			parseT.Fatalf("auth + shell boot step failed: %s", formatExample100VisitFirstChatSummary(parseArtifact))
		}
		if !parseArtifact.HasFirstPromptVisible || !parseArtifact.HasStreamCompletion || !parseArtifact.HasCanonicalThreadRoute {
			parseT.Fatalf("first-chat send/stream/route step failed: %s", formatExample100VisitFirstChatSummary(parseArtifact))
		}
		if !strings.HasPrefix(parseArtifact.GetThreadPath, "/app/thread/") {
			parseT.Fatalf("canonical thread path missing after first chat: %s", formatExample100VisitFirstChatSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100VisitFirstChatSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100VisitFirstChatSummary(parseArtifact))
		}
	})
}
