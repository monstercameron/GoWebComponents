package pwa

import (
	"strings"
	"testing"
)

func benchmarkReleaseManifest() WasmReleaseManifest {
	return WasmReleaseManifest{
		Package: "github.com/monstercameron/GoWebComponents/v4/app",
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

func BenchmarkMarshalManifestJSON(parseB *testing.B) {
	parseB.ReportAllocs()
	parseManifest := Manifest{
		Name:            "GoWebComponents App",
		ShortName:       "GWC",
		StartURL:        "/",
		Display:         "standalone",
		BackgroundColor: "#0f172a",
		ThemeColor:      "#0f172a",
	}
	for parseB.Loop() {
		parseData, parseErr := MarshalManifestJSON(parseManifest)
		if parseErr != nil {
			parseB.Fatalf("MarshalManifestJSON: %v", parseErr)
		}
		if len(parseData) == 0 {
			parseB.Fatal("MarshalManifestJSON returned empty payload")
		}
	}
}

func BenchmarkBuildServiceWorkerAssetPlan(parseB *testing.B) {
	parseB.ReportAllocs()
	parseManifest := benchmarkReleaseManifest()
	parseOptions := ServiceWorkerAssetPlanOptions{
		BaseURL:       "https://example.com",
		CachePrefix:   "bench-release",
		ImmutableURLs: []string{"/assets/wasm_exec.js", "/assets/app.css"},
		ShellURLs:     []string{"/", "/offline"},
	}
	for parseB.Loop() {
		parsePlan, parseErr := BuildServiceWorkerAssetPlan(parseManifest, parseOptions)
		if parseErr != nil {
			parseB.Fatalf("BuildServiceWorkerAssetPlan: %v", parseErr)
		}
		if parsePlan.CacheName == "" || len(parsePlan.PrecacheURLs) == 0 {
			parseB.Fatal("BuildServiceWorkerAssetPlan returned incomplete plan")
		}
	}
}
