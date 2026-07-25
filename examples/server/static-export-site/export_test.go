//go:build !js || !wasm

package main

import (
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/prerender"
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

// TestExportRoutesBuildExpectedMarkup verifies every exported route builder returns the expected page HTML.
func TestExportRoutesBuildExpectedMarkup(parseT *testing.T) {
	parseRoutes := exportRoutes()
	if len(parseRoutes) != len(exportedPages) {
		parseT.Fatalf("exportRoutes len = %d, want %d", len(parseRoutes), len(exportedPages))
	}

	for parseIndex, parseRoute := range parseRoutes {
		parsePage := exportedPages[parseIndex]
		if parseRoute.Path != parsePage.Path {
			parseT.Fatalf("route %d path = %q, want %q", parseIndex, parseRoute.Path, parsePage.Path)
		}

		parseOutput, parseErr := parseRoute.Build(prerender.Target{})
		if parseErr != nil {
			parseT.Fatalf("route %q build error = %v", parseRoute.Path, parseErr)
		}
		if !strings.Contains(parseOutput.HTML, parsePage.Headline) {
			parseT.Fatalf("route %q html missing headline %q", parseRoute.Path, parsePage.Headline)
		}
	}
}

// TestStaticExportAssetHelpersCoverFallbacks verifies asset lookup, srcset generation, and asset-root fallback branches.
func TestStaticExportAssetHelpersCoverFallbacks(parseT *testing.T) {
	if parseGot := assetURL("missing"); parseGot != "" {
		parseT.Fatalf("assetURL(missing) = %q, want empty string", parseGot)
	}
	if parseGot := responsiveSrcSet("missing"); parseGot != "" {
		parseT.Fatalf("responsiveSrcSet(missing) = %q, want empty string", parseGot)
	}

	parseFirstRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseFirstRoot, "assets"), 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(first assets) error = %v", parseErr)
	}
	parseT.Run("first-candidate", func(parseSubT *testing.T) {
		parseSubT.Chdir(parseFirstRoot)
		parseResolved, parseErr := resolveExampleAssetRoot()
		if parseErr != nil {
			parseSubT.Fatalf("resolveExampleAssetRoot first candidate error = %v", parseErr)
		}
		if parseResolved != "assets" {
			parseSubT.Fatalf("resolveExampleAssetRoot first candidate = %q, want assets", parseResolved)
		}
	})

	parseSecondRoot := parseT.TempDir()
	parseSecondAssets := filepath.Join(parseSecondRoot, "examples", "102-static-export-site", "assets")
	if parseErr := os.MkdirAll(parseSecondAssets, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(second assets) error = %v", parseErr)
	}
	parseT.Run("second-candidate", func(parseSubT *testing.T) {
		parseSubT.Chdir(parseSecondRoot)
		parseResolved, parseErr := resolveExampleAssetRoot()
		if parseErr != nil {
			parseSubT.Fatalf("resolveExampleAssetRoot second candidate error = %v", parseErr)
		}
		if parseResolved != filepath.Join("examples", "102-static-export-site", "assets") {
			parseSubT.Fatalf("resolveExampleAssetRoot second candidate = %q", parseResolved)
		}
	})

	parseT.Run("missing-assets", func(parseSubT *testing.T) {
		parseSubT.Chdir(parseSubT.TempDir())
		if _, parseErr := resolveExampleAssetRoot(); parseErr == nil || !strings.Contains(parseErr.Error(), "assets directory not found") {
			parseSubT.Fatalf("resolveExampleAssetRoot missing assets error = %v", parseErr)
		}
	})
}

// TestCopyExampleAssetsAndExportFailure verifies asset copying succeeds and exportExampleSite propagates missing-asset failures.
func TestCopyExampleAssetsAndExportFailure(parseT *testing.T) {
	parseSourceRoot := parseT.TempDir()
	parseAssetsRoot := filepath.Join(parseSourceRoot, "assets", "static", "media")
	if parseErr := os.MkdirAll(parseAssetsRoot, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(copy assets) error = %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseSourceRoot, "assets", "static", "site.css"), []byte("body{}"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile(site.css) error = %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseAssetsRoot, "hero.svg"), []byte("<svg/>"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile(hero.svg) error = %v", parseErr)
	}

	parseT.Run("copy-assets", func(parseSubT *testing.T) {
		parseSubT.Chdir(parseSourceRoot)
		parseOutputDir := parseSubT.TempDir()
		if parseErr := copyExampleAssets(parseOutputDir); parseErr != nil {
			parseSubT.Fatalf("copyExampleAssets error = %v", parseErr)
		}
		assertFileContains(parseSubT, filepath.Join(parseOutputDir, "static", "site.css"), "body{}")
		assertFileContains(parseSubT, filepath.Join(parseOutputDir, "static", "media", "hero.svg"), "<svg/>")
	})

	parseT.Run("missing-assets", func(parseSubT *testing.T) {
		parseSubT.Chdir(parseSubT.TempDir())
		if _, parseErr := exportExampleSite(parseSubT.TempDir()); parseErr == nil || !strings.Contains(parseErr.Error(), "assets directory not found") {
			parseSubT.Fatalf("exportExampleSite missing assets error = %v", parseErr)
		}
	})
}

// TestMainWritesStaticExport verifies the command entrypoint exports the site to the requested directory.
func TestMainWritesStaticExport(parseT *testing.T) {
	parseOutputDir := parseT.TempDir()

	parseOldArgs := os.Args
	parseOldStdout := os.Stdout
	parseOldCommandLine := flag.CommandLine
	parseReadPipe, parseWritePipe, parseErr := os.Pipe()
	if parseErr != nil {
		parseT.Fatalf("os.Pipe() error = %v", parseErr)
	}
	flag.CommandLine = flag.NewFlagSet("static-export-test", flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = []string{"static-export-test", "-out", parseOutputDir}
	os.Stdout = parseWritePipe
	parseT.Cleanup(func() {
		flag.CommandLine = parseOldCommandLine
		os.Args = parseOldArgs
		os.Stdout = parseOldStdout
	})

	main()

	if parseErr := parseWritePipe.Close(); parseErr != nil {
		parseT.Fatalf("stdout pipe close error = %v", parseErr)
	}
	parseOutput, parseErr := io.ReadAll(parseReadPipe)
	if parseErr != nil {
		parseT.Fatalf("ReadAll(stdout) error = %v", parseErr)
	}
	if !strings.Contains(string(parseOutput), "exported 3 html files") {
		parseT.Fatalf("main() output = %q, want exported summary", string(parseOutput))
	}
	assertFileContains(parseT, filepath.Join(parseOutputDir, "index.html"), "Prerender several routes once")
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
