//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/buyer-flow/form-submit.spec.md
//       (plus recovery-edge-cases -> recovery-failed-write, partially)
// =============================================================================
//
// SUBJECT: the CSRF-protected public restock-request form on /shop/frame-desk,
// end to end through the real browser.
//
// WHY THIS FORM
//
// Atlas's public write forms are ordinary <form method="post"> elements
// rendered by wasm and posted natively by the browser to /api/... . The server
// then does the whole CSRF dance:
//
//   - a csrf_token hidden input is rendered into the form from the SSR payload
//   - the matching atlas_csrf cookie is set on the document request
//   - POST validates same-origin, then cookie-vs-form-field equality
//   - on success it answers 303 back to the Referer with ?atlas_notice=<slug>
//
// That makes one form submit an honest test of five separate subsystems:
// hydration (the form only exists because wasm painted it), the bootstrap
// payload (the token comes from there), cookie handling, CSRF validation, and
// the post-redirect-get notice render. A Go unit test on the handler covers
// none of the browser half.
//
// FOUR CASES, AND WHY EACH ONE IS NEEDED
//
//   1. valid submit          -> 303 round trip and a visible success notice.
//   2. invalid email         -> 400 with a field-level message.
//   3. tampered csrf_token   -> 403 csrf_token_invalid.
//   4. deleted csrf cookie   -> 403 csrf_cookie_missing.
//
// Case 1 alone would pass against a server with CSRF validation entirely
// removed. Cases 3 and 4 are what make case 1 mean "CSRF works" rather than
// "the POST succeeded"; they attack each half of the double-submit pair
// separately, so a check that compares the token against nothing, or accepts a
// missing cookie, cannot pass all three. Case 2 separates "rejected because of
// CSRF" from "rejected because of validation" - without it, a server that 403s
// everything would look like a server that validates.

import (
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

const (
	// The buyer restock form. Selecting by its action attribute rather than by
	// a generated id is deliberate: Atlas element ids are runtime-generated
	// (gwc-7-3 and similar) and shift whenever hook order changes, so any spec
	// pinned to one would rot on the next unrelated edit. The action is a
	// product contract - it is the endpoint the form posts to.
	atlasRestockFormSelector = `form[action="/api/public/products/frame-desk/restock-requests"]`

	// Notice slug the server appends to the redirect target on success.
	// Atlas renders the raw slug into a banner, so this doubles as the visible
	// success text.
	atlasRestockSuccessNotice = "restock-request-submitted"
)

// prepareAtlasRestockForm opens the product route and returns once the restock
// form is fillable.
//
// The viewport is pinned to a desktop width first. The restock form lives in
// the product action rail, which Atlas marks `hidden ... xl:grid` - below the
// xl breakpoint the rail collapses behind an "Open buying drawer" button and
// the form is not interactable. Playwright's default 1280px viewport sits
// exactly on that breakpoint, which is the worst possible place to be: it
// works until something shifts by one pixel. Pinning 1440 puts the spec
// unambiguously in the desktop layout it means to test.
func prepareAtlasRestockForm(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseBaseURL string) {
	parseT.Helper()
	if parseErr := parsePage.SetViewportSize(1440, 900); parseErr != nil {
		parseT.Fatalf("set desktop viewport: %v", parseErr)
	}
	openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/shop/frame-desk", 0)

	// Wait for the form itself, not just for hydration. The action rail paints
	// as part of the route render, but waiting on the form is what turns
	// "the page mounted" into "the thing under test exists".
	if _, parseErr := parsePage.WaitForSelector(atlasRestockFormSelector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(atlasSpecActionTimeoutMS),
		State:   playwright.WaitForSelectorStateVisible,
	}); parseErr != nil {
		parseT.Fatalf("restock form never became visible on /shop/frame-desk: %v | %s", parseErr, parseDiagnostics.Summary())
	}

	// Assert the form's FIELD-NAME CONTRACT before touching anything.
	//
	// This check exists because of a real failure it caught. Without it, a form
	// whose email input had lost its name attribute failed later as
	// "fill restock email: timeout: 30000ms exceeded" - a message that blames
	// the harness and names nothing. With it, the failure states exactly which
	// input is missing and what the form renders instead.
	//
	// The names are asserted as an exact multiset, not as "contains email":
	// the observed defect rendered `preferred_warehouse_id` TWICE (both fields
	// collapsed onto the last one's props), which a presence-only check on
	// preferred_warehouse_id would have happily accepted. A duplicated name is
	// its own bug - the browser posts the value twice and the server reads
	// whichever comes first - so the count matters as much as the set.
	assertAtlasFormFieldNames(parseT, parsePage, parseDiagnostics, atlasRestockFormSelector,
		[]string{"csrf_token", "email", "preferred_warehouse_id"})

	// The hidden CSRF input must be present and non-empty BEFORE we submit.
	// If it were empty, every negative case below would 403 for the wrong
	// reason and the spec would look like it was proving CSRF while actually
	// proving that Atlas forgot to render a token.
	parseToken := atlasSpecFieldValue(parseT, parsePage, parseDiagnostics, atlasRestockFormSelector+` input[name="csrf_token"]`)
	if strings.TrimSpace(parseToken) == "" {
		parseT.Fatalf("restock form rendered an empty csrf_token; the SSR bootstrap payload is not carrying a token | %s", parseDiagnostics.Summary())
	}
}

