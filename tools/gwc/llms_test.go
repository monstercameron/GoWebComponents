package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeLLMSFixture builds a temp project with a README and two manual chapters
// (plus a manual README that must be excluded), returning the root.
func writeLLMSFixture(parseT *testing.T) string {
	parseT.Helper()
	parseRoot := parseT.TempDir()
	parseManual := filepath.Join(parseRoot, "docs", "REFERENCE_MANUAL")
	if parseErr := os.MkdirAll(parseManual, 0755); parseErr != nil {
		parseT.Fatalf("mkdir: %v", parseErr)
	}
	parseWrite := func(parsePath, parseContent string) {
		if parseErr := os.WriteFile(parsePath, []byte(parseContent), 0644); parseErr != nil {
			parseT.Fatalf("write %s: %v", parsePath, parseErr)
		}
	}
	parseWrite(filepath.Join(parseRoot, "README.md"),
		"# Acme Framework\n\n[![badge](x)](y)\n\nAcme is a delightful framework for builders.\n")
	parseWrite(filepath.Join(parseManual, "README.md"), "# Index\n\nshould be excluded\n")
	parseWrite(filepath.Join(parseManual, "02-routing.md"),
		"# 02 Routing\n\nUse this chapter for the router.\n\nMore detail about routing.\n")
	parseWrite(filepath.Join(parseManual, "01-getting-started.md"),
		"# 01 Getting Started\n\nThe shortest path to a working app.\n")
	return parseRoot
}

// TestLLMSIndexListsManualDocsInOrder proves the index has the project header and
// every manual chapter (excluding the manual README), ordered by filename.
func TestLLMSIndexListsManualDocsInOrder(parseT *testing.T) {
	parseRoot := writeLLMSFixture(parseT)
	parseIndex, _, parseErr := buildLLMS(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("buildLLMS: %v", parseErr)
	}
	for _, parseWant := range []string{
		"# Acme Framework",
		"> Acme is a delightful framework for builders.",
		"[01 Getting Started](docs/REFERENCE_MANUAL/01-getting-started.md): The shortest path to a working app.",
		"[02 Routing](docs/REFERENCE_MANUAL/02-routing.md): Use this chapter for the router.",
	} {
		if !strings.Contains(parseIndex, parseWant) {
			parseT.Fatalf("index missing %q:\n%s", parseWant, parseIndex)
		}
	}
	// Ordering: getting-started before routing.
	if strings.Index(parseIndex, "01-getting-started") > strings.Index(parseIndex, "02-routing") {
		parseT.Fatal("expected docs ordered by filename")
	}
	// The manual README must NOT be listed.
	if strings.Contains(parseIndex, "should be excluded") || strings.Contains(parseIndex, "REFERENCE_MANUAL/README.md") {
		parseT.Fatalf("manual README must be excluded:\n%s", parseIndex)
	}
}

// TestLLMSFullConcatenatesBodies proves llms-full.txt contains each chapter body
// with a source marker.
func TestLLMSFullConcatenatesBodies(parseT *testing.T) {
	parseRoot := writeLLMSFixture(parseT)
	_, parseFull, parseErr := buildLLMS(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("buildLLMS: %v", parseErr)
	}
	for _, parseWant := range []string{
		"<!-- source: docs/REFERENCE_MANUAL/01-getting-started.md -->",
		"The shortest path to a working app.",
		"<!-- source: docs/REFERENCE_MANUAL/02-routing.md -->",
		"More detail about routing.",
	} {
		if !strings.Contains(parseFull, parseWant) {
			parseT.Fatalf("full docs missing %q", parseWant)
		}
	}
}

// TestExtractTitleAndSummarySkipsBadgesAndMarkup proves the H1 + first prose line
// are picked, skipping badges/headings/lists.
func TestExtractTitleAndSummarySkipsBadgesAndMarkup(parseT *testing.T) {
	parseTitle, parseSummary := extractTitleAndSummary(
		"# Title X\n\n[![ci](a)](b)\n\n- a list item\n\nReal prose here.\n", "fallback.md")
	if parseTitle != "Title X" {
		parseT.Fatalf("title: got %q", parseTitle)
	}
	if parseSummary != "Real prose here." {
		parseT.Fatalf("summary: got %q", parseSummary)
	}
}

// TestLLMSStalenessDetection proves the CI gate: matching files are fresh; an edit
// makes them stale.
func TestLLMSStalenessDetection(parseT *testing.T) {
	parseRoot := writeLLMSFixture(parseT)
	parseIndex, parseFull, parseErr := buildLLMS(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("buildLLMS: %v", parseErr)
	}
	// Nothing written yet → both stale.
	if parseStale := llmsStaleFiles(parseRoot, parseIndex, parseFull); len(parseStale) != 2 {
		parseT.Fatalf("expected both files stale before writing, got %v", parseStale)
	}
	// Write them → fresh.
	_ = os.WriteFile(filepath.Join(parseRoot, llmsIndexFile), []byte(parseIndex), 0644)
	_ = os.WriteFile(filepath.Join(parseRoot, llmsFullFile), []byte(parseFull), 0644)
	if parseStale := llmsStaleFiles(parseRoot, parseIndex, parseFull); len(parseStale) != 0 {
		parseT.Fatalf("expected fresh after writing, got %v", parseStale)
	}
	// Tamper the index → stale again.
	_ = os.WriteFile(filepath.Join(parseRoot, llmsIndexFile), []byte("stale"), 0644)
	if parseStale := llmsStaleFiles(parseRoot, parseIndex, parseFull); len(parseStale) != 1 || parseStale[0] != llmsIndexFile {
		parseT.Fatalf("expected only %s stale, got %v", llmsIndexFile, parseStale)
	}
}

// TestLLMSDeterministic proves repeated generation is byte-identical (so the CI
// gate never flaps).
func TestLLMSDeterministic(parseT *testing.T) {
	parseRoot := writeLLMSFixture(parseT)
	parseIndex1, parseFull1, _ := buildLLMS(parseRoot)
	parseIndex2, parseFull2, _ := buildLLMS(parseRoot)
	if parseIndex1 != parseIndex2 || parseFull1 != parseFull2 {
		parseT.Fatal("expected deterministic llms output across runs")
	}
}
