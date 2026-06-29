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

	playwright "github.com/playwright-community/playwright-go"
)

type example100EmailVerificationRegressionArtifact struct {
	HasSignupUnverifiedEntry      bool
	GetQAVerificationTokenLength  int
	HasVerifiedOutcome            bool
	HasVerificationRedirectTarget bool
	HasReplayDeniedOutcome        bool
	HasInvalidTokenError          bool
	HasNeutralUnknownResend       bool
	HasVerifiedAppEntry           bool
	GetConsoleErrorCount          int
	GetPageErrorCount             int
	GetConsoleSampleText          string
	GetPageErrorSampleText        string
}

// formatExample100EmailVerificationRegressionSummary formats one email-verification regression artifact for concise logs.
func formatExample100EmailVerificationRegressionSummary(parseArtifact example100EmailVerificationRegressionArtifact) string {
	return fmt.Sprintf(
		"signup-unverified=%t qa-token-len=%d verified-outcome=%t redirect-target=%t replay-denied=%t invalid-token-err=%t neutral-unknown=%t verified-app-entry=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.HasSignupUnverifiedEntry,
		parseArtifact.GetQAVerificationTokenLength,
		parseArtifact.HasVerifiedOutcome,
		parseArtifact.HasVerificationRedirectTarget,
		parseArtifact.HasReplayDeniedOutcome,
		parseArtifact.HasInvalidTokenError,
		parseArtifact.HasNeutralUnknownResend,
		parseArtifact.HasVerifiedAppEntry,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// parseExample100SignupVerificationAPICall posts to one signup-verification endpoint and returns the decoded ok,
// outcome, verification_token, and redirect_target fields.
func parseExample100SignupVerificationAPICall(parseT *testing.T, parseBaseURL string, parseEndpoint string, parseBody []byte, parseLabel string) (bool, string, string, string) {
	parseT.Helper()
	parseReq, parseErr := http.NewRequest(http.MethodPost, parseBaseURL+parseEndpoint, bytes.NewReader(parseBody))
	if parseErr != nil {
		parseT.Fatalf("email-verification (%s): build request: %v", parseLabel, parseErr)
	}
	parseReq.Header.Set("Content-Type", "application/json")
	parseResp, parseErr := http.DefaultClient.Do(parseReq)
	if parseErr != nil {
		parseT.Fatalf("email-verification (%s): http: %v", parseLabel, parseErr)
	}
	defer parseResp.Body.Close()
	parseRawBody, parseErr := io.ReadAll(parseResp.Body)
	if parseErr != nil {
		parseT.Fatalf("email-verification (%s): read response body: %v", parseLabel, parseErr)
	}
	var parsePayload struct {
		OK                bool   `json:"ok"`
		Outcome           string `json:"outcome"`
		Message           string `json:"message"`
		VerificationToken string `json:"verification_token"`
		RedirectTarget    string `json:"redirect_target"`
	}
	if parseErr = json.Unmarshal(parseRawBody, &parsePayload); parseErr != nil {
		parseT.Fatalf("email-verification (%s): decode response: %v raw=%q", parseLabel, parseErr, string(parseRawBody))
	}
	return parsePayload.OK, parsePayload.Outcome, parsePayload.VerificationToken, parsePayload.RedirectTarget
}

// captureExample100EmailVerificationRegressionArtifact executes the full email-verification regression matrix for one seeded server.
// Covers: signup into an unverified state, QA-mode token retrieval via resend, valid token consumption,
// replay-denied protection, invalid/expired-token handling, neutral resend for unknown email, and the
// verified user's first app entry path.
func captureExample100EmailVerificationRegressionArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100EmailVerificationRegressionArtifact {
	parseT.Helper()
	parseArtifact := example100EmailVerificationRegressionArtifact{}
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

	// Credentials for the new account created in this regression run.
	// The server DB is always fresh (TempDir), so a fixed email is safe here.
	const parseSignupEmail = "sv-regression@example.com"
	const parseSignupPassword = "verification-pass-123"
	const parseSignupName = "SV Regression"

	// ── Browser leg 1: signup into an unverified state ───────────────────────────
	// Navigate to /signup and create a fresh account. The server issues an auth token
	// immediately and the client enters the authenticated chat shell, but the account's
	// email address is not yet verified on the server side.
	if _, parseErr := parsePage.Goto(parseBaseURL+"/signup", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("email-verification: goto /signup: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("email-verification: wait for signup form: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-name-input", parseSignupName); parseErr != nil {
		parseT.Fatalf("email-verification: fill display name: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", parseSignupEmail); parseErr != nil {
		parseT.Fatalf("email-verification: fill signup email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", parseSignupPassword); parseErr != nil {
		parseT.Fatalf("email-verification: fill signup password: %v", parseErr)
	}
	if parseErr := parsePage.Click(`button:has-text("Create account")`); parseErr != nil {
		parseT.Fatalf("email-verification: click create-account: %v", parseErr)
	}
	// After signup the client dispatches SetAuthenticated and the app shell renders.
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			body: ((document.body && document.body.innerText) || "").slice(0, 800),
		})`)
		parseT.Fatalf("email-verification: wait for chat shell after signup: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasSignupUnverifiedEntry = true

	// ── API leg 1: neutral resend response for an unknown email ──────────────────
	// The server must return the same accepted outcome regardless of account existence
	// to avoid leaking account enumeration information.
	parseUnknownBody, _ := json.Marshal(map[string]string{"email": "no-account-here-sv@example.com"})
	parseUnknownOK, parseUnknownOutcome, _, _ := parseExample100SignupVerificationAPICall(parseT, parseBaseURL, "/api/public/auth/signup-verification/resend", parseUnknownBody, "unknown-email-resend")
	if parseUnknownOK && parseUnknownOutcome == "accepted" {
		parseArtifact.HasNeutralUnknownResend = true
	}

	// ── API leg 2: resend for the registered email – get QA verification token ───
	// The server is started with CHAT_ENV="" (dev mode) which causes it to expose the
	// raw verification token in the response so integration tests can consume it.
	parseKnownBody, _ := json.Marshal(map[string]string{"email": parseSignupEmail})
	parseResendOK, parseResendOutcome, parseQAToken, _ := parseExample100SignupVerificationAPICall(parseT, parseBaseURL, "/api/public/auth/signup-verification/resend", parseKnownBody, "registered-email-resend")
	if !parseResendOK || parseResendOutcome != "accepted" {
		parseT.Fatalf("email-verification: expected accepted from resend for registered email, got ok=%t outcome=%q", parseResendOK, parseResendOutcome)
	}
	if strings.TrimSpace(parseQAToken) == "" {
		parseT.Fatalf("email-verification: expected QA verification token in dev-mode resend response, got empty")
	}
	parseArtifact.GetQAVerificationTokenLength = len(parseQAToken)

	// ── API leg 3: consume the valid verification token ──────────────────────────
	// Successful consumption must return outcome="verified" and the app redirect target.
	parseConsumeBody, _ := json.Marshal(map[string]string{"token": parseQAToken})
	parseConsumeOK, parseConsumeOutcome, _, parseRedirectTarget := parseExample100SignupVerificationAPICall(parseT, parseBaseURL, "/api/public/auth/signup-verification/consume", parseConsumeBody, "valid-consume")
	if !parseConsumeOK || parseConsumeOutcome != "verified" {
		parseT.Fatalf("email-verification: expected verified outcome from consume, got ok=%t outcome=%q", parseConsumeOK, parseConsumeOutcome)
	}
	parseArtifact.HasVerifiedOutcome = true
	if strings.TrimSpace(parseRedirectTarget) != "" {
		parseArtifact.HasVerificationRedirectTarget = true
	}

	// ── API leg 4: replay-denied (already-used token) ────────────────────────────
	// Consuming the same token a second time must be denied to prevent replay attacks.
	parseReplayBody, _ := json.Marshal(map[string]string{"token": parseQAToken})
	parseReplayOK, parseReplayOutcome, _, _ := parseExample100SignupVerificationAPICall(parseT, parseBaseURL, "/api/public/auth/signup-verification/consume", parseReplayBody, "replay-consume")
	if !parseReplayOK && parseReplayOutcome == "replay_denied" {
		parseArtifact.HasReplayDeniedOutcome = true
	}

	// ── API leg 5: invalid/garbage token (expired-token surface) ─────────────────
	// A token that does not exist must return a non-ok response. This covers the same
	// error path as an expired token; package-level TTL tests live in server/app.
	parseGarbageBody, _ := json.Marshal(map[string]string{"token": "garbage-token-sv-does-not-exist"})
	parseGarbageOK, _, _, _ := parseExample100SignupVerificationAPICall(parseT, parseBaseURL, "/api/public/auth/signup-verification/consume", parseGarbageBody, "garbage-token-consume")
	if !parseGarbageOK {
		parseArtifact.HasInvalidTokenError = true
	}

	// ── Browser leg 2: verified user's first app entry path ──────────────────────
	// After token consumption the server returns redirect_target="/app?auth_verified=1".
	// Navigate to that URL; the browser already holds a valid auth token from signup, so
	// the app shell must render without re-authentication.
	parseAppEntry := "/app?auth_verified=1"
	if strings.TrimSpace(parseRedirectTarget) != "" {
		parseAppEntry = parseRedirectTarget
	}
	if _, parseErr := parsePage.Goto(parseBaseURL+parseAppEntry, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("email-verification: goto verified app entry %q: %v", parseAppEntry, parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			body: ((document.body && document.body.innerText) || "").slice(0, 800),
		})`)
		parseT.Fatalf("email-verification: wait for chat shell at verified app entry: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasVerifiedAppEntry = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100EmailVerificationRegression verifies the email-verification lifecycle across signup into an
// unverified state, QA-mode token retrieval via the resend endpoint, valid token consumption with outcome and
// redirect-target assertions, replay-denied protection, invalid-token error handling, neutral resend copy for
// unknown email addresses, and the verified user's first app entry path via the server-issued redirect target.
func TestExample100EmailVerificationRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18126")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100EmailVerificationRegressionArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 email-verification regression: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))

		if !parseArtifact.HasSignupUnverifiedEntry {
			parseT.Fatalf("signup-unverified-entry leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
		if parseArtifact.GetQAVerificationTokenLength == 0 {
			parseT.Fatalf("qa-verification-token leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasVerifiedOutcome {
			parseT.Fatalf("verified-outcome leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasVerificationRedirectTarget {
			parseT.Fatalf("verification-redirect-target leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasReplayDeniedOutcome {
			parseT.Fatalf("replay-denied-outcome leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasInvalidTokenError {
			parseT.Fatalf("invalid-token-error leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasNeutralUnknownResend {
			parseT.Fatalf("neutral-unknown-resend leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
		if !parseArtifact.HasVerifiedAppEntry {
			parseT.Fatalf("verified-app-entry leg missing: %s", formatExample100EmailVerificationRegressionSummary(parseArtifact))
		}
	})
}
