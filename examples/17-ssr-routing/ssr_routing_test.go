//go:build !js || !wasm
// +build !js !wasm

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
		transportJSONSidecar,
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
	expectedRoutePath := "/docs/" + guideSectionSSR
	if payload.Route.Path != expectedRoutePath {
		t.Fatalf("expected bootstrap route %s, got %q", expectedRoutePath, payload.Route.Path)
	}
	if payload.Route.Params["section"] != guideSectionSSR {
		t.Fatalf("expected bootstrap section param %s, got %+v", guideSectionSSR, payload.Route.Params)
	}
	if bootstrapTransport(payload) != transportJSONSidecar {
		t.Fatalf("expected bootstrap transport %s, got %q", transportJSONSidecar, bootstrapTransport(payload))
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
