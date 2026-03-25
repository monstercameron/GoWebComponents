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

func (m Manifest) Normalized() Manifest {
	normalized := m
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.Name = strings.TrimSpace(normalized.Name)
	normalized.ShortName = strings.TrimSpace(normalized.ShortName)
	normalized.Description = strings.TrimSpace(normalized.Description)
	normalized.StartURL = strings.TrimSpace(normalized.StartURL)
	normalized.Scope = strings.TrimSpace(normalized.Scope)
	normalized.Display = normalizeManifestDisplay(normalized.Display)
	normalized.Orientation = normalizeManifestOrientation(normalized.Orientation)
	normalized.ThemeColor = strings.TrimSpace(normalized.ThemeColor)
	normalized.BackgroundColor = strings.TrimSpace(normalized.BackgroundColor)
	normalized.Lang = strings.TrimSpace(normalized.Lang)
	normalized.Dir = strings.TrimSpace(normalized.Dir)
	normalized.Categories = trimStrings(normalized.Categories)
	normalized.DisplayOverride = normalizeManifestDisplays(normalized.DisplayOverride)
	normalized.Icons = normalizeImages(normalized.Icons)
	normalized.Screenshots = normalizeImages(normalized.Screenshots)
	normalized.Shortcuts = normalizeShortcuts(normalized.Shortcuts)
	normalized.RelatedApplications = normalizeRelatedApplications(normalized.RelatedApplications)
	if normalized.ShortName == "" {
		normalized.ShortName = normalized.Name
	}
	if normalized.Display == "" {
		normalized.Display = ManifestDisplayStandalone
	}
	return normalized
}

func (m Manifest) Validate() error {
	normalized := m.Normalized()
	if normalized.Name == "" {
		return errors.New("pwa manifest requires a non-empty name")
	}
	if normalized.StartURL == "" {
		return errors.New("pwa manifest requires a non-empty start_url")
	}
	if normalized.Display == "" {
		return errors.New("pwa manifest requires a non-empty display mode")
	}
	if !normalized.Display.Valid() {
		return errors.New("pwa manifest display mode is invalid")
	}
	for _, display := range normalized.DisplayOverride {
		if !display.Valid() {
			return errors.New("pwa manifest display_override contains an invalid display mode")
		}
	}
	if normalized.Orientation != "" && !normalized.Orientation.Valid() {
		return errors.New("pwa manifest orientation is invalid")
	}
	for _, icon := range normalized.Icons {
		if icon.Src == "" {
			return errors.New("pwa manifest icons require a non-empty src")
		}
	}
	for _, shortcut := range normalized.Shortcuts {
		if shortcut.Name == "" {
			return errors.New("pwa manifest shortcuts require a non-empty name")
		}
		if shortcut.URL == "" {
			return errors.New("pwa manifest shortcuts require a non-empty url")
		}
	}
	return nil
}

// MarshalManifestJSON serializes a normalized, validated Manifest to JSON bytes.
func MarshalManifestJSON(manifest Manifest) ([]byte, error) {
	normalized := manifest.Normalized()
	if err := normalized.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(normalized)
}

// MarshalManifestJSONIndented serializes a normalized, validated Manifest to indented JSON bytes.
func MarshalManifestJSONIndented(manifest Manifest, prefix, indent string) ([]byte, error) {
	normalized := manifest.Normalized()
	if err := normalized.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(normalized, prefix, indent)
}

func trimStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			trimmed = append(trimmed, value)
		}
	}
	if len(trimmed) == 0 {
		return nil
	}
	return trimmed
}

func (d ManifestDisplay) Normalized() ManifestDisplay {
	return normalizeManifestDisplay(d)
}

func (d ManifestDisplay) Valid() bool {
	switch d.Normalized() {
	case ManifestDisplayBrowser, ManifestDisplayMinimalUI, ManifestDisplayStandalone, ManifestDisplayFullscreen, ManifestDisplayWindowControlsOverlay:
		return true
	default:
		return false
	}
}

func (o ManifestOrientation) Normalized() ManifestOrientation {
	return normalizeManifestOrientation(o)
}

func (o ManifestOrientation) Valid() bool {
	switch o.Normalized() {
	case ManifestOrientationAny, ManifestOrientationNatural, ManifestOrientationLandscape, ManifestOrientationLandscapePrimary, ManifestOrientationLandscapeSecondary, ManifestOrientationPortrait, ManifestOrientationPortraitPrimary, ManifestOrientationPortraitSecondary:
		return true
	default:
		return false
	}
}

func normalizeManifestDisplay(value ManifestDisplay) ManifestDisplay {
	return ManifestDisplay(strings.TrimSpace(string(value)))
}

func normalizeManifestOrientation(value ManifestOrientation) ManifestOrientation {
	return ManifestOrientation(strings.TrimSpace(string(value)))
}

func normalizeManifestDisplays(values []ManifestDisplay) []ManifestDisplay {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]ManifestDisplay, 0, len(values))
	for _, value := range values {
		value = value.Normalized()
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func normalizeImages(images []ManifestImage) []ManifestImage {
	if len(images) == 0 {
		return nil
	}
	result := make([]ManifestImage, 0, len(images))
	for _, image := range images {
		image.Src = strings.TrimSpace(image.Src)
		image.Sizes = strings.TrimSpace(image.Sizes)
		image.Type = strings.TrimSpace(image.Type)
		image.Purpose = strings.TrimSpace(image.Purpose)
		image.Label = strings.TrimSpace(image.Label)
		image.FormFactor = strings.TrimSpace(image.FormFactor)
		if image.Src != "" {
			result = append(result, image)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeShortcuts(shortcuts []ManifestShortcut) []ManifestShortcut {
	if len(shortcuts) == 0 {
		return nil
	}
	result := make([]ManifestShortcut, 0, len(shortcuts))
	for _, shortcut := range shortcuts {
		shortcut.Name = strings.TrimSpace(shortcut.Name)
		shortcut.ShortName = strings.TrimSpace(shortcut.ShortName)
		shortcut.Description = strings.TrimSpace(shortcut.Description)
		shortcut.URL = strings.TrimSpace(shortcut.URL)
		shortcut.Icons = normalizeImages(shortcut.Icons)
		if shortcut.Name != "" || shortcut.URL != "" {
			result = append(result, shortcut)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizeRelatedApplications(apps []RelatedApplication) []RelatedApplication {
	if len(apps) == 0 {
		return nil
	}
	result := make([]RelatedApplication, 0, len(apps))
	for _, app := range apps {
		app.Platform = strings.TrimSpace(app.Platform)
		app.URL = strings.TrimSpace(app.URL)
		app.ID = strings.TrimSpace(app.ID)
		if app.Platform != "" || app.URL != "" || app.ID != "" {
			result = append(result, app)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
