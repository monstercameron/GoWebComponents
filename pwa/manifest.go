package pwa

import (
	"encoding/json"
	"errors"
	"strings"
)

type ManifestDisplay string

const (
	ManifestDisplayBrowser               ManifestDisplay = "browser"
	ManifestDisplayMinimalUI             ManifestDisplay = "minimal-ui"
	ManifestDisplayStandalone            ManifestDisplay = "standalone"
	ManifestDisplayFullscreen            ManifestDisplay = "fullscreen"
	ManifestDisplayWindowControlsOverlay ManifestDisplay = "window-controls-overlay"
)

type ManifestOrientation string

const (
	ManifestOrientationAny                ManifestOrientation = "any"
	ManifestOrientationNatural            ManifestOrientation = "natural"
	ManifestOrientationLandscape          ManifestOrientation = "landscape"
	ManifestOrientationLandscapePrimary   ManifestOrientation = "landscape-primary"
	ManifestOrientationLandscapeSecondary ManifestOrientation = "landscape-secondary"
	ManifestOrientationPortrait           ManifestOrientation = "portrait"
	ManifestOrientationPortraitPrimary    ManifestOrientation = "portrait-primary"
	ManifestOrientationPortraitSecondary  ManifestOrientation = "portrait-secondary"
)

type Manifest struct {
	ID                        string               `json:"id,omitempty"`
	Name                      string               `json:"name,omitempty"`
	ShortName                 string               `json:"short_name,omitempty"`
	Description               string               `json:"description,omitempty"`
	StartURL                  string               `json:"start_url,omitempty"`
	Scope                     string               `json:"scope,omitempty"`
	Display                   ManifestDisplay      `json:"display,omitempty"`
	DisplayOverride           []ManifestDisplay    `json:"display_override,omitempty"`
	Orientation               ManifestOrientation  `json:"orientation,omitempty"`
	ThemeColor                string               `json:"theme_color,omitempty"`
	BackgroundColor           string               `json:"background_color,omitempty"`
	Lang                      string               `json:"lang,omitempty"`
	Dir                       string               `json:"dir,omitempty"`
	Categories                []string             `json:"categories,omitempty"`
	Icons                     []ManifestImage      `json:"icons,omitempty"`
	Screenshots               []ManifestImage      `json:"screenshots,omitempty"`
	Shortcuts                 []ManifestShortcut   `json:"shortcuts,omitempty"`
	PreferRelatedApplications bool                 `json:"prefer_related_applications,omitempty"`
	RelatedApplications       []RelatedApplication `json:"related_applications,omitempty"`
}

type ManifestImage struct {
	Src        string `json:"src,omitempty"`
	Sizes      string `json:"sizes,omitempty"`
	Type       string `json:"type,omitempty"`
	Purpose    string `json:"purpose,omitempty"`
	Label      string `json:"label,omitempty"`
	FormFactor string `json:"form_factor,omitempty"`
}

type ManifestShortcut struct {
	Name        string          `json:"name,omitempty"`
	ShortName   string          `json:"short_name,omitempty"`
	Description string          `json:"description,omitempty"`
	URL         string          `json:"url,omitempty"`
	Icons       []ManifestImage `json:"icons,omitempty"`
}

type RelatedApplication struct {
	Platform string `json:"platform,omitempty"`
	URL      string `json:"url,omitempty"`
	ID       string `json:"id,omitempty"`
}

