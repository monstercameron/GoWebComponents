//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// Atlas Commerce OS - executable browser-spec support layer
// =============================================================================
//
// WHY THIS FILE EXISTS
//
// Atlas is the flagship example in this repo (~29k lines: SSR bootstrap +
// wasm hydration + sqlite + mock auth + CSRF forms + buyer and operator
// surfaces) and until these specs landed it had NO executable browser
// coverage. Its intended coverage lived only as markdown plans under
// examples/tests/atlas-commerce-os/. The cost of that gap was concrete:
// Atlas was broken at boot for four months while its Go unit tests stayed
// green, because nothing ever loaded the app in a real browser.
//
// The specs in the atlas_*_spec_test.go files are the executable split-out of
// those plans. This file holds the machinery they share. Read this file first
// if you are writing a new GWC browser spec - the four helpers below encode
// the lessons that cost this repo real debugging time.
//
// THE FOUR RULES ENCODED HERE
//
//  1. ALWAYS CAPTURE BROWSER DIAGNOSTICS (atlasSpecDiagnostics).
//     A spec that can only report "waited 30 seconds for a selector" names no
//     cause. When Atlas panicked at boot, the real cause was a console log
//     ("GoUseAtom called outside component context") plus a wasm "exit code:
//     2" - both invisible to a test that only looks at the DOM. Every spec
//     attaches diagnostics BEFORE the first navigation and folds the captured
//     console output, page errors, failed requests and >=400 responses into
//     every failure message.
//
//  2. NEVER HARDCODE A PORT (reserveAtlasSpecPort).
//     Ports 8096/8097/8123/8231/8471 are routinely occupied on developer
//     machines by other agents and other example servers. A hardcoded port
//     that is already bound fails as "health check timed out", which blames
//     the app for a harness problem. We ask the OS for a free port instead.
//
//  3. PROVE THE CLIENT BUNDLE IS THERE BEFORE BLAMING THE APP
//     (assertAtlasSpecWASMPresent). Atlas serves an EMPTY <div id="app"></div>
//     and paints everything from wasm. If examples/static/bin/
//     atlas-commerce-os.wasm is missing, every route still returns HTTP 200
//     with a valid-looking document and every spec then dies waiting for
//     hydration. /healthz reports wasmPresent, so we check it up front and
//     fail with the exact build command instead.
//
//  4. AUTHENTICATE THROUGH THE THING THAT ACTUALLY CREATES THE SESSION
//     (signInAtlasSpecOperator / seedAtlasSpecOperatorCookie).
//     GET /auth/mock-sign-in only sets a CSRF cookie; it does NOT create a
//     session. An earlier pass used the GET and reported NINE GREEN OPERATOR
//     ROUTES when five of them were rendering the same unauthenticated
//     sign-in screen. The session comes from the POST (which sets
//     atlas_mock_role) - so either drive the real form or set that cookie.
//     Whichever you use, assert on operator-only content afterwards; that is
//     the only thing that distinguishes "signed in" from "looking at the
//     sign-in page with a 200 status".

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

// atlasSpecHydrationTimeoutMS bounds the wait for wasm to paint into #app.
//
// Local hydration of one Atlas route measures around 0.9s, but the FIRST
// route of a run also pays for downloading and instantiating a ~19MB wasm
// bundle on a cold HTTP cache, and CI machines are slower and noisier than
// that. The budget is deliberately generous: a too-tight timeout turns a slow
// machine into a fake product failure, and because rule 1 gives us the real
// console output on timeout we do not need a short timeout to learn the cause.
const atlasSpecHydrationTimeoutMS = 45000

// atlasSpecActionTimeoutMS bounds waits for post-hydration DOM work (a form
// appearing, a table filtering, a client-side route swap). These are cheap
// once the bundle is live, so this is much shorter than hydration - a wait
// this long failing is real information, not machine noise.
const atlasSpecActionTimeoutMS = 20000

// -----------------------------------------------------------------------------
// Diagnostics
// -----------------------------------------------------------------------------

// atlasSpecDiagnostics records everything the browser said that a DOM
// assertion cannot see: console messages, uncaught page errors, requests that
// never completed, and responses with a >=400 status.
//
// Playwright delivers these on its own event goroutines, so every field is
// mutex guarded. Reads happen from the test goroutine while the page is still
// live, so unsynchronised access here would be a real data race, not a
// theoretical one.
type atlasSpecDiagnostics struct {
	mu               sync.Mutex
	consoleMessages  []string
	consoleErrors    []string
	pageErrors       []string
	failedRequests   []string
	responseFailures []string
	allowedPatterns  []string
}

