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
	if legacy.Redirect != "/docs/routing?tab=loader" || legacy.Status != http.StatusFound {
		t.Fatalf("expected legacy redirect, got %+v", legacy)
	}

	secure := resolveRoute("/secure", url.Values{})
	if secure.Redirect != "/signin?from=secure" || secure.Status != http.StatusFound {
		t.Fatalf("expected secure redirect, got %+v", secure)
	}
}

func TestResolveRouteBuildsServerRenderedDocsView(t *testing.T) {
	resolved := resolveRoute("/docs/ssr", url.Values{"tab": {"loader"}, "refresh": {"3"}})
	if resolved.Status != http.StatusOK {
		t.Fatalf("expected OK status, got %d", resolved.Status)
	}
	if resolved.View.SectionID != "ssr" || resolved.View.CurrentTab != "loader" || resolved.View.Revision != 3 {
		t.Fatalf("unexpected docs view: %+v", resolved.View)
	}
	if resolved.Bootstrap.Route.Path != "/docs/ssr" {
		t.Fatalf("expected bootstrap path /docs/ssr, got %+v", resolved.Bootstrap.Route)
	}
}

func TestBootstrapReferenceURLIncludesRouteAndQuery(t *testing.T) {
	query := url.Values{"tab": {"loader"}, "q": {"routing"}}
	ref := bootstrapReferenceURL("/docs/ssr", query)
	if !strings.Contains(ref, "path=%2Fdocs%2Fssr") || !strings.Contains(ref, "tab=loader") || !strings.Contains(ref, "q=routing") {
		t.Fatalf("unexpected bootstrap reference url: %s", ref)
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
	}
	for _, check := range checks {
		if !strings.Contains(body, check) {
			t.Fatalf("expected SSR document to contain %q, got %q", check, body)
		}
	}
}
