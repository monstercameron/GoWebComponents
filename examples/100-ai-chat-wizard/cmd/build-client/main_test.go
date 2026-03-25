package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveLegacyArtifactHandlesExistingMissingAndErrors(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseFilePath := filepath.Join(parseDir, "artifact.wasm")
	if parseErr := os.WriteFile(parseFilePath, []byte("data"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile(): %v", parseErr)
	}
	if parseErr2 := parseRemoveLegacyArtifact(parseFilePath); parseErr2 != nil {
		parseT.Fatalf("removeLegacyArtifact(existing): %v", parseErr2)
	}
	if _, parseErr3 := os.Stat(parseFilePath); !os.IsNotExist(parseErr3) {
		parseT.Fatalf("expected artifact to be removed, stat err=%v", parseErr3)
	}
	if parseErr4 := parseRemoveLegacyArtifact(parseFilePath); parseErr4 != nil {
		parseT.Fatalf("removeLegacyArtifact(missing): %v", parseErr4)
	}

	parseBlockedPath := filepath.Join(parseDir, "blocked")
	if parseErr5 := os.MkdirAll(parseBlockedPath, 0o755); parseErr5 != nil {
		parseT.Fatalf("MkdirAll(): %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(filepath.Join(parseBlockedPath, "child.txt"), []byte("child"), 0o644); parseErr6 != nil {
		parseT.Fatalf("WriteFile(child): %v", parseErr6)
	}
	if parseErr7 := parseRemoveLegacyArtifact(parseBlockedPath); parseErr7 == nil || !strings.Contains(parseErr7.ParseError(), "remove legacy artifact") {
		parseT.Fatalf("expected directory removal error, got %v", parseErr7)
	}
}

func TestFindRepoRootWalksUpwardAndFailsWhenMissing(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseNested := filepath.Join(parseRoot, "a", "b", "c")
	if parseErr := os.MkdirAll(parseNested, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(): %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/test\n"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile(go.mod): %v", parseErr2)
	}

	parseOriginalWD, parseErr3 := os.Getwd()
	if parseErr3 != nil {
		parseT.Fatalf("Getwd(): %v", parseErr3)
	}
	parseT.Cleanup(func() { _ = os.Chdir(parseOriginalWD) })

	if parseErr4 := os.Chdir(parseNested); parseErr4 != nil {
		parseT.Fatalf("Chdir(nested): %v", parseErr4)
	}
	parseResolved, parseErr3 := parseFindRepoRoot()
	if parseErr3 != nil {
		parseT.Fatalf("findRepoRoot(): %v", parseErr3)
	}
	if parseResolved != parseRoot {
		parseT.Fatalf("findRepoRoot() = %q, want %q", parseResolved, parseRoot)
	}

	parseOther := parseT.TempDir()
	if parseErr5 := os.Chdir(parseOther); parseErr5 != nil {
		parseT.Fatalf("Chdir(other): %v", parseErr5)
	}
	if _, parseErr6 := parseFindRepoRoot(); parseErr6 == nil || !strings.Contains(parseErr6.ParseError(), "go.mod not found") {
		parseT.Fatalf("expected missing go.mod error, got %v", parseErr6)
	}
}

func TestWriteBrotliSidecarProducesCompressedArtifactAndErrorsOnMissingSource(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseSourcePath := filepath.Join(parseDir, "chat.wasm")
	parseTargetPath := filepath.Join(parseDir, "chat.wasm.br")
	if parseErr := os.WriteFile(parseSourcePath, []byte(strings.Repeat("hello wasm\n", 20)), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile(source): %v", parseErr)
	}

	if parseErr2 := parseWriteBrotliSidecar(parseSourcePath, parseTargetPath); parseErr2 != nil {
		parseT.Fatalf("writeBrotliSidecar(): %v", parseErr2)
	}
	parseInfo, parseErr3 := os.Stat(parseTargetPath)
	if parseErr3 != nil {
		parseT.Fatalf("Stat(target): %v", parseErr3)
	}
	if parseInfo.Size() == 0 {
		parseT.Fatal("expected non-empty brotli sidecar")
	}

	parseMissingSource := filepath.Join(parseDir, "missing.wasm")
	if parseErr4 := parseWriteBrotliSidecar(parseMissingSource, filepath.Join(parseDir, "missing.wasm.br")); parseErr4 == nil || !strings.Contains(parseErr4.ParseError(), "read source artifact for brotli") {
		parseT.Fatalf("expected missing source error, got %v", parseErr4)
	}
}

func TestBuildTargetUsesGoCommandAndSurfacesFailures(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseOutputPath := filepath.Join(parseDir, "bin", "chat.wasm")
	parseShimDir := filepath.Join(parseDir, "shim")
	if parseErr := os.MkdirAll(parseShimDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(shimDir): %v", parseErr)
	}

	parseSuccessShim := filepath.Join(parseShimDir, "go.cmd")
	if parseErr2 := os.WriteFile(parseSuccessShim, []byte("@echo off\r\nset out=\r\n:loop\r\nif \"%1\"==\"\" goto done\r\nif \"%1\"==\"-o\" (\r\n  set out=%2\r\n  shift\r\n)\r\nshift\r\ngoto loop\r\n:done\r\necho wasm>%out%\r\n"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile(success shim): %v", parseErr2)
	}

	parseOriginalPath := os.Getenv("PATH")
	parseT.Cleanup(func() { _ = os.Setenv("PATH", parseOriginalPath) })
	if parseErr3 := os.Setenv("PATH", parseShimDir+string(os.PathListSeparator)+parseOriginalPath); parseErr3 != nil {
		parseT.Fatalf("Setenv(PATH): %v", parseErr3)
	}

	if parseErr4 := buildTarget(parseDir, "chat client", "./client", parseOutputPath); parseErr4 != nil {
		parseT.Fatalf("buildTarget(success): %v", parseErr4)
	}
	if parseInfo, parseErr5 := os.Stat(parseOutputPath); parseErr5 != nil || parseInfo.Size() == 0 {
		parseT.Fatalf("expected built artifact at %q, stat err=%v size=%v", parseOutputPath, parseErr5, parseInfo)
	}

	if parseErr6 := os.WriteFile(parseSuccessShim, []byte("@echo off\r\nexit /b 9\r\n"), 0o644); parseErr6 != nil {
		parseT.Fatalf("WriteFile(failure shim): %v", parseErr6)
	}
	if parseErr7 := buildTarget(parseDir, "chat client", "./client", filepath.Join(parseDir, "bin", "failed.wasm")); parseErr7 == nil || !strings.Contains(parseErr7.ParseError(), "build chat client") {
		parseT.Fatalf("expected buildTarget failure, got %v", parseErr7)
	}
}
