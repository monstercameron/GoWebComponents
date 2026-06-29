package cachepolicy

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/cachecore"
)

const modelCatalogResourcePrefix = "model_catalog.metadata"

// ModelCatalogSnapshotInput stores one model-catalog/provider-metadata cache envelope.
type ModelCatalogSnapshotInput struct {
	ScopeKey      string
	MetadataClass string
	ServerVersion string
	CatalogHash   string
}

// BuildModelCatalogSnapshotPolicy returns one model-catalog/provider-metadata cache policy.
func BuildModelCatalogSnapshotPolicy() cachecore.CachePolicy {
	return cachecore.BuildCachePolicy(cachecore.PolicyClassStatic, 1*time.Minute, 2*time.Hour, true, true, false)
}

// BuildModelCatalogResourceKey builds one model-catalog/provider-metadata resource key keyed by server version/hash.
func BuildModelCatalogResourceKey(parseMetadataClass string, parseServerVersion string, parseCatalogHash string) string {
	parseMetadataClass = normalizeModelCatalogMetadataClass(parseMetadataClass)
	parseServerVersion = strings.TrimSpace(parseServerVersion)
	parseCatalogHash = strings.TrimSpace(parseCatalogHash)
	return fmt.Sprintf("%s|class=%s|version=%s|hash=%s", modelCatalogResourcePrefix, parseMetadataClass, parseServerVersion, parseCatalogHash)
}

// StoreModelCatalogSnapshot stores one model/provider metadata snapshot payload.
func StoreModelCatalogSnapshot(parseCtx context.Context, parseStorage cachecore.Storage, parseInput ModelCatalogSnapshotInput, parsePayload []byte) error {
	if parseStorage == nil {
		return nil
	}
	parseInput = normalizeModelCatalogSnapshotInput(parseInput)
	parsePolicy := BuildModelCatalogSnapshotPolicy()
	parseRecord := cachecore.BuildCacheRecordEnvelope(
		parseInput.ScopeKey,
		BuildModelCatalogResourceKey(parseInput.MetadataClass, parseInput.ServerVersion, parseInput.CatalogHash),
		parseInput.ServerVersion,
		parseInput.CatalogHash,
		time.Now().UTC(),
		parsePolicy.StaleAfter,
		parsePolicy.ExpiresAfter,
		append([]byte(nil), parsePayload...),
		cachecore.CacheStatusReady,
		"",
	)
	return parseStorage.Set(parseCtx, parseRecord)
}

// ReadModelCatalogSnapshot reads one model/provider metadata snapshot with snapshot-first SWR behavior.
func ReadModelCatalogSnapshot(
	parseCtx context.Context,
	parseAPI *cachecore.UIAPI,
	parseInput ModelCatalogSnapshotInput,
	parseRefresh func(context.Context, ModelCatalogSnapshotInput) (cachecore.CacheRecordEnvelope, error),
) (cachecore.CachedResourceView, error) {
	if parseAPI == nil {
		return cachecore.CachedResourceView{}, nil
	}
	parseInput = normalizeModelCatalogSnapshotInput(parseInput)
	parseResourceKey := BuildModelCatalogResourceKey(parseInput.MetadataClass, parseInput.ServerVersion, parseInput.CatalogHash)
	return parseAPI.ReadCachedResource(parseCtx, parseInput.ScopeKey, parseResourceKey, BuildModelCatalogSnapshotPolicy(), func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (cachecore.CacheRecordEnvelope, error) {
		_ = parseScopeKey
		_ = parseResourceKey
		if parseRefresh == nil {
			return cachecore.CacheRecordEnvelope{}, nil
		}
		return parseRefresh(parseCtx, parseInput)
	})
}

// ApplyModelCatalogInvalidationForVersionHashChange evicts model/provider metadata records that mismatch one active version/hash.
func ApplyModelCatalogInvalidationForVersionHashChange(parseCtx context.Context, parseStorage cachecore.Storage, parseSnapshot *cachecore.SnapshotStore, parseActiveServerVersion string, parseActiveCatalogHash string) (int, error) {
	parseActiveServerVersion = strings.TrimSpace(parseActiveServerVersion)
	parseActiveCatalogHash = strings.TrimSpace(parseActiveCatalogHash)
	return applyModelCatalogInvalidation(parseCtx, parseStorage, parseSnapshot, func(parseVersion string, parseHash string) bool {
		isParseVersionMismatch := parseActiveServerVersion != "" && parseVersion != parseActiveServerVersion
		isParseHashMismatch := parseActiveCatalogHash != "" && parseHash != parseActiveCatalogHash
		return isParseVersionMismatch || isParseHashMismatch
	})
}

// normalizeModelCatalogSnapshotInput normalizes one model-catalog/provider-metadata input envelope.
func normalizeModelCatalogSnapshotInput(parseInput ModelCatalogSnapshotInput) ModelCatalogSnapshotInput {
	parseInput.ScopeKey = strings.TrimSpace(parseInput.ScopeKey)
	parseInput.MetadataClass = normalizeModelCatalogMetadataClass(parseInput.MetadataClass)
	parseInput.ServerVersion = strings.TrimSpace(parseInput.ServerVersion)
	parseInput.CatalogHash = strings.TrimSpace(parseInput.CatalogHash)
	return parseInput
}

// normalizeModelCatalogMetadataClass normalizes one model-catalog metadata class token.
func normalizeModelCatalogMetadataClass(parseMetadataClass string) string {
	parseMetadataClass = strings.TrimSpace(strings.ToLower(parseMetadataClass))
	switch parseMetadataClass {
	case "providers", "models", "pricing", "capabilities":
		return parseMetadataClass
	default:
		return "providers"
	}
}

// applyModelCatalogInvalidation evicts model-catalog resources that match one mismatch predicate.
func applyModelCatalogInvalidation(
	parseCtx context.Context,
	parseStorage cachecore.Storage,
	parseSnapshot *cachecore.SnapshotStore,
	parseShouldEvict func(string, string) bool,
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
		parseVersion, parseHash, isParseModelCatalog := parseResolveModelCatalogVersionHash(parseRecord.ResourceKey)
		if !isParseModelCatalog {
			continue
		}
		if !parseShouldEvict(parseVersion, parseHash) {
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

// parseResolveModelCatalogVersionHash extracts version/hash metadata from one model-catalog resource key.
func parseResolveModelCatalogVersionHash(parseResourceKey string) (string, string, bool) {
	parseResourceKey = strings.TrimSpace(parseResourceKey)
	if !strings.HasPrefix(parseResourceKey, modelCatalogResourcePrefix+"|") {
		return "", "", false
	}
	parseParts := strings.Split(parseResourceKey, "|")
	parseVersion := ""
	parseHash := ""
	for _, parsePart := range parseParts {
		parsePair := strings.SplitN(strings.TrimSpace(parsePart), "=", 2)
		if len(parsePair) != 2 {
			continue
		}
		switch strings.TrimSpace(strings.ToLower(parsePair[0])) {
		case "version":
			parseVersion = strings.TrimSpace(parsePair[1])
		case "hash":
			parseHash = strings.TrimSpace(parsePair[1])
		}
	}
	if parseVersion == "" && parseHash == "" {
		return "", "", false
	}
	return parseVersion, parseHash, true
}
