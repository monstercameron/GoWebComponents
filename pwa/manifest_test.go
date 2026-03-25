package pwa

import (
	"encoding/json"
	"testing"
)

func TestManifestNormalizesDefaultsAndTrimsFields(parseT *testing.T) {
	parseManifest := Manifest{
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

	parseNormalized := parseManifest.Normalized()
	if parseNormalized.Name != "Atlas Commerce OS" {
		parseT.Fatalf("expected trimmed name, got %q", parseNormalized.Name)
	}
	if parseNormalized.ShortName != "Atlas Commerce OS" {
		parseT.Fatalf("expected short name fallback, got %q", parseNormalized.ShortName)
	}
	if parseNormalized.Display != ManifestDisplayStandalone {
		parseT.Fatalf("expected standalone display default, got %q", parseNormalized.Display)
	}
	if len(parseNormalized.DisplayOverride) != 2 || parseNormalized.DisplayOverride[0] != ManifestDisplayMinimalUI || parseNormalized.DisplayOverride[1] != ManifestDisplayStandalone {
		parseT.Fatalf("unexpected normalized display override: %#v", parseNormalized.DisplayOverride)
	}
	if parseNormalized.Orientation != ManifestOrientationPortraitPrimary {
		parseT.Fatalf("unexpected normalized orientation: %q", parseNormalized.Orientation)
	}
	if len(parseNormalized.Categories) != 2 || parseNormalized.Categories[0] != "shopping" || parseNormalized.Categories[1] != "productivity" {
		parseT.Fatalf("unexpected categories: %#v", parseNormalized.Categories)
	}
	if len(parseNormalized.Icons) != 1 || parseNormalized.Icons[0].Src != "/icon-192.png" {
		parseT.Fatalf("unexpected normalized icons: %#v", parseNormalized.Icons)
	}
	if len(parseNormalized.Shortcuts) != 1 || parseNormalized.Shortcuts[0].URL != "/orders" {
		parseT.Fatalf("unexpected normalized shortcuts: %#v", parseNormalized.Shortcuts)
	}
}

func TestManifestValidateRequiresNameAndStartURL(parseT *testing.T) {
	if parseErr := (Manifest{StartURL: "/"}).Validate(); parseErr == nil {
		parseT.Fatal("expected missing name validation error")
	}
	if parseErr2 := (Manifest{Name: "Atlas"}).Validate(); parseErr2 == nil {
		parseT.Fatal("expected missing start_url validation error")
	}
}

func TestManifestValidateRejectsInvalidDisplayAndOrientation(parseT *testing.T) {
	if parseErr := (Manifest{Name: "Atlas", StartURL: "/", Display: ManifestDisplay("immersive")}).Validate(); parseErr == nil {
		parseT.Fatal("expected invalid display validation error")
	}
	if parseErr2 := (Manifest{Name: "Atlas", StartURL: "/", Orientation: ManifestOrientation("diagonal")}).Validate(); parseErr2 == nil {
		parseT.Fatal("expected invalid orientation validation error")
	}
}

func TestManifestDisplayAndOrientationHelpersNormalizeAndValidate(parseT *testing.T) {
	if parseNormalized := ManifestDisplay(" standalone ").Normalized(); parseNormalized != ManifestDisplayStandalone {
		parseT.Fatalf("expected normalized standalone display, got %q", parseNormalized)
	}
	if !ManifestDisplayMinimalUI.Valid() {
		parseT.Fatal("expected minimal-ui display to be valid")
	}
	if ManifestDisplay("immersive").Valid() {
		parseT.Fatal("expected immersive display to be invalid")
	}
	if parseNormalized2 := ManifestOrientation(" portrait ").Normalized(); parseNormalized2 != ManifestOrientationPortrait {
		parseT.Fatalf("expected normalized portrait orientation, got %q", parseNormalized2)
	}
	if !ManifestOrientationLandscapeSecondary.Valid() {
		parseT.Fatal("expected landscape-secondary orientation to be valid")
	}
	if ManifestOrientation("upside-down").Valid() {
		parseT.Fatal("expected upside-down orientation to be invalid")
	}
}

func TestMarshalManifestJSONProducesExpectedShape(parseT *testing.T) {
	parseData, parseErr := MarshalManifestJSON(Manifest{
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
	if parseErr != nil {
		parseT.Fatalf("expected manifest JSON marshal to succeed, got %v", parseErr)
	}
	var parseDecoded map[string]any
	if parseErr2 := json.Unmarshal(parseData, &parseDecoded); parseErr2 != nil {
		parseT.Fatalf("expected manifest JSON to decode, got %v", parseErr2)
	}
	if parseDecoded["name"] != "Atlas" || parseDecoded["start_url"] != "/" || parseDecoded["display"] != "minimal-ui" || parseDecoded["orientation"] != "landscape" {
		parseT.Fatalf("unexpected manifest payload: %#v", parseDecoded)
	}
	parseIcons := parseDecoded["icons"].([]any)
	if len(parseIcons) != 1 {
		parseT.Fatalf("expected one icon, got %#v", parseDecoded)
	}
	parseIcon := parseIcons[0].(map[string]any)
	if parseIcon["src"] != "/icons/icon-192.png" {
		parseT.Fatalf("unexpected icon payload: %#v", parseIcon)
	}
}
