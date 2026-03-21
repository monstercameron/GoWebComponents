package pwa

import (
	"encoding/json"
	"testing"
)

func TestManifestNormalizesDefaultsAndTrimsFields(t *testing.T) {
	manifest := Manifest{
		Name:            "  Atlas Commerce OS  ",
		ShortName:       " ",
		StartURL:        " /app ",
		Display:         " ",
		DisplayOverride: []ManifestDisplay{" minimal-ui ", " ", ManifestDisplayStandalone},
		Orientation:     " portrait-primary ",
		Categories:      []string{" shopping ", "", " productivity "},
		Icons:           []ManifestImage{{Src: " /icon-192.png ", Sizes: " 192x192 "}, {Src: "  "}},
		Shortcuts:       []ManifestShortcut{{Name: " Orders ", URL: " /orders "}},
	}

	normalized := manifest.Normalized()
	if normalized.Name != "Atlas Commerce OS" {
		t.Fatalf("expected trimmed name, got %q", normalized.Name)
	}
	if normalized.ShortName != "Atlas Commerce OS" {
		t.Fatalf("expected short name fallback, got %q", normalized.ShortName)
	}
	if normalized.Display != ManifestDisplayStandalone {
		t.Fatalf("expected standalone display default, got %q", normalized.Display)
	}
	if len(normalized.DisplayOverride) != 2 || normalized.DisplayOverride[0] != ManifestDisplayMinimalUI || normalized.DisplayOverride[1] != ManifestDisplayStandalone {
		t.Fatalf("unexpected normalized display override: %#v", normalized.DisplayOverride)
	}
	if normalized.Orientation != ManifestOrientationPortraitPrimary {
		t.Fatalf("unexpected normalized orientation: %q", normalized.Orientation)
	}
	if len(normalized.Categories) != 2 || normalized.Categories[0] != "shopping" || normalized.Categories[1] != "productivity" {
		t.Fatalf("unexpected categories: %#v", normalized.Categories)
	}
	if len(normalized.Icons) != 1 || normalized.Icons[0].Src != "/icon-192.png" {
		t.Fatalf("unexpected normalized icons: %#v", normalized.Icons)
	}
	if len(normalized.Shortcuts) != 1 || normalized.Shortcuts[0].URL != "/orders" {
		t.Fatalf("unexpected normalized shortcuts: %#v", normalized.Shortcuts)
	}
}

func TestManifestValidateRequiresNameAndStartURL(t *testing.T) {
	if err := (Manifest{StartURL: "/"}).Validate(); err == nil {
		t.Fatal("expected missing name validation error")
	}
	if err := (Manifest{Name: "Atlas"}).Validate(); err == nil {
		t.Fatal("expected missing start_url validation error")
	}
}

func TestManifestValidateRejectsInvalidDisplayAndOrientation(t *testing.T) {
	if err := (Manifest{Name: "Atlas", StartURL: "/", Display: ManifestDisplay("immersive")}).Validate(); err == nil {
		t.Fatal("expected invalid display validation error")
	}
	if err := (Manifest{Name: "Atlas", StartURL: "/", Orientation: ManifestOrientation("diagonal")}).Validate(); err == nil {
		t.Fatal("expected invalid orientation validation error")
	}
}

func TestManifestDisplayAndOrientationHelpersNormalizeAndValidate(t *testing.T) {
	if normalized := ManifestDisplay(" standalone ").Normalized(); normalized != ManifestDisplayStandalone {
		t.Fatalf("expected normalized standalone display, got %q", normalized)
	}
	if !ManifestDisplayMinimalUI.Valid() {
		t.Fatal("expected minimal-ui display to be valid")
	}
	if ManifestDisplay("immersive").Valid() {
		t.Fatal("expected immersive display to be invalid")
	}
	if normalized := ManifestOrientation(" portrait ").Normalized(); normalized != ManifestOrientationPortrait {
		t.Fatalf("expected normalized portrait orientation, got %q", normalized)
	}
	if !ManifestOrientationLandscapeSecondary.Valid() {
		t.Fatal("expected landscape-secondary orientation to be valid")
	}
	if ManifestOrientation("upside-down").Valid() {
		t.Fatal("expected upside-down orientation to be invalid")
	}
}

func TestMarshalManifestJSONProducesExpectedShape(t *testing.T) {
	data, err := MarshalManifestJSON(Manifest{
		Name:            "Atlas",
		ShortName:       "Atlas",
		StartURL:        "/",
		Display:         ManifestDisplayMinimalUI,
		Orientation:     ManifestOrientationLandscape,
		ThemeColor:      "#07111f",
		BackgroundColor: "#ffffff",
		Icons: []ManifestImage{{
			Src:   "/icons/icon-192.png",
			Sizes: "192x192",
			Type:  "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("expected manifest JSON marshal to succeed, got %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("expected manifest JSON to decode, got %v", err)
	}
	if decoded["name"] != "Atlas" || decoded["start_url"] != "/" || decoded["display"] != "minimal-ui" || decoded["orientation"] != "landscape" {
		t.Fatalf("unexpected manifest payload: %#v", decoded)
	}
	icons := decoded["icons"].([]any)
	if len(icons) != 1 {
		t.Fatalf("expected one icon, got %#v", decoded)
	}
	icon := icons[0].(map[string]any)
	if icon["src"] != "/icons/icon-192.png" {
		t.Fatalf("unexpected icon payload: %#v", icon)
	}
}
