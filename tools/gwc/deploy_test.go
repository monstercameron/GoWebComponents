package main

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseDeployConfigRejectsUnknownAdapter verifies adapter validation.
func TestParseDeployConfigRejectsUnknownAdapter(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseManifestPath := filepath.Join(parseRoot, "manifest.json")
	parseManifest := `{"package":"example.com/app","profile":"release","goos":"js","goarch":"wasm","artifacts":{"wasm":{"path":"app.wasm","bytes":4,"sha256":"abc"}}}`
	if parseWriteErr := os.WriteFile(parseManifestPath, []byte(parseManifest), 0644); parseWriteErr != nil {
		parseT.Fatalf("write manifest fixture: %v", parseWriteErr)
	}
	_, parseErr := parseDeployConfig(deployConfig{
		rootPath:     parseRoot,
		manifestPath: parseManifestPath,
		adapterName:  "custom",
		targetPath:   filepath.Join(parseRoot, "deploy"),
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "unknown deploy adapter") {
		parseT.Fatalf("expected unknown adapter error, got %v", parseErr)
	}
}

// TestRunDeployFilesystemCopiesArtifacts verifies filesystem adapter packaging behavior.
func TestRunDeployFilesystemCopiesArtifacts(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseArtifactDir := filepath.Join(parseRoot, "bin", "wasm-release")
	if parseWriteErr := os.MkdirAll(parseArtifactDir, 0755); parseWriteErr != nil {
		parseT.Fatalf("mkdir artifact dir: %v", parseWriteErr)
	}
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseArtifactDir, "app.wasm"), []byte("wasm"), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write wasm artifact: %v", parseWriteErr2)
	}
	parseManifest := map[string]any{
		"package": "example.com/app",
		"profile": "release",
		"goos":    "js",
		"goarch":  "wasm",
		"artifacts": map[string]any{
			"wasm": map[string]any{
				"path":   "app.wasm",
				"bytes":  4,
				"sha256": "abc",
			},
		},
	}
	parseManifestBytes, parseEncodeErr := json.MarshalIndent(parseManifest, "", "  ")
	if parseEncodeErr != nil {
		parseT.Fatalf("encode manifest fixture: %v", parseEncodeErr)
	}
	parseManifestBytes = append(parseManifestBytes, '\n')
	parseManifestPath := filepath.Join(parseArtifactDir, "wasm-release-manifest.json")
	if parseWriteErr3 := os.WriteFile(parseManifestPath, parseManifestBytes, 0644); parseWriteErr3 != nil {
		parseT.Fatalf("write release manifest: %v", parseWriteErr3)
	}
	parseTargetPath := filepath.Join(parseRoot, "deploy-target")

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		parseT.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if parseRunErr := (launcher{}).runDeploy([]string{
		"-root", parseRoot,
		"-manifest", parseManifestPath,
		"-adapter", "filesystem",
		"-target", parseTargetPath,
		"-json",
	}); parseRunErr != nil {
		parseT.Fatalf("run deploy: %v", parseRunErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		parseT.Fatalf("read deploy output: %v", parseOutputErr)
	}
	var parseSummary deploySummary
	if parseDecodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); parseDecodeErr != nil {
		parseT.Fatalf("decode deploy summary: %v\n%s", parseDecodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.Adapter != "filesystem" {
		parseT.Fatalf("unexpected deploy summary: %#v", parseSummary)
	}
	if !fileExists(filepath.Join(parseTargetPath, "app.wasm")) {
		parseT.Fatalf("expected copied deploy artifact under %s", parseTargetPath)
	}
	if !fileExists(filepath.Join(parseTargetPath, "wasm-release-manifest.json")) {
		parseT.Fatalf("expected copied deploy manifest under %s", parseTargetPath)
	}
}

// TestRunDeployZipCreatesArchive verifies zip adapter packaging behavior.
func TestRunDeployZipCreatesArchive(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseArtifactDir := filepath.Join(parseRoot, "bin", "wasm-release")
	if parseWriteErr := os.MkdirAll(parseArtifactDir, 0755); parseWriteErr != nil {
		parseT.Fatalf("mkdir artifact dir: %v", parseWriteErr)
	}
	if parseWriteErr2 := os.WriteFile(filepath.Join(parseArtifactDir, "app.wasm"), []byte("wasm"), 0644); parseWriteErr2 != nil {
		parseT.Fatalf("write wasm artifact: %v", parseWriteErr2)
	}
	parseManifest := `{
  "package": "example.com/app",
  "profile": "release",
  "goos": "js",
  "goarch": "wasm",
  "artifacts": {
    "wasm": {
      "path": "app.wasm",
      "bytes": 4,
      "sha256": "abc"
    }
  }
}`
	parseManifestPath := filepath.Join(parseArtifactDir, "wasm-release-manifest.json")
	if parseWriteErr3 := os.WriteFile(parseManifestPath, []byte(parseManifest), 0644); parseWriteErr3 != nil {
		parseT.Fatalf("write release manifest: %v", parseWriteErr3)
	}
	parseTargetPath := filepath.Join(parseRoot, "deploy-package.zip")
	if parseRunErr := (launcher{}).runDeploy([]string{
		"-root", parseRoot,
		"-manifest", parseManifestPath,
		"-adapter", "zip",
		"-target", parseTargetPath,
	}); parseRunErr != nil {
		parseT.Fatalf("run deploy zip: %v", parseRunErr)
	}
	if !fileExists(parseTargetPath) {
		parseT.Fatalf("expected deploy zip to exist: %s", parseTargetPath)
	}
	parseZip, parseErr := zip.OpenReader(parseTargetPath)
	if parseErr != nil {
		parseT.Fatalf("open deploy zip: %v", parseErr)
	}
	defer parseZip.Close()
	parseEntries := map[string]struct{}{}
	for _, parseFile := range parseZip.File {
		parseEntries[parseFile.Name] = struct{}{}
	}
	for _, parseExpected := range []string{"app.wasm", "wasm-release-manifest.json"} {
		if _, parseExists := parseEntries[parseExpected]; !parseExists {
			parseT.Fatalf("expected zip entry %s, entries=%v", parseExpected, parseEntries)
		}
	}
}
