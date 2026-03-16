package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestDefaultServerViewRendersToString(t *testing.T) {
	markup, err := ui.RenderToString(renderDemoShell(defaultServerView()))
	if err != nil {
		t.Fatalf("unexpected render error: %v", err)
	}

	checks := []string{
		"SSR Routing Demo",
		"Server render first, hydrate into advanced routes",
		"SSR transport and hydration",
		"json-sidecar",
	}
	for _, check := range checks {
		if !strings.Contains(markup, check) {
			t.Fatalf("expected SSR markup to contain %q, got %q", check, markup)
		}
	}
}

func TestDemoBootstrapSidecarMatchesExpectedRoute(t *testing.T) {
	path := filepath.Join("bootstrap.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("unexpected bootstrap read error: %v", err)
	}

	payload, err := ui.UnmarshalSSRBootstrap(data)
	if err != nil {
		t.Fatalf("unexpected bootstrap parse error: %v", err)
	}
	if payload.Route.Path != "/docs/ssr" {
		t.Fatalf("expected bootstrap route /docs/ssr, got %q", payload.Route.Path)
	}
	if payload.Route.Params["section"] != "ssr" {
		t.Fatalf("expected bootstrap section param ssr, got %+v", payload.Route.Params)
	}
	if bootstrapTransport(payload) != "json-sidecar" {
		t.Fatalf("expected bootstrap transport json-sidecar, got %q", bootstrapTransport(payload))
	}
}

func TestDemoBootstrapReferenceScriptPointsToSidecar(t *testing.T) {
	script, err := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{URL: "bootstrap.json", Format: ui.SSRBootstrapFormatJSON}, "")
	if err != nil {
		t.Fatalf("unexpected reference render error: %v", err)
	}
	if !strings.Contains(script, "bootstrap.json") {
		t.Fatalf("expected bootstrap reference script to point at bootstrap.json, got %q", script)
	}
}