// attachAtlasSpecDiagnostics wires the capture hooks onto a page.
//
// Call this BEFORE the first Goto. Playwright only reports events that occur
// after the listener is registered, so a diagnostics object attached after
// navigation misses exactly the boot-time console output that explains a boot
// failure - which is the case we care most about.
func attachAtlasSpecDiagnostics(parsePage playwright.Page) *atlasSpecDiagnostics {
	parseDiagnostics := &atlasSpecDiagnostics{}

	// A favicon request is issued by the browser, not by Atlas, and Atlas
	// answers unknown GET paths with its 404 recovery page. That is correct
	// product behaviour but it shows up as a console error, so it is allowed
	// by default - otherwise every single spec would fail on it and the
	// signal would be trained out of the suite.
	parseDiagnostics.Allow("favicon")

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseEntry := fmt.Sprintf("%s: %s", parseMessage.Type(), parseText)
		parseDiagnostics.mu.Lock()
		defer parseDiagnostics.mu.Unlock()
		// The cap keeps a chatty debug build from producing a megabyte-long
		// failure message; 120 entries is far more than any Atlas route emits.
		if len(parseDiagnostics.consoleMessages) < 120 {
			parseDiagnostics.consoleMessages = append(parseDiagnostics.consoleMessages, parseEntry)
		}
		if parseMessage.Type() == "error" {
			parseDiagnostics.consoleErrors = append(parseDiagnostics.consoleErrors, parseEntry)
		}
	})

	// OnPageError catches uncaught exceptions. A wasm panic surfaces here (or
	// as a console error) and nowhere in the DOM, which is precisely why the
	// Atlas boot panic stayed invisible for months.
	parsePage.OnPageError(func(parseErr error) {
		if parseErr == nil {
			return
		}
		parseDiagnostics.mu.Lock()
		defer parseDiagnostics.mu.Unlock()
		parseDiagnostics.pageErrors = append(parseDiagnostics.pageErrors, strings.TrimSpace(parseErr.Error()))
	})

	// OnRequestFailed fires for requests that never produced a response at all
	// (connection reset, blocked, aborted). A truncated wasm fetch lands here
	// and would otherwise look identical to "hydration is just slow".
	parsePage.OnRequestFailed(func(parseRequest playwright.Request) {
		parseReason := ""
		if parseFailure := parseRequest.Failure(); parseFailure != nil {
			parseReason = parseFailure.Error()
		}
		parseDiagnostics.mu.Lock()
		defer parseDiagnostics.mu.Unlock()
		parseDiagnostics.failedRequests = append(parseDiagnostics.failedRequests,
			fmt.Sprintf("%s %s (%s)", parseRequest.Method(), parseRequest.URL(), parseReason))
	})

	// OnResponse catches server-side failures the DOM hides. A missing asset
	// or a rejected API call still leaves a rendered page behind, so without
	// this a spec can pass while the route is quietly broken.
	parsePage.OnResponse(func(parseResponse playwright.Response) {
		if parseResponse.Status() < 400 {
			return
		}
		parseDiagnostics.mu.Lock()
		defer parseDiagnostics.mu.Unlock()
		parseDiagnostics.responseFailures = append(parseDiagnostics.responseFailures,
			fmt.Sprintf("HTTP %d %s", parseResponse.Status(), parseResponse.URL()))
	})

	return parseDiagnostics
}

// Allow marks a substring as expected, so entries containing it stop counting
// as unexpected noise.
//
// Some Atlas specs deliberately provoke failures - a tampered CSRF token must
// produce 403, a bad email must produce 400, an unknown route must produce
// 404. Those are the assertions, not accidents. Without an allowlist a spec
// would have to choose between asserting the failure and asserting the absence
// of noise; with one it can do both, and the allowlist itself documents which
// failures the spec intends.
func (parseD *atlasSpecDiagnostics) Allow(parseSubstring string) {
	parseD.mu.Lock()
	defer parseD.mu.Unlock()
	parseD.allowedPatterns = append(parseD.allowedPatterns, strings.ToLower(strings.TrimSpace(parseSubstring)))
}

