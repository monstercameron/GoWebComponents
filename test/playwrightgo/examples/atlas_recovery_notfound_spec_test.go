//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/recovery-edge-cases/README.md
//       manifest.json -> failure_recovery_stories -> "recovery-not-found"
// =============================================================================
//
// The manifest states three assertions for this story:
//
//   - status is 404
//   - a route-appropriate recovery heading is visible
//   - a primary recovery action is available
//
// All three are covered here, plus one the manifest implies but does not spell
// out: the recovery action must actually WORK. A recovery page that renders a
// dead-end button is worse than no recovery page, because it looks handled.
//
// WHY A 404 SPEC IS EASY TO GET WRONG
//
// Atlas answers unknown GET routes by buffering the mux response, noticing the
// 404, and rendering a full recovery DOCUMENT - shell, hero, bootstrap payload
// and all - with a 404 status. That means the page looks like a normal Atlas
// route and hydrates normally. Two traps follow:
//
//   1. Asserting only "status == 404" would pass for a blank page, a stack
//      trace, or Go's default "404 page not found" text. So this spec asserts
//      the recovery CONTENT and the recovery ACTION as well.
//   2. Asserting only content would pass if Atlas served the recovery screen
//      with a 200, which would be a real (and SEO-relevant) defect. So the
//      status is asserted too.
//
// The spec also asserts that the missing-product route does NOT render product
// content. That is the failure this catches: a route that silently falls back
// to a placeholder product, telling the buyer a nonexistent SKU exists.
//
// SURFACE-SPECIFIC RECOVERY
//
// Atlas differentiates: public unknown routes recover to /shop ("Back to
// shop"), internal ones recover to /app/dashboard ("Back to dashboard"). A
// single shared assertion would hide a regression that sent operators to the
// storefront, so the table below encodes the expected destination per surface.

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// atlasRecoveryCase is one not-found route and the recovery it must offer.
type atlasRecoveryCase struct {
	// Name is the subtest name.
	Name string
	// Route is the unknown route to request.
	Route string
	// Heading is the <h1> the recovery document must render.
	Heading string
	// RecoveryHref is where the primary recovery action must point.
	RecoveryHref string
	// RecoveryLabel is the visible label of that action.
	RecoveryLabel string
	// RecoveredMarker is content that must appear AFTER following the action,
	// proving the recovery destination really rendered.
	RecoveredMarker string
	// ForbiddenMarkers must NOT appear on the recovery page. These are the
	// "pretend the record exists" failure modes.
	ForbiddenMarkers []string
	// RequiresSession is true for /app routes: without a session they redirect
	// to sign-in with a 200 and never reach the recovery renderer at all.
	RequiresSession bool
}

var atlasRecoveryCases = []atlasRecoveryCase{
	{
		Name:          "unknown_public_route",
		Route:         "/not-a-real-route",
		Heading:       "Atlas Route Not Found",
		RecoveryHref:  "/shop",
		RecoveryLabel: "Back to shop",
		// The catalog's own marker: proves we landed on the real catalog and
		// not on another recovery page.
		RecoveredMarker: "Catalog overview",
	},
	{
		Name:            "missing_public_product",
		Route:           "/shop/not-a-real-product",
		Heading:         "Atlas Route Not Found",
		RecoveryHref:    "/shop",
		RecoveryLabel:   "Back to shop",
		RecoveredMarker: "Catalog overview",
		// A missing product must not borrow a real product's page. These are
		// the product-detail markers from the frame-desk route; if any of them
		// appear here, the route fell back to rendering a product.
		ForbiddenMarkers: []string{"Why this product page is easier to use", "Starting at"},
	},
	{
		Name:            "missing_internal_record",
		Route:           "/app/inventory/not-a-real-sku",
		Heading:         "Atlas Internal Route Not Found",
		RecoveryHref:    "/app/dashboard",
		RecoveryLabel:   "Back to dashboard",
		RecoveredMarker: "Demand and operations overview",
		// An unknown SKU must not render the lane workspace with empty or
		// borrowed data.
		ForbiddenMarkers: []string{"Inventory lane workspace", "Lane roster"},
		RequiresSession:  true,
	},
}

