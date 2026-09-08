package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// getDesktopSafetyFixture creates a canonical target and installs a controlled process boundary.
func getDesktopSafetyFixture(parseT *testing.T) (launcher, desktopConfig) {
	parseT.Helper()
	parseRoot := parseT.TempDir()
	parseMetadata := `{"schemaVersion":1,"desktop":{"version":1,"nativeEntry":"cmd/desktop/main.go","frontendEntry":"frontend/main.go","assetsDir":"assets","outputPath":"bin/wails-counter.exe","wailsVersion":"v3.0.0-beta.17"}}`
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(parseMetadata), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseL := launcher{repoRoot: mustDesktopRepoRoot(parseT)}
	parsePrevious := desktopRunProcess
	parseT.Cleanup(func() { desktopRunProcess = parsePrevious })
	desktopRunProcess = func(parseContext context.Context, parseName string, parseArgs []string, parseDir string, parseEnv []string) (string, error) {
		if parseName == "git" {
			if parseArgs[len(parseArgs)-1] == "HEAD" {
				return desktopWailsRevision, nil
			}
			return "", nil
		}
		if parseName == "go" {
			switch parseArgs[0] {
			case "version":
				return "go version go1.26.3 windows/amd64", nil
			case "list":
				parseJSON, _ := json.Marshal(map[string]any{"Version": desktopWailsVersion, "Replace": map[string]string{"Dir": filepath.Join(parseL.repoRoot, "third_party", "wails", "v3")}})
				return string(parseJSON), nil
			case "run":
				if parseErr := os.MkdirAll(filepath.Join(parseRoot, "bin"), 0o755); parseErr != nil {
					return "", parseErr
				}
				return "built", os.WriteFile(filepath.Join(parseRoot, "bin", "wails-counter.exe"), []byte("verified executable bytes"), 0o644)
			}
		}
		parseJSON, _ := json.Marshal(map[string]any{"ok": true, "checks": desktopRequiredSmokeChecks, "error": ""})
		return string(parseJSON), nil
	}
	return parseL, desktopConfig{root: parseRoot}
}

// TestDesktopPackageManifestMatchesArchive verifies success metadata and both final artifact hashes.
func TestDesktopPackageManifestMatchesArchive(parseT *testing.T) {
	parseL, parseConfig := getDesktopSafetyFixture(parseT)
	parseSummary, parseErr := parseL.desktopPackage(parseConfig)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseManifestBytes, parseErr := os.ReadFile(filepath.Join(parseConfig.root, parseSummary.Manifest))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	var parseManifest desktopSummary
	if parseErr = json.Unmarshal(parseManifestBytes, &parseManifest); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if !parseManifest.OK || parseManifest.Signed || parseManifest.ToolVersions["wailsRevision"] != desktopWailsRevision {
		parseT.Fatalf("manifest=%+v", parseManifest)
	}
	parseArchive, parseErr := zip.OpenReader(filepath.Join(parseConfig.root, parseSummary.Artifact))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	defer parseArchive.Close()
	if len(parseArchive.File) != 1 {
		parseT.Fatalf("archive entries=%d", len(parseArchive.File))
	}
	parseEntry, parseErr := parseArchive.File[0].Open()
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseBytes, parseErr := io.ReadAll(parseEntry)
	_ = parseEntry.Close()
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseHash := sha256.Sum256(parseBytes)
	if parseManifest.Hashes[parseArchive.File[0].Name] != hex.EncodeToString(parseHash[:]) {
		parseT.Fatal("executable hash disagrees with archive bytes")
	}
	parseBytes, parseErr = os.ReadFile(filepath.Join(parseConfig.root, parseSummary.Artifact))
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseHash = sha256.Sum256(parseBytes)
	if parseManifest.Hashes[filepath.Base(parseSummary.Artifact)] != hex.EncodeToString(parseHash[:]) {
		parseT.Fatal("archive hash disagrees with manifest")
	}
}

