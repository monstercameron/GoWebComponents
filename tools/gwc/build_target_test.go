package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNormalizeBuildTargetPreservesWebDefault verifies the legacy target.
func TestNormalizeBuildTargetPreservesWebDefault(parseT *testing.T) {
	for _, parseTarget := range []string{"", "web", "WEB"} {
		if parseGot, parseErr := normalizeBuildTarget(parseTarget); parseErr != nil || parseGot != "web" {
			parseT.Fatalf("normalizeBuildTarget(%q) = %q, %v", parseTarget, parseGot, parseErr)
		}
	}
}

// TestApplyBuildTargetAddsDesktopTag verifies only desktop builds opt into the tag.
func TestApplyBuildTargetAddsDesktopTag(parseT *testing.T) {
	parseWeb, parseErr := applyBuildTarget(buildProfile{Tags: "production"}, "web")
	if parseErr != nil || parseWeb.Tags != "production" {
		parseT.Fatalf("web profile changed: %#v, %v", parseWeb, parseErr)
	}
	parseDesktop, parseErr := applyBuildTarget(buildProfile{Tags: "production"}, "desktop")
	if parseErr != nil || parseDesktop.Tags != "production,gwc_desktop" {
		parseT.Fatalf("desktop profile = %#v, %v", parseDesktop, parseErr)
	}
}

// TestApplyBuildTargetRejectsUnsupportedTarget verifies invalid and incompatible targets.
func TestApplyBuildTargetRejectsUnsupportedTarget(parseT *testing.T) {
	if _, parseErr := applyBuildTarget(buildProfile{}, "mobile"); parseErr == nil {
		parseT.Fatal("unsupported target accepted")
	}
	if _, parseErr := applyBuildTarget(buildProfile{Toolchain: "tinygo"}, "desktop"); parseErr == nil {
		parseT.Fatal("tinygo desktop target accepted")
	}
}

// TestNormalizeDesktopFeaturesRejectsMalformedValues verifies validation before child builds.
func TestNormalizeDesktopFeaturesRejectsMalformedValues(parseT *testing.T) {
	if parseGot, parseErr := normalizeDesktopFeatures("persistent-storage,clipboard,persistent-storage"); parseErr != nil || parseGot != "clipboard,persistent-storage" {
		parseT.Fatalf("feature normalization = %q, %v", parseGot, parseErr)
	}
	for _, parseValue := range []string{"", "storage feature", "storage;exec", "storage\nother"} {
		if _, parseErr := normalizeDesktopFeatures(parseValue); parseErr == nil {
			parseT.Fatalf("invalid feature %q accepted", parseValue)
		}
	}
}

// TestDesktopBuildTagFixtureSelectsExactlyOneSource verifies web/desktop source separation.
func TestDesktopBuildTagFixtureSelectsExactlyOneSource(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module example.com/tagfixture\n\ngo 1.26\n"), 0o644); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseSources := map[string]string{
		"web.go":      "//go:build !gwc_desktop\n\npackage fixture\n\nconst Target = \"web\"\n",
		"desktop.go":  "//go:build gwc_desktop\n\npackage fixture\n\nconst Target = \"desktop\"\n",
		"web_only.go": "//go:build !gwc_desktop\n\npackage fixture\n\nconst WebOnly = true\n",
	}
	for parseName, parseSource := range parseSources {
		if parseErr := os.WriteFile(filepath.Join(parseRoot, parseName), []byte(parseSource), 0o644); parseErr != nil {
			parseT.Fatal(parseErr)
		}
	}
	parseList := func(parseTags string) string {
		parseArgs := []string{"list"}
		if parseTags != "" {
			parseArgs = append(parseArgs, "-tags", parseTags)
		}
		parseArgs = append(parseArgs, "-f", "{{join .GoFiles \" \"}}")
		parseCommand := exec.Command("go", parseArgs...)
		parseCommand.Dir = parseRoot
		parseCommand.Env = desktopNativeEnv()
		parseOutput, parseErr := parseCommand.Output()
		if parseErr != nil {
			parseT.Fatal(parseErr)
		}
		parseBuildArgs := []string{"build"}
		if parseTags != "" {
			parseBuildArgs = append(parseBuildArgs, "-tags", parseTags)
		}
		parseBuild := exec.Command("go", parseBuildArgs...)
		parseBuild.Dir = parseRoot
		parseBuild.Env = desktopNativeEnv()
		if parseBuildOutput, parseBuildErr := parseBuild.CombinedOutput(); parseBuildErr != nil {
			parseT.Fatalf("selected source files do not actually compile: %v\n%s", parseBuildErr, parseBuildOutput)
		}
		return string(parseOutput)
	}
	if parseWeb := parseList(""); !strings.Contains(parseWeb, "web_only.go") || strings.Contains(parseWeb, "desktop.go") {
		parseT.Fatalf("web files = %s", parseWeb)
	}
	if parseDesktop := parseList("gwc_desktop"); !strings.Contains(parseDesktop, "desktop.go") || strings.Contains(parseDesktop, "web_only.go") {
		parseT.Fatalf("desktop files = %s", parseDesktop)
	}
	for _, parseTags := range []string{"", "gwc_desktop"} {
		parseArgs := []string{"test", "-run", "^$"}
		if parseTags != "" {
			parseArgs = append(parseArgs, "-tags", parseTags)
		}
		parseCommand := exec.Command("go", parseArgs...)
		parseCommand.Dir = parseRoot
		if parseOutput, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
			parseT.Fatalf("fixture test tags=%q: %v\n%s", parseTags, parseErr, parseOutput)
		}
	}
}
