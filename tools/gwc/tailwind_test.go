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
func TestResolveTailwindConfigUsesThirdPartyCache(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseStaticDirPath := filepath.Join(parseRootPath, "examples", "static")
	if parseErr := os.MkdirAll(parseStaticDirPath, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseStaticDirPath, "tailwind.input.css"), []byte("@import \"tailwindcss\";\n"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write tailwind input: %v", parseErr2)
	}

	parseResolvedConfig, parseErr3 := resolveTailwindConfig(tailwindConfig{rootPath: parseRootPath})
	if parseErr3 != nil {
		parseT.Fatalf("resolve tailwind config: %v", parseErr3)
	}
	if parseResolvedConfig.cacheDirPath != filepath.Join(parseRootPath, "third_party", "tailwindcss", "bin") {
		parseT.Fatalf("expected third_party cache path, got %q", parseResolvedConfig.cacheDirPath)
	}
	if parseResolvedConfig.outputFilePath != filepath.Join(parseRootPath, "examples", "static", "css", "tailwind.css") {
		parseT.Fatalf("unexpected output file path: %q", parseResolvedConfig.outputFilePath)
	}
	if parseResolvedConfig.versionTag != tailwindDefaultVersion {
		parseT.Fatalf("expected default version %q, got %q", tailwindDefaultVersion, parseResolvedConfig.versionTag)
	}
}

