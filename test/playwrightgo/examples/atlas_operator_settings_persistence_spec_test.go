//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/operator-flow/settings-persistence.spec.md
//       assertion 1 - "theme, locale, density, and default warehouse persist
//                      after save"
//       assertion 3 - "direct-entry reload preserves the selected settings
//                      across route families"
// STORY: manifest.json -> manual_stories -> "manual-settings-preference-persistence"
// =============================================================================
//
// SUBJECT: saving operator preferences and proving they survive a full document
// reload and a hop into a different route family.
//
// WHY THE ASSERTIONS ARE SHAPED THE WAY THEY ARE
//
// "Persistence" is the easiest thing in a UI to fake-verify. Three different
// mechanisms could satisfy a naive check, and only one of them is persistence:
//
//   - the form still shows what you typed because the DOM was never replaced;
//   - the value came back from client memory (an atom, localStorage) that a
//     real new visitor would not have;
//   - the value came back from the server, which is what we actually want.
//
// This spec pins the third by asserting on things only the SERVER can produce:
//
//   1. after saving, we do a fresh Goto (a new document, new wasm instance) and
//      read the form values the server rendered into the bootstrap payload;
//   2. we assert the <html> class attribute, which Atlas stamps during SSR from
//      the same preference record (atlas-theme-dark|light plus
//      atlas-density-compact|comfortable) so direct-entry pages do not flash
//      default styling. That attribute exists before any client code runs, so
//      no amount of client-side state can forge it;
//   3. we then load a DIFFERENT route family (/app/dashboard) and assert the
//      same server-side witness, which is the plan's "across route families"
//      requirement.
//
// AND IT IS RELATIVE, NOT ABSOLUTE
//
// The spec reads the current preferences first and saves the OPPOSITE of each
// one, then asserts the opposite came back. It never asserts a fixed value.
// Two reasons: the preference record lives in a checked-in shared sqlite file
// so its starting state is whatever the last run left behind; and a flip-based
// assertion cannot be satisfied by a constant. If the save path were a no-op,
// asserting "density == comfortable" might pass by luck - asserting "density
// changed to the value it was not" cannot.
//
// The original values are restored at the end, so the shared fixture is left as
// found.

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

// atlasPreferencesFormSelector targets the operator preference form by the
// endpoint it posts to - a product contract, unlike the generated gwc-* ids.
const atlasPreferencesFormSelector = `form[action="/api/app/preferences"]`

// atlasPreferenceSnapshot is the four-field preference state the plan names.
type atlasPreferenceSnapshot struct {
	Theme            string
	Locale           string
	Density          string
	DefaultWarehouse string
}

func (parseP atlasPreferenceSnapshot) String() string {
	return fmt.Sprintf("theme=%s locale=%s density=%s default_warehouse=%s",
		parseP.Theme, parseP.Locale, parseP.Density, parseP.DefaultWarehouse)
}

// flipped returns a snapshot that differs in ALL FOUR fields.
//
// Every value is chosen from the server's own accepted set
// (validatePreferencesRequest allows theme dark|light, locale en|fr|ar, density
// compact|comfortable) so a rejected save can only mean a real defect, never a
// bad test fixture.
func (parseP atlasPreferenceSnapshot) flipped() atlasPreferenceSnapshot {
	parseFlip := func(parseCurrent string, parseA string, parseB string) string {
		if strings.EqualFold(strings.TrimSpace(parseCurrent), parseA) {
			return parseB
		}
		return parseA
	}
	return atlasPreferenceSnapshot{
		Theme:            parseFlip(parseP.Theme, "dark", "light"),
		Locale:           parseFlip(parseP.Locale, "en", "fr"),
		Density:          parseFlip(parseP.Density, "compact", "comfortable"),
		DefaultWarehouse: parseFlip(parseP.DefaultWarehouse, "new-jersey-hub", "illinois-hub"),
	}
}

// expectedDocumentClass is the <html> class Atlas must render for a snapshot.
// Only theme and density are represented there; locale rides the lang
// attribute and the default warehouse is data, not presentation.
func (parseP atlasPreferenceSnapshot) expectedDocumentClass() []string {
	parseTheme := "atlas-theme-dark"
	if strings.EqualFold(parseP.Theme, "light") {
		parseTheme = "atlas-theme-light"
	}
	parseDensity := "atlas-density-compact"
	if strings.EqualFold(parseP.Density, "comfortable") {
		parseDensity = "atlas-density-comfortable"
	}
	return []string{parseTheme, parseDensity}
}

