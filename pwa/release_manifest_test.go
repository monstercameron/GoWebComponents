package pwa

import "testing"

func TestParseWasmReleaseManifestJSONValidatesAndNormalizes(parseT *testing.T) {
	parseManifest, parseErr := ParseWasmReleaseManifestJSON([]byte(`
{
  "package": " ./examples/86-atlas-commerce-os/client ",
  "profile": " production ",
  "goos": " js ",
  "goarch": " wasm ",
  "flags": {"trimpath": true, "ldflags": " -s -w ", "buildvcs": " false ", "compression": true},
  "artifacts": {
    "wasm": {"path": " dist/app.1234.wasm ", "bytes": 42, "sha256": " ABCD "}
  }
}`))
	if parseErr != nil {
		parseT.Fatalf("expected release manifest parse to succeed, got %v", parseErr)
	}
	if parseManifest.Package != "./examples/86-atlas-commerce-os/client" {
		parseT.Fatalf("unexpected normalized package: %q", parseManifest.Package)
	}
	if parseManifest.Artifacts["wasm"].Path != "dist/app.1234.wasm" || parseManifest.Artifacts["wasm"].SHA256 != "abcd" {
		parseT.Fatalf("unexpected normalized artifact: %#v", parseManifest.Artifacts["wasm"])
	}
}

func TestBuildServiceWorkerAssetPlanUsesManifestRevisionAndURLs(parseT *testing.T) {
	parseManifest := WasmReleaseManifest{
		Package: "./examples/app",
		Profile: "production",
		GOOS:    "js",
		GOARCH:  "wasm",
		Artifacts: map[string]WasmReleaseArtifact{
			"wasm": {Path: "assets/app.1234.wasm", SHA256: "deadbeef"},
		},
	}
	parsePlan, parseErr := BuildServiceWorkerAssetPlan(parseManifest, ServiceWorkerAssetPlanOptions{
		BaseURL:       "/static",
		CachePrefix:   "atlas-release",
		ShellURLs:     []string{"/index.html", "/offline.html"},
		ImmutableURLs: []string{"/wasm_exec.js", "/assets/app.css", "/wasm_exec.js"},
	})
	if parseErr != nil {
		parseT.Fatalf("expected asset plan build to succeed, got %v", parseErr)
	}
	if parsePlan.WasmURL != "/static/assets/app.1234.wasm" {
		parseT.Fatalf("unexpected wasm URL: %q", parsePlan.WasmURL)
	}
	if len(parsePlan.PrecacheURLs) != 5 {
		parseT.Fatalf("unexpected precache urls: %#v", parsePlan.PrecacheURLs)
	}
	if parsePlan.CacheName == "" || parsePlan.ManifestRevision == "" {
		parseT.Fatalf("expected derived cache name and revision, got %+v", parsePlan)
	}
	if parsePlan.CacheName[:14] != "atlas-release-" {
		parseT.Fatalf("expected cache prefix, got %q", parsePlan.CacheName)
	}
}

func TestWasmReleaseManifestValidateRequiresWasmArtifact(parseT *testing.T) {
	parseErr := (WasmReleaseManifest{Package: "./examples/app", GOOS: "js", GOARCH: "wasm"}).Validate()
	if parseErr == nil {
		parseT.Fatal("expected missing wasm artifact validation error")
	}
}
