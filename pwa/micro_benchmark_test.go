package pwa

import (
	"strings"
	"testing"
)

func BenchmarkBuildServiceWorkerAssetPlanMicro(b *testing.B) {
	manifest := WasmReleaseManifest{
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
	options := ServiceWorkerAssetPlanOptions{
		BaseURL:       "/static",
		CachePrefix:   "bench",
		ImmutableURLs: []string{"/wasm_exec.js", "/assets/app.css"},
		ShellURLs:     []string{"/index.html", "/offline.html"},
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := BuildServiceWorkerAssetPlan(manifest, options)
		if err != nil {
			b.Fatal(err)
		}
	}
}
