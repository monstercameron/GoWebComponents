//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

type example100PasswordRecoveryRegressionArtifact struct {
	HasResetFormEntry       bool
	HasResetConfirmCard     bool
	HasNeutralResetBodyCopy bool
	HasNeutralUnknownEmail  bool
	GetQAResetTokenLength   int
	HasPasswordUpdated      bool
	HasReplayDeniedOutcome  bool
	HasInvalidTokenError    bool
	HasPostResetLogin       bool
	HasAuthTokenAfterReset  bool
	GetConsoleErrorCount    int
	GetPageErrorCount       int
	GetConsoleSampleText    string
	GetPageErrorSampleText  string
}

// formatExample100PasswordRecoveryRegressionSummary formats one password-recovery regression artifact for concise logs.
func formatExample100PasswordRecoveryRegressionSummary(parseArtifact example100PasswordRecoveryRegressionArtifact) string {
	return fmt.Sprintf(
		"reset-form=%t confirm-card=%t neutral-copy=%t neutral-unknown=%t qa-token-len=%d pw-updated=%t replay-denied=%t invalid-token-err=%t post-reset-login=%t auth-token=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.HasResetFormEntry,
		parseArtifact.HasResetConfirmCard,
		parseArtifact.HasNeutralResetBodyCopy,
		parseArtifact.HasNeutralUnknownEmail,
		parseArtifact.GetQAResetTokenLength,
		parseArtifact.HasPasswordUpdated,
		parseArtifact.HasReplayDeniedOutcome,
		parseArtifact.HasInvalidTokenError,
		parseArtifact.HasPostResetLogin,
		parseArtifact.HasAuthTokenAfterReset,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// parseExample100PasswordResetAPICall posts to one password-reset endpoint and returns the decoded ok, outcome, and reset_token fields.
func parseExample100PasswordResetAPICall(parseT *testing.T, parseBaseURL string, parseEndpoint string, parseBody []byte, parseLabel string) (bool, string, string) {
	parseT.Helper()
	parseReq, parseErr := http.NewRequest(http.MethodPost, parseBaseURL+parseEndpoint, bytes.NewReader(parseBody))
	if parseErr != nil {
		parseT.Fatalf("password-recovery (%s): build request: %v", parseLabel, parseErr)
	}
	parseReq.Header.Set("Content-Type", "application/json")
	parseResp, parseErr := http.DefaultClient.Do(parseReq)
	if parseErr != nil {
		parseT.Fatalf("password-recovery (%s): http: %v", parseLabel, parseErr)
	}
	defer parseResp.Body.Close()
	parseRawBody, parseErr := io.ReadAll(parseResp.Body)
	if parseErr != nil {
		parseT.Fatalf("password-recovery (%s): read response body: %v", parseLabel, parseErr)
	}
	var parsePayload struct {
		OK         bool   `json:"ok"`
		Outcome    string `json:"outcome"`
		Message    string `json:"message"`
		ResetToken string `json:"reset_token"`
	}
	if parseErr = json.Unmarshal(parseRawBody, &parsePayload); parseErr != nil {
		parseT.Fatalf("password-recovery (%s): decode response: %v raw=%q", parseLabel, parseErr, string(parseRawBody))
	}
	return parsePayload.OK, parsePayload.Outcome, parsePayload.ResetToken
}

// captureExample100PasswordRecoveryRegressionArtifact executes the full password-recovery regression matrix for one seeded server.
// Covers: reset-request form entry, success confirmation card, neutral email copy, QA-mode token request,
// valid token consumption, replay-denied protection, invalid/expired-token handling, and post-reset login.
func captureExample100PasswordRecoveryRegressionArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100PasswordRecoveryRegressionArtifact {
	parseT.Helper()
	parseArtifact := example100PasswordRecoveryRegressionArtifact{}
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

	// ── Browser leg 1: reset-request form entry ──────────────────────────────────
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("password-recovery: goto /login: %v", parseErr)
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "password-recovery /login")

	// Click the "Forgot password?" link in the login form to enter the reset shell.
	if parseErr := parsePage.Click(`a:has-text("Forgot password?")`); parseErr != nil {
		parseT.Fatalf("password-recovery: click forgot-password link: %v", parseErr)
	}
	// The reset form must surface the email input and the send button.
	if _, parseErr := parsePage.WaitForSelector(`button:has-text("Send reset link")`); parseErr != nil {
		parseT.Fatalf("password-recovery: wait for send-reset-link button: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("password-recovery: wait for email input on reset form: %v", parseErr)
	}
	parseArtifact.HasResetFormEntry = true

	// ── Browser leg 2: success confirmation card ─────────────────────────────────
	if parseErr := parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("password-recovery: fill reset email: %v", parseErr)
	}
	if parseErr := parsePage.Click(`button:has-text("Send reset link")`); parseErr != nil {
		parseT.Fatalf("password-recovery: click send-reset-link: %v", parseErr)
	}
	// After the button fires, the component switches to the confirmation card showing "Check your inbox".
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.body && document.body.innerText.includes("Check your inbox")`,
		nil,
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => document.body && document.body.innerText.slice(0, 800)`)
		parseT.Fatalf("password-recovery: wait for reset confirmation card: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasResetConfirmCard = true

	// ── Browser leg 3: neutral body copy (no account-existence hint) ─────────────
	// The confirmation body must say "If an account exists" rather than confirming account existence.
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.body && document.body.innerText.includes("If an account exists")`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("password-recovery: wait for neutral reset body copy: %v", parseErr)
	}
	parseArtifact.HasNeutralResetBodyCopy = true

	// Navigate back to login via the "Back to log in" anchor so the login form is ready for the post-reset leg.
	if parseErr := parsePage.Click(`a:has-text("Back to log in")`); parseErr != nil {
		parseT.Fatalf("password-recovery: click back-to-login from confirmation card: %v", parseErr)
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "password-recovery back-to-login")

	// ── API leg 1: neutral response for unknown email ─────────────────────────────
	// The server must accept and return the same "accepted" outcome regardless of account existence.
	parseUnknownBody, _ := json.Marshal(map[string]string{"email": "no-account-here-pr@example.com"})
	parseUnknownOK, parseUnknownOutcome, _ := parseExample100PasswordResetAPICall(parseT, parseBaseURL, "/api/public/auth/password-reset/request", parseUnknownBody, "unknown-email")
	if parseUnknownOK && parseUnknownOutcome == "accepted" {
		parseArtifact.HasNeutralUnknownEmail = true
	}

	// ── API leg 2: valid token request for known account ─────────────────────────
	// The server is started without CHAT_ENV, which defaults to "" and exposes QA tokens.
	parseKnownBody, _ := json.Marshal(map[string]string{"email": "customer@email.com"})
	parseRequestOK, parseRequestOutcome, parseQAToken := parseExample100PasswordResetAPICall(parseT, parseBaseURL, "/api/public/auth/password-reset/request", parseKnownBody, "valid-email-request")
	if !parseRequestOK || parseRequestOutcome != "accepted" {
		parseT.Fatalf("password-recovery: expected accepted outcome for valid email, got ok=%t outcome=%q", parseRequestOK, parseRequestOutcome)
	}
	if strings.TrimSpace(parseQAToken) == "" {
		parseT.Fatalf("password-recovery: expected QA reset token in development-mode response, got empty")
	}
	parseArtifact.GetQAResetTokenLength = len(parseQAToken)

	// ── API leg 3: valid token consumption (password update) ─────────────────────
	parseNewPassword := "recovery-reset-456"
	parseConsumeBody, _ := json.Marshal(map[string]string{"token": parseQAToken, "password": parseNewPassword})
	parseConsumeOK, parseConsumeOutcome, _ := parseExample100PasswordResetAPICall(parseT, parseBaseURL, "/api/public/auth/password-reset/consume", parseConsumeBody, "valid-consume")
	if !parseConsumeOK || parseConsumeOutcome != "password_updated" {
		parseT.Fatalf("password-recovery: expected password_updated outcome, got ok=%t outcome=%q", parseConsumeOK, parseConsumeOutcome)
	}
	parseArtifact.HasPasswordUpdated = true

	// ── API leg 4: replay-denied (consumed-token handling) ───────────────────────
	// Consuming the same token a second time must be denied.
	parseReplayBody, _ := json.Marshal(map[string]string{"token": parseQAToken, "password": parseNewPassword})
	parseReplayOK, parseReplayOutcome, _ := parseExample100PasswordResetAPICall(parseT, parseBaseURL, "/api/public/auth/password-reset/consume", parseReplayBody, "replay")
	if !parseReplayOK && parseReplayOutcome == "replay_denied" {
		parseArtifact.HasReplayDeniedOutcome = true
	}

	// ── API leg 5: invalid/garbage token (expired-token–like surface) ────────────
	// A token that does not exist must return an error, not a success. This covers the
	// same UI path as an expired token; package-level expired-token tests are in server/app.
	parseGarbageBody, _ := json.Marshal(map[string]string{"token": "garbage-token-pr-does-not-exist", "password": parseNewPassword})
	parseGarbageOK, _, _ := parseExample100PasswordResetAPICall(parseT, parseBaseURL, "/api/public/auth/password-reset/consume", parseGarbageBody, "garbage-token")
	if !parseGarbageOK {
		parseArtifact.HasInvalidTokenError = true
	}

	// ── Browser leg 4: post-reset login with the new password ────────────────────
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("password-recovery: goto /login for post-reset login: %v", parseErr)
	}
	parseWaitForAuthEmailInput(parseT, parsePage, "password-recovery post-reset /login")
	if parseErr := parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("password-recovery: fill post-reset email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", parseNewPassword); parseErr != nil {
		parseT.Fatalf("password-recovery: fill post-reset new password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("password-recovery: submit post-reset login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			body: ((document.body && document.body.innerText) || "").slice(0, 800),
		})`)
		parseT.Fatalf("password-recovery: wait for chat input after post-reset login: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasPostResetLogin = true

	// Confirm the auth token is stored in localStorage for the new session.
	parseTokenValue, parseErr := parsePage.Evaluate(
		fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey),
	)
	if parseErr != nil {
		parseT.Fatalf("password-recovery: read auth token after post-reset login: %v", parseErr)
	}
	if parseToken, _ := parseTokenValue.(string); strings.TrimSpace(parseToken) != "" {
		parseArtifact.HasAuthTokenAfterReset = true
	}

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100PasswordRecoveryRegression verifies the password-recovery lifecycle across reset-request
// form entry, success confirmation with neutral email copy, valid token consumption via the QA-mode API,
// replay-denied protection, invalid-token error handling, and post-reset browser login with the new password.
func TestExample100PasswordRecoveryRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18125")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100PasswordRecoveryRegressionArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 password-recovery regression: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))

		if !parseArtifact.HasResetFormEntry {
			parseT.Fatalf("reset-form-entry leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasResetConfirmCard {
			parseT.Fatalf("reset-confirm-card leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasNeutralResetBodyCopy {
			parseT.Fatalf("neutral-reset-body-copy leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasNeutralUnknownEmail {
			parseT.Fatalf("neutral-unknown-email leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if parseArtifact.GetQAResetTokenLength == 0 {
			parseT.Fatalf("qa-reset-token leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasPasswordUpdated {
			parseT.Fatalf("password-updated leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasReplayDeniedOutcome {
			parseT.Fatalf("replay-denied leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasInvalidTokenError {
			parseT.Fatalf("invalid-token-error leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasPostResetLogin {
			parseT.Fatalf("post-reset-login leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasAuthTokenAfterReset {
			parseT.Fatalf("auth-token-after-reset leg missing: %s", formatExample100PasswordRecoveryRegressionSummary(parseArtifact))
		}
	})
}
