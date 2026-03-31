//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

// parseLogoutExample100CrossAccount logs the current user out via the logout button in the app shell.
// It expects the caller to have an authenticated session open on the page.
func parseLogoutExample100CrossAccount(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	// The logout button sits inside the user-menu area of the app toolbar.
	// It contains the text resolved from "auth.logout" — "Sign out" in EN.
	parseLogoutSelector := `button:has-text("Sign out")`
	parseLogout, parseErr := parsePage.WaitForSelector(parseLogoutSelector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(8000),
	})
	if parseErr != nil {
		parseT.Fatalf("cross-account isolation: wait for logout button: %v", parseErr)
	}
	if parseErr := parseLogout.Click(); parseErr != nil {
		parseT.Fatalf("cross-account isolation: click logout button: %v", parseErr)
	}
	// After logout the auth shell must be present.
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10000),
	}); parseErr != nil {
		parseT.Fatalf("cross-account isolation: wait for auth shell after logout: %v", parseErr)
	}
}

// assertExample100CrossAccountSettingsSection navigates to the settings route for one panel and verifies
// that the expected section content element is present and the unexpected one is absent.
func assertExample100CrossAccountSettingsSection(
	parseT *testing.T,
	parsePage playwright.Page,
	parseBaseURL string,
	parsePanelID string,
	parseExpectPresentID string,
	parseExpectAbsentID string,
	parseLabel string,
) {
	parseT.Helper()
	parseSettingsRoute := parseBaseURL + "/app/settings?panel=" + parsePanelID
	if _, parseErr := parsePage.Goto(parseSettingsRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("cross-account isolation (%s): goto settings route %s: %v", parseLabel, parseSettingsRoute, parseErr)
	}
	// Wait for the expected section pane element.
	if _, parseErr := parsePage.WaitForSelector(
		"#"+parseExpectPresentID,
		playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(12000)},
	); parseErr != nil {
		parseT.Fatalf("cross-account isolation (%s): wait for #%s: %v", parseLabel, parseExpectPresentID, parseErr)
	}
	// Confirm the absent section pane is not in the DOM.
	parseAbsentCountValue, parseErr := parsePage.Evaluate(
		fmt.Sprintf(`() => document.querySelectorAll('#%s').length`, parseExpectAbsentID),
	)
	if parseErr != nil {
		parseT.Fatalf("cross-account isolation (%s): count #%s: %v", parseLabel, parseExpectAbsentID, parseErr)
	}
	parseAbsentCount, _ := parseAbsentCountValue.(float64)
	if int(parseAbsentCount) != 0 {
		parseT.Fatalf("cross-account isolation (%s): expected #%s absent but found %d instance(s)", parseLabel, parseExpectAbsentID, int(parseAbsentCount))
	}
}

