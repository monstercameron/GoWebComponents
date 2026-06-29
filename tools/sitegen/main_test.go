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
	for range 6 {
		if _, parseStatErr := os.Stat(filepath.Join(parseDir, "go.mod")); parseStatErr == nil {
			return parseDir
		}
		parseDir = filepath.Dir(parseDir)
	}
	parseT.Fatal("module root not found")
	return ""
}

// TestGenerateSiteBuildsWasmAndShell pins the site-build contract: the docs
// app compiles to wasm and the generated boot shell inlines the Go runtime
// loader and boots site.wasm.
func TestGenerateSiteBuildsWasmAndShell(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("site wasm build is slow; skipped in -short mode")
	}
	parseRepoRoot := repoRootForTest(parseT)
	parseOutDir := parseT.TempDir()

	if parseErr := generateSite(parseRepoRoot, parseOutDir); parseErr != nil {
		parseT.Fatalf("generateSite: %v", parseErr)
	}

	parseWasmInfo, parseErr := os.Stat(filepath.Join(parseOutDir, "site.wasm"))
	if parseErr != nil {
		parseT.Fatalf("site.wasm missing: %v", parseErr)
	}
	if parseWasmInfo.Size() < 1_000_000 {
		parseT.Fatalf("site.wasm suspiciously small: %d bytes", parseWasmInfo.Size())
	}

	parseShellRaw, parseErr2 := os.ReadFile(filepath.Join(parseOutDir, "index.html"))
	if parseErr2 != nil {
		parseT.Fatalf("boot shell missing: %v", parseErr2)
	}
	parseShell := string(parseShellRaw)
	for _, parseNeedle := range []string{"site.wasm", "new Go()", "WebAssembly.instantiate", "crypto.subtle.digest", `id="app"`} {
		if !strings.Contains(parseShell, parseNeedle) {
			parseT.Fatalf("boot shell missing %q", parseNeedle)
		}
	}
}
