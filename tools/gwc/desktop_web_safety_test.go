package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDesktopWebChildEnvIsTargetScoped preserves ordinary web settings while neutralizing desktop tags.
func TestDesktopWebChildEnvIsTargetScoped(parseTest *testing.T) {
	parseTest.Setenv("GOFLAGS", "-tags=gwc_desktop")
	for _, isDesktopWeb := range []bool{false, true} {
		parseExpected := "GOFLAGS=-tags=gwc_desktop"
		if isDesktopWeb {
			parseExpected = "GOFLAGS=-mod=mod"
		}
		parseCount := 0
		for _, parseEntry := range buildDevChildEnv(devConfig{isDesktopWeb: isDesktopWeb}) {
			parseKey, _, _ := strings.Cut(parseEntry, "=")
			if strings.EqualFold(parseKey, "GOFLAGS") {
				parseCount++
				if parseEntry != parseExpected {
					parseTest.Fatalf("wrong target environment: %s", parseEntry)
				}
			}
		}
		if parseCount != 1 {
			parseTest.Fatalf("GOFLAGS count = %d", parseCount)
		}
	}
}

// TestDesktopWebRejectsIgnoredOverrides keeps native metadata dispatch explicit and read-only on invalid input.
func TestDesktopWebRejectsIgnoredOverrides(parseTest *testing.T) {
	parseRoot := filepath.Join(parseTest.TempDir(), "web-overrides")
	parseLauncher := launcher{repoRoot: mustDesktopRepoRoot(parseTest)}
	if _, parseErr := parseLauncher.desktopInit(desktopConfig{root: parseRoot}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	for _, parseFlag := range []string{"app", "main", "out", "output", "profile"} {
		if parseErr := parseLauncher.runBuild([]string{"-target", "web", "-root", parseRoot, "-" + parseFlag, "unsupported"}); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported") {
			parseTest.Fatalf("build ignored -%s: %v", parseFlag, parseErr)
		}
	}
	for _, parseFlag := range []string{"app", "main", "html", "index", "wasm", "output"} {
		if parseErr := parseLauncher.runDev([]string{"-target", "web", "-root", parseRoot, "-dry-run", "-" + parseFlag, "unsupported"}); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported") {
			parseTest.Fatalf("dev ignored -%s: %v", parseFlag, parseErr)
		}
	}
	if _, parseErr := os.Stat(filepath.Join(parseRoot, "assets", "web")); !os.IsNotExist(parseErr) {
		parseTest.Fatalf("invalid options caused web build writes: %v", parseErr)
	}
}

// TestDesktopWebBuildRejectsSymlinkedOutput checks the boundary before the helper can overwrite another file.
func TestDesktopWebBuildRejectsSymlinkedOutput(parseTest *testing.T) {
	parseRoot := filepath.Join(parseTest.TempDir(), "web-symlink")
	parseLauncher := launcher{repoRoot: mustDesktopRepoRoot(parseTest)}
	if _, parseErr := parseLauncher.desktopInit(desktopConfig{root: parseRoot}); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseWeb := filepath.Join(parseRoot, "assets", "web")
	if parseErr := os.MkdirAll(parseWeb, 0o755); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	parseOutside := filepath.Join(parseTest.TempDir(), "preserve.txt")
	if parseErr := os.WriteFile(parseOutside, []byte("preserve"), 0o644); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	if parseErr := os.Symlink(parseOutside, filepath.Join(parseWeb, "bootstrap.js")); parseErr != nil {
		parseTest.Skipf("symlink creation unavailable: %v", parseErr)
	}
	if _, parseErr := parseLauncher.desktopWebBuild(desktopConfig{root: parseRoot}); parseErr == nil || !strings.Contains(parseErr.Error(), "symlink") {
		parseTest.Fatalf("web helper accepted symlinked destination: %v", parseErr)
	}
	if parseData, parseErr := os.ReadFile(parseOutside); parseErr != nil || string(parseData) != "preserve" {
		parseTest.Fatalf("external target changed: %q, %v", parseData, parseErr)
	}
}
