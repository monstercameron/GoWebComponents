package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStaticDirectoriesUsesNearestExistingPaths(parseT *testing.T) {
	parseOriginalWD, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("Getwd: %v", parseErr)
	}
	parseTempDir := parseT.TempDir()
	if parseErr2 := os.MkdirAll(filepath.Join(parseTempDir, "examples", "server", "ai-chat-wizard", "bin", "client"), 0o755); parseErr2 != nil {
		parseT.Fatalf("MkdirAll client bin: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(filepath.Join(parseTempDir, "examples", "static"), 0o755); parseErr3 != nil {
		parseT.Fatalf("MkdirAll static: %v", parseErr3)
	}
	if parseErr4 := os.Chdir(parseTempDir); parseErr4 != nil {
		parseT.Fatalf("Chdir tempDir: %v", parseErr4)
	}
	defer func() {
		if parseErr5 := os.Chdir(parseOriginalWD); parseErr5 != nil {
			parseT.Fatalf("restore cwd: %v", parseErr5)
		}
	}()

	parseClientDir, parseSharedDir := parseResolveStaticDirectories()
	if filepath.ToSlash(parseClientDir) != "examples/server/ai-chat-wizard/bin/client" {
		parseT.Fatalf("unexpected client dir: %q", parseClientDir)
	}
	if filepath.ToSlash(parseSharedDir) != "examples/static" {
		parseT.Fatalf("unexpected shared dir: %q", parseSharedDir)
	}
}
