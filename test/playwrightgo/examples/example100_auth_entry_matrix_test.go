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

type example100AuthEntryMatrixArtifact struct {
	HasSignupFlow               bool
	HasLoginFlow                bool
	HasRefreshFlow              bool
	HasLogoutFlow               bool
	HasPasswordResetFlow        bool
	HasMixedProviderEntryFlow   bool
	HasWorkspaceSSOEntryFlow    bool
	HasCallbackReturnEntryFlow  bool
	HasFirstChatFlow            bool
	GetFirstChatThreadPath      string
	GetConsoleErrorCount        int
	GetPageErrorCount           int
	GetConsoleSampleText        string
	GetPageErrorSampleText      string
	GetMixedProviderRouteState  string
	GetCallbackReturnRouteState string
	GetWorkspaceSSORouteState   string
}

// formatExample100AuthEntryMatrixSummary formats one auth-entry regression artifact for concise logs.
func formatExample100AuthEntryMatrixSummary(parseArtifact example100AuthEntryMatrixArtifact) string {
	return fmt.Sprintf(
		"signup=%t login=%t refresh=%t logout=%t reset=%t mixed-provider=%t workspace-sso-entry=%t callback-return=%t first-chat=%t thread-path=%q mixed-route=%q sso-route=%q callback-route=%q console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.HasSignupFlow,
		parseArtifact.HasLoginFlow,
		parseArtifact.HasRefreshFlow,
		parseArtifact.HasLogoutFlow,
		parseArtifact.HasPasswordResetFlow,
		parseArtifact.HasMixedProviderEntryFlow,
		parseArtifact.HasWorkspaceSSOEntryFlow,
		parseArtifact.HasCallbackReturnEntryFlow,
		parseArtifact.HasFirstChatFlow,
		parseArtifact.GetFirstChatThreadPath,
		parseArtifact.GetMixedProviderRouteState,
		parseArtifact.GetWorkspaceSSORouteState,
		parseArtifact.GetCallbackReturnRouteState,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// parseWaitForAuthEmailInput waits for the local password auth-entry shell to render.
func parseWaitForAuthEmailInput(parseT *testing.T, parsePage playwright.Page, parseLabel string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			title: document.title || "",
			body: ((document.body && document.body.innerText) || "").slice(0, 1400),
		})`)
		parseT.Fatalf("%s wait for auth email input: %v debug=%#v", parseLabel, parseErr, parseDebugValue)
	}
}

// parseWaitForSignedOutEntry waits for one signed-out auth-entry state (auth form or public landing) after logout.
func parseWaitForSignedOutEntry(parseT *testing.T, parsePage playwright.Page, parseLabel string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			const path = window.location.pathname;
			const hasAuthInput = !!document.querySelector("#auth-email-input");
			const hasChatInput = !!document.querySelector("#chat-input");
			const isPublicRoute = path === "/" || path === "/home" || path === "/pricing" || path === "/signup";
			return hasAuthInput || (isPublicRoute && !hasChatInput);
		}`,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(45000)},
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			title: document.title || "",
			hasAuthInput: !!document.querySelector("#auth-email-input"),
			hasChatInput: !!document.querySelector("#chat-input"),
			body: ((document.body && document.body.innerText) || "").slice(0, 1400),
		})`)
		parseT.Fatalf("%s wait for signed-out auth-entry state: %v debug=%#v", parseLabel, parseErr, parseDebugValue)
	}
}

// parseLoginExample100AuthEntryUser signs in one local email/password user and waits for chat-shell readiness.
func parseLoginExample100AuthEntryUser(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseEmail string, parsePassword string) {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login (%s): %v", parseEmail, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => !!document.querySelector("#auth-email-input") || !!document.querySelector("#chat-input")`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for login-or-chat shell (%s): %v", parseEmail, parseErr)
	}
	parseHasChatInputValue, parseErr := parsePage.Evaluate(`() => !!document.querySelector("#chat-input")`)
	if parseErr != nil {
		parseT.Fatalf("evaluate authenticated shell presence (%s): %v", parseEmail, parseErr)
	}
	if parseHasChatInput, _ := parseHasChatInputValue.(bool); parseHasChatInput {
		return
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "login")
	if parseErr := parsePage.Fill("#auth-email-input", parseEmail); parseErr != nil {
		parseT.Fatalf("fill auth email (%s): %v", parseEmail, parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", parsePassword); parseErr != nil {
		parseT.Fatalf("fill auth password (%s): %v", parseEmail, parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit auth form (%s): %v", parseEmail, parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after login (%s): %v", parseEmail, parseErr)
	}
}

