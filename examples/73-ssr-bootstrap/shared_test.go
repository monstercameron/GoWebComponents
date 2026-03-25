//go:build !js || !wasm
// +build !js !wasm

package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestBootstrapHelpersAndRenderedView(t *testing.T) {
	payload := ui.SSRBootstrap{
		Route: ui.SSRRouteBootstrap{Path: ""},
		Data:  map[string]interface{}{"message": "hydrated"},
	}
	view := bootstrapViewFromPayload(payload)
	if view.Path != "/bootstrap" || view.Message != "hydrated" {
		t.Fatalf("bootstrapViewFromPayload() = %+v, want default path and custom message", view)
	}

	markup, err := ui.RenderToString(renderBootstrapView(view))
	if err != nil {
		t.Fatalf("RenderToString(renderBootstrapView) error = %v", err)
	}
	for _, expected := range []string{"Dedicated SSR bootstrap", "hydrated", "/bootstrap"} {
		if !strings.Contains(markup, expected) {
			t.Fatalf("rendered bootstrap markup missing %q\n%s", expected, markup)
		}
	}
}

func TestBootstrapRequestReportAndRepoRoot(t *testing.T) {
	report := bootstrapRequestReport("/orders", errors.New("render failed"), "request could not complete", "inspect the server log")
	if report.Code != "GWC-EXAMPLE-SERVER-REQUEST" || report.Path != "/orders" || !strings.Contains(report.Runtime, "request could not complete") || !strings.Contains(report.Next, "inspect the server log") {
		t.Fatalf("bootstrapRequestReport() = %+v, want populated diagnostics report", report)
	}

	root := t.TempDir()
	nested := filepath.Join(root, "examples", "73-ssr-bootstrap")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}
	resolved, err := repoRoot(nested)
	if err != nil {
		t.Fatalf("repoRoot() error = %v", err)
	}
	if resolved != root {
		t.Fatalf("repoRoot() = %q, want %q", resolved, root)
	}

	if _, err := repoRoot(t.TempDir()); err == nil || !strings.Contains(err.Error(), "could not find repo root") {
		t.Fatalf("repoRoot() error = %v, want missing go.mod error", err)
	}
}
