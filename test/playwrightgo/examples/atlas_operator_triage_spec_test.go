//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/operator-flow/dashboard-triage.spec.md
//       examples/tests/atlas-commerce-os/operator-flow/inventory-warehouse-workflow.spec.md
//         (assertion 1: "saved-view and table filters keep route-shell stability")
// STORY: manifest.json -> operator-flow -> "operator-alert-driven-workflow"
// =============================================================================
//
// SUBJECT: one operator session walking dashboard -> alert-driven inventory
// queue -> SKU lane detail, asserting each hop renders its own surface AND that
// the filter carried by the alert link actually changes the data.
//
// WHAT MAKES THIS SPEC WORTH ITS RUNTIME
//
// A route-list smoke test over /app/dashboard, /app/inventory and
// /app/inventory/frame-desk would pass if all three rendered the same screen -
// which is exactly what happens when the session is missing (all three become
// the sign-in page) or when the router falls back to a default child. So this
// spec:
//
//   1. asserts a per-surface marker AND the absence of the other surfaces'
//      markers, so two hops cannot both be satisfied by one screen;
//   2. asserts the operator session is visible on every hop, because the
//      unauthenticated failure mode is a 200 page that hydrates cleanly;
//   3. asserts the alert link's status filter changes the QUEUE CONTENTS -
//      the filtered set must be a strict subset of the unfiltered set and
//      every remaining row must carry the filtered status. A filter that is
//      wired to nothing renders an identical table, and that is precisely the
//      bug an "it still renders" check cannot see.

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

const (
	// Inventory queue rows. Atlas renders the dense queue as a real <table>,
	// so tbody rows are a stable structural handle - unlike the generated
	// element ids (gwc-*), which change with hook order.
	atlasInventoryRowSelector = "table tbody tr"

	// Surface markers: each string appears on exactly one of the three hops.
	atlasTriageDashboardMarker = "Demand and operations overview"
	atlasTriageInventoryMarker = "Inventory triage shell"
	atlasTriageSKUMarker       = "Inventory lane workspace"

	// The dashboard quick action that represents "an alert leads to inventory
	// work". Selecting the anchor by href keeps the spec tied to the product's
	// routing contract rather than to link copy.
	atlasPromiseRiskAlertHref = "/app/inventory?status=promise_risk"
)

// atlasInventoryRow is one row of the inventory queue, split into the two
// things a filter assertion needs to reason about separately.
//
// SKU is the row's IDENTITY, taken from its "Open SKU" link
// (/app/inventory/<sku>) rather than from its text. Text is the row's visible
// content, used for status assertions.
//
// The split exists because of a mistake worth not repeating. The first version
// of this spec asserted that each filtered row's full text also appeared in the
// unfiltered set. That failed - correctly, and for a product reason that is not
// a bug: filtering by lane status also narrows the LANES aggregated into each
// surviving row, so the same SKU legitimately renders "1 total lanes ... 3
// available" when filtered and "3 total lanes ... 22 available" when not. Row
// text is therefore the wrong identity key; the SKU is the right one.
type atlasInventoryRow struct {
	SKU  string
	Text string
}

