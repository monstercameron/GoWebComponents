//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/buyer-flow/browse.spec.md
// STORY: manifest.json -> buyer-flow -> "buyer-public-navigation"
// =============================================================================
//
// The markdown plan asks for three things:
//
//   1. the public route shell remains stable after each navigation
//   2. catalog recovery returns to /shop
//   3. checkpoints cover landing, catalog, product and warehouse availability
//
// This file makes (1) and (2) executable. (3) is screenshot capture, which the
// plan scopes to a later pass and which proves nothing about behaviour on its
// own, so it is intentionally NOT covered here - see the coverage note at the
// bottom of the file.
//
// HOW THIS SPEC AVOIDS BEING VACUOUS
//
// The trap with a multi-route walk is that every assertion is true on every
// route: the Atlas header, the "Atlas Commerce OS" wordmark and a 200 status
// are present on the landing page, the catalog, the 404 recovery page and the
// unauthenticated sign-in screen alike. A walk built on those would report
// five green routes while rendering one screen five times.
//
// So this spec asserts a MATRIX. Every route declares (a) its own <h1> and
// (b) a copy phrase that appears on that route and nowhere else in the walk.
// For each route we assert its own pair is present AND that every other
// route's pair is absent. Adding a route to the table automatically tightens
// every other route's negative check, and a regression that collapses two
// routes onto one screen fails immediately instead of passing twice.

