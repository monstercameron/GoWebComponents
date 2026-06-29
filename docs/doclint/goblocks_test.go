package doclint

import (
	"os"
	"path/filepath"
	"testing"
)

// TestValidateGoBlocks proves the doc-sample guard: a complete-file ```go sample that parses is
// accepted, a broken one is reported, and a fragment (no package clause) is skipped (high
// precision — only copy-paste-runnable samples are checked).
func TestValidateGoBlocks(parseT *testing.T) {
	parseRoot := parseT.TempDir()

	parseGood := "# Good\n\n```go\npackage main\n\nfunc main() { _ = 1 }\n```\n"
	parseBad := "# Bad\n\n```go\npackage main\n\nfunc main( {\n```\n"
	parseFragment := "# Fragment\n\n```go\ncount := ui.UseState(0)\n```\n"

	mustWriteDoc(parseT, parseRoot, "good.md", parseGood)
	mustWriteDoc(parseT, parseRoot, "fragment.md", parseFragment)

	// With only good + fragment docs, there must be zero errors (fragment is not validated).
	parseErrs, parseErr := ValidateGoBlocks(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("ValidateGoBlocks: %v", parseErr)
	}
	if len(parseErrs) != 0 {
		parseT.Fatalf("expected no errors for good+fragment docs, got %+v", parseErrs)
	}

	// Adding a broken complete-file sample must be reported.
	mustWriteDoc(parseT, parseRoot, "bad.md", parseBad)
	parseErrs, parseErr = ValidateGoBlocks(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("ValidateGoBlocks: %v", parseErr)
	}
	if len(parseErrs) != 1 || parseErrs[0].DocPath != "bad.md" {
		parseT.Fatalf("expected one error in bad.md, got %+v", parseErrs)
	}
}

// TestScanGoBlocksSkipsFragments proves only complete-file blocks are collected.
func TestScanGoBlocksSkipsFragments(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	mustWriteDoc(parseT, parseRoot, "mix.md",
		"```go\npackage app\n\nvar X = 1\n```\n\n```go\nX := 1\n_ = X\n```\n")
	parseBlocks, parseErr := ScanGoBlocks(parseRoot)
	if parseErr != nil {
		parseT.Fatalf("ScanGoBlocks: %v", parseErr)
	}
	if len(parseBlocks) != 1 {
		parseT.Fatalf("expected 1 complete-file block (fragment skipped), got %d", len(parseBlocks))
	}
}

func mustWriteDoc(parseT *testing.T, parseDir, parseName, parseContent string) {
	parseT.Helper()
	if parseErr := os.WriteFile(filepath.Join(parseDir, parseName), []byte(parseContent), 0o644); parseErr != nil {
		parseT.Fatalf("write %s: %v", parseName, parseErr)
	}
}
