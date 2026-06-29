//go:build !js || !wasm

package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func testServer(parseT *testing.T) *appServer {
	parseT.Helper()
	parseWorkingDir, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("unexpected getwd error: %v", parseErr)
	}
	parseRoot, parseErr := findRepoRoot(parseWorkingDir)
	if parseErr != nil {
		parseT.Fatalf("unexpected repo root resolution error: %v", parseErr)
	}
	return newAppServer(parseRoot)
}

func TestResolveRouteRedirectsAndSecureGuard(parseT *testing.T) {
	parseLegacy := resolveRoute("/legacy", url.Values{})
	if parseLegacy.Redirect != legacyRedirectPath || parseLegacy.Status != http.StatusFound {
		parseT.Fatalf("expected legacy redirect, got %+v", parseLegacy)
	}

	parseSecure := resolveRoute("/secure", url.Values{})
	if parseSecure.Redirect != secureRedirectPath || parseSecure.Status != http.StatusFound {
		parseT.Fatalf("expected secure redirect, got %+v", parseSecure)
	}
}

func TestResolveRouteBuildsServerRenderedDocsView(parseT *testing.T) {
	parseResolved := resolveRoute("/docs/"+serverGuideSectionSSR, url.Values{"tab": {serverTabLoader}, "refresh": {"3"}})
	if parseResolved.Status != http.StatusOK {
		parseT.Fatalf("expected OK status, got %d", parseResolved.Status)
	}
	if parseResolved.View.SectionID != serverGuideSectionSSR || parseResolved.View.CurrentTab != serverTabLoader || parseResolved.View.Revision != 3 {
		parseT.Fatalf("unexpected docs view: %+v", parseResolved.View)
	}
	parseExpectedRoutePath := "/docs/" + serverGuideSectionSSR
	if parseResolved.Bootstrap.Route.Path != parseExpectedRoutePath {
		parseT.Fatalf("expected bootstrap path %s, got %+v", parseExpectedRoutePath, parseResolved.Bootstrap.Route)
	}
}

func TestBootstrapReferenceURLIncludesRouteAndQuery(parseT *testing.T) {
	parseQuery := url.Values{"tab": {serverTabLoader}, "q": {serverGuideSectionRouting}}
	parseBootstrapURL := bootstrapReferenceURL("/docs/"+serverGuideSectionSSR, parseQuery)
	if !strings.Contains(parseBootstrapURL, "path=%2Fdocs%2Fssr") || !strings.Contains(parseBootstrapURL, "tab=loader") || !strings.Contains(parseBootstrapURL, "q=routing") {
		parseT.Fatalf("unexpected bootstrap reference url: %s", parseBootstrapURL)
	}
}

func TestBootstrapEndpointServesRouteSpecificPayload(parseT *testing.T) {
	parseServer := testServer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, bootstrapEndpointPath+"?path=%2Fsearch&q=routing", nil)
	parseRes := httptest.NewRecorder()

	parseServer.handleBootstrap(parseRes, parseReq)
	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected bootstrap endpoint OK, got %d", parseRes.Code)
	}
	var parsePayload map[string]any
	if parseErr := json.Unmarshal(parseRes.Body.Bytes(), &parsePayload); parseErr != nil {
		parseT.Fatalf("unexpected bootstrap json error: %v", parseErr)
	}
	parseRoute := parsePayload["route"].(map[string]any)
	if parseRoute["path"] != "/search" {
		parseT.Fatalf("expected bootstrap path /search, got %#v", parseRoute["path"])
	}
}

