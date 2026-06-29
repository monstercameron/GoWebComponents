package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestTailwindResolveBinaryHelpers verifies binary path and asset-name resolution.
func TestTailwindResolveBinaryHelpers(parseT *testing.T) {
	parseExplicitPath, parseErr := resolveTailwindBinaryPath(tailwindConfig{binaryFilePath: "C:/bin/tailwind"})
	if parseErr != nil {
		parseT.Fatalf("resolveTailwindBinaryPath explicit: %v", parseErr)
	}
	if parseExplicitPath != "C:/bin/tailwind" {
		parseT.Fatalf("explicit binary path = %q, want C:/bin/tailwind", parseExplicitPath)
	}

	parseAssetName, parseErr := resolveTailwindAssetName(runtime.GOOS, runtime.GOARCH)
	if parseErr != nil {
		parseT.Fatalf("resolveTailwindAssetName current platform: %v", parseErr)
	}
	parseResolvedPath, parseErr := resolveTailwindBinaryPath(tailwindConfig{
		cacheDirPath: "C:/cache",
		versionTag:   "v-test",
	})
	if parseErr != nil {
		parseT.Fatalf("resolveTailwindBinaryPath default: %v", parseErr)
	}
	if parseResolvedPath != filepath.Join("C:/cache", "v-test", parseAssetName) {
		parseT.Fatalf("default binary path = %q, want %q", parseResolvedPath, filepath.Join("C:/cache", "v-test", parseAssetName))
	}
	if _, parseErr2 := resolveTailwindAssetName("plan9", "386"); parseErr2 == nil {
		parseT.Fatal("expected unsupported platform error")
	}
}

// TestTailwindDownloadAndChecksumHelpers verifies download, checksum, and cache installation helpers.
func TestTailwindDownloadAndChecksumHelpers(parseT *testing.T) {
	parseAssetName, parseErr := resolveTailwindAssetName(runtime.GOOS, runtime.GOARCH)
	if parseErr != nil {
		parseT.Fatalf("resolveTailwindAssetName: %v", parseErr)
	}
	parseAssetBytes := []byte("tailwind-binary")
	parseChecksum := sha256.Sum256(parseAssetBytes)
	parseChecksumText := hex.EncodeToString(parseChecksum[:])

	parseOriginalFetch := tailwindFetchURLBytes
	parseT.Cleanup(func() {
		tailwindFetchURLBytes = parseOriginalFetch
	})
	parseFetchCount := 0
	tailwindFetchURLBytes = func(parseDownloadURL string) ([]byte, error) {
		parseFetchCount++
		switch {
		case strings.HasSuffix(parseDownloadURL, "/sha256sums.txt"):
			return fmt.Appendf(nil, "%s *%s\n", parseChecksumText, parseAssetName), nil
		case strings.HasSuffix(parseDownloadURL, "/"+parseAssetName):
			return parseAssetBytes, nil
		default:
			return nil, fmt.Errorf("unexpected download url %q", parseDownloadURL)
		}
	}

	parseBinaryPath := filepath.Join(parseT.TempDir(), "tailwind", parseAssetName)
	parseConfig := tailwindConfig{
		versionTag:          "v-test",
		shouldForceDownload: true,
	}
	if parseErr2 := ensureTailwindBinaryFile(parseConfig, parseBinaryPath); parseErr2 != nil {
		parseT.Fatalf("ensureTailwindBinaryFile download: %v", parseErr2)
	}
	parseInstalledBytes, parseErr := os.ReadFile(parseBinaryPath)
	if parseErr != nil {
		parseT.Fatalf("ReadFile downloaded binary: %v", parseErr)
	}
	if string(parseInstalledBytes) != string(parseAssetBytes) {
		parseT.Fatalf("installed binary bytes = %q, want %q", string(parseInstalledBytes), string(parseAssetBytes))
	}
	if parseFetchCount != 2 {
		parseT.Fatalf("expected checksum and asset fetches, got %d", parseFetchCount)
	}

	parseFetchCount = 0
	parseConfig.shouldForceDownload = false
	if parseErr2 := ensureTailwindBinaryFile(parseConfig, parseBinaryPath); parseErr2 != nil {
		parseT.Fatalf("ensureTailwindBinaryFile existing binary: %v", parseErr2)
	}
	if parseFetchCount != 0 {
		parseT.Fatalf("expected existing binary to skip downloads, got %d fetches", parseFetchCount)
	}
}

