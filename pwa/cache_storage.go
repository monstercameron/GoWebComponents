package pwa

import "context"

type CacheStorageStrategy string

const (
	CacheStorageStrategyCacheFirst           CacheStorageStrategy = "cache-first"
	CacheStorageStrategyNetworkFirst         CacheStorageStrategy = "network-first"
	CacheStorageStrategyStaleWhileRevalidate CacheStorageStrategy = "stale-while-revalidate"
)

type CacheStorageAssetKind string

const (
	CacheStorageAssetKindShell  CacheStorageAssetKind = "shell"
	CacheStorageAssetKindWasm   CacheStorageAssetKind = "wasm"
	CacheStorageAssetKindScript CacheStorageAssetKind = "script"
	CacheStorageAssetKindStyle  CacheStorageAssetKind = "style"
	CacheStorageAssetKindMedia  CacheStorageAssetKind = "media"
	CacheStorageAssetKindAsset  CacheStorageAssetKind = "asset"
)

type CacheStorageEntry struct {
	URL      string
	Kind     CacheStorageAssetKind
	Strategy CacheStorageStrategy
}

type CacheStoragePlanOptions struct {
	CachePrefix   string
	ShellURLs     []string
	ScriptURLs    []string
	StyleURLs     []string
	MediaURLs     []string
	AssetURLs     []string
	ShellStrategy CacheStorageStrategy
}

type CacheStoragePlan struct {
	CacheName        string
	CachePrefix      string
	ManifestRevision string
	Entries          []CacheStorageEntry
}

type CacheStorageSnapshot struct {
	CacheName   string
	CachePrefix string
	EntryCount  int
	Entries     []CacheStorageEntry
	CacheNames  []string
}

func BuildCacheStoragePlan(assetPlan ServiceWorkerAssetPlan, options ...CacheStoragePlanOptions) (CacheStoragePlan, error) {
	resolved := CacheStoragePlanOptions{}
	if len(options) > 0 {
		resolved = options[0]
	}
	if resolved.CachePrefix == "" {
		resolved.CachePrefix = assetPlan.CacheName
	}
	if resolved.ShellStrategy == "" {
		resolved.ShellStrategy = CacheStorageStrategyNetworkFirst
	}
	entries := make([]CacheStorageEntry, 0, 1+len(assetPlan.ShellURLs)+len(assetPlan.ImmutableURLs)+len(resolved.ScriptURLs)+len(resolved.StyleURLs)+len(resolved.MediaURLs)+len(resolved.AssetURLs))
	if assetPlan.WasmURL != "" {
		entries = append(entries, CacheStorageEntry{URL: assetPlan.WasmURL, Kind: CacheStorageAssetKindWasm, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, url := range dedupeServiceWorkerURLs(append(append([]string{}, assetPlan.ShellURLs...), resolved.ShellURLs...)) {
		entries = append(entries, CacheStorageEntry{URL: url, Kind: CacheStorageAssetKindShell, Strategy: resolved.ShellStrategy})
	}
	for _, url := range dedupeServiceWorkerURLs(append(append([]string{}, assetPlan.ImmutableURLs...), resolved.AssetURLs...)) {
		entries = append(entries, CacheStorageEntry{URL: url, Kind: CacheStorageAssetKindAsset, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, url := range dedupeServiceWorkerURLs(resolved.ScriptURLs) {
		entries = append(entries, CacheStorageEntry{URL: url, Kind: CacheStorageAssetKindScript, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, url := range dedupeServiceWorkerURLs(resolved.StyleURLs) {
		entries = append(entries, CacheStorageEntry{URL: url, Kind: CacheStorageAssetKindStyle, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, url := range dedupeServiceWorkerURLs(resolved.MediaURLs) {
		entries = append(entries, CacheStorageEntry{URL: url, Kind: CacheStorageAssetKindMedia, Strategy: CacheStorageStrategyCacheFirst})
	}
	entries = dedupeCacheStorageEntries(entries)
	return CacheStoragePlan{
		CacheName:        assetPlan.CacheName,
		CachePrefix:      resolved.CachePrefix,
		ManifestRevision: assetPlan.ManifestRevision,
		Entries:          entries,
	}, nil
}

func dedupeCacheStorageEntries(entries []CacheStorageEntry) []CacheStorageEntry {
	if len(entries) == 0 {
		return nil
	}
	seen := map[string]bool{}
	resolved := make([]CacheStorageEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.URL == "" || seen[entry.URL] {
			continue
		}
		seen[entry.URL] = true
		resolved = append(resolved, entry)
	}
	if len(resolved) == 0 {
		return nil
	}
	return resolved
}

type CacheStorageManager struct {
	sync    func(context.Context, CacheStoragePlan) (CacheStorageSnapshot, error)
	inspect func(context.Context, CacheStoragePlan) (CacheStorageSnapshot, error)
}

func (m CacheStorageManager) Sync(ctx context.Context, plan CacheStoragePlan) (CacheStorageSnapshot, error) {
	if m.sync == nil {
		return CacheStorageSnapshot{}, cacheStorageUnavailable("CacheStorageManager.Sync", plan.CacheName)
	}
	return m.sync(ctx, plan)
}

func (m CacheStorageManager) Inspect(ctx context.Context, plan CacheStoragePlan) (CacheStorageSnapshot, error) {
	if m.inspect == nil {
		return CacheStorageSnapshot{}, cacheStorageUnavailable("CacheStorageManager.Inspect", plan.CacheName)
	}
	return m.inspect(ctx, plan)
}
