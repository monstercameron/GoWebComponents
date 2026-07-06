// Command pack builds the distributable GoWebComponents DevTools extension archive from the
// extension source folder — a single, dependency-free `go run` step instead of the manual
// Load-unpacked dance (audit FB6). It produces dist/gwc-devtools-<version>.zip, which is the
// directly-loadable package: Chrome Web Store / Edge Add-ons take the zip as-is, and Firefox
// loads it via about:debugging → Load Temporary Add-on (a signed .xpi is `web-ext sign` of the
// same zip). Using archive/zip from the standard library keeps the framework's zero-npm posture
// intact — no node, no web-ext, no toolchain to install.
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// packagedFiles is the allow-list of extension source files that ship in the package. Docs,
// tests, and the packager itself are intentionally excluded so the artifact is exactly the
// runtime extension.
var packagedFiles = []string{
	"manifest.json",
	"devtools.html",
	"devtools.js",
	"panel.html",
	"panel.js",
	"bridge.js",
}

func main() {
	parseSrcDir := "tools/devtools-extension"
	if len(os.Args) > 1 {
		parseSrcDir = os.Args[1]
	}
	parseOut, parseErr := PackageExtension(parseSrcDir, "")
	if parseErr != nil {
		fmt.Fprintln(os.Stderr, "pack:", parseErr)
		os.Exit(1)
	}
	fmt.Println("packaged extension →", parseOut)
}

// PackageExtension zips the extension's runtime files from srcDir into outPath and returns the
// written path. When outPath is empty it defaults to <srcDir>/dist/gwc-devtools-<version>.zip,
// where <version> is read from manifest.json. It validates the manifest is well-formed JSON and
// that every packaged file exists, so a broken package fails loudly instead of shipping.
func PackageExtension(parseSrcDir string, parseOutPath string) (string, error) {
	parseVersion, parseErr := readManifestVersion(filepath.Join(parseSrcDir, "manifest.json"))
	if parseErr != nil {
		return "", parseErr
	}
	if !isPathSafeVersion(parseVersion) {
		// The version is concatenated into the output file path; a value with a
		// path separator or ".." would write the archive outside dist/.
		return "", fmt.Errorf("manifest.json version %q is not a safe path token", parseVersion)
	}
	if parseOutPath == "" {
		parseOutPath = filepath.Join(parseSrcDir, "dist", "gwc-devtools-"+parseVersion+".zip")
	}
	if parseErr := os.MkdirAll(filepath.Dir(parseOutPath), 0755); parseErr != nil {
		return "", fmt.Errorf("create dist dir: %w", parseErr)
	}

	// Write to a temp file and rename into place only on full success, so a
	// mid-package failure (a missing source file) cannot truncate/replace a
	// previously-good artifact at parseOutPath.
	parseTemp, parseErr := os.CreateTemp(filepath.Dir(parseOutPath), filepath.Base(parseOutPath)+".*.tmp")
	if parseErr != nil {
		return "", fmt.Errorf("create archive temp: %w", parseErr)
	}
	parseTempPath := parseTemp.Name()
	defer func() {
		_ = parseTemp.Close()
		_ = os.Remove(parseTempPath)
	}()

	parseZip := zip.NewWriter(parseTemp)
	for _, parseName := range packagedFiles {
		if parseErr := addFileToZip(parseZip, parseSrcDir, parseName); parseErr != nil {
			_ = parseZip.Close()
			return "", parseErr
		}
	}
	if parseErr := parseZip.Close(); parseErr != nil {
		return "", fmt.Errorf("finalize archive: %w", parseErr)
	}
	if parseErr := parseTemp.Close(); parseErr != nil {
		return "", fmt.Errorf("close archive temp: %w", parseErr)
	}
	if parseErr := os.Rename(parseTempPath, parseOutPath); parseErr != nil {
		return "", fmt.Errorf("finalize archive path: %w", parseErr)
	}
	return parseOutPath, nil
}

// isPathSafeVersion reports whether a manifest version is safe to embed in a
// file path: non-empty and free of path separators and ".." segments.
func isPathSafeVersion(parseVersion string) bool {
	if parseVersion == "" {
		return false
	}
	if strings.ContainsAny(parseVersion, `/\`) || strings.Contains(parseVersion, "..") {
		return false
	}
	return true
}

// readManifestVersion validates the manifest is JSON and returns its version field.
func readManifestVersion(parsePath string) (string, error) {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return "", fmt.Errorf("read manifest: %w", parseErr)
	}
	var parseManifest struct {
		Version string `json:"version"`
	}
	if parseErr := json.Unmarshal(parseData, &parseManifest); parseErr != nil {
		return "", fmt.Errorf("manifest.json is not valid JSON: %w", parseErr)
	}
	if parseManifest.Version == "" {
		return "", fmt.Errorf("manifest.json has no version")
	}
	return parseManifest.Version, nil
}

// addFileToZip copies one source file into the archive at its base name.
func addFileToZip(parseZip *zip.Writer, parseSrcDir string, parseName string) error {
	parseSource, parseErr := os.Open(filepath.Join(parseSrcDir, parseName))
	if parseErr != nil {
		return fmt.Errorf("package file %q: %w", parseName, parseErr)
	}
	defer parseSource.Close()

	parseEntry, parseErr := parseZip.Create(parseName)
	if parseErr != nil {
		return fmt.Errorf("zip entry %q: %w", parseName, parseErr)
	}
	if _, parseErr := io.Copy(parseEntry, parseSource); parseErr != nil {
		return fmt.Errorf("write %q into archive: %w", parseName, parseErr)
	}
	return nil
}