func TestPageHandlerRendersSSRDocument(parseT *testing.T) {
	parseServer := testServer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader", nil)
	parseRes := httptest.NewRecorder()

	parseServer.handlePage(parseRes, parseReq)
	if parseRes.Code != http.StatusOK {
		parseT.Fatalf("expected page handler OK, got %d", parseRes.Code)
	}
	parseBody := parseRes.Body.String()
	parseChecks := []string{
		"Server SSR Demo",
		"Server-rendered bootstrap flow",
		bootstrapEndpointPath,
		"ssr-server-routing.wasm",
		`data-gwc-router-managed="true"`,
		`rel="canonical"`,
		`name="robots"`,
		`property="og:title"`,
		`type="application/ld+json"`,
		`rel="preload"`,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseBody, parseCheck) {
			parseT.Fatalf("expected SSR document to contain %q, got %q", parseCheck, parseBody)
		}
	}
	if parseGot := strings.Count(parseBody, `data-gwc-router-managed="true"`); parseGot != 3 {
		parseT.Fatalf("expected exactly 3 managed head tags in SSR output, got %d in %q", parseGot, parseBody)
	}
	if parseGot2 := strings.Count(parseBody, `<title `); parseGot2 != 1 {
		parseT.Fatalf("expected exactly one title tag in SSR output, got %d in %q", parseGot2, parseBody)
	}
	if !strings.Contains(parseBody, `name="description"`) {
		parseT.Fatalf("expected SSR document to include description metadata, got %q", parseBody)
	}
	if !strings.Contains(parseBody, `href="http://127.0.0.1:8079/docs/ssr?tab=loader"`) {
		parseT.Fatalf("expected SSR document to include route canonical URL, got %q", parseBody)
	}
	if !strings.Contains(parseBody, `content="index,follow"`) {
		parseT.Fatalf("expected SSR document to include robots metadata, got %q", parseBody)
	}
	if !strings.Contains(parseBody, `id="route-jsonld"`) {
		parseT.Fatalf("expected SSR document to include JSON-LD markup, got %q", parseBody)
	}
}

func TestPageHandlerStreamsDeferredDocsPanel(parseT *testing.T) {
	parseServer := testServer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader", nil)
	parseRes := httptest.NewRecorder()

	parseServer.handlePage(parseRes, parseReq)
	if !parseRes.Flushed {
		parseT.Fatal("expected streamed docs response to flush shell output")
	}
	parseBody := parseRes.Body.String()
	parseChecks := []string{
		`id="` + serverDeferredPanelID + `"`,
		"Streaming nested docs panel...",
		"Streamed docs insights ready",
		"target.outerHTML=",
		"ssr-server-routing.wasm",
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseBody, parseCheck) {
			parseT.Fatalf("expected streamed document to contain %q, got %q", parseCheck, parseBody)
		}
	}
}

func TestPageHandlerStreamsDeferredDocsErrorReplacement(parseT *testing.T) {
	parseServer := testServer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader&stream=error", nil)
	parseRes := httptest.NewRecorder()

	parseServer.handlePage(parseRes, parseReq)
	if !parseRes.Flushed {
		parseT.Fatal("expected streamed docs error response to flush shell output")
	}
	parseBody := parseRes.Body.String()
	parseChecks := []string{
		"Deferred docs panel failed after shell flush",
		"explicit error region",
		"target.outerHTML=",
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseBody, parseCheck) {
			parseT.Fatalf("expected streamed error document to contain %q, got %q", parseCheck, parseBody)
		}
	}
}

func TestPageHandlerStreamsNestedDocsLayout(parseT *testing.T) {
	parseServer := testServer(parseT)
	parseReq := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader&stream=nested", nil)
	parseRes := httptest.NewRecorder()

	parseServer.handlePage(parseRes, parseReq)
	if !parseRes.Flushed {
		parseT.Fatal("expected streamed nested docs response to flush shell output")
	}
	parseBody := parseRes.Body.String()
	parseChecks := []string{
		"Nested streamed layout ready",
		"Outer layout",
		"Nested child panel",
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseBody, parseCheck) {
			parseT.Fatalf("expected streamed nested-layout document to contain %q, got %q", parseCheck, parseBody)
		}
	}
}
