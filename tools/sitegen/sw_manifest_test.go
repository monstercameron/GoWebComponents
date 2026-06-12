package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestManifestJSON verifies that buildManifestJSON returns valid JSON
// containing the required PWA manifest fields.
func TestManifestJSON(t *testing.T) {
	parseData, parseErr := buildManifestJSON()
	if parseErr != nil {
		t.Fatalf("buildManifestJSON: %v", parseErr)
	}

	// Must be valid JSON.
	var parseDoc map[string]any
	if parseErr2 := json.Unmarshal(parseData, &parseDoc); parseErr2 != nil {
		t.Fatalf("manifest JSON is not valid JSON: %v\n%s", parseErr2, parseData)
	}

	// Required fields.
	parseName, _ := parseDoc["name"].(string)
	if parseName != "GoWebComponents" {
		t.Errorf("manifest name = %q, want %q", parseName, "GoWebComponents")
	}

	parseStartURL, _ := parseDoc["start_url"].(string)
	if parseStartURL == "" {
		t.Error("manifest missing start_url")
	}

	parseDisplay, _ := parseDoc["display"].(string)
	if parseDisplay == "" {
		t.Error("manifest missing display")
	}

	// Icons array must be present with at least one entry pointing at the favicon.
	parseIcons, _ := parseDoc["icons"].([]any)
	if len(parseIcons) == 0 {
		t.Fatal("manifest icons array is empty")
	}
	parseFirstIcon, _ := parseIcons[0].(map[string]any)
	parseSrc, _ := parseFirstIcon["src"].(string)
	if !strings.Contains(parseSrc, "favicon.svg") {
		t.Errorf("manifest icon src = %q, want path containing favicon.svg", parseSrc)
	}

	// Output should be indented (contains newlines).
	if !strings.Contains(string(parseData), "\n") {
		t.Error("manifest JSON is not indented (no newlines found)")
	}
}

// TestServiceWorkerJS verifies that buildServiceWorkerJS generates a service
// worker with a versioned cache name, the precache list, and an activate
// handler that deletes old caches.
func TestServiceWorkerJS(t *testing.T) {
	parseCacheVersion := "abc123def456"
	parseAssets := []string{"index.html", "site.wasm", "favicon.svg", "og-image.svg", "manifest.json"}

	parseSW := buildServiceWorkerJS(parseCacheVersion, parseAssets)

	// Versioned cache name must appear in the output.
	parseExpectedCache := "gwc-shell-" + parseCacheVersion
	if !strings.Contains(parseSW, parseExpectedCache) {
		t.Errorf("sw.js does not contain versioned cache name %q", parseExpectedCache)
	}

	// Every precache asset must appear in the JS output.
	for _, parseAsset := range parseAssets {
		if !strings.Contains(parseSW, "\""+parseAsset+"\"") {
			t.Errorf("sw.js does not contain precache entry %q", parseAsset)
		}
	}

	// activate handler for stale-cache deletion.
	if !strings.Contains(parseSW, "activate") {
		t.Error("sw.js missing activate event listener")
	}
	if !strings.Contains(parseSW, "caches.delete") {
		t.Error("sw.js missing caches.delete call in activate handler")
	}

	// install handler.
	if !strings.Contains(parseSW, "install") {
		t.Error("sw.js missing install event listener")
	}

	// fetch handler.
	if !strings.Contains(parseSW, "fetch") {
		t.Error("sw.js missing fetch event listener")
	}
}

// TestSWManifestBuildBootShellPWAWiring verifies that buildBootShell returns
// HTML containing the manifest link, theme-color meta, and SW registration.
// This test invokes go env GOROOT so it requires the Go toolchain on PATH.
func TestSWManifestBuildBootShellPWAWiring(t *testing.T) {
	parseShell, parseErr := buildBootShell()
	if parseErr != nil {
		t.Skipf("buildBootShell: %v (toolchain not available?)", parseErr)
	}

	if !strings.Contains(parseShell, `<link rel="manifest" href="manifest.json">`) {
		t.Error("boot shell missing <link rel=\"manifest\" href=\"manifest.json\">")
	}
	if !strings.Contains(parseShell, `<meta name="theme-color" content="#0a0f1a">`) {
		t.Error("boot shell missing theme-color meta tag")
	}
	if !strings.Contains(parseShell, `navigator.serviceWorker.register`) {
		t.Error("boot shell missing navigator.serviceWorker.register call")
	}
	if !strings.Contains(parseShell, `sw.js`) {
		t.Error("boot shell does not reference sw.js")
	}
}

// TestWasmCacheVersion verifies that wasmCacheVersion returns a 12-character
// hex string and that different inputs produce different versions.
func TestWasmCacheVersion(t *testing.T) {
	parseVersionA := wasmCacheVersion([]byte("content-a"))
	parseVersionB := wasmCacheVersion([]byte("content-b"))

	if len(parseVersionA) != 12 {
		t.Errorf("cache version length = %d, want 12", len(parseVersionA))
	}
	if parseVersionA == parseVersionB {
		t.Error("different wasm bytes produced the same cache version")
	}
	// Same input must be stable.
	if wasmCacheVersion([]byte("content-a")) != parseVersionA {
		t.Error("wasmCacheVersion is not deterministic")
	}
}
