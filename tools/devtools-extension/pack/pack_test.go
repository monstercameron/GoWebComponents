package main

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// TestPackageExtensionProducesLoadableZip proves the one-command packager builds a zip
// containing exactly the runtime extension files (and a valid manifest), so `go run
// ./tools/devtools-extension/pack` yields a directly-loadable artifact — no manual
// Load-unpacked dance, no npm/web-ext dependency.
func TestPackageExtensionProducesLoadableZip(parseT *testing.T) {
	parseSrc := parseT.TempDir()
	mustWriteExt := func(parseName string, parseBody string) {
		if parseErr := os.WriteFile(filepath.Join(parseSrc, parseName), []byte(parseBody), 0644); parseErr != nil {
			parseT.Fatalf("write %s: %v", parseName, parseErr)
		}
	}
	mustWriteExt("manifest.json", `{"manifest_version":3,"name":"x","version":"1.2.3"}`)
	for _, parseName := range []string{"devtools.html", "devtools.js", "panel.html", "panel.js", "bridge.js"} {
		mustWriteExt(parseName, "// "+parseName)
	}

	parseOut := filepath.Join(parseT.TempDir(), "ext.zip")
	parseGot, parseErr := PackageExtension(parseSrc, parseOut)
	if parseErr != nil {
		parseT.Fatalf("PackageExtension: %v", parseErr)
	}
	if parseGot != parseOut {
		parseT.Fatalf("expected output path %q, got %q", parseOut, parseGot)
	}

	parseReader, parseErr := zip.OpenReader(parseOut)
	if parseErr != nil {
		parseT.Fatalf("the package must be a valid zip: %v", parseErr)
	}
	defer parseReader.Close()
	parseInZip := map[string]bool{}
	for _, parseEntry := range parseReader.File {
		parseInZip[parseEntry.Name] = true
	}
	for _, parseWant := range packagedFiles {
		if !parseInZip[parseWant] {
			parseT.Fatalf("package is missing %q (have %v)", parseWant, parseInZip)
		}
	}
	if parseInZip["pack_test.go"] || parseInZip["README.md"] {
		parseT.Fatal("package must contain only runtime files, not docs/tests")
	}
}

// TestPackageExtensionDefaultsVersionedPath proves the default output path is derived from the
// manifest version.
func TestPackageExtensionDefaultsVersionedPath(parseT *testing.T) {
	parseSrc := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseSrc, "manifest.json"), []byte(`{"version":"9.9.9"}`), 0644); parseErr != nil {
		parseT.Fatalf("write manifest: %v", parseErr)
	}
	for _, parseName := range []string{"devtools.html", "devtools.js", "panel.html", "panel.js", "bridge.js"} {
		_ = os.WriteFile(filepath.Join(parseSrc, parseName), []byte("x"), 0644)
	}
	parseGot, parseErr := PackageExtension(parseSrc, "")
	if parseErr != nil {
		parseT.Fatalf("PackageExtension: %v", parseErr)
	}
	if filepath.Base(parseGot) != "gwc-devtools-9.9.9.zip" {
		parseT.Fatalf("expected version-stamped archive name, got %q", filepath.Base(parseGot))
	}
}

// TestPackageExtensionRejectsBadManifest proves a malformed manifest fails loudly rather than
// shipping a broken package.
func TestPackageExtensionRejectsBadManifest(parseT *testing.T) {
	parseSrc := parseT.TempDir()
	_ = os.WriteFile(filepath.Join(parseSrc, "manifest.json"), []byte(`{not json`), 0644)
	if _, parseErr := PackageExtension(parseSrc, filepath.Join(parseT.TempDir(), "o.zip")); parseErr == nil {
		parseT.Fatal("expected a malformed manifest to fail packaging")
	}
}
