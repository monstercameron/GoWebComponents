package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDesktopInitCopiesAllowlistedContributorTemplate(parseT *testing.T) {
	parseRoot := filepath.Join(parseT.TempDir(), "fresh-desktop")
	parseLauncher := launcher{repoRoot: mustDesktopRepoRoot(parseT)}
	parseSummary, parseErr := parseLauncher.desktopInit(desktopConfig{action: "init", root: parseRoot})
	if parseErr != nil {
		parseT.Fatalf("desktop init: %v", parseErr)
	}
	if parseSummary.Template == "" || len(parseSummary.Files) == 0 {
		parseT.Fatalf("desktop init summary missing template/files: %#v", parseSummary)
	}
	if _, parseErr := os.Stat(filepath.Join(parseRoot, "gwc-start.json")); parseErr != nil {
		parseT.Fatalf("desktop metadata missing: %v", parseErr)
	}
	if _, parseErr := os.Stat(filepath.Join(parseRoot, "assets", "dist")); !os.IsNotExist(parseErr) {
		parseT.Fatalf("generated dist unexpectedly copied, err=%v", parseErr)
	}
	if _, parseErr := os.Stat(filepath.Join(parseRoot, "bin")); !os.IsNotExist(parseErr) {
		parseT.Fatalf("generated bin unexpectedly copied, err=%v", parseErr)
	}
	parseGoMod, parseErr := os.ReadFile(filepath.Join(parseRoot, "go.mod"))
	if parseErr != nil || !strings.Contains(string(parseGoMod), "replace github.com/monstercameron/GoWebComponents/v6 => "+strconv.Quote(filepath.ToSlash(parseLauncher.repoRoot))) {
		parseT.Fatalf("fresh module replacement missing: err=%v\n%s", parseErr, parseGoMod)
	}
}

func TestDesktopInitRefusesExistingRoot(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if _, parseErr := (launcher{repoRoot: mustDesktopRepoRoot(parseT)}).desktopInit(desktopConfig{action: "init", root: parseRoot}); parseErr == nil {
		parseT.Fatal("desktop init accepted existing root")
	}
}

func TestDesktopBuildRejectsNonCanonicalTarget(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMetadata := `{"schemaVersion":1,"desktop":{"version":1,"nativeEntry":"cmd/desktop/main.go","frontendEntry":"frontend/main.go","assetsDir":"assets","outputPath":"bin/custom.exe","wailsVersion":"v3.0.0-beta.17"}}`
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(parseMetadata), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if _, parseErr := (launcher{}).desktopBuild(desktopConfig{action: "build", root: parseRoot}); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected non-canonical target rejection, got %v", parseErr)
	}
}

func TestDesktopJSONPreservesActionErrors(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseMetadata := `{"schemaVersion":1,"desktop":{"version":1,"nativeEntry":"cmd/desktop/main.go","frontendEntry":"frontend/main.go","assetsDir":"assets","outputPath":"bin/custom.exe","wailsVersion":"v3.0.0-beta.17"}}`
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(parseMetadata), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseOutput, parseErr := captureLauncherStdout(func() error {
		return (launcher{}).runDesktopCommand([]string{"build", "-root", parseRoot, "-json"})
	})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected JSON desktop action error, got %v", parseErr)
	}
	if !strings.Contains(parseOutput, `"ok":false`) {
		parseT.Fatalf("expected failed JSON summary, got %q", parseOutput)
	}
}

func TestDesktopCommandRegistryAndHelpExposeDesktop(t *testing.T) {
	var parseFoundRegistry bool
	for _, parseCommand := range listLauncherCommandRegistry() {
		if parseCommand.Name == "desktop" {
			parseFoundRegistry = true
			if !parseCommand.JSON || !parseCommand.Mutating {
				t.Fatalf("desktop registry metadata inaccurate: %#v", parseCommand)
			}
		}
	}
	if !parseFoundRegistry {
		t.Fatal("desktop missing from launcher registry")
	}
	var parseFoundHelp bool
	for _, parseCommand := range listGwcHelpCommands() {
		if parseCommand.Name == "desktop" {
			parseFoundHelp = true
		}
	}
	if !parseFoundHelp {
		t.Fatal("desktop missing from help registry")
	}
}

func mustDesktopRepoRoot(parseT *testing.T) string {
	parseRoot, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	return filepath.Clean(filepath.Join(parseRoot, "..", ".."))
}
