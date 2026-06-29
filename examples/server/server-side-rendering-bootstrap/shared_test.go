//go:build !js || !wasm

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestBootstrapHelpersAndRenderedView(parseT *testing.T) {
	parsePayload := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: ""},
		Data:  map[string]any{"message": "hydrated"},
	}
	parseView := bootstrapViewFromPayload(parsePayload)
	if parseView.Path != "/bootstrap" || parseView.Message != "hydrated" {
		parseT.Fatalf("bootstrapViewFromPayload() = %+v, want default path and custom message", parseView)
	}

	parseMarkup, parseErr := ui.RenderToString(renderBootstrapView(parseView))
	if parseErr != nil {
		parseT.Fatalf("RenderToString(renderBootstrapView) error = %v", parseErr)
	}
	for _, parseExpected := range []string{"Dedicated SSR bootstrap", "hydrated", "/bootstrap"} {
		if !strings.Contains(parseMarkup, parseExpected) {
			parseT.Fatalf("rendered bootstrap markup missing %q\n%s", parseExpected, parseMarkup)
		}
	}
}

func TestBootstrapRequestReportAndRepoRoot(parseT *testing.T) {
	parseReport := bootstrapRequestReport("/orders", errors.New("render failed"), "request could not complete", "inspect the server log")
	if parseReport.Code != "GWC-EXAMPLE-SERVER-REQUEST" || parseReport.Path != "/orders" || !strings.Contains(parseReport.Runtime, "request could not complete") || !strings.Contains(parseReport.Next, "inspect the server log") {
		parseT.Fatalf("bootstrapRequestReport() = %+v, want populated diagnostics report", parseReport)
	}

	parseRoot := parseT.TempDir()
	parseNested := filepath.Join(parseRoot, "examples", "73-ssr-bootstrap")
	if parseErr := os.MkdirAll(parseNested, 0755); parseErr != nil {
		parseT.Fatalf("MkdirAll() error = %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/test\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("WriteFile(go.mod) error = %v", parseErr2)
	}
	parseResolved, parseErr3 := repoRoot(parseNested)
	if parseErr3 != nil {
		parseT.Fatalf("repoRoot() error = %v", parseErr3)
	}
	if parseResolved != parseRoot {
		parseT.Fatalf("repoRoot() = %q, want %q", parseResolved, parseRoot)
	}

	if _, parseErr4 := repoRoot(parseT.TempDir()); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "could not find repo root") {
		parseT.Fatalf("repoRoot() error = %v, want missing go.mod error", parseErr4)
	}
}