// readAtlasInventoryRows reads the inventory queue as identity+text pairs.
//
// Reading rows rather than counting them is deliberate: a count tells you the
// table changed, the content tells you it changed CORRECTLY. A filter bug that
// drops the wrong rows still changes the count.
func readAtlasInventoryRows(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) []atlasInventoryRow {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(
		`(selector) => Array.from(document.querySelectorAll(selector)).map(row => {
			const link = row.querySelector('a[href^="/app/inventory/"]');
			const href = link ? link.getAttribute("href") : "";
			return {
				sku: href ? href.split("/").pop() : "",
				text: row.innerText.replace(/\s+/g, " ").trim()
			};
		})`,
		atlasInventoryRowSelector)
	if parseErr != nil {
		parseT.Fatalf("read inventory queue rows: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	parseItems, parseOK := parseValue.([]interface{})
	if !parseOK {
		parseT.Fatalf("unexpected row payload type %T | %s", parseValue, parseDiagnostics.Summary())
	}
	parseRows := make([]atlasInventoryRow, 0, len(parseItems))
	for _, parseItem := range parseItems {
		parseFields, parseFieldsOK := parseItem.(map[string]interface{})
		if !parseFieldsOK {
			continue
		}
		parseSKU, _ := parseFields["sku"].(string)
		parseText, _ := parseFields["text"].(string)
		if strings.TrimSpace(parseSKU) == "" {
			// A queue row with no SKU link cannot be identified or handed off
			// from, which is itself a defect worth failing on rather than
			// quietly skipping.
			parseT.Fatalf("inventory queue row %q has no /app/inventory/<sku> link, so it offers no handoff into lane work | %s",
				parseText, parseDiagnostics.Summary())
		}
		parseRows = append(parseRows, atlasInventoryRow{SKU: parseSKU, Text: parseText})
	}
	return parseRows
}

// atlasInventorySKUs projects rows down to their identities.
func atlasInventorySKUs(parseRows []atlasInventoryRow) []string {
	parseSKUs := make([]string, 0, len(parseRows))
	for _, parseRow := range parseRows {
		parseSKUs = append(parseSKUs, parseRow.SKU)
	}
	return parseSKUs
}

// atlasInventoryContainsSKU reports whether a SKU is present in a row set.
func atlasInventoryContainsSKU(parseRows []atlasInventoryRow, parseSKU string) bool {
	for _, parseRow := range parseRows {
		if parseRow.SKU == parseSKU {
			return true
		}
	}
	return false
}

// assertAtlasSpecInternalSurface asserts the page is the expected internal
// surface, is authenticated, and is not one of the other surfaces.
func assertAtlasSpecInternalSurface(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseScope string, parseHeading string, parseMarker string, parseForeignMarkers ...string) {
	parseT.Helper()

	// Authentication first. If the session is gone, every other assertion
	// below is being made against the sign-in screen and the failure message
	// should say so rather than complaining about missing copy.
	assertAtlasSpecOperatorSessionVisible(parseT, parsePage, parseDiagnostics, parseScope)

	parseActualHeading := atlasSpecPrimaryHeading(parseT, parsePage, parseDiagnostics)
	if !atlasSpecTextContains(parseActualHeading, parseHeading) {
		parseT.Fatalf("%s: primary <h1> = %q, expected it to contain %q | %s",
			parseScope, parseActualHeading, parseHeading, parseDiagnostics.Summary())
	}

	parseText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
	assertAtlasSpecTextContains(parseT, parseScope, parseText, parseMarker, parseDiagnostics)
	for _, parseForeign := range parseForeignMarkers {
		assertAtlasSpecTextMissing(parseT, parseScope, parseText, parseForeign, parseDiagnostics)
	}

	// Internal shell stability: the workspace nav must survive every hop. It
	// is the operator's only way out of a route, so losing it is a dead end
	// even when the route content is correct.
	for _, parseShellSelector := range []string{
		"#atlas-shell-root",
		`#atlas-shell-root a[href="/app/dashboard"]`,
		`#atlas-shell-root a[href="/app/inventory"]`,
		`#atlas-shell-root a[href="/app/settings"]`,
	} {
		if parseCount := atlasSpecCountElements(parseT, parsePage, parseDiagnostics, parseShellSelector); parseCount == 0 {
			parseT.Fatalf("%s: internal shell landmark %q is missing | %s", parseScope, parseShellSelector, parseDiagnostics.Summary())
		}
	}
}

// TestAtlasOperatorDashboardToInventoryTriage walks the alert-driven operator
// workflow and proves the alert's filter really narrows the queue.
func TestAtlasOperatorDashboardToInventoryTriage(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)

		// The session is seeded as a cookie here rather than driven through the
		// form. Sign-in has its own spec (TestAtlasOperatorMockSignInReaches-
		// Dashboard); re-driving it in every operator spec would only add
		// seconds. What must NOT be skipped is asserting the session is
		// actually live on each surface - assertAtlasSpecInternalSurface does
		// that, because a cookie typo produces the sign-in screen with a 200.
		seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "ops_lead")

		// ---------------------------------------------------------------------
		// BASELINE - the unfiltered inventory queue.
		// ---------------------------------------------------------------------
		//
		// Captured BEFORE the dashboard hop so that the later filtered set has
		// something honest to be compared against. Without a baseline,
		// "the filtered queue has 2 rows" is a magic number that silently
		// becomes wrong when the seed data changes; with one, the assertion is
		// relational ("strictly fewer rows, all of them promise risk") and
		// survives seed edits while still catching a dead filter.
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/inventory", 0)
		waitForAtlasSpecCondition(parseT, parsePage, parseDiagnostics,
			"the unfiltered inventory queue table to have rows",
			fmt.Sprintf(`() => document.querySelectorAll(%q).length > 0`, atlasInventoryRowSelector))
		parseUnfilteredRows := readAtlasInventoryRows(parseT, parsePage, parseDiagnostics)
		parseT.Logf("atlas inventory baseline rows=%d skus=%v", len(parseUnfilteredRows), atlasInventorySKUs(parseUnfilteredRows))

		// A queue with a single row cannot demonstrate filtering at all, so
		// fail loudly rather than run a test that cannot detect the bug.
		if len(parseUnfilteredRows) < 2 {
			parseT.Fatalf("the unfiltered inventory queue has %d row(s); this spec needs at least 2 to prove filtering narrows it | %s",
				len(parseUnfilteredRows), parseDiagnostics.Summary())
		}

		// ---------------------------------------------------------------------
		// HOP 1 - the operator dashboard.
		// ---------------------------------------------------------------------
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/dashboard", 0)
		assertAtlasSpecInternalSurface(parseT, parsePage, parseDiagnostics,
			"operator dashboard", "Atlas Ops Dashboard", atlasTriageDashboardMarker,
			atlasTriageInventoryMarker, atlasTriageSKUMarker)

		// Triage-surface structure from dashboard-triage.spec.md: the dashboard
		// is meant to read as a triage surface, so the summary band, the action
		// cluster and the activity feed all have to be there. Asserting the
		// heading alone would let a dashboard that lost its entire action
		// cluster still pass.
		parseDashboardText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
		for _, parseSection := range []string{"Dashboard summary band", "Action cluster", "Activity feed", "Open alerts"} {
			assertAtlasSpecTextContains(parseT, "operator dashboard sections", parseDashboardText, parseSection, parseDiagnostics)
		}

		// ---------------------------------------------------------------------
		// HOP 2 - follow the promise-risk alert into the inventory queue.
		// ---------------------------------------------------------------------
		//
		// This is the story's core claim: an alert on the dashboard leads to
		// the inventory work that resolves it. We click the real anchor instead
		// of navigating by URL, because navigating by URL would still pass if
		// the dashboard rendered no link at all.
		parseAlertLink := parsePage.Locator(fmt.Sprintf(`a[href=%q]`, atlasPromiseRiskAlertHref))
		parseAlertCount, parseAlertErr := parseAlertLink.Count()
		if parseAlertErr != nil {
			parseT.Fatalf("count promise-risk alert links: %v | %s", parseAlertErr, parseDiagnostics.Summary())
		}
		if parseAlertCount == 0 {
			parseT.Fatalf("the dashboard renders no %s alert link, so alert-driven triage is unreachable | %s",
				atlasPromiseRiskAlertHref, parseDiagnostics.Summary())
		}

		// Plant a same-document marker so we can tell client routing from a
		// full reload. Atlas routes internally; a regression to full document
		// loads would still land on the right URL with the right content, so
		// URL and content alone cannot see it.
		if _, parseErr := parsePage.Evaluate(`() => { window.__atlasTriageSpecMarker = "same-document"; }`); parseErr != nil {
			parseT.Fatalf("plant same-document marker: %v | %s", parseErr, parseDiagnostics.Summary())
		}

		if parseErr := parseAlertLink.First().Click(); parseErr != nil {
			parseT.Fatalf("click the promise-risk alert link: %v | %s", parseErr, parseDiagnostics.Summary())
		}

		// Wait for the inventory surface to be the thing on screen. Waiting on
		// the marker copy (rather than on the URL) is what distinguishes
		// "the router changed the address" from "the new route rendered".
		waitForAtlasSpecVisibleText(parseT, parsePage, parseDiagnostics,
			"the inventory triage surface after following the dashboard alert",
			atlasTriageInventoryMarker)

		if !strings.Contains(parsePage.URL(), "status=promise_risk") {
			parseT.Fatalf("after following the alert the URL is %s; the promise-risk filter was dropped | %s",
				parsePage.URL(), parseDiagnostics.Summary())
		}
		parseMarker, _ := parsePage.Evaluate(`() => window.__atlasTriageSpecMarker || "document-reloaded"`)
		if parseMarker != "same-document" {
			parseT.Fatalf("the dashboard alert performed a full document load (marker=%v); internal navigation is expected to stay client-side | %s",
				parseMarker, parseDiagnostics.Summary())
		}

		assertAtlasSpecInternalSurface(parseT, parsePage, parseDiagnostics,
			"filtered inventory queue", "Atlas Inventory", atlasTriageInventoryMarker,
			atlasTriageDashboardMarker, atlasTriageSKUMarker)

		// --- the assertion this whole spec is built around ---
		waitForAtlasSpecCondition(parseT, parsePage, parseDiagnostics,
			"the filtered inventory queue table to have rows",
			fmt.Sprintf(`() => document.querySelectorAll(%q).length > 0`, atlasInventoryRowSelector))
		parseFilteredRows := readAtlasInventoryRows(parseT, parsePage, parseDiagnostics)
		parseT.Logf("atlas inventory promise-risk rows=%d skus=%v", len(parseFilteredRows), atlasInventorySKUs(parseFilteredRows))

		// (a) the filter removed something. If the query parameter were
		//     ignored, this is the assertion that fails.
		if len(parseFilteredRows) >= len(parseUnfilteredRows) {
			parseT.Fatalf("promise-risk filter did not narrow the queue: %d rows filtered vs %d unfiltered - the status query parameter is not reaching the query | %s",
				len(parseFilteredRows), len(parseUnfilteredRows), parseDiagnostics.Summary())
		}
		// (b) it did not remove everything. An empty table would technically
		//     satisfy (a) while proving the filter is broken in the other
		//     direction.
		if len(parseFilteredRows) == 0 {
			parseT.Fatalf("promise-risk filter emptied the inventory queue | %s", parseDiagnostics.Summary())
		}
		// (c) every surviving row genuinely carries the filtered status. This
		//     is what makes it a correctness check rather than a count check.
		for _, parseRow := range parseFilteredRows {
			if !atlasSpecTextContains(parseRow.Text, "promise risk") {
				parseT.Fatalf("row %q (sku %s) survived the promise-risk filter without a promise-risk status | %s",
					parseRow.Text, parseRow.SKU, parseDiagnostics.Summary())
			}
		}
		// (d) the filtered set is a subset of the baseline BY SKU. Catches a
		//     filter that swaps in a different dataset instead of narrowing this
		//     one. Compared by SKU, not by row text, because the filter also
		//     narrows the lanes aggregated into each row - so a surviving SKU
		//     legitimately shows different lane counts and unit totals.
		for _, parseRow := range parseFilteredRows {
			if !atlasInventoryContainsSKU(parseUnfilteredRows, parseRow.SKU) {
				parseT.Fatalf("filtered SKU %q was not present in the unfiltered queue %v; the filter is not narrowing the same dataset | %s",
					parseRow.SKU, atlasInventorySKUs(parseUnfilteredRows), parseDiagnostics.Summary())
			}
		}
		// (e) at least one baseline SKU that is NOT promise risk was actually
		//     dropped. Without this, a dataset where everything is promise risk
		//     would make (a)-(d) pass for free, and the spec would be unable to
		//     tell a working filter from a coincidence.
		parseDroppedNonRisk := ""
		for _, parseBaselineRow := range parseUnfilteredRows {
			if atlasSpecTextContains(parseBaselineRow.Text, "promise risk") {
				continue
			}
			if !atlasInventoryContainsSKU(parseFilteredRows, parseBaselineRow.SKU) {
				parseDroppedNonRisk = parseBaselineRow.SKU
				break
			}
		}
		if parseDroppedNonRisk == "" {
			parseT.Fatalf("no non-promise-risk SKU was dropped by the filter (baseline=%v filtered=%v); this spec cannot distinguish a working filter from a no-op on the current dataset | %s",
				atlasInventorySKUs(parseUnfilteredRows), atlasInventorySKUs(parseFilteredRows), parseDiagnostics.Summary())
		}
		parseT.Logf("atlas inventory filter dropped non-risk sku=%s", parseDroppedNonRisk)

		// ---------------------------------------------------------------------
		// HOP 3 - open the SKU lane workspace from a queue row.
		// ---------------------------------------------------------------------
		//
		// The queue's "Open SKU" action is the handoff from triage to the fix.
		// frame-desk is a seeded SKU that appears in the promise-risk set, so
		// its row link is reachable from the filtered queue we are on.
		parseSKULink := parsePage.Locator(`a[href="/app/inventory/frame-desk"]`)
		parseSKUCount, parseSKUCountErr := parseSKULink.Count()
		if parseSKUCountErr != nil {
			parseT.Fatalf("count frame-desk SKU links: %v | %s", parseSKUCountErr, parseDiagnostics.Summary())
		}
		if parseSKUCount == 0 {
			parseT.Fatalf("the filtered inventory queue offers no link into /app/inventory/frame-desk, so triage has no handoff into lane work | %s",
				parseDiagnostics.Summary())
		}
		if parseErr := parseSKULink.First().Click(); parseErr != nil {
			parseT.Fatalf("click the frame-desk SKU link: %v | %s", parseErr, parseDiagnostics.Summary())
		}
		waitForAtlasSpecVisibleText(parseT, parsePage, parseDiagnostics,
			"the SKU lane workspace after opening a queue row",
			atlasTriageSKUMarker)

		assertAtlasSpecInternalSurface(parseT, parsePage, parseDiagnostics,
			"sku lane workspace", "Atlas SKU Detail", atlasTriageSKUMarker,
			atlasTriageDashboardMarker, atlasTriageInventoryMarker)

		// Route-origin context, from the story's "originating alert context is
		// preserved" assertion: the SKU route must be about frame-desk
		// specifically and must offer the way back to the queue.
		parseSKUText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
		assertAtlasSpecTextContains(parseT, "sku lane workspace", parseSKUText, "SKU frame-desk", parseDiagnostics)
		assertAtlasSpecTextContains(parseT, "sku lane workspace", parseSKUText, "Lane roster", parseDiagnostics)
		assertAtlasSpecTextContains(parseT, "sku lane workspace", parseSKUText, "Back to inventory queue", parseDiagnostics)

		assertAtlasSpecClean(parseT, parseDiagnostics, "atlas operator dashboard to inventory triage")
	})
}

// COVERAGE NOTE
//
// COVERED (dashboard-triage.spec.md): the mock session reaching the dashboard,
// alert-driven navigation into inventory with the alert's filter intact, and
// route-origin context on the SKU route.
//
// COVERED (inventory-warehouse-workflow.spec.md): route-shell stability across
// the queue and SKU routes, and that table filters change the data.
//
// NOT COVERED HERE: the threshold overlay save and its activity-history append;
// the warehouse-item update returning to warehouse context
// (/app/warehouses/{id}/items/{sku}); and the "threshold update refreshes
// dashboard summaries after returning" assertion. Those are mutation flows
// against a SHARED sqlite file (examples/server/atlas-commerce-os/server/data/
// atlas-commerce-os.db, with no env override) so they would leave persistent
// residue in a checked-in fixture and could interfere with a concurrently
// running Atlas server. They need an isolated database path before they can be
// automated safely.
