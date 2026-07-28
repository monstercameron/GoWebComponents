//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// =============================================================================
// SPEC: examples/tests/atlas-commerce-os/buyer-flow/form-submit.spec.md
//         ("public comment shows optimistic pending state and moderation
//           expectation messaging")
//       examples/tests/atlas-commerce-os/buyer-flow/progressive-enhancement.spec.md
//         ("quote, restock, and comment submissions work after reload")
// =============================================================================
//
// SUBJECT: the two DEFERRED modules on the product route - the buyer
// review-and-question workflow (publicProductFeedbackSection, mounted through
// atlasLazySection = ui.UseLazyNode + ui.AsyncBoundary) and the regional
// promise-lanes island (useAtlasResource + ui.AsyncBoundary).
//
// WHY DEFERRED CONTENT NEEDS ITS OWN SPEC
//
// This spec was written against a build where BOTH modules silently never
// mounted, and it is worth recording what that looked like, because it is the
// failure mode deferred rendering has and no other kind of test catches it:
//
//   - #app hydrated normally; hydrate.done and router.mount both logged.
//   - The slot between the "Quick buying notes" band and the promise-lanes
//     panel was EMPTY - publicProductFeedbackSection rendered neither its
//     content nor its AsyncBoundary fallback. Still empty after 30s.
//   - The promise-lanes island rendered its skeleton fallback forever and
//     NO GET /api/public/warehouses was ever issued, so the resource effect in
//     that subtree never ran.
//   - A sibling island on the same route DID work
//     (GET /api/public/products/frame-desk/related-products resolved), so the
//     transport was fine - the effects simply never started.
//   - Zero console errors. Zero page errors. Zero failed requests. The page
//     looked finished.
//
// Nothing about the route's heading, status, shell or primary content would have
// changed. An entire buyer workflow can disappear from a page while every
// conventional assertion stays green - which is why the assertions below wait on
// the DEFERRED artefacts specifically (the comment form, the buyer-note counter,
// a real availability link) rather than on anything the route paints eagerly.
//
// BLAST RADIUS if this regresses again: atlasLazySection has three call sites.
// Besides this one, the purchase-order detail side panel (page.go, around
// purchaseOrderDetailPanel) and the receiving detail side panel use it, so a
// recurrence would blank those panels too - and they have no spec yet.
//
// A NOTE ON SCROLLING: each subtest scrolls to the bottom before asserting. If a
// future implementation gates deferred mounting on visibility, that keeps this
// spec honest; and it means a failure can never be dismissed as "the test never
// scrolled the module into view".