// readAtlasPreferenceSnapshot reads the four fields out of the hydrated form.
func readAtlasPreferenceSnapshot(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) atlasPreferenceSnapshot {
	parseT.Helper()
	return atlasPreferenceSnapshot{
		Theme:            atlasSpecFieldValue(parseT, parsePage, parseDiagnostics, atlasPreferencesFormSelector+` input[name="theme"]`),
		Locale:           atlasSpecFieldValue(parseT, parsePage, parseDiagnostics, atlasPreferencesFormSelector+` input[name="locale"]`),
		Density:          atlasSpecFieldValue(parseT, parsePage, parseDiagnostics, atlasPreferencesFormSelector+` select[name="density"]`),
		DefaultWarehouse: atlasSpecFieldValue(parseT, parsePage, parseDiagnostics, atlasPreferencesFormSelector+` input[name="default_warehouse_id"]`),
	}
}

// openAtlasSettingsForm opens /app/settings by DIRECT ENTRY and waits for the
// preference form.
//
// Direct entry (a fresh Goto) rather than client-side navigation is the point:
// the plan's third assertion is about direct-entry reload, and only a new
// document forces the server to re-render the preference state from storage.
func openAtlasSettingsForm(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseBaseURL string) {
	parseT.Helper()
	openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/settings", 0)
	assertAtlasSpecOperatorSessionVisible(parseT, parsePage, parseDiagnostics, "operator settings route")
	if _, parseErr := parsePage.WaitForSelector(atlasPreferencesFormSelector, playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(atlasSpecActionTimeoutMS),
		State:   playwright.WaitForSelectorStateVisible,
	}); parseErr != nil {
		parseT.Fatalf("operator preference form never appeared on /app/settings: %v | %s", parseErr, parseDiagnostics.Summary())
	}
}

