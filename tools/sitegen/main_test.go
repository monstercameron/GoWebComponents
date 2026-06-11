package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRootForTest walks up from the working directory to the module root.
func repoRootForTest(parseT *testing.T) string {
	parseT.Helper()
	parseDir, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("getwd: %v", parseErr)
	}
	for parseDepth := 0; parseDepth < 6; parseDepth++ {
		if _, parseStatErr := os.Stat(filepath.Join(parseDir, "go.mod")); parseStatErr == nil {
			return parseDir
		}
		parseDir = filepath.Dir(parseDir)
	}
	parseT.Fatal("module root not found")
	return ""
}

// TestGenerateSiteProducesCorePages pins the static-site contract: the
// generator succeeds against the real repo content and emits the landing,
// learn, gallery, and search assets with the expected anchors.
func TestGenerateSiteProducesCorePages(parseT *testing.T) {
	parseRepoRoot := repoRootForTest(parseT)
	parseOutDir := parseT.TempDir()

	if parseErr := generateSite(parseRepoRoot, parseOutDir); parseErr != nil {
		parseT.Fatalf("generateSite: %v", parseErr)
	}

	parseChecks := map[string]string{
		"index.html":                          "Go. In the",
		"learn/index.html":                    "The manual",
		"learn/01-getting-started/index.html": "Getting Started",
		"examples/index.html":                 "live examples",
		"api/index.html":                      "API reference",
		"site/site.css":                       "--accent",
		"site/site.js":                        "search-index.json",
		"site/search-index.json":              "Getting Started",
	}
	for parseFile, parseNeedle := range parseChecks {
		parseRaw, parseErr := os.ReadFile(filepath.Join(parseOutDir, filepath.FromSlash(parseFile)))
		if parseErr != nil {
			parseT.Fatalf("expected output %s: %v", parseFile, parseErr)
		}
		if !strings.Contains(string(parseRaw), parseNeedle) {
			parseT.Fatalf("output %s missing %q", parseFile, parseNeedle)
		}
	}

	// Chapter tables must render as real tables, not flattened paragraphs.
	parseChapter, parseErr := os.ReadFile(filepath.Join(parseOutDir, "learn", "01-getting-started", "index.html"))
	if parseErr != nil {
		parseT.Fatalf("read chapter: %v", parseErr)
	}
	if !strings.Contains(string(parseChapter), "<table>") {
		parseT.Fatal("chapter page lost its markdown tables")
	}
}
