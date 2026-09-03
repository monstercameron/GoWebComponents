//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/operator-flow/dashboard-triage.spec.md
//       assertion 1 - "mock sign-in or mock session reaches the dashboard"
// STORY: manifest.json -> manual_stories -> "manual-internal-sign-in-recovery"
// =============================================================================
//
// THIS SPEC EXISTS BECAUSE OF A SPECIFIC PAST FAILURE
//
// An earlier attempt at Atlas browser coverage reported NINE GREEN OPERATOR
// ROUTES. Five of them were rendering the same unauthenticated screen. The
// mechanism is worth understanding, because it will catch the next person too:
//
//   - GET /auth/mock-sign-in sets ONLY the atlas_csrf cookie. It does NOT
//     create a session. The session cookie (atlas_mock_role) is set by the
//     POST.
//   - An unauthenticated request to any /app route is answered with a 303 to
//     /auth/mock-sign-in?next=<route>, which the browser follows silently.
//   - The result is HTTP 200, a valid document, the Atlas header, a populated
//     #app, and a perfectly successful hydration.
//
// So "status < 400", "the page hydrated", "the shell is present" and "the
// wordmark is visible" are ALL TRUE of the failure. Any spec built only on
// those signals is vacuous on every internal route.
//
// This spec therefore does two things no route smoke test does: it asserts the
// UNAUTHENTICATED state positively (so we know what the trap screen looks
// like), and it drives the real sign-in POST and asserts the authenticated
// state is observably DIFFERENT from it.

