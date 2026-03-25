//go:build js && wasm

package app

import "testing"

func TestNormalizeSettingsSectionID(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "settings-profile", want: settingsSectionProfile},
		{raw: "#settings-prompt", want: settingsSectionPrompt},
		{raw: " SETTINGS-LANGUAGE ", want: settingsSectionLanguage},
		{raw: "#unknown", want: ""},
		{raw: "", want: ""},
	}

	for _, test := range tests {
		if got := normalizeSettingsSectionID(test.raw); got != test.want {
			t.Fatalf("normalizeSettingsSectionID(%q) = %q, want %q", test.raw, got, test.want)
		}
	}
}

func TestBuildSettingsRoute(t *testing.T) {
	if got := buildSettingsRoute(settingsSectionIntelligence); got != "/app/settings?panel=settings-intelligence" {
		t.Fatalf("buildSettingsRoute() = %q", got)
	}
	if got := buildSettingsRoute("unknown"); got != "/app/settings?panel=settings-profile" {
		t.Fatalf("buildSettingsRoute() fallback = %q", got)
	}
}

func TestBuildSettingsReturnRoute(t *testing.T) {
	if got := buildSettingsReturnRoute("/app/thread/abc", settingsSectionIntelligence); got != "/app/thread/abc#settings-intelligence" {
		t.Fatalf("buildSettingsReturnRoute() = %q", got)
	}
	if got := buildSettingsReturnRoute("", settingsSectionIntelligence); got != "/app#settings-intelligence" {
		t.Fatalf("buildSettingsReturnRoute() fallback = %q", got)
	}
}