func (parseM Manifest) Normalized() Manifest {
	parseNormalized := parseM
	parseNormalized.ID = strings.TrimSpace(parseNormalized.ID)
	parseNormalized.Name = strings.TrimSpace(parseNormalized.Name)
	parseNormalized.ShortName = strings.TrimSpace(parseNormalized.ShortName)
	parseNormalized.Description = strings.TrimSpace(parseNormalized.Description)
	parseNormalized.StartURL = strings.TrimSpace(parseNormalized.StartURL)
	parseNormalized.Scope = strings.TrimSpace(parseNormalized.Scope)
	parseNormalized.Display = normalizeManifestDisplay(parseNormalized.Display)
	parseNormalized.Orientation = normalizeManifestOrientation(parseNormalized.Orientation)
	parseNormalized.ThemeColor = strings.TrimSpace(parseNormalized.ThemeColor)
	parseNormalized.BackgroundColor = strings.TrimSpace(parseNormalized.BackgroundColor)
	parseNormalized.Lang = strings.TrimSpace(parseNormalized.Lang)
	parseNormalized.Dir = strings.TrimSpace(parseNormalized.Dir)
	parseNormalized.Categories = trimStrings(parseNormalized.Categories)
	parseNormalized.DisplayOverride = normalizeManifestDisplays(parseNormalized.DisplayOverride)
	parseNormalized.Icons = normalizeImages(parseNormalized.Icons)
	parseNormalized.Screenshots = normalizeImages(parseNormalized.Screenshots)
	parseNormalized.Shortcuts = normalizeShortcuts(parseNormalized.Shortcuts)
	parseNormalized.RelatedApplications = normalizeRelatedApplications(parseNormalized.RelatedApplications)
	if parseNormalized.ShortName == "" {
		parseNormalized.ShortName = parseNormalized.Name
	}
	if parseNormalized.Display == "" {
		parseNormalized.Display = ManifestDisplayStandalone
	}
	return parseNormalized
}

func (parseM Manifest) Validate() error {
	parseNormalized := parseM.Normalized()
	if parseNormalized.Name == "" {
		return errors.New("pwa manifest requires a non-empty name")
	}
	if parseNormalized.StartURL == "" {
		return errors.New("pwa manifest requires a non-empty start_url")
	}
	if parseNormalized.Display == "" {
		return errors.New("pwa manifest requires a non-empty display mode")
	}
	if !parseNormalized.Display.Valid() {
		return errors.New("pwa manifest display mode is invalid")
	}
	for _, parseDisplay := range parseNormalized.DisplayOverride {
		if !parseDisplay.Valid() {
			return errors.New("pwa manifest display_override contains an invalid display mode")
		}
	}
	if parseNormalized.Orientation != "" && !parseNormalized.Orientation.Valid() {
		return errors.New("pwa manifest orientation is invalid")
	}
	for _, parseIcon := range parseNormalized.Icons {
		if parseIcon.Src == "" {
			return errors.New("pwa manifest icons require a non-empty src")
		}
	}
	for _, parseShortcut := range parseNormalized.Shortcuts {
		if parseShortcut.Name == "" {
			return errors.New("pwa manifest shortcuts require a non-empty name")
		}
		if parseShortcut.URL == "" {
			return errors.New("pwa manifest shortcuts require a non-empty url")
		}
	}
	return nil
}

// MarshalManifestJSON serializes a normalized, validated Manifest to JSON bytes.
func MarshalManifestJSON(parseManifest Manifest) ([]byte, error) {
	parseNormalized := parseManifest.Normalized()
	if parseErr := parseNormalized.Validate(); parseErr != nil {
		return nil, parseErr
	}
	return json.Marshal(parseNormalized)
}

// MarshalManifestJSONIndented serializes a normalized, validated Manifest to indented JSON bytes.
func MarshalManifestJSONIndented(parseManifest Manifest, parsePrefix, parseIndent string) ([]byte, error) {
	parseNormalized := parseManifest.Normalized()
	if parseErr := parseNormalized.Validate(); parseErr != nil {
		return nil, parseErr
	}
	return json.MarshalIndent(parseNormalized, parsePrefix, parseIndent)
}

func trimStrings(parseValues []string) []string {
	if len(parseValues) == 0 {
		return nil
	}
	parseTrimmed := make([]string, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseValue = strings.TrimSpace(parseValue)
		if parseValue != "" {
			parseTrimmed = append(parseTrimmed, parseValue)
		}
	}
	if len(parseTrimmed) == 0 {
		return nil
	}
	return parseTrimmed
}

func (parseD ManifestDisplay) Normalized() ManifestDisplay {
	return normalizeManifestDisplay(parseD)
}