// TestAtlasRecoveryNotFoundRoutes covers the recovery-not-found story across
// both surfaces.
func TestAtlasRecoveryNotFoundRoutes(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	for _, parseCase := range atlasRecoveryCases {
		parseCase := parseCase
		parseT.Run(parseCase.Name, func(parseT *testing.T) {
			withExamplesPage(parseT, func(parsePage playwright.Page) {
				parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)

				// The 404 IS the subject here, so it is allowed rather than
				// treated as noise. Both the response entry and the console
				// entry Chromium emits for a 404 main document are covered.
				// Note this allowlist is narrow on purpose: it names the route
				// under test, so an unrelated 404 elsewhere on the page still
				// fails the run.
				parseDiagnostics.Allow(parseCase.Route)
				parseDiagnostics.Allow("404 (Not Found)")

				if parseCase.RequiresSession {
					// Without a session an /app 404 never reaches the recovery
					// renderer: RequireInternalSession redirects to sign-in
					// first, which answers 200. The spec would then be
					// asserting against the sign-in screen.
					seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "ops_lead")
				}

				// Allow the 404 as the response status; openAtlasSpecRoute
				// still fails on any other >=400.
				parseResponse := openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, parseCase.Route, 404)

				// --- assertion 1: status is 404 ---
				//
				// Checked explicitly (not just "not 200"): a soft 404 served as
				// 200 is a real defect, and a 500 would mean the recovery path
				// itself blew up.
				if parseResponse.Status() != 404 {
					parseT.Fatalf("route %s returned status %d, expected 404 | %s",
						parseCase.Route, parseResponse.Status(), parseDiagnostics.Summary())
				}

				// --- assertion 2: a route-appropriate recovery heading ---
				parseHeading := atlasSpecPrimaryHeading(parseT, parsePage, parseDiagnostics)
				if !atlasSpecTextContains(parseHeading, parseCase.Heading) {
					parseT.Fatalf("route %s recovery heading = %q, expected it to contain %q | %s",
						parseCase.Route, parseHeading, parseCase.Heading, parseDiagnostics.Summary())
				}

				parseText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
				// The recovery card copy. This is what distinguishes an Atlas
				// recovery screen from a bare framework 404 body.
				assertAtlasSpecTextContains(parseT, "recovery page "+parseCase.Route, parseText, "Route not found", parseDiagnostics)
				assertAtlasSpecTextContains(parseT, "recovery page "+parseCase.Route, parseText, "not part of the current demo route set", parseDiagnostics)

				// The route must not pretend the missing record exists.
				for _, parseForbidden := range parseCase.ForbiddenMarkers {
					assertAtlasSpecTextMissing(parseT, "recovery page "+parseCase.Route, parseText, parseForbidden, parseDiagnostics)
				}

				// --- assertion 3: a primary recovery action is available ---
				parseRecoverySelector := fmt.Sprintf(`a[href=%q]`, parseCase.RecoveryHref)
				parseRecoveryLink := parsePage.Locator(parseRecoverySelector)
				parseCount, parseCountErr := parseRecoveryLink.Count()
				if parseCountErr != nil {
					parseT.Fatalf("count recovery links on %s: %v | %s", parseCase.Route, parseCountErr, parseDiagnostics.Summary())
				}
				if parseCount == 0 {
					parseT.Fatalf("route %s offers no recovery action pointing at %s | %s",
						parseCase.Route, parseCase.RecoveryHref, parseDiagnostics.Summary())
				}
				// The label matters as much as the href: the shell nav also
				// links to /shop and /app/dashboard on every page, so a
				// href-only check would be satisfied by the chrome and would
				// pass even if the recovery card rendered no button at all.
				assertAtlasSpecTextContains(parseT, "recovery action label on "+parseCase.Route, parseText, parseCase.RecoveryLabel, parseDiagnostics)

				// --- the assertion the manifest implies: recovery WORKS ---
				//
				// Click the labelled recovery action, not just any link with
				// that href, so we exercise the affordance the recovery card
				// actually offers the user.
				parseLabelledRecovery := parsePage.Locator(parseRecoverySelector, playwright.PageLocatorOptions{
					HasText: parseCase.RecoveryLabel,
				})
				parseLabelledCount, parseLabelledErr := parseLabelledRecovery.Count()
				if parseLabelledErr != nil {
					parseT.Fatalf("count labelled recovery actions on %s: %v | %s", parseCase.Route, parseLabelledErr, parseDiagnostics.Summary())
				}
				if parseLabelledCount == 0 {
					parseT.Fatalf("route %s renders no %q action pointing at %s (the href exists only in the shell nav) | %s",
						parseCase.Route, parseCase.RecoveryLabel, parseCase.RecoveryHref, parseDiagnostics.Summary())
				}
				if parseErr := parseLabelledRecovery.First().Click(); parseErr != nil {
					parseT.Fatalf("click the recovery action on %s: %v | %s", parseCase.Route, parseErr, parseDiagnostics.Summary())
				}

				// Wait for the destination's own content, not for the URL. A
				// URL change with no render is the dead-end case this spec is
				// here to catch.
				waitForAtlasSpecVisibleText(parseT, parsePage, parseDiagnostics,
					"the recovery destination content after following "+parseCase.RecoveryLabel,
					parseCase.RecoveredMarker)

				if !strings.Contains(parsePage.URL(), parseCase.RecoveryHref) {
					parseT.Fatalf("recovery from %s landed on %s, expected %s | %s",
						parseCase.Route, parsePage.URL(), parseCase.RecoveryHref, parseDiagnostics.Summary())
				}
				// And the recovery screen must be gone - otherwise "the
				// destination content appeared" could be true of a page that
				// rendered both.
				parseRecoveredText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
				assertAtlasSpecTextMissing(parseT, "recovery destination from "+parseCase.Route, parseRecoveredText, "Route not found", parseDiagnostics)

				assertAtlasSpecClean(parseT, parseDiagnostics, "atlas recovery "+parseCase.Route)
			})
		})
	}
}

// COVERAGE NOTE for recovery-edge-cases
//
// COVERED (recovery-not-found): unknown public route, missing public product,
// missing internal record - each with status, recovery heading, recovery copy,
// a working surface-appropriate recovery action, and a guard against rendering
// borrowed content for a record that does not exist.
//
// NOT COVERED HERE, and why:
//   - recovery-not-found also lists /warehouses/not-a-real-warehouse and
//     /warehouses/new-jersey-hub/availability/not-a-real-sku. Both were checked
//     manually and behave identically to the public cases above (404 + "Back to
//     shop"), so they would add runtime without adding a distinct mechanism.
//   - recovery-server-error needs a way to force the store closed
//     ("status is 500 when the test store is forced closed"). Atlas exposes no
//     such switch to a browser test, so it cannot be automated from here yet.
//   - recovery-network-interruption, recovery-duplicate-submit-timeout and
//     recovery-async-navigation-race all need request interception/throttling
//     plus mutation routes against the shared checked-in sqlite fixture.
//   - recovery-invalid-input-state (malformed query params) is partially
//     exercised elsewhere: the inventory triage spec drives ?status=promise_risk
//     as a valid value, but no spec yet asserts fallback for unsupported values
//     such as ?sort=unsupported or ?density=giant.
