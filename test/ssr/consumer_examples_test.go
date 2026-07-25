package ssr_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/html"
	ssr "github.com/monstercameron/GoWebComponents/v5/test/ssr"
	"github.com/monstercameron/GoWebComponents/v5/ui"
)

func TestConsumerSSRPattern_SnapshotAndPayload(parseT *testing.T) {
	parseBootstrap := ui.SSRBootstrap{}
	if parseErr := ui.RegisterBootstrapPayload(&parseBootstrap, "viewer", map[string]string{"name": "Cam"}); parseErr != nil {
		parseT.Fatalf("expected bootstrap registration to succeed, got %v", parseErr)
	}

	parseSnapshot := ssr.Render(parseT, html.Main(html.Props{ID: "ssr-page"}, html.Text("SSR page")))
	parseViewer := ssr.RequirePayload[map[string]string](parseT, parseBootstrap, "viewer")

	if !parseSnapshot.Contains("SSR page") {
		parseT.Fatalf("expected snapshot to contain rendered HTML, got %q", parseSnapshot.HTML)
	}
	if parseViewer.Value["name"] != "Cam" {
		parseT.Fatalf("expected typed bootstrap payload, got %+v", parseViewer.Value)
	}
}

func TestConsumerSSRPattern_LoadStaticExport(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseOutputDir, "index.html"), []byte("<!doctype html><html><body>home</body></html>"), 0o644); parseErr != nil {
		parseT.Fatalf("write html snapshot: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Join(parseOutputDir, "bootstrap"), 0o755); parseErr2 != nil {
		parseT.Fatalf("create bootstrap dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseOutputDir, "bootstrap", "index.json"), []byte(`{"page":"home"}`), 0o644); parseErr3 != nil {
		parseT.Fatalf("write bootstrap sidecar: %v", parseErr3)
	}

	parseExport := ssr.LoadStaticExport(parseT, parseOutputDir)
	parseRoute, parseErr4 := parseExport.Route("/")
	if parseErr4 != nil {
		parseT.Fatalf("resolve exported root route: %v", parseErr4)
	}
	if parseRoute.HTMLFile != "index.html" {
		parseT.Fatalf("expected root html file index.html, got %q", parseRoute.HTMLFile)
	}
	if parseRoute.BootstrapFile != filepath.ToSlash(filepath.Join("bootstrap", "index.json")) {
		parseT.Fatalf("expected root bootstrap file, got %q", parseRoute.BootstrapFile)
	}
}
