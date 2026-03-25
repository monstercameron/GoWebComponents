package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveLegacyArtifactHandlesExistingMissingAndErrors(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "artifact.wasm")
	if err := os.WriteFile(filePath, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile(): %v", err)
	}
	if err := removeLegacyArtifact(filePath); err != nil {
		t.Fatalf("removeLegacyArtifact(existing): %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expected artifact to be removed, stat err=%v", err)
	}
	if err := removeLegacyArtifact(filePath); err != nil {
		t.Fatalf("removeLegacyArtifact(missing): %v", err)
	}

	blockedPath := filepath.Join(dir, "blocked")
	if err := os.MkdirAll(blockedPath, 0o755); err != nil {
		t.Fatalf("MkdirAll(): %v", err)
	}
	if err := os.WriteFile(filepath.Join(blockedPath, "child.txt"), []byte("child"), 0o644); err != nil {
		t.Fatalf("WriteFile(child): %v", err)
	}
	if err := removeLegacyArtifact(blockedPath); err == nil || !strings.Contains(err.Error(), "remove legacy artifact") {
		t.Fatalf("expected directory removal error, got %v", err)
	}
}

func TestFindRepoRootWalksUpwardAndFailsWhenMissing(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll(): %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod): %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd(): %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalWD) })

	if err := os.Chdir(nested); err != nil {
		t.Fatalf("Chdir(nested): %v", err)
	}
	resolved, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot(): %v", err)
	}
	if resolved != root {
		t.Fatalf("findRepoRoot() = %q, want %q", resolved, root)
	}

	other := t.TempDir()
	if err := os.Chdir(other); err != nil {
		t.Fatalf("Chdir(other): %v", err)
	}
	if _, err := findRepoRoot(); err == nil || !strings.Contains(err.Error(), "go.mod not found") {
		t.Fatalf("expected missing go.mod error, got %v", err)
	}
}

func TestWriteBrotliSidecarProducesCompressedArtifactAndErrorsOnMissingSource(t *testing.T) {
	dir := t.TempDir()
	sourcePath := filepath.Join(dir, "chat.wasm")
	targetPath := filepath.Join(dir, "chat.wasm.br")
	if err := os.WriteFile(sourcePath, []byte(strings.Repeat("hello wasm\n", 20)), 0o644); err != nil {
		t.Fatalf("WriteFile(source): %v", err)
	}

	if err := writeBrotliSidecar(sourcePath, targetPath); err != nil {
		t.Fatalf("writeBrotliSidecar(): %v", err)
	}
	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("Stat(target): %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("expected non-empty brotli sidecar")
	}

	missingSource := filepath.Join(dir, "missing.wasm")
	if err := writeBrotliSidecar(missingSource, filepath.Join(dir, "missing.wasm.br")); err == nil || !strings.Contains(err.Error(), "read source artifact for brotli") {
		t.Fatalf("expected missing source error, got %v", err)
	}
}

func TestBuildTargetUsesGoCommandAndSurfacesFailures(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "bin", "chat.wasm")
	shimDir := filepath.Join(dir, "shim")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(shimDir): %v", err)
	}

	successShim := filepath.Join(shimDir, "go.cmd")
	if err := os.WriteFile(successShim, []byte("@echo off\r\nset out=\r\n:loop\r\nif \"%1\"==\"\" goto done\r\nif \"%1\"==\"-o\" (\r\n  set out=%2\r\n  shift\r\n)\r\nshift\r\ngoto loop\r\n:done\r\necho wasm>%out%\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(success shim): %v", err)
	}

	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	if err := os.Setenv("PATH", shimDir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("Setenv(PATH): %v", err)
	}

	if err := buildTarget(dir, "chat client", "./client", outputPath); err != nil {
		t.Fatalf("buildTarget(success): %v", err)
	}
	if info, err := os.Stat(outputPath); err != nil || info.Size() == 0 {
		t.Fatalf("expected built artifact at %q, stat err=%v size=%v", outputPath, err, info)
	}

	if err := os.WriteFile(successShim, []byte("@echo off\r\nexit /b 9\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(failure shim): %v", err)
	}
	if err := buildTarget(dir, "chat client", "./client", filepath.Join(dir, "bin", "failed.wasm")); err == nil || !strings.Contains(err.Error(), "build chat client") {
		t.Fatalf("expected buildTarget failure, got %v", err)
	}
}
