package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestScanNPMArtifactsSkipsEditorTooling proves the npm scan ignores the VS Code extension's
// package.json (editor tooling, not framework/app supply chain), so adding the extension
// cannot break the zero-npm claim — while a package.json elsewhere is still flagged.
func TestScanNPMArtifactsSkipsEditorTooling(parseT *testing.T) {
	parseRoot := parseT.TempDir()

	parseExtDir := filepath.Join(parseRoot, "tools", "vscode-gwc")
	if parseErr := os.MkdirAll(parseExtDir, 0755); parseErr != nil {
		parseT.Fatalf("mkdir: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseExtDir, "package.json"), []byte(`{"name":"gwc"}`), 0644); parseErr != nil {
		parseT.Fatalf("write ext package.json: %v", parseErr)
	}

	if parseArtifacts := scanNPMArtifacts(parseRoot); len(parseArtifacts) != 0 {
		parseT.Fatalf("editor-tooling package.json must be excluded, got %v", parseArtifacts)
	}

	// A package.json at the app root is still a real npm artifact.
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "package.json"), []byte(`{"name":"app"}`), 0644); parseErr != nil {
		parseT.Fatalf("write root package.json: %v", parseErr)
	}
	if parseArtifacts := scanNPMArtifacts(parseRoot); len(parseArtifacts) != 1 {
		parseT.Fatalf("a root package.json must still be flagged, got %v", parseArtifacts)
	}
}
