//go:build js && wasm

package app

import "testing"

func TestNormalizeSettingsSectionID(parseT *testing.T) {
	parseTests := []struct {
		raw  string
		want string
	}{
		{raw: "settings-profile", want: settingsSectionProfile},
		{raw: "#settings-prompt", want: settingsSectionPrompt},
		{raw: "settings-speech", want: settingsSectionSpeech},
		{raw: " SETTINGS-LANGUAGE ", want: settingsSectionLanguage},
		{raw: "#unknown", want: ""},
		{raw: "", want: ""},
	}

	for _, parseTest := range parseTests {
		if parseGot := parseNormalizeSettingsSectionID(parseTest.raw); parseGot != parseTest.want {
			parseT.Fatalf("normalizeSettingsSectionID(%q) = %q, want %q", parseTest.raw, parseGot, parseTest.want)
		}
	}
}

func TestBuildSettingsRoute(parseT *testing.T) {
	if parseGot := buildSettingsRoute(settingsSectionIntelligence); parseGot != "/app/settings?panel=settings-intelligence" {
		parseT.Fatalf("buildSettingsRoute() = %q", parseGot)
	}
	if parseGot2 := buildSettingsRoute("unknown"); parseGot2 != "/app/settings?panel=settings-profile" {
		parseT.Fatalf("buildSettingsRoute() fallback = %q", parseGot2)
	}
}

func TestBuildSettingsReturnRoute(parseT *testing.T) {
	if parseGot := buildSettingsReturnRoute("/app/thread/abc", settingsSectionIntelligence); parseGot != "/app/thread/abc#settings-intelligence" {
		parseT.Fatalf("buildSettingsReturnRoute() = %q", parseGot)
	}
	if parseGot2 := buildSettingsReturnRoute("", settingsSectionIntelligence); parseGot2 != "/app#settings-intelligence" {
		parseT.Fatalf("buildSettingsReturnRoute() fallback = %q", parseGot2)
	}
}
