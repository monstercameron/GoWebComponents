//go:build !js || !wasm

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestDefaultServerViewRendersToString(parseT *testing.T) {
	parseMarkup, parseErr := ui.RenderToString(renderDemoShell(defaultServerView()))
	if parseErr != nil {
		parseT.Fatalf("unexpected render error: %v", parseErr)
	}

	parseChecks := []string{
		"SSR Routing",
		"Server render first, hydrate matching DOM",
		"SSR transport and hydration",
		transportJSONSidecar,
	}
	for _, parseCheck := range parseChecks {
		if !strings.Contains(parseMarkup, parseCheck) {
			parseT.Fatalf("expected SSR markup to contain %q, got %q", parseCheck, parseMarkup)
		}
	}
}

func TestDemoBootstrapSidecarMatchesExpectedRoute(parseT *testing.T) {
	parsePath := filepath.Join("bootstrap.json")
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("unexpected bootstrap read error: %v", parseErr)
	}

	parsePayload, parseErr := ui.UnmarshalSSRBootstrap(parseData)
	if parseErr != nil {
		parseT.Fatalf("unexpected bootstrap parse error: %v", parseErr)
	}
	parseExpectedRoutePath := "/docs/" + guideSectionSSR
	if parsePayload.Route.Path != parseExpectedRoutePath {
		parseT.Fatalf("expected bootstrap route %s, got %q", parseExpectedRoutePath, parsePayload.Route.Path)
	}
	if parsePayload.Route.Params["section"] != guideSectionSSR {
		parseT.Fatalf("expected bootstrap section param %s, got %+v", guideSectionSSR, parsePayload.Route.Params)
	}
	if bootstrapTransport(parsePayload) != transportJSONSidecar {
		parseT.Fatalf("expected bootstrap transport %s, got %q", transportJSONSidecar, bootstrapTransport(parsePayload))
	}
}

func TestDemoBootstrapReferenceScriptPointsToSidecar(parseT *testing.T) {
	parseScript, parseErr := ui.RenderBootstrapReferenceScript(ui.SSRBootstrapReference{URL: "bootstrap.json", Format: ui.SSRBootstrapFormatJSON}, "")
	if parseErr != nil {
		parseT.Fatalf("unexpected reference render error: %v", parseErr)
	}
	if !strings.Contains(parseScript, "bootstrap.json") {
		parseT.Fatalf("expected bootstrap reference script to point at bootstrap.json, got %q", parseScript)
	}
}