func (parseD *atlasSpecDiagnostics) isAllowedLocked(parseEntry string) bool {
	parseLower := strings.ToLower(parseEntry)
	for _, parsePattern := range parseD.allowedPatterns {
		if parsePattern != "" && strings.Contains(parseLower, parsePattern) {
			return true
		}
	}
	return false
}

// Unexpected returns every captured problem that no Allow pattern covers.
// A spec asserts len(Unexpected()) == 0; the returned slice is the evidence.
func (parseD *atlasSpecDiagnostics) Unexpected() []string {
	parseD.mu.Lock()
	defer parseD.mu.Unlock()
	parseIssues := make([]string, 0, 8)
	for _, parseGroup := range [][]string{
		parseD.consoleErrors,
		parseD.pageErrors,
		parseD.failedRequests,
		parseD.responseFailures,
	} {
		for _, parseEntry := range parseGroup {
			if !parseD.isAllowedLocked(parseEntry) {
				parseIssues = append(parseIssues, parseEntry)
			}
		}
	}
	return parseIssues
}

// Summary renders everything captured so far as one line for a failure
// message. Every Fatalf in these specs ends with this: the whole point is that
// a failure names its cause instead of only naming the timeout.
func (parseD *atlasSpecDiagnostics) Summary() string {
	parseD.mu.Lock()
	defer parseD.mu.Unlock()
	return fmt.Sprintf(
		"console-errors=%d page-errors=%d failed-requests=%d http-failures=%d | console=%q | page-errors=%q | failed-requests=%q | http-failures=%q",
		len(parseD.consoleErrors),
		len(parseD.pageErrors),
		len(parseD.failedRequests),
		len(parseD.responseFailures),
		strings.Join(parseD.consoleMessages, " || "),
		strings.Join(parseD.pageErrors, " || "),
		strings.Join(parseD.failedRequests, " || "),
		strings.Join(parseD.responseFailures, " || "),
	)
}

// assertAtlasSpecClean fails the test if any unexpected browser problem was
// captured. Specs call this LAST, after their content assertions.
//
// Order matters: a content assertion failure is more specific and more
// actionable than "there were 3 console errors", so we want the content check
// to fire first when both are broken.
func assertAtlasSpecClean(parseT *testing.T, parseDiagnostics *atlasSpecDiagnostics, parseScope string) {
	parseT.Helper()
	if parseIssues := parseDiagnostics.Unexpected(); len(parseIssues) > 0 {
		parseT.Fatalf("%s: %d unexpected browser problem(s): %v | %s",
			parseScope, len(parseIssues), parseIssues, parseDiagnostics.Summary())
	}
}

// -----------------------------------------------------------------------------
// Server lifecycle
// -----------------------------------------------------------------------------

// reserveAtlasSpecPort asks the OS for a free loopback port.
//
// There is a small race between closing the listener and the Atlas server
// binding the same port, but it is far smaller than the near-certainty of
// collision from hardcoding: this machine already has 8096, 8097 and 8231
// bound, and the existing example specs have claimed most of 18090-18105.
// Reserving from the OS also lets several Atlas specs run back to back in one
// package without a port-assignment table to maintain.
func reserveAtlasSpecPort(parseT *testing.T) string {
	parseT.Helper()
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("reserve free port for atlas spec server: %v", parseErr)
	}
	parsePort := parseListener.Addr().(*net.TCPAddr).Port
	if parseCloseErr := parseListener.Close(); parseCloseErr != nil {
		parseT.Fatalf("release reserved port %d: %v", parsePort, parseCloseErr)
	}
	return strconv.Itoa(parsePort)
}

// startAtlasSpecServer boots one Atlas server for one spec and returns its
// base URL. The server is stopped by t.Cleanup inside startAtlasExamplesServer
// (which also kills the whole `go run` process tree on Windows, where the
// compiled child would otherwise survive its parent and hold the port).
//
// We reuse the existing startAtlasExamplesServer helper on purpose rather than
// re-implementing process management here: one lifecycle path means one place
// where a leaked listener can be fixed.
func startAtlasSpecServer(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parseBaseURL := startAtlasExamplesServer(parseT, parseRepoRoot, reserveAtlasSpecPort(parseT))
	assertAtlasSpecWASMPresent(parseT, parseBaseURL)
	return parseBaseURL
}