func (parseD ManifestDisplay) Valid() bool {
	switch parseD.Normalized() {
	case ManifestDisplayBrowser, ManifestDisplayMinimalUI, ManifestDisplayStandalone, ManifestDisplayFullscreen, ManifestDisplayWindowControlsOverlay:
		return true
	default:
		return false
	}
}

func (parseO ManifestOrientation) Normalized() ManifestOrientation {
	return normalizeManifestOrientation(parseO)
}

func (parseO ManifestOrientation) Valid() bool {
	switch parseO.Normalized() {
	case ManifestOrientationAny, ManifestOrientationNatural, ManifestOrientationLandscape, ManifestOrientationLandscapePrimary, ManifestOrientationLandscapeSecondary, ManifestOrientationPortrait, ManifestOrientationPortraitPrimary, ManifestOrientationPortraitSecondary:
		return true
	default:
		return false
	}
}

func normalizeManifestDisplay(parseValue ManifestDisplay) ManifestDisplay {
	return ManifestDisplay(strings.TrimSpace(string(parseValue)))
}

func normalizeManifestOrientation(parseValue ManifestOrientation) ManifestOrientation {
	return ManifestOrientation(strings.TrimSpace(string(parseValue)))
}

func normalizeManifestDisplays(parseValues []ManifestDisplay) []ManifestDisplay {
	if len(parseValues) == 0 {
		return nil
	}
	parseNormalized := make([]ManifestDisplay, 0, len(parseValues))
	for _, parseValue := range parseValues {
		parseValue = parseValue.Normalized()
		if parseValue != "" {
			parseNormalized = append(parseNormalized, parseValue)
		}
	}
	if len(parseNormalized) == 0 {
		return nil
	}
	return parseNormalized
}

func normalizeImages(parseImages []ManifestImage) []ManifestImage {
	if len(parseImages) == 0 {
		return nil
	}
	parseResult := make([]ManifestImage, 0, len(parseImages))
	for _, parseImage := range parseImages {
		parseImage.Src = strings.TrimSpace(parseImage.Src)
		parseImage.Sizes = strings.TrimSpace(parseImage.Sizes)
		parseImage.Type = strings.TrimSpace(parseImage.Type)
		parseImage.Purpose = strings.TrimSpace(parseImage.Purpose)
		parseImage.Label = strings.TrimSpace(parseImage.Label)
		parseImage.FormFactor = strings.TrimSpace(parseImage.FormFactor)
		if parseImage.Src != "" {
			parseResult = append(parseResult, parseImage)
		}
	}
	if len(parseResult) == 0 {
		return nil
	}
	return parseResult
}

func normalizeShortcuts(parseShortcuts []ManifestShortcut) []ManifestShortcut {
	if len(parseShortcuts) == 0 {
		return nil
	}
	parseResult := make([]ManifestShortcut, 0, len(parseShortcuts))
	for _, parseShortcut := range parseShortcuts {
		parseShortcut.Name = strings.TrimSpace(parseShortcut.Name)
		parseShortcut.ShortName = strings.TrimSpace(parseShortcut.ShortName)
		parseShortcut.Description = strings.TrimSpace(parseShortcut.Description)
		parseShortcut.URL = strings.TrimSpace(parseShortcut.URL)
		parseShortcut.Icons = normalizeImages(parseShortcut.Icons)
		if parseShortcut.Name != "" || parseShortcut.URL != "" {
			parseResult = append(parseResult, parseShortcut)
		}
	}
	if len(parseResult) == 0 {
		return nil
	}
	return parseResult
}

func normalizeRelatedApplications(parseApps []RelatedApplication) []RelatedApplication {
	if len(parseApps) == 0 {
		return nil
	}
	parseResult := make([]RelatedApplication, 0, len(parseApps))
	for _, parseApp := range parseApps {
		parseApp.Platform = strings.TrimSpace(parseApp.Platform)
		parseApp.URL = strings.TrimSpace(parseApp.URL)
		parseApp.ID = strings.TrimSpace(parseApp.ID)
		if parseApp.Platform != "" || parseApp.URL != "" || parseApp.ID != "" {
			parseResult = append(parseResult, parseApp)
		}
	}
	if len(parseResult) == 0 {
		return nil
	}
	return parseResult
}