import (
	"runtime"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

// TestAtlasBuyerDeferredProductModulesMount asserts the two deferred modules on
// the product route actually mount.
func TestAtlasBuyerDeferredProductModulesMount(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseBaseURL := startAtlasSpecServer(parseT, atlasSpecRepoRoot(parseFile))

	parseT.Run("buyer_review_and_question_workflow_mounts", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
			if parseErr := parsePage.SetViewportSize(1440, 900); parseErr != nil {
				parseT.Fatalf("set desktop viewport: %v", parseErr)
			}
			openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/shop/frame-desk", 0)

			// Scroll to the bottom before asserting. If the deferred module were
			// gated on visibility (an IntersectionObserver-style lazy mount)
			// this is what would trigger it - so doing it first means a failure
			// cannot be dismissed as "the test never scrolled to it".
			if _, parseErr := parsePage.Evaluate(`() => { window.scrollTo(0, document.body.scrollHeight); }`); parseErr != nil {
				parseT.Fatalf("scroll to the deferred section: %v | %s", parseErr, parseDiagnostics.Summary())
			}

			// The comment form is the workflow's entry point. It is a distinct
			// form from the restock form in the action rail, so requiring TWO
			// forms on the route is a precise statement: "the deferred module
			// contributed its form".
			waitForAtlasSpecCondition(parseT, parsePage, parseDiagnostics,
				"the deferred buyer review-and-question form to mount on /shop/frame-desk "+
					"(publicProductFeedbackSection via atlasLazySection; a failure here means the deferred slot stayed empty - "+
					"check whether its AsyncBoundary fallback appeared either, and whether effects ran in that subtree)",
				`() => !!document.querySelector('form[action="/api/public/products/frame-desk/comments"]')`)

			// The section heading and the buyer-note counter come from the same
			// component. Asserting them separately from the form means a
			// partial mount (form without list, or fallback text only) still
			// fails.
			parseText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
			assertAtlasSpecTextContains(parseT, "deferred feedback section", parseText, "Customer reviews and questions", parseDiagnostics)
			assertAtlasSpecTextContains(parseT, "deferred feedback section", parseText, "What buyers are asking before they commit.", parseDiagnostics)
			// "buyer note"/"buyer notes" is the rendered count label; its
			// presence means the comments resource resolved, not merely that a
			// skeleton painted.
			assertAtlasSpecTextContains(parseT, "deferred feedback section", parseText, "buyer note", parseDiagnostics)

			assertAtlasSpecClean(parseT, parseDiagnostics, "atlas deferred buyer feedback module")
		})
	})

	parseT.Run("regional_promise_lanes_island_resolves", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseDiagnostics := attachAtlasSpecDiagnostics(parsePage)
			if parseErr := parsePage.SetViewportSize(1440, 900); parseErr != nil {
				parseT.Fatalf("set desktop viewport: %v", parseErr)
			}
			openAtlasSpecRoute(parseT, parsePage, parseDiagnostics, parseBaseURL, "/shop/frame-desk", 0)
			if _, parseErr := parsePage.Evaluate(`() => { window.scrollTo(0, document.body.scrollHeight); }`); parseErr != nil {
				parseT.Fatalf("scroll to the promise-lanes island: %v | %s", parseErr, parseDiagnostics.Summary())
			}

			// The skeleton and the resolved card share a heading, so asserting
			// the heading would pass on the broken state. The two states are
			// distinguished by their body copy, and the resolved card is the
			// only one that renders availability links - so we assert on the
			// link, which is the thing a buyer can actually use.
			waitForAtlasSpecCondition(parseT, parsePage, parseDiagnostics,
				"the regional promise-lanes island to resolve on /shop/frame-desk "+
					"(useAtlasResource + AsyncBoundary; if this fails, check whether GET /api/public/warehouses was issued at all - "+
					"a stuck skeleton with no request means the resource effect never ran)",
				`() => {
					const text = document.body ? document.body.innerText : "";
					const resolved = text.includes("This below-the-fold module loads after the main product story");
					const link = document.querySelector('a[href="/warehouses/new-jersey-hub/availability/frame-desk"]');
					return resolved && !!link;
				}`)

			parseText := atlasSpecBodyText(parseT, parsePage, parseDiagnostics)
			// The skeleton's own copy must be gone: a page rendering both would
			// mean the boundary resolved without replacing its fallback.
			assertAtlasSpecTextMissing(parseT, "promise-lanes island", parseText,
				"Atlas defers this secondary lane module until after hydration", parseDiagnostics)

			assertAtlasSpecClean(parseT, parseDiagnostics, "atlas promise-lanes island")
		})
	})
}

// COVERAGE NOTE
//
// COVERED: that both deferred product modules reach the DOM and resolve their
// data - the prerequisite for everything else the plans ask of them.
//
// NOT COVERED HERE: actually driving the hydrated comment form - its client-side
// field validation ("Enter your name before sharing feedback."), its optimistic
// pending state, the "Your comment was submitted for review." message and the
// "Comment queued" toast. That flow writes a comment row into the SHARED
// checked-in sqlite fixture (examples/server/atlas-commerce-os/server/data/
// atlas-commerce-os.db - there is no env override for the path), and unlike a
// preference record a created comment cannot be restored afterwards. It is
// deferred until the example accepts an isolated database path. The
// non-mutating half of that plan - CSRF-protected submit round trips - is
// covered against the restock form in atlas_buyer_form_submit_spec_test.go.
//
// SEPARATE FINDING about progressive-enhancement.spec.md - "direct-entry SSR
// exposes usable forms before hydration": Atlas does not have this property, and
// no test in this suite asserts it. The server sends an EMPTY
// <div id="app"></div> and every byte of markup comes from wasm
// (see server.go's document template). With scripting disabled the page is
// blank - no forms, no headings, no content. That plan describes a
// server-rendered-markup design Atlas does not implement, so asserting it would
// encode a false property. It is reported as a finding instead.
