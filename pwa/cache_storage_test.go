package pwa

import "testing"

func TestBuildCacheStoragePlanBuildsExplicitStrategies(parseT *testing.T) {
	parseAssetPlan := ServiceWorkerAssetPlan{
		CacheName:        "atlas-release-1234",
		ManifestRevision: "1234",
		WasmURL:          "/static/app.wasm",
		ShellURLs:        []string{"/index.html", "/offline.html"},
		ImmutableURLs:    []string{"/wasm_exec.js", "/assets/app.css"},
	}
	parsePlan, parseErr := BuildCacheStoragePlan(parseAssetPlan, CacheStoragePlanOptions{
		CachePrefix:   "atlas-release-",
		ScriptURLs:    []string{"/static/app.js"},
		StyleURLs:     []string{"/static/app.css"},
		MediaURLs:     []string{"/static/hero.webp"},
		ShellStrategy: CacheStorageStrategyStaleWhileRevalidate,
	})
	if parseErr != nil {
		parseT.Fatalf("expected cache storage plan, got %v", parseErr)
	}
	if parsePlan.CacheName != "atlas-release-1234" || parsePlan.CachePrefix != "atlas-release-" {
		parseT.Fatalf("unexpected cache plan identity: %+v", parsePlan)
	}
	if len(parsePlan.Entries) != 8 {
		parseT.Fatalf("unexpected cache entry count: %#v", parsePlan.Entries)
	}
	if parsePlan.Entries[0].Kind != CacheStorageAssetKindWasm || parsePlan.Entries[0].Strategy != CacheStorageStrategyCacheFirst {
		parseT.Fatalf("expected wasm cache-first entry, got %+v", parsePlan.Entries[0])
	}
}
