package cachecore

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestReadCachedResource verifies UI API cached-read metadata and refresh triggering.
func TestReadCachedResource(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseCoordinator := BuildSWRCoordinator()
	parseInvalidation := BuildInvalidationEngine(parseStorage, BuildInvalidator())
	parseAPI := BuildUIAPI(parseStorage, parseCoordinator, parseInvalidation)
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseResourceKey := "resource.profile"
	parseRecord := BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v1", "etag-1", time.Now().UTC().Add(-2*time.Minute), 30*time.Second, 5*time.Minute, []byte(`{"name":"sam"}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseRecord); parseErr != nil {
		parseT.Fatalf("Set(seed): %v", parseErr)
	}
	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := parseAPI.ReadCachedResource(parseCtx, parseScopeKey, parseResourceKey, ResolveCachePolicyDefaults(PolicyClassSession), func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "etag-2", time.Now().UTC(), 30*time.Second, 5*time.Minute, []byte(`{"name":"sam2"}`), CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadCachedResource: %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected view metadata: %+v", parseView)
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for UI API background refresh")
	}
}

// TestReadQueuedMutationState verifies queued-mutation state helper behavior.
func TestReadQueuedMutationState(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseAPI := BuildUIAPI(parseStorage, BuildSWRCoordinator(), BuildInvalidationEngine(parseStorage, BuildInvalidator()))
	parseQueueKey := "queue.logs"
	_, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseQueue []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return []OutboxRecordEnvelope{
			BuildOutboxRecordEnvelope(parseQueueKey, "resource.logs", "op-1", time.Now().UTC(), time.Now().UTC().Add(30*time.Second), []byte(`{"log":"x"}`), OutboxStatusQueued, "", ""),
		}, nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(seed): %v", parseErr)
	}
	parseView, parseErr := parseAPI.ReadQueuedMutationState(parseCtx, parseQueueKey)
	if parseErr != nil {
		parseT.Fatalf("ReadQueuedMutationState: %v", parseErr)
	}
	if !parseView.IsPending || parseView.PendingCount != 1 {
		parseT.Fatalf("unexpected queued mutation view: %+v", parseView)
	}
}

// TestApplyOptimisticPatch verifies optimistic patch helper updates storage records.
func TestApplyOptimisticPatch(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseResourceKey := "resource.settings"
	parseSeedRecord := BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v1", "etag-1", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"theme":"light"}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseSeedRecord); parseErr != nil {
		parseT.Fatalf("Set(seed): %v", parseErr)
	}
	parseAPI := BuildUIAPI(parseStorage, BuildSWRCoordinator(), BuildInvalidationEngine(parseStorage, BuildInvalidator()))
	if parseErr := parseAPI.ApplyOptimisticPatch(parseCtx, parseScopeKey, parseResourceKey, func(parseRecord CacheRecordEnvelope) CacheRecordEnvelope {
		parseRecord.Version = "v1-optimistic"
		parseRecord.Payload = []byte(`{"theme":"dark"}`)
		return parseRecord
	}); parseErr != nil {
		parseT.Fatalf("ApplyOptimisticPatch: %v", parseErr)
	}
	parseUpdatedRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, BuildScopedResourceKey(parseScopeKey, parseResourceKey))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(updated): found=%v err=%v", isParseFound, parseErr)
	}
	if parseUpdatedRecord.Version != "v1-optimistic" {
		parseT.Fatalf("expected optimistic version update, got %+v", parseUpdatedRecord)
	}
}

// parseStorageReadGuard wraps one Storage and allows tests to assert render paths avoid synchronous Get calls.
type parseStorageReadGuard struct {
	parseStorage Storage
	parseMu      sync.Mutex
	parseGetHits int
}

// Get records one get-hit and returns one failure when called.
func (parseGuard *parseStorageReadGuard) Get(parseCtx context.Context, parseScopedResourceKey string) (CacheRecordEnvelope, bool, error) {
	_ = parseCtx
	_ = parseScopedResourceKey
	parseGuard.parseMu.Lock()
	parseGuard.parseGetHits++
	parseGuard.parseMu.Unlock()
	return CacheRecordEnvelope{}, false, errors.New("unexpected synchronous storage.Get during render")
}

// List delegates list requests to wrapped storage.
func (parseGuard *parseStorageReadGuard) List(parseCtx context.Context, parseScopePrefix string) ([]CacheRecordEnvelope, error) {
	if parseGuard == nil || parseGuard.parseStorage == nil {
		return []CacheRecordEnvelope{}, nil
	}
	return parseGuard.parseStorage.List(parseCtx, parseScopePrefix)
}