// TestExample100CrossAccountIsolation checks that account-owned in-memory UI state cannot bleed from one
// authenticated session to the next when a different user logs in on the same browser page.
//
// Flow:
//  1. Login as user A, navigate to the Security settings panel, confirm it is open.
//  2. Logout — auth token and workspace state are cleared.
//  3. Login as user B.
//  4. Open settings — the panel must default to Profile (not Security from user A's session).
//  5. Auth token in localStorage must belong to user B (non-empty, same-origin token present).
func TestExample100CrossAccountIsolation(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18124")

	// The seeded happy-path DB includes two known accounts.
	// customer@email.com is the primary end-user fixture.
	// admin@email.com is the superuser fixture from the admin journey seed.
	const (
		parseUserAEmail    = "customer@email.com"
		parseUserAPassword = "password"
		parseUserBEmail    = "admin@email.com"
		parseUserBPassword = "password"
	)

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		// ── Step 1: Login as user A ──────────────────────────────────────────────
		parseLoginExample100AuthEntryUser(parseT, parsePage, parseBaseURL, parseUserAEmail, parseUserAPassword)

		// Confirm user A starts at the authenticated shell (chat screen).
		if _, parseErr := parsePage.WaitForSelector("#chat-input", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseErr != nil {
			parseT.Fatalf("cross-account isolation: wait for chat input after user A login: %v", parseErr)
		}

		// Navigate to the Security settings panel for user A to set a non-default active section.
		assertExample100CrossAccountSettingsSection(
			parseT, parsePage, parseBaseURL,
			"settings-security",
			"settings-security", // expect present
			"settings-profile",  // expect absent (security replaces profile pane)
			"user A security panel open",
		)

		// ── Step 2: Logout user A ────────────────────────────────────────────────
		parseLogoutExample100CrossAccount(parseT, parsePage)

		// Verify the auth token was cleared from localStorage.
		parseTokenAfterLogoutValue, parseErr := parsePage.Evaluate(
			fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey),
		)
		if parseErr != nil {
			parseT.Fatalf("cross-account isolation: read auth token after logout: %v", parseErr)
		}
		parseTokenAfterLogout := strings.TrimSpace(fmt.Sprintf("%v", parseTokenAfterLogoutValue))
		if strings.EqualFold(parseTokenAfterLogout, "<nil>") {
			parseTokenAfterLogout = ""
		}
		if parseTokenAfterLogout != "" {
			parseT.Fatalf("cross-account isolation: auth token not cleared after logout: %s", parseTokenAfterLogout)
		}

		// ── Step 3: Login as user B ──────────────────────────────────────────────
		parseLoginExample100AuthEntryUser(parseT, parsePage, parseBaseURL, parseUserBEmail, parseUserBPassword)

		if _, parseErr := parsePage.WaitForSelector("#chat-input", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(10000),
		}); parseErr != nil {
			parseT.Fatalf("cross-account isolation: wait for chat input after user B login: %v", parseErr)
		}

		// ── Step 4: Open settings for user B — must default to Profile, not Security ──
		// Navigate to the settings route without a panel query to exercise the default resolution.
		parseSettingsDefaultRoute := parseBaseURL + "/app/settings"
		if _, parseErr := parsePage.Goto(parseSettingsDefaultRoute, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("cross-account isolation: goto default settings route for user B: %v", parseErr)
		}
		// The default section is settings-profile.
		if _, parseErr := parsePage.WaitForSelector("#settings-profile", playwright.PageWaitForSelectorOptions{
			Timeout: playwright.Float(12000),
		}); parseErr != nil {
			parseT.Fatalf("cross-account isolation: user B settings modal must open to profile section, not user A's security section: %v", parseErr)
		}
		// Confirm the Security pane from user A's session is not rendered for user B.
		parseSecurityCountValue, parseErr := parsePage.Evaluate(`() => document.querySelectorAll('#settings-security').length`)
		if parseErr != nil {
			parseT.Fatalf("cross-account isolation: count #settings-security for user B: %v", parseErr)
		}
		parseSecurityCount, _ := parseSecurityCountValue.(float64)
		if int(parseSecurityCount) != 0 {
			parseT.Fatalf("cross-account isolation: user B sees user A's security panel (%d instance(s)): ActiveSettingsSection was not reset on account switch", int(parseSecurityCount))
		}

		// ── Step 5: Auth token in localStorage belongs to user B ─────────────────
		parseTokenAfterLoginValue, parseErr2 := parsePage.Evaluate(
			fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey),
		)
		if parseErr2 != nil {
			parseT.Fatalf("cross-account isolation: read auth token for user B: %v", parseErr2)
		}
		parseTokenAfterLogin := strings.TrimSpace(fmt.Sprintf("%v", parseTokenAfterLoginValue))
		if strings.EqualFold(parseTokenAfterLogin, "<nil>") {
			parseTokenAfterLogin = ""
		}
		if parseTokenAfterLogin == "" {
			parseT.Fatal("cross-account isolation: user B auth token missing from localStorage after login")
		}

		// Ensure sufficient settle time so any async state-sync side-effects land before the test ends.
		time.Sleep(300 * time.Millisecond)
	})
}