// parseCaptureExample100AuthEntryRouteState returns the current path+query for route-state matrix assertions.
func parseCaptureExample100AuthEntryRouteState(parseT *testing.T, parsePage playwright.Page, parseLabel string) string {
	parseT.Helper()
	parseRouteValue, parseErr := parsePage.Evaluate(`() => String(window.location.pathname || "") + String(window.location.search || "")`)
	if parseErr != nil {
		parseT.Fatalf("%s read route state: %v", parseLabel, parseErr)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", parseRouteValue))
}

// parseLogoutExample100AuthEntryUser signs out through settings and verifies the auth-entry shell is restored.
func parseLogoutExample100AuthEntryUser(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-profile", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app/settings for logout: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
		parseT.Fatalf("wait for settings profile before logout: %v", parseErr)
	}
	parseSignOutButton, parseErr := parsePage.WaitForSelector(`button:has-text("Sign out")`)
	if parseErr != nil {
		parseT.Fatalf("wait for settings sign-out button: %v", parseErr)
	}
	if parseErr := parseSignOutButton.ScrollIntoViewIfNeeded(); parseErr != nil {
		parseT.Fatalf("scroll settings sign-out button: %v", parseErr)
	}
	if parseErr := parseSignOutButton.Click(); parseErr != nil {
		parseT.Fatalf("click settings sign-out button: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !window.localStorage.getItem("chat-wizard:auth-token")`, nil); parseErr != nil {
		parseT.Fatalf("wait for auth token clear after sign-out: %v", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app after sign-out: %v", parseErr)
	}
	parseWaitForSignedOutEntry(parseT, parsePage, "logout")
}

// captureExample100AuthEntryMatrixArtifact executes one local-auth matrix for signup/login/refresh/logout/reset/first-chat coverage.
func captureExample100AuthEntryMatrixArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100AuthEntryMatrixArtifact {
	parseT.Helper()
	parseArtifact := example100AuthEntryMatrixArtifact{}
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

	parseSignupEmail := fmt.Sprintf("auth-entry-%d@example.com", time.Now().UnixNano()%1_000_000_000)
	if _, parseErr := parsePage.Goto(parseBaseURL+"/signup", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /signup: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-name-input"); parseErr != nil {
		parseT.Fatalf("wait for signup name input: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-name-input", "Auth Entry QA"); parseErr != nil {
		parseT.Fatalf("fill signup name: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", parseSignupEmail); parseErr != nil {
		parseT.Fatalf("fill signup email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", "password123"); parseErr != nil {
		parseT.Fatalf("fill signup password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit signup form: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after signup: %v", parseErr)
	}
	parseArtifact.HasSignupFlow = true

	parseLogoutExample100AuthEntryUser(parseT, parsePage, parseBaseURL)
	parseArtifact.HasLogoutFlow = true

	parseLoginExample100AuthEntryUser(parseT, parsePage, parseBaseURL, "customer@email.com", "password")
	parseArtifact.HasLoginFlow = true

	if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("reload after customer login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after refresh: %v", parseErr)
	}
	parseArtifact.HasRefreshFlow = true

	parseLogoutExample100AuthEntryUser(parseT, parsePage, parseBaseURL)

	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login for password-reset entry: %v", parseErr)
	}
	if parseErr := parsePage.Click(`a:has-text("Forgot password?")`); parseErr != nil {
		parseT.Fatalf("click forgot-password entry: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector(`button:has-text("Send reset link")`); parseErr != nil {
		parseT.Fatalf("wait for send-reset-link button: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("fill reset email: %v", parseErr)
	}
	if parseErr := parsePage.Click(`button:has-text("Send reset link")`); parseErr != nil {
		parseT.Fatalf("click send reset link: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-password-input"); parseErr != nil {
		parseT.Fatalf("wait for password input after reset handoff: %v", parseErr)
	}
	parseArtifact.HasPasswordResetFlow = true

	if _, parseErr := parsePage.Goto(parseBaseURL+"/login?provider=google&entry=multi&next=%2Fapp%2Fdashboard", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login mixed-provider entry path: %v", parseErr)
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "mixed-provider entry")
	parseArtifact.GetMixedProviderRouteState = parseCaptureExample100AuthEntryRouteState(parseT, parsePage, "mixed-provider entry route")
	parseLoginExample100AuthEntryUser(parseT, parsePage, parseBaseURL, "customer@email.com", "password")
	parseArtifact.HasMixedProviderEntryFlow = true

	parseLogoutExample100AuthEntryUser(parseT, parsePage, parseBaseURL)
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login?provider=workspace-sso&workspace=acme-corp&entry=sso", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login workspace-sso entry path: %v", parseErr)
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "workspace-sso entry")
	parseArtifact.GetWorkspaceSSORouteState = parseCaptureExample100AuthEntryRouteState(parseT, parsePage, "workspace-sso entry route")
	parseArtifact.HasWorkspaceSSOEntryFlow = true

	if _, parseErr := parsePage.Goto(parseBaseURL+"/login?auth_callback=google&callback_state=expired&callback_error=access_denied&return_to=%2Fapp", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login callback-return entry path: %v", parseErr)
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "callback-return entry")
	parseArtifact.GetCallbackReturnRouteState = parseCaptureExample100AuthEntryRouteState(parseT, parsePage, "callback-return route")
	parseLoginExample100AuthEntryUser(parseT, parsePage, parseBaseURL, "customer@email.com", "password")
	parseArtifact.HasCallbackReturnEntryFlow = true

	parseLoginExample100AuthEntryUser(parseT, parsePage, parseBaseURL, "customer@email.com", "password")
	parsePrompt := fmt.Sprintf("auth-entry-first-chat-%d", time.Now().UnixNano()%1_000_000)
	if parseErr := parsePage.Click(`button:has-text("New chat")`); parseErr != nil {
		parseT.Fatalf("click new chat for first-chat matrix leg: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#chat-input", parsePrompt); parseErr != nil {
		parseT.Fatalf("fill first-chat prompt: %v", parseErr)
	}
	if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
		parseT.Fatalf("click send for first-chat matrix leg: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#streaming-assistant-bubble"); parseErr != nil {
		parseT.Fatalf("wait for streaming bubble in first-chat matrix leg: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parsePrompt),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for first-chat prompt visibility: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("streaming-assistant-bubble")`, nil); parseErr != nil {
		parseT.Fatalf("wait for first-chat stream completion: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for first-chat canonical route: %v", parseErr)
	}
	parseThreadPathValue, parseErr := parsePage.Evaluate(`() => window.location.pathname`)
	if parseErr != nil {
		parseT.Fatalf("read first-chat thread path: %v", parseErr)
	}
	parseArtifact.GetFirstChatThreadPath = strings.TrimSpace(fmt.Sprintf("%v", parseThreadPathValue))
	parseArtifact.HasFirstChatFlow = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100AuthEntryMatrixRegression verifies local password auth-entry coverage across signup/login/refresh/logout/reset/first-chat flows.
func TestExample100AuthEntryMatrixRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18115")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100AuthEntryMatrixArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 auth-entry matrix: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))

		if !parseArtifact.HasSignupFlow {
			parseT.Fatalf("signup leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasLoginFlow {
			parseT.Fatalf("login leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasRefreshFlow {
			parseT.Fatalf("refresh leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasLogoutFlow {
			parseT.Fatalf("logout leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasPasswordResetFlow {
			parseT.Fatalf("password-reset leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasMixedProviderEntryFlow || !strings.Contains(parseArtifact.GetMixedProviderRouteState, "provider=google") {
			parseT.Fatalf("mixed-provider leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasWorkspaceSSOEntryFlow || !strings.Contains(parseArtifact.GetWorkspaceSSORouteState, "provider=workspace-sso") {
			parseT.Fatalf("workspace-sso entry leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasCallbackReturnEntryFlow || !strings.Contains(parseArtifact.GetCallbackReturnRouteState, "auth_callback=google") {
			parseT.Fatalf("callback-return leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if !parseArtifact.HasFirstChatFlow || !strings.HasPrefix(parseArtifact.GetFirstChatThreadPath, "/app/thread/") {
			parseT.Fatalf("first-chat leg missing: %s", formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100AuthEntryMatrixSummary(parseArtifact))
		}
	})
}