// TestDesktopPackageRefusesExistingManifestBeforeBuild preserves existing outputs before any subprocess mutation.
func TestDesktopPackageRefusesExistingManifestBeforeBuild(parseT *testing.T) {
	parseL, parseConfig := getDesktopSafetyFixture(parseT)
	parsePath := filepath.Join(parseConfig.root, "bin", "wails-counter.manifest.json")
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseErr := os.WriteFile(parsePath, []byte("keep me"), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	desktopRunProcess = func(context.Context, string, []string, string, []string) (string, error) {
		parseT.Error("process started before no-overwrite preflight")
		return "", nil
	}
	if _, parseErr := parseL.desktopPackage(parseConfig); parseErr == nil {
		parseT.Fatal("existing manifest accepted")
	}
	if parseBytes, _ := os.ReadFile(parsePath); string(parseBytes) != "keep me" {
		parseT.Fatal("manifest changed")
	}
}

// TestDesktopCopyDoesNotOverwrite confirms exclusive copying preserves an existing target.
func TestDesktopCopyDoesNotOverwrite(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseSource, parseTarget := filepath.Join(parseDir, "source"), filepath.Join(parseDir, "target")
	if parseErr := os.WriteFile(parseSource, []byte("new"), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseErr := os.WriteFile(parseTarget, []byte("old"), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseErr := desktopCopyRegularFile(parseSource, parseTarget); parseErr == nil {
		parseT.Fatal("existing copy target overwritten")
	}
	if parseBytes, _ := os.ReadFile(parseTarget); string(parseBytes) != "old" {
		parseT.Fatal("old bytes lost")
	}
}

// TestDesktopPinRejectsDifferentRevision ensures successful git execution alone is insufficient.
func TestDesktopPinRejectsDifferentRevision(parseT *testing.T) {
	parseL, parseConfig := getDesktopSafetyFixture(parseT)
	desktopRunProcess = func(context.Context, string, []string, string, []string) (string, error) {
		return strings.Repeat("a", 40), nil
	}
	if parseErr := parseL.desktopVerifyPin(parseConfig); parseErr == nil || !strings.Contains(parseErr.Error(), "does not match pin") {
		parseT.Fatalf("pin=%v", parseErr)
	}
}

// TestDesktopBuildPreservesFailureDiagnostics keeps compiler output in the surfaced error.
func TestDesktopBuildPreservesFailureDiagnostics(parseT *testing.T) {
	parseL, parseConfig := getDesktopSafetyFixture(parseT)
	parseBase := desktopRunProcess
	desktopRunProcess = func(parseContext context.Context, parseName string, parseArgs []string, parseDir string, parseEnv []string) (string, error) {
		if parseName == "go" && parseArgs[0] == "run" {
			return "frontend compile detail", errors.New("build failed")
		}
		return parseBase(parseContext, parseName, parseArgs, parseDir, parseEnv)
	}
	if _, parseErr := parseL.desktopBuild(parseConfig); parseErr == nil || !strings.Contains(parseErr.Error(), "frontend compile detail") {
		parseT.Fatalf("build error=%v", parseErr)
	}
}

// TestDesktopOutputRejectsSymlinkRootsAndDescendants verifies generated writes remain inside the intended directory.
func TestDesktopOutputRejectsSymlinkRootsAndDescendants(parseT *testing.T) {
	parseRoot, parseExternal := parseT.TempDir(), parseT.TempDir()
	parseLink := filepath.Join(parseRoot, "linked")
	if parseErr := os.Symlink(parseExternal, parseLink); parseErr != nil {
		parseT.Skipf("symlink privilege unavailable: %v", parseErr)
	}
	if parseErr := desktopRejectSymlinkPath(parseLink, filepath.Join(parseLink, "artifact")); parseErr == nil {
		parseT.Fatal("linked root accepted")
	}
	if parseErr := desktopRejectOutputTree(parseRoot, parseRoot); parseErr == nil {
		parseT.Fatal("nested output link accepted")
	}
}

// TestDesktopSmokeRequiresCompleteOutcome distinguishes expected fault evidence from incidental logs.
func TestDesktopSmokeRequiresCompleteOutcome(parseT *testing.T) {
	for _, parseOutput := range []string{"ordinary output", `{"ok":true,"checks":["dom"],"error":""}`, `{"ok":false,"checks":[],"error":"host crashed"}`} {
		if parseErr := validateDesktopSmokeOutput(parseOutput, ""); parseErr == nil {
			parseT.Fatalf("incomplete success accepted: %s", parseOutput)
		}
	}
	parseFailure := `{"ok":false,"checks":["visible-boot-error","native-controls-unavailable"],"error":"missing module"}`
	if parseErr := validateDesktopSmokeOutput(parseFailure, "missing-binding"); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseErr := validateDesktopSmokeOutput(parseFailure, "streaming"); parseErr == nil {
		parseT.Fatal("streaming failure accepted")
	}
}

// TestDesktopProcessHonorsDeadline verifies an actual child is stopped and reaped on cancellation.
func TestDesktopProcessHonorsDeadline(parseT *testing.T) {
	parseExe, parseErr := os.Executable()
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseContext, parseCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer parseCancel()
	parseStarted := time.Now()
	_, parseErr = runDesktopProcess(parseContext, parseExe, []string{"-test.run=^TestDesktopProcessHelper$"}, "", append(os.Environ(), "GWC_DESKTOP_PROCESS_HELPER=1"))
	if !errors.Is(parseErr, context.DeadlineExceeded) {
		parseT.Fatalf("deadline=%v", parseErr)
	}
	if time.Since(parseStarted) > 5*time.Second {
		parseT.Fatal("child cancellation exceeded bound")
	}
}

// TestDesktopProcessHelper blocks only in the dedicated subprocess deadline test.
func TestDesktopProcessHelper(parseT *testing.T) {
	if os.Getenv("GWC_DESKTOP_PROCESS_HELPER") != "1" {
		return
	}
	fmt.Fprintln(os.Stderr, "deadline helper started")
	time.Sleep(10 * time.Second)
	os.Exit(0)
}