// TestBuildTailwindManifestFileScansExampleSources verifies source-literal scanning writes generated manifest rows.
func TestBuildTailwindManifestFileScansExampleSources(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseStaticDirPath := filepath.Join(parseRootPath, "examples", "static")
	parseSourceDirPath := filepath.Join(parseRootPath, "examples", "01-counter")
	if parseErr := os.MkdirAll(parseStaticDirPath, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(parseSourceDirPath, 0o755); parseErr2 != nil {
		parseT.Fatalf("mkdir source dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseStaticDirPath, "tailwind.input.css"), []byte("@import \"tailwindcss\";\n"), 0o644); parseErr3 != nil {
		parseT.Fatalf("write tailwind input: %v", parseErr3)
	}
	parseSourceText := "package main\n\nfunc buildView() string {\n\treturn \"bg-red-500 text-white\"\n}\n"
	if parseErr4 := os.WriteFile(filepath.Join(parseSourceDirPath, "main.go"), []byte(parseSourceText), 0o644); parseErr4 != nil {
		parseT.Fatalf("write source file: %v", parseErr4)
	}

	parseResolvedConfig, parseErr5 := resolveTailwindConfig(tailwindConfig{rootPath: parseRootPath})
	if parseErr5 != nil {
		parseT.Fatalf("resolve tailwind config: %v", parseErr5)
	}
	parseManifestFilePath, parseErr5 := buildTailwindManifestFile(parseResolvedConfig)
	if parseErr5 != nil {
		parseT.Fatalf("build tailwind manifest: %v", parseErr5)
	}
	parseManifestBytes, parseErr5 := os.ReadFile(parseManifestFilePath)
	if parseErr5 != nil {
		parseT.Fatalf("read manifest: %v", parseErr5)
	}
	parseManifestText := string(parseManifestBytes)
	if !strings.Contains(parseManifestText, "<!-- 01-counter/main.go -->") {
		parseT.Fatalf("expected source comment row in manifest, got:\n%s", parseManifestText)
	}
	if !strings.Contains(parseManifestText, `<div class="bg-red-500 text-white"></div>`) {
		parseT.Fatalf("expected class row in manifest, got:\n%s", parseManifestText)
	}
}

// TestRunTailwindUsesExplicitBinary verifies gwc tailwind can run with a provided CLI binary path.
func TestRunTailwindUsesExplicitBinary(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseStaticDirPath := filepath.Join(parseRootPath, "examples", "static")
	if parseErr := os.MkdirAll(parseStaticDirPath, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr)
	}
	parseInputFilePath := filepath.Join(parseStaticDirPath, "tailwind.input.css")
	if parseErr2 := os.WriteFile(parseInputFilePath, []byte("@import \"tailwindcss\";\n"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write tailwind input: %v", parseErr2)
	}
	parseBinaryFilePath := buildTailwindStubBinary(parseT, parseRootPath)

	if parseErr3 := (launcher{}).runTailwind([]string{"-root", parseRootPath, "-binary", parseBinaryFilePath, "-skip-manifest"}); parseErr3 != nil {
		parseT.Fatalf("run tailwind: %v", parseErr3)
	}

	parseOutputFilePath := filepath.Join(parseStaticDirPath, "css", "tailwind.css")
	parseOutputBytes, parseErr4 := os.ReadFile(parseOutputFilePath)
	if parseErr4 != nil {
		parseT.Fatalf("read output file: %v", parseErr4)
	}
	if strings.TrimSpace(string(parseOutputBytes)) != strings.TrimSpace("@import \"tailwindcss\";") {
		parseT.Fatalf("expected output to mirror input fixture, got %q", string(parseOutputBytes))
	}
}

// TestRunTailwindJSONOutput verifies JSON mode emits a machine-readable tailwind summary.
func TestRunTailwindJSONOutput(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseStaticDirPath := filepath.Join(parseRootPath, "examples", "static")
	if parseErr := os.MkdirAll(parseStaticDirPath, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir static dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseStaticDirPath, "tailwind.input.css"), []byte("@import \"tailwindcss\";\n"), 0o644); parseErr2 != nil {
		parseT.Fatalf("write tailwind input: %v", parseErr2)
	}
	parseBinaryFilePath := buildTailwindStubBinary(parseT, parseRootPath)

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).runTailwind([]string{"-root", parseRootPath, "-binary", parseBinaryFilePath, "-skip-manifest", "-json"}); parseErr4 != nil {
		parseT.Fatalf("run tailwind json: %v", parseErr4)
	}

	parseOutputText, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	var parseSummary tailwindSummary
	if parseErr5 := json.Unmarshal([]byte(parseOutputText), &parseSummary); parseErr5 != nil {
		parseT.Fatalf("unmarshal tailwind summary: %v\n%s", parseErr5, parseOutputText)
	}
	if !parseSummary.OK {
		parseT.Fatalf("expected successful tailwind summary, got %#v", parseSummary)
	}
	if parseSummary.RootPath != parseRootPath {
		parseT.Fatalf("expected root path %q, got %#v", parseRootPath, parseSummary)
	}
	if parseSummary.ManifestFilePath != "" {
		parseT.Fatalf("expected skipped manifest path to be empty, got %#v", parseSummary)
	}
}

// buildTailwindStubBinary creates a tiny cross-platform fake Tailwind executable for command tests.
func buildTailwindStubBinary(parseT *testing.T, parseRootPath string) string {
	parseT.Helper()
	parseStubDirPath := filepath.Join(parseRootPath, "tailwind-stub")
	if parseErr := os.MkdirAll(parseStubDirPath, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir stub dir: %v", parseErr)
	}

	if runtime.GOOS == "windows" {
		parseStubFilePath := filepath.Join(parseStubDirPath, "tailwind.cmd")
		parseStubScript := "@echo off\r\n" +
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
		if parseErr2 := os.WriteFile(parseStubFilePath, []byte(parseStubScript), 0o755); parseErr2 != nil {
			parseT.Fatalf("write windows stub: %v", parseErr2)
		}
		return parseStubFilePath
	}

	parseStubFilePath2 := filepath.Join(parseStubDirPath, "tailwind")
	parseStubScript2 := "#!/usr/bin/env bash\n" +
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
	if parseErr3 := os.WriteFile(parseStubFilePath2, []byte(parseStubScript2), 0o755); parseErr3 != nil {
		parseT.Fatalf("write unix stub: %v", parseErr3)
	}
	return parseStubFilePath2
}