import (
	"runtime"
	"strings"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

const (
	// Content that exists only on the mock sign-in screen.
	atlasSignInMarker = "Mock internal access"
	// Content that exists only on the authenticated ops dashboard. Chosen from
	// the dashboard's own summary band rather than from the shell, because the
	// shell renders on the sign-in screen too.
	atlasDashboardMarker = "Demand and operations overview"
	// Notice slug the sign-in POST stamps on its redirect target.
	atlasSignInNotice = "mock-session-started"
)

// atlasSpecCookieValue returns one cookie's value from the browser context, or
// "" when it is absent.
//
// Reading the cookie is not decoration: it is the difference between "the page
// looks signed in" and "a session cookie exists". atlas_mock_role is HttpOnly,
// so document.cookie cannot see it - the context API is the only way to check
// it from a spec.
func atlasSpecCookieValue(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseName string) string {
	parseT.Helper()
	parseCookies, parseErr := parsePage.Context().Cookies(parseBaseURL)
	if parseErr != nil {
		parseT.Fatalf("read browser cookies: %v", parseErr)
	}
	for _, parseCookie := range parseCookies {
		if parseCookie.Name == parseName {
			return parseCookie.Value
		}
	}
	return ""
}

// TestAtlasOperatorMockSignInReachesDashboard drives the real mock sign-in form
// and proves the authenticated dashboard differs from the sign-in screen.
func TestAtlasOperatorMockSignInReachesDashboard(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)

		// ---------------------------------------------------------------------
		// PHASE 1 - unauthenticated entry must land on the sign-in recovery
		// route, and must NOT look like the dashboard.
		// ---------------------------------------------------------------------
		//
		// Note the expected status is 200, not 401 or 302: the 303 is followed
		// by the browser, so the response we observe is the sign-in document.
		// That is exactly why status is useless as an auth signal here.
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/dashboard", 0)

		parseSignInURL := parsePage.URL()
		if !strings.Contains(parseSignInURL, "/auth/mock-sign-in") {
			parseT.Fatalf("unauthenticated /app/dashboard did not redirect to the mock sign-in route; landed on %s | %s",
				parseSignInURL, parseDiagnostics.Summary())
		}
		// The "next" round trip is what makes sign-in a recovery flow rather
		// than a dead end: the server must remember where the operator was
		// going. Asserting it here is what makes the PHASE 3 landing check
		// meaningful.
		if !strings.Contains(parseSignInURL, "next=%2Fapp%2Fdashboard") {
			parseT.Fatalf("mock sign-in URL %q does not carry next=/app/dashboard; the requested route was lost | %s",
				parseSignInURL, parseDiagnostics.Summary())
		}

		parseSignInHeading := atlasSpecPrimaryHeading(parseT, parsePage, parseDiagnostics)
		if !atlasSpecTextContains(parseSignInHeading, "Atlas Mock Sign In") {
			parseT.Fatalf("expected the mock sign-in heading, got %q | %s", parseSignInHeading, parseDiagnostics.Summary())
		}

		parseSignInText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
		assertAtlasSpecTextContains(parseT, "mock sign-in screen", parseSignInText, atlasSignInMarker, parseDiagnostics)
		assertAtlasSpecTextContains(parseT, "mock sign-in screen", parseSignInText, "Operations Lead", parseDiagnostics)
		// The trap, stated as an assertion: this screen must not be mistakable
		// for the dashboard.
		assertAtlasSpecTextMissing(parseT, "mock sign-in screen", parseSignInText, atlasDashboardMarker, parseDiagnostics)
		assertAtlasSpecTextMissing(parseT, "mock sign-in screen", parseSignInText, "Atlas Demo Operator", parseDiagnostics)

		// And no session cookie exists yet. This is the machine-checkable
		// version of "the GET did not sign anybody in".
		if parseRole := atlasSpecCookieValue(parseT, parsePage, parseBaseURL, "atlas_mock_role"); parseRole != "" {
			parseT.Fatalf("atlas_mock_role cookie already set to %q before any POST; the GET sign-in page must not create a session", parseRole)
		}

		// ---------------------------------------------------------------------
		// PHASE 2 - drive the real form for one specific role.
		// ---------------------------------------------------------------------
		//
		// The page renders one form per role, each carrying hidden role and
		// next inputs. signInAtlasSpecOperator selects by role value and
		// asserts exactly one match, so a layout change that duplicates or
		// drops a card fails loudly instead of clicking the wrong role.
		parseLanded := signInAtlasSpecOperator(parseT, parsePage, parseDiagnostics, parseBaseURL, "ops_lead")

		// ---------------------------------------------------------------------
		// PHASE 3 - the authenticated dashboard.
		// ---------------------------------------------------------------------
		if !strings.Contains(parseLanded, "/app/dashboard") {
			parseT.Fatalf("sign-in did not return to the requested route; landed on %s | %s", parseLanded, parseDiagnostics.Summary())
		}
		if !strings.Contains(parseLanded, "atlas_notice="+atlasSignInNotice) {
			parseT.Fatalf("sign-in landing URL %q is missing atlas_notice=%s | %s",
				parseLanded, atlasSignInNotice, parseDiagnostics.Summary())
		}

		// The session cookie is now real, and carries the role we chose - not
		// just "some role". A handler that ignored the submitted role and
		// defaulted to inventory_manager would pass a presence-only check.
		if parseRole := atlasSpecCookieValue(parseT, parsePage, parseBaseURL, "atlas_mock_role"); parseRole != "ops_lead" {
			parseT.Fatalf("atlas_mock_role cookie = %q after signing in as ops_lead | %s", parseRole, parseDiagnostics.Summary())
		}

		parseDashboardHeading := atlasSpecPrimaryHeading(parseT, parsePage, parseDiagnostics)
		if !atlasSpecTextContains(parseDashboardHeading, "Atlas Ops Dashboard") {
			parseT.Fatalf("expected the ops dashboard heading after sign-in, got %q | %s", parseDashboardHeading, parseDiagnostics.Summary())
		}

		parseDashboardText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
		// Operator identity: rendered by the internal shell only when a
		// session resolved.
		assertAtlasSpecTextContains(parseT, "authenticated dashboard", parseDashboardText, "Atlas Demo Operator", parseDiagnostics)
		// The role label is derived server-side from the cookie, so this ties
		// the rendered page back to the specific role we submitted.
		assertAtlasSpecTextContains(parseT, "authenticated dashboard", parseDashboardText, "ops lead", parseDiagnostics)
		// Dashboard-only content, and the post-redirect notice.
		assertAtlasSpecTextContains(parseT, "authenticated dashboard", parseDashboardText, atlasDashboardMarker, parseDiagnostics)
		assertAtlasSpecTextContains(parseT, "authenticated dashboard", parseDashboardText, atlasSignInNotice, parseDiagnostics)
		// The sign-in screen must be gone. Paired with the PHASE 1 assertions
		// this closes the loop: the two states are provably different screens.
		assertAtlasSpecTextMissing(parseT, "authenticated dashboard", parseDashboardText, atlasSignInMarker, parseDiagnostics)
		assertAtlasSpecTextMissing(parseT, "authenticated dashboard", parseDashboardText, "Start session", parseDiagnostics)

		// ---------------------------------------------------------------------
		// PHASE 4 - re-entering the sign-in route with a live session must
		// bounce back into the console instead of offering sign-in again.
		// ---------------------------------------------------------------------
		//
		// This is cheap and it guards a real annoyance class: a sign-in page
		// that keeps showing itself to an already-authenticated operator, or
		// worse, one that resets the session on revisit.
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/auth/mock-sign-in?next=%2Fapp%2Fdashboard", 0)
		if strings.Contains(parsePage.URL(), "/auth/mock-sign-in") {
			parseT.Fatalf("an authenticated visit to /auth/mock-sign-in stayed on the sign-in route (%s) | %s",
				parsePage.URL(), parseDiagnostics.Summary())
		}
		assertAtlasSpecOperatorSessionVisible(parseT, parsePage, parseDiagnostics, "sign-in revisit with live session")

		assertAtlasSpecClean(parseT, parseDiagnostics, "atlas operator mock sign-in")
	})
}

// COVERAGE NOTE
//
// COVERED: unauthenticated redirect and its next round trip; the real sign-in
// POST for a chosen role; session cookie creation; authenticated dashboard
// identity, role label and notice; authenticated revisit of the sign-in route.
//
// NOT COVERED HERE: sign-out (POST /auth/mock-sign-out), the other two mock
// roles, and role-scoped authorisation differences between roles. The manifest
// does not describe per-role permission differences for these surfaces, so
// there is nothing specific to assert yet beyond what this spec covers.
