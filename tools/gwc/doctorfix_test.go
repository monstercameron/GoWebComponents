package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestGenerateMissingScaffoldMetadata proves `gwc doctor --fix` generates a minimal
// gwc-start.json for a hand-built app (one with a main.go but no metadata), and never overwrites
// existing metadata.
func TestGenerateMissingScaffoldMetadata(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "go.mod", "module example.com/app\n\ngo 1.26.0\n")
	mustWrite(parseT, parseDir, "main.go", "package main\n\nfunc main() {}\n")

	parseGenerated, parseErr := generateMissingScaffoldMetadata(parseDir)
	if parseErr != nil {
		parseT.Fatalf("generateMissingScaffoldMetadata: %v", parseErr)
	}
	if parseGenerated != "gwc-start.json" {
		parseT.Fatalf("expected gwc-start.json generated, got %q", parseGenerated)
	}
	parseMeta, parsePresent, parseErr := loadScaffoldMetadata(parseDir)
	if parseErr != nil || !parsePresent {
		parseT.Fatalf("generated metadata should now load: present=%v err=%v", parsePresent, parseErr)
	}
	if parseMeta.ModulePath != "example.com/app" {
		parseT.Fatalf("expected module example.com/app captured, got %q", parseMeta.ModulePath)
	}

	// Idempotent: a second run leaves the existing file untouched and generates nothing.
	parseBefore, _ := os.ReadFile(filepath.Join(parseDir, "gwc-start.json"))
	parseAgain, parseErr := generateMissingScaffoldMetadata(parseDir)
	if parseErr != nil {
		parseT.Fatalf("second run errored: %v", parseErr)
	}
	if parseAgain != "" {
		parseT.Fatalf("existing metadata must not be regenerated, got %q", parseAgain)
	}
	parseAfter, _ := os.ReadFile(filepath.Join(parseDir, "gwc-start.json"))
	if string(parseBefore) != string(parseAfter) {
		parseT.Fatal("existing gwc-start.json must not be modified")
	}
}

// TestGenerateScaffoldMetadataSkipsNonApp proves no metadata is generated for a directory with
// no detectable app entrypoint (nothing to anchor it to).
func TestGenerateScaffoldMetadataSkipsNonApp(parseT *testing.T) {
	parseDir := parseT.TempDir()
	mustWrite(parseT, parseDir, "go.mod", "module example.com/lib\n\ngo 1.26.0\n")
	mustWrite(parseT, parseDir, "lib.go", "package lib\n")

	parseGenerated, parseErr := generateMissingScaffoldMetadata(parseDir)
	if parseErr != nil {
		parseT.Fatalf("generateMissingScaffoldMetadata: %v", parseErr)
	}
	if parseGenerated != "" {
		parseT.Fatalf("a library dir with no entrypoint should generate nothing, got %q", parseGenerated)
	}
	if _, parseErr := os.Stat(filepath.Join(parseDir, "gwc-start.json")); !os.IsNotExist(parseErr) {
		parseT.Fatal("no gwc-start.json should have been written")
	}
}
