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

type example100AuthCallbackRouteArtifact struct {
	HasInvalidStateScreen             bool
	HasExpiredNonceScreen             bool
	HasDeniedConsentScreen            bool
	HasCallbackErrorScreen            bool
	HasSuccessfulCallbackCompletion   bool
	HasSuccessfulHistoryBackForward   bool
	GetSuccessfulCompletionRouteState string
	GetConsoleErrorCount              int
	GetPageErrorCount                 int
	GetConsoleSampleText              string
	GetPageErrorSampleText            string
}

// formatExample100AuthCallbackRouteSummary formats one callback-route matrix artifact for concise logs.
func formatExample100AuthCallbackRouteSummary(parseArtifact example100AuthCallbackRouteArtifact) string {
	return fmt.Sprintf(
		"invalid-state=%t expired-nonce=%t denied-consent=%t callback-error=%t success=%t history=%t success-route=%q console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.HasInvalidStateScreen,
		parseArtifact.HasExpiredNonceScreen,
		parseArtifact.HasDeniedConsentScreen,
		parseArtifact.HasCallbackErrorScreen,
		parseArtifact.HasSuccessfulCallbackCompletion,
		parseArtifact.HasSuccessfulHistoryBackForward,
		parseArtifact.GetSuccessfulCompletionRouteState,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// parseWaitExample100CallbackAuthShell waits for one callback route to land on one sane auth-entry screen.
func parseWaitExample100CallbackAuthShell(parseT *testing.T, parsePage playwright.Page, parseLabel string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			title: document.title || "",
			body: ((document.body && document.body.innerText) || "").slice(0, 1400),
		})`)
		parseT.Fatalf("%s wait for callback auth shell: %v debug=%#v", parseLabel, parseErr, parseDebugValue)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
		parseT.Fatalf("%s wait for boot-shell removal: %v", parseLabel, parseErr)
	}
}

// parseCaptureExample100CallbackRouteState returns one path+query snapshot for callback route assertions.
func parseCaptureExample100CallbackRouteState(parseT *testing.T, parsePage playwright.Page, parseLabel string) string {
	parseT.Helper()
	parseRouteValue, parseErr := parsePage.Evaluate(`() => String(window.location.pathname || "") + String(window.location.search || "")`)
	if parseErr != nil {
		parseT.Fatalf("%s read callback route state: %v", parseLabel, parseErr)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", parseRouteValue))
}

// parseVisitExample100AuthCallbackRoute visits one callback route case and asserts the auth shell remains sane.
func parseVisitExample100AuthCallbackRoute(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseCaseURL string, parseLabel string) string {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+parseCaseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("%s goto callback route %q: %v", parseLabel, parseCaseURL, parseErr)
	}
	parseWaitExample100CallbackAuthShell(parseT, parsePage, parseLabel)
	return parseCaptureExample100CallbackRouteState(parseT, parsePage, parseLabel)
}

// captureExample100AuthCallbackRouteArtifact executes callback route matrix checks for invalid/expired/denied/error/success paths.
func captureExample100AuthCallbackRouteArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100AuthCallbackRouteArtifact {
	parseT.Helper()
	parseArtifact := example100AuthCallbackRouteArtifact{}
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

	parseInvalidStateRoute := parseVisitExample100AuthCallbackRoute(
		parseT,
		parsePage,
		parseBaseURL,
		"/login?auth_callback=google&callback_state=invalid&return_to=%2Fapp",
		"invalid-state",
	)
	parseArtifact.HasInvalidStateScreen = strings.Contains(parseInvalidStateRoute, "callback_state=invalid")

	parseExpiredNonceRoute := parseVisitExample100AuthCallbackRoute(
		parseT,
		parsePage,
		parseBaseURL,
		"/login?auth_callback=google&callback_nonce=expired&return_to=%2Fapp",
		"expired-nonce",
	)
	parseArtifact.HasExpiredNonceScreen = strings.Contains(parseExpiredNonceRoute, "callback_nonce=expired")

	parseDeniedConsentRoute := parseVisitExample100AuthCallbackRoute(
		parseT,
		parsePage,
		parseBaseURL,
		"/login?auth_callback=google&callback_error=access_denied&return_to=%2Fapp",
		"denied-consent",
	)
	parseArtifact.HasDeniedConsentScreen = strings.Contains(parseDeniedConsentRoute, "callback_error=access_denied")

	parseCallbackErrorRoute := parseVisitExample100AuthCallbackRoute(
		parseT,
		parsePage,
		parseBaseURL,
		"/login?auth_callback=oidc&callback_error=server_error&return_to=%2Fapp",
		"callback-error",
	)
	parseArtifact.HasCallbackErrorScreen = strings.Contains(parseCallbackErrorRoute, "callback_error=server_error")

	if _, parseErr := parsePage.Goto(parseBaseURL+"/login?auth_callback=google&callback_status=success&return_to=%2Fapp", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("successful callback goto /login: %v", parseErr)
	}
	parseWaitExample100CallbackAuthShell(parseT, parsePage, "success-callback")
	if parseErr := parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("success-callback fill email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", "password"); parseErr != nil {
		parseT.Fatalf("success-callback fill password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("success-callback submit login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("success-callback wait chat shell: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
		parseT.Fatalf("success-callback wait boot-shell removal after login: %v", parseErr)
	}
	parseArtifact.GetSuccessfulCompletionRouteState = parseCaptureExample100CallbackRouteState(parseT, parsePage, "success-callback route state")
	parseArtifact.HasSuccessfulCallbackCompletion = strings.HasPrefix(parseArtifact.GetSuccessfulCompletionRouteState, "/app")

	if _, parseErr := parsePage.GoBack(playwright.PageGoBackOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("success-callback history back: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !!document.querySelector("#auth-email-input") || !!document.querySelector("#chat-input")`, nil); parseErr != nil {
		parseT.Fatalf("success-callback history back wait for sane shell: %v", parseErr)
	}
	if _, parseErr := parsePage.GoForward(playwright.PageGoForwardOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("success-callback history forward: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("success-callback history forward wait chat shell: %v", parseErr)
	}
	parseArtifact.HasSuccessfulHistoryBackForward = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100AuthCallbackRouteMatrixRegression verifies callback-route failure/success handling lands on sane auth screens without broken history.
func TestExample100AuthCallbackRouteMatrixRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18122")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100AuthCallbackRouteArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 auth callback-route matrix: %s", formatExample100AuthCallbackRouteSummary(parseArtifact))

		if !parseArtifact.HasInvalidStateScreen {
			parseT.Fatalf("invalid-state callback route leg missing: %s", formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
		if !parseArtifact.HasExpiredNonceScreen {
			parseT.Fatalf("expired-nonce callback route leg missing: %s", formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
		if !parseArtifact.HasDeniedConsentScreen {
			parseT.Fatalf("denied-consent callback route leg missing: %s", formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
		if !parseArtifact.HasCallbackErrorScreen {
			parseT.Fatalf("callback-error callback route leg missing: %s", formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
		if !parseArtifact.HasSuccessfulCallbackCompletion {
			parseT.Fatalf("successful callback completion leg missing: %s", formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
		if !parseArtifact.HasSuccessfulHistoryBackForward {
			parseT.Fatalf("successful callback history leg missing: %s", formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100AuthCallbackRouteSummary(parseArtifact))
		}
	})
}
