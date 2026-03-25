package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticExportExampleWritesMultipleRoutes(t *testing.T) {
	outputDir := t.TempDir()

	summary, err := exportExampleSite(outputDir)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if len(summary.HTMLFiles) != len(exportedPages) {
		t.Fatalf("expected %d html files, got %d", len(exportedPages), len(summary.HTMLFiles))
	}

	assertFileContains(t, filepath.Join(outputDir, "index.html"), "Prerender several routes once")
	assertFileContains(t, filepath.Join(outputDir, "index.html"), `rel="preload"`)
	assertFileContains(t, filepath.Join(outputDir, "index.html"), `srcset=`)
	assertFileContains(t, filepath.Join(outputDir, "index.html"), `loading="lazy"`)
	assertFileContains(t, filepath.Join(outputDir, "pricing", "index.html"), "pricing page is just another prerendered route")
	assertFileContains(t, filepath.Join(outputDir, "pricing", "index.html"), `rel="prefetch"`)
	assertFileContains(t, filepath.Join(outputDir, "docs", "getting-started", "index.html"), "Nested paths export to nested folders")
	assertFileContains(t, filepath.Join(outputDir, "static", "site-export.4f3a2b1c.css"), ".media-frame")
	assertFileContains(t, filepath.Join(outputDir, "static", "media", "hero-home.2f71c1a0.svg"), "Static export home")
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(data), want) {
		t.Fatalf("%s does not contain %q", path, want)
	}
}
