package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestResolveTailwindConfigUsesThirdPartyCache verifies default Tailwind cache placement under third_party.
func TestResolveTailwindConfigUsesThirdPartyCache(t *testing.T) {
	rootPath := t.TempDir()
	staticDirPath := filepath.Join(rootPath, "examples", "static")
	if err := os.MkdirAll(staticDirPath, 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDirPath, "tailwind.input.css"), []byte("@import \"tailwindcss\";\n"), 0o644); err != nil {
		t.Fatalf("write tailwind input: %v", err)
	}

	resolvedConfig, err := resolveTailwindConfig(tailwindConfig{rootPath: rootPath})
	if err != nil {
		t.Fatalf("resolve tailwind config: %v", err)
	}
	if resolvedConfig.cacheDirPath != filepath.Join(rootPath, "third_party", "tailwindcss", "bin") {
		t.Fatalf("expected third_party cache path, got %q", resolvedConfig.cacheDirPath)
	}
	if resolvedConfig.outputFilePath != filepath.Join(rootPath, "examples", "static", "css", "tailwind.css") {
		t.Fatalf("unexpected output file path: %q", resolvedConfig.outputFilePath)
	}
	if resolvedConfig.versionTag != tailwindDefaultVersion {
		t.Fatalf("expected default version %q, got %q", tailwindDefaultVersion, resolvedConfig.versionTag)
	}
}