// saveAtlasPreferences fills the form with a snapshot and submits it.
//
// The submit is a native POST; the server answers 303 back to the Referer with
// ?atlas_notice=preferences-saved, so this is a post-redirect-get round trip
// and ExpectNavigation is required to stay in step with the document swap.
func saveAtlasPreferences(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseTarget atlasPreferenceSnapshot) {
	parseT.Helper()

	if parseErr := parsePage.Fill(atlasPreferencesFormSelector+` input[name="theme"]`, parseTarget.Theme); parseErr != nil {
		parseT.Fatalf("fill theme: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	if parseErr := parsePage.Fill(atlasPreferencesFormSelector+` input[name="locale"]`, parseTarget.Locale); parseErr != nil {
		parseT.Fatalf("fill locale: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	// Density is a <select>, so it needs SelectOption rather than Fill.
	// SelectOption also fires the change event the bound GWC form listens to,
	// which keeps the client-side form state and the DOM in agreement.
	if _, parseErr := parsePage.SelectOption(atlasPreferencesFormSelector+` select[name="density"]`,
		playwright.SelectOptionValues{Values: &[]string{parseTarget.Density}}); parseErr != nil {
		parseT.Fatalf("select density %q: %v | %s", parseTarget.Density, parseErr, parseDiagnostics.Summary())
	}
	if parseErr := parsePage.Fill(atlasPreferencesFormSelector+` input[name="default_warehouse_id"]`, parseTarget.DefaultWarehouse); parseErr != nil {
		parseT.Fatalf("fill default warehouse: %v | %s", parseErr, parseDiagnostics.Summary())
	}

	parseResponse, parseErr := parsePage.ExpectNavigation(func() error {
		return parsePage.Click(atlasPreferencesFormSelector + ` button[type="submit"]`)
	}, playwright.PageExpectNavigationOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(atlasSpecActionTimeoutMS),
	})
	if parseErr != nil {
		parseT.Fatalf("preference save did not navigate: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	if parseResponse == nil || parseResponse.Status() != 200 {
		// A 400 here means the server rejected the values, which for this form
		// means either a validation regression or a bad fixture - and the JSON
		// body names which. Surfacing the status keeps those distinguishable.
		parseStatus := 0
		if parseResponse != nil {
			parseStatus = parseResponse.Status()
		}
		parseBody := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
		parseT.Fatalf("preference save ended on status %d with body %q | %s", parseStatus, parseBody, parseDiagnostics.Summary())
	}

	// The redirect must come back to settings with the notice - that is the
	// operator's only confirmation the save happened.
	parseLanded := parsePage.URL()
	if !strings.Contains(parseLanded, "/app/settings") || !strings.Contains(parseLanded, "atlas_notice=preferences-saved") {
		parseT.Fatalf("preference save landed on %q, expected /app/settings with atlas_notice=preferences-saved | %s",
			parseLanded, parseDiagnostics.Summary())
	}
	waitForAtlasSpecHydration(parseT, parsePage, parseDiagnostics, parseLanded)
	assertAtlasSpecTextContains(parseT, "post-save settings route",
		atlasSpecBodyText(parseT, parsePage, parseDiagnostics), "preferences-saved", parseDiagnostics)
}

// assertAtlasDocumentPresentation asserts the SSR <html> class AND lang match a
// snapshot - the server-side witnesses for persisted preferences.
//
// Three of the four preference fields are observable here: theme and density in
// the class attribute, locale in lang. The default warehouse is data rather
// than presentation, so it is asserted through the reloaded form value instead.
func assertAtlasDocumentPresentation(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseScope string, parseExpected atlasPreferenceSnapshot) {
	parseT.Helper()
	parseClass := atlasSpecDocumentClass(parseT, parsePage, parseDiagnostics)
	for _, parseWanted := range parseExpected.expectedDocumentClass() {
		if !strings.Contains(parseClass, parseWanted) {
			parseT.Fatalf("%s: <html class=%q> is missing %q for %s | %s",
				parseScope, parseClass, parseWanted, parseExpected, parseDiagnostics.Summary())
		}
	}
	if parseLang := atlasSpecDocumentLang(parseT, parsePage, parseDiagnostics); parseLang != parseExpected.Locale {
		parseT.Fatalf("%s: <html lang=%q>, expected %q from the saved locale preference | %s",
			parseScope, parseLang, parseExpected.Locale, parseDiagnostics.Summary())
	}
}

// TestAtlasOperatorPreferencesPersistAcrossReload saves flipped preferences and
// proves they survive a direct-entry reload and a different route family.
func TestAtlasOperatorPreferencesPersistAcrossReload(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)

		// KNOWN DEFECT ALLOWANCE, scoped to one exact asset URL.
		//
		// /app/settings spawns a Web Worker from
		// /assets/script/atlas-saved-view-import-worker.js, and that file does
		// not exist (examples/static/script contains only example-logger.js and
		// wasm_exec.js), so the route emits three 404s, three failed requests
		// and three console errors on every visit.
		//
		// That defect is asserted - and left failing - by
		// TestAtlasOperatorSettingsSavedViewWorkerAssetExists below. It is
		// allowed HERE because this spec's subject is preference persistence,
		// and letting one unrelated missing asset keep this spec permanently red
		// would destroy its value as a signal: nobody could tell whether
		// persistence had broken. The allowance names the exact file, so any
		// OTHER missing asset on this route still fails the run.
		parseDiagnostics.Allow("atlas-saved-view-import-worker.js")

		seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "ops_lead")

		// --- read the starting state -----------------------------------------
		openAtlasSettingsForm(parseT, parsePage, parseDiagnostics, parseBaseURL)
		parseOriginal := readAtlasPreferenceSnapshot(parseT, parsePage, parseDiagnostics)
		parseT.Logf("atlas settings original: %s", parseOriginal)

		// Sanity-check the baseline against the SSR witness. If these already
		// disagree, preferences are broken before we change anything and the
		// later assertions would be measuring from a false zero.
		assertAtlasDocumentPresentation(parseT, parsePage, parseDiagnostics, "baseline settings document", parseOriginal)

		parseTarget := parseOriginal.flipped()
		parseT.Logf("atlas settings target: %s", parseTarget)

		// Guard the flip itself: if any field failed to change, the persistence
		// assertion below would be trivially satisfied by doing nothing.
		if parseTarget.Theme == parseOriginal.Theme ||
			parseTarget.Locale == parseOriginal.Locale ||
			parseTarget.Density == parseOriginal.Density ||
			parseTarget.DefaultWarehouse == parseOriginal.DefaultWarehouse {
			parseT.Fatalf("target preferences (%s) do not differ from the original (%s) in every field; the persistence assertion would be vacuous",
				parseTarget, parseOriginal)
		}

		// --- save -------------------------------------------------------------
		saveAtlasPreferences(parseT, parsePage, parseDiagnostics, parseTarget)

		// --- assertion 1: direct-entry reload -------------------------------
		//
		// A completely fresh document: new wasm instance, no client state
		// carried over. Whatever the form shows now came from the server.
		openAtlasSettingsForm(parseT, parsePage, parseDiagnostics, parseBaseURL)
		parseReloaded := readAtlasPreferenceSnapshot(parseT, parsePage, parseDiagnostics)
		if parseReloaded != parseTarget {
			parseT.Fatalf("preferences did not persist across a direct-entry reload: got %s, expected %s | %s",
				parseReloaded, parseTarget, parseDiagnostics.Summary())
		}
		// The SSR witness must have flipped too. This is what rules out "the
		// form remembered" as an explanation.
		assertAtlasDocumentPresentation(parseT, parsePage, parseDiagnostics, "reloaded settings document", parseTarget)

		// And it must NOT still carry the original presentation classes -
		// otherwise a document that stamped both would pass the check above.
		parseClass := atlasSpecDocumentClass(parseT, parsePage, parseDiagnostics)
		for _, parseStale := range parseOriginal.expectedDocumentClass() {
			parseStillPresent := false
			for _, parseWanted := range parseTarget.expectedDocumentClass() {
				if parseWanted == parseStale {
					parseStillPresent = true
					break
				}
			}
			if parseStillPresent {
				continue
			}
			if strings.Contains(parseClass, parseStale) {
				parseT.Fatalf("reloaded settings document still carries the pre-save presentation class %q (class=%q) | %s",
					parseStale, parseClass, parseDiagnostics.Summary())
			}
		}

		// --- assertion 3: across route families ------------------------------
		//
		// /app/dashboard is a different internal route family that renders no
		// preference form at all. If the saved preferences only lived in the
		// settings screen's own state, this hop would show the old
		// presentation.
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/dashboard", 0)
		assertAtlasSpecOperatorSessionVisible(parseT, parsePage, parseDiagnostics, "dashboard after preference save")
		assertAtlasDocumentPresentation(parseT, parsePage, parseDiagnostics, "dashboard document after preference save", parseTarget)

		// The dashboard also renders the saved default warehouse in its
		// workspace chips, so the operator can see the persisted scope outside
		// the settings route. The slug is distinctive enough that a substring
		// match is meaningful (unlike "en", which matches half the English
		// language - which is exactly why locale is asserted via <html lang>
		// above rather than as page copy).
		assertAtlasSpecTextContains(parseT, "dashboard workspace chips",
			atlasSpecBodyText(parseT, parsePage, parseDiagnostics), parseTarget.DefaultWarehouse, parseDiagnostics)

		// --- scoping: the public surface must NOT inherit operator prefs ------
		//
		// Atlas keeps operator preferences on the internal console; the public
		// storefront always renders the default dark/compact presentation in
		// English. Asserting that keeps the persistence result honest: if the
		// preference record were applied globally, "it persisted" would be
		// indistinguishable from "a global default happens to match".
		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/shop", 0)
		assertAtlasSpecTextContains(parseT, "public catalog after preference save",
			atlasSpecBodyText(parseT, parsePage, parseDiagnostics), "Catalog overview", parseDiagnostics)
		parsePublicClass := atlasSpecDocumentClass(parseT, parsePage, parseDiagnostics)
		for _, parsePublicDefault := range []string{"atlas-theme-dark", "atlas-density-compact"} {
			if !strings.Contains(parsePublicClass, parsePublicDefault) {
				parseT.Fatalf("public /shop document class %q lost the public default %q; operator preferences appear to have leaked into the public surface | %s",
					parsePublicClass, parsePublicDefault, parseDiagnostics.Summary())
			}
		}
		if parsePublicLang := atlasSpecDocumentLang(parseT, parsePage, parseDiagnostics); parsePublicLang != "en" {
			parseT.Fatalf("public /shop rendered <html lang=%q>; the operator locale preference leaked into the public surface | %s",
				parsePublicLang, parseDiagnostics.Summary())
		}

		// --- restore the fixture ---------------------------------------------
		//
		// The preference record lives in a checked-in shared sqlite file, so this
		// spec puts back what it found.
		//
		// This runs INLINE rather than from t.Cleanup, and that is not a style
		// preference: withExamplesPage closes the page and the browser with
		// `defer`, while a t.Cleanup registered inside its callback runs after the
		// test function returns - i.e. AFTER the browser is gone. A cleanup-based
		// restore therefore silently no-ops on a closed page, which is exactly
		// what an earlier version of this spec did.
		//
		// If an assertion above fails, the restore is skipped and the flipped
		// values stay behind. That is harmless BY DESIGN: this spec never expects
		// a fixed starting state, it reads whatever it finds and flips it, so the
		// next run simply flips back. Depending on absolute values here is what
		// would have made an aborted run poison the following one.
		openAtlasSettingsForm(parseT, parsePage, parseDiagnostics, parseBaseURL)
		saveAtlasPreferences(parseT, parsePage, parseDiagnostics, parseOriginal)
		openAtlasSettingsForm(parseT, parsePage, parseDiagnostics, parseBaseURL)
		if parseRestored := readAtlasPreferenceSnapshot(parseT, parsePage, parseDiagnostics); parseRestored != parseOriginal {
			parseT.Fatalf("failed to restore the shared preference fixture: got %s, expected %s | %s",
				parseRestored, parseOriginal, parseDiagnostics.Summary())
		}
		parseT.Logf("atlas settings restored to: %s", parseOriginal)

		assertAtlasSpecClean(parseT, parseDiagnostics, "atlas operator preference persistence")
	})
}

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/operator-flow/settings-persistence.spec.md
// assertion 2 - "saved-view import or export validation is visible"
// =============================================================================
//
// !!! THIS TEST IS EXPECTED TO FAIL TODAY. IT DOCUMENTS A LIVE DEFECT. !!!
//
// DEFECT: /app/settings mounts a saved-view import validator backed by a Web
// Worker at /assets/script/atlas-saved-view-import-worker.js
// (shared/atlas/page.go, savedViewTransferCard -> useAtlasWorkerTask). That file
// does not exist. examples/static/script contains only example-logger.js and
// wasm_exec.js, and /assets/ is served from examples/static, so the worker
// script 404s on every visit to the settings route. Observed per visit:
//
//   - HTTP 404 /assets/script/atlas-saved-view-import-worker.js  (x3, retried)
//   - net::ERR_FAILED on the same URL                            (x3)
//   - console error: [interop/NewWorker] worker "..." reported an error during
//     startup: NewWorker ... [remote_error]: {"isTrusted":true}  (x3)
//
// CONSEQUENCE: the import-validation feedback the plan asks for can never
// appear, because the validator that produces it never starts. Nothing in the
// UI says so - the page renders, the form is usable, the validation card just
// stays quiet. This is precisely the class of failure a DOM-only assertion
// cannot see and captured console/network diagnostics can.
//
// The fix is either to ship the worker script into examples/static/script or to
// stop declaring the worker; either way this test turns green and becomes its
// regression guard. It is a separate test function so it can be skipped by name
// without losing the persistence coverage above.
func TestAtlasOperatorSettingsSavedViewWorkerAssetExists(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		// NOTE: no Allow() for the worker URL here. This test's whole purpose is
		// to fail on it.
		parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
		seedAtlasSpecOperatorCookie(parseT, parsePage, parseBaseURL, "ops_lead")

		openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/app/settings", 0)
		assertAtlasSpecOperatorSessionVisible(parseT, parsePage, parseDiagnostics, "operator settings route")

		// The saved-view import surface must actually be on the page; otherwise
		// a missing worker would be irrelevant and this test would be asserting
		// something Atlas no longer does.
		assertAtlasSpecTextContains(parseT, "settings saved-view transfer card",
			atlasSpecBodyText(parseT, parsePage, parseDiagnostics), "Import saved views", parseDiagnostics)

		// Give the worker its startup window. Waiting is what makes the
		// diagnostics assertion below meaningful: the failure arrives
		// asynchronously, so asserting immediately after hydration could pass
		// simply by being early.
		parsePage.WaitForTimeout(2000)

		assertAtlasSpecClean(parseT, parseDiagnostics, "atlas settings saved-view import worker")
	})
}

// COVERAGE NOTE for operator-flow/settings-persistence.spec.md
//
// COVERED: theme, locale, density and default warehouse persisting through a
// save; direct-entry reload; persistence visible in a different route family;
// the SSR presentation witness flipping with the save.
//
// PARTIALLY COVERED: the saved-view import surface (the plan's second
// assertion). TestAtlasOperatorSettingsSavedViewWorkerAssetExists covers the
// prerequisite - that the validation worker can start at all - and currently
// fails because its script is missing. Driving an actual import and asserting
// its validation output is NOT covered: an import writes rows into the shared
// checked-in sqlite fixture that this spec cannot reliably undo the way it
// restores the single preference record, so it is deferred until the example
// accepts an isolated database path.
//
// NOT COVERED HERE: the /app/settings/appearance, /app/settings/locale and
// /app/settings/workspace-defaults subroutes.
