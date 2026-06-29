package devtools

import (
	"encoding/json"
	"strings"
)

const extensionManifestVersion = 3

// BrowserExtensionManifest describes the browser-extension shell that hosts the
// GWC devtools panel in Chrome-compatible and Firefox-compatible browsers.
type BrowserExtensionManifest struct {
	ManifestVersion int               `json:"manifest_version"`
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	Description     string            `json:"description"`
	DevtoolsPage    string            `json:"devtools_page"`
	Permissions     []string          `json:"permissions,omitempty"`
	HostPermissions []string          `json:"host_permissions,omitempty"`
	Background      map[string]string `json:"background,omitempty"`
	BrowserSpecific map[string]any    `json:"browser_specific_settings,omitempty"`
}

// ExtensionPanelPayload is the stable JSON envelope consumed by companion
// browser extensions. It intentionally mirrors Snapshot sections instead of DOM
// internals so Chrome and Firefox panels can share one renderer.
type ExtensionPanelPayload struct {
	SchemaVersion string             `json:"schemaVersion"`
	Route         Route              `json:"route"`
	Tree          *Node              `json:"tree,omitempty"`
	Stats         Stats              `json:"stats"`
	Profiling     Profiling          `json:"profiling"`
	Extensions    []ExtensionSection `json:"extensions,omitempty"`
	Diagnostics   []Diagnostic       `json:"diagnostics,omitempty"`
	Logs          []Log              `json:"logs,omitempty"`
}

// BrowserExtensionManifestFor returns a manifest for a GWC companion devtools extension.
// Pass "firefox" to include Firefox's browser_specific_settings block.
func BrowserExtensionManifestFor(parseBrowser string) BrowserExtensionManifest {
	parseManifest := BrowserExtensionManifest{
		ManifestVersion: extensionManifestVersion,
		Name:            "GWC Devtools",
		Version:         "1.0.0",
		Description:     "Inspect GoWebComponents component trees, props/state, atoms, commits, and diagnostics.",
		DevtoolsPage:    "devtools.html",
		Permissions:     []string{"storage", "scripting"},
		HostPermissions: []string{"<all_urls>"},
		Background: map[string]string{
			"service_worker": "background.js",
		},
	}
	if strings.EqualFold(strings.TrimSpace(parseBrowser), "firefox") {
		parseManifest.BrowserSpecific = map[string]any{
			"gecko": map[string]any{
				"id": "gwc-devtools@monstercameron.dev",
			},
		}
	}
	return parseManifest
}

// ExportBrowserExtensionManifestJSON serializes a browser-specific extension manifest.
func ExportBrowserExtensionManifestJSON(parseBrowser string) ([]byte, error) {
	return json.MarshalIndent(BrowserExtensionManifestFor(parseBrowser), "", "  ")
}

// BuildExtensionPanelPayload projects a live devtools snapshot into the stable
// extension bridge payload.
func BuildExtensionPanelPayload(parseSnapshot Snapshot) ExtensionPanelPayload {
	return ExtensionPanelPayload{
		SchemaVersion: "gwc.devtools.extension.v1",
		Route:         parseSnapshot.Route,
		Tree:          parseSnapshot.Tree,
		Stats:         parseSnapshot.Stats,
		Profiling:     parseSnapshot.Profiling,
		Extensions:    cloneExtensionSections(parseSnapshot.Extensions),
		Diagnostics:   append([]Diagnostic(nil), parseSnapshot.Diagnostics...),
		Logs:          append([]Log(nil), parseSnapshot.Logs...),
	}
}

// ExportExtensionPanelPayloadJSON serializes the stable browser-extension panel payload.
func ExportExtensionPanelPayloadJSON(parseSnapshot Snapshot) ([]byte, error) {
	return json.Marshal(BuildExtensionPanelPayload(parseSnapshot))
}
