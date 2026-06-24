//go:build js && wasm

package router

import "testing"

// TestFragmentHrefPreservesQuery checks that an in-page anchor keeps the current
// query string (so clicking it does not navigate away from ?x=1 state).
func TestFragmentHrefPreservesQuery(parseT *testing.T) {
	installRouterBrowserEnv(parseT)
	lastPublishedLocationKey = ""
	globalRouter = NewHistoryRouter()
	registerLocationProbeRoutes(globalRouter)
	Navigate("/beta?x=1")

	parseQuery := getCurrentQueryValues().Encode()
	parseT.Logf("current query = %q", parseQuery)
	parseGot := FragmentHref("main")
	parseT.Logf("FragmentHref = %q", parseGot)

	if parseQuery != "" && parseGot != "/beta?"+parseQuery+"#main" {
		parseT.Fatalf("FragmentHref dropped the query: got %q, want /beta?%s#main", parseGot, parseQuery)
	}
}
