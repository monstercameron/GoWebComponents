package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStaticDirectoriesUsesNearestExistingPaths(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "examples", "100-ai-chat-wizard", "bin", "client"), 0o755); err != nil {
		t.Fatalf("MkdirAll client bin: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "examples", "static"), 0o755); err != nil {
		t.Fatalf("MkdirAll static: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir tempDir: %v", err)
	}
	defer os.Chdir(originalWD)

	clientDir, sharedDir := resolveStaticDirectories()
	if filepath.ToSlash(clientDir) != "examples/100-ai-chat-wizard/bin/client" {
		t.Fatalf("unexpected client dir: %q", clientDir)
	}
	if filepath.ToSlash(sharedDir) != "examples/static" {
		t.Fatalf("unexpected shared dir: %q", sharedDir)
	}
}