// fillAtlasRestockForm enters a valid-shaped payload, overriding the email.
func fillAtlasRestockForm(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseEmail string) {
	parseT.Helper()
	if parseErr := parsePage.Fill(atlasRestockFormSelector+` input[name="email"]`, parseEmail); parseErr != nil {
		parseT.Fatalf("fill restock email: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	// new-jersey-hub is a seeded warehouse slug. The server only requires a
	// non-empty value here, but using a real slug keeps the success path
	// exercising a genuine foreign-key-shaped write instead of storing junk.
	if parseErr := parsePage.Fill(atlasRestockFormSelector+` input[name="preferred_warehouse_id"]`, "new-jersey-hub"); parseErr != nil {
		parseT.Fatalf("fill restock preferred warehouse: %v | %s", parseErr, parseDiagnostics.Summary())
	}
}

// submitAtlasRestockForm clicks submit and returns the resulting main response.
//
// ExpectNavigation wraps the click because a native form POST and the browser's
// document swap are one event. Clicking and then polling the DOM would race the
// navigation: the assertion could read the pre-submit document and pass, or read
// a half-torn-down one and fail for no product reason.
func submitAtlasRestockForm(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) playwright.Response {
	parseT.Helper()
	parseResponse, parseErr := parsePage.ExpectNavigation(func() error {
		return parsePage.Click(atlasRestockFormSelector + ` button[type="submit"]`)
	}, playwright.PageExpectNavigationOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(atlasSpecActionTimeoutMS),
	})
	if parseErr != nil {
		parseT.Fatalf("restock form submit did not navigate: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	if parseResponse == nil {
		parseT.Fatalf("restock form submit produced a nil response | %s", parseDiagnostics.Summary())
	}
	return parseResponse
}

// TestAtlasBuyerRestockFormCSRFRoundTrip covers the success path and the three
// rejection paths of the CSRF-protected public restock form.
func TestAtlasBuyerRestockFormCSRFRoundTrip(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	// One server for all four cases. They are independent at the HTTP level
	// and re-booting per case would triple the runtime for nothing.
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	// -------------------------------------------------------------------------
	parseT.Run("valid_submit_redirects_with_success_notice", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
			prepareAtlasRestockForm(parseT, parsePage, parseDiagnostics, parseBaseURL)

			// Baseline: the success notice must NOT already be on the page.
			// Without this, "notice is present after submit" could be true of
			// a page that always renders it, and the assertion would be free.
			parseBefore := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
			assertAtlasSpecTextMissing(parseT, "pre-submit product route", parseBefore, atlasRestockSuccessNotice, parseDiagnostics)

			fillAtlasRestockForm(parseT, parsePage, parseDiagnostics, "atlas-spec-buyer@example.com")
			parseResponse := submitAtlasRestockForm(parseT, parsePage, parseDiagnostics)

			// The server answers 303 and the browser follows it, so the
			// response we see is the FINAL document: 200 on the product route.
			// Asserting 200 here therefore asserts "the redirect resolved to a
			// renderable Atlas page", not "the POST returned 200".
			if parseResponse.Status() != 200 {
				parseT.Fatalf("restock success round trip ended on status %d (expected the 303 to resolve to a 200 page) | %s",
					parseResponse.Status(), parseDiagnostics.Summary())
			}

			// The redirect target proves the server chose the post-redirect-get
			// path and stamped its notice, rather than dumping JSON at the
			// browser or bouncing the buyer somewhere unrelated.
			parseLanded := parsePage.URL()
			if !strings.Contains(parseLanded, "/shop/frame-desk") {
				parseT.Fatalf("restock success redirected away from the product route: %s | %s", parseLanded, parseDiagnostics.Summary())
			}
			if !strings.Contains(parseLanded, "atlas_notice="+atlasRestockSuccessNotice) {
				parseT.Fatalf("restock success URL %q is missing atlas_notice=%s | %s",
					parseLanded, atlasRestockSuccessNotice, parseDiagnostics.Summary())
			}

			// ...and the notice must be VISIBLE, not merely in the URL. Atlas
			// reads the query parameter out of the SSR bootstrap payload and
			// paints a banner from wasm, so this last step is the one that
			// proves the buyer is actually told the request landed.
			waitForAtlasSpecHydration(parseT, parsePage, parseDiagnostics, parseLanded)
			parseAfter := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
			assertAtlasSpecTextContains(parseT, "post-submit product route", parseAfter, atlasRestockSuccessNotice, parseDiagnostics)
			// Still the product route, not a bare JSON page.
			//
			// The marker was "Why this product page is easier to use", a three-card
			// strip in which the page graded its own UX for the buyer. The design
			// system pass deleted it as content-free filler, so this now keys on the
			// promise-lane module, which exists ONLY on the product route and is a
			// stronger claim besides: it proves the deferred island re-mounted after
			// the post-redirect-get rather than merely that some copy survived.
			assertAtlasSpecTextContains(parseT, "post-submit product route", parseAfter, "Regional promise lanes", parseDiagnostics)

			assertAtlasSpecClean(parseT, parseDiagnostics, "atlas restock valid submit")
		})
	})

	// -------------------------------------------------------------------------
	parseT.Run("invalid_email_is_rejected_with_field_error", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
			// This case provokes a 400 on purpose, so allow it. Chromium also
			// logs "Failed to load resource: ... 400" as a console error for
			// the same request; allowing the status code covers the response
			// entry, and the console entry is allowed explicitly below.
			parseDiagnostics.Allow("restock-requests")
			parseDiagnostics.Allow("400 (Bad Request)")

			prepareAtlasRestockForm(parseT, parsePage, parseDiagnostics, parseBaseURL)
			fillAtlasRestockForm(parseT, parsePage, parseDiagnostics, "not-an-email")
			parseResponse := submitAtlasRestockForm(parseT, parsePage, parseDiagnostics)

			if parseResponse.Status() != 400 {
				parseT.Fatalf("invalid restock email produced status %d, expected 400 | %s",
					parseResponse.Status(), parseDiagnostics.Summary())
			}

			// Assert the SHAPE of the rejection, not just the status. A 400
			// with an empty body would satisfy a status-only check while
			// telling the buyer nothing.
			parseBody := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
			for _, parseWanted := range []string{
				"invalid_restock_request",
				"Enter a valid email address for restock updates.",
			} {
				assertAtlasSpecTextContains(parseT, "invalid restock response", parseBody, parseWanted, parseDiagnostics)
			}
			// It must NOT look like the success path.
			assertAtlasSpecTextMissing(parseT, "invalid restock response", parseBody, atlasRestockSuccessNotice, parseDiagnostics)

			assertAtlasSpecClean(parseT, parseDiagnostics, "atlas restock invalid email")
		})
	})

	// -------------------------------------------------------------------------
	parseT.Run("tampered_csrf_token_is_rejected", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
			parseDiagnostics.Allow("restock-requests")
			parseDiagnostics.Allow("403 (Forbidden)")

			prepareAtlasRestockForm(parseT, parsePage, parseDiagnostics, parseBaseURL)
			fillAtlasRestockForm(parseT, parsePage, parseDiagnostics, "atlas-spec-buyer@example.com")

			// Rewrite the hidden token in the live DOM. This simulates a forged
			// submission that carries a valid cookie but a wrong form field -
			// the half of double-submit validation that catches a cross-site
			// form post.
			if _, parseErr := parsePage.Evaluate(
				`(selector) => { document.querySelector(selector + ' input[name="csrf_token"]').value = "atlas-spec-tampered-token"; }`,
				atlasRestockFormSelector,
			); parseErr != nil {
				parseT.Fatalf("tamper csrf token: %v | %s", parseErr, parseDiagnostics.Summary())
			}

			parseResponse := submitAtlasRestockForm(parseT, parsePage, parseDiagnostics)
			if parseResponse.Status() != 403 {
				parseT.Fatalf("tampered csrf token produced status %d, expected 403 - CSRF form-field validation is not enforced | %s",
					parseResponse.Status(), parseDiagnostics.Summary())
			}
			parseBody := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
			// The specific error code matters: it distinguishes "rejected
			// because the token did not match" from "rejected for some other
			// reason that happens to be 403".
			assertAtlasSpecTextContains(parseT, "tampered csrf response", parseBody, "csrf_token_invalid", parseDiagnostics)
			assertAtlasSpecTextMissing(parseT, "tampered csrf response", parseBody, atlasRestockSuccessNotice, parseDiagnostics)

			assertAtlasSpecClean(parseT, parseDiagnostics, "atlas restock tampered csrf")
		})
	})

	// -------------------------------------------------------------------------
	parseT.Run("missing_csrf_cookie_is_rejected", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
			parseDiagnostics.Allow("restock-requests")
			parseDiagnostics.Allow("403 (Forbidden)")

			prepareAtlasRestockForm(parseT, parsePage, parseDiagnostics, parseBaseURL)
			fillAtlasRestockForm(parseT, parsePage, parseDiagnostics, "atlas-spec-buyer@example.com")

			// Drop every cookie while keeping the rendered token. This attacks
			// the OTHER half of the pair: a server that only checks "a token
			// was submitted" would happily accept this.
			if parseErr := parsePage.Context().ClearCookies(); parseErr != nil {
				parseT.Fatalf("clear cookies: %v | %s", parseErr, parseDiagnostics.Summary())
			}

			parseResponse := submitAtlasRestockForm(parseT, parsePage, parseDiagnostics)
			if parseResponse.Status() != 403 {
				parseT.Fatalf("submit without the atlas_csrf cookie produced status %d, expected 403 - the token is being trusted without its cookie | %s",
					parseResponse.Status(), parseDiagnostics.Summary())
			}
			parseBody := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
			assertAtlasSpecTextContains(parseT, "missing csrf cookie response", parseBody, "csrf_cookie_missing", parseDiagnostics)

			assertAtlasSpecClean(parseT, parseDiagnostics, "atlas restock missing csrf cookie")
		})
	})
}

// COVERAGE NOTE for buyer-flow/form-submit.spec.md
//
// COVERED: the restock request path in full - success round trip with a visible
// notice, validation rejection, and both CSRF rejection modes.
//
// NOT COVERED HERE: the quote-request and public-comment forms on
// /warehouses/new-jersey-hub/availability/frame-desk. They are the same
// native-POST shape as restock, so a third and fourth copy of this spec would
// add runtime without adding coverage of a different mechanism.
//
// FINDING - the plan expects behaviour Atlas does not have. form-submit.spec.md
// asserts "quote request preserves invalid input and shows field plus summary
// errors" and "public comment shows optimistic pending state". Neither is true
// of these forms today: they are plain native POSTs, so a rejected submit
// REPLACES the page with a raw JSON error document
// ({"error":"invalid_restock_request","fields":{...}}) and every keystroke the
// buyer typed is gone. There is no field-level error rendering and no
// preserved input to assert on. This spec therefore asserts what the server
// really does (case 2 above) instead of asserting a property Atlas does not
// implement. The optimistic/hydrated comment form DOES exist in the code
// (publicProductFeedbackSection) but never mounts - see
// atlas_buyer_deferred_feedback_spec_test.go.