import (
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// atlasBuyerRouteExpectation is one row of the route matrix.
//
// The Marker phrases were chosen by rendering all five routes and keeping only
// phrases that occur on exactly one of them. Prefer structural, editorial copy
// ("Why this product page is easier to use") over data that seed changes can
// move (a price, a stock count): the spec should fail when the ROUTE breaks,
// not when the fixture data is edited.
type atlasBuyerRouteExpectation struct {
	Route   string
	Heading string
	Marker  string
}

// atlasBuyerBrowseWalk is the buyer-public-navigation route list from
// manifest.json, in the order the story walks it.
//
// The last entry repeats /shop because the story's third assertion is that the
// buyer can get back to the catalog. That final hop is performed by CLICKING
// the shell nav rather than by another Goto - see the client-navigation
// section below for why that distinction carries the whole assertion.
var atlasBuyerBrowseWalk = []atlasBuyerRouteExpectation{
	{
		Route:   "/",
		Heading: "Atlas Commerce OS",
		Marker:  "Why Atlas feels ready",
	},
	{
		Route: "/shop",
		// Was "Catalog overview": the label on a second hero band that sat between
		// the page head and the first product, carrying its own stat cards (one of
		// which printed a hardcoded "24 workspace products" over a computed "4").
		// The design system pass collapsed the three stacked heroes into one, so the
		// marker is now the page head's own sentence — still unique to /shop.
		Heading: "Atlas Shop",
		Marker:  "compare price and availability line by line",
	},
	{
		Route: "/shop/frame-desk",
		// Was "Why this product page is easier to use", a strip of three cards in
		// which the page graded its own UX for the buyer. Deleted as content-free.
		// The promise-lane module is product-route-only and is real content.
		Heading: "Atlas Frame Desk",
		Marker:  "Regional promise lanes",
	},
	{
		Route:   "/warehouses/new-jersey-hub",
		Heading: "Atlas Warehouse Detail",
		Marker:  "Browse products in this region",
	},
	{
		Route: "/warehouses/new-jersey-hub/availability/frame-desk",
		// Was "Why this availability page is easier to use" — same self-grading
		// strip, same deletion. "Warehouse-specific promise" heads the band that
		// says what THIS hub can do for THIS product, and appears on no other route.
		Heading: "Atlas Warehouse Availability",
		Marker:  "Warehouse-specific promise",
	},
}

// assertAtlasBuyerRouteIsDistinct runs the matrix check for one route.
func assertAtlasBuyerRouteIsDistinct(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseExpected atlasBuyerRouteExpectation) {
	parseT.Helper()
	parseScope := "route " + parseExpected.Route

	// The heading is the route's identity claim. Compared case-insensitively
	// because InnerText returns text as rendered and Atlas uppercases some
	// headings via CSS - see atlasSpecTextContains.
	parseHeading := atlasSpecPrimaryHeading(parseT, parsePage, parseDiagnostics)
	if !atlasSpecTextContains(parseHeading, parseExpected.Heading) {
		parseT.Fatalf("%s: primary <h1> = %q, expected it to contain %q | %s",
			parseScope, parseHeading, parseExpected.Heading, parseDiagnostics.Summary())
	}

	parseText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
	assertAtlasSpecTextContains(parseT, parseScope, parseText, parseExpected.Marker, parseDiagnostics)

	// The negative half of the matrix: no OTHER route's identity may appear.
	// This is what stops the spec from passing on a screen that is not the
	// route under test.
	for _, parseOther := range atlasBuyerBrowseWalk {
		if parseOther.Route == parseExpected.Route {
			continue
		}
		if atlasSpecTextContains(parseHeading, parseOther.Heading) {
			parseT.Fatalf("%s: primary <h1> %q also matches route %s (%q) - the two routes are not rendering distinct screens | %s",
				parseScope, parseHeading, parseOther.Route, parseOther.Heading, parseDiagnostics.Summary())
		}
		assertAtlasSpecTextMissing(parseT, parseScope+" (must not look like "+parseOther.Route+")", parseText, parseOther.Marker, parseDiagnostics)
	}

	// Shell stability, the plan's first assertion. The public shell must still
	// be mounted after every navigation: the wordmark, the nav links a buyer
	// recovers through, and the client-owned shell root. If a route swap tore
	// the shell down, a route-content assertion alone would still pass.
	assertAtlasSpecTextContains(parseT, parseScope+" shell", parseText, "Atlas Commerce OS", parseDiagnostics)
	for _, parseShellSelector := range []string{
		"#atlas-shell-root",
		`#atlas-shell-root header nav a[href="/shop"]`,
		`#atlas-shell-root header nav a[href="/warehouses"]`,
	} {
		if parseCount := atlasSpecCountElements(parseT, parsePage, parseDiagnostics, parseShellSelector); parseCount == 0 {
			parseT.Fatalf("%s: public shell landmark %q is missing after navigation | %s",
				parseScope, parseShellSelector, parseDiagnostics.Summary())
		}
	}
}

// TestAtlasBuyerPublicBrowseJourney walks the buyer-public-navigation story in
// one browser session and asserts each route renders its own screen.
func TestAtlasBuyerPublicBrowseJourney(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		// Attach diagnostics before the first navigation: boot-time console
		// output is exactly what a hydration failure needs, and Playwright
		// only reports events after the listener exists.
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)

		// ONE page for the whole walk, on purpose. A fresh page per route
		// would re-download and re-instantiate the wasm bundle each time and,
		// more importantly, would not exercise route-to-route transitions -
		// where shell teardown bugs actually live.
		for _, parseExpected := range atlasBuyerBrowseWalk {
			openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, parseExpected.Route, 0)
			assertAtlasBuyerRouteIsDistinct(parseT, parsePage, parseDiagnostics, parseExpected)
			parseT.Logf("atlas buyer browse: %s rendered %q", parseExpected.Route, atlasSpecPrimaryHeading(parseT, parsePage, parseDiagnostics))
		}

		// --- catalog recovery, the plan's second assertion ---
		//
		// The buyer is on the availability route (the deepest point of the
		// walk) and must be able to get back to the catalog through the shell.
		//
		// We plant a marker on `window` first. Atlas routes client-side, so a
		// correct nav click swaps the route WITHOUT a document load and the
		// marker survives; a regression that degrades the link into a full
		// page load wipes it. Without this check, "URL is /shop and the
		// heading says Atlas Shop" would pass either way and the spec would be
		// blind to losing client-side routing entirely.
		if _, parseErr := parsePage.Evaluate(`() => { window.__atlasBrowseSpecMarker = "same-document"; }`); parseErr != nil {
			parseT.Fatalf("plant same-document marker: %v | %s", parseErr, parseDiagnostics.Summary())
		}

		if parseErr := parsePage.Click(`#atlas-shell-root header nav a[href="/shop"]`); parseErr != nil {
			parseT.Fatalf("click shell nav back to catalog: %v | %s", parseErr, parseDiagnostics.Summary())
		}

		// Wait on the OUTCOME (catalog heading painted), not on a sleep: the
		// client route swap is asynchronous and a fixed delay would be either
		// flaky or wasteful.
		waitForAtlasSpecCondition(parseT, parsePage, parseDiagnostics,
			"the catalog heading after clicking the shell Shop link",
			`() => {
				const heading = document.querySelector("h1");
				return !!heading && heading.textContent.toLowerCase().includes("atlas shop");
			}`)

		if !strings.HasSuffix(parsePage.URL(), "/shop") {
			parseT.Fatalf("catalog recovery: expected URL to end with /shop, got %s | %s", parsePage.URL(), parseDiagnostics.Summary())
		}

		parseMarker, parseMarkerErr := parsePage.Evaluate(`() => window.__atlasBrowseSpecMarker || "document-reloaded"`)
		if parseMarkerErr != nil {
			parseT.Fatalf("read same-document marker: %v | %s", parseMarkerErr, parseDiagnostics.Summary())
		}
		if parseMarker != "same-document" {
			parseT.Fatalf("catalog recovery performed a full document load (marker=%v); Atlas is expected to route client-side from the shell nav | %s",
				parseMarker, parseDiagnostics.Summary())
		}

		// Re-run the full matrix on the recovered catalog. Arriving at /shop by
		// client navigation must produce the same screen as arriving by direct
		// entry - a client-routed render that silently drops catalog content
		// would otherwise slip through.
		assertAtlasBuyerRouteIsDistinct(parseT, parsePage, parseDiagnostics, atlasBuyerBrowseWalk[1])

		assertAtlasSpecClean(parseT, parseDiagnostics, "atlas buyer public browse journey")
	})
}

// COVERAGE NOTE for buyer-flow/browse.spec.md
//
// COVERED: public route shell stability across all five story routes; per-route
// distinctness; client-side catalog recovery from the deepest route.
//
// NOT COVERED HERE: the four screenshot checkpoints (public-landing-start,
// public-catalog-browsing, public-product-decision,
// public-warehouse-availability). Screenshot capture needs a baseline set and
// a diffing policy, and the manifest's own run order puts design-parity after
// core behaviour passes. A screenshot with no baseline asserts nothing, so
// adding one here would only add runtime.
//
// NOT COVERED HERE: buyer-flow/mobile-nav.spec.md. It needs a mobile viewport
// matrix and open/close assertions on the "Menu" button; it is a separate
// spec in the plan and remains unimplemented.
