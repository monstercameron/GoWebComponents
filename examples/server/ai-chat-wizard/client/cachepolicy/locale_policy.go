package cachepolicy

import (
	"context"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/client/cachecore"
)

const localeCatalogResourcePrefix = "catalog.locale"

// LocaleCatalogReadInput stores one locale-catalog cache read envelope.
type LocaleCatalogReadInput struct {
	ScopeKey      string
	Locale        string
	Namespace     string
	BundleVersion string
}

// localeCatalogResourceParts stores parsed locale-catalog resource key metadata.
type localeCatalogResourceParts struct {
	Locale        string
	Namespace     string
	BundleVersion string
}

// BuildLocaleCatalogPolicy returns one policy tuned for locale catalog reuse plus SWR refresh.
func BuildLocaleCatalogPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassStatic, 5*time.Minute, 24*time.Hour, true, true, false)
}

// BuildLocaleCatalogResourceKey builds one locale+namespace+bundle_version cache key.
func BuildLocaleCatalogResourceKey(parseLocale string, parseNamespace string, parseBundleVersion string) string {
	parseLocale = strings.TrimSpace(strings.ToLower(parseLocale))
	parseNamespace = strings.TrimSpace(strings.ToLower(parseNamespace))
	parseBundleVersion = strings.TrimSpace(parseBundleVersion)
	return strings.Join([]string{
		localeCatalogResourcePrefix,
		"locale=" + parseLocale,
		"namespace=" + parseNamespace,
		"bundle=" + parseBundleVersion,
	}, "|")
}

// ReadLocaleCatalogResource reads one locale namespace record from cache with snapshot-first SWR behavior.
func ReadLocaleCatalogResource(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseInput LocaleCatalogReadInput,
	parseRefresh func(context.Context, LocaleCatalogReadInput) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	parseInput = normalizeLocaleCatalogReadInput(parseInput)
	parseResourceKey := BuildLocaleCatalogResourceKey(parseInput.Locale, parseInput.Namespace, parseInput.BundleVersion)
	return parseAPI.ReadCachedResource(parseCtx, parseInput.ScopeKey, parseResourceKey, BuildLocaleCatalogPolicy(), func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseScopeKey
		_ = parseResourceKey
		if parseRefresh == nil {
			return cachecore.CacheRecordEnvelope{}, nil
		}
		return parseRefresh(parseCtx, parseInput)
	})
}

// StoreLocaleCatalogResource writes one locale namespace payload using the locale catalog policy freshness windows.
func StoreLocaleCatalogResource(parseCtx context.Context, parseStorage cachecore.Storage, parseInput LocaleCatalogReadInput, parseBundleHash string, parsePayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parseInput = normalizeLocaleCatalogReadInput(parseInput)
	parsePolicy := BuildLocaleCatalogPolicy()
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		parseInput.ScopeKey,
		BuildLocaleCatalogResourceKey(parseInput.Locale, parseInput.Namespace, parseInput.BundleVersion),
		parseInput.BundleVersion,
		strings.TrimSpace(parseBundleHash),
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		append([]byte(nil), parsePayload...),
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ApplyLocaleCatalogInvalidationForLocaleChange evicts locale-catalog entries that do not match one active locale.
func ApplyLocaleCatalogInvalidationForLocaleChange(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseActiveLocale string) (int, error) {
	parseActiveLocale = strings.TrimSpace(strings.ToLower(parseActiveLocale))
	return applyLocaleCatalogInvalidation(parseCtx, parseStorage, parseSnapshot, func(parseParts localeCatalogResourceParts) bool {
		return parseActiveLocale != "" && parseParts.Locale != parseActiveLocale
	})
}

// ApplyLocaleCatalogInvalidationForServerVersionChange evicts locale-catalog entries that do not match one active bundle version.
func ApplyLocaleCatalogInvalidationForServerVersionChange(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseActiveBundleVersion string) (int, error) {
	parseActiveBundleVersion = strings.TrimSpace(parseActiveBundleVersion)
	return applyLocaleCatalogInvalidation(parseCtx, parseStorage, parseSnapshot, func(parseParts localeCatalogResourceParts) bool {
		return parseActiveBundleVersion != "" && parseParts.BundleVersion != parseActiveBundleVersion
	})
}

// normalizeLocaleCatalogReadInput normalizes one locale-catalog read envelope.
func normalizeLocaleCatalogReadInput(parseInput LocaleCatalogReadInput) LocaleCatalogReadInput {
	parseInput.ScopeKey = strings.TrimSpace(parseInput.ScopeKey)
	parseInput.Locale = strings.TrimSpace(strings.ToLower(parseInput.Locale))
	parseInput.Namespace = strings.TrimSpace(strings.ToLower(parseInput.Namespace))
	parseInput.BundleVersion = strings.TrimSpace(parseInput.BundleVersion)
	return parseInput
}

// applyLocaleCatalogInvalidation applies one locale-catalog eviction predicate against storage plus snapshot mirrors.
func applyLocaleCatalogInvalidation(
	parseCtx context.Context,
	parseStorage cachecore.Storage,
	parseSnapshot *cachecore.SnapshotStore,
	parseShouldEvict func(localeCatalogResourceParts) bool,
) (int, error) {
	if parseStorage == nil || parseShouldEvict == nil {
		return 0, nil
	}
	parseRecords, parseErr := parseStorage.List(parseCtx, "")
	if parseErr != nil {
		return 0, parseErr
	}
	parseEvictedCount := 0
	for _, parseRecord := range parseRecords {
		parseParts, isParseLocaleCatalog := parseResolveLocaleCatalogResourceParts(parseRecord.ResourceKey)
		if !isParseLocaleCatalog {
			continue
		}
		if !parseShouldEvict(parseParts) {
			continue
		}
		parseScopedResourceKey := cachecore.BuildScopedResourceKey(parseRecord.ScopeKey, parseRecord.ResourceKey)
		if parseErr = parseStorage.Delete(parseCtx, parseScopedResourceKey); parseErr != nil {
			return parseEvictedCount, parseErr
		}
		if parseSnapshot != nil {
			parseSnapshot.DeleteSync(parseRecord.ScopeKey, parseRecord.ResourceKey)
		}
		parseEvictedCount++
	}
	return parseEvictedCount, nil
}

// parseResolveLocaleCatalogResourceParts parses one locale-catalog resource key contract shape.
func parseResolveLocaleCatalogResourceParts(parseResourceKey string) (localeCatalogResourceParts, bool) {
	parseParts := strings.Split(strings.TrimSpace(parseResourceKey), "|")
	if len(parseParts) < 4 || strings.TrimSpace(parseParts[0]) != localeCatalogResourcePrefix {
		return localeCatalogResourceParts{}, false
	}
	parseParsed := localeCatalogResourceParts{}
	for _, parsePart := range parseParts[1:] {
		parsePair := strings.SplitN(strings.TrimSpace(parsePart), "=", 2)
		if len(parsePair) != 2 {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(parsePair[0])) {
		case "locale":
			parseParsed.Locale = strings.TrimSpace(strings.ToLower(parsePair[1]))
		case "namespace":
			parseParsed.Namespace = strings.TrimSpace(strings.ToLower(parsePair[1]))
		case "bundle":
			parseParsed.BundleVersion = strings.TrimSpace(parsePair[1])
		}
	}
	if parseParsed.Locale == "" || parseParsed.Namespace == "" || parseParsed.BundleVersion == "" {
		return localeCatalogResourceParts{}, false
	}
	return parseParsed, true
}
