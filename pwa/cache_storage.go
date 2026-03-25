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

// BuildCacheStoragePlan builds a CacheStoragePlan from a ServiceWorkerAssetPlan and optional options.
func BuildCacheStoragePlan(parseAssetPlan ServiceWorkerAssetPlan, parseOptions ...CacheStoragePlanOptions) (CacheStoragePlan, error) {
	parseResolved := CacheStoragePlanOptions{}
	if len(parseOptions) > 0 {
		parseResolved = parseOptions[0]
	}
	if parseResolved.CachePrefix == "" {
		parseResolved.CachePrefix = parseAssetPlan.CacheName
	}
	if parseResolved.ShellStrategy == "" {
		parseResolved.ShellStrategy = CacheStorageStrategyNetworkFirst
	}
	parseEntries := make([]CacheStorageEntry, 0, 1+len(parseAssetPlan.ShellURLs)+len(parseAssetPlan.ImmutableURLs)+len(parseResolved.ScriptURLs)+len(parseResolved.StyleURLs)+len(parseResolved.MediaURLs)+len(parseResolved.AssetURLs))
	if parseAssetPlan.WasmURL != "" {
		parseEntries = append(parseEntries, CacheStorageEntry{URL: parseAssetPlan.WasmURL, Kind: CacheStorageAssetKindWasm, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, parseUrl := range dedupeServiceWorkerURLs(append(append([]string{}, parseAssetPlan.ShellURLs...), parseResolved.ShellURLs...)) {
		parseEntries = append(parseEntries, CacheStorageEntry{URL: parseUrl, Kind: CacheStorageAssetKindShell, Strategy: parseResolved.ShellStrategy})
	}
	for _, parseUrl2 := range dedupeServiceWorkerURLs(append(append([]string{}, parseAssetPlan.ImmutableURLs...), parseResolved.AssetURLs...)) {
		parseEntries = append(parseEntries, CacheStorageEntry{URL: parseUrl2, Kind: CacheStorageAssetKindAsset, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, parseUrl3 := range dedupeServiceWorkerURLs(parseResolved.ScriptURLs) {
		parseEntries = append(parseEntries, CacheStorageEntry{URL: parseUrl3, Kind: CacheStorageAssetKindScript, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, parseUrl4 := range dedupeServiceWorkerURLs(parseResolved.StyleURLs) {
		parseEntries = append(parseEntries, CacheStorageEntry{URL: parseUrl4, Kind: CacheStorageAssetKindStyle, Strategy: CacheStorageStrategyCacheFirst})
	}
	for _, parseUrl5 := range dedupeServiceWorkerURLs(parseResolved.MediaURLs) {
		parseEntries = append(parseEntries, CacheStorageEntry{URL: parseUrl5, Kind: CacheStorageAssetKindMedia, Strategy: CacheStorageStrategyCacheFirst})
	}
	parseEntries = dedupeCacheStorageEntries(parseEntries)
	return CacheStoragePlan{
		CacheName:        parseAssetPlan.CacheName,
		CachePrefix:      parseResolved.CachePrefix,
		ManifestRevision: parseAssetPlan.ManifestRevision,
		Entries:          parseEntries,
	}, nil
}

func dedupeCacheStorageEntries(parseEntries []CacheStorageEntry) []CacheStorageEntry {
	if len(parseEntries) == 0 {
		return nil
	}
	parseSeen := map[string]bool{}
	parseResolved := make([]CacheStorageEntry, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		if parseEntry.URL == "" || parseSeen[parseEntry.URL] {
			continue
		}
		parseSeen[parseEntry.URL] = true
		parseResolved = append(parseResolved, parseEntry)
	}
	if len(parseResolved) == 0 {
		return nil
	}
	return parseResolved
}

type CacheStorageManager struct {
	sync    func(context.Context, CacheStoragePlan) (CacheStorageSnapshot, error)
	inspect func(context.Context, CacheStoragePlan) (CacheStorageSnapshot, error)
}

func (parseM CacheStorageManager) Sync(parseCtx context.Context, parsePlan CacheStoragePlan) (CacheStorageSnapshot, error) {
	if parseM.sync == nil {
		return CacheStorageSnapshot{}, cacheStorageUnavailable("CacheStorageManager.Sync", parsePlan.CacheName)
	}
	return parseM.sync(parseCtx, parsePlan)
}

func (parseM CacheStorageManager) Inspect(parseCtx context.Context, parsePlan CacheStoragePlan) (CacheStorageSnapshot, error) {
	if parseM.inspect == nil {
		return CacheStorageSnapshot{}, cacheStorageUnavailable("CacheStorageManager.Inspect", parsePlan.CacheName)
	}
	return parseM.inspect(parseCtx, parsePlan)
}
