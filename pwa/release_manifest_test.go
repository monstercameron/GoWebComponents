package pwa

import "testing"

func TestParseWasmReleaseManifestJSONValidatesAndNormalizes(t *testing.T) {
	manifest, err := ParseWasmReleaseManifestJSON([]byte(`
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
	if err != nil {
		t.Fatalf("expected release manifest parse to succeed, got %v", err)
	}
	if manifest.Package != "./examples/86-atlas-commerce-os/client" {
		t.Fatalf("unexpected normalized package: %q", manifest.Package)
	}
	if manifest.Artifacts["wasm"].Path != "dist/app.1234.wasm" || manifest.Artifacts["wasm"].SHA256 != "abcd" {
		t.Fatalf("unexpected normalized artifact: %#v", manifest.Artifacts["wasm"])
	}
}

func TestBuildServiceWorkerAssetPlanUsesManifestRevisionAndURLs(t *testing.T) {
	manifest := WasmReleaseManifest{
		Package: "./examples/app",
		Profile: "production",
		GOOS:    "js",
		GOARCH:  "wasm",
		Artifacts: map[string]WasmReleaseArtifact{
			"wasm": {Path: "assets/app.1234.wasm", SHA256: "deadbeef"},
		},
	}
	plan, err := BuildServiceWorkerAssetPlan(manifest, ServiceWorkerAssetPlanOptions{
		BaseURL:       "/static",
		CachePrefix:   "atlas-release",
		ShellURLs:     []string{"/index.html", "/offline.html"},
		ImmutableURLs: []string{"/wasm_exec.js", "/assets/app.css", "/wasm_exec.js"},
	})
	if err != nil {
		t.Fatalf("expected asset plan build to succeed, got %v", err)
	}
	if plan.WasmURL != "/static/assets/app.1234.wasm" {
		t.Fatalf("unexpected wasm URL: %q", plan.WasmURL)
	}
	if len(plan.PrecacheURLs) != 5 {
		t.Fatalf("unexpected precache urls: %#v", plan.PrecacheURLs)
	}
	if plan.CacheName == "" || plan.ManifestRevision == "" {
		t.Fatalf("expected derived cache name and revision, got %+v", plan)
	}
	if plan.CacheName[:14] != "atlas-release-" {
		t.Fatalf("expected cache prefix, got %q", plan.CacheName)
	}
}

func TestWasmReleaseManifestValidateRequiresWasmArtifact(t *testing.T) {
	err := (WasmReleaseManifest{Package: "./examples/app", GOOS: "js", GOARCH: "wasm"}).Validate()
	if err == nil {
		t.Fatal("expected missing wasm artifact validation error")
	}
}