// TestBuildTailwindManifestFileScansExampleSources verifies source-literal scanning writes generated manifest rows.
func TestBuildTailwindManifestFileScansExampleSources(t *testing.T) {
	rootPath := t.TempDir()
	staticDirPath := filepath.Join(rootPath, "examples", "static")
	sourceDirPath := filepath.Join(rootPath, "examples", "01-counter")
	if err := os.MkdirAll(staticDirPath, 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.MkdirAll(sourceDirPath, 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDirPath, "tailwind.input.css"), []byte("@import \"tailwindcss\";\n"), 0o644); err != nil {
		t.Fatalf("write tailwind input: %v", err)
	}
	sourceText := "package main\n\nfunc buildView() string {\n\treturn \"bg-red-500 text-white\"\n}\n"
	if err := os.WriteFile(filepath.Join(sourceDirPath, "main.go"), []byte(sourceText), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	resolvedConfig, err := resolveTailwindConfig(tailwindConfig{rootPath: rootPath})
	if err != nil {
		t.Fatalf("resolve tailwind config: %v", err)
	}
	manifestFilePath, err := buildTailwindManifestFile(resolvedConfig)
	if err != nil {
		t.Fatalf("build tailwind manifest: %v", err)
	}
	manifestBytes, err := os.ReadFile(manifestFilePath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifestText := string(manifestBytes)
	if !strings.Contains(manifestText, "<!-- 01-counter/main.go -->") {
		t.Fatalf("expected source comment row in manifest, got:\n%s", manifestText)
	}
	if !strings.Contains(manifestText, `<div class="bg-red-500 text-white"></div>`) {
		t.Fatalf("expected class row in manifest, got:\n%s", manifestText)
	}
}

// TestRunTailwindUsesExplicitBinary verifies gwc tailwind can run with a provided CLI binary path.
func TestRunTailwindUsesExplicitBinary(t *testing.T) {
	rootPath := t.TempDir()
	staticDirPath := filepath.Join(rootPath, "examples", "static")
	if err := os.MkdirAll(staticDirPath, 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	inputFilePath := filepath.Join(staticDirPath, "tailwind.input.css")
	if err := os.WriteFile(inputFilePath, []byte("@import \"tailwindcss\";\n"), 0o644); err != nil {
		t.Fatalf("write tailwind input: %v", err)
	}
	binaryFilePath := buildTailwindStubBinary(t, rootPath)

	if err := (launcher{}).runTailwind([]string{"-root", rootPath, "-binary", binaryFilePath, "-skip-manifest"}); err != nil {
		t.Fatalf("run tailwind: %v", err)
	}

	outputFilePath := filepath.Join(staticDirPath, "css", "tailwind.css")
	outputBytes, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if strings.TrimSpace(string(outputBytes)) != strings.TrimSpace("@import \"tailwindcss\";") {
		t.Fatalf("expected output to mirror input fixture, got %q", string(outputBytes))
	}
}

// TestRunTailwindJSONOutput verifies JSON mode emits a machine-readable tailwind summary.
func TestRunTailwindJSONOutput(t *testing.T) {
	rootPath := t.TempDir()
	staticDirPath := filepath.Join(rootPath, "examples", "static")
	if err := os.MkdirAll(staticDirPath, 0o755); err != nil {
		t.Fatalf("mkdir static dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(staticDirPath, "tailwind.input.css"), []byte("@import \"tailwindcss\";\n"), 0o644); err != nil {
		t.Fatalf("write tailwind input: %v", err)
	}
	binaryFilePath := buildTailwindStubBinary(t, rootPath)

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runTailwind([]string{"-root", rootPath, "-binary", binaryFilePath, "-skip-manifest", "-json"}); err != nil {
		t.Fatalf("run tailwind json: %v", err)
	}

	outputText, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary tailwindSummary
	if err := json.Unmarshal([]byte(outputText), &summary); err != nil {
		t.Fatalf("unmarshal tailwind summary: %v\n%s", err, outputText)
	}
	if !summary.OK {
		t.Fatalf("expected successful tailwind summary, got %#v", summary)
	}
	if summary.RootPath != rootPath {
		t.Fatalf("expected root path %q, got %#v", rootPath, summary)
	}
	if summary.ManifestFilePath != "" {
		t.Fatalf("expected skipped manifest path to be empty, got %#v", summary)
	}
}

// buildTailwindStubBinary creates a tiny cross-platform fake Tailwind executable for command tests.
func buildTailwindStubBinary(t *testing.T, rootPath string) string {
	t.Helper()
	stubDirPath := filepath.Join(rootPath, "tailwind-stub")
	if err := os.MkdirAll(stubDirPath, 0o755); err != nil {
		t.Fatalf("mkdir stub dir: %v", err)
	}

	if runtime.GOOS == "windows" {
		stubFilePath := filepath.Join(stubDirPath, "tailwind.cmd")
		stubScript := "@echo off\r\n" +
			"set in=\r\n" +
			"set out=\r\n" +
			":loop\r\n" +
			"if \"%1\"==\"\" goto done\r\n" +
			"if \"%1\"==\"-i\" (\r\n" +
			"  set in=%2\r\n" +
			"  shift\r\n" +
			")\r\n" +
			"if \"%1\"==\"-o\" (\r\n" +
			"  set out=%2\r\n" +
			"  shift\r\n" +
			")\r\n" +
			"shift\r\n" +
			"goto loop\r\n" +
			":done\r\n" +
			"copy /Y \"%in%\" \"%out%\" >NUL\r\n" +
			"echo stub tailwind build complete\r\n"
		if err := os.WriteFile(stubFilePath, []byte(stubScript), 0o755); err != nil {
			t.Fatalf("write windows stub: %v", err)
		}
		return stubFilePath
	}

	stubFilePath := filepath.Join(stubDirPath, "tailwind")
	stubScript := "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"IN=\"\"\n" +
		"OUT=\"\"\n" +
		"while [[ $# -gt 0 ]]; do\n" +
		"  case \"$1\" in\n" +
		"    -i) IN=\"$2\"; shift 2 ;;\n" +
		"    -o) OUT=\"$2\"; shift 2 ;;\n" +
		"    *) shift ;;\n" +
		"  esac\n" +
		"done\n" +
		"cp \"$IN\" \"$OUT\"\n" +
		"echo \"stub tailwind build complete\"\n"
	if err := os.WriteFile(stubFilePath, []byte(stubScript), 0o755); err != nil {
		t.Fatalf("write unix stub: %v", err)
	}
	return stubFilePath
}
