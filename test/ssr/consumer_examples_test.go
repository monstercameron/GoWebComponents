package ssr_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	ssr "github.com/monstercameron/GoWebComponents/test/ssr"
	"github.com/monstercameron/GoWebComponents/ui"
)

func TestConsumerSSRPattern_SnapshotAndPayload(t *testing.T) {
	bootstrap := ui.SSRBootstrap{}
	if err := ui.RegisterBootstrapPayload(&bootstrap, "viewer", map[string]string{"name": "Cam"}); err != nil {
		t.Fatalf("expected bootstrap registration to succeed, got %v", err)
	}

	snapshot := ssr.Render(t, html.Main(html.Props{ID: "ssr-page"}, html.Text("SSR page")))
	viewer := ssr.RequirePayload[map[string]string](t, bootstrap, "viewer")

	if !snapshot.Contains("SSR page") {
		t.Fatalf("expected snapshot to contain rendered HTML, got %q", snapshot.HTML)
	}
	if viewer.Value["name"] != "Cam" {
		t.Fatalf("expected typed bootstrap payload, got %+v", viewer.Value)
	}
}

func TestConsumerSSRPattern_LoadStaticExport(t *testing.T) {
	outputDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outputDir, "index.html"), []byte("<!doctype html><html><body>home</body></html>"), 0o644); err != nil {
		t.Fatalf("write html snapshot: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "bootstrap"), 0o755); err != nil {
		t.Fatalf("create bootstrap dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(outputDir, "bootstrap", "index.json"), []byte(`{"page":"home"}`), 0o644); err != nil {
		t.Fatalf("write bootstrap sidecar: %v", err)
	}

	export := ssr.LoadStaticExport(t, outputDir)
	route, err := export.Route("/")
	if err != nil {
		t.Fatalf("resolve exported root route: %v", err)
	}
	if route.HTMLFile != "index.html" {
		t.Fatalf("expected root html file index.html, got %q", route.HTMLFile)
	}
	if route.BootstrapFile != filepath.ToSlash(filepath.Join("bootstrap", "index.json")) {
		t.Fatalf("expected root bootstrap file, got %q", route.BootstrapFile)
	}
}
