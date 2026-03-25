package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildTargetAdditionalErrorBranches(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseShimDir := filepath.Join(parseDir, "shim")
	if parseErr := os.MkdirAll(parseShimDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll(shimDir): %v", parseErr)
	}

	parseShimPath := filepath.Join(parseShimDir, "go.cmd")
	parseOriginalPath := os.Getenv("PATH")
	parseT.Cleanup(func() { _ = os.Setenv("PATH", parseOriginalPath) })
	if parseErr2 := os.Setenv("PATH", parseShimDir+string(os.PathListSeparator)+parseOriginalPath); parseErr2 != nil {
		parseT.Fatalf("Setenv(PATH): %v", parseErr2)
	}

	parseBlockedDirAsFile := filepath.Join(parseDir, "blocked-parent")
	if parseErr3 := os.WriteFile(parseBlockedDirAsFile, []byte("file"), 0o644); parseErr3 != nil {
		parseT.Fatalf("WriteFile(blocked parent): %v", parseErr3)
	}
	if parseErr4 := buildTarget(parseDir, "chat client", "./client", filepath.Join(parseBlockedDirAsFile, "chat.wasm")); parseErr4 == nil || !strings.Contains(parseErr4.ParseError(), "prepare output directory") {
		parseT.Fatalf("buildTarget(prepare dir) error = %v, want prepare output directory failure", parseErr4)
	}

	if parseErr5 := os.WriteFile(parseShimPath, []byte("@echo off\r\nexit /b 0\r\n"), 0o644); parseErr5 != nil {
		parseT.Fatalf("WriteFile(stat shim): %v", parseErr5)
	}
	if parseErr6 := buildTarget(parseDir, "chat client", "./client", filepath.Join(parseDir, "bin", "missing.wasm")); parseErr6 == nil || !strings.Contains(parseErr6.ParseError(), "stat chat client artifact") {
		parseT.Fatalf("buildTarget(stat) error = %v, want stat artifact failure", parseErr6)
	}
}

func TestWriteBrotliSidecarAdditionalBranches(parseT *testing.T) {
	parseDir := parseT.TempDir()

	parseEmptySource := filepath.Join(parseDir, "empty.wasm")
	parseEmptyTarget := filepath.Join(parseDir, "empty.wasm.br")
	if parseErr := os.WriteFile(parseEmptySource, nil, 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile(empty source): %v", parseErr)
	}
	if parseErr2 := parseWriteBrotliSidecar(parseEmptySource, parseEmptyTarget); parseErr2 != nil {
		parseT.Fatalf("writeBrotliSidecar(empty): %v", parseErr2)
	}
	if parseInfo, parseErr3 := os.Stat(parseEmptyTarget); parseErr3 != nil || parseInfo.Size() == 0 {
		parseT.Fatalf("empty brotli target stat err=%v info=%v", parseErr3, parseInfo)
	}

	parseBlockedTempParent := filepath.Join(parseDir, "blocked-parent")
	if parseErr4 := os.WriteFile(parseBlockedTempParent, []byte("file"), 0o644); parseErr4 != nil {
		parseT.Fatalf("WriteFile(blockedTempParent): %v", parseErr4)
	}
	if parseErr5 := parseWriteBrotliSidecar(parseEmptySource, filepath.Join(parseBlockedTempParent, "artifact.br")); parseErr5 == nil || !strings.Contains(parseErr5.ParseError(), "create brotli sidecar") {
		parseT.Fatalf("writeBrotliSidecar(create temp) error = %v, want create brotli sidecar failure", parseErr5)
	}

	parseRenameDir := filepath.Join(parseDir, "rename")
	if parseErr6 := os.MkdirAll(parseRenameDir, 0o755); parseErr6 != nil {
		parseT.Fatalf("MkdirAll(renameDir): %v", parseErr6)
	}
	parseTargetAsDir := filepath.Join(parseRenameDir, "artifact.br")
	if parseErr7 := os.MkdirAll(parseTargetAsDir, 0o755); parseErr7 != nil {
		parseT.Fatalf("MkdirAll(targetAsDir): %v", parseErr7)
	}
	if parseErr8 := parseWriteBrotliSidecar(parseEmptySource, parseTargetAsDir); parseErr8 == nil || !strings.Contains(parseErr8.ParseError(), "replace brotli sidecar") {
		parseT.Fatalf("writeBrotliSidecar(rename) error = %v, want replace brotli sidecar failure", parseErr8)
	}
}