// Set delegates set requests to wrapped storage.
func (parseGuard *parseStorageReadGuard) Set(parseCtx context.Context, parseRecord CacheRecordEnvelope) error {
	if parseGuard == nil || parseGuard.parseStorage == nil {
		return nil
	}
	return parseGuard.parseStorage.Set(parseCtx, parseRecord)
}

// Delete delegates delete requests to wrapped storage.
func (parseGuard *parseStorageReadGuard) Delete(parseCtx context.Context, parseScopedResourceKey string) error {
	if parseGuard == nil || parseGuard.parseStorage == nil {
		return nil
	}
	return parseGuard.parseStorage.Delete(parseCtx, parseScopedResourceKey)
}

// DeleteByPrefix delegates delete-prefix requests to wrapped storage.
func (parseGuard *parseStorageReadGuard) DeleteByPrefix(parseCtx context.Context, parseScopePrefix string) (int, error) {
	if parseGuard == nil || parseGuard.parseStorage == nil {
		return 0, nil
	}
	return parseGuard.parseStorage.DeleteByPrefix(parseCtx, parseScopePrefix)
}

// UpdateQueueAtomic delegates queue updates to wrapped storage.
func (parseGuard *parseStorageReadGuard) UpdateQueueAtomic(parseCtx context.Context, parseQueueKey string, parseUpdate func([]OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error)) ([]OutboxRecordEnvelope, error) {
	if parseGuard == nil || parseGuard.parseStorage == nil {
		if parseUpdate == nil {
			return []OutboxRecordEnvelope{}, nil
		}
		return parseUpdate([]OutboxRecordEnvelope{})
	}
	return parseGuard.parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, parseUpdate)
}

// getReadHits returns one snapshot count of synchronous Get calls.
func (parseGuard *parseStorageReadGuard) getReadHits() int {
	parseGuard.parseMu.Lock()
	defer parseGuard.parseMu.Unlock()
	return parseGuard.parseGetHits
}

// TestReadCachedResourceSnapshotGuardrail verifies render reads stay snapshot-first without sync worker/storage round-trips.
func TestReadCachedResourceSnapshotGuardrail(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseSnapshot := BuildSnapshotStore()
	parseGuard := &parseStorageReadGuard{parseStorage: parseStorage}
	parseAPI := BuildUIAPIWithSnapshot(parseGuard, parseSnapshot, BuildSWRCoordinator(), BuildInvalidationEngine(parseStorage, BuildInvalidator()))

	parseScopeKey := BuildScopeKey(ScopeKey{AppVersion: "v1", SessionID: "sid-1", UserID: "u-1", Workspace: "w-1", Locale: "en-US", RouteKey: "/app", ThreadKey: "thread-1"})
	parseResourceKey := "resource.profile"
	parseRecord := BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v1", "etag-1", time.Now().UTC().Add(-3*time.Minute), 30*time.Second, 5*time.Minute, []byte(`{"name":"sam"}`), CacheStatusReady, "")
	parseSnapshot.SetSync(parseRecord)

	parseRefreshDone := make(chan struct{}, 1)
	parseView, parseErr := parseAPI.ReadCachedResource(parseCtx, parseScopeKey, parseResourceKey, ResolveCachePolicyDefaults(PolicyClassSession), func(parseCtx context.Context, parseScopeKey string, parseResourceKey string) (CacheRecordEnvelope, error) {
		_ = parseCtx
		parseRefreshDone <- struct{}{}
		return BuildCacheRecordEnvelope(parseScopeKey, parseResourceKey, "v2", "etag-2", time.Now().UTC(), 30*time.Second, 5*time.Minute, []byte(`{"name":"sam2"}`), CacheStatusReady, ""), nil
	})
	if parseErr != nil {
		parseT.Fatalf("ReadCachedResource(snapshot): %v", parseErr)
	}
	if !parseView.IsFound || !parseView.IsCached || !parseView.IsStale || !parseView.RefreshTriggered {
		parseT.Fatalf("unexpected snapshot-first view metadata: %+v", parseView)
	}
	if parseGuard.getReadHits() != 0 {
		parseT.Fatalf("expected zero synchronous storage Get calls, got %d", parseGuard.getReadHits())
	}
	select {
	case <-parseRefreshDone:
	case <-time.After(2 * time.Second):
		parseT.Fatal("timed out waiting for snapshot-first background refresh")
	}
	parseRefreshedRecord, isParseFound, parseErr := parseStorage.Get(parseCtx, BuildScopedResourceKey(parseScopeKey, parseResourceKey))
	if parseErr != nil || !isParseFound {
		parseT.Fatalf("Get(refreshed): found=%t err=%v", isParseFound, parseErr)
	}
	if !strings.EqualFold(parseRefreshedRecord.Version, "v2") {
		parseT.Fatalf("expected refreshed storage version v2, got %+v", parseRefreshedRecord)
	}
}
