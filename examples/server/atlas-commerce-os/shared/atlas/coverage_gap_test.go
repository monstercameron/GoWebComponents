package atlas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/examples/server/atlas-commerce-os/shared/design"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TestAtlasResourceCacheFetchBranches covers HTTP fetch and startup resource branches.
func TestAtlasResourceCacheFetchBranches(parseT *testing.T) {
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		switch parseR.URL.Path {
		case "/ok":
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"name":"atlas"}`))
		case "/bad-json":
			parseW.Header().Set("Content-Type", "application/json")
			_, _ = parseW.Write([]byte(`{"name":`))
		default:
			http.Error(parseW, "unavailable", http.StatusServiceUnavailable)
		}
	}))
	defer parseServer.Close()

	type parseFetchResponse struct {
		Name string `json:"name"`
	}

	parsePayload, parseErr := fetchAtlasJSON[parseFetchResponse](context.Background(), parseServer.URL+"/ok")
	if parseErr != nil || parsePayload.Name != "atlas" {
		parseT.Fatalf("fetchAtlasJSON success = (%#v, %v), want name atlas", parsePayload, parseErr)
	}

	if _, parseErr2 := fetchAtlasJSON[parseFetchResponse](context.Background(), "://bad-url"); parseErr2 == nil {
		parseT.Fatal("expected invalid request URL to fail")
	}
	if _, parseErr3 := fetchAtlasJSON[parseFetchResponse](context.Background(), parseServer.URL+"/status"); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "status 503") {
		parseT.Fatalf("expected status error, got %v", parseErr3)
	}
	if _, parseErr4 := fetchAtlasJSON[parseFetchResponse](context.Background(), parseServer.URL+"/bad-json"); parseErr4 == nil {
		parseT.Fatal("expected invalid json response to fail")
	}

	// The first two payloads bail out before reaching a hook (no startup request /
	// blank URL), so they are safe to call directly. The third does reach
	// useAtlasCachedResource, which is a real framework hook now, so it must run
	// inside a render pass - see renderStartupPageResourceState.
	parseZeroState := useAtlasStartupPageResource(Payload{}).Get()
	if parseZeroState != (atlasCachedResourceState[any]{}) {
		parseT.Fatalf("expected empty startup resource state, got %#v", parseZeroState)
	}
	parseZeroState = useAtlasStartupPageResource(Payload{Requests: map[string]Request{"page": {URL: "   "}}}).Get()
	if parseZeroState != (atlasCachedResourceState[any]{}) {
		parseT.Fatalf("expected blank-url startup resource state, got %#v", parseZeroState)
	}
	// Asserting the not-ready zero state is the real contract, not an artifact of
	// a stub: fetch.UseCachedResource starts its loader from ui.UseEffect, and
	// native effects never run, so parseLoader is never called and no HTTP request
	// reaches parseServer during a server render.
	parseZeroState = renderStartupPageResourceState(parseT, Payload{Requests: map[string]Request{"page": {URL: parseServer.URL + "/ok"}}})
	if parseZeroState != (atlasCachedResourceState[any]{}) {
		parseT.Fatalf("expected not-ready startup resource state during native render, got %#v", parseZeroState)
	}
}

// renderStartupPageResourceState runs useAtlasStartupPageResource inside a real
// render pass and returns the cached-resource state it observed.
//
// The indirection is the whole lesson: useAtlasStartupPageResource calls a
// framework hook, and framework hooks read per-component slots off the fiber the
// runtime is currently rendering. Calling one from test-function scope panics
// with GWC-RUNTIME-HOOK-OUTSIDE-COMPONENT. It used to "work" only because the
// native hook surface was stubbed out, which is the same blind spot that let
// client/main.go ship an eager atlas.App(payload) call that white-screened the
// browser.
func renderStartupPageResourceState(parseT *testing.T, parsePayload Payload) atlasCachedResourceState[any] {
	parseT.Helper()

	var parseCaptured atlasCachedResourceState[any]
	if _, parseErr := renderAtlasNodeForTest(ui.CreateElement(func() ui.Node {
		parseCaptured = useAtlasStartupPageResource(parsePayload).Get()
		return ui.Text("")
	})); parseErr != nil {
		parseT.Fatalf("render startup resource probe: %v", parseErr)
	}
	return parseCaptured
}

// TestAtlasBootstrapDecodeBranches covers bootstrap decode failure and request cloning branches.
func TestAtlasBootstrapDecodeBranches(parseT *testing.T) {
	var parseDecoded map[string]any
	if parseErr := decodeInto(map[string]any{"bad": func() {}}, &parseDecoded); parseErr == nil {
		parseT.Fatal("expected decodeInto to fail for non-json-marshalable input")
	}

	parseRequest, parseOk := StartupRequest(Payload{
		Requests: map[string]Request{
			"page": {
				URL:  "/api/page",
				Data: map[string]any{"page": "atlas"},
			},
		},
	}, "  page  ")
	if !parseOk || parseRequest.URL != "/api/page" || parseRequest.Data["page"] != "atlas" {
		parseT.Fatalf("unexpected startup request clone ok=%v request=%#v", parseOk, parseRequest)
	}
	parseRequest.Data["page"] = "mutated"
	parseOriginal, parseOk2 := StartupRequest(Payload{
		Requests: map[string]Request{
			"page": {
				URL:  "/api/page",
				Data: map[string]any{"page": "atlas"},
			},
		},
	}, "page")
	if !parseOk2 || parseOriginal.Data["page"] != "atlas" {
		parseT.Fatalf("expected StartupRequest to clone request data, got %#v", parseOriginal)
	}
	if _, parseOk3 := StartupRequest(Payload{}, "page"); parseOk3 {
		parseT.Fatal("expected missing startup request to return false")
	}
}

// TestAtlasPageUtilityBranches covers remaining page helper branches and small rendered nodes.
func TestAtlasPageUtilityBranches(parseT *testing.T) {
	if noticeBanner(Payload{}) != nil {
		parseT.Fatal("expected empty notice banner to return nil")
	}

	parsePublicPayload := samplePayloadForRoute(RouteCatalog, sampleProductCards(), nil)
	parsePublicPayload.Route.Query["atlas_notice"] = []string{"inventory+updated"}
	parsePublicMarkup := renderAtlasMarkupForTest(parseT, noticeBanner(parsePublicPayload))
	if !strings.Contains(parsePublicMarkup, "inventory updated") || !strings.Contains(parsePublicMarkup, "rounded-[1.75rem]") {
		parseT.Fatalf("unexpected public notice banner markup %q", parsePublicMarkup)
	}

	// The console notice is a design.Surface carrying a VERIFIED status chip, not a
	// green-tinted rounded box, so the assertion is on the folded design class and the
	// semantic tone rather than on a utility class name. Asserting the emitted class is
	// the only way to check this: the class name is a hash of the rule-set.
	parseInternalPayload := samplePayloadForRoute(RouteDashboard, dashboardPage{}, nil)
	parseInternalPayload.Route.Query["atlas_notice"] = []string{"comment+submitted"}
	parseInternalMarkup := renderAtlasMarkupForTest(parseT, noticeBanner(parseInternalPayload))
	if !strings.Contains(parseInternalMarkup, "comment submitted") || !strings.Contains(parseInternalMarkup, design.Class(design.Surface(), design.Cluster(design.Space3))) {
		parseT.Fatalf("unexpected internal notice banner markup %q", parseInternalMarkup)
	}
	if !strings.Contains(parseInternalMarkup, design.Class(design.StatusChip(design.ToneVerified))) {
		parseT.Fatalf("expected internal notice to carry the verified status chip, got %q", parseInternalMarkup)
	}

	if shellToastBanner(atlasShellToast{}) != nil {
		parseT.Fatal("expected blank shell toast to return nil")
	}
	// The toast's tone no longer repaints the whole banner in a per-tone border and
	// tint; it drives ONE status chip on a plain Surface. The assertion follows: each
	// tone maps to the design system's semantic chip, and the warn case deliberately
	// shares TonePending because the palette has four tones, not five.
	parseToneCases := []struct {
		parseToast atlasShellToast
		parseWant  design.StatusTone
	}{
		{parseToast: atlasShellToast{Title: "Saved", Detail: "Inventory synced", Tone: "success"}, parseWant: design.ToneVerified},
		{parseToast: atlasShellToast{Title: "Heads up", Tone: "warn"}, parseWant: design.TonePending},
		{parseToast: atlasShellToast{Title: "Failed", Tone: "error"}, parseWant: design.ToneException},
		{parseToast: atlasShellToast{Title: "Info", Tone: "other"}, parseWant: design.TonePending},
	}
	for _, parseCase := range parseToneCases {
		parseMarkup := renderAtlasMarkupForTest(parseT, shellToastBanner(parseCase.parseToast))
		if !strings.Contains(parseMarkup, parseCase.parseToast.Title) || !strings.Contains(parseMarkup, design.Class(design.StatusChip(parseCase.parseWant))) {
			parseT.Fatalf("unexpected shell toast markup %q", parseMarkup)
		}
	}

	if renderGuidedDemoPanel(parsePublicPayload) != nil {
		parseT.Fatal("expected guided demo panel to stay hidden without demo query")
	}
	parsePublicPayload.Route.Query["demo"] = []string{"1"}
	parseDemoMarkup := renderAtlasMarkupForTest(parseT, renderGuidedDemoPanel(parsePublicPayload))
	if !strings.Contains(parseDemoMarkup, "Guided demo mode") || !strings.Contains(parseDemoMarkup, "/app/settings") {
		parseT.Fatalf("unexpected guided demo markup %q", parseDemoMarkup)
	}

	parseRouteKey := shellRouteKey("/app/dashboard", map[string][]string{"status": {"pending"}, "sort": {"updated"}})
	if !strings.Contains(parseRouteKey, "/app/dashboard?") || !strings.Contains(parseRouteKey, "sort=updated") || !strings.Contains(parseRouteKey, "status=pending") {
		parseT.Fatalf("unexpected shell route key %q", parseRouteKey)
	}

	parsePayload := Payload{
		Data: map[string]any{"page": "route-data"},
		Requests: map[string]Request{
			"page": {Data: map[string]any{"page": "request-data"}},
		},
	}
	if parseValue := payloadDataValue(parsePayload, "page"); parseValue != "request-data" {
		parseT.Fatalf("expected request-scoped page data, got %#v", parseValue)
	}
	parsePayload.Requests["page"] = Request{Data: map[string]any{"other": "value"}}
	if parseValue := payloadDataValue(parsePayload, "page"); parseValue != "route-data" {
		parseT.Fatalf("expected payload page fallback, got %#v", parseValue)
	}
	if parseValue := payloadDataValue(Payload{}, "page"); parseValue != nil {
		parseT.Fatalf("expected nil payload data fallback, got %#v", parseValue)
	}
	if parseValue := pageData(parsePayload); parseValue != "route-data" {
		parseT.Fatalf("expected pageData helper to reuse payload data, got %#v", parseValue)
	}

	parseDecodedJSON := decodeJSONText[map[string]int](`{"alerts":3}`)
	if parseDecodedJSON["alerts"] != 3 {
		parseT.Fatalf("expected decodeJSONText to decode values, got %#v", parseDecodedJSON)
	}
	if parseDecodedBlank := decodeJSONText[map[string]int]("   "); len(parseDecodedBlank) != 0 {
		parseT.Fatalf("expected blank decodeJSONText result, got %#v", parseDecodedBlank)
	}
	if parseDecodedInvalid := decodeJSONText[map[string]int]("{"); len(parseDecodedInvalid) != 0 {
		parseT.Fatalf("expected invalid decodeJSONText result to stay zero, got %#v", parseDecodedInvalid)
	}
	if parseDecodedUnsupported := decode[map[string]int](make(chan int)); len(parseDecodedUnsupported) != 0 {
		parseT.Fatalf("expected unsupported decode input to stay zero, got %#v", parseDecodedUnsupported)
	}

	if parseLines := sortedMapStrings(nil); parseLines != nil {
		parseT.Fatalf("expected nil sorted map strings for nil input, got %#v", parseLines)
	}
	parseLines := sortedMapStrings(map[string]any{"beta": 2, "alpha": 1})
	if len(parseLines) != 2 || parseLines[0] != "alpha: 1" || parseLines[1] != "beta: 2" {
		parseT.Fatalf("unexpected sorted map strings %#v", parseLines)
	}
}

// TestAtlasMutationAndNativeFallbackBranches covers remaining mutation prefixes and native no-op methods.
func TestAtlasMutationAndNativeFallbackBranches(parseT *testing.T) {
	parsePrefixCases := map[string]string{
		"/app/dashboard":       "/api/app/dashboard",
		"/app/products":        "/api/app/products",
		"/app/inventory":       "/api/app/inventory",
		"/app/warehouses":      "/api/app/warehouses",
		"/app/transfers":       "/api/app/transfers",
		"/app/purchase-orders": "/api/app/purchase-orders",
		"/app/receiving":       "/api/app/receiving",
		"/app/comments":        "/api/app/comments",
	}
	for parseInput, parseWant := range parsePrefixCases {
		parseGot := mutationRequestPrefixesForRoutePrefix(parseInput)
		if len(parseGot) == 0 || parseGot[0] != parseWant {
			parseT.Fatalf("mutationRequestPrefixesForRoutePrefix(%q) = %#v, want first %q", parseInput, parseGot, parseWant)
		}
	}

	var parseTransition atlasTransition
	if parseTransition.Pending() {
		parseT.Fatal("expected zero atlasTransition pending=false")
	}
	isParseTransitionCalled := false
	parseTransition.Start(func() {
		isParseTransitionCalled = true
	})
	if !isParseTransitionCalled {
		parseT.Fatal("expected zero atlasTransition to fall back to startAtlasTransition")
	}

	var parseThrottled atlasThrottled[int]
	if parseThrottled.Get() != 0 || parseThrottled.Pending() {
		parseT.Fatalf("expected zero atlasThrottled state, got value=%d pending=%v", parseThrottled.Get(), parseThrottled.Pending())
	}

	parseSearch := atlasSearchParams{}
	parseSearch.ReplaceAll(url.Values{"demo": {"1"}})
}