// assertAtlasSpecWASMPresent fails early, and with the fix in the message, when
// the client bundle has not been built.
//
// Atlas ships an empty <div id="app"></div>; all markup comes from wasm. With
// no bundle every route still answers HTTP 200 with a well-formed document, so
// the only symptom is that hydration never happens - which reads as a product
// bug. /healthz reports wasmPresent from the same constant the document's
// loader snippet fetches, so it is an honest pre-flight check.
//
// This helper deliberately does NOT build the bundle. The output path is a
// single shared file, and this repo has already been burned by concurrent
// builds writing one wasm path and corrupting it. Naming the command is safer
// than racing another agent's build.
func assertAtlasSpecWASMPresent(parseT *testing.T, parseBaseURL string) {
	parseT.Helper()
	parseResponse, parseErr := http.Get(parseBaseURL + "/healthz")
	if parseErr != nil {
		parseT.Fatalf("atlas /healthz request failed: %v", parseErr)
	}
	defer func() { _ = parseResponse.Body.Close() }()

	var parseHealth struct {
		OK          bool   `json:"ok"`
		WASMPresent bool   `json:"wasmPresent"`
		Service     string `json:"service"`
	}
	if parseDecodeErr := json.NewDecoder(parseResponse.Body).Decode(&parseHealth); parseDecodeErr != nil {
		parseT.Fatalf("decode atlas /healthz body: %v", parseDecodeErr)
	}
	if !parseHealth.OK {
		parseT.Fatalf("atlas /healthz reported ok=false (service=%q)", parseHealth.Service)
	}
	if !parseHealth.WASMPresent {
		parseT.Fatalf(
			"atlas client bundle missing: /healthz reported wasmPresent=false. "+
				"Every route would still return HTTP 200 and never hydrate. Build it with:\n"+
				"  GOOS=js GOARCH=wasm go build -o examples/static/bin/atlas-commerce-os.wasm ./examples/server/atlas-commerce-os/client\n"+
				"(base=%s)", parseBaseURL)
	}
}

// -----------------------------------------------------------------------------
// Navigation and hydration
// -----------------------------------------------------------------------------

// gotoAtlasSpecRoute navigates to one route and returns the main response so
// the caller can assert on status.
//
// WaitUntilStateDomcontentloaded, not "load" or "networkidle": Atlas keeps
// fetching after first paint (island resources, cache revalidation), so
// networkidle would be flaky-slow, while domcontentloaded plus the explicit
// hydration wait below expresses what we actually depend on.
func gotoAtlasSpecRoute(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseBaseURL string, parseRoute string) playwright.Response {
	parseT.Helper()
	parseResponse, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto %s: %v | %s", parseRoute, parseErr, parseDiagnostics.Summary())
	}
	if parseResponse == nil {
		parseT.Fatalf("goto %s returned a nil response | %s", parseRoute, parseDiagnostics.Summary())
	}
	return parseResponse
}

