package prerender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

func TestExportWritesHTMLAndBootstrapFiles(t *testing.T) {
	outputDir := t.TempDir()

	summary, err := Export(outputDir, []Route{
		{
			Path:            "/",
			BootstrapFormat: ui.SSRBootstrapFormatJSON,
			Build: func(target Target) (RouteOutput, error) {
				if target.HTMLFile != "index.html" {
					t.Fatalf("unexpected root html file: %s", target.HTMLFile)
				}
				if target.BootstrapURL != "/bootstrap/index.json" {
					t.Fatalf("unexpected root bootstrap url: %s", target.BootstrapURL)
				}
				return RouteOutput{
					HTML:      "<!doctype html><html><body>home</body></html>",
					Bootstrap: []byte(`{"page":"home"}`),
				}, nil
			},
		},
		{
			Path:            "/docs/getting-started",
			BootstrapFormat: ui.SSRBootstrapFormatCBOR,
			Build: func(target Target) (RouteOutput, error) {
				if target.HTMLFile != "docs/getting-started/index.html" {
					t.Fatalf("unexpected nested html file: %s", target.HTMLFile)
				}
				if target.BootstrapURL != "/bootstrap/docs/getting-started.cbor" {
					t.Fatalf("unexpected nested bootstrap url: %s", target.BootstrapURL)
				}
				return RouteOutput{
					HTML:      "<!doctype html><html><body>docs</body></html>",
					Bootstrap: []byte{1, 2, 3, 4},
				}, nil
			},
		},
	})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if len(summary.HTMLFiles) != 2 {
		t.Fatalf("expected 2 html files, got %d", len(summary.HTMLFiles))
	}
	if len(summary.BootstrapFiles) != 2 {
		t.Fatalf("expected 2 bootstrap files, got %d", len(summary.BootstrapFiles))
	}

	assertFileContains(t, filepath.Join(outputDir, "index.html"), "home")
	assertFileContains(t, filepath.Join(outputDir, "docs", "getting-started", "index.html"), "docs")
	assertFileContains(t, filepath.Join(outputDir, "bootstrap", "index.json"), `"page":"home"`)
	assertFileBytes(t, filepath.Join(outputDir, "bootstrap", "docs", "getting-started.cbor"), []byte{1, 2, 3, 4})
}

func TestExportRejectsInvalidRoutes(t *testing.T) {
	outputDir := t.TempDir()

	_, err := Export(outputDir, []Route{{
		Path:  "docs",
		Build: func(Target) (RouteOutput, error) { return RouteOutput{HTML: "nope"}, nil },
	}})
	if err == nil || !strings.Contains(err.Error(), "must start with '/'") {
		t.Fatalf("expected invalid route error, got %v", err)
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q: %s", path, want, string(data))
	}
}

func assertFileBytes(t *testing.T, path string, want []byte) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != string(want) {
		t.Fatalf("%s bytes mismatch: got %v want %v", path, data, want)
	}
}
