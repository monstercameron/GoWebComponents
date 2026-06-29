package cachepolicy

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/examples/server/ai-chat-wizard/client/cachecore"
)

// TestBuildModelCatalogResourceKey verifies version/hash keyed model-catalog cache keys.
func TestBuildModelCatalogResourceKey(parseT *testing.T) {
	parseKey := BuildModelCatalogResourceKey("models", "2026-03-28", "hash-v1")
	parseExpected := "model_catalog.metadata|class=models|version=2026-03-28|hash=hash-v1"
	if parseKey != parseExpected {
		parseT.Fatalf("unexpected key: %q != %q", parseKey, parseExpected)
	}
}

// TestStoreReadModelCatalogSnapshot verifies model/provider metadata snapshots restore quickly and refresh in background.
func TestStoreReadModelCatalogSnapshot(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseAPI := cachecore.BuildUIAPIWithSnapshot(parseStorage, parseSnapshot, cachecore.BuildSWRCoordinator(), cachecore.BuildInvalidationEngine(parseStorage, cachecore.BuildInvalidator()))
	parseInput := ModelCatalogSnapshotInput{
		ScopeKey:      cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"}),
		MetadataClass: "providers",
		ServerVersion: "sv1",
		CatalogHash:   "hash-1",
	}
	parsePolicy := BuildModelCatalogSnapshotPolicy()
	parseResourceKey := BuildModelCatalogResourceKey(parseInput.MetadataClass, parseInput.ServerVersion, parseInput.CatalogHash)
	parseStaleRecord := cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, parseResourceKey, parseInput.ServerVersion, parseInput.CatalogHash, time.Now().UTC().Add(-10*time.Minute), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"providers":["openai"]}`), cachecore.CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseStaleRecord); parseErr != nil {
		parseT.Fatalf("Set(seed): %v", parseErr)
	}
	parseSnapshot.SetSync(parseStaleRecord)
	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := ReadModelCatalogSnapshot(parseCtx, parseAPI, parseInput, func(parseCtx context.Context, parseInput ModelCatalogSnapshotInput) (cachecore.CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return cachecore.BuildCacheRecordEnvelope(parseInput.ScopeKey, BuildModelCatalogResourceKey(parseInput.MetadataClass, parseInput.ServerVersion, parseInput.CatalogHash), parseInput.ServerVersion, parseInput.CatalogHash, time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"providers":["openai","anthropic"]}`), cachecore.CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadModelCatalogSnapshot: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected snapshot view: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for model-catalog background refresh")
	}
}

// TestApplyModelCatalogInvalidationForVersionHashChange verifies stale version/hash model-catalog records are evicted.
func TestApplyModelCatalogInvalidationForVersionHashChange(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := cachecore.BuildMemoryStorage()
	parseSnapshot := cachecore.BuildSnapshotStore()
	parseScopeKey := cachecore.BuildScopeKey(cachecore.ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1"})
	parsePolicy := BuildModelCatalogSnapshotPolicy()
	parseSetRecord := func(parseClass string, parseVersion string, parseHash string) {
		parseResourceKey := BuildModelCatalogResourceKey(parseClass, parseVersion, parseHash)
		parseRecord := cachecore.BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, parseVersion, parseHash, time.Now().UTC(), parsePolicy.StaleAfter, parsePolicy.ExpiresAfter, []byte(`{"ok":1}`), cachecore.CacheStatusReady, "")
		if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
			parseT.Fatalf("Set(%s): %v", parseResourceKey, parseErr)
		}
		parseSnapshot.SetSync(parseRecord)
	}
	parseSetRecord("providers", "sv1", "h1")
	parseSetRecord("models", "sv1", "h2")
	parseSetRecord("pricing", "sv2", "h2")

	parseEvictedCount, parseErr := ApplyModelCatalogInvalidationForVersionHashChange(parseCtx, parseStorage, parseSnapshot, "sv2", "h2")
	if parseErr != nil {
		parseT.Fatalf("ApplyModelCatalogInvalidationForVersionHashChange: %v", parseErr)
	}
	if parseEvictedCount != 2 {
		parseT.Fatalf("expected 2 evicted records, got %d", parseEvictedCount)
	}
	parseRemaining, parseErr := parseStorage.List(parseCtx, parseScopeKey)
	if parseErr != nil {
		parseT.Fatalf("List(remaining): %v", parseErr)
	}
	if len(parseRemaining) != 1 {
		parseT.Fatalf("expected one remaining model-catalog record, got %d", len(parseRemaining))
	}
	parseVersion, parseHash, isParseModelCatalog := parseResolveModelCatalogVersionHash(parseRemaining[0].ResourceKey)
	if !isParseModelCatalog || parseVersion != "sv2" || parseHash != "h2" {
		parseT.Fatalf("unexpected remaining model-catalog row: resource=%q version=%q hash=%q", parseRemaining[0].ResourceKey, parseVersion, parseHash)
	}
}
