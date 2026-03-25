package pwa

import (
	"strings"
	"testing"
)

func benchmarkReleaseManifest() WasmReleaseManifest {
	return WasmReleaseManifest{
		Package: "github.com/monstercameron/GoWebComponents/app",
		Profile: "release",
		GOOS:    "js",
		GOARCH:  "wasm",
		Artifacts: map[string]WasmReleaseArtifact{
			"wasm": {
				Path:   "/static/bin/app.wasm",
				Bytes:  1024,
				SHA256: strings.Repeat("a", 64),
			},
		},
	}
}

func BenchmarkMarshalManifestJSON(b *testing.B) {
	b.ReportAllocs()
	manifest := Manifest{
		Name:            "GoWebComponents App",
		ShortName:       "GWC",
		StartURL:        "/",
		Display:         "standalone",
		BackgroundColor: "#0f172a",
		ThemeColor:      "#0f172a",
	}
	for b.Loop() {
		data, err := MarshalManifestJSON(manifest)
		if err != nil {
			b.Fatalf("MarshalManifestJSON: %v", err)
		}
		if len(data) == 0 {
			b.Fatal("MarshalManifestJSON returned empty payload")
		}
	}
}

func BenchmarkBuildServiceWorkerAssetPlan(b *testing.B) {
	b.ReportAllocs()
	manifest := benchmarkReleaseManifest()
	options := ServiceWorkerAssetPlanOptions{
		BaseURL:       "https://example.com",
		CachePrefix:   "bench-release",
		ImmutableURLs: []string{"/assets/wasm_exec.js", "/assets/app.css"},
		ShellURLs:     []string{"/", "/offline"},
	}
	for b.Loop() {
		plan, err := BuildServiceWorkerAssetPlan(manifest, options)
		if err != nil {
			b.Fatalf("BuildServiceWorkerAssetPlan: %v", err)
		}
		if plan.CacheName == "" || len(plan.PrecacheURLs) == 0 {
			b.Fatal("BuildServiceWorkerAssetPlan returned incomplete plan")
		}
	}
}
