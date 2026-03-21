package pwa

import "testing"

func TestBuildCacheStoragePlanBuildsExplicitStrategies(t *testing.T) {
	assetPlan := ServiceWorkerAssetPlan{
		CacheName:        "atlas-release-1234",
		ManifestRevision: "1234",
		WasmURL:          "/static/app.wasm",
		ShellURLs:        []string{"/index.html", "/offline.html"},
		ImmutableURLs:    []string{"/wasm_exec.js", "/assets/app.css"},
	}
	plan, err := BuildCacheStoragePlan(assetPlan, CacheStoragePlanOptions{
		CachePrefix:   "atlas-release-",
		ScriptURLs:    []string{"/static/app.js"},
		StyleURLs:     []string{"/static/app.css"},
		MediaURLs:     []string{"/static/hero.webp"},
		ShellStrategy: CacheStorageStrategyStaleWhileRevalidate,
	})
	if err != nil {
		t.Fatalf("expected cache storage plan, got %v", err)
	}
	if plan.CacheName != "atlas-release-1234" || plan.CachePrefix != "atlas-release-" {
		t.Fatalf("unexpected cache plan identity: %+v", plan)
	}
	if len(plan.Entries) != 8 {
		t.Fatalf("unexpected cache entry count: %#v", plan.Entries)
	}
	if plan.Entries[0].Kind != CacheStorageAssetKindWasm || plan.Entries[0].Strategy != CacheStorageStrategyCacheFirst {
		t.Fatalf("expected wasm cache-first entry, got %+v", plan.Entries[0])
	}
}
