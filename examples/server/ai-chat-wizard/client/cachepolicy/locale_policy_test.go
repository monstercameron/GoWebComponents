package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/client/cachecore"
)

// TestBuildLocaleCatalogResourceKey verifies locale catalog keys stay deterministic and include locale/namespace/bundle dimensions.
func TestBuildLocaleCatalogResourceKey(parseT *testing.T) {
	parseKey := BuildLocaleCatalogResourceKey("EN", "Marketing.Home", "2026-03-28")
	parseExpected := "catalog.locale|locale=en|namespace=marketing.home|bundle=2026-03-28"
	if parseKey != parseExpected {
		parseT.Fatalf("unexpected locale catalog key: %q != %q", parseKey, parseExpected)
	}
}

// TestReadLocaleCatalogResourceSnapshotFirst verifies locale catalog reads can return cached snapshot state on first paint and trigger async refresh.
func TestReadLocaleCatalogResourceSnapshotFirst(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseInput := LocaleCatalogReadInput{
		ScopeKey:      cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en", RouteKey: "/home"}),
		Locale:        "en",
		Namespace:     "marketing.home",
		BundleVersion: "2026-03-28",
	}
	parsePolicy := BuildLocaleCatalogPolicy()
	parseResourceKey := BuildLocaleCatalogResourceKey(parseInput.Locale, parseInput.Namespace, parseInput.BundleVersion)
	parseRecord := cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, parseResourceKey, parseInput.BundleVersion, "hash-v1", time.Now().UTC().Add(-10*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"hero.title":"Cached title"}`), cachecore.CacheStatusReady, "")
	parseSnapshot.SetSync(parseRecord)
	if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		parseT.Fatalf("Set(seed): %v", parseErr)
	}

	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := ReadLocaleCatalogResource(parseCtx, parseAPI, parseInput, func(parseCtx context.Context, parseInput LocaleCatalogReadInput) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, BuildLocaleCatalogResourceKey(parseInput.Locale, parseInput.Namespace, parseInput.BundleVersion), parseInput.BundleVersion, "hash-v2", time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"hero.title":"Fresh title"}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadLocaleCatalogResource: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected snapshot-first locale read metadata: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for locale background refresh")
	}
}

// TestApplyLocaleCatalogInvalidationRules verifies locale and server-bundle invalidation behavior.
func TestApplyLocaleCatalogInvalidationRules(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parsePolicy := BuildLocaleCatalogPolicy()
	parseSetRecord := func(parseLocale string, parseNamespace string, parseBundle string) {
		parseResourceKey := BuildLocaleCatalogResourceKey(parseLocale, parseNamespace, parseBundle)
		parseRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, parseBundle, "hash-"+parseLocale+"-"+parseBundle, time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"k":"v"}`), cachecore.CacheStatusReady, "")
		if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
			parseT.Fatalf("Set(%s): %v", parseResourceKey, parseErr)
		}
		parseSnapshot.SetSync(parseRecord)
	}
	parseSetRecord("en", "marketing.home", "v1")
	parseSetRecord("es", "marketing.home", "v1")
	parseSetRecord("en", "auth.login", "v2")

	parseLocaleEvicted, parseErr := ApplyLocaleCatalogInvalidationForLocaleChange(parseCtx, parseStorage, parseSnapshot, "en")
	if parseErr != nil {
		parseT.Fatalf("ApplyLocaleCatalogInvalidationForLocaleChange: %v", parseErr)
	}
	if parseLocaleEvicted != 1 {
		parseT.Fatalf("expected 1 locale-evicted record, got %d", parseLocaleEvicted)
	}
	parseBundleEvicted, parseErr := ApplyLocaleCatalogInvalidationForServerVersionChange(parseCtx, parseStorage, parseSnapshot, "v2")
	if parseErr != nil {
		parseT.Fatalf("ApplyLocaleCatalogInvalidationForServerVersionChange: %v", parseErr)
	}
	if parseBundleEvicted != 1 {
		parseT.Fatalf("expected 1 bundle-evicted record, got %d", parseBundleEvicted)
	}

	parseRemaining, parseErr := parseStorage.List(parseCtx, parseScopeKey)
	if parseErr != nil {
		parseT.Fatalf("List(remaining): %v", parseErr)
	}
	if len(parseRemaining) != 1 {
		parseT.Fatalf("expected 1 remaining locale catalog record, got %d", len(parseRemaining))
	}
	parseParts, isParseLocaleCatalog := parseResolveLocaleCatalogResourceParts(parseRemaining[0].ResourceKey)
	if !isParseLocaleCatalog || parseParts.Locale != "en" || parseParts.BundleVersion != "v2" {
		parseT.Fatalf("unexpected remaining record %q -> %+v", parseRemaining[0].ResourceKey, parseParts)
	}
}
