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
func TestParseDeployConfigRejectsUnknownAdapter(t *testing.T) {
	parseRoot := t.TempDir()
	parseManifestPath := filepath.Join(parseRoot, "manifest.json")
	parseManifest := `{"package":"example.com/app","profile":"release","goos":"js","goarch":"wasm","artifacts":{"wasm":{"path":"app.wasm","bytes":4,"sha256":"abc"}}}`
	if writeErr := os.WriteFile(parseManifestPath, []byte(parseManifest), 0644); writeErr != nil {
		t.Fatalf("write manifest fixture: %v", writeErr)
	}
	_, parseErr := parseDeployConfig(deployConfig{
		rootPath:     parseRoot,
		manifestPath: parseManifestPath,
		adapterName:  "custom",
		targetPath:   filepath.Join(parseRoot, "deploy"),
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "unknown deploy adapter") {
		t.Fatalf("expected unknown adapter error, got %v", parseErr)
	}
}

// TestRunDeployFilesystemCopiesArtifacts verifies filesystem adapter packaging behavior.
func TestRunDeployFilesystemCopiesArtifacts(t *testing.T) {
	parseRoot := t.TempDir()
	parseArtifactDir := filepath.Join(parseRoot, "bin", "wasm-release")
	if writeErr := os.MkdirAll(parseArtifactDir, 0755); writeErr != nil {
		t.Fatalf("mkdir artifact dir: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseArtifactDir, "app.wasm"), []byte("wasm"), 0644); writeErr != nil {
		t.Fatalf("write wasm artifact: %v", writeErr)
	}
	parseManifest := map[string]interface{}{
		"package": "example.com/app",
		"profile": "release",
		"goos":    "js",
		"goarch":  "wasm",
		"artifacts": map[string]interface{}{
			"wasm": map[string]interface{}{
				"path":   "app.wasm",
				"bytes":  4,
				"sha256": "abc",
			},
		},
	}
	parseManifestBytes, parseEncodeErr := json.MarshalIndent(parseManifest, "", "  ")
	if parseEncodeErr != nil {
		t.Fatalf("encode manifest fixture: %v", parseEncodeErr)
	}
	parseManifestBytes = append(parseManifestBytes, '\n')
	parseManifestPath := filepath.Join(parseArtifactDir, "wasm-release-manifest.json")
	if writeErr := os.WriteFile(parseManifestPath, parseManifestBytes, 0644); writeErr != nil {
		t.Fatalf("write release manifest: %v", writeErr)
	}
	parseTargetPath := filepath.Join(parseRoot, "deploy-target")

	parseStdout, parseRestoreStdout, parseCaptureErr := captureExamplesStdout()
	if parseCaptureErr != nil {
		t.Fatalf("capture stdout: %v", parseCaptureErr)
	}
	defer parseRestoreStdout()

	if runErr := (launcher{}).runDeploy([]string{
		"-root", parseRoot,
		"-manifest", parseManifestPath,
		"-adapter", "filesystem",
		"-target", parseTargetPath,
		"-json",
	}); runErr != nil {
		t.Fatalf("run deploy: %v", runErr)
	}
	parseOutput, parseOutputErr := parseStdout()
	if parseOutputErr != nil {
		t.Fatalf("read deploy output: %v", parseOutputErr)
	}
	var parseSummary deploySummary
	if decodeErr := json.Unmarshal([]byte(parseOutput), &parseSummary); decodeErr != nil {
		t.Fatalf("decode deploy summary: %v\n%s", decodeErr, parseOutput)
	}
	if !parseSummary.OK || parseSummary.Adapter != "filesystem" {
		t.Fatalf("unexpected deploy summary: %#v", parseSummary)
	}
	if !fileExists(filepath.Join(parseTargetPath, "app.wasm")) {
		t.Fatalf("expected copied deploy artifact under %s", parseTargetPath)
	}
	if !fileExists(filepath.Join(parseTargetPath, "wasm-release-manifest.json")) {
		t.Fatalf("expected copied deploy manifest under %s", parseTargetPath)
	}
}

// TestRunDeployZipCreatesArchive verifies zip adapter packaging behavior.
func TestRunDeployZipCreatesArchive(t *testing.T) {
	parseRoot := t.TempDir()
	parseArtifactDir := filepath.Join(parseRoot, "bin", "wasm-release")
	if writeErr := os.MkdirAll(parseArtifactDir, 0755); writeErr != nil {
		t.Fatalf("mkdir artifact dir: %v", writeErr)
	}
	if writeErr := os.WriteFile(filepath.Join(parseArtifactDir, "app.wasm"), []byte("wasm"), 0644); writeErr != nil {
		t.Fatalf("write wasm artifact: %v", writeErr)
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
	if writeErr := os.WriteFile(parseManifestPath, []byte(parseManifest), 0644); writeErr != nil {
		t.Fatalf("write release manifest: %v", writeErr)
	}
	parseTargetPath := filepath.Join(parseRoot, "deploy-package.zip")
	if runErr := (launcher{}).runDeploy([]string{
		"-root", parseRoot,
		"-manifest", parseManifestPath,
		"-adapter", "zip",
		"-target", parseTargetPath,
	}); runErr != nil {
		t.Fatalf("run deploy zip: %v", runErr)
	}
	if !fileExists(parseTargetPath) {
		t.Fatalf("expected deploy zip to exist: %s", parseTargetPath)
	}
	parseZip, parseErr := zip.OpenReader(parseTargetPath)
	if parseErr != nil {
		t.Fatalf("open deploy zip: %v", parseErr)
	}
	defer parseZip.Close()
	parseEntries := map[string]struct{}{}
	for _, parseFile := range parseZip.File {
		parseEntries[parseFile.Name] = struct{}{}
	}
	for _, parseExpected := range []string{"app.wasm", "wasm-release-manifest.json"} {
		if _, parseExists := parseEntries[parseExpected]; !parseExists {
			t.Fatalf("expected zip entry %s, entries=%v", parseExpected, parseEntries)
		}
	}
}
