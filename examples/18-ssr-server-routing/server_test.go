//go:build !js || !wasm
// +build !js !wasm

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

func testServer(t *testing.T) *appServer {
	t.Helper()
	workingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("unexpected getwd error: %v", err)
	}
	root, err := findRepoRoot(workingDir)
	if err != nil {
		t.Fatalf("unexpected repo root resolution error: %v", err)
	}
	return newAppServer(root)
}

func TestResolveRouteRedirectsAndSecureGuard(t *testing.T) {
	legacy := resolveRoute("/legacy", url.Values{})
	if legacy.Redirect != legacyRedirectPath || legacy.Status != http.StatusFound {
		t.Fatalf("expected legacy redirect, got %+v", legacy)
	}

	secure := resolveRoute("/secure", url.Values{})
	if secure.Redirect != secureRedirectPath || secure.Status != http.StatusFound {
		t.Fatalf("expected secure redirect, got %+v", secure)
	}
}

func TestResolveRouteBuildsServerRenderedDocsView(t *testing.T) {
	resolved := resolveRoute("/docs/"+serverGuideSectionSSR, url.Values{"tab": {serverTabLoader}, "refresh": {"3"}})
	if resolved.Status != http.StatusOK {
		t.Fatalf("expected OK status, got %d", resolved.Status)
	}
	if resolved.View.SectionID != serverGuideSectionSSR || resolved.View.CurrentTab != serverTabLoader || resolved.View.Revision != 3 {
		t.Fatalf("unexpected docs view: %+v", resolved.View)
	}
	expectedRoutePath := "/docs/" + serverGuideSectionSSR
	if resolved.Bootstrap.Route.Path != expectedRoutePath {
		t.Fatalf("expected bootstrap path %s, got %+v", expectedRoutePath, resolved.Bootstrap.Route)
	}
}

func TestBootstrapReferenceURLIncludesRouteAndQuery(t *testing.T) {
	query := url.Values{"tab": {serverTabLoader}, "q": {serverGuideSectionRouting}}
	bootstrapURL := bootstrapReferenceURL("/docs/"+serverGuideSectionSSR, query)
	if !strings.Contains(bootstrapURL, "path=%2Fdocs%2Fssr") || !strings.Contains(bootstrapURL, "tab=loader") || !strings.Contains(bootstrapURL, "q=routing") {
		t.Fatalf("unexpected bootstrap reference url: %s", bootstrapURL)
	}
}

func TestBootstrapEndpointServesRouteSpecificPayload(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, bootstrapEndpointPath+"?path=%2Fsearch&q=routing", nil)
	res := httptest.NewRecorder()

	server.handleBootstrap(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected bootstrap endpoint OK, got %d", res.Code)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unexpected bootstrap json error: %v", err)
	}
	route := payload["route"].(map[string]interface{})
	if route["path"] != "/search" {
		t.Fatalf("expected bootstrap path /search, got %#v", route["path"])
	}
}

func TestPageHandlerRendersSSRDocument(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader", nil)
	res := httptest.NewRecorder()

	server.handlePage(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected page handler OK, got %d", res.Code)
	}
	body := res.Body.String()
	checks := []string{
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
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected SSR document to contain %q, got %q", check, body)
		}
	}
	if got := strings.Count(body, `data-gwc-router-managed="true"`); got != 3 {
		t.Fatalf("expected exactly 3 managed head tags in SSR output, got %d in %q", got, body)
	}
	if got := strings.Count(body, `<title `); got != 1 {
		t.Fatalf("expected exactly one title tag in SSR output, got %d in %q", got, body)
	}
	if !strings.Contains(body, `name="description"`) {
		t.Fatalf("expected SSR document to include description metadata, got %q", body)
	}
	if !strings.Contains(body, `href="http://127.0.0.1:8079/docs/ssr?tab=loader"`) {
		t.Fatalf("expected SSR document to include route canonical URL, got %q", body)
	}
	if !strings.Contains(body, `content="index,follow"`) {
		t.Fatalf("expected SSR document to include robots metadata, got %q", body)
	}
	if !strings.Contains(body, `id="route-jsonld"`) {
		t.Fatalf("expected SSR document to include JSON-LD markup, got %q", body)
	}
}

func TestPageHandlerStreamsDeferredDocsPanel(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader", nil)
	res := httptest.NewRecorder()

	server.handlePage(res, req)
	if !res.Flushed {
		t.Fatal("expected streamed docs response to flush shell output")
	}
	body := res.Body.String()
	checks := []string{
		`id="` + serverDeferredPanelID + `"`,
		"Streaming nested docs panel...",
		"Streamed docs insights ready",
		"target.outerHTML=",
		"ssr-server-routing.wasm",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected streamed document to contain %q, got %q", check, body)
		}
	}
}

func TestPageHandlerStreamsDeferredDocsErrorReplacement(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader&stream=error", nil)
	res := httptest.NewRecorder()

	server.handlePage(res, req)
	if !res.Flushed {
		t.Fatal("expected streamed docs error response to flush shell output")
	}
	body := res.Body.String()
	checks := []string{
		"Deferred docs panel failed after shell flush",
		"explicit error region",
		"target.outerHTML=",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected streamed error document to contain %q, got %q", check, body)
		}
	}
}

func TestPageHandlerStreamsNestedDocsLayout(t *testing.T) {
	server := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/docs/ssr?tab=loader&stream=nested", nil)
	res := httptest.NewRecorder()

	server.handlePage(res, req)
	if !res.Flushed {
		t.Fatal("expected streamed nested docs response to flush shell output")
	}
	body := res.Body.String()
	checks := []string{
		"Nested streamed layout ready",
		"Outer layout",
		"Nested child panel",
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected streamed nested-layout document to contain %q, got %q", check, body)
		}
	}
}
