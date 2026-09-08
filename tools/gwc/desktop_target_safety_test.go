package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestWebBuildOverridesInheritedDesktopTags compiles against both process and persisted GOFLAGS.
func TestWebBuildOverridesInheritedDesktopTags(parseTest *testing.T) {
	for _, parseMode := range []string{"process", "persisted"} {
		parseTest.Run(parseMode, func(parseTest *testing.T) {
			parseRoot := parseTest.TempDir()
			for parseName, parseSource := range map[string]string{
				"go.mod":     "module example.com/webgate\n\ngo 1.26\n",
				"main.go":    "package main\nfunc main() {}\n",
				"desktop.go": "//go:build gwc_desktop\n\npackage main\nvar _ = forbiddenDesktopFileWasCompiled\n",
			} {
				if parseErr := os.WriteFile(filepath.Join(parseRoot, parseName), []byte(parseSource), 0o644); parseErr != nil {
					parseTest.Fatal(parseErr)
				}
			}
			parseTest.Setenv("GOFLAGS", "-tags=gwc_desktop")
			if parseMode == "persisted" {
				parseEnvFile := filepath.Join(parseRoot, "goenv")
				if parseErr := os.WriteFile(parseEnvFile, []byte("GOFLAGS=-tags=gwc_desktop\n"), 0o644); parseErr != nil {
					parseTest.Fatal(parseErr)
				}
				parseTest.Setenv("GOENV", parseEnvFile)
				parseTest.Setenv("GOFLAGS", "")
			}
			parseName, parseArgs, parseEnv, parseErr := buildCommandForProfile(buildProfile{Toolchain: "go"}, filepath.Join(parseRoot, "app.wasm"))
			if parseErr != nil {
				parseTest.Fatal(parseErr)
			}
			parseControl := exec.Command("go", "build", "-o", filepath.Join(parseRoot, "control.wasm"), ".")
			parseControl.Dir, parseControl.Env = parseRoot, parseEnv
			if parseOutput, parseErr := parseControl.CombinedOutput(); parseErr == nil || !strings.Contains(string(parseOutput), "forbiddenDesktopFileWasCompiled") {
				parseTest.Fatalf("fixture did not reproduce inherited tag leakage: %v\n%s", parseErr, parseOutput)
			}
			parseCommand := exec.Command(parseName, parseArgs...)
			parseCommand.Dir, parseCommand.Env = parseRoot, parseEnv
			if parseOutput, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
				parseTest.Fatalf("inherited desktop tag leaked into web build: %v\n%s", parseErr, parseOutput)
			}
		})
	}
}

// TestDesktopDevDryRunNeverBuildsOrStarts verifies both planning flags preserve read-only semantics.
func TestDesktopDevDryRunNeverBuildsOrStarts(parseTest *testing.T) {
	parseRoot := parseTest.TempDir()
	parseMetadata := `{"schemaVersion":1,"desktop":{"version":1,"nativeEntry":"cmd/desktop/main.go","frontendEntry":"frontend/main.go","assetsDir":"assets","outputPath":"bin/wails-counter.exe","wailsVersion":"v3.0.0-beta.17"}}`
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "gwc-start.json"), []byte(parseMetadata), 0o644); parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	for _, parseFlag := range []string{"-dry-run", "-json"} {
		parseOutput, parseErr := captureLauncherStdout(func() error {
			return (launcher{}).runDev([]string{"-target", "desktop", "-root", parseRoot, "-features", "screens, clipboard", parseFlag})
		})
		if parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		var parsePlan struct {
			DryRun   bool   `json:"dryRun"`
			Features string `json:"features"`
		}
		if parseErr = json.Unmarshal([]byte(parseOutput), &parsePlan); parseErr != nil || !parsePlan.DryRun || parsePlan.Features != "clipboard,screens" {
			parseTest.Fatalf("invalid read-only plan: %s %v", parseOutput, parseErr)
		}
	}
	parseFiles, parseErr := os.ReadDir(parseRoot)
	if parseErr != nil || len(parseFiles) != 1 {
		parseTest.Fatalf("planning changed project files: %v %v", parseFiles, parseErr)
	}
}

// TestDesktopTargetRejectsIgnoredFlags prevents apparently honored output and server configuration.
func TestDesktopTargetRejectsIgnoredFlags(parseTest *testing.T) {
	for _, parseArgs := range [][]string{{"-target", "desktop", "-app", "wrong.go"}, {"-target", "desktop", "-profile", "tinygo"}, {"-target", "desktop", "-out", "wrong.exe"}} {
		if parseErr := (launcher{}).runBuild(parseArgs); parseErr == nil || !strings.Contains(parseErr.Error(), "unsupported") {
			parseTest.Fatalf("ignored desktop build flag: %v: %v", parseArgs, parseErr)
		}
	}
	if parseErr := (launcher{}).runDev([]string{"-target", "desktop", "-port", "9999"}); parseErr == nil || !strings.Contains(parseErr.Error(), "does not support") {
		parseTest.Fatalf("ignored desktop server flag: %v", parseErr)
	}
	if parseErr := (launcher{}).runDev([]string{"-features", "none", "-dry-run"}); parseErr == nil || !strings.Contains(parseErr.Error(), "only valid") {
		parseTest.Fatalf("web accepted desktop features: %v", parseErr)
	}
}

// TestWebBuildRejectsDesktopProfileTag enforces compile-time source exclusion for the claimed target.
func TestWebBuildRejectsDesktopProfileTag(parseTest *testing.T) {
	for _, parseTags := range []string{"gwc_desktop", "production,gwc_desktop", "production gwc_desktop"} {
		if _, parseErr := applyBuildTarget(buildProfile{Tags: parseTags}, "web"); parseErr == nil {
			parseTest.Fatalf("web target accepted desktop tags: %s", parseTags)
		}
	}
}

// TestDesktopTargetAlwaysResolvesItsLane covers direct watch and all-lane configuration.
func TestDesktopTargetAlwaysResolvesItsLane(parseTest *testing.T) {
	for _, parseLanes := range [][]string{nil, {"all"}, {"unit"}, {"desktop"}} {
		parseConfig, parseErr := resolveTestConfig(testConfig{rootPath: parseTest.TempDir(), target: "desktop", features: "screens", lanes: parseLanes})
		if parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		parseCount := 0
		for _, parseLane := range parseConfig.lanes {
			if parseLane == "desktop" {
				parseCount++
			}
		}
		if parseCount != 1 || parseConfig.features != "screens" {
			parseTest.Fatalf("desktop target lost lane/features: %+v", parseConfig)
		}
	}
}