// waitForAtlasSpecHydration blocks until the wasm client has taken over the
// document.
//
// Two conditions, both necessary:
//
//   - #app has children. The server sends <div id="app"></div> EMPTY, so
//     children appearing is the one unambiguous signal that the client ran.
//     Waiting on any specific text instead would confuse "not hydrated yet"
//     with "hydrated the wrong route".
//   - #atlas-shell-root exists. This is the client's own shell wrapper. It
//     rules out a half-mount where a stray node lands in #app but the route
//     shell never renders.
//
// Deliberately NOT waited on: a fixed sleep, or "networkidle". Atlas islands
// keep fetching in the background, so both would trade correctness for either
// flakiness or wasted seconds.
func waitForAtlasSpecHydration(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseRoute string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			const root = document.getElementById("app");
			if (!root || root.children.length === 0) {
				return false;
			}
			return !!document.getElementById("atlas-shell-root");
		}`,
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(atlasSpecHydrationTimeoutMS)},
	); parseErr != nil {
		// This is the failure mode the whole diagnostics layer exists for:
		// "hydration never completed" is useless on its own, while the console
		// output usually contains the panic or the missing-hook message.
		parseT.Fatalf("route %s never hydrated (#app stayed empty or #atlas-shell-root missing): %v | %s",
			parseRoute, parseErr, parseDiagnostics.Summary())
	}
}

// openAtlasSpecRoute is the goto + hydrate pair every spec starts from, plus a
// status guard. parseAllowStatus lets recovery specs accept a deliberate 404.
func openAtlasSpecRoute(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseBaseURL string, parseRoute string, parseAllowStatus int) playwright.Response {
	parseT.Helper()
	parseResponse := gotoAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, parseRoute)
	parseStatus := parseResponse.Status()
	if parseStatus >= 400 && parseStatus != parseAllowStatus {
		parseT.Fatalf("route %s returned status %d (expected <400 or %d) | %s",
			parseRoute, parseStatus, parseAllowStatus, parseDiagnostics.Summary())
	}
	waitForAtlasSpecHydration(parseT, parsePage, parseDiagnostics, parseRoute)
	return parseResponse
}

// waitForAtlasSpecCondition waits for a post-hydration DOM predicate.
// parseDescription is what shows up in the failure message, so it should read
// as a product statement ("the inventory queue table has rows"), not as a
// selector.
func waitForAtlasSpecCondition(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseDescription string, parseExpression string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(
		parseExpression,
		nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(atlasSpecActionTimeoutMS)},
	); parseErr != nil {
		parseT.Fatalf("timed out waiting for %s: %v | %s", parseDescription, parseErr, parseDiagnostics.Summary())
	}
}

// waitForAtlasSpecVisibleText waits until a phrase appears in the rendered
// document text.
//
// The comparison is lowercased on BOTH sides inside the browser, for the same
// text-transform reason documented on atlasSpecTextContains. Keeping the
// lowercasing inside the JS (rather than fetching text and comparing in Go)
// means the wait retries on the browser's own schedule instead of polling
// across the wire.
func waitForAtlasSpecVisibleText(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseDescription string, parsePhrase string) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(
		`(phrase) => {
			const text = document.body ? document.body.innerText : "";
			return text.toLowerCase().includes(String(phrase).toLowerCase());
		}`,
		parsePhrase,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(atlasSpecActionTimeoutMS)},
	); parseErr != nil {
		parseT.Fatalf("timed out waiting for %s (phrase %q): %v | %s",
			parseDescription, parsePhrase, parseErr, parseDiagnostics.Summary())
	}
}

// -----------------------------------------------------------------------------
// Reading the rendered page
// -----------------------------------------------------------------------------

// atlasSpecPrimaryHeading returns the text of the first <h1>.
//
// Every Atlas route paints a route-specific h1 ("Atlas Shop", "Atlas Inventory",
// "Atlas Route Not Found", ...), which makes it the cheapest honest answer to
// "which route am I actually looking at". Note that internal routes render the
// same string twice (shell crumb + page hero); we take the first and compare
// with Contains so that duplication is irrelevant.
func atlasSpecPrimaryHeading(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) string {
	parseT.Helper()
	parseHeading, parseErr := parsePage.InnerText("h1")
	if parseErr != nil {
		parseT.Fatalf("read primary <h1>: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	return strings.TrimSpace(parseHeading)
}

// atlasSpecBodyText returns the rendered (visible) text of the page.
//
// InnerText, not Content/innerHTML: class names in Atlas markup contain words
// like "grid" and colour tokens, and asserting against raw HTML lets a class
// name satisfy a copy assertion. InnerText also skips display:none subtrees,
// so a "visible" assertion means visible.
func atlasSpecBodyText(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) string {
	parseT.Helper()
	parseText, parseErr := parsePage.InnerText("body")
	if parseErr != nil {
		parseT.Fatalf("read body text: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	return parseText
}

// atlasSpecTextContains reports whether rendered text contains a phrase,
// ignoring case.
//
// Case is ignored for a concrete reason, and it cost time to learn: InnerText
// returns text as RENDERED, and Atlas styles many of its section eyebrows with
// Tailwind's `uppercase`. The source copy "Why Atlas feels ready" arrives from
// InnerText as "WHY ATLAS FEELS READY". A case-sensitive assertion against the
// copy in the Go source therefore fails on a perfectly healthy page, which is
// the worst kind of test failure: it blames the product for a harness mistake.
//
// The alternative - asserting against innerHTML to dodge text-transform - is
// worse: class attributes are full of English words ("grid", "hidden",
// "comfortable"), so an innerHTML assertion can be satisfied by styling rather
// than by content.
func atlasSpecTextContains(parseText string, parseWanted string) bool {
	return strings.Contains(strings.ToLower(parseText), strings.ToLower(parseWanted))
}

// assertAtlasSpecTextContains asserts one phrase is present in rendered text.
func assertAtlasSpecTextContains(parseT *testing.T, parseScope string, parseText string, parseWanted string, parseDiagnostics *atlasSpecDiagnostics) {
	parseT.Helper()
	if !atlasSpecTextContains(parseText, parseWanted) {
		parseT.Fatalf("%s: expected rendered text to contain %q (case-insensitive); it did not | %s",
			parseScope, parseWanted, parseDiagnostics.Summary())
	}
}

// assertAtlasSpecTextMissing asserts a phrase is ABSENT.
//
// This is the anti-vacuity half of every route assertion in this suite. A
// positive assertion alone can pass for the wrong reason - the earlier Atlas
// pass "verified" nine operator routes that were all rendering the same
// sign-in screen, because every assertion it made was true of that screen too.
// Pairing "route A's marker is present" with "route B's marker is absent"
// makes it impossible for one screen to satisfy two different routes.
func assertAtlasSpecTextMissing(parseT *testing.T, parseScope string, parseText string, parseUnwanted string, parseDiagnostics *atlasSpecDiagnostics) {
	parseT.Helper()
	if atlasSpecTextContains(parseText, parseUnwanted) {
		parseT.Fatalf("%s: expected rendered text NOT to contain %q, but it did - the route is probably rendering a different screen than expected | %s",
			parseScope, parseUnwanted, parseDiagnostics.Summary())
	}
}

// atlasSpecCountElements counts matches for a CSS selector in the live DOM.
func atlasSpecCountElements(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseSelector string) int {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(`(selector) => document.querySelectorAll(selector).length`, parseSelector)
	if parseErr != nil {
		parseT.Fatalf("count elements for %q: %v | %s", parseSelector, parseErr, parseDiagnostics.Summary())
	}
	parseCount, parseOK := parseValue.(int)
	if !parseOK {
		// Playwright returns JS numbers as float64 when they are not integral.
		if parseFloat, parseFloatOK := parseValue.(float64); parseFloatOK {
			return int(parseFloat)
		}
		parseT.Fatalf("unexpected element count type %T for %q", parseValue, parseSelector)
	}
	return parseCount
}

// atlasSpecFieldValue reads one form control's current value.
//
// It reads the live .value property rather than the value ATTRIBUTE. For a
// hydrated GWC form these differ: the attribute is what was rendered, the
// property is what the user (or a prior save) actually has in the field, and
// the property is what will be posted.
func atlasSpecFieldValue(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseSelector string) string {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(
		`(selector) => { const el = document.querySelector(selector); return el ? String(el.value) : null; }`,
		parseSelector)
	if parseErr != nil {
		parseT.Fatalf("read field value %q: %v | %s", parseSelector, parseErr, parseDiagnostics.Summary())
	}
	if parseValue == nil {
		parseT.Fatalf("form control %q is not present in the hydrated DOM | %s", parseSelector, parseDiagnostics.Summary())
	}
	parseText, parseOK := parseValue.(string)
	if !parseOK {
		parseT.Fatalf("unexpected field value type %T for %q", parseValue, parseSelector)
	}
	return parseText
}

// assertAtlasFormFieldNames asserts a form renders exactly the expected set of
// named controls, in any order, with no duplicates and no extras.
//
// WHY AN EXACT SET RATHER THAN "contains the field I need"
//
// A GWC form is assembled from components, and a component-identity bug can
// make two sibling fields render the SAME name (both taking the last sibling's
// props). The consequences are silent: the browser posts one name twice, the
// server reads the first occurrence, and the value the user typed into the other
// field is never sent. Nothing errors. A presence check on the surviving name
// passes.
//
// Comparing the exact multiset catches all three shapes of that bug - a missing
// name, a duplicated name, and an unexpected extra - and reports the actual
// rendered names, so the failure explains itself without a debugging session.
//
// Field ORDER is deliberately not asserted: it is a layout decision, not a
// contract, and pinning it would make every visual tweak a test failure.
func assertAtlasFormFieldNames(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseFormSelector string, parseExpected []string) {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(
		`(selector) => {
			const form = document.querySelector(selector);
			if (!form) { return null; }
			return Array.from(form.querySelectorAll("input,select,textarea"))
				.map(el => el.getAttribute("name") || "<unnamed>");
		}`, parseFormSelector)
	if parseErr != nil {
		parseT.Fatalf("read form field names for %q: %v | %s", parseFormSelector, parseErr, parseDiagnostics.Summary())
	}
	if parseValue == nil {
		parseT.Fatalf("form %q is not present in the hydrated DOM | %s", parseFormSelector, parseDiagnostics.Summary())
	}
	parseItems, parseOK := parseValue.([]interface{})
	if !parseOK {
		parseT.Fatalf("unexpected field-name payload type %T for %q", parseValue, parseFormSelector)
	}
	parseActual := make([]string, 0, len(parseItems))
	for _, parseItem := range parseItems {
		if parseName, parseNameOK := parseItem.(string); parseNameOK {
			parseActual = append(parseActual, parseName)
		}
	}

	parseActualSorted := append([]string(nil), parseActual...)
	parseExpectedSorted := append([]string(nil), parseExpected...)
	sort.Strings(parseActualSorted)
	sort.Strings(parseExpectedSorted)
	if strings.Join(parseActualSorted, ",") != strings.Join(parseExpectedSorted, ",") {
		parseT.Fatalf(
			"form %q field-name contract broken.\n  rendered: %v\n  expected: %v\n"+
				"A duplicated name means two sibling field components collapsed onto one set of props; "+
				"a missing name means the browser will never post that value. | %s",
			parseFormSelector, parseActual, parseExpected, parseDiagnostics.Summary())
	}
}

// atlasSpecDocumentClass returns the <html> element's class attribute.
//
// Atlas stamps persisted theme and density onto <html> during SSR
// (atlas-theme-dark|light + atlas-density-compact|comfortable) so a
// direct-entry page does not flash default styling before resuming saved
// preferences. That makes this attribute a server-side witness for preference
// persistence: it is produced by the server from the database, not by the
// client from a form, so asserting on it cannot be satisfied by leftover
// client state.
func atlasSpecDocumentClass(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) string {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(`() => document.documentElement.className`)
	if parseErr != nil {
		parseT.Fatalf("read <html> class: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	parseText, _ := parseValue.(string)
	return parseText
}

// atlasSpecDocumentLang returns the <html lang> attribute.
//
// The Atlas server renders lang from the resolved i18n locale of the route
// payload, which for internal routes comes from the operator's saved
// preferences. Like the class attribute above, it is present in the HTML the
// server sent, so it is a server-side witness that client state cannot forge.
func atlasSpecDocumentLang(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics) string {
	parseT.Helper()
	parseValue, parseErr := parsePage.Evaluate(`() => document.documentElement.getAttribute("lang") || ""`)
	if parseErr != nil {
		parseT.Fatalf("read <html lang>: %v | %s", parseErr, parseDiagnostics.Summary())
	}
	parseText, _ := parseValue.(string)
	return parseText
}

// -----------------------------------------------------------------------------
// Authentication
// -----------------------------------------------------------------------------

// atlasSpecOperatorRoles are the roles the mock session manager accepts.
// Anything else normalises to "" and creates no session at all.
var atlasSpecOperatorRoles = []string{"inventory_manager", "warehouse_supervisor", "ops_lead"}

// signInAtlasSpecOperator signs in by driving the REAL sign-in form.
//
// Why the form and not a cookie: this is the only path that exercises what a
// user does - the POST handler, the role/next hidden inputs, the Set-Cookie,
// the 303 back to the requested route, and the post-redirect notice. A cookie
// shortcut skips all of it and would keep passing if the sign-in POST broke.
//
// The role is selected structurally (`:has(input[value="<role>"])`) instead of
// by position, because the page renders one form per role and their order is
// not part of any contract.
//
// Returns the URL landed on so the caller can assert the "next" round trip.
func signInAtlasSpecOperator(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseBaseURL string, parseRole string) string {
	parseT.Helper()
	parseSelector := fmt.Sprintf(`form[action="/auth/mock-sign-in"]:has(input[value=%q]) button[type="submit"]`, parseRole)

	// Guard the selector before clicking. A silent zero-match click would
	// surface later as "the dashboard did not render", pointing at the wrong
	// component entirely.
	parseButtons := parsePage.Locator(parseSelector)
	parseCount, parseErr := parseButtons.Count()
	if parseErr != nil {
		parseT.Fatalf("count sign-in submit buttons for role %q: %v | %s", parseRole, parseErr, parseDiagnostics.Summary())
	}
	if parseCount != 1 {
		parseT.Fatalf("expected exactly 1 sign-in form for role %q, found %d - the mock sign-in page is not rendering the role cards | %s",
			parseRole, parseCount, parseDiagnostics.Summary())
	}

	// ExpectNavigation, not click-then-poll: the POST answers 303 and the
	// browser follows it, so the click and the document swap are one event.
	// Polling afterwards would race the navigation and could read the old DOM.
	if _, parseNavErr := parsePage.ExpectNavigation(func() error {
		return parseButtons.First().Click()
	}, playwright.PageExpectNavigationOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(atlasSpecActionTimeoutMS),
	}); parseNavErr != nil {
		parseT.Fatalf("mock sign-in POST for role %q did not navigate: %v | %s", parseRole, parseNavErr, parseDiagnostics.Summary())
	}

	parseLanded := parsePage.URL()
	waitForAtlasSpecHydration(parseT, parsePage, parseDiagnostics, parseLanded)
	return parseLanded
}

// seedAtlasSpecOperatorCookie installs the mock session cookie directly.
//
// This is the fast path for specs whose subject is an internal route rather
// than sign-in itself; TestAtlasOperatorMockSignIn covers the real form, so
// re-driving it in every operator spec would only add seconds.
//
// The cookie name matters: atlas_mock_role IS the session. atlas_csrf is only
// the CSRF pairing cookie, and the GET /auth/mock-sign-in page sets ONLY that
// one. Setting the CSRF cookie and assuming a session is the exact mistake
// that produced nine "green" operator routes rendering the sign-in screen -
// which is why every spec using this helper still asserts operator-only
// content afterwards.
func seedAtlasSpecOperatorCookie(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseRole string) {
	parseT.Helper()
	parseKnown := false
	for _, parseCandidate := range atlasSpecOperatorRoles {
		if parseCandidate == parseRole {
			parseKnown = true
			break
		}
	}
	if !parseKnown {
		// An unknown role normalises to "" server-side and creates NO session,
		// so a typo here would silently produce unauthenticated pages.
		parseT.Fatalf("role %q is not one of the mock roles %v; it would produce no session at all", parseRole, atlasSpecOperatorRoles)
	}
	if parseErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{{
		Name:  "atlas_mock_role",
		Value: parseRole,
		URL:   playwright.String(parseBaseURL),
	}}); parseErr != nil {
		parseT.Fatalf("seed atlas_mock_role cookie: %v", parseErr)
	}
}

// assertAtlasSpecOperatorSessionVisible proves the page is an AUTHENTICATED
// operator surface and not the sign-in screen wearing a 200 status.
//
// Both halves are load bearing. The positive half checks the operator identity
// chip the internal shell only renders with a session. The negative half
// checks that the sign-in call to action is gone: an unauthenticated visit to
// an /app route is a 303 to /auth/mock-sign-in, which also answers 200 and
// also renders the Atlas header, so status and shell prove nothing.
func assertAtlasSpecOperatorSessionVisible(parseT *testing.T, parsePage playwright.Page, parseDiagnostics *atlasSpecDiagnostics, parseScope string) {
	parseT.Helper()
	parseText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
	assertAtlasSpecTextContains(parseT, parseScope, parseText, "Atlas Demo Operator", parseDiagnostics)
	assertAtlasSpecTextMissing(parseT, parseScope, parseText, "Start session", parseDiagnostics)
	if strings.Contains(parsePage.URL(), "/auth/mock-sign-in") {
		parseT.Fatalf("%s: landed on the mock sign-in route (%s) instead of an authenticated operator route | %s",
			parseScope, parsePage.URL(), parseDiagnostics.Summary())
	}
}

// -----------------------------------------------------------------------------
// Misc
// -----------------------------------------------------------------------------

// atlasSpecRepoRoot resolves the repo root from a test file path.
// Thin alias over the shared helper, kept so the Atlas specs read uniformly.
func atlasSpecRepoRoot(parseTestFile string) string {
	return examplesRepoRootFromFile(parseTestFile)
}

// atlasSpecSettleBriefly yields to the event loop for a bounded moment.
//
// Used ONLY after an action whose completion has no observable DOM signal
// (for example: confirming a client-side navigation did not also trigger a
// document reload). Never use it as a substitute for waiting on a condition -
// a sleep that stands in for a wait is how a suite becomes slow and flaky at
// the same time.
func atlasSpecSettleBriefly(parsePage playwright.Page) {
	parsePage.WaitForTimeout(float64(250 * time.Millisecond / time.Millisecond))
}
