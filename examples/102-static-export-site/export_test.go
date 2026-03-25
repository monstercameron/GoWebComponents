package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticExportExampleWritesMultipleRoutes(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()

	parseSummary, parseErr := exportExampleSite(parseOutputDir)
	if parseErr != nil {
		parseT.Fatalf("Export() error = %v", parseErr)
	}
	if len(parseSummary.HTMLFiles) != len(exportedPages) {
		parseT.Fatalf("expected %d html files, got %d", len(exportedPages), len(parseSummary.HTMLFiles))
	}

	assertFileContains(parseT, filepath.Join(parseOutputDir, "index.html"), "Prerender several routes once")
	assertFileContains(parseT, filepath.Join(parseOutputDir, "index.html"), `rel="preload"`)
	assertFileContains(parseT, filepath.Join(parseOutputDir, "index.html"), `srcset=`)
	assertFileContains(parseT, filepath.Join(parseOutputDir, "index.html"), `loading="lazy"`)
	assertFileContains(parseT, filepath.Join(parseOutputDir, "pricing", "index.html"), "pricing page is just another prerendered route")
	assertFileContains(parseT, filepath.Join(parseOutputDir, "pricing", "index.html"), `rel="prefetch"`)
	assertFileContains(parseT, filepath.Join(parseOutputDir, "docs", "getting-started", "index.html"), "Nested paths export to nested folders")
	assertFileContains(parseT, filepath.Join(parseOutputDir, "static", "site-export.4f3a2b1c.css"), ".media-frame")
	assertFileContains(parseT, filepath.Join(parseOutputDir, "static", "media", "hero-home.2f71c1a0.svg"), "Static export home")
}

func assertFileContains(parseT *testing.T, parsePath string, parseWant string) {
	parseT.Helper()
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("read %s: %v", parsePath, parseErr)
	}
	if !strings.Contains(string(parseData), parseWant) {
		parseT.Fatalf("%s does not contain %q", parsePath, parseWant)
	}
}