// TestTailwindFetchAndChecksumBranches verifies fetch and checksum branch behavior.
func TestTailwindFetchAndChecksumBranches(parseT *testing.T) {
	parseServer := httptest.NewServer(http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		switch parseR.URL.Path {
		case "/ok":
			_, _ = parseW.Write([]byte("payload"))
		case "/empty":
			parseW.WriteHeader(http.StatusOK)
		default:
			http.Error(parseW, "boom", http.StatusBadGateway)
		}
	}))
	defer parseServer.Close()

	parseBytes, parseErr := fetchTailwindURLBytes(parseServer.URL + "/ok")
	if parseErr != nil {
		parseT.Fatalf("fetchTailwindURLBytes ok: %v", parseErr)
	}
	if string(parseBytes) != "payload" {
		parseT.Fatalf("fetchTailwindURLBytes payload = %q, want payload", string(parseBytes))
	}
	if _, parseErr2 := fetchTailwindURLBytes(parseServer.URL + "/bad"); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "unexpected status") {
		parseT.Fatalf("expected status error, got %v", parseErr2)
	}
	if _, parseErr3 := fetchTailwindURLBytes(parseServer.URL + "/empty"); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "empty response") {
		parseT.Fatalf("expected empty response error, got %v", parseErr3)
	}

	parseChecksumMap := parseTailwindChecksums("abc123 *tailwindcss-linux-x64\n# ignored\n\nff00 tailwindcss-macos-arm64\n")
	if parseChecksumMap["tailwindcss-linux-x64"] != "abc123" || parseChecksumMap["tailwindcss-macos-arm64"] != "ff00" {
		parseT.Fatalf("unexpected checksum map: %#v", parseChecksumMap)
	}

	parsePayload := []byte("payload")
	parsePayloadChecksum := sha256.Sum256(parsePayload)
	if parseErr4 := verifyTailwindChecksum(parsePayload, hex.EncodeToString(parsePayloadChecksum[:])); parseErr4 != nil {
		parseT.Fatalf("verifyTailwindChecksum ok: %v", parseErr4)
	}
	if parseErr5 := verifyTailwindChecksum(parsePayload, "deadbeef"); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "checksum mismatch") {
		parseT.Fatalf("expected checksum mismatch, got %v", parseErr5)
	}
}

// TestExecuteTailwindCommandSurfacesCLIOutput verifies command failure output propagation.
func TestExecuteTailwindCommandSurfacesCLIOutput(parseT *testing.T) {
	parseWorkDir := parseT.TempDir()
	parseInputPath := filepath.Join(parseWorkDir, "tailwind.input.css")
	parseOutputPath := filepath.Join(parseWorkDir, "tailwind.css")
	if parseErr := os.WriteFile(parseInputPath, []byte("@import \"tailwindcss\";"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile input: %v", parseErr)
	}

	parseBinaryPath := filepath.Join(parseWorkDir, "tailwind-fail")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".cmd"
		if parseErr2 := os.WriteFile(parseBinaryPath, []byte("@echo off\r\necho broken tailwind\r\nexit /b 1\r\n"), 0o755); parseErr2 != nil {
			parseT.Fatalf("WriteFile windows stub: %v", parseErr2)
		}
	} else {
		if parseErr3 := os.WriteFile(parseBinaryPath, []byte("#!/usr/bin/env sh\necho broken tailwind\nexit 1\n"), 0o755); parseErr3 != nil {
			parseT.Fatalf("WriteFile unix stub: %v", parseErr3)
		}
	}

	if _, parseErr4 := executeTailwindCommand(parseBinaryPath, parseInputPath, parseOutputPath, parseWorkDir); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "broken tailwind") {
		parseT.Fatalf("expected tailwind command failure output, got %v", parseErr4)
	}
}
