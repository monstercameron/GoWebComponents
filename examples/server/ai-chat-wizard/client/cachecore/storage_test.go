package cachecore

import (
	"context"
	"testing"
	"time"
)

// TestMemoryStorageCRUDAndDeleteByPrefix verifies cache record get/list/set/delete/delete-by-prefix behavior.
func TestMemoryStorageCRUDAndDeleteByPrefix(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseScopeA := BuildScopeKey(ScopeKey{
		AppVersion: "v2026.03.28",
		SessionID:  "sid-1",
		UserID:     "u-1",
		Workspace:  "w-1",
		Locale:     "en-US",
		RouteKey:   "/app",
		ThreadKey:  "thread-a",
	})
	parseScopeB := BuildScopeKey(ScopeKey{
		AppVersion: "v2026.03.28",
		SessionID:  "sid-1",
		UserID:     "u-1",
		Workspace:  "w-2",
		Locale:     "en-US",
		RouteKey:   "/app",
		ThreadKey:  "thread-b",
	})
	parseRecordA := BuildCacheRecordEnvelope(parseScopeA, "resource.a", "v1", "etag-a", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"a":1}`), CacheStatusReady, "")
	parseRecordB := BuildCacheRecordEnvelope(parseScopeB, "resource.b", "v1", "etag-b", time.Now().UTC(), 30*time.Second, 120*time.Second, []byte(`{"b":1}`), CacheStatusReady, "")
	if parseErr := parseStorage.Set(parseCtx, parseRecordA); parseErr != nil {
		parseT.Fatalf("Set(recordA): %v", parseErr)
	}
	if parseErr := parseStorage.Set(parseCtx, parseRecordB); parseErr != nil {
		parseT.Fatalf("Set(recordB): %v", parseErr)
	}
	parseScopedResourceKeyA := BuildScopedResourceKey(parseScopeA, "resource.a")
	parseLoadedA, isParseLoadedA, parseErr := parseStorage.Get(parseCtx, parseScopedResourceKeyA)
	if parseErr != nil || !isParseLoadedA {
		parseT.Fatalf("Get(recordA): loaded=%v err=%v", isParseLoadedA, parseErr)
	}
	if parseLoadedA.ResourceKey != "resource.a" {
		parseT.Fatalf("unexpected loaded resource key: %+v", parseLoadedA)
	}
	parseListA, parseErr := parseStorage.List(parseCtx, parseScopeA)
	if parseErr != nil || len(parseListA) != 1 {
		parseT.Fatalf("List(scopeA): len=%d err=%v", len(parseListA), parseErr)
	}
	parseDeletedCount, parseErr := parseStorage.DeleteByPrefix(parseCtx, parseScopeA)
	if parseErr != nil || parseDeletedCount != 1 {
		parseT.Fatalf("DeleteByPrefix(scopeA): count=%d err=%v", parseDeletedCount, parseErr)
	}
	if _, isParseLoadedA, _ = parseStorage.Get(parseCtx, parseScopedResourceKeyA); isParseLoadedA {
		parseT.Fatal("expected scopeA record to be deleted")
	}
	if parseErr = parseStorage.Delete(parseCtx, BuildScopedResourceKey(parseScopeB, "resource.b")); parseErr != nil {
		parseT.Fatalf("Delete(recordB): %v", parseErr)
	}
	if parseAllRecords, parseErr2 := parseStorage.List(parseCtx, ""); parseErr2 != nil || len(parseAllRecords) != 0 {
		parseT.Fatalf("expected empty storage after deletes, len=%d err=%v", len(parseAllRecords), parseErr2)
	}
}

// TestMemoryStorageUpdateQueueAtomic verifies atomic queue mutation behavior.
func TestMemoryStorageUpdateQueueAtomic(parseT *testing.T) {
	parseCtx := context.Background()
	parseStorage := BuildMemoryStorage()
	parseQueueKey := "queue.logs"
	parseQueuedRecord := BuildOutboxRecordEnvelope(
		parseQueueKey,
		"resource.logs",
		"op-1",
		time.Now().UTC(),
		time.Now().UTC().Add(30*time.Second),
		[]byte(`{"log":"hello"}`),
		OutboxStatusQueued,
		"",
		"",
	)
	parseUpdatedQueue, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseCurrent []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		return append(parseCurrent, parseQueuedRecord), nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(append): %v", parseErr)
	}
	if len(parseUpdatedQueue) != 1 || parseUpdatedQueue[0].OperationKey != "op-1" {
		parseT.Fatalf("unexpected queue after append: %+v", parseUpdatedQueue)
	}
	parseAckedQueue, parseErr := parseStorage.UpdateQueueAtomic(parseCtx, parseQueueKey, func(parseCurrent []OutboxRecordEnvelope) ([]OutboxRecordEnvelope, error) {
		if len(parseCurrent) == 0 {
			return parseCurrent, nil
		}
		parseCurrent[0].Status = OutboxStatusAcked
		parseCurrent[0].AckKey = "ack-1"
		return parseCurrent, nil
	})
	if parseErr != nil {
		parseT.Fatalf("UpdateQueueAtomic(ack): %v", parseErr)
	}
	if len(parseAckedQueue) != 1 || parseAckedQueue[0].Status != OutboxStatusAcked || parseAckedQueue[0].AckKey != "ack-1" {
		parseT.Fatalf("unexpected queue after ack mutation: %+v", parseAckedQueue)
	}
}
