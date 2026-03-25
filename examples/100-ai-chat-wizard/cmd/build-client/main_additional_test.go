package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildTargetAdditionalErrorBranches(t *testing.T) {
	dir := t.TempDir()
	shimDir := filepath.Join(dir, "shim")
	if err := os.MkdirAll(shimDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(shimDir): %v", err)
	}

	shimPath := filepath.Join(shimDir, "go.cmd")
	originalPath := os.Getenv("PATH")
	t.Cleanup(func() { _ = os.Setenv("PATH", originalPath) })
	if err := os.Setenv("PATH", shimDir+string(os.PathListSeparator)+originalPath); err != nil {
		t.Fatalf("Setenv(PATH): %v", err)
	}

	blockedDirAsFile := filepath.Join(dir, "blocked-parent")
	if err := os.WriteFile(blockedDirAsFile, []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile(blocked parent): %v", err)
	}
	if err := buildTarget(dir, "chat client", "./client", filepath.Join(blockedDirAsFile, "chat.wasm")); err == nil || !strings.Contains(err.Error(), "prepare output directory") {
		t.Fatalf("buildTarget(prepare dir) error = %v, want prepare output directory failure", err)
	}

	if err := os.WriteFile(shimPath, []byte("@echo off\r\nexit /b 0\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(stat shim): %v", err)
	}
	if err := buildTarget(dir, "chat client", "./client", filepath.Join(dir, "bin", "missing.wasm")); err == nil || !strings.Contains(err.Error(), "stat chat client artifact") {
		t.Fatalf("buildTarget(stat) error = %v, want stat artifact failure", err)
	}
}

func TestWriteBrotliSidecarAdditionalBranches(t *testing.T) {
	dir := t.TempDir()

	emptySource := filepath.Join(dir, "empty.wasm")
	emptyTarget := filepath.Join(dir, "empty.wasm.br")
	if err := os.WriteFile(emptySource, nil, 0o644); err != nil {
		t.Fatalf("WriteFile(empty source): %v", err)
	}
	if err := writeBrotliSidecar(emptySource, emptyTarget); err != nil {
		t.Fatalf("writeBrotliSidecar(empty): %v", err)
	}
	if info, err := os.Stat(emptyTarget); err != nil || info.Size() == 0 {
		t.Fatalf("empty brotli target stat err=%v info=%v", err, info)
	}

	blockedTempParent := filepath.Join(dir, "blocked-parent")
	if err := os.WriteFile(blockedTempParent, []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile(blockedTempParent): %v", err)
	}
	if err := writeBrotliSidecar(emptySource, filepath.Join(blockedTempParent, "artifact.br")); err == nil || !strings.Contains(err.Error(), "create brotli sidecar") {
		t.Fatalf("writeBrotliSidecar(create temp) error = %v, want create brotli sidecar failure", err)
	}

	renameDir := filepath.Join(dir, "rename")
	if err := os.MkdirAll(renameDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(renameDir): %v", err)
	}
	targetAsDir := filepath.Join(renameDir, "artifact.br")
	if err := os.MkdirAll(targetAsDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(targetAsDir): %v", err)
	}
	if err := writeBrotliSidecar(emptySource, targetAsDir); err == nil || !strings.Contains(err.Error(), "replace brotli sidecar") {
		t.Fatalf("writeBrotliSidecar(rename) error = %v, want replace brotli sidecar failure", err)
	}
}
