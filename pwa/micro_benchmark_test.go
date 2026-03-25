package pwa

import (
	"strings"
	"testing"
)

func BenchmarkBuildServiceWorkerAssetPlanMicro(parseB *testing.B) {
	parseManifest := WasmReleaseManifest{
		Package: "./examples/app",
		Profile: "production",
		GOOS:    "js",
		GOARCH:  "wasm",
		Artifacts: map[string]WasmReleaseArtifact{
			"wasm": {
				Path:   "assets/app.1234.wasm",
				SHA256: strings.Repeat("a", 64),
			},
		},
	}
	parseOptions := ServiceWorkerAssetPlanOptions{
		BaseURL:       "/static",
		CachePrefix:   "bench",
		ImmutableURLs: []string{"/wasm_exec.js", "/assets/app.css"},
		ShellURLs:     []string{"/index.html", "/offline.html"},
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_, parseErr := BuildServiceWorkerAssetPlan(parseManifest, parseOptions)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
	}
}
